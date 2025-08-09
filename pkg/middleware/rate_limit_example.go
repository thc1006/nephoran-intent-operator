// Example usage of the rate limiter middleware
// This file demonstrates how to use the IP-based rate limiter
package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

func ExampleRateLimiterUsage() {
	// Create logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Configure rate limiter
	config := RateLimiterConfig{
		QPS:             20,               // Allow 20 requests per second per IP
		Burst:           40,               // Allow burst of 40 requests
		CleanupInterval: 10 * time.Minute, // Clean up old IP entries every 10 minutes
		IPTimeout:       1 * time.Hour,    // Remove IP entries after 1 hour of inactivity
	}

	// Create rate limiter
	rateLimiter := NewRateLimiter(config, logger)
	defer rateLimiter.Stop()

	// Create HTTP handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	// Apply rate limiting middleware
	limitedHandler := rateLimiter.Middleware(handler)

	// Create router and add the rate-limited handler
	router := mux.NewRouter()
	router.Handle("/api", limitedHandler).Methods("GET", "POST")

	// Start server
	http.ListenAndServe(":8080", router)
}

func ExamplePostOnlyRateLimiterUsage() {
	// Create logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Configure rate limiter for POST requests only
	config := RateLimiterConfig{
		QPS:             10,               // Allow 10 POST requests per second per IP
		Burst:           20,               // Allow burst of 20 POST requests
		CleanupInterval: 10 * time.Minute, // Clean up old IP entries every 10 minutes
		IPTimeout:       1 * time.Hour,    // Remove IP entries after 1 hour of inactivity
	}

	// Create POST-only rate limiter
	postRateLimiter := NewPostOnlyRateLimiter(config, logger)
	defer postRateLimiter.Stop()

	// Create HTTP handlers
	postHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("POST request processed"))
	})

	getHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("GET request processed"))
	})

	// Create router
	router := mux.NewRouter()

	// Apply POST-only rate limiting middleware to the entire router
	router.Use(postRateLimiter.Middleware)

	// Add handlers - only POST requests will be rate limited
	router.Handle("/api/data", postHandler).Methods("POST")
	router.Handle("/api/data", getHandler).Methods("GET")

	// Start server
	http.ListenAndServe(":8080", router)
}

/*
Rate Limiter Features:

1. IP-based Rate Limiting:
   - Each client IP address has its own rate limit bucket
   - Uses token bucket algorithm with configurable QPS and burst capacity

2. Memory Management:
   - Automatic cleanup of inactive IP entries to prevent memory leaks
   - Configurable cleanup interval and IP timeout

3. HTTP Headers:
   - Sets X-RateLimit-Limit header with the QPS limit
   - Sets X-RateLimit-Remaining header with approximate remaining tokens
   - Sets Retry-After header when rate limit is exceeded

4. Proxy Support:
   - Supports X-Forwarded-For header for load balancers
   - Supports X-Real-IP header for proxies
   - Supports CF-Connecting-IP header for Cloudflare
   - Falls back to RemoteAddr for direct connections

5. POST-only Filtering:
   - PostOnlyRateLimiter applies rate limiting only to POST requests
   - GET, HEAD, OPTIONS and other methods are not rate limited
   - Useful for protecting write operations while allowing read operations

6. Production Ready:
   - Thread-safe implementation using sync.Map
   - Comprehensive logging with structured logging (slog)
   - Graceful shutdown support
   - Statistics and monitoring support

Environment Variables (when integrated with LLM processor):
- RATE_LIMIT_ENABLED: Enable/disable rate limiting (default: true)
- RATE_LIMIT_QPS: Queries per second per IP (default: 20)
- RATE_LIMIT_BURST_TOKENS: Burst capacity per IP (default: 40)

Example Response Headers:
- X-RateLimit-Limit: 20
- X-RateLimit-Remaining: 15
- Retry-After: 1 (when rate limited)

HTTP Status Codes:
- 200: Request allowed
- 429: Too Many Requests (rate limited)
*/
