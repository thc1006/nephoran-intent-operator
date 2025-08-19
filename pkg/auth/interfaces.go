package auth

import (
	"context"
	"crypto/rsa"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

// JWTManagerInterface defines the interface for JWT management
type JWTManagerInterface interface {
	GenerateToken(claims jwt.MapClaims) (string, error)
	ValidateToken(tokenString string) (*jwt.Token, error)
	RefreshToken(tokenString string) (string, error)
	RevokeToken(tokenString string) error
	SetSigningKey(privateKey *rsa.PrivateKey, keyID string) error
	Close()
}

// RBACManagerInterface defines the interface for role-based access control
type RBACManagerInterface interface {
	CheckPermission(ctx context.Context, userID, resource, action string) (bool, error)
	AssignRole(ctx context.Context, userID, role string) error
	RevokeRole(ctx context.Context, userID, role string) error
	GetUserRoles(ctx context.Context, userID string) ([]string, error)
	GetRolePermissions(ctx context.Context, role string) ([]string, error)
}

// SessionManagerInterface defines the interface for session management
type SessionManagerInterface interface {
	CreateSession(ctx context.Context, userInfo *providers.UserInfo) (*UserSession, error)
	GetSession(ctx context.Context, sessionID string) (*UserSession, error)
	UpdateSession(ctx context.Context, sessionID string, updates map[string]interface{}) error
	DeleteSession(ctx context.Context, sessionID string) error
	ListUserSessions(ctx context.Context, userID string) ([]*UserSession, error)
	SetSessionCookie(w http.ResponseWriter, sessionID string)
	GetSessionFromRequest(r *http.Request) (*UserSession, error)
	Close()
}

// AuthHandlersInterface defines the interface for authentication handlers
type AuthHandlersInterface interface {
	RegisterRoutes(router interface{})
	GetProvidersHandler(w http.ResponseWriter, r *http.Request)
	InitiateLoginHandler(w http.ResponseWriter, r *http.Request)
	CallbackHandler(w http.ResponseWriter, r *http.Request)
	LogoutHandler(w http.ResponseWriter, r *http.Request)
	GetUserInfoHandler(w http.ResponseWriter, r *http.Request)
}