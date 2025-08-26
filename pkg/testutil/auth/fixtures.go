package auth

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Local type definitions to avoid import cycles

// UserInfo represents test user information
type UserInfo struct {
	Subject       string                 `json:"sub"`
	Email         string                 `json:"email"`
	EmailVerified bool                   `json:"email_verified"`
	Name          string                 `json:"name"`
	GivenName     string                 `json:"given_name"`
	FamilyName    string                 `json:"family_name"`
	Username      string                 `json:"username"`
	Provider      string                 `json:"provider"`
	ProviderID    string                 `json:"provider_id"`
	Groups        []string               `json:"groups"`
	Roles         []string               `json:"roles"`
	Attributes    map[string]interface{} `json:"attributes"`
	UpdatedAt     int64                  `json:"updated_at"`
}

// TokenResponse represents OAuth2 token response
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	Scope        string    `json:"scope"`
	IDToken      string    `json:"id_token,omitempty"`
	IssuedAt     time.Time `json:"issued_at"`
}

// PKCEChallenge represents PKCE challenge data
type PKCEChallenge struct {
	CodeVerifier  string `json:"code_verifier"`
	CodeChallenge string `json:"code_challenge"`
	Method        string `json:"code_challenge_method"`
}

// ProviderFeature represents provider capabilities
type ProviderFeature string

const (
	FeatureOIDC         ProviderFeature = "oidc"
	FeaturePKCE         ProviderFeature = "pkce"
	FeatureTokenRefresh ProviderFeature = "token_refresh"
	FeatureUserInfo     ProviderFeature = "user_info"
)

// ProviderEndpoints defines OAuth2 provider endpoints
type ProviderEndpoints struct {
	AuthURL      string `json:"authorization_endpoint"`
	TokenURL     string `json:"token_endpoint"`
	UserInfoURL  string `json:"userinfo_endpoint"`
	JWKSURL      string `json:"jwks_uri"`
	RevokeURL    string `json:"revocation_endpoint"`
	DiscoveryURL string `json:"issuer"`
}

// ProviderConfig represents OAuth2 provider configuration
type ProviderConfig struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	ClientID     string            `json:"client_id"`
	ClientSecret string            `json:"client_secret"`
	RedirectURL  string            `json:"redirect_url"`
	Scopes       []string          `json:"scopes"`
	Endpoints    ProviderEndpoints `json:"endpoints"`
	Features     []ProviderFeature `json:"features"`
}

// JWTConfig represents JWT configuration
type JWTConfig struct {
	Issuer               string        `json:"issuer"`
	DefaultTTL           time.Duration `json:"default_ttl"`
	RefreshTTL           time.Duration `json:"refresh_ttl"`
	KeyRotationPeriod    time.Duration `json:"key_rotation_period"`
	RequireSecureCookies bool          `json:"require_secure_cookies"`
	CookieDomain         string        `json:"cookie_domain"`
	CookiePath           string        `json:"cookie_path"`
	Algorithm            string        `json:"algorithm"`
}

// RBACConfig represents RBAC configuration
type RBACConfig struct {
	CacheTTL           time.Duration `json:"cache_ttl"`
	EnableHierarchical bool          `json:"enable_hierarchical"`
	DefaultRole        string        `json:"default_role"`
	SuperAdminRole     string        `json:"super_admin_role"`
}

// SessionConfig represents session configuration
type SessionConfig struct {
	SessionTTL    time.Duration `json:"session_ttl"`
	CleanupPeriod time.Duration `json:"cleanup_period"`
	CookieName    string        `json:"cookie_name"`
	CookiePath    string        `json:"cookie_path"`
	CookieDomain  string        `json:"cookie_domain"`
	SecureCookies bool          `json:"secure_cookies"`
	HTTPOnly      bool          `json:"http_only"`
	SameSite      int           `json:"same_site"`
}

// Role represents an RBAC role
type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	ParentRoles []string  `json:"parent_roles,omitempty"`
	ChildRoles  []string  `json:"child_roles,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Permission represents an RBAC permission
type Permission struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Effect      string    `json:"effect"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Session represents a user session
type Session struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ProviderError represents provider-specific errors
type ProviderError struct {
	Provider    string `json:"provider"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Cause       error  `json:"-"`
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s provider error [%s]: %s", e.Provider, e.Code, e.Description)
}

// NewProviderError creates a new provider error
func NewProviderError(provider, code, description string, cause error) *ProviderError {
	return &ProviderError{
		Provider:    provider,
		Code:        code,
		Description: description,
		Cause:       cause,
	}
}

// GeneratePKCEChallenge generates a PKCE challenge for testing
func GeneratePKCEChallenge() (*PKCEChallenge, error) {
	// Simple test implementation
	verifier := fmt.Sprintf("test-verifier-%d", rand.Int63())
	challenge := fmt.Sprintf("test-challenge-%d", rand.Int63())

	return &PKCEChallenge{
		CodeVerifier:  verifier,
		CodeChallenge: challenge,
		Method:        "S256",
	}, nil
}

// UserFactory provides methods to create test users with various configurations
type UserFactory struct {
	counter int
}

func NewUserFactory() *UserFactory {
	return &UserFactory{}
}

// CreateBasicUser creates a basic user for testing
func (f *UserFactory) CreateBasicUser() *UserInfo {
	f.counter++
	id := fmt.Sprintf("user%d", f.counter)

	return &UserInfo{
		Subject:       fmt.Sprintf("test-%s", id),
		Email:         fmt.Sprintf("%s@example.com", id),
		EmailVerified: true,
		Name:          fmt.Sprintf("Test User %d", f.counter),
		GivenName:     "Test",
		FamilyName:    fmt.Sprintf("User%d", f.counter),
		Username:      id,
		Provider:      "test",
		ProviderID:    fmt.Sprintf("test-%s", id),
		Groups:        []string{"users"},
		Roles:         []string{"viewer"},
		Attributes: map[string]interface{}{
			"department": "engineering",
			"created_at": time.Now().Format(time.RFC3339),
		},
	}
}

// CreateAdminUser creates an admin user for testing
func (f *UserFactory) CreateAdminUser() *UserInfo {
	user := f.CreateBasicUser()
	user.Name = "Admin " + user.Name
	user.Groups = append(user.Groups, "admins")
	user.Roles = []string{"admin"}
	user.Attributes["role_level"] = "admin"
	return user
}

// CreateUserWithGroups creates a user with specific groups
func (f *UserFactory) CreateUserWithGroups(groups []string) *UserInfo {
	user := f.CreateBasicUser()
	user.Groups = groups
	return user
}

// CreateUserWithRoles creates a user with specific roles
func (f *UserFactory) CreateUserWithRoles(roles []string) *UserInfo {
	user := f.CreateBasicUser()
	user.Roles = roles
	return user
}

// CreateUserWithProvider creates a user from a specific provider
func (f *UserFactory) CreateUserWithProvider(provider string) *UserInfo {
	user := f.CreateBasicUser()
	user.Provider = provider
	user.ProviderID = fmt.Sprintf("%s-%s", provider, user.Subject)

	// Provider-specific adjustments
	switch provider {
	case "github":
		user.Username = fmt.Sprintf("gh_%s", user.Username)
		user.Attributes["company"] = "GitHub Inc"
	case "google":
		user.Attributes["domain"] = "gmail.com"
	case "azuread":
		user.Attributes["tenant_id"] = "test-tenant-123"
		user.Attributes["object_id"] = fmt.Sprintf("obj-%s", user.Subject)
	}

	return user
}

// CreateExpiredUser creates a user with expired attributes
func (f *UserFactory) CreateExpiredUser() *UserInfo {
	user := f.CreateBasicUser()
	user.UpdatedAt = time.Now().Add(-30 * 24 * time.Hour).Unix() // 30 days ago
	user.Attributes["account_expires"] = time.Now().Add(-time.Hour).Format(time.RFC3339)
	return user
}

// TokenFactory provides methods to create test tokens
type TokenFactory struct {
	issuer string
}

func NewTokenFactory(issuer string) *TokenFactory {
	if issuer == "" {
		issuer = "test-issuer"
	}
	return &TokenFactory{issuer: issuer}
}

// CreateBasicToken creates a basic JWT token
func (f *TokenFactory) CreateBasicToken(subject string) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss": f.issuer,
		"sub": subject,
		"aud": []string{"test-audience"},
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
		"nbf": now.Unix(),
		"jti": fmt.Sprintf("token-%d", rand.Int63()),
	}
}

// CreateTokenWithScopes creates a token with specific scopes
func (f *TokenFactory) CreateTokenWithScopes(subject string, scopes []string) jwt.MapClaims {
	claims := f.CreateBasicToken(subject)
	claims["scope"] = scopes
	return claims
}

// CreateTokenWithRoles creates a token with roles
func (f *TokenFactory) CreateTokenWithRoles(subject string, roles []string) jwt.MapClaims {
	claims := f.CreateBasicToken(subject)
	claims["roles"] = roles
	return claims
}

// CreateTokenWithGroups creates a token with groups
func (f *TokenFactory) CreateTokenWithGroups(subject string, groups []string) jwt.MapClaims {
	claims := f.CreateBasicToken(subject)
	claims["groups"] = groups
	return claims
}

// CreateExpiredToken creates an expired token
func (f *TokenFactory) CreateExpiredToken(subject string) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss": f.issuer,
		"sub": subject,
		"aud": []string{"test-audience"},
		"exp": now.Add(-time.Hour).Unix(), // Expired 1 hour ago
		"iat": now.Add(-2 * time.Hour).Unix(),
		"nbf": now.Add(-2 * time.Hour).Unix(),
		"jti": fmt.Sprintf("expired-token-%d", rand.Int63()),
	}
}

// CreateTokenNotValidYet creates a token that's not valid yet
func (f *TokenFactory) CreateTokenNotValidYet(subject string) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss": f.issuer,
		"sub": subject,
		"aud": []string{"test-audience"},
		"exp": now.Add(2 * time.Hour).Unix(),
		"iat": now.Unix(),
		"nbf": now.Add(time.Hour).Unix(), // Not valid for 1 hour
		"jti": fmt.Sprintf("future-token-%d", rand.Int63()),
	}
}

// CreateTokenWithCustomClaims creates a token with custom claims
func (f *TokenFactory) CreateTokenWithCustomClaims(subject string, customClaims map[string]interface{}) jwt.MapClaims {
	claims := f.CreateBasicToken(subject)
	for key, value := range customClaims {
		claims[key] = value
	}
	return claims
}

// OAuthResponseFactory creates OAuth2 responses for testing
type OAuthResponseFactory struct{}

func NewOAuthResponseFactory() *OAuthResponseFactory {
	return &OAuthResponseFactory{}
}

// CreateTokenResponse creates a standard OAuth2 token response
func (f *OAuthResponseFactory) CreateTokenResponse() *TokenResponse {
	return &TokenResponse{
		AccessToken:  "test-access-token-" + fmt.Sprintf("%d", rand.Int63()),
		RefreshToken: "test-refresh-token-" + fmt.Sprintf("%d", rand.Int63()),
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		Scope:        "openid email profile",
		IssuedAt:     time.Now(),
	}
}

// CreateTokenResponseWithCustomTTL creates a token response with custom TTL
func (f *OAuthResponseFactory) CreateTokenResponseWithCustomTTL(ttl int64) *TokenResponse {
	resp := f.CreateTokenResponse()
	resp.ExpiresIn = ttl
	return resp
}

// CreateExpiredTokenResponse creates an expired token response
func (f *OAuthResponseFactory) CreateExpiredTokenResponse() *TokenResponse {
	resp := f.CreateTokenResponse()
	resp.ExpiresIn = -3600 // Expired 1 hour ago
	resp.IssuedAt = time.Now().Add(-2 * time.Hour)
	return resp
}

// CreateTokenResponseWithIDToken creates a token response with ID token
func (f *OAuthResponseFactory) CreateTokenResponseWithIDToken(idToken string) *TokenResponse {
	resp := f.CreateTokenResponse()
	resp.IDToken = idToken
	return resp
}

// PKCEFactory creates PKCE challenges for testing
type PKCEFactory struct{}

func NewPKCEFactory() *PKCEFactory {
	return &PKCEFactory{}
}

// CreatePKCEChallenge creates a valid PKCE challenge
func (f *PKCEFactory) CreatePKCEChallenge() *PKCEChallenge {
	challenge, _ := GeneratePKCEChallenge()
	return challenge
}

// CreateInvalidPKCEChallenge creates an invalid PKCE challenge
func (f *PKCEFactory) CreateInvalidPKCEChallenge() *PKCEChallenge {
	return &PKCEChallenge{
		CodeVerifier:  "invalid-verifier",
		CodeChallenge: "invalid-challenge",
		Method:        "S256",
	}
}

// ConfigFactory creates configuration objects for testing
type ConfigFactory struct{}

func NewConfigFactory() *ConfigFactory {
	return &ConfigFactory{}
}

// CreateJWTConfig creates a JWT configuration for testing
func (f *ConfigFactory) CreateJWTConfig() *JWTConfig {
	return &JWTConfig{
		Issuer:               "test-issuer",
		DefaultTTL:           time.Hour,
		RefreshTTL:           24 * time.Hour,
		KeyRotationPeriod:    7 * 24 * time.Hour,
		RequireSecureCookies: false,
		CookieDomain:         "localhost",
		CookiePath:           "/",
		Algorithm:            "RS256",
	}
}

// CreateRBACConfig creates an RBAC configuration for testing
func (f *ConfigFactory) CreateRBACConfig() *RBACConfig {
	return &RBACConfig{
		CacheTTL:           5 * time.Minute,
		EnableHierarchical: true,
		DefaultRole:        "viewer",
		SuperAdminRole:     "superadmin",
	}
}

// CreateSessionConfig creates a session configuration for testing
func (f *ConfigFactory) CreateSessionConfig() *SessionConfig {
	return &SessionConfig{
		SessionTTL:    time.Hour,
		CleanupPeriod: time.Minute,
		CookieName:    "test-session",
		CookiePath:    "/",
		CookieDomain:  "localhost",
		SecureCookies: false,
		HTTPOnly:      true,
		SameSite:      4, // SameSiteStrictMode
	}
}

// CreateProviderConfig creates an OAuth2 provider configuration
func (f *ConfigFactory) CreateProviderConfig(providerName string) *ProviderConfig {
	baseURL := "https://oauth.example.com"
	if providerName == "github" {
		baseURL = "https://github.com"
	} else if providerName == "google" {
		baseURL = "https://accounts.google.com"
	} else if providerName == "azuread" {
		baseURL = "https://login.microsoftonline.com/common"
	}

	return &ProviderConfig{
		Name:         providerName,
		Type:         "oauth2",
		ClientID:     fmt.Sprintf("test-%s-client-id", providerName),
		ClientSecret: fmt.Sprintf("test-%s-client-secret", providerName),
		RedirectURL:  "http://localhost:8080/auth/callback",
		Scopes:       []string{"openid", "email", "profile"},
		Endpoints: ProviderEndpoints{
			AuthURL:      baseURL + "/oauth/authorize",
			TokenURL:     baseURL + "/oauth/token",
			UserInfoURL:  baseURL + "/user",
			JWKSURL:      baseURL + "/.well-known/jwks.json",
			RevokeURL:    baseURL + "/oauth/revoke",
			DiscoveryURL: baseURL + "/.well-known/openid_configuration",
		},
		Features: []ProviderFeature{
			FeatureOIDC,
			FeaturePKCE,
			FeatureTokenRefresh,
			FeatureUserInfo,
		},
	}
}

// RoleFactory creates roles for RBAC testing
type RoleFactory struct {
	counter int
}

func NewRoleFactory() *RoleFactory {
	return &RoleFactory{}
}

// CreateBasicRole creates a basic role
func (f *RoleFactory) CreateBasicRole() *Role {
	f.counter++
	return &Role{
		ID:          fmt.Sprintf("role-%d", f.counter),
		Name:        fmt.Sprintf("test-role-%d", f.counter),
		Description: fmt.Sprintf("Test role %d", f.counter),
		Permissions: []string{"read:basic"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateAdminRole creates an admin role
func (f *RoleFactory) CreateAdminRole() *Role {
	role := f.CreateBasicRole()
	role.Name = "admin"
	role.Description = "Administrator role"
	role.Permissions = []string{
		"read:*", "write:*", "delete:*", "admin:*",
	}
	return role
}

// CreateRoleWithPermissions creates a role with specific permissions
func (f *RoleFactory) CreateRoleWithPermissions(permissions []string) *Role {
	role := f.CreateBasicRole()
	role.Permissions = permissions
	return role
}

// CreateHierarchicalRole creates a role with parent/child relationships
func (f *RoleFactory) CreateHierarchicalRole(parentRoles, childRoles []string) *Role {
	role := f.CreateBasicRole()
	role.ParentRoles = parentRoles
	role.ChildRoles = childRoles
	return role
}

// PermissionFactory creates permissions for RBAC testing
type PermissionFactory struct {
	counter int
}

func NewPermissionFactory() *PermissionFactory {
	return &PermissionFactory{}
}

// CreateBasicPermission creates a basic permission
func (f *PermissionFactory) CreateBasicPermission() *Permission {
	f.counter++
	return &Permission{
		ID:          fmt.Sprintf("perm-%d", f.counter),
		Name:        fmt.Sprintf("test:permission:%d", f.counter),
		Description: fmt.Sprintf("Test permission %d", f.counter),
		Resource:    "test-resource",
		Action:      "read",
		Effect:      "allow",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreateResourcePermissions creates permissions for a specific resource
func (f *PermissionFactory) CreateResourcePermissions(resource string, actions []string) []*Permission {
	var permissions []*Permission
	for _, action := range actions {
		f.counter++
		perm := &Permission{
			ID:          fmt.Sprintf("perm-%d", f.counter),
			Name:        fmt.Sprintf("%s:%s", resource, action),
			Description: fmt.Sprintf("%s permission on %s", action, resource),
			Resource:    resource,
			Action:      action,
			Effect:      "allow",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		permissions = append(permissions, perm)
	}
	return permissions
}

// CreateDenyPermission creates a deny permission
func (f *PermissionFactory) CreateDenyPermission(resource, action string) *Permission {
	f.counter++
	return &Permission{
		ID:          fmt.Sprintf("deny-perm-%d", f.counter),
		Name:        fmt.Sprintf("deny:%s:%s", resource, action),
		Description: fmt.Sprintf("Deny %s permission on %s", action, resource),
		Resource:    resource,
		Action:      action,
		Effect:      "deny",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// SessionFactory creates sessions for testing
type SessionFactory struct {
	counter int
}

func NewSessionFactory() *SessionFactory {
	return &SessionFactory{}
}

// CreateBasicSession creates a basic session
func (f *SessionFactory) CreateBasicSession(userID string) *Session {
	f.counter++
	return &Session{
		ID:        fmt.Sprintf("session-%d", f.counter),
		UserID:    userID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		IPAddress: "127.0.0.1",
		UserAgent: "test-user-agent",
		Metadata: map[string]interface{}{
			"login_method": "oauth2",
			"provider":     "test",
		},
	}
}

// CreateExpiredSession creates an expired session
func (f *SessionFactory) CreateExpiredSession(userID string) *Session {
	session := f.CreateBasicSession(userID)
	session.ExpiresAt = time.Now().Add(-time.Hour)
	return session
}

// CreateSessionWithMetadata creates a session with custom metadata
func (f *SessionFactory) CreateSessionWithMetadata(userID string, metadata map[string]interface{}) *Session {
	session := f.CreateBasicSession(userID)
	session.Metadata = metadata
	return session
}

// ErrorFactory creates various error scenarios for testing
type ErrorFactory struct{}

func NewErrorFactory() *ErrorFactory {
	return &ErrorFactory{}
}

// CreateProviderError creates a provider-specific error
func (f *ErrorFactory) CreateProviderError(provider, code, description string) *ProviderError {
	return NewProviderError(provider, code, description, nil)
}

// CreateAuthError creates authentication errors
func (f *ErrorFactory) CreateAuthError(message string) error {
	return fmt.Errorf("auth error: %s", message)
}

// CreateTokenError creates token-related errors
func (f *ErrorFactory) CreateTokenError(message string) error {
	return fmt.Errorf("token error: %s", message)
}

// CreateValidationError creates validation errors
func (f *ErrorFactory) CreateValidationError(field, message string) error {
	return fmt.Errorf("validation error for %s: %s", field, message)
}

// Helper functions for common test scenarios

// CreateCompleteTestSetup creates a complete test setup with all components
func CreateCompleteTestSetup() (*UserFactory, *TokenFactory, *OAuthResponseFactory, *ConfigFactory, *RoleFactory, *PermissionFactory, *SessionFactory) {
	return NewUserFactory(),
		NewTokenFactory("test-issuer"),
		NewOAuthResponseFactory(),
		NewConfigFactory(),
		NewRoleFactory(),
		NewPermissionFactory(),
		NewSessionFactory()
}

// CreateTestData creates a set of test data for comprehensive testing
func CreateTestData() map[string]interface{} {
	uf := NewUserFactory()
	tf := NewTokenFactory("test-issuer")
	rf := NewRoleFactory()
	pf := NewPermissionFactory()
	sf := NewSessionFactory()

	data := map[string]interface{}{
		"users": map[string]*UserInfo{
			"basic":  uf.CreateBasicUser(),
			"admin":  uf.CreateAdminUser(),
			"github": uf.CreateUserWithProvider("github"),
			"google": uf.CreateUserWithProvider("google"),
			"azure":  uf.CreateUserWithProvider("azuread"),
		},
		"tokens": map[string]jwt.MapClaims{
			"valid":      tf.CreateBasicToken("test-user"),
			"expired":    tf.CreateExpiredToken("test-user"),
			"future":     tf.CreateTokenNotValidYet("test-user"),
			"with_roles": tf.CreateTokenWithRoles("test-user", []string{"admin"}),
		},
		"roles": map[string]*Role{
			"basic": rf.CreateBasicRole(),
			"admin": rf.CreateAdminRole(),
		},
		"permissions": pf.CreateResourcePermissions("test", []string{"read", "write", "delete"}),
		"sessions": map[string]*Session{
			"valid":   sf.CreateBasicSession("test-user"),
			"expired": sf.CreateExpiredSession("test-user"),
		},
	}

	return data
}
