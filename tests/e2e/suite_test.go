package e2e

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoran "github.com/nephio-project/nephoran-intent-operator/api/v1"
	"github.com/nephio-project/nephoran-intent-operator/pkg/controllers"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	suiteCfg      *rest.Config
	suiteClient   client.Client
	testEnv       *envtest.Environment
	suiteCtx      context.Context
	suiteCancel   context.CancelFunc
)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	suiteCtx, suiteCancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
		},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// suiteCfg is defined in this file globally.
	suiteCfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(suiteCfg).NotTo(BeNil())

	err = nephoran.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	suiteClient, err = client.New(suiteCfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(suiteClient).NotTo(BeNil())

	// Start the controller manager
	k8sManager, err := ctrl.NewManager(suiteCfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.NetworkIntentReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&controllers.E2NodeSetReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(suiteCtx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	// Wait for the manager to be ready
	Eventually(func() bool {
		return k8sManager.GetCache().WaitForCacheSync(suiteCtx)
	}, 30*time.Second, 1*time.Second).Should(BeTrue())
})

var _ = AfterSuite(func() {
	suiteCancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
