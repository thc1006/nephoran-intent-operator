package shared

import (
	"os"
	"time"
)

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold    int64         `json:"failure_threshold"`
	FailureRate         float64       `json:"failure_rate"`
	MinimumRequestCount int64         `json:"minimum_request_count"`
	Timeout             time.Duration `json:"timeout"`
	HalfOpenTimeout     time.Duration `json:"half_open_timeout"`
	SuccessThreshold    int64         `json:"success_threshold"`
	HalfOpenMaxRequests int64         `json:"half_open_max_requests"`
	ResetTimeout        time.Duration `json:"reset_timeout"`
	SlidingWindowSize   int           `json:"sliding_window_size"`
	EnableHealthCheck   bool          `json:"enable_health_check"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	HealthCheckTimeout  time.Duration `json:"health_check_timeout"`
	// Advanced features
	EnableAdaptiveTimeout bool `json:"enable_adaptive_timeout"`
	MaxConcurrentRequests int  `json:"max_concurrent_requests"`
}

// GetEnv is a utility function to get environment variables with default values
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
