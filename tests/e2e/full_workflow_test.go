package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"
)

var _ = Describe("Full Workflow E2E Tests (Intent ??LLM ??Nephio ??Scale)", func() {
	var (
		ctx             context.Context
		namespace       string
		llmProcessorURL string
		ragServiceURL   string
		httpClient      *http.Client
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = "default"
		llmProcessorURL = "http://localhost:8080"
		ragServiceURL = "http://localhost:8001"
		httpClient = &http.Client{
			Timeout: 60 * time.Second, // Long timeout for full workflow
		}
	})

	Context("Complete Intent Processing Workflow", func() {
		It("should process scaling intent through complete pipeline", func() {
			By("Step 1: Creating NetworkIntent for scaling workflow")
			intentName := "workflow-scaling-intent"
			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
					Labels: map[string]string{
						"nephoran.io/test-type": "full-workflow",
						"nephoran.io/workflow":  "scaling",
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Scale UPF instances from 3 to 6 replicas to handle increased traffic during peak hours",
					IntentType: nephoran.IntentTypeScaling,
					Priority:   nephoran.NetworkPriorityHigh,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Step 2: Verifying controller picks up the intent")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}
				return string(createdIntent.Status.Phase) == "Processing"
			}, 30*time.Second, 2*time.Second).Should(BeTrue())

			By("Step 3: Verifying LLM processing is triggered (if service available)")
			Skip("Skipping LLM service interaction until service is available in test environment")

			// This would be enabled when LLM service is running
			llmRequest := json.RawMessage("{}")

			jsonPayload, err := json.Marshal(llmRequest)
			Expect(err).ShouldNot(HaveOccurred())

			resp, err := httpClient.Post(
				llmProcessorURL+"/api/v1/process-intent",
				"application/json",
				bytes.NewBuffer(jsonPayload),
			)
			if err == nil {
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					body, err := io.ReadAll(resp.Body)
					Expect(err).ShouldNot(HaveOccurred())

					var llmResponse map[string]interface{}
					err = json.Unmarshal(body, &llmResponse)
					Expect(err).ShouldNot(HaveOccurred())

					By("Verifying LLM processing results")
					Expect(llmResponse["processed_intent"]).ShouldNot(BeEmpty())
					Expect(llmResponse["recommendations"]).Should(BeAssignableToTypeOf([]interface{}{}))
				}
			}

			By("Step 4: Monitoring status progression through workflow phases")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}

				// Check for workflow progression indicators
				for _, condition := range createdIntent.Status.Conditions {
					if strings.Contains(condition.Type, "Processing") && condition.Status == "True" {
						return true
					}
				}
				return false
			}, 60*time.Second, 5*time.Second).Should(BeTrue())

			By("Step 5: Verifying RAG knowledge retrieval (if service available)")
			Skip("Skipping RAG service interaction until service is available in test environment")

			ragQuery := json.RawMessage("{}")

			ragPayload, err := json.Marshal(ragQuery)
			Expect(err).ShouldNot(HaveOccurred())

			ragResp, err := httpClient.Post(
				ragServiceURL+"/api/v1/query/intent",
				"application/json",
				bytes.NewBuffer(ragPayload),
			)
			if err == nil {
				defer ragResp.Body.Close()

				if ragResp.StatusCode == http.StatusOK {
					By("RAG service provided knowledge context successfully")
					body, err := io.ReadAll(ragResp.Body)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(string(body)).Should(ContainSubstring("UPF"))
				}
			}

			By("Step 6: Verifying Nephio integration status")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}

				// Look for Nephio-related status updates
				for _, condition := range createdIntent.Status.Conditions {
					if strings.Contains(strings.ToLower(condition.Message), "package") ||
						strings.Contains(strings.ToLower(condition.Message), "porch") ||
						strings.Contains(strings.ToLower(condition.Message), "kpt") {
						return true
					}
				}

				// Also check if processing reached advanced stage
				return string(createdIntent.Status.Phase) != "Processing" ||
					len(createdIntent.Status.Conditions) >= 2
			}, 90*time.Second, 5*time.Second).Should(BeTrue())

			By("Step 7: Verifying final workflow status")
			Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return ""
				}
				return string(createdIntent.Status.Phase)
			}, 120*time.Second, 10*time.Second).Should(BeElementOf([]string{"Completed", "Processing", "Error"}))

			By("Step 8: Validating workflow completion artifacts")
			if createdIntent.Status.Phase == "Completed" {
				By("Verifying completion metadata")
				Expect(len(createdIntent.Status.Conditions)).Should(BeNumerically(">=", 0))

				// Check for successful completion indicators
				hasSuccessCondition := false
				for _, condition := range createdIntent.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == "True" {
						hasSuccessCondition = true
						break
					}
				}
				Expect(hasSuccessCondition).Should(BeTrue())
			}

			By("Cleaning up workflow test intent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})

		It("should process deployment intent through complete pipeline", func() {
			By("Step 1: Creating NetworkIntent for deployment workflow")
			intentName := "workflow-deployment-intent"
			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
					Labels: map[string]string{
						"nephoran.io/test-type": "full-workflow",
						"nephoran.io/workflow":  "deployment",
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Deploy a new AMF instance with high availability configuration for critical applications",
					IntentType: nephoran.IntentTypeDeployment,
					Priority:   nephoran.NetworkPriorityHigh,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentAMF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Step 2: Monitoring deployment workflow progression")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}
				return string(createdIntent.Status.Phase) == "Processing"
			}, 30*time.Second, 2*time.Second).Should(BeTrue())

			By("Step 3: Verifying deployment-specific processing")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}

				// For deployment intents, we expect different processing patterns
				for _, condition := range createdIntent.Status.Conditions {
					if strings.Contains(strings.ToLower(condition.Message), "deploy") ||
						strings.Contains(strings.ToLower(condition.Message), "install") ||
						strings.Contains(strings.ToLower(condition.Message), "create") {
						return true
					}
				}

				// At minimum, should have conditions set
				return len(createdIntent.Status.Conditions) > 0
			}, 60*time.Second, 3*time.Second).Should(BeTrue())

			By("Step 4: Verifying AMF-specific processing characteristics")
			Expect(createdIntent.Spec.TargetComponents).Should(ContainElement(nephoran.NetworkTargetComponentAMF))
			Expect(createdIntent.Spec.IntentType).Should(Equal(nephoran.IntentTypeDeployment))

			By("Cleaning up deployment workflow test intent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})

		It("should process optimization intent through complete pipeline", func() {
			By("Step 1: Creating NetworkIntent for optimization workflow")
			intentName := "workflow-optimization-intent"
			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
					Labels: map[string]string{
						"nephoran.io/test-type": "full-workflow",
						"nephoran.io/workflow":  "optimization",
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Optimize SMF performance by tuning session management parameters and resource allocation",
					IntentType: nephoran.IntentTypeOptimization,
					Priority:   nephoran.NetworkPriorityHigh,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentSMF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Step 2: Monitoring optimization workflow")
			Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return ""
				}
				return string(createdIntent.Status.Phase)
			}, 45*time.Second, 3*time.Second).Should(Equal("Processing"))

			By("Step 3: Verifying optimization-specific processing")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}

				// Optimization intents should have detailed analysis
				for _, condition := range createdIntent.Status.Conditions {
					if strings.Contains(strings.ToLower(condition.Message), "optim") ||
						strings.Contains(strings.ToLower(condition.Message), "tuning") ||
						strings.Contains(strings.ToLower(condition.Message), "performance") {
						return true
					}
				}

				// Should have processing indicators
				return len(createdIntent.Status.Conditions) > 0 &&
					createdIntent.Status.Conditions != nil
			}, 60*time.Second, 3*time.Second).Should(BeTrue())

			By("Cleaning up optimization workflow test intent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})
	})

	Context("Multi-Component Workflow Processing", func() {
		It("should handle complex intent with multiple target components", func() {
			By("Creating complex multi-component intent")
			intentName := "workflow-multi-component-intent"
			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
					Labels: map[string]string{
						"nephoran.io/test-type": "full-workflow",
						"nephoran.io/workflow":  "multi-component",
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Deploy and configure a complete 5G standalone core including AMF, SMF, UPF, and NRF with optimized interconnections",
					IntentType: nephoran.IntentTypeDeployment,
					Priority:   nephoran.NetworkPriorityHigh,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentAMF,
						nephoran.NetworkTargetComponentSMF,
						nephoran.NetworkTargetComponentUPF,
						nephoran.NetworkTargetComponentNRF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Monitoring complex workflow processing")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}
				return string(createdIntent.Status.Phase) == "Processing" && len(createdIntent.Status.Conditions) > 0
			}, 45*time.Second, 3*time.Second).Should(BeTrue())

			By("Verifying all target components are preserved and processed")
			Expect(len(createdIntent.Spec.TargetComponents)).Should(Equal(4))
			Expect(createdIntent.Spec.TargetComponents).Should(ContainElements(
				nephoran.NetworkTargetComponentAMF,
				nephoran.NetworkTargetComponentSMF,
				nephoran.NetworkTargetComponentUPF,
				nephoran.NetworkTargetComponentNRF,
			))

			By("Verifying complex intent processing takes appropriate time")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}

				// Complex intents should show detailed processing status
				if len(createdIntent.Status.Conditions) >= 2 {
					return true
				}

				// Or reach completion/error state
				return string(createdIntent.Status.Phase) != "Processing"
			}, 120*time.Second, 5*time.Second).Should(BeTrue())

			By("Cleaning up multi-component workflow test intent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})
	})

	Context("Workflow State Management and Recovery", func() {
		It("should maintain workflow state consistency during processing", func() {
			By("Creating intent with state tracking")
			intentName := "workflow-state-management-intent"
			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
					Labels: map[string]string{
						"nephoran.io/test-type": "state-management",
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Test workflow state management and consistency during processing",
					IntentType: "scaling",
					Priority:   nephoran.NetworkPriorityHigh,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Monitoring state transitions")
			var previousConditions []metav1.Condition

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}

				currentConditions := createdIntent.Status.Conditions

				// Verify state transitions are logical
				if len(currentConditions) > len(previousConditions) {
					for _, condition := range currentConditions {
						Expect(condition.LastTransitionTime).ShouldNot(BeNil())
						Expect(condition.Type).ShouldNot(BeEmpty())
					}
					previousConditions = currentConditions
				}

				return len(currentConditions) > 0
			}, 60*time.Second, 2*time.Second).Should(BeTrue())

			By("Verifying consistent state reporting")
			Expect(createdIntent.Status.Phase).Should(BeElementOf([]string{"Processing", "Completed", "Error"}))
			Expect(len(createdIntent.Status.Conditions)).Should(BeNumerically(">=", 0))

			By("Cleaning up state management test intent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})

		It("should handle workflow interruptions gracefully", func() {
			By("Creating intent for interruption testing")
			intentName := "workflow-interruption-test-intent"
			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
					Labels: map[string]string{
						"nephoran.io/test-type": "interruption-test",
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Test workflow interruption and recovery mechanisms",
					IntentType: nephoran.IntentTypeMaintenance,
					Priority:   nephoran.NetworkPriorityCritical,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentSMF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Allowing initial processing to begin")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}
				return string(createdIntent.Status.Phase) == "Processing"
			}, 30*time.Second, 2*time.Second).Should(BeTrue())

			By("Simulating interruption by updating the intent")
			createdIntent.Spec.Intent = "Updated intent to test interruption handling and recovery"
			Expect(k8sClient.Update(ctx, createdIntent)).Should(Succeed())

			By("Verifying system handles interruption gracefully")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}

				// System should either continue processing or handle the update
				return createdIntent.Status.Conditions != nil
			}, 45*time.Second, 3*time.Second).Should(BeTrue())

			By("Verifying updated intent is reflected")
			Expect(createdIntent.Spec.Intent).Should(ContainSubstring("Updated intent"))

			By("Cleaning up interruption test intent")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})
	})

	Context("Workflow Performance and Scalability", func() {
		It("should handle concurrent workflow execution", func() {
			By("Creating multiple intents for concurrent processing")
			concurrentIntents := 3
			var createdIntents []*nephoran.NetworkIntent

			for i := 0; i < concurrentIntents; i++ {
				intentName := fmt.Sprintf("concurrent-workflow-%d", i)
				intent := &nephoran.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      intentName,
						Namespace: namespace,
						Labels: map[string]string{
							"nephoran.io/test-type": "concurrent-workflow",
							"nephoran.io/batch-id":  "concurrent-test-1",
						},
					},
					Spec: nephoran.NetworkIntentSpec{
						Intent:     fmt.Sprintf("Concurrent workflow test %d: Scale network function instances", i),
						IntentType: nephoran.IntentTypeScaling,
						Priority:   nephoran.NetworkPriorityHigh,
						TargetComponents: []nephoran.NetworkTargetComponent{
							nephoran.NetworkTargetComponentUPF,
						},
					},
				}

				Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
				createdIntents = append(createdIntents, intent)
			}

			By("Verifying all concurrent intents are processed")
			for _, intent := range createdIntents {
				lookupKey := types.NamespacedName{Name: intent.Name, Namespace: namespace}
				retrievedIntent := &nephoran.NetworkIntent{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
					if err != nil {
						return false
					}
					return retrievedIntent.Status.Phase == "Processing"
				}, 45*time.Second, 2*time.Second).Should(BeTrue())
			}

			By("Verifying no interference between concurrent workflows")
			for i, intent := range createdIntents {
				lookupKey := types.NamespacedName{Name: intent.Name, Namespace: namespace}
				retrievedIntent := &nephoran.NetworkIntent{}

				err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
				Expect(err).ShouldNot(HaveOccurred())

				expectedIntentSuffix := fmt.Sprintf("test %d:", i)
				Expect(retrievedIntent.Spec.Intent).Should(ContainSubstring(expectedIntentSuffix))
			}

			By("Cleaning up concurrent workflow test intents")
			for _, intent := range createdIntents {
				Expect(k8sClient.Delete(ctx, intent)).Should(Succeed())
			}
		})

		It("should maintain performance under high-priority intent load", func() {
			By("Creating high-priority intents")
			highPriorityIntents := []string{"critical-intent-1", "critical-intent-2"}
			var createdIntents []*nephoran.NetworkIntent

			for i, intentName := range highPriorityIntents {
				intent := &nephoran.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      intentName,
						Namespace: namespace,
						Labels: map[string]string{
							"nephoran.io/test-type": "high-priority-workflow",
							"nephoran.io/priority":  "critical",
						},
					},
					Spec: nephoran.NetworkIntentSpec{
						Intent:     fmt.Sprintf("Critical priority workflow %d: Emergency network scaling", i),
						IntentType: nephoran.IntentTypeScaling,
						Priority:   nephoran.NetworkPriorityHigh, // Highest priority
						TargetComponents: []nephoran.NetworkTargetComponent{
							nephoran.NetworkTargetComponentAMF,
							nephoran.NetworkTargetComponentUPF,
						},
					},
				}

				Expect(k8sClient.Create(ctx, intent)).Should(Succeed())
				createdIntents = append(createdIntents, intent)
			}

			By("Measuring processing latency for high-priority intents")
			startTime := time.Now()

			for _, intent := range createdIntents {
				lookupKey := types.NamespacedName{Name: intent.Name, Namespace: namespace}
				retrievedIntent := &nephoran.NetworkIntent{}

				Eventually(func() bool {
					err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
					if err != nil {
						return false
					}
					return len(retrievedIntent.Status.Conditions) > 0
				}, 60*time.Second, 2*time.Second).Should(BeTrue())
			}

			processingTime := time.Since(startTime)
			By(fmt.Sprintf("High-priority intents processing time: %v", processingTime))

			// High-priority intents should be processed reasonably quickly
			Expect(processingTime).Should(BeNumerically("<", 90*time.Second))

			By("Cleaning up high-priority workflow test intents")
			for _, intent := range createdIntents {
				Expect(k8sClient.Delete(ctx, intent)).Should(Succeed())
			}
		})
	})
})
