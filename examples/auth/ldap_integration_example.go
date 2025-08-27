package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth/providers"
)

// Example: Complete LDAP Integration with Nephoran Intent Operator
// This example demonstrates how to integrate LDAP authentication with the Nephoran Intent Operator
// including OAuth2, session management, RBAC, and security features.

func main() {
	// Initialize structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load authentication configuration
	authConfig, err := auth.LoadAuthConfig("")
	if err != nil {
		log.Fatalf("Failed to load auth config: %v", err)
	}

	// Convert AuthConfig to AuthManagerConfig
	authManagerConfig := &auth.AuthManagerConfig{
		JWTConfig: &auth.JWTConfig{
			Issuer:        "nephoran-intent-operator",
			SigningKey:    authConfig.JWTSecretKey,
			DefaultTTL:    authConfig.TokenTTL,
			RefreshTTL:    authConfig.RefreshTTL,
			CookieDomain:  "",
			CookiePath:    "/",
		},
		SessionConfig: &auth.SessionConfig{
			SessionTimeout:   24 * time.Hour,
			RefreshThreshold: 15 * time.Minute,
			MaxSessions:      10,
			SecureCookies:    true,
			SameSiteCookies:  "Lax",
		},
		RBACConfig: &auth.RBACManagerConfig{
			CacheTTL:           15 * time.Minute,
			EnableHierarchy:    true,
			DefaultDenyAll:     true,
			PolicyEvaluation:   "deny-overrides",
			MaxPolicyDepth:     10,
			EnableAuditLogging: true,
		},
		OAuth2Config: &auth.OAuth2ManagerConfig{
			Enabled: len(authConfig.Providers) > 0,
		},
		SecurityConfig: &auth.SecurityManagerConfig{
			CSRFSecret: []byte("csrf-secret-key-change-in-production"),
			PKCE: auth.PKCEConfig{
				TTL: 10 * time.Minute,
			},
			CSRF: auth.CSRFConfig{
				Secret: []byte("csrf-secret-key-change-in-production"),
				TTL:    1 * time.Hour,
			},
		},
	}

	// Create authentication manager
	authManager, err := auth.NewAuthManager(authManagerConfig, logger)
	if err != nil {
		log.Fatalf("Failed to create auth manager: %v", err)
	}

	// Create HTTP router
	router := mux.NewRouter()

	// Setup authentication routes and middleware
	setupAuthRoutes(router, authManager)
	setupProtectedRoutes(router, authManager)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Shutdown auth manager
	if err := authManager.Close(); err != nil {
		logger.Error("Error shutting down auth manager", "error", err)
	}

	logger.Info("Server exited")
}

// setupAuthRoutes configures authentication-related routes
func setupAuthRoutes(router *mux.Router, authManager *auth.AuthManager) {
	// Health check endpoint
	router.HandleFunc("/health", authManager.HandleHealthCheck).Methods("GET")

	// Authentication information endpoint
	router.HandleFunc("/auth/info", handleAuthInfo(authManager)).Methods("GET")

	// LDAP authentication endpoints
	if ldapMiddleware := authManager.GetLDAPMiddleware(); ldapMiddleware != nil {
		router.HandleFunc("/auth/ldap/login", ldapMiddleware.HandleLDAPLogin).Methods("POST")
		router.HandleFunc("/auth/ldap/logout", ldapMiddleware.HandleLDAPLogout).Methods("POST")

		// LDAP user info endpoint (for admin purposes)
		router.Handle("/auth/ldap/user/{username}",
			authManager.GetMiddleware().RequirePermissionMiddleware("users:manage")(
				http.HandlerFunc(handleLDAPUserInfo(ldapMiddleware))),
		).Methods("GET")

		// LDAP test connection endpoint
		router.Handle("/auth/ldap/test",
			authManager.GetMiddleware().RequireAdminMiddleware(
				http.HandlerFunc(handleLDAPTest(ldapMiddleware))),
		).Methods("GET")
	}

	// OAuth2 endpoints (if OAuth2 is configured)
	if oauth2Manager := authManager.GetOAuth2Manager(); oauth2Manager != nil {
		setupOAuth2Routes(router, oauth2Manager)
	}

	// JWT token refresh endpoint
	router.HandleFunc("/auth/refresh", handleTokenRefresh(authManager)).Methods("POST")

	// Session management endpoints
	router.HandleFunc("/auth/session", handleSessionInfo(authManager)).Methods("GET")
	router.HandleFunc("/auth/session", handleSessionInvalidate(authManager)).Methods("DELETE")
}

// setupOAuth2Routes configures OAuth2-related routes
func setupOAuth2Routes(router *mux.Router, oauth2Manager *auth.OAuth2Manager) {
	router.HandleFunc("/auth/oauth2/providers", handleOAuth2Providers(oauth2Manager)).Methods("GET")
	router.HandleFunc("/auth/oauth2/authorize/{provider}", handleOAuth2Authorize(oauth2Manager)).Methods("GET")
	router.HandleFunc("/auth/oauth2/callback/{provider}", handleOAuth2Callback(oauth2Manager)).Methods("GET")
}

// setupProtectedRoutes configures routes that require authentication
func setupProtectedRoutes(router *mux.Router, authManager *auth.AuthManager) {
	// Apply authentication middleware to protected routes
	protected := router.PathPrefix("/api").Subrouter()

	// LDAP authentication middleware for direct LDAP auth
	if ldapMiddleware := authManager.GetLDAPMiddleware(); ldapMiddleware != nil {
		protected.Use(ldapMiddleware.LDAPAuthenticateMiddleware)
	} else {
		// Fallback to standard auth middleware
		protected.Use(authManager.GetMiddleware().AuthenticateMiddleware)
	}

	// Example protected endpoints
	protected.HandleFunc("/profile", handleUserProfile).Methods("GET")
	protected.Handle("/intents",
		authManager.GetMiddleware().RequirePermissionMiddleware("intent:read")(
			http.HandlerFunc(handleIntentsList)),
	).Methods("GET")

	protected.Handle("/intents",
		authManager.GetMiddleware().RequirePermissionMiddleware("intent:create")(
			http.HandlerFunc(handleIntentCreate)),
	).Methods("POST")

	// Admin-only endpoints
	protected.Handle("/admin/users",
		authManager.GetMiddleware().RequireAdminMiddleware(
			http.HandlerFunc(handleAdminUsers)),
	).Methods("GET")
}

// HTTP Handlers

func handleAuthInfo(authManager *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providers := authManager.ListProviders()

		info := map[string]interface{}{
			"enabled":   true,
			"providers": providers,
			"features": map[string]bool{
				"ldap":    len(providers["ldap"].(map[string]interface{})) > 0,
				"oauth2":  len(providers["oauth2"].(map[string]interface{})) > 0,
				"session": true,
				"jwt":     true,
				"rbac":    true,
				"csrf":    true,
				"pkce":    true,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}
}

func handleLDAPUserInfo(ldapMiddleware *auth.LDAPAuthMiddleware) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		username := vars["username"]
		provider := r.URL.Query().Get("provider")

		userInfo, err := ldapMiddleware.GetUserInfo(r.Context(), username, provider)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get user info: %v", err), http.StatusNotFound)
			return
		}

		// Remove sensitive information
		response := map[string]interface{}{
			"username":     userInfo.Username,
			"email":        userInfo.Email,
			"display_name": userInfo.Name,
			"first_name":   userInfo.GivenName,
			"last_name":    userInfo.FamilyName,
			"groups":       userInfo.Groups,
			"roles":        userInfo.Roles,
			"provider":     userInfo.Provider,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func handleLDAPTest(ldapMiddleware *auth.LDAPAuthMiddleware) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		results := ldapMiddleware.TestLDAPConnection(r.Context())

		response := map[string]interface{}{
			"results": make(map[string]interface{}),
		}

		allHealthy := true
		for provider, err := range results {
			if err != nil {
				response["results"].(map[string]interface{})[provider] = map[string]interface{}{
					"status": "unhealthy",
					"error":  err.Error(),
				}
				allHealthy = false
			} else {
				response["results"].(map[string]interface{})[provider] = map[string]interface{}{
					"status": "healthy",
				}
			}
		}

		response["overall_status"] = "healthy"
		if !allHealthy {
			response["overall_status"] = "unhealthy"
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func handleOAuth2Providers(oauth2Manager *auth.OAuth2Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// This would be implemented to list available OAuth2 providers
		providers := map[string]interface{}{
			"providers": []string{},
			"message":   "OAuth2 providers endpoint - implementation depends on OAuth2Manager interface",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(providers)
	}
}

func handleOAuth2Authorize(oauth2Manager *auth.OAuth2Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// This would be implemented to handle OAuth2 authorization
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "OAuth2 authorization endpoint - implementation depends on OAuth2Manager interface",
		})
	}
}

func handleOAuth2Callback(oauth2Manager *auth.OAuth2Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// This would be implemented to handle OAuth2 callbacks
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "OAuth2 callback endpoint - implementation depends on OAuth2Manager interface",
		})
	}
}

func handleTokenRefresh(authManager *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			RefreshToken string `json:"refresh_token"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		accessToken, refreshToken, err := authManager.RefreshTokens(r.Context(), request.RefreshToken)
		if err != nil {
			http.Error(w, "Token refresh failed", http.StatusUnauthorized)
			return
		}

		response := map[string]interface{}{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "Bearer",
			"expires_in":    3600,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func handleSessionInfo(authManager *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session ID from cookie or header
		sessionID := getSessionID(r)
		if sessionID == "" {
			http.Error(w, "No session found", http.StatusUnauthorized)
			return
		}

		sessionInfo, err := authManager.ValidateSession(r.Context(), sessionID)
		if err != nil {
			http.Error(w, "Invalid session", http.StatusUnauthorized)
			return
		}

		response := map[string]interface{}{
			"session_id":     sessionInfo.ID,
			"user_id":        sessionInfo.UserID,
			"provider":       sessionInfo.Provider,
			"roles":          sessionInfo.Roles,
			"created_at":     sessionInfo.CreatedAt,
			"last_activity":  sessionInfo.LastActivity,
			"expires_at":     sessionInfo.ExpiresAt,
			"ip_address":     sessionInfo.IPAddress,
			"user_agent":     sessionInfo.UserAgent,
			"sso_enabled":    sessionInfo.SSOEnabled,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func handleSessionInvalidate(authManager *auth.AuthManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := getSessionID(r)
		if sessionID == "" {
			http.Error(w, "No session found", http.StatusBadRequest)
			return
		}

		sessionManager := authManager.GetSessionManager()
		if err := sessionManager.InvalidateSession(r.Context(), sessionID); err != nil {
			http.Error(w, "Failed to invalidate session", http.StatusInternalServerError)
			return
		}

		// Clear session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "nephoran_session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   r.TLS != nil,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Session invalidated successfully",
		})
	}
}

func handleUserProfile(w http.ResponseWriter, r *http.Request) {
	authContext := auth.GetAuthContext(r.Context())
	if authContext == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	profile := map[string]interface{}{
		"user_id":     authContext.UserID,
		"provider":    authContext.Provider,
		"roles":       authContext.Roles,
		"permissions": authContext.Permissions,
		"is_admin":    authContext.IsAdmin,
		"attributes":  authContext.Attributes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func handleIntentsList(w http.ResponseWriter, r *http.Request) {
	authContext := auth.GetAuthContext(r.Context())

	// Example intents list - in real implementation, this would query the database
	intents := []map[string]interface{}{
		{
			"id":          "intent-001",
			"description": "Deploy 5G AMF in production with high availability",
			"status":      "deployed",
			"created_by":  authContext.UserID,
			"created_at":  time.Now().Add(-2 * time.Hour),
		},
		{
			"id":          "intent-002",
			"description": "Scale UPF instances for increased traffic",
			"status":      "pending",
			"created_by":  authContext.UserID,
			"created_at":  time.Now().Add(-1 * time.Hour),
		},
	}

	response := map[string]interface{}{
		"intents": intents,
		"total":   len(intents),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleIntentCreate(w http.ResponseWriter, r *http.Request) {
	authContext := auth.GetAuthContext(r.Context())

	var request struct {
		Description string                 `json:"description"`
		Type        string                 `json:"type"`
		Parameters  map[string]interface{} `json:"parameters"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Example intent creation - in real implementation, this would create the intent
	intent := map[string]interface{}{
		"id":          fmt.Sprintf("intent-%d", time.Now().Unix()),
		"description": request.Description,
		"type":        request.Type,
		"parameters":  request.Parameters,
		"status":      "pending",
		"created_by":  authContext.UserID,
		"created_at":  time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(intent)
}

func handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	// Example admin endpoint - in real implementation, this would query user data
	users := []map[string]interface{}{
		{
			"user_id":      "admin1",
			"username":     "admin1",
			"email":        "admin1@company.com",
			"display_name": "System Administrator",
			"roles":        []string{"system-admin"},
			"last_login":   time.Now().Add(-1 * time.Hour),
			"active":       true,
		},
		{
			"user_id":      "operator1",
			"username":     "operator1",
			"email":        "operator1@company.com",
			"display_name": "Network Operator",
			"roles":        []string{"network-operator"},
			"last_login":   time.Now().Add(-2 * time.Hour),
			"active":       true,
		},
	}

	response := map[string]interface{}{
		"users": users,
		"total": len(users),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions

func getSessionID(r *http.Request) string {
	// Try cookie first
	cookie, err := r.Cookie("nephoran_session")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Try header
	return r.Header.Get("X-Session-ID")
}

// ExampleLDAPIntegrationTest demonstrates how to test LDAP integration
func ExampleLDAPIntegrationTest() {
	fmt.Println("Example LDAP Integration Test")

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create LDAP configuration
	ldapConfig := &providers.LDAPConfig{
		Host:              "localhost",
		Port:              389,
		UseSSL:            false,
		UseTLS:            true,
		BaseDN:            "dc=example,dc=com",
		BindDN:            "cn=admin,dc=example,dc=com",
		BindPassword:      "admin",
		UserSearchBase:    "ou=people,dc=example,dc=com",
		GroupSearchBase:   "ou=groups,dc=example,dc=com",
		IsActiveDirectory: false,
		RoleMappings: map[string][]string{
			"cn=admins,ou=groups,dc=example,dc=com": {"admin"},
			"cn=users,ou=groups,dc=example,dc=com":  {"user"},
		},
		DefaultRoles: []string{"user"},
	}

	// Create LDAP provider
	ldapProvider := providers.NewLDAPProvider(ldapConfig, logger)

	// Test connection
	ctx := context.Background()
	if err := ldapProvider.Connect(ctx); err != nil {
		fmt.Printf("LDAP connection failed: %v\n", err)
		return
	}

	fmt.Println("LDAP connection successful")

	// Test authentication
	userInfo, err := ldapProvider.Authenticate(ctx, "testuser", "testpass")
	if err != nil {
		fmt.Printf("LDAP authentication failed: %v\n", err)
		return
	}

	fmt.Printf("User authenticated: %s (%s)\n", userInfo.Username, userInfo.Email)
	fmt.Printf("Groups: %v\n", userInfo.Groups)
	fmt.Printf("Roles: %v\n", userInfo.Roles)

	// Clean up
	ldapProvider.Close()
}
