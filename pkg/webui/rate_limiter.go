/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webui

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
)

// RateLimiter provides distributed rate limiting functionality
type RateLimiter struct {
	limits         map[string]*rateLimitBucket
	requestsPerMin int
	burstSize      int
	window         time.Duration
	mutex          sync.RWMutex
	stats          *RateLimitStats
	cleanup        chan struct{}
}

// rateLimitBucket represents a token bucket for rate limiting
type rateLimitBucket struct {
	tokens     int
	lastRefill time.Time
	requests   []time.Time
	mutex      sync.Mutex
}

// RateLimitStats provides rate limiting statistics
type RateLimitStats struct {
	TotalRequests     int64
	AllowedRequests   int64
	DeniedRequests    int64
	ActiveBuckets     int64
	ExpiredBuckets    int64
	AvgRequestsPerMin float64
	mutex             sync.RWMutex
}

// RateLimitConfig holds rate limiter configuration
type RateLimitConfig struct {
	RequestsPerMin   int
	BurstSize        int
	Window           time.Duration
	CleanupInterval  time.Duration
	MaxBuckets       int
	WhitelistIPs     []string
	WhitelistUserIDs []string
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(requestsPerMin, burstSize int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		limits:         make(map[string]*rateLimitBucket),
		requestsPerMin: requestsPerMin,
		burstSize:      burstSize,
		window:         window,
		stats:          &RateLimitStats{},
		cleanup:        make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupExpiredBuckets()

	return rl
}

// Allow checks if a request should be allowed based on rate limiting
func (rl *RateLimiter) Allow(identifier string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket, exists := rl.limits[identifier]
	if !exists {
		bucket = &rateLimitBucket{
			tokens:     rl.burstSize,
			lastRefill: time.Now(),
			requests:   make([]time.Time, 0),
		}
		rl.limits[identifier] = bucket
	}

	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	now := time.Now()

	// Refill tokens based on time passed
	rl.refillTokens(bucket, now)

	// Clean old requests from sliding window
	rl.cleanOldRequests(bucket, now)

	// Check if request is allowed
	allowed := false

	// Token bucket check (for burst)
	if bucket.tokens > 0 {
		bucket.tokens--
		allowed = true
	} else {
		// Sliding window check
		if len(bucket.requests) < rl.requestsPerMin {
			allowed = true
		}
	}

	// Record request
	rl.updateStats(func(s *RateLimitStats) {
		s.TotalRequests++
		if allowed {
			s.AllowedRequests++
			bucket.requests = append(bucket.requests, now)
		} else {
			s.DeniedRequests++
		}
		s.ActiveBuckets = int64(len(rl.limits))
	})

	return allowed
}

// AllowN checks if N requests should be allowed
func (rl *RateLimiter) AllowN(identifier string, n int) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket, exists := rl.limits[identifier]
	if !exists {
		bucket = &rateLimitBucket{
			tokens:     rl.burstSize,
			lastRefill: time.Now(),
			requests:   make([]time.Time, 0),
		}
		rl.limits[identifier] = bucket
	}

	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	now := time.Now()
	rl.refillTokens(bucket, now)
	rl.cleanOldRequests(bucket, now)

	// Check if we can allow N requests
	allowed := false
	if bucket.tokens >= n {
		bucket.tokens -= n
		allowed = true
	} else if len(bucket.requests)+n <= rl.requestsPerMin {
		allowed = true
	}

	// Record requests
	if allowed {
		for i := 0; i < n; i++ {
			bucket.requests = append(bucket.requests, now)
		}
	}

	rl.updateStats(func(s *RateLimitStats) {
		s.TotalRequests += int64(n)
		if allowed {
			s.AllowedRequests += int64(n)
		} else {
			s.DeniedRequests += int64(n)
		}
	})

	return allowed
}

// GetLimit returns the current rate limit information for an identifier
func (rl *RateLimiter) GetLimit(identifier string) *RateLimitInfo {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	bucket, exists := rl.limits[identifier]
	if !exists {
		return &RateLimitInfo{
			Limit:     rl.requestsPerMin,
			Remaining: rl.requestsPerMin,
			ResetTime: time.Now().Add(rl.window),
			Burst:     rl.burstSize,
		}
	}

	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	now := time.Now()
	rl.refillTokens(bucket, now)
	rl.cleanOldRequests(bucket, now)

	remaining := rl.requestsPerMin - len(bucket.requests)
	if remaining < 0 {
		remaining = 0
	}

	resetTime := now.Add(rl.window)
	if len(bucket.requests) > 0 {
		oldestRequest := bucket.requests[0]
		resetTime = oldestRequest.Add(rl.window)
	}

	return &RateLimitInfo{
		Limit:     rl.requestsPerMin,
		Remaining: remaining,
		ResetTime: resetTime,
		Burst:     bucket.tokens,
	}
}

// Stats returns rate limiting statistics
func (rl *RateLimiter) Stats() RateLimitStats {
	rl.stats.mutex.RLock()
	defer rl.stats.mutex.RUnlock()

	stats := *rl.stats // Create a copy
	return stats
}

// Reset clears all rate limit buckets
func (rl *RateLimiter) Reset() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.limits = make(map[string]*rateLimitBucket)

	rl.updateStats(func(s *RateLimitStats) {
		s.ActiveBuckets = 0
	})
}

// Stop stops the rate limiter and cleanup goroutines
func (rl *RateLimiter) Stop() {
	close(rl.cleanup)
}

// RateLimitInfo contains rate limit information for a specific identifier
type RateLimitInfo struct {
	Limit     int       `json:"limit"`      // Requests per window
	Remaining int       `json:"remaining"`  // Remaining requests in current window
	ResetTime time.Time `json:"reset_time"` // When the window resets
	Burst     int       `json:"burst"`      // Available burst tokens
}

// Internal methods

func (rl *RateLimiter) refillTokens(bucket *rateLimitBucket, now time.Time) {
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed.Minutes() * float64(rl.requestsPerMin))

	if tokensToAdd > 0 {
		bucket.tokens += tokensToAdd
		if bucket.tokens > rl.burstSize {
			bucket.tokens = rl.burstSize
		}
		bucket.lastRefill = now
	}
}

func (rl *RateLimiter) cleanOldRequests(bucket *rateLimitBucket, now time.Time) {
	cutoff := now.Add(-rl.window)

	// Remove requests older than the window
	validRequests := bucket.requests[:0]
	for _, requestTime := range bucket.requests {
		if requestTime.After(cutoff) {
			validRequests = append(validRequests, requestTime)
		}
	}
	bucket.requests = validRequests
}

func (rl *RateLimiter) cleanupExpiredBuckets() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rl.cleanup:
			return
		case <-ticker.C:
			rl.performCleanup()
		}
	}
}

func (rl *RateLimiter) performCleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	expiredCount := 0

	for identifier, bucket := range rl.limits {
		bucket.mutex.Lock()

		// Check if bucket is inactive (no recent requests)
		cutoff := now.Add(-2 * rl.window)
		hasRecentActivity := false

		for _, requestTime := range bucket.requests {
			if requestTime.After(cutoff) {
				hasRecentActivity = true
				break
			}
		}

		if !hasRecentActivity && bucket.lastRefill.Before(cutoff) {
			bucket.mutex.Unlock()
			delete(rl.limits, identifier)
			expiredCount++
		} else {
			bucket.mutex.Unlock()
		}
	}

	if expiredCount > 0 {
		rl.updateStats(func(s *RateLimitStats) {
			s.ExpiredBuckets += int64(expiredCount)
			s.ActiveBuckets = int64(len(rl.limits))
		})
	}
}

func (rl *RateLimiter) updateStats(updateFn func(*RateLimitStats)) {
	rl.stats.mutex.Lock()
	defer rl.stats.mutex.Unlock()
	updateFn(rl.stats)

	// Calculate average requests per minute
	if rl.stats.TotalRequests > 0 {
		// This is a simplified calculation - in production, you'd track this over time
		rl.stats.AvgRequestsPerMin = float64(rl.stats.AllowedRequests)
	}
}

// HTTP middleware integration

// RateLimitMiddleware creates HTTP middleware for rate limiting
func (s *NephoranAPIServer) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.rateLimiter == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Determine identifier (user ID or IP)
		identifier := s.getRateLimitIdentifier(r)

		// Check rate limit
		if !s.rateLimiter.Allow(identifier) {
			s.metrics.RateLimitExceeded.Inc()

			// Get limit info for headers
			limitInfo := s.rateLimiter.GetLimit(identifier)

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limitInfo.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(limitInfo.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(limitInfo.ResetTime.Unix(), 10))
			w.Header().Set("Retry-After", strconv.FormatInt(int64(limitInfo.ResetTime.Sub(time.Now()).Seconds()), 10))

			s.writeErrorResponse(w, http.StatusTooManyRequests, "rate_limit_exceeded",
				"Rate limit exceeded. Please try again later.")
			return
		}

		// Get and set rate limit headers
		limitInfo := s.rateLimiter.GetLimit(identifier)
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limitInfo.Limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(limitInfo.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(limitInfo.ResetTime.Unix(), 10))

		next.ServeHTTP(w, r)
	})
}

func (s *NephoranAPIServer) getRateLimitIdentifier(r *http.Request) string {
	// Try to get authenticated user ID first
	userID := auth.GetUserID(r.Context())
	if userID != "" {
		return fmt.Sprintf("user:%s", userID)
	}

	// Fall back to IP address
	ip := s.getClientIP(r)
	return fmt.Sprintf("ip:%s", ip)
}

func (s *NephoranAPIServer) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		clientIP := strings.TrimSpace(ips[0])
		if net.ParseIP(clientIP) != nil {
			return clientIP
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" && net.ParseIP(xri) != nil {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Advanced rate limiting features

// SetCustomLimit sets a custom rate limit for a specific identifier
func (rl *RateLimiter) SetCustomLimit(identifier string, requestsPerMin, burstSize int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket := &rateLimitBucket{
		tokens:     burstSize,
		lastRefill: time.Now(),
		requests:   make([]time.Time, 0),
	}
	rl.limits[identifier] = bucket
}

// RemoveLimit removes rate limiting for a specific identifier
func (rl *RateLimiter) RemoveLimit(identifier string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	delete(rl.limits, identifier)
}

// IsRateLimited checks if an identifier is currently rate limited without consuming a token
func (rl *RateLimiter) IsRateLimited(identifier string) bool {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	bucket, exists := rl.limits[identifier]
	if !exists {
		return false
	}

	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	now := time.Now()
	rl.refillTokens(bucket, now)
	rl.cleanOldRequests(bucket, now)

	return bucket.tokens == 0 && len(bucket.requests) >= rl.requestsPerMin
}
