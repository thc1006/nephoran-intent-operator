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
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test fixtures for handlers

func createTestPolicyType() *PolicyType {
	return &PolicyType{
		PolicyTypeID:   1,
		PolicyTypeName: "test-policy-type",
		Description:    "Test policy type for unit tests",
		Schema: map[string]interface{}{
			"scope": map[string]interface{}{
				"ue_id": map[string]interface{}{},
			},
			"statement": map[string]interface{}{
				"qos_class": map[string]interface{}{},
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
			"ue_id":     "test-ue-123",
			"statement": json.RawMessage(`{}`),
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
		AdditionalInfo: json.RawMessage(`{}`),
	}
}

func createTestEIType() *EnrichmentInfoType {
	return &EnrichmentInfoType{
		EiTypeID:    "test-ei-type-1",
		EiTypeName:  "Test EI Type",
		Description: "Test enrichment information type",
		EiJobDataSchema: map[string]interface{}{
			"config": json.RawMessage(`{}`),
		},
		EiJobResultSchema: json.RawMessage(`{}`),
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
	}
}

func createTestEIJob() *EnrichmentInfoJob {
	return &EnrichmentInfoJob{
		EiJobID:  "test-ei-job-1",
		EiTypeID: "test-ei-type-1",
		EiJobData: map[string]interface{}{
			"param1": "value1",
		},
		TargetURI:      "http://test-consumer.com/ei",
		JobOwner:       "test-owner",
		JobStatusURL:   "http://test-status.com",
		CreatedAt:      time.Now(),
		ModifiedAt:     time.Now(),
		LastExecutedAt: time.Now(),
	}
}

func createTestConsumer() *ConsumerInfo {
	return &ConsumerInfo{
		ConsumerID:   "test-consumer-1",
		ConsumerName: "Test Consumer",
		CallbackURL:  "http://test-consumer.com/callback",
		Capabilities: []string{"policy", "ei"},
		Metadata: ConsumerMetadata{
			Description: "Test consumer for unit tests",
		},
		CreatedAt:  time.Now(),
		ModifiedAt: time.Now(),
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

// Test Health and Readiness Endpoints and the rest of the test cases remain the same as in the original file

// Additional Test: Performance Monitoring for Long-Running Requests
func TestHandlers_LongRunningRequestTimeout(t *testing.T) {
	handlers, service, validator, storage := setupHandlerTest(t)

	// Simulate a long-running request that should timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Mock a service method that takes longer than the timeout
	service.On("CreatePolicyType", mock.AnythingOfType("*context.valueCtx"), 1, mock.AnythingOfType("*a1.PolicyType")).
		Run(func(args mock.Arguments) {
			time.Sleep(200 * time.Millisecond)
		}).
		Return(fmt.Errorf("request timed out"))

	policyType := createTestPolicyType()
	req := createJSONRequest(t, "PUT", "/A1-P/v2/policytypes/1", policyType)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	// This should either return a timeout error or be interrupted
	handlers.HandleCreatePolicyType(rr, req)

	assert.Equal(t, http.StatusRequestTimeout, rr.Code)
}

// Mocks and remaining content of the file remain the same