package monitoring

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDuplicateRegistrationFix validates that our fix prevents
// "panic: duplicate metrics collector registration attempted" errors
func TestDuplicateRegistrationFix(t *testing.T) {
	t.Run("no panic on duplicate registration", func(t *testing.T) {
		SetupTestMetrics(t)
		gr := GetGlobalRegistry()
		
		// Create the same metric multiple times (this simulates what happens in real scenarios)
		createMetric := func() prometheus.Counter {
			return prometheus.NewCounter(prometheus.CounterOpts{
				Name: "test_duplicate_counter",
				Help: "Counter for testing duplicate registration",
			})
		}
		
		metric1 := createMetric()
		metric2 := createMetric()
		
		// This should not panic
		require.NotPanics(t, func() {
			err1 := gr.SafeRegister("duplicate-test", metric1)
			assert.NoError(t, err1)
			
			err2 := gr.SafeRegister("duplicate-test", metric2)
			assert.NoError(t, err2) // Should still succeed (no error)
		})
	})

	t.Run("concurrent registration scenario", func(t *testing.T) {
		SetupTestMetrics(t)
		gr := GetGlobalRegistry()
		
		const numWorkers = 50
		var wg sync.WaitGroup
		wg.Add(numWorkers)
		
		// Simulate multiple goroutines trying to register the same metrics
		// This is common in test scenarios where init() functions run concurrently
		for i := 0; i < numWorkers; i++ {
			go func(workerID int) {
				defer wg.Done()
				
				// Each worker tries to register the same metric
				counter := prometheus.NewCounter(prometheus.CounterOpts{
					Name: "concurrent_registration_counter",
					Help: "Counter for testing concurrent registration",
				})
				
				// This should never panic, even under high concurrency
				require.NotPanics(t, func() {
					err := gr.SafeRegister("concurrent-component", counter)
					assert.NoError(t, err)
				})
			}(i)
		}
		
		wg.Wait()
		
		// Verify the component was registered
		assert.True(t, gr.IsComponentRegistered("concurrent-component"))
	})

	t.Run("test isolation prevents cross-test conflicts", func(t *testing.T) {
		// Run multiple sub-tests that would conflict if not properly isolated
		for i := 0; i < 3; i++ {
			t.Run("isolated-subtest", func(t *testing.T) {
				SetupTestMetrics(t)
				gr := GetGlobalRegistry()
				
				// Each subtest can register the same metric name without conflicts
				counter := prometheus.NewCounter(prometheus.CounterOpts{
					Name: "isolation_test_counter",
					Help: "Counter for testing isolation",
				})
				
				require.NotPanics(t, func() {
					err := gr.SafeRegister("isolation-component", counter)
					assert.NoError(t, err)
				})
			})
		}
	})

	t.Run("mixed MustRegister and SafeRegister", func(t *testing.T) {
		SetupTestMetrics(t)
		gr := GetGlobalRegistry()
		
		counter1 := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "mixed_test_counter1",
			Help: "Counter for testing mixed registration 1",
		})
		
		counter2 := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "mixed_test_counter2",
			Help: "Counter for testing mixed registration 2",
		})
		
		// Use SafeRegister first
		require.NotPanics(t, func() {
			err := gr.SafeRegister("mixed-component1", counter1)
			assert.NoError(t, err)
		})
		
		// Use MustRegister - should not panic due to centralized tracking
		require.NotPanics(t, func() {
			gr.MustRegister("mixed-component2", counter2)
		})
		
		// Try to register same components again - should not panic
		require.NotPanics(t, func() {
			err := gr.SafeRegister("mixed-component1", counter1)
			assert.NoError(t, err)
			gr.MustRegister("mixed-component2", counter2)
		})
	})
}

// TestRealWorldScenarios tests scenarios that commonly cause duplicate registration issues
func TestRealWorldScenarios(t *testing.T) {
	t.Run("package init functions", func(t *testing.T) {
		// Simulate what happens when package init() functions run multiple times
		// This often happens in test environments
		SetupTestMetrics(t)
		gr := GetGlobalRegistry()
		
		// Simulate init function running multiple times
		initFunction := func() {
			counter := prometheus.NewCounter(prometheus.CounterOpts{
				Name: "package_init_counter",
				Help: "Counter typically registered in init()",
			})
			
			// This should not panic even when called multiple times
			err := gr.SafeRegister("package-metrics", counter)
			assert.NoError(t, err)
		}
		
		// Call init function multiple times
		for i := 0; i < 5; i++ {
			require.NotPanics(t, func() {
				initFunction()
			})
		}
	})

	t.Run("controller reconciliation metrics", func(t *testing.T) {
		// Controllers often register metrics during startup/reconciliation
		SetupTestMetrics(t)
		gr := GetGlobalRegistry()
		
		// Simulate controller startup function
		controllerSetup := func(controllerName string) {
			reconcilesTotal := prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: "controller_reconciles_total",
					Help: "Total number of reconciliations",
				},
				[]string{"controller"},
			)
			
			err := gr.SafeRegister("controller-reconciles-"+controllerName, reconcilesTotal)
			assert.NoError(t, err)
		}
		
		// Multiple controllers can register similar metrics
		controllers := []string{"networkintent", "e2nodeset", "audittrail"}
		for _, controller := range controllers {
			require.NotPanics(t, func() {
				controllerSetup(controller)
			})
		}
		
		// Simulate restart - controllers try to register again
		for _, controller := range controllers {
			require.NotPanics(t, func() {
				controllerSetup(controller)
			})
		}
	})

	t.Run("monitoring service metrics", func(t *testing.T) {
		// Various monitoring services registering metrics
		SetupTestMetrics(t)
		gr := GetGlobalRegistry()
		
		// Different services that might register overlapping metrics
		services := []struct {
			name      string
			metricName string
		}{
			{"audit-system", "audit_queries_total"},
			{"health-checker", "health_checks_total"},
			{"llm-processor", "llm_requests_total"},
			{"alert-manager", "alerts_processed_total"},
		}
		
		for _, service := range services {
			require.NotPanics(t, func() {
				counter := prometheus.NewCounter(prometheus.CounterOpts{
					Name: service.metricName,
					Help: "Service counter for " + service.name,
				})
				
				err := gr.SafeRegister(service.name, counter)
				assert.NoError(t, err)
				
				// Service restarts should not cause issues
				err = gr.SafeRegister(service.name, counter)
				assert.NoError(t, err)
			})
		}
	})
}
