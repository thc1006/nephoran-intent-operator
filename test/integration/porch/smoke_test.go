package porch_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	networkintentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
)

var _ = Describe("Test Environment Smoke Test", func() {
	Context("Basic Connectivity", func() {
		It("Should be able to create and delete a namespace", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			nsName := "smoke-test-" + randomString(6)

			// Create namespace
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
				},
			}
			
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			// Verify namespace exists
			var retrievedNs corev1.Namespace
			Expect(k8sClient.Get(ctx, client.ObjectKey{Name: nsName}, &retrievedNs)).Should(Succeed())

			// Clean up
			err := k8sClient.Delete(ctx, ns)
			if err != nil && !errors.IsNotFound(err) {
				Expect(err).Should(Succeed())
			}
		})

		It("Should be able to create a NetworkIntent CRD", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			nsName := "smoke-crd-test-" + randomString(6)

			// Create namespace first
			createNamespace(ctx, k8sClient, nsName)
			defer func() {
				deleteNamespace(ctx, k8sClient, nsName)
			}()

			// Create NetworkIntent
			intent := &networkintentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "smoke-test-intent",
					Namespace: nsName,
				},
				Spec: networkintentv1alpha1.NetworkIntentSpec{
					Source:     "smoke-test",
					IntentType: "scaling",
					Target:     "smoke-nf",
					Namespace:  nsName,
					Replicas:   1,
					ScalingParameters: networkintentv1alpha1.ScalingConfig{
						Replicas: 1,
					},
				},
			}

			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())

			// Verify NetworkIntent exists
			var retrievedIntent networkintentv1alpha1.NetworkIntent
			Expect(k8sClient.Get(ctx, client.ObjectKey{
				Name:      "smoke-test-intent",
				Namespace: nsName,
			}, &retrievedIntent)).Should(Succeed())

			Expect(retrievedIntent.Spec.Target).To(Equal("smoke-nf"))

			// Clean up intent
			err := k8sClient.Delete(ctx, intent)
			if err != nil && !errors.IsNotFound(err) {
				Expect(err).Should(Succeed())
			}
		})
	})
})