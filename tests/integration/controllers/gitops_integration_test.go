//go:build integration

package integration_tests_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

var _ = Describe("GitOps Integration Tests", func() {
	var (
		namespace         *corev1.Namespace
		testCtx           context.Context
		mockGitServer     *httptest.Server
		mockNephioServer  *httptest.Server
		gitRequestTracker *GitRequestTracker
		tempRepoDir       string
	)

	BeforeEach(func() {
		namespace = CreateTestNamespace()
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, 8*time.Minute)
		DeferCleanup(cancel)

		// Create temporary directory for Git repository simulation
		var err error
		tempRepoDir, err = ioutil.TempDir("", "nephoran-git-test-*")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			os.RemoveAll(tempRepoDir)
		})

		// Initialize request tracker
		gitRequestTracker = NewGitRequestTracker(tempRepoDir)

		// Setup mock Git server
		mockGitServer = setupMockGitServer(gitRequestTracker)
		DeferCleanup(mockGitServer.Close)

		// Setup mock Nephio Porch server
		mockNephioServer = setupMockNephioServer(gitRequestTracker)
		DeferCleanup(mockNephioServer.Close)
	})

	Describe("Nephio Package Generation and Deployment", func() {
		Context("when NetworkIntent requires GitOps deployment", func() {
			It("should generate Nephio package and commit to Git repository", func() {
				By("creating NetworkIntent that triggers package generation")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "gitops-package-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/gitops-enabled":  "true",
							"nephoran.com/git-repo-url":    mockGitServer.URL + "/repo.git",
							"nephoran.com/nephio-endpoint": mockNephioServer.URL,
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy production AMF with GitOps workflow using Nephio packages",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityHigh,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentAMF,
						},
						TargetNamespace: "production",
						TargetCluster:   "main-cluster",
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for controller to process the intent")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 90*time.Second, 3*time.Second).Should(Or(
					Equal("Processing"),
					Equal("Deploying"),
					Equal("Ready"),
					Equal("Failed"),
				))

				By("verifying Git repository operations were performed")
				Eventually(func() int {
					return gitRequestTracker.GetRequestCount("git-clone")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				Eventually(func() int {
					return gitRequestTracker.GetRequestCount("git-commit")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("verifying Nephio package generation was called")
				Eventually(func() int {
					return gitRequestTracker.GetRequestCount("nephio-package-gen")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))

				By("checking generated package content")
				if createdIntent.Status.GitCommitHash != "" {
					packageFiles := gitRequestTracker.GetGeneratedFiles()
					Expect(packageFiles).To(HaveKey("Kptfile"))
					Expect(packageFiles).To(HaveKey("deployment.yaml"))

					// Verify Kptfile structure
					var kptfile map[string]interface{}
					err := yaml.Unmarshal([]byte(packageFiles["Kptfile"]), &kptfile)
					Expect(err).NotTo(HaveOccurred())
					Expect(kptfile["apiVersion"]).To(Equal("kpt.dev/v1"))
					Expect(kptfile["kind"]).To(Equal("Kptfile"))
				}
			})

			It("should handle package generation failures with retry logic", func() {
				By("configuring Nephio server to return errors initially")
				gitRequestTracker.SetErrorMode("nephio-package-gen", true, 2) // Fail first 2 attempts

				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "retry-package-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/gitops-enabled":  "true",
							"nephoran.com/git-repo-url":    mockGitServer.URL + "/repo.git",
							"nephoran.com/nephio-endpoint": mockNephioServer.URL,
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy SMF with package generation retry",
						IntentType: nephoranv1.IntentTypeDeployment,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentSMF,
						},
						MaxRetries: &[]int32{3}[0],
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for retry attempts to be made")
				Eventually(func() int {
					return gitRequestTracker.GetRequestCount("nephio-package-gen")
				}, 120*time.Second, 3*time.Second).Should(BeNumerically(">=", 3))

				By("verifying eventual success after retries")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 150*time.Second, 5*time.Second).Should(Or(
					Equal("Deploying"),
					Equal("Ready"),
				))

				By("verifying Git commit was eventually successful")
				Eventually(func() int {
					return gitRequestTracker.GetRequestCount("git-commit")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">", 0))
			})

			It("should generate different package types for different network functions", func() {
				testCases := []struct {
					name          string
					component     nephoranv1.TargetComponent
					expectedFiles []string
				}{
					{
						name:          "AMF package",
						component:     nephoranv1.TargetComponentAMF,
						expectedFiles: []string{"Kptfile", "deployment.yaml", "service.yaml", "configmap.yaml"},
					},
					{
						name:          "SMF package",
						component:     nephoranv1.TargetComponentSMF,
						expectedFiles: []string{"Kptfile", "deployment.yaml", "service.yaml", "pvc.yaml"},
					},
					{
						name:          "UPF package",
						component:     nephoranv1.TargetComponentUPF,
						expectedFiles: []string{"Kptfile", "deployment.yaml", "service.yaml", "networkpolicy.yaml"},
					},
					{
						name:          "Near-RT RIC package",
						component:     nephoranv1.TargetComponentNearRTRIC,
						expectedFiles: []string{"Kptfile", "deployment.yaml", "service.yaml", "rbac.yaml"},
					},
				}

				for i, tc := range testCases {
					By(fmt.Sprintf("creating %s intent", tc.name))
					intent := &nephoranv1.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("package-type-%d", i),
							Namespace: namespace.Name,
							Annotations: map[string]string{
								"nephoran.com/gitops-enabled":  "true",
								"nephoran.com/git-repo-url":    mockGitServer.URL + "/repo.git",
								"nephoran.com/nephio-endpoint": mockNephioServer.URL,
							},
						},
						Spec: nephoranv1.NetworkIntentSpec{
							Intent:     fmt.Sprintf("Deploy %s with specific package structure", tc.component),
							IntentType: nephoranv1.IntentTypeDeployment,
							TargetComponents: []nephoranv1.TargetComponent{
								tc.component,
							},
						},
					}

					Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

					By(fmt.Sprintf("waiting for %s package to be generated", tc.name))
					Eventually(func() bool {
						files := gitRequestTracker.GetGeneratedFilesForIntent(intent.Name)
						if len(files) == 0 {
							return false
						}

						// Check if all expected files are generated
						for _, expectedFile := range tc.expectedFiles {
							if _, exists := files[expectedFile]; !exists {
								return false
							}
						}
						return true
					}, 90*time.Second, 3*time.Second).Should(BeTrue())

					By(fmt.Sprintf("verifying %s package content", tc.name))
					files := gitRequestTracker.GetGeneratedFilesForIntent(intent.Name)
					for _, expectedFile := range tc.expectedFiles {
						Expect(files).To(HaveKey(expectedFile))
						Expect(files[expectedFile]).NotTo(BeEmpty())
					}
				}
			})
		})
	})

	Describe("GitOps Workflow Verification", func() {
		Context("when tracking deployment status through GitOps", func() {
			It("should track deployment status from Git commit to cluster deployment", func() {
				By("creating NetworkIntent with deployment tracking")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deployment-tracking-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/gitops-enabled":   "true",
							"nephoran.com/git-repo-url":     mockGitServer.URL + "/repo.git",
							"nephoran.com/nephio-endpoint":  mockNephioServer.URL,
							"nephoran.com/track-deployment": "true",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy UPF with deployment status tracking",
						IntentType: nephoranv1.IntentTypeDeployment,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentUPF,
						},
						TargetCluster: "edge-cluster",
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for Git commit phase")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() bool {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return false
					}
					return createdIntent.Status.GitCommitHash != ""
				}, 90*time.Second, 3*time.Second).Should(BeTrue())

				By("verifying deployment timing information is tracked")
				Expect(createdIntent.Status.ProcessingStartTime).NotTo(BeNil())
				if createdIntent.Status.DeploymentStartTime != nil {
					Expect(createdIntent.Status.DeploymentStartTime.Time).To(BeTemporally(">=",
						createdIntent.Status.ProcessingStartTime.Time))
				}

				By("simulating cluster deployment completion")
				gitRequestTracker.SetDeploymentStatus(createdIntent.Status.GitCommitHash, "deployed")

				By("waiting for final deployment status")
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 120*time.Second, 5*time.Second).Should(Or(
					Equal("Deploying"),
					Equal("Ready"),
				))

				By("verifying deployment completion timestamps")
				if createdIntent.Status.DeploymentCompletionTime != nil && createdIntent.Status.DeploymentStartTime != nil {
					Expect(createdIntent.Status.DeploymentCompletionTime.Time).To(BeTemporally(">=",
						createdIntent.Status.DeploymentStartTime.Time))
				}
			})

			It("should handle Git repository conflicts and resolution", func() {
				By("simulating concurrent modifications to Git repository")
				gitRequestTracker.EnableConflictSimulation(true)

				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "conflict-resolution-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/gitops-enabled":  "true",
							"nephoran.com/git-repo-url":    mockGitServer.URL + "/repo.git",
							"nephoran.com/nephio-endpoint": mockNephioServer.URL,
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy NSSF with conflict resolution",
						IntentType: nephoranv1.IntentTypeDeployment,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentNSSF,
						},
						MaxRetries: &[]int32{5}[0], // Allow more retries for conflict resolution
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for conflict resolution attempts")
				Eventually(func() int {
					return gitRequestTracker.GetRequestCount("git-commit")
				}, 150*time.Second, 5*time.Second).Should(BeNumerically(">=", 2))

				By("disabling conflict simulation to allow success")
				gitRequestTracker.EnableConflictSimulation(false)

				By("verifying eventual success after conflict resolution")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 180*time.Second, 5*time.Second).Should(Or(
					Equal("Deploying"),
					Equal("Ready"),
				))

				Expect(createdIntent.Status.GitCommitHash).NotTo(BeEmpty())
			})
		})

		Context("when managing multi-cluster deployments", func() {
			It("should deploy packages to multiple target clusters", func() {
				By("creating NetworkIntent targeting multiple clusters")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-cluster-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/gitops-enabled":  "true",
							"nephoran.com/git-repo-url":    mockGitServer.URL + "/repo.git",
							"nephoran.com/nephio-endpoint": mockNephioServer.URL,
							"nephoran.com/target-clusters": "cluster-1,cluster-2,cluster-3",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy distributed AMF across multiple clusters",
						IntentType: nephoranv1.IntentTypeDeployment,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentAMF,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for multi-cluster package generation")
				Eventually(func() int {
					return gitRequestTracker.GetRequestCount("nephio-package-gen")
				}, 90*time.Second, 3*time.Second).Should(BeNumerically(">=", 3))

				By("verifying packages were created for each target cluster")
				packages := gitRequestTracker.GetGeneratedPackages()
				clusterPackages := 0
				for packageName := range packages {
					if strings.Contains(packageName, "cluster-") {
						clusterPackages++
					}
				}
				Expect(clusterPackages).To(BeNumerically(">=", 3))

				By("verifying deployment status across clusters")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() []nephoranv1.TargetComponent {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return nil
					}
					return createdIntent.Status.DeployedComponents
				}, 120*time.Second, 5*time.Second).Should(ContainElement(nephoranv1.TargetComponentAMF))
			})
		})
	})

	Describe("Package Validation and Quality", func() {
		Context("when validating generated packages", func() {
			It("should generate valid Kubernetes manifests", func() {
				By("creating NetworkIntent with validation requirements")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "validation-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/gitops-enabled":     "true",
							"nephoran.com/git-repo-url":       mockGitServer.URL + "/repo.git",
							"nephoran.com/nephio-endpoint":    mockNephioServer.URL,
							"nephoran.com/validate-manifests": "true",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy validated AMF with comprehensive manifests",
						IntentType: nephoranv1.IntentTypeDeployment,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentAMF,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for package generation with validation")
				Eventually(func() bool {
					return gitRequestTracker.HasGeneratedFiles(intent.Name)
				}, 90*time.Second, 3*time.Second).Should(BeTrue())

				By("validating generated manifest structure")
				files := gitRequestTracker.GetGeneratedFilesForIntent(intent.Name)

				// Validate deployment.yaml
				if deploymentYaml, exists := files["deployment.yaml"]; exists {
					var deployment map[string]interface{}
					err := yaml.Unmarshal([]byte(deploymentYaml), &deployment)
					Expect(err).NotTo(HaveOccurred())

					Expect(deployment["apiVersion"]).To(Equal("apps/v1"))
					Expect(deployment["kind"]).To(Equal("Deployment"))
					Expect(deployment["metadata"]).To(HaveKey("name"))
					Expect(deployment["spec"]).To(HaveKey("replicas"))
					Expect(deployment["spec"]).To(HaveKey("selector"))
					Expect(deployment["spec"]).To(HaveKey("template"))
				}

				// Validate service.yaml
				if serviceYaml, exists := files["service.yaml"]; exists {
					var service map[string]interface{}
					err := yaml.Unmarshal([]byte(serviceYaml), &service)
					Expect(err).NotTo(HaveOccurred())

					Expect(service["apiVersion"]).To(Equal("v1"))
					Expect(service["kind"]).To(Equal("Service"))
					Expect(service["metadata"]).To(HaveKey("name"))
					Expect(service["spec"]).To(HaveKey("selector"))
					Expect(service["spec"]).To(HaveKey("ports"))
				}
			})
		})
	})
})

// GitRequestTracker tracks Git and Nephio operations for testing
type GitRequestTracker struct {
	mu                 sync.RWMutex
	requestCounts      map[string]int
	errorModes         map[string]ErrorConfig
	generatedFiles     map[string]map[string]string // intent-name -> file-name -> content
	generatedPackages  map[string]string
	deploymentStatuses map[string]string // commit-hash -> status
	conflictSimulation bool
	tempDir            string
}

type ErrorConfig struct {
	enabled      bool
	failureCount int
	currentFails int
}

func NewGitRequestTracker(tempDir string) *GitRequestTracker {
	return &GitRequestTracker{
		requestCounts:      make(map[string]int),
		errorModes:         make(map[string]ErrorConfig),
		generatedFiles:     make(map[string]map[string]string),
		generatedPackages:  make(map[string]string),
		deploymentStatuses: make(map[string]string),
		tempDir:            tempDir,
	}
}

func (grt *GitRequestTracker) IncrementRequest(operation string) {
	grt.mu.Lock()
	defer grt.mu.Unlock()
	grt.requestCounts[operation]++
}

func (grt *GitRequestTracker) GetRequestCount(operation string) int {
	grt.mu.RLock()
	defer grt.mu.RUnlock()
	return grt.requestCounts[operation]
}

func (grt *GitRequestTracker) SetErrorMode(operation string, enabled bool, failureCount int) {
	grt.mu.Lock()
	defer grt.mu.Unlock()
	grt.errorModes[operation] = ErrorConfig{
		enabled:      enabled,
		failureCount: failureCount,
		currentFails: 0,
	}
}

func (grt *GitRequestTracker) ShouldReturnError(operation string) bool {
	grt.mu.Lock()
	defer grt.mu.Unlock()

	config, exists := grt.errorModes[operation]
	if !exists || !config.enabled {
		return false
	}

	if config.currentFails < config.failureCount {
		config.currentFails++
		grt.errorModes[operation] = config
		return true
	}

	// Disable error mode after reaching failure count
	config.enabled = false
	grt.errorModes[operation] = config
	return false
}

func (grt *GitRequestTracker) AddGeneratedFiles(intentName string, files map[string]string) {
	grt.mu.Lock()
	defer grt.mu.Unlock()
	grt.generatedFiles[intentName] = files
}

func (grt *GitRequestTracker) GetGeneratedFiles() map[string]string {
	grt.mu.RLock()
	defer grt.mu.RUnlock()

	allFiles := make(map[string]string)
	for _, intentFiles := range grt.generatedFiles {
		for filename, content := range intentFiles {
			allFiles[filename] = content
		}
	}
	return allFiles
}

func (grt *GitRequestTracker) GetGeneratedFilesForIntent(intentName string) map[string]string {
	grt.mu.RLock()
	defer grt.mu.RUnlock()

	if files, exists := grt.generatedFiles[intentName]; exists {
		return files
	}
	return make(map[string]string)
}

func (grt *GitRequestTracker) HasGeneratedFiles(intentName string) bool {
	grt.mu.RLock()
	defer grt.mu.RUnlock()

	files, exists := grt.generatedFiles[intentName]
	return exists && len(files) > 0
}

func (grt *GitRequestTracker) GetGeneratedPackages() map[string]string {
	grt.mu.RLock()
	defer grt.mu.RUnlock()

	result := make(map[string]string)
	for k, v := range grt.generatedPackages {
		result[k] = v
	}
	return result
}

func (grt *GitRequestTracker) SetDeploymentStatus(commitHash, status string) {
	grt.mu.Lock()
	defer grt.mu.Unlock()
	grt.deploymentStatuses[commitHash] = status
}

func (grt *GitRequestTracker) GetDeploymentStatus(commitHash string) string {
	grt.mu.RLock()
	defer grt.mu.RUnlock()
	return grt.deploymentStatuses[commitHash]
}

func (grt *GitRequestTracker) EnableConflictSimulation(enabled bool) {
	grt.mu.Lock()
	defer grt.mu.Unlock()
	grt.conflictSimulation = enabled
}

func (grt *GitRequestTracker) ShouldSimulateConflict() bool {
	grt.mu.RLock()
	defer grt.mu.RUnlock()
	return grt.conflictSimulation
}

// setupMockGitServer creates a mock Git server for testing
func setupMockGitServer(tracker *GitRequestTracker) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/git/clone"):
			handleGitClone(w, r, tracker)
		case strings.Contains(r.URL.Path, "/git/commit"):
			handleGitCommit(w, r, tracker)
		case strings.Contains(r.URL.Path, "/git/push"):
			handleGitPush(w, r, tracker)
		default:
			http.NotFound(w, r)
		}
	}))
}

// setupMockNephioServer creates a mock Nephio Porch server for testing
func setupMockNephioServer(tracker *GitRequestTracker) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/api/porch/v1alpha1/packages"):
			handleNephioPackageGen(w, r, tracker)
		case strings.Contains(r.URL.Path, "/api/porch/v1alpha1/packagerevisions"):
			handleNephioPackageRevision(w, r, tracker)
		default:
			http.NotFound(w, r)
		}
	}))
}

func handleGitClone(w http.ResponseWriter, r *http.Request, tracker *GitRequestTracker) {
	tracker.IncrementRequest("git-clone")

	if tracker.ShouldReturnError("git-clone") {
		http.Error(w, "Git clone failed", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status": "success",
		"path":   "/tmp/repo",
	}
	json.NewEncoder(w).Encode(response)
}

func handleGitCommit(w http.ResponseWriter, r *http.Request, tracker *GitRequestTracker) {
	tracker.IncrementRequest("git-commit")

	if tracker.ShouldReturnError("git-commit") || tracker.ShouldSimulateConflict() {
		http.Error(w, "Git commit conflict", http.StatusConflict)
		return
	}

	commitHash := fmt.Sprintf("commit-%d-%d", time.Now().Unix(), tracker.GetRequestCount("git-commit"))
	response := map[string]interface{}{
		"status":     "success",
		"commitHash": commitHash,
	}

	// Set deployment status to pending
	tracker.SetDeploymentStatus(commitHash, "pending")

	json.NewEncoder(w).Encode(response)
}

func handleGitPush(w http.ResponseWriter, r *http.Request, tracker *GitRequestTracker) {
	tracker.IncrementRequest("git-push")

	if tracker.ShouldReturnError("git-push") {
		http.Error(w, "Git push failed", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status": "success",
	}
	json.NewEncoder(w).Encode(response)
}

func handleNephioPackageGen(w http.ResponseWriter, r *http.Request, tracker *GitRequestTracker) {
	tracker.IncrementRequest("nephio-package-gen")

	if tracker.ShouldReturnError("nephio-package-gen") {
		http.Error(w, "Nephio package generation failed", http.StatusInternalServerError)
		return
	}

	var request struct {
		IntentName      string                 `json:"intentName"`
		NetworkFunction string                 `json:"networkFunction"`
		TargetCluster   string                 `json:"targetCluster"`
		Parameters      map[string]interface{} `json:"parameters"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Generate package files based on network function
	files := generatePackageFiles(request.NetworkFunction, request.Parameters)
	tracker.AddGeneratedFiles(request.IntentName, files)

	// Add to generated packages
	packageName := fmt.Sprintf("%s-%s", request.NetworkFunction, request.TargetCluster)
	tracker.generatedPackages[packageName] = "generated"

	response := map[string]interface{}{
		"status":      "success",
		"packageName": packageName,
		"files":       len(files),
	}
	json.NewEncoder(w).Encode(response)
}

func handleNephioPackageRevision(w http.ResponseWriter, r *http.Request, tracker *GitRequestTracker) {
	tracker.IncrementRequest("nephio-package-revision")

	response := map[string]interface{}{
		"status":   "success",
		"revision": "v1",
	}
	json.NewEncoder(w).Encode(response)
}

// generatePackageFiles generates package files based on network function type
func generatePackageFiles(networkFunction string, parameters map[string]interface{}) map[string]string {
	files := make(map[string]string)

	// Always include Kptfile
	files["Kptfile"] = generateKptfile(networkFunction)

	// Generate files based on network function type
	switch strings.ToUpper(networkFunction) {
	case "AMF":
		files["deployment.yaml"] = generateAMFDeployment(parameters)
		files["service.yaml"] = generateAMFService(parameters)
		files["configmap.yaml"] = generateAMFConfigMap(parameters)
	case "SMF":
		files["deployment.yaml"] = generateSMFDeployment(parameters)
		files["service.yaml"] = generateSMFService(parameters)
		files["pvc.yaml"] = generateSMFPVC(parameters)
	case "UPF":
		files["deployment.yaml"] = generateUPFDeployment(parameters)
		files["service.yaml"] = generateUPFService(parameters)
		files["networkpolicy.yaml"] = generateUPFNetworkPolicy(parameters)
	case "NEAR-RT-RIC", "NEARRTRIC":
		files["deployment.yaml"] = generateRICDeployment(parameters)
		files["service.yaml"] = generateRICService(parameters)
		files["rbac.yaml"] = generateRICRBAC(parameters)
	default:
		files["deployment.yaml"] = generateGenericDeployment(networkFunction, parameters)
		files["service.yaml"] = generateGenericService(networkFunction, parameters)
	}

	return files
}

func generateKptfile(networkFunction string) string {
	return fmt.Sprintf(`apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: %s-package
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: %s network function package generated by Nephoran Intent Operator
pipeline:
  mutators: []
  validators: []`, strings.ToLower(networkFunction), networkFunction)
}

func generateAMFDeployment(parameters map[string]interface{}) string {
	replicas := 3
	if r, ok := parameters["replicas"].(int); ok {
		replicas = r
	}

	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: amf-deployment
  labels:
    app: amf
    component: 5g-core
spec:
  replicas: %d
  selector:
    matchLabels:
      app: amf
  template:
    metadata:
      labels:
        app: amf
    spec:
      containers:
      - name: amf
        image: amf:latest
        ports:
        - containerPort: 8080
        resources:
          requests:
            cpu: "500m"
            memory: "1Gi"
          limits:
            cpu: "2000m"
            memory: "4Gi"`, replicas)
}

func generateAMFService(parameters map[string]interface{}) string {
	return `apiVersion: v1
kind: Service
metadata:
  name: amf-service
spec:
  selector:
    app: amf
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP`
}

func generateAMFConfigMap(parameters map[string]interface{}) string {
	return `apiVersion: v1
kind: ConfigMap
metadata:
  name: amf-config
data:
  amf.conf: |
    logLevel: info
    # AMF configuration here`
}

func generateSMFDeployment(parameters map[string]interface{}) string {
	return `apiVersion: apps/v1
kind: Deployment
metadata:
  name: smf-deployment
spec:
  replicas: 2
  selector:
    matchLabels:
      app: smf
  template:
    metadata:
      labels:
        app: smf
    spec:
      containers:
      - name: smf
        image: smf:latest
        ports:
        - containerPort: 8080`
}

func generateSMFService(parameters map[string]interface{}) string {
	return `apiVersion: v1
kind: Service
metadata:
  name: smf-service
spec:
  selector:
    app: smf
  ports:
  - port: 80
    targetPort: 8080`
}

func generateSMFPVC(parameters map[string]interface{}) string {
	return `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: smf-storage
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi`
}

func generateUPFDeployment(parameters map[string]interface{}) string {
	return `apiVersion: apps/v1
kind: Deployment
metadata:
  name: upf-deployment
spec:
  replicas: 2
  selector:
    matchLabels:
      app: upf
  template:
    metadata:
      labels:
        app: upf
    spec:
      containers:
      - name: upf
        image: upf:latest
        ports:
        - containerPort: 8080`
}

func generateUPFService(parameters map[string]interface{}) string {
	return `apiVersion: v1
kind: Service
metadata:
  name: upf-service
spec:
  selector:
    app: upf
  ports:
  - port: 80
    targetPort: 8080`
}

func generateUPFNetworkPolicy(parameters map[string]interface{}) string {
	return `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: upf-network-policy
spec:
  podSelector:
    matchLabels:
      app: upf
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from: []
  egress:
  - to: []`
}

func generateRICDeployment(parameters map[string]interface{}) string {
	return `apiVersion: apps/v1
kind: Deployment
metadata:
  name: near-rt-ric-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: near-rt-ric
  template:
    metadata:
      labels:
        app: near-rt-ric
    spec:
      containers:
      - name: near-rt-ric
        image: near-rt-ric:latest
        ports:
        - containerPort: 38080`
}

func generateRICService(parameters map[string]interface{}) string {
	return `apiVersion: v1
kind: Service
metadata:
  name: near-rt-ric-service
spec:
  selector:
    app: near-rt-ric
  ports:
  - port: 38080
    targetPort: 38080`
}

func generateRICRBAC(parameters map[string]interface{}) string {
	return `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: near-rt-ric-role
rules:
- apiGroups: [""]
  resources: ["*"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: near-rt-ric-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: near-rt-ric-role
subjects:
- kind: ServiceAccount
  name: near-rt-ric-sa
  namespace: default`
}

func generateGenericDeployment(networkFunction string, parameters map[string]interface{}) string {
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: %s
        image: %s:latest`,
		strings.ToLower(networkFunction),
		strings.ToLower(networkFunction),
		strings.ToLower(networkFunction),
		strings.ToLower(networkFunction),
		strings.ToLower(networkFunction))
}

func generateGenericService(networkFunction string, parameters map[string]interface{}) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s-service
spec:
  selector:
    app: %s
  ports:
  - port: 80
    targetPort: 8080`,
		strings.ToLower(networkFunction),
		strings.ToLower(networkFunction))
}
