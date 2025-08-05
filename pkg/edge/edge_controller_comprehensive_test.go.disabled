package edge

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

func TestEdgeController_Reconcile(t *testing.T) {
	testCases := []struct {
		name             string
		networkIntent    *nephoranv1.NetworkIntent
		existingObjects  []ctrlclient.Object
		edgeNodesSetup   func(*EdgeController)
		expectedResult   reconcile.Result
		expectedError    bool
		expectedPhase    string
		requiresEdge     bool
		description      string
	}{
		{
			name: "successful_edge_deployment_urllc",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "urllc-intent",
					Namespace: "default",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy ultra-low latency URLLC service for autonomous vehicles with <1ms latency",
				},
			},
			existingObjects: []ctrlclient.Object{},
			edgeNodesSetup: func(ec *EdgeController) {
				// Setup edge nodes with URLLC capabilities
				ec.edgeNodes["edge-node-1"] = &EdgeNode{
					ID:     "edge-node-1",
					Name:   "Metro Edge Node 1",
					Zone:   "metro-zone-1",
					Status: EdgeNodeActive,
					Capabilities: EdgeCapabilities{
						LowLatencyProcessing: true,
						LocalRICSupport:      true,
						ComputeIntensive:     true,
					},
					HealthMetrics: EdgeHealthMetrics{
						AverageLatency:   1.5,
						UptimePercent:    99.95,
						ThroughputMbps:   1000,
						Availability:     0.9995,
					},
					Resources: EdgeResources{
						Utilization: ResourceUtilization{
							CPUPercent: 60.0,
						},
					},
				}
			},
			expectedResult: reconcile.Result{RequeueAfter: 5 * time.Minute},
			expectedError:  false,
			expectedPhase:  "Deployed",
			requiresEdge:   true,
			description:    "Should successfully deploy URLLC intent to edge nodes",
		},
		{
			name: "edge_deployment_iot_gateway",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "iot-intent",
					Namespace: "default",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy IoT gateway for industrial sensors with local processing",
				},
			},
			existingObjects: []ctrlclient.Object{},
			edgeNodesSetup: func(ec *EdgeController) {
				ec.edgeNodes["edge-node-2"] = &EdgeNode{
					ID:     "edge-node-2",
					Name:   "Industrial Edge Node",
					Zone:   "industrial-zone-1",
					Status: EdgeNodeActive,
					Capabilities: EdgeCapabilities{
						IoTGateway:       true,
						CachingSupport:   true,
						VideoProcessing:  false,
					},
					HealthMetrics: EdgeHealthMetrics{
						AverageLatency: 5.0,
						UptimePercent:  99.8,
						Availability:   0.998,
					},
					Resources: EdgeResources{
						Utilization: ResourceUtilization{
							CPUPercent: 40.0,
						},
					},
				}
			},
			expectedResult: reconcile.Result{RequeueAfter: 5 * time.Minute},
			expectedError:  false,
			expectedPhase:  "Deployed",
			requiresEdge:   true,
			description:    "Should successfully deploy IoT intent to edge nodes with IoT capabilities",
		},
		{
			name: "edge_deployment_ai_ml",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ai-intent",
					Namespace: "default",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy AI inference for real-time video analytics at the edge",
				},
			},
			existingObjects: []ctrlclient.Object{},
			edgeNodesSetup: func(ec *EdgeController) {
				ec.edgeNodes["edge-node-3"] = &EdgeNode{
					ID:     "edge-node-3",
					Name:   "AI Edge Node",
					Zone:   "ai-zone-1",
					Status: EdgeNodeActive,
					Capabilities: EdgeCapabilities{
						ComputeIntensive:     true,
						VideoProcessing:      true,
						GPUEnabled:           true,
						AcceleratorTypes:     []string{"GPU", "FPGA"},
						MLFrameworks:         []string{"TensorFlow", "PyTorch"},
					},
					HealthMetrics: EdgeHealthMetrics{
						AverageLatency: 3.0,
						UptimePercent:  99.9,
						Availability:   0.999,
					},
					Resources: EdgeResources{
						GPU: 2,
						Utilization: ResourceUtilization{
							CPUPercent: 70.0,
						},
					},
				}
			},
			expectedResult: reconcile.Result{RequeueAfter: 5 * time.Minute},
			expectedError:  false,
			expectedPhase:  "Deployed",
			requiresEdge:   true,
			description:    "Should successfully deploy AI/ML intent to edge nodes with AI capabilities",
		},
		{
			name: "no_edge_required_intent",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-intent",
					Namespace: "default",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy standard cloud service with normal latency requirements",
				},
			},
			existingObjects: []ctrlclient.Object{},
			edgeNodesSetup:  func(ec *EdgeController) {},
			expectedResult:  reconcile.Result{},
			expectedError:   false,
			expectedPhase:   "", // Phase should not be updated
			requiresEdge:    false,
			description:     "Should skip processing for intents that don't require edge deployment",
		},
		{
			name: "no_suitable_edge_nodes",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "urllc-intent",
					Namespace: "default",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy URLLC service with ultra-low latency requirements",
				},
			},
			existingObjects: []ctrlclient.Object{},
			edgeNodesSetup: func(ec *EdgeController) {
				// Setup edge nodes without URLLC capabilities or that are offline
				ec.edgeNodes["edge-node-offline"] = &EdgeNode{
					ID:     "edge-node-offline",
					Name:   "Offline Edge Node",
					Zone:   "test-zone",
					Status: EdgeNodeOffline, // Offline node
					Capabilities: EdgeCapabilities{
						LowLatencyProcessing: true,
					},
					HealthMetrics: EdgeHealthMetrics{
						AverageLatency: 15.0, // Too high latency
					},
				}
			},
			expectedResult: reconcile.Result{RequeueAfter: 30 * time.Second},
			expectedError:  true,
			expectedPhase:  "",
			requiresEdge:   true,
			description:    "Should handle case when no suitable edge nodes are available",
		},
		{
			name: "edge_node_overloaded",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "edge-intent",
					Namespace: "default",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy local processing service at the edge",
				},
			},
			existingObjects: []ctrlclient.Object{},
			edgeNodesSetup: func(ec *EdgeController) {
				ec.edgeNodes["edge-node-overloaded"] = &EdgeNode{
					ID:     "edge-node-overloaded",
					Name:   "Overloaded Edge Node",
					Zone:   "test-zone",
					Status: EdgeNodeActive,
					Capabilities: EdgeCapabilities{
						LowLatencyProcessing: true,
					},
					HealthMetrics: EdgeHealthMetrics{
						AverageLatency: 2.0,
						UptimePercent:  99.0,
						Availability:   0.99,
					},
					Resources: EdgeResources{
						Utilization: ResourceUtilization{
							CPUPercent: 95.0, // Overloaded
						},
					},
				}
			},
			expectedResult: reconcile.Result{RequeueAfter: 30 * time.Second},
			expectedError:  true,
			expectedPhase:  "",
			requiresEdge:   true,
			description:    "Should handle case when edge nodes are overloaded",
		},
		{
			name: "resource_not_found",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "nonexistent-intent",
					Namespace: "default",
				},
			},
			existingObjects: []ctrlclient.Object{}, // Don't add the NetworkIntent
			edgeNodesSetup:  func(ec *EdgeController) {},
			expectedResult:  reconcile.Result{},
			expectedError:   false,
			expectedPhase:   "",
			requiresEdge:    false,
			description:     "Should handle resource not found gracefully",
		},
		{
			name: "multiple_edge_zones_deployment",
			networkIntent: &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "multi-zone-intent",
					Namespace: "default",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy real-time AR/VR service across multiple edge zones with <5ms latency",
				},
			},
			existingObjects: []ctrlclient.Object{},
			edgeNodesSetup: func(ec *EdgeController) {
				ec.edgeNodes["edge-node-zone1"] = &EdgeNode{
					ID:     "edge-node-zone1",
					Name:   "Zone 1 Edge Node",
					Zone:   "zone-1",
					Status: EdgeNodeActive,
					Capabilities: EdgeCapabilities{
						LowLatencyProcessing: true,
						VideoProcessing:      true,
					},
					HealthMetrics: EdgeHealthMetrics{
						AverageLatency: 2.5,
						UptimePercent:  99.9,
						Availability:   0.999,
					},
					Resources: EdgeResources{
						Utilization: ResourceUtilization{
							CPUPercent: 50.0,
						},
					},
				}
				ec.edgeNodes["edge-node-zone2"] = &EdgeNode{
					ID:     "edge-node-zone2",
					Name:   "Zone 2 Edge Node",
					Zone:   "zone-2",
					Status: EdgeNodeActive,
					Capabilities: EdgeCapabilities{
						LowLatencyProcessing: true,
						VideoProcessing:      true,
					},
					HealthMetrics: EdgeHealthMetrics{
						AverageLatency: 3.0,
						UptimePercent:  99.8,
						Availability:   0.998,
					},
					Resources: EdgeResources{
						Utilization: ResourceUtilization{
							CPUPercent: 60.0,
						},
					},
				}
			},
			expectedResult: reconcile.Result{RequeueAfter: 5 * time.Minute},
			expectedError:  false,
			expectedPhase:  "Deployed",
			requiresEdge:   true,
			description:    "Should successfully deploy to multiple edge zones",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the scheme
			s := runtime.NewScheme()
			err := nephoranv1.AddToScheme(s)
			require.NoError(t, err)

			// Add objects to existing objects
			objects := tc.existingObjects
			if tc.name != "resource_not_found" {
				objects = append(objects, tc.networkIntent)
			}

			// Create fake clients
			fakeClient := ctrlclientfake.NewClientBuilder().
				WithScheme(s).
				WithObjects(objects...).
				WithStatusSubresource(&nephoranv1.NetworkIntent{}).
				Build()

			fakeKubeClient := fake.NewSimpleClientset()

			// Create edge controller with test configuration
			config := &EdgeControllerConfig{
				NodeDiscoveryEnabled:    true,
				DiscoveryInterval:       30 * time.Second,
				NodeHealthCheckInterval: 10 * time.Second,
				MaxLatencyMs:            5,
				EdgeResourceThreshold:   0.8,
				EdgeFailoverEnabled:     true,
			}

			controller := NewEdgeController(
				fakeClient,
				fakeKubeClient,
				log.Log.WithName("edge-controller-test"),
				s,
				config,
			)

			// Apply edge nodes setup
			tc.edgeNodesSetup(controller)

			// Execute reconcile
			ctx := context.Background()
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      tc.networkIntent.Name,
					Namespace: tc.networkIntent.Namespace,
				},
			}

			result, err := controller.Reconcile(ctx, req)

			// Verify results
			if tc.expectedError {
				assert.Error(t, err, tc.description)
			} else {
				assert.NoError(t, err, tc.description)
			}

			assert.Equal(t, tc.expectedResult, result, tc.description)

			// Verify edge processing requirement detection
			if tc.name != "resource_not_found" {
				requiresEdge := controller.requiresEdgeProcessing(tc.networkIntent)
				assert.Equal(t, tc.requiresEdge, requiresEdge, "Edge processing requirement detection should match expected")
			}

			// Verify status update if object exists and phase is expected
			if tc.name != "resource_not_found" && tc.expectedPhase != "" {
				var updatedIntent nephoranv1.NetworkIntent
				err = fakeClient.Get(ctx, req.NamespacedName, &updatedIntent)
				if err == nil {
					assert.Equal(t, tc.expectedPhase, updatedIntent.Status.Phase, "Phase should match expected")
				}
			}
		})
	}
}

func TestEdgeController_NodeDiscovery(t *testing.T) {
	testCases := []struct {
		name               string
		initialNodes       map[string]*EdgeNode
		expectedNodeCount  int
		expectedZoneCount  int
		description        string
	}{
		{
			name:               "discover_initial_nodes",
			initialNodes:       map[string]*EdgeNode{},
			expectedNodeCount:  2, // Based on simulateEdgeNodeDiscovery
			expectedZoneCount:  2,
			description:        "Should discover initial edge nodes and create zones",
		},
		{
			name: "update_existing_nodes",
			initialNodes: map[string]*EdgeNode{
				"edge-node-1": {
					ID:     "edge-node-1",
					Name:   "Existing Node",
					Zone:   "existing-zone",
					Status: EdgeNodeActive,
				},
			},
			expectedNodeCount: 2, // Should have both existing and discovered nodes
			expectedZoneCount: 2,
			description:       "Should update existing nodes and discover new ones",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := runtime.NewScheme()
			err := nephoranv1.AddToScheme(s)
			require.NoError(t, err)

			fakeClient := ctrlclientfake.NewClientBuilder().WithScheme(s).Build()
			fakeKubeClient := fake.NewSimpleClientset()

			config := &EdgeControllerConfig{
				NodeDiscoveryEnabled:    true,
				DiscoveryInterval:       30 * time.Second,
				NodeHealthCheckInterval: 10 * time.Second,
			}

			controller := NewEdgeController(
				fakeClient,
				fakeKubeClient,
				log.Log.WithName("edge-controller-test"),
				s,
				config,
			)

			// Set initial nodes
			controller.edgeNodes = tc.initialNodes

			// Run node discovery
			ctx := context.Background()
			err = controller.discoverEdgeNodes(ctx)
			require.NoError(t, err)

			// Verify node count
			nodes := controller.GetEdgeNodes()
			assert.Equal(t, tc.expectedNodeCount, len(nodes), tc.description)

			// Verify zone count
			zones := controller.GetEdgeZones()
			assert.Equal(t, tc.expectedZoneCount, len(zones), tc.description)
		})
	}
}

func TestEdgeController_NodeSuitability(t *testing.T) {
	testCases := []struct {
		name        string
		node        *EdgeNode
		intent      *nephoranv1.NetworkIntent
		config      *EdgeControllerConfig
		suitable    bool
		description string
	}{
		{
			name: "suitable_urllc_node",
			node: &EdgeNode{
				ID:     "urllc-node",
				Status: EdgeNodeActive,
				Capabilities: EdgeCapabilities{
					LowLatencyProcessing: true,
				},
				HealthMetrics: EdgeHealthMetrics{
					AverageLatency: 1.5,
				},
				Resources: EdgeResources{
					Utilization: ResourceUtilization{
						CPUPercent: 60.0,
					},
				},
			},
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy URLLC service with ultra-low latency",
				},
			},
			config: &EdgeControllerConfig{
				MaxLatencyMs:          5,
				EdgeResourceThreshold: 0.8,
			},
			suitable:    true,
			description: "Should identify node as suitable for URLLC",
		},
		{
			name: "unsuitable_high_latency",
			node: &EdgeNode{
				ID:     "high-latency-node",
				Status: EdgeNodeActive,
				Capabilities: EdgeCapabilities{
					LowLatencyProcessing: true,
				},
				HealthMetrics: EdgeHealthMetrics{
					AverageLatency: 10.0, // Too high
				},
				Resources: EdgeResources{
					Utilization: ResourceUtilization{
						CPUPercent: 50.0,
					},
				},
			},
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy URLLC service",
				},
			},
			config: &EdgeControllerConfig{
				MaxLatencyMs:          5,
				EdgeResourceThreshold: 0.8,
			},
			suitable:    false,
			description: "Should reject node with high latency",
		},
		{
			name: "unsuitable_overloaded",
			node: &EdgeNode{
				ID:     "overloaded-node",
				Status: EdgeNodeActive,
				Capabilities: EdgeCapabilities{
					LowLatencyProcessing: true,
				},
				HealthMetrics: EdgeHealthMetrics{
					AverageLatency: 2.0,
				},
				Resources: EdgeResources{
					Utilization: ResourceUtilization{
						CPUPercent: 90.0, // Overloaded
					},
				},
			},
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy edge service",
				},
			},
			config: &EdgeControllerConfig{
				MaxLatencyMs:          5,
				EdgeResourceThreshold: 0.8,
			},
			suitable:    false,
			description: "Should reject overloaded node",
		},
		{
			name: "unsuitable_offline_node",
			node: &EdgeNode{
				ID:     "offline-node",
				Status: EdgeNodeOffline, // Offline
				Capabilities: EdgeCapabilities{
					LowLatencyProcessing: true,
				},
				HealthMetrics: EdgeHealthMetrics{
					AverageLatency: 1.0,
				},
				Resources: EdgeResources{
					Utilization: ResourceUtilization{
						CPUPercent: 30.0,
					},
				},
			},
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy edge service",
				},
			},
			config: &EdgeControllerConfig{
				MaxLatencyMs:          5,
				EdgeResourceThreshold: 0.8,
			},
			suitable:    false,
			description: "Should reject offline node",
		},
		{
			name: "suitable_ai_node",
			node: &EdgeNode{
				ID:     "ai-node",
				Status: EdgeNodeActive,
				Capabilities: EdgeCapabilities{
					ComputeIntensive: true,
					GPUEnabled:       true,
				},
				HealthMetrics: EdgeHealthMetrics{
					AverageLatency: 3.0,
				},
				Resources: EdgeResources{
					GPU: 2,
					Utilization: ResourceUtilization{
						CPUPercent: 40.0,
					},
				},
			},
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy AI inference service for machine learning",
				},
			},
			config: &EdgeControllerConfig{
				MaxLatencyMs:          5,
				EdgeResourceThreshold: 0.8,
			},
			suitable:    true,
			description: "Should identify node as suitable for AI workloads",
		},
		{
			name: "unsuitable_no_ai_capability",
			node: &EdgeNode{
				ID:     "basic-node",
				Status: EdgeNodeActive,
				Capabilities: EdgeCapabilities{
					ComputeIntensive: false, // No AI capability
				},
				HealthMetrics: EdgeHealthMetrics{
					AverageLatency: 2.0,
				},
				Resources: EdgeResources{
					Utilization: ResourceUtilization{
						CPUPercent: 30.0,
					},
				},
			},
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy AI inference service for machine learning",
				},
			},
			config: &EdgeControllerConfig{
				MaxLatencyMs:          5,
				EdgeResourceThreshold: 0.8,
			},
			suitable:    false,
			description: "Should reject node without AI capabilities for AI workloads",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := runtime.NewScheme()
			fakeClient := ctrlclientfake.NewClientBuilder().WithScheme(s).Build()
			fakeKubeClient := fake.NewSimpleClientset()

			controller := NewEdgeController(
				fakeClient,
				fakeKubeClient,
				log.Log.WithName("edge-controller-test"),
				s,
				tc.config,
			)

			suitable := controller.isNodeSuitable(tc.node, tc.intent)
			assert.Equal(t, tc.suitable, suitable, tc.description)
		})
	}
}

func TestEdgeController_RequiresEdgeProcessing(t *testing.T) {
	testCases := []struct {
		name         string
		intent       *nephoranv1.NetworkIntent
		requiresEdge bool
		description  string
	}{
		{
			name: "urllc_requires_edge",
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy URLLC service with ultra-low latency requirements",
				},
			},
			requiresEdge: true,
			description:  "URLLC keyword should require edge processing",
		},
		{
			name: "iot_requires_edge",
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy IoT gateway for sensor data processing",
				},
			},
			requiresEdge: true,
			description:  "IoT keyword should require edge processing",
		},
		{
			name: "ar_vr_requires_edge",
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy AR/VR service for immersive experiences",
				},
			},
			requiresEdge: true,
			description:  "AR/VR keyword should require edge processing",
		},
		{
			name: "latency_requirement_requires_edge",
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy service with 1ms latency requirement",
				},
			},
			requiresEdge: true,
			description:  "1ms latency requirement should require edge processing",
		},
		{
			name: "real_time_requires_edge",
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy real-time video processing service",
				},
			},
			requiresEdge: true,
			description:  "Real-time keyword should require edge processing",
		},
		{
			name: "cloud_service_no_edge",
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy standard cloud service with normal requirements",
				},
			},
			requiresEdge: false,
			description:  "Standard cloud service should not require edge processing",
		},
		{
			name: "batch_processing_no_edge",
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy batch processing service for data analytics",
				},
			},
			requiresEdge: false,
			description:  "Batch processing should not require edge processing",
		},
		{
			name: "high_latency_no_edge",
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy service with 100ms latency requirement",
				},
			},
			requiresEdge: false,
			description:  "High latency requirement should not require edge processing",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := runtime.NewScheme()
			fakeClient := ctrlclientfake.NewClientBuilder().WithScheme(s).Build()
			fakeKubeClient := fake.NewSimpleClientset()

			config := &EdgeControllerConfig{}
			controller := NewEdgeController(
				fakeClient,
				fakeKubeClient,
				log.Log.WithName("edge-controller-test"),
				s,
				config,
			)

			requiresEdge := controller.requiresEdgeProcessing(tc.intent)
			assert.Equal(t, tc.requiresEdge, requiresEdge, tc.description)
		})
	}
}

func TestEdgeController_NodeScoring(t *testing.T) {
	testCases := []struct {
		name          string
		node          *EdgeNode
		intent        *nephoranv1.NetworkIntent
		expectedScore float64
		description   string
	}{
		{
			name: "high_score_optimal_node",
			node: &EdgeNode{
				ID: "optimal-node",
				HealthMetrics: EdgeHealthMetrics{
					AverageLatency: 1.0,  // Low latency
					UptimePercent:  99.99, // High uptime
				},
				Resources: EdgeResources{
					Utilization: ResourceUtilization{
						CPUPercent: 30.0, // Low utilization
					},
				},
				Capabilities: EdgeCapabilities{
					ComputeIntensive:     true,
					LowLatencyProcessing: true,
				},
			},
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy AI service with URLLC requirements",
				},
			},
			expectedScore: 154.999, // High score due to optimal conditions
			description:   "Should give high score to optimal node",
		},
		{
			name: "low_score_suboptimal_node",
			node: &EdgeNode{
				ID: "suboptimal-node",
				HealthMetrics: EdgeHealthMetrics{
					AverageLatency: 8.0,  // High latency
					UptimePercent:  95.0, // Lower uptime
				},
				Resources: EdgeResources{
					Utilization: ResourceUtilization{
						CPUPercent: 80.0, // High utilization
					},
				},
				Capabilities: EdgeCapabilities{
					ComputeIntensive:     false,
					LowLatencyProcessing: false,
				},
			},
			intent: &nephoranv1.NetworkIntent{
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: "Deploy standard service",
				},
			},
			expectedScore: 39.5, // Lower score due to suboptimal conditions
			description:   "Should give low score to suboptimal node",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := runtime.NewScheme()
			fakeClient := ctrlclientfake.NewClientBuilder().WithScheme(s).Build()
			fakeKubeClient := fake.NewSimpleClientset()

			config := &EdgeControllerConfig{}
			controller := NewEdgeController(
				fakeClient,
				fakeKubeClient,
				log.Log.WithName("edge-controller-test"),
				s,
				config,
			)

			score := controller.calculateNodeScore(tc.node, tc.intent)
			assert.InDelta(t, tc.expectedScore, score, 0.1, tc.description)
		})
	}
}

func TestEdgeController_HealthChecks(t *testing.T) {
	s := runtime.NewScheme()
	fakeClient := ctrlclientfake.NewClientBuilder().WithScheme(s).Build()
	fakeKubeClient := fake.NewSimpleClientset()

	config := &EdgeControllerConfig{
		NodeHealthCheckInterval: 1 * time.Second,
	}

	controller := NewEdgeController(
		fakeClient,
		fakeKubeClient,
		log.Log.WithName("edge-controller-test"),
		s,
		config,
	)

	// Add test nodes
	controller.edgeNodes["test-node-1"] = &EdgeNode{
		ID:     "test-node-1",
		Status: EdgeNodeActive,
		HealthMetrics: EdgeHealthMetrics{
			AverageLatency: 2.0,
		},
	}
	controller.edgeNodes["test-node-2"] = &EdgeNode{
		ID:     "test-node-2",
		Status: EdgeNodeActive,
		HealthMetrics: EdgeHealthMetrics{
			AverageLatency: 15.0, // High latency - should become degraded
		},
	}

	// Perform health checks
	ctx := context.Background()
	controller.performHealthChecks(ctx)

	// Verify health status updates
	nodes := controller.GetEdgeNodes()
	assert.Equal(t, EdgeNodeActive, nodes["test-node-1"].Status, "Node 1 should remain active")
	assert.Equal(t, EdgeNodeDegraded, nodes["test-node-2"].Status, "Node 2 should become degraded due to high latency")
}

func TestEdgeController_ZoneManagement(t *testing.T) {
	s := runtime.NewScheme()
	fakeClient := ctrlclientfake.NewClientBuilder().WithScheme(s).Build()
	fakeKubeClient := fake.NewSimpleClientset()

	config := &EdgeControllerConfig{
		ZoneRedundancyFactor:  2,
		EdgeResourceThreshold: 0.8,
	}

	controller := NewEdgeController(
		fakeClient,
		fakeKubeClient,
		log.Log.WithName("edge-controller-test"),
		s,
		config,
	)

	// Add nodes to different zones
	controller.edgeNodes["node-zone1-1"] = &EdgeNode{
		ID:   "node-zone1-1",
		Zone: "zone-1",
		Resources: EdgeResources{
			CPU:    8,
			Memory: 16 * 1024 * 1024 * 1024,
			Utilization: ResourceUtilization{
				CPUPercent: 60.0,
			},
		},
	}
	controller.edgeNodes["node-zone1-2"] = &EdgeNode{
		ID:   "node-zone1-2",
		Zone: "zone-1",
		Resources: EdgeResources{
			CPU:    4,
			Memory: 8 * 1024 * 1024 * 1024,
			Utilization: ResourceUtilization{
				CPUPercent: 80.0,
			},
		},
	}

	// Update edge zones
	controller.updateEdgeZones()

	// Verify zone creation and aggregation
	zones := controller.GetEdgeZones()
	assert.Contains(t, zones, "zone-1", "Zone 1 should be created")
	
	zone1 := zones["zone-1"]
	assert.Equal(t, 2, len(zone1.Nodes), "Zone 1 should have 2 nodes")
	assert.Equal(t, 12, zone1.TotalCapacity.CPU, "Zone 1 should have aggregated CPU capacity")
	assert.Equal(t, int64(24*1024*1024*1024), zone1.TotalCapacity.Memory, "Zone 1 should have aggregated memory capacity")

	// Test zone load optimization
	ctx := context.Background()
	controller.optimizeZoneLoad(ctx, "zone-1", zone1)
	// This should detect the 70% average utilization and potentially trigger scaling
}

// Helper function to check if a string contains any of the given substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}