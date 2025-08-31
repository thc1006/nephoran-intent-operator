package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
)

// BasicHealthChecker provides a basic implementation of HealthChecker
type BasicHealthChecker struct {
	version       string
	k8sClient     kubernetes.Interface
	recorder      *MetricsRecorder
	healthChecks  map[string]HealthChecker
	mu            sync.RWMutex
	status        HealthStatus
	lastCheck     time.Time
	message       string
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(version string, k8sClient kubernetes.Interface, recorder *MetricsRecorder) *BasicHealthChecker {
	return &BasicHealthChecker{
		version:      version,
		k8sClient:    k8sClient,
		recorder:     recorder,
		healthChecks: make(map[string]HealthChecker),
		status:       HealthStatusUnknown,
		lastCheck:    time.Now(),
		message:      "Health checker initialized",
	}
}

// CheckHealth implements the HealthChecker interface
func (hc *BasicHealthChecker) CheckHealth() (HealthStatus, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	return hc.status, nil
}

// GetName implements the HealthChecker interface
func (hc *BasicHealthChecker) GetName() string {
	return "system-health-checker"
}

// Start implements the HealthChecker interface
func (hc *BasicHealthChecker) Start(ctx context.Context) error {
	// Start periodic health checking
	go hc.healthCheckLoop(ctx)
	return nil
}

// RegisterCheck registers a health check
func (hc *BasicHealthChecker) RegisterCheck(checker HealthChecker) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.healthChecks[checker.GetName()] = checker
}

// UnregisterCheck removes a health check
func (hc *BasicHealthChecker) UnregisterCheck(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	delete(hc.healthChecks, name)
}

// GetSystemHealth returns overall system health
func (hc *BasicHealthChecker) GetSystemHealth() (*SystemHealth, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	components := make(map[string]*ComponentHealth)
	overallStatus := HealthStatusHealthy
	healthyCount := 0
	totalCount := len(hc.healthChecks)
	var issues []string
	
	// Check each registered health check
	for name, checker := range hc.healthChecks {
		status, err := checker.CheckHealth()
		componentHealth := &ComponentHealth{
			Name:        name,
			Status:      status,
			Timestamp:   time.Now(),
			LastChecked: time.Now(),
		}
		
		if err != nil {
			componentHealth.Status = HealthStatusUnhealthy
			componentHealth.Message = err.Error()
			issues = append(issues, fmt.Sprintf("%s: %v", name, err))
		} else {
			componentHealth.Message = "Healthy"
			if status == HealthStatusHealthy {
				healthyCount++
			}
		}
		
		components[name] = componentHealth
		
		// Determine overall status
		switch status {
		case HealthStatusUnhealthy:
			overallStatus = HealthStatusUnhealthy
		case HealthStatusDegraded:
			if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
		}
	}
	
	// Calculate health score
	healthScore := 0.0
	if totalCount > 0 {
		healthScore = float64(healthyCount) / float64(totalCount)
	}
	
	return &SystemHealth{
		OverallStatus: overallStatus,
		Components:    components,
		LastUpdated:   time.Now(),
		HealthScore:   healthScore,
		Issues:        issues,
	}, nil
}

// healthCheckLoop runs periodic health checks
func (hc *BasicHealthChecker) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hc.performHealthCheck()
		}
	}
}

// performHealthCheck performs a health check cycle
func (hc *BasicHealthChecker) performHealthCheck() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	overallHealthy := true
	healthyCount := 0
	totalCount := len(hc.healthChecks)
	var messages []string
	
	for name, checker := range hc.healthChecks {
		status, err := checker.CheckHealth()
		if err != nil || status != HealthStatusHealthy {
			overallHealthy = false
			messages = append(messages, fmt.Sprintf("%s: %v", name, err))
		} else {
			healthyCount++
		}
	}
	
	// Update overall status
	if overallHealthy && totalCount > 0 {
		hc.status = HealthStatusHealthy
		hc.message = fmt.Sprintf("All %d components healthy", totalCount)
	} else if float64(healthyCount)/float64(totalCount) > 0.5 {
		hc.status = HealthStatusDegraded
		hc.message = fmt.Sprintf("%d/%d components healthy", healthyCount, totalCount)
	} else {
		hc.status = HealthStatusUnhealthy
		hc.message = fmt.Sprintf("Only %d/%d components healthy", healthyCount, totalCount)
	}
	
	hc.lastCheck = time.Now()
}

// GetVersion returns the health checker version
func (hc *BasicHealthChecker) GetVersion() string {
	return hc.version
}

// GetLastCheckTime returns the last health check time
func (hc *BasicHealthChecker) GetLastCheckTime() time.Time {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.lastCheck
}

// GetMessage returns the current health message
func (hc *BasicHealthChecker) GetMessage() string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.message
}