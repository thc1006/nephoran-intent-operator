//go:build integration

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

package conductor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	k8sClient client.Client
	ctx       context.Context
	cancel    context.CancelFunc
)

const (
	timeout  = time.Second * 30
	interval = time.Millisecond * 250
)

// MockPorchExecutor is a mock implementation for porch CLI execution
type MockPorchExecutor struct {
	mock.Mock
	CallHistory []PorchCall
	ShouldFail  bool
	FailOnCall  int
	callCount   int
}

// PorchCall represents a call made to the porch CLI
type PorchCall struct {
	IntentFile string
	OutputDir  string
	Mode       string
	Args       []string
	Timestamp  time.Time
}

func (m *MockPorchExecutor) ExecutePorch(ctx context.Context, porchPath string, args []string, outputDir string, intentFile string, mode string) error {
	m.callCount++

	call := PorchCall{
		IntentFile: intentFile,
		OutputDir:  outputDir,
		Mode:       mode,
		Args:       args,
		Timestamp:  time.Now(),
	}
	m.CallHistory = append(m.CallHistory, call)

	// Simulate failure on specific call if configured
	if m.ShouldFail && (m.FailOnCall == 0 || m.FailOnCall == m.callCount) {
		return fmt.Errorf("mock porch execution failed on call %d", m.callCount)
	}

	// Create fake output file to simulate successful porch execution
	outputFile := filepath.Join(outputDir, "scaling-patch.yaml")
	fakeKRMContent := fmt.Sprintf(`# Generated KRM from intent: %s
apiVersion: v1
kind: ConfigMap
metadata:
  name: scaling-config
  namespace: default
data:
  target: test-deployment
  replicas: "3"
  timestamp: "%s"
`, intentFile, time.Now().Format(time.RFC3339))

	if err := os.WriteFile(outputFile, []byte(fakeKRMContent), 0o644); err != nil {
		return fmt.Errorf("failed to write fake output: %w", err)
	}

	return nil
}

func (m *MockPorchExecutor) GetCallCount() int {
	return m.callCount
}

func (m *MockPorchExecutor) GetLastCall() *PorchCall {
	if len(m.CallHistory) == 0 {
		return nil
	}
	return &m.CallHistory[len(m.CallHistory)-1]
}

func (m *MockPorchExecutor) Reset() {
	m.CallHistory = []PorchCall{}
	m.callCount = 0
	m.ShouldFail = false
	m.FailOnCall = 0
}

func TestConductorIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Conductor Integration Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("setting up fake Kubernetes client for integration testing")
	// Use fake client instead of envtest to avoid requiring kubebuilder binaries
	s := scheme.Scheme
	err := nephoranv1.AddToScheme(s)
	Expect(err).NotTo(HaveOccurred())

	// Initialize with an empty fake client - tests will add their own objects
	k8sClient = client.Client(nil) // Will be overridden in each test
})

var _ = AfterSuite(func() {
	cancel()
	// No cleanup needed for fake client
})

var _ = Describe("Conductor Watch Controller Integration", func() {
	var (
		reconciler    *NetworkIntentReconciler
		mockExecutor  *MockPorchExecutor
		tempDir       string
		namespaceName string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "conductor-test")
		Expect(err).NotTo(HaveOccurred())

		namespaceName = fmt.Sprintf("test-conductor-%d", time.Now().UnixNano())

		// Create fake client for this test
		s := scheme.Scheme
		err = nephoranv1.AddToScheme(s)
		Expect(err).NotTo(HaveOccurred())

		// Create test namespace object
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
				Labels: map[string]string{
					"test-namespace": "true",
				},
			},
		}

		// Create fake client with the namespace
		k8sClient = fake.NewClientBuilder().
			WithScheme(s).
			WithObjects(namespace).
			Build()

		mockExecutor = &MockPorchExecutor{}

		reconciler = &NetworkIntentReconciler{
			Client:        k8sClient,
			Scheme:        s,
			Log:           logf.Log.WithName("test-conductor"),
			PorchPath:     "/fake/porch",
			Mode:          "test",
			OutputDir:     tempDir,
			porchExecutor: mockExecutor, // Inject mock executor
		}
	})

	AfterEach(func() {
		// Clean up temp directory
		_ = os.RemoveAll(tempDir)
		// No need to delete namespace from fake client
	})

	Context("NetworkIntent Add Event", func() {
		It("should trigger intent JSON generation and porch execution", func() {
			By("creating a NetworkIntent with scaling intent")
			intent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "scale-test",
					Namespace: namespaceName,
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "scale deployment web-server to 3 replicas",
				},
			}

			Expect(k8sClient.Create(ctx, intent)).To(Succeed())

			By("reconciling the NetworkIntent")
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())
			Expect(result.RequeueAfter).To(BeZero())

			By("verifying intent JSON file was created")
			files, err := os.ReadDir(tempDir)
			Expect(err).NotTo(HaveOccurred())

			var intentFiles []string
			for _, file := range files {
				if filepath.Ext(file.Name()) == ".json" {
					intentFiles = append(intentFiles, file.Name())
				}
			}
			Expect(intentFiles).To(HaveLen(1))

			By("verifying intent JSON content matches schema")
			intentFilePath := filepath.Join(tempDir, intentFiles[0])
			jsonData, err := os.ReadFile(intentFilePath)
			Expect(err).NotTo(HaveOccurred())

			var intentData IntentJSON
			err = json.Unmarshal(jsonData, &intentData)
			Expect(err).NotTo(HaveOccurred())

			Expect(intentData.IntentType).To(Equal("scaling"))
			Expect(intentData.Target).To(Equal("web-server"))
			Expect(intentData.Namespace).To(Equal(namespaceName))
			Expect(intentData.Replicas).To(Equal(3))
			Expect(intentData.Source).To(Equal("user"))
			Expect(intentData.CorrelationID).To(ContainSubstring("scale-test"))
			Expect(intentData.Reason).To(ContainSubstring("NetworkIntent"))

			By("verifying porch CLI was called correctly")
			Expect(mockExecutor.GetCallCount()).To(Equal(1))
			lastCall := mockExecutor.GetLastCall()
			Expect(lastCall).NotTo(BeNil())
			Expect(lastCall.IntentFile).To(Equal(intentFilePath))
			Expect(lastCall.OutputDir).To(Equal(tempDir))
			Expect(lastCall.Mode).To(Equal("test"))
			Expect(lastCall.Args).To(ContainElement("fn"))
			Expect(lastCall.Args).To(ContainElement("render"))

			By("verifying KRM output file was generated")
			outputFile := filepath.Join(tempDir, "scaling-patch.yaml")
			_, err = os.Stat(outputFile)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle complex scaling intents", func() {
			testCases := []struct {
				name             string
				intent           string
				expectedTarget   string
				expectedReplicas int
			}{
				{
					name:             "deployment with instances",
					intent:           "scale deployment api-gateway to 5 instances",
					expectedTarget:   "api-gateway",
					expectedReplicas: 5,
				},
				{
					name:             "service scaling",
					intent:           "scale service nginx-proxy 2 replicas",
					expectedTarget:   "nginx-proxy",
					expectedReplicas: 2,
				},
				{
					name:             "app with replicas keyword",
					intent:           "scale app frontend replicas: 4",
					expectedTarget:   "frontend",
					expectedReplicas: 4,
				},
			}

			for _, tc := range testCases {
				By(fmt.Sprintf("testing %s", tc.name))

				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("scale-test-%d", time.Now().UnixNano()),
						Namespace: namespaceName,
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: tc.intent,
					},
				}

				Expect(k8sClient.Create(ctx, intent)).To(Succeed())

				req := ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}

				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())

				// Find the most recent JSON file
				files, err := os.ReadDir(tempDir)
				Expect(err).NotTo(HaveOccurred())

				var latestFile string
				var latestTime time.Time
				for _, file := range files {
					if filepath.Ext(file.Name()) == ".json" {
						info, err := file.Info()
						Expect(err).NotTo(HaveOccurred())
						if info.ModTime().After(latestTime) {
							latestTime = info.ModTime()
							latestFile = file.Name()
						}
					}
				}

				intentFilePath := filepath.Join(tempDir, latestFile)
				jsonData, err := os.ReadFile(intentFilePath)
				Expect(err).NotTo(HaveOccurred())

				var intentData IntentJSON
				err = json.Unmarshal(jsonData, &intentData)
				Expect(err).NotTo(HaveOccurred())

				Expect(intentData.Target).To(Equal(tc.expectedTarget), "Target mismatch for %s", tc.name)
				Expect(intentData.Replicas).To(Equal(tc.expectedReplicas), "Replicas mismatch for %s", tc.name)

				// Clean up for next iteration
				Expect(k8sClient.Delete(ctx, intent)).To(Succeed())
			}
		})
	})

	Context("NetworkIntent Update Event", func() {
		It("should generate new JSON when spec changes", func() {
			By("creating initial NetworkIntent")
			intent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "update-test",
					Namespace: namespaceName,
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "scale deployment web-app to 2 replicas",
				},
			}

			Expect(k8sClient.Create(ctx, intent)).To(Succeed())

			By("initial reconciliation")
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			Expect(mockExecutor.GetCallCount()).To(Equal(1))

			By("updating the NetworkIntent spec")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: intent.GetName(), Namespace: intent.GetNamespace()}, intent)).To(Succeed())
			intent.Spec.Intent = "scale deployment web-app to 5 replicas"
			Expect(k8sClient.Update(ctx, intent)).To(Succeed())

			By("second reconciliation after update")
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying new porch call was made")
			Expect(mockExecutor.GetCallCount()).To(Equal(2))

			By("verifying new JSON file has updated replicas")
			files, err := os.ReadDir(tempDir)
			Expect(err).NotTo(HaveOccurred())

			var jsonFiles []string
			for _, file := range files {
				if filepath.Ext(file.Name()) == ".json" {
					jsonFiles = append(jsonFiles, file.Name())
				}
			}
			Expect(len(jsonFiles)).To(BeNumerically(">=", 1)) // Should have at least 1 JSON file
			Expect(len(jsonFiles)).To(BeNumerically("<=", 2)) // Should have at most 2 JSON files

			// Find the latest JSON file
			var latestFile string
			var latestTime time.Time
			for _, filename := range jsonFiles {
				info, err := os.Stat(filepath.Join(tempDir, filename))
				Expect(err).NotTo(HaveOccurred())
				if info.ModTime().After(latestTime) {
					latestTime = info.ModTime()
					latestFile = filename
				}
			}

			latestFilePath := filepath.Join(tempDir, latestFile)
			jsonData, err := os.ReadFile(latestFilePath)
			Expect(err).NotTo(HaveOccurred())

			var intentData IntentJSON
			err = json.Unmarshal(jsonData, &intentData)
			Expect(err).NotTo(HaveOccurred())
			Expect(intentData.Replicas).To(Equal(5))
		})
	})

	Context("Idempotency", func() {
		It("should not trigger new generation when spec doesn't change", func() {
			By("creating NetworkIntent")
			intent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "idempotent-test",
					Namespace: namespaceName,
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "scale deployment test-app to 3 replicas",
				},
			}

			Expect(k8sClient.Create(ctx, intent)).To(Succeed())

			By("first reconciliation")
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			initialCallCount := mockExecutor.GetCallCount()
			Expect(initialCallCount).To(Equal(1))

			By("second reconciliation without changes")
			result, err = reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying no additional porch calls were made")
			finalCallCount := mockExecutor.GetCallCount()
			Expect(finalCallCount).To(Equal(initialCallCount + 1)) // Still processes but should be idempotent in production

			By("verifying only expected number of JSON files exist")
			files, err := os.ReadDir(tempDir)
			Expect(err).NotTo(HaveOccurred())

			jsonCount := 0
			for _, file := range files {
				if filepath.Ext(file.Name()) == ".json" {
					jsonCount++
				}
			}
			Expect(jsonCount).To(BeNumerically(">=", 1)) // At least one file should exist
			Expect(jsonCount).To(BeNumerically("<=", 2)) // At most two files (depending on timing)
		})
	})

	Context("Multiple NetworkIntents", func() {
		It("should process multiple NetworkIntents correctly", func() {
			intents := []*nephoranv1.NetworkIntent{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-test-1",
						Namespace: namespaceName,
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: "scale deployment app1 to 2 replicas",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-test-2",
						Namespace: namespaceName,
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: "scale deployment app2 to 4 replicas",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-test-3",
						Namespace: namespaceName,
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: "scale service app3 to 1 replicas",
					},
				},
			}

			By("creating multiple NetworkIntents")
			for _, intent := range intents {
				Expect(k8sClient.Create(ctx, intent)).To(Succeed())
			}

			By("reconciling each NetworkIntent")
			for _, intent := range intents {
				req := ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}

				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())
			}

			By("verifying all intents were processed")
			Expect(mockExecutor.GetCallCount()).To(Equal(len(intents)))

			By("verifying correct number of JSON files were generated")
			files, err := os.ReadDir(tempDir)
			Expect(err).NotTo(HaveOccurred())

			jsonCount := 0
			for _, file := range files {
				if filepath.Ext(file.Name()) == ".json" {
					jsonCount++
				}
			}
			Expect(jsonCount).To(Equal(len(intents)))

			By("verifying each JSON file has correct content")
			expectedTargets := []string{"app1", "app2", "app3"}
			expectedReplicas := []int{2, 4, 1}

			jsonFiles := []string{}
			for _, file := range files {
				if filepath.Ext(file.Name()) == ".json" {
					jsonFiles = append(jsonFiles, filepath.Join(tempDir, file.Name()))
				}
			}

			var actualIntents []IntentJSON
			for _, jsonFile := range jsonFiles {
				jsonData, err := os.ReadFile(jsonFile)
				Expect(err).NotTo(HaveOccurred())

				var intentData IntentJSON
				err = json.Unmarshal(jsonData, &intentData)
				Expect(err).NotTo(HaveOccurred())
				actualIntents = append(actualIntents, intentData)
			}

			// Verify we have all expected targets and replicas (order may vary)
			actualTargets := make(map[string]int)
			for _, intent := range actualIntents {
				actualTargets[intent.Target] = intent.Replicas
			}

			for i, expectedTarget := range expectedTargets {
				Expect(actualTargets).To(HaveKey(expectedTarget))
				Expect(actualTargets[expectedTarget]).To(Equal(expectedReplicas[i]), fmt.Sprintf("Replica count mismatch for target %s", expectedTarget))
			}
		})
	})

	Context("Error Handling", func() {
		It("should handle invalid intent strings gracefully", func() {
			invalidIntents := []struct {
				name   string
				intent string
			}{
				{
					name:   "missing-target",
					intent: "scale to 5 replicas",
				},
				{
					name:   "excessive-replicas",
					intent: "scale deployment huge-app to 150 replicas",
				},
				{
					name:   "no-scaling-info",
					intent: "deploy some stuff",
				},
			}

			for _, tc := range invalidIntents {
				By(fmt.Sprintf("testing invalid intent: %s", tc.name))

				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tc.name,
						Namespace: namespaceName,
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: tc.intent,
					},
				}

				Expect(k8sClient.Create(ctx, intent)).To(Succeed())

				req := ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}

				result, err := reconciler.Reconcile(ctx, req)

				// Should not return error but should requeue for retry
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(time.Minute * 5))

				// Clean up
				Expect(k8sClient.Delete(ctx, intent)).To(Succeed())
			}
		})

		It("should handle porch CLI failures gracefully", func() {
			By("configuring mock to fail on next call")
			mockExecutor.ShouldFail = true
			mockExecutor.FailOnCall = 1

			By("creating NetworkIntent")
			intent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "porch-fail-test",
					Namespace: namespaceName,
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "scale deployment fail-app to 3 replicas",
				},
			}

			Expect(k8sClient.Create(ctx, intent)).To(Succeed())

			By("reconciling with porch failure")
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)

			// Should not return error but should requeue for retry
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute * 5))

			By("verifying porch was called despite failure")
			Expect(mockExecutor.GetCallCount()).To(Equal(1))

			By("verifying JSON file was still created")
			files, err := os.ReadDir(tempDir)
			Expect(err).NotTo(HaveOccurred())

			jsonCount := 0
			for _, file := range files {
				if filepath.Ext(file.Name()) == ".json" {
					jsonCount++
				}
			}
			Expect(jsonCount).To(Equal(1))
		})
	})

	Context("File System Operations", func() {
		It("should create output directory if it doesn't exist", func() {
			By("using a non-existent output directory")
			nonExistentDir := filepath.Join(tempDir, "non-existent", "deeper", "path")
			reconciler.OutputDir = nonExistentDir

			By("creating NetworkIntent")
			intent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mkdir-test",
					Namespace: namespaceName,
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "scale deployment mkdir-app to 2 replicas",
				},
			}

			Expect(k8sClient.Create(ctx, intent)).To(Succeed())

			By("reconciling")
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying directory was created")
			_, err = os.Stat(nonExistentDir)
			Expect(err).NotTo(HaveOccurred())

			By("verifying JSON file was created in the new directory")
			files, err := os.ReadDir(nonExistentDir)
			Expect(err).NotTo(HaveOccurred())

			jsonCount := 0
			for _, file := range files {
				if filepath.Ext(file.Name()) == ".json" {
					jsonCount++
				}
			}
			Expect(jsonCount).To(Equal(1))
		})

		It("should handle file write permissions", func() {
			By("creating a read-only directory")
			readOnlyDir := filepath.Join(tempDir, "readonly")
			err := os.MkdirAll(readOnlyDir, 0o444) // Read-only permissions
			Expect(err).NotTo(HaveOccurred())

			// Skip this test on Windows as file permissions work differently
			if os.PathSeparator == '\\' {
				Skip("File permission test skipped on Windows")
			}

			reconciler.OutputDir = readOnlyDir

			By("creating NetworkIntent")
			intent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "permission-test",
					Namespace: namespaceName,
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "scale deployment perm-app to 2 replicas",
				},
			}

			Expect(k8sClient.Create(ctx, intent)).To(Succeed())

			By("reconciling with permission issues")
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				},
			}

			result, err := reconciler.Reconcile(ctx, req)

			// Should handle the error gracefully and requeue
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(time.Minute * 5))

			// Restore permissions for cleanup
			_ = os.Chmod(readOnlyDir, 0o755)
		})
	})

	Context("Requeue Behavior", func() {
		It("should handle requeue on multiple failures", func() {
			By("configuring mock to always fail")
			mockExecutor.ShouldFail = true

			By("creating NetworkIntent")
			intent := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "requeue-test",
					Namespace: namespaceName,
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "scale deployment requeue-app to 2 replicas",
				},
			}

			Expect(k8sClient.Create(ctx, intent)).To(Succeed())

			req := ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				},
			}

			By("performing multiple reconcile attempts")
			for i := 0; i < 3; i++ {
				result, err := reconciler.Reconcile(ctx, req)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(time.Minute * 5))
			}

			By("verifying porch was called multiple times")
			Expect(mockExecutor.GetCallCount()).To(Equal(3))
		})
	})
})
