package auth

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nephio-project/nephoran-intent-operator/pkg/auth/providers"
)

// RBACManager manages role-based access control.

type RBACManager struct {
	roles map[string]*Role

	permissions map[string]*Permission

	policies map[string]*Policy

	userRoles map[string][]string // userID -> roles

	// Caching.

	cache *RBACCache

	cacheTTL time.Duration

	logger *slog.Logger

	mutex sync.RWMutex
}

// Role represents a role with associated permissions.

type Role struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Permissions []string `json:"permissions"`

	Metadata map[string]string `json:"metadata,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`

	// Hierarchical roles.

	ParentRoles []string `json:"parent_roles,omitempty"`

	ChildRoles []string `json:"child_roles,omitempty"`
}

// Permission represents a specific permission.

type Permission struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Resource string `json:"resource"`

	Action string `json:"action"`

	Scope string `json:"scope,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`
}

// Policy represents an access control policy.

type Policy struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Rules []PolicyRule `json:"rules"`

	Effect PolicyEffect `json:"effect"`

	Conditions []PolicyCondition `json:"conditions,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`

	Priority int `json:"priority"`
}

// PolicyRule represents a rule within a policy.

type PolicyRule struct {
	Resources []string `json:"resources"`

	Actions []string `json:"actions"`

	Permissions []string `json:"permissions,omitempty"`
}

// PolicyCondition represents conditions for policy evaluation.

type PolicyCondition struct {
	Type ConditionType `json:"type"`

	Attribute string `json:"attribute"`

	Operator string `json:"operator"`

	Values []string `json:"values"`
}

// PolicyEffect determines allow or deny.

type PolicyEffect string

const (

	// PolicyEffectAllow holds policyeffectallow value.

	PolicyEffectAllow PolicyEffect = "allow"

	// PolicyEffectDeny holds policyeffectdeny value.

	PolicyEffectDeny PolicyEffect = "deny"
)

// ConditionType represents types of policy conditions.

type ConditionType string

const (

	// ConditionTypeAttribute holds conditiontypeattribute value.

	ConditionTypeAttribute ConditionType = "attribute"

	// ConditionTypeTime holds conditiontypetime value.

	ConditionTypeTime ConditionType = "time"

	// ConditionTypeIP holds conditiontypeip value.

	ConditionTypeIP ConditionType = "ip"

	// ConditionTypeGroup holds conditiontypegroup value.

	ConditionTypeGroup ConditionType = "group"
)

// AccessRequest represents an access control request.

type AccessRequest struct {
	UserID string `json:"user_id"`

	Resource string `json:"resource"`

	Action string `json:"action"`

	Context map[string]interface{} `json:"context,omitempty"`

	Attributes map[string]interface{} `json:"attributes,omitempty"`

	IPAddress string `json:"ip_address,omitempty"`

	UserAgent string `json:"user_agent,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	RequestID string `json:"request_id,omitempty"`
}

// AccessDecision represents the result of an access control evaluation.

type AccessDecision struct {
	Allowed bool `json:"allowed"`

	Reason string `json:"reason"`

	AppliedPolicies []string `json:"applied_policies"`

	RequiredRoles []string `json:"required_roles,omitempty"`

	MissingPermissions []string `json:"missing_permissions,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`

	EvaluatedAt time.Time `json:"evaluated_at"`

	TTL time.Duration `json:"ttl,omitempty"`
}

// RBACCache represents cached RBAC data.

type RBACCache struct {
	userPermissions map[string][]string

	roleHierarchy map[string][]string

	lastUpdated time.Time

	mutex sync.RWMutex
}

// RBACManagerConfig represents RBAC manager configuration.

type RBACManagerConfig struct {
	CacheTTL time.Duration `json:"cache_ttl"`

	EnableHierarchy bool `json:"enable_hierarchy"`

	DefaultDenyAll bool `json:"default_deny_all"`

	PolicyEvaluation string `json:"policy_evaluation"` // "first-applicable", "deny-overrides", "permit-overrides"

	MaxPolicyDepth int `json:"max_policy_depth"`

	EnableAuditLogging bool `json:"enable_audit_logging"`
}

// Predefined roles and permissions for Nephoran.

var (

	// System Administrator - full access.

	SystemAdminRole = &Role{

		ID: "system-admin",

		Name: "System Administrator",

		Description: "Full system access with all privileges",

		Permissions: []string{

			"system:*",

			"intent:*",

			"e2:*",

			"auth:*",

			"config:*",

			"monitoring:*",

			"user:*",

			"rbac:*",
		},

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}

	// Intent Operator - network intent management.

	IntentOperatorRole = &Role{

		ID: "intent-operator",

		Name: "Intent Operator",

		Description: "Manage network intents and deployments",

		Permissions: []string{

			"intent:create",

			"intent:read",

			"intent:update",

			"intent:delete",

			"intent:deploy",

			"e2:read",

			"config:read",

			"monitoring:read",
		},

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}

	// E2 Interface Manager - E2 interface operations.

	E2ManagerRole = &Role{

		ID: "e2-manager",

		Name: "E2 Interface Manager",

		Description: "Manage E2 interface connections and service models",

		Permissions: []string{

			"e2:create",

			"e2:read",

			"e2:update",

			"e2:delete",

			"e2:subscribe",

			"e2:control",

			"monitoring:read",
		},

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}

	// Read-Only User - view access only.

	ReadOnlyRole = &Role{

		ID: "read-only",

		Name: "Read Only User",

		Description: "Read-only access to system resources",

		Permissions: []string{

			"intent:read",

			"e2:read",

			"config:read",

			"monitoring:read",
		},

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}

	// Nephoran Permissions.

	NephoranPermissions = []*Permission{

		// System permissions.

		{ID: "system:admin", Name: "System Administration", Resource: "system", Action: "*"},

		{ID: "system:config", Name: "System Configuration", Resource: "system", Action: "config"},

		// Intent management permissions.

		{ID: "intent:create", Name: "Create Intent", Resource: "intent", Action: "create"},

		{ID: "intent:read", Name: "Read Intent", Resource: "intent", Action: "read"},

		{ID: "intent:update", Name: "Update Intent", Resource: "intent", Action: "update"},

		{ID: "intent:delete", Name: "Delete Intent", Resource: "intent", Action: "delete"},

		{ID: "intent:deploy", Name: "Deploy Intent", Resource: "intent", Action: "deploy"},

		// E2 interface permissions.

		{ID: "e2:create", Name: "Create E2 Connection", Resource: "e2", Action: "create"},

		{ID: "e2:read", Name: "Read E2 Data", Resource: "e2", Action: "read"},

		{ID: "e2:update", Name: "Update E2 Configuration", Resource: "e2", Action: "update"},

		{ID: "e2:delete", Name: "Delete E2 Connection", Resource: "e2", Action: "delete"},

		{ID: "e2:subscribe", Name: "Subscribe to E2 Events", Resource: "e2", Action: "subscribe"},

		{ID: "e2:control", Name: "Control E2 Functions", Resource: "e2", Action: "control"},

		// Configuration permissions.

		{ID: "config:read", Name: "Read Configuration", Resource: "config", Action: "read"},

		{ID: "config:write", Name: "Write Configuration", Resource: "config", Action: "write"},

		// Monitoring permissions.

		{ID: "monitoring:read", Name: "Read Monitoring Data", Resource: "monitoring", Action: "read"},

		{ID: "monitoring:write", Name: "Write Monitoring Config", Resource: "monitoring", Action: "write"},

		// Authentication permissions.

		{ID: "auth:read", Name: "Read Auth Configuration", Resource: "auth", Action: "read"},

		{ID: "auth:write", Name: "Write Auth Configuration", Resource: "auth", Action: "write"},

		{ID: "auth:manage", Name: "Manage Authentication", Resource: "auth", Action: "manage"},

		// User management permissions.

		{ID: "user:read", Name: "Read User Data", Resource: "user", Action: "read"},

		{ID: "user:write", Name: "Write User Data", Resource: "user", Action: "write"},

		{ID: "user:manage", Name: "Manage Users", Resource: "user", Action: "manage"},

		// RBAC permissions.

		{ID: "rbac:read", Name: "Read RBAC Configuration", Resource: "rbac", Action: "read"},

		{ID: "rbac:write", Name: "Write RBAC Configuration", Resource: "rbac", Action: "write"},

		{ID: "rbac:manage", Name: "Manage RBAC", Resource: "rbac", Action: "manage"},
	}
)

// NewRBACManager creates a new RBAC manager.

func NewRBACManager(config *RBACManagerConfig, logger *slog.Logger) *RBACManager {

	if config == nil {

		config = &RBACManagerConfig{

			CacheTTL: 15 * time.Minute,

			EnableHierarchy: true,

			DefaultDenyAll: true,

			PolicyEvaluation: "deny-overrides",

			MaxPolicyDepth: 10,

			EnableAuditLogging: true,
		}

	}

	manager := &RBACManager{

		roles: make(map[string]*Role),

		permissions: make(map[string]*Permission),

		policies: make(map[string]*Policy),

		userRoles: make(map[string][]string),

		cache: &RBACCache{

			userPermissions: make(map[string][]string),

			roleHierarchy: make(map[string][]string),
		},

		cacheTTL: config.CacheTTL,

		logger: logger,
	}

	// Initialize with default roles and permissions.

	manager.initializeDefaults()

	return manager

}

// InitializeDefaults sets up default roles and permissions.

func (r *RBACManager) initializeDefaults() {

	// Add default permissions.

	for _, perm := range NephoranPermissions {

		perm.CreatedAt = time.Now()

		perm.UpdatedAt = time.Now()

		perm.Description = fmt.Sprintf("Permission to %s on %s", perm.Action, perm.Resource)

		r.permissions[perm.ID] = perm

	}

	// Add default roles.

	defaultRoles := []*Role{SystemAdminRole, IntentOperatorRole, E2ManagerRole, ReadOnlyRole}

	for _, role := range defaultRoles {

		r.roles[role.ID] = role

	}

}

// GrantRoleToUser assigns a role to a user.

func (r *RBACManager) GrantRoleToUser(ctx context.Context, userID, roleID string) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	// Verify role exists.

	if _, exists := r.roles[roleID]; !exists {

		return fmt.Errorf("role %s does not exist", roleID)

	}

	// Add role to user.

	userRoles := r.userRoles[userID]

	for _, existingRole := range userRoles {

		if existingRole == roleID {

			return nil // Already has role

		}

	}

	r.userRoles[userID] = append(userRoles, roleID)

	r.invalidateUserCache(userID)

	r.logger.Info("Role granted to user",

		"user_id", userID,

		"role_id", roleID)

	return nil

}

// RevokeRoleFromUser removes a role from a user.

func (r *RBACManager) RevokeRoleFromUser(ctx context.Context, userID, roleID string) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	userRoles := r.userRoles[userID]

	for i, role := range userRoles {

		if role == roleID {

			// Remove role.

			r.userRoles[userID] = append(userRoles[:i], userRoles[i+1:]...)

			r.invalidateUserCache(userID)

			r.logger.Info("Role revoked from user",

				"user_id", userID,

				"role_id", roleID)

			return nil

		}

	}

	return fmt.Errorf("user %s does not have role %s", userID, roleID)

}

// GetUserRoles returns all roles for a user.

func (r *RBACManager) GetUserRoles(ctx context.Context, userID string) []string {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	roles := r.userRoles[userID]

	result := make([]string, len(roles))

	copy(result, roles)

	return result

}

// GetUserPermissions returns all permissions for a user (with role hierarchy).

func (r *RBACManager) GetUserPermissions(ctx context.Context, userID string) []string {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	// Check cache first.

	r.cache.mutex.RLock()

	if cachedPerms, exists := r.cache.userPermissions[userID]; exists &&

		time.Since(r.cache.lastUpdated) < r.cacheTTL {

		r.cache.mutex.RUnlock()

		return cachedPerms

	}

	r.cache.mutex.RUnlock()

	// Calculate permissions.

	permissions := r.calculateUserPermissions(userID)

	// Cache result.

	r.cache.mutex.Lock()

	r.cache.userPermissions[userID] = permissions

	r.cache.lastUpdated = time.Now()

	r.cache.mutex.Unlock()

	return permissions

}

// CheckPermission checks if a user has a specific permission.

func (r *RBACManager) CheckPermission(ctx context.Context, userID, permission string) bool {

	userPermissions := r.GetUserPermissions(ctx, userID)

	for _, perm := range userPermissions {

		if r.matchesPermission(perm, permission) {

			return true

		}

	}

	return false

}

// CheckAccess evaluates an access request against RBAC policies.

func (r *RBACManager) CheckAccess(ctx context.Context, request *AccessRequest) *AccessDecision {

	startTime := time.Now()

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	decision := &AccessDecision{

		EvaluatedAt: startTime,

		Metadata: make(map[string]interface{}),
	}

	// Get user permissions.

	userPermissions := r.GetUserPermissions(ctx, request.UserID)

	userRoles := r.GetUserRoles(ctx, request.UserID)

	// Check if user has required permission.

	requiredPermission := fmt.Sprintf("%s:%s", request.Resource, request.Action)

	hasPermission := false

	for _, perm := range userPermissions {

		if r.matchesPermission(perm, requiredPermission) {

			hasPermission = true

			break

		}

	}

	if !hasPermission {

		decision.Allowed = false

		decision.Reason = fmt.Sprintf("User lacks required permission: %s", requiredPermission)

		decision.MissingPermissions = []string{requiredPermission}

		return decision

	}

	// Evaluate policies.

	decision = r.evaluatePolicies(ctx, request, decision)

	// Add metadata.

	decision.Metadata["evaluation_time_ms"] = time.Since(startTime).Milliseconds()

	decision.Metadata["user_roles"] = userRoles

	decision.Metadata["checked_permission"] = requiredPermission

	// Audit logging.

	if decision.Allowed {

		r.logger.Info("Access granted",

			"user_id", request.UserID,

			"resource", request.Resource,

			"action", request.Action,

			"reason", decision.Reason)

	} else {

		r.logger.Warn("Access denied",

			"user_id", request.UserID,

			"resource", request.Resource,

			"action", request.Action,

			"reason", decision.Reason)

	}

	return decision

}

// CreateRole creates a new role.

func (r *RBACManager) CreateRole(ctx context.Context, role *Role) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	if role.ID == "" {

		return fmt.Errorf("role ID cannot be empty")

	}

	if _, exists := r.roles[role.ID]; exists {

		return fmt.Errorf("role %s already exists", role.ID)

	}

	// Validate permissions exist.

	for _, permID := range role.Permissions {

		if _, exists := r.permissions[permID]; !exists {

			return fmt.Errorf("permission %s does not exist", permID)

		}

	}

	now := time.Now()

	role.CreatedAt = now

	role.UpdatedAt = now

	r.roles[role.ID] = role

	r.invalidateAllCache()

	r.logger.Info("Role created",

		"role_id", role.ID,

		"role_name", role.Name)

	return nil

}

// UpdateRole updates an existing role.

func (r *RBACManager) UpdateRole(ctx context.Context, role *Role) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	if _, exists := r.roles[role.ID]; !exists {

		return fmt.Errorf("role %s does not exist", role.ID)

	}

	// Validate permissions exist.

	for _, permID := range role.Permissions {

		if _, exists := r.permissions[permID]; !exists {

			return fmt.Errorf("permission %s does not exist", permID)

		}

	}

	role.UpdatedAt = time.Now()

	r.roles[role.ID] = role

	r.invalidateAllCache()

	r.logger.Info("Role updated",

		"role_id", role.ID,

		"role_name", role.Name)

	return nil

}

// DeleteRole deletes a role.

func (r *RBACManager) DeleteRole(ctx context.Context, roleID string) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	if _, exists := r.roles[roleID]; !exists {

		return fmt.Errorf("role %s does not exist", roleID)

	}

	// Remove role from all users.

	for userID, userRoles := range r.userRoles {

		for i, role := range userRoles {

			if role == roleID {

				r.userRoles[userID] = append(userRoles[:i], userRoles[i+1:]...)

				break

			}

		}

	}

	delete(r.roles, roleID)

	r.invalidateAllCache()

	r.logger.Info("Role deleted", "role_id", roleID)

	return nil

}

// ListRoles returns all roles.

func (r *RBACManager) ListRoles(ctx context.Context) []*Role {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	roles := make([]*Role, 0, len(r.roles))

	for _, role := range r.roles {

		roles = append(roles, role)

	}

	// Sort by name.

	sort.Slice(roles, func(i, j int) bool {

		return roles[i].Name < roles[j].Name

	})

	return roles

}

// GetRole returns a specific role.

func (r *RBACManager) GetRole(ctx context.Context, roleID string) (*Role, error) {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	role, exists := r.roles[roleID]

	if !exists {

		return nil, fmt.Errorf("role %s does not exist", roleID)

	}

	return role, nil

}

// CreatePermission creates a new permission.

func (r *RBACManager) CreatePermission(ctx context.Context, perm *Permission) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	if perm.ID == "" {

		return fmt.Errorf("permission ID cannot be empty")

	}

	if _, exists := r.permissions[perm.ID]; exists {

		return fmt.Errorf("permission %s already exists", perm.ID)

	}

	now := time.Now()

	perm.CreatedAt = now

	perm.UpdatedAt = now

	r.permissions[perm.ID] = perm

	r.invalidateAllCache()

	r.logger.Info("Permission created",

		"permission_id", perm.ID,

		"permission_name", perm.Name)

	return nil

}

// ListPermissions returns all permissions.

func (r *RBACManager) ListPermissions(ctx context.Context) []*Permission {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	permissions := make([]*Permission, 0, len(r.permissions))

	for _, perm := range r.permissions {

		permissions = append(permissions, perm)

	}

	// Sort by resource and action.

	sort.Slice(permissions, func(i, j int) bool {

		if permissions[i].Resource == permissions[j].Resource {

			return permissions[i].Action < permissions[j].Action

		}

		return permissions[i].Resource < permissions[j].Resource

	})

	return permissions

}

// AssignRolesFromClaims assigns roles based on JWT claims and provider groups.

func (r *RBACManager) AssignRolesFromClaims(ctx context.Context, userInfo *providers.UserInfo) error {

	userID := userInfo.Subject

	// Clear existing roles for fresh assignment.

	r.mutex.Lock()

	r.userRoles[userID] = []string{}

	r.mutex.Unlock()

	// Map provider groups/roles to Nephoran roles.

	roleMappings := r.getRoleMappings(userInfo)

	for _, roleID := range roleMappings {

		if err := r.GrantRoleToUser(ctx, userID, roleID); err != nil {

			r.logger.Warn("Failed to grant role from claims",

				"user_id", userID,

				"role_id", roleID,

				"error", err)

		}

	}

	r.logger.Info("Roles assigned from claims",

		"user_id", userID,

		"provider", userInfo.Provider,

		"roles", roleMappings)

	return nil

}

// Private helper methods.

func (r *RBACManager) calculateUserPermissions(userID string) []string {

	var allPermissions []string

	permissionSet := make(map[string]bool)

	// Get direct role permissions.

	userRoles := r.userRoles[userID]

	for _, roleID := range userRoles {

		if role, exists := r.roles[roleID]; exists {

			for _, permID := range role.Permissions {

				if !permissionSet[permID] {

					allPermissions = append(allPermissions, permID)

					permissionSet[permID] = true

				}

			}

		}

	}

	sort.Strings(allPermissions)

	return allPermissions

}

func (r *RBACManager) matchesPermission(granted, required string) bool {

	// Handle wildcard permissions.

	if granted == "*" || granted == required {

		return true

	}

	// Handle resource-level wildcards (e.g., "intent:*" matches "intent:read").

	if strings.HasSuffix(granted, ":*") {

		grantedResource := strings.TrimSuffix(granted, ":*")

		requiredParts := strings.SplitN(required, ":", 2)

		if len(requiredParts) == 2 && requiredParts[0] == grantedResource {

			return true

		}

	}

	return false

}

func (r *RBACManager) evaluatePolicies(ctx context.Context, request *AccessRequest, decision *AccessDecision) *AccessDecision {

	// For now, return allow if user has permission.

	// In future, implement complex policy evaluation.

	decision.Allowed = true

	decision.Reason = "Permission granted by RBAC"

	return decision

}

func (r *RBACManager) getRoleMappings(userInfo *providers.UserInfo) []string {

	var roles []string

	// Provider-specific role mappings.

	switch userInfo.Provider {

	case "github":

		roles = r.mapGitHubRoles(userInfo)

	case "google":

		roles = r.mapGoogleRoles(userInfo)

	case "azuread":

		roles = r.mapAzureADRoles(userInfo)

	}

	// Default to read-only if no specific mapping.

	if len(roles) == 0 {

		roles = append(roles, "read-only")

	}

	return roles

}

func (r *RBACManager) mapGitHubRoles(userInfo *providers.UserInfo) []string {

	var roles []string

	// Check if user is in admin organizations.

	for _, org := range userInfo.Organizations {

		if strings.Contains(strings.ToLower(org.Name), "admin") {

			roles = append(roles, "system-admin")

			break

		}

	}

	// Check for team-based roles.

	for _, group := range userInfo.Groups {

		if strings.Contains(group, "/admin") || strings.Contains(group, "/owners") {

			roles = append(roles, "intent-operator")

		} else if strings.Contains(group, "/developers") {

			roles = append(roles, "e2-manager")

		}

	}

	return roles

}

func (r *RBACManager) mapGoogleRoles(userInfo *providers.UserInfo) []string {

	var roles []string

	// Check hosted domain for admin access.

	if hostedDomain, exists := userInfo.Attributes["hosted_domain"]; exists {

		if hostedDomain == "nephoran.io" || hostedDomain == "admin.nephoran.io" {

			roles = append(roles, "intent-operator")

		}

	}

	return roles

}

func (r *RBACManager) mapAzureADRoles(userInfo *providers.UserInfo) []string {

	var roles []string

	// Map Azure AD roles to Nephoran roles.

	for _, role := range userInfo.Roles {

		switch strings.ToLower(role) {

		case "global administrator", "user administrator":

			roles = append(roles, "system-admin")

		case "application administrator":

			roles = append(roles, "intent-operator")

		case "directory readers":

			roles = append(roles, "read-only")

		}

	}

	// Check groups.

	for _, group := range userInfo.Groups {

		if strings.Contains(strings.ToLower(group), "admin") {

			roles = append(roles, "intent-operator")

		} else if strings.Contains(strings.ToLower(group), "operator") {

			roles = append(roles, "e2-manager")

		}

	}

	return roles

}

func (r *RBACManager) invalidateUserCache(userID string) {

	r.cache.mutex.Lock()

	defer r.cache.mutex.Unlock()

	delete(r.cache.userPermissions, userID)

}

func (r *RBACManager) invalidateAllCache() {

	r.cache.mutex.Lock()

	defer r.cache.mutex.Unlock()

	r.cache.userPermissions = make(map[string][]string)

	r.cache.roleHierarchy = make(map[string][]string)

	r.cache.lastUpdated = time.Time{}

}

// GetRBACStatus returns current RBAC status and statistics.

func (r *RBACManager) GetRBACStatus(ctx context.Context) map[string]interface{} {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	return map[string]interface{}{

		"roles_count": len(r.roles),

		"permissions_count": len(r.permissions),

		"users_count": len(r.userRoles),

		"policies_count": len(r.policies),

		"cache_ttl": r.cacheTTL,

		"cache_entries": len(r.cache.userPermissions),

		"last_cache_update": r.cache.lastUpdated,
	}

}
