//go:build integration

package main

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/middleware"
)

// TestRateLimitIntegration tests the rate limiter integration with the main server setup
func TestRateLimitIntegration(t *testing.T) {
	// Create a test configuration with rate limiting enabled
	cfg := config.DefaultLLMProcessorConfig()
	cfg.RateLimitEnabled = true
	cfg.RateLimitQPS = 2
	cfg.RateLimitBurstTokens = 3

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Initialize rate limiter (similar to main.go)
	var postRateLimiter *middleware.PostOnlyRateLimiter
	if cfg.RateLimitEnabled {
		rateLimiterConfig := middleware.RateLimiterConfig{
			QPS:             cfg.RateLimitQPS,
			Burst:           cfg.RateLimitBurstTokens,
			CleanupInterval: 10 * time.Minute,
			IPTimeout:       1 * time.Hour,
		}
		postRateLimiter = middleware.NewPostOnlyRateLimiter(rateLimiterConfig, logger)
	}
	defer func() {
		if postRateLimiter != nil {
			postRateLimiter.Stop()
		}
	}()

	// Create router and apply middleware (similar to main.go setup)
	router := mux.NewRouter()

	// Apply rate limiter middleware
	if postRateLimiter != nil {
		router.Use(postRateLimiter.Middleware)
	}

	// Add test endpoints
	router.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Process endpoint"))
	}).Methods("POST")

	router.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Stream endpoint"))
	}).Methods("POST")

	router.HandleFunc("/process_intent", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Process intent endpoint"))
	}).Methods("POST")

	router.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Health check"))
	}).Methods("GET")

	router.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Metrics"))
	}).Methods("GET")

	// Test 1: GET requests should not be rate limited
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/healthz", nil)
		req.RemoteAddr = "127.0.0.1:8080"

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GET request %d should not be rate limited, got status %d", i+1, w.Code)
		}
	}

	// Test 2: POST requests should be rate limited after burst capacity
	postEndpoints := []string{"/process", "/stream", "/process_intent"}

	for _, endpoint := range postEndpoints {
		t.Run("POST_"+strings.TrimPrefix(endpoint, "/"), func(t *testing.T) {
			// First 3 POST requests should pass (burst capacity)
			for i := 0; i < 3; i++ {
				req := httptest.NewRequest("POST", endpoint, bytes.NewReader([]byte("{}")))
				req.Header.Set("Content-Type", "application/json")
				req.RemoteAddr = "192.168.1.100:8080" // Different IP for each endpoint test

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("POST request %d to %s should have passed, got status %d, body: %s", i+1, endpoint, w.Code, w.Body.String())
				}

				// Check that rate limit headers are set
				if w.Header().Get("X-RateLimit-Limit") == "" {
					t.Errorf("X-RateLimit-Limit header should be set for POST request to %s", endpoint)
				}
			}

			// Fourth POST request should be rate limited
			req := httptest.NewRequest("POST", endpoint, bytes.NewReader([]byte("{}")))
			req.Header.Set("Content-Type", "application/json")
			req.RemoteAddr = "192.168.1.100:8080"

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusTooManyRequests {
				t.Errorf("Fourth POST request to %s should be rate limited, got status %d, body: %s", endpoint, w.Code, w.Body.String())
			}

			// Check rate limit headers for rejected request
			if w.Header().Get("X-RateLimit-Remaining") != "0" {
				t.Errorf("X-RateLimit-Remaining should be '0' for rate limited request to %s, got '%s'", endpoint, w.Header().Get("X-RateLimit-Remaining"))
			}
		})
	}

	// Test 3: Different IPs should have separate rate limits
	req1 := httptest.NewRequest("POST", "/process", bytes.NewReader([]byte("{}")))
	req1.Header.Set("Content-Type", "application/json")
	req1.RemoteAddr = "10.0.0.1:8080"

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("POST request from different IP should not be affected by previous rate limiting, got status %d", w1.Code)
	}

	// Test 4: Verify rate limiter statistics are working
	if postRateLimiter != nil {
		stats := postRateLimiter.GetStats()
		if stats["qps_limit"] != cfg.RateLimitQPS {
			t.Errorf("Expected QPS limit to be %d, got %v", cfg.RateLimitQPS, stats["qps_limit"])
		}
		if stats["burst_limit"] != cfg.RateLimitBurstTokens {
			t.Errorf("Expected burst limit to be %d, got %v", cfg.RateLimitBurstTokens, stats["burst_limit"])
		}
	}
}

// TestRateLimitDisabled tests that when rate limiting is disabled, no rate limiting occurs
func TestRateLimitDisabled(t *testing.T) {
	// Create a test configuration with rate limiting disabled
	cfg := config.DefaultLLMProcessorConfig()
	cfg.RateLimitEnabled = false

	// Create logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	// Initialize rate limiter (should be nil)
	var postRateLimiter *middleware.PostOnlyRateLimiter
	if cfg.RateLimitEnabled {
		rateLimiterConfig := middleware.RateLimiterConfig{
			QPS:             cfg.RateLimitQPS,
			Burst:           cfg.RateLimitBurstTokens,
			CleanupInterval: 10 * time.Minute,
			IPTimeout:       1 * time.Hour,
		}
		postRateLimiter = middleware.NewPostOnlyRateLimiter(rateLimiterConfig, logger)
	}

	// Verify rate limiter is not initialized
	if postRateLimiter != nil {
		t.Fatal("Rate limiter should be nil when disabled")
	}

	// Create router and apply middleware (should not apply rate limiting)
	router := mux.NewRouter()

	// Apply rate limiter middleware (should be skipped)
	if postRateLimiter != nil {
		router.Use(postRateLimiter.Middleware)
	}

	// Add test endpoint
	router.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Process endpoint"))
	}).Methods("POST")

	// Make many POST requests - none should be rate limited
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/process", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "127.0.0.1:8080"

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("POST request %d should not be rate limited when rate limiting is disabled, got status %d", i+1, w.Code)
		}

		// Rate limit headers should not be set when rate limiting is disabled
		if w.Header().Get("X-RateLimit-Limit") != "" {
			t.Errorf("X-RateLimit-Limit header should not be set when rate limiting is disabled")
		}
	}
}
