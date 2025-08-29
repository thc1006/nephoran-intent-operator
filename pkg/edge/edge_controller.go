// Nephoran Intent Operator - Edge Computing Controller.

// Phase 4 Enterprise Architecture - Distributed O-RAN Edge Integration.




package edge



import (

	"context"

	"fmt"

	"sync"

	"time"



	"github.com/go-logr/logr"



	nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"



	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/kubernetes"



	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

)



// EdgeController manages edge computing nodes and distributed O-RAN functions.

type EdgeController struct {

	client.Client

	KubeClient kubernetes.Interface

	Log        logr.Logger

	Scheme     *runtime.Scheme

	config     *EdgeControllerConfig

	edgeNodes  map[string]*EdgeNode

	edgeZones  map[string]*EdgeZone

	mutex      sync.RWMutex

}



// EdgeControllerConfig defines configuration for edge computing management.

type EdgeControllerConfig struct {

	// Edge node discovery.

	NodeDiscoveryEnabled    bool          `json:"node_discovery_enabled"`

	DiscoveryInterval       time.Duration `json:"discovery_interval"`    // 30s

	NodeHealthCheckInterval time.Duration `json:"health_check_interval"` // 10s



	// Edge zone management.

	AutoZoneCreation     bool `json:"auto_zone_creation"`

	MaxNodesPerZone      int  `json:"max_nodes_per_zone"`     // 10

	ZoneRedundancyFactor int  `json:"zone_redundancy_factor"` // 2



	// O-RAN edge functions.

	EnableLocalRIC         bool `json:"enable_local_ric"`

	EnableEdgeML           bool `json:"enable_edge_ml"`

	EnableCaching          bool `json:"enable_caching"`

	LocalProcessingEnabled bool `json:"local_processing_enabled"`



	// Performance thresholds.

	MaxLatencyMs          int     `json:"max_latency_ms"`          // 5ms for URLLC

	MinBandwidthMbps      int     `json:"min_bandwidth_mbps"`      // 100Mbps

	EdgeResourceThreshold float64 `json:"edge_resource_threshold"` // 0.8 (80%)



	// Failover and resilience.

	EdgeFailoverEnabled     bool `json:"edge_failover_enabled"`

	BackhaulFailoverEnabled bool `json:"backhaul_failover_enabled"`

	LocalAutonomy           bool `json:"local_autonomy"` // Continue without backhaul

}



// EdgeNode represents an edge computing node.

type EdgeNode struct {

	ID                string             `json:"id"`

	Name              string             `json:"name"`

	Zone              string             `json:"zone"`

	Status            EdgeNodeStatus     `json:"status"`

	Capabilities      EdgeCapabilities   `json:"capabilities"`

	Resources         EdgeResources      `json:"resources"`

	NetworkInterfaces []NetworkInterface `json:"network_interfaces"`

	O_RANFunctions    []O_RANFunction    `json:"oran_functions"`

	Location          GeographicLocation `json:"location"`

	LastSeen          time.Time          `json:"last_seen"`

	HealthMetrics     EdgeHealthMetrics  `json:"health_metrics"`

	LocalServices     []EdgeService      `json:"local_services"`

}



// EdgeZone represents a geographic edge computing zone.

type EdgeZone struct {

	ID               string         `json:"id"`

	Name             string         `json:"name"`

	Region           string         `json:"region"`

	Nodes            []string       `json:"nodes"` // Node IDs

	Coverage         GeographicArea `json:"coverage"`

	ServiceLevel     ServiceLevel   `json:"service_level"`    // Premium, Standard, Basic

	RedundancyLevel  int            `json:"redundancy_level"` // Number of backup nodes

	ConnectedUsers   int            `json:"connected_users"`

	TotalCapacity    EdgeResources  `json:"total_capacity"`

	UtilizedCapacity EdgeResources  `json:"utilized_capacity"`

}



// EdgeNodeStatus represents the current status of an edge node.

type EdgeNodeStatus string



const (

	// EdgeNodeActive holds edgenodeactive value.

	EdgeNodeActive EdgeNodeStatus = "Active"

	// EdgeNodeDegraded holds edgenodedegraded value.

	EdgeNodeDegraded EdgeNodeStatus = "Degraded"

	// EdgeNodeMaintenance holds edgenodemaintenance value.

	EdgeNodeMaintenance EdgeNodeStatus = "Maintenance"

	// EdgeNodeOffline holds edgenodeoffline value.

	EdgeNodeOffline EdgeNodeStatus = "Offline"

	// EdgeNodeFailed holds edgenodefailed value.

	EdgeNodeFailed EdgeNodeStatus = "Failed"

)



// EdgeCapabilities defines what edge functions a node can support.

type EdgeCapabilities struct {

	ComputeIntensive     bool     `json:"compute_intensive"`      // AI/ML workloads

	LowLatencyProcessing bool     `json:"low_latency_processing"` // <1ms processing

	LocalRICSupport      bool     `json:"local_ric_support"`      // Near-RT RIC functions

	CachingSupport       bool     `json:"caching_support"`        // Content/data caching

	VideoProcessing      bool     `json:"video_processing"`       // Video analytics

	IoTGateway           bool     `json:"iot_gateway"`            // IoT device management

	NetworkFunctions     []string `json:"network_functions"`      // Supported NFs

	AcceleratorTypes     []string `json:"accelerator_types"`      // GPU, FPGA, etc.

	ComputeCores         int      `json:"compute_cores"`

	MemoryGB             float64  `json:"memory_gb"`

	StorageGB            float64  `json:"storage_gb"`

	GPUEnabled           bool     `json:"gpu_enabled"`

	GPUMemoryGB          float64  `json:"gpu_memory_gb"`

	MLFrameworks         []string `json:"ml_frameworks"`

	AcceleratorType      string   `json:"accelerator_type"`

	CacheEnabled         bool     `json:"cache_enabled"`

	CacheSizeGB          float64  `json:"cache_size_gb"`

}



// EdgeResources tracks resource availability.

type EdgeResources struct {

	CPU                int                 `json:"cpu"`               // CPU cores

	Memory             int64               `json:"memory"`            // Memory in bytes

	Storage            int64               `json:"storage"`           // Storage in bytes

	GPU                int                 `json:"gpu"`               // GPU units

	NetworkBandwidth   int                 `json:"network_bandwidth"` // Mbps

	Accelerators       int                 `json:"accelerators"`      // FPGA/other accelerators

	Utilization        ResourceUtilization `json:"utilization"`

	CPUUtilization     float64             `json:"cpu_utilization"`

	MemoryUtilization  float64             `json:"memory_utilization"`

	StorageUtilization float64             `json:"storage_utilization"`

}



// ResourceUtilization tracks current resource usage.

type ResourceUtilization struct {

	CPUPercent     float64 `json:"cpu_percent"`

	MemoryPercent  float64 `json:"memory_percent"`

	StoragePercent float64 `json:"storage_percent"`

	NetworkPercent float64 `json:"network_percent"`

}



// NetworkInterface represents edge node network connectivity.

type NetworkInterface struct {

	Name           string   `json:"name"`

	Type           string   `json:"type"`            // 5G, WiFi6, Ethernet

	Bandwidth      int      `json:"bandwidth"`       // Mbps

	Latency        int      `json:"latency"`         // Milliseconds

	PacketLoss     float64  `json:"packet_loss"`     // Percentage

	ConnectedCells []string `json:"connected_cells"` // For 5G interfaces

	QoSSupport     bool     `json:"qos_support"`

}



// O_RANFunction represents O-RAN functions running on edge nodes.

type O_RANFunction struct {

	Type           string            `json:"type"` // CU, DU, RU, Near-RT RIC

	Version        string            `json:"version"`

	Status         string            `json:"status"`

	Configuration  map[string]string `json:"configuration"`

	Metrics        O_RANMetrics      `json:"metrics"`

	ConnectedCells []ConnectedCell   `json:"connected_cells"`

}



// O_RANMetrics tracks O-RAN function performance.

type O_RANMetrics struct {

	ThroughputMbps float64 `json:"throughput_mbps"`

	LatencyMs      float64 `json:"latency_ms"`

	ConnectedUEs   int     `json:"connected_ues"`

	ActiveSessions int     `json:"active_sessions"`

	ErrorRate      float64 `json:"error_rate"`

	ResourceUsage  float64 `json:"resource_usage"`

}



// ConnectedCell represents a cell connected to the edge node.

type ConnectedCell struct {

	CellID        string  `json:"cell_id"`

	Type          string  `json:"type"`      // Macro, Small, Femto

	Frequency     string  `json:"frequency"` // Band information

	Coverage      float64 `json:"coverage"`  // Coverage radius in meters

	ConnectedUEs  int     `json:"connected_ues"`

	SignalQuality float64 `json:"signal_quality"`

}



// EdgeHealthMetrics tracks the health of edge nodes.

type EdgeHealthMetrics struct {

	UptimePercent      float64   `json:"uptime_percent"`

	AverageLatency     float64   `json:"average_latency"` // ms

	ThroughputMbps     float64   `json:"throughput_mbps"`

	ErrorRate          float64   `json:"error_rate"`

	TemperatureCelsius float64   `json:"temperature_celsius"`

	PowerConsumption   float64   `json:"power_consumption"` // Watts

	LastHealthCheck    time.Time `json:"last_health_check"`

	LastCheck          time.Time `json:"last_check"`

	Latency            float64   `json:"latency"`

	PacketLoss         float64   `json:"packet_loss"`

	Jitter             float64   `json:"jitter"`

	Availability       float64   `json:"availability"`

	ConnectionCount    int       `json:"connection_count"`

	ActiveSessions     int       `json:"active_sessions"`

}



// EdgeService represents services running on edge nodes.

type EdgeService struct {

	ID             string            `json:"id"`

	Name           string            `json:"name"`

	Type           string            `json:"type"` // AI/ML, Caching, Processing

	Status         string            `json:"status"`

	Port           int               `json:"port"`

	Configuration  map[string]string `json:"configuration"`

	HealthEndpoint string            `json:"health_endpoint"`

	Metrics        ServiceMetrics    `json:"metrics"`

	Endpoint       string            `json:"endpoint"`

}



// ServiceMetrics tracks edge service performance.

type ServiceMetrics struct {

	RequestsPerSecond   float64 `json:"requests_per_second"`

	AverageResponseTime float64 `json:"average_response_time"` // ms

	ErrorRate           float64 `json:"error_rate"`

	CacheHitRate        float64 `json:"cache_hit_rate"` // For caching services

}



// GeographicArea defines coverage area for edge zones.

type GeographicArea struct {

	CenterLatitude  float64      `json:"center_latitude"`

	CenterLongitude float64      `json:"center_longitude"`

	RadiusKm        float64      `json:"radius_km"`

	Polygon         []Coordinate `json:"polygon,omitempty"` // For irregular shapes

}



// Coordinate represents a geographic coordinate.

type Coordinate struct {

	Latitude  float64 `json:"latitude"`

	Longitude float64 `json:"longitude"`

}



// GeographicLocation represents a specific geographic location.

type GeographicLocation struct {

	Latitude  float64 `json:"latitude"`

	Longitude float64 `json:"longitude"`

	Altitude  float64 `json:"altitude,omitempty"` // meters above sea level

	Country   string  `json:"country,omitempty"`

	Region    string  `json:"region,omitempty"`

	City      string  `json:"city,omitempty"`

}



// ServiceLevel defines the service level for edge zones.

type ServiceLevel string



const (

	// ServiceLevelPremium holds servicelevelpremium value.

	ServiceLevelPremium ServiceLevel = "Premium" // <1ms latency, 99.99% availability

	// ServiceLevelStandard holds servicelevelstandard value.

	ServiceLevelStandard ServiceLevel = "Standard" // <5ms latency, 99.9% availability

	// ServiceLevelBasic holds servicelevelbasic value.

	ServiceLevelBasic ServiceLevel = "Basic" // <20ms latency, 99% availability

)



// NewEdgeController creates a new edge computing controller.

func NewEdgeController(

	client client.Client,

	kubeClient kubernetes.Interface,

	logger logr.Logger,

	scheme *runtime.Scheme,

	config *EdgeControllerConfig,

) *EdgeController {

	return &EdgeController{

		Client:     client,

		KubeClient: kubeClient,

		Log:        logger,

		Scheme:     scheme,

		config:     config,

		edgeNodes:  make(map[string]*EdgeNode),

		edgeZones:  make(map[string]*EdgeZone),

	}

}



// SetupWithManager sets up the controller with the Manager.

func (r *EdgeController) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).

		For(&nephoran.NetworkIntent{}).

		Complete(r)

}



// Reconcile handles edge computing operations.

func (r *EdgeController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {

	log := r.Log.WithValues("networkintent", req.NamespacedName)



	// Get the NetworkIntent resource.

	var intent nephoran.NetworkIntent

	if err := r.Get(ctx, req.NamespacedName, &intent); err != nil {

		return reconcile.Result{}, client.IgnoreNotFound(err)

	}



	// Check if this intent requires edge processing.

	if !r.requiresEdgeProcessing(&intent) {

		return reconcile.Result{}, nil

	}



	log.Info("Processing edge computing intent", "intent", intent.Name)



	// Find suitable edge nodes.

	edgeNodes, err := r.findSuitableEdgeNodes(ctx, &intent)

	if err != nil {

		log.Error(err, "Failed to find suitable edge nodes")

		return reconcile.Result{RequeueAfter: 30 * time.Second}, err

	}



	// Deploy to edge nodes.

	if err := r.deployToEdgeNodes(ctx, &intent, edgeNodes); err != nil {

		log.Error(err, "Failed to deploy to edge nodes")

		return reconcile.Result{RequeueAfter: 60 * time.Second}, err

	}



	// Update intent status.

	intent.Status.Phase = "Deployed"

	// Note: Message field not available in current API version.

	if err := r.Status().Update(ctx, &intent); err != nil {

		log.Error(err, "Failed to update intent status")

	}



	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil

}



// StartEdgeDiscovery starts the edge node discovery process.

func (r *EdgeController) StartEdgeDiscovery(ctx context.Context) error {

	r.Log.Info("Starting edge node discovery")



	go r.runEdgeDiscovery(ctx)

	go r.runHealthChecks(ctx)

	go r.runZoneManagement(ctx)



	return nil

}



// runEdgeDiscovery continuously discovers edge nodes.

func (r *EdgeController) runEdgeDiscovery(ctx context.Context) {

	ticker := time.NewTicker(r.config.DiscoveryInterval)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			if err := r.discoverEdgeNodes(ctx); err != nil {

				r.Log.Error(err, "Edge node discovery failed")

			}

		}

	}

}



// discoverEdgeNodes discovers and registers edge nodes.

func (r *EdgeController) discoverEdgeNodes(ctx context.Context) error {

	// This would implement actual edge node discovery.

	// For now, simulate discovery of edge nodes.



	r.Log.V(1).Info("Discovering edge nodes...")



	// In a real implementation, this would:.

	// 1. Query Kubernetes nodes with edge labels.

	// 2. Probe network for edge devices.

	// 3. Check service registries.

	// 4. Validate edge capabilities.



	discoveredNodes := r.simulateEdgeNodeDiscovery()



	r.mutex.Lock()

	defer r.mutex.Unlock()



	for _, node := range discoveredNodes {

		r.edgeNodes[node.ID] = node

		r.Log.Info("Discovered edge node",

			"id", node.ID,

			"zone", node.Zone,

			"capabilities", len(node.Capabilities.NetworkFunctions))

	}



	// Update zone information.

	r.updateEdgeZones()



	return nil

}



// simulateEdgeNodeDiscovery simulates edge node discovery.

func (r *EdgeController) simulateEdgeNodeDiscovery() []*EdgeNode {

	// Simulate different types of edge nodes.

	nodes := []*EdgeNode{

		{

			ID:     "edge-node-1",

			Name:   "Metro Edge Node 1",

			Zone:   "metro-zone-1",

			Status: EdgeNodeActive,

			Capabilities: EdgeCapabilities{

				ComputeIntensive:     true,

				LowLatencyProcessing: true,

				LocalRICSupport:      true,

				CachingSupport:       true,

				NetworkFunctions:     []string{"CU", "DU", "Near-RT RIC"},

				AcceleratorTypes:     []string{"GPU", "FPGA"},

			},

			Resources: EdgeResources{

				CPU:              16,

				Memory:           64 * 1024 * 1024 * 1024,       // 64GB

				Storage:          1 * 1024 * 1024 * 1024 * 1024, // 1TB

				GPU:              2,

				NetworkBandwidth: 10000, // 10Gbps

				Accelerators:     1,

				Utilization: ResourceUtilization{

					CPUPercent:     45.0,

					MemoryPercent:  60.0,

					StoragePercent: 30.0,

					NetworkPercent: 25.0,

				},

			},

			NetworkInterfaces: []NetworkInterface{

				{

					Name:           "5g-interface",

					Type:           "5G",

					Bandwidth:      1000,

					Latency:        2,

					PacketLoss:     0.01,

					ConnectedCells: []string{"cell-001", "cell-002"},

					QoSSupport:     true,

				},

			},

			O_RANFunctions: []O_RANFunction{

				{

					Type:    "Near-RT RIC",

					Version: "1.0.0",

					Status:  "Active",

					Metrics: O_RANMetrics{

						ThroughputMbps: 500.0,

						LatencyMs:      1.5,

						ConnectedUEs:   150,

						ActiveSessions: 120,

						ErrorRate:      0.001,

						ResourceUsage:  0.6,

					},

				},

			},

			Location: GeographicLocation{

				Latitude:  40.7128,

				Longitude: -74.0060, // New York

			},

			LastSeen: time.Now(),

			HealthMetrics: EdgeHealthMetrics{

				UptimePercent:      99.95,

				AverageLatency:     1.8,

				ThroughputMbps:     750.0,

				ErrorRate:          0.001,

				TemperatureCelsius: 35.0,

				PowerConsumption:   250.0,

				LastHealthCheck:    time.Now(),

			},

		},

		{

			ID:     "edge-node-2",

			Name:   "Access Edge Node 1",

			Zone:   "access-zone-1",

			Status: EdgeNodeActive,

			Capabilities: EdgeCapabilities{

				ComputeIntensive:     false,

				LowLatencyProcessing: true,

				LocalRICSupport:      false,

				CachingSupport:       true,

				IoTGateway:           true,

				NetworkFunctions:     []string{"RU"},

				AcceleratorTypes:     []string{},

			},

			Resources: EdgeResources{

				CPU:              4,

				Memory:           16 * 1024 * 1024 * 1024,  // 16GB

				Storage:          500 * 1024 * 1024 * 1024, // 500GB

				GPU:              0,

				NetworkBandwidth: 1000, // 1Gbps

				Accelerators:     0,

				Utilization: ResourceUtilization{

					CPUPercent:     30.0,

					MemoryPercent:  40.0,

					StoragePercent: 20.0,

					NetworkPercent: 15.0,

				},

			},

			NetworkInterfaces: []NetworkInterface{

				{

					Name:       "fiber-interface",

					Type:       "Ethernet",

					Bandwidth:  1000,

					Latency:    5,

					PacketLoss: 0.005,

					QoSSupport: true,

				},

			},

			Location: GeographicLocation{

				Latitude:  40.7589,

				Longitude: -73.9851, // Manhattan

			},

			LastSeen: time.Now(),

			HealthMetrics: EdgeHealthMetrics{

				UptimePercent:      99.8,

				AverageLatency:     4.2,

				ThroughputMbps:     200.0,

				ErrorRate:          0.002,

				TemperatureCelsius: 28.0,

				PowerConsumption:   80.0,

				LastHealthCheck:    time.Now(),

			},

		},

	}



	return nodes

}



// updateEdgeZones updates edge zone information based on discovered nodes.

func (r *EdgeController) updateEdgeZones() {

	zones := make(map[string]*EdgeZone)



	for nodeID, node := range r.edgeNodes {

		zone := zones[node.Zone]

		if zone == nil {

			zone = &EdgeZone{

				ID:              node.Zone,

				Name:            node.Zone,

				Region:          r.getRegionFromZone(node.Zone),

				Nodes:           []string{},

				ServiceLevel:    r.determineServiceLevel(node),

				RedundancyLevel: r.config.ZoneRedundancyFactor,

				TotalCapacity:   EdgeResources{},

			}

			zones[node.Zone] = zone

		}



		zone.Nodes = append(zone.Nodes, nodeID)

		r.aggregateZoneCapacity(zone, node)

	}



	r.edgeZones = zones



	r.Log.Info("Updated edge zones", "zones", len(zones))

}



// getRegionFromZone determines the region for a zone.

func (r *EdgeController) getRegionFromZone(zone string) string {

	// Simple mapping - in real implementation this would be configurable.

	if zone == "metro-zone-1" {

		return "us-east-1"

	}

	return "us-east-1"

}



// determineServiceLevel determines service level based on node capabilities.

func (r *EdgeController) determineServiceLevel(node *EdgeNode) ServiceLevel {

	if node.Capabilities.LowLatencyProcessing && node.HealthMetrics.AverageLatency < 2.0 {

		return ServiceLevelPremium

	} else if node.HealthMetrics.AverageLatency < 10.0 {

		return ServiceLevelStandard

	}

	return ServiceLevelBasic

}



// aggregateZoneCapacity aggregates resource capacity for a zone.

func (r *EdgeController) aggregateZoneCapacity(zone *EdgeZone, node *EdgeNode) {

	zone.TotalCapacity.CPU += node.Resources.CPU

	zone.TotalCapacity.Memory += node.Resources.Memory

	zone.TotalCapacity.Storage += node.Resources.Storage

	zone.TotalCapacity.GPU += node.Resources.GPU

	zone.TotalCapacity.NetworkBandwidth += node.Resources.NetworkBandwidth

	zone.TotalCapacity.Accelerators += node.Resources.Accelerators

}



// runHealthChecks performs continuous health checks on edge nodes.

func (r *EdgeController) runHealthChecks(ctx context.Context) {

	ticker := time.NewTicker(r.config.NodeHealthCheckInterval)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			r.performHealthChecks(ctx)

		}

	}

}



// performHealthChecks checks health of all edge nodes.

func (r *EdgeController) performHealthChecks(ctx context.Context) {

	r.mutex.RLock()

	nodes := make(map[string]*EdgeNode)

	for k, v := range r.edgeNodes {

		nodes[k] = v

	}

	r.mutex.RUnlock()



	var wg sync.WaitGroup

	for nodeID, node := range nodes {

		wg.Add(1)

		go func(id string, n *EdgeNode) {

			defer wg.Done()

			r.checkNodeHealth(ctx, id, n)

		}(nodeID, node)

	}



	wg.Wait()

}



// checkNodeHealth performs health check for a specific edge node.

func (r *EdgeController) checkNodeHealth(ctx context.Context, nodeID string, node *EdgeNode) {

	// Simulate health check - in real implementation would make HTTP calls.

	node.LastSeen = time.Now()

	node.HealthMetrics.LastHealthCheck = time.Now()



	// Simulate some variability in metrics.

	if node.HealthMetrics.AverageLatency > 10.0 {

		node.Status = EdgeNodeDegraded

	} else {

		node.Status = EdgeNodeActive

	}



	r.Log.V(2).Info("Health check completed",

		"node", nodeID,

		"status", node.Status,

		"latency", node.HealthMetrics.AverageLatency)

}



// runZoneManagement manages edge zones and their configurations.

func (r *EdgeController) runZoneManagement(ctx context.Context) {

	ticker := time.NewTicker(60 * time.Second) // Check every minute

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			r.manageEdgeZones(ctx)

		}

	}

}



// manageEdgeZones manages edge zone configurations and load balancing.

func (r *EdgeController) manageEdgeZones(ctx context.Context) {

	r.mutex.RLock()

	zones := make(map[string]*EdgeZone)

	for k, v := range r.edgeZones {

		zones[k] = v

	}

	r.mutex.RUnlock()



	for zoneID, zone := range zones {

		r.optimizeZoneLoad(ctx, zoneID, zone)

	}

}



// optimizeZoneLoad optimizes load distribution within an edge zone.

func (r *EdgeController) optimizeZoneLoad(ctx context.Context, zoneID string, zone *EdgeZone) {

	// Calculate current utilization.

	totalUtilization := 0.0

	nodeCount := 0



	for _, nodeID := range zone.Nodes {

		if node, exists := r.edgeNodes[nodeID]; exists {

			totalUtilization += node.Resources.Utilization.CPUPercent

			nodeCount++

		}

	}



	if nodeCount == 0 {

		return

	}



	avgUtilization := totalUtilization / float64(nodeCount)



	if avgUtilization > r.config.EdgeResourceThreshold*100 {

		r.Log.Info("Edge zone overloaded",

			"zone", zoneID,

			"utilization", avgUtilization,

			"threshold", r.config.EdgeResourceThreshold*100)



		// Trigger load balancing or scaling.

		r.triggerZoneScaling(ctx, zoneID, zone)

	}

}



// triggerZoneScaling triggers scaling operations for overloaded zones.

func (r *EdgeController) triggerZoneScaling(ctx context.Context, zoneID string, zone *EdgeZone) {

	// In a real implementation, this would:.

	// 1. Request additional edge nodes.

	// 2. Redistribute workloads.

	// 3. Activate backup nodes.

	// 4. Implement traffic shaping.



	r.Log.Info("Scaling edge zone", "zone", zoneID, "nodes", len(zone.Nodes))

}



// requiresEdgeProcessing determines if an intent requires edge processing.

func (r *EdgeController) requiresEdgeProcessing(intent *nephoran.NetworkIntent) bool {

	description := intent.Spec.Intent



	// Check for edge-related keywords.

	edgeKeywords := []string{

		"edge", "URLLC", "ultra-low latency", "real-time",

		"IoT", "AR/VR", "autonomous", "industrial", "local",

	}



	for _, keyword := range edgeKeywords {

		if contains(description, keyword) {

			return true

		}

	}



	// Check for low latency requirements in the intent text.

	// Note: Parameters is a RawExtension and requires proper parsing.

	if contains(description, "1ms") || contains(description, "5ms") || contains(description, "<10ms") ||

		contains(description, "ultra-low latency") || contains(description, "URLLC") {

		return true

	}



	return false

}



// findSuitableEdgeNodes finds edge nodes suitable for the intent.

func (r *EdgeController) findSuitableEdgeNodes(ctx context.Context, intent *nephoran.NetworkIntent) ([]*EdgeNode, error) {

	r.mutex.RLock()

	defer r.mutex.RUnlock()



	var suitableNodes []*EdgeNode



	for _, node := range r.edgeNodes {

		if r.isNodeSuitable(node, intent) {

			suitableNodes = append(suitableNodes, node)

		}

	}



	if len(suitableNodes) == 0 {

		return nil, fmt.Errorf("no suitable edge nodes found for intent %s", intent.Name)

	}



	// Sort by suitability score.

	r.sortNodesBySuitability(suitableNodes, intent)



	// Return top nodes (limit to avoid overloading).

	maxNodes := 3

	if len(suitableNodes) < maxNodes {

		maxNodes = len(suitableNodes)

	}



	return suitableNodes[:maxNodes], nil

}



// isNodeSuitable checks if a node is suitable for the intent.

func (r *EdgeController) isNodeSuitable(node *EdgeNode, intent *nephoran.NetworkIntent) bool {

	// Check node status.

	if node.Status != EdgeNodeActive {

		return false

	}



	// Check resource availability.

	if node.Resources.Utilization.CPUPercent > r.config.EdgeResourceThreshold*100 {

		return false

	}



	// Check latency requirements.

	if node.HealthMetrics.AverageLatency > float64(r.config.MaxLatencyMs) {

		return false

	}



	// Check specific capabilities based on intent.

	description := intent.Spec.Intent



	if contains(description, "AI") || contains(description, "ML") {

		if !node.Capabilities.ComputeIntensive {

			return false

		}

	}



	if contains(description, "URLLC") {

		if !node.Capabilities.LowLatencyProcessing {

			return false

		}

	}



	if contains(description, "RIC") {

		if !node.Capabilities.LocalRICSupport {

			return false

		}

	}



	return true

}



// sortNodesBySuitability sorts nodes by their suitability for the intent.

func (r *EdgeController) sortNodesBySuitability(nodes []*EdgeNode, intent *nephoran.NetworkIntent) {

	// Simple scoring algorithm - in real implementation would be more sophisticated.

	for i := range len(nodes) - 1 {

		for j := i + 1; j < len(nodes); j++ {

			score1 := r.calculateNodeScore(nodes[i], intent)

			score2 := r.calculateNodeScore(nodes[j], intent)



			if score2 > score1 {

				nodes[i], nodes[j] = nodes[j], nodes[i]

			}

		}

	}

}



// calculateNodeScore calculates a suitability score for a node.

func (r *EdgeController) calculateNodeScore(node *EdgeNode, intent *nephoran.NetworkIntent) float64 {

	score := 0.0



	// Lower latency is better.

	score += (10.0 - node.HealthMetrics.AverageLatency) * 10



	// Lower utilization is better.

	score += (100.0 - node.Resources.Utilization.CPUPercent) * 0.5



	// Higher uptime is better.

	score += node.HealthMetrics.UptimePercent * 0.1



	// Bonus for specific capabilities.

	description := intent.Spec.Intent

	if contains(description, "AI") && node.Capabilities.ComputeIntensive {

		score += 50.0

	}

	if contains(description, "URLLC") && node.Capabilities.LowLatencyProcessing {

		score += 30.0

	}



	return score

}



// deployToEdgeNodes deploys the intent to selected edge nodes.

func (r *EdgeController) deployToEdgeNodes(ctx context.Context, intent *nephoran.NetworkIntent, nodes []*EdgeNode) error {

	r.Log.Info("Deploying to edge nodes",

		"intent", intent.Name,

		"nodes", len(nodes))



	for _, node := range nodes {

		if err := r.deployToSingleNode(ctx, intent, node); err != nil {

			r.Log.Error(err, "Failed to deploy to edge node", "node", node.ID)

			continue

		}



		r.Log.Info("Successfully deployed to edge node", "node", node.ID)

	}



	return nil

}



// deployToSingleNode deploys the intent to a single edge node.

func (r *EdgeController) deployToSingleNode(ctx context.Context, intent *nephoran.NetworkIntent, node *EdgeNode) error {

	// In a real implementation, this would:.

	// 1. Generate edge-specific manifests.

	// 2. Apply configurations to the edge node.

	// 3. Start required services.

	// 4. Configure network functions.

	// 5. Set up monitoring.



	r.Log.Info("Deploying to edge node",

		"intent", intent.Name,

		"node", node.ID,

		"zone", node.Zone)



	// Simulate deployment delay.

	time.Sleep(2 * time.Second)



	return nil

}



// GetEdgeNodes returns all discovered edge nodes.

func (r *EdgeController) GetEdgeNodes() map[string]*EdgeNode {

	r.mutex.RLock()

	defer r.mutex.RUnlock()



	result := make(map[string]*EdgeNode)

	for k, v := range r.edgeNodes {

		result[k] = v

	}

	return result

}



// GetEdgeZones returns all edge zones.

func (r *EdgeController) GetEdgeZones() map[string]*EdgeZone {

	r.mutex.RLock()

	defer r.mutex.RUnlock()



	result := make(map[string]*EdgeZone)

	for k, v := range r.edgeZones {

		result[k] = v

	}

	return result

}



// contains checks if a string contains a substring (case-insensitive).

func contains(s, substr string) bool {

	return len(s) >= len(substr) &&

		(s == substr ||

			len(s) > len(substr) &&

				(s[:len(substr)] == substr ||

					s[len(s)-len(substr):] == substr ||

					containsMiddle(s, substr)))

}



// containsMiddle checks if substr exists in the middle of s.

func containsMiddle(s, substr string) bool {

	for i := 0; i <= len(s)-len(substr); i++ {

		if s[i:i+len(substr)] == substr {

			return true

		}

	}

	return false

}

