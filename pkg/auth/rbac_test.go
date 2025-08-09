package auth

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

func TestNewRBACManager(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Test with default config
	manager := NewRBACManager(nil, logger)
	if manager == nil {
		t.Fatal("Expected RBACManager to be created")
	}

	// Verify default roles are loaded
	roles := manager.ListRoles(context.Background())
	if len(roles) != 4 {
		t.Errorf("Expected 4 default roles, got %d", len(roles))
	}

	// Verify default permissions are loaded
	permissions := manager.ListPermissions(context.Background())
	if len(permissions) < 10 {
		t.Errorf("Expected at least 10 default permissions, got %d", len(permissions))
	}
}

func TestRoleManagement(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewRBACManager(nil, logger)
	ctx := context.Background()

	// Test creating a custom role
	customRole := &Role{
		ID:          "custom-role",
		Name:        "Custom Role",
		Description: "A custom role for testing",
		Permissions: []string{"intent:read", "e2:read"},
	}

	err := manager.CreateRole(ctx, customRole)
	if err != nil {
		t.Fatalf("Failed to create custom role: %v", err)
	}

	// Test retrieving the role
	retrievedRole, err := manager.GetRole(ctx, "custom-role")
	if err != nil {
		t.Fatalf("Failed to retrieve custom role: %v", err)
	}

	if retrievedRole.Name != "Custom Role" {
		t.Errorf("Expected role name 'Custom Role', got '%s'", retrievedRole.Name)
	}

	// Test updating the role
	customRole.Description = "Updated description"
	err = manager.UpdateRole(ctx, customRole)
	if err != nil {
		t.Fatalf("Failed to update custom role: %v", err)
	}

	// Test role assignment to user
	userID := "test-user"
	err = manager.GrantRoleToUser(ctx, userID, "custom-role")
	if err != nil {
		t.Fatalf("Failed to grant role to user: %v", err)
	}

	// Verify user has the role
	userRoles := manager.GetUserRoles(ctx, userID)
	if len(userRoles) != 1 || userRoles[0] != "custom-role" {
		t.Errorf("Expected user to have 'custom-role', got %v", userRoles)
	}

	// Test role revocation
	err = manager.RevokeRoleFromUser(ctx, userID, "custom-role")
	if err != nil {
		t.Fatalf("Failed to revoke role from user: %v", err)
	}

	userRoles = manager.GetUserRoles(ctx, userID)
	if len(userRoles) != 0 {
		t.Errorf("Expected user to have no roles, got %v", userRoles)
	}

	// Test deleting the role
	err = manager.DeleteRole(ctx, "custom-role")
	if err != nil {
		t.Fatalf("Failed to delete custom role: %v", err)
	}
}

func TestPermissionChecking(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewRBACManager(nil, logger)
	ctx := context.Background()
	userID := "test-user"

	// Grant intent-operator role to user
	err := manager.GrantRoleToUser(ctx, userID, "intent-operator")
	if err != nil {
		t.Fatalf("Failed to grant role: %v", err)
	}

	// Test specific permission
	hasPermission := manager.CheckPermission(ctx, userID, "intent:read")
	if !hasPermission {
		t.Error("Expected user to have intent:read permission")
	}

	// Test permission user doesn't have
	hasPermission = manager.CheckPermission(ctx, userID, "system:admin")
	if hasPermission {
		t.Error("Expected user to NOT have system:admin permission")
	}

	// Test wildcard permission by granting system-admin role
	err = manager.GrantRoleToUser(ctx, userID, "system-admin")
	if err != nil {
		t.Fatalf("Failed to grant system-admin role: %v", err)
	}

	hasPermission = manager.CheckPermission(ctx, userID, "anything:any-action")
	if !hasPermission {
		t.Error("Expected system-admin to have wildcard permissions")
	}
}

func TestAccessControl(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewRBACManager(nil, logger)
	ctx := context.Background()
	userID := "test-user"

	// Grant e2-manager role
	err := manager.GrantRoleToUser(ctx, userID, "e2-manager")
	if err != nil {
		t.Fatalf("Failed to grant role: %v", err)
	}

	// Test allowed access
	request := &AccessRequest{
		UserID:    userID,
		Resource:  "e2",
		Action:    "read",
		Timestamp: time.Now(),
	}

	decision := manager.CheckAccess(ctx, request)
	if !decision.Allowed {
		t.Errorf("Expected access to be allowed, got: %s", decision.Reason)
	}

	// Test denied access
	request.Resource = "system"
	request.Action = "admin"

	decision = manager.CheckAccess(ctx, request)
	if decision.Allowed {
		t.Errorf("Expected access to be denied for system:admin")
	}
}

func TestRoleMappingFromProviders(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewRBACManager(nil, logger)
	ctx := context.Background()

	// Test GitHub role mapping
	githubUser := &providers.UserInfo{
		Subject:  "github-123",
		Provider: "github",
		Organizations: []providers.Organization{
			{Name: "admin-org", Role: "owner"},
		},
		Groups: []string{"admin-org/admin", "admin-org/developers"},
	}

	err := manager.AssignRolesFromClaims(ctx, githubUser)
	if err != nil {
		t.Fatalf("Failed to assign roles from GitHub claims: %v", err)
	}

	roles := manager.GetUserRoles(ctx, githubUser.Subject)
	if len(roles) == 0 {
		t.Error("Expected roles to be assigned from GitHub claims")
	}

	// Test Azure AD role mapping
	azureUser := &providers.UserInfo{
		Subject:  "azure-456",
		Provider: "azuread",
		Roles:    []string{"Global Administrator", "User Administrator"},
		Groups:   []string{"IT-Admin", "Network-Operators"},
	}

	err = manager.AssignRolesFromClaims(ctx, azureUser)
	if err != nil {
		t.Fatalf("Failed to assign roles from Azure AD claims: %v", err)
	}

	roles = manager.GetUserRoles(ctx, azureUser.Subject)
	found := false
	for _, role := range roles {
		if role == "system-admin" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected system-admin role to be assigned to Global Administrator")
	}
}

func TestPermissionCaching(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := &RBACConfig{
		CacheTTL: 100 * time.Millisecond, // Short TTL for testing
	}
	manager := NewRBACManager(config, logger)
	ctx := context.Background()
	userID := "test-user"

	// Grant role
	err := manager.GrantRoleToUser(ctx, userID, "intent-operator")
	if err != nil {
		t.Fatalf("Failed to grant role: %v", err)
	}

	// First call - should cache
	permissions1 := manager.GetUserPermissions(ctx, userID)

	// Second call - should use cache
	permissions2 := manager.GetUserPermissions(ctx, userID)

	if len(permissions1) != len(permissions2) {
		t.Error("Cache inconsistency detected")
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Grant additional role
	err = manager.GrantRoleToUser(ctx, userID, "read-only")
	if err != nil {
		t.Fatalf("Failed to grant additional role: %v", err)
	}

	// Should recalculate permissions
	permissions3 := manager.GetUserPermissions(ctx, userID)
	if len(permissions3) <= len(permissions1) {
		t.Error("Expected more permissions after granting additional role")
	}
}

func TestWildcardPermissions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewRBACManager(nil, logger)

	tests := []struct {
		granted  string
		required string
		expected bool
	}{
		{"*", "anything:action", true},
		{"intent:*", "intent:read", true},
		{"intent:*", "intent:write", true},
		{"intent:*", "e2:read", false},
		{"intent:read", "intent:read", true},
		{"intent:read", "intent:write", false},
		{"system:*", "system:admin", true},
		{"read:*", "write:data", false},
	}

	for _, test := range tests {
		result := manager.matchesPermission(test.granted, test.required)
		if result != test.expected {
			t.Errorf("Permission match failed: granted='%s', required='%s', expected=%v, got=%v",
				test.granted, test.required, test.expected, result)
		}
	}
}

func TestRBACStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewRBACManager(nil, logger)
	ctx := context.Background()

	status := manager.GetRBACStatus(ctx)

	expectedKeys := []string{
		"roles_count", "permissions_count", "users_count",
		"policies_count", "cache_ttl", "cache_entries", "last_cache_update",
	}

	for _, key := range expectedKeys {
		if _, exists := status[key]; !exists {
			t.Errorf("Expected status to contain key '%s'", key)
		}
	}

	// Verify counts are reasonable
	if rolesCount, ok := status["roles_count"].(int); ok && rolesCount < 4 {
		t.Errorf("Expected at least 4 default roles, got %d", rolesCount)
	}

	if permsCount, ok := status["permissions_count"].(int); ok && permsCount < 10 {
		t.Errorf("Expected at least 10 default permissions, got %d", permsCount)
	}
}

func BenchmarkPermissionCheck(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewRBACManager(nil, logger)
	ctx := context.Background()
	userID := "bench-user"

	// Setup user with multiple roles
	manager.GrantRoleToUser(ctx, userID, "intent-operator")
	manager.GrantRoleToUser(ctx, userID, "e2-manager")
	manager.GrantRoleToUser(ctx, userID, "read-only")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.CheckPermission(ctx, userID, "intent:read")
	}
}

func BenchmarkAccessCheck(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	manager := NewRBACManager(nil, logger)
	ctx := context.Background()
	userID := "bench-user"

	manager.GrantRoleToUser(ctx, userID, "intent-operator")

	request := &AccessRequest{
		UserID:    userID,
		Resource:  "intent",
		Action:    "read",
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.CheckAccess(ctx, request)
	}
}
