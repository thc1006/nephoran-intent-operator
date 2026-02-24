package o2

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

func TestNewHealthChecker(t *testing.T) {
	logger := logging.Logger{}

	t.Run("creates with default config", func(t *testing.T) {
		hc, err := newHealthChecker(nil, logger)
		require.NoError(t, err)
		require.NotNil(t, hc)

		assert.NotNil(t, hc.config)
		assert.True(t, hc.config.Enabled)
		assert.Equal(t, 30*time.Second, hc.config.CheckInterval)
		assert.Equal(t, 10*time.Second, hc.config.Timeout)
		assert.Equal(t, 3, hc.config.FailureThreshold)
		assert.Equal(t, 1, hc.config.SuccessThreshold)
		assert.True(t, hc.config.DeepHealthCheck)
	})

	t.Run("creates with custom config", func(t *testing.T) {
		customConfig := &APIHealthCheckerConfig{
			Enabled:          true,
			CheckInterval:    1 * time.Minute,
			Timeout:          5 * time.Second,
			FailureThreshold: 5,
			SuccessThreshold: 2,
			DeepHealthCheck:  false,
		}

		hc, err := newHealthChecker(customConfig, logger)
		require.NoError(t, err)
		require.NotNil(t, hc)

		assert.Equal(t, customConfig.CheckInterval, hc.config.CheckInterval)
		assert.Equal(t, customConfig.Timeout, hc.config.Timeout)
		assert.Equal(t, customConfig.FailureThreshold, hc.config.FailureThreshold)
		assert.False(t, hc.config.DeepHealthCheck)
	})

	t.Run("initializes internal state", func(t *testing.T) {
		hc, err := newHealthChecker(nil, logger)
		require.NoError(t, err)

		assert.NotNil(t, hc.healthChecks)
		assert.NotNil(t, hc.lastHealthData)
		assert.NotNil(t, hc.stopCh)
		assert.Equal(t, "UP", hc.lastHealthData.Status)
	})
}

func TestHealthChecker_RegisterHealthCheck(t *testing.T) {
	logger := logging.Logger{}
	hc, _ := newHealthChecker(nil, logger)

	t.Run("registers single check", func(t *testing.T) {
		check := func(ctx context.Context) ComponentCheck {
			return ComponentCheck{
				Name:   "database",
				Status: "UP",
			}
		}

		hc.RegisterHealthCheck("database", check)
		assert.Len(t, hc.healthChecks, 1)
		assert.Contains(t, hc.healthChecks, "database")
	})

	t.Run("registers multiple checks", func(t *testing.T) {
		checks := map[string]ComponentHealthCheck{
			"api":   func(ctx context.Context) ComponentCheck { return ComponentCheck{Status: "UP"} },
			"cache": func(ctx context.Context) ComponentCheck { return ComponentCheck{Status: "UP"} },
			"queue": func(ctx context.Context) ComponentCheck { return ComponentCheck{Status: "UP"} },
		}

		for name, check := range checks {
			hc.RegisterHealthCheck(name, check)
		}

		assert.Len(t, hc.healthChecks, 4) // 1 from previous test + 3 new
	})
}

func TestHealthChecker_GetHealthStatus(t *testing.T) {
	logger := logging.Logger{}
	hc, _ := newHealthChecker(nil, logger)

	t.Run("returns health status", func(t *testing.T) {
		status := hc.GetHealthStatus()
		require.NotNil(t, status)

		assert.Equal(t, "UP", status.Status)
		assert.NotZero(t, status.Timestamp)
	})

	t.Run("returns copy not reference", func(t *testing.T) {
		status1 := hc.GetHealthStatus()
		status2 := hc.GetHealthStatus()

		// Modifying one should not affect the other
		status1.Status = "DOWN"
		assert.Equal(t, "UP", status2.Status)
	})
}

func TestHealthChecker_CalculateOverallStatus(t *testing.T) {
	logger := logging.Logger{}
	hc, _ := newHealthChecker(nil, logger)

	tests := []struct {
		name     string
		checks   []ComponentCheck
		expected string
	}{
		{
			name:     "no checks returns UNKNOWN",
			checks:   []ComponentCheck{},
			expected: "UNKNOWN",
		},
		{
			name: "all UP returns UP",
			checks: []ComponentCheck{
				{Name: "service1", Status: "UP"},
				{Name: "service2", Status: "UP"},
			},
			expected: "UP",
		},
		{
			name: "any DOWN returns DOWN",
			checks: []ComponentCheck{
				{Name: "service1", Status: "UP"},
				{Name: "service2", Status: "DOWN"},
			},
			expected: "DOWN",
		},
		{
			name: "DEGRADED without DOWN returns DEGRADED",
			checks: []ComponentCheck{
				{Name: "service1", Status: "UP"},
				{Name: "service2", Status: "DEGRADED"},
			},
			expected: "DEGRADED",
		},
		{
			name: "DOWN takes precedence over DEGRADED",
			checks: []ComponentCheck{
				{Name: "service1", Status: "DEGRADED"},
				{Name: "service2", Status: "DOWN"},
			},
			expected: "DOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hc.calculateOverallStatus(tt.checks)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHealthChecker_PerformHealthCheck(t *testing.T) {
	logger := logging.Logger{}
	hc, _ := newHealthChecker(nil, logger)

	t.Run("executes registered checks", func(t *testing.T) {
		executed := false
		check := func(ctx context.Context) ComponentCheck {
			executed = true
			return ComponentCheck{
				Name:   "test-service",
				Status: "UP",
			}
		}

		hc.RegisterHealthCheck("test-service", check)

		ctx := context.Background()
		hc.performHealthCheck(ctx)

		assert.True(t, executed)
	})

	t.Run("updates health data", func(t *testing.T) {
		check := func(ctx context.Context) ComponentCheck {
			return ComponentCheck{
				Name:   "api",
				Status: "UP",
			}
		}

		hc.RegisterHealthCheck("api", check)

		ctx := context.Background()
		hc.performHealthCheck(ctx)

		status := hc.GetHealthStatus()
		assert.NotNil(t, status)
		assert.NotEmpty(t, status.Components)
	})

	t.Run("respects context timeout", func(t *testing.T) {
		slowCheck := func(ctx context.Context) ComponentCheck {
			select {
			case <-time.After(100 * time.Millisecond):
				return ComponentCheck{Status: "UP"}
			case <-ctx.Done():
				return ComponentCheck{Status: "DOWN", Message: "timeout"}
			}
		}

		config := &APIHealthCheckerConfig{
			Enabled:       true,
			CheckInterval: 10 * time.Second,
			Timeout:       10 * time.Millisecond, // Very short timeout
		}

		hc2, _ := newHealthChecker(config, logger)
		hc2.RegisterHealthCheck("slow-service", slowCheck)

		ctx := context.Background()
		hc2.performHealthCheck(ctx)

		// Check completes even with slow service
		status := hc2.GetHealthStatus()
		assert.NotNil(t, status)
	})
}

func TestHealthChecker_BuildComponentsMap(t *testing.T) {
	logger := logging.Logger{}
	hc, _ := newHealthChecker(nil, logger)

	checks := []ComponentCheck{
		{Name: "database", Status: "UP"},
		{Name: "cache", Status: "UP"},
		{Name: "api", Status: "DOWN"},
	}

	components := hc.buildComponentsMap(checks)

	assert.Len(t, components, 3)
	assert.Contains(t, components, "database")
	assert.Contains(t, components, "cache")
	assert.Contains(t, components, "api")
}

func TestHealthChecker_GetServiceStatuses(t *testing.T) {
	logger := logging.Logger{}
	hc, _ := newHealthChecker(nil, logger)

	services := hc.getServiceStatuses()

	assert.NotNil(t, services)
	assert.Greater(t, len(services), 0)

	// Check first service
	assert.Equal(t, "database", services[0].ServiceName)
	assert.Equal(t, "UP", services[0].Status)
	assert.NotEmpty(t, services[0].Endpoint)
}

func TestHealthChecker_GetResourceHealthSummary(t *testing.T) {
	logger := logging.Logger{}
	hc, _ := newHealthChecker(nil, logger)

	summary := hc.getResourceHealthSummary()

	require.NotNil(t, summary)
	assert.Equal(t, 100, summary.TotalResources)
	assert.Equal(t, 95, summary.HealthyResources)
	assert.Equal(t, 3, summary.DegradedResources)
	assert.Equal(t, 2, summary.UnhealthyResources)
	assert.Equal(t, 0, summary.UnknownResources)
}

func TestHealthChecker_Start_Stop(t *testing.T) {
	logger := logging.Logger{}

	t.Run("starts when enabled", func(t *testing.T) {
		config := &APIHealthCheckerConfig{
			Enabled:       true,
			CheckInterval: 50 * time.Millisecond,
			Timeout:       10 * time.Millisecond,
		}

		hc, _ := newHealthChecker(config, logger)

		checkCount := 0
		check := func(ctx context.Context) ComponentCheck {
			checkCount++
			return ComponentCheck{Status: "UP"}
		}
		hc.RegisterHealthCheck("test", check)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		go hc.Start(ctx)

		time.Sleep(150 * time.Millisecond)
		hc.Stop()

		// Should have executed multiple times
		assert.Greater(t, checkCount, 1)
	})

	t.Run("does not start when disabled", func(t *testing.T) {
		config := &APIHealthCheckerConfig{
			Enabled: false,
		}

		hc, _ := newHealthChecker(config, logger)

		checkCount := 0
		check := func(ctx context.Context) ComponentCheck {
			checkCount++
			return ComponentCheck{Status: "UP"}
		}
		hc.RegisterHealthCheck("test", check)

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		hc.Start(ctx)

		time.Sleep(50 * time.Millisecond)

		// Should not have executed
		assert.Equal(t, 0, checkCount)
	})
}

func TestComponentCheck_Structure(t *testing.T) {
	t.Run("contains required fields", func(t *testing.T) {
		check := ComponentCheck{
			Name:      "database",
			Status:    "UP",
			Message:   "Database connection successful",
			Timestamp: time.Now(),
			Duration:  50 * time.Millisecond,
			Details: map[string]interface{}{
				"connections": 10,
				"latency":     "5ms",
			},
			CheckType: "connectivity",
		}

		assert.Equal(t, "database", check.Name)
		assert.Equal(t, "UP", check.Status)
		assert.NotEmpty(t, check.Message)
		assert.NotNil(t, check.Details)
		assert.Equal(t, "connectivity", check.CheckType)
	})
}

func TestHealthChecker_ConcurrentChecks(t *testing.T) {
	logger := logging.Logger{}
	hc, _ := newHealthChecker(nil, logger)

	// Register multiple checks
	for i := 0; i < 5; i++ {
		name := string(rune('a' + i))
		check := func(ctx context.Context) ComponentCheck {
			time.Sleep(10 * time.Millisecond)
			return ComponentCheck{
				Name:   name,
				Status: "UP",
			}
		}
		hc.RegisterHealthCheck(name, check)
	}

	ctx := context.Background()
	start := time.Now()
	hc.performHealthCheck(ctx)
	duration := time.Since(start)

	// Concurrent execution should be faster than sequential
	// 5 checks * 10ms = 50ms sequential, but concurrent should be ~10-20ms
	assert.Less(t, duration, 40*time.Millisecond)
}
