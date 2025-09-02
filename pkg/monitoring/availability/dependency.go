package availability

import (
	
	"encoding/json"
"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DependencyType represents the type of dependency.

type DependencyType string

const (

	// DepTypeDatabase holds deptypedatabase value.

	DepTypeDatabase DependencyType = "database"

	// DepTypeExternalAPI holds deptypeexternalapi value.

	DepTypeExternalAPI DependencyType = "external_api"

	// DepTypeInternalService holds deptypeinternalservice value.

	DepTypeInternalService DependencyType = "internal_service"

	// DepTypeMessageQueue holds deptypemessagequeue value.

	DepTypeMessageQueue DependencyType = "message_queue"

	// DepTypeCache holds deptypecache value.

	DepTypeCache DependencyType = "cache"

	// DepTypeStorage holds deptypestorage value.

	DepTypeStorage DependencyType = "storage"

	// DepTypeLLMService holds deptypellmservice value.

	DepTypeLLMService DependencyType = "llm_service"

	// DepTypeK8sAPI holds deptypek8sapi value.

	DepTypeK8sAPI DependencyType = "kubernetes_api"
)

// DependencyStatus represents the health status of a dependency.

type DependencyStatus string

const (

	// DepStatusHealthy holds depstatushealthy value.

	DepStatusHealthy DependencyStatus = "healthy"

	// DepStatusDegraded holds depstatusdegraded value.

	DepStatusDegraded DependencyStatus = "degraded"

	// DepStatusUnhealthy holds depstatusunhealthy value.

	DepStatusUnhealthy DependencyStatus = "unhealthy"

	// DepStatusUnknown holds depstatusunknown value.

	DepStatusUnknown DependencyStatus = "unknown"

	// DepStatusCircuitOpen holds depstatuscircuitopen value.

	DepStatusCircuitOpen DependencyStatus = "circuit_open"
)

// FailureMode represents how a dependency failure impacts the system.

type FailureMode string

const (

	// FailureModeHardFail holds failuremodehardfail value.

	FailureModeHardFail FailureMode = "hard_fail" // Service cannot function

	// FailureModeSoftFail holds failuremodesoftfail value.

	FailureModeSoftFail FailureMode = "soft_fail" // Service degrades gracefully

	// FailureModeCircuitBreak holds failuremodecircuitbreak value.

	FailureModeCircuitBreak FailureMode = "circuit_break" // Circuit breaker protects

	// FailureModeRetry holds failuremoderetry value.

	FailureModeRetry FailureMode = "retry" // Automatic retry logic

)

// Dependency represents a service dependency.

type Dependency struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Type DependencyType `json:"type"`

	ServiceName string `json:"service_name"`

	Namespace string `json:"namespace"`

	Endpoint string `json:"endpoint"`

	BusinessImpact BusinessImpact `json:"business_impact"`

	FailureMode FailureMode `json:"failure_mode"`

	CircuitBreaker CircuitBreakerConfig `json:"circuit_breaker"`

	Dependencies []string `json:"dependencies"` // IDs of dependencies this depends on

	Dependents []string `json:"dependents"` // IDs of services that depend on this

	HealthChecks []DependencyHealthCheck `json:"health_checks"`

	SLARequirements SLARequirements `json:"sla_requirements"`

	Tags map[string]string `json:"tags"`
}

// CircuitBreakerConfig holds circuit breaker configuration.

type CircuitBreakerConfig struct {
	Enabled bool `json:"enabled"`

	FailureThreshold int `json:"failure_threshold"`

	RecoveryTimeout time.Duration `json:"recovery_timeout"`

	HalfOpenMaxCalls int `json:"half_open_max_calls"`

	MinRequestsThreshold int `json:"min_requests_threshold"`

	ConsecutiveSuccesses int `json:"consecutive_successes"`
}

// DependencyHealthCheck defines how to check dependency health.

type DependencyHealthCheck struct {
	Type string `json:"type"` // prometheus, http, tcp, dns

	Target string `json:"target"`

	Timeout time.Duration `json:"timeout"`

	Interval time.Duration `json:"interval"`

	FailureThreshold int `json:"failure_threshold"`

	SuccessThreshold int `json:"success_threshold"`

	Query string `json:"query,omitempty"` // For Prometheus queries

	ExpectedStatus int `json:"expected_status,omitempty"` // For HTTP checks
}

// SLARequirements defines SLA requirements for a dependency.

type SLARequirements struct {
	Availability float64 `json:"availability"` // 0.0 to 1.0

	ResponseTime time.Duration `json:"response_time"` // P95 response time requirement

	ErrorRate float64 `json:"error_rate"` // Maximum error rate (0.0 to 1.0)

	MTTR time.Duration `json:"mttr"` // Mean Time To Recovery

	MTBF time.Duration `json:"mtbf"` // Mean Time Between Failures
}

// DependencyHealth represents current health status of a dependency.

type DependencyHealth struct {
	DependencyID string `json:"dependency_id"`

	Status DependencyStatus `json:"status"`

	LastUpdate time.Time `json:"last_update"`

	ResponseTime time.Duration `json:"response_time"`

	ErrorRate float64 `json:"error_rate"`

	Availability float64 `json:"availability"`

	CircuitBreakerState string `json:"circuit_breaker_state"`

	FailureCount int `json:"failure_count"`

	LastFailure time.Time `json:"last_failure"`

	RecoveryTime time.Duration `json:"recovery_time"`

	HealthCheckResults []HealthCheckResult `json:"health_check_results"`
}

// HealthCheckResult represents the result of a health check.

type HealthCheckResult struct {
	CheckType string `json:"check_type"`

	Status string `json:"status"`

	ResponseTime time.Duration `json:"response_time"`

	Error string `json:"error,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// DependencyGraph represents the dependency relationships.

type DependencyGraph struct {
	Dependencies map[string]*Dependency `json:"dependencies"`

	AdjacencyMap map[string][]string `json:"adjacency_map"` // dependency_id -> [dependent_ids]

	ReverseMap map[string][]string `json:"reverse_map"` // dependent_id -> [dependency_ids]
}

// CascadeFailureAnalysis represents analysis of potential cascade failures.

type CascadeFailureAnalysis struct {
	FailedDependency string `json:"failed_dependency"`

	ImpactedServices []string `json:"impacted_services"`

	BusinessImpact float64 `json:"business_impact"`

	CascadeDepth int `json:"cascade_depth"`

	RecoveryPath []string `json:"recovery_path"`

	EstimatedRecoveryTime time.Duration `json:"estimated_recovery_time"`

	Timestamp time.Time `json:"timestamp"`
}

// DependencyTracker tracks and monitors service dependencies.

type DependencyTracker struct {
	// Configuration.

	config *DependencyTrackerConfig

	// Dependency management.

	graph *DependencyGraph

	graphMutex sync.RWMutex

	healthStatus map[string]*DependencyHealth

	healthMutex sync.RWMutex

	// Clients.

	kubeClient client.Client

	kubeClientset kubernetes.Interface

	promClient v1.API

	// Circuit breaker tracking.

	circuitStates map[string]*CircuitBreakerState

	stateMutex sync.RWMutex

	// Cascade failure analysis.

	cascadeHistory []CascadeFailureAnalysis

	cascadeMutex sync.RWMutex

	// Control.

	ctx context.Context

	cancel context.CancelFunc

	stopCh chan struct{}

	// Observability.

	tracer trace.Tracer
}

// CircuitBreakerState represents the current state of a circuit breaker.

type CircuitBreakerState struct {
	DependencyID string `json:"dependency_id"`

	State string `json:"state"` // closed, open, half_open

	FailureCount int `json:"failure_count"`

	LastFailureTime time.Time `json:"last_failure_time"`

	NextRetryTime time.Time `json:"next_retry_time"`

	HalfOpenCalls int `json:"half_open_calls"`

	ConsecutiveSuccesses int `json:"consecutive_successes"`
}

// DependencyTrackerConfig holds configuration for the dependency tracker.

type DependencyTrackerConfig struct {
	MonitoringInterval time.Duration `json:"monitoring_interval"`

	HealthCheckTimeout time.Duration `json:"health_check_timeout"`

	CascadeAnalysisDepth int `json:"cascade_analysis_depth"`

	RetentionPeriod time.Duration `json:"retention_period"`

	EnableCircuitBreaker bool `json:"enable_circuit_breaker"`

	// Service mesh integration.

	ServiceMeshEnabled bool `json:"service_mesh_enabled"`

	ServiceMeshType string `json:"service_mesh_type"` // istio, linkerd, consul

	// External monitoring.

	PrometheusEnabled bool `json:"prometheus_enabled"`

	JaegerEnabled bool `json:"jaeger_enabled"`

	// Alerting.

	AlertingEnabled bool `json:"alerting_enabled"`

	CriticalDependencies []string `json:"critical_dependencies"`
}

// NewDependencyTracker creates a new dependency tracker.

func NewDependencyTracker(
	config *DependencyTrackerConfig,

	kubeClient client.Client,

	kubeClientset kubernetes.Interface,

	promClient api.Client,
) (*DependencyTracker, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	var promAPI v1.API

	if promClient != nil && config.PrometheusEnabled {
		promAPI = v1.NewAPI(promClient)
	}

	tracker := &DependencyTracker{
		config: config,

		kubeClient: kubeClient,

		kubeClientset: kubeClientset,

		promClient: promAPI,

		graph: &DependencyGraph{
			Dependencies: make(map[string]*Dependency),

			AdjacencyMap: make(map[string][]string),

			ReverseMap: make(map[string][]string),
		},

		healthStatus: make(map[string]*DependencyHealth),

		circuitStates: make(map[string]*CircuitBreakerState),

		cascadeHistory: make([]CascadeFailureAnalysis, 0, 1000),

		ctx: ctx,

		cancel: cancel,

		stopCh: make(chan struct{}),

		tracer: otel.Tracer("dependency-tracker"),
	}

	return tracker, nil
}

// Start begins dependency tracking.

func (dt *DependencyTracker) Start() error {
	ctx, span := dt.tracer.Start(dt.ctx, "dependency-tracker-start")

	defer span.End()

	span.AddEvent("Starting dependency tracker")

	// Start monitoring goroutines.

	go dt.runHealthMonitoring(ctx)

	go dt.runCircuitBreakerManagement(ctx)

	go dt.runCascadeAnalysis(ctx)

	go dt.runServiceMeshIntegration(ctx)

	go dt.runCleanup(ctx)

	return nil
}

// Stop stops dependency tracking.

func (dt *DependencyTracker) Stop() error {
	dt.cancel()

	close(dt.stopCh)

	return nil
}

// AddDependency adds a dependency to the graph.

func (dt *DependencyTracker) AddDependency(dep *Dependency) error {
	if dep == nil || dep.ID == "" {
		return fmt.Errorf("invalid dependency")
	}

	dt.graphMutex.Lock()

	defer dt.graphMutex.Unlock()

	// Add dependency.

	dt.graph.Dependencies[dep.ID] = dep

	// Update adjacency maps.

	dt.updateAdjacencyMaps(dep)

	// Initialize health status.

	dt.healthMutex.Lock()

	dt.healthStatus[dep.ID] = &DependencyHealth{
		DependencyID: dep.ID,

		Status: DepStatusUnknown,

		LastUpdate: time.Now(),

		HealthCheckResults: make([]HealthCheckResult, 0),
	}

	dt.healthMutex.Unlock()

	// Initialize circuit breaker if enabled.

	if dt.config.EnableCircuitBreaker && dep.CircuitBreaker.Enabled {

		dt.stateMutex.Lock()

		dt.circuitStates[dep.ID] = &CircuitBreakerState{
			DependencyID: dep.ID,

			State: "closed",
		}

		dt.stateMutex.Unlock()

	}

	return nil
}

// updateAdjacencyMaps updates the adjacency maps for dependency relationships.

func (dt *DependencyTracker) updateAdjacencyMaps(dep *Dependency) {
	// Initialize if not exists.

	if dt.graph.AdjacencyMap[dep.ID] == nil {
		dt.graph.AdjacencyMap[dep.ID] = make([]string, 0)
	}

	if dt.graph.ReverseMap[dep.ID] == nil {
		dt.graph.ReverseMap[dep.ID] = make([]string, 0)
	}

	// Update forward mapping (this dependency -> its dependents).

	dt.graph.AdjacencyMap[dep.ID] = dep.Dependents

	// Update reverse mapping (dependencies -> this dependency).

	dt.graph.ReverseMap[dep.ID] = dep.Dependencies

	// Update reverse mappings for dependencies.

	for _, depID := range dep.Dependencies {

		if dt.graph.AdjacencyMap[depID] == nil {
			dt.graph.AdjacencyMap[depID] = make([]string, 0)
		}

		dt.graph.AdjacencyMap[depID] = append(dt.graph.AdjacencyMap[depID], dep.ID)

	}
}

// runHealthMonitoring runs continuous health monitoring.

func (dt *DependencyTracker) runHealthMonitoring(ctx context.Context) {
	ticker := time.NewTicker(dt.config.MonitoringInterval)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			dt.performHealthChecks(ctx)

		}
	}
}

// performHealthChecks performs health checks for all dependencies.

func (dt *DependencyTracker) performHealthChecks(ctx context.Context) {
	ctx, span := dt.tracer.Start(ctx, "perform-health-checks")

	defer span.End()

	dt.graphMutex.RLock()

	dependencies := make([]*Dependency, 0, len(dt.graph.Dependencies))

	for _, dep := range dt.graph.Dependencies {
		dependencies = append(dependencies, dep)
	}

	dt.graphMutex.RUnlock()

	// Perform checks concurrently.

	var wg sync.WaitGroup

	for _, dep := range dependencies {

		wg.Add(1)

		go func(dependency *Dependency) {
			defer wg.Done()

			dt.checkDependencyHealth(ctx, dependency)
		}(dep)

	}

	wg.Wait()

	span.AddEvent("Health checks completed",

		trace.WithAttributes(attribute.Int("dependencies_checked", len(dependencies))))
}

// checkDependencyHealth checks the health of a single dependency.

func (dt *DependencyTracker) checkDependencyHealth(ctx context.Context, dep *Dependency) {
	ctx, span := dt.tracer.Start(ctx, "check-dependency-health",

		trace.WithAttributes(

			attribute.String("dependency_id", dep.ID),

			attribute.String("dependency_type", string(dep.Type)),
		),
	)

	defer span.End()

	health := &DependencyHealth{
		DependencyID: dep.ID,

		LastUpdate: time.Now(),

		HealthCheckResults: make([]HealthCheckResult, 0),
	}

	// Perform each health check.

	overallHealthy := true

	var totalResponseTime time.Duration

	var errorCount int

	for _, healthCheck := range dep.HealthChecks {

		result := dt.performSingleHealthCheck(ctx, dep, &healthCheck)

		health.HealthCheckResults = append(health.HealthCheckResults, result)

		if result.Status != "healthy" {

			overallHealthy = false

			errorCount++

		}

		totalResponseTime += result.ResponseTime

	}

	// Calculate overall metrics.

	if len(dep.HealthChecks) > 0 {

		health.ResponseTime = totalResponseTime / time.Duration(len(dep.HealthChecks))

		health.ErrorRate = float64(errorCount) / float64(len(dep.HealthChecks))

	}

	// Determine overall status.

	if overallHealthy {
		health.Status = DepStatusHealthy
	} else if health.ErrorRate < 0.5 {
		health.Status = DepStatusDegraded
	} else {

		health.Status = DepStatusUnhealthy

		health.LastFailure = time.Now()

		health.FailureCount++

	}

	// Check circuit breaker state.

	if dt.config.EnableCircuitBreaker {

		health.CircuitBreakerState = dt.getCircuitBreakerState(dep.ID)

		if health.CircuitBreakerState == "open" {
			health.Status = DepStatusCircuitOpen
		}

	}

	// Calculate availability from historical data.

	health.Availability = dt.calculateAvailability(dep.ID, time.Hour)

	// Update health status.

	dt.healthMutex.Lock()

	dt.healthStatus[dep.ID] = health

	dt.healthMutex.Unlock()

	// Update circuit breaker if dependency failed.

	if health.Status == DepStatusUnhealthy {
		dt.updateCircuitBreaker(dep.ID, false)
	} else if health.Status == DepStatusHealthy {
		dt.updateCircuitBreaker(dep.ID, true)
	}

	span.AddEvent("Health check completed",

		trace.WithAttributes(

			attribute.String("status", string(health.Status)),

			attribute.Float64("error_rate", health.ErrorRate),

			attribute.Float64("availability", health.Availability),
		),
	)
}

// performSingleHealthCheck performs a single health check.

func (dt *DependencyTracker) performSingleHealthCheck(ctx context.Context, dep *Dependency, healthCheck *DependencyHealthCheck) HealthCheckResult {
	startTime := time.Now()

	result := HealthCheckResult{
		CheckType: healthCheck.Type,

		Timestamp: startTime,

		Metadata: make(map[string]interface{}),
	}

	// Create timeout context.

	checkCtx, cancel := context.WithTimeout(ctx, healthCheck.Timeout)

	defer cancel()

	switch healthCheck.Type {

	case "prometheus":

		result = dt.performPrometheusCheck(checkCtx, dep, healthCheck)

	case "http":

		result = dt.performHTTPCheck(checkCtx, dep, healthCheck)

	case "tcp":

		result = dt.performTCPCheck(checkCtx, dep, healthCheck)

	case "dns":

		result = dt.performDNSCheck(checkCtx, dep, healthCheck)

	case "kubernetes":

		result = dt.performKubernetesCheck(checkCtx, dep, healthCheck)

	default:

		result.Status = "error"

		result.Error = fmt.Sprintf("unknown health check type: %s", healthCheck.Type)

	}

	result.ResponseTime = time.Since(startTime)

	return result
}

// performPrometheusCheck performs a Prometheus-based health check.

func (dt *DependencyTracker) performPrometheusCheck(ctx context.Context, dep *Dependency, healthCheck *DependencyHealthCheck) HealthCheckResult {
	result := HealthCheckResult{
		CheckType: "prometheus",

		Timestamp: time.Now(),

		Status: "healthy",
	}

	if dt.promClient == nil {

		result.Status = "error"

		result.Error = "prometheus client not available"

		return result

	}

	// Execute Prometheus query.

	promResult, warnings, err := dt.promClient.Query(ctx, healthCheck.Query, time.Now())
	if err != nil {

		result.Status = "error"

		result.Error = fmt.Sprintf("prometheus query failed: %v", err)

		return result

	}

	if len(warnings) > 0 {
		result.Metadata["warnings"] = warnings
	}

	// Evaluate result based on query type.

	switch promResult.Type() {

	case model.ValVector:

		vector := promResult.(model.Vector)

		if len(vector) == 0 {

			result.Status = "unhealthy"

			result.Error = "no metrics found"

		} else {
			result.Metadata["value"] = float64(vector[0].Value)

			// You could add custom evaluation logic here based on the metric value.
		}

	case model.ValScalar:

		scalar := promResult.(*model.Scalar)

		result.Metadata["value"] = float64(scalar.Value)

	default:

		result.Status = "error"

		result.Error = "unexpected prometheus result type"

	}

	return result
}

// performHTTPCheck performs an HTTP-based health check.

func (dt *DependencyTracker) performHTTPCheck(ctx context.Context, dep *Dependency, healthCheck *DependencyHealthCheck) HealthCheckResult {
	result := HealthCheckResult{
		CheckType: "http",

		Timestamp: time.Now(),

		Status: "healthy",
	}

	req, err := http.NewRequestWithContext(ctx, "GET", healthCheck.Target, http.NoBody)
	if err != nil {

		result.Status = "error"

		result.Error = fmt.Sprintf("failed to create request: %v", err)

		return result

	}

	client := &http.Client{
		Timeout: healthCheck.Timeout,
	}

	resp, err := client.Do(req)
	if err != nil {

		result.Status = "error"

		result.Error = fmt.Sprintf("request failed: %v", err)

		return result

	}

	defer resp.Body.Close()

	result.Metadata["http_status"] = resp.StatusCode

	if healthCheck.ExpectedStatus != 0 && resp.StatusCode != healthCheck.ExpectedStatus {

		result.Status = "unhealthy"

		result.Error = fmt.Sprintf("expected status %d, got %d", healthCheck.ExpectedStatus, resp.StatusCode)

	} else if resp.StatusCode >= 400 {

		result.Status = "unhealthy"

		result.Error = fmt.Sprintf("HTTP error status: %d", resp.StatusCode)

	}

	return result
}

// performTCPCheck performs a TCP connection check.

func (dt *DependencyTracker) performTCPCheck(ctx context.Context, dep *Dependency, healthCheck *DependencyHealthCheck) HealthCheckResult {
	result := HealthCheckResult{
		CheckType: "tcp",

		Timestamp: time.Now(),

		Status: "healthy",
	}

	// For TCP checks, we just try to establish a connection.

	conn, err := net.DialTimeout("tcp", healthCheck.Target, healthCheck.Timeout)
	if err != nil {

		result.Status = "unhealthy"

		result.Error = fmt.Sprintf("TCP connection failed: %v", err)

		return result

	}

	defer conn.Close()

	return result
}

// performDNSCheck performs a DNS resolution check.

func (dt *DependencyTracker) performDNSCheck(ctx context.Context, dep *Dependency, healthCheck *DependencyHealthCheck) HealthCheckResult {
	result := HealthCheckResult{
		CheckType: "dns",

		Timestamp: time.Now(),

		Status: "healthy",
	}

	resolver := &net.Resolver{}

	ips, err := resolver.LookupIPAddr(ctx, healthCheck.Target)
	if err != nil {

		result.Status = "unhealthy"

		result.Error = fmt.Sprintf("DNS lookup failed: %v", err)

		return result

	}

	if len(ips) == 0 {

		result.Status = "unhealthy"

		result.Error = "no IP addresses found"

	} else {
		result.Metadata["resolved_ips"] = len(ips)
	}

	return result
}

// performKubernetesCheck performs a Kubernetes API check.

func (dt *DependencyTracker) performKubernetesCheck(ctx context.Context, dep *Dependency, healthCheck *DependencyHealthCheck) HealthCheckResult {
	result := HealthCheckResult{
		CheckType: "kubernetes",

		Timestamp: time.Now(),

		Status: "healthy",
	}

	if dt.kubeClientset == nil {

		result.Status = "error"

		result.Error = "kubernetes client not available"

		return result

	}

	// Check if service/pods are running.

	pods, err := dt.kubeClientset.CoreV1().Pods(dep.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", dep.ServiceName),
	})
	if err != nil {

		result.Status = "error"

		result.Error = fmt.Sprintf("failed to list pods: %v", err)

		return result

	}

	runningPods := 0

	for _, pod := range pods.Items {
		if pod.Status.Phase == "Running" {
			runningPods++
		}
	}

	result.Metadata["total_pods"] = len(pods.Items)

	result.Metadata["running_pods"] = runningPods

	if runningPods == 0 {

		result.Status = "unhealthy"

		result.Error = "no running pods found"

	}

	return result
}

// calculateAvailability calculates availability for a dependency over a time period.

func (dt *DependencyTracker) calculateAvailability(dependencyID string, period time.Duration) float64 {
	dt.healthMutex.RLock()

	defer dt.healthMutex.RUnlock()

	// This is a simplified calculation.

	// In practice, you'd query historical health data.

	currentHealth, exists := dt.healthStatus[dependencyID]

	if !exists {
		return 0.0
	}

	switch currentHealth.Status {

	case DepStatusHealthy:

		return 1.0

	case DepStatusDegraded:

		return 0.8

	case DepStatusCircuitOpen:

		return 0.5

	default:

		return 0.0

	}
}

// getCircuitBreakerState gets the current circuit breaker state.

func (dt *DependencyTracker) getCircuitBreakerState(dependencyID string) string {
	dt.stateMutex.RLock()

	defer dt.stateMutex.RUnlock()

	if state, exists := dt.circuitStates[dependencyID]; exists {
		return state.State
	}

	return "closed" // Default state
}

// updateCircuitBreaker updates the circuit breaker state based on health check results.

func (dt *DependencyTracker) updateCircuitBreaker(dependencyID string, success bool) {
	if !dt.config.EnableCircuitBreaker {
		return
	}

	dt.stateMutex.Lock()

	defer dt.stateMutex.Unlock()

	state, exists := dt.circuitStates[dependencyID]

	if !exists {
		return
	}

	dt.graphMutex.RLock()

	dep, depExists := dt.graph.Dependencies[dependencyID]

	dt.graphMutex.RUnlock()

	if !depExists || !dep.CircuitBreaker.Enabled {
		return
	}

	now := time.Now()

	switch state.State {

	case "closed":

		if success {
			state.FailureCount = 0
		} else {

			state.FailureCount++

			state.LastFailureTime = now

			if state.FailureCount >= dep.CircuitBreaker.FailureThreshold {

				state.State = "open"

				state.NextRetryTime = now.Add(dep.CircuitBreaker.RecoveryTimeout)

			}

		}

	case "open":

		if now.After(state.NextRetryTime) {

			state.State = "half_open"

			state.HalfOpenCalls = 0

			state.ConsecutiveSuccesses = 0

		}

	case "half_open":

		state.HalfOpenCalls++

		if success {

			state.ConsecutiveSuccesses++

			if state.ConsecutiveSuccesses >= dep.CircuitBreaker.ConsecutiveSuccesses {

				state.State = "closed"

				state.FailureCount = 0

			}

		} else {

			state.State = "open"

			state.NextRetryTime = now.Add(dep.CircuitBreaker.RecoveryTimeout)

			state.FailureCount++

			state.LastFailureTime = now

		}

		if state.HalfOpenCalls >= dep.CircuitBreaker.HalfOpenMaxCalls {
			if state.ConsecutiveSuccesses < dep.CircuitBreaker.ConsecutiveSuccesses {

				state.State = "open"

				state.NextRetryTime = now.Add(dep.CircuitBreaker.RecoveryTimeout)

			}
		}

	}
}

// runCircuitBreakerManagement manages circuit breaker states.

func (dt *DependencyTracker) runCircuitBreakerManagement(ctx context.Context) {
	if !dt.config.EnableCircuitBreaker {
		return
	}

	ticker := time.NewTicker(time.Minute)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			dt.manageCircuitBreakers(ctx)

		}
	}
}

// manageCircuitBreakers performs circuit breaker management tasks.

func (dt *DependencyTracker) manageCircuitBreakers(ctx context.Context) {
	_, span := dt.tracer.Start(ctx, "manage-circuit-breakers")

	defer span.End()

	dt.stateMutex.RLock()

	states := make(map[string]*CircuitBreakerState)

	for k, v := range dt.circuitStates {
		states[k] = v
	}

	dt.stateMutex.RUnlock()

	now := time.Now()

	stateChanges := 0

	for depID, state := range states {
		if state.State == "open" && now.After(state.NextRetryTime) {

			dt.stateMutex.Lock()

			dt.circuitStates[depID].State = "half_open"

			dt.circuitStates[depID].HalfOpenCalls = 0

			dt.circuitStates[depID].ConsecutiveSuccesses = 0

			dt.stateMutex.Unlock()

			stateChanges++

		}
	}

	span.AddEvent("Circuit breaker management completed",

		trace.WithAttributes(attribute.Int("state_changes", stateChanges)))
}

// runCascadeAnalysis performs cascade failure analysis.

func (dt *DependencyTracker) runCascadeAnalysis(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Analyze every 5 minutes

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			dt.performCascadeAnalysis(ctx)

		}
	}
}

// performCascadeAnalysis analyzes potential cascade failures.

func (dt *DependencyTracker) performCascadeAnalysis(ctx context.Context) {
	_, span := dt.tracer.Start(ctx, "perform-cascade-analysis")

	defer span.End()

	// Find unhealthy dependencies.

	dt.healthMutex.RLock()

	unhealthyDeps := make([]string, 0)

	for depID, health := range dt.healthStatus {
		if health.Status == DepStatusUnhealthy || health.Status == DepStatusCircuitOpen {
			unhealthyDeps = append(unhealthyDeps, depID)
		}
	}

	dt.healthMutex.RUnlock()

	// Analyze cascade impact for each unhealthy dependency.

	for _, depID := range unhealthyDeps {

		analysis := dt.analyzeCascadeImpact(depID)

		if analysis != nil {

			dt.cascadeMutex.Lock()

			dt.cascadeHistory = append(dt.cascadeHistory, *analysis)

			dt.cascadeMutex.Unlock()

		}

	}

	span.AddEvent("Cascade analysis completed",

		trace.WithAttributes(attribute.Int("unhealthy_dependencies", len(unhealthyDeps))))
}

// analyzeCascadeImpact analyzes the cascade impact of a failed dependency.

func (dt *DependencyTracker) analyzeCascadeImpact(failedDep string) *CascadeFailureAnalysis {
	dt.graphMutex.RLock()

	defer dt.graphMutex.RUnlock()

	analysis := &CascadeFailureAnalysis{
		FailedDependency: failedDep,

		Timestamp: time.Now(),

		ImpactedServices: make([]string, 0),

		RecoveryPath: make([]string, 0),
	}

	// Find all services impacted by this failure using BFS.

	visited := make(map[string]bool)

	queue := []string{failedDep}

	depth := 0

	for len(queue) > 0 && depth < dt.config.CascadeAnalysisDepth {

		nextQueue := make([]string, 0)

		for _, current := range queue {

			if visited[current] {
				continue
			}

			visited[current] = true

			// Add dependents to impact list.

			if dependents, exists := dt.graph.AdjacencyMap[current]; exists {
				for _, dependent := range dependents {
					if !visited[dependent] {

						analysis.ImpactedServices = append(analysis.ImpactedServices, dependent)

						nextQueue = append(nextQueue, dependent)

						// Calculate business impact.

						if dep, exists := dt.graph.Dependencies[dependent]; exists {
							analysis.BusinessImpact += float64(dep.BusinessImpact)
						}

					}
				}
			}

		}

		queue = nextQueue

		depth++

	}

	analysis.CascadeDepth = depth

	// Estimate recovery time based on dependency SLA requirements.

	if dep, exists := dt.graph.Dependencies[failedDep]; exists {

		analysis.EstimatedRecoveryTime = dep.SLARequirements.MTTR

		analysis.RecoveryPath = append(analysis.RecoveryPath, failedDep)

		// Add critical path dependencies to recovery path.

		for _, depID := range dep.Dependencies {
			if depHealth, exists := dt.healthStatus[depID]; exists {
				if depHealth.Status != DepStatusHealthy {
					analysis.RecoveryPath = append(analysis.RecoveryPath, depID)
				}
			}
		}

	}

	return analysis
}

// runServiceMeshIntegration integrates with service mesh for dependency discovery.

func (dt *DependencyTracker) runServiceMeshIntegration(ctx context.Context) {
	if !dt.config.ServiceMeshEnabled {
		return
	}

	ticker := time.NewTicker(10 * time.Minute) // Discover every 10 minutes

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			dt.discoverServiceMeshDependencies(ctx)

		}
	}
}

// discoverServiceMeshDependencies discovers dependencies from service mesh.

func (dt *DependencyTracker) discoverServiceMeshDependencies(ctx context.Context) {
	ctx, span := dt.tracer.Start(ctx, "discover-service-mesh-dependencies")

	defer span.End()

	// Implementation would depend on the service mesh type.

	switch dt.config.ServiceMeshType {

	case "istio":

		dt.discoverIstioDependencies(ctx)

	case "linkerd":

		dt.discoverLinkerdDependencies(ctx)

	case "consul":

		dt.discoverConsulDependencies(ctx)

	}
}

// discoverIstioDependencies discovers dependencies from Istio.

func (dt *DependencyTracker) discoverIstioDependencies(ctx context.Context) {
	// This would query Istio's telemetry data to discover service dependencies.

	// Implementation would use Istio's APIs or Prometheus metrics.

	// For now, this is a placeholder.
}

// discoverLinkerdDependencies discovers dependencies from Linkerd.

func (dt *DependencyTracker) discoverLinkerdDependencies(ctx context.Context) {
	// This would query Linkerd's tap API or metrics to discover dependencies.

	// Implementation would use Linkerd's APIs.

	// For now, this is a placeholder.
}

// discoverConsulDependencies discovers dependencies from Consul Connect.

func (dt *DependencyTracker) discoverConsulDependencies(ctx context.Context) {
	// This would query Consul's service registry and intentions.

	// Implementation would use Consul's APIs.

	// For now, this is a placeholder.
}

// runCleanup performs cleanup of old data.

func (dt *DependencyTracker) runCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			dt.performCleanup(ctx)

		}
	}
}

// performCleanup cleans up old data.

func (dt *DependencyTracker) performCleanup(ctx context.Context) {
	_, span := dt.tracer.Start(ctx, "perform-cleanup")

	defer span.End()

	cutoff := time.Now().Add(-dt.config.RetentionPeriod)

	// Clean cascade history.

	dt.cascadeMutex.Lock()

	validAnalyses := make([]CascadeFailureAnalysis, 0)

	for _, analysis := range dt.cascadeHistory {
		if analysis.Timestamp.After(cutoff) {
			validAnalyses = append(validAnalyses, analysis)
		}
	}

	removed := len(dt.cascadeHistory) - len(validAnalyses)

	dt.cascadeHistory = validAnalyses

	dt.cascadeMutex.Unlock()

	span.AddEvent("Cleanup completed",

		trace.WithAttributes(attribute.Int("cascade_analyses_removed", removed)))
}

// GetDependencyHealth returns current health status for a dependency.

func (dt *DependencyTracker) GetDependencyHealth(dependencyID string) (*DependencyHealth, bool) {
	dt.healthMutex.RLock()

	defer dt.healthMutex.RUnlock()

	health, exists := dt.healthStatus[dependencyID]

	return health, exists
}

// GetAllDependencyHealth returns health status for all dependencies.

func (dt *DependencyTracker) GetAllDependencyHealth() map[string]*DependencyHealth {
	dt.healthMutex.RLock()

	defer dt.healthMutex.RUnlock()

	result := make(map[string]*DependencyHealth)

	for k, v := range dt.healthStatus {
		result[k] = v
	}

	return result
}

// GetCascadeAnalysis returns recent cascade failure analyses.

func (dt *DependencyTracker) GetCascadeAnalysis(since time.Time) []CascadeFailureAnalysis {
	dt.cascadeMutex.RLock()

	defer dt.cascadeMutex.RUnlock()

	result := make([]CascadeFailureAnalysis, 0)

	for _, analysis := range dt.cascadeHistory {
		if analysis.Timestamp.After(since) {
			result = append(result, analysis)
		}
	}

	return result
}

// GetCircuitBreakerStates returns current circuit breaker states.

func (dt *DependencyTracker) GetCircuitBreakerStates() map[string]*CircuitBreakerState {
	dt.stateMutex.RLock()

	defer dt.stateMutex.RUnlock()

	result := make(map[string]*CircuitBreakerState)

	for k, v := range dt.circuitStates {
		result[k] = v
	}

	return result
}

// GetAvailabilityMetrics converts dependency health to availability metrics.

func (dt *DependencyTracker) GetAvailabilityMetrics(dependencyID string) (*AvailabilityMetric, error) {
	health, exists := dt.GetDependencyHealth(dependencyID)

	if !exists {
		return nil, fmt.Errorf("dependency %s not found", dependencyID)
	}

	dt.graphMutex.RLock()

	dep, depExists := dt.graph.Dependencies[dependencyID]

	dt.graphMutex.RUnlock()

	if !depExists {
		return nil, fmt.Errorf("dependency configuration for %s not found", dependencyID)
	}

	// Convert dependency status to availability status.

	var status HealthStatus

	switch health.Status {

	case DepStatusHealthy:

		status = HealthHealthy

	case DepStatusDegraded:

		status = HealthDegraded

	case DepStatusUnhealthy, DepStatusCircuitOpen:

		status = HealthUnhealthy

	default:

		status = HealthUnknown

	}

	// Determine layer based on dependency type.

	var layer ServiceLayer

	switch dep.Type {

	case DepTypeDatabase, DepTypeCache, DepTypeStorage:

		layer = LayerStorage

	case DepTypeExternalAPI, DepTypeLLMService:

		layer = LayerExternal

	case DepTypeInternalService, DepTypeK8sAPI:

		layer = LayerProcessor

	default:

		layer = LayerAPI

	}

	metric := &AvailabilityMetric{
		Timestamp: health.LastUpdate,

		Dimension: DimensionComponent, // Dependencies are component-level

		EntityID: dependencyID,

		EntityType: string(dep.Type),

		Status: status,

		ResponseTime: health.ResponseTime,

		ErrorRate: health.ErrorRate,

		BusinessImpact: dep.BusinessImpact,

		Layer: layer,

		Metadata: json.RawMessage(`{}`),
	}

	return metric, nil
}

