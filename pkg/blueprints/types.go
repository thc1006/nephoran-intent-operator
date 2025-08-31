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

package blueprints

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// NetworkFunctionConfigGenerator generates network function configurations.

type NetworkFunctionConfigGenerator struct {
	config *BlueprintConfig

	logger *zap.Logger
}

// NewNetworkFunctionConfigGenerator creates a new network function config generator.

func NewNetworkFunctionConfigGenerator(config *BlueprintConfig, logger *zap.Logger) (*NetworkFunctionConfigGenerator, error) {

	if config == nil {

		return nil, fmt.Errorf("config is required")

	}

	if logger == nil {

		logger = zap.NewNop()

	}

	return &NetworkFunctionConfigGenerator{

		config: config,

		logger: logger,
	}, nil

}

// GenerateConfigurations generates network function configurations from intent and templates.

func (nfcg *NetworkFunctionConfigGenerator) GenerateConfigurations(

	ctx context.Context,

	intent *v1.NetworkIntent,

	templates []*BlueprintTemplate,

) ([]NetworkFunctionConfig, error) {

	nfcg.logger.Info("Generating network function configurations",

		zap.String("intent_name", intent.Name),

		zap.Int("template_count", len(templates)))

	configs := make([]NetworkFunctionConfig, 0, len(templates)) // Preallocate with capacity

	for _, template := range templates {

		if template.NetworkConfig != nil {

			config := NetworkFunctionConfig{

				Interfaces: template.NetworkConfig.Interfaces,

				ServiceBindings: template.NetworkConfig.ServiceBindings,

				ResourceRequirements: template.NetworkConfig.ResourceRequirements,
			}

			configs = append(configs, config)

		}

	}

	nfcg.logger.Info("Generated network function configurations",

		zap.String("intent_name", intent.Name),

		zap.Int("config_count", len(configs)))

	return configs, nil

}

// ORANValidator validates O-RAN compliance.

type ORANValidator struct {
	config *BlueprintConfig

	logger *zap.Logger
}

// NewORANValidator creates a new O-RAN validator.

func NewORANValidator(config *BlueprintConfig, logger *zap.Logger) (*ORANValidator, error) {

	if config == nil {

		return nil, fmt.Errorf("config is required")

	}

	if logger == nil {

		logger = zap.NewNop()

	}

	return &ORANValidator{

		config: config,

		logger: logger,
	}, nil

}

// ValidateORANCompliance validates O-RAN compliance for rendered blueprint and network function configs.

func (ov *ORANValidator) ValidateORANCompliance(

	ctx context.Context,

	blueprint *RenderedBlueprint,

	nfConfigs []NetworkFunctionConfig,

) error {

	ov.logger.Info("Validating O-RAN compliance",

		zap.String("blueprint_name", blueprint.Name),

		zap.Int("nf_config_count", len(nfConfigs)))

	// Basic O-RAN compliance validation.

	if !blueprint.ORANCompliant {

		return fmt.Errorf("blueprint is not marked as O-RAN compliant")

	}

	// Validate required O-RAN interfaces.

	requiredInterfaces := []string{"a1", "o1", "o2", "e2"}

	for _, iface := range requiredInterfaces {

		if !ov.hasInterface(blueprint, iface) {

			ov.logger.Warn("Missing O-RAN interface", zap.String("interface", iface))

		}

	}

	// Validate network function configurations.

	for _, nfConfig := range nfConfigs {

		if err := ov.validateNetworkFunctionConfig(nfConfig); err != nil {

			return fmt.Errorf("network function validation failed: %w", err)

		}

	}

	ov.logger.Info("O-RAN compliance validation completed successfully",

		zap.String("blueprint_name", blueprint.Name))

	return nil

}

// hasInterface checks if blueprint has a specific interface.

func (ov *ORANValidator) hasInterface(blueprint *RenderedBlueprint, interfaceType string) bool {

	if blueprint.Metadata == nil {

		return false

	}

	for _, iface := range blueprint.Metadata.InterfaceTypes {

		if iface == interfaceType {

			return true

		}

	}

	return false

}

// validateNetworkFunctionConfig validates a network function configuration.

func (ov *ORANValidator) validateNetworkFunctionConfig(nfConfig NetworkFunctionConfig) error {

	// Validate interfaces are present.

	if len(nfConfig.Interfaces) == 0 {

		return fmt.Errorf("network function must have at least one interface")

	}

	// Validate service bindings.

	if len(nfConfig.ServiceBindings) == 0 {

		return fmt.Errorf("network function must have at least one service binding")

	}

	// Validate resource requirements.

	if nfConfig.ResourceRequirements.CPU == "" {

		return fmt.Errorf("CPU resource requirement is required")

	}

	if nfConfig.ResourceRequirements.Memory == "" {

		return fmt.Errorf("memory resource requirement is required")

	}

	return nil

}

// TemplateEngine processes blueprint templates.

type TemplateEngine struct {
	config *BlueprintConfig

	logger *zap.Logger
}

// NewTemplateEngine creates a new template engine.

func NewTemplateEngine(config *BlueprintConfig, logger *zap.Logger) (*TemplateEngine, error) {

	if config == nil {

		return nil, fmt.Errorf("config is required")

	}

	if logger == nil {

		logger = zap.NewNop()

	}

	return &TemplateEngine{

		config: config,

		logger: logger,
	}, nil

}

// ProcessTemplate processes a blueprint template.

func (te *TemplateEngine) ProcessTemplate(

	ctx context.Context,

	template *BlueprintTemplate,

	parameters map[string]interface{},

) (*ProcessedTemplate, error) {

	te.logger.Info("Processing template",

		zap.String("template_name", template.Name),

		zap.String("template_version", template.Version))

	// Create processed template.

	processed := &ProcessedTemplate{

		ID: template.ID,

		Name: template.Name,

		Version: template.Version,

		ProcessedAt: time.Now(),

		Parameters: parameters,

		GeneratedFiles: make(map[string]string),
	}

	// Process template components.

	for _, krmTemplate := range template.KRMResources {

		// Simple template processing - in real implementation this would use template engines.

		processed.GeneratedFiles[krmTemplate.Kind+".yaml"] = "# Generated from template: " + template.Name

	}

	te.logger.Info("Template processing completed",

		zap.String("template_name", template.Name),

		zap.Int("generated_files", len(processed.GeneratedFiles)))

	return processed, nil

}

// ProcessedTemplate represents a processed blueprint template.

type ProcessedTemplate struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Version string `json:"version"`

	ProcessedAt time.Time `json:"processedAt"`

	Parameters map[string]interface{} `json:"parameters"`

	GeneratedFiles map[string]string `json:"generatedFiles"`
}

// BlueprintOperation represents a blueprint operation.

type BlueprintOperation struct {
	ID string `json:"id"`

	Type BlueprintOperationType `json:"type"`

	Intent *v1.NetworkIntent `json:"intent"`

	Templates []*BlueprintTemplate `json:"templates"`

	Parameters map[string]interface{} `json:"parameters"`

	Status OperationStatus `json:"status"`

	CreatedAt time.Time `json:"createdAt"`

	StartedAt *time.Time `json:"startedAt,omitempty"`

	CompletedAt *time.Time `json:"completedAt,omitempty"`

	Error string `json:"error,omitempty"`

	Result *OperationResult `json:"result,omitempty"`
}

// BlueprintOperationType represents the type of blueprint operation.

type BlueprintOperationType string

const (

	// BlueprintOperationTypeRender renders a blueprint.

	BlueprintOperationTypeRender BlueprintOperationType = "render"

	// BlueprintOperationTypeValidate validates a blueprint.

	BlueprintOperationTypeValidate BlueprintOperationType = "validate"

	// BlueprintOperationTypeDeploy deploys a blueprint.

	BlueprintOperationTypeDeploy BlueprintOperationType = "deploy"
)

// OperationStatus represents the status of a blueprint operation.

type OperationStatus string

const (

	// OperationStatusPending indicates operation is pending.

	OperationStatusPending OperationStatus = "pending"

	// OperationStatusRunning indicates operation is running.

	OperationStatusRunning OperationStatus = "running"

	// OperationStatusCompleted indicates operation completed successfully.

	OperationStatusCompleted OperationStatus = "completed"

	// OperationStatusFailed indicates operation failed.

	OperationStatusFailed OperationStatus = "failed"
)

// OperationResult represents the result of a blueprint operation.

type OperationResult struct {
	RenderedBlueprint *RenderedBlueprint `json:"renderedBlueprint,omitempty"`

	ValidationResult *ValidationResult `json:"validationResult,omitempty"`

	DeploymentResult *DeploymentResult `json:"deploymentResult,omitempty"`

	GeneratedFiles map[string]string `json:"generatedFiles,omitempty"`

	ProcessedTemplates []*ProcessedTemplate `json:"processedTemplates,omitempty"`

	NfConfigs []NetworkFunctionConfig `json:"nfConfigs,omitempty"`
}

// ValidationResult represents the result of blueprint validation.

type ValidationResult struct {
	Valid bool `json:"valid"`

	Errors []string `json:"errors,omitempty"`

	Warnings []string `json:"warnings,omitempty"`

	Duration time.Duration `json:"duration"`

	ValidatedAt time.Time `json:"validatedAt"`
}

// DeploymentResult represents the result of blueprint deployment.

type DeploymentResult struct {
	Success bool `json:"success"`

	ResourcesCreated []string `json:"resourcesCreated,omitempty"`

	Errors []string `json:"errors,omitempty"`

	Duration time.Duration `json:"duration"`

	DeployedAt time.Time `json:"deployedAt"`
}
