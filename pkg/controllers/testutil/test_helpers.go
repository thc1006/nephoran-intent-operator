package testutil

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateIsolatedNamespace creates a unique namespace for test isolation.

func CreateIsolatedNamespace(baseName string) string {

	return fmt.Sprintf("%s-%d", baseName, rand.Intn(10000))

}

// CreateTestNamespace creates a namespace with the given name using the client.

func CreateTestNamespace(ctx context.Context, k8sClient client.Client, namespaceName string) error {

	namespace := &corev1.Namespace{

		ObjectMeta: metav1.ObjectMeta{

			Name: namespaceName,
		},
	}

	return k8sClient.Create(ctx, namespace)

}

// DeleteTestNamespace deletes a namespace using the client.

func DeleteTestNamespace(ctx context.Context, k8sClient client.Client, namespaceName string) error {

	namespace := &corev1.Namespace{

		ObjectMeta: metav1.ObjectMeta{

			Name: namespaceName,
		},
	}

	return k8sClient.Delete(ctx, namespace)

}

// CreateTestE2NodeSet creates a basic E2NodeSet for testing.

func CreateTestE2NodeSet(name, namespace string, replicas int32) *nephoranv1.E2NodeSet {

	return &nephoranv1.E2NodeSet{

		ObjectMeta: metav1.ObjectMeta{

			Name: name,

			Namespace: namespace,
		},

		Spec: nephoranv1.E2NodeSetSpec{

			Replicas: replicas,

			RicEndpoint: "http://localhost:38080",
		},
	}

}

// CreateTestNetworkIntent creates a basic NetworkIntent for testing.

func CreateTestNetworkIntent(name, namespace string) *nephoranv1.NetworkIntent {

	return &nephoranv1.NetworkIntent{

		ObjectMeta: metav1.ObjectMeta{

			Name: name,

			Namespace: namespace,
		},

		Spec: nephoranv1.NetworkIntentSpec{

			Intent: "Configure QoS with 100Mbps bandwidth and 10ms latency",

			IntentType: nephoranv1.IntentTypeOptimization,
		},
	}

}

// WaitForCondition waits for a condition to be met on a resource.

func WaitForCondition(ctx context.Context, k8sClient client.Client, obj client.Object, conditionCheck func() bool, timeout time.Duration) error {

	ctx, cancel := context.WithTimeout(ctx, timeout)

	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			return fmt.Errorf("timeout waiting for condition")

		case <-ticker.C:

			if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {

				continue

			}

			if conditionCheck() {

				return nil

			}

		}

	}

}

// EnsureResourceExists checks if a resource exists and creates it if it doesn't.

func EnsureResourceExists(ctx context.Context, k8sClient client.Client, obj client.Object) error {

	if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {

		// Resource doesn't exist, create it.

		return k8sClient.Create(ctx, obj)

	}

	return nil

}

// EnsureResourceDeleted ensures a resource is deleted.

func EnsureResourceDeleted(ctx context.Context, k8sClient client.Client, obj client.Object) error {

	return client.IgnoreNotFound(k8sClient.Delete(ctx, obj))

}
