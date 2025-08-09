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
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
)

// Mock implementations for testing
type MockTelecomResourceCalculator struct {
	shouldReturnError  bool
	calculationDelay   time.Duration
	baselineProfiles   map[string]*ResourceProfile
	scalingFactors     map[string]*ScalingProfile
	performanceTargets map[string]*PerformanceTarget
}

func NewMockTelecomResourceCalculator() *MockTelecomResourceCalculator {
	return &MockTelecomResourceCalculator{
		shouldReturnError: false,
		calculationDelay:  5 * time.Millisecond,
		baselineProfiles: map[string]*ResourceProfile{
			"amf": {
				NetworkFunction:  "amf",
				BaselineCPU:      resource.MustParse("500m"),
				BaselineMemory:   resource.MustParse("1Gi"),
				BaselineStorage:  resource.MustParse("10Gi"),
				NetworkBandwidth: "1Gbps",
				IOPSRequirement:  1000,
			},
			"smf": {
				NetworkFunction:  "smf",
				BaselineCPU:      resource.MustParse("500m"),
				BaselineMemory:   resource.MustParse("1Gi"),
				BaselineStorage:  resource.MustParse("10Gi"),
				NetworkBandwidth: "1Gbps",
				IOPSRequirement:  1000,
			},
			"upf": {
				NetworkFunction:  "upf",
				BaselineCPU:      resource.MustParse("2"),
				BaselineMemory:   resource.MustParse("4Gi"),
				BaselineStorage:  resource.MustParse("50Gi"),
				NetworkBandwidth: "10Gbps",
				IOPSRequirement:  5000,
			},
		},
		scalingFactors: map[string]*ScalingProfile{
			"amf": {
				NetworkFunction:      "amf",
				CPUScalingFactor:     1.2,
				MemoryScalingFactor:  1.1,
				StorageScalingFactor: 1.0,
				MaxReplicas:          10,
				MinReplicas:          2,
				ScalingThreshold:     0.8,
			},
			"upf": {
				NetworkFunction:      "upf",
				CPUScalingFactor:     1.5,
				MemoryScalingFactor:  1.3,
				StorageScalingFactor: 1.1,
				MaxReplicas:          5,
				MinReplicas:          1,
				ScalingThreshold:     0.9,
			},
		},
		performanceTargets: map[string]*PerformanceTarget{
			"amf": {
				NetworkFunction:    "amf",
				MaxLatency:         time.Millisecond * 10,
				MinThroughput:      1000.0,
				MaxErrorRate:       0.01,
				AvailabilityTarget: 0.999,
			},
			"upf": {
				NetworkFunction:    "upf",
				MaxLatency:         time.Millisecond * 5,
				MinThroughput:      10000.0,
				MaxErrorRate:       0.001,
				AvailabilityTarget: 0.9999,
			},
		},
	}
}

func (m *MockTelecomResourceCalculator) CalculateResources(nfType string, replicas int32) (*ResourceProfile, error) {
	if m.shouldReturnError {
		return nil, fmt.Errorf("mock resource calculator error")
	}

	if m.calculationDelay > 0 {
		time.Sleep(m.calculationDelay)
	}

	profile, exists := m.baselineProfiles[nfType]
	if !exists {
		// Return default profile
		return &ResourceProfile{
			NetworkFunction: nfType,
			BaselineCPU:     resource.MustParse("500m"),
			BaselineMemory:  resource.MustParse("1Gi"),
			BaselineStorage: resource.MustParse("10Gi"),
		}, nil
	}

	// Apply replica scaling
	scaledProfile := &ResourceProfile{
		NetworkFunction:  profile.NetworkFunction,
		BaselineCPU:      profile.BaselineCPU.DeepCopy(),
		BaselineMemory:   profile.BaselineMemory.DeepCopy(),
		BaselineStorage:  profile.BaselineStorage.DeepCopy(),
		NetworkBandwidth: profile.NetworkBandwidth,
		IOPSRequirement:  profile.IOPSRequirement,
		Metadata:         profile.Metadata,
	}

	// Scale resources based on replicas
	cpuValue := profile.BaselineCPU.MilliValue() * int64(replicas)
	memoryValue := profile.BaselineMemory.Value() * int64(replicas)
	storageValue := profile.BaselineStorage.Value() * int64(replicas)

	scaledProfile.BaselineCPU.SetMilli(cpuValue)
	scaledProfile.BaselineMemory.Set(memoryValue)
	scaledProfile.BaselineStorage.Set(storageValue)

	return scaledProfile, nil
}

func (m *MockTelecomResourceCalculator) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

func (m *MockTelecomResourceCalculator) SetCalculationDelay(delay time.Duration) {
	m.calculationDelay = delay
}

func (m *MockTelecomResourceCalculator) GetScalingProfile(nfType string) *ScalingProfile {
	return m.scalingFactors[nfType]
}

func (m *MockTelecomResourceCalculator) GetPerformanceTarget(nfType string) *PerformanceTarget {
	return m.performanceTargets[nfType]
}

type MockResourceOptimizationEngine struct {
	shouldReturnError   bool
	optimizationDelay   time.Duration
	optimizationEnabled bool
	costSavings         float64
}

func NewMockResourceOptimizationEngine() *MockResourceOptimizationEngine {
	return &MockResourceOptimizationEngine{
		shouldReturnError:   false,
		optimizationDelay:   10 * time.Millisecond,
		optimizationEnabled: true,
		costSavings:         0.15, // 15% cost savings
	}
}

func (m *MockResourceOptimizationEngine) OptimizeAllocation(ctx context.Context, requirements *interfaces.ResourceRequirements, config ResourcePlanningConfig) (*interfaces.OptimizedPlan, error) {
	if m.shouldReturnError {
		return nil, fmt.Errorf("mock optimization engine error")
	}

	if !m.optimizationEnabled {
		return nil, fmt.Errorf("optimization disabled")
	}

	if m.optimizationDelay > 0 {
		time.Sleep(m.optimizationDelay)
	}

	// Create original plan
	originalPlan := &interfaces.ResourcePlan{
		ResourceRequirements: *requirements,
	}

	// Create optimized requirements with reduced resources
	optimizedRequirements := *requirements

	// Apply CPU optimization
	if cpuQuantity, err := resource.ParseQuantity(requirements.CPU); err == nil {
		optimizedCPU := cpuQuantity.DeepCopy()
		optimizedValue := float64(cpuQuantity.MilliValue()) * (1 - m.costSavings)
		optimizedCPU.SetMilli(int64(optimizedValue))
		optimizedRequirements.CPU = optimizedCPU.String()
	}

	// Apply memory optimization
	if memQuantity, err := resource.ParseQuantity(requirements.Memory); err == nil {
		optimizedMem := memQuantity.DeepCopy()
		optimizedValue := float64(memQuantity.Value()) * (1 - m.costSavings)
		optimizedMem.Set(int64(optimizedValue))
		optimizedRequirements.Memory = optimizedMem.String()
	}

	optimizedPlan := &interfaces.ResourcePlan{
		ResourceRequirements: optimizedRequirements,
	}

	return &interfaces.OptimizedPlan{
		OriginalPlan:  originalPlan,
		OptimizedPlan: optimizedPlan,
		CostSavings:   m.costSavings,
		Optimizations: []interfaces.Optimization{
			{
				Type:        "resource_consolidation",
				Description: fmt.Sprintf("Applied %.1f%% resource optimization", m.costSavings*100),
				Impact:      "Reduced resource allocation while maintaining performance",
				Savings:     m.costSavings,
			},
		},
	}, nil
}

func (m *MockResourceOptimizationEngine) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

func (m *MockResourceOptimizationEngine) SetOptimizationEnabled(enabled bool) {
	m.optimizationEnabled = enabled
}

func (m *MockResourceOptimizationEngine) SetCostSavings(savings float64) {
	m.costSavings = savings
}

type MockResourceConstraintSolver struct {
	shouldReturnError    bool
	constraintViolations []string
	validationDelay      time.Duration
}

func NewMockResourceConstraintSolver() *MockResourceConstraintSolver {
	return &MockResourceConstraintSolver{
		shouldReturnError:    false,
		constraintViolations: []string{},
		validationDelay:      2 * time.Millisecond,
	}
}

func (m *MockResourceConstraintSolver) ValidateConstraints(ctx context.Context, plan *interfaces.ResourcePlan, rules []*ResourceConstraintRule) error {
	if m.shouldReturnError {
		return fmt.Errorf("mock constraint solver error")
	}

	if m.validationDelay > 0 {
		time.Sleep(m.validationDelay)
	}

	// Simulate constraint violations
	if len(m.constraintViolations) > 0 {
		return fmt.Errorf("constraint violations: %v", m.constraintViolations)
	}

	// Simulate specific constraint checks
	if cpuQuantity, err := resource.ParseQuantity(plan.ResourceRequirements.CPU); err == nil {
		if cpuQuantity.MilliValue() > 100000 { // 100 CPU cores limit
			return fmt.Errorf("CPU requirement exceeds limit: %s", plan.ResourceRequirements.CPU)
		}
	}

	if memQuantity, err := resource.ParseQuantity(plan.ResourceRequirements.Memory); err == nil {
		if memQuantity.Value() > 500*1024*1024*1024 { // 500 GB memory limit
			return fmt.Errorf("memory requirement exceeds limit: %s", plan.ResourceRequirements.Memory)
		}
	}

	return nil
}

func (m *MockResourceConstraintSolver) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

func (m *MockResourceConstraintSolver) SetConstraintViolations(violations []string) {
	m.constraintViolations = violations
}

func (m *MockResourceConstraintSolver) SetValidationDelay(delay time.Duration) {
	m.validationDelay = delay
}

type MockTelecomCostEstimator struct {
	shouldReturnError bool
	estimationDelay   time.Duration
	pricingModel      map[string]float64
}

func NewMockTelecomCostEstimator() *MockTelecomCostEstimator {
	return &MockTelecomCostEstimator{
		shouldReturnError: false,
		estimationDelay:   3 * time.Millisecond,
		pricingModel: map[string]float64{
			"cpu_core_hour":    0.05, // $0.05 per CPU core hour
			"memory_gb_hour":   0.01, // $0.01 per GB memory hour
			"storage_gb_month": 0.10, // $0.10 per GB storage per month
		},
	}
}

func (m *MockTelecomCostEstimator) EstimateCosts(ctx context.Context, plan *interfaces.ResourcePlan) (*interfaces.CostEstimate, error) {
	if m.shouldReturnError {
		return nil, fmt.Errorf("mock cost estimator error")
	}

	if m.estimationDelay > 0 {
		time.Sleep(m.estimationDelay)
	}

	costBreakdown := make(map[string]float64)

	// Parse CPU requirements
	if cpuQuantity, err := resource.ParseQuantity(plan.ResourceRequirements.CPU); err == nil {
		cpuCores := float64(cpuQuantity.MilliValue()) / 1000.0
		hourlyCPUCost := cpuCores * m.pricingModel["cpu_core_hour"]
		costBreakdown["cpu"] = hourlyCPUCost * 24 * 30 // Monthly cost
	}

	// Parse memory requirements
	if memQuantity, err := resource.ParseQuantity(plan.ResourceRequirements.Memory); err == nil {
		memoryGB := float64(memQuantity.Value()) / (1024 * 1024 * 1024)
		hourlyMemCost := memoryGB * m.pricingModel["memory_gb_hour"]
		costBreakdown["memory"] = hourlyMemCost * 24 * 30 // Monthly cost
	}

	// Parse storage requirements
	if storageQuantity, err := resource.ParseQuantity(plan.ResourceRequirements.Storage); err == nil {
		storageGB := float64(storageQuantity.Value()) / (1024 * 1024 * 1024)
		monthlyStoageCost := storageGB * m.pricingModel["storage_gb_month"]
		costBreakdown["storage"] = monthlyStoageCost
	}

	// Calculate total cost
	totalCost := 0.0
	for _, cost := range costBreakdown {
		totalCost += cost
	}

	return &interfaces.CostEstimate{
		TotalCost:      totalCost,
		Currency:       "USD",
		BillingPeriod:  "monthly",
		CostBreakdown:  costBreakdown,
		EstimationDate: time.Now(),
	}, nil
}

func (m *MockTelecomCostEstimator) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

func (m *MockTelecomCostEstimator) SetPricing(resource string, price float64) {
	m.pricingModel[resource] = price
}

var _ = Describe("SpecializedResourcePlanningController", func() {
	var (
		ctx                    context.Context
		controller             *SpecializedResourcePlanningController
		fakeClient             client.Client
		logger                 logr.Logger
		scheme                 *runtime.Scheme
		fakeRecorder           *record.FakeRecorder
		networkIntent          *nephoranv1.NetworkIntent
		mockResourceCalculator *MockTelecomResourceCalculator
		mockOptimizationEngine *MockResourceOptimizationEngine
		mockConstraintSolver   *MockResourceConstraintSolver
		mockCostEstimator      *MockTelecomCostEstimator
		config                 ResourcePlanningConfig
		sampleLLMResponse      map[string]interface{}
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))

		// Create scheme and add types
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(nephoranv1.AddToScheme(scheme)).To(Succeed())

		// Create fake client and recorder
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()
		fakeRecorder = record.NewFakeRecorder(100)

		// Create mock services
		mockResourceCalculator = NewMockTelecomResourceCalculator()
		mockOptimizationEngine = NewMockResourceOptimizationEngine()
		mockConstraintSolver = NewMockResourceConstraintSolver()
		mockCostEstimator = NewMockTelecomCostEstimator()

		// Create configuration
		config = ResourcePlanningConfig{
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
			MaxPlanningTime:        5 * time.Minute,
			ParallelPlanning:       false,
			CacheEnabled:           true,
			CacheTTL:               30 * time.Minute,
			MaxCacheEntries:        1000,
		}

		// Create controller with mock dependencies
		controller = &SpecializedResourcePlanningController{
			Client:             fakeClient,
			Scheme:             scheme,
			Recorder:           fakeRecorder,
			Logger:             logger,
			ResourceCalculator: mockResourceCalculator,
			OptimizationEngine: mockOptimizationEngine,
			ConstraintSolver:   mockConstraintSolver,
			CostEstimator:      mockCostEstimator,
			Config:             config,
			ResourceTemplates:  initializeResourceTemplates(),
			ConstraintRules:    initializeConstraintRules(),
			metrics:            NewResourcePlanningMetrics(),
			stopChan:           make(chan struct{}),
			healthStatus: interfaces.HealthStatus{
				Status:      "Healthy",
				Message:     "Resource planning controller initialized for testing",
				LastChecked: time.Now(),
			},
		}

		// Initialize cache
		controller.planningCache = &ResourcePlanCache{
			entries:    make(map[string]*PlanCacheEntry),
			ttl:        config.CacheTTL,
			maxEntries: config.MaxCacheEntries,
		}

		// Create test NetworkIntent
		networkIntent = &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-resource-planning",
				Namespace: "test-namespace",
				UID:       "test-uid-12345",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent:     "Deploy high-availability AMF and SMF for 5G core",
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
				},
			},
			Status: nephoranv1.NetworkIntentStatus{
				ProcessingPhase: interfaces.PhaseResourcePlanning,
				LLMResponse: map[string]interface{}{
					"network_functions": []interface{}{
						map[string]interface{}{
							"name":     "amf-deployment",
							"type":     "amf",
							"image":    "nephoran/amf:v1.0.0",
							"replicas": 3,
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":     "1",
									"memory":  "2Gi",
									"storage": "20Gi",
								},
								"limits": map[string]interface{}{
									"cpu":     "2",
									"memory":  "4Gi",
									"storage": "40Gi",
								},
							},
							"ports": []interface{}{
								map[string]interface{}{
									"name":        "sbi",
									"port":        80,
									"targetPort":  8080,
									"protocol":    "TCP",
									"serviceType": "ClusterIP",
								},
							},
						},
						map[string]interface{}{
							"name":     "smf-deployment",
							"type":     "smf",
							"image":    "nephoran/smf:v1.0.0",
							"replicas": 2,
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":     "500m",
									"memory":  "1Gi",
									"storage": "10Gi",
								},
								"limits": map[string]interface{}{
									"cpu":     "1",
									"memory":  "2Gi",
									"storage": "20Gi",
								},
							},
						},
					},
					"deployment_pattern": "high-availability",
					"confidence":         0.95,
					"resources": map[string]interface{}{
						"cpu":     "3",
						"memory":  "8Gi",
						"storage": "80Gi",
					},
				},
			},
		}

		// Extract LLM response for direct testing
		sampleLLMResponse = networkIntent.Status.LLMResponse.(map[string]interface{})
	})

	AfterEach(func() {
		if controller != nil && controller.started {
			controller.Stop(ctx)
		}
	})

	Describe("Controller Initialization", func() {
		It("should initialize with proper configuration", func() {
			Expect(controller).NotTo(BeNil())
			Expect(controller.ResourceCalculator).NotTo(BeNil())
			Expect(controller.OptimizationEngine).NotTo(BeNil())
			Expect(controller.ConstraintSolver).NotTo(BeNil())
			Expect(controller.CostEstimator).NotTo(BeNil())
			Expect(controller.Config.OptimizationEnabled).To(BeTrue())
			Expect(controller.Config.ConstraintCheckEnabled).To(BeTrue())
			Expect(controller.ResourceTemplates).To(HaveKey("amf"))
			Expect(controller.ResourceTemplates).To(HaveKey("smf"))
			Expect(controller.ResourceTemplates).To(HaveKey("upf"))
			Expect(controller.planningCache).NotTo(BeNil())
		})

		It("should start and stop successfully", func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(controller.started).To(BeTrue())

			err = controller.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(controller.started).To(BeFalse())
		})
	})

	Describe("Resource Planning", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should plan resources from LLM response successfully", func() {
			result, err := controller.planResourcesFromLLM(ctx, networkIntent, sampleLLMResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.NextPhase).To(Equal(interfaces.PhaseManifestGeneration))

			// Verify response data
			Expect(result.Data).To(HaveKey("resourcePlan"))
			Expect(result.Data).To(HaveKey("correlationId"))
			Expect(result.Data).To(HaveKey("networkFunctions"))

			// Verify metrics
			Expect(result.Metrics).To(HaveKey("planning_time_ms"))
			Expect(result.Metrics).To(HaveKey("calculation_time_ms"))
			Expect(result.Metrics).To(HaveKey("nfs_planned"))

			// Verify resource plan structure
			resourcePlan := result.Data["resourcePlan"].(*interfaces.ResourcePlan)
			Expect(resourcePlan).NotTo(BeNil())
			Expect(len(resourcePlan.NetworkFunctions)).To(Equal(2))
			Expect(resourcePlan.DeploymentPattern).To(Equal("high-availability"))
			Expect(len(resourcePlan.Constraints)).To(BeNumerically(">=", 2))
		})

		It("should process phase interface correctly", func() {
			result, err := controller.ProcessPhase(ctx, networkIntent, interfaces.PhaseResourcePlanning)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())
			Expect(result.NextPhase).To(Equal(interfaces.PhaseManifestGeneration))
		})

		It("should reject unsupported phases", func() {
			result, err := controller.ProcessPhase(ctx, networkIntent, interfaces.PhaseLLMProcessing)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorMessage).To(ContainSubstring("unsupported phase"))
		})

		It("should handle invalid LLM response data", func() {
			invalidIntent := networkIntent.DeepCopy()
			invalidIntent.Status.LLMResponse = "invalid-string-response"

			result, err := controller.ProcessPhase(ctx, invalidIntent, interfaces.PhaseResourcePlanning)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorCode).To(Equal("INVALID_INPUT"))
		})

		It("should parse network functions correctly", func() {
			networkFunctions, err := controller.parseNetworkFunctions(sampleLLMResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(networkFunctions)).To(Equal(2))

			// Verify AMF parsing
			amfFound := false
			smfFound := false
			for _, nf := range networkFunctions {
				switch nf.Type {
				case "amf":
					amfFound = true
					Expect(nf.Name).To(Equal("amf-deployment"))
					Expect(nf.Replicas).To(Equal(int32(3)))
					Expect(nf.Resources.Requests.CPU).To(Equal("1"))
					Expect(nf.Resources.Requests.Memory).To(Equal("2Gi"))
					Expect(len(nf.Ports)).To(Equal(1))
				case "smf":
					smfFound = true
					Expect(nf.Name).To(Equal("smf-deployment"))
					Expect(nf.Replicas).To(Equal(int32(2)))
					Expect(nf.Resources.Requests.CPU).To(Equal("500m"))
					Expect(nf.Resources.Requests.Memory).To(Equal("1Gi"))
				}
			}
			Expect(amfFound).To(BeTrue())
			Expect(smfFound).To(BeTrue())
		})

		It("should handle missing network functions gracefully", func() {
			invalidLLMResponse := map[string]interface{}{
				"deployment_pattern": "standard",
				"confidence":         0.8,
				// Missing network_functions
			}

			_, err := controller.parseNetworkFunctions(invalidLLMResponse)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no network functions found"))
		})

		It("should apply resource templates correctly", func() {
			nf := &interfaces.PlannedNetworkFunction{
				Name:     "test-amf",
				Type:     "amf",
				Replicas: 2,
				Resources: interfaces.ResourceSpec{
					Requests: interfaces.ResourceRequirements{
						CPU:     "", // Should be filled by template
						Memory:  "", // Should be filled by template
						Storage: "", // Should be filled by template
					},
				},
			}

			template := controller.ResourceTemplates["amf"]
			controller.applyResourceTemplate(nf, template)

			Expect(nf.Resources.Requests.CPU).To(Equal("500m"))
			Expect(nf.Resources.Requests.Memory).To(Equal("1Gi"))
			Expect(nf.Resources.Requests.Storage).To(Equal("10Gi"))
			Expect(nf.Resources.Limits.CPU).To(Equal("2"))
			Expect(nf.Resources.Limits.Memory).To(Equal("4Gi"))
		})
	})

	Describe("Resource Optimization", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should optimize resource allocation successfully", func() {
			requirements := &interfaces.ResourceRequirements{
				CPU:     "4",
				Memory:  "8Gi",
				Storage: "100Gi",
			}

			optimizedPlan, err := controller.OptimizeResourceAllocation(ctx, requirements)
			Expect(err).NotTo(HaveOccurred())
			Expect(optimizedPlan).NotTo(BeNil())
			Expect(optimizedPlan.OriginalPlan).NotTo(BeNil())
			Expect(optimizedPlan.OptimizedPlan).NotTo(BeNil())
			Expect(optimizedPlan.CostSavings).To(Equal(0.15))
			Expect(len(optimizedPlan.Optimizations)).To(BeNumerically(">=", 1))

			// Verify optimization actually reduced resources
			originalCPU, err := resource.ParseQuantity(optimizedPlan.OriginalPlan.ResourceRequirements.CPU)
			Expect(err).NotTo(HaveOccurred())
			optimizedCPU, err := resource.ParseQuantity(optimizedPlan.OptimizedPlan.ResourceRequirements.CPU)
			Expect(err).NotTo(HaveOccurred())
			Expect(optimizedCPU.MilliValue()).To(BeNumerically("<", originalCPU.MilliValue()))
		})

		It("should handle optimization disabled gracefully", func() {
			controller.Config.OptimizationEnabled = false

			requirements := &interfaces.ResourceRequirements{
				CPU:     "2",
				Memory:  "4Gi",
				Storage: "50Gi",
			}

			optimizedPlan, err := controller.OptimizeResourceAllocation(ctx, requirements)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("optimization not enabled"))
			Expect(optimizedPlan).To(BeNil())
		})

		It("should handle optimization engine failure", func() {
			mockOptimizationEngine.SetShouldReturnError(true)

			requirements := &interfaces.ResourceRequirements{
				CPU:     "2",
				Memory:  "4Gi",
				Storage: "50Gi",
			}

			optimizedPlan, err := controller.OptimizeResourceAllocation(ctx, requirements)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mock optimization engine error"))
			Expect(optimizedPlan).To(BeNil())
		})
	})

	Describe("Constraint Validation", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should validate constraints successfully", func() {
			resourcePlan := &interfaces.ResourcePlan{
				ResourceRequirements: interfaces.ResourceRequirements{
					CPU:     "4",
					Memory:  "8Gi",
					Storage: "100Gi",
				},
				NetworkFunctions: []interfaces.PlannedNetworkFunction{
					{
						Name:     "test-amf",
						Type:     "amf",
						Replicas: 2,
					},
				},
			}

			err := controller.ValidateResourceConstraints(ctx, resourcePlan)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should detect CPU constraint violations", func() {
			resourcePlan := &interfaces.ResourcePlan{
				ResourceRequirements: interfaces.ResourceRequirements{
					CPU:     "200", // 200 CPU cores - exceeds limit
					Memory:  "100Gi",
					Storage: "1000Gi",
				},
			}

			err := controller.ValidateResourceConstraints(ctx, resourcePlan)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("CPU requirement exceeds limit"))
		})

		It("should detect memory constraint violations", func() {
			resourcePlan := &interfaces.ResourcePlan{
				ResourceRequirements: interfaces.ResourceRequirements{
					CPU:     "10",
					Memory:  "1000Gi", // 1000 GB memory - exceeds limit
					Storage: "1000Gi",
				},
			}

			err := controller.ValidateResourceConstraints(ctx, resourcePlan)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("memory requirement exceeds limit"))
		})

		It("should handle constraint checking disabled", func() {
			controller.Config.ConstraintCheckEnabled = false

			resourcePlan := &interfaces.ResourcePlan{
				ResourceRequirements: interfaces.ResourceRequirements{
					CPU:     "200", // Would normally violate constraints
					Memory:  "1000Gi",
					Storage: "1000Gi",
				},
			}

			err := controller.ValidateResourceConstraints(ctx, resourcePlan)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle constraint solver errors", func() {
			mockConstraintSolver.SetShouldReturnError(true)

			resourcePlan := &interfaces.ResourcePlan{
				ResourceRequirements: interfaces.ResourceRequirements{
					CPU:     "4",
					Memory:  "8Gi",
					Storage: "100Gi",
				},
			}

			err := controller.ValidateResourceConstraints(ctx, resourcePlan)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mock constraint solver error"))
		})
	})

	Describe("Cost Estimation", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should estimate costs successfully", func() {
			resourcePlan := &interfaces.ResourcePlan{
				ResourceRequirements: interfaces.ResourceRequirements{
					CPU:     "4",
					Memory:  "8Gi",
					Storage: "100Gi",
				},
				NetworkFunctions: []interfaces.PlannedNetworkFunction{
					{
						Name:     "test-amf",
						Type:     "amf",
						Replicas: 2,
					},
				},
			}

			costEstimate, err := controller.EstimateResourceCosts(ctx, resourcePlan)
			Expect(err).NotTo(HaveOccurred())
			Expect(costEstimate).NotTo(BeNil())
			Expect(costEstimate.TotalCost).To(BeNumerically(">", 0))
			Expect(costEstimate.Currency).To(Equal("USD"))
			Expect(costEstimate.BillingPeriod).To(Equal("monthly"))
			Expect(costEstimate.CostBreakdown).To(HaveKey("cpu"))
			Expect(costEstimate.CostBreakdown).To(HaveKey("memory"))
			Expect(costEstimate.CostBreakdown).To(HaveKey("storage"))
		})

		It("should handle cost estimator errors", func() {
			mockCostEstimator.SetShouldReturnError(true)

			resourcePlan := &interfaces.ResourcePlan{
				ResourceRequirements: interfaces.ResourceRequirements{
					CPU:     "4",
					Memory:  "8Gi",
					Storage: "100Gi",
				},
			}

			costEstimate, err := controller.EstimateResourceCosts(ctx, resourcePlan)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("mock cost estimator error"))
			Expect(costEstimate).To(BeNil())
		})

		It("should calculate different costs for different resource levels", func() {
			smallPlan := &interfaces.ResourcePlan{
				ResourceRequirements: interfaces.ResourceRequirements{
					CPU:     "1",
					Memory:  "2Gi",
					Storage: "20Gi",
				},
			}

			largePlan := &interfaces.ResourcePlan{
				ResourceRequirements: interfaces.ResourceRequirements{
					CPU:     "8",
					Memory:  "16Gi",
					Storage: "200Gi",
				},
			}

			smallCost, err := controller.EstimateResourceCosts(ctx, smallPlan)
			Expect(err).NotTo(HaveOccurred())

			largeCost, err := controller.EstimateResourceCosts(ctx, largePlan)
			Expect(err).NotTo(HaveOccurred())

			Expect(largeCost.TotalCost).To(BeNumerically(">", smallCost.TotalCost))
		})
	})

	Describe("Caching", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should cache planning results", func() {
			// First planning - should not use cache
			result1, err := controller.planResourcesFromLLM(ctx, networkIntent, sampleLLMResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(result1.Success).To(BeTrue())
			Expect(result1.Metrics["cache_hit"]).To(Equal(float64(0)))

			// Second planning - should use cache
			result2, err := controller.planResourcesFromLLM(ctx, networkIntent, sampleLLMResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(result2.Success).To(BeTrue())
			Expect(result2.Metrics["cache_hit"]).To(Equal(float64(1)))
		})

		It("should handle cache expiration", func() {
			// Set very short cache TTL
			controller.planningCache.ttl = 1 * time.Millisecond

			// First planning
			result1, err := controller.planResourcesFromLLM(ctx, networkIntent, sampleLLMResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(result1.Success).To(BeTrue())

			// Wait for cache to expire
			time.Sleep(5 * time.Millisecond)

			// Second planning - should not use cache
			result2, err := controller.planResourcesFromLLM(ctx, networkIntent, sampleLLMResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(result2.Success).To(BeTrue())
			Expect(result2.Metrics["cache_hit"]).To(Equal(float64(0)))
		})

		It("should handle cache size limit", func() {
			// Set small cache size limit
			controller.planningCache.maxEntries = 1

			// Plan first intent
			result1, err := controller.planResourcesFromLLM(ctx, networkIntent, sampleLLMResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(result1.Success).To(BeTrue())

			// Plan second intent with different LLM response
			differentLLMResponse := map[string]interface{}{
				"network_functions": []interface{}{
					map[string]interface{}{
						"name": "upf-deployment",
						"type": "upf",
					},
				},
				"deployment_pattern": "edge",
			}

			result2, err := controller.planResourcesFromLLM(ctx, nil, differentLLMResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(result2.Success).To(BeTrue())

			// Cache should be limited to 1 entry
			Expect(len(controller.planningCache.entries)).To(Equal(1))
		})
	})

	Describe("Dependency and Constraint Calculation", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should calculate 5G core dependencies correctly", func() {
			networkFunctions := []interfaces.PlannedNetworkFunction{
				{Name: "amf", Type: "amf"},
				{Name: "smf", Type: "smf"},
				{Name: "upf", Type: "upf"},
				{Name: "nrf", Type: "nrf"},
				{Name: "ausf", Type: "ausf"},
				{Name: "udm", Type: "udm"},
			}

			dependencies := controller.calculateDependencies(networkFunctions)

			// Verify specific dependencies
			foundAMFtoNRF := false
			foundSMFtoUPF := false

			for _, dep := range dependencies {
				if dep.Metadata["source"] == "amf" && dep.Metadata["target"] == "nrf" {
					foundAMFtoNRF = true
				}
				if dep.Metadata["source"] == "smf" && dep.Metadata["target"] == "upf" {
					foundSMFtoUPF = true
				}
			}

			Expect(foundAMFtoNRF).To(BeTrue())
			Expect(foundSMFtoUPF).To(BeTrue())
		})

		It("should calculate constraints for high-availability deployment", func() {
			networkFunctions := []interfaces.PlannedNetworkFunction{
				{Name: "amf", Type: "amf", Replicas: 3},
				{Name: "smf", Type: "smf", Replicas: 2},
			}

			constraints := controller.calculateConstraints(networkFunctions, "high-availability")

			// Verify anti-affinity constraints for critical NFs
			foundAntiAffinity := false
			foundReplicaMinimum := false

			for _, constraint := range constraints {
				if constraint.Type == "anti-affinity" {
					foundAntiAffinity = true
				}
				if constraint.Type == "replica-minimum" {
					foundReplicaMinimum = true
				}
			}

			Expect(foundAntiAffinity).To(BeTrue())
			Expect(foundReplicaMinimum).To(BeTrue())
		})

		It("should calculate edge deployment constraints", func() {
			networkFunctions := []interfaces.PlannedNetworkFunction{
				{Name: "upf", Type: "upf", Replicas: 1},
			}

			constraints := controller.calculateConstraints(networkFunctions, "edge")

			// Verify resource limits for edge deployment
			foundCPULimit := false
			for _, constraint := range constraints {
				if constraint.Type == "resource-limit" && constraint.Resource == "cpu" {
					foundCPULimit = true
					break
				}
			}

			Expect(foundCPULimit).To(BeTrue())
		})
	})

	Describe("Health and Metrics", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return health status", func() {
			health, err := controller.GetHealthStatus(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(health.Status).To(Equal("Healthy"))
			Expect(health.Metrics).To(HaveKey("totalPlanned"))
			Expect(health.Metrics).To(HaveKey("successRate"))
			Expect(health.Metrics).To(HaveKey("averagePlanningTime"))
		})

		It("should return controller metrics", func() {
			metrics, err := controller.GetMetrics(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(metrics).To(HaveKey("total_planned"))
			Expect(metrics).To(HaveKey("successful_planned"))
			Expect(metrics).To(HaveKey("failed_planned"))
			Expect(metrics).To(HaveKey("success_rate"))
			Expect(metrics).To(HaveKey("average_planning_time_ms"))
		})

		It("should update metrics after planning", func() {
			initialMetrics, err := controller.GetMetrics(ctx)
			Expect(err).NotTo(HaveOccurred())
			initialTotal := initialMetrics["total_planned"]

			// Perform planning
			_, err = controller.planResourcesFromLLM(ctx, networkIntent, sampleLLMResponse)
			Expect(err).NotTo(HaveOccurred())

			updatedMetrics, err := controller.GetMetrics(ctx)
			Expect(err).NotTo(HaveOccurred())
			updatedTotal := updatedMetrics["total_planned"]

			Expect(updatedTotal).To(Equal(initialTotal + 1))
		})

		It("should track per NF type metrics", func() {
			// Plan resources to generate metrics
			_, err := controller.planResourcesFromLLM(ctx, networkIntent, sampleLLMResponse)
			Expect(err).NotTo(HaveOccurred())

			// Verify NF type metrics were created
			controller.metrics.mutex.RLock()
			defer controller.metrics.mutex.RUnlock()

			Expect(controller.metrics.NFTypeMetrics).To(HaveKey("amf-deployment"))
			Expect(controller.metrics.NFTypeMetrics).To(HaveKey("smf-deployment"))
		})
	})

	Describe("Phase Controller Interface", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return correct dependencies", func() {
			deps := controller.GetDependencies()
			Expect(deps).To(ContainElement(interfaces.PhaseLLMProcessing))
		})

		It("should return correct blocked phases", func() {
			blocked := controller.GetBlockedPhases()
			expectedPhases := []interfaces.ProcessingPhase{
				interfaces.PhaseManifestGeneration,
				interfaces.PhaseGitOpsCommit,
				interfaces.PhaseDeploymentVerification,
			}
			Expect(blocked).To(ContainElements(expectedPhases))
		})

		It("should get phase status correctly", func() {
			intentID := networkIntent.Name

			// Create a planning session
			session := &PlanningSession{
				IntentID:    intentID,
				StartTime:   time.Now(),
				Progress:    0.5,
				CurrentStep: "calculating_resources",
			}
			controller.activePlanning.Store(intentID, session)

			status, err := controller.GetPhaseStatus(ctx, intentID)
			Expect(err).NotTo(HaveOccurred())
			Expect(status.Phase).To(Equal(interfaces.PhaseResourcePlanning))
			Expect(status.Status).To(Equal("InProgress"))
			Expect(status.Metrics["progress"]).To(Equal(0.5))
		})

		It("should handle phase errors", func() {
			intentID := networkIntent.Name
			testError := fmt.Errorf("test planning error")

			err := controller.HandlePhaseError(ctx, intentID, testError)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("test planning error"))
		})
	})

	Describe("Error Handling", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle resource calculator failure", func() {
			mockResourceCalculator.SetShouldReturnError(true)

			// This should still succeed as we use default resources when calculator fails
			result, err := controller.planResourcesFromLLM(ctx, networkIntent, sampleLLMResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())
		})

		It("should handle malformed network functions data", func() {
			invalidLLMResponse := map[string]interface{}{
				"network_functions":  "invalid-string-instead-of-array",
				"deployment_pattern": "standard",
			}

			result, err := controller.planResourcesFromLLM(ctx, networkIntent, invalidLLMResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeFalse())
			Expect(result.ErrorCode).To(Equal("PARSING_ERROR"))
		})

		It("should continue processing when optional services fail", func() {
			// Make optimization fail but continue with basic planning
			mockOptimizationEngine.SetShouldReturnError(true)
			mockCostEstimator.SetShouldReturnError(true)

			result, err := controller.planResourcesFromLLM(ctx, networkIntent, sampleLLMResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Success).To(BeTrue())

			// Should not have optimization or cost data
			Expect(result.Data).NotTo(HaveKey("optimizedPlan"))
			Expect(result.Data).NotTo(HaveKey("costEstimate"))
		})
	})

	Describe("Concurrent Processing", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle multiple concurrent planning requests", func() {
			numRequests := 10
			results := make(chan interfaces.ProcessingResult, numRequests)
			errors := make(chan error, numRequests)

			// Create different LLM responses for concurrent testing
			for i := 0; i < numRequests; i++ {
				go func(index int) {
					defer GinkgoRecover()

					llmResponse := map[string]interface{}{
						"network_functions": []interface{}{
							map[string]interface{}{
								"name":     fmt.Sprintf("nf-%d", index),
								"type":     "amf",
								"replicas": 1,
								"resources": map[string]interface{}{
									"requests": map[string]interface{}{
										"cpu":    "500m",
										"memory": "1Gi",
									},
								},
							},
						},
						"deployment_pattern": "standard",
					}

					result, err := controller.planResourcesFromLLM(ctx, nil, llmResponse)
					if err != nil {
						errors <- err
					} else {
						results <- result
					}
				}(i)
			}

			// Collect results
			successCount := 0
			errorCount := 0
			timeout := time.After(30 * time.Second)

			for i := 0; i < numRequests; i++ {
				select {
				case result := <-results:
					if result.Success {
						successCount++
					}
				case <-errors:
					errorCount++
				case <-timeout:
					Fail("Timeout waiting for concurrent planning to complete")
				}
			}

			// Verify all requests were processed
			Expect(successCount + errorCount).To(Equal(numRequests))
			// Most should succeed
			Expect(successCount).To(BeNumerically(">=", numRequests/2))
		})

		It("should maintain thread safety under concurrent load", func() {
			numGoroutines := 25
			done := make(chan bool, numGoroutines)

			for i := 0; i < numGoroutines; i++ {
				go func(index int) {
					defer GinkgoRecover()
					defer func() { done <- true }()

					// Mix of operations to test thread safety
					switch index % 3 {
					case 0:
						// Plan resources
						controller.planResourcesFromLLM(ctx, nil, sampleLLMResponse)
					case 1:
						// Get health status
						controller.GetHealthStatus(ctx)
					case 2:
						// Get metrics
						controller.GetMetrics(ctx)
					}
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < numGoroutines; i++ {
				Eventually(done).Should(Receive())
			}
		})
	})

	Describe("Background Operations", func() {
		BeforeEach(func() {
			err := controller.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should perform session cleanup", func() {
			intentID := "cleanup-test-intent"

			// Create an old session
			oldSession := &PlanningSession{
				IntentID:  intentID,
				StartTime: time.Now().Add(-2 * time.Hour),
			}
			controller.activePlanning.Store(intentID, oldSession)

			// Verify session exists
			_, exists := controller.activePlanning.Load(intentID)
			Expect(exists).To(BeTrue())

			// Trigger cleanup
			controller.cleanupExpiredSessions()

			// Verify session was removed
			_, exists = controller.activePlanning.Load(intentID)
			Expect(exists).To(BeFalse())
		})

		It("should perform cache cleanup", func() {
			// Add expired entry to cache
			expiredHash := "expired-plan-entry"
			controller.planningCache.entries[expiredHash] = &PlanCacheEntry{
				Timestamp: time.Now().Add(-1 * time.Hour),
			}

			// Verify entry exists
			Expect(controller.planningCache.entries).To(HaveKey(expiredHash))

			// Trigger cache cleanup
			controller.cleanupExpiredCache()

			// Verify expired entry was removed
			Expect(controller.planningCache.entries).NotTo(HaveKey(expiredHash))
		})

		It("should perform health monitoring", func() {
			initialHealth := controller.healthStatus

			// Trigger health check
			controller.performHealthCheck()

			// Verify health status was updated
			Expect(controller.healthStatus.LastChecked).To(BeTemporally(">", initialHealth.LastChecked))
		})
	})
})

func TestSpecializedResourcePlanningController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SpecializedResourcePlanningController Suite")
}
