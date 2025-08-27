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

package resilience

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
)

// NewTimeoutManager creates a new timeout manager
func NewTimeoutManager(configs map[string]*TimeoutConfig, logger logr.Logger) *TimeoutManager {
	tm := &TimeoutManager{
		configs: make(map[string]*TimeoutConfig),
		metrics: &TimeoutMetrics{
			OperationsByTimeout: make(map[time.Duration]int64),
		},
		logger: logger.WithName("timeout-manager"),
	}

	// Copy configurations
	for name, config := range configs {
		tm.configs[name] = config
	}

	// Add default configuration if none exists
	if len(tm.configs) == 0 {
		tm.configs["default"] = &TimeoutConfig{
			Name:           "default",
			DefaultTimeout: 30 * time.Second,
			MaxTimeout:     5 * time.Minute,
			MinTimeout:     1 * time.Second,
		}
	}

	return tm
}

// Start starts the timeout manager
func (tm *TimeoutManager) Start(ctx context.Context) {
	tm.logger.Info("Starting timeout manager")

	// Start cleanup routine
	go tm.cleanupRoutine(ctx)

	// Start adaptive adjustment routine
	go tm.adaptiveAdjustmentRoutine(ctx)
}

// Stop stops the timeout manager
func (tm *TimeoutManager) Stop() {
	tm.logger.Info("Stopping timeout manager")

	// Cancel all active operations
	tm.operations.Range(func(key, value interface{}) bool {
		operation := value.(*TimeoutOperation)
		operation.CancelFunc()
		return true
	})
}

// ApplyTimeout applies timeout to a context
func (tm *TimeoutManager) ApplyTimeout(ctx context.Context, operationName string) context.Context {
	config := tm.getConfigForOperation(operationName)
	timeout := tm.calculateTimeout(operationName, config)

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)

	// Create timeout operation tracker
	operation := &TimeoutOperation{
		ID:            fmt.Sprintf("%s_%d", operationName, time.Now().UnixNano()),
		StartTime:     time.Now(),
		Timeout:       timeout,
		Context:       timeoutCtx,
		CancelFunc:    cancel,
		CompletedChan: make(chan struct{}),
		TimeoutChan:   make(chan struct{}),
	}

	// Store operation
	tm.operations.Store(operation.ID, operation)

	// Monitor timeout
	go tm.monitorTimeout(operation)

	// Update metrics
	atomic.AddInt64(&tm.metrics.TotalOperations, 1)
	tm.updateTimeoutMetrics(timeout)

	return timeoutCtx
}

// ExecuteWithTimeout executes a function with timeout
func (tm *TimeoutManager) ExecuteWithTimeout(ctx context.Context, operationName string, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	timeoutCtx := tm.ApplyTimeout(ctx, operationName)

	resultChan := make(chan struct {
		result interface{}
		err    error
	}, 1)

	// Execute function in goroutine
	go func() {
		result, err := fn(timeoutCtx)
		resultChan <- struct {
			result interface{}
			err    error
		}{result, err}
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result.result, result.err

	case <-timeoutCtx.Done():
		if timeoutCtx.Err() == context.DeadlineExceeded {
			atomic.AddInt64(&tm.metrics.TimeoutOperations, 1)
			return nil, fmt.Errorf("operation %s timed out after %v", operationName, tm.getConfigForOperation(operationName).DefaultTimeout)
		}
		return nil, timeoutCtx.Err()
	}
}

// calculateTimeout calculates the appropriate timeout for an operation
func (tm *TimeoutManager) calculateTimeout(operationName string, config *TimeoutConfig) time.Duration {
	timeout := config.DefaultTimeout

	// Apply gradient based on system load if enabled
	if config.TimeoutGradient {
		loadFactor := tm.getCurrentLoadFactor()
		adjustedTimeout := time.Duration(float64(timeout) * (1.0 + loadFactor))

		if adjustedTimeout > config.MaxTimeout {
			timeout = config.MaxTimeout
		} else if adjustedTimeout < config.MinTimeout {
			timeout = config.MinTimeout
		} else {
			timeout = adjustedTimeout
		}
	}

	// Apply adaptive adjustment if enabled
	if config.AdaptiveTimeout {
		adaptiveTimeout := tm.getAdaptiveTimeout(operationName)
		if adaptiveTimeout > 0 {
			// Use weighted average of configured and adaptive timeout
			timeout = time.Duration((float64(timeout) + float64(adaptiveTimeout)) / 2)
		}
	}

	// Ensure within bounds
	if timeout > config.MaxTimeout {
		timeout = config.MaxTimeout
	}
	if timeout < config.MinTimeout {
		timeout = config.MinTimeout
	}

	return timeout
}

// getCurrentLoadFactor returns current system load factor (0.0 to 1.0+)
func (tm *TimeoutManager) getCurrentLoadFactor() float64 {
	// This would implement actual load detection
	// For now, return a simulated value
	return 0.2 // 20% additional load
}

// getAdaptiveTimeout returns the learned optimal timeout for an operation
func (tm *TimeoutManager) getAdaptiveTimeout(operationName string) time.Duration {
	// This would implement machine learning-based timeout prediction
	// For now, return 0 to use configured timeout
	return 0
}

// getConfigForOperation returns the configuration for a specific operation
func (tm *TimeoutManager) getConfigForOperation(operationName string) *TimeoutConfig {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	// Try exact match first
	if config, exists := tm.configs[operationName]; exists {
		return config
	}

	// Try pattern matching (e.g., "llm_*" for any LLM operation)
	for name, config := range tm.configs {
		if tm.matchesPattern(operationName, name) {
			return config
		}
	}

	// Return default configuration
	return tm.configs["default"]
}

// matchesPattern checks if an operation name matches a configuration pattern
func (tm *TimeoutManager) matchesPattern(operationName, pattern string) bool {
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}

	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(operationName) >= len(prefix) && operationName[:len(prefix)] == prefix
	}

	return operationName == pattern
}

// monitorTimeout monitors a timeout operation
func (tm *TimeoutManager) monitorTimeout(operation *TimeoutOperation) {
	defer func() {
		tm.operations.Delete(operation.ID)
	}()

	select {
	case <-operation.Context.Done():
		if operation.Context.Err() == context.DeadlineExceeded {
			// Operation timed out
			close(operation.TimeoutChan)
			tm.handleTimeout(operation)
		} else {
			// Operation completed or cancelled
			close(operation.CompletedChan)
		}

	case <-operation.CompletedChan:
		// Operation completed successfully
		tm.handleCompletion(operation)
	}
}

// handleTimeout handles a timeout event
func (tm *TimeoutManager) handleTimeout(operation *TimeoutOperation) {
	duration := time.Since(operation.StartTime)

	tm.logger.Info("Operation timed out",
		"operationId", operation.ID,
		"timeout", operation.Timeout,
		"actualDuration", duration)

	// Update metrics
	atomic.AddInt64(&tm.metrics.TimeoutOperations, 1)

	// Record for adaptive learning
	tm.recordTimeoutEvent(operation, duration)
}

// handleCompletion handles successful operation completion
func (tm *TimeoutManager) handleCompletion(operation *TimeoutOperation) {
	duration := time.Since(operation.StartTime)

	tm.logger.Info("Operation completed within timeout",
		"operationId", operation.ID,
		"timeout", operation.Timeout,
		"actualDuration", duration)

	// Record for adaptive learning
	tm.recordCompletionEvent(operation, duration)
}

// recordTimeoutEvent records a timeout event for learning
func (tm *TimeoutManager) recordTimeoutEvent(operation *TimeoutOperation, duration time.Duration) {
	// This would implement learning from timeout events
	// to adjust future timeout values
}

// recordCompletionEvent records a completion event for learning
func (tm *TimeoutManager) recordCompletionEvent(operation *TimeoutOperation, duration time.Duration) {
	// This would implement learning from completion events
	// to optimize timeout values
}

// updateTimeoutMetrics updates timeout-related metrics
func (tm *TimeoutManager) updateTimeoutMetrics(timeout time.Duration) {
	tm.metrics.mutex.Lock()
	defer tm.metrics.mutex.Unlock()

	// Update average timeout
	totalOps := atomic.LoadInt64(&tm.metrics.TotalOperations)
	if totalOps > 0 {
		prevTotal := time.Duration(totalOps-1) * tm.metrics.AverageTimeout
		tm.metrics.AverageTimeout = (prevTotal + timeout) / time.Duration(totalOps)
	}

	// Update timeout distribution
	tm.metrics.OperationsByTimeout[timeout]++

	// Update timeout rate
	timeoutOps := atomic.LoadInt64(&tm.metrics.TimeoutOperations)
	if totalOps > 0 {
		tm.metrics.TimeoutRate = float64(timeoutOps) / float64(totalOps)
	}
}

// cleanupRoutine periodically cleans up completed operations
func (tm *TimeoutManager) cleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.cleanupCompletedOperations()

		case <-ctx.Done():
			return
		}
	}
}

// cleanupCompletedOperations removes completed operations from memory
func (tm *TimeoutManager) cleanupCompletedOperations() {
	cutoff := time.Now().Add(-5 * time.Minute) // Keep 5 minutes of history

	toDelete := make([]interface{}, 0)

	tm.operations.Range(func(key, value interface{}) bool {
		operation := value.(*TimeoutOperation)
		if operation.StartTime.Before(cutoff) {
			toDelete = append(toDelete, key)
		}
		return true
	})

	for _, key := range toDelete {
		tm.operations.Delete(key)
	}

	if len(toDelete) > 0 {
		tm.logger.Info("Cleaned up completed operations", "count", len(toDelete))
	}
}

// adaptiveAdjustmentRoutine periodically adjusts timeout configurations
func (tm *TimeoutManager) adaptiveAdjustmentRoutine(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // Adjust every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tm.performAdaptiveAdjustments()

		case <-ctx.Done():
			return
		}
	}
}

// performAdaptiveAdjustments performs adaptive timeout adjustments
func (tm *TimeoutManager) performAdaptiveAdjustments() {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	for name, config := range tm.configs {
		if !config.AdaptiveTimeout {
			continue
		}

		// Analyze recent performance for this configuration
		adjustmentNeeded := tm.analyzePerformanceForConfig(name, config)

		if adjustmentNeeded != 0 {
			oldTimeout := config.DefaultTimeout
			adjustment := time.Duration(float64(oldTimeout) * adjustmentNeeded)
			newTimeout := oldTimeout + adjustment

			// Ensure within bounds
			if newTimeout > config.MaxTimeout {
				newTimeout = config.MaxTimeout
			}
			if newTimeout < config.MinTimeout {
				newTimeout = config.MinTimeout
			}

			if newTimeout != oldTimeout {
				config.DefaultTimeout = newTimeout
				atomic.AddInt64(&tm.metrics.AdaptiveAdjustments, 1)

				tm.logger.Info("Adjusted timeout configuration",
					"name", name,
					"oldTimeout", oldTimeout,
					"newTimeout", newTimeout,
					"adjustment", adjustmentNeeded)
			}
		}
	}
}

// analyzePerformanceForConfig analyzes performance and returns adjustment factor
func (tm *TimeoutManager) analyzePerformanceForConfig(name string, config *TimeoutConfig) float64 {
	// This would implement sophisticated performance analysis
	// For now, return no adjustment
	return 0.0
}

// GetMetrics returns timeout manager metrics
func (tm *TimeoutManager) GetMetrics() *TimeoutMetrics {
	tm.metrics.mutex.RLock()
	defer tm.metrics.mutex.RUnlock()

	// Create a copy to prevent concurrent access
	operationsByTimeout := make(map[time.Duration]int64)
	for k, v := range tm.metrics.OperationsByTimeout {
		operationsByTimeout[k] = v
	}

	metricsCopy := &TimeoutMetrics{
		TotalOperations:     tm.metrics.TotalOperations,
		TimeoutOperations:   tm.metrics.TimeoutOperations,
		AverageTimeout:      tm.metrics.AverageTimeout,
		TimeoutRate:         tm.metrics.TimeoutRate,
		AdaptiveAdjustments: tm.metrics.AdaptiveAdjustments,
		OperationsByTimeout: operationsByTimeout,
	}

	return metricsCopy
}

// GetActiveOperations returns the count of currently active operations
func (tm *TimeoutManager) GetActiveOperations() int {
	count := 0
	tm.operations.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// GetOperationStatus returns the status of a specific operation
func (tm *TimeoutManager) GetOperationStatus(operationID string) (*TimeoutOperation, bool) {
	if op, exists := tm.operations.Load(operationID); exists {
		operation := op.(*TimeoutOperation)

		operation.mutex.RLock()
		defer operation.mutex.RUnlock()

		// Return a copy to prevent concurrent access issues
		operationCopy := &TimeoutOperation{
			ID:            operation.ID,
			StartTime:     operation.StartTime,
			Timeout:       operation.Timeout,
			Context:       operation.Context,
			CancelFunc:    operation.CancelFunc,
			CompletedChan: operation.CompletedChan,
			TimeoutChan:   operation.TimeoutChan,
		}
		return operationCopy, true
	}

	return nil, false
}

// UpdateConfiguration updates timeout configuration for an operation type
func (tm *TimeoutManager) UpdateConfiguration(name string, config *TimeoutConfig) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.configs[name] = config

	tm.logger.Info("Updated timeout configuration",
		"name", name,
		"defaultTimeout", config.DefaultTimeout,
		"maxTimeout", config.MaxTimeout,
		"minTimeout", config.MinTimeout)
}

// GetConfiguration returns the current configuration for an operation type
func (tm *TimeoutManager) GetConfiguration(name string) (*TimeoutConfig, bool) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	if config, exists := tm.configs[name]; exists {
		// Return a copy to prevent concurrent modification
		configCopy := *config
		return &configCopy, true
	}

	return nil, false
}

// ListConfigurations returns all current timeout configurations
func (tm *TimeoutManager) ListConfigurations() map[string]*TimeoutConfig {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	configs := make(map[string]*TimeoutConfig)
	for name, config := range tm.configs {
		// Return copies to prevent concurrent modification
		configCopy := *config
		configs[name] = &configCopy
	}

	return configs
}

// SetGlobalTimeout sets a global timeout that applies to all operations
func (tm *TimeoutManager) SetGlobalTimeout(timeout time.Duration) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	for name, config := range tm.configs {
		if config.DefaultTimeout > timeout {
			config.DefaultTimeout = timeout
			tm.logger.Info("Adjusted configuration for global timeout",
				"name", name,
				"newTimeout", timeout)
		}
	}
}

// IsHealthy returns whether the timeout manager is healthy
func (tm *TimeoutManager) IsHealthy() bool {
	// Check if timeout rate is within acceptable bounds
	timeoutRate := tm.metrics.TimeoutRate

	// Consider unhealthy if more than 10% of operations are timing out
	return timeoutRate <= 0.1
}

// GetHealthReport returns a detailed health report
func (tm *TimeoutManager) GetHealthReport() map[string]interface{} {
	activeOps := tm.GetActiveOperations()
	metrics := tm.GetMetrics()

	return map[string]interface{}{
		"healthy":              tm.IsHealthy(),
		"active_operations":    activeOps,
		"total_operations":     metrics.TotalOperations,
		"timeout_operations":   metrics.TimeoutOperations,
		"timeout_rate":         metrics.TimeoutRate,
		"average_timeout":      metrics.AverageTimeout,
		"adaptive_adjustments": metrics.AdaptiveAdjustments,
		"configurations":       len(tm.configs),
	}
}
