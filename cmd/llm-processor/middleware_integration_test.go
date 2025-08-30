package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/middleware"
)

// createIPAllowlistHandler creates a test handler with IP allowlist functionality
func createIPAllowlistHandler(next http.Handler, allowedCIDRs []string, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple IP check for testing purposes
		remoteIP := r.RemoteAddr
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			remoteIP = strings.TrimSpace(parts[0])
		}
		
		// For testing, allow localhost/127.0.0.1 and common test IPs
		if strings.Contains(remoteIP, "127.0.0.1") || strings.Contains(remoteIP, "192.168.") || 
		   strings.Contains(remoteIP, "10.0.") || remoteIP == "" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Check against allowed CIDRs for other cases
		for _, cidr := range allowedCIDRs {
			if strings.Contains(remoteIP, strings.Split(cidr, "/")[0]) {
				next.ServeHTTP(w, r)
				return
			}
		}
		
		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}

// TestSecurityHeadersMiddleware validates that security headers are correctly set
func TestSecurityHeadersMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name              string
		tlsEnabled        bool
		requestHeaders    map[string]string
		expectedHeaders   map[string]string
		unexpectedHeaders []string
		description       string
	}{
		{
			name:       "Basic security headers without TLS",
			tlsEnabled: false,
			expectedHeaders: map[string]string{
				"X-Frame-Options":         "DENY",
				"X-Content-Type-Options":  "nosniff",
				"Referrer-Policy":         "strict-origin-when-cross-origin",
				"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'; base-uri 'none'",
				"Permissions-Policy":      "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()",
				"X-XSS-Protection":        "1; mode=block",
			},
			unexpectedHeaders: []string{"Strict-Transport-Security"},
			description:       "Should set all security headers except HSTS when TLS is disabled",
		},
		{
			name:       "Security headers with TLS enabled",
			tlsEnabled: true,
			requestHeaders: map[string]string{
				"X-Forwarded-Proto": "https",
			},
			expectedHeaders: map[string]string{
				"X-Frame-Options":           "DENY",
				"X-Content-Type-Options":    "nosniff",
				"Referrer-Policy":           "strict-origin-when-cross-origin",
				"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'none'; base-uri 'none'",
				"Permissions-Policy":        "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()",
				"X-XSS-Protection":          "1; mode=block",
				"Strict-Transport-Security": "max-age=31536000",
			},
			description: "Should set all security headers including HSTS when TLS is enabled",
		},
		{
			name:           "TLS enabled but not secure connection",
			tlsEnabled:     true,
			requestHeaders: map[string]string{
				// No TLS indicators
			},
			expectedHeaders: map[string]string{
				"X-Frame-Options":        "DENY",
				"X-Content-Type-Options": "nosniff",
				"Referrer-Policy":        "strict-origin-when-cross-origin",
			},
			unexpectedHeaders: []string{"Strict-Transport-Security"},
			description:       "Should not set HSTS when TLS is enabled but connection is not secure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create security headers middleware
			securityConfig := middleware.DefaultSecurityHeadersConfig()
			securityConfig.EnableHSTS = tt.tlsEnabled
			securityConfig.ContentSecurityPolicy = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"

			securityHeaders := middleware.NewSecurityHeaders(securityConfig, logger)

			// Create test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Wrap with middleware
			handler := securityHeaders.Middleware(testHandler)

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)

			// Set TLS state if needed
			if tt.tlsEnabled && tt.requestHeaders["X-Forwarded-Proto"] == "https" {
				req.TLS = &tls.ConnectionState{} // Simulate TLS connection
			}

			// Set request headers
			for key, value := range tt.requestHeaders {
				req.Header.Set(key, value)
			}

			// Execute request
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			// Verify expected headers
			for expectedHeader, expectedValue := range tt.expectedHeaders {
				actualValue := rr.Header().Get(expectedHeader)
				if actualValue != expectedValue {
					t.Errorf("Test '%s': Expected header '%s' to be '%s', got '%s'. %s",
						tt.name, expectedHeader, expectedValue, actualValue, tt.description)
				}
			}

			// Verify unexpected headers are not present
			for _, unexpectedHeader := range tt.unexpectedHeaders {
				if actualValue := rr.Header().Get(unexpectedHeader); actualValue != "" {
					t.Errorf("Test '%s': Did not expect header '%s' to be set, but got '%s'. %s",
						tt.name, unexpectedHeader, actualValue, tt.description)
				}
			}

			// Verify response is successful
			if rr.Code != http.StatusOK {
				t.Errorf("Test '%s': Expected status 200, got %d", tt.name, rr.Code)
			}
		})
	}
}

// TestRedactLoggerMiddleware validates that sensitive data is properly redacted in logs
func TestRedactLoggerMiddleware(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	tests := []struct {
		name             string
		method           string
		path             string
		headers          map[string]string
		queryParams      string
		requestBody      string
		shouldSkip       bool
		expectedRedacted []string
		unexpectedInLogs []string
		description      string
	}{
		{
			name:   "Redact Authorization header",
			method: "POST",
			path:   "/process",
			headers: map[string]string{
				"Authorization": "Bearer secret-token-12345",
				"Content-Type":  "application/json",
			},
			requestBody:      `{"intent": "Deploy network function"}`,
			expectedRedacted: []string{"[REDACTED]"},
			unexpectedInLogs: []string{"secret-token-12345"},
			description:      "Authorization header should be redacted in logs",
		},
		{
			name:             "Redact sensitive query parameters",
			method:           "GET",
			path:             "/status",
			queryParams:      "token=secret123&api_key=key456&normal_param=value",
			expectedRedacted: []string{"[REDACTED]"},
			unexpectedInLogs: []string{"secret123", "key456"},
			description:      "Sensitive query parameters should be redacted",
		},
		{
			name:   "Redact JSON fields in request body",
			method: "POST",
			path:   "/process",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			requestBody:      `{"intent": "Deploy function", "password": "secret-password", "api_key": "key-12345"}`,
			expectedRedacted: []string{"[REDACTED]"},
			unexpectedInLogs: []string{"secret-password", "key-12345"},
			description:      "Sensitive JSON fields should be redacted in request body",
		},
		{
			name:        "Skip health check paths",
			method:      "GET",
			path:        "/healthz",
			shouldSkip:  true,
			description: "Health check paths should be skipped from logging",
		},
		{
			name:   "Multiple sensitive headers",
			method: "GET",
			path:   "/test",
			headers: map[string]string{
				"Authorization":   "Bearer token123",
				"X-API-Key":       "apikey456",
				"Cookie":          "session=abc123",
				"X-Custom-Header": "normal-value",
			},
			expectedRedacted: []string{"[REDACTED]"},
			unexpectedInLogs: []string{"token123", "apikey456", "abc123"},
			description:      "Multiple sensitive headers should be redacted while preserving normal headers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear log buffer
			logBuffer.Reset()

			// Create redact logger middleware
			redactConfig := middleware.DefaultRedactLoggerConfig()
			redactConfig.LogLevel = slog.LevelDebug
			redactConfig.LogRequestBody = true
			redactConfig.LogResponseBody = true

			redactLogger, err := middleware.NewRedactLogger(redactConfig, logger)
			if err != nil {
				t.Fatalf("Failed to create redact logger: %v", err)
			}

			// Create test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"status": "success"})
			})

			// Wrap with middleware
			handler := redactLogger.Middleware(testHandler)

			// Create request
			url := tt.path
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			var reqBody io.Reader
			if tt.requestBody != "" {
				reqBody = strings.NewReader(tt.requestBody)
			}

			req := httptest.NewRequest(tt.method, url, reqBody)

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Execute request
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			// Check if path should be skipped
			if tt.shouldSkip {
				logOutput := logBuffer.String()
				if logOutput != "" {
					t.Errorf("Test '%s': Expected no logs for skipped path, but got: %s", tt.name, logOutput)
				}
				return
			}

			// Verify response is successful
			if rr.Code != http.StatusOK {
				t.Errorf("Test '%s': Expected status 200, got %d", tt.name, rr.Code)
			}

			// Check log output
			logOutput := logBuffer.String()
			if logOutput == "" {
				t.Errorf("Test '%s': Expected log output, but got none", tt.name)
				return
			}

			// Verify sensitive data is redacted
			for _, sensitiveData := range tt.unexpectedInLogs {
				if strings.Contains(logOutput, sensitiveData) {
					t.Errorf("Test '%s': Found sensitive data '%s' in logs: %s. %s",
						tt.name, sensitiveData, logOutput, tt.description)
				}
			}

			// Verify redacted placeholder exists (if expected)
			if len(tt.expectedRedacted) > 0 {
				foundRedacted := false
				for _, redactedValue := range tt.expectedRedacted {
					if strings.Contains(logOutput, redactedValue) {
						foundRedacted = true
						break
					}
				}
				if !foundRedacted {
					t.Errorf("Test '%s': Expected to find redacted values in logs, but didn't. Log: %s. %s",
						tt.name, logOutput, tt.description)
				}
			}
		})
	}
}

// TestCorrelationIDGeneration validates correlation ID generation and propagation
func TestCorrelationIDGeneration(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	tests := []struct {
		name                string
		existingHeaderName  string
		existingHeaderValue string
		expectGeneration    bool
		expectedInResponse  string
		description         string
	}{
		{
			name:               "Generate new correlation ID when none exists",
			expectGeneration:   true,
			expectedInResponse: "X-Request-ID",
			description:        "Should generate new correlation ID and set response header",
		},
		{
			name:                "Use existing X-Request-ID",
			existingHeaderName:  "X-Request-ID",
			existingHeaderValue: "existing-request-123",
			expectGeneration:    false,
			expectedInResponse:  "existing-request-123",
			description:         "Should use existing X-Request-ID header",
		},
		{
			name:                "Use existing X-Correlation-ID",
			existingHeaderName:  "X-Correlation-ID",
			existingHeaderValue: "corr-456",
			expectGeneration:    false,
			expectedInResponse:  "corr-456",
			description:         "Should use existing X-Correlation-ID header",
		},
		{
			name:                "Use existing X-Trace-ID",
			existingHeaderName:  "X-Trace-ID",
			existingHeaderValue: "trace-789",
			expectGeneration:    false,
			expectedInResponse:  "trace-789",
			description:         "Should use existing X-Trace-ID header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear log buffer
			logBuffer.Reset()

			// Create redact logger middleware
			redactConfig := middleware.DefaultRedactLoggerConfig()
			redactConfig.LogLevel = slog.LevelDebug
			redactConfig.GenerateCorrelationID = true

			redactLogger, err := middleware.NewRedactLogger(redactConfig, logger)
			if err != nil {
				t.Fatalf("Failed to create redact logger: %v", err)
			}

			// Create test handler that can access correlation ID from context
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				correlationID := r.Context().Value("correlation_id")
				if correlationID == nil && tt.expectGeneration {
					t.Errorf("Test '%s': Expected correlation ID in context, but got none", tt.name)
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Wrap with middleware
			handler := redactLogger.Middleware(testHandler)

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)

			// Set existing header if provided
			if tt.existingHeaderName != "" && tt.existingHeaderValue != "" {
				req.Header.Set(tt.existingHeaderName, tt.existingHeaderValue)
			}

			// Execute request
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			// Verify response is successful
			if rr.Code != http.StatusOK {
				t.Errorf("Test '%s': Expected status 200, got %d", tt.name, rr.Code)
			}

			// Check response header
			responseHeaderValue := rr.Header().Get("X-Request-ID")
			if responseHeaderValue == "" {
				t.Errorf("Test '%s': Expected X-Request-ID header in response, but got none", tt.name)
			}

			// Verify correlation ID value
			if tt.expectedInResponse != "X-Request-ID" {
				// Exact value expected
				if responseHeaderValue != tt.expectedInResponse {
					t.Errorf("Test '%s': Expected correlation ID '%s', got '%s'. %s",
						tt.name, tt.expectedInResponse, responseHeaderValue, tt.description)
				}
			} else {
				// Generated value expected - should be non-empty UUID format
				if len(responseHeaderValue) < 10 {
					t.Errorf("Test '%s': Generated correlation ID seems too short: '%s'", tt.name, responseHeaderValue)
				}
			}

			// Check logs contain correlation ID
			logOutput := logBuffer.String()
			if !strings.Contains(logOutput, responseHeaderValue) {
				t.Errorf("Test '%s': Expected correlation ID '%s' to appear in logs, but didn't find it. Log: %s",
					tt.name, responseHeaderValue, logOutput)
			}
		})
	}
}

// TestMiddlewareOrdering validates that middlewares are applied in the correct order
func TestMiddlewareOrdering(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create a test configuration similar to main.go
	cfg := &config.LLMProcessorConfig{
		TLSEnabled:     true,
		LogLevel:       "debug",
		CORSEnabled:    true,
		AllowedOrigins: []string{"http://localhost:3000"},
	}

	// Create router similar to setupHTTPServer
	router := mux.NewRouter()

	// 1. Redact Logger (first)
	redactLoggerConfig := middleware.DefaultRedactLoggerConfig()
	redactLoggerConfig.LogLevel = slog.LevelDebug
	redactLoggerConfig.LogRequestBody = true
	redactLoggerConfig.LogResponseBody = true

	redactLogger, err := middleware.NewRedactLogger(redactLoggerConfig, logger)
	if err != nil {
		t.Fatalf("Failed to create redact logger: %v", err)
	}
	router.Use(redactLogger.Middleware)

	// 2. Security Headers (second)
	securityHeadersConfig := middleware.DefaultSecurityHeadersConfig()
	securityHeadersConfig.EnableHSTS = cfg.TLSEnabled
	securityHeadersConfig.ContentSecurityPolicy = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"

	securityHeaders := middleware.NewSecurityHeaders(securityHeadersConfig, logger)
	router.Use(securityHeaders.Middleware)

	// 3. CORS (third)
	if cfg.CORSEnabled {
		corsConfig := middleware.CORSConfig{
			AllowedOrigins: cfg.AllowedOrigins,
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
			MaxAge:         24 * time.Hour,
		}
		corsMiddleware := middleware.NewCORSMiddleware(corsConfig, logger)
		router.Use(corsMiddleware.Middleware)
	}

	// Test handler
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "success"})
	}).Methods("POST", "OPTIONS")

	tests := []struct {
		name            string
		method          string
		origin          string
		headers         map[string]string
		requestBody     string
		expectedStatus  int
		expectedHeaders map[string]string
		checkLogs       bool
		description     string
	}{
		{
			name:   "CORS preflight request",
			method: "OPTIONS",
			origin: "http://localhost:3000",
			headers: map[string]string{
				"Access-Control-Request-Method":  "POST",
				"Access-Control-Request-Headers": "Content-Type,Authorization",
			},
			expectedStatus: http.StatusNoContent,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "http://localhost:3000",
				"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
				"X-Frame-Options":              "DENY",
				"X-Content-Type-Options":       "nosniff",
				"X-Request-ID":                 "", // Should exist but value varies
			},
			checkLogs:   false, // OPTIONS requests might be skipped
			description: "CORS preflight should work with all middlewares",
		},
		{
			name:   "POST request with sensitive data",
			method: "POST",
			origin: "http://localhost:3000",
			headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer secret-token",
			},
			requestBody:    `{"intent": "test", "api_key": "sensitive-key"}`,
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Access-Control-Allow-Origin": "http://localhost:3000",
				"X-Frame-Options":             "DENY",
				"Content-Type":                "application/json",
				"X-Request-ID":                "", // Should exist
			},
			checkLogs:   true,
			description: "POST request should have all middleware effects applied",
		},
		{
			name:   "Blocked CORS origin",
			method: "POST",
			origin: "http://malicious-site.com",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			requestBody:    `{"intent": "test"}`,
			expectedStatus: http.StatusForbidden,
			checkLogs:      true,
			description:    "Request from blocked origin should be denied by CORS middleware",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear log buffer
			logBuffer.Reset()

			// Create request
			var reqBody io.Reader
			if tt.requestBody != "" {
				reqBody = strings.NewReader(tt.requestBody)
			}

			req := httptest.NewRequest(tt.method, "/test", reqBody)

			// Set origin if specified
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			// Set additional headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Simulate HTTPS connection for HSTS
			req.TLS = &tls.ConnectionState{}

			// Execute request
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Test '%s': Expected status %d, got %d. %s",
					tt.name, tt.expectedStatus, rr.Code, tt.description)
			}

			// Check expected headers
			for expectedHeader, expectedValue := range tt.expectedHeaders {
				actualValue := rr.Header().Get(expectedHeader)

				if expectedValue == "" {
					// Just check that header exists
					if actualValue == "" {
						t.Errorf("Test '%s': Expected header '%s' to exist, but it's missing",
							tt.name, expectedHeader)
					}
				} else {
					// Check exact value
					if actualValue != expectedValue {
						t.Errorf("Test '%s': Expected header '%s' to be '%s', got '%s'",
							tt.name, expectedHeader, expectedValue, actualValue)
					}
				}
			}

			// Check HSTS header is present for HTTPS
			hstsHeader := rr.Header().Get("Strict-Transport-Security")
			if hstsHeader == "" {
				t.Errorf("Test '%s': Expected HSTS header for HTTPS request", tt.name)
			}

			// Check logs if required
			if tt.checkLogs {
				logOutput := logBuffer.String()

				// Should contain correlation ID
				correlationID := rr.Header().Get("X-Request-ID")
				if correlationID != "" && !strings.Contains(logOutput, correlationID) {
					t.Errorf("Test '%s': Expected correlation ID '%s' in logs, but didn't find it",
						tt.name, correlationID)
				}

				// Should not contain sensitive data
				if tt.requestBody != "" && strings.Contains(tt.requestBody, "secret-token") {
					if strings.Contains(logOutput, "secret-token") {
						t.Errorf("Test '%s': Found sensitive token in logs: %s", tt.name, logOutput)
					}
				}
			}
		})
	}
}

// TestHSTSHeaderBehavior specifically tests HSTS header behavior with different TLS configurations
func TestHSTSHeaderBehavior(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name           string
		tlsEnabled     bool
		hasHSTSConfig  bool
		requestHasTLS  bool
		forwardedProto string
		expectedHSTS   string
		shouldHaveHSTS bool
		description    string
	}{
		{
			name:           "HSTS enabled with TLS connection",
			tlsEnabled:     true,
			hasHSTSConfig:  true,
			requestHasTLS:  true,
			expectedHSTS:   "max-age=31536000",
			shouldHaveHSTS: true,
			description:    "Should set HSTS header when TLS is enabled and connection is secure",
		},
		{
			name:           "HSTS enabled with X-Forwarded-Proto https",
			tlsEnabled:     true,
			hasHSTSConfig:  true,
			forwardedProto: "https",
			expectedHSTS:   "max-age=31536000",
			shouldHaveHSTS: true,
			description:    "Should set HSTS header when behind HTTPS proxy",
		},
		{
			name:           "HSTS disabled - no header",
			tlsEnabled:     false,
			hasHSTSConfig:  false,
			requestHasTLS:  true, // Even with TLS, HSTS is disabled in config
			shouldHaveHSTS: false,
			description:    "Should not set HSTS header when disabled in config",
		},
		{
			name:           "HSTS enabled but no secure connection",
			tlsEnabled:     true,
			hasHSTSConfig:  true,
			requestHasTLS:  false,
			forwardedProto: "http", // Explicitly HTTP
			shouldHaveHSTS: false,
			description:    "Should not set HSTS header for non-secure connections",
		},
		{
			name:           "HSTS with includeSubDomains",
			tlsEnabled:     true,
			hasHSTSConfig:  true,
			requestHasTLS:  true,
			expectedHSTS:   "max-age=31536000; includeSubDomains",
			shouldHaveHSTS: true,
			description:    "Should include subdomains directive when configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create security headers config
			config := middleware.DefaultSecurityHeadersConfig()
			config.EnableHSTS = tt.hasHSTSConfig
			config.HSTSMaxAge = 31536000

			// Special case for includeSubDomains test
			if strings.Contains(tt.expectedHSTS, "includeSubDomains") {
				config.HSTSIncludeSubDomains = true
			}

			securityHeaders := middleware.NewSecurityHeaders(config, logger)

			// Create test handler
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Wrap with middleware
			handler := securityHeaders.Middleware(testHandler)

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)

			// Set TLS state if needed
			if tt.requestHasTLS {
				req.TLS = &tls.ConnectionState{}
			}

			// Set forwarded proto header if specified
			if tt.forwardedProto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.forwardedProto)
			}

			// Execute request
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			// Check HSTS header
			hstsHeader := rr.Header().Get("Strict-Transport-Security")

			if tt.shouldHaveHSTS {
				if hstsHeader == "" {
					t.Errorf("Test '%s': Expected HSTS header but got none. %s", tt.name, tt.description)
				} else if hstsHeader != tt.expectedHSTS {
					t.Errorf("Test '%s': Expected HSTS header '%s', got '%s'. %s",
						tt.name, tt.expectedHSTS, hstsHeader, tt.description)
				}
			} else {
				if hstsHeader != "" {
					t.Errorf("Test '%s': Did not expect HSTS header but got '%s'. %s",
						tt.name, hstsHeader, tt.description)
				}
			}

			// Verify response is successful
			if rr.Code != http.StatusOK {
				t.Errorf("Test '%s': Expected status 200, got %d", tt.name, rr.Code)
			}
		})
	}
}

// TestMiddlewareWithExistingEndpoints tests middleware integration with actual endpoints
func TestMiddlewareWithExistingEndpoints(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Create configuration similar to main.go
	cfg := &config.LLMProcessorConfig{
		TLSEnabled:            true,
		LogLevel:              "debug",
		MaxRequestSize:        1024 * 1024, // 1MB
		CORSEnabled:           true,
		AllowedOrigins:        []string{"http://localhost:3000"},
		ExposeMetricsPublicly: false,
		MetricsAllowedCIDRs:   []string{"127.0.0.0/8"},
	}

	// Create router with all middlewares like in main.go
	router := mux.NewRouter()

	// Apply middlewares in the same order as main.go
	// 1. Redact Logger
	redactLoggerConfig := middleware.DefaultRedactLoggerConfig()
	redactLoggerConfig.LogLevel = slog.LevelDebug
	redactLoggerConfig.LogRequestBody = true
	redactLogger, _ := middleware.NewRedactLogger(redactLoggerConfig, logger)
	router.Use(redactLogger.Middleware)

	// 2. Security Headers
	securityHeadersConfig := middleware.DefaultSecurityHeadersConfig()
	securityHeadersConfig.EnableHSTS = cfg.TLSEnabled
	securityHeaders := middleware.NewSecurityHeaders(securityHeadersConfig, logger)
	router.Use(securityHeaders.Middleware)

	// 3. CORS
	corsConfig := middleware.CORSConfig{
		AllowedOrigins: cfg.AllowedOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		MaxAge:         24 * time.Hour,
	}
	corsMiddleware := middleware.NewCORSMiddleware(corsConfig, logger)
	router.Use(corsMiddleware.Middleware)

	// Add endpoints similar to main.go
	// Health endpoints (should be skipped from detailed logging)
	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")

	router.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	}).Methods("GET")

	// Metrics endpoint with IP allowlist
	metricsHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("# HELP test_metric A test metric\ntest_metric 1\n"))
	}

	if cfg.ExposeMetricsPublicly {
		router.HandleFunc("/metrics", metricsHandler).Methods("GET")
	} else {
		router.HandleFunc("/metrics", createIPAllowlistHandler(
			http.HandlerFunc(metricsHandler), cfg.MetricsAllowedCIDRs, logger)).Methods("GET")
	}

	// Main processing endpoints with size limits
	processHandler := middleware.MaxBytesHandler(cfg.MaxRequestSize, logger,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "success",
				"request_id": r.Header.Get("X-Request-ID"),
			})
		}))
	router.Handle("/process", processHandler).Methods("POST")

	tests := []struct {
		name           string
		endpoint       string
		method         string
		origin         string
		clientIP       string
		headers        map[string]string
		requestBody    string
		expectedStatus int
		checkHeaders   map[string]bool
		shouldHaveLogs bool
		description    string
	}{
		{
			name:           "Health check endpoint",
			endpoint:       "/healthz",
			method:         "GET",
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]bool{
				"X-Frame-Options":        true,
				"X-Content-Type-Options": true,
				"X-Request-ID":           true,
			},
			shouldHaveLogs: false, // Health checks are skipped
			description:    "Health check should have security headers but minimal logging",
		},
		{
			name:     "Process endpoint with authentication",
			endpoint: "/process",
			method:   "POST",
			origin:   "http://localhost:3000",
			headers: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer secret-token-123",
			},
			requestBody:    `{"intent": "Deploy a network function", "api_key": "sensitive-key"}`,
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]bool{
				"Access-Control-Allow-Origin": true,
				"X-Frame-Options":             true,
				"X-Request-ID":                true,
				"Strict-Transport-Security":   true,
			},
			shouldHaveLogs: true,
			description:    "Process endpoint should have full middleware chain",
		},
		{
			name:           "Metrics endpoint from allowed IP",
			endpoint:       "/metrics",
			method:         "GET",
			clientIP:       "127.0.0.1",
			expectedStatus: http.StatusOK,
			checkHeaders: map[string]bool{
				"X-Frame-Options": true,
				"Content-Type":    true,
			},
			shouldHaveLogs: true,
			description:    "Metrics from allowed IP should succeed",
		},
		{
			name:           "Metrics endpoint from blocked IP",
			endpoint:       "/metrics",
			method:         "GET",
			clientIP:       "8.8.8.8",
			expectedStatus: http.StatusForbidden,
			shouldHaveLogs: true,
			description:    "Metrics from blocked IP should be denied",
		},
		{
			name:     "CORS preflight for process endpoint",
			endpoint: "/process",
			method:   "OPTIONS",
			origin:   "http://localhost:3000",
			headers: map[string]string{
				"Access-Control-Request-Method":  "POST",
				"Access-Control-Request-Headers": "Content-Type,Authorization",
			},
			expectedStatus: http.StatusNoContent,
			checkHeaders: map[string]bool{
				"Access-Control-Allow-Origin":  true,
				"Access-Control-Allow-Methods": true,
			},
			shouldHaveLogs: false, // OPTIONS might be skipped
			description:    "CORS preflight should work correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear log buffer
			logBuffer.Reset()

			// Create request
			var reqBody io.Reader
			if tt.requestBody != "" {
				reqBody = strings.NewReader(tt.requestBody)
			}

			req := httptest.NewRequest(tt.method, tt.endpoint, reqBody)

			// Set origin if specified
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			// Set client IP if specified
			if tt.clientIP != "" {
				req.Header.Set("X-Forwarded-For", tt.clientIP)
			}

			// Set additional headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Simulate HTTPS connection for HSTS
			req.TLS = &tls.ConnectionState{}

			// Execute request
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Test '%s': Expected status %d, got %d. %s",
					tt.name, tt.expectedStatus, rr.Code, tt.description)
			}

			// Check expected headers
			for header, shouldExist := range tt.checkHeaders {
				headerValue := rr.Header().Get(header)
				if shouldExist && headerValue == "" {
					t.Errorf("Test '%s': Expected header '%s' to exist but it's missing",
						tt.name, header)
				} else if !shouldExist && headerValue != "" {
					t.Errorf("Test '%s': Did not expect header '%s' but got '%s'",
						tt.name, header, headerValue)
				}
			}

			// Check logs
			logOutput := logBuffer.String()
			if tt.shouldHaveLogs {
				if logOutput == "" {
					t.Errorf("Test '%s': Expected log output but got none", tt.name)
				} else {
					// Check that sensitive data is redacted in logs
					if tt.requestBody != "" {
						if strings.Contains(logOutput, "secret-token-123") {
							t.Errorf("Test '%s': Found sensitive token in logs", tt.name)
						}
						if strings.Contains(logOutput, "sensitive-key") {
							t.Errorf("Test '%s': Found sensitive API key in logs", tt.name)
						}
						// Should have redacted placeholder
						if !strings.Contains(logOutput, "[REDACTED]") {
							t.Errorf("Test '%s': Expected [REDACTED] in logs but didn't find it", tt.name)
						}
					}
				}
			} else {
				if logOutput != "" && tt.endpoint == "/healthz" {
					t.Errorf("Test '%s': Expected no logs for health check but got: %s", tt.name, logOutput)
				}
			}

			// For successful process requests, verify response structure
			if tt.endpoint == "/process" && rr.Code == http.StatusOK {
				var response map[string]interface{}
				if err := json.NewDecoder(strings.NewReader(rr.Body.String())).Decode(&response); err != nil {
					t.Errorf("Test '%s': Failed to decode response JSON: %v", tt.name, err)
				} else {
					if response["status"] != "success" {
						t.Errorf("Test '%s': Expected status 'success', got %v", tt.name, response["status"])
					}
					if response["request_id"] == nil || response["request_id"] == "" {
						t.Errorf("Test '%s': Expected request_id in response", tt.name)
					}
				}
			}
		})
	}
}

// TestMiddlewareErrorHandling tests how middlewares handle various error conditions
func TestMiddlewareErrorHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	tests := []struct {
		name            string
		setupMiddleware func() http.Handler
		requestSetup    func() *http.Request
		expectedStatus  int
		expectedHeaders map[string]string
		description     string
	}{
		{
			name: "Request size limit exceeded",
			setupMiddleware: func() http.Handler {
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("OK"))
				})
				return middleware.MaxBytesHandler(100, logger, handler) // Very small limit
			},
			requestSetup: func() *http.Request {
				largeBody := strings.Repeat("x", 200) // Exceeds limit
				return httptest.NewRequest("POST", "/test", strings.NewReader(largeBody))
			},
			expectedStatus: http.StatusRequestEntityTooLarge,
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			description: "Should return 413 for oversized requests",
		},
		{
			name: "Middleware with panic recovery",
			setupMiddleware: func() http.Handler {
				panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					panic("test panic")
				})

				// Wrap with security headers (should still apply headers even if handler panics)
				securityConfig := middleware.DefaultSecurityHeadersConfig()
				securityHeaders := middleware.NewSecurityHeaders(securityConfig, logger)

				// Add a basic recovery middleware
				recoveryWrapper := func(next http.Handler) http.Handler {
					return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						defer func() {
							if err := recover(); err != nil {
								w.WriteHeader(http.StatusInternalServerError)
								w.Write([]byte("Internal Server Error"))
							}
						}()
						next.ServeHTTP(w, r)
					})
				}

				return recoveryWrapper(securityHeaders.Middleware(panicHandler))
			},
			requestSetup: func() *http.Request {
				return httptest.NewRequest("GET", "/test", nil)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedHeaders: map[string]string{
				"X-Frame-Options":        "DENY",
				"X-Content-Type-Options": "nosniff",
			},
			description: "Security headers should be applied even when handler panics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup middleware and request
			handler := tt.setupMiddleware()
			req := tt.requestSetup()

			// Execute request
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Test '%s': Expected status %d, got %d. %s",
					tt.name, tt.expectedStatus, rr.Code, tt.description)
			}

			// Check expected headers
			for header, expectedValue := range tt.expectedHeaders {
				actualValue := rr.Header().Get(header)
				if actualValue != expectedValue {
					t.Errorf("Test '%s': Expected header '%s' to be '%s', got '%s'",
						tt.name, header, expectedValue, actualValue)
				}
			}
		})
	}
}

// Update todo status
func init() {
	// This test file comprehensively covers middleware integration testing
}
