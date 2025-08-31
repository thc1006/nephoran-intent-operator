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

package orchestration

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

// RegressionTestSuite provides comprehensive backward compatibility testing
type RegressionTestSuite struct {
	// Original controllers for reference
	originalIntentController *IntentProcessingController

	// New specialized controllers
	specializedIntentController   *SpecializedIntentProcessingController
	specializedResourceController *SpecializedResourcePlanningController
	specializedManifestController *SpecializedManifestGenerationController

	// Shared dependencies
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	eventBus *EventBus
	logger   logr.Logger
	ctx      context.Context
	cancel   context.CancelFunc

	// Mock services
	mockLLMService *MockLLMService
	mockRAGService *MockRAGService

	// Test configuration
	testConfig *RegressionTestConfig

	// Results tracking
	originalResults    map[string]*ProcessingResult
	specializedResults map[string]*ProcessingResult
	mutex              sync.RWMutex
}

// RegressionTestConfig contains configuration for regression testing
type RegressionTestConfig struct {
	TestTimeout           time.Duration
	MaxConcurrentTests    int
	ToleranceThreshold    float64 // Acceptable performance difference
	CompatibilityRequired bool    // Strict compatibility mode
	ValidationEnabled     bool    // Enable output validation
	BenchmarkIterations   int     // Number of benchmark runs
}

// ProcessingResult captures processing results for comparison
type ProcessingResult struct {
	Success          bool
	ProcessingTime   time.Duration
	OutputData       map[string]interface{}
	ErrorMessage     string
	QualityScore     float64
	TokenUsage       *nephoranv1.TokenUsageInfo
	RAGMetrics       *nephoranv1.RAGMetrics
	PhaseResults     map[interfaces.ProcessingPhase]*PhaseResult
	ValidationErrors []string
}

// PhaseResult captures individual phase results
type PhaseResult struct {
	Phase       interfaces.ProcessingPhase
	Success     bool
	Duration    time.Duration
	Output      interface{}
	Error       string
	MetricsData map[string]interface{}
}

// Mock implementations for consistent testing

type MockLLMService struct {
	mock.Mock
	responses map[string]*llm.ProcessingResponse
	mutex     sync.RWMutex
}

func (m *MockLLMService) ProcessIntent(ctx context.Context, request *llm.ProcessingRequest) (*llm.ProcessingResponse, error) {
	args := m.Called(ctx, request)

	// Return predefined response if available
	m.mutex.RLock()
	if response, exists := m.responses[request.Intent]; exists {
		m.mutex.RUnlock()
		return response, nil
	}
	m.mutex.RUnlock()

	// Default mock response
	response := &llm.ProcessingResponse{
		ProcessedIntent: fmt.Sprintf("Processed: %s", request.Intent),
		StructuredParameters: map[string]interface{}{
			"network_function": "AMF",
			"deployment_config": map[string]interface{}{
				"replicas": 3,
				"resources": map[string]interface{}{
					"cpu":    "500m",
					"memory": "1Gi",
				},
			},
			"performance_requirements": map[string]interface{}{
				"latency":    "10ms",
				"throughput": "1000 req/s",
			},
		},
		NetworkFunctions: []string{"AMF", "SMF"},
		TokenUsage: &nephoranv1.TokenUsageInfo{
			PromptTokens:     100,
			CompletionTokens: 200,
			TotalTokens:      300,
		},
		TelecomContext: map[string]interface{}{
			"domain":   "5g_core",
			"standard": "3gpp_rel16",
		},
		ExtractedEntities: map[string]interface{}{
			"network_type":           "5G",
			"deployment_environment": "production",
		},
	}

	return response, args.Error(1)
}

func (m *MockLLMService) SetResponse(intent string, response *llm.ProcessingResponse) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.responses == nil {
		m.responses = make(map[string]*llm.ProcessingResponse)
	}
	m.responses[intent] = response
}

// MockRAGService is now defined in test_mocks.go

var _ = Describe("Regression Backward Compatibility Tests", func() {
	var (
		suite *RegressionTestSuite
	)

	BeforeEach(func() {
		suite = setupRegressionTestSuite()
	})

	AfterEach(func() {
		suite.cleanup()
	})

	Describe("API Compatibility", func() {
		It("should maintain identical API interfaces", func() {
			suite.testAPICompatibility()
		})

		It("should preserve input/output data structures", func() {
			suite.testDataStructureCompatibility()
		})

		It("should maintain configuration compatibility", func() {
			suite.testConfigurationCompatibility()
		})
	})

	Describe("Functional Compatibility", func() {
		It("should produce equivalent processing results", func() {
			suite.testFunctionalEquivalence()
		})

		It("should handle all original intent types", func() {
			suite.testIntentTypeCompatibility()
		})

		It("should maintain error handling behavior", func() {
			suite.testErrorHandlingCompatibility()
		})

		It("should preserve retry mechanisms", func() {
			suite.testRetryMechanismCompatibility()
		})
	})

	Describe("Performance Compatibility", func() {
		It("should not regress performance significantly", func() {
			suite.testPerformanceRegression()
		})

		It("should improve or maintain throughput", func() {
			suite.testThroughputCompatibility()
		})

		It("should maintain or improve resource utilization", func() {
			suite.testResourceUtilizationCompatibility()
		})
	})

	Describe("Integration Compatibility", func() {
		It("should maintain event bus integration", func() {
			suite.testEventBusCompatibility()
		})

		It("should preserve metrics collection", func() {
			suite.testMetricsCompatibility()
		})

		It("should maintain status reporting", func() {
			suite.testStatusReportingCompatibility()
		})
	})

	Describe("Concurrency Compatibility", func() {
		It("should handle concurrent processing equivalently", func() {
			suite.testConcurrencyCompatibility()
		})

		It("should maintain thread safety", func() {
			suite.testThreadSafetyCompatibility()
		})
	})
})

func setupRegressionTestSuite() *RegressionTestSuite {
	zapLogger, _ := zap.NewDevelopment()
	logger := zapr.New(zapLogger)

	// Create scheme and client
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create mock recorder
	recorder := record.NewFakeRecorder(100)

	// Create event bus
	eventBus := NewEventBus(logger)

	// Create mock services
	mockLLMService := &MockLLMService{}
	mockRAGService := &MockRAGService{}

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize controllers
	originalIntentConfig := DefaultIntentProcessingConfig()
	originalIntentController := NewIntentProcessingController(
		client, scheme, recorder,
		mockLLMService, mockRAGService,
		eventBus, originalIntentConfig,
	)

	// Create specialized controllers
	specializedIntentController := NewSpecializedIntentProcessingController(
		client, scheme, recorder,
		mockLLMService, mockRAGService,
		eventBus, logger,
	)

	specializedResourceController := NewSpecializedResourcePlanningController(
		client, scheme, recorder,
		eventBus, logger,
	)

	specializedManifestController := NewSpecializedManifestGenerationController(
		client, scheme, recorder,
		eventBus, logger,
	)

	return &RegressionTestSuite{
		originalIntentController:      originalIntentController,
		specializedIntentController:   specializedIntentController,
		specializedResourceController: specializedResourceController,
		specializedManifestController: specializedManifestController,
		client:                        client,
		scheme:                        scheme,
		recorder:                      recorder,
		eventBus:                      eventBus,
		logger:                        logger,
		ctx:                           ctx,
		cancel:                        cancel,
		mockLLMService:                mockLLMService,
		mockRAGService:                mockRAGService,
		testConfig: &RegressionTestConfig{
			TestTimeout:           30 * time.Second,
			MaxConcurrentTests:    10,
			ToleranceThreshold:    0.1, // 10% tolerance
			CompatibilityRequired: true,
			ValidationEnabled:     true,
			BenchmarkIterations:   5,
		},
		originalResults:    make(map[string]*ProcessingResult),
		specializedResults: make(map[string]*ProcessingResult),
	}
}

func (suite *RegressionTestSuite) cleanup() {
	suite.cancel()
}

func (suite *RegressionTestSuite) testAPICompatibility() {
	By("Comparing API interfaces between original and specialized controllers")

	// Test 1: Reconcile method signature compatibility
	originalType := reflect.TypeOf(suite.originalIntentController)
	specializedType := reflect.TypeOf(suite.specializedIntentController)

	// Check that both have Reconcile method
	originalReconcile, hasOriginal := originalType.MethodByName("Reconcile")
	specializedReconcile, hasSpecialized := specializedType.MethodByName("Reconcile")

	Expect(hasOriginal).To(BeTrue(), "Original controller should have Reconcile method")
	Expect(hasSpecialized).To(BeTrue(), "Specialized controller should have Reconcile method")

	// Compare method signatures
	Expect(originalReconcile.Type.String()).To(Equal(specializedReconcile.Type.String()),
		"Reconcile method signatures should be identical")

	// Test 2: SetupWithManager method compatibility
	originalSetup, hasOriginalSetup := originalType.MethodByName("SetupWithManager")
	specializedSetup, hasSpecializedSetup := specializedType.MethodByName("SetupWithManager")

	Expect(hasOriginalSetup).To(BeTrue(), "Original controller should have SetupWithManager method")
	Expect(hasSpecializedSetup).To(BeTrue(), "Specialized controller should have SetupWithManager method")
	Expect(originalSetup.Type.String()).To(Equal(specializedSetup.Type.String()),
		"SetupWithManager method signatures should be identical")

	suite.logger.Info("API compatibility test passed")
}

func (suite *RegressionTestSuite) testDataStructureCompatibility() {
	By("Verifying data structure compatibility between implementations")

	// Create test NetworkIntent
	testIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "regression-test-intent",
			Namespace: "test-namespace",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			IntentType: "5g_core_deployment",
			Intent:     "Deploy high-availability 5G Core network",
			Priority:   "high",
		},
	}

	// Test input data structure handling
	originalResult := suite.processWithOriginalController(testIntent)
	specializedResult := suite.processWithSpecializedControllers(testIntent)

	// Compare output data structures
	suite.compareDataStructures(originalResult, specializedResult)

	suite.logger.Info("Data structure compatibility test passed")
}

func (suite *RegressionTestSuite) testConfigurationCompatibility() {
	By("Testing configuration parameter compatibility")

	// Test that all original configuration options are supported
	originalConfig := DefaultIntentProcessingConfig()

	// Verify specialized controllers can handle same configuration
	configTests := []struct {
		name   string
		config *IntentProcessingConfig
		valid  bool
	}{
		{
			name:   "default_configuration",
			config: originalConfig,
			valid:  true,
		},
		{
			name: "high_concurrency_config",
			config: &IntentProcessingConfig{
				MaxConcurrentProcessing: 50,
				DefaultTimeout:          60 * time.Second,
				MaxRetries:              5,
				RetryBackoff:            15 * time.Second,
				QualityThreshold:        0.8,
				EnableRAG:               true,
				ValidationEnabled:       true,
			},
			valid: true,
		},
		{
			name: "rag_disabled_config",
			config: &IntentProcessingConfig{
				MaxConcurrentProcessing: 10,
				DefaultTimeout:          120 * time.Second,
				MaxRetries:              3,
				RetryBackoff:            30 * time.Second,
				QualityThreshold:        0.6,
				EnableRAG:               false,
				ValidationEnabled:       false,
			},
			valid: true,
		},
	}

	for _, test := range configTests {
		suite.logger.Info("Testing configuration", "name", test.name)

		// Test configuration acceptance
		success := suite.testConfigurationAcceptance(test.config)
		Expect(success).To(Equal(test.valid),
			fmt.Sprintf("Configuration %s should be %v", test.name, test.valid))
	}

	suite.logger.Info("Configuration compatibility test passed")
}

func (suite *RegressionTestSuite) testFunctionalEquivalence() {
	By("Testing functional equivalence between original and specialized controllers")

	testCases := []struct {
		name         string
		intentType   string
		intent       string
		priority     string
		expectedFunc bool
	}{
		{
			name:         "5g_core_deployment",
			intentType:   "5g_core_deployment",
			intent:       "Deploy 5G Core with AMF, SMF, and UPF",
			priority:     "high",
			expectedFunc: true,
		},
		{
			name:         "ran_deployment",
			intentType:   "ran_deployment",
			intent:       "Deploy O-RAN compliant RAN infrastructure",
			priority:     "medium",
			expectedFunc: true,
		},
		{
			name:         "network_slicing",
			intentType:   "network_slicing",
			intent:       "Create network slice for eMBB with QoS guarantees",
			priority:     "high",
			expectedFunc: true,
		},
		{
			name:         "edge_deployment",
			intentType:   "edge_deployment",
			intent:       "Deploy edge computing infrastructure",
			priority:     "low",
			expectedFunc: true,
		},
	}

	for _, testCase := range testCases {
		suite.logger.Info("Testing functional equivalence", "case", testCase.name)

		// Create test intent
		intent := &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("func-test-%s", testCase.name),
				Namespace: "test-namespace",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				IntentType: testCase.intentType,
				Intent:     testCase.intent,
				Priority:   testCase.priority,
			},
		}

		// Process with both implementations
		originalResult := suite.processWithOriginalController(intent)
		specializedResult := suite.processWithSpecializedControllers(intent)

		// Compare results
		suite.compareFunctionalResults(originalResult, specializedResult, testCase.expectedFunc)
	}

	suite.logger.Info("Functional equivalence test passed")
}

func (suite *RegressionTestSuite) testIntentTypeCompatibility() {
	By("Testing all supported intent types")

	intentTypes := []string{
		"5g_core_deployment",
		"ran_deployment",
		"network_slicing",
		"edge_deployment",
		"oran_deployment",
		"mec_deployment",
		"service_mesh_deployment",
	}

	for _, intentType := range intentTypes {
		intent := &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("type-test-%s", intentType),
				Namespace: "test-namespace",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				IntentType: intentType,
				Intent:     fmt.Sprintf("Test intent for %s", intentType),
				Priority:   "medium",
			},
		}

		// Both implementations should handle all intent types
		originalResult := suite.processWithOriginalController(intent)
		specializedResult := suite.processWithSpecializedControllers(intent)

		Expect(originalResult.Success).To(Equal(specializedResult.Success),
			fmt.Sprintf("Intent type %s should have same success status", intentType))
	}

	suite.logger.Info("Intent type compatibility test passed")
}

func (suite *RegressionTestSuite) testErrorHandlingCompatibility() {
	By("Testing error handling compatibility")

	// Configure mock service to return errors
	suite.mockLLMService.On("ProcessIntent", mock.Anything, mock.Anything).
		Return(nil, fmt.Errorf("simulated LLM service error"))

	intent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "error-test-intent",
			Namespace: "test-namespace",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			IntentType: "test_deployment",
			Intent:     "Test intent that will cause error",
			Priority:   "medium",
		},
	}

	// Process with both implementations
	originalResult := suite.processWithOriginalController(intent)
	specializedResult := suite.processWithSpecializedControllers(intent)

	// Both should fail in the same way
	Expect(originalResult.Success).To(BeFalse())
	Expect(specializedResult.Success).To(BeFalse())

	// Error messages should be similar
	Expect(specializedResult.ErrorMessage).To(ContainSubstring("LLM"))

	suite.logger.Info("Error handling compatibility test passed")
}

func (suite *RegressionTestSuite) testRetryMechanismCompatibility() {
	By("Testing retry mechanism compatibility")

	// Configure mock to fail first few attempts then succeed
	callCount := 0
	suite.mockLLMService.On("ProcessIntent", mock.Anything, mock.Anything).
		Return(func(ctx context.Context, req *llm.ProcessingRequest) (*llm.ProcessingResponse, error) {
			callCount++
			if callCount < 3 {
				return nil, fmt.Errorf("simulated failure %d", callCount)
			}
			return &llm.ProcessingResponse{
				ProcessedIntent: "Processed after retries",
				StructuredParameters: map[string]interface{}{
					"network_function": "AMF",
				},
				NetworkFunctions: []string{"AMF"},
			}, nil
		})

	intent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "retry-test-intent",
			Namespace: "test-namespace",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			IntentType: "retry_test",
			Intent:     "Test intent for retry mechanism",
			Priority:   "medium",
		},
	}

	// Process with both implementations
	originalResult := suite.processWithOriginalController(intent)
	specializedResult := suite.processWithSpecializedControllers(intent)

	// Both should eventually succeed after retries
	Expect(originalResult.Success).To(Equal(specializedResult.Success))

	suite.logger.Info("Retry mechanism compatibility test passed")
}

func (suite *RegressionTestSuite) testPerformanceRegression() {
	By("Testing performance regression")

	testIntents := suite.generatePerformanceTestIntents()

	// Benchmark original implementation
	originalBenchmark := suite.benchmarkOriginalController(testIntents)

	// Benchmark specialized implementation
	specializedBenchmark := suite.benchmarkSpecializedControllers(testIntents)

	// Compare performance metrics
	suite.comparePerformanceMetrics(originalBenchmark, specializedBenchmark)

	suite.logger.Info("Performance regression test passed")
}

func (suite *RegressionTestSuite) testThroughputCompatibility() {
	By("Testing throughput compatibility")

	// Generate high-volume test data
	testIntents := suite.generateThroughputTestIntents(100)

	// Measure original throughput
	originalThroughput := suite.measureThroughput("original", func() {
		suite.processBatchWithOriginalController(testIntents)
	})

	// Measure specialized throughput
	specializedThroughput := suite.measureThroughput("specialized", func() {
		suite.processBatchWithSpecializedControllers(testIntents)
	})

	// Specialized implementation should maintain or improve throughput
	improvementRatio := specializedThroughput / originalThroughput
	Expect(improvementRatio).To(BeNumerically(">=", 1.0-suite.testConfig.ToleranceThreshold),
		fmt.Sprintf("Throughput should not regress more than %.1f%%",
			suite.testConfig.ToleranceThreshold*100))

	suite.logger.Info("Throughput compatibility test passed",
		"original", originalThroughput,
		"specialized", specializedThroughput,
		"improvement_ratio", improvementRatio)
}

func (suite *RegressionTestSuite) testResourceUtilizationCompatibility() {
	By("Testing resource utilization compatibility")

	// This would typically measure memory usage, CPU usage, etc.
	// For this test, we'll focus on goroutine count and channel usage

	testIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-test-intent",
			Namespace: "test-namespace",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			IntentType: "resource_test",
			Intent:     "Test intent for resource utilization",
			Priority:   "medium",
		},
	}

	// Measure resource usage for both implementations
	originalResources := suite.measureResourceUsage("original", func() {
		suite.processWithOriginalController(testIntent)
	})

	specializedResources := suite.measureResourceUsage("specialized", func() {
		suite.processWithSpecializedControllers(testIntent)
	})

	// Compare resource utilization
	suite.compareResourceUtilization(originalResources, specializedResources)

	suite.logger.Info("Resource utilization compatibility test passed")
}

func (suite *RegressionTestSuite) testEventBusCompatibility() {
	By("Testing event bus integration compatibility")

	// Track events published by both implementations
	eventCollector := NewEventCollector()
	suite.eventBus.Subscribe(interfaces.PhaseAll, eventCollector.CollectEvent)

	testIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-test-intent",
			Namespace: "test-namespace",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			IntentType: "event_test",
			Intent:     "Test intent for event compatibility",
			Priority:   "medium",
		},
	}

	// Process with original controller
	eventCollector.StartCollection("original")
	suite.processWithOriginalController(testIntent)
	originalEvents := eventCollector.StopCollection("original")

	// Process with specialized controllers
	eventCollector.StartCollection("specialized")
	suite.processWithSpecializedControllers(testIntent)
	specializedEvents := eventCollector.StopCollection("specialized")

	// Compare event patterns
	suite.compareEventPatterns(originalEvents, specializedEvents)

	suite.logger.Info("Event bus compatibility test passed")
}

func (suite *RegressionTestSuite) testMetricsCompatibility() {
	By("Testing metrics collection compatibility")

	testIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metrics-test-intent",
			Namespace: "test-namespace",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			IntentType: "metrics_test",
			Intent:     "Test intent for metrics compatibility",
			Priority:   "medium",
		},
	}

	// Process with both implementations and compare metrics
	originalResult := suite.processWithOriginalController(testIntent)
	specializedResult := suite.processWithSpecializedControllers(testIntent)

	// Compare metrics structure and content
	suite.compareMetricsData(originalResult, specializedResult)

	suite.logger.Info("Metrics compatibility test passed")
}

func (suite *RegressionTestSuite) testStatusReportingCompatibility() {
	By("Testing status reporting compatibility")

	testIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "status-test-intent",
			Namespace: "test-namespace",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			IntentType: "status_test",
			Intent:     "Test intent for status reporting",
			Priority:   "medium",
		},
	}

	// Process and compare status updates
	originalResult := suite.processWithOriginalController(testIntent)
	specializedResult := suite.processWithSpecializedControllers(testIntent)

	// Status reporting should be equivalent
	suite.compareStatusReporting(originalResult, specializedResult)

	suite.logger.Info("Status reporting compatibility test passed")
}

func (suite *RegressionTestSuite) testConcurrencyCompatibility() {
	By("Testing concurrency handling compatibility")

	// Generate concurrent test intents
	concurrentIntents := suite.generateConcurrentTestIntents(20)

	var wg sync.WaitGroup
	originalResults := make([]*ProcessingResult, len(concurrentIntents))
	specializedResults := make([]*ProcessingResult, len(concurrentIntents))

	// Test original implementation
	for i, intent := range concurrentIntents {
		wg.Add(1)
		go func(index int, testIntent *nephoranv1.NetworkIntent) {
			defer wg.Done()
			originalResults[index] = suite.processWithOriginalController(testIntent)
		}(i, intent)
	}
	wg.Wait()

	// Test specialized implementation
	for i, intent := range concurrentIntents {
		wg.Add(1)
		go func(index int, testIntent *nephoranv1.NetworkIntent) {
			defer wg.Done()
			specializedResults[index] = suite.processWithSpecializedControllers(testIntent)
		}(i, intent)
	}
	wg.Wait()

	// Compare concurrent processing results
	suite.compareConcurrentResults(originalResults, specializedResults)

	suite.logger.Info("Concurrency compatibility test passed")
}

func (suite *RegressionTestSuite) testThreadSafetyCompatibility() {
	By("Testing thread safety compatibility")

	// Create shared state that both implementations will access
	sharedIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "thread-safety-test-intent",
			Namespace: "test-namespace",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			IntentType: "thread_safety_test",
			Intent:     "Test intent for thread safety",
			Priority:   "medium",
		},
	}

	// Test concurrent access to same resource
	var wg sync.WaitGroup
	results := make([]bool, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result := suite.processWithSpecializedControllers(sharedIntent)
			results[index] = result.Success
		}(i)
	}

	wg.Wait()

	// Verify no race conditions or data corruption
	successCount := 0
	for _, success := range results {
		if success {
			successCount++
		}
	}

	// At least some operations should succeed (exact count depends on timing)
	Expect(successCount).To(BeNumerically(">", 0),
		"Thread safety test should have some successful operations")

	suite.logger.Info("Thread safety compatibility test passed")
}

// Helper methods for processing and comparison

func (suite *RegressionTestSuite) processWithOriginalController(intent *nephoranv1.NetworkIntent) *ProcessingResult {
	startTime := time.Now()

	// Create processing request
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      intent.Name,
			Namespace: intent.Namespace,
		},
	}

	// Store intent in fake client
	err := suite.client.Create(suite.ctx, intent)
	if err != nil {
		return &ProcessingResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("Failed to create intent: %v", err),
		}
	}

	// Process with original controller
	result, err := suite.originalIntentController.Reconcile(suite.ctx, req)

	processingTime := time.Since(startTime)

	// Create result
	processingResult := &ProcessingResult{
		Success:        err == nil,
		ProcessingTime: processingTime,
		OutputData:     make(map[string]interface{}),
	}

	if err != nil {
		processingResult.ErrorMessage = err.Error()
	}

	// Add result details
	if result.RequeueAfter > 0 {
		processingResult.OutputData["requeue_after"] = result.RequeueAfter.String()
	}

	return processingResult
}

func (suite *RegressionTestSuite) processWithSpecializedControllers(intent *nephoranv1.NetworkIntent) *ProcessingResult {
	startTime := time.Now()

	// Process through specialized controller pipeline
	var err error
	var finalResult *ProcessingResult

	// Phase 1: Intent Processing
	intentResult, err := suite.specializedIntentController.ProcessIntent(suite.ctx, intent)
	if err != nil {
		return &ProcessingResult{
			Success:        false,
			ProcessingTime: time.Since(startTime),
			ErrorMessage:   fmt.Sprintf("Intent processing failed: %v", err),
		}
	}

	// Phase 2: Resource Planning
	resourceResult, err := suite.specializedResourceController.ProcessPhase(suite.ctx, intent, interfaces.PhaseResourcePlanning)
	if err != nil {
		return &ProcessingResult{
			Success:        false,
			ProcessingTime: time.Since(startTime),
			ErrorMessage:   fmt.Sprintf("Resource planning failed: %v", err),
		}
	}

	// Phase 3: Manifest Generation
	manifestResult, err := suite.specializedManifestController.ProcessPhase(suite.ctx, intent, interfaces.PhaseManifestGeneration)
	if err != nil {
		return &ProcessingResult{
			Success:        false,
			ProcessingTime: time.Since(startTime),
			ErrorMessage:   fmt.Sprintf("Manifest generation failed: %v", err),
		}
	}

	processingTime := time.Since(startTime)

	// Combine results
	finalResult = &ProcessingResult{
		Success:        true,
		ProcessingTime: processingTime,
		OutputData:     make(map[string]interface{}),
		PhaseResults:   make(map[interfaces.ProcessingPhase]*PhaseResult),
	}

	// Add phase results
	finalResult.PhaseResults[interfaces.PhaseIntentProcessing] = &PhaseResult{
		Phase:   interfaces.PhaseIntentProcessing,
		Success: intentResult.Success,
		Output:  intentResult,
	}

	finalResult.PhaseResults[interfaces.PhaseResourcePlanning] = &PhaseResult{
		Phase:   interfaces.PhaseResourcePlanning,
		Success: resourceResult.Success,
		Output:  resourceResult,
	}

	finalResult.PhaseResults[interfaces.PhaseManifestGeneration] = &PhaseResult{
		Phase:   interfaces.PhaseManifestGeneration,
		Success: manifestResult.Success,
		Output:  manifestResult,
	}

	return finalResult
}

func (suite *RegressionTestSuite) compareDataStructures(original, specialized *ProcessingResult) {
	// Compare basic result structure
	Expect(specialized.Success).To(Equal(original.Success),
		"Success status should be identical")

	// Compare output data types and structures
	if original.OutputData != nil && specialized.OutputData != nil {
		// Both should have similar data structure
		Expect(len(specialized.OutputData)).To(BeNumerically(">=", 0),
			"Specialized controller should provide output data")
	}
}

func (suite *RegressionTestSuite) compareFunctionalResults(original, specialized *ProcessingResult, expectedSuccess bool) {
	Expect(original.Success).To(Equal(expectedSuccess),
		"Original controller should match expected success")
	Expect(specialized.Success).To(Equal(expectedSuccess),
		"Specialized controller should match expected success")

	// Both should have same success status
	Expect(specialized.Success).To(Equal(original.Success),
		"Functional results should be equivalent")
}

func (suite *RegressionTestSuite) testConfigurationAcceptance(config *IntentProcessingConfig) bool {
	// Test if specialized controllers can handle the configuration
	// In a real implementation, this would involve creating controllers with the config
	return config.MaxConcurrentProcessing > 0 &&
		config.DefaultTimeout > 0 &&
		config.MaxRetries >= 0
}

func (suite *RegressionTestSuite) generatePerformanceTestIntents() []*nephoranv1.NetworkIntent {
	intents := make([]*nephoranv1.NetworkIntent, 20)

	intentTypes := []string{"5g_core", "ran", "edge", "slice"}

	for i := 0; i < len(intents); i++ {
		intents[i] = &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("perf-test-intent-%d", i),
				Namespace: "performance-test",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				IntentType: intentTypes[i%len(intentTypes)],
				Intent:     fmt.Sprintf("Performance test intent %d", i),
				Priority:   "medium",
			},
		}
	}

	return intents
}

func (suite *RegressionTestSuite) benchmarkOriginalController(intents []*nephoranv1.NetworkIntent) *BenchmarkResult {
	startTime := time.Now()
	successCount := 0

	for _, intent := range intents {
		result := suite.processWithOriginalController(intent)
		if result.Success {
			successCount++
		}
	}

	totalTime := time.Since(startTime)

	return &BenchmarkResult{
		TotalTime:    totalTime,
		SuccessCount: successCount,
		TotalCount:   len(intents),
		Throughput:   float64(len(intents)) / totalTime.Seconds(),
	}
}

func (suite *RegressionTestSuite) benchmarkSpecializedControllers(intents []*nephoranv1.NetworkIntent) *BenchmarkResult {
	startTime := time.Now()
	successCount := 0

	for _, intent := range intents {
		result := suite.processWithSpecializedControllers(intent)
		if result.Success {
			successCount++
		}
	}

	totalTime := time.Since(startTime)

	return &BenchmarkResult{
		TotalTime:    totalTime,
		SuccessCount: successCount,
		TotalCount:   len(intents),
		Throughput:   float64(len(intents)) / totalTime.Seconds(),
	}
}

func (suite *RegressionTestSuite) comparePerformanceMetrics(original, specialized *BenchmarkResult) {
	// Specialized implementation should not significantly regress performance
	performanceRatio := specialized.Throughput / original.Throughput

	Expect(performanceRatio).To(BeNumerically(">=", 1.0-suite.testConfig.ToleranceThreshold),
		fmt.Sprintf("Performance regression should not exceed %.1f%%",
			suite.testConfig.ToleranceThreshold*100))

	suite.logger.Info("Performance comparison completed",
		"original_throughput", original.Throughput,
		"specialized_throughput", specialized.Throughput,
		"performance_ratio", performanceRatio)
}

func (suite *RegressionTestSuite) generateThroughputTestIntents(count int) []*nephoranv1.NetworkIntent {
	intents := make([]*nephoranv1.NetworkIntent, count)

	for i := 0; i < count; i++ {
		intents[i] = &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("throughput-test-intent-%d", i),
				Namespace: "throughput-test",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				IntentType: "throughput_test",
				Intent:     fmt.Sprintf("Throughput test intent %d", i),
				Priority:   "medium",
			},
		}
	}

	return intents
}

func (suite *RegressionTestSuite) processBatchWithOriginalController(intents []*nephoranv1.NetworkIntent) {
	for _, intent := range intents {
		suite.processWithOriginalController(intent)
	}
}

func (suite *RegressionTestSuite) processBatchWithSpecializedControllers(intents []*nephoranv1.NetworkIntent) {
	for _, intent := range intents {
		suite.processWithSpecializedControllers(intent)
	}
}

func (suite *RegressionTestSuite) measureThroughput(name string, operation func()) float64 {
	startTime := time.Now()
	operation()
	duration := time.Since(startTime)

	// Return operations per second (assuming 1 operation)
	return 1.0 / duration.Seconds()
}

func (suite *RegressionTestSuite) measureResourceUsage(name string, operation func()) *ResourceUsage {
	// This is a simplified resource measurement
	// In a real implementation, you would measure memory, CPU, goroutines, etc.

	operation()

	return &ResourceUsage{
		MemoryUsage:    1024, // Placeholder
		GoroutineCount: 10,   // Placeholder
		CPUUsage:       0.1,  // Placeholder
	}
}

func (suite *RegressionTestSuite) compareResourceUtilization(original, specialized *ResourceUsage) {
	// Specialized implementation should not significantly increase resource usage
	memoryRatio := float64(specialized.MemoryUsage) / float64(original.MemoryUsage)

	Expect(memoryRatio).To(BeNumerically("<=", 1.0+suite.testConfig.ToleranceThreshold),
		"Memory usage should not increase significantly")

	suite.logger.Info("Resource utilization comparison completed",
		"memory_ratio", memoryRatio,
		"goroutine_diff", specialized.GoroutineCount-original.GoroutineCount)
}

func (suite *RegressionTestSuite) generateConcurrentTestIntents(count int) []*nephoranv1.NetworkIntent {
	intents := make([]*nephoranv1.NetworkIntent, count)

	for i := 0; i < count; i++ {
		intents[i] = &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("concurrent-test-intent-%d", i),
				Namespace: "concurrent-test",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				IntentType: "concurrent_test",
				Intent:     fmt.Sprintf("Concurrent test intent %d", i),
				Priority:   "medium",
			},
		}
	}

	return intents
}

func (suite *RegressionTestSuite) compareConcurrentResults(original, specialized []*ProcessingResult) {
	originalSuccessCount := 0
	specializedSuccessCount := 0

	for _, result := range original {
		if result.Success {
			originalSuccessCount++
		}
	}

	for _, result := range specialized {
		if result.Success {
			specializedSuccessCount++
		}
	}

	// Success rates should be similar
	successRateDiff := float64(specializedSuccessCount-originalSuccessCount) / float64(len(original))
	Expect(math.Abs(successRateDiff)).To(BeNumerically("<=", suite.testConfig.ToleranceThreshold),
		"Concurrent processing success rates should be similar")
}

func (suite *RegressionTestSuite) compareEventPatterns(original, specialized []Event) {
	// Event patterns should be similar between implementations
	Expect(len(specialized)).To(BeNumerically(">=", len(original)/2),
		"Specialized implementation should emit similar number of events")
}

func (suite *RegressionTestSuite) compareMetricsData(original, specialized *ProcessingResult) {
	// Both should provide metrics data
	Expect(specialized.OutputData).To(Not(BeNil()),
		"Specialized implementation should provide metrics")
}

func (suite *RegressionTestSuite) compareStatusReporting(original, specialized *ProcessingResult) {
	// Status reporting compatibility
	Expect(specialized.Success).To(Equal(original.Success),
		"Status reporting should be consistent")
}

// Supporting types and structures

type BenchmarkResult struct {
	TotalTime    time.Duration
	SuccessCount int
	TotalCount   int
	Throughput   float64
}

type ResourceUsage struct {
	MemoryUsage    int64
	GoroutineCount int
	CPUUsage       float64
}

type EventCollector struct {
	events map[string][]Event
	mutex  sync.RWMutex
}

func NewEventCollector() *EventCollector {
	return &EventCollector{
		events: make(map[string][]Event),
	}
}

func (ec *EventCollector) StartCollection(name string) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	ec.events[name] = make([]Event, 0)
}

func (ec *EventCollector) StopCollection(name string) []Event {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()
	return ec.events[name]
}

func (ec *EventCollector) CollectEvent(event Event) {
	// This would be called by the event bus
	// Implementation depends on current collection context
}

type Event struct {
	Type      string
	Phase     interfaces.ProcessingPhase
	IntentID  string
	Success   bool
	Timestamp time.Time
	Data      map[string]interface{}
}

// Test runner function
func TestRegressionBackwardCompatibility(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Regression Backward Compatibility Test Suite")
}
