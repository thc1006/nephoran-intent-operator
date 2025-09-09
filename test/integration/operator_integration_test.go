//go:build integration
// +build integration

/*
Package integration provides end-to-end integration testing for the Nephoran
Kubernetes operator in real cluster environments following 2025 best practices.
*/

package integration

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/v1alpha1"
)

var _ = Describe("Operator Integration Tests", func() {
	Context("When deploying NetworkIntent in real cluster", func() {
		It("Should handle NetworkIntent lifecycle", func() {
			ctx := context.Background()
			
			intent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "integration-test-intent",
					Namespace: "default",
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					ScalingPriority: "medium",
					TargetClusters:  []string{"integration-cluster"},
				},
			}
			
			By("creating the NetworkIntent")
			Expect(k8sClient.Create(ctx, intent)).To(Succeed())
			
			By("verifying the NetworkIntent exists")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "integration-test-intent",
					Namespace: "default",
				}, intent)
			}, time.Second*30, time.Second).Should(Succeed())
			
			By("cleaning up the NetworkIntent")
			Expect(k8sClient.Delete(ctx, intent)).To(Succeed())
		})
	})
})
