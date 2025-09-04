package oranhealth

import (
	
	"encoding/json"
"context"
	"fmt"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/health"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/a1"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o1"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2"
)

// ORANHealthChecker provides comprehensive health checking for O-RAN interfaces.

type ORANHealthChecker struct {
	healthChecker *health.HealthChecker

	a1Adaptor *a1.A1Adaptor

	e2Adaptor *e2.E2Adaptor

	o1Adaptor *o1.O1Adaptor

	o2Adaptor *o2.O2Adaptor

	circuitBreakers map[string]*llm.CircuitBreaker

	// Configuration.

	checkInterval time.Duration

	dependencyChecks map[string]DependencyCheck

	// State tracking.

	lastHealthCheck time.Time

	healthHistory []HealthSnapshot

	mutex sync.RWMutex
}

// DependencyCheck represents a dependency health check configuration.

type DependencyCheck struct {
	Name string `json:"name"`

	URL string `json:"url"`

	Timeout time.Duration `json:"timeout"`

	ExpectedStatusCode int `json:"expected_status_code"`

	CriticalDependency bool `json:"critical_dependency"`

	CheckInterval time.Duration `json:"check_interval"`

	MaxFailures int `json:"max_failures"`

	CurrentFailures int `json:"current_failures"`

	LastCheck time.Time `json:"last_check"`

	LastError string `json:"last_error,omitempty"`
}

// HealthSnapshot represents a point-in-time health status.

type HealthSnapshot struct {
	Timestamp time.Time `json:"timestamp"`

	OverallStatus health.Status `json:"overall_status"`

	InterfaceStatus map[string]health.Status `json:"interface_status"`

	DependencyStatus map[string]health.Status `json:"dependency_status"`

	CircuitBreakerStats json.RawMessage `json:"circuit_breaker_stats"`

	Metrics HealthMetrics `json:"metrics"`
}

// HealthMetrics contains quantitative health metrics.

type HealthMetrics struct {
	TotalChecks int64 `json:"total_checks"`

	HealthyChecks int64 `json:"healthy_checks"`

	UnhealthyChecks int64 `json:"unhealthy_checks"`

	AverageResponseTime time.Duration `json:"average_response_time"`

	UpTime time.Duration `json:"uptime"`

	CircuitBreakerTrips int64 `json:"circuit_breaker_trips"`

	RetryAttempts int64 `json:"retry_attempts"`

	DependencyFailures int64 `json:"dependency_failures"`
}

// ORANHealthConfig holds configuration for O-RAN health checker.

type ORANHealthConfig struct {
	CheckInterval time.Duration `json:"check_interval"`

	HistorySize int `json:"history_size"`

	DependencyChecks map[string]DependencyCheck `json:"dependency_checks"`

	AlertingThresholds AlertingThresholds `json:"alerting_thresholds"`
}

// AlertingThresholds defines when to trigger alerts.

type AlertingThresholds struct {
	ConsecutiveFailures int `json:"consecutive_failures"`

	DependencyFailureRate float64 `json:"dependency_failure_rate"`

	CircuitBreakerOpenTime time.Duration `json:"circuit_breaker_open_time"`

	ResponseTimeThreshold time.Duration `json:"response_time_threshold"`
}

// NewORANHealthChecker creates a new O-RAN health checker.

func NewORANHealthChecker(
	healthChecker *health.HealthChecker,

	a1Adaptor *a1.A1Adaptor,

	e2Adaptor *e2.E2Adaptor,

	o1Adaptor *o1.O1Adaptor,

	o2Adaptor *o2.O2Adaptor,

	config *ORANHealthConfig,
) *ORANHealthChecker {
	if config == nil {
		config = &ORANHealthConfig{
			CheckInterval: 30 * time.Second,

			HistorySize: 100,

			DependencyChecks: make(map[string]DependencyCheck),

			AlertingThresholds: AlertingThresholds{
				ConsecutiveFailures: 3,

				DependencyFailureRate: 0.5,

				CircuitBreakerOpenTime: 5 * time.Minute,

				ResponseTimeThreshold: 5 * time.Second,
			},
		}
	}

	checker := &ORANHealthChecker{
		healthChecker: healthChecker,

		a1Adaptor: a1Adaptor,

		e2Adaptor: e2Adaptor,

		o1Adaptor: o1Adaptor,

		o2Adaptor: o2Adaptor,

		circuitBreakers: make(map[string]*llm.CircuitBreaker),

		checkInterval: config.CheckInterval,

		dependencyChecks: config.DependencyChecks,

		healthHistory: make([]HealthSnapshot, 0, config.HistorySize),
	}

	// Register O-RAN specific health checks.

	checker.registerORANHealthChecks()

	// Start periodic health checking.

	go checker.startPeriodicHealthCheck()

	return checker
}

// registerORANHealthChecks registers health checks for all O-RAN interfaces.

func (ohc *ORANHealthChecker) registerORANHealthChecks() {
	// A1 Interface Health Check.

	ohc.healthChecker.RegisterCheck("a1-interface", func(ctx context.Context) *health.Check {
		start := time.Now()

		status := health.StatusHealthy

		var message string

		var err error

		if ohc.a1Adaptor != nil {

			// Check circuit breaker status.

			stats := ohc.a1Adaptor.GetCircuitBreakerStats()

			if state, ok := stats["state"].(string); ok && state == "open" {

				status = health.StatusUnhealthy

				message = "A1 circuit breaker is open"

			} else {
				message = "A1 interface operational"
			}

		} else {

			status = health.StatusUnhealthy

			message = "A1 adaptor not initialized"

		}

		return &health.Check{
			Name: "a1-interface",

			Status: status,

			Message: message,

			Error: errorString(err),

			Duration: time.Since(start),

			Component: "oran-a1",

			Metadata: json.RawMessage(`{}`),
		}
	})

	// E2 Interface Health Check.

	ohc.healthChecker.RegisterCheck("e2-interface", func(ctx context.Context) *health.Check {
		start := time.Now()

		status := health.StatusHealthy

		var message string

		var err error

		if ohc.e2Adaptor != nil {

			stats := ohc.e2Adaptor.GetCircuitBreakerStats()

			if state, ok := stats["state"].(string); ok && state == "open" {

				status = health.StatusUnhealthy

				message = "E2 circuit breaker is open"

			} else {
				message = "E2 interface operational"
			}

		} else {

			status = health.StatusUnhealthy

			message = "E2 adaptor not initialized"

		}

		return &health.Check{
			Name: "e2-interface",

			Status: status,

			Message: message,

			Error: errorString(err),

			Duration: time.Since(start),

			Component: "oran-e2",

			Metadata: json.RawMessage(`{}`),
		}
	})

	// O1 Interface Health Check.

	ohc.healthChecker.RegisterCheck("o1-interface", func(ctx context.Context) *health.Check {
		start := time.Now()

		status := health.StatusHealthy

		var message string

		var err error

		if ohc.o1Adaptor != nil {
			// Check O1 interface connectivity.

			message = "O1 interface operational"
		} else {

			status = health.StatusUnhealthy

			message = "O1 adaptor not initialized"

		}

		return &health.Check{
			Name: "o1-interface",

			Status: status,

			Message: message,

			Error: errorString(err),

			Duration: time.Since(start),

			Component: "oran-o1",

			Metadata: json.RawMessage(`{}`),
		}
	})

	// O2 Interface Health Check.

	ohc.healthChecker.RegisterCheck("o2-interface", func(ctx context.Context) *health.Check {
		start := time.Now()

		status := health.StatusHealthy

		var message string

		var err error

		if ohc.o2Adaptor != nil {
			message = "O2 interface operational"
		} else {

			status = health.StatusUnhealthy

			message = "O2 adaptor not initialized"

		}

		return &health.Check{
			Name: "o2-interface",

			Status: status,

			Message: message,

			Error: errorString(err),

			Duration: time.Since(start),

			Component: "oran-o2",

			Metadata: json.RawMessage(`{}`),
		}
	})

	// Register dependency checks.

	for name, depCheck := range ohc.dependencyChecks {
		ohc.registerDependencyCheck(name, depCheck)
	}
}

// registerDependencyCheck registers a dependency health check.

func (ohc *ORANHealthChecker) registerDependencyCheck(name string, depCheck DependencyCheck) {
	ohc.healthChecker.RegisterDependency(name, func(ctx context.Context) *health.Check {
		return health.HTTPCheck(name, depCheck.URL)(ctx)
	})
}

// startPeriodicHealthCheck starts the periodic health checking routine.

func (ohc *ORANHealthChecker) startPeriodicHealthCheck() {
	ticker := time.NewTicker(ohc.checkInterval)

	defer ticker.Stop()

	for range ticker.C {
		ohc.performHealthCheck()
	}
}

// performHealthCheck performs a comprehensive health check.

func (ohc *ORANHealthChecker) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	// Create empty circuit breaker stats as raw JSON
	emptyStats := json.RawMessage(`{}`)

	snapshot := HealthSnapshot{
		Timestamp: time.Now(),

		InterfaceStatus: make(map[string]health.Status),

		DependencyStatus: make(map[string]health.Status),

		CircuitBreakerStats: emptyStats,
	}

	// Perform health check.

	healthResponse := ohc.healthChecker.Check(ctx)

	snapshot.OverallStatus = healthResponse.Status

	// Collect interface status.

	for _, check := range healthResponse.Checks {
		snapshot.InterfaceStatus[check.Name] = check.Status
	}

	// Collect dependency status.

	for _, dep := range healthResponse.Dependencies {
		snapshot.DependencyStatus[dep.Name] = dep.Status
	}

	// Collect circuit breaker stats and marshal to JSON.
	cbStats := make(map[string]interface{})

	if ohc.a1Adaptor != nil {
		cbStats["a1"] = ohc.a1Adaptor.GetCircuitBreakerStats()
	}

	if ohc.e2Adaptor != nil {
		cbStats["e2"] = ohc.e2Adaptor.GetCircuitBreakerStats()
	}

	// Marshal circuit breaker stats to JSON
	if statsJSON, err := json.Marshal(cbStats); err == nil {
		snapshot.CircuitBreakerStats = json.RawMessage(statsJSON)
	}

	// Calculate metrics.

	snapshot.Metrics = ohc.calculateMetrics()

	// Store snapshot.

	ohc.mutex.Lock()

	ohc.healthHistory = append(ohc.healthHistory, snapshot)

	if len(ohc.healthHistory) > 100 { // Keep last 100 snapshots

		ohc.healthHistory = ohc.healthHistory[1:]
	}

	ohc.lastHealthCheck = time.Now()

	ohc.mutex.Unlock()

	// Check for alerting conditions.

	ohc.checkAlertingConditions(snapshot)
}

// calculateMetrics calculates health metrics.

func (ohc *ORANHealthChecker) calculateMetrics() HealthMetrics {
	ohc.mutex.RLock()

	defer ohc.mutex.RUnlock()

	metrics := HealthMetrics{
		TotalChecks: int64(len(ohc.healthHistory)),
	}

	if len(ohc.healthHistory) == 0 {
		return metrics
	}

	// Calculate aggregate metrics from history.

	var healthyCount int64

	for _, snapshot := range ohc.healthHistory {
		if snapshot.OverallStatus == health.StatusHealthy {
			healthyCount++
		}
	}

	metrics.HealthyChecks = healthyCount

	metrics.UnhealthyChecks = metrics.TotalChecks - healthyCount

	// Calculate uptime (simplified).

	if len(ohc.healthHistory) > 0 {

		firstCheck := ohc.healthHistory[0].Timestamp

		metrics.UpTime = time.Since(firstCheck)

	}

	return metrics
}

// checkAlertingConditions checks if any alerting conditions are met.

func (ohc *ORANHealthChecker) checkAlertingConditions(snapshot HealthSnapshot) {
	// Check for consecutive failures.

	consecutiveFailures := ohc.getConsecutiveFailures()

	if consecutiveFailures >= ohc.getAlertingThresholds().ConsecutiveFailures {
		ohc.triggerAlert("consecutive_failures", fmt.Sprintf("Detected %d consecutive health check failures", consecutiveFailures))
	}

	// Check circuit breaker status by unmarshaling the JSON.
	var statsMap map[string]interface{}
	if err := json.Unmarshal(snapshot.CircuitBreakerStats, &statsMap); err == nil {
		for interfaceName, stats := range statsMap {
			if interfaceStats, ok := stats.(map[string]interface{}); ok {
				if state, ok := interfaceStats["state"].(string); ok && state == "open" {
					ohc.triggerAlert("circuit_breaker_open", fmt.Sprintf("Circuit breaker for %s interface is open", interfaceName))
				}
			}
		}
	}
}

// getConsecutiveFailures returns the number of consecutive failures.

func (ohc *ORANHealthChecker) getConsecutiveFailures() int {
	ohc.mutex.RLock()

	defer ohc.mutex.RUnlock()

	if len(ohc.healthHistory) == 0 {
		return 0
	}

	failures := 0

	for i := len(ohc.healthHistory) - 1; i >= 0; i-- {
		if ohc.healthHistory[i].OverallStatus != health.StatusHealthy {
			failures++
		} else {
			break
		}
	}

	return failures
}

// getAlertingThresholds returns the alerting thresholds.

func (ohc *ORANHealthChecker) getAlertingThresholds() AlertingThresholds {
	return AlertingThresholds{
		ConsecutiveFailures: 3,

		DependencyFailureRate: 0.5,

		CircuitBreakerOpenTime: 5 * time.Minute,

		ResponseTimeThreshold: 5 * time.Second,
	}
}

// triggerAlert triggers an alert for the given condition.

func (ohc *ORANHealthChecker) triggerAlert(alertType, message string) {
	// In a production system, this would integrate with alerting systems.

	// For now, we'll log the alert.

	fmt.Printf("ALERT [%s]: %s at %s\n", alertType, message, time.Now().Format(time.RFC3339))
}

// GetHealthSnapshot returns the latest health snapshot.

func (ohc *ORANHealthChecker) GetHealthSnapshot() *HealthSnapshot {
	ohc.mutex.RLock()

	defer ohc.mutex.RUnlock()

	if len(ohc.healthHistory) == 0 {
		return nil
	}

	latest := ohc.healthHistory[len(ohc.healthHistory)-1]

	return &latest
}

// GetHealthHistory returns the health history.

func (ohc *ORANHealthChecker) GetHealthHistory() []HealthSnapshot {
	ohc.mutex.RLock()

	defer ohc.mutex.RUnlock()

	history := make([]HealthSnapshot, len(ohc.healthHistory))

	copy(history, ohc.healthHistory)

	return history
}

// GetCircuitBreakerStats returns circuit breaker statistics for all interfaces.

func (ohc *ORANHealthChecker) GetCircuitBreakerStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if ohc.a1Adaptor != nil {
		stats["a1"] = ohc.a1Adaptor.GetCircuitBreakerStats()
	}

	if ohc.e2Adaptor != nil {
		stats["e2"] = ohc.e2Adaptor.GetCircuitBreakerStats()
	}

	return stats
}

// ResetCircuitBreakers resets all circuit breakers.

func (ohc *ORANHealthChecker) ResetCircuitBreakers() {
	if ohc.a1Adaptor != nil {
		ohc.a1Adaptor.ResetCircuitBreaker()
	}

	if ohc.e2Adaptor != nil {
		ohc.e2Adaptor.ResetCircuitBreaker()
	}
}

// IsHealthy returns true if all O-RAN interfaces are healthy.

func (ohc *ORANHealthChecker) IsHealthy() bool {
	snapshot := ohc.GetHealthSnapshot()

	if snapshot == nil {
		return false
	}

	return snapshot.OverallStatus == health.StatusHealthy
}

// errorString converts an error to string, handling nil case.

func errorString(err error) string {
	if err == nil {
		return ""
	}

	return err.Error()
}