package testutil

import (
	"context"
	"net/http"
	"strings"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

// UserContext represents user context for middleware
type UserContext struct {
	UserID      string                 `json:"user_id"`
	UserInfo    *providers.UserInfo    `json:"user_info"`
	Roles       []string               `json:"roles"`
	Permissions []string               `json:"permissions"`
	Attributes  map[string]interface{} `json:"attributes"`
}

// AuthMiddlewareConfig represents configuration for auth middleware
type AuthMiddlewareConfig struct {
	JWTManager     *JWTManagerMock
	SessionManager *SessionManagerMock
	RequireAuth    bool
	HeaderName     string
	CookieName     string
	ContextKey     string
}

// AuthMiddleware provides mock authentication middleware
type AuthMiddleware struct {
	config *AuthMiddlewareConfig
}

func NewAuthMiddleware(config *AuthMiddlewareConfig) *AuthMiddleware {
	return &AuthMiddleware{config: config}
}

func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.config.RequireAuth {
			next.ServeHTTP(w, r)
			return
		}

		var userCtx *UserContext

		// Try JWT authentication first
		if m.config.JWTManager != nil && m.config.HeaderName != "" {
			authHeader := r.Header.Get(m.config.HeaderName)
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if _, err := m.config.JWTManager.ValidateToken(token); err == nil {
					userCtx = &UserContext{
						UserID: "test-user",
						UserInfo: &providers.UserInfo{
							Subject: "test-user",
							Email:   "test@example.com",
							Name:    "Test User",
						},
						Roles:       []string{"user"},
						Permissions: []string{"read"},
					}
				}
			}
		}

		// Try session authentication
		if userCtx == nil && m.config.SessionManager != nil && m.config.CookieName != "" {
			if cookie, err := r.Cookie(m.config.CookieName); err == nil {
				if session, err := m.config.SessionManager.ValidateSession(r.Context(), cookie.Value); err == nil {
					userCtx = &UserContext{
						UserID:   session.UserID,
						UserInfo: session.UserInfo,
						Roles:    []string{"user"},
						Permissions: []string{"read"},
					}
				}
			}
		}

		if userCtx == nil && m.config.RequireAuth {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if userCtx != nil {
			ctx := context.WithValue(r.Context(), m.config.ContextKey, userCtx)
			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

// RBACMiddlewareConfig represents configuration for RBAC middleware
type RBACMiddlewareConfig struct {
	RBACManager       *RBACManagerMock
	ResourceExtractor func(*http.Request) string
	ActionExtractor   func(*http.Request) string
	UserIDExtractor   func(*http.Request) string
}

// RBACMiddleware provides mock RBAC middleware
type RBACMiddleware struct {
	config *RBACMiddlewareConfig
}

func NewRBACMiddleware(config *RBACMiddlewareConfig) *RBACMiddleware {
	return &RBACMiddleware{config: config}
}

func (m *RBACMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := m.config.UserIDExtractor(r)
		if userID == "" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		resource := m.config.ResourceExtractor(r)
		action := m.config.ActionExtractor(r)

		if allowed, _ := m.config.RBACManager.CheckPermission(r.Context(), userID, resource, action); !allowed {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}