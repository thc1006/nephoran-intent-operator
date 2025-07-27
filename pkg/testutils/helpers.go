package testutils

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// TestTimeout defines the default timeout for test operations
const TestTimeout = 30 * time.Second

// TestInterval defines the default polling interval for test operations
const TestInterval = 250 * time.Millisecond

// CreateNamespace creates a test namespace
func CreateNamespace(ctx context.Context, k8sClient client.Client, name string) *corev1.Namespace {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"test-namespace": "true",
			},
		},
	}

	Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
	return namespace
}

// DeleteNamespace deletes a test namespace
func DeleteNamespace(ctx context.Context, k8sClient client.Client, name string) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	// Use background deletion to speed up tests
	deletePolicy := metav1.DeletePropagationBackground
	deleteOptions := &client.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	_ = k8sClient.Delete(ctx, namespace, deleteOptions)
}

// WaitForNetworkIntentPhase waits for a NetworkIntent to reach a specific phase
func WaitForNetworkIntentPhase(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName, expectedPhase string) {
	Eventually(func() string {
		ni := &nephoranv1.NetworkIntent{}
		err := k8sClient.Get(ctx, namespacedName, ni)
		if err != nil {
			return ""
		}
		return ni.Status.Phase
	}, TestTimeout, TestInterval).Should(Equal(expectedPhase))
}

// WaitForNetworkIntentCondition waits for a NetworkIntent to have a specific condition
func WaitForNetworkIntentCondition(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName, conditionType string, status metav1.ConditionStatus) {
	Eventually(func() metav1.ConditionStatus {
		ni := &nephoranv1.NetworkIntent{}
		err := k8sClient.Get(ctx, namespacedName, ni)
		if err != nil {
			return metav1.ConditionUnknown
		}

		for _, condition := range ni.Status.Conditions {
			if condition.Type == conditionType {
				return condition.Status
			}
		}
		return metav1.ConditionUnknown
	}, TestTimeout, TestInterval).Should(Equal(status))
}

// WaitForE2NodeSetReady waits for an E2NodeSet to have the expected number of ready replicas
func WaitForE2NodeSetReady(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName, expectedReplicas int32) {
	Eventually(func() int32 {
		e2ns := &nephoranv1.E2NodeSet{}
		err := k8sClient.Get(ctx, namespacedName, e2ns)
		if err != nil {
			return -1
		}
		return e2ns.Status.ReadyReplicas
	}, TestTimeout, TestInterval).Should(Equal(expectedReplicas))
}

// WaitForConfigMapCount waits for a specific number of ConfigMaps with a given label selector
func WaitForConfigMapCount(ctx context.Context, k8sClient client.Client, namespace string, labelSelector map[string]string, expectedCount int) {
	Eventually(func() int {
		configMapList := &corev1.ConfigMapList{}
		listOptions := []client.ListOption{
			client.InNamespace(namespace),
			client.MatchingLabels(labelSelector),
		}

		err := k8sClient.List(ctx, configMapList, listOptions...)
		if err != nil {
			return -1
		}

		return len(configMapList.Items)
	}, TestTimeout, TestInterval).Should(Equal(expectedCount))
}

// GetNetworkIntent retrieves a NetworkIntent resource
func GetNetworkIntent(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName) *nephoranv1.NetworkIntent {
	ni := &nephoranv1.NetworkIntent{}
	Expect(k8sClient.Get(ctx, namespacedName, ni)).To(Succeed())
	return ni
}

// GetE2NodeSet retrieves an E2NodeSet resource
func GetE2NodeSet(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName) *nephoranv1.E2NodeSet {
	e2ns := &nephoranv1.E2NodeSet{}
	Expect(k8sClient.Get(ctx, namespacedName, e2ns)).To(Succeed())
	return e2ns
}

// UpdateNetworkIntentStatus updates the status of a NetworkIntent
func UpdateNetworkIntentStatus(ctx context.Context, k8sClient client.Client, ni *nephoranv1.NetworkIntent) {
	Expect(k8sClient.Status().Update(ctx, ni)).To(Succeed())
}

// UpdateE2NodeSetStatus updates the status of an E2NodeSet
func UpdateE2NodeSetStatus(ctx context.Context, k8sClient client.Client, e2ns *nephoranv1.E2NodeSet) {
	Expect(k8sClient.Status().Update(ctx, e2ns)).To(Succeed())
}

// CreateConfigMap creates a test ConfigMap
func CreateConfigMap(ctx context.Context, k8sClient client.Client, name, namespace string, data map[string]string, labels map[string]string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	}

	Expect(k8sClient.Create(ctx, cm)).To(Succeed())
	return cm
}

// DeleteConfigMap deletes a ConfigMap
func DeleteConfigMap(ctx context.Context, k8sClient client.Client, name, namespace string) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	_ = k8sClient.Delete(ctx, cm)
}

// GenerateUniqueNamespace generates a unique namespace name for testing
func GenerateUniqueNamespace(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// GenerateUniqueName generates a unique resource name for testing
func GenerateUniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// AssertNetworkIntentHasCondition asserts that a NetworkIntent has a specific condition
func AssertNetworkIntentHasCondition(ni *nephoranv1.NetworkIntent, conditionType string, status metav1.ConditionStatus, reason string) {
	found := false
	for _, condition := range ni.Status.Conditions {
		if condition.Type == conditionType {
			found = true
			Expect(condition.Status).To(Equal(status), "Condition %s should have status %s", conditionType, status)
			if reason != "" {
				Expect(condition.Reason).To(Equal(reason), "Condition %s should have reason %s", conditionType, reason)
			}
			break
		}
	}
	Expect(found).To(BeTrue(), "NetworkIntent should have condition %s", conditionType)
}

// AssertE2NodeSetHasOwnerReference asserts that ConfigMaps have the correct owner reference
func AssertE2NodeSetHasOwnerReference(ctx context.Context, k8sClient client.Client, e2ns *nephoranv1.E2NodeSet, expectedConfigMaps int) {
	configMapList := &corev1.ConfigMapList{}
	listOptions := []client.ListOption{
		client.InNamespace(e2ns.Namespace),
		client.MatchingLabels(map[string]string{
			"app.kubernetes.io/name":       "e2-node-simulator",
			"app.kubernetes.io/managed-by": "e2nodeset-controller",
			"e2nodeset":                    e2ns.Name,
		}),
	}

	Expect(k8sClient.List(ctx, configMapList, listOptions...)).To(Succeed())
	Expect(len(configMapList.Items)).To(Equal(expectedConfigMaps))

	for _, cm := range configMapList.Items {
		found := false
		for _, ownerRef := range cm.OwnerReferences {
			if ownerRef.UID == e2ns.UID && ownerRef.Kind == "E2NodeSet" {
				found = true
				Expect(ownerRef.Controller).ToNot(BeNil())
				Expect(*ownerRef.Controller).To(BeTrue())
				break
			}
		}
		Expect(found).To(BeTrue(), "ConfigMap %s should have E2NodeSet as owner", cm.Name)
	}
}

// CleanupTestResources performs cleanup of test resources in a namespace
func CleanupTestResources(ctx context.Context, k8sClient client.Client, namespace string) {
	// Delete all NetworkIntents
	niList := &nephoranv1.NetworkIntentList{}
	if err := k8sClient.List(ctx, niList, client.InNamespace(namespace)); err == nil {
		for _, ni := range niList.Items {
			_ = k8sClient.Delete(ctx, &ni)
		}
	}

	// Delete all E2NodeSets
	e2nsList := &nephoranv1.E2NodeSetList{}
	if err := k8sClient.List(ctx, e2nsList, client.InNamespace(namespace)); err == nil {
		for _, e2ns := range e2nsList.Items {
			_ = k8sClient.Delete(ctx, &e2ns)
		}
	}

	// Delete all ConfigMaps
	cmList := &corev1.ConfigMapList{}
	if err := k8sClient.List(ctx, cmList, client.InNamespace(namespace)); err == nil {
		for _, cm := range cmList.Items {
			_ = k8sClient.Delete(ctx, &cm)
		}
	}
}

// WaitForResourceDeletion waits for a resource to be deleted
func WaitForResourceDeletion(ctx context.Context, k8sClient client.Client, obj client.Object) {
	Eventually(func() bool {
		err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj)
		return err != nil
	}, TestTimeout, TestInterval).Should(BeTrue())
}

// CreateTestContext creates a context with timeout for testing
func CreateTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), TestTimeout)
}
