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

// AuthManager manages authentication providers and sessions
type AuthManager struct {
	config         *AuthConfig
	logger         *slog.Logger
	jwtManager     *JWTManager
	sessionManager *SessionManager
	rbacManager    *RBACManager
	
	// Authentication providers
	oauthProviders map[string]providers.OAuthProvider
	ldapProviders  map[string]providers.LDAPProvider
	
	// Middleware and handlers
	middleware     *AuthMiddleware
	ldapMiddleware *LDAPAuthMiddleware
	oauth2Manager  *OAuth2Manager
	handlers       *AuthHandlers
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config *AuthConfig, logger *slog.Logger) (*AuthManager, error) {
	if config == nil {
		return nil, fmt.Errorf("auth config is required")
	}
	
	if logger == nil {
		logger = slog.Default()
	}

	am := &AuthManager{
		config:         config,
		logger:         logger,
		oauthProviders: make(map[string]providers.OAuthProvider),
		ldapProviders:  make(map[string]providers.LDAPProvider),
	}

	// Initialize JWT manager
	if config.Enabled && config.JWTSecretKey != "" {
		// Use a stub JWT manager for now - would need proper config structure
		jwtConfig := &JWTConfig{
			Issuer:     "nephoran-intent-operator",
			SigningKey: config.JWTSecretKey,
			DefaultTTL: config.TokenTTL,
			RefreshTTL: config.RefreshTTL,
		}
		jwtManager, err := NewJWTManager(jwtConfig, nil, nil, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create JWT manager: %w", err)
		}
		am.jwtManager = jwtManager
	}

	// Initialize session manager - using stub for now
	sessionConfig := &SessionConfig{
		SessionTimeout: config.TokenTTL,
		MaxSessions:    100,
		SecureCookies:  true,
	}
	sessionManager := NewSessionManager(sessionConfig, am.jwtManager, am.rbacManager, logger)
	am.sessionManager = sessionManager

	// Initialize RBAC manager
	if config.RBAC.Enabled {
		rbacConfig := &RBACManagerConfig{
			CacheTTL:           5 * time.Minute,
			EnableHierarchy:    true,
			DefaultDenyAll:     true,
			PolicyEvaluation:   "first-applicable",
			MaxPolicyDepth:     10,
			EnableAuditLogging: true,
		}
		rbacManager := NewRBACManager(rbacConfig, logger)
		am.rbacManager = rbacManager
	}

	// Initialize LDAP providers first
	ldapProviders := make(map[string]providers.LDAPProvider)
	if len(config.LDAPProviders) > 0 {
		for name, ldapProviderConfig := range config.LDAPProviders {
			// Convert to providers.LDAPConfig
			ldapConfig := &providers.LDAPConfig{
				Host:                 ldapProviderConfig.Host,
				Port:                 ldapProviderConfig.Port,
				UseSSL:               ldapProviderConfig.UseSSL,
				UseTLS:               ldapProviderConfig.UseTLS,
				SkipVerify:           ldapProviderConfig.SkipVerify,
				BindDN:               ldapProviderConfig.BindDN,
				BindPassword:         ldapProviderConfig.BindPassword,
				BaseDN:               ldapProviderConfig.BaseDN,
				UserFilter:           ldapProviderConfig.UserFilter,
				GroupFilter:          ldapProviderConfig.GroupFilter,
				UserSearchBase:       ldapProviderConfig.UserSearchBase,
				GroupSearchBase:      ldapProviderConfig.GroupSearchBase,
				IsActiveDirectory:    ldapProviderConfig.IsActiveDirectory,
				Domain:               ldapProviderConfig.Domain,
				GroupMemberAttribute: ldapProviderConfig.GroupMemberAttribute,
				UserGroupAttribute:   ldapProviderConfig.UserGroupAttribute,
				RoleMappings:         ldapProviderConfig.RoleMappings,
				DefaultRoles:         ldapProviderConfig.DefaultRoles,
			}
			
			ldapClient := providers.NewLDAPClient(ldapConfig, logger)
			ldapProviders[name] = ldapClient
			am.ldapProviders[name] = ldapClient
		}
	}

	// Initialize middleware with existing signature  
	middlewareConfig := &MiddlewareConfig{
		SkipAuth: []string{"/health", "/auth/login", "/auth/callback"},
	}
	am.middleware = NewAuthMiddleware(am.sessionManager, am.jwtManager, am.rbacManager, middlewareConfig)
	
	// Initialize LDAP middleware if LDAP providers are configured
	if len(ldapProviders) > 0 {
		ldapMiddlewareConfig := &LDAPMiddlewareConfig{
			Realm:          "Nephoran Intent Operator",
			AllowBasicAuth: true,
		}
		am.ldapMiddleware = NewLDAPAuthMiddleware(ldapProviders, am.sessionManager, am.jwtManager, am.rbacManager, ldapMiddlewareConfig, logger)
	}

	// Initialize OAuth2 manager if OAuth2 providers are configured
	if len(config.Providers) > 0 {
		oauth2ManagerConfig := &OAuth2ManagerConfig{
			Enabled:       config.Enabled,
			JWTSecretKey:  config.JWTSecretKey,
			RequireAuth:   true,
			AdminUsers:    config.AdminUsers,
			OperatorUsers: config.OperatorUsers,
		}
		oauth2Manager, err := NewOAuth2Manager(oauth2ManagerConfig, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create OAuth2 manager: %w", err)
		}
		am.oauth2Manager = oauth2Manager
	}

	// Initialize handlers with existing signature
	handlersConfig := &HandlersConfig{}
	am.handlers = NewAuthHandlers(am.sessionManager, am.jwtManager, am.rbacManager, handlersConfig)

	logger.Info("AuthManager initialized successfully",
		"ldap_providers", len(am.ldapProviders),
		"oauth_providers", len(am.oauthProviders))

	return am, nil
}

// GetMiddleware returns the auth middleware
func (am *AuthManager) GetMiddleware() *AuthMiddleware {
	return am.middleware
}

// GetLDAPMiddleware returns the LDAP auth middleware
func (am *AuthManager) GetLDAPMiddleware() *LDAPAuthMiddleware {
	return am.ldapMiddleware
}

// GetOAuth2Manager returns the OAuth2 manager
func (am *AuthManager) GetOAuth2Manager() *OAuth2Manager {
	return am.oauth2Manager
}

// GetSessionManager returns the session manager
func (am *AuthManager) GetSessionManager() *SessionManager {
	return am.sessionManager
}

// GetJWTManager returns the JWT manager
func (am *AuthManager) GetJWTManager() *JWTManager {
	return am.jwtManager
}

// GetRBACManager returns the RBAC manager
func (am *AuthManager) GetRBACManager() *RBACManager {
	return am.rbacManager
}

// GetHandlers returns the auth handlers
func (am *AuthManager) GetHandlers() *AuthHandlers {
	return am.handlers
}

// ListProviders returns information about available authentication providers
func (am *AuthManager) ListProviders() map[string]interface{} {
	result := make(map[string]interface{})
	
	// LDAP providers
	ldapProviders := make(map[string]interface{})
	for name := range am.ldapProviders {
		ldapProviders[name] = map[string]interface{}{
			"type":    "ldap",
			"enabled": true,
		}
	}
	result["ldap"] = ldapProviders

	// OAuth2 providers  
	oauth2Providers := make(map[string]interface{})
	for name := range am.oauthProviders {
		oauth2Providers[name] = map[string]interface{}{
			"type":    "oauth2",
			"enabled": true,
		}
	}
	result["oauth2"] = oauth2Providers

	return result
}

// RefreshTokens refreshes access and refresh tokens
func (am *AuthManager) RefreshTokens(ctx context.Context, refreshToken string) (string, string, error) {
	if am.jwtManager == nil {
		return "", "", fmt.Errorf("JWT manager not initialized")
	}

	// Validate the refresh token
	claims, err := am.jwtManager.ValidateToken(ctx, refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Create a new user info from claims for token generation
	userInfo := &providers.UserInfo{
		Subject:       claims.Subject,
		Email:         claims.Email,
		PreferredName: claims.PreferredName,
		Name:          claims.Name,
		Roles:         claims.Roles,
		Provider:      claims.Provider,
	}

	// Generate new access token
	newAccessToken, _, err := am.jwtManager.GenerateAccessToken(ctx, userInfo, "session-"+claims.Subject)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token
	newRefreshToken, _, err := am.jwtManager.GenerateRefreshToken(ctx, userInfo, "session-"+claims.Subject)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Revoke the old refresh token
	if err := am.jwtManager.RevokeToken(ctx, refreshToken); err != nil {
		am.logger.Warn("Failed to revoke old refresh token", "error", err)
	}

	return newAccessToken, newRefreshToken, nil
}

// ValidateSession validates a session and returns session information
func (am *AuthManager) ValidateSession(ctx context.Context, sessionID string) (*UserSession, error) {
	if am.sessionManager == nil {
		return nil, fmt.Errorf("session manager not initialized")
	}

	session, err := am.sessionManager.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	// Check if session is expired - the session manager should handle this
	// Just return the session as-is for now

	return session, nil
}

// HandleHealthCheck handles health check requests
func (am *AuthManager) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status": "healthy",
		"services": map[string]string{
			"jwt_manager":     "unknown",
			"session_manager": "unknown",
			"rbac_manager":    "unknown",
		},
	}

	if am.jwtManager != nil {
		health["services"].(map[string]string)["jwt_manager"] = "healthy"
	}

	if am.sessionManager != nil {
		health["services"].(map[string]string)["session_manager"] = "healthy"
	}

	if am.rbacManager != nil {
		health["services"].(map[string]string)["rbac_manager"] = "healthy"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// Shutdown gracefully shuts down the auth manager
func (am *AuthManager) Shutdown(ctx context.Context) error {
	am.logger.Info("Shutting down AuthManager")

	var errors []error

	// Close LDAP providers
	for name, provider := range am.ldapProviders {
		if err := provider.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close LDAP provider %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	am.logger.Info("AuthManager shutdown completed")
	return nil
}