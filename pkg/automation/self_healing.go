package automation

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// Metrics for self-healing operations
	healingOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nephoran_self_healing_operations_total",
		Help: "Total number of self-healing operations performed",
	}, []string{"type", "status", "component"})

	healingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "nephoran_self_healing_duration_seconds",
		Help:    "Duration of self-healing operations",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
	}, []string{"type", "component"})

	systemHealth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "nephoran_system_health_status",
		Help: "Current health status of system components (0=unhealthy, 1=healthy)",
	}, []string{"component"})

	failureDetections = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nephoran_failure_detections_total",
		Help: "Total number of failures detected by predictive monitoring",
	}, []string{"type", "severity", "component"})
)

// SelfHealingManager manages autonomous system recovery and optimization
type SelfHealingManager struct {
	mu                   sync.RWMutex
	config               *SelfHealingConfig
	logger               *slog.Logger
	k8sClient            kubernetes.Interface
	ctrlClient           client.Client
	healthMonitor        *HealthMonitor
	failurePrediction    *FailurePrediction
	automatedRemediation *AutomatedRemediation
	alertManager         *AlertManager
	metrics              *SelfHealingMetrics
	running              bool
	stopCh               chan struct{}
}

// SelfHealingConfig defines self-healing configuration
type SelfHealingConfig struct {
	Enabled                   bool                        `json:"enabled"`
	MonitoringInterval        time.Duration               `json:"monitoring_interval"`
	PredictiveAnalysisEnabled bool                        `json:"predictive_analysis_enabled"`
	AutoRemediationEnabled    bool                        `json:"auto_remediation_enabled"`
	MaxConcurrentRemediations int                         `json:"max_concurrent_remediations"`
	HealthCheckTimeout        time.Duration               `json:"health_check_timeout"`
	FailureDetectionThreshold float64                     `json:"failure_detection_threshold"`
	ComponentConfigs          map[string]*ComponentConfig `json:"component_configs"`
	NotificationConfig        *NotificationConfig         `json:"notification_config"`
	BackupBeforeRemediation   bool                        `json:"backup_before_remediation"`
	RollbackOnFailure         bool                        `json:"rollback_on_failure"`
	LearningEnabled           bool                        `json:"learning_enabled"`
}

// ComponentConfig defines component-specific healing configuration
type ComponentConfig struct {
	Name                  string                 `json:"name"`
	HealthCheckEndpoint   string                 `json:"health_check_endpoint"`
	CriticalityLevel      string                 `json:"criticality_level"` // LOW, MEDIUM, HIGH, CRITICAL
	AutoHealingEnabled    bool                   `json:"auto_healing_enabled"`
	MaxRestartAttempts    int                    `json:"max_restart_attempts"`
	RestartCooldown       time.Duration          `json:"restart_cooldown"`
	ScalingEnabled        bool                   `json:"scaling_enabled"`
	MinReplicas           int32                  `json:"min_replicas"`
	MaxReplicas           int32                  `json:"max_replicas"`
	CustomRemediations    []*CustomRemediation   `json:"custom_remediations"`
	DependsOn             []string               `json:"depends_on"`
	ResourceLimits        *ResourceLimits        `json:"resource_limits"`
	PerformanceThresholds *PerformanceThresholds `json:"performance_thresholds"`
}

// CustomRemediation defines custom remediation actions
type CustomRemediation struct {
	Name        string                  `json:"name"`
	Trigger     string                  `json:"trigger"` // HEALTH_CHECK_FAILURE, HIGH_ERROR_RATE, etc.
	Action      string                  `json:"action"`  // RESTART, SCALE, REDEPLOY, CUSTOM_SCRIPT
	Parameters  map[string]interface{}  `json:"parameters"`
	Conditions  []*RemediationCondition `json:"conditions"`
	Timeout     time.Duration           `json:"timeout"`
	RetryPolicy *RetryPolicy            `json:"retry_policy"`
}

type RemediationCondition struct {
	Metric    string        `json:"metric"`
	Operator  string        `json:"operator"` // GT, LT, EQ, NE
	Threshold float64       `json:"threshold"`
	Duration  time.Duration `json:"duration"`
}

type RetryPolicy struct {
	MaxAttempts       int           `json:"max_attempts"`
	InitialDelay      time.Duration `json:"initial_delay"`
	MaxDelay          time.Duration `json:"max_delay"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
}

type ResourceLimits struct {
	MaxCPU    string `json:"max_cpu"`
	MaxMemory string `json:"max_memory"`
	MaxDisk   string `json:"max_disk"`
}

type PerformanceThresholds struct {
	MaxLatency    time.Duration `json:"max_latency"`
	MaxErrorRate  float64       `json:"max_error_rate"`
	MinThroughput float64       `json:"min_throughput"`
	MaxQueueDepth int64         `json:"max_queue_depth"`
}

// HealthMonitor continuously monitors system health
type HealthMonitor struct {
	mu             sync.RWMutex
	config         *SelfHealingConfig
	logger         *slog.Logger
	k8sClient      kubernetes.Interface
	healthCheckers map[string]*ComponentHealthChecker
	systemMetrics  *SystemHealthMetrics
	alertManager   *AlertManager
}

// ComponentHealthChecker monitors individual component health
type ComponentHealthChecker struct {
	component           *ComponentConfig
	lastCheck           time.Time
	consecutiveFailures int
	currentStatus       HealthStatus
	metrics             *ComponentMetrics
	restartHistory      []*RestartEvent
}

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "HEALTHY"
	HealthStatusDegraded  HealthStatus = "DEGRADED"
	HealthStatusUnhealthy HealthStatus = "UNHEALTHY"
	HealthStatusCritical  HealthStatus = "CRITICAL"
)

// FailurePrediction uses ML models to predict system failures
type FailurePrediction struct {
	mu                   sync.RWMutex
	config               *SelfHealingConfig
	logger               *slog.Logger
	predictionModels     map[string]*PredictionModel
	historicalData       *HistoricalDataStore
	anomalyDetector      *AnomalyDetector
	failureProbabilities map[string]float64
	predictionAccuracy   map[string]float64
}

type PredictionModel struct {
	Name             string             `json:"name"`
	Component        string             `json:"component"`
	ModelType        string             `json:"model_type"` // LINEAR_REGRESSION, NEURAL_NETWORK, ARIMA
	Features         []string           `json:"features"`
	Accuracy         float64            `json:"accuracy"`
	LastTraining     time.Time          `json:"last_training"`
	PredictionWindow time.Duration      `json:"prediction_window"`
	Thresholds       map[string]float64 `json:"thresholds"`
}

// AutomatedRemediation is defined in automated_remediation.go

type RemediationSession struct {
	ID           string                 `json:"id"`
	Component    string                 `json:"component"`
	Strategy     string                 `json:"strategy"`
	Status       string                 `json:"status"` // PENDING, RUNNING, COMPLETED, FAILED
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	Actions      []*RemediationAction   `json:"actions"`
	Results      map[string]interface{} `json:"results"`
	BackupID     string                 `json:"backup_id,omitempty"`
	RollbackPlan *RollbackPlan          `json:"rollback_plan,omitempty"`
}

type RemediationAction struct {
	Type       string                 `json:"type"`
	Target     string                 `json:"target"`
	Parameters map[string]interface{} `json:"parameters"`
	Status     string                 `json:"status"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Result     string                 `json:"result,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

type RemediationStrategy struct {
	Name        string                       `json:"name"`
	Conditions  []*RemediationCondition      `json:"conditions"`
	Actions     []*RemediationActionTemplate `json:"actions"`
	Priority    int                          `json:"priority"`
	Success     int                          `json:"success"`
	Total       int                          `json:"total"`
	SuccessRate float64                      `json:"success_rate"`
}

type RemediationActionTemplate struct {
	Type        string                 `json:"type"`
	Template    string                 `json:"template"`
	Parameters  map[string]interface{} `json:"parameters"`
	Timeout     time.Duration          `json:"timeout"`
	RetryPolicy *RetryPolicy           `json:"retry_policy"`
}

// Supporting types
type SystemHealthMetrics struct {
	OverallHealth       HealthStatus            `json:"overall_health"`
	ComponentHealth     map[string]HealthStatus `json:"component_health"`
	ActiveIncidents     int                     `json:"active_incidents"`
	ResolvedIncidents   int                     `json:"resolved_incidents"`
	PredictedFailures   map[string]float64      `json:"predicted_failures"`
	SystemLoad          float64                 `json:"system_load"`
	ResourceUtilization map[string]float64      `json:"resource_utilization"`
	PerformanceMetrics  map[string]float64      `json:"performance_metrics"`
}

type ComponentMetrics struct {
	ResponseTime time.Duration `json:"response_time"`
	ErrorRate    float64       `json:"error_rate"`
	Throughput   float64       `json:"throughput"`
	CPUUsage     float64       `json:"cpu_usage"`
	MemoryUsage  float64       `json:"memory_usage"`
	RestartCount int           `json:"restart_count"`
	LastRestart  *time.Time    `json:"last_restart,omitempty"`
}

type RestartEvent struct {
	Timestamp time.Time     `json:"timestamp"`
	Reason    string        `json:"reason"`
	Success   bool          `json:"success"`
	Duration  time.Duration `json:"duration"`
}

type SelfHealingMetrics struct {
	TotalHealingOperations      int64              `json:"total_healing_operations"`
	SuccessfulHealingOperations int64              `json:"successful_healing_operations"`
	FailedHealingOperations     int64              `json:"failed_healing_operations"`
	AverageHealingTime          time.Duration      `json:"average_healing_time"`
	ComponentAvailability       map[string]float64 `json:"component_availability"`
	MTTR                        time.Duration      `json:"mttr"` // Mean Time To Recovery
	MTBF                        time.Duration      `json:"mtbf"` // Mean Time Between Failures
}

// NewSelfHealingManager creates a new self-healing manager
func NewSelfHealingManager(config *SelfHealingConfig, k8sClient kubernetes.Interface, ctrlClient client.Client, logger *slog.Logger) (*SelfHealingManager, error) {
	if config == nil {
		return nil, fmt.Errorf("self-healing configuration is required")
	}

	// Set defaults
	if config.MonitoringInterval == 0 {
		config.MonitoringInterval = 30 * time.Second
	}
	if config.HealthCheckTimeout == 0 {
		config.HealthCheckTimeout = 10 * time.Second
	}
	if config.FailureDetectionThreshold == 0 {
		config.FailureDetectionThreshold = 0.8
	}
	if config.MaxConcurrentRemediations == 0 {
		config.MaxConcurrentRemediations = 3
	}

	manager := &SelfHealingManager{
		config:     config,
		logger:     logger,
		k8sClient:  k8sClient,
		ctrlClient: ctrlClient,
		stopCh:     make(chan struct{}),
		metrics:    &SelfHealingMetrics{},
	}

	// Initialize health monitor
	healthMonitor, err := NewHealthMonitor(config, k8sClient, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create health monitor: %w", err)
	}
	manager.healthMonitor = healthMonitor

	// Initialize failure prediction
	if config.PredictiveAnalysisEnabled {
		failurePrediction, err := NewFailurePrediction(config, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create failure prediction: %w", err)
		}
		manager.failurePrediction = failurePrediction
	}

	// Initialize automated remediation
	if config.AutoRemediationEnabled {
		automatedRemediation, err := NewAutomatedRemediation(config, k8sClient, ctrlClient, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create automated remediation: %w", err)
		}
		manager.automatedRemediation = automatedRemediation
	}

	// Initialize alert manager
	if config.NotificationConfig != nil {
		alertManager, err := NewAlertManager(config.NotificationConfig, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create alert manager: %w", err)
		}
		manager.alertManager = alertManager
		manager.healthMonitor.alertManager = alertManager
	}

	return manager, nil
}

// Start starts the self-healing manager
func (shm *SelfHealingManager) Start(ctx context.Context) error {
	shm.mu.Lock()
	defer shm.mu.Unlock()

	if !shm.config.Enabled {
		shm.logger.Info("Self-healing is disabled")
		return nil
	}

	if shm.running {
		return fmt.Errorf("self-healing manager is already running")
	}

	shm.logger.Info("Starting self-healing manager")

	// Start health monitoring
	go shm.healthMonitor.Start(ctx)

	// Start failure prediction if enabled
	if shm.failurePrediction != nil {
		go shm.failurePrediction.Start(ctx)
	}

	// Start automated remediation if enabled
	if shm.automatedRemediation != nil {
		go shm.automatedRemediation.Start(ctx)
	}

	// Start main self-healing loop
	go shm.run(ctx)

	shm.running = true
	shm.logger.Info("Self-healing manager started successfully")

	return nil
}

// Stop stops the self-healing manager
func (shm *SelfHealingManager) Stop() {
	shm.mu.Lock()
	defer shm.mu.Unlock()

	if !shm.running {
		return
	}

	shm.logger.Info("Stopping self-healing manager")
	close(shm.stopCh)
	shm.running = false
}

// run executes the main self-healing loop
func (shm *SelfHealingManager) run(ctx context.Context) {
	ticker := time.NewTicker(shm.config.MonitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-shm.stopCh:
			return
		case <-ticker.C:
			shm.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck performs comprehensive health check and initiates healing if needed
func (shm *SelfHealingManager) performHealthCheck(ctx context.Context) {
	start := time.Now()
	defer func() {
		healingDuration.WithLabelValues("health_check", "system").Observe(time.Since(start).Seconds())
	}()

	shm.logger.Debug("Performing system health check")

	// Get current system health
	systemHealth := shm.healthMonitor.GetSystemHealth()

	// Update metrics
	shm.updateHealthMetrics(systemHealth)

	// Check for components requiring healing
	for component, health := range systemHealth.ComponentHealth {
		if health == HealthStatusUnhealthy || health == HealthStatusCritical {
			shm.logger.Warn("Unhealthy component detected", "component", component, "status", health)

			if shm.automatedRemediation != nil {
				err := shm.automatedRemediation.InitiateRemediation(ctx, component, string(health))
				if err != nil {
					shm.logger.Error("Failed to initiate remediation", "component", component, "error", err)
					healingOperations.WithLabelValues("remediation", "failed", component).Inc()
				} else {
					healingOperations.WithLabelValues("remediation", "initiated", component).Inc()
				}
			}
		}
	}

	// Check predictive failures
	if shm.failurePrediction != nil {
		predictions := shm.failurePrediction.GetFailureProbabilities()
		for component, probability := range predictions {
			if probability > shm.config.FailureDetectionThreshold {
				shm.logger.Warn("High failure probability detected",
					"component", component, "probability", probability)

				// Proactive remediation
				if shm.automatedRemediation != nil {
					err := shm.automatedRemediation.InitiatePreventiveRemediation(ctx, component, probability)
					if err != nil {
						shm.logger.Error("Failed to initiate preventive remediation",
							"component", component, "error", err)
					}
				}
			}
		}
	}
}

// updateHealthMetrics updates system health metrics
func (shm *SelfHealingManager) updateHealthMetrics(health *SystemHealthMetrics) {
	for component, status := range health.ComponentHealth {
		var healthValue float64
		switch status {
		case HealthStatusHealthy:
			healthValue = 1.0
		case HealthStatusDegraded:
			healthValue = 0.7
		case HealthStatusUnhealthy:
			healthValue = 0.3
		case HealthStatusCritical:
			healthValue = 0.0
		}
		systemHealth.WithLabelValues(component).Set(healthValue)
	}
}

// GetSystemHealth returns current system health status
func (shm *SelfHealingManager) GetSystemHealth() *SystemHealthMetrics {
	return shm.healthMonitor.GetSystemHealth()
}

// GetRemediationStatus returns status of active remediations
func (shm *SelfHealingManager) GetRemediationStatus() map[string]*RemediationSession {
	if shm.automatedRemediation == nil {
		return make(map[string]*RemediationSession)
	}
	return shm.automatedRemediation.GetActiveRemediations()
}

// GetMetrics returns self-healing metrics
func (shm *SelfHealingManager) GetMetrics() *SelfHealingMetrics {
	shm.mu.RLock()
	defer shm.mu.RUnlock()

	metrics := *shm.metrics
	return &metrics
}

// ForceHealing manually triggers healing for a specific component
func (shm *SelfHealingManager) ForceHealing(ctx context.Context, component string, reason string) error {
	shm.logger.Info("Forcing healing for component", "component", component, "reason", reason)

	if shm.automatedRemediation == nil {
		return fmt.Errorf("automated remediation is not enabled")
	}

	return shm.automatedRemediation.InitiateRemediation(ctx, component, reason)
}
