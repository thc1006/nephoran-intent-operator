package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

// SessionManager manages user sessions and SSO.

type SessionManager struct {
	// Session storage.

	sessions map[string]*UserSession

	// Components.

	jwtManager *JWTManager

	rbacManager *RBACManager

	providers map[string]providers.OAuthProvider

	// Configuration.

	config *SessionConfig

	// State management.

	stateStore map[string]*State // CSRF state management

	logger *slog.Logger

	mutex sync.RWMutex
}

// UserSession is defined in interfaces.go.

// State represents OAuth2 authorization state.

type State struct {
	State string `json:"state"`

	Provider string `json:"provider"`

	RedirectURI string `json:"redirect_uri"`

	PKCEChallenge *providers.PKCEChallenge `json:"pkce_challenge,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	ExpiresAt time.Time `json:"expires_at"`

	IPAddress string `json:"ip_address"`

	UserAgent string `json:"user_agent"`

	// Custom parameters.

	CustomParams map[string]string `json:"custom_params,omitempty"`
}

// SessionConfig represents session configuration.

type SessionConfig struct {
	SessionTimeout time.Duration `json:"session_timeout"`

	RefreshThreshold time.Duration `json:"refresh_threshold"`

	MaxSessions int `json:"max_sessions"`

	SecureCookies bool `json:"secure_cookies"`

	SameSiteCookies string `json:"same_site_cookies"`

	CookieDomain string `json:"cookie_domain"`

	CookiePath string `json:"cookie_path"`

	// SSO settings.

	EnableSSO bool `json:"enable_sso"`

	SSODomain string `json:"sso_domain"`

	CrossDomainSSO bool `json:"cross_domain_sso"`

	// Security settings.

	EnableCSRF bool `json:"enable_csrf"`

	StateTimeout time.Duration `json:"state_timeout"`

	RequireHTTPS bool `json:"require_https"`

	// Session cleanup.

	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// LoginRequest represents a login initiation request.

type LoginRequest struct {
	Provider string `json:"provider"`

	RedirectURI string `json:"redirect_uri,omitempty"`

	State string `json:"state,omitempty"`

	Options map[string]string `json:"options,omitempty"`

	IPAddress string `json:"ip_address"`

	UserAgent string `json:"user_agent"`
}

// LoginResponse represents login initiation response.

type LoginResponse struct {
	AuthURL string `json:"auth_url"`

	State string `json:"state"`

	SessionID string `json:"session_id,omitempty"`

	PKCEChallenge string `json:"pkce_challenge,omitempty"`
}

// CallbackRequest represents OAuth2 callback request.

type CallbackRequest struct {
	Provider string `json:"provider"`

	Code string `json:"code"`

	State string `json:"state"`

	RedirectURI string `json:"redirect_uri"`

	IPAddress string `json:"ip_address"`

	UserAgent string `json:"user_agent"`
}

// CallbackResponse represents OAuth2 callback response.

type CallbackResponse struct {
	Success bool `json:"success"`

	SessionID string `json:"session_id"`

	AccessToken string `json:"access_token"`

	RefreshToken string `json:"refresh_token"`

	IDToken string `json:"id_token,omitempty"`

	UserInfo *providers.UserInfo `json:"user_info"`

	RedirectURL string `json:"redirect_url,omitempty"`

	Error string `json:"error,omitempty"`
}

// SessionInfo represents session information for API responses.

type SessionInfo struct {
	ID string `json:"id"`

	UserID string `json:"user_id"`

	Provider string `json:"provider"`

	CreatedAt time.Time `json:"created_at"`

	LastActivity time.Time `json:"last_activity"`

	ExpiresAt time.Time `json:"expires_at"`

	IPAddress string `json:"ip_address"`

	UserAgent string `json:"user_agent"`

	Roles []string `json:"roles"`

	SSOEnabled bool `json:"sso_enabled"`

	UserInfo *providers.UserInfo `json:"user_info,omitempty"`
}

// NewSessionManager creates a new session manager.

func NewSessionManager(config *SessionConfig, jwtManager *JWTManager, rbacManager *RBACManager, logger *slog.Logger) *SessionManager {
	if config == nil {
		config = &SessionConfig{
			SessionTimeout: 24 * time.Hour,

			RefreshThreshold: 15 * time.Minute,

			MaxSessions: 10,

			SecureCookies: true,

			SameSiteCookies: "Strict",

			CookiePath: "/",

			EnableSSO: true,

			EnableCSRF: true,

			StateTimeout: 10 * time.Minute,

			RequireHTTPS: true,

			CleanupInterval: 1 * time.Hour,
		}
	}

	manager := &SessionManager{
		sessions: make(map[string]*UserSession),

		stateStore: make(map[string]*State),

		providers: make(map[string]providers.OAuthProvider),

		jwtManager: jwtManager,

		rbacManager: rbacManager,

		config: config,

		logger: logger,
	}

	// Start background cleanup.

	go manager.cleanupLoop()

	return manager
}

// RegisterProvider registers an OAuth2 provider.

func (sm *SessionManager) RegisterProvider(provider providers.OAuthProvider) {
	sm.mutex.Lock()

	defer sm.mutex.Unlock()

	name := provider.GetProviderName()

	sm.providers[name] = provider

	sm.logger.Info("OAuth2 provider registered",

		"provider", name,

		"features", provider.GetConfiguration().Features)
}

// InitiateLogin starts the OAuth2 login flow.

func (sm *SessionManager) InitiateLogin(ctx context.Context, request *LoginRequest) (*LoginResponse, error) {
	sm.mutex.Lock()

	defer sm.mutex.Unlock()

	// Validate provider.

	provider, exists := sm.providers[request.Provider]

	if !exists {
		return nil, fmt.Errorf("unknown provider: %s", request.Provider)
	}

	// Generate state.

	state := request.State

	if state == "" {
		state = sm.generateState()
	}

	// Build auth options.

	capacity := len(request.Options)

	if sm.config.EnableCSRF {
		capacity++
	}

	authOptions := make([]providers.AuthOption, 0, capacity)

	if sm.config.EnableCSRF {
		authOptions = append(authOptions, providers.WithPKCE())
	}

	// Add custom options.

	for key, value := range request.Options {
		authOptions = append(authOptions, providers.WithCustomParam(key, value))
	}

	// Get authorization URL.

	authURL, challenge, err := provider.GetAuthorizationURL(state, request.RedirectURI, authOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization URL: %w", err)
	}

	// Store state.

	authState := &State{
		State: state,

		Provider: request.Provider,

		RedirectURI: request.RedirectURI,

		PKCEChallenge: challenge,

		CreatedAt: time.Now(),

		ExpiresAt: time.Now().Add(sm.config.StateTimeout),

		IPAddress: request.IPAddress,

		UserAgent: request.UserAgent,

		CustomParams: request.Options,
	}

	sm.stateStore[state] = authState

	sm.logger.Info("Login initiated",

		"provider", request.Provider,

		"state", state,

		"ip_address", request.IPAddress)

	response := &LoginResponse{
		AuthURL: authURL,

		State: state,
	}

	if challenge != nil {
		response.PKCEChallenge = challenge.CodeChallenge
	}

	return response, nil
}

// HandleCallback handles OAuth2 callback and creates session.

func (sm *SessionManager) HandleCallback(ctx context.Context, request *CallbackRequest) (*CallbackResponse, error) {
	sm.mutex.Lock()

	defer sm.mutex.Unlock()

	// Validate state.

	authState, exists := sm.stateStore[request.State]

	if !exists {
		return &CallbackResponse{
			Success: false,

			Error: "Invalid or expired state parameter",
		}, nil
	}

	// Check state expiration.

	if time.Now().After(authState.ExpiresAt) {

		delete(sm.stateStore, request.State)

		return &CallbackResponse{
			Success: false,

			Error: "Authorization state has expired",
		}, nil

	}

	// Validate provider match.

	if authState.Provider != request.Provider {
		return &CallbackResponse{
			Success: false,

			Error: "Provider mismatch in callback",
		}, nil
	}

	// Get provider.

	provider, exists := sm.providers[request.Provider]

	if !exists {
		return &CallbackResponse{
			Success: false,

			Error: "Unknown provider",
		}, nil
	}

	// Exchange code for token.

	tokenResponse, err := provider.ExchangeCodeForToken(ctx, request.Code, request.RedirectURI, authState.PKCEChallenge)
	if err != nil {

		sm.logger.Error("Token exchange failed",

			"provider", request.Provider,

			"error", err)

		return &CallbackResponse{
			Success: false,

			Error: "Failed to exchange authorization code",
		}, nil

	}

	// Get user info.

	userInfo, err := provider.GetUserInfo(ctx, tokenResponse.AccessToken)
	if err != nil {

		sm.logger.Error("Failed to get user info",

			"provider", request.Provider,

			"error", err)

		return &CallbackResponse{
			Success: false,

			Error: "Failed to retrieve user information",
		}, nil

	}

	// Create user session.

	session, err := sm.createUserSession(ctx, userInfo, tokenResponse, request)
	if err != nil {

		sm.logger.Error("Failed to create user session",

			"provider", request.Provider,

			"user_id", userInfo.Subject,

			"error", err)

		return &CallbackResponse{
			Success: false,

			Error: "Failed to create user session",
		}, nil

	}

	// Clean up state.

	delete(sm.stateStore, request.State)

	sm.logger.Info("User authentication successful",

		"provider", request.Provider,

		"user_id", userInfo.Subject,

		"session_id", session.ID,

		"ip_address", request.IPAddress)

	return &CallbackResponse{
		Success: true,

		SessionID: session.ID,

		AccessToken: session.AccessToken,

		RefreshToken: session.RefreshToken,

		IDToken: session.IDToken,

		UserInfo: userInfo,
	}, nil
}

// CreateSession creates a new user session.

func (sm *SessionManager) createUserSession(ctx context.Context, userInfo *providers.UserInfo, tokenResponse *providers.TokenResponse, request *CallbackRequest) (*UserSession, error) {
	// Assign roles based on provider claims.

	if sm.rbacManager != nil {
		if err := sm.rbacManager.AssignRolesFromClaims(ctx, userInfo); err != nil {
			sm.logger.Warn("Failed to assign roles from claims",

				"user_id", userInfo.Subject,

				"error", err)
		}
	}

	// Get user roles and permissions.

	var roles, permissions []string

	if sm.rbacManager != nil {

		roles = sm.rbacManager.GetUserRoles(ctx, userInfo.Subject)

		permissions = sm.rbacManager.GetUserPermissions(ctx, userInfo.Subject)

	}

	// Generate session ID and CSRF token.

	sessionID := sm.generateSessionID()

	csrfToken := sm.generateCSRFToken()

	// Create session.

	session := &UserSession{
		ID: sessionID,

		UserID: userInfo.Subject,

		UserInfo: userInfo,

		Provider: userInfo.Provider,

		AccessToken: tokenResponse.AccessToken,

		RefreshToken: tokenResponse.RefreshToken,

		IDToken: tokenResponse.IDToken,

		CreatedAt: time.Now(),

		LastActivity: time.Now(),

		ExpiresAt: time.Now().Add(sm.config.SessionTimeout),

		IPAddress: request.IPAddress,

		UserAgent: request.UserAgent,

		Roles: roles,

		Permissions: permissions,

		Attributes: userInfo.Attributes,

		SSOEnabled: sm.config.EnableSSO,

		LinkedSessions: make(map[string]string),

		CSRFToken: csrfToken,

		SecureContext: sm.config.RequireHTTPS,
	}

	// Check session limits.

	if err := sm.enforceSessionLimits(userInfo.Subject); err != nil {
		return nil, fmt.Errorf("session limit exceeded: %w", err)
	}

	// Store session.

	sm.sessions[sessionID] = session

	return session, nil
}

// GetSession retrieves a session by ID.

func (sm *SessionManager) GetSession(ctx context.Context, sessionID string) (*UserSession, error) {
	sm.mutex.RLock()

	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]

	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Check expiration.

	if time.Now().After(session.ExpiresAt) {

		sm.mutex.RUnlock()

		sm.mutex.Lock()

		delete(sm.sessions, sessionID)

		sm.mutex.Unlock()

		sm.mutex.RLock()

		return nil, fmt.Errorf("session expired")

	}

	// Update last activity.

	session.LastActivity = time.Now()

	return session, nil
}

// RefreshSession refreshes session tokens if needed.

func (sm *SessionManager) RefreshSession(ctx context.Context, sessionID string) error {
	sm.mutex.Lock()

	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]

	if !exists {
		return fmt.Errorf("session not found")
	}

	// Check if refresh is needed.

	if time.Until(session.ExpiresAt) > sm.config.RefreshThreshold {
		return nil // No refresh needed
	}

	// Get provider.

	provider, exists := sm.providers[session.Provider]

	if !exists {
		return fmt.Errorf("provider not available for refresh")
	}

	// Refresh token.

	tokenResponse, err := provider.RefreshToken(ctx, session.RefreshToken)
	if err != nil {

		sm.logger.Error("Token refresh failed",

			"session_id", sessionID,

			"user_id", session.UserID,

			"provider", session.Provider,

			"error", err)

		return fmt.Errorf("token refresh failed: %w", err)

	}

	// Update session.

	session.AccessToken = tokenResponse.AccessToken

	if tokenResponse.RefreshToken != "" {
		session.RefreshToken = tokenResponse.RefreshToken
	}

	if tokenResponse.IDToken != "" {
		session.IDToken = tokenResponse.IDToken
	}

	session.ExpiresAt = time.Now().Add(sm.config.SessionTimeout)

	session.LastActivity = time.Now()

	sm.logger.Info("Session refreshed",

		"session_id", sessionID,

		"user_id", session.UserID,

		"provider", session.Provider)

	return nil
}

// ValidateSession validates a session and returns user info.

func (sm *SessionManager) ValidateSession(ctx context.Context, sessionID string) (*SessionInfo, error) {
	session, err := sm.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Try to refresh if near expiration.

	if time.Until(session.ExpiresAt) < sm.config.RefreshThreshold {
		if refreshErr := sm.RefreshSession(ctx, sessionID); refreshErr != nil {
			sm.logger.Warn("Session refresh failed during validation",

				"session_id", sessionID,

				"error", refreshErr)
		}
	}

	return &SessionInfo{
		ID: session.ID,

		UserID: session.UserID,

		Provider: session.Provider,

		CreatedAt: session.CreatedAt,

		LastActivity: session.LastActivity,

		ExpiresAt: session.ExpiresAt,

		IPAddress: session.IPAddress,

		UserAgent: session.UserAgent,

		Roles: session.Roles,

		SSOEnabled: session.SSOEnabled,

		UserInfo: session.UserInfo,
	}, nil
}

// RevokeSession revokes a user session.

func (sm *SessionManager) RevokeSession(ctx context.Context, sessionID string) error {
	sm.mutex.Lock()

	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]

	if !exists {
		return fmt.Errorf("session not found")
	}

	// Revoke tokens with provider if supported.

	if provider, exists := sm.providers[session.Provider]; exists {
		if provider.SupportsFeature(providers.FeatureTokenRevocation) {
			if err := provider.RevokeToken(ctx, session.AccessToken); err != nil {
				sm.logger.Warn("Failed to revoke token with provider",

					"session_id", sessionID,

					"provider", session.Provider,

					"error", err)
			}
		}
	}

	// Revoke JWT tokens.

	if sm.jwtManager != nil {
		if err := sm.jwtManager.RevokeUserTokens(ctx, session.UserID); err != nil {
			sm.logger.Warn("Failed to revoke JWT tokens",

				"session_id", sessionID,

				"user_id", session.UserID,

				"error", err)
		}
	}

	// Remove session.

	delete(sm.sessions, sessionID)

	sm.logger.Info("Session revoked",

		"session_id", sessionID,

		"user_id", session.UserID,

		"provider", session.Provider)

	return nil
}

// RevokeUserSessions revokes all sessions for a user.

func (sm *SessionManager) RevokeUserSessions(ctx context.Context, userID string) error {
	sm.mutex.Lock()

	defer sm.mutex.Unlock()

	var sessionIDs []string

	for id, session := range sm.sessions {
		if session.UserID == userID {
			sessionIDs = append(sessionIDs, id)
		}
	}

	for _, sessionID := range sessionIDs {

		session := sm.sessions[sessionID]

		// Revoke tokens with provider.

		if provider, exists := sm.providers[session.Provider]; exists {
			if provider.SupportsFeature(providers.FeatureTokenRevocation) {
				provider.RevokeToken(ctx, session.AccessToken)
			}
		}

		delete(sm.sessions, sessionID)

	}

	// Revoke JWT tokens.

	if sm.jwtManager != nil {
		sm.jwtManager.RevokeUserTokens(ctx, userID)
	}

	sm.logger.Info("All user sessions revoked",

		"user_id", userID,

		"session_count", len(sessionIDs))

	return nil
}

// ListUserSessions returns all active sessions for a user.

func (sm *SessionManager) ListUserSessions(ctx context.Context, userID string) ([]*SessionInfo, error) {
	sm.mutex.RLock()

	defer sm.mutex.RUnlock()

	var sessions []*SessionInfo

	now := time.Now()

	for _, session := range sm.sessions {
		if session.UserID == userID && now.Before(session.ExpiresAt) {
			sessions = append(sessions, &SessionInfo{
				ID: session.ID,

				UserID: session.UserID,

				Provider: session.Provider,

				CreatedAt: session.CreatedAt,

				LastActivity: session.LastActivity,

				ExpiresAt: session.ExpiresAt,

				IPAddress: session.IPAddress,

				UserAgent: session.UserAgent,

				Roles: session.Roles,

				SSOEnabled: session.SSOEnabled,
			})
		}
	}

	return sessions, nil
}

// GetSessionMetrics returns session statistics.

func (sm *SessionManager) GetSessionMetrics(ctx context.Context) map[string]interface{} {
	sm.mutex.RLock()

	defer sm.mutex.RUnlock()

	now := time.Now()

	activeSessions := 0

	expiredSessions := 0

	providerCounts := make(map[string]int)

	for _, session := range sm.sessions {
		if now.Before(session.ExpiresAt) {

			activeSessions++

			providerCounts[session.Provider]++

		} else {
			expiredSessions++
		}
	}

	return map[string]interface{}{
		"active_sessions": activeSessions,

		"expired_sessions": expiredSessions,

		"total_sessions": len(sm.sessions),

		"provider_counts": providerCounts,

		"active_states": len(sm.stateStore),

		"registered_providers": len(sm.providers),

		"sso_enabled": sm.config.EnableSSO,

		"session_timeout": sm.config.SessionTimeout,
	}
}

// Helper methods.

func (sm *SessionManager) generateSessionID() string {
	bytes := make([]byte, 32)

	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based ID if random generation fails.

		return fmt.Sprintf("%x", time.Now().UnixNano())
	}

	return hex.EncodeToString(bytes)
}

func (sm *SessionManager) generateState() string {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based state if random generation fails.

		return fmt.Sprintf("%x", time.Now().UnixNano())
	}

	return hex.EncodeToString(bytes)
}

func (sm *SessionManager) generateCSRFToken() string {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based token if random generation fails.

		return fmt.Sprintf("%x", time.Now().UnixNano())
	}

	return hex.EncodeToString(bytes)
}

func (sm *SessionManager) enforceSessionLimits(userID string) error {
	if sm.config.MaxSessions <= 0 {
		return nil
	}

	userSessionCount := 0

	now := time.Now()

	for _, session := range sm.sessions {
		if session.UserID == userID && now.Before(session.ExpiresAt) {
			userSessionCount++
		}
	}

	if userSessionCount >= sm.config.MaxSessions {
		return fmt.Errorf("maximum sessions (%d) exceeded for user", sm.config.MaxSessions)
	}

	return nil
}

func (sm *SessionManager) cleanupLoop() {
	ticker := time.NewTicker(sm.config.CleanupInterval)

	defer ticker.Stop()

	for range ticker.C {

		sm.cleanupExpiredSessions()

		sm.cleanupExpiredStates()

	}
}

func (sm *SessionManager) cleanupExpiredSessions() {
	sm.mutex.Lock()

	defer sm.mutex.Unlock()

	now := time.Now()

	expiredCount := 0

	for sessionID, session := range sm.sessions {
		if now.After(session.ExpiresAt) {

			delete(sm.sessions, sessionID)

			expiredCount++

		}
	}

	if expiredCount > 0 {
		sm.logger.Info("Cleaned up expired sessions", "count", expiredCount)
	}
}

func (sm *SessionManager) cleanupExpiredStates() {
	sm.mutex.Lock()

	defer sm.mutex.Unlock()

	now := time.Now()

	expiredCount := 0

	for state, authState := range sm.stateStore {
		if now.After(authState.ExpiresAt) {

			delete(sm.stateStore, state)

			expiredCount++

		}
	}

	if expiredCount > 0 {
		sm.logger.Info("Cleaned up expired auth states", "count", expiredCount)
	}
}

// SetSessionCookie sets session cookie on HTTP response.

func (sm *SessionManager) SetSessionCookie(w http.ResponseWriter, sessionID string) {
	cookie := &http.Cookie{
		Name: "nephoran_session",

		Value: sessionID,

		Path: sm.config.CookiePath,

		Domain: sm.config.CookieDomain,

		Secure: sm.config.SecureCookies,

		HttpOnly: true,

		MaxAge: int(sm.config.SessionTimeout.Seconds()),
	}

	switch sm.config.SameSiteCookies {

	case "Strict":

		cookie.SameSite = http.SameSiteStrictMode

	case "Lax":

		cookie.SameSite = http.SameSiteLaxMode

	case "None":

		cookie.SameSite = http.SameSiteNoneMode

	}

	http.SetCookie(w, cookie)
}

// ClearSessionCookie clears session cookie.

func (sm *SessionManager) ClearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name: "nephoran_session",

		Value: "",

		Path: sm.config.CookiePath,

		Domain: sm.config.CookieDomain,

		Expires: time.Unix(0, 0),

		MaxAge: -1,

		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
}

// SessionData represents data used to create a new session.

type SessionData struct {
	UserID string `json:"user_id"`

	Username string `json:"username"`

	Email string `json:"email"`

	DisplayName string `json:"display_name"`

	Provider string `json:"provider"`

	Groups []string `json:"groups"`

	Roles []string `json:"roles"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// CreateSession creates a new user session from session data.

func (sm *SessionManager) CreateSession(ctx context.Context, data *SessionData) (*UserSession, error) {
	sessionID := sm.generateSessionID()

	now := time.Now()

	session := &UserSession{
		ID: sessionID,

		UserID: data.UserID,

		UserInfo: &providers.UserInfo{
			Username: data.Username,

			Email: data.Email,

			Name: data.DisplayName,

			Groups: data.Groups,

			Roles: data.Roles,
		},

		Provider: data.Provider,

		CreatedAt: now,

		LastActivity: now,

		ExpiresAt: now.Add(sm.config.SessionTimeout),

		Roles: data.Roles,
	}

	sm.mutex.Lock()

	sm.sessions[sessionID] = session

	sm.mutex.Unlock()

	return session, nil
}

// InvalidateSession invalidates a session by session ID.

func (sm *SessionManager) InvalidateSession(ctx context.Context, sessionID string) error {
	sm.mutex.Lock()

	defer sm.mutex.Unlock()

	if _, exists := sm.sessions[sessionID]; !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	delete(sm.sessions, sessionID)

	return nil
}
