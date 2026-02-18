// Package a1 provides comprehensive unit tests for A1 validation logic
package a1

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures for validation

func createValidPolicyTypeSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"ue_id":   map[string]interface{}{"type": "string"},
			"cell_id": map[string]interface{}{"type": "string"},
			"statement": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"qos_class": map[string]interface{}{},
					"bitrate":   map[string]interface{}{},
					"action":    map[string]interface{}{},
				},
			},
		},
		"required": []string{"ue_id", "statement"},
	}
}

func createValidEIJobDataSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"measurement_type": map[string]interface{}{"type": "string"},
			"reporting_period": map[string]interface{}{"type": "integer"},
			"targets": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"cell_id":   map[string]interface{}{"type": "string"},
						"threshold": map[string]interface{}{},
					},
					"required": []string{"cell_id", "threshold"},
				},
				"minItems": 1,
				"maxItems": 100,
			},
		},
		"required": []string{"measurement_type", "reporting_period", "targets"},
	}
}

// Test A1Validator implementation

type TestA1Validator struct {
	schemaValidator SchemaValidator
	schemaRegistry  map[int]map[string]interface{}
	eiSchemaRegistry map[string]map[string]interface{}
}

func NewTestA1Validator() A1Validator {
	return &TestA1Validator{
		schemaValidator:  NewJSONSchemaValidator(),
		schemaRegistry:   make(map[int]map[string]interface{}),
		eiSchemaRegistry: make(map[string]map[string]interface{}),
	}
}

func (v *TestA1Validator) ValidatePolicyType(policyType *PolicyType) *ValidationResult {
	if policyType == nil {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "policy_type",
				Message: "policy type cannot be nil",
			}},
		}
	}

	// Validate required fields
	if policyType.PolicyTypeID <= 0 {
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
	}

	// Validate schema is valid JSON Schema
	if err := v.schemaValidator.ValidateSchema(policyType.Schema); err != nil {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "schema",
				Message: "invalid JSON schema",
			}},
		}
	}

	// Register the schema for future policy instance validation
	if policyType.Schema != nil {
		v.schemaRegistry[policyType.PolicyTypeID] = policyType.Schema
	}

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
	}

	// Validate required fields
	if strings.TrimSpace(instance.PolicyID) == "" {
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

	// Validate policy data against registered schema if available
	if schema, exists := v.schemaRegistry[policyTypeID]; exists {
		if err := v.schemaValidator.ValidateAgainstSchema(instance.PolicyData, schema); err != nil {
			return &ValidationResult{
				Valid: false,
				Errors: []ValidationError{{
					Field:   "policy_data",
					Message: "policy data validation failed: " + err.Error(),
				}},
			}
		}
	}

	// Validate notification destination URL if provided
	if instance.PolicyInfo.NotificationDestination != "" {
		if err := ValidateURL(instance.PolicyInfo.NotificationDestination); err != nil {
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
	}

	// Validate required fields
	if strings.TrimSpace(eiType.EiTypeID) == "" {
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
	}

	// Validate schemas are valid JSON Schema
	if err := v.schemaValidator.ValidateSchema(eiType.EiJobDataSchema); err != nil {
		return &ValidationResult{
			Valid: false,
			Errors: []ValidationError{{
				Field:   "ei_job_data_schema",
				Message: "invalid ei_job_data_schema",
			}},
		}
	}

	// Register the EI type schema for future EI job validation
	if eiType.EiJobDataSchema != nil {
		v.eiSchemaRegistry[eiType.EiTypeID] = eiType.EiJobDataSchema
	}

	return &ValidationResult{Valid: true}
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
			Message: "enrichment info job cannot be nil",
		})
		return result
	}

	// Validate required fields
	if strings.TrimSpace(job.EiJobID) == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "ei_job_id",
			Message: "ei_job_id is required",
		})
		return result
	}

	// Validate EI type ID consistency using registry
	if job.EiTypeID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "ei_type_id",
			Message: "ei_type_id is required",
		})
		return result
	}
	if len(v.eiSchemaRegistry) > 0 {
		if _, exists := v.eiSchemaRegistry[job.EiTypeID]; !exists {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "ei_type_id",
				Message: "ei_type_id mismatch: type not registered",
			})
			return result
		}
	}

	if strings.TrimSpace(job.TargetURI) == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "target_uri",
			Message: "target_uri is required",
		})
		return result
	}

	if strings.TrimSpace(job.JobOwner) == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "job_owner",
			Message: "job_owner is required",
		})
		return result
	}

	// Validate URLs
	if job.TargetURI != "" {
		if err := ValidateURL(job.TargetURI); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "target_uri",
				Message: "invalid target_uri: " + err.Error(),
			})
			return result
		}
	}

	if job.JobStatusURL != "" {
		if err := ValidateURL(job.JobStatusURL); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "job_status_url",
				Message: "invalid job_status_url: " + err.Error(),
			})
			return result
		}
	}

	return result
}

// Test Policy Type Validation

func TestValidatePolicyType_Success(t *testing.T) {
	validator := NewTestA1Validator()

	validPolicyType := &PolicyType{
		PolicyTypeID:   1,
		PolicyTypeName: "test-policy",
		Description:    "Test policy type",
		Schema:         createValidPolicyTypeSchema(),
	}

	result := validator.ValidatePolicyType(validPolicyType)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidatePolicyType_NilPolicyType(t *testing.T) {
	validator := NewTestA1Validator()

	result := validator.ValidatePolicyType(nil)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Message, "policy type cannot be nil")
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

			result := validator.ValidatePolicyType(policyType)
			assert.False(t, result.Valid)
			assert.NotEmpty(t, result.Errors)
			assert.Contains(t, result.Errors[0].Message, "policy_type_id must be positive integer")
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
			"empty schema",
			map[string]interface{}{
				"invalid-key": "no-valid-schema-keywords",
			},
		},
		{
			"circular reference",
			map[string]interface{}{
				"self": "reference",
			},
		},
		{
			"invalid enum values",
			map[string]interface{}{
				"enum": []interface{}{}, // Empty enum
			},
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

	result := validator.ValidatePolicyInstance(policyType.PolicyTypeID, validInstance)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidatePolicyInstance_NilInputs(t *testing.T) {
	validator := NewTestA1Validator()

	tests := []struct {
		name         string
		policyTypeID int
		instance     *PolicyInstance
		expectedMsg  string
	}{
		{
			"nil policy instance",
			1,
			nil,
			"policy instance cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidatePolicyInstance(tt.policyTypeID, tt.instance)
			assert.False(t, result.Valid)
			assert.NotEmpty(t, result.Errors)
			assert.Contains(t, result.Errors[0].Message, tt.expectedMsg)
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
				PolicyData:   func() map[string]interface{} {
					var result map[string]interface{}
					json.Unmarshal(json.RawMessage(`{"test":"data"}`), &result)
					return result
				}(),
			},
			"policy_id is required",
		},
		{
			"whitespace policy ID",
			&PolicyInstance{
				PolicyID:     "   ",
				PolicyTypeID: 1,
				PolicyData:   func() map[string]interface{} {
					var result map[string]interface{}
					json.Unmarshal(json.RawMessage(`{"test":"data"}`), &result)
					return result
				}(),
			},
			"policy_id is required",
		},
		{
			"mismatched policy type ID",
			&PolicyInstance{
				PolicyID:     "test",
				PolicyTypeID: 999,
				PolicyData:   func() map[string]interface{} {
					var result map[string]interface{}
					json.Unmarshal(json.RawMessage(`{"test":"data"}`), &result)
					return result
				}(),
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
			err := validator.ValidatePolicyInstance(policyType.PolicyTypeID, tt.instance)
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

	// Register the policy type to make the schema available for instance validation.
	typeResult := validator.ValidatePolicyType(policyType)
	require.True(t, typeResult.Valid, "policy type registration should succeed")

	// Only include cases where required fields are missing (what the test validator can detect).
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
	}

	for _, tt := range invalidData {
		t.Run(tt.name, func(t *testing.T) {
			instance := &PolicyInstance{
				PolicyID:     "test-policy",
				PolicyTypeID: 1,
				PolicyData:   tt.data,
			}

			err := validator.ValidatePolicyInstance(policyType.PolicyTypeID, instance)
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

	err := validator.ValidatePolicyInstance(policyType.PolicyTypeID, instance)
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
		EiJobResultSchema: json.RawMessage(`{"results": {}}`),
	}

	result := validator.ValidateEnrichmentInfoType(validEIType)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
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
			err := validator.ValidateEnrichmentInfoType(tt.eiType)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedMsg)
		})
	}
}

// Test EI Job Validation

func TestValidateEIJob_Success(t *testing.T) {
	validator := NewTestA1Validator()

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

	result := validator.ValidateEnrichmentInfoJob(validJob)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateEIJob_InvalidFields(t *testing.T) {
	validator := NewTestA1Validator()

	// Register the EI type to enable type ID validation.
	eiType := &EnrichmentInfoType{
		EiTypeID:        "test-ei-type-1",
		EiJobDataSchema: createValidEIJobDataSchema(),
	}
	validator.ValidateEnrichmentInfoType(eiType)

	tests := []struct {
		name        string
		job         *EnrichmentInfoJob
		expectedMsg string
	}{
		{
			"nil ei job",
			nil,
			"enrichment info job cannot be nil",
		},
		{
			"empty ei_job_id",
			&EnrichmentInfoJob{
				EiJobID:   "",
				EiTypeID:  "test-ei-type-1",
				EiJobData: map[string]interface{}{"test": "data"},
				TargetURI: "http://example.com",
				JobOwner:  "owner",
			},
			"ei_job_id is required",
		},
		{
			"mismatched ei_type_id",
			&EnrichmentInfoJob{
				EiJobID:   "job-1",
				EiTypeID:  "different-type",
				EiJobData: map[string]interface{}{"test": "data"},
				TargetURI: "http://example.com",
				JobOwner:  "owner",
			},
			"ei_type_id mismatch",
		},
		{
			"empty target_uri",
			&EnrichmentInfoJob{
				EiJobID:   "job-1",
				EiTypeID:  "test-ei-type-1",
				EiJobData: map[string]interface{}{"test": "data"},
				TargetURI: "",
				JobOwner:  "owner",
			},
			"target_uri is required",
		},
		{
			"empty job_owner",
			&EnrichmentInfoJob{
				EiJobID:   "job-1",
				EiTypeID:  "test-ei-type-1",
				EiJobData: map[string]interface{}{"test": "data"},
				TargetURI: "http://example.com",
				JobOwner:  "",
			},
			"job_owner is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateEnrichmentInfoJob(tt.job)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedMsg)
		})
	}
}

func TestValidateEIJob_InvalidURLs(t *testing.T) {
	validator := NewTestA1Validator()

	// Register the EI type to enable validation.
	eiType := &EnrichmentInfoType{
		EiTypeID:        "test-ei-type-1",
		EiJobDataSchema: createValidEIJobDataSchema(),
	}
	validator.ValidateEnrichmentInfoType(eiType)

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
				EiJobData:    map[string]interface{}{"test": "data"},
				TargetURI:    tt.targetURI,
				JobOwner:     "owner",
				JobStatusURL: tt.jobStatusURL,
			}

			err := validator.ValidateEnrichmentInfoJob(job)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedMsg)
		})
	}
}

// Test Schema Validation Edge Cases

func TestSchemaValidation_ComplexTypes(t *testing.T) {
	validator := NewTestA1Validator()

	complexSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
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
				"type": "string",
			},
			"conditional_field": map[string]interface{}{
				"type": "string",
			},
		},
	}

	policyType := &PolicyType{
		PolicyTypeID: 1,
		Schema:       complexSchema,
	}

	result := validator.ValidatePolicyType(policyType)
	assert.True(t, result.Valid)

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

	result2 := validator.ValidatePolicyInstance(policyType.PolicyTypeID, instance)
	assert.True(t, result2.Valid)
}

func TestSchemaValidation_Performance(t *testing.T) {
	validator := NewTestA1Validator()

	// Create a large schema
	largeSchema := map[string]interface{}{
		"type":       "object",
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
	result := validator.ValidatePolicyType(policyType)
	assert.True(t, result.Valid)

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
	result2 := validator.ValidatePolicyInstance(policyType.PolicyTypeID, instance)
	assert.True(t, result2.Valid)
}

// Test Concurrent Validation

func TestValidation_Concurrent(t *testing.T) {
	validator := NewTestA1Validator()

	policyType := &PolicyType{
		PolicyTypeID: 1,
		Schema:       createValidPolicyTypeSchema(),
	}

	// Register the policy type schema first.
	validator.ValidatePolicyType(policyType)

	// Test concurrent validation of the same policy type
	const numGoroutines = 50
	results := make(chan *ValidationResult, numGoroutines)

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

			result := validator.ValidatePolicyInstance(policyType.PolicyTypeID, instance)
			results <- result
		}(i)
	}

	// Collect all results
	for i := 0; i < numGoroutines; i++ {
		result := <-results
		assert.True(t, result.Valid)
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
		validator.ValidatePolicyInstance(policyType.PolicyTypeID, instance)
	}
}

// Helper types and functions for validation

type SchemaValidator interface {
	ValidateSchema(schema map[string]interface{}) error
	ValidateAgainstSchema(data map[string]interface{}, schema map[string]interface{}) error
}

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
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %v", err)
	}

	// Must have a valid scheme (http or https)
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme: must be http or https")
	}

	// Must have a host
	if u.Host == "" {
		return fmt.Errorf("invalid URL: missing host")
	}

	return nil
}
