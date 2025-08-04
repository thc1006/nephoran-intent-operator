package auth

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"
)

// OAuth2Manager handles OAuth2 authentication setup and middleware
type OAuth2Manager struct {
	authMiddleware *AuthMiddleware
	config         *OAuth2ManagerConfig
	logger         *slog.Logger
}

// OAuth2ManagerConfig holds configuration for OAuth2 manager
type OAuth2ManagerConfig struct {
	Enabled        bool
	AuthConfigFile string
	JWTSecretKey   string
	RequireAuth    bool
	AdminUsers     []string
	OperatorUsers  []string
}

// NewOAuth2Manager creates a new OAuth2Manager instance
func NewOAuth2Manager(config *OAuth2ManagerConfig, logger *slog.Logger) (*OAuth2Manager, error) {
	if !config.Enabled {
		logger.Info("OAuth2 authentication disabled")
		return &OAuth2Manager{
			config: config,
			logger: logger,
		}, nil
	}

	authConfig, err := LoadAuthConfig()
	if err != nil {
		return nil, err
	}

	oauth2Config, err := authConfig.ToOAuth2Config()
	if err != nil {
		return nil, err
	}

	authMiddleware := NewAuthMiddleware(oauth2Config, []byte(config.JWTSecretKey))

	logger.Info("OAuth2 authentication enabled", 
		slog.Int("providers", len(oauth2Config.Providers)))

	return &OAuth2Manager{
		authMiddleware: authMiddleware,
		config:        config,
		logger:        logger,
	}, nil
}

// SetupRoutes configures OAuth2 routes on the given router
func (om *OAuth2Manager) SetupRoutes(router *mux.Router) {
	if !om.config.Enabled || om.authMiddleware == nil {
		return
	}

	// OAuth2 authentication routes
	router.HandleFunc("/auth/login/{provider}", om.authMiddleware.LoginHandler).Methods("GET")
	router.HandleFunc("/auth/callback/{provider}", om.authMiddleware.CallbackHandler).Methods("GET")
	router.HandleFunc("/auth/refresh", om.authMiddleware.RefreshHandler).Methods("POST")
	router.HandleFunc("/auth/logout", om.authMiddleware.LogoutHandler).Methods("POST")
	router.HandleFunc("/auth/userinfo", om.authMiddleware.UserInfoHandler).Methods("GET")

	om.logger.Info("OAuth2 routes configured")
}

// ConfigureProtectedRoutes sets up protected routes with authentication middleware
func (om *OAuth2Manager) ConfigureProtectedRoutes(router *mux.Router, handlers *RouteHandlers) {
	if !om.config.Enabled || !om.config.RequireAuth || om.authMiddleware == nil {
		// No authentication required - setup direct routes
		om.setupDirectRoutes(router, handlers)
		return
	}

	// Apply authentication middleware to protected routes
	protectedRouter := router.PathPrefix("/").Subrouter()
	protectedRouter.Use(om.authMiddleware.Authenticate)

	// Main processing endpoint - requires operator role
	protectedRouter.HandleFunc("/process", handlers.ProcessIntent).Methods("POST")
	protectedRouter.Use(om.authMiddleware.RequireOperator())

	// Streaming endpoint - requires operator role
	if handlers.StreamingHandler != nil {
		protectedRouter.HandleFunc("/stream", handlers.StreamingHandler).Methods("POST")
	}

	// Admin endpoints - requires admin role
	adminRouter := protectedRouter.PathPrefix("/admin").Subrouter()
	adminRouter.Use(om.authMiddleware.RequireAdmin())
	adminRouter.HandleFunc("/status", handlers.Status).Methods("GET")
	adminRouter.HandleFunc("/circuit-breaker/status", handlers.CircuitBreakerStatus).Methods("GET")

	om.logger.Info("Protected routes configured with authentication")
}

// setupDirectRoutes configures routes without authentication
func (om *OAuth2Manager) setupDirectRoutes(router *mux.Router, handlers *RouteHandlers) {
	router.HandleFunc("/process", handlers.ProcessIntent).Methods("POST")
	router.HandleFunc("/status", handlers.Status).Methods("GET")
	router.HandleFunc("/circuit-breaker/status", handlers.CircuitBreakerStatus).Methods("GET")

	if handlers.StreamingHandler != nil {
		router.HandleFunc("/stream", handlers.StreamingHandler).Methods("POST")
	}

	om.logger.Info("Direct routes configured without authentication")
}

// IsEnabled returns true if OAuth2 authentication is enabled
func (om *OAuth2Manager) IsEnabled() bool {
	return om.config.Enabled
}

// RequiresAuth returns true if authentication is required
func (om *OAuth2Manager) RequiresAuth() bool {
	return om.config.RequireAuth
}

// RouteHandlers holds all the HTTP handlers for the service
type RouteHandlers struct {
	ProcessIntent        http.HandlerFunc
	Status               http.HandlerFunc
	CircuitBreakerStatus http.HandlerFunc
	StreamingHandler     http.HandlerFunc
	Metrics              http.HandlerFunc
}

// AuthenticationInfo provides information about the authentication state
type AuthenticationInfo struct {
	Enabled     bool     `json:"enabled"`
	RequireAuth bool     `json:"require_auth"`
	Providers   []string `json:"providers,omitempty"`
}

// GetAuthenticationInfo returns information about the current authentication configuration
func (om *OAuth2Manager) GetAuthenticationInfo() *AuthenticationInfo {
	info := &AuthenticationInfo{
		Enabled:     om.config.Enabled,
		RequireAuth: om.config.RequireAuth,
	}

	// TODO: Implement provider listing when AuthMiddleware has proper config access
	// if om.authMiddleware != nil {
	//     info.Providers = om.authMiddleware.GetProviders()
	// }

	return info
}

// ValidateConfiguration validates the OAuth2 manager configuration
func (config *OAuth2ManagerConfig) Validate() error {
	if config.Enabled && config.JWTSecretKey == "" {
		return ErrMissingJWTSecret
	}

	return nil
}

// AuthError represents an authentication error
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return e.Message
}

// Common errors
var (
	ErrMissingJWTSecret = &AuthError{Code: "missing_jwt_secret", Message: "JWT secret key is required when OAuth2 is enabled"}
)