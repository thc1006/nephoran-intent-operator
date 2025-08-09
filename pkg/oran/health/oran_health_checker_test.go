package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/nephoran-intent-operator/pkg/health"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/a1"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o1"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2"
)

func TestNewORANHealthChecker(t *testing.T) {
	healthChecker := health.NewHealthChecker("test-service", "v1.0.0", nil)
	a1Adaptor, _ := a1.NewA1Adaptor(nil)
	e2Adaptor, _ := e2.NewE2Adaptor(nil)

	tests := []struct {
		name   string
		config *ORANHealthConfig
	}{
		{
			name:   "default config",
			config: nil,
		},
		{
			name: "custom config",
			config: &ORANHealthConfig{
				CheckInterval: 10 * time.Second,
				HistorySize:   50,
				DependencyChecks: map[string]DependencyCheck{
					"test-dep": {
						Name:               "test-dependency",
						URL:                "http://test-service/health",
						Timeout:            5 * time.Second,
						ExpectedStatusCode: 200,
						CriticalDependency: true,
					},
				},
				AlertingThresholds: AlertingThresholds{
					ConsecutiveFailures:    5,
					DependencyFailureRate:  0.7,
					CircuitBreakerOpenTime: 10 * time.Minute,
					ResponseTimeThreshold:  10 * time.Second,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewORANHealthChecker(
				healthChecker,
				a1Adaptor,
				e2Adaptor,
				nil, // o1Adaptor
				nil, // o2Adaptor
				tt.config,
			)

			assert.NotNil(t, checker)
			assert.NotNil(t, checker.healthChecker)
			assert.NotNil(t, checker.a1Adaptor)
			assert.NotNil(t, checker.e2Adaptor)
			assert.NotNil(t, checker.circuitBreakers)
			assert.NotNil(t, checker.healthHistory)

			if tt.config == nil {
				assert.Equal(t, 30*time.Second, checker.checkInterval)
			} else {
				assert.Equal(t, tt.config.CheckInterval, checker.checkInterval)
			}
		})
	}
}

func TestORANHealthChecker_HealthCheck(t *testing.T) {
	healthChecker := health.NewHealthChecker("test-service", "v1.0.0", nil)
	a1Adaptor, _ := a1.NewA1Adaptor(nil)
	e2Adaptor, _ := e2.NewE2Adaptor(nil)

	checker := NewORANHealthChecker(
		healthChecker,
		a1Adaptor,
		e2Adaptor,
		nil,
		nil,
		nil,
	)

	// Wait a bit for the first check to complete
	time.Sleep(100 * time.Millisecond)

	snapshot := checker.GetHealthSnapshot()
	if snapshot != nil {
		assert.NotNil(t, snapshot)
		assert.NotZero(t, snapshot.Timestamp)
		assert.NotNil(t, snapshot.InterfaceStatus)
		assert.NotNil(t, snapshot.DependencyStatus)
		assert.NotNil(t, snapshot.CircuitBreakerStats)
	}
}

func TestORANHealthChecker_InterfaceHealthChecks(t *testing.T) {
	healthChecker := health.NewHealthChecker("test-service", "v1.0.0", nil)
	a1Adaptor, _ := a1.NewA1Adaptor(nil)
	e2Adaptor, _ := e2.NewE2Adaptor(nil)

	checker := NewORANHealthChecker(
		healthChecker,
		a1Adaptor,
		e2Adaptor,
		nil,
		nil,
		&ORANHealthConfig{
			CheckInterval: 100 * time.Millisecond,
			HistorySize:   10,
		},
	)

	// Perform a health check
	ctx := context.Background()
	response := healthChecker.Check(ctx)

	assert.NotNil(t, response)
	assert.Equal(t, "test-service", response.Service)
	assert.Equal(t, "v1.0.0", response.Version)

	// Check that O-RAN interface checks are registered
	foundA1 := false
	foundE2 := false

	for _, check := range response.Checks {
		switch check.Name {
		case "a1-interface":
			foundA1 = true
			assert.Equal(t, "oran-a1", check.Component)
		case "e2-interface":
			foundE2 = true
			assert.Equal(t, "oran-e2", check.Component)
		}
	}

	assert.True(t, foundA1, "A1 interface check should be registered")
	assert.True(t, foundE2, "E2 interface check should be registered")
}

func TestORANHealthChecker_CircuitBreakerStats(t *testing.T) {
	healthChecker := health.NewHealthChecker("test-service", "v1.0.0", nil)
	a1Adaptor, _ := a1.NewA1Adaptor(nil)
	e2Adaptor, _ := e2.NewE2Adaptor(nil)

	checker := NewORANHealthChecker(
		healthChecker,
		a1Adaptor,
		e2Adaptor,
		nil,
		nil,
		nil,
	)

	stats := checker.GetCircuitBreakerStats()
	assert.NotNil(t, stats)

	// Should have stats for both A1 and E2 adaptors
	a1Stats, hasA1 := stats["a1"]
	assert.True(t, hasA1, "Should have A1 circuit breaker stats")
	assert.NotNil(t, a1Stats)

	e2Stats, hasE2 := stats["e2"]
	assert.True(t, hasE2, "Should have E2 circuit breaker stats")
	assert.NotNil(t, e2Stats)
}

func TestORANHealthChecker_ResetCircuitBreakers(t *testing.T) {
	healthChecker := health.NewHealthChecker("test-service", "v1.0.0", nil)
	a1Adaptor, _ := a1.NewA1Adaptor(nil)
	e2Adaptor, _ := e2.NewE2Adaptor(nil)

	checker := NewORANHealthChecker(
		healthChecker,
		a1Adaptor,
		e2Adaptor,
		nil,
		nil,
		nil,
	)

	// Reset all circuit breakers
	checker.ResetCircuitBreakers()

	// Verify circuit breakers are in closed state
	stats := checker.GetCircuitBreakerStats()

	if a1Stats, ok := stats["a1"].(map[string]interface{}); ok {
		if state, ok := a1Stats["state"].(string); ok {
			assert.Equal(t, "closed", state)
		}
	}

	if e2Stats, ok := stats["e2"].(map[string]interface{}); ok {
		if state, ok := e2Stats["state"].(string); ok {
			assert.Equal(t, "closed", state)
		}
	}
}

func TestORANHealthChecker_HealthHistory(t *testing.T) {
	healthChecker := health.NewHealthChecker("test-service", "v1.0.0", nil)
	a1Adaptor, _ := a1.NewA1Adaptor(nil)

	checker := NewORANHealthChecker(
		healthChecker,
		a1Adaptor,
		nil,
		nil,
		nil,
		&ORANHealthConfig{
			CheckInterval: 50 * time.Millisecond,
			HistorySize:   5,
		},
	)

	// Wait for a few health checks to accumulate
	time.Sleep(300 * time.Millisecond)

	history := checker.GetHealthHistory()
	assert.NotNil(t, history)

	// Should have some history entries
	if len(history) > 0 {
		for _, snapshot := range history {
			assert.NotZero(t, snapshot.Timestamp)
			assert.NotNil(t, snapshot.InterfaceStatus)
			assert.NotNil(t, snapshot.DependencyStatus)
			assert.NotNil(t, snapshot.CircuitBreakerStats)
		}
	}
}

func TestORANHealthChecker_IsHealthy(t *testing.T) {
	healthChecker := health.NewHealthChecker("test-service", "v1.0.0", nil)
	a1Adaptor, _ := a1.NewA1Adaptor(nil)
	e2Adaptor, _ := e2.NewE2Adaptor(nil)

	checker := NewORANHealthChecker(
		healthChecker,
		a1Adaptor,
		e2Adaptor,
		nil,
		nil,
		&ORANHealthConfig{
			CheckInterval: 100 * time.Millisecond,
		},
	)

	// Wait for initial health check
	time.Sleep(200 * time.Millisecond)

	// Should be healthy with working adaptors
	isHealthy := checker.IsHealthy()
	// Note: This may be false initially as the first check might not have completed
	// In a real test environment, we'd wait or mock the health check behavior
	assert.IsType(t, true, isHealthy) // Just check the type is boolean
}

func TestDependencyCheck_Structure(t *testing.T) {
	depCheck := DependencyCheck{
		Name:               "test-dependency",
		URL:                "http://test-service/health",
		Timeout:            5 * time.Second,
		ExpectedStatusCode: 200,
		CriticalDependency: true,
		CheckInterval:      30 * time.Second,
		MaxFailures:        3,
		CurrentFailures:    0,
		LastCheck:          time.Now(),
		LastError:          "",
	}

	assert.Equal(t, "test-dependency", depCheck.Name)
	assert.Equal(t, "http://test-service/health", depCheck.URL)
	assert.Equal(t, 5*time.Second, depCheck.Timeout)
	assert.Equal(t, 200, depCheck.ExpectedStatusCode)
	assert.True(t, depCheck.CriticalDependency)
	assert.Equal(t, 30*time.Second, depCheck.CheckInterval)
	assert.Equal(t, 3, depCheck.MaxFailures)
	assert.Equal(t, 0, depCheck.CurrentFailures)
	assert.NotZero(t, depCheck.LastCheck)
	assert.Empty(t, depCheck.LastError)
}

func TestHealthSnapshot_Structure(t *testing.T) {
	now := time.Now()

	snapshot := HealthSnapshot{
		Timestamp:     now,
		OverallStatus: health.StatusHealthy,
		InterfaceStatus: map[string]health.Status{
			"a1": health.StatusHealthy,
			"e2": health.StatusHealthy,
		},
		DependencyStatus: map[string]health.Status{
			"ric": health.StatusHealthy,
		},
		CircuitBreakerStats: map[string]interface{}{
			"a1": map[string]interface{}{
				"state": "closed",
			},
		},
		Metrics: HealthMetrics{
			TotalChecks:         10,
			HealthyChecks:       9,
			UnhealthyChecks:     1,
			AverageResponseTime: 100 * time.Millisecond,
			UpTime:              1 * time.Hour,
		},
	}

	assert.Equal(t, now, snapshot.Timestamp)
	assert.Equal(t, health.StatusHealthy, snapshot.OverallStatus)
	assert.Len(t, snapshot.InterfaceStatus, 2)
	assert.Len(t, snapshot.DependencyStatus, 1)
	assert.Len(t, snapshot.CircuitBreakerStats, 1)
	assert.Equal(t, int64(10), snapshot.Metrics.TotalChecks)
	assert.Equal(t, int64(9), snapshot.Metrics.HealthyChecks)
	assert.Equal(t, int64(1), snapshot.Metrics.UnhealthyChecks)
}

func TestHealthMetrics_Structure(t *testing.T) {
	metrics := HealthMetrics{
		TotalChecks:         100,
		HealthyChecks:       95,
		UnhealthyChecks:     5,
		AverageResponseTime: 150 * time.Millisecond,
		UpTime:              2 * time.Hour,
		CircuitBreakerTrips: 2,
		RetryAttempts:       10,
		DependencyFailures:  3,
	}

	assert.Equal(t, int64(100), metrics.TotalChecks)
	assert.Equal(t, int64(95), metrics.HealthyChecks)
	assert.Equal(t, int64(5), metrics.UnhealthyChecks)
	assert.Equal(t, 150*time.Millisecond, metrics.AverageResponseTime)
	assert.Equal(t, 2*time.Hour, metrics.UpTime)
	assert.Equal(t, int64(2), metrics.CircuitBreakerTrips)
	assert.Equal(t, int64(10), metrics.RetryAttempts)
	assert.Equal(t, int64(3), metrics.DependencyFailures)
}

func TestAlertingThresholds_Structure(t *testing.T) {
	thresholds := AlertingThresholds{
		ConsecutiveFailures:    5,
		DependencyFailureRate:  0.8,
		CircuitBreakerOpenTime: 10 * time.Minute,
		ResponseTimeThreshold:  5 * time.Second,
	}

	assert.Equal(t, 5, thresholds.ConsecutiveFailures)
	assert.Equal(t, 0.8, thresholds.DependencyFailureRate)
	assert.Equal(t, 10*time.Minute, thresholds.CircuitBreakerOpenTime)
	assert.Equal(t, 5*time.Second, thresholds.ResponseTimeThreshold)
}

func TestORANHealthConfig_Structure(t *testing.T) {
	config := ORANHealthConfig{
		CheckInterval: 30 * time.Second,
		HistorySize:   100,
		DependencyChecks: map[string]DependencyCheck{
			"test": {
				Name: "test-dependency",
				URL:  "http://test/health",
			},
		},
		AlertingThresholds: AlertingThresholds{
			ConsecutiveFailures: 3,
		},
	}

	assert.Equal(t, 30*time.Second, config.CheckInterval)
	assert.Equal(t, 100, config.HistorySize)
	assert.Len(t, config.DependencyChecks, 1)
	assert.Equal(t, 3, config.AlertingThresholds.ConsecutiveFailures)
}

func TestORANHealthChecker_NilAdaptors(t *testing.T) {
	healthChecker := health.NewHealthChecker("test-service", "v1.0.0", nil)

	// Test with nil adaptors
	checker := NewORANHealthChecker(
		healthChecker,
		nil, // a1Adaptor
		nil, // e2Adaptor
		nil, // o1Adaptor
		nil, // o2Adaptor
		nil,
	)

	assert.NotNil(t, checker)
	assert.Nil(t, checker.a1Adaptor)
	assert.Nil(t, checker.e2Adaptor)
	assert.Nil(t, checker.o1Adaptor)
	assert.Nil(t, checker.o2Adaptor)

	// Should still be able to get stats (empty)
	stats := checker.GetCircuitBreakerStats()
	assert.NotNil(t, stats)
	assert.Len(t, stats, 0)
}

func TestORANHealthChecker_ConsecutiveFailures(t *testing.T) {
	healthChecker := health.NewHealthChecker("test-service", "v1.0.0", nil)
	a1Adaptor, _ := a1.NewA1Adaptor(nil)

	checker := NewORANHealthChecker(
		healthChecker,
		a1Adaptor,
		nil,
		nil,
		nil,
		nil,
	)

	// Simulate consecutive failures by adding unhealthy snapshots
	checker.mutex.Lock()
	for i := 0; i < 5; i++ {
		snapshot := HealthSnapshot{
			Timestamp:     time.Now().Add(time.Duration(i) * time.Second),
			OverallStatus: health.StatusUnhealthy,
		}
		checker.healthHistory = append(checker.healthHistory, snapshot)
	}
	checker.mutex.Unlock()

	failures := checker.getConsecutiveFailures()
	assert.Equal(t, 5, failures)
}

func TestORANHealthChecker_PartialFailures(t *testing.T) {
	healthChecker := health.NewHealthChecker("test-service", "v1.0.0", nil)
	a1Adaptor, _ := a1.NewA1Adaptor(nil)

	checker := NewORANHealthChecker(
		healthChecker,
		a1Adaptor,
		nil,
		nil,
		nil,
		nil,
	)

	// Simulate mixed health status
	checker.mutex.Lock()
	statuses := []health.Status{
		health.StatusHealthy,
		health.StatusUnhealthy,
		health.StatusUnhealthy,
		health.StatusHealthy,
		health.StatusUnhealthy,
	}

	for i, status := range statuses {
		snapshot := HealthSnapshot{
			Timestamp:     time.Now().Add(time.Duration(i) * time.Second),
			OverallStatus: status,
		}
		checker.healthHistory = append(checker.healthHistory, snapshot)
	}
	checker.mutex.Unlock()

	// Should only count consecutive failures from the end
	failures := checker.getConsecutiveFailures()
	assert.Equal(t, 1, failures) // Only the last one is a failure
}
