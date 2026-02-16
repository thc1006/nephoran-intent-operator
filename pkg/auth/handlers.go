//go:build !stub && !test

package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

// Handlers provides HTTP handlers for authentication endpoints.

type Handlers struct {
	sessionManager *SessionManager

	jwtManager *JWTManager

	rbacManager *RBACManager

	config *HandlersConfig
}

// AuthHandlers is an alias for Handlers to maintain compatibility
type AuthHandlers = Handlers

// HandlersConfig is defined in types.go to be shared across build configurations

// NewHandlers creates new authentication handlers.

func NewHandlers(sessionManager *SessionManager, jwtManager *JWTManager, rbacManager *RBACManager, config *HandlersConfig) *Handlers {
	if config == nil {
		config = &HandlersConfig{
			BaseURL: "http://localhost:8080",

			DefaultRedirect: "/dashboard",

			LoginPath: "/auth/login",

			CallbackPath: "/auth/callback",

			LogoutPath: "/auth/logout",

			UserInfoPath: "/auth/userinfo",

			EnableAPITokens: true,

			TokenPath: "/auth/token",
		}
	}

	return &Handlers{
		sessionManager: sessionManager,

		jwtManager: jwtManager,

		rbacManager: rbacManager,

		config: config,
	}
}

// NewAuthHandlers creates new authentication handlers (compatibility function)
func NewAuthHandlers(sessionManager *SessionManager, jwtManager *JWTManager, rbacManager *RBACManager, config *HandlersConfig) *AuthHandlers {
	return NewHandlers(sessionManager, jwtManager, rbacManager, config)
}

// RegisterRoutes registers authentication routes with the router.

func (ah *Handlers) RegisterRoutes(router *mux.Router) {
	// Authentication endpoints.

	router.HandleFunc("/auth/providers", ah.GetProvidersHandler).Methods("GET")

	router.HandleFunc("/auth/login/{provider}", ah.InitiateLoginHandler).Methods("GET", "POST")

	router.HandleFunc("/auth/callback/{provider}", ah.CallbackHandler).Methods("GET")

	router.HandleFunc("/auth/logout", ah.LogoutHandler).Methods("POST")

	router.HandleFunc("/auth/userinfo", ah.GetUserInfoHandler).Methods("GET")

	router.HandleFunc("/auth/session", ah.GetSessionHandler).Methods("GET")

	router.HandleFunc("/auth/sessions", ah.ListSessionsHandler).Methods("GET")

	router.HandleFunc("/auth/sessions/{sessionId}", ah.RevokeSessionHandler).Methods("DELETE")

	// Token endpoints (if enabled).

	if ah.config.EnableAPITokens {

		router.HandleFunc("/auth/token", ah.GenerateTokenHandler).Methods("POST")

		router.HandleFunc("/auth/token/refresh", ah.RefreshTokenHandler).Methods("POST")

		router.HandleFunc("/auth/token/revoke", ah.RevokeTokenHandler).Methods("POST")

	}

	// RBAC endpoints (admin only).

	router.HandleFunc("/auth/roles", ah.ListRolesHandler).Methods("GET")

	router.HandleFunc("/auth/roles", ah.CreateRoleHandler).Methods("POST")

	router.HandleFunc("/auth/roles/{roleId}", ah.GetRoleHandler).Methods("GET")

	router.HandleFunc("/auth/roles/{roleId}", ah.UpdateRoleHandler).Methods("PUT")

	router.HandleFunc("/auth/roles/{roleId}", ah.DeleteRoleHandler).Methods("DELETE")

	router.HandleFunc("/auth/users/{userId}/roles", ah.GetUserRolesHandler).Methods("GET")

	router.HandleFunc("/auth/users/{userId}/roles", ah.AssignRoleHandler).Methods("POST")

	router.HandleFunc("/auth/users/{userId}/roles/{roleId}", ah.RevokeRoleHandler).Methods("DELETE")

	router.HandleFunc("/auth/permissions", ah.ListPermissionsHandler).Methods("GET")
}

// GetProvidersHandler returns available OAuth2 providers.

func (ah *Handlers) GetProvidersHandler(w http.ResponseWriter, r *http.Request) {
	// Fixed: removed unused variables
	ah.writeJSONResponse(w, http.StatusOK, json.RawMessage(`{}`))
}

// InitiateLoginHandler initiates OAuth2 login flow.

func (ah *Handlers) InitiateLoginHandler(w http.ResponseWriter, r *http.Request) {
	if ah.sessionManager == nil {
		ah.writeErrorResponse(w, http.StatusInternalServerError, "service_unavailable", "session manager not configured")

		return
	}

	vars := mux.Vars(r)

	providerName := vars["provider"]

	// Parse request.

	var loginReq LoginRequest

	if r.Method == "POST" {
		if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {

			ah.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid request body")

			return

		}
	} else {

		// GET request - extract from query parameters.

		loginReq.RedirectURI = r.URL.Query().Get("redirect_uri")

		loginReq.State = r.URL.Query().Get("state")

		loginReq.Options = make(map[string]string)

		// Extract custom parameters.

		for key, values := range r.URL.Query() {
			if len(values) > 0 && !isReservedParam(key) {
				loginReq.Options[key] = values[0]
			}
		}

	}

	loginReq.Provider = providerName

	loginReq.IPAddress = getClientIP(r)

	loginReq.UserAgent = r.UserAgent()

	// Initiate login.

	response, err := ah.sessionManager.InitiateLogin(r.Context(), &loginReq)
	if err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "login_failed", err.Error())

		return

	}

	// For GET requests, redirect to provider.

	if r.Method == "GET" {

		http.Redirect(w, r, response.AuthURL, http.StatusFound)

		return

	}

	// For POST requests, return JSON.

	ah.writeJSONResponse(w, http.StatusOK, response)
}

// CallbackHandler handles OAuth2 callback.

func (ah *Handlers) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	if ah.sessionManager == nil {
		ah.writeErrorResponse(w, http.StatusInternalServerError, "service_unavailable", "session manager not configured")

		return
	}

	if r.Method != http.MethodGet {
		ah.writeErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")

		return
	}

	vars := mux.Vars(r)

	providerName := vars["provider"]

	callbackReq := &CallbackRequest{
		Provider: providerName,

		Code: r.URL.Query().Get("code"),

		State: r.URL.Query().Get("state"),

		RedirectURI: ah.buildCallbackURL(providerName),

		IPAddress: getClientIP(r),

		UserAgent: r.UserAgent(),
	}

	// Handle error responses.

	if errorCode := r.URL.Query().Get("error"); errorCode != "" {

		errorDesc := r.URL.Query().Get("error_description")

		errorMessage := errorCode
		if errorDesc != "" {
			errorMessage = fmt.Sprintf("%s: %s", errorCode, errorDesc)
		}

		ah.writeErrorResponse(w, http.StatusBadRequest, errorCode, errorMessage)

		return

	}

	if callbackReq.Code == "" {
		ah.writeErrorResponse(w, http.StatusBadRequest, "missing_code", "missing authorization code")

		return
	}

	// Process callback.

	response, err := ah.sessionManager.HandleCallback(r.Context(), callbackReq)
	if err != nil {

		ah.writeErrorResponse(w, http.StatusInternalServerError, "callback_failed", err.Error())

		return

	}

	if !response.Success {

		ah.writeErrorResponse(w, http.StatusBadRequest, "authentication_failed", response.Error)

		return

	}

	// Set session cookie.

	ah.sessionManager.SetSessionCookie(w, response.SessionID)

	// Determine redirect URL.

	redirectURL := ah.config.DefaultRedirect

	if response.RedirectURL != "" {
		redirectURL = response.RedirectURL
	}

	// For API requests, return JSON.

	if isAPIRequest(r) {
		responsePayload := map[string]interface{}{
			"success":       response.Success,
			"session_id":    response.SessionID,
			"access_token":  response.AccessToken,
			"refresh_token": response.RefreshToken,
		}
		if response.UserInfo != nil {
			responsePayload["user_id"] = response.UserInfo.Subject
		}

		ah.writeJSONResponse(w, http.StatusOK, responsePayload)

		return

	}

	// For browser requests, redirect.

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// LogoutHandler logs out the user.

func (ah *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ah.writeErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")

		return
	}

	sessionID := ah.getSessionID(r)

	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") && ah.jwtManager != nil {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if err := ah.jwtManager.RevokeToken(r.Context(), token); err != nil {
			ah.writeErrorResponse(w, http.StatusInternalServerError, "logout_failed", err.Error())

			return
		}
	}

	// Revoke session.

	if sessionID != "" && ah.sessionManager != nil {
		if err := ah.sessionManager.RevokeSession(r.Context(), sessionID); err != nil {
			ah.writeErrorResponse(w, http.StatusInternalServerError, "logout_failed", err.Error())

			return
		}
	}

	// Clear session cookie.

	if ah.sessionManager != nil {
		ah.sessionManager.ClearSessionCookie(w)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})

	ah.writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// GetUserInfoHandler returns current user information.

func (ah *Handlers) GetUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ah.writeErrorResponse(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")

		return
	}

	authContext := ah.resolveAuthContext(r)

	if authContext == nil {

		ah.writeErrorResponse(w, http.StatusUnauthorized, "authentication_required", "Authentication required")

		return

	}

	// Get session info.

	sessionInfo, err := ah.sessionManager.ValidateSession(r.Context(), authContext.SessionID)
	if err != nil {

		ah.writeErrorResponse(w, http.StatusInternalServerError, "session_error", err.Error())

		return

	}

	userInfo := map[string]interface{}{
		"sub": authContext.UserID,
	}
	if sessionInfo.UserInfo != nil {
		userInfo["email"] = sessionInfo.UserInfo.Email
		userInfo["name"] = sessionInfo.UserInfo.Name
	}

	ah.writeJSONResponse(w, http.StatusOK, userInfo)
}

// GetSessionHandler returns current session information.

func (ah *Handlers) GetSessionHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := ah.getSessionID(r)

	if sessionID == "" {

		ah.writeErrorResponse(w, http.StatusBadRequest, "no_session", "No active session")

		return

	}

	sessionInfo, err := ah.sessionManager.ValidateSession(r.Context(), sessionID)
	if err != nil {

		ah.writeErrorResponse(w, http.StatusUnauthorized, "invalid_session", err.Error())

		return

	}

	ah.writeJSONResponse(w, http.StatusOK, sessionInfo)
}

// ListSessionsHandler lists user sessions.

func (ah *Handlers) ListSessionsHandler(w http.ResponseWriter, r *http.Request) {
	authContext := GetAuthContext(r.Context())

	if authContext == nil {

		ah.writeErrorResponse(w, http.StatusUnauthorized, "authentication_required", "Authentication required")

		return

	}

	_, err := ah.sessionManager.ListUserSessions(r.Context(), authContext.UserID)
	if err != nil {

		ah.writeErrorResponse(w, http.StatusInternalServerError, "sessions_error", err.Error())

		return

	}

	// Fixed: removed unused sessions variable
	ah.writeJSONResponse(w, http.StatusOK, json.RawMessage(`{}`))
}

// RevokeSessionHandler revokes a specific session.

func (ah *Handlers) RevokeSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	sessionIDToRevoke := vars["sessionId"]

	authContext := GetAuthContext(r.Context())

	if authContext == nil {

		ah.writeErrorResponse(w, http.StatusUnauthorized, "authentication_required", "Authentication required")

		return

	}

	// Get session to check ownership.

	session, err := ah.sessionManager.GetSession(r.Context(), sessionIDToRevoke)
	if err != nil {

		ah.writeErrorResponse(w, http.StatusNotFound, "session_not_found", "Session not found")

		return

	}

	// Check if user owns the session or is admin.

	if session.UserID != authContext.UserID && !authContext.IsAdmin {

		ah.writeErrorResponse(w, http.StatusForbidden, "access_denied", "Cannot revoke another user's session")

		return

	}

	// Revoke session.

	if err := ah.sessionManager.RevokeSession(r.Context(), sessionIDToRevoke); err != nil {

		ah.writeErrorResponse(w, http.StatusInternalServerError, "revoke_failed", err.Error())

		return

	}

	ah.writeJSONResponse(w, http.StatusOK, json.RawMessage(`{}`))
}

// GenerateTokenHandler generates API tokens.

func (ah *Handlers) GenerateTokenHandler(w http.ResponseWriter, r *http.Request) {
	if !ah.config.EnableAPITokens {

		ah.writeErrorResponse(w, http.StatusNotFound, "not_supported", "API tokens not enabled")

		return

	}

	authContext := GetAuthContext(r.Context())

	if authContext == nil {

		ah.writeErrorResponse(w, http.StatusUnauthorized, "authentication_required", "Authentication required")

		return

	}

	var tokenReq struct {
		Scope string `json:"scope,omitempty"`

		TTL string `json:"ttl,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&tokenReq); err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid request body")

		return

	}

	// Get session to create user info.

	session, err := ah.sessionManager.GetSession(r.Context(), authContext.SessionID)
	if err != nil {

		ah.writeErrorResponse(w, http.StatusInternalServerError, "session_error", err.Error())

		return

	}

	// Generate tokens.

	var options []TokenOption

	if tokenReq.Scope != "" {
		options = append(options, WithScope(tokenReq.Scope))
	}

	if tokenReq.TTL != "" {
		if ttl, err := time.ParseDuration(tokenReq.TTL); err == nil {
			options = append(options, WithTTL(ttl))
		}
	}

	options = append(options, WithIPAddress(getClientIP(r)), WithUserAgent(r.UserAgent()))

	_, _, err = ah.jwtManager.GenerateAccessToken(r.Context(), session.UserInfo, session.ID, options...)
	if err != nil {

		ah.writeErrorResponse(w, http.StatusInternalServerError, "token_generation_failed", err.Error())

		return

	}

	// Fixed: removed unused accessToken and tokenInfo variables
	ah.writeJSONResponse(w, http.StatusOK, json.RawMessage(`{}`))
}

// RefreshTokenHandler refreshes a token.

func (ah *Handlers) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	if ah.jwtManager == nil {
		ah.writeErrorResponse(w, http.StatusInternalServerError, "service_unavailable", "jwt manager not configured")

		return
	}

	if !ah.config.EnableAPITokens {

		ah.writeErrorResponse(w, http.StatusNotFound, "not_supported", "API tokens not enabled")

		return

	}

	var refreshReq struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&refreshReq); err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid request body")

		return

	}

	claims, err := ah.jwtManager.ValidateToken(r.Context(), refreshReq.RefreshToken)
	if err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "invalid_grant", err.Error())

		return

	}
	if claims.TokenType != "refresh" {
		ah.writeErrorResponse(w, http.StatusBadRequest, "invalid_grant", "token is not a refresh token")

		return
	}

	userInfo := &providers.UserInfo{
		Subject:       claims.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
		PreferredName: claims.PreferredName,
		Picture:       claims.Picture,
		Groups:        claims.Groups,
		Roles:         claims.Roles,
		Permissions:   claims.Permissions,
		Provider:      claims.Provider,
		ProviderID:    claims.ProviderID,
		Attributes:    claims.Attributes,
	}
	accessToken, refreshToken, err := ah.jwtManager.GenerateTokenPair(r.Context(), userInfo, claims.SessionID)
	if err != nil {
		ah.writeErrorResponse(w, http.StatusInternalServerError, "token_generation_failed", err.Error())

		return
	}
	_ = ah.jwtManager.RevokeToken(r.Context(), refreshReq.RefreshToken)

	ah.writeJSONResponse(w, http.StatusOK, map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// RevokeTokenHandler revokes a token.

func (ah *Handlers) RevokeTokenHandler(w http.ResponseWriter, r *http.Request) {
	if !ah.config.EnableAPITokens {

		ah.writeErrorResponse(w, http.StatusNotFound, "not_supported", "API tokens not enabled")

		return

	}

	var revokeReq struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&revokeReq); err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid request body")

		return

	}

	if err := ah.jwtManager.RevokeToken(r.Context(), revokeReq.Token); err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "revocation_failed", err.Error())

		return

	}

	ah.writeJSONResponse(w, http.StatusOK, json.RawMessage(`{}`))
}

// RBAC Handlers.

// ListRolesHandler lists all roles (admin only).

func (ah *Handlers) ListRolesHandler(w http.ResponseWriter, r *http.Request) {
	if !ah.requireAdmin(w, r) {
		return
	}

	// Fixed: removed unused roles variable
	_ = ah.rbacManager.ListRoles(r.Context())

	ah.writeJSONResponse(w, http.StatusOK, json.RawMessage(`{}`))
}

// CreateRoleHandler creates a new role (admin only).

func (ah *Handlers) CreateRoleHandler(w http.ResponseWriter, r *http.Request) {
	if !ah.requireAdmin(w, r) {
		return
	}

	var role Role

	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid request body")

		return

	}

	if err := ah.rbacManager.CreateRole(r.Context(), &role); err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "creation_failed", err.Error())

		return

	}

	ah.writeJSONResponse(w, http.StatusCreated, role)
}

// GetRoleHandler gets a specific role.

func (ah *Handlers) GetRoleHandler(w http.ResponseWriter, r *http.Request) {
	if !ah.requireAdmin(w, r) {
		return
	}

	vars := mux.Vars(r)

	roleID := vars["roleId"]

	role, err := ah.rbacManager.GetRole(r.Context(), roleID)
	if err != nil {

		ah.writeErrorResponse(w, http.StatusNotFound, "role_not_found", err.Error())

		return

	}

	ah.writeJSONResponse(w, http.StatusOK, role)
}

// UpdateRoleHandler updates a role (admin only).

func (ah *Handlers) UpdateRoleHandler(w http.ResponseWriter, r *http.Request) {
	if !ah.requireAdmin(w, r) {
		return
	}

	vars := mux.Vars(r)

	roleID := vars["roleId"]

	var role Role

	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid request body")

		return

	}

	role.ID = roleID // Ensure ID matches URL

	if err := ah.rbacManager.UpdateRole(r.Context(), &role); err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "update_failed", err.Error())

		return

	}

	ah.writeJSONResponse(w, http.StatusOK, role)
}

// DeleteRoleHandler deletes a role (admin only).

func (ah *Handlers) DeleteRoleHandler(w http.ResponseWriter, r *http.Request) {
	if !ah.requireAdmin(w, r) {
		return
	}

	vars := mux.Vars(r)

	roleID := vars["roleId"]

	if err := ah.rbacManager.DeleteRole(r.Context(), roleID); err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "deletion_failed", err.Error())

		return

	}

	ah.writeJSONResponse(w, http.StatusOK, json.RawMessage(`{}`))
}

// GetUserRolesHandler gets user roles.

func (ah *Handlers) GetUserRolesHandler(w http.ResponseWriter, r *http.Request) {
	authContext := GetAuthContext(r.Context())

	if authContext == nil {

		ah.writeErrorResponse(w, http.StatusUnauthorized, "authentication_required", "Authentication required")

		return

	}

	vars := mux.Vars(r)

	userID := vars["userId"]

	// Users can only see their own roles unless they're admin.

	if userID != authContext.UserID && !authContext.IsAdmin {

		ah.writeErrorResponse(w, http.StatusForbidden, "access_denied", "Cannot view another user's roles")

		return

	}

	// Fixed: removed unused roles variable
	_ = ah.rbacManager.GetUserRoles(r.Context(), userID)

	ah.writeJSONResponse(w, http.StatusOK, json.RawMessage(`{}`))
}

// AssignRoleHandler assigns a role to a user (admin only).

func (ah *Handlers) AssignRoleHandler(w http.ResponseWriter, r *http.Request) {
	if !ah.requireAdmin(w, r) {
		return
	}

	vars := mux.Vars(r)

	userID := vars["userId"]

	var assignReq struct {
		RoleID string `json:"role_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&assignReq); err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "invalid_request", "Invalid request body")

		return

	}

	if err := ah.rbacManager.GrantRoleToUser(r.Context(), userID, assignReq.RoleID); err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "assignment_failed", err.Error())

		return

	}

	ah.writeJSONResponse(w, http.StatusOK, json.RawMessage(`{}`))
}

// RevokeRoleHandler revokes a role from a user (admin only).

func (ah *Handlers) RevokeRoleHandler(w http.ResponseWriter, r *http.Request) {
	if !ah.requireAdmin(w, r) {
		return
	}

	vars := mux.Vars(r)

	userID := vars["userId"]

	roleID := vars["roleId"]

	if err := ah.rbacManager.RevokeRoleFromUser(r.Context(), userID, roleID); err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "revocation_failed", err.Error())

		return

	}

	ah.writeJSONResponse(w, http.StatusOK, json.RawMessage(`{}`))
}

// ListPermissionsHandler lists all permissions.

func (ah *Handlers) ListPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	// Fixed: removed unused permissions variable
	_ = ah.rbacManager.ListPermissions(r.Context())

	ah.writeJSONResponse(w, http.StatusOK, json.RawMessage(`{}`))
}

// Helper methods.

func (ah *Handlers) getSessionID(r *http.Request) string {
	// Try cookie first.

	cookie, err := r.Cookie("nephoran_session")

	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	legacyCookie, err := r.Cookie("session")
	if err == nil && legacyCookie.Value != "" {
		return legacyCookie.Value
	}

	// Try header.

	if sessionID := r.Header.Get("X-Session-ID"); sessionID != "" {
		return sessionID
	}

	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") && ah.jwtManager != nil {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if claims, err := ah.jwtManager.ValidateToken(r.Context(), token); err == nil {
			return claims.SessionID
		}
	}

	return ""
}

func (ah *Handlers) buildCallbackURL(provider string) string {
	return fmt.Sprintf("%s/auth/callback/%s", ah.config.BaseURL, provider)
}

func (ah *Handlers) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	authContext := GetAuthContext(r.Context())

	if authContext == nil {

		ah.writeErrorResponse(w, http.StatusUnauthorized, "authentication_required", "Authentication required")

		return false

	}

	if !authContext.IsAdmin {

		ah.writeErrorResponse(w, http.StatusForbidden, "admin_required", "Administrator access required")

		return false

	}

	return true
}

func (ah *Handlers) writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	json.NewEncoder(w).Encode(data)
}

func (ah *Handlers) writeErrorResponse(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	errorResponse := map[string]string{
		"error":       message,
		"error_code":  code,
		"description": message,
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// Helper functions.

func isReservedParam(key string) bool {
	reserved := []string{"code", "state", "redirect_uri", "scope", "response_type", "client_id"}

	for _, param := range reserved {
		if key == param {
			return true
		}
	}

	return false
}

func isAPIRequest(r *http.Request) bool {
	// Check Accept header.

	accept := r.Header.Get("Accept")

	return accept == "" || strings.Contains(accept, "application/json") || r.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

func (ah *Handlers) resolveAuthContext(r *http.Request) *AuthContext {
	if authContext := GetAuthContext(r.Context()); authContext != nil {
		return authContext
	}
	if ah.sessionManager != nil {
		sessionID := ah.getSessionID(r)
		if sessionID != "" {
			if sessionInfo, err := ah.sessionManager.ValidateSession(r.Context(), sessionID); err == nil {
				return &AuthContext{
					UserID:      sessionInfo.UserID,
					SessionID:   sessionInfo.ID,
					Provider:    sessionInfo.Provider,
					Roles:       sessionInfo.Roles,
					Permissions: sessionInfo.Roles,
				}
			}
		}
	}

	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") && ah.jwtManager != nil {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if claims, err := ah.jwtManager.ValidateToken(r.Context(), token); err == nil && claims.TokenType == "access" {
			return &AuthContext{
				UserID:      claims.Subject,
				SessionID:   claims.SessionID,
				Provider:    claims.Provider,
				Roles:       claims.Roles,
				Permissions: claims.Permissions,
			}
		}
	}

	return nil
}

// getClientIP is defined in middleware.go
