// Package a1 provides comprehensive unit tests for A1 type definitions, serialization, and validation
package a1

import (
	"encoding/json"
	"testing"
	"time"

	validator "github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ValidateStruct validates a struct using the validator package
func ValidateStruct(s interface{}) error {
	validate := validator.New()
	return validate.Struct(s)
}

// Test A1Interface enum
func TestA1Interface_String(t *testing.T) {
	tests := []struct {
		name       string
		interface_ A1Interface
		expected   string
	}{
		{"Policy Interface", A1PolicyInterface, "A1-P"},
		{"Consumer Interface", A1ConsumerInterface, "A1-C"},
		{"Enrichment Interface", A1EnrichmentInterface, "A1-EI"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.interface_))
		})
	}
}

func TestA1Interface_JSON_Serialization(t *testing.T) {
	tests := []struct {
		name       string
		interface_ A1Interface
		expected   string
	}{
		{"Policy Interface", A1PolicyInterface, `"A1-P"`},
		{"Consumer Interface", A1ConsumerInterface, `"A1-C"`},
		{"Enrichment Interface", A1EnrichmentInterface, `"A1-EI"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.interface_)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(jsonData))

			var unmarshaled A1Interface
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err)
			assert.Equal(t, tt.interface_, unmarshaled)
		})
	}
}

// Test A1Version enum
func TestA1Version_Values(t *testing.T) {
	tests := []struct {
		name     string
		version  A1Version
		expected string
	}{
		{"Policy Version", A1PolicyVersion, "v2"},
		{"Consumer Version", A1ConsumerVersion, "v1"},
		{"Enrichment Version", A1EnrichmentVersion, "v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.version))
		})
	}
}

// The existing type tests remain the same as in the original file

// Additional Test: Deeper Validation for Complex A1 Scenarios
func TestTypes_AdvancedValidation(t *testing.T) {
	tests := []struct {
		name         string
		testStruct   interface{}
		expectErrors bool
		errorSubstr  []string
	}{
		{
			name: "Policy Type with Nested Complex Schema",
			testStruct: &PolicyType{
				PolicyTypeID: 1,
				Schema: map[string]interface{}{
					"nested": map[string]interface{}{
						"rules": []interface{}{
							map[string]interface{}{
								"type":     "qos",
								"priority": json.RawMessage(`{"value": 10}`),
							},
						},
					},
				},
			},
			expectErrors: false,
		},
		{
			name: "Policy Instance with Missing Critical Fields",
			testStruct: &PolicyInstance{
				PolicyID:     "",
				PolicyTypeID: 0,
				PolicyData:   nil,
			},
			expectErrors: true,
			errorSubstr:  []string{"policy_id", "policy_type_id", "policy_data"},
		},
		{
			name: "Enrichment Job with Invalid Target URI",
			testStruct: &EnrichmentInfoJob{
				EiJobID:   "test-job",
				EiTypeID:  "test-type",
				EiJobData: map[string]interface{}{"key": "value"},
				TargetURI: "invalid-uri",
				JobOwner:  "test-owner",
			},
			expectErrors: true,
			errorSubstr:  []string{"target_uri"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.testStruct)

			if tt.expectErrors {
				require.Error(t, err)
				for _, substr := range tt.errorSubstr {
					assert.Contains(t, err.Error(), substr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Enhanced Serialization Test for Mixed Data Types
func TestTypes_MixedDataTypesSerialization(t *testing.T) {
	complexData := map[string]interface{}{
		"string_field":  "test",
		"int_field":     42,
		"float_field":   3.14,
		"bool_field":    true,
		"null_field":    nil,
		"array_field":   []interface{}{1, "two", 3.0},
		"nested_object": map[string]interface{}{"inner": "value"},
	}

	instance := &PolicyInstance{
		PolicyID:     "mixed-types-policy",
		PolicyTypeID: 1,
		PolicyData:   complexData,
	}

	jsonData, err := json.Marshal(instance)
	require.NoError(t, err)

	var unmarshaled PolicyInstance
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	// Check type preservation during serialization/deserialization
	policyData := unmarshaled.PolicyData
	assert.Equal(t, "test", policyData["string_field"])
	assert.Equal(t, float64(42), policyData["int_field"])
	assert.Equal(t, 3.14, policyData["float_field"])
	assert.Equal(t, true, policyData["bool_field"])
	assert.Nil(t, policyData["null_field"])

	arrayField := policyData["array_field"].([]interface{})
	assert.Len(t, arrayField, 3)
	assert.Equal(t, float64(1), arrayField[0])
	assert.Equal(t, "two", arrayField[1])
	assert.Equal(t, float64(3.0), arrayField[2])

	nestedObject := policyData["nested_object"].(map[string]interface{})
	assert.Equal(t, "value", nestedObject["inner"])
}

// Regression Test for Known Edge Cases
func TestTypes_RegressionEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		testFunc func(t *testing.T)
	}{
		{
			name: "Empty JSON RawMessage Handling",
			testFunc: func(t *testing.T) {
				status := &PolicyStatus{
					EnforcementStatus: "ENFORCED",
					AdditionalInfo:    json.RawMessage(`{}`),
				}

				jsonData, err := json.Marshal(status)
				require.NoError(t, err)

				var unmarshaled PolicyStatus
				err = json.Unmarshal(jsonData, &unmarshaled)
				require.NoError(t, err)

				assert.JSONEq(t, `{}`, string(unmarshaled.AdditionalInfo))
			},
		},
		{
			name: "Timestamp Precision Preservation",
			testFunc: func(t *testing.T) {
				now := time.Now().UTC()
				policyType := &PolicyType{
					PolicyTypeID: 1,
					CreatedAt:    now,
					ModifiedAt:   now,
				}

				jsonData, err := json.Marshal(policyType)
				require.NoError(t, err)

				var unmarshaled PolicyType
				err = json.Unmarshal(jsonData, &unmarshaled)
				require.NoError(t, err)

				assert.True(t, now.Equal(unmarshaled.CreatedAt))
				assert.True(t, now.Equal(unmarshaled.ModifiedAt))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.testFunc)
	}
}

// The remaining helper types and validation function remain the same as in the original file
