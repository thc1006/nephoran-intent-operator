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
	"net"
	"net/http"
	"sync"
	"time"
	"unsafe"

	"github.com/valyala/fastjson"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// OptimizedHTTPClient provides high-performance HTTP operations for LLM requests.

type OptimizedHTTPClient struct {
	client *http.Client

	pool *ConnectionPool

	transport *OptimizedTransport

	// Request optimization.

	requestPool sync.Pool

	responsePool sync.Pool

	bufferPool sync.Pool

	// JSON optimization.

	parser *fastjson.Parser

	parserPool sync.Pool

	// Metrics and monitoring.

	stats *HTTPClientStats

	tracer trace.Tracer

	// Configuration.

	config *OptimizedClientConfig

	mutex sync.RWMutex
}

// OptimizedTransport extends http.RoundTripper with performance optimizations.

type OptimizedTransport struct {
	*http.Transport

	// Connection management.

	connPool *ConnectionPool

	healthCheck *HealthChecker

	// Performance tracking.

	stats *TransportStats

	// Request routing and load balancing.

	endpointPool *EndpointPool
}

// ConnectionPool manages HTTP connection reuse and optimization.

type ConnectionPool struct {
	// Per-host connection pools.

	hostPools map[string]*HostConnectionPool

	mutex sync.RWMutex

	// Configuration.

	maxConnsPerHost int

	maxIdleConns int

	idleConnTimeout time.Duration

	keepAliveTimeout time.Duration

	// Metrics.

	totalConns int64

	activeConns int64

	idleConns int64

	connCreated int64

	connClosed int64

	connReused int64
}

// HostConnectionPool manages connections for a specific host.

type HostConnectionPool struct {
	host string

	connections []*PooledConnection

	available chan *PooledConnection

	maxSize int

	created int

	mutex sync.Mutex

	lastUsed time.Time
}

// PooledConnection represents a reusable HTTP connection.

type PooledConnection struct {
	conn net.Conn

	created time.Time

	lastUsed time.Time

	requestCount int64

	isHealthy bool

	metadata map[string]interface{}
}

// OptimizedClientConfig holds configuration for the optimized client.

type OptimizedClientConfig struct {
	// Connection pooling.

	MaxConnsPerHost int `json:"max_conns_per_host"`

	MaxIdleConns int `json:"max_idle_conns"`

	IdleConnTimeout time.Duration `json:"idle_conn_timeout"`

	KeepAliveTimeout time.Duration `json:"keep_alive_timeout"`

	// HTTP timeouts.

	ConnectTimeout time.Duration `json:"connect_timeout"`

	RequestTimeout time.Duration `json:"request_timeout"`

	ResponseTimeout time.Duration `json:"response_timeout"`

	TLSHandshakeTimeout time.Duration `json:"tls_handshake_timeout"`

	// Buffer management.

	ReadBufferSize int `json:"read_buffer_size"`

	WriteBufferSize int `json:"write_buffer_size"`

	// Performance tuning.

	DisableCompression bool `json:"disable_compression"`

	DisableKeepAlives bool `json:"disable_keep_alives"`

	MaxResponseSize int `json:"max_response_size"`

	// Health checking.

	HealthCheckEnabled bool `json:"health_check_enabled"`

	HealthCheckInterval time.Duration `json:"health_check_interval"`

	// Load balancing.

	LoadBalancingEnabled bool `json:"load_balancing_enabled"`

	BackendURLs []string `json:"backend_urls"`

	// TLS optimization.

	TLSOptimization TLSConfig `json:"tls_optimization"`

	// API Configuration
	APIKey string `json:"api_key,omitempty"`

	// Batch processing configuration
	BatchConfig *BatchProcessorConfig `json:"batch_config,omitempty"`
}

// TLSConfig holds TLS optimization settings.

type TLSConfig struct {
	SessionCacheSize int `json:"session_cache_size"`

	ReuseThreshold int `json:"reuse_threshold"`

	SessionTimeout time.Duration `json:"session_timeout"`

	PreferServerCiphers bool `json:"prefer_server_ciphers"`

	MinVersion uint16 `json:"min_version"`
}

// HTTPClientStats tracks HTTP client performance.

type HTTPClientStats struct {
	RequestsTotal int64 `json:"requests_total"`

	RequestsSuccessful int64 `json:"requests_successful"`

	RequestsFailed int64 `json:"requests_failed"`

	// Timing statistics.

	TotalLatency time.Duration `json:"total_latency"`

	AverageLatency time.Duration `json:"average_latency"`

	P95Latency time.Duration `json:"p95_latency"`

	P99Latency time.Duration `json:"p99_latency"`

	// Connection statistics.

	ConnectionsCreated int64 `json:"connections_created"`

	ConnectionsReused int64 `json:"connections_reused"`

	ConnectionsClosed int64 `json:"connections_closed"`

	// Buffer utilization.

	BufferPoolHits int64 `json:"buffer_pool_hits"`

	BufferPoolMisses int64 `json:"buffer_pool_misses"`

	mutex sync.RWMutex
}

// NewOptimizedHTTPClient creates a new optimized HTTP client.

func NewOptimizedHTTPClient(config *OptimizedClientConfig) (*OptimizedHTTPClient, error) {
	if config == nil {
		config = getDefaultOptimizedClientConfig()
	}

	// Create connection pool.

	pool := &ConnectionPool{
		hostPools: make(map[string]*HostConnectionPool),

		maxConnsPerHost: config.MaxConnsPerHost,

		maxIdleConns: config.MaxIdleConns,

		idleConnTimeout: config.IdleConnTimeout,

		keepAliveTimeout: config.KeepAliveTimeout,
	}

	// Create optimized transport.

	transport := &OptimizedTransport{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: config.ConnectTimeout,

				KeepAlive: config.KeepAliveTimeout,

				DualStack: true,
			}).DialContext,

			// Connection pool settings.

			MaxIdleConns: config.MaxIdleConns,

			MaxIdleConnsPerHost: config.MaxConnsPerHost,

			MaxConnsPerHost: config.MaxConnsPerHost,

			IdleConnTimeout: config.IdleConnTimeout,

			// Timeout settings.

			TLSHandshakeTimeout: config.TLSHandshakeTimeout,

			ResponseHeaderTimeout: config.ResponseTimeout,

			ExpectContinueTimeout: time.Second,

			// Buffer settings.

			ReadBufferSize: config.ReadBufferSize,

			WriteBufferSize: config.WriteBufferSize,

			// Performance settings.

			DisableCompression: config.DisableCompression,

			DisableKeepAlives: config.DisableKeepAlives,

			// TLS optimization.

			TLSClientConfig: &tls.Config{
				MinVersion: config.TLSOptimization.MinVersion,

				ClientSessionCache: tls.NewLRUClientSessionCache(

					config.TLSOptimization.SessionCacheSize,
				),

				PreferServerCipherSuites: config.TLSOptimization.PreferServerCiphers,
			},
		},

		connPool: pool,

		stats: &TransportStats{},
	}

	// Create HTTP client.

	httpClient := &http.Client{
		Transport: transport,

		Timeout: config.RequestTimeout,
	}

	client := &OptimizedHTTPClient{
		client: httpClient,

		pool: pool,

		transport: transport,

		config: config,

		stats: &HTTPClientStats{},

		// Initialize object pools.

		requestPool: sync.Pool{
			New: func() interface{} {
				return &OptimizedRequest{}
			},
		},

		responsePool: sync.Pool{
			New: func() interface{} {
				return &OptimizedResponse{}
			},
		},

		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 4096) // 4KB initial capacity
			},
		},

		parserPool: sync.Pool{
			New: func() interface{} {
				return &fastjson.Parser{}
			},
		},
	}

	return client, nil
}

// OptimizedRequest represents a pooled HTTP request.

type OptimizedRequest struct {
	Method string

	URL string

	Headers map[string]string

	Body []byte

	ContentType string

	Timeout time.Duration

	Metadata map[string]interface{}
}

// OptimizedResponse represents a pooled HTTP response.

type OptimizedResponse struct {
	StatusCode int

	Headers map[string]string

	Body []byte

	Size int

	Latency time.Duration

	FromCache bool

	Content string // LLM response content

	FromBatch bool // Whether from batch processing

	TokensUsed int // Number of tokens used

	ProcessingTime time.Duration // Processing time

	Response string // Alias for Content

	Metadata map[string]interface{}
}

// ProcessLLMRequest processes an LLM request with all optimizations.

func (c *OptimizedHTTPClient) ProcessLLMRequest(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	ctx, span := c.tracer.Start(ctx, "optimized_http_client.process_llm_request")

	defer span.End()

	start := time.Now()

	// Get request from pool.

	optReq := c.requestPool.Get().(*OptimizedRequest)

	defer c.putRequest(optReq)

	// Get response from pool.

	optResp := c.responsePool.Get().(*OptimizedResponse)

	defer c.putResponse(optResp)

	// Prepare request with zero-copy optimizations.

	if err := c.prepareRequest(optReq, request); err != nil {

		span.SetAttributes(attribute.String("error", err.Error()))

		return nil, fmt.Errorf("failed to prepare request: %w", err)

	}

	// Execute request with connection reuse.

	if err := c.executeRequest(ctx, optReq, optResp); err != nil {

		span.SetAttributes(attribute.String("error", err.Error()))

		c.updateStats(false, time.Since(start))

		return nil, fmt.Errorf("failed to execute request: %w", err)

	}

	// Parse response with fast JSON parsing.

	llmResp, err := c.parseResponse(optResp)
	if err != nil {

		span.SetAttributes(attribute.String("error", err.Error()))

		c.updateStats(false, time.Since(start))

		return nil, fmt.Errorf("failed to parse response: %w", err)

	}

	latency := time.Since(start)

	c.updateStats(true, latency)

	span.SetAttributes(

		attribute.Int("status_code", optResp.StatusCode),

		attribute.Int("response_size", optResp.Size),

		attribute.Int64("latency_ms", latency.Milliseconds()),

		attribute.Bool("from_cache", optResp.FromCache),
	)

	return llmResp, nil
}

// prepareRequest optimizes request preparation with zero-copy techniques.

func (c *OptimizedHTTPClient) prepareRequest(optReq *OptimizedRequest, req *LLMRequest) error {
	// Reset request object.

	optReq.Method = "POST"

	optReq.URL = req.URL

	optReq.ContentType = "application/json"

	// Use buffer pool for JSON marshaling.

	buf := c.bufferPool.Get().([]byte)

	defer func() {
		// Reset buffer and return to pool
		buf = buf[:0]
		c.bufferPool.Put(buf)
	}()

	// Fast JSON encoding using unsafe operations where appropriate.

	jsonData, err := c.fastJSONEncode(req.Payload)
	if err != nil {
		return fmt.Errorf("JSON encoding failed: %w", err)
	}

	// Copy to request body (avoiding extra allocations).

	optReq.Body = make([]byte, len(jsonData))

	copy(optReq.Body, jsonData)

	// Prepare headers.

	if optReq.Headers == nil {
		optReq.Headers = make(map[string]string)
	}

	optReq.Headers["Content-Type"] = "application/json"

	optReq.Headers["User-Agent"] = "nephoran-intent-operator/v2.0.0-optimized"

	if req.APIKey != "" {
		optReq.Headers["Authorization"] = "Bearer " + req.APIKey
	}

	return nil
}

// executeRequest executes the HTTP request with connection reuse optimization.

func (c *OptimizedHTTPClient) executeRequest(ctx context.Context, req *OptimizedRequest, resp *OptimizedResponse) error {
	// Create HTTP request.

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers efficiently.

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Execute with optimized transport.

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}

	defer httpResp.Body.Close()

	// Read response body with buffer reuse.

	buf := c.bufferPool.Get().([]byte)

	defer func() {
		// Reset buffer and return to pool
		buf = buf[:0]
		c.bufferPool.Put(buf)
	}()

	// Use pre-allocated buffer with growth strategy.

	bodyData, err := c.readResponseBody(httpResp.Body, buf)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Populate response.

	resp.StatusCode = httpResp.StatusCode

	resp.Body = make([]byte, len(bodyData))

	copy(resp.Body, bodyData)

	resp.Size = len(bodyData)

	// Copy headers if needed.

	if resp.Headers == nil {
		resp.Headers = make(map[string]string)
	}

	for key, values := range httpResp.Header {
		if len(values) > 0 {
			resp.Headers[key] = values[0]
		}
	}

	return nil
}

// readResponseBody optimized response body reading with buffer management.

func (c *OptimizedHTTPClient) readResponseBody(body io.ReadCloser, buf []byte) ([]byte, error) {
	// Use buffer pool for reading.

	const maxResponseSize = 10 * 1024 * 1024 // 10MB limit

	if cap(buf) < 1024 {
		buf = make([]byte, 0, 4096)
	}

	buf = buf[:0]

	for {

		if len(buf) == cap(buf) {

			if len(buf) > maxResponseSize {
				return nil, fmt.Errorf("response too large: %d bytes", len(buf))
			}

			// Grow buffer.

			newBuf := make([]byte, len(buf), cap(buf)*2)

			copy(newBuf, buf)

			buf = newBuf

		}

		n, err := body.Read(buf[len(buf):cap(buf)])

		buf = buf[:len(buf)+n]

		if err != nil {

			if err == io.EOF {
				break
			}

			return nil, err

		}

	}

	return buf, nil
}

// parseResponse parses LLM response with fast JSON parsing.

func (c *OptimizedHTTPClient) parseResponse(resp *OptimizedResponse) (*LLMResponse, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(resp.Body))
	}

	// Get parser from pool.

	parser := c.parserPool.Get().(*fastjson.Parser)

	defer c.parserPool.Put(parser)

	// Parse JSON efficiently.

	value, err := parser.ParseBytes(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	// Extract response content based on backend type.

	var content string

	// Try OpenAI format first.

	choices := value.GetArray("choices")

	if len(choices) > 0 {

		message := choices[0].Get("message")

		if message != nil {

			contentBytes := message.GetStringBytes("content")

			if contentBytes != nil {
				// Use unsafe conversion to avoid allocation.

				content = *(*string)(unsafe.Pointer(&contentBytes))
			}

		}

	} else {

		// Try direct content extraction.

		contentBytes := value.GetStringBytes("content")

		if contentBytes != nil {
			content = *(*string)(unsafe.Pointer(&contentBytes))
		} else {
			// Fallback: convert entire response.

			content = string(resp.Body)
		}

	}

	return &LLMResponse{
		Content: content,

		StatusCode: resp.StatusCode,

		Latency: resp.Latency,

		Size: resp.Size,

		FromCache: resp.FromCache,

		Metadata: resp.Metadata,
	}, nil
}

// fastJSONEncode performs optimized JSON encoding.

func (c *OptimizedHTTPClient) fastJSONEncode(payload interface{}) ([]byte, error) {
	// Use standard library for now, but could be replaced with faster alternatives.

	// like jsoniter or easyjson for production use.

	return json.Marshal(payload)
}

// updateStats updates client statistics.

func (c *OptimizedHTTPClient) updateStats(success bool, latency time.Duration) {
	c.stats.mutex.Lock()

	defer c.stats.mutex.Unlock()

	c.stats.RequestsTotal++

	if success {
		c.stats.RequestsSuccessful++
	} else {
		c.stats.RequestsFailed++
	}

	c.stats.TotalLatency += latency

	c.stats.AverageLatency = time.Duration(int64(c.stats.TotalLatency) / c.stats.RequestsTotal)
}

// putRequest returns request to pool.

func (c *OptimizedHTTPClient) putRequest(req *OptimizedRequest) {
	// Reset request for reuse.

	req.Method = ""

	req.URL = ""

	req.Body = nil

	req.ContentType = ""

	req.Timeout = 0

	// Clear maps but keep allocated.

	for k := range req.Headers {
		delete(req.Headers, k)
	}

	for k := range req.Metadata {
		delete(req.Metadata, k)
	}

	c.requestPool.Put(req)
}

// putResponse returns response to pool.

func (c *OptimizedHTTPClient) putResponse(resp *OptimizedResponse) {
	// Reset response for reuse.

	resp.StatusCode = 0

	resp.Body = nil

	resp.Size = 0

	resp.Latency = 0

	resp.FromCache = false

	// Clear maps but keep allocated.

	for k := range resp.Headers {
		delete(resp.Headers, k)
	}

	for k := range resp.Metadata {
		delete(resp.Metadata, k)
	}

	c.responsePool.Put(resp)
}

// GetStats returns current HTTP client statistics.

func (c *OptimizedHTTPClient) GetStats() *HTTPClientStats {
	c.stats.mutex.RLock()

	defer c.stats.mutex.RUnlock()

	// Create a copy without the mutex.

	stats := HTTPClientStats{
		RequestsTotal: c.stats.RequestsTotal,

		RequestsSuccessful: c.stats.RequestsSuccessful,

		RequestsFailed: c.stats.RequestsFailed,

		TotalLatency: c.stats.TotalLatency,

		AverageLatency: c.stats.AverageLatency,

		P95Latency: c.stats.P95Latency,

		P99Latency: c.stats.P99Latency,

		ConnectionsCreated: c.stats.ConnectionsCreated,

		ConnectionsReused: c.stats.ConnectionsReused,

		ConnectionsClosed: c.stats.ConnectionsClosed,

		BufferPoolHits: c.stats.BufferPoolHits,

		BufferPoolMisses: c.stats.BufferPoolMisses,
	}

	return &stats
}

// getDefaultOptimizedClientConfig returns default configuration.

func getDefaultOptimizedClientConfig() *OptimizedClientConfig {
	return &OptimizedClientConfig{
		MaxConnsPerHost: 100,

		MaxIdleConns: 50,

		IdleConnTimeout: 90 * time.Second,

		KeepAliveTimeout: 30 * time.Second,

		ConnectTimeout: 10 * time.Second,

		RequestTimeout: 60 * time.Second,

		ResponseTimeout: 30 * time.Second,

		TLSHandshakeTimeout: 10 * time.Second,

		ReadBufferSize: 32 * 1024, // 32KB

		WriteBufferSize: 32 * 1024, // 32KB

		DisableCompression: false,

		DisableKeepAlives: false,

		MaxResponseSize: 10 * 1024 * 1024, // 10MB

		HealthCheckEnabled: true,

		HealthCheckInterval: 30 * time.Second,

		LoadBalancingEnabled: false,

		TLSOptimization: TLSConfig{
			SessionCacheSize: 1000,

			ReuseThreshold: 10,

			SessionTimeout: 24 * time.Hour,

			PreferServerCiphers: true,

			MinVersion: tls.VersionTLS12,
		},
	}
}

// Supporting types for the optimization.

type LLMRequest struct {
	URL string `json:"url"`

	Payload interface{} `json:"payload"`

	APIKey string `json:"api_key,omitempty"`

	Timeout time.Duration `json:"timeout,omitempty"`

	Headers map[string]string `json:"headers,omitempty"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// LLMResponse represents a llmresponse.

type LLMResponse struct {
	Content string `json:"content"`

	StatusCode int `json:"status_code"`

	Latency time.Duration `json:"latency"`

	Size int `json:"size"`

	FromCache bool `json:"from_cache"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// TransportStats represents a transportstats.

type TransportStats struct {
	RoundTrips int64 `json:"round_trips"`

	ConnectionsOpened int64 `json:"connections_opened"`

	ConnectionsReused int64 `json:"connections_reused"`

	TotalLatency time.Duration `json:"total_latency"`

	mutex sync.RWMutex
}
