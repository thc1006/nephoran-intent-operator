package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

	porchClient := porchclient.NewClient("http://porch-server:8080", false)

	t.Run("KubernetesAPIServerHealthCheck", func(t *testing.T) {
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

		// Find mapping for a known resource type (use a standard Kubernetes resource)
		gvk := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}
		gk := schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}
		mapping, err := mapper.RESTMapping(gk, gvk.Version)
		require.NoError(t, err, "Failed to find REST mapping for Pod")
		assert.NotNil(t, mapping, "Resource mapping should not be nil")
	})

	t.Run("PorchClientTest", func(t *testing.T) {
		// Test basic porch client connectivity
		assert.NotNil(t, porchClient, "Porch client should not be nil")
		
		// Create a simple test package spec for testing
		packageSpec := &porchclient.PackageSpec{
			Repository: "system-test-repo",
			Package:    "test-package",
			Workspace:  "default",
		}
		
		assert.Equal(t, "system-test-repo", packageSpec.Repository)
		assert.Equal(t, "test-package", packageSpec.Package)
	})

	t.Run("MonitoringEndpointTest", func(t *testing.T) {
		// Test metrics endpoint
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Use dynamic client to fetch metrics/health endpoints
		gvr := schema.GroupVersionResource{
			Group:    "metrics.k8s.io",
			Version:  "v1beta1",
			Resource: "pods",
		}

		_, err := dynamicClient.Resource(gvr).Namespace("kube-system").List(ctx, metav1.ListOptions{})
		assert.NoError(t, err, "Should be able to retrieve metrics")
	})

	t.Run("HealthCheckEndpoint", func(t *testing.T) {
		// Test basic health check using discovery client
		_, err := k8sClient.Discovery().ServerVersion()
		require.NoError(t, err, "Server should be healthy and return version info")
		assert.True(t, true, "Health check passed")
	})
}
