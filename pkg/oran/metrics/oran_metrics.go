package metrics

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// O-RAN Interface Metrics following O-RAN Alliance specifications.

// These metrics provide comprehensive monitoring of O-RAN interface operations.

var (

	// A1 Interface Metrics - Policy Management and Control.

	a1PolicyOperationsTotal = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Namespace: "nephoran",

			Subsystem: "a1",

			Name: "policy_operations_total",

			Help: "Total number of A1 policy operations",
		},

		[]string{"operation", "policy_type_id", "status"},
	)

	a1PolicyOperationDuration = promauto.NewHistogramVec(

		prometheus.HistogramOpts{

			Namespace: "nephoran",

			Subsystem: "a1",

			Name: "policy_operation_duration_seconds",

			Help: "Duration of A1 policy operations in seconds",

			Buckets: prometheus.DefBuckets,
		},

		[]string{"operation", "policy_type_id"},
	)

	a1PolicyTypesActive = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "a1",

			Name: "policy_types_active",

			Help: "Number of active A1 policy types",
		},

		[]string{"ric_id"},
	)

	a1PolicyInstancesActive = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "a1",

			Name: "policy_instances_active",

			Help: "Number of active A1 policy instances",
		},

		[]string{"policy_type_id", "ric_id", "enforcement_status"},
	)

	a1CircuitBreakerState = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "a1",

			Name: "circuit_breaker_state",

			Help: "A1 circuit breaker state (0=closed, 1=open, 2=half-open)",
		},

		[]string{"ric_id"},
	)

	a1RetryAttemptsTotal = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Namespace: "nephoran",

			Subsystem: "a1",

			Name: "retry_attempts_total",

			Help: "Total number of A1 retry attempts",
		},

		[]string{"operation", "final_status"},
	)

	// E2 Interface Metrics - Near-RT RIC Communication.

	e2MessageOperationsTotal = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Namespace: "nephoran",

			Subsystem: "e2",

			Name: "message_operations_total",

			Help: "Total number of E2AP message operations",
		},

		[]string{"message_type", "node_id", "status"},
	)

	e2MessageOperationDuration = promauto.NewHistogramVec(

		prometheus.HistogramOpts{

			Namespace: "nephoran",

			Subsystem: "e2",

			Name: "message_operation_duration_seconds",

			Help: "Duration of E2AP message operations in seconds",

			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},

		[]string{"message_type", "node_id"},
	)

	e2SubscriptionsActive = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "e2",

			Name: "subscriptions_active",

			Help: "Number of active E2 subscriptions",
		},

		[]string{"node_id", "ran_function_id", "subscription_type"},
	)

	e2NodesConnected = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "e2",

			Name: "nodes_connected",

			Help: "Number of connected E2 nodes",
		},

		[]string{"node_type", "plmn_id"},
	)

	e2IndicationsReceived = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Namespace: "nephoran",

			Subsystem: "e2",

			Name: "indications_received_total",

			Help: "Total number of E2 indications received",
		},

		[]string{"node_id", "ran_function_id", "action_id", "indication_type"},
	)

	e2ControlRequestsTotal = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Namespace: "nephoran",

			Subsystem: "e2",

			Name: "control_requests_total",

			Help: "Total number of E2 control requests",
		},

		[]string{"node_id", "ran_function_id", "status"},
	)

	e2CircuitBreakerState = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "e2",

			Name: "circuit_breaker_state",

			Help: "E2 circuit breaker state (0=closed, 1=open, 2=half-open)",
		},

		[]string{"node_id"},
	)

	// O1 Interface Metrics - Configuration and Fault Management.

	o1ConfigurationOperationsTotal = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Namespace: "nephoran",

			Subsystem: "o1",

			Name: "configuration_operations_total",

			Help: "Total number of O1 configuration operations",
		},

		[]string{"operation", "managed_element", "status"},
	)

	o1ConfigurationOperationDuration = promauto.NewHistogramVec(

		prometheus.HistogramOpts{

			Namespace: "nephoran",

			Subsystem: "o1",

			Name: "configuration_operation_duration_seconds",

			Help: "Duration of O1 configuration operations in seconds",

			Buckets: prometheus.DefBuckets,
		},

		[]string{"operation", "managed_element"},
	)

	o1FaultNotificationsTotal = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Namespace: "nephoran",

			Subsystem: "o1",

			Name: "fault_notifications_total",

			Help: "Total number of O1 fault notifications",
		},

		[]string{"managed_element", "alarm_type", "severity"},
	)

	o1ManagedElementsConnected = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "o1",

			Name: "managed_elements_connected",

			Help: "Number of connected O1 managed elements",
		},

		[]string{"element_type", "vendor"},
	)

	// O2 Interface Metrics - Cloud Infrastructure Management.

	o2InfrastructureOperationsTotal = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Namespace: "nephoran",

			Subsystem: "o2",

			Name: "infrastructure_operations_total",

			Help: "Total number of O2 infrastructure operations",
		},

		[]string{"operation", "resource_type", "status"},
	)

	o2InfrastructureOperationDuration = promauto.NewHistogramVec(

		prometheus.HistogramOpts{

			Namespace: "nephoran",

			Subsystem: "o2",

			Name: "infrastructure_operation_duration_seconds",

			Help: "Duration of O2 infrastructure operations in seconds",

			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 25, 50, 100},
		},

		[]string{"operation", "resource_type"},
	)

	o2ResourceUtilization = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "o2",

			Name: "resource_utilization_percent",

			Help: "O2 resource utilization percentage",
		},

		[]string{"resource_type", "resource_id", "cluster"},
	)

	o2DeploymentStatus = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "o2",

			Name: "deployment_status",

			Help: "O2 deployment status (0=failed, 1=deploying, 2=deployed, 3=terminated)",
		},

		[]string{"deployment_id", "application_type", "cluster"},
	)

	// General O-RAN Health and Performance Metrics.

	oranHealthCheckStatus = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "oran",

			Name: "health_check_status",

			Help: "O-RAN interface health check status (0=unhealthy, 1=healthy, 2=degraded)",
		},

		[]string{"interface", "component"},
	)

	oranHealthCheckDuration = promauto.NewHistogramVec(

		prometheus.HistogramOpts{

			Namespace: "nephoran",

			Subsystem: "oran",

			Name: "health_check_duration_seconds",

			Help: "Duration of O-RAN health checks in seconds",

			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},

		[]string{"interface", "component"},
	)

	oranDependencyStatus = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "oran",

			Name: "dependency_status",

			Help: "O-RAN dependency status (0=unhealthy, 1=healthy)",
		},

		[]string{"dependency_name", "dependency_type"},
	)

	oranCircuitBreakerTrips = promauto.NewCounterVec(

		prometheus.CounterOpts{

			Namespace: "nephoran",

			Subsystem: "oran",

			Name: "circuit_breaker_trips_total",

			Help: "Total number of circuit breaker trips across O-RAN interfaces",
		},

		[]string{"interface", "reason"},
	)

	// Performance and Resource Metrics.

	oranConcurrentConnections = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "oran",

			Name: "concurrent_connections",

			Help: "Number of concurrent connections per O-RAN interface",
		},

		[]string{"interface", "endpoint"},
	)

	oranThroughput = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "oran",

			Name: "throughput_messages_per_second",

			Help: "Message throughput per second for O-RAN interfaces",
		},

		[]string{"interface", "direction"}, // direction: inbound/outbound

	)

	oranErrorRate = promauto.NewGaugeVec(

		prometheus.GaugeOpts{

			Namespace: "nephoran",

			Subsystem: "oran",

			Name: "error_rate_percent",

			Help: "Error rate percentage for O-RAN interfaces",
		},

		[]string{"interface", "error_type"},
	)
)

// ORANMetricsCollector provides methods to update O-RAN metrics.

type ORANMetricsCollector struct {

	// Component registry for cleanup.

	componentRegistry map[string]bool
}

// NewORANMetricsCollector creates a new O-RAN metrics collector.

func NewORANMetricsCollector() *ORANMetricsCollector {

	collector := &ORANMetricsCollector{

		componentRegistry: make(map[string]bool),
	}

	// Register all metrics with controller-runtime.

	metrics.Registry.MustRegister(

		a1PolicyOperationsTotal,

		a1PolicyOperationDuration,

		a1PolicyTypesActive,

		a1PolicyInstancesActive,

		a1CircuitBreakerState,

		a1RetryAttemptsTotal,

		e2MessageOperationsTotal,

		e2MessageOperationDuration,

		e2SubscriptionsActive,

		e2NodesConnected,

		e2IndicationsReceived,

		e2ControlRequestsTotal,

		e2CircuitBreakerState,

		o1ConfigurationOperationsTotal,

		o1ConfigurationOperationDuration,

		o1FaultNotificationsTotal,

		o1ManagedElementsConnected,

		o2InfrastructureOperationsTotal,

		o2InfrastructureOperationDuration,

		o2ResourceUtilization,

		o2DeploymentStatus,

		oranHealthCheckStatus,

		oranHealthCheckDuration,

		oranDependencyStatus,

		oranCircuitBreakerTrips,

		oranConcurrentConnections,

		oranThroughput,

		oranErrorRate,
	)

	return collector

}

// A1 Interface Metrics Methods.

// RecordA1PolicyOperation records an A1 policy operation.

func (c *ORANMetricsCollector) RecordA1PolicyOperation(operation, policyTypeID, status string, duration time.Duration) {

	a1PolicyOperationsTotal.WithLabelValues(operation, policyTypeID, status).Inc()

	a1PolicyOperationDuration.WithLabelValues(operation, policyTypeID).Observe(duration.Seconds())

}

// UpdateA1PolicyTypesActive updates the number of active A1 policy types.

func (c *ORANMetricsCollector) UpdateA1PolicyTypesActive(ricID string, count int) {

	a1PolicyTypesActive.WithLabelValues(ricID).Set(float64(count))

}

// UpdateA1PolicyInstancesActive updates the number of active A1 policy instances.

func (c *ORANMetricsCollector) UpdateA1PolicyInstancesActive(policyTypeID, ricID, enforcementStatus string, count int) {

	a1PolicyInstancesActive.WithLabelValues(policyTypeID, ricID, enforcementStatus).Set(float64(count))

}

// UpdateA1CircuitBreakerState updates the A1 circuit breaker state.

func (c *ORANMetricsCollector) UpdateA1CircuitBreakerState(ricID, state string) {

	var stateValue float64

	switch state {

	case "closed":

		stateValue = 0

	case "open":

		stateValue = 1

	case "half-open":

		stateValue = 2

	}

	a1CircuitBreakerState.WithLabelValues(ricID).Set(stateValue)

}

// RecordA1RetryAttempt records an A1 retry attempt.

func (c *ORANMetricsCollector) RecordA1RetryAttempt(operation, finalStatus string) {

	a1RetryAttemptsTotal.WithLabelValues(operation, finalStatus).Inc()

}

// E2 Interface Metrics Methods.

// RecordE2MessageOperation records an E2AP message operation.

func (c *ORANMetricsCollector) RecordE2MessageOperation(messageType, nodeID, status string, duration time.Duration) {

	e2MessageOperationsTotal.WithLabelValues(messageType, nodeID, status).Inc()

	e2MessageOperationDuration.WithLabelValues(messageType, nodeID).Observe(duration.Seconds())

}

// UpdateE2SubscriptionsActive updates the number of active E2 subscriptions.

func (c *ORANMetricsCollector) UpdateE2SubscriptionsActive(nodeID, ranFunctionID, subscriptionType string, count int) {

	e2SubscriptionsActive.WithLabelValues(nodeID, ranFunctionID, subscriptionType).Set(float64(count))

}

// UpdateE2NodesConnected updates the number of connected E2 nodes.

func (c *ORANMetricsCollector) UpdateE2NodesConnected(nodeType, plmnID string, count int) {

	e2NodesConnected.WithLabelValues(nodeType, plmnID).Set(float64(count))

}

// RecordE2Indication records an E2 indication received.

func (c *ORANMetricsCollector) RecordE2Indication(nodeID, ranFunctionID, actionID, indicationType string) {

	e2IndicationsReceived.WithLabelValues(nodeID, ranFunctionID, actionID, indicationType).Inc()

}

// RecordE2ControlRequest records an E2 control request.

func (c *ORANMetricsCollector) RecordE2ControlRequest(nodeID, ranFunctionID, status string) {

	e2ControlRequestsTotal.WithLabelValues(nodeID, ranFunctionID, status).Inc()

}

// UpdateE2CircuitBreakerState updates the E2 circuit breaker state.

func (c *ORANMetricsCollector) UpdateE2CircuitBreakerState(nodeID, state string) {

	var stateValue float64

	switch state {

	case "closed":

		stateValue = 0

	case "open":

		stateValue = 1

	case "half-open":

		stateValue = 2

	}

	e2CircuitBreakerState.WithLabelValues(nodeID).Set(stateValue)

}

// O1 Interface Metrics Methods.

// RecordO1ConfigurationOperation records an O1 configuration operation.

func (c *ORANMetricsCollector) RecordO1ConfigurationOperation(operation, managedElement, status string, duration time.Duration) {

	o1ConfigurationOperationsTotal.WithLabelValues(operation, managedElement, status).Inc()

	o1ConfigurationOperationDuration.WithLabelValues(operation, managedElement).Observe(duration.Seconds())

}

// RecordO1FaultNotification records an O1 fault notification.

func (c *ORANMetricsCollector) RecordO1FaultNotification(managedElement, alarmType, severity string) {

	o1FaultNotificationsTotal.WithLabelValues(managedElement, alarmType, severity).Inc()

}

// UpdateO1ManagedElementsConnected updates the number of connected O1 managed elements.

func (c *ORANMetricsCollector) UpdateO1ManagedElementsConnected(elementType, vendor string, count int) {

	o1ManagedElementsConnected.WithLabelValues(elementType, vendor).Set(float64(count))

}

// O2 Interface Metrics Methods.

// RecordO2InfrastructureOperation records an O2 infrastructure operation.

func (c *ORANMetricsCollector) RecordO2InfrastructureOperation(operation, resourceType, status string, duration time.Duration) {

	o2InfrastructureOperationsTotal.WithLabelValues(operation, resourceType, status).Inc()

	o2InfrastructureOperationDuration.WithLabelValues(operation, resourceType).Observe(duration.Seconds())

}

// UpdateO2ResourceUtilization updates O2 resource utilization.

func (c *ORANMetricsCollector) UpdateO2ResourceUtilization(resourceType, resourceID, cluster string, utilizationPercent float64) {

	o2ResourceUtilization.WithLabelValues(resourceType, resourceID, cluster).Set(utilizationPercent)

}

// UpdateO2DeploymentStatus updates O2 deployment status.

func (c *ORANMetricsCollector) UpdateO2DeploymentStatus(deploymentID, applicationType, cluster, status string) {

	var statusValue float64

	switch status {

	case "failed":

		statusValue = 0

	case "deploying":

		statusValue = 1

	case "deployed":

		statusValue = 2

	case "terminated":

		statusValue = 3

	}

	o2DeploymentStatus.WithLabelValues(deploymentID, applicationType, cluster).Set(statusValue)

}

// General O-RAN Metrics Methods.

// RecordHealthCheck records an O-RAN health check.

func (c *ORANMetricsCollector) RecordHealthCheck(interfaceName, component, status string, duration time.Duration) {

	var statusValue float64

	switch status {

	case "unhealthy":

		statusValue = 0

	case "healthy":

		statusValue = 1

	case "degraded":

		statusValue = 2

	}

	oranHealthCheckStatus.WithLabelValues(interfaceName, component).Set(statusValue)

	oranHealthCheckDuration.WithLabelValues(interfaceName, component).Observe(duration.Seconds())

}

// UpdateDependencyStatus updates dependency status.

func (c *ORANMetricsCollector) UpdateDependencyStatus(dependencyName, dependencyType string, isHealthy bool) {

	var statusValue float64

	if isHealthy {

		statusValue = 1

	}

	oranDependencyStatus.WithLabelValues(dependencyName, dependencyType).Set(statusValue)

}

// RecordCircuitBreakerTrip records a circuit breaker trip.

func (c *ORANMetricsCollector) RecordCircuitBreakerTrip(interfaceName, reason string) {

	oranCircuitBreakerTrips.WithLabelValues(interfaceName, reason).Inc()

}

// UpdateConcurrentConnections updates concurrent connections count.

func (c *ORANMetricsCollector) UpdateConcurrentConnections(interfaceName, endpoint string, count int) {

	oranConcurrentConnections.WithLabelValues(interfaceName, endpoint).Set(float64(count))

}

// UpdateThroughput updates message throughput.

func (c *ORANMetricsCollector) UpdateThroughput(interfaceName, direction string, messagesPerSecond float64) {

	oranThroughput.WithLabelValues(interfaceName, direction).Set(messagesPerSecond)

}

// UpdateErrorRate updates error rate percentage.

func (c *ORANMetricsCollector) UpdateErrorRate(interfaceName, errorType string, errorRatePercent float64) {

	oranErrorRate.WithLabelValues(interfaceName, errorType).Set(errorRatePercent)

}

// Utility Methods.

// StartMetricsCollection starts periodic metrics collection.

func (c *ORANMetricsCollector) StartMetricsCollection(ctx context.Context, interval time.Duration) {

	ticker := time.NewTicker(interval)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			c.collectSystemMetrics()

		}

	}

}

// collectSystemMetrics collects system-level metrics.

func (c *ORANMetricsCollector) collectSystemMetrics() {

	// This method would collect system-level metrics.

	// Implementation would depend on the specific monitoring requirements.

	// and available system information.

}

// Reset resets all metrics (useful for testing).

func (c *ORANMetricsCollector) Reset() {

	// Reset counters by re-creating them.

	// Note: In production, this should be used carefully.

	a1PolicyOperationsTotal.Reset()

	e2MessageOperationsTotal.Reset()

	o1ConfigurationOperationsTotal.Reset()

	o2InfrastructureOperationsTotal.Reset()

	a1RetryAttemptsTotal.Reset()

	e2IndicationsReceived.Reset()

	e2ControlRequestsTotal.Reset()

	o1FaultNotificationsTotal.Reset()

	oranCircuitBreakerTrips.Reset()

}

// GetMetricsFamilies returns all registered metric families for debugging.

func (c *ORANMetricsCollector) GetMetricsFamilies() ([]*dto.MetricFamily, error) {

	return prometheus.DefaultGatherer.Gather()

}

// Helper function to convert int to string for labels.

func intToString(i int) string {

	return strconv.Itoa(i)

}

// Helper function to convert int64 to string for labels.

func int64ToString(i int64) string {

	return strconv.FormatInt(i, 10)

}
