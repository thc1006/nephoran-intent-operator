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
	"fmt"
	"time"
)

// ComponentType represents different types of components in the system
type ComponentType string

const (
	// Legacy component types for backward compatibility
	ComponentTypeLLMProcessor       ComponentType = "llm-processor"
	ComponentTypeResourcePlanner    ComponentType = "resource-planner"
	ComponentTypeManifestGenerator  ComponentType = "manifest-generator"
	ComponentTypeGitOpsController   ComponentType = "gitops-controller"
	ComponentTypeDeploymentVerifier ComponentType = "deployment-verifier"

	// New component type constants required by interfaces package
	ComponentUnknown    ComponentType = "unknown"
	ComponentIntentAPI  ComponentType = "intent-api"
	ComponentVIMSIM     ComponentType = "vim-sim"
	ComponentNEPHIO     ComponentType = "nephio"
	ComponentORAN       ComponentType = "oran"
	ComponentCACHE      ComponentType = "cache"
	ComponentCONTROLLER ComponentType = "controller"
	ComponentMETRICS    ComponentType = "metrics"
	ComponentPERF       ComponentType = "perf"
)

// ProcessingPhase represents the current phase of intent processing
type ProcessingPhase string

const (
	// Legacy phase constants for backward compatibility
	PhaseIntentReceived         ProcessingPhase = "IntentReceived"
	PhaseLLMProcessing          ProcessingPhase = "LLMProcessing"
	PhaseResourcePlanning       ProcessingPhase = "ResourcePlanning"
	PhaseManifestGeneration     ProcessingPhase = "ManifestGeneration"
	PhaseGitOpsCommit           ProcessingPhase = "GitOpsCommit"
	PhaseDeploymentVerification ProcessingPhase = "DeploymentVerification"
	PhaseCompleted              ProcessingPhase = "Completed"
	PhaseFailed                 ProcessingPhase = "Failed"

	// New phase constants required by interfaces package
	PhaseUnknown    ProcessingPhase = "Unknown"
	PhaseInit       ProcessingPhase = "Init"
	PhaseValidation ProcessingPhase = "Validation"
	PhaseProcessing ProcessingPhase = "Processing"
	PhaseComplete   ProcessingPhase = "Complete"
	PhaseError      ProcessingPhase = "Error"
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

// NetworkIntent represents a network intent contract
type NetworkIntent struct {
	ID              string                 `json:"id" validate:"required"`
	Type            string                 `json:"type" validate:"required,oneof=scaling deployment configuration"`
	Priority        int                    `json:"priority" validate:"min=0,max=10"`
	Description     string                 `json:"description" validate:"required,min=10,max=500"`
	Parameters      map[string]interface{} `json:"parameters" validate:"required"`
	TargetResources []string               `json:"target_resources" validate:"required,min=1"`
	Constraints     map[string]interface{} `json:"constraints,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Status          string                 `json:"status" validate:"required,oneof=pending processing completed failed"`
}

// ScalingIntent represents a scaling intent contract
type ScalingIntent struct {
	ID             string                 `json:"id" validate:"required"`
	ResourceType   string                 `json:"resource_type" validate:"required,oneof=deployment statefulset daemonset"`
	ResourceName   string                 `json:"resource_name" validate:"required"`
	Namespace      string                 `json:"namespace" validate:"required"`
	TargetScale    int                    `json:"target_scale" validate:"min=0,max=100"`
	CurrentScale   int                    `json:"current_scale" validate:"min=0"`
	ScaleDirection string                 `json:"scale_direction" validate:"required,oneof=up down auto"`
	Reason         string                 `json:"reason" validate:"required,min=5,max=200"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	Status         string                 `json:"status" validate:"required,oneof=pending in-progress completed failed"`
}

// IntentValidation provides validation rules for different intent types
type IntentValidation struct {
	RequiredFields  []string          `json:"required_fields"`
	OptionalFields  []string          `json:"optional_fields"`
	ValidationRules map[string]string `json:"validation_rules"`
	SchemaVersion   string            `json:"schema_version"`
}

// IntentContract defines the contract interface for all intent types
type IntentContract interface {
	GetID() string
	GetType() string
	GetStatus() string
	Validate() error
	SetStatus(status string)
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

// Implement IntentContract for NetworkIntent
func (ni *NetworkIntent) GetID() string           { return ni.ID }
func (ni *NetworkIntent) GetType() string         { return ni.Type }
func (ni *NetworkIntent) GetStatus() string       { return ni.Status }
func (ni *NetworkIntent) GetCreatedAt() time.Time { return ni.CreatedAt }
func (ni *NetworkIntent) GetUpdatedAt() time.Time { return ni.UpdatedAt }
func (ni *NetworkIntent) SetStatus(status string) {
	ni.Status = status
	ni.UpdatedAt = time.Now()
}

// Implement IntentContract for ScalingIntent
func (si *ScalingIntent) GetID() string           { return si.ID }
func (si *ScalingIntent) GetType() string         { return "scaling" }
func (si *ScalingIntent) GetStatus() string       { return si.Status }
func (si *ScalingIntent) GetCreatedAt() time.Time { return si.CreatedAt }
func (si *ScalingIntent) GetUpdatedAt() time.Time {
	if si.CompletedAt != nil {
		return *si.CompletedAt
	}
	return si.CreatedAt
}
func (si *ScalingIntent) SetStatus(status string) {
	si.Status = status
	now := time.Now()
	if status == "completed" || status == "failed" {
		si.CompletedAt = &now
	}
}

// Validate implementations
func (ni *NetworkIntent) Validate() error {
	if ni.ID == "" {
		return fmt.Errorf("network intent ID is required")
	}
	if ni.Type == "" {
		return fmt.Errorf("network intent type is required")
	}
	if len(ni.TargetResources) == 0 {
		return fmt.Errorf("at least one target resource is required")
	}
	if ni.Priority < 0 || ni.Priority > 10 {
		return fmt.Errorf("priority must be between 0 and 10")
	}
	return nil
}

func (si *ScalingIntent) Validate() error {
	if si.ID == "" {
		return fmt.Errorf("scaling intent ID is required")
	}
	if si.ResourceType == "" {
		return fmt.Errorf("resource type is required")
	}
	if si.ResourceName == "" {
		return fmt.Errorf("resource name is required")
	}
	if si.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if si.TargetScale < 0 {
		return fmt.Errorf("target scale cannot be negative")
	}
	if si.ScaleDirection == "" {
		return fmt.Errorf("scale direction is required")
	}
	return nil
}
