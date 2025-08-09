package rag

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
)

// WeaviatePoolMetricsRecorder defines an interface for recording Weaviate pool metrics
type WeaviatePoolMetricsRecorder interface {
	RecordWeaviatePoolConnectionCreated()
	RecordWeaviatePoolConnectionDestroyed()
	UpdateWeaviatePoolActiveConnections(count int)
	UpdateWeaviatePoolSize(size int)
	RecordWeaviatePoolHealthCheckPassed()
	RecordWeaviatePoolHealthCheckFailed()
}

// NoOpMetricsRecorder is a no-op implementation for testing or when metrics are disabled
type NoOpMetricsRecorder struct{}

func (n *NoOpMetricsRecorder) RecordWeaviatePoolConnectionCreated()          {}
func (n *NoOpMetricsRecorder) RecordWeaviatePoolConnectionDestroyed()        {}
func (n *NoOpMetricsRecorder) UpdateWeaviatePoolActiveConnections(count int) {}
func (n *NoOpMetricsRecorder) UpdateWeaviatePoolSize(size int)               {}
func (n *NoOpMetricsRecorder) RecordWeaviatePoolHealthCheckPassed()          {}
func (n *NoOpMetricsRecorder) RecordWeaviatePoolHealthCheckFailed()          {}

// WeaviateConnectionPool manages a pool of Weaviate client connections
type WeaviateConnectionPool struct {
	config        *PoolConfig
	connections   chan *PooledConnection
	activeConns   map[string]*PooledConnection
	metrics       *PoolMetrics
	mu            sync.RWMutex
	started       bool
	ctx           context.Context
	cancel        context.CancelFunc
	cleanupTicker *time.Ticker

	// Metrics recorder interface
	metricsRecorder WeaviatePoolMetricsRecorder
}

// PooledConnection represents a connection in the pool
type PooledConnection struct {
	client     *weaviate.Client
	id         string
	createdAt  time.Time
	lastUsed   time.Time
	usageCount int64
	isHealthy  bool
	inUse      bool
	mu         sync.RWMutex
}

// PoolConfig configures the connection pool
type PoolConfig struct {
	// Connection settings
	URL     string
	APIKey  string
	Scheme  string
	Host    string
	Headers map[string]string

	// Pool settings
	MinConnections      int           // Minimum pool size
	MaxConnections      int           // Maximum pool size
	MaxIdleTime         time.Duration // Max idle time before connection cleanup
	MaxLifetime         time.Duration // Max connection lifetime
	HealthCheckInterval time.Duration // Health check frequency
	ConnectionTimeout   time.Duration // Connection establishment timeout
	RequestTimeout      time.Duration // Individual request timeout

	// Performance settings
	EnableMetrics     bool
	EnableHealthCheck bool
	RetryAttempts     int
	RetryDelay        time.Duration
}

// PoolMetrics tracks pool performance
type PoolMetrics struct {
	TotalConnections     int64
	ActiveConnections    int64
	IdleConnections      int64
	ConnectionsCreated   int64
	ConnectionsDestroyed int64
	ConnectionRequests   int64
	ConnectionFailures   int64
	HealthCheckPassed    int64
	HealthCheckFailed    int64
	AverageLatency       time.Duration
	TotalLatency         time.Duration
	mu                   sync.RWMutex
}

// DefaultPoolConfig returns sensible defaults for production use
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		URL:                 "http://localhost:8080",
		Scheme:              "http",
		Host:                "localhost:8080",
		MinConnections:      2,
		MaxConnections:      10,
		MaxIdleTime:         15 * time.Minute,
		MaxLifetime:         1 * time.Hour,
		HealthCheckInterval: 30 * time.Second,
		ConnectionTimeout:   10 * time.Second,
		RequestTimeout:      30 * time.Second,
		EnableMetrics:       true,
		EnableHealthCheck:   true,
		RetryAttempts:       3,
		RetryDelay:          1 * time.Second,
	}
}

// NewWeaviateConnectionPool creates a new connection pool
func NewWeaviateConnectionPool(config *PoolConfig) *WeaviateConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &WeaviateConnectionPool{
		config:      config,
		connections: make(chan *PooledConnection, config.MaxConnections),
		activeConns: make(map[string]*PooledConnection),
		metrics:     &PoolMetrics{},
		ctx:         ctx,
		cancel:      cancel,
	}

	return pool
}

// NewWeaviateConnectionPoolWithMetrics creates a new connection pool with metrics recorder
func NewWeaviateConnectionPoolWithMetrics(config *PoolConfig, metricsRecorder WeaviatePoolMetricsRecorder) *WeaviateConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &WeaviateConnectionPool{
		config:          config,
		connections:     make(chan *PooledConnection, config.MaxConnections),
		activeConns:     make(map[string]*PooledConnection),
		metrics:         &PoolMetrics{},
		ctx:             ctx,
		cancel:          cancel,
		metricsRecorder: metricsRecorder,
	}

	return pool
}

// SetMetricsRecorder sets the metrics recorder after pool creation
func (p *WeaviateConnectionPool) SetMetricsRecorder(recorder WeaviatePoolMetricsRecorder) {
	p.metricsRecorder = recorder
}

// updatePoolSizeMetrics updates the pool size metrics through the metrics recorder
func (p *WeaviateConnectionPool) updatePoolSizeMetrics() {
	if p.metricsRecorder == nil {
		return // Metrics recorder not available
	}

	p.mu.RLock()
	activeCount := len(p.activeConns)
	idleCount := len(p.connections)
	p.mu.RUnlock()

	totalConnections := activeCount + idleCount

	p.metricsRecorder.UpdateWeaviatePoolActiveConnections(activeCount)
	p.metricsRecorder.UpdateWeaviatePoolSize(totalConnections)
}

// Start initializes the connection pool
func (p *WeaviateConnectionPool) Start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return fmt.Errorf("connection pool already started")
	}

	// Create minimum connections
	for i := 0; i < p.config.MinConnections; i++ {
		conn, err := p.createConnection()
		if err != nil {
			return fmt.Errorf("failed to create initial connection %d: %w", i, err)
		}

		p.connections <- conn
		p.activeConns[conn.id] = conn
	}

	// Update initial metrics
	p.updatePoolSizeMetrics()

	// Start background tasks
	if p.config.EnableHealthCheck {
		p.cleanupTicker = time.NewTicker(p.config.HealthCheckInterval)
		go p.healthCheckLoop()
	}

	p.started = true
	return nil
}

// Stop gracefully shuts down the pool
func (p *WeaviateConnectionPool) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return fmt.Errorf("connection pool not started")
	}

	// Stop background tasks
	p.cancel()
	if p.cleanupTicker != nil {
		p.cleanupTicker.Stop()
	}

	// Close all connections
	close(p.connections)
	for conn := range p.connections {
		p.destroyConnection(conn)
	}

	// Clean up active connections
	for _, conn := range p.activeConns {
		p.destroyConnection(conn)
	}
	p.activeConns = make(map[string]*PooledConnection)

	p.started = false
	return nil
}

// GetConnection retrieves a connection from the pool
func (p *WeaviateConnectionPool) GetConnection(ctx context.Context) (*PooledConnection, error) {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return nil, fmt.Errorf("connection pool not started")
	}
	p.mu.Unlock()

	// Update metrics
	atomic.AddInt64(&p.metrics.ConnectionRequests, 1)

	// Try to get existing connection
	select {
	case conn := <-p.connections:
		if p.isConnectionHealthy(conn) {
			conn.mu.Lock()
			conn.lastUsed = time.Now()
			conn.usageCount++
			conn.inUse = true
			conn.mu.Unlock()
			return conn, nil
		} else {
			// Connection unhealthy, destroy and create new one
			p.destroyConnection(conn)
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// No available connections, try to create new one
	}

	// Create new connection if under limit
	p.mu.Lock()
	currentConns := len(p.activeConns)
	if currentConns < p.config.MaxConnections {
		p.mu.Unlock()
		conn, err := p.createConnection()
		if err != nil {
			atomic.AddInt64(&p.metrics.ConnectionFailures, 1)
			return nil, fmt.Errorf("failed to create connection: %w", err)
		}

		p.mu.Lock()
		p.activeConns[conn.id] = conn
		p.mu.Unlock()

		// Update pool size metrics after adding connection
		p.updatePoolSizeMetrics()

		conn.mu.Lock()
		conn.inUse = true
		conn.mu.Unlock()

		return conn, nil
	}
	p.mu.Unlock()

	// Wait for available connection with timeout
	timeout := time.NewTimer(p.config.ConnectionTimeout)
	defer timeout.Stop()

	select {
	case conn := <-p.connections:
		if p.isConnectionHealthy(conn) {
			conn.mu.Lock()
			conn.lastUsed = time.Now()
			conn.usageCount++
			conn.inUse = true
			conn.mu.Unlock()
			return conn, nil
		} else {
			p.destroyConnection(conn)
			return nil, fmt.Errorf("no healthy connections available")
		}
	case <-timeout.C:
		return nil, fmt.Errorf("timeout waiting for connection")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ReturnConnection returns a connection to the pool
func (p *WeaviateConnectionPool) ReturnConnection(conn *PooledConnection) error {
	if conn == nil {
		return fmt.Errorf("cannot return nil connection")
	}

	conn.mu.Lock()
	conn.inUse = false
	conn.lastUsed = time.Now()
	conn.mu.Unlock()

	// Check if connection should be destroyed
	if !p.isConnectionHealthy(conn) || p.shouldDestroyConnection(conn) {
		p.mu.Lock()
		delete(p.activeConns, conn.id)
		p.mu.Unlock()
		p.destroyConnection(conn)
		p.updatePoolSizeMetrics()
		return nil
	}

	// Return to pool
	select {
	case p.connections <- conn:
		return nil
	default:
		// Pool is full, destroy connection
		p.mu.Lock()
		delete(p.activeConns, conn.id)
		p.mu.Unlock()
		p.destroyConnection(conn)
		p.updatePoolSizeMetrics()
		return nil
	}
}

// WithConnection executes a function with a pooled connection
func (p *WeaviateConnectionPool) WithConnection(ctx context.Context, fn func(*weaviate.Client) error) error {
	conn, err := p.GetConnection(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if returnErr := p.ReturnConnection(conn); returnErr != nil {
			// Log error but don't override the original error
		}
	}()

	startTime := time.Now()
	err = fn(conn.client)
	duration := time.Since(startTime)

	// Update metrics
	p.metrics.mu.Lock()
	p.metrics.TotalLatency += duration
	if p.metrics.ConnectionRequests > 0 {
		p.metrics.AverageLatency = time.Duration(int64(p.metrics.TotalLatency) / p.metrics.ConnectionRequests)
	}
	p.metrics.mu.Unlock()

	return err
}

// ExecuteWithRetry executes a function with retry logic
func (p *WeaviateConnectionPool) ExecuteWithRetry(ctx context.Context, fn func(*weaviate.Client) error) error {
	var lastErr error

	for attempt := 0; attempt < p.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(p.config.RetryDelay * time.Duration(attempt)):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := p.WithConnection(ctx, fn)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !p.isRetryableError(err) {
			break
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", p.config.RetryAttempts, lastErr)
}

// GetMetrics returns current pool metrics
func (p *WeaviateConnectionPool) GetMetrics() PoolMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	p.mu.RLock()
	activeCount := int64(len(p.activeConns))
	idleCount := int64(len(p.connections))
	p.mu.RUnlock()

	metrics := *p.metrics
	metrics.ActiveConnections = activeCount
	metrics.IdleConnections = idleCount
	metrics.TotalConnections = activeCount + idleCount

	return metrics
}

// GetConnectionInfo returns information about all connections
func (p *WeaviateConnectionPool) GetConnectionInfo() []ConnectionInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var info []ConnectionInfo
	for _, conn := range p.activeConns {
		conn.mu.RLock()
		info = append(info, ConnectionInfo{
			ID:         conn.id,
			CreatedAt:  conn.createdAt,
			LastUsed:   conn.lastUsed,
			UsageCount: conn.usageCount,
			IsHealthy:  conn.isHealthy,
			InUse:      conn.inUse,
		})
		conn.mu.RUnlock()
	}

	return info
}

// ConnectionInfo provides details about a connection
type ConnectionInfo struct {
	ID         string
	CreatedAt  time.Time
	LastUsed   time.Time
	UsageCount int64
	IsHealthy  bool
	InUse      bool
}

// Private methods

func (p *WeaviateConnectionPool) createConnection() (*PooledConnection, error) {
	cfg := weaviate.Config{
		Host:   p.config.Host,
		Scheme: p.config.Scheme,
	}

	// Add authentication if API key is provided
	if p.config.APIKey != "" {
		cfg.AuthConfig = auth.ApiKey{Value: p.config.APIKey}
	}

	// Add custom headers
	if len(p.config.Headers) > 0 {
		cfg.Headers = p.config.Headers
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Weaviate client: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), p.config.ConnectionTimeout)
	defer cancel()

	_, err = client.Misc().ReadyChecker().Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("connection health check failed: %w", err)
	}

	conn := &PooledConnection{
		client:    client,
		id:        fmt.Sprintf("conn-%d", time.Now().UnixNano()),
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		isHealthy: true,
	}

	atomic.AddInt64(&p.metrics.ConnectionsCreated, 1)

	// Update Prometheus metrics through recorder
	if p.metricsRecorder != nil {
		p.metricsRecorder.RecordWeaviatePoolConnectionCreated()
	}

	return conn, nil
}

func (p *WeaviateConnectionPool) destroyConnection(conn *PooledConnection) {
	if conn == nil {
		return
	}

	// Weaviate Go client doesn't have explicit close method
	// Connection will be closed by garbage collector

	atomic.AddInt64(&p.metrics.ConnectionsDestroyed, 1)

	// Update Prometheus metrics through recorder
	if p.metricsRecorder != nil {
		p.metricsRecorder.RecordWeaviatePoolConnectionDestroyed()
	}
}

func (p *WeaviateConnectionPool) isConnectionHealthy(conn *PooledConnection) bool {
	if conn == nil {
		return false
	}

	conn.mu.RLock()
	isHealthy := conn.isHealthy
	conn.mu.RUnlock()

	return isHealthy
}

func (p *WeaviateConnectionPool) shouldDestroyConnection(conn *PooledConnection) bool {
	if conn == nil {
		return true
	}

	conn.mu.RLock()
	defer conn.mu.RUnlock()

	now := time.Now()

	// Check max lifetime
	if p.config.MaxLifetime > 0 && now.Sub(conn.createdAt) > p.config.MaxLifetime {
		return true
	}

	// Check max idle time
	if p.config.MaxIdleTime > 0 && now.Sub(conn.lastUsed) > p.config.MaxIdleTime {
		return true
	}

	return false
}

func (p *WeaviateConnectionPool) healthCheckLoop() {
	for {
		select {
		case <-p.cleanupTicker.C:
			p.performHealthCheck()
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *WeaviateConnectionPool) performHealthCheck() {
	p.mu.RLock()
	connections := make([]*PooledConnection, 0, len(p.activeConns))
	for _, conn := range p.activeConns {
		connections = append(connections, conn)
	}
	p.mu.RUnlock()

	for _, conn := range connections {
		conn.mu.RLock()
		inUse := conn.inUse
		conn.mu.RUnlock()

		if inUse {
			continue // Skip connections in use
		}

		// Perform health check
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := conn.client.Misc().ReadyChecker().Do(ctx)
		cancel()

		conn.mu.Lock()
		conn.isHealthy = (err == nil)
		conn.mu.Unlock()

		if err != nil {
			atomic.AddInt64(&p.metrics.HealthCheckFailed, 1)
			// Update Prometheus metrics through recorder
			if p.metricsRecorder != nil {
				p.metricsRecorder.RecordWeaviatePoolHealthCheckFailed()
			}
			// Remove unhealthy connection
			p.mu.Lock()
			delete(p.activeConns, conn.id)
			p.mu.Unlock()
			p.destroyConnection(conn)
		} else {
			atomic.AddInt64(&p.metrics.HealthCheckPassed, 1)
			// Update Prometheus metrics through recorder
			if p.metricsRecorder != nil {
				p.metricsRecorder.RecordWeaviatePoolHealthCheckPassed()
			}
		}

		// Check if connection should be destroyed due to age/idle time
		if p.shouldDestroyConnection(conn) {
			p.mu.Lock()
			delete(p.activeConns, conn.id)
			p.mu.Unlock()
			p.destroyConnection(conn)
		}
	}

	// Ensure minimum connections
	p.ensureMinimumConnections()

	// Update pool metrics after all health check operations
	p.updatePoolSizeMetrics()
}

func (p *WeaviateConnectionPool) ensureMinimumConnections() {
	p.mu.Lock()
	currentCount := len(p.activeConns)
	needed := p.config.MinConnections - currentCount
	p.mu.Unlock()

	for i := 0; i < needed; i++ {
		conn, err := p.createConnection()
		if err != nil {
			// Log error but continue
			continue
		}

		p.mu.Lock()
		p.activeConns[conn.id] = conn
		p.mu.Unlock()

		// Try to add to pool
		select {
		case p.connections <- conn:
		default:
			// Pool is full, but connection is tracked in activeConns
		}
	}
}

func (p *WeaviateConnectionPool) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Add logic to determine if error is retryable
	// For now, assume most errors are retryable except context cancellation
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	return true
}
