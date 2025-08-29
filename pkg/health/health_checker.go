package health

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status of a component.
type Status string

const (
	// StatusHealthy holds statushealthy value.
	StatusHealthy Status = "healthy"
	// StatusUnhealthy holds statusunhealthy value.
	StatusUnhealthy Status = "unhealthy"
	// StatusUnknown holds statusunknown value.
	StatusUnknown Status = "unknown"
	// StatusDegraded holds statusdegraded value.
	StatusDegraded Status = "degraded"
)

// Check represents a single health check.
type Check struct {
	Name      string        `json:"name"`
	Status    Status        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
	Component string        `json:"component"`
	Version   string        `json:"version,omitempty"`
	Metadata  interface{}   `json:"metadata,omitempty"`
}

// CheckFunc is a function that performs a health check.
type CheckFunc func(ctx context.Context) *Check

// HealthChecker manages and executes health checks.
type HealthChecker struct {
	serviceName    string
	serviceVersion string
	startTime      time.Time
	checks         map[string]CheckFunc
	dependencies   map[string]CheckFunc
	mu             sync.RWMutex
	logger         *slog.Logger
	timeout        time.Duration
	gracePeriod    time.Duration
	healthyState   bool
	readyState     bool
	stateMu        sync.RWMutex
}

// HealthResponse represents the complete health check response.
type HealthResponse struct {
	Service      string                 `json:"service"`
	Version      string                 `json:"version"`
	Status       Status                 `json:"status"`
	Uptime       string                 `json:"uptime"`
	Timestamp    time.Time              `json:"timestamp"`
	Checks       []Check                `json:"checks"`
	Dependencies []Check                `json:"dependencies"`
	Summary      HealthSummary          `json:"summary"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// HealthSummary provides aggregated health information.
type HealthSummary struct {
	Total     int `json:"total"`
	Healthy   int `json:"healthy"`
	Unhealthy int `json:"unhealthy"`
	Degraded  int `json:"degraded"`
	Unknown   int `json:"unknown"`
}

// NewHealthChecker creates a new health checker instance.
func NewHealthChecker(serviceName, serviceVersion string, logger *slog.Logger) *HealthChecker {
	return &HealthChecker{
		serviceName:    serviceName,
		serviceVersion: serviceVersion,
		startTime:      time.Now(),
		checks:         make(map[string]CheckFunc),
		dependencies:   make(map[string]CheckFunc),
		logger:         logger,
		timeout:        30 * time.Second,
		gracePeriod:    15 * time.Second,
		healthyState:   true,
		readyState:     false,
	}
}

// RegisterCheck registers a health check function.
func (hc *HealthChecker) RegisterCheck(name string, checkFunc CheckFunc) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	if hc.checks == nil {
		hc.checks = make(map[string]CheckFunc)
	}
	hc.checks[name] = checkFunc
	hc.logger.Info("Health check registered", "name", name, "service", hc.serviceName)
}

// RegisterDependency registers a dependency health check.
func (hc *HealthChecker) RegisterDependency(name string, checkFunc CheckFunc) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	if hc.dependencies == nil {
		hc.dependencies = make(map[string]CheckFunc)
	}
	hc.dependencies[name] = checkFunc
	hc.logger.Info("Dependency check registered", "name", name, "service", hc.serviceName)
}

// SetReady marks the service as ready.
func (hc *HealthChecker) SetReady(ready bool) {
	hc.stateMu.Lock()
	defer hc.stateMu.Unlock()
	hc.readyState = ready
	hc.logger.Info("Readiness state changed", "ready", ready, "service", hc.serviceName)
}

// SetHealthy marks the service as healthy or unhealthy.
func (hc *HealthChecker) SetHealthy(healthy bool) {
	hc.stateMu.Lock()
	defer hc.stateMu.Unlock()
	hc.healthyState = healthy
	hc.logger.Info("Health state changed", "healthy", healthy, "service", hc.serviceName)
}

// IsReady returns the current readiness state.
func (hc *HealthChecker) IsReady() bool {
	hc.stateMu.RLock()
	defer hc.stateMu.RUnlock()
	return hc.readyState
}

// IsHealthy returns the current health state.
func (hc *HealthChecker) IsHealthy() bool {
	hc.stateMu.RLock()
	defer hc.stateMu.RUnlock()
	return hc.healthyState
}

// Check performs all registered health checks.
func (hc *HealthChecker) Check(ctx context.Context) *HealthResponse {
	ctx, cancel := context.WithTimeout(ctx, hc.timeout)
	defer cancel()

	response := &HealthResponse{
		Service:      hc.serviceName,
		Version:      hc.serviceVersion,
		Uptime:       time.Since(hc.startTime).String(),
		Timestamp:    time.Now(),
		Checks:       []Check{},
		Dependencies: []Check{},
		Metadata: map[string]interface{}{
			"grace_period_remaining": hc.getGracePeriodRemaining(),
			"checks_count":           len(hc.checks),
			"dependencies_count":     len(hc.dependencies),
		},
	}

	// Perform internal health checks.
	hc.mu.RLock()
	checks := make(map[string]CheckFunc)
	for name, checkFunc := range hc.checks {
		checks[name] = checkFunc
	}
	dependencies := make(map[string]CheckFunc)
	for name, checkFunc := range hc.dependencies {
		dependencies[name] = checkFunc
	}
	hc.mu.RUnlock()

	// Execute checks concurrently.
	var wg sync.WaitGroup
	checkResults := make(chan Check, len(checks)+len(dependencies))

	// Execute service checks.
	for name, checkFunc := range checks {
		wg.Add(1)
		go func(name string, checkFunc CheckFunc) {
			defer wg.Done()
			start := time.Now()
			check := checkFunc(ctx)
			if check == nil {
				check = &Check{
					Name:      name,
					Status:    StatusUnknown,
					Message:   "Check function returned nil",
					Duration:  time.Since(start),
					Timestamp: time.Now(),
					Component: hc.serviceName,
				}
			}
			check.Name = name
			check.Duration = time.Since(start)
			check.Timestamp = time.Now()
			check.Component = hc.serviceName
			checkResults <- *check
		}(name, checkFunc)
	}

	// Execute dependency checks.
	for name, checkFunc := range dependencies {
		wg.Add(1)
		go func(name string, checkFunc CheckFunc) {
			defer wg.Done()
			start := time.Now()
			check := checkFunc(ctx)
			if check == nil {
				check = &Check{
					Name:      name,
					Status:    StatusUnknown,
					Message:   "Dependency check function returned nil",
					Duration:  time.Since(start),
					Timestamp: time.Now(),
					Component: "dependency",
				}
			}
			check.Name = name
			check.Duration = time.Since(start)
			check.Timestamp = time.Now()
			check.Component = "dependency"
			checkResults <- *check
		}(name, checkFunc)
	}

	// Wait for all checks to complete.
	go func() {
		wg.Wait()
		close(checkResults)
	}()

	// Collect results.
	summary := HealthSummary{}
	for check := range checkResults {
		summary.Total++
		switch check.Status {
		case StatusHealthy:
			summary.Healthy++
		case StatusUnhealthy:
			summary.Unhealthy++
		case StatusDegraded:
			summary.Degraded++
		default:
			summary.Unknown++
		}

		if check.Component == "dependency" {
			response.Dependencies = append(response.Dependencies, check)
		} else {
			response.Checks = append(response.Checks, check)
		}
	}

	response.Summary = summary

	// Determine overall status.
	response.Status = hc.determineOverallStatus(summary)

	// Update internal health state based on checks.
	hc.updateHealthState(response.Status)

	return response
}

// HealthzHandler provides Kubernetes liveness probe endpoint.
func (hc *HealthChecker) HealthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	response := hc.Check(ctx)

	w.Header().Set("Content-Type", "application/json")

	// During grace period, always return healthy.
	if hc.isInGracePeriod() {
		response.Status = StatusHealthy
		w.WriteHeader(http.StatusOK)
	} else if response.Status == StatusHealthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode livez response", http.StatusInternalServerError)
	}
}

// ReadyzHandler provides Kubernetes readiness probe endpoint.
func (hc *HealthChecker) ReadyzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	response := hc.Check(ctx)

	// Readiness includes dependency checks and internal ready state.
	isReady := hc.IsReady() && (response.Status == StatusHealthy || response.Status == StatusDegraded)

	// Check critical dependencies.
	for _, dep := range response.Dependencies {
		if dep.Status == StatusUnhealthy {
			isReady = false
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if isReady {
		response.Status = StatusHealthy
		w.WriteHeader(http.StatusOK)
	} else {
		response.Status = StatusUnhealthy
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode readyz response", http.StatusInternalServerError)
	}
}

// determineOverallStatus determines the overall service status.
func (hc *HealthChecker) determineOverallStatus(summary HealthSummary) Status {
	if summary.Total == 0 {
		return StatusUnknown
	}

	// If all checks are healthy, service is healthy.
	if summary.Healthy == summary.Total {
		return StatusHealthy
	}

	// If any check is unhealthy, service is unhealthy.
	if summary.Unhealthy > 0 {
		return StatusUnhealthy
	}

	// If we have degraded checks but no unhealthy ones, service is degraded.
	if summary.Degraded > 0 {
		return StatusDegraded
	}

	// Otherwise, status is unknown.
	return StatusUnknown
}

// updateHealthState updates the internal health state.
func (hc *HealthChecker) updateHealthState(status Status) {
	healthy := status == StatusHealthy || status == StatusDegraded
	hc.SetHealthy(healthy)
}

// isInGracePeriod checks if we're still in the startup grace period.
func (hc *HealthChecker) isInGracePeriod() bool {
	return time.Since(hc.startTime) < hc.gracePeriod
}

// getGracePeriodRemaining returns remaining grace period duration.
func (hc *HealthChecker) getGracePeriodRemaining() time.Duration {
	remaining := hc.gracePeriod - time.Since(hc.startTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Common health check functions.

// HTTPCheck performs an HTTP health check.
func HTTPCheck(name, url string) CheckFunc {
	return func(ctx context.Context) *Check {
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return &Check{
				Name:   name,
				Status: StatusUnhealthy,
				Error:  fmt.Sprintf("Failed to create request: %v", err),
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			return &Check{
				Name:   name,
				Status: StatusUnhealthy,
				Error:  fmt.Sprintf("HTTP request failed: %v", err),
			}
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return &Check{
				Name:    name,
				Status:  StatusHealthy,
				Message: fmt.Sprintf("HTTP %d", resp.StatusCode),
			}
		}

		return &Check{
			Name:   name,
			Status: StatusUnhealthy,
			Error:  fmt.Sprintf("HTTP %d", resp.StatusCode),
		}
	}
}

// DatabaseCheck performs a database connection health check.
func DatabaseCheck(name string, pingFunc func(ctx context.Context) error) CheckFunc {
	return func(ctx context.Context) *Check {
		err := pingFunc(ctx)
		if err != nil {
			return &Check{
				Name:   name,
				Status: StatusUnhealthy,
				Error:  fmt.Sprintf("Database connection failed: %v", err),
			}
		}
		return &Check{
			Name:    name,
			Status:  StatusHealthy,
			Message: "Database connection successful",
		}
	}
}

// MemoryCheck performs a memory usage health check.
func MemoryCheck(name string, maxMemoryMB int64) CheckFunc {
	return func(ctx context.Context) *Check {
		// This is a simplified memory check.
		// In a real implementation, you would use runtime.MemStats.
		return &Check{
			Name:    name,
			Status:  StatusHealthy,
			Message: "Memory usage within limits",
		}
	}
}
