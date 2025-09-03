package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"
)

var _ = Describe("Health Monitoring E2E Tests", func() {
	var (
		ctx        context.Context
		namespace  string
		httpClient *http.Client
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = "default"
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	})

	Context("Controller Health Monitoring", func() {
		It("should maintain healthy controller metrics", func() {
			By("Creating a NetworkIntent to trigger controller activity")
			intentName := "health-monitoring-intent"
			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Health monitoring test for controller",
					IntentType: nephoran.IntentTypeScaling,
					Priority:   nephoran.NetworkPriorityHigh,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			By("Verifying controller processes the intent")
			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			Eventually(func() string {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return ""
				}
				return string(createdIntent.Status.Phase)
			}, 30*time.Second, 2*time.Second).Should(Equal("Processing"))

			By("Verifying controller health through status conditions")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}
				return len(createdIntent.Status.Conditions) > 0
			}, 20*time.Second, 1*time.Second).Should(BeTrue())

			// Verify at least one condition is present
			Expect(createdIntent.Status.Conditions[0].Type).ShouldNot(BeEmpty())
			Expect(createdIntent.Status.Conditions[0].Status).Should(BeElementOf([]string{"True", "False", "Unknown"}))

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})

		It("should report controller errors through status conditions", func() {
			By("Creating an intent that might cause processing issues")
			errorIntentName := "error-inducing-intent"
			errorIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      errorIntentName,
					Namespace: namespace,
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "This intent is designed to test error handling and should trigger error conditions",
					IntentType: "scaling",
					Priority:   nephoran.NetworkPriorityHigh,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
						nephoran.NetworkTargetComponentSMF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, errorIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: errorIntentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Monitoring for error conditions in status")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}

				for _, condition := range createdIntent.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == "False" {
						return condition.Reason != ""
					}
				}
				return false
			}, 45*time.Second, 3*time.Second).Should(BeTrue())

			By("Verifying error details are captured")
			hasErrorCondition := false
			for _, condition := range createdIntent.Status.Conditions {
				if condition.Status == "False" {
					Expect(condition.Message).ShouldNot(BeEmpty())
					Expect(condition.LastTransitionTime).ShouldNot(BeNil())
					hasErrorCondition = true
					break
				}
			}
			Expect(hasErrorCondition).Should(BeTrue())

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})
	})

	Context("Service Health Endpoints", func() {
		It("should provide comprehensive health status for all services", func() {
			Skip("Skipping until services are running in test environment")

			services := []struct {
				name string
				url  string
				port string
			}{
				{"controller", "http://localhost:8080", "8080"},
				{"llm-processor", "http://localhost:8080", "8080"},
				{"rag-service", "http://localhost:8001", "8001"},
			}

			for _, service := range services {
				By(fmt.Sprintf("Checking health for %s service", service.name))

				resp, err := httpClient.Get(service.url + "/health")
				if err != nil {
					Skip(fmt.Sprintf("%s service not available: %v", service.name, err))
				}
				defer resp.Body.Close() // #nosec G307 - Error handled in defer

				Expect(resp.StatusCode).Should(Equal(http.StatusOK))

				body, err := io.ReadAll(resp.Body)
				Expect(err).ShouldNot(HaveOccurred())

				var healthResponse map[string]interface{}
				err = json.Unmarshal(body, &healthResponse)
				Expect(err).ShouldNot(HaveOccurred())

				By(fmt.Sprintf("Verifying %s health response structure", service.name))
				Expect(healthResponse["status"]).Should(Equal("healthy"))
				Expect(healthResponse["timestamp"]).ShouldNot(BeEmpty())

				if uptime, ok := healthResponse["uptime_seconds"]; ok {
					Expect(uptime).Should(BeNumerically(">=", 0))
				}
			}
		})

		It("should provide detailed metrics endpoints", func() {
			Skip("Skipping until services are running in test environment")

			services := []string{
				"http://localhost:8080/metrics", // Controller metrics
				"http://localhost:8001/metrics", // RAG service metrics
			}

			for _, metricsURL := range services {
				By(fmt.Sprintf("Checking metrics endpoint: %s", metricsURL))

				resp, err := httpClient.Get(metricsURL)
				if err != nil {
					Skip(fmt.Sprintf("Metrics endpoint not available: %v", err))
				}
				defer resp.Body.Close() // #nosec G307 - Error handled in defer

				Expect(resp.StatusCode).Should(Equal(http.StatusOK))

				body, err := io.ReadAll(resp.Body)
				Expect(err).ShouldNot(HaveOccurred())

				metricsContent := string(body)
				By("Verifying Prometheus metrics format")
				Expect(metricsContent).Should(ContainSubstring("# HELP"))
				Expect(metricsContent).Should(ContainSubstring("# TYPE"))
			}
		})
	})

	Context("Resource Health and Resource Usage", func() {
		It("should monitor and report resource usage", func() {
			By("Creating multiple NetworkIntents to generate load")
			intentNames := []string{"resource-test-1", "resource-test-2", "resource-test-3"}
			var createdIntents []*nephoran.NetworkIntent

			for _, name := range intentNames {
				intent := &nephoran.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: nephoran.NetworkIntentSpec{
						Intent:     fmt.Sprintf("Resource monitoring test intent: %s", name),
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

			By("Waiting for all intents to be processed")
			for _, intent := range createdIntents {
				lookupKey := types.NamespacedName{Name: intent.Name, Namespace: namespace}
				retrievedIntent := &nephoran.NetworkIntent{}

				Eventually(func() string {
					err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
					if err != nil {
						return ""
					}
					return string(retrievedIntent.Status.Phase)
				}, 30*time.Second, 2*time.Second).Should(Equal("Processing"))
			}

			By("Verifying system remains healthy under load")
			// All intents should maintain healthy status
			for _, intent := range createdIntents {
				lookupKey := types.NamespacedName{Name: intent.Name, Namespace: namespace}
				retrievedIntent := &nephoran.NetworkIntent{}

				err := k8sClient.Get(ctx, lookupKey, retrievedIntent)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(retrievedIntent.Status.Phase).Should(Equal("Processing"))
			}

			By("Cleaning up all test intents")
			for _, intent := range createdIntents {
				Expect(k8sClient.Delete(ctx, intent)).Should(Succeed())
			}
		})

		It("should handle resource cleanup gracefully", func() {
			By("Creating and deleting intents rapidly")
			for i := 0; i < 5; i++ {
				intentName := fmt.Sprintf("cleanup-test-%d", i)

				intent := &nephoran.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      intentName,
						Namespace: namespace,
					},
					Spec: nephoran.NetworkIntentSpec{
						Intent:     fmt.Sprintf("Cleanup test intent %d", i),
						IntentType: nephoran.IntentTypeScaling,
						Priority:   nephoran.NetworkPriorityHigh,
						TargetComponents: []nephoran.NetworkTargetComponent{
							nephoran.NetworkTargetComponentUPF,
						},
					},
				}

				Expect(k8sClient.Create(ctx, intent)).Should(Succeed())

				// Wait a short time then delete
				time.Sleep(500 * time.Millisecond)
				Expect(k8sClient.Delete(ctx, intent)).Should(Succeed())
			}

			By("Verifying no lingering resources")
			time.Sleep(5 * time.Second) // Allow cleanup to complete

			// List all NetworkIntents and verify our test ones are gone
			intentList := &nephoran.NetworkIntentList{}
			err := k8sClient.List(ctx, intentList, &client.ListOptions{Namespace: namespace})
			Expect(err).ShouldNot(HaveOccurred())

			for _, intent := range intentList.Items {
				Expect(intent.Name).ShouldNot(ContainSubstring("cleanup-test-"))
			}
		})
	})

	Context("End-to-End Health Validation", func() {
		It("should validate complete system health after processing intents", func() {
			By("Creating a comprehensive test intent")
			e2eIntentName := "e2e-health-validation"
			e2eIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      e2eIntentName,
					Namespace: namespace,
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "End-to-end health validation: Deploy and scale 5G core network with monitoring",
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

			Expect(k8sClient.Create(ctx, e2eIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: e2eIntentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Monitoring intent processing health")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}
				return string(createdIntent.Status.Phase) == "Processing"
			}, 45*time.Second, 3*time.Second).Should(BeTrue())

			By("Verifying comprehensive status reporting")
			Expect(len(createdIntent.Status.Conditions)).Should(BeNumerically(">=", 0))
			Expect(len(createdIntent.Status.Conditions)).Should(BeNumerically(">=", 1))

			// Check for expected condition types
			conditionTypes := make(map[string]bool)
			for _, condition := range createdIntent.Status.Conditions {
				conditionTypes[condition.Type] = true
			}

			// Should have at least basic condition types
			expectedTypes := []string{"Ready", "Processing"}
			foundExpectedType := false
			for _, expectedType := range expectedTypes {
				if conditionTypes[expectedType] {
					foundExpectedType = true
					break
				}
			}
			Expect(foundExpectedType).Should(BeTrue())

			By("Verifying intent metadata preservation")
			Expect(createdIntent.Spec.Intent).Should(Equal("End-to-end health validation: Deploy and scale 5G core network with monitoring"))
			Expect(len(createdIntent.Spec.TargetComponents)).Should(Equal(4))

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})

		It("should recover from temporary failures", func() {
			By("Simulating recovery from processing errors")
			recoveryIntentName := "recovery-test-intent"
			recoveryIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      recoveryIntentName,
					Namespace: namespace,
					Annotations: map[string]string{
						"nephoran.io/test-recovery": "true",
					},
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Test recovery mechanisms and error handling resilience",
					IntentType: nephoran.IntentTypeOptimization,
					Priority:   nephoran.NetworkPriorityHigh,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			Expect(k8sClient.Create(ctx, recoveryIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: recoveryIntentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			By("Waiting for initial processing")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				if err != nil {
					return false
				}
				return len(createdIntent.Status.Conditions) > 0
			}, 30*time.Second, 2*time.Second).Should(BeTrue())

			By("Verifying system continues processing despite potential errors")
			// The system should either process successfully or handle errors gracefully
			Expect(createdIntent.Status.Phase).Should(BeElementOf([]string{"Processing", "Error", "Completed"}))

			if createdIntent.Status.Phase == "Error" {
				By("Verifying error is properly documented")
				hasErrorCondition := false
				for _, condition := range createdIntent.Status.Conditions {
					if condition.Status == "False" && condition.Message != "" {
						hasErrorCondition = true
						break
					}
				}
				Expect(hasErrorCondition).Should(BeTrue())
			}

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})
	})
})
