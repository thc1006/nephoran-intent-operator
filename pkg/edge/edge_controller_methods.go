package edge

import (
	"fmt"
	"sort"
	"time"

	nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// Missing struct definitions for tests.

type EdgeOptimizationPlan struct {
	OptimizationType string `json:"optimization_type"`

	TargetNodes []string `json:"target_nodes"`

	Actions []string `json:"actions"`

	EstimatedBenefit float64 `json:"estimated_benefit"`
}

// NodeHealth represents a nodehealth.

type NodeHealth struct {
	NodeID string `json:"node_id"`

	Status EdgeNodeStatus `json:"status"`

	IsHealthy bool `json:"is_healthy"`

	Issues []string `json:"issues"`
}

// ResourceRequest represents a resourcerequest.

type ResourceRequest struct {
	CPUCores int `json:"cpu_cores"`

	MemoryGB float64 `json:"memory_gb"`

	StorageGB float64 `json:"storage_gb"`

	GPUCores int `json:"gpu_cores"`
}

// ResourceAllocation represents a resourceallocation.

type ResourceAllocation struct {
	AllocationID string `json:"allocation_id"`

	NodeID string `json:"node_id"`

	Request ResourceRequest `json:"request"`

	AllocatedAt time.Time `json:"allocated_at"`
}

// FailoverPlan represents a failoverplan.

type FailoverPlan struct {
	ID string `json:"id"`

	SourceNode string `json:"source_node"`

	TargetNode string `json:"target_node"`

	ServicesToMigrate []EdgeService `json:"services_to_migrate"`

	Status string `json:"status"`

	CreatedAt time.Time `json:"created_at"`
}

// MLModel represents a mlmodel.

type MLModel struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Framework string `json:"framework"`

	Version string `json:"version"`

	SizeGB float64 `json:"size_gb"`

	RequiresGPU bool `json:"requires_gpu"`
}

// MLDeployment represents a mldeployment.

type MLDeployment struct {
	ID string `json:"id"`

	ModelID string `json:"model_id"`

	NodeID string `json:"node_id"`

	Status string `json:"status"`

	Endpoint string `json:"endpoint"`

	DeployedAt time.Time `json:"deployed_at"`
}

// InferenceRequest represents a inferencerequest.

type InferenceRequest struct {
	ModelID string `json:"model_id"`

	Input []byte `json:"input"`

	Options map[string]string `json:"options"`
}

// InferenceResult represents a inferenceresult.

type InferenceResult struct {
	RequestID string `json:"request_id"`

	Output []byte `json:"output"`

	InferenceTimeMs int `json:"inference_time_ms"`

	Confidence float64 `json:"confidence"`
}

// MLOptimization represents a mloptimization.

type MLOptimization struct {
	ModelID string `json:"model_id"`

	Recommendations []string `json:"recommendations"`

	EstimatedSpeedup float64 `json:"estimated_speedup"`
}

// CacheConfig represents a cacheconfig.

type CacheConfig struct {
	MaxSizeGB float64 `json:"max_size_gb"`

	EvictionPolicy string `json:"eviction_policy"`

	TTL int `json:"ttl"`

	ContentTypes []string `json:"content_types"`
}

// CacheContent represents a cachecontent.

type CacheContent struct {
	ID string `json:"id"`

	Type string `json:"type"`

	SizeMB float64 `json:"size_mb"`

	Popularity float64 `json:"popularity"`

	LastAccess time.Time `json:"last_access"`
}

// CacheStats represents a cachestats.

type CacheStats struct {
	HitRate float64 `json:"hit_rate"`

	MissRate float64 `json:"miss_rate"`

	UsedGB float64 `json:"used_gb"`

	AvailableGB float64 `json:"available_gb"`

	EvictionRate float64 `json:"eviction_rate"`
}

// ZoneReport represents a zonereport.

type ZoneReport struct {
	ZoneID string `json:"zone_id"`

	TotalNodes int `json:"total_nodes"`

	ActiveNodes int `json:"active_nodes"`

	AverageLatency float64 `json:"average_latency"`

	TotalCapacity EdgeResources `json:"total_capacity"`

	UsedCapacity EdgeResources `json:"used_capacity"`

	HealthScore float64 `json:"health_score"`
}

// ResourcePrediction represents a resourceprediction.

type ResourcePrediction struct {
	ZoneID string `json:"zone_id"`

	TimeHorizon time.Duration `json:"time_horizon"`

	PredictedCPU float64 `json:"predicted_cpu"`

	PredictedMemory float64 `json:"predicted_memory"`

	PredictedStorage float64 `json:"predicted_storage"`

	Confidence float64 `json:"confidence"`
}

// PlacementOptimization represents a placementoptimization.

type PlacementOptimization struct {
	Recommendations []PlacementRecommendation `json:"recommendations"`

	EstimatedSavings float64 `json:"estimated_savings"`
}

// PlacementRecommendation represents a placementrecommendation.

type PlacementRecommendation struct {
	NodeID string `json:"node_id"`

	Action string `json:"action"`

	Reason string `json:"reason"`

	Priority int `json:"priority"`
}

// Node Management Methods.

func (r *EdgeController) RegisterNode(node *EdgeNode) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	if node.ID == "" {

		return fmt.Errorf("node ID cannot be empty")

	}

	r.edgeNodes[node.ID] = node

	r.Log.Info("Registered edge node", "nodeID", node.ID, "name", node.Name)

	return nil

}

// UpdateNodeStatus performs updatenodestatus operation.

func (r *EdgeController) UpdateNodeStatus(nodeID string, status EdgeNodeStatus) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	node, exists := r.edgeNodes[nodeID]

	if !exists {

		return fmt.Errorf("node %s not found", nodeID)

	}

	node.Status = status

	node.LastSeen = time.Now()

	r.Log.Info("Updated node status", "nodeID", nodeID, "status", status)

	return nil

}

// GetNode performs getnode operation.

func (r *EdgeController) GetNode(nodeID string) (*EdgeNode, error) {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	node, exists := r.edgeNodes[nodeID]

	if !exists {

		return nil, fmt.Errorf("node %s not found", nodeID)

	}

	return node, nil

}

// RemoveNode performs removenode operation.

func (r *EdgeController) RemoveNode(nodeID string) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	delete(r.edgeNodes, nodeID)

	r.Log.Info("Removed edge node", "nodeID", nodeID)

	return nil

}

// Zone Management Methods.

func (r *EdgeController) CreateZone(zone *EdgeZone) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	if zone.ID == "" {

		return fmt.Errorf("zone ID cannot be empty")

	}

	r.edgeZones[zone.ID] = zone

	r.Log.Info("Created edge zone", "zoneID", zone.ID, "name", zone.Name)

	return nil

}

// UpdateZone performs updatezone operation.

func (r *EdgeController) UpdateZone(zone *EdgeZone) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	if _, exists := r.edgeZones[zone.ID]; !exists {

		return fmt.Errorf("zone %s not found", zone.ID)

	}

	r.edgeZones[zone.ID] = zone

	r.Log.Info("Updated edge zone", "zoneID", zone.ID)

	return nil

}

// GetZone performs getzone operation.

func (r *EdgeController) GetZone(zoneID string) (*EdgeZone, error) {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	zone, exists := r.edgeZones[zoneID]

	if !exists {

		return nil, fmt.Errorf("zone %s not found", zoneID)

	}

	return zone, nil

}

// DeleteZone performs deletezone operation.

func (r *EdgeController) DeleteZone(zoneID string) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	delete(r.edgeZones, zoneID)

	r.Log.Info("Deleted edge zone", "zoneID", zoneID)

	return nil

}

// Network Intent Processing.

func (r *EdgeController) ProcessNetworkIntent(intent *nephoran.NetworkIntent) (*EdgeOptimizationPlan, error) {

	plan := &EdgeOptimizationPlan{

		OptimizationType: "",

		TargetNodes: []string{},

		Actions: []string{},
	}

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	switch intent.Spec.IntentType {

	case "low-latency":

		plan.OptimizationType = "latency-optimization"

		// Find nodes with lowest latency.

		for id, node := range r.edgeNodes {

			if node.Status == EdgeNodeActive && node.Resources.CPUUtilization < 0.7 {

				plan.TargetNodes = append(plan.TargetNodes, id)

			}

		}

		plan.Actions = append(plan.Actions, "enable-edge-processing", "optimize-routing")

	case "high-throughput":

		plan.OptimizationType = "throughput-optimization"

		// Find all available nodes.

		for id, node := range r.edgeNodes {

			if node.Status == EdgeNodeActive {

				plan.TargetNodes = append(plan.TargetNodes, id)

			}

		}

		plan.Actions = append(plan.Actions, "enable-load-balancing", "optimize-bandwidth")

	case "ml-inference":

		plan.OptimizationType = "ml-optimization"

		// Find GPU-enabled nodes.

		for id, node := range r.edgeNodes {

			if node.Status == EdgeNodeActive && node.Capabilities.GPUEnabled {

				plan.TargetNodes = append(plan.TargetNodes, id)

			}

		}

		plan.Actions = append(plan.Actions, "deploy-ml-model", "enable-gpu-acceleration")

	default:

		plan.OptimizationType = "general-optimization"

	}

	plan.EstimatedBenefit = 0.8 // Placeholder

	return plan, nil

}

// Health Monitoring Methods.

func (r *EdgeController) CheckNodeHealth(nodeID string) (*NodeHealth, error) {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	node, exists := r.edgeNodes[nodeID]

	if !exists {

		return nil, fmt.Errorf("node %s not found", nodeID)

	}

	health := &NodeHealth{

		NodeID: nodeID,

		Status: node.Status,

		IsHealthy: true,

		Issues: []string{},
	}

	// Check various health metrics.

	if node.HealthMetrics.Latency > float64(r.config.MaxLatencyMs) {

		health.IsHealthy = false

		health.Issues = append(health.Issues, "high latency")

		health.Status = EdgeNodeDegraded

	}

	if node.HealthMetrics.PacketLoss > 0.05 {

		health.IsHealthy = false

		health.Issues = append(health.Issues, "high packet loss")

		health.Status = EdgeNodeDegraded

	}

	if time.Since(node.LastSeen) > time.Minute {

		health.IsHealthy = false

		health.Issues = append(health.Issues, "node offline")

		health.Status = EdgeNodeOffline

	}

	return health, nil

}

// UpdateHealthMetrics performs updatehealthmetrics operation.

func (r *EdgeController) UpdateHealthMetrics(nodeID string, metrics EdgeHealthMetrics) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	node, exists := r.edgeNodes[nodeID]

	if !exists {

		return fmt.Errorf("node %s not found", nodeID)

	}

	node.HealthMetrics = metrics

	node.LastSeen = time.Now()

	// Update status based on metrics.

	if metrics.Latency > float64(r.config.MaxLatencyMs) || metrics.PacketLoss > 0.05 {

		node.Status = EdgeNodeDegraded

	} else {

		node.Status = EdgeNodeActive

	}

	return nil

}

// DetectUnhealthyNodes performs detectunhealthynodes operation.

func (r *EdgeController) DetectUnhealthyNodes() []string {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	unhealthyNodes := []string{}

	for id, node := range r.edgeNodes {

		if node.Status != EdgeNodeActive || time.Since(node.LastSeen) > time.Minute {

			unhealthyNodes = append(unhealthyNodes, id)

		}

	}

	return unhealthyNodes

}

// Resource Allocation Methods.

func (r *EdgeController) AllocateResources(request ResourceRequest) (*ResourceAllocation, error) {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	// Find node with sufficient resources.

	var selectedNode *EdgeNode

	var selectedNodeID string

	for id, node := range r.edgeNodes {

		if node.Status != EdgeNodeActive {

			continue

		}

		// Check if node has enough resources.

		availableCPU := float64(node.Capabilities.ComputeCores) * (1 - node.Resources.CPUUtilization)

		availableMemory := node.Capabilities.MemoryGB * (1 - node.Resources.MemoryUtilization)

		availableStorage := node.Capabilities.StorageGB * (1 - node.Resources.StorageUtilization)

		if availableCPU >= float64(request.CPUCores) &&

			availableMemory >= request.MemoryGB &&

			availableStorage >= request.StorageGB {

			if selectedNode == nil || node.Resources.CPUUtilization < selectedNode.Resources.CPUUtilization {

				selectedNode = node

				selectedNodeID = id

			}

		}

	}

	if selectedNode == nil {

		return nil, fmt.Errorf("insufficient resources available")

	}

	// Update node utilization.

	cpuIncrease := float64(request.CPUCores) / float64(selectedNode.Capabilities.ComputeCores)

	memoryIncrease := request.MemoryGB / selectedNode.Capabilities.MemoryGB

	storageIncrease := request.StorageGB / selectedNode.Capabilities.StorageGB

	selectedNode.Resources.CPUUtilization += cpuIncrease

	selectedNode.Resources.MemoryUtilization += memoryIncrease

	selectedNode.Resources.StorageUtilization += storageIncrease

	allocation := &ResourceAllocation{

		AllocationID: fmt.Sprintf("alloc-%d", time.Now().Unix()),

		NodeID: selectedNodeID,

		Request: request,

		AllocatedAt: time.Now(),
	}

	return allocation, nil

}

// ReleaseResources performs releaseresources operation.

func (r *EdgeController) ReleaseResources(nodeID string, request ResourceRequest) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	node, exists := r.edgeNodes[nodeID]

	if !exists {

		return fmt.Errorf("node %s not found", nodeID)

	}

	// Update node utilization.

	cpuDecrease := float64(request.CPUCores) / float64(node.Capabilities.ComputeCores)

	memoryDecrease := request.MemoryGB / node.Capabilities.MemoryGB

	storageDecrease := request.StorageGB / node.Capabilities.StorageGB

	node.Resources.CPUUtilization -= cpuDecrease

	node.Resources.MemoryUtilization -= memoryDecrease

	node.Resources.StorageUtilization -= storageDecrease

	// Ensure utilization doesn't go negative.

	if node.Resources.CPUUtilization < 0 {

		node.Resources.CPUUtilization = 0

	}

	if node.Resources.MemoryUtilization < 0 {

		node.Resources.MemoryUtilization = 0

	}

	if node.Resources.StorageUtilization < 0 {

		node.Resources.StorageUtilization = 0

	}

	return nil

}

// Failover Methods.

func (r *EdgeController) InitiateFailover(sourceNodeID string) (*FailoverPlan, error) {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	sourceNode, exists := r.edgeNodes[sourceNodeID]

	if !exists {

		return nil, fmt.Errorf("source node %s not found", sourceNodeID)

	}

	// Find suitable target node.

	var targetNodeID string

	minUtilization := 1.0

	for id, node := range r.edgeNodes {

		if id == sourceNodeID || node.Status != EdgeNodeActive {

			continue

		}

		// Check if in same zone.

		if node.Zone == sourceNode.Zone {

			avgUtilization := (node.Resources.CPUUtilization +

				node.Resources.MemoryUtilization +

				node.Resources.StorageUtilization) / 3

			if avgUtilization < minUtilization {

				targetNodeID = id

				minUtilization = avgUtilization

			}

		}

	}

	if targetNodeID == "" {

		return nil, fmt.Errorf("no suitable target node found for failover")

	}

	plan := &FailoverPlan{

		ID: fmt.Sprintf("failover-%d", time.Now().Unix()),

		SourceNode: sourceNodeID,

		TargetNode: targetNodeID,

		ServicesToMigrate: sourceNode.LocalServices,

		Status: "planned",

		CreatedAt: time.Now(),
	}

	return plan, nil

}

// ExecuteFailover performs executefailover operation.

func (r *EdgeController) ExecuteFailover(plan *FailoverPlan) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	targetNode, exists := r.edgeNodes[plan.TargetNode]

	if !exists {

		return fmt.Errorf("target node %s not found", plan.TargetNode)

	}

	sourceNode, exists := r.edgeNodes[plan.SourceNode]

	if exists {

		// Clear services from source node.

		sourceNode.LocalServices = []EdgeService{}

	}

	// Migrate services to target node.

	targetNode.LocalServices = append(targetNode.LocalServices, plan.ServicesToMigrate...)

	r.Log.Info("Executed failover", "source", plan.SourceNode, "target", plan.TargetNode,

		"services", len(plan.ServicesToMigrate))

	return nil

}

// InitiateFailback performs initiatefailback operation.

func (r *EdgeController) InitiateFailback(primaryNodeID, backupNodeID string) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	primaryNode, exists := r.edgeNodes[primaryNodeID]

	if !exists {

		return fmt.Errorf("primary node %s not found", primaryNodeID)

	}

	backupNode, exists := r.edgeNodes[backupNodeID]

	if !exists {

		return fmt.Errorf("backup node %s not found", backupNodeID)

	}

	// Move services back to primary.

	primaryNode.LocalServices = append(primaryNode.LocalServices, backupNode.LocalServices...)

	backupNode.LocalServices = []EdgeService{}

	r.Log.Info("Initiated failback", "primary", primaryNodeID, "backup", backupNodeID)

	return nil

}

// ML Capabilities Methods.

func (r *EdgeController) DeployMLModel(model *MLModel, nodeID string) (*MLDeployment, error) {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	node, exists := r.edgeNodes[nodeID]

	if !exists {

		return nil, fmt.Errorf("node %s not found", nodeID)

	}

	if model.RequiresGPU && !node.Capabilities.GPUEnabled {

		return nil, fmt.Errorf("node %s does not have GPU support", nodeID)

	}

	deployment := &MLDeployment{

		ID: fmt.Sprintf("ml-deploy-%d", time.Now().Unix()),

		ModelID: model.ID,

		NodeID: nodeID,

		Status: "deployed",

		Endpoint: fmt.Sprintf("http://%s:8080/inference/%s", node.Name, model.ID),

		DeployedAt: time.Now(),
	}

	// Add to node's services.

	service := EdgeService{

		Name: model.Name,

		Type: "ml-inference",

		Status: "running",

		Endpoint: deployment.Endpoint,
	}

	node.LocalServices = append(node.LocalServices, service)

	return deployment, nil

}

// RunInference performs runinference operation.

func (r *EdgeController) RunInference(nodeID string, request *InferenceRequest) (*InferenceResult, error) {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	_, exists := r.edgeNodes[nodeID]

	if !exists {

		return nil, fmt.Errorf("node %s not found", nodeID)

	}

	// Simulate inference.

	result := &InferenceResult{

		RequestID: fmt.Sprintf("inf-%d", time.Now().Unix()),

		Output: []byte("inference-result"),

		InferenceTimeMs: 50, // Simulated

		Confidence: 0.95,
	}

	return result, nil

}

// OptimizeMLDeployment performs optimizemldeployment operation.

func (r *EdgeController) OptimizeMLDeployment(modelID string) (*MLOptimization, error) {

	optimization := &MLOptimization{

		ModelID: modelID,

		Recommendations: []string{

			"quantization",

			"batching",

			"model-pruning",

			"edge-specific-optimization",
		},

		EstimatedSpeedup: 2.5,
	}

	return optimization, nil

}

// Caching Methods.

func (r *EdgeController) ConfigureCache(nodeID string, config *CacheConfig) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	node, exists := r.edgeNodes[nodeID]

	if !exists {

		return fmt.Errorf("node %s not found", nodeID)

	}

	if !node.Capabilities.CacheEnabled {

		return fmt.Errorf("node %s does not support caching", nodeID)

	}

	// Store cache configuration (in real implementation, this would configure the actual cache).

	r.Log.Info("Configured cache", "nodeID", nodeID, "maxSize", config.MaxSizeGB)

	return nil

}

// CacheContent performs cachecontent operation.

func (r *EdgeController) CacheContent(nodeID string, content *CacheContent) error {

	r.mutex.Lock()

	defer r.mutex.Unlock()

	node, exists := r.edgeNodes[nodeID]

	if !exists {

		return fmt.Errorf("node %s not found", nodeID)

	}

	if !node.Capabilities.CacheEnabled {

		return fmt.Errorf("node %s does not support caching", nodeID)

	}

	// Cache the content (simulated).

	r.Log.Info("Cached content", "nodeID", nodeID, "contentID", content.ID, "size", content.SizeMB)

	return nil

}

// GetCacheStats performs getcachestats operation.

func (r *EdgeController) GetCacheStats(nodeID string) (*CacheStats, error) {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	node, exists := r.edgeNodes[nodeID]

	if !exists {

		return nil, fmt.Errorf("node %s not found", nodeID)

	}

	if !node.Capabilities.CacheEnabled {

		return nil, fmt.Errorf("node %s does not support caching", nodeID)

	}

	// Return simulated stats.

	stats := &CacheStats{

		HitRate: 0.85,

		MissRate: 0.15,

		UsedGB: 25.5,

		AvailableGB: 74.5,

		EvictionRate: 0.05,
	}

	return stats, nil

}

// Analytics Methods.

func (r *EdgeController) GenerateZoneReport(zoneID string) (*ZoneReport, error) {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	zone, exists := r.edgeZones[zoneID]

	if !exists {

		return nil, fmt.Errorf("zone %s not found", zoneID)

	}

	report := &ZoneReport{

		ZoneID: zoneID,

		TotalNodes: 0,

		ActiveNodes: 0,

		AverageLatency: 0,

		TotalCapacity: zone.TotalCapacity,

		UsedCapacity: zone.UtilizedCapacity,

		HealthScore: 0,
	}

	totalLatency := 0.0

	for _, nodeID := range zone.Nodes {

		if node, exists := r.edgeNodes[nodeID]; exists {

			report.TotalNodes++

			if node.Status == EdgeNodeActive {

				report.ActiveNodes++

			}

			totalLatency += node.HealthMetrics.Latency

		}

	}

	if report.TotalNodes > 0 {

		report.AverageLatency = totalLatency / float64(report.TotalNodes)

		report.HealthScore = float64(report.ActiveNodes) / float64(report.TotalNodes)

	}

	return report, nil

}

// PredictResourceDemand performs predictresourcedemand operation.

func (r *EdgeController) PredictResourceDemand(zoneID string, horizon time.Duration) (*ResourcePrediction, error) {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	_, exists := r.edgeZones[zoneID]

	if !exists {

		return nil, fmt.Errorf("zone %s not found", zoneID)

	}

	// Simple prediction (in real implementation, would use ML models).

	prediction := &ResourcePrediction{

		ZoneID: zoneID,

		TimeHorizon: horizon,

		PredictedCPU: 0.75,

		PredictedMemory: 0.80,

		PredictedStorage: 0.60,

		Confidence: 0.85,
	}

	return prediction, nil

}

// OptimizeNodePlacement performs optimizenodeplacement operation.

func (r *EdgeController) OptimizeNodePlacement() (*PlacementOptimization, error) {

	r.mutex.RLock()

	defer r.mutex.RUnlock()

	optimization := &PlacementOptimization{

		Recommendations: []PlacementRecommendation{},

		EstimatedSavings: 0,
	}

	// Analyze node utilization and make recommendations.

	for id, node := range r.edgeNodes {

		avgUtilization := (node.Resources.CPUUtilization +

			node.Resources.MemoryUtilization +

			node.Resources.StorageUtilization) / 3

		if avgUtilization < 0.2 {

			optimization.Recommendations = append(optimization.Recommendations, PlacementRecommendation{

				NodeID: id,

				Action: "consolidate",

				Reason: "low utilization",

				Priority: 2,
			})

			optimization.EstimatedSavings += 1000 // Placeholder savings

		} else if avgUtilization > 0.9 {

			optimization.Recommendations = append(optimization.Recommendations, PlacementRecommendation{

				NodeID: id,

				Action: "scale-out",

				Reason: "high utilization",

				Priority: 1,
			})

		}

	}

	// Sort by priority.

	sort.Slice(optimization.Recommendations, func(i, j int) bool {

		return optimization.Recommendations[i].Priority < optimization.Recommendations[j].Priority

	})

	return optimization, nil

}
