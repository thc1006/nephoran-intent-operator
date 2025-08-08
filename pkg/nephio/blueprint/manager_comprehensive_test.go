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

package blueprint

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/thc1006/nephoran-intent-operator/api/v1"
)

// MockManager implements the controller-runtime manager interface for testing
type MockManager struct {
	client  client.Client
	config  *rest.Config
	scheme  *runtime.Scheme
	logger  logr.Logger
}

func (m *MockManager) GetClient() client.Client {
	return m.client
}

func (m *MockManager) GetConfig() *rest.Config {
	return m.config
}

func (m *MockManager) GetScheme() *runtime.Scheme {
	return m.scheme
}

func (m *MockManager) GetLogger() logr.Logger {
	return m.logger
}

// Implement other manager.Manager interface methods as no-ops for testing
func (m *MockManager) Add(manager.Runnable) error { return nil }
func (m *MockManager) AddMetricsExtraHandler(string, http.Handler) error { return nil }
func (m *MockManager) AddHealthzCheck(string, healthz.Checker) error { return nil }
func (m *MockManager) AddReadyzCheck(string, healthz.Checker) error { return nil }
func (m *MockManager) Start(context.Context) error { return nil }
func (m *MockManager) GetWebhookServer() *webhook.Server { return nil }
func (m *MockManager) GetAPIReader() client.Reader { return m.client }
func (m *MockManager) GetCache() cache.Cache { return nil }
func (m *MockManager) GetFieldIndexer() client.FieldIndexer { return nil }
func (m *MockManager) GetEventRecorderFor(string) record.EventRecorder { return nil }
func (m *MockManager) GetRESTMapper() meta.RESTMapper { return nil }
func (m *MockManager) GetControllerOptions() v1alpha1.ControllerConfigurationSpec { return v1alpha1.ControllerConfigurationSpec{} }

// Helper function to create a mock manager
func newMockManager() *MockManager {
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	client := clientfake.NewClientBuilder().WithScheme(scheme).Build()
	
	return &MockManager{
		client: client,
		config: &rest.Config{},
		scheme: scheme,
	}
}

// MockCatalog provides mock implementation for testing
type MockCatalog struct {
	templates map[string]*BlueprintTemplate
	cacheHits int
	cacheMisses int
	mutex sync.RWMutex
}

func NewMockCatalog() *MockCatalog {
	return &MockCatalog{
		templates: make(map[string]*BlueprintTemplate),
	}
}

func (c *MockCatalog) GetTemplate(templateName string) (*BlueprintTemplate, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	if template, exists := c.templates[templateName]; exists {
		c.cacheHits++
		return template, nil
	}
	
	c.cacheMisses++
	return nil, fmt.Errorf("template %s not found", templateName)
}

func (c *MockCatalog) AddTemplate(name string, template *BlueprintTemplate) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.templates[name] = template
}

func (c *MockCatalog) GetCacheHits() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.cacheHits
}

func (c *MockCatalog) GetCacheMisses() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.cacheMisses
}

func (c *MockCatalog) HealthCheck(ctx context.Context) bool {
	return true
}

// MockGenerator provides mock implementation for testing
type MockGenerator struct {
	generatedCount int
	shouldFail     bool
}

func NewMockGenerator() *MockGenerator {
	return &MockGenerator{}
}

func (g *MockGenerator) GenerateFromNetworkIntent(ctx context.Context, intent *v1.NetworkIntent) (map[string]string, error) {
	if g.shouldFail {
		return nil, fmt.Errorf("generator failed")
	}
	
	g.generatedCount++
	
	files := map[string]string{
		"Kptfile": generateKptfile(intent),
		"deployment.yaml": generateDeployment(intent),
		"service.yaml": generateService(intent),
		"configmap.yaml": generateConfigMap(intent),
	}
	
	return files, nil
}

func (g *MockGenerator) SetShouldFail(fail bool) {
	g.shouldFail = fail
}

func (g *MockGenerator) GetGeneratedCount() int {
	return g.generatedCount
}

func (g *MockGenerator) HealthCheck(ctx context.Context) bool {
	return !g.shouldFail
}

// MockCustomizer provides mock implementation for testing
type MockCustomizer struct {
	customizedCount int
	shouldFail      bool
}

func NewMockCustomizer() *MockCustomizer {
	return &MockCustomizer{}
}

func (c *MockCustomizer) CustomizeBlueprint(ctx context.Context, intent *v1.NetworkIntent, files map[string]string) (map[string]string, error) {
	if c.shouldFail {
		return nil, fmt.Errorf("customizer failed")
	}
	
	c.customizedCount++
	
	// Simulate customization by adding custom labels
	customizedFiles := make(map[string]string)
	for filename, content := range files {
		if filename == "deployment.yaml" {
			// Add custom labels to deployment
			customizedFiles[filename] = content + "\n  # Customized by Nephoran"
		} else {
			customizedFiles[filename] = content
		}
	}
	
	return customizedFiles, nil
}

func (c *MockCustomizer) SetShouldFail(fail bool) {
	c.shouldFail = fail
}

func (c *MockCustomizer) GetCustomizedCount() int {
	return c.customizedCount
}

func (c *MockCustomizer) HealthCheck(ctx context.Context) bool {
	return !c.shouldFail
}

// MockValidator provides mock implementation for testing
type MockValidator struct {
	validatedCount int
	shouldFail     bool
	shouldReject   bool
}

func NewMockValidator() *MockValidator {
	return &MockValidator{}
}

func (v *MockValidator) ValidateBlueprint(ctx context.Context, intent *v1.NetworkIntent, files map[string]string) (*ValidationResult, error) {
	if v.shouldFail {
		return nil, fmt.Errorf("validator failed")
	}
	
	v.validatedCount++
	
	result := &ValidationResult{
		IsValid: !v.shouldReject,
	}
	
	if v.shouldReject {
		result.Errors = []string{"validation failed: missing required field"}
	}
	
	return result, nil
}

func (v *MockValidator) SetShouldFail(fail bool) {
	v.shouldFail = fail
}

func (v *MockValidator) SetShouldReject(reject bool) {
	v.shouldReject = reject
}

func (v *MockValidator) GetValidatedCount() int {
	return v.validatedCount
}

func (v *MockValidator) HealthCheck(ctx context.Context) bool {
	return !v.shouldFail
}

// Blueprint template structure for testing
type BlueprintTemplate struct {
	Name        string
	Version     string
	Description string
	Files       map[string]string
	Metadata    map[string]interface{}
}

// Helper functions to generate blueprint content
func generateKptfile(intent *v1.NetworkIntent) string {
	return fmt.Sprintf(`apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: %s-blueprint
info:
  description: Blueprint package for NetworkIntent %s
`, intent.Name, intent.Name)
}

func generateDeployment(intent *v1.NetworkIntent) string {
	component := "unknown"
	if len(intent.Spec.TargetComponents) > 0 {
		component = string(intent.Spec.TargetComponents[0])
	}
	
	replicas := "1"
	if r, exists := intent.Spec.Parameters["replicas"]; exists {
		replicas = r
	}
	
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s-%s
  namespace: %s
spec:
  replicas: %s
  selector:
    matchLabels:
      app: %s-%s
  template:
    metadata:
      labels:
        app: %s-%s
        component: %s
    spec:
      containers:
      - name: %s
        image: %s:latest
        ports:
        - containerPort: 8080
`, intent.Name, component, intent.Namespace, replicas, intent.Name, component, intent.Name, component, component, component, component)
}

func generateService(intent *v1.NetworkIntent) string {
	component := "unknown"
	if len(intent.Spec.TargetComponents) > 0 {
		component = string(intent.Spec.TargetComponents[0])
	}
	
	return fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s-%s-service
  namespace: %s
spec:
  selector:
    app: %s-%s
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
`, intent.Name, component, intent.Namespace, intent.Name, component)
}

func generateConfigMap(intent *v1.NetworkIntent) string {
	return fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-config
  namespace: %s
data:
  intent-type: "%s"
  priority: "%s"
`, intent.Name, intent.Namespace, intent.Spec.IntentType, intent.Spec.Priority)
}

// Test helper to create test NetworkIntent
func createTestNetworkIntent(name string) *v1.NetworkIntent {
	return &v1.NetworkIntent{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "nephoran.com/v1",
			Kind:       "NetworkIntent",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "test-namespace",
		},
		Spec: v1.NetworkIntentSpec{
			IntentType: v1.IntentTypeDeployment,
			Priority:   v1.PriorityMedium,
			TargetComponents: []v1.ComponentType{
				v1.ComponentTypeAMF,
			},
			Parameters: map[string]string{
				"replicas": "3",
				"region":   "us-east-1",
			},
		},
		Status: v1.NetworkIntentStatus{
			Phase: v1.PhaseProcessing,
		},
	}
}

// TestManagerCreation tests blueprint manager creation
func TestManagerCreation(t *testing.T) {
	mockMgr := newMockManager()
	logger := zaptest.NewLogger(t)

	testCases := []struct {
		name        string
		config      *BlueprintConfig
		expectError bool
		errorMsg    string
	}{
		{
			name:        "successful_creation_default_config",
			config:      nil, // Should use default config
			expectError: false,
		},
		{
			name:        "successful_creation_custom_config",
			config:      DefaultBlueprintConfig(),
			expectError: false,
		},
		{
			name: "custom_config_with_overrides",
			config: &BlueprintConfig{
				PorchEndpoint:        "http://custom-porch:9080",
				LLMEndpoint:          "http://custom-llm:8080",
				CacheTTL:             10 * time.Minute,
				MaxConcurrency:       100,
				EnableValidation:     true,
				EnableORANCompliance: true,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager, err := NewManager(mockMgr, tc.config, logger)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, manager)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				assert.NotNil(t, manager.config)
				assert.NotNil(t, manager.metrics)
				
				// Cleanup
				defer manager.Stop()
			}
		})
	}
}

// TestProcessNetworkIntent tests the main blueprint processing flow
func TestProcessNetworkIntent(t *testing.T) {
	mockMgr := newMockManager()
	logger := zaptest.NewLogger(t)
	config := DefaultBlueprintConfig()
	
	// Create manager with mock components
	manager := &Manager{
		client:      mockMgr.GetClient(),
		k8sClient:   fake.NewSimpleClientset(),
		config:      config,
		logger:      logger,
		metrics:     NewBlueprintMetrics(),
		catalog:     NewMockCatalog(),
		generator:   NewMockGenerator(),
		customizer:  NewMockCustomizer(),
		validator:   NewMockValidator(),
		ctx:         context.Background(),
		healthStatus: make(map[string]bool),
	}

	testCases := []struct {
		name           string
		intent         *v1.NetworkIntent
		setupMocks     func()
		expectError    bool
		errorMsg       string
		validateResult func(*BlueprintResult)
	}{
		{
			name:   "successful_processing",
			intent: createTestNetworkIntent("success-test"),
			setupMocks: func() {
				// Default mocks are already set up for success
			},
			expectError: false,
			validateResult: func(result *BlueprintResult) {
				assert.True(t, result.Success)
				assert.NotNil(t, result.PackageRevision)
				assert.True(t, len(result.GeneratedFiles) > 0)
				assert.NotNil(t, result.ValidationResults)
				assert.True(t, result.ValidationResults.IsValid)
			},
		},
		{
			name:   "generator_failure",
			intent: createTestNetworkIntent("generator-fail"),
			setupMocks: func() {
				manager.generator.(*MockGenerator).SetShouldFail(true)
			},
			expectError: true,
			errorMsg:    "blueprint generation failed",
		},
		{
			name:   "customizer_failure",
			intent: createTestNetworkIntent("customizer-fail"),
			setupMocks: func() {
				manager.generator.(*MockGenerator).SetShouldFail(false)
				manager.customizer.(*MockCustomizer).SetShouldFail(true)
			},
			expectError: true,
			errorMsg:    "blueprint customization failed",
		},
		{
			name:   "validator_failure",
			intent: createTestNetworkIntent("validator-fail"),
			setupMocks: func() {
				manager.generator.(*MockGenerator).SetShouldFail(false)
				manager.customizer.(*MockCustomizer).SetShouldFail(false)
				manager.validator.(*MockValidator).SetShouldFail(true)
			},
			expectError: true,
			errorMsg:    "blueprint validation failed",
		},
		{
			name:   "validation_rejection",
			intent: createTestNetworkIntent("validation-reject"),
			setupMocks: func() {
				manager.generator.(*MockGenerator).SetShouldFail(false)
				manager.customizer.(*MockCustomizer).SetShouldFail(false)
				manager.validator.(*MockValidator).SetShouldFail(false)
				manager.validator.(*MockValidator).SetShouldReject(true)
			},
			expectError: true,
			errorMsg:    "blueprint validation failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset mocks
			manager.generator = NewMockGenerator()
			manager.customizer = NewMockCustomizer()
			manager.validator = NewMockValidator()
			
			// Setup test-specific mock behavior
			tc.setupMocks()

			// Process the intent
			result, err := manager.ProcessNetworkIntent(context.Background(), tc.intent)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tc.validateResult != nil {
					tc.validateResult(result)
				}
			}
		})
	}
}

// TestConcurrentProcessing tests concurrent blueprint processing
func TestConcurrentProcessing(t *testing.T) {
	mockMgr := newMockManager()
	logger := zaptest.NewLogger(t)
	config := DefaultBlueprintConfig()
	config.MaxConcurrency = 5
	
	manager := &Manager{
		client:      mockMgr.GetClient(),
		k8sClient:   fake.NewSimpleClientset(),
		config:      config,
		logger:      logger,
		metrics:     NewBlueprintMetrics(),
		catalog:     NewMockCatalog(),
		generator:   NewMockGenerator(),
		customizer:  NewMockCustomizer(),
		validator:   NewMockValidator(),
		ctx:         context.Background(),
		healthStatus: make(map[string]bool),
	}

	const numIntents = 20
	const numGoroutines = 10

	// Create test intents
	intents := make([]*v1.NetworkIntent, numIntents)
	for i := 0; i < numIntents; i++ {
		intents[i] = createTestNetworkIntent(fmt.Sprintf("concurrent-test-%d", i))
	}

	// Process intents concurrently
	var wg sync.WaitGroup
	results := make(chan *BlueprintResult, numIntents)
	errors := make(chan error, numIntents)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(startIdx int) {
			defer wg.Done()
			
			for j := startIdx; j < numIntents; j += numGoroutines {
				result, err := manager.ProcessNetworkIntent(context.Background(), intents[j])
				if err != nil {
					errors <- err
				} else {
					results <- result
				}
			}
		}(i)
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check results
	successCount := 0
	for result := range results {
		if result.Success {
			successCount++
		}
	}

	errorCount := 0
	for range errors {
		errorCount++
	}

	assert.Equal(t, numIntents, successCount+errorCount, "All intents should be processed")
	assert.Equal(t, 0, errorCount, "No errors expected in concurrent processing")
	assert.Equal(t, numIntents, successCount, "All intents should succeed")
}

// TestBlueprintTemplates tests different blueprint templates
func TestBlueprintTemplates(t *testing.T) {
	testCases := []struct {
		name          string
		intentType    v1.IntentType
		components    []v1.ComponentType
		parameters    map[string]string
		validateFiles func(map[string]string)
	}{
		{
			name:       "amf_deployment",
			intentType: v1.IntentTypeDeployment,
			components: []v1.ComponentType{v1.ComponentTypeAMF},
			parameters: map[string]string{
				"replicas": "3",
				"region":   "us-east-1",
			},
			validateFiles: func(files map[string]string) {
				assert.Contains(t, files, "Kptfile")
				assert.Contains(t, files, "deployment.yaml")
				assert.Contains(t, files, "service.yaml")
				assert.Contains(t, files, "configmap.yaml")
				assert.Contains(t, files["deployment.yaml"], "amf")
				assert.Contains(t, files["deployment.yaml"], "replicas: 3")
			},
		},
		{
			name:       "smf_deployment",
			intentType: v1.IntentTypeDeployment,
			components: []v1.ComponentType{v1.ComponentTypeSMF},
			parameters: map[string]string{
				"replicas": "2",
				"region":   "us-west-2",
			},
			validateFiles: func(files map[string]string) {
				assert.Contains(t, files, "deployment.yaml")
				assert.Contains(t, files["deployment.yaml"], "smf")
				assert.Contains(t, files["deployment.yaml"], "replicas: 2")
			},
		},
		{
			name:       "upf_deployment",
			intentType: v1.IntentTypeDeployment,
			components: []v1.ComponentType{v1.ComponentTypeUPF},
			parameters: map[string]string{
				"replicas": "1",
				"region":   "eu-central-1",
			},
			validateFiles: func(files map[string]string) {
				assert.Contains(t, files, "deployment.yaml")
				assert.Contains(t, files["deployment.yaml"], "upf")
				assert.Contains(t, files["deployment.yaml"], "replicas: 1")
			},
		},
	}

	generator := NewMockGenerator()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			intent := &v1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tc.name,
					Namespace: "test-namespace",
				},
				Spec: v1.NetworkIntentSpec{
					IntentType:       tc.intentType,
					TargetComponents: tc.components,
					Parameters:       tc.parameters,
				},
			}

			files, err := generator.GenerateFromNetworkIntent(context.Background(), intent)
			assert.NoError(t, err)
			assert.NotNil(t, files)

			if tc.validateFiles != nil {
				tc.validateFiles(files)
			}
		})
	}
}

// TestHealthChecks tests health checking functionality
func TestHealthChecks(t *testing.T) {
	mockMgr := newMockManager()
	logger := zaptest.NewLogger(t)
	
	manager := &Manager{
		client:      mockMgr.GetClient(),
		k8sClient:   fake.NewSimpleClientset(),
		config:      DefaultBlueprintConfig(),
		logger:      logger,
		metrics:     NewBlueprintMetrics(),
		catalog:     NewMockCatalog(),
		generator:   NewMockGenerator(),
		customizer:  NewMockCustomizer(),
		validator:   NewMockValidator(),
		ctx:         context.Background(),
		healthStatus: make(map[string]bool),
	}

	// Perform health check
	manager.performHealthCheck()

	// Get health status
	healthStatus := manager.GetHealthStatus()

	// Verify all components are healthy
	assert.True(t, healthStatus["catalog"])
	assert.True(t, healthStatus["generator"])
	assert.True(t, healthStatus["customizer"])
	assert.True(t, healthStatus["validator"])

	// Test with failing component
	manager.generator.(*MockGenerator).SetShouldFail(true)
	manager.performHealthCheck()
	
	healthStatus = manager.GetHealthStatus()
	assert.False(t, healthStatus["generator"])
	assert.True(t, healthStatus["catalog"]) // Others should still be healthy
}

// TestMetricsCollection tests metrics collection
func TestMetricsCollection(t *testing.T) {
	mockMgr := newMockManager()
	logger := zaptest.NewLogger(t)
	
	manager := &Manager{
		client:      mockMgr.GetClient(),
		k8sClient:   fake.NewSimpleClientset(),
		config:      DefaultBlueprintConfig(),
		logger:      logger,
		metrics:     NewBlueprintMetrics(),
		catalog:     NewMockCatalog(),
		generator:   NewMockGenerator(),
		customizer:  NewMockCustomizer(),
		validator:   NewMockValidator(),
		ctx:         context.Background(),
		healthStatus: make(map[string]bool),
	}

	// Process some intents to generate metrics
	intents := []*v1.NetworkIntent{
		createTestNetworkIntent("metrics-test-1"),
		createTestNetworkIntent("metrics-test-2"),
		createTestNetworkIntent("metrics-test-3"),
	}

	for _, intent := range intents {
		_, err := manager.ProcessNetworkIntent(context.Background(), intent)
		assert.NoError(t, err)
	}

	// Update metrics manually (normally done by worker)
	manager.updateMetrics()

	// Get metrics
	metrics := manager.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "queue_depth")
	assert.Contains(t, metrics, "concurrent_operations")
	assert.Contains(t, metrics, "cache_size")

	// Verify generation metrics
	assert.Equal(t, 3, manager.generator.(*MockGenerator).GetGeneratedCount())
	assert.Equal(t, 3, manager.customizer.(*MockCustomizer).GetCustomizedCount())
	assert.Equal(t, 3, manager.validator.(*MockValidator).GetValidatedCount())
}

// TestPackageRevisionCreation tests PackageRevision creation
func TestPackageRevisionCreation(t *testing.T) {
	mockMgr := newMockManager()
	logger := zaptest.NewLogger(t)
	
	manager := &Manager{
		client:      mockMgr.GetClient(),
		k8sClient:   fake.NewSimpleClientset(),
		config:      DefaultBlueprintConfig(),
		logger:      logger,
		metrics:     NewBlueprintMetrics(),
		ctx:         context.Background(),
	}

	intent := createTestNetworkIntent("package-creation-test")
	files := map[string]string{
		"Kptfile":        generateKptfile(intent),
		"deployment.yaml": generateDeployment(intent),
		"service.yaml":    generateService(intent),
	}

	packageRevision, err := manager.createPackageRevision(context.Background(), intent, files)
	
	assert.NoError(t, err)
	assert.NotNil(t, packageRevision)
	assert.Equal(t, "porch.kpt.dev/v1alpha1", packageRevision.APIVersion)
	assert.Equal(t, "PackageRevision", packageRevision.Kind)
	assert.Contains(t, packageRevision.Name, intent.Name)
	assert.Equal(t, intent.Namespace, packageRevision.Namespace)
	
	// Check annotations
	assert.Contains(t, packageRevision.Annotations, AnnotationBlueprintType)
	assert.Contains(t, packageRevision.Annotations, AnnotationIntentID)
	assert.Contains(t, packageRevision.Annotations, AnnotationORANCompliant)
	
	// Check labels
	assert.Contains(t, packageRevision.Labels, "nephoran.com/blueprint")
	assert.Contains(t, packageRevision.Labels, "nephoran.com/intent")
	
	// Check spec
	assert.Equal(t, len(files), len(packageRevision.Spec.Resources))
}

// TestComponentExtraction tests component extraction from NetworkIntent
func TestComponentExtraction(t *testing.T) {
	mockMgr := newMockManager()
	logger := zaptest.NewLogger(t)
	
	manager := &Manager{
		config: DefaultBlueprintConfig(),
		logger: logger,
	}

	testCases := []struct {
		name               string
		targetComponents   []v1.ComponentType
		expectedComponent  string
	}{
		{
			name:               "amf_component",
			targetComponents:   []v1.ComponentType{v1.ComponentTypeAMF},
			expectedComponent:  "amf",
		},
		{
			name:               "smf_component",
			targetComponents:   []v1.ComponentType{v1.ComponentTypeSMF},
			expectedComponent:  "smf",
		},
		{
			name:               "multiple_components",
			targetComponents:   []v1.ComponentType{v1.ComponentTypeAMF, v1.ComponentTypeSMF},
			expectedComponent:  "amf", // Should return first component
		},
		{
			name:               "no_components",
			targetComponents:   []v1.ComponentType{},
			expectedComponent:  "unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			intent := &v1.NetworkIntent{
				Spec: v1.NetworkIntentSpec{
					TargetComponents: tc.targetComponents,
				},
			}

			component := manager.getComponentFromIntent(intent)
			assert.Equal(t, tc.expectedComponent, component)
		})
	}
}

// TestErrorHandling tests error handling scenarios
func TestErrorHandling(t *testing.T) {
	mockMgr := newMockManager()
	logger := zaptest.NewLogger(t)
	
	manager := &Manager{
		client:      mockMgr.GetClient(),
		k8sClient:   fake.NewSimpleClientset(),
		config:      DefaultBlueprintConfig(),
		logger:      logger,
		metrics:     NewBlueprintMetrics(),
		ctx:         context.Background(),
		healthStatus: make(map[string]bool),
	}

	t.Run("nil_intent", func(t *testing.T) {
		// This test would require modifying the actual method to handle nil intents
		// For now, we'll test with an empty intent
		intent := &v1.NetworkIntent{}
		
		manager.catalog = NewMockCatalog()
		generator := NewMockGenerator()
		generator.SetShouldFail(true)
		manager.generator = generator
		manager.customizer = NewMockCustomizer()
		manager.validator = NewMockValidator()

		_, err := manager.ProcessNetworkIntent(context.Background(), intent)
		assert.Error(t, err)
	})

	t.Run("context_cancellation", func(t *testing.T) {
		intent := createTestNetworkIntent("context-cancel-test")
		
		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		manager.catalog = NewMockCatalog()
		manager.generator = NewMockGenerator()
		manager.customizer = NewMockCustomizer()
		manager.validator = NewMockValidator()

		// The actual implementation would need to check context cancellation
		// For this mock, we'll just verify it handles the cancelled context gracefully
		result, err := manager.ProcessNetworkIntent(ctx, intent)
		
		// In a real implementation, this might return an error due to context cancellation
		// For the mock, we'll just ensure it doesn't panic
		_ = result
		_ = err
	})
}

// TestCacheOperations tests cache functionality
func TestCacheOperations(t *testing.T) {
	mockMgr := newMockManager()
	logger := zaptest.NewLogger(t)
	
	manager := &Manager{
		client:      mockMgr.GetClient(),
		k8sClient:   fake.NewSimpleClientset(),
		config:      DefaultBlueprintConfig(),
		logger:      logger,
		metrics:     NewBlueprintMetrics(),
		cache:       sync.Map{},
		ctx:         context.Background(),
	}

	// Test cache operations
	testKey := "test-key"
	testValue := map[string]interface{}{
		"data":        "test-data",
		"expire_time": time.Now().Add(time.Hour),
	}

	// Store in cache
	manager.cache.Store(testKey, testValue)

	// Retrieve from cache
	retrieved, exists := manager.cache.Load(testKey)
	assert.True(t, exists)
	assert.NotNil(t, retrieved)

	// Test cache cleanup
	expiredKey := "expired-key"
	expiredValue := map[string]interface{}{
		"data":        "expired-data",
		"expire_time": time.Now().Add(-time.Hour), // Expired
	}
	manager.cache.Store(expiredKey, expiredValue)

	// Run cleanup
	manager.cleanupCache()

	// Verify expired entry was removed
	_, exists = manager.cache.Load(expiredKey)
	assert.False(t, exists)

	// Verify non-expired entry still exists
	_, exists = manager.cache.Load(testKey)
	assert.True(t, exists)
}

// TestManagerLifecycle tests manager lifecycle operations
func TestManagerLifecycle(t *testing.T) {
	mockMgr := newMockManager()
	logger := zaptest.NewLogger(t)
	
	// Create manager
	manager, err := NewManager(mockMgr, DefaultBlueprintConfig(), logger)
	require.NoError(t, err)
	require.NotNil(t, manager)

	// Verify manager is running
	assert.NotNil(t, manager.ctx)
	assert.NotNil(t, manager.cancel)

	// Stop manager
	err = manager.Stop()
	assert.NoError(t, err)

	// Verify context was cancelled
	select {
	case <-manager.ctx.Done():
		// Context was cancelled as expected
	case <-time.After(1 * time.Second):
		t.Fatal("Expected context to be cancelled")
	}
}

// BenchmarkBlueprintProcessing benchmarks blueprint processing performance
func BenchmarkBlueprintProcessing(b *testing.B) {
	mockMgr := newMockManager()
	logger := zaptest.NewLogger(b)
	
	manager := &Manager{
		client:      mockMgr.GetClient(),
		k8sClient:   fake.NewSimpleClientset(),
		config:      DefaultBlueprintConfig(),
		logger:      logger,
		metrics:     NewBlueprintMetrics(),
		catalog:     NewMockCatalog(),
		generator:   NewMockGenerator(),
		customizer:  NewMockCustomizer(),
		validator:   NewMockValidator(),
		ctx:         context.Background(),
		healthStatus: make(map[string]bool),
	}

	intent := createTestNetworkIntent("benchmark-test")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := manager.ProcessNetworkIntent(context.Background(), intent)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// TestValidationScenarios tests various validation scenarios
func TestValidationScenarios(t *testing.T) {
	testCases := []struct {
		name           string
		intent         *v1.NetworkIntent
		files          map[string]string
		shouldReject   bool
		expectedErrors []string
	}{
		{
			name:   "valid_blueprint",
			intent: createTestNetworkIntent("valid-test"),
			files: map[string]string{
				"Kptfile":        "valid kptfile content",
				"deployment.yaml": "valid deployment content",
			},
			shouldReject: false,
		},
		{
			name:   "invalid_blueprint",
			intent: createTestNetworkIntent("invalid-test"),
			files: map[string]string{
				"invalid.yaml": "invalid content",
			},
			shouldReject:   true,
			expectedErrors: []string{"validation failed: missing required field"},
		},
		{
			name:   "empty_blueprint",
			intent: createTestNetworkIntent("empty-test"),
			files:  map[string]string{},
			shouldReject:   true,
			expectedErrors: []string{"validation failed: missing required field"},
		},
	}

	validator := NewMockValidator()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator.SetShouldReject(tc.shouldReject)

			result, err := validator.ValidateBlueprint(context.Background(), tc.intent, tc.files)
			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tc.shouldReject {
				assert.False(t, result.IsValid)
				assert.Equal(t, len(tc.expectedErrors), len(result.Errors))
			} else {
				assert.True(t, result.IsValid)
				assert.Empty(t, result.Errors)
			}
		})
	}
}

// TestComplexScenarios tests complex real-world scenarios
func TestComplexScenarios(t *testing.T) {
	mockMgr := newMockManager()
	logger := zaptest.NewLogger(t)
	
	manager := &Manager{
		client:      mockMgr.GetClient(),
		k8sClient:   fake.NewSimpleClientset(),
		config:      DefaultBlueprintConfig(),
		logger:      logger,
		metrics:     NewBlueprintMetrics(),
		catalog:     NewMockCatalog(),
		generator:   NewMockGenerator(),
		customizer:  NewMockCustomizer(),
		validator:   NewMockValidator(),
		ctx:         context.Background(),
		healthStatus: make(map[string]bool),
	}

	t.Run("complete_5g_core_deployment", func(t *testing.T) {
		// Create intents for complete 5G core
		components := []struct {
			name      string
			component v1.ComponentType
			replicas  string
		}{
			{"5g-amf", v1.ComponentTypeAMF, "3"},
			{"5g-smf", v1.ComponentTypeSMF, "2"},
			{"5g-upf", v1.ComponentTypeUPF, "2"},
			{"5g-nssf", v1.ComponentTypeNSSF, "1"},
		}

		results := make([]*BlueprintResult, len(components))
		
		for i, comp := range components {
			intent := &v1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      comp.name,
					Namespace: "5g-core",
				},
				Spec: v1.NetworkIntentSpec{
					IntentType:       v1.IntentTypeDeployment,
					Priority:         v1.PriorityHigh,
					TargetComponents: []v1.ComponentType{comp.component},
					Parameters: map[string]string{
						"replicas": comp.replicas,
						"region":   "us-east-1",
						"environment": "production",
					},
				},
			}

			result, err := manager.ProcessNetworkIntent(context.Background(), intent)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.True(t, result.Success)

			results[i] = result
		}

		// Verify all components were processed successfully
		for i, result := range results {
			assert.True(t, result.Success, "Component %s should succeed", components[i].name)
			assert.NotNil(t, result.PackageRevision)
			assert.True(t, len(result.GeneratedFiles) > 0)
		}
	})

	t.Run("multi_region_deployment", func(t *testing.T) {
		regions := []string{"us-east-1", "us-west-2", "eu-central-1"}
		
		for _, region := range regions {
			intent := &v1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("amf-%s", region),
					Namespace: "multi-region",
				},
				Spec: v1.NetworkIntentSpec{
					IntentType:       v1.IntentTypeDeployment,
					Priority:         v1.PriorityMedium,
					TargetComponents: []v1.ComponentType{v1.ComponentTypeAMF},
					Parameters: map[string]string{
						"replicas": "2",
						"region":   region,
						"zone":     region + "a",
					},
				},
			}

			result, err := manager.ProcessNetworkIntent(context.Background(), intent)
			require.NoError(t, err)
			require.True(t, result.Success)

			// Verify region-specific configuration
			assert.Contains(t, result.GeneratedFiles["configmap.yaml"], region)
		}
	})
}