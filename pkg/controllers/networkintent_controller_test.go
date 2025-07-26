package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	nephoranv1alpha1 "github.com/nephoran/nephoran-intent-operator/api/v1alpha1"
	"github.com/nephoran/nephoran-intent-operator/pkg/git"
)

// MockGitClient is a mock implementation of the GitClient for testing.
type MockGitClient struct {
	CommitAndPushFunc func(ctx context.Context, commitMessage string, modify func(repoPath string) error) (string, error)
}

func (m *MockGitClient) CommitAndPush(ctx context.Context, commitMessage string, modify func(repoPath string) error) (string, error) {
	if m.CommitAndPushFunc != nil {
		return m.CommitAndPushFunc(ctx, commitMessage, modify)
	}
	return "mock-commit-hash", nil
}

var _ git.ClientInterface = &MockGitClient{} // Verify that MockGitClient implements the interface.

var _ = Describe("NetworkIntent Controller", func() {
	const (
		IntentName      = "test-intent-for-real"
		IntentNamespace = "default"
	)

	Context("When reconciling a NetworkIntent", func() {
		It("Should process a deployment intent correctly", func() {
			// 1. Setup a mock HTTP server to simulate the LLM Processor
			mockLLMServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				response := map[string]interface{}{
					"type":      "NetworkFunctionDeployment",
					"name":      "test-nf-from-llm",
					"namespace": "llm-ns",
				}
				Expect(json.NewEncoder(w).Encode(response)).To(Succeed())
			}))
			defer mockLLMServer.Close()

			// 2. Setup the reconciler with the mock client and server
			reconciler := &NetworkIntentReconciler{
				Client:          k8sClient,
				Scheme:          k8sClient.Scheme(),
				GitClient:       &MockGitClient{},
				LLMProcessorURL: mockLLMServer.URL,
				HTTPClient:      mockLLMServer.Client(),
			}

			// 3. Create the NetworkIntent resource
			intent := &nephoranv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      IntentName,
					Namespace: IntentNamespace,
				},
				Spec: nephoranv1alpha1.NetworkIntentSpec{
					Intent: "Deploy a new network function please",
				},
			}
			Expect(k8sClient.Create(ctx, intent)).Should(Succeed())

			// 4. Trigger the reconciliation
			req := ctrl.Request{
				NamespacedName: types.NamespacedName{Name: IntentName, Namespace: IntentNamespace},
			}
			_, err := reconciler.Reconcile(context.Background(), req)
			Expect(err).NotTo(HaveOccurred())

			// 5. Verify the status of the NetworkIntent resource
			updatedIntent := &nephoranv1alpha1.NetworkIntent{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, req.NamespacedName, updatedIntent)
				if err != nil {
					return false
				}
				return meta.IsStatusConditionTrue(updatedIntent.Status.Conditions, "Processed")
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())
		})
	})
})
