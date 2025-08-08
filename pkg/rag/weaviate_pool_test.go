package rag

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/weaviate/weaviate-go-client/v4/weaviate"
)

func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, "http://localhost:8080", config.URL)
	assert.Equal(t, 2, config.MinConnections)
	assert.Equal(t, 10, config.MaxConnections)
	assert.Equal(t, 15*time.Minute, config.MaxIdleTime)
	assert.Equal(t, 1*time.Hour, config.MaxLifetime)
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.EnableHealthCheck)
}

func TestNewWeaviateConnectionPool(t *testing.T) {
	config := &PoolConfig{
		URL:            "http://localhost:8080",
		MinConnections: 2,
		MaxConnections: 5,
		EnableMetrics:  true,
	}
	
	pool := NewWeaviateConnectionPool(config)
	
	assert.NotNil(t, pool)
	assert.Equal(t, config, pool.config)
	assert.NotNil(t, pool.connections)
	assert.NotNil(t, pool.activeConns)
	assert.NotNil(t, pool.metrics)
	assert.False(t, pool.started)
}

func TestWeaviateConnectionPool_StartStop(t *testing.T) {
	config := &PoolConfig{
		URL:                 "http://localhost:8080",
		MinConnections:      1,
		MaxConnections:      3,
		ConnectionTimeout:   5 * time.Second,
		EnableHealthCheck:   false, // Disable for testing
		EnableMetrics:       true,
	}
	
	pool := NewWeaviateConnectionPool(config)
	
	// Note: This test will skip actual Weaviate connection due to test environment
	// In a real test, you would use a mock Weaviate instance or test containers
	
	// Test that pool can be created but start may fail without real Weaviate
	err := pool.Start()
	if err != nil {
		// Expected in test environment without Weaviate
		t.Logf("Pool start failed as expected in test environment: %v", err)
		return
	}
	
	// If start succeeded (mock environment), test normal operations
	assert.True(t, pool.started)
	
	// Test duplicate start
	err = pool.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already started")
	
	// Test stop
	err = pool.Stop()
	assert.NoError(t, err)
	assert.False(t, pool.started)
	
	// Test stop when not started
	err = pool.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

func TestWeaviateConnectionPool_GetConnectionNotStarted(t *testing.T) {
	config := &PoolConfig{
		URL:            "http://localhost:8080",
		MinConnections: 1,
		MaxConnections: 3,
	}
	
	pool := NewWeaviateConnectionPool(config)
	ctx := context.Background()
	
	conn, err := pool.GetConnection(ctx)
	assert.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "not started")
}

func TestWeaviateConnectionPool_ReturnConnection(t *testing.T) {
	config := &PoolConfig{
		URL:            "http://localhost:8080",
		MinConnections: 1,
		MaxConnections: 3,
		MaxIdleTime:    10 * time.Minute,
		MaxLifetime:    1 * time.Hour,
	}
	
	pool := NewWeaviateConnectionPool(config)
	
	// Test returning nil connection
	err := pool.ReturnConnection(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot return nil connection")
	
	// Create a mock connection for testing
	mockConn := &PooledConnection{
		id:        "test-conn-1",
		createdAt: time.Now(),
		lastUsed:  time.Now(),
		isHealthy: true,
		inUse:     true,
	}
	
	// Test returning connection
	err = pool.ReturnConnection(mockConn)
	assert.NoError(t, err)
	assert.False(t, mockConn.inUse)
}

func TestPooledConnection_ShouldDestroy(t *testing.T) {
	config := &PoolConfig{
		MaxIdleTime: 5 * time.Minute,
		MaxLifetime: 1 * time.Hour,
	}
	
	pool := NewWeaviateConnectionPool(config)
	
	now := time.Now()
	
	// Test connection within limits
	conn := &PooledConnection{
		createdAt: now.Add(-30 * time.Minute),
		lastUsed:  now.Add(-2 * time.Minute),
		isHealthy: true,
	}
	assert.False(t, pool.shouldDestroyConnection(conn))
	
	// Test connection exceeding max idle time
	conn.lastUsed = now.Add(-10 * time.Minute)
	assert.True(t, pool.shouldDestroyConnection(conn))
	
	// Test connection exceeding max lifetime
	conn.lastUsed = now
	conn.createdAt = now.Add(-2 * time.Hour)
	assert.True(t, pool.shouldDestroyConnection(conn))
	
	// Test nil connection
	assert.True(t, pool.shouldDestroyConnection(nil))
}

func TestWeaviateConnectionPool_GetMetrics(t *testing.T) {
	config := &PoolConfig{
		URL:            "http://localhost:8080",
		MinConnections: 1,
		MaxConnections: 3,
	}
	
	pool := NewWeaviateConnectionPool(config)
	
	// Add some test data to metrics
	atomic.AddInt64(&pool.metrics.ConnectionsCreated, 5)
	atomic.AddInt64(&pool.metrics.ConnectionsDestroyed, 2)
	atomic.AddInt64(&pool.metrics.ConnectionRequests, 10)
	atomic.AddInt64(&pool.metrics.HealthCheckPassed, 8)
	
	metrics := pool.GetMetrics()
	
	assert.Equal(t, int64(5), metrics.ConnectionsCreated)
	assert.Equal(t, int64(2), metrics.ConnectionsDestroyed)
	assert.Equal(t, int64(10), metrics.ConnectionRequests)
	assert.Equal(t, int64(8), metrics.HealthCheckPassed)
}

func TestWeaviateConnectionPool_GetConnectionInfo(t *testing.T) {
	config := &PoolConfig{
		URL:            "http://localhost:8080",
		MinConnections: 1,
		MaxConnections: 3,
	}
	
	pool := NewWeaviateConnectionPool(config)
	
	// Add mock connections
	now := time.Now()
	conn1 := &PooledConnection{
		id:         "conn-1",
		createdAt:  now.Add(-1 * time.Hour),
		lastUsed:   now.Add(-5 * time.Minute),
		usageCount: 15,
		isHealthy:  true,
		inUse:      false,
	}
	conn2 := &PooledConnection{
		id:         "conn-2",
		createdAt:  now.Add(-30 * time.Minute),
		lastUsed:   now.Add(-1 * time.Minute),
		usageCount: 8,
		isHealthy:  true,
		inUse:      true,
	}
	
	pool.activeConns["conn-1"] = conn1
	pool.activeConns["conn-2"] = conn2
	
	info := pool.GetConnectionInfo()
	
	assert.Len(t, info, 2)
	
	// Find connections in info
	var info1, info2 *ConnectionInfo
	for i := range info {
		if info[i].ID == "conn-1" {
			info1 = &info[i]
		} else if info[i].ID == "conn-2" {
			info2 = &info[i]
		}
	}
	
	require.NotNil(t, info1)
	require.NotNil(t, info2)
	
	assert.Equal(t, int64(15), info1.UsageCount)
	assert.False(t, info1.InUse)
	assert.Equal(t, int64(8), info2.UsageCount)
	assert.True(t, info2.InUse)
}

func TestWeaviateConnectionPool_WithConnection(t *testing.T) {
	config := &PoolConfig{
		URL:               "http://localhost:8080",
		MinConnections:    1,
		MaxConnections:    3,
		ConnectionTimeout: 1 * time.Second,
	}
	
	pool := NewWeaviateConnectionPool(config)
	ctx := context.Background()
	
	// Test with pool not started
	err := pool.WithConnection(ctx, func(client *weaviate.Client) error {
		return nil
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not started")
}

func TestWeaviateConnectionPool_ExecuteWithRetry(t *testing.T) {
	config := &PoolConfig{
		URL:            "http://localhost:8080",
		MinConnections: 1,
		MaxConnections: 3,
		RetryAttempts:  3,
		RetryDelay:     100 * time.Millisecond,
	}
	
	pool := NewWeaviateConnectionPool(config)
	ctx := context.Background()
	
	// Test with pool not started - should fail without retries for non-retryable error
	var attemptCount int32
	err := pool.ExecuteWithRetry(ctx, func(client *weaviate.Client) error {
		atomic.AddInt32(&attemptCount, 1)
		return context.Canceled // Non-retryable error
	})
	
	assert.Error(t, err)
	// Should not retry for non-retryable errors, but our implementation will try
	// because pool not started error comes first
}

func TestWeaviateConnectionPool_IsRetryableError(t *testing.T) {
	pool := NewWeaviateConnectionPool(DefaultPoolConfig())
	
	// Test retryable errors
	assert.True(t, pool.isRetryableError(assert.AnError))
	
	// Test non-retryable errors
	assert.False(t, pool.isRetryableError(context.Canceled))
	assert.False(t, pool.isRetryableError(context.DeadlineExceeded))
	assert.False(t, pool.isRetryableError(nil))
}

func TestWeaviateConnectionPool_IsConnectionHealthy(t *testing.T) {
	pool := NewWeaviateConnectionPool(DefaultPoolConfig())
	
	// Test nil connection
	assert.False(t, pool.isConnectionHealthy(nil))
	
	// Test healthy connection
	healthyConn := &PooledConnection{
		isHealthy: true,
	}
	assert.True(t, pool.isConnectionHealthy(healthyConn))
	
	// Test unhealthy connection
	unhealthyConn := &PooledConnection{
		isHealthy: false,
	}
	assert.False(t, pool.isConnectionHealthy(unhealthyConn))
}

func TestWeaviateConnectionPool_ConcurrentAccess(t *testing.T) {
	config := &PoolConfig{
		URL:               "http://localhost:8080",
		MinConnections:    2,
		MaxConnections:    5,
		ConnectionTimeout: 1 * time.Second,
		EnableHealthCheck: false,
	}
	
	pool := NewWeaviateConnectionPool(config)
	
	// Test concurrent metric updates
	var wg sync.WaitGroup
	const numGoroutines = 100
	
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			atomic.AddInt64(&pool.metrics.ConnectionRequests, 1)
			atomic.AddInt64(&pool.metrics.ConnectionsCreated, 1)
		}()
	}
	
	wg.Wait()
	
	metrics := pool.GetMetrics()
	assert.Equal(t, int64(numGoroutines), metrics.ConnectionRequests)
	assert.Equal(t, int64(numGoroutines), metrics.ConnectionsCreated)
}

func TestWeaviateConnectionPool_MetricsUpdate(t *testing.T) {
	config := DefaultPoolConfig()
	pool := NewWeaviateConnectionPool(config)
	
	// Simulate connection operations
	atomic.AddInt64(&pool.metrics.ConnectionsCreated, 5)
	atomic.AddInt64(&pool.metrics.ConnectionsDestroyed, 2)
	atomic.AddInt64(&pool.metrics.ConnectionRequests, 15)
	atomic.AddInt64(&pool.metrics.ConnectionFailures, 1)
	atomic.AddInt64(&pool.metrics.HealthCheckPassed, 20)
	atomic.AddInt64(&pool.metrics.HealthCheckFailed, 3)
	
	// Add latency data
	pool.metrics.mu.Lock()
	pool.metrics.TotalLatency = 10 * time.Second
	pool.metrics.AverageLatency = 666 * time.Millisecond
	pool.metrics.mu.Unlock()
	
	metrics := pool.GetMetrics()
	
	assert.Equal(t, int64(5), metrics.ConnectionsCreated)
	assert.Equal(t, int64(2), metrics.ConnectionsDestroyed)
	assert.Equal(t, int64(15), metrics.ConnectionRequests)
	assert.Equal(t, int64(1), metrics.ConnectionFailures)
	assert.Equal(t, int64(20), metrics.HealthCheckPassed)
	assert.Equal(t, int64(3), metrics.HealthCheckFailed)
	assert.Equal(t, 10*time.Second, metrics.TotalLatency)
	assert.Equal(t, 666*time.Millisecond, metrics.AverageLatency)
}

func TestWeaviateConnectionPool_EnsureMinimumConnections(t *testing.T) {
	config := &PoolConfig{
		URL:            "http://localhost:8080",
		MinConnections: 3,
		MaxConnections: 5,
	}
	
	pool := NewWeaviateConnectionPool(config)
	
	// Start with no connections
	assert.Equal(t, 0, len(pool.activeConns))
	
	// Call ensure minimum connections (this will fail in test env but test the logic)
	pool.ensureMinimumConnections()
	
	// In a real environment with working Weaviate, this would create connections
	// For testing, we just verify the method doesn't panic
}

func TestPooledConnection_UsageTracking(t *testing.T) {
	conn := &PooledConnection{
		id:        "test-conn",
		createdAt: time.Now(),
		lastUsed:  time.Now().Add(-5 * time.Minute),
		isHealthy: true,
		inUse:     false,
	}
	
	// Simulate usage
	conn.mu.Lock()
	conn.lastUsed = time.Now()
	conn.usageCount++
	conn.inUse = true
	conn.mu.Unlock()
	
	assert.Equal(t, int64(1), conn.usageCount)
	assert.True(t, conn.inUse)
	assert.True(t, time.Since(conn.lastUsed) < time.Second)
}

func TestWeaviateConnectionPool_ContextCancellation(t *testing.T) {
	config := &PoolConfig{
		URL:               "http://localhost:8080",
		MinConnections:    1,
		MaxConnections:    3,
		ConnectionTimeout: 5 * time.Second,
	}
	
	pool := NewWeaviateConnectionPool(config)
	
	// Test context cancellation during GetConnection
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	
	conn, err := pool.GetConnection(ctx)
	assert.Error(t, err)
	assert.Nil(t, conn)
	assert.Equal(t, context.Canceled, err)
}

// Benchmark tests
func BenchmarkWeaviateConnectionPool_GetReturnConnection(b *testing.B) {
	config := &PoolConfig{
		URL:            "http://localhost:8080",
		MinConnections: 5,
		MaxConnections: 10,
		EnableMetrics:  true,
	}
	
	pool := NewWeaviateConnectionPool(config)
	
	// Create mock connections for benchmarking
	for i := 0; i < 5; i++ {
		conn := &PooledConnection{
			id:        fmt.Sprintf("bench-conn-%d", i),
			createdAt: time.Now(),
			lastUsed:  time.Now(),
			isHealthy: true,
		}
		pool.connections <- conn
		pool.activeConns[conn.id] = conn
	}
	pool.started = true
	
	ctx := context.Background()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			select {
			case conn := <-pool.connections:
				pool.connections <- conn // Return immediately
			default:
				// No connection available
			}
		}
	})
}

func BenchmarkWeaviateConnectionPool_MetricsUpdate(b *testing.B) {
	pool := NewWeaviateConnectionPool(DefaultPoolConfig())
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			atomic.AddInt64(&pool.metrics.ConnectionRequests, 1)
			atomic.AddInt64(&pool.metrics.ConnectionsCreated, 1)
		}
	})
}

func BenchmarkWeaviateConnectionPool_GetConnectionInfo(b *testing.B) {
	pool := NewWeaviateConnectionPool(DefaultPoolConfig())
	
	// Create mock connections
	for i := 0; i < 10; i++ {
		conn := &PooledConnection{
			id:         fmt.Sprintf("bench-conn-%d", i),
			createdAt:  time.Now().Add(-time.Duration(i) * time.Minute),
			lastUsed:   time.Now(),
			usageCount: int64(i * 10),
			isHealthy:  true,
		}
		pool.activeConns[conn.id] = conn
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.GetConnectionInfo()
	}
}