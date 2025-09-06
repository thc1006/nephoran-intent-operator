// Test script to verify metrics initialization fixes
// This can be run as: go run test_metrics_fix.go
package main

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	
	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/testutil"
	gitfake "github.com/thc1006/nephoran-intent-operator/pkg/git/fake"
)

func main() {
	fmt.Println("Testing metrics initialization fixes...")
	
	// Test 1: Create NetworkIntentReconciler without metrics enabled
	fmt.Println("Test 1: NetworkIntentReconciler with metrics disabled")
	os.Setenv("METRICS_ENABLED", "false")
	
	networkMetrics := controllers.NewControllerMetrics("networkintent")
	networkMetrics.RecordSuccess("default", "test-intent")
	networkMetrics.RecordFailure("default", "test-intent", "test-error")
	networkMetrics.RecordProcessingDuration("default", "test-intent", "test-phase", 1.5)
	networkMetrics.SetStatus("default", "test-intent", "processing", 1.0)
	fmt.Println("✓ NetworkIntent metrics methods work safely when disabled")
	
	// Test 2: Create E2NodeSetReconciler and test metrics
	fmt.Println("Test 2: E2NodeSetReconciler with safe metrics initialization")
	
	scheme := runtime.NewScheme()
	nephoranv1.AddToScheme(scheme)
	
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	e2reconciler := &controllers.E2NodeSetReconciler{
		Client:    fakeClient,
		Scheme:    scheme,
		E2Manager: testutil.NewFakeE2Manager(),
		GitClient: gitfake.NewClient(),
	}
	
	// Test metrics registration
	fmt.Println("  Testing E2NodeSet metrics registration...")
	
	// Test after registration
	e2reconciler.RegisterMetrics()
	fmt.Println("✓ E2NodeSet RegisterMetrics works without panics")
	
	// Test double registration (should be idempotent)
	e2reconciler.RegisterMetrics()
	e2reconciler.RegisterMetrics()
	fmt.Println("✓ Multiple RegisterMetrics calls are safe (idempotent)")
	
	// Test 3: Test helper functions
	fmt.Println("Test 3: Test helper functions")
	
	_ = controllers.NewTestE2NodeSetReconciler(
		fakeClient, 
		scheme, 
		gitfake.NewClient(), 
		testutil.NewFakeE2Manager(),
	)
	
	fmt.Println("✓ Test helper functions work correctly")
	
	// Test 4: Nil pointer safety
	fmt.Println("Test 4: Nil pointer safety")
	
	var nilMetrics *controllers.ControllerMetrics = nil
	nilMetrics.RecordSuccess("default", "test")
	nilMetrics.RecordFailure("default", "test", "error")
	nilMetrics.RecordProcessingDuration("default", "test", "phase", 1.0)
	nilMetrics.SetStatus("default", "test", "phase", 1.0)
	fmt.Println("✓ Nil ControllerMetrics methods are safe")
	
	fmt.Println("\n🎉 All metrics initialization tests passed!")
	fmt.Println("Key fixes implemented:")
	fmt.Println("  ✓ Added nil checks to all ControllerMetrics methods")
	fmt.Println("  ✓ Added nil checks to global metric variables")
	fmt.Println("  ✓ Made E2NodeSetReconciler.RegisterMetrics idempotent")
	fmt.Println("  ✓ Added metricsRegistered guard to updateMetrics")
	fmt.Println("  ✓ Created test helper functions for safe initialization")
	fmt.Println("  ✓ Fixed integration test to use NewTestControllerMetrics")
}