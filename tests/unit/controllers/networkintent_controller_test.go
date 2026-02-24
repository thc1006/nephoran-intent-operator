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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
	"github.com/thc1006/nephoran-intent-operator/controllers"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// buildScheme returns a scheme with the correct intent API group registered.
func buildScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, intentv1alpha1.AddToScheme(s))
	return s
}

// NetworkIntentControllerTestSuite implements a comprehensive test suite for the NetworkIntent controller.
type NetworkIntentControllerTestSuite struct {
	suite.Suite
	reconciler    *controllers.NetworkIntentReconciler
	client        client.Client
	scheme        *runtime.Scheme
	ctx           context.Context
	logger        logging.Logger
	llmServer     *httptest.Server
	testNamespace string
}

func (suite *NetworkIntentControllerTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	logging.InitGlobalLogger(logging.DebugLevel)
	suite.logger = logging.NewLogger(logging.ComponentController)
	suite.testNamespace = "nephoran-test"

	suite.scheme = runtime.NewScheme()
	suite.Require().NoError(intentv1alpha1.AddToScheme(suite.scheme))

	suite.setupMockLLMServer()
}

func (suite *NetworkIntentControllerTestSuite) TearDownSuite() {
	if suite.llmServer != nil {
		suite.llmServer.Close()
	}
}

func (suite *NetworkIntentControllerTestSuite) SetupTest() {
	suite.client = fake.NewClientBuilder().
		WithScheme(suite.scheme).
		WithStatusSubresource(&intentv1alpha1.NetworkIntent{}).
		Build()

	suite.reconciler = &controllers.NetworkIntentReconciler{
		Client:              suite.client,
		Scheme:              suite.scheme,
		Logger:              suite.logger,
		EnableLLMIntent:     true,
		EnableA1Integration: false, // A1 not available in unit tests
		LLMProcessorURL:     suite.llmServer.URL + "/process",
	}
}

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

func (suite *NetworkIntentControllerTestSuite) makeIntent(name, source string) *intentv1alpha1.NetworkIntent {
	return &intentv1alpha1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: suite.testNamespace,
		},
		Spec: intentv1alpha1.NetworkIntentSpec{
			Source:     source,
			IntentType: "scaling",
			Target:     "smf",
			Replicas:   1,
		},
	}
}

func (suite *NetworkIntentControllerTestSuite) reconcileRequest(name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: suite.testNamespace,
		},
	}
}

// TestReconcile_NewNetworkIntent — happy path: LLM enabled, mock server responds OK.
func (suite *NetworkIntentControllerTestSuite) TestReconcile_NewNetworkIntent() {
	intent := suite.makeIntent("test-intent", "scale up smf")
	suite.Require().NoError(suite.client.Create(suite.ctx, intent))

	result, err := suite.reconciler.Reconcile(suite.ctx, suite.reconcileRequest(intent.Name))
	suite.NoError(err)
	suite.Equal(ctrl.Result{}, result)

	updated := &intentv1alpha1.NetworkIntent{}
	suite.NoError(suite.client.Get(suite.ctx, types.NamespacedName{
		Name: intent.Name, Namespace: intent.Namespace,
	}, updated))
	suite.NotEmpty(updated.Status.Phase)
}

// TestReconcile_NonExistentNetworkIntent — object not found → no error (normal GC flow).
func (suite *NetworkIntentControllerTestSuite) TestReconcile_NonExistentNetworkIntent() {
	result, err := suite.reconciler.Reconcile(suite.ctx, suite.reconcileRequest("does-not-exist"))
	suite.NoError(err)
	suite.Equal(ctrl.Result{}, result)
}

// TestReconcile_LLMDisabled — with both LLM and A1 disabled, reconcile completes cleanly.
func (suite *NetworkIntentControllerTestSuite) TestReconcile_LLMDisabled() {
	suite.reconciler.EnableLLMIntent = false
	intent := suite.makeIntent("intent-no-llm", "scale down amf")
	suite.Require().NoError(suite.client.Create(suite.ctx, intent))

	result, err := suite.reconciler.Reconcile(suite.ctx, suite.reconcileRequest(intent.Name))
	suite.NoError(err)
	suite.Equal(ctrl.Result{}, result)
}

// TestReconcile_LLMServerDown — unreachable LLM server → status set to Error, no requeue error.
func (suite *NetworkIntentControllerTestSuite) TestReconcile_LLMServerDown() {
	suite.reconciler.LLMProcessorURL = "http://127.0.0.1:19999/process" // nothing listening
	intent := suite.makeIntent("intent-server-down", "scale smf")
	suite.Require().NoError(suite.client.Create(suite.ctx, intent))

	result, err := suite.reconciler.Reconcile(suite.ctx, suite.reconcileRequest(intent.Name))
	// Controller handles external service failures gracefully: updates status to Error, no error returned.
	suite.NoError(err)
	suite.Equal(ctrl.Result{}, result)
}

// TestReconcile_EmptySource — empty Source field → validation error reflected in status.
func (suite *NetworkIntentControllerTestSuite) TestReconcile_EmptySource() {
	intent := &intentv1alpha1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{Name: "empty-source", Namespace: suite.testNamespace},
		Spec:       intentv1alpha1.NetworkIntentSpec{IntentType: "scaling", Target: "smf"},
	}
	suite.Require().NoError(suite.client.Create(suite.ctx, intent))

	result, err := suite.reconciler.Reconcile(suite.ctx, suite.reconcileRequest(intent.Name))
	suite.NoError(err)
	suite.Equal(ctrl.Result{}, result)

	updated := &intentv1alpha1.NetworkIntent{}
	suite.NoError(suite.client.Get(suite.ctx, types.NamespacedName{
		Name: intent.Name, Namespace: intent.Namespace,
	}, updated))
	suite.Equal(intentv1alpha1.PhaseError, updated.Status.Phase)
}

// TestSetupWithManager verifies the reconciler implements the Reconciler interface.
func (suite *NetworkIntentControllerTestSuite) TestSetupWithManager() {
	suite.NotNil(suite.reconciler)
	suite.Implements((*reconcile.Reconciler)(nil), suite.reconciler)
}

func TestNetworkIntentControllerTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkIntentControllerTestSuite))
}

// ── Standalone table-driven tests ──────────────────────────────────────────────

func TestNetworkIntentReconciler_Reconcile_BasicFlow(t *testing.T) {
	logging.InitGlobalLogger(logging.DebugLevel)
	scheme := buildScheme(t)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&intentv1alpha1.NetworkIntent{}).
		Build()

	reconciler := &controllers.NetworkIntentReconciler{
		Client:              fakeClient,
		Scheme:              scheme,
		Logger:              logging.NewLogger(logging.ComponentController),
		EnableLLMIntent:     false,
		EnableA1Integration: false,
	}

	intent := &intentv1alpha1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{Name: "basic-test", Namespace: "default"},
		Spec: intentv1alpha1.NetworkIntentSpec{
			Source:     "basic scaling intent",
			IntentType: "scaling",
			Target:     "smf",
			Replicas:   2,
		},
	}
	ctx := context.Background()
	require.NoError(t, fakeClient.Create(ctx, intent))

	result, err := reconciler.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace},
	})
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	retrieved := &intentv1alpha1.NetworkIntent{}
	require.NoError(t, fakeClient.Get(ctx, types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace}, retrieved))
	assert.Equal(t, intentv1alpha1.PhaseValidated, retrieved.Status.Phase)
}

func TestNetworkIntentReconciler_ConcurrentReconcile(t *testing.T) {
	logging.InitGlobalLogger(logging.DebugLevel)
	scheme := buildScheme(t)
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&intentv1alpha1.NetworkIntent{}).
		Build()
	reconciler := &controllers.NetworkIntentReconciler{
		Client:              fakeClient,
		Scheme:              scheme,
		Logger:              logging.NewLogger(logging.ComponentController),
		EnableLLMIntent:     false,
		EnableA1Integration: false,
	}

	ctx := context.Background()
	const n = 10
	requests := make([]ctrl.Request, n)
	for i := range n {
		obj := &intentv1alpha1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("concurrent-%d", i),
				Namespace: "default",
			},
			Spec: intentv1alpha1.NetworkIntentSpec{
				Source:     fmt.Sprintf("intent %d", i),
				IntentType: "scaling",
				Target:     "smf",
			},
		}
		require.NoError(t, fakeClient.Create(ctx, obj))
		requests[i] = ctrl.Request{NamespacedName: types.NamespacedName{Name: obj.Name, Namespace: obj.Namespace}}
	}

	errs := make(chan error, n)
	done := make(chan struct{}, n)
	for _, req := range requests {
		go func(r ctrl.Request) {
			_, err := reconciler.Reconcile(ctx, r)
			if err != nil {
				errs <- err
			}
			done <- struct{}{}
		}(req)
	}

	for range n {
		select {
		case err := <-errs:
			t.Errorf("concurrent reconcile error: %v", err)
		case <-done:
		case <-time.After(30 * time.Second):
			t.Fatal("timeout waiting for concurrent reconciliation")
		}
	}
}

func TestNetworkIntentReconciler_ErrorConditions(t *testing.T) {
	logging.InitGlobalLogger(logging.DebugLevel)
	scheme := buildScheme(t)

	tests := []struct {
		name        string
		setup       func() (*controllers.NetworkIntentReconciler, ctrl.Request)
		expectError bool
	}{
		{
			name: "object not found returns no error",
			setup: func() (*controllers.NetworkIntentReconciler, ctrl.Request) {
				r := &controllers.NetworkIntentReconciler{
					Client:              fake.NewClientBuilder().WithScheme(scheme).Build(),
					Scheme:              scheme,
					Logger:              logging.NewLogger(logging.ComponentController),
					EnableA1Integration: false,
					EnableLLMIntent:     false,
				}
				return r, ctrl.Request{
					NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "default"},
				}
			},
			expectError: false,
		},
		{
			name: "empty namespace/name returns no error",
			setup: func() (*controllers.NetworkIntentReconciler, ctrl.Request) {
				r := &controllers.NetworkIntentReconciler{
					Client:              fake.NewClientBuilder().WithScheme(scheme).Build(),
					Scheme:              scheme,
					Logger:              logging.NewLogger(logging.ComponentController),
					EnableA1Integration: false,
					EnableLLMIntent:     false,
				}
				return r, ctrl.Request{}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reconciler, req := tt.setup()
			_, err := reconciler.Reconcile(context.Background(), req)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
