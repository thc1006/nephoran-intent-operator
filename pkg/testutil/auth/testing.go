// Package auth provides testing utilities and helpers for auth package
package auth

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
	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

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

	// Managers under test
	JWTManager     *auth.JWTManager
	RBACManager    *auth.RBACManager
	SessionManager *auth.SessionManager

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

// SetupJWTManager initializes JWT manager for testing
func (tc *TestContext) SetupJWTManager() *auth.JWTManager {
	if tc.JWTManager != nil {
		return tc.JWTManager
	}

	config := &auth.JWTConfig{
		Issuer:               "test-issuer",
		DefaultTTL:           time.Hour,
		RefreshTTL:           24 * time.Hour,
		KeyRotationPeriod:    7 * 24 * time.Hour,
		RequireSecureCookies: false, // Disable for testing
		CookieDomain:         "localhost",
		CookiePath:           "/",
	}

	// Create mock dependencies for JWT manager
	tokenStore := NewMockTokenStore()
	blacklist := NewMockTokenBlacklist()
	
	jwtManager, err := auth.NewJWTManager(config, tokenStore, blacklist, tc.Logger)
	require.NoError(tc.T, err)

	// Note: JWTManager initializes keys internally based on config
	// Test keys are handled through the config's SigningKey field

	tc.JWTManager = jwtManager
	// Note: JWTManager doesn't have a Close method
	tc.AddCleanup(func() {
		// Cleanup will be handled by garbage collector
	})

	return jwtManager
}

// SetupRBACManager initializes RBAC manager for testing
func (tc *TestContext) SetupRBACManager() *auth.RBACManager {
	if tc.RBACManager != nil {
		return tc.RBACManager
	}

	config := &auth.RBACManagerConfig{
		CacheTTL:        5 * time.Minute,
		EnableHierarchy: true,
		DefaultDenyAll:  false,
	}

	tc.RBACManager = auth.NewRBACManager(config, tc.Logger)
	return tc.RBACManager
}

// SetupSessionManager initializes session manager for testing
func (tc *TestContext) SetupSessionManager() *auth.SessionManager {
	if tc.SessionManager != nil {
		return tc.SessionManager
	}

	config := &auth.SessionConfig{
		SessionTTL:       time.Hour,
		SessionTimeout:   time.Hour, // Also set the primary field
		CleanupPeriod:    time.Minute,
		CleanupInterval:  time.Minute, // Map to existing field
		CookieName:       "test-session",
		CookiePath:       "/",
		CookieDomain:     "localhost",
		SecureCookies:    false,
		HTTPOnly:         true,
		SameSiteCookies:  "Strict", // Use string instead of constant
	}

	tc.SessionManager = auth.NewSessionManager(config, tc.JWTManager, tc.RBACManager, tc.Logger)
	tc.AddCleanup(func() {
		// No cleanup needed for SessionManager
	})

	return tc.SessionManager
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
	// Convert public key to JWK format for validation but use simplified structure for testing
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

func (m *MockTokenStore) StoreToken(ctx context.Context, tokenID string, token *auth.TokenInfo) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.tokens[tokenID] = token
	return nil
}

func (m *MockTokenStore) GetToken(ctx context.Context, tokenID string) (*auth.TokenInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	if value, exists := m.tokens[tokenID]; exists {
		if token, ok := value.(*auth.TokenInfo); ok {
			return token, nil
		}
	}
	return nil, fmt.Errorf("token not found")
}

func (m *MockTokenStore) UpdateToken(ctx context.Context, tokenID string, token *auth.TokenInfo) error {
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

func (m *MockTokenStore) ListUserTokens(ctx context.Context, userID string) ([]*auth.TokenInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var tokens []*auth.TokenInfo
	for _, value := range m.tokens {
		if token, ok := value.(*auth.TokenInfo); ok && token.UserID == userID {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

func (m *MockTokenStore) DeleteUserData(ctx context.Context, userID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for tokenID, value := range m.tokens {
		if token, ok := value.(*auth.TokenInfo); ok && token.UserID == userID {
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
		if token, ok := value.(*auth.TokenInfo); ok && token.UserID == userID {
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
		if token, ok := value.(*auth.TokenInfo); ok && token.IssuedAt.Before(cutoff) {
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
		if token, ok := value.(*auth.TokenInfo); ok && now.After(token.ExpiresAt) {
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

func (m *MockTokenBlacklist) GetBlacklistAuditTrail(ctx context.Context, tokenID string) ([]auth.AuditEvent, error) {
	// Return empty audit trail for mock
	return []auth.AuditEvent{}, nil
}

func (m *MockTokenBlacklist) Close() error {
	return nil
}
