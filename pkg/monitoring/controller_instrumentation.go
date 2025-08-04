package monitoring

import (
	"context"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"k8s.io/client-go/kubernetes"

	nephoranv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// InstrumentedReconciler wraps a reconciler with monitoring instrumentation
type InstrumentedReconciler struct {
	reconcile.Reconciler
	Name         string
	Metrics      *MetricsCollector
	HealthChecker *HealthChecker
}

// NewInstrumentedReconciler creates a new instrumented reconciler
func NewInstrumentedReconciler(reconciler reconcile.Reconciler, name string, metrics *MetricsCollector, kubeClient kubernetes.Interface, metricsRecorder *MetricsRecorder) *InstrumentedReconciler {
	return &InstrumentedReconciler{
		Reconciler: reconciler,
		Name:       name,
		Metrics:    metrics,
		HealthChecker: NewHealthChecker("1.0.0", kubeClient, metricsRecorder),
	}
}

// Reconcile wraps the reconciler with monitoring
func (ir *InstrumentedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	start := time.Now()
	logger := log.FromContext(ctx)
	
	// Record API latency for Kubernetes operations
	ir.recordAPILatency(ctx, start)
	
	// Call the wrapped reconciler
	result, err := ir.Reconciler.Reconcile(ctx, req)
	
	duration := time.Since(start)
	
	// Record reconciliation metrics
	if err != nil {
		logger.Error(err, "reconciliation failed", "controller", ir.Name, "duration", duration)
	} else {
		logger.V(1).Info("reconciliation completed", "controller", ir.Name, "duration", duration)
	}
	
	// Update controller health status
	ir.Metrics.UpdateControllerHealth(ir.Name, "reconciler", err == nil)
	
	return result, err
}

func (ir *InstrumentedReconciler) recordAPILatency(ctx context.Context, start time.Time) {
	// This would be called after each Kubernetes API operation
	// For now, we'll simulate by recording the start time
	ir.Metrics.RecordKubernetesAPILatency(time.Since(start))
}

// NetworkIntentInstrumentation provides instrumentation for NetworkIntent controller
type NetworkIntentInstrumentation struct {
	Metrics *MetricsCollector
}

// NewNetworkIntentInstrumentation creates new NetworkIntent instrumentation
func NewNetworkIntentInstrumentation(metrics *MetricsCollector) *NetworkIntentInstrumentation {
	return &NetworkIntentInstrumentation{
		Metrics: metrics,
	}
}

// RecordIntentProcessingStart records the start of intent processing
func (ni *NetworkIntentInstrumentation) RecordIntentProcessingStart(intent *nephoranv1alpha1.NetworkIntent) {
	intentType := "network_intent" // Default type since we don't have Spec.Type field
	ni.Metrics.UpdateNetworkIntentStatus(intent.Name, intent.Namespace, intentType, "processing")
}

// RecordIntentProcessingComplete records the completion of intent processing
func (ni *NetworkIntentInstrumentation) RecordIntentProcessingComplete(intent *nephoranv1alpha1.NetworkIntent, duration time.Duration, success bool) {
	status := "completed"
	if !success {
		status = "failed"
	}
	
	intentType := "network_intent" // Default type since we don't have Spec.Type field
	ni.Metrics.RecordNetworkIntentProcessed(intentType, status, duration)
	ni.Metrics.UpdateNetworkIntentStatus(intent.Name, intent.Namespace, intentType, status)
}

// RecordLLMProcessing records LLM processing metrics
func (ni *NetworkIntentInstrumentation) RecordLLMProcessing(model string, duration time.Duration, tokensUsed int, success bool) {
	status := "success"
	if !success {
		status = "error"
	}
	
	ni.Metrics.RecordLLMRequest(model, status, duration, tokensUsed)
}

// RecordRetry records a retry event
func (ni *NetworkIntentInstrumentation) RecordRetry(intent *nephoranv1alpha1.NetworkIntent, reason string) {
	ni.Metrics.RecordNetworkIntentRetry(intent.Name, intent.Namespace, reason)
}

// E2NodeSetInstrumentation provides instrumentation for E2NodeSet controller
type E2NodeSetInstrumentation struct {
	Metrics *MetricsCollector
}

// NewE2NodeSetInstrumentation creates new E2NodeSet instrumentation
func NewE2NodeSetInstrumentation(metrics *MetricsCollector) *E2NodeSetInstrumentation {
	return &E2NodeSetInstrumentation{
		Metrics: metrics,
	}
}

// RecordReconciliation records an E2NodeSet reconciliation
func (e2i *E2NodeSetInstrumentation) RecordReconciliation(operation string, duration time.Duration) {
	e2i.Metrics.RecordE2NodeSetOperation(operation, duration)
}

// UpdateReplicaStatus updates replica status metrics
func (e2i *E2NodeSetInstrumentation) UpdateReplicaStatus(e2nodeSet *nephoranv1alpha1.E2NodeSet) {
	e2i.Metrics.UpdateE2NodeSetReplicas(
		e2nodeSet.Name, 
		e2nodeSet.Namespace, 
		"desired", 
		int(e2nodeSet.Spec.Replicas),
	)
	
	e2i.Metrics.UpdateE2NodeSetReplicas(
		e2nodeSet.Name, 
		e2nodeSet.Namespace, 
		"ready", 
		int(e2nodeSet.Status.ReadyReplicas),
	)
}

// RecordScalingEvent records a scaling event
func (e2i *E2NodeSetInstrumentation) RecordScalingEvent(e2nodeSet *nephoranv1alpha1.E2NodeSet, oldReplicas, newReplicas int32) {
	direction := "up"
	if newReplicas < oldReplicas {
		direction = "down"
	}
	
	e2i.Metrics.RecordE2NodeSetScaling(e2nodeSet.Name, e2nodeSet.Namespace, direction)
}

// ORANInstrumentation provides instrumentation for O-RAN interfaces
type ORANInstrumentation struct {
	Metrics *MetricsCollector
}

// NewORANInstrumentation creates new O-RAN instrumentation
func NewORANInstrumentation(metrics *MetricsCollector) *ORANInstrumentation {
	return &ORANInstrumentation{
		Metrics: metrics,
	}
}

// RecordA1Operation records an A1 interface operation
func (oi *ORANInstrumentation) RecordA1Operation(operation string, duration time.Duration, success bool, errorType string) {
	status := "success"
	if !success {
		status = "error"
		oi.Metrics.RecordORANInterfaceError("A1", operation, errorType)
	}
	
	oi.Metrics.RecordORANInterfaceRequest("A1", operation, status, duration)
}

// RecordO1Operation records an O1 interface operation
func (oi *ORANInstrumentation) RecordO1Operation(operation string, duration time.Duration, success bool, errorType string) {
	status := "success"
	if !success {
		status = "error"
		oi.Metrics.RecordORANInterfaceError("O1", operation, errorType)
	}
	
	oi.Metrics.RecordORANInterfaceRequest("O1", operation, status, duration)
}

// RecordO2Operation records an O2 interface operation
func (oi *ORANInstrumentation) RecordO2Operation(operation string, duration time.Duration, success bool, errorType string) {
	status := "success"
	if !success {
		status = "error"
		oi.Metrics.RecordORANInterfaceError("O2", operation, errorType)
	}
	
	oi.Metrics.RecordORANInterfaceRequest("O2", operation, status, duration)
}

// UpdateConnectionStatus updates O-RAN connection status
func (oi *ORANInstrumentation) UpdateConnectionStatus(interfaceType, endpoint string, connected bool) {
	oi.Metrics.UpdateORANConnectionStatus(interfaceType, endpoint, connected)
}

// UpdatePolicyInstances updates policy instance counts
func (oi *ORANInstrumentation) UpdatePolicyInstances(policyType, status string, count int) {
	oi.Metrics.UpdateORANPolicyInstances(policyType, status, count)
}

// RAGInstrumentation provides instrumentation for RAG operations
type RAGInstrumentation struct {
	Metrics *MetricsCollector
}

// NewRAGInstrumentation creates new RAG instrumentation
func NewRAGInstrumentation(metrics *MetricsCollector) *RAGInstrumentation {
	return &RAGInstrumentation{
		Metrics: metrics,
	}
}

// RecordRetrieval records a RAG retrieval operation
func (ri *RAGInstrumentation) RecordRetrieval(duration time.Duration, cacheHit bool) {
	ri.Metrics.RecordRAGOperation(duration, cacheHit)
}

// UpdateDocumentCount updates the indexed document count
func (ri *RAGInstrumentation) UpdateDocumentCount(count int) {
	ri.Metrics.UpdateRAGDocumentsIndexed(count)
}

// GitOpsInstrumentation provides instrumentation for GitOps operations
type GitOpsInstrumentation struct {
	Metrics *MetricsCollector
}

// NewGitOpsInstrumentation creates new GitOps instrumentation
func NewGitOpsInstrumentation(metrics *MetricsCollector) *GitOpsInstrumentation {
	return &GitOpsInstrumentation{
		Metrics: metrics,
	}
}

// RecordPackageGeneration records package generation
func (gi *GitOpsInstrumentation) RecordPackageGeneration(duration time.Duration, success bool) {
	gi.Metrics.RecordGitOpsOperation("package_generation", duration, success)
}

// RecordCommit records a Git commit operation
func (gi *GitOpsInstrumentation) RecordCommit(duration time.Duration, success bool) {
	gi.Metrics.RecordGitOpsOperation("commit", duration, success)
}

// UpdateSyncStatus updates GitOps sync status
func (gi *GitOpsInstrumentation) UpdateSyncStatus(repository, branch string, inSync bool) {
	gi.Metrics.UpdateGitOpsSyncStatus(repository, branch, inSync)
}

// SystemInstrumentation provides system-level instrumentation
type SystemInstrumentation struct {
	Metrics *MetricsCollector
}

// NewSystemInstrumentation creates new system instrumentation
func NewSystemInstrumentation(metrics *MetricsCollector) *SystemInstrumentation {
	return &SystemInstrumentation{
		Metrics: metrics,
	}
}

// UpdateResourceUtilization updates resource utilization metrics
func (si *SystemInstrumentation) UpdateResourceUtilization(resourceType, unit string, value float64) {
	si.Metrics.UpdateResourceUtilization(resourceType, unit, value)
}

// UpdateWorkerQueueMetrics updates worker queue metrics
func (si *SystemInstrumentation) UpdateWorkerQueueMetrics(queueName string, depth int, latency time.Duration) {
	si.Metrics.UpdateWorkerQueueMetrics(queueName, depth, latency)
}

// InstrumentationManager manages all instrumentation components
type InstrumentationManager struct {
	Metrics                    *MetricsCollector
	HealthChecker              *HealthChecker
	NetworkIntentInstrumentation *NetworkIntentInstrumentation
	E2NodeSetInstrumentation     *E2NodeSetInstrumentation
	ORANInstrumentation          *ORANInstrumentation
	RAGInstrumentation           *RAGInstrumentation
	GitOpsInstrumentation        *GitOpsInstrumentation
	SystemInstrumentation        *SystemInstrumentation
}

// NewInstrumentationManager creates a new instrumentation manager
func NewInstrumentationManager(kubeClient kubernetes.Interface, metricsRecorder *MetricsRecorder) *InstrumentationManager {
	metrics := NewMetricsCollector()
	healthChecker := NewHealthChecker("1.0.0", kubeClient, metricsRecorder)
	
	// Register health checks (these are now handled internally by healthChecker)
	// The health checks are registered automatically in the Start() method
	
	return &InstrumentationManager{
		Metrics:                    metrics,
		HealthChecker:              healthChecker,
		NetworkIntentInstrumentation: NewNetworkIntentInstrumentation(metrics),
		E2NodeSetInstrumentation:     NewE2NodeSetInstrumentation(metrics),
		ORANInstrumentation:          NewORANInstrumentation(metrics),
		RAGInstrumentation:           NewRAGInstrumentation(metrics),
		GitOpsInstrumentation:        NewGitOpsInstrumentation(metrics),
		SystemInstrumentation:        NewSystemInstrumentation(metrics),
	}
}

// StartHealthChecks starts periodic health checks
func (im *InstrumentationManager) StartHealthChecks(ctx context.Context) error {
	return im.HealthChecker.Start(ctx)
}

// GetMetricsCollector returns the metrics collector
func (im *InstrumentationManager) GetMetricsCollector() *MetricsCollector {
	return im.Metrics
}