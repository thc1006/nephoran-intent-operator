package porch_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	networkintentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

// findKubebuilderAssets attempts to locate kubebuilder test assets
func findKubebuilderAssets() string {
	// Check KUBEBUILDER_ASSETS environment variable first
	if assets := os.Getenv("KUBEBUILDER_ASSETS"); assets != "" {
		return assets
	}

	// Try to find setup-envtest assets in common locations
	var possiblePaths []string
	
	if runtime.GOOS == "windows" {
		userProfile := os.Getenv("USERPROFILE")
		if userProfile != "" {
			possiblePaths = append(possiblePaths,
				filepath.Join(userProfile, "AppData", "Local", "kubebuilder-envtest", "k8s", "1.34.0-windows-amd64"),
				filepath.Join(userProfile, "AppData", "Local", "kubebuilder-envtest", "k8s", "1.31.0-windows-amd64"),
			)
		}
	} else {
		homeDir := os.Getenv("HOME")
		if homeDir != "" {
			possiblePaths = append(possiblePaths,
				filepath.Join(homeDir, ".local", "share", "kubebuilder-envtest", "k8s", "1.34.0-linux-amd64"),
				filepath.Join(homeDir, ".local", "share", "kubebuilder-envtest", "k8s", "1.31.0-linux-amd64"),
			)
		}
	}

	// Check if any of these paths exist
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return "" // Will use default behavior
}

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Porch Integration Test Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	By("bootstrapping test environment")
	
	// Construct absolute path to CRDs for reliable loading
	// We are in test/integration/porch/, so go up 3 levels to project root
	crdPath := filepath.Join("..", "..", "..", "deployments", "crds")
	
	// Convert to absolute path for reliability
	if absPath, err := filepath.Abs(crdPath); err == nil {
		crdPath = absPath
	}
	
	By("Using CRD path: " + crdPath)
	
	// Create test environment with proper binary path detection and CRD loading
	envConfig := &envtest.Environment{
		CRDDirectoryPaths:     []string{crdPath},
		ErrorIfCRDPathMissing: true,  // Enable CRD path requirement to load our custom CRDs
	}
	
	// Try to find kubebuilder assets
	if assetsPath := findKubebuilderAssets(); assetsPath != "" {
		By("Found kubebuilder assets at: " + assetsPath)
		envConfig.BinaryAssetsDirectory = assetsPath
	} else {
		By("No kubebuilder assets found, using default behavior")
	}
	
	testEnv = envConfig

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = networkintentv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	ctx, cancel = context.WithCancel(context.TODO())
})

var _ = AfterSuite(func() {
	if cancel != nil {
		cancel()
	}
	By("tearing down the test environment")
	if testEnv != nil {
		err := testEnv.Stop()
		// On Windows, signal-based termination is not supported, but this is expected
		// The processes will still be terminated when the test process exits
		if err != nil {
			By("Test environment stop failed (expected on Windows): " + err.Error())
		}
	}
})

var suiteLetterBytes = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
