//go:build !disable_rag

package rag

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"
)

// ConnectionPool manages database connections for RAG operations
type ConnectionPool struct {
	pool        map[string]*sql.DB
	maxIdle     int
	maxOpen     int
	maxLifetime time.Duration
	mu          sync.RWMutex
	stats       ConnectionPoolStats
}

// ConnectionPoolStats tracks connection pool statistics
type ConnectionPoolStats struct {
	TotalConnections int           `json:"total_connections"`
	IdleConnections  int           `json:"idle_connections"`
	ActiveQueries    int           `json:"active_queries"`
	AverageLatency   time.Duration `json:"average_latency"`
	ErrorCount       int64         `json:"error_count"`
	LastUsed         time.Time     `json:"last_used"`
}

// ConnectionConfig defines connection configuration
type ConnectionConfig struct {
	DSN             string        `json:"dsn"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	MaxOpenConns    int           `json:"max_open_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	Timeout         time.Duration `json:"timeout"`
	RetryAttempts   int           `json:"retry_attempts"`
	RetryDelay      time.Duration `json:"retry_delay"`
}

// ConnectionManager manages multiple connection pools
type ConnectionManager struct {
	pools   map[string]*ConnectionPool
	config  *ConnectionManagerConfig
	mu      sync.RWMutex
	metrics *ConnectionMetrics
}

// ConnectionManagerConfig defines connection manager configuration
type ConnectionManagerConfig struct {
	MaxPools        int           `json:"max_pools"`
	HealthInterval  time.Duration `json:"health_interval"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	Enabled         bool          `json:"enabled"`
}

// ConnectionMetrics tracks connection metrics
type ConnectionMetrics struct {
	totalQueries  int64
	failedQueries int64
	totalLatency  time.Duration
	activeConns   int64
	idleConns     int64
	mu            sync.RWMutex
}

// RAGConnection represents a connection to a RAG system
type RAGConnection struct {
	id       string
	provider string
	config   *ConnectionConfig
	pool     *ConnectionPool
	health   *ConnectionHealth
	mu       sync.RWMutex
}

// ConnectionHealth represents connection health status
type ConnectionHealth struct {
	IsHealthy      bool          `json:"is_healthy"`
	LastCheck      time.Time     `json:"last_check"`
	ResponseTime   time.Duration `json:"response_time"`
	ErrorRate      float64       `json:"error_rate"`
	ConsecutiveErr int           `json:"consecutive_errors"`
	Details        string        `json:"details"`
}

// Error definitions for connection management
var (
	ErrConnectionNotFound = errors.New("connection not found")
	ErrPoolNotFound       = errors.New("pool not found")
	ErrConnectionFailed   = errors.New("connection failed")
	ErrPoolClosed         = errors.New("pool is closed")
	ErrInvalidConfig      = errors.New("invalid configuration")
)

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *ConnectionConfig) (*ConnectionPool, error) {
	return &ConnectionPool{
		pool:        make(map[string]*sql.DB),
		maxIdle:     config.MaxIdleConns,
		maxOpen:     config.MaxOpenConns,
		maxLifetime: config.ConnMaxLifetime,
	}, nil
}

// GetConnection retrieves a connection from the pool
func (cp *ConnectionPool) GetConnection(ctx context.Context, name string) (*sql.DB, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if conn, exists := cp.pool[name]; exists {
		return conn, nil
	}

	return nil, ErrConnectionNotFound
}

// AddConnection adds a new connection to the pool
func (cp *ConnectionPool) AddConnection(name string, db *sql.DB) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.pool[name] = db
	return nil
}

// RemoveConnection removes a connection from the pool
func (cp *ConnectionPool) RemoveConnection(name string) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if conn, exists := cp.pool[name]; exists {
		conn.Close()
		delete(cp.pool, name)
	}

	return nil
}

// GetStats returns connection pool statistics
func (cp *ConnectionPool) GetStats() ConnectionPoolStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return cp.stats
}

// Close closes all connections in the pool
func (cp *ConnectionPool) Close() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	for _, conn := range cp.pool {
		conn.Close()
	}

	cp.pool = make(map[string]*sql.DB)
	return nil
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(config *ConnectionManagerConfig) *ConnectionManager {
	return &ConnectionManager{
		pools:   make(map[string]*ConnectionPool),
		config:  config,
		metrics: &ConnectionMetrics{},
	}
}

// AddPool adds a connection pool to the manager
func (cm *ConnectionManager) AddPool(name string, pool *ConnectionPool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.pools[name] = pool
}

// GetPool retrieves a connection pool by name
func (cm *ConnectionManager) GetPool(name string) (*ConnectionPool, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if pool, exists := cm.pools[name]; exists {
		return pool, nil
	}

	return nil, ErrPoolNotFound
}

// RemovePool removes a connection pool from the manager
func (cm *ConnectionManager) RemovePool(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if pool, exists := cm.pools[name]; exists {
		pool.Close()
		delete(cm.pools, name)
	}

	return nil
}

// GetMetrics returns connection manager metrics
func (cm *ConnectionManager) GetMetrics() *ConnectionMetrics {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.metrics
}

// Shutdown gracefully shuts down the connection manager
func (cm *ConnectionManager) Shutdown(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for name, pool := range cm.pools {
		if err := pool.Close(); err != nil {
			return err
		}
		delete(cm.pools, name)
	}

	return nil
}

// NewRAGConnection creates a new RAG connection
func NewRAGConnection(id, provider string, config *ConnectionConfig) (*RAGConnection, error) {
	pool, err := NewConnectionPool(config)
	if err != nil {
		return nil, err
	}

	return &RAGConnection{
		id:       id,
		provider: provider,
		config:   config,
		pool:     pool,
		health: &ConnectionHealth{
			IsHealthy: true,
			LastCheck: time.Now(),
		},
	}, nil
}

// GetHealth returns the connection health status
func (rc *RAGConnection) GetHealth() *ConnectionHealth {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	return rc.health
}

// UpdateHealth updates the connection health status
func (rc *RAGConnection) UpdateHealth(healthy bool, details string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.health.IsHealthy = healthy
	rc.health.LastCheck = time.Now()
	rc.health.Details = details

	if !healthy {
		rc.health.ConsecutiveErr++
	} else {
		rc.health.ConsecutiveErr = 0
	}
}

// Close closes the RAG connection
func (rc *RAGConnection) Close() error {
	return rc.pool.Close()
}

// RecordQuery records query metrics
func (cm *ConnectionMetrics) RecordQuery(duration time.Duration, success bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.totalQueries++
	cm.totalLatency += duration

	if !success {
		cm.failedQueries++
	}
}

// GetStats returns connection metrics statistics
func (cm *ConnectionMetrics) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var avgLatency time.Duration
	if cm.totalQueries > 0 {
		avgLatency = cm.totalLatency / time.Duration(cm.totalQueries)
	}

	var errorRate float64
	if cm.totalQueries > 0 {
		errorRate = float64(cm.failedQueries) / float64(cm.totalQueries)
	}

	return map[string]interface{}{
		"total_queries":      cm.totalQueries,
		"failed_queries":     cm.failedQueries,
		"average_latency":    avgLatency,
		"error_rate":         errorRate,
		"active_connections": cm.activeConns,
		"idle_connections":   cm.idleConns,
	}
}
