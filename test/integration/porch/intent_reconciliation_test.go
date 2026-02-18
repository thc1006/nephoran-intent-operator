package porch_test

import (
	"context"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	networkintentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyz"
)

var _ = Describe("Porch Intent Reconciliation", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
		ns     string
	)

	BeforeEach(func() {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Minute)
		ns = "test-intent-" + randomString(8)

		// Create test namespace
		createNamespace(ctx, k8sClient, ns)
	})

	AfterEach(func() {
		defer cancel()
		deleteNamespace(ctx, k8sClient, ns)
	})

	Context("Package Creation Workflow", func() {
		It("Should create a package from an intent", func() {
			intent := &networkintentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-network-intent",
					Namespace: ns,
				},
				Spec: networkintentv1alpha1.NetworkIntentSpec{
					Target:     "test-nf",
					IntentType: "scaling",
					Namespace:  ns,
				},
			}

			// Create intent
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())

			// TODO: Wait for porch package to be created when porch integration is implemented
			// For now, verify that the intent was created successfully
			Eventually(func() bool {
				var retrievedIntent networkintentv1alpha1.NetworkIntent
				err := k8sClient.Get(ctx, client.ObjectKey{Name: "test-network-intent", Namespace: ns}, &retrievedIntent)
				return err == nil
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			// TODO: Verify package details when porch types are available
			// For now, verify intent properties
			var retrievedIntent networkintentv1alpha1.NetworkIntent
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: "test-network-intent", Namespace: ns}, &retrievedIntent)).Should(Succeed())
			Expect(retrievedIntent.Spec.Target).To(Equal("test-nf"))
			Expect(retrievedIntent.Spec.IntentType).To(Equal("scaling"))
		})
	})

	Context("Package Update Workflow", func() {
		It("Should update package when intent changes", func() {
			// Similar to creation test, but modify intent and verify package update
			// Test scaling, network function modifications
		})
	})

	Context("Package Deletion Workflow", func() {
		It("Should delete package when intent is deleted", func() {
			// Create intent
			// Delete intent
			// Verify package is also deleted
		})
	})
})

// Utility functions
func randomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func createNamespace(ctx context.Context, client client.Client, namespace string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	Expect(client.Create(ctx, ns)).Should(Succeed())
}

func deleteNamespace(ctx context.Context, client client.Client, namespace string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	Expect(client.Delete(ctx, ns)).Should(Succeed())
}
