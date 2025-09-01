// Package o2 implements comprehensive infrastructure monitoring for O2 IMS.

package o2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/providers"
)

// InfrastructureMonitoringService provides comprehensive infrastructure monitoring.

type InfrastructureMonitoringService struct {
	config *InfrastructureMonitoringConfig

	logger *logging.StructuredLogger

	// Provider registry for multi-cloud monitoring.

	providerRegistry providers.ProviderRegistry

	// Monitoring components.

	metrics *InfrastructureMetrics

	healthChecker InfrastructureHealthChecker

	slaMonitor *SLAMonitor

	eventProcessor *EventProcessor

	// Data collection and storage.

	resourceMonitors map[string]*ResourceMonitor

	collectors []MetricsCollector

	// Synchronization.

	mu sync.RWMutex

	// Lifecycle management.

	ctx context.Context

	cancel context.CancelFunc

	wg sync.WaitGroup
}

// InfrastructureMonitoringConfig configuration for infrastructure monitoring.

type InfrastructureMonitoringConfig struct {

	// Collection intervals.

	MetricsCollectionInterval time.Duration `json:"metricsCollectionInterval,omitempty"`

	HealthCheckInterval time.Duration `json:"healthCheckInterval,omitempty"`

	SLAEvaluationInterval time.Duration `json:"slaEvaluationInterval,omitempty"`

	// Thresholds and limits.

	CPUAlertThreshold float64 `json:"cpuAlertThreshold,omitempty"`

	MemoryAlertThreshold float64 `json:"memoryAlertThreshold,omitempty"`

	StorageAlertThreshold float64 `json:"storageAlertThreshold,omitempty"`

	NetworkLatencyThreshold time.Duration `json:"networkLatencyThreshold,omitempty"`

	// Retention policies.

	MetricsRetentionPeriod time.Duration `json:"metricsRetentionPeriod,omitempty"`

	AlertRetentionPeriod time.Duration `json:"alertRetentionPeriod,omitempty"`

	EventRetentionPeriod time.Duration `json:"eventRetentionPeriod,omitempty"`

	// Integration settings.

	PrometheusEnabled bool `json:"prometheusEnabled,omitempty"`

	GrafanaEnabled bool `json:"grafanaEnabled,omitempty"`

	AlertmanagerEnabled bool `json:"alertmanagerEnabled,omitempty"`

	JaegerEnabled bool `json:"jaegerEnabled,omitempty"`

	// Telecommunications-specific settings.

	TelcoKPIEnabled bool `json:"telcoKPIEnabled,omitempty"`

	FiveGCoreMonitoringEnabled bool `json:"5gCoreMonitoringEnabled,omitempty"`

	ORANInterfaceMonitoring bool `json:"oranInterfaceMonitoring,omitempty"`

	NetworkSliceMonitoring bool `json:"networkSliceMonitoring,omitempty"`
}

// InfrastructureMetrics contains Prometheus metrics for infrastructure monitoring.

type InfrastructureMetrics struct {

	// Resource utilization metrics.

	CPUUtilization *prometheus.GaugeVec

	MemoryUtilization *prometheus.GaugeVec

	StorageUtilization *prometheus.GaugeVec

	NetworkUtilization *prometheus.GaugeVec

	// Infrastructure health metrics.

	ResourceHealth *prometheus.GaugeVec

	NodeHealth *prometheus.GaugeVec

	ServiceHealth *prometheus.GaugeVec

	// Performance metrics.

	NetworkLatency *prometheus.HistogramVec

	StorageIOPS *prometheus.GaugeVec

	StorageThroughput *prometheus.GaugeVec

	NetworkThroughput *prometheus.GaugeVec

	// SLA compliance metrics.

	SLACompliance *prometheus.GaugeVec

	SLAViolations *prometheus.CounterVec

	ServiceAvailability *prometheus.GaugeVec

	// Telecommunications-specific KPIs.

	PRBUtilization *prometheus.GaugeVec

	HandoverSuccessRate *prometheus.GaugeVec

	SessionSetupTime *prometheus.HistogramVec

	PacketLossRate *prometheus.GaugeVec

	ThroughputPerSlice *prometheus.GaugeVec

	// Alert and event metrics.

	ActiveAlerts *prometheus.GaugeVec

	AlertsResolved *prometheus.CounterVec

	EventsProcessed *prometheus.CounterVec

	// Cost optimization metrics.

	ResourceCost *prometheus.GaugeVec

	UnusedResources *prometheus.GaugeVec

	CostOptimizationSavings *prometheus.GaugeVec
}

// ResourceMonitor monitors a specific resource.

type ResourceMonitor struct {
	Resource *models.Resource

	Provider providers.CloudProvider

	LastUpdate time.Time

	Status *models.ResourceStatus

	Metrics map[string]interface{}

	Alerts []Alert

	mu sync.RWMutex
}

// Alert represents an infrastructure alert.

type Alert struct {
	ID string `json:"id"`

	Type string `json:"type"`

	Severity string `json:"severity"`

	Title string `json:"title"`

	Description string `json:"description"`

	Resource string `json:"resource"`

	Timestamp time.Time `json:"timestamp"`

	Status string `json:"status"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MetricsCollector interface for collecting metrics from different sources.

type MetricsCollector interface {
	CollectMetrics(ctx context.Context) (map[string]interface{}, error)

	GetCollectorType() string

	IsHealthy() bool
}

// NewInfrastructureMonitoringService creates a new infrastructure monitoring service.

func NewInfrastructureMonitoringService(

	config *InfrastructureMonitoringConfig,

	providerRegistry providers.ProviderRegistry,

	logger *logging.StructuredLogger,

) (*InfrastructureMonitoringService, error) {

	if config == nil {

		config = DefaultInfrastructureMonitoringConfig()

	}

	if logger == nil {

		logger = logging.NewStructuredLogger(logging.DefaultConfig("infrastructure-monitoring", "1.0.0", "production"))

	}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize metrics.

	metrics := newInfrastructureMetrics()

	// Initialize health checker.

	healthChecker := newInfrastructureHealthChecker(config, logger)

	// Initialize SLA monitor.

	slaMonitor := newSLAMonitor(config, logger)

	// Initialize event processor.

	eventProcessor := newEventProcessor(config, logger)

	service := &InfrastructureMonitoringService{

		config: config,

		logger: logger,

		providerRegistry: providerRegistry,

		metrics: metrics,

		healthChecker: healthChecker,

		slaMonitor: slaMonitor,

		eventProcessor: eventProcessor,

		resourceMonitors: make(map[string]*ResourceMonitor),

		collectors: []MetricsCollector{},

		ctx: ctx,

		cancel: cancel,
	}

	// Initialize collectors.

	if err := service.initializeCollectors(); err != nil {

		cancel()

		return nil, fmt.Errorf("failed to initialize collectors: %w", err)

	}

	return service, nil

}

// DefaultInfrastructureMonitoringConfig returns default configuration.

func DefaultInfrastructureMonitoringConfig() *InfrastructureMonitoringConfig {

	return &InfrastructureMonitoringConfig{

		MetricsCollectionInterval: 30 * time.Second,

		HealthCheckInterval: 60 * time.Second,

		SLAEvaluationInterval: 300 * time.Second,

		CPUAlertThreshold: 80.0,

		MemoryAlertThreshold: 85.0,

		StorageAlertThreshold: 90.0,

		NetworkLatencyThreshold: 100 * time.Millisecond,

		MetricsRetentionPeriod: 7 * 24 * time.Hour,

		AlertRetentionPeriod: 30 * 24 * time.Hour,

		EventRetentionPeriod: 14 * 24 * time.Hour,

		PrometheusEnabled: true,

		GrafanaEnabled: true,

		AlertmanagerEnabled: true,

		JaegerEnabled: true,

		TelcoKPIEnabled: true,

		FiveGCoreMonitoringEnabled: true,

		ORANInterfaceMonitoring: true,

		NetworkSliceMonitoring: true,
	}

}

// newInfrastructureMetrics creates new Prometheus metrics for infrastructure monitoring.

func newInfrastructureMetrics() *InfrastructureMetrics {

	return &InfrastructureMetrics{

		CPUUtilization: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_cpu_utilization_percent",

			Help: "CPU utilization percentage by resource",
		}, []string{"resource_id", "resource_type", "provider", "region", "zone"}),

		MemoryUtilization: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_memory_utilization_percent",

			Help: "Memory utilization percentage by resource",
		}, []string{"resource_id", "resource_type", "provider", "region", "zone"}),

		StorageUtilization: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_storage_utilization_percent",

			Help: "Storage utilization percentage by resource",
		}, []string{"resource_id", "resource_type", "provider", "region", "zone", "volume_type"}),

		NetworkUtilization: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_network_utilization_percent",

			Help: "Network utilization percentage by resource",
		}, []string{"resource_id", "resource_type", "provider", "region", "zone", "interface"}),

		ResourceHealth: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_resource_health",

			Help: "Resource health status (0=unhealthy, 1=degraded, 2=healthy)",
		}, []string{"resource_id", "resource_type", "provider", "region", "zone"}),

		NodeHealth: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_node_health",

			Help: "Node health status (0=not_ready, 1=degraded, 2=ready)",
		}, []string{"node_id", "node_type", "provider", "region", "zone"}),

		ServiceHealth: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_service_health",

			Help: "Service health status (0=down, 1=degraded, 2=up)",
		}, []string{"service_name", "service_type", "provider", "region", "zone"}),

		NetworkLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{

			Name: "nephoran_infrastructure_network_latency_seconds",

			Help: "Network latency between infrastructure components",

			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
		}, []string{"source", "destination", "provider", "region"}),

		StorageIOPS: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_storage_iops",

			Help: "Storage IOPS by resource",
		}, []string{"resource_id", "provider", "region", "zone", "volume_type"}),

		StorageThroughput: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_storage_throughput_bytes_per_second",

			Help: "Storage throughput in bytes per second",
		}, []string{"resource_id", "provider", "region", "zone", "volume_type", "operation"}),

		NetworkThroughput: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_network_throughput_bytes_per_second",

			Help: "Network throughput in bytes per second",
		}, []string{"resource_id", "provider", "region", "zone", "interface", "direction"}),

		SLACompliance: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_sla_compliance_percent",

			Help: "SLA compliance percentage",
		}, []string{"sla_type", "service", "provider", "region"}),

		SLAViolations: promauto.NewCounterVec(prometheus.CounterOpts{

			Name: "nephoran_infrastructure_sla_violations_total",

			Help: "Total number of SLA violations",
		}, []string{"sla_type", "service", "provider", "region", "violation_type"}),

		ServiceAvailability: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_service_availability_percent",

			Help: "Service availability percentage",
		}, []string{"service_name", "service_type", "provider", "region"}),

		// Telecommunications-specific KPIs.

		PRBUtilization: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_telco_prb_utilization_percent",

			Help: "Physical Resource Block utilization percentage",
		}, []string{"cell_id", "frequency_band", "provider", "region"}),

		HandoverSuccessRate: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_telco_handover_success_rate_percent",

			Help: "Handover success rate percentage",
		}, []string{"source_cell", "target_cell", "handover_type", "provider"}),

		SessionSetupTime: promauto.NewHistogramVec(prometheus.HistogramOpts{

			Name: "nephoran_telco_session_setup_time_seconds",

			Help: "Session setup time in seconds",

			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 2.0, 5.0},
		}, []string{"session_type", "network_function", "provider", "region"}),

		PacketLossRate: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_telco_packet_loss_rate_percent",

			Help: "Packet loss rate percentage",
		}, []string{"network_function", "interface", "provider", "region"}),

		ThroughputPerSlice: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_telco_throughput_per_slice_mbps",

			Help: "Throughput per network slice in Mbps",
		}, []string{"slice_id", "slice_type", "provider", "region"}),

		ActiveAlerts: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_active_alerts",

			Help: "Number of active alerts",
		}, []string{"severity", "alert_type", "provider", "region"}),

		AlertsResolved: promauto.NewCounterVec(prometheus.CounterOpts{

			Name: "nephoran_infrastructure_alerts_resolved_total",

			Help: "Total number of resolved alerts",
		}, []string{"severity", "alert_type", "provider", "region"}),

		EventsProcessed: promauto.NewCounterVec(prometheus.CounterOpts{

			Name: "nephoran_infrastructure_events_processed_total",

			Help: "Total number of processed events",
		}, []string{"event_type", "source", "provider"}),

		ResourceCost: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_resource_cost_usd_per_hour",

			Help: "Resource cost in USD per hour",
		}, []string{"resource_id", "resource_type", "provider", "region", "zone"}),

		UnusedResources: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_unused_resources",

			Help: "Number of unused resources",
		}, []string{"resource_type", "provider", "region", "zone"}),

		CostOptimizationSavings: promauto.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_infrastructure_cost_optimization_savings_usd",

			Help: "Cost optimization savings in USD",
		}, []string{"optimization_type", "provider", "region"}),
	}

}

// initializeCollectors initializes metrics collectors for different providers.

func (s *InfrastructureMonitoringService) initializeCollectors() error {

	s.logger.Info("initializing metrics collectors")

	// Initialize collectors for each registered provider.

	providerNames := s.providerRegistry.ListProviders()

	for _, providerName := range providerNames {

		provider, err := s.providerRegistry.GetProvider(providerName)

		if err != nil {

			s.logger.Error("failed to get provider", "name", providerName, "error", err)

			continue

		}

		collector, err := s.createCollectorForProvider(provider)

		if err != nil {

			providerInfo := provider.GetProviderInfo()

			s.logger.Error("failed to create collector for provider",

				"provider", providerInfo.Type,

				"error", err)

			continue

		}

		s.collectors = append(s.collectors, collector)

		s.logger.Info("initialized collector for provider",

			"provider", getProviderType(provider),

			"collector_type", collector.GetCollectorType())

	}

	return nil

}

// createCollectorForProvider creates a metrics collector for a specific provider.

func (s *InfrastructureMonitoringService) createCollectorForProvider(provider providers.CloudProvider) (MetricsCollector, error) {

	providerInfo := provider.GetProviderInfo()

	switch providerInfo.Type {

	case "aws":

		return NewAWSMetricsCollector(), nil

	case "azure":

		return NewAzureMetricsCollector(), nil

	case "gcp":

		return NewGCPMetricsCollector(), nil

	case "openstack":

		return &stubMetricsCollector{name: "openstack"}, nil

	case "kubernetes":

		return &stubMetricsCollector{name: "kubernetes"}, nil

	default:

		return &stubMetricsCollector{name: "generic"}, nil

	}

}

// Start starts the infrastructure monitoring service.

func (s *InfrastructureMonitoringService) Start(ctx context.Context) error {

	s.logger.Info("starting infrastructure monitoring service")

	// Start background goroutines.

	s.wg.Add(4)

	// Start metrics collection.

	go s.metricsCollectionLoop()

	// Start health checking.

	go s.healthCheckLoop()

	// Start SLA monitoring.

	go s.slaMonitoringLoop()

	// Start event processing.

	go s.eventProcessingLoop()

	return nil

}

// Stop stops the infrastructure monitoring service.

func (s *InfrastructureMonitoringService) Stop() error {

	s.logger.Info("stopping infrastructure monitoring service")

	// Cancel context.

	s.cancel()

	// Wait for all goroutines to complete.

	s.wg.Wait()

	s.logger.Info("infrastructure monitoring service stopped")

	return nil

}

// metricsCollectionLoop runs the metrics collection loop.

func (s *InfrastructureMonitoringService) metricsCollectionLoop() {

	defer s.wg.Done()

	ticker := time.NewTicker(s.config.MetricsCollectionInterval)

	defer ticker.Stop()

	for {

		select {

		case <-s.ctx.Done():

			return

		case <-ticker.C:

			s.collectAllMetrics()

		}

	}

}

// healthCheckLoop runs the health checking loop.

func (s *InfrastructureMonitoringService) healthCheckLoop() {

	defer s.wg.Done()

	ticker := time.NewTicker(s.config.HealthCheckInterval)

	defer ticker.Stop()

	for {

		select {

		case <-s.ctx.Done():

			return

		case <-ticker.C:

			s.performHealthChecks()

		}

	}

}

// slaMonitoringLoop runs the SLA monitoring loop.

func (s *InfrastructureMonitoringService) slaMonitoringLoop() {

	defer s.wg.Done()

	ticker := time.NewTicker(s.config.SLAEvaluationInterval)

	defer ticker.Stop()

	for {

		select {

		case <-s.ctx.Done():

			return

		case <-ticker.C:

			s.evaluateSLACompliance()

		}

	}

}

// eventProcessingLoop runs the event processing loop.

func (s *InfrastructureMonitoringService) eventProcessingLoop() {

	defer s.wg.Done()

	for {

		select {

		case <-s.ctx.Done():

			return

		default:

			s.processEvents()

			time.Sleep(1 * time.Second) // Small delay to prevent busy waiting

		}

	}

}

// collectAllMetrics collects metrics from all collectors.

func (s *InfrastructureMonitoringService) collectAllMetrics() {

	s.logger.Debug("collecting infrastructure metrics")

	for _, collector := range s.collectors {

		if !collector.IsHealthy() {

			s.logger.Warn("skipping unhealthy collector",

				"collector_type", collector.GetCollectorType())

			continue

		}

		metrics, err := collector.CollectMetrics(s.ctx)

		if err != nil {

			s.logger.Error("failed to collect metrics",

				"collector_type", collector.GetCollectorType(),

				"error", err)

			continue

		}

		s.processCollectedMetrics(collector.GetCollectorType(), metrics)

	}

}

// processCollectedMetrics processes collected metrics and updates Prometheus metrics.

func (s *InfrastructureMonitoringService) processCollectedMetrics(collectorType string, metricsData map[string]interface{}) {

	// Process different types of metrics.

	// CPU utilization metrics.

	if cpuMetrics, ok := metricsData["cpu"]; ok {

		if cpuMap, ok := cpuMetrics.(map[string]interface{}); ok {

			for resourceID, utilization := range cpuMap {

				if util, ok := utilization.(float64); ok {

					s.metrics.CPUUtilization.WithLabelValues(

						resourceID, "compute", collectorType, "default", "default").Set(util)

				}

			}

		}

	}

	// Memory utilization metrics.

	if memoryMetrics, ok := metricsData["memory"]; ok {

		if memoryMap, ok := memoryMetrics.(map[string]interface{}); ok {

			for resourceID, utilization := range memoryMap {

				if util, ok := utilization.(float64); ok {

					s.metrics.MemoryUtilization.WithLabelValues(

						resourceID, "compute", collectorType, "default", "default").Set(util)

				}

			}

		}

	}

	// Storage utilization metrics.

	if storageMetrics, ok := metricsData["storage"]; ok {

		if storageMap, ok := storageMetrics.(map[string]interface{}); ok {

			for resourceID, utilization := range storageMap {

				if util, ok := utilization.(float64); ok {

					s.metrics.StorageUtilization.WithLabelValues(

						resourceID, "storage", collectorType, "default", "default", "ssd").Set(util)

				}

			}

		}

	}

	// Network utilization metrics.

	if networkMetrics, ok := metricsData["network"]; ok {

		if networkMap, ok := networkMetrics.(map[string]interface{}); ok {

			for resourceID, utilization := range networkMap {

				if util, ok := utilization.(float64); ok {

					s.metrics.NetworkUtilization.WithLabelValues(

						resourceID, "network", collectorType, "default", "default", "eth0").Set(util)

				}

			}

		}

	}

	// Telecommunications-specific metrics.

	if s.config.TelcoKPIEnabled {

		s.processTelcoMetrics(collectorType, metricsData)

	}

}

// processTelcoMetrics processes telecommunications-specific metrics.

func (s *InfrastructureMonitoringService) processTelcoMetrics(collectorType string, metricsData map[string]interface{}) {

	// PRB utilization.

	if prbMetrics, ok := metricsData["prb_utilization"]; ok {

		if prbMap, ok := prbMetrics.(map[string]interface{}); ok {

			for cellID, utilization := range prbMap {

				if util, ok := utilization.(float64); ok {

					s.metrics.PRBUtilization.WithLabelValues(

						cellID, "3.5GHz", collectorType, "default").Set(util)

				}

			}

		}

	}

	// Handover success rate.

	if handoverMetrics, ok := metricsData["handover_success_rate"]; ok {

		if handoverMap, ok := handoverMetrics.(map[string]interface{}); ok {

			for handoverID, rate := range handoverMap {

				if successRate, ok := rate.(float64); ok {

					s.metrics.HandoverSuccessRate.WithLabelValues(

						handoverID, "target_cell", "intra_frequency", collectorType).Set(successRate)

				}

			}

		}

	}

	// Packet loss rate.

	if packetLossMetrics, ok := metricsData["packet_loss_rate"]; ok {

		if packetLossMap, ok := packetLossMetrics.(map[string]interface{}); ok {

			for nfID, lossRate := range packetLossMap {

				if loss, ok := lossRate.(float64); ok {

					s.metrics.PacketLossRate.WithLabelValues(

						nfID, "n3", collectorType, "default").Set(loss)

				}

			}

		}

	}

}

// performHealthChecks performs health checks on all monitored resources.

func (s *InfrastructureMonitoringService) performHealthChecks() {

	s.logger.Debug("performing infrastructure health checks")

	s.mu.RLock()

	defer s.mu.RUnlock()

	for resourceID, monitor := range s.resourceMonitors {

		go s.checkResourceHealth(resourceID, monitor)

	}

}

// checkResourceHealth checks the health of a specific resource.

func (s *InfrastructureMonitoringService) checkResourceHealth(resourceID string, monitor *ResourceMonitor) {

	monitor.mu.Lock()

	defer monitor.mu.Unlock()

	// Perform health check through provider.

	healthStatus, err := monitor.Provider.GetResourceHealth(s.ctx, resourceID)

	healthy := false

	details := make(map[string]interface{})

	if err == nil && healthStatus != nil {

		healthy = healthStatus.Status == "healthy"

		// Convert string metadata to interface{} map
		for k, v := range healthStatus.Metadata {
			details[k] = v
		}

	}

	if err != nil {

		s.logger.Error("failed to check resource health",

			"resource_id", resourceID,

			"error", err)

		// Update metrics to indicate unhealthy state.

		s.metrics.ResourceHealth.WithLabelValues(

			resourceID, monitor.Resource.ResourceTypeID,

			getProviderType(monitor.Provider), "default", "default").Set(0)

		return

	}

	// Update resource status.

	if monitor.Resource.Status == nil {

		monitor.Resource.Status = &models.ResourceStatus{}

	}

	if healthy {

		monitor.Resource.Status.Health = models.HealthStateHealthy

		s.metrics.ResourceHealth.WithLabelValues(

			resourceID, monitor.Resource.ResourceTypeID,

			getProviderType(monitor.Provider), "default", "default").Set(2)

	} else {

		monitor.Resource.Status.Health = models.HealthStateUnhealthy

		s.metrics.ResourceHealth.WithLabelValues(

			resourceID, monitor.Resource.ResourceTypeID,

			getProviderType(monitor.Provider), "default", "default").Set(0)

		// Create alert for unhealthy resource.

		alert := Alert{

			ID: fmt.Sprintf("health-%s-%d", resourceID, time.Now().Unix()),

			Type: "resource_health",

			Severity: "critical",

			Title: "Resource Health Check Failed",

			Description: fmt.Sprintf("Resource %s health check failed", resourceID),

			Resource: resourceID,

			Timestamp: time.Now(),

			Status: "active",

			Metadata: details,
		}

		monitor.Alerts = append(monitor.Alerts, alert)

		s.metrics.ActiveAlerts.WithLabelValues("critical", "resource_health",

			getProviderType(monitor.Provider), "default").Inc()

	}

	monitor.Resource.Status.LastHealthCheck = time.Now()

	monitor.LastUpdate = time.Now()

}

// evaluateSLACompliance evaluates SLA compliance for all services.

func (s *InfrastructureMonitoringService) evaluateSLACompliance() {

	s.logger.Debug("evaluating SLA compliance")

	// This would typically evaluate SLA compliance based on collected metrics.

	// and predefined SLA thresholds.

	// Example: Service availability SLA.

	s.evaluateServiceAvailabilitySLA()

	// Example: Network latency SLA.

	s.evaluateNetworkLatencySLA()

	// Example: Resource utilization SLA.

	s.evaluateResourceUtilizationSLA()

}

// evaluateServiceAvailabilitySLA evaluates service availability SLA.

func (s *InfrastructureMonitoringService) evaluateServiceAvailabilitySLA() {

	// Example implementation - in practice this would query actual service metrics.

	services := []string{"amf", "smf", "upf", "nssf", "udr", "udm"}

	for _, service := range services {

		// Simulate availability calculation.

		availability := 99.95 // This would be calculated from actual metrics

		s.metrics.ServiceAvailability.WithLabelValues(

			service, "5g_core", "kubernetes", "default").Set(availability)

		// Check SLA compliance (assuming 99.9% SLA).

		slaTarget := 99.9

		compliance := (availability / slaTarget) * 100

		if compliance > 100 {

			compliance = 100

		}

		s.metrics.SLACompliance.WithLabelValues(

			"availability", service, "kubernetes", "default").Set(compliance)

		if availability < slaTarget {

			s.metrics.SLAViolations.WithLabelValues(

				"availability", service, "kubernetes", "default", "below_threshold").Inc()

		}

	}

}

// evaluateNetworkLatencySLA evaluates network latency SLA.

func (s *InfrastructureMonitoringService) evaluateNetworkLatencySLA() {

	// Example implementation for network latency SLA evaluation.

	// This would typically query Prometheus for actual latency metrics.

}

// evaluateResourceUtilizationSLA evaluates resource utilization SLA.

func (s *InfrastructureMonitoringService) evaluateResourceUtilizationSLA() {

	// Example implementation for resource utilization SLA evaluation.

	// This would typically check if resources are within acceptable utilization ranges.

}

// processEvents processes infrastructure events.

func (s *InfrastructureMonitoringService) processEvents() {

	// This would typically consume events from a message queue or event stream.

	// and process them for alerting, metrics, and automation.

}

// AddResourceMonitor adds a resource to monitoring.

func (s *InfrastructureMonitoringService) AddResourceMonitor(resource *models.Resource, provider providers.CloudProvider) error {

	s.mu.Lock()

	defer s.mu.Unlock()

	monitor := &ResourceMonitor{

		Resource: resource,

		Provider: provider,

		LastUpdate: time.Now(),

		Status: resource.Status,

		Metrics: make(map[string]interface{}),

		Alerts: []Alert{},
	}

	s.resourceMonitors[resource.ResourceID] = monitor

	s.logger.Info("added resource to monitoring",

		"resource_id", resource.ResourceID,

		"resource_type", resource.ResourceTypeID,

		"provider", getProviderType(provider))

	return nil

}

// RemoveResourceMonitor removes a resource from monitoring.

func (s *InfrastructureMonitoringService) RemoveResourceMonitor(resourceID string) error {

	s.mu.Lock()

	defer s.mu.Unlock()

	if monitor, exists := s.resourceMonitors[resourceID]; exists {

		delete(s.resourceMonitors, resourceID)

		s.logger.Info("removed resource from monitoring",

			"resource_id", resourceID,

			"provider", getProviderType(monitor.Provider))

	}

	return nil

}

// GetResourceHealth returns the current health status of a resource.

func (s *InfrastructureMonitoringService) GetResourceHealth(resourceID string) (*models.ResourceStatus, error) {

	s.mu.RLock()

	defer s.mu.RUnlock()

	monitor, exists := s.resourceMonitors[resourceID]

	if !exists {

		return nil, fmt.Errorf("resource %s not found in monitoring", resourceID)

	}

	monitor.mu.RLock()

	defer monitor.mu.RUnlock()

	return monitor.Resource.Status, nil

}

// GetActiveAlerts returns all active alerts.

func (s *InfrastructureMonitoringService) GetActiveAlerts() []Alert {

	s.mu.RLock()

	defer s.mu.RUnlock()

	var alerts []Alert

	for _, monitor := range s.resourceMonitors {

		monitor.mu.RLock()

		for _, alert := range monitor.Alerts {

			if alert.Status == "active" {

				alerts = append(alerts, alert)

			}

		}

		monitor.mu.RUnlock()

	}

	return alerts

}

// GetResourceMetrics returns current metrics for a resource.

func (s *InfrastructureMonitoringService) GetResourceMetrics(resourceID string) (map[string]interface{}, error) {

	s.mu.RLock()

	defer s.mu.RUnlock()

	monitor, exists := s.resourceMonitors[resourceID]

	if !exists {

		return nil, fmt.Errorf("resource %s not found in monitoring", resourceID)

	}

	monitor.mu.RLock()

	defer monitor.mu.RUnlock()

	return monitor.Metrics, nil

}

// InfrastructureHealthCheckerImpl implements the InfrastructureHealthChecker interface
type InfrastructureHealthCheckerImpl struct {
	config *InfrastructureMonitoringConfig
	logger *logging.StructuredLogger
}

// CheckHealth checks the health of a specific resource
func (h *InfrastructureHealthCheckerImpl) CheckHealth(ctx context.Context, resourceID string) (*HealthStatus, error) {
	return &HealthStatus{Status: "healthy"}, nil
}

// CheckAllResources checks the health of all resources
func (h *InfrastructureHealthCheckerImpl) CheckAllResources(ctx context.Context) (map[string]*HealthStatus, error) {
	return make(map[string]*HealthStatus), nil
}

// StartHealthMonitoring starts health monitoring for a resource
func (h *InfrastructureHealthCheckerImpl) StartHealthMonitoring(ctx context.Context, resourceID string, interval time.Duration) error {
	return nil
}

// StopHealthMonitoring stops health monitoring for a resource
func (h *InfrastructureHealthCheckerImpl) StopHealthMonitoring(ctx context.Context, resourceID string) error {
	return nil
}

// SetHealthPolicy sets the health policy for a resource
func (h *InfrastructureHealthCheckerImpl) SetHealthPolicy(ctx context.Context, resourceID string, policy interface{}) error {
	return nil
}

// GetHealthPolicy gets the health policy for a resource
func (h *InfrastructureHealthCheckerImpl) GetHealthPolicy(ctx context.Context, resourceID string) (interface{}, error) {
	return nil, nil
}

// GetHealthHistory gets the health history for a resource
func (h *InfrastructureHealthCheckerImpl) GetHealthHistory(ctx context.Context, resourceID string, duration time.Duration) ([]interface{}, error) {
	return nil, nil
}

// GetHealthTrends gets the health trends for a resource
func (h *InfrastructureHealthCheckerImpl) GetHealthTrends(ctx context.Context, resourceID string, duration time.Duration) (interface{}, error) {
	return nil, nil
}

// RegisterHealthEventCallback registers a health event callback
func (h *InfrastructureHealthCheckerImpl) RegisterHealthEventCallback(ctx context.Context, resourceID string, callback func(interface{}) error) error {
	return nil
}

// UnregisterHealthEventCallback unregisters a health event callback
func (h *InfrastructureHealthCheckerImpl) UnregisterHealthEventCallback(ctx context.Context, resourceID string) error {
	return nil
}

// EmitHealthEvent emits a health event
func (h *InfrastructureHealthCheckerImpl) EmitHealthEvent(ctx context.Context, event interface{}) error {
	return nil
}

// Stub implementations for missing functions.

func newInfrastructureHealthChecker(config *InfrastructureMonitoringConfig, logger *logging.StructuredLogger) InfrastructureHealthChecker {

	return &InfrastructureHealthCheckerImpl{
		config: config,
		logger: logger,
	}

}

func newSLAMonitor(config *InfrastructureMonitoringConfig, logger *logging.StructuredLogger) *SLAMonitor {

	return &SLAMonitor{}

}

func newEventProcessor(config *InfrastructureMonitoringConfig, logger *logging.StructuredLogger) *EventProcessor {

	return &EventProcessor{}

}

// Stub collector implementations.

func NewAWSMetricsCollector() MetricsCollector {

	return &stubMetricsCollector{name: "aws"}

}

// NewAzureMetricsCollector performs newazuremetricscollector operation.

func NewAzureMetricsCollector() MetricsCollector {

	return &stubMetricsCollector{name: "azure"}

}

// NewGCPMetricsCollector performs newgcpmetricscollector operation.

func NewGCPMetricsCollector() MetricsCollector {

	return &stubMetricsCollector{name: "gcp"}

}

// stubMetricsCollector provides a basic implementation of MetricsCollector.

type stubMetricsCollector struct {
	name string
}

// CollectMetrics performs collectmetrics operation.

func (c *stubMetricsCollector) CollectMetrics(ctx context.Context) (map[string]interface{}, error) {

	return map[string]interface{}{

		"collector": c.name,

		"timestamp": time.Now(),

		"status": "healthy",
	}, nil

}

// GetName performs getname operation.

func (c *stubMetricsCollector) GetName() string {

	return c.name

}

// GetCollectorType performs getcollectortype operation.

func (c *stubMetricsCollector) GetCollectorType() string {

	return c.name + "_collector"

}

// IsHealthy performs ishealthy operation.

func (c *stubMetricsCollector) IsHealthy() bool {

	return true

}

// Stop performs stop operation.

func (c *stubMetricsCollector) Stop() error {

	return nil

}

// Helper function to get provider type from CloudProvider interface.

func getProviderType(provider providers.CloudProvider) string {

	if provider == nil {

		return "unknown"

	}

	providerInfo := provider.GetProviderInfo()

	if providerInfo == nil {

		return "unknown"

	}

	return providerInfo.Type

}
