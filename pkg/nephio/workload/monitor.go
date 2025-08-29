
package workload



import (

	"context"

	"fmt"

	"sync"

	"time"



	"github.com/go-logr/logr"

	"github.com/prometheus/client_golang/prometheus"



	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/resource"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/metrics/pkg/client/clientset/versioned"



	"sigs.k8s.io/controller-runtime/pkg/client"

)



// WorkloadMonitor provides comprehensive monitoring for workload clusters.

type WorkloadMonitor struct {

	registry            *ClusterRegistry

	healthChecker       *HealthChecker

	performanceMonitor  *PerformanceMonitor

	resourceMonitor     *ResourceMonitor

	alertManager        *AlertManager

	dashboardIntegrator *DashboardIntegrator

	predictiveAnalyzer  *PredictiveAnalyzer

	autoRemediator      *AutoRemediator

	logger              logr.Logger

	metrics             *monitoringMetrics

	client              client.Client

	mu                  sync.RWMutex

	stopCh              chan struct{}

}



// HealthChecker monitors cluster health.

type HealthChecker struct {

	healthChecks  map[string]*HealthCheck

	healthHistory map[string][]*HealthSnapshot

	thresholds    HealthThresholds

	logger        logr.Logger

	mu            sync.RWMutex

}



// HealthCheck defines a health check configuration.

type HealthCheck struct {

	Name       string                 `json:"name"`

	Type       string                 `json:"type"`

	Target     string                 `json:"target"`

	Interval   time.Duration          `json:"interval"`

	Timeout    time.Duration          `json:"timeout"`

	Retries    int                    `json:"retries"`

	Enabled    bool                   `json:"enabled"`

	Parameters map[string]interface{} `json:"parameters"`

}



// HealthSnapshot represents a point-in-time health snapshot.

type HealthSnapshot struct {

	Timestamp       time.Time          `json:"timestamp"`

	ClusterID       string             `json:"cluster_id"`

	OverallHealth   string             `json:"overall_health"`

	ComponentHealth map[string]string  `json:"component_health"`

	Metrics         map[string]float64 `json:"metrics"`

	Issues          []HealthIssue      `json:"issues"`

}



// HealthThresholds defines health check thresholds.

type HealthThresholds struct {

	CPUUtilization    float64 `json:"cpu_utilization"`

	MemoryUtilization float64 `json:"memory_utilization"`

	DiskUtilization   float64 `json:"disk_utilization"`

	NetworkLatency    float64 `json:"network_latency"`

	ErrorRate         float64 `json:"error_rate"`

	NodeReadiness     float64 `json:"node_readiness"`

}



// PerformanceMonitor monitors cluster performance metrics.

type PerformanceMonitor struct {

	metrics    map[string]*PerformanceMetrics

	benchmarks map[string]*PerformanceBenchmark

	trends     map[string]*PerformanceTrend

	baselines  map[string]*PerformanceBaseline

	logger     logr.Logger

	mu         sync.RWMutex

}



// PerformanceMetrics contains performance metrics for a cluster.

type PerformanceMetrics struct {

	ClusterID      string             `json:"cluster_id"`

	Timestamp      time.Time          `json:"timestamp"`

	CPUMetrics     CPUMetrics         `json:"cpu_metrics"`

	MemoryMetrics  MemoryMetrics      `json:"memory_metrics"`

	StorageMetrics StorageMetrics     `json:"storage_metrics"`

	NetworkMetrics NetworkMetrics     `json:"network_metrics"`

	PodMetrics     PodMetrics         `json:"pod_metrics"`

	ServiceMetrics ServiceMetrics     `json:"service_metrics"`

	CustomMetrics  map[string]float64 `json:"custom_metrics"`

}



// CPUMetrics contains CPU-related metrics.

type CPUMetrics struct {

	TotalCores         int     `json:"total_cores"`

	UsedCores          float64 `json:"used_cores"`

	UtilizationPercent float64 `json:"utilization_percent"`

	LoadAverage1m      float64 `json:"load_average_1m"`

	LoadAverage5m      float64 `json:"load_average_5m"`

	LoadAverage15m     float64 `json:"load_average_15m"`

	ThrottledCPU       float64 `json:"throttled_cpu"`

}



// MemoryMetrics contains memory-related metrics.

type MemoryMetrics struct {

	TotalBytes         int64   `json:"total_bytes"`

	UsedBytes          int64   `json:"used_bytes"`

	AvailableBytes     int64   `json:"available_bytes"`

	UtilizationPercent float64 `json:"utilization_percent"`

	CacheBytes         int64   `json:"cache_bytes"`

	BufferBytes        int64   `json:"buffer_bytes"`

	SwapTotalBytes     int64   `json:"swap_total_bytes"`

	SwapUsedBytes      int64   `json:"swap_used_bytes"`

}



// StorageMetrics contains storage-related metrics.

type StorageMetrics struct {

	TotalBytes         int64              `json:"total_bytes"`

	UsedBytes          int64              `json:"used_bytes"`

	AvailableBytes     int64              `json:"available_bytes"`

	UtilizationPercent float64            `json:"utilization_percent"`

	IOPSRead           float64            `json:"iops_read"`

	IOPSWrite          float64            `json:"iops_write"`

	ThroughputRead     float64            `json:"throughput_read"`

	ThroughputWrite    float64            `json:"throughput_write"`

	VolumeMetrics      map[string]float64 `json:"volume_metrics"`

}



// NetworkMetrics contains network-related metrics.

type NetworkMetrics struct {

	BytesReceived      int64              `json:"bytes_received"`

	BytesTransmitted   int64              `json:"bytes_transmitted"`

	PacketsReceived    int64              `json:"packets_received"`

	PacketsTransmitted int64              `json:"packets_transmitted"`

	ErrorsReceived     int64              `json:"errors_received"`

	ErrorsTransmitted  int64              `json:"errors_transmitted"`

	Latency            float64            `json:"latency"`

	InterfaceMetrics   map[string]float64 `json:"interface_metrics"`

}



// PodMetrics contains pod-related metrics.

type PodMetrics struct {

	TotalPods        int            `json:"total_pods"`

	RunningPods      int            `json:"running_pods"`

	PendingPods      int            `json:"pending_pods"`

	FailedPods       int            `json:"failed_pods"`

	RestartCount     int64          `json:"restart_count"`

	PodsByPhase      map[string]int `json:"pods_by_phase"`

	NamespaceMetrics map[string]int `json:"namespace_metrics"`

}



// ServiceMetrics contains service-related metrics.

type ServiceMetrics struct {

	TotalServices    int            `json:"total_services"`

	ExternalServices int            `json:"external_services"`

	RequestRate      float64        `json:"request_rate"`

	ErrorRate        float64        `json:"error_rate"`

	ResponseTime     float64        `json:"response_time"`

	ServicesByType   map[string]int `json:"services_by_type"`

}



// PerformanceBenchmark represents a performance benchmark.

type PerformanceBenchmark struct {

	Name       string             `json:"name"`

	ClusterID  string             `json:"cluster_id"`

	Category   string             `json:"category"`

	Metrics    map[string]float64 `json:"metrics"`

	Score      float64            `json:"score"`

	Percentile float64            `json:"percentile"`

	Timestamp  time.Time          `json:"timestamp"`

}



// PerformanceTrend represents performance trends.

type PerformanceTrend struct {

	ClusterID  string  `json:"cluster_id"`

	MetricName string  `json:"metric_name"`

	Trend      string  `json:"trend"`

	ChangeRate float64 `json:"change_rate"`

	Prediction float64 `json:"prediction"`

	Confidence float64 `json:"confidence"`

	TimeRange  string  `json:"time_range"`

}



// PerformanceBaseline represents performance baselines.

type PerformanceBaseline struct {

	ClusterID     string    `json:"cluster_id"`

	MetricName    string    `json:"metric_name"`

	BaselineValue float64   `json:"baseline_value"`

	Tolerance     float64   `json:"tolerance"`

	CreatedAt     time.Time `json:"created_at"`

	UpdatedAt     time.Time `json:"updated_at"`

}



// ResourceMonitor monitors resource usage and availability.

type ResourceMonitor struct {

	resourceUsage   map[string]*ResourceUsage

	quotas          map[string]*ResourceQuotas

	recommendations map[string]*ResourceRecommendation

	logger          logr.Logger

	mu              sync.RWMutex

}



// ResourceUsage tracks resource usage for a cluster.

type ResourceUsage struct {

	ClusterID      string                     `json:"cluster_id"`

	Timestamp      time.Time                  `json:"timestamp"`

	NodeResources  map[string]*NodeResource   `json:"node_resources"`

	NamespaceUsage map[string]*NamespaceUsage `json:"namespace_usage"`

	TotalUsage     ResourceCapacity           `json:"total_usage"`

	Efficiency     ResourceEfficiency         `json:"efficiency"`

}



// NodeResource represents resource usage for a node.

type NodeResource struct {

	NodeName     string            `json:"node_name"`

	CPUUsage     resource.Quantity `json:"cpu_usage"`

	MemoryUsage  resource.Quantity `json:"memory_usage"`

	StorageUsage resource.Quantity `json:"storage_usage"`

	PodCount     int               `json:"pod_count"`

	PodCapacity  int               `json:"pod_capacity"`

}



// NamespaceUsage represents resource usage for a namespace.

type NamespaceUsage struct {

	Namespace    string            `json:"namespace"`

	CPUUsage     resource.Quantity `json:"cpu_usage"`

	MemoryUsage  resource.Quantity `json:"memory_usage"`

	StorageUsage resource.Quantity `json:"storage_usage"`

	PodCount     int               `json:"pod_count"`

}



// ResourceQuotas represents resource quotas for a cluster.

type ResourceQuotas struct {

	ClusterID       string                                  `json:"cluster_id"`

	GlobalQuotas    map[string]resource.Quantity            `json:"global_quotas"`

	NamespaceQuotas map[string]map[string]resource.Quantity `json:"namespace_quotas"`

	Enforcement     string                                  `json:"enforcement"`

}



// ResourceRecommendation represents resource optimization recommendations.

type ResourceRecommendation struct {

	ClusterID   string    `json:"cluster_id"`

	Type        string    `json:"type"`

	Resource    string    `json:"resource"`

	Current     float64   `json:"current"`

	Recommended float64   `json:"recommended"`

	Reason      string    `json:"reason"`

	Impact      string    `json:"impact"`

	Confidence  float64   `json:"confidence"`

	CreatedAt   time.Time `json:"created_at"`

}



// ResourceEfficiency tracks resource efficiency metrics.

type ResourceEfficiency struct {

	CPUEfficiency     float64 `json:"cpu_efficiency"`

	MemoryEfficiency  float64 `json:"memory_efficiency"`

	StorageEfficiency float64 `json:"storage_efficiency"`

	OverallEfficiency float64 `json:"overall_efficiency"`

	WastePercentage   float64 `json:"waste_percentage"`

}



// AlertManager manages alerts and notifications.

type AlertManager struct {

	alertRules           map[string]*AlertRule

	activeAlerts         map[string]*Alert

	alertHistory         map[string][]*Alert

	notificationChannels map[string]*NotificationChannel

	suppressions         map[string]*AlertSuppression

	escalations          map[string]*AlertEscalation

	logger               logr.Logger

	mu                   sync.RWMutex

}



// AlertRule defines an alert rule.

type AlertRule struct {

	ID          string            `json:"id"`

	Name        string            `json:"name"`

	Description string            `json:"description"`

	Query       string            `json:"query"`

	Condition   string            `json:"condition"`

	Threshold   float64           `json:"threshold"`

	Duration    time.Duration     `json:"duration"`

	Severity    string            `json:"severity"`

	Labels      map[string]string `json:"labels"`

	Annotations map[string]string `json:"annotations"`

	Enabled     bool              `json:"enabled"`

}



// Alert represents an active alert.

type Alert struct {

	ID          string            `json:"id"`

	RuleID      string            `json:"rule_id"`

	ClusterID   string            `json:"cluster_id"`

	Status      string            `json:"status"`

	Severity    string            `json:"severity"`

	Message     string            `json:"message"`

	Value       float64           `json:"value"`

	Labels      map[string]string `json:"labels"`

	Annotations map[string]string `json:"annotations"`

	StartsAt    time.Time         `json:"starts_at"`

	EndsAt      time.Time         `json:"ends_at,omitempty"`

	UpdatedAt   time.Time         `json:"updated_at"`

}



// NotificationChannel represents a notification channel.

type NotificationChannel struct {

	Name    string                 `json:"name"`

	Type    string                 `json:"type"`

	Config  map[string]interface{} `json:"config"`

	Enabled bool                   `json:"enabled"`

}



// AlertSuppression represents alert suppression rules.

type AlertSuppression struct {

	ID        string            `json:"id"`

	Matchers  map[string]string `json:"matchers"`

	StartTime time.Time         `json:"start_time"`

	EndTime   time.Time         `json:"end_time"`

	Reason    string            `json:"reason"`

}



// AlertEscalation represents alert escalation rules.

type AlertEscalation struct {

	RuleID  string            `json:"rule_id"`

	Levels  []EscalationLevel `json:"levels"`

	Enabled bool              `json:"enabled"`

}



// EscalationLevel represents an escalation level.

type EscalationLevel struct {

	Level    int           `json:"level"`

	Duration time.Duration `json:"duration"`

	Channels []string      `json:"channels"`

}



// DashboardIntegrator integrates with monitoring dashboards.

type DashboardIntegrator struct {

	dashboards  map[string]*Dashboard

	datasources map[string]*DataSource

	panels      map[string]*Panel

	logger      logr.Logger

	mu          sync.RWMutex

}



// Dashboard represents a monitoring dashboard.

type Dashboard struct {

	ID          string                 `json:"id"`

	Title       string                 `json:"title"`

	Description string                 `json:"description"`

	Tags        []string               `json:"tags"`

	Panels      []string               `json:"panels"`

	Variables   map[string]interface{} `json:"variables"`

	RefreshRate string                 `json:"refresh_rate"`

}



// DataSource represents a data source configuration.

type DataSource struct {

	Name     string                 `json:"name"`

	Type     string                 `json:"type"`

	URL      string                 `json:"url"`

	Auth     map[string]interface{} `json:"auth"`

	Settings map[string]interface{} `json:"settings"`

}



// Panel represents a dashboard panel.

type Panel struct {

	ID            string                 `json:"id"`

	Title         string                 `json:"title"`

	Type          string                 `json:"type"`

	Query         string                 `json:"query"`

	DataSource    string                 `json:"data_source"`

	Visualization map[string]interface{} `json:"visualization"`

	Thresholds    []Threshold            `json:"thresholds"`

}



// Threshold represents a panel threshold.

type Threshold struct {

	Value float64 `json:"value"`

	Color string  `json:"color"`

	Mode  string  `json:"mode"`

}



// PredictiveAnalyzer provides predictive analytics.

type PredictiveAnalyzer struct {

	models      map[string]*PredictiveModel

	predictions map[string]*Prediction

	anomalies   map[string][]*Anomaly

	logger      logr.Logger

	mu          sync.RWMutex

}



// PredictiveModel represents a predictive model.

type PredictiveModel struct {

	ID         string                 `json:"id"`

	Name       string                 `json:"name"`

	Type       string                 `json:"type"`

	MetricName string                 `json:"metric_name"`

	Parameters map[string]interface{} `json:"parameters"`

	Accuracy   float64                `json:"accuracy"`

	TrainedAt  time.Time              `json:"trained_at"`

	LastUsed   time.Time              `json:"last_used"`

}



// Prediction represents a future prediction.

type Prediction struct {

	ClusterID  string        `json:"cluster_id"`

	MetricName string        `json:"metric_name"`

	Value      float64       `json:"value"`

	Confidence float64       `json:"confidence"`

	Timestamp  time.Time     `json:"timestamp"`

	Horizon    time.Duration `json:"horizon"`

}



// Anomaly represents an anomaly detection.

type Anomaly struct {

	ClusterID     string    `json:"cluster_id"`

	MetricName    string    `json:"metric_name"`

	Value         float64   `json:"value"`

	ExpectedValue float64   `json:"expected_value"`

	Deviation     float64   `json:"deviation"`

	Severity      string    `json:"severity"`

	DetectedAt    time.Time `json:"detected_at"`

}



// AutoRemediator provides automated remediation.

type AutoRemediator struct {

	remediationRules   map[string]*RemediationRule

	remediationHistory map[string][]*RemediationAction

	logger             logr.Logger

	mu                 sync.RWMutex

}



// RemediationRule defines an automated remediation rule.

type RemediationRule struct {

	ID           string        `json:"id"`

	Name         string        `json:"name"`

	Trigger      string        `json:"trigger"`

	Condition    string        `json:"condition"`

	Actions      []string      `json:"actions"`

	Cooldown     time.Duration `json:"cooldown"`

	MaxRetries   int           `json:"max_retries"`

	Enabled      bool          `json:"enabled"`

	SafetyChecks []string      `json:"safety_checks"`

}



// RemediationAction represents a remediation action taken.

type RemediationAction struct {

	ID          string    `json:"id"`

	RuleID      string    `json:"rule_id"`

	ClusterID   string    `json:"cluster_id"`

	Action      string    `json:"action"`

	Status      string    `json:"status"`

	Result      string    `json:"result"`

	ExecutedAt  time.Time `json:"executed_at"`

	CompletedAt time.Time `json:"completed_at"`

}



// monitoringMetrics contains Prometheus metrics for monitoring.

type monitoringMetrics struct {

	clusterHealth       *prometheus.GaugeVec

	performanceScore    *prometheus.GaugeVec

	resourceUtilization *prometheus.GaugeVec

	activeAlerts        *prometheus.GaugeVec

	remediationActions  *prometheus.CounterVec

	anomalyDetections   *prometheus.CounterVec

	predictionAccuracy  *prometheus.GaugeVec

	healthCheckDuration *prometheus.HistogramVec

}



// NewWorkloadMonitor creates a new workload monitor.

func NewWorkloadMonitor(client client.Client, registry *ClusterRegistry, logger logr.Logger) *WorkloadMonitor {

	metrics := &monitoringMetrics{

		clusterHealth: prometheus.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "nephio_cluster_health_score",

				Help: "Cluster health score (0-100)",

			},

			[]string{"cluster", "component"},

		),

		performanceScore: prometheus.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "nephio_cluster_performance_score",

				Help: "Cluster performance score (0-100)",

			},

			[]string{"cluster", "category"},

		),

		resourceUtilization: prometheus.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "nephio_cluster_resource_utilization_percent",

				Help: "Cluster resource utilization percentage",

			},

			[]string{"cluster", "resource"},

		),

		activeAlerts: prometheus.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "nephio_cluster_active_alerts",

				Help: "Number of active alerts",

			},

			[]string{"cluster", "severity"},

		),

		remediationActions: prometheus.NewCounterVec(

			prometheus.CounterOpts{

				Name: "nephio_remediation_actions_total",

				Help: "Total number of remediation actions",

			},

			[]string{"cluster", "action", "result"},

		),

		anomalyDetections: prometheus.NewCounterVec(

			prometheus.CounterOpts{

				Name: "nephio_anomaly_detections_total",

				Help: "Total number of anomaly detections",

			},

			[]string{"cluster", "metric", "severity"},

		),

		predictionAccuracy: prometheus.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "nephio_prediction_accuracy",

				Help: "Accuracy of predictive models",

			},

			[]string{"model", "metric"},

		),

		healthCheckDuration: prometheus.NewHistogramVec(

			prometheus.HistogramOpts{

				Name:    "nephio_health_check_duration_seconds",

				Help:    "Duration of health checks",

				Buckets: prometheus.DefBuckets,

			},

			[]string{"cluster", "check_type"},

		),

	}



	// Register metrics.

	prometheus.MustRegister(

		metrics.clusterHealth,

		metrics.performanceScore,

		metrics.resourceUtilization,

		metrics.activeAlerts,

		metrics.remediationActions,

		metrics.anomalyDetections,

		metrics.predictionAccuracy,

		metrics.healthCheckDuration,

	)



	return &WorkloadMonitor{

		registry: registry,

		healthChecker: &HealthChecker{

			healthChecks:  make(map[string]*HealthCheck),

			healthHistory: make(map[string][]*HealthSnapshot),

			thresholds: HealthThresholds{

				CPUUtilization:    80.0,

				MemoryUtilization: 85.0,

				DiskUtilization:   90.0,

				NetworkLatency:    100.0,

				ErrorRate:         0.01,

				NodeReadiness:     0.95,

			},

			logger: logger.WithName("health-checker"),

		},

		performanceMonitor: &PerformanceMonitor{

			metrics:    make(map[string]*PerformanceMetrics),

			benchmarks: make(map[string]*PerformanceBenchmark),

			trends:     make(map[string]*PerformanceTrend),

			baselines:  make(map[string]*PerformanceBaseline),

			logger:     logger.WithName("performance-monitor"),

		},

		resourceMonitor: &ResourceMonitor{

			resourceUsage:   make(map[string]*ResourceUsage),

			quotas:          make(map[string]*ResourceQuotas),

			recommendations: make(map[string]*ResourceRecommendation),

			logger:          logger.WithName("resource-monitor"),

		},

		alertManager: &AlertManager{

			alertRules:           make(map[string]*AlertRule),

			activeAlerts:         make(map[string]*Alert),

			alertHistory:         make(map[string][]*Alert),

			notificationChannels: make(map[string]*NotificationChannel),

			suppressions:         make(map[string]*AlertSuppression),

			escalations:          make(map[string]*AlertEscalation),

			logger:               logger.WithName("alert-manager"),

		},

		dashboardIntegrator: &DashboardIntegrator{

			dashboards:  make(map[string]*Dashboard),

			datasources: make(map[string]*DataSource),

			panels:      make(map[string]*Panel),

			logger:      logger.WithName("dashboard-integrator"),

		},

		predictiveAnalyzer: &PredictiveAnalyzer{

			models:      make(map[string]*PredictiveModel),

			predictions: make(map[string]*Prediction),

			anomalies:   make(map[string][]*Anomaly),

			logger:      logger.WithName("predictive-analyzer"),

		},

		autoRemediator: &AutoRemediator{

			remediationRules:   make(map[string]*RemediationRule),

			remediationHistory: make(map[string][]*RemediationAction),

			logger:             logger.WithName("auto-remediator"),

		},

		logger:  logger.WithName("workload-monitor"),

		metrics: metrics,

		client:  client,

		stopCh:  make(chan struct{}),

	}

}



// Start starts the workload monitor.

func (wm *WorkloadMonitor) Start(ctx context.Context) error {

	wm.logger.Info("Starting workload monitor")



	// Start monitoring loops.

	go wm.runHealthChecks(ctx)

	go wm.runPerformanceMonitoring(ctx)

	go wm.runResourceMonitoring(ctx)

	go wm.runAlertProcessing(ctx)

	go wm.runPredictiveAnalysis(ctx)

	go wm.runAutoRemediation(ctx)



	// Initialize default configurations.

	wm.initializeDefaults()



	return nil

}



// Stop stops the workload monitor.

func (wm *WorkloadMonitor) Stop() {

	wm.logger.Info("Stopping workload monitor")

	close(wm.stopCh)

}



// GetClusterHealth retrieves health information for a cluster.

func (wm *WorkloadMonitor) GetClusterHealth(clusterID string) (*HealthSnapshot, error) {

	wm.healthChecker.mu.RLock()

	defer wm.healthChecker.mu.RUnlock()



	history, exists := wm.healthChecker.healthHistory[clusterID]

	if !exists || len(history) == 0 {

		return nil, fmt.Errorf("no health data available for cluster %s", clusterID)

	}



	return history[len(history)-1], nil

}



// GetPerformanceMetrics retrieves performance metrics for a cluster.

func (wm *WorkloadMonitor) GetPerformanceMetrics(clusterID string) (*PerformanceMetrics, error) {

	wm.performanceMonitor.mu.RLock()

	defer wm.performanceMonitor.mu.RUnlock()



	metrics, exists := wm.performanceMonitor.metrics[clusterID]

	if !exists {

		return nil, fmt.Errorf("no performance metrics available for cluster %s", clusterID)

	}



	return metrics, nil

}



// GetResourceUsage retrieves resource usage for a cluster.

func (wm *WorkloadMonitor) GetResourceUsage(clusterID string) (*ResourceUsage, error) {

	wm.resourceMonitor.mu.RLock()

	defer wm.resourceMonitor.mu.RUnlock()



	usage, exists := wm.resourceMonitor.resourceUsage[clusterID]

	if !exists {

		return nil, fmt.Errorf("no resource usage data available for cluster %s", clusterID)

	}



	return usage, nil

}



// GetActiveAlerts retrieves active alerts for a cluster.

func (wm *WorkloadMonitor) GetActiveAlerts(clusterID string) ([]*Alert, error) {

	wm.alertManager.mu.RLock()

	defer wm.alertManager.mu.RUnlock()



	alerts := []*Alert{}

	for _, alert := range wm.alertManager.activeAlerts {

		if alert.ClusterID == clusterID {

			alerts = append(alerts, alert)

		}

	}



	return alerts, nil

}



// GetPredictions retrieves predictions for a cluster.

func (wm *WorkloadMonitor) GetPredictions(clusterID string) ([]*Prediction, error) {

	wm.predictiveAnalyzer.mu.RLock()

	defer wm.predictiveAnalyzer.mu.RUnlock()



	predictions := []*Prediction{}

	for _, prediction := range wm.predictiveAnalyzer.predictions {

		if prediction.ClusterID == clusterID {

			predictions = append(predictions, prediction)

		}

	}



	return predictions, nil

}



// CreateAlert creates a new alert.

func (wm *WorkloadMonitor) CreateAlert(alert *Alert) error {

	return wm.alertManager.CreateAlert(alert)

}



// CreateDashboard creates a new dashboard.

func (wm *WorkloadMonitor) CreateDashboard(dashboard *Dashboard) error {

	return wm.dashboardIntegrator.CreateDashboard(dashboard)

}



// AddRemediationRule adds a new remediation rule.

func (wm *WorkloadMonitor) AddRemediationRule(rule *RemediationRule) error {

	return wm.autoRemediator.AddRule(rule)

}



// Private methods.



func (wm *WorkloadMonitor) initializeDefaults() {

	// Initialize default health checks.

	wm.initializeDefaultHealthChecks()



	// Initialize default alert rules.

	wm.initializeDefaultAlertRules()



	// Initialize default dashboards.

	wm.initializeDefaultDashboards()



	// Initialize default remediation rules.

	wm.initializeDefaultRemediationRules()

}



func (wm *WorkloadMonitor) initializeDefaultHealthChecks() {

	defaultChecks := []*HealthCheck{

		{

			Name:     "api-server-health",

			Type:     "kubernetes-api",

			Target:   "kube-apiserver",

			Interval: 30 * time.Second,

			Timeout:  10 * time.Second,

			Retries:  3,

			Enabled:  true,

		},

		{

			Name:     "node-health",

			Type:     "node-status",

			Target:   "nodes",

			Interval: 60 * time.Second,

			Timeout:  15 * time.Second,

			Retries:  2,

			Enabled:  true,

		},

		{

			Name:     "pod-health",

			Type:     "pod-status",

			Target:   "pods",

			Interval: 30 * time.Second,

			Timeout:  10 * time.Second,

			Retries:  2,

			Enabled:  true,

		},

		{

			Name:     "storage-health",

			Type:     "storage-status",

			Target:   "persistent-volumes",

			Interval: 120 * time.Second,

			Timeout:  20 * time.Second,

			Retries:  2,

			Enabled:  true,

		},

	}



	wm.healthChecker.mu.Lock()

	for _, check := range defaultChecks {

		wm.healthChecker.healthChecks[check.Name] = check

	}

	wm.healthChecker.mu.Unlock()

}



func (wm *WorkloadMonitor) initializeDefaultAlertRules() {

	defaultRules := []*AlertRule{

		{

			ID:          "high-cpu-usage",

			Name:        "High CPU Usage",

			Description: "CPU usage is above threshold",

			Query:       "cpu_utilization",

			Condition:   ">",

			Threshold:   80.0,

			Duration:    5 * time.Minute,

			Severity:    "warning",

			Enabled:     true,

		},

		{

			ID:          "high-memory-usage",

			Name:        "High Memory Usage",

			Description: "Memory usage is above threshold",

			Query:       "memory_utilization",

			Condition:   ">",

			Threshold:   85.0,

			Duration:    5 * time.Minute,

			Severity:    "warning",

			Enabled:     true,

		},

		{

			ID:          "node-down",

			Name:        "Node Down",

			Description: "Node is not ready",

			Query:       "node_readiness",

			Condition:   "<",

			Threshold:   1.0,

			Duration:    2 * time.Minute,

			Severity:    "critical",

			Enabled:     true,

		},

		{

			ID:          "pod-restart-loop",

			Name:        "Pod Restart Loop",

			Description: "Pod is restarting frequently",

			Query:       "pod_restart_rate",

			Condition:   ">",

			Threshold:   5.0,

			Duration:    10 * time.Minute,

			Severity:    "warning",

			Enabled:     true,

		},

	}



	wm.alertManager.mu.Lock()

	for _, rule := range defaultRules {

		wm.alertManager.alertRules[rule.ID] = rule

	}

	wm.alertManager.mu.Unlock()

}



func (wm *WorkloadMonitor) initializeDefaultDashboards() {

	defaultDashboards := []*Dashboard{

		{

			ID:          "cluster-overview",

			Title:       "Cluster Overview",

			Description: "High-level cluster health and performance overview",

			Tags:        []string{"overview", "health"},

			Panels:      []string{"health-panel", "resource-panel", "performance-panel"},

			RefreshRate: "30s",

		},

		{

			ID:          "resource-utilization",

			Title:       "Resource Utilization",

			Description: "Detailed resource utilization metrics",

			Tags:        []string{"resources", "utilization"},

			Panels:      []string{"cpu-panel", "memory-panel", "storage-panel"},

			RefreshRate: "15s",

		},

		{

			ID:          "performance-metrics",

			Title:       "Performance Metrics",

			Description: "Performance and latency metrics",

			Tags:        []string{"performance", "latency"},

			Panels:      []string{"latency-panel", "throughput-panel", "errors-panel"},

			RefreshRate: "10s",

		},

	}



	wm.dashboardIntegrator.mu.Lock()

	for _, dashboard := range defaultDashboards {

		wm.dashboardIntegrator.dashboards[dashboard.ID] = dashboard

	}

	wm.dashboardIntegrator.mu.Unlock()

}



func (wm *WorkloadMonitor) initializeDefaultRemediationRules() {

	defaultRules := []*RemediationRule{

		{

			ID:           "restart-failed-pods",

			Name:         "Restart Failed Pods",

			Trigger:      "pod_failed",

			Condition:    "status == 'Failed'",

			Actions:      []string{"delete-pod"},

			Cooldown:     5 * time.Minute,

			MaxRetries:   3,

			Enabled:      true,

			SafetyChecks: []string{"check-pod-age", "check-restart-count"},

		},

		{

			ID:           "scale-up-on-high-cpu",

			Name:         "Scale Up on High CPU",

			Trigger:      "high_cpu_usage",

			Condition:    "cpu_utilization > 80",

			Actions:      []string{"scale-deployment"},

			Cooldown:     10 * time.Minute,

			MaxRetries:   2,

			Enabled:      false, // Disabled by default for safety

			SafetyChecks: []string{"check-max-replicas", "check-resource-limits"},

		},

	}



	wm.autoRemediator.mu.Lock()

	for _, rule := range defaultRules {

		wm.autoRemediator.remediationRules[rule.ID] = rule

	}

	wm.autoRemediator.mu.Unlock()

}



func (wm *WorkloadMonitor) runHealthChecks(ctx context.Context) {

	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return

		case <-wm.stopCh:

			return

		case <-ticker.C:

			wm.performHealthChecks(ctx)

		}

	}

}



func (wm *WorkloadMonitor) performHealthChecks(ctx context.Context) {

	clusters := wm.registry.ListClusters()



	for _, cluster := range clusters {

		go wm.checkClusterHealth(ctx, cluster)

	}

}



func (wm *WorkloadMonitor) checkClusterHealth(ctx context.Context, cluster *ClusterEntry) {

	timer := prometheus.NewTimer(wm.metrics.healthCheckDuration.WithLabelValues(cluster.Metadata.ID, "comprehensive"))

	defer timer.ObserveDuration()



	snapshot := &HealthSnapshot{

		Timestamp:       time.Now(),

		ClusterID:       cluster.Metadata.ID,

		ComponentHealth: make(map[string]string),

		Metrics:         make(map[string]float64),

		Issues:          []HealthIssue{},

	}



	// Check API server health.

	apiHealth := wm.checkAPIServerHealth(ctx, cluster)

	snapshot.ComponentHealth["api-server"] = apiHealth

	if apiHealth != "healthy" {

		snapshot.Issues = append(snapshot.Issues, HealthIssue{

			Severity:    "critical",

			Component:   "api-server",

			Description: "API server is not healthy",

			DetectedAt:  time.Now(),

		})

	}



	// Check node health.

	nodeHealth, nodeMetrics := wm.checkNodeHealth(ctx, cluster)

	snapshot.ComponentHealth["nodes"] = nodeHealth

	for k, v := range nodeMetrics {

		snapshot.Metrics[k] = v

	}



	// Check pod health.

	podHealth, podMetrics := wm.checkPodHealth(ctx, cluster)

	snapshot.ComponentHealth["pods"] = podHealth

	for k, v := range podMetrics {

		snapshot.Metrics[k] = v

	}



	// Check storage health.

	storageHealth := wm.checkStorageHealth(ctx, cluster)

	snapshot.ComponentHealth["storage"] = storageHealth



	// Calculate overall health.

	snapshot.OverallHealth = wm.calculateOverallHealth(snapshot)



	// Store health snapshot.

	wm.healthChecker.mu.Lock()

	wm.healthChecker.healthHistory[cluster.Metadata.ID] = append(

		wm.healthChecker.healthHistory[cluster.Metadata.ID],

		snapshot,

	)



	// Keep only last 100 snapshots.

	history := wm.healthChecker.healthHistory[cluster.Metadata.ID]

	if len(history) > 100 {

		wm.healthChecker.healthHistory[cluster.Metadata.ID] = history[len(history)-100:]

	}

	wm.healthChecker.mu.Unlock()



	// Update metrics.

	healthScore := wm.calculateHealthScore(snapshot)

	wm.metrics.clusterHealth.WithLabelValues(cluster.Metadata.ID, "overall").Set(healthScore)



	for component, health := range snapshot.ComponentHealth {

		score := 100.0

		if health != "healthy" {

			score = 0.0

		}

		wm.metrics.clusterHealth.WithLabelValues(cluster.Metadata.ID, component).Set(score)

	}

}



func (wm *WorkloadMonitor) checkAPIServerHealth(ctx context.Context, cluster *ClusterEntry) string {

	// Check if we can list namespaces.

	_, err := cluster.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})

	if err != nil {

		return "unhealthy"

	}

	return "healthy"

}



func (wm *WorkloadMonitor) checkNodeHealth(ctx context.Context, cluster *ClusterEntry) (string, map[string]float64) {

	metrics := make(map[string]float64)



	nodeList, err := cluster.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})

	if err != nil {

		return "unknown", metrics

	}



	totalNodes := len(nodeList.Items)

	readyNodes := 0



	for _, node := range nodeList.Items {

		for _, condition := range node.Status.Conditions {

			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {

				readyNodes++

				break

			}

		}

	}



	metrics["total_nodes"] = float64(totalNodes)

	metrics["ready_nodes"] = float64(readyNodes)



	if totalNodes == 0 {

		return "unknown", metrics

	}



	readinessRatio := float64(readyNodes) / float64(totalNodes)

	metrics["node_readiness_ratio"] = readinessRatio



	if readinessRatio >= wm.healthChecker.thresholds.NodeReadiness {

		return "healthy", metrics

	} else if readinessRatio > 0.5 {

		return "degraded", metrics

	}



	return "unhealthy", metrics

}



func (wm *WorkloadMonitor) checkPodHealth(ctx context.Context, cluster *ClusterEntry) (string, map[string]float64) {

	metrics := make(map[string]float64)



	podList, err := cluster.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})

	if err != nil {

		return "unknown", metrics

	}



	totalPods := len(podList.Items)

	runningPods := 0

	failedPods := 0

	pendingPods := 0



	for _, pod := range podList.Items {

		switch pod.Status.Phase {

		case corev1.PodRunning:

			runningPods++

		case corev1.PodFailed:

			failedPods++

		case corev1.PodPending:

			pendingPods++

		}

	}



	metrics["total_pods"] = float64(totalPods)

	metrics["running_pods"] = float64(runningPods)

	metrics["failed_pods"] = float64(failedPods)

	metrics["pending_pods"] = float64(pendingPods)



	if totalPods == 0 {

		return "healthy", metrics

	}



	failureRate := float64(failedPods) / float64(totalPods)

	if failureRate <= 0.05 {

		return "healthy", metrics

	} else if failureRate <= 0.15 {

		return "degraded", metrics

	}



	return "unhealthy", metrics

}



func (wm *WorkloadMonitor) checkStorageHealth(ctx context.Context, cluster *ClusterEntry) string {

	// Check persistent volumes.

	pvList, err := cluster.Clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})

	if err != nil {

		return "unknown"

	}



	failedPVs := 0

	for _, pv := range pvList.Items {

		if pv.Status.Phase == corev1.VolumeFailed {

			failedPVs++

		}

	}



	if failedPVs == 0 {

		return "healthy"

	} else if failedPVs < 3 {

		return "degraded"

	}



	return "unhealthy"

}



func (wm *WorkloadMonitor) calculateOverallHealth(snapshot *HealthSnapshot) string {

	healthyCount := 0

	totalCount := 0



	for _, health := range snapshot.ComponentHealth {

		totalCount++

		if health == "healthy" {

			healthyCount++

		}

	}



	if totalCount == 0 {

		return "unknown"

	}



	ratio := float64(healthyCount) / float64(totalCount)

	if ratio >= 0.8 {

		return "healthy"

	} else if ratio >= 0.5 {

		return "degraded"

	}



	return "unhealthy"

}



func (wm *WorkloadMonitor) calculateHealthScore(snapshot *HealthSnapshot) float64 {

	totalScore := 0.0

	componentCount := 0



	for _, health := range snapshot.ComponentHealth {

		componentCount++

		switch health {

		case "healthy":

			totalScore += 100.0

		case "degraded":

			totalScore += 50.0

		case "unhealthy":

			totalScore += 0.0

		default:

			totalScore += 25.0 // unknown

		}

	}



	if componentCount == 0 {

		return 0.0

	}



	return totalScore / float64(componentCount)

}



func (wm *WorkloadMonitor) runPerformanceMonitoring(ctx context.Context) {

	ticker := time.NewTicker(1 * time.Minute)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return

		case <-wm.stopCh:

			return

		case <-ticker.C:

			wm.collectPerformanceMetrics(ctx)

		}

	}

}



func (wm *WorkloadMonitor) collectPerformanceMetrics(ctx context.Context) {

	clusters := wm.registry.ListClusters()



	for _, cluster := range clusters {

		go wm.collectClusterPerformanceMetrics(ctx, cluster)

	}

}



func (wm *WorkloadMonitor) collectClusterPerformanceMetrics(ctx context.Context, cluster *ClusterEntry) {

	metrics := &PerformanceMetrics{

		ClusterID:     cluster.Metadata.ID,

		Timestamp:     time.Now(),

		CustomMetrics: make(map[string]float64),

	}



	// Collect CPU metrics.

	metrics.CPUMetrics = wm.collectCPUMetrics(ctx, cluster)



	// Collect memory metrics.

	metrics.MemoryMetrics = wm.collectMemoryMetrics(ctx, cluster)



	// Collect storage metrics.

	metrics.StorageMetrics = wm.collectStorageMetrics(ctx, cluster)



	// Collect network metrics.

	metrics.NetworkMetrics = wm.collectNetworkMetrics(ctx, cluster)



	// Collect pod metrics.

	metrics.PodMetrics = wm.collectPodMetrics(ctx, cluster)



	// Collect service metrics.

	metrics.ServiceMetrics = wm.collectServiceMetrics(ctx, cluster)



	// Store metrics.

	wm.performanceMonitor.mu.Lock()

	wm.performanceMonitor.metrics[cluster.Metadata.ID] = metrics

	wm.performanceMonitor.mu.Unlock()



	// Update Prometheus metrics.

	wm.updatePerformanceMetrics(cluster.Metadata.ID, metrics)



	// Analyze trends.

	wm.analyzeTrends(cluster.Metadata.ID, metrics)

}



func (wm *WorkloadMonitor) collectCPUMetrics(ctx context.Context, cluster *ClusterEntry) CPUMetrics {

	// This would typically integrate with metrics-server or Prometheus.

	// Placeholder implementation.

	return CPUMetrics{

		TotalCores:         int(cluster.Metadata.Resources.TotalCPU / 1000), // Convert millicores to cores

		UsedCores:          float64(cluster.Metadata.Resources.TotalCPU-cluster.Metadata.Resources.AvailableCPU) / 1000,

		UtilizationPercent: cluster.Metadata.Resources.Utilization * 100,

		LoadAverage1m:      2.5, // Placeholder

		LoadAverage5m:      2.1, // Placeholder

		LoadAverage15m:     1.8, // Placeholder

		ThrottledCPU:       0.1, // Placeholder

	}

}



func (wm *WorkloadMonitor) collectMemoryMetrics(ctx context.Context, cluster *ClusterEntry) MemoryMetrics {

	totalBytes := cluster.Metadata.Resources.TotalMemory

	usedBytes := cluster.Metadata.Resources.TotalMemory - cluster.Metadata.Resources.AvailableMemory



	return MemoryMetrics{

		TotalBytes:         totalBytes,

		UsedBytes:          usedBytes,

		AvailableBytes:     cluster.Metadata.Resources.AvailableMemory,

		UtilizationPercent: float64(usedBytes) / float64(totalBytes) * 100,

		CacheBytes:         totalBytes / 20,  // Placeholder

		BufferBytes:        totalBytes / 50,  // Placeholder

		SwapTotalBytes:     totalBytes / 4,   // Placeholder

		SwapUsedBytes:      totalBytes / 100, // Placeholder

	}

}



func (wm *WorkloadMonitor) collectStorageMetrics(ctx context.Context, cluster *ClusterEntry) StorageMetrics {

	totalBytes := cluster.Metadata.Resources.TotalStorage

	usedBytes := cluster.Metadata.Resources.TotalStorage - cluster.Metadata.Resources.AvailableStorage



	return StorageMetrics{

		TotalBytes:         totalBytes,

		UsedBytes:          usedBytes,

		AvailableBytes:     cluster.Metadata.Resources.AvailableStorage,

		UtilizationPercent: float64(usedBytes) / float64(totalBytes) * 100,

		IOPSRead:           1000.0, // Placeholder

		IOPSWrite:          800.0,  // Placeholder

		ThroughputRead:     50.0,   // Placeholder MB/s

		ThroughputWrite:    40.0,   // Placeholder MB/s

		VolumeMetrics:      make(map[string]float64),

	}

}



func (wm *WorkloadMonitor) collectNetworkMetrics(ctx context.Context, cluster *ClusterEntry) NetworkMetrics {

	return NetworkMetrics{

		BytesReceived:      1024 * 1024 * 1024, // Placeholder

		BytesTransmitted:   800 * 1024 * 1024,  // Placeholder

		PacketsReceived:    1000000,            // Placeholder

		PacketsTransmitted: 950000,             // Placeholder

		ErrorsReceived:     100,                // Placeholder

		ErrorsTransmitted:  50,                 // Placeholder

		Latency:            5.2,                // Placeholder ms

		InterfaceMetrics:   make(map[string]float64),

	}

}



func (wm *WorkloadMonitor) collectPodMetrics(ctx context.Context, cluster *ClusterEntry) PodMetrics {

	podList, err := cluster.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})

	if err != nil {

		return PodMetrics{}

	}



	metrics := PodMetrics{

		TotalPods:        len(podList.Items),

		PodsByPhase:      make(map[string]int),

		NamespaceMetrics: make(map[string]int),

	}



	var restartCount int64

	for _, pod := range podList.Items {

		// Count pods by phase.

		metrics.PodsByPhase[string(pod.Status.Phase)]++



		// Count pods by namespace.

		metrics.NamespaceMetrics[pod.Namespace]++



		// Count by status.

		switch pod.Status.Phase {

		case corev1.PodRunning:

			metrics.RunningPods++

		case corev1.PodPending:

			metrics.PendingPods++

		case corev1.PodFailed:

			metrics.FailedPods++

		}



		// Sum restart counts.

		for _, container := range pod.Status.ContainerStatuses {

			restartCount += int64(container.RestartCount)

		}

	}



	metrics.RestartCount = restartCount

	return metrics

}



func (wm *WorkloadMonitor) collectServiceMetrics(ctx context.Context, cluster *ClusterEntry) ServiceMetrics {

	serviceList, err := cluster.Clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})

	if err != nil {

		return ServiceMetrics{}

	}



	metrics := ServiceMetrics{

		TotalServices:  len(serviceList.Items),

		ServicesByType: make(map[string]int),

		RequestRate:    1000.0, // Placeholder

		ErrorRate:      0.01,   // Placeholder

		ResponseTime:   50.0,   // Placeholder ms

	}



	for _, service := range serviceList.Items {

		metrics.ServicesByType[string(service.Spec.Type)]++



		if service.Spec.Type == corev1.ServiceTypeLoadBalancer ||

			service.Spec.Type == corev1.ServiceTypeNodePort {

			metrics.ExternalServices++

		}

	}



	return metrics

}



func (wm *WorkloadMonitor) updatePerformanceMetrics(clusterID string, metrics *PerformanceMetrics) {

	wm.metrics.performanceScore.WithLabelValues(clusterID, "cpu").Set(100 - metrics.CPUMetrics.UtilizationPercent)

	wm.metrics.performanceScore.WithLabelValues(clusterID, "memory").Set(100 - metrics.MemoryMetrics.UtilizationPercent)

	wm.metrics.performanceScore.WithLabelValues(clusterID, "storage").Set(100 - metrics.StorageMetrics.UtilizationPercent)



	wm.metrics.resourceUtilization.WithLabelValues(clusterID, "cpu").Set(metrics.CPUMetrics.UtilizationPercent)

	wm.metrics.resourceUtilization.WithLabelValues(clusterID, "memory").Set(metrics.MemoryMetrics.UtilizationPercent)

	wm.metrics.resourceUtilization.WithLabelValues(clusterID, "storage").Set(metrics.StorageMetrics.UtilizationPercent)

}



func (wm *WorkloadMonitor) analyzeTrends(clusterID string, metrics *PerformanceMetrics) {

	// Analyze CPU trend.

	cpuTrend := wm.calculateTrend(clusterID, "cpu_utilization", metrics.CPUMetrics.UtilizationPercent)

	if cpuTrend != nil {

		wm.performanceMonitor.mu.Lock()

		wm.performanceMonitor.trends[fmt.Sprintf("%s-cpu", clusterID)] = cpuTrend

		wm.performanceMonitor.mu.Unlock()

	}



	// Analyze memory trend.

	memoryTrend := wm.calculateTrend(clusterID, "memory_utilization", metrics.MemoryMetrics.UtilizationPercent)

	if memoryTrend != nil {

		wm.performanceMonitor.mu.Lock()

		wm.performanceMonitor.trends[fmt.Sprintf("%s-memory", clusterID)] = memoryTrend

		wm.performanceMonitor.mu.Unlock()

	}

}



func (wm *WorkloadMonitor) calculateTrend(clusterID, metricName string, currentValue float64) *PerformanceTrend {

	// This would implement trend analysis algorithms.

	// Placeholder implementation.

	return &PerformanceTrend{

		ClusterID:  clusterID,

		MetricName: metricName,

		Trend:      "stable",

		ChangeRate: 0.5,

		Prediction: currentValue * 1.05,

		Confidence: 0.75,

		TimeRange:  "1h",

	}

}



func (wm *WorkloadMonitor) runResourceMonitoring(ctx context.Context) {

	ticker := time.NewTicker(2 * time.Minute)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return

		case <-wm.stopCh:

			return

		case <-ticker.C:

			wm.collectResourceUsage(ctx)

		}

	}

}



func (wm *WorkloadMonitor) collectResourceUsage(ctx context.Context) {

	clusters := wm.registry.ListClusters()



	for _, cluster := range clusters {

		go wm.collectClusterResourceUsage(ctx, cluster)

	}

}



func (wm *WorkloadMonitor) collectClusterResourceUsage(ctx context.Context, cluster *ClusterEntry) {

	usage := &ResourceUsage{

		ClusterID:      cluster.Metadata.ID,

		Timestamp:      time.Now(),

		NodeResources:  make(map[string]*NodeResource),

		NamespaceUsage: make(map[string]*NamespaceUsage),

		TotalUsage:     cluster.Metadata.Resources,

	}



	// Collect node resource usage.

	wm.collectNodeResourceUsage(ctx, cluster, usage)



	// Collect namespace resource usage.

	wm.collectNamespaceResourceUsage(ctx, cluster, usage)



	// Calculate resource efficiency.

	usage.Efficiency = wm.calculateResourceEfficiency(usage)



	// Generate recommendations.

	recommendations := wm.generateResourceRecommendations(usage)



	// Store usage data.

	wm.resourceMonitor.mu.Lock()

	wm.resourceMonitor.resourceUsage[cluster.Metadata.ID] = usage

	for _, rec := range recommendations {

		key := fmt.Sprintf("%s-%s", cluster.Metadata.ID, rec.Type)

		wm.resourceMonitor.recommendations[key] = rec

	}

	wm.resourceMonitor.mu.Unlock()

}



func (wm *WorkloadMonitor) collectNodeResourceUsage(ctx context.Context, cluster *ClusterEntry, usage *ResourceUsage) {

	nodeList, err := cluster.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})

	if err != nil {

		wm.logger.Error(err, "Failed to list nodes", "cluster", cluster.Metadata.ID)

		return

	}



	// Try to get metrics from metrics-server.

	metricsClient, err := versioned.NewForConfig(cluster.Config)

	if err != nil {

		wm.logger.Error(err, "Failed to create metrics client", "cluster", cluster.Metadata.ID)

		return

	}



	nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})

	if err != nil {

		wm.logger.Error(err, "Failed to get node metrics", "cluster", cluster.Metadata.ID)

		// Fall back to capacity-based estimates.

		for _, node := range nodeList.Items {

			nodeResource := &NodeResource{

				NodeName:    node.Name,

				CPUUsage:    *resource.NewMilliQuantity(int64(float64(node.Status.Allocatable.Cpu().MilliValue())*0.5), resource.DecimalSI),

				MemoryUsage: *resource.NewQuantity(int64(float64(node.Status.Allocatable.Memory().Value())*0.6), resource.BinarySI),

				PodCapacity: int(node.Status.Capacity.Pods().Value()),

			}



			// Count actual pods on this node.

			podList, _ := cluster.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{

				FieldSelector: "spec.nodeName=" + node.Name,

			})

			nodeResource.PodCount = len(podList.Items)



			usage.NodeResources[node.Name] = nodeResource

		}

		return

	}



	// Use actual metrics.

	for _, nodeMetric := range nodeMetrics.Items {

		nodeResource := &NodeResource{

			NodeName:    nodeMetric.Name,

			CPUUsage:    nodeMetric.Usage[corev1.ResourceCPU],

			MemoryUsage: nodeMetric.Usage[corev1.ResourceMemory],

		}



		// Find corresponding node for capacity info.

		for _, node := range nodeList.Items {

			if node.Name == nodeMetric.Name {

				nodeResource.PodCapacity = int(node.Status.Capacity.Pods().Value())

				break

			}

		}



		// Count pods on this node.

		podList, _ := cluster.Clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{

			FieldSelector: "spec.nodeName=" + nodeMetric.Name,

		})

		nodeResource.PodCount = len(podList.Items)



		usage.NodeResources[nodeMetric.Name] = nodeResource

	}

}



func (wm *WorkloadMonitor) collectNamespaceResourceUsage(ctx context.Context, cluster *ClusterEntry, usage *ResourceUsage) {

	namespaceList, err := cluster.Clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})

	if err != nil {

		wm.logger.Error(err, "Failed to list namespaces", "cluster", cluster.Metadata.ID)

		return

	}



	for _, namespace := range namespaceList.Items {

		nsUsage := &NamespaceUsage{

			Namespace: namespace.Name,

		}



		// Count pods in namespace.

		podList, err := cluster.Clientset.CoreV1().Pods(namespace.Name).List(ctx, metav1.ListOptions{})

		if err == nil {

			nsUsage.PodCount = len(podList.Items)

		}



		// Try to get pod metrics for this namespace.

		metricsClient, err := versioned.NewForConfig(cluster.Config)

		if err == nil {

			podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace.Name).List(ctx, metav1.ListOptions{})

			if err == nil {

				var totalCPU, totalMemory int64

				for _, podMetric := range podMetrics.Items {

					for _, container := range podMetric.Containers {

						totalCPU += container.Usage.Cpu().MilliValue()

						totalMemory += container.Usage.Memory().Value()

					}

				}

				nsUsage.CPUUsage = *resource.NewMilliQuantity(totalCPU, resource.DecimalSI)

				nsUsage.MemoryUsage = *resource.NewQuantity(totalMemory, resource.BinarySI)

			}

		}



		usage.NamespaceUsage[namespace.Name] = nsUsage

	}

}



func (wm *WorkloadMonitor) calculateResourceEfficiency(usage *ResourceUsage) ResourceEfficiency {

	// Calculate CPU efficiency.

	totalCPU := usage.TotalUsage.TotalCPU

	usedCPU := usage.TotalUsage.TotalCPU - usage.TotalUsage.AvailableCPU

	cpuEfficiency := 0.0

	if totalCPU > 0 {

		cpuEfficiency = float64(usedCPU) / float64(totalCPU) * 100

	}



	// Calculate memory efficiency.

	totalMemory := usage.TotalUsage.TotalMemory

	usedMemory := usage.TotalUsage.TotalMemory - usage.TotalUsage.AvailableMemory

	memoryEfficiency := 0.0

	if totalMemory > 0 {

		memoryEfficiency = float64(usedMemory) / float64(totalMemory) * 100

	}



	// Calculate storage efficiency.

	totalStorage := usage.TotalUsage.TotalStorage

	usedStorage := usage.TotalUsage.TotalStorage - usage.TotalUsage.AvailableStorage

	storageEfficiency := 0.0

	if totalStorage > 0 {

		storageEfficiency = float64(usedStorage) / float64(totalStorage) * 100

	}



	// Calculate overall efficiency.

	overallEfficiency := (cpuEfficiency + memoryEfficiency + storageEfficiency) / 3



	// Calculate waste percentage (resources allocated but not used effectively).

	wastePercentage := 100 - overallEfficiency

	if wastePercentage < 0 {

		wastePercentage = 0

	}



	return ResourceEfficiency{

		CPUEfficiency:     cpuEfficiency,

		MemoryEfficiency:  memoryEfficiency,

		StorageEfficiency: storageEfficiency,

		OverallEfficiency: overallEfficiency,

		WastePercentage:   wastePercentage,

	}

}



func (wm *WorkloadMonitor) generateResourceRecommendations(usage *ResourceUsage) []*ResourceRecommendation {

	recommendations := []*ResourceRecommendation{}



	// CPU recommendations.

	if usage.Efficiency.CPUEfficiency < 30 {

		recommendations = append(recommendations, &ResourceRecommendation{

			ClusterID:   usage.ClusterID,

			Type:        "scale-down",

			Resource:    "cpu",

			Current:     usage.Efficiency.CPUEfficiency,

			Recommended: usage.Efficiency.CPUEfficiency + 20,

			Reason:      "Low CPU utilization detected",

			Impact:      "Cost savings",

			Confidence:  0.8,

			CreatedAt:   time.Now(),

		})

	} else if usage.Efficiency.CPUEfficiency > 85 {

		recommendations = append(recommendations, &ResourceRecommendation{

			ClusterID:   usage.ClusterID,

			Type:        "scale-up",

			Resource:    "cpu",

			Current:     usage.Efficiency.CPUEfficiency,

			Recommended: usage.Efficiency.CPUEfficiency - 15,

			Reason:      "High CPU utilization detected",

			Impact:      "Performance improvement",

			Confidence:  0.9,

			CreatedAt:   time.Now(),

		})

	}



	// Memory recommendations.

	if usage.Efficiency.MemoryEfficiency < 40 {

		recommendations = append(recommendations, &ResourceRecommendation{

			ClusterID:   usage.ClusterID,

			Type:        "optimize",

			Resource:    "memory",

			Current:     usage.Efficiency.MemoryEfficiency,

			Recommended: usage.Efficiency.MemoryEfficiency + 15,

			Reason:      "Low memory utilization detected",

			Impact:      "Cost optimization",

			Confidence:  0.7,

			CreatedAt:   time.Now(),

		})

	}



	return recommendations

}



func (wm *WorkloadMonitor) runAlertProcessing(ctx context.Context) {

	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return

		case <-wm.stopCh:

			return

		case <-ticker.C:

			wm.processAlerts(ctx)

		}

	}

}



func (wm *WorkloadMonitor) processAlerts(ctx context.Context) {

	clusters := wm.registry.ListClusters()



	for _, cluster := range clusters {

		wm.evaluateAlertRules(ctx, cluster)

	}



	// Process active alerts for escalation.

	wm.processAlertEscalations(ctx)

}



func (wm *WorkloadMonitor) evaluateAlertRules(ctx context.Context, cluster *ClusterEntry) {

	wm.alertManager.mu.RLock()

	rules := make([]*AlertRule, 0, len(wm.alertManager.alertRules))

	for _, rule := range wm.alertManager.alertRules {

		if rule.Enabled {

			rules = append(rules, rule)

		}

	}

	wm.alertManager.mu.RUnlock()



	for _, rule := range rules {

		wm.evaluateAlertRule(ctx, cluster, rule)

	}

}



func (wm *WorkloadMonitor) evaluateAlertRule(ctx context.Context, cluster *ClusterEntry, rule *AlertRule) {

	// Get current metric value.

	value := wm.getMetricValue(cluster, rule.Query)



	// Check condition.

	triggered := false

	switch rule.Condition {

	case ">":

		triggered = value > rule.Threshold

	case "<":

		triggered = value < rule.Threshold

	case ">=":

		triggered = value >= rule.Threshold

	case "<=":

		triggered = value <= rule.Threshold

	case "==":

		triggered = value == rule.Threshold

	case "!=":

		triggered = value != rule.Threshold

	}



	alertID := fmt.Sprintf("%s-%s", cluster.Metadata.ID, rule.ID)



	wm.alertManager.mu.Lock()

	existingAlert, exists := wm.alertManager.activeAlerts[alertID]



	if triggered {

		if !exists {

			// Create new alert.

			alert := &Alert{

				ID:          alertID,

				RuleID:      rule.ID,

				ClusterID:   cluster.Metadata.ID,

				Status:      "firing",

				Severity:    rule.Severity,

				Message:     fmt.Sprintf("%s: %s (value: %.2f, threshold: %.2f)", rule.Name, rule.Description, value, rule.Threshold),

				Value:       value,

				Labels:      make(map[string]string),

				Annotations: make(map[string]string),

				StartsAt:    time.Now(),

				UpdatedAt:   time.Now(),

			}



			// Copy labels and annotations from rule.

			for k, v := range rule.Labels {

				alert.Labels[k] = v

			}

			for k, v := range rule.Annotations {

				alert.Annotations[k] = v

			}

			alert.Labels["cluster"] = cluster.Metadata.ID



			wm.alertManager.activeAlerts[alertID] = alert



			// Send notifications.

			wm.sendAlertNotifications(alert)



			// Update metrics.

			wm.metrics.activeAlerts.WithLabelValues(cluster.Metadata.ID, rule.Severity).Inc()

		} else {

			// Update existing alert.

			existingAlert.Value = value

			existingAlert.UpdatedAt = time.Now()

			existingAlert.Message = fmt.Sprintf("%s: %s (value: %.2f, threshold: %.2f)", rule.Name, rule.Description, value, rule.Threshold)

		}

	} else if exists {

		// Resolve alert.

		existingAlert.Status = "resolved"

		existingAlert.EndsAt = time.Now()

		existingAlert.UpdatedAt = time.Now()



		// Move to history.

		wm.alertManager.alertHistory[cluster.Metadata.ID] = append(

			wm.alertManager.alertHistory[cluster.Metadata.ID],

			existingAlert,

		)



		// Remove from active alerts.

		delete(wm.alertManager.activeAlerts, alertID)



		// Update metrics.

		wm.metrics.activeAlerts.WithLabelValues(cluster.Metadata.ID, rule.Severity).Dec()



		// Send resolution notification.

		wm.sendAlertResolution(existingAlert)

	}



	wm.alertManager.mu.Unlock()

}



func (wm *WorkloadMonitor) getMetricValue(cluster *ClusterEntry, query string) float64 {

	// This would typically query Prometheus or another metrics system.

	// For now, return values from our collected metrics.



	switch query {

	case "cpu_utilization":

		wm.performanceMonitor.mu.RLock()

		metrics, exists := wm.performanceMonitor.metrics[cluster.Metadata.ID]

		wm.performanceMonitor.mu.RUnlock()

		if exists {

			return metrics.CPUMetrics.UtilizationPercent

		}

		return cluster.Metadata.Resources.Utilization * 100

	case "memory_utilization":

		wm.performanceMonitor.mu.RLock()

		metrics, exists := wm.performanceMonitor.metrics[cluster.Metadata.ID]

		wm.performanceMonitor.mu.RUnlock()

		if exists {

			return metrics.MemoryMetrics.UtilizationPercent

		}

		return 0.0

	case "node_readiness":

		wm.healthChecker.mu.RLock()

		history, exists := wm.healthChecker.healthHistory[cluster.Metadata.ID]

		wm.healthChecker.mu.RUnlock()

		if exists && len(history) > 0 {

			latest := history[len(history)-1]

			if readyNodes, ok := latest.Metrics["ready_nodes"]; ok {

				if totalNodes, ok := latest.Metrics["total_nodes"]; ok && totalNodes > 0 {

					return readyNodes / totalNodes

				}

			}

		}

		return 1.0

	case "pod_restart_rate":

		wm.performanceMonitor.mu.RLock()

		metrics, exists := wm.performanceMonitor.metrics[cluster.Metadata.ID]

		wm.performanceMonitor.mu.RUnlock()

		if exists {

			// Calculate restart rate per hour (simplified).

			return float64(metrics.PodMetrics.RestartCount) / 24.0

		}

		return 0.0

	default:

		return 0.0

	}

}



func (wm *WorkloadMonitor) sendAlertNotifications(alert *Alert) {

	// Implementation would send notifications via configured channels.

	wm.logger.Info("Alert triggered",

		"alert", alert.ID,

		"cluster", alert.ClusterID,

		"severity", alert.Severity,

		"message", alert.Message)

}



func (wm *WorkloadMonitor) sendAlertResolution(alert *Alert) {

	// Implementation would send resolution notifications.

	wm.logger.Info("Alert resolved",

		"alert", alert.ID,

		"cluster", alert.ClusterID,

		"duration", alert.EndsAt.Sub(alert.StartsAt))

}



func (wm *WorkloadMonitor) processAlertEscalations(ctx context.Context) {

	// Implementation would process alert escalations based on configured rules.

	// This is a placeholder for the escalation logic.

}



func (wm *WorkloadMonitor) runPredictiveAnalysis(ctx context.Context) {

	ticker := time.NewTicker(15 * time.Minute)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return

		case <-wm.stopCh:

			return

		case <-ticker.C:

			wm.performPredictiveAnalysis(ctx)

		}

	}

}



func (wm *WorkloadMonitor) performPredictiveAnalysis(ctx context.Context) {

	clusters := wm.registry.ListClusters()



	for _, cluster := range clusters {

		wm.analyzePredictiveMetrics(ctx, cluster)

		wm.detectAnomalies(ctx, cluster)

	}

}



func (wm *WorkloadMonitor) analyzePredictiveMetrics(ctx context.Context, cluster *ClusterEntry) {

	// This would implement machine learning models for prediction.

	// Placeholder implementation.



	prediction := &Prediction{

		ClusterID:  cluster.Metadata.ID,

		MetricName: "cpu_utilization",

		Value:      cluster.Metadata.Resources.Utilization * 100 * 1.1, // Simple prediction

		Confidence: 0.75,

		Timestamp:  time.Now(),

		Horizon:    1 * time.Hour,

	}



	wm.predictiveAnalyzer.mu.Lock()

	key := fmt.Sprintf("%s-%s", cluster.Metadata.ID, prediction.MetricName)

	wm.predictiveAnalyzer.predictions[key] = prediction

	wm.predictiveAnalyzer.mu.Unlock()

}



func (wm *WorkloadMonitor) detectAnomalies(ctx context.Context, cluster *ClusterEntry) {

	// This would implement anomaly detection algorithms.

	// Placeholder implementation.



	wm.performanceMonitor.mu.RLock()

	metrics, exists := wm.performanceMonitor.metrics[cluster.Metadata.ID]

	wm.performanceMonitor.mu.RUnlock()



	if !exists {

		return

	}



	// Simple anomaly detection based on thresholds.

	if metrics.CPUMetrics.UtilizationPercent > 95 {

		anomaly := &Anomaly{

			ClusterID:     cluster.Metadata.ID,

			MetricName:    "cpu_utilization",

			Value:         metrics.CPUMetrics.UtilizationPercent,

			ExpectedValue: 70.0,

			Deviation:     metrics.CPUMetrics.UtilizationPercent - 70.0,

			Severity:      "high",

			DetectedAt:    time.Now(),

		}



		wm.predictiveAnalyzer.mu.Lock()

		wm.predictiveAnalyzer.anomalies[cluster.Metadata.ID] = append(

			wm.predictiveAnalyzer.anomalies[cluster.Metadata.ID],

			anomaly,

		)

		wm.predictiveAnalyzer.mu.Unlock()



		wm.metrics.anomalyDetections.WithLabelValues(cluster.Metadata.ID, "cpu_utilization", "high").Inc()

	}

}



func (wm *WorkloadMonitor) runAutoRemediation(ctx context.Context) {

	ticker := time.NewTicker(1 * time.Minute)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return

		case <-wm.stopCh:

			return

		case <-ticker.C:

			wm.processAutoRemediation(ctx)

		}

	}

}



func (wm *WorkloadMonitor) processAutoRemediation(ctx context.Context) {

	clusters := wm.registry.ListClusters()



	for _, cluster := range clusters {

		wm.evaluateRemediationRules(ctx, cluster)

	}

}



func (wm *WorkloadMonitor) evaluateRemediationRules(ctx context.Context, cluster *ClusterEntry) {

	wm.autoRemediator.mu.RLock()

	rules := make([]*RemediationRule, 0, len(wm.autoRemediator.remediationRules))

	for _, rule := range wm.autoRemediator.remediationRules {

		if rule.Enabled {

			rules = append(rules, rule)

		}

	}

	wm.autoRemediator.mu.RUnlock()



	for _, rule := range rules {

		wm.evaluateRemediationRule(ctx, cluster, rule)

	}

}



func (wm *WorkloadMonitor) evaluateRemediationRule(ctx context.Context, cluster *ClusterEntry, rule *RemediationRule) {

	// Check if rule should be triggered.

	triggered := wm.shouldTriggerRemediation(ctx, cluster, rule)

	if !triggered {

		return

	}



	// Check safety conditions.

	if !wm.checkSafetyConditions(ctx, cluster, rule) {

		wm.logger.Info("Remediation rule skipped due to safety checks",

			"cluster", cluster.Metadata.ID,

			"rule", rule.ID)

		return

	}



	// Execute remediation actions.

	for _, actionType := range rule.Actions {

		action := &RemediationAction{

			ID:         fmt.Sprintf("%s-%s-%d", cluster.Metadata.ID, rule.ID, time.Now().Unix()),

			RuleID:     rule.ID,

			ClusterID:  cluster.Metadata.ID,

			Action:     actionType,

			Status:     "executing",

			ExecutedAt: time.Now(),

		}



		result := wm.executeRemediationAction(ctx, cluster, action)

		action.Status = result

		action.CompletedAt = time.Now()



		// Store action history.

		wm.autoRemediator.mu.Lock()

		wm.autoRemediator.remediationHistory[cluster.Metadata.ID] = append(

			wm.autoRemediator.remediationHistory[cluster.Metadata.ID],

			action,

		)

		wm.autoRemediator.mu.Unlock()



		// Update metrics.

		wm.metrics.remediationActions.WithLabelValues(cluster.Metadata.ID, actionType, result).Inc()



		wm.logger.Info("Remediation action executed",

			"cluster", cluster.Metadata.ID,

			"action", actionType,

			"result", result)

	}

}



func (wm *WorkloadMonitor) shouldTriggerRemediation(ctx context.Context, cluster *ClusterEntry, rule *RemediationRule) bool {

	// This would implement rule evaluation logic.

	// Placeholder implementation based on rule trigger.



	switch rule.Trigger {

	case "pod_failed":

		wm.performanceMonitor.mu.RLock()

		metrics, exists := wm.performanceMonitor.metrics[cluster.Metadata.ID]

		wm.performanceMonitor.mu.RUnlock()

		return exists && metrics.PodMetrics.FailedPods > 0

	case "high_cpu_usage":

		wm.performanceMonitor.mu.RLock()

		metrics, exists := wm.performanceMonitor.metrics[cluster.Metadata.ID]

		wm.performanceMonitor.mu.RUnlock()

		return exists && metrics.CPUMetrics.UtilizationPercent > 80

	default:

		return false

	}

}



func (wm *WorkloadMonitor) checkSafetyConditions(ctx context.Context, cluster *ClusterEntry, rule *RemediationRule) bool {

	// Implement safety checks.

	for _, check := range rule.SafetyChecks {

		switch check {

		case "check-pod-age":

			// Check if pods are old enough for remediation.

			continue

		case "check-restart-count":

			// Check if restart count is within limits.

			continue

		case "check-max-replicas":

			// Check if scaling won't exceed limits.

			continue

		case "check-resource-limits":

			// Check if there are enough resources.

			continue

		default:

			wm.logger.Info("Unknown safety check", "check", check)

			return false

		}

	}

	return true

}



func (wm *WorkloadMonitor) executeRemediationAction(ctx context.Context, cluster *ClusterEntry, action *RemediationAction) string {

	// Execute the remediation action.

	switch action.Action {

	case "delete-pod":

		// Implementation would delete failed pods.

		return "success"

	case "scale-deployment":

		// Implementation would scale deployments.

		return "success"

	case "restart-service":

		// Implementation would restart services.

		return "success"

	default:

		return "unknown-action"

	}

}



// AlertManager methods.



// CreateAlert creates a new alert.

func (am *AlertManager) CreateAlert(alert *Alert) error {

	am.mu.Lock()

	defer am.mu.Unlock()



	am.activeAlerts[alert.ID] = alert

	am.logger.Info("Alert created", "id", alert.ID, "severity", alert.Severity)



	return nil

}



// DashboardIntegrator methods.



// CreateDashboard creates a new dashboard.

func (di *DashboardIntegrator) CreateDashboard(dashboard *Dashboard) error {

	di.mu.Lock()

	defer di.mu.Unlock()



	di.dashboards[dashboard.ID] = dashboard

	di.logger.Info("Dashboard created", "id", dashboard.ID, "title", dashboard.Title)



	return nil

}



// AutoRemediator methods.



// AddRule adds a new remediation rule.

func (ar *AutoRemediator) AddRule(rule *RemediationRule) error {

	ar.mu.Lock()

	defer ar.mu.Unlock()



	ar.remediationRules[rule.ID] = rule

	ar.logger.Info("Remediation rule added", "id", rule.ID, "name", rule.Name)



	return nil

}

