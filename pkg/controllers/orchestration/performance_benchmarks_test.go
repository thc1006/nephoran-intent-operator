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
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
	"github.com/thc1006/nephoran-intent-operator/pkg/testutils"
)

// BenchmarkSuite provides performance testing utilities
type BenchmarkSuite struct {
	// Test infrastructure
	ctx        context.Context
	logger     logr.Logger
	fakeClient client.Client
	scheme     *k8sruntime.Scheme
	recorder   *record.FakeRecorder

	// Controllers
	intentController   *SpecializedIntentProcessingController
	resourceController *SpecializedResourcePlanningController
	manifestController *SpecializedManifestGenerationController

	// Mock services
	mockLLMClient          *testutils.MockLLMClient
	mockRAGService         *MockRAGService
	mockResourceCalculator *MockTelecomResourceCalculator
	mockTemplateEngine     *MockKubernetesTemplateEngine

	// Performance tracking
	metrics        *PerformanceMetrics
	memoryProfiler *MemoryProfiler
}

// PerformanceMetrics tracks performance data
type PerformanceMetrics struct {
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	Operations int64
	Throughput float64 // operations per second

	// Latency statistics
	MinLatency time.Duration
	MaxLatency time.Duration
	AvgLatency time.Duration
	P50Latency time.Duration
	P95Latency time.Duration
	P99Latency time.Duration

	// Memory statistics
	InitialMemory uint64
	PeakMemory    uint64
	FinalMemory   uint64
	MemoryGrowth  uint64

	// Error statistics
	SuccessCount int64
	ErrorCount   int64
	ErrorRate    float64

	// Concurrency
	MaxConcurrency int

	mutex sync.RWMutex
}

// MemoryProfiler tracks memory usage
type MemoryProfiler struct {
	samples []MemorySample
	mutex   sync.RWMutex
}

// MemorySample represents a memory usage sample
type MemorySample struct {
	Timestamp time.Time
	HeapAlloc uint64
	HeapSys   uint64
	NumGC     uint32
}

// LatencyTracker tracks latency measurements
type LatencyTracker struct {
	measurements []time.Duration
	mutex        sync.RWMutex
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite() *BenchmarkSuite {
	ctx := context.Background()
	logger := zap.New(zap.WriteTo(testing.Verbose()), zap.UseDevMode(true))

	// Create scheme and add types
	scheme := k8sruntime.NewScheme()
	corev1.AddToScheme(scheme)
	nephoranv1.AddToScheme(scheme)

	// Create fake client and recorder
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(1000)

	suite := &BenchmarkSuite{
		ctx:            ctx,
		logger:         logger.WithName("benchmark-suite"),
		fakeClient:     fakeClient,
		scheme:         scheme,
		recorder:       recorder,
		metrics:        NewPerformanceMetrics(),
		memoryProfiler: NewMemoryProfiler(),
	}

	suite.initializeMockServices()
	suite.initializeControllers()

	return suite
}

// initializeMockServices creates optimized mock services for benchmarking
func (s *BenchmarkSuite) initializeMockServices() {
	// Configure mocks for high performance
	s.mockLLMClient = NewMockLLMClient()
	s.mockLLMClient.SetResponseDelay(5 * time.Millisecond) // Fast response

	s.mockRAGService = NewMockRAGService()
	s.mockRAGService.SetQueryDelay(2 * time.Millisecond)

	s.mockResourceCalculator = NewMockTelecomResourceCalculator()
	s.mockResourceCalculator.SetCalculationDelay(3 * time.Millisecond)

	s.mockTemplateEngine = NewMockKubernetesTemplateEngine()
	s.mockTemplateEngine.SetProcessingDelay(1 * time.Millisecond)
}

// initializeControllers creates controllers with optimized configurations
func (s *BenchmarkSuite) initializeControllers() {
	// Intent Processing Controller
	intentConfig := IntentProcessingConfig{
		LLMEndpoint:           "http://mock-llm:8080",
		LLMAPIKey:             "benchmark-key",
		LLMModel:              "gpt-4o-mini",
		MaxTokens:             1000,
		Temperature:           0.7,
		RAGEndpoint:           "http://mock-rag:8080",
		MaxContextChunks:      5,
		SimilarityThreshold:   0.7,
		StreamingEnabled:      false, // Disable for benchmarks
		CacheEnabled:          true,
		CacheTTL:              5 * time.Minute,
		MaxRetries:            1, // Reduce retries for benchmark
		Timeout:               5 * time.Second,
		CircuitBreakerEnabled: false, // Disable for benchmarks
	}

	s.intentController = &SpecializedIntentProcessingController{
		Client:               s.fakeClient,
		Scheme:               s.scheme,
		Recorder:             s.recorder,
		Logger:               s.logger.WithName("intent-benchmark"),
		LLMClient:            s.mockLLMClient,
		RAGService:           s.mockRAGService,
		PromptEngine:         NewMockPromptEngine(),
		StreamingProcessor:   NewMockStreamingProcessor(),
		PerformanceOptimizer: NewMockPerformanceOptimizer(),
		Config:               intentConfig,
		SupportedIntents:     []string{"5g-deployment", "network-slice", "cnf-deployment"},
		ConfidenceThreshold:  0.7,
		metrics:              NewIntentProcessingMetrics(),
		stopChan:             make(chan struct{}),
		healthStatus: interfaces.HealthStatus{
			Status:      "Healthy",
			Message:     "Controller ready for benchmarking",
			LastChecked: time.Now(),
		},
	}

	s.intentController.cache = &IntentProcessingCache{
		entries:    make(map[string]*CacheEntry),
		ttl:        intentConfig.CacheTTL,
		maxEntries: 10000, // Large cache for benchmarks
	}

	// Resource Planning Controller
	resourceConfig := ResourcePlanningConfig{
		DefaultCPURequest:      "500m",
		DefaultMemoryRequest:   "1Gi",
		DefaultStorageRequest:  "10Gi",
		CPUOvercommitRatio:     1.2,
		MemoryOvercommitRatio:  1.1,
		OptimizationEnabled:    false, // Disable for benchmark speed
		ConstraintCheckEnabled: false, // Disable for benchmark speed
		MaxPlanningTime:        30 * time.Second,
		ParallelPlanning:       false,
		CacheEnabled:           true,
		CacheTTL:               5 * time.Minute,
		MaxCacheEntries:        10000,
	}

	s.resourceController = &SpecializedResourcePlanningController{
		Client:             s.fakeClient,
		Scheme:             s.scheme,
		Recorder:           s.recorder,
		Logger:             s.logger.WithName("resource-benchmark"),
		ResourceCalculator: s.mockResourceCalculator,
		OptimizationEngine: NewMockResourceOptimizationEngine(),
		ConstraintSolver:   NewMockResourceConstraintSolver(),
		CostEstimator:      NewMockTelecomCostEstimator(),
		Config:             resourceConfig,
		ResourceTemplates:  initializeResourceTemplates(),
		ConstraintRules:    initializeConstraintRules(),
		metrics:            NewResourcePlanningMetrics(),
		stopChan:           make(chan struct{}),
		healthStatus: interfaces.HealthStatus{
			Status:      "Healthy",
			Message:     "Controller ready for benchmarking",
			LastChecked: time.Now(),
		},
	}

	s.resourceController.planningCache = &ResourcePlanCache{
		entries:    make(map[string]*PlanCacheEntry),
		ttl:        resourceConfig.CacheTTL,
		maxEntries: resourceConfig.MaxCacheEntries,
	}

	// Manifest Generation Controller
	manifestConfig := ManifestGenerationConfig{
		TemplateDirectory:   "/templates",
		DefaultNamespace:    "nephoran-system",
		EnableHelm:          false,
		EnableKustomize:     false,
		ValidateManifests:   false, // Disable for benchmark speed
		OptimizeManifests:   false, // Disable for benchmark speed
		EnforcePolicies:     false, // Disable for benchmark speed
		CacheEnabled:        true,
		CacheTTL:            5 * time.Minute,
		MaxCacheEntries:     10000,
		MaxGenerationTime:   30 * time.Second,
		ParallelGeneration:  false,
		ConcurrentTemplates: 1,
	}

	s.manifestController = &SpecializedManifestGenerationController{
		Client:            s.fakeClient,
		Scheme:            s.scheme,
		Recorder:          s.recorder,
		Logger:            s.logger.WithName("manifest-benchmark"),
		TemplateEngine:    s.mockTemplateEngine,
		ManifestValidator: NewMockManifestValidator(),
		PolicyEnforcer:    NewMockManifestPolicyEnforcer(),
		ManifestOptimizer: NewMockManifestOptimizer(),
		Config:            manifestConfig,
		Templates:         initializeManifestTemplates(),
		PolicyRules:       initializeManifestPolicyRules(),
		metrics:           NewManifestGenerationMetrics(),
		stopChan:          make(chan struct{}),
		healthStatus: interfaces.HealthStatus{
			Status:      "Healthy",
			Message:     "Controller ready for benchmarking",
			LastChecked: time.Now(),
		},
	}

	s.manifestController.manifestCache = &ManifestCache{
		entries:    make(map[string]*ManifestCacheEntry),
		ttl:        manifestConfig.CacheTTL,
		maxEntries: manifestConfig.MaxCacheEntries,
	}
}

// StartControllers starts all controllers for benchmarking
func (s *BenchmarkSuite) StartControllers() error {
	if err := s.intentController.Start(s.ctx); err != nil {
		return fmt.Errorf("failed to start intent controller: %w", err)
	}

	if err := s.resourceController.Start(s.ctx); err != nil {
		return fmt.Errorf("failed to start resource controller: %w", err)
	}

	if err := s.manifestController.Start(s.ctx); err != nil {
		return fmt.Errorf("failed to start manifest controller: %w", err)
	}

	return nil
}

// StopControllers stops all controllers
func (s *BenchmarkSuite) StopControllers() error {
	var errors []error

	if err := s.manifestController.Stop(s.ctx); err != nil {
		errors = append(errors, err)
	}

	if err := s.resourceController.Stop(s.ctx); err != nil {
		errors = append(errors, err)
	}

	if err := s.intentController.Stop(s.ctx); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping controllers: %v", errors)
	}

	return nil
}

// NewPerformanceMetrics creates a new performance metrics instance
func NewPerformanceMetrics() *PerformanceMetrics {
	return &PerformanceMetrics{
		MinLatency: time.Hour, // Initialize to high value
	}
}

// NewMemoryProfiler creates a new memory profiler
func NewMemoryProfiler() *MemoryProfiler {
	return &MemoryProfiler{
		samples: make([]MemorySample, 0),
	}
}

// NewLatencyTracker creates a new latency tracker
func NewLatencyTracker() *LatencyTracker {
	return &LatencyTracker{
		measurements: make([]time.Duration, 0),
	}
}

// StartTracking starts performance tracking
func (m *PerformanceMetrics) StartTracking() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.StartTime = time.Now()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	m.InitialMemory = ms.HeapAlloc
}

// RecordOperation records a successful operation
func (m *PerformanceMetrics) RecordOperation(latency time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.Operations++
	m.SuccessCount++

	// Update latency statistics
	if latency < m.MinLatency {
		m.MinLatency = latency
	}
	if latency > m.MaxLatency {
		m.MaxLatency = latency
	}
}

// RecordError records a failed operation
func (m *PerformanceMetrics) RecordError() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.Operations++
	m.ErrorCount++
}

// FinishTracking completes performance tracking and calculates final metrics
func (m *PerformanceMetrics) FinishTracking() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.EndTime = time.Now()
	m.Duration = m.EndTime.Sub(m.StartTime)

	if m.Duration > 0 {
		m.Throughput = float64(m.Operations) / m.Duration.Seconds()
	}

	if m.Operations > 0 {
		m.ErrorRate = float64(m.ErrorCount) / float64(m.Operations)
	}

	// Update memory statistics
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	m.FinalMemory = ms.HeapAlloc
	if m.FinalMemory > m.InitialMemory {
		m.MemoryGrowth = m.FinalMemory - m.InitialMemory
	}
}

// TakeSample takes a memory usage sample
func (mp *MemoryProfiler) TakeSample() {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	sample := MemorySample{
		Timestamp: time.Now(),
		HeapAlloc: ms.HeapAlloc,
		HeapSys:   ms.HeapSys,
		NumGC:     ms.NumGC,
	}

	mp.samples = append(mp.samples, sample)
}

// GetPeakMemory returns the peak memory usage
func (mp *MemoryProfiler) GetPeakMemory() uint64 {
	mp.mutex.RLock()
	defer mp.mutex.RUnlock()

	var peak uint64
	for _, sample := range mp.samples {
		if sample.HeapAlloc > peak {
			peak = sample.HeapAlloc
		}
	}
	return peak
}

// StartProfiling starts continuous memory profiling
func (mp *MemoryProfiler) StartProfiling(interval time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mp.TakeSample()
		case <-stop:
			return
		}
	}
}

// AddMeasurement adds a latency measurement
func (lt *LatencyTracker) AddMeasurement(latency time.Duration) {
	lt.mutex.Lock()
	defer lt.mutex.Unlock()

	lt.measurements = append(lt.measurements, latency)
}

// GetPercentile calculates the specified percentile
func (lt *LatencyTracker) GetPercentile(percentile float64) time.Duration {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()

	if len(lt.measurements) == 0 {
		return 0
	}

	// Make a copy and sort
	measurements := make([]time.Duration, len(lt.measurements))
	copy(measurements, lt.measurements)

	// Simple bubble sort for demo (use proper sort in production)
	for i := 0; i < len(measurements)-1; i++ {
		for j := 0; j < len(measurements)-i-1; j++ {
			if measurements[j] > measurements[j+1] {
				measurements[j], measurements[j+1] = measurements[j+1], measurements[j]
			}
		}
	}

	index := int(float64(len(measurements)) * percentile / 100.0)
	if index >= len(measurements) {
		index = len(measurements) - 1
	}

	return measurements[index]
}

// Benchmark functions

// BenchmarkIntentProcessingController benchmarks intent processing performance
func BenchmarkIntentProcessingController(b *testing.B) {
	suite := NewBenchmarkSuite()
	err := suite.StartControllers()
	if err != nil {
		b.Fatalf("Failed to start controllers: %v", err)
	}
	defer suite.StopControllers()

	// Create test intent
	intent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-intent",
			Namespace: "test-namespace",
			UID:       "benchmark-uid",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent:     "Deploy AMF for 5G core network",
			IntentType: "5g-deployment",
		},
	}

	b.ResetTimer()
	suite.metrics.StartTracking()

	// Start memory profiling
	stopProfiler := make(chan struct{})
	go suite.memoryProfiler.StartProfiling(100*time.Millisecond, stopProfiler)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			start := time.Now()

			// Create unique intent for each iteration
			testIntent := intent.DeepCopy()
			testIntent.Name = fmt.Sprintf("benchmark-intent-%d", b.N)
			testIntent.UID = nephoranv1.UID(fmt.Sprintf("benchmark-uid-%d", b.N))

			_, err := suite.intentController.ProcessIntent(suite.ctx, testIntent)
			latency := time.Since(start)

			if err != nil {
				suite.metrics.RecordError()
			} else {
				suite.metrics.RecordOperation(latency)
			}
		}
	})

	close(stopProfiler)
	suite.metrics.FinishTracking()
	suite.metrics.PeakMemory = suite.memoryProfiler.GetPeakMemory()

	// Report results
	b.ReportMetric(suite.metrics.Throughput, "ops/sec")
	b.ReportMetric(float64(suite.metrics.MinLatency.Microseconds()), "min-latency-μs")
	b.ReportMetric(float64(suite.metrics.MaxLatency.Microseconds()), "max-latency-μs")
	b.ReportMetric(float64(suite.metrics.PeakMemory)/1024/1024, "peak-memory-MB")
	b.ReportMetric(suite.metrics.ErrorRate*100, "error-rate-%")
}

// BenchmarkResourcePlanningController benchmarks resource planning performance
func BenchmarkResourcePlanningController(b *testing.B) {
	suite := NewBenchmarkSuite()
	err := suite.StartControllers()
	if err != nil {
		b.Fatalf("Failed to start controllers: %v", err)
	}
	defer suite.StopControllers()

	// Create sample LLM response
	sampleLLMResponse := map[string]interface{}{
		"network_functions": []interface{}{
			map[string]interface{}{
				"name":     "amf",
				"type":     "amf",
				"replicas": 2,
				"resources": map[string]interface{}{
					"requests": map[string]interface{}{
						"cpu":    "500m",
						"memory": "1Gi",
					},
				},
			},
		},
		"deployment_pattern": "standard",
		"confidence":         0.9,
	}

	b.ResetTimer()
	suite.metrics.StartTracking()

	// Start memory profiling
	stopProfiler := make(chan struct{})
	go suite.memoryProfiler.StartProfiling(100*time.Millisecond, stopProfiler)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			start := time.Now()

			// Create unique intent for each iteration
			intent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("resource-benchmark-%d", b.N),
					Namespace: "test-namespace",
					UID:       nephoranv1.UID(fmt.Sprintf("resource-uid-%d", b.N)),
				},
				Status: nephoranv1.NetworkIntentStatus{
					ProcessingPhase: interfaces.PhaseResourcePlanning,
					LLMResponse:     sampleLLMResponse,
				},
			}

			_, err := suite.resourceController.ProcessPhase(suite.ctx, intent, interfaces.PhaseResourcePlanning)
			latency := time.Since(start)

			if err != nil {
				suite.metrics.RecordError()
			} else {
				suite.metrics.RecordOperation(latency)
			}
		}
	})

	close(stopProfiler)
	suite.metrics.FinishTracking()
	suite.metrics.PeakMemory = suite.memoryProfiler.GetPeakMemory()

	// Report results
	b.ReportMetric(suite.metrics.Throughput, "ops/sec")
	b.ReportMetric(float64(suite.metrics.MinLatency.Microseconds()), "min-latency-μs")
	b.ReportMetric(float64(suite.metrics.MaxLatency.Microseconds()), "max-latency-μs")
	b.ReportMetric(float64(suite.metrics.PeakMemory)/1024/1024, "peak-memory-MB")
	b.ReportMetric(suite.metrics.ErrorRate*100, "error-rate-%")
}

// BenchmarkManifestGenerationController benchmarks manifest generation performance
func BenchmarkManifestGenerationController(b *testing.B) {
	suite := NewBenchmarkSuite()
	err := suite.StartControllers()
	if err != nil {
		b.Fatalf("Failed to start controllers: %v", err)
	}
	defer suite.StopControllers()

	// Create sample resource plan
	sampleResourcePlan := &interfaces.ResourcePlan{
		NetworkFunctions: []interfaces.PlannedNetworkFunction{
			{
				Name:     "amf-deployment",
				Type:     "amf",
				Image:    "nephoran/amf",
				Version:  "v1.0.0",
				Replicas: 2,
				Resources: interfaces.ResourceSpec{
					Requests: interfaces.ResourceRequirements{
						CPU:     "500m",
						Memory:  "1Gi",
						Storage: "10Gi",
					},
				},
			},
		},
		DeploymentPattern: "standard",
	}

	b.ResetTimer()
	suite.metrics.StartTracking()

	// Start memory profiling
	stopProfiler := make(chan struct{})
	go suite.memoryProfiler.StartProfiling(100*time.Millisecond, stopProfiler)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			start := time.Now()

			_, err := suite.manifestController.GenerateManifests(suite.ctx, sampleResourcePlan)
			latency := time.Since(start)

			if err != nil {
				suite.metrics.RecordError()
			} else {
				suite.metrics.RecordOperation(latency)
			}
		}
	})

	close(stopProfiler)
	suite.metrics.FinishTracking()
	suite.metrics.PeakMemory = suite.memoryProfiler.GetPeakMemory()

	// Report results
	b.ReportMetric(suite.metrics.Throughput, "ops/sec")
	b.ReportMetric(float64(suite.metrics.MinLatency.Microseconds()), "min-latency-μs")
	b.ReportMetric(float64(suite.metrics.MaxLatency.Microseconds()), "max-latency-μs")
	b.ReportMetric(float64(suite.metrics.PeakMemory)/1024/1024, "peak-memory-MB")
	b.ReportMetric(suite.metrics.ErrorRate*100, "error-rate-%")
}

// BenchmarkEndToEndWorkflow benchmarks the complete workflow
func BenchmarkEndToEndWorkflow(b *testing.B) {
	suite := NewBenchmarkSuite()
	err := suite.StartControllers()
	if err != nil {
		b.Fatalf("Failed to start controllers: %v", err)
	}
	defer suite.StopControllers()

	// Create test intent
	baseIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-benchmark",
			Namespace: "test-namespace",
			UID:       "e2e-benchmark-uid",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent:     "Deploy AMF and SMF for 5G core",
			IntentType: "5g-deployment",
		},
	}

	b.ResetTimer()
	suite.metrics.StartTracking()

	// Start memory profiling
	stopProfiler := make(chan struct{})
	go suite.memoryProfiler.StartProfiling(100*time.Millisecond, stopProfiler)

	latencyTracker := NewLatencyTracker()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		// Create unique intent for each iteration
		intent := baseIntent.DeepCopy()
		intent.Name = fmt.Sprintf("e2e-benchmark-%d", i)
		intent.UID = nephoranv1.UID(fmt.Sprintf("e2e-uid-%d", i))

		// Step 1: Intent Processing
		intent.Status.ProcessingPhase = interfaces.PhaseLLMProcessing
		llmResult, err := suite.intentController.ProcessIntent(suite.ctx, intent)
		if err != nil {
			suite.metrics.RecordError()
			continue
		}

		// Step 2: Resource Planning
		intent.Status.LLMResponse = llmResult.Data
		intent.Status.ProcessingPhase = interfaces.PhaseResourcePlanning
		resourceResult, err := suite.resourceController.ProcessPhase(suite.ctx, intent, interfaces.PhaseResourcePlanning)
		if err != nil {
			suite.metrics.RecordError()
			continue
		}

		// Step 3: Manifest Generation
		intent.Status.ResourcePlan = resourceResult.Data
		intent.Status.ProcessingPhase = interfaces.PhaseManifestGeneration
		_, err = suite.manifestController.ProcessPhase(suite.ctx, intent, interfaces.PhaseManifestGeneration)
		if err != nil {
			suite.metrics.RecordError()
			continue
		}

		latency := time.Since(start)
		latencyTracker.AddMeasurement(latency)
		suite.metrics.RecordOperation(latency)
	}

	close(stopProfiler)
	suite.metrics.FinishTracking()
	suite.metrics.PeakMemory = suite.memoryProfiler.GetPeakMemory()
	suite.metrics.P50Latency = latencyTracker.GetPercentile(50)
	suite.metrics.P95Latency = latencyTracker.GetPercentile(95)
	suite.metrics.P99Latency = latencyTracker.GetPercentile(99)

	// Report detailed results
	b.ReportMetric(suite.metrics.Throughput, "ops/sec")
	b.ReportMetric(float64(suite.metrics.MinLatency.Microseconds()), "min-latency-μs")
	b.ReportMetric(float64(suite.metrics.MaxLatency.Microseconds()), "max-latency-μs")
	b.ReportMetric(float64(suite.metrics.P50Latency.Microseconds()), "p50-latency-μs")
	b.ReportMetric(float64(suite.metrics.P95Latency.Microseconds()), "p95-latency-μs")
	b.ReportMetric(float64(suite.metrics.P99Latency.Microseconds()), "p99-latency-μs")
	b.ReportMetric(float64(suite.metrics.PeakMemory)/1024/1024, "peak-memory-MB")
	b.ReportMetric(suite.metrics.ErrorRate*100, "error-rate-%")

	b.Logf("End-to-end benchmark results:")
	b.Logf("  Total operations: %d", suite.metrics.Operations)
	b.Logf("  Duration: %v", suite.metrics.Duration)
	b.Logf("  Throughput: %.2f ops/sec", suite.metrics.Throughput)
	b.Logf("  Min latency: %v", suite.metrics.MinLatency)
	b.Logf("  Max latency: %v", suite.metrics.MaxLatency)
	b.Logf("  P50 latency: %v", suite.metrics.P50Latency)
	b.Logf("  P95 latency: %v", suite.metrics.P95Latency)
	b.Logf("  P99 latency: %v", suite.metrics.P99Latency)
	b.Logf("  Peak memory: %.2f MB", float64(suite.metrics.PeakMemory)/1024/1024)
	b.Logf("  Error rate: %.2f%%", suite.metrics.ErrorRate*100)
}

// BenchmarkConcurrentIntentProcessing benchmarks concurrent processing
func BenchmarkConcurrentIntentProcessing(b *testing.B) {
	suite := NewBenchmarkSuite()
	err := suite.StartControllers()
	if err != nil {
		b.Fatalf("Failed to start controllers: %v", err)
	}
	defer suite.StopControllers()

	concurrencyLevels := []int{1, 2, 5, 10, 20, 50}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			suite.metrics = NewPerformanceMetrics() // Reset metrics
			suite.metrics.MaxConcurrency = concurrency

			b.ResetTimer()
			suite.metrics.StartTracking()

			// Start memory profiling
			stopProfiler := make(chan struct{})
			go suite.memoryProfiler.StartProfiling(100*time.Millisecond, stopProfiler)

			b.SetParallelism(concurrency)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					start := time.Now()

					intent := &nephoranv1.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("concurrent-intent-%d", b.N),
							Namespace: "test-namespace",
							UID:       nephoranv1.UID(fmt.Sprintf("concurrent-uid-%d", b.N)),
						},
						Spec: nephoranv1.NetworkIntentSpec{
							Intent:     "Deploy AMF for concurrent test",
							IntentType: "5g-deployment",
						},
					}

					_, err := suite.intentController.ProcessIntent(suite.ctx, intent)
					latency := time.Since(start)

					if err != nil {
						suite.metrics.RecordError()
					} else {
						suite.metrics.RecordOperation(latency)
					}
				}
			})

			close(stopProfiler)
			suite.metrics.FinishTracking()
			suite.metrics.PeakMemory = suite.memoryProfiler.GetPeakMemory()

			// Report results for this concurrency level
			b.ReportMetric(suite.metrics.Throughput, "ops/sec")
			b.ReportMetric(float64(suite.metrics.PeakMemory)/1024/1024, "peak-memory-MB")
			b.ReportMetric(suite.metrics.ErrorRate*100, "error-rate-%")

			b.Logf("Concurrency %d: %.2f ops/sec, %.2f MB peak memory, %.2f%% errors",
				concurrency,
				suite.metrics.Throughput,
				float64(suite.metrics.PeakMemory)/1024/1024,
				suite.metrics.ErrorRate*100)
		})
	}
}

// BenchmarkCachePerformance benchmarks caching effectiveness
func BenchmarkCachePerformance(b *testing.B) {
	suite := NewBenchmarkSuite()
	err := suite.StartControllers()
	if err != nil {
		b.Fatalf("Failed to start controllers: %v", err)
	}
	defer suite.StopControllers()

	// Create test intent that will be reused to trigger cache hits
	intent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cache-test-intent",
			Namespace: "test-namespace",
			UID:       "cache-test-uid",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent:     "Deploy AMF for cache test", // Same intent to trigger cache
			IntentType: "5g-deployment",
		},
	}

	b.Run("WithCache", func(b *testing.B) {
		suite.intentController.Config.CacheEnabled = true
		suite.metrics = NewPerformanceMetrics()

		b.ResetTimer()
		suite.metrics.StartTracking()

		for i := 0; i < b.N; i++ {
			start := time.Now()

			testIntent := intent.DeepCopy()
			testIntent.Name = fmt.Sprintf("cache-intent-%d", i)

			_, err := suite.intentController.ProcessIntent(suite.ctx, testIntent)
			latency := time.Since(start)

			if err != nil {
				suite.metrics.RecordError()
			} else {
				suite.metrics.RecordOperation(latency)
			}
		}

		suite.metrics.FinishTracking()
		b.ReportMetric(suite.metrics.Throughput, "ops/sec")
	})

	b.Run("WithoutCache", func(b *testing.B) {
		// Disable cache and create new controller
		suite.intentController.Config.CacheEnabled = false
		suite.intentController.cache = nil
		suite.metrics = NewPerformanceMetrics()

		b.ResetTimer()
		suite.metrics.StartTracking()

		for i := 0; i < b.N; i++ {
			start := time.Now()

			testIntent := intent.DeepCopy()
			testIntent.Name = fmt.Sprintf("no-cache-intent-%d", i)

			_, err := suite.intentController.ProcessIntent(suite.ctx, testIntent)
			latency := time.Since(start)

			if err != nil {
				suite.metrics.RecordError()
			} else {
				suite.metrics.RecordOperation(latency)
			}
		}

		suite.metrics.FinishTracking()
		b.ReportMetric(suite.metrics.Throughput, "ops/sec")
	})
}

// BenchmarkMemoryUsage specifically focuses on memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	suite := NewBenchmarkSuite()
	err := suite.StartControllers()
	if err != nil {
		b.Fatalf("Failed to start controllers: %v", err)
	}
	defer suite.StopControllers()

	// Force garbage collection before benchmark
	runtime.GC()

	var startMemStats, endMemStats runtime.MemStats
	runtime.ReadMemStats(&startMemStats)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		intent := &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("memory-test-%d", i),
				Namespace: "test-namespace",
				UID:       nephoranv1.UID(fmt.Sprintf("memory-uid-%d", i)),
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent:     "Deploy AMF for memory test",
				IntentType: "5g-deployment",
			},
		}

		_, err := suite.intentController.ProcessIntent(suite.ctx, intent)
		if err != nil {
			b.Errorf("Intent processing failed: %v", err)
		}

		// Periodic garbage collection
		if i%100 == 0 {
			runtime.GC()
		}
	}

	runtime.GC() // Final garbage collection
	runtime.ReadMemStats(&endMemStats)

	// Report memory metrics
	memoryGrowth := endMemStats.HeapAlloc - startMemStats.HeapAlloc
	b.ReportMetric(float64(startMemStats.HeapAlloc)/1024/1024, "start-memory-MB")
	b.ReportMetric(float64(endMemStats.HeapAlloc)/1024/1024, "end-memory-MB")
	b.ReportMetric(float64(memoryGrowth)/1024/1024, "memory-growth-MB")
	b.ReportMetric(float64(endMemStats.NumGC-startMemStats.NumGC), "gc-cycles")

	b.Logf("Memory usage: start=%.2fMB, end=%.2fMB, growth=%.2fMB, GC cycles=%d",
		float64(startMemStats.HeapAlloc)/1024/1024,
		float64(endMemStats.HeapAlloc)/1024/1024,
		float64(memoryGrowth)/1024/1024,
		endMemStats.NumGC-startMemStats.NumGC)
}

// BenchmarkLatencyDistribution analyzes latency distribution patterns
func BenchmarkLatencyDistribution(b *testing.B) {
	suite := NewBenchmarkSuite()
	err := suite.StartControllers()
	if err != nil {
		b.Fatalf("Failed to start controllers: %v", err)
	}
	defer suite.StopControllers()

	latencyTracker := NewLatencyTracker()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		intent := &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("latency-test-%d", i),
				Namespace: "test-namespace",
				UID:       nephoranv1.UID(fmt.Sprintf("latency-uid-%d", i)),
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent:     "Deploy AMF for latency test",
				IntentType: "5g-deployment",
			},
		}

		_, err := suite.intentController.ProcessIntent(suite.ctx, intent)
		latency := time.Since(start)

		if err == nil {
			latencyTracker.AddMeasurement(latency)
		}
	}

	// Calculate and report latency percentiles
	p10 := latencyTracker.GetPercentile(10)
	p25 := latencyTracker.GetPercentile(25)
	p50 := latencyTracker.GetPercentile(50)
	p75 := latencyTracker.GetPercentile(75)
	p90 := latencyTracker.GetPercentile(90)
	p95 := latencyTracker.GetPercentile(95)
	p99 := latencyTracker.GetPercentile(99)

	b.ReportMetric(float64(p10.Microseconds()), "p10-latency-μs")
	b.ReportMetric(float64(p25.Microseconds()), "p25-latency-μs")
	b.ReportMetric(float64(p50.Microseconds()), "p50-latency-μs")
	b.ReportMetric(float64(p75.Microseconds()), "p75-latency-μs")
	b.ReportMetric(float64(p90.Microseconds()), "p90-latency-μs")
	b.ReportMetric(float64(p95.Microseconds()), "p95-latency-μs")
	b.ReportMetric(float64(p99.Microseconds()), "p99-latency-μs")

	b.Logf("Latency distribution:")
	b.Logf("  P10: %v", p10)
	b.Logf("  P25: %v", p25)
	b.Logf("  P50: %v", p50)
	b.Logf("  P75: %v", p75)
	b.Logf("  P90: %v", p90)
	b.Logf("  P95: %v", p95)
	b.Logf("  P99: %v", p99)
}
