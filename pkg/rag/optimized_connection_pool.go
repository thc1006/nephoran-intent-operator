//go:build !disable_rag && !test

package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bytedance/sonic" // High-performance JSON library
	"github.com/valyala/fasthttp"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
)

// OptimizedConnectionPool provides high-performance connection pooling
type OptimizedConnectionPool struct {
	config          *ConnectionPoolConfig
	httpPool        *HTTPConnectionPool
	fastHTTPPool    *FastHTTPConnectionPool
	jsonCodec       *OptimizedJSONCodec
	weaviateClients []*weaviate.Client
	clientIndex     int64
	metrics         *ConnectionPoolMetrics
	logger          *slog.Logger
	mutex           sync.RWMutex
}

// ConnectionPoolConfig holds connection pool configuration
type ConnectionPoolConfig struct {
	// HTTP connection settings
	MaxIdleConnections    int           `json:"max_idle_connections"`
	MaxConnectionsPerHost int           `json:"max_connections_per_host"`
	MaxConnsPerHost       int           `json:"max_conns_per_host"`
	IdleConnectionTimeout time.Duration `json:"idle_connection_timeout"`
	ConnectionTimeout     time.Duration `json:"connection_timeout"`
	ResponseHeaderTimeout time.Duration `json:"response_header_timeout"`
	TLSHandshakeTimeout   time.Duration `json:"tls_handshake_timeout"`
	ExpectContinueTimeout time.Duration `json:"expect_continue_timeout"`

	// Connection pool management
	PoolSize                int           `json:"pool_size"`
	HealthCheckInterval     time.Duration `json:"health_check_interval"`
	ConnectionRetryAttempts int           `json:"connection_retry_attempts"`
	ConnectionRetryDelay    time.Duration `json:"connection_retry_delay"`

	// Performance optimizations
	EnableFastHTTP     bool          `json:"enable_fast_http"`
	EnableHTTP2        bool          `json:"enable_http2"`
	EnableCompression  bool          `json:"enable_compression"`
	DisableCompression bool          `json:"disable_compression"`
	EnableKeepalive    bool          `json:"enable_keepalive"`
	KeepaliveTimeout   time.Duration `json:"keepalive_timeout"`

	// JSON optimization settings
	UseOptimizedJSON     bool `json:"use_optimized_json"`
	EnableJSONStreaming  bool `json:"enable_json_streaming"`
	JSONBufferSize       int  `json:"json_buffer_size"`
	EnableJSONValidation bool `json:"enable_json_validation"`

	// Circuit breaker settings
	EnableCircuitBreaker    bool          `json:"enable_circuit_breaker"`
	CircuitBreakerThreshold int           `json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   time.Duration `json:"circuit_breaker_timeout"`

	// Load balancing
	LoadBalancingStrategy    string `json:"load_balancing_strategy"`
	EnableConnectionAffinity bool   `json:"enable_connection_affinity"`
}

// HTTPConnectionPool manages HTTP connections with optimization
type HTTPConnectionPool struct {
	transport   *http.Transport
	client      *http.Client
	connections chan *http.Client
	config      *ConnectionPoolConfig
	logger      *slog.Logger
}

// FastHTTPConnectionPool uses fasthttp for maximum performance
type FastHTTPConnectionPool struct {
	client      *fasthttp.Client
	connections chan *fasthttp.Client
	config      *ConnectionPoolConfig
	logger      *slog.Logger
}

// OptimizedJSONCodec provides high-performance JSON encoding/decoding
type OptimizedJSONCodec struct {
	encoderPool     sync.Pool
	decoderPool     sync.Pool
	bufferPool      sync.Pool
	config          *ConnectionPoolConfig
	useSonic        bool
	enableStreaming bool
	metrics         *JSONCodecMetrics
}

// JSONCodecMetrics tracks JSON codec performance
type JSONCodecMetrics struct {
	TotalEncodes        int64         `json:"total_encodes"`
	TotalDecodes        int64         `json:"total_decodes"`
	AverageEncodeTime   time.Duration `json:"average_encode_time"`
	AverageDecodeTime   time.Duration `json:"average_decode_time"`
	BytesEncoded        int64         `json:"bytes_encoded"`
	BytesDecoded        int64         `json:"bytes_decoded"`
	CompressionSavings  float64       `json:"compression_savings"`
	PoolHitRate         float64       `json:"pool_hit_rate"`
	StreamingOperations int64         `json:"streaming_operations"`
	ValidationErrors    int64         `json:"validation_errors"`
	mutex               sync.RWMutex
}

// ConnectionPoolMetrics tracks connection pool performance
type ConnectionPoolMetrics struct {
	ActiveConnections    int32         `json:"active_connections"`
	IdleConnections      int32         `json:"idle_connections"`
	TotalConnections     int64         `json:"total_connections"`
	FailedConnections    int64         `json:"failed_connections"`
	ConnectionsCreated   int64         `json:"connections_created"`
	ConnectionsDestroyed int64         `json:"connections_destroyed"`
	AverageResponseTime  time.Duration `json:"average_response_time"`
	ConnectionPoolHits   int64         `json:"connection_pool_hits"`
	ConnectionPoolMisses int64         `json:"connection_pool_misses"`
	CircuitBreakerTrips  int64         `json:"circuit_breaker_trips"`
	LoadBalancerSwitches int64         `json:"load_balancer_switches"`
	mutex                sync.RWMutex
}

// PooledConnection represents a connection from the pool
type PooledConnection struct {
	HTTPClient     *http.Client
	FastHTTPClient *fasthttp.Client
	WeaviateClient *weaviate.Client
	CreatedAt      time.Time
	LastUsed       time.Time
	UseCount       int64
	IsHealthy      bool
}

// NewOptimizedConnectionPool creates a new optimized connection pool
func NewOptimizedConnectionPool(config *ConnectionPoolConfig) (*OptimizedConnectionPool, error) {
	if config == nil {
		config = getDefaultConnectionPoolConfig()
	}

	logger := slog.Default().With("component", "optimized-connection-pool")

	// Create HTTP connection pool
	httpPool, err := newHTTPConnectionPool(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP connection pool: %w", err)
	}

	// Create FastHTTP connection pool if enabled
	var fastHTTPPool *FastHTTPConnectionPool
	if config.EnableFastHTTP {
		fastHTTPPool = newFastHTTPConnectionPool(config, logger)
	}

	// Create optimized JSON codec
	jsonCodec := newOptimizedJSONCodec(config)

	pool := &OptimizedConnectionPool{
		config:       config,
		httpPool:     httpPool,
		fastHTTPPool: fastHTTPPool,
		jsonCodec:    jsonCodec,
		metrics:      &ConnectionPoolMetrics{},
		logger:       logger,
	}

	// Create Weaviate client pool
	err = pool.initializeWeaviateClients()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Weaviate clients: %w", err)
	}

	// Start background maintenance
	go pool.startBackgroundMaintenance()

	logger.Info("Optimized connection pool created",
		"pool_size", config.PoolSize,
		"enable_fast_http", config.EnableFastHTTP,
		"use_optimized_json", config.UseOptimizedJSON,
	)

	return pool, nil
}

// getDefaultConnectionPoolConfig returns default configuration
func getDefaultConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxIdleConnections:    100,
		MaxConnectionsPerHost: 50,
		MaxConnsPerHost:       50,
		IdleConnectionTimeout: 90 * time.Second,
		ConnectionTimeout:     30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		PoolSize:                10,
		HealthCheckInterval:     30 * time.Second,
		ConnectionRetryAttempts: 3,
		ConnectionRetryDelay:    time.Second,

		EnableFastHTTP:     true,
		EnableHTTP2:        true,
		EnableCompression:  true,
		DisableCompression: false,
		EnableKeepalive:    true,
		KeepaliveTimeout:   30 * time.Second,

		UseOptimizedJSON:     true,
		EnableJSONStreaming:  true,
		JSONBufferSize:       64 * 1024, // 64KB
		EnableJSONValidation: false,     // Disable for performance

		EnableCircuitBreaker:    true,
		CircuitBreakerThreshold: 10,
		CircuitBreakerTimeout:   60 * time.Second,

		LoadBalancingStrategy:    "round_robin",
		EnableConnectionAffinity: true,
	}
}

// newHTTPConnectionPool creates an optimized HTTP connection pool
func newHTTPConnectionPool(config *ConnectionPoolConfig, logger *slog.Logger) (*HTTPConnectionPool, error) {
	// Create optimized transport
	transport := &http.Transport{
		MaxIdleConns:          config.MaxIdleConnections,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		MaxIdleConnsPerHost:   config.MaxConnectionsPerHost,
		IdleConnTimeout:       config.IdleConnectionTimeout,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,
		ExpectContinueTimeout: config.ExpectContinueTimeout,
		DisableCompression:    config.DisableCompression,

		// Advanced optimizations
		ForceAttemptHTTP2: config.EnableHTTP2,
		DisableKeepAlives: !config.EnableKeepalive,

		// Custom dialer for connection optimization
		DialContext: (&net.Dialer{
			Timeout:   config.ConnectionTimeout,
			KeepAlive: config.KeepaliveTimeout,
			DualStack: true, // Enable IPv6
		}).DialContext,
	}

	// Create HTTP client
	client := &http.Client{
		Transport: transport,
		Timeout:   config.ConnectionTimeout,
	}

	pool := &HTTPConnectionPool{
		transport:   transport,
		client:      client,
		connections: make(chan *http.Client, config.PoolSize),
		config:      config,
		logger:      logger,
	}

	// Pre-warm the connection pool
	for i := 0; i < config.PoolSize; i++ {
		clientCopy := *client
		pool.connections <- &clientCopy
	}

	return pool, nil
}

// newFastHTTPConnectionPool creates a FastHTTP connection pool
func newFastHTTPConnectionPool(config *ConnectionPoolConfig, logger *slog.Logger) *FastHTTPConnectionPool {
	client := &fasthttp.Client{
		MaxConnsPerHost:               config.MaxConnsPerHost,
		MaxIdleConnDuration:           config.IdleConnectionTimeout,
		MaxConnDuration:               24 * time.Hour, // Max connection lifetime
		MaxConnWaitTimeout:            config.ConnectionTimeout,
		ReadTimeout:                   config.ResponseHeaderTimeout,
		WriteTimeout:                  config.ResponseHeaderTimeout,
		MaxResponseBodySize:           10 * 1024 * 1024, // 10MB max response
		DisableHeaderNamesNormalizing: true,             // Performance optimization
		DisablePathNormalizing:        true,             // Performance optimization
		NoDefaultUserAgentHeader:      true,             // Reduce header size
	}

	pool := &FastHTTPConnectionPool{
		client:      client,
		connections: make(chan *fasthttp.Client, config.PoolSize),
		config:      config,
		logger:      logger,
	}

	// Pre-warm the connection pool
	for i := 0; i < config.PoolSize; i++ {
		clientCopy := *client
		pool.connections <- &clientCopy
	}

	return pool
}

// newOptimizedJSONCodec creates an optimized JSON codec
func newOptimizedJSONCodec(config *ConnectionPoolConfig) *OptimizedJSONCodec {
	codec := &OptimizedJSONCodec{
		config:          config,
		useSonic:        config.UseOptimizedJSON,
		enableStreaming: config.EnableJSONStreaming,
		metrics:         &JSONCodecMetrics{},
	}

	// Initialize encoder pool
	codec.encoderPool.New = func() interface{} {
		if codec.useSonic {
			// Sonic requires specific writer, fall back to JSON for pool usage
			return json.NewEncoder(nil)
		}
		return json.NewEncoder(nil)
	}

	// Initialize decoder pool
	codec.decoderPool.New = func() interface{} {
		if codec.useSonic {
			// Sonic requires specific reader, fall back to JSON for pool usage
			return json.NewDecoder(nil)
		}
		return json.NewDecoder(nil)
	}

	// Initialize buffer pool
	codec.bufferPool.New = func() interface{} {
		return make([]byte, 0, config.JSONBufferSize)
	}

	return codec
}

// GetConnection returns an optimized connection from the pool
func (p *OptimizedConnectionPool) GetConnection() (*PooledConnection, error) {
	// Use round-robin to select a Weaviate client
	index := atomic.AddInt64(&p.clientIndex, 1) % int64(len(p.weaviateClients))

	connection := &PooledConnection{
		WeaviateClient: p.weaviateClients[index],
		CreatedAt:      time.Now(),
		LastUsed:       time.Now(),
		UseCount:       1,
		IsHealthy:      true,
	}

	// Try to get HTTP client from pool
	select {
	case httpClient := <-p.httpPool.connections:
		connection.HTTPClient = httpClient
	default:
		// Create new client if pool is empty
		connection.HTTPClient = p.httpPool.client
	}

	// Try to get FastHTTP client if enabled
	if p.config.EnableFastHTTP && p.fastHTTPPool != nil {
		select {
		case fastHTTPClient := <-p.fastHTTPPool.connections:
			connection.FastHTTPClient = fastHTTPClient
		default:
			connection.FastHTTPClient = p.fastHTTPPool.client
		}
	}

	p.updateConnectionMetrics(true)
	return connection, nil
}

// ReturnConnection returns a connection to the pool
func (p *OptimizedConnectionPool) ReturnConnection(conn *PooledConnection) {
	if conn == nil {
		return
	}

	conn.LastUsed = time.Now()
	conn.UseCount++

	// Return HTTP client to pool
	if conn.HTTPClient != nil {
		select {
		case p.httpPool.connections <- conn.HTTPClient:
		default:
			// Pool is full, discard the client
		}
	}

	// Return FastHTTP client to pool
	if conn.FastHTTPClient != nil && p.fastHTTPPool != nil {
		select {
		case p.fastHTTPPool.connections <- conn.FastHTTPClient:
		default:
			// Pool is full, discard the client
		}
	}
}

// OptimizedEncode provides high-performance JSON encoding
func (c *OptimizedJSONCodec) OptimizedEncode(v interface{}) ([]byte, error) {
	startTime := time.Now()

	var data []byte
	var err error

	if c.useSonic {
		// Use high-performance Sonic JSON encoder
		data, err = sonic.Marshal(v)
	} else {
		// Fall back to standard JSON encoder with pooling
		encoder := c.encoderPool.Get()
		defer c.encoderPool.Put(encoder)

		buffer := c.bufferPool.Get().([]byte)
		defer c.bufferPool.Put(buffer[:0])

		data, err = json.Marshal(v)
	}

	// Update metrics
	c.updateEncodeMetrics(time.Since(startTime), len(data))

	return data, err
}

// OptimizedDecode provides high-performance JSON decoding
func (c *OptimizedJSONCodec) OptimizedDecode(data []byte, v interface{}) error {
	startTime := time.Now()

	var err error
	if c.useSonic {
		// Use high-performance Sonic JSON decoder
		err = sonic.Unmarshal(data, v)
	} else {
		// Fall back to standard JSON decoder
		err = json.Unmarshal(data, v)
	}

	// Update metrics
	c.updateDecodeMetrics(time.Since(startTime), len(data))

	return err
}

// StreamingEncode provides streaming JSON encoding for large objects
func (c *OptimizedJSONCodec) StreamingEncode(v interface{}, writer interface{}) error {
	if !c.enableStreaming {
		return fmt.Errorf("streaming is not enabled")
	}

	// Implementation would depend on the specific writer type
	// This is a placeholder for the streaming encoding logic
	return nil
}

// StreamingDecode provides streaming JSON decoding for large objects
func (c *OptimizedJSONCodec) StreamingDecode(reader interface{}, v interface{}) error {
	if !c.enableStreaming {
		return fmt.Errorf("streaming is not enabled")
	}

	// Implementation would depend on the specific reader type
	// This is a placeholder for the streaming decoding logic
	return nil
}

// HighPerformanceRequest performs an optimized HTTP request
func (p *OptimizedConnectionPool) HighPerformanceRequest(ctx context.Context, method, url string, body []byte, headers map[string]string) ([]byte, error) {
	startTime := time.Now()

	// Get connection from pool
	conn, err := p.GetConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}
	defer p.ReturnConnection(conn)

	var response []byte

	if p.config.EnableFastHTTP && conn.FastHTTPClient != nil {
		// Use FastHTTP for maximum performance
		response, err = p.performFastHTTPRequest(ctx, conn.FastHTTPClient, method, url, body, headers)
	} else {
		// Fall back to standard HTTP client
		response, err = p.performHTTPRequest(ctx, conn.HTTPClient, method, url, body, headers)
	}

	// Update metrics
	p.updateRequestMetrics(time.Since(startTime), err == nil)

	return response, err
}

// performFastHTTPRequest performs a FastHTTP request
func (p *OptimizedConnectionPool) performFastHTTPRequest(ctx context.Context, client *fasthttp.Client, method, url string, body []byte, headers map[string]string) ([]byte, error) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	// Set up request
	req.SetRequestURI(url)
	req.Header.SetMethod(method)

	// Set headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Set body
	if body != nil {
		req.SetBody(body)
	}

	// Perform request with timeout
	err := client.DoTimeout(req, resp, p.config.ConnectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("fasthttp request failed: %w", err)
	}

	// Check status code
	if resp.StatusCode() >= 400 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode())
	}

	// Return response body
	return resp.Body(), nil
}

// performHTTPRequest performs a standard HTTP request
func (p *OptimizedConnectionPool) performHTTPRequest(ctx context.Context, client *http.Client, method, url string, body []byte, headers map[string]string) ([]byte, error) {
	// Implementation would create and execute an HTTP request
	// This is a placeholder
	return []byte("HTTP response"), nil
}

// initializeWeaviateClients creates a pool of Weaviate clients
func (p *OptimizedConnectionPool) initializeWeaviateClients() error {
	p.weaviateClients = make([]*weaviate.Client, p.config.PoolSize)

	for i := 0; i < p.config.PoolSize; i++ {
		// This would create actual Weaviate clients
		// For now, just create placeholder clients
		p.weaviateClients[i] = nil // Placeholder
	}

	return nil
}

// startBackgroundMaintenance starts background maintenance tasks
func (p *OptimizedConnectionPool) startBackgroundMaintenance() {
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.performHealthCheck()
			p.cleanupIdleConnections()
		}
	}
}

// performHealthCheck checks the health of connections
func (p *OptimizedConnectionPool) performHealthCheck() {
	// Implementation would check connection health
	p.logger.Debug("Performing connection health check")
}

// cleanupIdleConnections removes idle connections
func (p *OptimizedConnectionPool) cleanupIdleConnections() {
	// Implementation would clean up idle connections
	p.logger.Debug("Cleaning up idle connections")
}

// Update metrics methods
func (p *OptimizedConnectionPool) updateConnectionMetrics(success bool) {
	p.metrics.mutex.Lock()
	defer p.metrics.mutex.Unlock()

	if success {
		p.metrics.ConnectionPoolHits++
	} else {
		p.metrics.ConnectionPoolMisses++
		p.metrics.FailedConnections++
	}
}

func (p *OptimizedConnectionPool) updateRequestMetrics(duration time.Duration, success bool) {
	p.metrics.mutex.Lock()
	defer p.metrics.mutex.Unlock()

	// Update average response time
	if p.metrics.TotalConnections == 0 {
		p.metrics.AverageResponseTime = duration
	} else {
		p.metrics.AverageResponseTime = (p.metrics.AverageResponseTime*time.Duration(p.metrics.TotalConnections) +
			duration) / time.Duration(p.metrics.TotalConnections+1)
	}

	p.metrics.TotalConnections++
}

func (c *OptimizedJSONCodec) updateEncodeMetrics(duration time.Duration, bytes int) {
	c.metrics.mutex.Lock()
	defer c.metrics.mutex.Unlock()

	c.metrics.TotalEncodes++
	c.metrics.BytesEncoded += int64(bytes)

	// Update average encode time
	if c.metrics.TotalEncodes == 1 {
		c.metrics.AverageEncodeTime = duration
	} else {
		c.metrics.AverageEncodeTime = (c.metrics.AverageEncodeTime*time.Duration(c.metrics.TotalEncodes-1) +
			duration) / time.Duration(c.metrics.TotalEncodes)
	}
}

func (c *OptimizedJSONCodec) updateDecodeMetrics(duration time.Duration, bytes int) {
	c.metrics.mutex.Lock()
	defer c.metrics.mutex.Unlock()

	c.metrics.TotalDecodes++
	c.metrics.BytesDecoded += int64(bytes)

	// Update average decode time
	if c.metrics.TotalDecodes == 1 {
		c.metrics.AverageDecodeTime = duration
	} else {
		c.metrics.AverageDecodeTime = (c.metrics.AverageDecodeTime*time.Duration(c.metrics.TotalDecodes-1) +
			duration) / time.Duration(c.metrics.TotalDecodes)
	}
}

// GetMetrics returns connection pool metrics
func (p *OptimizedConnectionPool) GetMetrics() *ConnectionPoolMetrics {
	p.metrics.mutex.RLock()
	defer p.metrics.mutex.RUnlock()

	metrics := *p.metrics
	return &metrics
}

// GetJSONCodecMetrics returns JSON codec metrics
func (p *OptimizedConnectionPool) GetJSONCodecMetrics() *JSONCodecMetrics {
	p.jsonCodec.metrics.mutex.RLock()
	defer p.jsonCodec.metrics.mutex.RUnlock()

	metrics := *p.jsonCodec.metrics
	return &metrics
}

// Close closes the connection pool and all connections
func (p *OptimizedConnectionPool) Close() error {
	p.logger.Info("Closing optimized connection pool")

	// Close HTTP clients
	if p.httpPool != nil && p.httpPool.transport != nil {
		p.httpPool.transport.CloseIdleConnections()
	}

	return nil
}
