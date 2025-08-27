package admission

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
)

var (
	k8sClient  client.Client
	testEnv    *envtest.Environment
	ctx        context.Context
	cancel     context.CancelFunc
	testScheme *runtime.Scheme
)

func TestWebhooks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webhook EnvTest Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "deployments", "crds")},
		ErrorIfCRDPathMissing: false,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			Paths: []string{filepath.Join("..", "..", "config", "webhook")},
		},
	}

	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	testScheme = runtime.NewScheme()
	err = scheme.AddToScheme(testScheme)
	Expect(err).NotTo(HaveOccurred())
	err = intentv1alpha1.AddToScheme(testScheme)
	Expect(err).NotTo(HaveOccurred())

	// Create manager with webhook server
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: testScheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    testEnv.WebhookInstallOptions.LocalServingHost,
			Port:    testEnv.WebhookInstallOptions.LocalServingPort,
			CertDir: testEnv.WebhookInstallOptions.LocalServingCertDir,
		}),
	})
	Expect(err).NotTo(HaveOccurred())

	// Setup webhook
	err = (&intentv1alpha1.NetworkIntent{}).SetupWebhookWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	// Start manager
	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()

	// Create client
	k8sClient = mgr.GetClient()
	Expect(k8sClient).NotTo(BeNil())

	// Wait for webhook to be ready
	Eventually(func() error {
		return mgr.GetWebhookServer().StartedChecker()(nil)
	}, 10*time.Second, 100*time.Millisecond).Should(Succeed())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	cancel()
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("NetworkIntent Admission Webhook", func() {
	const (
		testNamespace = "default"
		timeout       = time.Second * 10
		interval      = time.Millisecond * 250
	)

	Context("Defaulting webhook", func() {
		It("should set default source to 'user' when not specified", func() {
			ni := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-default-source",
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "my-deployment",
					Namespace:  testNamespace,
					Replicas:   3,
					// Source not specified - should default to "user"
				},
			}

			err := k8sClient.Create(ctx, ni)
			Expect(err).NotTo(HaveOccurred())

			// Retrieve the created object
			created := &intentv1alpha1.NetworkIntent{}
			err = k8sClient.Get(ctx, client.ObjectKey{
				Name:      ni.Name,
				Namespace: ni.Namespace,
			}, created)
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Spec.Source).To(Equal("user"))

			// Cleanup
			err = k8sClient.Delete(ctx, created)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should not override existing source value", func() {
			ni := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-keep-source",
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "my-deployment",
					Namespace:  testNamespace,
					Replicas:   3,
					Source:     "planner",
				},
			}

			err := k8sClient.Create(ctx, ni)
			Expect(err).NotTo(HaveOccurred())

			// Retrieve the created object
			created := &intentv1alpha1.NetworkIntent{}
			err = k8sClient.Get(ctx, client.ObjectKey{
				Name:      ni.Name,
				Namespace: ni.Namespace,
			}, created)
			Expect(err).NotTo(HaveOccurred())
			Expect(created.Spec.Source).To(Equal("planner"))

			// Cleanup
			err = k8sClient.Delete(ctx, created)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Validating webhook", func() {
		It("should accept valid NetworkIntent", func() {
			ni := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-valid",
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "my-deployment",
					Namespace:  testNamespace,
					Replicas:   5,
					Source:     "user",
				},
			}

			err := k8sClient.Create(ctx, ni)
			Expect(err).NotTo(HaveOccurred())

			// Cleanup
			err = k8sClient.Delete(ctx, ni)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should reject NetworkIntent with negative replicas", func() {
			ni := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-replicas",
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "my-deployment",
					Namespace:  testNamespace,
					Replicas:   -1, // Invalid
					Source:     "user",
				},
			}

			err := k8sClient.Create(ctx, ni)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be >= 0"))
		})

		It("should reject NetworkIntent with invalid intent type", func() {
			ni := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-type",
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "provisioning", // Invalid - only "scaling" supported
					Target:     "my-deployment",
					Namespace:  testNamespace,
					Replicas:   5,
					Source:     "user",
				},
			}

			err := k8sClient.Create(ctx, ni)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("only 'scaling' supported"))
		})

		It("should reject NetworkIntent with empty target", func() {
			ni := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-empty-target",
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "", // Empty
					Namespace:  testNamespace,
					Replicas:   5,
					Source:     "user",
				},
			}

			err := k8sClient.Create(ctx, ni)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be non-empty"))
		})

		It("should reject NetworkIntent with invalid source", func() {
			ni := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-invalid-source",
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "my-deployment",
					Namespace:  testNamespace,
					Replicas:   5,
					Source:     "invalid", // Invalid - must be user, planner, or test
				},
			}

			err := k8sClient.Create(ctx, ni)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be 'user', 'planner', or 'test'"))
		})

		It("should validate updates with same rules", func() {
			// First create a valid intent
			ni := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-update",
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "my-deployment",
					Namespace:  testNamespace,
					Replicas:   5,
					Source:     "user",
				},
			}

			err := k8sClient.Create(ctx, ni)
			Expect(err).NotTo(HaveOccurred())

			// Try to update with invalid data
			ni.Spec.Replicas = -2
			err = k8sClient.Update(ctx, ni)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must be >= 0"))

			// Cleanup
			ni.Spec.Replicas = 5
			err = k8sClient.Update(ctx, ni)
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Delete(ctx, ni)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should allow deletion without validation", func() {
			// Create a valid intent
			ni := &intentv1alpha1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-delete",
					Namespace: testNamespace,
				},
				Spec: intentv1alpha1.NetworkIntentSpec{
					IntentType: "scaling",
					Target:     "my-deployment",
					Namespace:  testNamespace,
					Replicas:   5,
					Source:     "user",
				},
			}

			err := k8sClient.Create(ctx, ni)
			Expect(err).NotTo(HaveOccurred())

			// Delete should succeed
			err = k8sClient.Delete(ctx, ni)
			Expect(err).NotTo(HaveOccurred())

			// Verify deletion
			err = k8sClient.Get(ctx, client.ObjectKey{
				Name:      ni.Name,
				Namespace: ni.Namespace,
			}, ni)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})
})
