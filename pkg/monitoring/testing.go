// Package monitoring - Testing utilities for metrics
package monitoring

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// testSetupOnce ensures test setup only happens once per test run
	testSetupOnce sync.Once
	// testTeardownFuncs holds cleanup functions
	testTeardownFuncs []func()
	testMutex         sync.Mutex
)

// SetupTestMetrics prepares metrics for testing with proper isolation
// This should be called in TestMain or at the beginning of each test
func SetupTestMetrics(t *testing.T) {
	testMutex.Lock()
	defer testMutex.Unlock()

	// Enable test mode on global registry
	gr := GetGlobalRegistry()
	gr.SetTestMode(true)
	gr.ResetForTest()

	// Register cleanup
	t.Cleanup(func() {
		TeardownTestMetrics()
	})
}

// TeardownTestMetrics cleans up test metrics
func TeardownTestMetrics() {
	testMutex.Lock()
	defer testMutex.Unlock()

	// Reset the global registry
	gr := GetGlobalRegistry()
	gr.SetTestMode(false)
	gr.ResetForTest()

	// Run any registered cleanup functions
	for _, cleanup := range testTeardownFuncs {
		cleanup()
	}
	testTeardownFuncs = nil
}

// RegisterTestCleanup registers a cleanup function to be called during teardown
func RegisterTestCleanup(cleanup func()) {
	testMutex.Lock()
	defer testMutex.Unlock()
	testTeardownFuncs = append(testTeardownFuncs, cleanup)
}

// NewTestRegistry creates a new isolated registry for testing
func NewTestRegistry() prometheus.Registerer {
	return prometheus.NewRegistry()
}

// WithTestRegistry runs a function with a specific test registry
func WithTestRegistry(registry prometheus.Registerer, fn func()) {
	gr := GetGlobalRegistry()
	gr.mu.Lock()
	oldRegistry := gr.test
	oldMode := gr.testMode
	gr.test = registry
	gr.testMode = true
	gr.initFlags = make(map[string]bool) // Reset init flags
	gr.mu.Unlock()

	defer func() {
		gr.mu.Lock()
		gr.test = oldRegistry
		gr.testMode = oldMode
		gr.mu.Unlock()
	}()

	fn()
}
