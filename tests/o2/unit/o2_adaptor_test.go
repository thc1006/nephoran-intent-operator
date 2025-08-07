package unit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2"
)

func TestO2Adaptor_DeployVNF(t *testing.T) {
	tests := []struct {
		name    string
		request *o2.VNFDeployRequest
		setup   func(*fake.Clientset, client.Client)
		want    func(*testing.T, *o2.VNFInstance, error)
	}{
		{
			name: "successful VNF deployment with basic configuration",
			request: &o2.VNFDeployRequest{
				Name:         "test-amf",
				Namespace:    "o-ran-vnfs",
				VNFPackageID: "amf-v1.0.0",
				FlavorID:     "standard",
				Image:        "registry.example.com/amf:1.0.0",
				Replicas:     2,
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("1"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
				Environment: []corev1.EnvVar{
					{Name: "LOG_LEVEL", Value: "DEBUG"},
				},
				Ports: []corev1.ContainerPort{
					{Name: "sbi", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
				},
				Metadata: map[string]string{
					"app.kubernetes.io/name":    "amf",
					"app.kubernetes.io/version": "1.0.0",
				},
			},
			setup: func(clientset *fake.Clientset, k8sClient client.Client) {
				// Setup any required resources
			},
			want: func(t *testing.T, instance *o2.VNFInstance, err error) {
				require.NoError(t, err)
				assert.NotNil(t, instance)
				assert.Equal(t, "test-amf", instance.Name)
				assert.Equal(t, "o-ran-vnfs", instance.Namespace)
				assert.Equal(t, "amf-v1.0.0", instance.VNFPackageID)
				assert.Equal(t, "INSTANTIATED", instance.Status.State)
				assert.NotEmpty(t, instance.ID)
				assert.NotZero(t, instance.CreatedAt)
			},
		},
		{
			name: "VNF deployment with network configuration",
			request: &o2.VNFDeployRequest{
				Name:         "test-smf",
				Namespace:    "o-ran-vnfs",
				VNFPackageID: "smf-v1.0.0",
				FlavorID:     "standard",
				Image:        "registry.example.com/smf:1.0.0",
				Replicas:     1,
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				},
				Ports: []corev1.ContainerPort{
					{Name: "sbi", ContainerPort: 8080},
					{Name: "pfcp", ContainerPort: 8805, Protocol: corev1.ProtocolUDP},
				},
				NetworkConfig: &o2.NetworkConfig{
					ServiceType: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{Name: "sbi", Port: 8080, Protocol: corev1.ProtocolTCP},
						{Name: "pfcp", Port: 8805, Protocol: corev1.ProtocolUDP},
					},
				},
			},
			want: func(t *testing.T, instance *o2.VNFInstance, err error) {
				require.NoError(t, err)
				assert.NotNil(t, instance)
				assert.Equal(t, "test-smf", instance.Name)
				assert.Len(t, instance.NetworkEndpoints, 2)
				
				// Check that network endpoints are created
				endpointNames := make([]string, len(instance.NetworkEndpoints))
				for i, ep := range instance.NetworkEndpoints {
					endpointNames[i] = ep.Name
				}
				assert.Contains(t, endpointNames, "sbi")
				assert.Contains(t, endpointNames, "pfcp")
			},
		},
		{
			name: "VNF deployment with volume configuration",
			request: &o2.VNFDeployRequest{
				Name:         "test-upf",
				Namespace:    "o-ran-vnfs",
				VNFPackageID: "upf-v1.0.0",
				FlavorID:     "high-performance",
				Image:        "registry.example.com/upf:1.0.0",
				Replicas:     3,
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2"),
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
				},
				VolumeConfig: []o2.VolumeConfig{
					{
						Name:      "config",
						MountPath: "/etc/upf",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "upf-config",
								},
							},
						},
					},
					{
						Name:      "data",
						MountPath: "/var/lib/upf",
						VolumeSource: corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: "upf-data",
							},
						},
					},
				},
			},
			want: func(t *testing.T, instance *o2.VNFInstance, err error) {
				require.NoError(t, err)
				assert.NotNil(t, instance)
				assert.Equal(t, "test-upf", instance.Name)
				assert.Equal(t, "high-performance", instance.FlavorID)
				assert.Equal(t, "2", instance.Resources.CPU)
				assert.Equal(t, "4Gi", instance.Resources.Memory)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			scheme := runtime.NewScheme()
			corev1.AddToScheme(scheme)
			appsv1.AddToScheme(scheme)

			k8sClient := fakeClient.NewClientBuilder().WithScheme(scheme).Build()
			clientset := fake.NewSimpleClientset()

			if tt.setup != nil {
				tt.setup(clientset, k8sClient)
			}

			adaptor := o2.NewO2Adaptor(k8sClient, clientset, &o2.O2Config{
				Namespace: "default",
				DefaultResources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
				},
			})

			ctx := context.Background()
			instance, err := adaptor.DeployVNF(ctx, tt.request)

			tt.want(t, instance, err)
		})
	}
}

func TestO2Adaptor_ScaleVNF(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
		request    *o2.VNFScaleRequest
		setup      func(*fake.Clientset, client.Client)
		want       func(*testing.T, error)
	}{
		{
			name:       "scale out VNF successfully",
			instanceID: "o-ran-vnfs-test-amf",
			request: &o2.VNFScaleRequest{
				ScaleType:     "SCALE_OUT",
				NumberOfSteps: 2,
				AspectID:      "default",
			},
			setup: func(clientset *fake.Clientset, k8sClient client.Client) {
				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-amf",
						Namespace: "o-ran-vnfs",
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: int32Ptr(2),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "test-amf"},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"app": "test-amf"},
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name:  "amf",
									Image: "registry.example.com/amf:1.0.0",
								}},
							},
						},
					},
				}
				require.NoError(t, k8sClient.Create(context.Background(), deployment))
			},
			want: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:       "scale in VNF successfully",
			instanceID: "o-ran-vnfs-test-smf",
			request: &o2.VNFScaleRequest{
				ScaleType:     "SCALE_IN",
				NumberOfSteps: 1,
				AspectID:      "default",
			},
			setup: func(clientset *fake.Clientset, k8sClient client.Client) {
				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-smf",
						Namespace: "o-ran-vnfs",
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: int32Ptr(3),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "test-smf"},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"app": "test-smf"},
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name:  "smf",
									Image: "registry.example.com/smf:1.0.0",
								}},
							},
						},
					},
				}
				require.NoError(t, k8sClient.Create(context.Background(), deployment))
			},
			want: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:       "handle invalid instance ID format",
			instanceID: "invalid-format",
			request: &o2.VNFScaleRequest{
				ScaleType:     "SCALE_OUT",
				NumberOfSteps: 1,
			},
			want: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid instance ID format")
			},
		},
		{
			name:       "handle non-existent deployment",
			instanceID: "o-ran-vnfs-non-existent",
			request: &o2.VNFScaleRequest{
				ScaleType:     "SCALE_OUT",
				NumberOfSteps: 1,
			},
			want: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get deployment")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			corev1.AddToScheme(scheme)
			appsv1.AddToScheme(scheme)

			k8sClient := fakeClient.NewClientBuilder().WithScheme(scheme).Build()
			clientset := fake.NewSimpleClientset()

			if tt.setup != nil {
				tt.setup(clientset, k8sClient)
			}

			adaptor := o2.NewO2Adaptor(k8sClient, clientset, nil)

			ctx := context.Background()
			err := adaptor.ScaleVNF(ctx, tt.instanceID, tt.request)

			tt.want(t, err)
		})
	}
}

func TestO2Adaptor_GetVNFInstance(t *testing.T) {
	tests := []struct {
		name       string
		instanceID string
		setup      func(*fake.Clientset, client.Client)
		want       func(*testing.T, *o2.VNFInstance, error)
	}{
		{
			name:       "get existing VNF instance successfully",
			instanceID: "o-ran-vnfs-test-amf",
			setup: func(clientset *fake.Clientset, k8sClient client.Client) {
				deployment := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-amf",
						Namespace: "o-ran-vnfs",
						Labels: map[string]string{
							"app":                     "test-amf",
							"app.kubernetes.io/name":  "amf",
							"nephoran.com/vnf":        "true",
						},
						CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Hour)),
					},
					Spec: appsv1.DeploymentSpec{
						Replicas: int32Ptr(2),
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "test-amf"},
						},
						Template: corev1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"app": "test-amf"},
							},
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{{
									Name:  "amf",
									Image: "registry.example.com/amf:1.0.0",
								}},
							},
						},
					},
					Status: appsv1.DeploymentStatus{
						ReadyReplicas: 2,
					},
				}
				require.NoError(t, k8sClient.Create(context.Background(), deployment))

				// Create associated service
				service := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-amf-service",
						Namespace: "o-ran-vnfs",
						Labels:    map[string]string{"app": "test-amf"},
					},
					Spec: corev1.ServiceSpec{
						Selector: map[string]string{"app": "test-amf"},
						Ports: []corev1.ServicePort{
							{Name: "sbi", Port: 8080, Protocol: corev1.ProtocolTCP},
						},
						ClusterIP: "10.96.1.100",
					},
				}
				require.NoError(t, k8sClient.Create(context.Background(), service))
			},
			want: func(t *testing.T, instance *o2.VNFInstance, err error) {
				require.NoError(t, err)
				assert.NotNil(t, instance)
				assert.Equal(t, "o-ran-vnfs-test-amf", instance.ID)
				assert.Equal(t, "test-amf", instance.Name)
				assert.Equal(t, "o-ran-vnfs", instance.Namespace)
				assert.Equal(t, "RUNNING", instance.Status.DetailedState)
				assert.Len(t, instance.NetworkEndpoints, 1)
				assert.Equal(t, "sbi", instance.NetworkEndpoints[0].Name)
				assert.Equal(t, "10.96.1.100", instance.NetworkEndpoints[0].Address)
				assert.Equal(t, int32(8080), instance.NetworkEndpoints[0].Port)
			},
		},
		{
			name:       "handle non-existent VNF instance",
			instanceID: "o-ran-vnfs-non-existent",
			want: func(t *testing.T, instance *o2.VNFInstance, err error) {
				require.Error(t, err)
				assert.Nil(t, instance)
				assert.Contains(t, err.Error(), "failed to get deployment")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			corev1.AddToScheme(scheme)
			appsv1.AddToScheme(scheme)

			k8sClient := fakeClient.NewClientBuilder().WithScheme(scheme).Build()
			clientset := fake.NewSimpleClientset()

			if tt.setup != nil {
				tt.setup(clientset, k8sClient)
			}

			adaptor := o2.NewO2Adaptor(k8sClient, clientset, nil)

			ctx := context.Background()
			instance, err := adaptor.GetVNFInstance(ctx, tt.instanceID)

			tt.want(t, instance, err)
		})
	}
}

func TestO2Adaptor_GetInfrastructureInfo(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*fake.Clientset, client.Client)
		want  func(*testing.T, *o2.InfrastructureInfo, error)
	}{
		{
			name: "get infrastructure info successfully",
			setup: func(clientset *fake.Clientset, k8sClient client.Client) {
				// Create test nodes
				node1 := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker-1",
						Labels: map[string]string{
							"kubernetes.io/os":                   "linux",
							"node-role.kubernetes.io/worker":     "",
						},
					},
					Status: corev1.NodeStatus{
						Capacity: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("4"),
							corev1.ResourceMemory: resource.MustParse("8Gi"),
						},
						Allocatable: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("3900m"),
							corev1.ResourceMemory: resource.MustParse("7600Mi"),
						},
					},
				}
				require.NoError(t, k8sClient.Create(context.Background(), node1))

				node2 := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "worker-2",
						Labels: map[string]string{
							"kubernetes.io/os":               "linux",
							"node-role.kubernetes.io/worker": "",
						},
					},
					Status: corev1.NodeStatus{
						Capacity: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("8"),
							corev1.ResourceMemory: resource.MustParse("16Gi"),
						},
					},
				}
				require.NoError(t, k8sClient.Create(context.Background(), node2))
			},
			want: func(t *testing.T, info *o2.InfrastructureInfo, err error) {
				require.NoError(t, err)
				assert.NotNil(t, info)
				assert.Equal(t, 2, info.NodeCount)
				assert.Equal(t, "kubernetes-cluster", info.ClusterName)
				assert.NotEmpty(t, info.KubernetesVersion)
				assert.NotNil(t, info.TotalResources)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			corev1.AddToScheme(scheme)

			k8sClient := fakeClient.NewClientBuilder().WithScheme(scheme).Build()
			clientset := fake.NewSimpleClientset()

			if tt.setup != nil {
				tt.setup(clientset, k8sClient)
			}

			adaptor := o2.NewO2Adaptor(k8sClient, clientset, nil)

			ctx := context.Background()
			info, err := adaptor.GetInfrastructureInfo(ctx)

			tt.want(t, info, err)
		})
	}
}

// Benchmark tests for performance validation
func BenchmarkO2Adaptor_DeployVNF(b *testing.B) {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)

	k8sClient := fakeClient.NewClientBuilder().WithScheme(scheme).Build()
	clientset := fake.NewSimpleClientset()
	adaptor := o2.NewO2Adaptor(k8sClient, clientset, nil)

	request := &o2.VNFDeployRequest{
		Name:         "bench-vnf",
		Namespace:    "default",
		VNFPackageID: "test-package",
		FlavorID:     "standard",
		Image:        "test:latest",
		Replicas:     1,
		Resources: &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request.Name = string(rune('a' + i%26)) + string(rune('a' + (i/26)%26)) // Generate unique names
		_, err := adaptor.DeployVNF(ctx, request)
		if err != nil {
			b.Fatalf("DeployVNF failed: %v", err)
		}
	}
}

func BenchmarkO2Adaptor_GetVNFInstance(b *testing.B) {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	appsv1.AddToScheme(scheme)

	k8sClient := fakeClient.NewClientBuilder().WithScheme(scheme).Build()
	clientset := fake.NewSimpleClientset()
	adaptor := o2.NewO2Adaptor(k8sClient, clientset, nil)

	// Setup test deployment
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bench-vnf",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "bench-vnf"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "bench-vnf"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "test",
						Image: "test:latest",
					}},
				},
			},
		},
	}
	require.NoError(b, k8sClient.Create(context.Background(), deployment))

	ctx := context.Background()
	instanceID := "default-bench-vnf"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := adaptor.GetVNFInstance(ctx, instanceID)
		if err != nil {
			b.Fatalf("GetVNFInstance failed: %v", err)
		}
	}
}

// Helper functions
func int32Ptr(i int32) *int32 {
	return &i
}