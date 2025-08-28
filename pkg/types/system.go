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

package types

import (
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

// ServiceConfig represents service configuration
type ServiceConfig struct {
	Name        string                 `json:"name"`
	Port        int                    `json:"port"`
	Host        string                 `json:"host"`
	TLS         bool                   `json:"tls"`
	Timeout     time.Duration          `json:"timeout"`
	Retries     int                    `json:"retries"`
	CircuitBreaker *CircuitBreakerConfig `json:"circuit_breaker,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// HealthStatus represents health status
type HealthStatus struct {
	Healthy     bool                   `json:"healthy"`
	Message     string                 `json:"message"`
	LastCheck   time.Time              `json:"last_check"`
	Details     map[string]interface{} `json:"details"`
}

// NetworkIntent represents a network intent
type NetworkIntent struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Priority        int                    `json:"priority"`
	Description     string                 `json:"description"`
	Parameters      map[string]interface{} `json:"parameters"`
	TargetResources []string               `json:"target_resources"`
	Constraints     map[string]interface{} `json:"constraints"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Status          string                 `json:"status"`
}

// ScalingIntent represents a scaling intent
type ScalingIntent struct {
	ID            string                 `json:"id"`
	ResourceType  string                 `json:"resource_type"`
	ResourceName  string                 `json:"resource_name"`
	TargetScale   int                    `json:"target_scale"`
	CurrentScale  int                    `json:"current_scale"`
	ScaleDirection string                `json:"scale_direction"` // "up", "down", "auto"
	Reason        string                 `json:"reason"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	CompletedAt   *time.Time             `json:"completed_at,omitempty"`
	Status        string                 `json:"status"`
}