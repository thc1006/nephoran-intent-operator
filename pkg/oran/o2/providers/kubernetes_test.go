package providers

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

// TestKubernetesProviderBasics tests basic provider functionality
func TestKubernetesProviderBasics(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	provider, err := NewKubernetesProvider(nil, fakeClient, map[string]string{
		"endpoint":   "https://kubernetes.default.svc",
		"in_cluster": "true",
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Test provider info
	info := provider.GetProviderInfo()
	if info.Name != "kubernetes" {
		t.Errorf("Expected name 'kubernetes', got %s", info.Name)
	}
	if info.Type != ProviderTypeKubernetes {
		t.Errorf("Expected type '%s', got %s", ProviderTypeKubernetes, info.Type)
	}

	// Test supported resource types
	resourceTypes := provider.GetSupportedResourceTypes()
	if len(resourceTypes) == 0 {
		t.Error("Expected resource types to be returned")
	}

	// Test capabilities
	capabilities := provider.GetCapabilities()
	if capabilities == nil {
		t.Error("Expected capabilities to be returned")
	}
}

// TestKubernetesProviderGetDeployment tests the getDeployment method
func TestKubernetesProviderGetDeployment(t *testing.T) {
	// Create a fake deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
			UID:       "test-uid",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 3,
		},
	}

	fakeClient := fake.NewSimpleClientset(deployment)
	provider, err := NewKubernetesProvider(nil, fakeClient, nil)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	kProvider := provider.(*KubernetesProvider)

	// Test getDeployment
	response, err := kProvider.GetDeployment(context.Background(), "default", "test-deployment")
	if err != nil {
		t.Fatalf("Failed to get deployment: %v", err)
	}

	if response.Name != "test-deployment" {
		t.Errorf("Expected name 'test-deployment', got %s", response.Name)
	}
	if response.Type != "deployment" {
		t.Errorf("Expected type 'deployment', got %s", response.Type)
	}
	if response.Health != HealthStatusHealthy {
		t.Errorf("Expected health status 'healthy', got %s", response.Health)
	}
}

// TestKubernetesProviderGetService tests the getService method
func TestKubernetesProviderGetService(t *testing.T) {
	// Create a fake service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.0.0.1",
			Type:      corev1.ServiceTypeClusterIP,
		},
	}

	fakeClient := fake.NewSimpleClientset(service)
	provider, err := NewKubernetesProvider(nil, fakeClient, nil)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	kProvider := provider.(*KubernetesProvider)

	// Test getService
	response, err := kProvider.GetService(context.Background(), "default", "test-service")
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}

	if response.Name != "test-service" {
		t.Errorf("Expected name 'test-service', got %s", response.Name)
	}
	if response.Type != "service" {
		t.Errorf("Expected type 'service', got %s", response.Type)
	}
}

// TestKubernetesProviderGetConfigMap tests the getConfigMap method
func TestKubernetesProviderGetConfigMap(t *testing.T) {
	// Create a fake configmap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
			UID:       "test-uid",
		},
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	fakeClient := fake.NewSimpleClientset(configMap)
	provider, err := NewKubernetesProvider(nil, fakeClient, nil)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	kProvider := provider.(*KubernetesProvider)

	// Test getConfigMap
	response, err := kProvider.GetConfigMap(context.Background(), "default", "test-config")
	if err != nil {
		t.Fatalf("Failed to get configmap: %v", err)
	}

	if response.Name != "test-config" {
		t.Errorf("Expected name 'test-config', got %s", response.Name)
	}
	if response.Type != "configmap" {
		t.Errorf("Expected type 'configmap', got %s", response.Type)
	}
}

// TestKubernetesProviderGetSecret tests the getSecret method
func TestKubernetesProviderGetSecret(t *testing.T) {
	// Create a fake secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			UID:       "test-uid",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("secret123"),
		},
	}

	fakeClient := fake.NewSimpleClientset(secret)
	provider, err := NewKubernetesProvider(nil, fakeClient, nil)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	kProvider := provider.(*KubernetesProvider)

	// Test getSecret
	response, err := kProvider.GetSecret(context.Background(), "default", "test-secret")
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if response.Name != "test-secret" {
		t.Errorf("Expected name 'test-secret', got %s", response.Name)
	}
	if response.Type != "secret" {
		t.Errorf("Expected type 'secret', got %s", response.Type)
	}

	// Verify data keys are exposed but not actual data
	spec := response.Specification
	if spec == nil {
		t.Error("Expected specification to be present")
	} else {
		if dataKeys, ok := spec["dataKeys"].([]string); ok {
			if len(dataKeys) != 2 {
				t.Errorf("Expected 2 data keys, got %d", len(dataKeys))
			}
		} else {
			t.Error("Expected dataKeys in specification")
		}
	}
}

// TestKubernetesProviderGetPVC tests the getPersistentVolumeClaim method
func TestKubernetesProviderGetPVC(t *testing.T) {
	// Create a fake PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
			UID:       "test-uid",
		},
		Status: corev1.PersistentVolumeClaimStatus{
			Phase: corev1.ClaimBound,
		},
	}

	fakeClient := fake.NewSimpleClientset(pvc)
	provider, err := NewKubernetesProvider(nil, fakeClient, nil)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	kProvider := provider.(*KubernetesProvider)

	// Test getPersistentVolumeClaim
	response, err := kProvider.GetPersistentVolumeClaim(context.Background(), "default", "test-pvc")
	if err != nil {
		t.Fatalf("Failed to get pvc: %v", err)
	}

	if response.Name != "test-pvc" {
		t.Errorf("Expected name 'test-pvc', got %s", response.Name)
	}
	if response.Type != "persistentvolumeclaim" {
		t.Errorf("Expected type 'persistentvolumeclaim', got %s", response.Type)
	}
	if response.Health != HealthStatusHealthy {
		t.Errorf("Expected health status 'healthy', got %s", response.Health)
	}
}

// TestKubernetesProviderUpdateDeployment tests the updateDeployment method
func TestKubernetesProviderUpdateDeployment(t *testing.T) {
	// Create a fake deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
		},
	}

	fakeClient := fake.NewSimpleClientset(deployment)
	provider, err := NewKubernetesProvider(nil, fakeClient, nil)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	kProvider := provider.(*KubernetesProvider)

	// Test updateDeployment with replica update
	updateReq := &UpdateResourceRequest{
		Specification: map[string]interface{}{
			"replicas": float64(5),
		},
		Labels: map[string]string{
			"updated": "true",
		},
	}

	response, err := kProvider.UpdateDeployment(context.Background(), "default", "test-deployment", updateReq)
	if err != nil {
		t.Fatalf("Failed to update deployment: %v", err)
	}

	if response.Name != "test-deployment" {
		t.Errorf("Expected name 'test-deployment', got %s", response.Name)
	}
}

// TestKubernetesProviderScaling tests the scaling functionality
func TestKubernetesProviderScaling(t *testing.T) {
	// Create a fake deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
		},
	}

	fakeClient := fake.NewSimpleClientset(deployment)

	// Add a reaction to handle the GetScale call
	fakeClient.PrependReactor("get", "deployments", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
		if action.GetSubresource() == "scale" {
			scale := &autoscalingv1.Scale{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: autoscalingv1.ScaleSpec{
					Replicas: 3,
				},
			}
			return true, scale, nil
		}
		return false, nil, nil
	})

	provider, err := NewKubernetesProvider(nil, fakeClient, nil)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	kProvider := provider.(*KubernetesProvider)

	// Test scaleDeployment
	scaleReq := &ScaleRequest{
		Type:      ScaleTypeHorizontal,
		Direction: ScaleDirectionUp,
		Amount:    2,
	}

	err = kProvider.scaleDeployment(context.Background(), "default", "test-deployment", scaleReq)
	if err != nil {
		t.Fatalf("Failed to scale deployment: %v", err)
	}
}

// Helper function
func int32Ptr(i int32) *int32 {
	return &i
}
