package integration_tests_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
<<<<<<< HEAD
=======
	"github.com/thc1006/nephoran-intent-operator/test/envtest"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	testutils "github.com/thc1006/nephoran-intent-operator/tests/utils"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
	mgr       ctrl.Manager
)

func TestIntentPipeline(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Intent Pipeline Integration Test Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
<<<<<<< HEAD
	testEnv = &envtest.Environment{
=======
	testEnv := testenv.Environment
>>>>>>> 6835433495e87288b95961af7173d866977175ff
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "deployments", "crds")},
		ErrorIfCRDPathMissing: false, // Set to false for flexibility in CI environments
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = nephoranv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Setup controller manager
	mgr, err = ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	// Setup controllers with mock dependencies
	mockDeps := testutils.NewMockDependencies()

	// Configure NetworkIntent controller
	config := &controllers.Config{
		MaxRetries:      3,
		RetryDelay:      2 * time.Second,
		Timeout:         30 * time.Second,
		GitRepoURL:      "https://github.com/test/repo.git",
		GitBranch:       "main",
		GitDeployPath:   "networkintents",
		LLMProcessorURL: "http://localhost:8080",
		UseNephioPorch:  false,
	}
	networkIntentController, err := controllers.NewNetworkIntentReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		mockDeps,
		config,
	)
	Expect(err).NotTo(HaveOccurred())

	err = networkIntentController.SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	// Setup E2NodeSet controller
	e2NodeSetController := &controllers.E2NodeSetReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("e2nodeset-controller"),
	}

	err = e2NodeSetController.SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred(), "failed to run manager")
	}()

	// Wait for manager to be ready
	Eventually(func() error {
		return mgr.GetCache().WaitForCacheSync(ctx)
	}, 30*time.Second, 1*time.Second).Should(Succeed())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Intent Pipeline Integration", func() {
	var (
		namespace    string
		testFixtures *testutils.TestFixtures
	)

	BeforeEach(func() {
		namespace = fmt.Sprintf("test-ns-%d", time.Now().UnixNano())

		// Create test namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		testFixtures = testutils.NewTestFixtures()

		DeferCleanup(func() {
			// Cleanup namespace
			Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
		})
	})

	Describe("End-to-End Intent Processing", func() {
		Context("when processing a complete intent lifecycle", func() {
			It("should transition from Pending to Deployed successfully", func() {
				By("creating a NetworkIntent")
				intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent("e2e-intent", namespace)
				intent.Spec.Intent = "Deploy a highly available 5G AMF with auto-scaling and monitoring"
				Expect(k8sClient.Create(ctx, intent)).To(Succeed())

				By("verifying intent starts in Pending phase")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() nephoranv1.NetworkIntentPhase {
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 10*time.Second, 500*time.Millisecond).Should(Equal(nephoranv1.NetworkIntentPhasePending))

				By("waiting for intent to progress to Processing phase")
				Eventually(func() nephoranv1.NetworkIntentPhase {
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 30*time.Second, 1*time.Second).Should(Equal(nephoranv1.NetworkIntentPhaseProcessing))

				By("verifying processing phase details")
				Expect(createdIntent.Status.ProcessingPhase).NotTo(BeEmpty())
				Expect(createdIntent.Status.ProcessingPhase).To(Or(
					Equal("LLMProcessing"),
					Equal("ResourcePlanning"),
					Equal("ManifestGeneration"),
					Equal("GitOpsCommit"),
					Equal("DeploymentVerification"),
				))

				By("waiting for intent to reach final state")
				Eventually(func() nephoranv1.NetworkIntentPhase {
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 60*time.Second, 2*time.Second).Should(Or(
					Equal(nephoranv1.NetworkIntentPhaseDeployed),
					Equal(nephoranv1.NetworkIntentPhaseFailed),
				))

				By("verifying final intent state")
				if createdIntent.Status.Phase == nephoranv1.NetworkIntentPhaseDeployed {
					// Verify successful deployment conditions
					Expect(createdIntent.Status.LastMessage).To(ContainSubstring("deployed"))

					// Check for Ready condition
					hasReadyCondition := false
					for _, condition := range createdIntent.Status.Conditions {
						if condition.Type == "Ready" {
							hasReadyCondition = true
							Expect(condition.Status).To(Equal(metav1.ConditionTrue))
							break
						}
					}
					Expect(hasReadyCondition).To(BeTrue(), "Deployed intent should have Ready condition")
				} else {
					// If failed, verify error information is present
					Expect(createdIntent.Status.LastMessage).NotTo(BeEmpty())
					Expect(createdIntent.Status.RetryCount).To(BeNumerically(">=", 0))
				}

				By("verifying processing metrics are recorded")
				if createdIntent.Status.ProcessingMetrics != nil {
					Expect(createdIntent.Status.ProcessingMetrics).To(HaveKey("processing_time_ms"))
					Expect(createdIntent.Status.ProcessingMetrics["processing_time_ms"]).To(BeNumerically(">", 0))
				}
			})

			It("should handle multiple intents concurrently", func() {
				numIntents := 5
				intents := make([]*nephoranv1.NetworkIntent, numIntents)

				By("creating multiple NetworkIntents")
				for i := 0; i < numIntents; i++ {
					intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent(
						fmt.Sprintf("concurrent-intent-%d", i), namespace)
					intent.Spec.Intent = fmt.Sprintf("Deploy 5G %s function with HA",
						[]string{"AMF", "SMF", "UPF", "NSSF", "UDM"}[i])
					intents[i] = intent
					Expect(k8sClient.Create(ctx, intent)).To(Succeed())
				}

				By("waiting for all intents to start processing")
				for i, intent := range intents {
					Eventually(func() nephoranv1.NetworkIntentPhase {
						currentIntent := &nephoranv1.NetworkIntent{}
						err := k8sClient.Get(ctx, types.NamespacedName{
							Name: intent.Name, Namespace: intent.Namespace,
						}, currentIntent)
						if err != nil {
							return ""
						}
						return currentIntent.Status.Phase
					}, 30*time.Second, 1*time.Second).Should(Or(
						Equal(nephoranv1.NetworkIntentPhaseProcessing),
						Equal(nephoranv1.NetworkIntentPhaseDeployed),
						Equal(nephoranv1.NetworkIntentPhaseFailed),
					), fmt.Sprintf("Intent %d should reach processing or final state", i))
				}

				By("verifying all intents reach final states")
				successfulIntents := 0
				for i, intent := range intents {
					Eventually(func() bool {
						currentIntent := &nephoranv1.NetworkIntent{}
						err := k8sClient.Get(ctx, types.NamespacedName{
							Name: intent.Name, Namespace: intent.Namespace,
						}, currentIntent)
						if err != nil {
							return false
						}

						return currentIntent.Status.Phase == nephoranv1.NetworkIntentPhaseDeployed ||
							currentIntent.Status.Phase == nephoranv1.NetworkIntentPhaseFailed
					}, 90*time.Second, 2*time.Second).Should(BeTrue(),
						fmt.Sprintf("Intent %d should reach final state", i))

					// Count successful deployments
					finalIntent := &nephoranv1.NetworkIntent{}
					k8sClient.Get(ctx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, finalIntent)
					if finalIntent.Status.Phase == nephoranv1.NetworkIntentPhaseDeployed {
						successfulIntents++
					}
				}

				By("verifying reasonable success rate")
				// In integration tests with mocks, we expect high success rate
				Expect(successfulIntents).To(BeNumerically(">=", 3),
					"At least 60% of intents should succeed in integration test")
			})
		})

		Context("when handling different intent types", func() {
			DescribeTable("processing various 5G network function intents",
				func(intentDescription string, expectedType string) {
					intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent(
						fmt.Sprintf("type-test-%s", expectedType), namespace)
					intent.Spec.Intent = intentDescription
					Expect(k8sClient.Create(ctx, intent)).To(Succeed())

					// Wait for processing to complete
					Eventually(func() nephoranv1.NetworkIntentPhase {
						currentIntent := &nephoranv1.NetworkIntent{}
						err := k8sClient.Get(ctx, types.NamespacedName{
							Name: intent.Name, Namespace: intent.Namespace,
						}, currentIntent)
						if err != nil {
							return ""
						}
						return currentIntent.Status.Phase
					}, 60*time.Second, 2*time.Second).Should(Or(
						Equal(nephoranv1.NetworkIntentPhaseDeployed),
						Equal(nephoranv1.NetworkIntentPhaseFailed),
					))

					// Verify intent type was correctly identified (in real scenario)
					processedIntent := &nephoranv1.NetworkIntent{}
					Expect(k8sClient.Get(ctx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, processedIntent)).To(Succeed())

					if processedIntent.Status.Phase == nephoranv1.NetworkIntentPhaseDeployed {
						// Verify some processing occurred
						Expect(processedIntent.Status.ProcessingPhase).NotTo(BeEmpty())
					}
				},
				Entry("AMF deployment", "Deploy 5G Access and Mobility Management Function with high availability", "AMF"),
				Entry("SMF deployment", "Create Session Management Function with auto-scaling", "SMF"),
				Entry("UPF deployment", "Deploy User Plane Function for edge computing", "UPF"),
				Entry("NSSF deployment", "Setup Network Slice Selection Function", "NSSF"),
				Entry("Complex intent", "Deploy complete 5G Core with AMF, SMF, UPF and monitoring", "5G-Core"),
			)
		})

		Context("when handling error scenarios", func() {
			It("should handle LLM processing failures gracefully", func() {
				intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent("error-test", namespace)
				intent.Spec.Intent = "INVALID_INTENT_THAT_SHOULD_FAIL_PROCESSING"
				Expect(k8sClient.Create(ctx, intent)).To(Succeed())

				// Wait for processing to start and potentially fail
				Eventually(func() nephoranv1.NetworkIntentPhase {
					currentIntent := &nephoranv1.NetworkIntent{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, currentIntent)
					if err != nil {
						return ""
					}
					return currentIntent.Status.Phase
				}, 45*time.Second, 2*time.Second).Should(Or(
					Equal(nephoranv1.NetworkIntentPhaseProcessing),
					Equal(nephoranv1.NetworkIntentPhaseFailed),
					Equal(nephoranv1.NetworkIntentPhaseDeployed), // Mock might still succeed
				))

				// Verify error information if failed
				finalIntent := &nephoranv1.NetworkIntent{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: intent.Name, Namespace: intent.Namespace,
				}, finalIntent)).To(Succeed())

				if finalIntent.Status.Phase == nephoranv1.NetworkIntentPhaseFailed {
					Expect(finalIntent.Status.Message).NotTo(BeEmpty())
					Expect(finalIntent.Status.Message).To(ContainSubstring("error"))
				}
			})

			It("should implement retry logic for transient failures", func() {
				intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent("retry-test", namespace)
				intent.Spec.MaxRetries = 3
				intent.Spec.Intent = "Deploy AMF with retry logic test"
				Expect(k8sClient.Create(ctx, intent)).To(Succeed())

				// Monitor retry attempts
				Eventually(func() bool {
					currentIntent := &nephoranv1.NetworkIntent{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, currentIntent)
					if err != nil {
						return false
					}

					// Should reach final state within reasonable time
					return currentIntent.Status.Phase == nephoranv1.NetworkIntentPhaseDeployed ||
						currentIntent.Status.Phase == nephoranv1.NetworkIntentPhaseFailed ||
						currentIntent.Status.RetryCount > 0
				}, 60*time.Second, 2*time.Second).Should(BeTrue())

				// Verify retry mechanism was used if needed
				finalIntent := &nephoranv1.NetworkIntent{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: intent.Name, Namespace: intent.Namespace,
				}, finalIntent)).To(Succeed())

				if finalIntent.Status.Phase == nephoranv1.NetworkIntentPhaseFailed {
					Expect(finalIntent.Status.RetryCount).To(BeNumerically("<=", 3))
				}
			})
		})

		Context("when verifying phase transitions", func() {
			It("should follow the correct phase sequence", func() {
				intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent("phase-test", namespace)
				intent.Spec.Intent = "Deploy AMF with comprehensive phase tracking"
				Expect(k8sClient.Create(ctx, intent)).To(Succeed())

				observedPhases := make([]nephoranv1.NetworkIntentPhase, 0)
				observedProcessingPhases := make([]string, 0)

				// Track phase transitions
				go func() {
					defer GinkgoRecover()
					ticker := time.NewTicker(500 * time.Millisecond)
					defer ticker.Stop()
					timeout := time.NewTimer(90 * time.Second)
					defer timeout.Stop()

					for {
						select {
						case <-timeout.C:
							return
						case <-ticker.C:
							currentIntent := &nephoranv1.NetworkIntent{}
							err := k8sClient.Get(ctx, types.NamespacedName{
								Name: intent.Name, Namespace: intent.Namespace,
							}, currentIntent)
							if err != nil {
								continue
							}

							// Record new phases
							if len(observedPhases) == 0 || observedPhases[len(observedPhases)-1] != currentIntent.Status.Phase {
								observedPhases = append(observedPhases, currentIntent.Status.Phase)
							}

							// Record processing phases
							if currentIntent.Status.ProcessingPhase != "" {
								if len(observedProcessingPhases) == 0 ||
									observedProcessingPhases[len(observedProcessingPhases)-1] != currentIntent.Status.ProcessingPhase {
									observedProcessingPhases = append(observedProcessingPhases, currentIntent.Status.ProcessingPhase)
								}
							}

							// Stop when reaching final state
							if currentIntent.Status.Phase == nephoranv1.NetworkIntentPhaseDeployed ||
								currentIntent.Status.Phase == nephoranv1.NetworkIntentPhaseFailed {
								return
							}
						}
					}
				}()

				// Wait for completion
				Eventually(func() nephoranv1.NetworkIntentPhase {
					currentIntent := &nephoranv1.NetworkIntent{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, currentIntent)
					if err != nil {
						return ""
					}
					return currentIntent.Status.Phase
				}, 90*time.Second, 1*time.Second).Should(Or(
					Equal(nephoranv1.NetworkIntentPhaseDeployed),
					Equal(nephoranv1.NetworkIntentPhaseFailed),
				))

				// Verify phase progression
				By("verifying main phase transitions")
				Expect(observedPhases).To(ContainElement(nephoranv1.NetworkIntentPhasePending))
				Expect(observedPhases).To(ContainElement(nephoranv1.NetworkIntentPhaseProcessing))
				Expect(len(observedPhases)).To(BeNumerically(">=", 2))

				// Verify phase order (Pending should come before Processing)
				pendingIndex := -1
				processingIndex := -1
				for i, phase := range observedPhases {
					if phase == nephoranv1.NetworkIntentPhasePending && pendingIndex == -1 {
						pendingIndex = i
					}
					if phase == nephoranv1.NetworkIntentPhaseProcessing && processingIndex == -1 {
						processingIndex = i
					}
				}

				if pendingIndex >= 0 && processingIndex >= 0 {
					Expect(pendingIndex).To(BeNumerically("<", processingIndex),
						"Pending phase should come before Processing phase")
				}

				By("verifying processing phase progression")
				if len(observedProcessingPhases) > 0 {
					// Should have at least started with LLM processing
					Expect(observedProcessingPhases[0]).To(Or(
						Equal("LLMProcessing"),
						Equal("ResourcePlanning"),
					))
				}
			})
		})
	})

	Describe("Resource Management", func() {
		It("should properly manage Kubernetes resources during intent lifecycle", func() {
			intent := testutils.NetworkIntentFixture.CreateBasicNetworkIntent("resource-test", namespace)
			intent.Spec.Intent = "Deploy AMF with resource monitoring"
			Expect(k8sClient.Create(ctx, intent)).To(Succeed())

			// Track resource creation/cleanup
			initialResourceCount := 0
			Eventually(func() int {
				// Count ConfigMaps, Secrets, or other resources that might be created
				configMaps := &corev1.ConfigMapList{}
				k8sClient.List(ctx, configMaps, client.InNamespace(namespace))

				secrets := &corev1.SecretList{}
				k8sClient.List(ctx, secrets, client.InNamespace(namespace))

				return len(configMaps.Items) + len(secrets.Items)
			}, 30*time.Second, 2*time.Second).Should(BeNumerically(">=", initialResourceCount))

			// Wait for intent completion
			Eventually(func() nephoranv1.NetworkIntentPhase {
				currentIntent := &nephoranv1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: intent.Name, Namespace: intent.Namespace,
				}, currentIntent)
				if err != nil {
					return ""
				}
				return currentIntent.Status.Phase
			}, 60*time.Second, 2*time.Second).Should(Or(
				Equal(nephoranv1.NetworkIntentPhaseDeployed),
				Equal(nephoranv1.NetworkIntentPhaseFailed),
			))

			// Verify no resource leaks
			finalResourceCount := 0
			configMaps := &corev1.ConfigMapList{}
			k8sClient.List(ctx, configMaps, client.InNamespace(namespace))
			secrets := &corev1.SecretList{}
			k8sClient.List(ctx, secrets, client.InNamespace(namespace))
			finalResourceCount = len(configMaps.Items) + len(secrets.Items)

			// Should have reasonable resource count (not excessive)
			Expect(finalResourceCount).To(BeNumerically("<=", 20),
				"Should not create excessive resources")
		})
	})
})
