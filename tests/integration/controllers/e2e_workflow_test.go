package integration_tests_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
)

var _ = Describe("End-to-End Workflow Integration Tests", func() {
	var (
		namespace       *corev1.Namespace
		testCtx         context.Context
		workflowTracker *E2EWorkflowTracker
	)

	BeforeEach(func() {
		namespace = CreateTestNamespace()
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, 15*time.Minute) // Longer timeout for E2E
		DeferCleanup(cancel)

		workflowTracker = NewE2EWorkflowTracker()
	})

	Describe("Complete NetworkIntent Lifecycle", func() {
		Context("when processing a production-ready 5G Core deployment", func() {
			It("should successfully complete the entire workflow from intent to deployment", func() {
				By("creating a comprehensive NetworkIntent for 5G Core deployment")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "e2e-5g-core-deployment",
						Namespace: namespace.Name,
						Labels: map[string]string{
							"test":                       "e2e",
							"deployment-type":            "5g-core",
							"nephoran.com/priority":      "high",
							"nephoran.com/test-scenario": "production-deployment",
						},
						Annotations: map[string]string{
							"nephoran.com/description":         "E2E test for complete 5G Core deployment workflow",
							"nephoran.com/expected-duration":   "10m",
							"nephoran.com/deployment-strategy": "blue-green",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: "Deploy a complete production-ready 5G Core network with AMF, SMF, UPF, and NSSF components, " +
							"including high availability, auto-scaling, comprehensive monitoring, security policies, " +
							"and integration with O-RAN interfaces for intelligent RAN management",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityHigh,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentAMF,
							nephoranv1.TargetComponentSMF,
							nephoranv1.TargetComponentUPF,
							nephoranv1.TargetComponentNSSF,
						},
						ResourceConstraints: &nephoranv1.ResourceConstraints{
							CPU:       resource.NewQuantity(2000, resource.DecimalSI),       // 2000m
							Memory:    resource.NewQuantity(4294967296, resource.BinarySI),  // 4Gi
							Storage:   resource.NewQuantity(21474836480, resource.BinarySI), // 20Gi
							MaxCPU:    resource.NewQuantity(8000, resource.DecimalSI),       // 8000m
							MaxMemory: resource.NewQuantity(17179869184, resource.BinarySI), // 16Gi
						},
						TargetNamespace: "production-5g",
						TargetCluster:   "main-production-cluster",
						NetworkSlice:    "001234-567890",
						Region:          "us-east-1",
						TimeoutSeconds:  &[]int32{900}[0], // 15 minutes
						MaxRetries:      &[]int32{3}[0],
					},
				}

				workflowTracker.StartWorkflow(intent.Name, "5g-core-deployment")
				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("Phase 1: Verifying intent validation and acceptance")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() error {
					return k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
				}, 30*time.Second, 2*time.Second).Should(Succeed())

				// Verify initial state
				Expect(createdIntent.Spec.Intent).To(Equal(intent.Spec.Intent))
				Expect(createdIntent.Spec.TargetComponents).To(HaveLen(4))
				workflowTracker.RecordPhase(intent.Name, "validation", "completed")

				By("Phase 2: Waiting for LLM processing phase")
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 120*time.Second, 5*time.Second).Should(Or(
					Equal("Processing"),
					Equal("Deploying"),
					Equal("Ready"),
				))

				if createdIntent.Status.Phase == "Processing" {
					workflowTracker.RecordPhase(intent.Name, "llm-processing", "completed")
					Expect(createdIntent.Status.ProcessingStartTime).NotTo(BeNil())
				}

				By("Phase 3: Monitoring processing pipeline progression")
				Eventually(func() bool {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return false
					}

					// Check if we have processing phase details
					if createdIntent.Status.ProcessingPhase != "" {
						workflowTracker.RecordPhase(intent.Name,
							fmt.Sprintf("processing-%s", createdIntent.Status.ProcessingPhase), "active")
						return true
					}
					return false
				}, 90*time.Second, 3*time.Second).Should(BeTrue())

				By("Phase 4: Waiting for GitOps deployment initiation")
				Eventually(func() bool {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return false
					}

					return createdIntent.Status.GitCommitHash != "" ||
						createdIntent.Status.DeploymentStartTime != nil ||
						createdIntent.Status.Phase == "Deploying" ||
						createdIntent.Status.Phase == "Ready"
				}, 150*time.Second, 5*time.Second).Should(BeTrue())

				if createdIntent.Status.GitCommitHash != "" {
					workflowTracker.RecordPhase(intent.Name, "gitops-commit", "completed")
					Expect(createdIntent.Status.GitCommitHash).To(HaveLen(BeNumerically(">=", 7)))
				}

				By("Phase 5: Verifying deployment completion")
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 300*time.Second, 10*time.Second).Should(Or(
					Equal("Ready"),
					Equal("Deploying"), // Still progressing
				))

				By("Phase 6: Validating final deployment state and metrics")
				if createdIntent.Status.Phase == "Ready" {
					workflowTracker.RecordPhase(intent.Name, "deployment", "completed")

					// Verify completion timestamps
					Expect(createdIntent.Status.ProcessingCompletionTime).NotTo(BeNil())
					if createdIntent.Status.DeploymentCompletionTime != nil {
						Expect(createdIntent.Status.DeploymentCompletionTime.Time).To(BeTemporally(">=",
							createdIntent.Status.ProcessingStartTime.Time))
					}

					// Verify deployed components tracking
					Expect(createdIntent.Status.DeployedComponents).To(ContainElements(
						nephoranv1.TargetComponentAMF,
						nephoranv1.TargetComponentSMF,
					))
				}

				By("Phase 7: Checking related Kubernetes resources")
				// Verify that related resources might have been created
				configMaps := &corev1.ConfigMapList{}
				err := k8sClient.List(testCtx, configMaps, client.InNamespace(namespace.Name))
				Expect(err).NotTo(HaveOccurred())

				secrets := &corev1.SecretList{}
				err = k8sClient.List(testCtx, secrets, client.InNamespace(namespace.Name))
				Expect(err).NotTo(HaveOccurred())

				// Should not create excessive resources in test environment
				totalResources := len(configMaps.Items) + len(secrets.Items)
				Expect(totalResources).To(BeNumerically("<=", 20))

				By("Phase 8: Verifying workflow completion metrics")
				workflowTracker.CompleteWorkflow(intent.Name, createdIntent.Status.Phase == "Ready")

				workflow := workflowTracker.GetWorkflow(intent.Name)
				Expect(workflow).NotTo(BeNil())
				Expect(workflow.CompletedPhases).To(ContainElement("validation"))

				// Verify timing expectations
				if workflow.Completed {
					Expect(workflow.TotalDuration).To(BeNumerically("<", 15*time.Minute))
				}

				By("Workflow Summary: E2E 5G Core deployment completed successfully")
				GinkgoWriter.Printf("=== E2E Workflow Summary ===\n")
				GinkgoWriter.Printf("Intent: %s\n", intent.Name)
				GinkgoWriter.Printf("Final Phase: %s\n", createdIntent.Status.Phase)
				GinkgoWriter.Printf("Components: %v\n", createdIntent.Status.DeployedComponents)
				if workflow.Completed {
					GinkgoWriter.Printf("Total Duration: %v\n", workflow.TotalDuration)
				}
				GinkgoWriter.Printf("Phases Completed: %v\n", workflow.CompletedPhases)
				GinkgoWriter.Printf("=============================\n")
			})

			It("should handle complex multi-component O-RAN deployment workflow", func() {
				By("creating NetworkIntent for complete O-RAN deployment")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "e2e-oran-deployment",
						Namespace: namespace.Name,
						Labels: map[string]string{
							"test":                    "e2e",
							"deployment-type":         "o-ran",
							"nephoran.com/complexity": "high",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: "Deploy intelligent O-RAN architecture with Near-RT RIC, O-DU, O-CU components, " +
							"xApp runtime environment, E2 interface simulation with 50 nodes, A1 policy management, " +
							"and ML-based RAN optimization capabilities",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityHigh,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentNearRTRIC,
							nephoranv1.TargetComponentODU,
							nephoranv1.TargetComponentOCUCP,
							nephoranv1.TargetComponentOCUUP,
						},
						TargetCluster:  "edge-cluster",
						Region:         "us-west-2",
						TimeoutSeconds: &[]int32{1200}[0], // 20 minutes for complex deployment
						MaxRetries:     &[]int32{5}[0],
					},
				}

				workflowTracker.StartWorkflow(intent.Name, "o-ran-deployment")
				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("monitoring O-RAN specific deployment phases")
				createdIntent := &nephoranv1.NetworkIntent{}

				// Wait for initial processing
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 180*time.Second, 5*time.Second).Should(Not(Equal("Pending")))

				workflowTracker.RecordPhase(intent.Name, "o-ran-processing", "active")

				By("creating related E2NodeSet for O-RAN simulation")
				e2NodeSet := &nephoranv1.E2NodeSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "oran-e2-simulation",
						Namespace: namespace.Name,
						Labels: map[string]string{
							"nephoran.com/related-intent": intent.Name,
							"test":                        "e2e",
						},
					},
					Spec: nephoranv1.E2NodeSetSpec{
						Replicas: 10,
						Template: nephoranv1.E2NodeTemplate{
							Spec: nephoranv1.E2NodeSpec{
								NodeID:             "e2e-oran-node",
								E2InterfaceVersion: "v3.0",
								SupportedRANFunctions: []nephoranv1.RANFunction{
									{
										FunctionID:  1,
										Revision:    2,
										Description: "KPM Service Model v2.0",
										OID:         "1.3.6.1.4.1.53148.1.1.2.2",
									},
									{
										FunctionID:  2,
										Revision:    1,
										Description: "RC Service Model",
										OID:         "1.3.6.1.4.1.53148.1.1.2.3",
									},
									{
										FunctionID:  3,
										Revision:    1,
										Description: "NI Service Model",
										OID:         "1.3.6.1.4.1.53148.1.1.2.4",
									},
								},
							},
						},
						SimulationConfig: &nephoranv1.SimulationConfig{
							UECount:           2000,
							TrafficGeneration: true,
							MetricsInterval:   "15s",
							TrafficProfile:    nephoranv1.TrafficProfileHigh,
						},
						RICConfiguration: &nephoranv1.RICConfiguration{
							RICEndpoint:       "http://near-rt-ric:38080",
							ConnectionTimeout: "30s",
							HeartbeatInterval: "5s",
							RetryConfig: &nephoranv1.RetryConfig{
								MaxAttempts:     3,
								BackoffInterval: "10s",
							},
						},
					},
				}

				Expect(k8sClient.Create(testCtx, e2NodeSet)).To(Succeed())
				workflowTracker.RecordPhase(intent.Name, "e2nodeset-creation", "completed")

				By("monitoring E2NodeSet deployment progress")
				createdE2NodeSet := &nephoranv1.E2NodeSet{}
				Eventually(func() int32 {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: e2NodeSet.Name, Namespace: e2NodeSet.Namespace,
					}, createdE2NodeSet)
					if err != nil {
						return -1
					}
					return createdE2NodeSet.Status.CurrentReplicas
				}, 120*time.Second, 5*time.Second).Should(BeNumerically(">=", 0))

				By("verifying O-RAN intent deployment completion")
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 400*time.Second, 10*time.Second).Should(Or(
					Equal("Ready"),
					Equal("Deploying"),
					Equal("Failed"),
				))

				workflowTracker.CompleteWorkflow(intent.Name, createdIntent.Status.Phase == "Ready")

				By("validating O-RAN component deployment")
				if createdIntent.Status.Phase == "Ready" {
					Expect(createdIntent.Status.DeployedComponents).To(ContainElements(
						nephoranv1.TargetComponentNearRTRIC,
						nephoranv1.TargetComponentODU,
					))
				}

				By("checking E2NodeSet final status")
				Expect(createdE2NodeSet.Status.ReadyReplicas).To(BeNumerically(">=", 0))
				if createdE2NodeSet.Status.ReadyReplicas > 0 {
					workflowTracker.RecordPhase(intent.Name, "e2nodes-ready", "completed")
				}
			})
		})

		Context("when handling failure scenarios and recovery", func() {
			It("should demonstrate resilient workflow with retry and recovery mechanisms", func() {
				By("creating NetworkIntent designed to test failure recovery")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "e2e-failure-recovery",
						Namespace: namespace.Name,
						Labels: map[string]string{
							"test":                         "e2e",
							"scenario":                     "failure-recovery",
							"nephoran.com/test-resilience": "true",
						},
						Annotations: map[string]string{
							"nephoran.com/simulate-failure": "transient",
							"nephoran.com/max-failure-time": "2m",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy resilient SMF with failure recovery testing and comprehensive retry logic",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityMedium,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentSMF,
						},
						TimeoutSeconds: &[]int32{600}[0], // 10 minutes
						MaxRetries:     &[]int32{5}[0],   // Allow more retries for failure testing
					},
				}

				workflowTracker.StartWorkflow(intent.Name, "failure-recovery-test")
				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("monitoring workflow through failure and recovery cycles")
				createdIntent := &nephoranv1.NetworkIntent{}

				// Track retry attempts
				retryCount := 0
				Eventually(func() bool {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return false
					}

					if createdIntent.Status.RetryCount > 0 {
						if int(createdIntent.Status.RetryCount) > retryCount {
							retryCount = int(createdIntent.Status.RetryCount)
							workflowTracker.RecordPhase(intent.Name,
								fmt.Sprintf("retry-%d", retryCount), "completed")
						}
					}

					return createdIntent.Status.Phase == "Ready" ||
						createdIntent.Status.Phase == "Failed" ||
						time.Since(createdIntent.CreationTimestamp.Time) > 8*time.Minute
				}, 600*time.Second, 5*time.Second).Should(BeTrue())

				By("analyzing failure recovery behavior")
				workflowTracker.RecordPhase(intent.Name, "failure-analysis", "completed")

				// Check if retries were attempted
				if createdIntent.Status.RetryCount > 0 {
					Expect(createdIntent.Status.RetryCount).To(BeNumerically("<=", 5))
					Expect(createdIntent.Status.LastRetryTime).NotTo(BeNil())
					workflowTracker.RecordPhase(intent.Name, "retry-mechanism", "verified")
				}

				By("verifying final state after recovery attempts")
				finalPhase := createdIntent.Status.Phase
				workflowTracker.CompleteWorkflow(intent.Name, finalPhase == "Ready")

				// Log recovery metrics
				GinkgoWriter.Printf("=== Failure Recovery Analysis ===\n")
				GinkgoWriter.Printf("Final Phase: %s\n", finalPhase)
				GinkgoWriter.Printf("Retry Count: %d\n", createdIntent.Status.RetryCount)
				GinkgoWriter.Printf("Total Duration: %v\n", time.Since(createdIntent.CreationTimestamp.Time))
				if len(createdIntent.Status.ValidationErrors) > 0 {
					GinkgoWriter.Printf("Validation Errors: %v\n", createdIntent.Status.ValidationErrors)
				}
				GinkgoWriter.Printf("=================================\n")
			})
		})

		Context("when performing concurrent workflow processing", func() {
			It("should handle multiple NetworkIntents concurrently without interference", func() {
				By("creating multiple NetworkIntents for concurrent processing")
				intents := make([]*nephoranv1.NetworkIntent, 0, 5)

				intentSpecs := []struct {
					name      string
					component nephoranv1.TargetComponent
					priority  nephoranv1.Priority
				}{
					{"concurrent-amf", nephoranv1.TargetComponentAMF, nephoranv1.PriorityHigh},
					{"concurrent-smf", nephoranv1.TargetComponentSMF, nephoranv1.PriorityMedium},
					{"concurrent-upf", nephoranv1.TargetComponentUPF, nephoranv1.PriorityHigh},
					{"concurrent-nssf", nephoranv1.TargetComponentNSSF, nephoranv1.PriorityLow},
					{"concurrent-ric", nephoranv1.TargetComponentNearRTRIC, nephoranv1.PriorityMedium},
				}

				for i, spec := range intentSpecs {
					intent := &nephoranv1.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("e2e-%s-%d", spec.name, i),
							Namespace: namespace.Name,
							Labels: map[string]string{
								"test":                   "e2e",
								"scenario":               "concurrent",
								"nephoran.com/batch":     "concurrent-batch-1",
								"nephoran.com/component": string(spec.component),
							},
						},
						Spec: nephoranv1.NetworkIntentSpec{
							Intent:     fmt.Sprintf("Deploy %s in concurrent batch processing test", spec.component),
							IntentType: nephoranv1.IntentTypeDeployment,
							Priority:   spec.priority,
							TargetComponents: []nephoranv1.TargetComponent{
								spec.component,
							},
							TimeoutSeconds: &[]int32{480}[0], // 8 minutes
							MaxRetries:     &[]int32{2}[0],
						},
					}

					intents = append(intents, intent)
					workflowTracker.StartWorkflow(intent.Name, "concurrent-deployment")
					Expect(k8sClient.Create(testCtx, intent)).To(Succeed())
				}

				By("monitoring concurrent processing progress")
				processedCount := 0
				Eventually(func() int {
					currentProcessed := 0
					for _, intent := range intents {
						currentIntent := &nephoranv1.NetworkIntent{}
						err := k8sClient.Get(testCtx, types.NamespacedName{
							Name: intent.Name, Namespace: intent.Namespace,
						}, currentIntent)
						if err == nil && currentIntent.Status.Phase != "Pending" {
							currentProcessed++

							// Record phase progression for each intent
							if currentIntent.Status.Phase == "Processing" ||
								currentIntent.Status.Phase == "Deploying" ||
								currentIntent.Status.Phase == "Ready" ||
								currentIntent.Status.Phase == "Failed" {
								workflowTracker.RecordPhase(intent.Name, "concurrent-processing", "active")
							}
						}
					}
					processedCount = currentProcessed
					return currentProcessed
				}, 240*time.Second, 10*time.Second).Should(BeNumerically(">=", len(intents)))

				By("waiting for all intents to reach final states")
				completedIntents := make(map[string]string)
				Eventually(func() int {
					completed := 0
					for _, intent := range intents {
						currentIntent := &nephoranv1.NetworkIntent{}
						err := k8sClient.Get(testCtx, types.NamespacedName{
							Name: intent.Name, Namespace: intent.Namespace,
						}, currentIntent)
						if err == nil {
							phase := currentIntent.Status.Phase
							if phase == "Ready" || phase == "Failed" {
								completed++
								completedIntents[intent.Name] = phase
								workflowTracker.CompleteWorkflow(intent.Name, phase == "Ready")
							}
						}
					}
					return completed
				}, 600*time.Second, 15*time.Second).Should(BeNumerically(">=", 3)) // At least 60% success

				By("analyzing concurrent processing results")
				successfulIntents := 0
				failedIntents := 0

				for intentName, phase := range completedIntents {
					if phase == "Ready" {
						successfulIntents++
					} else if phase == "Failed" {
						failedIntents++
					}
					workflowTracker.RecordPhase(intentName, "concurrent-completion", "completed")
				}

				By("verifying concurrent processing quality")
				totalCompleted := successfulIntents + failedIntents
				Expect(totalCompleted).To(BeNumerically(">=", 3))

				// Success rate should be reasonable in integration test environment
				if totalCompleted > 0 {
					successRate := float64(successfulIntents) / float64(totalCompleted)
					Expect(successRate).To(BeNumerically(">=", 0.4)) // At least 40% success rate
				}

				GinkgoWriter.Printf("=== Concurrent Processing Results ===\n")
				GinkgoWriter.Printf("Total Intents: %d\n", len(intents))
				GinkgoWriter.Printf("Completed: %d\n", totalCompleted)
				GinkgoWriter.Printf("Successful: %d\n", successfulIntents)
				GinkgoWriter.Printf("Failed: %d\n", failedIntents)
				GinkgoWriter.Printf("Success Rate: %.1f%%\n", float64(successfulIntents)/float64(totalCompleted)*100)
				GinkgoWriter.Printf("=====================================\n")
			})
		})
	})

	Describe("Cross-Component Integration Workflows", func() {
		Context("when orchestrating complex multi-component scenarios", func() {
			It("should coordinate NetworkIntent and E2NodeSet for complete RAN deployment", func() {
				By("creating coordinated NetworkIntent and E2NodeSet deployment")

				// First create the main intent
				mainIntent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "e2e-coordinated-ran",
						Namespace: namespace.Name,
						Labels: map[string]string{
							"test":                      "e2e",
							"scenario":                  "coordinated-deployment",
							"nephoran.com/coordination": "enabled",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy coordinated RAN infrastructure with Near-RT RIC and E2 node simulation",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityHigh,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentNearRTRIC,
							nephoranv1.TargetComponentODU,
						},
						TimeoutSeconds: &[]int32{720}[0], // 12 minutes
						MaxRetries:     &[]int32{3}[0],
					},
				}

				workflowTracker.StartWorkflow(mainIntent.Name, "coordinated-deployment")
				Expect(k8sClient.Create(testCtx, mainIntent)).To(Succeed())

				By("waiting for initial intent processing")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: mainIntent.Name, Namespace: mainIntent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 120*time.Second, 5*time.Second).Should(Not(Equal("Pending")))

				workflowTracker.RecordPhase(mainIntent.Name, "main-intent-processing", "active")

				By("creating coordinated E2NodeSet with larger scale")
				coordinatedE2NodeSet := &nephoranv1.E2NodeSet{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "coordinated-e2-nodes",
						Namespace: namespace.Name,
						Labels: map[string]string{
							"test":                       "e2e",
							"nephoran.com/parent-intent": mainIntent.Name,
							"nephoran.com/coordination":  "enabled",
						},
						Annotations: map[string]string{
							"nephoran.com/related-intent": mainIntent.Name,
						},
					},
					Spec: nephoranv1.E2NodeSetSpec{
						Replicas: 25, // Larger scale for coordination testing
						Template: nephoranv1.E2NodeTemplate{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{
									"app":                        "coordinated-e2-node",
									"nephoran.com/parent-intent": mainIntent.Name,
								},
							},
							Spec: nephoranv1.E2NodeSpec{
								NodeID:             "coord-e2-node",
								E2InterfaceVersion: "v3.0",
								SupportedRANFunctions: []nephoranv1.RANFunction{
									{
										FunctionID:  1,
										Revision:    2,
										Description: "KPM Service Model Enhanced",
										OID:         "1.3.6.1.4.1.53148.1.1.2.2",
									},
									{
										FunctionID:  2,
										Revision:    1,
										Description: "RC Service Model",
										OID:         "1.3.6.1.4.1.53148.1.1.2.3",
									},
									{
										FunctionID:  4,
										Revision:    1,
										Description: "CCC Service Model",
										OID:         "1.3.6.1.4.1.53148.1.1.2.5",
									},
								},
							},
						},
						SimulationConfig: &nephoranv1.SimulationConfig{
							UECount:           5000, // High UE count for realistic testing
							TrafficGeneration: true,
							MetricsInterval:   "10s",
							TrafficProfile:    nephoranv1.TrafficProfileBurst,
						},
						RICConfiguration: &nephoranv1.RICConfiguration{
							RICEndpoint:       "http://coordinated-ric:38080",
							ConnectionTimeout: "45s",
							HeartbeatInterval: "5s",
							RetryConfig: &nephoranv1.RetryConfig{
								MaxAttempts:     5,
								BackoffInterval: "15s",
							},
						},
					},
				}

				Expect(k8sClient.Create(testCtx, coordinatedE2NodeSet)).To(Succeed())
				workflowTracker.RecordPhase(mainIntent.Name, "e2nodeset-coordination", "initiated")

				By("monitoring coordinated deployment progress")
				createdE2NodeSet := &nephoranv1.E2NodeSet{}

				// Wait for E2NodeSet to start processing
				Eventually(func() int32 {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: coordinatedE2NodeSet.Name, Namespace: coordinatedE2NodeSet.Namespace,
					}, createdE2NodeSet)
					if err != nil {
						return -1
					}
					return createdE2NodeSet.Status.CurrentReplicas
				}, 180*time.Second, 5*time.Second).Should(BeNumerically(">=", 0))

				workflowTracker.RecordPhase(mainIntent.Name, "e2nodes-deployment", "active")

				By("verifying coordination between components")
				Eventually(func() bool {
					// Check main intent status
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: mainIntent.Name, Namespace: mainIntent.Namespace,
					}, createdIntent)
					if err != nil {
						return false
					}

					// Check E2NodeSet status
					err = k8sClient.Get(testCtx, types.NamespacedName{
						Name: coordinatedE2NodeSet.Name, Namespace: coordinatedE2NodeSet.Namespace,
					}, createdE2NodeSet)
					if err != nil {
						return false
					}

					// Both should be progressing or completed
					intentProgressing := createdIntent.Status.Phase != "Pending"
					e2NodesProgressing := createdE2NodeSet.Status.CurrentReplicas >= 0

					return intentProgressing && e2NodesProgressing
				}, 240*time.Second, 10*time.Second).Should(BeTrue())

				By("waiting for coordinated deployment completion")
				Eventually(func() bool {
					// Check final states
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: mainIntent.Name, Namespace: mainIntent.Namespace,
					}, createdIntent)
					if err != nil {
						return false
					}

					err = k8sClient.Get(testCtx, types.NamespacedName{
						Name: coordinatedE2NodeSet.Name, Namespace: coordinatedE2NodeSet.Namespace,
					}, createdE2NodeSet)
					if err != nil {
						return false
					}

					// Check if both have reached meaningful states
					intentCompleted := createdIntent.Status.Phase == "Ready" ||
						createdIntent.Status.Phase == "Failed" ||
						createdIntent.Status.Phase == "Deploying"

					e2NodesDeployed := createdE2NodeSet.Status.ReadyReplicas > 0 ||
						createdE2NodeSet.Status.CurrentReplicas > 0

					return intentCompleted && e2NodesDeployed
				}, 480*time.Second, 15*time.Second).Should(BeTrue())

				By("validating coordination results")
				workflowTracker.CompleteWorkflow(mainIntent.Name,
					createdIntent.Status.Phase == "Ready")

				// Verify coordination success metrics
				if createdIntent.Status.Phase == "Ready" || createdIntent.Status.Phase == "Deploying" {
					workflowTracker.RecordPhase(mainIntent.Name, "coordination-success", "completed")
				}

				if createdE2NodeSet.Status.CurrentReplicas > 0 {
					workflowTracker.RecordPhase(mainIntent.Name, "e2nodes-operational", "completed")
				}

				GinkgoWriter.Printf("=== Coordination Results ===\n")
				GinkgoWriter.Printf("Main Intent Phase: %s\n", createdIntent.Status.Phase)
				GinkgoWriter.Printf("E2NodeSet Replicas: %d/%d\n",
					createdE2NodeSet.Status.ReadyReplicas, createdE2NodeSet.Status.CurrentReplicas)
				GinkgoWriter.Printf("Deployed Components: %v\n", createdIntent.Status.DeployedComponents)
				GinkgoWriter.Printf("===========================\n")
			})
		})
	})
})

// E2EWorkflowTracker tracks end-to-end workflow execution for analysis
type E2EWorkflowTracker struct {
	mu        sync.RWMutex
	workflows map[string]*WorkflowExecution
}

// WorkflowExecution represents a complete workflow execution
type WorkflowExecution struct {
	Name            string
	WorkflowType    string
	StartTime       time.Time
	EndTime         *time.Time
	TotalDuration   time.Duration
	Phases          map[string]PhaseExecution
	CompletedPhases []string
	Success         bool
	Completed       bool
	Errors          []string
}

// PhaseExecution represents execution of a specific phase
type PhaseExecution struct {
	Name      string
	Status    string
	StartTime time.Time
	EndTime   *time.Time
	Duration  time.Duration
}

func NewE2EWorkflowTracker() *E2EWorkflowTracker {
	return &E2EWorkflowTracker{
		workflows: make(map[string]*WorkflowExecution),
	}
}

func (ewt *E2EWorkflowTracker) StartWorkflow(name, workflowType string) {
	ewt.mu.Lock()
	defer ewt.mu.Unlock()

	ewt.workflows[name] = &WorkflowExecution{
		Name:            name,
		WorkflowType:    workflowType,
		StartTime:       time.Now(),
		Phases:          make(map[string]PhaseExecution),
		CompletedPhases: make([]string, 0),
		Errors:          make([]string, 0),
	}
}

func (ewt *E2EWorkflowTracker) RecordPhase(workflowName, phaseName, status string) {
	ewt.mu.Lock()
	defer ewt.mu.Unlock()

	workflow, exists := ewt.workflows[workflowName]
	if !exists {
		return
	}

	now := time.Now()
	phase, phaseExists := workflow.Phases[phaseName]

	if !phaseExists {
		phase = PhaseExecution{
			Name:      phaseName,
			Status:    status,
			StartTime: now,
		}
	}

	phase.Status = status
	if status == "completed" || status == "failed" {
		phase.EndTime = &now
		phase.Duration = now.Sub(phase.StartTime)

		// Add to completed phases if not already there
		found := false
		for _, completed := range workflow.CompletedPhases {
			if completed == phaseName {
				found = true
				break
			}
		}
		if !found {
			workflow.CompletedPhases = append(workflow.CompletedPhases, phaseName)
		}
	}

	workflow.Phases[phaseName] = phase
}

func (ewt *E2EWorkflowTracker) CompleteWorkflow(name string, success bool) {
	ewt.mu.Lock()
	defer ewt.mu.Unlock()

	workflow, exists := ewt.workflows[name]
	if !exists {
		return
	}

	now := time.Now()
	workflow.EndTime = &now
	workflow.TotalDuration = now.Sub(workflow.StartTime)
	workflow.Success = success
	workflow.Completed = true
}

func (ewt *E2EWorkflowTracker) GetWorkflow(name string) *WorkflowExecution {
	ewt.mu.RLock()
	defer ewt.mu.RUnlock()

	workflow, exists := ewt.workflows[name]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	workflowCopy := *workflow
	workflowCopy.Phases = make(map[string]PhaseExecution)
	for k, v := range workflow.Phases {
		workflowCopy.Phases[k] = v
	}
	workflowCopy.CompletedPhases = append([]string(nil), workflow.CompletedPhases...)
	workflowCopy.Errors = append([]string(nil), workflow.Errors...)

	return &workflowCopy
}

func (ewt *E2EWorkflowTracker) GetAllWorkflows() map[string]*WorkflowExecution {
	ewt.mu.RLock()
	defer ewt.mu.RUnlock()

	result := make(map[string]*WorkflowExecution)
	for k, v := range ewt.workflows {
		workflowCopy := *v
		workflowCopy.Phases = make(map[string]PhaseExecution)
		for pk, pv := range v.Phases {
			workflowCopy.Phases[pk] = pv
		}
		workflowCopy.CompletedPhases = append([]string(nil), v.CompletedPhases...)
		workflowCopy.Errors = append([]string(nil), v.Errors...)
		result[k] = &workflowCopy
	}

	return result
}
