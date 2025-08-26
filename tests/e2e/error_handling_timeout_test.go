package e2e

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"
)

var _ = Describe("Error Handling and Timeout E2E Tests", func() {
	var (
		ctx       context.Context
		namespace string
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = "default"
	})

	Context("Input Validation and Error Handling", func() {
		It("should reject invalid NetworkIntent specifications", func() {
			testCases := []struct {
				name        string
				intent      *nephoran.NetworkIntent
				shouldFail  bool
				description string
			}{
				{
					name: "empty-intent-text",
					intent: &nephoran.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "empty-intent-test",
							Namespace: namespace,
						},
						Spec: nephoran.NetworkIntentSpec{
							Intent:     "", // Empty intent should be rejected
							IntentType: nephoran.IntentTypeScaling,
							Priority:   1,
						},
					},
					shouldFail:  true,
					description: "Empty intent text should be rejected",
				},
				{
					name: "invalid-priority",
					intent: &nephoran.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "invalid-priority-test",
							Namespace: namespace,
						},
						Spec: nephoran.NetworkIntentSpec{
							Intent:     "Valid intent text",
							IntentType: nephoran.IntentTypeScaling,
							Priority:   0, // Priority should be >= 1
						},
					},
					shouldFail:  true,
					description: "Invalid priority should be rejected",
				},
				{
					name: "no-target-components",
					intent: &nephoran.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "no-components-test",
							Namespace: namespace,
						},
						Spec: nephoran.NetworkIntentSpec{
							Intent:           "Valid intent text",
							IntentType:       nephoran.IntentTypeScaling,
							Priority:         1,
							TargetComponents: []nephoran.NetworkTargetComponent{}, // Empty components might be invalid
						},
					},
					shouldFail:  false, // This might be allowed depending on validation rules
					description: "Empty target components test",
				},
			}

			for _, tc := range testCases {
				By(fmt.Sprintf("Testing %s: %s", tc.name, tc.description))
				
				err := k8sClient.Create(ctx, tc.intent)
				
				if tc.shouldFail {
					Expect(err).Should(HaveOccurred())
					By(fmt.Sprintf("Correctly rejected invalid intent: %s", tc.name))
				} else {
					if err == nil {
						// If creation succeeded, clean up
						Expect(k8sClient.Delete(ctx, tc.intent)).Should(Succeed())
					}
				}
			}
		})

		It("should handle malformed intent text gracefully", func() {
			malformedIntents := []struct {
				name   string
				intent string
			}{
				{
					name:   "extremely-long-intent",
					intent: strings.Repeat("Scale network functions ", 1000), // Very long intent
				},
				{
					name:   "special-characters-intent",
					intent: "Intent with special chars: !@#$%^&*()[]{}|\\:;\"'<>,.?/~`",
				},
				{
					name:   "unicode-intent",
					intent: "Intent with Unicode: 网络意图 नेटवर्क इंटेंट شبكة القصد",
				},
				{
					name:   "json-injection-intent",
					intent: `{"malicious": "json", "injection": true}`,
				},
			}

			for i, tc := range malformedIntents {
				intentName := fmt.Sprintf("malformed-intent-%d", i)
				
				By(fmt.Sprintf("Testing malformed intent: %s", tc.name))
				
				networkIntent := &nephoran.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      intentName,
						Namespace: namespace,
					},
					Spec: nephoran.NetworkIntentSpec{
						Intent:     tc.intent,
						IntentType: nephoran.IntentTypeScaling,
						Priority:   1,
						TargetComponents: []nephoran.NetworkTargetComponent{
							nephoran.NetworkTargetComponentUPF,
						},
					},
				}

				err := k8sClient.Create(ctx, networkIntent)
				
				if err == nil {
					// If creation succeeded, verify controller handles it gracefully
					lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
					createdIntent := &nephoran.NetworkIntent{}

					By("Verifying controller processes malformed intent without crashing")
					Eventually(func() bool {
						err := k8sClient.Get(ctx, lookupKey, createdIntent)
						if err != nil {
							return false
						}
						
						// Controller should either process it or report an error gracefully
						return len(createdIntent.Status.Conditions) > 0
					}, 30*time.Second, 2*time.Second).Should(BeTrue())

					By("Verifying error conditions are properly reported for malformed input")
					hasCondition := false
					for _, condition := range createdIntent.Status.Conditions {
						if condition.Status == "False" || condition.Status == "Unknown" {
							Expect(condition.Message).ShouldNot(BeEmpty())
							hasCondition = true
							break
						}
					}
					
					// Should have at least some status reporting
					Expect(hasCondition || createdIntent.Status.Phase == "Processing").Should(BeTrue())

					// Clean up
					Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
				}
			}
		})
	})

	Context("Processing Timeout Scenarios", func() {
		It("should handle processing timeouts gracefully", func() {
			By("Creating intent designed to test timeout handling")
			intentName := "timeout-test-intent"
			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
					Annotations: map[string]string{
						"nephoran.io/test-timeout":  "true",
						"nephoran.io/complexity":    "high",
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent: "Process extremely complex network intent that should test timeout mechanisms: " +
						"Deploy comprehensive 5G network with advanced analytics, ML-based optimization, " +
						"multi-tier security, edge computing integration, network slicing for multiple " +
						"verticals including automotive, healthcare, and industrial IoT with custom SLAs",
					IntentType: nephoran.IntentTypeDeployment,
					Priority:   1,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentAMF,
						nephoran.NetworkTargetComponentSMF,
						nephoran.NetworkTargetComponentUPF,
						nephoran.NetworkTargetComponentNRF,
						nephoran.NetworkTargetComponentUDM,
						nephoran.NetworkTargetComponentUDR,
						nephoran.NetworkTargetComponentPCF,
						nephoran.NetworkTargetComponentAUSF,
						nephoran.NetworkTargetComponentNSSF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Monitoring for timeout handling in controller")
			timeoutDetected := false
			
			// Monitor for a reasonable time, but expect timeout handling
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}

				// Check for timeout-related conditions or messages
				for _, condition := range createdIntent.Status.Conditions {
					if strings.Contains(strings.ToLower(condition.Message), "timeout") ||
					   strings.Contains(strings.ToLower(condition.Message), "timed out") ||
					   strings.Contains(strings.ToLower(condition.Reason), "timeout") {
						timeoutDetected = true
						return true
					}
				}

				// Or check if it's still processing after reasonable time
				return createdIntent.Status.Phase == "Processing" && 
					   len(createdIntent.Status.Conditions) > 0
			}, 120*time.Second, 5*time.Second).Should(BeTrue())

			By("Verifying timeout is handled gracefully with proper error reporting")
			if timeoutDetected {
				By("Timeout was properly detected and reported")
				
				// Verify timeout condition has proper details
				hasTimeoutCondition := false
				for _, condition := range createdIntent.Status.Conditions {
					if strings.Contains(strings.ToLower(condition.Message), "timeout") {
						Expect(condition.Status).Should(Equal("False"))
						Expect(condition.LastTransitionTime).ShouldNot(BeNil())
						hasTimeoutCondition = true
						break
					}
				}
				Expect(hasTimeoutCondition).Should(BeTrue())
			} else {
				By("Intent processed without timeout, verifying proper status")
				Expect(createdIntent.Status.Phase).Should(BeElementOf([]string{"Processing", "Completed", "Error"}))
			}

			By("Cleaning up timeout test intent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})

		It("should recover from temporary processing failures", func() {
			By("Creating intent to test failure recovery")
			intentName := "recovery-test-intent"
			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
					Annotations: map[string]string{
						"nephoran.io/test-recovery": "true",
						"nephoran.io/simulate-failure": "transient",
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Test recovery from transient failures during processing",
					IntentType: nephoran.IntentTypeConfiguration,
					Priority:   1,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Monitoring recovery behavior")
			var statusProgression []string
			
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}

				currentPhase := createdIntent.Status.Phase
				if len(statusProgression) == 0 || statusProgression[len(statusProgression)-1] != currentPhase {
					statusProgression = append(statusProgression, currentPhase)
				}

				// Look for recovery indicators
				for _, condition := range createdIntent.Status.Conditions {
					if strings.Contains(strings.ToLower(condition.Message), "retry") ||
					   strings.Contains(strings.ToLower(condition.Message), "recover") ||
					   condition.Type == "Recovering" {
						return true
					}
				}

				// Or successful processing after some time
				return currentPhase == "Processing" && len(createdIntent.Status.Conditions) > 0
			}, 90*time.Second, 3*time.Second).Should(BeTrue())

			By("Verifying recovery progression")
			Expect(len(statusProgression)).Should(BeNumerically(">=", 1))
			
			// Should have some status updates indicating activity
			Expect(createdIntent.Status.LastProcessed).ShouldNot(BeNil())
			Expect(len(createdIntent.Status.Conditions)).Should(BeNumerically(">=", 1))

			By("Cleaning up recovery test intent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})
	})

	Context("Resource Constraint and Limit Handling", func() {
		It("should handle resource constraint errors appropriately", func() {
			By("Creating intent that might hit resource constraints")
			intentName := "resource-constraint-intent"
			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
					Annotations: map[string]string{
						"nephoran.io/test-constraints": "true",
						"nephoran.io/resource-intensive": "true",
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent: "Deploy resource-intensive configuration that may exceed cluster capacity: " +
						"Launch 100 UPF instances with high memory requirements for extreme load testing",
					IntentType: nephoran.IntentTypeDeployment,
					Priority:   1,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Monitoring for resource constraint handling")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}

				// Look for resource-related error conditions
				for _, condition := range createdIntent.Status.Conditions {
					message := strings.ToLower(condition.Message)
					if strings.Contains(message, "resource") ||
					   strings.Contains(message, "capacity") ||
					   strings.Contains(message, "insufficient") ||
					   strings.Contains(message, "quota") ||
					   strings.Contains(message, "limit") {
						return true
					}
				}

				// Or normal processing
				return len(createdIntent.Status.Conditions) > 0
			}, 60*time.Second, 3*time.Second).Should(BeTrue())

			By("Verifying appropriate error handling for resource constraints")
			hasResourceCondition := false
			for _, condition := range createdIntent.Status.Conditions {
				message := strings.ToLower(condition.Message)
				if strings.Contains(message, "resource") || 
				   strings.Contains(message, "capacity") ||
				   condition.Status == "False" {
					Expect(condition.Message).ShouldNot(BeEmpty())
					Expect(condition.LastTransitionTime).ShouldNot(BeNil())
					hasResourceCondition = true
					break
				}
			}

			// Should have some form of status reporting
			Expect(hasResourceCondition || createdIntent.Status.Phase != "").Should(BeTrue())

			By("Cleaning up resource constraint test intent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})

		It("should handle network connectivity errors gracefully", func() {
			By("Creating intent to test network error handling")
			intentName := "network-error-intent"
			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
					Annotations: map[string]string{
						"nephoran.io/test-network-errors": "true",
						"nephoran.io/external-dependencies": "true",
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Test network connectivity error handling for external service dependencies",
					IntentType: nephoran.IntentTypeConfiguration,
					Priority:   2,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentAMF,
						nephoran.NetworkTargetComponentSMF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Monitoring for network error handling")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}

				// Look for network-related conditions
				for _, condition := range createdIntent.Status.Conditions {
					message := strings.ToLower(condition.Message)
					reason := strings.ToLower(condition.Reason)
					if strings.Contains(message, "connection") ||
					   strings.Contains(message, "network") ||
					   strings.Contains(message, "unreachable") ||
					   strings.Contains(reason, "network") {
						return true
					}
				}

				// Or standard processing
				return createdIntent.Status.Phase == "Processing" && len(createdIntent.Status.Conditions) > 0
			}, 45*time.Second, 3*time.Second).Should(BeTrue())

			By("Verifying network error conditions are properly documented")
			// Should have status information regardless of network issues
			Expect(len(createdIntent.Status.Conditions)).Should(BeNumerically(">=", 1))
			Expect(createdIntent.Status.LastProcessed).ShouldNot(BeNil())

			By("Cleaning up network error test intent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})
	})

	Context("Concurrent Error Scenarios", func() {
		It("should handle errors during concurrent processing", func() {
			By("Creating multiple intents that may cause concurrent errors")
			concurrentErrorIntents := 3
			var createdIntents []*nephoran.NetworkIntent

			for i := 0; i < concurrentErrorIntents; i++ {
				intentName := fmt.Sprintf("concurrent-error-intent-%d", i)
				intent := &nephoran.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      intentName,
						Namespace: namespace,
						Annotations: map[string]string{
							"nephoran.io/test-concurrent-errors": "true",
							"nephoran.io/error-prone":            "true",
						},
					},
					Spec: nephoran.NetworkIntentSpec{
						Intent:     fmt.Sprintf("Concurrent error test %d: Process complex intent with potential conflicts", i),
						IntentType: nephoran.IntentTypeDeployment,
						Priority:   1,
						TargetComponents: []nephoran.NetworkTargetComponent{
							nephoran.NetworkTargetComponentUPF,
							nephoran.NetworkTargetComponentSMF,
						},
					},
				}

				Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
				createdIntents = append(createdIntents, intent)
			}

			By("Monitoring concurrent error handling")
			for _, intent := range createdIntents {
				lookupKey := types.NamespacedName{Name: intent.Name, Namespace: namespace}
				retrievedIntent := &nephoran.NetworkIntent{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
					if err != nil {
						return false
					}
					
					// Should have some status, whether success or error
					return len(retrievedIntent.Status.Conditions) > 0 || retrievedIntent.Status.Phase != ""
				}, 60*time.Second, 2*time.Second).Should(BeTrue())
			}

			By("Verifying each intent maintains independent error handling")
			for i, intent := range createdIntents {
				lookupKey := types.NamespacedName{Name: intent.Name, Namespace: namespace}
				retrievedIntent := &nephoran.NetworkIntent{}

				err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
				Expect(err).ShouldNot(HaveOccurred())

				// Each intent should maintain its identity
				expectedSuffix := fmt.Sprintf("test %d:", i)
				Expect(retrievedIntent.Spec.Intent).Should(ContainSubstring(expectedSuffix))

				// Should have some processing status
				Expect(retrievedIntent.Status.LastProcessed).ShouldNot(BeNil())
			}

			By("Cleaning up concurrent error test intents")
			for _, intent := range createdIntents {
				Expect(k8sClient.Delete(ctx, intent)).Should(Succeed())
			}
		})
	})

	Context("System Recovery and Stability", func() {
		It("should maintain system stability after error scenarios", func() {
			By("Creating and processing various error-inducing intents")
			errorScenarios := []string{
				"system-stability-test-1",
				"system-stability-test-2", 
				"system-stability-test-3",
			}

			// Process each error scenario
			for _, scenarioName := range errorScenarios {
				intent := &nephoran.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      scenarioName,
						Namespace: namespace,
						Annotations: map[string]string{
							"nephoran.io/stability-test": "true",
						},
					},
					Spec: nephoran.NetworkIntentSpec{
						Intent:     fmt.Sprintf("System stability test: %s", scenarioName),
						IntentType: nephoran.IntentTypeConfiguration,
						Priority:   1,
						TargetComponents: []nephoran.NetworkTargetComponent{
							nephoran.NetworkTargetComponentUPF,
						},
					},
				}

				Expect(k8sClient.Create(ctx, intent)).Should(Succeed())

				// Wait for processing to begin
				lookupKey := types.NamespacedName{Name: scenarioName, Namespace: namespace}
				createdIntent := &nephoran.NetworkIntent{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, lookupKey, createdIntent)
					if err != nil {
						return false
					}
					return len(createdIntent.Status.Conditions) > 0
				}, 30*time.Second, 2*time.Second).Should(BeTrue())

				// Clean up immediately
				Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
			}

			By("Verifying system can still process normal intents after error scenarios")
			finalTestIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "post-error-stability-test",
					Namespace: namespace,
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Final stability verification: normal intent processing after errors",
					IntentType: nephoran.IntentTypeScaling,
					Priority:   1,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, finalTestIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: "post-error-stability-test", Namespace: namespace}
			retrievedIntent := &nephoran.NetworkIntent{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
				if err != nil {
					return false
				}
				return retrievedIntent.Status.Phase == "Processing"
			}, 30*time.Second, 2*time.Second).Should(BeTrue())

			By("Verifying normal processing is restored")
			Expect(retrievedIntent.Status.LastProcessed).ShouldNot(BeNil())
			Expect(len(retrievedIntent.Status.Conditions)).Should(BeNumerically(">=", 1))

			By("Cleaning up final stability test intent")
			Expect(k8sClient.Delete(ctx, retrievedIntent)).Should(Succeed())
		})
	})
})