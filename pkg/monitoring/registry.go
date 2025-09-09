// Package monitoring - Centralized metrics registry management
package monitoring

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// GlobalRegistry manages a global metrics registry with proper isolation for tests
type GlobalRegistry struct {
	production prometheus.Registerer
	test       prometheus.Registerer
	testMode   bool
	mu         sync.RWMutex
	once       sync.Once
	initFlags  map[string]bool // Track what's been initialized to prevent duplicates
}

var (
	// globalRegistry is the singleton instance
	globalRegistry *GlobalRegistry
	globalOnce     sync.Once
)

// GetGlobalRegistry returns the singleton global registry instance
func GetGlobalRegistry() *GlobalRegistry {
	globalOnce.Do(func() {
		globalRegistry = &GlobalRegistry{
			production: prometheus.DefaultRegisterer,
			test:       prometheus.NewRegistry(),
			testMode:   false,
			initFlags:  make(map[string]bool),
		}
	})
	return globalRegistry
}

// SetTestMode enables/disables test mode with registry isolation
func (gr *GlobalRegistry) SetTestMode(enabled bool) {
	gr.mu.Lock()
	defer gr.mu.Unlock()
	gr.testMode = enabled
	if enabled {
		// Create a fresh test registry
		gr.test = prometheus.NewRegistry()
		// Clear initialization flags for test isolation
		gr.initFlags = make(map[string]bool)
	}
}

// GetRegistry returns the appropriate registry based on current mode
func (gr *GlobalRegistry) GetRegistry() prometheus.Registerer {
	gr.mu.RLock()
	defer gr.mu.RUnlock()
	if gr.testMode {
		return gr.test
	}
	return gr.production
}

// IsTestMode returns whether we're in test mode
func (gr *GlobalRegistry) IsTestMode() bool {
	gr.mu.RLock()
	defer gr.mu.RUnlock()
	return gr.testMode
}

// SafeRegister safely registers a metric collector, avoiding panics from duplicate registrations
// It includes component tracking to prevent multiple initializations
func (gr *GlobalRegistry) SafeRegister(component string, collector prometheus.Collector) error {
	gr.mu.Lock()
	defer gr.mu.Unlock()

	// Check if this component was already initialized in current mode
	key := component
	if gr.testMode {
		key = component + "_test"
	}

	if gr.initFlags[key] {
		// Already registered in this mode, skip
		log.Log.V(1).Info("Metrics component already registered, skipping", "component", component, "testMode", gr.testMode)
		return nil
	}

	registry := gr.production
	if gr.testMode {
		registry = gr.test
	}

	if err := registry.Register(collector); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
			// Mark as registered even if it was a duplicate to prevent future attempts
			gr.initFlags[key] = true
			log.Log.V(1).Info("Metric collector already registered, marking as initialized", "component", component)
			return nil
		}
		return err
	}

	// Mark as successfully registered
	gr.initFlags[key] = true
	log.Log.V(1).Info("Successfully registered metrics component", "component", component, "testMode", gr.testMode)
	return nil
}

// MustRegister is a convenience method that panics on error (use SafeRegister instead)
func (gr *GlobalRegistry) MustRegister(component string, collectors ...prometheus.Collector) {
	for _, collector := range collectors {
		if err := gr.SafeRegister(component, collector); err != nil {
			panic(err)
		}
	}
}

// ResetForTest resets the registry state for testing
func (gr *GlobalRegistry) ResetForTest() {
	gr.mu.Lock()
	defer gr.mu.Unlock()
	gr.test = prometheus.NewRegistry()
	gr.initFlags = make(map[string]bool)
	log.Log.V(1).Info("Reset metrics registry for testing")
}

// GetTestRegistry returns the test registry (for advanced test scenarios)
func (gr *GlobalRegistry) GetTestRegistry() prometheus.Registerer {
	gr.mu.RLock()
	defer gr.mu.RUnlock()
	return gr.test
}

// IsComponentRegistered checks if a component has already been registered
func (gr *GlobalRegistry) IsComponentRegistered(component string) bool {
	gr.mu.RLock()
	defer gr.mu.RUnlock()
	key := component
	if gr.testMode {
		key = component + "_test"
	}
	return gr.initFlags[key]
}
