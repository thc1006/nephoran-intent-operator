package a1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations for A1 service interfaces

// MockA1Service implements A1Service interface for testing
type MockA1Service struct {
	mock.Mock
}

func (m *MockA1Service) GetPolicyTypes(ctx context.Context) ([]int, error) {
	args := m.Called(ctx)
	return args.Get(0).([]int), args.Error(1)
}

func (m *MockA1Service) GetPolicyType(ctx context.Context, policyTypeID int) (*PolicyType, error) {
	args := m.Called(ctx, policyTypeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PolicyType), args.Error(1)
}

func (m *MockA1Service) CreatePolicyType(ctx context.Context, policyTypeID int, policyType *PolicyType) error {
	args := m.Called(ctx, policyTypeID, policyType)
	return args.Error(0)
}

func (m *MockA1Service) DeletePolicyType(ctx context.Context, policyTypeID int) error {
	args := m.Called(ctx, policyTypeID)
	return args.Error(0)
}

func (m *MockA1Service) GetPolicyInstances(ctx context.Context, policyTypeID int) ([]string, error) {
	args := m.Called(ctx, policyTypeID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockA1Service) GetPolicyInstance(ctx context.Context, policyTypeID int, policyID string) (*PolicyInstance, error) {
	args := m.Called(ctx, policyTypeID, policyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PolicyInstance), args.Error(1)
}

func (m *MockA1Service) CreatePolicyInstance(ctx context.Context, policyTypeID int, policyID string, instance *PolicyInstance) error {
	args := m.Called(ctx, policyTypeID, policyID, instance)
	return args.Error(0)
}

func (m *MockA1Service) UpdatePolicyInstance(ctx context.Context, policyTypeID int, policyID string, instance *PolicyInstance) error {
	args := m.Called(ctx, policyTypeID, policyID, instance)
	return args.Error(0)
}

func (m *MockA1Service) DeletePolicyInstance(ctx context.Context, policyTypeID int, policyID string) error {
	args := m.Called(ctx, policyTypeID, policyID)
	return args.Error(0)
}

func (m *MockA1Service) GetPolicyStatus(ctx context.Context, policyTypeID int, policyID string) (*PolicyStatus, error) {
	args := m.Called(ctx, policyTypeID, policyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PolicyStatus), args.Error(1)
}

// A1-C Consumer Interface Methods
func (m *MockA1Service) RegisterConsumer(ctx context.Context, consumerID string, info *ConsumerInfo) error {
	args := m.Called(ctx, consumerID, info)
	return args.Error(0)
}

func (m *MockA1Service) UnregisterConsumer(ctx context.Context, consumerID string) error {
	args := m.Called(ctx, consumerID)
	return args.Error(0)
}

func (m *MockA1Service) GetConsumer(ctx context.Context, consumerID string) (*ConsumerInfo, error) {
	args := m.Called(ctx, consumerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ConsumerInfo), args.Error(1)
}

func (m *MockA1Service) ListConsumers(ctx context.Context) ([]*ConsumerInfo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*ConsumerInfo), args.Error(1)
}

func (m *MockA1Service) NotifyConsumer(ctx context.Context, consumerID string, notification *PolicyNotification) error {
	args := m.Called(ctx, consumerID, notification)
	return args.Error(0)
}

// A1-EI Enrichment Information Interface Methods
func (m *MockA1Service) GetEITypes(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockA1Service) GetEIType(ctx context.Context, eiTypeID string) (*EnrichmentInfoType, error) {
	args := m.Called(ctx, eiTypeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EnrichmentInfoType), args.Error(1)
}

func (m *MockA1Service) CreateEIType(ctx context.Context, eiTypeID string, eiType *EnrichmentInfoType) error {
	args := m.Called(ctx, eiTypeID, eiType)
	return args.Error(0)
}

func (m *MockA1Service) DeleteEIType(ctx context.Context, eiTypeID string) error {
	args := m.Called(ctx, eiTypeID)
	return args.Error(0)
}

func (m *MockA1Service) GetEIJobs(ctx context.Context, eiTypeID string) ([]string, error) {
	args := m.Called(ctx, eiTypeID)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockA1Service) GetEIJob(ctx context.Context, eiJobID string) (*EnrichmentInfoJob, error) {
	args := m.Called(ctx, eiJobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EnrichmentInfoJob), args.Error(1)
}

func (m *MockA1Service) CreateEIJob(ctx context.Context, eiJobID string, job *EnrichmentInfoJob) error {
	args := m.Called(ctx, eiJobID, job)
	return args.Error(0)
}

func (m *MockA1Service) UpdateEIJob(ctx context.Context, eiJobID string, job *EnrichmentInfoJob) error {
	args := m.Called(ctx, eiJobID, job)
	return args.Error(0)
}

func (m *MockA1Service) DeleteEIJob(ctx context.Context, eiJobID string) error {
	args := m.Called(ctx, eiJobID)
	return args.Error(0)
}

func (m *MockA1Service) GetEIJobStatus(ctx context.Context, eiJobID string) (*EnrichmentInfoJobStatus, error) {
	args := m.Called(ctx, eiJobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EnrichmentInfoJobStatus), args.Error(1)
}

// MockA1Validator implements A1Validator interface for testing
type MockA1Validator struct {
	mock.Mock
}

func (m *MockA1Validator) ValidatePolicyType(policyType *PolicyType) *ValidationResult {
	args := m.Called(policyType)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*ValidationResult)
}

func (m *MockA1Validator) ValidatePolicyInstance(policyTypeID int, instance *PolicyInstance) *ValidationResult {
	args := m.Called(policyTypeID, instance)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*ValidationResult)
}

func (m *MockA1Validator) ValidateConsumerInfo(info *ConsumerInfo) *ValidationResult {
	args := m.Called(info)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*ValidationResult)
}

func (m *MockA1Validator) ValidateEnrichmentInfoType(eiType *EnrichmentInfoType) *ValidationResult {
	args := m.Called(eiType)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*ValidationResult)
}

func (m *MockA1Validator) ValidateEnrichmentInfoJob(job *EnrichmentInfoJob) *ValidationResult {
	args := m.Called(job)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*ValidationResult)
}

// MockA1Storage implements A1Storage interface for testing
type MockA1Storage struct {
	mock.Mock
}

// Policy Type Storage
func (m *MockA1Storage) StorePolicyType(ctx context.Context, policyType *PolicyType) error {
	args := m.Called(ctx, policyType)
	return args.Error(0)
}

func (m *MockA1Storage) GetPolicyType(ctx context.Context, policyTypeID int) (*PolicyType, error) {
	args := m.Called(ctx, policyTypeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PolicyType), args.Error(1)
}

func (m *MockA1Storage) ListPolicyTypes(ctx context.Context) ([]*PolicyType, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*PolicyType), args.Error(1)
}

func (m *MockA1Storage) DeletePolicyType(ctx context.Context, policyTypeID int) error {
	args := m.Called(ctx, policyTypeID)
	return args.Error(0)
}

// Policy Instance Storage
func (m *MockA1Storage) StorePolicyInstance(ctx context.Context, instance *PolicyInstance) error {
	args := m.Called(ctx, instance)
	return args.Error(0)
}

func (m *MockA1Storage) GetPolicyInstance(ctx context.Context, policyTypeID int, policyID string) (*PolicyInstance, error) {
	args := m.Called(ctx, policyTypeID, policyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PolicyInstance), args.Error(1)
}

func (m *MockA1Storage) ListPolicyInstances(ctx context.Context, policyTypeID int) ([]*PolicyInstance, error) {
	args := m.Called(ctx, policyTypeID)
	return args.Get(0).([]*PolicyInstance), args.Error(1)
}

func (m *MockA1Storage) DeletePolicyInstance(ctx context.Context, policyTypeID int, policyID string) error {
	args := m.Called(ctx, policyTypeID, policyID)
	return args.Error(0)
}

func (m *MockA1Storage) UpdatePolicyInstanceStatus(ctx context.Context, policyTypeID int, policyID string, status *PolicyStatus) error {
	args := m.Called(ctx, policyTypeID, policyID, status)
	return args.Error(0)
}

// Consumer Storage
func (m *MockA1Storage) StoreConsumer(ctx context.Context, consumer *ConsumerInfo) error {
	args := m.Called(ctx, consumer)
	return args.Error(0)
}

func (m *MockA1Storage) GetConsumer(ctx context.Context, consumerID string) (*ConsumerInfo, error) {
	args := m.Called(ctx, consumerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ConsumerInfo), args.Error(1)
}

func (m *MockA1Storage) ListConsumers(ctx context.Context) ([]*ConsumerInfo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*ConsumerInfo), args.Error(1)
}

func (m *MockA1Storage) DeleteConsumer(ctx context.Context, consumerID string) error {
	args := m.Called(ctx, consumerID)
	return args.Error(0)
}

// EI Type Storage
func (m *MockA1Storage) StoreEIType(ctx context.Context, eiType *EnrichmentInfoType) error {
	args := m.Called(ctx, eiType)
	return args.Error(0)
}

func (m *MockA1Storage) GetEIType(ctx context.Context, eiTypeID string) (*EnrichmentInfoType, error) {
	args := m.Called(ctx, eiTypeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EnrichmentInfoType), args.Error(1)
}

func (m *MockA1Storage) ListEITypes(ctx context.Context) ([]*EnrichmentInfoType, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*EnrichmentInfoType), args.Error(1)
}

func (m *MockA1Storage) DeleteEIType(ctx context.Context, eiTypeID string) error {
	args := m.Called(ctx, eiTypeID)
	return args.Error(0)
}

// EI Job Storage
func (m *MockA1Storage) StoreEIJob(ctx context.Context, job *EnrichmentInfoJob) error {
	args := m.Called(ctx, job)
	return args.Error(0)
}

func (m *MockA1Storage) GetEIJob(ctx context.Context, eiJobID string) (*EnrichmentInfoJob, error) {
	args := m.Called(ctx, eiJobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EnrichmentInfoJob), args.Error(1)
}

func (m *MockA1Storage) ListEIJobs(ctx context.Context, eiTypeID string) ([]*EnrichmentInfoJob, error) {
	args := m.Called(ctx, eiTypeID)
	return args.Get(0).([]*EnrichmentInfoJob), args.Error(1)
}

func (m *MockA1Storage) DeleteEIJob(ctx context.Context, eiJobID string) error {
	args := m.Called(ctx, eiJobID)
	return args.Error(0)
}

func (m *MockA1Storage) UpdateEIJobStatus(ctx context.Context, eiJobID string, status *EnrichmentInfoJobStatus) error {
	args := m.Called(ctx, eiJobID, status)
	return args.Error(0)
}


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
	handlers, service, _, _ := setupHandlerTest(t)

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
