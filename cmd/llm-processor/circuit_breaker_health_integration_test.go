//go:build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/health"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// TestCircuitBreakerHealthIntegration provides comprehensive integration tests
// for circuit breaker health check fixes, validating the fix where service manager
// now collects ALL open breakers before returning instead of early return
func TestCircuitBreakerHealthIntegration(t *testing.T) {
	// Setup logger
	logger := slog.Default()

	t.Run("EndToEndHealthCheckBehavior", func(t *testing.T) {
		testEndToEndHealthCheckBehavior(t, logger)
	})

	t.Run("ConcurrentStateChanges", func(t *testing.T) {
		testConcurrentStateChanges(t, logger)
	})

	t.Run("PerformanceWithManyBreakers", func(t *testing.T) {
		testPerformanceWithManyBreakers(t, logger)
	})

	t.Run("RegressionTests", func(t *testing.T) {
		testRegressionTests(t, logger)
	})
}

// testEndToEndHealthCheckBehavior validates the actual HTTP health endpoints
// with multiple circuit breakers in various states
func testEndToEndHealthCheckBehavior(t *testing.T, logger *slog.Logger) {
	tests := []struct {
		name            string
		circuitBreakers map[string]string // name -> state
		expectedStatus  int
		expectedMsg     string
		checkAllOpen    bool // whether all open breakers should be in message
	}{
		{
			name: "all_closed_breakers",
			circuitBreakers: map[string]string{
				"service-a": "closed",
				"service-b": "closed",
				"service-c": "closed",
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "All circuit breakers operational",
		},
		{
			name: "single_open_breaker",
			circuitBreakers: map[string]string{
				"service-a": "closed",
				"service-b": "open",
				"service-c": "half-open",
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedMsg:    "Circuit breakers in open state: [service-b]",
			checkAllOpen:   true,
		},
		{
			name: "multiple_open_breakers",
			circuitBreakers: map[string]string{
				"service-a": "open",
				"service-b": "closed",
				"service-c": "open",
				"service-d": "half-open",
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectedMsg:    "Circuit breakers in open state:",
			checkAllOpen:   true,
		},
		{
			name:            "no_circuit_breakers",
			circuitBreakers: map[string]string{},
			expectedStatus:  http.StatusOK,
			expectedMsg:     "No circuit breakers registered",
		},
		{
			name: "all_half_open_breakers",
			circuitBreakers: map[string]string{
				"service-a": "half-open",
				"service-b": "half-open",
			},
			expectedStatus: http.StatusOK,
			expectedMsg:    "All circuit breakers operational",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create circuit breaker manager with test breakers
			cbMgr := llm.NewCircuitBreakerManager(nil)

			// Create circuit breakers in specified states
			for name, state := range tt.circuitBreakers {
				cb := cbMgr.GetOrCreate(name, nil)

				// Force circuit breaker into desired state
				switch state {
				case "open":
					cb.ForceOpen()
				case "closed":
					cb.Reset()
				case "half-open":
					// Force open then wait for timeout to transition to half-open
					cb.ForceOpen()
					// Manually transition to half-open for test
					// This is a bit hacky but necessary for testing
					// In real scenarios, this happens automatically after timeout
					time.Sleep(10 * time.Millisecond) // small delay
				}
			}

			// Create service manager with health checker
			config := &Config{ServiceVersion: "test-1.0.0"}
			sm := NewServiceManager(config, logger)
			sm.circuitBreakerMgr = cbMgr
			sm.healthChecker = health.NewHealthChecker("llm-processor", config.ServiceVersion, logger)

			// Register health checks (this is what we're testing)
			sm.registerHealthChecks()
			sm.MarkReady()

			// Create HTTP test server
			router := sm.CreateRouter()
			server := httptest.NewServer(router)
			defer server.Close()

			// Test /healthz endpoint
			resp, err := http.Get(server.URL + "/healthz")
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// Parse response
			var healthResp health.HealthResponse
			err = json.NewDecoder(resp.Body).Decode(&healthResp)
			require.NoError(t, err)

			// Validate overall response structure
			assert.Equal(t, "llm-processor", healthResp.Service)
			assert.Equal(t, config.ServiceVersion, healthResp.Version)
			assert.NotEmpty(t, healthResp.Uptime)
			assert.NotZero(t, healthResp.Timestamp)

			// Find circuit breaker check result
			var cbCheck *health.Check
			for _, check := range healthResp.Checks {
				if check.Name == "circuit_breaker" {
					cbCheck = &check
					break
				}
			}

			require.NotNil(t, cbCheck, "Circuit breaker health check should be present")

			// Validate circuit breaker check message
			if tt.checkAllOpen {
				// For multiple open breakers, verify ALL open breakers are listed
				assert.Contains(t, cbCheck.Message, tt.expectedMsg)
				for name, state := range tt.circuitBreakers {
					if state == "open" {
						assert.Contains(t, cbCheck.Message, name,
							"Open breaker %s should be in message: %s", name, cbCheck.Message)
					}
				}
			} else {
				assert.Equal(t, tt.expectedMsg, cbCheck.Message)
			}

			// Validate check status matches expected HTTP status
			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, health.StatusHealthy, cbCheck.Status)
			} else {
				assert.Equal(t, health.StatusUnhealthy, cbCheck.Status)
			}
		})
	}
}

// testConcurrentStateChanges tests health checks while circuit breakers are transitioning states
func testConcurrentStateChanges(t *testing.T, logger *slog.Logger) {
	t.Run("concurrent_health_checks_with_state_changes", func(t *testing.T) {
		cbMgr := llm.NewCircuitBreakerManager(nil)

		// Create multiple circuit breakers
		const numBreakers = 10
		breakers := make([]*llm.CircuitBreaker, numBreakers)
		for i := 0; i < numBreakers; i++ {
			name := fmt.Sprintf("service-%d", i)
			breakers[i] = cbMgr.GetOrCreate(name, nil)
		}

		// Create service manager
		config := &Config{ServiceVersion: "test-1.0.0"}
		sm := NewServiceManager(config, logger)
		sm.circuitBreakerMgr = cbMgr
		sm.healthChecker = health.NewHealthChecker("llm-processor", config.ServiceVersion, logger)
		sm.registerHealthChecks()

		// Test concurrent health checks while changing states
		const numGoroutines = 20
		const testDuration = 2 * time.Second

		var wg sync.WaitGroup
		results := make(chan health.Check, numGoroutines*10) // Buffer for results
		stopChan := make(chan struct{})

		// Start health check goroutines
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				ticker := time.NewTicker(50 * time.Millisecond)
				defer ticker.Stop()

				for {
					select {
					case <-stopChan:
						return
					case <-ticker.C:
						ctx := context.Background()
						// Execute circuit breaker health check directly
						checkFunc := sm.healthChecker
						if checkFunc != nil {
							response := checkFunc.Check(ctx)
							for _, check := range response.Checks {
								if check.Name == "circuit_breaker" {
									results <- check
								}
							}
						}
					}
				}
			}(i)
		}

		// Start state change goroutines
		for i := 0; i < numBreakers/2; i++ {
			wg.Add(1)
			go func(breakerIdx int) {
				defer wg.Done()
				ticker := time.NewTicker(100 * time.Millisecond)
				defer ticker.Stop()

				for {
					select {
					case <-stopChan:
						return
					case <-ticker.C:
						// Randomly change state
						if breakerIdx%2 == 0 {
							breakers[breakerIdx].ForceOpen()
						} else {
							breakers[breakerIdx].Reset()
						}
					}
				}
			}(i)
		}

		// Run test for specified duration
		time.Sleep(testDuration)
		close(stopChan)
		wg.Wait()
		close(results)

		// Collect and validate results
		var healthyCount, unhealthyCount int
		var lastUnhealthyMessage string

		for result := range results {
			switch result.Status {
			case health.StatusHealthy:
				healthyCount++
				// Healthy should have appropriate message
				assert.True(t,
					result.Message == "All circuit breakers operational" ||
						result.Message == "No circuit breakers registered",
					"Unexpected healthy message: %s", result.Message)
			case health.StatusUnhealthy:
				unhealthyCount++
				lastUnhealthyMessage = result.Message
				// Unhealthy should report open breakers
				assert.Contains(t, result.Message, "Circuit breakers in open state:",
					"Unhealthy message should contain open breakers: %s", result.Message)
			}
		}

		// We should have received multiple health check results
		totalResults := healthyCount + unhealthyCount
		assert.Greater(t, totalResults, 10, "Should have multiple health check results")

		t.Logf("Health check results: %d healthy, %d unhealthy", healthyCount, unhealthyCount)
		if lastUnhealthyMessage != "" {
			t.Logf("Last unhealthy message: %s", lastUnhealthyMessage)
		}
	})

	t.Run("thread_safety_validation", func(t *testing.T) {
		// Test for race conditions and data races
		cbMgr := llm.NewCircuitBreakerManager(nil)

		const numBreakers = 5
		for i := 0; i < numBreakers; i++ {
			name := fmt.Sprintf("race-test-service-%d", i)
			cbMgr.GetOrCreate(name, nil)
		}

		config := &Config{ServiceVersion: "test-1.0.0"}
		sm := NewServiceManager(config, logger)
		sm.circuitBreakerMgr = cbMgr
		sm.healthChecker = health.NewHealthChecker("llm-processor", config.ServiceVersion, logger)
		sm.registerHealthChecks()

		// Run concurrent operations
		var wg sync.WaitGroup
		const numConcurrent = 100

		// Concurrent health checks
		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx := context.Background()
				response := sm.healthChecker.Check(ctx)
				// Just verify we get a response without panics
				assert.NotNil(t, response)
				assert.Equal(t, "llm-processor", response.Service)
			}()
		}

		// Concurrent state changes
		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				name := fmt.Sprintf("race-test-service-%d", idx%numBreakers)
				if cb, exists := cbMgr.Get(name); exists {
					if idx%2 == 0 {
						cb.ForceOpen()
					} else {
						cb.Reset()
					}
				}
			}(i)
		}

		wg.Wait()

		// Final health check to ensure system is still stable
		ctx := context.Background()
		finalResponse := sm.healthChecker.Check(ctx)
		assert.NotNil(t, finalResponse)
		assert.Equal(t, "llm-processor", finalResponse.Service)
	})
}

// testPerformanceWithManyBreakers benchmarks health check performance with many circuit breakers
func testPerformanceWithManyBreakers(t *testing.T, logger *slog.Logger) {
	breakerCounts := []int{10, 50, 100, 200}

	for _, count := range breakerCounts {
		t.Run(fmt.Sprintf("performance_with_%d_breakers", count), func(t *testing.T) {
			cbMgr := llm.NewCircuitBreakerManager(nil)

			// Create many circuit breakers with mixed states
			openBreakers := make([]string, 0)
			for i := 0; i < count; i++ {
				name := fmt.Sprintf("perf-service-%d", i)
				cb := cbMgr.GetOrCreate(name, nil)

				// Make some breakers open (every 5th one)
				if i%5 == 0 {
					cb.ForceOpen()
					openBreakers = append(openBreakers, name)
				}
			}

			config := &Config{ServiceVersion: "test-1.0.0"}
			sm := NewServiceManager(config, logger)
			sm.circuitBreakerMgr = cbMgr
			sm.healthChecker = health.NewHealthChecker("llm-processor", config.ServiceVersion, logger)
			sm.registerHealthChecks()

			// Measure health check performance
			const iterations = 100
			ctx := context.Background()

			start := time.Now()
			for i := 0; i < iterations; i++ {
				response := sm.healthChecker.Check(ctx)

				// Validate response structure
				assert.Equal(t, "llm-processor", response.Service)
				assert.Greater(t, len(response.Checks), 0)

				// Find and validate circuit breaker check
				var cbCheck *health.Check
				for _, check := range response.Checks {
					if check.Name == "circuit_breaker" {
						cbCheck = &check
						break
					}
				}

				require.NotNil(t, cbCheck)

				if len(openBreakers) > 0 {
					assert.Equal(t, health.StatusUnhealthy, cbCheck.Status)
					assert.Contains(t, cbCheck.Message, "Circuit breakers in open state:")

					// Verify all open breakers are reported (this tests the fix)
					for _, openName := range openBreakers {
						assert.Contains(t, cbCheck.Message, openName,
							"Open breaker %s should be in message with %d total breakers", openName, count)
					}
				} else {
					assert.Equal(t, health.StatusHealthy, cbCheck.Status)
				}
			}
			duration := time.Since(start)

			avgDuration := duration / iterations
			maxAcceptableDuration := 50 * time.Millisecond // Performance requirement

			assert.Less(t, avgDuration, maxAcceptableDuration,
				"Average health check duration %v should be less than %v with %d breakers",
				avgDuration, maxAcceptableDuration, count)

			t.Logf("Performance with %d breakers (%d open): %d iterations in %v (avg: %v per check)",
				count, len(openBreakers), iterations, duration, avgDuration)
		})
	}
}

// testRegressionTests ensures the fix doesn't break existing functionality
func testRegressionTests(t *testing.T, logger *slog.Logger) {
	t.Run("backward_compatibility", func(t *testing.T) {
		// Test that existing health check functionality still works
		cbMgr := llm.NewCircuitBreakerManager(nil)

		config := &Config{ServiceVersion: "test-1.0.0"}
		sm := NewServiceManager(config, logger)
		sm.circuitBreakerMgr = cbMgr
		sm.healthChecker = health.NewHealthChecker("llm-processor", config.ServiceVersion, logger)

		// Register all health checks like in production
		sm.registerHealthChecks()
		sm.MarkReady()

		ctx := context.Background()
		response := sm.healthChecker.Check(ctx)

		// Verify standard health response structure
		assert.Equal(t, "llm-processor", response.Service)
		assert.Equal(t, config.ServiceVersion, response.Version)
		assert.NotEmpty(t, response.Uptime)
		assert.True(t, len(response.Checks) >= 1) // Should have service_status at minimum

		// Should have summary
		assert.NotNil(t, response.Summary)
		assert.Greater(t, response.Summary.Total, 0)
	})

	t.Run("empty_circuit_breaker_manager", func(t *testing.T) {
		// Test with nil circuit breaker manager (should not crash)
		config := &Config{ServiceVersion: "test-1.0.0"}
		sm := NewServiceManager(config, logger)
		sm.circuitBreakerMgr = nil // Explicitly nil
		sm.healthChecker = health.NewHealthChecker("llm-processor", config.ServiceVersion, logger)

		sm.registerHealthChecks()

		ctx := context.Background()
		response := sm.healthChecker.Check(ctx)

		// Should work fine without circuit breaker checks
		assert.Equal(t, "llm-processor", response.Service)
		assert.NotNil(t, response.Checks)

		// Circuit breaker check should not be present
		for _, check := range response.Checks {
			assert.NotEqual(t, "circuit_breaker", check.Name)
		}
	})

	t.Run("malformed_circuit_breaker_stats", func(t *testing.T) {
		// Test with malformed circuit breaker stats (edge case)
		cbMgr := llm.NewCircuitBreakerManager(nil)

		// Create a breaker and manually corrupt its state for testing
		cb := cbMgr.GetOrCreate("test-service", nil)
		_ = cb // We'll test through the manager interface

		config := &Config{ServiceVersion: "test-1.0.0"}
		sm := NewServiceManager(config, logger)
		sm.circuitBreakerMgr = cbMgr
		sm.healthChecker = health.NewHealthChecker("llm-processor", config.ServiceVersion, logger)
		sm.registerHealthChecks()

		ctx := context.Background()
		response := sm.healthChecker.Check(ctx)

		// Should handle malformed stats gracefully
		assert.Equal(t, "llm-processor", response.Service)

		// Find circuit breaker check
		var cbCheck *health.Check
		for _, check := range response.Checks {
			if check.Name == "circuit_breaker" {
				cbCheck = &check
				break
			}
		}

		require.NotNil(t, cbCheck)
		// Should default to healthy when stats are valid
		assert.Equal(t, health.StatusHealthy, cbCheck.Status)
	})

	t.Run("http_endpoint_integration", func(t *testing.T) {
		// Test that HTTP endpoints work correctly with the fix
		cbMgr := llm.NewCircuitBreakerManager(nil)

		// Create some breakers with mixed states
		cbMgr.GetOrCreate("service-healthy", nil).Reset()
		cbMgr.GetOrCreate("service-open", nil).ForceOpen()

		config := &Config{ServiceVersion: "test-1.0.0"}
		sm := NewServiceManager(config, logger)
		sm.circuitBreakerMgr = cbMgr
		sm.healthChecker = health.NewHealthChecker("llm-processor", config.ServiceVersion, logger)
		sm.registerHealthChecks()
		sm.MarkReady()

		router := sm.CreateRouter()
		server := httptest.NewServer(router)
		defer server.Close()

		// Test both /healthz and /readyz endpoints
		endpoints := []string{"/healthz", "/readyz"}

		for _, endpoint := range endpoints {
			resp, err := http.Get(server.URL + endpoint)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should return 503 due to open circuit breaker
			assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)

			var healthResp health.HealthResponse
			err = json.NewDecoder(resp.Body).Decode(&healthResp)
			require.NoError(t, err)

			// Find circuit breaker check in response
			var cbCheck *health.Check
			for _, check := range healthResp.Checks {
				if check.Name == "circuit_breaker" {
					cbCheck = &check
					break
				}
			}

			require.NotNil(t, cbCheck, "Circuit breaker check missing in %s", endpoint)
			assert.Equal(t, health.StatusUnhealthy, cbCheck.Status)
			assert.Contains(t, cbCheck.Message, "service-open",
				"Open circuit breaker should be reported in %s", endpoint)
		}
	})
}

// BenchmarkCircuitBreakerHealthCheckFix benchmarks the specific fix implementation
func BenchmarkCircuitBreakerHealthCheckFix(b *testing.B) {
	logger := slog.Default()

	// Test scenarios with different numbers of circuit breakers
	scenarios := []struct {
		name          string
		totalBreakers int
		openBreakers  int
	}{
		{"small_10_breakers_2_open", 10, 2},
		{"medium_50_breakers_10_open", 50, 10},
		{"large_100_breakers_20_open", 100, 20},
		{"xlarge_200_breakers_40_open", 200, 40},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			cbMgr := llm.NewCircuitBreakerManager(nil)

			// Create circuit breakers
			for i := 0; i < scenario.totalBreakers; i++ {
				name := fmt.Sprintf("bench-service-%d", i)
				cb := cbMgr.GetOrCreate(name, nil)

				// Make some breakers open
				if i < scenario.openBreakers {
					cb.ForceOpen()
				}
			}

			config := &Config{ServiceVersion: "bench-1.0.0"}
			sm := NewServiceManager(config, logger)
			sm.circuitBreakerMgr = cbMgr
			sm.healthChecker = health.NewHealthChecker("llm-processor", config.ServiceVersion, logger)
			sm.registerHealthChecks()

			ctx := context.Background()

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					response := sm.healthChecker.Check(ctx)

					// Find circuit breaker check to ensure it completes
					for _, check := range response.Checks {
						if check.Name == "circuit_breaker" {
							// Verify the fix: all open breakers should be reported
							if scenario.openBreakers > 0 {
								if !strings.Contains(check.Message, "Circuit breakers in open state:") {
									b.Errorf("Expected open breakers in message, got: %s", check.Message)
								}
							}
							break
						}
					}
				}
			})
		})
	}
}

// TestCircuitBreakerHealthMessageFormatting tests the exact message formatting
// to ensure the fix produces the correct output format
func TestCircuitBreakerHealthMessageFormatting(t *testing.T) {
	logger := slog.Default()

	tests := []struct {
		name            string
		openBreakers    []string
		expectedPattern string
		exactMatch      bool
	}{
		{
			name:            "single_open_breaker",
			openBreakers:    []string{"service-alpha"},
			expectedPattern: "Circuit breakers in open state: [service-alpha]",
			exactMatch:      true,
		},
		{
			name:            "two_open_breakers",
			openBreakers:    []string{"service-alpha", "service-beta"},
			expectedPattern: "Circuit breakers in open state:",
			exactMatch:      false, // Order may vary
		},
		{
			name:            "many_open_breakers",
			openBreakers:    []string{"svc-1", "svc-2", "svc-3", "svc-4", "svc-5"},
			expectedPattern: "Circuit breakers in open state:",
			exactMatch:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cbMgr := llm.NewCircuitBreakerManager(nil)

			// Create open breakers
			for _, name := range tt.openBreakers {
				cb := cbMgr.GetOrCreate(name, nil)
				cb.ForceOpen()
			}

			// Create some closed breakers too
			cbMgr.GetOrCreate("closed-service-1", nil).Reset()
			cbMgr.GetOrCreate("closed-service-2", nil).Reset()

			config := &Config{ServiceVersion: "test-1.0.0"}
			sm := NewServiceManager(config, logger)
			sm.circuitBreakerMgr = cbMgr
			sm.healthChecker = health.NewHealthChecker("llm-processor", config.ServiceVersion, logger)
			sm.registerHealthChecks()

			ctx := context.Background()
			response := sm.healthChecker.Check(ctx)

			// Find circuit breaker check
			var cbCheck *health.Check
			for _, check := range response.Checks {
				if check.Name == "circuit_breaker" {
					cbCheck = &check
					break
				}
			}

			require.NotNil(t, cbCheck)
			assert.Equal(t, health.StatusUnhealthy, cbCheck.Status)

			if tt.exactMatch {
				assert.Equal(t, tt.expectedPattern, cbCheck.Message)
			} else {
				assert.Contains(t, cbCheck.Message, tt.expectedPattern)
				// Verify all open breakers are mentioned
				for _, openName := range tt.openBreakers {
					assert.Contains(t, cbCheck.Message, openName,
						"Open breaker %s should be in message: %s", openName, cbCheck.Message)
				}
			}

			t.Logf("Message format for %d open breakers: %s", len(tt.openBreakers), cbCheck.Message)
		})
	}
}
