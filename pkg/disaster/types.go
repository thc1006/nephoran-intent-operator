package disaster

import (
	"time"
)

// FailoverResult represents the result of a failover operation
type FailoverResult struct {
	ServiceID string
	Success   bool
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Error     error
	Details   map[string]interface{}
}

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	ServiceID    string
	Status       string
	LastCheck    time.Time
	ResponseTime time.Duration
	Error        error
	Details      map[string]interface{}
}

// FailoverStatus represents the current failover status of a service
type FailoverStatus struct {
	ServiceID      string
	Status         string
	ActiveEndpoint string
	LastFailover   time.Time
	FailoverCount  int
	Details        map[string]interface{}
}

// ServiceConfig defines a service configuration for failover
type ServiceConfig struct {
	ID          string
	Name        string
	Primary     string
	Backup      string
	Priority    int
	HealthCheck string
	Enabled     bool
}

// FailoverMetrics tracks failover statistics
type FailoverMetrics struct {
	FailoversExecuted     int64
	HealthChecksPerformed int64
	FailoverFailures      int64
	AverageFailoverTime   time.Duration
	LastFailoverTime      time.Time
}
