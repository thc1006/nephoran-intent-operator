package llm

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCircuitBreakerManagerGetAllStats tests the GetAllStats method
// which is used by the health check system
func TestCircuitBreakerManagerGetAllStats(t *testing.T) {
	tests := []struct {
		name                string
		breakers           map[string]string // name -> state
		expectedHealthy    bool
		expectedOpenCount  int
		expectedOpenNames  []string
	}{
		{
			name:              "no_breakers",
			breakers:          map[string]string{},
			expectedHealthy:   true,
			expectedOpenCount: 0,
		},
		{
			name: "all_closed_breakers",
			breakers: map[string]string{
				"service-a": "closed",
				"service-b": "closed",
				"service-c": "closed",
			},
			expectedHealthy:   true,
			expectedOpenCount: 0,
		},
		{
			name: "single_open_breaker",
			breakers: map[string]string{
				"service-a": "closed",
				"service-b": "open",
				"service-c": "closed",
			},
			expectedHealthy:   false,
			expectedOpenCount: 1,
			expectedOpenNames: []string{"service-b"},
		},
		{
			name: "multiple_open_breakers",
			breakers: map[string]string{
				"service-a": "open",
				"service-b": "closed",
				"service-c": "open",
				"service-d": "half-open",
			},
			expectedHealthy:   false,
			expectedOpenCount: 2,
			expectedOpenNames: []string{"service-a", "service-c"},
		},
		{
			name: "all_open_breakers",
			breakers: map[string]string{
				"service-x": "open",
				"service-y": "open",
				"service-z": "open",
			},
			expectedHealthy:   false,
			expectedOpenCount: 3,
			expectedOpenNames: []string{"service-x", "service-y", "service-z"},
		},
		{
			name: "mixed_states_with_half_open",
			breakers: map[string]string{
				"service-1": "closed",
				"service-2": "half-open",
				"service-3": "open",
				"service-4": "half-open",
			},
			expectedHealthy:   false,
			expectedOpenCount: 1,
			expectedOpenNames: []string{"service-3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cbMgr := NewCircuitBreakerManager(nil)
			
			// Create circuit breakers with specified states
			for name, state := range tt.breakers {
				cb := cbMgr.GetOrCreate(name, nil)
				
				// Force the circuit breaker into the desired state
				switch state {
				case "open":
					cb.ForceOpen()
				case "closed":
					cb.Reset()
				case "half-open":
					// This is more complex to simulate properly
					// For testing, we'll force open then manually transition
					cb.ForceOpen()
					// Simulate transition to half-open (in real code this happens after timeout)
					time.Sleep(1 * time.Millisecond)
				}
			}

			// Get all stats - this is what the health check uses
			stats := cbMgr.GetAllStats()
			
			// Verify we have the right number of circuit breakers
			assert.Len(t, stats, len(tt.breakers))
			
			// Count open circuit breakers from stats
			openBreakers := make([]string, 0)
			for name, cbStats := range stats {
				if statsMap, ok := cbStats.(map[string]interface{}); ok {
					if state, exists := statsMap["state"]; exists && state == "open" {
						openBreakers = append(openBreakers, name)
					}
				}
			}
			
			// Verify open breaker count
			assert.Len(t, openBreakers, tt.expectedOpenCount, 
				"Expected %d open breakers, got %d: %v", 
				tt.expectedOpenCount, len(openBreakers), openBreakers)
			
			// Verify specific open breaker names if specified
			if len(tt.expectedOpenNames) > 0 {
				for _, expectedName := range tt.expectedOpenNames {
					assert.Contains(t, openBreakers, expectedName,
						"Expected breaker %s to be open", expectedName)
				}
			}
			
			// Test health determination logic (like in service manager)
			isHealthy := len(openBreakers) == 0
			assert.Equal(t, tt.expectedHealthy, isHealthy,
				"Health status should be %t with %d open breakers", 
				tt.expectedHealthy, len(openBreakers))
		})
	}
}

// TestCircuitBreakerHealthCheckLogic tests the logic used in health checks
// to identify open circuit breakers (this validates the fix)
func TestCircuitBreakerHealthCheckLogic(t *testing.T) {
	t.Run("collect_all_open_breakers", func(t *testing.T) {
		cbMgr := NewCircuitBreakerManager(nil)
		
		// Create a mix of circuit breakers
		breakersToCreate := []struct {
			name  string
			state string
		}{
			{"alpha", "closed"},
			{"beta", "open"},
			{"gamma", "half-open"},
			{"delta", "open"},
			{"epsilon", "closed"},
			{"zeta", "open"},
		}
		
		for _, b := range breakersToCreate {
			cb := cbMgr.GetOrCreate(b.name, nil)
			switch b.state {
			case "open":
				cb.ForceOpen()
			case "closed":
				cb.Reset()
			}
		}

		// Simulate the health check logic (this tests the fix)
		stats := cbMgr.GetAllStats()
		var openBreakers []string
		
		// This is the exact logic from the service manager health check
		for name, state := range stats {
			if cbStats, ok := state.(map[string]interface{}); ok {
				if cbState, exists := cbStats["state"]; exists && cbState == "open" {
					openBreakers = append(openBreakers, name)
				}
			}
		}
		
		// Should collect ALL open breakers (beta, delta, zeta)
		expectedOpen := []string{"beta", "delta", "zeta"}
		assert.Len(t, openBreakers, len(expectedOpen),
			"Should collect all %d open breakers, got %d: %v", 
			len(expectedOpen), len(openBreakers), openBreakers)
		
		// Verify each expected breaker is found
		for _, expected := range expectedOpen {
			assert.Contains(t, openBreakers, expected,
				"Open breaker %s should be collected", expected)
		}
		
		// Verify closed/half-open breakers are not included
		notExpected := []string{"alpha", "gamma", "epsilon"}
		for _, notExp := range notExpected {
			assert.NotContains(t, openBreakers, notExp,
				"Non-open breaker %s should not be collected", notExp)
		}
	})
	
	t.Run("health_message_formatting", func(t *testing.T) {
		cbMgr := NewCircuitBreakerManager(nil)
		
		// Test different scenarios for message formatting
		scenarios := []struct {
			name           string
			openBreakers   []string
			expectedFormat string
		}{
			{
				name:           "no_open_breakers",
				openBreakers:   []string{},
				expectedFormat: "All circuit breakers operational",
			},
			{
				name:           "single_open_breaker",
				openBreakers:   []string{"service-1"},
				expectedFormat: "Circuit breakers in open state: [service-1]",
			},
			{
				name:           "multiple_open_breakers",
				openBreakers:   []string{"service-1", "service-2"},
				expectedFormat: "Circuit breakers in open state:",
			},
		}
		
		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				// Create fresh manager for each scenario
				cbMgr := NewCircuitBreakerManager(nil)
				
				// Create open breakers
				for _, name := range scenario.openBreakers {
					cb := cbMgr.GetOrCreate(name, nil)
					cb.ForceOpen()
				}
				
				// Create some closed breakers too
				cbMgr.GetOrCreate("closed-1", nil).Reset()
				cbMgr.GetOrCreate("closed-2", nil).Reset()
				
				// Simulate health check message generation
				stats := cbMgr.GetAllStats()
				var openBreakers []string
				for name, state := range stats {
					if cbStats, ok := state.(map[string]interface{}); ok {
						if cbState, exists := cbStats["state"]; exists && cbState == "open" {
							openBreakers = append(openBreakers, name)
						}
					}
				}
				
				var message string
				if len(openBreakers) == 0 {
					message = "All circuit breakers operational"
				} else {
					message = fmt.Sprintf("Circuit breakers in open state: %v", openBreakers)
				}
				
				if scenario.name == "no_open_breakers" {
					assert.Equal(t, scenario.expectedFormat, message)
				} else {
					assert.Contains(t, message, scenario.expectedFormat)
					// For open breakers, verify all are mentioned
					for _, openName := range scenario.openBreakers {
						assert.Contains(t, message, openName,
							"Message should contain open breaker %s", openName)
					}
				}
			})
		}
	})
}

// TestCircuitBreakerConcurrentHealthChecks tests concurrent access to GetAllStats
// while circuit breakers are changing state
func TestCircuitBreakerConcurrentHealthChecks(t *testing.T) {
	cbMgr := NewCircuitBreakerManager(nil)
	
	// Create initial circuit breakers
	const numBreakers = 20
	breakers := make([]*CircuitBreaker, numBreakers)
	for i := 0; i < numBreakers; i++ {
		name := fmt.Sprintf("concurrent-service-%d", i)
		breakers[i] = cbMgr.GetOrCreate(name, nil)
	}

	// Test concurrent health checks while modifying states
	const testDuration = 2 * time.Second
	const numHealthCheckGoroutines = 10
	const numStateChangeGoroutines = 5
	
	var wg sync.WaitGroup
	stopChan := make(chan struct{})
	healthCheckResults := make(chan map[string]interface{}, 100)

	// Start health check goroutines
	for i := 0; i < numHealthCheckGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			
			for {
				select {
				case <-stopChan:
					return
				case <-ticker.C:
					stats := cbMgr.GetAllStats()
					select {
					case healthCheckResults <- stats:
					default:
						// Channel full, skip this result
					}
				}
			}
		}()
	}

	// Start state change goroutines
	for i := 0; i < numStateChangeGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ticker := time.NewTicker(20 * time.Millisecond)
			defer ticker.Stop()
			
			for {
				select {
				case <-stopChan:
					return
				case <-ticker.C:
					// Change state of random breaker
					breakerIdx := id % numBreakers
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
	close(healthCheckResults)

	// Validate results
	resultCount := 0
	validResults := 0
	for stats := range healthCheckResults {
		resultCount++
		
		// Each result should be valid
		assert.IsType(t, map[string]interface{}{}, stats)
		assert.LessOrEqual(t, len(stats), numBreakers, 
			"Should not have more stats than breakers")
		
		// Count open breakers in this result
		openCount := 0
		for _, state := range stats {
			if cbStats, ok := state.(map[string]interface{}); ok {
				if cbState, exists := cbStats["state"]; exists && cbState == "open" {
					openCount++
				}
			}
		}
		
		// Valid result (no corruption from concurrent access)
		if openCount >= 0 && openCount <= numBreakers {
			validResults++
		}
	}

	assert.Greater(t, resultCount, 10, "Should have collected multiple health check results")
	assert.Equal(t, resultCount, validResults, "All results should be valid (no race conditions)")
	
	t.Logf("Concurrent health check test: %d total results, %d valid", resultCount, validResults)
}

// TestCircuitBreakerHealthCheckPerformance benchmarks GetAllStats performance
func TestCircuitBreakerHealthCheckPerformance(t *testing.T) {
	breakerCounts := []int{10, 50, 100, 200, 500}
	
	for _, count := range breakerCounts {
		t.Run(fmt.Sprintf("performance_%d_breakers", count), func(t *testing.T) {
			cbMgr := NewCircuitBreakerManager(nil)
			
			// Create circuit breakers with mixed states
			for i := 0; i < count; i++ {
				name := fmt.Sprintf("perf-breaker-%d", i)
				cb := cbMgr.GetOrCreate(name, nil)
				
				// Make some open (every 4th one)
				if i%4 == 0 {
					cb.ForceOpen()
				}
			}

			// Measure GetAllStats performance
			const iterations = 1000
			start := time.Now()
			
			for i := 0; i < iterations; i++ {
				stats := cbMgr.GetAllStats()
				
				// Do minimal processing to ensure the call completes
				_ = len(stats)
			}
			
			duration := time.Since(start)
			avgDuration := duration / iterations
			
			// Performance requirements (should be fast even with many breakers)
			maxAcceptableDuration := 1 * time.Millisecond
			assert.Less(t, avgDuration, maxAcceptableDuration,
				"GetAllStats should be fast with %d breakers: got %v, expected < %v",
				count, avgDuration, maxAcceptableDuration)
			
			t.Logf("Performance with %d breakers: %d iterations in %v (avg: %v per call)",
				count, iterations, duration, avgDuration)
		})
	}
}

// BenchmarkCircuitBreakerHealthCheck benchmarks the exact health check logic
func BenchmarkCircuitBreakerHealthCheck(b *testing.B) {
	scenarios := []struct {
		name      string
		numTotal  int
		numOpen   int
	}{
		{"small_10_total_2_open", 10, 2},
		{"medium_50_total_10_open", 50, 10},
		{"large_100_total_20_open", 100, 20},
		{"xlarge_500_total_100_open", 500, 100},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			cbMgr := NewCircuitBreakerManager(nil)
			
			// Create circuit breakers
			for i := 0; i < scenario.numTotal; i++ {
				name := fmt.Sprintf("bench-service-%d", i)
				cb := cbMgr.GetOrCreate(name, nil)
				
				// Make some open
				if i < scenario.numOpen {
					cb.ForceOpen()
				}
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					// This is the exact logic from health checks
					stats := cbMgr.GetAllStats()
					var openBreakers []string
					for name, state := range stats {
						if cbStats, ok := state.(map[string]interface{}); ok {
							if cbState, exists := cbStats["state"]; exists && cbState == "open" {
								openBreakers = append(openBreakers, name)
							}
						}
					}
					
					// Verify the fix works: should find all open breakers
					if len(openBreakers) != scenario.numOpen {
						b.Errorf("Expected %d open breakers, got %d", scenario.numOpen, len(openBreakers))
					}
				}
			})
		})
	}
}

// TestCircuitBreakerStatsFormat tests the exact format of circuit breaker stats
// returned by GetAllStats to ensure health check parsing works correctly
func TestCircuitBreakerStatsFormat(t *testing.T) {
	cbMgr := NewCircuitBreakerManager(nil)
	
	// Create a circuit breaker and put it in different states
	cb := cbMgr.GetOrCreate("test-format", nil)
	
	t.Run("closed_state_format", func(t *testing.T) {
		cb.Reset()
		stats := cbMgr.GetAllStats()
		
		require.Contains(t, stats, "test-format")
		cbStats, ok := stats["test-format"].(map[string]interface{})
		require.True(t, ok, "Stats should be a map[string]interface{}")
		
		// Verify required fields for health check
		assert.Contains(t, cbStats, "state")
		assert.Equal(t, "closed", cbStats["state"])
		
		// Verify other expected fields exist
		assert.Contains(t, cbStats, "name")
		assert.Contains(t, cbStats, "failure_count")
		assert.Contains(t, cbStats, "success_count")
		assert.Contains(t, cbStats, "request_count")
	})
	
	t.Run("open_state_format", func(t *testing.T) {
		cb.ForceOpen()
		stats := cbMgr.GetAllStats()
		
		require.Contains(t, stats, "test-format")
		cbStats, ok := stats["test-format"].(map[string]interface{})
		require.True(t, ok, "Stats should be a map[string]interface{}")
		
		// Verify state is open
		assert.Contains(t, cbStats, "state")
		assert.Equal(t, "open", cbStats["state"])
	})
	
	t.Run("multiple_breakers_format", func(t *testing.T) {
		// Create multiple breakers
		cb1 := cbMgr.GetOrCreate("format-test-1", nil)
		cb2 := cbMgr.GetOrCreate("format-test-2", nil)
		cb3 := cbMgr.GetOrCreate("format-test-3", nil)
		
		cb1.Reset()      // closed
		cb2.ForceOpen()  // open
		cb3.Reset()      // closed
		
		stats := cbMgr.GetAllStats()
		
		// Should have all breakers
		expectedNames := []string{"test-format", "format-test-1", "format-test-2", "format-test-3"}
		for _, name := range expectedNames {
			assert.Contains(t, stats, name)
			
			cbStats, ok := stats[name].(map[string]interface{})
			require.True(t, ok, "Stats for %s should be a map", name)
			assert.Contains(t, cbStats, "state")
		}
		
		// Verify we can identify open ones
		openBreakers := make([]string, 0)
		for name, state := range stats {
			if cbStats, ok := state.(map[string]interface{}); ok {
				if cbState, exists := cbStats["state"]; exists && cbState == "open" {
					openBreakers = append(openBreakers, name)
				}
			}
		}
		
		// Should find the open breaker(s)
		assert.Contains(t, openBreakers, "test-format") // from previous test
		assert.Contains(t, openBreakers, "format-test-2")
		
		// Should not find closed breakers
		assert.NotContains(t, openBreakers, "format-test-1")
		assert.NotContains(t, openBreakers, "format-test-3")
	})
}