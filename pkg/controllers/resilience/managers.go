package resilience

import (
	"context"
	"github.com/go-logr/logr"
)

// This file provides stub implementations to avoid conflicts.
// Main implementations are in patterns.go to match expected interfaces.

// Metric types are defined in patterns.go to avoid redeclaration conflicts

// Required constructor functions that are called from patterns.go
func NewBulkheadManager(configs map[string]*BulkheadConfig, logger logr.Logger) *BulkheadManager {
	return &BulkheadManager{}
}

func NewCircuitBreakerManager(configs map[string]*CircuitBreakerConfig, logger logr.Logger) *CircuitBreakerManager {
	return &CircuitBreakerManager{}
}

func NewRateLimiterManager(configs map[string]*RateLimitConfig, logger logr.Logger) *RateLimiterManager {
	return &RateLimiterManager{}
}

// Required methods for BulkheadManager
func (m *BulkheadManager) Start(ctx context.Context) error { return nil }
func (m *BulkheadManager) Stop(ctx context.Context) error  { return nil }

// Required methods for CircuitBreakerManager  
func (m *CircuitBreakerManager) Start(ctx context.Context) error { return nil }
func (m *CircuitBreakerManager) Stop(ctx context.Context) error  { return nil }

// Required methods for RateLimiterManager
func (m *RateLimiterManager) Start(ctx context.Context) error { return nil }
func (m *RateLimiterManager) Stop(ctx context.Context) error  { return nil }
func (m *RateLimiterManager) Allow(key string) bool          { return true }

// Required methods for CircuitBreakerManager
func (m *CircuitBreakerManager) CanExecute(key string) bool           { return true }
func (m *CircuitBreakerManager) RecordResult(key string, success bool, duration ...interface{}) {}

// Required methods for BulkheadManager  
func (m *BulkheadManager) Execute(ctx context.Context, key string, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) { 
	return fn(ctx) 
}
func (m *BulkheadManager) CheckHealth() map[string]interface{} { return map[string]interface{}{} }

// Additional required methods
func (m *CircuitBreakerManager) CheckHealth() map[string]interface{} { return map[string]interface{}{} }
func (m *RateLimiterManager) CheckHealth() map[string]interface{} { return map[string]interface{}{} }

// GetMetrics methods
func (m *BulkheadManager) GetMetrics() *BulkheadMetrics { return &BulkheadMetrics{} }
func (m *CircuitBreakerManager) GetMetrics() *CircuitBreakerMetrics { return &CircuitBreakerMetrics{} }
func (m *RateLimiterManager) GetMetrics() *RateLimitMetrics { return &RateLimitMetrics{} }