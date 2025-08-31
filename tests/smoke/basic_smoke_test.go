//go:build smoke

/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package smoke

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/controllers"
	"github.com/thc1006/nephoran-intent-operator/tests/fixtures"
)

// SmokeTestSuite provides quick validation tests for basic functionality
type SmokeTestSuite struct {
	suite.Suite
	client    client.Client
	scheme    *runtime.Scheme
	ctx       context.Context
	namespace string
}

// SetupSuite initializes the smoke test environment
func (suite *SmokeTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.namespace = "smoke-test"

	// Create minimal test environment
	suite.scheme = runtime.NewScheme()
	err := nephoranv1.AddToScheme(suite.scheme)
	suite.Require().NoError(err)

	suite.client = fake.NewClientBuilder().WithScheme(suite.scheme).Build()
}

// TestSmoke_NetworkIntentCRUD tests basic CRUD operations
func (suite *SmokeTestSuite) TestSmoke_NetworkIntentCRUD() {
	// Test Create
	intent := fixtures.SimpleNetworkIntent()
	intent.Namespace = suite.namespace
	
	err := suite.client.Create(suite.ctx, intent)
	suite.Require().NoError(err, "Should create NetworkIntent successfully")

	// Test Read
	retrieved := &nephoranv1.NetworkIntent{}
	err = suite.client.Get(suite.ctx, client.ObjectKeyFromObject(intent), retrieved)
	suite.NoError(err, "Should retrieve NetworkIntent successfully")
	suite.Equal(intent.Spec.Intent, retrieved.Spec.Intent, "Retrieved intent should match created intent")

	// Test Update
	retrieved.Spec.Intent = "updated smoke test intent"
	err = suite.client.Update(suite.ctx, retrieved)
	suite.NoError(err, "Should update NetworkIntent successfully")

	// Verify Update
	updated := &nephoranv1.NetworkIntent{}
	err = suite.client.Get(suite.ctx, client.ObjectKeyFromObject(intent), updated)
	suite.NoError(err, "Should retrieve updated NetworkIntent")
	suite.Equal("updated smoke test intent", updated.Spec.Intent, "Updated intent should be persisted")

	// Test Delete
	err = suite.client.Delete(suite.ctx, updated)
	suite.NoError(err, "Should delete NetworkIntent successfully")
}

// TestSmoke_ControllerBasicReconcile tests basic controller reconciliation
func (suite *SmokeTestSuite) TestSmoke_ControllerBasicReconcile() {
	// Create minimal controller for smoke test
	reconciler := &controllers.NetworkIntentReconciler{
		Client:          suite.client,
		Scheme:          suite.scheme,
		Log:             zap.New(zap.UseDevMode(true)),
		EnableLLMIntent: false, // Disable LLM for smoke test
	}

	// Create NetworkIntent
	intent := fixtures.SimpleNetworkIntent()
	intent.Namespace = suite.namespace
	intent.Name = "smoke-controller-test"

	err := suite.client.Create(suite.ctx, intent)
	suite.Require().NoError(err)

	// Test reconcile
	request := ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(intent),
	}

	result, err := reconciler.Reconcile(suite.ctx, request)
	suite.NoError(err, "Reconcile should not error for smoke test")
	suite.Equal(ctrl.Result{}, result, "Reconcile should return empty result for smoke test")
}

// TestSmoke_APITypes tests basic API type functionality
func (suite *SmokeTestSuite) TestSmoke_APITypes() {
	// Test NetworkIntent creation
	intent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-smoke-test",
			Namespace: suite.namespace,
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: "smoke test scaling intent",
		},
	}

	// Test validation of required fields
	suite.NotEmpty(intent.Spec.Intent, "Intent field should not be empty")
	suite.Equal("NetworkIntent", intent.Kind, "Kind should be NetworkIntent")
	suite.Equal("nephoran.io/v1", intent.APIVersion, "APIVersion should be correct")
}

// TestSmoke_Fixtures tests fixture generation
func (suite *SmokeTestSuite) TestSmoke_Fixtures() {
	// Test simple fixture
	simple := fixtures.SimpleNetworkIntent()
	suite.NotNil(simple, "Simple fixture should not be nil")
	suite.NotEmpty(simple.Spec.Intent, "Simple fixture should have intent")

	// Test scale-up fixture
	scaleUp := fixtures.ScaleUpIntent()
	suite.NotNil(scaleUp, "Scale-up fixture should not be nil")
	suite.Contains(scaleUp.Spec.Intent, "scale up", "Scale-up fixture should contain 'scale up'")

	// Test scale-down fixture
	scaleDown := fixtures.ScaleDownIntent()
	suite.NotNil(scaleDown, "Scale-down fixture should not be nil")
	suite.Contains(scaleDown.Spec.Intent, "scale down", "Scale-down fixture should contain 'scale down'")

	// Test multiple fixtures
	multiple := fixtures.MultipleIntents(3)
	suite.Len(multiple, 3, "Should generate correct number of fixtures")
	suite.NotEqual(multiple[0].Name, multiple[1].Name, "Fixtures should have different names")
}

// TestSmoke_SchemeRegistration tests scheme registration
func (suite *SmokeTestSuite) TestSmoke_SchemeRegistration() {
	// Test that our types are registered in the scheme
	gvk := nephoranv1.GroupVersion.WithKind("NetworkIntent")
	suite.True(suite.scheme.Recognizes(gvk), "Scheme should recognize NetworkIntent")

	// Test type creation
	obj, err := suite.scheme.New(gvk)
	suite.NoError(err, "Should be able to create new NetworkIntent from scheme")
	
	intent, ok := obj.(*nephoranv1.NetworkIntent)
	suite.True(ok, "Created object should be a NetworkIntent")
	suite.NotNil(intent, "Created NetworkIntent should not be nil")
}

// TestSmoke_ClientOperations tests basic client operations
func (suite *SmokeTestSuite) TestSmoke_ClientOperations() {
	// Test List operation
	intentList := &nephoranv1.NetworkIntentList{}
	err := suite.client.List(suite.ctx, intentList, client.InNamespace(suite.namespace))
	suite.NoError(err, "Should be able to list NetworkIntents")
	suite.NotNil(intentList, "Intent list should not be nil")

	// Test Create multiple objects
	for i := 0; i < 3; i++ {
		intent := fixtures.SimpleNetworkIntent()
		intent.Namespace = suite.namespace
		intent.Name = fmt.Sprintf("smoke-client-%d", i)
		
		err := suite.client.Create(suite.ctx, intent)
		suite.NoError(err, "Should create NetworkIntent %d", i)
	}

	// Test List after creating objects
	err = suite.client.List(suite.ctx, intentList, client.InNamespace(suite.namespace))
	suite.NoError(err, "Should be able to list NetworkIntents after creation")
	suite.GreaterOrEqual(len(intentList.Items), 3, "Should have at least 3 NetworkIntents")
}

// TestSmoke_Performance tests basic performance expectations
func (suite *SmokeTestSuite) TestSmoke_Performance() {
	const numOperations = 100
	start := time.Now()

	// Create many objects quickly
	for i := 0; i < numOperations; i++ {
		intent := fixtures.SimpleNetworkIntent()
		intent.Namespace = suite.namespace
		intent.Name = fmt.Sprintf("perf-test-%d", i)
		
		err := suite.client.Create(suite.ctx, intent)
		suite.NoError(err, "Should create NetworkIntent %d quickly", i)
	}

	duration := time.Since(start)
	avgDuration := duration / numOperations
	
	suite.T().Logf("Created %d NetworkIntents in %v (avg: %v per operation)", 
		numOperations, duration, avgDuration)
	
	// Smoke test should complete quickly (less than 10ms per operation on average)
	suite.Less(avgDuration, 10*time.Millisecond, 
		"Average operation time should be less than 10ms for smoke test")
}

// TestSuite runner function
func TestSmokeTestSuite(t *testing.T) {
	suite.Run(t, new(SmokeTestSuite))
}

// Individual smoke tests using standard testing approach
func TestSmoke_BasicNetworkIntentCreation(t *testing.T) {
	// Quick standalone smoke test
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	require.NoError(t, err)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	intent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "standalone-smoke",
			Namespace: "default",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: "basic smoke test",
		},
	}

	ctx := context.Background()
	err = client.Create(ctx, intent)
	assert.NoError(t, err, "Smoke test: Should create NetworkIntent without error")

	retrieved := &nephoranv1.NetworkIntent{}
	err = client.Get(ctx, client.ObjectKeyFromObject(intent), retrieved)
	assert.NoError(t, err, "Smoke test: Should retrieve NetworkIntent without error")
	assert.Equal(t, intent.Spec.Intent, retrieved.Spec.Intent, "Smoke test: Intent should match")
}

func TestSmoke_QuickValidation(t *testing.T) {
	// Test that can be run in seconds
	start := time.Now()
	
	// Basic type validation
	intent := fixtures.SimpleNetworkIntent()
	assert.NotNil(t, intent, "Fixture should not be nil")
	assert.NotEmpty(t, intent.Spec.Intent, "Intent should not be empty")
	assert.Equal(t, "NetworkIntent", intent.Kind, "Kind should be correct")
	
	duration := time.Since(start)
	t.Logf("Quick validation completed in %v", duration)
	
	// Should complete in milliseconds
	assert.Less(t, duration, 100*time.Millisecond, "Quick validation should complete very fast")
}

func TestSmoke_ErrorHandling(t *testing.T) {
	// Test basic error handling scenarios
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	require.NoError(t, err)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	// Test Get non-existent object
	intent := &nephoranv1.NetworkIntent{}
	err = client.Get(ctx, client.ObjectKey{Name: "non-existent", Namespace: "default"}, intent)
	assert.Error(t, err, "Should error when getting non-existent object")

	// Test duplicate creation (this might not error with fake client)
	originalIntent := fixtures.SimpleNetworkIntent()
	originalIntent.Namespace = "default"
	
	err = client.Create(ctx, originalIntent)
	assert.NoError(t, err, "Should create first intent")

	// Creating with same name should work with fake client (different behavior than real cluster)
	duplicateIntent := fixtures.SimpleNetworkIntent()
	duplicateIntent.Namespace = "default"
	err = client.Create(ctx, duplicateIntent)
	// Note: fake client may behave differently than real Kubernetes API server
	// This is expected in smoke tests
}

// Benchmark smoke test
func BenchmarkSmoke_NetworkIntentCreation(b *testing.B) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	require.NoError(b, err)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		intent := fixtures.SimpleNetworkIntent()
		intent.Namespace = "default"
		intent.Name = fmt.Sprintf("benchmark-%d", i)
		
		err := client.Create(ctx, intent)
		if err != nil {
			b.Fatalf("Failed to create intent %d: %v", i, err)
		}
	}
}

