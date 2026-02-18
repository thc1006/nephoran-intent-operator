package e2e

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

var (
	kubeconfig    string
	clusterName   string
	testNamespace string
	skipCleanup   bool
	verbose       bool
	parallel      int
	timeout       time.Duration

	cfg        *rest.Config
	clientset  *kubernetes.Clientset
	testCtx    context.Context
	testCancel context.CancelFunc
	// Note: k8sClient is declared in suite_test.go
)

func init() {
	// Guard flag registration to avoid "flag redefined" panics when a
	// transitive dependency has already registered the same flag name on
	// the default CommandLine FlagSet (e.g. envtest, klog, or another
	// test binary linked into the same binary).
	registerStringFlag := func(p *string, name, value, usage string) {
		if flag.Lookup(name) == nil {
			flag.StringVar(p, name, value, usage)
		} else {
			*p = value // use the default; the existing flag owner controls the value
		}
	}
	registerBoolFlag := func(p *bool, name string, value bool, usage string) {
		if flag.Lookup(name) == nil {
			flag.BoolVar(p, name, value, usage)
		} else {
			*p = value
		}
	}
	registerIntFlag := func(p *int, name string, value int, usage string) {
		if flag.Lookup(name) == nil {
			flag.IntVar(p, name, value, usage)
		} else {
			*p = value
		}
	}
	registerDurationFlag := func(p *time.Duration, name string, value time.Duration, usage string) {
		if flag.Lookup(name) == nil {
			flag.DurationVar(p, name, value, usage)
		} else {
			*p = value
		}
	}

	registerStringFlag(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	registerStringFlag(&clusterName, "cluster", "nephoran-e2e", "Cluster name for testing")
	registerStringFlag(&testNamespace, "namespace", "nephoran-e2e-test", "Namespace for E2E tests")
	registerBoolFlag(&skipCleanup, "skip-cleanup", false, "Skip cleanup after tests")
	registerBoolFlag(&verbose, "verbose", false, "Enable verbose output")
	registerIntFlag(&parallel, "parallel", 1, "Number of parallel test nodes")
	registerDurationFlag(&timeout, "timeout", 30*time.Minute, "Overall test timeout")
}

func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	suiteConfig, reporterConfig := ginkgo.GinkgoConfiguration()
	suiteConfig.Timeout = timeout
	suiteConfig.ParallelTotal = parallel

	if verbose {
		reporterConfig.Verbose = true
		reporterConfig.VeryVerbose = true
	}

	ginkgo.RunSpecs(t, "Nephoran E2E Test Suite", suiteConfig, reporterConfig)
}

var _ = ginkgo.BeforeSuite(func() {
	if os.Getenv("E2E_ENABLED") != "true" {
		ginkgo.Skip("skipping e2e tests: set E2E_ENABLED=true to run")
	}

	ginkgo.By("Setting up E2E test environment")

	// Setup logging
	opts := zap.Options{Development: verbose}
	logger := zap.New(zap.UseFlagOptions(&opts))
	logf.SetLogger(logger)

	// Setup test context
	testCtx, testCancel = context.WithTimeout(context.Background(), timeout)

	// Setup Kubernetes client
	var err error
	if kubeconfig == "" {
		// Try in-cluster config first
		cfg, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig
			loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
			configOverrides := &clientcmd.ConfigOverrides{}
			kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
			cfg, err = kubeConfig.ClientConfig()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to get kubeconfig")
		}
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to load kubeconfig from file")
	}

	// Create clients
	clientset, err = kubernetes.NewForConfig(cfg)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create clientset")

	scheme := runtime.NewScheme()
	err = nephoranv1.AddToScheme(scheme)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to add Nephoran to scheme")

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create controller-runtime client")

	// Verify cluster connectivity
	ginkgo.By("Verifying cluster connectivity")
	_, err = clientset.Discovery().ServerVersion()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to connect to cluster")

	// Create test namespace
	ginkgo.By(fmt.Sprintf("Creating test namespace: %s", testNamespace))
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
			Labels: map[string]string{
				"test":              "e2e",
				"nephoran.com/test": "true",
			},
		},
	}
	err = k8sClient.Create(testCtx, ns)
	if err != nil && !errors.IsAlreadyExists(err) {
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create test namespace")
	}

	// Verify CRDs are installed
	ginkgo.By("Verifying CRDs are installed")
	crdList := &apiextensionsv1.CustomResourceDefinitionList{}
	err = k8sClient.List(testCtx, crdList)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to list CRDs")

	foundNetworkIntent := false
	foundE2NodeSet := false
	for _, crd := range crdList.Items {
		if crd.Name == "networkintents.intent.nephoran.io" {
			foundNetworkIntent = true
		}
		if crd.Name == "e2nodesets.intent.nephoran.io" {
			foundE2NodeSet = true
		}
	}

	gomega.Expect(foundNetworkIntent).To(gomega.BeTrue(), "NetworkIntent CRD not found")
	gomega.Expect(foundE2NodeSet).To(gomega.BeTrue(), "E2NodeSet CRD not found")

	logger.Info("E2E test environment setup complete",
		"cluster", clusterName,
		"namespace", testNamespace,
		"timeout", timeout.String())
})

var _ = ginkgo.AfterSuite(func() {
	if testCancel != nil {
		defer testCancel()
	}

	// Skip cleanup when the BeforeSuite was skipped (k8sClient is nil).
	if k8sClient == nil {
		return
	}

	if !skipCleanup {
		ginkgo.By(fmt.Sprintf("Cleaning up test namespace: %s", testNamespace))

		ns := &corev1.Namespace{}
		err := k8sClient.Get(testCtx, types.NamespacedName{Name: testNamespace}, ns)
		if err == nil {
			err = k8sClient.Delete(testCtx, ns)
			if err != nil {
				ginkgo.GinkgoWriter.Printf("Warning: Failed to delete test namespace: %v\n", err)
			}
		}
	} else {
		ginkgo.GinkgoWriter.Printf("Skipping cleanup as requested. Test namespace '%s' preserved.\n", testNamespace)
	}
})
