// Package monitoring provides synthetic availability monitoring for O-RAN systems
package monitoring

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// AvailabilityStats holds availability statistics
type AvailabilityStats struct {
	CheckID          string        `json:"check_id"`
	TotalChecks      int64         `json:"total_checks"`
	SuccessfulChecks int64         `json:"successful_checks"`
	FailedChecks     int64         `json:"failed_checks"`
	Availability     float64       `json:"availability"`
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	LastCheck        time.Time     `json:"last_check"`
	Uptime           time.Duration `json:"uptime"`
	Downtime         time.Duration `json:"downtime"`
}

// AddCheck adds a synthetic check
func (sm *SyntheticMonitor) AddCheck(check *SyntheticCheck) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if check.ID == "" {
		return fmt.Errorf("check ID cannot be empty")
	}

	if check.Interval == 0 {
		check.Interval = 30 * time.Second
	}

	if check.Timeout == 0 {
		check.Timeout = 10 * time.Second
	}

	sm.checks[check.ID] = check

	sm.logger.Info("Added synthetic check",
		"checkID", check.ID,
		"name", check.Name,
		"type", check.Type,
		"target", check.Target)

	return nil
}

// RemoveCheck removes a synthetic check
func (sm *SyntheticMonitor) RemoveCheck(checkID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.checks[checkID]; !exists {
		return fmt.Errorf("check not found: %s", checkID)
	}

	delete(sm.checks, checkID)
	delete(sm.results, checkID)

	sm.logger.Info("Removed synthetic check", "checkID", checkID)
	return nil
}

// StartMonitoring starts all enabled synthetic checks
func (sm *SyntheticMonitor) StartMonitoring(ctx context.Context) {
	sm.mutex.RLock()
	checks := make([]*SyntheticCheck, 0, len(sm.checks))
	for _, check := range sm.checks {
		if check.Enabled {
			checks = append(checks, check)
		}
	}
	sm.mutex.RUnlock()

	for _, check := range checks {
		go sm.runCheck(ctx, check)
	}

	sm.logger.Info("Started synthetic monitoring", "activeChecks", len(checks))
}

// runCheck runs a single synthetic check continuously
func (sm *SyntheticMonitor) runCheck(ctx context.Context, check *SyntheticCheck) {
	ticker := time.NewTicker(check.Interval)
	defer ticker.Stop()

	sm.logger.Info("Starting synthetic check",
		"checkID", check.ID,
		"interval", check.Interval.String())

	for {
		select {
		case <-ctx.Done():
			sm.logger.Info("Stopping synthetic check", "checkID", check.ID)
			return
		case <-ticker.C:
			result := sm.executeCheck(ctx, check)
			sm.recordResult(result)
		}
	}
}

// executeCheck executes a single check and returns the result
func (sm *SyntheticMonitor) executeCheck(ctx context.Context, check *SyntheticCheck) *CheckResult {
	start := time.Now()
	result := &CheckResult{
		CheckID:   check.ID,
		Timestamp: start,
	}

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, check.Timeout)
	defer cancel()

	switch check.Type {
	case "http", "https":
		result = sm.executeHTTPCheck(checkCtx, check, result)
	case "tcp":
		result = sm.executeTCPCheck(checkCtx, check, result)
	case "grpc":
		result = sm.executeGRPCCheck(checkCtx, check, result)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("unsupported check type: %s", check.Type)
	}

	result.ResponseTime = time.Since(start)

	// Check if response time exceeds threshold
	if check.ThresholdMs > 0 && result.ResponseTime > time.Duration(check.ThresholdMs)*time.Millisecond {
		result.Success = false
		result.Message = fmt.Sprintf("Response time %s exceeds threshold %dms",
			result.ResponseTime, check.ThresholdMs)
	}

	sm.logger.V(1).Info("Executed synthetic check",
		"checkID", check.ID,
		"success", result.Success,
		"responseTime", result.ResponseTime.String(),
		"error", result.Error)

	return result
}

// executeHTTPCheck performs an HTTP check
func (sm *SyntheticMonitor) executeHTTPCheck(ctx context.Context, check *SyntheticCheck, result *CheckResult) *CheckResult {
	req, err := http.NewRequestWithContext(ctx, "GET", check.Target, nil)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	// Add custom headers
	for key, value := range check.Headers {
		req.Header.Set(key, value)
	}

	// Set user agent
	req.Header.Set("User-Agent", "Nephio-Synthetic-Monitor/1.0")

	resp, err := sm.httpClient.Do(req)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("HTTP request failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	// Check expected status code
	expectedCode := check.ExpectedCode
	if expectedCode == 0 {
		expectedCode = 200 // Default expected code
	}

	if resp.StatusCode != expectedCode {
		result.Success = false
		result.Error = fmt.Sprintf("unexpected status code: %d, expected: %d",
			resp.StatusCode, expectedCode)
		return result
	}

	// TODO: Check expected body if specified
	if check.ExpectedBody != "" {
		// Implementation for body check would go here
	}

	result.Success = true
	result.Message = "HTTP check successful"
	return result
}

// executeTCPCheck performs a TCP connection check
func (sm *SyntheticMonitor) executeTCPCheck(ctx context.Context, check *SyntheticCheck, result *CheckResult) *CheckResult {
	// TODO: Implement TCP check
	result.Success = false
	result.Error = "TCP checks not implemented yet"
	return result
}

// executeGRPCCheck performs a gRPC health check
func (sm *SyntheticMonitor) executeGRPCCheck(ctx context.Context, check *SyntheticCheck, result *CheckResult) *CheckResult {
	// TODO: Implement gRPC check
	result.Success = false
	result.Error = "gRPC checks not implemented yet"
	return result
}

// recordResult records a check result
func (sm *SyntheticMonitor) recordResult(result *CheckResult) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.results[result.CheckID] = result

	// Calculate availability (simplified - should use sliding window)
	// For now, just use the current success status
	if result.Success {
		result.Availability = 100.0
	} else {
		result.Availability = 0.0
	}

	sm.logger.V(1).Info("Recorded check result",
		"checkID", result.CheckID,
		"success", result.Success,
		"responseTime", result.ResponseTime.String(),
		"availability", result.Availability)
}

// GetCheckResult returns the latest result for a check (CheckResult type fix)
func (sm *SyntheticMonitor) GetCheckResult(checkID string) (*CheckResult, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	result, exists := sm.results[checkID]
	if !exists {
		return nil, fmt.Errorf("no results found for check: %s", checkID)
	}

	// Return a copy to avoid concurrent access issues
	resultCopy := *result
	return &resultCopy, nil
}

// GetAllResults returns all current check results
func (sm *SyntheticMonitor) GetAllResults() map[string]*CheckResult {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	results := make(map[string]*CheckResult)
	for checkID, result := range sm.results {
		resultCopy := *result
		results[checkID] = &resultCopy
	}

	return results
}

// GetAvailabilityStats returns availability statistics for a check
func (sm *SyntheticMonitor) GetAvailabilityStats(checkID string) (*AvailabilityStats, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	result, exists := sm.results[checkID]
	if !exists {
		return nil, fmt.Errorf("no results found for check: %s", checkID)
	}

	// TODO: Implement proper stats calculation with historical data
	// For now, return basic stats based on current result
	stats := &AvailabilityStats{
		CheckID:          checkID,
		TotalChecks:      1,
		SuccessfulChecks: 0,
		FailedChecks:     0,
		Availability:     result.Availability,
		AvgResponseTime:  result.ResponseTime,
		LastCheck:        result.Timestamp,
	}

	if result.Success {
		stats.SuccessfulChecks = 1
		stats.Uptime = result.ResponseTime
	} else {
		stats.FailedChecks = 1
		stats.Downtime = result.ResponseTime
	}

	return stats, nil
}

// ListChecks returns all configured checks
func (sm *SyntheticMonitor) ListChecks() []*SyntheticCheck {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	checks := make([]*SyntheticCheck, 0, len(sm.checks))
	for _, check := range sm.checks {
		checkCopy := *check
		checks = append(checks, &checkCopy)
	}

	return checks
}

// UpdateCheck updates an existing check
func (sm *SyntheticMonitor) UpdateCheck(check *SyntheticCheck) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.checks[check.ID]; !exists {
		return fmt.Errorf("check not found: %s", check.ID)
	}

	sm.checks[check.ID] = check

	sm.logger.Info("Updated synthetic check",
		"checkID", check.ID,
		"name", check.Name,
		"enabled", check.Enabled)

	return nil
}

// Shutdown gracefully shuts down the synthetic monitor
func (sm *SyntheticMonitor) Shutdown(ctx context.Context) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Clear all checks and results
	sm.checks = make(map[string]*SyntheticCheck)
	sm.results = make(map[string]*CheckResult)

	sm.logger.Info("Synthetic monitor shutdown completed")
	return nil
}

// Note: SyntheticMonitor, SyntheticCheck, and CheckResult types are defined in types.go to avoid duplicates

// NewSyntheticMonitor creates a new synthetic monitor
func NewSyntheticMonitor(logger *slog.Logger) *SyntheticMonitor {
	if logger == nil {
		logger = slog.Default()
	}

	return &SyntheticMonitor{
		checks:  make(map[string]*SyntheticCheck),
		results: make(map[string]*CheckResult),
		logger:  logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}
