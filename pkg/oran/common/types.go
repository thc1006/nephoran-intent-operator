// Package common provides shared types and structures used across all O-RAN interfaces
// This includes common health check types, status definitions, and other shared data structures
package common

import (
	
	"encoding/json"
"time"
)

// HealthCheck represents health status information for O-RAN services
// This unified type is used across all O-RAN interfaces (A1, O1, O2, E2)
type HealthCheck struct {
	Status     string                 `json:"status" validate:"required,oneof=UP DOWN DEGRADED"`
	Timestamp  time.Time              `json:"timestamp"`
	Version    string                 `json:"version,omitempty"`
	Uptime     time.Duration          `json:"uptime,omitempty"`
	Components json.RawMessage `json:"components,omitempty"`
	Checks     []ComponentCheck       `json:"checks,omitempty"`
	// O2-specific fields (optional for other interfaces)
	Services  []ServiceStatus        `json:"services,omitempty"`
	Resources *ResourceHealthSummary `json:"resources,omitempty"`
}

// ComponentCheck represents individual component health status
// Used for detailed health reporting across all O-RAN interfaces
type ComponentCheck struct {
	Name      string                 `json:"name" validate:"required"`
	Status    string                 `json:"status" validate:"required,oneof=UP DOWN DEGRADED"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration,omitempty"`
	Details   json.RawMessage `json:"details,omitempty"`
	// O2-specific field (optional for other interfaces)
	CheckType string `json:"check_type,omitempty"` // connectivity, resource, dependency
}

// ServiceStatus represents the status of external service dependencies
// Used primarily by O2 IMS but can be used by other interfaces
type ServiceStatus struct {
	ServiceName string        `json:"service_name"`
	Status      string        `json:"status"`
	Endpoint    string        `json:"endpoint,omitempty"`
	LastCheck   time.Time     `json:"last_check"`
	Latency     time.Duration `json:"latency,omitempty"`
	Error       string        `json:"error,omitempty"`
}

// ResourceHealthSummary provides a summary of resource health across the infrastructure
// Used primarily by O2 IMS for infrastructure health reporting
type ResourceHealthSummary struct {
	TotalResources     int `json:"total_resources"`
	HealthyResources   int `json:"healthy_resources"`
	DegradedResources  int `json:"degraded_resources"`
	UnhealthyResources int `json:"unhealthy_resources"`
	UnknownResources   int `json:"unknown_resources"`
}

// Common health status constants used across all O-RAN interfaces
const (
	StatusUP       = "UP"
	StatusDown     = "DOWN"
	StatusDegraded = "DEGRADED"
)

// Common component check types
const (
	CheckTypeConnectivity = "connectivity"
	CheckTypeResource     = "resource"
	CheckTypeDependency   = "dependency"
)

// IsHealthy returns true if the health check status is UP
func (hc *HealthCheck) IsHealthy() bool {
	return hc.Status == StatusUP
}

// IsDegraded returns true if the health check status is DEGRADED
func (hc *HealthCheck) IsDegraded() bool {
	return hc.Status == StatusDegraded
}

// IsDown returns true if the health check status is DOWN
func (hc *HealthCheck) IsDown() bool {
	return hc.Status == StatusDown
}

// IsHealthy returns true if the component check status is UP
func (cc *ComponentCheck) IsHealthy() bool {
	return cc.Status == StatusUP
}

// IsDegraded returns true if the component check status is DEGRADED
func (cc *ComponentCheck) IsDegraded() bool {
	return cc.Status == StatusDegraded
}

// IsDown returns true if the component check status is DOWN
func (cc *ComponentCheck) IsDown() bool {
	return cc.Status == StatusDown
}

// GetHealthyPercentage calculates the percentage of healthy resources
func (rhs *ResourceHealthSummary) GetHealthyPercentage() float64 {
	if rhs.TotalResources == 0 {
		return 100.0
	}
	return float64(rhs.HealthyResources) / float64(rhs.TotalResources) * 100.0
}

// GetDegradedPercentage calculates the percentage of degraded resources
func (rhs *ResourceHealthSummary) GetDegradedPercentage() float64 {
	if rhs.TotalResources == 0 {
		return 0.0
	}
	return float64(rhs.DegradedResources) / float64(rhs.TotalResources) * 100.0
}

// GetUnhealthyPercentage calculates the percentage of unhealthy resources
func (rhs *ResourceHealthSummary) GetUnhealthyPercentage() float64 {
	if rhs.TotalResources == 0 {
		return 0.0
	}
	return float64(rhs.UnhealthyResources) / float64(rhs.TotalResources) * 100.0
}
