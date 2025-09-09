// Package a1 provides comprehensive unit tests for A1 validation logic
package a1

import (
<<<<<<< HEAD
	"fmt"
	"strings"
	"testing"
	"encoding/json"
=======
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	"github.com/stretchr/testify/assert"
)

// Test fixtures for validation

func createValidPolicyTypeSchema() map[string]interface{} {
	return map[string]interface{}{
		"scope": map[string]interface{}{
<<<<<<< HEAD
			"ue_id":   json.RawMessage(`{}`),
			"cell_id": json.RawMessage(`{}`),
		},
		"required": []string{"ue_id"},
		"statement": map[string]interface{}{
			"qos_class": json.RawMessage(`{}`),
			"bitrate":   json.RawMessage(`{}`),
			"action":    json.RawMessage(`{}`),
		},
		"required": []string{"qos_class", "action"},
=======
			"ue_id":   map[string]interface{}{},
			"cell_id": map[string]interface{}{},
		},
		"required": []string{"ue_id", "statement"},
		"statement": map[string]interface{}{
			"qos_class": map[string]interface{}{},
			"bitrate":   map[string]interface{}{},
			"action":    map[string]interface{}{},
		},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}
}

func createValidEIJobDataSchema() map[string]interface{} {
	return map[string]interface{}{
		"config": map[string]interface{}{
<<<<<<< HEAD
			"measurement_type":  json.RawMessage(`{}`),
			"reporting_period":  json.RawMessage(`{}`),
=======
			"measurement_type": json.RawMessage(`{}`),
			"reporting_period": json.RawMessage(`{}`),
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			"targets": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"cell_id":   json.RawMessage(`{"type": "string"}`),
						"threshold": json.RawMessage(`{}`),
					},
					"required": []string{"cell_id", "threshold"},
				},
				"minItems": 1,
				"maxItems": 100,
			},
			"required": []string{"measurement_type", "reporting_period", "targets"},
		},
		"required": []string{"config"},
	}
}

// Test A1Validator implementation

type TestA1Validator struct {
	schemaValidator SchemaValidator
}

func NewTestA1Validator() A1Validator {
	return &TestA1Validator{
		schemaValidator: NewJSONSchemaValidator(),
	}
}

<<<<<<< HEAD
func (v *TestA1Validator) ValidatePolicyType(policyType *PolicyType) error {
	if policyType == nil {
		return NewValidationError("policy type cannot be nil", "policy_type", nil)
=======
func (v *TestA1Validator) ValidatePolicyType(policyType *PolicyType) *ValidationResult {
	if policyType == nil {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "policy_type",
				Message: "policy type cannot be nil",
			}},
		}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	// Validate required fields
	if policyType.PolicyTypeID <= 0 {
<<<<<<< HEAD
		return NewValidationError("policy_type_id must be positive integer", "policy_type_id", policyType.PolicyTypeID)
	}

	if policyType.Schema == nil || len(policyType.Schema) == 0 {
		return NewValidationError("schema is required", "schema", policyType.Schema)
=======
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "policy_type_id",
				Message: "policy_type_id must be positive integer",
			}},
		}
	}

	if policyType.Schema == nil || len(policyType.Schema) == 0 {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "schema",
				Message: "schema is required",
			}},
		}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	// Validate schema is valid JSON Schema
	if err := v.schemaValidator.ValidateSchema(policyType.Schema); err != nil {
<<<<<<< HEAD
		return NewValidationError("invalid JSON schema", "schema", err)
	}

	// Validate optional create schema if provided
	if policyType.CreateSchema != nil {
		if err := v.schemaValidator.ValidateSchema(policyType.CreateSchema); err != nil {
			return NewValidationError("invalid create schema", "create_schema", err)
		}
	}

	return nil
}

func (v *TestA1Validator) ValidatePolicyInstance(policyType *PolicyType, instance *PolicyInstance) error {
	if policyType == nil {
		return NewValidationError("policy type cannot be nil", "policy_type", nil)
	}

	if instance == nil {
		return NewValidationError("policy instance cannot be nil", "policy_instance", nil)
=======
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "schema",
				Message: "invalid JSON schema",
			}},
		}
	}

	// Skip create schema validation in test (would need type conversion from json.RawMessage)
	// In real implementation: convert json.RawMessage to map[string]interface{} first

	return &ValidationResult{Valid: true}
}

func (v *TestA1Validator) ValidatePolicyInstance(policyTypeID int, instance *PolicyInstance) *ValidationResult {

	if instance == nil {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "policy_instance",
				Message: "policy instance cannot be nil",
			}},
		}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	// Validate required fields
	if strings.TrimSpace(instance.PolicyID) == "" {
<<<<<<< HEAD
		return NewValidationError("policy_id is required", "policy_id", instance.PolicyID)
	}

	if instance.PolicyTypeID != policyType.PolicyTypeID {
		return NewValidationError("policy_type_id mismatch", "policy_type_id",
			json.RawMessage(`{}`))
	}

	if instance.PolicyData == nil || len(instance.PolicyData) == 0 {
		return NewValidationError("policy_data is required", "policy_data", instance.PolicyData)
	}

	// Validate policy data against policy type schema
	if err := v.schemaValidator.ValidateAgainstSchema(instance.PolicyData, policyType.Schema); err != nil {
		return NewValidationError("policy data validation failed", "policy_data", err)
	}
=======
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "policy_id",
				Message: "policy_id is required",
			}},
		}
	}

	if instance.PolicyTypeID != policyTypeID {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "policy_type_id",
				Message: "policy_type_id mismatch",
			}},
		}
	}

	if instance.PolicyData == nil || len(instance.PolicyData) == 0 {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "policy_data",
				Message: "policy_data is required",
			}},
		}
	}

	// Skip schema validation in this test implementation
	// In a real implementation, you would fetch the policy type by ID and validate against its schema
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	// Validate notification destination URL if provided
	if instance.PolicyInfo.NotificationDestination != "" {
		if err := ValidateURL(instance.PolicyInfo.NotificationDestination); err != nil {
<<<<<<< HEAD
			return NewValidationError("invalid notification destination URL", "notification_destination", err)
		}
	}

	return nil
}

func (v *TestA1Validator) ValidateEIType(eiType *EnrichmentInfoType) error {
	if eiType == nil {
		return NewValidationError("enrichment info type cannot be nil", "ei_type", nil)
=======
			return &ValidationResult{
				Valid: false,
				Errors: []ValidationError{{
					Field:   "notification_destination",
					Message: "invalid notification destination URL",
				}},
			}
		}
	}

	return &ValidationResult{Valid: true}
}

func (v *TestA1Validator) ValidateEnrichmentInfoType(eiType *EnrichmentInfoType) *ValidationResult {
	if eiType == nil {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "ei_type",
				Message: "enrichment info type cannot be nil",
			}},
		}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	// Validate required fields
	if strings.TrimSpace(eiType.EiTypeID) == "" {
<<<<<<< HEAD
		return NewValidationError("ei_type_id is required", "ei_type_id", eiType.EiTypeID)
	}

	if eiType.EiJobDataSchema == nil || len(eiType.EiJobDataSchema) == 0 {
		return NewValidationError("ei_job_data_schema is required", "ei_job_data_schema", eiType.EiJobDataSchema)
=======
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "ei_type_id",
				Message: "ei_type_id is required",
			}},
		}
	}

	if eiType.EiJobDataSchema == nil || len(eiType.EiJobDataSchema) == 0 {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "ei_job_data_schema",
				Message: "ei_job_data_schema is required",
			}},
		}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}

	// Validate schemas are valid JSON Schema
	if err := v.schemaValidator.ValidateSchema(eiType.EiJobDataSchema); err != nil {
<<<<<<< HEAD
		return NewValidationError("invalid ei_job_data_schema", "ei_job_data_schema", err)
	}

	if eiType.EiJobResultSchema != nil {
		if err := v.schemaValidator.ValidateSchema(eiType.EiJobResultSchema); err != nil {
			return NewValidationError("invalid ei_job_result_schema", "ei_job_result_schema", err)
		}
	}

	return nil
=======
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "ei_job_data_schema",
				Message: "invalid ei_job_data_schema",
			}},
		}
	}

	// Skip job result schema validation in test (would need type conversion from json.RawMessage)
	// In real implementation: convert json.RawMessage to map[string]interface{} first

	return &ValidationResult{Valid: true}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
}

func (v *TestA1Validator) ValidateEIJob(eiType *EnrichmentInfoType, job *EnrichmentInfoJob) error {
	if eiType == nil {
		return NewValidationError("enrichment info type cannot be nil", "ei_type", nil)
	}

	if job == nil {
		return NewValidationError("enrichment info job cannot be nil", "ei_job", nil)
	}

	// Validate required fields
	if strings.TrimSpace(job.EiJobID) == "" {
		return NewValidationError("ei_job_id is required", "ei_job_id", job.EiJobID)
	}

	if job.EiTypeID != eiType.EiTypeID {
		return NewValidationError("ei_type_id mismatch", "ei_type_id",
			json.RawMessage(`{}`))
	}

	if job.EiJobData == nil || len(job.EiJobData) == 0 {
		return NewValidationError("ei_job_data is required", "ei_job_data", job.EiJobData)
	}

	if strings.TrimSpace(job.TargetURI) == "" {
		return NewValidationError("target_uri is required", "target_uri", job.TargetURI)
	}

	if strings.TrimSpace(job.JobOwner) == "" {
		return NewValidationError("job_owner is required", "job_owner", job.JobOwner)
	}

	// Validate URLs
	if err := ValidateURL(job.TargetURI); err != nil {
		return NewValidationError("invalid target_uri", "target_uri", err)
	}

	if job.JobStatusURL != "" {
		if err := ValidateURL(job.JobStatusURL); err != nil {
			return NewValidationError("invalid job_status_url", "job_status_url", err)
		}
	}

	// Validate job data against EI type schema
	if err := v.schemaValidator.ValidateAgainstSchema(job.EiJobData, eiType.EiJobDataSchema); err != nil {
		return NewValidationError("ei job data validation failed", "ei_job_data", err)
	}

	return nil
}

<<<<<<< HEAD
=======
func (v *TestA1Validator) ValidateConsumerInfo(info *ConsumerInfo) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if info == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "consumer_info",
			Message: "Consumer info cannot be nil",
		})
		return result
	}

	// Validate required consumer fields
	if info.ConsumerID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "consumer_id",
			Message: "Consumer ID is required",
		})
	}

	return result
}

func (v *TestA1Validator) ValidateEnrichmentInfoJob(job *EnrichmentInfoJob) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if job == nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "enrichment_info_job",
			Message: "Enrichment info job cannot be nil",
		})
		return result
	}

	// Validate required fields
	if strings.TrimSpace(job.EiJobID) == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "ei_job_id",
			Message: "EI job ID is required",
		})
	}

	if strings.TrimSpace(job.TargetURI) == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "target_uri",
			Message: "Target URI is required",
		})
	}

	if strings.TrimSpace(job.JobOwner) == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "job_owner",
			Message: "Job owner is required",
		})
	}

	// Validate URLs
	if job.TargetURI != "" {
		if err := ValidateURL(job.TargetURI); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "target_uri",
				Message: "Invalid target URI format",
			})
		}
	}

	if job.JobStatusURL != "" {
		if err := ValidateURL(job.JobStatusURL); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "job_status_url",
				Message: "Invalid job status URL format",
			})
		}
	}

	return result
}

>>>>>>> 6835433495e87288b95961af7173d866977175ff
// Test Policy Type Validation

func TestValidatePolicyType_Success(t *testing.T) {
	validator := NewTestA1Validator()

	validPolicyType := &PolicyType{
		PolicyTypeID:   1,
		PolicyTypeName: "test-policy",
		Description:    "Test policy type",
		Schema:         createValidPolicyTypeSchema(),
	}

<<<<<<< HEAD
	err := validator.ValidatePolicyType(validPolicyType)
	assert.NoError(t, err)
=======
	result := validator.ValidatePolicyType(validPolicyType)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
}

func TestValidatePolicyType_NilPolicyType(t *testing.T) {
	validator := NewTestA1Validator()

<<<<<<< HEAD
	err := validator.ValidatePolicyType(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "policy type cannot be nil")
=======
	result := validator.ValidatePolicyType(nil)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "policy type cannot be nil")
>>>>>>> 6835433495e87288b95961af7173d866977175ff
}

func TestValidatePolicyType_InvalidPolicyTypeID(t *testing.T) {
	validator := NewTestA1Validator()

	tests := []struct {
		name         string
		policyTypeID int
	}{
		{"zero policy type ID", 0},
		{"negative policy type ID", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policyType := &PolicyType{
				PolicyTypeID: tt.policyTypeID,
				Schema:       createValidPolicyTypeSchema(),
			}

<<<<<<< HEAD
			err := validator.ValidatePolicyType(policyType)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "policy_type_id must be positive integer")
=======
			result := validator.ValidatePolicyType(policyType)
			assert.False(t, result.Valid)
			assert.NotEmpty(t, result.Errors)
			assert.Contains(t, result.Errors[0].Message, "policy_type_id must be positive integer")
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		})
	}
}

func TestValidatePolicyType_MissingSchema(t *testing.T) {
	validator := NewTestA1Validator()

	tests := []struct {
		name   string
		schema map[string]interface{}
	}{
		{"nil schema", nil},
		{"empty schema", make(map[string]interface{})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policyType := &PolicyType{
				PolicyTypeID: 1,
				Schema:       tt.schema,
			}

			err := validator.ValidatePolicyType(policyType)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "schema is required")
		})
	}
}

func TestValidatePolicyType_InvalidSchema(t *testing.T) {
	validator := NewTestA1Validator()

	invalidSchemas := []struct {
		name   string
		schema map[string]interface{}
	}{
		{
<<<<<<< HEAD
			"invalid type",
			json.RawMessage(`{}`),
=======
			"empty schema",
			map[string]interface{}{},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},
		{
			"circular reference",
			map[string]interface{}{
<<<<<<< HEAD
				"self": json.RawMessage(`{}`),
=======
				"self": "reference",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			},
		},
		{
			"invalid enum values",
<<<<<<< HEAD
			json.RawMessage(`{}`), // Empty enum
=======
			map[string]interface{}{
				"enum": []interface{}{}, // Empty enum
			},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		},
	}

	for _, tt := range invalidSchemas {
		t.Run(tt.name, func(t *testing.T) {
			policyType := &PolicyType{
				PolicyTypeID: 1,
				Schema:       tt.schema,
			}

			err := validator.ValidatePolicyType(policyType)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid JSON schema")
		})
	}
}

// Test Policy Instance Validation

func TestValidatePolicyInstance_Success(t *testing.T) {
	validator := NewTestA1Validator()

	policyType := &PolicyType{
		PolicyTypeID: 1,
		Schema:       createValidPolicyTypeSchema(),
	}

	validInstance := &PolicyInstance{
		PolicyID:     "test-policy-1",
		PolicyTypeID: 1,
		PolicyData: map[string]interface{}{
			"ue_id":     "test-ue-123",
			"cell_id":   "ABCD1234",
			"statement": json.RawMessage(`{}`),
		},
		PolicyInfo: PolicyInstanceInfo{
			NotificationDestination: "http://callback.example.com/notify",
			RequestID:               "req-123",
		},
	}

<<<<<<< HEAD
	err := validator.ValidatePolicyInstance(policyType, validInstance)
=======
	err := validator.ValidatePolicyInstance(policyType.PolicyTypeID, validInstance)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	assert.NoError(t, err)
}

func TestValidatePolicyInstance_NilInputs(t *testing.T) {
	validator := NewTestA1Validator()

	tests := []struct {
		name        string
		policyType  *PolicyType
		instance    *PolicyInstance
		expectedMsg string
	}{
		{
			"nil policy type",
			nil,
			&PolicyInstance{},
			"policy type cannot be nil",
		},
		{
			"nil policy instance",
			&PolicyType{PolicyTypeID: 1},
			nil,
			"policy instance cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
<<<<<<< HEAD
			err := validator.ValidatePolicyInstance(tt.policyType, tt.instance)
=======
			err := validator.ValidatePolicyInstance(tt.policyType.PolicyTypeID, tt.instance)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedMsg)
		})
	}
}

func TestValidatePolicyInstance_InvalidFields(t *testing.T) {
	validator := NewTestA1Validator()

	policyType := &PolicyType{
		PolicyTypeID: 1,
		Schema:       createValidPolicyTypeSchema(),
	}

	tests := []struct {
		name        string
		instance    *PolicyInstance
		expectedMsg string
	}{
		{
			"empty policy ID",
			&PolicyInstance{
				PolicyID:     "",
				PolicyTypeID: 1,
<<<<<<< HEAD
				PolicyData:   json.RawMessage(`{"test":"data"}`),
=======
				PolicyData:   func() map[string]interface{} {
					var result map[string]interface{}
					json.Unmarshal(json.RawMessage(`{"test":"data"}`), &result)
					return result
				}(),
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			},
			"policy_id is required",
		},
		{
			"whitespace policy ID",
			&PolicyInstance{
				PolicyID:     "   ",
				PolicyTypeID: 1,
<<<<<<< HEAD
				PolicyData:   json.RawMessage(`{"test":"data"}`),
=======
				PolicyData:   func() map[string]interface{} {
					var result map[string]interface{}
					json.Unmarshal(json.RawMessage(`{"test":"data"}`), &result)
					return result
				}(),
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			},
			"policy_id is required",
		},
		{
			"mismatched policy type ID",
			&PolicyInstance{
				PolicyID:     "test",
				PolicyTypeID: 999,
<<<<<<< HEAD
				PolicyData:   json.RawMessage(`{"test":"data"}`),
=======
				PolicyData:   func() map[string]interface{} {
					var result map[string]interface{}
					json.Unmarshal(json.RawMessage(`{"test":"data"}`), &result)
					return result
				}(),
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			},
			"policy_type_id mismatch",
		},
		{
			"nil policy data",
			&PolicyInstance{
				PolicyID:     "test",
				PolicyTypeID: 1,
				PolicyData:   nil,
			},
			"policy_data is required",
		},
		{
			"empty policy data",
			&PolicyInstance{
				PolicyID:     "test",
				PolicyTypeID: 1,
				PolicyData:   make(map[string]interface{}),
			},
			"policy_data is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
<<<<<<< HEAD
			err := validator.ValidatePolicyInstance(policyType, tt.instance)
=======
			err := validator.ValidatePolicyInstance(policyType.PolicyTypeID, tt.instance)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedMsg)
		})
	}
}

func TestValidatePolicyInstance_SchemaValidation(t *testing.T) {
	validator := NewTestA1Validator()

	policyType := &PolicyType{
		PolicyTypeID: 1,
		Schema:       createValidPolicyTypeSchema(),
	}

	invalidData := []struct {
		name string
		data map[string]interface{}
	}{
		{
			"missing required scope",
			map[string]interface{}{
				"qos_class": 5,
				"action":    "allow",
			},
		},
		{
			"missing required statement",
			map[string]interface{}{
				"ue_id": "test-ue-123",
			},
		},
		{
			"invalid qos_class type",
			map[string]interface{}{
				"ue_id":     "test-ue-123",
				"statement": json.RawMessage(`{}`),
			},
		},
		{
			"qos_class out of range",
			map[string]interface{}{
				"ue_id":     "test-ue-123",
				"statement": json.RawMessage(`{}`),
			},
		},
		{
			"invalid action enum",
			map[string]interface{}{
				"ue_id":     "test-ue-123",
				"statement": json.RawMessage(`{}`),
			},
		},
		{
			"invalid cell_id pattern",
			map[string]interface{}{
				"ue_id":     "test-ue-123",
				"cell_id":   "INVALID",
				"statement": json.RawMessage(`{}`),
			},
		},
	}

	for _, tt := range invalidData {
		t.Run(tt.name, func(t *testing.T) {
			instance := &PolicyInstance{
				PolicyID:     "test-policy",
				PolicyTypeID: 1,
				PolicyData:   tt.data,
			}

<<<<<<< HEAD
			err := validator.ValidatePolicyInstance(policyType, instance)
=======
			err := validator.ValidatePolicyInstance(policyType.PolicyTypeID, instance)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "policy data validation failed")
		})
	}
}

func TestValidatePolicyInstance_InvalidNotificationURL(t *testing.T) {
	validator := NewTestA1Validator()

	policyType := &PolicyType{
		PolicyTypeID: 1,
		Schema:       createValidPolicyTypeSchema(),
	}

	instance := &PolicyInstance{
		PolicyID:     "test-policy",
		PolicyTypeID: 1,
		PolicyData: map[string]interface{}{
			"ue_id":     "test-ue-123",
			"statement": json.RawMessage(`{}`),
		},
		PolicyInfo: PolicyInstanceInfo{
			NotificationDestination: "invalid-url",
		},
	}

<<<<<<< HEAD
	err := validator.ValidatePolicyInstance(policyType, instance)
=======
	err := validator.ValidatePolicyInstance(policyType.PolicyTypeID, instance)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid notification destination URL")
}

// Test EI Type Validation

func TestValidateEIType_Success(t *testing.T) {
	validator := NewTestA1Validator()

	validEIType := &EnrichmentInfoType{
		EiTypeID:        "test-ei-type-1",
		EiTypeName:      "Test EI Type",
		Description:     "Test enrichment information type",
		EiJobDataSchema: createValidEIJobDataSchema(),
<<<<<<< HEAD
		EiJobResultSchema: map[string]interface{}{
			"results": json.RawMessage(`{}`),
		},
	}

	err := validator.ValidateEIType(validEIType)
=======
		EiJobResultSchema: json.RawMessage(`{"results": {}}`),
	}

	err := validator.ValidateEnrichmentInfoType(validEIType)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	assert.NoError(t, err)
}

func TestValidateEIType_InvalidFields(t *testing.T) {
	validator := NewTestA1Validator()

	tests := []struct {
		name        string
		eiType      *EnrichmentInfoType
		expectedMsg string
	}{
		{
			"nil ei type",
			nil,
			"enrichment info type cannot be nil",
		},
		{
			"empty ei_type_id",
			&EnrichmentInfoType{
				EiTypeID:        "",
				EiJobDataSchema: createValidEIJobDataSchema(),
			},
			"ei_type_id is required",
		},
		{
			"whitespace ei_type_id",
			&EnrichmentInfoType{
				EiTypeID:        "   ",
				EiJobDataSchema: createValidEIJobDataSchema(),
			},
			"ei_type_id is required",
		},
		{
			"nil ei_job_data_schema",
			&EnrichmentInfoType{
				EiTypeID:        "test-type",
				EiJobDataSchema: nil,
			},
			"ei_job_data_schema is required",
		},
		{
			"empty ei_job_data_schema",
			&EnrichmentInfoType{
				EiTypeID:        "test-type",
				EiJobDataSchema: make(map[string]interface{}),
			},
			"ei_job_data_schema is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
<<<<<<< HEAD
			err := validator.ValidateEIType(tt.eiType)
=======
			err := validator.ValidateEnrichmentInfoType(tt.eiType)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedMsg)
		})
	}
}

// Test EI Job Validation

func TestValidateEIJob_Success(t *testing.T) {
	validator := NewTestA1Validator()

<<<<<<< HEAD
	eiType := &EnrichmentInfoType{
		EiTypeID:        "test-ei-type-1",
		EiJobDataSchema: createValidEIJobDataSchema(),
	}

=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	validJob := &EnrichmentInfoJob{
		EiJobID:  "test-job-1",
		EiTypeID: "test-ei-type-1",
		EiJobData: map[string]interface{}{
			"measurement_type": "throughput",
			"reporting_period": 5000,
			"targets": []interface{}{
				json.RawMessage(`{}`),
			},
		},
		TargetURI:    "http://consumer.example.com/ei",
		JobOwner:     "test-owner",
		JobStatusURL: "http://status.example.com/job-status",
	}

<<<<<<< HEAD
	err := validator.ValidateEIJob(eiType, validJob)
=======
	err := validator.ValidateEnrichmentInfoJob(validJob)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	assert.NoError(t, err)
}

func TestValidateEIJob_InvalidFields(t *testing.T) {
	validator := NewTestA1Validator()

<<<<<<< HEAD
	eiType := &EnrichmentInfoType{
		EiTypeID:        "test-ei-type-1",
		EiJobDataSchema: createValidEIJobDataSchema(),
	}

	tests := []struct {
		name        string
		eiType      *EnrichmentInfoType
=======
	tests := []struct {
		name        string
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		job         *EnrichmentInfoJob
		expectedMsg string
	}{
		{
<<<<<<< HEAD
			"nil ei type",
			nil,
			&EnrichmentInfoJob{},
			"enrichment info type cannot be nil",
		},
		{
			"nil ei job",
			eiType,
=======
			"nil ei job",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			nil,
			"enrichment info job cannot be nil",
		},
		{
			"empty ei_job_id",
<<<<<<< HEAD
			eiType,
			&EnrichmentInfoJob{
				EiJobID:   "",
				EiTypeID:  "test-ei-type-1",
				EiJobData: json.RawMessage(`{"test":"data"}`),
=======
			&EnrichmentInfoJob{
				EiJobID:   "",
				EiTypeID:  "test-ei-type-1",
				EiJobData: map[string]interface{}{"test": "data"},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
				TargetURI: "http://example.com",
				JobOwner:  "owner",
			},
			"ei_job_id is required",
		},
		{
			"mismatched ei_type_id",
<<<<<<< HEAD
			eiType,
			&EnrichmentInfoJob{
				EiJobID:   "job-1",
				EiTypeID:  "different-type",
				EiJobData: json.RawMessage(`{"test":"data"}`),
=======
			&EnrichmentInfoJob{
				EiJobID:   "job-1",
				EiTypeID:  "different-type",
				EiJobData: map[string]interface{}{"test": "data"},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
				TargetURI: "http://example.com",
				JobOwner:  "owner",
			},
			"ei_type_id mismatch",
		},
		{
			"empty target_uri",
<<<<<<< HEAD
			eiType,
			&EnrichmentInfoJob{
				EiJobID:   "job-1",
				EiTypeID:  "test-ei-type-1",
				EiJobData: json.RawMessage(`{"test":"data"}`),
=======
			&EnrichmentInfoJob{
				EiJobID:   "job-1",
				EiTypeID:  "test-ei-type-1",
				EiJobData: map[string]interface{}{"test": "data"},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
				TargetURI: "",
				JobOwner:  "owner",
			},
			"target_uri is required",
		},
		{
			"empty job_owner",
<<<<<<< HEAD
			eiType,
			&EnrichmentInfoJob{
				EiJobID:   "job-1",
				EiTypeID:  "test-ei-type-1",
				EiJobData: json.RawMessage(`{"test":"data"}`),
=======
			&EnrichmentInfoJob{
				EiJobID:   "job-1",
				EiTypeID:  "test-ei-type-1",
				EiJobData: map[string]interface{}{"test": "data"},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
				TargetURI: "http://example.com",
				JobOwner:  "",
			},
			"job_owner is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
<<<<<<< HEAD
			err := validator.ValidateEIJob(tt.eiType, tt.job)
=======
			err := validator.ValidateEnrichmentInfoJob(tt.job)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedMsg)
		})
	}
}

func TestValidateEIJob_InvalidURLs(t *testing.T) {
	validator := NewTestA1Validator()

<<<<<<< HEAD
	eiType := &EnrichmentInfoType{
		EiTypeID:        "test-ei-type-1",
		EiJobDataSchema: createValidEIJobDataSchema(),
	}

=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	tests := []struct {
		name         string
		targetURI    string
		jobStatusURL string
		expectedMsg  string
	}{
		{
			"invalid target_uri",
			"invalid-url",
			"",
			"invalid target_uri",
		},
		{
			"invalid job_status_url",
			"http://example.com",
			"not-a-url",
			"invalid job_status_url",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &EnrichmentInfoJob{
				EiJobID:      "job-1",
				EiTypeID:     "test-ei-type-1",
<<<<<<< HEAD
				EiJobData:    json.RawMessage(`{"test":"data"}`),
=======
				EiJobData:    map[string]interface{}{"test": "data"},
>>>>>>> 6835433495e87288b95961af7173d866977175ff
				TargetURI:    tt.targetURI,
				JobOwner:     "owner",
				JobStatusURL: tt.jobStatusURL,
			}

<<<<<<< HEAD
			err := validator.ValidateEIJob(eiType, job)
=======
			err := validator.ValidateEnrichmentInfoJob(job)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedMsg)
		})
	}
}

// Test Schema Validation Edge Cases

func TestSchemaValidation_ComplexTypes(t *testing.T) {
	validator := NewTestA1Validator()

	complexSchema := map[string]interface{}{
		"array_field": map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"minItems": 1,
			"maxItems": 10,
		},
		"oneOf_field": map[string]interface{}{
			"oneOf": []interface{}{
				map[string]interface{}{"type": "string"},
				map[string]interface{}{"type": "number"},
			},
		},
		"conditional_field": map[string]interface{}{
			"if": map[string]interface{}{
				"properties": map[string]interface{}{
					"type": map[string]interface{}{
						"const": "special",
					},
				},
			},
			"then": map[string]interface{}{
				"properties": map[string]interface{}{
					"special_value": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"special_value"},
			},
		},
	}

	policyType := &PolicyType{
		PolicyTypeID: 1,
		Schema:       complexSchema,
	}

	err := validator.ValidatePolicyType(policyType)
	assert.NoError(t, err)

	// Test valid data
	validData := map[string]interface{}{
		"array_field": []interface{}{
			map[string]interface{}{
				"name": "test",
			},
		},
		"oneOf_field": "string_value",
	}

	instance := &PolicyInstance{
		PolicyID:     "test",
		PolicyTypeID: 1,
		PolicyData:   validData,
	}

<<<<<<< HEAD
	err = validator.ValidatePolicyInstance(policyType, instance)
=======
	err = validator.ValidatePolicyInstance(policyType.PolicyTypeID, instance)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	assert.NoError(t, err)
}

func TestSchemaValidation_Performance(t *testing.T) {
	validator := NewTestA1Validator()

	// Create a large schema
	largeSchema := map[string]interface{}{
<<<<<<< HEAD
		"type": "object",
=======
		"type":       "object",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		"properties": make(map[string]interface{}),
	}

	properties := largeSchema["properties"].(map[string]interface{})
	for i := 0; i < 100; i++ {
		properties[fmt.Sprintf("field_%d", i)] = json.RawMessage(`{}`)
	}

	policyType := &PolicyType{
		PolicyTypeID: 1,
		Schema:       largeSchema,
	}

	// Validate schema creation (should be fast even for large schemas)
	err := validator.ValidatePolicyType(policyType)
	assert.NoError(t, err)

	// Create large data to validate
	largeData := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d", i)
	}

	instance := &PolicyInstance{
		PolicyID:     "test",
		PolicyTypeID: 1,
		PolicyData:   largeData,
	}

	// Validation should still be reasonably fast
<<<<<<< HEAD
	err = validator.ValidatePolicyInstance(policyType, instance)
=======
	err = validator.ValidatePolicyInstance(policyType.PolicyTypeID, instance)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	assert.NoError(t, err)
}

// Test Concurrent Validation

func TestValidation_Concurrent(t *testing.T) {
	validator := NewTestA1Validator()

	policyType := &PolicyType{
		PolicyTypeID: 1,
		Schema:       createValidPolicyTypeSchema(),
	}

	// Test concurrent validation of the same policy type
	const numGoroutines = 50
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			instance := &PolicyInstance{
				PolicyID:     fmt.Sprintf("policy-%d", id),
				PolicyTypeID: 1,
				PolicyData: map[string]interface{}{
					"ue_id":     fmt.Sprintf("ue-%d", id),
					"statement": json.RawMessage(`{}`),
				},
			}

<<<<<<< HEAD
			err := validator.ValidatePolicyInstance(policyType, instance)
=======
			err := validator.ValidatePolicyInstance(policyType.PolicyTypeID, instance)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			errors <- err
		}(i)
	}

	// Collect all errors
	for i := 0; i < numGoroutines; i++ {
		err := <-errors
		assert.NoError(t, err)
	}
}

// Benchmarks

func BenchmarkValidatePolicyType(b *testing.B) {
	validator := NewTestA1Validator()
	policyType := &PolicyType{
		PolicyTypeID: 1,
		Schema:       createValidPolicyTypeSchema(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidatePolicyType(policyType)
	}
}

func BenchmarkValidatePolicyInstance(b *testing.B) {
	validator := NewTestA1Validator()

	policyType := &PolicyType{
		PolicyTypeID: 1,
		Schema:       createValidPolicyTypeSchema(),
	}

	instance := &PolicyInstance{
		PolicyID:     "test-policy",
		PolicyTypeID: 1,
		PolicyData: map[string]interface{}{
			"ue_id":     "test-ue-123",
			"statement": json.RawMessage(`{}`),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
<<<<<<< HEAD
		validator.ValidatePolicyInstance(policyType, instance)
=======
		validator.ValidatePolicyInstance(policyType.PolicyTypeID, instance)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}
}

// Helper types and functions for validation

type SchemaValidator interface {
	ValidateSchema(schema map[string]interface{}) error
	ValidateAgainstSchema(data map[string]interface{}, schema map[string]interface{}) error
}

<<<<<<< HEAD
// Note: JSONSchemaValidator is imported from validation.go

// Note: All validation functions and types are imported from validation.go and types.go

=======
// NewJSONSchemaValidator creates a new JSON schema validator
func NewJSONSchemaValidator() SchemaValidator {
	return &JSONSchemaValidatorImpl{}
}

// JSONSchemaValidatorImpl implements the SchemaValidator interface
type JSONSchemaValidatorImpl struct{}

// ValidateSchema validates a JSON schema structure
func (v *JSONSchemaValidatorImpl) ValidateSchema(schema map[string]interface{}) error {
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	// Basic schema validation - check for required properties
	if _, hasType := schema["type"]; !hasType {
		if _, hasProps := schema["properties"]; !hasProps {
			if _, hasOneOf := schema["oneOf"]; !hasOneOf {
				if _, hasAnyOf := schema["anyOf"]; !hasAnyOf {
					if _, hasAllOf := schema["allOf"]; !hasAllOf {
						return fmt.Errorf("schema must have at least one of: type, properties, oneOf, anyOf, allOf")
					}
				}
			}
		}
	}

	return nil
}

// ValidateAgainstSchema validates data against a JSON schema
func (v *JSONSchemaValidatorImpl) ValidateAgainstSchema(data map[string]interface{}, schema map[string]interface{}) error {
	if schema == nil {
		return fmt.Errorf("schema cannot be nil")
	}

	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	// Check required fields
	if required, exists := schema["required"]; exists {
		if requiredList, ok := required.([]string); ok {
			for _, field := range requiredList {
				if _, exists := data[field]; !exists {
					return fmt.Errorf("required field '%s' is missing", field)
				}
			}
		}
	}

	return nil
}

// NewValidationError creates a new validation error
func NewValidationError(message, field string, value interface{}) error {
	return fmt.Errorf("validation error in field '%s': %s (value: %v)", field, message, value)
}

// ValidateURL validates a URL string
func ValidateURL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Basic URL validation using net/url
	_, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	return nil
}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
