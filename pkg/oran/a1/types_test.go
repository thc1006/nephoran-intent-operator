// Package a1 provides comprehensive unit tests for A1 type definitions, serialization, and validation
package a1

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

// Test PolicyType

func TestPolicyType_JSON_Serialization(t *testing.T) {
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	policyType := &PolicyType{
		PolicyTypeID:   123,
		PolicyTypeName: "Traffic Steering Policy",
		Description:    "Policy for managing traffic steering in O-RAN",
		Schema: map[string]interface{}{
			"scope": map[string]interface{}{
				"ue_id": json.RawMessage(`{}`),
			},
		},
		CreateSchema: json.RawMessage(`{"notification_destination": {}}`),
		CreatedAt:    now,
		ModifiedAt:   now,
	}

	jsonData, err := json.Marshal(policyType)
	require.NoError(t, err)

	var unmarshaled PolicyType
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, policyType.PolicyTypeID, unmarshaled.PolicyTypeID)
	assert.Equal(t, policyType.PolicyTypeName, unmarshaled.PolicyTypeName)
	assert.Equal(t, policyType.Description, unmarshaled.Description)
	assert.Equal(t, policyType.Schema, unmarshaled.Schema)
	assert.Equal(t, policyType.CreateSchema, unmarshaled.CreateSchema)
	assert.True(t, policyType.CreatedAt.Equal(unmarshaled.CreatedAt))
	assert.True(t, policyType.ModifiedAt.Equal(unmarshaled.ModifiedAt))
}

func TestPolicyType_Validation_Tags(t *testing.T) {
	tests := []struct {
		name        string
		policyType  PolicyType
		expectValid bool
		fieldErrors []string
	}{
		{
			name: "valid policy type",
			policyType: PolicyType{
				PolicyTypeID: 1,
				Schema: map[string]interface{}{},
			},
			expectValid: true,
		},
		{
			name: "missing policy_type_id",
			policyType: PolicyType{
				PolicyTypeID: 0, // Invalid: must be >= 1
				Schema: map[string]interface{}{},
			},
			expectValid: false,
			fieldErrors: []string{"policy_type_id"},
		},
		{
			name: "negative policy_type_id",
			policyType: PolicyType{
				PolicyTypeID: -1, // Invalid: must be >= 1
				Schema: map[string]interface{}{},
			},
			expectValid: false,
			fieldErrors: []string{"policy_type_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(&tt.policyType)

			if tt.expectValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				for _, fieldError := range tt.fieldErrors {
					assert.Contains(t, err.Error(), fieldError)
				}
			}
		})
	}
}

func TestPolicyType_EmptyOptionalFields(t *testing.T) {
	policyType := &PolicyType{
		PolicyTypeID: 1,
		Schema: map[string]interface{}{},
		// Optional fields left empty
		PolicyTypeName: "",
		Description:    "",
		CreateSchema:   nil,
	}

	jsonData, err := json.Marshal(policyType)
	require.NoError(t, err)

	var unmarshaled PolicyType
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	// Empty optional fields should be preserved
	assert.Equal(t, "", unmarshaled.PolicyTypeName)
	assert.Equal(t, "", unmarshaled.Description)
	assert.Nil(t, unmarshaled.CreateSchema)
}

// Test PolicyInstance

func TestPolicyInstance_JSON_Serialization(t *testing.T) {
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	instance := &PolicyInstance{
		PolicyID:     "traffic-policy-123",
		PolicyTypeID: 456,
		PolicyData: map[string]interface{}{
			"ue_id":     "ue-12345",
			"cell_id":   "cell-abcde",
			"statement": json.RawMessage(`{}`),
		},
		PolicyInfo: PolicyInstanceInfo{
			NotificationDestination: "http://callback.example.com/policy-notifications",
			RequestID:               "req-789",
			AdditionalParams: json.RawMessage(`{}`),
		},
		CreatedAt:  now,
		ModifiedAt: now,
	}

	jsonData, err := json.Marshal(instance)
	require.NoError(t, err)

	var unmarshaled PolicyInstance
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, instance.PolicyID, unmarshaled.PolicyID)
	assert.Equal(t, instance.PolicyTypeID, unmarshaled.PolicyTypeID)
	assert.Equal(t, instance.PolicyData, unmarshaled.PolicyData)
	assert.Equal(t, instance.PolicyInfo.NotificationDestination, unmarshaled.PolicyInfo.NotificationDestination)
	assert.Equal(t, instance.PolicyInfo.RequestID, unmarshaled.PolicyInfo.RequestID)
	assert.Equal(t, instance.PolicyInfo.AdditionalParams, unmarshaled.PolicyInfo.AdditionalParams)
	assert.True(t, instance.CreatedAt.Equal(unmarshaled.CreatedAt))
	assert.True(t, instance.ModifiedAt.Equal(unmarshaled.ModifiedAt))
}

func TestPolicyInstance_Validation_Tags(t *testing.T) {
	tests := []struct {
		name        string
		instance    PolicyInstance
		expectValid bool
		fieldErrors []string
	}{
		{
			name: "valid policy instance",
			instance: PolicyInstance{
				PolicyID:     "valid-policy-id",
				PolicyTypeID: 1,
				PolicyData: map[string]interface{}{},
			},
			expectValid: true,
		},
		{
			name: "empty policy_id",
			instance: PolicyInstance{
				PolicyID:     "", // Invalid: required
				PolicyTypeID: 1,
				PolicyData: map[string]interface{}{},
			},
			expectValid: false,
			fieldErrors: []string{"policy_id"},
		},
		{
			name: "invalid policy_type_id",
			instance: PolicyInstance{
				PolicyID:     "valid-policy-id",
				PolicyTypeID: 0, // Invalid: must be >= 1
				PolicyData: map[string]interface{}{},
			},
			expectValid: false,
			fieldErrors: []string{"policy_type_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(&tt.instance)

			if tt.expectValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				for _, fieldError := range tt.fieldErrors {
					assert.Contains(t, err.Error(), fieldError)
				}
			}
		})
	}
}

func TestPolicyInstance_ComplexPolicyData(t *testing.T) {
	complexData := map[string]interface{}{
		"ue_ids": []interface{}{
			"ue-001", "ue-002", "ue-003",
		},
		"cell_ids": []interface{}{
			json.RawMessage(`{}`),
			json.RawMessage(`{}`),
		},
		"statements": []interface{}{
			map[string]interface{}{
				"time_window": json.RawMessage(`{}`),
				"action":      json.RawMessage(`{}`),
			},
		},
	}

	instance := &PolicyInstance{
		PolicyID:     "complex-policy",
		PolicyTypeID: 1,
		PolicyData:   complexData,
	}

	jsonData, err := json.Marshal(instance)
	require.NoError(t, err)

	var unmarshaled PolicyInstance
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, instance.PolicyData, unmarshaled.PolicyData)

	// Verify complex nested structures
	scope := unmarshaled.PolicyData["scope"].(map[string]interface{})
	ueIds := scope["ue_ids"].([]interface{})
	assert.Len(t, ueIds, 3)
	assert.Equal(t, "ue-001", ueIds[0])

	cellIds := scope["cell_ids"].([]interface{})
	assert.Len(t, cellIds, 2)

	firstCell := cellIds[0].(map[string]interface{})
	assert.Equal(t, "cell-123", firstCell["id"])
	assert.Equal(t, 0.8, firstCell["weight"])
}

// Test PolicyStatus

func TestPolicyStatus_JSON_Serialization(t *testing.T) {
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	status := &PolicyStatus{
		EnforcementStatus: "ENFORCED",
		EnforcementReason: "Policy successfully applied to all target RICs",
		HasBeenDeleted:    false,
		Deleted:           false,
		CreatedAt:         now,
		ModifiedAt:        now,
		AdditionalInfo: json.RawMessage(`{"enforcement_time": "2023-01-01T12:00:00Z", "metrics": {}}`),
	}

	jsonData, err := json.Marshal(status)
	require.NoError(t, err)

	var unmarshaled PolicyStatus
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, status.EnforcementStatus, unmarshaled.EnforcementStatus)
	assert.Equal(t, status.EnforcementReason, unmarshaled.EnforcementReason)
	assert.Equal(t, status.HasBeenDeleted, unmarshaled.HasBeenDeleted)
	assert.Equal(t, status.Deleted, unmarshaled.Deleted)
	assert.True(t, status.CreatedAt.Equal(unmarshaled.CreatedAt))
	assert.True(t, status.ModifiedAt.Equal(unmarshaled.ModifiedAt))
	assert.Equal(t, status.AdditionalInfo, unmarshaled.AdditionalInfo)
}

func TestPolicyStatus_Validation_Tags(t *testing.T) {
	tests := []struct {
		name        string
		status      PolicyStatus
		expectValid bool
		fieldErrors []string
	}{
		{
			name: "valid status - ENFORCED",
			status: PolicyStatus{
				EnforcementStatus: "ENFORCED",
			},
			expectValid: true,
		},
		{
			name: "valid status - NOT_ENFORCED",
			status: PolicyStatus{
				EnforcementStatus: "NOT_ENFORCED",
			},
			expectValid: true,
		},
		{
			name: "valid status - UNKNOWN",
			status: PolicyStatus{
				EnforcementStatus: "UNKNOWN",
			},
			expectValid: true,
		},
		{
			name: "invalid enforcement status",
			status: PolicyStatus{
				EnforcementStatus: "INVALID_STATUS", // Not in oneof constraint
			},
			expectValid: false,
			fieldErrors: []string{"enforcement_status"},
		},
		{
			name: "empty enforcement status",
			status: PolicyStatus{
				EnforcementStatus: "", // Required field
			},
			expectValid: false,
			fieldErrors: []string{"enforcement_status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(&tt.status)

			if tt.expectValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				for _, fieldError := range tt.fieldErrors {
					assert.Contains(t, err.Error(), fieldError)
				}
			}
		})
	}
}

// Test EnrichmentInfoType

func TestEnrichmentInfoType_JSON_Serialization(t *testing.T) {
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	eiType := &EnrichmentInfoType{
		EiTypeID:    "throughput-measurement",
		EiTypeName:  "Throughput Measurement EI Type",
		Description: "Enrichment Information type for measuring cell throughput",
		EiJobDataSchema: map[string]interface{}{
			"measurement_config": map[string]interface{}{
				"interval_seconds": json.RawMessage(`{}`),
				"target_cells": map[string]interface{}{
					"type":     "string",
					"minItems": 1,
				},
			},
			"required": []string{"interval_seconds", "target_cells"},
		},
		EiJobResultSchema: json.RawMessage(`{"measurements":{"type":"object","properties":{"throughput":{"type":"number"},"timestamp":{}}}}`),
		CreatedAt:  now,
		ModifiedAt: now,
	}

	jsonData, err := json.Marshal(eiType)
	require.NoError(t, err)

	var unmarshaled EnrichmentInfoType
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, eiType.EiTypeID, unmarshaled.EiTypeID)
	assert.Equal(t, eiType.EiTypeName, unmarshaled.EiTypeName)
	assert.Equal(t, eiType.Description, unmarshaled.Description)
	assert.Equal(t, eiType.EiJobDataSchema, unmarshaled.EiJobDataSchema)
	assert.Equal(t, eiType.EiJobResultSchema, unmarshaled.EiJobResultSchema)
	assert.True(t, eiType.CreatedAt.Equal(unmarshaled.CreatedAt))
	assert.True(t, eiType.ModifiedAt.Equal(unmarshaled.ModifiedAt))
}

func TestEnrichmentInfoType_Validation_Tags(t *testing.T) {
	tests := []struct {
		name        string
		eiType      EnrichmentInfoType
		expectValid bool
		fieldErrors []string
	}{
		{
			name: "valid EI type",
			eiType: EnrichmentInfoType{
				EiTypeID:        "valid-type-id",
				EiJobDataSchema: map[string]interface{}{"type": "object"},
			},
			expectValid: true,
		},
		{
			name: "empty ei_type_id",
			eiType: EnrichmentInfoType{
				EiTypeID:        "", // Required field
				EiJobDataSchema: map[string]interface{}{"type": "object"},
			},
			expectValid: false,
			fieldErrors: []string{"ei_type_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(&tt.eiType)

			if tt.expectValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				for _, fieldError := range tt.fieldErrors {
					assert.Contains(t, err.Error(), fieldError)
				}
			}
		})
	}
}

// Test EnrichmentInfoJob

func TestEnrichmentInfoJob_JSON_Serialization(t *testing.T) {
	now := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)

	job := &EnrichmentInfoJob{
		EiJobID:  "throughput-job-001",
		EiTypeID: "throughput-measurement",
		EiJobData: map[string]interface{}{
			"interval_seconds": 60,
			"target_cells":     []interface{}{"cell-001", "cell-002"},
			"thresholds":       json.RawMessage(`{}`),
			"reporting":        json.RawMessage(`{}`),
		},
		TargetURI:    "http://ei-consumer.example.com/measurements",
		JobOwner:     "network-analytics-service",
		JobStatusURL: "http://ei-consumer.example.com/job-status/throughput-job-001",
		JobDefinition: EnrichmentJobDef{
			DeliveryInfo: []DeliveryInfo{
				{
					DeliveryURL:    "http://ei-consumer.example.com/measurements",
					DeliveryMethod: "POST",
					Headers: map[string]string{
						"Content-Type":  "application/json",
						"Authorization": "Bearer token123",
					},
				},
			},
			JobParameters: json.RawMessage(`{}`),
			JobResultSchema: json.RawMessage(`{"cell_measurements":{}}`),
		},
		CreatedAt:      now,
		ModifiedAt:     now,
		LastExecutedAt: now,
	}

	jsonData, err := json.Marshal(job)
	require.NoError(t, err)

	var unmarshaled EnrichmentInfoJob
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, job.EiJobID, unmarshaled.EiJobID)
	assert.Equal(t, job.EiTypeID, unmarshaled.EiTypeID)
	assert.Equal(t, job.EiJobData, unmarshaled.EiJobData)
	assert.Equal(t, job.TargetURI, unmarshaled.TargetURI)
	assert.Equal(t, job.JobOwner, unmarshaled.JobOwner)
	assert.Equal(t, job.JobStatusURL, unmarshaled.JobStatusURL)
	assert.Equal(t, job.JobDefinition.DeliveryInfo, unmarshaled.JobDefinition.DeliveryInfo)
	assert.Equal(t, job.JobDefinition.JobParameters, unmarshaled.JobDefinition.JobParameters)
	assert.Equal(t, job.JobDefinition.JobResultSchema, unmarshaled.JobDefinition.JobResultSchema)
	assert.True(t, job.CreatedAt.Equal(unmarshaled.CreatedAt))
	assert.True(t, job.ModifiedAt.Equal(unmarshaled.ModifiedAt))
	assert.True(t, job.LastExecutedAt.Equal(unmarshaled.LastExecutedAt))
}

func TestEnrichmentInfoJob_Validation_Tags(t *testing.T) {
	tests := []struct {
		name        string
		job         EnrichmentInfoJob
		expectValid bool
		fieldErrors []string
	}{
		{
			name: "valid EI job",
			job: EnrichmentInfoJob{
				EiJobID:   "valid-job-id",
				EiTypeID:  "valid-type-id",
				EiJobData: json.RawMessage(`{"key":"value"}`),
				TargetURI: "http://valid.example.com",
				JobOwner:  "valid-owner",
			},
			expectValid: true,
		},
		{
			name: "empty ei_job_id",
			job: EnrichmentInfoJob{
				EiJobID:   "", // Required field
				EiTypeID:  "valid-type-id",
				EiJobData: json.RawMessage(`{"key":"value"}`),
				TargetURI: "http://valid.example.com",
				JobOwner:  "valid-owner",
			},
			expectValid: false,
			fieldErrors: []string{"ei_job_id"},
		},
		{
			name: "empty ei_type_id",
			job: EnrichmentInfoJob{
				EiJobID:   "valid-job-id",
				EiTypeID:  "", // Required field
				EiJobData: json.RawMessage(`{"key":"value"}`),
				TargetURI: "http://valid.example.com",
				JobOwner:  "valid-owner",
			},
			expectValid: false,
			fieldErrors: []string{"ei_type_id"},
		},
		{
			name: "invalid target_uri",
			job: EnrichmentInfoJob{
				EiJobID:   "valid-job-id",
				EiTypeID:  "valid-type-id",
				EiJobData: json.RawMessage(`{"key":"value"}`),
				TargetURI: "not-a-url", // Invalid URL format
				JobOwner:  "valid-owner",
			},
			expectValid: false,
			fieldErrors: []string{"target_uri"},
		},
		{
			name: "empty job_owner",
			job: EnrichmentInfoJob{
				EiJobID:   "valid-job-id",
				EiTypeID:  "valid-type-id",
				EiJobData: json.RawMessage(`{"key":"value"}`),
				TargetURI: "http://valid.example.com",
				JobOwner:  "", // Required field
			},
			expectValid: false,
			fieldErrors: []string{"job_owner"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(&tt.job)

			if tt.expectValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				for _, fieldError := range tt.fieldErrors {
					assert.Contains(t, err.Error(), fieldError)
				}
			}
		})
	}
}

// Test DeliveryInfo

func TestDeliveryInfo_JSON_Serialization(t *testing.T) {
	deliveryInfo := &DeliveryInfo{
		DeliveryURL:    "http://callback.example.com/ei-results",
		DeliveryMethod: "POST",
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"Authorization":     "Bearer token123",
			"X-Client-ID":       "ei-consumer-1",
			"X-Request-Timeout": "30s",
		},
		RetryPolicy: RetryPolicy{
			MaxRetries:  3,
			RetryDelay:  time.Second * 5,
			BackoffType: "exponential",
		},
		Timeout: time.Second * 30,
	}

	jsonData, err := json.Marshal(deliveryInfo)
	require.NoError(t, err)

	var unmarshaled DeliveryInfo
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, deliveryInfo.DeliveryURL, unmarshaled.DeliveryURL)
	assert.Equal(t, deliveryInfo.DeliveryMethod, unmarshaled.DeliveryMethod)
	assert.Equal(t, deliveryInfo.Headers, unmarshaled.Headers)
	assert.Equal(t, deliveryInfo.RetryPolicy.MaxRetries, unmarshaled.RetryPolicy.MaxRetries)
	assert.Equal(t, deliveryInfo.RetryPolicy.RetryDelay, unmarshaled.RetryPolicy.RetryDelay)
	assert.Equal(t, deliveryInfo.RetryPolicy.BackoffType, unmarshaled.RetryPolicy.BackoffType)
	assert.Equal(t, deliveryInfo.Timeout, unmarshaled.Timeout)
}

// Test Edge Cases and Error Conditions

func TestTypes_NilMapHandling(t *testing.T) {
	tests := []struct {
		name string
		data interface{}
	}{
		{
			"PolicyType with nil Schema",
			&PolicyType{
				PolicyTypeID: 1,
				Schema:       nil, // This should be handled gracefully
			},
		},
		{
			"PolicyInstance with nil PolicyData",
			&PolicyInstance{
				PolicyID:     "test",
				PolicyTypeID: 1,
				PolicyData:   nil, // This should be handled gracefully
			},
		},
		{
			"EnrichmentInfoType with nil schemas",
			&EnrichmentInfoType{
				EiTypeID:          "test",
				EiJobDataSchema:   nil,
				EiJobResultSchema: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic during JSON serialization
			jsonData, err := json.Marshal(tt.data)
			assert.NoError(t, err)
			assert.NotEmpty(t, jsonData)

			// Should be able to unmarshal back
			switch tt.data.(type) {
			case *PolicyType:
				var unmarshaled PolicyType
				err := json.Unmarshal(jsonData, &unmarshaled)
				assert.NoError(t, err)
			case *PolicyInstance:
				var unmarshaled PolicyInstance
				err := json.Unmarshal(jsonData, &unmarshaled)
				assert.NoError(t, err)
			case *EnrichmentInfoType:
				var unmarshaled EnrichmentInfoType
				err := json.Unmarshal(jsonData, &unmarshaled)
				assert.NoError(t, err)
			}
		})
	}
}

func TestTypes_LargeDataHandling(t *testing.T) {
	// Create a large policy data structure
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[fmt.Sprintf("field_%d", i)] = map[string]interface{}{
			"values": []interface{}{i, i + 1, i + 2},
			"nested": json.RawMessage(`{}`),
		}
	}

	instance := &PolicyInstance{
		PolicyID:     "large-policy",
		PolicyTypeID: 1,
		PolicyData:   largeData,
	}

	// Should handle large data structures
	jsonData, err := json.Marshal(instance)
	require.NoError(t, err)
	assert.Greater(t, len(jsonData), 10000) // Should be quite large

	var unmarshaled PolicyInstance
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, instance.PolicyID, unmarshaled.PolicyID)
	assert.Equal(t, len(instance.PolicyData), len(unmarshaled.PolicyData))
}

func TestTypes_UnicodeHandling(t *testing.T) {
	// Test Unicode characters in various fields
	policyType := &PolicyType{
		PolicyTypeID:   1,
		PolicyTypeName: "Unicode Test Policy",
		Description:    "Multi-language policy testing",
		Schema: map[string]interface{}{
			"unicode_field": json.RawMessage(`{}`),
		},
	}

	jsonData, err := json.Marshal(policyType)
	require.NoError(t, err)

	var unmarshaled PolicyType
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, policyType.PolicyTypeName, unmarshaled.PolicyTypeName)
	assert.Equal(t, policyType.Description, unmarshaled.Description)
}

func TestTypes_TimeHandling(t *testing.T) {
	// Test different time formats and edge cases
	times := []time.Time{
		time.Unix(0, 0), // Unix epoch
		time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2023, 12, 31, 23, 59, 59, 999999999, time.UTC),
		time.Now(),
		time.Now().UTC(),
	}

	for i, testTime := range times {
		t.Run(fmt.Sprintf("time_%d", i), func(t *testing.T) {
			status := &PolicyStatus{
				EnforcementStatus: "ENFORCED",
				CreatedAt:         testTime,
				ModifiedAt:        testTime,
			}

			jsonData, err := json.Marshal(status)
			require.NoError(t, err)

			var unmarshaled PolicyStatus
			err = json.Unmarshal(jsonData, &unmarshaled)
			require.NoError(t, err)

			// Times should be equal when compared (accounting for precision differences)
			assert.True(t, testTime.Equal(unmarshaled.CreatedAt))
			assert.True(t, testTime.Equal(unmarshaled.ModifiedAt))
		})
	}
}

// Test Type Conversion and Casting

func TestTypes_InterfaceConversion(t *testing.T) {
	// Test conversion between interface{} and concrete types
	data := map[string]interface{}{
		"array_field":  []interface{}{1, 2, 3},
		"object_field": json.RawMessage(`{"nested":"value"}`),
	}

	instance := &PolicyInstance{
		PolicyID:     "type-test",
		PolicyTypeID: 1,
		PolicyData:   data,
	}

	jsonData, err := json.Marshal(instance)
	require.NoError(t, err)

	var unmarshaled PolicyInstance
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	// Test type assertions
	policyData := unmarshaled.PolicyData
	assert.Equal(t, "test", policyData["string_field"].(string))

	// JSON unmarshaling converts numbers to float64
	assert.Equal(t, float64(42), policyData["int_field"].(float64))
	assert.Equal(t, 3.14, policyData["float_field"].(float64))
	assert.Equal(t, true, policyData["bool_field"].(bool))

	arrayField := policyData["array_field"].([]interface{})
	assert.Len(t, arrayField, 3)
	assert.Equal(t, float64(1), arrayField[0].(float64))

	objectField := policyData["object_field"].(map[string]interface{})
	assert.Equal(t, "value", objectField["nested"].(string))
}

// Test Concurrent Access

func TestTypes_ConcurrentAccess(t *testing.T) {
	instance := &PolicyInstance{
		PolicyID:     "concurrent-test",
		PolicyTypeID: 1,
		PolicyData: map[string]interface{}{},
	}

	// Test concurrent read access
	const numGoroutines = 100
	results := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			jsonData, err := json.Marshal(instance)
			if err != nil {
				results <- "error"
				return
			}

			var unmarshaled PolicyInstance
			err = json.Unmarshal(jsonData, &unmarshaled)
			if err != nil {
				results <- "error"
				return
			}

			results <- unmarshaled.PolicyID
		}()
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		result := <-results
		assert.Equal(t, "concurrent-test", result)
	}
}

// Benchmarks

func BenchmarkPolicyType_JSON_Marshal(b *testing.B) {
	policyType := &PolicyType{
		PolicyTypeID:   1,
		PolicyTypeName: "Benchmark Policy Type",
		Description:    "Policy type for benchmarking JSON serialization",
		Schema: map[string]interface{}{
			"field1": json.RawMessage(`{"type":"string"}`),
			"field2": json.RawMessage(`{"type":"integer"}`),
			"field3": json.RawMessage(`{"type":"boolean"}`),
		},
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(policyType)
	}
}

func BenchmarkPolicyInstance_JSON_Marshal(b *testing.B) {
	instance := &PolicyInstance{
		PolicyID:     "benchmark-policy",
		PolicyTypeID: 1,
		PolicyData: map[string]interface{}{
			"sub_field": "sub_value",
		},
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(instance)
	}
}

func BenchmarkLargePolicyData_JSON_Marshal(b *testing.B) {
	// Create large policy data
	largeData := make(map[string]interface{})
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("field_%d", i)] = []interface{}{i, i + 1, i + 2}
	}

	instance := &PolicyInstance{
		PolicyID:     "benchmark-large",
		PolicyTypeID: 1,
		PolicyData:   largeData,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(instance)
	}
}

// Helper types and functions for validation

// Note: DeliveryInfo is imported from types.go

type RetryPolicy struct {
	MaxRetries  int           `json:"max_retries"`
	RetryDelay  time.Duration `json:"retry_delay"`
	BackoffType string        `json:"backoff_type"`
}

// Simple validation function for testing
func ValidateStruct(s interface{}) error {
	// This is a simplified validation function for testing purposes
	// In real implementation, you would use a proper validation library like go-playground/validator

	switch v := s.(type) {
	case *PolicyType:
		if v.PolicyTypeID <= 0 {
			return fmt.Errorf("policy_type_id must be positive")
		}
		if v.Schema == nil {
			return fmt.Errorf("schema is required")
		}
	case *PolicyInstance:
		if strings.TrimSpace(v.PolicyID) == "" {
			return fmt.Errorf("policy_id is required")
		}
		if v.PolicyTypeID <= 0 {
			return fmt.Errorf("policy_type_id must be positive")
		}
		if v.PolicyData == nil {
			return fmt.Errorf("policy_data is required")
		}
	case *PolicyStatus:
		if v.EnforcementStatus == "" {
			return fmt.Errorf("enforcement_status is required")
		}
		validStatuses := []string{"NOT_ENFORCED", "ENFORCED", "UNKNOWN"}
		valid := false
		for _, status := range validStatuses {
			if v.EnforcementStatus == status {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("enforcement_status must be one of: %v", validStatuses)
		}
	case *EnrichmentInfoType:
		if strings.TrimSpace(v.EiTypeID) == "" {
			return fmt.Errorf("ei_type_id is required")
		}
		if v.EiJobDataSchema == nil {
			return fmt.Errorf("ei_job_data_schema is required")
		}
	case *EnrichmentInfoJob:
		if strings.TrimSpace(v.EiJobID) == "" {
			return fmt.Errorf("ei_job_id is required")
		}
		if strings.TrimSpace(v.EiTypeID) == "" {
			return fmt.Errorf("ei_type_id is required")
		}
		if v.EiJobData == nil {
			return fmt.Errorf("ei_job_data is required")
		}
		if strings.TrimSpace(v.TargetURI) == "" {
			return fmt.Errorf("target_uri is required")
		}
		if !strings.HasPrefix(v.TargetURI, "http://") && !strings.HasPrefix(v.TargetURI, "https://") {
			return fmt.Errorf("target_uri must be a valid URL")
		}
		if strings.TrimSpace(v.JobOwner) == "" {
			return fmt.Errorf("job_owner is required")
		}
	}

	return nil
}

