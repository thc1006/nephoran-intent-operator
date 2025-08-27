package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

func TestJWTManagerMock_GenerateAccessToken(t *testing.T) {
	tc := NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()

	user := tc.CreateTestUser("test-user")
	sessionID := "test-session-123"

	token, tokenInfo, err := jwtManager.GenerateAccessToken(context.Background(), user, sessionID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if token == "" {
		t.Fatal("Expected non-empty token")
	}

	if tokenInfo == nil {
		t.Fatal("Expected token info")
	}

	if tokenInfo.TokenType != "access" {
		t.Errorf("Expected token type 'access', got %s", tokenInfo.TokenType)
	}

	if tokenInfo.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, tokenInfo.SessionID)
	}
}

func TestJWTManagerMock_ValidateToken(t *testing.T) {
	tc := NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()

	user := tc.CreateTestUser("test-user")
	token, _, err := jwtManager.GenerateAccessToken(context.Background(), user, "test-session")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := jwtManager.ValidateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("Expected no error validating token, got %v", err)
	}

	if claims == nil {
		t.Fatal("Expected claims")
	}

	if claims.Subject != user.Subject {
		t.Errorf("Expected subject %s, got %s", user.Subject, claims.Subject)
	}

	if claims.TokenType != "access" {
		t.Errorf("Expected token type 'access', got %s", claims.TokenType)
	}
}

func TestJWTManagerMock_RevokeToken(t *testing.T) {
	tc := NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()

	user := tc.CreateTestUser("test-user")
	token, _, err := jwtManager.GenerateAccessToken(context.Background(), user, "test-session")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Token should be valid initially
	_, err = jwtManager.ValidateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("Token should be valid initially: %v", err)
	}

	// Revoke the token
	err = jwtManager.RevokeToken(context.Background(), token)
	if err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}

	// Token should be invalid after revocation
	_, err = jwtManager.ValidateToken(context.Background(), token)
	if err == nil {
		t.Fatal("Token should be invalid after revocation")
	}
}

func TestRBACManagerMock_CreateRoleAndGrantToUser(t *testing.T) {
	tc := NewTestContext(t)
	defer tc.Cleanup()

	rbacManager := tc.SetupRBACManager()

	// Create a permission first
	perm := &Permission{
		ID:          "test:read",
		Name:        "Test Read Permission",
		Description: "Permission to read test resources",
		Resource:    "test",
		Action:      "read",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := rbacManager.CreatePermission(context.Background(), perm)
	if err != nil {
		t.Fatalf("Failed to create permission: %v", err)
	}

	// Create a role
	role := &Role{
		ID:          "test-role",
		Name:        "Test Role",
		Description: "A test role",
		Permissions: []string{"test:read"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = rbacManager.CreateRole(context.Background(), role)
	if err != nil {
		t.Fatalf("Failed to create role: %v", err)
	}

	// Grant role to user
	userID := "test-user-123"
	err = rbacManager.GrantRoleToUser(context.Background(), userID, "test-role")
	if err != nil {
		t.Fatalf("Failed to grant role to user: %v", err)
	}

	// Check user has the role
	roles := rbacManager.GetUserRoles(context.Background(), userID)
	if len(roles) != 1 || roles[0] != "test-role" {
		t.Errorf("Expected user to have role 'test-role', got %v", roles)
	}

	// Check user has permission
	hasPermission := rbacManager.CheckPermission(context.Background(), userID, "test:read")
	if !hasPermission {
		t.Error("Expected user to have permission 'test:read'")
	}
}

func TestRBACManagerMock_GetRole(t *testing.T) {
	tc := NewTestContext(t)
	defer tc.Cleanup()

	rbacManager := tc.SetupRBACManager()

	// Create a permission first
	perm := &Permission{
		ID:          "test:read",
		Name:        "Test Read Permission",
		Description: "Permission to read test resources",
		Resource:    "test",
		Action:      "read",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := rbacManager.CreatePermission(context.Background(), perm)
	if err != nil {
		t.Fatalf("Failed to create permission: %v", err)
	}

	// Create a role
	originalRole := &Role{
		ID:          "test-role",
		Name:        "Test Role",
		Description: "A test role",
		Permissions: []string{"test:read"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = rbacManager.CreateRole(context.Background(), originalRole)
	if err != nil {
		t.Fatalf("Failed to create role: %v", err)
	}

	// Get the role
	role, err := rbacManager.GetRole(context.Background(), "test-role")
	if err != nil {
		t.Fatalf("Failed to get role: %v", err)
	}

	if role == nil {
		t.Fatal("Expected role to be returned")
	}

	if role.ID != "test-role" {
		t.Errorf("Expected role ID 'test-role', got %s", role.ID)
	}

	if role.Name != "Test Role" {
		t.Errorf("Expected role name 'Test Role', got %s", role.Name)
	}

	if len(role.Permissions) != 1 || role.Permissions[0] != "test:read" {
		t.Errorf("Expected role to have permission 'test:read', got %v", role.Permissions)
	}
}

func TestRBACManagerMock_CheckAccess(t *testing.T) {
	tc := NewTestContext(t)
	defer tc.Cleanup()

	rbacManager := tc.SetupRBACManager()

	// Create a permission first
	perm := &Permission{
		ID:          "intent:read",
		Name:        "Intent Read Permission",
		Description: "Permission to read intents",
		Resource:    "intent",
		Action:      "read",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := rbacManager.CreatePermission(context.Background(), perm)
	if err != nil {
		t.Fatalf("Failed to create permission: %v", err)
	}

	// Create a role
	role := &Role{
		ID:          "intent-reader",
		Name:        "Intent Reader",
		Description: "Can read intents",
		Permissions: []string{"intent:read"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = rbacManager.CreateRole(context.Background(), role)
	if err != nil {
		t.Fatalf("Failed to create role: %v", err)
	}

	// Grant role to user
	userID := "test-user-123"
	err = rbacManager.GrantRoleToUser(context.Background(), userID, "intent-reader")
	if err != nil {
		t.Fatalf("Failed to grant role to user: %v", err)
	}

	// Create access request
	request := &AccessRequest{
		UserID:    userID,
		Resource:  "intent",
		Action:    "read",
		Timestamp: time.Now(),
	}

	// Check access
	decision := rbacManager.CheckAccess(context.Background(), request)
	if decision == nil {
		t.Fatal("Expected access decision to be returned")
	}

	if !decision.Allowed {
		t.Errorf("Expected access to be allowed, got: %s", decision.Reason)
	}

	if decision.Reason != "Permission granted by RBAC" {
		t.Errorf("Expected reason 'Permission granted by RBAC', got: %s", decision.Reason)
	}
}

func TestSessionManagerMock_CreateAndValidateSession(t *testing.T) {
	sessionManager := NewSessionManagerMock()
	defer sessionManager.Close()

	user := &providers.UserInfo{
		Subject: "test-user",
		Email:   "test@example.com",
		Name:    "Test User",
	}

	session, err := sessionManager.CreateSession(context.Background(), user)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if session == nil {
		t.Fatal("Expected session to be returned")
	}

	if session.UserID != user.Subject {
		t.Errorf("Expected user ID %s, got %s", user.Subject, session.UserID)
	}

	// Validate session
	validatedSession, err := sessionManager.ValidateSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("Failed to validate session: %v", err)
	}

	if validatedSession.ID != session.ID {
		t.Errorf("Expected session ID %s, got %s", session.ID, validatedSession.ID)
	}
}

func TestIntegration_FullAuthFlow(t *testing.T) {
	tc := NewTestContext(t)
	defer tc.Cleanup()

	jwtManager := tc.SetupJWTManager()
	rbacManager := tc.SetupRBACManager()
	sessionManager := tc.SetupSessionManager()

	// Create user
	user := tc.CreateTestUser("integration-user")

	// Create session
	session, err := sessionManager.CreateSession(context.Background(), user)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create permission and role
	perm := &Permission{
		ID:        "test:access",
		Name:      "Test Access",
		Resource:  "test",
		Action:    "access",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = rbacManager.CreatePermission(context.Background(), perm)
	if err != nil {
		t.Fatalf("Failed to create permission: %v", err)
	}

	role := &Role{
		ID:          "test-user-role",
		Name:        "Test User",
		Permissions: []string{"test:access"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = rbacManager.CreateRole(context.Background(), role)
	if err != nil {
		t.Fatalf("Failed to create role: %v", err)
	}

	err = rbacManager.GrantRoleToUser(context.Background(), user.Subject, "test-user-role")
	if err != nil {
		t.Fatalf("Failed to grant role: %v", err)
	}

	// Generate JWT token
	token, tokenInfo, err := jwtManager.GenerateAccessToken(context.Background(), user, session.ID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate token
	claims, err := jwtManager.ValidateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// Check access
	request := &AccessRequest{
		UserID:   claims.Subject,
		Resource: "test",
		Action:   "access",
	}

	decision := rbacManager.CheckAccess(context.Background(), request)
	if !decision.Allowed {
		t.Errorf("Expected access to be allowed: %s", decision.Reason)
	}

	// Verify token info matches
	if tokenInfo.SessionID != session.ID {
		t.Errorf("Expected token session ID %s, got %s", session.ID, tokenInfo.SessionID)
	}
}
