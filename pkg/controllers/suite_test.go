package controllers

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/testutils"
	corev1 "k8s.io/api/core/v1"
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

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

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

	// Create test namespaces
	By("creating test namespaces")
	createTestNamespaces()
})

var _ = AfterSuite(func() {
	By("cleaning up test resources")
	cleanupTestResources()

	cancel()

	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// getCRDPaths returns the paths to CRD files with Windows-compatible path handling
func getCRDPaths() []string {
	// Use cross-platform path handling
	crdPath := filepath.Join("..", "..", "deployments", "crds")

	// Convert to absolute path for better reliability
	if absPath, err := filepath.Abs(crdPath); err == nil {
		return []string{absPath}
	}

	return []string{crdPath}
}

// getBinaryAssetsDirectory returns the binary assets directory for envtest
func getBinaryAssetsDirectory() string {
	// On Windows, we might need to specify the binary assets directory
	if runtime.GOOS == "windows" {
		// Try common locations for envtest binaries
		commonPaths := []string{
			filepath.Join("bin", "k8s"),
			filepath.Join("..", "..", "bin", "k8s"),
			"testbin",
		}

		for _, path := range commonPaths {
			if absPath, err := filepath.Abs(path); err == nil {
				return absPath
			}
		}
	}

	return ""
}

// createTestNamespaces creates common test namespaces
func createTestNamespaces() {
	namespaces := []string{
		"test-default",
		"test-5g-core",
		"test-o-ran",
		"test-edge-apps",
	}

	for _, ns := range namespaces {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
				Labels: map[string]string{
					"test-namespace": "true",
				},
			},
		}

		err := k8sClient.Create(ctx, namespace)
		if err != nil {
			// Namespace might already exist, which is fine
			GinkgoLogr.Info("Namespace creation result", "namespace", ns, "error", err)
		}
	}
}

// cleanupTestResources performs cleanup of test resources
func cleanupTestResources() {
	// Clean up test namespaces
	namespaceList := &corev1.NamespaceList{}
	listOptions := []client.ListOption{
		client.MatchingLabels(map[string]string{"test-namespace": "true"}),
	}

	if err := k8sClient.List(ctx, namespaceList, listOptions...); err == nil {
		for _, ns := range namespaceList.Items {
			// Clean up resources in namespace first
			testutils.CleanupTestResources(ctx, k8sClient, ns.Name)

			// Delete namespace with background deletion
			deletePolicy := metav1.DeletePropagationBackground
			deleteOptions := &client.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			}
			_ = k8sClient.Delete(ctx, &ns, deleteOptions)
		}
	}
}

// Helper functions for test categories

// CreateTestNetworkIntent creates a NetworkIntent for testing
func CreateTestNetworkIntent(name, namespace, intent string) *nephoranv1.NetworkIntent {
	fixture := testutils.NetworkIntentFixture{
		Name:      name,
		Namespace: namespace,
		Intent:    intent,
		Expected: nephoranv1.NetworkIntentSpec{
			Intent: intent,
		},
	}
	return testutils.CreateNetworkIntent(fixture)
}

// CreateTestE2NodeSet creates an E2NodeSet for testing
func CreateTestE2NodeSet(name, namespace string, replicas int32) *nephoranv1.E2NodeSet {
	fixture := testutils.E2NodeSetFixture{
		Name:      name,
		Namespace: namespace,
		Replicas:  replicas,
		Expected: nephoranv1.E2NodeSetStatus{
			ReadyReplicas: replicas,
		},
	}
	return testutils.CreateE2NodeSet(fixture)
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
