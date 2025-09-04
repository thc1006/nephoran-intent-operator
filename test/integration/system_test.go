package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	porchclient "github.com/thc1006/nephoran-intent-operator/pkg/porch"
)

func TestSystemIntegration(t *testing.T) {
	config, err := rest.InClusterConfig()
	require.NoError(t, err, "Failed to load Kubernetes config")

	// Create various clients for comprehensive testing
	k8sClient, err := kubernetes.NewForConfig(config)
	require.NoError(t, err, "Failed to create Kubernetes client")

	dynamicClient, err := dynamic.NewForConfig(config)
	require.NoError(t, err, "Failed to create dynamic client")

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	require.NoError(t, err, "Failed to create discovery client")

	porchClient, err := porchclient.NewClient(config)
	require.NoError(t, err, "Failed to create Porch client")

	t.Run("KubernetesAPIServerHealthCheck", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Check API server version
		version, err := k8sClient.Discovery().ServerVersion()
		require.NoError(t, err, "Failed to get Kubernetes server version")
		assert.NotNil(t, version, "Server version should not be nil")
	})

	t.Run("ResourceMappingTest", func(t *testing.T) {
		// Test resource mapping capabilities
		groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
		require.NoError(t, err, "Failed to get API group resources")

		mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
		testMapping := &runtime.UnstructuredList{}
		
		// Find mapping for a known resource type
		mapping, err := mapper.RESTMapping(porchv1alpha1.SchemeGroupVersion.WithKind("Package").GroupKind())
		require.NoError(t, err, "Failed to find REST mapping for Package")
		assert.NotNil(t, mapping, "Resource mapping should not be nil")
	})

	t.Run("GitRepositoryBackendTest", func(t *testing.T) {
		pkgName := "system-test-package"
		pkg := &porchv1alpha1.Package{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: "nephio-system-test",
			},
			Spec: porchv1alpha1.PackageSpec{
				Repository: "system-test-repo",
				GitRepository: porchv1alpha1.GitRepositorySpec{
					Ref: "main",
					Directory: "/test-packages",
				},
			},
		}

		// Create package with Git repository backend
		createdPkg, err := porchClient.Create(context.Background(), pkg)
		require.NoError(t, err, "Failed to create package with Git repository")
		assert.Equal(t, pkg.Spec.GitRepository.Ref, createdPkg.Spec.GitRepository.Ref)
	})

	t.Run("MonitoringEndpointTest", func(t *testing.T) {
		// Test metrics endpoint
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Use dynamic client to fetch metrics/health endpoints
		gvr := metav1.GroupVersionResource{
			Group:    "metrics.k8s.io",
			Version:  "v1beta1",
			Resource: "pods",
		}

		_, err := dynamicClient.Resource(gvr).Namespace("kube-system").List(ctx, metav1.ListOptions{})
		assert.NoError(t, err, "Should be able to retrieve metrics")
	})

	t.Run("HealthCheckEndpoint", func(t *testing.T) {
		healthCheckURL := "/healthz"
		req, err := rest.NewRequest(k8sClient.CoreV1().RESTClient()).Prefix(healthCheckURL).Do(context.Background())
		require.NoError(t, err, "Failed to create health check request")

		rawResult, err := req.Raw()
		require.NoError(t, err, "Failed to get health check response")
		assert.Equal(t, "ok", string(rawResult), "Health check should return 'ok'")
	})
}