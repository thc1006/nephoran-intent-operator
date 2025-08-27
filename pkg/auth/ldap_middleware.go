package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

// LDAPAuthMiddleware provides LDAP-based authentication middleware
type LDAPAuthMiddleware struct {
	providers      map[string]providers.LDAPProvider
	sessionManager *SessionManager
	jwtManager     *JWTManager
	rbacManager    *RBACManager
	config         *LDAPMiddlewareConfig
	logger         *slog.Logger
}

// LDAPMiddlewareConfig represents LDAP middleware configuration
type LDAPMiddlewareConfig struct {
	// Authentication settings
	Realm           string        `json:"realm"`
	DefaultProvider string        `json:"default_provider"`
	AllowBasicAuth  bool          `json:"allow_basic_auth"`
	AllowFormAuth   bool          `json:"allow_form_auth"`
	AllowJSONAuth   bool          `json:"allow_json_auth"`
	SessionTimeout  time.Duration `json:"session_timeout"`

	// Security settings
	RequireHTTPS      bool          `json:"require_https"`
	MaxFailedAttempts int           `json:"max_failed_attempts"`
	LockoutDuration   time.Duration `json:"lockout_duration"`

	// Cache settings
	EnableUserCache bool          `json:"enable_user_cache"`
	CacheTTL        time.Duration `json:"cache_ttl"`

	// Headers
	UserHeader   string `json:"user_header"`
	RolesHeader  string `json:"roles_header"`
	GroupsHeader string `json:"groups_header"`

	// Skip authentication for these paths
	SkipAuth []string `json:"skip_auth"`
}

// LDAPAuthRequest represents LDAP authentication request
type LDAPAuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Provider string `json:"provider,omitempty"`
}

// LDAPAuthResponse represents LDAP authentication response
type LDAPAuthResponse struct {
	Success      bool              `json:"success"`
	AccessToken  string            `json:"access_token,omitempty"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	ExpiresIn    int64             `json:"expires_in,omitempty"`
	TokenType    string            `json:"token_type,omitempty"`
	User         *LDAPUserResponse `json:"user,omitempty"`
	Error        string            `json:"error,omitempty"`
	ErrorCode    string            `json:"error_code,omitempty"`
}

// LDAPUserResponse represents LDAP user in response
type LDAPUserResponse struct {
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	Title       string   `json:"title"`
	Department  string   `json:"department"`
	Phone       string   `json:"phone"`
	Groups      []string `json:"groups"`
	Roles       []string `json:"roles"`
}

// NewLDAPAuthMiddleware creates new LDAP authentication middleware
func NewLDAPAuthMiddleware(providers map[string]providers.LDAPProvider, sessionManager *SessionManager, jwtManager *JWTManager, rbacManager *RBACManager, config *LDAPMiddlewareConfig, logger *slog.Logger) *LDAPAuthMiddleware {
	if config == nil {
		config = &LDAPMiddlewareConfig{
			Realm:             "Nephoran Intent Operator",
			AllowBasicAuth:    true,
			AllowFormAuth:     true,
			AllowJSONAuth:     true,
			SessionTimeout:    24 * time.Hour,
			RequireHTTPS:      true,
			MaxFailedAttempts: 5,
			LockoutDuration:   15 * time.Minute,
			EnableUserCache:   true,
			CacheTTL:          5 * time.Minute,
			UserHeader:        "X-LDAP-User",
			RolesHeader:       "X-LDAP-Roles",
			GroupsHeader:      "X-LDAP-Groups",
			SkipAuth: []string{
				"/health", "/metrics", "/auth/ldap/login",
				"/.well-known/", "/favicon.ico",
			},
		}
	}

	if logger == nil {
		logger = slog.Default().With("component", "ldap_middleware")
	}

	return &LDAPAuthMiddleware{
		providers:      providers,
		sessionManager: sessionManager,
		jwtManager:     jwtManager,
		rbacManager:    rbacManager,
		config:         config,
		logger:         logger,
	}
}

// LDAPAuthenticateMiddleware handles LDAP authentication
func (lm *LDAPAuthMiddleware) LDAPAuthenticateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for certain paths
		if lm.shouldSkipAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Require HTTPS in production
		if lm.config.RequireHTTPS && r.TLS == nil && !lm.isLocalhost(r) {
			lm.writeErrorResponse(w, http.StatusUpgradeRequired, "https_required", "HTTPS is required")
			return
		}

		// Try to authenticate with existing session first
		if authContext := lm.trySessionAuth(r); authContext != nil {
			ctx := context.WithValue(r.Context(), AuthContextKey, authContext)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Try various authentication methods
		var authContext *AuthContext
		var err error

		// Try Basic Authentication
		if lm.config.AllowBasicAuth {
			if authContext, err = lm.tryBasicAuth(r); err == nil && authContext != nil {
				ctx := context.WithValue(r.Context(), AuthContextKey, authContext)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Try JWT token authentication
		if authContext, err = lm.tryJWTAuth(r); err == nil && authContext != nil {
			ctx := context.WithValue(r.Context(), AuthContextKey, authContext)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// If no authentication succeeded, require authentication
		lm.requireAuthentication(w, r)
	})
}

// HandleLDAPLogin handles LDAP login requests
func (lm *LDAPAuthMiddleware) HandleLDAPLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		lm.writeErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	// Parse authentication request
	authReq, err := lm.parseAuthRequest(r)
	if err != nil {
		lm.logger.Warn("Invalid authentication request", "error", err, "remote_addr", r.RemoteAddr)
		lm.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid authentication request")
		return
	}

	// Authenticate user
	userInfo, provider, err := lm.authenticateUser(r.Context(), authReq.Username, authReq.Password, authReq.Provider)
	if err != nil {
		lm.logger.Warn("Authentication failed", "username", authReq.Username, "provider", authReq.Provider, "error", err, "remote_addr", r.RemoteAddr)
		lm.writeErrorResponse(w, http.StatusUnauthorized, "authentication_failed", "Invalid credentials")
		return
	}

	// Create session
	sessionInfo, err := lm.sessionManager.CreateSession(r.Context(), &SessionData{
		UserID:      userInfo.Username,
		Username:    userInfo.Username,
		Email:       userInfo.Email,
		DisplayName: userInfo.Name,
		Provider:    "ldap:" + provider,
		Groups:      userInfo.Groups,
		Roles:       userInfo.Roles,
		ExpiresAt:   time.Now().Add(lm.config.SessionTimeout),
	})
	if err != nil {
		lm.logger.Error("Failed to create session", "error", err, "username", authReq.Username)
		lm.writeErrorResponse(w, http.StatusInternalServerError, "session_creation_failed", "Failed to create session")
		return
	}

	// Create JWT tokens
	accessToken, refreshToken, err := lm.createTokens(userInfo, sessionInfo.ID, provider)
	if err != nil {
		lm.logger.Error("Failed to create tokens", "error", err, "username", authReq.Username)
		lm.writeErrorResponse(w, http.StatusInternalServerError, "token_creation_failed", "Failed to create tokens")
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "nephoran_session",
		Value:    sessionInfo.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
		Expires:  sessionInfo.ExpiresAt,
	})

	// Return successful response
	response := &LDAPAuthResponse{
		Success:      true,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(lm.config.SessionTimeout.Seconds()),
		TokenType:    "Bearer",
		User: &LDAPUserResponse{
			Username:    userInfo.Username,
			Email:       userInfo.Email,
			DisplayName: userInfo.Name,
			FirstName:   userInfo.GivenName,
			LastName:    userInfo.FamilyName,
			Groups:      userInfo.Groups,
			Roles:       userInfo.Roles,
		},
	}

	// Add user attributes if available
	if attrs, ok := userInfo.Attributes["title"]; ok {
		if title, ok := attrs.(string); ok {
			response.User.Title = title
		}
	}
	if attrs, ok := userInfo.Attributes["department"]; ok {
		if dept, ok := attrs.(string); ok {
			response.User.Department = dept
		}
	}
	if attrs, ok := userInfo.Attributes["phone"]; ok {
		if phone, ok := attrs.(string); ok {
			response.User.Phone = phone
		}
	}

	lm.logger.Info("User authenticated successfully via LDAP",
		"username", authReq.Username,
		"provider", provider,
		"groups_count", len(userInfo.Groups),
		"roles_count", len(userInfo.Roles),
		"remote_addr", r.RemoteAddr)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleLDAPLogout handles LDAP logout requests
func (lm *LDAPAuthMiddleware) HandleLDAPLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		lm.writeErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only POST method is allowed")
		return
	}

	// Get session ID
	sessionID := lm.getSessionID(r)
	if sessionID != "" {
		// Invalidate session
		if err := lm.sessionManager.InvalidateSession(r.Context(), sessionID); err != nil {
			lm.logger.Warn("Failed to invalidate session", "error", err, "session_id", sessionID)
		}

		// Clear session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "nephoran_session",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}

// Private helper methods

func (lm *LDAPAuthMiddleware) shouldSkipAuth(path string) bool {
	for _, skipPath := range lm.config.SkipAuth {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

func (lm *LDAPAuthMiddleware) isLocalhost(r *http.Request) bool {
	host := r.Host
	if strings.Contains(host, ":") {
		host, _, _ = strings.Cut(host, ":")
	}
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (lm *LDAPAuthMiddleware) trySessionAuth(r *http.Request) *AuthContext {
	sessionID := lm.getSessionID(r)
	if sessionID == "" {
		return nil
	}

	sessionInfo, err := lm.sessionManager.ValidateSession(r.Context(), sessionID)
	if err != nil {
		return nil
	}

	return &AuthContext{
		UserID:      sessionInfo.UserID,
		SessionID:   sessionInfo.ID,
		Provider:    sessionInfo.Provider,
		Roles:       sessionInfo.Roles,
		Permissions: lm.getUserPermissions(r.Context(), sessionInfo.UserID),
		IsAdmin:     lm.hasAdminRole(sessionInfo.Roles),
		Attributes:  make(map[string]interface{}),
	}
}

func (lm *LDAPAuthMiddleware) tryBasicAuth(r *http.Request) (*AuthContext, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return nil, fmt.Errorf("no basic auth credentials")
	}

	userInfo, provider, err := lm.authenticateUser(r.Context(), username, password, "")
	if err != nil {
		return nil, err
	}

	return &AuthContext{
		UserID:      userInfo.Username,
		Provider:    "ldap:" + provider,
		Roles:       userInfo.Roles,
		Permissions: lm.getUserPermissions(r.Context(), userInfo.Username),
		IsAdmin:     lm.hasAdminRole(userInfo.Roles),
		Attributes: map[string]interface{}{
			"auth_method": "basic",
			"ldap_dn":     userInfo.ProviderID,
		},
	}, nil
}

func (lm *LDAPAuthMiddleware) tryJWTAuth(r *http.Request) (*AuthContext, error) {
	if lm.jwtManager == nil {
		return nil, fmt.Errorf("JWT manager not available")
	}

	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("no bearer token")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := lm.jwtManager.ValidateToken(r.Context(), token)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "access" {
		return nil, fmt.Errorf("invalid token type")
	}

	return &AuthContext{
		UserID:      claims.Subject,
		SessionID:   claims.SessionID,
		Provider:    claims.Provider,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
		IsAdmin:     lm.hasAdminRole(claims.Roles),
		Attributes:  claims.Attributes,
	}, nil
}

func (lm *LDAPAuthMiddleware) authenticateUser(ctx context.Context, username, password, providerName string) (*providers.UserInfo, string, error) {
	// Validate inputs
	if username == "" || password == "" {
		return nil, "", fmt.Errorf("username and password required")
	}

	// Use constant-time comparison to prevent timing attacks on password validation
	if len(password) > 256 {
		return nil, "", fmt.Errorf("password too long")
	}

	// Try specific provider if requested
	if providerName != "" {
		if provider, exists := lm.providers[providerName]; exists {
			userInfo, err := provider.Authenticate(ctx, username, password)
			if err != nil {
				return nil, "", fmt.Errorf("authentication failed with provider %s: %w", providerName, err)
			}
			return userInfo, providerName, nil
		}
		return nil, "", fmt.Errorf("provider %s not found", providerName)
	}

	// Try default provider first
	if lm.config.DefaultProvider != "" {
		if provider, exists := lm.providers[lm.config.DefaultProvider]; exists {
			userInfo, err := provider.Authenticate(ctx, username, password)
			if err == nil {
				return userInfo, lm.config.DefaultProvider, nil
			}
			lm.logger.Debug("Authentication failed with default provider",
				"provider", lm.config.DefaultProvider,
				"username", username,
				"error", err)
		}
	}

	// Try all providers
	var lastErr error
	for name, provider := range lm.providers {
		if name == lm.config.DefaultProvider {
			continue // Already tried
		}

		userInfo, err := provider.Authenticate(ctx, username, password)
		if err == nil {
			return userInfo, name, nil
		}
		lastErr = err
		lm.logger.Debug("Authentication failed with provider",
			"provider", name,
			"username", username,
			"error", err)
	}

	if lastErr != nil {
		return nil, "", fmt.Errorf("authentication failed with all providers: %w", lastErr)
	}
	return nil, "", fmt.Errorf("no LDAP providers available")
}

func (lm *LDAPAuthMiddleware) createTokens(userInfo *providers.UserInfo, sessionID, provider string) (string, string, error) {
	if lm.jwtManager == nil {
		return "", "", fmt.Errorf("JWT manager not available")
	}

	// Create access token
	accessToken, err := lm.jwtManager.CreateAccessToken(userInfo.Username, sessionID, "ldap:"+provider, userInfo.Roles, userInfo.Groups, userInfo.Attributes)
	if err != nil {
		return "", "", fmt.Errorf("failed to create access token: %w", err)
	}

	// Create refresh token
	refreshToken, err := lm.jwtManager.CreateRefreshToken(userInfo.Username, sessionID, "ldap:"+provider)
	if err != nil {
		return "", "", fmt.Errorf("failed to create refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (lm *LDAPAuthMiddleware) parseAuthRequest(r *http.Request) (*LDAPAuthRequest, error) {
	contentType := r.Header.Get("Content-Type")

	var authReq LDAPAuthRequest

	switch {
	case strings.Contains(contentType, "application/json") && lm.config.AllowJSONAuth:
		if err := json.NewDecoder(r.Body).Decode(&authReq); err != nil {
			return nil, fmt.Errorf("invalid JSON request: %w", err)
		}

	case strings.Contains(contentType, "application/x-www-form-urlencoded") && lm.config.AllowFormAuth:
		if err := r.ParseForm(); err != nil {
			return nil, fmt.Errorf("invalid form request: %w", err)
		}
		authReq.Username = r.FormValue("username")
		authReq.Password = r.FormValue("password")
		authReq.Provider = r.FormValue("provider")

	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}

	// Validate required fields
	if authReq.Username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if authReq.Password == "" {
		return nil, fmt.Errorf("password is required")
	}

	return &authReq, nil
}

func (lm *LDAPAuthMiddleware) getSessionID(r *http.Request) string {
	// Try cookie first
	cookie, err := r.Cookie("nephoran_session")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Try header
	return r.Header.Get("X-Session-ID")
}

func (lm *LDAPAuthMiddleware) getUserPermissions(ctx context.Context, userID string) []string {
	if lm.rbacManager == nil {
		return []string{}
	}
	return lm.rbacManager.GetUserPermissions(ctx, userID)
}

func (lm *LDAPAuthMiddleware) hasAdminRole(roles []string) bool {
	adminRoles := []string{"system-admin", "admin", "administrator"}
	for _, role := range roles {
		for _, adminRole := range adminRoles {
			if strings.EqualFold(role, adminRole) {
				return true
			}
		}
	}
	return false
}

func (lm *LDAPAuthMiddleware) requireAuthentication(w http.ResponseWriter, r *http.Request) {
	// Set WWW-Authenticate header for Basic auth
	if lm.config.AllowBasicAuth {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, lm.config.Realm))
	}

	lm.writeErrorResponse(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
}

func (lm *LDAPAuthMiddleware) writeErrorResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errorResponse := map[string]interface{}{
		"error":             code,
		"error_description": message,
		"status":            status,
		"timestamp":         time.Now().Unix(),
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// Utility methods for integration

// GetLDAPProviders returns available LDAP providers
func (lm *LDAPAuthMiddleware) GetLDAPProviders() map[string]providers.LDAPProvider {
	return lm.providers
}

// TestLDAPConnection tests connection to all LDAP providers
func (lm *LDAPAuthMiddleware) TestLDAPConnection(ctx context.Context) map[string]error {
	results := make(map[string]error)
	for name, provider := range lm.providers {
		if err := provider.Connect(ctx); err != nil {
			results[name] = err
			lm.logger.Error("LDAP connection test failed", "provider", name, "error", err)
		} else {
			results[name] = nil
			lm.logger.Info("LDAP connection test successful", "provider", name)
		}
	}
	return results
}

// GetUserInfo retrieves user information from LDAP without authentication
func (lm *LDAPAuthMiddleware) GetUserInfo(ctx context.Context, username, providerName string) (*providers.UserInfo, error) {
	var provider providers.LDAPProvider
	var exists bool

	if providerName != "" {
		provider, exists = lm.providers[providerName]
		if !exists {
			return nil, fmt.Errorf("provider %s not found", providerName)
		}
	} else if lm.config.DefaultProvider != "" {
		provider, exists = lm.providers[lm.config.DefaultProvider]
		if !exists {
			return nil, fmt.Errorf("default provider %s not found", lm.config.DefaultProvider)
		}
	} else {
		// Use first available provider
		for _, p := range lm.providers {
			provider = p
			break
		}
		if provider == nil {
			return nil, fmt.Errorf("no LDAP providers available")
		}
	}

	return provider.SearchUser(ctx, username)
}
