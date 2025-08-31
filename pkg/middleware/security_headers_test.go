package middleware

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDefaultSecurityHeadersConfig tests the default configuration
func TestDefaultSecurityHeadersConfig(t *testing.T) {
	config := DefaultSecurityHeadersConfig()

	assert.False(t, config.EnableHSTS, "HSTS should be disabled by default")
	assert.Equal(t, 31536000, config.HSTSMaxAge, "HSTS max age should be 1 year")
	assert.False(t, config.HSTSIncludeSubDomains, "Include subdomains should be false by default")
	assert.False(t, config.HSTSPreload, "Preload should be false by default")
	assert.Equal(t, "DENY", config.FrameOptions, "Frame options should be DENY")
	assert.True(t, config.ContentTypeOptions, "Content type options should be enabled")
	assert.Equal(t, "strict-origin-when-cross-origin", config.ReferrerPolicy, "Referrer policy should be set")
	assert.NotEmpty(t, config.ContentSecurityPolicy, "CSP should have a default value")
	assert.NotEmpty(t, config.PermissionsPolicy, "Permissions policy should have a default value")
	assert.Equal(t, "1; mode=block", config.XSSProtection, "XSS protection should be enabled")
	assert.NotNil(t, config.CustomHeaders, "Custom headers map should be initialized")
}

// TestNewSecurityHeaders tests the constructor
func TestNewSecurityHeaders(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("with nil config", func(t *testing.T) {
		sh := NewSecurityHeaders(nil, logger)
		assert.NotNil(t, sh)
		assert.NotNil(t, sh.config)
		assert.Equal(t, "DENY", sh.config.FrameOptions)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &SecurityHeadersConfig{
			EnableHSTS:    true,
			HSTSMaxAge:    86400,
			FrameOptions:  "SAMEORIGIN",
			CustomHeaders: map[string]string{"X-Custom": "value"},
		}
		sh := NewSecurityHeaders(config, logger)
		assert.NotNil(t, sh)
		assert.True(t, sh.config.EnableHSTS)
		assert.Equal(t, 86400, sh.config.HSTSMaxAge)
		assert.Equal(t, "SAMEORIGIN", sh.config.FrameOptions)
	})

	t.Run("with invalid config values", func(t *testing.T) {
		config := &SecurityHeadersConfig{
			HSTSMaxAge:   -1,
			FrameOptions: "",
		}
		sh := NewSecurityHeaders(config, logger)
		assert.Equal(t, 31536000, sh.config.HSTSMaxAge, "Negative max age should be corrected")
		assert.Equal(t, "DENY", sh.config.FrameOptions, "Empty frame options should use default")
	})
}

// TestSecurityHeadersMiddleware tests the middleware functionality
func TestSecurityHeadersMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create a test handler that just returns 200 OK
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("basic security headers", func(t *testing.T) {
		config := &SecurityHeadersConfig{
			FrameOptions:          "DENY",
			ContentTypeOptions:    true,
			ReferrerPolicy:        "no-referrer",
			ContentSecurityPolicy: "default-src 'none'",
			PermissionsPolicy:     "geolocation=()",
			XSSProtection:         "1; mode=block",
		}

		sh := NewSecurityHeaders(config, logger)
		handler := sh.Middleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
		assert.Equal(t, "default-src 'none'", rec.Header().Get("Content-Security-Policy"))
		assert.Equal(t, "geolocation=()", rec.Header().Get("Permissions-Policy"))
		assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
	})

	t.Run("HSTS with HTTPS", func(t *testing.T) {
		config := &SecurityHeadersConfig{
			EnableHSTS:            true,
			HSTSMaxAge:            31536000,
			HSTSIncludeSubDomains: true,
			HSTSPreload:           true,
		}

		sh := NewSecurityHeaders(config, logger)
		handler := sh.Middleware(testHandler)

		req := httptest.NewRequest("GET", "https://example.com/test", nil)
		req.TLS = &tls.ConnectionState{} // Simulate HTTPS connection
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		hstsHeader := rec.Header().Get("Strict-Transport-Security")
		assert.Contains(t, hstsHeader, "max-age=31536000")
		assert.Contains(t, hstsHeader, "includeSubDomains")
		assert.Contains(t, hstsHeader, "preload")
	})

	t.Run("HSTS with HTTP (should not set)", func(t *testing.T) {
		config := &SecurityHeadersConfig{
			EnableHSTS: true,
			HSTSMaxAge: 31536000,
		}

		sh := NewSecurityHeaders(config, logger)
		handler := sh.Middleware(testHandler)

		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Empty(t, rec.Header().Get("Strict-Transport-Security"))
	})

	t.Run("custom headers", func(t *testing.T) {
		config := &SecurityHeadersConfig{
			CustomHeaders: map[string]string{
				"X-Custom-Header-1": "value1",
				"X-Custom-Header-2": "value2",
			},
		}

		sh := NewSecurityHeaders(config, logger)
		handler := sh.Middleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, "value1", rec.Header().Get("X-Custom-Header-1"))
		assert.Equal(t, "value2", rec.Header().Get("X-Custom-Header-2"))
	})
}

// TestIsSecureConnection tests the secure connection detection
func TestIsSecureConnection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sh := NewSecurityHeaders(nil, logger)

	t.Run("direct TLS connection", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.TLS = &tls.ConnectionState{}
		assert.True(t, sh.isSecureConnection(req))
	})

	t.Run("X-Forwarded-Proto header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		assert.True(t, sh.isSecureConnection(req))
	})

	t.Run("Forwarded header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Forwarded", "for=192.0.2.60;proto=https;by=203.0.113.43")
		assert.True(t, sh.isSecureConnection(req))
	})

	t.Run("HTTPS URL scheme", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://example.com/test", nil)
		assert.True(t, sh.isSecureConnection(req))
	})

	t.Run("HTTP connection", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.com/test", nil)
		assert.False(t, sh.isSecureConnection(req))
	})
}

// TestBuildHSTSHeader tests HSTS header construction
func TestBuildHSTSHeader(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name     string
		config   *SecurityHeadersConfig
		expected string
	}{
		{
			name: "basic HSTS",
			config: &SecurityHeadersConfig{
				HSTSMaxAge: 31536000,
			},
			expected: "max-age=31536000",
		},
		{
			name: "HSTS with subdomains",
			config: &SecurityHeadersConfig{
				HSTSMaxAge:            31536000,
				HSTSIncludeSubDomains: true,
			},
			expected: "max-age=31536000; includeSubDomains",
		},
		{
			name: "HSTS with preload",
			config: &SecurityHeadersConfig{
				HSTSMaxAge:            31536000,
				HSTSIncludeSubDomains: true,
				HSTSPreload:           true,
			},
			expected: "max-age=31536000; includeSubDomains; preload",
		},
		{
			name: "HSTS preload without subdomains (should not include preload)",
			config: &SecurityHeadersConfig{
				HSTSMaxAge:            31536000,
				HSTSIncludeSubDomains: false,
				HSTSPreload:           true,
			},
			expected: "max-age=31536000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sh := NewSecurityHeaders(tt.config, logger)
			result := sh.buildHSTSHeader()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestValidateConfig tests configuration validation
func TestValidateConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("valid configuration", func(t *testing.T) {
		config := &SecurityHeadersConfig{
			EnableHSTS:            true,
			HSTSMaxAge:            31536000,
			HSTSIncludeSubDomains: true,
			HSTSPreload:           true,
			FrameOptions:          "DENY",
		}
		sh := NewSecurityHeaders(config, logger)
		err := sh.ValidateConfig()
		assert.NoError(t, err)
	})

	t.Run("HSTS preload without includeSubDomains", func(t *testing.T) {
		config := &SecurityHeadersConfig{
			EnableHSTS:            true,
			HSTSMaxAge:            31536000,
			HSTSIncludeSubDomains: false,
			HSTSPreload:           true,
		}
		sh := NewSecurityHeaders(config, logger)
		err := sh.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "includeSubDomains")
	})

	t.Run("HSTS preload with insufficient max-age", func(t *testing.T) {
		config := &SecurityHeadersConfig{
			EnableHSTS:            true,
			HSTSMaxAge:            86400, // 1 day, too short for preload
			HSTSIncludeSubDomains: true,
			HSTSPreload:           true,
		}
		sh := NewSecurityHeaders(config, logger)
		err := sh.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "18 weeks")
	})

	t.Run("invalid X-Frame-Options", func(t *testing.T) {
		config := &SecurityHeadersConfig{
			FrameOptions: "INVALID",
		}
		sh := NewSecurityHeaders(config, logger)
		err := sh.ValidateConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "X-Frame-Options")
	})

	t.Run("ALLOW-FROM directive", func(t *testing.T) {
		config := &SecurityHeadersConfig{
			FrameOptions: "ALLOW-FROM https://example.com",
		}
		sh := NewSecurityHeaders(config, logger)
		err := sh.ValidateConfig()
		assert.NoError(t, err)
	})
}

// TestGetActiveHeaders tests getting active headers
func TestGetActiveHeaders(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	config := &SecurityHeadersConfig{
		EnableHSTS:            true,
		HSTSMaxAge:            31536000,
		FrameOptions:          "DENY",
		ContentTypeOptions:    true,
		ReferrerPolicy:        "no-referrer",
		ContentSecurityPolicy: "default-src 'none'",
		CustomHeaders: map[string]string{
			"X-Custom": "value",
		},
	}

	sh := NewSecurityHeaders(config, logger)

	t.Run("secure connection", func(t *testing.T) {
		headers := sh.GetActiveHeaders(true)
		assert.Contains(t, headers, "Strict-Transport-Security")
		assert.Contains(t, headers, "X-Frame-Options")
		assert.Contains(t, headers, "X-Content-Type-Options")
		assert.Contains(t, headers, "Referrer-Policy")
		assert.Contains(t, headers, "Content-Security-Policy")
		assert.Contains(t, headers, "X-Custom")
		assert.Equal(t, "value", headers["X-Custom"])
	})

	t.Run("insecure connection", func(t *testing.T) {
		headers := sh.GetActiveHeaders(false)
		assert.NotContains(t, headers, "Strict-Transport-Security")
		assert.Contains(t, headers, "X-Frame-Options")
		assert.Contains(t, headers, "X-Content-Type-Options")
	})
}

// TestReportOnlyCSP tests CSP report-only mode
func TestReportOnlyCSP(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	sh := NewSecurityHeaders(nil, logger)

	t.Run("CSP report-only without report URI", func(t *testing.T) {
		handler := sh.ReportOnlyCSP("default-src 'self'", "")
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		cspHeader := rec.Header().Get("Content-Security-Policy-Report-Only")
		assert.Equal(t, "default-src 'self'", cspHeader)
	})

	t.Run("CSP report-only with report URI", func(t *testing.T) {
		handler := sh.ReportOnlyCSP("default-src 'self'", "https://example.com/csp-report")
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		cspHeader := rec.Header().Get("Content-Security-Policy-Report-Only")
		assert.Contains(t, cspHeader, "default-src 'self'")
		assert.Contains(t, cspHeader, "report-uri https://example.com/csp-report")
	})
}

// BenchmarkSecurityHeadersMiddleware benchmarks the middleware performance
func BenchmarkSecurityHeadersMiddleware(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := DefaultSecurityHeadersConfig()
	config.EnableHSTS = true

	sh := NewSecurityHeaders(config, logger)
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := sh.Middleware(testHandler)

	req := httptest.NewRequest("GET", "https://example.com/test", nil)
	req.TLS = &tls.ConnectionState{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}
}

// TestSecurityHeadersIntegration tests the middleware in a more realistic scenario
func TestSecurityHeadersIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create a chain of middleware
	config := &SecurityHeadersConfig{
		EnableHSTS:            true,
		HSTSMaxAge:            31536000,
		HSTSIncludeSubDomains: true,
		FrameOptions:          "DENY",
		ContentTypeOptions:    true,
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline'",
		PermissionsPolicy:     "geolocation=(), microphone=()",
		XSSProtection:         "1; mode=block",
	}

	sh := NewSecurityHeaders(config, logger)

	// Create a test handler that writes JSON response
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Apply middleware
	handler := sh.Middleware(apiHandler)

	t.Run("all headers are set correctly", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://api.example.com/v1/status", nil)
		req.TLS = &tls.ConnectionState{}
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// Check response
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		// Check security headers
		assert.NotEmpty(t, rec.Header().Get("Strict-Transport-Security"))
		assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
		assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "default-src 'self'")
		assert.Contains(t, rec.Header().Get("Permissions-Policy"), "geolocation=()")
		assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))

		// Check response body
		assert.Equal(t, `{"status":"ok"}`, rec.Body.String())
	})

	t.Run("headers work with HEAD requests", func(t *testing.T) {
		req := httptest.NewRequest("HEAD", "https://api.example.com/v1/status", nil)
		req.TLS = &tls.ConnectionState{}
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotEmpty(t, rec.Header().Get("Strict-Transport-Security"))
		assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	})

	t.Run("headers work with OPTIONS requests", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "https://api.example.com/v1/status", nil)
		req.TLS = &tls.ConnectionState{}
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotEmpty(t, rec.Header().Get("Strict-Transport-Security"))
		assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	})
}

// TestSecurityHeadersConcurrency tests the middleware under concurrent load
func TestSecurityHeadersConcurrency(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := DefaultSecurityHeadersConfig()
	config.EnableHSTS = true

	sh := NewSecurityHeaders(config, logger)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := sh.Middleware(testHandler)

	// Run concurrent requests
	concurrency := 100
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			req := httptest.NewRequest("GET", "https://example.com/test", nil)
			req.TLS = &tls.ConnectionState{}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			// Verify headers are set
			assert.NotEmpty(t, rec.Header().Get("X-Frame-Options"))
			assert.NotEmpty(t, rec.Header().Get("X-Content-Type-Options"))

			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
}
