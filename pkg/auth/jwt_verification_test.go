package auth

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

// TestJWTClaimsStructure verifies that NephoranJWTClaims has all required time fields
func TestJWTClaimsStructure(t *testing.T) {
	claims := &NephoranJWTClaims{}

	// Verify embedded RegisteredClaims provides time fields
	now := time.Now()

	// Test ExpiresAt field access
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Hour))
	assert.NotNil(t, claims.ExpiresAt)
	assert.True(t, claims.ExpiresAt.Time.After(now))

	// Test IssuedAt field access
	claims.IssuedAt = jwt.NewNumericDate(now)
	assert.NotNil(t, claims.IssuedAt)
	assert.Equal(t, now.Unix(), claims.IssuedAt.Unix())

	// Test NotBefore field access
	claims.NotBefore = jwt.NewNumericDate(now)
	assert.NotNil(t, claims.NotBefore)
	assert.Equal(t, now.Unix(), claims.NotBefore.Unix())

	// Test that all custom fields are accessible
	claims.Email = "test@example.com"
	claims.EmailVerified = true
	claims.Name = "Test User"
	claims.TokenType = "access"
	claims.SessionID = "session-123"

	assert.Equal(t, "test@example.com", claims.Email)
	assert.True(t, claims.EmailVerified)
	assert.Equal(t, "Test User", claims.Name)
	assert.Equal(t, "access", claims.TokenType)
	assert.Equal(t, "session-123", claims.SessionID)
}

// TestJWTManagerTimeFieldHandling verifies proper time field handling in JWT operations
func TestJWTManagerTimeFieldHandling(t *testing.T) {
	// Create test JWT manager
	config := &JWTConfig{
		Issuer:     "test-issuer",
		DefaultTTL: time.Hour,
		RefreshTTL: 24 * time.Hour,
	}

	// Create in-memory stores
	tokenStore := NewMemoryTokenStore()
	blacklist := NewMemoryTokenBlacklist()
	logger := slog.Default()

	manager, err := NewJWTManager(context.Background(), config, tokenStore, blacklist, logger)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Create test user info
	userInfo := &providers.UserInfo{
		Subject:       "test-user-123",
		Email:         "test@example.com",
		EmailVerified: true,
		Name:          "Test User",
		Provider:      "test-provider",
	}

	ctx := context.Background()

	t.Run("access_token_time_fields", func(t *testing.T) {
		beforeGeneration := time.Now()

		// Generate access token
		tokenString, tokenInfo, err := manager.GenerateAccessToken(ctx, userInfo, "session-123")
		require.NoError(t, err)
		require.NotEmpty(t, tokenString)
		require.NotNil(t, tokenInfo)

		afterGeneration := time.Now()

		// Validate token and check time fields
		claims, err := manager.ValidateToken(ctx, tokenString)
		require.NoError(t, err)
		require.NotNil(t, claims)

		// Verify time fields are properly set
		assert.NotNil(t, claims.ExpiresAt, "ExpiresAt should not be nil")
		assert.NotNil(t, claims.IssuedAt, "IssuedAt should not be nil")
		assert.NotNil(t, claims.NotBefore, "NotBefore should not be nil")

		// Verify time field values are reasonable
		assert.True(t, claims.IssuedAt.Time.After(beforeGeneration.Add(-time.Second)),
			"IssuedAt should be after generation started")
		assert.True(t, claims.IssuedAt.Time.Before(afterGeneration.Add(time.Second)),
			"IssuedAt should be before generation ended")

		assert.True(t, claims.ExpiresAt.Time.After(claims.IssuedAt.Time),
			"ExpiresAt should be after IssuedAt")
		assert.True(t, claims.ExpiresAt.Time.Before(claims.IssuedAt.Time.Add(2*time.Hour)),
			"ExpiresAt should be reasonable (within 2 hours)")

		assert.True(t, claims.NotBefore.Time.Equal(claims.IssuedAt.Time) ||
			claims.NotBefore.Time.Before(claims.IssuedAt.Time),
			"NotBefore should be equal to or before IssuedAt")

		// Verify TokenInfo has correct time fields
		assert.True(t, tokenInfo.IssuedAt.After(beforeGeneration.Add(-time.Second)))
		assert.True(t, tokenInfo.ExpiresAt.After(tokenInfo.IssuedAt))
		assert.Equal(t, claims.ExpiresAt.Time.Unix(), tokenInfo.ExpiresAt.Unix())
	})

	t.Run("refresh_token_time_fields", func(t *testing.T) {
		beforeGeneration := time.Now()

		// Generate refresh token
		tokenString, tokenInfo, err := manager.GenerateRefreshToken(ctx, userInfo, "session-123")
		require.NoError(t, err)
		require.NotEmpty(t, tokenString)
		require.NotNil(t, tokenInfo)

		// Validate token and check time fields
		claims, err := manager.ValidateToken(ctx, tokenString)
		require.NoError(t, err)
		require.NotNil(t, claims)

		// Verify time fields are properly set for refresh token
		assert.NotNil(t, claims.ExpiresAt, "Refresh token ExpiresAt should not be nil")
		assert.NotNil(t, claims.IssuedAt, "Refresh token IssuedAt should not be nil")
		assert.NotNil(t, claims.NotBefore, "Refresh token NotBefore should not be nil")

		// Refresh tokens should have longer expiration
		assert.True(t, claims.ExpiresAt.Time.After(beforeGeneration.Add(20*time.Hour)),
			"Refresh token should expire much later than access token")

		assert.Equal(t, "refresh", claims.TokenType)
	})

	t.Run("custom_ttl_handling", func(t *testing.T) {
		customTTL := 30 * time.Minute

		// Generate token with custom TTL
		tokenString, _, err := manager.GenerateAccessToken(ctx, userInfo, "session-123", WithTTL(customTTL))
		require.NoError(t, err)

		// Validate and check expiration
		claims, err := manager.ValidateToken(ctx, tokenString)
		require.NoError(t, err)

		expectedExpiry := claims.IssuedAt.Time.Add(customTTL)
		actualExpiry := claims.ExpiresAt.Time

		// Allow 1 second tolerance for timing differences
		assert.True(t, actualExpiry.After(expectedExpiry.Add(-time.Second)) &&
			actualExpiry.Before(expectedExpiry.Add(time.Second)),
			"Custom TTL should be respected (expected: %v, actual: %v)", expectedExpiry, actualExpiry)
	})

	t.Run("token_expiration_validation", func(t *testing.T) {
		// Generate token with very short TTL
		shortTTL := 10 * time.Millisecond
		tokenString, _, err := manager.GenerateAccessToken(ctx, userInfo, "session-123", WithTTL(shortTTL))
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(20 * time.Millisecond)

		// Validation should fail for expired token
		_, err = manager.ValidateToken(ctx, tokenString)
		assert.Error(t, err, "Expired token validation should fail")
		assert.Contains(t, err.Error(), "token is expired", "Error should mention token expiration")
	})

	t.Run("blacklist_with_expiration", func(t *testing.T) {
		// Generate token
		tokenString, _, err := manager.GenerateAccessToken(ctx, userInfo, "session-123")
		require.NoError(t, err)

		// Validate works before blacklisting
		_, err = manager.ValidateToken(ctx, tokenString)
		require.NoError(t, err)

		// Revoke token (adds to blacklist with expiration)
		err = manager.RevokeToken(ctx, tokenString)
		require.NoError(t, err)

		// Validation should fail for blacklisted token
		_, err = manager.ValidateToken(ctx, tokenString)
		assert.Error(t, err, "Blacklisted token validation should fail")
		assert.Contains(t, err.Error(), "blacklisted", "Error should mention blacklisting")
	})
}

// TestJWTManagerBackwardCompatibility tests that legacy token generation still works
func TestJWTManagerBackwardCompatibility(t *testing.T) {
	config := &JWTConfig{
		Issuer:     "test-issuer",
		DefaultTTL: time.Hour,
		RefreshTTL: 24 * time.Hour,
	}

	tokenStore := NewMemoryTokenStore()
	blacklist := NewMemoryTokenBlacklist()
	logger := slog.Default()

	manager, err := NewJWTManager(context.Background(), config, tokenStore, blacklist, logger)
	require.NoError(t, err)

	userInfo := &providers.UserInfo{
		Subject:  "test-user-123",
		Email:    "test@example.com",
		Name:     "Test User",
		Provider: "test-provider",
	}

	// Test legacy GenerateToken method
	customClaims := map[string]interface{}{
		"custom_field": "custom_value",
		"admin":        true,
	}

	tokenString, err := manager.GenerateToken(userInfo, customClaims)
	require.NoError(t, err)
	require.NotEmpty(t, tokenString)

	// Verify we can extract claims from legacy token
	claims, err := manager.ExtractClaims(tokenString)
	require.NoError(t, err)

	// Check standard time fields are present
	assert.Contains(t, claims, "exp", "Legacy token should have 'exp' field")
	assert.Contains(t, claims, "iat", "Legacy token should have 'iat' field")
	assert.Contains(t, claims, "nbf", "Legacy token should have 'nbf' field")

	// Check custom claims are present
	assert.Equal(t, "custom_value", claims["custom_field"])
	assert.Equal(t, true, claims["admin"])

	// Verify time fields are numeric and reasonable
	exp, ok := claims["exp"].(float64)
	require.True(t, ok, "exp should be numeric")
	iat, ok := claims["iat"].(float64)
	require.True(t, ok, "iat should be numeric")

	assert.True(t, exp > iat, "exp should be after iat")
	assert.True(t, exp > float64(time.Now().Unix()), "exp should be in the future")
}
