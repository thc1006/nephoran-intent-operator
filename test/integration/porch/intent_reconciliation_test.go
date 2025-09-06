package porch_test

import (
	"context"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	networkintentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789"  // RFC 1123 compliant: lowercase letters and digits only
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
					Source:     "integration-test",
					IntentType: "scaling",
					Target:     "test-nf",
					Namespace:  ns,
					Replicas:   3,
					ScalingParameters: networkintentv1alpha1.ScalingConfig{
						Replicas: 3,
					},
				},
			}

			// Create intent
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())

			// Wait for package to be created
			Eventually(func() bool {
				packageList := &porchv1alpha1.PackageList{}
				err := k8sClient.List(ctx, packageList, client.InNamespace(ns))
				return err == nil && len(packageList.Items) > 0
			}, 2*time.Minute, 10*time.Second).Should(BeTrue())

			// Verify package details
			var createdPackage porchv1alpha1.Package
			packageList := &porchv1alpha1.PackageList{}
			Expect(k8sClient.List(ctx, packageList, client.InNamespace(ns))).Should(Succeed())
			Expect(len(packageList.Items)).To(BeNumerically(">", 0))
			createdPackage = packageList.Items[0]

			Expect(createdPackage.Name).NotTo(BeEmpty())
			Expect(createdPackage.Namespace).To(Equal(ns))
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
	err := client.Delete(ctx, ns)
	// Use errors.IsNotFound to ignore "not found" errors during cleanup
	// namespace may not have been created successfully due to validation errors
	if err != nil && !errors.IsNotFound(err) {
		Expect(err).Should(Succeed())
	}
}
