package o1

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	oranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran"
)

const (
	// O1ControllerName holds o1controllername value.
	O1ControllerName = "o1-controller"
	// FinalizerName holds finalizername value.
	FinalizerName = "o1.oran.nephio.org/finalizer"
)

// O1InterfaceController reconciles O1Interface objects.
type O1InterfaceController struct {
	client.Client
	Log                  logr.Logger
	Scheme               *runtime.Scheme
	o1AdapterManager     *O1AdapterManager
	streamingService     *StreamingService
	netconfServerManager *NetconfServerManager
	performanceManager   *CompletePerformanceManager
	faultManager         *EnhancedFaultManager
	configManager        *AdvancedConfigurationManager
	securityManager      *ComprehensiveSecurityManager
	accountingManager    *ComprehensiveAccountingManager
	smoIntegration       *SMOIntegrationLayer
	metrics              *O1ControllerMetrics
	config               *O1ControllerConfig
}

// O1ControllerConfig holds configuration for the O1 controller.
type O1ControllerConfig struct {
	ReconcileInterval       time.Duration  `yaml:"reconcile_interval"`
	MaxConcurrentReconciles int            `yaml:"max_concurrent_reconciles"`
	EnableMetrics           bool           `yaml:"enable_metrics"`
	EnableWebhooks          bool           `yaml:"enable_webhooks"`
	DefaultO1Config         *oran.O1Config `yaml:"default_o1_config"`
	HealthCheckInterval     time.Duration  `yaml:"health_check_interval"`
	StatusUpdateInterval    time.Duration  `yaml:"status_update_interval"`
}

// O1InterfaceConfig represents O1 interface configuration.
type O1InterfaceConfig struct {
	NetconfPort           int               `yaml:"netconf_port"`
	StreamingPort         int               `yaml:"streaming_port"`
	EnableTLS             bool              `yaml:"enable_tls"`
	TLSCertSecret         string            `yaml:"tls_cert_secret"`
	AuthenticationMethod  string            `yaml:"authentication_method"`
	MaxConnections        int               `yaml:"max_connections"`
	SessionTimeout        time.Duration     `yaml:"session_timeout"`
	PerformanceCollection PerformanceConfig `yaml:"performance_collection"`
	FaultManagement       FaultConfig       `yaml:"fault_management"`
	SecurityPolicies      SecurityConfig    `yaml:"security_policies"`
}

// PerformanceConfig holds performance management configuration.
type PerformanceConfig struct {
	Enabled            bool          `yaml:"enabled"`
	CollectionInterval time.Duration `yaml:"collection_interval"`
	RetentionPeriod    time.Duration `yaml:"retention_period"`
	Metrics            []string      `yaml:"metrics"`
}

// FaultConfig holds fault management configuration.
type FaultConfig struct {
	Enabled             bool     `yaml:"enabled"`
	AlarmForwarding     bool     `yaml:"alarm_forwarding"`
	CorrelationEnabled  bool     `yaml:"correlation_enabled"`
	NotificationTargets []string `yaml:"notification_targets"`
	SeverityFilters     []string `yaml:"severity_filters"`
}

// SecurityConfig holds security configuration.
type SecurityConfig struct {
	EnableAuthentication  bool     `yaml:"enable_authentication"`
	EnableAuthorization   bool     `yaml:"enable_authorization"`
	RequiredRoles         []string `yaml:"required_roles"`
	CertificateValidation bool     `yaml:"certificate_validation"`
	AuditLogging          bool     `yaml:"audit_logging"`
}

// O1ControllerMetrics holds Prometheus metrics for the controller.
type O1ControllerMetrics struct {
	ReconciliationsTotal   prometheus.CounterVec
	ReconciliationDuration prometheus.HistogramVec
	ReconciliationErrors   prometheus.CounterVec
	ActiveO1Interfaces     prometheus.Gauge
	O1ConnectionsActive    prometheus.GaugeVec
	ConfigurationChanges   prometheus.CounterVec
	AlarmsSent             prometheus.CounterVec
	PerformanceDataPoints  prometheus.CounterVec
}

// O1AdapterManager manages O1 adapter instances.
type O1AdapterManager struct {
	adapters map[string]*O1AdaptorInstance
	mutex    sync.RWMutex
	logger   logr.Logger
}

// O1AdaptorInstance represents an instance of O1 adaptor for a specific network function.
type O1AdaptorInstance struct {
	Name             string
	Namespace        string
	NetworkFunction  string
	O1Adaptor        *O1Adaptor
	NetconfServer    *NetconfServer
	StreamingHandler *StreamingService
	Status           O1InstanceStatus
	CreatedAt        time.Time
	LastUpdate       time.Time
}

// O1InstanceStatus represents the status of an O1 adapter instance.
type O1InstanceStatus struct {
	Phase                O1InstancePhase `json:"phase"`
	Message              string          `json:"message,omitempty"`
	LastTransitionTime   metav1.Time     `json:"lastTransitionTime,omitempty"`
	ActiveConnections    int32           `json:"activeConnections"`
	ProcessedAlarms      int64           `json:"processedAlarms"`
	ProcessedPerfData    int64           `json:"processedPerfData"`
	ConfigurationVersion string          `json:"configurationVersion"`
}

// O1InstancePhase represents the phase of an O1 instance.
type O1InstancePhase string

const (
	// O1InstancePhasePending holds o1instancephasepending value.
	O1InstancePhasePending O1InstancePhase = "Pending"
	// O1InstancePhaseInitializing holds o1instancephaseinitializing value.
	O1InstancePhaseInitializing O1InstancePhase = "Initializing"
	// O1InstancePhaseReady holds o1instancephaseready value.
	O1InstancePhaseReady O1InstancePhase = "Ready"
	// O1InstancePhaseFailed holds o1instancephasefailed value.
	O1InstancePhaseFailed O1InstancePhase = "Failed"
	// O1InstancePhaseTerminating holds o1instancephaseterminating value.
	O1InstancePhaseTerminating O1InstancePhase = "Terminating"
)

// NetconfServerManager manages NETCONF server instances.
type NetconfServerManager struct {
	servers map[string]*NetconfServer
	mutex   sync.RWMutex
	logger  logr.Logger
}

// NewO1InterfaceController creates a new O1 interface controller.
func NewO1InterfaceController(
	client client.Client,
	logger logr.Logger,
	scheme *runtime.Scheme,
	config *O1ControllerConfig,
) *O1InterfaceController {
	// Initialize metrics.
	metrics := &O1ControllerMetrics{
		ReconciliationsTotal: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "o1_controller_reconciliations_total",
				Help: "Total number of reconciliations performed by O1 controller",
			},
			[]string{"namespace", "name", "result"},
		),
		ReconciliationDuration: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "o1_controller_reconciliation_duration_seconds",
				Help: "Duration of reconciliation operations",
			},
			[]string{"namespace", "name"},
		),
		ReconciliationErrors: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "o1_controller_reconciliation_errors_total",
				Help: "Total number of reconciliation errors",
			},
			[]string{"namespace", "name", "error_type"},
		),
		ActiveO1Interfaces: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "o1_controller_active_interfaces",
				Help: "Number of active O1 interfaces",
			},
		),
		O1ConnectionsActive: *prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "o1_interface_connections_active",
				Help: "Number of active connections per O1 interface",
			},
			[]string{"namespace", "name", "protocol"},
		),
		ConfigurationChanges: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "o1_interface_configuration_changes_total",
				Help: "Total number of configuration changes",
			},
			[]string{"namespace", "name", "change_type"},
		),
		AlarmsSent: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "o1_interface_alarms_sent_total",
				Help: "Total number of alarms sent",
			},
			[]string{"namespace", "name", "severity"},
		),
		PerformanceDataPoints: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "o1_interface_performance_data_points_total",
				Help: "Total number of performance data points collected",
			},
			[]string{"namespace", "name", "metric_type"},
		),
	}

	// Register metrics.
	if config.EnableMetrics {
		prometheus.MustRegister(
			metrics.ReconciliationsTotal,
			metrics.ReconciliationDuration,
			metrics.ReconciliationErrors,
			metrics.ActiveO1Interfaces,
			metrics.O1ConnectionsActive,
			metrics.ConfigurationChanges,
			metrics.AlarmsSent,
			metrics.PerformanceDataPoints,
		)
	}

	// Initialize managers.
	o1AdapterManager := &O1AdapterManager{
		adapters: make(map[string]*O1AdaptorInstance),
		logger:   logger,
	}

	netconfServerManager := &NetconfServerManager{
		servers: make(map[string]*NetconfServer),
		logger:  logger,
	}

	// Initialize streaming service with proper zap logger.
	streamingConfig := &StreamingConfig{
		MaxConnections:          1000,
		ConnectionTimeout:       5 * time.Minute,
		HeartbeatInterval:       30 * time.Second,
		MaxSubscriptionsPerConn: 100,
		BufferSize:              1024,
		CompressionEnabled:      true,
		EnableAuth:              true,
		RateLimitPerSecond:      100,
	}
	// Create a simple zap logger - for now we'll use a basic logger.
	zapLogger, _ := zap.NewDevelopment()
	if zapLogger == nil {
		zapLogger = zap.NewNop() // Fallback to no-op logger
	}
	streamingService := NewStreamingService(streamingConfig, zapLogger)

	return &O1InterfaceController{
		Client:               client,
		Log:                  logger,
		Scheme:               scheme,
		o1AdapterManager:     o1AdapterManager,
		streamingService:     streamingService,
		netconfServerManager: netconfServerManager,
		metrics:              metrics,
		config:               config,
	}
}

// SetupWithManager sets up the controller with the manager.
func (r *O1InterfaceController) SetupWithManager(mgr ctrl.Manager) error {
	controllerOptions := controller.Options{
		MaxConcurrentReconciles: r.config.MaxConcurrentReconciles,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&oranv1.O1Interface{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Secret{}).
		WithOptions(controllerOptions).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

// Reconcile reconciles O1Interface resources.
func (r *O1InterfaceController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("o1interface", req.NamespacedName)
	startTime := time.Now()

	// Get O1Interface instance.
	o1Interface := &oranv1.O1Interface{}
	err := r.Get(ctx, req.NamespacedName, o1Interface)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("O1Interface resource not found, likely deleted")
			r.handleDeletion(ctx, req.NamespacedName)
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get O1Interface")
		r.recordReconciliationMetrics(req.NamespacedName, "error", startTime)
		return ctrl.Result{}, err
	}

	// Handle deletion.
	if !o1Interface.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDeletion(ctx, o1Interface)
	}

	// Add finalizer if not present.
	if !controllerutil.ContainsFinalizer(o1Interface, FinalizerName) {
		controllerutil.AddFinalizer(o1Interface, FinalizerName)
		err := r.Update(ctx, o1Interface)
		if err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile the O1Interface.
	result, err := r.reconcileO1Interface(ctx, o1Interface)
	if err != nil {
		r.recordReconciliationMetrics(req.NamespacedName, "error", startTime)
		r.metrics.ReconciliationErrors.WithLabelValues(
			req.Namespace,
			req.Name,
			"reconcile_error",
		).Inc()
	} else {
		r.recordReconciliationMetrics(req.NamespacedName, "success", startTime)
	}

	return result, err
}

// reconcileO1Interface reconciles the O1Interface resource.
func (r *O1InterfaceController) reconcileO1Interface(ctx context.Context, o1Interface *oranv1.O1Interface) (ctrl.Result, error) {
	log := r.Log.WithValues("o1interface", o1Interface.Name, "namespace", o1Interface.Namespace)

	// Validate configuration.
	if err := r.validateO1InterfaceSpec(o1Interface); err != nil {
		log.Error(err, "Invalid O1Interface specification")
		r.updateStatus(ctx, o1Interface, O1InstancePhaseFailed, err.Error())
		return ctrl.Result{}, err
	}

	// Get or create O1 adapter instance.
	adapterInstance, err := r.getOrCreateAdapterInstance(ctx, o1Interface)
	if err != nil {
		log.Error(err, "Failed to get or create adapter instance")
		r.updateStatus(ctx, o1Interface, O1InstancePhaseFailed, err.Error())
		return ctrl.Result{}, err
	}

	// Update status to initializing.
	r.updateStatus(ctx, o1Interface, O1InstancePhaseInitializing, "Initializing O1 interface components")

	// Create or update NETCONF server.
	if err := r.reconcileNetconfServer(ctx, o1Interface, adapterInstance); err != nil {
		log.Error(err, "Failed to reconcile NETCONF server")
		r.updateStatus(ctx, o1Interface, O1InstancePhaseFailed, fmt.Sprintf("NETCONF server error: %v", err))
		return ctrl.Result{}, err
	}

	// Create or update streaming service.
	if err := r.reconcileStreamingService(ctx, o1Interface, adapterInstance); err != nil {
		log.Error(err, "Failed to reconcile streaming service")
		r.updateStatus(ctx, o1Interface, O1InstancePhaseFailed, fmt.Sprintf("Streaming service error: %v", err))
		return ctrl.Result{}, err
	}

	// Initialize FCAPS managers.
	if err := r.initializeFCAPSManagers(ctx, o1Interface, adapterInstance); err != nil {
		log.Error(err, "Failed to initialize FCAPS managers")
		r.updateStatus(ctx, o1Interface, O1InstancePhaseFailed, fmt.Sprintf("FCAPS initialization error: %v", err))
		return ctrl.Result{}, err
	}

	// Create Kubernetes resources (Services, ConfigMaps, etc.).
	if err := r.reconcileKubernetesResources(ctx, o1Interface); err != nil {
		log.Error(err, "Failed to reconcile Kubernetes resources")
		r.updateStatus(ctx, o1Interface, O1InstancePhaseFailed, fmt.Sprintf("Kubernetes resources error: %v", err))
		return ctrl.Result{}, err
	}

	// Start health monitoring.
	go r.startHealthMonitoring(ctx, adapterInstance)

	// Update status to ready.
	r.updateStatus(ctx, o1Interface, O1InstancePhaseReady, "O1 interface is ready and operational")

	// Update metrics.
	r.metrics.ActiveO1Interfaces.Inc()
	r.metrics.O1ConnectionsActive.WithLabelValues(
		o1Interface.Namespace,
		o1Interface.Name,
		"netconf",
	).Set(float64(adapterInstance.Status.ActiveConnections))

	log.Info("Successfully reconciled O1Interface")
	return ctrl.Result{RequeueAfter: r.config.ReconcileInterval}, nil
}

// reconcileDeletion handles deletion of O1Interface resources.
func (r *O1InterfaceController) reconcileDeletion(ctx context.Context, o1Interface *oranv1.O1Interface) (ctrl.Result, error) {
	log := r.Log.WithValues("o1interface", o1Interface.Name, "namespace", o1Interface.Namespace)

	log.Info("Handling O1Interface deletion")

	// Update status.
	r.updateStatus(ctx, o1Interface, O1InstancePhaseTerminating, "Terminating O1 interface")

	// Clean up adapter instance.
	instanceKey := fmt.Sprintf("%s/%s", o1Interface.Namespace, o1Interface.Name)
	if err := r.cleanupAdapterInstance(ctx, instanceKey); err != nil {
		log.Error(err, "Failed to cleanup adapter instance")
		return ctrl.Result{}, err
	}

	// Clean up NETCONF server.
	if err := r.cleanupNetconfServer(ctx, instanceKey); err != nil {
		log.Error(err, "Failed to cleanup NETCONF server")
		return ctrl.Result{}, err
	}

	// Update metrics.
	r.metrics.ActiveO1Interfaces.Dec()
	r.metrics.O1ConnectionsActive.DeleteLabelValues(
		o1Interface.Namespace,
		o1Interface.Name,
		"netconf",
	)

	// Remove finalizer.
	controllerutil.RemoveFinalizer(o1Interface, FinalizerName)
	err := r.Update(ctx, o1Interface)
	if err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.Info("Successfully handled O1Interface deletion")
	return ctrl.Result{}, nil
}

// validateO1InterfaceSpec validates the O1Interface specification.
func (r *O1InterfaceController) validateO1InterfaceSpec(o1Interface *oranv1.O1Interface) error {
	// Validate host is provided.
	if o1Interface.Spec.Host == "" {
		return fmt.Errorf("host is required")
	}

	// Validate port range.
	if o1Interface.Spec.Port < 1 || o1Interface.Spec.Port > 65535 {
		return fmt.Errorf("invalid port: %d", o1Interface.Spec.Port)
	}

	// Validate protocol.
	if o1Interface.Spec.Protocol != "" && o1Interface.Spec.Protocol != "ssh" && o1Interface.Spec.Protocol != "tls" {
		return fmt.Errorf("invalid protocol: %s, must be 'ssh' or 'tls'", o1Interface.Spec.Protocol)
	}

	// Validate TLS configuration if needed.
	if o1Interface.Spec.Protocol == "tls" && o1Interface.Spec.SecurityConfig != nil {
		if o1Interface.Spec.SecurityConfig.TLSConfig != nil && !o1Interface.Spec.SecurityConfig.TLSConfig.Enabled {
			return fmt.Errorf("TLS protocol specified but TLS is not enabled in security config")
		}
	}

	return nil
}

// getOrCreateAdapterInstance gets or creates an O1 adapter instance.
func (r *O1InterfaceController) getOrCreateAdapterInstance(ctx context.Context, o1Interface *oranv1.O1Interface) (*O1AdaptorInstance, error) {
	instanceKey := fmt.Sprintf("%s/%s", o1Interface.Namespace, o1Interface.Name)

	r.o1AdapterManager.mutex.Lock()
	defer r.o1AdapterManager.mutex.Unlock()

	// Check if instance already exists.
	if instance, exists := r.o1AdapterManager.adapters[instanceKey]; exists {
		// Update configuration if needed - use a simple config comparison.
		currentConfigVersion := fmt.Sprintf("%s:%d:%s", o1Interface.Spec.Host, o1Interface.Spec.Port, o1Interface.Spec.Protocol)
		if instance.Status.ConfigurationVersion != currentConfigVersion {
			instance.Status.ConfigurationVersion = currentConfigVersion
			instance.LastUpdate = time.Now()
		}
		return instance, nil
	}

	// Create new O1 adapter configuration using oran.O1Config.
	o1Config := &oran.O1Config{
		Endpoint:       fmt.Sprintf("%s:%d", o1Interface.Spec.Host, o1Interface.Spec.Port),
		Timeout:        30 * time.Second,
		RetryAttempts:  3,
		TLSConfig:      r.buildTLSConfig(o1Interface),
		Authentication: r.buildAuthConfig(o1Interface),
	}

	// Create O1 adapter.
	o1Adaptor := NewO1Adaptor(o1Config, r.Client)

	// Create adapter instance.
	instance := &O1AdaptorInstance{
		Name:            o1Interface.Name,
		Namespace:       o1Interface.Namespace,
		NetworkFunction: fmt.Sprintf("%s-%s", o1Interface.Namespace, o1Interface.Name),
		O1Adaptor:       o1Adaptor,
		CreatedAt:       time.Now(),
		LastUpdate:      time.Now(),
		Status: O1InstanceStatus{
			Phase:                O1InstancePhasePending,
			LastTransitionTime:   metav1.Now(),
			ConfigurationVersion: fmt.Sprintf("%s:%d:%s", o1Interface.Spec.Host, o1Interface.Spec.Port, o1Interface.Spec.Protocol),
		},
	}

	r.o1AdapterManager.adapters[instanceKey] = instance

	return instance, nil
}

// reconcileNetconfServer creates or updates the NETCONF server.
func (r *O1InterfaceController) reconcileNetconfServer(ctx context.Context, o1Interface *oranv1.O1Interface, instance *O1AdaptorInstance) error {
	serverKey := fmt.Sprintf("%s/%s", o1Interface.Namespace, o1Interface.Name)

	r.netconfServerManager.mutex.Lock()
	defer r.netconfServerManager.mutex.Unlock()

	// Check if server already exists.
	if server, exists := r.netconfServerManager.servers[serverKey]; exists {
		// Update configuration if needed.
		return r.updateNetconfServerConfig(server, o1Interface)
	}

	// Create new NETCONF server.
	serverConfig := r.buildNetconfServerConfig(o1Interface)
	server := NewNetconfServer(serverConfig)

	// Start server.
	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("failed to start NETCONF server: %w", err)
	}

	r.netconfServerManager.servers[serverKey] = server
	instance.NetconfServer = server

	return nil
}

// reconcileStreamingService configures the streaming service for the O1 interface.
func (r *O1InterfaceController) reconcileStreamingService(ctx context.Context, o1Interface *oranv1.O1Interface, instance *O1AdaptorInstance) error {
	// Configure streaming service for this instance.
	instance.StreamingHandler = r.streamingService

	// Start streaming service if not already running.
	if r.streamingService != nil {
		return r.streamingService.Start(ctx)
	}

	return nil
}

// initializeFCAPSManagers initializes FCAPS management components.
func (r *O1InterfaceController) initializeFCAPSManagers(ctx context.Context, o1Interface *oranv1.O1Interface, instance *O1AdaptorInstance) error {
	// Initialize fault manager.
	if o1Interface.Spec.FCAPS.FaultManagement.Enabled {
		faultConfig := &FaultManagerConfig{
			MaxAlarms:           10000,
			MaxHistoryEntries:   1000,
			CorrelationWindow:   5 * time.Minute,
			NotificationTimeout: 30 * time.Second,
			EnableWebSocket:     true,
			PrometheusURL:       "http://prometheus:9090",
			AlertManagerURL:     "http://alertmanager:9093",
			EnableRootCause:     o1Interface.Spec.FCAPS.FaultManagement.RootCauseAnalysis,
		}

		faultManager := NewEnhancedFaultManager(faultConfig)
		r.faultManager = faultManager
	}

	// Initialize performance manager.
	if o1Interface.Spec.FCAPS.PerformanceManagement.Enabled {
		perfConfig := &PerformanceManagerConfig{
			PrometheusURL:           "http://prometheus:9090",
			GrafanaURL:              "http://grafana:3000",
			CollectionIntervals:     map[string]time.Duration{"default": time.Duration(o1Interface.Spec.FCAPS.PerformanceManagement.CollectionInterval) * time.Second},
			RetentionPeriods:        map[string]time.Duration{"default": 30 * 24 * time.Hour},
			DefaultGranularity:      time.Minute,
			MaxDataPoints:           10000,
			EnableRealTimeStreaming: true,
			EnableAnomalyDetection:  o1Interface.Spec.FCAPS.PerformanceManagement.AnomalyDetection,
			EnableReporting:         true,
			ReportingInterval:       time.Hour,
		}

		perfManager := NewCompletePerformanceManager(perfConfig)
		r.performanceManager = perfManager
	}

	// Initialize configuration manager.
	configMgrConfig := &ConfigManagerConfig{
		MaxVersions:          100,
		GitRepository:        "", // Will be configured externally
		GitBranch:            "main",
		EnableDriftDetection: o1Interface.Spec.FCAPS.ConfigurationManagement.DriftDetection,
		EnableBulkOps:        true,
	}

	// Create a proper YANG registry.
	yangRegistry := NewExtendedYANGModelRegistry()
	configManager := NewAdvancedConfigurationManager(configMgrConfig, yangRegistry)
	r.configManager = configManager

	// Initialize security manager.
	if o1Interface.Spec.FCAPS.SecurityManagement.Enabled {
		// For now, we'll skip the comprehensive security manager initialization.
		// as it requires complex configuration and dependencies.
		r.Log.Info("Security management enabled for O1Interface")
	}

	// Initialize accounting manager.
	if o1Interface.Spec.FCAPS.AccountingManagement.Enabled {
		// For now, we'll skip the comprehensive accounting manager initialization.
		// as it requires complex configuration and dependencies.
		r.Log.Info("Accounting management enabled for O1Interface")
	}

	return nil
}

// reconcileKubernetesResources creates or updates Kubernetes resources.
func (r *O1InterfaceController) reconcileKubernetesResources(ctx context.Context, o1Interface *oranv1.O1Interface) error {
	// Create Service for NETCONF.
	service := r.buildNetconfService(o1Interface)
	if err := r.createOrUpdateService(ctx, service); err != nil {
		return fmt.Errorf("failed to create NETCONF service: %w", err)
	}

	// Create Service for streaming.
	streamingService := r.buildStreamingService(o1Interface)
	if err := r.createOrUpdateService(ctx, streamingService); err != nil {
		return fmt.Errorf("failed to create streaming service: %w", err)
	}

	// Create ConfigMap for YANG models.
	configMap := r.buildYANGModelsConfigMap(o1Interface)
	if err := r.createOrUpdateConfigMap(ctx, configMap); err != nil {
		return fmt.Errorf("failed to create YANG models ConfigMap: %w", err)
	}

	return nil
}

// updateStatus updates the O1Interface status.
func (r *O1InterfaceController) updateStatus(ctx context.Context, o1Interface *oranv1.O1Interface, phase O1InstancePhase, message string) {
	o1Interface.Status.Phase = string(phase)
	o1Interface.Status.ErrorMessage = message
	now := metav1.Now()
	o1Interface.Status.LastSyncTime = &now

	// Update conditions.
	conditionType := "Ready"
	conditionStatus := metav1.ConditionTrue
	conditionReason := "Successful"
	if phase == O1InstancePhaseFailed {
		conditionStatus = metav1.ConditionFalse
		conditionReason = "Failed"
	} else if phase == O1InstancePhaseInitializing {
		conditionStatus = metav1.ConditionFalse
		conditionReason = "Initializing"
	}

	// Find existing condition or create new one.
	conditionFound := false
	for i := range o1Interface.Status.Conditions {
		if o1Interface.Status.Conditions[i].Type == conditionType {
			o1Interface.Status.Conditions[i].Status = conditionStatus
			o1Interface.Status.Conditions[i].Reason = conditionReason
			o1Interface.Status.Conditions[i].Message = message
			o1Interface.Status.Conditions[i].LastTransitionTime = now
			conditionFound = true
			break
		}
	}

	if !conditionFound {
		o1Interface.Status.Conditions = append(o1Interface.Status.Conditions, metav1.Condition{
			Type:               conditionType,
			Status:             conditionStatus,
			Reason:             conditionReason,
			Message:            message,
			LastTransitionTime: now,
		})
	}

	if err := r.Status().Update(ctx, o1Interface); err != nil {
		r.Log.Error(err, "Failed to update O1Interface status")
	}
}

// startHealthMonitoring starts health monitoring for the adapter instance.
func (r *O1InterfaceController) startHealthMonitoring(ctx context.Context, instance *O1AdaptorInstance) {
	ticker := time.NewTicker(r.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.performHealthCheck(instance)
		}
	}
}

// performHealthCheck performs health check on the adapter instance.
func (r *O1InterfaceController) performHealthCheck(instance *O1AdaptorInstance) {
	// Basic health check - ensure components are not nil and status is not failed.
	if instance.NetconfServer == nil {
		instance.Status.Phase = O1InstancePhaseFailed
		instance.Status.Message = "NETCONF server is not initialized"
		return
	}

	if instance.O1Adaptor == nil {
		instance.Status.Phase = O1InstancePhaseFailed
		instance.Status.Message = "O1 adaptor is not initialized"
		return
	}

	// Update status if healthy.
	if instance.Status.Phase == O1InstancePhaseFailed {
		instance.Status.Phase = O1InstancePhaseReady
		instance.Status.Message = "O1 interface is healthy"
	}
}

// Helper methods for building configurations and resources.

func (r *O1InterfaceController) buildNetconfServerConfig(o1Interface *oranv1.O1Interface) *NetconfServerConfig {
	// Default values.
	port := o1Interface.Spec.Port
	if port == 0 {
		port = 830 // Default NETCONF port
	}

	maxConnections := 100 // Default value
	if o1Interface.Spec.StreamingConfig != nil {
		maxConnections = o1Interface.Spec.StreamingConfig.MaxConnections
	}

	return &NetconfServerConfig{
		Host:                "0.0.0.0",
		Port:                port,
		TLSPort:             port + 1000,
		SSHPort:             port + 2000,
		TLSConfig:           nil, // Will be configured later if needed
		SSHHostKeyFile:      "/etc/ssh/host_key",
		MaxSessions:         maxConnections,
		SessionTimeout:      5 * time.Minute,
		EnableNotifications: true,
		EnableCandidate:     true,
		EnableStartup:       true,
		EnableXPath:         true,
		EnableValidation:    true,
	}
}

func (r *O1InterfaceController) buildPerformanceConfig(o1Interface *oranv1.O1Interface) interface{} {
	if o1Interface.Spec.FCAPS.PerformanceManagement.Enabled {
		return PerformanceConfig{
			Enabled:            true,
			CollectionInterval: time.Duration(o1Interface.Spec.FCAPS.PerformanceManagement.CollectionInterval) * time.Second,
			RetentionPeriod:    30 * 24 * time.Hour, // Default 30 days
			Metrics:            []string{"cpu", "memory", "network", "storage"},
		}
	}
	return PerformanceConfig{Enabled: false}
}

func (r *O1InterfaceController) buildFaultConfig(o1Interface *oranv1.O1Interface) interface{} {
	faultMgmt := o1Interface.Spec.FCAPS.FaultManagement
	return FaultConfig{
		Enabled:             faultMgmt.Enabled,
		AlarmForwarding:     true, // Default enabled
		CorrelationEnabled:  faultMgmt.CorrelationEnabled,
		NotificationTargets: []string{}, // Will be populated from external config
		SeverityFilters:     faultMgmt.SeverityFilter,
	}
}

func (r *O1InterfaceController) buildSecurityConfig(o1Interface *oranv1.O1Interface) interface{} {
	securityMgmt := o1Interface.Spec.FCAPS.SecurityManagement
	return SecurityConfig{
		EnableAuthentication:  securityMgmt.Enabled,
		EnableAuthorization:   securityMgmt.ComplianceMonitoring,
		RequiredRoles:         []string{"admin", "operator"}, // Default roles
		CertificateValidation: securityMgmt.CertificateManagement,
		AuditLogging:          securityMgmt.Enabled,
	}
}

func (r *O1InterfaceController) buildStreamingConfig(o1Interface *oranv1.O1Interface) interface{} {
	streamingConfig := o1Interface.Spec.StreamingConfig
	if streamingConfig == nil {
		// Default streaming configuration.
		return &StreamingConfig{
			MaxConnections:          100,
			ConnectionTimeout:       5 * time.Minute,
			HeartbeatInterval:       30 * time.Second,
			MaxSubscriptionsPerConn: 100,
			BufferSize:              1024,
			CompressionEnabled:      true,
			EnableAuth:              true,
			RateLimitPerSecond:      100,
		}
	}

	return &StreamingConfig{
		MaxConnections:          streamingConfig.MaxConnections,
		ConnectionTimeout:       5 * time.Minute, // Use default timeout
		HeartbeatInterval:       30 * time.Second,
		MaxSubscriptionsPerConn: 100,
		BufferSize:              1024,
		CompressionEnabled:      true,
		EnableAuth:              true,
		RateLimitPerSecond:      100,
	}
}

func (r *O1InterfaceController) buildNetconfService(o1Interface *oranv1.O1Interface) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-netconf", o1Interface.Name),
			Namespace: o1Interface.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "o1-interface",
				"app.kubernetes.io/instance":   o1Interface.Name,
				"app.kubernetes.io/component":  "netconf",
				"app.kubernetes.io/managed-by": O1ControllerName,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:     "netconf",
					Port:     int32(o1Interface.Spec.Port),
					Protocol: corev1.ProtocolTCP,
				},
				{
					Name:     "netconf-ssh",
					Port:     int32(o1Interface.Spec.Port + 1000),
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name":     "o1-interface",
				"app.kubernetes.io/instance": o1Interface.Name,
			},
		},
	}
}

func (r *O1InterfaceController) buildStreamingService(o1Interface *oranv1.O1Interface) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-streaming", o1Interface.Name),
			Namespace: o1Interface.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "o1-interface",
				"app.kubernetes.io/instance":   o1Interface.Name,
				"app.kubernetes.io/component":  "streaming",
				"app.kubernetes.io/managed-by": O1ControllerName,
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name: "streaming",
					Port: int32(func() int {
						if o1Interface.Spec.StreamingConfig != nil {
							return o1Interface.Spec.StreamingConfig.WebSocketPort
						}
						return 8080 // Default streaming port
					}()),
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name":     "o1-interface",
				"app.kubernetes.io/instance": o1Interface.Name,
			},
		},
	}
}

func (r *O1InterfaceController) buildYANGModelsConfigMap(o1Interface *oranv1.O1Interface) *corev1.ConfigMap {
	// Build YANG models data.
	yangModelsData := make(map[string]string)
	// Add standard O-RAN YANG models.
	yangModelsData["o-ran-hardware.yang"] = getORANHardwareYANG()
	yangModelsData["o-ran-software-management.yang"] = getORANSoftwareYANG()
	yangModelsData["o-ran-performance-management.yang"] = getORANPerformanceYANG()
	yangModelsData["o-ran-fault-management.yang"] = getORANFaultYANG()

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-yang-models", o1Interface.Name),
			Namespace: o1Interface.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "o1-interface",
				"app.kubernetes.io/instance":   o1Interface.Name,
				"app.kubernetes.io/component":  "yang-models",
				"app.kubernetes.io/managed-by": O1ControllerName,
			},
		},
		Data: yangModelsData,
	}
}

// Utility methods for resource management.
func (r *O1InterfaceController) createOrUpdateService(ctx context.Context, service *corev1.Service) error {
	existing := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, service)
	} else if err != nil {
		return err
	}

	// Update existing service.
	service.ResourceVersion = existing.ResourceVersion
	return r.Update(ctx, service)
}

func (r *O1InterfaceController) createOrUpdateConfigMap(ctx context.Context, configMap *corev1.ConfigMap) error {
	existing := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, existing)
	if errors.IsNotFound(err) {
		return r.Create(ctx, configMap)
	} else if err != nil {
		return err
	}

	// Update existing configMap.
	configMap.ResourceVersion = existing.ResourceVersion
	return r.Update(ctx, configMap)
}

// Cleanup methods.
func (r *O1InterfaceController) handleDeletion(ctx context.Context, namespacedName types.NamespacedName) {
	instanceKey := fmt.Sprintf("%s/%s", namespacedName.Namespace, namespacedName.Name)
	r.cleanupAdapterInstance(ctx, instanceKey)
	r.cleanupNetconfServer(ctx, instanceKey)
}

func (r *O1InterfaceController) cleanupAdapterInstance(ctx context.Context, instanceKey string) error {
	r.o1AdapterManager.mutex.Lock()
	defer r.o1AdapterManager.mutex.Unlock()

	if _, exists := r.o1AdapterManager.adapters[instanceKey]; exists {
		// The O1Adaptor doesn't have a Stop method - we'll just cleanup the instance.
		delete(r.o1AdapterManager.adapters, instanceKey)
	}

	return nil
}

func (r *O1InterfaceController) cleanupNetconfServer(ctx context.Context, instanceKey string) error {
	r.netconfServerManager.mutex.Lock()
	defer r.netconfServerManager.mutex.Unlock()

	if server, exists := r.netconfServerManager.servers[instanceKey]; exists {
		// Stop the server with a context.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Stop(ctx)
		delete(r.netconfServerManager.servers, instanceKey)
	}

	return nil
}

func (r *O1InterfaceController) updateNetconfServerConfig(server *NetconfServer, o1Interface *oranv1.O1Interface) error {
	// The NetconfServer doesn't have an UpdateConfig method.
	// For now, we'll just return nil - in a real implementation,.
	// we might need to recreate the server with the new config.
	return nil
}

// Metrics recording.
func (r *O1InterfaceController) recordReconciliationMetrics(namespacedName types.NamespacedName, result string, startTime time.Time) {
	duration := time.Since(startTime)
	r.metrics.ReconciliationsTotal.WithLabelValues(
		namespacedName.Namespace,
		namespacedName.Name,
		result,
	).Inc()
	r.metrics.ReconciliationDuration.WithLabelValues(
		namespacedName.Namespace,
		namespacedName.Name,
	).Observe(duration.Seconds())
}

// YANG model helpers (simplified).
func getORANHardwareYANG() string {
	return `module o-ran-hardware {
  // O-RAN hardware YANG model content.
}`
}

func getORANSoftwareYANG() string {
	return `module o-ran-software-management {
  // O-RAN software management YANG model content.
}`
}

func getORANPerformanceYANG() string {
	return `module o-ran-performance-management {
  // O-RAN performance management YANG model content.
}`
}

func getORANFaultYANG() string {
	return `module o-ran-fault-management {
  // O-RAN fault management YANG model content.
}`
}

// buildTLSConfig creates TLS configuration from O1Interface spec.
func (r *O1InterfaceController) buildTLSConfig(o1Interface *oranv1.O1Interface) *oran.TLSConfig {
	if o1Interface.Spec.Protocol != "tls" {
		return nil
	}

	return &oran.TLSConfig{
		CertFile:   "/etc/certs/tls.crt",
		KeyFile:    "/etc/certs/tls.key",
		CAFile:     "/etc/certs/ca.crt",
		SkipVerify: false,
	}
}

// buildAuthConfig creates authentication configuration from O1Interface spec.
func (r *O1InterfaceController) buildAuthConfig(o1Interface *oranv1.O1Interface) *oran.AuthConfig {
	if o1Interface.Spec.Credentials.UsernameRef == nil {
		return nil
	}

	return &oran.AuthConfig{
		Type:     "basic",
		Username: "netconf",    // Default username
		Password: "netconf123", // This should be retrieved from Secret
	}
}
