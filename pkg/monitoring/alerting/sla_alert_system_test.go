package alerting

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// TestSLAAlertingSystemIntegration demonstrates the complete SLA alerting workflow
// DISABLED: func TestSLAAlertingSystemIntegration(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewLogger("test", "debug")

	// Initialize the complete SLA alerting system
	alertManager, err := setupSLAAlertingSystem(t, logger)
	require.NoError(t, err)
	require.NotNil(t, alertManager)

	// Start all components
	err = alertManager.Start(ctx)
	require.NoError(t, err)
	defer func() {
		err := alertManager.Stop(ctx)
		assert.NoError(t, err)
	}()

	t.Run("AvailabilityViolationDetection", func(t *testing.T) {
		testAvailabilityViolationDetection(t, ctx, alertManager)
	})

	t.Run("LatencyViolationPrediction", func(t *testing.T) {
		testLatencyViolationPrediction(t, ctx, alertManager)
	})

	t.Run("MultiWindowBurnRateAlerting", func(t *testing.T) {
		testMultiWindowBurnRateAlerting(t, ctx, alertManager)
	})

	t.Run("AlertDeduplicationAndCorrelation", func(t *testing.T) {
		testAlertDeduplicationAndCorrelation(t, ctx, alertManager)
	})

	t.Run("EscalationWorkflow", func(t *testing.T) {
		testEscalationWorkflow(t, ctx, alertManager)
	})
}

// setupSLAAlertingSystem creates a complete SLA alerting system for testing
func setupSLAAlertingSystem(t *testing.T, logger *logging.StructuredLogger) (*SLAAlertManager, error) {
	// Configure for testing
	config := &SLAAlertConfig{
		EvaluationInterval:   1 * time.Second, // Fast evaluation for tests
		BurnRateInterval:     500 * time.Millisecond,
		PredictiveInterval:   2 * time.Second,
		MaxActiveAlerts:      100,
		DeduplicationWindow:  30 * time.Second,
		NotificationTimeout:  5 * time.Second,
		EnableBusinessImpact: true,
		MetricCacheSize:      50,
		QueryTimeout:         2 * time.Second,
	}

	return NewSLAAlertManager(config, logger)
}

// setupSLAAlertingSystemForBenchmark creates SLA alert manager for benchmark tests
func setupSLAAlertingSystemForBenchmark(logger *logging.StructuredLogger) (*SLAAlertManager, error) {
	// Configure for benchmarking
	config := &SLAAlertConfig{
		EvaluationInterval:   1 * time.Second, // Fast evaluation for benchmarks
		BurnRateInterval:     500 * time.Millisecond,
		PredictiveInterval:   2 * time.Second,
		MaxActiveAlerts:      100,
		DeduplicationWindow:  30 * time.Second,
		NotificationTimeout:  5 * time.Second,
		EnableBusinessImpact: true,
		MetricCacheSize:      50,
		QueryTimeout:         2 * time.Second,
	}

	return NewSLAAlertManager(config, logger)
}

// testAvailabilityViolationDetection tests availability SLA violation detection
func testAvailabilityViolationDetection(t *testing.T, ctx context.Context, sam *SLAAlertManager) {
	// Simulate availability drop below 99.95%
	// Expected metrics: 10000 total requests, 9990 success (99.9% success rate - below 99.95% target)
	// Expected error budget remaining: 20%
	_ = map[string]float64{
		"http_requests_total":    10000,
		"http_requests_success":  9990, // 99.9% success rate (below 99.95% target)
		"error_budget_remaining": 0.2,  // 20% error budget remaining
	}

	// Wait for evaluation cycle
	time.Sleep(2 * time.Second)

	// Check that alerts were generated
	activeAlerts := sam.GetActiveAlerts()

	// Should have availability alerts
	availabilityAlerts := filterAlertsByType(activeAlerts, SLATypeAvailability)
	assert.NotEmpty(t, availabilityAlerts, "Should generate availability violation alerts")

	// Verify alert properties
	if len(availabilityAlerts) > 0 {
		alert := availabilityAlerts[0]
		assert.Equal(t, SLATypeAvailability, alert.SLAType)
		assert.True(t, alert.CurrentValue < 99.95, "Current value should be below target")
		assert.True(t, alert.ErrorBudget.Remaining < 1.0, "Error budget should be consumed")
		assert.NotEmpty(t, alert.Context.RelatedMetrics, "Should include related metrics")
	}
}

// testLatencyViolationPrediction tests predictive latency violation detection
func testLatencyViolationPrediction(t *testing.T, ctx context.Context, sam *SLAAlertManager) {
	// Simulate increasing latency trend that will violate SLA
	predictiveAlerting := sam.predictiveAlerting

	currentMetrics := map[string]float64{
		"p95_latency_seconds":     1.8,  // Currently under 2s target
		"cpu_usage_percent":       85.0, // High CPU usage
		"memory_usage_percent":    78.0, // High memory usage
		"request_rate_per_second": 1500, // High load
		"error_rate_percent":      0.05, // Normal error rate
	}

	prediction, err := predictiveAlerting.Predict(ctx, SLATypeLatency, currentMetrics)
	require.NoError(t, err)
	require.NotNil(t, prediction)

	// Verify prediction quality
	assert.Equal(t, SLATypeLatency, prediction.SLAType)
	assert.True(t, prediction.Confidence >= 0.75, "Prediction confidence should be high")

	// If violation is predicted, verify early warning
	if prediction.ViolationProbability > 0.8 {
		assert.NotNil(t, prediction.TimeToViolation, "Should provide time to violation")
		assert.True(t, *prediction.TimeToViolation >= 15*time.Minute, "Should provide early warning")
		assert.NotEmpty(t, prediction.RecommendedActions, "Should provide recommended actions")

		// Check contributing factors
		assert.NotEmpty(t, prediction.ContributingFactors, "Should identify contributing factors")

		// Verify CPU usage is identified as a contributing factor
		hasCPUFactor := false
		for _, factor := range prediction.ContributingFactors {
			if factor.Feature == "cpu_usage" {
				hasCPUFactor = true
				assert.True(t, factor.Importance > 0.5, "CPU should be a significant factor")
				break
			}
		}
		assert.True(t, hasCPUFactor, "Should identify CPU usage as contributing factor")
	}
}

// testMultiWindowBurnRateAlerting tests Google SRE multi-window burn rate patterns
func testMultiWindowBurnRateAlerting(t *testing.T, ctx context.Context, sam *SLAAlertManager) {
	burnRateCalc := sam.burnRateCalculator

	// Test urgent alert conditions (short window + long window)
	burnRates, err := burnRateCalc.Calculate(ctx, SLATypeAvailability)
	require.NoError(t, err)

	// Verify burn rate structure
	assert.NotZero(t, burnRates.ShortWindow.Duration, "Should have short window")
	assert.NotZero(t, burnRates.MediumWindow.Duration, "Should have medium window")
	assert.NotZero(t, burnRates.LongWindow.Duration, "Should have long window")

	// Test burn rate thresholds
	config := DefaultBurnRateConfig()

	// Short window should have highest threshold (urgent alerts)
	assert.Equal(t, config.FastBurnThreshold, 14.4, "Fast burn threshold should be 14.4x")

	// Medium window should have medium threshold (critical alerts)
	assert.Equal(t, config.MediumBurnThreshold, 6.0, "Medium burn threshold should be 6x")

	// Long window should have lowest threshold (major alerts)
	assert.Equal(t, config.SlowBurnThreshold, 3.0, "Slow burn threshold should be 3x")

	// Simulate high burn rate scenario
	if burnRates.ShortWindow.BurnRate > config.FastBurnThreshold {
		assert.True(t, burnRates.ShortWindow.IsViolating, "Should trigger urgent alert")

		// Verify this would generate appropriate alert
		activeAlerts := sam.GetActiveAlerts()
		urgentAlerts := filterAlertsBySeverity(activeAlerts, AlertSeverityUrgent)

		if len(urgentAlerts) > 0 {
			alert := urgentAlerts[0]
			assert.True(t, alert.BurnRate.CurrentRate > config.FastBurnThreshold)
			assert.NotNil(t, alert.BurnRate.ShortWindow.IsViolating)
		}
	}
}

// testAlertDeduplicationAndCorrelation tests intelligent alert grouping
func testAlertDeduplicationAndCorrelation(t *testing.T, ctx context.Context, sam *SLAAlertManager) {
	alertRouter := sam.alertRouter

	// Create similar alerts that should be deduplicated
	baseAlert := &SLAAlert{
		ID:       "test-alert-1",
		SLAType:  SLATypeAvailability,
		Severity: AlertSeverityCritical,
		Context: AlertContext{
			Component:   "networkintent-controller",
			Service:     "nephoran-intent-operator",
			Region:      "us-east-1",
			Environment: "production",
		},
		Labels: map[string]string{
			"cluster": "main",
			"region":  "us-east-1",
		},
		StartsAt: time.Now(),
	}

	// Create duplicate alert
	duplicateAlert := &SLAAlert{
		ID:       "test-alert-2",
		SLAType:  SLATypeAvailability,
		Severity: AlertSeverityCritical,
		Context: AlertContext{
			Component:   "networkintent-controller",
			Service:     "nephoran-intent-operator",
			Region:      "us-east-1",
			Environment: "production",
		},
		Labels: map[string]string{
			"cluster": "main",
			"region":  "us-east-1",
		},
		StartsAt: time.Now(),
	}

	// Route both alerts
	err := alertRouter.Route(ctx, baseAlert)
	require.NoError(t, err)

	// Small delay to ensure processing
	time.Sleep(100 * time.Millisecond)

	err = alertRouter.Route(ctx, duplicateAlert)
	require.NoError(t, err)

	// Wait for processing
	time.Sleep(500 * time.Millisecond)

	// Verify deduplication occurred
	stats := alertRouter.GetStats()
	assert.Greater(t, stats.AlertsDeduped, int64(0), "Should have deduplicated similar alerts")

	// Check that only one alert group exists for this fingerprint
	fingerprint := alertRouter.generateDeduplicationFingerprint(baseAlert)

	alertRouter.mu.RLock()
	alertGroup, exists := alertRouter.alertFingerprints[fingerprint]
	alertRouter.mu.RUnlock()

	assert.True(t, exists, "Alert group should exist")
	assert.Greater(t, len(alertGroup.Members), 1, "Alert group should contain multiple alerts")
}

// testEscalationWorkflow tests automated escalation policies
func testEscalationWorkflow(t *testing.T, ctx context.Context, sam *SLAAlertManager) {
	escalationEngine := sam.escalationEngine

	// Create high-severity alert for escalation
	criticalAlert := &SLAAlert{
		ID:       "escalation-test-alert",
		SLAType:  SLATypeAvailability,
		Severity: AlertSeverityUrgent,
		Context: AlertContext{
			Component:   "llm-processor",
			Service:     "nephoran-intent-operator",
			Environment: "production",
		},
		BusinessImpact: BusinessImpactInfo{
			Severity:       "high",
			AffectedUsers:  1000,
			RevenueImpact:  5000.0,
			SLABreach:      true,
			CustomerFacing: true,
			ServiceTier:    "critical",
		},
		StartsAt: time.Now(),
	}

	// Start escalation
	err := escalationEngine.StartEscalation(ctx, criticalAlert)
	require.NoError(t, err)

	// Wait for initial processing
	time.Sleep(1 * time.Second)

	// Verify escalation was created
	escalationEngine.mu.RLock()
	activeEscalations := len(escalationEngine.activeEscalations)
	escalationEngine.mu.RUnlock()

	assert.Greater(t, activeEscalations, 0, "Should have active escalations")

	// Find the escalation for our alert
	var testEscalation *ActiveEscalation
	escalationEngine.mu.RLock()
	for _, escalation := range escalationEngine.activeEscalations {
		if escalation.AlertID == criticalAlert.ID {
			testEscalation = escalation
			break
		}
	}
	escalationEngine.mu.RUnlock()

	require.NotNil(t, testEscalation, "Should find escalation for test alert")

	// Verify escalation properties
	assert.Equal(t, EscalationStateActive, testEscalation.State)
	assert.Equal(t, 0, testEscalation.CurrentLevel, "Should start at level 0")
	assert.True(t, testEscalation.Priority >= 4, "Urgent alert should have high priority")
	assert.True(t, testEscalation.BusinessImpact.OverallScore > 0.5, "Should have high business impact")

	// Verify business impact calculation
	assert.True(t, testEscalation.BusinessImpact.OverallScore > 0.0)

	// Check escalation statistics
	stats := escalationEngine.escalationStats
	assert.Greater(t, stats.TotalEscalations, int64(0), "Should track escalation count")
	assert.Greater(t, stats.EscalationsByLevel[0], int64(0), "Should track level 0 escalations")
}

// TestSLAMetricsAndObservability tests metrics and monitoring capabilities
// DISABLED: func TestSLAMetricsAndObservability(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewLogger("test", "debug")

	alertManager, err := setupSLAAlertingSystem(t, logger)
	require.NoError(t, err)

	err = alertManager.Start(ctx)
	require.NoError(t, err)
	defer func() {
		err := alertManager.Stop(ctx)
		assert.NoError(t, err)
	}()

	// Wait for metrics to be generated
	time.Sleep(2 * time.Second)

	// Test SLA compliance metrics
	metrics := alertManager.metrics
	assert.NotNil(t, metrics.SLACompliance, "Should have SLA compliance metrics")
	assert.NotNil(t, metrics.ErrorBudgetBurn, "Should have error budget burn metrics")
	assert.NotNil(t, metrics.AlertsGenerated, "Should have alert generation metrics")

	// Test burn rate calculator metrics
	burnRateStats := alertManager.burnRateCalculator.GetStats()
	assert.Contains(t, burnRateStats, "calculation_count")
	assert.Contains(t, burnRateStats, "cache_size")
	assert.Contains(t, burnRateStats, "cache_hit_rate")

	// Test service-level metrics
	serviceStats := alertManager.GetMetrics()
	assert.GreaterOrEqual(t, serviceStats.ProcessingRate, 0.0)
	assert.GreaterOrEqual(t, serviceStats.MemoryUsageMB, 0.0)
	assert.GreaterOrEqual(t, serviceStats.CPUUsagePercent, 0.0)
}

// TestSLAAlertingConfiguration tests configuration management
// DISABLED: func TestSLAAlertingConfiguration(t *testing.T) {
	// Test default configuration
	config := DefaultSLAAlertConfig()
	assert.Equal(t, 30*time.Second, config.EvaluationInterval)
	assert.Equal(t, 1000, config.MaxActiveAlerts)
	assert.True(t, config.EnableBusinessImpact)

	// Test SLA targets
	targets := DefaultSLATargets()

	// Verify availability target
	availabilityTarget := targets[SLATypeAvailability]
	assert.Equal(t, 99.95, availabilityTarget.Target)
	assert.True(t, availabilityTarget.CustomerFacing)
	assert.Equal(t, "critical", availabilityTarget.BusinessTier)
	assert.Len(t, availabilityTarget.Windows, 3) // urgent, critical, major

	// Verify latency target
	latencyTarget := targets[SLATypeLatency]
	assert.Equal(t, 2000.0, latencyTarget.Target) // 2 seconds in milliseconds
	assert.Equal(t, "high", latencyTarget.BusinessTier)

	// Verify throughput target
	throughputTarget := targets[SLAThroughput]
	assert.Equal(t, 45.0, throughputTarget.Target) // 45 intents per minute
	assert.False(t, throughputTarget.CustomerFacing)

	// Verify error rate target
	errorRateTarget := targets[SLAErrorRate]
	assert.Equal(t, 0.1, errorRateTarget.Target) // 0.1% error rate
	assert.True(t, errorRateTarget.CustomerFacing)
}

// Helper functions for testing

func filterAlertsByType(alerts []*SLAAlert, slaType SLAType) []*SLAAlert {
	var filtered []*SLAAlert
	for _, alert := range alerts {
		if alert.SLAType == slaType {
			filtered = append(filtered, alert)
		}
	}
	return filtered
}

func filterAlertsBySeverity(alerts []*SLAAlert, severity AlertSeverity) []*SLAAlert {
	var filtered []*SLAAlert
	for _, alert := range alerts {
		if alert.Severity == severity {
			filtered = append(filtered, alert)
		}
	}
	return filtered
}

// BenchmarkSLAAlertProcessing benchmarks alert processing performance
func BenchmarkSLAAlertProcessing(b *testing.B) {
	ctx := context.Background()
	logger := logging.NewLogger("benchmark", "info")

	alertManager, err := setupSLAAlertingSystemForBenchmark(logger)
	require.NoError(b, err)

	err = alertManager.Start(ctx)
	require.NoError(b, err)
	defer func() {
		err := alertManager.Stop(ctx)
		assert.NoError(b, err)
	}()

	// Create test alert
	testAlert := &SLAAlert{
		SLAType:  SLATypeAvailability,
		Severity: AlertSeverityCritical,
		Context: AlertContext{
			Component: "test-component",
			Service:   "test-service",
		},
		StartsAt: time.Now(),
	}

	b.ResetTimer()

	// Benchmark alert routing
	b.Run("AlertRouting", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			testAlert.ID = fmt.Sprintf("bench-alert-%d", i)
			err := alertManager.alertRouter.Route(ctx, testAlert)
			require.NoError(b, err)
		}
	})

	// Benchmark burn rate calculation
	b.Run("BurnRateCalculation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := alertManager.burnRateCalculator.Calculate(ctx, SLATypeAvailability)
			require.NoError(b, err)
		}
	})

	// Benchmark prediction
	currentMetrics := map[string]float64{
		"cpu_usage":    50.0,
		"memory_usage": 60.0,
		"request_rate": 1000.0,
	}

	b.Run("PredictiveAlerting", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := alertManager.predictiveAlerting.Predict(ctx, SLATypeLatency, currentMetrics)
			require.NoError(b, err)
		}
	})
}
