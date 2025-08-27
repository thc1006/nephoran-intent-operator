//go:build go1.24

package performance

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"sync"
	"time"
)

// DBMonitor provides real-time database performance monitoring
type DBMonitor struct {
	db              *sql.DB
	slowQueryLog    map[string]*SlowQueryInfo
	queryPatterns   map[string]*QueryPattern
	alertThresholds *AlertThresholds
	metrics         *DBMonitorMetrics
	mu              sync.RWMutex
	stopCh          chan struct{}
	logger          *log.Logger
}

// SlowQueryInfo contains information about slow queries
type SlowQueryInfo struct {
	Query          string        `json:"query"`
	AverageTime    time.Duration `json:"average_time"`
	MaxTime        time.Duration `json:"max_time"`
	ExecutionCount int64         `json:"execution_count"`
	LastSeen       time.Time     `json:"last_seen"`
	Impact         string        `json:"impact"` // HIGH, MEDIUM, LOW
	Suggestions    []string      `json:"suggestions"`
}

// QueryPattern identifies common query patterns for optimization
type QueryPattern struct {
	Pattern     string        `json:"pattern"`
	Count       int64         `json:"count"`
	AvgDuration time.Duration `json:"avg_duration"`
	Tables      []string      `json:"tables"`
	LastSeen    time.Time     `json:"last_seen"`
}

// AlertThresholds defines when to trigger alerts
type AlertThresholds struct {
	SlowQueryThreshold    time.Duration `json:"slow_query_threshold"`
	ConnectionUtilization float64       `json:"connection_utilization"`
	ErrorRate             float64       `json:"error_rate"`
	CacheHitRate          float64       `json:"cache_hit_rate"`
	DeadlockCount         int64         `json:"deadlock_count"`
	ReplicationLag        time.Duration `json:"replication_lag"`
}

// DBMonitorMetrics tracks monitoring metrics
type DBMonitorMetrics struct {
	TotalQueries        int64         `json:"total_queries"`
	SlowQueries         int64         `json:"slow_queries"`
	FastQueries         int64         `json:"fast_queries"`
	ErrorQueries        int64         `json:"error_queries"`
	AverageQueryTime    time.Duration `json:"average_query_time"`
	P99QueryTime        time.Duration `json:"p99_query_time"`
	ConnectionPoolUsage float64       `json:"connection_pool_usage"`
	LastUpdated         time.Time     `json:"last_updated"`
}

// NewDBMonitor creates a new database monitor
func NewDBMonitor(db *sql.DB) *DBMonitor {
	return &DBMonitor{
		db:            db,
		slowQueryLog:  make(map[string]*SlowQueryInfo),
		queryPatterns: make(map[string]*QueryPattern),
		alertThresholds: &AlertThresholds{
			SlowQueryThreshold:    500 * time.Millisecond,
			ConnectionUtilization: 0.8,
			ErrorRate:             0.05, // 5%
			CacheHitRate:          0.8,  // 80%
			DeadlockCount:         10,
			ReplicationLag:        5 * time.Second,
		},
		metrics: &DBMonitorMetrics{},
		stopCh:  make(chan struct{}),
		logger:  log.New(log.Writer(), "[DBMonitor] ", log.LstdFlags),
	}
}

// Start begins monitoring database performance
func (dm *DBMonitor) Start(ctx context.Context) error {
	// Monitor slow queries
	go dm.monitorSlowQueries(ctx)

	// Monitor connection pool
	go dm.monitorConnectionPool(ctx)

	// Monitor query patterns
	go dm.analyzeQueryPatterns(ctx)

	// Generate performance reports
	go dm.generateReports(ctx)

	dm.logger.Println("Database monitor started")
	return nil
}

// Stop stops the database monitor
func (dm *DBMonitor) Stop() {
	close(dm.stopCh)
	dm.logger.Println("Database monitor stopped")
}

// monitorSlowQueries monitors and analyzes slow queries
func (dm *DBMonitor) monitorSlowQueries(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-dm.stopCh:
			return
		case <-ticker.C:
			dm.analyzeSlowQueries(ctx)
		}
	}
}

// analyzeSlowQueries queries pg_stat_statements for slow queries
func (dm *DBMonitor) analyzeSlowQueries(ctx context.Context) {
	query := `
		SELECT 
			query,
			mean_exec_time,
			max_exec_time,
			calls,
			total_exec_time
		FROM pg_stat_statements 
		WHERE mean_exec_time > $1
		ORDER BY mean_exec_time DESC 
		LIMIT 20`

	rows, err := dm.db.QueryContext(ctx, query, dm.alertThresholds.SlowQueryThreshold.Milliseconds())
	if err != nil {
		dm.logger.Printf("Error querying slow queries: %v", err)
		return
	}
	defer rows.Close()

	dm.mu.Lock()
	defer dm.mu.Unlock()

	for rows.Next() {
		var queryText string
		var meanTime, maxTime, totalTime float64
		var calls int64

		if err := rows.Scan(&queryText, &meanTime, &maxTime, &calls, &totalTime); err != nil {
			continue
		}

		// Normalize query (remove specific values)
		normalizedQuery := dm.normalizeQuery(queryText)

		slowQuery := &SlowQueryInfo{
			Query:          normalizedQuery,
			AverageTime:    time.Duration(meanTime) * time.Millisecond,
			MaxTime:        time.Duration(maxTime) * time.Millisecond,
			ExecutionCount: calls,
			LastSeen:       time.Now(),
			Impact:         dm.calculateImpact(meanTime, calls, totalTime),
			Suggestions:    dm.generateOptimizationSuggestions(queryText),
		}

		dm.slowQueryLog[normalizedQuery] = slowQuery
		dm.metrics.SlowQueries++
	}
}

// monitorConnectionPool monitors database connection pool metrics
func (dm *DBMonitor) monitorConnectionPool(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-dm.stopCh:
			return
		case <-ticker.C:
			dm.updateConnectionMetrics()
		}
	}
}

// updateConnectionMetrics updates connection pool metrics
func (dm *DBMonitor) updateConnectionMetrics() {
	stats := dm.db.Stats()

	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.metrics.ConnectionPoolUsage = float64(stats.OpenConnections) / float64(stats.MaxOpenConnections)
	dm.metrics.LastUpdated = time.Now()

	// Alert if connection utilization is high
	if dm.metrics.ConnectionPoolUsage > dm.alertThresholds.ConnectionUtilization {
		dm.logger.Printf("HIGH CONNECTION UTILIZATION: %.2f%% (%d/%d connections)",
			dm.metrics.ConnectionPoolUsage*100,
			stats.OpenConnections,
			stats.MaxOpenConnections)
	}
}

// analyzeQueryPatterns identifies common query patterns
func (dm *DBMonitor) analyzeQueryPatterns(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-dm.stopCh:
			return
		case <-ticker.C:
			dm.identifyPatterns(ctx)
		}
	}
}

// identifyPatterns identifies common query patterns from pg_stat_statements
func (dm *DBMonitor) identifyPatterns(ctx context.Context) {
	query := `
		SELECT 
			query,
			calls,
			mean_exec_time
		FROM pg_stat_statements 
		WHERE calls > 10
		ORDER BY calls DESC 
		LIMIT 50`

	rows, err := dm.db.QueryContext(ctx, query)
	if err != nil {
		dm.logger.Printf("Error analyzing query patterns: %v", err)
		return
	}
	defer rows.Close()

	patterns := make(map[string]*QueryPattern)

	for rows.Next() {
		var queryText string
		var calls int64
		var meanTime float64

		if err := rows.Scan(&queryText, &calls, &meanTime); err != nil {
			continue
		}

		pattern := dm.extractPattern(queryText)
		tables := dm.extractTables(queryText)

		if existing, found := patterns[pattern]; found {
			existing.Count += calls
			existing.AvgDuration = (existing.AvgDuration + time.Duration(meanTime)*time.Millisecond) / 2
		} else {
			patterns[pattern] = &QueryPattern{
				Pattern:     pattern,
				Count:       calls,
				AvgDuration: time.Duration(meanTime) * time.Millisecond,
				Tables:      tables,
				LastSeen:    time.Now(),
			}
		}
	}

	dm.mu.Lock()
	dm.queryPatterns = patterns
	dm.mu.Unlock()
}

// generateReports generates performance reports
func (dm *DBMonitor) generateReports(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-dm.stopCh:
			return
		case <-ticker.C:
			dm.generatePerformanceReport()
		}
	}
}

// generatePerformanceReport creates a performance report
func (dm *DBMonitor) generatePerformanceReport() {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	dm.logger.Println("=== DATABASE PERFORMANCE REPORT ===")
	dm.logger.Printf("Connection Pool Usage: %.2f%%", dm.metrics.ConnectionPoolUsage*100)
	dm.logger.Printf("Total Slow Queries: %d", len(dm.slowQueryLog))
	dm.logger.Printf("Query Patterns Identified: %d", len(dm.queryPatterns))

	// Top 5 slowest queries
	dm.logger.Println("\n--- TOP 5 SLOWEST QUERIES ---")
	count := 0
	for _, slow := range dm.slowQueryLog {
		if count >= 5 {
			break
		}
		dm.logger.Printf("%d. Query: %s...", count+1, dm.truncateQuery(slow.Query, 80))
		dm.logger.Printf("   Avg Time: %v, Max Time: %v, Executions: %d, Impact: %s",
			slow.AverageTime, slow.MaxTime, slow.ExecutionCount, slow.Impact)
		count++
	}

	// Top query patterns
	dm.logger.Println("\n--- TOP QUERY PATTERNS ---")
	count = 0
	for _, pattern := range dm.queryPatterns {
		if count >= 3 {
			break
		}
		dm.logger.Printf("%d. Pattern: %s", count+1, pattern.Pattern)
		dm.logger.Printf("   Count: %d, Avg Duration: %v, Tables: %v",
			pattern.Count, pattern.AvgDuration, pattern.Tables)
		count++
	}
}

// GetTopSlowQueries returns the top N slow queries
func (dm *DBMonitor) GetTopSlowQueries(n int) []*SlowQueryInfo {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	queries := make([]*SlowQueryInfo, 0, len(dm.slowQueryLog))
	for _, query := range dm.slowQueryLog {
		queries = append(queries, query)
	}

	// Sort by average time descending
	for i := 0; i < len(queries)-1; i++ {
		for j := i + 1; j < len(queries); j++ {
			if queries[i].AverageTime < queries[j].AverageTime {
				queries[i], queries[j] = queries[j], queries[i]
			}
		}
	}

	if n > len(queries) {
		n = len(queries)
	}
	return queries[:n]
}

// GetQueryPatterns returns identified query patterns
func (dm *DBMonitor) GetQueryPatterns() map[string]*QueryPattern {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	patterns := make(map[string]*QueryPattern)
	for k, v := range dm.queryPatterns {
		patterns[k] = v
	}
	return patterns
}

// Helper functions

// normalizeQuery removes specific values to create a pattern
func (dm *DBMonitor) normalizeQuery(query string) string {
	// Simple normalization - in production, use a more sophisticated approach
	normalized := query
	normalized = strings.ReplaceAll(normalized, "'[^']+'", "?")
	normalized = strings.ReplaceAll(normalized, "\\d+", "?")
	return normalized
}

// calculateImpact determines the performance impact of a slow query
func (dm *DBMonitor) calculateImpact(meanTime float64, calls int64, totalTime float64) string {
	if meanTime > 5000 || totalTime > 60000 { // >5s avg or >1min total
		return "HIGH"
	} else if meanTime > 1000 || totalTime > 10000 { // >1s avg or >10s total
		return "MEDIUM"
	}
	return "LOW"
}

// generateOptimizationSuggestions provides optimization suggestions for queries
func (dm *DBMonitor) generateOptimizationSuggestions(query string) []string {
	var suggestions []string
	queryUpper := strings.ToUpper(query)

	if strings.Contains(queryUpper, "SELECT *") {
		suggestions = append(suggestions, "Consider selecting specific columns instead of using SELECT *")
	}

	if strings.Contains(queryUpper, "ORDER BY") && !strings.Contains(queryUpper, "LIMIT") {
		suggestions = append(suggestions, "Consider adding LIMIT to ORDER BY queries")
	}

	if strings.Contains(queryUpper, "WHERE") {
		if strings.Contains(query, "networkintents") {
			suggestions = append(suggestions, "Ensure idx_networkintent_status index is being used")
		}
		if strings.Contains(query, "ves_events") {
			suggestions = append(suggestions, "Add time-based filtering to VES event queries")
		}
	}

	if strings.Contains(queryUpper, "JOIN") && strings.Count(queryUpper, "JOIN") > 3 {
		suggestions = append(suggestions, "Consider breaking down complex JOINs into smaller queries")
	}

	return suggestions
}

// extractPattern extracts a query pattern by removing literals
func (dm *DBMonitor) extractPattern(query string) string {
	// Simplified pattern extraction
	pattern := query
	pattern = strings.ReplaceAll(pattern, "'[^']+'", "'?'")
	pattern = strings.ReplaceAll(pattern, "\\d+", "?")
	pattern = strings.ReplaceAll(pattern, "\\$\\d+", "$?")
	return pattern
}

// extractTables extracts table names from a query
func (dm *DBMonitor) extractTables(query string) []string {
	var tables []string
	queryUpper := strings.ToUpper(query)

	// Common O-RAN tables
	oranTables := []string{
		"NETWORKINTENTS", "VES_EVENTS", "A1_POLICIES", "E2_SUBSCRIPTIONS",
		"PERFORMANCE_METRICS", "DOCUMENT_CHUNKS", "EMBEDDINGS_CACHE",
	}

	for _, table := range oranTables {
		if strings.Contains(queryUpper, table) {
			tables = append(tables, strings.ToLower(table))
		}
	}

	return tables
}

// truncateQuery truncates a query string for display
func (dm *DBMonitor) truncateQuery(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen-3] + "..."
}
