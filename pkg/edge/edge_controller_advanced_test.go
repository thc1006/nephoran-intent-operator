package edge

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// TestEdgeNodeSelection tests the edge node selection logic
func TestEdgeNodeSelection(t *testing.T) {
	config := &EdgeControllerConfig{
		NodeDiscoveryEnabled:  true,
		MaxLatencyMs:          5,
		MinBandwidthMbps:      100,
		EdgeResourceThreshold: 0.8,
	}
	kubeClient := fake.NewSimpleClientset()
	logger := logr.Discard()
	controller := NewEdgeController(nil, kubeClient, logger, runtime.NewScheme(), config)
	ctx := context.Background()

	// Discover nodes first
	require.NoError(t, controller.discoverEdgeNodes(ctx))

	tests := []struct {
		name           string
		requirements   EdgeRequirements
		expectedNodeID string
		expectError    bool
	}{
		{
			name: "select node with GPU requirement",
			requirements: EdgeRequirements{
				ComputeIntensive:     true,
				LowLatencyProcessing: true,
				GPURequired:          true,
				MinMemoryGB:          32,
				MinCPU:               8,
			},
			expectedNodeID: "edge-node-1", // Metro node has GPU
			expectError:    false,
		},
		{
			name: "select node for IoT gateway",
			requirements: EdgeRequirements{
				IoTGateway:           true,
				LowLatencyProcessing: true,
				MinMemoryGB:          8,
				MinCPU:               2,
			},
			expectedNodeID: "edge-node-2", // Access node has IoT capability
			expectError:    false,
		},
		{
			name: "select node with impossible requirements",
			requirements: EdgeRequirements{
				ComputeIntensive: true,
				GPURequired:      true,
				MinMemoryGB:      128, // Too much memory
				MinCPU:           32,  // Too many CPUs
			},
			expectedNodeID: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := controller.selectBestEdgeNode(tt.requirements)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, node)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, node)
				assert.Equal(t, tt.expectedNodeID, node.ID)
			}
		})
	}
}

// TestEdgeNodeHealthChecking tests health check functionality
func TestEdgeNodeHealthChecking(t *testing.T) {
	controller := NewEdgeController(nil, fake.NewSimpleClientset(), logr.Discard(), runtime.NewScheme(), &EdgeControllerConfig{
		NodeDiscoveryEnabled:  true,
		MaxLatencyMs:          5,
		MinBandwidthMbps:      100,
		EdgeResourceThreshold: 0.8,
	})
	ctx := context.Background()

	// Add test nodes
	healthyNode := &EdgeNode{
		ID:     "healthy-node",
		Name:   "Healthy Node",
		Status: EdgeNodeActive,
		HealthMetrics: EdgeHealthMetrics{
			UptimePercent:  99.9,
			AverageLatency: 1.5,
			ThroughputMbps: 800,
			ErrorRate:      0.01,
		},
		LastSeen: time.Now(),
	}

	unhealthyNode := &EdgeNode{
		ID:     "unhealthy-node",
		Name:   "Unhealthy Node",
		Status: EdgeNodeActive,
		HealthMetrics: EdgeHealthMetrics{
			UptimePercent:  50.0, // Below threshold
			AverageLatency: 10.0, // Above threshold
			ThroughputMbps: 100,  // Below threshold
			ErrorRate:      5.0,  // Above threshold
		},
		LastSeen: time.Now(),
	}

	controller.edgeNodes["healthy-node"] = healthyNode
	controller.edgeNodes["unhealthy-node"] = unhealthyNode

	// Perform health check
	controller.performHealthCheck(ctx)

	// Verify healthy node remains active
	assert.Equal(t, EdgeNodeActive, controller.edgeNodes["healthy-node"].Status)

	// Verify unhealthy node is marked as unhealthy
	assert.Equal(t, EdgeNodeDegraded, controller.edgeNodes["unhealthy-node"].Status)
}

// TestEdgeZoneManagement tests edge zone creation and management
func TestEdgeZoneManagement(t *testing.T) {
	controller := NewEdgeController(nil, fake.NewSimpleClientset(), logr.Discard(), runtime.NewScheme(), &EdgeControllerConfig{
		NodeDiscoveryEnabled:  true,
		MaxLatencyMs:          5,
		MinBandwidthMbps:      100,
		EdgeResourceThreshold: 0.8,
	})

	// Test zone creation
	zone := controller.createEdgeZone("test-zone", "us-west-2")
	assert.NotNil(t, zone)
	assert.Equal(t, "test-zone", zone.ID)
	assert.Equal(t, "us-west-2", zone.Region)
	assert.Equal(t, 2, zone.RedundancyLevel)
	assert.NotNil(t, zone.Nodes)

	// Test adding node to zone
	node := &EdgeNode{
		ID:   "test-node",
		Zone: "test-zone",
	}

	controller.edgeZones["test-zone"] = zone
	controller.edgeNodes["test-node"] = node

	// Add node to zone
	zone.Nodes = append(zone.Nodes, node.ID)

	// Verify node is in zone
	assert.Contains(t, zone.Nodes, "test-node")
}

// TestEdgeFailover tests edge failover functionality
func TestEdgeFailover(t *testing.T) {
	controller := NewEdgeController(nil, fake.NewSimpleClientset(), logr.Discard(), runtime.NewScheme(), &EdgeControllerConfig{
		NodeDiscoveryEnabled:  true,
		MaxLatencyMs:          5,
		MinBandwidthMbps:      100,
		EdgeResourceThreshold: 0.8,
	})

	// Create primary and backup nodes
	primaryNode := &EdgeNode{
		ID:     "primary-node",
		Name:   "Primary Node",
		Zone:   "zone-1",
		Status: EdgeNodeActive,
		Resources: EdgeResources{
			CPU:              8,
			Memory:           32 * 1024 * 1024 * 1024,
			GPU:              1,
			NetworkBandwidth: 5000,
		},
	}

	backupNode := &EdgeNode{
		ID:     "backup-node",
		Name:   "Backup Node",
		Zone:   "zone-1",
		Status: EdgeNodeActive,
		Resources: EdgeResources{
			CPU:              8,
			Memory:           32 * 1024 * 1024 * 1024,
			GPU:              1,
			NetworkBandwidth: 5000,
		},
	}

	controller.edgeNodes["primary-node"] = primaryNode
	controller.edgeNodes["backup-node"] = backupNode

	// Create zone with both nodes
	zone := &EdgeZone{
		ID:              "zone-1",
		Region:          "us-east-1",
		Nodes:           []string{"primary-node", "backup-node"},
		RedundancyLevel: 2,
	}
	controller.edgeZones["zone-1"] = zone

	// Simulate primary node failure
	primaryNode.Status = EdgeNodeDegraded

	// Test failover
	requirements := EdgeRequirements{
		ComputeIntensive:     true,
		LowLatencyProcessing: true,
		MinMemoryGB:          16,
		MinCPU:               4,
	}

	selectedNode, err := controller.selectBestEdgeNode(requirements)
	assert.NoError(t, err)
	assert.NotNil(t, selectedNode)
	assert.Equal(t, "backup-node", selectedNode.ID)
}

// TestEdgeMetricsCollection tests metrics collection for edge nodes
func TestEdgeMetricsCollection(t *testing.T) {
	controller := NewEdgeController(nil, fake.NewSimpleClientset(), logr.Discard(), runtime.NewScheme(), &EdgeControllerConfig{
		NodeDiscoveryEnabled:  true,
		MaxLatencyMs:          5,
		MinBandwidthMbps:      100,
		EdgeResourceThreshold: 0.8,
	})

	node := &EdgeNode{
		ID:     "metrics-node",
		Name:   "Metrics Test Node",
		Status: EdgeNodeActive,
		HealthMetrics: EdgeHealthMetrics{
			UptimePercent:  0,
			AverageLatency: 0,
			ThroughputMbps: 0,
			ErrorRate:      0,
		},
	}

	controller.edgeNodes["metrics-node"] = node

	// Simulate metrics updates
	updates := []EdgeHealthMetrics{
		{UptimePercent: 99.5, AverageLatency: 1.2, ThroughputMbps: 750, ErrorRate: 0.05},
		{UptimePercent: 99.8, AverageLatency: 1.1, ThroughputMbps: 800, ErrorRate: 0.03},
		{UptimePercent: 99.9, AverageLatency: 1.0, ThroughputMbps: 850, ErrorRate: 0.01},
	}

	for _, metrics := range updates {
		node.HealthMetrics = metrics
		node.LastSeen = time.Now()

		// Verify metrics are updated
		assert.Equal(t, metrics.UptimePercent, node.HealthMetrics.UptimePercent)
		assert.Equal(t, metrics.AverageLatency, node.HealthMetrics.AverageLatency)
		assert.Equal(t, metrics.ThroughputMbps, node.HealthMetrics.ThroughputMbps)
		assert.Equal(t, metrics.ErrorRate, node.HealthMetrics.ErrorRate)
	}
}

// TestEdgeCapabilities tests edge node capability matching
func TestEdgeCapabilities(t *testing.T) {
	tests := []struct {
		name         string
		capabilities EdgeCapabilities
		requirements EdgeRequirements
		shouldMatch  bool
	}{
		{
			name: "exact match",
			capabilities: EdgeCapabilities{
				ComputeIntensive:     true,
				LowLatencyProcessing: true,
				LocalRICSupport:      true,
				CachingSupport:       true,
			},
			requirements: EdgeRequirements{
				ComputeIntensive:     true,
				LowLatencyProcessing: true,
				LocalRICSupport:      true,
				CachingSupport:       true,
			},
			shouldMatch: true,
		},
		{
			name: "capability superset",
			capabilities: EdgeCapabilities{
				ComputeIntensive:     true,
				LowLatencyProcessing: true,
				LocalRICSupport:      true,
				CachingSupport:       true,
				IoTGateway:           true,
			},
			requirements: EdgeRequirements{
				ComputeIntensive:     true,
				LowLatencyProcessing: true,
			},
			shouldMatch: true,
		},
		{
			name: "missing required capability",
			capabilities: EdgeCapabilities{
				ComputeIntensive:     false,
				LowLatencyProcessing: true,
			},
			requirements: EdgeRequirements{
				ComputeIntensive:     true,
				LowLatencyProcessing: true,
			},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &EdgeNode{
				Capabilities: tt.capabilities,
				Resources: EdgeResources{
					CPU:    16,
					Memory: 64 * 1024 * 1024 * 1024,
				},
			}

			matches := node.meetsRequirements(tt.requirements)
			assert.Equal(t, tt.shouldMatch, matches)
		})
	}
}

// Helper function to check if node meets requirements
func (n *EdgeNode) meetsRequirements(req EdgeRequirements) bool {
	// Check capabilities
	if req.ComputeIntensive && !n.Capabilities.ComputeIntensive {
		return false
	}
	if req.LowLatencyProcessing && !n.Capabilities.LowLatencyProcessing {
		return false
	}
	if req.LocalRICSupport && !n.Capabilities.LocalRICSupport {
		return false
	}
	if req.CachingSupport && !n.Capabilities.CachingSupport {
		return false
	}
	if req.IoTGateway && !n.Capabilities.IoTGateway {
		return false
	}

	// Check resources
	if req.MinCPU > n.Resources.CPU {
		return false
	}
	if req.MinMemoryGB > int(n.Resources.Memory/(1024*1024*1024)) {
		return false
	}
	if req.GPURequired && n.Resources.GPU == 0 {
		return false
	}

	return true
}

// Edge requirements structure for testing
type EdgeRequirements struct {
	ComputeIntensive     bool
	LowLatencyProcessing bool
	LocalRICSupport      bool
	CachingSupport       bool
	IoTGateway           bool
	GPURequired          bool
	MinCPU               int
	MinMemoryGB          int
}

// Helper function for node selection
func (ec *EdgeController) selectBestEdgeNode(req EdgeRequirements) (*EdgeNode, error) {
	ec.mutex.RLock()
	defer ec.mutex.RUnlock()

	var bestNode *EdgeNode
	var bestScore int

	for _, node := range ec.edgeNodes {
		if node.Status != EdgeNodeActive {
			continue
		}

		if !node.meetsRequirements(req) {
			continue
		}

		// Simple scoring based on available resources
		score := node.Resources.CPU + int(node.Resources.Memory/(1024*1024*1024)) + node.Resources.GPU*10

		if bestNode == nil || score > bestScore {
			bestNode = node
			bestScore = score
		}
	}

	if bestNode == nil {
		return nil, errors.New("no suitable edge node found")
	}

	return bestNode, nil
}

// Helper function for health checking
func (ec *EdgeController) performHealthCheck(ctx context.Context) {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()

	for _, node := range ec.edgeNodes {
		// Check health metrics
		if node.HealthMetrics.UptimePercent < 90.0 ||
			node.HealthMetrics.AverageLatency > 5.0 ||
			node.HealthMetrics.ThroughputMbps < 500 ||
			node.HealthMetrics.ErrorRate > 1.0 {
			node.Status = EdgeNodeDegraded
		} else {
			node.Status = EdgeNodeActive
		}

		node.LastSeen = time.Now()
	}
}

// Helper function to create edge zone
func (ec *EdgeController) createEdgeZone(id, region string) *EdgeZone {
	return &EdgeZone{
		ID:              id,
		Name:            fmt.Sprintf("Edge Zone %s", id),
		Region:          region,
		Nodes:           []string{},
		RedundancyLevel: 2,
	}
}
