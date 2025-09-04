package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

// AuthManager provides a unified interface for all authentication components
type AuthManager struct {
	JWTManager      *JWTManager
	SessionManager  *SessionManager
	RBACManager     *RBACManager
	OAuth2Manager   *OAuth2Manager
	SecurityManager *SecurityManager
	logger          *slog.Logger
}

// AuthManagerConfig holds configuration for the unified auth manager
type AuthManagerConfig struct {
	JWTConfig      *JWTConfig
	SessionConfig  *SessionConfig
	RBACConfig     *RBACManagerConfig
	OAuth2Config   *OAuth2ManagerConfig
	SecurityConfig *SecurityManagerConfig
}

// SecurityManagerConfig holds configuration for security manager
type SecurityManagerConfig struct {
	CSRFSecret []byte     `json:"csrf_secret"`
	PKCE       PKCEConfig `json:"pkce"`
	CSRF       CSRFConfig `json:"csrf"`
}

// PKCEConfig holds PKCE configuration
type PKCEConfig struct {
	TTL time.Duration `json:"ttl"`
}

// CSRFConfig holds CSRF configuration
type CSRFConfig struct {
	Secret []byte        `json:"secret"`
	TTL    time.Duration `json:"ttl"`
}

// NewAuthManager creates a new unified authentication manager
func NewAuthManager(config *AuthManagerConfig, logger *slog.Logger) (*AuthManager, error) {
	if logger == nil {
		logger = slog.Default()
	}

	// Create JWT manager
	ctx := context.Background() // Use a background context for initialization
	jwtManager, err := NewJWTManager(ctx, config.JWTConfig, nil, nil, logger)
	if err != nil {
		return nil, err
	}

	// Create RBAC manager
	rbacManager := NewRBACManager(config.RBACConfig, logger)

	// Create session manager
	sessionManager := NewSessionManager(config.SessionConfig, jwtManager, rbacManager, logger)

	// Create OAuth2 manager
	oauth2Manager, err := NewOAuth2Manager(ctx, config.OAuth2Config, logger)
	if err != nil {
		return nil, err
	}

	// Create security manager
	securityManager := NewSecurityManager(config.SecurityConfig.CSRFSecret)

	authManager := &AuthManager{
		JWTManager:      jwtManager,
		SessionManager:  sessionManager,
		RBACManager:     rbacManager,
		OAuth2Manager:   oauth2Manager,
		SecurityManager: securityManager,
		logger:          logger,
	}

	logger.Info("Auth manager initialized successfully")
	return authManager, nil
}

// Close performs cleanup of all managed components
func (am *AuthManager) Close() error {
	am.logger.Info("Shutting down auth manager")
	// Add cleanup logic for all components if needed
	return nil
}

// HandleHealthCheck provides health check endpoint
func (am *AuthManager) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// GetLDAPMiddleware returns LDAP middleware if available
func (am *AuthManager) GetLDAPMiddleware() *LDAPAuthMiddleware {
	// For this example, return nil as LDAP middleware is not initialized
	// In a full implementation, this would return a properly configured LDAP middleware
	return nil
}

// GetMiddleware returns the auth middleware
func (am *AuthManager) GetMiddleware() *AuthMiddleware {
	return NewAuthMiddlewareWithComponents(am.SessionManager, am.JWTManager, am.RBACManager, nil)
}

// GetOAuth2Manager returns OAuth2 manager
func (am *AuthManager) GetOAuth2Manager() *OAuth2Manager {
	return am.OAuth2Manager
}

// GetSessionManager returns session manager
func (am *AuthManager) GetSessionManager() *SessionManager {
	return am.SessionManager
}

// ListProviders returns available authentication providers
func (am *AuthManager) ListProviders() map[string]interface{} {
	return map[string]interface{}{
		"ldap": map[string]interface{}{
			"available": false,
			"message":   "LDAP providers not configured in this example",
		},
		"oauth2": map[string]interface{}{
			"available": false,
			"message":   "OAuth2 providers not configured in this example",
		},
	}
}

// RefreshTokens refreshes access and refresh tokens
func (am *AuthManager) RefreshTokens(ctx context.Context, refreshToken string) (string, string, error) {
	if am.JWTManager == nil {
		return "", "", fmt.Errorf("JWT manager not available")
	}

	// Validate refresh token
	claims, err := am.JWTManager.ValidateToken(ctx, refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.TokenType != "refresh" {
		return "", "", fmt.Errorf("invalid token type for refresh")
	}

	// Create new access token
	userInfo := &providers.UserInfo{
		Subject: claims.Subject,
		Email:   claims.Subject, // Use subject as email for now
		Name:    claims.Subject,
		Roles:   claims.Roles,
		Groups:  []string{},
	}
	accessToken, _, err := am.JWTManager.CreateAccessToken(
		context.Background(),
		userInfo,
		claims.SessionID,
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create access token: %w", err)
	}

	// Create new refresh token
	newRefreshToken, _, err := am.JWTManager.CreateRefreshToken(
		context.Background(),
		userInfo,
		claims.SessionID,
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create refresh token: %w", err)
	}

	return accessToken, newRefreshToken, nil
}

// ValidateSession validates a session
func (am *AuthManager) ValidateSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	if am.SessionManager == nil {
		return nil, fmt.Errorf("session manager not available")
	}

	// Return the session info directly from the session manager
	return am.SessionManager.ValidateSession(ctx, sessionID)
}
