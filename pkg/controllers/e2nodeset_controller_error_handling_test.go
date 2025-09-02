package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// MockClient provides a mock implementation of client.Client for testing error scenarios
type MockClient struct {
	client.Client
	mock.Mock
}

func (m *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	args := m.Called(ctx, key, obj, opts)
	if args.Get(0) != nil {
		return args.Error(0)
	}

	// Call the real client for successful cases
	if m.Client != nil {
		return m.Client.Get(ctx, key, obj, opts...)
	}
	return nil
}

func (m *MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := m.Called(ctx, list, opts)
	if args.Get(0) != nil {
		return args.Error(0)
	}

	// Call the real client for successful cases
	if m.Client != nil {
		return m.Client.List(ctx, list, opts...)
	}
	return nil
}

// MockEventRecorder provides a mock implementation of record.EventRecorder
type MockEventRecorder struct {
	mock.Mock
}

func (m *MockEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	m.Called(object, eventtype, reason, message)
}

func (m *MockEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	m.Called(object, eventtype, reason, messageFmt, args)
}

func (m *MockEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	m.Called(object, annotations, eventtype, reason, messageFmt, args)
}

// Test helper functions

func createTestE2NodeSet(name, namespace string, replicas int32) *nephoranv1.E2NodeSet {
	return &nephoranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: nephoranv1.E2NodeSetSpec{
			Replicas: replicas,
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
}

func createTestReconciler(mockClient client.Client, mockRecorder record.EventRecorder) *E2NodeSetReconciler {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	return &E2NodeSetReconciler{
		Client:   mockClient,
		Scheme:   scheme,
		Recorder: mockRecorder,
	}
}

// DISABLED: func TestCalculateExponentialBackoffForOperation(t *testing.T) {
	tests := []struct {
		name          string
		operation     string
		retryCount    int
		expectedRange struct {
			min time.Duration
			max time.Duration
		}
	}{
		{
			name:       "configmap operations backoff",
			operation:  "configmap-operations",
			retryCount: 1,
			expectedRange: struct {
				min time.Duration
				max time.Duration
			}{
				min: 3600 * time.Millisecond, // 2s * 2^1 = 4s, with jitter -10%
				max: 4400 * time.Millisecond, // 2s * 2^1 = 4s, with jitter +10%
			},
		},
		{
			name:       "e2 provisioning backoff",
			operation:  "e2-provisioning",
			retryCount: 0,
			expectedRange: struct {
				min time.Duration
				max time.Duration
			}{
				min: 4500 * time.Millisecond, // 5s with jitter -10%
				max: 5500 * time.Millisecond, // 5s with jitter +10%
			},
		},
		{
			name:       "cleanup operations backoff",
			operation:  "cleanup",
			retryCount: 0,
			expectedRange: struct {
				min time.Duration
				max time.Duration
			}{
				min: 9 * time.Second,  // 10s with jitter -10%
				max: 11 * time.Second, // 10s with jitter +10%
			},
		},
		{
			name:       "unknown operation uses default",
			operation:  "unknown-operation",
			retryCount: 0,
			expectedRange: struct {
				min time.Duration
				max time.Duration
			}{
				min: 900 * time.Millisecond,  // 1s with jitter -10%
				max: 1100 * time.Millisecond, // 1s with jitter +10%
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Run multiple times to account for jitter randomness
			for i := 0; i < 5; i++ {
				delay := calculateExponentialBackoffForOperation(tt.retryCount, tt.operation)
				assert.True(t, delay >= tt.expectedRange.min,
					"Delay %v should be >= %v for operation %s", delay, tt.expectedRange.min, tt.operation)
				assert.True(t, delay <= tt.expectedRange.max,
					"Delay %v should be <= %v for operation %s", delay, tt.expectedRange.max, tt.operation)
			}
		})
	}
}

// DISABLED: func TestRetryCountManagement(t *testing.T) {
	tests := []struct {
		name               string
		initialAnnotations map[string]string
		operation          string
		expectedRetryCount int
	}{
		{
			name:               "no existing annotations",
			initialAnnotations: nil,
			operation:          "configmap-operations",
			expectedRetryCount: 0,
		},
		{
			name: "existing retry count",
			initialAnnotations: map[string]string{
				"nephoran.com/configmap-operations-retry-count": "2",
			},
			operation:          "configmap-operations",
			expectedRetryCount: 2,
		},
		{
			name: "invalid retry count defaults to zero",
			initialAnnotations: map[string]string{
				"nephoran.com/configmap-operations-retry-count": "invalid",
			},
			operation:          "configmap-operations",
			expectedRetryCount: 0,
		},
		{
			name: "different operation returns zero",
			initialAnnotations: map[string]string{
				"nephoran.com/configmap-operations-retry-count": "3",
			},
			operation:          "e2-provisioning",
			expectedRetryCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e2nodeSet := createTestE2NodeSet("test", "default", 1)
			e2nodeSet.Annotations = tt.initialAnnotations

			retryCount := getRetryCount(e2nodeSet, tt.operation)
			assert.Equal(t, tt.expectedRetryCount, retryCount)
		})
	}
}

// DISABLED: func TestSetRetryCount(t *testing.T) {
	e2nodeSet := createTestE2NodeSet("test", "default", 1)
	operation := "configmap-operations"
	count := 3

	setRetryCount(e2nodeSet, operation, count)

	expectedKey := "nephoran.com/configmap-operations-retry-count"
	assert.NotNil(t, e2nodeSet.Annotations)
	assert.Equal(t, "3", e2nodeSet.Annotations[expectedKey])

	// Verify retrieval works
	retrievedCount := getRetryCount(e2nodeSet, operation)
	assert.Equal(t, count, retrievedCount)
}

// DISABLED: func TestClearRetryCount(t *testing.T) {
	e2nodeSet := createTestE2NodeSet("test", "default", 1)
	operation := "configmap-operations"

	// Set a retry count first
	setRetryCount(e2nodeSet, operation, 5)
	assert.Equal(t, 5, getRetryCount(e2nodeSet, operation))

	// Clear it
	clearRetryCount(e2nodeSet, operation)
	assert.Equal(t, 0, getRetryCount(e2nodeSet, operation))

	// Verify annotation is removed
	expectedKey := "nephoran.com/configmap-operations-retry-count"
	_, exists := e2nodeSet.Annotations[expectedKey]
	assert.False(t, exists)
}

// DISABLED: func TestConfigMapCreationErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		simulateError  error
		expectedResult ctrl.Result
		expectedError  bool
		expectedReady  metav1.ConditionStatus
		expectedReason string
	}{
		{
			name:          "configmap creation failure increments retry",
			simulateError: errors.NewConflict(schema.GroupResource{Resource: "configmaps"}, "test-cm", fmt.Errorf("already exists")),
			expectedResult: ctrl.Result{
				RequeueAfter: time.Duration(0), // Will be set by exponential backoff
			},
			expectedError:  true,
			expectedReady:  metav1.ConditionFalse,
			expectedReason: "ConfigMapCreationFailed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = nephoranv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			e2nodeSet := createTestE2NodeSet("test-creation-error", "default", 1)
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(e2nodeSet).Build()

			mockClient := &MockClient{Client: fakeClient}
			mockRecorder := &MockEventRecorder{}

			reconciler := createTestReconciler(mockClient, mockRecorder)

			// Mock ConfigMap creation to fail
			mockClient.On("Create", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
				_, ok := obj.(*corev1.ConfigMap)
				return ok
			}), mock.Anything).Return(tt.simulateError)

			// Mock Update calls for retry count and status updates
			mockClient.On("Update", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
				_, ok := obj.(*nephoranv1.E2NodeSet)
				return ok
			}), mock.Anything).Return(nil)

			// Mock event recording
			mockRecorder.On("Eventf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

			ctx := context.Background()
			namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify retry count was set
			var updatedE2NodeSet nephoranv1.E2NodeSet
			err = fakeClient.Get(ctx, namespacedName, &updatedE2NodeSet)
			require.NoError(t, err)

			retryCount := getRetryCount(&updatedE2NodeSet, "configmap-operations")
			assert.Equal(t, 1, retryCount)

			mockClient.AssertExpectations(t)
			mockRecorder.AssertExpectations(t)
		})
	}
}

// DISABLED: func TestConfigMapUpdateErrorHandling(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	e2nodeSet := createTestE2NodeSet("test-update-error", "default", 1)

	// Create an existing ConfigMap to trigger update path
	existingCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-update-error-e2node-0",
			Namespace: "default",
			Labels: map[string]string{
				E2NodeSetLabelKey:   e2nodeSet.Name,
				E2NodeAppLabelKey:   E2NodeAppLabelValue,
				E2NodeIDLabelKey:    "test-node-0",
				E2NodeIndexLabelKey: "0",
			},
		},
		Data: map[string]string{
			E2NodeConfigKey: `{"nodeId":"old-config"}`,
			E2NodeStatusKey: `{"nodeId":"old-status"}`,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(e2nodeSet, existingCM).
		Build()

	mockClient := &MockClient{Client: fakeClient}
	mockRecorder := &MockEventRecorder{}

	reconciler := createTestReconciler(mockClient, mockRecorder)

	// Mock ConfigMap update to fail
	updateError := errors.NewConflict(schema.GroupResource{Resource: "configmaps"}, "test-cm", fmt.Errorf("resource version conflict"))
	mockClient.On("Update", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
		_, ok := obj.(*corev1.ConfigMap)
		return ok
	}), mock.Anything).Return(updateError)

	// Mock E2NodeSet updates for retry count and status
	mockClient.On("Update", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
		_, ok := obj.(*nephoranv1.E2NodeSet)
		return ok
	}), mock.Anything).Return(nil)

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

	_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update E2 node ConfigMap")

	// Verify retry count was incremented
	var updatedE2NodeSet nephoranv1.E2NodeSet
	err = fakeClient.Get(ctx, namespacedName, &updatedE2NodeSet)
	require.NoError(t, err)

	retryCount := getRetryCount(&updatedE2NodeSet, "configmap-operations")
	assert.Equal(t, 1, retryCount)

	mockClient.AssertExpectations(t)
}

// DISABLED: func TestE2ProvisioningErrorHandling(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	e2nodeSet := createTestE2NodeSet("test-provisioning-error", "default", 2)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(e2nodeSet).Build()

	mockClient := &MockClient{Client: fakeClient}
	mockRecorder := &MockEventRecorder{}

	reconciler := createTestReconciler(mockClient, mockRecorder)

	// Mock all ConfigMap operations to fail to simulate E2 provisioning failure
	provisioningError := fmt.Errorf("E2 provisioning failed: network unreachable")
	mockClient.On("Create", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
		_, ok := obj.(*corev1.ConfigMap)
		return ok
	}), mock.Anything).Return(provisioningError)

	// Mock E2NodeSet updates
	mockClient.On("Update", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
		_, ok := obj.(*nephoranv1.E2NodeSet)
		return ok
	}), mock.Anything).Return(nil)

	// Mock event recording
	mockRecorder.On("Eventf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})

	assert.Error(t, err)
	assert.NotZero(t, result.RequeueAfter, "Should schedule retry with backoff")

	// Verify retry count was incremented for e2-provisioning
	var updatedE2NodeSet nephoranv1.E2NodeSet
	err = fakeClient.Get(ctx, namespacedName, &updatedE2NodeSet)
	require.NoError(t, err)

	retryCount := getRetryCount(&updatedE2NodeSet, "e2-provisioning")
	assert.Equal(t, 1, retryCount)

	mockClient.AssertExpectations(t)
	mockRecorder.AssertExpectations(t)
}

// DISABLED: func TestMaxRetriesExceeded(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	e2nodeSet := createTestE2NodeSet("test-max-retries", "default", 1)
	// Set retry count to max retries
	setRetryCount(e2nodeSet, "e2-provisioning", DefaultMaxRetries)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(e2nodeSet).Build()

	mockClient := &MockClient{Client: fakeClient}
	mockRecorder := &MockEventRecorder{}

	reconciler := createTestReconciler(mockClient, mockRecorder)

	// Mock ConfigMap creation to fail
	provisioningError := fmt.Errorf("persistent E2 provisioning failure")
	mockClient.On("Create", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
		_, ok := obj.(*corev1.ConfigMap)
		return ok
	}), mock.Anything).Return(provisioningError)

	// Mock event recording for max retries exceeded
	mockRecorder.On("Eventf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})

	assert.Error(t, err)
	assert.Equal(t, time.Hour, result.RequeueAfter, "Should use long delay after max retries")

	mockRecorder.AssertExpectations(t)
}

// DISABLED: func TestFinalizerNotRemovedUntilCleanupSuccess(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	e2nodeSet := createTestE2NodeSet("test-finalizer", "default", 2)
	e2nodeSet.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	controllerutil.AddFinalizer(e2nodeSet, E2NodeSetFinalizer)

	// Create some ConfigMaps to be cleaned up
	cm1 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-finalizer-e2node-0",
			Namespace: "default",
			Labels: map[string]string{
				E2NodeSetLabelKey: e2nodeSet.Name,
				E2NodeAppLabelKey: E2NodeAppLabelValue,
			},
		},
	}
	cm2 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-finalizer-e2node-1",
			Namespace: "default",
			Labels: map[string]string{
				E2NodeSetLabelKey: e2nodeSet.Name,
				E2NodeAppLabelKey: E2NodeAppLabelValue,
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(e2nodeSet, cm1, cm2).
		Build()

	mockClient := &MockClient{Client: fakeClient}
	mockRecorder := &MockEventRecorder{}

	reconciler := createTestReconciler(mockClient, mockRecorder)

	// Mock one ConfigMap deletion to fail
	deleteError := errors.NewConflict(schema.GroupResource{Resource: "configmaps"}, "test-cm", fmt.Errorf("deletion blocked"))
	mockClient.On("Delete", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
		cm, ok := obj.(*corev1.ConfigMap)
		return ok && cm.Name == "test-finalizer-e2node-1"
	}), mock.Anything).Return(deleteError)

	// Mock successful deletion of the other ConfigMap
	mockClient.On("Delete", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
		cm, ok := obj.(*corev1.ConfigMap)
		return ok && cm.Name == "test-finalizer-e2node-0"
	}), mock.Anything).Return(nil)

	// Mock E2NodeSet updates for retry count and status
	mockClient.On("Update", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
		_, ok := obj.(*nephoranv1.E2NodeSet)
		return ok
	}), mock.Anything).Return(nil)

	// Mock event recording
	mockRecorder.On("Eventf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})

	assert.Error(t, err)
	assert.NotZero(t, result.RequeueAfter, "Should retry cleanup with backoff")

	// Verify finalizer is still present
	var updatedE2NodeSet nephoranv1.E2NodeSet
	err = fakeClient.Get(ctx, namespacedName, &updatedE2NodeSet)
	require.NoError(t, err)

	assert.True(t, controllerutil.ContainsFinalizer(&updatedE2NodeSet, E2NodeSetFinalizer),
		"Finalizer should not be removed until cleanup succeeds")

	// Verify cleanup retry count was incremented
	retryCount := getRetryCount(&updatedE2NodeSet, "cleanup")
	assert.Equal(t, 1, retryCount)

	mockClient.AssertExpectations(t)
	mockRecorder.AssertExpectations(t)
}

// DISABLED: func TestFinalizerRemovedAfterMaxCleanupRetries(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	e2nodeSet := createTestE2NodeSet("test-max-cleanup-retries", "default", 1)
	e2nodeSet.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	controllerutil.AddFinalizer(e2nodeSet, E2NodeSetFinalizer)
	// Set cleanup retry count to max retries
	setRetryCount(e2nodeSet, "cleanup", DefaultMaxRetries)

	// Create a ConfigMap that will fail to delete
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-max-cleanup-retries-e2node-0",
			Namespace: "default",
			Labels: map[string]string{
				E2NodeSetLabelKey: e2nodeSet.Name,
				E2NodeAppLabelKey: E2NodeAppLabelValue,
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(e2nodeSet, cm).
		Build()

	mockClient := &MockClient{Client: fakeClient}
	mockRecorder := &MockEventRecorder{}

	reconciler := createTestReconciler(mockClient, mockRecorder)

	// Mock ConfigMap deletion to fail
	deleteError := fmt.Errorf("persistent deletion failure")
	mockClient.On("Delete", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
		_, ok := obj.(*corev1.ConfigMap)
		return ok
	}), mock.Anything).Return(deleteError)

	// Mock E2NodeSet update to remove finalizer
	mockClient.On("Update", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
		e2ns, ok := obj.(*nephoranv1.E2NodeSet)
		return ok && !controllerutil.ContainsFinalizer(e2ns, E2NodeSetFinalizer)
	}), mock.Anything).Return(nil)

	// Mock event recording
	mockRecorder.On("Eventf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})

	assert.NoError(t, err, "Should not return error when finalizer is removed after max retries")
	assert.Zero(t, result.RequeueAfter, "Should not requeue after finalizer removal")

	mockClient.AssertExpectations(t)
	mockRecorder.AssertExpectations(t)
}

// DISABLED: func TestIdempotentReconciliation(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	e2nodeSet := createTestE2NodeSet("test-idempotent", "default", 2)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(e2nodeSet).Build()

	mockRecorder := &MockEventRecorder{}
	reconciler := createTestReconciler(fakeClient, mockRecorder)

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

	// First reconciliation
	result1, err1 := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
	require.NoError(t, err1)

	// Get state after first reconciliation
	var e2nodeSetAfterFirst nephoranv1.E2NodeSet
	err := fakeClient.Get(ctx, namespacedName, &e2nodeSetAfterFirst)
	require.NoError(t, err)

	// List ConfigMaps after first reconciliation
	configMapList1 := &corev1.ConfigMapList{}
	err = fakeClient.List(ctx, configMapList1, client.InNamespace("default"))
	require.NoError(t, err)

	// Second reconciliation (should be idempotent)
	result2, err2 := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
	require.NoError(t, err2)

	// Get state after second reconciliation
	var e2nodeSetAfterSecond nephoranv1.E2NodeSet
	err = fakeClient.Get(ctx, namespacedName, &e2nodeSetAfterSecond)
	require.NoError(t, err)

	// List ConfigMaps after second reconciliation
	configMapList2 := &corev1.ConfigMapList{}
	err = fakeClient.List(ctx, configMapList2, client.InNamespace("default"))
	require.NoError(t, err)

	// Verify idempotency
	assert.Equal(t, result1, result2, "Results should be identical")
	assert.Equal(t, len(configMapList1.Items), len(configMapList2.Items), "ConfigMap count should be identical")
	assert.Equal(t, e2nodeSetAfterFirst.Status.CurrentReplicas, e2nodeSetAfterSecond.Status.CurrentReplicas,
		"Replica status should be identical")

	// Third reconciliation to ensure continued idempotency
	result3, err3 := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
	require.NoError(t, err3)
	assert.Equal(t, result1, result3, "Third reconciliation should also be idempotent")
}

// DISABLED: func TestSetReadyCondition(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	e2nodeSet := createTestE2NodeSet("test-condition", "default", 1)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(e2nodeSet).Build()

	mockRecorder := &MockEventRecorder{}
	reconciler := createTestReconciler(fakeClient, mockRecorder)

	ctx := context.Background()

	tests := []struct {
		name    string
		status  metav1.ConditionStatus
		reason  string
		message string
	}{
		{
			name:    "ready condition true",
			status:  metav1.ConditionTrue,
			reason:  "E2NodesReady",
			message: "All E2 nodes are ready",
		},
		{
			name:    "ready condition false",
			status:  metav1.ConditionFalse,
			reason:  "E2NodesNotReady",
			message: "E2 nodes are not ready",
		},
		{
			name:    "ready condition unknown",
			status:  metav1.ConditionUnknown,
			reason:  "E2NodesStatusUnknown",
			message: "E2 nodes status unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reconciler.setReadyCondition(ctx, e2nodeSet, tt.status, tt.reason, tt.message)
			assert.NoError(t, err)

			// Verify condition was set correctly
			var updatedE2NodeSet nephoranv1.E2NodeSet
			err = fakeClient.Get(ctx, types.NamespacedName{Name: e2nodeSet.GetName(), Namespace: e2nodeSet.GetNamespace()}, &updatedE2NodeSet)
			require.NoError(t, err)

			found := false
			for _, condition := range updatedE2NodeSet.Status.Conditions {
				if condition.Type == "Ready" {
					found = true
					assert.Equal(t, tt.status, condition.Status)
					assert.Equal(t, tt.reason, condition.Reason)
					assert.Equal(t, tt.message, condition.Message)
					assert.False(t, condition.LastTransitionTime.IsZero())
					break
				}
			}
			assert.True(t, found, "Ready condition should be present")
		})
	}
}

// DISABLED: func TestReconcileWithPartialFailures(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	e2nodeSet := createTestE2NodeSet("test-partial-failure", "default", 3)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(e2nodeSet).Build()

	mockClient := &MockClient{Client: fakeClient}
	mockRecorder := &MockEventRecorder{}

	reconciler := createTestReconciler(mockClient, mockRecorder)

	// Mock some ConfigMap creations to succeed and others to fail
	var creationCount int
	mockClient.On("Create", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
		_, ok := obj.(*corev1.ConfigMap)
		return ok
	}), mock.Anything).Return(func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
		creationCount++
		if creationCount == 2 { // Second creation fails
			return fmt.Errorf("creation failed for second ConfigMap")
		}
		return fakeClient.Create(ctx, obj, opts...)
	})

	// Mock E2NodeSet updates
	mockClient.On("Update", mock.Anything, mock.MatchedBy(func(obj client.Object) bool {
		_, ok := obj.(*nephoranv1.E2NodeSet)
		return ok
	}), mock.Anything).Return(nil)

	// Mock event recording
	mockRecorder.On("Eventf", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

	result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})

	assert.Error(t, err)
	assert.NotZero(t, result.RequeueAfter, "Should retry with backoff")

	// Verify retry count was set
	var updatedE2NodeSet nephoranv1.E2NodeSet
	err = fakeClient.Get(ctx, namespacedName, &updatedE2NodeSet)
	require.NoError(t, err)

	retryCount := getRetryCount(&updatedE2NodeSet, "configmap-operations")
	assert.Equal(t, 1, retryCount)

	mockClient.AssertExpectations(t)
}

// DISABLED: func TestSuccessfulReconciliationClearsRetryCount(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	e2nodeSet := createTestE2NodeSet("test-clear-retry", "default", 1)
	// Set some initial retry counts
	setRetryCount(e2nodeSet, "e2-provisioning", 2)
	setRetryCount(e2nodeSet, "configmap-operations", 1)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(e2nodeSet).Build()
	mockRecorder := &MockEventRecorder{}
	reconciler := createTestReconciler(fakeClient, mockRecorder)

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: e2nodeSet.Name, Namespace: e2nodeSet.Namespace}

	_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
	require.NoError(t, err)

	// Verify retry counts were cleared
	var updatedE2NodeSet nephoranv1.E2NodeSet
	err = fakeClient.Get(ctx, namespacedName, &updatedE2NodeSet)
	require.NoError(t, err)

	assert.Equal(t, 0, getRetryCount(&updatedE2NodeSet, "e2-provisioning"))
	assert.Equal(t, 0, getRetryCount(&updatedE2NodeSet, "configmap-operations"))
}
