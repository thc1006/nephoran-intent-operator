package auth_test

import (
	"context"
	"crypto/rsa"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth/testutil"
)

// Mock implementations for testing
type mockTokenStore struct{}

func (m *mockTokenStore) StoreToken(ctx context.Context, tokenID string, token *auth.TokenInfo) error {
	return nil
}

func (m *mockTokenStore) GetToken(ctx context.Context, tokenID string) (*auth.TokenInfo, error) {
	return nil, nil
}

func (m *mockTokenStore) UpdateToken(ctx context.Context, tokenID string, token *auth.TokenInfo) error {
	return nil
}

func (m *mockTokenStore) DeleteToken(ctx context.Context, tokenID string) error {
	return nil
}

func (m *mockTokenStore) ListUserTokens(ctx context.Context, userID string) ([]*auth.TokenInfo, error) {
	return nil, nil
}

func (m *mockTokenStore) CleanupExpired(ctx context.Context) error {
	return nil
}

type mockTokenBlacklist struct {
	blacklisted map[string]time.Time
	mutex       sync.RWMutex
}

func newMockTokenBlacklist() *mockTokenBlacklist {
	return &mockTokenBlacklist{
		blacklisted: make(map[string]time.Time),
	}
}

func (m *mockTokenBlacklist) BlacklistToken(ctx context.Context, tokenID string, expiresAt time.Time) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.blacklisted[tokenID] = expiresAt
	return nil
}

func (m *mockTokenBlacklist) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	_, exists := m.blacklisted[tokenID]
	return exists, nil
}

func (m *mockTokenBlacklist) CleanupExpired(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	now := time.Now()
	for tokenID, expiresAt := range m.blacklisted {
		if now.After(expiresAt) {
			delete(m.blacklisted, tokenID)
		}
	}
	return nil
}

func TestNewJWTManager(t *testing.T) {
	tests := []struct {
		name        string
		config      *auth.JWTConfig
		expectError bool
		checkConfig func(*testing.T, *auth.JWTManager)
	}{
		{
			name: "Valid configuration",
			config: &auth.JWTConfig{
				Issuer:               "test-issuer",
				DefaultTTL:           time.Hour,
				RefreshTTL:           24 * time.Hour,
				KeyRotationPeriod:    7 * 24 * time.Hour,
				RequireSecureCookies: false,
				CookieDomain:         "localhost",
				CookiePath:           "/",
				Algorithm:            "RS256",
			},
			expectError: false,
			checkConfig: func(t *testing.T, manager *auth.JWTManager) {
				assert.Equal(t, "test-issuer", manager.GetIssuer())
				assert.Equal(t, time.Hour, manager.GetDefaultTTL())
				assert.Equal(t, 24*time.Hour, manager.GetRefreshTTL())
				assert.False(t, manager.GetRequireSecureCookies())
			},
		},
		{
			name:        "Nil configuration uses defaults",
			config:      nil,
			expectError: false,
			checkConfig: func(t *testing.T, manager *auth.JWTManager) {
				assert.Equal(t, "nephoran-intent-operator", manager.GetIssuer())
				assert.Equal(t, time.Hour, manager.GetDefaultTTL())
				assert.Equal(t, 24*time.Hour, manager.GetRefreshTTL())
			},
		},
		{
			name: "Configuration with short TTL",
			config: &auth.JWTConfig{
				Issuer:     "test-issuer",
				DefaultTTL: 5 * time.Minute,
				RefreshTTL: 30 * time.Minute,
			},
			expectError: false,
			checkConfig: func(t *testing.T, manager *auth.JWTManager) {
				assert.Equal(t, 5*time.Minute, manager.GetDefaultTTL())
				assert.Equal(t, 30*time.Minute, manager.GetRefreshTTL())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := testutil.NewTestContext(t)
			defer tc.Cleanup()

			// Create mock implementations for test
		mockStore := &mockTokenStore{}
		mockBlacklist := newMockTokenBlacklist()
		manager, err := auth.NewJWTManager(tt.config, mockStore, mockBlacklist, tc.Logger)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, manager)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, manager)
			if tt.checkConfig != nil {
				tt.checkConfig(t, manager)
			}
		})
	}
}

func TestJWTManager_SetSigningKey(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	// Create real JWT manager for this test
	config := &auth.JWTConfig{
		Issuer:     "test-issuer",
		DefaultTTL: time.Hour,
		RefreshTTL: 24 * time.Hour,
	}
	mockStore := &mockTokenStore{}
	mockBlacklist := newMockTokenBlacklist()
	manager, err := auth.NewJWTManager(config, mockStore, mockBlacklist, tc.Logger)
	require.NoError(t, err)

	tests := []struct {
		name        string
		privateKey  *rsa.PrivateKey
		keyID       string
		expectError bool
	}{
		{
			name:        "Valid key",
			privateKey:  tc.PrivateKey,
			keyID:       "test-key-1",
			expectError: false,
		},
		{
			name:        "Nil key",
			privateKey:  nil,
			keyID:       "test-key-2",
			expectError: true,
		},
		{
			name:        "Empty key ID",
			privateKey:  tc.PrivateKey,
			keyID:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.SetSigningKey(tt.privateKey, tt.keyID)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestJWTManager_GenerateToken(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)

	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()

	tests := []struct {
		name         string
		userInfo     interface{}
		customClaims map[string]interface{}
		ttl          *time.Duration
		expectError  bool
		checkToken   func(*testing.T, string)
	}{
		{
			name:        "Valid token generation",
			userInfo:    user,
			expectError: false,
			checkToken: func(t *testing.T, tokenStr string) {
				assert.NotEmpty(t, tokenStr)

				// Parse and validate token
				token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
					return tc.PublicKey, nil
				})
				require.NoError(t, err)
				assert.True(t, token.Valid)

				claims := token.Claims.(jwt.MapClaims)
				assert.Equal(t, user.Subject, claims["sub"])
				assert.Equal(t, user.Email, claims["email"])
				assert.Equal(t, "test-issuer", claims["iss"])
			},
		},
		{
			name:     "Token with custom claims",
			userInfo: user,
			customClaims: map[string]interface{}{
				"custom_field": "custom_value",
				"roles":        []string{"admin", "user"},
			},
			expectError: false,
			checkToken: func(t *testing.T, tokenStr string) {
				token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
					return tc.PublicKey, nil
				})
				require.NoError(t, err)

				claims := token.Claims.(jwt.MapClaims)
				assert.Equal(t, "custom_value", claims["custom_field"])

				roles, ok := claims["roles"].([]interface{})
				assert.True(t, ok)
				assert.Len(t, roles, 2)
			},
		},
		{
			name:        "Token with custom TTL",
			userInfo:    user,
			ttl:         &[]time.Duration{30 * time.Minute}[0],
			expectError: false,
			checkToken: func(t *testing.T, tokenStr string) {
				token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
					return tc.PublicKey, nil
				})
				require.NoError(t, err)

				claims := token.Claims.(jwt.MapClaims)
				exp := claims["exp"].(float64)
				iat := claims["iat"].(float64)

				// Check that the expiration is approximately 30 minutes from issued time
				expectedExp := iat + (30 * 60)          // 30 minutes in seconds
				assert.InDelta(t, expectedExp, exp, 60) // Allow 1 minute tolerance
			},
		},
		{
			name:        "Nil user info",
			userInfo:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tokenStr string
			var err error

			if tt.ttl != nil {
				if tt.userInfo != nil {
					userInfo := tt.userInfo.(*providers.UserInfo)
					tokenStr, err = manager.GenerateTokenWithTTL(userInfo, tt.customClaims, *tt.ttl)
				} else {
					tokenStr, err = manager.GenerateTokenWithTTL(nil, tt.customClaims, *tt.ttl)
				}
			} else {
				if tt.userInfo != nil {
					userInfo := tt.userInfo.(*providers.UserInfo)
					tokenStr, err = manager.GenerateToken(userInfo, tt.customClaims)
				} else {
					tokenStr, err = manager.GenerateToken(nil, tt.customClaims)
				}
			}

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, tokenStr)
				return
			}

			assert.NoError(t, err)
			if tt.checkToken != nil {
				tt.checkToken(t, tokenStr)
			}
		})
	}
}

func TestJWTManager_ValidateToken(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	// Create real JWT manager for this test
	config := &auth.JWTConfig{
		Issuer:     "test-issuer",
		DefaultTTL: time.Hour,
		RefreshTTL: 24 * time.Hour,
	}
	mockStore := &mockTokenStore{}
	mockBlacklist := newMockTokenBlacklist()
	manager, err := auth.NewJWTManager(config, mockStore, mockBlacklist, tc.Logger)
	require.NoError(t, err)

	uf := testutil.NewUserFactory()
	tf := testutil.NewTokenFactory("test-issuer")

	// Generate a valid token
	user := uf.CreateBasicUser()
	validToken, _, err := manager.GenerateAccessToken(context.Background(), user, "test-session")
	require.NoError(t, err)

	// Create test tokens
	expiredClaims := tf.CreateExpiredToken(user.Subject)
	expiredToken := tc.CreateTestToken(expiredClaims)

	futureClaims := tf.CreateTokenNotValidYet(user.Subject)
	futureToken := tc.CreateTestToken(futureClaims)

	tests := []struct {
		name        string
		token       string
		expectError bool
		checkClaims func(*testing.T, *auth.NephoranJWTClaims)
	}{
		{
			name:        "Valid token",
			token:       validToken,
			expectError: false,
			checkClaims: func(t *testing.T, claims *auth.NephoranJWTClaims) {
				assert.Equal(t, user.Subject, claims.Subject)
				assert.Equal(t, user.Email, claims.Email)
				assert.Equal(t, "test-issuer", claims.Issuer)
			},
		},
		{
			name:        "Expired token",
			token:       expiredToken,
			expectError: true,
		},
		{
			name:        "Future token (not valid yet)",
			token:       futureToken,
			expectError: true,
		},
		{
			name:        "Malformed token",
			token:       "invalid.token.string",
			expectError: true,
		},
		{
			name:        "Empty token",
			token:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateToken(context.Background(), tt.token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, claims)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, claims)
			if tt.checkClaims != nil {
				tt.checkClaims(t, claims)
			}
		})
	}
}

func TestJWTManager_RefreshToken(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)
	uf := testutil.NewUserFactory()

	user := uf.CreateBasicUser()

	// Generate initial tokens
	accessToken, refreshTokenStr, err := manager.GenerateTokenPair(context.Background(), user, "test-session")
	require.NoError(t, err)
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshTokenStr)

	tests := []struct {
		name         string
		refreshToken string
		expectError  bool
		checkTokens  func(*testing.T, string, string)
	}{
		{
			name:         "Valid refresh token",
			refreshToken: refreshTokenStr,
			expectError:  false,
			checkTokens: func(t *testing.T, newAccess, newRefresh string) {
				assert.NotEmpty(t, newAccess)
				assert.NotEmpty(t, newRefresh)
				assert.NotEqual(t, accessToken, newAccess)
				assert.NotEqual(t, refreshTokenStr, newRefresh)

				// Validate new access token
				claims, err := manager.ValidateToken(context.Background(), newAccess)
				assert.NoError(t, err)
				assert.Equal(t, user.Subject, claims.Subject)
			},
		},
		{
			name:         "Invalid refresh token",
			refreshToken: "invalid-refresh-token",
			expectError:  true,
		},
		{
			name:         "Empty refresh token",
			refreshToken: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newAccess, newRefresh, err := manager.RefreshToken(tt.refreshToken)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, newAccess)
				assert.Empty(t, newRefresh)
				return
			}

			assert.NoError(t, err)
			if tt.checkTokens != nil {
				tt.checkTokens(t, newAccess, newRefresh)
			}
		})
	}
}

func TestJWTManager_BlacklistToken(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)
	uf := testutil.NewUserFactory()

	user := uf.CreateBasicUser()
	token, err := manager.GenerateToken(user, nil)
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "Valid token blacklisting",
			token:       token,
			expectError: false,
		},
		{
			name:        "Invalid token blacklisting",
			token:       "invalid.token.string",
			expectError: true,
		},
		{
			name:        "Empty token",
			token:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.BlacklistToken(tt.token)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Verify token is blacklisted
			isBlacklisted, err := manager.IsTokenBlacklisted(context.Background(), tt.token)
			assert.NoError(t, err)
			assert.True(t, isBlacklisted)

			// Verify blacklisted token fails validation
			_, err = manager.ValidateToken(context.Background(), tt.token)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "blacklisted")
		})
	}
}

func TestJWTManager_GetPublicKey(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)

	tests := []struct {
		name        string
		keyID       string
		expectError bool
	}{
		{
			name:        "Get existing public key",
			keyID:       tc.KeyID,
			expectError: false,
		},
		{
			name:        "Get non-existent key",
			keyID:       "non-existent-key",
			expectError: true,
		},
		{
			name:        "Empty key ID",
			keyID:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publicKey, err := manager.GetPublicKey(tt.keyID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, publicKey)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, publicKey)
			assert.Equal(t, tc.PublicKey, publicKey)
		})
	}
}

func TestJWTManager_GetJWKS(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)

	jwks, err := manager.GetJWKS()
	assert.NoError(t, err)
	assert.NotNil(t, jwks)
	
	keys, ok := jwks["keys"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, keys, 1)

	key, ok := keys[0].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "RSA", key["kty"])
	assert.Equal(t, manager.GetKeyID(), key["kid"])
	assert.Equal(t, "sig", key["use"])
	assert.Equal(t, "RS256", key["alg"])
}

func TestJWTManager_RotateKeys(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)

	// Get initial key ID
	initialKeyID := manager.GetKeyID()

	// Rotate keys
	err := manager.RotateKeys()
	assert.NoError(t, err)

	// Verify key ID changed
	assert.NotEqual(t, initialKeyID, manager.GetKeyID())

	// Verify we can still generate and validate tokens
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()

	token, err := manager.GenerateToken(user, nil)
	assert.NoError(t, err)

	claims, err := manager.ValidateToken(context.Background(), token)
	assert.NoError(t, err)
	assert.Equal(t, user.Subject, claims.Subject)
}

func TestJWTManager_ExtractClaims(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)
	uf := testutil.NewUserFactory()

	user := uf.CreateBasicUser()
	customClaims := map[string]interface{}{
		"roles":       []string{"admin", "user"},
		"permissions": []string{"read", "write"},
		"department":  "engineering",
	}

	token, err := manager.GenerateToken(user, customClaims)
	require.NoError(t, err)

	tests := []struct {
		name         string
		token        string
		expectedKeys []string
		expectError  bool
	}{
		{
			name:         "Valid token extraction",
			token:        token,
			expectedKeys: []string{"sub", "email", "roles", "permissions", "department", "iss", "exp", "iat"},
			expectError:  false,
		},
		{
			name:        "Invalid token",
			token:       "invalid.token",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ExtractClaims(tt.token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, claims)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, claims)

			for _, key := range tt.expectedKeys {
				assert.Contains(t, claims, key, "Expected claim %s not found", key)
			}

			// Verify custom claims
			assert.Equal(t, []interface{}{"admin", "user"}, claims["roles"])
			assert.Equal(t, []interface{}{"read", "write"}, claims["permissions"])
			assert.Equal(t, "engineering", claims["department"])
		})
	}
}

func TestJWTManager_GenerateTokenPair(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	// Create real JWT manager for this test
	config := &auth.JWTConfig{
		Issuer:     "test-issuer",
		DefaultTTL: time.Hour,
		RefreshTTL: 24 * time.Hour,
	}
	mockStore := &mockTokenStore{}
	mockBlacklist := newMockTokenBlacklist()
	manager, err := auth.NewJWTManager(config, mockStore, mockBlacklist, tc.Logger)
	require.NoError(t, err)

	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()

	tests := []struct {
		name         string
		userInfo     interface{}
		customClaims map[string]interface{}
		expectError  bool
		checkTokens  func(*testing.T, string, string)
	}{
		{
			name:        "Valid token pair generation",
			userInfo:    user,
			expectError: false,
			checkTokens: func(t *testing.T, accessToken, refreshToken string) {
				assert.NotEmpty(t, accessToken)
				assert.NotEmpty(t, refreshToken)
				assert.NotEqual(t, accessToken, refreshToken)

				// Validate access token
				accessClaims, err := manager.ValidateToken(context.Background(), accessToken)
				assert.NoError(t, err)
				assert.Equal(t, user.Subject, accessClaims.Subject)

				// Validate refresh token structure (without expiration validation)
				refreshClaims, err := manager.ExtractClaims(refreshToken)
				assert.NoError(t, err)
				assert.Equal(t, user.Subject, refreshClaims["sub"])
				assert.Equal(t, "refresh", refreshClaims["type"])
			},
		},
		{
			name:     "Token pair with custom claims",
			userInfo: user,
			customClaims: map[string]interface{}{
				"roles": []string{"admin"},
			},
			expectError: false,
			checkTokens: func(t *testing.T, accessToken, refreshToken string) {
				accessClaims, err := manager.ValidateToken(context.Background(), accessToken)
				assert.NoError(t, err)

				assert.NotNil(t, accessClaims.Roles)
				assert.Len(t, accessClaims.Roles, 1)
				assert.Equal(t, "admin", accessClaims.Roles[0])
			},
		},
		{
			name:        "Nil user info",
			userInfo:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var accessToken, refreshToken string
			var err error

			if tt.userInfo != nil {
				userInfo := tt.userInfo.(*providers.UserInfo)
				accessToken, refreshToken, err = manager.GenerateTokenPair(context.Background(), userInfo, "test-session")
			} else {
				accessToken, refreshToken, err = manager.GenerateTokenPair(context.Background(), nil, "test-session")
			}

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, accessToken)
				assert.Empty(t, refreshToken)
				return
			}

			assert.NoError(t, err)
			if tt.checkTokens != nil {
				tt.checkTokens(t, accessToken, refreshToken)
			}
		})
	}
}

func TestJWTManager_TokenValidationWithContext(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)
	uf := testutil.NewUserFactory()

	user := uf.CreateBasicUser()
	token, err := manager.GenerateToken(user, nil)
	require.NoError(t, err)

	tests := []struct {
		name        string
		ctx         context.Context
		token       string
		expectError bool
	}{
		{
			name:        "Valid context and token",
			ctx:         context.Background(),
			token:       token,
			expectError: false,
		},
		{
			name: "Cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			token:       token,
			expectError: true,
		},
		{
			name: "Context with timeout",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
				defer cancel()
				time.Sleep(time.Millisecond) // Ensure timeout
				return ctx
			}(),
			token:       token,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := manager.ValidateTokenWithContext(tt.ctx, tt.token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, claims)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, claims)
		})
	}
}

func TestJWTManager_CleanupBlacklist(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)
	uf := testutil.NewUserFactory()

	user := uf.CreateBasicUser()

	// Generate tokens with short TTL
	shortTTL := 100 * time.Millisecond
	token1, err := manager.GenerateTokenWithTTL(user, nil, shortTTL)
	require.NoError(t, err)

	token2, err := manager.GenerateTokenWithTTL(user, nil, time.Hour)
	require.NoError(t, err)

	// Blacklist both tokens
	err = manager.BlacklistToken(token1)
	require.NoError(t, err)

	err = manager.BlacklistToken(token2)
	require.NoError(t, err)

	// Verify both are blacklisted
	isBlacklisted1, err := manager.IsTokenBlacklisted(context.Background(), token1)
	assert.NoError(t, err)
	assert.True(t, isBlacklisted1)
	
	isBlacklisted2, err := manager.IsTokenBlacklisted(context.Background(), token2)
	assert.NoError(t, err)
	assert.True(t, isBlacklisted2)

	// Wait for first token to expire
	time.Sleep(150 * time.Millisecond)

	// Run cleanup
	err = manager.CleanupBlacklist()
	assert.NoError(t, err)

	// Verify expired token is removed from blacklist but valid token remains
	isBlacklistedAfter1, err := manager.IsTokenBlacklisted(context.Background(), token1)
	assert.NoError(t, err)
	assert.False(t, isBlacklistedAfter1) // Expired, should be cleaned up
	
	isBlacklistedAfter2, err := manager.IsTokenBlacklisted(context.Background(), token2)
	assert.NoError(t, err)
	assert.True(t, isBlacklistedAfter2)  // Still valid, should remain
}

// Benchmark tests
func BenchmarkJWTManager_GenerateToken(b *testing.B) {
	tc := testutil.NewTestContext(&testing.T{})
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.GenerateToken(user, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJWTManager_ValidateToken(b *testing.B) {
	tc := testutil.NewTestContext(&testing.T{})
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()

	token, err := manager.GenerateToken(user, nil)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.ValidateToken(context.Background(), token)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJWTManager_GenerateTokenPair(b *testing.B) {
	tc := testutil.NewTestContext(&testing.T{})
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := manager.GenerateTokenPair(context.Background(), user, "test-session")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJWTManager_RefreshToken(b *testing.B) {
	tc := testutil.NewTestContext(&testing.T{})
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()

	// Generate initial token pair
	_, refreshToken, err := manager.GenerateTokenPair(context.Background(), user, "test-session")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use the same refresh token for benchmarking
		// In real scenarios, you'd use the new refresh token
		_, _, err := manager.RefreshToken(refreshToken)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper functions for testing
func createJWTManagerForTest(t *testing.T) (*auth.JWTManager, *testutil.TestContext) {
	tc := testutil.NewTestContext(t)
	
	// Create real JWT manager for tests
	config := &auth.JWTConfig{
		Issuer:     "test-issuer",
		DefaultTTL: time.Hour,
		RefreshTTL: 24 * time.Hour,
	}
	mockStore := &mockTokenStore{}
	mockBlacklist := newMockTokenBlacklist()
	manager, err := auth.NewJWTManager(config, mockStore, mockBlacklist, tc.Logger)
	if err != nil {
		t.Fatalf("Failed to create JWT manager: %v", err)
	}

	return manager, tc
}

func setupRealJWTManager(tc *testutil.TestContext) *auth.JWTManager {
	// Create real JWT manager for tests
	config := &auth.JWTConfig{
		Issuer:     "test-issuer",
		DefaultTTL: time.Hour,
		RefreshTTL: 24 * time.Hour,
	}
	mockStore := &mockTokenStore{}
	mockBlacklist := newMockTokenBlacklist()
	manager, err := auth.NewJWTManager(config, mockStore, mockBlacklist, tc.Logger)
	if err != nil {
		tc.T.Fatalf("Failed to create JWT manager: %v", err)
	}
	
	// Set the JWT manager to use the test context's key for consistency
	err = manager.SetSigningKey(tc.PrivateKey, tc.KeyID)
	if err != nil {
		tc.T.Fatalf("Failed to set signing key: %v", err)
	}
	
	return manager
}

func generateTestTokenWithClaims(t *testing.T, manager *auth.JWTManager, claims map[string]interface{}) string {
	uf := testutil.NewUserFactory()
	user := uf.CreateBasicUser()

	token, err := manager.GenerateToken(user, claims)
	require.NoError(t, err)
	return token
}

// Table-driven test for comprehensive JWT validation scenarios
func TestJWTManager_ComprehensiveValidation(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := setupRealJWTManager(tc)
	uf := testutil.NewUserFactory()
	tf := testutil.NewTokenFactory("test-issuer")

	testCases := []struct {
		name        string
		setupToken  func() string
		expectError bool
		errorType   string
	}{
		{
			name: "Valid token",
			setupToken: func() string {
				user := uf.CreateBasicUser()
				token, _ := manager.GenerateToken(user, nil)
				return token
			},
			expectError: false,
		},
		{
			name: "Expired token",
			setupToken: func() string {
				claims := tf.CreateExpiredToken("test-user")
				return tc.CreateTestToken(claims)
			},
			expectError: true,
			errorType:   "expired",
		},
		{
			name: "Future token",
			setupToken: func() string {
				claims := tf.CreateTokenNotValidYet("test-user")
				return tc.CreateTestToken(claims)
			},
			expectError: true,
			errorType:   "not valid yet",
		},
		{
			name: "Wrong issuer",
			setupToken: func() string {
				claims := jwt.MapClaims{
					"iss": "wrong-issuer",
					"sub": "test-user",
					"exp": time.Now().Add(time.Hour).Unix(),
					"iat": time.Now().Unix(),
				}
				return tc.CreateTestToken(claims)
			},
			expectError: true,
			errorType:   "invalid issuer",
		},
		{
			name: "Missing required claims",
			setupToken: func() string {
				claims := jwt.MapClaims{
					"exp": time.Now().Add(time.Hour).Unix(),
					"iat": time.Now().Unix(),
				}
				return tc.CreateTestToken(claims)
			},
			expectError: true,
			errorType:   "missing claims",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			token := tt.setupToken()
			claims, err := manager.ValidateToken(context.Background(), token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, claims)
				if tt.errorType != "" {
					// Could add specific error type checking here
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
			}
		})
	}
}
