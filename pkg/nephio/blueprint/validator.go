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

package blueprint

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ghodss/yaml"
	"github.com/xeipuuv/gojsonschema"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// Validator handles blueprint validation and O-RAN compliance checking.

type Validator struct {
	config *BlueprintConfig

	logger *zap.Logger

	// Validation engines.

	kubernetesValidator *KubernetesValidator

	oranValidator *ORANValidator

	securityValidator *SecurityValidator

	policyValidator *PolicyValidator

	// Schema validation.

	schemaCache sync.Map

	schemaLoader gojsonschema.JSONLoader

	// O-RAN compliance checking.

	oranSpecifications map[string]*ORANSpecification

	complianceRules map[string]*ORANComplianceRule

	// Performance and caching.

	validationCache sync.Map

	httpClient *http.Client

	validationMutex sync.RWMutex
}

// ValidationResult represents the result of blueprint validation.

type ValidationResult struct {
	IsValid bool `json:"isValid"`

	Errors []ValidationError `json:"errors,omitempty"`

	Warnings []ValidationWarning `json:"warnings,omitempty"`

	ORANCompliance *ORANComplianceResult `json:"oranCompliance,omitempty"`

	SecurityCompliance *SecurityComplianceResult `json:"securityCompliance,omitempty"`

	PolicyCompliance *PolicyComplianceResult `json:"policyCompliance,omitempty"`

	ResourceValidation *ResourceValidationResult `json:"resourceValidation,omitempty"`

	// Metrics.

	ValidationDuration time.Duration `json:"validationDuration"`

	ValidatedFiles int `json:"validatedFiles"`

	ValidatedResources int `json:"validatedResources"`

	// Recommendations.

	Recommendations []ValidationRecommendation `json:"recommendations,omitempty"`
}

// ValidationError represents a validation error.

type ValidationError struct {
	Code string `json:"code"`

	Message string `json:"message"`

	Severity ErrorSeverity `json:"severity"`

	Source string `json:"source"`

	Field string `json:"field,omitempty"`

	Value interface{} `json:"value,omitempty"`

	Suggestion string `json:"suggestion,omitempty"`

	DocumentRef string `json:"documentRef,omitempty"`
}

// ValidationWarning represents a validation warning.

type ValidationWarning struct {
	Code string `json:"code"`

	Message string `json:"message"`

	Source string `json:"source"`

	Field string `json:"field,omitempty"`

	Suggestion string `json:"suggestion,omitempty"`
}

// ValidationRecommendation represents an improvement recommendation.

type ValidationRecommendation struct {
	Type RecommendationType `json:"type"`

	Title string `json:"title"`

	Description string `json:"description"`

	Impact ImpactLevel `json:"impact"`

	Effort EffortLevel `json:"effort"`

	Category string `json:"category"`

	Actions []RecommendedAction `json:"actions,omitempty"`
}

// RecommendedAction represents a recommendedaction.

type RecommendedAction struct {
	Action string `json:"action"`

	Parameters map[string]string `json:"parameters,omitempty"`

	Description string `json:"description"`
}

// Enums for validation types.

type (
	ErrorSeverity string

	// RecommendationType represents a recommendationtype.

	RecommendationType string

	// ImpactLevel represents a impactlevel.

	ImpactLevel string

	// EffortLevel represents a effortlevel.

	EffortLevel string
)

const (

	// SeverityError holds severityerror value.

	SeverityError ErrorSeverity = "error"

	// SeverityWarning holds severitywarning value.

	SeverityWarning ErrorSeverity = "warning"

	// SeverityInfo holds severityinfo value.

	SeverityInfo ErrorSeverity = "info"
)

const (

	// RecommendationSecurity holds recommendationsecurity value.

	RecommendationSecurity RecommendationType = "security"

	// RecommendationPerformance holds recommendationperformance value.

	RecommendationPerformance RecommendationType = "performance"

	// RecommendationReliability holds recommendationreliability value.

	RecommendationReliability RecommendationType = "reliability"

	// RecommendationCompliance holds recommendationcompliance value.

	RecommendationCompliance RecommendationType = "compliance"

	// RecommendationOptimization holds recommendationoptimization value.

	RecommendationOptimization RecommendationType = "optimization"
)

const (

	// ImpactHigh holds impacthigh value.

	ImpactHigh ImpactLevel = "high"

	// ImpactMedium holds impactmedium value.

	ImpactMedium ImpactLevel = "medium"

	// ImpactLow holds impactlow value.

	ImpactLow ImpactLevel = "low"
)

const (

	// EffortHigh holds efforthigh value.

	EffortHigh EffortLevel = "high"

	// EffortMedium holds effortmedium value.

	EffortMedium EffortLevel = "medium"

	// EffortLow holds effortlow value.

	EffortLow EffortLevel = "low"
)

// Compliance result structures.

type ORANComplianceResult struct {
	IsCompliant bool `json:"isCompliant"`

	ComplianceScore float64 `json:"complianceScore"`

	Interfaces map[string]InterfaceCompliance `json:"interfaces"`

	Specifications []SpecificationCompliance `json:"specifications"`

	Violations []ComplianceViolation `json:"violations,omitempty"`
}

// InterfaceCompliance represents a interfacecompliance.

type InterfaceCompliance struct {
	Interface string `json:"interface"` // A1, O1, O2, E2

	Compliant bool `json:"compliant"`

	Score float64 `json:"score"`

	Issues []string `json:"issues,omitempty"`

	Implemented bool `json:"implemented"`
}

// SpecificationCompliance represents a specificationcompliance.

type SpecificationCompliance struct {
	Specification string `json:"specification"` // O-RAN.WG1.O1, O-RAN.WG2.A1, etc.

	Version string `json:"version"`

	Compliant bool `json:"compliant"`

	Coverage float64 `json:"coverage"`

	MissingItems []string `json:"missingItems,omitempty"`
}

// ComplianceViolation represents a complianceviolation.

type ComplianceViolation struct {
	Rule string `json:"rule"`

	Description string `json:"description"`

	Severity ErrorSeverity `json:"severity"`

	Component string `json:"component"`

	Remediation string `json:"remediation"`
}

// SecurityComplianceResult represents a securitycomplianceresult.

type SecurityComplianceResult struct {
	IsCompliant bool `json:"isCompliant"`

	SecurityScore float64 `json:"securityScore"`

	Vulnerabilities []SecurityVulnerability `json:"vulnerabilities,omitempty"`

	MissingControls []SecurityControl `json:"missingControls,omitempty"`

	Recommendations []SecurityRecommendation `json:"recommendations,omitempty"`
}

// SecurityVulnerability represents a securityvulnerability.

type SecurityVulnerability struct {
	ID string `json:"id"`

	Title string `json:"title"`

	Description string `json:"description"`

	Severity ErrorSeverity `json:"severity"`

	CVSS float64 `json:"cvss,omitempty"`

	Component string `json:"component"`

	Mitigation string `json:"mitigation"`
}

// SecurityControl represents a securitycontrol.

type SecurityControl struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Category string `json:"category"`

	Description string `json:"description"`

	Required bool `json:"required"`
}

// SecurityRecommendation represents a securityrecommendation.

type SecurityRecommendation struct {
	Control string `json:"control"`

	Action string `json:"action"`

	Priority string `json:"priority"`

	Description string `json:"description"`
}

// PolicyComplianceResult represents a policycomplianceresult.

type PolicyComplianceResult struct {
	IsCompliant bool `json:"isCompliant"`

	PolicyScore float64 `json:"policyScore"`

	Violations []PolicyViolation `json:"violations,omitempty"`

	MissingPolicies []RequiredPolicy `json:"missingPolicies,omitempty"`
}

// PolicyViolation represents a policyviolation.

type PolicyViolation struct {
	Policy string `json:"policy"`

	Rule string `json:"rule"`

	Component string `json:"component"`

	Description string `json:"description"`

	Severity string `json:"severity"`
}

// RequiredPolicy represents a requiredpolicy.

type RequiredPolicy struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Description string `json:"description"`

	Required bool `json:"required"`
}

// ResourceValidationResult represents a resourcevalidationresult.

type ResourceValidationResult struct {
	IsValid bool `json:"isValid"`

	ResourceScore float64 `json:"resourceScore"`

	ResourceIssues []ResourceIssue `json:"resourceIssues,omitempty"`

	Recommendations []ResourceRecommendation `json:"recommendations,omitempty"`
}

// ResourceIssue represents a resourceissue.

type ResourceIssue struct {
	Resource string `json:"resource"`

	Issue string `json:"issue"`

	Severity ErrorSeverity `json:"severity"`

	Current string `json:"current,omitempty"`

	Recommended string `json:"recommended,omitempty"`
}

// ResourceRecommendation represents a resourcerecommendation.

type ResourceRecommendation struct {
	Resource string `json:"resource"`

	Action string `json:"action"`

	Reason string `json:"reason"`

	Impact string `json:"impact"`
}

// Validator components.

type KubernetesValidator struct {
	schemas map[string]gojsonschema.JSONLoader

	mutex sync.RWMutex
}

// ORANValidator represents a oranvalidator.

type ORANValidator struct {
	specifications map[string]*ORANSpecification

	interfaceRules map[string]*InterfaceRule

	mutex sync.RWMutex
}

// SecurityValidator represents a securityvalidator.

type SecurityValidator struct {
	securityRules map[string]*SecurityRule

	vulnerabilityDB map[string]*VulnerabilityInfo

	mutex sync.RWMutex
}

// PolicyValidator represents a policyvalidator.

type PolicyValidator struct {
	organizationalPolicies map[string]*OrganizationalPolicy

	compliancePolicies map[string]*CompliancePolicy

	mutex sync.RWMutex
}

// Rule and specification definitions.

type ORANSpecification struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Version string `json:"version"`

	Interfaces []string `json:"interfaces"`

	Requirements []ORANRequirement `json:"requirements"`

	Schema gojsonschema.JSONLoader `json:"-"`
}

// ORANRequirement represents a oranrequirement.

type ORANRequirement struct {
	ID string `json:"id"`

	Description string `json:"description"`

	Type string `json:"type"`

	Mandatory bool `json:"mandatory"`

	Validation string `json:"validation"`

	References []string `json:"references,omitempty"`
}

// InterfaceRule represents a interfacerule.

type InterfaceRule struct {
	Interface string `json:"interface"` // A1, O1, O2, E2

	Requirements []InterfaceRequirement `json:"requirements"`

	Schema gojsonschema.JSONLoader `json:"-"`
}

// InterfaceRequirement represents a interfacerequirement.

type InterfaceRequirement struct {
	Field string `json:"field"`

	Type string `json:"type"`

	Required bool `json:"required"`

	Pattern string `json:"pattern,omitempty"`

	MinValue interface{} `json:"minValue,omitempty"`

	MaxValue interface{} `json:"maxValue,omitempty"`

	AllowedValues []interface{} `json:"allowedValues,omitempty"`
}

// SecurityRule represents a securityrule.

type SecurityRule struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Category string `json:"category"`

	Severity ErrorSeverity `json:"severity"`

	Description string `json:"description"`

	Check SecurityCheck `json:"check"`

	Mitigation string `json:"mitigation"`
}

// SecurityCheck represents a securitycheck.

type SecurityCheck struct {
	Type string `json:"type"`

	Target string `json:"target"`

	Conditions map[string]interface{} `json:"conditions"`
}

// VulnerabilityInfo represents a vulnerabilityinfo.

type VulnerabilityInfo struct {
	ID string `json:"id"`

	CVSS float64 `json:"cvss"`

	Severity ErrorSeverity `json:"severity"`

	Description string `json:"description"`

	Affected []string `json:"affected"`

	Fixed []string `json:"fixed,omitempty"`
}

// OrganizationalPolicy represents a organizationalpolicy.

type OrganizationalPolicy struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Type string `json:"type"`

	Rules []OrganizationalRule `json:"rules"`

	Enforcement PolicyEnforcement `json:"enforcement"`
}

// OrganizationalRule represents a organizationalrule.

type OrganizationalRule struct {
	Condition string `json:"condition"`

	Action string `json:"action"`

	Message string `json:"message"`

	Severity ErrorSeverity `json:"severity"`
}

// CompliancePolicy represents a compliancepolicy.

type CompliancePolicy struct {
	Standard string `json:"standard"` // SOC2, ISO27001, etc.

	Version string `json:"version"`

	Controls []ComplianceControl `json:"controls"`
}

// ComplianceControl represents a compliancecontrol.

type ComplianceControl struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Required bool `json:"required"`

	Validation string `json:"validation"`
}

// PolicyEnforcement represents a policyenforcement.

type PolicyEnforcement struct {
	Mode string `json:"mode"` // warn, block, monitor

	Actions []string `json:"actions,omitempty"`
}

// ORANComplianceRule represents an O-RAN compliance rule.

type ORANComplianceRule struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Category string `json:"category"`

	Severity ErrorSeverity `json:"severity"`

	Description string `json:"description"`

	Check ComplianceCheck `json:"check"`

	Remediation string `json:"remediation"`

	Interfaces []string `json:"interfaces,omitempty"`
}

// ComplianceCheck represents a compliance check.

type ComplianceCheck struct {
	Type string `json:"type"`

	Target string `json:"target"`

	Conditions map[string]interface{} `json:"conditions"`
}

// NewValidator creates a new blueprint validator.

func NewValidator(config *BlueprintConfig, logger *zap.Logger) (*Validator, error) {

	if config == nil {

		config = DefaultBlueprintConfig()

	}

	if logger == nil {

		logger = zap.NewNop()

	}

	validator := &Validator{

		config: config,

		logger: logger,

		oranSpecifications: make(map[string]*ORANSpecification),

		complianceRules: make(map[string]*ORANComplianceRule),

		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	// Initialize sub-validators.

	var err error

	validator.kubernetesValidator, err = NewKubernetesValidator()

	if err != nil {

		return nil, fmt.Errorf("failed to create Kubernetes validator: %w", err)

	}

	validator.oranValidator, err = NewORANValidator()

	if err != nil {

		return nil, fmt.Errorf("failed to create O-RAN validator: %w", err)

	}

	validator.securityValidator, err = NewSecurityValidator()

	if err != nil {

		return nil, fmt.Errorf("failed to create security validator: %w", err)

	}

	validator.policyValidator, err = NewPolicyValidator()

	if err != nil {

		return nil, fmt.Errorf("failed to create policy validator: %w", err)

	}

	// Load O-RAN specifications and compliance rules.

	if err := validator.loadORANSpecifications(); err != nil {

		return nil, fmt.Errorf("failed to load O-RAN specifications: %w", err)

	}

	logger.Info("Blueprint validator initialized",

		zap.Bool("oran_compliance", config.EnableORANCompliance),

		zap.Int("oran_specifications", len(validator.oranSpecifications)),

		zap.Int("compliance_rules", len(validator.complianceRules)))

	return validator, nil

}

// ValidateBlueprint validates a blueprint against all configured rules and standards.

func (v *Validator) ValidateBlueprint(ctx context.Context, intent *v1.NetworkIntent, files map[string]string) (*ValidationResult, error) {

	startTime := time.Now()

	v.logger.Info("Validating blueprint",

		zap.String("intent_name", intent.Name),

		zap.String("intent_type", string(intent.Spec.IntentType)),

		zap.Int("files", len(files)))

	result := &ValidationResult{

		IsValid: true,

		Errors: []ValidationError{},

		Warnings: []ValidationWarning{},

		Recommendations: []ValidationRecommendation{},

		ValidatedFiles: len(files),

		ValidatedResources: 0,
	}

	// Validate individual files.

	for filename, content := range files {

		if err := v.validateFile(ctx, filename, content, result); err != nil {

			v.logger.Warn("File validation failed",

				zap.String("filename", filename),

				zap.Error(err))

		}

	}

	// Perform cross-file validations.

	if err := v.validateCrossFileConsistency(ctx, files, result); err != nil {

		v.logger.Warn("Cross-file validation failed", zap.Error(err))

	}

	// O-RAN compliance validation.

	if v.config.EnableORANCompliance {

		oranResult, err := v.validateORANCompliance(ctx, intent, files)

		if err != nil {

			v.logger.Warn("O-RAN compliance validation failed", zap.Error(err))

		} else {

			result.ORANCompliance = oranResult

			if !oranResult.IsCompliant {

				result.IsValid = false

			}

		}

	}

	// Security compliance validation.

	securityResult, err := v.validateSecurityCompliance(ctx, intent, files)

	if err != nil {

		v.logger.Warn("Security compliance validation failed", zap.Error(err))

	} else {

		result.SecurityCompliance = securityResult

		if !securityResult.IsCompliant {

			result.IsValid = false

		}

	}

	// Policy compliance validation.

	policyResult, err := v.validatePolicyCompliance(ctx, intent, files)

	if err != nil {

		v.logger.Warn("Policy compliance validation failed", zap.Error(err))

	} else {

		result.PolicyCompliance = policyResult

		if !policyResult.IsCompliant {

			result.IsValid = false

		}

	}

	// Resource validation.

	resourceResult, err := v.validateResourceRequirements(ctx, intent, files)

	if err != nil {

		v.logger.Warn("Resource validation failed", zap.Error(err))

	} else {

		result.ResourceValidation = resourceResult

		if !resourceResult.IsValid {

			result.IsValid = false

		}

	}

	// Generate recommendations.

	v.generateRecommendations(result)

	// Final validation status.

	if len(result.Errors) > 0 {

		for _, err := range result.Errors {

			if err.Severity == SeverityError {

				result.IsValid = false

				break

			}

		}

	}

	result.ValidationDuration = time.Since(startTime)

	v.logger.Info("Blueprint validation completed",

		zap.String("intent_name", intent.Name),

		zap.Bool("is_valid", result.IsValid),

		zap.Int("errors", len(result.Errors)),

		zap.Int("warnings", len(result.Warnings)),

		zap.Duration("duration", result.ValidationDuration))

	return result, nil

}

// validateFile validates an individual file.

func (v *Validator) validateFile(ctx context.Context, filename, content string, result *ValidationResult) error {

	// Skip non-YAML files.

	if !v.isYAMLFile(filename, content) {

		return nil

	}

	// Parse YAML content.

	var obj map[string]interface{}

	if err := yaml.Unmarshal([]byte(content), &obj); err != nil {

		result.Errors = append(result.Errors, ValidationError{

			Code: "YAML_PARSE_ERROR",

			Message: fmt.Sprintf("Failed to parse YAML: %v", err),

			Severity: SeverityError,

			Source: filename,
		})

		return err

	}

	result.ValidatedResources++

	// Kubernetes validation.

	if err := v.kubernetesValidator.ValidateResource(obj, filename, result); err != nil {

		v.logger.Debug("Kubernetes validation failed",

			zap.String("file", filename),

			zap.Error(err))

	}

	// Component-specific validation.

	if err := v.validateComponentSpecific(obj, filename, result); err != nil {

		v.logger.Debug("Component-specific validation failed",

			zap.String("file", filename),

			zap.Error(err))

	}

	return nil

}

// validateORANCompliance validates O-RAN compliance.

func (v *Validator) validateORANCompliance(ctx context.Context, intent *v1.NetworkIntent, files map[string]string) (*ORANComplianceResult, error) {

	result := &ORANComplianceResult{

		IsCompliant: true,

		Interfaces: make(map[string]InterfaceCompliance),

		Specifications: []SpecificationCompliance{},

		Violations: []ComplianceViolation{},
	}

	// Check each target component for O-RAN compliance.

	for _, component := range intent.Spec.TargetComponents {

		if err := v.oranValidator.ValidateComponent(convertNetworkTargetComponentToORANComponent(component), files, result); err != nil {

			v.logger.Warn("O-RAN component validation failed",

				zap.String("component", string(component)),

				zap.Error(err))

		}

	}

	// Validate O-RAN interfaces.

	v.validateORANInterfaces(files, result)

	// Calculate compliance score.

	result.ComplianceScore = v.calculateORANComplianceScore(result)

	return result, nil

}

// validateORANInterfaces validates O-RAN interface implementations.

func (v *Validator) validateORANInterfaces(files map[string]string, result *ORANComplianceResult) {

	interfaces := []string{"A1", "O1", "O2", "E2"}

	for _, interfaceName := range interfaces {

		compliance := InterfaceCompliance{

			Interface: interfaceName,

			Compliant: true,

			Score: 100.0,

			Issues: []string{},

			Implemented: false,
		}

		// Check if interface is implemented.

		for filename, content := range files {

			if v.containsInterfaceImplementation(content, interfaceName) {

				compliance.Implemented = true

				// Validate interface-specific requirements.

				if err := v.validateInterfaceImplementation(filename, content, interfaceName, &compliance); err != nil {

					v.logger.Debug("Interface validation failed",

						zap.String("interface", interfaceName),

						zap.String("file", filename),

						zap.Error(err))

				}

				break

			}

		}

		result.Interfaces[interfaceName] = compliance

		if !compliance.Compliant {

			result.IsCompliant = false

		}

	}

}

// validateSecurityCompliance validates security compliance.

func (v *Validator) validateSecurityCompliance(ctx context.Context, intent *v1.NetworkIntent, files map[string]string) (*SecurityComplianceResult, error) {

	result := &SecurityComplianceResult{

		IsCompliant: true,

		SecurityScore: 100.0,

		Vulnerabilities: []SecurityVulnerability{},

		MissingControls: []SecurityControl{},

		Recommendations: []SecurityRecommendation{},
	}

	// Validate each file for security issues.

	for filename, content := range files {

		if err := v.securityValidator.ValidateFile(filename, content, result); err != nil {

			v.logger.Debug("Security validation failed",

				zap.String("file", filename),

				zap.Error(err))

		}

	}

	// Calculate security score.

	result.SecurityScore = v.calculateSecurityScore(result)

	result.IsCompliant = result.SecurityScore >= 80.0 // 80% threshold

	return result, nil

}

// validatePolicyCompliance validates policy compliance.

func (v *Validator) validatePolicyCompliance(ctx context.Context, intent *v1.NetworkIntent, files map[string]string) (*PolicyComplianceResult, error) {

	result := &PolicyComplianceResult{

		IsCompliant: true,

		PolicyScore: 100.0,

		Violations: []PolicyViolation{},

		MissingPolicies: []RequiredPolicy{},
	}

	// Validate organizational policies.

	if err := v.policyValidator.ValidateOrganizationalPolicies(intent, files, result); err != nil {

		v.logger.Debug("Organizational policy validation failed", zap.Error(err))

	}

	// Validate compliance policies.

	if err := v.policyValidator.ValidateCompliancePolicies(intent, files, result); err != nil {

		v.logger.Debug("Compliance policy validation failed", zap.Error(err))

	}

	// Calculate policy score.

	result.PolicyScore = v.calculatePolicyScore(result)

	result.IsCompliant = len(result.Violations) == 0

	return result, nil

}

// validateResourceRequirements validates resource requirements and constraints.

func (v *Validator) validateResourceRequirements(ctx context.Context, intent *v1.NetworkIntent, files map[string]string) (*ResourceValidationResult, error) {

	result := &ResourceValidationResult{

		IsValid: true,

		ResourceScore: 100.0,

		ResourceIssues: []ResourceIssue{},

		Recommendations: []ResourceRecommendation{},
	}

	// Validate resource specifications in each file.

	for filename, content := range files {

		if v.isKubernetesManifest(content) {

			if err := v.validateResourceSpecifications(filename, content, intent, result); err != nil {

				v.logger.Debug("Resource specification validation failed",

					zap.String("file", filename),

					zap.Error(err))

			}

		}

	}

	// Validate against intent constraints.

	if intent.Spec.ResourceConstraints != nil {

		v.validateAgainstResourceConstraints(files, intent.Spec.ResourceConstraints, result)

	}

	// Calculate resource score.

	result.ResourceScore = v.calculateResourceScore(result)

	result.IsValid = len(result.ResourceIssues) == 0

	return result, nil

}

// Helper methods for specific validations.

func (v *Validator) validateComponentSpecific(obj map[string]interface{}, filename string, result *ValidationResult) error {

	kind, ok := obj["kind"].(string)

	if !ok {

		return nil

	}

	switch kind {

	case "Deployment":

		return v.validateDeployment(obj, filename, result)

	case "Service":

		return v.validateService(obj, filename, result)

	case "ConfigMap":

		return v.validateConfigMap(obj, filename, result)

	case "Secret":

		return v.validateSecret(obj, filename, result)

	case "NetworkPolicy":

		return v.validateNetworkPolicy(obj, filename, result)

	default:

		// Generic Kubernetes resource validation.

		return v.validateGenericKubernetesResource(obj, filename, result)

	}

}

func (v *Validator) validateDeployment(obj map[string]interface{}, filename string, result *ValidationResult) error {

	// Validate deployment-specific requirements.

	spec, ok := obj["spec"].(map[interface{}]interface{})

	if !ok {

		result.Errors = append(result.Errors, ValidationError{

			Code: "DEPLOYMENT_MISSING_SPEC",

			Message: "Deployment is missing spec section",

			Severity: SeverityError,

			Source: filename,
		})

		return nil

	}

	// Validate replicas.

	if replicas, ok := spec["replicas"]; ok {

		if replicasNum, ok := replicas.(int); ok {

			if replicasNum < 1 {

				result.Errors = append(result.Errors, ValidationError{

					Code: "DEPLOYMENT_INVALID_REPLICAS",

					Message: "Deployment replicas must be at least 1",

					Severity: SeverityError,

					Source: filename,

					Field: "spec.replicas",

					Value: replicasNum,
				})

			} else if replicasNum == 1 {

				result.Warnings = append(result.Warnings, ValidationWarning{

					Code: "DEPLOYMENT_SINGLE_REPLICA",

					Message: "Deployment has only 1 replica, consider increasing for high availability",

					Source: filename,

					Field: "spec.replicas",

					Suggestion: "Consider setting replicas to 2 or more for production deployments",
				})

			}

		}

	}

	// Validate template.

	if template, ok := spec["template"].(map[interface{}]interface{}); ok {

		if err := v.validatePodTemplate(template, filename, result); err != nil {

			return err

		}

	}

	return nil

}

func (v *Validator) validatePodTemplate(template map[interface{}]interface{}, filename string, result *ValidationResult) error {

	spec, ok := template["spec"].(map[interface{}]interface{})

	if !ok {

		result.Errors = append(result.Errors, ValidationError{

			Code: "POD_TEMPLATE_MISSING_SPEC",

			Message: "Pod template is missing spec section",

			Severity: SeverityError,

			Source: filename,
		})

		return nil

	}

	// Validate containers.

	if containers, ok := spec["containers"].([]interface{}); ok {

		for i, container := range containers {

			if err := v.validateContainer(container, i, filename, result); err != nil {

				return err

			}

		}

	} else {

		result.Errors = append(result.Errors, ValidationError{

			Code: "POD_TEMPLATE_MISSING_CONTAINERS",

			Message: "Pod template is missing containers",

			Severity: SeverityError,

			Source: filename,
		})

	}

	return nil

}

func (v *Validator) validateContainer(container interface{}, index int, filename string, result *ValidationResult) error {

	containerMap, ok := container.(map[interface{}]interface{})

	if !ok {

		return nil

	}

	// Validate image.

	if image, ok := containerMap["image"]; ok {

		if imageStr, ok := image.(string); ok {

			if err := v.validateContainerImage(imageStr, filename, result); err != nil {

				return err

			}

		}

	} else {

		result.Errors = append(result.Errors, ValidationError{

			Code: "CONTAINER_MISSING_IMAGE",

			Message: fmt.Sprintf("Container %d is missing image specification", index),

			Severity: SeverityError,

			Source: filename,

			Field: fmt.Sprintf("spec.template.spec.containers[%d].image", index),
		})

	}

	// Validate resource requirements.

	if resources, ok := containerMap["resources"].(map[interface{}]interface{}); ok {

		if err := v.validateContainerResources(resources, index, filename, result); err != nil {

			return err

		}

	} else {

		result.Warnings = append(result.Warnings, ValidationWarning{

			Code: "CONTAINER_MISSING_RESOURCES",

			Message: fmt.Sprintf("Container %d is missing resource requirements", index),

			Source: filename,

			Field: fmt.Sprintf("spec.template.spec.containers[%d].resources", index),

			Suggestion: "Define resource requests and limits for better resource management",
		})

	}

	// Validate security context.

	if err := v.validateContainerSecurity(containerMap, index, filename, result); err != nil {

		return err

	}

	return nil

}

func (v *Validator) validateContainerImage(image, filename string, result *ValidationResult) error {

	// Check for security best practices.

	if strings.Contains(image, ":latest") {

		result.Warnings = append(result.Warnings, ValidationWarning{

			Code: "CONTAINER_IMAGE_LATEST_TAG",

			Message: "Container image uses 'latest' tag",

			Source: filename,

			Suggestion: "Use specific version tags for better version control and security",
		})

	}

	// Check for private registry or known secure sources.

	if !v.isFromTrustedRegistry(image) {

		result.Warnings = append(result.Warnings, ValidationWarning{

			Code: "CONTAINER_IMAGE_UNTRUSTED_REGISTRY",

			Message: "Container image is not from a known trusted registry",

			Source: filename,

			Suggestion: "Consider using images from trusted registries for better security",
		})

	}

	return nil

}

func (v *Validator) validateContainerResources(resources map[interface{}]interface{}, index int, filename string, result *ValidationResult) error {

	// Validate requests.

	if requests, ok := resources["requests"].(map[interface{}]interface{}); ok {

		if cpu, ok := requests["cpu"]; ok {

			if cpuStr, ok := cpu.(string); ok {

				if _, err := resource.ParseQuantity(cpuStr); err != nil {

					result.Errors = append(result.Errors, ValidationError{

						Code: "CONTAINER_INVALID_CPU_REQUEST",

						Message: fmt.Sprintf("Container %d has invalid CPU request: %v", index, err),

						Severity: SeverityError,

						Source: filename,

						Field: fmt.Sprintf("spec.template.spec.containers[%d].resources.requests.cpu", index),

						Value: cpuStr,
					})

				}

			}

		}

		if memory, ok := requests["memory"]; ok {

			if memoryStr, ok := memory.(string); ok {

				if _, err := resource.ParseQuantity(memoryStr); err != nil {

					result.Errors = append(result.Errors, ValidationError{

						Code: "CONTAINER_INVALID_MEMORY_REQUEST",

						Message: fmt.Sprintf("Container %d has invalid memory request: %v", index, err),

						Severity: SeverityError,

						Source: filename,

						Field: fmt.Sprintf("spec.template.spec.containers[%d].resources.requests.memory", index),

						Value: memoryStr,
					})

				}

			}

		}

	}

	// Validate limits.

	if limits, ok := resources["limits"].(map[interface{}]interface{}); ok {

		if cpu, ok := limits["cpu"]; ok {

			if cpuStr, ok := cpu.(string); ok {

				if _, err := resource.ParseQuantity(cpuStr); err != nil {

					result.Errors = append(result.Errors, ValidationError{

						Code: "CONTAINER_INVALID_CPU_LIMIT",

						Message: fmt.Sprintf("Container %d has invalid CPU limit: %v", index, err),

						Severity: SeverityError,

						Source: filename,

						Field: fmt.Sprintf("spec.template.spec.containers[%d].resources.limits.cpu", index),

						Value: cpuStr,
					})

				}

			}

		}

	}

	return nil

}

func (v *Validator) validateContainerSecurity(container map[interface{}]interface{}, index int, filename string, result *ValidationResult) error {

	// Check for security context.

	securityContext, hasSecurityContext := container["securityContext"].(map[interface{}]interface{})

	// Warn if running as root.

	if !hasSecurityContext {

		result.Warnings = append(result.Warnings, ValidationWarning{

			Code: "CONTAINER_MISSING_SECURITY_CONTEXT",

			Message: fmt.Sprintf("Container %d is missing security context", index),

			Source: filename,

			Field: fmt.Sprintf("spec.template.spec.containers[%d].securityContext", index),

			Suggestion: "Define security context to improve security posture",
		})

	} else {

		// Check for non-root user.

		if runAsUser, ok := securityContext["runAsUser"]; ok {

			if userID, ok := runAsUser.(int); ok && userID == 0 {

				result.Warnings = append(result.Warnings, ValidationWarning{

					Code: "CONTAINER_RUNS_AS_ROOT",

					Message: fmt.Sprintf("Container %d runs as root user", index),

					Source: filename,

					Field: fmt.Sprintf("spec.template.spec.containers[%d].securityContext.runAsUser", index),

					Suggestion: "Use non-root user for better security",
				})

			}

		}

		// Check for privileged mode.

		if privileged, ok := securityContext["privileged"]; ok {

			if isPrivileged, ok := privileged.(bool); ok && isPrivileged {

				result.Warnings = append(result.Warnings, ValidationWarning{

					Code: "CONTAINER_PRIVILEGED",

					Message: fmt.Sprintf("Container %d runs in privileged mode", index),

					Source: filename,

					Field: fmt.Sprintf("spec.template.spec.containers[%d].securityContext.privileged", index),

					Suggestion: "Avoid privileged mode unless absolutely necessary",
				})

			}

		}

	}

	return nil

}

// Utility methods.

func (v *Validator) isYAMLFile(filename, content string) bool {

	return strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") ||

		(strings.Contains(content, "apiVersion:") && strings.Contains(content, "kind:"))

}

func (v *Validator) isKubernetesManifest(content string) bool {

	return strings.Contains(content, "apiVersion:") && strings.Contains(content, "kind:")

}

func (v *Validator) isFromTrustedRegistry(image string) bool {

	trustedRegistries := []string{

		"docker.io",

		"gcr.io",

		"quay.io",

		"registry.k8s.io",

		"k8s.gcr.io",

		"docker.elastic.co",

		"registry.redhat.io",
	}

	for _, registry := range trustedRegistries {

		if strings.HasPrefix(image, registry) {

			return true

		}

	}

	// Check if it's an official image (no registry prefix).

	if !strings.Contains(image, "/") || strings.Count(image, "/") == 1 {

		return true

	}

	return false

}

func (v *Validator) containsInterfaceImplementation(content, interfaceName string) bool {

	patterns := map[string][]string{

		"A1": {"a1", "policy", "near-rt-ric"},

		"O1": {"o1", "netconf", "yang", "fcaps"},

		"O2": {"o2", "infrastructure", "cloud"},

		"E2": {"e2", "subscription", "indication"},
	}

	if interfacePatterns, ok := patterns[interfaceName]; ok {

		contentLower := strings.ToLower(content)

		for _, pattern := range interfacePatterns {

			if strings.Contains(contentLower, pattern) {

				return true

			}

		}

	}

	return false

}

// Score calculation methods.

func (v *Validator) calculateORANComplianceScore(result *ORANComplianceResult) float64 {

	if len(result.Interfaces) == 0 {

		return 0.0

	}

	totalScore := 0.0

	for _, compliance := range result.Interfaces {

		totalScore += compliance.Score

	}

	return totalScore / float64(len(result.Interfaces))

}

func (v *Validator) calculateSecurityScore(result *SecurityComplianceResult) float64 {

	baseScore := 100.0

	// Deduct points for vulnerabilities.

	for _, vuln := range result.Vulnerabilities {

		switch vuln.Severity {

		case SeverityError:

			baseScore -= 10.0

		case SeverityWarning:

			baseScore -= 5.0

		case SeverityInfo:

			baseScore -= 2.0

		}

	}

	// Deduct points for missing controls.

	baseScore -= float64(len(result.MissingControls)) * 3.0

	if baseScore < 0 {

		return 0.0

	}

	return baseScore

}

func (v *Validator) calculatePolicyScore(result *PolicyComplianceResult) float64 {

	baseScore := 100.0

	// Deduct points for violations.

	for _, violation := range result.Violations {

		switch violation.Severity {

		case "critical":

			baseScore -= 15.0

		case "high":

			baseScore -= 10.0

		case "medium":

			baseScore -= 5.0

		case "low":

			baseScore -= 2.0

		}

	}

	// Deduct points for missing policies.

	baseScore -= float64(len(result.MissingPolicies)) * 5.0

	if baseScore < 0 {

		return 0.0

	}

	return baseScore

}

func (v *Validator) calculateResourceScore(result *ResourceValidationResult) float64 {

	baseScore := 100.0

	for _, issue := range result.ResourceIssues {

		switch issue.Severity {

		case SeverityError:

			baseScore -= 10.0

		case SeverityWarning:

			baseScore -= 5.0

		case SeverityInfo:

			baseScore -= 2.0

		}

	}

	if baseScore < 0 {

		return 0.0

	}

	return baseScore

}

// HealthCheck performs health check on the validator.

func (v *Validator) HealthCheck(ctx context.Context) bool {

	// Check if sub-validators are available.

	if v.kubernetesValidator == nil || v.oranValidator == nil ||

		v.securityValidator == nil || v.policyValidator == nil {

		v.logger.Warn("One or more sub-validators are not available")

		return false

	}

	// Check if O-RAN specifications are loaded.

	if v.config.EnableORANCompliance && len(v.oranSpecifications) == 0 {

		v.logger.Warn("O-RAN specifications not loaded")

		return false

	}

	return true

}

// Placeholder implementations for complex components that would need full implementation.

func NewKubernetesValidator() (*KubernetesValidator, error) {

	return &KubernetesValidator{

		schemas: make(map[string]gojsonschema.JSONLoader),
	}, nil

}

// NewORANValidator performs neworanvalidator operation.

func NewORANValidator() (*ORANValidator, error) {

	return &ORANValidator{

		specifications: make(map[string]*ORANSpecification),

		interfaceRules: make(map[string]*InterfaceRule),
	}, nil

}

// NewSecurityValidator performs newsecurityvalidator operation.

func NewSecurityValidator() (*SecurityValidator, error) {

	return &SecurityValidator{

		securityRules: make(map[string]*SecurityRule),

		vulnerabilityDB: make(map[string]*VulnerabilityInfo),
	}, nil

}

// NewPolicyValidator performs newpolicyvalidator operation.

func NewPolicyValidator() (*PolicyValidator, error) {

	return &PolicyValidator{

		organizationalPolicies: make(map[string]*OrganizationalPolicy),

		compliancePolicies: make(map[string]*CompliancePolicy),
	}, nil

}

// loadORANSpecifications loads O-RAN specifications from configuration.

func (v *Validator) loadORANSpecifications() error {

	// Stub implementation - would load from files or remote sources.

	v.logger.Debug("Loading O-RAN specifications (stub)")

	return nil

}

// validateCrossFileConsistency validates consistency across multiple files.

func (v *Validator) validateCrossFileConsistency(ctx context.Context, files map[string]string, result *ValidationResult) error {

	// Stub implementation - would check for naming consistency, reference integrity, etc.

	v.logger.Debug("Validating cross-file consistency (stub)")

	return nil

}

// generateRecommendations generates improvement recommendations.

func (v *Validator) generateRecommendations(result *ValidationResult) {

	// Stub implementation - would analyze validation results and generate recommendations.

	v.logger.Debug("Generating recommendations (stub)")

}

// validateInterfaceImplementation validates specific O-RAN interface implementation.

func (v *Validator) validateInterfaceImplementation(filename, content, interfaceName string, compliance *InterfaceCompliance) error {

	// Stub implementation - would validate interface-specific requirements.

	v.logger.Debug("Validating interface implementation (stub)",

		zap.String("interface", interfaceName),

		zap.String("file", filename))

	return nil

}

// validateResourceSpecifications validates Kubernetes resource specifications.

func (v *Validator) validateResourceSpecifications(filename, content string, intent *v1.NetworkIntent, result *ResourceValidationResult) error {

	// Stub implementation - would validate resource limits, requests, etc.

	v.logger.Debug("Validating resource specifications (stub)", zap.String("file", filename))

	return nil

}

// validateAgainstResourceConstraints validates against intent resource constraints.

func (v *Validator) validateAgainstResourceConstraints(files map[string]string, constraints *v1.ResourceConstraints, result *ResourceValidationResult) {

	// Stub implementation - would validate against specified constraints.

	v.logger.Debug("Validating against resource constraints (stub)")

}

// KubernetesValidator method stubs.

func (kv *KubernetesValidator) ValidateResource(obj map[string]interface{}, filename string, result *ValidationResult) error {

	// Stub implementation.

	return nil

}

// ORANValidator method stubs.

func (ov *ORANValidator) ValidateComponent(component v1.ORANComponent, files map[string]string, result *ORANComplianceResult) error {

	// Stub implementation.

	return nil

}

// SecurityValidator method stubs.

func (sv *SecurityValidator) ValidateFile(filename, content string, result *SecurityComplianceResult) error {

	// Stub implementation.

	return nil

}

// PolicyValidator method stubs.

func (pv *PolicyValidator) ValidateOrganizationalPolicies(intent *v1.NetworkIntent, files map[string]string, result *PolicyComplianceResult) error {

	// Stub implementation.

	return nil

}

// ValidateCompliancePolicies performs validatecompliancepolicies operation.

func (pv *PolicyValidator) ValidateCompliancePolicies(intent *v1.NetworkIntent, files map[string]string, result *PolicyComplianceResult) error {

	// Stub implementation.

	return nil

}

// Additional helper methods for service, configmap, secret validation.

func (v *Validator) validateService(obj map[string]interface{}, filename string, result *ValidationResult) error {

	// Stub implementation.

	return nil

}

func (v *Validator) validateConfigMap(obj map[string]interface{}, filename string, result *ValidationResult) error {

	// Stub implementation.

	return nil

}

func (v *Validator) validateSecret(obj map[string]interface{}, filename string, result *ValidationResult) error {

	// Stub implementation.

	return nil

}

func (v *Validator) validateNetworkPolicy(obj map[string]interface{}, filename string, result *ValidationResult) error {

	// Stub implementation.

	return nil

}

func (v *Validator) validateGenericKubernetesResource(obj map[string]interface{}, filename string, result *ValidationResult) error {

	// Stub implementation.

	return nil

}
