// Package auth provides authentication testing utilities for the Nephoran Intent Operator.

// This package contains helper functions and utilities for testing authentication scenarios.

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// SessionResult represents a session with an ID field
type SessionResult struct {
	ID        string      `json:"id"`
	User      interface{} `json:"user"`
	CreatedAt time.Time   `json:"created_at"`
	ExpiresAt time.Time   `json:"expires_at"`
}

// UserFactory creates test users.
type UserFactory struct{}

// TokenFactory creates test tokens.
type TokenFactory struct {
	issuer string
}

// OAuthResponseFactory creates test OAuth responses.
type OAuthResponseFactory struct{}

// TokenResponse represents an OAuth token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// NewUserFactory creates a new user factory.
func NewUserFactory() *UserFactory {
	return &UserFactory{}
}

// CreateBasicUser creates a basic test user.
func (uf *UserFactory) CreateBasicUser() *TestUser {
	return &TestUser{
		Username:      "testuser",
		Password:      "password123",
		Email:         "test@example.com",
		Roles:         []string{"user"},
		Claims:        make(map[string]interface{}),
		Enabled:       true,
		Subject:       "test-user-123",
		Name:          "Test User",
		EmailVerified: true,
		Provider:      "test",
	}
}

// NewTokenFactory creates a new token factory.
func NewTokenFactory(issuer string) *TokenFactory {
	return &TokenFactory{issuer: issuer}
}

// CreateBasicToken creates basic token claims.
func (tf *TokenFactory) CreateBasicToken(subject string) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss": tf.issuer,
		"sub": subject,
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
	}
}

// CreateExpiredToken creates expired token claims.
func (tf *TokenFactory) CreateExpiredToken(subject string) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss": tf.issuer,
		"sub": subject,
		"exp": now.Add(-time.Hour).Unix(),     // Expired 1 hour ago
		"iat": now.Add(-2 * time.Hour).Unix(), // Issued 2 hours ago
	}
}

// CreateTokenNotValidYet creates token claims not valid yet (future nbf).
func (tf *TokenFactory) CreateTokenNotValidYet(subject string) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss": tf.issuer,
		"sub": subject,
		"exp": now.Add(2 * time.Hour).Unix(), // Valid for 2 hours from now
		"iat": now.Unix(),
		"nbf": now.Add(time.Hour).Unix(), // Not valid until 1 hour from now
	}
}

// NewOAuthResponseFactory creates a new OAuth response factory.
func NewOAuthResponseFactory() *OAuthResponseFactory {
	return &OAuthResponseFactory{}
}

// CreateTokenResponse creates a test token response.
func (of *OAuthResponseFactory) CreateTokenResponse() *TokenResponse {
	return &TokenResponse{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}
}

// JWTManagerMock is a mock JWT manager for testing.
type JWTManagerMock struct {
	blacklistedTokens map[string]bool
	tokenStore        map[string]interface{}
}

// RBACManagerMock is a mock RBAC manager for testing.
type RBACManagerMock struct {
	roles           map[string][]string
	permissions     map[string][]string
	roleStore       map[string]interface{}
	permissionStore map[string]interface{}
}

// SessionManagerMock is a mock session manager for testing.
type SessionManagerMock struct {
	sessions map[string]*SessionResult
	Config   struct {
		SessionTTL time.Duration
	}
}

// NewJWTManagerMock creates a new JWT manager mock.
func NewJWTManagerMock() *JWTManagerMock {
	return &JWTManagerMock{
		blacklistedTokens: make(map[string]bool),
		tokenStore:        make(map[string]interface{}),
	}
}

// NewRBACManagerMock creates a new RBAC manager mock.
func NewRBACManagerMock() *RBACManagerMock {
	return &RBACManagerMock{
		roles:           make(map[string][]string),
		permissions:     make(map[string][]string),
		roleStore:       make(map[string]interface{}),
		permissionStore: make(map[string]interface{}),
	}
}

// CreateRole creates a new role (mock implementation)
func (rbac *RBACManagerMock) CreateRole(ctx context.Context, role interface{}) (*TestRole, error) {
	if testRole, ok := role.(*TestRole); ok {
		// Store the role and return it
		rbac.roleStore[testRole.ID] = testRole
		return testRole, nil
	}
	return nil, fmt.Errorf("invalid role type")
}

// CreatePermission creates a new permission (mock implementation)
func (rbac *RBACManagerMock) CreatePermission(ctx context.Context, permission interface{}) (*TestPermission, error) {
	if testPerm, ok := permission.(*TestPermission); ok {
		// Store the permission and return it
		rbac.permissionStore[testPerm.ID] = testPerm
		return testPerm, nil
	}
	return nil, fmt.Errorf("invalid permission type")
}

// AssignRoleToUser assigns a role to a user (mock implementation)
func (rbac *RBACManagerMock) AssignRoleToUser(ctx context.Context, userID, roleID string) error {
	// Mock implementation - just return success
	if rbac.roles == nil {
		rbac.roles = make(map[string][]string)
	}
	rbac.roles[userID] = append(rbac.roles[userID], roleID)
	return nil
}

// NewSessionManagerMock creates a new session manager mock.
func NewSessionManagerMock() *SessionManagerMock {
	mock := &SessionManagerMock{
		sessions: make(map[string]*SessionResult),
	}
	mock.Config.SessionTTL = time.Hour
	return mock
}

// CreateCompleteTestSetup creates a complete test setup with all factories.
func CreateCompleteTestSetup() (*UserFactory, *TokenFactory, *OAuthResponseFactory, interface{}, interface{}, interface{}, interface{}) {
	uf := NewUserFactory()
	tf := NewTokenFactory("test-issuer")
	of := NewOAuthResponseFactory()
	cf := struct{}{} // placeholder
	rf := struct{}{} // placeholder
	pf := struct{}{} // placeholder
	sf := struct{}{} // placeholder
	return uf, tf, of, cf, rf, pf, sf
}

// CreateTestData creates test data for authentication scenarios.
func CreateTestData() map[string]interface{} {
	return map[string]interface{}{
		"users": map[string]interface{}{
			"basic":  map[string]string{"id": "user1", "name": "Basic User"},
			"admin":  map[string]string{"id": "user2", "name": "Admin User"},
			"github": map[string]string{"id": "user3", "name": "GitHub User"},
		},
		"tokens":      []string{"token1", "token2"},
		"roles":       []string{"user", "admin"},
		"permissions": []string{"read", "write"},
		"sessions":    map[string]string{"session1": "user1"},
	}
}

// AuthTestSuite provides a comprehensive authentication testing framework.

type AuthTestSuite struct {
	t          *testing.T
	fixtures   *AuthFixtures
	mocks      *AuthMocks
	testServer *httptest.Server
	testClient *http.Client
}

// AuthMocks contains all authentication mocks for testing.

type AuthMocks struct {
	Authenticator  *MockAuthenticator
	OAuth2Provider *MockOAuth2Provider
	SAMLProvider   *MockSAMLProvider
	LDAPProvider   *MockLDAPProvider
	CertProvider   *MockCertificateProvider
	MFAProvider    *MockMultiFactorProvider
	AuthzProvider  *MockAuthorizationProvider
}

// NewAuthTestSuite creates a new authentication test suite.

func NewAuthTestSuite(t *testing.T) *AuthTestSuite {
	fixtures := DefaultAuthFixtures()
	mocks := &AuthMocks{
		Authenticator:  NewMockAuthenticator().WithFixtures(fixtures),
		OAuth2Provider: NewMockOAuth2Provider(),
		SAMLProvider:   NewMockSAMLProvider(),
		LDAPProvider:   NewMockLDAPProvider(),
		CertProvider:   NewMockCertificateProvider().WithCertificateFixtures(fixtures),
		MFAProvider:    NewMockMultiFactorProvider(),
		AuthzProvider:  NewMockAuthorizationProvider().WithAuthorizationFixtures(fixtures),
	}

	suite := &AuthTestSuite{
		t:        t,
		fixtures: fixtures,
		mocks:    mocks,
	}

	// Generate test certificates
	err := fixtures.GenerateTestCertificates()
	require.NoError(t, err, "Failed to generate test certificates")

	return suite
}

// SetupTestServer creates a test HTTP server with authentication middleware.

func (ats *AuthTestSuite) SetupTestServer() {
	mux := http.NewServeMux()

	// Setup authentication routes
	mux.HandleFunc("/auth/login", ats.handleLogin)
	mux.HandleFunc("/auth/logout", ats.handleLogout)
	mux.HandleFunc("/auth/token", ats.handleTokenValidation)
	mux.HandleFunc("/auth/refresh", ats.handleTokenRefresh)

	// Setup OAuth2 routes
	mux.HandleFunc("/oauth2/authorize", ats.handleOAuth2Authorize)
	mux.HandleFunc("/oauth2/token", ats.handleOAuth2Token)
	mux.HandleFunc("/oauth2/userinfo", ats.handleOAuth2UserInfo)

	// Setup SAML routes
	mux.HandleFunc("/saml/login", ats.handleSAMLLogin)
	mux.HandleFunc("/saml/acs", ats.handleSAMLACS)
	mux.HandleFunc("/saml/metadata", ats.handleSAMLMetadata)

	// Setup protected routes
	mux.HandleFunc("/api/networkintents", ats.authMiddleware(ats.handleNetworkIntents))
	mux.HandleFunc("/api/e2nodesets", ats.authMiddleware(ats.handleE2NodeSets))
	mux.HandleFunc("/api/admin", ats.requireRole("cluster-admin", ats.handleAdmin))

	// Create test server
	ats.testServer = httptest.NewServer(mux)

	// Setup test client
	ats.testClient = &http.Client{
		Timeout: 30 * time.Second,
	}
}

// SetupTLSTestServer creates a TLS test server for mTLS testing.

func (ats *AuthTestSuite) SetupTLSTestServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/secure", ats.clientCertMiddleware(ats.handleSecureAPI))

	// Create TLS server with client certificate verification
	ats.testServer = httptest.NewUnstartedServer(mux)
	ats.testServer.TLS = &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		// In a real scenario, you would set up proper certificate verification
	}
	ats.testServer.StartTLS()

	// Setup test client with certificate
	cert, exists := ats.fixtures.GetCertificateByName("admin-client")
	require.True(ats.t, exists, "Admin client certificate should exist")

	clientCert, err := tls.X509KeyPair(cert.Certificate, cert.PrivateKey)
	require.NoError(ats.t, err, "Failed to load client certificate")

	ats.testClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates:       []tls.Certificate{clientCert},
				InsecureSkipVerify: true, // For testing only
			},
		},
	}
}

// Teardown cleans up the test suite.

func (ats *AuthTestSuite) Teardown() {
	if ats.testServer != nil {
		ats.testServer.Close()
	}
}

// HTTP Handlers

func (ats *AuthTestSuite) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	session, err := ats.mocks.Authenticator.Authenticate(r.Context(), loginReq.Username, loginReq.Password)
	if err != nil {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	token, err := ats.mocks.Authenticator.GenerateToken(r.Context(), loginReq.Username)
	if err != nil {
		http.Error(w, "Token generation failed", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"session_id": session.SessionID,
		"token":      token,
		"expires_at": session.ExpiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ats *AuthTestSuite) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	err := ats.mocks.Authenticator.Logout(r.Context(), sessionID)
	if err != nil {
		http.Error(w, "Logout failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (ats *AuthTestSuite) handleTokenValidation(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if token == "" {
		http.Error(w, "Token required", http.StatusBadRequest)
		return
	}

	// Remove "Bearer " prefix
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	jwtToken, err := ats.mocks.Authenticator.ValidateToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(claims)
}

func (ats *AuthTestSuite) handleTokenRefresh(w http.ResponseWriter, r *http.Request) {
	var refreshReq struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&refreshReq); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// For simplicity, we'll generate a new token for the same user
	// In a real implementation, you would validate the refresh token
	response := map[string]interface{}{
		"access_token": "new-access-token-" + fmt.Sprintf("%d", time.Now().Unix()),
		"token_type":   "Bearer",
		"expires_in":   3600,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ats *AuthTestSuite) handleOAuth2Authorize(w http.ResponseWriter, r *http.Request) {
	clientID := r.URL.Query().Get("client_id")
	redirectURI := r.URL.Query().Get("redirect_uri")
	scopes := r.URL.Query()["scope"]

	if clientID == "" || redirectURI == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// For testing, assume user is authenticated and consents
	code, err := ats.mocks.OAuth2Provider.GenerateAuthorizationCode(
		r.Context(), clientID, "test-user", redirectURI, scopes)
	if err != nil {
		http.Error(w, "Failed to generate authorization code", http.StatusInternalServerError)
		return
	}

	// Redirect with authorization code
	redirectURL := fmt.Sprintf("%s?code=%s&state=%s", redirectURI, code, r.URL.Query().Get("state"))
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (ats *AuthTestSuite) handleOAuth2Token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")
	code := r.FormValue("code")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	if grantType != "authorization_code" {
		http.Error(w, "Unsupported grant type", http.StatusBadRequest)
		return
	}

	tokenResponse, err := ats.mocks.OAuth2Provider.ExchangeCodeForTokens(
		r.Context(), code, clientID, clientSecret)
	if err != nil {
		http.Error(w, "Token exchange failed", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResponse)
}

func (ats *AuthTestSuite) handleOAuth2UserInfo(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if token == "" {
		http.Error(w, "Token required", http.StatusUnauthorized)
		return
	}

	// Remove "Bearer " prefix
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	accessToken, err := ats.mocks.OAuth2Provider.ValidateAccessToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	userInfo := map[string]interface{}{
		"sub":   accessToken.UserID,
		"scope": accessToken.Scopes,
		"exp":   accessToken.ExpiresAt.Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInfo)
}

func (ats *AuthTestSuite) handleSAMLLogin(w http.ResponseWriter, r *http.Request) {
	entityID := r.URL.Query().Get("entity_id")
	if entityID == "" {
		entityID = "https://api.nephoran.local/saml/sp"
	}

	samlRequest, err := ats.mocks.SAMLProvider.GenerateSAMLRequest(r.Context(), entityID)
	if err != nil {
		http.Error(w, "Failed to generate SAML request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/xml")
	w.Write([]byte(samlRequest))
}

func (ats *AuthTestSuite) handleSAMLACS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		http.Error(w, "SAML response required", http.StatusBadRequest)
		return
	}

	assertion, err := ats.mocks.SAMLProvider.ValidateSAMLResponse(r.Context(), samlResponse)
	if err != nil {
		http.Error(w, "Invalid SAML response", http.StatusUnauthorized)
		return
	}

	// Create session for authenticated user
	session := &UserSession{
		SessionID: fmt.Sprintf("saml-session-%d", time.Now().UnixNano()),
		Username:  assertion.Subject,
		CreatedAt: time.Now(),
		ExpiresAt: assertion.NotOnOrAfter,
		Active:    true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (ats *AuthTestSuite) handleSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	metadata := `<?xml version="1.0" encoding="UTF-8"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" entityID="https://api.nephoran.local/saml/sp">
  <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://api.nephoran.local/saml/acs" index="0"/>
  </SPSSODescriptor>
</EntityDescriptor>`

	w.Header().Set("Content-Type", "text/xml")
	w.Write([]byte(metadata))
}

// Protected route handlers

func (ats *AuthTestSuite) handleNetworkIntents(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"message": "Network intents data",
		"user":    r.Header.Get("X-User-ID"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ats *AuthTestSuite) handleE2NodeSets(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"message": "E2 node sets data",
		"user":    r.Header.Get("X-User-ID"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ats *AuthTestSuite) handleAdmin(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"message": "Admin data - restricted access",
		"user":    r.Header.Get("X-User-ID"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (ats *AuthTestSuite) handleSecureAPI(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"message": "Secure API accessed with client certificate",
		"cert":    r.Header.Get("X-Client-Cert-Subject"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Middleware functions

func (ats *AuthTestSuite) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Authorization required", http.StatusUnauthorized)
			return
		}

		// Remove "Bearer " prefix
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		jwtToken, err := ats.mocks.Authenticator.ValidateToken(r.Context(), token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := jwtToken.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Add user information to request headers
		if sub, exists := claims["sub"]; exists {
			r.Header.Set("X-User-ID", sub.(string))
		}

		next(w, r)
	}
}

func (ats *AuthTestSuite) requireRole(requiredRole string, next http.HandlerFunc) http.HandlerFunc {
	return ats.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			http.Error(w, "User ID not found", http.StatusUnauthorized)
			return
		}

		roles, err := ats.mocks.AuthzProvider.GetUserRoles(r.Context(), userID)
		if err != nil {
			http.Error(w, "Failed to get user roles", http.StatusInternalServerError)
			return
		}

		hasRole := false
		for _, role := range roles {
			if role == requiredRole {
				hasRole = true
				break
			}
		}

		if !hasRole {
			http.Error(w, "Insufficient privileges", http.StatusForbidden)
			return
		}

		next(w, r)
	})
}

func (ats *AuthTestSuite) clientCertMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			http.Error(w, "Client certificate required", http.StatusUnauthorized)
			return
		}

		cert := r.TLS.PeerCertificates[0]
		r.Header.Set("X-Client-Cert-Subject", cert.Subject.CommonName)

		next(w, r)
	}
}

// Helper methods for testing

// TestBasicAuthentication tests basic username/password authentication.

func (ats *AuthTestSuite) TestBasicAuthentication() {
	require.NotNil(ats.t, ats.fixtures, "Fixtures should be initialized")

	// Test valid credentials
	session, err := ats.mocks.Authenticator.Authenticate(context.Background(), "admin", "admin123")
	require.NoError(ats.t, err, "Authentication should succeed with valid credentials")
	require.NotNil(ats.t, session, "Session should be created")
	require.Equal(ats.t, "admin", session.Username, "Username should match")

	// Test invalid credentials
	_, err = ats.mocks.Authenticator.Authenticate(context.Background(), "admin", "wrongpassword")
	require.Error(ats.t, err, "Authentication should fail with invalid credentials")
}

// TestJWTTokenGeneration tests JWT token generation and validation.

func (ats *AuthTestSuite) TestJWTTokenGeneration() {
	// Generate token
	token, err := ats.mocks.Authenticator.GenerateToken(context.Background(), "admin")
	require.NoError(ats.t, err, "Token generation should succeed")
	require.NotEmpty(ats.t, token, "Token should not be empty")

	// Validate token
	jwtToken, err := ats.mocks.Authenticator.ValidateToken(context.Background(), token)
	require.NoError(ats.t, err, "Token validation should succeed")
	require.True(ats.t, jwtToken.Valid, "Token should be valid")

	// Check claims
	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	require.True(ats.t, ok, "Claims should be accessible")
	require.Equal(ats.t, "admin", claims["sub"], "Subject should match username")
}

// TestOAuth2Flow tests the OAuth2 authorization code flow.

func (ats *AuthTestSuite) TestOAuth2Flow() {
	clientID := "nephoran-ui"
	clientSecret := "ui-secret-2025"
	userID := "admin"
	redirectURI := "http://localhost:3000/auth/callback"
	scopes := []string{"openid", "profile", "network:read"}

	// Generate authorization code
	code, err := ats.mocks.OAuth2Provider.GenerateAuthorizationCode(
		context.Background(), clientID, userID, redirectURI, scopes)
	require.NoError(ats.t, err, "Authorization code generation should succeed")
	require.NotEmpty(ats.t, code, "Authorization code should not be empty")

	// Exchange code for tokens
	tokenResponse, err := ats.mocks.OAuth2Provider.ExchangeCodeForTokens(
		context.Background(), code, clientID, clientSecret)
	require.NoError(ats.t, err, "Token exchange should succeed")
	require.NotNil(ats.t, tokenResponse, "Token response should not be nil")
	require.NotEmpty(ats.t, tokenResponse.AccessToken, "Access token should not be empty")

	// Validate access token
	accessToken, err := ats.mocks.OAuth2Provider.ValidateAccessToken(
		context.Background(), tokenResponse.AccessToken)
	require.NoError(ats.t, err, "Access token validation should succeed")
	require.Equal(ats.t, userID, accessToken.UserID, "User ID should match")
}

// TestPermissionChecking tests RBAC permission checking.

func (ats *AuthTestSuite) TestPermissionChecking() {
	// Test admin permissions
	hasPermission := ats.fixtures.HasPermission("admin", "networkintents:create:default")
	require.True(ats.t, hasPermission, "Admin should have create permission")

	hasPermission = ats.fixtures.HasPermission("admin", "e2nodesets:delete:production")
	require.True(ats.t, hasPermission, "Admin should have delete permission")

	// Test operator permissions
	hasPermission = ats.fixtures.HasPermission("operator", "networkintents:read:default")
	require.True(ats.t, hasPermission, "Operator should have read permission")

	hasPermission = ats.fixtures.HasPermission("operator", "networkintents:delete:production")
	require.False(ats.t, hasPermission, "Operator should not have delete permission")

	// Test viewer permissions
	hasPermission = ats.fixtures.HasPermission("viewer", "networkintents:read:default")
	require.True(ats.t, hasPermission, "Viewer should have read permission")

	hasPermission = ats.fixtures.HasPermission("viewer", "networkintents:update:default")
	require.False(ats.t, hasPermission, "Viewer should not have update permission")
}

// TestCertificateAuthentication tests client certificate authentication.

func (ats *AuthTestSuite) TestCertificateAuthentication() {
	// Get admin client certificate
	cert, exists := ats.fixtures.GetCertificateByName("admin-client")
	require.True(ats.t, exists, "Admin client certificate should exist")

	// Validate certificate
	certInfo, err := ats.mocks.CertProvider.ValidateCertificate(context.Background(), cert.Certificate)
	require.NoError(ats.t, err, "Certificate validation should succeed")
	require.NotNil(ats.t, certInfo, "Certificate info should not be nil")
	require.True(ats.t, certInfo.Valid, "Certificate should be valid")
	require.Equal(ats.t, cert.CommonName, certInfo.CommonName, "Common names should match")
}

// GetFixtures returns the test fixtures.

func (ats *AuthTestSuite) GetFixtures() *AuthFixtures {
	return ats.fixtures
}

// GetMocks returns the authentication mocks.

func (ats *AuthTestSuite) GetMocks() *AuthMocks {
	return ats.mocks
}

// GetTestServer returns the test server.

func (ats *AuthTestSuite) GetTestServer() *httptest.Server {
	return ats.testServer
}

// GetTestClient returns the test HTTP client.

func (ats *AuthTestSuite) GetTestClient() *http.Client {
	return ats.testClient
}

// TestContext provides a comprehensive testing context for authentication scenarios
type TestContext struct {
	t          *testing.T
	fixtures   *AuthFixtures
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	KeyID      string // Key ID for JWT signing
	Logger     TestLogger
	SlogLogger *slog.Logger        // For slog.Logger compatibility
	TokenStore *MockTokenStore     // For JWT manager compatibility
	Blacklist  *MockTokenBlacklist // For JWT manager compatibility
	cleanupFns []func()
}

// TestLogger provides a simple test logger interface
type TestLogger interface {
	Log(args ...interface{})
	Logf(format string, args ...interface{})
}

// testLogger implements TestLogger for testing
type testLogger struct {
	t *testing.T
}

func (tl *testLogger) Log(args ...interface{}) {
	tl.t.Log(args...)
}

func (tl *testLogger) Logf(format string, args ...interface{}) {
	tl.t.Logf(format, args...)
}

// NewTestContext creates a new test context for authentication testing
func NewTestContext(t *testing.T) *TestContext {
	// Generate RSA key pair for JWT testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "Failed to generate RSA private key")

	tc := &TestContext{
		t:          t,
		fixtures:   DefaultAuthFixtures(),
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		KeyID:      "test-key-id",
		Logger:     &testLogger{t: t},
		SlogLogger: NewMockSlogLogger(),
		TokenStore: NewMockTokenStore(),
		Blacklist:  NewMockTokenBlacklist(),
		cleanupFns: make([]func(), 0),
	}

	// Generate test certificates
	err = tc.fixtures.GenerateTestCertificates()
	require.NoError(t, err, "Failed to generate test certificates")

	return tc
}

// Cleanup cleans up resources created during testing
func (tc *TestContext) Cleanup() {
	for i := len(tc.cleanupFns) - 1; i >= 0; i-- {
		tc.cleanupFns[i]()
	}
	tc.cleanupFns = nil
}

// AddCleanup adds a cleanup function to be called when the test context is cleaned up
func (tc *TestContext) AddCleanup(fn func()) {
	tc.cleanupFns = append(tc.cleanupFns, fn)
}

// SetupJWTManager creates a JWT manager for testing
func (tc *TestContext) SetupJWTManager() *JWTManagerMock {
	return NewJWTManagerMock()
}

// SetupSessionManager creates a session manager for testing
func (tc *TestContext) SetupSessionManager() *SessionManagerMock {
	return NewSessionManagerMock()
}

// SetupRBACManager creates an RBAC manager for testing
func (tc *TestContext) SetupRBACManager() *RBACManagerMock {
	return NewRBACManagerMock()
}

// CreateTestToken creates a test JWT token with the given claims
func (tc *TestContext) CreateTestToken(claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(tc.PrivateKey)
	require.NoError(tc.t, err, "Failed to sign test token")
	return tokenString
}

// RoleFactory creates test roles
type RoleFactory struct{}

// NewRoleFactory creates a new role factory
func NewRoleFactory() *RoleFactory {
	return &RoleFactory{}
}

// CreateRole creates a basic test role
func (rf *RoleFactory) CreateRole(name, description string, permissions []string) *TestRole {
	return &TestRole{
		ID:          fmt.Sprintf("role-%s", name),
		Name:        name,
		Description: description,
		Permissions: permissions,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateRoleWithPermissions creates a test role with given permission IDs
func (rf *RoleFactory) CreateRoleWithPermissions(permissionIDs []string) *TestRole {
	return &TestRole{
		ID:          fmt.Sprintf("role-%d", time.Now().UnixNano()),
		Name:        fmt.Sprintf("test-role-%d", time.Now().UnixNano()),
		Description: "Test role created with permissions",
		Permissions: permissionIDs,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// PermissionFactory creates test permissions
type PermissionFactory struct{}

// NewPermissionFactory creates a new permission factory
func NewPermissionFactory() *PermissionFactory {
	return &PermissionFactory{}
}

// CreatePermission creates a basic test permission
func (pf *PermissionFactory) CreatePermission(resource, action, scope string) *TestPermission {
	name := fmt.Sprintf("%s:%s:%s", resource, action, scope)
	return &TestPermission{
		ID:          fmt.Sprintf("perm-%s", name),
		Name:        name,
		Description: fmt.Sprintf("Permission to %s %s in %s", action, resource, scope),
		Resource:    resource,
		Action:      action,
		Scope:       scope,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateResourcePermissions creates multiple permissions for a resource with given actions
func (pf *PermissionFactory) CreateResourcePermissions(resource string, actions []string) []*TestPermission {
	permissions := make([]*TestPermission, 0, len(actions))
	for _, action := range actions {
		permission := pf.CreatePermission(resource, action, "default")
		permissions = append(permissions, permission)
	}
	return permissions
}
