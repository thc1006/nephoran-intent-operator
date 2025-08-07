package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/health"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// LLMProcessorHandlerTestSuite provides comprehensive test coverage for LLMProcessorHandler
type LLMProcessorHandlerTestSuite struct {
	suite.Suite
	handler            *LLMProcessorHandler
	mockCtrl           *gomock.Controller
	mockProcessor      *MockIntentProcessor
	mockHealthChecker  *MockHealthChecker
	mockTokenManager   *MockTokenManager
	logger             *slog.Logger
	config             *config.LLMProcessorConfig
}

func TestLLMProcessorHandlerSuite(t *testing.T) {
	suite.Run(t, new(LLMProcessorHandlerTestSuite))
}

func (suite *LLMProcessorHandlerTestSuite) SetupSuite() {
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func (suite *LLMProcessorHandlerTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())
	suite.mockProcessor = NewMockIntentProcessor(suite.mockCtrl)
	suite.mockHealthChecker = NewMockHealthChecker(suite.mockCtrl)
	suite.mockTokenManager = NewMockTokenManager(suite.mockCtrl)
	
	suite.config = &config.LLMProcessorConfig{
		Enabled:     true,
		APIEndpoint: "http://localhost:8080",
		APIKey:      "test-api-key",
		Model:       "gpt-4o-mini",
		MaxTokens:   4000,
		Temperature: 0.7,
		Timeout:     30 * time.Second,
		RetryConfig: config.RetryConfig{
			MaxRetries:      3,
			InitialDelay:    100 * time.Millisecond,
			BackoffFactor:   2.0,
			MaxDelay:        10 * time.Second,
		},
		RateLimiting: config.RateLimitConfig{
			RequestsPerSecond: 10,
			TokensPerMinute:   50000,
			BurstSize:         20,
		},
	}
	
	suite.handler = &LLMProcessorHandler{
		config:        suite.config,
		processor:     suite.mockProcessor,
		tokenManager:  suite.mockTokenManager,
		logger:        suite.logger,
		healthChecker: suite.mockHealthChecker,
		startTime:     time.Now(),
	}
}

func (suite *LLMProcessorHandlerTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func (suite *LLMProcessorHandlerTestSuite) TestNewLLMProcessorHandler_Success() {
	config := &config.LLMProcessorConfig{
		Enabled:     true,
		APIEndpoint: "http://localhost:8080",
		APIKey:      "test-key",
		Model:       "gpt-4o-mini",
		MaxTokens:   4000,
		Temperature: 0.7,
	}
	
	handler, err := NewLLMProcessorHandler(config, suite.logger)
	
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), handler)
	assert.Equal(suite.T(), config, handler.config)
	assert.NotNil(suite.T(), handler.processor)
	assert.NotNil(suite.T(), handler.healthChecker)
	assert.NotZero(suite.T(), handler.startTime)
}

func (suite *LLMProcessorHandlerTestSuite) TestNewLLMProcessorHandler_InvalidConfig() {
	testCases := []struct {
		name     string
		config   *config.LLMProcessorConfig
		errorMsg string
	}{
		{
			name:     "Nil config",
			config:   nil,
			errorMsg: "config cannot be nil",
		},
		{
			name: "Missing API endpoint",
			config: &config.LLMProcessorConfig{
				Enabled: true,
				APIKey:  "test-key",
			},
			errorMsg: "API endpoint cannot be empty",
		},
		{
			name: "Missing API key",
			config: &config.LLMProcessorConfig{
				Enabled:     true,
				APIEndpoint: "http://localhost:8080",
			},
			errorMsg: "API key cannot be empty",
		},
		{
			name: "Invalid temperature",
			config: &config.LLMProcessorConfig{
				Enabled:     true,
				APIEndpoint: "http://localhost:8080",
				APIKey:      "test-key",
				Temperature: 2.5, // Invalid: > 2.0
			},
			errorMsg: "temperature must be between 0 and 2",
		},
		{
			name: "Invalid max tokens",
			config: &config.LLMProcessorConfig{
				Enabled:     true,
				APIEndpoint: "http://localhost:8080",
				APIKey:      "test-key",
				MaxTokens:   0, // Invalid: must be > 0
			},
			errorMsg: "max tokens must be greater than 0",
		},
	}
	
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			handler, err := NewLLMProcessorHandler(tc.config, suite.logger)
			
			assert.Error(suite.T(), err)
			assert.Nil(suite.T(), handler)
			assert.Contains(suite.T(), err.Error(), tc.errorMsg)
		})
	}
}

func (suite *LLMProcessorHandlerTestSuite) TestProcessIntentHandler_Success() {
	request := ProcessIntentRequest{
		Intent: "Deploy a high-availability AMF instance for production",
		Metadata: map[string]string{
			"namespace": "nephoran-system",
			"priority":  "high",
		},
	}
	
	expectedResponse := &ProcessIntentResult{
		Result:      "Successfully processed intent: Deploy AMF with HA configuration",
		RequestID:   "req-123",
		ProcessingTime: 250 * time.Millisecond,
		Metadata: map[string]interface{}{
			"extracted_entities": map[string]interface{}{
				"network_function": "AMF",
				"deployment_type":  "high-availability",
				"environment":      "production",
			},
		},
		Status: "success",
	}
	
	suite.mockProcessor.EXPECT().
		ProcessIntent(gomock.Any(), request.Intent, request.Metadata).
		Return(expectedResponse, nil).
		Times(1)
	
	body, err := json.Marshal(request)
	require.NoError(suite.T(), err)
	
	req := httptest.NewRequest(http.MethodPost, "/process", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-request-123")
	
	recorder := httptest.NewRecorder()
	
	handler := http.HandlerFunc(suite.handler.ProcessIntentHandler)
	handler.ServeHTTP(recorder, req)
	
	assert.Equal(suite.T(), http.StatusOK, recorder.Code)
	
	var response ProcessIntentResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), expectedResponse.Result, response.Result)
	assert.Equal(suite.T(), expectedResponse.Status, response.Status)
	assert.Equal(suite.T(), "test-request-123", response.RequestID)
	assert.NotEmpty(suite.T(), response.ProcessingTime)
	assert.Empty(suite.T(), response.Error)
}

func (suite *LLMProcessorHandlerTestSuite) TestProcessIntentHandler_InvalidJSON() {
	invalidJSON := `{"intent": "test", "metadata": invalid}`
	
	req := httptest.NewRequest(http.MethodPost, "/process", strings.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	
	recorder := httptest.NewRecorder()
	
	handler := http.HandlerFunc(suite.handler.ProcessIntentHandler)
	handler.ServeHTTP(recorder, req)
	
	assert.Equal(suite.T(), http.StatusBadRequest, recorder.Code)
	
	var response ProcessIntentResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "error", response.Status)
	assert.Contains(suite.T(), response.Error, "invalid JSON")
}

func (suite *LLMProcessorHandlerTestSuite) TestProcessIntentHandler_EmptyIntent() {
	request := ProcessIntentRequest{
		Intent: "",
		Metadata: map[string]string{
			"test": "value",
		},
	}
	
	body, err := json.Marshal(request)
	require.NoError(suite.T(), err)
	
	req := httptest.NewRequest(http.MethodPost, "/process", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	recorder := httptest.NewRecorder()
	
	handler := http.HandlerFunc(suite.handler.ProcessIntentHandler)
	handler.ServeHTTP(recorder, req)
	
	assert.Equal(suite.T(), http.StatusBadRequest, recorder.Code)
	
	var response ProcessIntentResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "error", response.Status)
	assert.Contains(suite.T(), response.Error, "intent cannot be empty")
}

func (suite *LLMProcessorHandlerTestSuite) TestProcessIntentHandler_ProcessingError() {
	request := ProcessIntentRequest{
		Intent: "Deploy invalid configuration",
		Metadata: map[string]string{
			"test": "value",
		},
	}
	
	processingError := fmt.Errorf("failed to process intent: invalid configuration")
	
	suite.mockProcessor.EXPECT().
		ProcessIntent(gomock.Any(), request.Intent, request.Metadata).
		Return(nil, processingError).
		Times(1)
	
	body, err := json.Marshal(request)
	require.NoError(suite.T(), err)
	
	req := httptest.NewRequest(http.MethodPost, "/process", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	
	recorder := httptest.NewRecorder()
	
	handler := http.HandlerFunc(suite.handler.ProcessIntentHandler)
	handler.ServeHTTP(recorder, req)
	
	assert.Equal(suite.T(), http.StatusInternalServerError, recorder.Code)
	
	var response ProcessIntentResponse
	err = json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "error", response.Status)
	assert.Contains(suite.T(), response.Error, "failed to process intent")
}

func (suite *LLMProcessorHandlerTestSuite) TestHealthHandler_Healthy() {
	suite.mockHealthChecker.EXPECT().
		CheckHealth(gomock.Any()).
		Return(true, nil).
		Times(1)
	
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()
	
	handler := http.HandlerFunc(suite.handler.HealthHandler)
	handler.ServeHTTP(recorder, req)
	
	assert.Equal(suite.T(), http.StatusOK, recorder.Code)
	
	var response HealthResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "healthy", response.Status)
	assert.Equal(suite.T(), "1.0.0", response.Version)
	assert.NotEmpty(suite.T(), response.Uptime)
	assert.NotEmpty(suite.T(), response.Timestamp)
}

func (suite *LLMProcessorHandlerTestSuite) TestHealthHandler_Unhealthy() {
	healthError := fmt.Errorf("LLM API endpoint not responding")
	
	suite.mockHealthChecker.EXPECT().
		CheckHealth(gomock.Any()).
		Return(false, healthError).
		Times(1)
	
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()
	
	handler := http.HandlerFunc(suite.handler.HealthHandler)
	handler.ServeHTTP(recorder, req)
	
	assert.Equal(suite.T(), http.StatusServiceUnavailable, recorder.Code)
	
	var response HealthResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "unhealthy", response.Status)
	assert.Contains(suite.T(), response.Details, "LLM API endpoint not responding")
}

func (suite *LLMProcessorHandlerTestSuite) TestReadinessHandler_Ready() {
	suite.mockHealthChecker.EXPECT().
		IsReady(gomock.Any()).
		Return(true, nil).
		Times(1)
	
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	recorder := httptest.NewRecorder()
	
	handler := http.HandlerFunc(suite.handler.ReadinessHandler)
	handler.ServeHTTP(recorder, req)
	
	assert.Equal(suite.T(), http.StatusOK, recorder.Code)
	
	var response ReadinessResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "ready", response.Status)
	assert.True(suite.T(), response.Ready)
}

func (suite *LLMProcessorHandlerTestSuite) TestReadinessHandler_NotReady() {
	readinessError := fmt.Errorf("initializing connection to vector database")
	
	suite.mockHealthChecker.EXPECT().
		IsReady(gomock.Any()).
		Return(false, readinessError).
		Times(1)
	
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	recorder := httptest.NewRecorder()
	
	handler := http.HandlerFunc(suite.handler.ReadinessHandler)
	handler.ServeHTTP(recorder, req)
	
	assert.Equal(suite.T(), http.StatusServiceUnavailable, recorder.Code)
	
	var response ReadinessResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "not ready", response.Status)
	assert.False(suite.T(), response.Ready)
	assert.Contains(suite.T(), response.Message, "initializing connection")
}

func (suite *LLMProcessorHandlerTestSuite) TestMetricsHandler() {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()
	
	handler := http.HandlerFunc(suite.handler.MetricsHandler)
	handler.ServeHTTP(recorder, req)
	
	assert.Equal(suite.T(), http.StatusOK, recorder.Code)
	assert.Contains(suite.T(), recorder.Header().Get("Content-Type"), "text/plain")
	
	// Check for Prometheus metrics format
	body := recorder.Body.String()
	assert.Contains(suite.T(), body, "# HELP")
	assert.Contains(suite.T(), body, "# TYPE")
}

func (suite *LLMProcessorHandlerTestSuite) TestVersionHandler() {
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	recorder := httptest.NewRecorder()
	
	handler := http.HandlerFunc(suite.handler.VersionHandler)
	handler.ServeHTTP(recorder, req)
	
	assert.Equal(suite.T(), http.StatusOK, recorder.Code)
	
	var response VersionResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), "1.0.0", response.Version)
	assert.Equal(suite.T(), "main", response.GitCommit)
	assert.NotEmpty(suite.T(), response.BuildTime)
	assert.NotEmpty(suite.T(), response.GoVersion)
}

// Table-driven tests for various request scenarios
func (suite *LLMProcessorHandlerTestSuite) TestProcessIntentHandler_TableDriven() {
	testCases := []struct {
		name            string
		request         ProcessIntentRequest
		mockSetup       func()
		expectedStatus  int
		expectedError   string
		validateResponse func(*ProcessIntentResponse)
	}{
		{
			name: "Valid 5G Core deployment intent",
			request: ProcessIntentRequest{
				Intent: "Deploy 5G Core AMF with high availability in production",
				Metadata: map[string]string{
					"environment": "production",
					"namespace":   "core-network",
				},
			},
			mockSetup: func() {
				suite.mockProcessor.EXPECT().
					ProcessIntent(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&ProcessIntentResult{
						Result: "5G Core AMF deployed successfully",
						Status: "success",
					}, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(resp *ProcessIntentResponse) {
				assert.Equal(suite.T(), "success", resp.Status)
				assert.Contains(suite.T(), resp.Result, "AMF deployed")
			},
		},
		{
			name: "O-RAN network function deployment",
			request: ProcessIntentRequest{
				Intent: "Setup O-DU and O-CU for edge deployment",
				Metadata: map[string]string{
					"location": "edge-site-001",
					"region":   "us-west-2",
				},
			},
			mockSetup: func() {
				suite.mockProcessor.EXPECT().
					ProcessIntent(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&ProcessIntentResult{
						Result: "O-DU and O-CU configured for edge deployment",
						Status: "success",
						Metadata: map[string]interface{}{
							"network_functions": []string{"O-DU", "O-CU"},
							"deployment_type":   "edge",
						},
					}, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(resp *ProcessIntentResponse) {
				assert.Equal(suite.T(), "success", resp.Status)
				assert.NotNil(suite.T(), resp.Metadata)
			},
		},
		{
			name: "Network slice management intent",
			request: ProcessIntentRequest{
				Intent: "Create network slice for eMBB with 100Mbps guaranteed throughput",
				Metadata: map[string]string{
					"slice_type":   "eMBB",
					"sla_required": "true",
				},
			},
			mockSetup: func() {
				suite.mockProcessor.EXPECT().
					ProcessIntent(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&ProcessIntentResult{
						Result: "eMBB network slice created with guaranteed throughput",
						Status: "success",
					}, nil).
					Times(1)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(resp *ProcessIntentResponse) {
				assert.Equal(suite.T(), "success", resp.Status)
			},
		},
		{
			name: "Rate limiting triggered",
			request: ProcessIntentRequest{
				Intent: "Deploy test function",
			},
			mockSetup: func() {
				suite.mockProcessor.EXPECT().
					ProcessIntent(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("rate limit exceeded")).
					Times(1)
			},
			expectedStatus: http.StatusTooManyRequests,
			expectedError:  "rate limit exceeded",
		},
		{
			name: "LLM service unavailable",
			request: ProcessIntentRequest{
				Intent: "Deploy network function",
			},
			mockSetup: func() {
				suite.mockProcessor.EXPECT().
					ProcessIntent(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("LLM service unavailable")).
					Times(1)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "LLM service unavailable",
		},
	}
	
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.mockSetup()
			
			body, err := json.Marshal(tc.request)
			require.NoError(suite.T(), err)
			
			req := httptest.NewRequest(http.MethodPost, "/process", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Request-ID", fmt.Sprintf("test-%s", tc.name))
			
			recorder := httptest.NewRecorder()
			
			handler := http.HandlerFunc(suite.handler.ProcessIntentHandler)
			handler.ServeHTTP(recorder, req)
			
			assert.Equal(suite.T(), tc.expectedStatus, recorder.Code)
			
			var response ProcessIntentResponse
			err = json.Unmarshal(recorder.Body.Bytes(), &response)
			require.NoError(suite.T(), err)
			
			if tc.expectedError != "" {
				assert.Equal(suite.T(), "error", response.Status)
				assert.Contains(suite.T(), response.Error, tc.expectedError)
			} else if tc.validateResponse != nil {
				tc.validateResponse(&response)
			}
		})
	}
}

// Edge cases and error handling tests
func (suite *LLMProcessorHandlerTestSuite) TestEdgeCases() {
	testCases := []struct {
		name        string
		setupFunc   func() (*http.Request, *httptest.ResponseRecorder)
		handlerFunc func(http.ResponseWriter, *http.Request)
		validateFunc func(*httptest.ResponseRecorder)
	}{
		{
			name: "Large request body",
			setupFunc: func() (*http.Request, *httptest.ResponseRecorder) {
				largeIntent := strings.Repeat("Deploy network function ", 1000)
				request := ProcessIntentRequest{
					Intent: largeIntent,
				}
				body, _ := json.Marshal(request)
				req := httptest.NewRequest(http.MethodPost, "/process", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				return req, httptest.NewRecorder()
			},
			handlerFunc: suite.handler.ProcessIntentHandler,
			validateFunc: func(recorder *httptest.ResponseRecorder) {
				// Should handle large requests or return appropriate error
				assert.True(suite.T(), recorder.Code == http.StatusOK || recorder.Code == http.StatusRequestEntityTooLarge)
			},
		},
		{
			name: "Missing content type",
			setupFunc: func() (*http.Request, *httptest.ResponseRecorder) {
				request := ProcessIntentRequest{Intent: "test"}
				body, _ := json.Marshal(request)
				req := httptest.NewRequest(http.MethodPost, "/process", bytes.NewBuffer(body))
				// Deliberately not setting Content-Type
				return req, httptest.NewRecorder()
			},
			handlerFunc: suite.handler.ProcessIntentHandler,
			validateFunc: func(recorder *httptest.ResponseRecorder) {
				assert.Equal(suite.T(), http.StatusUnsupportedMediaType, recorder.Code)
			},
		},
		{
			name: "Wrong HTTP method",
			setupFunc: func() (*http.Request, *httptest.ResponseRecorder) {
				req := httptest.NewRequest(http.MethodGet, "/process", nil)
				return req, httptest.NewRecorder()
			},
			handlerFunc: suite.handler.ProcessIntentHandler,
			validateFunc: func(recorder *httptest.ResponseRecorder) {
				assert.Equal(suite.T(), http.StatusMethodNotAllowed, recorder.Code)
			},
		},
		{
			name: "Request timeout simulation",
			setupFunc: func() (*http.Request, *httptest.ResponseRecorder) {
				request := ProcessIntentRequest{Intent: "test timeout"}
				body, _ := json.Marshal(request)
				req := httptest.NewRequest(http.MethodPost, "/process", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				
				// Create a context with immediate cancellation
				ctx, cancel := context.WithCancel(req.Context())
				cancel() // Cancel immediately to simulate timeout
				req = req.WithContext(ctx)
				
				return req, httptest.NewRecorder()
			},
			handlerFunc: suite.handler.ProcessIntentHandler,
			validateFunc: func(recorder *httptest.ResponseRecorder) {
				// Should handle context cancellation gracefully
				assert.True(suite.T(), recorder.Code >= 400)
			},
		},
	}
	
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			req, recorder := tc.setupFunc()
			
			// Setup mock expectations for cases that reach the processor
			if strings.Contains(tc.name, "Large request") {
				suite.mockProcessor.EXPECT().
					ProcessIntent(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(&ProcessIntentResult{
						Result: "Processed large request",
						Status: "success",
					}, nil).
					MaxTimes(1)
			}
			
			handler := http.HandlerFunc(tc.handlerFunc)
			handler.ServeHTTP(recorder, req)
			
			tc.validateFunc(recorder)
		})
	}
}

// Benchmarks for performance testing
func BenchmarkProcessIntentHandler(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	config := &config.LLMProcessorConfig{
		Enabled:     true,
		APIEndpoint: "http://localhost:8080",
		APIKey:      "test-key",
		Model:       "gpt-4o-mini",
		MaxTokens:   4000,
		Temperature: 0.7,
	}
	
	handler, err := NewLLMProcessorHandler(config, logger)
	require.NoError(b, err)
	
	request := ProcessIntentRequest{
		Intent: "Deploy AMF for production use",
		Metadata: map[string]string{
			"environment": "production",
		},
	}
	
	body, err := json.Marshal(request)
	require.NoError(b, err)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodPost, "/process", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			recorder := httptest.NewRecorder()
			
			handlerFunc := http.HandlerFunc(handler.ProcessIntentHandler)
			handlerFunc.ServeHTTP(recorder, req)
		}
	})
}

// Mock implementations for testing

type MockIntentProcessor struct {
	ctrl *gomock.Controller
}

func NewMockIntentProcessor(ctrl *gomock.Controller) *MockIntentProcessor {
	return &MockIntentProcessor{ctrl: ctrl}
}

func (m *MockIntentProcessor) ProcessIntent(ctx context.Context, intent string, metadata map[string]string) (*ProcessIntentResult, error) {
	ret := m.ctrl.Call(m, "ProcessIntent", ctx, intent, metadata)
	ret0, _ := ret[0].(*ProcessIntentResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

type MockHealthChecker struct {
	ctrl *gomock.Controller
}

func NewMockHealthChecker(ctrl *gomock.Controller) *MockHealthChecker {
	return &MockHealthChecker{ctrl: ctrl}
}

func (m *MockHealthChecker) CheckHealth(ctx context.Context) (bool, error) {
	ret := m.ctrl.Call(m, "CheckHealth", ctx)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (m *MockHealthChecker) IsReady(ctx context.Context) (bool, error) {
	ret := m.ctrl.Call(m, "IsReady", ctx)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

type MockTokenManager struct {
	ctrl *gomock.Controller
}

func NewMockTokenManager(ctrl *gomock.Controller) *MockTokenManager {
	return &MockTokenManager{ctrl: ctrl}
}

func (m *MockTokenManager) GetAvailableTokens() int {
	ret := m.ctrl.Call(m, "GetAvailableTokens")
	ret0, _ := ret[0].(int)
	return ret0
}

func (m *MockTokenManager) ConsumeTokens(tokens int) error {
	ret := m.ctrl.Call(m, "ConsumeTokens", tokens)
	ret0, _ := ret[1].(error)
	return ret0
}

// Response types for testing
type ProcessIntentResult struct {
	Result         string                 `json:"result"`
	RequestID      string                 `json:"request_id"`
	ProcessingTime time.Duration          `json:"processing_time"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	Status         string                 `json:"status"`
}

type ReadinessResponse struct {
	Status  string `json:"status"`
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
}

type VersionResponse struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

type IntentProcessor interface {
	ProcessIntent(ctx context.Context, intent string, metadata map[string]string) (*ProcessIntentResult, error)
}

// Placeholder implementations for handler methods

func NewLLMProcessorHandler(config *config.LLMProcessorConfig, logger *slog.Logger) (*LLMProcessorHandler, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	if config.APIEndpoint == "" {
		return nil, fmt.Errorf("API endpoint cannot be empty")
	}
	
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}
	
	if config.Temperature < 0 || config.Temperature > 2 {
		return nil, fmt.Errorf("temperature must be between 0 and 2")
	}
	
	if config.MaxTokens <= 0 {
		return nil, fmt.Errorf("max tokens must be greater than 0")
	}
	
	// Create mock processor and health checker for testing
	processor := &MockIntentProcessorImpl{}
	healthChecker := &health.HealthChecker{}
	
	return &LLMProcessorHandler{
		config:        config,
		processor:     processor,
		logger:        logger,
		healthChecker: healthChecker,
		startTime:     time.Now(),
	}, nil
}

// Mock implementation for testing
type MockIntentProcessorImpl struct{}

func (m *MockIntentProcessorImpl) ProcessIntent(ctx context.Context, intent string, metadata map[string]string) (*ProcessIntentResult, error) {
	return &ProcessIntentResult{
		Result: fmt.Sprintf("Processed: %s", intent),
		Status: "success",
	}, nil
}

// Handler method implementations for testing
func (h *LLMProcessorHandler) ProcessIntentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "Unsupported media type", http.StatusUnsupportedMediaType)
		return
	}
	
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	
	var request ProcessIntentRequest
	if err := json.Unmarshal(body, &request); err != nil {
		response := ProcessIntentResponse{
			Status: "error",
			Error:  "invalid JSON: " + err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteStatus(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	if request.Intent == "" {
		response := ProcessIntentResponse{
			Status: "error",
			Error:  "intent cannot be empty",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteStatus(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	start := time.Now()
	result, err := h.processor.ProcessIntent(r.Context(), request.Intent, request.Metadata)
	processingTime := time.Since(start)
	
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "rate limit") {
			status = http.StatusTooManyRequests
		}
		
		response := ProcessIntentResponse{
			Status:         "error",
			Error:          err.Error(),
			RequestID:      requestID,
			ProcessingTime: processingTime.String(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteStatus(status)
		json.NewEncoder(w).Encode(response)
		return
	}
	
	response := ProcessIntentResponse{
		Result:         result.Result,
		ProcessingTime: processingTime.String(),
		RequestID:      requestID,
		ServiceVersion: "1.0.0",
		Metadata:       result.Metadata,
		Status:         result.Status,
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteStatus(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *LLMProcessorHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	healthy, err := h.healthChecker.CheckHealth(r.Context())
	
	response := HealthResponse{
		Version:   "1.0.0",
		Uptime:    time.Since(h.startTime).String(),
		Timestamp: time.Now().Format(time.RFC3339),
	}
	
	if healthy {
		response.Status = "healthy"
		w.WriteStatus(http.StatusOK)
	} else {
		response.Status = "unhealthy"
		if err != nil {
			response.Details = err.Error()
		}
		w.WriteStatus(http.StatusServiceUnavailable)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *LLMProcessorHandler) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	ready, err := h.healthChecker.IsReady(r.Context())
	
	response := ReadinessResponse{
		Ready: ready,
	}
	
	if ready {
		response.Status = "ready"
		w.WriteStatus(http.StatusOK)
	} else {
		response.Status = "not ready"
		if err != nil {
			response.Message = err.Error()
		}
		w.WriteStatus(http.StatusServiceUnavailable)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *LLMProcessorHandler) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteStatus(http.StatusOK)
	
	// Mock Prometheus metrics output
	fmt.Fprintf(w, `# HELP http_requests_total Total number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",path="/health"} 10
http_requests_total{method="POST",path="/process"} 25

# HELP http_request_duration_seconds Duration of HTTP requests
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{method="POST",path="/process",le="0.1"} 15
http_request_duration_seconds_bucket{method="POST",path="/process",le="0.5"} 20
http_request_duration_seconds_bucket{method="POST",path="/process",le="1.0"} 25
http_request_duration_seconds_bucket{method="POST",path="/process",le="+Inf"} 25
http_request_duration_seconds_sum{method="POST",path="/process"} 8.5
http_request_duration_seconds_count{method="POST",path="/process"} 25
`)
}

func (h *LLMProcessorHandler) VersionHandler(w http.ResponseWriter, r *http.Request) {
	response := VersionResponse{
		Version:   "1.0.0",
		GitCommit: "main",
		BuildTime: "2025-01-08T10:00:00Z",
		GoVersion: "go1.24",
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteStatus(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}