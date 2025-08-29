
package o2



import (

	"context"

	"sync"

	"time"



	"github.com/nephio-project/nephoran-intent-operator/pkg/logging"

)



// newHealthChecker creates a new health checker instance.

func newHealthChecker(config *APIHealthCheckerConfig, logger *logging.StructuredLogger) (*HealthChecker, error) {

	if config == nil {

		config = &APIHealthCheckerConfig{

			Enabled:          true,

			CheckInterval:    30 * time.Second,

			Timeout:          10 * time.Second,

			FailureThreshold: 3,

			SuccessThreshold: 1,

			DeepHealthCheck:  true,

		}

	}



	return &HealthChecker{

		config:       config,

		healthChecks: make(map[string]ComponentHealthCheck),

		lastHealthData: &HealthCheck{

			Status:     "UP",

			Timestamp:  time.Now(),

			Components: make(map[string]interface{}),

		},

		stopCh: make(chan struct{}),

	}, nil

}



// RegisterHealthCheck registers a health check for a component.

func (h *HealthChecker) RegisterHealthCheck(name string, check ComponentHealthCheck) {

	h.healthChecks[name] = check

}



// Start starts the health checker background process.

func (h *HealthChecker) Start(ctx context.Context) {

	if !h.config.Enabled {

		return

	}



	h.ticker = time.NewTicker(h.config.CheckInterval)

	defer h.ticker.Stop()



	// Perform initial health check.

	h.performHealthCheck(ctx)



	for {

		select {

		case <-ctx.Done():

			return

		case <-h.stopCh:

			return

		case <-h.ticker.C:

			h.performHealthCheck(ctx)

		}

	}

}



// Stop stops the health checker.

func (h *HealthChecker) Stop() {

	close(h.stopCh)

}



// GetHealthStatus returns the current health status.

func (h *HealthChecker) GetHealthStatus() *HealthCheck {

	// Return a copy to prevent modifications.

	healthCopy := *h.lastHealthData

	return &healthCopy

}



// performHealthCheck executes all registered health checks.

func (h *HealthChecker) performHealthCheck(ctx context.Context) {

	checkCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)

	defer cancel()



	var checks []ComponentCheck

	var wg sync.WaitGroup

	checksChan := make(chan ComponentCheck, len(h.healthChecks))



	// Execute all health checks concurrently.

	for name, check := range h.healthChecks {

		wg.Add(1)

		go func(name string, check ComponentHealthCheck) {

			defer wg.Done()



			componentCheck := check(checkCtx)

			componentCheck.Name = name

			checksChan <- componentCheck

		}(name, check)

	}



	// Wait for all checks to complete.

	go func() {

		wg.Wait()

		close(checksChan)

	}()



	// Collect results.

	for check := range checksChan {

		checks = append(checks, check)

	}



	// Calculate overall health status.

	overallStatus := h.calculateOverallStatus(checks)



	// Update health data.

	h.lastHealthData = &HealthCheck{

		Status:     overallStatus,

		Timestamp:  time.Now(),

		Version:    "1.0.0",                // Could be injected from build

		Uptime:     time.Since(time.Now()), // This should track actual uptime

		Components: h.buildComponentsMap(checks),

		Checks:     checks,

		Services:   h.getServiceStatuses(),

		Resources:  h.getResourceHealthSummary(),

	}

}



// calculateOverallStatus determines the overall health status based on component checks.

func (h *HealthChecker) calculateOverallStatus(checks []ComponentCheck) string {

	if len(checks) == 0 {

		return "UNKNOWN"

	}



	hasUnhealthy := false

	hasDegraded := false



	for _, check := range checks {

		switch check.Status {

		case "DOWN":

			hasUnhealthy = true

		case "DEGRADED":

			hasDegraded = true

		}

	}



	if hasUnhealthy {

		return "DOWN"

	}

	if hasDegraded {

		return "DEGRADED"

	}

	return "UP"

}



// buildComponentsMap builds a map of component health information.

func (h *HealthChecker) buildComponentsMap(checks []ComponentCheck) map[string]interface{} {

	components := make(map[string]interface{})



	for _, check := range checks {

		components[check.Name] = map[string]interface{}{

			"status":     check.Status,

			"message":    check.Message,

			"timestamp":  check.Timestamp,

			"duration":   check.Duration,

			"details":    check.Details,

			"check_type": check.CheckType,

		}

	}



	return components

}



// getServiceStatuses returns the status of external service dependencies.

func (h *HealthChecker) getServiceStatuses() []HealthServiceStatus {

	// This would typically check external dependencies like databases, message queues, etc.

	var services []HealthServiceStatus



	// Example service status check.

	services = append(services, HealthServiceStatus{

		ServiceName: "database",

		Status:      "UP",

		Endpoint:    "postgresql://localhost:5432",

		LastCheck:   time.Now(),

		Latency:     5 * time.Millisecond,

	})



	return services

}



// getResourceHealthSummary returns a summary of resource health.

func (h *HealthChecker) getResourceHealthSummary() *ResourceHealthSummary {

	// This would typically aggregate resource health across all providers.

	return &ResourceHealthSummary{

		TotalResources:     100,

		HealthyResources:   95,

		DegradedResources:  3,

		UnhealthyResources: 2,

		UnknownResources:   0,

	}

}



// ComponentCheck represents the result of a component health check.

type ComponentCheck struct {

	Name      string                 `json:"name" validate:"required"`

	Status    string                 `json:"status" validate:"required,oneof=UP DOWN DEGRADED"`

	Message   string                 `json:"message,omitempty"`

	Timestamp time.Time              `json:"timestamp"`

	Duration  time.Duration          `json:"duration,omitempty"`

	Details   map[string]interface{} `json:"details,omitempty"`

	CheckType string                 `json:"check_type,omitempty"` // connectivity, resource, dependency

}

