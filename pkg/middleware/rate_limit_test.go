package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestRateLimiter_BasicFunctionality(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	
	config := RateLimiterConfig{
		QPS:             100, // High QPS to allow refilling quickly
		Burst:           2,   // Only allow 2 requests in burst
		CleanupInterval: 1 * time.Minute,
		IPTimeout:       1 * time.Hour,
	}
	
	rateLimiter := NewRateLimiter(config, logger)
	defer rateLimiter.Stop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	limitedHandler := rateLimiter.Middleware(handler)
	
	// First two requests should pass (burst capacity)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "127.0.0.1:8080"
		
		w := httptest.NewRecorder()
		limitedHandler.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Request %d should have passed, got status %d, body: %s", i+1, w.Code, w.Body.String())
		}
		
		// Check that rate limit headers are set for successful requests
		if w.Header().Get("X-RateLimit-Limit") == "" {
			t.Errorf("X-RateLimit-Limit header should be set")
		}
	}
	
	// Third request should be rate limited immediately (burst exhausted, no time for refill)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	
	w := httptest.NewRecorder()
	limitedHandler.ServeHTTP(w, req)
	
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Third request should be rate limited, got status %d, body: %s", w.Code, w.Body.String())
	}
	
	// Check that rate limit headers are set for rejected requests
	if w.Header().Get("X-RateLimit-Limit") != "100" {
		t.Errorf("Expected X-RateLimit-Limit to be '100', got '%s'", w.Header().Get("X-RateLimit-Limit"))
	}
	if w.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Errorf("Expected X-RateLimit-Remaining to be '0', got '%s'", w.Header().Get("X-RateLimit-Remaining"))
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	
	config := RateLimiterConfig{
		QPS:             1,
		Burst:           1,
		CleanupInterval: 1 * time.Minute,
		IPTimeout:       1 * time.Hour,
	}
	
	rateLimiter := NewRateLimiter(config, logger)
	defer rateLimiter.Stop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	limitedHandler := rateLimiter.Middleware(handler)
	
	// Request from first IP should pass
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "127.0.0.1:8080"
	
	w1 := httptest.NewRecorder()
	limitedHandler.ServeHTTP(w1, req1)
	
	if w1.Code != http.StatusOK {
		t.Errorf("Request from first IP should have passed, got status %d", w1.Code)
	}
	
	// Request from second IP should also pass (different IP)
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = "192.168.1.1:8080"
	
	w2 := httptest.NewRecorder()
	limitedHandler.ServeHTTP(w2, req2)
	
	if w2.Code != http.StatusOK {
		t.Errorf("Request from second IP should have passed, got status %d", w2.Code)
	}
	
	// Second request from first IP should be rate limited
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.RemoteAddr = "127.0.0.1:8080"
	
	w3 := httptest.NewRecorder()
	limitedHandler.ServeHTTP(w3, req3)
	
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("Second request from first IP should be rate limited, got status %d", w3.Code)
	}
}

func TestPostOnlyRateLimiter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	
	config := RateLimiterConfig{
		QPS:             1,
		Burst:           1,
		CleanupInterval: 1 * time.Minute,
		IPTimeout:       1 * time.Hour,
	}
	
	postRateLimiter := NewPostOnlyRateLimiter(config, logger)
	defer postRateLimiter.Stop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	limitedHandler := postRateLimiter.Middleware(handler)
	
	// GET request should not be rate limited
	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = "127.0.0.1:8080"
	
	w1 := httptest.NewRecorder()
	limitedHandler.ServeHTTP(w1, req1)
	
	if w1.Code != http.StatusOK {
		t.Errorf("GET request should not be rate limited, got status %d", w1.Code)
	}
	
	// POST request should pass first time
	req2 := httptest.NewRequest("POST", "/", nil)
	req2.RemoteAddr = "127.0.0.1:8080"
	
	w2 := httptest.NewRecorder()
	limitedHandler.ServeHTTP(w2, req2)
	
	if w2.Code != http.StatusOK {
		t.Errorf("First POST request should pass, got status %d", w2.Code)
	}
	
	// Second POST request should be rate limited
	req3 := httptest.NewRequest("POST", "/", nil)
	req3.RemoteAddr = "127.0.0.1:8080"
	
	w3 := httptest.NewRecorder()
	limitedHandler.ServeHTTP(w3, req3)
	
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("Second POST request should be rate limited, got status %d", w3.Code)
	}
	
	// Another GET request should still pass (not rate limited)
	req4 := httptest.NewRequest("GET", "/", nil)
	req4.RemoteAddr = "127.0.0.1:8080"
	
	w4 := httptest.NewRecorder()
	limitedHandler.ServeHTTP(w4, req4)
	
	if w4.Code != http.StatusOK {
		t.Errorf("GET request after POST rate limiting should still pass, got status %d", w4.Code)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remoteAddr string
		expected string
	}{
		{
			name: "X-Forwarded-For single IP",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
			},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.1",
		},
		{
			name: "X-Forwarded-For multiple IPs",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1, 10.0.0.1, 172.16.0.1",
			},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.1",
		},
		{
			name: "X-Real-IP",
			headers: map[string]string{
				"X-Real-IP": "192.168.1.1",
			},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.1",
		},
		{
			name: "CF-Connecting-IP",
			headers: map[string]string{
				"CF-Connecting-IP": "192.168.1.1",
			},
			remoteAddr: "127.0.0.1:8080",
			expected:   "192.168.1.1",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "127.0.0.1:8080",
			expected:   "127.0.0.1",
		},
		{
			name:       "RemoteAddr without port",
			headers:    map[string]string{},
			remoteAddr: "127.0.0.1",
			expected:   "127.0.0.1",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			
			result := getClientIP(req)
			if result != tt.expected {
				t.Errorf("Expected IP %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRateLimiterStats(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	
	config := RateLimiterConfig{
		QPS:             5,
		Burst:           10,
		CleanupInterval: 1 * time.Minute,
		IPTimeout:       1 * time.Hour,
	}
	
	rateLimiter := NewRateLimiter(config, logger)
	defer rateLimiter.Stop()
	
	stats := rateLimiter.GetStats()
	
	if stats["qps_limit"] != 5 {
		t.Errorf("Expected QPS limit to be 5, got %v", stats["qps_limit"])
	}
	
	if stats["burst_limit"] != 10 {
		t.Errorf("Expected burst limit to be 10, got %v", stats["burst_limit"])
	}
	
	if stats["active_ips"] != 0 {
		t.Errorf("Expected active IPs to be 0 initially, got %v", stats["active_ips"])
	}
	
	// Make a request to create an active IP entry
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	limitedHandler := rateLimiter.Middleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:8080"
	
	w := httptest.NewRecorder()
	limitedHandler.ServeHTTP(w, req)
	
	stats = rateLimiter.GetStats()
	if stats["active_ips"] != 1 {
		t.Errorf("Expected active IPs to be 1 after request, got %v", stats["active_ips"])
	}
}

// Benchmark tests to verify performance
func BenchmarkRateLimiter(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	
	config := RateLimiterConfig{
		QPS:             1000,
		Burst:           1000,
		CleanupInterval: 1 * time.Minute,
		IPTimeout:       1 * time.Hour,
	}
	
	rateLimiter := NewRateLimiter(config, logger)
	defer rateLimiter.Stop()
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	limitedHandler := rateLimiter.Middleware(handler)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = fmt.Sprintf("127.0.0.%d:8080", b.N%255+1)
			
			w := httptest.NewRecorder()
			limitedHandler.ServeHTTP(w, req)
		}
	})
}