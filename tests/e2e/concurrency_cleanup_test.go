package e2e

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"
)

var _ = Describe("Concurrency and Resource Cleanup E2E Tests", func() {
	var (
		ctx       context.Context
		namespace string
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = "default"
	})

	Context("Concurrent Request Processing", func() {
		It("should handle multiple simultaneous NetworkIntent creations", func() {
			By("Creating multiple NetworkIntents concurrently")
			concurrentCount := 5
			var wg sync.WaitGroup
			var mu sync.Mutex
			results := make([]error, concurrentCount)
			createdIntents := make([]*nephoran.NetworkIntent, concurrentCount)

			// Create intents concurrently
			for i := 0; i < concurrentCount; i++ {
				wg.Add(1)
				go func(index int) {
					defer GinkgoRecover()
					defer wg.Done()

					intentName := fmt.Sprintf("concurrent-creation-%d", index)
					intent := &nephoran.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      intentName,
							Namespace: namespace,
							Labels: map[string]string{
								"nephoran.io/test-type": "concurrent-creation",
								"nephoran.io/batch-id":  "concurrent-batch-1",
							},
						},
						Spec: nephoran.NetworkIntentSpec{
							Intent:     fmt.Sprintf("Concurrent creation test %d: Scale UPF instances", index),
							IntentType: nephoran.IntentTypeScaling,
							Priority:   1,
							TargetComponents: []nephoran.NetworkTargetComponent{
								nephoran.NetworkTargetComponentUPF,
							},
						},
					}

					err := k8sClient.Create(ctx, intent)

					mu.Lock()
					results[index] = err
					if err == nil {
						createdIntents[index] = intent
					}
					mu.Unlock()
				}(i)
			}

			wg.Wait()

			By("Verifying all concurrent creations succeeded")
			successCount := 0
			for i, err := range results {
				if err == nil {
					successCount++
				} else {
					By(fmt.Sprintf("Intent %d creation failed: %v", i, err))
				}
			}
			Expect(successCount).Should(Equal(concurrentCount))

			By("Verifying all created intents are processed independently")
			for i, intent := range createdIntents {
				if intent == nil {
					continue
				}

				lookupKey := types.NamespacedName{Name: intent.Name, Namespace: namespace}
				retrievedIntent := &nephoran.NetworkIntent{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
					if err != nil {
						return false
					}
					return retrievedIntent.Status.Phase == "Processing"
				}, 30*time.Second, 1*time.Second).Should(BeTrue())

				By(fmt.Sprintf("Verifying intent %d maintains correct spec", i))
				expectedText := fmt.Sprintf("Concurrent creation test %d:", i)
				Expect(retrievedIntent.Spec.Intent).Should(ContainSubstring(expectedText))
			}

			By("Cleaning up concurrently created intents")
			var cleanupWg sync.WaitGroup
			for _, intent := range createdIntents {
				if intent == nil {
					continue
				}

				cleanupWg.Add(1)
				go func(intentToDelete *nephoran.NetworkIntent) {
					defer GinkgoRecover()
					defer cleanupWg.Done()
					Expect(k8sClient.Delete(ctx, intentToDelete)).Should(Succeed())
				}(intent)
			}
			cleanupWg.Wait()
		})

		It("should handle concurrent updates to the same NetworkIntent", func() {
			By("Creating a NetworkIntent for concurrent updates")
			intentName := "concurrent-update-intent"
			baseIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
					Labels: map[string]string{
						"nephoran.io/test-type": "concurrent-updates",
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Base intent for concurrent update testing",
					IntentType: nephoran.IntentTypeScaling,
					Priority:   1,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, baseIntent)).Should(Succeed())

			By("Performing concurrent updates")
			concurrentUpdates := 3
			var updateWg sync.WaitGroup
			var mu sync.Mutex
			updateResults := make([]error, concurrentUpdates)

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}

			for i := 0; i < concurrentUpdates; i++ {
				updateWg.Add(1)
				go func(updateIndex int) {
					defer GinkgoRecover()
					defer updateWg.Done()

					// Get fresh copy for update
					intentToUpdate := &nephoran.NetworkIntent{}
					err := k8sClient.Get(ctx, lookupKey, intentToUpdate)
					if err != nil {
						mu.Lock()
						updateResults[updateIndex] = err
						mu.Unlock()
						return
					}

					// Modify the intent
					intentToUpdate.Spec.Intent = fmt.Sprintf("Updated intent %d: %s",
						updateIndex, intentToUpdate.Spec.Intent)

					// Add annotation to track update
					if intentToUpdate.Annotations == nil {
						intentToUpdate.Annotations = make(map[string]string)
					}
					intentToUpdate.Annotations[fmt.Sprintf("update-%d", updateIndex)] = "applied"

					err = k8sClient.Update(ctx, intentToUpdate)
					mu.Lock()
					updateResults[updateIndex] = err
					mu.Unlock()
				}(i)
			}

			updateWg.Wait()

			By("Verifying update results")
			successfulUpdates := 0
			conflictErrors := 0

			for i, err := range updateResults {
				if err == nil {
					successfulUpdates++
				} else if errors.IsConflict(err) {
					conflictErrors++
					By(fmt.Sprintf("Update %d had expected conflict: %v", i, err))
				} else {
					Fail(fmt.Sprintf("Update %d had unexpected error: %v", i, err))
				}
			}

			// At least one update should succeed, others may conflict
			Expect(successfulUpdates).Should(BeNumerically(">=", 1))
			Expect(successfulUpdates + conflictErrors).Should(Equal(concurrentUpdates))

			By("Verifying final state is consistent")
			finalIntent := &nephoran.NetworkIntent{}
			Eventually(func() error {
				return k8sClient.Get(ctx, lookupKey, finalIntent)
			}, 10*time.Second, 1*time.Second).Should(Succeed())

			Expect(finalIntent.Spec.Intent).Should(ContainSubstring("Updated intent"))
			Expect(len(finalIntent.Annotations)).Should(BeNumerically(">=", 1))

			By("Cleaning up concurrent update test intent")
			Expect(k8sClient.Delete(ctx, finalIntent)).Should(Succeed())
		})

		It("should handle concurrent deletions gracefully", func() {
			By("Creating multiple intents for concurrent deletion testing")
			deletionIntents := 4
			createdIntents := make([]*nephoran.NetworkIntent, deletionIntents)

			for i := 0; i < deletionIntents; i++ {
				intentName := fmt.Sprintf("concurrent-deletion-%d", i)
				intent := &nephoran.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      intentName,
						Namespace: namespace,
						Labels: map[string]string{
							"nephoran.io/test-type": "concurrent-deletion",
						},
					},
					Spec: nephoran.NetworkIntentSpec{
						Intent:     fmt.Sprintf("Concurrent deletion test %d", i),
						IntentType: nephoran.IntentTypeScaling,
						Priority:   1,
						TargetComponents: []nephoran.NetworkTargetComponent{
							nephoran.NetworkTargetComponentUPF,
						},
					},
				}

				Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
				createdIntents[i] = intent
			}

			By("Waiting for all intents to be processed initially")
			for _, intent := range createdIntents {
				lookupKey := types.NamespacedName{Name: intent.Name, Namespace: namespace}
				retrievedIntent := &nephoran.NetworkIntent{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
					if err != nil {
						return false
					}
					return len(retrievedIntent.Status.Conditions) > 0
				}, 30*time.Second, 1*time.Second).Should(BeTrue())
			}

			By("Performing concurrent deletions")
			var deletionWg sync.WaitGroup
			var mu sync.Mutex
			deletionResults := make([]error, deletionIntents)

			for i, intent := range createdIntents {
				deletionWg.Add(1)
				go func(index int, intentToDelete *nephoran.NetworkIntent) {
					defer GinkgoRecover()
					defer deletionWg.Done()

					err := k8sClient.Delete(ctx, intentToDelete)
					mu.Lock()
					deletionResults[index] = err
					mu.Unlock()
				}(i, intent)
			}

			deletionWg.Wait()

			By("Verifying deletion results")
			successfulDeletions := 0
			for i, err := range deletionResults {
				if err == nil {
					successfulDeletions++
				} else {
					By(fmt.Sprintf("Deletion %d failed: %v", i, err))
				}
			}

			// All deletions should succeed
			Expect(successfulDeletions).Should(Equal(deletionIntents))

			By("Verifying all intents are actually deleted")
			for _, intent := range createdIntents {
				lookupKey := types.NamespacedName{Name: intent.Name, Namespace: namespace}
				deletedIntent := &nephoran.NetworkIntent{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, lookupKey, deletedIntent)
					return errors.IsNotFound(err)
				}, 15*time.Second, 1*time.Second).Should(BeTrue())
			}
		})
	})

	Context("Resource Cleanup and Lifecycle Management", func() {
		It("should clean up resources when NetworkIntents are deleted", func() {
			By("Creating NetworkIntent with potential dependent resources")
			intentName := "resource-cleanup-intent"
			intent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
					Labels: map[string]string{
						"nephoran.io/test-type": "resource-cleanup",
					},
					Finalizers: []string{
						"nephoran.io/cleanup-finalizer", // This might be added by controller
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Test resource cleanup when intent is deleted",
					IntentType: nephoran.IntentTypeDeployment,
					Priority:   1,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
						nephoran.NetworkTargetComponentSMF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Waiting for intent processing to create resources")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}
				return len(createdIntent.Status.Conditions) > 0
			}, 30*time.Second, 2*time.Second).Should(BeTrue())

			By("Recording initial state before deletion")
			_ = len(createdIntent.Status.Conditions)
			_ = createdIntent.Status.Phase

			By("Deleting the NetworkIntent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())

			By("Verifying cleanup process")
			if len(createdIntent.Finalizers) > 0 {
				By("Monitoring finalizer-based cleanup")
				Eventually(func() bool {
					err := k8sClient.Get(ctx, lookupKey, createdIntent)
					if errors.IsNotFound(err) {
						return true // Cleanup completed
					}
					if err != nil {
						return false
					}

					// Check if finalizers are being processed
					return len(createdIntent.Finalizers) < len(intent.Finalizers) ||
						createdIntent.DeletionTimestamp != nil
				}, 45*time.Second, 2*time.Second).Should(BeTrue())
			}

			By("Verifying complete resource deletion")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				return errors.IsNotFound(err)
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			By("Verifying no orphaned resources remain")
			// Check that no related resources are left behind
			relatedIntents := &nephoran.NetworkIntentList{}
			err := k8sClient.List(ctx, relatedIntents,
				client.InNamespace(namespace),
				client.MatchingLabels{"nephoran.io/test-type": "resource-cleanup"})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(len(relatedIntents.Items)).Should(Equal(0))
		})

		It("should handle cleanup during rapid create/delete cycles", func() {
			By("Performing rapid create/delete cycles")
			cycles := 5

			for cycle := 0; cycle < cycles; cycle++ {
				intentName := fmt.Sprintf("rapid-cleanup-cycle-%d", cycle)

				By(fmt.Sprintf("Cycle %d: Creating intent %s", cycle, intentName))
				intent := &nephoran.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      intentName,
						Namespace: namespace,
						Labels: map[string]string{
							"nephoran.io/test-type": "rapid-cleanup",
							"nephoran.io/cycle":     fmt.Sprintf("%d", cycle),
						},
					},
					Spec: nephoran.NetworkIntentSpec{
						Intent:     fmt.Sprintf("Rapid cleanup cycle %d test", cycle),
						IntentType: nephoran.IntentTypeScaling,
						Priority:   1,
						TargetComponents: []nephoran.NetworkTargetComponent{
							nephoran.NetworkTargetComponentUPF,
						},
					},
				}

				Expect(k8sClient.Create(ctx, intent)).Should(Succeed())

				// Very short processing time before deletion
				time.Sleep(200 * time.Millisecond)

				By(fmt.Sprintf("Cycle %d: Deleting intent %s", cycle, intentName))
				Expect(k8sClient.Delete(ctx, intent)).Should(Succeed())

				// Verify deletion completed before next cycle
				lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
				Eventually(func() bool {
					err := k8sClient.Get(ctx, lookupKey, intent)
					return errors.IsNotFound(err)
				}, 10*time.Second, 500*time.Millisecond).Should(BeTrue())
			}

			By("Verifying system stability after rapid cycles")
			// Create one final intent to verify system is still functional
			finalIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "post-rapid-cleanup-test",
					Namespace: namespace,
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Post rapid cleanup stability test",
					IntentType: nephoran.IntentTypeScaling,
					Priority:   1,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, finalIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: "post-rapid-cleanup-test", Namespace: namespace}
			retrievedIntent := &nephoran.NetworkIntent{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
				if err != nil {
					return false
				}
				return retrievedIntent.Status.Phase == "Processing"
			}, 20*time.Second, 1*time.Second).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, retrievedIntent)).Should(Succeed())
		})

		It("should prevent resource leaks during error scenarios", func() {
			By("Creating intent that may cause resource allocation errors")
			intentName := "resource-leak-test-intent"
			intent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
					Labels: map[string]string{
						"nephoran.io/test-type": "resource-leak-prevention",
					},
					Annotations: map[string]string{
						"nephoran.io/error-prone": "true",
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Test resource leak prevention during error scenarios",
					IntentType: nephoran.IntentTypeDeployment,
					Priority:   1,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentAMF,
						nephoran.NetworkTargetComponentSMF,
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Waiting for processing to begin (potentially with errors)")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}
				return len(createdIntent.Status.Conditions) > 0
			}, 30*time.Second, 2*time.Second).Should(BeTrue())

			By("Recording resource state before deletion")
			_ = createdIntent.Status.Phase
			_ = len(createdIntent.Status.Conditions)

			By("Deleting intent during potential error processing")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())

			By("Verifying clean deletion despite processing errors")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				return errors.IsNotFound(err)
			}, 30*time.Second, 1*time.Second).Should(BeTrue())

			By("Verifying no leaked resources after error scenario cleanup")
			// List all NetworkIntents to ensure our test intent is gone
			allIntents := &nephoran.NetworkIntentList{}
			err := k8sClient.List(ctx, allIntents, client.InNamespace(namespace))
			Expect(err).ShouldNot(HaveOccurred())

			for _, remainingIntent := range allIntents.Items {
				Expect(remainingIntent.Name).ShouldNot(Equal(intentName))
			}
		})
	})

	Context("Load Testing and Resource Management", func() {
		It("should handle high-volume concurrent operations", func() {
			By("Creating high volume of concurrent intents")
			highVolumeCount := 10
			var creationWg sync.WaitGroup
			var mu sync.Mutex
			createdIntents := make([]*nephoran.NetworkIntent, highVolumeCount)
			creationErrors := make([]error, highVolumeCount)

			// Phase 1: Concurrent creation
			for i := 0; i < highVolumeCount; i++ {
				creationWg.Add(1)
				go func(index int) {
					defer GinkgoRecover()
					defer creationWg.Done()

					intentName := fmt.Sprintf("high-volume-%d", index)
					intent := &nephoran.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      intentName,
							Namespace: namespace,
							Labels: map[string]string{
								"nephoran.io/test-type": "high-volume",
								"nephoran.io/batch":     "volume-test-1",
							},
						},
						Spec: nephoran.NetworkIntentSpec{
							Intent:     fmt.Sprintf("High volume test %d: Load testing intent processing", index),
							IntentType: nephoran.IntentTypeScaling,
							Priority:   1,
							TargetComponents: []nephoran.NetworkTargetComponent{
								nephoran.NetworkTargetComponentUPF,
							},
						},
					}

					err := k8sClient.Create(ctx, intent)

					mu.Lock()
					creationErrors[index] = err
					if err == nil {
						createdIntents[index] = intent
					}
					mu.Unlock()
				}(i)
			}

			creationWg.Wait()

			By("Verifying creation success rate")
			successfulCreations := 0
			for i, err := range creationErrors {
				if err == nil {
					successfulCreations++
				} else {
					By(fmt.Sprintf("Creation %d failed: %v", i, err))
				}
			}

			// Should achieve high success rate
			successRate := float64(successfulCreations) / float64(highVolumeCount)
			Expect(successRate).Should(BeNumerically(">=", 0.9)) // 90% success rate

			By("Monitoring processing under high load")
			processedCount := 0

			for _, intent := range createdIntents {
				if intent == nil {
					continue
				}

				lookupKey := types.NamespacedName{Name: intent.Name, Namespace: namespace}
				retrievedIntent := &nephoran.NetworkIntent{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
					if err != nil {
						return false
					}
					return len(retrievedIntent.Status.Conditions) > 0
				}, 60*time.Second, 1*time.Second).Should(BeTrue())

				processedCount++
			}

			By(fmt.Sprintf("Successfully processed %d/%d intents under high load", processedCount, successfulCreations))

			By("Phase 2: Concurrent cleanup of high-volume intents")
			var cleanupWg sync.WaitGroup
			cleanupErrors := make([]error, len(createdIntents))

			for i, intent := range createdIntents {
				if intent == nil {
					continue
				}

				cleanupWg.Add(1)
				go func(index int, intentToDelete *nephoran.NetworkIntent) {
					defer GinkgoRecover()
					defer cleanupWg.Done()

					err := k8sClient.Delete(ctx, intentToDelete)
					mu.Lock()
					cleanupErrors[index] = err
					mu.Unlock()
				}(i, intent)
			}

			cleanupWg.Wait()

			By("Verifying cleanup success rate")
			successfulCleanups := 0
			for i, err := range cleanupErrors {
				if err == nil {
					successfulCleanups++
				} else if err != nil {
					By(fmt.Sprintf("Cleanup %d failed: %v", i, err))
				}
			}

			cleanupRate := float64(successfulCleanups) / float64(len(createdIntents))
			Expect(cleanupRate).Should(BeNumerically(">=", 0.9)) // 90% cleanup success rate
		})

		It("should maintain performance under sustained concurrent load", func() {
			By("Running sustained concurrent operations")
			duration := 30 * time.Second
			maxConcurrent := 3
			var operationWg sync.WaitGroup
			var mu sync.Mutex
			operationCount := 0
			errorCount := 0

			startTime := time.Now()

			// Run operations for specified duration
			for time.Since(startTime) < duration {
				if operationCount >= maxConcurrent {
					time.Sleep(100 * time.Millisecond)
					continue
				}

				operationWg.Add(1)
				go func(opIndex int) {
					defer GinkgoRecover()
					defer operationWg.Done()
					defer func() {
						mu.Lock()
						operationCount--
						mu.Unlock()
					}()

					mu.Lock()
					operationCount++
					currentOp := operationCount
					mu.Unlock()

					intentName := fmt.Sprintf("sustained-load-%d-%d", time.Now().Unix(), opIndex)
					intent := &nephoran.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      intentName,
							Namespace: namespace,
							Labels: map[string]string{
								"nephoran.io/test-type": "sustained-load",
							},
						},
						Spec: nephoran.NetworkIntentSpec{
							Intent:     fmt.Sprintf("Sustained load test operation %d", currentOp),
							IntentType: nephoran.IntentTypeScaling,
							Priority:   1,
							TargetComponents: []nephoran.NetworkTargetComponent{
								nephoran.NetworkTargetComponentUPF,
							},
						},
					}

					// Create, verify, and cleanup
					err := k8sClient.Create(ctx, intent)
					if err != nil {
						mu.Lock()
						errorCount++
						mu.Unlock()
						return
					}

					// Short processing time
					time.Sleep(500 * time.Millisecond)

					// Cleanup
					err = k8sClient.Delete(ctx, intent)
					if err != nil {
						mu.Lock()
						errorCount++
						mu.Unlock()
					}
				}(operationCount)

				time.Sleep(200 * time.Millisecond) // Throttle operation creation
			}

			operationWg.Wait()

			By("Verifying sustained performance metrics")
			totalOperations := operationCount + errorCount
			if totalOperations > 0 {
				errorRate := float64(errorCount) / float64(totalOperations)
				By(fmt.Sprintf("Sustained load: %d operations, %d errors, %.2f%% error rate",
					totalOperations, errorCount, errorRate*100))

				// Should maintain reasonable error rate under sustained load
				Expect(errorRate).Should(BeNumerically("<=", 0.2)) // Max 20% error rate
			}

			By("Verifying system stability after sustained load")
			stabilityIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "post-sustained-load-stability",
					Namespace: namespace,
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Post sustained load stability verification",
					IntentType: nephoran.IntentTypeScaling,
					Priority:   1,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, stabilityIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: "post-sustained-load-stability", Namespace: namespace}
			retrievedIntent := &nephoran.NetworkIntent{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
				if err != nil {
					return false
				}
				return retrievedIntent.Status.Phase == "Processing"
			}, 20*time.Second, 1*time.Second).Should(BeTrue())

			Expect(k8sClient.Delete(ctx, retrievedIntent)).Should(Succeed())
		})
	})
})
