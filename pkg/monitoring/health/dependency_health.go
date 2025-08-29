
package health



import (

	"context"

	"fmt"

	"log/slog"

	"net/http"

	"os"

	"strings"

	"sync"

	"time"



	"github.com/prometheus/client_golang/prometheus"

	"github.com/sony/gobreaker"



	"github.com/thc1006/nephoran-intent-operator/pkg/health"



	"k8s.io/client-go/kubernetes"

)



// DependencyHealthTracker manages comprehensive health monitoring of external dependencies.

type DependencyHealthTracker struct {

	// Core configuration.

	logger      *slog.Logger

	serviceName string



	// Dependency registry.

	dependencies  map[string]*DependencyConfig

	healthResults map[string]*DependencyHealth

	mu            sync.RWMutex



	// Circuit breakers per dependency.

	circuitBreakers map[string]*gobreaker.CircuitBreaker



	// Health propagation.

	propagationRules map[string][]PropagationRule

	impactAnalysis   *ImpactAnalyzer



	// Recovery tracking.

	recoveryTracker *RecoveryTracker



	// Metrics.

	metrics *DependencyMetrics



	// Kubernetes client for cluster health.

	kubeClient kubernetes.Interface

}



// DependencyConfig holds configuration for a dependency.

type DependencyConfig struct {

	Name        string                `json:"name"`

	Type        DependencyType        `json:"type"`

	Category    DependencyCategory    `json:"category"`

	Criticality DependencyCriticality `json:"criticality"`



	// Connection details.

	Endpoint      string        `json:"endpoint"`

	Timeout       time.Duration `json:"timeout"`

	CheckInterval time.Duration `json:"check_interval"`



	// Health check configuration.

	HealthCheckConfig HealthCheckConfig `json:"health_check_config"`



	// Circuit breaker configuration.

	CircuitBreakerConfig CircuitBreakerConfig `json:"circuit_breaker_config"`



	// Service mesh configuration.

	ServiceMeshConfig *ServiceMeshConfig `json:"service_mesh_config,omitempty"`



	// Custom metadata.

	Metadata map[string]interface{} `json:"metadata,omitempty"`

}



// DependencyType represents the type of dependency.

type DependencyType string



const (

	// DepTypeLLMAPI holds deptypellmapi value.

	DepTypeLLMAPI DependencyType = "llm_api"

	// DepTypeDatabase holds deptypedatabase value.

	DepTypeDatabase DependencyType = "database"

	// DepTypeKubernetesAPI holds deptypekubernetesapi value.

	DepTypeKubernetesAPI DependencyType = "kubernetes_api"

	// DepTypeStorage holds deptypestorage value.

	DepTypeStorage DependencyType = "storage"

	// DepTypeMessageQueue holds deptypemessagequeue value.

	DepTypeMessageQueue DependencyType = "message_queue"

	// DepTypeExternalAPI holds deptypeexternalapi value.

	DepTypeExternalAPI DependencyType = "external_api"

	// DepTypeServiceMesh holds deptypeservicemesh value.

	DepTypeServiceMesh DependencyType = "service_mesh"

	// DepTypeCache holds deptypecache value.

	DepTypeCache DependencyType = "cache"

)



// DependencyCategory categorizes dependencies by their role.

type DependencyCategory string



const (

	// CatCore holds catcore value.

	CatCore DependencyCategory = "core"

	// CatInfrastructure holds catinfrastructure value.

	CatInfrastructure DependencyCategory = "infrastructure"

	// CatExternal holds catexternal value.

	CatExternal DependencyCategory = "external"

	// CatOptional holds catoptional value.

	CatOptional DependencyCategory = "optional"

)



// DependencyCriticality represents how critical a dependency is.

type DependencyCriticality string



const (

	// CriticalityEssential holds criticalityessential value.

	CriticalityEssential DependencyCriticality = "essential"

	// CriticalityImportant holds criticalityimportant value.

	CriticalityImportant DependencyCriticality = "important"

	// CriticalityNormal holds criticalitynormal value.

	CriticalityNormal DependencyCriticality = "normal"

	// CriticalityOptional holds criticalityoptional value.

	CriticalityOptional DependencyCriticality = "optional"

)



// DependencyHealth represents the health status of a dependency.

type DependencyHealth struct {

	Name         string         `json:"name"`

	Type         DependencyType `json:"type"`

	Status       health.Status  `json:"status"`

	LastChecked  time.Time      `json:"last_checked"`

	ResponseTime time.Duration  `json:"response_time"`



	// Availability metrics.

	Uptime           time.Duration `json:"uptime"`

	DowntimeTotal    time.Duration `json:"downtime_total"`

	AvailabilityRate float64       `json:"availability_rate"`



	// Error tracking.

	ErrorCount        int       `json:"error_count"`

	ConsecutiveErrors int       `json:"consecutive_errors"`

	LastError         string    `json:"last_error,omitempty"`

	LastErrorTime     time.Time `json:"last_error_time,omitempty"`



	// Circuit breaker status.

	CircuitBreakerState CircuitBreakerState `json:"circuit_breaker_state"`



	// Connection pool status.

	ConnectionPool *ConnectionPoolHealth `json:"connection_pool,omitempty"`



	// Service mesh health.

	ServiceMeshHealth *ServiceMeshHealth `json:"service_mesh_health,omitempty"`



	// Recovery information.

	RecoveryInfo *RecoveryInfo `json:"recovery_info,omitempty"`



	// Impact assessment.

	ImpactAssessment *ImpactAssessment `json:"impact_assessment,omitempty"`



	// Additional details.

	Details map[string]interface{} `json:"details,omitempty"`

}



// HealthCheckConfig configures health check behavior.

type HealthCheckConfig struct {

	Method         string            `json:"method"`          // GET, POST, etc.

	Path           string            `json:"path"`            // Health check endpoint path

	ExpectedStatus []int             `json:"expected_status"` // Expected HTTP status codes

	Headers        map[string]string `json:"headers,omitempty"`

	Body           string            `json:"body,omitempty"`



	// Validation rules.

	ResponseValidation *ResponseValidation `json:"response_validation,omitempty"`

}



// ResponseValidation defines rules for validating health check responses.

type ResponseValidation struct {

	ContentType     string                 `json:"content_type,omitempty"`

	RequiredFields  []string               `json:"required_fields,omitempty"`

	ExpectedValues  map[string]interface{} `json:"expected_values,omitempty"`

	MaxResponseSize int64                  `json:"max_response_size,omitempty"`

}



// CircuitBreakerConfig configures circuit breaker behavior.

type CircuitBreakerConfig struct {

	Enabled          bool          `json:"enabled"`

	MaxRequests      uint32        `json:"max_requests"`

	Interval         time.Duration `json:"interval"`

	Timeout          time.Duration `json:"timeout"`

	FailureThreshold float64       `json:"failure_threshold"`

}



// ServiceMeshConfig holds service mesh specific configuration.

type ServiceMeshConfig struct {

	Enabled       bool           `json:"enabled"`

	ServiceName   string         `json:"service_name"`

	Namespace     string         `json:"namespace"`

	MeshType      string         `json:"mesh_type"` // istio, linkerd, etc.

	RetryPolicy   *RetryPolicy   `json:"retry_policy,omitempty"`

	TimeoutPolicy *TimeoutPolicy `json:"timeout_policy,omitempty"`

}



// RetryPolicy defines retry behavior in service mesh.

type RetryPolicy struct {

	Attempts      int           `json:"attempts"`

	PerTryTimeout time.Duration `json:"per_try_timeout"`

	RetryOn       []string      `json:"retry_on"`

}



// TimeoutPolicy defines timeout behavior in service mesh.

type TimeoutPolicy struct {

	RequestTimeout time.Duration `json:"request_timeout"`

	IdleTimeout    time.Duration `json:"idle_timeout"`

	ConnectTimeout time.Duration `json:"connect_timeout"`

}



// CircuitBreakerState represents circuit breaker states.

type CircuitBreakerState string



const (

	// CBStateClosed holds cbstateclosed value.

	CBStateClosed CircuitBreakerState = "closed"

	// CBStateOpen holds cbstateopen value.

	CBStateOpen CircuitBreakerState = "open"

	// CBStateHalfOpen holds cbstatehalfopen value.

	CBStateHalfOpen CircuitBreakerState = "half_open"

	// CBStateDisabled holds cbstatedisabled value.

	CBStateDisabled CircuitBreakerState = "disabled"

)



// ConnectionPoolHealth represents connection pool health.

type ConnectionPoolHealth struct {

	ActiveConnections int           `json:"active_connections"`

	IdleConnections   int           `json:"idle_connections"`

	MaxConnections    int           `json:"max_connections"`

	ConnectionWait    time.Duration `json:"connection_wait"`

	PoolUtilization   float64       `json:"pool_utilization"`

}



// ServiceMeshHealth represents service mesh specific health information.

type ServiceMeshHealth struct {

	ProxyStatus     string  `json:"proxy_status"`

	RequestsSuccess float64 `json:"requests_success"`

	RequestsError   float64 `json:"requests_error"`

	LatencyP50      float64 `json:"latency_p50"`

	LatencyP95      float64 `json:"latency_p95"`

	LatencyP99      float64 `json:"latency_p99"`

}



// RecoveryInfo tracks recovery progress and timing.

type RecoveryInfo struct {

	RecoveryStarted     time.Time     `json:"recovery_started"`

	RecoveryDuration    time.Duration `json:"recovery_duration"`

	RecoveryAttempts    int           `json:"recovery_attempts"`

	AutoRecoveryEnabled bool          `json:"auto_recovery_enabled"`

	RecoveryStrategy    string        `json:"recovery_strategy"`

	RecoveryProgress    float64       `json:"recovery_progress"` // 0.0 to 1.0

}



// ImpactAssessment assesses the impact of dependency issues.

type ImpactAssessment struct {

	ServiceImpact      ServiceImpactLevel  `json:"service_impact"`

	AffectedFeatures   []string            `json:"affected_features"`

	UserImpact         UserImpactLevel     `json:"user_impact"`

	BusinessImpact     BusinessImpactLevel `json:"business_impact"`

	MitigationActive   bool                `json:"mitigation_active"`

	MitigationStrategy string              `json:"mitigation_strategy,omitempty"`

}



// ServiceImpactLevel represents the level of service impact.

type ServiceImpactLevel string



const (

	// ServiceImpactNone holds serviceimpactnone value.

	ServiceImpactNone ServiceImpactLevel = "none"

	// ServiceImpactPartial holds serviceimpactpartial value.

	ServiceImpactPartial ServiceImpactLevel = "partial"

	// ServiceImpactMajor holds serviceimpactmajor value.

	ServiceImpactMajor ServiceImpactLevel = "major"

	// ServiceImpactComplete holds serviceimpactcomplete value.

	ServiceImpactComplete ServiceImpactLevel = "complete"

)



// UserImpactLevel represents the level of user impact.

type UserImpactLevel string



const (

	// UserImpactNone holds userimpactnone value.

	UserImpactNone UserImpactLevel = "none"

	// UserImpactMinor holds userimpactminor value.

	UserImpactMinor UserImpactLevel = "minor"

	// UserImpactModerate holds userimpactmoderate value.

	UserImpactModerate UserImpactLevel = "moderate"

	// UserImpactSevere holds userimpactsevere value.

	UserImpactSevere UserImpactLevel = "severe"

)



// BusinessImpactLevel represents the level of business impact.

type BusinessImpactLevel string



const (

	// BusinessImpactNone holds businessimpactnone value.

	BusinessImpactNone BusinessImpactLevel = "none"

	// BusinessImpactLow holds businessimpactlow value.

	BusinessImpactLow BusinessImpactLevel = "low"

	// BusinessImpactMedium holds businessimpactmedium value.

	BusinessImpactMedium BusinessImpactLevel = "medium"

	// BusinessImpactHigh holds businessimpacthigh value.

	BusinessImpactHigh BusinessImpactLevel = "high"

	// BusinessImpactCritical holds businessimpactcritical value.

	BusinessImpactCritical BusinessImpactLevel = "critical"

)



// PropagationRule defines how dependency health propagates to service health.

type PropagationRule struct {

	SourceDependency string                 `json:"source_dependency"`

	TargetService    string                 `json:"target_service"`

	PropagationType  PropagationType        `json:"propagation_type"`

	WeightFactor     float64                `json:"weight_factor"`

	Conditions       []PropagationCondition `json:"conditions,omitempty"`

}



// PropagationType defines how health propagates.

type PropagationType string



const (

	// PropagationDirect holds propagationdirect value.

	PropagationDirect PropagationType = "direct" // 1:1 propagation

	// PropagationWeighted holds propagationweighted value.

	PropagationWeighted PropagationType = "weighted" // Weighted propagation

	// PropagationThreshold holds propagationthreshold value.

	PropagationThreshold PropagationType = "threshold" // Threshold-based

	// PropagationAggregated holds propagationaggregated value.

	PropagationAggregated PropagationType = "aggregated" // Aggregated with others

)



// PropagationCondition defines conditions for health propagation.

type PropagationCondition struct {

	Field    string      `json:"field"`

	Operator string      `json:"operator"`

	Value    interface{} `json:"value"`

}



// RecoveryTracker tracks dependency recovery processes.

type RecoveryTracker struct {

	recoveryMap map[string]*RecoveryState

	recoveryMu  sync.RWMutex

	logger      *slog.Logger

}



// RecoveryState tracks the state of a dependency recovery.

type RecoveryState struct {

	DependencyName  string           `json:"dependency_name"`

	StartTime       time.Time        `json:"start_time"`

	Strategy        RecoveryStrategy `json:"strategy"`

	Attempts        int              `json:"attempts"`

	MaxAttempts     int              `json:"max_attempts"`

	BackoffStrategy BackoffStrategy  `json:"backoff_strategy"`

	CurrentBackoff  time.Duration    `json:"current_backoff"`

	LastAttempt     time.Time        `json:"last_attempt"`

	Success         bool             `json:"success"`

	Automated       bool             `json:"automated"`

}



// RecoveryStrategy defines the strategy for dependency recovery.

type RecoveryStrategy string



const (

	// RecoveryRetry holds recoveryretry value.

	RecoveryRetry RecoveryStrategy = "retry"

	// RecoveryReconnect holds recoveryreconnect value.

	RecoveryReconnect RecoveryStrategy = "reconnect"

	// RecoveryRestart holds recoveryrestart value.

	RecoveryRestart RecoveryStrategy = "restart"

	// RecoveryFailover holds recoveryfailover value.

	RecoveryFailover RecoveryStrategy = "failover"

	// RecoveryManual holds recoverymanual value.

	RecoveryManual RecoveryStrategy = "manual"

)



// BackoffStrategy defines backoff behavior for recovery attempts.

type BackoffStrategy string



const (

	// BackoffLinear holds backofflinear value.

	BackoffLinear BackoffStrategy = "linear"

	// BackoffExponential holds backoffexponential value.

	BackoffExponential BackoffStrategy = "exponential"

	// BackoffFixed holds backofffixed value.

	BackoffFixed BackoffStrategy = "fixed"

	// BackoffJittered holds backoffjittered value.

	BackoffJittered BackoffStrategy = "jittered"

)



// ImpactAnalyzer analyzes the impact of dependency failures.

type ImpactAnalyzer struct {

	dependencyGraph *DependencyGraph

	impactRules     map[string][]ImpactRule

	logger          *slog.Logger

}



// ImpactRule defines how to assess impact from dependency failures.

type ImpactRule struct {

	DependencyType DependencyType        `json:"dependency_type"`

	Criticality    DependencyCriticality `json:"criticality"`

	ServiceImpact  ServiceImpactLevel    `json:"service_impact"`

	UserImpact     UserImpactLevel       `json:"user_impact"`

	BusinessImpact BusinessImpactLevel   `json:"business_impact"`

	Features       []string              `json:"features"`

	Conditions     []ImpactCondition     `json:"conditions,omitempty"`

}



// ImpactCondition defines conditions for impact assessment.

type ImpactCondition struct {

	Field    string      `json:"field"`

	Operator string      `json:"operator"`

	Value    interface{} `json:"value"`

}



// DependencyMetrics contains Prometheus metrics for dependency health.

type DependencyMetrics struct {

	HealthStatus          *prometheus.GaugeVec

	ResponseTime          *prometheus.HistogramVec

	ErrorCount            *prometheus.CounterVec

	AvailabilityRate      *prometheus.GaugeVec

	CircuitBreakerState   *prometheus.GaugeVec

	RecoveryAttempts      *prometheus.CounterVec

	ConnectionPoolMetrics *prometheus.GaugeVec

}



// NewDependencyHealthTracker creates a new dependency health tracker.

func NewDependencyHealthTracker(serviceName string, kubeClient kubernetes.Interface, logger *slog.Logger) *DependencyHealthTracker {

	if logger == nil {

		logger = slog.Default()

	}



	tracker := &DependencyHealthTracker{

		logger:           logger.With("component", "dependency_health_tracker"),

		serviceName:      serviceName,

		dependencies:     make(map[string]*DependencyConfig),

		healthResults:    make(map[string]*DependencyHealth),

		circuitBreakers:  make(map[string]*gobreaker.CircuitBreaker),

		propagationRules: make(map[string][]PropagationRule),

		kubeClient:       kubeClient,

		metrics:          initializeDependencyMetrics(),

	}



	// Initialize components.

	tracker.recoveryTracker = &RecoveryTracker{

		recoveryMap: make(map[string]*RecoveryState),

		logger:      logger.With("component", "recovery_tracker"),

	}



	tracker.impactAnalysis = &ImpactAnalyzer{

		dependencyGraph: &DependencyGraph{},

		impactRules:     make(map[string][]ImpactRule),

		logger:          logger.With("component", "impact_analyzer"),

	}



	// Register default dependencies.

	tracker.registerDefaultDependencies()



	return tracker

}



// initializeDependencyMetrics initializes Prometheus metrics.

func initializeDependencyMetrics() *DependencyMetrics {

	return &DependencyMetrics{

		HealthStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "dependency_health_status",

			Help: "Health status of dependencies (0=unhealthy, 1=healthy, 0.5=degraded)",

		}, []string{"dependency", "type", "criticality"}),



		ResponseTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name:    "dependency_response_time_seconds",

			Help:    "Response time of dependency health checks",

			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),

		}, []string{"dependency", "type"}),



		ErrorCount: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "dependency_errors_total",

			Help: "Total number of dependency errors",

		}, []string{"dependency", "type", "error_type"}),



		AvailabilityRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "dependency_availability_rate",

			Help: "Availability rate of dependencies",

		}, []string{"dependency", "type"}),



		CircuitBreakerState: prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "dependency_circuit_breaker_state",

			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",

		}, []string{"dependency"}),



		RecoveryAttempts: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "dependency_recovery_attempts_total",

			Help: "Total number of dependency recovery attempts",

		}, []string{"dependency", "strategy"}),



		ConnectionPoolMetrics: prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "dependency_connection_pool_status",

			Help: "Connection pool status metrics",

		}, []string{"dependency", "metric_type"}),

	}

}



// RegisterDependency registers a new dependency for monitoring.

func (dht *DependencyHealthTracker) RegisterDependency(config *DependencyConfig) error {

	if config == nil {

		return fmt.Errorf("dependency config cannot be nil")

	}

	if config.Name == "" {

		return fmt.Errorf("dependency name cannot be empty")

	}



	dht.mu.Lock()

	defer dht.mu.Unlock()



	// Store configuration.

	dht.dependencies[config.Name] = config



	// Initialize health result.

	dht.healthResults[config.Name] = &DependencyHealth{

		Name:             config.Name,

		Type:             config.Type,

		Status:           health.StatusUnknown,

		LastChecked:      time.Time{},

		AvailabilityRate: 0.0,

		Details:          make(map[string]interface{}),

	}



	// Create circuit breaker if enabled.

	if config.CircuitBreakerConfig.Enabled {

		dht.createCircuitBreaker(config)

	}



	dht.logger.Info("Dependency registered",

		"name", config.Name,

		"type", config.Type,

		"endpoint", config.Endpoint,

		"criticality", config.Criticality)



	return nil

}



// createCircuitBreaker creates a circuit breaker for a dependency.

func (dht *DependencyHealthTracker) createCircuitBreaker(config *DependencyConfig) {

	settings := gobreaker.Settings{

		Name:        config.Name,

		MaxRequests: config.CircuitBreakerConfig.MaxRequests,

		Interval:    config.CircuitBreakerConfig.Interval,

		Timeout:     config.CircuitBreakerConfig.Timeout,

		ReadyToTrip: func(counts gobreaker.Counts) bool {

			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)

			return counts.Requests >= 3 && failureRatio >= config.CircuitBreakerConfig.FailureThreshold

		},

		OnStateChange: func(name string, from, to gobreaker.State) {

			dht.logger.Info("Circuit breaker state changed",

				"dependency", name,

				"from", from.String(),

				"to", to.String())



			// Update metrics.

			stateValue := dht.circuitBreakerStateToFloat(to)

			dht.metrics.CircuitBreakerState.WithLabelValues(name).Set(stateValue)

		},

	}



	dht.circuitBreakers[config.Name] = gobreaker.NewCircuitBreaker(settings)

}



// circuitBreakerStateToFloat converts circuit breaker state to float for metrics.

func (dht *DependencyHealthTracker) circuitBreakerStateToFloat(state gobreaker.State) float64 {

	switch state {

	case gobreaker.StateClosed:

		return 0.0

	case gobreaker.StateOpen:

		return 1.0

	case gobreaker.StateHalfOpen:

		return 2.0

	default:

		return -1.0

	}

}



// CheckDependencyHealth performs a health check on a specific dependency.

func (dht *DependencyHealthTracker) CheckDependencyHealth(ctx context.Context, dependencyName string) (*DependencyHealth, error) {

	dht.mu.RLock()

	config, exists := dht.dependencies[dependencyName]

	dht.mu.RUnlock()



	if !exists {

		return nil, fmt.Errorf("dependency %s not found", dependencyName)

	}



	start := time.Now()



	// Create timeout context.

	checkCtx, cancel := context.WithTimeout(ctx, config.Timeout)

	defer cancel()



	var healthResult *DependencyHealth

	var err error



	// Execute health check with circuit breaker if available.

	if cb, exists := dht.circuitBreakers[dependencyName]; exists {

		_, err = cb.Execute(func() (interface{}, error) {

			healthResult, err = dht.performHealthCheck(checkCtx, config)

			return healthResult, err

		})

	} else {

		healthResult, err = dht.performHealthCheck(checkCtx, config)

	}



	if err != nil {

		healthResult = &DependencyHealth{

			Name:          dependencyName,

			Type:          config.Type,

			Status:        health.StatusUnhealthy,

			LastChecked:   time.Now(),

			ResponseTime:  time.Since(start),

			LastError:     err.Error(),

			LastErrorTime: time.Now(),

			Details:       make(map[string]interface{}),

		}



		// Update consecutive errors.

		dht.mu.RLock()

		if previous, exists := dht.healthResults[dependencyName]; exists {

			healthResult.ConsecutiveErrors = previous.ConsecutiveErrors + 1

			healthResult.ErrorCount = previous.ErrorCount + 1

		} else {

			healthResult.ConsecutiveErrors = 1

			healthResult.ErrorCount = 1

		}

		dht.mu.RUnlock()

	} else {

		// Reset consecutive errors on success.

		healthResult.ConsecutiveErrors = 0

	}



	// Update health result.

	dht.mu.Lock()

	dht.healthResults[dependencyName] = healthResult

	dht.mu.Unlock()



	// Record metrics.

	dht.recordHealthMetrics(healthResult, config)



	// Assess impact if unhealthy.

	if healthResult.Status != health.StatusHealthy {

		healthResult.ImpactAssessment = dht.impactAnalysis.assessImpact(config, healthResult)



		// Trigger recovery if needed.

		if config.CircuitBreakerConfig.Enabled {

			dht.recoveryTracker.triggerRecovery(dependencyName, config, healthResult)

		}

	}



	return healthResult, nil

}



// performHealthCheck performs the actual health check based on dependency type.

func (dht *DependencyHealthTracker) performHealthCheck(ctx context.Context, config *DependencyConfig) (*DependencyHealth, error) {

	switch config.Type {

	case DepTypeLLMAPI:

		return dht.checkLLMAPI(ctx, config)

	case DepTypeDatabase:

		return dht.checkDatabase(ctx, config)

	case DepTypeKubernetesAPI:

		return dht.checkKubernetesAPI(ctx, config)

	case DepTypeStorage:

		return dht.checkStorage(ctx, config)

	case DepTypeExternalAPI:

		return dht.checkExternalAPI(ctx, config)

	case DepTypeServiceMesh:

		return dht.checkServiceMesh(ctx, config)

	case DepTypeCache:

		return dht.checkCache(ctx, config)

	default:

		return dht.checkGenericHTTP(ctx, config)

	}

}



// checkLLMAPI performs health check for LLM APIs.

func (dht *DependencyHealthTracker) checkLLMAPI(ctx context.Context, config *DependencyConfig) (*DependencyHealth, error) {

	// Implement LLM API specific health check.

	start := time.Now()



	client := &http.Client{

		Timeout: config.Timeout,

	}



	url := config.Endpoint + "/health"

	if config.HealthCheckConfig.Path != "" {

		url = config.Endpoint + config.HealthCheckConfig.Path

	}



	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}



	// Add headers if configured.

	for key, value := range config.HealthCheckConfig.Headers {

		req.Header.Set(key, value)

	}



	resp, err := client.Do(req)

	if err != nil {

		return nil, fmt.Errorf("health check failed: %w", err)

	}

	defer resp.Body.Close()



	status := health.StatusUnhealthy

	message := fmt.Sprintf("HTTP %d", resp.StatusCode)



	// Check if status code is expected.

	if len(config.HealthCheckConfig.ExpectedStatus) == 0 {

		// Default to 200-299 range.

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {

			status = health.StatusHealthy

		}

	} else {

		for _, expectedStatus := range config.HealthCheckConfig.ExpectedStatus {

			if resp.StatusCode == expectedStatus {

				status = health.StatusHealthy

				break

			}

		}

	}



	return &DependencyHealth{

		Name:         config.Name,

		Type:         config.Type,

		Status:       status,

		LastChecked:  time.Now(),

		ResponseTime: time.Since(start),

		Details: map[string]interface{}{

			"status_code": resp.StatusCode,

			"endpoint":    url,

			"message":     message,

		},

	}, nil

}



// checkKubernetesAPI performs health check for Kubernetes API.

func (dht *DependencyHealthTracker) checkKubernetesAPI(ctx context.Context, config *DependencyConfig) (*DependencyHealth, error) {

	if dht.kubeClient == nil {

		return nil, fmt.Errorf("kubernetes client not available")

	}



	start := time.Now()



	// Try to get server version.

	version, err := dht.kubeClient.Discovery().ServerVersion()

	if err != nil {

		return nil, fmt.Errorf("kubernetes API check failed: %w", err)

	}



	return &DependencyHealth{

		Name:         config.Name,

		Type:         config.Type,

		Status:       health.StatusHealthy,

		LastChecked:  time.Now(),

		ResponseTime: time.Since(start),

		Details: map[string]interface{}{

			"server_version": version.GitVersion,

			"platform":       version.Platform,

		},

	}, nil

}



// checkDatabase performs health check for database connections.

func (dht *DependencyHealthTracker) checkDatabase(ctx context.Context, config *DependencyConfig) (*DependencyHealth, error) {

	// This would implement database-specific health checks.

	// For now, implement a generic approach.

	return dht.checkGenericHTTP(ctx, config)

}



// checkStorage performs health check for storage systems.

func (dht *DependencyHealthTracker) checkStorage(ctx context.Context, config *DependencyConfig) (*DependencyHealth, error) {

	// This would implement storage-specific health checks.

	// For now, implement a generic approach.

	return dht.checkGenericHTTP(ctx, config)

}



// checkExternalAPI performs health check for external APIs.

func (dht *DependencyHealthTracker) checkExternalAPI(ctx context.Context, config *DependencyConfig) (*DependencyHealth, error) {

	return dht.checkGenericHTTP(ctx, config)

}



// checkServiceMesh performs health check for service mesh components.

func (dht *DependencyHealthTracker) checkServiceMesh(ctx context.Context, config *DependencyConfig) (*DependencyHealth, error) {

	// Implement service mesh specific checks.

	return dht.checkGenericHTTP(ctx, config)

}



// checkCache performs health check for cache systems.

func (dht *DependencyHealthTracker) checkCache(ctx context.Context, config *DependencyConfig) (*DependencyHealth, error) {

	// Implement cache-specific checks.

	return dht.checkGenericHTTP(ctx, config)

}



// checkGenericHTTP performs generic HTTP health check.

func (dht *DependencyHealthTracker) checkGenericHTTP(ctx context.Context, config *DependencyConfig) (*DependencyHealth, error) {

	start := time.Now()



	client := &http.Client{

		Timeout: config.Timeout,

	}



	method := "GET"

	if config.HealthCheckConfig.Method != "" {

		method = config.HealthCheckConfig.Method

	}



	url := config.Endpoint

	if config.HealthCheckConfig.Path != "" {

		url = config.Endpoint + config.HealthCheckConfig.Path

	}



	req, err := http.NewRequestWithContext(ctx, method, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}



	// Add headers if configured.

	for key, value := range config.HealthCheckConfig.Headers {

		req.Header.Set(key, value)

	}



	resp, err := client.Do(req)

	if err != nil {

		return nil, fmt.Errorf("health check failed: %w", err)

	}

	defer resp.Body.Close()



	status := health.StatusUnhealthy

	message := fmt.Sprintf("HTTP %d", resp.StatusCode)



	// Check if status code is expected.

	if len(config.HealthCheckConfig.ExpectedStatus) == 0 {

		// Default to 200-299 range.

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {

			status = health.StatusHealthy

		}

	} else {

		for _, expectedStatus := range config.HealthCheckConfig.ExpectedStatus {

			if resp.StatusCode == expectedStatus {

				status = health.StatusHealthy

				break

			}

		}

	}



	return &DependencyHealth{

		Name:         config.Name,

		Type:         config.Type,

		Status:       status,

		LastChecked:  time.Now(),

		ResponseTime: time.Since(start),

		Details: map[string]interface{}{

			"status_code": resp.StatusCode,

			"endpoint":    url,

			"message":     message,

			"method":      method,

		},

	}, nil

}



// recordHealthMetrics records metrics for dependency health.

func (dht *DependencyHealthTracker) recordHealthMetrics(healthResult *DependencyHealth, config *DependencyConfig) {

	labels := []string{healthResult.Name, string(healthResult.Type), string(config.Criticality)}



	// Health status metric.

	statusValue := 0.0

	if healthResult.Status == health.StatusHealthy {

		statusValue = 1.0

	} else if healthResult.Status == health.StatusDegraded {

		statusValue = 0.5

	}

	dht.metrics.HealthStatus.WithLabelValues(labels...).Set(statusValue)



	// Response time metric.

	dht.metrics.ResponseTime.WithLabelValues(healthResult.Name, string(healthResult.Type)).Observe(healthResult.ResponseTime.Seconds())



	// Availability rate metric.

	dht.metrics.AvailabilityRate.WithLabelValues(healthResult.Name, string(healthResult.Type)).Set(healthResult.AvailabilityRate)



	// Error count if there was an error.

	if healthResult.LastError != "" {

		errorType := "unknown"

		if healthResult.Status == health.StatusUnhealthy {

			errorType = "unhealthy"

		}

		dht.metrics.ErrorCount.WithLabelValues(healthResult.Name, string(healthResult.Type), errorType).Inc()

	}

}



// registerDefaultDependencies registers default dependencies for the Nephoran Intent Operator.

func (dht *DependencyHealthTracker) registerDefaultDependencies() {

	// LLM Processor API.

	dht.RegisterDependency(&DependencyConfig{

		Name:          "llm-processor",

		Type:          DepTypeLLMAPI,

		Category:      CatCore,

		Criticality:   CriticalityEssential,

		Endpoint:      getEnv("LLM_PROCESSOR_URL", "http://llm-processor:8080"),

		Timeout:       15 * time.Second,

		CheckInterval: 30 * time.Second,

		HealthCheckConfig: HealthCheckConfig{

			Method:         "GET",

			Path:           "/healthz",

			ExpectedStatus: []int{200},

		},

		CircuitBreakerConfig: CircuitBreakerConfig{

			Enabled:          true,

			MaxRequests:      10,

			Interval:         30 * time.Second,

			Timeout:          60 * time.Second,

			FailureThreshold: 0.6,

		},

	})



	// RAG API - Smart endpoint detection.

	ragAPIURL := getEnv("RAG_API_URL", "http://rag-api:5001")

	ragHealthPath := "/health" // Always use /health for health checks



	// If the configured URL already has an endpoint path, extract the base for health checks.

	if strings.HasSuffix(ragAPIURL, "/process_intent") || strings.HasSuffix(ragAPIURL, "/process") {

		// Extract base URL and use it for health checks.

		if strings.HasSuffix(ragAPIURL, "/process_intent") {

			ragAPIURL = strings.TrimSuffix(ragAPIURL, "/process_intent")

		} else if strings.HasSuffix(ragAPIURL, "/process") {

			ragAPIURL = strings.TrimSuffix(ragAPIURL, "/process")

		}

	}



	dht.RegisterDependency(&DependencyConfig{

		Name:          "rag-api",

		Type:          DepTypeExternalAPI,

		Category:      CatCore,

		Criticality:   CriticalityImportant,

		Endpoint:      ragAPIURL,

		Timeout:       10 * time.Second,

		CheckInterval: 30 * time.Second,

		HealthCheckConfig: HealthCheckConfig{

			Method:         "GET",

			Path:           ragHealthPath,

			ExpectedStatus: []int{200},

		},

		CircuitBreakerConfig: CircuitBreakerConfig{

			Enabled:          true,

			MaxRequests:      10,

			Interval:         30 * time.Second,

			Timeout:          60 * time.Second,

			FailureThreshold: 0.6,

		},

	})



	// Weaviate Vector Database.

	dht.RegisterDependency(&DependencyConfig{

		Name:          "weaviate",

		Type:          DepTypeDatabase,

		Category:      CatInfrastructure,

		Criticality:   CriticalityImportant,

		Endpoint:      getEnv("WEAVIATE_URL", "http://weaviate:8080"),

		Timeout:       10 * time.Second,

		CheckInterval: 30 * time.Second,

		HealthCheckConfig: HealthCheckConfig{

			Method:         "GET",

			Path:           "/v1/.well-known/ready",

			ExpectedStatus: []int{200},

		},

		CircuitBreakerConfig: CircuitBreakerConfig{

			Enabled:          true,

			MaxRequests:      5,

			Interval:         30 * time.Second,

			Timeout:          120 * time.Second,

			FailureThreshold: 0.6,

		},

	})



	// Kubernetes API.

	dht.RegisterDependency(&DependencyConfig{

		Name:          "kubernetes-api",

		Type:          DepTypeKubernetesAPI,

		Category:      CatInfrastructure,

		Criticality:   CriticalityEssential,

		Endpoint:      "kubernetes-api",

		Timeout:       5 * time.Second,

		CheckInterval: 30 * time.Second,

		CircuitBreakerConfig: CircuitBreakerConfig{

			Enabled: false, // Don't circuit break Kubernetes API

		},

	})

}



// GetAllDependencyHealth returns health status for all dependencies.

func (dht *DependencyHealthTracker) GetAllDependencyHealth(ctx context.Context) map[string]*DependencyHealth {

	dht.mu.RLock()

	defer dht.mu.RUnlock()



	result := make(map[string]*DependencyHealth)

	for name, health := range dht.healthResults {

		// Return a copy to avoid race conditions.

		healthCopy := *health

		result[name] = &healthCopy

	}



	return result

}



// assessImpact assesses the impact of a dependency failure.

func (ia *ImpactAnalyzer) assessImpact(config *DependencyConfig, health *DependencyHealth) *ImpactAssessment {

	// Default impact assessment based on criticality.

	assessment := &ImpactAssessment{

		ServiceImpact:    ServiceImpactNone,

		AffectedFeatures: []string{},

		UserImpact:       UserImpactNone,

		BusinessImpact:   BusinessImpactNone,

		MitigationActive: false,

	}



	// Assess impact based on criticality and type.

	switch config.Criticality {

	case CriticalityEssential:

		assessment.ServiceImpact = ServiceImpactMajor

		assessment.UserImpact = UserImpactSevere

		assessment.BusinessImpact = BusinessImpactHigh

	case CriticalityImportant:

		assessment.ServiceImpact = ServiceImpactPartial

		assessment.UserImpact = UserImpactModerate

		assessment.BusinessImpact = BusinessImpactMedium

	case CriticalityNormal:

		assessment.ServiceImpact = ServiceImpactPartial

		assessment.UserImpact = UserImpactMinor

		assessment.BusinessImpact = BusinessImpactLow

	case CriticalityOptional:

		assessment.ServiceImpact = ServiceImpactNone

		assessment.UserImpact = UserImpactNone

		assessment.BusinessImpact = BusinessImpactNone

	}



	// Add affected features based on dependency type.

	switch config.Type {

	case DepTypeLLMAPI:

		assessment.AffectedFeatures = []string{"intent_processing", "natural_language_parsing"}

	case DepTypeDatabase:

		assessment.AffectedFeatures = []string{"data_storage", "query_processing"}

	case DepTypeKubernetesAPI:

		assessment.AffectedFeatures = []string{"resource_management", "deployment", "scaling"}

	}



	return assessment

}



// triggerRecovery triggers recovery process for a failed dependency.

func (rt *RecoveryTracker) triggerRecovery(dependencyName string, config *DependencyConfig, health *DependencyHealth) {

	rt.recoveryMu.Lock()

	defer rt.recoveryMu.Unlock()



	// Check if recovery is already in progress.

	if existing, exists := rt.recoveryMap[dependencyName]; exists && !existing.Success {

		// Update existing recovery state.

		existing.Attempts++

		existing.LastAttempt = time.Now()



		// Calculate backoff.

		existing.CurrentBackoff = rt.calculateBackoff(existing)



		rt.logger.Info("Recovery attempt updated",

			"dependency", dependencyName,

			"attempt", existing.Attempts,

			"backoff", existing.CurrentBackoff)



		return

	}



	// Start new recovery process.

	recovery := &RecoveryState{

		DependencyName:  dependencyName,

		StartTime:       time.Now(),

		Strategy:        RecoveryRetry, // Default strategy

		Attempts:        1,

		MaxAttempts:     5,

		BackoffStrategy: BackoffExponential,

		CurrentBackoff:  time.Second,

		LastAttempt:     time.Now(),

		Success:         false,

		Automated:       true,

	}



	rt.recoveryMap[dependencyName] = recovery



	rt.logger.Info("Recovery process started",

		"dependency", dependencyName,

		"strategy", recovery.Strategy)

}



// calculateBackoff calculates the next backoff duration.

func (rt *RecoveryTracker) calculateBackoff(state *RecoveryState) time.Duration {

	switch state.BackoffStrategy {

	case BackoffLinear:

		return time.Duration(state.Attempts) * time.Second

	case BackoffExponential:

		return time.Duration(1<<uint(state.Attempts)) * time.Second

	case BackoffFixed:

		return 5 * time.Second

	case BackoffJittered:

		base := time.Duration(1<<uint(state.Attempts)) * time.Second

		// Add jitter (simplified).

		return base + time.Duration(state.Attempts%3)*time.Millisecond*100

	default:

		return time.Second

	}

}



// getEnv gets environment variable with default value.

func getEnv(key, defaultValue string) string {

	if value := os.Getenv(key); value != "" {

		return value

	}

	return defaultValue

}

