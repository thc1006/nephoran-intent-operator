package middleware

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiterConfig holds configuration for the rate limiter.

type RateLimiterConfig struct {
	// QPS is the queries per second allowed per IP.

	QPS int

	// Burst is the burst capacity for token bucket per IP.

	Burst int

	// CleanupInterval is how often to clean up old IP entries (default: 10 minutes).

	CleanupInterval time.Duration

	// IPTimeout is how long to keep inactive IP entries (default: 1 hour).

	IPTimeout time.Duration
}

// DefaultRateLimiterConfig returns a configuration with sensible defaults.

func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		QPS: 20,

		Burst: 40,

		CleanupInterval: 10 * time.Minute,

		IPTimeout: 1 * time.Hour,
	}
}

// ipRateLimiter tracks the rate limiter and last access time for an IP.

type ipRateLimiter struct {
	limiter *rate.Limiter

	lastAccess time.Time
}

// RateLimiter implements IP-based rate limiting using token bucket algorithm.

type RateLimiter struct {
	config RateLimiterConfig

	limiters sync.Map // map[string]*ipRateLimiter - thread-safe map of IP -> rate limiter

	logger *slog.Logger

	stopCh chan struct{}

	wg sync.WaitGroup
}

// NewRateLimiter creates a new IP-based rate limiter with the given configuration.

func NewRateLimiter(config RateLimiterConfig, logger *slog.Logger) *RateLimiter {
	if config.QPS <= 0 {
		config.QPS = 20
	}

	if config.Burst <= 0 {
		config.Burst = 40
	}

	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 10 * time.Minute
	}

	if config.IPTimeout <= 0 {
		config.IPTimeout = 1 * time.Hour
	}

	rl := &RateLimiter{
		config: config,

		logger: logger.With(slog.String("component", "rate_limiter")),

		stopCh: make(chan struct{}),
	}

	// Start cleanup goroutine.

	rl.wg.Add(1)

	go rl.cleanupLoop()

	logger.Info("Rate limiter initialized",

		slog.Int("qps", config.QPS),

		slog.Int("burst", config.Burst),

		slog.Duration("cleanup_interval", config.CleanupInterval),

		slog.Duration("ip_timeout", config.IPTimeout))

	return rl
}

// getLimiterForIP gets or creates a rate limiter for the given IP address.

func (rl *RateLimiter) getLimiterForIP(ip string) *rate.Limiter {
	now := time.Now()

	// Try to get existing limiter.

	if existing, ok := rl.limiters.Load(ip); ok {

		limiterInfo, ok := existing.(*ipRateLimiter)
		if !ok {
			rl.logger.Error("Failed to cast existing limiter to ipRateLimiter", "ip", ip)
			// Fall through to create new limiter
		} else {
			// Update last access time.
			limiterInfo.lastAccess = now
			return limiterInfo.limiter
		}
	}

	// Create new limiter.

	limiter := rate.NewLimiter(rate.Limit(rl.config.QPS), rl.config.Burst)

	limiterInfo := &ipRateLimiter{
		limiter: limiter,

		lastAccess: now,
	}

	// Store it (LoadOrStore ensures atomicity).

	if actual, loaded := rl.limiters.LoadOrStore(ip, limiterInfo); loaded {

		// Another goroutine created it first, use that one.

		actualInfo, ok := actual.(*ipRateLimiter)
		if !ok {
			rl.logger.Error("Failed to cast actual limiter to ipRateLimiter", "ip", ip)
			// Return our own limiter as fallback
			return limiterInfo.limiter
		}

		actualInfo.lastAccess = now

		return actualInfo.limiter

	}

	// We created it successfully.

	rl.logger.Debug("Created new rate limiter for IP",

		slog.String("ip", ip),

		slog.Int("qps", rl.config.QPS),

		slog.Int("burst", rl.config.Burst))

	return limiter
}

// Middleware returns an HTTP middleware that enforces rate limiting per IP address.

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP.

		clientIP := getClientIP(r)

		if clientIP == "" {

			rl.logger.Warn("Unable to determine client IP for rate limiting",

				slog.String("remote_addr", r.RemoteAddr),

				slog.String("path", r.URL.Path))

			// Allow request to proceed if we can't determine IP.

			next.ServeHTTP(w, r)

			return

		}

		// Get rate limiter for this IP.

		limiter := rl.getLimiterForIP(clientIP)

		// Check if request is allowed.

		if !limiter.Allow() {

			rl.logger.Warn("Rate limit exceeded",

				slog.String("client_ip", clientIP),

				slog.String("method", r.Method),

				slog.String("path", r.URL.Path),

				slog.String("user_agent", r.Header.Get("User-Agent")))

			// Set rate limit headers for client.

			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.QPS))

			w.Header().Set("X-RateLimit-Remaining", "0")

			w.Header().Set("Retry-After", "1")

			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)

			return

		}

		// Request allowed, add rate limit headers.

		// Calculate remaining tokens (approximate) - tokens may be fractional after Allow().

		tokens := limiter.Tokens()

		remaining := int(tokens)

		if remaining < 0 {
			remaining = 0
		}

		// Add one back since we consumed one for this request.

		if remaining < rl.config.Burst {
			remaining++
		}

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.QPS))

		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		rl.logger.Debug("Rate limit check passed",

			slog.String("client_ip", clientIP),

			slog.String("method", r.Method),

			slog.String("path", r.URL.Path))

		// Proceed with request.

		next.ServeHTTP(w, r)
	})
}

// MiddlewareFunc returns a middleware function that can be used with router.Use().

func (rl *RateLimiter) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return rl.Middleware(next).ServeHTTP
}

// cleanupLoop runs periodically to remove old IP entries to prevent memory leaks.

func (rl *RateLimiter) cleanupLoop() {
	defer rl.wg.Done()

	ticker := time.NewTicker(rl.config.CleanupInterval)

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			rl.cleanup()

		case <-rl.stopCh:

			return

		}
	}
}

// cleanup removes old IP entries that haven't been accessed recently.

func (rl *RateLimiter) cleanup() {
	now := time.Now()

	cutoff := now.Add(-rl.config.IPTimeout)

	removedCount := 0

	rl.limiters.Range(func(key, value interface{}) bool {
		ip, ok := key.(string)
		if !ok {
			rl.logger.Error("Failed to cast key to string during cleanup")
			return true // Continue iteration
		}

		limiterInfo, ok := value.(*ipRateLimiter)
		if !ok {
			rl.logger.Error("Failed to cast value to ipRateLimiter during cleanup", "ip", ip)
			return true // Continue iteration
		}

		if limiterInfo.lastAccess.Before(cutoff) {

			rl.limiters.Delete(ip)

			removedCount++

		}

		return true
	})

	if removedCount > 0 {
		rl.logger.Debug("Cleaned up old rate limiter entries",

			slog.Int("removed_count", removedCount),

			slog.Time("cutoff_time", cutoff))
	}
}

// Stop gracefully shuts down the rate limiter.

func (rl *RateLimiter) Stop() {
	close(rl.stopCh)

	rl.wg.Wait()

	rl.logger.Info("Rate limiter stopped")
}

// GetStats returns statistics about the rate limiter.

func (rl *RateLimiter) GetStats() map[string]interface{} {
	activeIPs := 0

	rl.limiters.Range(func(key, value interface{}) bool {
		activeIPs++

		return true
	})

	return map[string]interface{}{
		"active_ips": activeIPs,

		"qps_limit": rl.config.QPS,

		"burst_limit": rl.config.Burst,

		"cleanup_interval": rl.config.CleanupInterval.String(),

		"ip_timeout": rl.config.IPTimeout.String(),
	}
}

// getClientIP extracts the real client IP address from various headers.

// This handles common proxy scenarios including load balancers and reverse proxies.

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (most common).

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {

		// X-Forwarded-For can contain multiple IPs, use the first one.

		ips := strings.Split(xff, ",")

		ip := strings.TrimSpace(ips[0])

		if ip != "" && net.ParseIP(ip) != nil {
			return ip
		}

	}

	// Check X-Real-IP header (used by some proxies).

	if xri := r.Header.Get("X-Real-IP"); xri != "" {

		ip := strings.TrimSpace(xri)

		if ip != "" && net.ParseIP(ip) != nil {
			return ip
		}

	}

	// Check CF-Connecting-IP header (Cloudflare).

	if cfip := r.Header.Get("CF-Connecting-IP"); cfip != "" {

		ip := strings.TrimSpace(cfip)

		if ip != "" && net.ParseIP(ip) != nil {
			return ip
		}

	}

	// Fall back to RemoteAddr (direct connection).

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {

		// RemoteAddr might not have port (e.g., in tests).

		if net.ParseIP(r.RemoteAddr) != nil {
			return r.RemoteAddr
		}

		return ""

	}

	if net.ParseIP(ip) != nil {
		return ip
	}

	return ""
}

// PostOnlyRateLimiter wraps the rate limiter to only apply to POST requests.

type PostOnlyRateLimiter struct {
	rateLimiter *RateLimiter

	logger *slog.Logger
}

// NewPostOnlyRateLimiter creates a rate limiter that only applies to POST requests.

func NewPostOnlyRateLimiter(config RateLimiterConfig, logger *slog.Logger) *PostOnlyRateLimiter {
	return &PostOnlyRateLimiter{
		rateLimiter: NewRateLimiter(config, logger),

		logger: logger.With(slog.String("component", "post_only_rate_limiter")),
	}
}

// Middleware returns an HTTP middleware that enforces rate limiting only on POST requests.

func (prl *PostOnlyRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only apply rate limiting to POST requests.

		if r.Method != http.MethodPost {

			prl.logger.Debug("Skipping rate limit for non-POST request",

				slog.String("method", r.Method),

				slog.String("path", r.URL.Path))

			next.ServeHTTP(w, r)

			return

		}

		// Apply rate limiting to POST requests.

		prl.rateLimiter.Middleware(next).ServeHTTP(w, r)
	})
}

// Stop gracefully shuts down the rate limiter.

func (prl *PostOnlyRateLimiter) Stop() {
	prl.rateLimiter.Stop()
}

// GetStats returns statistics about the rate limiter.

func (prl *PostOnlyRateLimiter) GetStats() map[string]interface{} {
	stats := prl.rateLimiter.GetStats()

	stats["applies_to"] = "POST requests only"

	return stats
}
