//go:build integration

package controllers

import (
	"encoding/json"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/testutils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Type aliases for mock clients
type MockLLMClient = testutils.MockLLMClient
type MockGitClient = testutils.MockGitClient

// int32Ptr helper function is already defined in resource_planner.go

var _ = Describe("Error Handling and Recovery Tests", func() {
	const (
		timeout  = time.Second * 60
		interval = time.Millisecond * 250
	)

	var (
		namespaceName           string
		networkIntentReconciler *NetworkIntentReconciler
		e2nodeSetReconciler     *E2NodeSetReconciler
	)

	BeforeEach(func() {
		By("Creating isolated namespace for error recovery tests")
		namespaceName = CreateIsolatedNamespace("error-recovery")

		By("Setting up reconcilers for error testing")
		// Create a mock config for the NetworkIntentReconciler
		config := &Config{
			MaxRetries:      3,
			RetryDelay:      time.Second * 1,
			GitRepoURL:      "https://github.com/test/deployments.git",
			GitBranch:       "main",
			GitDeployPath:   "networkintents",
		}
		networkIntentReconciler = &NetworkIntentReconciler{
			Client: k8sClient,
			Scheme: testEnv.Scheme,
			config: config,
		}

		e2nodeSetReconciler = &E2NodeSetReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
	})

	AfterEach(func() {
		By("Cleaning up error recovery test namespace")
		CleanupIsolatedNamespace(namespaceName)
	})

	Context("NetworkIntent Error Scenarios", func() {
		It("Should handle persistent LLM failures gracefully", func() {
			By("Creating NetworkIntent with persistently failing LLM")
			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("persistent-llm-failure"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
						"test-scenario": "persistent-failure",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy network function that will consistently fail LLM processing",
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Setting up persistently failing LLM client")
			mockDeps := &testutils.MockDependencies{
				LLMClient: &MockLLMClient{
					Error: fmt.Errorf("persistent LLM service unavailable"),
				},
			}
			networkIntentReconciler.deps = mockDeps

			namespacedName := types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}

			By("Exhausting all retry attempts")
			for i := 0; i <= networkIntentReconciler.config.MaxRetries; i++ {
				result, err := networkIntentReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: namespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				if i < networkIntentReconciler.config.MaxRetries {
					Expect(result.RequeueAfter).To(Equal(networkIntentReconciler.config.RetryDelay))
				} else {
					Expect(result.Requeue).To(BeFalse())
					Expect(result.RequeueAfter).To(Equal(time.Duration(0)))
				}
			}

			By("Verifying NetworkIntent is marked as failed after max retries")
			finalIntent := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, namespacedName, finalIntent)).To(Succeed())

			processedCondition := testGetCondition(finalIntent.Status.Conditions, "Processed")
			Expect(processedCondition).NotTo(BeNil())
			Expect(processedCondition.Status).To(Equal(metav1.ConditionFalse))
			Expect(processedCondition.Reason).To(Equal("LLMProcessingFailedMaxRetries"))
			Expect(finalIntent.Status.Phase).To(Equal("Failed"))
		})

		It("Should recover from temporary Git failures", func() {
			By("Creating NetworkIntent with processed parameters")
			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("git-failure-recovery"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
						"test-scenario": "git-recovery",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy network function for Git recovery testing",
					ProcessedParameters: &nephoranv1.ProcessedParameters{
						NetworkFunction: "test-nf",
						ScaleParameters: &nephoranv1.ScaleParameters{
							MinReplicas: int32Ptr(2),
						},
					},
				},
				Status: nephoranv1.NetworkIntentStatus{
					Phase: "Processed",
					Conditions: []metav1.Condition{
						{
							Type:               "Processed",
							Status:             metav1.ConditionTrue,
							Reason:             "LLMProcessingSucceeded",
							Message:            "Intent processed successfully",
							LastTransitionTime: metav1.Now(),
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Setting up temporarily failing Git client")
			mockGitClient := &MockGitClient{}
			mockGitClient.SetCommitPushError(fmt.Errorf("temporary git authentication failure"))
			mockDeps := &testutils.MockDependencies{
				GitClient: mockGitClient,
			}
			networkIntentReconciler.deps = mockDeps

			namespacedName := types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}

			By("First reconciliation should fail Git operation")
			result, err := networkIntentReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(networkIntentReconciler.config.RetryDelay))

			By("Second reconciliation should also fail")
			result, err = networkIntentReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(networkIntentReconciler.config.RetryDelay))

			By("Third reconciliation should succeed after Git recovery")
			// Clear the commit push error to allow success
			mockGitClient.SetCommitPushError(nil)

			result, err = networkIntentReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying NetworkIntent deployment eventually succeeds")
			Eventually(func() bool {
				updated := &nephoranv1.NetworkIntent{}
				if err := k8sClient.Get(ctx, namespacedName, updated); err != nil {
					return false
				}
				return isConditionTrue(updated.Status.Conditions, "Deployed")
			}, timeout, interval).Should(BeTrue())

			By("Verifying final state")
			finalIntent := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, namespacedName, finalIntent)).To(Succeed())
			// GitCommitHash field doesn't exist in NetworkIntentStatus, skip this check
			Expect(finalIntent.Status.Phase).To(Equal("Completed"))
		})

		It("Should handle malformed LLM responses gracefully", func() {
			By("Creating NetworkIntent for malformed response testing")
			networkIntent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("malformed-response"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
						"test-scenario": "malformed-response",
					},
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy function to test malformed LLM responses",
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Setting up LLM client with malformed JSON responses")
			malformedResponses := []string{
				`{"incomplete": json`,               // Invalid JSON
				`null`,                              // Null response
				`"not an object"`,                   // Non-object response
				`{}`,                                // Empty object
				`{"missing_required_fields": true}`, // Missing required fields
				`{"action": "deploy", "component": null, "replicas": -1}`, // Invalid field values
			}

			namespacedName := types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}

			By("Testing each malformed response type")
			for i, malformedResponse := range malformedResponses {
				By(fmt.Sprintf("Testing malformed response %d: %s", i+1, malformedResponse))

				mockLLMClient := &MockLLMClient{}
				mockLLMClient.SetResponse("", malformedResponse)
				mockDeps := &testutils.MockDependencies{
					LLMClient: mockLLMClient,
				}
				networkIntentReconciler.deps = mockDeps

				result, err := networkIntentReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: namespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(networkIntentReconciler.config.RetryDelay))

				// Verify error condition is set
				updatedIntent := &nephoranv1.NetworkIntent{}
				Expect(k8sClient.Get(ctx, namespacedName, updatedIntent)).To(Succeed())

				processedCondition := testGetCondition(updatedIntent.Status.Conditions, "Processed")
				Expect(processedCondition).NotTo(BeNil())
				Expect(processedCondition.Status).To(Equal(metav1.ConditionFalse))
				Expect(processedCondition.Reason).To(Equal("LLMResponseParsingFailed"))
			}
		})

		It("Should handle concurrent NetworkIntent processing safely", func() {
			By("Creating multiple NetworkIntents simultaneously")
			networkIntents := []*nephoranv1.NetworkIntent{}

			for i := 0; i < 5; i++ {
				ni := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      GetUniqueName(fmt.Sprintf("concurrent-%d", i)),
						Namespace: namespaceName,
						Labels: map[string]string{
							"test-resource": "true",
							"test-scenario": "concurrent",
							"batch-id":      "batch-1",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: fmt.Sprintf("Deploy concurrent network function %d", i),
					},
				}
				Expect(k8sClient.Create(ctx, ni)).To(Succeed())
				networkIntents = append(networkIntents, ni)
			}

			By("Setting up LLM client for concurrent processing")
			successResponse := map[string]interface{}{
				"action":    "deploy",
				"component": "concurrent-nf",
				"replicas":  1,
			}
			successResponseBytes, _ := json.Marshal(successResponse)
			mockLLMClient := &MockLLMClient{}
			mockLLMClient.SetResponse("", string(successResponseBytes))
			mockGitClient := &MockGitClient{}
			mockDeps := &testutils.MockDependencies{
				LLMClient: mockLLMClient,
				GitClient: mockGitClient,
			}
			networkIntentReconciler.deps = mockDeps

			By("Processing all NetworkIntents concurrently")
			done := make(chan bool, len(networkIntents))
			errors := make(chan error, len(networkIntents))

			for _, ni := range networkIntents {
				go func(intent *nephoranv1.NetworkIntent) {
					defer GinkgoRecover()
					namespacedName := types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					}

					result, err := networkIntentReconciler.Reconcile(ctx, reconcile.Request{
						NamespacedName: namespacedName,
					})
					if err != nil {
						errors <- err
					} else {
						Expect(result.Requeue).To(BeFalse())
					}
					done <- true
				}(ni)
			}

			By("Waiting for all concurrent reconciliations to complete")
			for i := 0; i < len(networkIntents); i++ {
				select {
				case <-done:
					// Success
				case err := <-errors:
					Fail(fmt.Sprintf("Concurrent reconciliation failed: %v", err))
				case <-time.After(30 * time.Second):
					Fail("Concurrent reconciliation timed out")
				}
			}

			By("Verifying all NetworkIntents processed successfully")
			for _, ni := range networkIntents {
				Eventually(func() bool {
					updated := &nephoranv1.NetworkIntent{}
					namespacedName := types.NamespacedName{
						Name:      ni.Name,
						Namespace: ni.Namespace,
					}
					if err := k8sClient.Get(ctx, namespacedName, updated); err != nil {
						return false
					}
					return isConditionTrue(updated.Status.Conditions, "Processed") &&
						isConditionTrue(updated.Status.Conditions, "Deployed")
				}, timeout, interval).Should(BeTrue())
			}
		})
	})

	Context("E2NodeSet Error Scenarios", func() {
		It("Should handle ConfigMap creation conflicts and recovery", func() {
			By("Creating E2NodeSet for conflict testing")
			e2nodeSet := &nephoranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("conflict-recovery"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
						"test-scenario": "conflict-recovery",
					},
				},
				Spec: nephoranv1.E2NodeSetSpec{
					Replicas: 3,
				},
			}

			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			By("Pre-creating conflicting ConfigMaps")
			conflictingCMs := []*corev1.ConfigMap{}
			for i := 0; i < 2; i++ {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("%s-node-%d", e2nodeSet.Name, i),
						Namespace: namespaceName,
						Labels: map[string]string{
							"conflicting":  "true",
							"test-created": "true",
						},
					},
					Data: map[string]string{
						"conflict": "true",
						"index":    fmt.Sprintf("%d", i),
					},
				}
				Expect(k8sClient.Create(ctx, cm)).To(Succeed())
				conflictingCMs = append(conflictingCMs, cm)
			}

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("Initial reconciliation should encounter conflicts")
			result, err := e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).To(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			By("Verifying E2NodeSet status reflects partial creation")
			partialE2NodeSet := &nephoranv1.E2NodeSet{}
			Expect(k8sClient.Get(ctx, namespacedName, partialE2NodeSet)).To(Succeed())
			// Should have created only 1 ConfigMap (node-2) since 0 and 1 conflicted
			Expect(partialE2NodeSet.Status.ReadyReplicas).To(BeNumerically("<=", 1))

			By("Resolving conflicts by deleting conflicting ConfigMaps")
			for _, cm := range conflictingCMs {
				Expect(k8sClient.Delete(ctx, cm)).To(Succeed())
			}

			By("Retry reconciliation should succeed after conflict resolution")
			result, err = e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying E2NodeSet recovers and creates all ConfigMaps")
			testutils.WaitForConfigMapCount(ctx, k8sClient, namespaceName, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 3)

			WaitForE2NodeSetReady(namespacedName, 3)
		})

		It("Should handle partial ConfigMap deletions during scale-down", func() {
			By("Creating E2NodeSet with initial replicas")
			e2nodeSet := &nephoranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("partial-deletion"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
						"test-scenario": "partial-deletion",
					},
				},
				Spec: nephoranv1.E2NodeSetSpec{
					Replicas: 5,
				},
			}

			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("Initial reconciliation to create all ConfigMaps")
			result, err := e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for all ConfigMaps to be created")
			testutils.WaitForConfigMapCount(ctx, k8sClient, namespaceName, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 5)

			By("Adding finalizers to some ConfigMaps to prevent deletion")
			protectedConfigMaps := []string{
				fmt.Sprintf("%s-node-3", e2nodeSet.Name),
				fmt.Sprintf("%s-node-4", e2nodeSet.Name),
			}

			for _, cmName := range protectedConfigMaps {
				cm := &corev1.ConfigMap{}
				cmNamespacedName := types.NamespacedName{
					Name:      cmName,
					Namespace: namespaceName,
				}
				Expect(k8sClient.Get(ctx, cmNamespacedName, cm)).To(Succeed())

				cm.Finalizers = append(cm.Finalizers, "test.nephoran.com/prevent-deletion")
				Expect(k8sClient.Update(ctx, cm)).To(Succeed())
			}

			By("Scaling down to 2 replicas")
			Eventually(func() error {
				var currentE2NodeSet nephoranv1.E2NodeSet
				if err := k8sClient.Get(ctx, namespacedName, &currentE2NodeSet); err != nil {
					return err
				}
				currentE2NodeSet.Spec.Replicas = 2
				return k8sClient.Update(ctx, &currentE2NodeSet)
			}, timeout, interval).Should(Succeed())

			By("Reconciling after scale down")
			result, err = e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})

			By("Verifying reconciliation handles deletion failures")
			Expect(err).To(HaveOccurred()) // Should fail due to finalizers
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			By("Removing finalizers to allow cleanup")
			for _, cmName := range protectedConfigMaps {
				Eventually(func() error {
					cm := &corev1.ConfigMap{}
					cmNamespacedName := types.NamespacedName{
						Name:      cmName,
						Namespace: namespaceName,
					}
					if err := k8sClient.Get(ctx, cmNamespacedName, cm); err != nil {
						return err
					}
					cm.Finalizers = []string{}
					return k8sClient.Update(ctx, cm)
				}, timeout, interval).Should(Succeed())
			}

			By("Retry reconciliation should succeed after finalizer removal")
			result, err = e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("Verifying scale down completed successfully")
			testutils.WaitForConfigMapCount(ctx, k8sClient, namespaceName, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 2)

			WaitForE2NodeSetReady(namespacedName, 2)
		})

		It("Should handle rapid scaling operations", func() {
			By("Creating E2NodeSet for rapid scaling test")
			e2nodeSet := &nephoranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("rapid-scaling"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
						"test-scenario": "rapid-scaling",
					},
				},
				Spec: nephoranv1.E2NodeSetSpec{
					Replicas: 1,
				},
			}

			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("Initial state establishment")
			_, err := e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			WaitForE2NodeSetReady(namespacedName, 1)

			By("Performing rapid scaling operations")
			scaleSequence := []int32{5, 2, 8, 3, 1, 6}

			for i, targetReplicas := range scaleSequence {
				By(fmt.Sprintf("Rapid scale operation %d: scaling to %d replicas", i+1, targetReplicas))

				// Update replica count
				Eventually(func() error {
					var currentE2NodeSet nephoranv1.E2NodeSet
					if err := k8sClient.Get(ctx, namespacedName, &currentE2NodeSet); err != nil {
						return err
					}
					currentE2NodeSet.Spec.Replicas = targetReplicas
					return k8sClient.Update(ctx, &currentE2NodeSet)
				}, timeout, interval).Should(Succeed())

				// Reconcile
				_, err := e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: namespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				// Verify scaling completed
				WaitForE2NodeSetReady(namespacedName, targetReplicas)
				testutils.WaitForConfigMapCount(ctx, k8sClient, namespaceName, map[string]string{
					"app":       "e2node",
					"e2nodeset": e2nodeSet.Name,
				}, int(targetReplicas))
			}

			By("Verifying final state consistency")
			finalE2NodeSet := &nephoranv1.E2NodeSet{}
			Expect(k8sClient.Get(ctx, namespacedName, finalE2NodeSet)).To(Succeed())

			finalReplicas := scaleSequence[len(scaleSequence)-1]
			Expect(finalE2NodeSet.Spec.Replicas).To(Equal(finalReplicas))
			Expect(finalE2NodeSet.Status.ReadyReplicas).To(Equal(finalReplicas))
		})

		It("Should handle E2NodeSet resource corruption and recovery", func() {
			By("Creating E2NodeSet for corruption testing")
			e2nodeSet := &nephoranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      GetUniqueName("corruption-recovery"),
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource": "true",
						"test-scenario": "corruption-recovery",
					},
				},
				Spec: nephoranv1.E2NodeSetSpec{
					Replicas: 3,
				},
			}

			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			namespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			By("Initial reconciliation")
			_, err := e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for normal operation")
			WaitForE2NodeSetReady(namespacedName, 3)

			By("Simulating ConfigMap corruption by modifying labels")
			configMapList := &corev1.ConfigMapList{}
			listOptions := []client.ListOption{
				client.InNamespace(namespaceName),
				client.MatchingLabels(map[string]string{
					"app":       "e2node",
					"e2nodeset": e2nodeSet.Name,
				}),
			}
			Expect(k8sClient.List(ctx, configMapList, listOptions...)).To(Succeed())
			Expect(len(configMapList.Items)).To(Equal(3))

			// Corrupt one ConfigMap by removing essential labels
			corruptedCM := &configMapList.Items[0]
			delete(corruptedCM.Labels, "e2nodeset")
			delete(corruptedCM.Labels, "app")
			corruptedCM.Labels["corrupted"] = "true"
			Expect(k8sClient.Update(ctx, corruptedCM)).To(Succeed())

			By("Reconciling after corruption - should detect and recover")
			_, err = e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: namespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying controller recovers from corruption")
			// The controller should recreate the missing/corrupted ConfigMap
			Eventually(func() int {
				correctConfigMaps := &corev1.ConfigMapList{}
				Expect(k8sClient.List(ctx, correctConfigMaps, listOptions...)).To(Succeed())
				return len(correctConfigMaps.Items)
			}, timeout, interval).Should(Equal(3))

			WaitForE2NodeSetReady(namespacedName, 3)

			By("Verifying all ConfigMaps have correct labels and data")
			finalConfigMaps := &corev1.ConfigMapList{}
			Expect(k8sClient.List(ctx, finalConfigMaps, listOptions...)).To(Succeed())

			for _, cm := range finalConfigMaps.Items {
				Expect(cm.Labels["app"]).To(Equal("e2node"))
				Expect(cm.Labels["e2nodeset"]).To(Equal(e2nodeSet.Name))
				Expect(cm.Labels["nephoran.com/component"]).To(Equal("simulated-gnb"))
				Expect(cm.Data["nodeType"]).To(Equal("simulated-gnb"))
				Expect(cm.Data["status"]).To(Equal("active"))
			}
		})
	})

	Context("Cross-Controller Error Scenarios", func() {
		It("Should handle cascading failures across NetworkIntent and E2NodeSet", func() {
			By("Creating NetworkIntent that will drive E2NodeSet creation")
			networkIntent := CreateTestNetworkIntent(
				GetUniqueName("cascading-failure"),
				namespaceName,
				"Deploy E2NodeSet that will experience cascading failures",
			)

			// Set up successful LLM processing
			mockResponse := map[string]interface{}{
				"action":        "deploy",
				"component":     "e2nodeset",
				"resource_name": "cascading-test-e2nodeset",
				"replicas":      3,
			}
			mockResponseBytes, _ := json.Marshal(mockResponse)
			mockLLMClient := &MockLLMClient{}
			mockLLMClient.SetResponse("", string(mockResponseBytes))
			mockGitClient := &MockGitClient{}
			mockDeps := &testutils.MockDependencies{
				LLMClient: mockLLMClient,
				GitClient: mockGitClient,
			}
			networkIntentReconciler.deps = mockDeps

			Expect(k8sClient.Create(ctx, networkIntent)).To(Succeed())

			By("Processing NetworkIntent successfully")
			niNamespacedName := types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}

			result, err := networkIntentReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: niNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying NetworkIntent processing completed")
			Eventually(func() bool {
				updated := &nephoranv1.NetworkIntent{}
				if err := k8sClient.Get(ctx, niNamespacedName, updated); err != nil {
					return false
				}
				return isConditionTrue(updated.Status.Conditions, "Processed") &&
					isConditionTrue(updated.Status.Conditions, "Deployed")
			}, timeout, interval).Should(BeTrue())

			By("Creating E2NodeSet as would be done by GitOps system")
			e2nodeSet := &nephoranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cascading-test-e2nodeset",
					Namespace: namespaceName,
					Labels: map[string]string{
						"test-resource":       "true",
						"source-intent":       networkIntent.Name,
						"nephoran.com/intent": networkIntent.Name,
					},
					Annotations: map[string]string{
						"nephoran.com/intent-source": networkIntent.Name,
					},
				},
				Spec: nephoranv1.E2NodeSetSpec{
					Replicas: 3,
				},
			}

			Expect(k8sClient.Create(ctx, e2nodeSet)).To(Succeed())

			By("Simulating E2NodeSet failures during processing")
			e2nsNamespacedName := types.NamespacedName{
				Name:      e2nodeSet.Name,
				Namespace: e2nodeSet.Namespace,
			}

			// Create some ConfigMaps manually with invalid configurations
			invalidCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-node-1", e2nodeSet.Name),
					Namespace: namespaceName,
					Labels: map[string]string{
						"invalid": "true",
					},
				},
				Data: map[string]string{
					"invalid": "configuration",
				},
			}
			Expect(k8sClient.Create(ctx, invalidCM)).To(Succeed())

			By("E2NodeSet reconciliation should handle the invalid ConfigMap")
			result, err = e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: e2nsNamespacedName,
			})
			// Should fail due to conflicting ConfigMap
			Expect(err).To(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(30 * time.Second))

			By("Resolving the conflict and retrying")
			Expect(k8sClient.Delete(ctx, invalidCM)).To(Succeed())

			result, err = e2nodeSetReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: e2nsNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Verifying both controllers eventually reach consistent state")
			WaitForE2NodeSetReady(e2nsNamespacedName, 3)
			testutils.WaitForConfigMapCount(ctx, k8sClient, namespaceName, map[string]string{
				"app":       "e2node",
				"e2nodeset": e2nodeSet.Name,
			}, 3)

			By("Verifying NetworkIntent and E2NodeSet relationship is maintained")
			finalNetworkIntent := &nephoranv1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, niNamespacedName, finalNetworkIntent)).To(Succeed())
			Expect(finalNetworkIntent.Status.Phase).To(Equal("Completed"))

			finalE2NodeSet := &nephoranv1.E2NodeSet{}
			Expect(k8sClient.Get(ctx, e2nsNamespacedName, finalE2NodeSet)).To(Succeed())
			Expect(finalE2NodeSet.Annotations["nephoran.com/intent-source"]).To(Equal(networkIntent.Name))
			Expect(finalE2NodeSet.Status.ReadyReplicas).To(Equal(int32(3)))
		})
	})
})

// Helper functions specific to error handling tests

func testIsConditionTrue(conditions []metav1.Condition, conditionType string) bool {
	return getConditionStatus(conditions, conditionType) == metav1.ConditionTrue
}

func getConditionStatus(conditions []metav1.Condition, conditionType string) metav1.ConditionStatus {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return metav1.ConditionUnknown
}

func testGetCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

func testGetRetryCount(ni *nephoranv1.NetworkIntent, operation string) int {
	for _, condition := range ni.Status.Conditions {
		if condition.Reason == "LLMProcessingRetrying" && condition.Message != "" {
			return 1
		}
	}
	return 0
}
