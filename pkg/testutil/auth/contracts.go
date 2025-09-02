package authtestutil

import "time"

// Forward declarations and aliases to break circular dependencies.

// These are minimal types used only for test data generation.

// Tests should cast these to the actual auth package types when needed.

// TestRole represents a minimal role structure for test data generation.

type TestRole struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Permissions []string `json:"permissions"`

	ParentRoles []string `json:"parent_roles,omitempty"`

	ChildRoles []string `json:"child_roles,omitempty"`

	Metadata map[string]string `json:"metadata,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`
}

// TestPermission represents a minimal permission structure for test data generation.

type TestPermission struct {
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

// TestSession represents a minimal session structure for test data generation.

type TestSession struct {
	ID string `json:"id"`

	UserID string `json:"user_id"`

	CreatedAt time.Time `json:"created_at"`

	ExpiresAt time.Time `json:"expires_at"`

	IPAddress string `json:"ip_address"`

	UserAgent string `json:"user_agent"`

	Metadata map[string]interface{} `json:"metadata"`
}

// Configuration types for testing.

type TestJWTConfig struct {
	Issuer string `json:"issuer"`

	DefaultTTL time.Duration `json:"default_ttl"`

	RefreshTTL time.Duration `json:"refresh_ttl"`

	KeyRotationPeriod time.Duration `json:"key_rotation_period"`

	RequireSecureCookies bool `json:"require_secure_cookies"`

	CookieDomain string `json:"cookie_domain"`

	CookiePath string `json:"cookie_path"`

	Algorithm string `json:"algorithm"`
}

// TestRBACConfig represents a testrbacconfig.

type TestRBACConfig struct {
	CacheTTL time.Duration `json:"cache_ttl"`

	EnableHierarchical bool `json:"enable_hierarchical"`

	DefaultRole string `json:"default_role"`

	SuperAdminRole string `json:"super_admin_role"`
}

// TestSessionConfig represents a testsessionconfig.

type TestSessionConfig struct {
	SessionTTL time.Duration `json:"session_ttl"`

	CleanupPeriod time.Duration `json:"cleanup_period"`

	CookieName string `json:"cookie_name"`

	CookiePath string `json:"cookie_path"`

	CookieDomain string `json:"cookie_domain"`

	SecureCookies bool `json:"secure_cookies"`

	HTTPOnly bool `json:"http_only"`

	SameSite int `json:"same_site"`
}
