package monitoring

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalRegistry(t *testing.T) {
	t.Run("singleton behavior", func(t *testing.T) {
		gr1 := GetGlobalRegistry()
		gr2 := GetGlobalRegistry()
		assert.Same(t, gr1, gr2, "Should return same instance")
	})

	t.Run("test mode isolation", func(t *testing.T) {
		gr := GetGlobalRegistry()
		
		// Initially in production mode
		assert.False(t, gr.IsTestMode())
		
		// Switch to test mode
		gr.SetTestMode(true)
		assert.True(t, gr.IsTestMode())
		
		// Switch back to production mode
		gr.SetTestMode(false)
		assert.False(t, gr.IsTestMode())
	})

	t.Run("safe registration prevents duplicates", func(t *testing.T) {
		SetupTestMetrics(t)
		gr := GetGlobalRegistry()
		
		// Create a test metric
		testCounter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_counter_duplicates",
			Help: "Test counter for duplicates",
		})
		
		// First registration should succeed
		err1 := gr.SafeRegister("test-component", testCounter)
		assert.NoError(t, err1)
		
		// Second registration should not fail
		err2 := gr.SafeRegister("test-component", testCounter)
		assert.NoError(t, err2)
		
		// Component should be marked as registered
		assert.True(t, gr.IsComponentRegistered("test-component"))
	})

	t.Run("component tracking per mode", func(t *testing.T) {
		gr := GetGlobalRegistry()
		gr.ResetForTest()
		
		testCounter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_counter_modes",
			Help: "Test counter for mode tracking",
		})
		
		// Register in production mode
		gr.SetTestMode(false)
		gr.SafeRegister("test-mode-component", testCounter)
		assert.True(t, gr.IsComponentRegistered("test-mode-component"))
		
		// Switch to test mode - should not be registered there
		gr.SetTestMode(true)
		assert.False(t, gr.IsComponentRegistered("test-mode-component"))
		
		// Register in test mode
		testCounter2 := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_counter_modes2",
			Help: "Test counter for mode tracking 2",
		})
		gr.SafeRegister("test-mode-component", testCounter2)
		assert.True(t, gr.IsComponentRegistered("test-mode-component"))
	})
}

func TestConcurrentAccess(t *testing.T) {
	SetupTestMetrics(t)
	gr := GetGlobalRegistry()
	
	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	// Test concurrent registration of the same metric
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			
			counter := prometheus.NewCounter(prometheus.CounterOpts{
				Name: "concurrent_test_counter",
				Help: "Test counter for concurrency",
			})
			
			// This should not panic or error even with concurrent access
			err := gr.SafeRegister("concurrent-component", counter)
			assert.NoError(t, err)
		}(i)
	}
	
	wg.Wait()
	
	// Component should be registered exactly once
	assert.True(t, gr.IsComponentRegistered("concurrent-component"))
}

func TestWithTestRegistry(t *testing.T) {
	testRegistry := NewTestRegistry()
	var registeredInside bool
	
	WithTestRegistry(testRegistry, func() {
		gr := GetGlobalRegistry()
		assert.True(t, gr.IsTestMode())
		assert.Equal(t, testRegistry, gr.GetTestRegistry())
		
		// Register something in the custom registry
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "custom_registry_counter",
			Help: "Test counter for custom registry",
		})
		err := gr.SafeRegister("custom-component", counter)
		assert.NoError(t, err)
		registeredInside = gr.IsComponentRegistered("custom-component")
	})
	
	assert.True(t, registeredInside)
	
	// After exiting the scope, should be back to normal state
	gr := GetGlobalRegistry()
	assert.False(t, gr.IsTestMode())
}

func TestSetupTeardownTestMetrics(t *testing.T) {
	t.Run("proper cleanup", func(t *testing.T) {
		SetupTestMetrics(t)
		gr := GetGlobalRegistry()
		
		// Should be in test mode
		assert.True(t, gr.IsTestMode())
		
		// Register a component
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "cleanup_test_counter",
			Help: "Test counter for cleanup",
		})
		gr.SafeRegister("cleanup-component", counter)
		assert.True(t, gr.IsComponentRegistered("cleanup-component"))
		
		// Manually trigger cleanup to test behavior
		TeardownTestMetrics()
		
		// Should be back to production mode
		assert.False(t, gr.IsTestMode())
	})

	t.Run("cleanup functions", func(t *testing.T) {
		var cleanupCalled bool
		RegisterTestCleanup(func() {
			cleanupCalled = true
		})
		
		TeardownTestMetrics()
		assert.True(t, cleanupCalled)
	})
}

// TestMetricsDuplicateRegistrationScenario tests the specific scenario that was causing panics
func TestMetricsDuplicateRegistrationScenario(t *testing.T) {
	// This test simulates what happens when tests run concurrently or sequentially
	// and try to register the same metrics multiple times
	
	t.Run("sequential test runs", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			t.Run("subtest", func(t *testing.T) {
				SetupTestMetrics(t)
				gr := GetGlobalRegistry()
				
				// This should work every time without panicking
				counter := prometheus.NewCounter(prometheus.CounterOpts{
					Name: "sequential_test_counter",
					Help: "Test counter for sequential tests",
				})
				
				require.NotPanics(t, func() {
					err := gr.SafeRegister("sequential-component", counter)
					assert.NoError(t, err)
				})
			})
		}
	})

	t.Run("multiple registrations same test", func(t *testing.T) {
		SetupTestMetrics(t)
		gr := GetGlobalRegistry()
		
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "multiple_reg_counter",
			Help: "Test counter for multiple registrations",
		})
		
		// Register the same metric multiple times - should not panic
		for i := 0; i < 5; i++ {
			require.NotPanics(t, func() {
				err := gr.SafeRegister("multiple-reg-component", counter)
				assert.NoError(t, err)
			})
		}
	})
}
