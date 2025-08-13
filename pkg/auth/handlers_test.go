package auth_test

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
	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth/testutil"
)

func TestAuthHandlers_Login(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	// Setup mock OAuth server
	oauthServer := testutil.NewOAuth2MockServer("test")
	defer oauthServer.Close()

	// Setup dependencies
	jwtManager := tc.SetupJWTManager()
	sessionManager := tc.SetupSessionManager()
	rbacManager := tc.SetupRBACManager()

	// Create handlers
	handlers := NewAuthHandlers(&AuthHandlersConfig{
		JWTManager:     jwtManager,
		SessionManager: sessionManager,
		RBACManager:    rbacManager,
		OAuthProviders: map[string]interface{}{
			"test": &testutil.MockOAuthProvider{Name: "test"},
		},
		BaseURL: "http://localhost:8080",
		Logger:  tc.Logger,
	})

	tests := []struct {
		name          string
		method        string
		path          string
		body          interface{}
		expectStatus  int
		checkResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "Valid login request",
			method: "POST",
			path:   "/auth/login",
			body: map[string]interface{}{
				"provider":     "test",
				"redirect_uri": "http://localhost:8080/callback",
			},
			expectStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response["auth_url"])
				assert.NotEmpty(t, response["state"])
			},
		},
		{
			name:   "Login with PKCE",
			method: "POST",
			path:   "/auth/login",
			body: map[string]interface{}{
				"provider":     "test",
				"redirect_uri": "http://localhost:8080/callback",
				"use_pkce":     true,
			},
			expectStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response["auth_url"])
				assert.NotEmpty(t, response["state"])
				assert.NotEmpty(t, response["code_challenge"])
			},
		},
		{
			name:   "Missing provider",
			method: "POST",
			path:   "/auth/login",
			body: map[string]interface{}{
				"redirect_uri": "http://localhost:8080/callback",
			},
			expectStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], "provider")
			},
		},
		{
			name:   "Invalid provider",
			method: "POST",
			path:   "/auth/login",
			body: map[string]interface{}{
				"provider":     "invalid",
				"redirect_uri": "http://localhost:8080/callback",
			},
			expectStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], "provider")
			},
		},
		{
			name:   "Invalid redirect URI",
			method: "POST",
			path:   "/auth/login",
			body: map[string]interface{}{
				"provider":     "test",
				"redirect_uri": "javascript:alert('xss')",
			},
			expectStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], "redirect")
			},
		},
		{
			name:         "GET method not allowed",
			method:       "GET",
			path:         "/auth/login",
			expectStatus: http.StatusMethodNotAllowed,
		},
		{
			name:         "Invalid JSON body",
			method:       "POST",
			path:         "/auth/login",
			body:         "invalid json",
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					body = []byte(str)
				} else {
					body, err = json.Marshal(tt.body)
					require.NoError(t, err)
				}
			}

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader(body))
			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			handlers.LoginHandler(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestAuthHandlers_Callback(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	// Setup dependencies
	jwtManager := tc.SetupJWTManager()
	sessionManager := tc.SetupSessionManager()
	rbacManager := tc.SetupRBACManager()

	// Setup mock provider
	mockProvider := testutil.NewMockOAuthProvider("test")
	uf := testutil.NewUserFactory()
	tf := testutil.NewTokenFactory("test")
	of := testutil.NewOAuthResponseFactory()

	testUser := uf.CreateBasicUser()
	tokenResponse := of.CreateTokenResponse()

	// Configure mock expectations
	mockProvider.On("ExchangeCodeForToken",
		context.Background(), "valid-code", "http://localhost:8080/callback", (*providers.PKCEChallenge)(nil)).
		Return(tokenResponse, nil)

	mockProvider.On("GetUserInfo", context.Background(), tokenResponse.AccessToken).
		Return(testUser, nil)

	handlers := NewAuthHandlers(&AuthHandlersConfig{
		JWTManager:     jwtManager,
		SessionManager: sessionManager,
		RBACManager:    rbacManager,
		OAuthProviders: map[string]interface{}{
			"test": mockProvider,
		},
		BaseURL: "http://localhost:8080",
		Logger:  tc.Logger,
	})

	tests := []struct {
		name          string
		setupRequest  func() *http.Request
		expectStatus  int
		checkResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Valid callback",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/auth/callback", nil)
				q := req.URL.Query()
				q.Set("code", "valid-code")
				q.Set("state", "test-state-123")
				q.Set("provider", "test")
				req.URL.RawQuery = q.Encode()
				return req
			},
			expectStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response["access_token"])
				assert.NotEmpty(t, response["refresh_token"])
				assert.Equal(t, testUser.Subject, response["user_id"])

				// Check session cookie
				cookies := w.Result().Cookies()
				assert.NotEmpty(t, cookies)
			},
		},
		{
			name: "Missing authorization code",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/auth/callback", nil)
				q := req.URL.Query()
				q.Set("state", "test-state-123")
				q.Set("provider", "test")
				req.URL.RawQuery = q.Encode()
				return req
			},
			expectStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], "code")
			},
		},
		{
			name: "OAuth error in callback",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/auth/callback", nil)
				q := req.URL.Query()
				q.Set("error", "access_denied")
				q.Set("error_description", "User denied access")
				q.Set("state", "test-state-123")
				q.Set("provider", "test")
				req.URL.RawQuery = q.Encode()
				return req
			},
			expectStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], "access_denied")
			},
		},
		{
			name: "Missing provider",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/auth/callback", nil)
				q := req.URL.Query()
				q.Set("code", "valid-code")
				q.Set("state", "test-state-123")
				req.URL.RawQuery = q.Encode()
				return req
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid provider",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/auth/callback", nil)
				q := req.URL.Query()
				q.Set("code", "valid-code")
				q.Set("state", "test-state-123")
				q.Set("provider", "invalid")
				req.URL.RawQuery = q.Encode()
				return req
			},
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			w := httptest.NewRecorder()

			handlers.CallbackHandler(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestAuthHandlers_RefreshToken(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()
	sessionManager := tc.SetupSessionManager()
	rbacManager := tc.SetupRBACManager()

	// Create test user and tokens
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()

	accessToken, refreshToken, err := jwtManager.GenerateTokenPair(user, nil)
	require.NoError(t, err)

	handlers := NewAuthHandlers(&AuthHandlersConfig{
		JWTManager:     jwtManager,
		SessionManager: sessionManager,
		RBACManager:    rbacManager,
		Logger:         tc.Logger,
	})

	tests := []struct {
		name          string
		setupRequest  func() *http.Request
		expectStatus  int
		checkResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Valid refresh token",
			setupRequest: func() *http.Request {
				body := map[string]string{
					"refresh_token": refreshToken,
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewReader(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			expectStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response["access_token"])
				assert.NotEmpty(t, response["refresh_token"])
				assert.NotEqual(t, accessToken, response["access_token"])
				assert.NotEqual(t, refreshToken, response["refresh_token"])
			},
		},
		{
			name: "Invalid refresh token",
			setupRequest: func() *http.Request {
				body := map[string]string{
					"refresh_token": "invalid-token",
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewReader(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			expectStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"], "token")
			},
		},
		{
			name: "Missing refresh token",
			setupRequest: func() *http.Request {
				body := map[string]string{}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewReader(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name: "GET method not allowed",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/auth/refresh", nil)
			},
			expectStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			w := httptest.NewRecorder()

			handlers.RefreshTokenHandler(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestAuthHandlers_Logout(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()
	sessionManager := tc.SetupSessionManager()
	rbacManager := tc.SetupRBACManager()

	// Create test session and token
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()

	session, err := sessionManager.CreateSession(context.Background(), user, nil)
	require.NoError(t, err)

	token, err := jwtManager.GenerateToken(user, nil)
	require.NoError(t, err)

	handlers := NewAuthHandlers(&AuthHandlersConfig{
		JWTManager:     jwtManager,
		SessionManager: sessionManager,
		RBACManager:    rbacManager,
		Logger:         tc.Logger,
	})

	tests := []struct {
		name          string
		setupRequest  func() *http.Request
		expectStatus  int
		checkResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Logout with session",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/auth/logout", nil)
				req.AddCookie(&http.Cookie{
					Name:  "session",
					Value: session.ID,
				})
				return req
			},
			expectStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Check that session cookie is cleared
				cookies := w.Result().Cookies()
				var sessionCookie *http.Cookie
				for _, cookie := range cookies {
					if cookie.Name == "session" {
						sessionCookie = cookie
						break
					}
				}
				assert.NotNil(t, sessionCookie)
				assert.Empty(t, sessionCookie.Value)
				assert.True(t, sessionCookie.Expires.Before(time.Now()))
			},
		},
		{
			name: "Logout with JWT token",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/auth/logout", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				return req
			},
			expectStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "Logged out successfully", response["message"])

				// Verify token is blacklisted
				isBlacklisted := jwtManager.IsTokenBlacklisted(token)
				assert.True(t, isBlacklisted)
			},
		},
		{
			name: "Logout all sessions",
			setupRequest: func() *http.Request {
				body := map[string]bool{
					"all_sessions": true,
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest("POST", "/auth/logout", bytes.NewReader(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer "+token)
				return req
			},
			expectStatus: http.StatusOK,
		},
		{
			name: "Logout without authentication",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("POST", "/auth/logout", nil)
			},
			expectStatus: http.StatusOK, // Should succeed even without auth
		},
		{
			name: "GET method not allowed",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/auth/logout", nil)
			},
			expectStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			w := httptest.NewRecorder()

			handlers.LogoutHandler(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestAuthHandlers_UserInfo(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()
	sessionManager := tc.SetupSessionManager()
	rbacManager := tc.SetupRBACManager()

	// Create test user and token
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()

	token, err := jwtManager.GenerateToken(user, nil)
	require.NoError(t, err)

	handlers := NewAuthHandlers(&AuthHandlersConfig{
		JWTManager:     jwtManager,
		SessionManager: sessionManager,
		RBACManager:    rbacManager,
		Logger:         tc.Logger,
	})

	tests := []struct {
		name          string
		setupRequest  func() *http.Request
		expectStatus  int
		checkResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Valid user info request",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/auth/userinfo", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				return req
			},
			expectStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, user.Subject, response["sub"])
				assert.Equal(t, user.Email, response["email"])
				assert.Equal(t, user.Name, response["name"])
			},
		},
		{
			name: "Missing authorization",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/auth/userinfo", nil)
			},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid token",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/auth/userinfo", nil)
				req.Header.Set("Authorization", "Bearer invalid-token")
				return req
			},
			expectStatus: http.StatusUnauthorized,
		},
		{
			name: "POST method not allowed",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/auth/userinfo", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				return req
			},
			expectStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			w := httptest.NewRecorder()

			handlers.UserInfoHandler(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestAuthHandlers_HealthCheck(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()
	sessionManager := tc.SetupSessionManager()
	rbacManager := tc.SetupRBACManager()

	handlers := NewAuthHandlers(&AuthHandlersConfig{
		JWTManager:     jwtManager,
		SessionManager: sessionManager,
		RBACManager:    rbacManager,
		Logger:         tc.Logger,
	})

	tests := []struct {
		name          string
		method        string
		expectStatus  int
		checkResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:         "Health check GET",
			method:       "GET",
			expectStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, "healthy", response["status"])
				assert.NotNil(t, response["timestamp"])
				assert.Contains(t, response, "components")

				components := response["components"].(map[string]interface{})
				assert.Equal(t, "healthy", components["jwt_manager"])
				assert.Equal(t, "healthy", components["session_manager"])
				assert.Equal(t, "healthy", components["rbac_manager"])
			},
		},
		{
			name:         "Health check HEAD",
			method:       "HEAD",
			expectStatus: http.StatusOK,
		},
		{
			name:         "POST method not allowed",
			method:       "POST",
			expectStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/auth/health", nil)
			w := httptest.NewRecorder()

			handlers.HealthCheckHandler(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestAuthHandlers_JWKS(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()

	handlers := NewAuthHandlers(&AuthHandlersConfig{
		JWTManager: jwtManager,
		Logger:     tc.Logger,
	})

	req := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
	w := httptest.NewRecorder()

	handlers.JWKSHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var jwks map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &jwks)
	assert.NoError(t, err)
	assert.Contains(t, jwks, "keys")

	keys := jwks["keys"].([]interface{})
	assert.NotEmpty(t, keys)

	key := keys[0].(map[string]interface{})
	assert.Equal(t, "RSA", key["kty"])
	assert.Equal(t, tc.KeyID, key["kid"])
	assert.Equal(t, "sig", key["use"])
	assert.Equal(t, "RS256", key["alg"])
}

func TestAuthHandlers_ErrorHandling(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	// Create handlers with nil dependencies to trigger errors
	handlers := NewAuthHandlers(&AuthHandlersConfig{
		JWTManager:     nil,
		SessionManager: nil,
		RBACManager:    nil,
		Logger:         tc.Logger,
	})

	tests := []struct {
		name         string
		handler      http.HandlerFunc
		setupRequest func() *http.Request
		expectStatus int
	}{
		{
			name:    "Login with nil providers",
			handler: handlers.LoginHandler,
			setupRequest: func() *http.Request {
				body := map[string]string{
					"provider":     "test",
					"redirect_uri": "http://localhost:8080/callback",
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:    "Refresh with nil JWT manager",
			handler: handlers.RefreshTokenHandler,
			setupRequest: func() *http.Request {
				body := map[string]string{
					"refresh_token": "test-token",
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewReader(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			expectStatus: http.StatusInternalServerError,
		},
		{
			name:    "JWKS with nil JWT manager",
			handler: handlers.JWKSHandler,
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
			},
			expectStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			w := httptest.NewRecorder()

			tt.handler(w, req)

			assert.Equal(t, tt.expectStatus, w.Code)
		})
	}
}

func TestAuthHandlers_CSRF(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()
	sessionManager := tc.SetupSessionManager()
	rbacManager := tc.SetupRBACManager()

	handlers := NewAuthHandlers(&AuthHandlersConfig{
		JWTManager:     jwtManager,
		SessionManager: sessionManager,
		RBACManager:    rbacManager,
		EnableCSRF:     true,
		Logger:         tc.Logger,
	})

	tests := []struct {
		name          string
		setupRequest  func() *http.Request
		expectStatus  int
		checkResponse func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "Login without CSRF token",
			setupRequest: func() *http.Request {
				body := map[string]string{
					"provider":     "test",
					"redirect_uri": "http://localhost:8080/callback",
				}
				jsonBody, _ := json.Marshal(body)
				req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(jsonBody))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			expectStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, strings.ToLower(response["error"]), "csrf")
			},
		},
		{
			name: "Get CSRF token",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/auth/csrf-token", nil)
			},
			expectStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response["csrf_token"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			w := httptest.NewRecorder()

			if req.URL.Path == "/auth/csrf-token" {
				handlers.CSRFTokenHandler(w, req)
			} else {
				handlers.LoginHandler(w, req)
			}

			assert.Equal(t, tt.expectStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// Benchmark tests
func BenchmarkAuthHandlers_UserInfo(b *testing.B) {
	tc := testutil.NewTestContext(&testing.T{})
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()

	token, err := jwtManager.GenerateToken(user, nil)
	if err != nil {
		b.Fatal(err)
	}

	handlers := NewAuthHandlers(&AuthHandlersConfig{
		JWTManager: jwtManager,
		Logger:     tc.Logger,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/auth/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handlers.UserInfoHandler(w, req)
	}
}

func BenchmarkAuthHandlers_JWKS(b *testing.B) {
	tc := testutil.NewTestContext(&testing.T{})
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()

	handlers := NewAuthHandlers(&AuthHandlersConfig{
		JWTManager: jwtManager,
		Logger:     tc.Logger,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/.well-known/jwks.json", nil)
		w := httptest.NewRecorder()

		handlers.JWKSHandler(w, req)
	}
}

// Integration test with real HTTP server
func TestAuthHandlers_HTTPServerIntegration(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()
	sessionManager := tc.SetupSessionManager()
	rbacManager := tc.SetupRBACManager()

	handlers := NewAuthHandlers(&AuthHandlersConfig{
		JWTManager:     jwtManager,
		SessionManager: sessionManager,
		RBACManager:    rbacManager,
		Logger:         tc.Logger,
	})

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", handlers.LoginHandler)
	mux.HandleFunc("/auth/callback", handlers.CallbackHandler)
	mux.HandleFunc("/auth/refresh", handlers.RefreshTokenHandler)
	mux.HandleFunc("/auth/logout", handlers.LogoutHandler)
	mux.HandleFunc("/auth/userinfo", handlers.UserInfoHandler)
	mux.HandleFunc("/auth/health", handlers.HealthCheckHandler)
	mux.HandleFunc("/.well-known/jwks.json", handlers.JWKSHandler)

	server := httptest.NewServer(mux)
	defer server.Close()

	// Test health check
	resp, err := http.Get(server.URL + "/auth/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var health map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&health)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", health["status"])

	// Test JWKS endpoint
	resp, err = http.Get(server.URL + "/.well-known/jwks.json")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jwks map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jwks)
	assert.NoError(t, err)
	assert.Contains(t, jwks, "keys")
}
