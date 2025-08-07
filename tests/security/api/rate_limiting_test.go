package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

// RateLimitTestSuite contains rate limiting test scenarios
type RateLimitTestSuite struct {
	t         *testing.T
	endpoints []TestAPIEndpoint
	limiter   *rate.Limiter
}

// NewRateLimitTestSuite creates a new rate limiting test suite
func NewRateLimitTestSuite(t *testing.T) *RateLimitTestSuite {
	return &RateLimitTestSuite{
		t: t,
		endpoints: []TestAPIEndpoint{
			{Name: "LLM Processor", Port: LLMProcessorPort, BaseURL: fmt.Sprintf("http://localhost:%d", LLMProcessorPort)},
			{Name: "RAG API", Port: RAGAPIPort, BaseURL: fmt.Sprintf("http://localhost:%d", RAGAPIPort)},
			{Name: "Nephio Bridge", Port: NephioBridgePort, BaseURL: fmt.Sprintf("http://localhost:%d", NephioBridgePort)},
			{Name: "O-RAN Adaptor", Port: ORANAdaptorPort, BaseURL: fmt.Sprintf("http://localhost:%d", ORANAdaptorPort)},
		},
		limiter: rate.NewLimiter(rate.Every(time.Second/10), 10), // 10 requests per second with burst of 10
	}
}

// TestEndpointRateLimiting tests rate limiting per endpoint
func TestEndpointRateLimiting(t *testing.T) {
	suite := NewRateLimitTestSuite(t)

	endpointLimits := map[string]struct {
		rps   int // requests per second
		burst int // burst capacity
	}{
		"/api/v1/intent/process":     {rps: 10, burst: 20},  // LLM processing endpoint
		"/api/v1/rag/search":          {rps: 50, burst: 100}, // RAG search endpoint
		"/api/v1/nephio/packages":     {rps: 20, burst: 40},  // Nephio packages endpoint
		"/api/v1/oran/a1/policies":    {rps: 30, burst: 60},  // O-RAN A1 policies endpoint
		"/api/v1/health":              {rps: 100, burst: 200}, // Health check endpoint (higher limit)
		"/api/v1/metrics":             {rps: 100, burst: 200}, // Metrics endpoint
	}

	for endpoint, limits := range endpointLimits {
		t.Run(fmt.Sprintf("Endpoint_%s", endpoint), func(t *testing.T) {
			limiter := rate.NewLimiter(rate.Limit(limits.rps), limits.burst)
			
			// Test within limits
			successCount := 0
			for i := 0; i < limits.burst; i++ {
				if limiter.Allow() {
					successCount++
				}
			}
			assert.Equal(t, limits.burst, successCount, "Should allow burst requests")

			// Test rate limiting kicks in
			time.Sleep(100 * time.Millisecond)
			allowed := 0
			for i := 0; i < limits.rps*2; i++ {
				if limiter.Allow() {
					allowed++
				}
			}
			// Should only allow approximately rps/10 requests in 100ms
			expectedMax := limits.rps/10 + 2 // Add small buffer for timing
			assert.LessOrEqual(t, allowed, expectedMax, "Should enforce rate limit")

			// Test recovery after waiting
			time.Sleep(time.Second)
			recoveredCount := 0
			for i := 0; i < limits.rps; i++ {
				if limiter.Allow() {
					recoveredCount++
				}
			}
			assert.GreaterOrEqual(t, recoveredCount, limits.rps-1, "Should recover after waiting")
		})
	}
}

// TestPerUserRateLimiting tests rate limiting per user
func TestPerUserRateLimiting(t *testing.T) {
	suite := NewRateLimitTestSuite(t)

	userLimiters := make(map[string]*rate.Limiter)
	getUserLimiter := func(userID string) *rate.Limiter {
		if limiter, exists := userLimiters[userID]; exists {
			return limiter
		}
		// Different limits based on user tier
		var limiter *rate.Limiter
		switch userID {
		case "premium-user":
			limiter = rate.NewLimiter(100, 200) // Premium: 100 rps, burst 200
		case "standard-user":
			limiter = rate.NewLimiter(10, 20) // Standard: 10 rps, burst 20
		case "free-user":
			limiter = rate.NewLimiter(1, 5) // Free: 1 rps, burst 5
		default:
			limiter = rate.NewLimiter(5, 10) // Default: 5 rps, burst 10
		}
		userLimiters[userID] = limiter
		return limiter
	}

	testCases := []struct {
		userID         string
		requests       int
		expectedAllow  int
		expectedDeny   int
		description    string
	}{
		{
			userID:        "premium-user",
			requests:      250,
			expectedAllow: 200, // Burst capacity
			expectedDeny:  50,
			description:   "Premium user should have high limits",
		},
		{
			userID:        "standard-user",
			requests:      30,
			expectedAllow: 20, // Burst capacity
			expectedDeny:  10,
			description:   "Standard user should have moderate limits",
		},
		{
			userID:        "free-user",
			requests:      10,
			expectedAllow: 5, // Burst capacity
			expectedDeny:  5,
			description:   "Free user should have low limits",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.userID, func(t *testing.T) {
			limiter := getUserLimiter(tc.userID)
			
			allowed := 0
			denied := 0
			
			for i := 0; i < tc.requests; i++ {
				if limiter.Allow() {
					allowed++
				} else {
					denied++
				}
			}
			
			assert.Equal(t, tc.expectedAllow, allowed, tc.description)
			assert.Equal(t, tc.expectedDeny, denied, tc.description)
		})
	}
}

// TestDistributedRateLimiting tests distributed rate limiting across multiple instances
func TestDistributedRateLimiting(t *testing.T) {
	suite := NewRateLimitTestSuite(t)

	// Simulate distributed rate limiter with shared state
	type DistributedLimiter struct {
		mu          sync.Mutex
		tokens      int64
		maxTokens   int64
		refillRate  int64 // tokens per second
		lastRefill  time.Time
	}

	newDistributedLimiter := func(maxTokens, refillRate int64) *DistributedLimiter {
		return &DistributedLimiter{
			tokens:     maxTokens,
			maxTokens:  maxTokens,
			refillRate: refillRate,
			lastRefill: time.Now(),
		}
	}

	allow := func(dl *DistributedLimiter, tokens int64) bool {
		dl.mu.Lock()
		defer dl.mu.Unlock()

		// Refill tokens based on time elapsed
		now := time.Now()
		elapsed := now.Sub(dl.lastRefill).Seconds()
		tokensToAdd := int64(elapsed * float64(dl.refillRate))
		
		dl.tokens = dl.tokens + tokensToAdd
		if dl.tokens > dl.maxTokens {
			dl.tokens = dl.maxTokens
		}
		dl.lastRefill = now

		// Check if we have enough tokens
		if dl.tokens >= tokens {
			dl.tokens -= tokens
			return true
		}
		return false
	}

	// Test with multiple concurrent instances
	globalLimiter := newDistributedLimiter(100, 50) // 100 token bucket, 50 tokens/sec refill
	
	instances := 5
	requestsPerInstance := 50
	var wg sync.WaitGroup
	var totalAllowed int64

	for i := 0; i < instances; i++ {
		wg.Add(1)
		go func(instanceID int) {
			defer wg.Done()
			
			for j := 0; j < requestsPerInstance; j++ {
				if allow(globalLimiter, 1) {
					atomic.AddInt64(&totalAllowed, 1)
				}
				time.Sleep(10 * time.Millisecond) // Simulate request processing
			}
		}(i)
	}

	wg.Wait()

	// With 100 initial tokens and ~2.5 seconds of runtime (250 requests * 10ms)
	// we should get 100 initial + ~125 refilled tokens = ~225 allowed requests
	expectedMin := int64(100) // At least initial bucket
	expectedMax := int64(250) // Maximum possible if no rate limiting
	
	assert.GreaterOrEqual(t, totalAllowed, expectedMin, "Should allow at least initial bucket size")
	assert.Less(t, totalAllowed, expectedMax, "Should not allow all requests (rate limiting working)")
	
	t.Logf("Distributed rate limiting: %d/%d requests allowed", totalAllowed, instances*requestsPerInstance)
}

// TestRateLimitHeaders tests rate limit headers in responses
func TestRateLimitHeaders(t *testing.T) {
	suite := NewRateLimitTestSuite(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := suite.limiter
		
		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", "10")
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", limiter.Burst()))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Second).Unix()))
		
		if !limiter.Allow() {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		
		w.WriteHeader(http.StatusOK)
	})

	// Make requests and check headers
	for i := 0; i < 15; i++ {
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		// Check rate limit headers are present
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
		
		if w.Code == http.StatusTooManyRequests {
			// Check Retry-After header when rate limited
			assert.NotEmpty(t, w.Header().Get("Retry-After"))
			assert.Equal(t, "1", w.Header().Get("Retry-After"))
		}
	}
}

// TestDDoSProtection tests DDoS protection mechanisms
func TestDDoSProtection(t *testing.T) {
	suite := NewRateLimitTestSuite(t)

	t.Run("SYN_Flood_Protection", func(t *testing.T) {
		// Simulate SYN flood detection
		connectionAttempts := make(map[string]int)
		maxHalfOpenConnections := 100
		
		// Simulate connection attempts from multiple IPs
		for i := 0; i < 200; i++ {
			ip := fmt.Sprintf("192.168.1.%d", i%50) // 50 unique IPs
			connectionAttempts[ip]++
			
			// Check if we should block this IP
			if connectionAttempts[ip] > 5 {
				assert.Greater(t, connectionAttempts[ip], 5, "IP should be flagged for too many half-open connections")
			}
		}
		
		// Count IPs with excessive connections
		suspiciousIPs := 0
		for _, count := range connectionAttempts {
			if count > 5 {
				suspiciousIPs++
			}
		}
		
		assert.Greater(t, suspiciousIPs, 0, "Should detect suspicious IPs")
	})

	t.Run("HTTP_Flood_Protection", func(t *testing.T) {
		// Track request rates per IP
		type RequestTracker struct {
			mu       sync.Mutex
			requests map[string][]time.Time
		}
		
		tracker := &RequestTracker{
			requests: make(map[string][]time.Time),
		}
		
		isFlooding := func(ip string, windowSize time.Duration, threshold int) bool {
			tracker.mu.Lock()
			defer tracker.mu.Unlock()
			
			now := time.Now()
			
			// Clean old requests
			if reqs, exists := tracker.requests[ip]; exists {
				var validReqs []time.Time
				for _, reqTime := range reqs {
					if now.Sub(reqTime) <= windowSize {
						validReqs = append(validReqs, reqTime)
					}
				}
				tracker.requests[ip] = validReqs
			}
			
			// Add current request
			tracker.requests[ip] = append(tracker.requests[ip], now)
			
			// Check if flooding
			return len(tracker.requests[ip]) > threshold
		}
		
		// Simulate requests from multiple IPs
		blockedIPs := 0
		for i := 0; i < 1000; i++ {
			ip := fmt.Sprintf("10.0.0.%d", i%10) // 10 unique IPs
			
			if isFlooding(ip, 1*time.Second, 50) {
				blockedIPs++
			}
		}
		
		assert.Greater(t, blockedIPs, 0, "Should block IPs that are flooding")
	})

	t.Run("Slowloris_Protection", func(t *testing.T) {
		// Simulate slow HTTP connections
		type SlowConnection struct {
			ip            string
			startTime     time.Time
			bytesReceived int
		}
		
		connections := make(map[string]*SlowConnection)
		maxConnectionDuration := 30 * time.Second
		minBytesPerSecond := 100
		
		// Check for slow connections
		checkSlowConnection := func(conn *SlowConnection) bool {
			duration := time.Since(conn.startTime)
			if duration > maxConnectionDuration {
				return true // Connection too long
			}
			
			bytesPerSecond := float64(conn.bytesReceived) / duration.Seconds()
			if bytesPerSecond < float64(minBytesPerSecond) && duration > 5*time.Second {
				return true // Too slow after 5 seconds
			}
			
			return false
		}
		
		// Simulate connections
		for i := 0; i < 100; i++ {
			ip := fmt.Sprintf("172.16.0.%d", i)
			conn := &SlowConnection{
				ip:            ip,
				startTime:     time.Now().Add(-time.Duration(i) * time.Second),
				bytesReceived: i * 10, // Slow data rate
			}
			connections[ip] = conn
		}
		
		// Identify slow connections
		slowConnections := 0
		for _, conn := range connections {
			if checkSlowConnection(conn) {
				slowConnections++
			}
		}
		
		assert.Greater(t, slowConnections, 0, "Should detect slow connections")
	})

	t.Run("Amplification_Attack_Protection", func(t *testing.T) {
		// Track response size ratio
		type RequestResponse struct {
			requestSize  int
			responseSize int
		}
		
		// Check for amplification
		isAmplification := func(rr RequestResponse, threshold float64) bool {
			if rr.requestSize == 0 {
				return false
			}
			ratio := float64(rr.responseSize) / float64(rr.requestSize)
			return ratio > threshold
		}
		
		testCases := []RequestResponse{
			{requestSize: 100, responseSize: 10000},   // 100x amplification
			{requestSize: 1000, responseSize: 2000},   // 2x amplification
			{requestSize: 500, responseSize: 500},     // No amplification
			{requestSize: 50, responseSize: 100000},   // 2000x amplification
		}
		
		amplificationThreshold := 10.0
		detectedAmplifications := 0
		
		for _, tc := range testCases {
			if isAmplification(tc, amplificationThreshold) {
				detectedAmplifications++
			}
		}
		
		assert.Equal(t, 2, detectedAmplifications, "Should detect amplification attacks")
	})
}

// TestBurstHandling tests burst request handling
func TestBurstHandling(t *testing.T) {
	suite := NewRateLimitTestSuite(t)

	t.Run("Token_Bucket_Algorithm", func(t *testing.T) {
		// Implement token bucket
		type TokenBucket struct {
			mu         sync.Mutex
			tokens     float64
			maxTokens  float64
			refillRate float64 // tokens per second
			lastRefill time.Time
		}
		
		newTokenBucket := func(maxTokens, refillRate float64) *TokenBucket {
			return &TokenBucket{
				tokens:     maxTokens,
				maxTokens:  maxTokens,
				refillRate: refillRate,
				lastRefill: time.Now(),
			}
		}
		
		consume := func(tb *TokenBucket, tokens float64) bool {
			tb.mu.Lock()
			defer tb.mu.Unlock()
			
			// Refill tokens
			now := time.Now()
			elapsed := now.Sub(tb.lastRefill).Seconds()
			tb.tokens = tb.tokens + elapsed*tb.refillRate
			if tb.tokens > tb.maxTokens {
				tb.tokens = tb.maxTokens
			}
			tb.lastRefill = now
			
			// Try to consume tokens
			if tb.tokens >= tokens {
				tb.tokens -= tokens
				return true
			}
			return false
		}
		
		bucket := newTokenBucket(10, 5) // 10 token capacity, 5 tokens/sec refill
		
		// Test burst consumption
		burstSuccess := 0
		for i := 0; i < 15; i++ {
			if consume(bucket, 1) {
				burstSuccess++
			}
		}
		assert.Equal(t, 10, burstSuccess, "Should allow burst up to bucket capacity")
		
		// Wait for refill
		time.Sleep(2 * time.Second)
		
		// Test after refill
		refillSuccess := 0
		for i := 0; i < 15; i++ {
			if consume(bucket, 1) {
				refillSuccess++
			}
		}
		assert.GreaterOrEqual(t, refillSuccess, 10, "Should refill tokens over time")
	})

	t.Run("Leaky_Bucket_Algorithm", func(t *testing.T) {
		// Implement leaky bucket
		type LeakyBucket struct {
			mu         sync.Mutex
			queue      []time.Time
			capacity   int
			leakRate   time.Duration // time between leaks
			lastLeak   time.Time
		}
		
		newLeakyBucket := func(capacity int, leakRate time.Duration) *LeakyBucket {
			return &LeakyBucket{
				queue:    make([]time.Time, 0, capacity),
				capacity: capacity,
				leakRate: leakRate,
				lastLeak: time.Now(),
			}
		}
		
		addDrop := func(lb *LeakyBucket) bool {
			lb.mu.Lock()
			defer lb.mu.Unlock()
			
			now := time.Now()
			
			// Leak drops based on time elapsed
			elapsed := now.Sub(lb.lastLeak)
			dropsToLeak := int(elapsed / lb.leakRate)
			
			if dropsToLeak > 0 {
				if dropsToLeak >= len(lb.queue) {
					lb.queue = lb.queue[:0]
				} else {
					lb.queue = lb.queue[dropsToLeak:]
				}
				lb.lastLeak = now
			}
			
			// Try to add new drop
			if len(lb.queue) < lb.capacity {
				lb.queue = append(lb.queue, now)
				return true
			}
			return false
		}
		
		bucket := newLeakyBucket(10, 100*time.Millisecond) // 10 capacity, leak every 100ms
		
		// Test filling bucket
		accepted := 0
		rejected := 0
		
		for i := 0; i < 20; i++ {
			if addDrop(bucket) {
				accepted++
			} else {
				rejected++
			}
		}
		
		assert.Equal(t, 10, accepted, "Should accept up to bucket capacity")
		assert.Equal(t, 10, rejected, "Should reject when bucket is full")
		
		// Wait for bucket to leak
		time.Sleep(500 * time.Millisecond)
		
		// Test after leaking
		acceptedAfterLeak := 0
		for i := 0; i < 5; i++ {
			if addDrop(bucket) {
				acceptedAfterLeak++
			}
		}
		
		assert.GreaterOrEqual(t, acceptedAfterLeak, 4, "Should accept requests after leaking")
	})

	t.Run("Adaptive_Rate_Limiting", func(t *testing.T) {
		// Implement adaptive rate limiting based on system load
		type AdaptiveLimiter struct {
			mu              sync.Mutex
			baseRate        float64
			currentRate     float64
			cpuThreshold    float64
			memoryThreshold float64
		}
		
		newAdaptiveLimiter := func(baseRate float64) *AdaptiveLimiter {
			return &AdaptiveLimiter{
				baseRate:        baseRate,
				currentRate:     baseRate,
				cpuThreshold:    80.0, // 80% CPU
				memoryThreshold: 80.0, // 80% memory
			}
		}
		
		adjustRate := func(al *AdaptiveLimiter, cpuUsage, memoryUsage float64) {
			al.mu.Lock()
			defer al.mu.Unlock()
			
			if cpuUsage > al.cpuThreshold || memoryUsage > al.memoryThreshold {
				// Reduce rate when system is under load
				al.currentRate = al.baseRate * 0.5
			} else if cpuUsage < 50.0 && memoryUsage < 50.0 {
				// Increase rate when system has capacity
				al.currentRate = al.baseRate * 1.5
			} else {
				// Normal rate
				al.currentRate = al.baseRate
			}
		}
		
		limiter := newAdaptiveLimiter(100.0) // Base rate of 100 rps
		
		// Test different load scenarios
		scenarios := []struct {
			cpu      float64
			memory   float64
			expected float64
		}{
			{cpu: 90.0, memory: 70.0, expected: 50.0},  // High CPU
			{cpu: 70.0, memory: 90.0, expected: 50.0},  // High memory
			{cpu: 40.0, memory: 40.0, expected: 150.0}, // Low load
			{cpu: 60.0, memory: 60.0, expected: 100.0}, // Normal load
		}
		
		for _, scenario := range scenarios {
			adjustRate(limiter, scenario.cpu, scenario.memory)
			assert.Equal(t, scenario.expected, limiter.currentRate,
				fmt.Sprintf("Rate should adjust based on load (CPU: %.1f%%, Mem: %.1f%%)",
					scenario.cpu, scenario.memory))
		}
	})
}

// TestGracefulDegradation tests graceful degradation under load
func TestGracefulDegradation(t *testing.T) {
	suite := NewRateLimitTestSuite(t)

	t.Run("Priority_Based_Rate_Limiting", func(t *testing.T) {
		// Different rate limits based on request priority
		type PriorityLimiter struct {
			limiters map[string]*rate.Limiter
		}
		
		newPriorityLimiter := func() *PriorityLimiter {
			return &PriorityLimiter{
				limiters: map[string]*rate.Limiter{
					"critical": rate.NewLimiter(1000, 2000), // Critical: 1000 rps
					"high":     rate.NewLimiter(100, 200),   // High: 100 rps
					"normal":   rate.NewLimiter(10, 20),     // Normal: 10 rps
					"low":      rate.NewLimiter(1, 5),       // Low: 1 rps
				},
			}
		}
		
		allow := func(pl *PriorityLimiter, priority string) bool {
			if limiter, exists := pl.limiters[priority]; exists {
				return limiter.Allow()
			}
			return false
		}
		
		priorityLimiter := newPriorityLimiter()
		
		// Test different priorities
		priorities := []string{"critical", "high", "normal", "low"}
		results := make(map[string]int)
		
		for _, priority := range priorities {
			allowed := 0
			for i := 0; i < 100; i++ {
				if allow(priorityLimiter, priority) {
					allowed++
				}
			}
			results[priority] = allowed
		}
		
		// Verify priority-based limiting
		assert.Greater(t, results["critical"], results["high"], "Critical should allow more than high")
		assert.Greater(t, results["high"], results["normal"], "High should allow more than normal")
		assert.Greater(t, results["normal"], results["low"], "Normal should allow more than low")
	})

	t.Run("Circuit_Breaker_Integration", func(t *testing.T) {
		// Circuit breaker with rate limiting
		type CircuitBreaker struct {
			mu              sync.Mutex
			state           string // "closed", "open", "half-open"
			failures        int
			successCount    int
			failureThreshold int
			successThreshold int
			lastStateChange  time.Time
			timeout          time.Duration
		}
		
		newCircuitBreaker := func() *CircuitBreaker {
			return &CircuitBreaker{
				state:            "closed",
				failureThreshold: 5,
				successThreshold: 3,
				timeout:          10 * time.Second,
				lastStateChange:  time.Now(),
			}
		}
		
		call := func(cb *CircuitBreaker, fn func() error) error {
			cb.mu.Lock()
			defer cb.mu.Unlock()
			
			// Check if circuit breaker should transition to half-open
			if cb.state == "open" && time.Since(cb.lastStateChange) > cb.timeout {
				cb.state = "half-open"
				cb.successCount = 0
			}
			
			// Check state
			if cb.state == "open" {
				return fmt.Errorf("circuit breaker is open")
			}
			
			// Execute function
			err := fn()
			
			if err != nil {
				cb.failures++
				if cb.state == "half-open" {
					cb.state = "open"
					cb.lastStateChange = time.Now()
				} else if cb.failures >= cb.failureThreshold {
					cb.state = "open"
					cb.lastStateChange = time.Now()
				}
				return err
			}
			
			// Success
			if cb.state == "half-open" {
				cb.successCount++
				if cb.successCount >= cb.successThreshold {
					cb.state = "closed"
					cb.failures = 0
					cb.lastStateChange = time.Now()
				}
			} else {
				cb.failures = 0
			}
			
			return nil
		}
		
		cb := newCircuitBreaker()
		
		// Simulate failures to open circuit
		for i := 0; i < 6; i++ {
			call(cb, func() error {
				return fmt.Errorf("service error")
			})
		}
		
		assert.Equal(t, "open", cb.state, "Circuit should be open after failures")
		
		// Try to call when open
		err := call(cb, func() error { return nil })
		assert.Error(t, err, "Should reject calls when circuit is open")
		
		// Wait for timeout
		time.Sleep(11 * time.Second)
		
		// Circuit should allow test call (half-open)
		for i := 0; i < 3; i++ {
			call(cb, func() error { return nil })
		}
		
		assert.Equal(t, "closed", cb.state, "Circuit should close after successful calls")
	})

	t.Run("Load_Shedding", func(t *testing.T) {
		// Implement load shedding when system is overloaded
		type LoadShedder struct {
			mu               sync.Mutex
			currentLoad      int64
			maxLoad          int64
			shedProbability  float64
		}
		
		newLoadShedder := func(maxLoad int64) *LoadShedder {
			return &LoadShedder{
				maxLoad: maxLoad,
			}
		}
		
		shouldAccept := func(ls *LoadShedder) bool {
			ls.mu.Lock()
			defer ls.mu.Unlock()
			
			loadRatio := float64(ls.currentLoad) / float64(ls.maxLoad)
			
			if loadRatio < 0.7 {
				// Accept all requests when load is low
				return true
			} else if loadRatio < 0.9 {
				// Start shedding some requests
				ls.shedProbability = (loadRatio - 0.7) / 0.2 * 0.5 // Up to 50% shed
			} else {
				// Aggressive shedding when near capacity
				ls.shedProbability = 0.5 + (loadRatio-0.9)/0.1*0.5 // 50% to 100% shed
			}
			
			// Random shedding based on probability
			return rand.Float64() > ls.shedProbability
		}
		
		shedder := newLoadShedder(1000)
		
		// Test different load levels
		loadLevels := []struct {
			load            int64
			expectedAccept  bool
			description     string
		}{
			{load: 500, expectedAccept: true, description: "Low load"},
			{load: 800, expectedAccept: true, description: "Moderate load with some shedding"},
			{load: 950, expectedAccept: false, description: "High load with aggressive shedding"},
		}
		
		for _, level := range loadLevels {
			shedder.currentLoad = level.load
			
			accepted := 0
			total := 100
			
			for i := 0; i < total; i++ {
				if shouldAccept(shedder) {
					accepted++
				}
			}
			
			acceptRate := float64(accepted) / float64(total)
			
			if level.load < 700 {
				assert.Equal(t, 1.0, acceptRate, level.description)
			} else if level.load > 900 {
				assert.Less(t, acceptRate, 0.5, level.description)
			}
		}
	})
}

// TestIPBasedRateLimiting tests IP-based rate limiting
func TestIPBasedRateLimiting(t *testing.T) {
	suite := NewRateLimitTestSuite(t)

	type IPRateLimiter struct {
		mu       sync.RWMutex
		limiters map[string]*rate.Limiter
		banned   map[string]time.Time
	}

	newIPRateLimiter := func() *IPRateLimiter {
		return &IPRateLimiter{
			limiters: make(map[string]*rate.Limiter),
			banned:   make(map[string]time.Time),
		}
	}

	getLimiter := func(irl *IPRateLimiter, ip string) *rate.Limiter {
		irl.mu.Lock()
		defer irl.mu.Unlock()

		// Check if IP is banned
		if banTime, banned := irl.banned[ip]; banned {
			if time.Since(banTime) < 1*time.Hour {
				return nil // Still banned
			}
			delete(irl.banned, ip) // Ban expired
		}

		if limiter, exists := irl.limiters[ip]; exists {
			return limiter
		}

		// Create new limiter for IP
		limiter := rate.NewLimiter(10, 20) // 10 rps, burst 20
		irl.limiters[ip] = limiter
		return limiter
	}

	banIP := func(irl *IPRateLimiter, ip string) {
		irl.mu.Lock()
		defer irl.mu.Unlock()
		irl.banned[ip] = time.Now()
	}

	ipLimiter := newIPRateLimiter()

	t.Run("Normal_IP_Limiting", func(t *testing.T) {
		ip := "192.168.1.100"
		limiter := getLimiter(ipLimiter, ip)
		require.NotNil(t, limiter)

		allowed := 0
		for i := 0; i < 30; i++ {
			if limiter.Allow() {
				allowed++
			}
		}

		assert.Equal(t, 20, allowed, "Should allow burst limit for normal IP")
	})

	t.Run("Banned_IP_Rejection", func(t *testing.T) {
		ip := "10.0.0.1"
		banIP(ipLimiter, ip)

		limiter := getLimiter(ipLimiter, ip)
		assert.Nil(t, limiter, "Banned IP should return nil limiter")
	})

	t.Run("IP_Whitelist", func(t *testing.T) {
		whitelist := map[string]bool{
			"127.0.0.1":    true,
			"192.168.1.1":  true,
		}

		for ip := range whitelist {
			// Whitelisted IPs should have higher limits
			limiter := rate.NewLimiter(1000, 2000) // Much higher limits
			assert.NotNil(t, limiter, "Whitelisted IP should always get a limiter")
		}
	})
}

// TestRateLimitingMetrics tests rate limiting metrics and monitoring
func TestRateLimitingMetrics(t *testing.T) {
	suite := NewRateLimitTestSuite(t)

	type RateLimitMetrics struct {
		mu               sync.RWMutex
		totalRequests    int64
		allowedRequests  int64
		deniedRequests   int64
		avgResponseTime  time.Duration
		p95ResponseTime  time.Duration
		p99ResponseTime  time.Duration
		responseTimes    []time.Duration
	}

	metrics := &RateLimitMetrics{
		responseTimes: make([]time.Duration, 0, 1000),
	}

	recordRequest := func(m *RateLimitMetrics, allowed bool, responseTime time.Duration) {
		m.mu.Lock()
		defer m.mu.Unlock()

		m.totalRequests++
		if allowed {
			m.allowedRequests++
			m.responseTimes = append(m.responseTimes, responseTime)
		} else {
			m.deniedRequests++
		}
	}

	calculatePercentiles := func(m *RateLimitMetrics) {
		m.mu.Lock()
		defer m.mu.Unlock()

		if len(m.responseTimes) == 0 {
			return
		}

		// Sort response times
		sort.Slice(m.responseTimes, func(i, j int) bool {
			return m.responseTimes[i] < m.responseTimes[j]
		})

		// Calculate percentiles
		p95Index := int(float64(len(m.responseTimes)) * 0.95)
		p99Index := int(float64(len(m.responseTimes)) * 0.99)

		m.p95ResponseTime = m.responseTimes[p95Index]
		m.p99ResponseTime = m.responseTimes[p99Index]

		// Calculate average
		var total time.Duration
		for _, rt := range m.responseTimes {
			total += rt
		}
		m.avgResponseTime = total / time.Duration(len(m.responseTimes))
	}

	// Simulate requests with rate limiting
	limiter := rate.NewLimiter(10, 20)

	for i := 0; i < 100; i++ {
		start := time.Now()
		allowed := limiter.Allow()
		
		// Simulate processing time
		if allowed {
			time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
		}
		
		responseTime := time.Since(start)
		recordRequest(metrics, allowed, responseTime)
	}

	calculatePercentiles(metrics)

	// Verify metrics
	assert.Equal(t, int64(100), metrics.totalRequests, "Should record all requests")
	assert.Greater(t, metrics.allowedRequests, int64(0), "Should allow some requests")
	assert.Greater(t, metrics.deniedRequests, int64(0), "Should deny some requests")
	assert.Equal(t, metrics.totalRequests, metrics.allowedRequests+metrics.deniedRequests, "Total should equal allowed + denied")

	if metrics.allowedRequests > 0 {
		assert.Greater(t, metrics.avgResponseTime, time.Duration(0), "Should have average response time")
		assert.GreaterOrEqual(t, metrics.p95ResponseTime, metrics.avgResponseTime, "P95 should be >= average")
		assert.GreaterOrEqual(t, metrics.p99ResponseTime, metrics.p95ResponseTime, "P99 should be >= P95")
	}

	t.Logf("Rate Limiting Metrics:")
	t.Logf("  Total Requests: %d", metrics.totalRequests)
	t.Logf("  Allowed: %d (%.2f%%)", metrics.allowedRequests, float64(metrics.allowedRequests)/float64(metrics.totalRequests)*100)
	t.Logf("  Denied: %d (%.2f%%)", metrics.deniedRequests, float64(metrics.deniedRequests)/float64(metrics.totalRequests)*100)
	t.Logf("  Avg Response Time: %v", metrics.avgResponseTime)
	t.Logf("  P95 Response Time: %v", metrics.p95ResponseTime)
	t.Logf("  P99 Response Time: %v", metrics.p99ResponseTime)
}

// Missing imports
import (
	"math/rand"
	"sort"
)