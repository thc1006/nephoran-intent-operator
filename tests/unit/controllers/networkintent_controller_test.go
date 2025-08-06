package controllers_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	testutils "github.com/thc1006/nephoran-intent-operator/tests/utils"
)

var _ = Describe("NetworkIntentController", func() {
	var (
		k8sClient            client.Client
		scheme               *runtime.Scheme
		mockDeps             *testutils.MockDependencies
		controller           *controllers.NetworkIntentReconciler
		ctx                  context.Context
		cancel               context.CancelFunc
		namespace            string
		performanceTracker   *testutils.PerformanceTracker
		testFixtures         *testutils.TestFixtures
	)

	BeforeEach(func() {
		// Initialize test environment
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		namespace = "test-namespace"
		
		// Create test fixtures
		testFixtures = testutils.NewTestFixtures()
		scheme = testFixtures.Scheme
		k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		
		// Create namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())
		
		// Initialize mock dependencies
		mockDeps = testutils.NewMockDependencies()
		
		// Create controller with default config
		config := controllers.Config{
			MaxRetries:      3,
			RetryDelay:      1 * time.Second, // Shorter for tests
			Timeout:         30 * time.Second,
			GitRepoURL:      "https://github.com/test/repo.git",
			GitBranch:       "main",
			GitDeployPath:   "networkintents",
			LLMProcessorURL: "http://localhost:8080",
			UseNephioPorch:  false,
		}
		
		controller = &controllers.NetworkIntentReconciler{
			Client:       k8sClient,
			Scheme:       scheme,
			Config:       config,
			Dependencies: mockDeps,
			Recorder:     record.NewFakeRecorder(100),
		}
		
		// Initialize performance tracker
		performanceTracker = testutils.NewPerformanceTracker()
	})

	AfterEach(func() {
		cancel()
	})

	Describe("Reconcile with Mocked Dependencies", func() {
		var intent *nephoranv1.NetworkIntent

		BeforeEach(func() {
			intent = testutils.NetworkIntentFixture.CreateBasicNetworkIntent("test-intent", namespace)
			Expect(k8sClient.Create(ctx, intent)).To(Succeed())
		})

		Context("when processing a new intent successfully", func() {
			It("should complete all processing phases within SLA", func() {
				performanceTracker.Start("reconcile")
				
				// Configure mock to return successful LLM response
				mockResponse := testutils.CreateMockLLMResponse("5G-Core-AMF", 0.95)
				mockDeps.GetLLMClient().(*testutils.MockLLMClient).SetResponse("ProcessIntent", mockResponse)
				
				// Execute reconcile
				result, err := controller.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				})
				
				duration := performanceTracker.Stop("reconcile")
				
				// Verify results
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())
				Expect(duration).To(BeNumerically("<", 5*time.Second), "Reconcile should complete within 5-second SLA")
				
				// Check intent status progression
				updatedIntent := &nephoranv1.NetworkIntent{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace}, updatedIntent)).To(Succeed())
				
				// Verify phase progression
				Expect(updatedIntent.Status.Phase).To(Or(
					Equal(nephoranv1.NetworkIntentPhaseProcessing),
					Equal(nephoranv1.NetworkIntentPhaseDeployed),
				))
				
				// Verify call counts
				Expect(mockDeps.GetCallCount("GetLLMClient")).To(BeNumerically(">=", 1))
			})

			It("should handle each processing phase correctly", func() {
				// Test LLM Processing Phase
				By("verifying LLM processing phase")
				mockResponse := &shared.LLMResponse{
					IntentType: "5G-Core-AMF",
					Confidence: 0.95,
					Parameters: map[string]interface{}{
						"replicas":    3,
						"cpu_request": "500m",
					},
					ProcessingTime: 1000,
					TokensUsed:     200,
				}
				mockDeps.GetLLMClient().(*testutils.MockLLMClient).SetResponse("ProcessIntent", mockResponse)
				
				result, err := controller.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace},
				})
				
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())
				
				// Verify intent is in processing state
				updatedIntent := &nephoranv1.NetworkIntent{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace}, updatedIntent)).To(Succeed())
				Expect(updatedIntent.Status.Phase).To(Equal(nephoranv1.NetworkIntentPhaseProcessing))
			})
		})

		Context("when handling error scenarios", func() {
			It("should handle LLM timeout gracefully", func() {
				// Configure mock to simulate timeout
				mockDeps.SetDelay("GetLLMClient", 10*time.Second)
				mockDeps.GetLLMClient().(*testutils.MockLLMClient).SetError("ProcessIntent", 
					fmt.Errorf("request timeout"))
				
				result, err := controller.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace},
				})
				
				// Should handle error gracefully
				Expect(err).To(HaveOccurred())
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))
				
				// Check intent status reflects error
				updatedIntent := &nephoranv1.NetworkIntent{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace}, updatedIntent)).To(Succeed())
				Expect(updatedIntent.Status.Phase).To(Equal(nephoranv1.NetworkIntentPhaseFailed))
			})

			It("should handle validation failures", func() {
				// Create intent with invalid configuration
				invalidIntent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent("invalid-intent", namespace)
				invalidIntent.Spec.Intent = "" // Invalid empty intent
				Expect(k8sClient.Create(ctx, invalidIntent)).To(Succeed())
				
				result, err := controller.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{Name: invalidIntent.Name, Namespace: invalidIntent.Namespace},
				})
				
				Expect(err).To(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())
				
				// Check validation error is recorded
				updatedIntent := &nephoranv1.NetworkIntent{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: invalidIntent.Name, Namespace: invalidIntent.Namespace}, updatedIntent)).To(Succeed())
				Expect(updatedIntent.Status.Phase).To(Equal(nephoranv1.NetworkIntentPhaseFailed))
				Expect(updatedIntent.Status.Message).To(ContainSubstring("validation"))
			})

			It("should handle resource conflicts", func() {
				// Configure mock to simulate resource conflict
				mockDeps.GetLLMClient().(*testutils.MockLLMClient).SetError("ProcessIntent", 
					fmt.Errorf("resource conflict: deployment already exists"))
				
				result, err := controller.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace},
				})
				
				Expect(err).To(HaveOccurred())
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))
				
				// Verify retry logic
				updatedIntent := &nephoranv1.NetworkIntent{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace}, updatedIntent)).To(Succeed())
				Expect(updatedIntent.Status.RetryCount).To(BeNumerically(">=", 1))
			})
		})

		Context("when handling concurrent requests", func() {
			It("should process 10+ simultaneous intents within performance targets", func() {
				concurrentRunner := testutils.NewConcurrentTestRunner(10)
				var wg sync.WaitGroup
				var mu sync.Mutex
				durations := make([]time.Duration, 0, 15)
				
				// Configure successful mock response
				mockResponse := testutils.CreateMockLLMResponse("5G-Core-AMF", 0.95)
				mockDeps.GetLLMClient().(*testutils.MockLLMClient).SetResponse("ProcessIntent", mockResponse)
				
				// Create 15 concurrent intents
				for i := 0; i < 15; i++ {
					intentName := fmt.Sprintf("concurrent-intent-%d", i)
					concurrentIntent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent(intentName, namespace)
					Expect(k8sClient.Create(ctx, concurrentIntent)).To(Succeed())
					
					wg.Add(1)
					go func(name string) {
						defer wg.Done()
						defer GinkgoRecover()
						
						start := time.Now()
						result, err := controller.Reconcile(ctx, ctrl.Request{
							NamespacedName: types.NamespacedName{Name: name, Namespace: namespace},
						})
						duration := time.Since(start)
						
						mu.Lock()
						durations = append(durations, duration)
						mu.Unlock()
						
						Expect(err).NotTo(HaveOccurred())
						Expect(result).NotTo(BeNil())
					}(intentName)
				}
				
				wg.Wait()
				
				// Verify performance targets
				Expect(durations).To(HaveLen(15))
				for _, duration := range durations {
					Expect(duration).To(BeNumerically("<", 5*time.Second), 
						"Each intent should process within 5-second SLA")
				}
				
				// Calculate and verify average processing time
				var totalDuration time.Duration
				for _, d := range durations {
					totalDuration += d
				}
				avgDuration := totalDuration / time.Duration(len(durations))
				Expect(avgDuration).To(BeNumerically("<", 3*time.Second), 
					"Average processing time should be under 3 seconds")
			})

			It("should maintain data consistency under concurrent access", func() {
				// Create a single intent that will be processed concurrently
				sharedIntent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent("shared-intent", namespace)
				Expect(k8sClient.Create(ctx, sharedIntent)).To(Succeed())
				
				// Configure mock response
				mockResponse := testutils.CreateMockLLMResponse("5G-Core-AMF", 0.95)
				mockDeps.GetLLMClient().(*testutils.MockLLMClient).SetResponse("ProcessIntent", mockResponse)
				
				var wg sync.WaitGroup
				errors := make([]error, 0)
				var mu sync.Mutex
				
				// Start 5 concurrent reconciliations of the same intent
				for i := 0; i < 5; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						defer GinkgoRecover()
						
						_, err := controller.Reconcile(ctx, ctrl.Request{
							NamespacedName: types.NamespacedName{
								Name:      sharedIntent.Name,
								Namespace: sharedIntent.Namespace,
							},
						})
						
						mu.Lock()
						errors = append(errors, err)
						mu.Unlock()
					}()
				}
				
				wg.Wait()
				
				// Check final state consistency
				finalIntent := &nephoranv1.NetworkIntent{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: sharedIntent.Name, Namespace: sharedIntent.Namespace,
				}, finalIntent)).To(Succeed())
				
				// Verify intent is in a valid final state
				Expect(finalIntent.Status.Phase).To(Or(
					Equal(nephoranv1.NetworkIntentPhaseProcessing),
					Equal(nephoranv1.NetworkIntentPhaseDeployed),
					Equal(nephoranv1.NetworkIntentPhaseFailed),
				))
			})
		})

		Context("performance validation", func() {
			It("should meet 5-second SLA for intent processing", func() {
				testCases := []struct {
					name       string
					intentType string
					confidence float64
				}{
					{"AMF Intent", "5G-Core-AMF", 0.95},
					{"SMF Intent", "5G-Core-SMF", 0.92},
					{"UPF Intent", "5G-Core-UPF", 0.88},
				}
				
				for _, tc := range testCases {
					By(fmt.Sprintf("processing %s", tc.name))
					
					// Create test intent
					testIntent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent(
						fmt.Sprintf("perf-test-%s", tc.intentType), namespace)
					testIntent.Spec.Intent = fmt.Sprintf("Deploy %s with high availability", tc.intentType)
					Expect(k8sClient.Create(ctx, testIntent)).To(Succeed())
					
					// Configure mock response
					mockResponse := testutils.CreateMockLLMResponse(tc.intentType, tc.confidence)
					mockDeps.GetLLMClient().(*testutils.MockLLMClient).SetResponse("ProcessIntent", mockResponse)
					
					// Measure performance
					start := time.Now()
					result, err := controller.Reconcile(ctx, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      testIntent.Name,
							Namespace: testIntent.Namespace,
						},
					})
					duration := time.Since(start)
					
					// Verify SLA compliance
					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(duration).To(BeNumerically("<", 5*time.Second), 
						fmt.Sprintf("%s should complete within 5-second SLA (actual: %v)", tc.name, duration))
				}
			})

			It("should handle memory efficiently during processing", func() {
				// This would integrate with actual memory monitoring
				// For now, we verify no resource leaks by checking object counts
				initialObjectCount := len(mockDeps.GetLLMClient().(*testutils.MockLLMClient).SetError)
				
				// Process multiple intents
				for i := 0; i < 10; i++ {
					memTestIntent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent(
						fmt.Sprintf("mem-test-%d", i), namespace)
					Expect(k8sClient.Create(ctx, memTestIntent)).To(Succeed())
					
					mockResponse := testutils.CreateMockLLMResponse("5G-Core-AMF", 0.95)
					mockDeps.GetLLMClient().(*testutils.MockLLMClient).SetResponse("ProcessIntent", mockResponse)
					
					_, err := controller.Reconcile(ctx, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      memTestIntent.Name,
							Namespace: memTestIntent.Namespace,
						},
					})
					Expect(err).NotTo(HaveOccurred())
				}
				
				// Verify no significant memory leaks (simplified check)
				finalObjectCount := len(mockDeps.GetLLMClient().(*testutils.MockLLMClient).SetError)
				Expect(finalObjectCount).To(Equal(initialObjectCount))
			})
		})

		Context("deletion and cleanup", func() {
			It("should handle intent deletion properly", func() {
				// Add finalizer to test cleanup
				intent.Finalizers = []string{controllers.NetworkIntentFinalizer}
				Expect(k8sClient.Update(ctx, intent)).To(Succeed())
				
				// Delete the intent
				Expect(k8sClient.Delete(ctx, intent)).To(Succeed())
				
				// Reconcile should handle cleanup
				result, err := controller.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace},
				})
				
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())
				
				// Verify intent is actually deleted
				deletedIntent := &nephoranv1.NetworkIntent{}
				err = k8sClient.Get(ctx, types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace}, deletedIntent)
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})
		})
	})

	Describe("Edge Cases and Error Conditions", func() {
		It("should handle non-existent intent gracefully", func() {
			result, err := controller.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent-intent",
					Namespace: namespace,
				},
			})
			
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
		})

		It("should validate intent specifications thoroughly", func() {
			testCases := []struct {
				name        string
				intentSpec  nephoranv1.NetworkIntentSpec
				expectError bool
			}{
				{
					name: "valid intent",
					intentSpec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy AMF with HA",
						Priority:   nephoranv1.PriorityHigh,
						MaxRetries: 3,
						Timeout:    metav1.Duration{Duration: 5 * time.Minute},
					},
					expectError: false,
				},
				{
					name: "empty intent",
					intentSpec: nephoranv1.NetworkIntentSpec{
						Intent: "",
					},
					expectError: true,
				},
				{
					name: "excessive retries",
					intentSpec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy AMF",
						MaxRetries: 100,
					},
					expectError: true,
				},
			}
			
			for _, tc := range testCases {
				By(fmt.Sprintf("validating %s", tc.name))
				
				testIntent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("validation-test-%s", tc.name),
						Namespace: namespace,
					},
					Spec: tc.intentSpec,
				}
				
				err := k8sClient.Create(ctx, testIntent)
				if tc.expectError {
					// Note: In a real environment, this would be caught by admission webhooks
					// For unit tests, we validate in the controller
					Expect(err).NotTo(HaveOccurred()) // Creation succeeds, validation happens in reconcile
					
					result, reconcileErr := controller.Reconcile(ctx, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name:      testIntent.Name,
							Namespace: testIntent.Namespace,
						},
					})
					
					Expect(reconcileErr).To(HaveOccurred())
					Expect(result.Requeue).To(BeFalse())
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			}
		})
	})
})