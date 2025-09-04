// Copyright 2024 Nephio
// SPDX-License-Identifier: Apache-2.0

package monitoring

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// HealthCheckerImpl implements the HealthChecker interface
type HealthCheckerImpl struct {
	checks   map[string]HealthCheck
	results  map[string]HealthStatus
	mu       sync.RWMutex
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// HealthCheck represents a health check function
type HealthCheck struct {
	Name        string
	Description string
	CheckFunc   func(ctx context.Context) error
	Timeout     time.Duration
	Critical    bool
}

// Health check status constants
const (
	HealthStatusHealthy   = "healthy"
	HealthStatusDegraded  = "degraded"
	HealthStatusUnhealthy = "unhealthy"
	HealthStatusUnknown   = "unknown"
)

// HealthStatus represents the status of a health check
type HealthStatus struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Message     string    `json:"message,omitempty"`
	LastChecked time.Time `json:"last_checked"`
	Duration    string    `json:"duration"`
	Critical    bool      `json:"critical"`
	Error       string    `json:"error,omitempty"`
}

// NewHealthCheckerImpl creates a new health checker implementation
func NewHealthCheckerImpl(interval time.Duration) *HealthCheckerImpl {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &HealthCheckerImpl{
		checks:   make(map[string]HealthCheck),
		results:  make(map[string]HealthStatus),
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// AddCheck adds a health check
func (h *HealthCheckerImpl) AddCheck(check HealthCheck) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	if check.Timeout == 0 {
		check.Timeout = 30 * time.Second
	}
	
	h.checks[check.Name] = check
	
	// Initialize with unknown status
	h.results[check.Name] = HealthStatus{
		Name:        check.Name,
		Status:      "unknown",
		Message:     "Not yet checked",
		LastChecked: time.Time{},
		Duration:    "0s",
		Critical:    check.Critical,
	}
}

// RemoveCheck removes a health check
func (h *HealthCheckerImpl) RemoveCheck(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	delete(h.checks, name)
	delete(h.results, name)
}

// GetStatus returns the current health status
func (h *HealthCheckerImpl) GetStatus() map[string]HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	result := make(map[string]HealthStatus, len(h.results))
	for k, v := range h.results {
		result[k] = v
	}
	
	return result
}

// IsHealthy returns true if all critical checks are passing
func (h *HealthCheckerImpl) IsHealthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	for _, status := range h.results {
		if status.Critical && status.Status != "healthy" {
			return false
		}
	}
	
	return true
}

// GetOverallStatus returns the overall health status
func (h *HealthCheckerImpl) GetOverallStatus() HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	overallStatus := "healthy"
	var messages []string
	criticalCount := 0
	unhealthyCount := 0
	totalChecks := len(h.results)
	
	for _, status := range h.results {
		if status.Critical {
			criticalCount++
			if status.Status != "healthy" {
				overallStatus = "unhealthy"
				messages = append(messages, fmt.Sprintf("%s: %s", status.Name, status.Message))
				unhealthyCount++
			}
		} else if status.Status != "healthy" {
			unhealthyCount++
		}
	}
	
	// Determine overall status
	if unhealthyCount > 0 {
		if criticalCount > 0 && overallStatus == "unhealthy" {
			overallStatus = "critical"
		} else if overallStatus != "critical" {
			overallStatus = "degraded"
		}
	}
	
	var message string
	if len(messages) > 0 {
		message = fmt.Sprintf("Critical issues: %v", messages)
	} else {
		healthyPercentage := float64(totalChecks-unhealthyCount) / float64(totalChecks) * 100
		// Convert float64 to string using strconv.FormatFloat
		message = fmt.Sprintf("System health: %s%% (%d/%d checks passing)", 
			strconv.FormatFloat(healthyPercentage, 'f', 1, 64), 
			totalChecks-unhealthyCount, 
			totalChecks)
	}
	
	return HealthStatus{
		Name:        "overall",
		Status:      overallStatus,
		Message:     message,
		LastChecked: time.Now(),
		Duration:    "0s",
		Critical:    true,
	}
}

// Start begins periodic health checking
func (h *HealthCheckerImpl) Start(ctx context.Context) error {
	h.wg.Add(1)
	go h.runHealthChecks()
	return nil
}

// CheckHealth returns the overall health status
func (h *HealthCheckerImpl) CheckHealth(ctx context.Context) (*ComponentHealth, error) {
	overall := h.GetOverallStatus()
	
	return &ComponentHealth{
		Name:        overall.Name,
		Status:      overall,
		Message:     overall.Message,
		Timestamp:   overall.LastChecked,
		LastChecked: overall.LastChecked,
	}, nil
}

// GetName returns the name of this health checker
func (h *HealthCheckerImpl) GetName() string {
	return "health_checker_impl"
}

// Stop stops the health checker
func (h *HealthCheckerImpl) Stop() error {
	h.cancel()
	h.wg.Wait()
	return nil
}

// runHealthChecks runs health checks periodically
func (h *HealthCheckerImpl) runHealthChecks() {
	defer h.wg.Done()
	
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	
	// Run initial check
	h.performAllChecks()
	
	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.performAllChecks()
		}
	}
}

// performAllChecks executes all health checks
func (h *HealthCheckerImpl) performAllChecks() {
	h.mu.RLock()
	checks := make([]HealthCheck, 0, len(h.checks))
	for _, check := range h.checks {
		checks = append(checks, check)
	}
	h.mu.RUnlock()
	
	// Run checks concurrently
	var wg sync.WaitGroup
	for _, check := range checks {
		wg.Add(1)
		go func(c HealthCheck) {
			defer wg.Done()
			h.performSingleCheck(c)
		}(check)
	}
	
	wg.Wait()
}

// performSingleCheck executes a single health check
func (h *HealthCheckerImpl) performSingleCheck(check HealthCheck) {
	startTime := time.Now()
	
	ctx, cancel := context.WithTimeout(h.ctx, check.Timeout)
	defer cancel()
	
	status := HealthStatus{
		Name:        check.Name,
		Status:      "healthy",
		LastChecked: startTime,
		Critical:    check.Critical,
	}
	
	err := check.CheckFunc(ctx)
	duration := time.Since(startTime)
	status.Duration = duration.String()
	
	if err != nil {
		status.Status = "unhealthy"
		status.Message = err.Error()
		status.Error = err.Error()
		
		log.Log.Error(err, "Health check failed", 
			"check", check.Name, 
			"duration", duration,
			"critical", check.Critical)
	} else {
		status.Message = "Check passed successfully"
		
		log.Log.V(1).Info("Health check passed", 
			"check", check.Name, 
			"duration", duration)
	}
	
	// Handle timeout
	if ctx.Err() == context.DeadlineExceeded {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("Check timed out after %v", check.Timeout)
		status.Error = "timeout"
	}
	
	h.mu.Lock()
	h.results[check.Name] = status
	h.mu.Unlock()
}

// RunCheck executes a specific health check immediately
func (h *HealthCheckerImpl) RunCheck(name string) (*HealthStatus, error) {
	h.mu.RLock()
	check, exists := h.checks[name]
	h.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("health check '%s' not found", name)
	}
	
	h.performSingleCheck(check)
	
	h.mu.RLock()
	status := h.results[name]
	h.mu.RUnlock()
	
	return &status, nil
}

// GetCheckDetails returns detailed information about all checks
func (h *HealthCheckerImpl) GetCheckDetails() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	details := make(map[string]interface{})
	
	for name, check := range h.checks {
		status, exists := h.results[name]
		if !exists {
			status = HealthStatus{
				Name:    name,
				Status:  "unknown",
				Message: "Not yet checked",
			}
		}
		
		details[name] = map[string]interface{}{
			"description": check.Description,
			"timeout":     check.Timeout.String(),
			"critical":    check.Critical,
			"status":      status,
		}
	}
	
	return details
}

// Common health check functions

// DatabaseHealthCheck creates a health check for database connectivity
func DatabaseHealthCheck(name string, pingFunc func(ctx context.Context) error, critical bool) HealthCheck {
	return HealthCheck{
		Name:        name,
		Description: fmt.Sprintf("Database connectivity check for %s", name),
		CheckFunc:   pingFunc,
		Timeout:     10 * time.Second,
		Critical:    critical,
	}
}

// HTTPHealthCheck creates a health check for HTTP endpoints
func HTTPHealthCheck(name, url string, critical bool) HealthCheck {
	return HealthCheck{
		Name:        name,
		Description: fmt.Sprintf("HTTP endpoint health check for %s", url),
		CheckFunc: func(ctx context.Context) error {
			// This would typically use http.Client with context
			// For now, just return nil as implementation would depend on actual HTTP client
			return nil
		},
		Timeout:  15 * time.Second,
		Critical: critical,
	}
}

// KubernetesHealthCheck creates a health check for Kubernetes API connectivity
func KubernetesHealthCheck(name string, checkFunc func(ctx context.Context) error, critical bool) HealthCheck {
	return HealthCheck{
		Name:        name,
		Description: fmt.Sprintf("Kubernetes API connectivity check for %s", name),
		CheckFunc:   checkFunc,
		Timeout:     10 * time.Second,
		Critical:    critical,
	}
}

// CustomHealthCheck creates a custom health check
func CustomHealthCheck(name, description string, checkFunc func(ctx context.Context) error, timeout time.Duration, critical bool) HealthCheck {
	return HealthCheck{
		Name:        name,
		Description: description,
		CheckFunc:   checkFunc,
		Timeout:     timeout,
		Critical:    critical,
	}
}

// MemoryHealthCheck creates a health check for memory usage
func MemoryHealthCheck(name string, maxMemoryMB int64, critical bool) HealthCheck {
	return HealthCheck{
		Name:        name,
		Description: fmt.Sprintf("Memory usage check (max: %d MB)", maxMemoryMB),
		CheckFunc: func(ctx context.Context) error {
			// This would typically check actual memory usage
			// For now, just return nil as implementation would depend on runtime metrics
			return nil
		},
		Timeout:  5 * time.Second,
		Critical: critical,
	}
}

// DiskSpaceHealthCheck creates a health check for disk space
func DiskSpaceHealthCheck(name, path string, minFreePercentage int, critical bool) HealthCheck {
	return HealthCheck{
		Name:        name,
		Description: fmt.Sprintf("Disk space check for %s (min: %d%% free)", path, minFreePercentage),
		CheckFunc: func(ctx context.Context) error {
			// This would typically check actual disk usage
			// For now, just return nil as implementation would depend on filesystem calls
			return nil
		},
		Timeout:  5 * time.Second,
		Critical: critical,
	}
}

// ServiceHealthCheck creates a health check for external services
func ServiceHealthCheck(name string, checkFunc func(ctx context.Context) error, timeout time.Duration, critical bool) HealthCheck {
	return HealthCheck{
		Name:        name,
		Description: fmt.Sprintf("External service health check for %s", name),
		CheckFunc:   checkFunc,
		Timeout:     timeout,
		Critical:    critical,
	}
}

// CompositeHealthChecker combines multiple health checkers
type CompositeHealthChecker struct {
	checkers []HealthChecker
	mu       sync.RWMutex
}

// NewCompositeHealthChecker creates a new composite health checker
func NewCompositeHealthChecker(checkers ...HealthChecker) *CompositeHealthChecker {
	return &CompositeHealthChecker{
		checkers: checkers,
	}
}

// AddChecker adds a health checker
func (c *CompositeHealthChecker) AddChecker(checker HealthChecker) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checkers = append(c.checkers, checker)
}

// CheckHealth performs health check by calling all underlying checkers
func (c *CompositeHealthChecker) CheckHealth(ctx context.Context) (*ComponentHealth, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	allHealthy := true
	var messages []string
	var firstError error
	
	for i, checker := range c.checkers {
		health, err := checker.CheckHealth(ctx)
		if err != nil {
			if firstError == nil {
				firstError = err
			}
			allHealthy = false
			messages = append(messages, fmt.Sprintf("checker_%d (%s): %v", i, checker.GetName(), err))
		} else if health != nil && health.Status.Status != HealthStatusHealthy {
			allHealthy = false
			messages = append(messages, fmt.Sprintf("checker_%d (%s): %s", i, checker.GetName(), health.Message))
		}
	}
	
	status := HealthStatusHealthy
	message := "All composite checks passed"
	if !allHealthy {
		status = HealthStatusUnhealthy
		message = fmt.Sprintf("Some checks failed: %v", messages)
	}
	
	return &ComponentHealth{
		Name: "composite_health_checker",
		Status: HealthStatus{
			Status:      status,
			Message:     message,
			LastChecked: time.Now(),
		},
		Message:   message,
		Timestamp: time.Now(),
	}, firstError
}

// GetName returns the name of this health checker
func (c *CompositeHealthChecker) GetName() string {
	return "composite_health_checker"
}

// Start starts all health checkers
func (c *CompositeHealthChecker) Start(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	for _, checker := range c.checkers {
		if err := checker.Start(ctx); err != nil {
			return fmt.Errorf("failed to start health checker: %w", err)
		}
	}
	
	return nil
}

// Stop stops all health checkers
func (c *CompositeHealthChecker) Stop() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	var lastError error
	for _, checker := range c.checkers {
		if err := checker.Stop(); err != nil {
			log.Log.Error(err, "failed to stop health checker")
			lastError = err
		}
	}
	
	return lastError
}