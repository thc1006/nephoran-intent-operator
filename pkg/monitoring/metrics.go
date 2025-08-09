package monitoring

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// registrationOnce ensures metrics are only registered once
	registrationOnce sync.Once
)

// MetricsCollector handles all Prometheus metrics for the Nephoran Intent Operator
type MetricsCollector struct {
	// NetworkIntent metrics
	NetworkIntentTotal       prometheus.Counter
	NetworkIntentDuration    *prometheus.HistogramVec
	NetworkIntentStatus      *prometheus.GaugeVec
	NetworkIntentLLMDuration *prometheus.HistogramVec
	NetworkIntentRetries     *prometheus.CounterVec

	// E2NodeSet metrics
	E2NodeSetTotal             prometheus.Counter
	E2NodeSetReplicas          *prometheus.GaugeVec
	E2NodeSetReconcileDuration *prometheus.HistogramVec
	E2NodeSetScalingEvents     *prometheus.CounterVec

	// O-RAN Interface metrics
	ORANInterfaceRequests *prometheus.CounterVec
	ORANInterfaceDuration *prometheus.HistogramVec
	ORANInterfaceErrors   *prometheus.CounterVec
	ORANPolicyInstances   *prometheus.GaugeVec
	ORANConnectionStatus  *prometheus.GaugeVec

	// LLM and RAG metrics
	LLMRequestsTotal     prometheus.Counter
	LLMRequestDuration   prometheus.Histogram
	LLMTokensUsed        prometheus.Counter
	RAGRetrievalDuration prometheus.Histogram
	RAGCacheHits         prometheus.Counter
	RAGCacheMisses       prometheus.Counter
	RAGDocumentsIndexed  prometheus.Gauge

	// GitOps metrics
	GitOpsPackagesGenerated prometheus.Counter
	GitOpsCommitDuration    prometheus.Histogram
	GitOpsErrors            *prometheus.CounterVec
	GitOpsSyncStatus        *prometheus.GaugeVec
	GitPushInFlight         prometheus.Gauge

	// HTTP and SSE metrics
	HTTPRequestDuration *prometheus.HistogramVec
	SSEStreamDuration   *prometheus.HistogramVec

	// System health metrics
	ControllerHealthStatus *prometheus.GaugeVec
	KubernetesAPILatency   prometheus.Histogram
	ResourceUtilization    *prometheus.GaugeVec
	WorkerQueueDepth       *prometheus.GaugeVec
	WorkerQueueLatency     *prometheus.HistogramVec

	// Weaviate Connection Pool metrics
	WeaviatePoolConnectionsCreated   prometheus.Counter
	WeaviatePoolConnectionsDestroyed prometheus.Counter
	WeaviatePoolActiveConnections    prometheus.Gauge
	WeaviatePoolSize                 prometheus.Gauge
	WeaviatePoolHealthChecksPassed   prometheus.Counter
	WeaviatePoolHealthChecksFailed   prometheus.Counter
}

// NewMetricsCollector creates a new metrics collector with all Prometheus metrics
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		// NetworkIntent metrics
		NetworkIntentTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_networkintent_total",
			Help: "Total number of NetworkIntent resources processed",
		}),

		NetworkIntentDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nephoran_networkintent_duration_seconds",
			Help:    "Duration of NetworkIntent processing",
			Buckets: prometheus.DefBuckets,
		}, []string{"intent_type", "status"}),

		NetworkIntentStatus: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_networkintent_status",
			Help: "Current status of NetworkIntent resources (0=pending, 1=processing, 2=completed, 3=failed)",
		}, []string{"name", "namespace", "intent_type"}),

		NetworkIntentLLMDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nephoran_networkintent_llm_duration_seconds",
			Help:    "Duration of LLM processing for NetworkIntents",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0},
		}, []string{"model", "status"}),

		NetworkIntentRetries: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_networkintent_retries_total",
			Help: "Total number of NetworkIntent processing retries",
		}, []string{"name", "namespace", "reason"}),

		// E2NodeSet metrics
		E2NodeSetTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_e2nodeset_total",
			Help: "Total number of E2NodeSet resources processed",
		}),

		E2NodeSetReplicas: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_e2nodeset_replicas",
			Help: "Current number of E2NodeSet replicas",
		}, []string{"name", "namespace", "type"}),

		E2NodeSetReconcileDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nephoran_e2nodeset_reconcile_duration_seconds",
			Help:    "Duration of E2NodeSet reconciliation",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),

		E2NodeSetScalingEvents: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_e2nodeset_scaling_events_total",
			Help: "Total number of E2NodeSet scaling events",
		}, []string{"name", "namespace", "direction"}),

		// O-RAN Interface metrics
		ORANInterfaceRequests: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_oran_interface_requests_total",
			Help: "Total number of O-RAN interface requests",
		}, []string{"interface", "operation", "status"}),

		ORANInterfaceDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nephoran_oran_interface_duration_seconds",
			Help:    "Duration of O-RAN interface operations",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
		}, []string{"interface", "operation"}),

		ORANInterfaceErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_oran_interface_errors_total",
			Help: "Total number of O-RAN interface errors",
		}, []string{"interface", "operation", "error_type"}),

		ORANPolicyInstances: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_oran_policy_instances",
			Help: "Number of active O-RAN policy instances",
		}, []string{"policy_type", "status"}),

		ORANConnectionStatus: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_oran_connection_status",
			Help: "O-RAN interface connection status (0=disconnected, 1=connected)",
		}, []string{"interface", "endpoint"}),

		// LLM and RAG metrics
		LLMRequestsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_llm_requests_total",
			Help: "Total number of LLM requests",
		}),

		LLMRequestDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "nephoran_llm_request_duration_seconds",
			Help:    "Duration of LLM requests",
			Buckets: []float64{0.5, 1.0, 2.0, 5.0, 10.0, 30.0, 60.0, 120.0},
		}),

		LLMTokensUsed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_llm_tokens_used_total",
			Help: "Total number of LLM tokens consumed",
		}),

		RAGRetrievalDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "nephoran_rag_retrieval_duration_seconds",
			Help:    "Duration of RAG knowledge retrieval",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
		}),

		RAGCacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_rag_cache_hits_total",
			Help: "Total number of RAG cache hits",
		}),

		RAGCacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_rag_cache_misses_total",
			Help: "Total number of RAG cache misses",
		}),

		RAGDocumentsIndexed: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "nephoran_rag_documents_indexed",
			Help: "Number of documents indexed in RAG knowledge base",
		}),

		// GitOps metrics
		GitOpsPackagesGenerated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_gitops_packages_generated_total",
			Help: "Total number of GitOps packages generated",
		}),

		GitOpsCommitDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "nephoran_gitops_commit_duration_seconds",
			Help:    "Duration of GitOps commit operations",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0},
		}),

		GitOpsErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_gitops_errors_total",
			Help: "Total number of GitOps errors",
		}, []string{"operation", "error_type"}),

		GitOpsSyncStatus: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_gitops_sync_status",
			Help: "GitOps synchronization status (0=out-of-sync, 1=in-sync)",
		}, []string{"repository", "branch"}),

		GitPushInFlight: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "nephoran_git_push_in_flight",
			Help: "Number of git push operations currently in flight",
		}),

		// HTTP and SSE metrics
		HTTPRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nephoran_http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		}, []string{"method", "path", "code"}),

		SSEStreamDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nephoran_sse_stream_duration_seconds",
			Help:    "Duration of SSE streaming connections",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		}, []string{"route"}),

		// System health metrics
		ControllerHealthStatus: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_controller_health_status",
			Help: "Controller health status (0=unhealthy, 1=healthy)",
		}, []string{"controller", "component"}),

		KubernetesAPILatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "nephoran_kubernetes_api_latency_seconds",
			Help:    "Latency of Kubernetes API requests",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 2.0},
		}),

		ResourceUtilization: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_resource_utilization",
			Help: "Resource utilization metrics",
		}, []string{"resource_type", "unit"}),

		WorkerQueueDepth: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_worker_queue_depth",
			Help: "Depth of worker queues",
		}, []string{"queue_name"}),

		WorkerQueueLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "nephoran_worker_queue_latency_seconds",
			Help:    "Latency of items in worker queues",
			Buckets: prometheus.DefBuckets,
		}, []string{"queue_name"}),

		// Weaviate Connection Pool metrics
		WeaviatePoolConnectionsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_weaviate_pool_connections_created_total",
			Help: "Total number of Weaviate connections created in the pool",
		}),

		WeaviatePoolConnectionsDestroyed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_weaviate_pool_connections_destroyed_total",
			Help: "Total number of Weaviate connections destroyed in the pool",
		}),

		WeaviatePoolActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "nephoran_weaviate_pool_active_connections",
			Help: "Current number of active Weaviate connections in the pool",
		}),

		WeaviatePoolSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "nephoran_weaviate_pool_size",
			Help: "Current size of the Weaviate connection pool (total connections)",
		}),

		WeaviatePoolHealthChecksPassed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_weaviate_pool_health_checks_passed_total",
			Help: "Total number of successful Weaviate connection health checks",
		}),

		WeaviatePoolHealthChecksFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "nephoran_weaviate_pool_health_checks_failed_total",
			Help: "Total number of failed Weaviate connection health checks",
		}),
	}

	// Register all metrics with controller-runtime metrics registry
	// Use sync.Once to ensure metrics are only registered once
	registrationOnce.Do(func() {
		metrics.Registry.MustRegister(
			mc.NetworkIntentTotal,
			mc.NetworkIntentDuration,
			mc.NetworkIntentStatus,
			mc.NetworkIntentLLMDuration,
			mc.NetworkIntentRetries,
			mc.E2NodeSetTotal,
			mc.E2NodeSetReplicas,
			mc.E2NodeSetReconcileDuration,
			mc.E2NodeSetScalingEvents,
			mc.ORANInterfaceRequests,
			mc.ORANInterfaceDuration,
			mc.ORANInterfaceErrors,
			mc.ORANPolicyInstances,
			mc.ORANConnectionStatus,
			mc.LLMRequestsTotal,
			mc.LLMRequestDuration,
			mc.LLMTokensUsed,
			mc.RAGRetrievalDuration,
			mc.RAGCacheHits,
			mc.RAGCacheMisses,
			mc.RAGDocumentsIndexed,
			mc.GitOpsPackagesGenerated,
			mc.GitOpsCommitDuration,
			mc.GitOpsErrors,
			mc.GitOpsSyncStatus,
			mc.GitPushInFlight,
			mc.HTTPRequestDuration,
			mc.SSEStreamDuration,
			mc.ControllerHealthStatus,
			mc.KubernetesAPILatency,
			mc.ResourceUtilization,
			mc.WorkerQueueDepth,
			mc.WorkerQueueLatency,
			mc.WeaviatePoolConnectionsCreated,
			mc.WeaviatePoolConnectionsDestroyed,
			mc.WeaviatePoolActiveConnections,
			mc.WeaviatePoolSize,
			mc.WeaviatePoolHealthChecksPassed,
			mc.WeaviatePoolHealthChecksFailed,
		)
	})

	return mc
}

// Convenience methods for recording metrics

// RecordNetworkIntentProcessed records a processed NetworkIntent
func (mc *MetricsCollector) RecordNetworkIntentProcessed(intentType, status string, duration time.Duration) {
	mc.NetworkIntentTotal.Inc()
	mc.NetworkIntentDuration.WithLabelValues(intentType, status).Observe(duration.Seconds())
}

// UpdateNetworkIntentStatus updates the status of a NetworkIntent
func (mc *MetricsCollector) UpdateNetworkIntentStatus(name, namespace, intentType, status string) {
	var statusValue float64
	switch status {
	case "pending":
		statusValue = 0
	case "processing":
		statusValue = 1
	case "completed":
		statusValue = 2
	case "failed":
		statusValue = 3
	}
	mc.NetworkIntentStatus.WithLabelValues(name, namespace, intentType).Set(statusValue)
}

// RecordNetworkIntentRetry records a NetworkIntent retry
func (mc *MetricsCollector) RecordNetworkIntentRetry(name, namespace, reason string) {
	mc.NetworkIntentRetries.WithLabelValues(name, namespace, reason).Inc()
}

// RecordLLMRequest records an LLM request
func (mc *MetricsCollector) RecordLLMRequest(model, status string, duration time.Duration, tokensUsed int) {
	mc.LLMRequestsTotal.Inc()
	mc.LLMRequestDuration.Observe(duration.Seconds())
	mc.NetworkIntentLLMDuration.WithLabelValues(model, status).Observe(duration.Seconds())
	mc.LLMTokensUsed.Add(float64(tokensUsed))
}

// RecordE2NodeSetOperation records an E2NodeSet operation
func (mc *MetricsCollector) RecordE2NodeSetOperation(operation string, duration time.Duration) {
	mc.E2NodeSetTotal.Inc()
	mc.E2NodeSetReconcileDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// UpdateE2NodeSetReplicas updates E2NodeSet replica metrics
func (mc *MetricsCollector) UpdateE2NodeSetReplicas(name, namespace, replicaType string, count int) {
	mc.E2NodeSetReplicas.WithLabelValues(name, namespace, replicaType).Set(float64(count))
}

// RecordE2NodeSetScaling records an E2NodeSet scaling event
func (mc *MetricsCollector) RecordE2NodeSetScaling(name, namespace, direction string) {
	mc.E2NodeSetScalingEvents.WithLabelValues(name, namespace, direction).Inc()
}

// RecordORANInterfaceRequest records an O-RAN interface request
func (mc *MetricsCollector) RecordORANInterfaceRequest(interfaceType, operation, status string, duration time.Duration) {
	mc.ORANInterfaceRequests.WithLabelValues(interfaceType, operation, status).Inc()
	mc.ORANInterfaceDuration.WithLabelValues(interfaceType, operation).Observe(duration.Seconds())
}

// RecordORANInterfaceError records an O-RAN interface error
func (mc *MetricsCollector) RecordORANInterfaceError(interfaceType, operation, errorType string) {
	mc.ORANInterfaceErrors.WithLabelValues(interfaceType, operation, errorType).Inc()
}

// UpdateORANConnectionStatus updates O-RAN connection status
func (mc *MetricsCollector) UpdateORANConnectionStatus(interfaceType, endpoint string, connected bool) {
	var status float64
	if connected {
		status = 1
	}
	mc.ORANConnectionStatus.WithLabelValues(interfaceType, endpoint).Set(status)
}

// UpdateORANPolicyInstances updates O-RAN policy instance count
func (mc *MetricsCollector) UpdateORANPolicyInstances(policyType, status string, count int) {
	mc.ORANPolicyInstances.WithLabelValues(policyType, status).Set(float64(count))
}

// RecordRAGOperation records RAG operations
func (mc *MetricsCollector) RecordRAGOperation(duration time.Duration, cacheHit bool) {
	mc.RAGRetrievalDuration.Observe(duration.Seconds())
	if cacheHit {
		mc.RAGCacheHits.Inc()
	} else {
		mc.RAGCacheMisses.Inc()
	}
}

// UpdateRAGDocumentsIndexed updates the count of indexed documents
func (mc *MetricsCollector) UpdateRAGDocumentsIndexed(count int) {
	mc.RAGDocumentsIndexed.Set(float64(count))
}

// RecordGitOpsOperation records GitOps operations
func (mc *MetricsCollector) RecordGitOpsOperation(operation string, duration time.Duration, success bool) {
	if operation == "package_generation" {
		mc.GitOpsPackagesGenerated.Inc()
	}
	if operation == "commit" {
		mc.GitOpsCommitDuration.Observe(duration.Seconds())
	}
	if !success {
		mc.GitOpsErrors.WithLabelValues(operation, "execution_failed").Inc()
	}
}

// UpdateGitOpsSyncStatus updates GitOps sync status
func (mc *MetricsCollector) UpdateGitOpsSyncStatus(repository, branch string, inSync bool) {
	var status float64
	if inSync {
		status = 1
	}
	mc.GitOpsSyncStatus.WithLabelValues(repository, branch).Set(status)
}

// RecordHTTPRequest records HTTP request metrics
func (mc *MetricsCollector) RecordHTTPRequest(method, path, statusCode string, duration time.Duration) {
	mc.HTTPRequestDuration.WithLabelValues(method, path, statusCode).Observe(duration.Seconds())
}

// RecordSSEStream records SSE stream connection metrics
func (mc *MetricsCollector) RecordSSEStream(route string, duration time.Duration) {
	mc.SSEStreamDuration.WithLabelValues(route).Observe(duration.Seconds())
}

// UpdateControllerHealth updates controller health status
func (mc *MetricsCollector) UpdateControllerHealth(controller, component string, healthy bool) {
	var status float64
	if healthy {
		status = 1
	}
	mc.ControllerHealthStatus.WithLabelValues(controller, component).Set(status)
}

// RecordKubernetesAPILatency records Kubernetes API request latency
func (mc *MetricsCollector) RecordKubernetesAPILatency(duration time.Duration) {
	mc.KubernetesAPILatency.Observe(duration.Seconds())
}

// UpdateResourceUtilization updates resource utilization metrics
func (mc *MetricsCollector) UpdateResourceUtilization(resourceType, unit string, value float64) {
	mc.ResourceUtilization.WithLabelValues(resourceType, unit).Set(value)
}

// UpdateWorkerQueueMetrics updates worker queue metrics
func (mc *MetricsCollector) UpdateWorkerQueueMetrics(queueName string, depth int, latency time.Duration) {
	mc.WorkerQueueDepth.WithLabelValues(queueName).Set(float64(depth))
	mc.WorkerQueueLatency.WithLabelValues(queueName).Observe(latency.Seconds())
}

// Weaviate Connection Pool metrics methods

// RecordWeaviatePoolConnectionCreated records a created connection
func (mc *MetricsCollector) RecordWeaviatePoolConnectionCreated() {
	mc.WeaviatePoolConnectionsCreated.Inc()
}

// RecordWeaviatePoolConnectionDestroyed records a destroyed connection
func (mc *MetricsCollector) RecordWeaviatePoolConnectionDestroyed() {
	mc.WeaviatePoolConnectionsDestroyed.Inc()
}

// UpdateWeaviatePoolActiveConnections updates the active connections count
func (mc *MetricsCollector) UpdateWeaviatePoolActiveConnections(count int) {
	mc.WeaviatePoolActiveConnections.Set(float64(count))
}

// UpdateWeaviatePoolSize updates the total pool size
func (mc *MetricsCollector) UpdateWeaviatePoolSize(size int) {
	mc.WeaviatePoolSize.Set(float64(size))
}

// RecordWeaviatePoolHealthCheckPassed records a successful health check
func (mc *MetricsCollector) RecordWeaviatePoolHealthCheckPassed() {
	mc.WeaviatePoolHealthChecksPassed.Inc()
}

// RecordWeaviatePoolHealthCheckFailed records a failed health check
func (mc *MetricsCollector) RecordWeaviatePoolHealthCheckFailed() {
	mc.WeaviatePoolHealthChecksFailed.Inc()
}

// UpdateWeaviatePoolMetrics updates all pool metrics from pool metrics struct
func (mc *MetricsCollector) UpdateWeaviatePoolMetrics(activeCount, totalCount int) {
	mc.UpdateWeaviatePoolActiveConnections(activeCount)
	mc.UpdateWeaviatePoolSize(totalCount)
}

// HealthChecker functionality moved to health_checks.go to avoid duplication
