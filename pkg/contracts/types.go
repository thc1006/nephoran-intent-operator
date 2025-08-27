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

package contracts

import (
	"time"
)

// ComponentType represents different types of components in the system
type ComponentType string

const (
	ComponentTypeLLMProcessor       ComponentType = "llm-processor"
	ComponentTypeResourcePlanner    ComponentType = "resource-planner"
	ComponentTypeManifestGenerator  ComponentType = "manifest-generator"
	ComponentTypeGitOpsController   ComponentType = "gitops-controller"
	ComponentTypeDeploymentVerifier ComponentType = "deployment-verifier"
)

// ProcessingPhase represents the current phase of intent processing
type ProcessingPhase string

const (
	PhaseIntentReceived         ProcessingPhase = "IntentReceived"
	PhaseLLMProcessing          ProcessingPhase = "LLMProcessing"
	PhaseResourcePlanning       ProcessingPhase = "ResourcePlanning"
	PhaseManifestGeneration     ProcessingPhase = "ManifestGeneration"
	PhaseGitOpsCommit           ProcessingPhase = "GitOpsCommit"
	PhaseDeploymentVerification ProcessingPhase = "DeploymentVerification"
	PhaseCompleted              ProcessingPhase = "Completed"
	PhaseFailed                 ProcessingPhase = "Failed"
)

// ComponentStatus represents the status of a component
type ComponentStatus struct {
	Type       ComponentType          `json:"type"`
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	Healthy    bool                   `json:"healthy"`
	LastUpdate time.Time              `json:"lastUpdate"`
	Version    string                 `json:"version,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Metrics    map[string]float64     `json:"metrics,omitempty"`
	Errors     []string               `json:"errors,omitempty"`
}
