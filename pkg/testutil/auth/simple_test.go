package authtestutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserFactory(t *testing.T) {
	uf := NewUserFactory()
	require.NotNil(t, uf)

	// Test basic user creation
	user := uf.CreateBasicUser()
	require.NotNil(t, user)
	assert.NotEmpty(t, user.Subject)
	assert.NotEmpty(t, user.Email)
	assert.NotEmpty(t, user.Name)
	assert.True(t, user.EmailVerified)
	assert.Equal(t, "test", user.Provider)
}

func TestTokenFactory(t *testing.T) {
	tf := NewTokenFactory("test-issuer")
	require.NotNil(t, tf)

	// Test basic token creation
	claims := tf.CreateBasicToken("test-user")
	require.NotNil(t, claims)
	assert.Equal(t, "test-issuer", claims["iss"])
	assert.Equal(t, "test-user", claims["sub"])
	assert.NotNil(t, claims["exp"])
	assert.NotNil(t, claims["iat"])
}

func TestOAuthResponseFactory(t *testing.T) {
	of := NewOAuthResponseFactory()
	require.NotNil(t, of)

	// Test token response creation
	tokenResponse := of.CreateTokenResponse()
	require.NotNil(t, tokenResponse)
	assert.NotEmpty(t, tokenResponse.AccessToken)
	assert.NotEmpty(t, tokenResponse.RefreshToken)
	assert.Equal(t, "Bearer", tokenResponse.TokenType)
	assert.Equal(t, int64(3600), tokenResponse.ExpiresIn)
}

func TestJWTManagerMock(t *testing.T) {
	jwtManager := NewJWTManagerMock()
	require.NotNil(t, jwtManager)

	// Test that the mock is properly initialized
	assert.NotNil(t, jwtManager.blacklistedTokens)
	assert.NotNil(t, jwtManager.tokenStore)
}

func TestRBACManagerMock(t *testing.T) {
	rbacManager := NewRBACManagerMock()
	require.NotNil(t, rbacManager)

	// Test that the mock is properly initialized
	assert.NotNil(t, rbacManager.roles)
	assert.NotNil(t, rbacManager.permissions)
	assert.NotNil(t, rbacManager.roleStore)
	assert.NotNil(t, rbacManager.permissionStore)
}

func TestSessionManagerMock(t *testing.T) {
	sessionManager := NewSessionManagerMock()
	require.NotNil(t, sessionManager)

	// Test that the mock is properly initialized
	assert.NotNil(t, sessionManager.sessions)
	assert.Equal(t, 1*60*60, int(sessionManager.config.SessionTTL.Seconds())) // 1 hour
}

func TestCompleteTestSetup(t *testing.T) {
	uf, tf, of, cf, rf, pf, sf := CreateCompleteTestSetup()

	assert.NotNil(t, uf)
	assert.NotNil(t, tf)
	assert.NotNil(t, of)
	assert.NotNil(t, cf)
	assert.NotNil(t, rf)
	assert.NotNil(t, pf)
	assert.NotNil(t, sf)
}

func TestCreateTestData(t *testing.T) {
	data := CreateTestData()
	require.NotNil(t, data)

	// Test that all expected keys are present
	assert.Contains(t, data, "users")
	assert.Contains(t, data, "tokens")
	assert.Contains(t, data, "roles")
	assert.Contains(t, data, "permissions")
	assert.Contains(t, data, "sessions")

	// Test users data
	users, ok := data["users"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, users, "basic")
	assert.Contains(t, users, "admin")
	assert.Contains(t, users, "github")
}
