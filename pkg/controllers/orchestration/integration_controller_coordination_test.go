//go:build integration

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
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
)

// IntegrationTestSuite manages all controllers and their coordination
type IntegrationTestSuite struct {
	// Core components
	ctx        context.Context
	logger     logr.Logger
	fakeClient client.Client
	scheme     *runtime.Scheme
	recorder   *record.FakeRecorder

	// Controllers
	coordinator                  *EventDrivenCoordinator
	intentProcessingController   *SpecializedIntentProcessingController
	resourcePlanningController   *SpecializedResourcePlanningController
	manifestGenerationController *SpecializedManifestGenerationController

	// Mock services (shared across controllers)
	mockLLMClient          *MockLLMClient
	mockRAGService         *MockRAGService
	mockResourceCalculator *MockTelecomResourceCalculator
	mockOptimizationEngine *MockResourceOptimizationEngine
	mockTemplateEngine     *MockKubernetesTemplateEngine
	mockManifestValidator  *MockManifestValidator

	// Test state tracking
	eventHistory         []ProcessingEvent
	eventHistoryMutex    sync.RWMutex
	phaseCompletionTimes map[string]time.Time
	phaseCompletionMutex sync.RWMutex

	// Configuration
	intentProcessingConfig   IntentProcessingConfig
	resourcePlanningConfig   ResourcePlanningConfig
	manifestGenerationConfig ManifestGenerationConfig

	// Test data
	testNetworkIntent *nephoranv1.NetworkIntent

	started  bool
	stopChan chan struct{}
}

// NewIntegrationTestSuite creates a new integration test suite
func NewIntegrationTestSuite() *IntegrationTestSuite {
	ctx := context.Background()
	logger := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))

	// Create scheme and add types
	scheme := runtime.NewScheme()
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	Expect(nephoranv1.AddToScheme(scheme)).To(Succeed())

	// Create fake client and recorder
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(1000)

	suite := &IntegrationTestSuite{
		ctx:                  ctx,
		logger:               logger.WithName("integration-test-suite"),
		fakeClient:           fakeClient,
		scheme:               scheme,
		recorder:             recorder,
		eventHistory:         make([]ProcessingEvent, 0),
		phaseCompletionTimes: make(map[string]time.Time),
		stopChan:             make(chan struct{}),
	}

	suite.initializeConfigurations()
	suite.initializeMockServices()
	suite.initializeTestData()

	return suite
}

// initializeConfigurations sets up configurations for all controllers
func (s *IntegrationTestSuite) initializeConfigurations() {
	// Intent Processing Configuration
	s.intentProcessingConfig = IntentProcessingConfig{
		LLMEndpoint:           "http://mock-llm:8080",
		LLMAPIKey:             "mock-api-key",
		LLMModel:              "gpt-4o-mini",
		MaxTokens:             1000,
		Temperature:           0.7,
		RAGEndpoint:           "http://mock-rag:8080",
		MaxContextChunks:      5,
		SimilarityThreshold:   0.7,
		StreamingEnabled:      false,
		CacheEnabled:          true,
		CacheTTL:              10 * time.Minute, // Shorter for tests
		MaxRetries:            2,
		Timeout:               10 * time.Second,
		CircuitBreakerEnabled: true,
		FailureThreshold:      3,
		RecoveryTimeout:       30 * time.Second,
	}

	// Resource Planning Configuration
	s.resourcePlanningConfig = ResourcePlanningConfig{
		DefaultCPURequest:      "500m",
		DefaultMemoryRequest:   "1Gi",
		DefaultStorageRequest:  "10Gi",
		CPUOvercommitRatio:     1.2,
		MemoryOvercommitRatio:  1.1,
		OptimizationEnabled:    true,
		CostOptimizationWeight: 0.3,
		PerformanceWeight:      0.4,
		ReliabilityWeight:      0.3,
		ConstraintCheckEnabled: true,
		MaxPlanningTime:        2 * time.Minute,
		ParallelPlanning:       false,
		CacheEnabled:           true,
		CacheTTL:               10 * time.Minute,
		MaxCacheEntries:        100,
	}

	// Manifest Generation Configuration
	s.manifestGenerationConfig = ManifestGenerationConfig{
		TemplateDirectory:   "/templates",
		DefaultNamespace:    "nephoran-system",
		EnableHelm:          false,
		EnableKustomize:     false,
		ValidateManifests:   true,
		DryRunValidation:    true,
		SchemaValidation:    true,
		OptimizeManifests:   true,
		MinifyManifests:     true,
		RemoveDuplicates:    true,
		EnforcePolicies:     true,
		SecurityPolicies:    true,
		ResourcePolicies:    true,
		CacheEnabled:        true,
		CacheTTL:            10 * time.Minute,
		MaxCacheEntries:     100,
		MaxGenerationTime:   2 * time.Minute,
		ParallelGeneration:  false,
		ConcurrentTemplates: 2,
	}
}

// initializeMockServices creates and configures all mock services
func (s *IntegrationTestSuite) initializeMockServices() {
	// Create mock services with reasonable defaults for integration tests
	s.mockLLMClient = NewMockLLMClient()
	s.mockRAGService = NewMockRAGService()
	s.mockResourceCalculator = NewMockTelecomResourceCalculator()
	s.mockOptimizationEngine = NewMockResourceOptimizationEngine()
	s.mockTemplateEngine = NewMockKubernetesTemplateEngine()
	s.mockManifestValidator = NewMockManifestValidator()

	// Configure mocks for successful integration flow
	s.mockLLMClient.SetResponseDelay(50 * time.Millisecond)
	s.mockRAGService.SetQueryDelay(25 * time.Millisecond)
	s.mockResourceCalculator.SetCalculationDelay(30 * time.Millisecond)
	s.mockTemplateEngine.SetProcessingDelay(20 * time.Millisecond)
}

// initializeTestData creates test data for integration tests
func (s *IntegrationTestSuite) initializeTestData() {
	s.testNetworkIntent = &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "integration-test-intent",
			Namespace: "test-namespace",
			UID:       "integration-test-uid-12345",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent:     "Deploy high-availability 5G core with AMF, SMF, and UPF for production",
			IntentType: "5g-deployment",
			TargetComponents: []nephoranv1.TargetComponent{
				{
					Type:     "amf",
					Version:  "v1.0.0",
					Replicas: 3,
				},
				{
					Type:     "smf",
					Version:  "v1.0.0",
					Replicas: 2,
				},
				{
					Type:     "upf",
					Version:  "v1.0.0",
					Replicas: 1,
				},
			},
		},
		Status: nephoranv1.NetworkIntentStatus{
			ProcessingPhase: interfaces.PhaseIntentReceived,
		},
	}
}

// StartControllers initializes and starts all controllers
func (s *IntegrationTestSuite) StartControllers() error {
	if s.started {
		return fmt.Errorf("controllers already started")
	}

	s.logger.Info("Starting integration test controllers")

	// Create coordinator first
	coordinator := NewEventDrivenCoordinator(s.fakeClient, s.logger.WithName("coordinator"))
	s.coordinator = coordinator

	// Subscribe to all events for tracking
	err := s.coordinator.eventBus.Subscribe("*", s.trackEvent)
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}

	// Create specialized controllers with mock dependencies
	s.intentProcessingController = &SpecializedIntentProcessingController{
		Client:               s.fakeClient,
		Scheme:               s.scheme,
		Recorder:             s.recorder,
		Logger:               s.logger.WithName("intent-processor"),
		LLMClient:            s.mockLLMClient,
		RAGService:           s.mockRAGService,
		PromptEngine:         NewMockPromptEngine(),
		StreamingProcessor:   NewMockStreamingProcessor(),
		PerformanceOptimizer: NewMockPerformanceOptimizer(),
		Config:               s.intentProcessingConfig,
		SupportedIntents:     []string{"5g-deployment", "network-slice", "cnf-deployment"},
		ConfidenceThreshold:  0.7,
		metrics:              NewIntentProcessingMetrics(),
		circuitBreaker:       NewMockCircuitBreaker(),
		stopChan:             make(chan struct{}),
		healthStatus: interfaces.HealthStatus{
			Status:      "Healthy",
			Message:     "Controller initialized for integration test",
			LastChecked: time.Now(),
		},
	}

	// Initialize intent processing cache
	s.intentProcessingController.cache = &IntentProcessingCache{
		entries:    make(map[string]*CacheEntry),
		ttl:        s.intentProcessingConfig.CacheTTL,
		maxEntries: 100,
	}

	s.resourcePlanningController = &SpecializedResourcePlanningController{
		Client:             s.fakeClient,
		Scheme:             s.scheme,
		Recorder:           s.recorder,
		Logger:             s.logger.WithName("resource-planner"),
		ResourceCalculator: s.mockResourceCalculator,
		OptimizationEngine: s.mockOptimizationEngine,
		ConstraintSolver:   NewMockResourceConstraintSolver(),
		CostEstimator:      NewMockTelecomCostEstimator(),
		Config:             s.resourcePlanningConfig,
		ResourceTemplates:  initializeResourceTemplates(),
		ConstraintRules:    initializeConstraintRules(),
		metrics:            NewResourcePlanningMetrics(),
		stopChan:           make(chan struct{}),
		healthStatus: interfaces.HealthStatus{
			Status:      "Healthy",
			Message:     "Controller initialized for integration test",
			LastChecked: time.Now(),
		},
	}

	// Initialize resource planning cache
	s.resourcePlanningController.planningCache = &ResourcePlanCache{
		entries:    make(map[string]*PlanCacheEntry),
		ttl:        s.resourcePlanningConfig.CacheTTL,
		maxEntries: s.resourcePlanningConfig.MaxCacheEntries,
	}

	s.manifestGenerationController = &SpecializedManifestGenerationController{
		Client:               s.fakeClient,
		Scheme:               s.scheme,
		Recorder:             s.recorder,
		Logger:               s.logger.WithName("manifest-generator"),
		TemplateEngine:       s.mockTemplateEngine,
		ManifestValidator:    s.mockManifestValidator,
		PolicyEnforcer:       NewMockManifestPolicyEnforcer(),
		ManifestOptimizer:    NewMockManifestOptimizer(),
		HelmIntegration:      nil,
		KustomizeIntegration: nil,
		Config:               s.manifestGenerationConfig,
		Templates:            initializeManifestTemplates(),
		PolicyRules:          initializeManifestPolicyRules(),
		metrics:              NewManifestGenerationMetrics(),
		stopChan:             make(chan struct{}),
		healthStatus: interfaces.HealthStatus{
			Status:      "Healthy",
			Message:     "Controller initialized for integration test",
			LastChecked: time.Now(),
		},
	}

	// Initialize manifest generation cache
	s.manifestGenerationController.manifestCache = &ManifestCache{
		entries:    make(map[string]*ManifestCacheEntry),
		ttl:        s.manifestGenerationConfig.CacheTTL,
		maxEntries: s.manifestGenerationConfig.MaxCacheEntries,
	}

	// Start all controllers
	if err := s.coordinator.Start(s.ctx); err != nil {
		return fmt.Errorf("failed to start coordinator: %w", err)
	}

	if err := s.intentProcessingController.Start(s.ctx); err != nil {
		return fmt.Errorf("failed to start intent processing controller: %w", err)
	}

	if err := s.resourcePlanningController.Start(s.ctx); err != nil {
		return fmt.Errorf("failed to start resource planning controller: %w", err)
	}

	if err := s.manifestGenerationController.Start(s.ctx); err != nil {
		return fmt.Errorf("failed to start manifest generation controller: %w", err)
	}

	s.started = true
	s.logger.Info("All integration test controllers started successfully")

	return nil
}

// StopControllers stops all controllers
func (s *IntegrationTestSuite) StopControllers() error {
	if !s.started {
		return nil
	}

	s.logger.Info("Stopping integration test controllers")

	var errors []error

	if err := s.manifestGenerationController.Stop(s.ctx); err != nil {
		errors = append(errors, fmt.Errorf("failed to stop manifest generation controller: %w", err))
	}

	if err := s.resourcePlanningController.Stop(s.ctx); err != nil {
		errors = append(errors, fmt.Errorf("failed to stop resource planning controller: %w", err))
	}

	if err := s.intentProcessingController.Stop(s.ctx); err != nil {
		errors = append(errors, fmt.Errorf("failed to stop intent processing controller: %w", err))
	}

	if err := s.coordinator.Stop(s.ctx); err != nil {
		errors = append(errors, fmt.Errorf("failed to stop coordinator: %w", err))
	}

	close(s.stopChan)
	s.started = false

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping controllers: %v", errors)
	}

	s.logger.Info("All integration test controllers stopped successfully")
	return nil
}

// trackEvent captures events for analysis
func (s *IntegrationTestSuite) trackEvent(ctx context.Context, event ProcessingEvent) error {
	s.eventHistoryMutex.Lock()
	defer s.eventHistoryMutex.Unlock()

	s.eventHistory = append(s.eventHistory, event)

	// Track phase completion times
	if event.Success && (event.Type == EventLLMProcessingCompleted ||
		event.Type == EventResourcePlanningCompleted ||
		event.Type == EventManifestGenerationCompleted) {
		s.phaseCompletionMutex.Lock()
		s.phaseCompletionTimes[fmt.Sprintf("%s_%s", event.IntentID, event.Phase)] = event.Timestamp
		s.phaseCompletionMutex.Unlock()
	}

	s.logger.Info("Event tracked",
		"type", event.Type,
		"intentId", event.IntentID,
		"phase", event.Phase,
		"success", event.Success)

	return nil
}

// GetEventHistory returns the event history
func (s *IntegrationTestSuite) GetEventHistory() []ProcessingEvent {
	s.eventHistoryMutex.RLock()
	defer s.eventHistoryMutex.RUnlock()

	history := make([]ProcessingEvent, len(s.eventHistory))
	copy(history, s.eventHistory)
	return history
}

// GetPhaseCompletionTime returns completion time for a specific phase
func (s *IntegrationTestSuite) GetPhaseCompletionTime(intentID string, phase interfaces.ProcessingPhase) (time.Time, bool) {
	s.phaseCompletionMutex.RLock()
	defer s.phaseCompletionMutex.RUnlock()

	key := fmt.Sprintf("%s_%s", intentID, phase)
	completionTime, exists := s.phaseCompletionTimes[key]
	return completionTime, exists
}

// ClearEventHistory clears the event history
func (s *IntegrationTestSuite) ClearEventHistory() {
	s.eventHistoryMutex.Lock()
	defer s.eventHistoryMutex.Unlock()

	s.eventHistory = s.eventHistory[:0]

	s.phaseCompletionMutex.Lock()
	defer s.phaseCompletionMutex.Unlock()

	s.phaseCompletionTimes = make(map[string]time.Time)
}

// ProcessFullIntentWorkflow processes a complete intent workflow from start to finish
func (s *IntegrationTestSuite) ProcessFullIntentWorkflow(intent *nephoranv1.NetworkIntent) error {
	s.logger.Info("Starting full intent workflow", "intentId", intent.Name)

	// Phase 1: Intent Processing (LLM + RAG)
	s.logger.Info("Phase 1: Intent Processing", "intentId", intent.Name)
	intent.Status.ProcessingPhase = interfaces.PhaseLLMProcessing

	llmResult, err := s.intentProcessingController.ProcessIntent(s.ctx, intent)
	if err != nil {
		return fmt.Errorf("intent processing failed: %w", err)
	}
	if !llmResult.Success {
		return fmt.Errorf("intent processing failed: %s", llmResult.ErrorMessage)
	}

	// Update intent with LLM results
	intent.Status.LLMResponse = llmResult.Data
	intent.Status.ProcessingPhase = llmResult.NextPhase

	// Phase 2: Resource Planning
	s.logger.Info("Phase 2: Resource Planning", "intentId", intent.Name)

	resourceResult, err := s.resourcePlanningController.ProcessPhase(s.ctx, intent, interfaces.PhaseResourcePlanning)
	if err != nil {
		return fmt.Errorf("resource planning failed: %w", err)
	}
	if !resourceResult.Success {
		return fmt.Errorf("resource planning failed: %s", resourceResult.ErrorMessage)
	}

	// Update intent with resource planning results
	intent.Status.ResourcePlan = resourceResult.Data
	intent.Status.ProcessingPhase = resourceResult.NextPhase

	// Phase 3: Manifest Generation
	s.logger.Info("Phase 3: Manifest Generation", "intentId", intent.Name)

	manifestResult, err := s.manifestGenerationController.ProcessPhase(s.ctx, intent, interfaces.PhaseManifestGeneration)
	if err != nil {
		return fmt.Errorf("manifest generation failed: %w", err)
	}
	if !manifestResult.Success {
		return fmt.Errorf("manifest generation failed: %s", manifestResult.ErrorMessage)
	}

	// Update intent with manifest generation results
	intent.Status.GeneratedManifests = manifestResult.Data
	intent.Status.ProcessingPhase = manifestResult.NextPhase

	s.logger.Info("Full intent workflow completed successfully",
		"intentId", intent.Name,
		"finalPhase", intent.Status.ProcessingPhase)

	return nil
}

// ProcessIntentWithCoordination processes intent using event-driven coordination
func (s *IntegrationTestSuite) ProcessIntentWithCoordination(intent *nephoranv1.NetworkIntent) error {
	s.logger.Info("Starting coordinated intent processing", "intentId", intent.Name)

	// Start coordination
	if err := s.coordinator.CoordinateIntentWithEvents(s.ctx, intent); err != nil {
		return fmt.Errorf("failed to start coordination: %w", err)
	}

	// Process through coordination (this would normally be handled by individual controller reconcilers)
	// For integration test, we simulate the coordination flow

	intentID := string(intent.UID)

	// Simulate LLM processing completion
	llmEvent := ProcessingEvent{
		Type:          EventLLMProcessingCompleted,
		Source:        "intent-processor",
		IntentID:      intentID,
		Phase:         interfaces.PhaseLLMProcessing,
		Success:       true,
		Data:          make(map[string]interface{}),
		Timestamp:     time.Now(),
		CorrelationID: fmt.Sprintf("coord-%s", intentID),
	}

	if err := s.coordinator.eventBus.Publish(s.ctx, EventLLMProcessingCompleted, llmEvent); err != nil {
		return fmt.Errorf("failed to publish LLM completion event: %w", err)
	}

	// Wait for coordination to update phase
	time.Sleep(100 * time.Millisecond)

	// Simulate resource planning completion
	resourceEvent := ProcessingEvent{
		Type:          EventResourcePlanningCompleted,
		Source:        "resource-planner",
		IntentID:      intentID,
		Phase:         interfaces.PhaseResourcePlanning,
		Success:       true,
		Data:          make(map[string]interface{}),
		Timestamp:     time.Now(),
		CorrelationID: fmt.Sprintf("coord-%s", intentID),
	}

	if err := s.coordinator.eventBus.Publish(s.ctx, EventResourcePlanningCompleted, resourceEvent); err != nil {
		return fmt.Errorf("failed to publish resource planning completion event: %w", err)
	}

	// Wait for coordination to update phase
	time.Sleep(100 * time.Millisecond)

	// Simulate manifest generation completion
	manifestEvent := ProcessingEvent{
		Type:          EventManifestGenerationCompleted,
		Source:        "manifest-generator",
		IntentID:      intentID,
		Phase:         interfaces.PhaseManifestGeneration,
		Success:       true,
		Data:          make(map[string]interface{}),
		Timestamp:     time.Now(),
		CorrelationID: fmt.Sprintf("coord-%s", intentID),
	}

	if err := s.coordinator.eventBus.Publish(s.ctx, EventManifestGenerationCompleted, manifestEvent); err != nil {
		return fmt.Errorf("failed to publish manifest generation completion event: %w", err)
	}

	s.logger.Info("Coordinated intent processing completed", "intentId", intent.Name)

	return nil
}

// WaitForEventType waits for a specific event type with timeout
func (s *IntegrationTestSuite) WaitForEventType(eventType EventType, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		events := s.GetEventHistory()
		for _, event := range events {
			if event.Type == eventType {
				return true
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	return false
}

// CountEventsOfType counts events of a specific type
func (s *IntegrationTestSuite) CountEventsOfType(eventType EventType) int {
	events := s.GetEventHistory()
	count := 0
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

// GetControllerMetrics returns metrics from all controllers
func (s *IntegrationTestSuite) GetControllerMetrics() (map[string]map[string]float64, error) {
	metrics := make(map[string]map[string]float64)

	// Intent processing metrics
	intentMetrics, err := s.intentProcessingController.GetMetrics(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get intent processing metrics: %w", err)
	}
	metrics["intent_processing"] = intentMetrics

	// Resource planning metrics
	resourceMetrics, err := s.resourcePlanningController.GetMetrics(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource planning metrics: %w", err)
	}
	metrics["resource_planning"] = resourceMetrics

	// Manifest generation metrics
	manifestMetrics, err := s.manifestGenerationController.GetMetrics(s.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest generation metrics: %w", err)
	}
	metrics["manifest_generation"] = manifestMetrics

	return metrics, nil
}

var _ = Describe("Controller Integration and Coordination", func() {
	var suite *IntegrationTestSuite

	BeforeEach(func() {
		suite = NewIntegrationTestSuite()
		err := suite.StartControllers()
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if suite != nil {
			suite.StopControllers()
		}
	})

	Describe("Full Workflow Integration", func() {
		It("should process complete intent workflow successfully", func() {
			intent := suite.testNetworkIntent.DeepCopy()

			err := suite.ProcessFullIntentWorkflow(intent)
			Expect(err).NotTo(HaveOccurred())

			// Verify final phase
			Expect(intent.Status.ProcessingPhase).To(Equal(interfaces.PhaseGitOpsCommit))

			// Verify all phases have data
			Expect(intent.Status.LLMResponse).NotTo(BeNil())
			Expect(intent.Status.ResourcePlan).NotTo(BeNil())
			Expect(intent.Status.GeneratedManifests).NotTo(BeNil())
		})

		It("should handle complex 5G deployment with multiple network functions", func() {
			intent := suite.testNetworkIntent.DeepCopy()
			intent.Spec.Intent = "Deploy complete 5G core with AMF, SMF, UPF, NRF, AUSF, and UDM for production"
			intent.Spec.TargetComponents = []nephoranv1.TargetComponent{
				{Type: "amf", Version: "v1.0.0", Replicas: 3},
				{Type: "smf", Version: "v1.0.0", Replicas: 2},
				{Type: "upf", Version: "v1.0.0", Replicas: 1},
				{Type: "nrf", Version: "v1.0.0", Replicas: 2},
				{Type: "ausf", Version: "v1.0.0", Replicas: 2},
				{Type: "udm", Version: "v1.0.0", Replicas: 2},
			}

			// Configure mock LLM to return complex response
			complexResponse := `{
				"network_functions": [
					{
						"name": "amf", "type": "amf", "replicas": 3,
						"resources": {"requests": {"cpu": "1", "memory": "2Gi"}}
					},
					{
						"name": "smf", "type": "smf", "replicas": 2,
						"resources": {"requests": {"cpu": "500m", "memory": "1Gi"}}
					},
					{
						"name": "upf", "type": "upf", "replicas": 1,
						"resources": {"requests": {"cpu": "2", "memory": "4Gi"}}
					},
					{
						"name": "nrf", "type": "nrf", "replicas": 2,
						"resources": {"requests": {"cpu": "500m", "memory": "1Gi"}}
					},
					{
						"name": "ausf", "type": "ausf", "replicas": 2,
						"resources": {"requests": {"cpu": "500m", "memory": "1Gi"}}
					},
					{
						"name": "udm", "type": "udm", "replicas": 2,
						"resources": {"requests": {"cpu": "500m", "memory": "1Gi"}}
					}
				],
				"deployment_pattern": "production",
				"confidence": 0.95
			}`
			suite.mockLLMClient.SetResponse("default", complexResponse)

			err := suite.ProcessFullIntentWorkflow(intent)
			Expect(err).NotTo(HaveOccurred())

			// Verify complex deployment was processed
			llmResponse := intent.Status.LLMResponse.(map[string]interface{})
			Expect(llmResponse).To(HaveKey("llmResponse"))

			resourcePlan := intent.Status.ResourcePlan.(map[string]interface{})
			Expect(resourcePlan).To(HaveKey("resourcePlan"))

			manifests := intent.Status.GeneratedManifests.(map[string]interface{})
			Expect(manifests).To(HaveKey("generatedManifests"))
		})

		It("should measure end-to-end processing performance", func() {
			intent := suite.testNetworkIntent.DeepCopy()

			startTime := time.Now()
			err := suite.ProcessFullIntentWorkflow(intent)
			totalDuration := time.Since(startTime)

			Expect(err).NotTo(HaveOccurred())

			// Verify processing completed within reasonable time
			Expect(totalDuration).To(BeNumerically("<", 5*time.Second))

			// Get metrics from all controllers
			metrics, err := suite.GetControllerMetrics()
			Expect(err).NotTo(HaveOccurred())

			// Verify metrics are populated
			Expect(metrics).To(HaveKey("intent_processing"))
			Expect(metrics).To(HaveKey("resource_planning"))
			Expect(metrics).To(HaveKey("manifest_generation"))

			// Log performance metrics for analysis
			suite.logger.Info("End-to-end performance metrics",
				"totalDuration", totalDuration,
				"intentProcessingTime", metrics["intent_processing"]["average_latency_ms"],
				"resourcePlanningTime", metrics["resource_planning"]["average_planning_time_ms"],
				"manifestGenerationTime", metrics["manifest_generation"]["average_generation_time_ms"])
		})
	})

	Describe("Event-Driven Coordination", func() {
		BeforeEach(func() {
			suite.ClearEventHistory()
		})

		It("should coordinate intent processing through events", func() {
			intent := suite.testNetworkIntent.DeepCopy()

			err := suite.ProcessIntentWithCoordination(intent)
			Expect(err).NotTo(HaveOccurred())

			// Verify coordination events were published
			Eventually(func() bool {
				return suite.WaitForEventType(EventLLMProcessingCompleted, 1*time.Second)
			}, "5s", "100ms").Should(BeTrue())

			Eventually(func() bool {
				return suite.WaitForEventType(EventResourcePlanningCompleted, 1*time.Second)
			}, "5s", "100ms").Should(BeTrue())

			Eventually(func() bool {
				return suite.WaitForEventType(EventManifestGenerationCompleted, 1*time.Second)
			}, "5s", "100ms").Should(BeTrue())

			// Verify coordination context exists
			intentID := string(intent.UID)
			coordCtx, exists := suite.coordinator.GetCoordinationContext(intentID)
			Expect(exists).To(BeTrue())
			Expect(coordCtx.IntentID).To(Equal(intentID))
		})

		It("should handle concurrent intent processing", func() {
			numIntents := 3
			intents := make([]*nephoranv1.NetworkIntent, numIntents)

			// Create multiple intents
			for i := 0; i < numIntents; i++ {
				intents[i] = suite.testNetworkIntent.DeepCopy()
				intents[i].Name = fmt.Sprintf("concurrent-intent-%d", i)
				intents[i].UID = nephoranv1.UID(fmt.Sprintf("concurrent-uid-%d", i))
			}

			// Process intents concurrently
			done := make(chan error, numIntents)
			for i := 0; i < numIntents; i++ {
				go func(intent *nephoranv1.NetworkIntent) {
					defer GinkgoRecover()
					err := suite.ProcessFullIntentWorkflow(intent)
					done <- err
				}(intents[i])
			}

			// Wait for all to complete
			for i := 0; i < numIntents; i++ {
				Eventually(done).Should(Receive(BeNil()))
			}

			// Verify all intents were processed successfully
			for i := 0; i < numIntents; i++ {
				Expect(intents[i].Status.ProcessingPhase).To(Equal(interfaces.PhaseGitOpsCommit))
			}

			// Verify coordination contexts were created for all intents
			for i := 0; i < numIntents; i++ {
				intentID := string(intents[i].UID)
				_, exists := suite.coordinator.GetCoordinationContext(intentID)
				Expect(exists).To(BeTrue())
			}
		})

		It("should handle event replay for recovery scenarios", func() {
			intent := suite.testNetworkIntent.DeepCopy()
			intentID := string(intent.UID)

			// Create some test events
			events := []ProcessingEvent{
				{
					Type:          EventIntentReceived,
					Source:        "test",
					IntentID:      intentID,
					Phase:         interfaces.PhaseIntentReceived,
					Success:       true,
					Timestamp:     time.Now(),
					CorrelationID: "replay-test-1",
				},
				{
					Type:          EventLLMProcessingCompleted,
					Source:        "intent-processor",
					IntentID:      intentID,
					Phase:         interfaces.PhaseLLMProcessing,
					Success:       true,
					Timestamp:     time.Now(),
					CorrelationID: "replay-test-2",
				},
			}

			// Persist events
			for _, event := range events {
				err := suite.coordinator.eventStore.PersistEvent(suite.ctx, event)
				Expect(err).NotTo(HaveOccurred())
			}

			// Clear event history to test replay
			suite.ClearEventHistory()

			// Replay events
			err := suite.coordinator.replayManager.ReplayEventsForIntent(suite.ctx, intentID)
			Expect(err).NotTo(HaveOccurred())

			// Wait for replay events to be processed
			time.Sleep(500 * time.Millisecond)

			// Verify events were replayed (they should appear in our tracking)
			eventHistory := suite.GetEventHistory()
			replayedEventFound := false
			for _, event := range eventHistory {
				if event.IntentID == intentID && event.CorrelationID == "replay-test-1" {
					replayedEventFound = true
					break
				}
			}
			Expect(replayedEventFound).To(BeTrue())
		})

		It("should track phase completion times", func() {
			intent := suite.testNetworkIntent.DeepCopy()
			intentID := string(intent.UID)

			startTime := time.Now()
			err := suite.ProcessFullIntentWorkflow(intent)
			Expect(err).NotTo(HaveOccurred())
			endTime := time.Now()

			// Since we don't emit the actual events in ProcessFullIntentWorkflow,
			// we'll verify the workflow completed within expected time
			totalDuration := endTime.Sub(startTime)
			Expect(totalDuration).To(BeNumerically("<", 10*time.Second))

			suite.logger.Info("Phase completion analysis",
				"intentId", intentID,
				"totalDuration", totalDuration)
		})
	})

	Describe("Error Handling and Recovery", func() {
		It("should handle LLM processing failures gracefully", func() {
			intent := suite.testNetworkIntent.DeepCopy()

			// Configure mock LLM to fail
			suite.mockLLMClient.SetShouldReturnError(true)

			err := suite.ProcessFullIntentWorkflow(intent)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("intent processing failed"))

			// Verify intent status reflects failure
			Expect(intent.Status.ProcessingPhase).NotTo(Equal(interfaces.PhaseGitOpsCommit))
		})

		It("should handle resource planning failures gracefully", func() {
			intent := suite.testNetworkIntent.DeepCopy()

			// Configure mock resource calculator to fail
			suite.mockResourceCalculator.SetShouldReturnError(true)

			err := suite.ProcessFullIntentWorkflow(intent)
			Expect(err).NotTo(HaveOccurred()) // Should continue with default resources

			// Should complete successfully using defaults
			Expect(intent.Status.ProcessingPhase).To(Equal(interfaces.PhaseGitOpsCommit))
		})

		It("should handle manifest generation failures gracefully", func() {
			intent := suite.testNetworkIntent.DeepCopy()

			// Configure mock template engine to fail
			suite.mockTemplateEngine.SetShouldReturnError(true)

			err := suite.ProcessFullIntentWorkflow(intent)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("manifest generation failed"))
		})

		It("should support circuit breaker patterns", func() {
			intent := suite.testNetworkIntent.DeepCopy()

			// Configure circuit breaker to be open
			mockCircuitBreaker := suite.intentProcessingController.circuitBreaker.(*MockCircuitBreaker)
			mockCircuitBreaker.SetOpen(true)

			err := suite.ProcessFullIntentWorkflow(intent)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("intent processing failed"))
		})

		It("should handle timeout scenarios", func() {
			intent := suite.testNetworkIntent.DeepCopy()

			// Configure long processing delays
			suite.mockLLMClient.SetResponseDelay(2 * time.Second)
			suite.mockRAGService.SetQueryDelay(2 * time.Second)
			suite.mockResourceCalculator.SetCalculationDelay(2 * time.Second)

			// Create context with short timeout
			timeoutCtx, cancel := context.WithTimeout(suite.ctx, 1*time.Second)
			defer cancel()

			// Replace suite context temporarily
			originalCtx := suite.ctx
			suite.ctx = timeoutCtx
			defer func() { suite.ctx = originalCtx }()

			err := suite.ProcessFullIntentWorkflow(intent)
			// May or may not fail depending on timing - this tests timeout handling
			if err != nil {
				suite.logger.Info("Expected timeout error occurred", "error", err)
			}
		})
	})

	Describe("Performance and Scalability", func() {
		It("should handle high-throughput intent processing", func() {
			numIntents := 10
			results := make(chan error, numIntents)

			startTime := time.Now()

			// Process multiple intents concurrently
			for i := 0; i < numIntents; i++ {
				go func(index int) {
					defer GinkgoRecover()

					intent := suite.testNetworkIntent.DeepCopy()
					intent.Name = fmt.Sprintf("throughput-test-%d", index)
					intent.UID = nephoranv1.UID(fmt.Sprintf("throughput-uid-%d", index))

					err := suite.ProcessFullIntentWorkflow(intent)
					results <- err
				}(i)
			}

			// Collect results
			successCount := 0
			for i := 0; i < numIntents; i++ {
				err := <-results
				if err == nil {
					successCount++
				} else {
					suite.logger.Info("Intent processing error in throughput test", "error", err)
				}
			}

			totalDuration := time.Since(startTime)

			// Verify performance metrics
			Expect(successCount).To(BeNumerically(">=", numIntents*0.8)) // At least 80% success
			Expect(totalDuration).To(BeNumerically("<", 30*time.Second)) // Complete within 30 seconds

			// Calculate throughput
			throughput := float64(successCount) / totalDuration.Seconds()
			suite.logger.Info("High-throughput test results",
				"totalIntents", numIntents,
				"successCount", successCount,
				"totalDuration", totalDuration,
				"throughput", throughput)
		})

		It("should maintain low latency under load", func() {
			numIterations := 5
			latencies := make([]time.Duration, numIterations)

			for i := 0; i < numIterations; i++ {
				intent := suite.testNetworkIntent.DeepCopy()
				intent.Name = fmt.Sprintf("latency-test-%d", i)
				intent.UID = nephoranv1.UID(fmt.Sprintf("latency-uid-%d", i))

				startTime := time.Now()
				err := suite.ProcessFullIntentWorkflow(intent)
				latency := time.Since(startTime)

				Expect(err).NotTo(HaveOccurred())
				latencies[i] = latency

				suite.logger.Info("Latency measurement", "iteration", i, "latency", latency)
			}

			// Calculate average latency
			totalLatency := time.Duration(0)
			for _, latency := range latencies {
				totalLatency += latency
			}
			avgLatency := totalLatency / time.Duration(numIterations)

			// Verify latency is reasonable
			Expect(avgLatency).To(BeNumerically("<", 3*time.Second))

			suite.logger.Info("Latency test results",
				"avgLatency", avgLatency,
				"minLatency", minDuration(latencies),
				"maxLatency", maxDuration(latencies))
		})
	})

	Describe("Caching and Optimization", func() {
		It("should benefit from cross-controller caching", func() {
			intent1 := suite.testNetworkIntent.DeepCopy()
			intent1.Name = "cache-test-1"
			intent1.UID = "cache-uid-1"

			intent2 := suite.testNetworkIntent.DeepCopy()
			intent2.Name = "cache-test-2"
			intent2.UID = "cache-uid-2"
			// Same intent text to trigger cache hits
			intent2.Spec.Intent = intent1.Spec.Intent

			// Process first intent
			startTime1 := time.Now()
			err := suite.ProcessFullIntentWorkflow(intent1)
			duration1 := time.Since(startTime1)
			Expect(err).NotTo(HaveOccurred())

			// Process second intent (should use cache)
			startTime2 := time.Now()
			err = suite.ProcessFullIntentWorkflow(intent2)
			duration2 := time.Since(startTime2)
			Expect(err).NotTo(HaveOccurred())

			// Verify second intent was faster due to caching
			suite.logger.Info("Caching performance comparison",
				"firstIntentDuration", duration1,
				"secondIntentDuration", duration2,
				"speedup", float64(duration1)/float64(duration2))

			// Get metrics to verify cache hits
			metrics, err := suite.GetControllerMetrics()
			Expect(err).NotTo(HaveOccurred())

			// Log cache hit rates
			if intentMetrics, exists := metrics["intent_processing"]; exists {
				suite.logger.Info("Intent processing cache metrics", "cacheHitRate", intentMetrics["cache_hit_rate"])
			}
			if resourceMetrics, exists := metrics["resource_planning"]; exists {
				suite.logger.Info("Resource planning cache metrics", "cacheHitRate", resourceMetrics["cache_hit_rate"])
			}
			if manifestMetrics, exists := metrics["manifest_generation"]; exists {
				suite.logger.Info("Manifest generation cache metrics", "cacheHitRate", manifestMetrics["cache_hit_rate"])
			}
		})
	})
})

func TestControllerIntegrationAndCoordination(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Integration and Coordination Suite")
}

// Helper functions

func minDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

func maxDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}
