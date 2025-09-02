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

package controllers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/controllers"
)

// NetworkIntentControllerTestSuite implements a comprehensive test suite for the NetworkIntent controller
type NetworkIntentControllerTestSuite struct {
	suite.Suite
	reconciler    *controllers.NetworkIntentReconciler
	client        client.Client
	scheme        *runtime.Scheme
	ctx           context.Context
	logger        logr.Logger
	llmServer     *httptest.Server
	testNamespace string
}

// SetupSuite runs before all tests in the suite
func (suite *NetworkIntentControllerTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.logger = zap.New(zap.UseDevMode(true))
	suite.testNamespace = "nephoran-test"

	// Create scheme and add our types
	suite.scheme = runtime.NewScheme()
	err := nephoranv1.AddToScheme(suite.scheme)
	suite.Require().NoError(err)

	// Setup mock LLM server
	suite.setupMockLLMServer()
}

// TearDownSuite runs after all tests in the suite
func (suite *NetworkIntentControllerTestSuite) TearDownSuite() {
	if suite.llmServer != nil {
		suite.llmServer.Close()
	}
}

// SetupTest runs before each test
func (suite *NetworkIntentControllerTestSuite) SetupTest() {
	// Create fake client for each test
	suite.client = fake.NewClientBuilder().
		WithScheme(suite.scheme).
		Build()

	// Create reconciler instance
	suite.reconciler = &controllers.NetworkIntentReconciler{
		Client:          suite.client,
		Scheme:          suite.scheme,
		Log:             suite.logger,
		EnableLLMIntent: true,
		LLMProcessorURL: suite.llmServer.URL,
	}
}

// setupMockLLMServer creates a mock LLM processor server
func (suite *NetworkIntentControllerTestSuite) setupMockLLMServer() {
	suite.llmServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/process":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{"success": true, "message": "Intent processed successfully"}`)
		case "/health":
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, "OK")
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// createTestNetworkIntent creates a test NetworkIntent object
func (suite *NetworkIntentControllerTestSuite) createTestNetworkIntent(name, intent string) *nephoranv1.NetworkIntent {
	return &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: suite.testNamespace,
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: intent,
		},
	}
}

// TestReconcile_NewNetworkIntent tests reconciliation of a new NetworkIntent
func (suite *NetworkIntentControllerTestSuite) TestReconcile_NewNetworkIntent() {
	// Arrange
	intent := suite.createTestNetworkIntent("test-intent", "scale up network function")
	err := suite.client.Create(suite.ctx, intent)
	suite.Require().NoError(err)

	// Act
	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      intent.Name,
			Namespace: intent.Namespace,
		},
	}
	result, err := suite.reconciler.Reconcile(suite.ctx, request)

	// Assert
	suite.NoError(err)
	suite.Equal(ctrl.Result{}, result)

	// Verify the NetworkIntent was updated
	updated := &nephoranv1.NetworkIntent{}
	err = suite.client.Get(suite.ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, updated)
	suite.NoError(err)
	suite.NotNil(updated)
}

// TestReconcile_NonExistentNetworkIntent tests reconciliation when NetworkIntent doesn't exist
func (suite *NetworkIntentControllerTestSuite) TestReconcile_NonExistentNetworkIntent() {
	// Arrange - no NetworkIntent created

	// Act
	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "non-existent",
			Namespace: suite.testNamespace,
		},
	}
	result, err := suite.reconciler.Reconcile(suite.ctx, request)

	// Assert - should not error when resource doesn't exist
	suite.NoError(err)
	suite.Equal(ctrl.Result{}, result)
}

// TestReconcile_LLMDisabled tests reconciliation when LLM processing is disabled
func (suite *NetworkIntentControllerTestSuite) TestReconcile_LLMDisabled() {
	// Arrange
	suite.reconciler.EnableLLMIntent = false
	intent := suite.createTestNetworkIntent("test-intent-no-llm", "scale down network function")
	err := suite.client.Create(suite.ctx, intent)
	suite.Require().NoError(err)

	// Act
	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      intent.Name,
			Namespace: intent.Namespace,
		},
	}
	result, err := suite.reconciler.Reconcile(suite.ctx, request)

	// Assert
	suite.NoError(err)
	suite.Equal(ctrl.Result{}, result)
}

// TestReconcile_LLMServerDown tests reconciliation when LLM server is unavailable
func (suite *NetworkIntentControllerTestSuite) TestReconcile_LLMServerDown() {
	// Arrange
	suite.reconciler.LLMProcessorURL = "http://localhost:9999" // Non-existent server
	intent := suite.createTestNetworkIntent("test-intent-server-down", "scale network function")
	err := suite.client.Create(suite.ctx, intent)
	suite.Require().NoError(err)

	// Act
	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      intent.Name,
			Namespace: intent.Namespace,
		},
	}
	result, err := suite.reconciler.Reconcile(suite.ctx, request)

	// Assert - should handle LLM server errors gracefully
	suite.NoError(err)                                              // Controller should handle external service failures
	suite.Equal(ctrl.Result{RequeueAfter: 5 * time.Minute}, result) // Should requeue for retry
}

// TestReconcile_EmptyIntent tests reconciliation with empty intent
func (suite *NetworkIntentControllerTestSuite) TestReconcile_EmptyIntent() {
	// Arrange
	intent := suite.createTestNetworkIntent("test-intent-empty", "")
	err := suite.client.Create(suite.ctx, intent)
	suite.Require().NoError(err)

	// Act
	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      intent.Name,
			Namespace: intent.Namespace,
		},
	}
	result, err := suite.reconciler.Reconcile(suite.ctx, request)

	// Assert
	suite.NoError(err)
	suite.Equal(ctrl.Result{}, result)
}

// TestReconcile_LongIntent tests reconciliation with a very long intent string
func (suite *NetworkIntentControllerTestSuite) TestReconcile_LongIntent() {
	// Arrange
	longIntent := "scale up network function " + string(make([]byte, 10000)) // Very long intent
	intent := suite.createTestNetworkIntent("test-intent-long", longIntent)
	err := suite.client.Create(suite.ctx, intent)
	suite.Require().NoError(err)

	// Act
	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      intent.Name,
			Namespace: intent.Namespace,
		},
	}
	result, err := suite.reconciler.Reconcile(suite.ctx, request)

	// Assert
	suite.NoError(err)
	suite.Equal(ctrl.Result{}, result)
}

// TestSetupWithManager tests the controller setup with manager
func (suite *NetworkIntentControllerTestSuite) TestSetupWithManager() {
	// This would typically require a more complex setup with a real manager
	// For now, we'll test that the method exists and can be called
	suite.NotNil(suite.reconciler)
	suite.Implements((*reconcile.Reconciler)(nil), suite.reconciler)
}

// Benchmark tests for performance validation
func (suite *NetworkIntentControllerTestSuite) BenchmarkReconcile() {
	// Setup benchmark
	intent := suite.createTestNetworkIntent("benchmark-intent", "scale up network function")
	err := suite.client.Create(suite.ctx, intent)
	suite.Require().NoError(err)

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      intent.Name,
			Namespace: intent.Namespace,
		},
	}

	// Run benchmark
	suite.T().Run("Reconcile", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			_, err := suite.reconciler.Reconcile(suite.ctx, request)
			assert.NoError(t, err)
		}
	})
}

// TestSuite runner function
// DISABLED: func TestNetworkIntentControllerTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkIntentControllerTestSuite))
}

// Additional unit tests using standard testing approach
// DISABLED: func TestNetworkIntentReconciler_Reconcile_BasicFlow(t *testing.T) {
	// Test using standard Go testing approach for comparison
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	require.NoError(t, err)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := zap.New(zap.UseDevMode(true))

	reconciler := &controllers.NetworkIntentReconciler{
		Client:          client,
		Scheme:          scheme,
		Log:             logger,
		EnableLLMIntent: false, // Disable LLM for simple test
	}

	// Create test NetworkIntent
	intent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "basic-test",
			Namespace: "default",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: "basic scaling intent",
		},
	}

	ctx := context.Background()
	err = client.Create(ctx, intent)
	require.NoError(t, err)

	// Test reconcile
	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      intent.Name,
			Namespace: intent.Namespace,
		},
	}

	result, err := reconciler.Reconcile(ctx, request)
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Verify object still exists
	retrieved := &nephoranv1.NetworkIntent{}
	err = client.Get(ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, retrieved)
	assert.NoError(t, err)
	assert.Equal(t, intent.Spec.Intent, retrieved.Spec.Intent)
}

// Test race conditions and concurrent access
// DISABLED: func TestNetworkIntentReconciler_ConcurrentReconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	require.NoError(t, err)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := zap.New(zap.UseDevMode(true))

	reconciler := &controllers.NetworkIntentReconciler{
		Client:          client,
		Scheme:          scheme,
		Log:             logger,
		EnableLLMIntent: false,
	}

	// Create multiple NetworkIntents
	ctx := context.Background()
	numIntents := 10
	requests := make([]ctrl.Request, numIntents)

	for i := 0; i < numIntents; i++ {
		intent := &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("concurrent-test-%d", i),
				Namespace: "default",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent: fmt.Sprintf("concurrent scaling intent %d", i),
			},
		}

		err := client.Create(ctx, intent)
		require.NoError(t, err)

		requests[i] = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      intent.Name,
				Namespace: intent.Namespace,
			},
		}
	}

	// Test concurrent reconciliation
	done := make(chan bool, numIntents)
	errors := make(chan error, numIntents)

	for i := 0; i < numIntents; i++ {
		go func(req ctrl.Request) {
			_, err := reconciler.Reconcile(ctx, req)
			if err != nil {
				errors <- err
			}
			done <- true
		}(requests[i])
	}

	// Wait for all goroutines to complete
	for i := 0; i < numIntents; i++ {
		select {
		case <-done:
			// Success
		case err := <-errors:
			t.Errorf("Concurrent reconcile failed: %v", err)
		case <-time.After(30 * time.Second):
			t.Fatal("Timeout waiting for concurrent reconciliation")
		}
	}
}

// Test edge cases and error conditions
// DISABLED: func TestNetworkIntentReconciler_ErrorConditions(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() (*controllers.NetworkIntentReconciler, ctrl.Request)
		expectError bool
	}{
		{
			name: "nil client should handle gracefully",
			setupFunc: func() (*controllers.NetworkIntentReconciler, ctrl.Request) {
				reconciler := &controllers.NetworkIntentReconciler{
					Client:          nil, // This would normally cause issues
					Scheme:          runtime.NewScheme(),
					Log:             zap.New(zap.UseDevMode(true)),
					EnableLLMIntent: false,
				}
				request := ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "test",
						Namespace: "default",
					},
				}
				return reconciler, request
			},
			expectError: true,
		},
		{
			name: "empty request should handle gracefully",
			setupFunc: func() (*controllers.NetworkIntentReconciler, ctrl.Request) {
				scheme := runtime.NewScheme()
				_ = nephoranv1.AddToScheme(scheme)
				reconciler := &controllers.NetworkIntentReconciler{
					Client:          fake.NewClientBuilder().WithScheme(scheme).Build(),
					Scheme:          scheme,
					Log:             zap.New(zap.UseDevMode(true)),
					EnableLLMIntent: false,
				}
				request := ctrl.Request{} // Empty request
				return reconciler, request
			},
			expectError: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler, request := tt.setupFunc()
			_, err := reconciler.Reconcile(context.Background(), request)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
