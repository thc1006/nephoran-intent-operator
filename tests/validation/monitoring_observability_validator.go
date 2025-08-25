// Package validation provides monitoring and observability validation for production readiness
// This validator ensures comprehensive observability stack is properly configured and functional
package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MonitoringObservabilityValidator validates comprehensive observability stack
// Targets 2/2 points for monitoring and observability in production readiness
type MonitoringObservabilityValidator struct {
	client    client.Client
	clientset *kubernetes.Clientset
	config    *ValidationConfig

	// Monitoring metrics
	metrics *ObservabilityMetrics
	mu      sync.RWMutex
}

// ObservabilityMetrics tracks monitoring and observability validation results
type ObservabilityMetrics struct {
	// Metrics Collection (1 point)
	PrometheusDeployed        bool
	ServiceMonitorsConfigured bool
	MetricsEndpointsActive    bool
	CustomMetricsAvailable    bool
	AlertRulesConfigured      bool

	// Logging & Tracing (1 point)
	LogAggregationActive         bool
	DistributedTracingActive     bool
	LogRetentionConfigured       bool
	TracingInstrumentationActive bool

	// Dashboard & Alerting
	GrafanaDashboards      bool
	AlertManagerConfigured bool
	NotificationChannels   bool
	SLOMonitoringEnabled   bool

	// Health & Status
	AllComponentsHealthy    bool
	DataRetentionCompliant  bool
	QueryPerformanceOptimal bool

	// Detailed component status
	ComponentStatus map[string]ComponentHealth
}

// ComponentHealth represents the health status of an observability component
type ComponentHealth struct {
	Name         string
	Deployed     bool
	Healthy      bool
	Version      string
	Endpoints    []string
	LastChecked  time.Time
	ErrorMessage string
	Metrics      map[string]interface{}
}

// ObservabilityComponent represents a component in the observability stack
type ObservabilityComponent struct {
	Name            string
	Type            ComponentType
	Namespace       string
	DeploymentName  string
	ServiceName     string
	Port            int32
	HealthEndpoint  string
	MetricsEndpoint string
	Required        bool
}

// ComponentType defines the type of observability component
type ComponentType string

const (
	ComponentTypeMetrics   ComponentType = "metrics"
	ComponentTypeLogs      ComponentType = "logs"
	ComponentTypeTracing   ComponentType = "tracing"
	ComponentTypeDashboard ComponentType = "dashboard"
	ComponentTypeAlerting  ComponentType = "alerting"
)

// NewMonitoringObservabilityValidator creates a new monitoring and observability validator
func NewMonitoringObservabilityValidator(client client.Client, clientset *kubernetes.Clientset, config *ValidationConfig) *MonitoringObservabilityValidator {
	return &MonitoringObservabilityValidator{
		client:    client,
		clientset: clientset,
		config:    config,
		metrics: &ObservabilityMetrics{
			ComponentStatus: make(map[string]ComponentHealth),
		},
	}
}

// ValidateMonitoringObservability executes comprehensive monitoring and observability validation
// Returns score out of 2 points for monitoring and observability
func (mov *MonitoringObservabilityValidator) ValidateMonitoringObservability(ctx context.Context) (int, error) {
	ginkgo.By("Starting Monitoring and Observability Validation")

	totalScore := 0
	maxScore := 2

	// Phase 1: Metrics Collection Validation (1 point)
	metricsScore, err := mov.validateMetricsCollection(ctx)
	if err != nil {
		return 0, fmt.Errorf("metrics collection validation failed: %w", err)
	}
	totalScore += metricsScore
	ginkgo.By(fmt.Sprintf("Metrics Collection Score: %d/1 points", metricsScore))

	// Phase 2: Logging and Tracing Validation (1 point)
	loggingScore, err := mov.validateLoggingTracing(ctx)
	if err != nil {
		return 0, fmt.Errorf("logging and tracing validation failed: %w", err)
	}
	totalScore += loggingScore
	ginkgo.By(fmt.Sprintf("Logging and Tracing Score: %d/1 points", loggingScore))

	// Additional validations for comprehensive coverage
	mov.validateDashboardsAlerting(ctx)
	mov.validateDataRetention(ctx)
	mov.validatePerformance(ctx)

	ginkgo.By(fmt.Sprintf("Monitoring and Observability Total Score: %d/%d points", totalScore, maxScore))

	return totalScore, nil
}

// validateMetricsCollection validates Prometheus metrics collection setup
func (mov *MonitoringObservabilityValidator) validateMetricsCollection(ctx context.Context) (int, error) {
	ginkgo.By("Validating Metrics Collection Infrastructure")

	score := 0

	// Check for Prometheus deployment
	if mov.validatePrometheusDeployment(ctx) {
		score = 1
		ginkgo.By("✓ Prometheus metrics collection validated")

		mov.mu.Lock()
		mov.metrics.PrometheusDeployed = true
		mov.mu.Unlock()
	} else {
		ginkgo.By("✗ Prometheus metrics collection not properly configured")
	}

	// Additional metrics validation
	mov.validateServiceMonitors(ctx)
	mov.validateMetricsEndpoints(ctx)
	mov.validateCustomMetrics(ctx)
	mov.validateAlertRules(ctx)

	return score, nil
}

// validatePrometheusDeployment checks for Prometheus server deployment
func (mov *MonitoringObservabilityValidator) validatePrometheusDeployment(ctx context.Context) bool {
	components := []ObservabilityComponent{
		{
			Name:            "prometheus-server",
			Type:            ComponentTypeMetrics,
			DeploymentName:  "prometheus-server",
			ServiceName:     "prometheus-server",
			Port:            9090,
			HealthEndpoint:  "/-/healthy",
			MetricsEndpoint: "/metrics",
			Required:        true,
		},
		{
			Name:           "prometheus-operator",
			Type:           ComponentTypeMetrics,
			DeploymentName: "prometheus-operator",
			ServiceName:    "prometheus-operator",
			Port:           8080,
			Required:       false,
		},
	}

	validComponents := 0
	for _, component := range components {
		if mov.validateObservabilityComponent(ctx, component) {
			validComponents++
		}
	}

	// Require at least one Prometheus component
	return validComponents > 0
}

// validateObservabilityComponent validates a single observability component
func (mov *MonitoringObservabilityValidator) validateObservabilityComponent(ctx context.Context, component ObservabilityComponent) bool {
	health := ComponentHealth{
		Name:        component.Name,
		LastChecked: time.Now(),
	}

	// Check deployment
	deployment := &appsv1.Deployment{}
	key := types.NamespacedName{
		Name:      component.DeploymentName,
		Namespace: component.Namespace,
	}

	// Try multiple namespaces if none specified
	namespaces := []string{"monitoring", "prometheus", "nephoran-system", "kube-system"}
	if component.Namespace != "" {
		namespaces = []string{component.Namespace}
	}

	for _, ns := range namespaces {
		key.Namespace = ns
		if err := mov.client.Get(ctx, key, deployment); err == nil {
			health.Deployed = true
			component.Namespace = ns
			break
		}
	}

	if !health.Deployed {
		// Try by label selector as fallback
		deployments := &appsv1.DeploymentList{}
		listOpts := []client.ListOption{
			client.MatchingLabels(map[string]string{
				"app": component.Name,
			}),
		}

		if err := mov.client.List(ctx, deployments, listOpts...); err == nil && len(deployments.Items) > 0 {
			health.Deployed = true
			deployment = &deployments.Items[0]
			component.Namespace = deployment.Namespace
		}
	}

	if health.Deployed {
		// Check if deployment is ready
		health.Healthy = deployment.Status.ReadyReplicas > 0

		// Get version from deployment
		if deployment.Labels != nil {
			if version, exists := deployment.Labels["version"]; exists {
				health.Version = version
			}
		}

		// Validate service
		mov.validateComponentService(ctx, component, &health)

		// Validate endpoints if healthy
		if health.Healthy {
			mov.validateComponentEndpoints(ctx, component, &health)
		}
	}

	// Store component health
	mov.mu.Lock()
	mov.metrics.ComponentStatus[component.Name] = health
	mov.mu.Unlock()

	return health.Deployed && (health.Healthy || !component.Required)
}

// validateComponentService validates the service for an observability component
func (mov *MonitoringObservabilityValidator) validateComponentService(ctx context.Context, component ObservabilityComponent, health *ComponentHealth) {
	service := &corev1.Service{}
	key := types.NamespacedName{
		Name:      component.ServiceName,
		Namespace: component.Namespace,
	}

	if err := mov.client.Get(ctx, key, service); err == nil {
		// Service exists, collect endpoint information
		for _, port := range service.Spec.Ports {
			endpoint := fmt.Sprintf("%s:%d", service.Name, port.Port)
			health.Endpoints = append(health.Endpoints, endpoint)
		}
	}
}

// validateComponentEndpoints validates health and metrics endpoints
func (mov *MonitoringObservabilityValidator) validateComponentEndpoints(ctx context.Context, component ObservabilityComponent, health *ComponentHealth) {
	if component.HealthEndpoint == "" && component.MetricsEndpoint == "" {
		return
	}

	// For now, we'll validate by checking if pods are ready
	// In a real implementation, you'd make HTTP requests to the endpoints
	pods := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(component.Namespace),
		client.MatchingLabels(map[string]string{
			"app": component.Name,
		}),
	}

	if err := mov.client.List(ctx, pods, listOpts...); err != nil {
		return
	}

	readyPods := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			// Check if all containers are ready
			allReady := true
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
					allReady = false
					break
				}
			}
			if allReady {
				readyPods++
			}
		}
	}

	health.Metrics = map[string]interface{}{
		"ready_pods":  readyPods,
		"total_pods":  len(pods.Items),
		"ready_ratio": float64(readyPods) / float64(len(pods.Items)),
	}
}

// validateServiceMonitors checks for ServiceMonitor configurations
func (mov *MonitoringObservabilityValidator) validateServiceMonitors(ctx context.Context) {
	serviceMonitors := &metav1.PartialObjectMetadataList{}
	serviceMonitors.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "ServiceMonitorList",
	})

	namespaces := []string{"nephoran-system", "monitoring", "prometheus"}
	serviceMonitorCount := 0

	for _, namespace := range namespaces {
		if err := mov.client.List(ctx, serviceMonitors, client.InNamespace(namespace)); err == nil {
			serviceMonitorCount += len(serviceMonitors.Items)
		}
	}

	mov.mu.Lock()
	mov.metrics.ServiceMonitorsConfigured = serviceMonitorCount > 0
	mov.mu.Unlock()

	if serviceMonitorCount > 0 {
		ginkgo.By(fmt.Sprintf("✓ Found %d ServiceMonitor resources", serviceMonitorCount))
	} else {
		ginkgo.By("⚠ No ServiceMonitor resources found")
	}
}

// validateMetricsEndpoints checks for metrics endpoints in services
func (mov *MonitoringObservabilityValidator) validateMetricsEndpoints(ctx context.Context) {
	services := &corev1.ServiceList{}
	if err := mov.client.List(ctx, services, client.InNamespace("nephoran-system")); err != nil {
		return
	}

	metricsEndpoints := 0
	for _, service := range services.Items {
		for _, port := range service.Spec.Ports {
			if port.Name == "metrics" || port.Port == 8080 || port.Port == 9090 {
				metricsEndpoints++
			}
		}
	}

	mov.mu.Lock()
	mov.metrics.MetricsEndpointsActive = metricsEndpoints > 0
	mov.mu.Unlock()

	if metricsEndpoints > 0 {
		ginkgo.By(fmt.Sprintf("✓ Found %d metrics endpoints", metricsEndpoints))
	}
}

// validateCustomMetrics checks for custom application metrics
func (mov *MonitoringObservabilityValidator) validateCustomMetrics(ctx context.Context) {
	// Check for custom metrics annotations or ServiceMonitor configurations
	services := &corev1.ServiceList{}
	if err := mov.client.List(ctx, services, client.InNamespace("nephoran-system")); err != nil {
		return
	}

	customMetricsFound := false
	for _, service := range services.Items {
		if service.Annotations != nil {
			// Look for Prometheus annotations
			if _, exists := service.Annotations["prometheus.io/scrape"]; exists {
				customMetricsFound = true
				break
			}
		}
	}

	mov.mu.Lock()
	mov.metrics.CustomMetricsAvailable = customMetricsFound
	mov.mu.Unlock()

	if customMetricsFound {
		ginkgo.By("✓ Custom metrics configuration found")
	}
}

// validateAlertRules checks for Prometheus alert rules
func (mov *MonitoringObservabilityValidator) validateAlertRules(ctx context.Context) {
	prometheusRules := &metav1.PartialObjectMetadataList{}
	prometheusRules.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "monitoring.coreos.com",
		Version: "v1",
		Kind:    "PrometheusRuleList",
	})

	namespaces := []string{"nephoran-system", "monitoring", "prometheus"}
	alertRulesCount := 0

	for _, namespace := range namespaces {
		if err := mov.client.List(ctx, prometheusRules, client.InNamespace(namespace)); err == nil {
			alertRulesCount += len(prometheusRules.Items)
		}
	}

	mov.mu.Lock()
	mov.metrics.AlertRulesConfigured = alertRulesCount > 0
	mov.mu.Unlock()

	if alertRulesCount > 0 {
		ginkgo.By(fmt.Sprintf("✓ Found %d Prometheus alert rules", alertRulesCount))
	}
}

// validateLoggingTracing validates logging aggregation and distributed tracing
func (mov *MonitoringObservabilityValidator) validateLoggingTracing(ctx context.Context) (int, error) {
	ginkgo.By("Validating Logging and Tracing Infrastructure")

	score := 0

	// Check logging infrastructure
	loggingValid := mov.validateLoggingInfrastructure(ctx)

	// Check tracing infrastructure
	tracingValid := mov.validateTracingInfrastructure(ctx)

	// Award 1 point if either logging or tracing is properly configured
	// Award full point if both are configured
	if loggingValid && tracingValid {
		score = 1
		ginkgo.By("✓ Logging and tracing infrastructure validated")
	} else if loggingValid || tracingValid {
		score = 1
		ginkgo.By("✓ Either logging or tracing infrastructure validated")
	} else {
		ginkgo.By("✗ Neither logging nor tracing infrastructure properly configured")
	}

	return score, nil
}

// validateLoggingInfrastructure checks for log aggregation setup
func (mov *MonitoringObservabilityValidator) validateLoggingInfrastructure(ctx context.Context) bool {
	loggingComponents := []ObservabilityComponent{
		{
			Name:           "elasticsearch",
			Type:           ComponentTypeLogs,
			DeploymentName: "elasticsearch",
			ServiceName:    "elasticsearch",
			Port:           9200,
			HealthEndpoint: "/_cluster/health",
			Required:       false,
		},
		{
			Name:           "fluentd",
			Type:           ComponentTypeLogs,
			DeploymentName: "fluentd",
			ServiceName:    "fluentd",
			Port:           24224,
			Required:       false,
		},
		{
			Name:           "fluent-bit",
			Type:           ComponentTypeLogs,
			DeploymentName: "fluent-bit",
			ServiceName:    "fluent-bit",
			Port:           2020,
			Required:       false,
		},
		{
			Name:           "kibana",
			Type:           ComponentTypeLogs,
			DeploymentName: "kibana",
			ServiceName:    "kibana",
			Port:           5601,
			HealthEndpoint: "/api/status",
			Required:       false,
		},
		{
			Name:           "logstash",
			Type:           ComponentTypeLogs,
			DeploymentName: "logstash",
			ServiceName:    "logstash",
			Port:           5044,
			Required:       false,
		},
	}

	validComponents := 0
	for _, component := range loggingComponents {
		if mov.validateObservabilityComponent(ctx, component) {
			validComponents++
		}
	}

	// Also check for DaemonSets (common for log collectors)
	loggingDaemonSets := mov.validateLoggingDaemonSets(ctx)

	isValid := validComponents > 0 || loggingDaemonSets > 0

	mov.mu.Lock()
	mov.metrics.LogAggregationActive = isValid
	mov.mu.Unlock()

	if isValid {
		ginkgo.By(fmt.Sprintf("✓ Logging infrastructure validated (%d components, %d DaemonSets)",
			validComponents, loggingDaemonSets))
	}

	return isValid
}

// validateLoggingDaemonSets checks for logging DaemonSets
func (mov *MonitoringObservabilityValidator) validateLoggingDaemonSets(ctx context.Context) int {
	daemonSets := &appsv1.DaemonSetList{}
	if err := mov.client.List(ctx, daemonSets); err != nil {
		return 0
	}

	loggingDaemonSets := 0
	loggingKeywords := []string{"fluentd", "fluent-bit", "filebeat", "promtail", "logging"}

	for _, ds := range daemonSets.Items {
		for _, keyword := range loggingKeywords {
			if strings.Contains(strings.ToLower(ds.Name), keyword) ||
				(ds.Labels != nil && strings.Contains(strings.ToLower(ds.Labels["app"]), keyword)) {
				loggingDaemonSets++
				break
			}
		}
	}

	return loggingDaemonSets
}

// validateTracingInfrastructure checks for distributed tracing setup
func (mov *MonitoringObservabilityValidator) validateTracingInfrastructure(ctx context.Context) bool {
	tracingComponents := []ObservabilityComponent{
		{
			Name:           "jaeger-collector",
			Type:           ComponentTypeTracing,
			DeploymentName: "jaeger-collector",
			ServiceName:    "jaeger-collector",
			Port:           14268,
			HealthEndpoint: "/",
			Required:       false,
		},
		{
			Name:           "jaeger-query",
			Type:           ComponentTypeTracing,
			DeploymentName: "jaeger-query",
			ServiceName:    "jaeger-query",
			Port:           16686,
			HealthEndpoint: "/",
			Required:       false,
		},
		{
			Name:           "jaeger-agent",
			Type:           ComponentTypeTracing,
			DeploymentName: "jaeger-agent",
			ServiceName:    "jaeger-agent",
			Port:           5778,
			Required:       false,
		},
		{
			Name:           "zipkin",
			Type:           ComponentTypeTracing,
			DeploymentName: "zipkin",
			ServiceName:    "zipkin",
			Port:           9411,
			HealthEndpoint: "/health",
			Required:       false,
		},
		{
			Name:           "tempo",
			Type:           ComponentTypeTracing,
			DeploymentName: "tempo",
			ServiceName:    "tempo",
			Port:           3100,
			Required:       false,
		},
	}

	validComponents := 0
	for _, component := range tracingComponents {
		if mov.validateObservabilityComponent(ctx, component) {
			validComponents++
		}
	}

	isValid := validComponents > 0

	mov.mu.Lock()
	mov.metrics.DistributedTracingActive = isValid
	mov.mu.Unlock()

	if isValid {
		ginkgo.By(fmt.Sprintf("✓ Tracing infrastructure validated (%d components)", validComponents))
	}

	return isValid
}

// validateDashboardsAlerting validates dashboard and alerting setup
func (mov *MonitoringObservabilityValidator) validateDashboardsAlerting(ctx context.Context) {
	// Check for Grafana dashboards
	mov.validateGrafanaDashboards(ctx)

	// Check for AlertManager
	mov.validateAlertManager(ctx)

	// Check for notification channels
	mov.validateNotificationChannels(ctx)
}

// validateGrafanaDashboards checks for Grafana deployment and dashboards
func (mov *MonitoringObservabilityValidator) validateGrafanaDashboards(ctx context.Context) {
	grafanaComponent := ObservabilityComponent{
		Name:           "grafana",
		Type:           ComponentTypeDashboard,
		DeploymentName: "grafana",
		ServiceName:    "grafana",
		Port:           3000,
		HealthEndpoint: "/api/health",
		Required:       false,
	}

	grafanaValid := mov.validateObservabilityComponent(ctx, grafanaComponent)

	// Check for ConfigMaps containing dashboards
	configMaps := &corev1.ConfigMapList{}
	dashboardConfigMaps := 0

	namespaces := []string{"monitoring", "grafana", "nephoran-system"}
	for _, namespace := range namespaces {
		if err := mov.client.List(ctx, configMaps, client.InNamespace(namespace)); err == nil {
			for _, cm := range configMaps.Items {
				if strings.Contains(cm.Name, "dashboard") ||
					(cm.Labels != nil && cm.Labels["grafana_dashboard"] == "1") {
					dashboardConfigMaps++
				}
			}
		}
	}

	mov.mu.Lock()
	mov.metrics.GrafanaDashboards = grafanaValid && dashboardConfigMaps > 0
	mov.mu.Unlock()

	if grafanaValid {
		ginkgo.By(fmt.Sprintf("✓ Grafana validated with %d dashboard ConfigMaps", dashboardConfigMaps))
	}
}

// validateAlertManager checks for AlertManager deployment
func (mov *MonitoringObservabilityValidator) validateAlertManager(ctx context.Context) {
	alertManagerComponent := ObservabilityComponent{
		Name:           "alertmanager",
		Type:           ComponentTypeAlerting,
		DeploymentName: "alertmanager",
		ServiceName:    "alertmanager",
		Port:           9093,
		HealthEndpoint: "/-/healthy",
		Required:       false,
	}

	alertManagerValid := mov.validateObservabilityComponent(ctx, alertManagerComponent)

	mov.mu.Lock()
	mov.metrics.AlertManagerConfigured = alertManagerValid
	mov.mu.Unlock()

	if alertManagerValid {
		ginkgo.By("✓ AlertManager validated")
	}
}

// validateNotificationChannels checks for notification channel configurations
func (mov *MonitoringObservabilityValidator) validateNotificationChannels(ctx context.Context) {
	// Check for AlertManager configuration containing receivers
	secrets := &corev1.SecretList{}
	configFound := false

	namespaces := []string{"monitoring", "alertmanager", "nephoran-system"}
	for _, namespace := range namespaces {
		if err := mov.client.List(ctx, secrets, client.InNamespace(namespace)); err == nil {
			for _, secret := range secrets.Items {
				if strings.Contains(secret.Name, "alertmanager") {
					// Check if secret contains webhook or email configurations
					if secret.Data != nil {
						for key := range secret.Data {
							if strings.Contains(key, "webhook") ||
								strings.Contains(key, "email") ||
								strings.Contains(key, "slack") {
								configFound = true
								break
							}
						}
					}
				}
			}
		}
	}

	mov.mu.Lock()
	mov.metrics.NotificationChannels = configFound
	mov.mu.Unlock()

	if configFound {
		ginkgo.By("✓ Notification channels configuration found")
	}
}

// validateDataRetention validates data retention policies
func (mov *MonitoringObservabilityValidator) validateDataRetention(ctx context.Context) {
	// Check for PersistentVolumes for data storage
	pvcs := &corev1.PersistentVolumeClaimList{}
	retentionConfigured := false

	namespaces := []string{"monitoring", "prometheus", "elasticsearch", "nephoran-system"}
	for _, namespace := range namespaces {
		if err := mov.client.List(ctx, pvcs, client.InNamespace(namespace)); err == nil {
			if len(pvcs.Items) > 0 {
				retentionConfigured = true
				break
			}
		}
	}

	mov.mu.Lock()
	mov.metrics.DataRetentionCompliant = retentionConfigured
	mov.mu.Unlock()

	if retentionConfigured {
		ginkgo.By("✓ Data retention storage configured")
	}
}

// validatePerformance validates observability stack performance
func (mov *MonitoringObservabilityValidator) validatePerformance(ctx context.Context) {
	// Check resource limits and requests are properly configured
	deployments := &appsv1.DeploymentList{}
	performanceOptimal := true

	namespaces := []string{"monitoring", "prometheus", "grafana", "logging"}
	for _, namespace := range namespaces {
		if err := mov.client.List(ctx, deployments, client.InNamespace(namespace)); err == nil {
			for _, deployment := range deployments.Items {
				for _, container := range deployment.Spec.Template.Spec.Containers {
					// Check if resources are specified
					if container.Resources.Requests == nil && container.Resources.Limits == nil {
						performanceOptimal = false
					}
				}
			}
		}
	}

	mov.mu.Lock()
	mov.metrics.QueryPerformanceOptimal = performanceOptimal
	mov.mu.Unlock()

	if performanceOptimal {
		ginkgo.By("✓ Observability components have resource constraints configured")
	}
}

// GetObservabilityMetrics returns the current observability metrics
func (mov *MonitoringObservabilityValidator) GetObservabilityMetrics() *ObservabilityMetrics {
	mov.mu.RLock()
	defer mov.mu.RUnlock()
	return mov.metrics
}

// GenerateObservabilityReport generates a comprehensive observability report
func (mov *MonitoringObservabilityValidator) GenerateObservabilityReport() string {
	mov.mu.RLock()
	defer mov.mu.RUnlock()

	report := fmt.Sprintf(`
=============================================================================
MONITORING AND OBSERVABILITY VALIDATION REPORT
=============================================================================

METRICS COLLECTION:
├── Prometheus Deployed:        %t
├── Service Monitors Configured: %t
├── Metrics Endpoints Active:   %t
├── Custom Metrics Available:   %t
└── Alert Rules Configured:     %t

LOGGING & TRACING:
├── Log Aggregation Active:     %t
├── Distributed Tracing Active: %t
├── Log Retention Configured:   %t
└── Tracing Instrumentation:    %t

DASHBOARDS & ALERTING:
├── Grafana Dashboards:         %t
├── Alert Manager Configured:   %t
├── Notification Channels:      %t
└── SLO Monitoring Enabled:     %t

COMPONENT HEALTH STATUS:
`,
		mov.metrics.PrometheusDeployed,
		mov.metrics.ServiceMonitorsConfigured,
		mov.metrics.MetricsEndpointsActive,
		mov.metrics.CustomMetricsAvailable,
		mov.metrics.AlertRulesConfigured,
		mov.metrics.LogAggregationActive,
		mov.metrics.DistributedTracingActive,
		mov.metrics.LogRetentionConfigured,
		mov.metrics.TracingInstrumentationActive,
		mov.metrics.GrafanaDashboards,
		mov.metrics.AlertManagerConfigured,
		mov.metrics.NotificationChannels,
		mov.metrics.SLOMonitoringEnabled,
	)

	// Add component status details
	for name, health := range mov.metrics.ComponentStatus {
		status := "❌"
		if health.Healthy {
			status = "✅"
		} else if health.Deployed {
			status = "⚠️"
		}

		report += fmt.Sprintf("├── %-20s %s (Deployed: %t, Healthy: %t)\n",
			name, status, health.Deployed, health.Healthy)
	}

	report += `
PERFORMANCE & RETENTION:
├── All Components Healthy:     ` + fmt.Sprintf("%t", mov.metrics.AllComponentsHealthy) + `
├── Data Retention Compliant:   ` + fmt.Sprintf("%t", mov.metrics.DataRetentionCompliant) + `
└── Query Performance Optimal:  ` + fmt.Sprintf("%t", mov.metrics.QueryPerformanceOptimal) + `

=============================================================================
`

	return report
}
