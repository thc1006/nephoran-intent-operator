// Package auth provides testing utilities and helpers for auth package
package authtest

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

// Define auth types locally to avoid import cycles

// TokenInfo represents token information for testing
type TokenInfo struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenType string    `json:"token_type"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
	Revoked   bool      `json:"revoked"`
}

// AuditEvent represents an audit event for testing
type AuditEvent struct {
	ID        string                 `json:"id"`
	EventType string                 `json:"event_type"`
	UserID    string                 `json:"user_id"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details"`
}

// Mock implementations for breaking import cycles

// JWTManagerMock provides mock JWT functionality
type JWTManagerMock struct {
	privateKey *rsa.PrivateKey
	keyID      string
}

func (j *JWTManagerMock) GenerateToken(claims jwt.MapClaims) (string, error) {
	if j.privateKey == nil {
		return "", fmt.Errorf("private key not set")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = j.keyID
	return token.SignedString(j.privateKey)
}

func (j *JWTManagerMock) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return &j.privateKey.PublicKey, nil
	})
}

func (j *JWTManagerMock) RefreshToken(tokenString string) (string, error) {
	claims := jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	return j.GenerateToken(claims)
}

func (j *JWTManagerMock) RevokeToken(tokenString string) error {
	return nil // Mock implementation
}

func (j *JWTManagerMock) SetSigningKey(privateKey *rsa.PrivateKey, keyID string) error {
	j.privateKey = privateKey
	j.keyID = keyID
	return nil
}

func (j *JWTManagerMock) Close() {
	// Mock implementation
}

// RBACManagerMock provides mock RBAC functionality
type RBACManagerMock struct {
	roles       map[string][]string // userID -> roles
	permissions map[string][]string // role -> permissions
}

func (r *RBACManagerMock) CheckPermission(ctx context.Context, userID, resource, action string) (bool, error) {
	return true, nil // Mock: always allow
}

func (r *RBACManagerMock) AssignRole(ctx context.Context, userID, role string) error {
	if r.roles == nil {
		r.roles = make(map[string][]string)
	}
	r.roles[userID] = append(r.roles[userID], role)
	return nil
}

func (r *RBACManagerMock) RevokeRole(ctx context.Context, userID, role string) error {
	if r.roles == nil {
		return nil
	}
	roles := r.roles[userID]
	for i, userRole := range roles {
		if userRole == role {
			r.roles[userID] = append(roles[:i], roles[i+1:]...)
			break
		}
	}
	return nil
}

func (r *RBACManagerMock) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	if r.roles == nil {
		return []string{}, nil
	}
	return r.roles[userID], nil
}

func (r *RBACManagerMock) GetRolePermissions(ctx context.Context, role string) ([]string, error) {
	if r.permissions == nil {
		return []string{}, nil
	}
	return r.permissions[role], nil
}

// SessionManagerMock provides mock session functionality
type SessionManagerMock struct {
	sessions map[string]*MockSession
}

type MockSession struct {
	ID        string
	UserID    string
	UserInfo  *providers.UserInfo
	CreatedAt time.Time
	ExpiresAt time.Time
	Data      map[string]interface{}
}

func (s *SessionManagerMock) CreateSession(ctx context.Context, userInfo *providers.UserInfo) (*MockSession, error) {
	if s.sessions == nil {
		s.sessions = make(map[string]*MockSession)
	}
	session := &MockSession{
		ID:        fmt.Sprintf("test-session-%d", time.Now().UnixNano()),
		UserID:    userInfo.Subject,
		UserInfo:  userInfo,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Data:      make(map[string]interface{}),
	}
	s.sessions[session.ID] = session
	return session, nil
}

func (s *SessionManagerMock) GetSession(ctx context.Context, sessionID string) (*MockSession, error) {
	if s.sessions == nil {
		return nil, fmt.Errorf("session not found")
	}
	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

func (s *SessionManagerMock) UpdateSession(ctx context.Context, sessionID string, updates map[string]interface{}) error {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	for k, v := range updates {
		session.Data[k] = v
	}
	return nil
}

func (s *SessionManagerMock) DeleteSession(ctx context.Context, sessionID string) error {
	if s.sessions != nil {
		delete(s.sessions, sessionID)
	}
	return nil
}

func (s *SessionManagerMock) ListUserSessions(ctx context.Context, userID string) ([]*MockSession, error) {
	var sessions []*MockSession
	if s.sessions != nil {
		for _, session := range s.sessions {
			if session.UserID == userID {
				sessions = append(sessions, session)
			}
		}
	}
	return sessions, nil
}

func (s *SessionManagerMock) SetSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "test-session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // For testing
	})
}

func (s *SessionManagerMock) GetSessionFromRequest(r *http.Request) (*MockSession, error) {
	cookie, err := r.Cookie("test-session")
	if err != nil {
		return nil, err
	}
	return s.GetSession(r.Context(), cookie.Value)
}

func (s *SessionManagerMock) Close() {
	// Mock implementation
}

// TestContext provides a complete testing environment for auth tests
type TestContext struct {
	T      *testing.T
	Ctx    context.Context
	Logger *slog.Logger

	// Keys for testing
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	KeyID      string

	// Test servers
	OAuthServer *httptest.Server
	LDAPServer  *MockLDAPServer

	// Mock implementations for testing
	JWTManager     JWTManagerMock
	RBACManager    RBACManagerMock
	SessionManager SessionManagerMock

	// Cleanup functions
	cleanupFuncs []func()
	mutex        sync.Mutex
}

// NewTestContext creates a new test context with default configuration
func NewTestContext(t *testing.T) *TestContext {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Generate test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	publicKey := &privateKey.PublicKey
	keyID := "test-key-id"

	tc := &TestContext{
		T:          t,
		Ctx:        ctx,
		Logger:     logger,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		KeyID:      keyID,
	}

	return tc
}

// SetupJWTManager initializes JWT manager mock for testing
func (tc *TestContext) SetupJWTManager() *JWTManagerMock {
	// Set test keys
	err := tc.JWTManager.SetSigningKey(tc.PrivateKey, tc.KeyID)
	require.NoError(tc.T, err)

	tc.AddCleanup(func() {
		tc.JWTManager.Close()
	})

	return &tc.JWTManager
}

// SetupRBACManager initializes RBAC manager mock for testing
func (tc *TestContext) SetupRBACManager() *RBACManagerMock {
	return &tc.RBACManager
}

// SetupSessionManager initializes session manager mock for testing
func (tc *TestContext) SetupSessionManager() *SessionManagerMock {
	tc.AddCleanup(func() {
		tc.SessionManager.Close()
	})

	return &tc.SessionManager
}

// SetupOAuthServer creates a mock OAuth2 server for testing
func (tc *TestContext) SetupOAuthServer() *httptest.Server {
	if tc.OAuthServer != nil {
		return tc.OAuthServer
	}

	mux := http.NewServeMux()

	// Authorization endpoint
	mux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		tc.handleOAuthAuth(w, r)
	})

	// Token endpoint
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		tc.handleOAuthToken(w, r)
	})

	// User info endpoint
	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		tc.handleOAuthUserInfo(w, r)
	})

	// JWKS endpoint
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		tc.handleJWKS(w, r)
	})

	// OIDC discovery endpoint
	mux.HandleFunc("/.well-known/openid_configuration", func(w http.ResponseWriter, r *http.Request) {
		tc.handleOIDCDiscovery(w, r)
	})

	server := httptest.NewServer(mux)
	tc.OAuthServer = server
	tc.AddCleanup(func() {
		if tc.OAuthServer != nil {
			tc.OAuthServer.Close()
		}
	})

	return server
}

// SetupLDAPServer creates a mock LDAP server for testing
func (tc *TestContext) SetupLDAPServer() *MockLDAPServer {
	if tc.LDAPServer != nil {
		return tc.LDAPServer
	}

	ldapServer := NewMockLDAPServer()
	tc.LDAPServer = ldapServer
	tc.AddCleanup(func() {
		if tc.LDAPServer != nil {
			tc.LDAPServer.Close()
		}
	})

	return ldapServer
}

// CreateTestToken creates a test JWT token
func (tc *TestContext) CreateTestToken(claims jwt.MapClaims) string {
	if claims == nil {
		claims = jwt.MapClaims{}
	}

	// Set default claims if not provided
	if claims["iss"] == nil {
		claims["iss"] = "test-issuer"
	}
	if claims["sub"] == nil {
		claims["sub"] = "test-user"
	}
	if claims["exp"] == nil {
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}
	if claims["iat"] == nil {
		claims["iat"] = time.Now().Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = tc.KeyID

	tokenString, err := token.SignedString(tc.PrivateKey)
	require.NoError(tc.T, err)

	return tokenString
}

// CreateTestUser creates a test user info
func (tc *TestContext) CreateTestUser(userID string) *providers.UserInfo {
	return &providers.UserInfo{
		Subject:       userID,
		Email:         fmt.Sprintf("%s@example.com", userID),
		EmailVerified: true,
		Name:          fmt.Sprintf("Test %s", strings.Title(userID)),
		GivenName:     "Test",
		FamilyName:    strings.Title(userID),
		Username:      userID,
		Provider:      "test",
		ProviderID:    fmt.Sprintf("test-%s", userID),
		Groups:        []string{"users", "testers"},
		Roles:         []string{"viewer"},
		Attributes: map[string]interface{}{
			"department": "engineering",
			"team":       "platform",
		},
	}
}

// AddCleanup adds a cleanup function to be called when the test finishes
func (tc *TestContext) AddCleanup(cleanup func()) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	tc.cleanupFuncs = append(tc.cleanupFuncs, cleanup)
}

// Cleanup performs all cleanup operations
func (tc *TestContext) Cleanup() {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	// Run cleanup functions in reverse order
	for i := len(tc.cleanupFuncs) - 1; i >= 0; i-- {
		tc.cleanupFuncs[i]()
	}
	tc.cleanupFuncs = nil
}

// OAuth2 server handlers
func (tc *TestContext) handleOAuthAuth(w http.ResponseWriter, r *http.Request) {
	// Return authorization code
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")
	code := "test-auth-code"

	redirectURL, _ := url.Parse(redirectURI)
	q := redirectURL.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	redirectURL.RawQuery = q.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func (tc *TestContext) handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	if code != "test-auth-code" {
		http.Error(w, "Invalid authorization code", http.StatusBadRequest)
		return
	}

	// Create token response
	tokenResponse := map[string]interface{}{
		"access_token":  "test-access-token",
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": "test-refresh-token",
		"scope":         "openid email profile",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResponse)
}

func (tc *TestContext) handleOAuthUserInfo(w http.ResponseWriter, r *http.Request) {
	// Check authorization header
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token != "test-access-token" {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Return user info
	userInfo := map[string]interface{}{
		"sub":            "test-user-123",
		"email":          "testuser@example.com",
		"email_verified": true,
		"name":           "Test User",
		"given_name":     "Test",
		"family_name":    "User",
		"picture":        "https://example.com/avatar.jpg",
		"locale":         "en",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInfo)
}

func (tc *TestContext) handleJWKS(w http.ResponseWriter, r *http.Request) {
	// Convert public key to JWK format
	_, err := x509.MarshalPKIXPublicKey(tc.PublicKey)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// For simplicity, return a basic JWKS structure
	// In a real implementation, you'd properly format the RSA key using publicKeyBytes
	jwks := map[string]interface{}{
		"keys": []map[string]interface{}{
			{
				"kty": "RSA",
				"kid": tc.KeyID,
				"use": "sig",
				"alg": "RS256",
				// Note: In production, you'd include the proper n and e values from publicKeyBytes
				"n": "test-modulus",
				"e": "AQAB",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jwks)
}

func (tc *TestContext) handleOIDCDiscovery(w http.ResponseWriter, r *http.Request) {
	baseURL := tc.OAuthServer.URL

	config := map[string]interface{}{
		"issuer":                 baseURL,
		"authorization_endpoint": baseURL + "/auth",
		"token_endpoint":         baseURL + "/token",
		"userinfo_endpoint":      baseURL + "/userinfo",
		"jwks_uri":               baseURL + "/.well-known/jwks.json",
		"scopes_supported": []string{
			"openid", "email", "profile",
		},
		"response_types_supported": []string{
			"code",
		},
		"grant_types_supported": []string{
			"authorization_code", "refresh_token",
		},
		"subject_types_supported": []string{
			"public",
		},
		"id_token_signing_alg_values_supported": []string{
			"RS256",
		},
		"code_challenge_methods_supported": []string{
			"S256",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// Assertion helpers
func AssertNoError(t *testing.T, err error) {
	assert.NoError(t, err)
}

func AssertError(t *testing.T, err error) {
	assert.Error(t, err)
}

func AssertEqual(t *testing.T, expected, actual interface{}) {
	assert.Equal(t, expected, actual)
}

func AssertNotEqual(t *testing.T, expected, actual interface{}) {
	assert.NotEqual(t, expected, actual)
}

func AssertContains(t *testing.T, haystack, needle interface{}) {
	assert.Contains(t, haystack, needle)
}

func AssertNotContains(t *testing.T, haystack, needle interface{}) {
	assert.NotContains(t, haystack, needle)
}

func AssertTrue(t *testing.T, value bool) {
	assert.True(t, value)
}

func AssertFalse(t *testing.T, value bool) {
	assert.False(t, value)
}

func AssertNil(t *testing.T, value interface{}) {
	assert.Nil(t, value)
}

func AssertNotNil(t *testing.T, value interface{}) {
	assert.NotNil(t, value)
}

// PEM helpers for key generation in tests
func GenerateTestKeyPair() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

func PrivateKeyToPEM(key *rsa.PrivateKey) string {
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	keyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: keyBytes,
	}
	return string(pem.EncodeToMemory(keyBlock))
}

func PublicKeyToPEM(key *rsa.PublicKey) (string, error) {
	keyBytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return "", err
	}
	keyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: keyBytes,
	}
	return string(pem.EncodeToMemory(keyBlock)), nil
}

// Mock implementations for JWT manager dependencies

// MockTokenStore provides a mock token store implementation
type MockTokenStore struct {
	tokens map[string]interface{}
	mutex  sync.RWMutex
}

func NewMockTokenStore() *MockTokenStore {
	return &MockTokenStore{
		tokens: make(map[string]interface{}),
	}
}

func (m *MockTokenStore) StoreToken(ctx context.Context, tokenID string, token *TokenInfo) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.tokens[tokenID] = token
	return nil
}

func (m *MockTokenStore) GetToken(ctx context.Context, tokenID string) (*TokenInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if value, exists := m.tokens[tokenID]; exists {
		if token, ok := value.(*TokenInfo); ok {
			return token, nil
		}
	}
	return nil, fmt.Errorf("token not found")
}

func (m *MockTokenStore) UpdateToken(ctx context.Context, tokenID string, token *TokenInfo) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.tokens[tokenID] = token
	return nil
}

func (m *MockTokenStore) DeleteToken(ctx context.Context, tokenID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.tokens, tokenID)
	return nil
}

func (m *MockTokenStore) ListUserTokens(ctx context.Context, userID string) ([]*TokenInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var tokens []*TokenInfo
	for _, value := range m.tokens {
		if token, ok := value.(*TokenInfo); ok && token.UserID == userID {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func (m *MockTokenStore) DeleteUserData(ctx context.Context, userID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for tokenID, value := range m.tokens {
		if token, ok := value.(*TokenInfo); ok && token.UserID == userID {
			delete(m.tokens, tokenID)
		}
	}
	return nil
}

func (m *MockTokenStore) ExportUserData(ctx context.Context, userID string) (map[string]interface{}, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	data := make(map[string]interface{})
	for tokenID, value := range m.tokens {
		if token, ok := value.(*TokenInfo); ok && token.UserID == userID {
			data[tokenID] = token
		}
	}
	return data, nil
}

func (m *MockTokenStore) ApplyDataRetention(ctx context.Context, retentionDays int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	for tokenID, value := range m.tokens {
		if token, ok := value.(*TokenInfo); ok && token.IssuedAt.Before(cutoff) {
			delete(m.tokens, tokenID)
		}
	}
	return nil
}

func (m *MockTokenStore) CleanupExpired(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	now := time.Now()
	for tokenID, value := range m.tokens {
		if token, ok := value.(*TokenInfo); ok && now.After(token.ExpiresAt) {
			delete(m.tokens, tokenID)
		}
	}
	return nil
}

func (m *MockTokenStore) Close() error {
	return nil
}

// MockTokenBlacklist provides a mock token blacklist implementation
type MockTokenBlacklist struct {
	blacklist map[string]time.Time
	mutex     sync.RWMutex
}

func NewMockTokenBlacklist() *MockTokenBlacklist {
	return &MockTokenBlacklist{
		blacklist: make(map[string]time.Time),
	}
}

func (m *MockTokenBlacklist) BlacklistToken(ctx context.Context, tokenID string, expiresAt time.Time) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.blacklist[tokenID] = expiresAt
	return nil
}

func (m *MockTokenBlacklist) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	expiry, exists := m.blacklist[tokenID]
	return exists && time.Now().Before(expiry), nil
}

func (m *MockTokenBlacklist) CleanupExpired(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	now := time.Now()
	for tokenID, expiry := range m.blacklist {
		if now.After(expiry) {
			delete(m.blacklist, tokenID)
		}
	}
	return nil
}

func (m *MockTokenBlacklist) BlacklistUserTokens(ctx context.Context, userID string, reason string) error {
	// For mock, we'll just record this operation
	return nil
}

func (m *MockTokenBlacklist) GetBlacklistAuditTrail(ctx context.Context, tokenID string) ([]AuditEvent, error) {
	// Return empty audit trail for mock
	return []AuditEvent{}, nil
}

func (m *MockTokenBlacklist) Close() error {
	return nil
}

// MockLDAPServer provides a mock LDAP server for testing
type MockLDAPServer struct {
	users map[string]*MockLDAPUser
	groups map[string]*MockLDAPGroup
	running bool
	mutex sync.RWMutex
}

type MockLDAPUser struct {
	DN string
	CN string
	SN string
	Mail string
	UID string
	GidNumber int
	Groups []string
}

type MockLDAPGroup struct {
	DN string
	CN string
	GidNumber int
	Members []string
}

func NewMockLDAPServer() *MockLDAPServer {
	return &MockLDAPServer{
		users: make(map[string]*MockLDAPUser),
		groups: make(map[string]*MockLDAPGroup),
		running: true,
	}
}

func (m *MockLDAPServer) AddUser(user *MockLDAPUser) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.users[user.UID] = user
}

func (m *MockLDAPServer) AddGroup(group *MockLDAPGroup) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.groups[group.CN] = group
}

func (m *MockLDAPServer) GetUser(uid string) (*MockLDAPUser, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	user, exists := m.users[uid]
	return user, exists
}

func (m *MockLDAPServer) GetGroup(cn string) (*MockLDAPGroup, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	group, exists := m.groups[cn]
	return group, exists
}

func (m *MockLDAPServer) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.running = false
}
