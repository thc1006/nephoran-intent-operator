//go:build !disable_rag
// +build !disable_rag

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

	"github.com/nephio-project/nephoran-intent-operator/pkg/shared"
)

// ClientMetrics tracks client performance metrics.

type ClientMetrics struct {
	RequestsTotal int64 `json:"requests_total"`

	RequestsSuccess int64 `json:"requests_success"`

	RequestsFailure int64 `json:"requests_failure"`

	TotalLatency time.Duration `json:"total_latency"`

	CacheHits int64 `json:"cache_hits"`

	CacheMisses int64 `json:"cache_misses"`

	RetryAttempts int64 `json:"retry_attempts"`

	FallbackAttempts int64 `json:"fallback_attempts"`

	mutex sync.RWMutex
}

// Client is the unified LLM client with all performance optimizations built-in.

type Client struct {

	// HTTP client with optimization.

	httpClient *http.Client

	transport *http.Transport

	// Core configuration.

	url string

	apiKey string

	modelName string

	maxTokens int

	backendType string

	// Smart endpoints for RAG backend.

	processEndpoint string

	healthEndpoint string

	// Performance components.

	cache *ResponseCache

	circuitBreaker *CircuitBreaker

	retryConfig RetryConfig

	// Observability.

	logger *slog.Logger

	metrics *ClientMetrics

	metricsIntegrator *MetricsIntegrator

	// Concurrency control.

	mutex sync.RWMutex

	fallbackURLs []string

	// Token and cost tracking.

	tokenTracker *SimpleTokenTracker
}

// ClientConfig holds unified configuration.

type ClientConfig struct {
	APIKey string `json:"api_key"`

	ModelName string `json:"model_name"`

	MaxTokens int `json:"max_tokens"`

	BackendType string `json:"backend_type"`

	Timeout time.Duration `json:"timeout"`

	SkipTLSVerification bool `json:"skip_tls_verification"`

	// Performance settings.

	MaxConnsPerHost int `json:"max_conns_per_host"`

	MaxIdleConns int `json:"max_idle_conns"`

	IdleConnTimeout time.Duration `json:"idle_conn_timeout"`

	KeepAliveTimeout time.Duration `json:"keep_alive_timeout"`

	// Cache settings.

	CacheTTL time.Duration `json:"cache_ttl"`

	CacheMaxSize int `json:"cache_max_size"`

	// Circuit breaker settings.

	CircuitBreakerConfig *CircuitBreakerConfig `json:"circuit_breaker_config"`
}

// RetryConfig defines retry behavior.

type RetryConfig struct {
	MaxRetries int `json:"max_retries"`

	BaseDelay time.Duration `json:"base_delay"`

	MaxDelay time.Duration `json:"max_delay"`

	JitterEnabled bool `json:"jitter_enabled"`

	BackoffFactor float64 `json:"backoff_factor"`
}

// Use CircuitState and constants from circuit_breaker.go to avoid duplicates.

// Use CircuitBreaker from circuit_breaker.go to avoid duplicates.

// Use NewCircuitBreaker from circuit_breaker.go to avoid duplicates.

// Use Execute method from circuit_breaker.go to avoid duplicates.

// Use shouldTrip method from circuit_breaker.go to avoid duplicates.

// Use Reset method from circuit_breaker.go to avoid duplicates.

// Use ForceOpen method from circuit_breaker.go to avoid duplicates.

// NewClientMetrics creates new client metrics.

func NewClientMetrics() *ClientMetrics {

	return &ClientMetrics{}

}

// NewClient creates a new unified LLM client.

func NewClient(url string) *Client {

	return NewClientWithConfig(url, ClientConfig{

		ModelName: "gpt-4o-mini",

		MaxTokens: 2048,

		BackendType: "openai",

		Timeout: 60 * time.Second,

		// Performance defaults.

		MaxConnsPerHost: 100,

		MaxIdleConns: 50,

		IdleConnTimeout: 90 * time.Second,

		KeepAliveTimeout: 30 * time.Second,

		// Cache defaults.

		CacheTTL: 5 * time.Minute,

		CacheMaxSize: 1000,
	})

}

// NewClientWithConfig creates a client with specific configuration.

func NewClientWithConfig(url string, config ClientConfig) *Client {

	logger := slog.Default().With("component", "llm-client")

	// Create optimized TLS configuration.

	tlsConfig := &tls.Config{

		MinVersion: tls.VersionTLS12,

		CipherSuites: []uint16{

			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,

			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,

			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,

			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},

		PreferServerCipherSuites: true,
	}

	// Security check for TLS verification.

	if config.SkipTLSVerification {

		if !allowInsecureClient() {

			logger.Error("SECURITY VIOLATION: Attempted to skip TLS verification without proper authorization")

			panic("Security violation: TLS verification cannot be disabled")

		}

		logger.Warn("SECURITY WARNING: TLS verification disabled")

		tlsConfig.InsecureSkipVerify = true

	}

	// Create optimized HTTP transport.

	transport := &http.Transport{

		DialContext: (&net.Dialer{

			Timeout: 10 * time.Second,

			KeepAlive: config.KeepAliveTimeout,
		}).DialContext,

		MaxIdleConns: config.MaxIdleConns,

		MaxIdleConnsPerHost: config.MaxConnsPerHost,

		IdleConnTimeout: config.IdleConnTimeout,

		TLSHandshakeTimeout: 10 * time.Second,

		ExpectContinueTimeout: 1 * time.Second,

		TLSClientConfig: tlsConfig,

		// Performance optimizations.

		WriteBufferSize: 32 * 1024, // 32KB

		ReadBufferSize: 32 * 1024, // 32KB

		ForceAttemptHTTP2: true,
	}

	httpClient := &http.Client{

		Timeout: config.Timeout,

		Transport: transport,
	}

	// Create circuit breaker with config or defaults.

	cbConfig := config.CircuitBreakerConfig

	if cbConfig == nil {

		cbConfig = getDefaultCircuitBreakerConfig()

	}

	circuitBreaker := NewCircuitBreaker("llm-client", cbConfig)

	client := &Client{

		httpClient: httpClient,

		transport: transport,

		url: url,

		apiKey: config.APIKey,

		modelName: config.ModelName,

		maxTokens: config.MaxTokens,

		backendType: config.BackendType,

		logger: logger,

		metrics: NewClientMetrics(),

		cache: NewResponseCache(config.CacheTTL, config.CacheMaxSize),

		circuitBreaker: circuitBreaker,

		retryConfig: RetryConfig{

			MaxRetries: 3,

			BaseDelay: time.Second,

			MaxDelay: 30 * time.Second,

			JitterEnabled: true,

			BackoffFactor: 2.0,
		},

		tokenTracker: NewSimpleTokenTracker(),
	}

	// Initialize metrics integrator for Prometheus support.

	metricsCollector := NewMetricsCollector()

	client.metricsIntegrator = NewMetricsIntegrator(metricsCollector)

	// Initialize smart endpoints for RAG backend.

	if config.BackendType == "rag" {

		client.initializeRAGEndpoints()

	}

	return client

}

// initializeRAGEndpoints initializes smart endpoints for RAG backend.

func (c *Client) initializeRAGEndpoints() {

	baseURL := strings.TrimSuffix(c.url, "/")

	// Determine process endpoint based on URL pattern.

	if strings.HasSuffix(c.url, "/process_intent") {

		// Legacy pattern - use as configured.

		c.processEndpoint = c.url

	} else if strings.HasSuffix(c.url, "/process") {

		// New pattern - use as configured.

		c.processEndpoint = c.url

	} else {

		// Base URL pattern - default to /process for new installations.

		c.processEndpoint = baseURL + "/process"

	}

	// Health endpoint.

	processBase := baseURL

	if strings.HasSuffix(c.processEndpoint, "/process_intent") {

		processBase = strings.TrimSuffix(c.processEndpoint, "/process_intent")

	} else if strings.HasSuffix(c.processEndpoint, "/process") {

		processBase = strings.TrimSuffix(c.processEndpoint, "/process")

	}

	c.healthEndpoint = processBase + "/health"

	c.logger.Info("Initialized RAG client endpoints",

		slog.String("process_endpoint", c.processEndpoint),

		slog.String("health_endpoint", c.healthEndpoint),
	)

}

// ProcessIntent processes an intent with all optimizations.

func (c *Client) ProcessIntent(ctx context.Context, intent string) (string, error) {

	start := time.Now()

	var success bool

	var cacheHit bool

	var retryCount int

	var processingError error

	defer func() {

		c.updateMetrics(success, time.Since(start), cacheHit, retryCount)

		// Record specific error types for Prometheus metrics.

		if processingError != nil && c.metricsIntegrator != nil {

			errorType := c.categorizeError(processingError)

			c.metricsIntegrator.prometheusMetrics.RecordError(c.modelName, errorType)

		}

	}()

	c.logger.Debug("Processing intent", slog.String("intent", intent))

	// Check cache first.

	cacheKey := c.generateCacheKey(intent)

	if cached, found := c.cache.Get(cacheKey); found {

		cacheHit = true

		success = true

		return cached, nil

	}

	// Process with circuit breaker protection.

	result, err := c.circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {

		return c.processWithRetry(ctx, intent, &retryCount)

	})

	if err != nil {

		processingError = err

		// Check if this is a circuit breaker error.

		if strings.Contains(err.Error(), "circuit breaker is open") && c.metricsIntegrator != nil {

			c.metricsIntegrator.RecordCircuitBreakerEvent("llm-client", "rejected", c.modelName)

		}

		return "", err

	}

	response := result.(string)

	// Cache successful response.

	c.cache.Set(cacheKey, response)

	success = true

	return response, nil

}

// processWithRetry handles retry logic.

func (c *Client) processWithRetry(ctx context.Context, intent string, retryCount *int) (string, error) {

	var lastErr error

	delay := c.retryConfig.BaseDelay

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {

		if attempt > 0 {

			select {

			case <-ctx.Done():

				return "", ctx.Err()

			case <-time.After(delay):

				// Exponential backoff with jitter.

				delay = time.Duration(float64(delay) * c.retryConfig.BackoffFactor)

				if c.retryConfig.JitterEnabled {

					jitter := time.Duration(float64(delay) * 0.25 * (2.0*float64(time.Now().UnixNano()%1000)/1000.0 - 1.0))

					delay += jitter

				}

				if delay > c.retryConfig.MaxDelay {

					delay = c.retryConfig.MaxDelay

				}

			}

		}

		*retryCount = attempt

		result, err := c.processWithBackend(ctx, intent)

		if err == nil {

			return result, nil

		}

		lastErr = err

		if !c.isRetryableError(err) {

			return "", err

		}

	}

	return "", fmt.Errorf("operation failed after %d retries: %w", c.retryConfig.MaxRetries, lastErr)

}

// processWithBackend handles different LLM backends.

func (c *Client) processWithBackend(ctx context.Context, intent string) (string, error) {

	switch c.backendType {

	case "openai", "mistral":

		return c.processWithChatCompletion(ctx, intent)

	case "rag":

		return c.processWithRAGAPI(ctx, intent)

	default:

		return c.processWithChatCompletion(ctx, intent)

	}

}

// processWithChatCompletion handles OpenAI/Mistral-style APIs.

func (c *Client) processWithChatCompletion(ctx context.Context, intent string) (string, error) {

	requestBody := map[string]interface{}{

		"model": c.modelName,

		"messages": []map[string]string{

			{"role": "system", "content": "You are a telecommunications network expert."},

			{"role": "user", "content": intent},
		},

		"max_tokens": c.maxTokens,

		"temperature": 0.0,

		"response_format": map[string]string{"type": "json_object"},
	}

	reqBody, err := json.Marshal(requestBody)

	if err != nil {

		return "", fmt.Errorf("failed to marshal request: %w", err)

	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(reqBody))

	if err != nil {

		return "", fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("User-Agent", "nephoran-intent-operator/v1.0.0")

	if c.apiKey != "" {

		req.Header.Set("Authorization", "Bearer "+c.apiKey)

	}

	resp, err := c.httpClient.Do(req)

	if err != nil {

		return "", fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)

	if err != nil {

		return "", fmt.Errorf("failed to read response: %w", err)

	}

	if resp.StatusCode != http.StatusOK {

		return "", fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(respBody))

	}

	// Parse response.

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`

		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &chatResp); err != nil {

		return "", fmt.Errorf("failed to decode response: %w", err)

	}

	if len(chatResp.Choices) == 0 {

		return "", fmt.Errorf("no choices in response")

	}

	// Track token usage.

	c.tokenTracker.RecordUsage(chatResp.Usage.TotalTokens)

	return chatResp.Choices[0].Message.Content, nil

}

// processWithRAGAPI handles RAG API requests.

func (c *Client) processWithRAGAPI(ctx context.Context, intent string) (string, error) {

	req := map[string]interface{}{

		"spec": map[string]string{

			"intent": intent,
		},
	}

	reqBody, err := json.Marshal(req)

	if err != nil {

		return "", fmt.Errorf("failed to marshal request: %w", err)

	}

	// Use smart endpoint if available, otherwise fall back to URL construction.

	endpointURL := c.url + "/process" // Default fallback

	if c.processEndpoint != "" {

		endpointURL = c.processEndpoint

	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpointURL, bytes.NewBuffer(reqBody))

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

	respBody, err := io.ReadAll(resp.Body)

	if err != nil {

		return "", fmt.Errorf("failed to read response: %w", err)

	}

	if resp.StatusCode != http.StatusOK {

		return "", fmt.Errorf("RAG API returned status %d: %s", resp.StatusCode, string(respBody))

	}

	return string(respBody), nil

}

// Helper methods.

func (c *Client) generateCacheKey(intent string) string {

	return fmt.Sprintf("%s:%s:%s", c.backendType, c.modelName, intent)

}

func (c *Client) isRetryableError(err error) bool {

	errorStr := strings.ToLower(err.Error())

	retryablePatterns := []string{

		"connection refused", "timeout", "temporary failure",

		"service unavailable", "internal server error", "bad gateway",
	}

	for _, pattern := range retryablePatterns {

		if strings.Contains(errorStr, pattern) {

			return true

		}

	}

	return false

}

func allowInsecureClient() bool {

	return os.Getenv("ALLOW_INSECURE_CLIENT") == "true"

}

// GetMetrics returns current client metrics.

func (c *Client) GetMetrics() ClientMetrics {

	c.metrics.mutex.RLock()

	defer c.metrics.mutex.RUnlock()

	// Return a copy without the mutex to avoid copying lock values.

	return ClientMetrics{

		RequestsTotal: c.metrics.RequestsTotal,

		RequestsSuccess: c.metrics.RequestsSuccess,

		RequestsFailure: c.metrics.RequestsFailure,

		TotalLatency: c.metrics.TotalLatency,

		CacheHits: c.metrics.CacheHits,

		CacheMisses: c.metrics.CacheMisses,

		RetryAttempts: c.metrics.RetryAttempts,

		FallbackAttempts: c.metrics.FallbackAttempts,
	}

}

// SetFallbackURLs configures fallback URLs.

func (c *Client) SetFallbackURLs(urls []string) {

	c.mutex.Lock()

	defer c.mutex.Unlock()

	c.fallbackURLs = urls

}

// Shutdown gracefully shuts down the client.

func (c *Client) Shutdown() {

	c.mutex.Lock()

	defer c.mutex.Unlock()

	if c.cache != nil {

		c.cache.Stop()

	}

	if c.transport != nil {

		c.transport.CloseIdleConnections()

	}

}

// categorizeError categorizes errors for better Prometheus metrics.

func (c *Client) categorizeError(err error) string {

	if err == nil {

		return "none"

	}

	errMsg := err.Error()

	switch {

	case strings.Contains(errMsg, "circuit breaker is open"):

		return "circuit_breaker_open"

	case strings.Contains(errMsg, "timeout"):

		return "timeout"

	case strings.Contains(errMsg, "context deadline exceeded"):

		return "timeout"

	case strings.Contains(errMsg, "connection refused"):

		return "connection_refused"

	case strings.Contains(errMsg, "no such host"):

		return "dns_resolution"

	case strings.Contains(errMsg, "TLS"):

		return "tls_error"

	case strings.Contains(errMsg, "401"):

		return "authentication_error"

	case strings.Contains(errMsg, "403"):

		return "authorization_error"

	case strings.Contains(errMsg, "429"):

		return "rate_limit_exceeded"

	case strings.Contains(errMsg, "500"):

		return "server_error"

	case strings.Contains(errMsg, "502"), strings.Contains(errMsg, "503"), strings.Contains(errMsg, "504"):

		return "server_unavailable"

	case strings.Contains(errMsg, "parse") || strings.Contains(errMsg, "decode"):

		return "parsing_error"

	default:

		return "unknown_error"

	}

}

// updateMetrics updates client metrics.

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

	// Record Prometheus metrics via integrator.

	if c.metricsIntegrator != nil {

		status := "success"

		if !success {

			status = "error"

		}

		// Get current token stats for this request.

		// Note: This gives cumulative stats, not per-request, but it's the best we can do.

		// with the current TokenTracker implementation.

		tokenStats := c.tokenTracker.GetStats()

		totalTokensInt64 := tokenStats["total_tokens"].(int64)
		if totalTokensInt64 > 2147483647 || totalTokensInt64 < 0 {
			c.logger.Warn("Total tokens out of int range", "tokens", totalTokensInt64)
			totalTokensInt64 = 2147483647 // Cap at max int value
		}
		totalTokens := int(totalTokensInt64)

		// Record LLM request metrics.

		c.metricsIntegrator.RecordLLMRequest(c.modelName, status, latency, totalTokens)

		// Record cache operation.

		c.metricsIntegrator.RecordCacheOperation(c.modelName, "get", cacheHit)

		// Record retry attempts if any occurred.

		for range retryCount {

			c.metricsIntegrator.RecordRetryAttempt(c.modelName)

		}

	}

}

// Missing interface methods for shared.ClientInterface.

// ProcessIntentStream processes intent with streaming.

func (c *Client) ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *shared.StreamingChunk) error {

	// For now, simulate streaming by processing normally and chunking the response.

	response, err := c.ProcessIntent(ctx, prompt)

	if err != nil {

		return err

	}

	// Split response into chunks and send.

	words := strings.Fields(response)

	chunkSize := 10 // words per chunk

	for i := 0; i < len(words); i += chunkSize {

		end := i + chunkSize

		if end > len(words) {

			end = len(words)

		}

		chunk := &shared.StreamingChunk{

			Content: strings.Join(words[i:end], " "),

			IsLast: end >= len(words),

			Timestamp: time.Now(),
		}

		select {

		case chunks <- chunk:

		case <-ctx.Done():

			return ctx.Err()

		}

		// Add small delay to simulate streaming.

		time.Sleep(50 * time.Millisecond)

	}

	close(chunks)

	return nil

}

// GetSupportedModels returns supported models.

func (c *Client) GetSupportedModels() []string {

	return []string{c.modelName, "gpt-4o-mini", "gpt-3.5-turbo"}

}

// GetModelCapabilities returns model capabilities.

func (c *Client) GetModelCapabilities(modelName string) (*shared.ModelCapabilities, error) {

	switch modelName {

	case "gpt-4o-mini":

		return &shared.ModelCapabilities{

			MaxTokens: 128000,

			SupportsChat: true,

			SupportsFunction: true,

			SupportsStreaming: true,

			CostPerToken: 0.00015,
		}, nil

	case "gpt-3.5-turbo":

		return &shared.ModelCapabilities{

			MaxTokens: 4096,

			SupportsChat: true,

			SupportsFunction: false,

			SupportsStreaming: true,

			CostPerToken: 0.0015,
		}, nil

	default:

		return &shared.ModelCapabilities{

			MaxTokens: c.maxTokens,

			SupportsChat: true,

			SupportsFunction: false,

			SupportsStreaming: false,

			CostPerToken: 0.001,
		}, nil

	}

}

// ValidateModel validates if model is supported.

func (c *Client) ValidateModel(modelName string) error {

	supported := c.GetSupportedModels()

	for _, m := range supported {

		if m == modelName {

			return nil

		}

	}

	return fmt.Errorf("model %s not supported", modelName)

}

// EstimateTokens estimates token count for text.

func (c *Client) EstimateTokens(text string) int {

	// Simple estimation: ~4 characters per token.

	return len(text) / 4

}

// GetMaxTokens returns maximum tokens for model.

func (c *Client) GetMaxTokens(modelName string) int {

	caps, err := c.GetModelCapabilities(modelName)

	if err != nil {

		return c.maxTokens

	}

	return caps.MaxTokens

}

// Close closes the client and cleans up resources.

func (c *Client) Close() error {

	c.Shutdown()

	return nil

}
