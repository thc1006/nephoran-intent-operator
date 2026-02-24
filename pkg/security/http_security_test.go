package security

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecureHTTPClient(t *testing.T) {
	tests := []struct {
		name           string
		timeout        time.Duration
		expectedConfig bool
	}{
		{
			name:           "with custom timeout",
			timeout:        10 * time.Second,
			expectedConfig: true,
		},
		{
			name:           "with zero timeout uses default",
			timeout:        0,
			expectedConfig: true,
		},
		{
			name:           "with short timeout",
			timeout:        1 * time.Second,
			expectedConfig: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := SecureHTTPClient(tt.timeout)
			require.NotNil(t, client)

			// Verify timeout is set correctly
			if tt.timeout == 0 {
				assert.Equal(t, 30*time.Second, client.Timeout)
			} else {
				assert.Equal(t, tt.timeout, client.Timeout)
			}

			// Verify transport is configured
			assert.NotNil(t, client.Transport)
			assert.NotNil(t, client.CheckRedirect)
		})
	}
}

func TestSecureHTTPClient_RedirectLimit(t *testing.T) {
	client := SecureHTTPClient(5 * time.Second)

	// Create mock requests to test redirect limit
	via := make([]*http.Request, 10)
	for i := 0; i < 10; i++ {
		via[i] = httptest.NewRequest("GET", "http://example.com", nil)
	}

	err := client.CheckRedirect(nil, via)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many redirects")
}

func TestSecureHTTPClient_RedirectAllowed(t *testing.T) {
	client := SecureHTTPClient(5 * time.Second)

	// Test with fewer redirects
	via := make([]*http.Request, 5)
	for i := 0; i < 5; i++ {
		via[i] = httptest.NewRequest("GET", "http://example.com", nil)
	}

	err := client.CheckRedirect(nil, via)
	assert.NoError(t, err)
}

func TestValidateHTTPURL(t *testing.T) {
	tests := []struct {
		name      string
		rawURL    string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid https URL",
			rawURL:    "https://example.com/path",
			wantError: false,
		},
		{
			name:      "valid http URL",
			rawURL:    "http://example.com",
			wantError: false,
		},
		{
			name:      "invalid scheme",
			rawURL:    "ftp://example.com",
			wantError: true,
			errorMsg:  "invalid URL scheme",
		},
		{
			name:      "URL with credentials",
			rawURL:    "https://user:pass@example.com",
			wantError: true,
			errorMsg:  "URLs with credentials are not allowed",
		},
		{
			name:      "localhost URL",
			rawURL:    "http://localhost:8080",
			wantError: true,
			errorMsg:  "local addresses are not allowed",
		},
		{
			name:      "127.0.0.1 URL",
			rawURL:    "http://127.0.0.1",
			wantError: true,
			errorMsg:  "local addresses are not allowed",
		},
		{
			name:      "private IP",
			rawURL:    "http://192.168.1.1",
			wantError: true,
			errorMsg:  "local addresses are not allowed",
		},
		{
			name:      "URL without host",
			rawURL:    "https://",
			wantError: true,
			errorMsg:  "URL must have a host",
		},
		{
			name:      "malformed URL",
			rawURL:    "ht!tp://invalid",
			wantError: true,
			errorMsg:  "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := ValidateHTTPURL(tt.rawURL)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, url)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, url)
			}
		})
	}
}

func TestIsLocalAddress(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected bool
	}{
		{"localhost", "localhost", true},
		{"127.0.0.1", "127.0.0.1", true},
		{"IPv6 loopback", "::1", true},
		{"0.0.0.0", "0.0.0.0", true},
		{"private IP 192.168", "192.168.1.1", true},
		{"private IP 10", "10.0.0.1", true},
		{"private IP 172", "172.16.0.1", true},
		{"public IP", "8.8.8.8", false},
		{"domain name", "example.com", false},
		{"with port", "localhost:8080", true},
		{"link local", "169.254.1.1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLocalAddress(tt.host)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSecureHTTPServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := SecureHTTPServer(":8080", handler)
	require.NotNil(t, server)

	assert.Equal(t, ":8080", server.Addr)
	assert.NotNil(t, server.Handler)
	assert.Equal(t, 15*time.Second, server.ReadTimeout)
	assert.Equal(t, 10*time.Second, server.ReadHeaderTimeout)
	assert.Equal(t, 15*time.Second, server.WriteTimeout)
	assert.Equal(t, 60*time.Second, server.IdleTimeout)
	assert.Equal(t, 1<<20, server.MaxHeaderBytes)
}

func TestBasicSecurityHeaders(t *testing.T) {
	handler := BasicSecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify security headers are set
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
	assert.Contains(t, rec.Header().Get("Strict-Transport-Security"), "max-age=31536000")
	assert.NotEmpty(t, rec.Header().Get("Content-Security-Policy"))
	assert.NotEmpty(t, rec.Header().Get("Referrer-Policy"))
	assert.NotEmpty(t, rec.Header().Get("Permissions-Policy"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "OK", rec.Body.String())
}

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(10, 1*time.Minute)
	require.NotNil(t, rl)

	assert.Equal(t, 10, rl.limit)
	assert.Equal(t, 1*time.Minute, rl.window)
	assert.NotNil(t, rl.requests)
}

func TestRateLimiter_Middleware(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		rl := NewRateLimiter(5, 1*time.Minute)
		handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Make requests within limit
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.0.2.1:1234"
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		rl := NewRateLimiter(3, 1*time.Minute)
		handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Make requests up to limit
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.0.2.1:1234"
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}

		// Next request should be rate limited
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	})

	t.Run("uses X-Forwarded-For header", func(t *testing.T) {
		rl := NewRateLimiter(2, 1*time.Minute)
		handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestSecureRequest(t *testing.T) {
	t.Run("creates request with valid URL", func(t *testing.T) {
		ctx := context.Background()
		req, err := SecureRequest(ctx, "GET", "https://example.com/api", 5*time.Second)

		require.NoError(t, err)
		require.NotNil(t, req)
		assert.Equal(t, "GET", req.Method)
		assert.Equal(t, "https://example.com/api", req.URL.String())
		assert.Equal(t, "Nephoran-Security-Client/1.0", req.Header.Get("User-Agent"))
		assert.Equal(t, "application/json", req.Header.Get("Accept"))
	})

	t.Run("rejects invalid URL", func(t *testing.T) {
		ctx := context.Background()
		// Use an invalid URL that will fail validation
		req, err := SecureRequest(ctx, "GET", "ftp://example.com", 5*time.Second)

		assert.Error(t, err)
		assert.Nil(t, req)
	})

	t.Run("handles timeout", func(t *testing.T) {
		ctx := context.Background()
		req, err := SecureRequest(ctx, "GET", "https://example.com", 100*time.Millisecond)

		require.NoError(t, err)
		require.NotNil(t, req)
		assert.NotNil(t, req.Context())
	})
}

func TestSafeCloseBody(t *testing.T) {
	t.Run("handles nil response", func(t *testing.T) {
		err := SafeCloseBody(nil)
		assert.NoError(t, err)
	})

	t.Run("handles nil body", func(t *testing.T) {
		resp := &http.Response{Body: nil}
		err := SafeCloseBody(resp)
		assert.NoError(t, err)
	})

	t.Run("closes valid body", func(t *testing.T) {
		resp := &http.Response{
			Body: io.NopCloser(strings.NewReader("test")),
		}
		err := SafeCloseBody(resp)
		assert.NoError(t, err)
	})
}

func TestRateLimiter_WindowExpiration(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make 2 requests
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.0.2.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should allow new requests
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
