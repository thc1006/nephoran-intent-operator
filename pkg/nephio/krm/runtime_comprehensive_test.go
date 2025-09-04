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

package krm

import (
	
	"encoding/json"
"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// KRMResource represents a Kubernetes resource manifest for KRM functions
type KRMResource struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   json.RawMessage `json:"metadata"`
	Spec       json.RawMessage `json:"spec,omitempty"`
	Status     json.RawMessage `json:"status,omitempty"`
	Data       json.RawMessage `json:"data,omitempty"`
}

// FunctionConfig defines KRM function configuration
type FunctionConfig struct {
	Image      string                 `json:"image"`
	ConfigPath string                 `json:"configPath,omitempty"`
	ConfigMap  json.RawMessage `json:"configMap,omitempty"`
	Selectors  []TestResourceSelector     `json:"selectors,omitempty"`
	Exec       *ExecConfig            `json:"exec,omitempty"`
}

// TestResourceSelector defines resource selection criteria for tests
type TestResourceSelector struct {
	APIVersion string            `json:"apiVersion,omitempty"`
	Kind       string            `json:"kind,omitempty"`
	Name       string            `json:"name,omitempty"`
	Namespace  string            `json:"namespace,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// ExecConfig defines execution configuration for container functions
type ExecConfig struct {
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	WorkDir string            `json:"workDir,omitempty"`
}

// FunctionRequest represents a function execution request
type FunctionRequest struct {
	FunctionConfig FunctionConfig   `json:"functionConfig"`
	Resources      []KRMResource    `json:"resources"`
	Context        *FunctionContext `json:"context,omitempty"`
}

// FunctionResponse represents a function execution response
type FunctionResponse struct {
	Resources []KRMResource     `json:"resources"`
	Results   []*FunctionResult `json:"results,omitempty"`
	Logs      []string          `json:"logs,omitempty"`
	Error     *FunctionError    `json:"error,omitempty"`
}

// FunctionResult represents a function execution result
type FunctionResult struct {
	Message  string            `json:"message"`
	Severity string            `json:"severity"`
	Field    string            `json:"field,omitempty"`
	File     string            `json:"file,omitempty"`
	Tags     map[string]string `json:"tags,omitempty"`
}

// FunctionError represents a function execution error
type FunctionError struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// FunctionContext provides context for function execution
type FunctionContext struct {
	PackagePath string            `json:"packagePath,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

// TestPipeline defines a sequence of KRM functions for tests
type TestPipeline struct {
	Name      string                 `json:"name"`
	Functions []FunctionConfig       `json:"functions"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

// PipelineRequest represents a pipeline execution request
type PipelineRequest struct {
	Pipeline  TestPipeline         `json:"pipeline"`
	Resources []KRMResource    `json:"resources"`
	Context   *FunctionContext `json:"context,omitempty"`
}

// PipelineResponse represents a pipeline execution response
type PipelineResponse struct {
	Resources []KRMResource     `json:"resources"`
	Results   []*FunctionResult `json:"results,omitempty"`
	Error     *FunctionError    `json:"error,omitempty"`
}

// MockKRMRuntime provides a mock implementation of KRM function runtime
type MockKRMRuntime struct {
	client            client.Client
	k8sClient         *fake.Clientset
	logger            *zap.Logger
	functions         map[string]*RegisteredFunction
	executionHistory  []ExecutionEvent
	mutex             sync.RWMutex
	simulateFailure   bool
	simulateLatency   time.Duration
	validationEnabled bool
}

// RegisteredFunction represents a registered KRM function
type RegisteredFunction struct {
	Name        string
	Image       string
	Description string
	Version     string
	Handler     FunctionHandler
	Schema      *FunctionSchema
	Examples    []FunctionExample
}

// FunctionHandler defines the signature for function handlers
type FunctionHandler func(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error)

// FunctionSchema defines function configuration schema
type FunctionSchema struct {
	Properties map[string]SchemaProperty `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
}

// SchemaProperty defines a schema property
type TestSchemaProperty struct {
	Type        string        `json:"type"`
	Description string        `json:"description,omitempty"`
	Default     interface{}   `json:"default,omitempty"`
	Examples    []interface{} `json:"examples,omitempty"`
}

// FunctionExample contains function usage examples
type TestFunctionExample struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Config      *FunctionConfig `json:"config"`
	Input       []KRMResource   `json:"input"`
	Output      []KRMResource   `json:"output"`
}

// ExecutionEvent records function execution history
type ExecutionEvent struct {
	FunctionName string
	Request      *FunctionRequest
	Response     *FunctionResponse
	Error        error
	Timestamp    time.Time
	Duration     time.Duration
}

// NewMockKRMRuntime creates a new mock KRM runtime
func NewMockKRMRuntime() *MockKRMRuntime {
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	client := clientfake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := fake.NewSimpleClientset()

	runtime := &MockKRMRuntime{
		client:            client,
		k8sClient:         k8sClient,
		logger:            zaptest.NewLogger(&testing.T{}),
		functions:         make(map[string]*RegisteredFunction),
		executionHistory:  []ExecutionEvent{},
		validationEnabled: true,
	}

	// Register standard functions
	runtime.registerStandardFunctions()

	return runtime
}

// registerStandardFunctions registers commonly used KRM functions
func (r *MockKRMRuntime) registerStandardFunctions() {
	// Set Labels function
	r.RegisterFunction(&RegisteredFunction{
		Name:        "set-labels",
		Image:       "gcr.io/kpt-fn/set-labels:v0.2.0",
		Description: "Sets labels on Kubernetes resources",
		Version:     "v0.2.0",
		Handler:     r.setLabelsHandler,
		Schema: &FunctionSchema{
			Properties: map[string]SchemaProperty{
				"labels": {
					Type:        "object",
					Description: "Labels to set on resources",
				},
			},
			Required: []string{"labels"},
		},
		Examples: []FunctionExample{
			{
				Name:        "basic-labels",
				Description: "Set basic labels on resources",
				Config: map[string]interface{}{
					"image": "gcr.io/kpt-fn/set-labels:v0.2.0",
					"labels": map[string]interface{}{
						"app": "test-app",
					},
				},
				Input: []KRMResource{
					generateTestResource("apps/v1", "Deployment", "test-deployment", "default"),
				},
				Output: []KRMResource{
					generateTestResourceWithLabels("apps/v1", "Deployment", "test-deployment", "default", map[string]string{
					"app": "my-app",
					"env": "production",
					}),
				},
			},
		},
	})

	// Set Namespace function
	r.RegisterFunction(&RegisteredFunction{
		Name:        "set-namespace",
		Image:       "gcr.io/kpt-fn/set-namespace:v0.4.1",
		Description: "Sets namespace on Kubernetes resources",
		Version:     "v0.4.1",
		Handler:     r.setNamespaceHandler,
		Schema: &FunctionSchema{
			Properties: map[string]SchemaProperty{
				"namespace": {
					Type:        "string",
					Description: "Target namespace for resources",
				},
			},
			Required: []string{"namespace"},
		},
	})

	// Apply Replacements function
	r.RegisterFunction(&RegisteredFunction{
		Name:        "apply-replacements",
		Image:       "gcr.io/kpt-fn/apply-replacements:v0.1.1",
		Description: "Applies replacements to Kubernetes resources",
		Version:     "v0.1.1",
		Handler:     r.applyReplacementsHandler,
	})

	// Search Replace function
	r.RegisterFunction(&RegisteredFunction{
		Name:        "search-replace",
		Image:       "gcr.io/kpt-fn/search-replace:v0.2.0",
		Description: "Performs search and replace operations on resources",
		Version:     "v0.2.0",
		Handler:     r.searchReplaceHandler,
	})

	// Ensure Name Substring function
	r.RegisterFunction(&RegisteredFunction{
		Name:        "ensure-name-substring",
		Image:       "gcr.io/kpt-fn/ensure-name-substring:v0.1.1",
		Description: "Ensures resource names contain specified substring",
		Version:     "v0.1.1",
		Handler:     r.ensureNameSubstringHandler,
	})

	// O-RAN specific functions
	r.RegisterFunction(&RegisteredFunction{
		Name:        "oran-interface-config",
		Image:       "nephoran.io/kpt-fn/oran-interface-config:v1.0.0",
		Description: "Configures O-RAN interfaces",
		Version:     "v1.0.0",
		Handler:     r.oranInterfaceConfigHandler,
		Schema: &FunctionSchema{
			Properties: map[string]SchemaProperty{
				"interfaces": {
					Type:        "array",
					Description: "List of O-RAN interfaces to configure",
				},
			},
		},
	})

	// Network Slice Config function
	r.RegisterFunction(&RegisteredFunction{
		Name:        "network-slice-config",
		Image:       "nephoran.io/kpt-fn/network-slice-config:v1.0.0",
		Description: "Configures network slices",
		Version:     "v1.0.0",
		Handler:     r.networkSliceConfigHandler,
	})
}

// RegisterFunction registers a new KRM function
func (r *MockKRMRuntime) RegisterFunction(function *RegisteredFunction) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if function.Name == "" {
		return fmt.Errorf("function name is required")
	}

	if function.Handler == nil {
		return fmt.Errorf("function handler is required")
	}

	r.functions[function.Name] = function
	return nil
}

// ExecuteFunction executes a single KRM function
func (r *MockKRMRuntime) ExecuteFunction(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	start := time.Now()

	if r.simulateLatency > 0 {
		time.Sleep(r.simulateLatency)
	}

	if r.simulateFailure {
		return nil, fmt.Errorf("simulated function execution failure")
	}

	// Find function by image
	var function *RegisteredFunction
	r.mutex.RLock()
	for _, fn := range r.functions {
		if fn.Image == req.FunctionConfig.Image {
			function = fn
			break
		}
	}
	r.mutex.RUnlock()

	if function == nil {
		return nil, fmt.Errorf("function not found: %s", req.FunctionConfig.Image)
	}

	// Validate function configuration if validation is enabled
	if r.validationEnabled {
		if err := r.validateFunctionConfig(function, &req.FunctionConfig); err != nil {
			return nil, fmt.Errorf("function configuration validation failed: %w", err)
		}
	}

	// Execute function
	response, err := function.Handler(ctx, req)

	// Record execution history
	event := ExecutionEvent{
		FunctionName: function.Name,
		Request:      req,
		Response:     response,
		Error:        err,
		Timestamp:    start,
		Duration:     time.Since(start),
	}

	r.mutex.Lock()
	r.executionHistory = append(r.executionHistory, event)
	r.mutex.Unlock()

	return response, err
}

// ExecutePipeline executes a pipeline of KRM functions
func (r *MockKRMRuntime) ExecutePipeline(ctx context.Context, req *PipelineRequest) (*PipelineResponse, error) {
	resources := req.Resources
	var allResults []*FunctionResult

	r.logger.Info("Executing pipeline", zap.String("name", req.Pipeline.Name), zap.Int("functions", len(req.Pipeline.Functions)))

	// Execute each function in sequence
	for i, funcConfig := range req.Pipeline.Functions {
		r.logger.Info("Executing function in pipeline", zap.Int("step", i+1), zap.String("image", funcConfig.Image))

		funcReq := &FunctionRequest{
			FunctionConfig: funcConfig,
			Resources:      resources,
			Context:        req.Context,
		}

		funcResp, err := r.ExecuteFunction(ctx, funcReq)
		if err != nil {
			return &PipelineResponse{
				Resources: resources,
				Results:   allResults,
				Error: &FunctionError{
					Message: fmt.Sprintf("pipeline failed at step %d: %v", i+1, err),
					Code:    "PIPELINE_EXECUTION_FAILED",
					Details: fmt.Sprintf("Function: %s", funcConfig.Image),
				},
			}, nil
		}

		// Update resources for next function
		resources = funcResp.Resources

		// Collect results
		if funcResp.Results != nil {
			allResults = append(allResults, funcResp.Results...)
		}

		// Check for function errors
		if funcResp.Error != nil {
			return &PipelineResponse{
				Resources: resources,
				Results:   allResults,
				Error:     funcResp.Error,
			}, nil
		}
	}

	return &PipelineResponse{
		Resources: resources,
		Results:   allResults,
	}, nil
}

// GetExecutionHistory returns the execution history
func (r *MockKRMRuntime) GetExecutionHistory() []ExecutionEvent {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	history := make([]ExecutionEvent, len(r.executionHistory))
	copy(history, r.executionHistory)
	return history
}

// ClearExecutionHistory clears the execution history
func (r *MockKRMRuntime) ClearExecutionHistory() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.executionHistory = []ExecutionEvent{}
}

// SetSimulateFailure enables or disables failure simulation
func (r *MockKRMRuntime) SetSimulateFailure(simulate bool) {
	r.simulateFailure = simulate
}

// SetSimulateLatency sets artificial latency for function execution
func (r *MockKRMRuntime) SetSimulateLatency(latency time.Duration) {
	r.simulateLatency = latency
}

// SetValidationEnabled enables or disables function validation
func (r *MockKRMRuntime) SetValidationEnabled(enabled bool) {
	r.validationEnabled = enabled
}

// Function handlers

// setLabelsHandler implements the set-labels function
func (r *MockKRMRuntime) setLabelsHandler(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	labels, ok := req.FunctionConfig.ConfigMap["labels"].(map[string]interface{})
	if !ok {
		return &FunctionResponse{
			Resources: req.Resources,
			Error: &FunctionError{
				Message: "labels configuration is required",
				Code:    "INVALID_CONFIG",
			},
		}, nil
	}

	// Convert to string map
	stringLabels := make(map[string]string)
	for k, v := range labels {
		if str, ok := v.(string); ok {
			stringLabels[k] = str
		}
	}

	// Apply labels to all resources
	modifiedResources := make([]KRMResource, len(req.Resources))
	for i, resource := range req.Resources {
		modifiedResource := resource
		if modifiedResource.Metadata == nil {
			modifiedResource.Metadata = make(map[string]interface{})
		}

		resourceLabels, exists := modifiedResource.Metadata["labels"]
		if !exists {
			modifiedResource.Metadata["labels"] = stringLabels
		} else {
			// Merge labels
			if existingLabels, ok := resourceLabels.(map[string]interface{}); ok {
				for k, v := range stringLabels {
					existingLabels[k] = v
				}
			}
		}

		modifiedResources[i] = modifiedResource
	}

	return &FunctionResponse{
		Resources: modifiedResources,
		Results: []*FunctionResult{
			{
				Message:  fmt.Sprintf("Applied labels to %d resources", len(modifiedResources)),
				Severity: "info",
			},
		},
	}, nil
}

// setNamespaceHandler implements the set-namespace function
func (r *MockKRMRuntime) setNamespaceHandler(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	namespace, ok := req.FunctionConfig.ConfigMap["namespace"].(string)
	if !ok {
		return &FunctionResponse{
			Resources: req.Resources,
			Error: &FunctionError{
				Message: "namespace configuration is required",
				Code:    "INVALID_CONFIG",
			},
		}, nil
	}

	modifiedResources := make([]KRMResource, len(req.Resources))
	for i, resource := range req.Resources {
		modifiedResource := resource
		if modifiedResource.Metadata == nil {
			modifiedResource.Metadata = make(map[string]interface{})
		}
		modifiedResource.Metadata["namespace"] = namespace
		modifiedResources[i] = modifiedResource
	}

	return &FunctionResponse{
		Resources: modifiedResources,
		Results: []*FunctionResult{
			{
				Message:  fmt.Sprintf("Set namespace to %s for %d resources", namespace, len(modifiedResources)),
				Severity: "info",
			},
		},
	}, nil
}

// applyReplacementsHandler implements the apply-replacements function
func (r *MockKRMRuntime) applyReplacementsHandler(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	// Simulate apply-replacements logic
	return &FunctionResponse{
		Resources: req.Resources,
		Results: []*FunctionResult{
			{
				Message:  "Applied replacements to resources",
				Severity: "info",
			},
		},
	}, nil
}

// searchReplaceHandler implements the search-replace function
func (r *MockKRMRuntime) searchReplaceHandler(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	// Simulate search-replace logic
	return &FunctionResponse{
		Resources: req.Resources,
		Results: []*FunctionResult{
			{
				Message:  "Performed search and replace operations",
				Severity: "info",
			},
		},
	}, nil
}

// ensureNameSubstringHandler implements the ensure-name-substring function
func (r *MockKRMRuntime) ensureNameSubstringHandler(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	substring, ok := req.FunctionConfig.ConfigMap["substring"].(string)
	if !ok {
		return &FunctionResponse{
			Resources: req.Resources,
			Error: &FunctionError{
				Message: "substring configuration is required",
				Code:    "INVALID_CONFIG",
			},
		}, nil
	}

	modifiedResources := make([]KRMResource, len(req.Resources))
	for i, resource := range req.Resources {
		modifiedResource := resource
		if metadata := modifiedResource.Metadata; metadata != nil {
			if name, exists := metadata["name"]; exists {
				if nameStr, ok := name.(string); ok && !strings.Contains(nameStr, substring) {
					metadata["name"] = nameStr + "-" + substring
				}
			}
		}
		modifiedResources[i] = modifiedResource
	}

	return &FunctionResponse{
		Resources: modifiedResources,
		Results: []*FunctionResult{
			{
				Message:  fmt.Sprintf("Ensured substring '%s' in resource names", substring),
				Severity: "info",
			},
		},
	}, nil
}

// oranInterfaceConfigHandler implements O-RAN interface configuration
func (r *MockKRMRuntime) oranInterfaceConfigHandler(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	interfaces, ok := req.FunctionConfig.ConfigMap["interfaces"].([]interface{})
	if !ok {
		return &FunctionResponse{
			Resources: req.Resources,
			Error: &FunctionError{
				Message: "interfaces configuration is required",
				Code:    "INVALID_CONFIG",
			},
		}, nil
	}

	// Process O-RAN interface configuration
	modifiedResources := req.Resources

	for _, iface := range interfaces {
		if ifaceMap, ok := iface.(map[string]interface{}); ok {
			// Add O-RAN specific annotations and labels
			for i, resource := range modifiedResources {
				if resource.Metadata == nil {
					resource.Metadata = make(map[string]interface{})
				}

				// Add O-RAN annotations
				annotations := make(map[string]interface{})
				if existing, exists := resource.Metadata["annotations"]; exists {
					if existingMap, ok := existing.(map[string]interface{}); ok {
						annotations = existingMap
					}
				}

				if ifaceName, exists := ifaceMap["name"]; exists {
					annotations["oran.nephoran.com/interface"] = ifaceName
				}
				if ifaceType, exists := ifaceMap["type"]; exists {
					annotations["oran.nephoran.com/interface-type"] = ifaceType
				}

				resource.Metadata["annotations"] = annotations
				modifiedResources[i] = resource
			}
		}
	}

	return &FunctionResponse{
		Resources: modifiedResources,
		Results: []*FunctionResult{
			{
				Message:  fmt.Sprintf("Configured %d O-RAN interfaces", len(interfaces)),
				Severity: "info",
			},
		},
	}, nil
}

// networkSliceConfigHandler implements network slice configuration
func (r *MockKRMRuntime) networkSliceConfigHandler(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	sliceConfig, ok := req.FunctionConfig.ConfigMap["sliceConfig"].(map[string]interface{})
	if !ok {
		return &FunctionResponse{
			Resources: req.Resources,
			Error: &FunctionError{
				Message: "sliceConfig is required",
				Code:    "INVALID_CONFIG",
			},
		}, nil
	}

	modifiedResources := make([]KRMResource, len(req.Resources))
	for i, resource := range req.Resources {
		modifiedResource := resource

		// Add network slice configuration
		if modifiedResource.Spec == nil {
			modifiedResource.Spec = make(map[string]interface{})
		}
		modifiedResource.Spec["networkSlice"] = sliceConfig

		modifiedResources[i] = modifiedResource
	}

	return &FunctionResponse{
		Resources: modifiedResources,
		Results: []*FunctionResult{
			{
				Message:  "Applied network slice configuration",
				Severity: "info",
			},
		},
	}, nil
}

// Helper functions

// validateFunctionConfig validates function configuration against schema
func (r *MockKRMRuntime) validateFunctionConfig(function *RegisteredFunction, config *FunctionConfig) error {
	if function.Schema == nil {
		return nil // No schema to validate against
	}

	// Check required fields
	for _, required := range function.Schema.Required {
		if _, exists := config.ConfigMap[required]; !exists {
			return fmt.Errorf("required field '%s' is missing", required)
		}
	}

	return nil
}

// generateTestResource creates a test KRM resource
func generateTestResource(apiVersion, kind, name, namespace string) KRMResource {
	return KRMResource{
		APIVersion: apiVersion,
		Kind:       kind,
		Metadata: json.RawMessage(`{}`),
		Spec: json.RawMessage(`{}`),
	}
}

// generateTestResourceWithLabels creates a test KRM resource with labels
func generateTestResourceWithLabels(apiVersion, kind, name, namespace string, labels map[string]string) KRMResource {
	resource := generateTestResource(apiVersion, kind, name, namespace)
	resource.Metadata["labels"] = labels
	return resource
}

// Test Cases

// TestKRMRuntimeCreation tests KRM runtime creation
func TestKRMRuntimeCreation(t *testing.T) {
	runtime := NewMockKRMRuntime()

	assert.NotNil(t, runtime)
	assert.NotNil(t, runtime.client)
	assert.NotNil(t, runtime.k8sClient)
	assert.NotNil(t, runtime.logger)
	assert.True(t, len(runtime.functions) > 0)
	assert.True(t, runtime.validationEnabled)
}

// TestFunctionRegistration tests function registration
func TestFunctionRegistration(t *testing.T) {
	runtime := NewMockKRMRuntime()

	testFunction := &RegisteredFunction{
		Name:        "test-function",
		Image:       "test/test-function:v1.0.0",
		Description: "Test function",
		Handler: func(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
			return &FunctionResponse{
				Resources: req.Resources,
				Results: []*FunctionResult{
					{Message: "test function executed", Severity: "info"},
				},
			}, nil
		},
	}

	err := runtime.RegisterFunction(testFunction)
	assert.NoError(t, err)

	// Verify function was registered
	runtime.mutex.RLock()
	registeredFunc, exists := runtime.functions["test-function"]
	runtime.mutex.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, testFunction.Name, registeredFunc.Name)
	assert.Equal(t, testFunction.Image, registeredFunc.Image)
}

// TestFunctionRegistrationValidation tests function registration validation
func TestFunctionRegistrationValidation(t *testing.T) {
	runtime := NewMockKRMRuntime()

	testCases := []struct {
		name        string
		function    *RegisteredFunction
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing_name",
			function: &RegisteredFunction{
				Image:   "test/function:v1.0.0",
				Handler: func(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) { return nil, nil },
			},
			expectError: true,
			errorMsg:    "function name is required",
		},
		{
			name: "missing_handler",
			function: &RegisteredFunction{
				Name:  "test-function",
				Image: "test/function:v1.0.0",
			},
			expectError: true,
			errorMsg:    "function handler is required",
		},
		{
			name: "valid_function",
			function: &RegisteredFunction{
				Name:  "valid-function",
				Image: "test/function:v1.0.0",
				Handler: func(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
					return &FunctionResponse{Resources: req.Resources}, nil
				},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := runtime.RegisterFunction(tc.function)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestStandardFunctions tests standard KRM functions
func TestStandardFunctions(t *testing.T) {
	runtime := NewMockKRMRuntime()
	ctx := context.Background()

	t.Run("set_labels", func(t *testing.T) {
		resources := []KRMResource{
			generateTestResource("apps/v1", "Deployment", "test-deployment", "default"),
			generateTestResource("v1", "Service", "test-service", "default"),
		}

		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: "gcr.io/kpt-fn/set-labels:v0.2.0",
				ConfigMap: map[string]interface{}{
					"app": "test-app",
					"env": "production",
				},
			},
			Resources: resources,
		}

		resp, err := runtime.ExecuteFunction(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, len(resources), len(resp.Resources))
		assert.True(t, len(resp.Results) > 0)

		// Verify labels were added
		for _, resource := range resp.Resources {
			labels, exists := resource.Metadata["labels"]
			assert.True(t, exists)

			if labelMap, ok := labels.(map[string]interface{}); ok {
				assert.Equal(t, "test-app", labelMap["app"])
				assert.Equal(t, "production", labelMap["env"])
			}
		}
	})

	t.Run("set_namespace", func(t *testing.T) {
		resources := []KRMResource{
			generateTestResource("apps/v1", "Deployment", "test-deployment", "default"),
		}

		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: "gcr.io/kpt-fn/set-namespace:v0.4.1",
				ConfigMap: json.RawMessage(`{}`),
			},
			Resources: resources,
		}

		resp, err := runtime.ExecuteFunction(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, len(resources), len(resp.Resources))

		// Verify namespace was set
		for _, resource := range resp.Resources {
			namespace, exists := resource.Metadata["namespace"]
			assert.True(t, exists)
			assert.Equal(t, "production", namespace)
		}
	})

	t.Run("ensure_name_substring", func(t *testing.T) {
		resources := []KRMResource{
			generateTestResource("apps/v1", "Deployment", "deployment", "default"),
		}

		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: "gcr.io/kpt-fn/ensure-name-substring:v0.1.1",
				ConfigMap: json.RawMessage(`{}`),
			},
			Resources: resources,
		}

		resp, err := runtime.ExecuteFunction(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify substring was added to name
		for _, resource := range resp.Resources {
			name, exists := resource.Metadata["name"]
			assert.True(t, exists)
			assert.Contains(t, name.(string), "prod")
		}
	})
}

// TestORANFunctions tests O-RAN specific functions
func TestORANFunctions(t *testing.T) {
	runtime := NewMockKRMRuntime()
	ctx := context.Background()

	t.Run("oran_interface_config", func(t *testing.T) {
		resources := []KRMResource{
			generateTestResource("apps/v1", "Deployment", "amf-deployment", "5g-core"),
		}

		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: "nephoran.io/kpt-fn/oran-interface-config:v1.0.0",
				ConfigMap: map[string]interface{}{
					"interfaces": []interface{}{},
				},
			},
			Resources: resources,
		}

		resp, err := runtime.ExecuteFunction(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, len(resp.Results) > 0)

		// Verify O-RAN annotations were added
		for _, resource := range resp.Resources {
			annotations, exists := resource.Metadata["annotations"]
			assert.True(t, exists)

			if annotMap, ok := annotations.(map[string]interface{}); ok {
				assert.Contains(t, annotMap, "oran.nephoran.com/interface")
				assert.Contains(t, annotMap, "oran.nephoran.com/interface-type")
			}
		}
	})

	t.Run("network_slice_config", func(t *testing.T) {
		resources := []KRMResource{
			generateTestResource("apps/v1", "Deployment", "slice-deployment", "5g-core"),
		}

		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: "nephoran.io/kpt-fn/network-slice-config:v1.0.0",
				ConfigMap: map[string]interface{}{
					"sliceType": "eMBB",
					"sst":       1,
					"sd":        "000001",
				},
			},
			Resources: resources,
		}

		resp, err := runtime.ExecuteFunction(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify network slice configuration was added
		for _, resource := range resp.Resources {
			networkSlice, exists := resource.Spec["networkSlice"]
			assert.True(t, exists)

			if sliceMap, ok := networkSlice.(map[string]interface{}); ok {
				assert.Equal(t, "eMBB", sliceMap["sliceType"])
				assert.Equal(t, 1, sliceMap["sst"])
				assert.Equal(t, "000001", sliceMap["sd"])
			}
		}
	})
}

// TestPipelineExecution tests pipeline execution
func TestPipelineExecution(t *testing.T) {
	runtime := NewMockKRMRuntime()
	ctx := context.Background()

	resources := []KRMResource{
		generateTestResource("apps/v1", "Deployment", "test-deployment", "default"),
		generateTestResource("v1", "Service", "test-service", "default"),
	}

	pipeline := Pipeline{
		Name: "standard-pipeline",
		Functions: []FunctionConfig{
			{
				Image: "gcr.io/kpt-fn/set-namespace:v0.4.1",
				ConfigMap: json.RawMessage(`{}`),
			},
			{
				Image: "gcr.io/kpt-fn/set-labels:v0.2.0",
				ConfigMap: map[string]interface{}{
					"app": "test-app",
					"env": "production",
				},
			},
		},
	}

	req := &PipelineRequest{
		Pipeline:  pipeline,
		Resources: resources,
		Context: &FunctionContext{
			PackagePath: "/test/package",
			Namespace:   "test",
		},
	}

	resp, err := runtime.ExecutePipeline(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, len(resources), len(resp.Resources))
	assert.True(t, len(resp.Results) > 0)
	assert.Nil(t, resp.Error)

	// Verify all functions were applied
	for _, resource := range resp.Resources {
		// Check namespace was set
		namespace, exists := resource.Metadata["namespace"]
		assert.True(t, exists)
		assert.Equal(t, "production", namespace)

		// Check labels were set
		labels, exists := resource.Metadata["labels"]
		assert.True(t, exists)

		if labelMap, ok := labels.(map[string]interface{}); ok {
			assert.Equal(t, "test-app", labelMap["app"])
			assert.Equal(t, "production", labelMap["env"])
		}
	}
}

// TestPipelineExecutionFailure tests pipeline execution with failures
func TestPipelineExecutionFailure(t *testing.T) {
	runtime := NewMockKRMRuntime()
	ctx := context.Background()

	// Register a failing function
	runtime.RegisterFunction(&RegisteredFunction{
		Name:  "failing-function",
		Image: "test/failing-function:v1.0.0",
		Handler: func(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
			return nil, fmt.Errorf("function failed")
		},
	})

	resources := []KRMResource{
		generateTestResource("apps/v1", "Deployment", "test-deployment", "default"),
	}

	pipeline := Pipeline{
		Name: "failing-pipeline",
		Functions: []FunctionConfig{
			{
				Image: "gcr.io/kpt-fn/set-namespace:v0.4.1",
				ConfigMap: json.RawMessage(`{}`),
			},
			{
				Image: "test/failing-function:v1.0.0",
			},
		},
	}

	req := &PipelineRequest{
		Pipeline:  pipeline,
		Resources: resources,
	}

	resp, err := runtime.ExecutePipeline(ctx, req)
	assert.NoError(t, err) // Pipeline execution itself doesn't fail
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Contains(t, resp.Error.Message, "pipeline failed at step 2")
}

// TestFunctionValidation tests function configuration validation
func TestFunctionValidation(t *testing.T) {
	runtime := NewMockKRMRuntime()
	ctx := context.Background()

	t.Run("valid_configuration", func(t *testing.T) {
		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: "gcr.io/kpt-fn/set-labels:v0.2.0",
				ConfigMap: map[string]interface{}{
					"app": "test",
				},
			},
			Resources: []KRMResource{
				generateTestResource("apps/v1", "Deployment", "test", "default"),
			},
		}

		resp, err := runtime.ExecuteFunction(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Nil(t, resp.Error)
	})

	t.Run("invalid_configuration", func(t *testing.T) {
		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image:     "gcr.io/kpt-fn/set-labels:v0.2.0",
				ConfigMap: json.RawMessage(`{}`), // Missing required 'labels' field
			},
			Resources: []KRMResource{
				generateTestResource("apps/v1", "Deployment", "test", "default"),
			},
		}

		resp, err := runtime.ExecuteFunction(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required field 'labels' is missing")
		assert.Nil(t, resp, "Response should be nil when function execution fails")
	})

	t.Run("validation_disabled", func(t *testing.T) {
		runtime.SetValidationEnabled(false)

		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image:     "gcr.io/kpt-fn/set-labels:v0.2.0",
				ConfigMap: json.RawMessage(`{}`), // Missing required field, but validation is disabled
			},
			Resources: []KRMResource{
				generateTestResource("apps/v1", "Deployment", "test", "default"),
			},
		}

		resp, err := runtime.ExecuteFunction(ctx, req)
		// Should fail in the handler, not validation
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Error) // Function should return error in response

		// Re-enable validation
		runtime.SetValidationEnabled(true)
	})
}

// TestConcurrentExecution tests concurrent function execution
func TestConcurrentExecution(t *testing.T) {
	runtime := NewMockKRMRuntime()
	ctx := context.Background()

	const numGoroutines = 10
	const operationsPerGoroutine = 5

	resources := []KRMResource{
		generateTestResource("apps/v1", "Deployment", "test-deployment", "default"),
	}

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				req := &FunctionRequest{
					FunctionConfig: FunctionConfig{
						Image: "gcr.io/kpt-fn/set-labels:v0.2.0",
						ConfigMap: map[string]interface{}{
							"worker": fmt.Sprintf("worker-%d", id),
							"job":    fmt.Sprintf("job-%d", j),
						},
					},
					Resources: resources,
				}

				_, err := runtime.ExecuteFunction(ctx, req)
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent execution failed: %v", err)
			errorCount++
		}
	}

	assert.Equal(t, 0, errorCount, "Expected no errors in concurrent execution")

	// Verify execution history
	history := runtime.GetExecutionHistory()
	assert.Equal(t, numGoroutines*operationsPerGoroutine, len(history))
}

// TestExecutionHistory tests execution history tracking
func TestExecutionHistory(t *testing.T) {
	runtime := NewMockKRMRuntime()
	ctx := context.Background()

	// Clear any existing history
	runtime.ClearExecutionHistory()

	resources := []KRMResource{
		generateTestResource("apps/v1", "Deployment", "test-deployment", "default"),
	}

	// Execute multiple functions
	functions := []string{
		"gcr.io/kpt-fn/set-labels:v0.2.0",
		"gcr.io/kpt-fn/set-namespace:v0.4.1",
	}

	for i, funcImage := range functions {
		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: funcImage,
				ConfigMap: map[string]interface{}{
					"namespace": "test-namespace",
				},
			},
			Resources: resources,
		}

		start := time.Now()
		_, err := runtime.ExecuteFunction(ctx, req)
		assert.NoError(t, err)

		// Verify execution was recorded
		history := runtime.GetExecutionHistory()
		assert.Equal(t, i+1, len(history))

		lastExecution := history[len(history)-1]
		assert.NotEmpty(t, lastExecution.FunctionName)
		assert.NotNil(t, lastExecution.Request)
		assert.NotNil(t, lastExecution.Response)
		assert.True(t, lastExecution.Timestamp.After(start) || lastExecution.Timestamp.Equal(start))
		assert.True(t, lastExecution.Duration > 0)
	}

	// Test clearing history
	runtime.ClearExecutionHistory()
	history := runtime.GetExecutionHistory()
	assert.Equal(t, 0, len(history))
}

// TestErrorHandling tests error handling scenarios
func TestErrorHandling(t *testing.T) {
	runtime := NewMockKRMRuntime()
	ctx := context.Background()

	t.Run("unknown_function", func(t *testing.T) {
		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: "unknown/function:v1.0.0",
			},
			Resources: []KRMResource{
				generateTestResource("apps/v1", "Deployment", "test", "default"),
			},
		}

		_, err := runtime.ExecuteFunction(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function not found")
	})

	t.Run("simulated_failure", func(t *testing.T) {
		runtime.SetSimulateFailure(true)

		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: "gcr.io/kpt-fn/set-labels:v0.2.0",
			},
			Resources: []KRMResource{
				generateTestResource("apps/v1", "Deployment", "test", "default"),
			},
		}

		_, err := runtime.ExecuteFunction(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "simulated function execution failure")

		// Disable failure simulation
		runtime.SetSimulateFailure(false)
	})

	t.Run("context_cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: "gcr.io/kpt-fn/set-labels:v0.2.0",
				ConfigMap: json.RawMessage(`{}`),
			},
			Resources: []KRMResource{
				generateTestResource("apps/v1", "Deployment", "test", "default"),
			},
		}

		// The mock doesn't actually check context cancellation,
		// but in a real implementation it would
		_, err := runtime.ExecuteFunction(cancelCtx, req)

		// For the mock, we'll just verify it doesn't panic
		_ = err
	})
}

// TestPerformanceCharacteristics tests performance aspects
func TestPerformanceCharacteristics(t *testing.T) {
	runtime := NewMockKRMRuntime()
	ctx := context.Background()

	t.Run("latency_simulation", func(t *testing.T) {
		latency := 100 * time.Millisecond
		runtime.SetSimulateLatency(latency)

		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: "gcr.io/kpt-fn/set-labels:v0.2.0",
				ConfigMap: json.RawMessage(`{}`),
			},
			Resources: []KRMResource{
				generateTestResource("apps/v1", "Deployment", "test", "default"),
			},
		}

		start := time.Now()
		_, err := runtime.ExecuteFunction(ctx, req)
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.True(t, elapsed >= latency, "Expected execution to take at least %v, took %v", latency, elapsed)

		// Reset latency
		runtime.SetSimulateLatency(0)
	})
}

// BenchmarkFunctionExecution benchmarks function execution performance
func BenchmarkFunctionExecution(b *testing.B) {
	runtime := NewMockKRMRuntime()
	ctx := context.Background()

	resources := []KRMResource{
		generateTestResource("apps/v1", "Deployment", "test-deployment", "default"),
		generateTestResource("v1", "Service", "test-service", "default"),
	}

	req := &FunctionRequest{
		FunctionConfig: FunctionConfig{
			Image: "gcr.io/kpt-fn/set-labels:v0.2.0",
			ConfigMap: map[string]interface{}{
				"app": "test-app",
				"env": "production",
			},
		},
		Resources: resources,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := runtime.ExecuteFunction(ctx, req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkPipelineExecution benchmarks pipeline execution performance
func BenchmarkPipelineExecution(b *testing.B) {
	runtime := NewMockKRMRuntime()
	ctx := context.Background()

	resources := []KRMResource{
		generateTestResource("apps/v1", "Deployment", "test-deployment", "default"),
		generateTestResource("v1", "Service", "test-service", "default"),
	}

	pipeline := Pipeline{
		Name: "benchmark-pipeline",
		Functions: []FunctionConfig{
			{
				Image: "gcr.io/kpt-fn/set-namespace:v0.4.1",
				ConfigMap: json.RawMessage(`{}`),
			},
			{
				Image: "gcr.io/kpt-fn/set-labels:v0.2.0",
				ConfigMap: map[string]interface{}{
					"app": "test-app",
					"env": "production",
				},
			},
		},
	}

	req := &PipelineRequest{
		Pipeline:  pipeline,
		Resources: resources,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := runtime.ExecutePipeline(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

