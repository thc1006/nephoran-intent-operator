/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

package disaster

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (

	// Component health metrics.

	componentHealthStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{

		Name: "disaster_recovery_component_health",

		Help: "Health status of disaster recovery components (0=unhealthy, 1=healthy)",
	}, []string{"component"})

	// Recovery operation metrics.

	recoveryOperations = promauto.NewCounterVec(prometheus.CounterOpts{

		Name: "disaster_recovery_operations_total",

		Help: "Total number of disaster recovery operations",
	}, []string{"operation", "status"})

	// RTO/RPO tracking - commented out unused metrics.

	// recoveryTimeObjective = promauto.NewGaugeVec(prometheus.GaugeOpts{.

	//	Name: "disaster_recovery_rto_seconds",

	//	Help: "Recovery Time Objective in seconds",

	// }, []string{"region"}).

	// recoveryPointObjective = promauto.NewGaugeVec(prometheus.GaugeOpts{.

	//	Name: "disaster_recovery_rpo_seconds",

	//	Help: "Recovery Point Objective in seconds",

	// }, []string{"region"}).

	// Service restart duration.

	serviceRestartDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{

		Name: "disaster_recovery_service_restart_duration_seconds",

		Help: "Duration of service restart operations",

		Buckets: prometheus.ExponentialBuckets(1, 2, 10),
	}, []string{"service"})
)

// ComponentHealthChecker defines health check interface for components.

type ComponentHealthChecker interface {
	CheckHealth(ctx context.Context) error

	GetComponentName() string

	GetDependencies() []string
}

// DisasterRecoveryManager manages disaster recovery operations with real Kubernetes integration.

type DisasterRecoveryManager struct {
	mu sync.RWMutex

	logger *slog.Logger

	k8sClient kubernetes.Interface

	redisClient *redis.Client

	config *DisasterRecoveryConfig

	healthCheckers map[string]ComponentHealthChecker

	backupManager *BackupManager

	failoverManager *FailoverManager

	restoreManager *RestoreManager

	recoveryStatus map[string]*ComponentStatus

	lastHealthCheck time.Time

	// Recovery procedures.

	recoverySequence []RecoveryStep

	// Notifications.

	alertManager AlertManager
}

// DisasterRecoveryConfig holds configuration for disaster recovery.

type DisasterRecoveryConfig struct {

	// Health check configuration.

	HealthCheckInterval time.Duration `json:"health_check_interval"`

	HealthCheckTimeout time.Duration `json:"health_check_timeout"`

	ComponentEndpoints map[string]string `json:"component_endpoints"`

	// Recovery configuration.

	RecoveryEnabled bool `json:"recovery_enabled"`

	AutoRecovery bool `json:"auto_recovery"`

	RecoveryTimeoutMinutes int `json:"recovery_timeout_minutes"`

	// Dependency configuration.

	ComponentDependencies map[string][]string `json:"component_dependencies"`

	// Notification configuration.

	AlertingConfig AlertingConfig `json:"alerting_config"`

	// Redis configuration.

	RedisConfig RedisConfig `json:"redis_config"`
}

// ComponentStatus tracks the status of a component.

type ComponentStatus struct {
	Name string `json:"name"`

	Healthy bool `json:"healthy"`

	LastCheck time.Time `json:"last_check"`

	LastHealthy time.Time `json:"last_healthy"`

	ErrorCount int `json:"error_count"`

	Dependencies []string `json:"dependencies"`

	RecoveryAttempts int `json:"recovery_attempts"`

	LastRecovery time.Time `json:"last_recovery"`

	Metadata map[string]interface{} `json:"metadata"`
}

// RecoveryStep represents a step in the recovery process.

type RecoveryStep struct {
	Name string `json:"name"`

	Component string `json:"component"`

	Action func(ctx context.Context, drm *DisasterRecoveryManager) error

	Dependencies []string `json:"dependencies"`

	Timeout time.Duration `json:"timeout"`

	Critical bool `json:"critical"`

	Retries int `json:"retries"`
}

// AlertingConfig defines alerting configuration.

type AlertingConfig struct {
	SlackWebhook string `json:"slack_webhook"`

	EmailRecipients []string `json:"email_recipients"`

	AlertThreshold int `json:"alert_threshold"`
}

// RedisConfig defines Redis connection configuration.

type RedisConfig struct {
	Address string `json:"address"`

	Password string `json:"password"`

	DB int `json:"db"`
}

// AlertManager interface for sending alerts.

type AlertManager interface {
	SendAlert(ctx context.Context, alert Alert) error
}

// Alert represents an alert message.

type Alert struct {
	Level string `json:"level"`

	Component string `json:"component"`

	Message string `json:"message"`

	Timestamp time.Time `json:"timestamp"`

	Metadata map[string]interface{} `json:"metadata"`
}

// NewDisasterRecoveryManager creates a new disaster recovery manager.

func NewDisasterRecoveryManager(config *DisasterRecoveryConfig, k8sClient kubernetes.Interface, logger *slog.Logger) (*DisasterRecoveryManager, error) {

	if config == nil {

		return nil, fmt.Errorf("disaster recovery configuration is required")

	}

	if k8sClient == nil {

		return nil, fmt.Errorf("kubernetes client is required")

	}

	// Initialize Redis client.

	var redisClient *redis.Client

	if config.RedisConfig.Address != "" {

		redisClient = redis.NewClient(&redis.Options{

			Addr: config.RedisConfig.Address,

			Password: config.RedisConfig.Password,

			DB: config.RedisConfig.DB,
		})

	}

	drm := &DisasterRecoveryManager{

		logger: logger,

		k8sClient: k8sClient,

		redisClient: redisClient,

		config: config,

		healthCheckers: make(map[string]ComponentHealthChecker),

		recoveryStatus: make(map[string]*ComponentStatus),
	}

	// Initialize health checkers for core components.

	drm.initializeHealthCheckers()

	// Initialize recovery sequence.

	drm.initializeRecoverySequence()

	// Initialize sub-managers.

	var err error

	drm.backupManager, err = NewBackupManager(config, k8sClient, logger)

	if err != nil {

		logger.Error("Failed to initialize backup manager", "error", err)

	}

	drm.failoverManager, err = NewFailoverManager(config, k8sClient, logger)

	if err != nil {

		logger.Error("Failed to initialize failover manager", "error", err)

	}

	drm.restoreManager, err = NewRestoreManager(config, k8sClient, logger)

	if err != nil {

		logger.Error("Failed to initialize restore manager", "error", err)

	}

	return drm, nil

}

// initializeHealthCheckers sets up health checkers for all components.

func (drm *DisasterRecoveryManager) initializeHealthCheckers() {

	// LLM Processor health checker.

	drm.healthCheckers["llm-processor"] = &HTTPHealthChecker{

		name: "llm-processor",

		endpoint: drm.getComponentEndpoint("llm-processor", "http://llm-processor:8080/healthz"),

		timeout: drm.config.HealthCheckTimeout,

		logger: drm.logger,
	}

	// Weaviate health checker.

	drm.healthCheckers["weaviate"] = &HTTPHealthChecker{

		name: "weaviate",

		endpoint: drm.getComponentEndpoint("weaviate", "http://weaviate:8080/v1/.well-known/ready"),

		timeout: drm.config.HealthCheckTimeout,

		logger: drm.logger,
	}

	// Redis health checker.

	if drm.redisClient != nil {

		drm.healthCheckers["redis"] = &RedisHealthChecker{

			name: "redis",

			redisClient: drm.redisClient,

			timeout: drm.config.HealthCheckTimeout,

			logger: drm.logger,
		}

	}

	// Nephio Bridge health checker.

	drm.healthCheckers["nephio-bridge"] = &HTTPHealthChecker{

		name: "nephio-bridge",

		endpoint: drm.getComponentEndpoint("nephio-bridge", "http://nephio-bridge:8080/healthz"),

		timeout: drm.config.HealthCheckTimeout,

		logger: drm.logger,
	}

	// O-RAN Adaptor health checker.

	drm.healthCheckers["oran-adaptor"] = &HTTPHealthChecker{

		name: "oran-adaptor",

		endpoint: drm.getComponentEndpoint("oran-adaptor", "http://oran-adaptor:8080/healthz"),

		timeout: drm.config.HealthCheckTimeout,

		logger: drm.logger,
	}

	// Prometheus health checker.

	drm.healthCheckers["prometheus"] = &HTTPHealthChecker{

		name: "prometheus",

		endpoint: drm.getComponentEndpoint("prometheus", "http://prometheus:9090/-/healthy"),

		timeout: drm.config.HealthCheckTimeout,

		logger: drm.logger,
	}

	// Grafana health checker.

	drm.healthCheckers["grafana"] = &HTTPHealthChecker{

		name: "grafana",

		endpoint: drm.getComponentEndpoint("grafana", "http://grafana:3000/api/health"),

		timeout: drm.config.HealthCheckTimeout,

		logger: drm.logger,
	}

}

// getComponentEndpoint returns the endpoint for a component with fallback to default.

func (drm *DisasterRecoveryManager) getComponentEndpoint(component, defaultEndpoint string) string {

	if endpoint, exists := drm.config.ComponentEndpoints[component]; exists {

		return endpoint

	}

	return defaultEndpoint

}

// initializeRecoverySequence sets up the service restart sequence with dependency ordering.

func (drm *DisasterRecoveryManager) initializeRecoverySequence() {

	drm.recoverySequence = []RecoveryStep{

		{

			Name: "Restart Redis",

			Component: "redis",

			Action: drm.restartRedis,

			Dependencies: []string{},

			Timeout: 5 * time.Minute,

			Critical: true,

			Retries: 3,
		},

		{

			Name: "Restart Weaviate",

			Component: "weaviate",

			Action: drm.restartWeaviate,

			Dependencies: []string{},

			Timeout: 10 * time.Minute,

			Critical: true,

			Retries: 3,
		},

		{

			Name: "Restart Prometheus",

			Component: "prometheus",

			Action: drm.restartPrometheus,

			Dependencies: []string{},

			Timeout: 5 * time.Minute,

			Critical: false,

			Retries: 2,
		},

		{

			Name: "Restart LLM Processor",

			Component: "llm-processor",

			Action: drm.restartLLMProcessor,

			Dependencies: []string{"weaviate", "redis"},

			Timeout: 5 * time.Minute,

			Critical: true,

			Retries: 3,
		},

		{

			Name: "Restart Nephio Bridge",

			Component: "nephio-bridge",

			Action: drm.restartNephioBridge,

			Dependencies: []string{"llm-processor"},

			Timeout: 3 * time.Minute,

			Critical: true,

			Retries: 3,
		},

		{

			Name: "Restart O-RAN Adaptor",

			Component: "oran-adaptor",

			Action: drm.restartORANAdaptor,

			Dependencies: []string{"nephio-bridge"},

			Timeout: 3 * time.Minute,

			Critical: true,

			Retries: 3,
		},

		{

			Name: "Restart Grafana",

			Component: "grafana",

			Action: drm.restartGrafana,

			Dependencies: []string{"prometheus"},

			Timeout: 3 * time.Minute,

			Critical: false,

			Retries: 2,
		},
	}

}

// Start starts the disaster recovery manager.

func (drm *DisasterRecoveryManager) Start(ctx context.Context) error {

	drm.logger.Info("Starting disaster recovery manager")

	// Start health monitoring.

	go drm.startHealthMonitoring(ctx)

	// Start recovery monitoring.

	if drm.config.AutoRecovery {

		go drm.startRecoveryMonitoring(ctx)

	}

	// Initialize component status.

	for name := range drm.healthCheckers {

		drm.recoveryStatus[name] = &ComponentStatus{

			Name: name,

			Healthy: false,

			Dependencies: drm.getComponentDependencies(name),

			Metadata: make(map[string]interface{}),
		}

	}

	drm.logger.Info("Disaster recovery manager started successfully")

	return nil

}

// startHealthMonitoring starts continuous health monitoring.

func (drm *DisasterRecoveryManager) startHealthMonitoring(ctx context.Context) {

	ticker := time.NewTicker(drm.config.HealthCheckInterval)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			drm.logger.Info("Health monitoring stopped")

			return

		case <-ticker.C:

			drm.performHealthChecks(ctx)

		}

	}

}

// performHealthChecks performs health checks on all components.

func (drm *DisasterRecoveryManager) performHealthChecks(ctx context.Context) {

	drm.mu.Lock()

	defer drm.mu.Unlock()

	drm.lastHealthCheck = time.Now()

	for name, checker := range drm.healthCheckers {

		status := drm.recoveryStatus[name]

		checkCtx, cancel := context.WithTimeout(ctx, drm.config.HealthCheckTimeout)

		err := checker.CheckHealth(checkCtx)

		cancel()

		status.LastCheck = time.Now()

		if err != nil {

			drm.logger.Error("Health check failed", "component", name, "error", err)

			if status.Healthy {

				// Component just became unhealthy.

				componentHealthStatus.WithLabelValues(name).Set(0)

				drm.sendAlert(ctx, Alert{

					Level: "warning",

					Component: name,

					Message: fmt.Sprintf("Component %s is unhealthy: %v", name, err),

					Timestamp: time.Now(),

					Metadata: map[string]interface{}{

						"error": err.Error(),
					},
				})

			}

			status.Healthy = false

			status.ErrorCount++

		} else {

			if !status.Healthy {

				// Component just became healthy.

				componentHealthStatus.WithLabelValues(name).Set(1)

				drm.logger.Info("Component recovered", "component", name)

				drm.sendAlert(ctx, Alert{

					Level: "info",

					Component: name,

					Message: fmt.Sprintf("Component %s has recovered", name),

					Timestamp: time.Now(),
				})

			}

			status.Healthy = true

			status.LastHealthy = time.Now()

			status.ErrorCount = 0

		}

	}

}

// startRecoveryMonitoring monitors for components that need recovery.

func (drm *DisasterRecoveryManager) startRecoveryMonitoring(ctx context.Context) {

	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			drm.logger.Info("Recovery monitoring stopped")

			return

		case <-ticker.C:

			drm.checkForRecoveryNeeded(ctx)

		}

	}

}

// checkForRecoveryNeeded checks if any components need recovery.

func (drm *DisasterRecoveryManager) checkForRecoveryNeeded(ctx context.Context) {

	drm.mu.RLock()

	defer drm.mu.RUnlock()

	for name, status := range drm.recoveryStatus {

		if !status.Healthy && status.ErrorCount >= drm.config.AlertingConfig.AlertThreshold {

			// Check if we haven't attempted recovery recently.

			if time.Since(status.LastRecovery) > 10*time.Minute {

				drm.logger.Info("Initiating automatic recovery", "component", name)

				go func(componentName string) {
					if err := drm.recoverComponent(ctx, componentName); err != nil {
						drm.logger.Error("Failed to recover component", "component", componentName, "error", err)
					}
				}(name)

			}

		}

	}

}

// recoverComponent attempts to recover a specific component.

func (drm *DisasterRecoveryManager) recoverComponent(ctx context.Context, componentName string) error {

	drm.mu.Lock()

	status := drm.recoveryStatus[componentName]

	status.RecoveryAttempts++

	status.LastRecovery = time.Now()

	drm.mu.Unlock()

	drm.logger.Info("Starting component recovery", "component", componentName)

	// Find the recovery step for this component.

	var recoveryStep *RecoveryStep

	for _, step := range drm.recoverySequence {

		if step.Component == componentName {

			recoveryStep = &step

			break

		}

	}

	if recoveryStep == nil {

		return fmt.Errorf("no recovery step found for component %s", componentName)

	}

	// Check dependencies are healthy.

	for _, dep := range recoveryStep.Dependencies {

		depStatus := drm.recoveryStatus[dep]

		if depStatus != nil && !depStatus.Healthy {

			drm.logger.Warn("Dependency not healthy, attempting recovery", "component", componentName, "dependency", dep)

			if err := drm.recoverComponent(ctx, dep); err != nil {

				return fmt.Errorf("failed to recover dependency %s: %w", dep, err)

			}

		}

	}

	// Execute recovery action.

	recoveryCtx, cancel := context.WithTimeout(ctx, recoveryStep.Timeout)

	defer cancel()

	start := time.Now()

	err := recoveryStep.Action(recoveryCtx, drm)

	duration := time.Since(start)

	serviceRestartDuration.WithLabelValues(componentName).Observe(duration.Seconds())

	if err != nil {

		recoveryOperations.WithLabelValues("restart", "failed").Inc()

		drm.logger.Error("Component recovery failed", "component", componentName, "error", err)

		drm.sendAlert(ctx, Alert{

			Level: "error",

			Component: componentName,

			Message: fmt.Sprintf("Recovery failed for component %s: %v", componentName, err),

			Timestamp: time.Now(),

			Metadata: map[string]interface{}{

				"error": err.Error(),

				"duration": duration.String(),
			},
		})

		return err

	}

	recoveryOperations.WithLabelValues("restart", "success").Inc()

	drm.logger.Info("Component recovery completed", "component", componentName, "duration", duration)

	// Wait for component to become healthy.

	maxWait := 2 * time.Minute

	checkInterval := 10 * time.Second

	for elapsed := time.Duration(0); elapsed < maxWait; elapsed += checkInterval {

		time.Sleep(checkInterval)

		checker := drm.healthCheckers[componentName]

		if checker != nil {

			checkCtx, cancel := context.WithTimeout(ctx, drm.config.HealthCheckTimeout)

			err := checker.CheckHealth(checkCtx)

			cancel()

			if err == nil {

				drm.logger.Info("Component recovery verified", "component", componentName)

				return nil

			}

		}

	}

	return fmt.Errorf("component %s did not become healthy after recovery", componentName)

}

// getComponentDependencies returns the dependencies for a component.

func (drm *DisasterRecoveryManager) getComponentDependencies(component string) []string {

	if deps, exists := drm.config.ComponentDependencies[component]; exists {

		return deps

	}

	return []string{}

}

// sendAlert sends an alert through the alert manager.

func (drm *DisasterRecoveryManager) sendAlert(ctx context.Context, alert Alert) {

	if drm.alertManager != nil {

		go func() {

			if err := drm.alertManager.SendAlert(ctx, alert); err != nil {

				drm.logger.Error("Failed to send alert", "error", err)

			}

		}()

	}

}

// GetComponentStatus returns the status of a specific component.

func (drm *DisasterRecoveryManager) GetComponentStatus(componentName string) (*ComponentStatus, error) {

	drm.mu.RLock()

	defer drm.mu.RUnlock()

	status, exists := drm.recoveryStatus[componentName]

	if !exists {

		return nil, fmt.Errorf("component %s not found", componentName)

	}

	return status, nil

}

// GetAllComponentStatus returns the status of all components.

func (drm *DisasterRecoveryManager) GetAllComponentStatus() map[string]*ComponentStatus {

	drm.mu.RLock()

	defer drm.mu.RUnlock()

	result := make(map[string]*ComponentStatus)

	for name, status := range drm.recoveryStatus {

		result[name] = status

	}

	return result

}

// TriggerFailover triggers failover to a target region.

func (drm *DisasterRecoveryManager) TriggerFailover(ctx context.Context, targetRegion string) error {

	if drm.failoverManager == nil {

		return fmt.Errorf("failover manager not initialized")

	}

	return drm.failoverManager.TriggerFailover(ctx, targetRegion)

}

// HTTPHealthChecker implements health checking via HTTP endpoints.

type HTTPHealthChecker struct {
	name string

	endpoint string

	timeout time.Duration

	logger *slog.Logger
}

// CheckHealth performs checkhealth operation.

func (h *HTTPHealthChecker) CheckHealth(ctx context.Context) error {

	client := &http.Client{

		Timeout: h.timeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", h.endpoint, http.NoBody)

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	resp, err := client.Do(req)

	if err != nil {

		return fmt.Errorf("failed to perform health check: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		return fmt.Errorf("health check failed with status %d", resp.StatusCode)

	}

	return nil

}

// GetComponentName performs getcomponentname operation.

func (h *HTTPHealthChecker) GetComponentName() string {

	return h.name

}

// GetDependencies performs getdependencies operation.

func (h *HTTPHealthChecker) GetDependencies() []string {

	return []string{}

}

// RedisHealthChecker implements health checking for Redis.

type RedisHealthChecker struct {
	name string

	redisClient *redis.Client

	timeout time.Duration

	logger *slog.Logger
}

// CheckHealth performs checkhealth operation.

func (r *RedisHealthChecker) CheckHealth(ctx context.Context) error {

	pingCtx, cancel := context.WithTimeout(ctx, r.timeout)

	defer cancel()

	_, err := r.redisClient.Ping(pingCtx).Result()

	if err != nil {

		return fmt.Errorf("redis ping failed: %w", err)

	}

	return nil

}

// GetComponentName performs getcomponentname operation.

func (r *RedisHealthChecker) GetComponentName() string {

	return r.name

}

// GetDependencies performs getdependencies operation.

func (r *RedisHealthChecker) GetDependencies() []string {

	return []string{}

}

// Service restart methods using Kubernetes API.

// restartRedis restarts Redis deployment.

func (drm *DisasterRecoveryManager) restartRedis(ctx context.Context, _ *DisasterRecoveryManager) error {

	deploymentName := "redis"

	namespace := "nephoran-system"

	drm.logger.Info("Restarting Redis deployment", "deployment", deploymentName, "namespace", namespace)

	return drm.restartDeployment(ctx, deploymentName, namespace)

}

// restartWeaviate restarts Weaviate StatefulSet.

func (drm *DisasterRecoveryManager) restartWeaviate(ctx context.Context, _ *DisasterRecoveryManager) error {

	statefulSetName := "weaviate"

	namespace := "nephoran-system"

	drm.logger.Info("Restarting Weaviate StatefulSet", "statefulset", statefulSetName, "namespace", namespace)

	return drm.restartStatefulSet(ctx, statefulSetName, namespace)

}

// restartPrometheus restarts Prometheus deployment.

func (drm *DisasterRecoveryManager) restartPrometheus(ctx context.Context, _ *DisasterRecoveryManager) error {

	deploymentName := "prometheus"

	namespace := "nephoran-system"

	drm.logger.Info("Restarting Prometheus deployment", "deployment", deploymentName, "namespace", namespace)

	return drm.restartDeployment(ctx, deploymentName, namespace)

}

// restartLLMProcessor restarts LLM Processor deployment.

func (drm *DisasterRecoveryManager) restartLLMProcessor(ctx context.Context, _ *DisasterRecoveryManager) error {

	deploymentName := "llm-processor"

	namespace := "nephoran-system"

	drm.logger.Info("Restarting LLM Processor deployment", "deployment", deploymentName, "namespace", namespace)

	return drm.restartDeployment(ctx, deploymentName, namespace)

}

// restartNephioBridge restarts Nephio Bridge deployment.

func (drm *DisasterRecoveryManager) restartNephioBridge(ctx context.Context, _ *DisasterRecoveryManager) error {

	deploymentName := "nephio-bridge"

	namespace := "nephoran-system"

	drm.logger.Info("Restarting Nephio Bridge deployment", "deployment", deploymentName, "namespace", namespace)

	return drm.restartDeployment(ctx, deploymentName, namespace)

}

// restartORANAdaptor restarts O-RAN Adaptor deployment.

func (drm *DisasterRecoveryManager) restartORANAdaptor(ctx context.Context, _ *DisasterRecoveryManager) error {

	deploymentName := "oran-adaptor"

	namespace := "nephoran-system"

	drm.logger.Info("Restarting O-RAN Adaptor deployment", "deployment", deploymentName, "namespace", namespace)

	return drm.restartDeployment(ctx, deploymentName, namespace)

}

// restartGrafana restarts Grafana deployment.

func (drm *DisasterRecoveryManager) restartGrafana(ctx context.Context, _ *DisasterRecoveryManager) error {

	deploymentName := "grafana"

	namespace := "nephoran-system"

	drm.logger.Info("Restarting Grafana deployment", "deployment", deploymentName, "namespace", namespace)

	return drm.restartDeployment(ctx, deploymentName, namespace)

}

// restartDeployment restarts a Kubernetes deployment by adding restart annotation.

func (drm *DisasterRecoveryManager) restartDeployment(ctx context.Context, name, namespace string) error {

	deploymentsClient := drm.k8sClient.AppsV1().Deployments(namespace)

	// Get the deployment.

	deployment, err := deploymentsClient.Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return fmt.Errorf("failed to get deployment %s/%s: %w", namespace, name, err)

	}

	// Add restart annotation to trigger rolling restart.

	if deployment.Spec.Template.Annotations == nil {

		deployment.Spec.Template.Annotations = make(map[string]string)

	}

	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	// Update the deployment.

	_, err = deploymentsClient.Update(ctx, deployment, metav1.UpdateOptions{})

	if err != nil {

		return fmt.Errorf("failed to restart deployment %s/%s: %w", namespace, name, err)

	}

	drm.logger.Info("Deployment restart initiated", "deployment", name, "namespace", namespace)

	// Wait for deployment to be ready.

	return drm.waitForDeploymentReady(ctx, name, namespace, 5*time.Minute)

}

// restartStatefulSet restarts a Kubernetes StatefulSet.

func (drm *DisasterRecoveryManager) restartStatefulSet(ctx context.Context, name, namespace string) error {

	statefulSetsClient := drm.k8sClient.AppsV1().StatefulSets(namespace)

	// Get the StatefulSet.

	statefulSet, err := statefulSetsClient.Get(ctx, name, metav1.GetOptions{})

	if err != nil {

		return fmt.Errorf("failed to get statefulset %s/%s: %w", namespace, name, err)

	}

	// Add restart annotation to trigger rolling restart.

	if statefulSet.Spec.Template.Annotations == nil {

		statefulSet.Spec.Template.Annotations = make(map[string]string)

	}

	statefulSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	// Update the StatefulSet.

	_, err = statefulSetsClient.Update(ctx, statefulSet, metav1.UpdateOptions{})

	if err != nil {

		return fmt.Errorf("failed to restart statefulset %s/%s: %w", namespace, name, err)

	}

	drm.logger.Info("StatefulSet restart initiated", "statefulset", name, "namespace", namespace)

	// Wait for StatefulSet to be ready.

	return drm.waitForStatefulSetReady(ctx, name, namespace, 10*time.Minute)

}

// waitForDeploymentReady waits for a deployment to be ready.

func (drm *DisasterRecoveryManager) waitForDeploymentReady(ctx context.Context, name, namespace string, timeout time.Duration) error {

	ctx, cancel := context.WithTimeout(ctx, timeout)

	defer cancel()

	deploymentsClient := drm.k8sClient.AppsV1().Deployments(namespace)

	for {

		select {

		case <-ctx.Done():

			return fmt.Errorf("timeout waiting for deployment %s/%s to be ready", namespace, name)

		default:

		}

		deployment, err := deploymentsClient.Get(ctx, name, metav1.GetOptions{})

		if err != nil {

			drm.logger.Error("Failed to get deployment status", "deployment", name, "error", err)

			time.Sleep(10 * time.Second)

			continue

		}

		// Check if deployment is ready.

		if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas &&

			deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas {

			drm.logger.Info("Deployment is ready", "deployment", name, "namespace", namespace)

			return nil

		}

		drm.logger.Debug("Waiting for deployment to be ready",

			"deployment", name,

			"ready", deployment.Status.ReadyReplicas,

			"desired", *deployment.Spec.Replicas)

		time.Sleep(10 * time.Second)

	}

}

// waitForStatefulSetReady waits for a StatefulSet to be ready.

func (drm *DisasterRecoveryManager) waitForStatefulSetReady(ctx context.Context, name, namespace string, timeout time.Duration) error {

	ctx, cancel := context.WithTimeout(ctx, timeout)

	defer cancel()

	statefulSetsClient := drm.k8sClient.AppsV1().StatefulSets(namespace)

	for {

		select {

		case <-ctx.Done():

			return fmt.Errorf("timeout waiting for statefulset %s/%s to be ready", namespace, name)

		default:

		}

		statefulSet, err := statefulSetsClient.Get(ctx, name, metav1.GetOptions{})

		if err != nil {

			drm.logger.Error("Failed to get statefulset status", "statefulset", name, "error", err)

			time.Sleep(10 * time.Second)

			continue

		}

		// Check if StatefulSet is ready.

		if statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas &&

			statefulSet.Status.UpdatedReplicas == *statefulSet.Spec.Replicas {

			drm.logger.Info("StatefulSet is ready", "statefulset", name, "namespace", namespace)

			return nil

		}

		drm.logger.Debug("Waiting for statefulset to be ready",

			"statefulset", name,

			"ready", statefulSet.Status.ReadyReplicas,

			"desired", *statefulSet.Spec.Replicas)

		time.Sleep(10 * time.Second)

	}

}
