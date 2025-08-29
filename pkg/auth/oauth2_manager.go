
package auth



import (

	"context"

	"fmt"

	"log/slog"

	"net/http"

	"sync"

	"time"



	"github.com/gorilla/mux"



	"github.com/thc1006/nephoran-intent-operator/pkg/middleware"

)



// OAuth2Manager handles OAuth2 authentication setup and middleware.

type OAuth2Manager struct {

	authMiddleware *AuthMiddleware

	authHandlers   *AuthHandlers

	config         *OAuth2ManagerConfig

	logger         *slog.Logger

}



// OAuth2ManagerConfig holds configuration for OAuth2 manager.

type OAuth2ManagerConfig struct {

	Enabled          bool

	AuthConfigFile   string

	JWTSecretKey     string

	RequireAuth      bool

	AdminUsers       []string

	OperatorUsers    []string

	StreamingEnabled bool

	MaxRequestSize   int64

}



// NewOAuth2Manager creates a new OAuth2Manager instance.

func NewOAuth2Manager(config *OAuth2ManagerConfig, logger *slog.Logger) (*OAuth2Manager, error) {

	if !config.Enabled {

		logger.Info("OAuth2 authentication disabled")

		return &OAuth2Manager{

			config: config,

			logger: logger,

		}, nil

	}



	authConfig, err := LoadAuthConfig(config.AuthConfigFile)

	if err != nil {

		return nil, err

	}



	oauth2Config, err := authConfig.ToOAuth2Config()

	if err != nil {

		return nil, err

	}



	// Initialize JWT manager first (required for session manager).

	jwtConfig := &JWTConfig{

		Issuer:               "nephoran-intent-operator",

		SigningKey:           config.JWTSecretKey,

		KeyRotationPeriod:    24 * time.Hour,

		DefaultTTL:           24 * time.Hour,

		RefreshTTL:           168 * time.Hour, // 7 days

		RequireSecureCookies: true,

		CookieDomain:         "",

		CookiePath:           "/",

	}



	// Create simple in-memory token store and blacklist.

	tokenStore := NewMemoryTokenStore()

	tokenBlacklist := NewMemoryTokenBlacklist()



	jwtManager, err := NewJWTManager(jwtConfig, tokenStore, tokenBlacklist, logger)

	if err != nil {

		return nil, fmt.Errorf("failed to create JWT manager: %w", err)

	}



	// Initialize RBAC manager (required for session manager).

	rbacManager := NewRBACManager(&RBACManagerConfig{

		CacheTTL:           24 * time.Hour,

		EnableHierarchy:    true,

		DefaultDenyAll:     false,

		PolicyEvaluation:   "deny-overrides",

		MaxPolicyDepth:     10,

		EnableAuditLogging: true,

	}, logger)



	// Initialize session manager.

	sessionManager := NewSessionManager(&SessionConfig{

		SessionTimeout:   24 * time.Hour,

		RefreshThreshold: 1 * time.Hour,

		MaxSessions:      10000,

		SecureCookies:    true,

		SameSiteCookies:  "strict",

		CookieDomain:     "",

		CookiePath:       "/",

		EnableSSO:        false,

		EnableCSRF:       true,

		StateTimeout:     10 * time.Minute,

		RequireHTTPS:     true,

		CleanupInterval:  1 * time.Hour,

	}, jwtManager, rbacManager, logger)



	middlewareConfig := &MiddlewareConfig{

		SkipAuth:              []string{"/health", "/ready", "/metrics"},

		EnableCORS:            true,

		AllowedOrigins:        []string{"*"},

		AllowedMethods:        []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},

		AllowedHeaders:        []string{"Content-Type", "Authorization"},

		AllowCredentials:      true,

		MaxAge:                3600,

		EnableSecurityHeaders: true,

	}



	authMiddleware := NewAuthMiddleware(sessionManager, jwtManager, rbacManager, middlewareConfig)



	// Initialize auth handlers.

	handlerConfig := &HandlersConfig{

		BaseURL:         "http://localhost:8080",

		DefaultRedirect: "/",

		LoginPath:       "/auth/login",

		CallbackPath:    "/auth/callback",

		LogoutPath:      "/auth/logout",

		UserInfoPath:    "/auth/userinfo",

		EnableAPITokens: true,

		TokenPath:       "/auth/token",

	}

	authHandlers := NewAuthHandlers(sessionManager, jwtManager, rbacManager, handlerConfig)



	logger.Info("OAuth2 authentication enabled",

		slog.Int("providers", len(oauth2Config.Providers)))



	return &OAuth2Manager{

		authMiddleware: authMiddleware,

		authHandlers:   authHandlers,

		config:         config,

		logger:         logger,

	}, nil

}



// SetupRoutes configures OAuth2 routes on the given router.

func (om *OAuth2Manager) SetupRoutes(router *mux.Router) {

	if !om.config.Enabled || om.authHandlers == nil {

		return

	}



	// OAuth2 authentication routes.

	router.HandleFunc("/auth/login/{provider}", om.authHandlers.InitiateLoginHandler).Methods("GET")

	router.HandleFunc("/auth/callback/{provider}", om.authHandlers.CallbackHandler).Methods("GET")

	router.HandleFunc("/auth/refresh", om.authHandlers.RefreshTokenHandler).Methods("POST")

	router.HandleFunc("/auth/logout", om.authHandlers.LogoutHandler).Methods("POST")

	router.HandleFunc("/auth/userinfo", om.authHandlers.GetUserInfoHandler).Methods("GET")



	om.logger.Info("OAuth2 routes configured")

}



// ConfigureProtectedRoutes sets up protected routes with authentication middleware.

func (om *OAuth2Manager) ConfigureProtectedRoutes(router *mux.Router, handlers *RouteHandlers) {

	if !om.config.Enabled || !om.config.RequireAuth || om.authMiddleware == nil {

		// No authentication required - setup direct routes.

		om.setupDirectRoutes(router, handlers)

		return

	}



	// Apply authentication middleware to protected routes.

	protectedRouter := router.PathPrefix("/").Subrouter()

	protectedRouter.Use(om.authMiddleware.AuthenticateMiddleware)



	// Main processing endpoint - requires operator role.

	protectedRouter.HandleFunc("/process", handlers.ProcessIntent).Methods("POST")

	protectedRouter.Use(om.authMiddleware.RequireOperator())



	// Streaming endpoint - requires operator role (conditional registration).

	if om.config.StreamingEnabled && handlers.StreamingHandler != nil {

		protectedRouter.HandleFunc("/stream", handlers.StreamingHandler).Methods("POST")

	}



	// Admin endpoints - requires admin role.

	adminRouter := protectedRouter.PathPrefix("/admin").Subrouter()

	adminRouter.Use(om.authMiddleware.RequireAdmin())

	adminRouter.HandleFunc("/status", handlers.Status).Methods("GET")

	adminRouter.HandleFunc("/circuit-breaker/status", handlers.CircuitBreakerStatus).Methods("GET")



	om.logger.Info("Protected routes configured with authentication")

}



// setupDirectRoutes configures routes without authentication.

func (om *OAuth2Manager) setupDirectRoutes(router *mux.Router, handlers *RouteHandlers) {

	router.HandleFunc("/process", handlers.ProcessIntent).Methods("POST")

	router.HandleFunc("/status", handlers.Status).Methods("GET")

	router.HandleFunc("/circuit-breaker/status", handlers.CircuitBreakerStatus).Methods("GET")



	// Streaming endpoint (conditional registration).

	if om.config.StreamingEnabled && handlers.StreamingHandler != nil {

		router.HandleFunc("/stream", handlers.StreamingHandler).Methods("POST")

	}



	om.logger.Info("Direct routes configured without authentication")

}



// IsEnabled returns true if OAuth2 authentication is enabled.

func (om *OAuth2Manager) IsEnabled() bool {

	return om.config.Enabled

}



// RequiresAuth returns true if authentication is required.

func (om *OAuth2Manager) RequiresAuth() bool {

	return om.config.RequireAuth

}



// RouteHandlers holds all the HTTP handlers for the service.

type RouteHandlers struct {

	ProcessIntent        http.HandlerFunc

	Status               http.HandlerFunc

	CircuitBreakerStatus http.HandlerFunc

	StreamingHandler     http.HandlerFunc

	Metrics              http.HandlerFunc

}



// CreateHandlersWithSizeLimit creates RouteHandlers with MaxBytesHandler applied to POST endpoints.

func (om *OAuth2Manager) CreateHandlersWithSizeLimit(

	processIntent http.HandlerFunc,

	status http.HandlerFunc,

	circuitBreakerStatus http.HandlerFunc,

	streamingHandler http.HandlerFunc,

	metrics http.HandlerFunc,

) *RouteHandlers {

	// Apply MaxBytesHandler to POST endpoints that need request size limiting.

	var processIntentHandler http.HandlerFunc

	var streamingHandlerWrapped http.HandlerFunc



	if om.config.MaxRequestSize > 0 {

		processIntentHandler = middleware.MaxBytesHandler(om.config.MaxRequestSize, om.logger, processIntent)

		if streamingHandler != nil {

			streamingHandlerWrapped = middleware.MaxBytesHandler(om.config.MaxRequestSize, om.logger, streamingHandler)

		}

	} else {

		processIntentHandler = processIntent

		streamingHandlerWrapped = streamingHandler

	}



	return &RouteHandlers{

		ProcessIntent:        processIntentHandler,

		Status:               status,

		CircuitBreakerStatus: circuitBreakerStatus,

		StreamingHandler:     streamingHandlerWrapped,

		Metrics:              metrics,

	}

}



// AuthenticationInfo provides information about the authentication state.

type AuthenticationInfo struct {

	Enabled     bool     `json:"enabled"`

	RequireAuth bool     `json:"require_auth"`

	Providers   []string `json:"providers,omitempty"`

}



// GetAuthenticationInfo returns information about the current authentication configuration.

func (om *OAuth2Manager) GetAuthenticationInfo() *AuthenticationInfo {

	info := &AuthenticationInfo{

		Enabled:     om.config.Enabled,

		RequireAuth: om.config.RequireAuth,

	}



	// TODO: Get configured providers from auth middleware.

	// if om.authMiddleware != nil {.

	//	info.Providers = om.authMiddleware.GetProviders()

	// }.



	return info

}



// ValidateConfiguration validates the OAuth2 manager configuration.

func (config *OAuth2ManagerConfig) Validate() error {

	if config.Enabled && config.JWTSecretKey == "" {

		return ErrMissingJWTSecret

	}



	return nil

}



// AuthError represents an authentication error.

type AuthError struct {

	Code    string `json:"code"`

	Message string `json:"message"`

}



// Error performs error operation.

func (e *AuthError) Error() string {

	return e.Message

}



// Common errors.

var (

	// ErrMissingJWTSecret holds errmissingjwtsecret value.

	ErrMissingJWTSecret = &AuthError{Code: "missing_jwt_secret", Message: "JWT secret key is required when OAuth2 is enabled"}

)



// Simple in-memory implementations for basic functionality.



// MemoryTokenStore provides a simple in-memory token store.

type MemoryTokenStore struct {

	tokens map[string]*TokenInfo

	mu     sync.RWMutex

}



// NewMemoryTokenStore performs newmemorytokenstore operation.

func NewMemoryTokenStore() *MemoryTokenStore {

	return &MemoryTokenStore{

		tokens: make(map[string]*TokenInfo),

	}

}



// StoreToken performs storetoken operation.

func (m *MemoryTokenStore) StoreToken(ctx context.Context, tokenID string, token *TokenInfo) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.tokens[tokenID] = token

	return nil

}



// GetToken performs gettoken operation.

func (m *MemoryTokenStore) GetToken(ctx context.Context, tokenID string) (*TokenInfo, error) {

	m.mu.RLock()

	defer m.mu.RUnlock()

	token, exists := m.tokens[tokenID]

	if !exists {

		return nil, fmt.Errorf("token not found")

	}

	return token, nil

}



// UpdateToken performs updatetoken operation.

func (m *MemoryTokenStore) UpdateToken(ctx context.Context, tokenID string, token *TokenInfo) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.tokens[tokenID] = token

	return nil

}



// DeleteToken performs deletetoken operation.

func (m *MemoryTokenStore) DeleteToken(ctx context.Context, tokenID string) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	delete(m.tokens, tokenID)

	return nil

}



// ListUserTokens performs listusertokens operation.

func (m *MemoryTokenStore) ListUserTokens(ctx context.Context, userID string) ([]*TokenInfo, error) {

	m.mu.RLock()

	defer m.mu.RUnlock()

	var tokens []*TokenInfo

	for _, token := range m.tokens {

		if token.UserID == userID {

			tokens = append(tokens, token)

		}

	}

	return tokens, nil

}



// CleanupExpired performs cleanupexpired operation.

func (m *MemoryTokenStore) CleanupExpired(ctx context.Context) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	now := time.Now()

	for tokenID, token := range m.tokens {

		if token.ExpiresAt.Before(now) {

			delete(m.tokens, tokenID)

		}

	}

	return nil

}



// MemoryTokenBlacklist provides a simple in-memory token blacklist.

type MemoryTokenBlacklist struct {

	blacklisted map[string]time.Time

	mu          sync.RWMutex

}



// NewMemoryTokenBlacklist performs newmemorytokenblacklist operation.

func NewMemoryTokenBlacklist() *MemoryTokenBlacklist {

	return &MemoryTokenBlacklist{

		blacklisted: make(map[string]time.Time),

	}

}



// BlacklistToken performs blacklisttoken operation.

func (m *MemoryTokenBlacklist) BlacklistToken(ctx context.Context, tokenID string, expiresAt time.Time) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.blacklisted[tokenID] = expiresAt

	return nil

}



// IsTokenBlacklisted performs istokenblacklisted operation.

func (m *MemoryTokenBlacklist) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {

	m.mu.RLock()

	defer m.mu.RUnlock()

	_, exists := m.blacklisted[tokenID]

	return exists, nil

}



// CleanupExpired performs cleanupexpired operation.

func (m *MemoryTokenBlacklist) CleanupExpired(ctx context.Context) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	now := time.Now()

	for tokenID, expiresAt := range m.blacklisted {

		if expiresAt.Before(now) {

			delete(m.blacklisted, tokenID)

		}

	}

	return nil

}

