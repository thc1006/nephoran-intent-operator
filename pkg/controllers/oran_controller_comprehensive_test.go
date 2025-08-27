package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/testutil"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/a1"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o1"
)

// fakeO1Adaptor implements O1Adaptor for testing
type fakeO1Adaptor struct {
	shouldFail      bool
	callCount       int
	lastManagedElem *nephoranv1.ManagedElement
	appliedConfigs  []string
}

func (f *fakeO1Adaptor) ApplyConfiguration(ctx context.Context, me *nephoranv1.ManagedElement) error {
	f.callCount++
	f.lastManagedElem = me
	f.appliedConfigs = append(f.appliedConfigs, me.Spec.O1Config)

	if f.shouldFail {
		return fmt.Errorf("fake O1 configuration failure")
	}
	return nil
}

func (f *fakeO1Adaptor) Reset() {
	f.shouldFail = false
	f.callCount = 0
	f.lastManagedElem = nil
	f.appliedConfigs = []string{}
}

// fakeA1Adaptor implements A1Adaptor for testing
type fakeA1Adaptor struct {
	shouldFail      bool
	callCount       int
	lastManagedElem *nephoranv1.ManagedElement
	appliedPolicies []runtime.RawExtension
}

func (f *fakeA1Adaptor) ApplyPolicy(ctx context.Context, me *nephoranv1.ManagedElement) error {
	f.callCount++
	f.lastManagedElem = me
	f.appliedPolicies = append(f.appliedPolicies, me.Spec.A1Policy)

	if f.shouldFail {
		return fmt.Errorf("fake A1 policy application failure")
	}
	return nil
}

func (f *fakeA1Adaptor) Reset() {
	f.shouldFail = false
	f.callCount = 0
	f.lastManagedElem = nil
	f.appliedPolicies = []runtime.RawExtension{}
}

func TestOranAdaptorReconciler_Reconcile(t *testing.T) {
	testCases := []struct {
		name            string
		managedElement  *nephoranv1.ManagedElement
		deployment      *appsv1.Deployment
		existingObjects []client.Object
		o1Setup         func(*fakeO1Adaptor)
		a1Setup         func(*fakeA1Adaptor)
		expectedResult  ctrl.Result
		expectedError   bool
		expectedCalls   func(*testing.T, *fakeO1Adaptor, *fakeA1Adaptor)
		expectedStatus  func(*testing.T, *nephoranv1.ManagedElement)
		description     string
	}{
		{
			name: "successful_reconcile_with_o1_and_a1",
			managedElement: &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-managed-element",
					Namespace: "default",
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "test-deployment",
					O1Config:       "netconf://example.com:830",
					A1Policy:       runtime.RawExtension{Raw: []byte(`{"policy_type": "test", "policy_id": "123"}`)},
				},
			},
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32PtrOran(3),
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 3,
				},
			},
			existingObjects: []client.Object{},
			o1Setup: func(o1 *fakeO1Adaptor) {
				// O1 should succeed
			},
			a1Setup: func(a1 *fakeA1Adaptor) {
				// A1 should succeed
			},
			expectedResult: ctrl.Result{},
			expectedError:  false,
			expectedCalls: func(t *testing.T, o1 *fakeO1Adaptor, a1 *fakeA1Adaptor) {
				assert.Equal(t, 1, o1.callCount, "O1 ApplyConfiguration should be called once")
				assert.Equal(t, 1, a1.callCount, "A1 ApplyPolicy should be called once")
				assert.Equal(t, "netconf://example.com:830", o1.appliedConfigs[0], "O1 config should match")
			},
			expectedStatus: func(t *testing.T, me *nephoranv1.ManagedElement) {
				// Check Ready condition
				readyCondition := testutil.GetCondition(me.Status.Conditions, typeReadyManagedElement)
				assert.NotNil(t, readyCondition, "Ready condition should exist")
				assert.Equal(t, metav1.ConditionTrue, readyCondition.Status, "Ready condition should be true")

				// Check O1 condition
				o1Condition := testutil.GetCondition(me.Status.Conditions, O1ConfiguredCondition)
				assert.NotNil(t, o1Condition, "O1 condition should exist")
				assert.Equal(t, metav1.ConditionTrue, o1Condition.Status, "O1 condition should be true")

				// Check A1 condition
				a1Condition := testutil.GetCondition(me.Status.Conditions, A1PolicyAppliedCondition)
				assert.NotNil(t, a1Condition, "A1 condition should exist")
				assert.Equal(t, metav1.ConditionTrue, a1Condition.Status, "A1 condition should be true")
			},
			description: "Should successfully reconcile with both O1 and A1 configurations",
		},
		{
			name: "deployment_not_ready",
			managedElement: &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-managed-element",
					Namespace: "default",
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "test-deployment",
					O1Config:       "netconf://example.com:830",
					A1Policy:       runtime.RawExtension{Raw: []byte(`{"policy_type": "test"}`)},
				},
			},
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32PtrOran(3),
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 1, // Not fully available
				},
			},
			existingObjects: []client.Object{},
			o1Setup:         func(o1 *fakeO1Adaptor) {},
			a1Setup:         func(a1 *fakeA1Adaptor) {},
			expectedResult:  ctrl.Result{},
			expectedError:   false,
			expectedCalls: func(t *testing.T, o1 *fakeO1Adaptor, a1 *fakeA1Adaptor) {
				assert.Equal(t, 0, o1.callCount, "O1 should not be called when deployment not ready")
				assert.Equal(t, 0, a1.callCount, "A1 should not be called when deployment not ready")
			},
			expectedStatus: func(t *testing.T, me *nephoranv1.ManagedElement) {
				readyCondition := getCondition(me.Status.Conditions, typeReadyManagedElement)
				assert.NotNil(t, readyCondition, "Ready condition should exist")
				assert.Equal(t, metav1.ConditionFalse, readyCondition.Status, "Ready condition should be false")
				assert.Equal(t, "Progressing", readyCondition.Reason, "Ready condition reason should be Progressing")
			},
			description: "Should not apply O1/A1 configurations when deployment is not ready",
		},
		{
			name: "deployment_not_found",
			managedElement: &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-managed-element",
					Namespace: "default",
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "nonexistent-deployment",
					O1Config:       "netconf://example.com:830",
				},
			},
			deployment:      nil, // No deployment
			existingObjects: []client.Object{},
			o1Setup:         func(o1 *fakeO1Adaptor) {},
			a1Setup:         func(a1 *fakeA1Adaptor) {},
			expectedResult:  ctrl.Result{},
			expectedError:   false,
			expectedCalls: func(t *testing.T, o1 *fakeO1Adaptor, a1 *fakeA1Adaptor) {
				assert.Equal(t, 0, o1.callCount, "O1 should not be called when deployment not found")
				assert.Equal(t, 0, a1.callCount, "A1 should not be called when deployment not found")
			},
			expectedStatus: func(t *testing.T, me *nephoranv1.ManagedElement) {
				readyCondition := getCondition(me.Status.Conditions, typeReadyManagedElement)
				assert.NotNil(t, readyCondition, "Ready condition should exist")
				assert.Equal(t, metav1.ConditionFalse, readyCondition.Status, "Ready condition should be false")
				assert.Equal(t, "DeploymentNotFound", readyCondition.Reason, "Ready condition reason should be DeploymentNotFound")
			},
			description: "Should handle deployment not found error",
		},
		{
			name: "o1_configuration_only",
			managedElement: &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-managed-element",
					Namespace: "default",
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "test-deployment",
					O1Config:       "netconf://example.com:830",
					// No A1Policy specified
				},
			},
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32PtrOran(2),
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 2,
				},
			},
			existingObjects: []client.Object{},
			o1Setup:         func(o1 *fakeO1Adaptor) {},
			a1Setup:         func(a1 *fakeA1Adaptor) {},
			expectedResult:  ctrl.Result{},
			expectedError:   false,
			expectedCalls: func(t *testing.T, o1 *fakeO1Adaptor, a1 *fakeA1Adaptor) {
				assert.Equal(t, 1, o1.callCount, "O1 should be called")
				assert.Equal(t, 0, a1.callCount, "A1 should not be called when no policy specified")
			},
			expectedStatus: func(t *testing.T, me *nephoranv1.ManagedElement) {
				readyCondition := getCondition(me.Status.Conditions, typeReadyManagedElement)
				assert.Equal(t, metav1.ConditionTrue, readyCondition.Status, "Ready condition should be true")

				o1Condition := getCondition(me.Status.Conditions, O1ConfiguredCondition)
				assert.Equal(t, metav1.ConditionTrue, o1Condition.Status, "O1 condition should be true")

				// A1 condition should not exist
				a1Condition := getCondition(me.Status.Conditions, A1PolicyAppliedCondition)
				assert.Nil(t, a1Condition, "A1 condition should not exist when no policy specified")
			},
			description: "Should apply only O1 configuration when A1 policy not specified",
		},
		{
			name: "a1_policy_only",
			managedElement: &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-managed-element",
					Namespace: "default",
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "test-deployment",
					// No O1Config specified
					A1Policy: runtime.RawExtension{Raw: []byte(`{"policy_type": "test"}`)},
				},
			},
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32PtrOran(1),
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 1,
				},
			},
			existingObjects: []client.Object{},
			o1Setup:         func(o1 *fakeO1Adaptor) {},
			a1Setup:         func(a1 *fakeA1Adaptor) {},
			expectedResult:  ctrl.Result{},
			expectedError:   false,
			expectedCalls: func(t *testing.T, o1 *fakeO1Adaptor, a1 *fakeA1Adaptor) {
				assert.Equal(t, 0, o1.callCount, "O1 should not be called when no config specified")
				assert.Equal(t, 1, a1.callCount, "A1 should be called")
			},
			expectedStatus: func(t *testing.T, me *nephoranv1.ManagedElement) {
				readyCondition := getCondition(me.Status.Conditions, typeReadyManagedElement)
				assert.Equal(t, metav1.ConditionTrue, readyCondition.Status, "Ready condition should be true")

				// O1 condition should not exist
				o1Condition := getCondition(me.Status.Conditions, O1ConfiguredCondition)
				assert.Nil(t, o1Condition, "O1 condition should not exist when no config specified")

				a1Condition := getCondition(me.Status.Conditions, A1PolicyAppliedCondition)
				assert.Equal(t, metav1.ConditionTrue, a1Condition.Status, "A1 condition should be true")
			},
			description: "Should apply only A1 policy when O1 config not specified",
		},
		{
			name: "o1_configuration_failure",
			managedElement: &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-managed-element",
					Namespace: "default",
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "test-deployment",
					O1Config:       "netconf://example.com:830",
					A1Policy:       runtime.RawExtension{Raw: []byte(`{"policy_type": "test"}`)},
				},
			},
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32PtrOran(1),
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 1,
				},
			},
			existingObjects: []client.Object{},
			o1Setup: func(o1 *fakeO1Adaptor) {
				o1.shouldFail = true
			},
			a1Setup:        func(a1 *fakeA1Adaptor) {},
			expectedResult: ctrl.Result{},
			expectedError:  false,
			expectedCalls: func(t *testing.T, o1 *fakeO1Adaptor, a1 *fakeA1Adaptor) {
				assert.Equal(t, 1, o1.callCount, "O1 should be called even if it fails")
				assert.Equal(t, 1, a1.callCount, "A1 should still be called after O1 failure")
			},
			expectedStatus: func(t *testing.T, me *nephoranv1.ManagedElement) {
				readyCondition := getCondition(me.Status.Conditions, typeReadyManagedElement)
				assert.Equal(t, metav1.ConditionTrue, readyCondition.Status, "Ready condition should still be true")

				o1Condition := getCondition(me.Status.Conditions, O1ConfiguredCondition)
				assert.Equal(t, metav1.ConditionFalse, o1Condition.Status, "O1 condition should be false on failure")
				assert.Equal(t, "Failed", o1Condition.Reason, "O1 condition reason should be Failed")

				a1Condition := getCondition(me.Status.Conditions, A1PolicyAppliedCondition)
				assert.Equal(t, metav1.ConditionTrue, a1Condition.Status, "A1 condition should be true")
			},
			description: "Should continue with A1 configuration even if O1 fails",
		},
		{
			name: "a1_policy_failure",
			managedElement: &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-managed-element",
					Namespace: "default",
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "test-deployment",
					O1Config:       "netconf://example.com:830",
					A1Policy:       runtime.RawExtension{Raw: []byte(`{"policy_type": "test"}`)},
				},
			},
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32PtrOran(1),
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 1,
				},
			},
			existingObjects: []client.Object{},
			o1Setup:         func(o1 *fakeO1Adaptor) {},
			a1Setup: func(a1 *fakeA1Adaptor) {
				a1.shouldFail = true
			},
			expectedResult: ctrl.Result{},
			expectedError:  false,
			expectedCalls: func(t *testing.T, o1 *fakeO1Adaptor, a1 *fakeA1Adaptor) {
				assert.Equal(t, 1, o1.callCount, "O1 should be called")
				assert.Equal(t, 1, a1.callCount, "A1 should be called even if it fails")
			},
			expectedStatus: func(t *testing.T, me *nephoranv1.ManagedElement) {
				readyCondition := getCondition(me.Status.Conditions, typeReadyManagedElement)
				assert.Equal(t, metav1.ConditionTrue, readyCondition.Status, "Ready condition should be true")

				o1Condition := getCondition(me.Status.Conditions, O1ConfiguredCondition)
				assert.Equal(t, metav1.ConditionTrue, o1Condition.Status, "O1 condition should be true")

				a1Condition := getCondition(me.Status.Conditions, A1PolicyAppliedCondition)
				assert.Equal(t, metav1.ConditionFalse, a1Condition.Status, "A1 condition should be false on failure")
				assert.Equal(t, "Failed", a1Condition.Reason, "A1 condition reason should be Failed")
			},
			description: "Should handle A1 policy application failure",
		},
		{
			name: "resource_not_found",
			managedElement: &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nonexistent-managed-element",
					Namespace: "default",
				},
			},
			deployment:      nil,
			existingObjects: []client.Object{}, // Don't add the ManagedElement
			o1Setup:         func(o1 *fakeO1Adaptor) {},
			a1Setup:         func(a1 *fakeA1Adaptor) {},
			expectedResult:  ctrl.Result{},
			expectedError:   false,
			expectedCalls: func(t *testing.T, o1 *fakeO1Adaptor, a1 *fakeA1Adaptor) {
				assert.Equal(t, 0, o1.callCount, "O1 should not be called for nonexistent resource")
				assert.Equal(t, 0, a1.callCount, "A1 should not be called for nonexistent resource")
			},
			expectedStatus: func(t *testing.T, me *nephoranv1.ManagedElement) {
				// No status checks for nonexistent resource
			},
			description: "Should handle resource not found gracefully",
		},
		{
			name: "empty_o1_config_skipped",
			managedElement: &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-managed-element",
					Namespace: "default",
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "test-deployment",
					O1Config:       "", // Empty O1 config
					A1Policy:       runtime.RawExtension{Raw: []byte(`{"policy_type": "test"}`)},
				},
			},
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32PtrOran(1),
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 1,
				},
			},
			existingObjects: []client.Object{},
			o1Setup:         func(o1 *fakeO1Adaptor) {},
			a1Setup:         func(a1 *fakeA1Adaptor) {},
			expectedResult:  ctrl.Result{},
			expectedError:   false,
			expectedCalls: func(t *testing.T, o1 *fakeO1Adaptor, a1 *fakeA1Adaptor) {
				assert.Equal(t, 0, o1.callCount, "O1 should not be called for empty config")
				assert.Equal(t, 1, a1.callCount, "A1 should be called")
			},
			expectedStatus: func(t *testing.T, me *nephoranv1.ManagedElement) {
				readyCondition := getCondition(me.Status.Conditions, typeReadyManagedElement)
				assert.Equal(t, metav1.ConditionTrue, readyCondition.Status, "Ready condition should be true")

				a1Condition := getCondition(me.Status.Conditions, A1PolicyAppliedCondition)
				assert.Equal(t, metav1.ConditionTrue, a1Condition.Status, "A1 condition should be true")
			},
			description: "Should skip O1 configuration when config is empty",
		},
		{
			name: "nil_a1_policy_skipped",
			managedElement: &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-managed-element",
					Namespace: "default",
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "test-deployment",
					O1Config:       "netconf://example.com:830",
					A1Policy:       runtime.RawExtension{Raw: nil}, // Nil A1 policy
				},
			},
			deployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32PtrOran(1),
				},
				Status: appsv1.DeploymentStatus{
					AvailableReplicas: 1,
				},
			},
			existingObjects: []client.Object{},
			o1Setup:         func(o1 *fakeO1Adaptor) {},
			a1Setup:         func(a1 *fakeA1Adaptor) {},
			expectedResult:  ctrl.Result{},
			expectedError:   false,
			expectedCalls: func(t *testing.T, o1 *fakeO1Adaptor, a1 *fakeA1Adaptor) {
				assert.Equal(t, 1, o1.callCount, "O1 should be called")
				assert.Equal(t, 0, a1.callCount, "A1 should not be called for nil policy")
			},
			expectedStatus: func(t *testing.T, me *nephoranv1.ManagedElement) {
				readyCondition := getCondition(me.Status.Conditions, typeReadyManagedElement)
				assert.Equal(t, metav1.ConditionTrue, readyCondition.Status, "Ready condition should be true")

				o1Condition := getCondition(me.Status.Conditions, O1ConfiguredCondition)
				assert.Equal(t, metav1.ConditionTrue, o1Condition.Status, "O1 condition should be true")
			},
			description: "Should skip A1 policy when policy is nil",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the scheme
			s := scheme.Scheme
			err := nephoranv1.AddToScheme(s)
			require.NoError(t, err)

			// Create fake adaptors
			o1Adaptor := &fakeO1Adaptor{}
			a1Adaptor := &fakeA1Adaptor{}

			// Apply test-specific setup
			tc.o1Setup(o1Adaptor)
			tc.a1Setup(a1Adaptor)

			// Add objects to the fake client
			objects := tc.existingObjects
			if tc.name != "resource_not_found" {
				objects = append(objects, tc.managedElement)
			}
			if tc.deployment != nil {
				objects = append(objects, tc.deployment)
			}

			// Create fake client with existing objects
			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(objects...).
				WithStatusSubresource(&nephoranv1.ManagedElement{}).
				Build()

			// Create reconciler
			reconciler := &OranAdaptorReconciler{
				Client:    fakeClient,
				Scheme:    s,
				O1Adaptor: o1Adaptor,
				A1Adaptor: a1Adaptor,
			}

			// Execute reconcile
			ctx := context.Background()
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.managedElement.Name,
					Namespace: tc.managedElement.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)

			// Verify results
			if tc.expectedError {
				assert.Error(t, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
			}

			assert.Equal(t, tc.expectedResult, result, tc.description)

			// Verify expected calls
			tc.expectedCalls(t, o1Adaptor, a1Adaptor)

			// Verify status if object exists
			if tc.name != "resource_not_found" {
				var updatedME nephoranv1.ManagedElement
				err = fakeClient.Get(ctx, req.NamespacedName, &updatedME)
				if err == nil {
					tc.expectedStatus(t, &updatedME)
				}
			}
		})
	}
}

func TestOranAdaptorReconciler_ConcurrentReconciles(t *testing.T) {
	// Test concurrent reconcile operations
	s := scheme.Scheme
	err := nephoranv1.AddToScheme(s)
	require.NoError(t, err)

	me := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-managed-element",
			Namespace: "default",
		},
		Spec: nephoranv1.ManagedElementSpec{
			DeploymentName: "test-deployment",
			O1Config:       "netconf://example.com:830",
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32PtrOran(1),
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(me, deployment).
		WithStatusSubresource(&nephoranv1.ManagedElement{}).
		Build()

	reconciler := &OranAdaptorReconciler{
		Client:    fakeClient,
		Scheme:    s,
		O1Adaptor: &fakeO1Adaptor{},
		A1Adaptor: &fakeA1Adaptor{},
	}

	ctx := context.Background()
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      me.Name,
			Namespace: me.Namespace,
		},
	}

	// Run concurrent reconciles
	done := make(chan bool, 2)
	errors := make(chan error, 2)

	for i := 0; i < 2; i++ {
		go func() {
			defer func() { done <- true }()
			_, err := reconciler.Reconcile(ctx, req)
			if err != nil {
				errors <- err
			}
		}()
	}

	// Wait for completion
	for i := 0; i < 2; i++ {
		<-done
	}

	// Check for errors
	select {
	case err := <-errors:
		t.Logf("Concurrent reconcile error (may be acceptable): %v", err)
	default:
		// No errors
	}

	// Verify final state consistency
	var finalME nephoranv1.ManagedElement
	err = fakeClient.Get(ctx, req.NamespacedName, &finalME)
	require.NoError(t, err)

	// Should have Ready condition
	readyCondition := getCondition(finalME.Status.Conditions, typeReadyManagedElement)
	assert.NotNil(t, readyCondition, "Ready condition should exist")
}

func TestOranAdaptorReconciler_EdgeCases(t *testing.T) {
	testCases := []struct {
		name          string
		setup         func() (*OranAdaptorReconciler, *nephoranv1.ManagedElement, context.Context)
		expectedError bool
		description   string
	}{
		{
			name: "nil_o1_adaptor",
			setup: func() (*OranAdaptorReconciler, *nephoranv1.ManagedElement, context.Context) {
				s := scheme.Scheme
				_ = nephoranv1.AddToScheme(s)

				me := &nephoranv1.ManagedElement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-managed-element",
						Namespace: "default",
					},
					Spec: nephoranv1.ManagedElementSpec{
						DeploymentName: "test-deployment",
						O1Config:       "netconf://example.com:830",
					},
				}

				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment",
						Namespace: "default",
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: int32PtrOran(1),
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
					},
				}

				fakeClient := fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(me, deployment).
					WithStatusSubresource(&nephoranv1.ManagedElement{}).
					Build()

				reconciler := &OranAdaptorReconciler{
					Client:    fakeClient,
					Scheme:    s,
					O1Adaptor: nil, // Nil O1Adaptor
					A1Adaptor: &fakeA1Adaptor{},
				}

				return reconciler, me, context.Background()
			},
			expectedError: false, // Should handle nil adaptor gracefully by skipping O1 config
			description:   "Should handle nil O1Adaptor gracefully",
		},
		{
			name: "context_cancellation",
			setup: func() (*OranAdaptorReconciler, *nephoranv1.ManagedElement, context.Context) {
				s := scheme.Scheme
				_ = nephoranv1.AddToScheme(s)

				me := &nephoranv1.ManagedElement{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-managed-element",
						Namespace: "default",
					},
					Spec: nephoranv1.ManagedElementSpec{
						DeploymentName: "test-deployment",
					},
				}

				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-deployment",
						Namespace: "default",
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: int32PtrOran(1),
					},
					Status: appsv1.DeploymentStatus{
						AvailableReplicas: 1,
					},
				}

				fakeClient := fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(me, deployment).
					WithStatusSubresource(&nephoranv1.ManagedElement{}).
					Build()

				reconciler := &OranAdaptorReconciler{
					Client:    fakeClient,
					Scheme:    s,
					O1Adaptor: &fakeO1Adaptor{},
					A1Adaptor: &fakeA1Adaptor{},
				}

				// Create cancelled context
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return reconciler, me, ctx
			},
			expectedError: false, // Should handle context cancellation gracefully
			description:   "Should handle context cancellation gracefully",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reconciler, me, ctx := tc.setup()

			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      me.Name,
					Namespace: me.Namespace,
				},
			}

			_, err := reconciler.Reconcile(ctx, req)

			if tc.expectedError {
				assert.Error(t, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
			}
		})
	}
}

// Helper functions

func int32PtrOran(i int32) *int32 {
	return &i
}
