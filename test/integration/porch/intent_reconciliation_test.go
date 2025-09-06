package porch_test

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	networkintentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/v1alpha1"
	porchv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/v1alpha1"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
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
					Deployment: networkintentv1alpha1.DeploymentSpec{
						ClusterSelector: map[string]string{
							"environment": "test",
						},
						NetworkFunctions: []networkintentv1alpha1.NetworkFunction{
							{
								Name: "test-nf",
								Type: "CNF",
							},
						},
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

			Expect(createdPackage.Spec.WorkspaceName).NotTo(BeEmpty())
			Expect(createdPackage.Spec.RepositoryName).NotTo(BeEmpty())
			Expect(createdPackage.Status.Phase).To(Equal(porchv1alpha1.PackagePhaseCreated))
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