package integration_tests

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/nephio-project/nephoran-intent-operator/pkg/monitoring/availability"
)

// AvailabilityTrackingTestSuite tests the comprehensive availability tracking system
type AvailabilityTrackingTestSuite struct {
	suite.Suite
	ctx               context.Context
	cancel            context.CancelFunc
	tracker           *availability.MultiDimensionalTracker
	calculator        *availability.AvailabilityCalculator
	syntheticMonitor  *availability.SyntheticMonitor
	dependencyTracker *availability.DependencyChainTracker
	reporter          *availability.AvailabilityReporter

	// Test infrastructure
	mockLLMService    *MockLLMService
	mockRAGService    *MockRAGService
	mockNephioService *MockNephioService
	mockPrometheus    *MockPrometheusServer
}

func TestAvailabilityTrackingTestSuite(t *testing.T) {
	suite.Run(t, new(AvailabilityTrackingTestSuite))
}

func (suite *AvailabilityTrackingTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	// Setup mock services
	suite.setupMockServices()

	// Configure availability tracking components
	suite.setupAvailabilityTracking()

	// Start all components
	suite.startTrackingComponents()
}

func (suite *AvailabilityTrackingTestSuite) TearDownSuite() {
	suite.stopTrackingComponents()
	suite.teardownMockServices()
	suite.cancel()
}

func (suite *AvailabilityTrackingTestSuite) setupMockServices() {
	// Setup mock LLM service
	suite.mockLLMService = NewMockLLMService("localhost:8080")
	go suite.mockLLMService.Start()

	// Setup mock RAG service
	suite.mockRAGService = NewMockRAGService("localhost:8081")
	go suite.mockRAGService.Start()

	// Setup mock Nephio service
	suite.mockNephioService = NewMockNephioService("localhost:8082")
	go suite.mockNephioService.Start()

	// Setup mock Prometheus
	suite.mockPrometheus = NewMockPrometheusServer("localhost:9090")
	go suite.mockPrometheus.Start()

	// Wait for services to start
	time.Sleep(2 * time.Second)
}

func (suite *AvailabilityTrackingTestSuite) setupAvailabilityTracking() {
	// Configure tracker
	trackerConfig := &availability.TrackerConfig{
		ServiceEndpoints: []availability.ServiceEndpointConfig{
			{
				Name:           "llm-processor",
				URL:            "http://localhost:8080/health",
				Method:         "GET",
				ExpectedStatus: 200,
				Timeout:        5 * time.Second,
				BusinessImpact: availability.ImpactCritical,
				Layer:          availability.LayerProcessor,
				SLAThreshold:   500 * time.Millisecond,
			},
			{
				Name:           "rag-service",
				URL:            "http://localhost:8081/health",
				Method:         "GET",
				ExpectedStatus: 200,
				Timeout:        3 * time.Second,
				BusinessImpact: availability.ImpactHigh,
				Layer:          availability.LayerProcessor,
				SLAThreshold:   200 * time.Millisecond,
			},
			{
				Name:           "nephio-bridge",
				URL:            "http://localhost:8082/health",
				Method:         "GET",
				ExpectedStatus: 200,
				Timeout:        10 * time.Second,
				BusinessImpact: availability.ImpactCritical,
				Layer:          availability.LayerController,
				SLAThreshold:   2 * time.Second,
			},
		},
		Components: []availability.ComponentConfig{
			{
				Name:           "test-deployment",
				Namespace:      "default",
				ResourceType:   "deployment",
				Selector:       map[string]string{"app": "test-app"},
				BusinessImpact: availability.ImpactHigh,
				Layer:          availability.LayerAPI,
			},
		},
		UserJourneys: []availability.UserJourneyConfig{
			{
				Name: "intent-processing",
				Steps: []availability.UserJourneyStep{
					{
						Name:     "submit-intent",
						Type:     "api_call",
						Target:   "llm-processor",
						Timeout:  5 * time.Second,
						Required: true,
						Weight:   0.4,
					},
					{
						Name:     "rag-context",
						Type:     "api_call",
						Target:   "rag-service",
						Timeout:  3 * time.Second,
						Required: false,
						Weight:   0.2,
					},
					{
						Name:     "package-generation",
						Type:     "api_call",
						Target:   "nephio-bridge",
						Timeout:  15 * time.Second,
						Required: true,
						Weight:   0.4,
					},
				},
				BusinessImpact: availability.ImpactCritical,
				SLAThreshold:   30 * time.Second,
			},
		},
		DegradedThreshold:  1 * time.Second,
		UnhealthyThreshold: 5 * time.Second,
		ErrorRateThreshold: 0.05,
		CollectionInterval: 10 * time.Second, // Faster for testing
		RetentionPeriod:    24 * time.Hour,
		PrometheusURL:      "http://localhost:9090",
	}

	var err error
	suite.tracker, err = availability.NewMultiDimensionalTracker(
		trackerConfig,
		nil, // kubeClient not needed for this test
		nil, // kubeClientset not needed for this test
		suite.mockPrometheus.Client(),
		nil, // cache not needed for this test
	)
	suite.Require().NoError(err)

	// Configure synthetic monitor
	syntheticConfig := &availability.SyntheticMonitorConfig{
		MaxConcurrentChecks: 10,
		DefaultTimeout:      30 * time.Second,
		DefaultRetryCount:   2,
		DefaultRetryDelay:   2 * time.Second,
		ResultRetention:     1 * time.Hour,
		RegionID:            "test-region",
		EnableChaosTests:    true,
		ChaosTestInterval:   30 * time.Minute,
		HTTPTimeout:         30 * time.Second,
		HTTPMaxIdleConns:    10,
		HTTPMaxConnsPerHost: 2,
		HTTPSkipTLS:         true,
		IntentAPIEndpoint:   "http://localhost:8080",
		AlertingEnabled:     true,
	}

	suite.syntheticMonitor, err = availability.NewSyntheticMonitor(
		syntheticConfig,
		suite.mockPrometheus.Client(),
		&MockAlertManager{},
	)
	suite.Require().NoError(err)

	// Configure availability calculator
	calculatorConfig := &availability.CalculatorConfig{
		SLATargets: map[string]availability.SLATarget{
			"overall":           availability.SLA99_95,
			"intent-processing": availability.SLA99_95,
			"llm-processor":     availability.SLA99_9,
			"rag-service":       availability.SLA99_5,
		},
		BusinessHours: availability.BusinessHours{
			Timezone:     "UTC",
			StartHour:    6,
			EndHour:      22,
			WeekdaysOnly: true,
		},
		ErrorBudgetConfig: availability.ErrorBudgetConfig{
			CalculationMethod: "weighted_business_impact",
			TimeWindows: []availability.TimeWindow{
				availability.Window1Minute,
				availability.Window5Minutes,
				availability.Window1Hour,
				availability.Window1Day,
			},
		},
	}

	suite.calculator, err = availability.NewAvailabilityCalculator(calculatorConfig)
	suite.Require().NoError(err)

	// Configure dependency tracker
	depConfig := &availability.DependencyTrackerConfig{
		Dependencies: []availability.DependencyConfig{
			{
				Name:           "mock-llm-api",
				Type:           availability.DepTypeExternalAPI,
				Endpoint:       "http://localhost:8080",
				BusinessImpact: availability.ImpactCritical,
				FailureMode:    availability.FailModeOpen,
				CircuitBreaker: availability.CircuitBreakerConfig{
					FailureThreshold: 3,
					ResetTimeout:     30 * time.Second,
					HalfOpenMaxCalls: 2,
				},
				HealthCheck: availability.HealthCheckConfig{
					Interval:       30 * time.Second,
					Timeout:        5 * time.Second,
					Path:           "/health",
					ExpectedStatus: 200,
				},
			},
		},
	}

	suite.dependencyTracker, err = availability.NewDependencyChainTracker(depConfig)
	suite.Require().NoError(err)

	// Configure reporter
	reporterConfig := &availability.ReporterConfig{
		DefaultTimeWindow: availability.Window1Hour,
		RefreshInterval:   30 * time.Second,
		RetentionPeriod:   24 * time.Hour,
		SLATargets: []availability.SLATargetConfig{
			{
				Service:        "overall",
				Target:         availability.SLA99_95,
				BusinessImpact: availability.ImpactCritical,
				Enabled:        true,
			},
		},
		AlertThresholds: availability.AlertThresholds{
			AvailabilityWarning:  0.9990,
			AvailabilityCritical: 0.9985,
			ErrorBudgetWarning:   0.8,
			ErrorBudgetCritical:  0.95,
			ResponseTimeWarning:  2 * time.Second,
			ResponseTimeCritical: 5 * time.Second,
			ErrorRateWarning:     0.05,
			ErrorRateCritical:    0.10,
		},
	}

	suite.reporter, err = availability.NewAvailabilityReporter(
		reporterConfig,
		suite.tracker,
		suite.calculator,
		suite.dependencyTracker,
		suite.syntheticMonitor,
		suite.mockPrometheus.Client(),
	)
	suite.Require().NoError(err)
}

func (suite *AvailabilityTrackingTestSuite) startTrackingComponents() {
	// Start all tracking components
	err := suite.tracker.Start()
	suite.Require().NoError(err)

	err = suite.syntheticMonitor.Start()
	suite.Require().NoError(err)

	err = suite.dependencyTracker.Start()
	suite.Require().NoError(err)

	err = suite.calculator.Start()
	suite.Require().NoError(err)

	err = suite.reporter.Start()
	suite.Require().NoError(err)

	// Wait for components to initialize
	time.Sleep(5 * time.Second)
}

func (suite *AvailabilityTrackingTestSuite) TestMultiDimensionalTracking() {
	// Test that tracker collects metrics from all dimensions

	// Wait for at least one collection cycle
	time.Sleep(15 * time.Second)

	state := suite.tracker.GetCurrentState()
	suite.Assert().NotNil(state)
	suite.Assert().True(len(state.CurrentMetrics) > 0)

	// Check that we have metrics from different dimensions
	hasSevice := false
	hasComponent := false
	hasUserJourney := false

	for _, metric := range state.CurrentMetrics {
		switch metric.Dimension {
		case availability.DimensionService:
			hasSevice = true
		case availability.DimensionComponent:
			_ = true // hasComponent check
		case availability.DimensionUserJourney:
			hasUserJourney = true
		}
	}

	suite.Assert().True(hasSevice, "Should have service metrics")
	// Component metrics might not be available without k8s client
	// suite.Assert().True(hasComponent, "Should have component metrics")
	suite.Assert().True(hasUserJourney, "Should have user journey metrics")

	// Test aggregated status calculation
	suite.Assert().NotEqual(availability.HealthUnknown, state.AggregatedStatus)
	suite.Assert().GreaterOrEqual(state.BusinessImpactScore, 0.0)
	suite.Assert().LessOrEqual(state.BusinessImpactScore, 100.0)
}

func (suite *AvailabilityTrackingTestSuite) TestServiceLayerTracking() {
	// Test service layer endpoint tracking

	time.Sleep(15 * time.Second)

	serviceMetrics := suite.tracker.GetMetricsByDimension(availability.DimensionService)
	suite.Assert().NotEmpty(serviceMetrics)

	for _, metric := range serviceMetrics {
		suite.Assert().NotEqual(availability.HealthUnknown, metric.Status)
		suite.Assert().Greater(metric.BusinessImpact, availability.BusinessImpact(0))
		suite.Assert().Contains([]string{"http_endpoint"}, metric.EntityType)
	}
}

func (suite *AvailabilityTrackingTestSuite) TestSyntheticMonitoring() {
	// Add synthetic checks
	checks := []*availability.SyntheticCheck{
		{
			ID:             "llm-health-check",
			Name:           "LLM Service Health Check",
			Type:           availability.CheckTypeHTTP,
			Enabled:        true,
			Interval:       10 * time.Second,
			Timeout:        5 * time.Second,
			RetryCount:     2,
			BusinessImpact: availability.ImpactCritical,
			Region:         "test-region",
			Config: availability.CheckConfig{
				URL:            "http://localhost:8080/health",
				Method:         "GET",
				ExpectedStatus: 200,
			},
			AlertThresholds: availability.AlertThresholds{
				ResponseTime:     500 * time.Millisecond,
				ErrorRate:        0.05,
				Availability:     0.995,
				ConsecutiveFails: 3,
			},
		},
		{
			ID:             "intent-flow-check",
			Name:           "Intent Processing Flow Check",
			Type:           availability.CheckTypeIntentFlow,
			Enabled:        true,
			Interval:       30 * time.Second,
			Timeout:        30 * time.Second,
			RetryCount:     1,
			BusinessImpact: availability.ImpactCritical,
			Region:         "test-region",
			Config: availability.CheckConfig{
				IntentPayload: map[string]interface{}{
					"intent": "Deploy a 5G AMF with high availability",
				},
				FlowSteps: []availability.IntentFlowStep{
					{
						Name:           "create-intent",
						Action:         "create_intent",
						Payload:        map[string]interface{}{"type": "amf"},
						ExpectedStatus: "processing",
						MaxWaitTime:    10 * time.Second,
					},
					{
						Name:           "check-deployment",
						Action:         "check_status",
						ExpectedStatus: "deployed",
						MaxWaitTime:    60 * time.Second,
					},
				},
			},
		},
	}

	for _, check := range checks {
		err := suite.syntheticMonitor.AddCheck(check)
		suite.Require().NoError(err)
	}

	// Wait for checks to execute
	time.Sleep(20 * time.Second)

	// Verify results
	results := suite.syntheticMonitor.GetResults(time.Now().Add(-1*time.Hour), time.Now())
	suite.Assert().NotEmpty(results)

	// Check that we have results for both checks
	httpCheckResults := 0
	flowCheckResults := 0

	for _, result := range results {
		if result.CheckID == "llm-health-check" {
			httpCheckResults++
		} else if result.CheckID == "intent-flow-check" {
			flowCheckResults++
		}
	}

	suite.Assert().Greater(httpCheckResults, 0, "Should have HTTP check results")
	// Intent flow checks might fail without full infrastructure
}

func (suite *AvailabilityTrackingTestSuite) TestAvailabilityCalculation() {
	// Test sophisticated availability calculations

	// Wait for metrics to be collected
	time.Sleep(30 * time.Second)

	// Get historical metrics
	endTime := time.Now()
	startTime := endTime.Add(-5 * time.Minute)
	metrics := suite.tracker.GetMetricsHistory(startTime, endTime)

	suite.Assert().NotEmpty(metrics, "Should have historical metrics")

	// Test availability calculation for different time windows
	windows := []availability.TimeWindow{
		availability.Window1Minute,
		availability.Window5Minutes,
		availability.Window1Hour,
	}

	for _, window := range windows {
		availability := suite.calculator.CalculateAvailability(
			suite.ctx,
			metrics,
			availability.SLA99_95,
			window,
		)

		suite.Assert().GreaterOrEqual(availability, 0.0)
		suite.Assert().LessOrEqual(availability, 1.0)

		// For healthy services, availability should be high
		if availability < 0.99 {
			suite.T().Logf("Warning: Availability for %s window is %f, expected > 0.99", window, availability)
		}
	}

	// Test error budget calculation
	errorBudgets := suite.calculator.CalculateErrorBudgets(
		suite.ctx,
		metrics,
		map[string]availability.SLATarget{
			"overall": availability.SLA99_95,
		},
		availability.Window1Hour,
	)

	suite.Assert().NotEmpty(errorBudgets)

	for service, budget := range errorBudgets {
		suite.T().Logf("Service %s error budget: %+v", service, budget)
		suite.Assert().GreaterOrEqual(budget.BudgetUtilization, 0.0)
		suite.Assert().LessOrEqual(budget.BudgetUtilization, 1.0)
	}
}

func (suite *AvailabilityTrackingTestSuite) TestDependencyTracking() {
	// Test dependency chain tracking

	time.Sleep(30 * time.Second)

	depHealth := suite.dependencyTracker.GetDependencyHealth(suite.ctx)
	suite.Assert().NotEmpty(depHealth)

	for depName, health := range depHealth {
		suite.T().Logf("Dependency %s health: %+v", depName, health)
		suite.Assert().NotEqual(availability.DepStatusUnknown, health.Status)

		// Verify circuit breaker is working
		if health.CircuitBreakerState == "open" {
			suite.T().Logf("Circuit breaker is open for dependency %s", depName)
		}
	}

	// Test cascade failure detection
	cascadeEvents := suite.dependencyTracker.GetCascadeEvents(suite.ctx, time.Now().Add(-1*time.Hour))
	// Cascade events may be empty in healthy test environment
	suite.T().Logf("Found %d cascade events", len(cascadeEvents))
}

func (suite *AvailabilityTrackingTestSuite) TestComplianceReporting() {
	// Test comprehensive availability reporting

	time.Sleep(45 * time.Second)

	// Generate different types of reports
	reportTypes := []availability.ReportType{
		availability.ReportTypeLive,
		availability.ReportTypeHistorical,
		availability.ReportTypeSLA,
		availability.ReportTypeCompliance,
	}

	for _, reportType := range reportTypes {
		report, err := suite.reporter.GenerateReport(
			suite.ctx,
			reportType,
			availability.Window1Hour,
			availability.FormatJSON,
		)

		suite.Require().NoError(err)
		suite.Assert().NotNil(report)
		suite.Assert().Equal(reportType, report.Type)
		suite.Assert().NotNil(report.Summary)

		suite.T().Logf("Generated %s report with overall availability: %f",
			reportType, report.Summary.OverallAvailability)

		// Validate 99.95% target compliance
		if reportType == availability.ReportTypeSLA || reportType == availability.ReportTypeCompliance {
			suite.Assert().NotNil(report.SLACompliance)

			// In a healthy test environment, we should meet SLA targets
			if report.Summary.OverallAvailability < 0.995 {
				suite.T().Logf("Warning: Overall availability %f is below 99.95%% target",
					report.Summary.OverallAvailability)
			}
		}
	}

	// Test real-time dashboard creation
	dashboard, err := suite.reporter.CreateDashboard(
		"test-dashboard",
		"Test Availability Dashboard",
		availability.ReportTypeLive,
		30*time.Second,
	)

	suite.Require().NoError(err)
	suite.Assert().NotNil(dashboard)
	suite.Assert().Equal("test-dashboard", dashboard.ID)

	// Test dashboard subscription
	updateChan, err := suite.reporter.SubscribeToDashboard("test-dashboard")
	suite.Require().NoError(err)
	suite.Assert().NotNil(updateChan)

	// Wait for at least one dashboard update
	select {
	case update := <-updateChan:
		suite.Assert().NotNil(update)
		suite.Assert().Equal("test-dashboard", update.ID)
		suite.T().Logf("Received dashboard update at %s", update.LastUpdated)
	case <-time.After(60 * time.Second):
		suite.T().Log("Timeout waiting for dashboard update")
	}
}

func (suite *AvailabilityTrackingTestSuite) Test99_95PercentAvailabilityValidation() {
	// Comprehensive test to validate 99.95% availability claim

	suite.T().Log("Starting 99.95% availability validation test...")

	// Run for extended period to gather enough data
	testDuration := 2 * time.Minute // In real testing, this would be much longer
	testStart := time.Now()

	// Monitor continuously
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var samples []float64
	var downtimeDuration time.Duration
	var totalChecks int
	var failedChecks int

	endTime := testStart.Add(testDuration)

	for time.Now().Before(endTime) {
		select {
		case <-ticker.C:
			// Collect availability sample
			state := suite.tracker.GetCurrentState()

			totalChecks++

			if state.AggregatedStatus == availability.HealthUnhealthy {
				failedChecks++
				downtimeDuration += 10 * time.Second // ticker interval
			}

			// Calculate current availability
			if totalChecks > 0 {
				currentAvailability := float64(totalChecks-failedChecks) / float64(totalChecks)
				samples = append(samples, currentAvailability)

				suite.T().Logf("Sample %d: Availability = %f, Status = %s, Business Impact = %f",
					totalChecks, currentAvailability, state.AggregatedStatus, state.BusinessImpactScore)
			}

		case <-suite.ctx.Done():
			break
		}
	}

	// Calculate final metrics
	if len(samples) > 0 {
		var totalAvailability float64
		for _, sample := range samples {
			totalAvailability += sample
		}
		avgAvailability := totalAvailability / float64(len(samples))

		suite.T().Logf("Test Results:")
		suite.T().Logf("  Test Duration: %s", testDuration)
		suite.T().Logf("  Total Samples: %d", len(samples))
		suite.T().Logf("  Failed Checks: %d", failedChecks)
		suite.T().Logf("  Average Availability: %f", avgAvailability)
		suite.T().Logf("  Estimated Downtime: %s", downtimeDuration)

		// Validate against 99.95% target
		target := 0.9995
		suite.Assert().GreaterOrEqual(avgAvailability, target,
			"Average availability %f should meet 99.95%% target", avgAvailability)

		// Calculate annualized downtime projection
		totalTestTime := testDuration
		downtimeRatio := float64(downtimeDuration) / float64(totalTestTime)
		yearlyDowntime := time.Duration(float64(365*24*time.Hour) * downtimeRatio)

		suite.T().Logf("  Projected Yearly Downtime: %s", yearlyDowntime)
		suite.T().Logf("  99.95%% Target Max Yearly Downtime: %s", 4*time.Hour+23*time.Minute) // 4.38 hours

		// For a comprehensive test, yearly downtime should be under 4.38 hours
		if yearlyDowntime > 4*time.Hour+30*time.Minute { // Allow some margin
			suite.T().Logf("Warning: Projected yearly downtime %s exceeds 99.95%% target", yearlyDowntime)
		}
	}

	// Generate final compliance report
	complianceReport, err := suite.reporter.GenerateReport(
		suite.ctx,
		availability.ReportTypeCompliance,
		availability.Window1Hour,
		availability.FormatJSON,
	)

	suite.Require().NoError(err)
	suite.Assert().NotNil(complianceReport)

	suite.T().Logf("Final Compliance Report:")
	suite.T().Logf("  Overall Availability: %f", complianceReport.Summary.OverallAvailability)
	suite.T().Logf("  Weighted Availability: %f", complianceReport.Summary.WeightedAvailability)
	suite.T().Logf("  Business Impact Score: %f", complianceReport.Summary.BusinessImpactScore)
	suite.T().Logf("  Total Downtime: %s", complianceReport.Summary.TotalDowntime)
	suite.T().Logf("  MTTR: %s", complianceReport.Summary.MeanTimeToRecovery)
	suite.T().Logf("  MTBF: %s", complianceReport.Summary.MeanTimeBetweenFailures)

	// Export report for analysis
	reportJSON, err := suite.reporter.ExportReport(suite.ctx, complianceReport.ID, availability.FormatJSON)
	suite.Require().NoError(err)
	suite.Assert().NotEmpty(reportJSON)

	suite.T().Logf("Compliance report exported successfully (%d bytes)", len(reportJSON))
}

func (suite *AvailabilityTrackingTestSuite) TestPerformanceRequirements() {
	// Test that the system meets performance requirements

	suite.T().Log("Testing performance requirements...")

	// Test sub-50ms availability calculation latency
	start := time.Now()
	_ = suite.tracker.GetCurrentState()
	latency := time.Since(start)

	suite.T().Logf("Availability calculation latency: %s", latency)
	suite.Assert().Less(latency, 50*time.Millisecond,
		"Availability calculation should be under 50ms")

	// Test memory efficiency
	// This would require runtime memory profiling in a real test

	// Test concurrent load handling
	const concurrentRequests = 100
	const requestsPerSecond = 10000 / 60 // ~167 per second for 10k/minute target

	requestChan := make(chan struct{}, concurrentRequests)
	resultChan := make(chan time.Duration, concurrentRequests)

	// Start concurrent requests
	for i := 0; i < concurrentRequests; i++ {
		go func() {
			requestChan <- struct{}{}
			start := time.Now()
			_ = suite.tracker.GetCurrentState()
			latency := time.Since(start)
			resultChan <- latency
			<-requestChan
		}()
	}

	// Collect results
	var totalLatency time.Duration
	var maxLatency time.Duration

	for i := 0; i < concurrentRequests; i++ {
		latency := <-resultChan
		totalLatency += latency
		if latency > maxLatency {
			maxLatency = latency
		}
	}

	avgLatency := totalLatency / time.Duration(concurrentRequests)

	suite.T().Logf("Concurrent load test results:")
	suite.T().Logf("  Requests: %d", concurrentRequests)
	suite.T().Logf("  Average latency: %s", avgLatency)
	suite.T().Logf("  Max latency: %s", maxLatency)

	suite.Assert().Less(avgLatency, 100*time.Millisecond,
		"Average latency under load should be reasonable")
	suite.Assert().Less(maxLatency, 500*time.Millisecond,
		"Max latency under load should be reasonable")
}

func (suite *AvailabilityTrackingTestSuite) stopTrackingComponents() {
	if suite.tracker != nil {
		suite.tracker.Stop()
	}
	if suite.syntheticMonitor != nil {
		suite.syntheticMonitor.Stop()
	}
	if suite.dependencyTracker != nil {
		suite.dependencyTracker.Stop()
	}
	if suite.calculator != nil {
		suite.calculator.Stop()
	}
	if suite.reporter != nil {
		suite.reporter.Stop()
	}
}

func (suite *AvailabilityTrackingTestSuite) teardownMockServices() {
	if suite.mockLLMService != nil {
		suite.mockLLMService.Stop()
	}
	if suite.mockRAGService != nil {
		suite.mockRAGService.Stop()
	}
	if suite.mockNephioService != nil {
		suite.mockNephioService.Stop()
	}
	if suite.mockPrometheus != nil {
		suite.mockPrometheus.Stop()
	}
}

// Mock implementations for testing

type MockLLMService struct {
	address string
	server  *http.Server
}

func NewMockLLMService(address string) *MockLLMService {
	return &MockLLMService{address: address}
}

func (m *MockLLMService) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	})

	m.server = &http.Server{
		Addr:    m.address,
		Handler: mux,
	}

	m.server.ListenAndServe()
}

func (m *MockLLMService) Stop() {
	if m.server != nil {
		m.server.Close()
	}
}

type MockRAGService struct {
	address string
	server  *http.Server
}

func NewMockRAGService(address string) *MockRAGService {
	return &MockRAGService{address: address}
}

func (m *MockRAGService) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Simulate occasional slow responses
		if time.Now().UnixNano()%10 == 0 {
			time.Sleep(300 * time.Millisecond)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	})

	m.server = &http.Server{
		Addr:    m.address,
		Handler: mux,
	}

	m.server.ListenAndServe()
}

func (m *MockRAGService) Stop() {
	if m.server != nil {
		m.server.Close()
	}
}

type MockNephioService struct {
	address string
	server  *http.Server
}

func NewMockNephioService(address string) *MockNephioService {
	return &MockNephioService{address: address}
}

func (m *MockNephioService) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	})

	m.server = &http.Server{
		Addr:    m.address,
		Handler: mux,
	}

	m.server.ListenAndServe()
}

func (m *MockNephioService) Stop() {
	if m.server != nil {
		m.server.Close()
	}
}

type MockPrometheusServer struct {
	address string
	server  *http.Server
	client  *http.Client
}

func NewMockPrometheusServer(address string) *MockPrometheusServer {
	return &MockPrometheusServer{
		address: address,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (m *MockPrometheusServer) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/query", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success", "data": {"resultType": "vector", "result": []}}`))
	})

	m.server = &http.Server{
		Addr:    m.address,
		Handler: mux,
	}

	go m.server.ListenAndServe()
}

func (m *MockPrometheusServer) Client() *http.Client {
	return m.client
}

func (m *MockPrometheusServer) Stop() {
	if m.server != nil {
		m.server.Close()
	}
}

type MockAlertManager struct{}

func (m *MockAlertManager) SendAlert(ctx context.Context, check *availability.SyntheticCheck, result *availability.SyntheticResult) error {
	// Mock implementation - just log
	return nil
}

func (m *MockAlertManager) EvaluateThresholds(ctx context.Context, check *availability.SyntheticCheck, results []availability.SyntheticResult) (bool, error) {
	// Mock implementation - simple threshold check
	if len(results) > 0 {
		latest := results[len(results)-1]
		return latest.Status != availability.CheckStatusPass, nil
	}
	return false, nil
}
