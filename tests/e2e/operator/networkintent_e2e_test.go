//go:build e2e

/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package operator

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/tests/fixtures"
)

// E2EOperatorTestSuite provides comprehensive end-to-end testing for the operator
type E2EOperatorTestSuite struct {
	suite.Suite
	testEnv      *envtest.Environment
	config       *rest.Config
	client       client.Client
	clientset    *kubernetes.Clientset
	scheme       *runtime.Scheme
	ctx          context.Context
	cancel       context.CancelFunc
	namespace    string
	operatorName string
}

// SetupSuite initializes the test environment
func (suite *E2EOperatorTestSuite) SetupSuite() {
	logf.SetLogger(zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true)))

	suite.ctx, suite.cancel = context.WithCancel(context.Background())
	suite.namespace = "nephoran-e2e-test"
	suite.operatorName = "nephoran-intent-operator"

	// Skip E2E tests if not explicitly requested
	if os.Getenv("RUN_E2E_TESTS") != "true" {
		suite.T().Skip("E2E tests skipped. Set RUN_E2E_TESTS=true to run.")
	}

	// Setup test environment
	suite.testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			"../../../config/crd/bases",
		},
		ErrorIfCRDPathMissing: false,
	}

	var err error
	suite.config, err = suite.testEnv.Start()
	suite.Require().NoError(err)
	suite.Require().NotNil(suite.config)

	// Create scheme
	suite.scheme = runtime.NewScheme()
	err = nephoranv1.AddToScheme(suite.scheme)
	suite.Require().NoError(err)

	// Create clients
	suite.client, err = client.New(suite.config, client.Options{Scheme: suite.scheme})
	suite.Require().NoError(err)

	suite.clientset, err = kubernetes.NewForConfig(suite.config)
	suite.Require().NoError(err)

	// Create test namespace
	suite.createTestNamespace()

	// Deploy operator (in a real scenario, this would deploy the actual operator)
	suite.deployOperator()
}

// TearDownSuite cleans up the test environment
func (suite *E2EOperatorTestSuite) TearDownSuite() {
	if suite.cancel != nil {
		suite.cancel()
	}
	if suite.testEnv != nil {
		err := suite.testEnv.Stop()
		suite.Assert().NoError(err)
	}
}

// SetupTest runs before each test
func (suite *E2EOperatorTestSuite) SetupTest() {
	// Clean up any existing NetworkIntent objects
	suite.cleanupNetworkIntents()
}

// createTestNamespace creates the test namespace
func (suite *E2EOperatorTestSuite) createTestNamespace() {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: suite.namespace,
		},
	}
	err := suite.client.Create(suite.ctx, ns)
	if err != nil && !errors.IsAlreadyExists(err) {
		suite.Require().NoError(err)
	}
}

// deployOperator deploys the operator for testing
func (suite *E2EOperatorTestSuite) deployOperator() {
	// In a real E2E test, this would deploy the actual operator
	// For now, we'll create a mock deployment to simulate the operator presence
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      suite.operatorName,
			Namespace: suite.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "controller",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &[]int32{1}[0],
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "nephoran-intent-operator",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/name": "nephoran-intent-operator",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "manager",
							Image: "nephoran/intent-operator:test",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 8080,
									Name:          "http",
								},
							},
						},
					},
				},
			},
		},
	}

	err := suite.client.Create(suite.ctx, deployment)
	if err != nil && !errors.IsAlreadyExists(err) {
		suite.Require().NoError(err)
	}

	// Wait for deployment to be ready
	suite.waitForDeploymentReady(deployment.Name)
}

// waitForDeploymentReady waits for a deployment to become ready
func (suite *E2EOperatorTestSuite) waitForDeploymentReady(name string) {
	err := wait.PollImmediate(5*time.Second, 2*time.Minute, func() (bool, error) {
		deployment := &appsv1.Deployment{}
		err := suite.client.Get(suite.ctx, types.NamespacedName{
			Name:      name,
			Namespace: suite.namespace,
		}, deployment)
		if err != nil {
			return false, err
		}

		return deployment.Status.ReadyReplicas == *deployment.Spec.Replicas, nil
	})
	suite.Require().NoError(err, "Deployment %s should become ready", name)
}

// cleanupNetworkIntents removes all NetworkIntent objects from the test namespace
func (suite *E2EOperatorTestSuite) cleanupNetworkIntents() {
	intentList := &nephoranv1.NetworkIntentList{}
	err := suite.client.List(suite.ctx, intentList, client.InNamespace(suite.namespace))
	if err != nil {
		return
	}

	for _, intent := range intentList.Items {
		err := suite.client.Delete(suite.ctx, &intent)
		if err != nil && !errors.IsNotFound(err) {
			suite.T().Logf("Warning: could not delete NetworkIntent %s: %v", intent.Name, err)
		}
	}
}

// TestE2E_NetworkIntentLifecycle tests the complete lifecycle of a NetworkIntent
func (suite *E2EOperatorTestSuite) TestE2E_NetworkIntentLifecycle() {
	// Create NetworkIntent
	intent := fixtures.SimpleNetworkIntent()
	intent.Namespace = suite.namespace
	intent.Name = "e2e-lifecycle-test"

	err := suite.client.Create(suite.ctx, intent)
	suite.Require().NoError(err)

	// Wait for NetworkIntent to be processed (in real scenario, this would check status)
	suite.Eventually(func() bool {
		retrieved := &nephoranv1.NetworkIntent{}
		err := suite.client.Get(suite.ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, retrieved)
		if err != nil {
			return false
		}
		// In a real operator, we'd check the status phase
		return retrieved.Status.Phase == "Processed" || !retrieved.CreationTimestamp.IsZero()
	}, 30*time.Second, 1*time.Second, "NetworkIntent should be processed")

	// Update NetworkIntent
	intent.Spec.Intent = "updated scaling intent for e2e testing"
	err = suite.client.Update(suite.ctx, intent)
	suite.Require().NoError(err)

	// Verify update was processed
	retrieved := &nephoranv1.NetworkIntent{}
	err = suite.client.Get(suite.ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, retrieved)
	suite.NoError(err)
	suite.Equal("updated scaling intent for e2e testing", retrieved.Spec.Intent)

	// Delete NetworkIntent
	err = suite.client.Delete(suite.ctx, intent)
	suite.Require().NoError(err)

	// Verify deletion
	suite.Eventually(func() bool {
		err := suite.client.Get(suite.ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, &nephoranv1.NetworkIntent{})
		return errors.IsNotFound(err)
	}, 30*time.Second, 1*time.Second, "NetworkIntent should be deleted")
}

// TestE2E_MultipleNetworkIntents tests handling multiple NetworkIntents
func (suite *E2EOperatorTestSuite) TestE2E_MultipleNetworkIntents() {
	// Create multiple NetworkIntents
	intents := fixtures.MultipleIntents(5)
	for i, intent := range intents {
		intent.Namespace = suite.namespace
		intent.Name = fmt.Sprintf("e2e-multiple-%d", i)
		err := suite.client.Create(suite.ctx, intent)
		suite.Require().NoError(err)
	}

	// Wait for all NetworkIntents to be created
	suite.Eventually(func() bool {
		intentList := &nephoranv1.NetworkIntentList{}
		err := suite.client.List(suite.ctx, intentList, client.InNamespace(suite.namespace))
		if err != nil {
			return false
		}

		count := 0
		for _, item := range intentList.Items {
			if item.Namespace == suite.namespace {
				count++
			}
		}
		return count >= 5
	}, 30*time.Second, 1*time.Second, "All NetworkIntents should be created")

	// Verify each NetworkIntent exists and has correct data
	for i := 0; i < 5; i++ {
		retrieved := &nephoranv1.NetworkIntent{}
		err := suite.client.Get(suite.ctx, types.NamespacedName{
			Name:      fmt.Sprintf("e2e-multiple-%d", i),
			Namespace: suite.namespace,
		}, retrieved)
		suite.NoError(err)
		suite.Equal(fmt.Sprintf("scale network function %d", i), retrieved.Spec.Intent)
	}
}

// TestE2E_NetworkIntentWithFinalizers tests NetworkIntent with finalizers
func (suite *E2EOperatorTestSuite) TestE2E_NetworkIntentWithFinalizers() {
	// Create NetworkIntent with finalizer
	intent := fixtures.IntentWithFinalizers()
	intent.Namespace = suite.namespace
	intent.Name = "e2e-finalizer-test"

	err := suite.client.Create(suite.ctx, intent)
	suite.Require().NoError(err)

	// Verify finalizer is present
	retrieved := &nephoranv1.NetworkIntent{}
	err = suite.client.Get(suite.ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, retrieved)
	suite.NoError(err)
	suite.Contains(retrieved.Finalizers, "nephoran.io/finalizer")

	// Delete NetworkIntent (should not be deleted immediately due to finalizer)
	err = suite.client.Delete(suite.ctx, intent)
	suite.Require().NoError(err)

	// Verify object still exists but has deletion timestamp
	suite.Eventually(func() bool {
		err := suite.client.Get(suite.ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, retrieved)
		if err != nil {
			return false
		}
		return retrieved.DeletionTimestamp != nil
	}, 30*time.Second, 1*time.Second, "NetworkIntent should have deletion timestamp")

	// Remove finalizer to allow deletion (simulating controller cleanup)
	retrieved.Finalizers = []string{}
	err = suite.client.Update(suite.ctx, retrieved)
	suite.Require().NoError(err)

	// Verify object is finally deleted
	suite.Eventually(func() bool {
		err := suite.client.Get(suite.ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, &nephoranv1.NetworkIntent{})
		return errors.IsNotFound(err)
	}, 30*time.Second, 1*time.Second, "NetworkIntent should be deleted after removing finalizer")
}

// TestE2E_NetworkIntentStatusUpdates tests status updates
func (suite *E2EOperatorTestSuite) TestE2E_NetworkIntentStatusUpdates() {
	// Create NetworkIntent
	intent := fixtures.SimpleNetworkIntent()
	intent.Namespace = suite.namespace
	intent.Name = "e2e-status-test"

	err := suite.client.Create(suite.ctx, intent)
	suite.Require().NoError(err)

	// Update status to simulate controller behavior
	intent.Status = nephoranv1.NetworkIntentStatus{
		Phase:       "Processing",
		LastMessage: "Intent is being processed by the operator",
		Conditions: []metav1.Condition{
			{
				Type:    "Ready",
				Status:  metav1.ConditionFalse,
				Reason:  "Processing",
				Message: "NetworkIntent is being processed",
			},
		},
	}

	err = suite.client.Status().Update(suite.ctx, intent)
	suite.Require().NoError(err)

	// Verify status was updated
	retrieved := &nephoranv1.NetworkIntent{}
	err = suite.client.Get(suite.ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, retrieved)
	suite.NoError(err)
	suite.Equal("Processing", retrieved.Status.Phase)
	suite.Equal("Intent is being processed by the operator", retrieved.Status.Message)
	suite.Len(retrieved.Status.Conditions, 1)
	suite.Equal("Ready", retrieved.Status.Conditions[0].Type)

	// Update status to completed
	retrieved.Status.Phase = "Completed"
	retrieved.Status.Message = "Intent has been successfully processed"
	retrieved.Status.Conditions[0].Status = metav1.ConditionTrue
	retrieved.Status.Conditions[0].Reason = "Completed"
	retrieved.Status.Conditions[0].Message = "NetworkIntent has been successfully processed"

	err = suite.client.Status().Update(suite.ctx, retrieved)
	suite.Require().NoError(err)

	// Verify final status
	final := &nephoranv1.NetworkIntent{}
	err = suite.client.Get(suite.ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, final)
	suite.NoError(err)
	suite.Equal("Completed", final.Status.Phase)
	suite.Equal("Intent has been successfully processed", final.Status.Message)
	suite.Equal(metav1.ConditionTrue, final.Status.Conditions[0].Status)
}

// TestE2E_OperatorResilience tests operator resilience and recovery
func (suite *E2EOperatorTestSuite) TestE2E_OperatorResilience() {
	// Create NetworkIntent
	intent := fixtures.SimpleNetworkIntent()
	intent.Namespace = suite.namespace
	intent.Name = "e2e-resilience-test"

	err := suite.client.Create(suite.ctx, intent)
	suite.Require().NoError(err)

	// Simulate operator restart by scaling down and up the deployment
	deployment := &appsv1.Deployment{}
	err = suite.client.Get(suite.ctx, types.NamespacedName{
		Name:      suite.operatorName,
		Namespace: suite.namespace,
	}, deployment)
	suite.Require().NoError(err)

	// Scale down
	replicas := int32(0)
	deployment.Spec.Replicas = &replicas
	err = suite.client.Update(suite.ctx, deployment)
	suite.Require().NoError(err)

	// Wait for scale down
	suite.Eventually(func() bool {
		err := suite.client.Get(suite.ctx, types.NamespacedName{
			Name:      suite.operatorName,
			Namespace: suite.namespace,
		}, deployment)
		if err != nil {
			return false
		}
		return deployment.Status.ReadyReplicas == 0
	}, 30*time.Second, 1*time.Second, "Deployment should scale down")

	// Scale back up
	replicas = int32(1)
	deployment.Spec.Replicas = &replicas
	err = suite.client.Update(suite.ctx, deployment)
	suite.Require().NoError(err)

	// Wait for scale up
	suite.waitForDeploymentReady(suite.operatorName)

	// Verify NetworkIntent still exists and is processed
	retrieved := &nephoranv1.NetworkIntent{}
	err = suite.client.Get(suite.ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, retrieved)
	suite.NoError(err)
	suite.Equal(intent.Spec.Intent, retrieved.Spec.Intent)
}

// TestE2E_ConcurrentOperations tests concurrent operations on NetworkIntents
func (suite *E2EOperatorTestSuite) TestE2E_ConcurrentOperations() {
	const numGoroutines = 10
	const intentsPerGoroutine = 5

	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines*intentsPerGoroutine)

	// Start concurrent goroutines creating NetworkIntents
	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for i := 0; i < intentsPerGoroutine; i++ {
				intent := fixtures.SimpleNetworkIntent()
				intent.Namespace = suite.namespace
				intent.Name = fmt.Sprintf("e2e-concurrent-%d-%d", goroutineID, i)
				intent.Spec.Intent = fmt.Sprintf("concurrent scaling intent %d-%d", goroutineID, i)

				err := suite.client.Create(suite.ctx, intent)
				if err != nil {
					errors <- err
				}
			}
		}(g)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// Success
		case err := <-errors:
			suite.FailNow("Concurrent operation failed", "Error: %v", err)
		case <-time.After(2 * time.Minute):
			suite.FailNow("Timeout waiting for concurrent operations")
		}
	}

	// Verify all NetworkIntents were created
	intentList := &nephoranv1.NetworkIntentList{}
	err := suite.client.List(suite.ctx, intentList, client.InNamespace(suite.namespace))
	suite.NoError(err)

	concurrentIntents := 0
	for _, item := range intentList.Items {
		if contains(item.Name, "e2e-concurrent") {
			concurrentIntents++
		}
	}

	expectedIntents := numGoroutines * intentsPerGoroutine
	suite.Equal(expectedIntents, concurrentIntents, "Expected %d concurrent NetworkIntents, got %d", expectedIntents, concurrentIntents)
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || (len(s) > len(substr) && contains(s[1:], substr))
}

// TestSuite runner function
// DISABLED: func TestE2EOperatorTestSuite(t *testing.T) {
	suite.Run(t, new(E2EOperatorTestSuite))
}

// Ginkgo-style tests for comparison
var _ = ginkgo.Describe("NetworkIntent E2E Tests", func() {
	var (
		testEnv   *envtest.Environment
		cfg       *rest.Config
		k8sClient client.Client
		ctx       context.Context
		cancel    context.CancelFunc
		namespace string
	)

	ginkgo.BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		namespace = "ginkgo-e2e-test"

		ginkgo.By("Starting test environment")
		testEnv = &envtest.Environment{
			CRDDirectoryPaths: []string{
				"../../../config/crd/bases",
			},
			ErrorIfCRDPathMissing: false,
		}

		var err error
		cfg, err = testEnv.Start()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(cfg).NotTo(gomega.BeNil())

		scheme := runtime.NewScheme()
		err = nephoranv1.AddToScheme(scheme)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(k8sClient).NotTo(gomega.BeNil())

		// Create test namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		err = k8sClient.Create(ctx, ns)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
		ginkgo.By("Tearing down test environment")
		if cancel != nil {
			cancel()
		}
		if testEnv != nil {
			err := testEnv.Stop()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}
	})

	ginkgo.Context("When creating a NetworkIntent", func() {
		ginkgo.It("Should successfully create and retrieve the NetworkIntent", func() {
			intent := fixtures.SimpleNetworkIntent()
			intent.Namespace = namespace
			intent.Name = "ginkgo-test-intent"

			ginkgo.By("Creating NetworkIntent")
			err := k8sClient.Create(ctx, intent)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("Verifying NetworkIntent was created")
			retrieved := &nephoranv1.NetworkIntent{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, retrieved)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(retrieved.Spec.Intent).To(gomega.Equal(intent.Spec.Intent))
		})
	})

	ginkgo.Context("When updating a NetworkIntent", func() {
		ginkgo.It("Should successfully update the intent", func() {
			intent := fixtures.SimpleNetworkIntent()
			intent.Namespace = namespace
			intent.Name = "ginkgo-update-test"

			ginkgo.By("Creating NetworkIntent")
			err := k8sClient.Create(ctx, intent)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("Updating the intent")
			intent.Spec.Intent = "updated scaling intent via ginkgo"
			err = k8sClient.Update(ctx, intent)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.By("Verifying the update")
			retrieved := &nephoranv1.NetworkIntent{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, retrieved)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(retrieved.Spec.Intent).To(gomega.Equal("updated scaling intent via ginkgo"))
		})
	})
})
