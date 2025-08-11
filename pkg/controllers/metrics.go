// Package controllers provides controller-runtime metrics integration for the Nephoran Intent Operator.
//
// This package implements Prometheus metrics collection for NetworkIntent controllers using
// the controller-runtime metrics registry. Metrics are conditionally registered based on
// the METRICS_ENABLED environment variable.
//
// Metrics exposed:
//   - networkintent_reconciles_total: Total reconciliation count by controller/result
//   - networkintent_reconcile_errors_total: Total reconciliation errors by controller/error_type
//   - networkintent_processing_duration_seconds: Processing duration histograms by controller/phase
//   - networkintent_status: Current status gauge (0=Failed, 1=Processing, 2=Ready)
//
// Usage:
//   metrics := NewControllerMetrics("networkintent")
//   metrics.RecordSuccess("default", "my-intent")
//   metrics.RecordProcessingDuration("default", "my-intent", "llm_processing", 1.5)
//
package controllers

import (
	"os"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// Global metrics registration state
	metricsRegistered bool
	metricsRegistryMu sync.Mutex

	// Controller-runtime metrics for NetworkIntent controller
	networkIntentReconcilesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "networkintent_reconciles_total",
			Help: "Total number of NetworkIntent reconciliations",
		},
		[]string{"controller", "namespace", "name", "result"},
	)

	networkIntentReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "networkintent_reconcile_errors_total",
			Help: "Total number of NetworkIntent reconciliation errors",
		},
		[]string{"controller", "namespace", "name", "error_type"},
	)

	networkIntentProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "networkintent_processing_duration_seconds",
			Help:    "Duration of NetworkIntent processing phases",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"controller", "namespace", "name", "phase"},
	)

	networkIntentStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "networkintent_status",
			Help: "Status of NetworkIntent resources (0=Failed, 1=Processing, 2=Ready)",
		},
		[]string{"controller", "namespace", "name", "phase"},
	)
)

// ControllerMetrics provides controller-runtime compatible metrics for controllers
type ControllerMetrics struct {
	controllerName string
	enabled        bool
}

// NewControllerMetrics creates a new ControllerMetrics instance
func NewControllerMetrics(controllerName string) *ControllerMetrics {
	enabled := isMetricsEnabled()

	// Register metrics once globally if enabled
	if enabled {
		registerMetricsOnce()
	}

	return &ControllerMetrics{
		controllerName: controllerName,
		enabled:        enabled,
	}
}

// isMetricsEnabled checks if metrics are enabled via environment variable
func isMetricsEnabled() bool {
	enabled, err := strconv.ParseBool(os.Getenv("METRICS_ENABLED"))
	if err != nil {
		return false // Default to disabled
	}
	return enabled
}

// registerMetricsOnce ensures metrics are registered only once
func registerMetricsOnce() {
	metricsRegistryMu.Lock()
	defer metricsRegistryMu.Unlock()

	if metricsRegistered {
		return
	}

	metrics.Registry.MustRegister(
		networkIntentReconcilesTotal,
		networkIntentReconcileErrors,
		networkIntentProcessingDuration,
		networkIntentStatus,
	)

	metricsRegistered = true
}

// RecordReconcileTotal increments the total reconciliations counter
func (m *ControllerMetrics) RecordReconcileTotal(namespace, name, result string) {
	if !m.enabled {
		return
	}
	networkIntentReconcilesTotal.WithLabelValues(m.controllerName, namespace, name, result).Inc()
}

// RecordReconcileError increments the reconciliation errors counter
func (m *ControllerMetrics) RecordReconcileError(namespace, name, errorType string) {
	if !m.enabled {
		return
	}
	networkIntentReconcileErrors.WithLabelValues(m.controllerName, namespace, name, errorType).Inc()
}

// RecordProcessingDuration records the duration of a processing phase
func (m *ControllerMetrics) RecordProcessingDuration(namespace, name, phase string, duration float64) {
	if !m.enabled {
		return
	}
	networkIntentProcessingDuration.WithLabelValues(m.controllerName, namespace, name, phase).Observe(duration)
}

// SetStatus sets the current status of a NetworkIntent
func (m *ControllerMetrics) SetStatus(namespace, name, phase string, status float64) {
	if !m.enabled {
		return
	}
	networkIntentStatus.WithLabelValues(m.controllerName, namespace, name, phase).Set(status)
}

// RecordSuccess is a convenience method to record successful reconciliation
func (m *ControllerMetrics) RecordSuccess(namespace, name string) {
	m.RecordReconcileTotal(namespace, name, "success")
}

// RecordFailure is a convenience method to record failed reconciliation with error
func (m *ControllerMetrics) RecordFailure(namespace, name, errorType string) {
	m.RecordReconcileTotal(namespace, name, "error")
	m.RecordReconcileError(namespace, name, errorType)
}

// StatusValues provides constants for status metric values
const (
	StatusFailed     float64 = 0
	StatusProcessing float64 = 1
	StatusReady      float64 = 2
)

// GetMetricsEnabled returns whether metrics are currently enabled
func GetMetricsEnabled() bool {
	return isMetricsEnabled()
}
