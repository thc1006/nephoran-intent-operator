// Package a1 provides comprehensive unit tests for A1 HTTP handlers
package a1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// Test fixtures for handlers

func createTestPolicyType() *PolicyType {
	return &PolicyType{
		PolicyTypeID:   1,
		PolicyTypeName: "test-policy-type",
		Description:    "Test policy type for unit tests",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"scope": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"ue_id": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"statement": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"qos_class": map[string]interface{}{
							"type": "integer",
						},
					},
				},
			},
		},
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
	}
}

func createTestPolicyInstance() *PolicyInstance {
	return &PolicyInstance{
		PolicyID:     "test-policy-1",
		PolicyTypeID: 1,
		PolicyData: map[string]interface{}{
			"scope": map[string]interface{}{
				"ue_id": "test-ue-123",
			},
			"statement": map[string]interface{}{
				"qos_class": 5,
			},
		},
		PolicyInfo: PolicyInstanceInfo{
			NotificationDestination: "http://test-callback.com",
			RequestID:               "req-123",
		},
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
	}
}

func createTestPolicyStatus() *PolicyStatus {
	return &PolicyStatus{
		EnforcementStatus: "ENFORCED",
		EnforcementReason: "Policy successfully applied",
		HasBeenDeleted:    false,
		Deleted:           false,
		CreatedAt:         time.Now(),
		ModifiedAt:        time.Now(),
		AdditionalInfo: map[string]interface{}{
			"enforcement_node": "test-ric",
		},
	}
}

func createTestEIType() *EnrichmentInfoType {
	return &EnrichmentInfoType{
		EiTypeID:    "test-ei-type-1",
		EiTypeName:  "Test EI Type",
		Description: "Test enrichment information type",
		EiJobDataSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"config": map[string]interface{}{
					"type": "object",
				},
			},
		},
		EiJobResultSchema: map[string]interface{}{
			"type": "object",
		},
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
	}
}

func createTestEIJob() *EnrichmentInfoJob {
	return &EnrichmentInfoJob{
		EiJobID:  "test-ei-job-1",
		EiTypeID: "test-ei-type-1",
		EiJobData: map[string]interface{}{
			"config": map[string]interface{}{
				"param1": "value1",
			},
		},
		TargetURI:      "http://test-consumer.com/ei",
		JobOwner:       "test-owner",
		JobStatusURL:   "http://test-status.com",
		CreatedAt:      time.Now(),
		ModifiedAt:     time.Now(),
		LastExecutedAt: time.Now(),
	}
}

func createTestConsumer() *Consumer {
	return &Consumer{
		ConsumerID:      "test-consumer-1",
		ConsumerName:    "Test Consumer",
		CallbackURL:     "http://test-consumer.com/callback",
		Description:     "Test consumer for unit tests",
		RegisteredAt:    time.Now(),
		LastActiveAt:    time.Now(),
		SubscribedTypes: []string{"policy", "ei"},
	}
}

func setupHandlerTest(t *testing.T) (*A1Handlers, *MockA1Service, *MockA1Validator, *MockA1Storage) {
	service := &MockA1Service{}
	validator := &MockA1Validator{}
	storage := &MockA1Storage{}

	logger := createTestLogger()
	config := createTestConfig()
	metrics := &noopMetrics{}

	handlers := &A1Handlers{
		service:   service,
		validator: validator,
		storage:   storage,
		metrics:   metrics,
		logger:    logger,
		config:    config,
	}

	return handlers, service, validator, storage
}

func createJSONRequest(t *testing.T, method, path string, body interface{}) *http.Request {
	var reader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, path, reader)
	require.NoError(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req
}

// Test Health and Readiness Endpoints

func TestHealthCheckHandler(t *testing.T) {
	handlers, _, _, _ := setupHandlerTest(t)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handlers.HealthCheckHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "healthy")
}

func TestReadinessCheckHandler(t *testing.T) {
	handlers, _, _, _ := setupHandlerTest(t)

	req := httptest.NewRequest("GET", "/ready", nil)
	rr := httptest.NewRecorder()

	handlers.ReadinessCheckHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "ready")
}

// Test A1-P Policy Interface Handlers

func TestHandleGetPolicyTypes(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockA1Storage)
		expectedStatus int
		expectedBody   []int
		expectError    bool
	}{
		{
			name: "successful get policy types",
			setupMocks: func(storage *MockA1Storage) {
				storage.On("GetPolicyTypes", mock.AnythingOfType("*context.valueCtx")).Return([]int{1, 2, 3}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []int{1, 2, 3},
		},
		{
			name: "storage error",
			setupMocks: func(storage *MockA1Storage) {
				storage.On("GetPolicyTypes", mock.AnythingOfType("*context.valueCtx")).Return([]int{}, fmt.Errorf("storage error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name: "empty policy types",
			setupMocks: func(storage *MockA1Storage) {
				storage.On("GetPolicyTypes", mock.AnythingOfType("*context.valueCtx")).Return([]int{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, _, _, storage := setupHandlerTest(t)
			tt.setupMocks(storage)

			req := httptest.NewRequest("GET", "/A1-P/v2/policytypes", nil)
			rr := httptest.NewRecorder()

			handlers.HandleGetPolicyTypes(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if !tt.expectError {
				var response []int
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
			}

			storage.AssertExpectations(t)
		})
	}
}

func TestHandleGetPolicyType(t *testing.T) {
	tests := []struct {
		name           string
		policyTypeID   string
		setupMocks     func(*MockA1Storage)
		expectedStatus int
		expectError    bool
	}{
		{
			name:         "successful get policy type",
			policyTypeID: "1",
			setupMocks: func(storage *MockA1Storage) {
				policyType := createTestPolicyType()
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return(policyType, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "policy type not found",
			policyTypeID: "999",
			setupMocks: func(storage *MockA1Storage) {
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 999).Return((*PolicyType)(nil), NewA1Error(ErrorTypePolicyTypeNotFound, "Policy type not found", http.StatusNotFound, nil))
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:         "invalid policy type ID",
			policyTypeID: "invalid",
			setupMocks: func(storage *MockA1Storage) {
				// No mock call expected for invalid ID
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, _, _, storage := setupHandlerTest(t)
			tt.setupMocks(storage)

			req := httptest.NewRequest("GET", fmt.Sprintf("/A1-P/v2/policytypes/%s", tt.policyTypeID), nil)
			req = mux.SetURLVars(req, map[string]string{"policy_type_id": tt.policyTypeID})
			rr := httptest.NewRecorder()

			handlers.HandleGetPolicyType(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if !tt.expectError && tt.expectedStatus == http.StatusOK {
				var response PolicyType
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, 1, response.PolicyTypeID)
			}

			storage.AssertExpectations(t)
		})
	}
}

func TestHandleCreatePolicyType(t *testing.T) {
	tests := []struct {
		name           string
		policyTypeID   string
		requestBody    interface{}
		setupMocks     func(*MockA1Service, *MockA1Validator, *MockA1Storage)
		expectedStatus int
		expectError    bool
	}{
		{
			name:         "successful create policy type",
			policyTypeID: "1",
			requestBody:  createTestPolicyType(),
			setupMocks: func(service *MockA1Service, validator *MockA1Validator, storage *MockA1Storage) {
				// Check if policy type already exists
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return((*PolicyType)(nil), NewA1Error(ErrorTypePolicyTypeNotFound, "Not found", http.StatusNotFound, nil))

				// Validate policy type
				validator.On("ValidatePolicyType", mock.AnythingOfType("*a1.PolicyType")).Return(nil)

				// Create policy type
				service.On("CreatePolicyType", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*a1.PolicyType")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:         "policy type already exists",
			policyTypeID: "1",
			requestBody:  createTestPolicyType(),
			setupMocks: func(service *MockA1Service, validator *MockA1Validator, storage *MockA1Storage) {
				existingPolicyType := createTestPolicyType()
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return(existingPolicyType, nil)
			},
			expectedStatus: http.StatusConflict,
			expectError:    true,
		},
		{
			name:           "invalid JSON",
			policyTypeID:   "1",
			requestBody:    "invalid-json",
			setupMocks:     func(*MockA1Service, *MockA1Validator, *MockA1Storage) {},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:         "validation failed",
			policyTypeID: "1",
			requestBody:  createTestPolicyType(),
			setupMocks: func(service *MockA1Service, validator *MockA1Validator, storage *MockA1Storage) {
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return((*PolicyType)(nil), NewA1Error(ErrorTypePolicyTypeNotFound, "Not found", http.StatusNotFound, nil))
				validator.On("ValidatePolicyType", mock.AnythingOfType("*a1.PolicyType")).Return(fmt.Errorf("validation failed"))
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, service, validator, storage := setupHandlerTest(t)
			tt.setupMocks(service, validator, storage)

			req := createJSONRequest(t, "PUT", fmt.Sprintf("/A1-P/v2/policytypes/%s", tt.policyTypeID), tt.requestBody)
			req = mux.SetURLVars(req, map[string]string{"policy_type_id": tt.policyTypeID})
			rr := httptest.NewRecorder()

			handlers.HandleCreatePolicyType(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusCreated {
				assert.Contains(t, rr.Header().Get("Location"), fmt.Sprintf("/A1-P/v2/policytypes/%s", tt.policyTypeID))
			}

			service.AssertExpectations(t)
			validator.AssertExpectations(t)
			storage.AssertExpectations(t)
		})
	}
}

func TestHandleDeletePolicyType(t *testing.T) {
	tests := []struct {
		name           string
		policyTypeID   string
		setupMocks     func(*MockA1Service, *MockA1Storage)
		expectedStatus int
		expectError    bool
	}{
		{
			name:         "successful delete policy type",
			policyTypeID: "1",
			setupMocks: func(service *MockA1Service, storage *MockA1Storage) {
				// Check policy type exists
				policyType := createTestPolicyType()
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return(policyType, nil)

				// Check no active policy instances
				storage.On("GetPolicyInstances", mock.AnythingOfType("*context.valueCtx"), 1).Return([]string{}, nil)

				// Delete policy type
				service.On("DeletePolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:         "policy type not found",
			policyTypeID: "999",
			setupMocks: func(service *MockA1Service, storage *MockA1Storage) {
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 999).Return((*PolicyType)(nil), NewA1Error(ErrorTypePolicyTypeNotFound, "Not found", http.StatusNotFound, nil))
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:         "policy type has active instances",
			policyTypeID: "1",
			setupMocks: func(service *MockA1Service, storage *MockA1Storage) {
				policyType := createTestPolicyType()
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return(policyType, nil)
				storage.On("GetPolicyInstances", mock.AnythingOfType("*context.valueCtx"), 1).Return([]string{"policy-1", "policy-2"}, nil)
			},
			expectedStatus: http.StatusConflict,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, service, _, storage := setupHandlerTest(t)
			tt.setupMocks(service, storage)

			req := httptest.NewRequest("DELETE", fmt.Sprintf("/A1-P/v2/policytypes/%s", tt.policyTypeID), nil)
			req = mux.SetURLVars(req, map[string]string{"policy_type_id": tt.policyTypeID})
			rr := httptest.NewRecorder()

			handlers.HandleDeletePolicyType(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			service.AssertExpectations(t)
			storage.AssertExpectations(t)
		})
	}
}

func TestHandleGetPolicyInstances(t *testing.T) {
	tests := []struct {
		name           string
		policyTypeID   string
		setupMocks     func(*MockA1Storage)
		expectedStatus int
		expectedBody   []string
		expectError    bool
	}{
		{
			name:         "successful get policy instances",
			policyTypeID: "1",
			setupMocks: func(storage *MockA1Storage) {
				// Check policy type exists
				policyType := createTestPolicyType()
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return(policyType, nil)

				// Get policy instances
				storage.On("GetPolicyInstances", mock.AnythingOfType("*context.valueCtx"), 1).Return([]string{"policy-1", "policy-2"}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []string{"policy-1", "policy-2"},
		},
		{
			name:         "policy type not found",
			policyTypeID: "999",
			setupMocks: func(storage *MockA1Storage) {
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 999).Return((*PolicyType)(nil), NewA1Error(ErrorTypePolicyTypeNotFound, "Not found", http.StatusNotFound, nil))
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:         "empty policy instances",
			policyTypeID: "1",
			setupMocks: func(storage *MockA1Storage) {
				policyType := createTestPolicyType()
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return(policyType, nil)
				storage.On("GetPolicyInstances", mock.AnythingOfType("*context.valueCtx"), 1).Return([]string{}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, _, _, storage := setupHandlerTest(t)
			tt.setupMocks(storage)

			req := httptest.NewRequest("GET", fmt.Sprintf("/A1-P/v2/policytypes/%s/policies", tt.policyTypeID), nil)
			req = mux.SetURLVars(req, map[string]string{"policy_type_id": tt.policyTypeID})
			rr := httptest.NewRecorder()

			handlers.HandleGetPolicyInstances(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if !tt.expectError {
				var response []string
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedBody, response)
			}

			storage.AssertExpectations(t)
		})
	}
}

func TestHandleCreatePolicyInstance(t *testing.T) {
	tests := []struct {
		name           string
		policyTypeID   string
		policyID       string
		requestBody    interface{}
		setupMocks     func(*MockA1Service, *MockA1Validator, *MockA1Storage)
		expectedStatus int
		expectError    bool
	}{
		{
			name:         "successful create policy instance",
			policyTypeID: "1",
			policyID:     "test-policy-1",
			requestBody:  createTestPolicyInstance().PolicyData,
			setupMocks: func(service *MockA1Service, validator *MockA1Validator, storage *MockA1Storage) {
				// Check policy type exists
				policyType := createTestPolicyType()
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return(policyType, nil)

				// Check instance doesn't exist
				storage.On("GetPolicyInstance", mock.AnythingOfType("*context.valueCtx"), 1, "test-policy-1").Return((*PolicyInstance)(nil), NewA1Error(ErrorTypePolicyInstanceNotFound, "Not found", http.StatusNotFound, nil))

				// Validate instance
				validator.On("ValidatePolicyInstance", mock.AnythingOfType("*a1.PolicyType"), mock.AnythingOfType("*a1.PolicyInstance")).Return(nil)

				// Create instance
				service.On("CreatePolicyInstance", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*a1.PolicyInstance")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:         "policy type not found",
			policyTypeID: "999",
			policyID:     "test-policy-1",
			requestBody:  createTestPolicyInstance().PolicyData,
			setupMocks: func(service *MockA1Service, validator *MockA1Validator, storage *MockA1Storage) {
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 999).Return((*PolicyType)(nil), NewA1Error(ErrorTypePolicyTypeNotFound, "Not found", http.StatusNotFound, nil))
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:         "policy instance already exists",
			policyTypeID: "1",
			policyID:     "test-policy-1",
			requestBody:  createTestPolicyInstance().PolicyData,
			setupMocks: func(service *MockA1Service, validator *MockA1Validator, storage *MockA1Storage) {
				policyType := createTestPolicyType()
				existingInstance := createTestPolicyInstance()
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return(policyType, nil)
				storage.On("GetPolicyInstance", mock.AnythingOfType("*context.valueCtx"), 1, "test-policy-1").Return(existingInstance, nil)
			},
			expectedStatus: http.StatusConflict,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, service, validator, storage := setupHandlerTest(t)
			tt.setupMocks(service, validator, storage)

			req := createJSONRequest(t, "PUT", fmt.Sprintf("/A1-P/v2/policytypes/%s/policies/%s", tt.policyTypeID, tt.policyID), tt.requestBody)
			req = mux.SetURLVars(req, map[string]string{"policy_type_id": tt.policyTypeID, "policy_id": tt.policyID})
			rr := httptest.NewRecorder()

			handlers.HandleCreatePolicyInstance(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusCreated {
				assert.Contains(t, rr.Header().Get("Location"), fmt.Sprintf("/A1-P/v2/policytypes/%s/policies/%s", tt.policyTypeID, tt.policyID))
			}

			service.AssertExpectations(t)
			validator.AssertExpectations(t)
			storage.AssertExpectations(t)
		})
	}
}

func TestHandleGetPolicyStatus(t *testing.T) {
	tests := []struct {
		name           string
		policyTypeID   string
		policyID       string
		setupMocks     func(*MockA1Storage)
		expectedStatus int
		expectError    bool
	}{
		{
			name:         "successful get policy status",
			policyTypeID: "1",
			policyID:     "test-policy-1",
			setupMocks: func(storage *MockA1Storage) {
				// Check policy type exists
				policyType := createTestPolicyType()
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return(policyType, nil)

				// Check policy instance exists
				policyInstance := createTestPolicyInstance()
				storage.On("GetPolicyInstance", mock.AnythingOfType("*context.valueCtx"), 1, "test-policy-1").Return(policyInstance, nil)

				// Get policy status
				policyStatus := createTestPolicyStatus()
				storage.On("GetPolicyStatus", mock.AnythingOfType("*context.valueCtx"), 1, "test-policy-1").Return(policyStatus, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:         "policy instance not found",
			policyTypeID: "1",
			policyID:     "non-existent-policy",
			setupMocks: func(storage *MockA1Storage) {
				policyType := createTestPolicyType()
				storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return(policyType, nil)
				storage.On("GetPolicyInstance", mock.AnythingOfType("*context.valueCtx"), 1, "non-existent-policy").Return((*PolicyInstance)(nil), NewA1Error(ErrorTypePolicyInstanceNotFound, "Not found", http.StatusNotFound, nil))
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, _, _, storage := setupHandlerTest(t)
			tt.setupMocks(storage)

			req := httptest.NewRequest("GET", fmt.Sprintf("/A1-P/v2/policytypes/%s/policies/%s/status", tt.policyTypeID, tt.policyID), nil)
			req = mux.SetURLVars(req, map[string]string{"policy_type_id": tt.policyTypeID, "policy_id": tt.policyID})
			rr := httptest.NewRecorder()

			handlers.HandleGetPolicyStatus(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if !tt.expectError && tt.expectedStatus == http.StatusOK {
				var response PolicyStatus
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, "ENFORCED", response.EnforcementStatus)
			}

			storage.AssertExpectations(t)
		})
	}
}

// Test A1-C Consumer Interface Handlers

func TestHandleListConsumers(t *testing.T) {
	handlers, _, _, storage := setupHandlerTest(t)

	consumers := []Consumer{
		*createTestConsumer(),
		{
			ConsumerID:   "consumer-2",
			ConsumerName: "Test Consumer 2",
			CallbackURL:  "http://consumer2.com/callback",
		},
	}

	storage.On("GetConsumers", mock.AnythingOfType("*context.valueCtx")).Return(consumers, nil)

	req := httptest.NewRequest("GET", "/A1-C/v1/consumers", nil)
	rr := httptest.NewRecorder()

	handlers.HandleListConsumers(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response []Consumer
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Len(t, response, 2)
	assert.Equal(t, "test-consumer-1", response[0].ConsumerID)

	storage.AssertExpectations(t)
}

func TestHandleRegisterConsumer(t *testing.T) {
	tests := []struct {
		name           string
		consumerID     string
		requestBody    interface{}
		setupMocks     func(*MockA1Service, *MockA1Storage)
		expectedStatus int
		expectError    bool
	}{
		{
			name:       "successful register consumer",
			consumerID: "new-consumer",
			requestBody: map[string]interface{}{
				"consumer_name": "New Consumer",
				"callback_url":  "http://new-consumer.com/callback",
				"description":   "A new test consumer",
			},
			setupMocks: func(service *MockA1Service, storage *MockA1Storage) {
				// Check consumer doesn't exist
				storage.On("GetConsumer", mock.AnythingOfType("*context.valueCtx"), "new-consumer").Return((*Consumer)(nil), NewA1Error(ErrorTypeConsumerNotFound, "Not found", http.StatusNotFound, nil))

				// Register consumer
				service.On("RegisterConsumer", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*a1.Consumer")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:       "consumer already exists",
			consumerID: "existing-consumer",
			requestBody: map[string]interface{}{
				"consumer_name": "Existing Consumer",
				"callback_url":  "http://existing.com/callback",
			},
			setupMocks: func(service *MockA1Service, storage *MockA1Storage) {
				existingConsumer := createTestConsumer()
				existingConsumer.ConsumerID = "existing-consumer"
				storage.On("GetConsumer", mock.AnythingOfType("*context.valueCtx"), "existing-consumer").Return(existingConsumer, nil)
			},
			expectedStatus: http.StatusConflict,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, service, _, storage := setupHandlerTest(t)
			tt.setupMocks(service, storage)

			req := createJSONRequest(t, "POST", fmt.Sprintf("/A1-C/v1/consumers/%s", tt.consumerID), tt.requestBody)
			req = mux.SetURLVars(req, map[string]string{"consumer_id": tt.consumerID})
			rr := httptest.NewRecorder()

			handlers.HandleRegisterConsumer(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusCreated {
				assert.Contains(t, rr.Header().Get("Location"), fmt.Sprintf("/A1-C/v1/consumers/%s", tt.consumerID))
			}

			service.AssertExpectations(t)
			storage.AssertExpectations(t)
		})
	}
}

// Test A1-EI Enrichment Interface Handlers

func TestHandleGetEITypes(t *testing.T) {
	handlers, _, _, storage := setupHandlerTest(t)

	eiTypes := []string{"ei-type-1", "ei-type-2"}
	storage.On("GetEITypes", mock.AnythingOfType("*context.valueCtx")).Return(eiTypes, nil)

	req := httptest.NewRequest("GET", "/A1-EI/v1/eitypes", nil)
	rr := httptest.NewRecorder()

	handlers.HandleGetEITypes(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var response []string
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, eiTypes, response)

	storage.AssertExpectations(t)
}

func TestHandleCreateEIJob(t *testing.T) {
	tests := []struct {
		name           string
		eiJobID        string
		requestBody    interface{}
		setupMocks     func(*MockA1Service, *MockA1Validator, *MockA1Storage)
		expectedStatus int
		expectError    bool
	}{
		{
			name:    "successful create EI job",
			eiJobID: "new-ei-job",
			requestBody: map[string]interface{}{
				"ei_type_id": "test-ei-type-1",
				"ei_job_data": map[string]interface{}{
					"config": map[string]interface{}{
						"param1": "value1",
					},
				},
				"target_uri": "http://consumer.com/ei",
				"job_owner":  "test-owner",
			},
			setupMocks: func(service *MockA1Service, validator *MockA1Validator, storage *MockA1Storage) {
				// Check EI type exists
				eiType := createTestEIType()
				storage.On("GetEIType", mock.AnythingOfType("*context.valueCtx"), "test-ei-type-1").Return(eiType, nil)

				// Check job doesn't exist
				storage.On("GetEIJob", mock.AnythingOfType("*context.valueCtx"), "new-ei-job").Return((*EnrichmentInfoJob)(nil), NewA1Error(ErrorTypeEIJobNotFound, "Not found", http.StatusNotFound, nil))

				// Validate job
				validator.On("ValidateEIJob", mock.AnythingOfType("*a1.EnrichmentInfoType"), mock.AnythingOfType("*a1.EnrichmentInfoJob")).Return(nil)

				// Create job
				service.On("CreateEIJob", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*a1.EnrichmentInfoJob")).Return(nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:    "EI type not found",
			eiJobID: "new-ei-job",
			requestBody: map[string]interface{}{
				"ei_type_id":  "non-existent-type",
				"ei_job_data": map[string]interface{}{},
				"target_uri":  "http://consumer.com/ei",
				"job_owner":   "test-owner",
			},
			setupMocks: func(service *MockA1Service, validator *MockA1Validator, storage *MockA1Storage) {
				storage.On("GetEIType", mock.AnythingOfType("*context.valueCtx"), "non-existent-type").Return((*EnrichmentInfoType)(nil), NewA1Error(ErrorTypeEITypeNotFound, "Not found", http.StatusNotFound, nil))
			},
			expectedStatus: http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, service, validator, storage := setupHandlerTest(t)
			tt.setupMocks(service, validator, storage)

			req := createJSONRequest(t, "PUT", fmt.Sprintf("/A1-EI/v1/eijobs/%s", tt.eiJobID), tt.requestBody)
			req = mux.SetURLVars(req, map[string]string{"ei_job_id": tt.eiJobID})
			rr := httptest.NewRecorder()

			handlers.HandleCreateEIJob(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusCreated {
				assert.Contains(t, rr.Header().Get("Location"), fmt.Sprintf("/A1-EI/v1/eijobs/%s", tt.eiJobID))
			}

			service.AssertExpectations(t)
			validator.AssertExpectations(t)
			storage.AssertExpectations(t)
		})
	}
}

// Test Error Handling and Edge Cases

func TestHandlers_InvalidHTTPMethods(t *testing.T) {
	handlers, _, _, _ := setupHandlerTest(t)

	tests := []struct {
		name    string
		method  string
		path    string
		handler http.HandlerFunc
	}{
		{"GET on PUT endpoint", "GET", "/A1-P/v2/policytypes/1", handlers.HandleCreatePolicyType},
		{"POST on GET endpoint", "POST", "/A1-P/v2/policytypes", handlers.HandleGetPolicyTypes},
		{"DELETE on GET endpoint", "DELETE", "/A1-P/v2/policytypes/1/policies/test", handlers.HandleGetPolicyInstance},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			tt.handler(rr, req)

			assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
		})
	}
}

func TestHandlers_InvalidContentType(t *testing.T) {
	handlers, _, _, _ := setupHandlerTest(t)

	req := httptest.NewRequest("PUT", "/A1-P/v2/policytypes/1", strings.NewReader("test data"))
	req.Header.Set("Content-Type", "text/plain")
	req = mux.SetURLVars(req, map[string]string{"policy_type_id": "1"})
	rr := httptest.NewRecorder()

	handlers.HandleCreatePolicyType(rr, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
}

func TestHandlers_LargeRequestBody(t *testing.T) {
	handlers, _, _, _ := setupHandlerTest(t)

	// Create a large request body
	largeData := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeData[fmt.Sprintf("key_%d", i)] = strings.Repeat("value", 1000)
	}

	req := createJSONRequest(t, "PUT", "/A1-P/v2/policytypes/1", largeData)
	req = mux.SetURLVars(req, map[string]string{"policy_type_id": "1"})
	rr := httptest.NewRecorder()

	handlers.HandleCreatePolicyType(rr, req)

	// Should handle large requests gracefully
	assert.NotEqual(t, http.StatusInternalServerError, rr.Code)
}

func TestHandlers_ConcurrentRequests(t *testing.T) {
	handlers, _, _, storage := setupHandlerTest(t)

	// Setup mock to be called multiple times
	storage.On("GetPolicyTypes", mock.AnythingOfType("*context.valueCtx")).Return([]int{1, 2, 3}, nil).Times(10)

	// Create multiple concurrent requests
	var results []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest("GET", "/A1-P/v2/policytypes", nil)
			rr := httptest.NewRecorder()

			handlers.HandleGetPolicyTypes(rr, req)

			mu.Lock()
			results = append(results, rr.Code)
			mu.Unlock()
		}()
	}

	wg.Wait()

	// All requests should succeed
	assert.Len(t, results, 10)
	for _, code := range results {
		assert.Equal(t, http.StatusOK, code)
	}

	storage.AssertExpectations(t)
}

// Benchmarks for performance testing

func BenchmarkHandleGetPolicyTypes(b *testing.B) {
	handlers, _, _, storage := setupHandlerTest(&testing.T{})

	storage.On("GetPolicyTypes", mock.AnythingOfType("*context.valueCtx")).Return([]int{1, 2, 3}, nil)

	req := httptest.NewRequest("GET", "/A1-P/v2/policytypes", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.HandleGetPolicyTypes(rr, req)
	}
}

func BenchmarkHandleCreatePolicyType(b *testing.B) {
	handlers, service, validator, storage := setupHandlerTest(&testing.T{})

	policyType := createTestPolicyType()
	storage.On("GetPolicyType", mock.AnythingOfType("*context.valueCtx"), 1).Return((*PolicyType)(nil), NewA1Error(ErrorTypePolicyTypeNotFound, "Not found", http.StatusNotFound, nil))
	validator.On("ValidatePolicyType", mock.AnythingOfType("*a1.PolicyType")).Return(nil)
	service.On("CreatePolicyType", mock.AnythingOfType("*context.valueCtx"), mock.AnythingOfType("*a1.PolicyType")).Return(nil)

	req := createJSONRequest(&testing.T{}, "PUT", "/A1-P/v2/policytypes/1", policyType)
	req = mux.SetURLVars(req, map[string]string{"policy_type_id": "1"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.HandleCreatePolicyType(rr, req)
	}
}

// Additional type definitions needed for compilation

type Consumer struct {
	ConsumerID      string    `json:"consumer_id"`
	ConsumerName    string    `json:"consumer_name"`
	CallbackURL     string    `json:"callback_url"`
	Description     string    `json:"description,omitempty"`
	RegisteredAt    time.Time `json:"registered_at,omitempty"`
	LastActiveAt    time.Time `json:"last_active_at,omitempty"`
	SubscribedTypes []string  `json:"subscribed_types,omitempty"`
}

// Mock methods for additional storage operations
func (m *MockA1Storage) GetConsumers(ctx context.Context) ([]Consumer, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Consumer), args.Error(1)
}

func (m *MockA1Storage) GetConsumer(ctx context.Context, consumerID string) (*Consumer, error) {
	args := m.Called(ctx, consumerID)
	return args.Get(0).(*Consumer), args.Error(1)
}

func (m *MockA1Storage) GetEITypes(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockA1Storage) GetEIType(ctx context.Context, eiTypeID string) (*EnrichmentInfoType, error) {
	args := m.Called(ctx, eiTypeID)
	return args.Get(0).(*EnrichmentInfoType), args.Error(1)
}

func (m *MockA1Storage) GetEIJob(ctx context.Context, eiJobID string) (*EnrichmentInfoJob, error) {
	args := m.Called(ctx, eiJobID)
	return args.Get(0).(*EnrichmentInfoJob), args.Error(1)
}

// Mock service methods for additional operations
func (m *MockA1Service) RegisterConsumer(ctx context.Context, consumer *Consumer) error {
	args := m.Called(ctx, consumer)
	return args.Error(0)
}

func (m *MockA1Service) CreateEIJob(ctx context.Context, job *EnrichmentInfoJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

// Error type for proper error handling in tests
func NewHandlerTestA1Error(errorType A1ErrorType, message string, statusCode int, cause error) error {
	return &A1HandlerTestError{
		Type:     errorType,
		Title:    message,
		Status:   statusCode,
		Detail:   message,
		Instance: "",
		Cause:    cause,
	}
}

type A1HandlerTestError struct {
	Type     A1ErrorType `json:"type"`
	Title    string      `json:"title"`
	Status   int         `json:"status"`
	Detail   string      `json:"detail"`
	Instance string      `json:"instance"`
	Cause    error       `json:"-"`
}
