package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// FakeEventRecorder is a simple implementation that captures events for testing
type FakeEventRecorder struct {
	Events []string
}

func (f *FakeEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	f.Events = append(f.Events, fmt.Sprintf("%s: %s - %s", eventtype, reason, message))
}

func (f *FakeEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	message := fmt.Sprintf(messageFmt, args...)
	f.Events = append(f.Events, fmt.Sprintf("%s: %s - %s", eventtype, reason, message))
}

func (f *FakeEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	message := fmt.Sprintf(messageFmt, args...)
	f.Events = append(f.Events, fmt.Sprintf("Annotated %s: %s - %s", eventtype, reason, message))
}

func TestCompleteErrorHandlingWorkflow(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	// Create test E2NodeSet
	e2nodeSet := &nephoranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-error-workflow",
			Namespace: "default",
		},
		Spec: nephoranv1.E2NodeSetSpec{
			Replicas: 2,
			Template: nephoranv1.E2NodeTemplate{
				Spec: nephoranv1.E2NodeSpec{
					NodeID:             "test-node",
					E2InterfaceVersion: "v3.0",
					SupportedRANFunctions: []nephoranv1.RANFunction{
						{
							FunctionID:  1,
							Revision:    1,
							Description: "KPM Service Model",
							OID:         "1.3.6.1.4.1.53148.1.1.2.2",
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(e2nodeSet).
		Build()

	fakeRecorder := &FakeEventRecorder{}
	reconciler := &E2NodeSetReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Recorder: fakeRecorder,
	}

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

	t.Run("successful reconciliation with no errors", func(t *testing.T) {
		// First reconciliation should succeed
		result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
		require.NoError(t, err)

		// Verify ConfigMaps were created
		configMapList := &corev1.ConfigMapList{}
		err = fakeClient.List(ctx, configMapList)
		require.NoError(t, err)
		
		// Should have 2 ConfigMaps for 2 replicas
		createdConfigMaps := 0
		for _, cm := range configMapList.Items {
			if cm.Labels[E2NodeSetLabelKey] == e2nodeSet.Name {
				createdConfigMaps++
			}
		}
		assert.Equal(t, 2, createdConfigMaps)

		// Verify no retry counts are set
		var updatedE2NodeSet nephoranv1.E2NodeSet
		err = fakeClient.Get(ctx, namespacedName, &updatedE2NodeSet)
		require.NoError(t, err)

		assert.Equal(t, 0, getRetryCount(&updatedE2NodeSet, "configmap-operations"))
		assert.Equal(t, 0, getRetryCount(&updatedE2NodeSet, "e2-provisioning"))
	})
}

func TestErrorHandlingWithRetryLogic(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	t.Run("configmap creation failure with retry progression", func(t *testing.T) {
		e2nodeSet := &nephoranv1.E2NodeSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-retry-progression",
				Namespace: "default",
			},
			Spec: nephoranv1.E2NodeSetSpec{
				Replicas: 1,
				Template: nephoranv1.E2NodeTemplate{
					Spec: nephoranv1.E2NodeSpec{
						NodeID:             "test-node",
						E2InterfaceVersion: "v3.0",
						SupportedRANFunctions: []nephoranv1.RANFunction{
							{
								FunctionID:  1,
								Revision:    1,
								Description: "KPM Service Model",
								OID:         "1.3.6.1.4.1.53148.1.1.2.2",
							},
						},
					},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(e2nodeSet).
			Build()

		fakeRecorder := &FakeEventRecorder{}
		reconciler := &E2NodeSetReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Recorder: fakeRecorder,
		}

		ctx := context.Background()
		namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

		// Create a conflicting ConfigMap to force creation failure
		conflictCM := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-retry-progression-e2node-0",
				Namespace: "default",
				Labels: map[string]string{
					"conflicting": "true",
				},
			},
			Data: map[string]string{
				"conflict": "true",
			},
		}
		err := fakeClient.Create(ctx, conflictCM)
		require.NoError(t, err)

		// Simulate multiple failed reconciliations
		for attempt := 1; attempt <= DefaultMaxRetries; attempt++ {
			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			
			// Should return error and schedule retry
			assert.Error(t, err)
			assert.NotZero(t, result.RequeueAfter, "Should schedule retry with backoff")

			// Verify retry count is incremented
			var updatedE2NodeSet nephoranv1.E2NodeSet
			err = fakeClient.Get(ctx, namespacedName, &updatedE2NodeSet)
			require.NoError(t, err)

			retryCount := getRetryCount(&updatedE2NodeSet, "configmap-operations")
			assert.Equal(t, attempt, retryCount, "Retry count should match attempt number")

			// Verify Ready condition is False
			readyCondition := findConditionByType(updatedE2NodeSet.Status.Conditions, "Ready")
			if readyCondition != nil {
				assert.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				assert.Contains(t, readyCondition.Message, fmt.Sprintf("attempt %d/%d", attempt, DefaultMaxRetries))
			}
		}

		// After max retries, should still error but with long delay
		result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
		assert.Error(t, err)
		assert.Equal(t, time.Hour, result.RequeueAfter, "Should use long delay after max retries exceeded")

		// Verify max retries condition is set
		var finalE2NodeSet nephoranv1.E2NodeSet
		err = fakeClient.Get(ctx, namespacedName, &finalE2NodeSet)
		require.NoError(t, err)

		readyCondition := findConditionByType(finalE2NodeSet.Status.Conditions, "Ready")
		if readyCondition != nil {
			assert.Equal(t, metav1.ConditionFalse, readyCondition.Status)
			assert.Contains(t, readyCondition.Message, "after 3 retries")
		}

		// Verify events were recorded
		assert.NotEmpty(t, fakeRecorder.Events)
		hasRetryEvent := false
		hasMaxRetriesEvent := false
		for _, event := range fakeRecorder.Events {
			if contains(event, "ReconciliationRetrying") {
				hasRetryEvent = true
			}
			if contains(event, "ReconciliationFailedMaxRetries") {
				hasMaxRetriesEvent = true
			}
		}
		assert.True(t, hasRetryEvent, "Should have recorded retry events")
		assert.True(t, hasMaxRetriesEvent, "Should have recorded max retries event")
	})
}

func TestCleanupErrorHandlingWithFinalizers(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	t.Run("finalizer not removed until cleanup succeeds", func(t *testing.T) {
		e2nodeSet := &nephoranv1.E2NodeSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-cleanup-finalizer",
				Namespace:         "default",
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
				Finalizers:        []string{E2NodeSetFinalizer},
			},
			Spec: nephoranv1.E2NodeSetSpec{
				Replicas: 2,
			},
		}

		// Create ConfigMaps to be cleaned up
		cm1 := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cleanup-finalizer-e2node-0",
				Namespace: "default",
				Labels: map[string]string{
					E2NodeSetLabelKey: e2nodeSet.Name,
					E2NodeAppLabelKey: E2NodeAppLabelValue,
				},
			},
		}
		
		cm2 := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cleanup-finalizer-e2node-1",
				Namespace: "default",
				Labels: map[string]string{
					E2NodeSetLabelKey: e2nodeSet.Name,
					E2NodeAppLabelKey: E2NodeAppLabelValue,
				},
				// Add finalizer to block deletion
				Finalizers: []string{"test.nephoran.com/prevent-deletion"},
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(e2nodeSet, cm1, cm2).
			Build()

		fakeRecorder := &FakeEventRecorder{}
		reconciler := &E2NodeSetReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Recorder: fakeRecorder,
		}

		ctx := context.Background()
		namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

		// First cleanup attempt - should fail due to finalizer on cm2
		result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
		assert.Error(t, err)
		assert.NotZero(t, result.RequeueAfter, "Should retry cleanup with backoff")

		// Verify finalizer is still present
		var updatedE2NodeSet nephoranv1.E2NodeSet
		err = fakeClient.Get(ctx, namespacedName, &updatedE2NodeSet)
		require.NoError(t, err)
		assert.True(t, controllerutil.ContainsFinalizer(&updatedE2NodeSet, E2NodeSetFinalizer),
			"Finalizer should not be removed until cleanup succeeds")

		// Verify cleanup retry count was set
		assert.Equal(t, 1, getRetryCount(&updatedE2NodeSet, "cleanup"))

		// Remove the blocking finalizer to allow cleanup
		var cm2Updated corev1.ConfigMap
		err = fakeClient.Get(ctx, types.NamespacedName{Name: cm2.Name, Namespace: cm2.Namespace}, &cm2Updated)
		require.NoError(t, err)
		cm2Updated.Finalizers = []string{}
		err = fakeClient.Update(ctx, &cm2Updated)
		require.NoError(t, err)

		// Second cleanup attempt - should succeed
		result, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
		assert.NoError(t, err)
		assert.Zero(t, result.RequeueAfter, "Should not requeue after successful cleanup")

		// Verify finalizer was removed
		err = fakeClient.Get(ctx, namespacedName, &updatedE2NodeSet)
		require.NoError(t, err)
		assert.False(t, controllerutil.ContainsFinalizer(&updatedE2NodeSet, E2NodeSetFinalizer),
			"Finalizer should be removed after successful cleanup")
	})

	t.Run("finalizer removed after max cleanup retries", func(t *testing.T) {
		e2nodeSet := &nephoranv1.E2NodeSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-max-cleanup-retries",
				Namespace:         "default",
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
				Finalizers:        []string{E2NodeSetFinalizer},
				Annotations: map[string]string{
					"nephoran.com/cleanup-retry-count": fmt.Sprintf("%d", DefaultMaxRetries),
				},
			},
			Spec: nephoranv1.E2NodeSetSpec{
				Replicas: 1,
			},
		}

		// Create a ConfigMap that can't be deleted (due to missing permissions or external finalizers)
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-max-cleanup-retries-e2node-0",
				Namespace: "default",
				Labels: map[string]string{
					E2NodeSetLabelKey: e2nodeSet.Name,
					E2NodeAppLabelKey: E2NodeAppLabelValue,
				},
				Finalizers: []string{"external-system/prevent-deletion"},
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(e2nodeSet, cm).
			Build()

		fakeRecorder := &FakeEventRecorder{}
		reconciler := &E2NodeSetReconciler{
			Client:   fakeClient,
			Scheme:   scheme,
			Recorder: fakeRecorder,
		}

		ctx := context.Background()
		namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

		// Cleanup attempt after max retries - should remove finalizer to prevent stuck resource
		result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
		assert.NoError(t, err, "Should not return error when finalizer is removed after max retries")
		assert.Zero(t, result.RequeueAfter, "Should not requeue after finalizer removal")

		// Verify finalizer was removed despite cleanup failure
		var updatedE2NodeSet nephoranv1.E2NodeSet
		err = fakeClient.Get(ctx, namespacedName, &updatedE2NodeSet)
		require.NoError(t, err)
		assert.False(t, controllerutil.ContainsFinalizer(&updatedE2NodeSet, E2NodeSetFinalizer),
			"Finalizer should be removed after max cleanup retries to prevent stuck resource")

		// Verify appropriate events were recorded
		hasMaxRetriesEvent := false
		for _, event := range fakeRecorder.Events {
			if contains(event, "CleanupFailedMaxRetries") {
				hasMaxRetriesEvent = true
			}
		}
		assert.True(t, hasMaxRetriesEvent, "Should have recorded max cleanup retries event")
	})
}

func TestIdempotentReconciliation(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	e2nodeSet := &nephoranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-idempotent",
			Namespace: "default",
		},
		Spec: nephoranv1.E2NodeSetSpec{
			Replicas: 2,
			Template: nephoranv1.E2NodeTemplate{
				Spec: nephoranv1.E2NodeSpec{
					NodeID:             "test-node",
					E2InterfaceVersion: "v3.0",
					SupportedRANFunctions: []nephoranv1.RANFunction{
						{
							FunctionID:  1,
							Revision:    1,
							Description: "KPM Service Model", 
							OID:         "1.3.6.1.4.1.53148.1.1.2.2",
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(e2nodeSet).
		Build()

	fakeRecorder := &FakeEventRecorder{}
	reconciler := &E2NodeSetReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Recorder: fakeRecorder,
	}

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

	// First reconciliation
	result1, err1 := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
	require.NoError(t, err1)

	// Get state after first reconciliation
	var e2nodeSetAfterFirst nephoranv1.E2NodeSet
	err := fakeClient.Get(ctx, namespacedName, &e2nodeSetAfterFirst)
	require.NoError(t, err)

	configMapList1 := &corev1.ConfigMapList{}
	err = fakeClient.List(ctx, configMapList1)
	require.NoError(t, err)

	// Second reconciliation (should be idempotent)
	result2, err2 := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
	require.NoError(t, err2)

	// Get state after second reconciliation
	var e2nodeSetAfterSecond nephoranv1.E2NodeSet
	err = fakeClient.Get(ctx, namespacedName, &e2nodeSetAfterSecond)
	require.NoError(t, err)

	configMapList2 := &corev1.ConfigMapList{}
	err = fakeClient.List(ctx, configMapList2)
	require.NoError(t, err)

	// Verify idempotency
	assert.Equal(t, result1.RequeueAfter, result2.RequeueAfter, "Requeue timing should be identical")
	assert.Equal(t, len(configMapList1.Items), len(configMapList2.Items), "ConfigMap count should be identical")
	
	// Count ConfigMaps for this E2NodeSet
	count1 := countConfigMapsForE2NodeSet(configMapList1.Items, e2nodeSet.Name)
	count2 := countConfigMapsForE2NodeSet(configMapList2.Items, e2nodeSet.Name)
	assert.Equal(t, count1, count2, "E2NodeSet ConfigMap count should be identical")
	assert.Equal(t, 2, count1, "Should have exactly 2 ConfigMaps")

	// Third reconciliation to ensure continued idempotency
	result3, err3 := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
	require.NoError(t, err3)
	assert.Equal(t, result1.RequeueAfter, result3.RequeueAfter, "Third reconciliation should also be idempotent")
}

func TestSuccessfulReconciliationClearsRetryCount(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	e2nodeSet := &nephoranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-clear-retry-count",
			Namespace: "default",
			Annotations: map[string]string{
				"nephoran.com/e2-provisioning-retry-count":      "2",
				"nephoran.com/configmap-operations-retry-count": "1",
			},
		},
		Spec: nephoranv1.E2NodeSetSpec{
			Replicas: 1,
			Template: nephoranv1.E2NodeTemplate{
				Spec: nephoranv1.E2NodeSpec{
					NodeID:             "test-node",
					E2InterfaceVersion: "v3.0",
					SupportedRANFunctions: []nephoranv1.RANFunction{
						{
							FunctionID:  1,
							Revision:    1,
							Description: "KPM Service Model",
							OID:         "1.3.6.1.4.1.53148.1.1.2.2",
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(e2nodeSet).
		Build()

	fakeRecorder := &FakeEventRecorder{}
	reconciler := &E2NodeSetReconciler{
		Client:   fakeClient,
		Scheme:   scheme,
		Recorder: fakeRecorder,
	}

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

	// Verify initial retry counts are set
	var initialE2NodeSet nephoranv1.E2NodeSet
	err := fakeClient.Get(ctx, namespacedName, &initialE2NodeSet)
	require.NoError(t, err)
	assert.Equal(t, 2, getRetryCount(&initialE2NodeSet, "e2-provisioning"))
	assert.Equal(t, 1, getRetryCount(&initialE2NodeSet, "configmap-operations"))

	// Successful reconciliation
	result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
	require.NoError(t, err)

	// Verify retry counts were cleared
	var updatedE2NodeSet nephoranv1.E2NodeSet
	err = fakeClient.Get(ctx, namespacedName, &updatedE2NodeSet)
	require.NoError(t, err)

	assert.Equal(t, 0, getRetryCount(&updatedE2NodeSet, "e2-provisioning"),
		"E2 provisioning retry count should be cleared on success")
	assert.Equal(t, 0, getRetryCount(&updatedE2NodeSet, "configmap-operations"),
		"ConfigMap operations retry count should be cleared on success")

	// Verify ConfigMap was created
	configMapList := &corev1.ConfigMapList{}
	err = fakeClient.List(ctx, configMapList)
	require.NoError(t, err)

	count := countConfigMapsForE2NodeSet(configMapList.Items, e2nodeSet.Name)
	assert.Equal(t, 1, count, "Should have created 1 ConfigMap")
}

// Helper functions

func findConditionByType(conditions []nephoranv1.E2NodeSetCondition, conditionType string) *nephoranv1.E2NodeSetCondition {
	for i, condition := range conditions {
		if condition.Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr || 
		   len(s) > len(substr) && s[:len(substr)] == substr ||
		   len(s) > len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func countConfigMapsForE2NodeSet(configMaps []corev1.ConfigMap, e2nodeSetName string) int {
	count := 0
	for _, cm := range configMaps {
		if cm.Labels[E2NodeSetLabelKey] == e2nodeSetName {
			count++
		}
	}
	return count
}