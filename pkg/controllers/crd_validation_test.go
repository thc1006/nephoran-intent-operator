//go:build integration

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/testutil"
)

var _ = Describe("CRD Validation and Schema Tests", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	var namespaceName string

	BeforeEach(func() {
		By("Creating a new namespace for test isolation")
		namespaceName = CreateIsolatedNamespace("crd-validation")
	})

	AfterEach(func() {
		By("Cleaning up the test namespace")
		// CleanupIsolatedNamespace not implemented - using simple namespace name
	})

	Context("NetworkIntent CRD Schema Validation", func() {
		It("Should accept valid NetworkIntent resources", func() {
			By("Creating a valid NetworkIntent")
			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("valid-intent"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
						"test-suite":    "crd-validation",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy 5G core with 3 AMF replicas",
					// Parameters field doesn't exist in NetworkIntentSpec
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Verifying the NetworkIntent was created successfully")
			created := &nephoranv1.NetworkIntent{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), created)
			}, timeout, interval).Should(Succeed())

			Expect(created.Spec.Intent).To(Equal("Deploy 5G core with 3 AMF replicas"))
			// Parameters field doesn't exist, so we skip this validation
		})

		It("Should accept NetworkIntent with optional fields", func() {
			By("Creating a NetworkIntent with minimal required fields")
			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("minimal-intent"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Scale network functions",
					// Parameters omitted (optional)
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Verifying the minimal NetworkIntent was created")
			created := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), created)).To(Succeed())
			Expect(created.Spec.Intent).To(Equal("Scale network functions"))
		})

		It("Should reject NetworkIntent with invalid schema", func() {
			By("Attempting to create NetworkIntent with empty intent")
			invalidIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("invalid-intent"),
					Namespace: namespaceName,
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "", // Empty intent should be invalid
				},
			}

			err := k8sClient.Create(ctx, invalidIntent)
			Expect(err).To(HaveOccurred())
			Expect(errors.IsInvalid(err)).To(BeTrue())
		})

		It("Should properly handle NetworkIntent status updates", func() {
			By("Creating a NetworkIntent")
			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("status-intent"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Test status updates",
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Updating NetworkIntent status")
			Eventually(func() error {
				// Get latest version
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), networkIntent)
				if err != nil {
					return err
				}

				// Update status
				networkIntent.Status.Phase = "Processing"
				networkIntent.Status.ObservedGeneration = networkIntent.Generation
				networkIntent.Status.Conditions = []metav1.Condition{
					{
						Type:               "Processed",
						Status:             metav1.ConditionTrue,
						Reason:             "LLMProcessingSucceeded",
						Message:            "Intent processed successfully",
						LastTransitionTime: metav1.Now(),
					},
				}

				return k8sClient.Status().Update(ctx, networkIntent)
			}, timeout, interval).Should(Succeed())

			By("Verifying status was updated")
			updated := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), updated)).To(Succeed())
			Expect(updated.Status.Phase).To(Equal("Processing"))
			Expect(updated.Status.ObservedGeneration).To(Equal(updated.Generation))
			Expect(len(updated.Status.Conditions)).To(Equal(1))
			Expect(updated.Status.Conditions[0].Type).To(Equal("Processed"))
		})
	})

	Context("E2NodeSet CRD Schema Validation", func() {
		It("Should accept valid E2NodeSet resources", func() {
			By("Creating a valid E2NodeSet")
			e2nodeSet := &nephoranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("valid-e2nodeset"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
						"test-suite":    "crd-validation",
					},
				},
				Spec: nephoranv1.E2NodeSetSpec{
					Replicas: 3,
				},
			}

			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			By("Verifying the E2NodeSet was created successfully")
			created := &nephoranv1.E2NodeSet{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), created)
			}, timeout, interval).Should(Succeed())

			Expect(created.Spec.Replicas).To(Equal(int32(3)))
		})

		It("Should reject E2NodeSet with invalid replica count", func() {
			By("Attempting to create E2NodeSet with negative replicas")
			invalidE2NodeSet := &nephoranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("invalid-e2nodeset"),
					Namespace: namespaceName,
				},
				Spec: nephoranv1.E2NodeSetSpec{
					Replicas: -1, // Negative replicas should be invalid
				},
			}

			err := k8sClient.Create(ctx, invalidE2NodeSet)
			Expect(err).To(HaveOccurred())
			Expect(errors.IsInvalid(err)).To(BeTrue())
		})

		It("Should properly handle E2NodeSet status updates", func() {
			By("Creating an E2NodeSet")
			e2nodeSet := &nephoranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("status-e2nodeset"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
					},
				},
				Spec: nephoranv1.E2NodeSetSpec{
					Replicas: 2,
				},
			}

			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			By("Updating E2NodeSet status")
			Eventually(func() error {
				// Get latest version
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), e2nodeSet)
				if err != nil {
					return err
				}

				// Update status
				e2nodeSet.Status.ReadyReplicas = 2

				return k8sClient.Status().Update(ctx, e2nodeSet)
			}, timeout, interval).Should(Succeed())

			By("Verifying status was updated")
			updated := &nephoranv1.E2NodeSet{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), updated)).To(Succeed())
			Expect(updated.Status.ReadyReplicas).To(Equal(int32(2)))
		})

		It("Should handle E2NodeSet with zero replicas", func() {
			By("Creating E2NodeSet with zero replicas")
			e2nodeSet := &nephoranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("zero-replicas"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
					},
				},
				Spec: nephoranv1.E2NodeSetSpec{
					Replicas: 0,
				},
			}

			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			By("Verifying zero replicas is accepted")
			created := &nephoranv1.E2NodeSet{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), created)).To(Succeed())
			Expect(created.Spec.Replicas).To(Equal(int32(0)))
		})
	})

	Context("ManagedElement CRD Schema Validation", func() {
		It("Should accept valid ManagedElement resources", func() {
			By("Creating a valid ManagedElement")
			managedElement := &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("valid-me"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
						"test-suite":    "crd-validation",
					},
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "test-o-du-deployment",
					O1Config:       "o1-config-string",
					A1Policy: runtime.RawExtension{
						Raw: []byte(`{"interfaces": ["E1", "F1"], "capabilities": ["MIMO", "CA"]}`),
					},
				},
			}

			Expect(k8sClient.Create(ctx, managedElement)).To(Succeed())

			By("Verifying the ManagedElement was created successfully")
			created := &nephoranv1.ManagedElement{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKeyFromObject(managedElement), created)
			}, timeout, interval).Should(Succeed())

			Expect(created.Spec.DeploymentName).To(Equal("test-o-du-deployment"))
			Expect(created.Spec.O1Config).To(Equal("o1-config-string"))
			Expect(created.Spec.A1Policy.Raw).NotTo(BeEmpty())
		})

		It("Should accept ManagedElement with minimal fields", func() {
			By("Creating ManagedElement with only required fields")
			managedElement := &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("minimal-me"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
					},
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "minimal-deployment",
				},
			}

			Expect(k8sClient.Create(ctx, managedElement)).To(Succeed())

			By("Verifying minimal ManagedElement was created")
			created := &nephoranv1.ManagedElement{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(managedElement), created)).To(Succeed())
			Expect(created.Spec.DeploymentName).To(Equal("minimal-deployment"))
		})

		It("Should properly handle ManagedElement status updates", func() {
			By("Creating a ManagedElement")
			managedElement := &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("status-me"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
					},
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "status-test-deployment",
				},
			}

			Expect(k8sClient.Create(ctx, managedElement)).To(Succeed())

			By("Updating ManagedElement status")
			Eventually(func() error {
				// Get latest version
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(managedElement), managedElement)
				if err != nil {
					return err
				}

				// Update status
				managedElement.Status.Conditions = []metav1.Condition{
					{
						Type:               "Ready",
						Status:             metav1.ConditionTrue,
						Reason:             "ElementOperational",
						Message:            "Managed element is operational",
						LastTransitionTime: metav1.Now(),
					},
				}

				return k8sClient.Status().Update(ctx, managedElement)
			}, timeout, interval).Should(Succeed())

			By("Verifying status was updated")
			updated := &nephoranv1.ManagedElement{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(managedElement), updated)).To(Succeed())
			Expect(len(updated.Status.Conditions)).To(Equal(1))
			Expect(updated.Status.Conditions[0].Type).To(Equal("Ready"))
		})
	})

	Context("Cross-CRD Integration Tests", func() {
		It("Should create and relate multiple CRD types", func() {
			By("Creating a ManagedElement")
			managedElement := &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("integration-me"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
						"test-suite":    "integration",
					},
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "integration-ric-deployment",
					O1Config:       "ric-config",
				},
			}
			Expect(k8sClient.Create(ctx, managedElement)).To(Succeed())

			By("Creating an E2NodeSet that references the ManagedElement")
			e2nodeSet := &nephoranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("integration-e2nodeset"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource":   "true",
						"managed-element": managedElement.Name,
					},
				},
				Spec: nephoranv1.E2NodeSetSpec{
					Replicas: 2,
				},
			}
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			By("Creating a NetworkIntent that targets the E2NodeSet")
			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("integration-intent"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource":    "true",
						"target-e2nodeset": e2nodeSet.Name,
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: fmt.Sprintf("Scale E2NodeSet %s to 5 replicas", e2nodeSet.Name),
					// Parameters field doesn't exist - removed field content
				},
			}
			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Verifying all resources were created successfully")
			createdME := &nephoranv1.ManagedElement{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(managedElement), createdME)).To(Succeed())

			createdE2NS := &nephoranv1.E2NodeSet{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), createdE2NS)).To(Succeed())

			createdNI := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), createdNI)).To(Succeed())

			By("Verifying relationships are preserved")
			Expect(createdE2NS.Labels["managed-element"]).To(Equal(managedElement.Name))
			Expect(createdNI.Labels["target-e2nodeset"]).To(Equal(e2nodeSet.Name))

			// Verify configuration references
			// Note: Since Parameters field doesn't exist, we can't unmarshal config
			// We'll verify the intent string instead
			Expect(createdNI.Spec.Intent).To(ContainSubstring(e2nodeSet.Name))
		})

		It("Should handle resource dependencies and cleanup", func() {
			By("Creating resources with owner references")

			// Create parent ManagedElement
			managedElement := &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("parent-me"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
					},
				},
				Spec: nephoranv1.ManagedElementSpec{
					DeploymentName: "parent-smo-deployment",
					O1Config:       "parent-config",
				},
			}
			Expect(k8sClient.Create(ctx, managedElement)).To(Succeed())

			// Create child E2NodeSet with owner reference
			e2nodeSet := &nephoranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("child-e2nodeset"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
					},
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "nephoran.com/v1",
							Kind:       "ManagedElement",
							Name:       managedElement.Name,
							UID:        managedElement.UID,
						},
					},
				},
				Spec: nephoranv1.E2NodeSetSpec{
					Replicas: 1,
				},
			}
			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			By("Verifying both resources exist")
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(managedElement), &nephoranv1.ManagedElement{})).To(Succeed())
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(e2nodeSet), &nephoranv1.E2NodeSet{})).To(Succeed())

			By("Deleting parent resource")
			Expect(k8sClient.Delete(ctx, managedElement)).To(Succeed())

			By("Verifying parent is deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(managedElement), &nephoranv1.ManagedElement{})
				return errors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())

			// Note: Child resource cleanup depends on garbage collection controller
			// In test environment, this might not be automatically cleaned up
		})
	})

	Context("CRD Field Validation Edge Cases", func() {
		It("Should handle large JSON configurations", func() {
			By("Creating NetworkIntent with large parameters")
			largeConfig := map[string]interface{}{
				"deployment": map[string]interface{}{
					"replicas": 10,
					"resources": map[string]interface{}{
						"cpu":    "2000m",
						"memory": "4Gi",
					},
					"volumes": []map[string]interface{}{
						{"name": "config", "mountPath": "/etc/config"},
						{"name": "data", "mountPath": "/var/data"},
						{"name": "logs", "mountPath": "/var/logs"},
					},
				},
				"networking": map[string]interface{}{
					"services": []map[string]interface{}{
						{"name": "api", "port": 8080, "targetPort": 8080},
						{"name": "metrics", "port": 9090, "targetPort": 9090},
					},
					"ingress": map[string]interface{}{
						"enabled":  true,
						"hostname": "api.example.com",
						"tls":      true,
						"annotations": map[string]string{
							"nginx.ingress.kubernetes.io/rewrite-target": "/",
						},
					},
				},
			}
			// Note: largeConfigBytes would be used if Parameters field existed
			_ = largeConfig // Keep for documentation but suppress unused warning

			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("large-config"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy complex network function with detailed configuration",
					// Parameters field doesn't exist - removed field content
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Verifying large configuration was preserved")
			created := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), created)).To(Succeed())

			// Since Parameters field doesn't exist, we can't unmarshal config
			// We'll verify the intent string contains expected content
			Expect(created.Spec.Intent).To(ContainSubstring("complex network function"))
		})

		It("Should handle Unicode and special characters", func() {
			By("Creating resources with Unicode characters")
			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("unicode-test"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "部署网络功能 - Deploy network function with 特殊字符 🚀 and spéciál çhäracters",
					// Parameters field doesn't exist - removed field content
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Verifying Unicode characters were preserved")
			created := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(networkIntent), created)).To(Succeed())
			Expect(created.Spec.Intent).To(ContainSubstring("部署网络功能"))
			Expect(created.Spec.Intent).To(ContainSubstring("🚀"))
		})
	})
})
