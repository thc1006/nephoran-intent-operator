package porch_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Kubebuilder Environment Verification", func() {
	Context("Basic API Server Connectivity", func() {
		It("Should be able to connect to the test API server", func() {
			By("Checking that cfg is not nil")
			Expect(cfg).NotTo(BeNil())

			By("Checking that k8sClient is not nil")
			Expect(k8sClient).NotTo(BeNil())
		})

		It("Should be able to create a simple namespace with valid RFC 1123 name", func() {
			By("Creating a namespace with lowercase name")
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-kubebuilder-verification",
				},
			}

			err := k8sClient.Create(ctx, ns)
			Expect(err).NotTo(HaveOccurred())

			By("Cleaning up the namespace")
			err = k8sClient.Delete(ctx, ns)
			Expect(err).NotTo(HaveOccurred())
		})
	})
})