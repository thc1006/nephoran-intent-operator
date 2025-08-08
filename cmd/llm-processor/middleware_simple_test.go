package main

import (
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thc1006/nephoran-intent-operator/pkg/middleware"
)

// TestSecurityHeadersBasic tests basic security headers functionality
func TestSecurityHeadersBasic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create security headers middleware
	config := middleware.DefaultSecurityHeadersConfig()
	config.EnableHSTS = true
	config.ContentSecurityPolicy = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"
	
	securityHeaders := middleware.NewSecurityHeaders(config, logger)

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with middleware
	handler := securityHeaders.Middleware(testHandler)

	t.Run("Security headers without TLS", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)

		expectedHeaders := map[string]string{
			"X-Frame-Options":        "DENY",
			"X-Content-Type-Options": "nosniff",
			"Referrer-Policy":        "strict-origin-when-cross-origin",
		}

		for header, expected := range expectedHeaders {
			if actual := rr.Header().Get(header); actual != expected {
				t.Errorf("Expected %s to be %s, got %s", header, expected, actual)
			}
		}

		// Should not have HSTS without secure connection
		if hsts := rr.Header().Get("Strict-Transport-Security"); hsts != "" {
			t.Errorf("Should not have HSTS header without secure connection, got: %s", hsts)
		}

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	t.Run("Security headers with TLS", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.TLS = &tls.ConnectionState{} // Simulate TLS
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)

		// Should have HSTS with secure connection
		if hsts := rr.Header().Get("Strict-Transport-Security"); hsts == "" {
			t.Error("Expected HSTS header with secure connection")
		} else if !strings.Contains(hsts, "max-age=31536000") {
			t.Errorf("Expected HSTS max-age, got: %s", hsts)
		}

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
}

// TestRedactLoggerBasic tests basic redact logger functionality  
func TestRedactLoggerBasic(t *testing.T) {
	// This test verifies the redact logger can be created and used
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))

	config := middleware.DefaultRedactLoggerConfig()
	config.LogLevel = slog.LevelDebug

	redactLogger, err := middleware.NewRedactLogger(config, logger)
	if err != nil {
		t.Fatalf("Failed to create redact logger: %v", err)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler := redactLogger.Middleware(testHandler)

	t.Run("Request with sensitive header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer secret-token")
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)

		// Should have correlation ID header
		if correlationID := rr.Header().Get("X-Request-ID"); correlationID == "" {
			t.Error("Expected X-Request-ID header to be set")
		}

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	t.Run("Skip health check paths", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/healthz", nil)
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)

		// Health check should still work
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
}

// TestMiddlewareChain tests basic middleware chaining
func TestMiddlewareChain(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create middlewares
	securityConfig := middleware.DefaultSecurityHeadersConfig()
	securityConfig.EnableHSTS = true
	securityHeaders := middleware.NewSecurityHeaders(securityConfig, logger)

	redactConfig := middleware.DefaultRedactLoggerConfig()
	redactLogger, err := middleware.NewRedactLogger(redactConfig, logger)
	if err != nil {
		t.Fatalf("Failed to create redact logger: %v", err)
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Chain middlewares: redact logger first, then security headers
	handler := securityHeaders.Middleware(redactLogger.Middleware(testHandler))

	t.Run("Chained middlewares", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.TLS = &tls.ConnectionState{} // Simulate TLS
		req.Header.Set("Authorization", "Bearer secret")
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)

		// Should have both middleware effects
		if rr.Header().Get("X-Frame-Options") == "" {
			t.Error("Expected security headers to be applied")
		}
		
		if rr.Header().Get("X-Request-ID") == "" {
			t.Error("Expected correlation ID from redact logger")
		}

		if rr.Header().Get("Strict-Transport-Security") == "" {
			t.Error("Expected HSTS header with TLS")
		}

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
}