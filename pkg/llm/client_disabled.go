//go:build disable_rag
// +build disable_rag

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
	"strings"
	"sync"
	"time"
)

// ClientMetrics tracks client performance metrics (simplified for disable_rag)
type ClientMetrics struct {
	RequestsTotal    int64         `json:"requests_total"`
	RequestsSuccess  int64         `json:"requests_success"`
	RequestsFailure  int64         `json:"requests_failure"`
	TotalLatency     time.Duration `json:"total_latency"`
	CacheHits        int64         `json:"cache_hits"`
	CacheMisses      int64         `json:"cache_misses"`
	RetryAttempts    int64         `json:"retry_attempts"`
	FallbackAttempts int64         `json:"fallback_attempts"`
	mutex            sync.RWMutex
}

// Client is the simplified LLM client when RAG is disabled
type Client struct {
	// HTTP client with optimization
	httpClient *http.Client
	transport  *http.Transport

	// Core configuration
	url         string
	apiKey      string
	modelName   string
	maxTokens   int
	backendType string

	// Smart endpoints for backends
	processEndpoint string
	healthEndpoint  string

	// Performance components (simplified)
	cache          *ResponseCache
	circuitBreaker *CircuitBreaker
	retryConfig    RetryConfig

	// Observability (simplified)
	logger  *slog.Logger
	metrics *ClientMetrics

	// Concurrency control
	mutex        sync.RWMutex
	fallbackURLs []string

	// Token and cost tracking
	tokenTracker *SimpleTokenTracker
}

// ClientConfig holds unified configuration (simplified)
type ClientConfig struct {
	APIKey              string        `json:"api_key"`
	ModelName           string        `json:"model_name"`
	MaxTokens           int           `json:"max_tokens"`
	BackendType         string        `json:"backend_type"`
	Timeout             time.Duration `json:"timeout"`
	SkipTLSVerification bool          `json:"skip_tls_verification"`

	// Performance settings
	MaxConnsPerHost  int           `json:"max_conns_per_host"`
	MaxIdleConns     int           `json:"max_idle_conns"`
	IdleConnTimeout  time.Duration `json:"idle_conn_timeout"`
	KeepAliveTimeout time.Duration `json:"keep_alive_timeout"`

	// Cache settings
	CacheTTL     time.Duration `json:"cache_ttl"`
	CacheMaxSize int           `json:"cache_max_size"`

	// Circuit breaker settings
	CircuitBreakerConfig *CircuitBreakerConfig `json:"circuit_breaker_config"`
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries    int           `json:"max_retries"`
	BaseDelay     time.Duration `json:"base_delay"`
	MaxDelay      time.Duration `json:"max_delay"`
	JitterEnabled bool          `json:"jitter_enabled"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// NewClient creates a new simplified client when RAG is disabled
func NewClient(url, apiKey, modelName string, maxTokens int, config *ClientConfig) *Client {
	if config == nil {
		config = getDefaultClientConfig()
	}

	transport := &http.Transport{
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: time.Second,
		ForceAttemptHTTP2:     true,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: config.KeepAliveTimeout,
		}).DialContext,
	}

	if config.SkipTLSVerification {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	client := &Client{
		httpClient:      httpClient,
		transport:       transport,
		url:             url,
		apiKey:          apiKey,
		modelName:       modelName,
		maxTokens:       maxTokens,
		backendType:     config.BackendType,
		processEndpoint: url,
		healthEndpoint:  strings.TrimSuffix(url, "/") + "/health",
		logger:          slog.Default().With("component", "llm-client"),
		metrics:         &ClientMetrics{},
		cache:           NewResponseCache(config.CacheTTL, config.CacheMaxSize),
		circuitBreaker:  NewCircuitBreaker("llm-client", config.CircuitBreakerConfig),
		retryConfig: RetryConfig{
			MaxRetries:    3,
			BaseDelay:     time.Second,
			MaxDelay:      30 * time.Second,
			JitterEnabled: true,
			BackoffFactor: 2.0,
		},
		tokenTracker: NewSimpleTokenTracker(),
	}

	return client
}

// ProcessIntent processes an intent (simplified implementation)
func (c *Client) ProcessIntent(ctx context.Context, intent string) (string, error) {
	startTime := time.Now()

	// Update metrics
	c.updateMetrics(func(m *ClientMetrics) {
		m.RequestsTotal++
	})

	// Check cache first
	if cached, found := c.cache.Get(intent); found {
		c.updateMetrics(func(m *ClientMetrics) {
			m.CacheHits++
			m.RequestsSuccess++
		})
		return cached, nil
	}

	// Create request
	reqBody := map[string]interface{}{
		"intent":     intent,
		"model":      c.modelName,
		"max_tokens": c.maxTokens,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		c.updateMetrics(func(m *ClientMetrics) {
			m.RequestsFailure++
		})
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Execute request with retry logic
	var response string
	err = c.executeWithRetry(ctx, func() error {
		var execErr error
		response, execErr = c.executeRequest(ctx, jsonData)
		return execErr
	})

	// Update metrics
	latency := time.Since(startTime)
	c.updateMetrics(func(m *ClientMetrics) {
		m.TotalLatency += latency
		if err != nil {
			m.RequestsFailure++
		} else {
			m.RequestsSuccess++
			m.CacheMisses++
		}
	})

	if err != nil {
		return "", err
	}

	// Cache the response
	c.cache.Set(intent, response)

	return response, nil
}

// executeRequest executes the HTTP request
func (c *Client) executeRequest(ctx context.Context, jsonData []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.processEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract content from response
	if content, ok := result["content"].(string); ok {
		return content, nil
	}

	if response, ok := result["response"].(string); ok {
		return response, nil
	}

	return string(responseBody), nil
}

// executeWithRetry executes a function with retry logic
func (c *Client) executeWithRetry(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := c.calculateBackoffDelay(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			c.updateMetrics(func(m *ClientMetrics) {
				m.RetryAttempts++
			})
		}

		if err := fn(); err != nil {
			lastErr = err
			if !c.isRetryableError(err) {
				break
			}
			continue
		}

		return nil
	}

	return lastErr
}

// calculateBackoffDelay calculates the delay for exponential backoff
func (c *Client) calculateBackoffDelay(attempt int) time.Duration {
	delay := time.Duration(float64(c.retryConfig.BaseDelay) * (c.retryConfig.BackoffFactor * float64(attempt)))
	if delay > c.retryConfig.MaxDelay {
		delay = c.retryConfig.MaxDelay
	}

	if c.retryConfig.JitterEnabled {
		// Add 20% jitter
		jitter := time.Duration(float64(delay) * 0.2)
		jitterFactor := (2.0*float64(time.Now().UnixNano()%100)/100.0 - 1.0)
		delay += time.Duration(float64(jitter) * jitterFactor)
	}

	return delay
}

// isRetryableError determines if an error is retryable
func (c *Client) isRetryableError(err error) bool {
	// Simple retry logic for common network errors
	return strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "temporary failure")
}

// updateMetrics safely updates metrics
func (c *Client) updateMetrics(updater func(*ClientMetrics)) {
	c.metrics.mutex.Lock()
	defer c.metrics.mutex.Unlock()
	updater(c.metrics)
}

// GetMetrics returns current metrics
func (c *Client) GetMetrics() ClientMetrics {
	c.metrics.mutex.RLock()
	defer c.metrics.mutex.RUnlock()
	return *c.metrics
}

// Close closes the client and cleans up resources
func (c *Client) Close() error {
	if c.cache != nil {
		c.cache.Stop()
	}
	if c.transport != nil {
		c.transport.CloseIdleConnections()
	}
	return nil
}

// Health checks the health of the backend
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.healthEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// getDefaultClientConfig returns default client configuration
func getDefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Timeout:          30 * time.Second,
		MaxConnsPerHost:  10,
		MaxIdleConns:     100,
		IdleConnTimeout:  90 * time.Second,
		KeepAliveTimeout: 30 * time.Second,
		CacheTTL:         5 * time.Minute,
		CacheMaxSize:     1000,
		BackendType:      "openai",
		CircuitBreakerConfig: &CircuitBreakerConfig{
			FailureThreshold: 5,
			Timeout:          60 * time.Second,
		},
	}
}
