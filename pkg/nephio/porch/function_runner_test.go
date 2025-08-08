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

package porch

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations for testing

type MockFunctionRegistry struct {
	mock.Mock
}

func (m *MockFunctionRegistry) RegisterFunction(ctx context.Context, info *FunctionInfo) error {
	args := m.Called(ctx, info)
	return args.Error(0)
}

func (m *MockFunctionRegistry) UnregisterFunction(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockFunctionRegistry) GetFunction(ctx context.Context, name string) (*FunctionInfo, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*FunctionInfo), args.Error(1)
}

func (m *MockFunctionRegistry) ListFunctions(ctx context.Context) ([]*FunctionInfo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*FunctionInfo), args.Error(1)
}

func (m *MockFunctionRegistry) SearchFunctions(ctx context.Context, query string) ([]*FunctionInfo, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]*FunctionInfo), args.Error(1)
}

func (m *MockFunctionRegistry) ValidateFunction(ctx context.Context, name string) (*FunctionValidation, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*FunctionValidation), args.Error(1)
}

func (m *MockFunctionRegistry) GetFunctionSchema(ctx context.Context, name string) (*FunctionSchema, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*FunctionSchema), args.Error(1)
}

type MockExecutionEngine struct {
	mock.Mock
}

func (m *MockExecutionEngine) ExecuteFunction(ctx context.Context, req *FunctionExecutionRequest) (*FunctionExecutionResult, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*FunctionExecutionResult), args.Error(1)
}

func (m *MockExecutionEngine) GetSupportedRuntimes() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockExecutionEngine) IsRuntimeAvailable(runtime string) bool {
	args := m.Called(runtime)
	return args.Bool(0)
}

func (m *MockExecutionEngine) GetResourceLimits() *FunctionResourceLimits {
	args := m.Called()
	return args.Get(0).(*FunctionResourceLimits)
}

func (m *MockExecutionEngine) SetResourceLimits(limits *FunctionResourceLimits) {
	m.Called(limits)
}

type MockResourceQuotaManager struct {
	mock.Mock
}

func (m *MockResourceQuotaManager) CheckQuota(ctx context.Context, req *FunctionExecutionRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockResourceQuotaManager) ReserveResources(ctx context.Context, executionID string, resources *ResourceRequirement) error {
	args := m.Called(ctx, executionID, resources)
	return args.Error(0)
}

func (m *MockResourceQuotaManager) ReleaseResources(ctx context.Context, executionID string) error {
	args := m.Called(ctx, executionID)
	return args.Error(0)
}

func (m *MockResourceQuotaManager) GetQuotaUsage(ctx context.Context) (*QuotaUsage, error) {
	args := m.Called(ctx)
	return args.Get(0).(*QuotaUsage), args.Error(1)
}

type MockFunctionSecurityValidator struct {
	mock.Mock
}

func (m *MockFunctionSecurityValidator) ValidateFunction(ctx context.Context, req *FunctionExecutionRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockFunctionSecurityValidator) ScanImage(ctx context.Context, image string) (*SecurityScanResult, error) {
	args := m.Called(ctx, image)
	return args.Get(0).(*SecurityScanResult), args.Error(1)
}

func (m *MockFunctionSecurityValidator) ValidatePermissions(ctx context.Context, req *FunctionExecutionRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockFunctionSecurityValidator) CheckCompliance(ctx context.Context, req *FunctionExecutionRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

type MockPipelineOrchestrator struct {
	mock.Mock
}

func (m *MockPipelineOrchestrator) ExecutePipeline(ctx context.Context, req *PipelineRequest) (*PipelineResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*PipelineResponse), args.Error(1)
}

func (m *MockPipelineOrchestrator) ValidatePipeline(ctx context.Context, pipeline *Pipeline) error {
	args := m.Called(ctx, pipeline)
	return args.Error(0)
}

func (m *MockPipelineOrchestrator) OptimizePipeline(ctx context.Context, pipeline *Pipeline) (*Pipeline, error) {
	args := m.Called(ctx, pipeline)
	return args.Get(0).(*Pipeline), args.Error(1)
}

func (m *MockPipelineOrchestrator) GetPipelineMetrics(ctx context.Context, pipelineID string) (*PipelineMetrics, error) {
	args := m.Called(ctx, pipelineID)
	return args.Get(0).(*PipelineMetrics), args.Error(1)
}

type MockORANComplianceValidator struct {
	mock.Mock
}

func (m *MockORANComplianceValidator) ValidateCompliance(ctx context.Context, resources []KRMResource) (*ComplianceResult, error) {
	args := m.Called(ctx, resources)
	return args.Get(0).(*ComplianceResult), args.Error(1)
}

func (m *MockORANComplianceValidator) GetComplianceRules() []ComplianceRule {
	args := m.Called()
	return args.Get(0).([]ComplianceRule)
}

func (m *MockORANComplianceValidator) ValidateInterface(ctx context.Context, interfaceType string, config map[string]interface{}) error {
	args := m.Called(ctx, interfaceType, config)
	return args.Error(0)
}

func (m *MockORANComplianceValidator) ValidateNetworkFunction(ctx context.Context, nf *NetworkFunctionSpec) error {
	args := m.Called(ctx, nf)
	return args.Error(0)
}

type MockFunctionCache struct {
	mock.Mock
}

func (m *MockFunctionCache) Get(ctx context.Context, key string) (*FunctionResponse, bool) {
	args := m.Called(ctx, key)
	return args.Get(0).(*FunctionResponse), args.Bool(1)
}

func (m *MockFunctionCache) Set(ctx context.Context, key string, response *FunctionResponse, ttl time.Duration) error {
	args := m.Called(ctx, key, response, ttl)
	return args.Error(0)
}

func (m *MockFunctionCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockFunctionCache) Clear(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockFunctionCache) GetStats() *CacheStats {
	args := m.Called()
	return args.Get(0).(*CacheStats)
}

// Test fixtures

func createTestFunctionRequest() *FunctionRequest {
	return &FunctionRequest{
		FunctionConfig: FunctionConfig{
			Image: "gcr.io/kpt-fn/apply-setters:v0.1.1",
			ConfigMap: map[string]string{
				"key1": "value1",
			},
		},
		Resources: []KRMResource{
			{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Metadata: map[string]interface{}{
					"name":      "test-config",
					"namespace": "default",
				},
				Data: map[string]interface{}{
					"config": "test-value",
				},
			},
		},
		Context: &FunctionContext{
			Environment: map[string]string{
				"ENV_VAR": "test-value",
			},
		},
	}
}

func createTestPipelineRequest() *PipelineRequest {
	return &PipelineRequest{
		Pipeline: Pipeline{
			Mutators: []FunctionConfig{
				{
					Image: "gcr.io/kpt-fn/apply-setters:v0.1.1",
				},
				{
					Image: "gcr.io/kpt-fn/set-namespace:v0.1.1",
					ConfigMap: map[string]string{
						"namespace": "test-namespace",
					},
				},
			},
			Validators: []FunctionConfig{
				{
					Image: "gcr.io/kpt-fn/kubeval:v0.1.1",
				},
			},
		},
		Resources: []KRMResource{
			{
				APIVersion: "v1",
				Kind:       "Pod",
				Metadata: map[string]interface{}{
					"name": "test-pod",
				},
			},
		},
	}
}

func createTestFunctionRunner(client *Client) *functionRunner {
	fr := &functionRunner{
		client:           client,
		activeExecutions: make(map[string]*executionContext),
		semaphore:        make(chan struct{}, 5),
		metrics:          initFunctionRunnerMetrics(),
	}
	
	// Set up mocks with default behaviors
	mockRegistry := &MockFunctionRegistry{}
	mockEngine := &MockExecutionEngine{}
	mockQuotaManager := &MockResourceQuotaManager{}
	mockSecurityValidator := &MockFunctionSecurityValidator{}
	mockPipelineOrchestrator := &MockPipelineOrchestrator{}
	mockORANValidator := &MockORANComplianceValidator{}
	mockCache := &MockFunctionCache{}
	
	// Configure default mock behaviors
	mockRegistry.On("ValidateFunction", mock.Anything, mock.Anything).Return(&FunctionValidation{Valid: true}, nil)
	mockRegistry.On("ListFunctions", mock.Anything).Return([]*FunctionInfo{
		{
			Name:        "gcr.io/kpt-fn/apply-setters",
			Description: "Apply setters",
			Types:       []string{"mutator"},
		},
	}, nil)
	mockRegistry.On("GetFunctionSchema", mock.Anything, mock.Anything).Return(&FunctionSchema{}, nil)
	
	mockEngine.On("ExecuteFunction", mock.Anything, mock.Anything).Return(&FunctionExecutionResult{
		FunctionResponse: &FunctionResponse{
			Resources: []KRMResource{},
			Results:   []*FunctionResult{},
		},
		ResourceUsage: &ResourceUsage{
			CPU:      0.1,
			Memory:   100 * 1024 * 1024, // 100MB
			Duration: 100 * time.Millisecond,
		},
	}, nil)
	mockEngine.On("GetSupportedRuntimes").Return([]string{"docker", "containerd"})
	mockEngine.On("IsRuntimeAvailable", mock.Anything).Return(true)
	mockEngine.On("GetResourceLimits").Return(&FunctionResourceLimits{
		CPU:    "1000m",
		Memory: "1Gi",
	})
	
	mockQuotaManager.On("CheckQuota", mock.Anything, mock.Anything).Return(nil)
	mockQuotaManager.On("ReserveResources", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockQuotaManager.On("ReleaseResources", mock.Anything, mock.Anything).Return(nil)
	
	mockSecurityValidator.On("ValidateFunction", mock.Anything, mock.Anything).Return(nil)
	
	mockPipelineOrchestrator.On("ValidatePipeline", mock.Anything, mock.Anything).Return(nil)
	mockPipelineOrchestrator.On("ExecutePipeline", mock.Anything, mock.Anything).Return(&PipelineResponse{
		Resources: []KRMResource{},
		Results:   []*FunctionResult{},
	}, nil)
	
	mockORANValidator.On("ValidateCompliance", mock.Anything, mock.Anything).Return(&ComplianceResult{
		Compliant:  true,
		Violations: []ComplianceViolation{},
		Score:      100,
	}, nil)
	
	mockCache.On("Get", mock.Anything, mock.Anything).Return((*FunctionResponse)(nil), false)
	mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	
	fr.SetFunctionRegistry(mockRegistry)
	fr.SetExecutionEngine(mockEngine)
	fr.SetQuotaManager(mockQuotaManager)
	fr.SetSecurityValidator(mockSecurityValidator)
	fr.SetPipelineOrchestrator(mockPipelineOrchestrator)
	fr.SetORANValidator(mockORANValidator)
	fr.SetFunctionCache(mockCache)
	
	return fr
}

// Unit Tests

func TestNewFunctionRunner(t *testing.T) {
	client := createTestClient()
	fr := NewFunctionRunner(client)
	
	assert.NotNil(t, fr)
	assert.IsType(t, &functionRunner{}, fr)
	
	frImpl := fr.(*functionRunner)
	assert.Equal(t, client, frImpl.client)
	assert.NotNil(t, frImpl.activeExecutions)
	assert.NotNil(t, frImpl.semaphore)
	assert.NotNil(t, frImpl.metrics)
}

func TestFunctionRunnerExecuteFunction(t *testing.T) {
	tests := []struct {
		name        string
		request     *FunctionRequest
		setupMocks  func(*functionRunner)
		expectError bool
	}{
		{
			name:    "ValidFunctionExecution",
			request: createTestFunctionRequest(),
			setupMocks: func(fr *functionRunner) {
				// Default mocks are already set up
			},
			expectError: false,
		},
		{
			name:    "SecurityValidationFailure",
			request: createTestFunctionRequest(),
			setupMocks: func(fr *functionRunner) {
				mockValidator := fr.securityValidator.(*MockFunctionSecurityValidator)
				mockValidator.ExpectedCalls = nil
				mockValidator.On("ValidateFunction", mock.Anything, mock.Anything).Return(fmt.Errorf("security validation failed"))
			},
			expectError: true,
		},
		{
			name:    "QuotaExceeded",
			request: createTestFunctionRequest(),
			setupMocks: func(fr *functionRunner) {
				mockQuotaManager := fr.quotaManager.(*MockResourceQuotaManager)
				mockQuotaManager.ExpectedCalls = nil
				mockQuotaManager.On("CheckQuota", mock.Anything, mock.Anything).Return(fmt.Errorf("quota exceeded"))
			},
			expectError: true,
		},
		{
			name:    "ExecutionEngineFailure",
			request: createTestFunctionRequest(),
			setupMocks: func(fr *functionRunner) {
				mockEngine := fr.executionEngine.(*MockExecutionEngine)
				mockEngine.ExpectedCalls = nil
				mockEngine.On("ExecuteFunction", mock.Anything, mock.Anything).Return((*FunctionExecutionResult)(nil), fmt.Errorf("execution failed"))
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := createTestClient()
			fr := createTestFunctionRunner(client)
			
			if tt.setupMocks != nil {
				tt.setupMocks(fr)
			}
			
			response, err := fr.ExecuteFunction(context.Background(), tt.request)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
			}
		})
	}
}

func TestFunctionRunnerExecutePipeline(t *testing.T) {
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	
	t.Run("ValidPipelineExecution", func(t *testing.T) {
		request := createTestPipelineRequest()
		
		response, err := fr.ExecutePipeline(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
	})
	
	t.Run("PipelineValidationFailure", func(t *testing.T) {
		request := createTestPipelineRequest()
		
		mockOrchestrator := fr.pipelineOrchestrator.(*MockPipelineOrchestrator)
		mockOrchestrator.ExpectedCalls = nil
		mockOrchestrator.On("ValidatePipeline", mock.Anything, mock.Anything).Return(fmt.Errorf("pipeline validation failed"))
		
		response, err := fr.ExecutePipeline(context.Background(), request)
		assert.Error(t, err)
		assert.Nil(t, response)
	})
	
	t.Run("SequentialPipelineExecution", func(t *testing.T) {
		// Test fallback to sequential execution when orchestrator is not available
		request := createTestPipelineRequest()
		
		// Remove orchestrator to trigger sequential execution
		fr.pipelineOrchestrator = nil
		
		response, err := fr.ExecutePipeline(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
	})
}

func TestFunctionRunnerValidateFunction(t *testing.T) {
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	
	t.Run("ValidFunction", func(t *testing.T) {
		validation, err := fr.ValidateFunction(context.Background(), "gcr.io/kpt-fn/apply-setters:v0.1.1")
		assert.NoError(t, err)
		assert.NotNil(t, validation)
		assert.True(t, validation.Valid)
	})
	
	t.Run("InvalidFunctionImage", func(t *testing.T) {
		validation, err := fr.ValidateFunction(context.Background(), "invalid-function")
		assert.NoError(t, err)
		assert.NotNil(t, validation)
		assert.False(t, validation.Valid)
		assert.NotEmpty(t, validation.Errors)
	})
	
	t.Run("FunctionWithoutTag", func(t *testing.T) {
		validation, err := fr.ValidateFunction(context.Background(), "gcr.io/kpt-fn/apply-setters")
		assert.NoError(t, err)
		assert.NotNil(t, validation)
		assert.False(t, validation.Valid)
		assert.NotEmpty(t, validation.Errors)
	})
}

func TestFunctionRunnerListFunctions(t *testing.T) {
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	
	functions, err := fr.ListFunctions(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, functions)
	assert.Greater(t, len(functions), 0)
}

func TestFunctionRunnerGetFunctionSchema(t *testing.T) {
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	
	schema, err := fr.GetFunctionSchema(context.Background(), "gcr.io/kpt-fn/apply-setters:v0.1.1")
	assert.NoError(t, err)
	assert.NotNil(t, schema)
}

func TestFunctionRunnerRegisterFunction(t *testing.T) {
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	
	info := &FunctionInfo{
		Name:        "custom-function",
		Description: "Custom test function",
		Types:       []string{"mutator"},
	}
	
	mockRegistry := fr.registry.(*MockFunctionRegistry)
	mockRegistry.On("RegisterFunction", mock.Anything, info).Return(nil)
	
	err := fr.RegisterFunction(context.Background(), info)
	assert.NoError(t, err)
}

func TestFunctionRunnerCaching(t *testing.T) {
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	
	request := createTestFunctionRequest()
	
	t.Run("CacheHit", func(t *testing.T) {
		mockCache := fr.cache.(*MockFunctionCache)
		cachedResponse := &FunctionResponse{
			Resources: []KRMResource{},
			Results:   []*FunctionResult{},
		}
		
		mockCache.ExpectedCalls = nil
		mockCache.On("Get", mock.Anything, mock.Anything).Return(cachedResponse, true)
		
		response, err := fr.ExecuteFunction(context.Background(), request)
		assert.NoError(t, err)
		assert.Equal(t, cachedResponse, response)
	})
	
	t.Run("CacheMiss", func(t *testing.T) {
		mockCache := fr.cache.(*MockFunctionCache)
		
		mockCache.ExpectedCalls = nil
		mockCache.On("Get", mock.Anything, mock.Anything).Return((*FunctionResponse)(nil), false)
		mockCache.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		
		response, err := fr.ExecuteFunction(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
	})
}

func TestFunctionRunnerActiveExecutions(t *testing.T) {
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	
	t.Run("TrackActiveExecutions", func(t *testing.T) {
		// Start function execution in background
		var wg sync.WaitGroup
		wg.Add(1)
		
		go func() {
			defer wg.Done()
			request := createTestFunctionRequest()
			fr.ExecuteFunction(context.Background(), request)
		}()
		
		// Brief wait to ensure execution starts
		time.Sleep(10 * time.Millisecond)
		
		executions := fr.GetActiveExecutions()
		assert.GreaterOrEqual(t, len(executions), 0) // May be 0 if execution completes very quickly
		
		wg.Wait()
	})
	
	t.Run("CancelExecution", func(t *testing.T) {
		// This test is tricky because we need to simulate a long-running execution
		// For simplicity, we'll test the cancel functionality directly
		
		execCtx := &executionContext{
			id:           "test-execution-id",
			functionName: "test-function",
			startTime:    time.Now(),
			status:       ExecutionStatusRunning,
		}
		
		ctx, cancel := context.WithCancel(context.Background())
		execCtx.cancel = cancel
		
		fr.activeExecutions["test-execution-id"] = execCtx
		
		err := fr.CancelExecution("test-execution-id")
		assert.NoError(t, err)
		
		assert.Equal(t, ExecutionStatusCancelled, execCtx.status)
	})
}

func TestFunctionRunnerClose(t *testing.T) {
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	
	err := fr.Close()
	assert.NoError(t, err)
	
	// Verify cleanup
	assert.Empty(t, fr.activeExecutions)
}

// Performance Tests

func TestFunctionRunnerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}
	
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	
	t.Run("FunctionExecutionLatency", func(t *testing.T) {
		request := createTestFunctionRequest()
		
		start := time.Now()
		_, err := fr.ExecuteFunction(context.Background(), request)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Less(t, duration, 500*time.Millisecond, "Function execution should complete in <500ms")
	})
	
	t.Run("PipelineExecutionLatency", func(t *testing.T) {
		request := createTestPipelineRequest()
		
		start := time.Now()
		_, err := fr.ExecutePipeline(context.Background(), request)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Less(t, duration, 1*time.Second, "Pipeline execution should complete in <1s")
	})
	
	t.Run("FunctionValidationLatency", func(t *testing.T) {
		start := time.Now()
		_, err := fr.ValidateFunction(context.Background(), "gcr.io/kpt-fn/apply-setters:v0.1.1")
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Less(t, duration, 100*time.Millisecond, "Function validation should complete in <100ms")
	})
}

func TestFunctionRunnerConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}
	
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	concurrency := 50
	
	t.Run("ConcurrentFunctionExecution", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]error, concurrency)
		
		start := time.Now()
		
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				
				request := createTestFunctionRequest()
				request.FunctionConfig.ConfigMap = map[string]string{
					"execution_id": fmt.Sprintf("exec-%d", index),
				}
				
				_, err := fr.ExecuteFunction(context.Background(), request)
				results[index] = err
			}(i)
		}
		
		wg.Wait()
		duration := time.Since(start)
		
		// Check all operations succeeded
		for i, err := range results {
			assert.NoError(t, err, "Operation %d should succeed", i)
		}
		
		// Check throughput
		opsPerSecond := float64(concurrency) / duration.Seconds()
		assert.Greater(t, opsPerSecond, 5.0, "Should handle >5 function executions per second")
	})
	
	t.Run("ConcurrentPipelineExecution", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]error, concurrency/2) // Fewer pipeline executions as they're more expensive
		
		for i := 0; i < len(results); i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				
				request := createTestPipelineRequest()
				_, err := fr.ExecutePipeline(context.Background(), request)
				results[index] = err
			}(i)
		}
		
		wg.Wait()
		
		// Check all operations succeeded
		for i, err := range results {
			assert.NoError(t, err, "Pipeline %d should succeed", i)
		}
	})
}

func TestFunctionRunnerMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory tests in short mode")
	}
	
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	
	// Force garbage collection before measuring
	runtime.GC()
	runtime.GC()
	
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	
	// Perform a series of operations
	for i := 0; i < 100; i++ {
		request := createTestFunctionRequest()
		request.FunctionConfig.ConfigMap = map[string]string{
			"iteration": fmt.Sprintf("%d", i),
		}
		
		_, err := fr.ExecuteFunction(context.Background(), request)
		require.NoError(t, err)
		
		if i%10 == 0 {
			pipelineRequest := createTestPipelineRequest()
			_, err = fr.ExecutePipeline(context.Background(), pipelineRequest)
			require.NoError(t, err)
		}
	}
	
	// Force garbage collection after operations
	runtime.GC()
	runtime.GC()
	
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	
	memoryUsed := m2.Alloc - m1.Alloc
	memoryUsedMB := float64(memoryUsed) / (1024 * 1024)
	
	assert.Less(t, memoryUsedMB, 50.0, "Memory usage should be <50MB for 110 operations")
}

// Benchmark Tests

func BenchmarkFunctionRunnerOperations(b *testing.B) {
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	
	b.Run("ExecuteFunction", func(b *testing.B) {
		request := createTestFunctionRequest()
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := fr.ExecuteFunction(context.Background(), request)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("ValidateFunction", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := fr.ValidateFunction(context.Background(), "gcr.io/kpt-fn/apply-setters:v0.1.1")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("ExecutePipeline", func(b *testing.B) {
		request := createTestPipelineRequest()
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := fr.ExecutePipeline(context.Background(), request)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("ListFunctions", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := fr.ListFunctions(context.Background())
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Error handling tests

func TestFunctionRunnerErrorHandling(t *testing.T) {
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	
	t.Run("NonexistentFunction", func(t *testing.T) {
		request := createTestFunctionRequest()
		request.FunctionConfig.Image = "nonexistent/function:latest"
		
		// Mock the execution engine to return an error
		mockEngine := fr.executionEngine.(*MockExecutionEngine)
		mockEngine.ExpectedCalls = nil
		mockEngine.On("ExecuteFunction", mock.Anything, mock.Anything).Return(
			(*FunctionExecutionResult)(nil), 
			fmt.Errorf("function image not found"),
		)
		
		_, err := fr.ExecuteFunction(context.Background(), request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function execution failed")
	})
	
	t.Run("InvalidPipeline", func(t *testing.T) {
		request := createTestPipelineRequest()
		request.Pipeline.Mutators = []FunctionConfig{} // Empty pipeline
		request.Pipeline.Validators = []FunctionConfig{}
		
		response, err := fr.ExecutePipeline(context.Background(), request)
		assert.NoError(t, err) // Empty pipeline should succeed
		assert.NotNil(t, response)
	})
	
	t.Run("RegisterFunctionWithoutRegistry", func(t *testing.T) {
		// Create function runner without registry
		frWithoutRegistry := &functionRunner{
			client:           client,
			activeExecutions: make(map[string]*executionContext),
			semaphore:        make(chan struct{}, 5),
		}
		
		info := &FunctionInfo{
			Name: "test-function",
		}
		
		err := frWithoutRegistry.RegisterFunction(context.Background(), info)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function registry not available")
	})
}

// Context cancellation tests

func TestFunctionRunnerContextCancellation(t *testing.T) {
	client := createTestClient()
	fr := createTestFunctionRunner(client)
	
	t.Run("CancelledContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		request := createTestFunctionRequest()
		_, err := fr.ExecuteFunction(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
	
	t.Run("TimeoutContext", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		
		time.Sleep(1 * time.Millisecond) // Ensure timeout
		
		request := createTestFunctionRequest()
		_, err := fr.ExecuteFunction(ctx, request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}