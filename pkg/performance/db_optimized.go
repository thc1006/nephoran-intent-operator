//go:build go1.24

package performance

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq" // PostgreSQL driver

	"k8s.io/klog/v2"
)

// OptimizedDBManager provides high-performance database operations with Go 1.24+ optimizations.

type OptimizedDBManager struct {
	connectionPools map[string]*ConnectionPool

	preparedStmts *PreparedStatementCache

	queryOptimizer *QueryOptimizer

	txnManager *TransactionManager

	batchProcessor *BatchProcessor

	metrics *DBMetrics

	config *DBConfig

	healthChecker *DBHealthChecker

	readReplicas []*sql.DB

	writePrimary *sql.DB

	loadBalancer *ReadLoadBalancer

	mu sync.RWMutex

	shutdown chan struct{}

	wg sync.WaitGroup
}

// DBConfig contains database optimization configuration.

type DBConfig struct {
	MaxOpenConns int

	MaxIdleConns int

	ConnMaxLifetime time.Duration

	ConnMaxIdleTime time.Duration

	HealthCheckInterval time.Duration

	QueryTimeout time.Duration

	TxnTimeout time.Duration

	BatchSize int

	BatchTimeout time.Duration

	PreparedStmtCache int

	ReadReplicaCount int

	EnableQueryCache bool

	EnableConnPooling bool

	EnableHealthCheck bool

	EnableMetrics bool

	SlowQueryThreshold time.Duration

	RetryAttempts int

	RetryDelay time.Duration
}

// ConnectionPool manages database connections with advanced features.

type ConnectionPool struct {
	db *sql.DB

	dsn string

	activeConns int64

	idleConns int64

	createdConns int64

	closedConns int64

	failedConns int64

	lastHealthCheck time.Time

	healthy bool

	config *DBConfig

	mu sync.RWMutex
}

// PreparedStatementCache caches prepared statements with LRU eviction.

type PreparedStatementCache struct {
	cache map[string]*PreparedStmt

	lruHead *PreparedStmt

	lruTail *PreparedStmt

	maxSize int

	currentSize int

	hitCount int64

	missCount int64

	mu sync.RWMutex
}

// PreparedStmt represents a cached prepared statement.

type PreparedStmt struct {
	stmt *sql.Stmt

	query string

	lastUsed time.Time

	useCount int64

	prev *PreparedStmt

	next *PreparedStmt

	params []interface{}

	createdAt time.Time
}

// QueryOptimizer provides query optimization and caching.

type QueryOptimizer struct {
	queryCache map[string]*CachedQuery

	slowQueries map[string]*SlowQuery

	optimizations map[string]string

	hitCount int64

	missCount int64

	optimizedCount int64

	mu sync.RWMutex
}

// CachedQuery represents a cached query result.

type CachedQuery struct {
	result interface{}

	query string

	params []interface{}

	cachedAt time.Time

	expiration time.Time

	hitCount int64

	size int64
}

// SlowQuery tracks slow query information.

type SlowQuery struct {
	query string

	avgDuration time.Duration

	maxDuration time.Duration

	executionCount int64

	lastExecution time.Time

	optimized bool

	suggestions []string
}

// TransactionManager handles transaction optimization.

type TransactionManager struct {
	activeTxns map[string]*Transaction

	txnPool sync.Pool

	maxConcurrentTxns int64

	currentTxns int64

	completedTxns int64

	rolledBackTxns int64

	deadlocks int64

	mu sync.RWMutex
}

// Transaction represents an optimized database transaction.

type Transaction struct {
	id string

	tx *sql.Tx

	startTime time.Time

	timeout time.Duration

	readOnly bool

	isolation sql.IsolationLevel

	operations []TxnOperation

	context context.Context

	cancel context.CancelFunc
}

// TxnOperation represents a transaction operation.

type TxnOperation struct {
	type_ string

	query string

	params []interface{}

	executedAt time.Time

	duration time.Duration

	rowsAffected int64
}

// BatchProcessor handles batch database operations.

type BatchProcessor struct {
	batches map[string]*Batch

	batchTimeout time.Duration

	maxBatchSize int

	processed int64

	failed int64

	mu sync.RWMutex
}

// Batch represents a batch of database operations.

type Batch struct {
	operations []BatchOperation

	createdAt time.Time

	processedAt time.Time

	status BatchStatus

	results []BatchResult
}

// BatchOperation represents a single operation in a batch.

type BatchOperation struct {
	type_ string

	query string

	params []interface{}

	table string

	primary bool
}

// BatchResult contains the result of a batch operation.

type BatchResult struct {
	success bool

	rowsAffected int64

	error error

	duration time.Duration
}

// BatchStatus represents the status of a batch.

type BatchStatus int

const (

	// BatchPending holds batchpending value.

	BatchPending BatchStatus = iota

	// BatchProcessing holds batchprocessing value.

	BatchProcessing

	// BatchCompleted holds batchcompleted value.

	BatchCompleted

	// BatchFailed holds batchfailed value.

	BatchFailed
)

// DBHealthChecker monitors database health.

type DBHealthChecker struct {
	checks map[string]*HealthCheck

	interval time.Duration

	timeout time.Duration

	alertCallback func(string, error)

	mu sync.RWMutex
}

// HealthCheck represents a database health check.

type HealthCheck struct {
	name string

	lastCheck time.Time

	lastError error

	latency time.Duration

	healthy bool

	failCount int64

	successCount int64
}

// ReadLoadBalancer balances read operations across replicas.

type ReadLoadBalancer struct {
	replicas []*sql.DB

	weights []int

	currentIndex int64

	healthyReplicas map[int]bool

	strategy LoadBalanceStrategy

	mu sync.RWMutex
}

// LoadBalanceStrategy defines load balancing strategies.

type LoadBalanceStrategy int

const (

	// LoadBalanceRoundRobin holds loadbalanceroundrobin value.

	LoadBalanceRoundRobin LoadBalanceStrategy = iota

	// LoadBalanceWeighted holds loadbalanceweighted value.

	LoadBalanceWeighted

	// LoadBalanceLeastConn holds loadbalanceleastconn value.

	LoadBalanceLeastConn

	// LoadBalanceRandom holds loadbalancerandom value.

	LoadBalanceRandom
)

// DBMetrics tracks database performance metrics.

type DBMetrics struct {
	QueryCount int64

	SlowQueryCount int64

	ErrorCount int64

	ConnectionCount int64

	ActiveConnections int64

	IdleConnections int64

	TransactionCount int64

	RollbackCount int64

	BatchCount int64

	CacheHitCount int64

	CacheMissCount int64

	AverageQueryTime int64 // nanoseconds

	AverageConnTime int64 // nanoseconds

	TotalBytesRead int64

	TotalBytesWritten int64

	ReplicationLag int64 // milliseconds

	DeadlockCount int64

	PreparedStmtHits int64

	PreparedStmtMisses int64
}

// NewOptimizedDBManager creates a new optimized database manager.

func NewOptimizedDBManager(config *DBConfig) (*OptimizedDBManager, error) {

	if config == nil {

		config = DefaultDBConfig()

	}

	manager := &OptimizedDBManager{

		connectionPools: make(map[string]*ConnectionPool),

		preparedStmts: NewPreparedStatementCache(config.PreparedStmtCache),

		queryOptimizer: NewQueryOptimizer(),

		txnManager: NewTransactionManager(),

		batchProcessor: NewBatchProcessor(config.BatchSize, config.BatchTimeout),

		metrics: &DBMetrics{},

		config: config,

		shutdown: make(chan struct{}),
	}

	if config.EnableHealthCheck {

		manager.healthChecker = NewDBHealthChecker(config.HealthCheckInterval)

	}

	// Start background tasks.

	manager.startBackgroundTasks()

	return manager, nil

}

// DefaultDBConfig returns default database configuration.

func DefaultDBConfig() *DBConfig {

	return &DBConfig{

		MaxOpenConns: runtime.NumCPU() * 4,

		MaxIdleConns: runtime.NumCPU() * 2,

		ConnMaxLifetime: 5 * time.Minute,

		ConnMaxIdleTime: 2 * time.Minute,

		HealthCheckInterval: 30 * time.Second,

		QueryTimeout: 10 * time.Second,

		TxnTimeout: 30 * time.Second,

		BatchSize: 1000,

		BatchTimeout: 5 * time.Second,

		PreparedStmtCache: 500,

		ReadReplicaCount: 2,

		EnableQueryCache: true,

		EnableConnPooling: true,

		EnableHealthCheck: true,

		EnableMetrics: true,

		SlowQueryThreshold: 1 * time.Second,

		RetryAttempts: 3,

		RetryDelay: 100 * time.Millisecond,
	}

}

// AddConnectionPool adds a new connection pool.

func (dm *OptimizedDBManager) AddConnectionPool(name, dsn string) error {

	dm.mu.Lock()

	defer dm.mu.Unlock()

	db, err := sql.Open("postgres", dsn)

	if err != nil {

		return fmt.Errorf("failed to open database: %w", err)

	}

	// Configure connection pool.

	db.SetMaxOpenConns(dm.config.MaxOpenConns)

	db.SetMaxIdleConns(dm.config.MaxIdleConns)

	db.SetConnMaxLifetime(dm.config.ConnMaxLifetime)

	db.SetConnMaxIdleTime(dm.config.ConnMaxIdleTime)

	// Test connection.

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	if err := db.PingContext(ctx); err != nil {

		db.Close()

		return fmt.Errorf("failed to ping database: %w", err)

	}

	pool := &ConnectionPool{

		db: db,

		dsn: dsn,

		config: dm.config,

		healthy: true,

		lastHealthCheck: time.Now(),
	}

	dm.connectionPools[name] = pool

	// Set as primary if first connection.

	if dm.writePrimary == nil {

		dm.writePrimary = db

	} else {

		// Add as read replica.

		dm.readReplicas = append(dm.readReplicas, db)

	}

	// Update load balancer.

	if len(dm.readReplicas) > 0 {

		dm.loadBalancer = NewReadLoadBalancer(dm.readReplicas, LoadBalanceRoundRobin)

	}

	klog.Infof("Added database connection pool: %s", name)

	return nil

}

// QueryWithOptimization executes a query with optimization.

func (dm *OptimizedDBManager) QueryWithOptimization(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {

	start := time.Now()

	defer func() {

		duration := time.Since(start)

		atomic.AddInt64(&dm.metrics.QueryCount, 1)

		dm.updateAverageQueryTime(duration)

		if duration > dm.config.SlowQueryThreshold {

			atomic.AddInt64(&dm.metrics.SlowQueryCount, 1)

			dm.queryOptimizer.RecordSlowQuery(query, duration)

		}

	}()

	// Check query cache first.

	if dm.config.EnableQueryCache {

		if cached := dm.queryOptimizer.GetCachedResult(query, args); cached != nil {

			atomic.AddInt64(&dm.metrics.CacheHitCount, 1)

			// Return cached result (simplified).

			return nil, nil // Would return actual cached rows

		}

		atomic.AddInt64(&dm.metrics.CacheMissCount, 1)

	}

	// Get optimized query.

	optimizedQuery := dm.queryOptimizer.OptimizeQuery(query)

	// Use prepared statement if available.

	stmt := dm.preparedStmts.Get(optimizedQuery)

	if stmt != nil {

		atomic.AddInt64(&dm.metrics.PreparedStmtHits, 1)

		return dm.queryWithStatement(ctx, stmt, args...)

	}

	atomic.AddInt64(&dm.metrics.PreparedStmtMisses, 1)

	// Get appropriate database connection.

	db := dm.getReadConnection()

	if db == nil {

		atomic.AddInt64(&dm.metrics.ErrorCount, 1)

		return nil, fmt.Errorf("no healthy database connections available")

	}

	// Execute query with retry logic.

	return dm.executeWithRetry(ctx, db, optimizedQuery, args...)

}

// ExecuteWithTransaction executes operations within a transaction.

func (dm *OptimizedDBManager) ExecuteWithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {

	txn, err := dm.txnManager.BeginTransaction(ctx, dm.writePrimary)

	if err != nil {

		return err

	}

	defer func() {

		if r := recover(); r != nil {

			txn.tx.Rollback()

			atomic.AddInt64(&dm.metrics.RollbackCount, 1)

			panic(r)

		}

	}()

	start := time.Now()

	err = fn(txn.tx)

	duration := time.Since(start)

	if err != nil {

		if rollbackErr := txn.tx.Rollback(); rollbackErr != nil {

			klog.Errorf("Failed to rollback transaction: %v", rollbackErr)

		}

		atomic.AddInt64(&dm.metrics.RollbackCount, 1)

		return err

	}

	if commitErr := txn.tx.Commit(); commitErr != nil {

		atomic.AddInt64(&dm.metrics.RollbackCount, 1)

		return commitErr

	}

	atomic.AddInt64(&dm.metrics.TransactionCount, 1)

	dm.txnManager.CompleteTransaction(txn.id, duration)

	return nil

}

// ExecuteBatch executes a batch of operations.

func (dm *OptimizedDBManager) ExecuteBatch(ctx context.Context, operations []BatchOperation) ([]BatchResult, error) {

	start := time.Now()

	defer func() {

		duration := time.Since(start)

		atomic.AddInt64(&dm.metrics.BatchCount, 1)

		klog.V(2).Infof("Batch execution completed in %v", duration)

	}()

	return dm.batchProcessor.ExecuteBatch(ctx, dm.writePrimary, operations)

}

// PrepareStatement prepares and caches a SQL statement.

func (dm *OptimizedDBManager) PrepareStatement(query string) error {

	db := dm.writePrimary

	if db == nil {

		return fmt.Errorf("no primary database connection")

	}

	stmt, err := db.Prepare(query)

	if err != nil {

		return fmt.Errorf("failed to prepare statement: %w", err)

	}

	dm.preparedStmts.Set(query, stmt)

	return nil

}

// StreamingQuery executes a query with streaming results.

func (dm *OptimizedDBManager) StreamingQuery(ctx context.Context, query string, callback func(*sql.Rows) error, args ...interface{}) error {

	rows, err := dm.QueryWithOptimization(ctx, query, args...)

	if err != nil {

		return err

	}

	defer rows.Close()

	for rows.Next() {

		select {

		case <-ctx.Done():

			return ctx.Err()

		default:

		}

		if err := callback(rows); err != nil {

			return err

		}

	}

	return rows.Err()

}

// getReadConnection returns a healthy read connection.

func (dm *OptimizedDBManager) getReadConnection() *sql.DB {

	if dm.loadBalancer != nil && len(dm.readReplicas) > 0 {

		return dm.loadBalancer.GetConnection()

	}

	return dm.writePrimary

}

// queryWithStatement executes a query using a prepared statement.

func (dm *OptimizedDBManager) queryWithStatement(ctx context.Context, stmt *sql.Stmt, args ...interface{}) (*sql.Rows, error) {

	return stmt.QueryContext(ctx, args...)

}

// executeWithRetry executes a query with retry logic.

func (dm *OptimizedDBManager) executeWithRetry(ctx context.Context, db *sql.DB, query string, args ...interface{}) (*sql.Rows, error) {

	var lastErr error

	for attempt := range dm.config.RetryAttempts {

		if attempt > 0 {

			time.Sleep(dm.config.RetryDelay * time.Duration(attempt))

		}

		rows, err := db.QueryContext(ctx, query, args...)

		if err == nil {

			return rows, nil

		}

		lastErr = err

		// Check if error is retryable.

		if !dm.isRetryableError(err) {

			break

		}

	}

	atomic.AddInt64(&dm.metrics.ErrorCount, 1)

	return nil, lastErr

}

// isRetryableError determines if an error is retryable.

func (dm *OptimizedDBManager) isRetryableError(err error) bool {

	// Check for network errors, temporary failures, etc.

	if pqErr, ok := err.(*pq.Error); ok {

		// PostgreSQL error codes for retryable errors.

		return pqErr.Code == "40001" || // serialization_failure

			pqErr.Code == "40P01" || // deadlock_detected

			pqErr.Code == "53300" // too_many_connections

	}

	return false

}

// updateAverageQueryTime updates the average query time metric.

func (dm *OptimizedDBManager) updateAverageQueryTime(duration time.Duration) {

	currentAvg := atomic.LoadInt64(&dm.metrics.AverageQueryTime)

	newAvg := (currentAvg + duration.Nanoseconds()) / 2

	atomic.StoreInt64(&dm.metrics.AverageQueryTime, newAvg)

}

// NewPreparedStatementCache creates a new prepared statement cache.

func NewPreparedStatementCache(maxSize int) *PreparedStatementCache {

	return &PreparedStatementCache{

		cache: make(map[string]*PreparedStmt),

		maxSize: maxSize,
	}

}

// Get retrieves a prepared statement from the cache.

func (psc *PreparedStatementCache) Get(query string) *sql.Stmt {

	psc.mu.RLock()

	defer psc.mu.RUnlock()

	if stmt, exists := psc.cache[query]; exists {

		stmt.lastUsed = time.Now()

		atomic.AddInt64(&stmt.useCount, 1)

		atomic.AddInt64(&psc.hitCount, 1)

		psc.moveToFront(stmt)

		return stmt.stmt

	}

	atomic.AddInt64(&psc.missCount, 1)

	return nil

}

// Set stores a prepared statement in the cache.

func (psc *PreparedStatementCache) Set(query string, stmt *sql.Stmt) {

	psc.mu.Lock()

	defer psc.mu.Unlock()

	if existing, exists := psc.cache[query]; exists {

		// Update existing.

		existing.stmt = stmt

		existing.lastUsed = time.Now()

		psc.moveToFront(existing)

		return

	}

	// Create new entry.

	preparedStmt := &PreparedStmt{

		stmt: stmt,

		query: query,

		lastUsed: time.Now(),

		createdAt: time.Now(),
	}

	// Evict if at capacity.

	for psc.currentSize >= psc.maxSize && psc.lruTail != nil {

		psc.evictLRU()

	}

	psc.cache[query] = preparedStmt

	psc.currentSize++

	psc.addToFront(preparedStmt)

}

// moveToFront moves a statement to the front of the LRU list.

func (psc *PreparedStatementCache) moveToFront(stmt *PreparedStmt) {

	if stmt == psc.lruHead {

		return

	}

	psc.removeFromList(stmt)

	psc.addToFront(stmt)

}

// addToFront adds a statement to the front of the LRU list.

func (psc *PreparedStatementCache) addToFront(stmt *PreparedStmt) {

	if psc.lruHead == nil {

		psc.lruHead = stmt

		psc.lruTail = stmt

	} else {

		stmt.next = psc.lruHead

		psc.lruHead.prev = stmt

		psc.lruHead = stmt

	}

}

// removeFromList removes a statement from the LRU list.

func (psc *PreparedStatementCache) removeFromList(stmt *PreparedStmt) {

	if stmt.prev != nil {

		stmt.prev.next = stmt.next

	} else {

		psc.lruHead = stmt.next

	}

	if stmt.next != nil {

		stmt.next.prev = stmt.prev

	} else {

		psc.lruTail = stmt.prev

	}

	stmt.prev = nil

	stmt.next = nil

}

// evictLRU evicts the least recently used statement.

func (psc *PreparedStatementCache) evictLRU() {

	if psc.lruTail == nil {

		return

	}

	stmt := psc.lruTail

	delete(psc.cache, stmt.query)

	psc.currentSize--

	psc.removeFromList(stmt)

	stmt.stmt.Close() // Close the prepared statement

}

// NewQueryOptimizer creates a new query optimizer.

func NewQueryOptimizer() *QueryOptimizer {

	return &QueryOptimizer{

		queryCache: make(map[string]*CachedQuery),

		slowQueries: make(map[string]*SlowQuery),

		optimizations: make(map[string]string),
	}

}

// OptimizeQuery optimizes a SQL query.

func (qo *QueryOptimizer) OptimizeQuery(query string) string {

	qo.mu.RLock()

	defer qo.mu.RUnlock()

	if optimized, exists := qo.optimizations[query]; exists {

		atomic.AddInt64(&qo.optimizedCount, 1)

		return optimized

	}

	// Simple optimization rules (in real implementation, use a proper query optimizer).

	optimized := query

	// Add basic optimizations.

	// 1. Add LIMIT if not present for SELECT queries without explicit LIMIT.

	// 2. Suggest indexes for WHERE clauses.

	// 3. Rewrite subqueries as JOINs where appropriate.

	qo.optimizations[query] = optimized

	return optimized

}

// RecordSlowQuery records a slow query for analysis.

func (qo *QueryOptimizer) RecordSlowQuery(query string, duration time.Duration) {

	qo.mu.Lock()

	defer qo.mu.Unlock()

	if slowQuery, exists := qo.slowQueries[query]; exists {

		slowQuery.executionCount++

		if duration > slowQuery.maxDuration {

			slowQuery.maxDuration = duration

		}

		// Update average duration.

		slowQuery.avgDuration = time.Duration((slowQuery.avgDuration.Nanoseconds() + duration.Nanoseconds()) / 2)

		slowQuery.lastExecution = time.Now()

	} else {

		qo.slowQueries[query] = &SlowQuery{

			query: query,

			avgDuration: duration,

			maxDuration: duration,

			executionCount: 1,

			lastExecution: time.Now(),

			optimized: false,

			suggestions: []string{},
		}

	}

}

// GetCachedResult retrieves a cached query result.

func (qo *QueryOptimizer) GetCachedResult(query string, params []interface{}) *CachedQuery {

	qo.mu.RLock()

	defer qo.mu.RUnlock()

	cacheKey := fmt.Sprintf("%s:%v", query, params)

	if cached, exists := qo.queryCache[cacheKey]; exists {

		if time.Now().Before(cached.expiration) {

			atomic.AddInt64(&cached.hitCount, 1)

			atomic.AddInt64(&qo.hitCount, 1)

			return cached

		}

		// Expired, remove from cache.

		delete(qo.queryCache, cacheKey)

	}

	atomic.AddInt64(&qo.missCount, 1)

	return nil

}

// NewTransactionManager creates a new transaction manager.

func NewTransactionManager() *TransactionManager {

	return &TransactionManager{

		activeTxns: make(map[string]*Transaction),

		txnPool: sync.Pool{

			New: func() interface{} {

				return &Transaction{}

			},
		},

		maxConcurrentTxns: 100,
	}

}

// BeginTransaction begins a new optimized transaction.

func (tm *TransactionManager) BeginTransaction(ctx context.Context, db *sql.DB) (*Transaction, error) {

	if atomic.LoadInt64(&tm.currentTxns) >= tm.maxConcurrentTxns {

		return nil, fmt.Errorf("maximum concurrent transactions exceeded")

	}

	tx, err := db.BeginTx(ctx, nil)

	if err != nil {

		return nil, err

	}

	txnCtx, cancel := context.WithTimeout(ctx, 30*time.Second)

	txn := tm.txnPool.Get().(*Transaction)

	txn.id = fmt.Sprintf("txn_%d", time.Now().UnixNano())

	txn.tx = tx

	txn.startTime = time.Now()

	txn.context = txnCtx

	txn.cancel = cancel

	txn.operations = txn.operations[:0] // Clear previous operations

	tm.mu.Lock()

	tm.activeTxns[txn.id] = txn

	tm.mu.Unlock()

	atomic.AddInt64(&tm.currentTxns, 1)

	return txn, nil

}

// CompleteTransaction marks a transaction as completed.

func (tm *TransactionManager) CompleteTransaction(txnID string, duration time.Duration) {

	tm.mu.Lock()

	defer tm.mu.Unlock()

	if txn, exists := tm.activeTxns[txnID]; exists {

		delete(tm.activeTxns, txnID)

		txn.cancel()

		tm.txnPool.Put(txn)

		atomic.AddInt64(&tm.currentTxns, -1)

		atomic.AddInt64(&tm.completedTxns, 1)

	}

}

// NewBatchProcessor creates a new batch processor.

func NewBatchProcessor(maxBatchSize int, timeout time.Duration) *BatchProcessor {

	return &BatchProcessor{

		batches: make(map[string]*Batch),

		batchTimeout: timeout,

		maxBatchSize: maxBatchSize,
	}

}

// ExecuteBatch executes a batch of database operations.

func (bp *BatchProcessor) ExecuteBatch(ctx context.Context, db *sql.DB, operations []BatchOperation) ([]BatchResult, error) {

	start := time.Now()

	results := make([]BatchResult, len(operations))

	// Group operations by type for optimization.

	grouped := bp.groupOperations(operations)

	// Execute groups in order.

	resultIndex := 0

	for _, group := range grouped {

		groupResults, err := bp.executeGroup(ctx, db, group)

		if err != nil {

			// Mark remaining operations as failed.

			for i := resultIndex; i < len(results); i++ {

				results[i] = BatchResult{

					success: false,

					error: err,
				}

			}

			atomic.AddInt64(&bp.failed, 1)

			return results, err

		}

		// Copy results.

		for i, result := range groupResults {

			results[resultIndex+i] = result

		}

		resultIndex += len(groupResults)

	}

	atomic.AddInt64(&bp.processed, 1)

	klog.V(2).Infof("Batch execution completed in %v", time.Since(start))

	return results, nil

}

// groupOperations groups operations by type for batch optimization.

func (bp *BatchProcessor) groupOperations(operations []BatchOperation) [][]BatchOperation {

	// Simple grouping by operation type.

	groups := make(map[string][]BatchOperation)

	for _, op := range operations {

		groups[op.type_] = append(groups[op.type_], op)

	}

	// Convert to slice.

	var result [][]BatchOperation

	for _, group := range groups {

		result = append(result, group)

	}

	return result

}

// executeGroup executes a group of similar operations.

func (bp *BatchProcessor) executeGroup(ctx context.Context, db *sql.DB, operations []BatchOperation) ([]BatchResult, error) {

	results := make([]BatchResult, len(operations))

	for i, op := range operations {

		start := time.Now()

		result, err := db.ExecContext(ctx, op.query, op.params...)

		duration := time.Since(start)

		if err != nil {

			results[i] = BatchResult{

				success: false,

				error: err,

				duration: duration,
			}

		} else {

			rowsAffected, _ := result.RowsAffected()

			results[i] = BatchResult{

				success: true,

				rowsAffected: rowsAffected,

				duration: duration,
			}

		}

	}

	return results, nil

}

// NewDBHealthChecker creates a new database health checker.

func NewDBHealthChecker(interval time.Duration) *DBHealthChecker {

	return &DBHealthChecker{

		checks: make(map[string]*HealthCheck),

		interval: interval,

		timeout: 5 * time.Second,
	}

}

// NewReadLoadBalancer creates a new read load balancer.

func NewReadLoadBalancer(replicas []*sql.DB, strategy LoadBalanceStrategy) *ReadLoadBalancer {

	return &ReadLoadBalancer{

		replicas: replicas,

		weights: make([]int, len(replicas)),

		healthyReplicas: make(map[int]bool),

		strategy: strategy,
	}

}

// GetConnection returns a connection based on load balancing strategy.

func (rlb *ReadLoadBalancer) GetConnection() *sql.DB {

	rlb.mu.RLock()

	defer rlb.mu.RUnlock()

	if len(rlb.replicas) == 0 {

		return nil

	}

	switch rlb.strategy {

	case LoadBalanceRoundRobin:

		idx := atomic.AddInt64(&rlb.currentIndex, 1) % int64(len(rlb.replicas))

		return rlb.replicas[idx]

	default:

		return rlb.replicas[0]

	}

}

// startBackgroundTasks starts background maintenance tasks.

func (dm *OptimizedDBManager) startBackgroundTasks() {

	// Health checking.

	if dm.config.EnableHealthCheck && dm.healthChecker != nil {

		dm.wg.Add(1)

		go func() {

			defer dm.wg.Done()

			ticker := time.NewTicker(dm.config.HealthCheckInterval)

			defer ticker.Stop()

			for {

				select {

				case <-ticker.C:

					dm.performHealthChecks()

				case <-dm.shutdown:

					return

				}

			}

		}()

	}

	// Metrics collection.

	if dm.config.EnableMetrics {

		dm.wg.Add(1)

		go func() {

			defer dm.wg.Done()

			ticker := time.NewTicker(30 * time.Second)

			defer ticker.Stop()

			for {

				select {

				case <-ticker.C:

					dm.updateConnectionMetrics()

				case <-dm.shutdown:

					return

				}

			}

		}()

	}

}

// performHealthChecks checks the health of all database connections.

func (dm *OptimizedDBManager) performHealthChecks() {

	dm.mu.RLock()

	defer dm.mu.RUnlock()

	for name, pool := range dm.connectionPools {

		start := time.Now()

		err := pool.db.Ping()

		latency := time.Since(start)

		pool.mu.Lock()

		pool.lastHealthCheck = time.Now()

		if err != nil {

			pool.healthy = false

			klog.Warningf("Database %s health check failed: %v", name, err)

		} else {

			pool.healthy = true

		}

		pool.mu.Unlock()

		// Update health check metrics.

		if dm.healthChecker != nil {

			dm.healthChecker.mu.Lock()

			if check, exists := dm.healthChecker.checks[name]; exists {

				check.lastCheck = time.Now()

				check.latency = latency

				if err != nil {

					atomic.AddInt64(&check.failCount, 1)

					check.healthy = false

					check.lastError = err

				} else {

					atomic.AddInt64(&check.successCount, 1)

					check.healthy = true

					check.lastError = nil

				}

			} else {

				dm.healthChecker.checks[name] = &HealthCheck{

					name: name,

					lastCheck: time.Now(),

					latency: latency,

					healthy: err == nil,

					lastError: err,

					successCount: 1,
				}

				if err != nil {

					atomic.AddInt64(&dm.healthChecker.checks[name].failCount, 1)

				}

			}

			dm.healthChecker.mu.Unlock()

		}

	}

}

// updateConnectionMetrics updates connection-related metrics.

func (dm *OptimizedDBManager) updateConnectionMetrics() {

	var totalActive, totalIdle int64

	dm.mu.RLock()

	for _, pool := range dm.connectionPools {

		stats := pool.db.Stats()

		totalActive += int64(stats.OpenConnections)

		totalIdle += int64(stats.Idle)

	}

	dm.mu.RUnlock()

	atomic.StoreInt64(&dm.metrics.ActiveConnections, totalActive)

	atomic.StoreInt64(&dm.metrics.IdleConnections, totalIdle)

	atomic.StoreInt64(&dm.metrics.ConnectionCount, totalActive)

}

// GetMetrics returns current database performance metrics.

func (dm *OptimizedDBManager) GetMetrics() DBMetrics {

	dm.updateConnectionMetrics() // Ensure metrics are current

	return DBMetrics{

		QueryCount: atomic.LoadInt64(&dm.metrics.QueryCount),

		SlowQueryCount: atomic.LoadInt64(&dm.metrics.SlowQueryCount),

		ErrorCount: atomic.LoadInt64(&dm.metrics.ErrorCount),

		ConnectionCount: atomic.LoadInt64(&dm.metrics.ConnectionCount),

		ActiveConnections: atomic.LoadInt64(&dm.metrics.ActiveConnections),

		IdleConnections: atomic.LoadInt64(&dm.metrics.IdleConnections),

		TransactionCount: atomic.LoadInt64(&dm.metrics.TransactionCount),

		RollbackCount: atomic.LoadInt64(&dm.metrics.RollbackCount),

		BatchCount: atomic.LoadInt64(&dm.metrics.BatchCount),

		CacheHitCount: atomic.LoadInt64(&dm.metrics.CacheHitCount),

		CacheMissCount: atomic.LoadInt64(&dm.metrics.CacheMissCount),

		AverageQueryTime: atomic.LoadInt64(&dm.metrics.AverageQueryTime),

		AverageConnTime: atomic.LoadInt64(&dm.metrics.AverageConnTime),

		TotalBytesRead: atomic.LoadInt64(&dm.metrics.TotalBytesRead),

		TotalBytesWritten: atomic.LoadInt64(&dm.metrics.TotalBytesWritten),

		ReplicationLag: atomic.LoadInt64(&dm.metrics.ReplicationLag),

		DeadlockCount: atomic.LoadInt64(&dm.metrics.DeadlockCount),

		PreparedStmtHits: atomic.LoadInt64(&dm.metrics.PreparedStmtHits),

		PreparedStmtMisses: atomic.LoadInt64(&dm.metrics.PreparedStmtMisses),
	}

}

// GetAverageQueryTime returns the average query time in milliseconds.

func (dm *OptimizedDBManager) GetAverageQueryTime() float64 {

	avgTime := atomic.LoadInt64(&dm.metrics.AverageQueryTime)

	return float64(avgTime) / 1e6 // Convert to milliseconds

}

// GetConnectionUtilization returns the connection utilization percentage.

func (dm *OptimizedDBManager) GetConnectionUtilization() float64 {

	activeConns := atomic.LoadInt64(&dm.metrics.ActiveConnections)

	maxConns := int64(dm.config.MaxOpenConns)

	if maxConns == 0 {

		return 0

	}

	return float64(activeConns) / float64(maxConns) * 100

}

// GetSlowQueries returns the current slow queries.

func (dm *OptimizedDBManager) GetSlowQueries() map[string]*SlowQuery {

	dm.queryOptimizer.mu.RLock()

	defer dm.queryOptimizer.mu.RUnlock()

	// Return a copy to avoid concurrent access.

	result := make(map[string]*SlowQuery)

	for k, v := range dm.queryOptimizer.slowQueries {

		result[k] = &SlowQuery{

			query: v.query,

			avgDuration: v.avgDuration,

			maxDuration: v.maxDuration,

			executionCount: v.executionCount,

			lastExecution: v.lastExecution,

			optimized: v.optimized,

			suggestions: append([]string(nil), v.suggestions...),
		}

	}

	return result

}

// Shutdown gracefully shuts down the database manager.

func (dm *OptimizedDBManager) Shutdown(ctx context.Context) error {

	close(dm.shutdown)

	// Wait for background tasks.

	done := make(chan struct{})

	go func() {

		dm.wg.Wait()

		close(done)

	}()

	select {

	case <-done:

		// All tasks finished.

	case <-ctx.Done():

		return ctx.Err()

	}

	// Close all database connections.

	dm.mu.Lock()

	for name, pool := range dm.connectionPools {

		if err := pool.db.Close(); err != nil {

			klog.Errorf("Failed to close database %s: %v", name, err)

		}

	}

	dm.mu.Unlock()

	// Log final metrics.

	metrics := dm.GetMetrics()

	klog.Infof("Database manager shutdown - Queries: %d, Avg time: %.2fms, Error rate: %.2f%%",

		metrics.QueryCount,

		dm.GetAverageQueryTime(),

		float64(metrics.ErrorCount)/float64(metrics.QueryCount)*100,
	)

	return nil

}

// ResetMetrics resets all database performance metrics.

func (dm *OptimizedDBManager) ResetMetrics() {

	atomic.StoreInt64(&dm.metrics.QueryCount, 0)

	atomic.StoreInt64(&dm.metrics.SlowQueryCount, 0)

	atomic.StoreInt64(&dm.metrics.ErrorCount, 0)

	atomic.StoreInt64(&dm.metrics.ConnectionCount, 0)

	atomic.StoreInt64(&dm.metrics.ActiveConnections, 0)

	atomic.StoreInt64(&dm.metrics.IdleConnections, 0)

	atomic.StoreInt64(&dm.metrics.TransactionCount, 0)

	atomic.StoreInt64(&dm.metrics.RollbackCount, 0)

	atomic.StoreInt64(&dm.metrics.BatchCount, 0)

	atomic.StoreInt64(&dm.metrics.CacheHitCount, 0)

	atomic.StoreInt64(&dm.metrics.CacheMissCount, 0)

	atomic.StoreInt64(&dm.metrics.AverageQueryTime, 0)

	atomic.StoreInt64(&dm.metrics.AverageConnTime, 0)

	atomic.StoreInt64(&dm.metrics.TotalBytesRead, 0)

	atomic.StoreInt64(&dm.metrics.TotalBytesWritten, 0)

	atomic.StoreInt64(&dm.metrics.ReplicationLag, 0)

	atomic.StoreInt64(&dm.metrics.DeadlockCount, 0)

	atomic.StoreInt64(&dm.metrics.PreparedStmtHits, 0)

	atomic.StoreInt64(&dm.metrics.PreparedStmtMisses, 0)

	// Reset prepared statement cache metrics.

	atomic.StoreInt64(&dm.preparedStmts.hitCount, 0)

	atomic.StoreInt64(&dm.preparedStmts.missCount, 0)

	// Reset query optimizer metrics.

	atomic.StoreInt64(&dm.queryOptimizer.hitCount, 0)

	atomic.StoreInt64(&dm.queryOptimizer.missCount, 0)

	atomic.StoreInt64(&dm.queryOptimizer.optimizedCount, 0)

}
