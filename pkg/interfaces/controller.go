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

package interfaces

import (
	"context"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/contracts"
)

// Type aliases for backward compatibility
type ComponentType = contracts.ComponentType
type ProcessingPhase = contracts.ProcessingPhase

// Constants for backward compatibility
const (
	ComponentTypeLLMProcessor       = contracts.ComponentTypeLLMProcessor
	ComponentTypeResourcePlanner    = contracts.ComponentTypeResourcePlanner
	ComponentTypeManifestGenerator  = contracts.ComponentTypeManifestGenerator
	ComponentTypeGitOpsController   = contracts.ComponentTypeGitOpsController
	ComponentTypeDeploymentVerifier = contracts.ComponentTypeDeploymentVerifier

	PhaseIntentReceived         = contracts.PhaseIntentReceived
	PhaseLLMProcessing          = contracts.PhaseLLMProcessing
	PhaseResourcePlanning       = contracts.PhaseResourcePlanning
	PhaseManifestGeneration     = contracts.PhaseManifestGeneration
	PhaseGitOpsCommit           = contracts.PhaseGitOpsCommit
	PhaseDeploymentVerification = contracts.PhaseDeploymentVerification
	PhaseCompleted              = contracts.PhaseCompleted
	PhaseFailed                 = contracts.PhaseFailed
)

// Type aliases for backward compatibility
type ProcessingResult = contracts.ProcessingResult
type ProcessingEvent = contracts.ProcessingEvent
type ResourceReference = contracts.ResourceReference

// ControllerInterface defines the contract for specialized controllers
type ControllerInterface interface {
	// Process handles the specific phase for this controller
	Process(ctx context.Context, intent *nephoranv1.NetworkIntent) (*ProcessingResult, error)
	
	// GetPhase returns the processing phase this controller handles
	GetPhase() ProcessingPhase
	
	// CanProcess checks if this controller can handle the given intent in its current state
	CanProcess(intent *nephoranv1.NetworkIntent) bool
	
	// GetDependencies returns the phases this controller depends on
	GetDependencies() []ProcessingPhase
	
	// GetMetrics returns controller-specific metrics
	GetMetrics() map[string]float64
	
	// HealthCheck performs a health check for this controller
	HealthCheck(ctx context.Context) error
	
	// GetComponentType returns the component type (for shared package compatibility)
	GetComponentType() ComponentType
	
	// IsHealthy returns current health status (for shared package compatibility)  
	IsHealthy() bool
}

// PhaseController is a minimal interface for phase processing without circular dependencies
type PhaseController interface {
	ProcessPhase(ctx context.Context, intent *nephoranv1.NetworkIntent, phase ProcessingPhase) (*ProcessingResult, error)
	GetSupportedPhases() []ProcessingPhase
}

// Interface aliases for backward compatibility
type StateManager = contracts.StateManager
type EventBus = contracts.EventBus