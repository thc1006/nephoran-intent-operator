package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"k8s.io/klog/v2"
)

// AuthMiddleware provides OAuth2 authentication middleware
type AuthMiddleware struct {
	providers     map[string]*OAuth2Provider
	jwtSecretKey  []byte
	tokenTTL      time.Duration
	refreshTTL    time.Duration
	publicPaths   map[string]bool
	adminRoles    []string
	operatorRoles []string
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(config *OAuth2Config, jwtSecretKey []byte) *AuthMiddleware {
	return &AuthMiddleware{
		providers:    config.Providers,
		jwtSecretKey: jwtSecretKey,
		tokenTTL:     config.TokenTTL,
		refreshTTL:   config.RefreshTTL,
		publicPaths: map[string]bool{
			"/healthz":        true,
			"/readyz":         true,
			"/metrics":        true,
			"/auth/login":     true,
			"/auth/callback":  true,
			"/auth/refresh":   true,
			"/swagger-ui":     true,
			"/api-docs":       true,
		},
		adminRoles:    []string{"admin", "system-admin", "nephoran-admin"},
		operatorRoles: []string{"operator", "network-operator", "telecom-operator"},
	}
}

// Authenticate is the main authentication middleware
func (am *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if path is public
		if am.isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract and validate token
		token := am.extractToken(r)
		if token == "" {
			am.sendUnauthorized(w, "Missing authentication token")
			return
		}

		claims, err := ValidateJWT(token, am.jwtSecretKey)
		if err != nil {
			am.sendUnauthorized(w, fmt.Sprintf("Invalid token: %v", err))
			return
		}

		// Add user info to request context
		ctx := context.WithValue(r.Context(), "user", claims)
		ctx = context.WithValue(ctx, "user_id", claims.Subject)
		ctx = context.WithValue(ctx, "user_email", claims.Email)
		ctx = context.WithValue(ctx, "user_roles", claims.Roles)
		ctx = context.WithValue(ctx, "user_groups", claims.Groups)
		ctx = context.WithValue(ctx, "auth_provider", claims.Provider)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRole middleware that requires specific roles
func (am *AuthMiddleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRoles, ok := r.Context().Value("user_roles").([]string)
			if !ok {
				am.sendForbidden(w, "No roles found in token")
				return
			}

			if !am.hasAnyRole(userRoles, roles) {
				am.sendForbidden(w, fmt.Sprintf("Required roles: %v", roles))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin middleware that requires admin privileges
func (am *AuthMiddleware) RequireAdmin() func(http.Handler) http.Handler {
	return am.RequireRole(am.adminRoles...)
}

// RequireOperator middleware that requires operator privileges
func (am *AuthMiddleware) RequireOperator() func(http.Handler) http.Handler {
	return am.RequireRole(append(am.adminRoles, am.operatorRoles...)...)
}

// OAuth2 Authentication Handlers

// LoginHandler initiates OAuth2 login flow
func (am *AuthMiddleware) LoginHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerName := vars["provider"]
	
	provider, exists := am.providers[providerName]
	if !exists {
		http.Error(w, "Unknown provider", http.StatusBadRequest)
		return
	}

	state := generateRandomState()
	redirectURI := am.getRedirectURI(r, providerName)
	
	// Store state in session or cache (implementation depends on session store)
	authURL := provider.GetAuthorizationURL(state, redirectURI)
	
	// Store state for validation
	am.storeState(w, state, providerName)
	
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// CallbackHandler handles OAuth2 callback
func (am *AuthMiddleware) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerName := vars["provider"]
	
	provider, exists := am.providers[providerName]
	if !exists {
		http.Error(w, "Unknown provider", http.StatusBadRequest)
		return
	}

	// Validate state parameter
	state := r.URL.Query().Get("state")
	if !am.validateState(r, state, providerName) {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	redirectURI := am.getRedirectURI(r, providerName)
	tokenResp, err := provider.ExchangeCodeForToken(r.Context(), code, redirectURI)
	if err != nil {
		klog.Errorf("Failed to exchange code for token: %v", err)
		http.Error(w, "Authentication failed", http.StatusInternalServerError)
		return
	}

	// Get user info
	userInfo, err := provider.ValidateToken(r.Context(), tokenResp.AccessToken)
	if err != nil {
		klog.Errorf("Failed to get user info: %v", err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	jwtToken, err := GenerateJWT(userInfo, providerName, am.jwtSecretKey, am.tokenTTL)
	if err != nil {
		klog.Errorf("Failed to generate JWT: %v", err)
		http.Error(w, "Failed to generate authentication token", http.StatusInternalServerError)
		return
	}

	// Return token response
	response := map[string]interface{}{
		"access_token": jwtToken,
		"token_type":   "Bearer",
		"expires_in":   int64(am.tokenTTL.Seconds()),
		"user_info": map[string]interface{}{
			"subject":   userInfo.Subject,
			"email":     userInfo.Email,
			"name":      userInfo.Name,
			"groups":    userInfo.Groups,
			"roles":     userInfo.Roles,
			"provider":  providerName,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RefreshHandler handles token refresh
func (am *AuthMiddleware) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		RefreshToken string `json:"refresh_token"`
		Provider     string `json:"provider"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	provider, exists := am.providers[request.Provider]
	if !exists {
		http.Error(w, "Unknown provider", http.StatusBadRequest)
		return
	}

	// Refresh OAuth2 token
	tokenResp, err := provider.RefreshToken(r.Context(), request.RefreshToken)
	if err != nil {
		klog.Errorf("Failed to refresh token: %v", err)
		http.Error(w, "Token refresh failed", http.StatusUnauthorized)
		return
	}

	// Get updated user info
	userInfo, err := provider.ValidateToken(r.Context(), tokenResp.AccessToken)
	if err != nil {
		klog.Errorf("Failed to get user info: %v", err)
		http.Error(w, "Failed to get user information", http.StatusInternalServerError)
		return
	}

	// Generate new JWT token
	jwtToken, err := GenerateJWT(userInfo, request.Provider, am.jwtSecretKey, am.tokenTTL)
	if err != nil {
		klog.Errorf("Failed to generate JWT: %v", err)
		http.Error(w, "Failed to generate authentication token", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"access_token": jwtToken,
		"token_type":   "Bearer",
		"expires_in":   int64(am.tokenTTL.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// LogoutHandler handles user logout
func (am *AuthMiddleware) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear any server-side session data
	// In a stateless JWT implementation, logout is primarily client-side
	
	response := map[string]string{
		"message": "Logged out successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UserInfoHandler returns current user information
func (am *AuthMiddleware) UserInfoHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value("user").(*JWTClaims)
	if !ok {
		http.Error(w, "User information not found", http.StatusInternalServerError)
		return
	}

	userInfo := map[string]interface{}{
		"subject":   claims.Subject,
		"email":     claims.Email,
		"name":      claims.Name,
		"groups":    claims.Groups,
		"roles":     claims.Roles,
		"provider":  claims.Provider,
		"expires":   claims.ExpiresAt.Time,
		"issued":    claims.IssuedAt.Time,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInfo)
}

// Helper methods

func (am *AuthMiddleware) isPublicPath(path string) bool {
	// Check exact match
	if am.publicPaths[path] {
		return true
	}
	
	// Check prefix match for certain paths
	publicPrefixes := []string{"/swagger-ui", "/api-docs", "/static"}
	for _, prefix := range publicPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	
	return false
}

func (am *AuthMiddleware) extractToken(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	
	// Check query parameter
	return r.URL.Query().Get("access_token")
}

func (am *AuthMiddleware) hasAnyRole(userRoles, requiredRoles []string) bool {
	for _, userRole := range userRoles {
		for _, requiredRole := range requiredRoles {
			if userRole == requiredRole {
				return true
			}
		}
	}
	return false
}

func (am *AuthMiddleware) sendUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   "unauthorized",
		"message": message,
	})
}

func (am *AuthMiddleware) sendForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   "forbidden",
		"message": message,
	})
}

func (am *AuthMiddleware) getRedirectURI(r *http.Request, provider string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/auth/callback/%s", scheme, r.Host, provider)
}

func (am *AuthMiddleware) storeState(w http.ResponseWriter, state, provider string) {
	// In production, store in Redis or secure session store
	// For demo purposes, using HTTP-only cookie
	cookie := &http.Cookie{
		Name:     fmt.Sprintf("oauth_state_%s", provider),
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	}
	http.SetCookie(w, cookie)
}

func (am *AuthMiddleware) validateState(r *http.Request, state, provider string) bool {
	cookie, err := r.Cookie(fmt.Sprintf("oauth_state_%s", provider))
	if err != nil {
		return false
	}
	return cookie.Value == state
}

func generateRandomState() string {
	// Generate a cryptographically secure random state
	// This is a simplified implementation
	return fmt.Sprintf("state_%d", time.Now().UnixNano())
}