package testutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
)

var (
	k8sClient ctrlclient.Client

	clientset *kubernetes.Clientset

	restConfig *rest.Config

	testEnv *envtest.Environment

	ctx context.Context

	cancel context.CancelFunc
)

// GetK8sClient returns the Kubernetes client for testing.

func GetK8sClient() ctrlclient.Client {

	if k8sClient == nil {

		panic("Kubernetes client not initialized. Call SetupTestEnvironment first.")

	}

	return k8sClient

}

// GetClientset returns the Kubernetes clientset for testing.

func GetClientset() *kubernetes.Clientset {

	if clientset == nil {

		panic("Kubernetes clientset not initialized. Call SetupTestEnvironment first.")

	}

	return clientset

}

// GetRestConfig returns the REST config for testing.

func GetRestConfig() *rest.Config {

	if restConfig == nil {

		panic("REST config not initialized. Call SetupTestEnvironment first.")

	}

	return restConfig

}

// GetTestNamespace returns the namespace used for testing.

func GetTestNamespace() string {

	namespace := os.Getenv("TEST_NAMESPACE")

	if namespace == "" {

		namespace = "nephoran-intent-operator-test"

	}

	return namespace

}

// GetTestContext returns the test context.

func GetTestContext() context.Context {

	if ctx == nil {

		ctx = context.Background()

	}

	return ctx

}

// SetupTestEnvironment initializes the test environment.

func SetupTestEnvironment() error {

	logf.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.Background())

	// Use existing cluster if available, otherwise use envtest.

	useExistingCluster := os.Getenv("USE_EXISTING_CLUSTER") == "true"

	testEnv = &envtest.Environment{

		CRDDirectoryPaths: []string{

			filepath.Join("..", "..", "deployments", "crds"),
		},

		ErrorIfCRDPathMissing: false,

		UseExistingCluster: &useExistingCluster,
	}

	var err error

	restConfig, err = testEnv.Start()

	if err != nil {

		return fmt.Errorf("failed to start test environment: %w", err)

	}

	// Add scheme.

	scheme := runtime.NewScheme()

	err = nephoranv1.AddToScheme(scheme)

	if err != nil {

		return fmt.Errorf("failed to add scheme: %w", err)

	}

	// Create client.

	k8sClient, err = ctrlclient.New(restConfig, ctrlclient.Options{Scheme: scheme})

	if err != nil {

		return fmt.Errorf("failed to create client: %w", err)

	}

	// Create clientset.

	clientset, err = kubernetes.NewForConfig(restConfig)

	if err != nil {

		return fmt.Errorf("failed to create clientset: %w", err)

	}

	return nil

}

// TeardownTestEnvironment cleans up the test environment.

func TeardownTestEnvironment() error {

	if cancel != nil {

		cancel()

	}

	if testEnv != nil {

		err := testEnv.Stop()

		if err != nil {

			return fmt.Errorf("failed to stop test environment: %w", err)

		}

	}

	return nil

}

// GetAllDeployments returns all deployments in the given namespace.

func GetAllDeployments(ctx context.Context, client ctrlclient.Client, namespace string) ([]appsv1.Deployment, error) {

	var deployments appsv1.DeploymentList

	err := client.List(ctx, &deployments, ctrlclient.InNamespace(namespace))

	if err != nil {

		return nil, err

	}

	return deployments.Items, nil

}

// GetAllPods returns all pods in the given namespace.

func GetAllPods(ctx context.Context, client ctrlclient.Client, namespace string) ([]corev1.Pod, error) {

	var pods corev1.PodList

	err := client.List(ctx, &pods, ctrlclient.InNamespace(namespace))

	if err != nil {

		return nil, err

	}

	return pods.Items, nil

}

// GetAllSecrets returns all secrets in the given namespace.

func GetAllSecrets(ctx context.Context, client ctrlclient.Client, namespace string) ([]corev1.Secret, error) {

	var secrets corev1.SecretList

	err := client.List(ctx, &secrets, ctrlclient.InNamespace(namespace))

	if err != nil {

		return nil, err

	}

	return secrets.Items, nil

}

// GetAllServices returns all services in the given namespace.

func GetAllServices(ctx context.Context, client ctrlclient.Client, namespace string) ([]corev1.Service, error) {

	var services corev1.ServiceList

	err := client.List(ctx, &services, ctrlclient.InNamespace(namespace))

	if err != nil {

		return nil, err

	}

	return services.Items, nil

}

// GetAllConfigMaps returns all config maps in the given namespace.

func GetAllConfigMaps(ctx context.Context, client ctrlclient.Client, namespace string) ([]corev1.ConfigMap, error) {

	var configMaps corev1.ConfigMapList

	err := client.List(ctx, &configMaps, ctrlclient.InNamespace(namespace))

	if err != nil {

		return nil, err

	}

	return configMaps.Items, nil

}

// CreateTestNamespace creates a namespace for testing.

func CreateTestNamespace(ctx context.Context, client ctrlclient.Client, namespace string) error {

	ns := &corev1.Namespace{

		ObjectMeta: metav1.ObjectMeta{

			Name: namespace,

			Labels: map[string]string{

				"security.nephoran.io/test": "true",

				"pod-security.kubernetes.io/enforce": "restricted",

				"pod-security.kubernetes.io/audit": "restricted",

				"pod-security.kubernetes.io/warn": "restricted",
			},
		},
	}

	err := client.Create(ctx, ns)

	if err != nil && !errors.IsAlreadyExists(err) {

		return err

	}

	return nil

}

// DeleteTestNamespace deletes a test namespace.

func DeleteTestNamespace(ctx context.Context, client ctrlclient.Client, namespace string) error {

	ns := &corev1.Namespace{

		ObjectMeta: metav1.ObjectMeta{

			Name: namespace,
		},
	}

	err := client.Delete(ctx, ns)

	if err != nil && !errors.IsNotFound(err) {

		return err

	}

	return nil

}

// WaitForDeploymentReady waits for a deployment to be ready.

func WaitForDeploymentReady(ctx context.Context, client ctrlclient.Client, namespace, name string, timeout time.Duration) error {

	ctx, cancel := context.WithTimeout(ctx, timeout)

	defer cancel()

	return wait.PollImmediateUntil(time.Second, func() (bool, error) {

		var deployment appsv1.Deployment

		err := client.Get(ctx, types.NamespacedName{

			Name: name,

			Namespace: namespace,
		}, &deployment)

		if err != nil {

			return false, err

		}

		return deployment.Status.ReadyReplicas == deployment.Status.Replicas, nil

	}, ctx.Done())

}

// SecurityTestHelper provides common security testing utilities.

type SecurityTestHelper struct {
	client ctrlclient.Client

	clientset *kubernetes.Clientset

	namespace string
}

// NewSecurityTestHelper creates a new security test helper.

func NewSecurityTestHelper(client ctrlclient.Client, clientset *kubernetes.Clientset, namespace string) *SecurityTestHelper {

	return &SecurityTestHelper{

		client: client,

		clientset: clientset,

		namespace: namespace,
	}

}

// ValidatePodSecurityContext validates a pod's security context.

func (h *SecurityTestHelper) ValidatePodSecurityContext(pod *corev1.Pod) []string {

	var violations []string

	// Check pod security context.

	if pod.Spec.SecurityContext == nil {

		violations = append(violations, "Pod missing security context")

	} else {

		secCtx := pod.Spec.SecurityContext

		if secCtx.RunAsNonRoot == nil || !*secCtx.RunAsNonRoot {

			violations = append(violations, "Pod not configured to run as non-root")

		}

		if secCtx.RunAsUser != nil && *secCtx.RunAsUser == 0 {

			violations = append(violations, "Pod running as root user")

		}

	}

	// Check container security contexts.

	for _, container := range pod.Spec.Containers {

		containerViolations := h.ValidateContainerSecurityContext(&container)

		for _, violation := range containerViolations {

			violations = append(violations, fmt.Sprintf("Container %s: %s", container.Name, violation))

		}

	}

	return violations

}

// ValidateContainerSecurityContext validates a container's security context.

func (h *SecurityTestHelper) ValidateContainerSecurityContext(container *corev1.Container) []string {

	var violations []string

	if container.SecurityContext == nil {

		violations = append(violations, "missing security context")

		return violations

	}

	secCtx := container.SecurityContext

	// Check non-root user.

	if secCtx.RunAsNonRoot == nil || !*secCtx.RunAsNonRoot {

		violations = append(violations, "not configured to run as non-root")

	}

	// Check read-only root filesystem.

	if secCtx.ReadOnlyRootFilesystem == nil || !*secCtx.ReadOnlyRootFilesystem {

		violations = append(violations, "not using read-only root filesystem")

	}

	// Check privilege escalation.

	if secCtx.AllowPrivilegeEscalation == nil || *secCtx.AllowPrivilegeEscalation {

		violations = append(violations, "allows privilege escalation")

	}

	// Check privileged.

	if secCtx.Privileged != nil && *secCtx.Privileged {

		violations = append(violations, "running in privileged mode")

	}

	// Check capabilities.

	if secCtx.Capabilities == nil || len(secCtx.Capabilities.Drop) == 0 {

		violations = append(violations, "does not drop capabilities")

	} else {

		found := false

		for _, cap := range secCtx.Capabilities.Drop {

			if cap == "ALL" {

				found = true

				break

			}

		}

		if !found {

			violations = append(violations, "does not drop ALL capabilities")

		}

	}

	return violations

}

// ValidateResourceLimits validates that containers have resource limits.

func (h *SecurityTestHelper) ValidateResourceLimits(container *corev1.Container) []string {

	var violations []string

	if container.Resources.Limits == nil {

		violations = append(violations, "missing resource limits")

		return violations

	}

	// Check memory limit.

	if container.Resources.Limits.Memory().IsZero() {

		violations = append(violations, "missing memory limit")

	}

	// Check CPU limit.

	if container.Resources.Limits.Cpu().IsZero() {

		violations = append(violations, "missing CPU limit")

	}

	// Check requests.

	if container.Resources.Requests == nil {

		violations = append(violations, "missing resource requests")

	} else {

		if container.Resources.Requests.Memory().IsZero() {

			violations = append(violations, "missing memory request")

		}

		if container.Resources.Requests.Cpu().IsZero() {

			violations = append(violations, "missing CPU request")

		}

	}

	return violations

}

// CheckImageSecurity validates container images.

func (h *SecurityTestHelper) CheckImageSecurity(container *corev1.Container) []string {

	var violations []string

	image := container.Image

	// Check for latest tag.

	if strings.HasSuffix(image, ":latest") {

		violations = append(violations, "using 'latest' tag")

	}

	// Check for missing tag.

	if !strings.Contains(image, ":") && !strings.Contains(image, "@") {

		violations = append(violations, "missing image tag")

	}

	// Check pull policy.

	if image != "" && !strings.HasSuffix(image, ":latest") {

		if container.ImagePullPolicy == corev1.PullAlways {

			violations = append(violations, "using 'Always' pull policy for tagged image")

		}

	}

	return violations

}

// GetSecurityViolations performs a comprehensive security check.

func (h *SecurityTestHelper) GetSecurityViolations(ctx context.Context) (map[string][]string, error) {

	violations := make(map[string][]string)

	// Check deployments.

	deployments, err := GetAllDeployments(ctx, h.client, h.namespace)

	if err != nil {

		return nil, err

	}

	for _, deployment := range deployments {

		deploymentViolations := []string{}

		// Check pod template.

		podTemplate := &deployment.Spec.Template

		// Validate security contexts.

		podViolations := h.ValidatePodSecurityContext(&corev1.Pod{

			Spec: podTemplate.Spec,
		})

		deploymentViolations = append(deploymentViolations, podViolations...)

		// Validate resource limits.

		for _, container := range podTemplate.Spec.Containers {

			resourceViolations := h.ValidateResourceLimits(&container)

			for _, violation := range resourceViolations {

				deploymentViolations = append(deploymentViolations, fmt.Sprintf("Container %s: %s", container.Name, violation))

			}

			imageViolations := h.CheckImageSecurity(&container)

			for _, violation := range imageViolations {

				deploymentViolations = append(deploymentViolations, fmt.Sprintf("Container %s: %s", container.Name, violation))

			}

		}

		if len(deploymentViolations) > 0 {

			violations[fmt.Sprintf("Deployment/%s", deployment.Name)] = deploymentViolations

		}

	}

	return violations, nil

}

// TestSuite represents a security test suite.

type TestSuite struct {
	Name string

	Description string

	Tests []TestCase
}

// TestCase represents a single test case.

type TestCase struct {
	Name string

	Description string

	TestFunc func(*testing.T, *SecurityTestHelper) error
}

// RunSecurityTestSuite runs a complete security test suite.

func RunSecurityTestSuite(t *testing.T, suite TestSuite, helper *SecurityTestHelper) {

	t.Run(suite.Name, func(t *testing.T) {

		for _, testCase := range suite.Tests {

			t.Run(testCase.Name, func(t *testing.T) {

				err := testCase.TestFunc(t, helper)

				if err != nil {

					t.Errorf("Test case %s failed: %v", testCase.Name, err)

				}

			})

		}

	})

}

// BenchmarkResult represents the results of a security benchmark test.

type BenchmarkResult struct {
	Name string

	Duration time.Duration

	Success bool

	Error error
}

// RunBenchmark runs a benchmark test.

func RunBenchmark(name string, testFunc func() error) BenchmarkResult {

	start := time.Now()

	err := testFunc()

	duration := time.Since(start)

	return BenchmarkResult{

		Name: name,

		Duration: duration,

		Success: err == nil,

		Error: err,
	}

}
