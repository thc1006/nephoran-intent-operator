package monitoring

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
)

// SimpleMetricsCollector provides a simple implementation of MetricsCollector
type SimpleMetricsCollector struct {
	mu       sync.RWMutex
	metrics  []*Metric
	client   kubernetes.Interface
	recorder *MetricsRecorder
}

// NewMetricsCollector creates a new simple metrics collector
func NewMetricsCollector() *SimpleMetricsCollector {
	return &SimpleMetricsCollector{
		metrics: make([]*Metric, 0),
	}
}

// NewMetricsCollectorWithClients creates a new metrics collector with Kubernetes client and recorder
func NewMetricsCollectorWithClients(client kubernetes.Interface, recorder *MetricsRecorder) *SimpleMetricsCollector {
	return &SimpleMetricsCollector{
		metrics:  make([]*Metric, 0),
		client:   client,
		recorder: recorder,
	}
}

// CollectMetrics implements MetricsCollector interface
func (c *SimpleMetricsCollector) CollectMetrics() ([]*Metric, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy of the metrics
	result := make([]*Metric, len(c.metrics))
	copy(result, c.metrics)
	return result, nil
}

// Start implements MetricsCollector interface
func (c *SimpleMetricsCollector) Start() error {
	// Simple collector doesn't need to start any background processes
	return nil
}

// Stop implements MetricsCollector interface
func (c *SimpleMetricsCollector) Stop() error {
	// Simple collector doesn't need to stop any background processes
	return nil
}

// RegisterMetrics implements MetricsCollector interface
func (c *SimpleMetricsCollector) RegisterMetrics(registry interface{}) error {
	// Simple collector doesn't need to register with external registries
	return nil
}

// UpdateControllerHealth records controller health status
func (c *SimpleMetricsCollector) UpdateControllerHealth(controllerName, component string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}

	metric := &Metric{
		Name:      "controller_health",
		Type:      MetricTypeGauge,
		Value:     value,
		Timestamp: time.Now(),
		Labels: map[string]string{
			"controller": controllerName,
			"component":  component,
		},
		Description: "Controller health status (1=healthy, 0=unhealthy)",
	}

	c.addMetric(metric)
}

// RecordKubernetesAPILatency records Kubernetes API call latency
func (c *SimpleMetricsCollector) RecordKubernetesAPILatency(latency time.Duration) {
	metric := &Metric{
		Name:        "kubernetes_api_duration_seconds",
		Type:        MetricTypeHistogram,
		Value:       latency.Seconds(),
		Timestamp:   time.Now(),
		Description: "Duration of Kubernetes API calls in seconds",
	}

	c.addMetric(metric)
}

// UpdateNetworkIntentStatus updates NetworkIntent status
func (c *SimpleMetricsCollector) UpdateNetworkIntentStatus(name, namespace, intentType, status string) {
	metric := &Metric{
		Name:      "network_intent_status",
		Type:      MetricTypeGauge,
		Value:     1.0,
		Timestamp: time.Now(),
		Labels: map[string]string{
			"name":        name,
			"namespace":   namespace,
			"intent_type": intentType,
			"status":      status,
		},
		Description: "NetworkIntent status",
	}

	c.addMetric(metric)
}

// RecordNetworkIntentProcessed records NetworkIntent processing metrics
func (c *SimpleMetricsCollector) RecordNetworkIntentProcessed(intentType, status string, duration time.Duration) {
	metric := &Metric{
		Name:      "network_intent_processing_duration_seconds",
		Type:      MetricTypeHistogram,
		Value:     duration.Seconds(),
		Timestamp: time.Now(),
		Labels: map[string]string{
			"intent_type": intentType,
			"status":      status,
		},
		Description: "Duration of NetworkIntent processing in seconds",
	}

	c.addMetric(metric)
}

// RecordNetworkIntentRetry records NetworkIntent retry events
func (c *SimpleMetricsCollector) RecordNetworkIntentRetry(name, namespace, reason string) {
	metric := &Metric{
		Name:      "network_intent_retries_total",
		Type:      MetricTypeCounter,
		Value:     1.0,
		Timestamp: time.Now(),
		Labels: map[string]string{
			"name":      name,
			"namespace": namespace,
			"reason":    reason,
		},
		Description: "Total number of NetworkIntent retry attempts",
	}

	c.addMetric(metric)
}

// RecordLLMRequest records LLM request metrics
func (c *SimpleMetricsCollector) RecordLLMRequest(model, status string, duration time.Duration, tokensUsed int) {
	metric := &Metric{
		Name:      "llm_request_duration_seconds",
		Type:      MetricTypeHistogram,
		Value:     duration.Seconds(),
		Timestamp: time.Now(),
		Labels: map[string]string{
			"model":  model,
			"status": status,
		},
		Description: "Duration of LLM requests in seconds",
	}
	c.addMetric(metric)

	tokensMetric := &Metric{
		Name:      "llm_tokens_used_total",
		Type:      MetricTypeCounter,
		Value:     float64(tokensUsed),
		Timestamp: time.Now(),
		Labels: map[string]string{
			"model":  model,
			"status": status,
		},
		Description: "Total number of LLM tokens used",
	}
	c.addMetric(tokensMetric)
}

// RecordE2NodeSetOperation records E2NodeSet operation metrics
func (c *SimpleMetricsCollector) RecordE2NodeSetOperation(operation string, duration time.Duration) {
	metric := &Metric{
		Name:      "e2nodeset_operation_duration_seconds",
		Type:      MetricTypeHistogram,
		Value:     duration.Seconds(),
		Timestamp: time.Now(),
		Labels: map[string]string{
			"operation": operation,
		},
		Description: "Duration of E2NodeSet operations in seconds",
	}

	c.addMetric(metric)
}

// UpdateE2NodeSetReplicas updates E2NodeSet replica metrics
func (c *SimpleMetricsCollector) UpdateE2NodeSetReplicas(name, namespace, status string, count int) {
	metric := &Metric{
		Name:      "e2nodeset_replicas",
		Type:      MetricTypeGauge,
		Value:     float64(count),
		Timestamp: time.Now(),
		Labels: map[string]string{
			"name":      name,
			"namespace": namespace,
			"status":    status,
		},
		Description: "Number of E2NodeSet replicas",
	}

	c.addMetric(metric)
}

// RecordE2NodeSetScaling records E2NodeSet scaling events
func (c *SimpleMetricsCollector) RecordE2NodeSetScaling(name, namespace, direction string) {
	metric := &Metric{
		Name:      "e2nodeset_scaling_events_total",
		Type:      MetricTypeCounter,
		Value:     1.0,
		Timestamp: time.Now(),
		Labels: map[string]string{
			"name":      name,
			"namespace": namespace,
			"direction": direction,
		},
		Description: "Total number of E2NodeSet scaling events",
	}

	c.addMetric(metric)
}

// RecordORANInterfaceRequest records O-RAN interface request metrics
func (c *SimpleMetricsCollector) RecordORANInterfaceRequest(interfaceType, operation, status string, duration time.Duration) {
	metric := &Metric{
		Name:      "oran_interface_request_duration_seconds",
		Type:      MetricTypeHistogram,
		Value:     duration.Seconds(),
		Timestamp: time.Now(),
		Labels: map[string]string{
			"interface": interfaceType,
			"operation": operation,
			"status":    status,
		},
		Description: "Duration of O-RAN interface requests in seconds",
	}

	c.addMetric(metric)
}

// RecordORANInterfaceError records O-RAN interface error metrics
func (c *SimpleMetricsCollector) RecordORANInterfaceError(interfaceType, operation, errorType string) {
	metric := &Metric{
		Name:      "oran_interface_errors_total",
		Type:      MetricTypeCounter,
		Value:     1.0,
		Timestamp: time.Now(),
		Labels: map[string]string{
			"interface":  interfaceType,
			"operation":  operation,
			"error_type": errorType,
		},
		Description: "Total number of O-RAN interface errors",
	}

	c.addMetric(metric)
}

// RecordCNFDeployment records CNF deployment metrics
func (c *SimpleMetricsCollector) RecordCNFDeployment(functionName string, duration time.Duration) {
	metric := &Metric{
		Name:      "cnf_deployment_duration_seconds",
		Type:      MetricTypeHistogram,
		Value:     duration.Seconds(),
		Timestamp: time.Now(),
		Labels: map[string]string{
			"function": functionName,
		},
		Description: "CNF deployment duration in seconds",
	}

	c.addMetric(metric)
}

// UpdateORANConnectionStatus updates O-RAN connection status
func (c *SimpleMetricsCollector) UpdateORANConnectionStatus(interfaceType, endpoint string, connected bool) {
	value := 0.0
	if connected {
		value = 1.0
	}

	metric := &Metric{
		Name:      "oran_connection_status",
		Type:      MetricTypeGauge,
		Value:     value,
		Timestamp: time.Now(),
		Labels: map[string]string{
			"interface": interfaceType,
			"endpoint":  endpoint,
		},
		Description: "O-RAN connection status (1=connected, 0=disconnected)",
	}

	c.addMetric(metric)
}

// UpdateORANPolicyInstances updates O-RAN policy instance counts
func (c *SimpleMetricsCollector) UpdateORANPolicyInstances(policyType, status string, count int) {
	metric := &Metric{
		Name:      "oran_policy_instances",
		Type:      MetricTypeGauge,
		Value:     float64(count),
		Timestamp: time.Now(),
		Labels: map[string]string{
			"policy_type": policyType,
			"status":      status,
		},
		Description: "Number of O-RAN policy instances",
	}

	c.addMetric(metric)
}

// RecordRAGOperation records RAG operation metrics
func (c *SimpleMetricsCollector) RecordRAGOperation(duration time.Duration, cacheHit bool) {
	metric := &Metric{
		Name:      "rag_operation_duration_seconds",
		Type:      MetricTypeHistogram,
		Value:     duration.Seconds(),
		Timestamp: time.Now(),
		Labels: map[string]string{
			"cache_hit": fmt.Sprintf("%t", cacheHit),
		},
		Description: "Duration of RAG operations in seconds",
	}

	c.addMetric(metric)
}

// UpdateRAGDocumentsIndexed updates RAG document count
func (c *SimpleMetricsCollector) UpdateRAGDocumentsIndexed(count int) {
	metric := &Metric{
		Name:        "rag_documents_indexed",
		Type:        MetricTypeGauge,
		Value:       float64(count),
		Timestamp:   time.Now(),
		Description: "Number of documents indexed in RAG system",
	}

	c.addMetric(metric)
}

// RecordGitOpsOperation records GitOps operation metrics
func (c *SimpleMetricsCollector) RecordGitOpsOperation(operation string, duration time.Duration, success bool) {
	metric := &Metric{
		Name:      "gitops_operation_duration_seconds",
		Type:      MetricTypeHistogram,
		Value:     duration.Seconds(),
		Timestamp: time.Now(),
		Labels: map[string]string{
			"operation": operation,
			"success":   fmt.Sprintf("%t", success),
		},
		Description: "Duration of GitOps operations in seconds",
	}

	c.addMetric(metric)
}

// UpdateGitOpsSyncStatus updates GitOps sync status
func (c *SimpleMetricsCollector) UpdateGitOpsSyncStatus(repository, branch string, inSync bool) {
	value := 0.0
	if inSync {
		value = 1.0
	}

	metric := &Metric{
		Name:      "gitops_sync_status",
		Type:      MetricTypeGauge,
		Value:     value,
		Timestamp: time.Now(),
		Labels: map[string]string{
			"repository": repository,
			"branch":     branch,
		},
		Description: "GitOps sync status (1=in_sync, 0=out_of_sync)",
	}

	c.addMetric(metric)
}

// UpdateResourceUtilization updates resource utilization metrics
func (c *SimpleMetricsCollector) UpdateResourceUtilization(resourceType, unit string, value float64) {
	metric := &Metric{
		Name:      "resource_utilization",
		Type:      MetricTypeGauge,
		Value:     value,
		Timestamp: time.Now(),
		Labels: map[string]string{
			"resource_type": resourceType,
			"unit":          unit,
		},
		Description: "Resource utilization metrics",
	}

	c.addMetric(metric)
}

// UpdateWorkerQueueMetrics updates worker queue metrics
func (c *SimpleMetricsCollector) UpdateWorkerQueueMetrics(queueName string, depth int, latency time.Duration) {
	depthMetric := &Metric{
		Name:      "worker_queue_depth",
		Type:      MetricTypeGauge,
		Value:     float64(depth),
		Timestamp: time.Now(),
		Labels: map[string]string{
			"queue": queueName,
		},
		Description: "Worker queue depth",
	}
	c.addMetric(depthMetric)

	latencyMetric := &Metric{
		Name:      "worker_queue_latency_seconds",
		Type:      MetricTypeHistogram,
		Value:     latency.Seconds(),
		Timestamp: time.Now(),
		Labels: map[string]string{
			"queue": queueName,
		},
		Description: "Worker queue latency in seconds",
	}
	c.addMetric(latencyMetric)
}

// RecordHTTPRequest records HTTP request metrics
func (c *SimpleMetricsCollector) RecordHTTPRequest(method, endpoint, status string, duration time.Duration) {
	metric := &Metric{
		Name:      "http_request_duration_seconds",
		Type:      MetricTypeHistogram,
		Value:     duration.Seconds(),
		Timestamp: time.Now(),
		Labels: map[string]string{
			"method":   method,
			"endpoint": endpoint,
			"status":   status,
		},
		Description: "Duration of HTTP requests in seconds",
	}
	c.addMetric(metric)
}

// RecordSSEStream records Server-Sent Events stream metrics
func (c *SimpleMetricsCollector) RecordSSEStream(endpoint string, connected bool) {
	value := 0.0
	if connected {
		value = 1.0
	}

	metric := &Metric{
		Name:      "sse_stream_active",
		Type:      MetricTypeGauge,
		Value:     value,
		Timestamp: time.Now(),
		Labels: map[string]string{
			"endpoint": endpoint,
		},
		Description: "Active SSE stream connections",
	}
	c.addMetric(metric)
}

// RecordLLMRequestError records LLM request error metrics
func (c *SimpleMetricsCollector) RecordLLMRequestError(model, errorType string) {
	metric := &Metric{
		Name:      "llm_request_errors_total",
		Type:      MetricTypeCounter,
		Value:     1.0,
		Timestamp: time.Now(),
		Labels: map[string]string{
			"model":      model,
			"error_type": errorType,
		},
		Description: "Total number of LLM request errors",
	}
	c.addMetric(metric)
}

// GetGauge returns a gauge metric interface (stub implementation)
func (c *SimpleMetricsCollector) GetGauge(name string) interface{} {
	// Stub implementation - return a simple struct that satisfies interface{} needs
	return json.RawMessage("{}")
}

// GetHistogram returns a histogram metric interface (stub implementation)
func (c *SimpleMetricsCollector) GetHistogram(name string) interface{} {
	// Stub implementation - return a simple struct that satisfies interface{} needs
	return json.RawMessage("{}")
}

// GetCounter returns a counter metric interface (stub implementation)
func (c *SimpleMetricsCollector) GetCounter(name string) interface{} {
	// Stub implementation - return a simple struct that satisfies interface{} needs
	return json.RawMessage("{}")
}

// addMetric adds a metric to the collection (thread-safe)
func (c *SimpleMetricsCollector) addMetric(metric *Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = append(c.metrics, metric)
}
