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

package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// ErrorTracker provides error tracking capabilities.

// This is a simplified version focused on the core functionality needed by tests.

type ErrorTracker struct {
	config *ErrorTrackingConfig

	logger logr.Logger

	mutex sync.RWMutex

	started bool

	stopChan chan struct{}

	// Error statistics.

	errorCounts map[string]int64

	errorsByType map[string]int64

	lastError time.Time
}

// ErrorTrackingConfig holds configuration for error tracking.

type ErrorTrackingConfig struct {
	EnablePrometheus bool `json:"enablePrometheus"`

	EnableOpenTelemetry bool `json:"enableOpenTelemetry"`

	AlertingEnabled bool `json:"alertingEnabled"`

	DashboardEnabled bool `json:"dashboardEnabled"`

	ReportsEnabled bool `json:"reportsEnabled"`
}

// ErrorSummary provides a summary of errors.

type ErrorSummary struct {
	TotalErrors int64 `json:"totalErrors"`

	ErrorsByType map[string]int64 `json:"errorsByType"`

	ErrorsByPattern map[string]int64 `json:"errorsByPattern"`

	LastErrorTime time.Time `json:"lastErrorTime"`

	ErrorRate float64 `json:"errorRate"`
}

// NewErrorTracker creates a new error tracker.

func NewErrorTracker(config *ErrorTrackingConfig, logger logr.Logger) (*ErrorTracker, error) {

	if config == nil {

		config = &ErrorTrackingConfig{

			EnablePrometheus: false,

			EnableOpenTelemetry: false,

			AlertingEnabled: false,

			DashboardEnabled: false,

			ReportsEnabled: false,
		}

	}

	return &ErrorTracker{

		config: config,

		logger: logger.WithName("error-tracker"),

		stopChan: make(chan struct{}),

		errorCounts: make(map[string]int64),

		errorsByType: make(map[string]int64),
	}, nil

}

// Start starts the error tracker.

func (et *ErrorTracker) Start(ctx context.Context) error {

	et.mutex.Lock()

	defer et.mutex.Unlock()

	if et.started {

		return fmt.Errorf("error tracker already started")

	}

	et.started = true

	et.logger.Info("Error tracker started")

	return nil

}

// Stop stops the error tracker.

func (et *ErrorTracker) Stop() error {

	et.mutex.Lock()

	defer et.mutex.Unlock()

	if !et.started {

		return nil

	}

	close(et.stopChan)

	et.started = false

	et.logger.Info("Error tracker stopped")

	return nil

}

// TrackError tracks an error.

func (et *ErrorTracker) TrackError(ctx context.Context, errorType, errorMessage string) {

	et.mutex.Lock()

	defer et.mutex.Unlock()

	et.errorCounts[errorType]++

	et.errorsByType[errorType]++

	et.lastError = time.Now()

	et.logger.Info("Tracked error", "type", errorType, "message", errorMessage)

}

// GetErrorSummary returns a summary of errors.

func (et *ErrorTracker) GetErrorSummary() *ErrorSummary {

	et.mutex.RLock()

	defer et.mutex.RUnlock()

	total := int64(0)

	errorsByType := make(map[string]int64)

	for errorType, count := range et.errorsByType {

		errorsByType[errorType] = count

		total += count

	}

	return &ErrorSummary{

		TotalErrors: total,

		ErrorsByType: errorsByType,

		ErrorsByPattern: make(map[string]int64), // Simplified for now

		LastErrorTime: et.lastError,

		ErrorRate: 0.0, // Would calculate based on time window

	}

}

// GetErrorsByPattern returns errors matching a pattern.

func (et *ErrorTracker) GetErrorsByPattern(pattern string) []map[string]interface{} {

	et.mutex.RLock()

	defer et.mutex.RUnlock()

	// Simplified implementation - just return empty slice for now.

	// In a full implementation, this would search through stored error events.

	return make([]map[string]interface{}, 0)

}

// GetMetrics returns error tracking metrics.

func (et *ErrorTracker) GetMetrics() map[string]interface{} {

	et.mutex.RLock()

	defer et.mutex.RUnlock()

	return map[string]interface{}{

		"total_errors": len(et.errorCounts),

		"errors_by_type": et.errorsByType,

		"last_error_time": et.lastError,

		"started": et.started,
	}

}

// HealthCheck returns the health status of the error tracker.

func (et *ErrorTracker) HealthCheck() error {

	et.mutex.RLock()

	defer et.mutex.RUnlock()

	if !et.started {

		return fmt.Errorf("error tracker not started")

	}

	return nil

}
