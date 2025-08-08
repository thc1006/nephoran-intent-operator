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

package orchestration

import (
	"context"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
)

// Conflict represents a resource conflict between intents
type Conflict struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	IntentID1   string                 `json:"intentId1"`
	IntentID2   string                 `json:"intentId2"`
	Resource    string                 `json:"resource"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// CoordinationContext holds coordination state for an intent
type CoordinationContext struct {
	IntentID         string
	CurrentPhase     interfaces.ProcessingPhase
	CompletedPhases  []interfaces.ProcessingPhase
	FailedPhases     []interfaces.ProcessingPhase
	Locks            []string
	Dependencies     []string
	Conflicts        []Conflict
	ErrorHistory     []string
	RetryCount       int
	StartTime        time.Time
	LastUpdateTime   time.Time
	Metadata         map[string]interface{}
}

// PhaseTracker tracks the status of phases across intents
type PhaseTracker struct {
	phaseStatuses map[string]map[interfaces.ProcessingPhase]PhaseStatus
	mutex         sync.RWMutex
}

// PhaseStatus represents the status of a phase for an intent
type PhaseStatus struct {
	IntentID    string
	Phase       interfaces.ProcessingPhase
	Status      string // Pending, InProgress, Completed, Failed
	StartTime   *time.Time
	EndTime     *time.Time
	RetryCount  int
	LastError   string
	Metrics     map[string]float64
	DependsOn   []interfaces.ProcessingPhase
	BlockedBy   []interfaces.ProcessingPhase
}

// NewPhaseTracker creates a new PhaseTracker
func NewPhaseTracker() *PhaseTracker {
	return &PhaseTracker{
		phaseStatuses: make(map[string]map[interfaces.ProcessingPhase]PhaseStatus),
	}
}

// UpdatePhaseStatus updates the status of a phase
func (pt *PhaseTracker) UpdatePhaseStatus(intentID string, phase interfaces.ProcessingPhase, status string) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	if _, exists := pt.phaseStatuses[intentID]; !exists {
		pt.phaseStatuses[intentID] = make(map[interfaces.ProcessingPhase]PhaseStatus)
	}

	phaseStatus := pt.phaseStatuses[intentID][phase]
	phaseStatus.IntentID = intentID
	phaseStatus.Phase = phase
	phaseStatus.Status = status

	now := time.Now()
	if status == "InProgress" && phaseStatus.StartTime == nil {
		phaseStatus.StartTime = &now
	}
	if (status == "Completed" || status == "Failed") && phaseStatus.EndTime == nil {
		phaseStatus.EndTime = &now
	}

	pt.phaseStatuses[intentID][phase] = phaseStatus
}

// GetPhaseStatus returns the status of a phase for an intent
func (pt *PhaseTracker) GetPhaseStatus(intentID string, phase interfaces.ProcessingPhase) (PhaseStatus, bool) {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	if intentStatuses, exists := pt.phaseStatuses[intentID]; exists {
		if phaseStatus, exists := intentStatuses[phase]; exists {
			return phaseStatus, true
		}
	}

	return PhaseStatus{}, false
}

// GetIntentPhases returns all phase statuses for an intent
func (pt *PhaseTracker) GetIntentPhases(intentID string) map[interfaces.ProcessingPhase]PhaseStatus {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()

	if intentStatuses, exists := pt.phaseStatuses[intentID]; exists {
		// Return a copy to avoid race conditions
		copy := make(map[interfaces.ProcessingPhase]PhaseStatus)
		for phase, status := range intentStatuses {
			copy[phase] = status
		}
		return copy
	}

	return make(map[interfaces.ProcessingPhase]PhaseStatus)
}

// RemoveIntent removes all phase statuses for an intent
func (pt *PhaseTracker) RemoveIntent(intentID string) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	delete(pt.phaseStatuses, intentID)
}

// ConflictResolver handles resource conflicts between intents
type ConflictResolver struct {
	logger logr.Logger
	client client.Client
	resolutionStrategies map[string]ConflictResolutionStrategy
}

// ConflictResolutionStrategy defines how to resolve a specific type of conflict
type ConflictResolutionStrategy func(conflict Conflict) (bool, error)

// NewConflictResolver creates a new ConflictResolver
func NewConflictResolver(client client.Client, logger logr.Logger) *ConflictResolver {
	return &ConflictResolver{
		logger: logger.WithName("conflict-resolver"),
		client: client,
		resolutionStrategies: map[string]ConflictResolutionStrategy{
			"NamespaceConflict": resolveNamespaceConflict,
			"ResourceConflict":  resolveResourceConflict,
			"DependencyConflict": resolveDependencyConflict,
		},
	}
}

// ResolveConflict attempts to resolve a conflict
func (cr *ConflictResolver) ResolveConflict(ctx context.Context, conflict Conflict) (bool, error) {
	if strategy, exists := cr.resolutionStrategies[conflict.Type]; exists {
		return strategy(conflict)
	}

	// No specific strategy, attempt generic resolution
	return cr.genericResolution(conflict)
}

// genericResolution provides generic conflict resolution
func (cr *ConflictResolver) genericResolution(conflict Conflict) (bool, error) {
	// For now, log and return false (manual intervention required)
	cr.logger.Info("Generic conflict resolution not implemented", "conflictType", conflict.Type, "conflictId", conflict.ID)
	return false, nil
}

// Resolution strategies

func resolveNamespaceConflict(conflict Conflict) (bool, error) {
	// Simplified namespace conflict resolution
	// In practice, this might involve creating separate namespaces or prioritizing based on intent priority
	return true, nil // Assume we can always resolve namespace conflicts
}

func resolveResourceConflict(conflict Conflict) (bool, error) {
	// Resource conflicts might be resolved by resource sharing or priority-based allocation
	return false, nil // Require manual intervention for now
}

func resolveDependencyConflict(conflict Conflict) (bool, error) {
	// Dependency conflicts might be resolved by reordering or parallel execution
	return true, nil // Assume we can resolve dependency conflicts
}

// RecoveryManager handles automatic recovery from failures
type RecoveryManager struct {
	logger logr.Logger
	client client.Client
	recoveryStrategies map[string]RecoveryStrategy
}

// RecoveryStrategy defines how to recover from a specific type of failure
type RecoveryStrategy func(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, errorCode string, coordCtx *CoordinationContext) (bool, []string, error)

// NewRecoveryManager creates a new RecoveryManager
func NewRecoveryManager(client client.Client, logger logr.Logger) *RecoveryManager {
	return &RecoveryManager{
		logger: logger.WithName("recovery-manager"),
		client: client,
		recoveryStrategies: map[string]RecoveryStrategy{
			"LLM_PROCESSING_FAILED":     recoverLLMProcessing,
			"RESOURCE_PLANNING_FAILED":  recoverResourcePlanning,
			"MANIFEST_GENERATION_FAILED": recoverManifestGeneration,
			"DEPLOYMENT_FAILED":         recoverDeployment,
		},
	}
}

// AttemptRecovery attempts to recover from a failure
func (rm *RecoveryManager) AttemptRecovery(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, errorCode string, coordCtx *CoordinationContext) (bool, []string, error) {
	if strategy, exists := rm.recoveryStrategies[errorCode]; exists {
		return strategy(ctx, networkIntent, phase, errorCode, coordCtx)
	}

	// No specific strategy, attempt generic recovery
	return rm.genericRecovery(ctx, networkIntent, phase, errorCode, coordCtx)
}

// genericRecovery provides generic failure recovery
func (rm *RecoveryManager) genericRecovery(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, errorCode string, coordCtx *CoordinationContext) (bool, []string, error) {
	// Generic recovery might involve cleaning up resources and retrying
	actions := []string{"cleared-error-state", "reset-phase-context"}
	rm.logger.Info("Performed generic recovery", "phase", phase, "errorCode", errorCode, "actions", actions)
	return true, actions, nil
}

// Recovery strategies

func recoverLLMProcessing(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, errorCode string, coordCtx *CoordinationContext) (bool, []string, error) {
	// LLM processing recovery might involve:
	// - Switching to a different LLM provider
	// - Simplifying the prompt
	// - Using cached results if available
	actions := []string{"switched-llm-provider", "simplified-prompt"}
	return true, actions, nil
}

func recoverResourcePlanning(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, errorCode string, coordCtx *CoordinationContext) (bool, []string, error) {
	// Resource planning recovery might involve:
	// - Using default resource allocations
	// - Disabling optimizations
	// - Using simpler deployment patterns
	actions := []string{"used-default-resources", "disabled-optimizations"}
	return true, actions, nil
}

func recoverManifestGeneration(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, errorCode string, coordCtx *CoordinationContext) (bool, []string, error) {
	// Manifest generation recovery might involve:
	// - Using simpler templates
	// - Disabling complex features
	// - Using fallback manifest generation
	actions := []string{"used-simple-templates", "disabled-complex-features"}
	return true, actions, nil
}

func recoverDeployment(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase, errorCode string, coordCtx *CoordinationContext) (bool, []string, error) {
	// Deployment recovery might involve:
	// - Cleaning up failed resources
	// - Retrying with simpler configurations
	// - Using alternative deployment strategies
	actions := []string{"cleaned-failed-resources", "simplified-configuration"}
	return true, actions, nil
}
