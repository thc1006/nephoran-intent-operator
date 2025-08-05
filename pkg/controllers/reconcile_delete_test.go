package controllers

import (
	"context"
	"net/http"
	"testing"
	"time"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/git/fake"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// SimpleDependencies is a simple implementation for testing
type SimpleDependencies struct {
	gitClient *fake.Client
}

func (d *SimpleDependencies) GetGitClient() git.ClientInterface {
	return d.gitClient
}

func (d *SimpleDependencies) GetLLMClient() shared.ClientInterface {
	return nil
}

func (d *SimpleDependencies) GetPackageGenerator() *nephio.PackageGenerator {
	return nil
}

func (d *SimpleDependencies) GetHTTPClient() *http.Client {
	return nil
}

func (d *SimpleDependencies) GetEventRecorder() record.EventRecorder {
	return &record.FakeRecorder{Events: make(chan string, 100)}
}

func TestReconcileDeleteWithFakeGitClient(t *testing.T) {
	// Create a scheme and add our types
	s := runtime.NewScheme()
	if err := nephoranv1.AddToScheme(s); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}
	if err := scheme.AddToScheme(s); err != nil {
		t.Fatalf("Failed to add core scheme: %v", err)
	}

	// Create a fake kubernetes client
	fakeClient := fakeclient.NewClientBuilder().WithScheme(s).Build()

	// Create a fake Git client
	fakeGitClient := fake.NewClient()

	// Create dependencies
	deps := &SimpleDependencies{
		gitClient: fakeGitClient,
	}

	// Create config
	config := &Config{
		MaxRetries:    3,
		RetryDelay:    time.Second,
		GitRepoURL:    "https://github.com/test/repo.git",
		GitDeployPath: "networkintents",
	}

	// Create reconciler
	reconciler, err := NewNetworkIntentReconciler(fakeClient, s, deps, config)
	if err != nil {
		t.Fatalf("Failed to create reconciler: %v", err)
	}

	t.Run("Should retain finalizer when Git operation fails", func(t *testing.T) {
		// Create NetworkIntent with finalizer and deletion timestamp
		networkIntent := &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-intent",
				Namespace:         "default",
				Finalizers:        []string{NetworkIntentFinalizer},
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent: "Test intent for deletion",
			},
		}

		// Create the NetworkIntent in the fake cluster
		if err := fakeClient.Create(context.TODO(), networkIntent); err != nil {
			t.Fatalf("Failed to create NetworkIntent: %v", err)
		}

		// Configure fake Git client to fail
		fakeGitClient.ShouldFailRemoveDirectory = true

		// Call reconcileDelete
		result, err := reconciler.reconcileDelete(context.TODO(), networkIntent)

		// Verify behavior
		if err != nil {
			t.Errorf("Expected no error from reconcileDelete on failure, got: %v", err)
		}

		if result.RequeueAfter <= 0 {
			t.Errorf("Expected retry to be scheduled, got RequeueAfter: %v", result.RequeueAfter)
		}

		// Get updated NetworkIntent
		updatedIntent := &nephoranv1.NetworkIntent{}
		if err := fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(networkIntent), updatedIntent); err != nil {
			t.Fatalf("Failed to get updated NetworkIntent: %v", err)
		}

		// Verify finalizer is still present
		if !containsFinalizer(updatedIntent.Finalizers, NetworkIntentFinalizer) {
			t.Errorf("Expected finalizer to be retained on Git failure")
		}

		// Verify condition is set correctly
		readyCondition := getConditionByType(updatedIntent.Status.Conditions, "Ready")
		if readyCondition == nil {
			t.Errorf("Expected Ready condition to be set")
		} else {
			if readyCondition.Status != metav1.ConditionFalse {
				t.Errorf("Expected Ready condition status to be False, got: %v", readyCondition.Status)
			}
			if readyCondition.Reason != "CleanupRetrying" {
				t.Errorf("Expected Ready condition reason to be CleanupRetrying, got: %v", readyCondition.Reason)
			}
		}

		// Verify Git client was called
		callHistory := fakeGitClient.GetCallHistory()
		if len(callHistory) == 0 {
			t.Errorf("Expected Git client to be called, but no calls were made")
		}
	})

	t.Run("Should remove finalizer when Git operation succeeds", func(t *testing.T) {
		// Reset the fake Git client
		fakeGitClient.Reset()

		// Create NetworkIntent with finalizer and deletion timestamp
		networkIntent := &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-intent-success",
				Namespace:         "default",
				Finalizers:        []string{NetworkIntentFinalizer},
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent: "Test intent for successful deletion",
			},
		}

		// Create the NetworkIntent in the fake cluster
		if err := fakeClient.Create(context.TODO(), networkIntent); err != nil {
			t.Fatalf("Failed to create NetworkIntent: %v", err)
		}

		// Configure fake Git client to succeed (default behavior)
		fakeGitClient.ShouldFailRemoveDirectory = false

		// Call reconcileDelete
		result, err := reconciler.reconcileDelete(context.TODO(), networkIntent)

		// Verify behavior
		if err != nil {
			t.Errorf("Expected no error from reconcileDelete on success, got: %v", err)
		}

		if result.Requeue || result.RequeueAfter > 0 {
			t.Errorf("Expected no requeue on success, got: %+v", result)
		}

		// Get updated NetworkIntent
		updatedIntent := &nephoranv1.NetworkIntent{}
		if err := fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(networkIntent), updatedIntent); err != nil {
			t.Fatalf("Failed to get updated NetworkIntent: %v", err)
		}

		// Verify finalizer is removed
		if containsFinalizer(updatedIntent.Finalizers, NetworkIntentFinalizer) {
			t.Errorf("Expected finalizer to be removed on Git success")
		}

		// Verify condition is set correctly
		readyCondition := getConditionByType(updatedIntent.Status.Conditions, "Ready")
		if readyCondition == nil {
			t.Errorf("Expected Ready condition to be set")
		} else {
			if readyCondition.Status != metav1.ConditionFalse {
				t.Errorf("Expected Ready condition status to be False, got: %v", readyCondition.Status)
			}
			if readyCondition.Reason != "CleanupCompleted" {
				t.Errorf("Expected Ready condition reason to be CleanupCompleted, got: %v", readyCondition.Reason)
			}
		}

		// Verify Git client was called
		callHistory := fakeGitClient.GetCallHistory()
		if len(callHistory) == 0 {
			t.Errorf("Expected Git client to be called, but no calls were made")
		}
	})
}

// Helper function to get condition by type
func getConditionByType(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}
