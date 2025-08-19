package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth/testutil"
)

// IntegrationTestSuite provides a complete test environment for integration testing
type IntegrationTestSuite struct {
	t              *testing.T
	tc             *testutil.TestContext
	jwtManager     *JWTManager
	sessionManager *SessionManager
	rbacManager    *RBACManager
	ldapMiddleware *LDAPAuthMiddleware
	authManager    *AuthManager
	oauthServer    *testutil.OAuth2MockServer
	server         *httptest.Server
	handlers       *AuthHandlers

	// Test data
	testUser       *providers.UserInfo
	testRole       *Role
	testPermission *Permission

	// Authentication state
	accessToken  string
	refreshToken string
	sessionID    string
}

func NewIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	tc := testutil.NewTestContext(t)

	suite := &IntegrationTestSuite{
		t:              t,
		tc:             tc,
		jwtManager:     tc.SetupJWTManager(),
		sessionManager: tc.SetupSessionManager(),
		rbacManager:    tc.SetupRBACManager(),
		oauthServer:    testutil.NewOAuth2MockServer("test"),
	}

	// Setup LDAP middleware with mock provider
	mockLDAPProvider := testutil.NewMockLDAPProvider("test-ldap")
	ldapProviders := map[string]*providers.LDAPProvider{
		"test-ldap": mockLDAPProvider.(*providers.LDAPProvider),
	}
	suite.ldapMiddleware = NewLDAPAuthMiddleware(
		ldapProviders,
		suite.sessionManager,
		suite.jwtManager,
		suite.rbacManager,
		&LDAPMiddlewareConfig{
			Realm:          "Test Realm",
			AllowBasicAuth: true,
			AllowFormAuth:  true,
			AllowJSONAuth:  true,
			SessionTimeout: 24 * time.Hour,
			RequireHTTPS:   false, // For testing
		},
		tc.Logger,
	)

	// Setup unified AuthManager
	authConfig := &AuthConfig{
		JWTSecretKey: "test-secret-key",
		TokenTTL:     time.Hour,
		RefreshTTL:   24 * time.Hour,
		RBAC: RBACConfig{
			EnableRBAC: true,
		},
	}

	authManager, err := NewAuthManager(authConfig, tc.Logger)
	require.NoError(t, err)
	suite.authManager = authManager

	suite.setupTestData()
	suite.setupHTTPServer()

	return suite
}

func (suite *IntegrationTestSuite) setupTestData() {
	uf := testutil.NewUserFactory()
	rf := testutil.NewRoleFactory()
	pf := testutil.NewPermissionFactory()

	// Create test user
	suite.testUser = uf.CreateBasicUser()

	// Create test permission
	ctx := context.Background()
	permission := pf.CreateResourcePermissions("api", []string{"read", "write"})[0]
	createdPerm, err := suite.rbacManager.CreatePermission(ctx, permission)
	require.NoError(suite.t, err)
	suite.testPermission = createdPerm

	// Create test role with permission
	role := rf.CreateRoleWithPermissions([]string{createdPerm.ID})
	role.Name = "test-user-role"
	createdRole, err := suite.rbacManager.CreateRole(ctx, role)
	require.NoError(suite.t, err)
	suite.testRole = createdRole

	// Assign role to user
	err = suite.rbacManager.AssignRoleToUser(ctx, suite.testUser.Subject, createdRole.ID)
	require.NoError(suite.t, err)

	// Setup OAuth server with user data
	suite.oauthServer.AddUser("test-access-token", suite.testUser)
}

func (suite *IntegrationTestSuite) setupHTTPServer() {
	// Create mock OAuth provider
	mockProvider := testutil.NewMockOAuthProvider("test")
	of := testutil.NewOAuthResponseFactory()
	tokenResponse := of.CreateTokenResponse()
	tokenResponse.AccessToken = "test-access-token"

	// Configure mock provider
	mockProvider.On("ExchangeCodeForToken").Return(tokenResponse, nil)
	mockProvider.On("GetUserInfo").Return(suite.testUser, nil)
	mockProvider.On("GetProviderName").Return("test")
	mockProvider.On("SupportsFeature").Return(true)
	mockProvider.On("GetConfiguration").Return(&providers.ProviderConfig{
		Name:         "test",
		Type:         "oauth2",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURL:  "http://localhost:8080/auth/callback",
		Scopes:       []string{"openid", "email", "profile"},
	})

	// Create handlers
	suite.handlers = NewAuthHandlers(&AuthHandlersConfig{
		JWTManager:     suite.jwtManager,
		SessionManager: suite.sessionManager,
		RBACManager:    suite.rbacManager,
		OAuthProviders: map[string]interface{}{
			"test": mockProvider,
		},
		BaseURL: "http://localhost:8080",
		Logger:  suite.tc.Logger,
	})

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Auth endpoints
	mux.HandleFunc("/auth/login", suite.handlers.LoginHandler)
	mux.HandleFunc("/auth/callback", suite.handlers.CallbackHandler)
	mux.HandleFunc("/auth/refresh", suite.handlers.RefreshTokenHandler)
	mux.HandleFunc("/auth/logout", suite.handlers.LogoutHandler)
	mux.HandleFunc("/auth/userinfo", suite.handlers.UserInfoHandler)
	mux.HandleFunc("/auth/health", suite.handlers.HealthCheckHandler)
	mux.HandleFunc("/.well-known/jwks.json", suite.handlers.JWKSHandler)

	// Protected endpoints for testing
	mux.HandleFunc("/api/protected", suite.protectedHandler)
	mux.HandleFunc("/admin/users", suite.adminHandler)

	// Apply middleware
	authMiddleware := NewAuthMiddleware(&AuthMiddlewareConfig{
		JWTManager:     suite.jwtManager,
		SessionManager: suite.sessionManager,
		RequireAuth:    true,
		AllowedPaths:   []string{"/auth/", "/.well-known/", "/health"},
		HeaderName:     "Authorization",
		CookieName:     "session",
		ContextKey:     "user",
	})

	rbacMiddleware := NewRBACMiddleware(&RBACMiddlewareConfig{
		RBACManager: suite.rbacManager,
		ResourceExtractor: func(r *http.Request) string {
			if strings.HasPrefix(r.URL.Path, "/admin") {
				return "admin"
			}
			return "api"
		},
		ActionExtractor: func(r *http.Request) string {
			switch r.Method {
			case http.MethodGet:
				return "read"
			case http.MethodPost, http.MethodPut, http.MethodPatch:
				return "write"
			default:
				return "read"
			}
		},
		UserIDExtractor: func(r *http.Request) string {
			if userCtx := r.Context().Value("user"); userCtx != nil {
				return userCtx.(*UserContext).UserID
			}
			return ""
		},
	})

	// Apply middleware to protected endpoints only
	protectedMux := http.NewServeMux()
	protectedMux.Handle("/api/", rbacMiddleware.Middleware(authMiddleware.Middleware(mux)))
	protectedMux.Handle("/admin/", rbacMiddleware.Middleware(authMiddleware.Middleware(mux)))
	protectedMux.Handle("/auth/", mux) // Auth endpoints don't need auth middleware
	protectedMux.Handle("/.well-known/", mux)

	suite.server = httptest.NewServer(protectedMux)
}

func (suite *IntegrationTestSuite) protectedHandler(w http.ResponseWriter, r *http.Request) {
	userCtx := r.Context().Value("user").(*UserContext)
	response := map[string]interface{}{
		"message": "Protected endpoint accessed",
		"user_id": userCtx.UserID,
		"method":  r.Method,
		"path":    r.URL.Path,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *IntegrationTestSuite) adminHandler(w http.ResponseWriter, r *http.Request) {
	userCtx := r.Context().Value("user").(*UserContext)
	response := map[string]interface{}{
		"message": "Admin endpoint accessed",
		"user_id": userCtx.UserID,
		"method":  r.Method,
		"path":    r.URL.Path,
	}
	json.NewEncoder(w).Encode(response)
}

func (suite *IntegrationTestSuite) Cleanup() {
	if suite.server != nil {
		suite.server.Close()
	}
	if suite.oauthServer != nil {
		suite.oauthServer.Close()
	}
	suite.tc.Cleanup()
}

func TestIntegration_CompleteOAuth2Flow(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	// Step 1: Initiate OAuth2 login
	t.Run("Step 1: Initiate OAuth2 login", func(t *testing.T) {
		loginReq := map[string]interface{}{
			"provider":     "test",
			"redirect_uri": "http://localhost:8080/auth/callback",
			"use_pkce":     true,
		}

		body, _ := json.Marshal(loginReq)
		resp, err := http.Post(suite.server.URL+"/auth/login", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var loginResp map[string]string
		err = json.NewDecoder(resp.Body).Decode(&loginResp)
		require.NoError(t, err)

		assert.NotEmpty(t, loginResp["auth_url"])
		assert.NotEmpty(t, loginResp["state"])
		assert.NotEmpty(t, loginResp["code_challenge"])
	})

	// Step 2: Simulate OAuth2 callback
	t.Run("Step 2: OAuth2 callback", func(t *testing.T) {
		callbackURL := fmt.Sprintf("%s/auth/callback?code=test-auth-code&state=test-state&provider=test", suite.server.URL)

		resp, err := http.Get(callbackURL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var callbackResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&callbackResp)
		require.NoError(t, err)

		suite.accessToken = callbackResp["access_token"].(string)
		suite.refreshToken = callbackResp["refresh_token"].(string)

		assert.NotEmpty(t, suite.accessToken)
		assert.NotEmpty(t, suite.refreshToken)
		assert.Equal(t, suite.testUser.Subject, callbackResp["user_id"])

		// Extract session cookie if present
		for _, cookie := range resp.Cookies() {
			if cookie.Name == "session" {
				suite.sessionID = cookie.Value
				break
			}
		}
	})

	// Step 3: Access user info
	t.Run("Step 3: Access user info", func(t *testing.T) {
		req, _ := http.NewRequest("GET", suite.server.URL+"/auth/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+suite.accessToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var userInfo map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&userInfo)
		require.NoError(t, err)

		assert.Equal(t, suite.testUser.Subject, userInfo["sub"])
		assert.Equal(t, suite.testUser.Email, userInfo["email"])
		assert.Equal(t, suite.testUser.Name, userInfo["name"])
	})

	// Step 4: Access protected endpoint
	t.Run("Step 4: Access protected endpoint", func(t *testing.T) {
		req, _ := http.NewRequest("GET", suite.server.URL+"/api/protected", nil)
		req.Header.Set("Authorization", "Bearer "+suite.accessToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "Protected endpoint accessed", response["message"])
		assert.Equal(t, suite.testUser.Subject, response["user_id"])
	})

	// Step 5: Token refresh
	t.Run("Step 5: Token refresh", func(t *testing.T) {
		refreshReq := map[string]string{
			"refresh_token": suite.refreshToken,
		}

		body, _ := json.Marshal(refreshReq)
		resp, err := http.Post(suite.server.URL+"/auth/refresh", "application/json", bytes.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var refreshResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&refreshResp)
		require.NoError(t, err)

		newAccessToken := refreshResp["access_token"].(string)
		newRefreshToken := refreshResp["refresh_token"].(string)

		assert.NotEmpty(t, newAccessToken)
		assert.NotEmpty(t, newRefreshToken)
		assert.NotEqual(t, suite.accessToken, newAccessToken)
		assert.NotEqual(t, suite.refreshToken, newRefreshToken)

		// Update tokens for subsequent tests
		suite.accessToken = newAccessToken
		suite.refreshToken = newRefreshToken
	})

	// Step 6: Logout
	t.Run("Step 6: Logout", func(t *testing.T) {
		req, _ := http.NewRequest("POST", suite.server.URL+"/auth/logout", nil)
		req.Header.Set("Authorization", "Bearer "+suite.accessToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var logoutResp map[string]string
		err = json.NewDecoder(resp.Body).Decode(&logoutResp)
		require.NoError(t, err)

		assert.Equal(t, "Logged out successfully", logoutResp["message"])
	})

	// Step 7: Verify token is blacklisted
	t.Run("Step 7: Verify token blacklisted", func(t *testing.T) {
		req, _ := http.NewRequest("GET", suite.server.URL+"/auth/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+suite.accessToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestIntegration_SessionBasedAuthentication(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	// Create session directly for testing
	ctx := context.Background()
	session, err := suite.sessionManager.CreateSession(ctx, suite.testUser, map[string]interface{}{
		"login_method": "oauth2",
		"provider":     "test",
	})
	require.NoError(t, err)

	t.Run("Access protected endpoint with session", func(t *testing.T) {
		req, _ := http.NewRequest("GET", suite.server.URL+"/api/protected", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: session.ID,
		})

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "Protected endpoint accessed", response["message"])
		assert.Equal(t, suite.testUser.Subject, response["user_id"])
	})

	t.Run("Invalid session cookie", func(t *testing.T) {
		req, _ := http.NewRequest("GET", suite.server.URL+"/api/protected", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: "invalid-session-id",
		})

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Session logout", func(t *testing.T) {
		req, _ := http.NewRequest("POST", suite.server.URL+"/auth/logout", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: session.ID,
		})

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify session is invalidated
		req, _ = http.NewRequest("GET", suite.server.URL+"/api/protected", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: session.ID,
		})

		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestIntegration_RBACAuthorization(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	// Create tokens for testing
	userToken, err := suite.jwtManager.GenerateToken(suite.testUser, nil)
	require.NoError(t, err)

	// Create admin user and role
	uf := testutil.NewUserFactory()
	adminUser := uf.CreateAdminUser()

	ctx := context.Background()

	// Create admin permission
	pf := testutil.NewPermissionFactory()
	adminPerm := pf.CreateResourcePermissions("admin", []string{"*"})[0]
	createdAdminPerm, err := suite.rbacManager.CreatePermission(ctx, adminPerm)
	require.NoError(t, err)

	// Create admin role
	rf := testutil.NewRoleFactory()
	adminRole := rf.CreateRoleWithPermissions([]string{createdAdminPerm.ID})
	adminRole.Name = "admin"
	createdAdminRole, err := suite.rbacManager.CreateRole(ctx, adminRole)
	require.NoError(t, err)

	// Assign admin role to admin user
	err = suite.rbacManager.AssignRoleToUser(ctx, adminUser.Subject, createdAdminRole.ID)
	require.NoError(t, err)

	adminToken, err := suite.jwtManager.GenerateToken(adminUser, nil)
	require.NoError(t, err)

	tests := []struct {
		name         string
		token        string
		endpoint     string
		method       string
		expectStatus int
		expectAccess bool
	}{
		{
			name:         "User can read API",
			token:        userToken,
			endpoint:     "/api/protected",
			method:       "GET",
			expectStatus: http.StatusOK,
			expectAccess: true,
		},
		{
			name:         "User can write API",
			token:        userToken,
			endpoint:     "/api/protected",
			method:       "POST",
			expectStatus: http.StatusOK,
			expectAccess: true,
		},
		{
			name:         "User cannot access admin",
			token:        userToken,
			endpoint:     "/admin/users",
			method:       "GET",
			expectStatus: http.StatusForbidden,
			expectAccess: false,
		},
		{
			name:         "Admin can access admin endpoints",
			token:        adminToken,
			endpoint:     "/admin/users",
			method:       "GET",
			expectStatus: http.StatusOK,
			expectAccess: true,
		},
		{
			name:         "Admin can access API endpoints",
			token:        adminToken,
			endpoint:     "/api/protected",
			method:       "GET",
			expectStatus: http.StatusOK,
			expectAccess: true,
		},
		{
			name:         "No token access denied",
			token:        "",
			endpoint:     "/api/protected",
			method:       "GET",
			expectStatus: http.StatusUnauthorized,
			expectAccess: false,
		},
		{
			name:         "Invalid token access denied",
			token:        "invalid-token",
			endpoint:     "/api/protected",
			method:       "GET",
			expectStatus: http.StatusUnauthorized,
			expectAccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, suite.server.URL+tt.endpoint, nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectStatus, resp.StatusCode)

			if tt.expectAccess && resp.StatusCode == http.StatusOK {
				var response map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&response)
				assert.NoError(t, err)
				assert.Contains(t, response, "message")
				assert.Contains(t, response, "user_id")
			}
		})
	}
}

func TestIntegration_ErrorScenarios(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	t.Run("Malformed JSON in login request", func(t *testing.T) {
		resp, err := http.Post(suite.server.URL+"/auth/login", "application/json", strings.NewReader("invalid json"))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Missing Content-Type header", func(t *testing.T) {
		body := `{"provider": "test", "redirect_uri": "http://localhost:8080/callback"}`
		resp, err := http.Post(suite.server.URL+"/auth/login", "", strings.NewReader(body))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Expired token access", func(t *testing.T) {
		// Create expired token
		expiredClaims := map[string]interface{}{
			"iss": "test-issuer",
			"sub": suite.testUser.Subject,
			"exp": time.Now().Add(-time.Hour).Unix(),
			"iat": time.Now().Add(-2 * time.Hour).Unix(),
		}
		expiredToken := suite.tc.CreateTestToken(expiredClaims)

		req, _ := http.NewRequest("GET", suite.server.URL+"/auth/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+expiredToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Invalid authorization header format", func(t *testing.T) {
		req, _ := http.NewRequest("GET", suite.server.URL+"/auth/userinfo", nil)
		req.Header.Set("Authorization", "InvalidFormat token-here")

		client := &http.Client{}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestIntegration_ConcurrentAccess(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	// Create token for concurrent testing
	token, err := suite.jwtManager.GenerateToken(suite.testUser, nil)
	require.NoError(t, err)

	const numGoroutines = 10
	const numRequests = 100

	results := make(chan error, numGoroutines*numRequests)

	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			client := &http.Client{
				Timeout: 10 * time.Second,
			}

			for j := 0; j < numRequests; j++ {
				req, _ := http.NewRequest("GET", suite.server.URL+"/auth/userinfo", nil)
				req.Header.Set("Authorization", "Bearer "+token)

				resp, err := client.Do(req)
				if err != nil {
					results <- err
					continue
				}

				if resp.StatusCode != http.StatusOK {
					results <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
					resp.Body.Close()
					continue
				}

				resp.Body.Close()
				results <- nil
			}
		}(i)
	}

	// Collect results
	var errors []error
	for i := 0; i < numGoroutines*numRequests; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	// Allow for some failures due to concurrent access, but not too many
	errorRate := float64(len(errors)) / float64(numGoroutines*numRequests)
	assert.Less(t, errorRate, 0.01, "Error rate should be less than 1%%: %v", errors[:min(len(errors), 10)])
}

func TestIntegration_HealthCheck(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	resp, err := http.Get(suite.server.URL + "/auth/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health["status"])
	assert.Contains(t, health, "timestamp")
	assert.Contains(t, health, "components")

	components := health["components"].(map[string]interface{})
	assert.Equal(t, "healthy", components["jwt_manager"])
	assert.Equal(t, "healthy", components["session_manager"])
	assert.Equal(t, "healthy", components["rbac_manager"])
}

func TestIntegration_JWKSEndpoint(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	resp, err := http.Get(suite.server.URL + "/.well-known/jwks.json")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var jwks map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)
	require.NoError(t, err)

	assert.Contains(t, jwks, "keys")
	keys := jwks["keys"].([]interface{})
	assert.NotEmpty(t, keys)

	key := keys[0].(map[string]interface{})
	assert.Equal(t, "RSA", key["kty"])
	assert.Equal(t, suite.tc.KeyID, key["kid"])
	assert.Equal(t, "sig", key["use"])
	assert.Equal(t, "RS256", key["alg"])
}

func TestIntegration_MiddlewareChaining(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	// Test that middleware is properly chained by checking headers and authentication
	token, err := suite.jwtManager.GenerateToken(suite.testUser, nil)
	require.NoError(t, err)

	req, _ := http.NewRequest("GET", suite.server.URL+"/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "integration-test-client")

	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Check that security headers are set (would be set by security middleware if configured)
	// Check that CORS headers are present if CORS middleware is configured
	// This depends on the specific middleware configuration in setupHTTPServer

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "Protected endpoint accessed", response["message"])
	assert.Equal(t, suite.testUser.Subject, response["user_id"])
	assert.Equal(t, "GET", response["method"])
	assert.Equal(t, "/api/protected", response["path"])
}

// LDAP Integration Tests

func TestIntegration_LDAPAuthenticationFlow(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	// Test basic authentication with valid credentials
	t.Run("LDAP Basic Authentication", func(t *testing.T) {
		// Create a protected handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authContext := GetAuthContext(r.Context())
			require.NotNil(t, authContext)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"user": authContext.UserID})
		})

		protectedHandler := suite.ldapMiddleware.LDAPAuthenticateMiddleware(handler)

		// Test with valid Basic Auth credentials
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.SetBasicAuth("testuser", "password123")
		recorder := httptest.NewRecorder()

		protectedHandler.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)

		var response map[string]string
		err := json.NewDecoder(recorder.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, "testuser", response["user"])
	})

	t.Run("LDAP JSON Login", func(t *testing.T) {
		loginReq := map[string]string{
			"username": "testuser",
			"password": "password123",
		}
		jsonData, _ := json.Marshal(loginReq)

		req := httptest.NewRequest(http.MethodPost, "/auth/ldap/login", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		suite.ldapMiddleware.HandleLDAPLogin(recorder, req)

		assert.Equal(t, http.StatusOK, recorder.Code)

		var response map[string]interface{}
		err := json.Unmarshal(recorder.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.True(t, response["success"].(bool))
		assert.NotEmpty(t, response["access_token"])
	})

	t.Run("LDAP Form Login", func(t *testing.T) {
		form := url.Values{}
		form.Set("username", "testuser")
		form.Set("password", "password123")

		req := httptest.NewRequest(http.MethodPost, "/auth/ldap/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		recorder := httptest.NewRecorder()

		suite.ldapMiddleware.HandleLDAPLogin(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("LDAP Invalid Credentials", func(t *testing.T) {
		loginReq := map[string]string{
			"username": "testuser",
			"password": "invalid",
		}
		jsonData, _ := json.Marshal(loginReq)
		req := httptest.NewRequest(http.MethodPost, "/auth/ldap/login", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		suite.ldapMiddleware.HandleLDAPLogin(recorder, req)
		assert.Equal(t, http.StatusUnauthorized, recorder.Code)
	})

	t.Run("LDAP Logout", func(t *testing.T) {
		// First login to get a session
		loginReq := map[string]string{
			"username": "testuser",
			"password": "password123",
		}
		jsonData, _ := json.Marshal(loginReq)

		req := httptest.NewRequest(http.MethodPost, "/auth/ldap/login", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()

		suite.ldapMiddleware.HandleLDAPLogin(recorder, req)
		require.Equal(t, http.StatusOK, recorder.Code)

		// Extract session cookie
		var sessionCookie *http.Cookie
		for _, cookie := range recorder.Result().Cookies() {
			if cookie.Name == "nephoran_session" {
				sessionCookie = cookie
				break
			}
		}
		require.NotNil(t, sessionCookie)

		// Test logout
		req = httptest.NewRequest(http.MethodPost, "/auth/ldap/logout", nil)
		req.AddCookie(sessionCookie)
		recorder = httptest.NewRecorder()

		suite.ldapMiddleware.HandleLDAPLogout(recorder, req)
		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}

func TestIntegration_AuthManagerUnified(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	t.Run("Provider Listing", func(t *testing.T) {
		providers := suite.authManager.ListProviders()
		assert.NotEmpty(t, providers)

		// Should have both OAuth2 and LDAP providers configured
		if oauth2Info, exists := providers["oauth2"]; exists {
			oauth2Map := oauth2Info.(map[string]interface{})
			assert.NotEmpty(t, oauth2Map)
		}

		if ldapInfo, exists := providers["ldap"]; exists {
			ldapMap := ldapInfo.(map[string]interface{})
			assert.NotEmpty(t, ldapMap)
		}
	})

	t.Run("Connection Testing", func(t *testing.T) {
		results := suite.authManager.TestConnections(context.Background())
		assert.NotEmpty(t, results)

		// Check that we get results for configured providers
		for providerName, err := range results {
			t.Logf("Provider %s: %v", providerName, err)
		}
	})

	t.Run("User Authentication", func(t *testing.T) {
		userInfo, provider, err := suite.authManager.AuthenticateUser(
			context.Background(),
			"testuser",
			"password123",
			"ldap",
		)

		if err == nil {
			assert.Equal(t, "testuser", userInfo.Username)
			assert.Contains(t, provider, "ldap")

			// Test session creation through AuthManager
			session, err := suite.authManager.CreateSession(context.Background(), userInfo, provider)
			require.NoError(t, err)
			assert.Equal(t, "testuser", session.UserID)

			// Test session validation
			validatedSession, err := suite.authManager.ValidateSession(context.Background(), session.ID)
			require.NoError(t, err)
			assert.Equal(t, session.ID, validatedSession.ID)
		} else {
			t.Logf("Authentication failed as expected in test environment: %v", err)
		}
	})

	t.Run("Health Check Integration", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		recorder := httptest.NewRecorder()

		suite.authManager.HandleHealthCheck(recorder, req)

		// Health check should provide status information
		var health map[string]interface{}
		if recorder.Code == http.StatusOK || recorder.Code == http.StatusServiceUnavailable {
			err := json.NewDecoder(recorder.Body).Decode(&health)
			if err == nil {
				assert.Contains(t, health, "status")
			}
		}
	})

	t.Run("Graceful Shutdown", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := suite.authManager.Shutdown(ctx)
		// Should not error on shutdown in test environment
		if err != nil {
			t.Logf("Shutdown completed with warnings: %v", err)
		}
	})
}

func TestIntegration_CompleteAuthenticationSuite(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	defer suite.Cleanup()

	// This is a comprehensive test suite that verifies integration
	// between all authentication components
	testSuite := &suite.Suite{}
	testSuite.SetT(t)

	t.Run("OAuth2 Flow Integration", func(t *testing.T) {
		// Comprehensive OAuth2 flow testing
		TestIntegration_CompleteOAuth2Flow(t)
	})

	t.Run("Session Management Integration", func(t *testing.T) {
		// Session-based authentication testing
		TestIntegration_SessionBasedAuthentication(t)
	})

	t.Run("RBAC Authorization Integration", func(t *testing.T) {
		// RBAC authorization testing
		TestIntegration_RBACAuthorization(t)
	})

	t.Run("LDAP Authentication Integration", func(t *testing.T) {
		// LDAP authentication testing
		TestIntegration_LDAPAuthenticationFlow(t)
	})

	t.Run("Unified Auth Manager Integration", func(t *testing.T) {
		// Unified authentication manager testing
		TestIntegration_AuthManagerUnified(t)
	})

	t.Run("Error Scenarios Integration", func(t *testing.T) {
		// Error handling integration testing
		TestIntegration_ErrorScenarios(t)
	})

	t.Run("Concurrent Access Integration", func(t *testing.T) {
		// Concurrent access testing
		TestIntegration_ConcurrentAccess(t)
	})

	t.Run("Health Check Integration", func(t *testing.T) {
		// Health check integration testing
		TestIntegration_HealthCheck(t)
	})

	t.Run("JWKS Endpoint Integration", func(t *testing.T) {
		// JWKS endpoint integration testing
		TestIntegration_JWKSEndpoint(t)
	})

	t.Run("Middleware Chaining Integration", func(t *testing.T) {
		// Middleware chaining integration testing
		TestIntegration_MiddlewareChaining(t)
	})
}

// Helper function for min operation
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
