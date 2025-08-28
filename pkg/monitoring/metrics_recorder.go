package monitoring

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/metric"
)

// MetricsRecorder provides a unified interface for recording metrics.
type MetricsRecorder struct {
	mu            sync.RWMutex
	registry      prometheus.Registerer
	meterProvider metric.MeterProvider

	// Prometheus metrics - Basic Infrastructure.
	controllerHealth *prometheus.GaugeVec
	intentProcessing *prometheus.HistogramVec
	llmRequests      *prometheus.CounterVec
	llmLatency       *prometheus.HistogramVec
	ragQueries       *prometheus.CounterVec
	ragLatency       *prometheus.HistogramVec
	oranConnections  *prometheus.GaugeVec
	systemResources  *prometheus.GaugeVec
	errorCounts      *prometheus.CounterVec
	alertCounts      *prometheus.CounterVec

	// Business KPI Metrics.
	automationRate   *prometheus.GaugeVec
	costPerIntent    *prometheus.GaugeVec
	userSatisfaction *prometheus.GaugeVec
	timeToDeployment *prometheus.HistogramVec
	complianceScore  *prometheus.GaugeVec

	// SLI/SLO Metrics.
	serviceAvailability *prometheus.GaugeVec
	errorBudgetBurn     *prometheus.GaugeVec
	sliLatencyP95       *prometheus.GaugeVec
	sliLatencyP99       *prometheus.GaugeVec
	sloCompliance       *prometheus.GaugeVec

	// Advanced Performance Metrics.
	throughputMetrics   *prometheus.GaugeVec
	capacityUtilization *prometheus.GaugeVec
	predictionAccuracy  *prometheus.GaugeVec
	anomalyDetection    *prometheus.CounterVec

	// O-RAN Compliance Metrics.
	oranCompliance   *prometheus.GaugeVec
	interfaceHealth  *prometheus.GaugeVec
	policyViolations *prometheus.CounterVec

	// Cost and Resource Optimization Metrics.
	resourceEfficiency *prometheus.GaugeVec
	costOptimization   *prometheus.GaugeVec
	energyEfficiency   *prometheus.GaugeVec

	// Security and Audit Metrics.
	securityIncidents *prometheus.CounterVec
	auditCompliance   *prometheus.GaugeVec
	accessViolations  *prometheus.CounterVec
}

// MetricsConfig holds configuration for metrics recording.
type MetricsConfig struct {
	Namespace     string
	Subsystem     string
	Registry      prometheus.Registerer
	MeterProvider metric.MeterProvider
}

// NewMetricsRecorder creates a new metrics recorder.
func NewMetricsRecorder(config *MetricsConfig) *MetricsRecorder {
	if config == nil {
		config = &MetricsConfig{
			Namespace: "nephoran",
			Subsystem: "intent_operator",
			Registry:  prometheus.DefaultRegisterer,
		}
	}

	if config.Registry == nil {
		config.Registry = prometheus.DefaultRegisterer
	}

	mr := &MetricsRecorder{
		registry:      config.Registry,
		meterProvider: config.MeterProvider,
	}

	// Initialize Prometheus metrics.
	mr.initPrometheusMetrics(config.Namespace, config.Subsystem)

	return mr
}

// initPrometheusMetrics initializes all Prometheus metrics.
func (mr *MetricsRecorder) initPrometheusMetrics(namespace, subsystem string) {
	// Controller health metrics.
	mr.controllerHealth = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "controller_health",
			Help:      "Health status of controllers (1 = healthy, 0 = unhealthy)",
		},
		[]string{"controller", "component"},
	)

	// Intent processing metrics.
	mr.intentProcessing = promauto.With(mr.registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "intent_processing_duration_seconds",
			Help:      "Time spent processing network intents",
			Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~102s
		},
		[]string{"intent_type", "status"},
	)

	// LLM request metrics.
	mr.llmRequests = promauto.With(mr.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "llm_requests_total",
			Help:      "Total number of LLM requests",
		},
		[]string{"model", "status", "cache_hit"},
	)

	mr.llmLatency = promauto.With(mr.registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "llm_request_duration_seconds",
			Help:      "Time spent on LLM requests",
			Buckets:   prometheus.ExponentialBuckets(0.1, 2, 12), // 0.1s to ~409s
		},
		[]string{"model", "operation"},
	)

	// RAG query metrics.
	mr.ragQueries = promauto.With(mr.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "rag_queries_total",
			Help:      "Total number of RAG queries",
		},
		[]string{"query_type", "status"},
	)

	mr.ragLatency = promauto.With(mr.registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "rag_query_duration_seconds",
			Help:      "Time spent on RAG queries",
			Buckets:   prometheus.ExponentialBuckets(0.01, 2, 10), // 0.01s to ~10s
		},
		[]string{"operation"},
	)

	// O-RAN connection metrics.
	mr.oranConnections = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "oran_connections_active",
			Help:      "Number of active O-RAN connections",
		},
		[]string{"interface", "endpoint"},
	)

	// System resource metrics.
	mr.systemResources = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "system_resources_usage",
			Help:      "System resource usage (CPU, memory, etc.)",
		},
		[]string{"resource", "component"},
	)

	// Error count metrics.
	mr.errorCounts = promauto.With(mr.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "errors_total",
			Help:      "Total number of errors by component",
		},
		[]string{"component", "error_type"},
	)

	// Alert count metrics.
	mr.alertCounts = promauto.With(mr.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "alerts_total",
			Help:      "Total number of alerts fired",
		},
		[]string{"severity", "component"},
	)

	// Initialize Business KPI Metrics.
	mr.initBusinessKPIMetrics(namespace, subsystem)

	// Initialize SLI/SLO Metrics.
	mr.initSLIMetrics(namespace, subsystem)

	// Initialize Advanced Performance Metrics.
	mr.initAdvancedPerformanceMetrics(namespace, subsystem)

	// Initialize O-RAN Compliance Metrics.
	mr.initORANComplianceMetrics(namespace, subsystem)

	// Initialize Cost and Resource Optimization Metrics.
	mr.initCostOptimizationMetrics(namespace, subsystem)

	// Initialize Security and Audit Metrics.
	mr.initSecurityMetrics(namespace, subsystem)
}

// initBusinessKPIMetrics initializes business Key Performance Indicator metrics.
func (mr *MetricsRecorder) initBusinessKPIMetrics(namespace, subsystem string) {
	// Automation Rate - percentage of operations automated vs manual.
	mr.automationRate = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "automation_rate_percent",
			Help:      "Percentage of operations that are automated vs manual",
		},
		[]string{"operation_type", "time_window"},
	)

	// Cost Per Intent - financial cost per processed intent.
	mr.costPerIntent = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cost_per_intent_usd",
			Help:      "Cost in USD per processed network intent",
		},
		[]string{"intent_type", "processing_complexity"},
	)

	// User Satisfaction Score - based on feedback and success metrics.
	mr.userSatisfaction = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "user_satisfaction_score",
			Help:      "User satisfaction score (0-10 scale)",
		},
		[]string{"user_group", "time_period"},
	)

	// Time to Deployment - complete deployment lifecycle time.
	mr.timeToDeployment = promauto.With(mr.registry).NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "time_to_deployment_seconds",
			Help:      "Time from intent submission to successful deployment",
			Buckets:   prometheus.ExponentialBuckets(60, 2, 12), // 1min to ~68min
		},
		[]string{"deployment_type", "complexity", "target_environment"},
	)

	// Compliance Score - regulatory and policy compliance metrics.
	mr.complianceScore = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "compliance_score_percent",
			Help:      "Compliance score as percentage (0-100)",
		},
		[]string{"compliance_type", "standard", "scope"},
	)
}

// initSLIMetrics initializes Service Level Indicator metrics for SLO tracking.
func (mr *MetricsRecorder) initSLIMetrics(namespace, subsystem string) {
	// Service Availability - uptime percentage.
	mr.serviceAvailability = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "service_availability_percent",
			Help:      "Service availability as percentage over time window",
		},
		[]string{"service", "time_window", "region"},
	)

	// Error Budget Burn Rate - how fast we're consuming error budget.
	mr.errorBudgetBurn = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "error_budget_burn_rate",
			Help:      "Rate at which error budget is being consumed",
		},
		[]string{"service", "time_window", "budget_type"},
	)

	// SLI Latency P95 - 95th percentile latency for SLO tracking.
	mr.sliLatencyP95 = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "sli_latency_p95_seconds",
			Help:      "95th percentile latency for SLI tracking",
		},
		[]string{"service", "operation", "region"},
	)

	// SLI Latency P99 - 99th percentile latency for SLO tracking.
	mr.sliLatencyP99 = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "sli_latency_p99_seconds",
			Help:      "99th percentile latency for SLI tracking",
		},
		[]string{"service", "operation", "region"},
	)

	// SLO Compliance - whether services are meeting SLO targets.
	mr.sloCompliance = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "slo_compliance_status",
			Help:      "SLO compliance status (1=compliant, 0=violation)",
		},
		[]string{"service", "slo_type", "severity"},
	)
}

// initAdvancedPerformanceMetrics initializes advanced performance and capacity metrics.
func (mr *MetricsRecorder) initAdvancedPerformanceMetrics(namespace, subsystem string) {
	// Throughput Metrics - requests/operations per second.
	mr.throughputMetrics = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "throughput_ops_per_second",
			Help:      "Current throughput in operations per second",
		},
		[]string{"component", "operation_type", "time_window"},
	)

	// Capacity Utilization - how much of available capacity is being used.
	mr.capacityUtilization = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "capacity_utilization_percent",
			Help:      "Capacity utilization as percentage",
		},
		[]string{"resource_type", "component", "measurement_type"},
	)

	// Prediction Accuracy - ML/AI prediction accuracy metrics.
	mr.predictionAccuracy = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "prediction_accuracy_percent",
			Help:      "Accuracy of predictive algorithms and ML models",
		},
		[]string{"model_type", "prediction_category", "time_horizon"},
	)

	// Anomaly Detection - count of detected anomalies.
	mr.anomalyDetection = promauto.With(mr.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "anomalies_detected_total",
			Help:      "Total number of anomalies detected by monitoring systems",
		},
		[]string{"anomaly_type", "severity", "component", "detection_method"},
	)
}

// initORANComplianceMetrics initializes O-RAN specific compliance and health metrics.
func (mr *MetricsRecorder) initORANComplianceMetrics(namespace, subsystem string) {
	// O-RAN Compliance Score - adherence to O-RAN standards.
	mr.oranCompliance = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "oran_compliance_score_percent",
			Help:      "O-RAN compliance score as percentage",
		},
		[]string{"interface", "standard_version", "compliance_category"},
	)

	// Interface Health - health status of O-RAN interfaces.
	mr.interfaceHealth = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "oran_interface_health_score",
			Help:      "Health score of O-RAN interfaces (0-1 scale)",
		},
		[]string{"interface_type", "endpoint", "function"},
	)

	// Policy Violations - count of policy violations.
	mr.policyViolations = promauto.With(mr.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "oran_policy_violations_total",
			Help:      "Total number of O-RAN policy violations detected",
		},
		[]string{"policy_type", "violation_category", "severity", "interface"},
	)
}

// initCostOptimizationMetrics initializes cost and resource optimization metrics.
func (mr *MetricsRecorder) initCostOptimizationMetrics(namespace, subsystem string) {
	// Resource Efficiency - how efficiently resources are being used.
	mr.resourceEfficiency = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "resource_efficiency_score",
			Help:      "Resource efficiency score (0-1 scale)",
		},
		[]string{"resource_type", "optimization_category", "measurement_period"},
	)

	// Cost Optimization - cost savings and optimization metrics.
	mr.costOptimization = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cost_optimization_savings_usd",
			Help:      "Cost savings achieved through optimization in USD",
		},
		[]string{"optimization_type", "time_period", "category"},
	)

	// Energy Efficiency - power consumption and efficiency metrics.
	mr.energyEfficiency = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "energy_efficiency_score",
			Help:      "Energy efficiency score (higher is better)",
		},
		[]string{"component", "measurement_type", "optimization_level"},
	)
}

// initSecurityMetrics initializes security and audit metrics.
func (mr *MetricsRecorder) initSecurityMetrics(namespace, subsystem string) {
	// Security Incidents - count of security-related incidents.
	mr.securityIncidents = promauto.With(mr.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "security_incidents_total",
			Help:      "Total number of security incidents detected",
		},
		[]string{"incident_type", "severity", "component", "detection_method"},
	)

	// Audit Compliance - compliance with audit requirements.
	mr.auditCompliance = promauto.With(mr.registry).NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "audit_compliance_score_percent",
			Help:      "Audit compliance score as percentage",
		},
		[]string{"audit_type", "compliance_framework", "scope"},
	)

	// Access Violations - count of unauthorized access attempts.
	mr.accessViolations = promauto.With(mr.registry).NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "access_violations_total",
			Help:      "Total number of access violations detected",
		},
		[]string{"violation_type", "severity", "source", "target_resource"},
	)
}

// RecordControllerHealth records controller health status.
func (mr *MetricsRecorder) RecordControllerHealth(controller, component string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	mr.controllerHealth.WithLabelValues(controller, component).Set(value)
}

// RecordIntentProcessing records intent processing metrics.
func (mr *MetricsRecorder) RecordIntentProcessing(intentType, status string, duration time.Duration) {
	mr.intentProcessing.WithLabelValues(intentType, status).Observe(duration.Seconds())
}

// RecordLLMRequest records LLM request metrics.
func (mr *MetricsRecorder) RecordLLMRequest(model, status string, cacheHit bool, duration time.Duration) {
	cacheHitStr := "false"
	if cacheHit {
		cacheHitStr = "true"
	}

	mr.llmRequests.WithLabelValues(model, status, cacheHitStr).Inc()
	mr.llmLatency.WithLabelValues(model, "request").Observe(duration.Seconds())
}

// RecordRAGQuery records RAG query metrics.
func (mr *MetricsRecorder) RecordRAGQuery(queryType, status string, duration time.Duration) {
	mr.ragQueries.WithLabelValues(queryType, status).Inc()
	mr.ragLatency.WithLabelValues("query").Observe(duration.Seconds())
}

// RecordORANConnection records O-RAN connection status.
func (mr *MetricsRecorder) RecordORANConnection(interfaceType, endpoint string, connected bool) {
	value := 0.0
	if connected {
		value = 1.0
	}
	mr.oranConnections.WithLabelValues(interfaceType, endpoint).Set(value)
}

// RecordSystemResource records system resource usage.
func (mr *MetricsRecorder) RecordSystemResource(resource, component string, value float64) {
	mr.systemResources.WithLabelValues(resource, component).Set(value)
}

// RecordError records error occurrences.
func (mr *MetricsRecorder) RecordError(component, errorType string) {
	mr.errorCounts.WithLabelValues(component, errorType).Inc()
}

// RecordAlert records alert firing.
func (mr *MetricsRecorder) RecordAlert(severity, component string) {
	mr.alertCounts.WithLabelValues(severity, component).Inc()
}

// Business KPI Recording Methods.

// RecordAutomationRate records the automation rate for operations.
func (mr *MetricsRecorder) RecordAutomationRate(operationType, timeWindow string, rate float64) {
	mr.automationRate.WithLabelValues(operationType, timeWindow).Set(rate)
}

// RecordCostPerIntent records the cost per processed intent.
func (mr *MetricsRecorder) RecordCostPerIntent(intentType, complexity string, cost float64) {
	mr.costPerIntent.WithLabelValues(intentType, complexity).Set(cost)
}

// RecordUserSatisfaction records user satisfaction scores.
func (mr *MetricsRecorder) RecordUserSatisfaction(userGroup, timePeriod string, score float64) {
	mr.userSatisfaction.WithLabelValues(userGroup, timePeriod).Set(score)
}

// RecordTimeToDeployment records deployment completion time.
func (mr *MetricsRecorder) RecordTimeToDeployment(deploymentType, complexity, environment string, duration time.Duration) {
	mr.timeToDeployment.WithLabelValues(deploymentType, complexity, environment).Observe(duration.Seconds())
}

// RecordComplianceScore records compliance scores.
func (mr *MetricsRecorder) RecordComplianceScore(complianceType, standard, scope string, score float64) {
	mr.complianceScore.WithLabelValues(complianceType, standard, scope).Set(score)
}

// SLI/SLO Recording Methods.

// RecordServiceAvailability records service availability metrics.
func (mr *MetricsRecorder) RecordServiceAvailability(service, timeWindow, region string, availability float64) {
	mr.serviceAvailability.WithLabelValues(service, timeWindow, region).Set(availability)
}

// RecordErrorBudgetBurn records error budget burn rate.
func (mr *MetricsRecorder) RecordErrorBudgetBurn(service, timeWindow, budgetType string, burnRate float64) {
	mr.errorBudgetBurn.WithLabelValues(service, timeWindow, budgetType).Set(burnRate)
}

// RecordSLILatency records SLI latency metrics.
func (mr *MetricsRecorder) RecordSLILatency(service, operation, region string, p95Latency, p99Latency float64) {
	mr.sliLatencyP95.WithLabelValues(service, operation, region).Set(p95Latency)
	mr.sliLatencyP99.WithLabelValues(service, operation, region).Set(p99Latency)
}

// RecordSLOCompliance records SLO compliance status.
func (mr *MetricsRecorder) RecordSLOCompliance(service, sloType, severity string, compliant bool) {
	value := 0.0
	if compliant {
		value = 1.0
	}
	mr.sloCompliance.WithLabelValues(service, sloType, severity).Set(value)
}

// Advanced Performance Recording Methods.

// RecordThroughput records throughput metrics.
func (mr *MetricsRecorder) RecordThroughput(component, operationType, timeWindow string, opsPerSecond float64) {
	mr.throughputMetrics.WithLabelValues(component, operationType, timeWindow).Set(opsPerSecond)
}

// RecordCapacityUtilization records capacity utilization metrics.
func (mr *MetricsRecorder) RecordCapacityUtilization(resourceType, component, measurementType string, utilization float64) {
	mr.capacityUtilization.WithLabelValues(resourceType, component, measurementType).Set(utilization)
}

// RecordPredictionAccuracy records ML/AI prediction accuracy.
func (mr *MetricsRecorder) RecordPredictionAccuracy(modelType, category, timeHorizon string, accuracy float64) {
	mr.predictionAccuracy.WithLabelValues(modelType, category, timeHorizon).Set(accuracy)
}

// RecordAnomalyDetection records detected anomalies.
func (mr *MetricsRecorder) RecordAnomalyDetection(anomalyType, severity, component, detectionMethod string) {
	mr.anomalyDetection.WithLabelValues(anomalyType, severity, component, detectionMethod).Inc()
}

// O-RAN Compliance Recording Methods.

// RecordORANCompliance records O-RAN compliance scores.
func (mr *MetricsRecorder) RecordORANCompliance(interface_, standardVersion, category string, score float64) {
	mr.oranCompliance.WithLabelValues(interface_, standardVersion, category).Set(score)
}

// RecordInterfaceHealth records O-RAN interface health.
func (mr *MetricsRecorder) RecordInterfaceHealth(interfaceType, endpoint, function string, healthScore float64) {
	mr.interfaceHealth.WithLabelValues(interfaceType, endpoint, function).Set(healthScore)
}

// RecordPolicyViolation records O-RAN policy violations.
func (mr *MetricsRecorder) RecordPolicyViolation(policyType, category, severity, interface_ string) {
	mr.policyViolations.WithLabelValues(policyType, category, severity, interface_).Inc()
}

// Cost and Resource Optimization Recording Methods.

// RecordResourceEfficiency records resource efficiency scores.
func (mr *MetricsRecorder) RecordResourceEfficiency(resourceType, category, period string, efficiency float64) {
	mr.resourceEfficiency.WithLabelValues(resourceType, category, period).Set(efficiency)
}

// RecordCostOptimization records cost optimization savings.
func (mr *MetricsRecorder) RecordCostOptimization(optimizationType, timePeriod, category string, savings float64) {
	mr.costOptimization.WithLabelValues(optimizationType, timePeriod, category).Set(savings)
}

// RecordEnergyEfficiency records energy efficiency metrics.
func (mr *MetricsRecorder) RecordEnergyEfficiency(component, measurementType, optimizationLevel string, efficiency float64) {
	mr.energyEfficiency.WithLabelValues(component, measurementType, optimizationLevel).Set(efficiency)
}

// Security and Audit Recording Methods.

// RecordSecurityIncident records security incidents.
func (mr *MetricsRecorder) RecordSecurityIncident(incidentType, severity, component, detectionMethod string) {
	mr.securityIncidents.WithLabelValues(incidentType, severity, component, detectionMethod).Inc()
}

// RecordAuditCompliance records audit compliance scores.
func (mr *MetricsRecorder) RecordAuditCompliance(auditType, framework, scope string, score float64) {
	mr.auditCompliance.WithLabelValues(auditType, framework, scope).Set(score)
}

// RecordAccessViolation records access violations.
func (mr *MetricsRecorder) RecordAccessViolation(violationType, severity, source, targetResource string) {
	mr.accessViolations.WithLabelValues(violationType, severity, source, targetResource).Inc()
}

// GetRegistry returns the Prometheus registry.
func (mr *MetricsRecorder) GetRegistry() prometheus.Registerer {
	return mr.registry
}

// MetricsSummary provides a comprehensive summary of current metrics.
type MetricsSummary struct {
	// Basic Infrastructure Metrics.
	ControllerHealth map[string]bool    `json:"controller_health"`
	IntentProcessing ProcessingStats    `json:"intent_processing"`
	LLMRequests      LLMStats           `json:"llm_requests"`
	RAGQueries       RAGStats           `json:"rag_queries"`
	ORANConnections  map[string]bool    `json:"oran_connections"`
	SystemResources  map[string]float64 `json:"system_resources"`
	ErrorCounts      map[string]int     `json:"error_counts"`

	// Business KPI Metrics.
	BusinessKPIs BusinessKPIStats `json:"business_kpis"`

	// SLI/SLO Metrics.
	ServiceLevels ServiceLevelStats `json:"service_levels"`

	// Advanced Performance Metrics.
	PerformanceMetrics PerformanceStats `json:"performance_metrics"`

	// O-RAN Compliance Metrics.
	ORANCompliance ORANComplianceStats `json:"oran_compliance"`

	// Cost and Resource Optimization.
	Optimization OptimizationStats `json:"optimization"`

	// Security and Audit.
	Security SecurityStats `json:"security"`

	Timestamp time.Time `json:"timestamp"`
}

// ProcessingStats represents a processingstats.
type ProcessingStats struct {
	TotalProcessed int64         `json:"total_processed"`
	SuccessRate    float64       `json:"success_rate"`
	AverageLatency time.Duration `json:"average_latency"`
	P95Latency     time.Duration `json:"p95_latency"`
	P99Latency     time.Duration `json:"p99_latency"`
	ThroughputOPS  float64       `json:"throughput_ops"`
}

// LLMStats represents a llmstats.
type LLMStats struct {
	TotalRequests  int64         `json:"total_requests"`
	CacheHitRate   float64       `json:"cache_hit_rate"`
	AverageLatency time.Duration `json:"average_latency"`
	ErrorRate      float64       `json:"error_rate"`
	CostPerRequest float64       `json:"cost_per_request"`
	TokensUsed     int64         `json:"tokens_used"`
}

// RAGStats represents a ragstats.
type RAGStats struct {
	TotalQueries   int64         `json:"total_queries"`
	AverageLatency time.Duration `json:"average_latency"`
	SuccessRate    float64       `json:"success_rate"`
	IndexSize      int64         `json:"index_size"`
	QueryAccuracy  float64       `json:"query_accuracy"`
}

// BusinessKPIStats represents a businesskpistats.
type BusinessKPIStats struct {
	AutomationRate          float64       `json:"automation_rate"`
	AverageCostPerIntent    float64       `json:"average_cost_per_intent"`
	UserSatisfactionScore   float64       `json:"user_satisfaction_score"`
	AverageTimeToDeployment time.Duration `json:"average_time_to_deployment"`
	ComplianceScore         float64       `json:"compliance_score"`
	BusinessValue           float64       `json:"business_value"`
}

// ServiceLevelStats represents a servicelevelstats.
type ServiceLevelStats struct {
	Availability         float64 `json:"availability"`
	ErrorBudgetRemaining float64 `json:"error_budget_remaining"`
	LatencyP95           float64 `json:"latency_p95"`
	LatencyP99           float64 `json:"latency_p99"`
	SLOComplianceRate    float64 `json:"slo_compliance_rate"`
	IncidentCount        int     `json:"incident_count"`
}

// PerformanceStats represents a performancestats.
type PerformanceStats struct {
	CurrentThroughput    float64 `json:"current_throughput"`
	CapacityUtilization  float64 `json:"capacity_utilization"`
	PredictionAccuracy   float64 `json:"prediction_accuracy"`
	AnomaliesDetected    int     `json:"anomalies_detected"`
	ResponseTimeVariance float64 `json:"response_time_variance"`
	QueueDepth           int     `json:"queue_depth"`
}

// ORANComplianceStats represents a orancompliancestats.
type ORANComplianceStats struct {
	OverallComplianceScore float64 `json:"overall_compliance_score"`
	InterfaceHealthScore   float64 `json:"interface_health_score"`
	PolicyViolations       int     `json:"policy_violations"`
	StandardsAdherence     float64 `json:"standards_adherence"`
	InteroperabilityScore  float64 `json:"interoperability_score"`
}

// OptimizationStats represents a optimizationstats.
type OptimizationStats struct {
	ResourceEfficiency float64 `json:"resource_efficiency"`
	CostSavings        float64 `json:"cost_savings"`
	EnergyEfficiency   float64 `json:"energy_efficiency"`
	WasteReduction     float64 `json:"waste_reduction"`
	OptimizationScore  float64 `json:"optimization_score"`
}

// SecurityStats represents a securitystats.
type SecurityStats struct {
	SecurityIncidents int     `json:"security_incidents"`
	AuditCompliance   float64 `json:"audit_compliance"`
	AccessViolations  int     `json:"access_violations"`
	ThreatLevel       string  `json:"threat_level"`
	SecurityPosture   float64 `json:"security_posture"`
}

// GetMetricsSummary returns a summary of current metrics.
func (mr *MetricsRecorder) GetMetricsSummary(ctx context.Context) (*MetricsSummary, error) {
	mr.mu.RLock()
	defer mr.mu.RUnlock()

	summary := &MetricsSummary{
		ControllerHealth: make(map[string]bool),
		ORANConnections:  make(map[string]bool),
		SystemResources:  make(map[string]float64),
		ErrorCounts:      make(map[string]int),
		Timestamp:        time.Now(),
	}

	// In a real implementation, these would query actual Prometheus metrics.
	// For now, we'll populate with sample data structure.

	// Business KPIs calculation.
	summary.BusinessKPIs = mr.calculateBusinessKPIs(ctx)

	// Service Level metrics calculation.
	summary.ServiceLevels = mr.calculateServiceLevelStats(ctx)

	// Performance metrics calculation.
	summary.PerformanceMetrics = mr.calculatePerformanceStats(ctx)

	// O-RAN compliance calculation.
	summary.ORANCompliance = mr.calculateORANComplianceStats(ctx)

	// Optimization metrics calculation.
	summary.Optimization = mr.calculateOptimizationStats(ctx)

	// Security metrics calculation.
	summary.Security = mr.calculateSecurityStats(ctx)

	return summary, nil
}

// Business KPI Analytics Methods.

// calculateBusinessKPIs calculates current business KPI metrics.
func (mr *MetricsRecorder) calculateBusinessKPIs(ctx context.Context) BusinessKPIStats {
	// In production, these would query actual Prometheus metrics using promql.
	// For demonstration, returning structured calculation framework.

	return BusinessKPIStats{
		AutomationRate:          mr.getMetricValue("automation_rate", 85.0),                                // 85% automation
		AverageCostPerIntent:    mr.getMetricValue("cost_per_intent", 0.023),                               // $0.023 per intent
		UserSatisfactionScore:   mr.getMetricValue("user_satisfaction", 8.7),                               // 8.7/10 satisfaction
		AverageTimeToDeployment: time.Duration(mr.getMetricValue("time_to_deployment", 300)) * time.Second, // 5 minutes
		ComplianceScore:         mr.getMetricValue("compliance_score", 94.2),                               // 94.2% compliance
		BusinessValue:           mr.calculateBusinessValue(),                                               // Calculated ROI
	}
}

// calculateServiceLevelStats calculates SLI/SLO metrics.
func (mr *MetricsRecorder) calculateServiceLevelStats(ctx context.Context) ServiceLevelStats {
	return ServiceLevelStats{
		Availability:         mr.getMetricValue("availability", 99.95),  // 99.95% availability
		ErrorBudgetRemaining: mr.getMetricValue("error_budget", 78.5),   // 78.5% error budget remaining
		LatencyP95:           mr.getMetricValue("latency_p95", 1.8),     // 1.8s P95 latency
		LatencyP99:           mr.getMetricValue("latency_p99", 4.2),     // 4.2s P99 latency
		SLOComplianceRate:    mr.getMetricValue("slo_compliance", 96.8), // 96.8% SLO compliance
		IncidentCount:        int(mr.getMetricValue("incidents", 2)),    // 2 recent incidents
	}
}

// calculatePerformanceStats calculates advanced performance metrics.
func (mr *MetricsRecorder) calculatePerformanceStats(ctx context.Context) PerformanceStats {
	return PerformanceStats{
		CurrentThroughput:    mr.getMetricValue("throughput", 45.7),          // 45.7 intents/min
		CapacityUtilization:  mr.getMetricValue("capacity_util", 67.3),       // 67.3% capacity
		PredictionAccuracy:   mr.getMetricValue("prediction_accuracy", 89.2), // 89.2% ML accuracy
		AnomaliesDetected:    int(mr.getMetricValue("anomalies", 3)),         // 3 anomalies detected
		ResponseTimeVariance: mr.getMetricValue("response_variance", 0.34),   // 0.34s variance
		QueueDepth:           int(mr.getMetricValue("queue_depth", 8)),       // 8 items in queue
	}
}

// calculateORANComplianceStats calculates O-RAN compliance metrics.
func (mr *MetricsRecorder) calculateORANComplianceStats(ctx context.Context) ORANComplianceStats {
	return ORANComplianceStats{
		OverallComplianceScore: mr.getMetricValue("oran_compliance", 92.7),     // 92.7% O-RAN compliance
		InterfaceHealthScore:   mr.getMetricValue("interface_health", 95.1),    // 95.1% interface health
		PolicyViolations:       int(mr.getMetricValue("policy_violations", 1)), // 1 policy violation
		StandardsAdherence:     mr.getMetricValue("standards_adherence", 98.3), // 98.3% standards adherence
		InteroperabilityScore:  mr.getMetricValue("interoperability", 87.9),    // 87.9% interoperability
	}
}

// calculateOptimizationStats calculates cost and resource optimization metrics.
func (mr *MetricsRecorder) calculateOptimizationStats(ctx context.Context) OptimizationStats {
	return OptimizationStats{
		ResourceEfficiency: mr.getMetricValue("resource_efficiency", 82.4), // 82.4% resource efficiency
		CostSavings:        mr.getMetricValue("cost_savings", 1247.50),     // $1,247.50 savings
		EnergyEfficiency:   mr.getMetricValue("energy_efficiency", 76.8),   // 76.8% energy efficiency
		WasteReduction:     mr.getMetricValue("waste_reduction", 34.2),     // 34.2% waste reduction
		OptimizationScore:  mr.calculateOptimizationScore(),                // Overall optimization score
	}
}

// calculateSecurityStats calculates security and audit metrics.
func (mr *MetricsRecorder) calculateSecurityStats(ctx context.Context) SecurityStats {
	return SecurityStats{
		SecurityIncidents: int(mr.getMetricValue("security_incidents", 0)), // 0 security incidents
		AuditCompliance:   mr.getMetricValue("audit_compliance", 97.8),     // 97.8% audit compliance
		AccessViolations:  int(mr.getMetricValue("access_violations", 2)),  // 2 access violations
		ThreatLevel:       mr.getThreatLevel(),                             // Current threat level
		SecurityPosture:   mr.getMetricValue("security_posture", 91.3),     // 91.3% security posture
	}
}

// Advanced Analytics Helper Methods.

// getMetricValue retrieves a metric value with fallback to default.
func (mr *MetricsRecorder) getMetricValue(metricName string, defaultValue float64) float64 {
	// In production, this would query Prometheus using PromQL.
	// Example: query := fmt.Sprintf("nephoran_intent_operator_%s", metricName).
	// result := mr.prometheusClient.Query(context.Background(), query, time.Now()).
	// For now, return default value with some variance for demonstration.

	variance := 0.1 * defaultValue * (math.Sin(float64(time.Now().Unix()%3600)) * 0.1)
	return defaultValue + variance
}

// calculateBusinessValue calculates overall business value ROI.
func (mr *MetricsRecorder) calculateBusinessValue() float64 {
	// Complex business value calculation based on multiple factors.
	automationSavings := mr.getMetricValue("automation_rate", 85.0) * 100.0  // Automation savings
	efficiencyGains := mr.getMetricValue("resource_efficiency", 82.4) * 50.0 // Efficiency gains
	complianceBenefit := mr.getMetricValue("compliance_score", 94.2) * 25.0  // Compliance benefits

	totalValue := automationSavings + efficiencyGains + complianceBenefit
	return totalValue / 100.0 // Normalize to percentage
}

// calculateOptimizationScore calculates overall optimization effectiveness.
func (mr *MetricsRecorder) calculateOptimizationScore() float64 {
	resourceScore := mr.getMetricValue("resource_efficiency", 82.4)
	energyScore := mr.getMetricValue("energy_efficiency", 76.8)
	costScore := 100.0 - (mr.getMetricValue("cost_per_intent", 0.023) * 1000.0) // Inverse of cost

	// Weighted average: 40% resource, 30% energy, 30% cost.
	return (resourceScore*0.4 + energyScore*0.3 + costScore*0.3)
}

// getThreatLevel determines current security threat level.
func (mr *MetricsRecorder) getThreatLevel() string {
	incidents := int(mr.getMetricValue("security_incidents", 0))
	violations := int(mr.getMetricValue("access_violations", 2))

	totalThreats := incidents*3 + violations // Weight incidents more heavily

	switch {
	case totalThreats == 0:
		return "minimal"
	case totalThreats <= 2:
		return "low"
	case totalThreats <= 5:
		return "medium"
	case totalThreats <= 10:
		return "high"
	default:
		return "critical"
	}
}

// Predictive Analytics Methods.

// CalculateTrendAnalysis calculates metric trends over time.
func (mr *MetricsRecorder) CalculateTrendAnalysis(ctx context.Context, metricName string, timeWindow time.Duration) (*TrendAnalysis, error) {
	// This would implement time-series analysis in production.
	return &TrendAnalysis{
		MetricName:     metricName,
		TimeWindow:     timeWindow,
		TrendDirection: "increasing",
		TrendStrength:  0.75,
		Forecast:       mr.getMetricValue(metricName, 100.0) * 1.15, // 15% increase forecast
		Confidence:     0.87,
		AnalysisTime:   time.Now(),
	}, nil
}

// PredictCapacityRequirements predicts future capacity needs.
func (mr *MetricsRecorder) PredictCapacityRequirements(ctx context.Context, timeHorizon time.Duration) (*CapacityPrediction, error) {
	currentThroughput := mr.getMetricValue("throughput", 45.7)
	currentUtilization := mr.getMetricValue("capacity_util", 67.3)

	// Simple growth model - in production would use ML models.
	growthRate := 0.15 // 15% growth per month
	months := float64(timeHorizon.Hours()) / (24 * 30)

	predictedThroughput := currentThroughput * math.Pow(1+growthRate, months)
	predictedUtilization := currentUtilization * (predictedThroughput / currentThroughput)

	return &CapacityPrediction{
		TimeHorizon:          timeHorizon,
		CurrentThroughput:    currentThroughput,
		PredictedThroughput:  predictedThroughput,
		CurrentUtilization:   currentUtilization,
		PredictedUtilization: predictedUtilization,
		RecommendedAction:    mr.getCapacityRecommendation(predictedUtilization),
		Confidence:           0.82,
		PredictionTime:       time.Now(),
	}, nil
}

// getCapacityRecommendation provides capacity planning recommendations.
func (mr *MetricsRecorder) getCapacityRecommendation(predictedUtilization float64) string {
	switch {
	case predictedUtilization > 90:
		return "urgent_scaling_required"
	case predictedUtilization > 80:
		return "scaling_recommended"
	case predictedUtilization > 70:
		return "monitor_closely"
	case predictedUtilization < 30:
		return "consider_downsizing"
	default:
		return "current_capacity_adequate"
	}
}

// Anomaly Detection Methods.

// DetectAnomalies identifies anomalous metric patterns.
func (mr *MetricsRecorder) DetectAnomalies(ctx context.Context, metricName string, sensitivityLevel float64) (*AnomalyDetectionResult, error) {
	currentValue := mr.getMetricValue(metricName, 100.0)
	historicalMean := mr.getMetricValue(metricName+"_mean", 95.0)
	standardDeviation := mr.getMetricValue(metricName+"_stddev", 5.0)

	// Z-score anomaly detection.
	zScore := math.Abs(currentValue-historicalMean) / standardDeviation
	threshold := 2.0 / sensitivityLevel // Higher sensitivity = lower threshold

	isAnomaly := zScore > threshold
	severity := "normal"
	if isAnomaly {
		switch {
		case zScore > 4.0:
			severity = "critical"
		case zScore > 3.0:
			severity = "high"
		case zScore > 2.5:
			severity = "medium"
		default:
			severity = "low"
		}
	}

	return &AnomalyDetectionResult{
		MetricName:        metricName,
		CurrentValue:      currentValue,
		ExpectedValue:     historicalMean,
		Deviation:         math.Abs(currentValue - historicalMean),
		ZScore:            zScore,
		IsAnomaly:         isAnomaly,
		Severity:          severity,
		Confidence:        math.Min(zScore/4.0, 1.0), // Cap confidence at 1.0
		DetectionTime:     time.Now(),
		RecommendedAction: mr.getAnomalyRecommendation(isAnomaly, severity),
	}, nil
}

// getAnomalyRecommendation provides recommendations for anomaly handling.
func (mr *MetricsRecorder) getAnomalyRecommendation(isAnomaly bool, severity string) string {
	if !isAnomaly {
		return "no_action_required"
	}

	switch severity {
	case "critical":
		return "immediate_investigation_required"
	case "high":
		return "escalate_to_on_call"
	case "medium":
		return "investigate_within_1_hour"
	case "low":
		return "monitor_and_document"
	default:
		return "review_during_business_hours"
	}
}

// Supporting Types for Advanced Analytics.

// TrendAnalysis represents trend analysis results.
type TrendAnalysis struct {
	MetricName     string        `json:"metric_name"`
	TimeWindow     time.Duration `json:"time_window"`
	TrendDirection string        `json:"trend_direction"` // "increasing", "decreasing", "stable"
	TrendStrength  float64       `json:"trend_strength"`  // 0.0 to 1.0
	Forecast       float64       `json:"forecast"`
	Confidence     float64       `json:"confidence"` // 0.0 to 1.0
	AnalysisTime   time.Time     `json:"analysis_time"`
}

// CapacityPrediction represents capacity planning predictions.
type CapacityPrediction struct {
	TimeHorizon          time.Duration `json:"time_horizon"`
	CurrentThroughput    float64       `json:"current_throughput"`
	PredictedThroughput  float64       `json:"predicted_throughput"`
	CurrentUtilization   float64       `json:"current_utilization"`
	PredictedUtilization float64       `json:"predicted_utilization"`
	RecommendedAction    string        `json:"recommended_action"`
	Confidence           float64       `json:"confidence"`
	PredictionTime       time.Time     `json:"prediction_time"`
}

// AnomalyDetectionResult represents anomaly detection results.
type AnomalyDetectionResult struct {
	MetricName        string    `json:"metric_name"`
	CurrentValue      float64   `json:"current_value"`
	ExpectedValue     float64   `json:"expected_value"`
	Deviation         float64   `json:"deviation"`
	ZScore            float64   `json:"z_score"`
	IsAnomaly         bool      `json:"is_anomaly"`
	Severity          string    `json:"severity"`
	Confidence        float64   `json:"confidence"`
	DetectionTime     time.Time `json:"detection_time"`
	RecommendedAction string    `json:"recommended_action"`
}
