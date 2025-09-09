<<<<<<< HEAD
=======
//go:build integration
// +build integration

>>>>>>> 6835433495e87288b95961af7173d866977175ff
/*
Package integration provides end-to-end integration testing for the Nephoran
Kubernetes operator in real cluster environments following 2025 best practices.
*/

<<<<<<< HEAD
//go:build integration
// +build integration

=======
>>>>>>> 6835433495e87288b95961af7173d866977175ff
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/v1alpha1"
)

const (
	// Integration test timeouts (longer than unit tests)
<<<<<<< HEAD
	operatorTimeout    = 5 * time.Minute
	resourceTimeout    = 2 * time.Minute
	deploymentTimeout  = 3 * time.Minute
	pollInterval       = 10 * time.Second
	shortPollInterval  = 2 * time.Second
=======
	operatorTimeout   = 5 * time.Minute
	resourceTimeout   = 2 * time.Minute
	deploymentTimeout = 3 * time.Minute
	pollInterval      = 10 * time.Second
	shortPollInterval = 2 * time.Second
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	// Operator deployment details
	operatorNamespace = "nephoran-system"
	operatorName      = "nephoran-controller-manager"
<<<<<<< HEAD
	
=======

>>>>>>> 6835433495e87288b95961af7173d866977175ff
	// Test namespace
	testNamespace = "nephoran-integration-test"
)

var (
<<<<<<< HEAD
	cfg         *rest.Config
	k8sClient   client.Client
	clientset   *kubernetes.Clientset
	ctx         context.Context
	cancel      context.CancelFunc
=======
	cfg       *rest.Config
	k8sClient client.Client
	clientset *kubernetes.Clientset
	ctx       context.Context
	cancel    context.CancelFunc
>>>>>>> 6835433495e87288b95961af7173d866977175ff
)

func TestIntegration(t *testing.T) {
	if os.Getenv("KUBECONFIG") == "" && os.Getenv("KUBERNETES_SERVICE_HOST") == "" {
		t.Skip("Integration tests require a Kubernetes cluster (KUBECONFIG or in-cluster config)")
	}

	RegisterFailHandler(Fail)

	suiteConfig := GinkgoConfiguration{
		LabelFilter:   "integration",
		ParallelTotal: 1, // Run integration tests sequentially
		Timeout:       10 * time.Minute,
	}

	RunSpecs(t, "Nephoran Operator Integration Test Suite", suiteConfig)
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("setting up Kubernetes client")
	var err error
	cfg, err = config.GetConfig()
	Expect(err).NotTo(HaveOccurred())

	// Setup controller-runtime client
	err = intentv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())

	// Setup standard Kubernetes clientset
	clientset, err = kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())

	By("verifying cluster connectivity")
	_, err = clientset.ServerVersion()
	Expect(err).NotTo(HaveOccurred())

	By("creating test namespace")
	testNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
			Labels: map[string]string{
				"test":    "integration",
				"app":     "nephoran",
				"version": "test",
			},
		},
	}
	err = k8sClient.Create(ctx, testNs)
	if err != nil && !errors.IsAlreadyExists(err) {
		Expect(err).NotTo(HaveOccurred())
	}

	By("verifying operator deployment")
	Eventually(func() error {
		deployment := &appsv1.Deployment{}
		return k8sClient.Get(ctx, types.NamespacedName{
			Name:      operatorName,
			Namespace: operatorNamespace,
		}, deployment)
	}, operatorTimeout, pollInterval).Should(Succeed())

	By("waiting for operator to be ready")
	Eventually(func() bool {
		deployment := &appsv1.Deployment{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      operatorName,
			Namespace: operatorNamespace,
		}, deployment)
		if err != nil {
			return false
		}
		return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas &&
			deployment.Status.ReadyReplicas > 0
	}, deploymentTimeout, pollInterval).Should(BeTrue())

	logTestStep("Integration test environment setup completed")
})

var _ = AfterSuite(func() {
	By("cleaning up test namespace")
	testNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
	}
	err := k8sClient.Delete(ctx, testNs)
	if err != nil && !errors.IsNotFound(err) {
		logf.Log.Error(err, "Failed to cleanup test namespace")
	}

	cancel()
})

var _ = Describe("Nephoran Operator Integration Tests", func() {
	Context("When the operator is deployed", func() {
		It("should have all required CRDs installed", func() {
			By("checking NetworkIntent CRD")
			Eventually(func() error {
				// Try to list NetworkIntents to verify CRD exists
				intentList := &intentv1alpha1.NetworkIntentList{}
				return k8sClient.List(ctx, intentList, client.InNamespace(testNamespace))
			}, resourceTimeout, pollInterval).Should(Succeed())

			By("checking OranCluster CRD")
			Eventually(func() error {
				// Try to list OranClusters to verify CRD exists
				clusterList := &intentv1alpha1.OranClusterList{}
				return k8sClient.List(ctx, clusterList, client.InNamespace(testNamespace))
			}, resourceTimeout, pollInterval).Should(Succeed())
		})

		It("should have proper RBAC permissions", func() {
			By("verifying service account exists")
			sa := &corev1.ServiceAccount{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      operatorName,
					Namespace: operatorNamespace,
				}, sa)
			}, resourceTimeout, pollInterval).Should(Succeed())

			By("verifying operator can watch its resources")
			// The operator should be able to watch NetworkIntents across namespaces
			intentList := &intentv1alpha1.NetworkIntentList{}
			Expect(k8sClient.List(ctx, intentList)).Should(Succeed())
		})

		It("should be responding to health checks", func() {
			By("checking operator pod health")
			pods := &corev1.PodList{}
			Eventually(func() error {
<<<<<<< HEAD
				return k8sClient.List(ctx, pods, client.InNamespace(operatorNamespace), 
=======
				return k8sClient.List(ctx, pods, client.InNamespace(operatorNamespace),
>>>>>>> 6835433495e87288b95961af7173d866977175ff
					client.MatchingLabels{"control-plane": "controller-manager"})
			}, resourceTimeout, pollInterval).Should(Succeed())

			Expect(pods.Items).To(HaveLen(1), "Expected exactly one operator pod")
			pod := pods.Items[0]
			Expect(pod.Status.Phase).To(Equal(corev1.PodRunning))

			// Check readiness
			for _, condition := range pod.Status.Conditions {
				if condition.Type == corev1.PodReady {
					Expect(condition.Status).To(Equal(corev1.ConditionTrue))
				}
			}
		})
	})

	Context("When creating NetworkIntent resources", func() {
		var networkIntent *intentv1alpha1.NetworkIntent

		BeforeEach(func() {
			networkIntent = &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "integration-test-intent",
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					ScalingIntent: intentv1alpha1.ScalingIntent{
						Action: "scale-up",
						Target: intentv1alpha1.ScalingTarget{
							Component: "cu-cp",
							Replicas:  3,
							Region:    "us-west-2",
						},
						Constraints: intentv1alpha1.ScalingConstraints{
							MaxReplicas: 10,
							MinReplicas: 1,
							ResourceLimits: map[string]string{
								"cpu":    "2",
								"memory": "4Gi",
							},
						},
					},
					Priority: "high",
					Source:   "integration-test",
				},
			}
		})

		AfterEach(func() {
			By("cleaning up NetworkIntent")
			err := k8sClient.Delete(ctx, networkIntent)
			if err != nil && !errors.IsNotFound(err) {
				logf.Log.Error(err, "Failed to cleanup NetworkIntent")
			}
		})

		It("should create and process NetworkIntent successfully", func() {
			By("creating the NetworkIntent")
			logTestStep("Creating NetworkIntent", "name", networkIntent.Name, "namespace", networkIntent.Namespace)
			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			By("waiting for NetworkIntent to be processed")
			Eventually(func() string {
				intent := &intentv1alpha1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      networkIntent.Name,
					Namespace: networkIntent.Namespace,
				}, intent)
				if err != nil {
					return ""
				}
				return intent.Status.Phase
			}, resourceTimeout, shortPollInterval).Should(Not(BeEmpty()))

			By("verifying NetworkIntent status is updated")
			Eventually(func() bool {
				intent := &intentv1alpha1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      networkIntent.Name,
					Namespace: networkIntent.Namespace,
				}, intent)
				if err != nil {
					return false
				}
				return intent.Status.LastUpdated != nil &&
					len(intent.Status.Message) > 0
			}, resourceTimeout, shortPollInterval).Should(BeTrue())
		})

		It("should handle NetworkIntent updates", func() {
			By("creating the initial NetworkIntent")
			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			By("waiting for initial processing")
			Eventually(func() string {
				intent := &intentv1alpha1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      networkIntent.Name,
					Namespace: networkIntent.Namespace,
				}, intent)
				if err != nil {
					return ""
				}
				return intent.Status.Phase
			}, resourceTimeout, shortPollInterval).Should(Not(BeEmpty()))

			By("updating the NetworkIntent")
			updatedIntent := &intentv1alpha1.NetworkIntent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      networkIntent.Name,
				Namespace: networkIntent.Namespace,
			}, updatedIntent)).Should(Succeed())

			updatedIntent.Spec.ScalingIntent.Target.Replicas = 5
			updatedIntent.Spec.Priority = "medium"
			Expect(k8sClient.Update(ctx, updatedIntent)).Should(Succeed())

			By("verifying the update is processed")
			Eventually(func() int32 {
				intent := &intentv1alpha1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      networkIntent.Name,
					Namespace: networkIntent.Namespace,
				}, intent)
				if err != nil {
					return 0
				}
				return intent.Spec.ScalingIntent.Target.Replicas
			}, resourceTimeout, shortPollInterval).Should(Equal(int32(5)))
		})
	})

	Context("When testing operator resilience", func() {
		It("should handle operator pod restart gracefully", func() {
			By("getting current operator pod")
			pods := &corev1.PodList{}
<<<<<<< HEAD
			Expect(k8sClient.List(ctx, pods, client.InNamespace(operatorNamespace), 
				client.MatchingLabels{"control-plane": "controller-manager"})).Should(Succeed())
			
=======
			Expect(k8sClient.List(ctx, pods, client.InNamespace(operatorNamespace),
				client.MatchingLabels{"control-plane": "controller-manager"})).Should(Succeed())

>>>>>>> 6835433495e87288b95961af7173d866977175ff
			Expect(pods.Items).To(HaveLen(1))
			originalPod := pods.Items[0]

			By("deleting operator pod to trigger restart")
			Expect(k8sClient.Delete(ctx, &originalPod)).Should(Succeed())

			By("waiting for new pod to be created and ready")
			Eventually(func() bool {
				newPods := &corev1.PodList{}
<<<<<<< HEAD
				err := k8sClient.List(ctx, newPods, client.InNamespace(operatorNamespace), 
=======
				err := k8sClient.List(ctx, newPods, client.InNamespace(operatorNamespace),
>>>>>>> 6835433495e87288b95961af7173d866977175ff
					client.MatchingLabels{"control-plane": "controller-manager"})
				if err != nil {
					return false
				}

				if len(newPods.Items) != 1 {
					return false
				}

				newPod := newPods.Items[0]
<<<<<<< HEAD
				return newPod.UID != originalPod.UID && 
=======
				return newPod.UID != originalPod.UID &&
>>>>>>> 6835433495e87288b95961af7173d866977175ff
					newPod.Status.Phase == corev1.PodRunning &&
					isPodReady(newPod)
			}, deploymentTimeout, pollInterval).Should(BeTrue())

			By("verifying operator functionality after restart")
			testIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "resilience-test-intent",
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					ScalingIntent: intentv1alpha1.ScalingIntent{
						Action: "scale-up",
						Target: intentv1alpha1.ScalingTarget{
							Component: "du",
							Replicas:  2,
							Region:    "us-east-1",
						},
					},
					Priority: "low",
					Source:   "resilience-test",
				},
			}

			Expect(k8sClient.Create(ctx, testIntent)).Should(Succeed())

			Eventually(func() string {
				intent := &intentv1alpha1.NetworkIntent{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      testIntent.Name,
					Namespace: testIntent.Namespace,
				}, intent)
				if err != nil {
					return ""
				}
				return intent.Status.Phase
			}, resourceTimeout, shortPollInterval).Should(Not(BeEmpty()))

			// Cleanup
			Expect(k8sClient.Delete(ctx, testIntent)).Should(Succeed())
		})

		It("should handle high load scenarios", func() {
			const numIntents = 20
			intents := make([]*intentv1alpha1.NetworkIntent, numIntents)

			By("creating multiple NetworkIntents concurrently")
			for i := 0; i < numIntents; i++ {
				intents[i] = &intentv1alpha1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("load-test-intent-%d", i),
						Namespace: testNamespace,
					},
					Spec: intentv1alpha1.NetworkIntentSpec{
						ScalingIntent: intentv1alpha1.ScalingIntent{
							Action: "scale-up",
							Target: intentv1alpha1.ScalingTarget{
								Component: "cu-up",
								Replicas:  int32(i%5 + 1),
								Region:    fmt.Sprintf("region-%d", i%3),
							},
						},
						Priority: []string{"low", "medium", "high"}[i%3],
						Source:   "load-test",
					},
				}

				Expect(k8sClient.Create(ctx, intents[i])).Should(Succeed())
			}

			By("verifying all NetworkIntents are processed")
			for i := 0; i < numIntents; i++ {
				Eventually(func() string {
					intent := &intentv1alpha1.NetworkIntent{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name:      intents[i].Name,
						Namespace: intents[i].Namespace,
					}, intent)
					if err != nil {
						return ""
					}
					return intent.Status.Phase
				}, resourceTimeout, shortPollInterval).Should(Not(BeEmpty()))
			}

			By("cleaning up load test resources")
			for i := 0; i < numIntents; i++ {
				err := k8sClient.Delete(ctx, intents[i])
				if err != nil && !errors.IsNotFound(err) {
					logf.Log.Error(err, "Failed to cleanup load test intent", "name", intents[i].Name)
				}
			}
		})
	})

	Context("When testing webhook functionality", func() {
		It("should validate NetworkIntent resources", func() {
			By("attempting to create invalid NetworkIntent")
			invalidIntent := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-intent",
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					ScalingIntent: intentv1alpha1.ScalingIntent{
						Action: "invalid-action", // Invalid action
						Target: intentv1alpha1.ScalingTarget{
							Component: "cu-cp",
							Replicas:  0, // Invalid replica count
						},
					},
				},
			}

			err := k8sClient.Create(ctx, invalidIntent)
			if err != nil {
				// Validation webhook is working
				Expect(err.Error()).To(ContainSubstring("validation"))
			} else {
				// No validation webhook, but controller should handle it
				Eventually(func() string {
					intent := &intentv1alpha1.NetworkIntent{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name:      invalidIntent.Name,
						Namespace: invalidIntent.Namespace,
					}, intent)
					if err != nil {
						return ""
					}
					return intent.Status.Phase
				}, resourceTimeout, shortPollInterval).Should(Equal("Failed"))

				// Cleanup
				Expect(k8sClient.Delete(ctx, invalidIntent)).Should(Succeed())
			}
		})
	})
})

// Helper functions

func logTestStep(msg string, keysAndValues ...interface{}) {
	logf.Log.Info(fmt.Sprintf("[INTEGRATION TEST] %s", msg), keysAndValues...)
}

func isPodReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
<<<<<<< HEAD
}
=======
}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
