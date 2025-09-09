package controllers

import (
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
)

// NewTestControllerMetrics creates a ControllerMetrics instance for testing.
// This ensures tests can safely use metrics without requiring METRICS_ENABLED to be set.
func NewTestControllerMetrics(controllerName string) *ControllerMetrics {
	return &ControllerMetrics{
		controllerName: controllerName,
		enabled:        false, // Always disabled in tests to avoid registration issues
	}
}

// EnableMetricsForTesting temporarily enables metrics for testing purposes.
// Call DisableMetricsForTesting() to restore the original state.
func EnableMetricsForTesting() func() {
	originalValue := os.Getenv("METRICS_ENABLED")
	os.Setenv("METRICS_ENABLED", "true")
	
	// Force re-evaluation of metrics enabled state
	registerMetricsOnce()
	
	return func() {
		if originalValue == "" {
			os.Unsetenv("METRICS_ENABLED")
		} else {
			os.Setenv("METRICS_ENABLED", originalValue)
		}
	}
}

// DisableMetricsForTesting explicitly disables metrics for testing.
func DisableMetricsForTesting() {
	os.Setenv("METRICS_ENABLED", "false")
}

// NewTestControllerMetricsEnabled creates a ControllerMetrics instance for testing with metrics enabled.
// Only use this in tests that specifically need to test metrics functionality.
func NewTestControllerMetricsEnabled(controllerName string) *ControllerMetrics {
	// Temporarily enable metrics for this test
	cleanup := EnableMetricsForTesting()
	defer cleanup()
	
	return &ControllerMetrics{
		controllerName: controllerName,
		enabled:        true,
	}
}

// InitializeE2NodeSetReconcilerForTesting ensures an E2NodeSetReconciler has safe metrics initialization.
// This helper can be called on any E2NodeSetReconciler instance in tests.
func InitializeE2NodeSetReconcilerForTesting(r *E2NodeSetReconciler) {
	if r == nil {
		return
	}
	// RegisterMetrics is now idempotent and safe to call multiple times
	r.RegisterMetrics()
}

// NewTestE2NodeSetReconciler creates a fully initialized E2NodeSetReconciler for testing.
func NewTestE2NodeSetReconciler(client client.Client, scheme *runtime.Scheme, gitClient git.ClientInterface, e2Manager e2.E2ManagerInterface) *E2NodeSetReconciler {
	r := &E2NodeSetReconciler{
		Client:    client,
		Scheme:    scheme,
		GitClient: gitClient,
		E2Manager: e2Manager,
		Recorder:  &record.FakeRecorder{},
	}
	
	// Initialize metrics to prevent nil pointer panics in tests
	InitializeE2NodeSetReconcilerForTesting(r)
	
	return r
}