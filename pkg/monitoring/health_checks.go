package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Name           string                 `json:"name"`
	Status         HealthStatus           `json:"status"`
	Message        string                 `json:"message,omitempty"`
	LastChecked    time.Time              `json:"last_checked"`
	ResponseTime   time.Duration          `json:"response_time"`
	Details        map[string]interface{} `json:"details,omitempty"`
	Dependencies   []ComponentHealth      `json:"dependencies,omitempty"`
	CheckInterval  time.Duration          `json:"check_interval"`
	FailureCount   int                    `json:"failure_count"`
	ConsecutiveFails int                  `json:"consecutive_fails"`
}

// SystemHealth represents the overall system health
type SystemHealth struct {
	Status      HealthStatus      `json:"status"`
	Version     string            `json:"version"`
	Timestamp   time.Time         `json:"timestamp"`
	Uptime      time.Duration     `json:"uptime"`
	Components  []ComponentHealth `json:"components"`
	Summary     HealthSummary     `json:"summary"`
}

// HealthSummary provides a summary of system health
type HealthSummary struct {
	TotalComponents int `json:"total_components"`
	HealthyComponents int `json:"healthy_components"`
	UnhealthyComponents int `json:"unhealthy_components"`
	DegradedComponents int `json:"degraded_components"`
}

// HealthCheckFunc defines a health check function
type HealthCheckFunc func(ctx context.Context) *ComponentHealth

// HealthCheckConfig holds configuration for a health check
type HealthCheckConfig struct {
	Name          string        `json:"name"`
	Interval      time.Duration `json:"interval"`
	Timeout       time.Duration `json:"timeout"`
	FailureThreshold int        `json:"failure_threshold"`
	Enabled       bool          `json:"enabled"`
}

// HealthChecker manages health checks for all components
type HealthChecker struct {
	mu            sync.RWMutex
	checks        map[string]*HealthCheckConfig
	checkResults  map[string]*ComponentHealth
	checkFuncs    map[string]HealthCheckFunc
	running       bool
	stopCh        chan struct{}
	startTime     time.Time
	version       string
	kubeClient    kubernetes.Interface
	metricsRecorder *MetricsRecorder
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(version string, kubeClient kubernetes.Interface, metricsRecorder *MetricsRecorder) *HealthChecker {
	return &HealthChecker{
		checks:          make(map[string]*HealthCheckConfig),
		checkResults:    make(map[string]*ComponentHealth),
		checkFuncs:      make(map[string]HealthCheckFunc),
		stopCh:          make(chan struct{}),
		startTime:       time.Now(),
		version:         version,
		kubeClient:      kubeClient,
		metricsRecorder: metricsRecorder,
	}
}

// RegisterHealthCheck registers a health check with configuration
func (hc *HealthChecker) RegisterHealthCheck(config *HealthCheckConfig, checkFunc HealthCheckFunc) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checks[config.Name] = config
	hc.checkFuncs[config.Name] = checkFunc
	
	// Initialize result
	hc.checkResults[config.Name] = &ComponentHealth{
		Name:          config.Name,
		Status:        HealthStatusUnknown,
		LastChecked:   time.Time{},
		CheckInterval: config.Interval,
	}
}

// Start starts the health checker
func (hc *HealthChecker) Start(ctx context.Context) error {
	hc.mu.Lock()
	if hc.running {
		hc.mu.Unlock()
		return fmt.Errorf("health checker is already running")
	}
	hc.running = true
	hc.mu.Unlock()

	// Register default health checks
	hc.registerDefaultHealthChecks()

	// Start health check goroutines
	for name, config := range hc.checks {
		if config.Enabled {
			go hc.runHealthCheck(ctx, name, config)
		}
	}

	return nil
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if !hc.running {
		return
	}

	close(hc.stopCh)
	hc.running = false
}

// GetSystemHealth returns the current system health
func (hc *HealthChecker) GetSystemHealth() *SystemHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	components := make([]ComponentHealth, 0, len(hc.checkResults))
	summary := HealthSummary{TotalComponents: len(hc.checkResults)}

	for _, result := range hc.checkResults {
		components = append(components, *result)
		
		switch result.Status {
		case HealthStatusHealthy:
			summary.HealthyComponents++
		case HealthStatusUnhealthy:
			summary.UnhealthyComponents++
		case HealthStatusDegraded:
			summary.DegradedComponents++
		}
	}

	// Determine overall system status
	systemStatus := HealthStatusHealthy
	if summary.UnhealthyComponents > 0 {
		systemStatus = HealthStatusUnhealthy
	} else if summary.DegradedComponents > 0 {
		systemStatus = HealthStatusDegraded
	}

	return &SystemHealth{
		Status:     systemStatus,
		Version:    hc.version,
		Timestamp:  time.Now(),
		Uptime:     time.Since(hc.startTime),
		Components: components,
		Summary:    summary,
	}
}

// GetComponentHealth returns the health of a specific component
func (hc *HealthChecker) GetComponentHealth(name string) (*ComponentHealth, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	result, exists := hc.checkResults[name]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	componentCopy := *result
	return &componentCopy, true
}

// runHealthCheck runs a specific health check in a loop
func (hc *HealthChecker) runHealthCheck(ctx context.Context, name string, config *HealthCheckConfig) {
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.executeHealthCheck(ctx, name, config)
		}
	}
}

// executeHealthCheck executes a single health check
func (hc *HealthChecker) executeHealthCheck(ctx context.Context, name string, config *HealthCheckConfig) {
	checkFunc, exists := hc.checkFuncs[name]
	if !exists {
		return
	}

	// Create timeout context
	checkCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	startTime := time.Now()
	result := checkFunc(checkCtx)
	duration := time.Since(startTime)

	if result == nil {
		result = &ComponentHealth{
			Name:         name,
			Status:       HealthStatusUnknown,
			Message:      "Health check returned nil result",
			ResponseTime: duration,
		}
	}

	result.LastChecked = time.Now()
	result.ResponseTime = duration
	result.CheckInterval = config.Interval

	// Update failure counts
	hc.mu.Lock()
	if oldResult, exists := hc.checkResults[name]; exists {
		result.FailureCount = oldResult.FailureCount
		result.ConsecutiveFails = oldResult.ConsecutiveFails

		if result.Status != HealthStatusHealthy {
			result.FailureCount++
			result.ConsecutiveFails++
		} else {
			result.ConsecutiveFails = 0
		}

		// Apply failure threshold logic
		if result.ConsecutiveFails >= config.FailureThreshold && result.Status == HealthStatusHealthy {
			result.Status = HealthStatusDegraded
			result.Message = fmt.Sprintf("Consecutive failures: %d", result.ConsecutiveFails)
		}
	}

	hc.checkResults[name] = result
	hc.mu.Unlock()

	// Record metrics
	if hc.metricsRecorder != nil {
		healthy := result.Status == HealthStatusHealthy
		hc.metricsRecorder.RecordControllerHealth("health_checker", name, healthy)
	}
}

// registerDefaultHealthChecks registers default health checks
func (hc *HealthChecker) registerDefaultHealthChecks() {
	// Kubernetes API Health Check
	hc.RegisterHealthCheck(&HealthCheckConfig{
		Name:             "kubernetes-api",
		Interval:         30 * time.Second,
		Timeout:          10 * time.Second,
		FailureThreshold: 3,
		Enabled:          true,
	}, hc.kubernetesAPIHealthCheck)

	// LLM Processor Health Check
	hc.RegisterHealthCheck(&HealthCheckConfig{
		Name:             "llm-processor",
		Interval:         30 * time.Second,
		Timeout:          15 * time.Second,
		FailureThreshold: 3,
		Enabled:          true,
	}, hc.llmProcessorHealthCheck)

	// RAG API Health Check
	hc.RegisterHealthCheck(&HealthCheckConfig{
		Name:             "rag-api",
		Interval:         30 * time.Second,
		Timeout:          15 * time.Second,
		FailureThreshold: 3,
		Enabled:          true,
	}, hc.ragAPIHealthCheck)

	// Weaviate Health Check
	hc.RegisterHealthCheck(&HealthCheckConfig{
		Name:             "weaviate",
		Interval:         30 * time.Second,
		Timeout:          10 * time.Second,
		FailureThreshold: 3,
		Enabled:          true,
	}, hc.weaviateHealthCheck)

	// Controller Health Check
	hc.RegisterHealthCheck(&HealthCheckConfig{
		Name:             "controller-manager",
		Interval:         60 * time.Second,
		Timeout:          10 * time.Second,
		FailureThreshold: 3,
		Enabled:          true,
	}, hc.controllerManagerHealthCheck)
}

// Specific health check implementations

// kubernetesAPIHealthCheck checks Kubernetes API connectivity
func (hc *HealthChecker) kubernetesAPIHealthCheck(ctx context.Context) *ComponentHealth {
	if hc.kubeClient == nil {
		return &ComponentHealth{
			Name:    "kubernetes-api",
			Status:  HealthStatusUnhealthy,
			Message: "Kubernetes client not initialized",
		}
	}

	startTime := time.Now()
	
	// Try to get server version
	version, err := hc.kubeClient.Discovery().ServerVersion()
	duration := time.Since(startTime)

	if err != nil {
		return &ComponentHealth{
			Name:         "kubernetes-api",
			Status:       HealthStatusUnhealthy,
			Message:      fmt.Sprintf("Failed to connect to Kubernetes API: %v", err),
			ResponseTime: duration,
		}
	}

	return &ComponentHealth{
		Name:         "kubernetes-api",
		Status:       HealthStatusHealthy,
		Message:      fmt.Sprintf("Connected to Kubernetes API server version %s", version.GitVersion),
		ResponseTime: duration,
		Details: map[string]interface{}{
			"server_version": version.GitVersion,
			"platform":       version.Platform,
		},
	}
}

// llmProcessorHealthCheck checks LLM processor service health
func (hc *HealthChecker) llmProcessorHealthCheck(ctx context.Context) *ComponentHealth {
	url := getEnv("LLM_PROCESSOR_URL", "http://llm-processor:8080")
	return hc.httpHealthCheck(ctx, "llm-processor", url+"/healthz")
}

// ragAPIHealthCheck checks RAG API service health
func (hc *HealthChecker) ragAPIHealthCheck(ctx context.Context) *ComponentHealth {
	url := getEnv("RAG_API_URL", "http://rag-api:8080")
	return hc.httpHealthCheck(ctx, "rag-api", url+"/health")
}

// weaviateHealthCheck checks Weaviate health
func (hc *HealthChecker) weaviateHealthCheck(ctx context.Context) *ComponentHealth {
	url := getEnv("WEAVIATE_URL", "http://weaviate:8080")
	return hc.httpHealthCheck(ctx, "weaviate", url+"/v1/.well-known/ready")
}

// controllerManagerHealthCheck checks controller manager health
func (hc *HealthChecker) controllerManagerHealthCheck(ctx context.Context) *ComponentHealth {
	// Check if we can list NetworkIntents (indicates controller is running)
	if hc.kubeClient == nil {
		return &ComponentHealth{
			Name:    "controller-manager",
			Status:  HealthStatusUnhealthy,
			Message: "Kubernetes client not initialized",
		}
	}

	startTime := time.Now()
	
	// Check if we can list custom resources
	_, err := hc.kubeClient.Discovery().ServerResourcesForGroupVersion("nephoran.com/v1")
	duration := time.Since(startTime)

	if err != nil {
		return &ComponentHealth{
			Name:         "controller-manager",
			Status:       HealthStatusUnhealthy,
			Message:      fmt.Sprintf("Failed to discover Nephoran CRDs: %v", err),
			ResponseTime: duration,
		}
	}

	return &ComponentHealth{
		Name:         "controller-manager",
		Status:       HealthStatusHealthy,
		Message:      "Controller manager is running and CRDs are available",
		ResponseTime: duration,
	}
}

// httpHealthCheck performs an HTTP health check
func (hc *HealthChecker) httpHealthCheck(ctx context.Context, name, url string) *ComponentHealth {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	startTime := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &ComponentHealth{
			Name:    name,
			Status:  HealthStatusUnhealthy,
			Message: fmt.Sprintf("Failed to create request: %v", err),
		}
	}

	resp, err := client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		return &ComponentHealth{
			Name:         name,
			Status:       HealthStatusUnhealthy,
			Message:      fmt.Sprintf("HTTP request failed: %v", err),
			ResponseTime: duration,
		}
	}
	defer resp.Body.Close()

	status := HealthStatusHealthy
	message := fmt.Sprintf("HTTP %d", resp.StatusCode)

	if resp.StatusCode >= 400 {
		status = HealthStatusUnhealthy
		message = fmt.Sprintf("HTTP %d - Service unhealthy", resp.StatusCode)
	} else if resp.StatusCode >= 300 {
		status = HealthStatusDegraded
		message = fmt.Sprintf("HTTP %d - Service degraded", resp.StatusCode)
	}

	return &ComponentHealth{
		Name:         name,
		Status:       status,
		Message:      message,
		ResponseTime: duration,
		Details: map[string]interface{}{
			"status_code": resp.StatusCode,
			"url":         url,
		},
	}
}

// HealthCheckHandler provides HTTP endpoints for health checks
type HealthCheckHandler struct {
	healthChecker *HealthChecker
	logger        *StructuredLogger
}

// NewHealthCheckHandler creates a new health check HTTP handler
func NewHealthCheckHandler(healthChecker *HealthChecker, logger *StructuredLogger) *HealthCheckHandler {
	return &HealthCheckHandler{
		healthChecker: healthChecker,
		logger:        logger,
	}
}

// ServeHTTP implements http.Handler
func (h *HealthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/healthz":
		h.handleLiveness(w, r)
	case "/readyz":
		h.handleReadiness(w, r)
	case "/health":
		h.handleHealth(w, r)
	default:
		http.NotFound(w, r)
	}
}

// handleLiveness handles liveness probe
func (h *HealthCheckHandler) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleReadiness handles readiness probe
func (h *HealthCheckHandler) handleReadiness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	systemHealth := h.healthChecker.GetSystemHealth()

	if systemHealth.Status == HealthStatusHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if h.logger != nil {
		h.logger.Debug(ctx, "Readiness check performed",
			slog.String("status", string(systemHealth.Status)),
			slog.Int("healthy_components", systemHealth.Summary.HealthyComponents),
			slog.Int("total_components", systemHealth.Summary.TotalComponents),
		)
	}

	w.Write([]byte(string(systemHealth.Status)))
}

// handleHealth handles detailed health check
func (h *HealthCheckHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	systemHealth := h.healthChecker.GetSystemHealth()

	w.Header().Set("Content-Type", "application/json")

	if systemHealth.Status == HealthStatusHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(systemHealth); err != nil {
		if h.logger != nil {
			h.logger.Error(ctx, "Failed to encode health response", err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if h.logger != nil {
		h.logger.Info(ctx, "Health check performed",
			slog.String("status", string(systemHealth.Status)),
			slog.Duration("uptime", systemHealth.Uptime),
			slog.Int("healthy_components", systemHealth.Summary.HealthyComponents),
			slog.Int("total_components", systemHealth.Summary.TotalComponents),
		)
	}
}

// CircuitBreakerHealthCheck implements health check for circuit breakers
type CircuitBreakerHealthCheck struct {
	name     string
	breakers map[string]*CircuitBreaker
}

// CircuitBreaker represents a simple circuit breaker
type CircuitBreaker struct {
	name           string
	failureCount   int
	lastFailure    time.Time
	state          CircuitBreakerState
	failureThreshold int
	timeout        time.Duration
}

// CircuitBreakerState represents circuit breaker states
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// NewCircuitBreakerHealthCheck creates a new circuit breaker health check
func NewCircuitBreakerHealthCheck(name string, breakers map[string]*CircuitBreaker) *CircuitBreakerHealthCheck {
	return &CircuitBreakerHealthCheck{
		name:     name,
		breakers: breakers,
	}
}

// Check performs the circuit breaker health check
func (cb *CircuitBreakerHealthCheck) Check(ctx context.Context) *ComponentHealth {
	if len(cb.breakers) == 0 {
		return &ComponentHealth{
			Name:    cb.name,
			Status:  HealthStatusHealthy,
			Message: "No circuit breakers configured",
		}
	}

	openBreakers := 0
	degradedBreakers := 0
	details := make(map[string]interface{})

	for name, breaker := range cb.breakers {
		switch breaker.state {
		case CircuitBreakerOpen:
			openBreakers++
		case CircuitBreakerHalfOpen:
			degradedBreakers++
		}

		details[name] = map[string]interface{}{
			"state":         breaker.state,
			"failure_count": breaker.failureCount,
			"last_failure":  breaker.lastFailure,
		}
	}

	status := HealthStatusHealthy
	message := "All circuit breakers are closed"

	if openBreakers > 0 {
		status = HealthStatusUnhealthy
		message = fmt.Sprintf("%d circuit breakers are open", openBreakers)
	} else if degradedBreakers > 0 {
		status = HealthStatusDegraded
		message = fmt.Sprintf("%d circuit breakers are half-open", degradedBreakers)
	}

	return &ComponentHealth{
		Name:    cb.name,
		Status:  status,
		Message: message,
		Details: details,
	}
}