//go:build !disable_rag && !test

package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RAGMonitor provides comprehensive monitoring and observability for RAG components
type RAGMonitor struct {
	config         *MonitoringConfig
	logger         *slog.Logger
	metricsServer  *http.Server
	healthCheckers map[string]HealthChecker
	alertManager   *AlertManager
	mutex          sync.RWMutex

	// Prometheus metrics
	queryLatencyHistogram   *prometheus.HistogramVec
	queryCountCounter       *prometheus.CounterVec
	embeddingLatencyHist    *prometheus.HistogramVec
	embeddingCountCounter   *prometheus.CounterVec
	cacheHitRateGauge       *prometheus.GaugeVec
	documentProcessingGauge prometheus.Gauge
	chunkingLatencyHist     *prometheus.HistogramVec
	retrievalLatencyHist    *prometheus.HistogramVec
	contextAssemblyHist     *prometheus.HistogramVec
	errorRateCounter        *prometheus.CounterVec
	systemResourceGauges    map[string]prometheus.Gauge
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	// Server configuration
	MetricsPort     int    `json:"metrics_port"`
	MetricsPath     string `json:"metrics_path"`
	HealthCheckPath string `json:"health_check_path"`

	// Collection intervals
	MetricsInterval     time.Duration `json:"metrics_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`

	// Alerting configuration
	EnableAlerting  bool                      `json:"enable_alerting"`
	AlertThresholds map[string]AlertThreshold `json:"alert_thresholds"`
	AlertWebhooks   []string                  `json:"alert_webhooks"`

	// Log configuration
	EnableStructuredLogs bool   `json:"enable_structured_logs"`
	LogLevel             string `json:"log_level"`

	// Trace sampling
	TraceSampleRate          float64 `json:"trace_sample_rate"`
	EnableDistributedTracing bool    `json:"enable_distributed_tracing"`

	// Performance monitoring
	EnableResourceMonitoring   bool          `json:"enable_resource_monitoring"`
	ResourceMonitoringInterval time.Duration `json:"resource_monitoring_interval"`
}

// AlertThreshold defines thresholds for alerting
type AlertThreshold struct {
	MetricName    string  `json:"metric_name"`
	Threshold     float64 `json:"threshold"`
	Comparison    string  `json:"comparison"` // "greater", "less", "equal"
	WindowMinutes int     `json:"window_minutes"`
	Severity      string  `json:"severity"` // "critical", "warning", "info"
}

// HealthChecker interface for component health checks
type HealthChecker interface {
	CheckHealth(ctx context.Context) HealthStatus
	GetComponentName() string
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	ComponentName       string                 `json:"component_name"`
	IsHealthy           bool                   `json:"is_healthy"`
	Status              string                 `json:"status"`
	LastCheck           time.Time              `json:"last_check"`
	ResponseTime        time.Duration          `json:"response_time"`
	AverageLatency      time.Duration          `json:"average_latency"`
	ConsecutiveFailures int                    `json:"consecutive_failures"`
	ErrorMessage        string                 `json:"error_message,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status        string                  `json:"status"`
	IsHealthy     bool                    `json:"is_healthy"`
	Components    map[string]HealthStatus `json:"components"`
	SystemMetrics SystemMetrics           `json:"system_metrics"`
	LastCheck     time.Time               `json:"last_check"`
	UptimeSeconds int64                   `json:"uptime_seconds"`
}

// SystemMetrics holds system-level metrics
type SystemMetrics struct {
	CPUUsagePercent    float64 `json:"cpu_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	DiskUsagePercent   float64 `json:"disk_usage_percent"`
	NetworkBytesIn     int64   `json:"network_bytes_in"`
	NetworkBytesOut    int64   `json:"network_bytes_out"`
	GoroutineCount     int     `json:"goroutine_count"`
	HeapSizeMB         float64 `json:"heap_size_mb"`
}

// AlertManager handles alerting based on metrics and thresholds
type AlertManager struct {
	config   *MonitoringConfig
	logger   *slog.Logger
	alerts   map[string]Alert
	webhooks []string
	mutex    sync.RWMutex
}

// Alert represents an active alert
type Alert struct {
	ID          string                 `json:"id"`
	MetricName  string                 `json:"metric_name"`
	Severity    string                 `json:"severity"`
	Message     string                 `json:"message"`
	Threshold   float64                `json:"threshold"`
	ActualValue float64                `json:"actual_value"`
	StartTime   time.Time              `json:"start_time"`
	LastSeen    time.Time              `json:"last_seen"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// RAGMetricsCollector collects metrics from all RAG components
type RAGMetricsCollector struct {
	monitor          *RAGMonitor
	ragService       *RAGService
	embeddingService *EmbeddingService
	retrievalService *EnhancedRetrievalService
	redisCache       *RedisCache
	mutex            sync.RWMutex
}

// NewRAGMonitor creates a new RAG monitoring system
func NewRAGMonitor(config *MonitoringConfig) *RAGMonitor {
	if config == nil {
		config = getDefaultMonitoringConfig()
	}

	monitor := &RAGMonitor{
		config:               config,
		logger:               slog.Default().With("component", "rag-monitor"),
		healthCheckers:       make(map[string]HealthChecker),
		alertManager:         NewAlertManager(config),
		systemResourceGauges: make(map[string]prometheus.Gauge),
	}

	// Initialize Prometheus metrics
	monitor.initializePrometheusMetrics()

	// Start metrics server
	if err := monitor.startMetricsServer(); err != nil {
		monitor.logger.Error("Failed to start metrics server", "error", err)
	}

	// Start background monitoring tasks
	go monitor.startHealthChecking()
	go monitor.startMetricsCollection()

	if config.EnableResourceMonitoring {
		go monitor.startResourceMonitoring()
	}

	return monitor
}

// initializePrometheusMetrics initializes all Prometheus metrics
func (rm *RAGMonitor) initializePrometheusMetrics() {
	// Query metrics
	rm.queryLatencyHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rag_query_latency_seconds",
			Help:    "Latency of RAG queries in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"intent_type", "enhancement_enabled", "reranking_enabled"},
	)

	rm.queryCountCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rag_queries_total",
			Help: "Total number of RAG queries processed",
		},
		[]string{"intent_type", "status"},
	)

	// Embedding metrics
	rm.embeddingLatencyHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rag_embedding_latency_seconds",
			Help:    "Latency of embedding generation in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"model_name", "batch_size"},
	)

	rm.embeddingCountCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rag_embeddings_total",
			Help: "Total number of embeddings generated",
		},
		[]string{"model_name", "status"},
	)

	// Cache metrics
	rm.cacheHitRateGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rag_cache_hit_rate",
			Help: "Cache hit rate for RAG components",
		},
		[]string{"cache_type"},
	)

	// Document processing metrics
	rm.documentProcessingGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "rag_documents_processing",
			Help: "Number of documents currently being processed",
		},
	)

	// Component latency metrics
	rm.chunkingLatencyHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rag_chunking_latency_seconds",
			Help:    "Latency of document chunking in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"strategy"},
	)

	rm.retrievalLatencyHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rag_retrieval_latency_seconds",
			Help:    "Latency of document retrieval in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"search_type"},
	)

	rm.contextAssemblyHist = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rag_context_assembly_latency_seconds",
			Help:    "Latency of context assembly in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"strategy"},
	)

	// Error metrics
	rm.errorRateCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rag_errors_total",
			Help: "Total number of errors in RAG components",
		},
		[]string{"component", "error_type"},
	)

	// System resource metrics
	resourceMetrics := []string{
		"cpu_usage_percent",
		"memory_usage_percent",
		"disk_usage_percent",
		"goroutine_count",
		"heap_size_mb",
	}

	for _, metric := range resourceMetrics {
		rm.systemResourceGauges[metric] = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: fmt.Sprintf("rag_system_%s", metric),
				Help: fmt.Sprintf("System %s metric", metric),
			},
		)
	}

	// Register all metrics
	prometheus.MustRegister(
		rm.queryLatencyHistogram,
		rm.queryCountCounter,
		rm.embeddingLatencyHist,
		rm.embeddingCountCounter,
		rm.cacheHitRateGauge,
		rm.documentProcessingGauge,
		rm.chunkingLatencyHist,
		rm.retrievalLatencyHist,
		rm.contextAssemblyHist,
		rm.errorRateCounter,
	)

	for _, gauge := range rm.systemResourceGauges {
		prometheus.MustRegister(gauge)
	}
}

// startMetricsServer starts the HTTP server for metrics and health checks
func (rm *RAGMonitor) startMetricsServer() error {
	mux := http.NewServeMux()

	// Metrics endpoint
	mux.Handle(rm.config.MetricsPath, promhttp.Handler())

	// Health check endpoint
	mux.HandleFunc(rm.config.HealthCheckPath, rm.handleHealthCheck)

	// System status endpoint
	mux.HandleFunc("/status", rm.handleSystemStatus)

	// Component metrics endpoint
	mux.HandleFunc("/metrics/components", rm.handleComponentMetrics)

	rm.metricsServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", rm.config.MetricsPort),
		Handler: mux,
	}

	go func() {
		rm.logger.Info("Starting metrics server", "port", rm.config.MetricsPort)
		if err := rm.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			rm.logger.Error("Metrics server failed", "error", err)
		}
	}()

	return nil
}

// RegisterHealthChecker registers a health checker for a component
func (rm *RAGMonitor) RegisterHealthChecker(checker HealthChecker) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	rm.healthCheckers[checker.GetComponentName()] = checker
	rm.logger.Info("Health checker registered", "component", checker.GetComponentName())
}

// startHealthChecking starts background health checking
func (rm *RAGMonitor) startHealthChecking() {
	ticker := time.NewTicker(rm.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		rm.performHealthChecks(context.Background())
	}
}

// performHealthChecks performs health checks on all registered components
func (rm *RAGMonitor) performHealthChecks(ctx context.Context) {
	rm.mutex.RLock()
	checkers := make(map[string]HealthChecker)
	for name, checker := range rm.healthCheckers {
		checkers[name] = checker
	}
	rm.mutex.RUnlock()

	for name, checker := range checkers {
		go func(name string, checker HealthChecker) {
			defer func() {
				if r := recover(); r != nil {
					rm.logger.Error("Health check panic", "component", name, "panic", r)
				}
			}()

			startTime := time.Now()
			status := checker.CheckHealth(ctx)
			status.ResponseTime = time.Since(startTime)
			status.LastCheck = time.Now()

			if !status.IsHealthy {
				rm.logger.Warn("Component unhealthy",
					"component", name,
					"status", status.Status,
					"error", status.ErrorMessage,
				)

				// Trigger alert if alerting is enabled
				if rm.config.EnableAlerting {
					rm.alertManager.TriggerAlert(Alert{
						ID:         fmt.Sprintf("health_%s_%d", name, time.Now().Unix()),
						MetricName: "component_health",
						Severity:   "critical",
						Message:    fmt.Sprintf("Component %s is unhealthy: %s", name, status.ErrorMessage),
						StartTime:  time.Now(),
						LastSeen:   time.Now(),
						Metadata: map[string]interface{}{
							"component": name,
							"status":    status.Status,
						},
					})
				}
			}
		}(name, checker)
	}
}

// startMetricsCollection starts background metrics collection
func (rm *RAGMonitor) startMetricsCollection() {
	ticker := time.NewTicker(rm.config.MetricsInterval)
	defer ticker.Stop()

	for range ticker.C {
		rm.collectAndUpdateMetrics()
	}
}

// collectAndUpdateMetrics collects metrics from all components and updates Prometheus metrics
func (rm *RAGMonitor) collectAndUpdateMetrics() {
	// This would collect metrics from registered components
	// For now, we'll update based on component health

	rm.mutex.RLock()
	checkers := make(map[string]HealthChecker)
	for name, checker := range rm.healthCheckers {
		checkers[name] = checker
	}
	rm.mutex.RUnlock()

	healthyComponents := 0
	totalComponents := len(checkers)

	for _, checker := range checkers {
		status := checker.CheckHealth(context.Background())
		if status.IsHealthy {
			healthyComponents++
		}
	}

	// Update system health metrics
	if totalComponents > 0 {
		healthRate := float64(healthyComponents) / float64(totalComponents)
		// In a real implementation, you would update specific metrics here
		rm.logger.Debug("System health rate", "rate", healthRate)
	}
}

// startResourceMonitoring starts system resource monitoring
func (rm *RAGMonitor) startResourceMonitoring() {
	ticker := time.NewTicker(rm.config.ResourceMonitoringInterval)
	defer ticker.Stop()

	for range ticker.C {
		metrics := rm.collectSystemMetrics()
		rm.updateSystemResourceMetrics(metrics)
	}
}

// collectSystemMetrics collects system resource metrics
func (rm *RAGMonitor) collectSystemMetrics() SystemMetrics {
	// In a real implementation, you would collect actual system metrics
	// For now, return mock data
	return SystemMetrics{
		CPUUsagePercent:    50.0,
		MemoryUsagePercent: 60.0,
		DiskUsagePercent:   70.0,
		NetworkBytesIn:     1024000,
		NetworkBytesOut:    512000,
		GoroutineCount:     100,
		HeapSizeMB:         128.5,
	}
}

// updateSystemResourceMetrics updates Prometheus metrics with system resource data
func (rm *RAGMonitor) updateSystemResourceMetrics(metrics SystemMetrics) {
	rm.systemResourceGauges["cpu_usage_percent"].Set(metrics.CPUUsagePercent)
	rm.systemResourceGauges["memory_usage_percent"].Set(metrics.MemoryUsagePercent)
	rm.systemResourceGauges["disk_usage_percent"].Set(metrics.DiskUsagePercent)
	rm.systemResourceGauges["goroutine_count"].Set(float64(metrics.GoroutineCount))
	rm.systemResourceGauges["heap_size_mb"].Set(metrics.HeapSizeMB)
}

// HTTP handlers

// handleHealthCheck handles health check requests
func (rm *RAGMonitor) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rm.mutex.RLock()
	checkers := make(map[string]HealthChecker)
	for name, checker := range rm.healthCheckers {
		checkers[name] = checker
	}
	rm.mutex.RUnlock()

	systemHealth := SystemHealth{
		Components:    make(map[string]HealthStatus),
		SystemMetrics: rm.collectSystemMetrics(),
		LastCheck:     time.Now(),
		UptimeSeconds: int64(time.Since(time.Now()).Seconds()), // Placeholder
	}

	allHealthy := true
	for name, checker := range checkers {
		status := checker.CheckHealth(ctx)
		systemHealth.Components[name] = status
		if !status.IsHealthy {
			allHealthy = false
		}
	}

	systemHealth.IsHealthy = allHealthy
	if allHealthy {
		systemHealth.Status = "healthy"
	} else {
		systemHealth.Status = "unhealthy"
	}

	w.Header().Set("Content-Type", "application/json")
	if !allHealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(systemHealth)
}

// handleSystemStatus handles system status requests
func (rm *RAGMonitor) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":        "nephoran-rag-pipeline",
		"version":        "1.0.0",
		"build_time":     time.Now().Format(time.RFC3339),
		"uptime_seconds": int64(time.Since(time.Now()).Seconds()),
		"monitoring": map[string]interface{}{
			"metrics_enabled":     true,
			"health_checks":       len(rm.healthCheckers),
			"alerting_enabled":    rm.config.EnableAlerting,
			"resource_monitoring": rm.config.EnableResourceMonitoring,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleComponentMetrics handles component-specific metrics requests
func (rm *RAGMonitor) handleComponentMetrics(w http.ResponseWriter, r *http.Request) {
	componentMetrics := make(map[string]interface{})

	// Collect metrics from components
	// This would be implemented based on actual component interfaces

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(componentMetrics)
}

// Metric recording methods

// RecordQueryLatency records query processing latency
func (rm *RAGMonitor) RecordQueryLatency(duration time.Duration, intentType string, enhancement, reranking bool) {
	rm.queryLatencyHistogram.WithLabelValues(
		intentType,
		fmt.Sprintf("%t", enhancement),
		fmt.Sprintf("%t", reranking),
	).Observe(duration.Seconds())

	rm.queryCountCounter.WithLabelValues(intentType, "success").Inc()
}

// RecordQueryError records query processing errors
func (rm *RAGMonitor) RecordQueryError(intentType, errorType string) {
	rm.queryCountCounter.WithLabelValues(intentType, "error").Inc()
	rm.errorRateCounter.WithLabelValues("query_processing", errorType).Inc()
}

// RecordEmbeddingLatency records embedding generation latency
func (rm *RAGMonitor) RecordEmbeddingLatency(duration time.Duration, modelName string, batchSize int) {
	rm.embeddingLatencyHist.WithLabelValues(
		modelName,
		fmt.Sprintf("%d", batchSize),
	).Observe(duration.Seconds())

	rm.embeddingCountCounter.WithLabelValues(modelName, "success").Inc()
}

// RecordCacheHitRate records cache hit rates
func (rm *RAGMonitor) RecordCacheHitRate(cacheType string, hitRate float64) {
	rm.cacheHitRateGauge.WithLabelValues(cacheType).Set(hitRate)
}

// RecordDocumentProcessing records document processing metrics
func (rm *RAGMonitor) RecordDocumentProcessing(count int) {
	rm.documentProcessingGauge.Set(float64(count))
}

// RecordChunkingLatency records document chunking latency
func (rm *RAGMonitor) RecordChunkingLatency(duration time.Duration, strategy string) {
	rm.chunkingLatencyHist.WithLabelValues(strategy).Observe(duration.Seconds())
}

// RecordRetrievalLatency records document retrieval latency
func (rm *RAGMonitor) RecordRetrievalLatency(duration time.Duration, searchType string) {
	rm.retrievalLatencyHist.WithLabelValues(searchType).Observe(duration.Seconds())
}

// RecordContextAssemblyLatency records context assembly latency
func (rm *RAGMonitor) RecordContextAssemblyLatency(duration time.Duration, strategy string) {
	rm.contextAssemblyHist.WithLabelValues(strategy).Observe(duration.Seconds())
}

// Alert Manager implementation

// NewAlertManager creates a new alert manager
func NewAlertManager(config *MonitoringConfig) *AlertManager {
	return &AlertManager{
		config:   config,
		logger:   slog.Default().With("component", "alert-manager"),
		alerts:   make(map[string]Alert),
		webhooks: config.AlertWebhooks,
	}
}

// TriggerAlert triggers an alert
func (am *AlertManager) TriggerAlert(alert Alert) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.alerts[alert.ID] = alert

	am.logger.Warn("Alert triggered",
		"id", alert.ID,
		"severity", alert.Severity,
		"message", alert.Message,
		"metric", alert.MetricName,
	)

	// Send webhook notifications
	for _, webhook := range am.webhooks {
		go am.sendWebhookNotification(webhook, alert)
	}
}

// sendWebhookNotification sends alert notification to webhook
func (am *AlertManager) sendWebhookNotification(webhook string, alert Alert) {
	// In a real implementation, you would send HTTP POST to the webhook
	am.logger.Info("Would send webhook notification",
		"webhook", webhook,
		"alert_id", alert.ID,
		"severity", alert.Severity,
	)
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []Alert {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	alerts := make([]Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		alerts = append(alerts, alert)
	}

	return alerts
}

// ClearAlert clears an alert
func (am *AlertManager) ClearAlert(alertID string) {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	delete(am.alerts, alertID)
	am.logger.Info("Alert cleared", "id", alertID)
}

// Shutdown gracefully shuts down the monitoring system
func (rm *RAGMonitor) Shutdown(ctx context.Context) error {
	rm.logger.Info("Shutting down RAG monitor")

	if rm.metricsServer != nil {
		if err := rm.metricsServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown metrics server: %w", err)
		}
	}

	return nil
}

// Health checker implementations for RAG components

// RAGServiceHealthChecker implements health checking for RAGService
type RAGServiceHealthChecker struct {
	ragService *RAGService
}

func (hc *RAGServiceHealthChecker) CheckHealth(ctx context.Context) HealthStatus {
	status := HealthStatus{
		ComponentName: "rag-service",
		IsHealthy:     true,
		Status:        "healthy",
		LastCheck:     time.Now(),
	}

	// Check if the service is operational
	if hc.ragService == nil {
		status.IsHealthy = false
		status.Status = "unhealthy"
		status.ErrorMessage = "RAG service is nil"
		return status
	}

	// Get service metrics and health
	health := hc.ragService.GetHealth()
	if healthStatus, ok := health["status"].(string); ok && healthStatus != "healthy" {
		status.IsHealthy = false
		status.Status = healthStatus
		status.ErrorMessage = "RAG service reported unhealthy status"
	}

	status.Metadata = health
	return status
}

func (hc *RAGServiceHealthChecker) GetComponentName() string {
	return "rag-service"
}

// EmbeddingServiceHealthChecker implements health checking for EmbeddingService
type EmbeddingServiceHealthChecker struct {
	embeddingService *EmbeddingService
}

func (hc *EmbeddingServiceHealthChecker) CheckHealth(ctx context.Context) HealthStatus {
	status := HealthStatus{
		ComponentName: "embedding-service",
		IsHealthy:     true,
		Status:        "healthy",
		LastCheck:     time.Now(),
	}

	if hc.embeddingService == nil {
		status.IsHealthy = false
		status.Status = "unhealthy"
		status.ErrorMessage = "Embedding service is nil"
		return status
	}

	// Get service metrics
	metrics := hc.embeddingService.GetMetrics()
	status.Metadata = map[string]interface{}{
		"total_requests":      metrics.TotalRequests,
		"successful_requests": metrics.SuccessfulRequests,
		"failed_requests":     metrics.FailedRequests,
		"average_latency":     metrics.AverageLatency.String(),
		"cache_hit_rate":      metrics.CacheStats.HitRate,
	}

	// Check error rate
	if metrics.TotalRequests > 0 {
		errorRate := float64(metrics.FailedRequests) / float64(metrics.TotalRequests)
		if errorRate > 0.1 { // 10% error rate threshold
			status.IsHealthy = false
			status.Status = "degraded"
			status.ErrorMessage = fmt.Sprintf("High error rate: %.2f%%", errorRate*100)
		}
	}

	return status
}

func (hc *EmbeddingServiceHealthChecker) GetComponentName() string {
	return "embedding-service"
}

// RedisHealthChecker implements health checking for Redis cache
type RedisHealthChecker struct {
	redisCache *RedisCache
}

func (hc *RedisHealthChecker) CheckHealth(ctx context.Context) HealthStatus {
	status := HealthStatus{
		ComponentName: "redis-cache",
		IsHealthy:     true,
		Status:        "healthy",
		LastCheck:     time.Now(),
	}

	if hc.redisCache == nil {
		status.IsHealthy = false
		status.Status = "unhealthy"
		status.ErrorMessage = "Redis cache is nil"
		return status
	}

	// Get Redis health status
	health := hc.redisCache.GetHealthStatus(ctx)
	if healthy, ok := health["healthy"].(bool); ok && !healthy {
		status.IsHealthy = false
		status.Status = "unhealthy"
		if errMsg, ok := health["error"].(string); ok {
			status.ErrorMessage = errMsg
		}
	}

	status.Metadata = health
	return status
}

func (hc *RedisHealthChecker) GetComponentName() string {
	return "redis-cache"
}
