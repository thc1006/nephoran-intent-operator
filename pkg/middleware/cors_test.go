package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCORSMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowCredentials: false,
		MaxAge:           time.Hour,
	}

	middleware := NewCORSMiddleware(config, logger)
	
	assert.NotNil(t, middleware)
	assert.Equal(t, []string{"https://example.com"}, middleware.allowedOrigins)
	assert.Equal(t, []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, middleware.allowedMethods)
	assert.Equal(t, []string{"Content-Type", "Authorization", "X-Requested-With", "Accept", "Origin"}, middleware.allowedHeaders)
	assert.False(t, middleware.allowCredentials)
	assert.Equal(t, 3600, middleware.maxAge)
}

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com", "https://test.com"},
		MaxAge:         time.Hour,
	}
	
	corsMiddleware := NewCORSMiddleware(config, logger)
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	// Test allowed origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	
	corsMiddleware.Middleware(handler).ServeHTTP(w, req)
	
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization, X-Requested-With, Accept, Origin", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "3600", w.Header().Get("Access-Control-Max-Age"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		MaxAge:         time.Hour,
	}
	
	corsMiddleware := NewCORSMiddleware(config, logger)
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	// Test disallowed origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://malicious.com")
	w := httptest.NewRecorder()
	
	corsMiddleware.Middleware(handler).ServeHTTP(w, req)
	
	// No CORS headers should be set for disallowed origins
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, w.Code) // Request still processed, but no CORS headers
}

func TestCORSMiddleware_WildcardOrigin(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	config := CORSConfig{
		AllowedOrigins: []string{"*"},
		MaxAge:         time.Hour,
	}
	
	corsMiddleware := NewCORSMiddleware(config, logger)
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	// Test wildcard allows any origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://any-origin.com")
	w := httptest.NewRecorder()
	
	corsMiddleware.Middleware(handler).ServeHTTP(w, req)
	
	assert.Equal(t, "https://any-origin.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		MaxAge:         time.Hour,
	}
	
	corsMiddleware := NewCORSMiddleware(config, logger)
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for preflight requests")
	})
	
	// Test preflight request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w := httptest.NewRecorder()
	
	corsMiddleware.Middleware(handler).ServeHTTP(w, req)
	
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCORSMiddleware_PreflightDisallowedMethod(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{"GET", "POST"},
		MaxAge:         time.Hour,
	}
	
	corsMiddleware := NewCORSMiddleware(config, logger)
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for blocked preflight requests")
	})
	
	// Test preflight request with disallowed method
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "DELETE")
	w := httptest.NewRecorder()
	
	corsMiddleware.Middleware(handler).ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCORSMiddleware_NoOriginHeader(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		MaxAge:         time.Hour,
	}
	
	corsMiddleware := NewCORSMiddleware(config, logger)
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	// Test request without Origin header (same-origin request)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	corsMiddleware.Middleware(handler).ServeHTTP(w, req)
	
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCORSMiddleware_WithCredentials(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	config := CORSConfig{
		AllowedOrigins:   []string{"https://example.com"},
		AllowCredentials: true,
		MaxAge:           time.Hour,
	}
	
	corsMiddleware := NewCORSMiddleware(config, logger)
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	
	corsMiddleware.Middleware(handler).ServeHTTP(w, req)
	
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestValidateConfig_WildcardWithCredentials(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	}
	
	err := ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wildcard origin '*' cannot be used with credentials enabled")
}

func TestValidateConfig_PartialWildcard(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"https://*.example.com"},
	}
	
	err := ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "partial wildcard origins like 'https://*.example.com' are not supported")
}

func TestValidateConfig_InvalidOriginFormat(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins: []string{"invalid-origin"},
	}
	
	err := ValidateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "origin 'invalid-origin' must start with http:// or https://")
}

func TestValidateConfig_ValidConfiguration(t *testing.T) {
	config := CORSConfig{
		AllowedOrigins:   []string{"https://example.com", "http://localhost:3000"},
		AllowCredentials: false,
	}
	
	err := ValidateConfig(config)
	assert.NoError(t, err)
}

func TestCORSMiddleware_ExposedHeaders(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		ExposedHeaders: []string{"X-Custom-Header", "X-Another-Header"},
		MaxAge:         time.Hour,
	}
	
	corsMiddleware := NewCORSMiddleware(config, logger)
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	
	corsMiddleware.Middleware(handler).ServeHTTP(w, req)
	
	assert.Equal(t, "X-Custom-Header, X-Another-Header", w.Header().Get("Access-Control-Expose-Headers"))
}

func TestIsOriginAllowed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	config := CORSConfig{
		AllowedOrigins: []string{"https://example.com", "https://test.com"},
	}
	
	corsMiddleware := NewCORSMiddleware(config, logger)
	
	tests := []struct {
		origin   string
		expected bool
	}{
		{"https://example.com", true},
		{"https://test.com", true},
		{"https://malicious.com", false},
		{"http://example.com", false}, // Different protocol
		{"", true}, // Empty origin (same-origin requests)
	}
	
	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			result := corsMiddleware.isOriginAllowed(tt.origin)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsMethodAllowed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	config := CORSConfig{
		AllowedMethods: []string{"GET", "POST", "PUT"},
	}
	
	corsMiddleware := NewCORSMiddleware(config, logger)
	
	tests := []struct {
		method   string
		expected bool
	}{
		{"GET", true},
		{"POST", true},
		{"PUT", true},
		{"DELETE", false},
		{"PATCH", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			result := corsMiddleware.isMethodAllowed(tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}