package automation

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"k8s.io/client-go/kubernetes/fake"
)

// Additional test types (non-conflicting)
type AnomalyModel struct {
	Name      string
	Algorithm string
	Accuracy  float64
}

func TestNewHealthMonitor_Success(t *testing.T) {
	config := &SelfHealingConfig{
		Enabled:                   true,
		MonitoringInterval:        30 * time.Second,
		HealthCheckTimeout:        10 * time.Second,
		FailureDetectionThreshold: 0.8,
		ComponentConfigs: map[string]*ComponentConfig{
			"test-component": {
				Name:                "test-component",
				HealthCheckEndpoint: "/health",
				CriticalityLevel:    "HIGH",
				AutoHealingEnabled:  true,
			},
		},
	}

	k8sClient := fake.NewSimpleClientset()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	monitor, err := NewHealthMonitor(config, k8sClient, logger)

	assert.NoError(t, err)
	assert.NotNil(t, monitor)
	assert.Equal(t, config, monitor.config)
	assert.Equal(t, k8sClient, monitor.k8sClient)
	assert.Equal(t, logger, monitor.logger)
	assert.NotNil(t, monitor.healthCheckers)
	assert.NotNil(t, monitor.systemMetrics)
}

func TestNewHealthMonitor_NilConfig(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	monitor, err := NewHealthMonitor(nil, k8sClient, logger)

	assert.Error(t, err)
	assert.Nil(t, monitor)
	assert.Contains(t, err.Error(), "configuration is required")
}

func TestComponentHealthChecker_Structure(t *testing.T) {
	component := &ComponentConfig{
		Name:                "api-service",
		HealthCheckEndpoint: "/api/health",
		CriticalityLevel:    "HIGH",
		AutoHealingEnabled:  true,
		MaxRestartAttempts:  3,
		RestartCooldown:     5 * time.Minute,
	}

	checker := &ComponentHealthChecker{
		component:           component,
		lastCheck:           time.Now(),
		consecutiveFailures: 0,
		currentStatus:       HealthStatusHealthy,
		metrics: &ComponentMetrics{
			ResponseTime: 150 * time.Millisecond,
			ErrorRate:    0.01,
			Throughput:   1000.0,
			CPUUsage:     45.5,
			MemoryUsage:  60.2,
			RestartCount: 2,
		},
		restartHistory: []*RestartEvent{
			{
				Timestamp: time.Now().Add(-1 * time.Hour),
				Reason:    "Health check failure",
				Success:   true,
				Duration:  30 * time.Second,
			},
		},
	}

	// Verify checker structure
	assert.Equal(t, component, checker.component)
	assert.Equal(t, 0, checker.consecutiveFailures)
	assert.Equal(t, HealthStatusHealthy, checker.currentStatus)
	assert.NotNil(t, checker.metrics)
	assert.Len(t, checker.restartHistory, 1)

	// Verify metrics
	assert.Equal(t, 150*time.Millisecond, checker.metrics.ResponseTime)
	assert.Equal(t, 0.01, checker.metrics.ErrorRate)
	assert.Equal(t, 1000.0, checker.metrics.Throughput)
	assert.Equal(t, 45.5, checker.metrics.CPUUsage)
	assert.Equal(t, 60.2, checker.metrics.MemoryUsage)
	assert.Equal(t, 2, checker.metrics.RestartCount)

	// Verify restart history
	restart := checker.restartHistory[0]
	assert.Equal(t, "Health check failure", restart.Reason)
	assert.True(t, restart.Success)
	assert.Equal(t, 30*time.Second, restart.Duration)
}

func TestHealthStatusValues(t *testing.T) {
	testCases := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthStatusHealthy, "HEALTHY"},
		{HealthStatusDegraded, "DEGRADED"},
		{HealthStatusUnhealthy, "UNHEALTHY"},
		{HealthStatusCritical, "CRITICAL"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.status), func(t *testing.T) {
			assert.Equal(t, tc.expected, string(tc.status))
		})
	}
}

func TestSystemHealthMetrics_Initialization(t *testing.T) {
	metrics := &SystemHealthMetrics{
		OverallHealth:       HealthStatusHealthy,
		ComponentHealth:     make(map[string]HealthStatus),
		ActiveIncidents:     0,
		ResolvedIncidents:   0,
		PredictedFailures:   make(map[string]float64),
		SystemLoad:          0.0,
		ResourceUtilization: make(map[string]float64),
		PerformanceMetrics:  make(map[string]float64),
	}

	assert.Equal(t, HealthStatusHealthy, metrics.OverallHealth)
	assert.NotNil(t, metrics.ComponentHealth)
	assert.NotNil(t, metrics.PredictedFailures)
	assert.NotNil(t, metrics.ResourceUtilization)
	assert.NotNil(t, metrics.PerformanceMetrics)
	assert.Equal(t, 0, metrics.ActiveIncidents)
	assert.Equal(t, 0, metrics.ResolvedIncidents)
	assert.Equal(t, 0.0, metrics.SystemLoad)
}

func TestComponentMetrics_Structure(t *testing.T) {
	now := time.Now()
	metrics := &ComponentMetrics{
		ResponseTime: 250 * time.Millisecond,
		ErrorRate:    0.02,
		Throughput:   500.0,
		CPUUsage:     75.5,
		MemoryUsage:  80.2,
		RestartCount: 5,
		LastRestart:  &now,
	}

	assert.Equal(t, 250*time.Millisecond, metrics.ResponseTime)
	assert.Equal(t, 0.02, metrics.ErrorRate)
	assert.Equal(t, 500.0, metrics.Throughput)
	assert.Equal(t, 75.5, metrics.CPUUsage)
	assert.Equal(t, 80.2, metrics.MemoryUsage)
	assert.Equal(t, 5, metrics.RestartCount)
	assert.Equal(t, &now, metrics.LastRestart)
}

func TestRestartEvent_Structure(t *testing.T) {
	timestamp := time.Now()
	event := &RestartEvent{
		Timestamp: timestamp,
		Reason:    "Memory leak detected",
		Success:   true,
		Duration:  45 * time.Second,
	}

	assert.Equal(t, timestamp, event.Timestamp)
	assert.Equal(t, "Memory leak detected", event.Reason)
	assert.True(t, event.Success)
	assert.Equal(t, 45*time.Second, event.Duration)
}

func TestSelfHealingMetrics_Structure(t *testing.T) {
	metrics := &SelfHealingMetrics{
		TotalHealingOperations:      100,
		SuccessfulHealingOperations: 85,
		FailedHealingOperations:     15,
		AverageHealingTime:          2 * time.Minute,
		ComponentAvailability: map[string]float64{
			"api":      99.9,
			"database": 99.5,
			"cache":    99.99,
		},
		MTTR: 5 * time.Minute,
		MTBF: 24 * time.Hour,
	}

	assert.Equal(t, int64(100), metrics.TotalHealingOperations)
	assert.Equal(t, int64(85), metrics.SuccessfulHealingOperations)
	assert.Equal(t, int64(15), metrics.FailedHealingOperations)
	assert.Equal(t, 2*time.Minute, metrics.AverageHealingTime)
	assert.Equal(t, 99.9, metrics.ComponentAvailability["api"])
	assert.Equal(t, 5*time.Minute, metrics.MTTR)
	assert.Equal(t, 24*time.Hour, metrics.MTBF)
}

// Test failure prediction components
func TestFailurePrediction_NewFailurePrediction(t *testing.T) {
	config := &SelfHealingConfig{
		Enabled:                   true,
		PredictiveAnalysisEnabled: true,
		ComponentConfigs: map[string]*ComponentConfig{
			"test-component": {
				Name:             "test-component",
				CriticalityLevel: "HIGH",
			},
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	fp, err := NewFailurePrediction(config, logger)

	assert.NoError(t, err)
	assert.NotNil(t, fp)
	assert.Equal(t, config, fp.config)
	assert.Equal(t, logger, fp.logger)
	assert.NotNil(t, fp.predictionModels)
	assert.NotNil(t, fp.failureProbabilities)
	assert.NotNil(t, fp.predictionAccuracy)
	assert.NotNil(t, fp.historicalData)
	assert.NotNil(t, fp.anomalyDetector)

	// Check that models were created for components
	assert.Contains(t, fp.predictionModels, "test-component")
	model := fp.predictionModels["test-component"]
	assert.Equal(t, "test-component-predictor", model.Name)
	assert.Equal(t, "test-component", model.Component)
	assert.Equal(t, "ARIMA", model.ModelType)
	assert.Contains(t, model.Features, "cpu_usage")
	assert.Contains(t, model.Features, "memory_usage")
	assert.Contains(t, model.Features, "error_rate")
	assert.Equal(t, 0.75, model.Accuracy)
	assert.Equal(t, 30*time.Minute, model.PredictionWindow)
}

func TestFailurePrediction_NilConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	fp, err := NewFailurePrediction(nil, logger)

	assert.Error(t, err)
	assert.Nil(t, fp)
	assert.Contains(t, err.Error(), "configuration is required")
}

func TestPredictionModel_Structure(t *testing.T) {
	model := &PredictionModel{
		Name:             "api-service-predictor",
		Component:        "api-service",
		ModelType:        "NEURAL_NETWORK",
		Features:         []string{"cpu_usage", "memory_usage", "error_rate", "response_time"},
		Accuracy:         0.85,
		LastTraining:     time.Now().Add(-12 * time.Hour),
		PredictionWindow: 1 * time.Hour,
		Thresholds: map[string]float64{
			"failure_probability": 0.9,
			"anomaly_score":       0.8,
			"degradation_rate":    0.7,
		},
	}

	assert.Equal(t, "api-service-predictor", model.Name)
	assert.Equal(t, "api-service", model.Component)
	assert.Equal(t, "NEURAL_NETWORK", model.ModelType)
	assert.Contains(t, model.Features, "cpu_usage")
	assert.Contains(t, model.Features, "memory_usage")
	assert.Contains(t, model.Features, "error_rate")
	assert.Contains(t, model.Features, "response_time")
	assert.Equal(t, 0.85, model.Accuracy)
	assert.Equal(t, 1*time.Hour, model.PredictionWindow)
	assert.Equal(t, 0.9, model.Thresholds["failure_probability"])
	assert.Equal(t, 0.8, model.Thresholds["anomaly_score"])
	assert.Equal(t, 0.7, model.Thresholds["degradation_rate"])
}

// Test remediation components
func TestRemediationStrategy_Structure(t *testing.T) {
	strategy := &RemediationStrategy{
		Name: "scale-and-restart",
		Conditions: []*RemediationCondition{
			{
				Metric:    "cpu_usage",
				Operator:  "GT",
				Threshold: 80.0,
				Duration:  5 * time.Minute,
			},
			{
				Metric:    "error_rate",
				Operator:  "GT",
				Threshold: 0.05,
				Duration:  2 * time.Minute,
			},
		},
		Actions: []*RemediationActionTemplate{
			{
				Type:     "SCALE",
				Template: "scale-deployment",
				Parameters: map[string]interface{}{
					"replicas": 5,
					"max_replicas": 10,
				},
				Timeout: 10 * time.Minute,
				RetryPolicy: &RetryPolicy{
					MaxAttempts:       3,
					InitialDelay:      30 * time.Second,
					MaxDelay:          5 * time.Minute,
					BackoffMultiplier: 2.0,
				},
			},
			{
				Type:     "RESTART",
				Template: "restart-deployment",
				Parameters: map[string]interface{}{
					"graceful": true,
					"timeout": "60s",
				},
				Timeout: 5 * time.Minute,
			},
		},
		Priority:    1,
		Success:     15,
		Total:       20,
		SuccessRate: 0.75,
	}

	assert.Equal(t, "scale-and-restart", strategy.Name)
	assert.Len(t, strategy.Conditions, 2)
	assert.Len(t, strategy.Actions, 2)
	assert.Equal(t, 1, strategy.Priority)
	assert.Equal(t, 15, strategy.Success)
	assert.Equal(t, 20, strategy.Total)
	assert.Equal(t, 0.75, strategy.SuccessRate)

	// Check conditions
	cpuCondition := strategy.Conditions[0]
	assert.Equal(t, "cpu_usage", cpuCondition.Metric)
	assert.Equal(t, "GT", cpuCondition.Operator)
	assert.Equal(t, 80.0, cpuCondition.Threshold)
	assert.Equal(t, 5*time.Minute, cpuCondition.Duration)

	errorCondition := strategy.Conditions[1]
	assert.Equal(t, "error_rate", errorCondition.Metric)
	assert.Equal(t, "GT", errorCondition.Operator)
	assert.Equal(t, 0.05, errorCondition.Threshold)
	assert.Equal(t, 2*time.Minute, errorCondition.Duration)

	// Check actions
	scaleAction := strategy.Actions[0]
	assert.Equal(t, "SCALE", scaleAction.Type)
	assert.Equal(t, "scale-deployment", scaleAction.Template)
	assert.Equal(t, 5, scaleAction.Parameters["replicas"])
	assert.Equal(t, 10*time.Minute, scaleAction.Timeout)
	assert.NotNil(t, scaleAction.RetryPolicy)

	restartAction := strategy.Actions[1]
	assert.Equal(t, "RESTART", restartAction.Type)
	assert.Equal(t, "restart-deployment", restartAction.Template)
	assert.Equal(t, true, restartAction.Parameters["graceful"])
	assert.Equal(t, 5*time.Minute, restartAction.Timeout)
}

func TestRemediationCondition_Structure(t *testing.T) {
	condition := &RemediationCondition{
		Metric:    "memory_usage",
		Operator:  "LT",
		Threshold: 90.0,
		Duration:  3 * time.Minute,
	}

	assert.Equal(t, "memory_usage", condition.Metric)
	assert.Equal(t, "LT", condition.Operator)
	assert.Equal(t, 90.0, condition.Threshold)
	assert.Equal(t, 3*time.Minute, condition.Duration)
}

func TestRetryPolicy_Structure(t *testing.T) {
	policy := &RetryPolicy{
		MaxAttempts:       5,
		InitialDelay:      15 * time.Second,
		MaxDelay:          10 * time.Minute,
		BackoffMultiplier: 1.5,
	}

	assert.Equal(t, 5, policy.MaxAttempts)
	assert.Equal(t, 15*time.Second, policy.InitialDelay)
	assert.Equal(t, 10*time.Minute, policy.MaxDelay)
	assert.Equal(t, 1.5, policy.BackoffMultiplier)
}

func TestResourceLimits_Structure(t *testing.T) {
	limits := &ResourceLimits{
		MaxCPU:    "4000m",
		MaxMemory: "8Gi",
		MaxDisk:   "50Gi",
	}

	assert.Equal(t, "4000m", limits.MaxCPU)
	assert.Equal(t, "8Gi", limits.MaxMemory)
	assert.Equal(t, "50Gi", limits.MaxDisk)
}

func TestPerformanceThresholds_Structure(t *testing.T) {
	thresholds := &PerformanceThresholds{
		MaxLatency:    500 * time.Millisecond,
		MaxErrorRate:  0.03,
		MinThroughput: 2000.0,
		MaxQueueDepth: 10000,
	}

	assert.Equal(t, 500*time.Millisecond, thresholds.MaxLatency)
	assert.Equal(t, 0.03, thresholds.MaxErrorRate)
	assert.Equal(t, 2000.0, thresholds.MinThroughput)
	assert.Equal(t, int64(10000), thresholds.MaxQueueDepth)
}

// Test notification configuration
func TestNotificationConfig_Structure(t *testing.T) {
	config := &NotificationConfig{
		Enabled:  true,
		Webhooks: []string{"https://hooks.slack.com/webhook1", "https://teams.webhook.com/webhook2"},
		Channels: []string{"slack", "email", "sms"},
		Escalation: &EscalationConfig{
			Levels: []EscalationLevel{
				{
					Threshold: 5 * time.Minute,
					Contacts:  []string{"oncall@company.com"},
				},
				{
					Threshold: 15 * time.Minute,
					Contacts:  []string{"manager@company.com", "cto@company.com"},
				},
			},
		},
	}

	assert.True(t, config.Enabled)
	assert.Len(t, config.Webhooks, 2)
	assert.Contains(t, config.Webhooks, "https://hooks.slack.com/webhook1")
	assert.Len(t, config.Channels, 3)
	assert.Contains(t, config.Channels, "slack")
	assert.Contains(t, config.Channels, "email")
	assert.Contains(t, config.Channels, "sms")
	assert.NotNil(t, config.Escalation)
	assert.Len(t, config.Escalation.Levels, 2)

	level1 := config.Escalation.Levels[0]
	assert.Equal(t, 5*time.Minute, level1.Threshold)
	assert.Equal(t, []string{"oncall@company.com"}, level1.Contacts)

	level2 := config.Escalation.Levels[1]
	assert.Equal(t, 15*time.Minute, level2.Threshold)
	assert.Contains(t, level2.Contacts, "manager@company.com")
	assert.Contains(t, level2.Contacts, "cto@company.com")
}

// Test data structures
func TestHistoricalDataStore_Structure(t *testing.T) {
	store := NewHistoricalDataStore()

	assert.NotNil(t, store)

	// Test data retrieval
	recentData := store.GetRecentData("api-service", 1*time.Hour)
	assert.NotNil(t, recentData)
	assert.Empty(t, recentData) // No data added yet
}

func TestAnomalyDetector_Structure(t *testing.T) {
	detector := NewAnomalyDetector()

	assert.NotNil(t, detector)

	// Test basic functionality
	ctx := context.Background()
	detector.Start(ctx)

	// Test model structure
	model := &AnomalyModel{
		Name:      "cpu-anomaly-detector",
		Algorithm: "ISOLATION_FOREST",
		Accuracy:  0.85,
	}

	assert.Equal(t, "cpu-anomaly-detector", model.Name)
	assert.Equal(t, "ISOLATION_FOREST", model.Algorithm)
	assert.Equal(t, 0.85, model.Accuracy)
}

// Benchmark tests
func BenchmarkComponentHealthChecker_Creation(b *testing.B) {
	component := &ComponentConfig{
		Name:                "benchmark-component",
		HealthCheckEndpoint: "/health",
		CriticalityLevel:    "HIGH",
		AutoHealingEnabled:  true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker := &ComponentHealthChecker{
			component:           component,
			lastCheck:           time.Now(),
			consecutiveFailures: 0,
			currentStatus:       HealthStatusHealthy,
			metrics: &ComponentMetrics{
				ResponseTime: 100 * time.Millisecond,
				ErrorRate:    0.01,
				Throughput:   1000.0,
			},
			restartHistory: make([]*RestartEvent, 0),
		}
		_ = checker
	}
}

func BenchmarkSystemHealthMetrics_Update(b *testing.B) {
	metrics := &SystemHealthMetrics{
		ComponentHealth:     make(map[string]HealthStatus),
		PredictedFailures:   make(map[string]float64),
		ResourceUtilization: make(map[string]float64),
		PerformanceMetrics:  make(map[string]float64),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.ComponentHealth["component1"] = HealthStatusHealthy
		metrics.PredictedFailures["component1"] = 0.1
		metrics.ResourceUtilization["cpu"] = 50.0
		metrics.PerformanceMetrics["latency"] = 100.0
	}
}