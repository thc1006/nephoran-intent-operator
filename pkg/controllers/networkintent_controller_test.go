/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephoranv1alpha1 "nephoran-intent-operator/pkg/apis/nephoran/v1alpha1"
)

var _ = Describe("NetworkIntent Controller", func() {
	const (
		IntentName      = "test-intent"
		IntentNamespace = "default"
		DeploymentName  = "test-deployment"
	)

	Context("When reconciling a NetworkIntent", func() {
		It("Should create a Deployment for a new NetworkIntent", func() {
			By("Creating a new NetworkIntent")
			ctx := context.Background()
			intent := &nephoranv1alpha1.NetworkIntent{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "nephoran.com/v1alpha1",
					Kind:       "NetworkIntent",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      IntentName,
					Namespace: IntentNamespace,
				},
				Spec: nephoranv1alpha1.NetworkIntentSpec{
					Intent: "Deploy a network function",
					Parameters: runtime.RawExtension{
						Raw: mustMarshal(NetworkFunctionDeploymentIntent{
							Type:      "NetworkFunctionDeployment",
							Name:      DeploymentName,
							Namespace: IntentNamespace,
							Spec: NetworkFunctionDeploymentSpec{
								Replicas: 2,
								Image:    "test-image:latest",
								Resources: ResourceRequirements{
									Requests: ResourceList{CPU: "100m", Memory: "128Mi"},
									Limits:   ResourceList{CPU: "200m", Memory: "256Mi"},
								},
							},
						}),
					},
				},
			}
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())

			deploymentLookupKey := types.NamespacedName{Name: DeploymentName, Namespace: IntentNamespace}
			createdDeployment := &appsv1.Deployment{}

			// We'll need to retry getting this deployment, as it may not be created immediately.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, deploymentLookupKey, createdDeployment)
				return err == nil
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			Expect(*createdDeployment.Spec.Replicas).Should(Equal(int32(2)))
			Expect(createdDeployment.Spec.Template.Spec.Containers[0].Image).Should(Equal("test-image:latest"))
		})

		It("Should update the Deployment when the NetworkIntent is updated", func() {
			By("Updating the NetworkIntent")
			ctx := context.Background()
			intent := &nephoranv1alpha1.NetworkIntent{}
			intentLookupKey := types.NamespacedName{Name: IntentName, Namespace: IntentNamespace}
			Expect(k8sClient.Get(ctx, intentLookupKey, intent)).Should(Succeed())

			intent.Spec.Parameters = runtime.RawExtension{
				Raw: mustMarshal(NetworkFunctionDeploymentIntent{
					Type:      "NetworkFunctionDeployment",
					Name:      DeploymentName,
					Namespace: IntentNamespace,
					Spec: NetworkFunctionDeploymentSpec{
						Replicas: 3, // Changed from 2 to 3
						Image:    "test-image:v2", // Changed image
						Resources: ResourceRequirements{
							Requests: ResourceList{CPU: "100m", Memory: "128Mi"},
							Limits:   ResourceList{CPU: "200m", Memory: "256Mi"},
						},
					},
				}),
			}
			Expect(k8sClient.Update(ctx, intent)).Should(Succeed())

			deploymentLookupKey := types.NamespacedName{Name: DeploymentName, Namespace: IntentNamespace}
			updatedDeployment := &appsv1.Deployment{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, deploymentLookupKey, updatedDeployment)
				if err != nil {
					return false
				}
				return *updatedDeployment.Spec.Replicas == int32(3) && updatedDeployment.Spec.Template.Spec.Containers[0].Image == "test-image:v2"
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())
		})

		It("Should update the NetworkIntent status", func() {
			By("Checking the NetworkIntent status")
			ctx := context.Background()
			intent := &nephoranv1alpha1.NetworkIntent{}
			intentLookupKey := types.NamespacedName{Name: IntentName, Namespace: IntentNamespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, intentLookupKey, intent)
				if err != nil {
					return false
				}
				for _, cond := range intent.Status.Conditions {
					if cond.Type == typeAvailableNetworkIntent && cond.Status == metav1.ConditionTrue {
						return true
					}
				}
				return false
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())
		})
	})
})

func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
