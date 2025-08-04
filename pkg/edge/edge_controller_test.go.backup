package edge

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	return args.Get(0).(meta.RESTMapper)
}

func (m *MockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	args := m.Called(obj)
	return args.Get(0).(schema.GroupVersionKind), args.Error(1)
}

// MockStatusWriter implements client.StatusWriter for testing
type MockStatusWriter struct {
	mock.Mock
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
		NodeDiscoveryEnabled:      true,
		DiscoveryInterval:         30 * time.Second,
		NodeHealthCheckInterval:   10 * time.Second,
		AutoZoneCreation:          true,
		MaxNodesPerZone:           10,
		ZoneRedundancyFactor:      2,
		EnableLocalRIC:            true,
		EnableEdgeML:              true,
		EnableCaching:             true,
		LocalProcessingEnabled:    true,
		MaxLatencyMs:              5,
		MinBandwidthMbps:          100,
		EdgeResourceThreshold:     0.8,
		EdgeFailoverEnabled:       true,
		BackhaulFailoverEnabled:   true,
		LocalAutonomy:             true,
	}

	kubeClient := kubefake.NewSimpleClientset()
	logger := logr.Discard()
	scheme := runtime.NewScheme()

	return NewEdgeController(nil, kubeClient, logger, scheme, config)
}

func createTestNetworkIntent(name, intent string) *nephoran.NetworkIntent {
	return &nephoran.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: nephoran.NetworkIntentSpec{
			Intent: intent,
		},
		Status: nephoran.NetworkIntentStatus{
			Phase: "Pending",
		},
	}
}

func TestNewEdgeController(t *testing.T) {
	config := &EdgeControllerConfig{
		NodeDiscoveryEnabled: true,
		MaxLatencyMs:         5,
	}
	kubeClient := kubefake.NewSimpleClientset()
	logger := logr.Discard()
	scheme := runtime.NewScheme()

	controller := NewEdgeController(nil, kubeClient, logger, scheme, config)

	assert.NotNil(t, controller)
	assert.Equal(t, config, controller.config)
	assert.Equal(t, kubeClient, controller.KubeClient)
	assert.Equal(t, logger, controller.Log)
	assert.Equal(t, scheme, controller.Scheme)
	assert.NotNil(t, controller.edgeNodes)
	assert.NotNil(t, controller.edgeZones)
}

func TestRequiresEdgeProcessing(t *testing.T) {
	controller := createTestEdgeController()

	testCases := []struct {
		name        string
		intent      string
		expected    bool
		description string
	}{
		{
			name:        "URLLC intent",
			intent:      "Deploy URLLC service for autonomous vehicles",
			expected:    true,
			description: "Should detect URLLC requirement",
		},
		{
			name:        "Edge computing intent",
			intent:      "Deploy edge ML inference service",
			expected:    true,
			description: "Should detect edge keyword",
		},
		{
			name:        "Low latency intent",
			intent:      "Deploy service with 1ms latency requirement",
			expected:    true,
			description: "Should detect latency requirement",
		},
		{
			name:        "IoT gateway intent",
			intent:      "Deploy IoT gateway for industrial sensors",
			expected:    true,
			description: "Should detect IoT requirement",
		},
		{
			name:        "Regular intent",
			intent:      "Deploy backend service",
			expected:    false,
			description: "Should not require edge processing",
		},
		{
			name:        "Ultra-low latency intent",
			intent:      "Deploy ultra-low latency processing service",
			expected:    true,
			description: "Should detect ultra-low latency requirement",
		},
		{
			name:        "AR/VR intent",
			intent:      "Deploy AR/VR streaming service",
			expected:    true,
			description: "Should detect AR/VR requirement",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			intent := createTestNetworkIntent(tc.name, tc.intent)
			result := controller.requiresEdgeProcessing(intent)
			assert.Equal(t, tc.expected, result, tc.description)
		})
	}
}

func TestSimulateEdgeNodeDiscovery(t *testing.T) {
	controller := createTestEdgeController()

	nodes := controller.simulateEdgeNodeDiscovery()

	assert.NotEmpty(t, nodes)
	assert.Len(t, nodes, 2) // Based on the implementation

	// Check first node (Metro Edge Node)
	metroNode := nodes[0]
	assert.Equal(t, "edge-node-1", metroNode.ID)
	assert.Equal(t, "Metro Edge Node 1", metroNode.Name)
	assert.Equal(t, "metro-zone-1", metroNode.Zone)
	assert.Equal(t, EdgeNodeActive, metroNode.Status)
	assert.True(t, metroNode.Capabilities.ComputeIntensive)
	assert.True(t, metroNode.Capabilities.LowLatencyProcessing)
	assert.True(t, metroNode.Capabilities.LocalRICSupport)
	assert.True(t, metroNode.Capabilities.CachingSupport)
	assert.Contains(t, metroNode.Capabilities.NetworkFunctions, "CU")
	assert.Contains(t, metroNode.Capabilities.NetworkFunctions, "DU")
	assert.Contains(t, metroNode.Capabilities.NetworkFunctions, "Near-RT RIC")

	// Check second node (Access Edge Node)
	accessNode := nodes[1]
	assert.Equal(t, "edge-node-2", accessNode.ID)
	assert.Equal(t, "Access Edge Node 1", accessNode.Name)
	assert.Equal(t, "access-zone-1", accessNode.Zone)
	assert.Equal(t, EdgeNodeActive, accessNode.Status)
	assert.False(t, accessNode.Capabilities.ComputeIntensive)
	assert.True(t, accessNode.Capabilities.LowLatencyProcessing)
	assert.False(t, accessNode.Capabilities.LocalRICSupport)
	assert.True(t, accessNode.Capabilities.CachingSupport)
	assert.True(t, accessNode.Capabilities.IoTGateway)
	assert.Contains(t, accessNode.Capabilities.NetworkFunctions, "RU")

	// Check resource information
	assert.Equal(t, 16, metroNode.Resources.CPU)
	assert.Equal(t, int64(64*1024*1024*1024), metroNode.Resources.Memory)
	assert.Equal(t, 2, metroNode.Resources.GPU)
	assert.Equal(t, 10000, metroNode.Resources.NetworkBandwidth)

	assert.Equal(t, 4, accessNode.Resources.CPU)
	assert.Equal(t, int64(16*1024*1024*1024), accessNode.Resources.Memory)
	assert.Equal(t, 0, accessNode.Resources.GPU)
	assert.Equal(t, 1000, accessNode.Resources.NetworkBandwidth)

	// Check health metrics
	assert.Greater(t, metroNode.HealthMetrics.UptimePercent, 99.0)
	assert.Less(t, metroNode.HealthMetrics.AverageLatency, 2.0)
	assert.Greater(t, metroNode.HealthMetrics.ThroughputMbps, 700.0)

	// Check O-RAN functions for metro node
	assert.NotEmpty(t, metroNode.O_RANFunctions)
	ricFunction := metroNode.O_RANFunctions[0]
	assert.Equal(t, "Near-RT RIC", ricFunction.Type)
	assert.Equal(t, "1.0.0", ricFunction.Version)
	assert.Equal(t, "Active", ricFunction.Status)
	assert.Equal(t, 500.0, ricFunction.Metrics.ThroughputMbps)
	assert.Equal(t, 1.5, ricFunction.Metrics.LatencyMs)
	assert.Equal(t, 150, ricFunction.Metrics.ConnectedUEs)
}

func TestDiscoverEdgeNodes(t *testing.T) {
	controller := createTestEdgeController()
	ctx := context.Background()

	// Initially no nodes
	assert.Empty(t, controller.edgeNodes)
	assert.Empty(t, controller.edgeZones)

	err := controller.discoverEdgeNodes(ctx)
	assert.NoError(t, err)

	// Should have discovered nodes
	assert.NotEmpty(t, controller.edgeNodes)
	assert.NotEmpty(t, controller.edgeZones)

	// Check that nodes were added
	assert.Contains(t, controller.edgeNodes, "edge-node-1")
	assert.Contains(t, controller.edgeNodes, "edge-node-2")

	// Check zones were created
	assert.Contains(t, controller.edgeZones, "metro-zone-1")
	assert.Contains(t, controller.edgeZones, "access-zone-1")

	// Verify zone properties
	metroZone := controller.edgeZones["metro-zone-1"]
	assert.Equal(t, "metro-zone-1", metroZone.ID)
	assert.Equal(t, "us-east-1", metroZone.Region)
	assert.Contains(t, metroZone.Nodes, "edge-node-1")
	assert.Equal(t, 2, metroZone.RedundancyLevel)
}

func TestContainsFunction(t *testing.T) {
	testCases := []struct {
		name     string
		str      string
		substr   string
		expected bool
	}{
		{
			name:     "Exact match",
			str:      "URLLC",
			substr:   "URLLC",
			expected: true,
		},
		{
			name:     "Start of string",
			str:      "edge computing",
			substr:   "edge",
			expected: true,
		},
		{
			name:     "End of string",
			str:      "ultra-low latency",
			substr:   "latency",
			expected: true,
		},
		{
			name:     "Middle of string",
			str:      "deploy AI service",
			substr:   "AI",
			expected: true,
		},
		{
			name:     "Not found",
			str:      "regular service",
			substr:   "edge",
			expected: false,
		},
		{
			name:     "Empty substring",
			str:      "any string",
			substr:   "",
			expected: true,
		},
		{
			name:     "Empty string",
			str:      "",
			substr:   "test",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := contains(tc.str, tc.substr)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetEdgeNodes(t *testing.T) {
	controller := createTestEdgeController()

	// Add some test nodes
	controller.edgeNodes["node-1"] = &EdgeNode{ID: "node-1"}
	controller.edgeNodes["node-2"] = &EdgeNode{ID: "node-2"}

	nodes := controller.GetEdgeNodes()

	assert.Len(t, nodes, 2)
	assert.Contains(t, nodes, "node-1")
	assert.Contains(t, nodes, "node-2")

	// Returned map should be a copy, not the original
	nodes["node-3"] = &EdgeNode{ID: "node-3"}
	assert.NotContains(t, controller.edgeNodes, "node-3")
}

func TestGetEdgeZones(t *testing.T) {
	controller := createTestEdgeController()

	// Add some test zones
	controller.edgeZones["zone-1"] = &EdgeZone{ID: "zone-1"}
	controller.edgeZones["zone-2"] = &EdgeZone{ID: "zone-2"}

	zones := controller.GetEdgeZones()

	assert.Len(t, zones, 2)
	assert.Contains(t, zones, "zone-1")
	assert.Contains(t, zones, "zone-2")

	// Returned map should be a copy, not the original
	zones["zone-3"] = &EdgeZone{ID: "zone-3"}
	assert.NotContains(t, controller.edgeZones, "zone-3")
}

// Benchmark tests
func BenchmarkRequiresEdgeProcessing(b *testing.B) {
	controller := createTestEdgeController()
	intent := createTestNetworkIntent("bench", "Deploy URLLC service for autonomous vehicles")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		controller.requiresEdgeProcessing(intent)
	}
}

func BenchmarkContainsFunction(b *testing.B) {
	str := "Deploy URLLC service for autonomous vehicles with ultra-low latency requirements"
	substr := "URLLC"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		contains(str, substr)
	}
}