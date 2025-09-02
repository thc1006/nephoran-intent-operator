package auth_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testutil "github.com/thc1006/nephoran-intent-operator/pkg/testutil/auth"
)

func TestRBACManager_CreateRole(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()

	tests := []struct {
		name        string
		role        *Role
		expectError bool
		checkRole   func(*testing.T, *Role)
	}{
		{
			name:        "Valid role creation",
			role:        rf.CreateBasicRole(),
			expectError: false,
			checkRole: func(t *testing.T, created *Role) {
				assert.NotEmpty(t, created.ID)
				assert.NotEmpty(t, created.Name)
				assert.NotZero(t, created.CreatedAt)
				assert.NotZero(t, created.UpdatedAt)
				assert.NotEmpty(t, created.Permissions)
			},
		},
		{
			name:        "Role with hierarchical structure",
			role:        rf.CreateHierarchicalRole([]string{"parent-role"}, []string{"child-role"}),
			expectError: false,
			checkRole: func(t *testing.T, created *Role) {
				assert.Contains(t, created.ParentRoles, "parent-role")
				assert.Contains(t, created.ChildRoles, "child-role")
			},
		},
		{
			name: "Role with duplicate name",
			role: func() *Role {
				role := rf.CreateBasicRole()
				role.Name = "duplicate-name"
				return role
			}(),
			expectError: false, // First creation should succeed
		},
		{
			name:        "Nil role",
			role:        nil,
			expectError: true,
		},
		{
			name: "Role with empty name",
			role: func() *Role {
				role := rf.CreateBasicRole()
				role.Name = ""
				return role
			}(),
			expectError: true,
		},
	}

	// Create a role to test duplicate name scenario
	duplicateRole := rf.CreateBasicRole()
	duplicateRole.Name = "duplicate-name"
	_, err := manager.CreateRole(context.Background(), duplicateRole)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			createdRole, err := manager.CreateRole(ctx, tt.role)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, createdRole)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, createdRole)
			if tt.checkRole != nil {
				tt.checkRole(t, createdRole)
			}

			// Additional test for duplicate name
			if tt.name == "Role with duplicate name" {
				// Try to create another role with same name
				duplicateRole2 := rf.CreateBasicRole()
				duplicateRole2.Name = "duplicate-name"
				_, err := manager.CreateRole(ctx, duplicateRole2)
				assert.Error(t, err, "Should not allow duplicate role names")
			}
		})
	}
}

func TestRBACManager_GetRole(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()

	// Create a test role
	testRole := rf.CreateBasicRole()
	createdRole, err := manager.CreateRole(context.Background(), testRole)
	require.NoError(t, err)

	tests := []struct {
		name        string
		roleID      string
		expectError bool
		expectNil   bool
		checkRole   func(*testing.T, *Role)
	}{
		{
			name:        "Get existing role by ID",
			roleID:      createdRole.ID,
			expectError: false,
			expectNil:   false,
			checkRole: func(t *testing.T, role *Role) {
				assert.Equal(t, createdRole.ID, role.ID)
				assert.Equal(t, createdRole.Name, role.Name)
				assert.Equal(t, createdRole.Description, role.Description)
			},
		},
		{
			name:        "Get existing role by name",
			roleID:      createdRole.Name,
			expectError: false,
			expectNil:   false,
			checkRole: func(t *testing.T, role *Role) {
				assert.Equal(t, createdRole.Name, role.Name)
			},
		},
		{
			name:        "Get non-existent role",
			roleID:      "non-existent-role",
			expectError: false,
			expectNil:   true,
		},
		{
			name:        "Empty role ID",
			roleID:      "",
			expectError: true,
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			role, err := manager.GetRole(ctx, tt.roleID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectNil {
				assert.Nil(t, role)
			} else {
				assert.NotNil(t, role)
				if tt.checkRole != nil {
					tt.checkRole(t, role)
				}
			}
		})
	}
}

func TestRBACManager_UpdateRole(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()

	// Create a test role
	originalRole := rf.CreateBasicRole()
	createdRole, err := manager.CreateRole(context.Background(), originalRole)
	require.NoError(t, err)

	tests := []struct {
		name        string
		roleID      string
		updates     *Role
		expectError bool
		checkRole   func(*testing.T, *Role, *Role)
	}{
		{
			name:   "Update role description",
			roleID: createdRole.ID,
			updates: func() *Role {
				updated := *createdRole
				updated.Description = "Updated description"
				return &updated
			}(),
			expectError: false,
			checkRole: func(t *testing.T, original, updated *Role) {
				assert.Equal(t, "Updated description", updated.Description)
				assert.Equal(t, original.ID, updated.ID)
				assert.True(t, updated.UpdatedAt.After(original.UpdatedAt))
			},
		},
		{
			name:   "Update role permissions",
			roleID: createdRole.ID,
			updates: func() *Role {
				updated := *createdRole
				updated.Permissions = []string{"read:advanced", "write:advanced"}
				return &updated
			}(),
			expectError: false,
			checkRole: func(t *testing.T, original, updated *Role) {
				assert.Contains(t, updated.Permissions, "read:advanced")
				assert.Contains(t, updated.Permissions, "write:advanced")
			},
		},
		{
			name:        "Update non-existent role",
			roleID:      "non-existent-role",
			updates:     rf.CreateBasicRole(),
			expectError: true,
		},
		{
			name:        "Update with nil role",
			roleID:      createdRole.ID,
			updates:     nil,
			expectError: true,
		},
		{
			name:        "Empty role ID",
			roleID:      "",
			updates:     rf.CreateBasicRole(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			updatedRole, err := manager.UpdateRole(ctx, tt.roleID, tt.updates)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, updatedRole)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, updatedRole)
			if tt.checkRole != nil {
				tt.checkRole(t, createdRole, updatedRole)
			}
		})
	}
}

func TestRBACManager_DeleteRole(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()

	tests := []struct {
		name         string
		setupRole    func() string
		expectError  bool
		checkDeleted func(*testing.T, string)
	}{
		{
			name: "Delete existing role",
			setupRole: func() string {
				role := rf.CreateBasicRole()
				createdRole, err := manager.CreateRole(context.Background(), role)
				require.NoError(t, err)
				return createdRole.ID
			},
			expectError: false,
			checkDeleted: func(t *testing.T, roleID string) {
				ctx := context.Background()
				deletedRole, err := manager.GetRole(ctx, roleID)
				assert.NoError(t, err)
				assert.Nil(t, deletedRole)
			},
		},
		{
			name: "Delete role with user assignments",
			setupRole: func() string {
				role := rf.CreateBasicRole()
				createdRole, err := manager.CreateRole(context.Background(), role)
				require.NoError(t, err)

				// Assign role to a user
				ctx := context.Background()
				err = manager.AssignRoleToUser(ctx, "test-user", createdRole.ID)
				require.NoError(t, err)

				return createdRole.ID
			},
			expectError: false, // Should succeed and also remove user assignments
			checkDeleted: func(t *testing.T, roleID string) {
				ctx := context.Background()

				// Role should be deleted
				deletedRole, err := manager.GetRole(ctx, roleID)
				assert.NoError(t, err)
				assert.Nil(t, deletedRole)

				// User assignments should be removed
				userRoles, err := manager.GetUserRoles(ctx, "test-user")
				assert.NoError(t, err)
				assert.NotContains(t, userRoles, roleID)
			},
		},
		{
			name: "Delete non-existent role",
			setupRole: func() string {
				return "non-existent-role"
			},
			expectError: true,
		},
		{
			name: "Delete with empty role ID",
			setupRole: func() string {
				return ""
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roleID := tt.setupRole()
			ctx := context.Background()
			err := manager.DeleteRole(ctx, roleID)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkDeleted != nil {
				tt.checkDeleted(t, roleID)
			}
		})
	}
}

func TestRBACManager_CreatePermission(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	pf := testutil.NewPermissionFactory()

	tests := []struct {
		name            string
		permission      *Permission
		expectError     bool
		checkPermission func(*testing.T, *Permission)
	}{
		{
			name:        "Valid permission creation",
			permission:  pf.CreateBasicPermission(),
			expectError: false,
			checkPermission: func(t *testing.T, created *Permission) {
				assert.NotEmpty(t, created.ID)
				assert.NotEmpty(t, created.Name)
				assert.NotEmpty(t, created.Resource)
				assert.NotEmpty(t, created.Action)
				assert.Equal(t, "allow", created.Effect)
			},
		},
		{
			name:        "Permission with deny effect",
			permission:  pf.CreateDenyPermission("sensitive-resource", "delete"),
			expectError: false,
			checkPermission: func(t *testing.T, created *Permission) {
				assert.Equal(t, "deny", created.Effect)
				assert.Equal(t, "sensitive-resource", created.Resource)
				assert.Equal(t, "delete", created.Action)
			},
		},
		{
			name:        "Nil permission",
			permission:  nil,
			expectError: true,
		},
		{
			name: "Permission with empty name",
			permission: func() *Permission {
				perm := pf.CreateBasicPermission()
				perm.Name = ""
				return perm
			}(),
			expectError: true,
		},
		{
			name: "Permission with empty resource",
			permission: func() *Permission {
				perm := pf.CreateBasicPermission()
				perm.Resource = ""
				return perm
			}(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			createdPermission, err := manager.CreatePermission(ctx, tt.permission)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, createdPermission)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, createdPermission)
			if tt.checkPermission != nil {
				tt.checkPermission(t, createdPermission)
			}
		})
	}
}

func TestRBACManager_AssignRoleToUser(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()

	// Create test roles
	basicRole := rf.CreateBasicRole()
	createdBasicRole, err := manager.CreateRole(context.Background(), basicRole)
	require.NoError(t, err)

	adminRole := rf.CreateAdminRole()
	createdAdminRole, err := manager.CreateRole(context.Background(), adminRole)
	require.NoError(t, err)

	tests := []struct {
		name            string
		userID          string
		roleID          string
		expectError     bool
		checkAssignment func(*testing.T, string, string)
	}{
		{
			name:        "Assign basic role to user",
			userID:      "test-user-1",
			roleID:      createdBasicRole.ID,
			expectError: false,
			checkAssignment: func(t *testing.T, userID, roleID string) {
				ctx := context.Background()
				userRoles, err := manager.GetUserRoles(ctx, userID)
				assert.NoError(t, err)
				assert.Contains(t, userRoles, roleID)
			},
		},
		{
			name:        "Assign admin role to user",
			userID:      "test-user-2",
			roleID:      createdAdminRole.ID,
			expectError: false,
			checkAssignment: func(t *testing.T, userID, roleID string) {
				ctx := context.Background()
				userRoles, err := manager.GetUserRoles(ctx, userID)
				assert.NoError(t, err)
				assert.Contains(t, userRoles, roleID)
			},
		},
		{
			name:        "Assign multiple roles to same user",
			userID:      "test-user-3",
			roleID:      createdBasicRole.ID,
			expectError: false,
			checkAssignment: func(t *testing.T, userID, roleID string) {
				ctx := context.Background()

				// Assign second role
				err := manager.AssignRoleToUser(ctx, userID, createdAdminRole.ID)
				assert.NoError(t, err)

				// Check both roles are assigned
				userRoles, err := manager.GetUserRoles(ctx, userID)
				assert.NoError(t, err)
				assert.Contains(t, userRoles, createdBasicRole.ID)
				assert.Contains(t, userRoles, createdAdminRole.ID)
			},
		},
		{
			name:        "Assign same role twice (should be idempotent)",
			userID:      "test-user-4",
			roleID:      createdBasicRole.ID,
			expectError: false,
			checkAssignment: func(t *testing.T, userID, roleID string) {
				ctx := context.Background()

				// Assign same role again
				err := manager.AssignRoleToUser(ctx, userID, roleID)
				assert.NoError(t, err)

				// Should still only have one instance
				userRoles, err := manager.GetUserRoles(ctx, userID)
				assert.NoError(t, err)

				count := 0
				for _, role := range userRoles {
					if role == roleID {
						count++
					}
				}
				assert.Equal(t, 1, count, "Role should only be assigned once")
			},
		},
		{
			name:        "Assign non-existent role",
			userID:      "test-user-5",
			roleID:      "non-existent-role",
			expectError: true,
		},
		{
			name:        "Assign to empty user ID",
			userID:      "",
			roleID:      createdBasicRole.ID,
			expectError: true,
		},
		{
			name:        "Assign empty role ID",
			userID:      "test-user-6",
			roleID:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := manager.AssignRoleToUser(ctx, tt.userID, tt.roleID)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkAssignment != nil {
				tt.checkAssignment(t, tt.userID, tt.roleID)
			}
		})
	}
}

func TestRBACManager_RevokeRoleFromUser(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()

	// Create and assign test role
	role := rf.CreateBasicRole()
	createdRole, err := manager.CreateRole(context.Background(), role)
	require.NoError(t, err)

	userID := "test-user-revoke"
	ctx := context.Background()
	err = manager.AssignRoleToUser(ctx, userID, createdRole.ID)
	require.NoError(t, err)

	tests := []struct {
		name         string
		userID       string
		roleID       string
		expectError  bool
		checkRevoked func(*testing.T, string, string)
	}{
		{
			name:        "Revoke assigned role",
			userID:      userID,
			roleID:      createdRole.ID,
			expectError: false,
			checkRevoked: func(t *testing.T, userID, roleID string) {
				ctx := context.Background()
				userRoles, err := manager.GetUserRoles(ctx, userID)
				assert.NoError(t, err)
				assert.NotContains(t, userRoles, roleID)
			},
		},
		{
			name:        "Revoke non-assigned role",
			userID:      "test-user-no-role",
			roleID:      createdRole.ID,
			expectError: false, // Should be idempotent
		},
		{
			name:        "Revoke non-existent role",
			userID:      userID,
			roleID:      "non-existent-role",
			expectError: true,
		},
		{
			name:        "Revoke from empty user ID",
			userID:      "",
			roleID:      createdRole.ID,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := manager.RevokeRoleFromUser(ctx, tt.userID, tt.roleID)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkRevoked != nil {
				tt.checkRevoked(t, tt.userID, tt.roleID)
			}
		})
	}
}

func TestRBACManager_CheckPermission(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()
	pf := testutil.NewPermissionFactory()

	// Create permissions
	readPermission := pf.CreateResourcePermissions("test-resource", []string{"read"})[0]
	createdReadPerm, err := manager.CreatePermission(context.Background(), readPermission)
	require.NoError(t, err)

	writePermission := pf.CreateResourcePermissions("test-resource", []string{"write"})[0]
	createdWritePerm, err := manager.CreatePermission(context.Background(), writePermission)
	require.NoError(t, err)

	denyPermission := pf.CreateDenyPermission("sensitive-resource", "delete")
	createdDenyPerm, err := manager.CreatePermission(context.Background(), denyPermission)
	require.NoError(t, err)

	// Create roles with permissions
	readerRole := rf.CreateRoleWithPermissions([]string{createdReadPerm.ID})
	createdReaderRole, err := manager.CreateRole(context.Background(), readerRole)
	require.NoError(t, err)

	writerRole := rf.CreateRoleWithPermissions([]string{createdReadPerm.ID, createdWritePerm.ID})
	createdWriterRole, err := manager.CreateRole(context.Background(), writerRole)
	require.NoError(t, err)

	restrictedRole := rf.CreateRoleWithPermissions([]string{createdReadPerm.ID, createdDenyPerm.ID})
	createdRestrictedRole, err := manager.CreateRole(context.Background(), restrictedRole)
	require.NoError(t, err)

	// Assign roles to users
	ctx := context.Background()
	err = manager.AssignRoleToUser(ctx, "reader-user", createdReaderRole.ID)
	require.NoError(t, err)

	err = manager.AssignRoleToUser(ctx, "writer-user", createdWriterRole.ID)
	require.NoError(t, err)

	err = manager.AssignRoleToUser(ctx, "restricted-user", createdRestrictedRole.ID)
	require.NoError(t, err)

	tests := []struct {
		name          string
		userID        string
		resource      string
		action        string
		expectAllowed bool
		expectError   bool
	}{
		{
			name:          "Reader can read",
			userID:        "reader-user",
			resource:      "test-resource",
			action:        "read",
			expectAllowed: true,
			expectError:   false,
		},
		{
			name:          "Reader cannot write",
			userID:        "reader-user",
			resource:      "test-resource",
			action:        "write",
			expectAllowed: false,
			expectError:   false,
		},
		{
			name:          "Writer can read and write",
			userID:        "writer-user",
			resource:      "test-resource",
			action:        "write",
			expectAllowed: true,
			expectError:   false,
		},
		{
			name:          "User with deny permission is denied",
			userID:        "restricted-user",
			resource:      "sensitive-resource",
			action:        "delete",
			expectAllowed: false,
			expectError:   false,
		},
		{
			name:          "User without role has no permissions",
			userID:        "no-role-user",
			resource:      "test-resource",
			action:        "read",
			expectAllowed: false,
			expectError:   false,
		},
		{
			name:          "Non-existent user",
			userID:        "non-existent-user",
			resource:      "test-resource",
			action:        "read",
			expectAllowed: false,
			expectError:   false,
		},
		{
			name:        "Empty user ID",
			userID:      "",
			resource:    "test-resource",
			action:      "read",
			expectError: true,
		},
		{
			name:        "Empty resource",
			userID:      "reader-user",
			resource:    "",
			action:      "read",
			expectError: true,
		},
		{
			name:        "Empty action",
			userID:      "reader-user",
			resource:    "test-resource",
			action:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			allowed, err := manager.CheckPermission(ctx, tt.userID, tt.resource, tt.action)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectAllowed, allowed)
		})
	}
}

func TestRBACManager_HierarchicalRoles(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()
	pf := testutil.NewPermissionFactory()

	// Create permissions
	basicPerms := pf.CreateResourcePermissions("basic-resource", []string{"read"})
	basicPerm, err := manager.CreatePermission(context.Background(), basicPerms[0])
	require.NoError(t, err)

	adminPerms := pf.CreateResourcePermissions("admin-resource", []string{"read", "write", "delete"})
	var createdAdminPerms []*Permission
	for _, perm := range adminPerms {
		created, err := manager.CreatePermission(context.Background(), perm)
		require.NoError(t, err)
		createdAdminPerms = append(createdAdminPerms, created)
	}

	// Create hierarchical roles
	ctx := context.Background()

	// Basic role
	basicRole := rf.CreateRoleWithPermissions([]string{basicPerm.ID})
	basicRole.Name = "basic"
	createdBasicRole, err := manager.CreateRole(ctx, basicRole)
	require.NoError(t, err)

	// Admin role that inherits from basic
	adminPermIDs := make([]string, len(createdAdminPerms))
	for i, perm := range createdAdminPerms {
		adminPermIDs[i] = perm.ID
	}

	adminRole := rf.CreateRoleWithPermissions(adminPermIDs)
	adminRole.Name = "admin"
	adminRole.ParentRoles = []string{createdBasicRole.ID}
	createdAdminRole, err := manager.CreateRole(ctx, adminRole)
	require.NoError(t, err)

	// Super admin role that inherits from admin
	superAdminRole := rf.CreateBasicRole()
	superAdminRole.Name = "superadmin"
	superAdminRole.ParentRoles = []string{createdAdminRole.ID}
	createdSuperAdminRole, err := manager.CreateRole(ctx, superAdminRole)
	require.NoError(t, err)

	// Assign roles to users
	err = manager.AssignRoleToUser(ctx, "basic-user", createdBasicRole.ID)
	require.NoError(t, err)

	err = manager.AssignRoleToUser(ctx, "admin-user", createdAdminRole.ID)
	require.NoError(t, err)

	err = manager.AssignRoleToUser(ctx, "superadmin-user", createdSuperAdminRole.ID)
	require.NoError(t, err)

	tests := []struct {
		name          string
		userID        string
		resource      string
		action        string
		expectAllowed bool
	}{
		// Basic user tests
		{
			name:          "Basic user has basic permissions",
			userID:        "basic-user",
			resource:      "basic-resource",
			action:        "read",
			expectAllowed: true,
		},
		{
			name:          "Basic user cannot access admin resource",
			userID:        "basic-user",
			resource:      "admin-resource",
			action:        "read",
			expectAllowed: false,
		},

		// Admin user tests (should inherit basic permissions)
		{
			name:          "Admin user has basic permissions through inheritance",
			userID:        "admin-user",
			resource:      "basic-resource",
			action:        "read",
			expectAllowed: true,
		},
		{
			name:          "Admin user has admin read permission",
			userID:        "admin-user",
			resource:      "admin-resource",
			action:        "read",
			expectAllowed: true,
		},
		{
			name:          "Admin user has admin write permission",
			userID:        "admin-user",
			resource:      "admin-resource",
			action:        "write",
			expectAllowed: true,
		},

		// Super admin user tests (should inherit all permissions)
		{
			name:          "Super admin has basic permissions through inheritance",
			userID:        "superadmin-user",
			resource:      "basic-resource",
			action:        "read",
			expectAllowed: true,
		},
		{
			name:          "Super admin has admin permissions through inheritance",
			userID:        "superadmin-user",
			resource:      "admin-resource",
			action:        "delete",
			expectAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			allowed, err := manager.CheckPermission(ctx, tt.userID, tt.resource, tt.action)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectAllowed, allowed, "Permission check failed for %s on %s:%s", tt.userID, tt.resource, tt.action)
		})
	}
}

func TestRBACManager_ListOperations(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()
	pf := testutil.NewPermissionFactory()

	// Create test data
	ctx := context.Background()

	// Create roles
	role1 := rf.CreateBasicRole()
	role1.Name = "test-role-1"
	createdRole1, err := manager.CreateRole(ctx, role1)
	require.NoError(t, err)

	role2 := rf.CreateAdminRole()
	role2.Name = "test-role-2"
	createdRole2, err := manager.CreateRole(ctx, role2)
	require.NoError(t, err)

	// Create permissions
	perm1 := pf.CreateBasicPermission()
	createdPerm1, err := manager.CreatePermission(ctx, perm1)
	require.NoError(t, err)

	perm2 := pf.CreateDenyPermission("test-resource", "delete")
	createdPerm2, err := manager.CreatePermission(ctx, perm2)
	require.NoError(t, err)

	tests := []struct {
		name      string
		operation func() (interface{}, error)
		checkFunc func(*testing.T, interface{})
	}{
		{
			name: "List all roles",
			operation: func() (interface{}, error) {
				return manager.ListRoles(ctx)
			},
			checkFunc: func(t *testing.T, result interface{}) {
				roles := result.([]*Role)
				assert.GreaterOrEqual(t, len(roles), 2) // At least our test roles plus defaults

				roleNames := make([]string, len(roles))
				for i, role := range roles {
					roleNames[i] = role.Name
				}
				assert.Contains(t, roleNames, "test-role-1")
				assert.Contains(t, roleNames, "test-role-2")
			},
		},
		{
			name: "List all permissions",
			operation: func() (interface{}, error) {
				return manager.ListPermissions(ctx)
			},
			checkFunc: func(t *testing.T, result interface{}) {
				permissions := result.([]*Permission)
				assert.GreaterOrEqual(t, len(permissions), 2) // At least our test permissions plus defaults

				permissionIDs := make([]string, len(permissions))
				for i, perm := range permissions {
					permissionIDs[i] = perm.ID
				}
				assert.Contains(t, permissionIDs, createdPerm1.ID)
				assert.Contains(t, permissionIDs, createdPerm2.ID)
			},
		},
		{
			name: "List user roles",
			operation: func() (interface{}, error) {
				// First assign some roles to a user
				err := manager.AssignRoleToUser(ctx, "test-user", createdRole1.ID)
				require.NoError(t, err)
				err = manager.AssignRoleToUser(ctx, "test-user", createdRole2.ID)
				require.NoError(t, err)

				return manager.GetUserRoles(ctx, "test-user")
			},
			checkFunc: func(t *testing.T, result interface{}) {
				roleIDs := result.([]string)
				assert.Len(t, roleIDs, 2)
				assert.Contains(t, roleIDs, createdRole1.ID)
				assert.Contains(t, roleIDs, createdRole2.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.operation()
			assert.NoError(t, err)
			assert.NotNil(t, result)
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkRBACManager_CheckPermission(b *testing.B) {
	tc := testutil.NewTestContext(&testing.T{})
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()
	pf := testutil.NewPermissionFactory()
	ctx := context.Background()

	// Setup test data
	perm := pf.CreateBasicPermission()
	createdPerm, err := manager.CreatePermission(ctx, perm)
	if err != nil {
		b.Fatal(err)
	}

	role := rf.CreateRoleWithPermissions([]string{createdPerm.ID})
	createdRole, err := manager.CreateRole(ctx, role)
	if err != nil {
		b.Fatal(err)
	}

	err = manager.AssignRoleToUser(ctx, "bench-user", createdRole.ID)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.CheckPermission(ctx, "bench-user", "test-resource", "read")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRBACManager_GetUserRoles(b *testing.B) {
	tc := testutil.NewTestContext(&testing.T{})
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()
	ctx := context.Background()

	// Create and assign multiple roles
	for i := 0; i < 10; i++ {
		role := rf.CreateBasicRole()
		createdRole, err := manager.CreateRole(ctx, role)
		if err != nil {
			b.Fatal(err)
		}

		err = manager.AssignRoleToUser(ctx, "bench-user", createdRole.ID)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.GetUserRoles(ctx, "bench-user")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test caching functionality
func TestRBACManager_Caching(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()
	pf := testutil.NewPermissionFactory()
	ctx := context.Background()

	// Create test data
	perm := pf.CreateBasicPermission()
	createdPerm, err := manager.CreatePermission(ctx, perm)
	require.NoError(t, err)

	role := rf.CreateRoleWithPermissions([]string{createdPerm.ID})
	createdRole, err := manager.CreateRole(ctx, role)
	require.NoError(t, err)

	err = manager.AssignRoleToUser(ctx, "cache-test-user", createdRole.ID)
	require.NoError(t, err)

	// First permission check (should cache result)
	allowed1, err := manager.CheckPermission(ctx, "cache-test-user", "test-resource", "read")
	assert.NoError(t, err)
	assert.True(t, allowed1)

	// Second permission check (should use cache)
	allowed2, err := manager.CheckPermission(ctx, "cache-test-user", "test-resource", "read")
	assert.NoError(t, err)
	assert.True(t, allowed2)

	// Revoke role and check if cache is invalidated
	err = manager.RevokeRoleFromUser(ctx, "cache-test-user", createdRole.ID)
	assert.NoError(t, err)

	// Permission check should now return false (cache should be invalidated)
	allowed3, err := manager.CheckPermission(ctx, "cache-test-user", "test-resource", "read")
	assert.NoError(t, err)
	assert.False(t, allowed3)
}

// Test concurrent access
func TestRBACManager_ConcurrentAccess(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()
	rf := testutil.NewRoleFactory()
	pf := testutil.NewPermissionFactory()
	ctx := context.Background()

	// Setup test data
	perm := pf.CreateBasicPermission()
	createdPerm, err := manager.CreatePermission(ctx, perm)
	require.NoError(t, err)

	role := rf.CreateRoleWithPermissions([]string{createdPerm.ID})
	createdRole, err := manager.CreateRole(ctx, role)
	require.NoError(t, err)

	// Run concurrent permission checks
	const numGoroutines = 10
	const numChecks = 100

	errChan := make(chan error, numGoroutines*numChecks)

	for i := 0; i < numGoroutines; i++ {
		go func(userID string) {
			// Assign role
			err := manager.AssignRoleToUser(ctx, userID, createdRole.ID)
			if err != nil {
				errChan <- err
				return
			}

			// Perform multiple permission checks
			for j := 0; j < numChecks; j++ {
				_, err := manager.CheckPermission(ctx, userID, "test-resource", "read")
				errChan <- err
			}
		}(fmt.Sprintf("concurrent-user-%d", i))
	}

	// Collect results
	for i := 0; i < numGoroutines*numChecks; i++ {
		err := <-errChan
		assert.NoError(t, err, "Concurrent permission check failed")
	}
}

func TestRBACManager_PolicyEngineIntegration(t *testing.T) {
	tc := testutil.NewTestContext(t)
	defer tc.Cleanup()

	manager := tc.SetupRBACManager()

	// Test policy-based permission evaluation
	tests := []struct {
		name          string
		policyRules   []string
		userContext   map[string]interface{}
		resource      string
		action        string
		expectAllowed bool
	}{
		{
			name: "Time-based access policy",
			policyRules: []string{
				"allow read on sensitive-data if time between 09:00 and 17:00",
			},
			userContext: map[string]interface{}{
				"time": "14:30",
				"user": "test-user",
			},
			resource:      "sensitive-data",
			action:        "read",
			expectAllowed: true,
		},
		{
			name: "IP-based access policy",
			policyRules: []string{
				"allow * on internal-resource if ip in 192.168.1.0/24",
			},
			userContext: map[string]interface{}{
				"ip_address": "192.168.1.100",
				"user":       "test-user",
			},
			resource:      "internal-resource",
			action:        "write",
			expectAllowed: true,
		},
		// More complex policy tests can be added here
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test demonstrates how RBAC could integrate with a policy engine
			// Implementation would depend on the specific policy engine used

			// For now, we'll test the basic structure
			ctx := context.Background()

			// In a real implementation, you would:
			// 1. Parse and store policy rules
			// 2. Evaluate policies based on user context
			// 3. Return policy decision

			// Basic check that the manager can handle complex scenarios
			allowed, err := manager.CheckPermissionWithContext(ctx, tt.userContext, tt.resource, tt.action)

			// This method might not exist yet, but demonstrates the interface
			if err != nil && err.Error() == "method not implemented" {
				t.Skip("CheckPermissionWithContext not implemented yet")
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectAllowed, allowed)
		})
	}
}
