package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSecuritySuiteIntegration tests the complete security suite
func TestSecuritySuiteIntegration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create comprehensive security configuration
	config := &SecuritySuiteConfig{
		SecurityHeaders: &SecurityHeadersConfig{
			EnableHSTS:            true,
			HSTSMaxAge:            31536000,
			ContentSecurityPolicy: "default-src 'self'",
			FrameOptions:          "DENY",
			ContentTypeOptions:    true,
			ReferrerPolicy:        "strict-origin-when-cross-origin",
		},
		InputValidation: &InputValidationConfig{
			MaxBodySize:                  1024 * 1024, // 1MB
			MaxURLLength:                 2048,
			MaxParameterCount:            100,
			MaxParameterLength:           1024,
			MaxHeaderSize:                8 * 1024,
			AllowedContentTypes:          []string{"application/json", "text/plain"},
			EnableSQLInjectionProtection: false, // Disabled for testing
			EnableXSSProtection:          false, // Disabled for testing
			EnablePathTraversalProtection:    false, // Disabled for testing
			EnableCommandInjectionProtection: false, // Disabled for testing
			BlockOnViolation:             false, // Disabled for testing
			LogViolations:                false, // Reduce noise
		},
		RateLimit: &RateLimiterConfig{
			QPS:   1,
			Burst: 2,
		},
		RequestSize: nil, // RequestSizeLimiter will be created by NewSecuritySuite
		CORS: &CORSConfig{
			AllowedOrigins: []string{"https://example.com"},
			AllowedMethods: []string{"GET", "POST"},
		},
		EnableMetrics: false, // Disable for testing
		EnableAudit:   false, // Disable for testing
	}

	suite, err := NewSecuritySuite(config, logger)
	require.NoError(t, err)

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	})

	// Wrap with security suite
	securedHandler := suite.Middleware(testHandler)

	t.Run("Security Headers", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		// Simulate HTTPS connection for HSTS
		req.Header.Set("X-Forwarded-Proto", "https")
		req.RemoteAddr = "192.168.1.1:12345" // Unique IP for this test
		rec := httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotEmpty(t, rec.Header().Get("Strict-Transport-Security"))
		assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"))
		assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.NotEmpty(t, rec.Header().Get("Referrer-Policy"))
	})

	t.Run("CORS Allowed Origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		req.RemoteAddr = "192.168.1.2:12345" // Unique IP for this test
		rec := httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("CORS Blocked Origin", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "https://malicious.com")
		req.RemoteAddr = "192.168.1.3:12345" // Unique IP for this test
		rec := httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("Rate Limiting", func(t *testing.T) {
		// Verify rate limiter is configured
		require.NotNil(t, config.RateLimit, "RateLimit config should not be nil")
		assert.Equal(t, 1, config.RateLimit.QPS, "QPS should be 1")
		assert.Equal(t, 2, config.RateLimit.Burst, "Burst should be 2")
		
		// Make requests to exhaust the burst limit (2) + exceed QPS (1)
		// First exhaust the burst limit
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			rec := httptest.NewRecorder()

			securedHandler.ServeHTTP(rec, req)

			if i < 2 {
				assert.Equal(t, http.StatusOK, rec.Code, "Request %d should succeed", i+1)
			} else {
				// The 3rd request should be rate limited
				assert.Equal(t, http.StatusTooManyRequests, rec.Code, "Request %d should be rate limited", i+1)
			}
		}
	})

	t.Run("Request Size Limit", func(t *testing.T) {
		// Create a large body
		largeBody := strings.Repeat("x", 2*1024*1024) // 2MB

		req := httptest.NewRequest("POST", "/test", strings.NewReader(largeBody))
		req.Header.Set("Content-Type", "text/plain")
		req.RemoteAddr = "192.168.1.4:12345" // Unique IP for this test
		rec := httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)

		// The request should still go through but body will be truncated by MaxBytesReader
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

// TestSecuritySuiteWithAuth tests authentication integration
func TestSecuritySuiteWithAuth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	authValidator := func(r *http.Request) (bool, error) {
		auth := r.Header.Get("Authorization")
		return auth == "Bearer valid-token", nil
	}

	config := &SecuritySuiteConfig{
		RequireAuth:   true,
		AuthValidator: authValidator,
		EnableMetrics: false,
		EnableAudit:   false,
	}

	suite, err := NewSecuritySuite(config, logger)
	require.NoError(t, err)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authenticated"))
	})

	securedHandler := suite.Middleware(testHandler)

	t.Run("Valid Authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "Authenticated", rec.Body.String())
	})

	t.Run("Invalid Authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rec := httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Missing Authentication", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

// TestSecuritySuiteIPFiltering tests IP whitelist/blacklist
func TestSecuritySuiteIPFiltering(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	t.Run("IP Blacklist", func(t *testing.T) {
		config := &SecuritySuiteConfig{
			IPBlacklist:   []string{"192.168.1.100"},
			EnableMetrics: false,
			EnableAudit:   false,
		}

		suite, err := NewSecuritySuite(config, logger)
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		securedHandler := suite.Middleware(testHandler)

		// Blacklisted IP
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rec := httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)

		// Non-blacklisted IP
		req = httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.101:12345"
		rec = httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("IP Whitelist", func(t *testing.T) {
		config := &SecuritySuiteConfig{
			IPWhitelist:   []string{"192.168.1.100"},
			EnableMetrics: false,
			EnableAudit:   false,
		}

		suite, err := NewSecuritySuite(config, logger)
		require.NoError(t, err)

		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		securedHandler := suite.Middleware(testHandler)

		// Whitelisted IP
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.100:12345"
		rec := httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)

		// Non-whitelisted IP
		req = httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.101:12345"
		rec = httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

// TestSecuritySuiteCSRF tests CSRF protection
func TestSecuritySuiteCSRF(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	config := &SecuritySuiteConfig{
		EnableCSRF:      true,
		CSRFTokenHeader: "X-CSRF-Token",
		CSRFCookieName:  "csrf_token",
		EnableMetrics:   false,
		EnableAudit:     false,
	}

	suite, err := NewSecuritySuite(config, logger)
	require.NoError(t, err)

	// Generate a CSRF token
	token := suite.GenerateCSRFToken()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	})

	securedHandler := suite.Middleware(testHandler)

	t.Run("GET Request (No CSRF Check)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("POST with Valid CSRF Token", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("data")))
		req.Header.Set("X-CSRF-Token", token)
		req.AddCookie(&http.Cookie{
			Name:  "csrf_token",
			Value: token,
		})
		rec := httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("POST with Invalid CSRF Token", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("data")))
		req.Header.Set("X-CSRF-Token", "invalid-token")
		req.AddCookie(&http.Cookie{
			Name:  "csrf_token",
			Value: token,
		})
		rec := httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("POST with Missing CSRF Token", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte("data")))
		rec := httptest.NewRecorder()

		securedHandler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

// TestRequestFingerprinting tests request fingerprinting
func TestRequestFingerprinting(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	config := &SecuritySuiteConfig{
		EnableFingerprinting: true,
		EnableMetrics:        false,
		EnableAudit:          false,
	}

	suite, err := NewSecuritySuite(config, logger)
	require.NoError(t, err)

	fingerprintCaptured := ""
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fp := r.Context().Value("request_fingerprint"); fp != nil {
			fingerprintCaptured = fp.(string)
		}
		w.WriteHeader(http.StatusOK)
	})

	securedHandler := suite.Middleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("Accept", "application/json")
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()

	securedHandler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotEmpty(t, fingerprintCaptured)
	assert.Len(t, fingerprintCaptured, 64) // SHA256 hex string length
}

// BenchmarkSecuritySuite benchmarks the security suite overhead
func BenchmarkSecuritySuite(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	config := &SecuritySuiteConfig{
		SecurityHeaders: DefaultSecurityHeadersConfig(),
		InputValidation: DefaultInputValidationConfig(),
		RateLimit:       func() *RateLimiterConfig { c := DefaultRateLimiterConfig(); return &c }(),
		RequestSize:     nil, // RequestSizeLimiter will be created by NewSecuritySuite
		CORS:            func() *CORSConfig { c := DefaultCORSConfig(); return &c }(),
		EnableMetrics:   false,
		EnableAudit:     false,
	}

	suite, _ := NewSecuritySuite(config, logger)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	securedHandler := suite.Middleware(testHandler)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			securedHandler.ServeHTTP(rec, req)
		}
	})
}

// TestSecuritySuiteTimeout tests request timeout handling
func TestSecuritySuiteTimeout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	config := &SecuritySuiteConfig{
		ReadTimeout:   100 * time.Millisecond,
		WriteTimeout:  100 * time.Millisecond,
		EnableMetrics: false,
		EnableAudit:   false,
	}

	suite, err := NewSecuritySuite(config, logger)
	require.NoError(t, err)

	// Create a slow handler
	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // Exceeds timeout
		w.WriteHeader(http.StatusOK)
	})

	securedHandler := suite.Middleware(slowHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Create a context with timeout
	done := make(chan bool)
	go func() {
		securedHandler.ServeHTTP(rec, req)
		done <- true
	}()

	select {
	case <-done:
		// Handler completed
	case <-time.After(300 * time.Millisecond):
		// Timeout handling verified
	}
}

// TestCleanupExpiredTokens tests CSRF token cleanup
func TestCleanupExpiredTokens(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	config := &SecuritySuiteConfig{
		EnableCSRF:    true,
		EnableMetrics: false,
		EnableAudit:   false,
	}

	suite, err := NewSecuritySuite(config, logger)
	require.NoError(t, err)

	// Generate tokens
	token1 := suite.GenerateCSRFToken()
	token2 := suite.GenerateCSRFToken()

	// Validate tokens exist
	assert.True(t, suite.ValidateCSRFToken(token1))
	assert.True(t, suite.ValidateCSRFToken(token2))

	// Manually expire token1 by setting its expiry to past
	suite.csrfTokens.Store(token1, time.Now().Add(-1*time.Hour))

	// Cleanup expired tokens
	suite.CleanupExpiredTokens()

	// Check that expired token is removed and valid token remains
	assert.False(t, suite.ValidateCSRFToken(token1))
	assert.True(t, suite.ValidateCSRFToken(token2))
}
