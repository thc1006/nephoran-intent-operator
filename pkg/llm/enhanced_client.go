package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// EnhancedClient provides additional functionality over the basic Client
type EnhancedClient struct {
	*Client
	circuitBreaker *CircuitBreaker
	rateLimiter    *RateLimiter
	healthChecker  *HealthChecker
}

// CircuitBreaker implements the circuit breaker pattern for LLM calls
type CircuitBreaker struct {
	failures    int64
	lastFailure time.Time
	state       CircuitState
	threshold   int64
	timeout     time.Duration
	mutex       sync.RWMutex
}

type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"
	CircuitOpen     CircuitState = "open"
	CircuitHalfOpen CircuitState = "half-open"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	tokens     int64
	maxTokens  int64
	refillRate int64
	lastRefill time.Time
	mutex      sync.Mutex
}

// HealthChecker monitors the health of LLM backends
type HealthChecker struct {
	client        *Client
	checkInterval time.Duration
	timeout       time.Duration
	healthStatus  map[string]BackendHealth
	mutex         sync.RWMutex
	stopChan      chan bool
}

type BackendHealth struct {
	Status       string        `json:"status"`
	LastCheck    time.Time     `json:"last_check"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorCount   int64         `json:"error_count"`
	Available    bool          `json:"available"`
}

// Enhanced error types for better error handling
type EnhancedError struct {
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Code      string                 `json:"code"`
	Context   map[string]interface{} `json:"context"`
	Timestamp time.Time              `json:"timestamp"`
	Retryable bool                   `json:"retryable"`
	Cause     error                  `json:"-"`
}

func (e *EnhancedError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *EnhancedError) Unwrap() error {
	return e.Cause
}

// Error types
const (
	ErrorTypeRateLimit      = "RATE_LIMIT_EXCEEDED"
	ErrorTypeCircuitBreaker = "CIRCUIT_BREAKER_OPEN"
	ErrorTypeTimeout        = "REQUEST_TIMEOUT"
	ErrorTypeValidation     = "VALIDATION_ERROR"
	ErrorTypeNetwork        = "NETWORK_ERROR"
	ErrorTypeLLM            = "LLM_PROCESSING_ERROR"
	ErrorTypeInternal       = "INTERNAL_ERROR"
)

// Error constructors
func NewRateLimitError(context map[string]interface{}) *EnhancedError {
	return &EnhancedError{
		Type:      ErrorTypeRateLimit,
		Message:   "Rate limit exceeded",
		Code:      "E1001",
		Context:   context,
		Timestamp: time.Now(),
		Retryable: true,
	}
}

func NewCircuitBreakerError(context map[string]interface{}) *EnhancedError {
	return &EnhancedError{
		Type:      ErrorTypeCircuitBreaker,
		Message:   "Circuit breaker is open",
		Code:      "E1002",
		Context:   context,
		Timestamp: time.Now(),
		Retryable: true,
	}
}

func NewTimeoutError(cause error, context map[string]interface{}) *EnhancedError {
	return &EnhancedError{
		Type:      ErrorTypeTimeout,
		Message:   "Request timeout",
		Code:      "E1003",
		Context:   context,
		Timestamp: time.Now(),
		Retryable: true,
		Cause:     cause,
	}
}

func NewValidationError(message string, context map[string]interface{}) *EnhancedError {
	return &EnhancedError{
		Type:      ErrorTypeValidation,
		Message:   message,
		Code:      "E1004",
		Context:   context,
		Timestamp: time.Now(),
		Retryable: false,
	}
}

func NewNetworkError(cause error, context map[string]interface{}) *EnhancedError {
	return &EnhancedError{
		Type:      ErrorTypeNetwork,
		Message:   "Network error",
		Code:      "E1005",
		Context:   context,
		Timestamp: time.Now(),
		Retryable: true,
		Cause:     cause,
	}
}

func NewLLMError(cause error, context map[string]interface{}) *EnhancedError {
	return &EnhancedError{
		Type:      ErrorTypeLLM,
		Message:   "LLM processing error",
		Code:      "E1006",
		Context:   context,
		Timestamp: time.Now(),
		Retryable: true,
		Cause:     cause,
	}
}

func NewInternalError(cause error, context map[string]interface{}) *EnhancedError {
	return &EnhancedError{
		Type:      ErrorTypeInternal,
		Message:   "Internal error",
		Code:      "E1007",
		Context:   context,
		Timestamp: time.Now(),
		Retryable: false,
		Cause:     cause,
	}
}

// EnhancedClientConfig extends ClientConfig with additional options
type EnhancedClientConfig struct {
	ClientConfig
	CircuitBreakerThreshold int64
	CircuitBreakerTimeout   time.Duration
	RateLimitTokens         int64
	RateLimitRefillRate     int64
	HealthCheckInterval     time.Duration
	HealthCheckTimeout      time.Duration
}

// NewEnhancedClient creates a new enhanced LLM client
func NewEnhancedClient(url string, config EnhancedClientConfig) *EnhancedClient {
	baseClient := NewClientWithConfig(url, config.ClientConfig)

	enhanced := &EnhancedClient{
		Client: baseClient,
		circuitBreaker: NewCircuitBreaker(
			config.CircuitBreakerThreshold,
			config.CircuitBreakerTimeout,
		),
		rateLimiter: NewRateLimiter(
			config.RateLimitTokens,
			config.RateLimitRefillRate,
		),
		healthChecker: NewHealthChecker(
			baseClient,
			config.HealthCheckInterval,
			config.HealthCheckTimeout,
		),
	}

	// Start health checking
	enhanced.healthChecker.Start()

	return enhanced
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int64, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold: threshold,
		timeout:   timeout,
		state:     CircuitClosed,
	}
}

// Call executes an operation through the circuit breaker
func (cb *CircuitBreaker) Call(operation func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Check if circuit should transition from open to half-open
	if cb.state == CircuitOpen && time.Since(cb.lastFailure) > cb.timeout {
		cb.state = CircuitHalfOpen
		cb.failures = 0
	}

	// If circuit is open, reject immediately
	if cb.state == CircuitOpen {
		return fmt.Errorf("circuit breaker is open")
	}

	// Execute operation
	err := operation()
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()

		// Open circuit if threshold exceeded
		if cb.failures >= cb.threshold {
			cb.state = CircuitOpen
		}
		return err
	}

	// Success - reset failures and close circuit
	cb.failures = 0
	cb.state = CircuitClosed
	return nil
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// NewRateLimiter creates a new token bucket rate limiter
func NewRateLimiter(maxTokens, refillRate int64) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed under the rate limit
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)

	// Refill tokens based on elapsed time
	tokensToAdd := int64(elapsed.Seconds()) * rl.refillRate
	rl.tokens = min(rl.maxTokens, rl.tokens+tokensToAdd)
	rl.lastRefill = now

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

// GetTokens returns the current number of available tokens
func (rl *RateLimiter) GetTokens() int64 {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	return rl.tokens
}

// min returns the minimum of two int64 values
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(client *Client, checkInterval, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		client:        client,
		checkInterval: checkInterval,
		timeout:       timeout,
		healthStatus:  make(map[string]BackendHealth),
		stopChan:      make(chan bool),
	}
}

// Start begins health checking
func (hc *HealthChecker) Start() {
	go hc.run()
}

// Stop stops health checking
func (hc *HealthChecker) Stop() {
	hc.stopChan <- true
}

// run executes the health checking loop
func (hc *HealthChecker) run() {
	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.checkHealth()
		case <-hc.stopChan:
			return
		}
	}
}

// checkHealth performs a health check on the LLM backend
func (hc *HealthChecker) checkHealth() {
	start := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	// Simple health check using a minimal intent
	_, err := hc.client.ProcessIntent(ctx, "health check")

	responseTime := time.Since(start)

	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	// Fix race condition: create new health status instead of modifying existing
	url := hc.client.url
	currentHealth, exists := hc.healthStatus[url]
	
	// Create new health status with atomic updates
	newHealth := BackendHealth{
		LastCheck:    time.Now(),
		ResponseTime: responseTime,
		Available:    err == nil,
		ErrorCount:   currentHealth.ErrorCount, // Preserve error count
	}
	
	if err != nil {
		newHealth.ErrorCount++
		newHealth.Status = "unhealthy"
	} else {
		newHealth.Status = "healthy"
		// Reset error count on successful health check
		if exists && currentHealth.ErrorCount > 0 {
			newHealth.ErrorCount = 0
		}
	}

	hc.healthStatus[url] = newHealth
}

// GetHealth returns the current health status
func (hc *HealthChecker) GetHealth() map[string]BackendHealth {
	hc.mutex.RLock()
	defer hc.mutex.RUnlock()

	result := make(map[string]BackendHealth)
	for k, v := range hc.healthStatus {
		result[k] = v
	}
	return result
}

// ProcessIntentWithEnhancements processes an intent with circuit breaker and rate limiting
func (ec *EnhancedClient) ProcessIntentWithEnhancements(ctx context.Context, intent string) (string, error) {
	// Create context for error reporting
	errorContext := map[string]interface{}{
		"intent":     intent,
		"backend":    ec.Client.backendType,
		"model":      ec.Client.modelName,
		"timestamp":  time.Now(),
	}

	// Check rate limiting
	if !ec.rateLimiter.Allow() {
		return "", NewRateLimitError(errorContext)
	}

	// Use circuit breaker
	var result string
	err := ec.circuitBreaker.Call(func() error {
		var processErr error
		result, processErr = ec.Client.ProcessIntent(ctx, intent)
		return processErr
	})

	if err != nil {
		// Check if it's a circuit breaker error
		if err.Error() == "circuit breaker is open" {
			return "", NewCircuitBreakerError(errorContext)
		}
		
		// Check for timeout errors
		if errors.Is(err, context.DeadlineExceeded) {
			return "", NewTimeoutError(err, errorContext)
		}
		
		// Check for network errors
		if strings.Contains(err.Error(), "connection") || strings.Contains(err.Error(), "network") {
			return "", NewNetworkError(err, errorContext)
		}
		
		// Default to LLM error
		return "", NewLLMError(err, errorContext)
	}

	return result, nil
}

// AsyncProcessingResult represents the result of an async processing operation
type AsyncProcessingResult struct {
	Result string
	Error  error
	Intent string
	StartTime time.Time
	EndTime   time.Time
}

// ProcessIntentWithEnhancementsAsync processes an intent asynchronously
func (ec *EnhancedClient) ProcessIntentWithEnhancementsAsync(ctx context.Context, intent string) <-chan AsyncProcessingResult {
	resultChan := make(chan AsyncProcessingResult, 1)
	
	go func() {
		defer close(resultChan)
		
		startTime := time.Now()
		result, err := ec.ProcessIntentWithEnhancements(ctx, intent)
		endTime := time.Now()
		
		resultChan <- AsyncProcessingResult{
			Result:    result,
			Error:     err,
			Intent:    intent,
			StartTime: startTime,
			EndTime:   endTime,
		}
	}()
	
	return resultChan
}

// GetEnhancedMetrics returns comprehensive metrics including circuit breaker and rate limiter status
func (ec *EnhancedClient) GetEnhancedMetrics() map[string]interface{} {
	baseMetrics := ec.Client.GetMetrics()

	return map[string]interface{}{
		"base_metrics": baseMetrics,
		"circuit_breaker": map[string]interface{}{
			"state": string(ec.circuitBreaker.GetState()),
		},
		"rate_limiter": map[string]interface{}{
			"available_tokens": ec.rateLimiter.GetTokens(),
		},
		"health_status": ec.healthChecker.GetHealth(),
	}
}

// CacheManager provides advanced caching functionality
type CacheManager struct {
	cache              *ResponseCache
	compressionEnabled bool
	encryptionEnabled  bool
	encryptionKey      []byte
	maxSize            int
	currentSize        int
	adaptiveTTL        bool
	usageStats         map[string]*CacheUsageStats
	mutex              sync.RWMutex
}

type CacheUsageStats struct {
	AccessCount     int64
	LastAccess      time.Time
	CreatedAt       time.Time
	AverageInterval time.Duration
}

// NewCacheManager creates a new cache manager with advanced features
func NewCacheManager(ttl time.Duration, maxSize int, compressionEnabled, encryptionEnabled bool, encryptionKey []byte) *CacheManager {
	return &CacheManager{
		cache:              NewResponseCache(ttl, maxSize),
		compressionEnabled: compressionEnabled,
		encryptionEnabled:  encryptionEnabled,
		encryptionKey:      encryptionKey,
		maxSize:            maxSize,
		adaptiveTTL:        true,
		usageStats:         make(map[string]*CacheUsageStats),
	}
}

// calculateAdaptiveTTL calculates TTL based on usage patterns
func (cm *CacheManager) calculateAdaptiveTTL(key string) time.Duration {
	cm.mutex.RLock()
	stats, exists := cm.usageStats[key]
	cm.mutex.RUnlock()
	
	if !exists || !cm.adaptiveTTL {
		return cm.cache.ttl // Use default TTL
	}
	
	// More frequently accessed items get longer TTL
	baseMultiplier := 1.0
	if stats.AccessCount > 10 {
		baseMultiplier = 2.0
	} else if stats.AccessCount > 5 {
		baseMultiplier = 1.5
	}
	
	// Recent items get longer TTL
	timeSinceCreation := time.Since(stats.CreatedAt)
	if timeSinceCreation < time.Hour {
		baseMultiplier *= 1.2
	}
	
	adaptiveTTL := time.Duration(float64(cm.cache.ttl) * baseMultiplier)
	
	// Cap the TTL to reasonable limits
	maxTTL := cm.cache.ttl * 4
	if adaptiveTTL > maxTTL {
		adaptiveTTL = maxTTL
	}
	
	return adaptiveTTL
}

// updateUsageStats updates cache usage statistics
func (cm *CacheManager) updateUsageStats(key string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	now := time.Now()
	stats, exists := cm.usageStats[key]
	
	if !exists {
		cm.usageStats[key] = &CacheUsageStats{
			AccessCount: 1,
			LastAccess:  now,
			CreatedAt:   now,
		}
		return
	}
	
	// Update access statistics
	if stats.AccessCount > 0 {
		interval := now.Sub(stats.LastAccess)
		stats.AverageInterval = time.Duration(
			(int64(stats.AverageInterval)*stats.AccessCount + int64(interval)) / (stats.AccessCount + 1),
		)
	}
	
	stats.AccessCount++
	stats.LastAccess = now
}

// evictLeastUsed removes the least recently used items when cache is full
func (cm *CacheManager) evictLeastUsed() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentSize <= cm.maxSize {
		return
	}
	
	// Find least recently used item
	var oldestKey string
	var oldestTime time.Time = time.Now()
	
	for key, stats := range cm.usageStats {
		if stats.LastAccess.Before(oldestTime) {
			oldestTime = stats.LastAccess
			oldestKey = key
		}
	}
	
	if oldestKey != "" {
		// Remove from both cache and usage stats
		cm.cache.mutex.Lock()
		delete(cm.cache.entries, oldestKey)
		cm.cache.mutex.Unlock()
		
		delete(cm.usageStats, oldestKey)
		cm.currentSize--
	}
}

// GetWithMetadata retrieves cached response with metadata
func (cm *CacheManager) GetWithMetadata(key string) (string, map[string]interface{}, bool) {
	response, found := cm.cache.Get(key)
	if !found {
		return "", nil, false
	}

	// Update usage statistics
	cm.updateUsageStats(key)

	// Get usage stats for metadata
	cm.mutex.RLock()
	stats := cm.usageStats[key]
	cm.mutex.RUnlock()

	metadata := map[string]interface{}{
		"cached":       true,
		"timestamp":    time.Now(),
		"access_count": stats.AccessCount,
		"created_at":   stats.CreatedAt,
		"last_access":  stats.LastAccess,
	}

	return response, metadata, true
}

// SetWithTags stores response with tags for categorization
func (cm *CacheManager) SetWithTags(key, response string, tags []string) {
	// Check if cache is full and evict if necessary
	if cm.currentSize >= cm.maxSize {
		cm.evictLeastUsed()
	}

	// Set in cache with adaptive TTL
	cm.cache.Set(key, response)
	
	// Initialize usage stats
	cm.mutex.Lock()
	cm.usageStats[key] = &CacheUsageStats{
		AccessCount: 0,
		LastAccess:  time.Now(),
		CreatedAt:   time.Now(),
	}
	cm.currentSize++
	cm.mutex.Unlock()
}

// InvalidateByPattern removes cache entries matching a pattern
func (cm *CacheManager) InvalidateByPattern(pattern string) int {
	// This is a simplified implementation
	// In production, you'd want to use a more efficient pattern matching
	count := 0
	cm.cache.mutex.Lock()
	defer cm.cache.mutex.Unlock()

	for key := range cm.cache.entries {
		if strings.Contains(key, pattern) {
			delete(cm.cache.entries, key)
			count++
		}
	}

	return count
}

// ResponseEnhancer provides response enhancement and validation
type ResponseEnhancer struct {
	logger          *slog.Logger
	validator       *ResponseValidator
	enrichmentRules map[string]func(map[string]interface{}) map[string]interface{}
}

// NewResponseEnhancer creates a new response enhancer
func NewResponseEnhancer() *ResponseEnhancer {
	return &ResponseEnhancer{
		logger:          slog.Default().With("component", "response-enhancer"),
		validator:       NewResponseValidator(),
		enrichmentRules: make(map[string]func(map[string]interface{}) map[string]interface{}),
	}
}

// EnhanceResponse enhances and validates an LLM response
func (re *ResponseEnhancer) EnhanceResponse(responseJSON string, intentType string) (string, error) {
	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Apply enrichment rules
	if enrichmentFunc, exists := re.enrichmentRules[intentType]; exists {
		response = enrichmentFunc(response)
	}

	// Add default enrichments
	response = re.addDefaultEnrichments(response)

	// Validate enhanced response
	enhancedJSON, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal enhanced response: %w", err)
	}

	if err := re.validator.ValidateResponse(enhancedJSON); err != nil {
		return "", fmt.Errorf("enhanced response validation failed: %w", err)
	}

	return string(enhancedJSON), nil
}

// addDefaultEnrichments adds standard enrichments to all responses
func (re *ResponseEnhancer) addDefaultEnrichments(response map[string]interface{}) map[string]interface{} {
	// Add processing metadata if not present
	if _, exists := response["processing_metadata"]; !exists {
		response["processing_metadata"] = map[string]interface{}{
			"enhanced":    true,
			"enhanced_at": time.Now().Format(time.RFC3339),
		}
	}

	// Add response ID for tracking
	if _, exists := response["response_id"]; !exists {
		hash := sha256.Sum256([]byte(fmt.Sprintf("%v", response)))
		response["response_id"] = hex.EncodeToString(hash[:])[:16]
	}

	return response
}

// AddEnrichmentRule adds a custom enrichment rule for specific intent types
func (re *ResponseEnhancer) AddEnrichmentRule(intentType string, rule func(map[string]interface{}) map[string]interface{}) {
	re.enrichmentRules[intentType] = rule
}

// RequestContextManager manages request context and tracing
type RequestContextManager struct {
	activeRequests map[string]*RequestContext
	mutex          sync.RWMutex
}

type RequestContext struct {
	ID        string                 `json:"id"`
	Intent    string                 `json:"intent"`
	StartTime time.Time              `json:"start_time"`
	UserID    string                 `json:"user_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewRequestContextManager creates a new request context manager
func NewRequestContextManager() *RequestContextManager {
	return &RequestContextManager{
		activeRequests: make(map[string]*RequestContext),
	}
}

// CreateContext creates a new request context
func (rcm *RequestContextManager) CreateContext(intent, userID, sessionID string) *RequestContext {
	ctx := &RequestContext{
		ID:        generateRequestID(),
		Intent:    intent,
		StartTime: time.Now(),
		UserID:    userID,
		SessionID: sessionID,
		Metadata:  make(map[string]interface{}),
	}

	rcm.mutex.Lock()
	defer rcm.mutex.Unlock()
	rcm.activeRequests[ctx.ID] = ctx

	return ctx
}

// GetContext retrieves a request context
func (rcm *RequestContextManager) GetContext(id string) (*RequestContext, bool) {
	rcm.mutex.RLock()
	defer rcm.mutex.RUnlock()
	ctx, exists := rcm.activeRequests[id]
	return ctx, exists
}

// CompleteContext marks a request as completed and removes it from active requests
func (rcm *RequestContextManager) CompleteContext(id string) {
	rcm.mutex.Lock()
	defer rcm.mutex.Unlock()
	delete(rcm.activeRequests, id)
}

// GetActiveRequests returns all currently active requests
func (rcm *RequestContextManager) GetActiveRequests() []RequestContext {
	rcm.mutex.RLock()
	defer rcm.mutex.RUnlock()

	requests := make([]RequestContext, 0, len(rcm.activeRequests))
	for _, ctx := range rcm.activeRequests {
		requests = append(requests, *ctx)
	}

	return requests
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(hash[:])[:16]
}
