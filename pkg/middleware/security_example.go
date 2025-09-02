// Package middleware provides comprehensive security examples
package middleware

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

// ExampleSecurityImplementation demonstrates how to use the security middleware
// in a real application with the Nephoran Intent Operator
func ExampleSecurityImplementation() {
	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create comprehensive security configuration
	securityConfig := &SecuritySuiteConfig{
		// Security Headers Configuration
		SecurityHeaders: &SecurityHeadersConfig{
			EnableHSTS:            true,
			HSTSMaxAge:            31536000, // 1 year
			HSTSIncludeSubDomains: true,
			HSTSPreload:           true,
			ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
			FrameOptions:          "SAMEORIGIN",
			ContentTypeOptions:    true,
			ReferrerPolicy:        "strict-origin-when-cross-origin",
			PermissionsPolicy:     "geolocation=(), microphone=(), camera=()",
			XSSProtection:         "1; mode=block",
		},

		// Input Validation Configuration
		InputValidation: &InputValidationConfig{
			MaxBodySize:                      10 * 1024 * 1024, // 10MB
			MaxHeaderSize:                    8 * 1024,         // 8KB
			MaxURLLength:                     2048,
			MaxParameterCount:                100,
			MaxParameterLength:               1024,
			AllowedContentTypes:              []string{"application/json", "application/x-www-form-urlencoded"},
			EnableSQLInjectionProtection:     true,
			EnableXSSProtection:              true,
			EnablePathTraversalProtection:    true,
			EnableCommandInjectionProtection: true,
			SanitizeInput:                    true,
			LogViolations:                    true,
			BlockOnViolation:                 true,
		},

		// Rate Limiting Configuration
		RateLimit: &RateLimiterConfig{
			QPS:             20, // 20 requests per second per IP
			Burst:           40, // Allow burst of 40 requests
			CleanupInterval: 10 * time.Minute,
			IPTimeout:       1 * time.Hour,
		},

		// Request Size Configuration
		RequestSize: NewRequestSizeLimiter(10*1024*1024, logger), // 10MB

		// CORS Configuration
		CORS: &CORSConfig{
			AllowedOrigins:   []string{"https://app.nephoran.io", "https://dashboard.nephoran.io"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Request-ID"},
			ExposedHeaders:   []string{"X-Request-ID", "X-RateLimit-Limit", "X-RateLimit-Remaining"},
			AllowCredentials: true,
			MaxAge:           24 * time.Hour,
		},

		// Authentication Configuration
		RequireAuth: true,
		AuthValidator: func(r *http.Request) (bool, error) {
			// Example JWT validation
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				return false, fmt.Errorf("missing authorization header")
			}
			// In production, validate JWT token here
			// For example purposes, we'll check for a specific prefix
			return len(authHeader) > 7 && authHeader[:7] == "Bearer ", nil
		},

		// Audit and Metrics
		EnableAudit:   true,
		AuditLogger:   logger.With(slog.String("component", "security-audit")),
		EnableMetrics: true,
		MetricsPrefix: "nephoran_security",

		// CSRF Protection
		EnableCSRF:      true,
		CSRFTokenHeader: "X-CSRF-Token",
		CSRFCookieName:  "nephoran_csrf",

		// Request Fingerprinting
		EnableFingerprinting: true,

		// IP Security (example IPs - adjust for your environment)
		IPWhitelist: []string{
			// Add trusted IPs here
			// "10.0.0.0/8",
			// "192.168.0.0/16",
		},
		IPBlacklist: []string{
			// Add blocked IPs here
			// "192.168.100.50",
		},

		// Timeouts
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Initialize security suite
	securitySuite, err := NewSecuritySuite(securityConfig, logger)
	if err != nil {
		logger.Error("Failed to initialize security suite", slog.String("error", err.Error()))
		return
	}

	// Create router
	router := mux.NewRouter()

	// Apply security middleware globally
	router.Use(securitySuite.Middleware)

	// API Routes for Nephoran Intent Operator
	setupAPIRoutes(router, logger)

	// Health check endpoint (no auth required)
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// CSRF token endpoint
	router.HandleFunc("/api/v1/csrf-token", csrfTokenHandler(securitySuite)).Methods("GET")

	// Start server with timeouts
	server := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info("Starting secure Nephoran Intent Operator API server",
		slog.String("address", server.Addr),
		slog.Bool("auth_required", securityConfig.RequireAuth),
		slog.Bool("csrf_enabled", securityConfig.EnableCSRF),
		slog.Int("rate_limit_qps", securityConfig.RateLimit.QPS))

	if err := server.ListenAndServe(); err != nil {
		logger.Error("Server failed", slog.String("error", err.Error()))
	}
}

// setupAPIRoutes configures the API routes for Nephoran Intent Operator
func setupAPIRoutes(router *mux.Router, logger *slog.Logger) {
	// API v1 routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Network Intent endpoints
	api.HandleFunc("/intents", listIntentsHandler).Methods("GET")
	api.HandleFunc("/intents", createIntentHandler).Methods("POST")
	api.HandleFunc("/intents/{id}", getIntentHandler).Methods("GET")
	api.HandleFunc("/intents/{id}", updateIntentHandler).Methods("PUT")
	api.HandleFunc("/intents/{id}", deleteIntentHandler).Methods("DELETE")

	// Scaling operations
	api.HandleFunc("/intents/{id}/scale", scaleIntentHandler).Methods("POST")

	// Validation endpoint
	api.HandleFunc("/validate", validateIntentHandler).Methods("POST")

	// Metrics endpoint (read-only)
	api.HandleFunc("/metrics", metricsHandler).Methods("GET")
}

// Example handlers for Nephoran Intent Operator

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(json.RawMessage("{}"))
}

func csrfTokenHandler(suite *SecuritySuite) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := suite.GenerateCSRFToken()

		// Set CSRF cookie
		http.SetCookie(w, &http.Cookie{
			Name:     suite.config.CSRFCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400, // 24 hours
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token": token,
		})
	}
}

func listIntentsHandler(w http.ResponseWriter, r *http.Request) {
	// Example: List all network intents
	intents := []map[string]interface{}{}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"intents": intents,
	})
}

func createIntentHandler(w http.ResponseWriter, r *http.Request) {
	// Parse and validate intent from request body
	var intent map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&intent); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate intent structure (example)
	if intent["name"] == nil || intent["target_type"] == nil {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Create intent (example response)
	intent["id"] = fmt.Sprintf("intent-%d", time.Now().Unix())
	intent["status"] = "pending"
	intent["created_at"] = time.Now()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(intent)
}

func getIntentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	intentID := vars["id"]

	// Example response
	intent := map[string]interface{}{
		"id": intentID,
		"spec": map[string]interface{}{
			"replicas":    3,
			"cpu_request": "500m",
			"mem_request": "1Gi",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(intent)
}

func updateIntentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	intentID := vars["id"]

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update intent (example response)
	updates["id"] = intentID
	updates["updated_at"] = time.Now()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updates)
}

func deleteIntentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	intentID := vars["id"]

	// Log the intentID for audit purposes
	log.Printf("Deleting intent: %s", intentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(json.RawMessage("{}"))
}

func scaleIntentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	intentID := vars["id"]

	// Log the intentID for audit purposes
	log.Printf("Scaling intent: %s", intentID)

	var scaleRequest map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&scaleRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate scale request
	if scaleRequest["replicas"] == nil {
		http.Error(w, "Missing replicas field", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(json.RawMessage("{}"))
}

func validateIntentHandler(w http.ResponseWriter, r *http.Request) {
	var intent map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&intent); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Perform validation
	validationResult := map[string]interface{}{
		"valid":    true,
		"errors":   []string{},
		"warnings": []string{},
	}

	// Example validation rules
	if intent["name"] == nil {
		validationResult["valid"] = false
		validationResult["errors"] = append(validationResult["errors"].([]string), "Missing required field: name")
	}

	if intent["target_type"] == nil {
		validationResult["valid"] = false
		validationResult["errors"] = append(validationResult["errors"].([]string), "Missing required field: target_type")
	}

	w.Header().Set("Content-Type", "application/json")
	if !validationResult["valid"].(bool) {
		w.WriteHeader(http.StatusBadRequest)
	}
	json.NewEncoder(w).Encode(validationResult)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	// Example metrics response
	metrics := json.RawMessage("{}")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// ExampleCustomValidation shows how to add custom validation rules
func ExampleCustomValidation() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Custom validator for Nephoran-specific rules
	nephoranValidator := func(r *http.Request) error {
		// Validate Nephoran-specific headers
		if r.Header.Get("X-Nephoran-Version") == "" {
			return fmt.Errorf("missing required header: X-Nephoran-Version")
		}

		// Validate intent structure for POST/PUT requests
		if r.Method == "POST" || r.Method == "PUT" {
			// Add specific validation logic here
		}

		return nil
	}

	config := &InputValidationConfig{
		EnableSQLInjectionProtection: true,
		EnableXSSProtection:          true,
		CustomValidators:             []ValidatorFunc{nephoranValidator},
		BlockOnViolation:             true,
	}

	validator, _ := NewInputValidator(config, logger)

	// Use validator in your middleware chain
	_ = validator
}
