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
		return string(ni.Status.Phase)
	}, TestTimeout, TestInterval).Should(Equal(expectedPhase))
}

// WaitForNetworkIntentMessage waits for a NetworkIntent to have a specific message
func WaitForNetworkIntentMessage(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName, expectedMessage string) {
	Eventually(func() string {
		ni := &nephoranv1.NetworkIntent{}
		err := k8sClient.Get(ctx, namespacedName, ni)
		if err != nil {
			return ""
		}
		return ni.Status.LastMessage
	}, TestTimeout, TestInterval).Should(ContainSubstring(expectedMessage))
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

// AssertNetworkIntentHasMessage asserts that a NetworkIntent has a specific message
func AssertNetworkIntentHasMessage(ni *nephoranv1.NetworkIntent, expectedMessage string) {
	Expect(ni.Status.LastMessage).To(ContainSubstring(expectedMessage), "NetworkIntent should have message containing %s", expectedMessage)
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

// WaitForMultipleE2NodeSetsReady waits for multiple E2NodeSets to be ready
func WaitForMultipleE2NodeSetsReady(ctx context.Context, k8sClient client.Client, e2nodeSets []*nephoranv1.E2NodeSet, expectedReplicas []int32) {
	for i, e2ns := range e2nodeSets {
		namespacedName := types.NamespacedName{
			Name:      e2ns.Name,
			Namespace: e2ns.Namespace,
		}
		WaitForE2NodeSetReady(ctx, k8sClient, namespacedName, expectedReplicas[i])
	}
}

// WaitForMultipleNetworkIntentsProcessed waits for multiple NetworkIntents to be processed
func WaitForMultipleNetworkIntentsProcessed(ctx context.Context, k8sClient client.Client, networkIntents []*nephoranv1.NetworkIntent) {
	for _, ni := range networkIntents {
		namespacedName := types.NamespacedName{
			Name:      ni.Name,
			Namespace: ni.Namespace,
		}
		WaitForNetworkIntentPhase(ctx, k8sClient, namespacedName, "Processed")
	}
}

// GetResourceCounts returns counts of different resource types in a namespace
func GetResourceCounts(ctx context.Context, k8sClient client.Client, namespace string) (map[string]int, error) {
	counts := make(map[string]int)

	// Count NetworkIntents
	niList := &nephoranv1.NetworkIntentList{}
	if err := k8sClient.List(ctx, niList, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	counts["NetworkIntent"] = len(niList.Items)

	// Count E2NodeSets
	e2nsList := &nephoranv1.E2NodeSetList{}
	if err := k8sClient.List(ctx, e2nsList, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	counts["E2NodeSet"] = len(e2nsList.Items)

	// Count ManagedElements
	meList := &nephoranv1.ManagedElementList{}
	if err := k8sClient.List(ctx, meList, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	counts["ManagedElement"] = len(meList.Items)

	// Count ConfigMaps
	cmList := &corev1.ConfigMapList{}
	if err := k8sClient.List(ctx, cmList, client.InNamespace(namespace)); err != nil {
		return nil, err
	}
	counts["ConfigMap"] = len(cmList.Items)

	return counts, nil
}

// VerifyResourceConsistency verifies that resources are in expected states
func VerifyResourceConsistency(ctx context.Context, k8sClient client.Client, namespace string) error {
	// Verify E2NodeSets and their ConfigMaps are consistent
	e2nsList := &nephoranv1.E2NodeSetList{}
	if err := k8sClient.List(ctx, e2nsList, client.InNamespace(namespace)); err != nil {
		return err
	}

	for _, e2ns := range e2nsList.Items {
		// Check ConfigMaps for this E2NodeSet
		configMapList := &corev1.ConfigMapList{}
		listOptions := []client.ListOption{
			client.InNamespace(namespace),
			client.MatchingLabels(map[string]string{
				"app":       "e2node",
				"e2nodeset": e2ns.Name,
			}),
		}
		if err := k8sClient.List(ctx, configMapList, listOptions...); err != nil {
			return err
		}

		expectedConfigMaps := int(e2ns.Spec.Replicas)
		actualConfigMaps := len(configMapList.Items)
		readyReplicas := int(e2ns.Status.ReadyReplicas)

		if actualConfigMaps != expectedConfigMaps {
			return fmt.Errorf("E2NodeSet %s: expected %d ConfigMaps, found %d",
				e2ns.Name, expectedConfigMaps, actualConfigMaps)
		}

		if readyReplicas != expectedConfigMaps {
			return fmt.Errorf("E2NodeSet %s: expected %d ready replicas, found %d",
				e2ns.Name, expectedConfigMaps, readyReplicas)
		}
	}

	return nil
}

// WaitForResourceConsistency waits for all resources to reach a consistent state
func WaitForResourceConsistency(ctx context.Context, k8sClient client.Client, namespace string) {
	Eventually(func() error {
		return VerifyResourceConsistency(ctx, k8sClient, namespace)
	}, TestTimeout, TestInterval).Should(Succeed())
}

// CreateTestSecret creates a test secret for authentication or configuration
func CreateTestSecret(ctx context.Context, k8sClient client.Client, name, namespace string, data map[string][]byte) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"test-resource": "true",
			},
		},
		Data: data,
	}

	Expect(k8sClient.Create(ctx, secret)).To(Succeed())
	return secret
}

// WaitForPhaseWithMessage waits for a phase and message combination
func WaitForPhaseWithMessage(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName, expectedPhase, expectedMessage string) {
	Eventually(func() bool {
		ni := &nephoranv1.NetworkIntent{}
		if err := k8sClient.Get(ctx, namespacedName, ni); err != nil {
			return false
		}
		return string(ni.Status.Phase) == expectedPhase && ni.Status.LastMessage == expectedMessage
	}, TestTimeout, TestInterval).Should(BeTrue())
}

// GetE2NodeSetConfigMaps returns all ConfigMaps belonging to an E2NodeSet
func GetE2NodeSetConfigMaps(ctx context.Context, k8sClient client.Client, e2nodeSet *nephoranv1.E2NodeSet) (*corev1.ConfigMapList, error) {
	configMapList := &corev1.ConfigMapList{}
	listOptions := []client.ListOption{
		client.InNamespace(e2nodeSet.Namespace),
		client.MatchingLabels(map[string]string{
			"app":       "e2node",
			"e2nodeset": e2nodeSet.Name,
		}),
	}

	err := k8sClient.List(ctx, configMapList, listOptions...)
	return configMapList, err
}

// VerifyOwnerReferences checks that child resources have correct owner references
func VerifyOwnerReferences(ctx context.Context, k8sClient client.Client, owner client.Object, childObjects []client.Object) error {
	for _, child := range childObjects {
		if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(child), child); err != nil {
			return fmt.Errorf("failed to get child object %s: %w", child.GetName(), err)
		}

		found := false
		for _, ownerRef := range child.GetOwnerReferences() {
			if ownerRef.UID == owner.GetUID() && ownerRef.Kind == owner.GetObjectKind().GroupVersionKind().Kind {
				found = true
				if ownerRef.Controller == nil || !*ownerRef.Controller {
					return fmt.Errorf("child %s does not have controller owner reference", child.GetName())
				}
				break
			}
		}

		if !found {
			return fmt.Errorf("child %s does not have owner reference to %s", child.GetName(), owner.GetName())
		}
	}

	return nil
}

// WaitForOwnerReferences waits for owner references to be properly set
func WaitForOwnerReferences(ctx context.Context, k8sClient client.Client, owner client.Object, childObjects []client.Object) {
	Eventually(func() error {
		return VerifyOwnerReferences(ctx, k8sClient, owner, childObjects)
	}, TestTimeout, TestInterval).Should(Succeed())
}

// E2NodeSet-specific test utilities

// WaitForE2NodeSetStatusUpdate waits for E2NodeSet status to be updated
func WaitForE2NodeSetStatusUpdate(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName, checkFunc func(*nephoranv1.E2NodeSet) bool) {
	Eventually(func() bool {
		e2ns := &nephoranv1.E2NodeSet{}
		err := k8sClient.Get(ctx, namespacedName, e2ns)
		if err != nil {
			return false
		}
		return checkFunc(e2ns)
	}, TestTimeout, TestInterval).Should(BeTrue())
}

// WaitForE2NodeSetDeletion waits for an E2NodeSet to be completely deleted
func WaitForE2NodeSetDeletion(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName) {
	Eventually(func() bool {
		e2ns := &nephoranv1.E2NodeSet{}
		err := k8sClient.Get(ctx, namespacedName, e2ns)
		return err != nil
	}, TestTimeout, TestInterval).Should(BeTrue())
}

// CreateE2NodeSetWithFinalizer creates an E2NodeSet with the specified finalizer
func CreateE2NodeSetWithFinalizer(ctx context.Context, k8sClient client.Client, name, namespace string, replicas int32, finalizer string) *nephoranv1.E2NodeSet {
	e2ns := &nephoranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Finalizers: []string{finalizer},
			Labels: map[string]string{
				"test-resource": "true",
			},
		},
		Spec: nephoranv1.E2NodeSetSpec{
			Replicas: replicas,
		},
	}

	Expect(k8sClient.Create(ctx, e2ns)).To(Succeed())
	return e2ns
}

// VerifyE2NodeSetFinalizer verifies that an E2NodeSet has the expected finalizer
func VerifyE2NodeSetFinalizer(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName, expectedFinalizer string, shouldExist bool) {
	e2ns := &nephoranv1.E2NodeSet{}
	Expect(k8sClient.Get(ctx, namespacedName, e2ns)).To(Succeed())

	found := false
	for _, finalizer := range e2ns.Finalizers {
		if finalizer == expectedFinalizer {
			found = true
			break
		}
	}

	if shouldExist {
		Expect(found).To(BeTrue(), "E2NodeSet should have finalizer %s", expectedFinalizer)
	} else {
		Expect(found).To(BeFalse(), "E2NodeSet should not have finalizer %s", expectedFinalizer)
	}
}

// WaitForE2NodeSetFinalizer waits for an E2NodeSet to have or not have a specific finalizer
func WaitForE2NodeSetFinalizer(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName, finalizer string, shouldExist bool) {
	Eventually(func() bool {
		e2ns := &nephoranv1.E2NodeSet{}
		err := k8sClient.Get(ctx, namespacedName, e2ns)
		if err != nil {
			return false
		}

		found := false
		for _, f := range e2ns.Finalizers {
			if f == finalizer {
				found = true
				break
			}
		}

		return found == shouldExist
	}, TestTimeout, TestInterval).Should(BeTrue())
}

// GetE2NodeSetReadyReplicasCount returns the number of ready replicas for an E2NodeSet
func GetE2NodeSetReadyReplicasCount(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName) int32 {
	e2ns := &nephoranv1.E2NodeSet{}
	Expect(k8sClient.Get(ctx, namespacedName, e2ns)).To(Succeed())
	return e2ns.Status.ReadyReplicas
}

// VerifyNoConfigMapOperations verifies that no ConfigMap operations were performed
func VerifyNoConfigMapOperations(ctx context.Context, k8sClient client.Client, namespace string, initialCount int) {
	// Wait a bit to ensure any operations would have been performed
	time.Sleep(100 * time.Millisecond)

	// Count current ConfigMaps
	configMapList := &corev1.ConfigMapList{}
	err := k8sClient.List(ctx, configMapList, client.InNamespace(namespace))
	Expect(err).NotTo(HaveOccurred())

	currentCount := len(configMapList.Items)
	Expect(currentCount).To(Equal(initialCount), "No ConfigMap operations should have been performed")
}

// CreateMinimalE2NodeSet creates a minimal E2NodeSet for testing
func CreateMinimalE2NodeSet(name, namespace string, replicas int32) *nephoranv1.E2NodeSet {
	return &nephoranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"test-resource": "true",
			},
		},
		Spec: nephoranv1.E2NodeSetSpec{
			Replicas: replicas,
		},
	}
}

// CreateE2NodeSetWithCustomEndpoint creates an E2NodeSet with a custom RIC endpoint
func CreateE2NodeSetWithCustomEndpoint(name, namespace string, replicas int32, ricEndpoint string) *nephoranv1.E2NodeSet {
	return &nephoranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"test-resource": "true",
			},
		},
		Spec: nephoranv1.E2NodeSetSpec{
			Replicas:    replicas,
			RicEndpoint: ricEndpoint,
		},
	}
}

// UpdateE2NodeSetReplicas updates the replica count of an E2NodeSet
func UpdateE2NodeSetReplicas(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName, newReplicas int32) {
	e2ns := &nephoranv1.E2NodeSet{}
	Expect(k8sClient.Get(ctx, namespacedName, e2ns)).To(Succeed())
	e2ns.Spec.Replicas = newReplicas
	Expect(k8sClient.Update(ctx, e2ns)).To(Succeed())
}

// WaitForE2NodeSetScaling waits for an E2NodeSet to complete scaling operation
func WaitForE2NodeSetScaling(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName, targetReplicas int32) {
	Eventually(func() bool {
		e2ns := &nephoranv1.E2NodeSet{}
		err := k8sClient.Get(ctx, namespacedName, e2ns)
		if err != nil {
			return false
		}
		// Check that both spec and status match target
		return e2ns.Spec.Replicas == targetReplicas && e2ns.Status.ReadyReplicas == targetReplicas
	}, TestTimeout*2, TestInterval).Should(BeTrue()) // Give more time for scaling operations
}

// GetE2NodeSetByLabels retrieves E2NodeSets by label selector
func GetE2NodeSetByLabels(ctx context.Context, k8sClient client.Client, namespace string, labels map[string]string) ([]nephoranv1.E2NodeSet, error) {
	e2nsList := &nephoranv1.E2NodeSetList{}
	listOptions := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels(labels),
	}

	err := k8sClient.List(ctx, e2nsList, listOptions...)
	if err != nil {
		return nil, err
	}

	return e2nsList.Items, nil
}

// CreateE2NodeSetWithAnnotations creates an E2NodeSet with custom annotations
func CreateE2NodeSetWithAnnotations(name, namespace string, replicas int32, annotations map[string]string) *nephoranv1.E2NodeSet {
	e2ns := CreateMinimalE2NodeSet(name, namespace, replicas)
	if e2ns.Annotations == nil {
		e2ns.Annotations = make(map[string]string)
	}
	for k, v := range annotations {
		e2ns.Annotations[k] = v
	}
	return e2ns
}

// VerifyE2NodeSetAnnotation verifies that an E2NodeSet has the expected annotation
func VerifyE2NodeSetAnnotation(ctx context.Context, k8sClient client.Client, namespacedName types.NamespacedName, key, expectedValue string) {
	e2ns := &nephoranv1.E2NodeSet{}
	Expect(k8sClient.Get(ctx, namespacedName, e2ns)).To(Succeed())

	actualValue, exists := e2ns.Annotations[key]
	Expect(exists).To(BeTrue(), "E2NodeSet should have annotation %s", key)
	Expect(actualValue).To(Equal(expectedValue), "Annotation %s should have value %s", key, expectedValue)
}
