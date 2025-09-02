package auth

import (
	"context"
	"crypto/rsa"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

// JWTManagerInterface defines the interface for JWT management.

type JWTManagerInterface interface {
	// Core token operations
	GenerateToken(user *providers.UserInfo, customClaims map[string]interface{}) (string, error)
	GenerateTokenWithTTL(user *providers.UserInfo, customClaims map[string]interface{}, ttl time.Duration) (string, error)
	GenerateTokenPair(user *providers.UserInfo, customClaims map[string]interface{}) (string, string, error)

	// Validation and claims
	ValidateToken(ctx context.Context, tokenString string) (*NephoranJWTClaims, error)
	ExtractClaims(tokenString string) (jwt.MapClaims, error)

	// Token lifecycle
	RefreshToken(refreshTokenString string) (string, string, error)
	BlacklistToken(tokenString string) error
	IsTokenBlacklisted(ctx context.Context, tokenString string) (bool, error)
	CleanupBlacklist() error

	// Key management
	SetSigningKey(privateKey *rsa.PrivateKey, keyID string) error
	GetPublicKey(keyID string) (*rsa.PublicKey, error)
	GetJWKS() (map[string]interface{}, error)
	RotateKeys() error
	GetKeyID() string

	// Configuration getters
	GetIssuer() string
	GetDefaultTTL() time.Duration
	GetRefreshTTL() time.Duration
	GetRequireSecureCookies() bool

	// Cleanup
	Close()
}

// RBACManagerInterface defines the interface for role-based access control.

type RBACManagerInterface interface {
	// Permission checking
	CheckPermission(ctx context.Context, userID, permission string) bool
	CheckPermissionLegacy(ctx context.Context, userID, resource, action string) (bool, error)
	CheckAccess(ctx context.Context, request *AccessRequest) *AccessDecision

	// Role management  
	AssignRole(ctx context.Context, userID, role string) error
	AssignRoleToUser(ctx context.Context, userID, roleID string) error
	RevokeRole(ctx context.Context, userID, role string) error
	RevokeRoleFromUser(ctx context.Context, userID, roleID string) error

	// User queries
	GetUserRoles(ctx context.Context, userID string) []string
	GetUserPermissions(ctx context.Context, userID string) []string
	GetRolePermissions(ctx context.Context, role string) ([]string, error)

	// RBAC data management
	CreateRole(ctx context.Context, role interface{}) (interface{}, error)
	UpdateRole(ctx context.Context, role interface{}) error  
	DeleteRole(ctx context.Context, roleID string) error
	ListRoles(ctx context.Context) interface{}
	GetRole(ctx context.Context, roleID string) (interface{}, error)

	CreatePermission(ctx context.Context, perm interface{}) (interface{}, error)
	ListPermissions(ctx context.Context) interface{}

	// Role assignment from claims
	AssignRolesFromClaims(ctx context.Context, userInfo *providers.UserInfo) error
}

// SessionManagerInterface defines the interface for session management.

type SessionManagerInterface interface {
	CreateSession(ctx context.Context, userInfo *providers.UserInfo, metadata ...map[string]interface{}) (interface{}, error)

	GetSession(ctx context.Context, sessionID string) (interface{}, error)

	UpdateSession(ctx context.Context, sessionID string, updates map[string]interface{}) error

	DeleteSession(ctx context.Context, sessionID string) error

	ListUserSessions(ctx context.Context, userID string) (interface{}, error)

	SetSessionCookie(w http.ResponseWriter, sessionID string)

	GetSessionFromRequest(r *http.Request) (interface{}, error)

	Close()
}

// HandlersInterface defines the interface for authentication handlers.

type HandlersInterface interface {
	RegisterRoutes(router interface{})

	GetProvidersHandler(w http.ResponseWriter, r *http.Request)

	InitiateLoginHandler(w http.ResponseWriter, r *http.Request)

	CallbackHandler(w http.ResponseWriter, r *http.Request)

	LogoutHandler(w http.ResponseWriter, r *http.Request)

	GetUserInfoHandler(w http.ResponseWriter, r *http.Request)
}

// UserSession represents an active user session.

type UserSession struct {
	ID string `json:"id"`

	UserID string `json:"user_id"`

	UserInfo *providers.UserInfo `json:"user_info"`

	Provider string `json:"provider"`

	AccessToken string `json:"access_token"`

	RefreshToken string `json:"refresh_token"`

	IDToken string `json:"id_token,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	LastActivity time.Time `json:"last_activity"`

	ExpiresAt time.Time `json:"expires_at"`

	IPAddress string `json:"ip_address"`

	UserAgent string `json:"user_agent"`

	Roles []string `json:"roles"`

	Permissions []string `json:"permissions"`

	Attributes map[string]interface{} `json:"attributes,omitempty"`

	// SSO state.

	SSOEnabled bool `json:"sso_enabled"`

	LinkedSessions map[string]string `json:"linked_sessions,omitempty"` // provider -> session_id

	// Security.

	CSRFToken string `json:"csrf_token"`

	SecureContext bool `json:"secure_context"`
}

// ManagerInterface defines the interface for the main authentication manager.

type ManagerInterface interface {
	GetMiddleware() interface{}

	GetLDAPMiddleware() interface{}

	GetOAuth2Manager() interface{}

	GetSessionManager() SessionManagerInterface

	GetJWTManager() JWTManagerInterface

	GetRBACManager() RBACManagerInterface

	ListProviders() map[string]interface{}

	RefreshTokens(ctx context.Context, refreshToken string) (string, string, error)

	ValidateSession(ctx context.Context, sessionID string) (*UserSession, error)

	Shutdown(ctx context.Context) error
}
