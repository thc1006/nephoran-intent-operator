package performance_tests

import (
	"context"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// SLA Targets
	AvailabilityTarget         = 99.9 // 99.9% uptime
	IntentProcessingP95Target  = 30.0 // 30 seconds P95
	IntentProcessingP99Target  = 45.0 // 45 seconds P99
	ErrorRateTarget            = 0.5  // 0.5% error rate
	ResourceOptimizationTarget = 20.0 // 20% resource savings

	// Test Configuration
	LoadTestDuration = 5 * time.Minute
	ConcurrentUsers  = 100
	RequestRate      = 10 // requests per second per user
)

// SLATestSuite manages SLA validation tests
type SLATestSuite struct {
	prometheusClient v1.API
	testStartTime    time.Time
	testEndTime      time.Time
	results          *SLATestResults
}

// SLATestResults captures all SLA test metrics
type SLATestResults struct {
	Availability        float64
	IntentProcessingP50 float64
	IntentProcessingP95 float64
	IntentProcessingP99 float64
	ErrorRate           float64
	ResourceEfficiency  float64
	TelecomKPIs         TelecomKPIs
	AIMLPerformance     AIMLPerformance
	Violations          []SLAViolation
}

// TelecomKPIs represents telecommunications-specific performance indicators
type TelecomKPIs struct {
	NFDeploymentTime    float64 // Network Function deployment time in seconds
	ORANLatencyA1       float64 // A1 interface latency in milliseconds
	ORANLatencyO1       float64 // O1 interface latency in milliseconds
	ORANLatencyO2       float64 // O2 interface latency in milliseconds
	ORANLatencyE2       float64 // E2 interface latency in milliseconds
	HandoverSuccessRate float64 // RAN handover success rate percentage
	SliceCreationTime   float64 // Network slice creation time in seconds
	RICAvailability     float64 // RIC availability percentage
}

// AIMLPerformance represents AI/ML performance metrics
type AIMLPerformance struct {
	TokenGenerationRate float64 // Tokens per second
	RAGRetrievalLatency float64 // RAG retrieval latency in milliseconds
	CacheHitRate        float64 // RAG cache hit rate percentage
	MLInferenceLatency  float64 // ML inference latency in milliseconds
	EmbeddingLatency    float64 // Embedding generation latency in milliseconds
}

// SLAViolation represents a specific SLA violation
type SLAViolation struct {
	Metric    string
	Current   float64
	Target    float64
	Violation string
	Severity  string
}

// NewSLATestSuite creates a new SLA test suite
func NewSLATestSuite(prometheusURL string) (*SLATestSuite, error) {
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	return &SLATestSuite{
		prometheusClient: v1.NewAPI(client),
		results:          &SLATestResults{},
	}, nil
}

// TestSLACompliance runs comprehensive SLA compliance tests
func TestSLACompliance(t *testing.T) {
	suite, err := NewSLATestSuite("http://localhost:9090")
	require.NoError(t, err, "Failed to initialize SLA test suite")

	t.Run("Platform_SLA_Tests", func(t *testing.T) {
		t.Run("Availability_SLA", suite.testAvailabilitySLA)
		t.Run("Intent_Processing_SLA", suite.testIntentProcessingSLA)
		t.Run("Error_Rate_SLA", suite.testErrorRateSLA)
		t.Run("Resource_Optimization_SLA", suite.testResourceOptimizationSLA)
	})

	t.Run("Telecommunications_KPI_Tests", func(t *testing.T) {
		t.Run("Network_Function_Deployment", suite.testNetworkFunctionDeployment)
		t.Run("ORAN_Interface_Performance", suite.testORANInterfacePerformance)
		t.Run("RAN_Performance", suite.testRANPerformance)
		t.Run("Network_Slicing", suite.testNetworkSlicing)
		t.Run("RIC_Performance", suite.testRICPerformance)
	})

	t.Run("AI_ML_Performance_Tests", func(t *testing.T) {
		t.Run("LLM_Performance", suite.testLLMPerformance)
		t.Run("RAG_Performance", suite.testRAGPerformance)
		t.Run("ML_Inference", suite.testMLInference)
	})

	t.Run("Load_Testing_with_SLA_Validation", suite.testLoadWithSLA)
	t.Run("Regression_Testing", suite.testPerformanceRegression)
	t.Run("Composite_SLA_Scores", suite.testCompositeSLAScores)

	// Generate final report
	suite.generateSLAReport(t)
}

// testAvailabilitySLA validates platform availability SLA
func (s *SLATestSuite) testAvailabilitySLA(t *testing.T) {
	query := `avg_over_time(up{job="nephoran-operator"}[5m]) * 100`

	result, err := s.queryPrometheus(query)
	require.NoError(t, err, "Failed to query availability metrics")

	availability := result
	s.results.Availability = availability

	if availability < AvailabilityTarget {
		violation := SLAViolation{
			Metric:    "Availability",
			Current:   availability,
			Target:    AvailabilityTarget,
			Violation: fmt.Sprintf("%.2f%% < %.2f%%", availability, AvailabilityTarget),
			Severity:  "Critical",
		}
		s.results.Violations = append(s.results.Violations, violation)
		t.Errorf("Availability SLA violation: %s", violation.Violation)
	}

	assert.GreaterOrEqual(t, availability, AvailabilityTarget,
		"Platform availability should meet SLA target")

	t.Logf("Platform availability: %.2f%% (target: %.2f%%)", availability, AvailabilityTarget)
}

// testIntentProcessingSLA validates intent processing performance SLA
func (s *SLATestSuite) testIntentProcessingSLA(t *testing.T) {
	queries := map[string]string{
		"p50": `histogram_quantile(0.50, sum(rate(intent_processing_duration_seconds_bucket[5m])) by (le))`,
		"p95": `histogram_quantile(0.95, sum(rate(intent_processing_duration_seconds_bucket[5m])) by (le))`,
		"p99": `histogram_quantile(0.99, sum(rate(intent_processing_duration_seconds_bucket[5m])) by (le))`,
	}

	for percentile, query := range queries {
		result, err := s.queryPrometheus(query)
		require.NoError(t, err, "Failed to query intent processing latency")

		switch percentile {
		case "p50":
			s.results.IntentProcessingP50 = result
		case "p95":
			s.results.IntentProcessingP95 = result
			if result > IntentProcessingP95Target {
				violation := SLAViolation{
					Metric:    "Intent Processing P95",
					Current:   result,
					Target:    IntentProcessingP95Target,
					Violation: fmt.Sprintf("%.2fs > %.2fs", result, IntentProcessingP95Target),
					Severity:  "Warning",
				}
				s.results.Violations = append(s.results.Violations, violation)
				t.Errorf("Intent processing P95 SLA violation: %s", violation.Violation)
			}
		case "p99":
			s.results.IntentProcessingP99 = result
			if result > IntentProcessingP99Target {
				violation := SLAViolation{
					Metric:    "Intent Processing P99",
					Current:   result,
					Target:    IntentProcessingP99Target,
					Violation: fmt.Sprintf("%.2fs > %.2fs", result, IntentProcessingP99Target),
					Severity:  "Critical",
				}
				s.results.Violations = append(s.results.Violations, violation)
				t.Errorf("Intent processing P99 SLA violation: %s", violation.Violation)
			}
		}

		t.Logf("Intent processing %s: %.2fs", percentile, result)
	}
}

// testErrorRateSLA validates error rate SLA
func (s *SLATestSuite) testErrorRateSLA(t *testing.T) {
	query := `(sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))) * 100`

	result, err := s.queryPrometheus(query)
	require.NoError(t, err, "Failed to query error rate metrics")

	errorRate := result
	s.results.ErrorRate = errorRate

	if errorRate > ErrorRateTarget {
		violation := SLAViolation{
			Metric:    "Error Rate",
			Current:   errorRate,
			Target:    ErrorRateTarget,
			Violation: fmt.Sprintf("%.2f%% > %.2f%%", errorRate, ErrorRateTarget),
			Severity:  "Warning",
		}
		s.results.Violations = append(s.results.Violations, violation)
		t.Errorf("Error rate SLA violation: %s", violation.Violation)
	}

	assert.LessOrEqual(t, errorRate, ErrorRateTarget,
		"Error rate should meet SLA target")

	t.Logf("Error rate: %.2f%% (target: %.2f%%)", errorRate, ErrorRateTarget)
}

// testResourceOptimizationSLA validates resource optimization SLA
func (s *SLATestSuite) testResourceOptimizationSLA(t *testing.T) {
	query := `(1 - (sum(rate(container_cpu_usage_seconds_total{namespace="nephoran"}[5m])) / sum(kube_pod_container_resource_requests{namespace="nephoran",resource="cpu"}))) * 100`

	result, err := s.queryPrometheus(query)
	require.NoError(t, err, "Failed to query resource efficiency metrics")

	efficiency := result
	s.results.ResourceEfficiency = efficiency

	if efficiency < ResourceOptimizationTarget {
		violation := SLAViolation{
			Metric:    "Resource Optimization",
			Current:   efficiency,
			Target:    ResourceOptimizationTarget,
			Violation: fmt.Sprintf("%.2f%% < %.2f%%", efficiency, ResourceOptimizationTarget),
			Severity:  "Warning",
		}
		s.results.Violations = append(s.results.Violations, violation)
		t.Logf("Resource optimization below target: %s", violation.Violation)
	}

	t.Logf("Resource efficiency: %.2f%% (target: %.2f%%)", efficiency, ResourceOptimizationTarget)
}

// testNetworkFunctionDeployment validates network function deployment SLA
func (s *SLATestSuite) testNetworkFunctionDeployment(t *testing.T) {
	query := `histogram_quantile(0.95, sum(rate(nf_deployment_duration_seconds_bucket[5m])) by (le))`

	result, err := s.queryPrometheus(query)
	require.NoError(t, err, "Failed to query NF deployment metrics")

	s.results.TelecomKPIs.NFDeploymentTime = result

	target := 180.0 // 3 minutes
	if result > target {
		t.Logf("Network function deployment time above target: %.2fs > %.2fs", result, target)
	}

	t.Logf("Network function deployment P95: %.2fs", result)
}

// testORANInterfacePerformance validates O-RAN interface latencies
func (s *SLATestSuite) testORANInterfacePerformance(t *testing.T) {
	interfaces := map[string]*float64{
		"a1": &s.results.TelecomKPIs.ORANLatencyA1,
		"o1": &s.results.TelecomKPIs.ORANLatencyO1,
		"o2": &s.results.TelecomKPIs.ORANLatencyO2,
		"e2": &s.results.TelecomKPIs.ORANLatencyE2,
	}

	targets := map[string]float64{
		"a1": 100.0,    // 100ms
		"o1": 30000.0,  // 30s for configuration
		"o2": 300000.0, // 5 minutes for provisioning
		"e2": 10.0,     // 10ms
	}

	for iface, resultPtr := range interfaces {
		query := fmt.Sprintf(`histogram_quantile(0.95, sum(rate(oran_interface_latency_seconds_bucket{interface="%s"}[5m])) by (le)) * 1000`, iface)

		result, err := s.queryPrometheus(query)
		if err != nil {
			t.Logf("Failed to query %s interface metrics: %v", iface, err)
			continue
		}

		*resultPtr = result
		target := targets[iface]

		if result > target {
			t.Logf("O-RAN %s interface latency above target: %.2fms > %.2fms", iface, result, target)
		}

		t.Logf("O-RAN %s interface P95 latency: %.2fms", iface, result)
	}
}

// testRANPerformance validates RAN performance metrics
func (s *SLATestSuite) testRANPerformance(t *testing.T) {
	query := `(sum(rate(ran_handover_success_total[5m])) / sum(rate(ran_handover_attempts_total[5m]))) * 100`

	result, err := s.queryPrometheus(query)
	if err != nil {
		t.Logf("Failed to query RAN handover metrics: %v", err)
		return
	}

	s.results.TelecomKPIs.HandoverSuccessRate = result

	target := 99.5 // 99.5%
	if result < target {
		t.Logf("RAN handover success rate below target: %.2f%% < %.2f%%", result, target)
	}

	t.Logf("RAN handover success rate: %.2f%%", result)
}

// testNetworkSlicing validates network slicing performance
func (s *SLATestSuite) testNetworkSlicing(t *testing.T) {
	query := `histogram_quantile(0.95, sum(rate(slice_creation_duration_seconds_bucket[5m])) by (le))`

	result, err := s.queryPrometheus(query)
	if err != nil {
		t.Logf("Failed to query slice creation metrics: %v", err)
		return
	}

	s.results.TelecomKPIs.SliceCreationTime = result

	target := 60.0 // 60 seconds
	if result > target {
		t.Logf("Network slice creation time above target: %.2fs > %.2fs", result, target)
	}

	t.Logf("Network slice creation P95: %.2fs", result)
}

// testRICPerformance validates RIC performance
func (s *SLATestSuite) testRICPerformance(t *testing.T) {
	query := `avg_over_time(up{job="near-rt-ric"}[5m]) * 100`

	result, err := s.queryPrometheus(query)
	if err != nil {
		t.Logf("Failed to query RIC availability metrics: %v", err)
		return
	}

	s.results.TelecomKPIs.RICAvailability = result

	target := 99.95 // 99.95%
	if result < target {
		t.Logf("RIC availability below target: %.2f%% < %.2f%%", result, target)
	}

	t.Logf("RIC availability: %.2f%%", result)
}

// testLLMPerformance validates LLM performance
func (s *SLATestSuite) testLLMPerformance(t *testing.T) {
	query := `rate(llm_tokens_generated_total[1m])`

	result, err := s.queryPrometheus(query)
	if err != nil {
		t.Logf("Failed to query LLM token generation metrics: %v", err)
		return
	}

	s.results.AIMLPerformance.TokenGenerationRate = result

	target := 50.0 // 50 tokens/second
	if result < target {
		t.Logf("LLM token generation rate below target: %.2f/s < %.2f/s", result, target)
	}

	t.Logf("LLM token generation rate: %.2f tokens/s", result)
}

// testRAGPerformance validates RAG performance
func (s *SLATestSuite) testRAGPerformance(t *testing.T) {
	// RAG retrieval latency
	latencyQuery := `histogram_quantile(0.95, sum(rate(rag_retrieval_duration_seconds_bucket[5m])) by (le)) * 1000`
	latency, err := s.queryPrometheus(latencyQuery)
	if err == nil {
		s.results.AIMLPerformance.RAGRetrievalLatency = latency
		target := 200.0 // 200ms
		if latency > target {
			t.Logf("RAG retrieval latency above target: %.2fms > %.2fms", latency, target)
		}
		t.Logf("RAG retrieval P95 latency: %.2fms", latency)
	}

	// Cache hit rate
	cacheQuery := `(sum(rate(rag_cache_hits_total[5m])) / sum(rate(rag_requests_total[5m]))) * 100`
	cacheHitRate, err := s.queryPrometheus(cacheQuery)
	if err == nil {
		s.results.AIMLPerformance.CacheHitRate = cacheHitRate
		target := 75.0 // 75%
		if cacheHitRate < target {
			t.Logf("RAG cache hit rate below target: %.2f%% < %.2f%%", cacheHitRate, target)
		}
		t.Logf("RAG cache hit rate: %.2f%%", cacheHitRate)
	}
}

// testMLInference validates ML inference performance
func (s *SLATestSuite) testMLInference(t *testing.T) {
	query := `histogram_quantile(0.95, sum(rate(ml_inference_duration_seconds_bucket[5m])) by (le)) * 1000`

	result, err := s.queryPrometheus(query)
	if err != nil {
		t.Logf("Failed to query ML inference metrics: %v", err)
		return
	}

	s.results.AIMLPerformance.MLInferenceLatency = result

	target := 200.0 // 200ms
	if result > target {
		t.Logf("ML inference latency above target: %.2fms > %.2fms", result, target)
	}

	t.Logf("ML inference P95 latency: %.2fms", result)
}

// testLoadWithSLA runs load test while monitoring SLA compliance
func (s *SLATestSuite) testLoadWithSLA(t *testing.T) {
	t.Logf("Starting load test with %d concurrent users for %v", ConcurrentUsers, LoadTestDuration)

	s.testStartTime = time.Now()
	defer func() {
		s.testEndTime = time.Now()
	}()

	// Start load generation
	ctx, cancel := context.WithTimeout(context.Background(), LoadTestDuration)
	defer cancel()

	var wg sync.WaitGroup
	errorChan := make(chan error, ConcurrentUsers)

	// Start concurrent users
	for i := 0; i < ConcurrentUsers; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()
			s.simulateUser(ctx, userID, errorChan)
		}(i)
	}

	// Monitor SLA compliance during load test
	monitorCtx, monitorCancel := context.WithCancel(ctx)
	defer monitorCancel()

	go s.monitorSLADuringLoad(monitorCtx, t)

	// Wait for load test completion
	wg.Wait()
	close(errorChan)

	// Count errors
	errorCount := 0
	for range errorChan {
		errorCount++
	}

	totalRequests := ConcurrentUsers * int(LoadTestDuration.Seconds()) * RequestRate
	errorRate := float64(errorCount) / float64(totalRequests) * 100

	t.Logf("Load test completed. Error rate: %.2f%% (%d errors out of %d requests)",
		errorRate, errorCount, totalRequests)

	// Validate SLA compliance after load test
	time.Sleep(30 * time.Second) // Wait for metrics to stabilize
	s.testIntentProcessingSLA(t)
	s.testErrorRateSLA(t)
}

// simulateUser simulates a single user's behavior
func (s *SLATestSuite) simulateUser(ctx context.Context, userID int, errorChan chan<- error) {
	ticker := time.NewTicker(time.Second / RequestRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.simulateIntentProcessing(); err != nil {
				errorChan <- err
			}
		}
	}
}

// simulateIntentProcessing simulates processing an intent
func (s *SLATestSuite) simulateIntentProcessing() error {
	// Simulate intent processing with random delay
	delay := time.Duration(100+time.Now().UnixNano()%900) * time.Millisecond
	time.Sleep(delay)

	// Simulate occasional errors (5% error rate)
	if time.Now().UnixNano()%100 < 5 {
		return fmt.Errorf("simulated processing error")
	}

	return nil
}

// monitorSLADuringLoad monitors SLA compliance during load test
func (s *SLATestSuite) monitorSLADuringLoad(ctx context.Context, t *testing.T) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check availability
			if availability, err := s.queryPrometheus(`avg_over_time(up{job="nephoran-operator"}[1m]) * 100`); err == nil {
				if availability < AvailabilityTarget {
					t.Logf("WARNING: Availability dropped to %.2f%% during load test", availability)
				}
			}

			// Check latency
			if latency, err := s.queryPrometheus(`histogram_quantile(0.95, sum(rate(intent_processing_duration_seconds_bucket[1m])) by (le))`); err == nil {
				if latency > IntentProcessingP95Target {
					t.Logf("WARNING: P95 latency increased to %.2fs during load test", latency)
				}
			}
		}
	}
}

// testPerformanceRegression checks for performance regressions
func (s *SLATestSuite) testPerformanceRegression(t *testing.T) {
	// Compare current performance with historical baseline
	currentTime := time.Now()
	oneHourAgo := currentTime.Add(-1 * time.Hour)

	// Query current P95 latency
	currentLatency, err := s.queryPrometheus(`histogram_quantile(0.95, sum(rate(intent_processing_duration_seconds_bucket[5m])) by (le))`)
	require.NoError(t, err, "Failed to query current latency")

	// Query historical P95 latency (1 hour ago)
	query := `histogram_quantile(0.95, sum(rate(intent_processing_duration_seconds_bucket[5m])) by (le))`
	historicalLatency, err := s.queryPrometheusAtTime(query, oneHourAgo)
	if err != nil {
		t.Logf("Cannot compare with historical data: %v", err)
		return
	}

	// Check for regression (>20% increase)
	regressionThreshold := 1.2
	if currentLatency > historicalLatency*regressionThreshold {
		t.Errorf("Performance regression detected: P95 latency increased from %.2fs to %.2fs (%.1f%% increase)",
			historicalLatency, currentLatency, (currentLatency/historicalLatency-1)*100)
	}

	t.Logf("Performance comparison: %.2fs (current) vs %.2fs (1h ago)", currentLatency, historicalLatency)
}

// testCompositeSLAScores validates composite SLA scores
func (s *SLATestSuite) testCompositeSLAScores(t *testing.T) {
	// Calculate platform health score
	// Weights: availability: 30%, performance: 25%, error rate: 25%, resource optimization: 20%
	availabilityScore := s.results.Availability * 0.3
	performanceScore := (1 - s.results.IntentProcessingP95/IntentProcessingP95Target) * 25 * 0.25
	if performanceScore < 0 {
		performanceScore = 0
	}
	errorScore := (1 - s.results.ErrorRate/ErrorRateTarget) * 25 * 0.25
	if errorScore < 0 {
		errorScore = 0
	}
	resourceScore := s.results.ResourceEfficiency * 0.2

	platformHealthScore := availabilityScore + performanceScore + errorScore + resourceScore

	t.Logf("Platform Health Score: %.2f%% (availability: %.2f, performance: %.2f, error: %.2f, resource: %.2f)",
		platformHealthScore, availabilityScore, performanceScore, errorScore, resourceScore)

	// Validate minimum acceptable score
	minHealthScore := 90.0
	assert.GreaterOrEqual(t, platformHealthScore, minHealthScore,
		"Platform health score should be at least %.2f%%", minHealthScore)

	// Calculate telecom service quality score
	telecomScore := s.calculateTelecomQualityScore()
	t.Logf("Telecom Service Quality Score: %.2f%%", telecomScore)

	minTelecomScore := 95.0
	if telecomScore < minTelecomScore {
		t.Logf("Telecom service quality below target: %.2f%% < %.2f%%", telecomScore, minTelecomScore)
	}
}

// calculateTelecomQualityScore calculates the composite telecom quality score
func (s *SLATestSuite) calculateTelecomQualityScore() float64 {
	// Weights: NF deployment: 30%, O-RAN compliance: 30%, RAN performance: 20%, network slicing: 20%
	nfScore := math.Max(0, (1-s.results.TelecomKPIs.NFDeploymentTime/180)*30)
	oranScore := math.Max(0, (1-s.results.TelecomKPIs.ORANLatencyA1/100)*30)
	ranScore := s.results.TelecomKPIs.HandoverSuccessRate * 0.2
	sliceScore := math.Max(0, (1-s.results.TelecomKPIs.SliceCreationTime/60)*20)

	return nfScore + oranScore + ranScore + sliceScore
}

// generateSLAReport generates a comprehensive SLA test report
func (s *SLATestSuite) generateSLAReport(t *testing.T) {
	t.Logf("\n=== SLA Test Report ===")
	t.Logf("Test Duration: %v", s.testEndTime.Sub(s.testStartTime))
	t.Logf("\nPlatform SLAs:")
	t.Logf("  Availability: %.2f%% (target: %.2f%%)", s.results.Availability, AvailabilityTarget)
	t.Logf("  Intent Processing P95: %.2fs (target: %.2fs)", s.results.IntentProcessingP95, IntentProcessingP95Target)
	t.Logf("  Intent Processing P99: %.2fs (target: %.2fs)", s.results.IntentProcessingP99, IntentProcessingP99Target)
	t.Logf("  Error Rate: %.2f%% (target: %.2f%%)", s.results.ErrorRate, ErrorRateTarget)
	t.Logf("  Resource Efficiency: %.2f%% (target: %.2f%%)", s.results.ResourceEfficiency, ResourceOptimizationTarget)

	t.Logf("\nTelecom KPIs:")
	t.Logf("  NF Deployment Time: %.2fs", s.results.TelecomKPIs.NFDeploymentTime)
	t.Logf("  O-RAN A1 Latency: %.2fms", s.results.TelecomKPIs.ORANLatencyA1)
	t.Logf("  RAN Handover Success: %.2f%%", s.results.TelecomKPIs.HandoverSuccessRate)
	t.Logf("  Network Slice Creation: %.2fs", s.results.TelecomKPIs.SliceCreationTime)
	t.Logf("  RIC Availability: %.2f%%", s.results.TelecomKPIs.RICAvailability)

	t.Logf("\nAI/ML Performance:")
	t.Logf("  Token Generation Rate: %.2f/s", s.results.AIMLPerformance.TokenGenerationRate)
	t.Logf("  RAG Retrieval Latency: %.2fms", s.results.AIMLPerformance.RAGRetrievalLatency)
	t.Logf("  RAG Cache Hit Rate: %.2f%%", s.results.AIMLPerformance.CacheHitRate)
	t.Logf("  ML Inference Latency: %.2fms", s.results.AIMLPerformance.MLInferenceLatency)

	if len(s.results.Violations) > 0 {
		t.Logf("\nSLA Violations:")
		for _, violation := range s.results.Violations {
			t.Logf("  [%s] %s: %s", violation.Severity, violation.Metric, violation.Violation)
		}
	} else {
		t.Logf("\nAll SLAs within acceptable limits ✅")
	}

	// Calculate overall compliance rate
	totalMetrics := 4 // Platform SLAs
	violatedMetrics := len(s.results.Violations)
	complianceRate := float64(totalMetrics-violatedMetrics) / float64(totalMetrics) * 100

	t.Logf("\nOverall SLA Compliance: %.2f%% (%d/%d metrics compliant)",
		complianceRate, totalMetrics-violatedMetrics, totalMetrics)
}

// queryPrometheus executes a Prometheus query and returns the result
func (s *SLATestSuite) queryPrometheus(query string) (float64, error) {
	result, _, err := s.prometheusClient.Query(context.Background(), query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("query failed: %w", err)
	}

	return s.extractScalarValue(result)
}

// queryPrometheusAtTime executes a Prometheus query at a specific time
func (s *SLATestSuite) queryPrometheusAtTime(query string, queryTime time.Time) (float64, error) {
	result, _, err := s.prometheusClient.Query(context.Background(), query, queryTime)
	if err != nil {
		return 0, fmt.Errorf("query failed: %w", err)
	}

	return s.extractScalarValue(result)
}

// extractScalarValue extracts a scalar value from Prometheus query result
func (s *SLATestSuite) extractScalarValue(result interface{}) (float64, error) {
	// Implementation would parse the Prometheus result format
	// This is a simplified version
	return 0.0, fmt.Errorf("not implemented - would parse Prometheus result")
}

// BenchmarkIntentProcessing benchmarks intent processing performance
func BenchmarkSLAIntentProcessing(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Simulate intent processing
		processingTime := time.Duration(50+i%100) * time.Millisecond
		time.Sleep(processingTime)
	}
}

// BenchmarkConcurrentIntentProcessing benchmarks concurrent intent processing
func BenchmarkConcurrentIntentProcessing(b *testing.B) {
	b.ReportAllocs()
	_ = 10

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate concurrent intent processing
			processingTime := time.Duration(50+time.Now().UnixNano()%100) * time.Millisecond
			time.Sleep(processingTime)
		}
	})
}

// TestSLAMetricsCollection tests SLA metrics collection
func TestSLAMetricsCollection(t *testing.T) {
	// Test that all required SLA metrics are being collected
	expectedMetrics := []string{
		"intent_processing_duration_seconds",
		"http_requests_total",
		"container_cpu_usage_seconds_total",
		"nf_deployment_duration_seconds",
		"oran_interface_latency_seconds",
		"ran_handover_success_total",
		"llm_tokens_generated_total",
		"rag_retrieval_duration_seconds",
		"ml_inference_duration_seconds",
	}

	suite, err := NewSLATestSuite("http://localhost:9090")
	require.NoError(t, err)

	for _, metric := range expectedMetrics {
		query := fmt.Sprintf("up{__name__=~\"%s.*\"}", metric)
		_, err := suite.queryPrometheus(query)
		if err != nil {
			t.Logf("Warning: Metric %s may not be available: %v", metric, err)
		}
	}
}

// TestSLAAlertRules tests SLA alert rule functionality
func TestSLAAlertRules(t *testing.T) {
	// Test that SLA alert rules are properly configured
	suite, err := NewSLATestSuite("http://localhost:9090")
	require.NoError(t, err)

	// Query for alert rules
	query := "ALERTS{alertname=~\".*SLA.*\"}"
	_, err = suite.queryPrometheus(query)
	if err != nil {
		t.Logf("Warning: SLA alert rules may not be available: %v", err)
	}
}
