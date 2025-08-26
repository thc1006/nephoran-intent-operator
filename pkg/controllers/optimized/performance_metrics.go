package optimized

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// ControllerMetrics provides performance metrics for optimized controllers
type ControllerMetrics struct {
	// Reconcile metrics
	ReconcileDuration *prometheus.HistogramVec
	ReconcileTotal    *prometheus.CounterVec
	ReconcileErrors   *prometheus.CounterVec
	ReconcileRequeue  *prometheus.CounterVec

	// Backoff metrics
	BackoffDelay   *prometheus.HistogramVec
	BackoffRetries *prometheus.HistogramVec
	BackoffResets  *prometheus.CounterVec

	// Status batcher metrics
	StatusBatchSize      *prometheus.HistogramVec
	StatusBatchDuration  *prometheus.HistogramVec
	StatusUpdatesQueued  *prometheus.CounterVec
	StatusUpdatesDropped *prometheus.CounterVec
	StatusUpdatesFailed  *prometheus.CounterVec
	StatusQueueSize      *prometheus.GaugeVec

	// API client metrics
	ApiCallDuration *prometheus.HistogramVec
	ApiCallTotal    *prometheus.CounterVec
	ApiCallErrors   *prometheus.CounterVec

	// Resource metrics
	ActiveReconcilers *prometheus.GaugeVec
	MemoryUsage       *prometheus.GaugeVec
	GoroutineCount    *prometheus.GaugeVec
}

// NewControllerMetrics creates a new ControllerMetrics instance
func NewControllerMetrics() *ControllerMetrics {
	cm := &ControllerMetrics{
		ReconcileDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "controller_reconcile_duration_seconds",
				Help:    "Time spent in controller reconcile loops",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to 16s
			},
			[]string{"controller", "namespace", "name", "phase"},
		),

		ReconcileTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "controller_reconcile_total",
				Help: "Total number of controller reconciliations",
			},
			[]string{"controller", "result"},
		),

		ReconcileErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "controller_reconcile_errors_total",
				Help: "Total number of controller reconcile errors",
			},
			[]string{"controller", "error_type", "error_category"},
		),

		ReconcileRequeue: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "controller_reconcile_requeue_total",
				Help: "Total number of controller requeues",
			},
			[]string{"controller", "requeue_type", "backoff_strategy"},
		),

		BackoffDelay: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "controller_backoff_delay_seconds",
				Help:    "Backoff delay duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 12), // 100ms to 6.8min
			},
			[]string{"controller", "error_type", "strategy"},
		),

		BackoffRetries: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "controller_backoff_retries",
				Help:    "Number of retries before success or giving up",
				Buckets: prometheus.LinearBuckets(0, 1, 11), // 0 to 10 retries
			},
			[]string{"controller", "error_type", "outcome"},
		),

		BackoffResets: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "controller_backoff_resets_total",
				Help: "Total number of successful backoff resets",
			},
			[]string{"controller", "resource_type"},
		),

		StatusBatchSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "controller_status_batch_size",
				Help:    "Size of status update batches",
				Buckets: prometheus.LinearBuckets(1, 2, 15), // 1 to 29
			},
			[]string{"controller", "priority"},
		),

		StatusBatchDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "controller_status_batch_duration_seconds",
				Help:    "Duration of status batch processing",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to 4s
			},
			[]string{"controller"},
		),

		StatusUpdatesQueued: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "controller_status_updates_queued_total",
				Help: "Total number of status updates queued",
			},
			[]string{"controller", "priority", "resource_type"},
		),

		StatusUpdatesDropped: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "controller_status_updates_dropped_total",
				Help: "Total number of status updates dropped due to queue full",
			},
			[]string{"controller", "resource_type"},
		),

		StatusUpdatesFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "controller_status_updates_failed_total",
				Help: "Total number of failed status updates",
			},
			[]string{"controller", "error_type", "resource_type"},
		),

		StatusQueueSize: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "controller_status_queue_size",
				Help: "Current size of status update queue",
			},
			[]string{"controller"},
		),

		ApiCallDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "controller_api_call_duration_seconds",
				Help:    "Duration of Kubernetes API calls",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to 4s
			},
			[]string{"controller", "operation", "resource_type"},
		),

		ApiCallTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "controller_api_call_total",
				Help: "Total number of Kubernetes API calls",
			},
			[]string{"controller", "operation", "result"},
		),

		ApiCallErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "controller_api_call_errors_total",
				Help: "Total number of Kubernetes API call errors",
			},
			[]string{"controller", "operation", "error_type"},
		),

		ActiveReconcilers: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "controller_active_reconcilers",
				Help: "Number of active reconciler goroutines",
			},
			[]string{"controller"},
		),

		MemoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "controller_memory_usage_bytes",
				Help: "Memory usage of the controller process",
			},
			[]string{"controller", "type"},
		),

		GoroutineCount: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "controller_goroutines",
				Help: "Number of goroutines in the controller process",
			},
			[]string{"controller"},
		),
	}

	// Register all metrics
	metrics.Registry.MustRegister(
		cm.ReconcileDuration,
		cm.ReconcileTotal,
		cm.ReconcileErrors,
		cm.ReconcileRequeue,
		cm.BackoffDelay,
		cm.BackoffRetries,
		cm.BackoffResets,
		cm.StatusBatchSize,
		cm.StatusBatchDuration,
		cm.StatusUpdatesQueued,
		cm.StatusUpdatesDropped,
		cm.StatusUpdatesFailed,
		cm.StatusQueueSize,
		cm.ApiCallDuration,
		cm.ApiCallTotal,
		cm.ApiCallErrors,
		cm.ActiveReconcilers,
		cm.MemoryUsage,
		cm.GoroutineCount,
	)

	return cm
}

// ReconcileTimer provides timing functionality for reconcile operations
type ReconcileTimer struct {
	metrics    *ControllerMetrics
	controller string
	namespace  string
	name       string
	phase      string
	startTime  time.Time
}

// NewReconcileTimer creates a new reconcile timer
func (cm *ControllerMetrics) NewReconcileTimer(controller, namespace, name, phase string) *ReconcileTimer {
	return &ReconcileTimer{
		metrics:    cm,
		controller: controller,
		namespace:  namespace,
		name:       name,
		phase:      phase,
		startTime:  time.Now(),
	}
}

// Finish completes the timing and records the duration
func (rt *ReconcileTimer) Finish() {
	duration := time.Since(rt.startTime).Seconds()
	rt.metrics.ReconcileDuration.WithLabelValues(
		rt.controller, rt.namespace, rt.name, rt.phase,
	).Observe(duration)
}

// ApiCallTimer provides timing functionality for API calls
type ApiCallTimer struct {
	metrics    *ControllerMetrics
	controller string
	operation  string
	resource   string
	startTime  time.Time
	mu         sync.Mutex
	finished   bool
}

// NewApiCallTimer creates a new API call timer
func (cm *ControllerMetrics) NewApiCallTimer(controller, operation, resource string) *ApiCallTimer {
	return &ApiCallTimer{
		metrics:    cm,
		controller: controller,
		operation:  operation,
		resource:   resource,
		startTime:  time.Now(),
	}
}

// FinishWithResult completes the timing and records the result
func (act *ApiCallTimer) FinishWithResult(success bool, errorType string) {
	act.mu.Lock()
	defer act.mu.Unlock()

	if act.finished {
		return
	}
	act.finished = true

	duration := time.Since(act.startTime).Seconds()
	act.metrics.ApiCallDuration.WithLabelValues(
		act.controller, act.operation, act.resource,
	).Observe(duration)

	result := "success"
	if !success {
		result = "error"
		act.metrics.ApiCallErrors.WithLabelValues(
			act.controller, act.operation, errorType,
		).Inc()
	}

	act.metrics.ApiCallTotal.WithLabelValues(
		act.controller, act.operation, result,
	).Inc()
}

// RecordBackoffDelay records a backoff delay
func (cm *ControllerMetrics) RecordBackoffDelay(controller string, errorType ErrorType, strategy BackoffStrategy, delay time.Duration) {
	cm.BackoffDelay.WithLabelValues(
		controller, string(errorType), string(strategy),
	).Observe(delay.Seconds())
}

// RecordBackoffRetries records backoff retry attempts
func (cm *ControllerMetrics) RecordBackoffRetries(controller string, errorType ErrorType, retries int, success bool) {
	outcome := "success"
	if !success {
		outcome = "failed"
	}

	cm.BackoffRetries.WithLabelValues(
		controller, string(errorType), outcome,
	).Observe(float64(retries))
}

// RecordBackoffReset records a successful backoff reset
func (cm *ControllerMetrics) RecordBackoffReset(controller, resourceType string) {
	cm.BackoffResets.WithLabelValues(controller, resourceType).Inc()
}

// RecordStatusBatch records status batch metrics
func (cm *ControllerMetrics) RecordStatusBatch(controller string, size int, duration time.Duration, priority string) {
	cm.StatusBatchSize.WithLabelValues(controller, priority).Observe(float64(size))
	cm.StatusBatchDuration.WithLabelValues(controller).Observe(duration.Seconds())
}

// RecordStatusUpdate records status update queue operations
func (cm *ControllerMetrics) RecordStatusUpdate(controller, priority, resourceType, outcome string) {
	switch outcome {
	case "queued":
		cm.StatusUpdatesQueued.WithLabelValues(controller, priority, resourceType).Inc()
	case "dropped":
		cm.StatusUpdatesDropped.WithLabelValues(controller, resourceType).Inc()
	case "failed":
		cm.StatusUpdatesFailed.WithLabelValues(controller, "unknown", resourceType).Inc()
	}
}

// UpdateStatusQueueSize updates the current queue size gauge
func (cm *ControllerMetrics) UpdateStatusQueueSize(controller string, size int) {
	cm.StatusQueueSize.WithLabelValues(controller).Set(float64(size))
}

// RecordReconcileResult records the result of a reconcile operation
func (cm *ControllerMetrics) RecordReconcileResult(controller, result string) {
	cm.ReconcileTotal.WithLabelValues(controller, result).Inc()
}

// RecordReconcileError records a reconcile error
func (cm *ControllerMetrics) RecordReconcileError(controller string, errorType ErrorType, category string) {
	cm.ReconcileErrors.WithLabelValues(controller, string(errorType), category).Inc()
}

// RecordRequeue records a requeue operation
func (cm *ControllerMetrics) RecordRequeue(controller, requeueType string, strategy BackoffStrategy) {
	cm.ReconcileRequeue.WithLabelValues(controller, requeueType, string(strategy)).Inc()
}

// UpdateActiveReconcilers updates the active reconcilers gauge
func (cm *ControllerMetrics) UpdateActiveReconcilers(controller string, count int) {
	cm.ActiveReconcilers.WithLabelValues(controller).Set(float64(count))
}

// UpdateMemoryUsage updates memory usage metrics
func (cm *ControllerMetrics) UpdateMemoryUsage(controller, memType string, bytes int64) {
	cm.MemoryUsage.WithLabelValues(controller, memType).Set(float64(bytes))
}

// UpdateGoroutineCount updates the goroutine count gauge
func (cm *ControllerMetrics) UpdateGoroutineCount(controller string, count int) {
	cm.GoroutineCount.WithLabelValues(controller).Set(float64(count))
}
