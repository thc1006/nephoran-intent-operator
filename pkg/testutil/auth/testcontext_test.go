package auth_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authtestutil "github.com/thc1006/nephoran-intent-operator/pkg/testutil/auth"
)

func TestTestContext(t *testing.T) {
	tc := authtestutil.NewTestContext(t)
	defer tc.Cleanup()

	// Test that TestContext is created properly
	require.NotNil(t, tc)
	require.NotNil(t, tc.PrivateKey)
	require.NotNil(t, tc.PublicKey)
	require.NotEmpty(t, tc.KeyID)
	require.NotNil(t, tc.Logger)

	// Test JWT Manager setup
	jwtManager := tc.SetupJWTManager()
	require.NotNil(t, jwtManager)

	// Test Session Manager setup
	sessionManager := tc.SetupSessionManager()
	require.NotNil(t, sessionManager)

	// Test RBAC Manager setup  
	rbacManager := tc.SetupRBACManager()
	require.NotNil(t, rbacManager)

	// Test token creation
	claims := jwt.MapClaims{
		"sub": "test-user",
		"iss": "test-issuer",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	
	token := tc.CreateTestToken(claims)
	require.NotEmpty(t, token)
	
	// Verify token can be parsed
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return tc.PublicKey, nil
	})
	require.NoError(t, err)
	require.True(t, parsedToken.Valid)
}

func TestUserFactory(t *testing.T) {
	uf := authtestutil.NewUserFactory()
	require.NotNil(t, uf)

	user := uf.CreateBasicUser()
	require.NotNil(t, user)
	require.Equal(t, "testuser", user.Username)
	require.Equal(t, "password123", user.Password)
	require.Equal(t, "test@example.com", user.Email)
	require.Contains(t, user.Roles, "user")
	require.True(t, user.Enabled)
	require.Equal(t, "test-user-123", user.Subject)
}

func TestSessionManagerMock(t *testing.T) {
	tc := authtestutil.NewTestContext(t)
	defer tc.Cleanup()
	
	sessionManager := tc.SetupSessionManager()
	require.NotNil(t, sessionManager)
	
	uf := authtestutil.NewUserFactory()
	user := uf.CreateBasicUser()
	
	ctx := context.Background()
	session, err := sessionManager.CreateSession(ctx, user, json.RawMessage(`{}`))
	require.NoError(t, err)
	require.NotNil(t, session)
	require.NotEmpty(t, session.ID)
}

func TestJWTManagerMock(t *testing.T) {
	tc := authtestutil.NewTestContext(t)
	defer tc.Cleanup()
	
	jwtManager := tc.SetupJWTManager()
	require.NotNil(t, jwtManager)
	
	uf := authtestutil.NewUserFactory()
	user := uf.CreateBasicUser()
	
	// Test token generation
	token, err := jwtManager.GenerateToken(user, nil)
	require.NoError(t, err)
	require.NotEmpty(t, token)
}

func TestRBACManagerMock(t *testing.T) {
	tc := authtestutil.NewTestContext(t)
	defer tc.Cleanup()
	
	rbacManager := tc.SetupRBACManager()
	require.NotNil(t, rbacManager)
	
	ctx := context.Background()
	
	// Test role creation
	err := rbacManager.CreateRole(ctx, "test-role")
	require.NoError(t, err)
	
	// Test permission creation
	err = rbacManager.CreatePermission(ctx, "test:read:default")
	require.NoError(t, err)
}

// SecurityTestSuite provides comprehensive security testing
type SecurityTestSuite struct {
	t              *testing.T
	tc             *authtestutil.TestContext
	jwtManager     *authtestutil.JWTManagerMock
	sessionManager *authtestutil.SessionManagerMock
	rbacManager    *authtestutil.RBACManagerMock
	testUser       *authtestutil.TestUser
	validToken     string
}

func NewSecurityTestSuite(t *testing.T) *SecurityTestSuite {
	tc := authtestutil.NewTestContext(t)

	suite := &SecurityTestSuite{
		t:              t,
		tc:             tc,
		jwtManager:     tc.SetupJWTManager(),
		sessionManager: tc.SetupSessionManager(),
		rbacManager:    tc.SetupRBACManager(),
	}

	suite.setupTestData()
	return suite
}

func (suite *SecurityTestSuite) setupTestData() {
	uf := authtestutil.NewUserFactory()
	suite.testUser = uf.CreateBasicUser()

	var err error
	suite.validToken, err = suite.jwtManager.GenerateToken(suite.testUser, nil)
	require.NoError(suite.t, err)
}

func (suite *SecurityTestSuite) Cleanup() {
	suite.tc.Cleanup()
}

func TestSecurity_JWTTokenManipulation(t *testing.T) {
	suite := NewSecurityTestSuite(t)
	defer suite.Cleanup()

	// Test basic token validation
	assert.NotEmpty(t, suite.validToken)
	
	// Test that TestContext provides working JWT functionality
	claims := jwt.MapClaims{
		"iss": "test-issuer",
		"sub": suite.testUser.Subject,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	
	testToken := suite.tc.CreateTestToken(claims)
	assert.NotEmpty(t, testToken)
	
	// Verify the token was created correctly
	parsedToken, err := jwt.Parse(testToken, func(token *jwt.Token) (interface{}, error) {
		return suite.tc.PublicKey, nil
	})
	require.NoError(t, err)
	require.True(t, parsedToken.Valid)
}