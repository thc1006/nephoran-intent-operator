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

	"fmt"



	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"

	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"

)



// StateMachine manages the state transitions for intent processing.

type StateMachine struct {

	// State transition rules.

	transitions map[interfaces.ProcessingPhase][]interfaces.ProcessingPhase



	// Parallel processing rules.

	parallelPhases map[interfaces.ProcessingPhase][]interfaces.ProcessingPhase



	// Phase dependencies.

	dependencies map[interfaces.ProcessingPhase][]interfaces.ProcessingPhase



	// Terminal states.

	terminalStates map[interfaces.ProcessingPhase]bool



	// Configuration.

	config *OrchestratorConfig

}



// NewStateMachine creates a new state machine.

func NewStateMachine(config *OrchestratorConfig) *StateMachine {

	sm := &StateMachine{

		transitions:    make(map[interfaces.ProcessingPhase][]interfaces.ProcessingPhase),

		parallelPhases: make(map[interfaces.ProcessingPhase][]interfaces.ProcessingPhase),

		dependencies:   make(map[interfaces.ProcessingPhase][]interfaces.ProcessingPhase),

		terminalStates: make(map[interfaces.ProcessingPhase]bool),

		config:         config,

	}



	sm.initializeTransitions()

	sm.initializeParallelProcessing()

	sm.initializeDependencies()

	sm.initializeTerminalStates()



	return sm

}



// initializeTransitions sets up the state transition rules.

func (sm *StateMachine) initializeTransitions() {

	// Standard sequential processing flow.

	sm.transitions[interfaces.PhaseIntentReceived] = []interfaces.ProcessingPhase{

		interfaces.PhaseLLMProcessing,

	}



	sm.transitions[interfaces.PhaseLLMProcessing] = []interfaces.ProcessingPhase{

		interfaces.PhaseResourcePlanning,

		interfaces.PhaseFailed, // In case of failure

	}



	sm.transitions[interfaces.PhaseResourcePlanning] = []interfaces.ProcessingPhase{

		interfaces.PhaseManifestGeneration,

		interfaces.PhaseFailed,

	}



	sm.transitions[interfaces.PhaseManifestGeneration] = []interfaces.ProcessingPhase{

		interfaces.PhaseGitOpsCommit,

		interfaces.PhaseFailed,

	}



	sm.transitions[interfaces.PhaseGitOpsCommit] = []interfaces.ProcessingPhase{

		interfaces.PhaseDeploymentVerification,

		interfaces.PhaseFailed,

	}



	sm.transitions[interfaces.PhaseDeploymentVerification] = []interfaces.ProcessingPhase{

		interfaces.PhaseCompleted,

		interfaces.PhaseFailed,

	}



	// Terminal states have no transitions.

	sm.transitions[interfaces.PhaseCompleted] = []interfaces.ProcessingPhase{}

	sm.transitions[interfaces.PhaseFailed] = []interfaces.ProcessingPhase{}

}



// initializeParallelProcessing sets up parallel processing rules.

func (sm *StateMachine) initializeParallelProcessing() {

	// After LLM processing, resource planning can start while preparing for manifest generation.

	sm.parallelPhases[interfaces.PhaseResourcePlanning] = []interfaces.ProcessingPhase{

		interfaces.PhaseManifestGeneration, // Can start template preparation

	}



	// During manifest generation, we can prepare GitOps workflows.

	sm.parallelPhases[interfaces.PhaseManifestGeneration] = []interfaces.ProcessingPhase{

		interfaces.PhaseGitOpsCommit, // Can prepare git operations

	}



	// Note: This is a simplified example. In practice, you might have more complex.

	// parallel processing rules based on resource types, network function types, etc.

}



// initializeDependencies sets up phase dependencies.

func (sm *StateMachine) initializeDependencies() {

	// LLM Processing has no dependencies (starts immediately).

	sm.dependencies[interfaces.PhaseLLMProcessing] = []interfaces.ProcessingPhase{}



	// Resource Planning depends on LLM Processing.

	sm.dependencies[interfaces.PhaseResourcePlanning] = []interfaces.ProcessingPhase{

		interfaces.PhaseLLMProcessing,

	}



	// Manifest Generation depends on Resource Planning.

	sm.dependencies[interfaces.PhaseManifestGeneration] = []interfaces.ProcessingPhase{

		interfaces.PhaseResourcePlanning,

	}



	// GitOps Commit depends on Manifest Generation.

	sm.dependencies[interfaces.PhaseGitOpsCommit] = []interfaces.ProcessingPhase{

		interfaces.PhaseManifestGeneration,

	}



	// Deployment Verification depends on GitOps Commit.

	sm.dependencies[interfaces.PhaseDeploymentVerification] = []interfaces.ProcessingPhase{

		interfaces.PhaseGitOpsCommit,

	}

}



// initializeTerminalStates marks terminal states.

func (sm *StateMachine) initializeTerminalStates() {

	sm.terminalStates[interfaces.PhaseCompleted] = true

	sm.terminalStates[interfaces.PhaseFailed] = true

}



// GetCurrentPhase determines the current phase based on intent status.

func (sm *StateMachine) GetCurrentPhase(intent *nephoranv1.NetworkIntent) interfaces.ProcessingPhase {

	// If no phase is set, start with intent received.

	if intent.Status.Phase == "" {

		return interfaces.PhaseIntentReceived

	}



	// Convert string to ProcessingPhase.

	currentPhase := interfaces.ProcessingPhase(intent.Status.Phase)



	// Validate the phase.

	if sm.isValidPhase(currentPhase) {

		return currentPhase

	}



	// If invalid phase, start from beginning.

	return interfaces.PhaseIntentReceived

}



// GetNextPhase returns the next phase in the processing pipeline.

func (sm *StateMachine) GetNextPhase(currentPhase interfaces.ProcessingPhase) interfaces.ProcessingPhase {

	transitions, exists := sm.transitions[currentPhase]

	if !exists || len(transitions) == 0 {

		return interfaces.PhaseFailed // No valid transitions

	}



	// Return the first (primary) transition.

	return transitions[0]

}



// GetPossibleTransitions returns all possible transitions from a phase.

func (sm *StateMachine) GetPossibleTransitions(currentPhase interfaces.ProcessingPhase) []interfaces.ProcessingPhase {

	transitions, exists := sm.transitions[currentPhase]

	if !exists {

		return []interfaces.ProcessingPhase{}

	}



	return transitions

}



// CanTransitionTo checks if a transition is valid.

func (sm *StateMachine) CanTransitionTo(from, to interfaces.ProcessingPhase) bool {

	transitions := sm.GetPossibleTransitions(from)

	for _, phase := range transitions {

		if phase == to {

			return true

		}

	}

	return false

}



// GetParallelPhases returns phases that can run in parallel with the given phase.

func (sm *StateMachine) GetParallelPhases(phase interfaces.ProcessingPhase) []interfaces.ProcessingPhase {

	if !sm.config.EnableParallelProcessing {

		return []interfaces.ProcessingPhase{}

	}



	parallelPhases, exists := sm.parallelPhases[phase]

	if !exists {

		return []interfaces.ProcessingPhase{}

	}



	return parallelPhases

}



// GetPhaseDependencies returns the dependencies for a phase.

func (sm *StateMachine) GetPhaseDependencies(phase interfaces.ProcessingPhase) []interfaces.ProcessingPhase {

	dependencies, exists := sm.dependencies[phase]

	if !exists {

		return []interfaces.ProcessingPhase{}

	}



	return dependencies

}



// IsTerminalPhase checks if a phase is terminal.

func (sm *StateMachine) IsTerminalPhase(phase interfaces.ProcessingPhase) bool {

	return sm.terminalStates[phase]

}



// ValidateTransition validates a phase transition.

func (sm *StateMachine) ValidateTransition(intent *nephoranv1.NetworkIntent, fromPhase, toPhase interfaces.ProcessingPhase) error {

	// Check if the transition is valid.

	if !sm.CanTransitionTo(fromPhase, toPhase) {

		return fmt.Errorf("invalid transition from %s to %s", fromPhase, toPhase)

	}



	// Check dependencies for the target phase.

	dependencies := sm.GetPhaseDependencies(toPhase)

	for _, depPhase := range dependencies {

		if !sm.isPhaseCompleted(intent, depPhase) {

			return fmt.Errorf("dependency %s not completed for phase %s", depPhase, toPhase)

		}

	}



	return nil

}



// GetProcessingWorkflow returns the complete workflow for an intent type.

func (sm *StateMachine) GetProcessingWorkflow(intentType nephoranv1.IntentType) *ProcessingWorkflow {

	workflow := &ProcessingWorkflow{

		IntentType: intentType,

		Phases:     make([]WorkflowPhase, 0),

	}



	// Build workflow based on intent type.

	switch intentType {

	case nephoranv1.IntentTypeDeployment:

		workflow.Phases = sm.buildDeploymentWorkflow()

	case nephoranv1.IntentTypeScaling:

		workflow.Phases = sm.buildScalingWorkflow()

	case nephoranv1.IntentTypeOptimization:

		workflow.Phases = sm.buildOptimizationWorkflow()

	case nephoranv1.IntentTypeMaintenance:

		workflow.Phases = sm.buildMaintenanceWorkflow()

	default:

		workflow.Phases = sm.buildDefaultWorkflow()

	}



	return workflow

}



// buildDeploymentWorkflow creates a deployment-specific workflow.

func (sm *StateMachine) buildDeploymentWorkflow() []WorkflowPhase {

	return []WorkflowPhase{

		{

			Phase:             interfaces.PhaseLLMProcessing,

			Required:          true,

			Parallel:          false,

			Dependencies:      []interfaces.ProcessingPhase{},

			EstimatedDuration: "2m",

		},

		{

			Phase:             interfaces.PhaseResourcePlanning,

			Required:          true,

			Parallel:          false,

			Dependencies:      []interfaces.ProcessingPhase{interfaces.PhaseLLMProcessing},

			EstimatedDuration: "1m",

		},

		{

			Phase:             interfaces.PhaseManifestGeneration,

			Required:          true,

			Parallel:          true, // Can start while resource planning completes

			Dependencies:      []interfaces.ProcessingPhase{interfaces.PhaseResourcePlanning},

			EstimatedDuration: "30s",

		},

		{

			Phase:             interfaces.PhaseGitOpsCommit,

			Required:          true,

			Parallel:          false,

			Dependencies:      []interfaces.ProcessingPhase{interfaces.PhaseManifestGeneration},

			EstimatedDuration: "1m",

		},

		{

			Phase:             interfaces.PhaseDeploymentVerification,

			Required:          true,

			Parallel:          false,

			Dependencies:      []interfaces.ProcessingPhase{interfaces.PhaseGitOpsCommit},

			EstimatedDuration: "3m",

		},

	}

}



// buildScalingWorkflow creates a scaling-specific workflow.

func (sm *StateMachine) buildScalingWorkflow() []WorkflowPhase {

	return []WorkflowPhase{

		{

			Phase:             interfaces.PhaseLLMProcessing,

			Required:          true,

			Parallel:          false,

			Dependencies:      []interfaces.ProcessingPhase{},

			EstimatedDuration: "1m", // Faster for scaling operations

		},

		{

			Phase:             interfaces.PhaseResourcePlanning,

			Required:          true,

			Parallel:          false,

			Dependencies:      []interfaces.ProcessingPhase{interfaces.PhaseLLMProcessing},

			EstimatedDuration: "30s", // Scaling is resource-focused

		},

		{

			Phase:             interfaces.PhaseManifestGeneration,

			Required:          false, // May not need new manifests for simple scaling

			Parallel:          false,

			Dependencies:      []interfaces.ProcessingPhase{interfaces.PhaseResourcePlanning},

			EstimatedDuration: "15s",

		},

		{

			Phase:             interfaces.PhaseGitOpsCommit,

			Required:          false, // May update existing resources directly

			Parallel:          false,

			Dependencies:      []interfaces.ProcessingPhase{interfaces.PhaseManifestGeneration},

			EstimatedDuration: "30s",

		},

		{

			Phase:             interfaces.PhaseDeploymentVerification,

			Required:          true,

			Parallel:          false,

			Dependencies:      []interfaces.ProcessingPhase{interfaces.PhaseGitOpsCommit},

			EstimatedDuration: "2m", // Verify scaling worked

		},

	}

}



// buildOptimizationWorkflow creates an optimization-specific workflow.

func (sm *StateMachine) buildOptimizationWorkflow() []WorkflowPhase {

	// Similar to deployment but with additional analysis phases.

	return sm.buildDeploymentWorkflow()

}



// buildMaintenanceWorkflow creates a maintenance-specific workflow.

func (sm *StateMachine) buildMaintenanceWorkflow() []WorkflowPhase {

	// May skip some phases for maintenance operations.

	return []WorkflowPhase{

		{

			Phase:             interfaces.PhaseLLMProcessing,

			Required:          true,

			Parallel:          false,

			Dependencies:      []interfaces.ProcessingPhase{},

			EstimatedDuration: "1m",

		},

		{

			Phase:             interfaces.PhaseManifestGeneration,

			Required:          true,

			Parallel:          false,

			Dependencies:      []interfaces.ProcessingPhase{interfaces.PhaseLLMProcessing},

			EstimatedDuration: "30s",

		},

		{

			Phase:             interfaces.PhaseGitOpsCommit,

			Required:          true,

			Parallel:          false,

			Dependencies:      []interfaces.ProcessingPhase{interfaces.PhaseManifestGeneration},

			EstimatedDuration: "1m",

		},

		{

			Phase:             interfaces.PhaseDeploymentVerification,

			Required:          true,

			Parallel:          false,

			Dependencies:      []interfaces.ProcessingPhase{interfaces.PhaseGitOpsCommit},

			EstimatedDuration: "2m",

		},

	}

}



// buildDefaultWorkflow creates the default workflow.

func (sm *StateMachine) buildDefaultWorkflow() []WorkflowPhase {

	return sm.buildDeploymentWorkflow()

}



// Helper methods.



func (sm *StateMachine) isValidPhase(phase interfaces.ProcessingPhase) bool {

	validPhases := []interfaces.ProcessingPhase{

		interfaces.PhaseIntentReceived,

		interfaces.PhaseLLMProcessing,

		interfaces.PhaseResourcePlanning,

		interfaces.PhaseManifestGeneration,

		interfaces.PhaseGitOpsCommit,

		interfaces.PhaseDeploymentVerification,

		interfaces.PhaseCompleted,

		interfaces.PhaseFailed,

	}



	for _, validPhase := range validPhases {

		if phase == validPhase {

			return true

		}

	}



	return false

}



func (sm *StateMachine) isPhaseCompleted(intent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase) bool {

	// Check if the phase has a completed condition.

	for _, condition := range intent.Status.Conditions {

		if condition.Type == string(phase) && condition.Status == "True" {

			return true

		}

	}



	return false

}



// Data structures for workflow management.



// ProcessingWorkflow represents a complete processing workflow for an intent type.

type ProcessingWorkflow struct {

	IntentType         nephoranv1.IntentType `json:"intentType"`

	Phases             []WorkflowPhase       `json:"phases"`

	TotalEstimatedTime string                `json:"totalEstimatedTime"`

	ParallelismEnabled bool                  `json:"parallelismEnabled"`

}



// WorkflowPhase represents a phase in a processing workflow.

type WorkflowPhase struct {

	Phase             interfaces.ProcessingPhase   `json:"phase"`

	Required          bool                         `json:"required"`

	Parallel          bool                         `json:"parallel"`

	Dependencies      []interfaces.ProcessingPhase `json:"dependencies"`

	EstimatedDuration string                       `json:"estimatedDuration"`

	Conditions        []PhaseCondition             `json:"conditions,omitempty"`

}



// PhaseCondition represents a condition for phase execution.

type PhaseCondition struct {

	Type     string      `json:"type"`

	Field    string      `json:"field"`

	Operator string      `json:"operator"`

	Value    interface{} `json:"value"`

}



// GetWorkflowProgress calculates the progress of workflow execution.

func (workflow *ProcessingWorkflow) GetWorkflowProgress(intent *nephoranv1.NetworkIntent) *WorkflowProgress {

	progress := &WorkflowProgress{

		TotalPhases:     len(workflow.Phases),

		CompletedPhases: 0,

		CurrentPhase:    interfaces.ProcessingPhase(intent.Status.Phase),

		PhaseStatuses:   make(map[interfaces.ProcessingPhase]string),

	}



	for _, phase := range workflow.Phases {

		status := "pending"



		// Check if phase is completed.

		for _, condition := range intent.Status.Conditions {

			if condition.Type == string(phase.Phase) {

				if condition.Status == "True" {

					status = "completed"

					progress.CompletedPhases++

				} else if condition.Reason == "Processing" {

					status = "in_progress"

				} else if condition.Status == "False" {

					status = "failed"

				}

				break

			}

		}



		progress.PhaseStatuses[phase.Phase] = status

	}



	// Calculate percentage.

	if progress.TotalPhases > 0 {

		progress.ProgressPercentage = float64(progress.CompletedPhases) / float64(progress.TotalPhases) * 100

	}



	return progress

}



// WorkflowProgress represents the current progress of a workflow.

type WorkflowProgress struct {

	TotalPhases        int                                   `json:"totalPhases"`

	CompletedPhases    int                                   `json:"completedPhases"`

	ProgressPercentage float64                               `json:"progressPercentage"`

	CurrentPhase       interfaces.ProcessingPhase            `json:"currentPhase"`

	PhaseStatuses      map[interfaces.ProcessingPhase]string `json:"phaseStatuses"`

}

