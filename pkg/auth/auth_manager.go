package auth

import (
	"log/slog"
	"time"
)

// AuthManager provides a unified interface for all authentication components
type AuthManager struct {
	JWTManager     *JWTManager
	SessionManager *SessionManager
	RBACManager    *RBACManager
	OAuth2Manager  *OAuth2Manager
	SecurityManager *SecurityManager
	logger         *slog.Logger
}

// AuthManagerConfig holds configuration for the unified auth manager
type AuthManagerConfig struct {
	JWTConfig     *JWTConfig
	SessionConfig *SessionConfig
	RBACConfig    *RBACManagerConfig
	OAuth2Config  *OAuth2ManagerConfig
	SecurityConfig *SecurityManagerConfig
}

// SecurityManagerConfig holds configuration for security manager
type SecurityManagerConfig struct {
	CSRFSecret []byte        `json:"csrf_secret"`
	PKCE       PKCEConfig    `json:"pkce"`
	CSRF       CSRFConfig    `json:"csrf"`
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
	jwtManager, err := NewJWTManager(config.JWTConfig, nil, nil, logger)
	if err != nil {
		return nil, err
	}

	// Create RBAC manager
	rbacManager := NewRBACManager(config.RBACConfig, logger)

	// Create session manager
	sessionManager := NewSessionManager(config.SessionConfig, jwtManager, rbacManager, logger)

	// Create OAuth2 manager
	oauth2Manager, err := NewOAuth2Manager(config.OAuth2Config, logger)
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