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

package shared

import (
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
	"k8s.io/apimachinery/pkg/types"
)

// IntentState represents the complete state of a network intent.
type IntentState struct {
	// Basic identification.
	NamespacedName types.NamespacedName `json:"namespacedName"`
	Version        string               `json:"version"`
	CreationTime   time.Time            `json:"creationTime"`
	LastModified   time.Time            `json:"lastModified"`

	// Processing state.
	CurrentPhase     interfaces.ProcessingPhase `json:"currentPhase"`
	PhaseStartTime   time.Time                  `json:"phaseStartTime"`
	PhaseTransitions []PhaseTransition          `json:"phaseTransitions"`

	// Phase-specific data.
	PhaseData   map[interfaces.ProcessingPhase]interface{} `json:"phaseData"`
	PhaseErrors map[interfaces.ProcessingPhase][]string    `json:"phaseErrors"`

	// Status conditions.
	Conditions []StateCondition `json:"conditions"`

	// Dependencies and relationships.
	Dependencies     []IntentDependency `json:"dependencies"`
	DependentIntents []string           `json:"dependentIntents"`

	// Error tracking.
	LastError     string    `json:"lastError,omitempty"`
	LastErrorTime time.Time `json:"lastErrorTime,omitempty"`
	RetryCount    int       `json:"retryCount"`

	// Resource tracking.
	AllocatedResources []ResourceAllocation `json:"allocatedResources"`
	ResourceLocks      []ResourceLock       `json:"resourceLocks"`

	// Processing metrics.
	ProcessingDuration time.Duration                               `json:"processingDuration"`
	PhaseMetrics       map[interfaces.ProcessingPhase]PhaseMetrics `json:"phaseMetrics"`

	// Metadata and annotations.
	Metadata map[string]interface{} `json:"metadata"`
	Tags     []string               `json:"tags"`
}

// PhaseTransition represents a transition between processing phases.
type PhaseTransition struct {
	FromPhase     interfaces.ProcessingPhase `json:"fromPhase"`
	ToPhase       interfaces.ProcessingPhase `json:"toPhase"`
	Timestamp     time.Time                  `json:"timestamp"`
	Duration      time.Duration              `json:"duration"`
	TriggerReason string                     `json:"triggerReason,omitempty"`
	Metadata      map[string]interface{}     `json:"metadata,omitempty"`
	Success       bool                       `json:"success"`
	ErrorMessage  string                     `json:"errorMessage,omitempty"`
}

// StateCondition represents a condition in the intent state.
type StateCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
	Severity           string    `json:"severity,omitempty"`
}

// IntentDependency represents a dependency on another intent.
type IntentDependency struct {
	Intent    string                     `json:"intent"`
	Phase     interfaces.ProcessingPhase `json:"phase"`
	Type      string                     `json:"type"` // "blocking", "soft", "notification"
	Timestamp time.Time                  `json:"timestamp"`
	Condition string                     `json:"condition,omitempty"`
	Timeout   time.Duration              `json:"timeout,omitempty"`
}

// ResourceAllocation represents allocated resources for an intent.
type ResourceAllocation struct {
	ResourceType string            `json:"resourceType"`
	ResourceID   string            `json:"resourceId"`
	Quantity     string            `json:"quantity"`
	Status       string            `json:"status"` // "allocated", "released", "pending"
	AllocatedAt  time.Time         `json:"allocatedAt"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// ResourceLock represents a resource lock held by an intent.
type ResourceLock struct {
	LockID       string            `json:"lockId"`
	ResourceType string            `json:"resourceType"`
	ResourceID   string            `json:"resourceId"`
	LockType     string            `json:"lockType"` // "exclusive", "shared"
	AcquiredAt   time.Time         `json:"acquiredAt"`
	ExpiresAt    time.Time         `json:"expiresAt"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// PhaseMetrics contains metrics for a specific processing phase.
type PhaseMetrics struct {
	StartTime       time.Time     `json:"startTime"`
	EndTime         time.Time     `json:"endTime"`
	Duration        time.Duration `json:"duration"`
	AttemptsCount   int           `json:"attemptsCount"`
	SuccessCount    int           `json:"successCount"`
	ErrorCount      int           `json:"errorCount"`
	ResourcesUsed   []string      `json:"resourcesUsed"`
	ProcessedItems  int           `json:"processedItems"`
	ThroughputRate  float64       `json:"throughputRate"`
	MemoryUsageMB   float64       `json:"memoryUsageMB"`
	CPUUsagePercent float64       `json:"cpuUsagePercent"`
}

// StateChangeEvent represents a state change event.
type StateChangeEvent struct {
	Type           string                     `json:"type"`
	IntentName     types.NamespacedName       `json:"intentName"`
	OldPhase       interfaces.ProcessingPhase `json:"oldPhase,omitempty"`
	NewPhase       interfaces.ProcessingPhase `json:"newPhase"`
	Version        string                     `json:"version"`
	Timestamp      time.Time                  `json:"timestamp"`
	ChangeReason   string                     `json:"changeReason,omitempty"`
	Metadata       map[string]interface{}     `json:"metadata,omitempty"`
	AffectedFields []string                   `json:"affectedFields,omitempty"`
}

// StateStatistics provides statistics about state management.
type StateStatistics struct {
	TotalStates           int            `json:"totalStates"`
	StatesByPhase         map[string]int `json:"statesByPhase"`
	CacheHitRate          float64        `json:"cacheHitRate"`
	ActiveLocks           int            `json:"activeLocks"`
	AverageUpdateTime     time.Duration  `json:"averageUpdateTime"`
	StateValidationErrors int64          `json:"stateValidationErrors"`
	LastSyncTime          time.Time      `json:"lastSyncTime"`
	ConflictResolutions   int64          `json:"conflictResolutions"`
	DependencyViolations  int64          `json:"dependencyViolations"`
}

// StateSyncRequest represents a request to synchronize state.
type StateSyncRequest struct {
	IntentName    types.NamespacedName `json:"intentName"`
	RequestedBy   string               `json:"requestedBy"`
	SyncType      string               `json:"syncType"` // "full", "incremental", "validate"
	ForceSync     bool                 `json:"forceSync"`
	Timestamp     time.Time            `json:"timestamp"`
	TargetVersion string               `json:"targetVersion,omitempty"`
}

// StateSyncResponse represents a response to state synchronization.
type StateSyncResponse struct {
	RequestID         string            `json:"requestId"`
	Success           bool              `json:"success"`
	ConflictsFound    int               `json:"conflictsFound"`
	ConflictsResolved int               `json:"conflictsResolved"`
	UpdatesApplied    int               `json:"updatesApplied"`
	ErrorMessage      string            `json:"errorMessage,omitempty"`
	ProcessingTime    time.Duration     `json:"processingTime"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

// StateConflict represents a conflict between states.
type StateConflict struct {
	IntentName      types.NamespacedName `json:"intentName"`
	ConflictType    string               `json:"conflictType"`
	Field           string               `json:"field"`
	CachedValue     interface{}          `json:"cachedValue"`
	KubernetesValue interface{}          `json:"kubernetesValue"`
	LastModified    time.Time            `json:"lastModified"`
	Severity        string               `json:"severity"` // "low", "medium", "high", "critical"
	Resolution      string               `json:"resolution,omitempty"`
}

// StateValidationResult represents the result of state validation.
type StateValidationResult struct {
	Valid           bool                     `json:"valid"`
	Errors          []StateValidationError   `json:"errors,omitempty"`
	Warnings        []StateValidationWarning `json:"warnings,omitempty"`
	ValidationTime  time.Duration            `json:"validationTime"`
	ValidatedFields int                      `json:"validatedFields"`
	Version         string                   `json:"version"`
}

// StateValidationError represents a state validation error.
type StateValidationError struct {
	Field      string                 `json:"field"`
	Value      interface{}            `json:"value"`
	Constraint string                 `json:"constraint"`
	Message    string                 `json:"message"`
	Severity   string                 `json:"severity"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// StateValidationWarning represents a state validation warning.
type StateValidationWarning struct {
	Field      string                 `json:"field"`
	Value      interface{}            `json:"value"`
	Message    string                 `json:"message"`
	Suggestion string                 `json:"suggestion,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// StateMutex represents a distributed mutex for state synchronization.
type StateMutex struct {
	Name       string            `json:"name"`
	Owner      string            `json:"owner"`
	AcquiredAt time.Time         `json:"acquiredAt"`
	ExpiresAt  time.Time         `json:"expiresAt"`
	Renewable  bool              `json:"renewable"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// StateSnapshot represents a point-in-time snapshot of intent state.
type StateSnapshot struct {
	IntentName     types.NamespacedName `json:"intentName"`
	Version        string               `json:"version"`
	SnapshotTime   time.Time            `json:"snapshotTime"`
	State          *IntentState         `json:"state"`
	Checksum       string               `json:"checksum"`
	CompressedSize int64                `json:"compressedSize"`
	Metadata       map[string]string    `json:"metadata,omitempty"`
}

// StateRecoveryInfo contains information for state recovery.
type StateRecoveryInfo struct {
	IntentName           types.NamespacedName `json:"intentName"`
	LastKnownGoodVersion string               `json:"lastKnownGoodVersion"`
	CorruptedVersion     string               `json:"corruptedVersion"`
	RecoveryStrategy     string               `json:"recoveryStrategy"`
	RecoveryTime         time.Time            `json:"recoveryTime"`
	DataLoss             bool                 `json:"dataLoss"`
	RecoveredFields      []string             `json:"recoveredFields"`
	Notes                string               `json:"notes,omitempty"`
}

// State query and filtering types.

// StateQuery represents a query for intent states.
type StateQuery struct {
	Namespace       string                       `json:"namespace,omitempty"`
	Name            string                       `json:"name,omitempty"`
	Phase           interfaces.ProcessingPhase   `json:"phase,omitempty"`
	Phases          []interfaces.ProcessingPhase `json:"phases,omitempty"`
	Labels          map[string]string            `json:"labels,omitempty"`
	Tags            []string                     `json:"tags,omitempty"`
	CreatedAfter    *time.Time                   `json:"createdAfter,omitempty"`
	CreatedBefore   *time.Time                   `json:"createdBefore,omitempty"`
	ModifiedAfter   *time.Time                   `json:"modifiedAfter,omitempty"`
	ModifiedBefore  *time.Time                   `json:"modifiedBefore,omitempty"`
	HasErrors       *bool                        `json:"hasErrors,omitempty"`
	HasDependencies *bool                        `json:"hasDependencies,omitempty"`
	OrderBy         string                       `json:"orderBy,omitempty"` // "name", "created", "modified", "phase"
	Order           string                       `json:"order,omitempty"`   // "asc", "desc"
	Limit           int                          `json:"limit,omitempty"`
	Offset          int                          `json:"offset,omitempty"`
}

// StateQueryResult represents the result of a state query.
type StateQueryResult struct {
	States        []*IntentState `json:"states"`
	TotalCount    int            `json:"totalCount"`
	FilteredCount int            `json:"filteredCount"`
	ExecutionTime time.Duration  `json:"executionTime"`
	HasMore       bool           `json:"hasMore"`
	NextOffset    int            `json:"nextOffset,omitempty"`
}

// Constants for state management.

const (
	// State condition types.
	ConditionTypeReady = "Ready"
	// ConditionTypeProgressing holds conditiontypeprogressing value.
	ConditionTypeProgressing = "Progressing"
	// ConditionTypeDegraded holds conditiontypedegraded value.
	ConditionTypeDegraded = "Degraded"
	// ConditionTypeAvailable holds conditiontypeavailable value.
	ConditionTypeAvailable = "Available"
	// ConditionTypeError holds conditiontypeerror value.
	ConditionTypeError = "Error"
	// ConditionTypeDependenciesMet holds conditiontypedependenciesmet value.
	ConditionTypeDependenciesMet = "DependenciesMet"
	// ConditionTypeResourcesAllocated holds conditiontyperesourcesallocated value.
	ConditionTypeResourcesAllocated = "ResourcesAllocated"
	// ConditionTypeValidated holds conditiontypevalidated value.
	ConditionTypeValidated = "Validated"

	// State condition statuses.
	ConditionStatusTrue = "True"
	// ConditionStatusFalse holds conditionstatusfalse value.
	ConditionStatusFalse = "False"
	// ConditionStatusUnknown holds conditionstatusunknown value.
	ConditionStatusUnknown = "Unknown"

	// Dependency types.
	DependencyTypeBlocking = "blocking"
	// DependencyTypeSoft holds dependencytypesoft value.
	DependencyTypeSoft = "soft"
	// DependencyTypeNotification holds dependencytypenotification value.
	DependencyTypeNotification = "notification"

	// Resource allocation statuses.
	ResourceStatusAllocated = "allocated"
	// ResourceStatusReleased holds resourcestatusreleased value.
	ResourceStatusReleased = "released"
	// ResourceStatusPending holds resourcestatuspending value.
	ResourceStatusPending = "pending"
	// ResourceStatusFailed holds resourcestatusfailed value.
	ResourceStatusFailed = "failed"

	// Lock types.
	LockTypeExclusive = "exclusive"
	// LockTypeShared holds locktypeshared value.
	LockTypeShared = "shared"

	// Conflict resolution modes.
	ConflictResolutionLatest = "latest"
	// ConflictResolutionMerge holds conflictresolutionmerge value.
	ConflictResolutionMerge = "merge"
	// ConflictResolutionManual holds conflictresolutionmanual value.
	ConflictResolutionManual = "manual"

	// Sync types.
	SyncTypeFull = "full"
	// SyncTypeIncremental holds synctypeincremental value.
	SyncTypeIncremental = "incremental"
	// SyncTypeValidate holds synctypevalidate value.
	SyncTypeValidate = "validate"

	// Severity levels.
	SeverityLow = "low"
	// SeverityMedium holds severitymedium value.
	SeverityMedium = "medium"
	// SeverityHigh holds severityhigh value.
	SeverityHigh = "high"
	// SeverityCritical holds severitycritical value.
	SeverityCritical = "critical"
)
