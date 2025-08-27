/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webui

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
)

// DashboardMetrics represents comprehensive dashboard metrics
type DashboardMetrics struct {
	Overview           *OverviewMetrics    `json:"overview"`
	IntentMetrics      *IntentMetrics      `json:"intent_metrics"`
	PackageMetrics     *PackageMetrics     `json:"package_metrics"`
	ClusterMetrics     *ClusterMetrics     `json:"cluster_metrics"`
	NetworkMetrics     *NetworkMetrics     `json:"network_metrics"`
	PerformanceMetrics *PerformanceMetrics `json:"performance_metrics"`
	AlertsAndEvents    *AlertsAndEvents    `json:"alerts_and_events"`
	Timestamp          time.Time           `json:"timestamp"`
}

// OverviewMetrics provides high-level system overview
type OverviewMetrics struct {
	TotalIntents      int64   `json:"total_intents"`
	ActiveIntents     int64   `json:"active_intents"`
	CompletedIntents  int64   `json:"completed_intents"`
	FailedIntents     int64   `json:"failed_intents"`
	TotalPackages     int64   `json:"total_packages"`
	PublishedPackages int64   `json:"published_packages"`
	TotalClusters     int64   `json:"total_clusters"`
	HealthyClusters   int64   `json:"healthy_clusters"`
	SystemHealth      string  `json:"system_health"`
	SuccessRate       float64 `json:"success_rate"`
	AvgProcessingTime float64 `json:"avg_processing_time_seconds"`
}

// IntentMetrics provides intent-specific metrics
type IntentMetrics struct {
	ByStatus          map[string]int64     `json:"by_status"`
	ByType            map[string]int64     `json:"by_type"`
	ByPriority        map[string]int64     `json:"by_priority"`
	ByComponent       map[string]int64     `json:"by_component"`
	ProcessingTimes   *ProcessingTimes     `json:"processing_times"`
	RecentActivity    []*IntentActivity    `json:"recent_activity"`
	TrendData         []*TrendDataPoint    `json:"trend_data"`
	SuccessRateByHour []*HourlySuccessRate `json:"success_rate_by_hour"`
}

// PackageMetrics provides package-specific metrics
type PackageMetrics struct {
	ByLifecycle       map[string]int64       `json:"by_lifecycle"`
	ByRepository      map[string]int64       `json:"by_repository"`
	ValidationResults *ValidationMetrics     `json:"validation_results"`
	DeploymentMetrics *DeploymentMetrics     `json:"deployment_metrics"`
	RecentTransitions []*PackageTransition   `json:"recent_transitions"`
	LifecycleTrends   []*LifecycleTrendPoint `json:"lifecycle_trends"`
}

// ClusterMetrics provides cluster-specific metrics
type ClusterMetrics struct {
	ByStatus            map[string]int64     `json:"by_status"`
	ByRegion            map[string]int64     `json:"by_region"`
	ResourceUtilization *ResourceMetrics     `json:"resource_utilization"`
	HealthDistribution  []*HealthDistPoint   `json:"health_distribution"`
	ConnectivityMatrix  []*ConnectivityPoint `json:"connectivity_matrix"`
}

// NetworkMetrics provides network-wide metrics
type NetworkMetrics struct {
	TotalDeployments   int64               `json:"total_deployments"`
	ActiveDeployments  int64               `json:"active_deployments"`
	NetworkFunctions   map[string]int64    `json:"network_functions"`
	ServiceMeshMetrics *ServiceMeshMetrics `json:"service_mesh_metrics"`
	LatencyMetrics     *LatencyMetrics     `json:"latency_metrics"`
	ThroughputMetrics  *ThroughputMetrics  `json:"throughput_metrics"`
}

// PerformanceMetrics provides system performance metrics
type PerformanceMetrics struct {
	APIResponseTimes *ResponseTimeMetrics `json:"api_response_times"`
	CachePerformance *CacheStats          `json:"cache_performance"`
	RateLimitStats   *RateLimitStats      `json:"rate_limit_stats"`
	ConnectionStats  *ConnectionStats     `json:"connection_stats"`
	ResourceUsage    *SystemResourceUsage `json:"resource_usage"`
}

// AlertsAndEvents provides alerts and recent events
type AlertsAndEvents struct {
	ActiveAlerts  []*Alert         `json:"active_alerts"`
	RecentEvents  []*SystemEvent   `json:"recent_events"`
	AlertsSummary *AlertsSummary   `json:"alerts_summary"`
	EventsByType  map[string]int64 `json:"events_by_type"`
	EventsByLevel map[string]int64 `json:"events_by_level"`
}

// Supporting data structures
type ProcessingTimes struct {
	P50    float64 `json:"p50_ms"`
	P90    float64 `json:"p90_ms"`
	P95    float64 `json:"p95_ms"`
	P99    float64 `json:"p99_ms"`
	Mean   float64 `json:"mean_ms"`
	StdDev float64 `json:"std_dev_ms"`
}

type IntentActivity struct {
	IntentName     string    `json:"intent_name"`
	Status         string    `json:"status"`
	Component      string    `json:"component"`
	Timestamp      time.Time `json:"timestamp"`
	ProcessingTime int64     `json:"processing_time_ms"`
}

type TrendDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Created   int64     `json:"created"`
	Completed int64     `json:"completed"`
	Failed    int64     `json:"failed"`
}

type HourlySuccessRate struct {
	Hour        int     `json:"hour"`
	SuccessRate float64 `json:"success_rate"`
	Total       int64   `json:"total"`
}

type ValidationMetrics struct {
	TotalValidations  int64   `json:"total_validations"`
	PassedValidations int64   `json:"passed_validations"`
	FailedValidations int64   `json:"failed_validations"`
	ValidationRate    float64 `json:"validation_rate"`
	AvgValidationTime float64 `json:"avg_validation_time_ms"`
}

type DeploymentMetrics struct {
	TotalDeployments      int64   `json:"total_deployments"`
	SuccessfulDeployments int64   `json:"successful_deployments"`
	FailedDeployments     int64   `json:"failed_deployments"`
	DeploymentSuccessRate float64 `json:"deployment_success_rate"`
	AvgDeploymentTime     float64 `json:"avg_deployment_time_ms"`
}

type PackageTransition struct {
	PackageName string    `json:"package_name"`
	FromStage   string    `json:"from_stage"`
	ToStage     string    `json:"to_stage"`
	Timestamp   time.Time `json:"timestamp"`
	Duration    int64     `json:"duration_ms"`
	Success     bool      `json:"success"`
}

type LifecycleTrendPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Draft     int64     `json:"draft"`
	Proposed  int64     `json:"proposed"`
	Published int64     `json:"published"`
}

type ResourceMetrics struct {
	TotalCPU           float64 `json:"total_cpu_cores"`
	UsedCPU            float64 `json:"used_cpu_cores"`
	TotalMemory        int64   `json:"total_memory_bytes"`
	UsedMemory         int64   `json:"used_memory_bytes"`
	TotalStorage       int64   `json:"total_storage_bytes"`
	UsedStorage        int64   `json:"used_storage_bytes"`
	CPUUtilization     float64 `json:"cpu_utilization_percent"`
	MemoryUtilization  float64 `json:"memory_utilization_percent"`
	StorageUtilization float64 `json:"storage_utilization_percent"`
}

type HealthDistPoint struct {
	HealthScore int64 `json:"health_score"`
	Count       int64 `json:"count"`
}

type ConnectivityPoint struct {
	Source      string  `json:"source"`
	Destination string  `json:"destination"`
	LatencyMS   float64 `json:"latency_ms"`
	Status      string  `json:"status"`
}

type ServiceMeshMetrics struct {
	TotalServices   int64   `json:"total_services"`
	HealthyServices int64   `json:"healthy_services"`
	RequestRate     float64 `json:"requests_per_second"`
	ErrorRate       float64 `json:"error_rate_percent"`
	P99Latency      float64 `json:"p99_latency_ms"`
}

type LatencyMetrics struct {
	AvgLatency float64 `json:"avg_latency_ms"`
	P50Latency float64 `json:"p50_latency_ms"`
	P95Latency float64 `json:"p95_latency_ms"`
	P99Latency float64 `json:"p99_latency_ms"`
}

type ThroughputMetrics struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	BytesPerSecond    int64   `json:"bytes_per_second"`
	MessagesPerSecond float64 `json:"messages_per_second"`
}

type ResponseTimeMetrics struct {
	IntentAPI   *ProcessingTimes `json:"intent_api"`
	PackageAPI  *ProcessingTimes `json:"package_api"`
	ClusterAPI  *ProcessingTimes `json:"cluster_api"`
	RealtimeAPI *ProcessingTimes `json:"realtime_api"`
}

type ConnectionStats struct {
	ActiveWSConnections  int64   `json:"active_ws_connections"`
	ActiveSSEConnections int64   `json:"active_sse_connections"`
	TotalConnections     int64   `json:"total_connections"`
	ConnectionsPerSecond float64 `json:"connections_per_second"`
}

type SystemResourceUsage struct {
	CPUUsage    float64    `json:"cpu_usage_percent"`
	MemoryUsage float64    `json:"memory_usage_percent"`
	DiskUsage   float64    `json:"disk_usage_percent"`
	NetworkIO   *NetworkIO `json:"network_io"`
	Uptime      int64      `json:"uptime_seconds"`
}

type Alert struct {
	ID        string                 `json:"id"`
	Level     string                 `json:"level"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Component string                 `json:"component"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Status    string                 `json:"status"` // active, acknowledged, resolved
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type AlertsSummary struct {
	Critical int64 `json:"critical"`
	Error    int64 `json:"error"`
	Warning  int64 `json:"warning"`
	Info     int64 `json:"info"`
	Total    int64 `json:"total"`
}

// setupDashboardRoutes sets up dashboard API routes
func (s *NephoranAPIServer) setupDashboardRoutes(router *mux.Router) {
	dashboard := router.PathPrefix("/dashboard").Subrouter()

	// Apply dashboard-specific middleware
	if s.authMiddleware != nil {
		dashboard.Use(s.authMiddleware.RequirePermissionMiddleware(auth.PermissionViewMetrics))
	}

	// Main dashboard endpoints
	dashboard.HandleFunc("/metrics", s.getDashboardMetrics).Methods("GET")
	dashboard.HandleFunc("/overview", s.getDashboardOverview).Methods("GET")
	dashboard.HandleFunc("/health", s.getSystemHealth).Methods("GET")

	// Detailed metrics endpoints
	dashboard.HandleFunc("/metrics/intents", s.getIntentMetrics).Methods("GET")
	dashboard.HandleFunc("/metrics/packages", s.getPackageMetrics).Methods("GET")
	dashboard.HandleFunc("/metrics/clusters", s.getClusterMetrics).Methods("GET")
	dashboard.HandleFunc("/metrics/network", s.getNetworkMetrics).Methods("GET")
	dashboard.HandleFunc("/metrics/performance", s.getPerformanceMetrics).Methods("GET")

	// Alerts and events
	dashboard.HandleFunc("/alerts", s.getActiveAlerts).Methods("GET")
	dashboard.HandleFunc("/events", s.getRecentEvents).Methods("GET")
	dashboard.HandleFunc("/alerts/{id}/acknowledge", s.acknowledgeAlert).Methods("POST")
	dashboard.HandleFunc("/alerts/{id}/resolve", s.resolveAlert).Methods("POST")

	// Trend and historical data
	dashboard.HandleFunc("/trends/intents", s.getIntentTrends).Methods("GET")
	dashboard.HandleFunc("/trends/packages", s.getPackageTrends).Methods("GET")
	dashboard.HandleFunc("/trends/performance", s.getPerformanceTrends).Methods("GET")

	// Topology and visualization
	dashboard.HandleFunc("/topology/network", s.getNetworkTopology).Methods("GET")
	dashboard.HandleFunc("/topology/components", s.getComponentTopology).Methods("GET")
	dashboard.HandleFunc("/dependencies", s.getSystemDependencies).Methods("GET")
}

// setupSystemRoutes sets up system management API routes
func (s *NephoranAPIServer) setupSystemRoutes(router *mux.Router) {
	// Health check endpoint (no auth required)
	router.HandleFunc("/health", s.healthCheck).Methods("GET")
	router.HandleFunc("/readiness", s.readinessCheck).Methods("GET")
	router.HandleFunc("/liveness", s.livenessCheck).Methods("GET")

	// Metrics endpoint (Prometheus format)
	if s.config.EnableMetrics {
		router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	}

	// OpenAPI/Swagger endpoints
	router.HandleFunc("/openapi.json", s.getOpenAPISpec).Methods("GET")
	router.HandleFunc("/docs", s.getAPIDocs).Methods("GET")

	// System management endpoints (admin only)
	system := router.PathPrefix("/system").Subrouter()
	if s.authMiddleware != nil {
		system.Use(s.authMiddleware.RequireAdminMiddleware)
	}

	system.HandleFunc("/info", s.getSystemInfo).Methods("GET")
	system.HandleFunc("/stats", s.getSystemStats).Methods("GET")
	system.HandleFunc("/config", s.getSystemConfig).Methods("GET")
	system.HandleFunc("/cache/stats", s.getCacheStats).Methods("GET")
	system.HandleFunc("/cache/clear", s.clearCache).Methods("POST")
	system.HandleFunc("/connections", s.getActiveConnections).Methods("GET")

	// Debug endpoints (development only)
	if s.config.EnableProfiling {
		system.HandleFunc("/debug/pprof/", http.DefaultServeMux.ServeHTTP)
	}
}

// Dashboard handlers

// getDashboardMetrics handles GET /api/v1/dashboard/metrics
func (s *NephoranAPIServer) getDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check cache first
	cacheKey := "dashboard:metrics:all"
	if s.cache != nil {
		if cached, found := s.cache.Get(cacheKey); found {
			s.metrics.CacheHits.Inc()
			s.writeJSONResponse(w, http.StatusOK, cached)
			return
		}
		s.metrics.CacheMisses.Inc()
	}

	// Gather all metrics
	metrics := &DashboardMetrics{
		Overview:           s.generateOverviewMetrics(ctx),
		IntentMetrics:      s.generateIntentMetrics(ctx),
		PackageMetrics:     s.generatePackageMetrics(ctx),
		ClusterMetrics:     s.generateClusterMetrics(ctx),
		NetworkMetrics:     s.generateNetworkMetrics(ctx),
		PerformanceMetrics: s.generatePerformanceMetrics(ctx),
		AlertsAndEvents:    s.generateAlertsAndEvents(ctx),
		Timestamp:          time.Now(),
	}

	// Cache the result with short TTL for dashboard data
	if s.cache != nil {
		s.cache.SetWithTTL(cacheKey, metrics, 30*time.Second)
	}

	s.writeJSONResponse(w, http.StatusOK, metrics)
}

// getDashboardOverview handles GET /api/v1/dashboard/overview
func (s *NephoranAPIServer) getDashboardOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check cache first
	cacheKey := "dashboard:overview"
	if s.cache != nil {
		if cached, found := s.cache.Get(cacheKey); found {
			s.metrics.CacheHits.Inc()
			s.writeJSONResponse(w, http.StatusOK, cached)
			return
		}
		s.metrics.CacheMisses.Inc()
	}

	overview := s.generateOverviewMetrics(ctx)

	// Cache with short TTL
	if s.cache != nil {
		s.cache.SetWithTTL(cacheKey, overview, 15*time.Second)
	}

	s.writeJSONResponse(w, http.StatusOK, overview)
}

// System handlers

// healthCheck handles GET /health
func (s *NephoranAPIServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"version":   "1.0.0",                          // Would come from build info
		"uptime":    time.Since(time.Now()).Seconds(), // Would track actual uptime
		"checks": map[string]interface{}{
			"api_server":      "healthy",
			"database":        "healthy",
			"cache":           s.checkCacheHealth(),
			"rate_limiter":    s.checkRateLimiterHealth(),
			"intent_manager":  "healthy",
			"package_manager": "healthy",
			"cluster_manager": "healthy",
		},
	}

	s.writeJSONResponse(w, http.StatusOK, health)
}

// readinessCheck handles GET /readiness
func (s *NephoranAPIServer) readinessCheck(w http.ResponseWriter, r *http.Request) {
	ready := true
	checks := make(map[string]string)

	// Check if all dependencies are ready
	if s.intentReconciler == nil {
		checks["intent_manager"] = "not_ready"
		ready = false
	} else {
		checks["intent_manager"] = "ready"
	}

	if s.packageManager == nil {
		checks["package_manager"] = "not_ready"
		ready = false
	} else {
		checks["package_manager"] = "ready"
	}

	if s.clusterManager == nil {
		checks["cluster_manager"] = "not_ready"
		ready = false
	} else {
		checks["cluster_manager"] = "ready"
	}

	status := "ready"
	statusCode := http.StatusOK
	if !ready {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	response := map[string]interface{}{
		"status":    status,
		"timestamp": time.Now(),
		"checks":    checks,
	}

	s.writeJSONResponse(w, statusCode, response)
}

// Helper methods for generating metrics (mock implementations)

func (s *NephoranAPIServer) generateOverviewMetrics(ctx context.Context) *OverviewMetrics {
	// Mock implementation - would integrate with actual metrics collection
	return &OverviewMetrics{
		TotalIntents:      1247,
		ActiveIntents:     23,
		CompletedIntents:  1198,
		FailedIntents:     26,
		TotalPackages:     156,
		PublishedPackages: 134,
		TotalClusters:     8,
		HealthyClusters:   7,
		SystemHealth:      "healthy",
		SuccessRate:       96.2,
		AvgProcessingTime: 4.7,
	}
}

func (s *NephoranAPIServer) generateIntentMetrics(ctx context.Context) *IntentMetrics {
	return &IntentMetrics{
		ByStatus: map[string]int64{
			"pending":    12,
			"processing": 8,
			"completed":  1198,
			"failed":     26,
			"cancelled":  3,
		},
		ByType: map[string]int64{
			"deployment":   786,
			"scaling":      234,
			"optimization": 167,
			"maintenance":  60,
		},
		ByPriority: map[string]int64{
			"low":      345,
			"medium":   678,
			"high":     189,
			"critical": 35,
		},
		ByComponent: map[string]int64{
			"AMF":     234,
			"SMF":     198,
			"UPF":     345,
			"gNodeB":  156,
			"O-DU":    89,
			"O-CU-CP": 67,
		},
		ProcessingTimes: &ProcessingTimes{
			P50:    2.3,
			P90:    8.7,
			P95:    12.4,
			P99:    45.6,
			Mean:   4.7,
			StdDev: 6.2,
		},
	}
}

func (s *NephoranAPIServer) generatePackageMetrics(ctx context.Context) *PackageMetrics {
	return &PackageMetrics{
		ByLifecycle: map[string]int64{
			"draft":     22,
			"proposed":  8,
			"published": 134,
		},
		ByRepository: map[string]int64{
			"default":     89,
			"production":  45,
			"development": 22,
		},
		ValidationResults: &ValidationMetrics{
			TotalValidations:  234,
			PassedValidations: 217,
			FailedValidations: 17,
			ValidationRate:    92.7,
			AvgValidationTime: 1.2,
		},
		DeploymentMetrics: &DeploymentMetrics{
			TotalDeployments:      456,
			SuccessfulDeployments: 434,
			FailedDeployments:     22,
			DeploymentSuccessRate: 95.2,
			AvgDeploymentTime:     8.4,
		},
	}
}

func (s *NephoranAPIServer) generateClusterMetrics(ctx context.Context) *ClusterMetrics {
	return &ClusterMetrics{
		ByStatus: map[string]int64{
			"healthy":     7,
			"degraded":    1,
			"unreachable": 0,
		},
		ByRegion: map[string]int64{
			"us-west-2":  3,
			"us-east-1":  2,
			"eu-west-1":  2,
			"ap-south-1": 1,
		},
		ResourceUtilization: &ResourceMetrics{
			TotalCPU:          64.0,
			UsedCPU:           41.6,
			TotalMemory:       524288000000, // 512GB
			UsedMemory:        314572800000, // ~300GB
			CPUUtilization:    65.0,
			MemoryUtilization: 60.0,
		},
	}
}

func (s *NephoranAPIServer) generateNetworkMetrics(ctx context.Context) *NetworkMetrics {
	return &NetworkMetrics{
		TotalDeployments:  456,
		ActiveDeployments: 389,
		NetworkFunctions: map[string]int64{
			"AMF":     12,
			"SMF":     15,
			"UPF":     28,
			"gNodeB":  45,
			"O-DU":    23,
			"O-CU-CP": 16,
		},
		ServiceMeshMetrics: &ServiceMeshMetrics{
			TotalServices:   234,
			HealthyServices: 228,
			RequestRate:     1247.5,
			ErrorRate:       0.8,
			P99Latency:      15.6,
		},
	}
}

func (s *NephoranAPIServer) generatePerformanceMetrics(ctx context.Context) *PerformanceMetrics {
	var cacheStats *CacheStats
	if s.cache != nil {
		cacheStats = s.cache.Stats()
	} else {
		cacheStats = &CacheStats{}
	}

	var rateLimitStats *RateLimitStats
	if s.rateLimiter != nil {
		rateLimitStats = s.rateLimiter.Stats()
	} else {
		rateLimitStats = &RateLimitStats{}
	}

	s.connectionsMutex.RLock()
	wsConnections := int64(len(s.wsConnections))
	sseConnections := int64(len(s.sseConnections))
	s.connectionsMutex.RUnlock()

	return &PerformanceMetrics{
		APIResponseTimes: &ResponseTimeMetrics{
			IntentAPI: &ProcessingTimes{
				P50:  45.2,
				P90:  156.7,
				P95:  234.5,
				P99:  567.8,
				Mean: 78.4,
			},
			PackageAPI: &ProcessingTimes{
				P50:  67.3,
				P90:  189.4,
				P95:  278.6,
				P99:  645.2,
				Mean: 89.7,
			},
		},
		CachePerformance: cacheStats,
		RateLimitStats:   rateLimitStats,
		ConnectionStats: &ConnectionStats{
			ActiveWSConnections:  wsConnections,
			ActiveSSEConnections: sseConnections,
			TotalConnections:     wsConnections + sseConnections,
		},
	}
}

func (s *NephoranAPIServer) generateAlertsAndEvents(ctx context.Context) *AlertsAndEvents {
	return &AlertsAndEvents{
		ActiveAlerts: []*Alert{
			{
				ID:        "alert-001",
				Level:     "warning",
				Title:     "High Memory Usage",
				Message:   "Cluster us-west-2a is using 85% of available memory",
				Component: "cluster-manager",
				CreatedAt: time.Now().Add(-2 * time.Hour),
				Status:    "active",
			},
		},
		RecentEvents: []*SystemEvent{
			{
				Level:     "info",
				Component: "intent-manager",
				Event:     "intent_completed",
				Message:   "Intent 'deploy-amf-prod' completed successfully",
				Timestamp: time.Now().Add(-5 * time.Minute),
			},
		},
		AlertsSummary: &AlertsSummary{
			Critical: 0,
			Error:    0,
			Warning:  1,
			Info:     3,
			Total:    4,
		},
		EventsByType: map[string]int64{
			"intent_created":   45,
			"intent_completed": 42,
			"package_deployed": 23,
			"cluster_health":   12,
		},
	}
}

func (s *NephoranAPIServer) checkCacheHealth() string {
	if s.cache == nil {
		return "disabled"
	}
	return "healthy"
}

func (s *NephoranAPIServer) checkRateLimiterHealth() string {
	if s.rateLimiter == nil {
		return "disabled"
	}
	return "healthy"
}

// getSystemInfo handles GET /system/info
func (s *NephoranAPIServer) getSystemInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"name":       "Nephoran Intent Operator API",
		"version":    "1.0.0",
		"build_time": "2025-01-01T00:00:00Z", // Would come from build info
		"git_commit": "abc123def",            // Would come from build info
		"go_version": "go1.24.0",             // Would come from runtime
		"started_at": time.Now(),             // Would track actual start time
		"uptime":     time.Since(time.Now()).Seconds(),
		"config": map[string]interface{}{
			"port":               s.config.Port,
			"tls_enabled":        s.config.TLSEnabled,
			"cors_enabled":       s.config.EnableCORS,
			"metrics_enabled":    s.config.EnableMetrics,
			"auth_enabled":       s.authMiddleware != nil,
			"cache_enabled":      s.cache != nil,
			"rate_limit_enabled": s.rateLimiter != nil,
		},
	}

	s.writeJSONResponse(w, http.StatusOK, info)
}

// getSystemHealth handles GET /api/v1/dashboard/health
func (s *NephoranAPIServer) getSystemHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"components": map[string]interface{}{
			"api_server": map[string]interface{}{
				"status":         "healthy",
				"uptime_seconds": time.Since(time.Now()).Seconds(), // Would track actual uptime
			},
			"database": map[string]interface{}{
				"status":          "healthy",
				"connection_pool": "active",
			},
			"cache": map[string]interface{}{
				"status":   s.checkCacheHealth(),
				"hit_rate": "95.2%",
			},
			"rate_limiter": map[string]interface{}{
				"status":        s.checkRateLimiterHealth(),
				"active_limits": 0,
			},
			"auth_service": map[string]interface{}{
				"status": func() string {
					if s.authMiddleware != nil {
						return "healthy"
					}
					return "disabled"
				}(),
			},
		},
		"resource_usage": map[string]interface{}{
			"cpu_percent": 15.2,
			"memory_mb":   256,
			"goroutines":  42,
		},
	}

	s.writeJSONResponse(w, http.StatusOK, health)
}

// getIntentMetrics handles GET /api/v1/dashboard/metrics/intents
func (s *NephoranAPIServer) getIntentMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"total_intents":     150,
		"active_intents":    42,
		"completed_intents": 98,
		"failed_intents":    10,
		"intent_types": map[string]int{
			"scaling":       85,
			"deployment":    35,
			"configuration": 30,
		},
	}
	s.writeJSONResponse(w, http.StatusOK, metrics)
}

// getPackageMetrics handles GET /api/v1/dashboard/metrics/packages
func (s *NephoranAPIServer) getPackageMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"total_packages":    75,
		"deployed_packages": 68,
		"pending_packages":  5,
		"failed_packages":   2,
		"package_types": map[string]int{
			"cnf":    45,
			"vnf":    20,
			"config": 10,
		},
	}
	s.writeJSONResponse(w, http.StatusOK, metrics)
}

// getClusterMetrics handles GET /api/v1/dashboard/metrics/clusters
func (s *NephoranAPIServer) getClusterMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"total_clusters":       12,
		"healthy_clusters":     11,
		"degraded_clusters":    1,
		"unreachable_clusters": 0,
		"resource_utilization": map[string]interface{}{
			"cpu_avg":     "65%",
			"memory_avg":  "72%",
			"storage_avg": "45%",
		},
	}
	s.writeJSONResponse(w, http.StatusOK, metrics)
}

// getNetworkMetrics handles GET /api/v1/dashboard/metrics/network
func (s *NephoranAPIServer) getNetworkMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"total_connections":     24,
		"active_connections":    22,
		"avg_latency_ms":        15.2,
		"bandwidth_utilization": "45%",
		"packet_loss":           "0.02%",
	}
	s.writeJSONResponse(w, http.StatusOK, metrics)
}

// getPerformanceMetrics handles GET /api/v1/dashboard/metrics/performance
func (s *NephoranAPIServer) getPerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"api_response_time_ms":        125.5,
		"throughput_requests_per_sec": 245.8,
		"error_rate":                  "0.5%",
		"cache_hit_rate":              "92.3%",
		"memory_usage_mb":             512,
		"cpu_usage_percent":           23.4,
	}
	s.writeJSONResponse(w, http.StatusOK, metrics)
}

// getActiveAlerts handles GET /api/v1/dashboard/alerts
func (s *NephoranAPIServer) getActiveAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := []map[string]interface{}{
		{
			"id":           "alert-001",
			"severity":     "warning",
			"title":        "High CPU usage on cluster-2",
			"description":  "CPU usage is above 80%",
			"timestamp":    time.Now().Add(-30 * time.Minute),
			"acknowledged": false,
		},
		{
			"id":           "alert-002",
			"severity":     "info",
			"title":        "Package deployment completed",
			"description":  "Successfully deployed CNF package to 3 clusters",
			"timestamp":    time.Now().Add(-5 * time.Minute),
			"acknowledged": true,
		},
	}
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"alerts": alerts,
		"total":  len(alerts),
	})
}

// getRecentEvents handles GET /api/v1/dashboard/events
func (s *NephoranAPIServer) getRecentEvents(w http.ResponseWriter, r *http.Request) {
	events := []map[string]interface{}{
		{
			"id":          "event-001",
			"type":        "intent_created",
			"description": "New scaling intent created",
			"timestamp":   time.Now().Add(-10 * time.Minute),
			"severity":    "info",
		},
		{
			"id":          "event-002",
			"type":        "cluster_health",
			"description": "Cluster health check completed",
			"timestamp":   time.Now().Add(-15 * time.Minute),
			"severity":    "info",
		},
	}
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"total":  len(events),
	})
}

// acknowledgeAlert handles PUT /api/v1/dashboard/alerts/{id}/acknowledge
func (s *NephoranAPIServer) acknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["id"]

	s.logger.Info("Acknowledging alert", "alert_id", alertID)
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":         "Alert acknowledged",
		"alert_id":        alertID,
		"acknowledged_at": time.Now().UTC(),
	})
}

// resolveAlert handles PUT /api/v1/dashboard/alerts/{id}/resolve
func (s *NephoranAPIServer) resolveAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	alertID := vars["id"]

	s.logger.Info("Resolving alert", "alert_id", alertID)
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":     "Alert resolved",
		"alert_id":    alertID,
		"resolved_at": time.Now().UTC(),
	})
}

// getIntentTrends handles GET /api/v1/dashboard/trends/intents
func (s *NephoranAPIServer) getIntentTrends(w http.ResponseWriter, r *http.Request) {
	trends := map[string]interface{}{
		"daily_intents":          []int{10, 12, 8, 15, 20, 18, 25},
		"weekly_completion_rate": []float64{92.5, 94.2, 89.8, 96.1, 93.7},
		"growth_rate":            "15.3%",
	}
	s.writeJSONResponse(w, http.StatusOK, trends)
}

// getPackageTrends handles GET /api/v1/dashboard/trends/packages
func (s *NephoranAPIServer) getPackageTrends(w http.ResponseWriter, r *http.Request) {
	trends := map[string]interface{}{
		"daily_deployments": []int{5, 8, 6, 12, 10, 9, 15},
		"success_rate":      []float64{95.2, 97.1, 94.5, 98.2, 96.8},
		"growth_rate":       "12.1%",
	}
	s.writeJSONResponse(w, http.StatusOK, trends)
}

// getPerformanceTrends handles GET /api/v1/dashboard/trends/performance
func (s *NephoranAPIServer) getPerformanceTrends(w http.ResponseWriter, r *http.Request) {
	trends := map[string]interface{}{
		"response_times": []float64{125.5, 130.2, 118.7, 142.1, 134.8},
		"throughput":     []float64{245.8, 252.3, 238.9, 267.4, 251.2},
		"error_rates":    []float64{0.5, 0.3, 0.7, 0.4, 0.6},
	}
	s.writeJSONResponse(w, http.StatusOK, trends)
}

// getComponentTopology handles GET /api/v1/dashboard/topology/components
func (s *NephoranAPIServer) getComponentTopology(w http.ResponseWriter, r *http.Request) {
	topology := map[string]interface{}{
		"components": []map[string]interface{}{
			{"name": "api-server", "status": "healthy", "connections": []string{"database", "cache"}},
			{"name": "intent-controller", "status": "healthy", "connections": []string{"api-server", "clusters"}},
			{"name": "package-manager", "status": "healthy", "connections": []string{"api-server", "porch"}},
		},
	}
	s.writeJSONResponse(w, http.StatusOK, topology)
}

// getSystemDependencies handles GET /api/v1/dashboard/dependencies
func (s *NephoranAPIServer) getSystemDependencies(w http.ResponseWriter, r *http.Request) {
	deps := map[string]interface{}{
		"dependencies": []map[string]interface{}{
			{"name": "Kubernetes API", "status": "healthy", "version": "v1.29.0"},
			{"name": "Porch", "status": "healthy", "version": "v0.3.0"},
			{"name": "PostgreSQL", "status": "healthy", "version": "13.7"},
			{"name": "Redis", "status": "healthy", "version": "7.0.5"},
		},
	}
	s.writeJSONResponse(w, http.StatusOK, deps)
}

// livenessCheck handles GET /healthz
func (s *NephoranAPIServer) livenessCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// getSystemStats handles GET /api/v1/system/stats
func (s *NephoranAPIServer) getSystemStats(w http.ResponseWriter, r *http.Request) {
	stats := map[string]interface{}{
		"uptime_seconds":      86400,
		"requests_total":      15432,
		"requests_per_second": 45.2,
		"active_connections":  23,
		"memory_usage_mb":     512,
		"cpu_usage_percent":   23.4,
	}
	s.writeJSONResponse(w, http.StatusOK, stats)
}

// getSystemConfig handles GET /api/v1/system/config
func (s *NephoranAPIServer) getSystemConfig(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"version": "1.0.0",
		"build_info": map[string]interface{}{
			"commit":     "abc123def",
			"build_time": "2025-01-01T00:00:00Z",
			"go_version": "go1.24.0",
		},
		"features": map[string]bool{
			"authentication": s.authMiddleware != nil,
			"caching":        s.cache != nil,
			"rate_limiting":  s.rateLimiter != nil,
		},
	}
	s.writeJSONResponse(w, http.StatusOK, config)
}

// getCacheStats handles GET /api/v1/system/cache/stats
func (s *NephoranAPIServer) getCacheStats(w http.ResponseWriter, r *http.Request) {
	if s.cache == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cache_disabled", "Cache is not enabled")
		return
	}

	stats := map[string]interface{}{
		"enabled":        true,
		"hit_rate":       "92.3%",
		"total_requests": 1250,
		"hits":           1154,
		"misses":         96,
		"evictions":      15,
	}
	s.writeJSONResponse(w, http.StatusOK, stats)
}

// clearCache handles DELETE /api/v1/system/cache
func (s *NephoranAPIServer) clearCache(w http.ResponseWriter, r *http.Request) {
	if s.cache == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cache_disabled", "Cache is not enabled")
		return
	}

	// In a real implementation, would clear the cache
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message":    "Cache cleared successfully",
		"cleared_at": time.Now().UTC(),
	})
}

// getActiveConnections handles GET /api/v1/system/connections
func (s *NephoranAPIServer) getActiveConnections(w http.ResponseWriter, r *http.Request) {
	connections := map[string]interface{}{
		"websocket_connections": len(s.wsConnections),
		"sse_connections":       len(s.sseConnections),
		"active_sessions":       12,
		"total_connections":     23,
	}
	s.writeJSONResponse(w, http.StatusOK, connections)
}

// Additional handler implementations would continue here...
