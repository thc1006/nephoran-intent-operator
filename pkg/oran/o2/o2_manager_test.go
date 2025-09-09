package o2

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
<<<<<<< HEAD
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
=======
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
)

func TestO2Manager_DiscoverResources(t *testing.T) {
	tests := []struct {
		name            string
		setupNodes      []corev1.Node
		setupNamespaces []corev1.Namespace
		setupPods       []corev1.Pod
		expectedError   bool
		validateFunc    func(*testing.T, *ResourceMap)
	}{
		{
			name: "successful resource discovery",
			setupNodes: []corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node-1",
						Labels: map[string]string{
							"node-role.kubernetes.io/control-plane": "",
							"kubernetes.io/hostname":                "node-1",
						},
					},
					Status: corev1.NodeStatus{
						Phase: corev1.NodeRunning,
						Capacity: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("4"),
							corev1.ResourceMemory: resource.MustParse("8Gi"),
							corev1.ResourcePods:   resource.MustParse("110"),
						},
						Allocatable: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("3.5"),
							corev1.ResourceMemory: resource.MustParse("7Gi"),
							corev1.ResourcePods:   resource.MustParse("110"),
						},
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node-2",
						Labels: map[string]string{
							"node-role.kubernetes.io/worker": "",
							"kubernetes.io/hostname":         "node-2",
						},
					},
					Status: corev1.NodeStatus{
						Phase: corev1.NodeRunning,
						Capacity: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("8"),
							corev1.ResourceMemory: resource.MustParse("16Gi"),
							corev1.ResourcePods:   resource.MustParse("110"),
						},
						Allocatable: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("7.5"),
							corev1.ResourceMemory: resource.MustParse("15Gi"),
							corev1.ResourcePods:   resource.MustParse("110"),
						},
						Conditions: []corev1.NodeCondition{
							{
								Type:   corev1.NodeReady,
								Status: corev1.ConditionTrue,
							},
						},
					},
				},
			},
			setupNamespaces: []corev1.Namespace{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
						Labels: map[string]string{
							"name": "default",
						},
					},
					Status: corev1.NamespaceStatus{
						Phase: corev1.NamespaceActive,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "o-ran-vnfs",
						Labels: map[string]string{
							"name": "o-ran-vnfs",
						},
					},
					Status: corev1.NamespaceStatus{
						Phase: corev1.NamespaceActive,
					},
				},
			},
			setupPods: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod-1",
						Namespace: "default",
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("500m"),
										corev1.ResourceMemory: resource.MustParse("1Gi"),
									},
								},
							},
						},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
					},
				},
			},
			expectedError: false,
			validateFunc: func(t *testing.T, rm *ResourceMap) {
				assert.NotNil(t, rm)
				assert.Len(t, rm.Nodes, 2)
				assert.Len(t, rm.Namespaces, 2)
				assert.NotNil(t, rm.Metrics)

				// Validate node information
				assert.Contains(t, rm.Nodes, "node-1")
				assert.Contains(t, rm.Nodes, "node-2")

				node1 := rm.Nodes["node-1"]
				assert.Equal(t, "node-1", node1.Name)
				assert.Contains(t, node1.Roles, "control-plane")

				node2 := rm.Nodes["node-2"]
				assert.Equal(t, "node-2", node2.Name)
				assert.Contains(t, node2.Roles, "worker")

				// Validate namespace information
				assert.Contains(t, rm.Namespaces, "default")
				assert.Contains(t, rm.Namespaces, "o-ran-vnfs")

				defaultNS := rm.Namespaces["default"]
				assert.Equal(t, int32(1), defaultNS.PodCount)

				// Validate cluster metrics
				assert.Equal(t, int32(2), rm.Metrics.TotalNodes)
				assert.Equal(t, int32(2), rm.Metrics.ReadyNodes)
				assert.Equal(t, int32(1), rm.Metrics.TotalPods)
			},
		},
		{
			name:            "empty cluster",
			setupNodes:      []corev1.Node{},
			setupNamespaces: []corev1.Namespace{},
			setupPods:       []corev1.Pod{},
			expectedError:   false,
			validateFunc: func(t *testing.T, rm *ResourceMap) {
				assert.NotNil(t, rm)
				assert.Len(t, rm.Nodes, 0)
				assert.Len(t, rm.Namespaces, 0)
				assert.NotNil(t, rm.Metrics)
				assert.Equal(t, int32(0), rm.Metrics.TotalNodes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clientset
			objects := []runtime.Object{}
			for i := range tt.setupNodes {
				objects = append(objects, &tt.setupNodes[i])
			}
			for i := range tt.setupNamespaces {
				objects = append(objects, &tt.setupNamespaces[i])
			}
			for i := range tt.setupPods {
				objects = append(objects, &tt.setupPods[i])
			}

			clientset := fakeclientset.NewSimpleClientset(objects...)

			// Create fake controller-runtime client
			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			ctrlClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()

			// Create O2 adaptor and manager
<<<<<<< HEAD
			adaptor := NewO2Adaptor(ctrlClient, clientset, nil)
=======
			adaptor, err := NewO2Adaptor(ctrlClient, clientset, nil)
			require.NoError(t, err)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			manager := NewO2Manager(adaptor)

			// Execute test
			ctx := context.Background()
			resourceMap, err := manager.DiscoverResources(ctx)

			// Validate results
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resourceMap)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, resourceMap)
				tt.validateFunc(t, resourceMap)
			}
		})
	}
}

func TestO2Manager_ScaleWorkload(t *testing.T) {
	tests := []struct {
		name            string
		workloadID      string
		replicas        int32
		setupDeployment *appsv1.Deployment
		expectedError   bool
		validateFunc    func(*testing.T, *appsv1.Deployment)
	}{
		{
			name:       "successful scale up",
			workloadID: "default/test-deployment",
			replicas:   5,
			setupDeployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(3),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 5,
				},
			},
			expectedError: false,
			validateFunc: func(t *testing.T, deployment *appsv1.Deployment) {
				assert.Equal(t, int32(5), *deployment.Spec.Replicas)
			},
		},
		{
			name:       "successful scale down",
			workloadID: "default/test-deployment",
			replicas:   1,
			setupDeployment: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(3),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: "nginx:latest",
								},
							},
						},
					},
				},
				Status: appsv1.DeploymentStatus{
					ReadyReplicas: 1,
				},
			},
			expectedError: false,
			validateFunc: func(t *testing.T, deployment *appsv1.Deployment) {
				assert.Equal(t, int32(1), *deployment.Spec.Replicas)
			},
		},
		{
			name:            "invalid workload ID format",
			workloadID:      "invalid-format",
			replicas:        3,
			setupDeployment: nil,
			expectedError:   true,
		},
		{
			name:            "deployment not found",
			workloadID:      "default/nonexistent",
			replicas:        3,
			setupDeployment: nil,
			expectedError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clients
			objects := []runtime.Object{}
			if tt.setupDeployment != nil {
				objects = append(objects, tt.setupDeployment)
			}

			clientset := fakeclientset.NewSimpleClientset(objects...)

			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = appsv1.AddToScheme(scheme)
			ctrlClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()

			// Create O2 adaptor and manager
<<<<<<< HEAD
			adaptor := NewO2Adaptor(ctrlClient, clientset, nil)
=======
			adaptor, err := NewO2Adaptor(ctrlClient, clientset, nil)
			require.NoError(t, err)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			manager := NewO2Manager(adaptor)

			// Execute test
			ctx := context.Background()
<<<<<<< HEAD
			err := manager.ScaleWorkload(ctx, tt.workloadID, tt.replicas)
=======
			err = manager.ScaleWorkload(ctx, tt.workloadID, tt.replicas)
>>>>>>> 6835433495e87288b95961af7173d866977175ff

			// Validate results
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.validateFunc != nil {
					// Get the updated deployment
					deployment := &appsv1.Deployment{}
					err := ctrlClient.Get(ctx, client.ObjectKey{
						Namespace: "default",
						Name:      "test-deployment",
					}, deployment)
					require.NoError(t, err)
					tt.validateFunc(t, deployment)
				}
			}
		})
	}
}

func TestO2Manager_DeployVNF(t *testing.T) {
	tests := []struct {
		name          string
		vnfSpec       *VNFDescriptor
		expectedError bool
		validateFunc  func(*testing.T, *DeploymentStatus)
	}{
		{
			name: "successful AMF deployment",
			vnfSpec: &VNFDescriptor{
				Name:     "test-amf",
				Type:     "amf",
				Version:  "v1.0.0",
				Vendor:   "test-vendor",
				Image:    "test-registry/amf:v1.0.0",
				Replicas: 3,
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1000m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2000m"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
				Environment: []corev1.EnvVar{
					{
						Name:  "LOG_LEVEL",
						Value: "INFO",
					},
				},
				Ports: []corev1.ContainerPort{
					{
						Name:          "sbi",
						ContainerPort: 8080,
						Protocol:      corev1.ProtocolTCP,
					},
				},
				NetworkConfig: &NetworkConfig{
					ServiceType: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{
							Name:     "sbi",
							Port:     8080,
							Protocol: corev1.ProtocolTCP,
						},
					},
				},
				Metadata: map[string]string{
					"component": "5g-core",
					"function":  "amf",
				},
			},
			expectedError: false,
			validateFunc: func(t *testing.T, status *DeploymentStatus) {
				assert.Equal(t, "test-amf", status.Name)
				assert.Equal(t, "PENDING", status.Status)
				assert.Equal(t, "Creating", status.Phase)
				assert.Equal(t, int32(3), status.Replicas)
				assert.NotEmpty(t, status.ID)
			},
		},
		{
			name: "successful UPF deployment with volumes",
			vnfSpec: &VNFDescriptor{
				Name:     "test-upf",
				Type:     "upf",
				Version:  "v2.0.0",
				Image:    "test-registry/upf:v2.0.0",
				Replicas: 2,
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2000m"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
				VolumeConfig: []VolumeConfig{
					{
						Name:      "config-volume",
						MountPath: "/etc/upf",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "upf-config",
								},
							},
						},
					},
				},
			},
			expectedError: false,
			validateFunc: func(t *testing.T, status *DeploymentStatus) {
				assert.Equal(t, "test-upf", status.Name)
				assert.Equal(t, int32(2), status.Replicas)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clients
			clientset := fakeclientset.NewSimpleClientset()

			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = appsv1.AddToScheme(scheme)
			ctrlClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			// Create O2 adaptor and manager
<<<<<<< HEAD
			adaptor := NewO2Adaptor(ctrlClient, clientset, nil)
=======
			adaptor, err := NewO2Adaptor(ctrlClient, clientset, nil)
			require.NoError(t, err)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			manager := NewO2Manager(adaptor)

			// Execute test
			ctx := context.Background()
			status, err := manager.DeployVNF(ctx, tt.vnfSpec)

			// Validate results
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, status)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, status)
				tt.validateFunc(t, status)

				// Verify deployment was created
				deployment := &appsv1.Deployment{}
				err := ctrlClient.Get(ctx, client.ObjectKey{
					Namespace: "o-ran-vnfs",
					Name:      tt.vnfSpec.Name,
				}, deployment)
				assert.NoError(t, err)
				assert.Equal(t, tt.vnfSpec.Name, deployment.Name)
				assert.Equal(t, tt.vnfSpec.Replicas, *deployment.Spec.Replicas)
			}
		})
	}
}

func TestO2Adaptor_DeployVNF(t *testing.T) {
	tests := []struct {
		name          string
		deployRequest *VNFDeployRequest
		expectedError bool
		validateFunc  func(*testing.T, *VNFInstance)
	}{
		{
			name: "basic VNF deployment",
			deployRequest: &VNFDeployRequest{
				Name:         "test-vnf",
				Namespace:    "test-ns",
				VNFPackageID: "vnf-package-1",
				FlavorID:     "standard",
				Image:        "test-registry/vnf:latest",
				Replicas:     2,
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				},
				Environment: []corev1.EnvVar{
					{Name: "ENV_VAR", Value: "test-value"},
				},
				Metadata: map[string]string{
					"component": "test",
				},
			},
			expectedError: false,
			validateFunc: func(t *testing.T, instance *VNFInstance) {
				assert.Equal(t, "test-vnf", instance.Name)
				assert.Equal(t, "test-ns", instance.Namespace)
				assert.Equal(t, "vnf-package-1", instance.VNFPackageID)
				assert.Equal(t, "INSTANTIATED", instance.Status.State)
				assert.Equal(t, "CREATING", instance.Status.DetailedState)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake clients
			clientset := fakeclientset.NewSimpleClientset()

			scheme := runtime.NewScheme()
			_ = corev1.AddToScheme(scheme)
			_ = appsv1.AddToScheme(scheme)
			ctrlClient := fake.NewClientBuilder().WithScheme(scheme).Build()

			// Create O2 adaptor
<<<<<<< HEAD
			adaptor := NewO2Adaptor(ctrlClient, clientset, nil)
=======
			adaptor, err := NewO2Adaptor(ctrlClient, clientset, nil)
			require.NoError(t, err)
>>>>>>> 6835433495e87288b95961af7173d866977175ff

			// Execute test
			ctx := context.Background()
			instance, err := adaptor.DeployVNF(ctx, tt.deployRequest)

			// Validate results
			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, instance)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, instance)
<<<<<<< HEAD
				tt.validateFunc(t, instance)
=======
				vnfInstance, ok := instance.(*VNFInstance)
				require.True(t, ok, "expected *VNFInstance, got %T", instance)
				tt.validateFunc(t, vnfInstance)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			}
		})
	}
}

func TestO2Adaptor_ScaleVNF(t *testing.T) {
	// Setup existing deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vnf",
			Namespace: "test-ns",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test-vnf"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "test-vnf"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "test", Image: "test:latest"},
					},
				},
			},
		},
	}

	clientset := fakeclientset.NewSimpleClientset()
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	ctrlClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(deployment).Build()

<<<<<<< HEAD
	adaptor := NewO2Adaptor(ctrlClient, clientset, nil)
=======
	adaptor, err := NewO2Adaptor(ctrlClient, clientset, nil)
	require.NoError(t, err)
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	tests := []struct {
		name             string
		instanceID       string
		scaleRequest     *VNFScaleRequest
		expectedError    bool
		expectedReplicas int32
	}{
		{
			name:       "scale out",
			instanceID: "test-ns-test-vnf",
			scaleRequest: &VNFScaleRequest{
				ScaleType:     "SCALE_OUT",
				NumberOfSteps: 2,
			},
			expectedError:    false,
			expectedReplicas: 5,
		},
		{
			name:       "scale in",
			instanceID: "test-ns-test-vnf",
			scaleRequest: &VNFScaleRequest{
				ScaleType:     "SCALE_IN",
				NumberOfSteps: 1,
			},
			expectedError:    false,
			expectedReplicas: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Set initial replicas
			err := ctrlClient.Get(ctx, client.ObjectKey{
				Namespace: "test-ns",
				Name:      "test-vnf",
			}, deployment)
			require.NoError(t, err)

			deployment.Spec.Replicas = int32Ptr(3)
			err = ctrlClient.Update(ctx, deployment)
			require.NoError(t, err)

			// Execute scaling
			err = adaptor.ScaleVNF(ctx, tt.instanceID, tt.scaleRequest)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify scaling result
				err = ctrlClient.Get(ctx, client.ObjectKey{
					Namespace: "test-ns",
					Name:      "test-vnf",
				}, deployment)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedReplicas, *deployment.Spec.Replicas)
			}
		})
	}
}

// Helper functions
// int32Ptr is defined in example_integration.go

// Integration tests

func TestO2Manager_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This would be an integration test with a real Kubernetes cluster
	// For now, we'll use fake clients but structure it as an integration test

	ctx := context.Background()

	// Setup
	clientset := fakeclientset.NewSimpleClientset()
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	ctrlClient := fake.NewClientBuilder().WithScheme(scheme).Build()

<<<<<<< HEAD
	adaptor := NewO2Adaptor(ctrlClient, clientset, nil)
=======
	adaptor, err := NewO2Adaptor(ctrlClient, clientset, nil)
	require.NoError(t, err)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	manager := NewO2Manager(adaptor)

	// Test complete workflow: discover -> deploy -> scale -> terminate
	t.Run("complete VNF lifecycle", func(t *testing.T) {
		// 1. Discover resources
		resourceMap, err := manager.DiscoverResources(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, resourceMap)

		// 2. Deploy VNF
		vnfSpec := &VNFDescriptor{
			Name:     "integration-test-vnf",
			Type:     "amf",
			Version:  "v1.0.0",
			Image:    "test-registry/amf:v1.0.0",
			Replicas: 2,
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("1000m"),
					corev1.ResourceMemory: resource.MustParse("2Gi"),
				},
			},
		}

		deployStatus, err := manager.DeployVNF(ctx, vnfSpec)
		assert.NoError(t, err)
		assert.NotNil(t, deployStatus)

		// 3. Scale workload
		workloadID := "o-ran-vnfs/integration-test-vnf"
		err = manager.ScaleWorkload(ctx, workloadID, 4)
		assert.NoError(t, err)

		// 4. Verify scaling
		deployment := &appsv1.Deployment{}
		err = ctrlClient.Get(ctx, client.ObjectKey{
			Namespace: "o-ran-vnfs",
			Name:      "integration-test-vnf",
		}, deployment)
		assert.NoError(t, err)
		assert.Equal(t, int32(4), *deployment.Spec.Replicas)
	})
}

// Benchmark tests

func BenchmarkO2Manager_DiscoverResources(b *testing.B) {
	// Setup
	nodes := make([]runtime.Object, 100)
	for i := 0; i < 100; i++ {
		nodes[i] = &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("node-%d", i),
			},
			Status: corev1.NodeStatus{
				Phase: corev1.NodeRunning,
				Capacity: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		}
	}

	clientset := fakeclientset.NewSimpleClientset(nodes...)
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	ctrlClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(nodes...).Build()

<<<<<<< HEAD
	adaptor := NewO2Adaptor(ctrlClient, clientset, nil)
=======
	adaptor, err := NewO2Adaptor(ctrlClient, clientset, nil)
	require.NoError(b, err)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	manager := NewO2Manager(adaptor)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.DiscoverResources(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkO2Adaptor_DeployVNF(b *testing.B) {
	clientset := fakeclientset.NewSimpleClientset()
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	ctrlClient := fake.NewClientBuilder().WithScheme(scheme).Build()

<<<<<<< HEAD
	adaptor := NewO2Adaptor(ctrlClient, clientset, nil)
=======
	adaptor, err := NewO2Adaptor(ctrlClient, clientset, nil)
	require.NoError(b, err)
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	ctx := context.Background()

	deployRequest := &VNFDeployRequest{
		Name:         "bench-vnf",
		Namespace:    "bench-ns",
		VNFPackageID: "bench-package",
		Image:        "bench:latest",
		Replicas:     1,
		Resources: &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		deployRequest.Name = fmt.Sprintf("bench-vnf-%d", i)
		deployRequest.Namespace = fmt.Sprintf("bench-ns-%d", i)

		_, err := adaptor.DeployVNF(ctx, deployRequest)
		if err != nil {
			b.Fatal(err)
		}
	}
}
