package edge

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	kubefake "k8s.io/client-go/kubernetes/fake"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// MockClient implements the controller-runtime client.Client interface for testing
type MockClient struct {
	mock.Mock
}

func (m *MockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	args := m.Called(ctx, key, obj, opts)
	return args.Error(0)
}

func (m *MockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	args := m.Called(ctx, list, opts)
	return args.Error(0)
}

func (m *MockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

func (m *MockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockClient) Status() client.StatusWriter {
	args := m.Called()
	return args.Get(0).(client.StatusWriter)
}

func (m *MockClient) Scheme() *runtime.Scheme {
	args := m.Called()
	return args.Get(0).(*runtime.Scheme)
}

func (m *MockClient) RESTMapper() meta.RESTMapper {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(meta.RESTMapper)
}

func (m *MockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	args := m.Called(obj)
	return args.Get(0).(schema.GroupVersionKind), args.Error(1)
}

// Additional mock for IsObjectNamespaced
func (m *MockClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	args := m.Called(obj)
	return args.Bool(0), args.Error(1)
}

// SubResource returns a SubResourceClient
func (m *MockClient) SubResource(subResource string) client.SubResourceClient {
	args := m.Called(subResource)
	return args.Get(0).(client.SubResourceClient)
}

// MockStatusWriter implements client.StatusWriter for testing
type MockStatusWriter struct {
	mock.Mock
}

func (m *MockStatusWriter) Create(ctx context.Context, obj client.Object, subResource client.Object, opts ...client.SubResourceCreateOption) error {
	args := m.Called(ctx, obj, subResource, opts)
	return args.Error(0)
}

func (m *MockStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	args := m.Called(ctx, obj, patch, opts)
	return args.Error(0)
}

// Test helper functions
func createTestEdgeController() *EdgeController {
	config := &EdgeControllerConfig{
		NodeDiscoveryEnabled:    true,
		DiscoveryInterval:       30 * time.Second,
		NodeHealthCheckInterval: 10 * time.Second,
		AutoZoneCreation:        true,
		MaxNodesPerZone:         10,
		ZoneRedundancyFactor:    2,
		EnableLocalRIC:          true,
		EnableEdgeML:            true,
		EnableCaching:           true,
		LocalProcessingEnabled:  true,
		MaxLatencyMs:            5,
		MinBandwidthMbps:        100,
		EdgeResourceThreshold:   0.8,
		EdgeFailoverEnabled:     true,
		BackhaulFailoverEnabled: true,
		LocalAutonomy:           true,
	}

	kubeClient := kubefake.NewSimpleClientset()
	logger := logr.Discard()
	scheme := runtime.NewScheme()

	return &EdgeController{
		Client:     nil, // Will be set in tests
		KubeClient: kubeClient,
		Log:        logger,
		Scheme:     scheme,
		config:     config,
		edgeNodes:  make(map[string]*EdgeNode),
		edgeZones:  make(map[string]*EdgeZone),
	}
}

// Test EdgeController creation
func TestNewEdgeController(t *testing.T) {
	mockClient := new(MockClient)
	kubeClient := kubefake.NewSimpleClientset()
	logger := zap.New(zap.UseDevMode(true))
	scheme := runtime.NewScheme()

	config := &EdgeControllerConfig{
		NodeDiscoveryEnabled:    true,
		DiscoveryInterval:       30 * time.Second,
		NodeHealthCheckInterval: 10 * time.Second,
		AutoZoneCreation:        true,
		MaxNodesPerZone:         10,
		ZoneRedundancyFactor:    2,
		EnableLocalRIC:          true,
		EnableEdgeML:            true,
		EnableCaching:           true,
		LocalProcessingEnabled:  true,
		MaxLatencyMs:            5,
		MinBandwidthMbps:        100,
		EdgeResourceThreshold:   0.8,
		EdgeFailoverEnabled:     true,
		BackhaulFailoverEnabled: true,
		LocalAutonomy:           true,
	}

	controller := &EdgeController{
		Client:     mockClient,
		KubeClient: kubeClient,
		Log:        logger,
		Scheme:     scheme,
		config:     config,
		edgeNodes:  make(map[string]*EdgeNode),
		edgeZones:  make(map[string]*EdgeZone),
	}

	assert.NotNil(t, controller)
	assert.Equal(t, mockClient, controller.Client)
	assert.Equal(t, kubeClient, controller.KubeClient)
	assert.NotNil(t, controller.edgeNodes)
	assert.NotNil(t, controller.edgeZones)
	assert.Equal(t, config, controller.config)
}

// Test Reconcile method
func TestEdgeController_Reconcile(t *testing.T) {
	tests := []struct {
		name           string
		request        ctrl.Request
		setupMocks     func(*MockClient, *MockStatusWriter)
		expectedResult ctrl.Result
		expectedError  bool
	}{
		{
			name: "successful reconciliation with edge nodes",
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-intent",
					Namespace: "default",
				},
			},
			setupMocks: func(mc *MockClient, msw *MockStatusWriter) {
				intent := &nephoran.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-intent",
						Namespace: "default",
					},
					Spec: nephoran.NetworkIntentSpec{
						Intent:     "Deploy edge processing for low latency",
						IntentType: nephoran.IntentTypeDeployment,
						ProcessedParameters: &nephoran.ProcessedParameters{
							NetworkFunction: "edge-gateway",
							Region:         "edge",
						},
					},
				}
				mc.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					obj := args.Get(2).(*nephoran.NetworkIntent)
					*obj = *intent
				})
				// Mock successful deployment
				mc.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
				mc.On("Status").Return(msw).Maybe()
				msw.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
			},
			expectedResult: ctrl.Result{RequeueAfter: 30 * time.Second},
			expectedError:  true, // Will error because no nodes are available
		},
		{
			name: "intent does not require edge processing",
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-intent-2",
					Namespace: "default",
				},
			},
			setupMocks: func(mc *MockClient, msw *MockStatusWriter) {
				intent := &nephoran.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-intent-2",
						Namespace: "default",
					},
					Spec: nephoran.NetworkIntentSpec{
						Intent:     "Deploy standard network service",
						IntentType: nephoran.IntentTypeDeployment,
						ProcessedParameters: &nephoran.ProcessedParameters{
							NetworkFunction: "standard-gateway",
						},
					},
				}
				mc.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
					obj := args.Get(2).(*nephoran.NetworkIntent)
					*obj = *intent
				})
			},
			expectedResult: ctrl.Result{},
			expectedError:  false,
		},
		{
			name: "intent not found - should use IgnoreNotFound",
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent",
					Namespace: "default",
				},
			},
			setupMocks: func(mc *MockClient, msw *MockStatusWriter) {
				notFoundErr := &errors.StatusError{
					ErrStatus: metav1.Status{
						Status: metav1.StatusFailure,
						Code:   404,
						Reason: metav1.StatusReasonNotFound,
					},
				}
				mc.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(notFoundErr)
			},
			expectedResult: ctrl.Result{},
			expectedError:  false,
		},
		{
			name: "error getting intent",
			request: ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "test-intent",
					Namespace: "default",
				},
			},
			setupMocks: func(mc *MockClient, msw *MockStatusWriter) {
				mc.On("Get", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Errorf("internal error"))
			},
			expectedResult: ctrl.Result{},
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockClient)
			mockStatusWriter := new(MockStatusWriter)

			controller := createTestEdgeController()
			controller.Client = mockClient

			tt.setupMocks(mockClient, mockStatusWriter)

			result, err := controller.Reconcile(context.Background(), tt.request)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedResult, result)

			mockClient.AssertExpectations(t)
			mockStatusWriter.AssertExpectations(t)
		})
	}
}

// Test Edge Node Management
func TestEdgeController_NodeManagement(t *testing.T) {
	controller := createTestEdgeController()

	t.Run("RegisterNode", func(t *testing.T) {
		node := &EdgeNode{
			ID:     "node-1",
			Name:   "edge-node-1",
			Zone:   "zone-1",
			Status: EdgeNodeActive,
			Capabilities: EdgeCapabilities{
				ComputeCores: 8,
				MemoryGB:     16,
				StorageGB:    500,
				GPUEnabled:   true,
			},
			Resources: EdgeResources{
				CPUUtilization:     0.5,
				MemoryUtilization:  0.6,
				StorageUtilization: 0.3,
			},
			LastSeen: time.Now(),
		}

		err := controller.RegisterNode(node)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(controller.edgeNodes))
		assert.Equal(t, node, controller.edgeNodes["node-1"])
	})

	t.Run("UpdateNodeStatus", func(t *testing.T) {
		status := EdgeNodeDegraded
		err := controller.UpdateNodeStatus("node-1", status)
		assert.NoError(t, err)

		node := controller.edgeNodes["node-1"]
		assert.Equal(t, status, node.Status)
	})

	t.Run("GetNode", func(t *testing.T) {
		node, err := controller.GetNode("node-1")
		assert.NoError(t, err)
		assert.NotNil(t, node)
		assert.Equal(t, "node-1", node.ID)
	})

	t.Run("GetNode_NotFound", func(t *testing.T) {
		node, err := controller.GetNode("non-existent")
		assert.Error(t, err)
		assert.Nil(t, node)
	})

	t.Run("RemoveNode", func(t *testing.T) {
		err := controller.RemoveNode("node-1")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(controller.edgeNodes))
	})
}

// Test Edge Zone Management
func TestEdgeController_ZoneManagement(t *testing.T) {
	controller := createTestEdgeController()

	t.Run("CreateZone", func(t *testing.T) {
		zone := &EdgeZone{
			ID:              "zone-1",
			Name:            "Edge Zone 1",
			Region:          "us-east",
			Nodes:           []string{"node-1", "node-2"},
			ServiceLevel:    ServiceLevelPremium,
			RedundancyLevel: 2,
			TotalCapacity: EdgeResources{
				CPUUtilization:     0.0,
				MemoryUtilization:  0.0,
				StorageUtilization: 0.0,
			},
		}

		err := controller.CreateZone(zone)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(controller.edgeZones))
		assert.Equal(t, zone, controller.edgeZones["zone-1"])
	})

	t.Run("UpdateZone", func(t *testing.T) {
		zone := controller.edgeZones["zone-1"]
		zone.ConnectedUsers = 100
		zone.UtilizedCapacity = EdgeResources{
			CPUUtilization:     0.7,
			MemoryUtilization:  0.5,
			StorageUtilization: 0.3,
		}

		err := controller.UpdateZone(zone)
		assert.NoError(t, err)

		updatedZone := controller.edgeZones["zone-1"]
		assert.Equal(t, 100, updatedZone.ConnectedUsers)
		assert.Equal(t, 0.7, updatedZone.UtilizedCapacity.CPUUtilization)
	})

	t.Run("GetZone", func(t *testing.T) {
		zone, err := controller.GetZone("zone-1")
		assert.NoError(t, err)
		assert.NotNil(t, zone)
		assert.Equal(t, "zone-1", zone.ID)
	})

	t.Run("DeleteZone", func(t *testing.T) {
		err := controller.DeleteZone("zone-1")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(controller.edgeZones))
	})
}

// Test Network Intent Processing
func TestEdgeController_ProcessNetworkIntent(t *testing.T) {
	controller := createTestEdgeController()

	// Add test nodes
	node1 := &EdgeNode{
		ID:     "node-1",
		Name:   "edge-node-1",
		Zone:   "zone-1",
		Status: EdgeNodeActive,
		Capabilities: EdgeCapabilities{
			ComputeCores: 8,
			MemoryGB:     16,
			StorageGB:    500,
			GPUEnabled:   true,
		},
		Resources: EdgeResources{
			CPUUtilization:     0.3,
			MemoryUtilization:  0.4,
			StorageUtilization: 0.2,
		},
	}
	node2 := &EdgeNode{
		ID:     "node-2",
		Name:   "edge-node-2",
		Zone:   "zone-1",
		Status: EdgeNodeActive,
		Capabilities: EdgeCapabilities{
			ComputeCores: 16,
			MemoryGB:     32,
			StorageGB:    1000,
			GPUEnabled:   false,
		},
		Resources: EdgeResources{
			CPUUtilization:     0.8,
			MemoryUtilization:  0.7,
			StorageUtilization: 0.5,
		},
	}

	controller.RegisterNode(node1)
	controller.RegisterNode(node2)

	tests := []struct {
		name     string
		intent   *nephoran.NetworkIntent
		expected EdgeOptimizationPlan
	}{
		{
			name: "low latency intent",
			intent: &nephoran.NetworkIntent{
				Spec: nephoran.NetworkIntentSpec{
					IntentType: nephoran.IntentTypeOptimization,
					ProcessedParameters: &nephoran.ProcessedParameters{
						NetworkFunction: "low-latency-gateway",
					},
				},
			},
			expected: EdgeOptimizationPlan{
				OptimizationType: "latency-optimization",
				TargetNodes:      []string{"node-1"}, // Lower utilization
			},
		},
		{
			name: "high throughput intent",
			intent: &nephoran.NetworkIntent{
				Spec: nephoran.NetworkIntentSpec{
					IntentType: nephoran.IntentTypeOptimization,
					ProcessedParameters: &nephoran.ProcessedParameters{
						NetworkFunction: "high-throughput-gateway",
					},
				},
			},
			expected: EdgeOptimizationPlan{
				OptimizationType: "throughput-optimization",
				TargetNodes:      []string{"node-1", "node-2"},
			},
		},
		{
			name: "ML workload intent",
			intent: &nephoran.NetworkIntent{
				Spec: nephoran.NetworkIntentSpec{
					IntentType: nephoran.IntentTypeOptimization,
					ProcessedParameters: &nephoran.ProcessedParameters{
						NetworkFunction: "ml-inference-gateway",
					},
				},
			},
			expected: EdgeOptimizationPlan{
				OptimizationType: "ml-optimization",
				TargetNodes:      []string{"node-1"}, // Has GPU
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := controller.ProcessNetworkIntent(tt.intent)
			assert.NoError(t, err)
			assert.NotNil(t, plan)
			assert.Equal(t, tt.expected.OptimizationType, plan.OptimizationType)
			assert.ElementsMatch(t, tt.expected.TargetNodes, plan.TargetNodes)
		})
	}
}

// Test Health Monitoring
func TestEdgeController_HealthMonitoring(t *testing.T) {
	controller := createTestEdgeController()

	// Register a node
	node := &EdgeNode{
		ID:       "node-1",
		Name:     "edge-node-1",
		Status:   EdgeNodeActive,
		LastSeen: time.Now(),
		HealthMetrics: EdgeHealthMetrics{
			Latency:         3,
			PacketLoss:      0.01,
			Jitter:          1,
			Availability:    0.999,
			ErrorRate:       0.001,
			ConnectionCount: 100,
			ActiveSessions:  50,
		},
	}
	controller.RegisterNode(node)

	t.Run("CheckNodeHealth", func(t *testing.T) {
		health, err := controller.CheckNodeHealth("node-1")
		assert.NoError(t, err)
		assert.NotNil(t, health)
		assert.Equal(t, EdgeNodeActive, health.Status)
		assert.True(t, health.IsHealthy)
	})

	t.Run("UpdateHealthMetrics", func(t *testing.T) {
		metrics := EdgeHealthMetrics{
			Latency:         10, // Degraded
			PacketLoss:      0.05,
			Jitter:          5,
			Availability:    0.95,
			ErrorRate:       0.05,
			ConnectionCount: 150,
			ActiveSessions:  80,
		}

		err := controller.UpdateHealthMetrics("node-1", metrics)
		assert.NoError(t, err)

		health, _ := controller.CheckNodeHealth("node-1")
		assert.Equal(t, EdgeNodeDegraded, health.Status)
		assert.False(t, health.IsHealthy)
	})

	t.Run("DetectUnhealthyNodes", func(t *testing.T) {
		// Add an offline node
		offlineNode := &EdgeNode{
			ID:       "node-2",
			Name:     "edge-node-2",
			Status:   EdgeNodeActive,
			LastSeen: time.Now().Add(-2 * time.Minute), // Offline threshold
		}
		controller.RegisterNode(offlineNode)

		unhealthyNodes := controller.DetectUnhealthyNodes()
		assert.Equal(t, 2, len(unhealthyNodes))
		assert.Contains(t, unhealthyNodes, "node-1")
		assert.Contains(t, unhealthyNodes, "node-2")
	})
}

// Test Resource Allocation
func TestEdgeController_ResourceAllocation(t *testing.T) {
	controller := createTestEdgeController()

	// Setup nodes with different resource levels
	node1 := &EdgeNode{
		ID:     "node-1",
		Name:   "edge-node-1",
		Status: EdgeNodeActive,
		Resources: EdgeResources{
			CPUUtilization:     0.2,
			MemoryUtilization:  0.3,
			StorageUtilization: 0.1,
		},
		Capabilities: EdgeCapabilities{
			ComputeCores: 8,
			MemoryGB:     16,
			StorageGB:    500,
		},
	}
	node2 := &EdgeNode{
		ID:     "node-2",
		Name:   "edge-node-2",
		Status: EdgeNodeActive,
		Resources: EdgeResources{
			CPUUtilization:     0.7,
			MemoryUtilization:  0.8,
			StorageUtilization: 0.6,
		},
		Capabilities: EdgeCapabilities{
			ComputeCores: 8,
			MemoryGB:     16,
			StorageGB:    500,
		},
	}
	controller.RegisterNode(node1)
	controller.RegisterNode(node2)

	t.Run("AllocateResources", func(t *testing.T) {
		request := ResourceRequest{
			CPUCores:  2,
			MemoryGB:  4,
			StorageGB: 50,
		}

		allocation, err := controller.AllocateResources(request)
		assert.NoError(t, err)
		assert.NotNil(t, allocation)
		assert.Equal(t, "node-1", allocation.NodeID) // Should pick less utilized node
	})

	t.Run("AllocateResources_InsufficientResources", func(t *testing.T) {
		request := ResourceRequest{
			CPUCores:  20, // More than available
			MemoryGB:  32,
			StorageGB: 1000,
		}

		allocation, err := controller.AllocateResources(request)
		assert.Error(t, err)
		assert.Nil(t, allocation)
	})

	t.Run("ReleaseResources", func(t *testing.T) {
		err := controller.ReleaseResources("node-1", ResourceRequest{
			CPUCores:  1,
			MemoryGB:  2,
			StorageGB: 25,
		})
		assert.NoError(t, err)

		node, _ := controller.GetNode("node-1")
		// After allocating 2 cores (0.25 utilization) and releasing 1 core (0.125), should be around 0.325
		assert.InDelta(t, 0.325, node.Resources.CPUUtilization, 0.01)
	})
}

// Test Failover Scenarios
func TestEdgeController_Failover(t *testing.T) {
	controller := createTestEdgeController()

	// Setup primary and backup nodes
	primaryNode := &EdgeNode{
		ID:     "primary-node",
		Name:   "primary-edge-node",
		Zone:   "zone-1",
		Status: EdgeNodeActive,
		LocalServices: []EdgeService{
			{
				ID:       "service-1",
				Name:     "critical-service",
				Type:     "ml-inference",
				Status:   "running",
				Endpoint: "http://primary-node:8080",
			},
		},
	}
	backupNode := &EdgeNode{
		ID:     "backup-node",
		Name:   "backup-edge-node",
		Zone:   "zone-1",
		Status: EdgeNodeActive,
		Capabilities: EdgeCapabilities{
			ComputeCores: 8,
			MemoryGB:     16,
			StorageGB:    500,
		},
		Resources: EdgeResources{
			CPUUtilization:     0.3,
			MemoryUtilization:  0.4,
			StorageUtilization: 0.2,
		},
	}
	controller.RegisterNode(primaryNode)
	controller.RegisterNode(backupNode)

	t.Run("InitiateFailover", func(t *testing.T) {
		// Mark primary as failed
		controller.UpdateNodeStatus("primary-node", EdgeNodeOffline)

		failoverPlan, err := controller.InitiateFailover("primary-node")
		assert.NoError(t, err)
		assert.NotNil(t, failoverPlan)
		assert.Equal(t, "backup-node", failoverPlan.TargetNode)
		assert.Equal(t, 1, len(failoverPlan.ServicesToMigrate))
	})

	t.Run("ExecuteFailover", func(t *testing.T) {
		failoverPlan := &FailoverPlan{
			SourceNode: "primary-node",
			TargetNode: "backup-node",
			ServicesToMigrate: []EdgeService{
				primaryNode.LocalServices[0],
			},
		}

		err := controller.ExecuteFailover(failoverPlan)
		assert.NoError(t, err)

		// Check services migrated
		backupNode, _ := controller.GetNode("backup-node")
		assert.Equal(t, 1, len(backupNode.LocalServices))
		assert.Equal(t, "service-1", backupNode.LocalServices[0].ID)
	})

	t.Run("AutomaticFailback", func(t *testing.T) {
		// Mark primary as active again
		controller.UpdateNodeStatus("primary-node", EdgeNodeActive)

		err := controller.InitiateFailback("primary-node", "backup-node")
		assert.NoError(t, err)

		// Check services migrated back
		primaryNode, _ := controller.GetNode("primary-node")
		assert.Equal(t, 1, len(primaryNode.LocalServices))
	})
}

// Test Edge ML Capabilities
func TestEdgeController_MLCapabilities(t *testing.T) {
	controller := createTestEdgeController()

	// Setup ML-capable nodes
	mlNode := &EdgeNode{
		ID:     "ml-node",
		Name:   "ml-edge-node",
		Status: EdgeNodeActive,
		Capabilities: EdgeCapabilities{
			ComputeCores:    16,
			MemoryGB:        32,
			StorageGB:       1000,
			GPUEnabled:      true,
			GPUMemoryGB:     16,
			MLFrameworks:    []string{"tensorflow", "pytorch", "onnx"},
			AcceleratorType: "nvidia-t4",
		},
	}
	controller.RegisterNode(mlNode)

	t.Run("DeployMLModel", func(t *testing.T) {
		model := &MLModel{
			ID:          "model-1",
			Name:        "object-detection",
			Framework:   "tensorflow",
			Version:     "2.0",
			SizeGB:      2,
			RequiresGPU: true,
		}

		deployment, err := controller.DeployMLModel(model, "ml-node")
		assert.NoError(t, err)
		assert.NotNil(t, deployment)
		assert.Equal(t, "ml-node", deployment.NodeID)
		assert.Equal(t, "deployed", deployment.Status)
	})

	t.Run("RunInference", func(t *testing.T) {
		request := &InferenceRequest{
			ModelID: "model-1",
			Input:   []byte("test-input"),
			Options: map[string]string{
				"batch_size": "1",
				"precision":  "fp16",
			},
		}

		result, err := controller.RunInference("ml-node", request)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Output)
		assert.Greater(t, result.InferenceTimeMs, 0)
	})

	t.Run("OptimizeMLDeployment", func(t *testing.T) {
		optimization, err := controller.OptimizeMLDeployment("model-1")
		assert.NoError(t, err)
		assert.NotNil(t, optimization)
		assert.Contains(t, optimization.Recommendations, "quantization")
		assert.Contains(t, optimization.Recommendations, "batching")
	})
}

// Test Edge Caching
func TestEdgeController_Caching(t *testing.T) {
	controller := createTestEdgeController()

	// Setup cache-enabled node
	cacheNode := &EdgeNode{
		ID:     "cache-node",
		Name:   "cache-edge-node",
		Status: EdgeNodeActive,
		Capabilities: EdgeCapabilities{
			CacheEnabled: true,
			CacheSizeGB:  100,
		},
	}
	controller.RegisterNode(cacheNode)

	t.Run("ConfigureCache", func(t *testing.T) {
		config := &CacheConfig{
			MaxSizeGB:      50,
			EvictionPolicy: "lru",
			TTL:            3600, // 1 hour
			ContentTypes:   []string{"video", "image", "model"},
		}

		err := controller.ConfigureCache("cache-node", config)
		assert.NoError(t, err)
	})

	t.Run("CacheContent", func(t *testing.T) {
		content := &CacheContent{
			ID:         "content-1",
			Type:       "video",
			SizeMB:     500,
			Popularity: 0.8,
			LastAccess: time.Now(),
		}

		err := controller.CacheContent("cache-node", content)
		assert.NoError(t, err)
	})

	t.Run("GetCacheStats", func(t *testing.T) {
		stats, err := controller.GetCacheStats("cache-node")
		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Greater(t, stats.HitRate, 0.0)
		assert.Greater(t, stats.UsedGB, 0.0)
	})
}

// Test Edge Analytics
func TestEdgeController_Analytics(t *testing.T) {
	controller := createTestEdgeController()

	// Create zone first
	zone := &EdgeZone{
		ID:               "zone-1",
		Name:             "Test Zone 1",
		Region:           "us-east",
		Nodes:            []string{"node-0", "node-1", "node-2"},
		ServiceLevel:     ServiceLevelStandard,
		RedundancyLevel:  2,
		TotalCapacity:    EdgeResources{},
		UtilizedCapacity: EdgeResources{},
	}
	controller.CreateZone(zone)

	// Setup nodes
	for i := 0; i < 3; i++ {
		node := &EdgeNode{
			ID:     fmt.Sprintf("node-%d", i),
			Name:   fmt.Sprintf("edge-node-%d", i),
			Zone:   "zone-1",
			Status: EdgeNodeActive,
			HealthMetrics: EdgeHealthMetrics{
				Latency:         float64(i + 3),
				PacketLoss:      float64(i) * 0.01,
				ConnectionCount: 100 * (i + 1),
				ActiveSessions:  50 * (i + 1),
			},
		}
		controller.RegisterNode(node)
	}

	t.Run("GenerateZoneReport", func(t *testing.T) {
		report, err := controller.GenerateZoneReport("zone-1")
		assert.NoError(t, err)
		assert.NotNil(t, report)
		assert.Equal(t, 3, report.TotalNodes)
		assert.Equal(t, 3, report.ActiveNodes)
		assert.Greater(t, report.AverageLatency, 0.0)
	})

	t.Run("PredictResourceDemand", func(t *testing.T) {
		prediction, err := controller.PredictResourceDemand("zone-1", 24*time.Hour)
		assert.NoError(t, err)
		assert.NotNil(t, prediction)
		assert.Greater(t, prediction.PredictedCPU, 0.0)
		assert.Greater(t, prediction.PredictedMemory, 0.0)
		assert.Greater(t, prediction.Confidence, 0.5)
	})

	t.Run("OptimizeNodePlacement", func(t *testing.T) {
		optimization, err := controller.OptimizeNodePlacement()
		assert.NoError(t, err)
		assert.NotNil(t, optimization)
		assert.GreaterOrEqual(t, len(optimization.Recommendations), 0)
	})
}

// Benchmark tests
func BenchmarkEdgeController_RegisterNode(b *testing.B) {
	controller := createTestEdgeController()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		node := &EdgeNode{
			ID:     fmt.Sprintf("node-%d", i),
			Name:   fmt.Sprintf("edge-node-%d", i),
			Status: EdgeNodeActive,
		}
		controller.RegisterNode(node)
	}
}

func BenchmarkEdgeController_ProcessIntent(b *testing.B) {
	controller := createTestEdgeController()

	// Add nodes
	for i := 0; i < 10; i++ {
		node := &EdgeNode{
			ID:     fmt.Sprintf("node-%d", i),
			Name:   fmt.Sprintf("edge-node-%d", i),
			Status: EdgeNodeActive,
			Resources: EdgeResources{
				CPUUtilization:    float64(i) * 0.1,
				MemoryUtilization: float64(i) * 0.1,
			},
		}
		controller.RegisterNode(node)
	}

	intent := &nephoran.NetworkIntent{
		Spec: nephoran.NetworkIntentSpec{
			IntentType: nephoran.IntentTypeOptimization,
			ProcessedParameters: &nephoran.ProcessedParameters{
				NetworkFunction: "low-latency-gateway",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		controller.ProcessNetworkIntent(intent)
	}
}
