package edge

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ctrlclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

func TestEdgeController_NodeDiscovery(t *testing.T) {
	testCases := []struct {
		name              string
		initialNodes      map[string]*EdgeNode
		expectedNodeCount int
		expectedZoneCount int
		description       string
	}{
		{
			name:              "discover_initial_nodes",
			initialNodes:      map[string]*EdgeNode{},
			expectedNodeCount: 2, // Based on simulateEdgeNodeDiscovery
			expectedZoneCount: 2,
			description:       "Should discover initial edge nodes and create zones",
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
					AverageLatency: 1.0,   // Low latency
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

// Helper function to check if a string contains any of the given substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}
