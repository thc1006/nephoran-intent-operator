package llm

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared/types"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

// Client is a client for the LLM processor.
type Client struct {
	httpClient   *http.Client
	url          string
	promptEngine *TelecomPromptEngine
	retryConfig  RetryConfig
	validator    *ResponseValidator
	apiKey       string
	modelName    string
	maxTokens    int
	backendType  string
	logger       *slog.Logger
	metrics      *ClientMetrics
	mutex        sync.RWMutex
	cache        *ResponseCache
	fallbackURLs []string
	ragClient    rag.RAGClient // Optional RAG client for enhanced processing
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	JitterEnabled bool
	BackoffFactor float64
}

// ClientMetrics tracks client performance metrics
type ClientMetrics struct {
	RequestsTotal    int64
	RequestsSuccess  int64
	RequestsFailure  int64
	TotalLatency     time.Duration
	CacheHits        int64
	CacheMisses      int64
	RetryAttempts    int64
	FallbackAttempts int64
	mutex            sync.RWMutex
}

// ResponseCache provides simple in-memory caching
type ResponseCache struct {
	entries  map[string]*types.CacheEntry
	mutex    sync.RWMutex
	ttl      time.Duration
	maxSize  int
	stopCh   chan struct{}
	stopOnce sync.Once
	stopped  bool
}

// CacheEntry is now defined in pkg/shared/types/common_types.go

// ValidationError represents a validation error with missing fields
type ValidationError struct {
	Message       string   `json:"message"`
	MissingFields []string `json:"missing_fields"`
}

// Error implements the error interface
func (ve *ValidationError) Error() string {
	if len(ve.MissingFields) == 0 {
		return ve.Message
	}
	return fmt.Sprintf("%s: missing fields %v", ve.Message, ve.MissingFields)
}

// ResponseValidator validates LLM responses
type ResponseValidator struct {
	requiredFields map[string]bool
}

// NewClient creates a new LLM client with enhanced capabilities.
func NewClient(url string) *Client {
	return NewClientWithConfig(url, ClientConfig{
		APIKey:          "",
		ModelName:       "gpt-4o-mini",
		MaxTokens:       2048,
		BackendType:     "openai",
		Timeout:         60 * time.Second,
		MaxRetries:      2,   // Default retry count
		CacheMaxEntries: 512, // Default cache size
	})
}

// ClientConfig holds configuration for LLM client
type ClientConfig struct {
	APIKey              string
	ModelName           string
	MaxTokens           int
	BackendType         string
	Timeout             time.Duration
	SkipTLSVerification bool // SECURITY WARNING: Only use in development environments
	MaxRetries          int  // Maximum retry attempts for LLM requests
	CacheMaxEntries     int  // Maximum entries in LLM cache
}

// allowInsecureClient checks if insecure TLS is allowed via environment variable
// This follows the principle of requiring explicit opt-in for insecure behavior
func allowInsecureClient() bool {
	// Only allow insecure connections if explicitly enabled
	// This requires setting ALLOW_INSECURE_CLIENT=true
	envValue := os.Getenv("ALLOW_INSECURE_CLIENT")
	return envValue == "true"
}

// NewClientWithConfig creates a new LLM client with specific configuration
func NewClientWithConfig(url string, config ClientConfig) *Client {
	// Initialize logger early for security logging
	logger := slog.Default().With("component", "llm-client")

	// Create TLS configuration with security by default
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		// Additional security hardening
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
	}

	// Security check: Only allow skipping TLS verification when both conditions are met
	if config.SkipTLSVerification {
		if !allowInsecureClient() {
			// Log security violation attempt
			logger.Error("SECURITY VIOLATION: Attempted to skip TLS verification without proper authorization",
				slog.String("url", url),
				slog.Bool("skip_tls_requested", true),
				slog.String("env_allow_insecure", os.Getenv("ALLOW_INSECURE_CLIENT")),
			)
			// Fail securely - do not create client with insecure settings
			panic("Security violation: TLS verification cannot be disabled without explicit environment permission")
		}

		// Both conditions met - log security warning
		logger.Warn("SECURITY WARNING: TLS verification disabled - THIS IS INSECURE",
			slog.String("url", url),
			slog.String("reason", "both SkipTLSVerification=true and ALLOW_INSECURE_CLIENT=true"),
			slog.String("recommendation", "Only use in development/testing environments"),
		)

		// Apply insecure settings
		tlsConfig.InsecureSkipVerify = true
	}

	// Create HTTP client with enhanced configuration
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       tlsConfig,
	}

	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	// Use configuration values passed in (already parsed from environment in config package)
	maxRetries := config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2 // Fallback default
	}

	cacheMaxEntries := config.CacheMaxEntries
	if cacheMaxEntries <= 0 {
		cacheMaxEntries = 512 // Fallback default
	}

	client := &Client{
		httpClient:   httpClient,
		url:          url,
		promptEngine: NewTelecomPromptEngine(),
		retryConfig: RetryConfig{
			MaxRetries:    maxRetries, // Use config value passed in
			BaseDelay:     time.Second,
			MaxDelay:      30 * time.Second,
			JitterEnabled: true,
			BackoffFactor: 2.0,
		},
		validator:    NewResponseValidator(),
		apiKey:       config.APIKey,
		modelName:    config.ModelName,
		maxTokens:    config.MaxTokens,
		backendType:  config.BackendType,
		logger:       logger,
		metrics:      NewClientMetrics(),
		cache:        NewResponseCache(5*time.Minute, cacheMaxEntries), // Use environment variable
		fallbackURLs: []string{},                                       // Can be configured for redundancy
	}

	// Initialize RAG client if backend type is "rag"
	if config.BackendType == "rag" {
		ragConfig := &rag.RAGClientConfig{
			Enabled:          true,
			MaxSearchResults: 5,
			MinConfidence:    0.7,
			WeaviateURL:      os.Getenv("WEAVIATE_URL"),
			WeaviateAPIKey:   os.Getenv("WEAVIATE_API_KEY"),
			LLMEndpoint:      url,
			LLMAPIKey:        config.APIKey,
			MaxTokens:        config.MaxTokens,
			Temperature:      0.0,
		}
		client.ragClient = rag.NewRAGClient(ragConfig)

		// Initialize the RAG client
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.ragClient.Initialize(ctx); err != nil {
			logger.Warn("Failed to initialize RAG client, will use fallback processing",
				slog.String("error", err.Error()))
			// Don't fail hard, just disable RAG
			client.ragClient = nil
		}
	}

	return client
}

// NewClientMetrics creates a new metrics tracker
func NewClientMetrics() *ClientMetrics {
	return &ClientMetrics{}
}

// NewResponseCache creates a new response cache
func NewResponseCache(ttl time.Duration, maxSize int) *ResponseCache {
	cache := &ResponseCache{
		entries: make(map[string]*types.CacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
		stopCh:  make(chan struct{}),
		stopped: false,
	}

	// Start cleanup routine
	go cache.cleanup()

	return cache
}

// cleanup removes expired cache entries
func (c *ResponseCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			// Graceful shutdown signal received
			return
		case <-ticker.C:
			// Check if we've been stopped during the cleanup operation
			c.mutex.RLock()
			if c.stopped {
				c.mutex.RUnlock()
				return
			}
			c.mutex.RUnlock()

			// Perform cleanup
			c.mutex.Lock()
			now := time.Now()
			for key, entry := range c.entries {
				if now.Sub(entry.Timestamp) > c.ttl {
					delete(c.entries, key)
				}
			}
			c.mutex.Unlock()
		}
	}
}

// Get retrieves a cached response
func (c *ResponseCache) Get(key string) (string, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Return false if cache is stopped
	if c.stopped {
		return "", false
	}

	entry, exists := c.entries[key]
	if !exists {
		return "", false
	}

	if time.Since(entry.Timestamp) > c.ttl {
		return "", false
	}

	entry.HitCount++
	entry.LastAccess = time.Now() // Update access time for LRU
	return entry.Response, true
}

// Set stores a response in cache with LRU eviction
func (c *ResponseCache) Set(key, response string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Don't set if cache is stopped
	if c.stopped {
		return
	}

	// LRU eviction when capacity is exceeded
	for len(c.entries) >= c.maxSize {
		// Find the least recently used entry (oldest LastAccess)
		lruKey := ""
		lruTime := time.Now()

		for k, v := range c.entries {
			// Use LastAccess if available, otherwise use Timestamp
			accessTime := v.Timestamp
			if !v.LastAccess.IsZero() {
				accessTime = v.LastAccess
			}

			if accessTime.Before(lruTime) {
				lruTime = accessTime
				lruKey = k
			}
		}

		if lruKey != "" {
			delete(c.entries, lruKey)
		}
	}

	now := time.Now()
	c.entries[key] = &types.CacheEntry{
		Response:   response,
		Timestamp:  now,
		LastAccess: now, // Initialize LastAccess
		HitCount:   0,
	}
}

// Stop gracefully shuts down the cache cleanup goroutine
func (c *ResponseCache) Stop() {
	c.stopOnce.Do(func() {
		c.mutex.Lock()
		c.stopped = true
		c.mutex.Unlock()

		// Signal the cleanup goroutine to stop
		close(c.stopCh)
	})
}

// GetMetrics returns current client metrics
func (c *Client) GetMetrics() ClientMetrics {
	c.metrics.mutex.RLock()
	defer c.metrics.mutex.RUnlock()
	return *c.metrics
}

// updateMetrics updates client metrics
func (c *Client) updateMetrics(success bool, latency time.Duration, cacheHit bool, retryCount int) {
	c.metrics.mutex.Lock()
	defer c.metrics.mutex.Unlock()

	c.metrics.RequestsTotal++
	if success {
		c.metrics.RequestsSuccess++
	} else {
		c.metrics.RequestsFailure++
	}
	c.metrics.TotalLatency += latency

	if cacheHit {
		c.metrics.CacheHits++
	} else {
		c.metrics.CacheMisses++
	}

	c.metrics.RetryAttempts += int64(retryCount)
}

// NewResponseValidator creates a new response validator
func NewResponseValidator() *ResponseValidator {
	return &ResponseValidator{
		requiredFields: map[string]bool{
			"type":      true,
			"name":      true,
			"namespace": true,
			"spec":      true,
		},
	}
}

func (c *Client) ProcessIntent(ctx context.Context, intent string) (string, error) {
	start := time.Now()
	var success bool
	var cacheHit bool
	var retryCount int

	defer func() {
		c.updateMetrics(success, time.Since(start), cacheHit, retryCount)
	}()

	// Use timeout from client configuration (already parsed from environment)
	timeout := c.httpClient.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second // Fallback default
	}
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Truncate intent for logging if needed
	truncatedIntent := intent
	if c.logger.Enabled(ctx, slog.LevelDebug) && len(intent) > 1000 {
		truncatedIntent = intent[:1000] + "..."
	}

	c.logger.Info("Processing intent",
		slog.String("intent_type", c.classifyIntent(intent)),
		slog.String("backend", c.backendType),
	)
	c.logger.Debug("Processing intent with full details",
		slog.String("intent", truncatedIntent),
		slog.String("backend", c.backendType),
	)

	// Check cache first
	cacheKey := c.generateCacheKey(intent)
	if cached, found := c.cache.Get(cacheKey); found {
		c.logger.Info("Cache hit for intent")
		c.logger.Debug("Cache hit for intent", slog.String("cache_key", cacheKey))
		cacheHit = true
		success = true
		return cached, nil
	}
	cacheHit = false

	// Process with enhanced logic
	// Classify intent to determine processing approach
	intentType := c.classifyIntent(intent)

	// Pre-process intent with parameter extraction
	extractedParams := c.promptEngine.ExtractParameters(intent)

	// Process with retry logic using appropriate backend
	var result string
	err := c.retryWithExponentialBackoff(ctxWithTimeout, func() error {
		var processErr error
		result, processErr = c.processWithLLMBackend(ctxWithTimeout, intent, intentType, extractedParams)
		retryCount++
		return processErr
	})

	if err != nil {
		// Try fallback URLs if available
		if len(c.fallbackURLs) > 0 {
			c.logger.Warn("Primary URL failed, trying fallback URLs", slog.String("error", err.Error()))
			for _, fallbackURL := range c.fallbackURLs {
				c.metrics.mutex.Lock()
				c.metrics.FallbackAttempts++
				c.metrics.mutex.Unlock()

				originalURL := c.url
				c.url = fallbackURL

				fallbackErr := c.retryWithExponentialBackoff(ctxWithTimeout, func() error {
					var processErr error
					result, processErr = c.processWithLLMBackend(ctxWithTimeout, intent, intentType, extractedParams)
					return processErr
				})

				c.url = originalURL // Restore original URL

				if fallbackErr == nil {
					c.logger.Info("Fallback URL succeeded", slog.String("fallback_url", fallbackURL))
					break
				}

				c.logger.Warn("Fallback URL failed", slog.String("fallback_url", fallbackURL), slog.String("error", fallbackErr.Error()))
			}
		}

		if result == "" {
			return "", fmt.Errorf("failed to process intent after retries and fallbacks: %w", err)
		}
	}

	// Validate the response
	if err := c.validator.ValidateResponse([]byte(result)); err != nil {
		// Truncate response for logging if needed
		truncatedResponse := result
		if c.logger.Enabled(ctx, slog.LevelDebug) && len(result) > 1000 {
			truncatedResponse = result[:1000] + "..."
		}
		c.logger.Error("Response validation failed", slog.String("error", err.Error()))
		c.logger.Debug("Response validation failed with details",
			slog.String("error", err.Error()),
			slog.String("response", truncatedResponse),
		)
		return "", fmt.Errorf("response validation failed: %w", err)
	}

	// Cache successful response
	c.cache.Set(cacheKey, result)
	c.logger.Info("Response cached successfully")
	c.logger.Debug("Response cached", slog.String("cache_key", cacheKey))

	success = true
	c.logger.Info("Intent processed successfully",
		slog.String("intent_type", intentType),
		slog.Duration("processing_time", time.Since(start)),
		slog.Int("retry_count", retryCount),
	)

	return result, nil
}

// generateCacheKey creates a cache key for the given intent
func (c *Client) generateCacheKey(intent string) string {
	// Simple hash-based cache key (in production, consider using a proper hash function)
	return fmt.Sprintf("%s:%s:%s", c.backendType, c.modelName, intent)
}

// SetFallbackURLs configures fallback URLs for redundancy
func (c *Client) SetFallbackURLs(urls []string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.fallbackURLs = urls
}

// Shutdown gracefully shuts down the client and its resources
func (c *Client) Shutdown() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.cache != nil {
		c.cache.Stop()
	}

	// Shutdown RAG client if present
	if c.ragClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.ragClient.Shutdown(ctx); err != nil {
			c.logger.Warn("Failed to shutdown RAG client cleanly",
				slog.String("error", err.Error()))
		}
	}

	// Close HTTP client connections
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}

// classifyIntent determines the type of network intent
func (c *Client) classifyIntent(intent string) string {
	lowerIntent := strings.ToLower(intent)

	scaleIndicators := []string{"scale", "increase", "decrease", "replicas", "instances", "resize"}
	deployIndicators := []string{"deploy", "create", "setup", "configure", "install", "provision"}

	for _, indicator := range scaleIndicators {
		if strings.Contains(lowerIntent, indicator) {
			return "NetworkFunctionScale"
		}
	}

	for _, indicator := range deployIndicators {
		if strings.Contains(lowerIntent, indicator) {
			return "NetworkFunctionDeployment"
		}
	}

	return "NetworkFunctionDeployment" // Default
}

// processWithLLMBackend handles processing with different LLM backends
func (c *Client) processWithLLMBackend(ctx context.Context, intent, intentType string, extractedParams map[string]interface{}) (string, error) {
	// Generate appropriate prompt based on intent type
	systemPrompt := c.promptEngine.GeneratePrompt(intentType, intent)

	// Create request based on backend type
	switch c.backendType {
	case "openai", "mistral":
		return c.processWithChatCompletion(ctx, systemPrompt, intent)
	case "rag":
		return c.processWithRAGAPI(ctx, intent)
	default:
		// Default to OpenAI-compatible API
		return c.processWithChatCompletion(ctx, systemPrompt, intent)
	}
}

// processWithChatCompletion handles OpenAI/Mistral-style chat completions
func (c *Client) processWithChatCompletion(ctx context.Context, systemPrompt, intent string) (string, error) {
	requestBody := map[string]interface{}{
		"model": c.modelName,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": intent},
		},
		"max_tokens":      c.maxTokens,
		"temperature":     0.0,
		"response_format": map[string]string{"type": "json_object"},
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "nephoran-intent-operator/v1.0.0")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Validate Content-Type header
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		return "", fmt.Errorf("invalid response content type: expected application/json, got %s", contentType)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Validate that the response body is valid JSON
	var tempJSON interface{}
	if err := json.Unmarshal(respBody, &tempJSON); err != nil {
		return "", fmt.Errorf("response body is not valid JSON: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response based on backend type
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// processWithRAGAPI handles RAG API requests
func (c *Client) processWithRAGAPI(ctx context.Context, intent string) (string, error) {
	// Use the RAG client interface if available
	if c.ragClient != nil {
		// Use the new Retrieve method to get relevant documents
		docs, err := c.ragClient.Retrieve(ctx, intent)
		if err != nil {
			c.logger.Debug("RAG client retrieval failed, falling back to direct API",
				slog.String("error", err.Error()))
			// Fall through to direct API call
		} else {
			// Build context from retrieved documents
			var contextBuilder strings.Builder
			contextBuilder.WriteString("Based on the following context:\n\n")

			for i, doc := range docs {
				contextBuilder.WriteString(fmt.Sprintf("Context %d (confidence: %.2f):\n%s\n\n",
					i+1, doc.Confidence, doc.Content))
			}

			// Generate enhanced response using ChatCompletion with RAG context
			systemPrompt := c.promptEngine.GeneratePrompt("NetworkFunctionDeployment", intent)
			enhancedPrompt := fmt.Sprintf("%s\n\nAdditional Context:\n%s", systemPrompt, contextBuilder.String())

			return c.processWithChatCompletion(ctx, enhancedPrompt, intent)
		}
	}

	// Fallback to direct API call (legacy compatibility)
	req := map[string]interface{}{
		"spec": map[string]string{
			"intent": intent,
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url+"/process", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "nephoran-intent-operator/v1.0.0")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Validate Content-Type header for RAG API
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		return "", fmt.Errorf("invalid response content type: expected application/json, got %s", contentType)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Validate that the response body is valid JSON
	var tempJSON interface{}
	if err := json.Unmarshal(respBody, &tempJSON); err != nil {
		return "", fmt.Errorf("response body is not valid JSON: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errorResp map[string]interface{}
		if json.Unmarshal(respBody, &errorResp) == nil {
			if errorMsg, ok := errorResp["error"].(string); ok {
				return "", fmt.Errorf("RAG API error (%d): %s", resp.StatusCode, errorMsg)
			}
		}
		return "", fmt.Errorf("RAG API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}

// retryWithExponentialBackoff implements retry logic with exponential backoff and jitter
func (c *Client) retryWithExponentialBackoff(ctx context.Context, operation func() error) error {
	var lastErr error
	delay := c.retryConfig.BaseDelay

	// Use max retries from client configuration (already parsed from environment)
	maxRetries := c.retryConfig.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 2 // Fallback default
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Debug("Retrying operation",
				slog.Int("attempt", attempt),
				slog.Duration("delay", delay),
				slog.String("last_error", lastErr.Error()),
			)

			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Exponential backoff with optional jitter
				delay = time.Duration(float64(delay) * c.retryConfig.BackoffFactor)
				if c.retryConfig.JitterEnabled {
					// Add jitter (±25% of delay)
					jitter := time.Duration(float64(delay) * 0.25 * (2.0*float64(time.Now().UnixNano()%1000)/1000.0 - 1.0))
					delay += jitter
				}
				if delay > c.retryConfig.MaxDelay {
					delay = c.retryConfig.MaxDelay
				}
			}
		}

		lastErr = operation()
		if lastErr == nil {
			if attempt > 0 {
				c.logger.Info("Operation succeeded after retry", slog.Int("attempts", attempt+1))
			}
			return nil // Success
		}

		// Check if error is retryable
		if !c.isRetryableError(lastErr) {
			c.logger.Debug("Error is not retryable", slog.String("error", lastErr.Error()))
			return lastErr
		}
	}

	c.logger.Error("Operation failed after all retries",
		slog.Int("max_retries", maxRetries),
		slog.String("final_error", lastErr.Error()),
	)
	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

// isRetryableError determines if an error warrants a retry
func (c *Client) isRetryableError(err error) bool {
	errorStr := strings.ToLower(err.Error())

	// Network-related errors are typically retryable
	retryablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"service unavailable",
		"internal server error",
		"bad gateway",
		"circuit breaker",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

// ValidateResponse validates the structure of an LLM response
func (v *ResponseValidator) ValidateResponse(responseBody []byte) error {
	var response map[string]interface{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("invalid JSON response: %w", err)
	}

	// Check required fields and collect missing ones
	var missingFields []string
	for field := range v.requiredFields {
		if _, exists := response[field]; !exists {
			missingFields = append(missingFields, field)
		}
	}

	// Return structured error if any fields are missing
	if len(missingFields) > 0 {
		return &ValidationError{
			Message:       "Response validation failed",
			MissingFields: missingFields,
		}
	}

	// Validate type field
	if responseType, ok := response["type"].(string); ok {
		validTypes := []string{"NetworkFunctionDeployment", "NetworkFunctionScale"}
		valid := false
		for _, validType := range validTypes {
			if responseType == validType {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid response type: %s", responseType)
		}
	} else {
		return fmt.Errorf("type field must be a string")
	}

	// Validate name field format (Kubernetes naming)
	if name, ok := response["name"].(string); ok {
		if !isValidKubernetesName(name) {
			return fmt.Errorf("invalid Kubernetes name format: %s", name)
		}
	} else {
		return fmt.Errorf("name field must be a string")
	}

	// Validate namespace field format
	if namespace, ok := response["namespace"].(string); ok {
		if !isValidKubernetesName(namespace) {
			return fmt.Errorf("invalid Kubernetes namespace format: %s", namespace)
		}
	} else {
		return fmt.Errorf("namespace field must be a string")
	}

	// Validate spec field structure
	if spec, ok := response["spec"].(map[string]interface{}); ok {
		if responseType := response["type"].(string); responseType == "NetworkFunctionDeployment" {
			return v.validateDeploymentSpec(spec)
		} else if responseType == "NetworkFunctionScale" {
			return v.validateScaleSpec(spec)
		}
	} else {
		return fmt.Errorf("spec field must be an object")
	}

	return nil
}

// validateDeploymentSpec validates NetworkFunctionDeployment spec
func (v *ResponseValidator) validateDeploymentSpec(spec map[string]interface{}) error {
	// Check required deployment fields
	requiredFields := []string{"replicas", "image"}
	for _, field := range requiredFields {
		if _, exists := spec[field]; !exists {
			return fmt.Errorf("missing required deployment spec field: %s", field)
		}
	}

	// Validate replicas
	if replicas, ok := spec["replicas"].(float64); ok {
		if replicas < 1 || replicas > 100 {
			return fmt.Errorf("replicas must be between 1 and 100, got: %v", replicas)
		}
	} else {
		return fmt.Errorf("replicas must be a number")
	}

	// Validate image
	if image, ok := spec["image"].(string); ok {
		if image == "" {
			return fmt.Errorf("image cannot be empty")
		}
	} else {
		return fmt.Errorf("image must be a string")
	}

	return nil
}

// validateScaleSpec validates NetworkFunctionScale spec
func (v *ResponseValidator) validateScaleSpec(spec map[string]interface{}) error {
	// For scaling, we need at least one scaling parameter
	hasHorizontal := false
	hasVertical := false

	if scaling, ok := spec["scaling"].(map[string]interface{}); ok {
		if horizontal, exists := scaling["horizontal"]; exists {
			hasHorizontal = true
			if h, ok := horizontal.(map[string]interface{}); ok {
				if replicas, exists := h["replicas"]; exists {
					if r, ok := replicas.(float64); ok {
						if r < 1 || r > 100 {
							return fmt.Errorf("horizontal scaling replicas must be between 1 and 100")
						}
					} else {
						return fmt.Errorf("horizontal scaling replicas must be a number")
					}
				}
			}
		}

		if vertical, exists := scaling["vertical"]; exists {
			hasVertical = true
			if v, ok := vertical.(map[string]interface{}); ok {
				// Validate CPU format if present
				if cpu, exists := v["cpu"]; exists {
					if cpuStr, ok := cpu.(string); ok {
						if !isValidCPUFormat(cpuStr) {
							return fmt.Errorf("invalid CPU format: %s", cpuStr)
						}
					} else {
						return fmt.Errorf("vertical scaling CPU must be a string")
					}
				}

				// Validate memory format if present
				if memory, exists := v["memory"]; exists {
					if memStr, ok := memory.(string); ok {
						if !isValidMemoryFormat(memStr) {
							return fmt.Errorf("invalid memory format: %s", memStr)
						}
					} else {
						return fmt.Errorf("vertical scaling memory must be a string")
					}
				}
			}
		}
	}

	if !hasHorizontal && !hasVertical {
		return fmt.Errorf("scaling spec must include either horizontal or vertical scaling parameters")
	}

	return nil
}

// Helper validation functions
func isValidKubernetesName(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}

	// Kubernetes names must match DNS subdomain format
	for i, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.') {
			return false
		}
		if i == 0 && (r == '-' || r == '.') {
			return false
		}
		if i == len(name)-1 && (r == '-' || r == '.') {
			return false
		}
	}

	return true
}

func isValidCPUFormat(cpu string) bool {
	// Valid formats: "100m", "0.1", "1", "2000m"
	if cpu == "" {
		return false
	}

	if strings.HasSuffix(cpu, "m") {
		// Millicores format
		cpuValue := strings.TrimSuffix(cpu, "m")
		for _, r := range cpuValue {
			if r < '0' || r > '9' {
				return false
			}
		}
		return len(cpuValue) > 0
	} else {
		// Cores format (can include decimal)
		for _, r := range cpu {
			if !((r >= '0' && r <= '9') || r == '.') {
				return false
			}
		}
		return len(cpu) > 0
	}
}

func isValidMemoryFormat(memory string) bool {
	// Valid formats: "256Mi", "1Gi", "512Mi", "2Gi"
	if memory == "" {
		return false
	}

	validSuffixes := []string{"Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "K", "M", "G", "T", "P", "E"}

	for _, suffix := range validSuffixes {
		if strings.HasSuffix(memory, suffix) {
			memoryValue := strings.TrimSuffix(memory, suffix)
			for _, r := range memoryValue {
				if r < '0' || r > '9' {
					return false
				}
			}
			return len(memoryValue) > 0
		}
	}

	return false
}
