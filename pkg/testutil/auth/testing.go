package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"log/slog"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestContext provides a comprehensive testing context for authentication tests
type TestContext struct {
	t          *testing.T
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
	KeyID      string
	Logger     *slog.Logger
}

// NewTestContext creates a new test context with all necessary components
func NewTestContext(t *testing.T) *TestContext {
	// Generate test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate test key pair: %v", err)
	}

	return &TestContext{
		t:          t,
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		KeyID:      "test-key-id",
		Logger:     NewMockSlogLogger(),
	}
}

// Cleanup cleans up test resources
func (tc *TestContext) Cleanup() {
	// Nothing to cleanup in mock implementation
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

// CreateTestToken creates a test JWT token
func (tc *TestContext) CreateTestToken(claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = tc.KeyID
	tokenString, err := token.SignedString(tc.PrivateKey)
	if err != nil {
		tc.t.Fatalf("Failed to sign test token: %v", err)
	}
	return tokenString
}

// ValidateTestToken validates a test JWT token
func (tc *TestContext) ValidateTestToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return tc.PublicKey, nil
	})
}

// GetPrivateKey returns the private key for testing
func (tc *TestContext) GetPrivateKey() *rsa.PrivateKey {
	return tc.PrivateKey
}

// GetPublicKey returns the public key for testing
func (tc *TestContext) GetPublicKey() *rsa.PublicKey {
	return tc.PublicKey
}

// GetKeyID returns the key ID for testing
func (tc *TestContext) GetKeyID() string {
	return tc.KeyID
}

// CreateUserSession creates a mock user session for testing
func (tc *TestContext) CreateUserSession(userInfo interface{}) *UserSession {
	return &UserSession{
		SessionID: "test-session-id",
		Username:  "test-user",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Active:    true,
	}
}

// TokenStore returns a mock token store
func (tc *TestContext) TokenStore() *MockTokenStore {
	return NewMockTokenStore()
}

// Blacklist returns a mock token blacklist
func (tc *TestContext) Blacklist() *MockTokenBlacklist {
	return NewMockTokenBlacklist()
}

// SlogLogger returns the test logger
func (tc *TestContext) SlogLogger() *slog.Logger {
	return tc.Logger
}

func TestStub(t *testing.T) { t.Skip("Test disabled") }
