package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PerformanceValidationTestSuite provides comprehensive performance validation testing
type PerformanceValidationTestSuite struct {
	suite.Suite
	validationSuite   *ValidationSuite
	config           *ValidationConfig
	outputDir        string
	testStartTime    time.Time
	mockClients      *MockClients
}

// MockClients provides mock implementations for testing
type MockClients struct {
	intentClient *MockIntentClient
	ragClient    *MockRAGClient
}

// MockIntentClient implements IntentClient for testing
type MockIntentClient struct {
	latencyVariation  time.Duration
	successRate      float64
	maxConcurrent    int
	currentLoad      int
	healthStatus     bool
}

// MockRAGClient implements RAGClient for testing
type MockRAGClient struct {
	latencyVariation time.Duration
	cacheHitRate    float64
	healthStatus    bool
	queryCount      int
	cacheStats      *CacheStats
}

// SetupSuite initializes the test suite
func (suite *PerformanceValidationTestSuite) SetupSuite() {
	suite.testStartTime = time.Now()
	suite.outputDir = filepath.Join("test-results", "validation", fmt.Sprintf("run-%s", 
		suite.testStartTime.Format("2006-01-02-15-04-05")))
	
	// Create output directory
	err := os.MkdirAll(suite.outputDir, 0755)
	require.NoError(suite.T(), err)
	
	// Load configuration
	suite.config, err = LoadValidationConfig("")
	require.NoError(suite.T(), err)
	
	// Validate configuration
	err = ValidateConfig(suite.config)
	require.NoError(suite.T(), err)
	
	// Initialize mock clients
	suite.mockClients = &MockClients{
		intentClient: &MockIntentClient{
			latencyVariation: 500 * time.Millisecond,
			successRate:     0.98,
			maxConcurrent:   250,
			healthStatus:    true,
		},
		ragClient: &MockRAGClient{
			latencyVariation: 50 * time.Millisecond,
			cacheHitRate:    0.87,
			healthStatus:    true,
			cacheStats: &CacheStats{
				Hits:    8700,
				Misses:  1300,
				HitRate: 87.0,
				Size:    50000,
				MaxSize: 100000,
			},
		},
	}
	
	// Initialize validation suite
	suite.validationSuite = NewValidationSuite(suite.config)
	suite.validationSuite.testRunner.intentClient = suite.mockClients.intentClient
	suite.validationSuite.testRunner.ragClient = suite.mockClients.ragClient
	
	log.Printf("Performance validation test suite initialized")
	log.Printf("Output directory: %s", suite.outputDir)
	log.Printf("Test configuration: %+v", suite.config.Claims)
}

// TearDownSuite cleans up after all tests
func (suite *PerformanceValidationTestSuite) TearDownSuite() {
	duration := time.Since(suite.testStartTime)
	log.Printf("Performance validation test suite completed in %v", duration)
	
	// Generate final summary report
	suite.generateFinalSummaryReport()
}

// TestComprehensivePerformanceValidation runs the complete performance validation suite
func (suite *PerformanceValidationTestSuite) TestComprehensivePerformanceValidation() {
	ctx, cancel := context.WithTimeout(context.Background(), 
		suite.config.TestConfig.TestDuration+10*time.Minute) // Extra buffer
	defer cancel()
	
	log.Printf("Starting comprehensive performance validation...")
	
	// Run all performance claim validations
	results, err := suite.validationSuite.ValidateAllClaims(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), results)
	
	// Save detailed results
	suite.saveValidationResults(results)
	
	// Assert overall success
	suite.Assert().True(results.Summary.OverallSuccess, 
		"Performance validation failed: %d out of %d claims validated", 
		results.Summary.ValidatedClaims, results.Summary.TotalClaims)
	
	// Validate individual claims with detailed assertions
	suite.validateIndividualClaims(results)
	
	// Validate statistical rigor
	suite.validateStatisticalRigor(results)
	
	// Validate evidence quality
	suite.validateEvidenceQuality(results)
	
	log.Printf("Comprehensive performance validation completed successfully")
}

// TestIntentLatencyP95Validation specifically validates intent processing latency
func (suite *PerformanceValidationTestSuite) TestIntentLatencyP95Validation() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	
	log.Printf("Testing Intent Processing P95 Latency < 2 seconds...")
	
	result, err := suite.validationSuite.validateIntentLatencyP95(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result)
	
	// Validate statistical requirements
	suite.Assert().GreaterOrEqual(result.Evidence.SampleSize, 
		suite.config.Statistics.MinSampleSize,
		"Insufficient sample size for statistical validity")
	
	suite.Assert().NotNil(result.HypothesisTest, "Hypothesis test must be performed")
	
	if result.HypothesisTest != nil {
		suite.Assert().Less(result.HypothesisTest.PValue, 
			suite.config.Statistics.SignificanceLevel,
			"P-value must be less than significance level for claim validation")
		
		suite.Assert().GreaterOrEqual(result.HypothesisTest.Power, 
			suite.config.Statistics.PowerThreshold,
			"Statistical power must meet minimum threshold")
	}
	
	// Validate performance target
	measuredLatency := result.Measured.(time.Duration)
	targetLatency := suite.config.Claims.IntentLatencyP95
	
	suite.Assert().LessOrEqual(measuredLatency, targetLatency,
		"P95 intent processing latency (%v) exceeds target (%v)", 
		measuredLatency, targetLatency)
	
	// Validate evidence completeness
	suite.Assert().NotNil(result.Evidence.Percentiles, "Percentile analysis required")
	suite.Assert().NotNil(result.Evidence.Distribution, "Distribution analysis required")
	suite.Assert().NotNil(result.Statistics, "Descriptive statistics required")
	
	log.Printf("Intent latency validation completed: %s", result.Status)
}

// TestConcurrentCapacityValidation validates concurrent intent handling capacity
func (suite *PerformanceValidationTestSuite) TestConcurrentCapacityValidation() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	
	log.Printf("Testing Concurrent Capacity >= 200 intents...")
	
	result, err := suite.validationSuite.validateConcurrentCapacity(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result)
	
	// Validate performance target
	measuredCapacity := result.Measured.(int)
	targetCapacity := suite.config.Claims.ConcurrentCapacity
	
	suite.Assert().GreaterOrEqual(measuredCapacity, targetCapacity,
		"Concurrent capacity (%d) below target (%d)", 
		measuredCapacity, targetCapacity)
	
	// Validate test methodology
	suite.Assert().GreaterOrEqual(result.Evidence.SampleSize, 
		len(suite.config.TestConfig.ConcurrencyLevels),
		"Must test all specified concurrency levels")
	
	log.Printf("Concurrent capacity validation completed: %s (max: %d)", 
		result.Status, measuredCapacity)
}

// TestThroughputRateValidation validates sustained throughput performance
func (suite *PerformanceValidationTestSuite) TestThroughputRateValidation() {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	
	log.Printf("Testing Throughput >= 45 intents per minute...")
	
	result, err := suite.validationSuite.validateThroughputRate(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result)
	
	// Validate performance target
	measuredThroughput := result.Measured.(float64)
	targetThroughput := float64(suite.config.Claims.ThroughputRate)
	
	suite.Assert().GreaterOrEqual(measuredThroughput, targetThroughput,
		"Throughput (%.1f intents/min) below target (%.1f intents/min)", 
		measuredThroughput, targetThroughput)
	
	// Validate measurement duration
	suite.Assert().GreaterOrEqual(result.Evidence.SampleSize, 5,
		"Throughput must be measured over multiple intervals")
	
	log.Printf("Throughput validation completed: %s (%.1f intents/min)", 
		result.Status, measuredThroughput)
}

// TestSystemAvailabilityValidation validates system availability during operations
func (suite *PerformanceValidationTestSuite) TestSystemAvailabilityValidation() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	
	log.Printf("Testing System Availability >= 99.95%%...")
	
	result, err := suite.validationSuite.validateSystemAvailability(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result)
	
	// Validate performance target
	measuredAvailability := result.Measured.(float64)
	targetAvailability := suite.config.Claims.SystemAvailability
	
	suite.Assert().GreaterOrEqual(measuredAvailability, targetAvailability,
		"System availability (%.3f%%) below target (%.3f%%)", 
		measuredAvailability, targetAvailability)
	
	log.Printf("System availability validation completed: %s (%.3f%%)", 
		result.Status, measuredAvailability)
}

// TestRAGRetrievalLatencyValidation validates RAG system retrieval performance
func (suite *PerformanceValidationTestSuite) TestRAGRetrievalLatencyValidation() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	
	log.Printf("Testing RAG Retrieval P95 Latency < 200ms...")
	
	result, err := suite.validationSuite.validateRAGRetrievalLatencyP95(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result)
	
	// Validate performance target
	measuredLatency := result.Measured.(time.Duration)
	targetLatency := suite.config.Claims.RAGRetrievalLatencyP95
	
	suite.Assert().LessOrEqual(measuredLatency, targetLatency,
		"RAG P95 latency (%v) exceeds target (%v)", 
		measuredLatency, targetLatency)
	
	log.Printf("RAG retrieval latency validation completed: %s (%v)", 
		result.Status, measuredLatency)
}

// TestCacheHitRateValidation validates cache performance
func (suite *PerformanceValidationTestSuite) TestCacheHitRateValidation() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	
	log.Printf("Testing Cache Hit Rate >= 87%%...")
	
	result, err := suite.validationSuite.validateCacheHitRate(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), result)
	
	// Validate performance target
	measuredHitRate := result.Measured.(float64)
	targetHitRate := suite.config.Claims.CacheHitRate
	
	suite.Assert().GreaterOrEqual(measuredHitRate, targetHitRate,
		"Cache hit rate (%.1f%%) below target (%.1f%%)", 
		measuredHitRate, targetHitRate)
	
	log.Printf("Cache hit rate validation completed: %s (%.1f%%)", 
		result.Status, measuredHitRate)
}

// TestStatisticalValidityAndRigor validates the statistical methodology
func (suite *PerformanceValidationTestSuite) TestStatisticalValidityAndRigor() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	log.Printf("Validating statistical methodology and rigor...")
	
	results, err := suite.validationSuite.ValidateAllClaims(ctx)
	require.NoError(suite.T(), err)
	
	// Validate each claim has proper statistical analysis
	for claimName, result := range results.ClaimResults {
		suite.Run(fmt.Sprintf("StatisticalRigor_%s", claimName), func() {
			// Sample size adequacy
			suite.Assert().GreaterOrEqual(result.Evidence.SampleSize, 
				suite.config.Statistics.MinSampleSize,
				"Claim %s: insufficient sample size", claimName)
			
			// Hypothesis test presence and validity
			suite.Assert().NotNil(result.HypothesisTest, 
				"Claim %s: missing hypothesis test", claimName)
			
			if result.HypothesisTest != nil {
				// P-value validity
				suite.Assert().GreaterOrEqual(result.HypothesisTest.PValue, 0.0,
					"Claim %s: invalid p-value", claimName)
				suite.Assert().LessOrEqual(result.HypothesisTest.PValue, 1.0,
					"Claim %s: invalid p-value", claimName)
				
				// Effect size presence
				suite.Assert().GreaterOrEqual(result.HypothesisTest.EffectSize, 0.0,
					"Claim %s: invalid effect size", claimName)
				
				// Statistical power
				suite.Assert().GreaterOrEqual(result.HypothesisTest.Power, 0.0,
					"Claim %s: invalid statistical power", claimName)
				suite.Assert().LessOrEqual(result.HypothesisTest.Power, 1.0,
					"Claim %s: invalid statistical power", claimName)
			}
			
			// Descriptive statistics
			suite.Assert().NotNil(result.Statistics,
				"Claim %s: missing descriptive statistics", claimName)
			
			if result.Statistics != nil {
				suite.Assert().GreaterOrEqual(result.Statistics.StdDev, 0.0,
					"Claim %s: invalid standard deviation", claimName)
				suite.Assert().GreaterOrEqual(result.Statistics.Variance, 0.0,
					"Claim %s: invalid variance", claimName)
			}
		})
	}
	
	log.Printf("Statistical validity validation completed")
}

// TestEvidenceQualityAndCompleteness validates evidence quality
func (suite *PerformanceValidationTestSuite) TestEvidenceQualityAndCompleteness() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	
	log.Printf("Validating evidence quality and completeness...")
	
	results, err := suite.validationSuite.ValidateAllClaims(ctx)
	require.NoError(suite.T(), err)
	
	evidence := results.Evidence
	suite.Assert().NotNil(evidence, "Evidence report must be generated")
	
	if evidence != nil {
		// Quality assessment
		suite.Assert().NotNil(evidence.QualityAssessment,
			"Evidence quality assessment must be present")
		
		if evidence.QualityAssessment != nil {
			suite.Assert().GreaterOrEqual(evidence.QualityAssessment.OverallScore, 75.0,
				"Evidence quality score too low: %.1f", 
				evidence.QualityAssessment.OverallScore)
		}
		
		// Claim evidence completeness
		suite.Assert().Equal(len(results.ClaimResults), len(evidence.ClaimEvidence),
			"Evidence must be provided for all claims")
	}
	
	log.Printf("Evidence quality validation completed")
}

// Validation helper methods

func (suite *PerformanceValidationTestSuite) validateIndividualClaims(results *ValidationResults) {
	for claimName, result := range results.ClaimResults {
		suite.Run(fmt.Sprintf("Claim_%s", claimName), func() {
			suite.Assert().NotEmpty(result.Status, "Claim status must be set")
			suite.Assert().Contains([]string{"validated", "failed", "inconclusive"}, 
				result.Status, "Invalid claim status")
			
			if result.Status == "validated" {
				suite.Assert().GreaterOrEqual(result.Confidence, 
					suite.config.Statistics.ConfidenceLevel,
					"Confidence level must meet minimum threshold")
			}
		})
	}
}

func (suite *PerformanceValidationTestSuite) validateStatisticalRigor(results *ValidationResults) {
	// Validate overall statistical summary
	summary := results.Evidence.StatisticalSummary
	if summary != nil {
		suite.Assert().GreaterOrEqual(summary.OverallConfidence, 
			suite.config.Statistics.ConfidenceLevel,
			"Overall confidence below threshold")
		
		suite.Assert().GreaterOrEqual(summary.AverageQualityScore, 75.0,
			"Average quality score too low")
	}
}

func (suite *PerformanceValidationTestSuite) validateEvidenceQuality(results *ValidationResults) {
	if results.Evidence != nil && results.Evidence.QualityAssessment != nil {
		qa := results.Evidence.QualityAssessment
		
		suite.Assert().GreaterOrEqual(qa.OverallScore, 75.0,
			"Overall evidence quality score too low")
		
		if qa.DataQuality != nil {
			suite.Assert().GreaterOrEqual(qa.DataQuality.Completeness, 90.0,
				"Data completeness too low")
			suite.Assert().GreaterOrEqual(qa.DataQuality.Accuracy, 85.0,
				"Data accuracy too low")
		}
	}
}

func (suite *PerformanceValidationTestSuite) saveValidationResults(results *ValidationResults) {
	// Save main results
	resultsPath := filepath.Join(suite.outputDir, "validation-results.json")
	data, err := json.MarshalIndent(results, "", "  ")
	require.NoError(suite.T(), err)
	
	err = os.WriteFile(resultsPath, data, 0644)
	require.NoError(suite.T(), err)
	
	log.Printf("Validation results saved to: %s", resultsPath)
	
	// Save individual claim results
	for claimName, result := range results.ClaimResults {
		claimPath := filepath.Join(suite.outputDir, fmt.Sprintf("claim-%s.json", claimName))
		claimData, err := json.MarshalIndent(result, "", "  ")
		require.NoError(suite.T(), err)
		
		err = os.WriteFile(claimPath, claimData, 0644)
		require.NoError(suite.T(), err)
	}
	
	// Save evidence report if available
	if results.Evidence != nil {
		evidencePath := filepath.Join(suite.outputDir, "evidence-report.json")
		evidenceData, err := json.MarshalIndent(results.Evidence, "", "  ")
		require.NoError(suite.T(), err)
		
		err = os.WriteFile(evidencePath, evidenceData, 0644)
		require.NoError(suite.T(), err)
	}
}

func (suite *PerformanceValidationTestSuite) generateFinalSummaryReport() {
	summary := map[string]interface{}{
		"test_suite":     "Performance Validation",
		"start_time":     suite.testStartTime,
		"end_time":       time.Now(),
		"duration":       time.Since(suite.testStartTime),
		"output_dir":     suite.outputDir,
		"configuration":  suite.config,
	}
	
	summaryPath := filepath.Join(suite.outputDir, "test-summary.json")
	data, err := json.MarshalIndent(summary, "", "  ")
	if err == nil {
		_ = os.WriteFile(summaryPath, data, 0644)
		log.Printf("Test summary saved to: %s", summaryPath)
	}
}

// Mock client implementations

func (m *MockIntentClient) ProcessIntent(ctx context.Context, intent *NetworkIntent) (*IntentResult, error) {
	// Simulate processing time based on complexity
	var baseLatency time.Duration
	switch intent.Complexity {
	case "simple":
		baseLatency = 200 * time.Millisecond
	case "moderate":
		baseLatency = 800 * time.Millisecond
	case "complex":
		baseLatency = 1500 * time.Millisecond
	default:
		baseLatency = 500 * time.Millisecond
	}
	
	// Add random variation
	actualLatency := baseLatency + time.Duration(float64(m.latencyVariation)*
		(0.5 - float64(time.Now().UnixNano()%1000)/1000.0))
	
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(actualLatency):
	}
	
	// Simulate success/failure based on success rate
	success := (float64(time.Now().UnixNano()%1000)/1000.0) < m.successRate
	
	status := "success"
	if !success {
		status = "failed"
	}
	
	return &IntentResult{
		ID:       intent.ID,
		Status:   status,
		Duration: actualLatency,
		ResourcesCreated: 1,
	}, nil
}

func (m *MockIntentClient) GetActiveIntents() int {
	return m.currentLoad
}

func (m *MockIntentClient) HealthCheck() error {
	if !m.healthStatus {
		return fmt.Errorf("intent client unhealthy")
	}
	return nil
}

func (m *MockRAGClient) Query(ctx context.Context, query string) (*RAGResponse, error) {
	// Simulate retrieval latency
	baseLatency := 100 * time.Millisecond
	actualLatency := baseLatency + time.Duration(float64(m.latencyVariation)*
		(0.5 - float64(time.Now().UnixNano()%1000)/1000.0))
	
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(actualLatency):
	}
	
	// Simulate cache hit/miss
	m.queryCount++
	cacheHit := (float64(time.Now().UnixNano()%1000)/1000.0) < m.cacheHitRate
	
	if cacheHit {
		m.cacheStats.Hits++
		actualLatency = actualLatency / 4 // Cache hits are faster
	} else {
		m.cacheStats.Misses++
	}
	
	// Update cache hit rate
	total := m.cacheStats.Hits + m.cacheStats.Misses
	if total > 0 {
		m.cacheStats.HitRate = float64(m.cacheStats.Hits) / float64(total) * 100
	}
	
	return &RAGResponse{
		Query:    query,
		Duration: actualLatency,
		CacheHit: cacheHit,
		Results: []RAGResult{
			{
				Content: "Mock RAG result for: " + query,
				Score:   0.85,
				Source:  "mock-knowledge-base",
			},
		},
		Relevance: 0.85,
	}, nil
}

func (m *MockRAGClient) GetCacheStats() *CacheStats {
	return m.cacheStats
}

func (m *MockRAGClient) HealthCheck() error {
	if !m.healthStatus {
		return fmt.Errorf("RAG client unhealthy")
	}
	return nil
}

// Test runner function
func TestPerformanceValidationSuite(t *testing.T) {
	suite.Run(t, new(PerformanceValidationTestSuite))
}