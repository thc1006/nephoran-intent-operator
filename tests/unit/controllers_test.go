// Package unit provides comprehensive unit tests for Nephoran Intent Operator controllers
package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nephranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	"github.com/thc1006/nephoran-intent-operator/tests/framework"
)

// ControllerTestSuite provides comprehensive testing for all controllers
type ControllerTestSuite struct {
	*framework.TestSuite
}

// TestControllers runs the controller test suite
func TestControllers(t *testing.T) {
	suite.Run(t, &ControllerTestSuite{
		TestSuite: framework.NewTestSuite(nil),
	})
}

// SetupSuite initializes the controller test environment
func (suite *ControllerTestSuite) SetupSuite() {
	suite.TestSuite.SetupSuite()

	// Setup controller-specific mocks
	suite.setupControllerMocks()
}

// setupControllerMocks configures mocks for controller testing
func (suite *ControllerTestSuite) setupControllerMocks() {
	// Setup LLM mock responses
	llmMock := suite.GetMocks().GetLLMMock()
	llmMock.On("ProcessIntent", mock.Anything, mock.Anything).Return(
		map[string]interface{}{
			"resources": map[string]interface{}{
				"requests": map[string]string{
					"cpu":    "1000m",
					"memory": "2Gi",
				},
				"limits": map[string]string{
					"cpu":    "2000m",
					"memory": "4Gi",
				},
			},
		}, nil)

	// Setup Weaviate mock for RAG queries
	weaviateMock := suite.GetMocks().GetWeaviateMock()
	weaviateMock.On("Query").Return(&MockGraphQLResponse{}, nil)
}

// TestNetworkIntentController tests the NetworkIntent controller comprehensively
func (suite *ControllerTestSuite) TestNetworkIntentController() {
	ginkgo.Describe("NetworkIntent Controller", func() {
		var (
			ctx        context.Context
			controller *controllers.NetworkIntentReconciler
			intent     *nephranv1.NetworkIntent
		)

		ginkgo.BeforeEach(func() {
			ctx = suite.GetContext()
			controller = &controllers.NetworkIntentReconciler{
				Client: suite.GetK8sClient(),
				Scheme: suite.GetK8sClient().Scheme(),
			}

			intent = &nephranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-intent",
					Namespace: "default",
				},
				Spec: nephranv1.NetworkIntentSpec{
					Description: "Deploy AMF with 3 replicas for high availability",
					Priority:    "high",
				},
			}
		})

		ginkgo.Context("Intent Processing", func() {
			ginkgo.It("should successfully process a deployment intent", func() {
				// Create the NetworkIntent
				err := suite.GetK8sClient().Create(ctx, intent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Reconcile the intent
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}

				result, err := controller.Reconcile(ctx, req)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(result.Requeue).To(gomega.BeFalse())

				// Verify the intent was processed
				processedIntent := &nephranv1.NetworkIntent{}
				err = suite.GetK8sClient().Get(ctx, req.NamespacedName, processedIntent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(processedIntent.Status.Phase).To(gomega.Equal("Processed"))
			})

			ginkgo.It("should handle LLM processing failures gracefully", func() {
				// Configure mock to return error
				llmMock := suite.GetMocks().GetLLMMock()
				llmMock.ExpectedCalls = nil // Reset expectations
				llmMock.On("ProcessIntent", mock.Anything, mock.Anything).Return(
					nil, fmt.Errorf("LLM service unavailable"))

				// Create the NetworkIntent
				err := suite.GetK8sClient().Create(ctx, intent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Reconcile the intent
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}

				result, err := controller.Reconcile(ctx, req)
				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(result.Requeue).To(gomega.BeTrue())

				// Verify error status
				processedIntent := &nephranv1.NetworkIntent{}
				err = suite.GetK8sClient().Get(ctx, req.NamespacedName, processedIntent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(processedIntent.Status.Phase).To(gomega.Equal("Failed"))
			})

			ginkgo.It("should validate intent specifications", func() {
				// Create invalid intent
				invalidIntent := &nephranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-intent",
						Namespace: "default",
					},
					Spec: nephranv1.NetworkIntentSpec{
						Description: "", // Empty description should fail validation
						Priority:    "invalid-priority",
					},
				}

				err := suite.GetK8sClient().Create(ctx, invalidIntent)
				gomega.Expect(err).To(gomega.HaveOccurred())
			})
		})

		ginkgo.Context("Status Management", func() {
			ginkgo.It("should update status during processing lifecycle", func() {
				// Create the NetworkIntent
				err := suite.GetK8sClient().Create(ctx, intent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Initial status should be empty
				gomega.Expect(intent.Status.Phase).To(gomega.BeEmpty())

				// Reconcile should update status
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}

				_, err = controller.Reconcile(ctx, req)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Check updated status
				updatedIntent := &nephranv1.NetworkIntent{}
				err = suite.GetK8sClient().Get(ctx, req.NamespacedName, updatedIntent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(updatedIntent.Status.Phase).NotTo(gomega.BeEmpty())
				gomega.Expect(updatedIntent.Status.LastUpdateTime).NotTo(gomega.BeNil())
			})
		})

		ginkgo.Context("Performance Testing", func() {
			ginkgo.It("should process intents within performance thresholds", func() {
				start := time.Now()

				// Create and process intent
				err := suite.GetK8sClient().Create(ctx, intent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}

				_, err = controller.Reconcile(ctx, req)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				processingTime := time.Since(start)

				// Record performance metrics
				suite.GetMetrics().RecordLatency("intent_processing", processingTime)

				// Assert performance threshold (2 seconds for unit tests)
				gomega.Expect(processingTime).To(gomega.BeNumerically("<", 2*time.Second))
			})
		})
	})
}

// TestE2NodeSetController tests the E2NodeSet controller
func (suite *ControllerTestSuite) TestE2NodeSetController() {
	ginkgo.Describe("E2NodeSet Controller", func() {
		var (
			ctx        context.Context
			controller *controllers.E2NodeSetReconciler
			nodeSet    *nephranv1.E2NodeSet
		)

		ginkgo.BeforeEach(func() {
			ctx = suite.GetContext()
			controller = &controllers.E2NodeSetReconciler{
				Client: suite.GetK8sClient(),
				Scheme: suite.GetK8sClient().Scheme(),
			}

			nodeSet = &nephranv1.E2NodeSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-nodeset",
					Namespace: "default",
				},
				Spec: nephranv1.E2NodeSetSpec{
					Replicas: 3,
					Template: nephranv1.E2NodeTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "e2-node",
							},
						},
						Spec: nephranv1.E2NodeSpec{
							NodeID:              "test-node",
							E2InterfaceVersion:  "v20",
							SupportedRANFunctions: []nephranv1.RANFunction{
								{
									FunctionID:  1,
									Revision:    1,
									Description: "Test RAN Function",
									OID:         "1.3.6.1.4.1.1.1",
								},
							},
						},
					},
				},
			}
		})

		ginkgo.Context("Replica Management", func() {
			ginkgo.It("should create specified number of E2 nodes", func() {
				// Create the E2NodeSet
				err := suite.GetK8sClient().Create(ctx, nodeSet)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Reconcile
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      nodeSet.Name,
						Namespace: nodeSet.Namespace,
					},
				}

				result, err := controller.Reconcile(ctx, req)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(result.Requeue).To(gomega.BeFalse())

				// Verify ConfigMaps were created for each replica
				configMaps := &corev1.ConfigMapList{}
				err = suite.GetK8sClient().List(ctx, configMaps, client.InNamespace("default"))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Count ConfigMaps with our labels
				e2NodeCMs := 0
				for _, cm := range configMaps.Items {
					if cm.Labels["e2nodeset"] == nodeSet.Name {
						e2NodeCMs++
					}
				}
				gomega.Expect(e2NodeCMs).To(gomega.Equal(int(nodeSet.Spec.Replicas)))
			})

			ginkgo.It("should scale replicas up and down", func() {
				// Create initial E2NodeSet
				err := suite.GetK8sClient().Create(ctx, nodeSet)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Initial reconcile
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      nodeSet.Name,
						Namespace: nodeSet.Namespace,
					},
				}

				_, err = controller.Reconcile(ctx, req)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Scale up to 5 replicas
				updatedNodeSet := &nephranv1.E2NodeSet{}
				err = suite.GetK8sClient().Get(ctx, req.NamespacedName, updatedNodeSet)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				updatedNodeSet.Spec.Replicas = 5
				err = suite.GetK8sClient().Update(ctx, updatedNodeSet)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Reconcile again
				_, err = controller.Reconcile(ctx, req)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Verify 5 ConfigMaps exist
				configMaps := &corev1.ConfigMapList{}
				err = suite.GetK8sClient().List(ctx, configMaps, client.InNamespace("default"))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				e2NodeCMs := 0
				for _, cm := range configMaps.Items {
					if cm.Labels["e2nodeset"] == nodeSet.Name {
						e2NodeCMs++
					}
				}
				gomega.Expect(e2NodeCMs).To(gomega.Equal(5))

				// Scale down to 2 replicas
				updatedNodeSet.Spec.Replicas = 2
				err = suite.GetK8sClient().Update(ctx, updatedNodeSet)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				_, err = controller.Reconcile(ctx, req)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Verify only 2 ConfigMaps remain
				configMaps = &corev1.ConfigMapList{}
				err = suite.GetK8sClient().List(ctx, configMaps, client.InNamespace("default"))
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				e2NodeCMs = 0
				for _, cm := range configMaps.Items {
					if cm.Labels["e2nodeset"] == nodeSet.Name {
						e2NodeCMs++
					}
				}
				gomega.Expect(e2NodeCMs).To(gomega.Equal(2))
			})
		})

		ginkgo.Context("Status Updates", func() {
			ginkgo.It("should maintain accurate ready replica count", func() {
				// Create the E2NodeSet
				err := suite.GetK8sClient().Create(ctx, nodeSet)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Reconcile
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      nodeSet.Name,
						Namespace: nodeSet.Namespace,
					},
				}

				_, err = controller.Reconcile(ctx, req)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				// Check status
				updatedNodeSet := &nephranv1.E2NodeSet{}
				err = suite.GetK8sClient().Get(ctx, req.NamespacedName, updatedNodeSet)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				gomega.Expect(updatedNodeSet.Status.ReadyReplicas).To(gomega.Equal(nodeSet.Spec.Replicas))
				gomega.Expect(updatedNodeSet.Status.CurrentReplicas).To(gomega.Equal(nodeSet.Spec.Replicas))
			})
		})
	})
}

// TestLoadTesting performs load testing on controllers
func (suite *ControllerTestSuite) TestLoadTesting() {
	ginkgo.Describe("Controller Load Testing", func() {
		ginkgo.It("should handle concurrent intent processing", func() {
			if !suite.GetTestConfig().LoadTestEnabled {
				ginkgo.Skip("Load testing disabled")
			}

			controller := &controllers.NetworkIntentReconciler{
				Client: suite.GetK8sClient(),
				Scheme: suite.GetK8sClient().Scheme(),
			}

			err := suite.RunLoadTest(func() error {
				// Create a unique intent for each load test iteration
				intent := &nephranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("load-test-intent-%d", time.Now().UnixNano()),
						Namespace: "default",
					},
					Spec: nephranv1.NetworkIntentSpec{
						Description: "Load test intent for AMF deployment",
						Priority:    "medium",
					},
				}

				err := suite.GetK8sClient().Create(suite.GetContext(), intent)
				if err != nil {
					return err
				}

				// Reconcile
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}

				_, err = controller.Reconcile(suite.GetContext(), req)
				return err
			})

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
}

// TestChaosEngineering performs chaos testing on controllers
func (suite *ControllerTestSuite) TestChaosEngineering() {
	ginkgo.Describe("Controller Chaos Testing", func() {
		ginkgo.It("should handle external service failures", func() {
			if !suite.GetTestConfig().ChaosTestEnabled {
				ginkgo.Skip("Chaos testing disabled")
			}

			controller := &controllers.NetworkIntentReconciler{
				Client: suite.GetK8sClient(),
				Scheme: suite.GetK8sClient().Scheme(),
			}

			err := suite.RunChaosTest(func() error {
				intent := &nephranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("chaos-test-intent-%d", time.Now().UnixNano()),
						Namespace: "default",
					},
					Spec: nephranv1.NetworkIntentSpec{
						Description: "Chaos test intent for resilience testing",
						Priority:    "low",
					},
				}

				err := suite.GetK8sClient().Create(suite.GetContext(), intent)
				if err != nil {
					return err
				}

				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}

				// Should handle failures gracefully
				_, _ = controller.Reconcile(suite.GetContext(), req)
				// Errors are expected in chaos testing
				return nil
			})

			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})
	})
}

// MockGraphQLResponse provides a simple mock for GraphQL responses
type MockGraphQLResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors interface{} `json:"errors,omitempty"`
}

var _ = ginkgo.Describe("Controller Integration", func() {
	var testSuite *ControllerTestSuite

	ginkgo.BeforeEach(func() {
		testSuite = &ControllerTestSuite{
			TestSuite: framework.NewTestSuite(nil),
		}
		testSuite.SetupSuite()
	})

	ginkgo.AfterEach(func() {
		testSuite.TearDownSuite()
	})

	ginkgo.Context("NetworkIntent Controller", func() {
		testSuite.TestNetworkIntentController()
	})

	ginkgo.Context("E2NodeSet Controller", func() {
		testSuite.TestE2NodeSetController()
	})

	ginkgo.Context("Load Testing", func() {
		testSuite.TestLoadTesting()
	})

	ginkgo.Context("Chaos Engineering", func() {
		testSuite.TestChaosEngineering()
	})
})
