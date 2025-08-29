// Package a1 implements comprehensive JSON schema validation for O-RAN A1 interfaces.

// This module provides validation services for policy types, instances, and other A1 entities.

package a1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"

	validator "github.com/go-playground/validator/v10"
	"github.com/nephio-project/nephoran-intent-operator/pkg/logging"
)

// A1ValidatorImpl implements comprehensive validation for A1 entities.

type A1ValidatorImpl struct {
	validator *validator.Validate

	logger *logging.StructuredLogger

	config *ValidationConfig

	schemaRegistry map[string]interface{} // Cache for JSON schemas

}

// ValidationResult, ValidationError, and ValidationWarning are defined in types.go.

// JSONSchemaValidator provides JSON schema validation capabilities.

type JSONSchemaValidator struct {
	Draft string `json:"$schema,omitempty"`

	ID string `json:"$id,omitempty"`

	Title string `json:"title,omitempty"`

	Description string `json:"description,omitempty"`

	Type interface{} `json:"type,omitempty"`

	Properties map[string]interface{} `json:"properties,omitempty"`

	Required []string `json:"required,omitempty"`

	AdditionalProps interface{} `json:"additionalProperties,omitempty"`

	PatternProps map[string]interface{} `json:"patternProperties,omitempty"`

	Dependencies map[string]interface{} `json:"dependencies,omitempty"`

	AllOf []interface{} `json:"allOf,omitempty"`

	AnyOf []interface{} `json:"anyOf,omitempty"`

	OneOf []interface{} `json:"oneOf,omitempty"`

	Not interface{} `json:"not,omitempty"`

	If interface{} `json:"if,omitempty"`

	Then interface{} `json:"then,omitempty"`

	Else interface{} `json:"else,omitempty"`

	Enum []interface{} `json:"enum,omitempty"`

	Const interface{} `json:"const,omitempty"`

	MultipleOf interface{} `json:"multipleOf,omitempty"`

	Maximum interface{} `json:"maximum,omitempty"`

	ExclusiveMaximum interface{} `json:"exclusiveMaximum,omitempty"`

	Minimum interface{} `json:"minimum,omitempty"`

	ExclusiveMinimum interface{} `json:"exclusiveMinimum,omitempty"`

	MaxLength interface{} `json:"maxLength,omitempty"`

	MinLength interface{} `json:"minLength,omitempty"`

	Pattern string `json:"pattern,omitempty"`

	MaxItems interface{} `json:"maxItems,omitempty"`

	MinItems interface{} `json:"minItems,omitempty"`

	UniqueItems interface{} `json:"uniqueItems,omitempty"`

	Contains interface{} `json:"contains,omitempty"`

	MaxProperties interface{} `json:"maxProperties,omitempty"`

	MinProperties interface{} `json:"minProperties,omitempty"`

	Format string `json:"format,omitempty"`

	ContentMediaType string `json:"contentMediaType,omitempty"`

	ContentEncoding string `json:"contentEncoding,omitempty"`

	Examples []interface{} `json:"examples,omitempty"`

	Default interface{} `json:"default,omitempty"`
}

// NewA1Validator creates a new A1 validator with the given configuration.

func NewA1Validator(config *ValidationConfig, logger *logging.StructuredLogger) *A1ValidatorImpl {

	if config == nil {

		config = &ValidationConfig{

			EnableSchemaValidation: true,

			StrictValidation: false,

			ValidateAdditionalFields: true,
		}

	}

	if logger == nil {

		logger = logging.NewStructuredLogger(logging.DefaultConfig("a1-validator", "1.0.0", "development"))

	}

	v := validator.New()

	// Register custom validation tags.

	v.RegisterValidation("oran_policy_type_id", validatePolicyTypeID)

	v.RegisterValidation("oran_policy_id", validatePolicyID)

	v.RegisterValidation("oran_ei_type_id", validateEITypeID)

	v.RegisterValidation("oran_ei_job_id", validateEIJobID)

	v.RegisterValidation("oran_consumer_id", validateConsumerID)

	v.RegisterValidation("json_schema", validateJSONSchema)

	v.RegisterValidation("uri_reference", validateURIReference)

	v.RegisterValidation("iso8601", validateISO8601DateTime)

	// Register custom type function for better field names.

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {

		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]

		if name == "-" {

			return ""

		}

		return name

	})

	return &A1ValidatorImpl{

		validator: v,

		logger: logger.WithComponent("a1-validator"),

		config: config,

		schemaRegistry: make(map[string]interface{}),
	}

}

// ValidatePolicyType validates an A1 policy type according to O-RAN specifications.

func (av *A1ValidatorImpl) ValidatePolicyType(policyType *PolicyType) *ValidationResult {

	result := &ValidationResult{Valid: true}

	if policyType == nil {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Field: "policy_type",

			Message: "Policy type cannot be nil",
		})

		return result

	}

	// Structural validation using validator tags.

	if err := av.validator.Struct(policyType); err != nil {

		result.Valid = false

		result.Errors = append(result.Errors, av.convertValidationErrors(err)...)

	}

	// JSON Schema validation.

	if av.config.EnableSchemaValidation && policyType.Schema != nil {

		if schemaResult := av.validateJSONSchemaStructure(policyType.Schema, "policy_type.schema"); !schemaResult.Valid {

			result.Valid = false

			result.Errors = append(result.Errors, schemaResult.Errors...)

			result.Warnings = append(result.Warnings, schemaResult.Warnings...)

		}

	}

	// Business logic validation.

	if businessResult := av.validatePolicyTypeBusinessRules(policyType); !businessResult.Valid {

		result.Valid = false

		result.Errors = append(result.Errors, businessResult.Errors...)

		result.Warnings = append(result.Warnings, businessResult.Warnings...)

	}

	return result

}

// ValidatePolicyInstance validates an A1 policy instance.

func (av *A1ValidatorImpl) ValidatePolicyInstance(policyTypeID int, instance *PolicyInstance) *ValidationResult {

	result := &ValidationResult{Valid: true}

	if instance == nil {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Field: "policy_instance",

			Message: "Policy instance cannot be nil",
		})

		return result

	}

	// Structural validation.

	if err := av.validator.Struct(instance); err != nil {

		result.Valid = false

		result.Errors = append(result.Errors, av.convertValidationErrors(err)...)

	}

	// Cross-reference validation.

	if instance.PolicyTypeID != policyTypeID {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Field: "policy_type_id",

			Value: instance.PolicyTypeID,

			Message: fmt.Sprintf("Policy instance type ID %d does not match expected type ID %d", instance.PolicyTypeID, policyTypeID),

			FieldPath: "policy_instance.policy_type_id",
		})

	}

	// Policy data validation against schema (if available in registry).

	if av.config.EnableSchemaValidation {

		if schema, exists := av.schemaRegistry[fmt.Sprintf("policy_type_%d", policyTypeID)]; exists {

			if dataResult := av.validateDataAgainstSchema(instance.PolicyData, schema, "policy_data"); !dataResult.Valid {

				result.Valid = false

				result.Errors = append(result.Errors, dataResult.Errors...)

				result.Warnings = append(result.Warnings, dataResult.Warnings...)

			}

		} else {

			result.Warnings = append(result.Warnings, ValidationWarning{

				Field: "policy_data",

				Message: fmt.Sprintf("Schema for policy type %d not found in registry, data validation skipped", policyTypeID),
			})

		}

	}

	// Business logic validation.

	if businessResult := av.validatePolicyInstanceBusinessRules(instance); !businessResult.Valid {

		result.Valid = false

		result.Errors = append(result.Errors, businessResult.Errors...)

		result.Warnings = append(result.Warnings, businessResult.Warnings...)

	}

	return result

}

// ValidateConsumerInfo validates A1-C consumer information.

func (av *A1ValidatorImpl) ValidateConsumerInfo(info *ConsumerInfo) *ValidationResult {

	result := &ValidationResult{Valid: true}

	if info == nil {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Field: "consumer_info",

			Message: "Consumer info cannot be nil",
		})

		return result

	}

	// Structural validation.

	if err := av.validator.Struct(info); err != nil {

		result.Valid = false

		result.Errors = append(result.Errors, av.convertValidationErrors(err)...)

	}

	// Callback URL validation.

	if callbackResult := av.validateCallbackURL(info.CallbackURL); !callbackResult.Valid {

		result.Valid = false

		result.Errors = append(result.Errors, callbackResult.Errors...)

	}

	// Business logic validation.

	if businessResult := av.validateConsumerBusinessRules(info); !businessResult.Valid {

		result.Valid = false

		result.Errors = append(result.Errors, businessResult.Errors...)

		result.Warnings = append(result.Warnings, businessResult.Warnings...)

	}

	return result

}

// ValidateEnrichmentInfoType validates A1-EI type information.

func (av *A1ValidatorImpl) ValidateEnrichmentInfoType(eiType *EnrichmentInfoType) *ValidationResult {

	result := &ValidationResult{Valid: true}

	if eiType == nil {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Field: "ei_type",

			Message: "EI type cannot be nil",
		})

		return result

	}

	// Structural validation.

	if err := av.validator.Struct(eiType); err != nil {

		result.Valid = false

		result.Errors = append(result.Errors, av.convertValidationErrors(err)...)

	}

	// Schema validation for EI job data schema.

	if av.config.EnableSchemaValidation && eiType.EiJobDataSchema != nil {

		if schemaResult := av.validateJSONSchemaStructure(eiType.EiJobDataSchema, "ei_job_data_schema"); !schemaResult.Valid {

			result.Valid = false

			result.Errors = append(result.Errors, schemaResult.Errors...)

			result.Warnings = append(result.Warnings, schemaResult.Warnings...)

		}

	}

	// Schema validation for EI job result schema (if provided).

	if av.config.EnableSchemaValidation && eiType.EiJobResultSchema != nil {

		if schemaResult := av.validateJSONSchemaStructure(eiType.EiJobResultSchema, "ei_job_result_schema"); !schemaResult.Valid {

			result.Valid = false

			result.Errors = append(result.Errors, schemaResult.Errors...)

			result.Warnings = append(result.Warnings, schemaResult.Warnings...)

		}

	}

	return result

}

// ValidateEnrichmentInfoJob validates A1-EI job information.

func (av *A1ValidatorImpl) ValidateEnrichmentInfoJob(job *EnrichmentInfoJob) *ValidationResult {

	result := &ValidationResult{Valid: true}

	if job == nil {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Field: "ei_job",

			Message: "EI job cannot be nil",
		})

		return result

	}

	// Structural validation.

	if err := av.validator.Struct(job); err != nil {

		result.Valid = false

		result.Errors = append(result.Errors, av.convertValidationErrors(err)...)

	}

	// Target URI validation.

	if targetResult := av.validateTargetURI(job.TargetURI); !targetResult.Valid {

		result.Valid = false

		result.Errors = append(result.Errors, targetResult.Errors...)

	}

	// EI job data validation against type schema (if available).

	if av.config.EnableSchemaValidation {

		if schema, exists := av.schemaRegistry[fmt.Sprintf("ei_type_%s", job.EiTypeID)]; exists {

			if dataResult := av.validateDataAgainstSchema(job.EiJobData, schema, "ei_job_data"); !dataResult.Valid {

				result.Valid = false

				result.Errors = append(result.Errors, dataResult.Errors...)

				result.Warnings = append(result.Warnings, dataResult.Warnings...)

			}

		} else {

			result.Warnings = append(result.Warnings, ValidationWarning{

				Field: "ei_job_data",

				Message: fmt.Sprintf("Schema for EI type %s not found in registry, data validation skipped", job.EiTypeID),
			})

		}

	}

	// Business logic validation.

	if businessResult := av.validateEIJobBusinessRules(job); !businessResult.Valid {

		result.Valid = false

		result.Errors = append(result.Errors, businessResult.Errors...)

		result.Warnings = append(result.Warnings, businessResult.Warnings...)

	}

	return result

}

// RegisterSchema registers a JSON schema in the validator's schema registry.

func (av *A1ValidatorImpl) RegisterSchema(schemaID string, schema interface{}) error {

	if schemaID == "" {

		return fmt.Errorf("schema ID cannot be empty")

	}

	if schema == nil {

		return fmt.Errorf("schema cannot be nil")

	}

	// Validate the schema structure itself.

	if result := av.validateJSONSchemaStructure(schema, schemaID); !result.Valid {

		return fmt.Errorf("invalid schema structure: %v", result.Errors)

	}

	av.schemaRegistry[schemaID] = schema

	av.logger.InfoWithContext("Schema registered", "schema_id", schemaID)

	return nil

}

// GetRegisteredSchema retrieves a schema from the registry.

func (av *A1ValidatorImpl) GetRegisteredSchema(schemaID string) (interface{}, bool) {

	schema, exists := av.schemaRegistry[schemaID]

	return schema, exists

}

// ListRegisteredSchemas returns all registered schema IDs.

func (av *A1ValidatorImpl) ListRegisteredSchemas() []string {

	var schemaIDs []string

	for id := range av.schemaRegistry {

		schemaIDs = append(schemaIDs, id)

	}

	return schemaIDs

}

// Private validation methods.

// convertValidationErrors converts validator.ValidationErrors to ValidationError slice.

func (av *A1ValidatorImpl) convertValidationErrors(err error) []ValidationError {

	var validationErrors []ValidationError

	var validatorErrs validator.ValidationErrors

	if errors.As(err, &validatorErrs) {

		for _, err := range validatorErrs {

			validationErrors = append(validationErrors, ValidationError{

				Field: err.Field(),

				Value: err.Value(),

				Tag: err.Tag(),

				Message: av.getValidationMessage(err),

				Param: err.Param(),

				StructName: err.StructNamespace(),

				FieldPath: strings.ToLower(err.StructNamespace()),
			})

		}

	}

	return validationErrors

}

// getValidationMessage returns a human-readable message for validation errors.

func (av *A1ValidatorImpl) getValidationMessage(err validator.FieldError) string {

	switch err.Tag() {

	case "required":

		return fmt.Sprintf("Field '%s' is required", err.Field())

	case "min":

		return fmt.Sprintf("Field '%s' must be at least %s", err.Field(), err.Param())

	case "max":

		return fmt.Sprintf("Field '%s' must be at most %s", err.Field(), err.Param())

	case "email":

		return fmt.Sprintf("Field '%s' must be a valid email address", err.Field())

	case "url":

		return fmt.Sprintf("Field '%s' must be a valid URL", err.Field())

	case "oneof":

		return fmt.Sprintf("Field '%s' must be one of: %s", err.Field(), err.Param())

	case "oran_policy_type_id":

		return fmt.Sprintf("Field '%s' must be a valid O-RAN policy type ID", err.Field())

	case "oran_policy_id":

		return fmt.Sprintf("Field '%s' must be a valid O-RAN policy ID", err.Field())

	case "oran_ei_type_id":

		return fmt.Sprintf("Field '%s' must be a valid O-RAN EI type ID", err.Field())

	case "oran_ei_job_id":

		return fmt.Sprintf("Field '%s' must be a valid O-RAN EI job ID", err.Field())

	case "oran_consumer_id":

		return fmt.Sprintf("Field '%s' must be a valid O-RAN consumer ID", err.Field())

	case "json_schema":

		return fmt.Sprintf("Field '%s' must be a valid JSON schema", err.Field())

	case "uri_reference":

		return fmt.Sprintf("Field '%s' must be a valid URI reference", err.Field())

	case "iso8601":

		return fmt.Sprintf("Field '%s' must be a valid ISO8601 datetime", err.Field())

	default:

		return fmt.Sprintf("Field '%s' validation failed: %s", err.Field(), err.Tag())

	}

}

// validatePolicyTypeBusinessRules validates business-specific rules for policy types.

func (av *A1ValidatorImpl) validatePolicyTypeBusinessRules(policyType *PolicyType) *ValidationResult {

	result := &ValidationResult{Valid: true}

	// Policy type ID range validation (O-RAN recommends 1-100000).

	if policyType.PolicyTypeID < 1 || policyType.PolicyTypeID > 100000 {

		result.Warnings = append(result.Warnings, ValidationWarning{

			Field: "policy_type_id",

			Value: policyType.PolicyTypeID,

			Message: "Policy type ID outside recommended range (1-100000)",
		})

	}

	// Schema complexity validation.

	if policyType.Schema != nil {

		if complexity := av.calculateSchemaComplexity(policyType.Schema); complexity > 100 {

			result.Warnings = append(result.Warnings, ValidationWarning{

				Field: "schema",

				Message: fmt.Sprintf("Schema complexity (%d) is high, may impact performance", complexity),
			})

		}

	}

	return result

}

// validatePolicyInstanceBusinessRules validates business-specific rules for policy instances.

func (av *A1ValidatorImpl) validatePolicyInstanceBusinessRules(instance *PolicyInstance) *ValidationResult {

	result := &ValidationResult{Valid: true}

	// Policy ID format validation (should be alphanumeric with hyphens/underscores).

	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(instance.PolicyID) {

		result.Errors = append(result.Errors, ValidationError{

			Field: "policy_id",

			Value: instance.PolicyID,

			Message: "Policy ID must contain only alphanumeric characters, hyphens, and underscores",
		})

		result.Valid = false

	}

	// Policy data size validation.

	if data, err := json.Marshal(instance.PolicyData); err == nil {

		if len(data) > 1024*1024 { // 1MB limit

			result.Warnings = append(result.Warnings, ValidationWarning{

				Field: "policy_data",

				Message: "Policy data size exceeds 1MB, may impact performance",
			})

		}

	}

	return result

}

// validateConsumerBusinessRules validates business-specific rules for consumers.

func (av *A1ValidatorImpl) validateConsumerBusinessRules(info *ConsumerInfo) *ValidationResult {

	result := &ValidationResult{Valid: true}

	// Consumer ID format validation.

	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(info.ConsumerID) {

		result.Errors = append(result.Errors, ValidationError{

			Field: "consumer_id",

			Value: info.ConsumerID,

			Message: "Consumer ID must contain only alphanumeric characters, hyphens, and underscores",
		})

		result.Valid = false

	}

	// Capabilities validation.

	validCapabilities := []string{"policy_notification", "enrichment_info", "real_time_control"}

	for _, capability := range info.Capabilities {

		found := false

		for _, valid := range validCapabilities {

			if capability == valid {

				found = true

				break

			}

		}

		if !found {

			result.Warnings = append(result.Warnings, ValidationWarning{

				Field: "capabilities",

				Value: capability,

				Message: fmt.Sprintf("Unknown capability '%s', supported: %v", capability, validCapabilities),
			})

		}

	}

	return result

}

// validateEIJobBusinessRules validates business-specific rules for EI jobs.

func (av *A1ValidatorImpl) validateEIJobBusinessRules(job *EnrichmentInfoJob) *ValidationResult {

	result := &ValidationResult{Valid: true}

	// EI job ID format validation.

	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(job.EiJobID) {

		result.Errors = append(result.Errors, ValidationError{

			Field: "ei_job_id",

			Value: job.EiJobID,

			Message: "EI job ID must contain only alphanumeric characters, hyphens, and underscores",
		})

		result.Valid = false

	}

	// Job owner validation.

	if job.JobOwner == "" {

		result.Errors = append(result.Errors, ValidationError{

			Field: "job_owner",

			Message: "Job owner is required",
		})

		result.Valid = false

	}

	return result

}

// validateJSONSchemaStructure validates that a given object is a valid JSON schema.

func (av *A1ValidatorImpl) validateJSONSchemaStructure(schema interface{}, fieldPath string) *ValidationResult {

	result := &ValidationResult{Valid: true}

	// Convert to map for inspection.

	schemaMap, ok := schema.(map[string]interface{})

	if !ok {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Field: fieldPath,

			Message: "Schema must be a JSON object",

			FieldPath: fieldPath,
		})

		return result

	}

	// Validate required schema properties.

	if _, exists := schemaMap["type"]; !exists {

		result.Warnings = append(result.Warnings, ValidationWarning{

			Field: fieldPath,

			Message: "Schema should specify a 'type' property",

			FieldPath: fieldPath,
		})

	}

	// Validate schema version if present.

	if schemaVersion, exists := schemaMap["$schema"]; exists {

		if versionStr, ok := schemaVersion.(string); ok {

			supportedVersions := []string{

				"http://json-schema.org/draft-07/schema#",

				"https://json-schema.org/draft/2019-09/schema",

				"https://json-schema.org/draft/2020-12/schema",
			}

			supported := false

			for _, version := range supportedVersions {

				if versionStr == version {

					supported = true

					break

				}

			}

			if !supported {

				result.Warnings = append(result.Warnings, ValidationWarning{

					Field: fieldPath,

					Value: versionStr,

					Message: fmt.Sprintf("Schema version may not be fully supported, recommended: %v", supportedVersions),

					FieldPath: fieldPath,
				})

			}

		}

	}

	return result

}

// validateDataAgainstSchema validates data against a JSON schema.

func (av *A1ValidatorImpl) validateDataAgainstSchema(data, schema interface{}, fieldPath string) *ValidationResult {

	result := &ValidationResult{Valid: true}

	// This is a simplified schema validation implementation.

	// In a production environment, you would use a proper JSON schema library.

	// like github.com/xeipuuv/gojsonschema.

	schemaMap, ok := schema.(map[string]interface{})

	if !ok {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Field: fieldPath,

			Message: "Invalid schema format",

			FieldPath: fieldPath,
		})

		return result

	}

	// Validate required fields.

	if required, exists := schemaMap["required"]; exists {

		if requiredList, ok := required.([]interface{}); ok {

			if dataMap, ok := data.(map[string]interface{}); ok {

				for _, req := range requiredList {

					if reqField, ok := req.(string); ok {

						if _, exists := dataMap[reqField]; !exists {

							result.Valid = false

							result.Errors = append(result.Errors, ValidationError{

								Field: reqField,

								Message: fmt.Sprintf("Required field '%s' is missing", reqField),

								FieldPath: fmt.Sprintf("%s.%s", fieldPath, reqField),
							})

						}

					}

				}

			}

		}

	}

	// Validate type.

	if schemaType, exists := schemaMap["type"]; exists {

		if typeStr, ok := schemaType.(string); ok {

			if !av.validateDataType(data, typeStr) {

				result.Valid = false

				result.Errors = append(result.Errors, ValidationError{

					Field: fieldPath,

					Value: data,

					Message: fmt.Sprintf("Data type does not match schema type '%s'", typeStr),

					FieldPath: fieldPath,
				})

			}

		}

	}

	return result

}

// validateDataType checks if data matches the expected JSON schema type.

func (av *A1ValidatorImpl) validateDataType(data interface{}, expectedType string) bool {

	switch expectedType {

	case "string":

		_, ok := data.(string)

		return ok

	case "number":

		_, ok1 := data.(float64)

		_, ok2 := data.(int)

		_, ok3 := data.(int64)

		return ok1 || ok2 || ok3

	case "integer":

		_, ok1 := data.(int)

		_, ok2 := data.(int64)

		if f, ok := data.(float64); ok {

			return f == float64(int64(f))

		}

		return ok1 || ok2

	case "boolean":

		_, ok := data.(bool)

		return ok

	case "array":

		_, ok := data.([]interface{})

		return ok

	case "object":

		_, ok := data.(map[string]interface{})

		return ok

	case "null":

		return data == nil

	default:

		return false

	}

}

// calculateSchemaComplexity calculates a complexity score for a JSON schema.

func (av *A1ValidatorImpl) calculateSchemaComplexity(schema interface{}) int {

	complexity := 0

	if schemaMap, ok := schema.(map[string]interface{}); ok {

		complexity += len(schemaMap) // Base complexity from number of properties

		// Add complexity for nested schemas.

		if properties, exists := schemaMap["properties"]; exists {

			if propsMap, ok := properties.(map[string]interface{}); ok {

				complexity += len(propsMap)

				// Recursively calculate nested complexity.

				for _, prop := range propsMap {

					if propMap, ok := prop.(map[string]interface{}); ok {

						complexity += av.calculateSchemaComplexity(propMap)

					}

				}

			}

		}

		// Add complexity for arrays.

		if items, exists := schemaMap["items"]; exists {

			complexity += av.calculateSchemaComplexity(items)

		}

		// Add complexity for conditional schemas.

		if allOf, exists := schemaMap["allOf"]; exists {

			if allOfList, ok := allOf.([]interface{}); ok {

				complexity += len(allOfList)

			}

		}

		if anyOf, exists := schemaMap["anyOf"]; exists {

			if anyOfList, ok := anyOf.([]interface{}); ok {

				complexity += len(anyOfList)

			}

		}

		if oneOf, exists := schemaMap["oneOf"]; exists {

			if oneOfList, ok := oneOf.([]interface{}); ok {

				complexity += len(oneOfList)

			}

		}

	}

	return complexity

}

// validateCallbackURL validates a callback URL for consumers.

func (av *A1ValidatorImpl) validateCallbackURL(callbackURL string) *ValidationResult {

	result := &ValidationResult{Valid: true}

	// Parse URL.

	u, err := url.Parse(callbackURL)

	if err != nil {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Field: "callback_url",

			Value: callbackURL,

			Message: fmt.Sprintf("Invalid URL format: %v", err),
		})

		return result

	}

	// Validate scheme.

	if u.Scheme != "http" && u.Scheme != "https" {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Field: "callback_url",

			Value: callbackURL,

			Message: "URL scheme must be http or https",
		})

	}

	// Validate host.

	if u.Host == "" {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Field: "callback_url",

			Value: callbackURL,

			Message: "URL must have a valid host",
		})

	}

	// Security warning for HTTP.

	if u.Scheme == "http" {

		result.Warnings = append(result.Warnings, ValidationWarning{

			Field: "callback_url",

			Value: callbackURL,

			Message: "Using HTTP instead of HTTPS may pose security risks",
		})

	}

	return result

}

// validateTargetURI validates a target URI for EI jobs.

func (av *A1ValidatorImpl) validateTargetURI(targetURI string) *ValidationResult {

	result := &ValidationResult{Valid: true}

	// Parse URI.

	u, err := url.Parse(targetURI)

	if err != nil {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Field: "target_uri",

			Value: targetURI,

			Message: fmt.Sprintf("Invalid URI format: %v", err),
		})

		return result

	}

	// Validate scheme.

	supportedSchemes := []string{"http", "https", "kafka", "amqp", "mqtt"}

	schemeValid := false

	for _, scheme := range supportedSchemes {

		if u.Scheme == scheme {

			schemeValid = true

			break

		}

	}

	if !schemeValid {

		result.Valid = false

		result.Errors = append(result.Errors, ValidationError{

			Field: "target_uri",

			Value: targetURI,

			Message: fmt.Sprintf("URI scheme must be one of: %v", supportedSchemes),
		})

	}

	return result

}

// Custom validation functions.

// validatePolicyTypeID validates O-RAN policy type ID format.

func validatePolicyTypeID(fl validator.FieldLevel) bool {

	value := fl.Field().Int()

	return value >= 1 && value <= 100000

}

// validatePolicyID validates O-RAN policy ID format.

func validatePolicyID(fl validator.FieldLevel) bool {

	value := fl.Field().String()

	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, value)

	return matched && len(value) <= 64

}

// validateEITypeID validates O-RAN EI type ID format.

func validateEITypeID(fl validator.FieldLevel) bool {

	value := fl.Field().String()

	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, value)

	return matched && len(value) <= 64

}

// validateEIJobID validates O-RAN EI job ID format.

func validateEIJobID(fl validator.FieldLevel) bool {

	value := fl.Field().String()

	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, value)

	return matched && len(value) <= 64

}

// validateConsumerID validates O-RAN consumer ID format.

func validateConsumerID(fl validator.FieldLevel) bool {

	value := fl.Field().String()

	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, value)

	return matched && len(value) <= 64

}

// validateJSONSchema validates that a field contains a valid JSON schema.

func validateJSONSchema(fl validator.FieldLevel) bool {

	// This is a basic validation - in production use a proper JSON schema validator.

	value := fl.Field().Interface()

	if schemaMap, ok := value.(map[string]interface{}); ok {

		// Check for basic schema structure.

		if _, hasType := schemaMap["type"]; hasType {

			return true

		}

		if _, hasProperties := schemaMap["properties"]; hasProperties {

			return true

		}

		if _, hasOneOf := schemaMap["oneOf"]; hasOneOf {

			return true

		}

		if _, hasAnyOf := schemaMap["anyOf"]; hasAnyOf {

			return true

		}

		if _, hasAllOf := schemaMap["allOf"]; hasAllOf {

			return true

		}

	}

	return false

}

// validateURIReference validates URI reference format.

func validateURIReference(fl validator.FieldLevel) bool {

	value := fl.Field().String()

	_, err := url.Parse(value)

	return err == nil

}

// validateISO8601DateTime validates ISO8601 datetime format.

func validateISO8601DateTime(fl validator.FieldLevel) bool {

	value := fl.Field().String()

	formats := []string{

		time.RFC3339,

		time.RFC3339Nano,

		"2006-01-02T15:04:05Z",

		"2006-01-02T15:04:05.000Z",
	}

	for _, format := range formats {

		if _, err := time.Parse(format, value); err == nil {

			return true

		}

	}

	return false

}

// ValidateWithContext validates any A1 entity with request context.

func (av *A1ValidatorImpl) ValidateWithContext(ctx context.Context, entity interface{}) *ValidationResult {

	switch e := entity.(type) {

	case *PolicyType:

		return av.ValidatePolicyType(e)

	case *PolicyInstance:

		// Extract policy type ID from context if available.

		if policyTypeID, ok := ctx.Value("policy_type_id").(int); ok {

			return av.ValidatePolicyInstance(policyTypeID, e)

		}

		return av.ValidatePolicyInstance(e.PolicyTypeID, e)

	case *ConsumerInfo:

		return av.ValidateConsumerInfo(e)

	case *EnrichmentInfoType:

		return av.ValidateEnrichmentInfoType(e)

	case *EnrichmentInfoJob:

		return av.ValidateEnrichmentInfoJob(e)

	default:

		return &ValidationResult{

			Valid: false,

			Errors: []ValidationError{{

				Field: "entity",

				Message: fmt.Sprintf("Unsupported entity type: %T", entity),
			}},
		}

	}

}

// GetValidationSummary returns a summary of validation results.

func (vr *ValidationResult) GetValidationSummary() string {

	if vr.Valid {

		if len(vr.Warnings) > 0 {

			return fmt.Sprintf("Valid with %d warnings", len(vr.Warnings))

		}

		return "Valid"

	}

	return fmt.Sprintf("Invalid (%d errors, %d warnings)", len(vr.Errors), len(vr.Warnings))

}
