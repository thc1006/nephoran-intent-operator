/*
Package envtest provides comprehensive controller testing for NetworkIntent
following 2025 Kubernetes operator testing best practices.
*/

package envtest

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/v1alpha1"
)

var _ = Describe("NetworkIntent Controller", Ordered, func() {
	Context("When creating a NetworkIntent", func() {
		var (
			networkIntent     *intentv1alpha1.NetworkIntent
			networkIntentName string
			testNamespace     string
		)

		BeforeEach(func() {
			testNamespace = "default"
			networkIntentName = "test-network-intent"
			
			networkIntent = &intentv1alpha1.NetworkIntent{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "nephoran.com/v1alpha1",
					Kind:       "NetworkIntent",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      networkIntentName,
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					ScalingPriority: "high",
					TargetClusters:  []string{"cluster-1", "cluster-2"},
				},
			}
		})

		It("Should create NetworkIntent successfully", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())
			
			createdIntent := &intentv1alpha1.NetworkIntent{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      networkIntentName,
					Namespace: testNamespace,
				}, createdIntent)
			}, time.Second*10, time.Millisecond*250).Should(Succeed())
			
			Expect(createdIntent.Spec.ScalingPriority).To(Equal("high"))
			Expect(createdIntent.Spec.TargetClusters).To(ContainElements("cluster-1", "cluster-2"))
		})
	})
})
