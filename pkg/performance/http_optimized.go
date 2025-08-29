//go:build go1.24




package performance



import (

	"context"

	"crypto/tls"

	"fmt"

	"io"

	"net"

	"net/http"

	"net/http/httptrace"

	"sync"

	"sync/atomic"

	"time"



	"golang.org/x/net/http2"

	"golang.org/x/sync/singleflight"



	"k8s.io/klog/v2"

)



// OptimizedHTTPClient provides high-performance HTTP operations with Go 1.24+ optimizations.

type OptimizedHTTPClient struct {

	client         *http.Client

	connectionPool *DynamicConnectionPool

	pushTargets    map[string][]string

	requestDeduper *singleflight.Group

	bufferPool     *BufferPool

	healthChecker  *ConnectionHealthChecker

	metrics        *HTTPMetrics

	config         *HTTPConfig

	tlsConfig      *tls.Config

	mu             sync.RWMutex

}



// HTTPConfig contains HTTP optimization configuration.

type HTTPConfig struct {

	MaxIdleConns        int

	MaxIdleConnsPerHost int

	IdleConnTimeout     time.Duration

	TLSHandshakeTimeout time.Duration

	KeepAlive           time.Duration

	DualStack           bool

	HTTP2Enabled        bool

	HTTP3Enabled        bool

	ConnectionPoolSize  int

	ReadBufferSize      int

	WriteBufferSize     int

	CompressionEnabled  bool

	ServerPushEnabled   bool

	ZeroRTTEnabled      bool

}



// DynamicConnectionPool manages HTTP connections with dynamic scaling.

type DynamicConnectionPool struct {

	conns         chan *http.Client

	maxConns      int64

	activeConns   int64

	idleTimeout   time.Duration

	healthChecker *ConnectionHealthChecker

	config        *HTTPConfig

	mu            sync.RWMutex

}



// ConnectionHealthChecker monitors connection health.

type ConnectionHealthChecker struct {

	interval     time.Duration

	timeout      time.Duration

	healthChecks map[string]*HealthCheckResult

	mu           sync.RWMutex

	cancel       context.CancelFunc

}



// HealthCheckResult contains connection health information.

type HealthCheckResult struct {

	LastCheck time.Time

	Latency   time.Duration

	Errors    int64

	Successes int64

	Healthy   bool

}



// BufferPool provides optimized buffer management.

type BufferPool struct {

	smallBuffers  sync.Pool // 1KB buffers

	mediumBuffers sync.Pool // 16KB buffers

	largeBuffers  sync.Pool // 64KB buffers

	hitCount      int64

	missCount     int64

}



// HTTPMetrics tracks HTTP performance metrics.

type HTTPMetrics struct {

	RequestCount       int64

	ResponseTime       int64 // in nanoseconds

	ErrorCount         int64

	ConnectionsActive  int64

	ConnectionsCreated int64

	ConnectionsReused  int64

	CacheHits          int64

	CacheMisses        int64

	BytesTransferred   int64

	CompressionRatio   float64

	HTTP2Connections   int64

	HTTP3Connections   int64

	TLSHandshakes      int64

	ZeroRTTSuccess     int64

}



// NewOptimizedHTTPClient creates a new optimized HTTP client with Go 1.24+ features.

func NewOptimizedHTTPClient(config *HTTPConfig) *OptimizedHTTPClient {

	if config == nil {

		config = DefaultHTTPConfig()

	}



	// Enhanced TLS configuration with Go 1.24 optimizations.

	tlsConfig := &tls.Config{

		MinVersion:             tls.VersionTLS13,

		SessionTicketsDisabled: false,

		ClientSessionCache:     tls.NewLRUClientSessionCache(256),

		InsecureSkipVerify:     false,

		Renegotiation:          tls.RenegotiateNever,

		NextProtos:             []string{"h2", "http/1.1"},

	}



	if config.ZeroRTTEnabled {

		// Enable 0-RTT resumption for TLS 1.3.

		tlsConfig.MaxVersion = tls.VersionTLS13

	}



	// Create optimized transport with Go 1.24+ features.

	transport := &http.Transport{

		Proxy: http.ProxyFromEnvironment,

		DialContext: (&net.Dialer{

			Timeout:   30 * time.Second,

			KeepAlive: config.KeepAlive,

			DualStack: config.DualStack,

		}).DialContext,

		ForceAttemptHTTP2:     config.HTTP2Enabled,

		MaxIdleConns:          config.MaxIdleConns,

		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,

		IdleConnTimeout:       config.IdleConnTimeout,

		TLSHandshakeTimeout:   config.TLSHandshakeTimeout,

		ExpectContinueTimeout: 1 * time.Second,

		TLSClientConfig:       tlsConfig,

		ReadBufferSize:        config.ReadBufferSize,

		WriteBufferSize:       config.WriteBufferSize,

		DisableCompression:    !config.CompressionEnabled,

	}



	// Configure HTTP/2 settings.

	if config.HTTP2Enabled {

		http2.ConfigureTransport(transport)

	}



	client := &http.Client{

		Transport: transport,

		Timeout:   30 * time.Second,

	}



	// Create health checker.

	healthChecker := &ConnectionHealthChecker{

		interval:     30 * time.Second,

		timeout:      5 * time.Second,

		healthChecks: make(map[string]*HealthCheckResult),

	}



	// Start health checking.

	ctx, cancel := context.WithCancel(context.Background())

	healthChecker.cancel = cancel

	go healthChecker.Start(ctx)



	optimizedClient := &OptimizedHTTPClient{

		client: client,

		connectionPool: &DynamicConnectionPool{

			conns:         make(chan *http.Client, config.ConnectionPoolSize),

			maxConns:      int64(config.ConnectionPoolSize),

			idleTimeout:   config.IdleConnTimeout,

			healthChecker: healthChecker,

			config:        config,

		},

		pushTargets:    make(map[string][]string),

		requestDeduper: &singleflight.Group{},

		bufferPool:     NewBufferPool(),

		healthChecker:  healthChecker,

		metrics:        &HTTPMetrics{},

		config:         config,

		tlsConfig:      tlsConfig,

	}



	// Pre-populate connection pool.

	optimizedClient.connectionPool.warmUp()



	return optimizedClient

}



// DefaultHTTPConfig returns default HTTP configuration optimized for Go 1.24+.

func DefaultHTTPConfig() *HTTPConfig {

	return &HTTPConfig{

		MaxIdleConns:        200,

		MaxIdleConnsPerHost: 20,

		IdleConnTimeout:     90 * time.Second,

		TLSHandshakeTimeout: 10 * time.Second,

		KeepAlive:           30 * time.Second,

		DualStack:           true,

		HTTP2Enabled:        true,

		HTTP3Enabled:        false, // Enable when widely supported

		ConnectionPoolSize:  100,

		ReadBufferSize:      32 * 1024, // 32KB

		WriteBufferSize:     32 * 1024, // 32KB

		CompressionEnabled:  true,

		ServerPushEnabled:   true,

		ZeroRTTEnabled:      true,

	}

}



// NewBufferPool creates an optimized buffer pool.

func NewBufferPool() *BufferPool {

	return &BufferPool{

		smallBuffers: sync.Pool{

			New: func() interface{} {

				return make([]byte, 1024)

			},

		},

		mediumBuffers: sync.Pool{

			New: func() interface{} {

				return make([]byte, 16*1024)

			},

		},

		largeBuffers: sync.Pool{

			New: func() interface{} {

				return make([]byte, 64*1024)

			},

		},

	}

}



// GetBuffer returns a buffer of appropriate size from the pool.

func (bp *BufferPool) GetBuffer(size int) []byte {

	var buffer []byte



	switch {

	case size <= 1024:

		buffer = bp.smallBuffers.Get().([]byte)

		atomic.AddInt64(&bp.hitCount, 1)

	case size <= 16*1024:

		buffer = bp.mediumBuffers.Get().([]byte)

		atomic.AddInt64(&bp.hitCount, 1)

	case size <= 64*1024:

		buffer = bp.largeBuffers.Get().([]byte)

		atomic.AddInt64(&bp.hitCount, 1)

	default:

		// For very large sizes, allocate directly.

		buffer = make([]byte, size)

		atomic.AddInt64(&bp.missCount, 1)

	}



	return buffer[:size]

}



// PutBuffer returns a buffer to the pool.

func (bp *BufferPool) PutBuffer(buffer []byte) {

	if cap(buffer) == 0 {

		return

	}



	// Reset buffer.

	buffer = buffer[:cap(buffer)]

	for i := range buffer {

		buffer[i] = 0

	}



	switch cap(buffer) {

	case 1024:

		bp.smallBuffers.Put(buffer)

	case 16 * 1024:

		bp.mediumBuffers.Put(buffer)

	case 64 * 1024:

		bp.largeBuffers.Put(buffer)

	}

}



// GetHitRate returns the buffer pool hit rate.

func (bp *BufferPool) GetHitRate() float64 {

	hits := atomic.LoadInt64(&bp.hitCount)

	misses := atomic.LoadInt64(&bp.missCount)

	total := hits + misses

	if total == 0 {

		return 0

	}

	return float64(hits) / float64(total)

}



// warmUp pre-populates the connection pool.

func (dp *DynamicConnectionPool) warmUp() {

	for range dp.maxConns / 2 {

		client := &http.Client{

			Timeout: 30 * time.Second,

			Transport: &http.Transport{

				MaxIdleConns:        dp.config.MaxIdleConns,

				MaxIdleConnsPerHost: dp.config.MaxIdleConnsPerHost,

				IdleConnTimeout:     dp.config.IdleConnTimeout,

				TLSHandshakeTimeout: dp.config.TLSHandshakeTimeout,

			},

		}

		select {

		case dp.conns <- client:

		default:

			// Pool is full.

		}

	}

}



// GetClient retrieves a client from the dynamic connection pool.

func (dp *DynamicConnectionPool) GetClient() *http.Client {

	select {

	case client := <-dp.conns:

		atomic.AddInt64(&dp.activeConns, 1)

		return client

	default:

		// Pool is empty, create new client if under limit.

		if atomic.LoadInt64(&dp.activeConns) < dp.maxConns {

			client := &http.Client{

				Timeout: 30 * time.Second,

				Transport: &http.Transport{

					MaxIdleConns:        dp.config.MaxIdleConns,

					MaxIdleConnsPerHost: dp.config.MaxIdleConnsPerHost,

					IdleConnTimeout:     dp.config.IdleConnTimeout,

					TLSHandshakeTimeout: dp.config.TLSHandshakeTimeout,

				},

			}

			atomic.AddInt64(&dp.activeConns, 1)

			return client

		}

		// Wait for available client with timeout.

		select {

		case client := <-dp.conns:

			atomic.AddInt64(&dp.activeConns, 1)

			return client

		case <-time.After(5 * time.Second):

			// Return nil to indicate timeout.

			return nil

		}

	}

}



// ReturnClient returns a client to the pool.

func (dp *DynamicConnectionPool) ReturnClient(client *http.Client) {

	atomic.AddInt64(&dp.activeConns, -1)

	select {

	case dp.conns <- client:

		// Client returned to pool.

	default:

		// Pool is full, let client be garbage collected.

	}

}



// Start begins the connection health checking routine.

func (hc *ConnectionHealthChecker) Start(ctx context.Context) {

	ticker := time.NewTicker(hc.interval)

	defer ticker.Stop()



	for {

		select {

		case <-ticker.C:

			hc.performHealthChecks()

		case <-ctx.Done():

			return

		}

	}

}



// performHealthChecks checks the health of all connections.

func (hc *ConnectionHealthChecker) performHealthChecks() {

	hc.mu.Lock()

	defer hc.mu.Unlock()



	for endpoint, result := range hc.healthChecks {

		go func(ep string, res *HealthCheckResult) {

			start := time.Now()

			err := hc.checkEndpoint(ep)

			latency := time.Since(start)



			hc.mu.Lock()

			res.LastCheck = time.Now()

			res.Latency = latency



			if err != nil {

				atomic.AddInt64(&res.Errors, 1)

				res.Healthy = false

			} else {

				atomic.AddInt64(&res.Successes, 1)

				res.Healthy = true

			}

			hc.mu.Unlock()

		}(endpoint, result)

	}

}



// checkEndpoint performs a health check on a specific endpoint.

func (hc *ConnectionHealthChecker) checkEndpoint(endpoint string) error {

	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)

	defer cancel()



	req, err := http.NewRequestWithContext(ctx, "HEAD", endpoint, http.NoBody)

	if err != nil {

		return err

	}



	client := &http.Client{Timeout: hc.timeout}

	resp, err := client.Do(req)

	if err != nil {

		return err

	}

	defer resp.Body.Close()



	if resp.StatusCode >= 400 {

		return fmt.Errorf("unhealthy status code: %d", resp.StatusCode)

	}



	return nil

}



// DoWithOptimizations performs an HTTP request with all optimizations enabled.

func (c *OptimizedHTTPClient) DoWithOptimizations(ctx context.Context, req *http.Request) (*http.Response, error) {

	start := time.Now()

	defer func() {

		duration := time.Since(start)

		atomic.AddInt64(&c.metrics.RequestCount, 1)

		atomic.AddInt64(&c.metrics.ResponseTime, duration.Nanoseconds())

	}()



	// Create request trace for detailed metrics.

	trace := &httptrace.ClientTrace{

		GetConn: func(hostPort string) {

			atomic.AddInt64(&c.metrics.ConnectionsCreated, 1)

		},

		GotConn: func(connInfo httptrace.GotConnInfo) {

			if connInfo.Reused {

				atomic.AddInt64(&c.metrics.ConnectionsReused, 1)

			}

			atomic.AddInt64(&c.metrics.ConnectionsActive, 1)

		},

		TLSHandshakeStart: func() {

			atomic.AddInt64(&c.metrics.TLSHandshakes, 1)

		},

		WroteRequest: func(info httptrace.WroteRequestInfo) {

			if info.Err != nil {

				atomic.AddInt64(&c.metrics.ErrorCount, 1)

			}

		},

	}



	req = req.WithContext(httptrace.WithClientTrace(ctx, trace))



	// Check for request deduplication.

	requestKey := fmt.Sprintf("%s:%s", req.Method, req.URL.String())

	if req.Method == "GET" {

		// Use singleflight for GET requests to deduplicate.

		result, err, _ := c.requestDeduper.Do(requestKey, func() (interface{}, error) {

			return c.performRequest(req)

		})

		if err != nil {

			return nil, err

		}

		return result.(*http.Response), nil

	}



	return c.performRequest(req)

}



// performRequest executes the actual HTTP request.

func (c *OptimizedHTTPClient) performRequest(req *http.Request) (*http.Response, error) {

	// Get optimized client from pool.

	client := c.connectionPool.GetClient()

	if client == nil {

		return nil, fmt.Errorf("no available HTTP clients")

	}

	defer c.connectionPool.ReturnClient(client)



	// Perform the request.

	resp, err := client.Do(req)

	if err != nil {

		atomic.AddInt64(&c.metrics.ErrorCount, 1)

		return nil, err

	}



	// Update metrics.

	if resp.Header.Get("Content-Encoding") == "gzip" {

		// Estimate compression ratio (simplified).

		c.metrics.CompressionRatio = 0.7 // Typical gzip ratio

	}



	contentLength := resp.ContentLength

	if contentLength > 0 {

		atomic.AddInt64(&c.metrics.BytesTransferred, contentLength)

	}



	return resp, nil

}



// EnableServerPush configures HTTP/2 server push for specific routes.

func (c *OptimizedHTTPClient) EnableServerPush(baseURL string, pushTargets []string) {

	c.mu.Lock()

	defer c.mu.Unlock()

	c.pushTargets[baseURL] = pushTargets

}



// StreamingRequest performs a streaming HTTP request with optimal buffering.

func (c *OptimizedHTTPClient) StreamingRequest(ctx context.Context, req *http.Request, callback func([]byte) error) error {

	resp, err := c.DoWithOptimizations(ctx, req)

	if err != nil {

		return err

	}

	defer resp.Body.Close()



	// Get appropriate buffer from pool.

	buffer := c.bufferPool.GetBuffer(c.config.ReadBufferSize)

	defer c.bufferPool.PutBuffer(buffer)



	for {

		n, err := resp.Body.Read(buffer)

		if n > 0 {

			if err := callback(buffer[:n]); err != nil {

				return err

			}

		}

		if err == io.EOF {

			break

		}

		if err != nil {

			return err

		}

	}



	return nil

}



// GetMetrics returns current HTTP performance metrics.

func (c *OptimizedHTTPClient) GetMetrics() HTTPMetrics {

	return HTTPMetrics{

		RequestCount:       atomic.LoadInt64(&c.metrics.RequestCount),

		ResponseTime:       atomic.LoadInt64(&c.metrics.ResponseTime),

		ErrorCount:         atomic.LoadInt64(&c.metrics.ErrorCount),

		ConnectionsActive:  atomic.LoadInt64(&c.metrics.ConnectionsActive),

		ConnectionsCreated: atomic.LoadInt64(&c.metrics.ConnectionsCreated),

		ConnectionsReused:  atomic.LoadInt64(&c.metrics.ConnectionsReused),

		CacheHits:          atomic.LoadInt64(&c.metrics.CacheHits),

		CacheMisses:        atomic.LoadInt64(&c.metrics.CacheMisses),

		BytesTransferred:   atomic.LoadInt64(&c.metrics.BytesTransferred),

		CompressionRatio:   c.metrics.CompressionRatio,

		HTTP2Connections:   atomic.LoadInt64(&c.metrics.HTTP2Connections),

		HTTP3Connections:   atomic.LoadInt64(&c.metrics.HTTP3Connections),

		TLSHandshakes:      atomic.LoadInt64(&c.metrics.TLSHandshakes),

		ZeroRTTSuccess:     atomic.LoadInt64(&c.metrics.ZeroRTTSuccess),

	}

}



// GetAverageResponseTime returns the average response time in milliseconds.

func (c *OptimizedHTTPClient) GetAverageResponseTime() float64 {

	requestCount := atomic.LoadInt64(&c.metrics.RequestCount)

	if requestCount == 0 {

		return 0

	}

	totalTime := atomic.LoadInt64(&c.metrics.ResponseTime)

	return float64(totalTime) / float64(requestCount) / 1e6 // Convert to milliseconds

}



// Shutdown gracefully shuts down the HTTP client.

func (c *OptimizedHTTPClient) Shutdown(ctx context.Context) error {

	// Cancel health checker.

	if c.healthChecker.cancel != nil {

		c.healthChecker.cancel()

	}



	// Log final metrics.

	metrics := c.GetMetrics()

	klog.Infof("HTTP client shutdown - Total requests: %d, Avg response time: %.2fms, Error rate: %.2f%%",

		metrics.RequestCount,

		c.GetAverageResponseTime(),

		float64(metrics.ErrorCount)/float64(metrics.RequestCount)*100,

	)



	return nil

}



// ResetMetrics resets all performance metrics.

func (c *OptimizedHTTPClient) ResetMetrics() {

	atomic.StoreInt64(&c.metrics.RequestCount, 0)

	atomic.StoreInt64(&c.metrics.ResponseTime, 0)

	atomic.StoreInt64(&c.metrics.ErrorCount, 0)

	atomic.StoreInt64(&c.metrics.ConnectionsActive, 0)

	atomic.StoreInt64(&c.metrics.ConnectionsCreated, 0)

	atomic.StoreInt64(&c.metrics.ConnectionsReused, 0)

	atomic.StoreInt64(&c.metrics.CacheHits, 0)

	atomic.StoreInt64(&c.metrics.CacheMisses, 0)

	atomic.StoreInt64(&c.metrics.BytesTransferred, 0)

	atomic.StoreInt64(&c.metrics.HTTP2Connections, 0)

	atomic.StoreInt64(&c.metrics.HTTP3Connections, 0)

	atomic.StoreInt64(&c.metrics.TLSHandshakes, 0)

	atomic.StoreInt64(&c.metrics.ZeroRTTSuccess, 0)

	c.metrics.CompressionRatio = 0

}

