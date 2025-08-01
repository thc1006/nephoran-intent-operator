package controllers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/testutils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context
var cancel context.CancelFunc

// Test configuration
const (
	timeout  = time.Second * 30
	interval = time.Millisecond * 250
)

// TestCategory represents different categories of tests
type TestCategory string

const (
	CategoryController  TestCategory = "controller"
	CategoryIntegration TestCategory = "integration"
	CategoryAPI         TestCategory = "api"
	CategoryService     TestCategory = "service"
)

// TestConfig holds configuration for different test categories
type TestConfig struct {
	Category           TestCategory
	RequireControllers bool
	RequireMocks       bool
	IsolateNamespaces  bool
	CleanupTimeout     time.Duration
}

// GetTestConfig returns configuration based on test labels or environment
func GetTestConfig() TestConfig {
	// Default configuration
	config := TestConfig{
		Category:           CategoryController,
		RequireControllers: true,
		RequireMocks:       false,
		IsolateNamespaces:  true,
		CleanupTimeout:     10 * time.Second,
	}

	// Check for test category from environment or labels
	if category := os.Getenv("TEST_CATEGORY"); category != "" {
		switch TestCategory(category) {
		case CategoryIntegration:
			config.Category = CategoryIntegration
			config.RequireControllers = true
			config.RequireMocks = false
			config.CleanupTimeout = 30 * time.Second
		case CategoryAPI:
			config.Category = CategoryAPI
			config.RequireControllers = false
			config.RequireMocks = false
			config.CleanupTimeout = 5 * time.Second
		case CategoryService:
			config.Category = CategoryService
			config.RequireControllers = false
			config.RequireMocks = true
			config.CleanupTimeout = 5 * time.Second
		}
	}

	return config
}

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	testConfig := GetTestConfig()

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     getCRDPaths(),
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: getBinaryAssetsDirectory(),
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = nephoranv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Create test namespaces based on configuration
	By("creating test namespaces")
	if testConfig.IsolateNamespaces {
		createTestNamespaces()
	}

	// Verify CRDs are properly installed
	By("verifying CRD installation")
	verifyCRDInstallation()
})

var _ = AfterSuite(func() {
	testConfig := GetTestConfig()

	By("cleaning up test resources")
	cleanupTestResourcesWithTimeout(testConfig.CleanupTimeout)

	cancel()

	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// getCRDPaths returns the paths to CRD files with Windows-compatible path handling
func getCRDPaths() []string {
	// Try multiple possible CRD locations for better cross-platform compatibility
	possiblePaths := []string{
		filepath.Join("..", "..", "deployments", "crds"),
		filepath.Join("deployments", "crds"),
		filepath.Join("..", "deployments", "crds"),
		"crds",
	}

	var validPaths []string

	for _, path := range possiblePaths {
		// Convert to absolute path and check if it exists
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				validPaths = append(validPaths, absPath)
				break // Use the first valid path found
			}
		}
	}

	// If no valid paths found, try the default relative path
	if len(validPaths) == 0 {
		defaultPath := filepath.Join("..", "..", "deployments", "crds")
		if absPath, err := filepath.Abs(defaultPath); err == nil {
			validPaths = append(validPaths, absPath)
		} else {
			validPaths = append(validPaths, defaultPath)
		}
	}

	// Log the paths being used for debugging
	GinkgoLogr.Info("Using CRD paths", "paths", validPaths)
	return validPaths
}

// getBinaryAssetsDirectory returns the binary assets directory for envtest
func getBinaryAssetsDirectory() string {
	// Try common locations for envtest binaries across platforms
	commonPaths := []string{
		filepath.Join("bin", "k8s"),
		filepath.Join("..", "..", "bin", "k8s"),
		filepath.Join("testbin", "k8s"),
		"testbin",
	}

	// Add Windows-specific paths
	if runtime.GOOS == "windows" {
		windowsPaths := []string{
			filepath.Join(os.Getenv("USERPROFILE"), ".local", "share", "kubebuilder-envtest", "k8s"),
			filepath.Join("C:", "kubebuilder", "bin"),
		}
		commonPaths = append(commonPaths, windowsPaths...)
	} else {
		// Add Unix-specific paths
		unixPaths := []string{
			filepath.Join(os.Getenv("HOME"), ".local", "share", "kubebuilder-envtest", "k8s"),
			"/usr/local/kubebuilder/bin",
		}
		commonPaths = append(commonPaths, unixPaths...)
	}

	// Check each path and return the first valid one
	for _, path := range commonPaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				GinkgoLogr.Info("Using binary assets directory", "path", absPath)
				return absPath
			}
		}
	}

	// If no specific path found, let envtest use its default discovery
	GinkgoLogr.Info("Using default envtest binary discovery")
	return ""
}

// createTestNamespaces creates common test namespaces with proper isolation
func createTestNamespaces() {
	namespaces := []string{
		"test-default",
		"test-5g-core",
		"test-o-ran",
		"test-edge-apps",
		"test-integration",
		"test-api-validation",
	}

	for _, ns := range namespaces {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
				Labels: map[string]string{
					"test-namespace":    "true",
					"test-suite":        "controller-suite",
					"cleanup-policy":    "delete",
					"isolation-enabled": "true",
				},
				Annotations: map[string]string{
					"test.nephoran.com/created-by": "controller-test-suite",
					"test.nephoran.com/created-at": time.Now().Format(time.RFC3339),
				},
			},
		}

		err := k8sClient.Create(ctx, namespace)
		if err != nil && !errors.IsAlreadyExists(err) {
			GinkgoLogr.Error(err, "Failed to create test namespace", "namespace", ns)
		} else {
			GinkgoLogr.Info("Test namespace created or already exists", "namespace", ns)
		}
	}
}

// cleanupTestResourcesWithTimeout performs cleanup of test resources with timeout
func cleanupTestResourcesWithTimeout(cleanupTimeout time.Duration) {
	cleanupCtx, cleanupCancel := context.WithTimeout(ctx, cleanupTimeout)
	defer cleanupCancel()

	By("cleaning up test namespaces and resources")

	// Clean up test namespaces
	namespaceList := &corev1.NamespaceList{}
	listOptions := []client.ListOption{
		client.MatchingLabels(map[string]string{"test-namespace": "true"}),
	}

	if err := k8sClient.List(cleanupCtx, namespaceList, listOptions...); err == nil {
		for _, ns := range namespaceList.Items {
			// Clean up resources in namespace first
			By(fmt.Sprintf("cleaning up resources in namespace %s", ns.Name))
			testutils.CleanupTestResources(cleanupCtx, k8sClient, ns.Name)

			// Delete namespace with background deletion
			deletePolicy := metav1.DeletePropagationBackground
			deleteOptions := &client.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			}

			if err := k8sClient.Delete(cleanupCtx, &ns, deleteOptions); err != nil && !errors.IsNotFound(err) {
				GinkgoLogr.Error(err, "Failed to delete test namespace", "namespace", ns.Name)
			} else {
				GinkgoLogr.Info("Test namespace deleted", "namespace", ns.Name)
			}
		}
	}

	// Clean up any orphaned resources that might not be in test namespaces
	By("cleaning up orphaned test resources")
	cleanupOrphanedResources(cleanupCtx)
}

// cleanupOrphanedResources cleans up resources that might not be in test namespaces
func cleanupOrphanedResources(ctx context.Context) {
	// Clean up NetworkIntents with test labels
	niList := &nephoranv1.NetworkIntentList{}
	if err := k8sClient.List(ctx, niList, client.MatchingLabels(map[string]string{"test-resource": "true"})); err == nil {
		for _, ni := range niList.Items {
			_ = k8sClient.Delete(ctx, &ni)
		}
	}

	// Clean up E2NodeSets with test labels
	e2nsList := &nephoranv1.E2NodeSetList{}
	if err := k8sClient.List(ctx, e2nsList, client.MatchingLabels(map[string]string{"test-resource": "true"})); err == nil {
		for _, e2ns := range e2nsList.Items {
			_ = k8sClient.Delete(ctx, &e2ns)
		}
	}

	// Clean up ManagedElements with test labels
	meList := &nephoranv1.ManagedElementList{}
	if err := k8sClient.List(ctx, meList, client.MatchingLabels(map[string]string{"test-resource": "true"})); err == nil {
		for _, me := range meList.Items {
			_ = k8sClient.Delete(ctx, &me)
		}
	}
}

// verifyCRDInstallation verifies that all required CRDs are properly installed
func verifyCRDInstallation() {
	// List of expected CRDs
	expectedCRDs := []string{
		"networkintents.nephoran.com",
		"e2nodesets.nephoran.com",
		"managedelements.nephoran.com",
	}

	for _, crdName := range expectedCRDs {
		Eventually(func() bool {
			// Try to create a simple list request to verify CRD is available
			switch {
			case strings.Contains(crdName, "networkintents"):
				niList := &nephoranv1.NetworkIntentList{}
				err := k8sClient.List(ctx, niList)
				return err == nil || !errors.IsNotFound(err)
			case strings.Contains(crdName, "e2nodesets"):
				e2nsList := &nephoranv1.E2NodeSetList{}
				err := k8sClient.List(ctx, e2nsList)
				return err == nil || !errors.IsNotFound(err)
			case strings.Contains(crdName, "managedelements"):
				meList := &nephoranv1.ManagedElementList{}
				err := k8sClient.List(ctx, meList)
				return err == nil || !errors.IsNotFound(err)
			}
			return false
		}, 30*time.Second, 1*time.Second).Should(BeTrue(), "CRD %s should be available", crdName)

		GinkgoLogr.Info("CRD verified", "crd", crdName)
	}
}

// Helper functions for test categories

// CreateTestNetworkIntent creates a NetworkIntent for testing with proper labeling
func CreateTestNetworkIntent(name, namespace, intent string) *nephoranv1.NetworkIntent {
	fixture := testutils.NetworkIntentFixture{
		Name:      name,
		Namespace: namespace,
		Intent:    intent,
		Expected: nephoranv1.NetworkIntentSpec{
			Intent: intent,
		},
	}
	ni := testutils.CreateNetworkIntent(fixture)

	// Add test labels for cleanup
	if ni.Labels == nil {
		ni.Labels = make(map[string]string)
	}
	ni.Labels["test-resource"] = "true"
	ni.Labels["test-suite"] = "controller-suite"

	return ni
}

// CreateTestE2NodeSet creates an E2NodeSet for testing with proper labeling
func CreateTestE2NodeSet(name, namespace string, replicas int32) *nephoranv1.E2NodeSet {
	fixture := testutils.E2NodeSetFixture{
		Name:      name,
		Namespace: namespace,
		Replicas:  replicas,
		Expected: nephoranv1.E2NodeSetStatus{
			ReadyReplicas: replicas,
		},
	}
	e2ns := testutils.CreateE2NodeSet(fixture)

	// Add test labels for cleanup
	if e2ns.Labels == nil {
		e2ns.Labels = make(map[string]string)
	}
	e2ns.Labels["test-resource"] = "true"
	e2ns.Labels["test-suite"] = "controller-suite"

	return e2ns
}

// CreateTestManagedElement creates a ManagedElement for testing with proper labeling
func CreateTestManagedElement(name, namespace string) *nephoranv1.ManagedElement {
	me := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"test-resource": "true",
				"test-suite":    "controller-suite",
			},
		},
		Spec: nephoranv1.ManagedElementSpec{
			// Add default spec fields as needed
		},
	}

	return me
}

// WaitForNetworkIntentPhase waits for a NetworkIntent to reach a specific phase
func WaitForNetworkIntentPhase(namespacedName types.NamespacedName, expectedPhase string) {
	testutils.WaitForNetworkIntentPhase(ctx, k8sClient, namespacedName, expectedPhase)
}

// WaitForE2NodeSetReady waits for an E2NodeSet to have the expected number of ready replicas
func WaitForE2NodeSetReady(namespacedName types.NamespacedName, expectedReplicas int32) {
	testutils.WaitForE2NodeSetReady(ctx, k8sClient, namespacedName, expectedReplicas)
}

// GetUniqueNamespace generates a unique namespace name for testing
func GetUniqueNamespace(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// GetUniqueName generates a unique resource name for testing
func GetUniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// CreateIsolatedNamespace creates an isolated namespace for a specific test
func CreateIsolatedNamespace(testName string) string {
	namespaceName := fmt.Sprintf("test-%s-%d", strings.ToLower(testName), time.Now().UnixNano())

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				"test-namespace": "true",
				"test-suite":     "controller-suite",
				"test-isolation": "true",
				"cleanup-policy": "delete",
			},
			Annotations: map[string]string{
				"test.nephoran.com/test-name":  testName,
				"test.nephoran.com/created-by": "controller-test-suite",
				"test.nephoran.com/created-at": time.Now().Format(time.RFC3339),
			},
		},
	}

	Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
	GinkgoLogr.Info("Created isolated test namespace", "namespace", namespaceName, "test", testName)

	return namespaceName
}

// CleanupIsolatedNamespace cleans up an isolated test namespace
func CleanupIsolatedNamespace(namespaceName string) {
	if namespaceName == "" {
		return
	}

	By(fmt.Sprintf("cleaning up isolated namespace %s", namespaceName))

	// Clean up resources in the namespace first
	testutils.CleanupTestResources(ctx, k8sClient, namespaceName)

	// Delete the namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}

	deletePolicy := metav1.DeletePropagationBackground
	deleteOptions := &client.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}

	if err := k8sClient.Delete(ctx, namespace, deleteOptions); err != nil && !errors.IsNotFound(err) {
		GinkgoLogr.Error(err, "Failed to delete isolated namespace", "namespace", namespaceName)
	} else {
		GinkgoLogr.Info("Isolated namespace deleted", "namespace", namespaceName)
	}
}

// WaitForNamespaceCleanup waits for a namespace to be fully cleaned up
func WaitForNamespaceCleanup(namespaceName string) {
	Eventually(func() bool {
		namespace := &corev1.Namespace{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)
		return errors.IsNotFound(err)
	}, 30*time.Second, 1*time.Second).Should(BeTrue(), "Namespace %s should be deleted", namespaceName)
}
