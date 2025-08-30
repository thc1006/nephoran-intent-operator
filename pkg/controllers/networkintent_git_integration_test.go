package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
	gitclient "github.com/nephio-project/nephoran-intent-operator/pkg/git"
)

var _ = Describe("NetworkIntent Controller (GitOps Integration)", func() {
	var gitRepo *git.Repository
	var repoPath string
	var sshKey ssh.AuthMethod

	BeforeEach(func() {
		// 1. Set up a bare git repository to act as the remote
		var err error
		repoPath, err = os.MkdirTemp("", "nephio-git-test-remote")
		Expect(err).NotTo(HaveOccurred())
		_, err = git.PlainInit(repoPath, true)
		Expect(err).NotTo(HaveOccurred())

		// 2. Generate a dummy SSH key for the test
		// In a real test, you might generate this dynamically
		privateKeyBytes := []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAaAAAABNlY2RzYS\n1zaGEyLW5pc3RwMjU2AAAACG5pc3RwMjU2AAAAQQR9r3vGZ2l4aC9sY2ltaW5nIGZyb20g\ndGhlIGNsaWVudCBpcyBub3Qgc28gZWFzeS4uLgAAAHEEfa97xmdpeGgvZGNpbWluZyBmcm9t\nIHRoZSBjbGllbnQgaXMgdGhlIGJlc3QuLi4=\n-----END OPENSSH PRIVATE KEY-----")
		sshKey, err = ssh.NewPublicKeys("git", privateKeyBytes, "")
		Expect(err).NotTo(HaveOccurred())

		// 3. Initialize the reconciler's GitClient
		k8sReconciler.GitClient = gitclient.NewClient(fmt.Sprintf("file://%s", repoPath), "main", string(privateKeyBytes))
	})

	AfterEach(func() {
		os.RemoveAll(repoPath)
	})

	It("Should commit KRM to a Git repository for deployment and scaling intents", func() {
		ctx := context.Background()
		intentName := "gitops-test-intent"
		nfName := "upf-gitops"
		namespace := "default"

		// === Test 1: Create a new deployment intent ===
		By("Creating a NetworkIntent for a new deployment")
		deploymentIntent := &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{Name: intentName, Namespace: namespace},
			Spec: nephoranv1.NetworkIntentSpec{
				Parameters: k8sruntime.RawExtension{
					Raw: mustMarshal(NetworkFunctionDeploymentIntent{
						Type: "NetworkFunctionDeployment", Name: nfName, Namespace: namespace,
						Spec: NetworkFunctionDeploymentSpec{Replicas: 1, Image: "upf:v1"},
					}),
				},
			},
		}
		Expect(k8sClient.Create(ctx, deploymentIntent)).Should(Succeed())

		// === Verification 1: Check the Git repository ===
		By("Verifying the new package was committed to Git")
		Eventually(func() bool {
			clonePath, _ := os.MkdirTemp("", "nephio-git-test-clone")
			defer os.RemoveAll(clonePath)
			_, err := git.PlainClone(clonePath, false, &git.CloneOptions{URL: repoPath})
			if err != nil {
				return false
			}

			deploymentYAML, err := os.ReadFile(filepath.Join(clonePath, namespace, nfName, "deployment.yaml"))
			if err != nil {
				return false
			}
			var deployment appsv1.Deployment
			yaml.Unmarshal(deploymentYAML, &deployment)
			return *deployment.Spec.Replicas == 1
		}, "10s", "250ms").Should(BeTrue())

		// === Test 2: Update the intent to scale the deployment ===
		By("Updating the NetworkIntent to scale the deployment")
		var updatedIntent nephoranv1.NetworkIntent
		Expect(k8sClient.Get(ctx, k8sclient.ObjectKey{Name: intentName, Namespace: namespace}, &updatedIntent)).Should(Succeed())
		updatedIntent.Spec.Intent = "Scale UPF to 3 replicas"
		updatedIntent.Spec.Parameters = k8sruntime.RawExtension{
			Raw: mustMarshal(NetworkFunctionScaleIntent{
				Type: "NetworkFunctionScale", Name: nfName, Namespace: namespace, Replicas: 3,
			}),
		}
		Expect(k8sClient.Update(ctx, &updatedIntent)).Should(Succeed())

		// === Trigger reconciliation again for the updated intent ===
		By("Triggering reconciliation of the updated NetworkIntent")
		result, err = reconciler.Reconcile(ctx, req)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Requeue).To(BeFalse())

		// === Verification 2: Check the Git repository was updated ===
		By("Verifying the updated package was committed to Git")
		Eventually(func() bool {
			clonePath, _ := os.MkdirTemp("", "nephio-git-test-clone")
			defer os.RemoveAll(clonePath)
			_, err := git.PlainClone(clonePath, false, &git.CloneOptions{URL: repoPath})
			if err != nil {
				return false
			}

			// Check for updated ConfigMap file with scale parameters
			configMapPath := filepath.Join(clonePath, "networkintents", namespace, fmt.Sprintf("%s-configmap.json", intentName))
			configMapData, err := os.ReadFile(configMapPath)
			if err != nil {
				return false
			}

			// Parse the ConfigMap to check if it contains scale parameters
			var configMap map[string]interface{}
			if err := json.Unmarshal(configMapData, &configMap); err != nil {
				return false
			}

			// Check if the data contains scaling information
			data, ok := configMap["data"].(map[string]interface{})
			if !ok {
				return false
			}

			// Look for replicas = 3 in the data
			if replicasFloat, exists := data["replicas"].(float64); exists {
				return int(replicasFloat) == 3
			}
			return false
		}, "15s", "500ms").Should(BeTrue())
	})
})
