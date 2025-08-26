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

package dependencies

import (
	"time"
)

// Final missing types from resolver.go

// ConflictStrategy represents a strategy for handling conflicts
type ConflictStrategy string

const (
	ConflictStrategyFail      ConflictStrategy = "fail"
	ConflictStrategyResolve   ConflictStrategy = "resolve"
	ConflictStrategyIgnore    ConflictStrategy = "ignore"
	ConflictStrategyInteractive ConflictStrategy = "interactive"
)

// ConflictResolution represents the result of conflict resolution
type ConflictResolution struct {
	ConflictID       string                      `json:"conflictId"`
	ResolutionMethod string                      `json:"resolutionMethod"`
	ResolvedPackages []*PackageReference         `json:"resolvedPackages"`
	Strategy         *ConflictResolutionStrategy `json:"strategy"`
	Success          bool                        `json:"success"`
	Resolution       string                      `json:"resolution,omitempty"`
	Impact           string                      `json:"impact,omitempty"`
	ResolvedAt       time.Time                   `json:"resolvedAt"`
	ResolutionTime   time.Duration               `json:"resolutionTime"`
}

// RollbackPlan represents a plan for rolling back changes
type RollbackPlan struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Description      string              `json:"description"`
	OriginalState    []*PackageReference `json:"originalState"`
	ChangesToRevert  []string            `json:"changesToRevert"`
	RollbackSteps    []string            `json:"rollbackSteps"`
	EstimatedTime    time.Duration       `json:"estimatedTime"`
	RiskAssessment   string              `json:"riskAssessment"`
	Prerequisites    []string            `json:"prerequisites,omitempty"`
	Validation       []string            `json:"validation,omitempty"`
	CreatedAt        time.Time           `json:"createdAt"`
}

// RollbackResult represents the result of a rollback operation
type RollbackResult struct {
	PlanID         string        `json:"planId"`
	Success        bool          `json:"success"`
	RolledBackTo   time.Time     `json:"rolledBackTo"`
	ExecutionTime  time.Duration `json:"executionTime"`
	StepsCompleted int           `json:"stepsCompleted"`
	StepsFailed    int           `json:"stepsFailed"`
	Errors         []string      `json:"errors,omitempty"`
	Warnings       []string      `json:"warnings,omitempty"`
	FinalState     []*PackageReference `json:"finalState"`
	ExecutedAt     time.Time     `json:"executedAt"`
}

// CacheStats represents caching statistics
type CacheStats struct {
	HitCount    int64         `json:"hitCount"`
	MissCount   int64         `json:"missCount"`
	TotalCount  int64         `json:"totalCount"`
	HitRate     float64       `json:"hitRate"`
	Size        int64         `json:"size"`
	MaxSize     int64         `json:"maxSize"`
	Evictions   int64         `json:"evictions"`
	LastUpdated time.Time     `json:"lastUpdated"`
	AvgLookupTime time.Duration `json:"avgLookupTime,omitempty"`
}

// ResolverHealth represents the health status of the resolver
type ResolverHealth struct {
	Status            string                  `json:"status"` // "healthy", "degraded", "unhealthy"
	Uptime            time.Duration           `json:"uptime"`
	LastHealthCheck   time.Time               `json:"lastHealthCheck"`
	ComponentHealth   map[string]string       `json:"componentHealth"`
	PerformanceMetrics *ResolverPerformanceMetrics `json:"performanceMetrics,omitempty"`
	Issues            []string                `json:"issues,omitempty"`
	Warnings          []string                `json:"warnings,omitempty"`
	Version           string                  `json:"version,omitempty"`
}

// ResolverPerformanceMetrics represents performance metrics for the resolver
type ResolverPerformanceMetrics struct {
	AvgResolutionTime   time.Duration `json:"avgResolutionTime"`
	MaxResolutionTime   time.Duration `json:"maxResolutionTime"`
	MinResolutionTime   time.Duration `json:"minResolutionTime"`
	ThroughputPerSecond float64       `json:"throughputPerSecond"`
	ErrorRate           float64       `json:"errorRate"`
	CacheEfficiency     float64       `json:"cacheEfficiency"`
	MemoryUsageMB       float64       `json:"memoryUsageMB"`
	CPUUsagePercent     float64       `json:"cpuUsagePercent"`
}

// ResolverMetrics represents comprehensive resolver metrics
type ResolverMetrics struct {
	// Resolution metrics
	TotalResolutions     int64         `json:"totalResolutions"`
	SuccessfulResolutions int64        `json:"successfulResolutions"`
	FailedResolutions    int64         `json:"failedResolutions"`
	AvgResolutionTime    time.Duration `json:"avgResolutionTime"`
	MaxResolutionTime    time.Duration `json:"maxResolutionTime"`
	
	// Conflict metrics
	ConflictsDetected    int64 `json:"conflictsDetected"`
	ConflictsResolved    int64 `json:"conflictsResolved"`
	ConflictsUnresolved  int64 `json:"conflictsUnresolved"`
	
	// Cache metrics
	CacheStats           *CacheStats `json:"cacheStats,omitempty"`
	
	// Performance metrics
	Performance          *ResolverPerformanceMetrics `json:"performance,omitempty"`
	
	// Error tracking
	ErrorCounts          map[string]int64 `json:"errorCounts,omitempty"`
	WarningCounts        map[string]int64 `json:"warningCounts,omitempty"`
	
	// Timing
	LastResetAt          time.Time `json:"lastResetAt"`
	CollectionStarted    time.Time `json:"collectionStarted"`
}

// ConstraintSolverMetrics represents metrics for the constraint solver
type ConstraintSolverMetrics struct {
	// Solving metrics
	TotalSolutions       int64         `json:"totalSolutions"`
	SuccessfulSolutions  int64         `json:"successfulSolutions"`
	FailedSolutions      int64         `json:"failedSolutions"`
	AvgSolveTime         time.Duration `json:"avgSolveTime"`
	MaxSolveTime         time.Duration `json:"maxSolveTime"`
	
	// Constraint metrics
	ConstraintsProcessed int64 `json:"constraintsProcessed"`
	ConstraintViolations int64 `json:"constraintViolations"`
	ConstraintsSatisfied int64 `json:"constraintsSatisfied"`
	
	// Algorithm metrics
	BacktrackingSteps    int64   `json:"backtrackingSteps"`
	OptimalSolutions     int64   `json:"optimalSolutions"`
	SuboptimalSolutions  int64   `json:"suboptimalSolutions"`
	AvgOptimalityScore   float64 `json:"avgOptimalityScore"`
	
	// Resource usage
	MemoryUsageBytes     int64     `json:"memoryUsageBytes"`
	MaxMemoryUsageBytes  int64     `json:"maxMemoryUsageBytes"`
	
	// Timing
	LastResetAt          time.Time `json:"lastResetAt"`
	CollectionStarted    time.Time `json:"collectionStarted"`
}