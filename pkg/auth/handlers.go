//go:build !stub && !test

package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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
	providers := make(map[string]interface{})

	for name, provider := range ah.sessionManager.providers {

		config := provider.GetConfiguration()

		providers[name] = map[string]interface{}{
			"name": config.Name,

			"type": config.Type,

			"features": config.Features,
		}

	}

	ah.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"providers": providers,
	})
}

// InitiateLoginHandler initiates OAuth2 login flow.

func (ah *Handlers) InitiateLoginHandler(w http.ResponseWriter, r *http.Request) {
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

		ah.writeErrorResponse(w, http.StatusBadRequest, errorCode, errorDesc)

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

		ah.writeJSONResponse(w, http.StatusOK, response)

		return

	}

	// For browser requests, redirect.

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// LogoutHandler logs out the user.

func (ah *Handlers) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := ah.getSessionID(r)

	if sessionID == "" {

		ah.writeErrorResponse(w, http.StatusBadRequest, "no_session", "No active session")

		return

	}

	// Revoke session.

	if err := ah.sessionManager.RevokeSession(r.Context(), sessionID); err != nil {

		ah.writeErrorResponse(w, http.StatusInternalServerError, "logout_failed", err.Error())

		return

	}

	// Clear session cookie.

	ah.sessionManager.ClearSessionCookie(w)

	ah.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Logged out successfully",
	})
}

// GetUserInfoHandler returns current user information.

func (ah *Handlers) GetUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	authContext := GetAuthContext(r.Context())

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
		"user_id": authContext.UserID,

		"session_id": authContext.SessionID,

		"provider": authContext.Provider,

		"roles": authContext.Roles,

		"permissions": authContext.Permissions,

		"is_admin": authContext.IsAdmin,

		"user_info": sessionInfo.UserInfo,

		"created_at": sessionInfo.CreatedAt,

		"last_activity": sessionInfo.LastActivity,

		"expires_at": sessionInfo.ExpiresAt,
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

	sessions, err := ah.sessionManager.ListUserSessions(r.Context(), authContext.UserID)
	if err != nil {

		ah.writeErrorResponse(w, http.StatusInternalServerError, "sessions_error", err.Error())

		return

	}

	ah.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
	})
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

	ah.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Session revoked successfully",
	})
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

	accessToken, tokenInfo, err := ah.jwtManager.GenerateAccessToken(r.Context(), session.UserInfo, session.ID, options...)
	if err != nil {

		ah.writeErrorResponse(w, http.StatusInternalServerError, "token_generation_failed", err.Error())

		return

	}

	ah.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"access_token": accessToken,

		"token_type": "Bearer",

		"expires_at": tokenInfo.ExpiresAt,

		"scope": tokenInfo.Scope,
	})
}

// RefreshTokenHandler refreshes a token.

func (ah *Handlers) RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
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

	accessToken, tokenInfo, err := ah.jwtManager.RefreshAccessToken(r.Context(), refreshReq.RefreshToken)
	if err != nil {

		ah.writeErrorResponse(w, http.StatusBadRequest, "invalid_grant", err.Error())

		return

	}

	ah.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"access_token": accessToken,

		"token_type": "Bearer",

		"expires_at": tokenInfo.ExpiresAt,
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

	ah.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Token revoked successfully",
	})
}

// RBAC Handlers.

// ListRolesHandler lists all roles (admin only).

func (ah *Handlers) ListRolesHandler(w http.ResponseWriter, r *http.Request) {
	if !ah.requireAdmin(w, r) {
		return
	}

	roles := ah.rbacManager.ListRoles(r.Context())

	ah.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"roles": roles,
	})
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

	ah.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Role deleted successfully",
	})
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

	roles := ah.rbacManager.GetUserRoles(r.Context(), userID)

	ah.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"user_id": userID,

		"roles": roles,
	})
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

	ah.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Role assigned successfully",
	})
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

	ah.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Role revoked successfully",
	})
}

// ListPermissionsHandler lists all permissions.

func (ah *Handlers) ListPermissionsHandler(w http.ResponseWriter, r *http.Request) {
	permissions := ah.rbacManager.ListPermissions(r.Context())

	ah.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"permissions": permissions,
	})
}

// Helper methods.

func (ah *Handlers) getSessionID(r *http.Request) string {
	// Try cookie first.

	cookie, err := r.Cookie("nephoran_session")

	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Try header.

	return r.Header.Get("X-Session-ID")
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

	errorResponse := map[string]interface{}{
		"error": code,

		"error_description": message,

		"status": status,

		"timestamp": time.Now().Unix(),
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

	return accept == "application/json" || r.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

// getClientIP is defined in middleware.go
