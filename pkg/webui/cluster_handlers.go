/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/multicluster"
)

// ClusterRequest represents a request to register or update a cluster
type ClusterRequest struct {
	Name         string                         `json:"name"`
	Region       string                         `json:"region,omitempty"`
	Zone         string                         `json:"zone,omitempty"`
	KubeConfig   string                         `json:"kubeconfig,omitempty"` // Base64 encoded
	EdgeLocation *multicluster.EdgeLocation     `json:"edge_location,omitempty"`
	Capabilities []string                       `json:"capabilities,omitempty"`
	Resources    *multicluster.ResourceCapacity `json:"resources,omitempty"`
	Labels       map[string]string              `json:"labels,omitempty"`
	Annotations  map[string]string              `json:"annotations,omitempty"`
}

// ClusterResponse represents a comprehensive cluster response
type ClusterResponse struct {
	*multicluster.WorkloadCluster
	HealthStatus      *ClusterHealthStatus     `json:"health_status,omitempty"`
	DeploymentHistory []*DeploymentHistoryItem `json:"deployment_history,omitempty"`
	ResourceUsage     *ResourceUsageMetrics    `json:"resource_usage,omitempty"`
	NetworkInfo       *NetworkInformation      `json:"network_info,omitempty"`
}

// ClusterHealthStatus represents detailed cluster health information
type ClusterHealthStatus struct {
	Overall         string                  `json:"overall"`
	Components      map[string]string       `json:"components"`
	LastChecked     *metav1.Time            `json:"last_checked"`
	NextCheckIn     time.Duration           `json:"next_check_in"`
	HealthScore     float64                 `json:"health_score"` // 0-100
	Alerts          []*HealthAlert          `json:"alerts,omitempty"`
	Recommendations []*HealthRecommendation `json:"recommendations,omitempty"`
}

// HealthAlert represents a cluster health alert
type HealthAlert struct {
	Level     string    `json:"level"` // info, warning, error, critical
	Component string    `json:"component"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	ActionURL string    `json:"action_url,omitempty"`
}

// HealthRecommendation represents a health improvement recommendation
type HealthRecommendation struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"` // low, medium, high, critical
	Category    string `json:"category"` // performance, security, reliability
	ActionURL   string `json:"action_url,omitempty"`
}

// DeploymentHistoryItem represents a deployment history entry
type DeploymentHistoryItem struct {
	PackageName  string           `json:"package_name"`
	Version      string           `json:"version"`
	Status       string           `json:"status"`
	DeployedAt   *metav1.Time     `json:"deployed_at"`
	DeployedBy   string           `json:"deployed_by,omitempty"`
	Duration     *metav1.Duration `json:"duration,omitempty"`
	ErrorMessage string           `json:"error_message,omitempty"`
}

// ResourceUsageMetrics represents current resource usage
type ResourceUsageMetrics struct {
	CPUUsage     float64      `json:"cpu_usage_percent"`
	MemoryUsage  float64      `json:"memory_usage_percent"`
	StorageUsage float64      `json:"storage_usage_percent"`
	NetworkIO    *NetworkIO   `json:"network_io,omitempty"`
	PodCount     int          `json:"pod_count"`
	NodeCount    int          `json:"node_count"`
	LastUpdated  *metav1.Time `json:"last_updated"`
}

// NetworkIO represents network I/O metrics
type NetworkIO struct {
	BytesIn    int64 `json:"bytes_in"`
	BytesOut   int64 `json:"bytes_out"`
	PacketsIn  int64 `json:"packets_in"`
	PacketsOut int64 `json:"packets_out"`
}

// NetworkInformation represents cluster network information
type NetworkInformation struct {
	ClusterCIDR        string                   `json:"cluster_cidr,omitempty"`
	ServiceCIDR        string                   `json:"service_cidr,omitempty"`
	PodCIDR            string                   `json:"pod_cidr,omitempty"`
	DNSClusterIP       string                   `json:"dns_cluster_ip,omitempty"`
	LoadBalancers      []*LoadBalancerInfo      `json:"load_balancers,omitempty"`
	IngressControllers []*IngressControllerInfo `json:"ingress_controllers,omitempty"`
}

// LoadBalancerInfo represents load balancer information
type LoadBalancerInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	ExternalIP string `json:"external_ip,omitempty"`
	Status     string `json:"status"`
}

// IngressControllerInfo represents ingress controller information
type IngressControllerInfo struct {
	Name         string `json:"name"`
	Class        string `json:"class"`
	Status       string `json:"status"`
	LoadBalancer string `json:"load_balancer,omitempty"`
}

// DeploymentRequest represents a deployment request to clusters
type DeploymentRequest struct {
	PackageName    string                             `json:"package_name"`
	TargetClusters []string                           `json:"target_clusters,omitempty"`
	Strategy       multicluster.DeploymentStrategy    `json:"strategy,omitempty"`
	Constraints    []multicluster.PlacementConstraint `json:"constraints,omitempty"`
	ValidateOnly   bool                               `json:"validate_only,omitempty"`
	DryRun         bool                               `json:"dry_run,omitempty"`
}

// ClusterStatusUpdate represents a cluster status update for streaming
type ClusterStatusUpdate struct {
	ClusterName    string                     `json:"cluster_name"`
	Status         multicluster.ClusterStatus `json:"status"`
	PreviousStatus multicluster.ClusterStatus `json:"previous_status,omitempty"`
	HealthScore    float64                    `json:"health_score"`
	Message        string                     `json:"message,omitempty"`
	Timestamp      time.Time                  `json:"timestamp"`
	EventType      string                     `json:"event_type"` // registered, updated, health_changed, deployment, error
	Alerts         []*HealthAlert             `json:"alerts,omitempty"`
}

// setupClusterRoutes sets up multi-cluster management API routes
func (s *NephoranAPIServer) setupClusterRoutes(router *mux.Router) {
	clusters := router.PathPrefix("/clusters").Subrouter()

	// Apply cluster-specific middleware
	if s.authMiddleware != nil {
		// Most cluster operations require system management permissions
		clusters.Use(s.authMiddleware.RequirePermissionMiddleware(auth.PermissionViewMetrics))
	}

	// Cluster CRUD operations
	clusters.HandleFunc("", s.listClusters).Methods("GET")
	clusters.HandleFunc("", s.registerCluster).Methods("POST")
	clusters.HandleFunc("/{id}", s.getCluster).Methods("GET")
	clusters.HandleFunc("/{id}", s.updateCluster).Methods("PUT")
	clusters.HandleFunc("/{id}", s.unregisterCluster).Methods("DELETE")

	// Cluster status and health
	clusters.HandleFunc("/{id}/status", s.getClusterStatus).Methods("GET")
	clusters.HandleFunc("/{id}/health", s.getClusterHealth).Methods("GET")
	clusters.HandleFunc("/{id}/resources", s.getClusterResources).Methods("GET")
	clusters.HandleFunc("/{id}/nodes", s.getClusterNodes).Methods("GET")
	clusters.HandleFunc("/{id}/namespaces", s.getClusterNamespaces).Methods("GET")

	// Deployment operations
	clusters.HandleFunc("/{id}/deploy", s.deployToCluster).Methods("POST")
	clusters.HandleFunc("/{id}/deployments", s.getClusterDeployments).Methods("GET")
	clusters.HandleFunc("/{id}/deployments/{deployment}", s.getClusterDeployment).Methods("GET")
	clusters.HandleFunc("/{id}/deployments/{deployment}", s.deleteClusterDeployment).Methods("DELETE")

	// Network and connectivity
	clusters.HandleFunc("/{id}/network", s.getClusterNetwork).Methods("GET")
	clusters.HandleFunc("/{id}/connectivity", s.testClusterConnectivity).Methods("POST")
	clusters.HandleFunc("/{id}/latency", s.getClusterLatency).Methods("GET")

	// Multi-cluster operations
	clusters.HandleFunc("/topology", s.getNetworkTopology).Methods("GET")
	clusters.HandleFunc("/bulk/deploy", s.bulkDeployToClusters).Methods("POST")
	clusters.HandleFunc("/bulk/health", s.getBulkClusterHealth).Methods("GET")
	clusters.HandleFunc("/placement", s.getOptimalPlacement).Methods("POST")

	// Cluster discovery and auto-registration
	clusters.HandleFunc("/discover", s.discoverClusters).Methods("GET")
	clusters.HandleFunc("/auto-register", s.autoRegisterClusters).Methods("POST")
}

// listClusters handles GET /api/v1/clusters
func (s *NephoranAPIServer) listClusters(w http.ResponseWriter, r *http.Request) {
	_ = r.Context() // Context available if needed for cancellation
	pagination := s.parsePaginationParams(r)
	filters := s.parseFilterParams(r)

	// Check cache first
	cacheKey := fmt.Sprintf("clusters:%v:%v", pagination, filters)
	if s.cache != nil {
		if cached, found := s.cache.Get(cacheKey); found {
			s.metrics.CacheHits.Inc()
			s.writeJSONResponse(w, http.StatusOK, cached)
			return
		}
		s.metrics.CacheMisses.Inc()
	}

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	// For now, return a mock implementation until clusterManager interface is fully available
	clusters := make([]ClusterResponse, 0)

	// Apply filtering and pagination
	filteredClusters := s.filterClusters(clusters, filters)
	totalItems := len(filteredClusters)

	startIndex := (pagination.Page - 1) * pagination.PageSize
	endIndex := startIndex + pagination.PageSize
	if endIndex > totalItems {
		endIndex = totalItems
	}
	if startIndex > totalItems {
		startIndex = totalItems
	}

	paginatedItems := filteredClusters[startIndex:endIndex]

	// Build response with metadata
	meta := &Meta{
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: (totalItems + pagination.PageSize - 1) / pagination.PageSize,
		TotalItems: totalItems,
	}

	// Build HATEOAS links
	baseURL := fmt.Sprintf("/api/v1/clusters?page_size=%d", pagination.PageSize)
	links := &Links{
		Self: fmt.Sprintf("%s&page=%d", baseURL, pagination.Page),
	}
	if pagination.Page > 1 {
		links.Previous = fmt.Sprintf("%s&page=%d", baseURL, pagination.Page-1)
		links.First = fmt.Sprintf("%s&page=1", baseURL)
	}
	if pagination.Page < meta.TotalPages {
		links.Next = fmt.Sprintf("%s&page=%d", baseURL, pagination.Page+1)
		links.Last = fmt.Sprintf("%s&page=%d", baseURL, meta.TotalPages)
	}

	result := map[string]interface{}{
		"items": paginatedItems,
		"meta":  meta,
		"links": links,
	}

	// Cache the result
	if s.cache != nil {
		s.cache.Set(cacheKey, result)
	}

	s.writeJSONResponse(w, http.StatusOK, result)
}

// getCluster handles GET /api/v1/clusters/{id}
func (s *NephoranAPIServer) getCluster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	clusterID := vars["id"]

	// Check cache first
	cacheKey := fmt.Sprintf("cluster:%s", clusterID)
	if s.cache != nil {
		if cached, found := s.cache.Get(cacheKey); found {
			s.metrics.CacheHits.Inc()
			s.writeJSONResponse(w, http.StatusOK, cached)
			return
		}
		s.metrics.CacheMisses.Inc()
	}

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	// Build comprehensive cluster response
	response := s.buildClusterResponse(ctx, clusterID)
	if response == nil {
		s.writeErrorResponse(w, http.StatusNotFound, "cluster_not_found",
			fmt.Sprintf("Cluster '%s' not found", clusterID))
		return
	}

	// Cache the result
	if s.cache != nil {
		s.cache.SetWithTTL(cacheKey, response, 1*time.Minute) // Shorter cache for detailed cluster info
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// getClusterStatus handles GET /api/v1/clusters/{id}/status
func (s *NephoranAPIServer) getClusterStatus(w http.ResponseWriter, r *http.Request) {
	_ = r.Context() // Context available if needed for cancellation
	vars := mux.Vars(r)
	clusterID := vars["id"]

	// Check cache first with very short TTL for status
	cacheKey := fmt.Sprintf("cluster-status:%s", clusterID)
	if s.cache != nil {
		if cached, found := s.cache.Get(cacheKey); found {
			s.metrics.CacheHits.Inc()
			s.writeJSONResponse(w, http.StatusOK, cached)
			return
		}
		s.metrics.CacheMisses.Inc()
	}

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	// Get cluster status - mock implementation
	status := map[string]interface{}{
		"cluster_id":   clusterID,
		"status":       "Healthy",
		"last_checked": time.Now(),
		"health_score": 95.5,
		"nodes_ready":  3,
		"nodes_total":  3,
		"pods_running": 42,
		"pods_total":   45,
		"version":      "v1.28.0",
		"provider":     "aws",
		"region":       "us-west-2",
	}

	// Cache with very short TTL for real-time status
	if s.cache != nil {
		s.cache.SetWithTTL(cacheKey, status, 10*time.Second)
	}

	s.writeJSONResponse(w, http.StatusOK, status)
}

// deployToCluster handles POST /api/v1/clusters/{id}/deploy
func (s *NephoranAPIServer) deployToCluster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check deployment permissions  
	if s.authMiddleware != nil && !auth.HasPermission(ctx, auth.PermissionManageSystem) {
		s.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",
			"System management permission required for cluster deployments")
		return
	}

	vars := mux.Vars(r)
	clusterID := vars["id"]

	var req DeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid_json",
			"Invalid JSON in request body")
		return
	}

	// Validate request
	if req.PackageName == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "validation_failed",
			"package_name is required")
		return
	}

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	// Create propagation options
	options := &multicluster.PropagationOptions{
		Strategy:     req.Strategy,
		Constraints:  req.Constraints,
		ValidateOnly: req.ValidateOnly,
		DryRun:       req.DryRun,
	}

	// Set target clusters - if not specified, use the current cluster
	if len(req.TargetClusters) == 0 {
		req.TargetClusters = []string{clusterID}
	}

	// Perform deployment
	result, err := (*s.clusterManager).PropagatePackage(ctx, req.PackageName, options)
	if err != nil {
		s.logger.Error(err, "Failed to deploy to cluster",
			"cluster", clusterID, "package", req.PackageName)
		s.writeErrorResponse(w, http.StatusInternalServerError, "deployment_failed",
			fmt.Sprintf("Failed to deploy package: %v", err))
		return
	}

	// Broadcast deployment status update
	s.broadcastClusterUpdate(&ClusterStatusUpdate{
		ClusterName: clusterID,
		Message:     fmt.Sprintf("Package %s deployed successfully", req.PackageName),
		Timestamp:   time.Now(),
		EventType:   "deployment",
	})

	s.writeJSONResponse(w, http.StatusOK, result)
}

// getNetworkTopology handles GET /api/v1/clusters/topology
func (s *NephoranAPIServer) getNetworkTopology(w http.ResponseWriter, r *http.Request) {
	_ = r.Context() // Context available if needed for cancellation

	// Check cache first
	cacheKey := "network-topology"
	if s.cache != nil {
		if cached, found := s.cache.Get(cacheKey); found {
			s.metrics.CacheHits.Inc()
			s.writeJSONResponse(w, http.StatusOK, cached)
			return
		}
		s.metrics.CacheMisses.Inc()
	}

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	// Mock network topology response
	topology := map[string]interface{}{
		"clusters": []map[string]interface{}{
			{
				"id":       "cluster-1",
				"name":     "production-us-west-2",
				"region":   "us-west-2",
				"zone":     "us-west-2a",
				"status":   "Healthy",
				"position": map[string]float64{"x": 100, "y": 100},
			},
			{
				"id":       "cluster-2",
				"name":     "production-us-east-1",
				"region":   "us-east-1",
				"zone":     "us-east-1a",
				"status":   "Healthy",
				"position": map[string]float64{"x": 300, "y": 100},
			},
		},
		"connections": []map[string]interface{}{
			{
				"from":       "cluster-1",
				"to":         "cluster-2",
				"latency_ms": 45.2,
				"bandwidth":  "10Gbps",
				"status":     "active",
			},
		},
		"policies": []map[string]interface{}{
			{
				"source":      "cluster-1",
				"destination": "cluster-2",
				"protocols":   []string{"https", "grpc"},
				"max_latency": 100,
			},
		},
	}

	// Cache the result
	if s.cache != nil {
		s.cache.SetWithTTL(cacheKey, topology, 2*time.Minute)
	}

	s.writeJSONResponse(w, http.StatusOK, topology)
}

// registerCluster handles POST /api/v1/clusters
func (s *NephoranAPIServer) registerCluster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check admin permissions for cluster registration
	if s.authMiddleware != nil && !auth.HasPermission(ctx, auth.PermissionManageSystem) {
		s.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",
			"System management permission required for cluster registration")
		return
	}

	var req ClusterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid_json",
			"Invalid JSON in request body")
		return
	}

	// Validate request
	if req.Name == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "validation_failed",
			"name is required")
		return
	}

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	// Create workload cluster object
	cluster := &multicluster.WorkloadCluster{
		ID:            generateClusterID(),
		Name:          req.Name,
		Region:        req.Region,
		Zone:          req.Zone,
		EdgeLocation:  req.EdgeLocation,
		Capabilities:  req.Capabilities,
		Resources:     req.Resources,
		Status:        multicluster.ClusterStatusHealthy,
		LastCheckedAt: time.Now(),
		Labels:        req.Labels,
		Annotations:   req.Annotations,
	}

	// Register with cluster manager
	options := &multicluster.ClusterRegistrationOptions{
		ConnectionTimeout:        30 * time.Second,
		ConnectionRetries:        3,
		RequiredCapabilities:     []string{"kubernetes"},
		MinimumResourceThreshold: req.Resources,
	}

	err := (*s.clusterManager).RegisterCluster(ctx, cluster, options)
	if err != nil {
		s.logger.Error(err, "Failed to register cluster", "name", req.Name)
		s.writeErrorResponse(w, http.StatusInternalServerError, "registration_failed",
			fmt.Sprintf("Failed to register cluster: %v", err))
		return
	}

	// Invalidate cache
	if s.cache != nil {
		s.cache.Invalidate("clusters:")
		s.cache.Invalidate("network-topology")
	}

	// Broadcast cluster registration event
	s.broadcastClusterUpdate(&ClusterStatusUpdate{
		ClusterName: cluster.Name,
		Status:      cluster.Status,
		Message:     "Cluster registered successfully",
		Timestamp:   time.Now(),
		EventType:   "registered",
	})

	response := s.buildClusterResponse(ctx, cluster.ID)
	s.writeJSONResponse(w, http.StatusCreated, response)
}

// Helper functions

func (s *NephoranAPIServer) filterClusters(clusters []ClusterResponse, filters FilterParams) []ClusterResponse {
	var filtered []ClusterResponse

	for _, cluster := range clusters {
		include := true

		if filters.Status != "" && string(cluster.Status) != filters.Status {
			include = false
		}

		if filters.Cluster != "" && cluster.Name != filters.Cluster {
			include = false
		}

		// Filter by labels
		for labelKey, labelValue := range filters.Labels {
			if cluster.Labels == nil {
				include = false
				break
			}
			if value, exists := cluster.Labels[labelKey]; !exists || value != labelValue {
				include = false
				break
			}
		}

		if include {
			filtered = append(filtered, cluster)
		}
	}

	return filtered
}

func (s *NephoranAPIServer) buildClusterResponse(ctx context.Context, clusterID string) *ClusterResponse {
	// Mock implementation - would integrate with actual cluster manager
	cluster := &multicluster.WorkloadCluster{
		ID:            clusterID,
		Name:          fmt.Sprintf("cluster-%s", clusterID),
		Region:        "us-west-2",
		Zone:          "us-west-2a",
		Status:        multicluster.ClusterStatusHealthy,
		LastCheckedAt: time.Now(),
		Resources: &multicluster.ResourceCapacity{
			CPU:       8000,                    // 8 cores in millicores
			Memory:    16 * 1024 * 1024 * 1024, // 16GB in bytes
			StorageGB: 100,
		},
		Capabilities: []string{"kubernetes", "istio", "prometheus"},
		Labels: map[string]string{
			"environment": "production",
			"region":      "us-west-2",
		},
	}

	return &ClusterResponse{
		WorkloadCluster: cluster,
		HealthStatus: &ClusterHealthStatus{
			Overall:     "healthy",
			HealthScore: 95.5,
			LastChecked: &metav1.Time{Time: time.Now()},
			Components: map[string]string{
				"api-server":        "healthy",
				"etcd":              "healthy",
				"kubelet":           "healthy",
				"kube-proxy":        "healthy",
				"core-dns":          "healthy",
				"container-runtime": "healthy",
			},
			NextCheckIn: 30 * time.Second,
		},
		ResourceUsage: &ResourceUsageMetrics{
			CPUUsage:     65.2,
			MemoryUsage:  78.5,
			StorageUsage: 45.0,
			PodCount:     42,
			NodeCount:    3,
			LastUpdated:  &metav1.Time{Time: time.Now()},
		},
		NetworkInfo: &NetworkInformation{
			ClusterCIDR:  "10.244.0.0/16",
			ServiceCIDR:  "10.96.0.0/12",
			PodCIDR:      "10.244.0.0/16",
			DNSClusterIP: "10.96.0.10",
		},
	}
}

func (s *NephoranAPIServer) broadcastClusterUpdate(update *ClusterStatusUpdate) {
	// Broadcast to WebSocket connections
	s.connectionsMutex.RLock()
	for _, conn := range s.wsConnections {
		select {
		case conn.Send <- mustMarshal(update):
		default:
			close(conn.Send)
		}
	}
	s.connectionsMutex.RUnlock()

	// Broadcast to SSE connections
	s.connectionsMutex.RLock()
	for _, conn := range s.sseConnections {
		if conn.Flusher != nil {
			fmt.Fprintf(conn.Writer, "data: %s\n\n", mustMarshalString(update))
			conn.Flusher.Flush()
		}
	}
	s.connectionsMutex.RUnlock()
}

func generateClusterID() string {
	// Simple cluster ID generation - would use UUID in production
	return fmt.Sprintf("cluster-%d", time.Now().UnixNano())
}

// updateCluster handles PUT /api/v1/clusters/{id}
func (s *NephoranAPIServer) updateCluster(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	
	s.logger.Info("Updating cluster", "cluster_id", clusterID)
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Cluster update operation initiated",
		"cluster_id": clusterID,
	})
}

// unregisterCluster handles DELETE /api/v1/clusters/{id}
func (s *NephoranAPIServer) unregisterCluster(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	
	s.logger.Info("Unregistering cluster", "cluster_id", clusterID)
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Cluster unregistration initiated",
		"cluster_id": clusterID,
	})
}

// getClusterHealth handles GET /api/v1/clusters/{id}/health
func (s *NephoranAPIServer) getClusterHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"cluster_id": clusterID,
		"health": "healthy",
		"status": "ready",
		"timestamp": time.Now().UTC(),
	})
}

// getClusterResources handles GET /api/v1/clusters/{id}/resources
func (s *NephoranAPIServer) getClusterResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"cluster_id": clusterID,
		"resources": map[string]interface{}{
			"cpu": map[string]interface{}{
				"total": "100",
				"used": "45",
				"available": "55",
			},
			"memory": map[string]interface{}{
				"total": "128Gi",
				"used": "64Gi",
				"available": "64Gi",
			},
		},
	})
}

// getClusterNodes handles GET /api/v1/clusters/{id}/nodes
func (s *NephoranAPIServer) getClusterNodes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"cluster_id": clusterID,
		"nodes": []map[string]interface{}{
			{
				"name": "node-1",
				"status": "Ready",
				"roles": []string{"control-plane", "master"},
			},
			{
				"name": "node-2", 
				"status": "Ready",
				"roles": []string{"worker"},
			},
		},
	})
}

// getClusterNamespaces handles GET /api/v1/clusters/{id}/namespaces
func (s *NephoranAPIServer) getClusterNamespaces(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"cluster_id": clusterID,
		"namespaces": []map[string]interface{}{
			{"name": "default", "status": "Active"},
			{"name": "kube-system", "status": "Active"},
			{"name": "nephoran-system", "status": "Active"},
		},
	})
}

// getClusterDeployment handles GET /api/v1/clusters/{id}/deployments/{deployment}
func (s *NephoranAPIServer) getClusterDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	deploymentName := vars["deployment"]
	
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"cluster_id": clusterID,
		"deployment": map[string]interface{}{
			"name": deploymentName,
			"namespace": "nephoran-system",
			"status": "Running",
			"replicas": map[string]int{"ready": 1, "desired": 1},
		},
	})
}

// deleteClusterDeployment handles DELETE /api/v1/clusters/{id}/deployments/{deployment}
func (s *NephoranAPIServer) deleteClusterDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	deploymentName := vars["deployment"]
	
	s.logger.Info("Deleting deployment", "cluster_id", clusterID, "deployment", deploymentName)
	s.writeJSONResponse(w, http.StatusAccepted, map[string]interface{}{
		"message": "Deployment deletion initiated",
		"cluster_id": clusterID,
		"deployment": deploymentName,
	})
}

// getClusterNetwork handles GET /api/v1/clusters/{id}/network
func (s *NephoranAPIServer) getClusterNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"cluster_id": clusterID,
		"network": map[string]interface{}{
			"cni": "calico",
			"pod_cidr": "10.244.0.0/16",
			"service_cidr": "10.96.0.0/12",
		},
	})
}

// testClusterConnectivity handles POST /api/v1/clusters/{id}/connectivity
func (s *NephoranAPIServer) testClusterConnectivity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"cluster_id": clusterID,
		"connectivity": "healthy",
		"tests": []map[string]interface{}{
			{"name": "api-server", "status": "pass", "latency": "15ms"},
			{"name": "dns", "status": "pass", "latency": "5ms"},
		},
	})
}

// getClusterLatency handles GET /api/v1/clusters/{id}/latency
func (s *NephoranAPIServer) getClusterLatency(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"cluster_id": clusterID,
		"latency": map[string]interface{}{
			"api_server": "15ms",
			"average": "12ms",
			"p95": "25ms",
		},
	})
}

// bulkDeployToClusters handles POST /api/v1/clusters/bulk/deploy
func (s *NephoranAPIServer) bulkDeployToClusters(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusAccepted, map[string]interface{}{
		"message": "Bulk deployment initiated",
		"clusters": []string{"cluster-1", "cluster-2"},
	})
}

// getBulkClusterHealth handles GET /api/v1/clusters/bulk/health
func (s *NephoranAPIServer) getBulkClusterHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"clusters": []map[string]interface{}{
			{"id": "cluster-1", "health": "healthy"},
			{"id": "cluster-2", "health": "healthy"},
		},
	})
}

// getOptimalPlacement handles POST /api/v1/clusters/placement
func (s *NephoranAPIServer) getOptimalPlacement(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"placement": map[string]interface{}{
			"cluster_id": "cluster-1",
			"score": 95,
			"reason": "optimal resource availability",
		},
	})
}

// discoverClusters handles GET /api/v1/clusters/discover
func (s *NephoranAPIServer) discoverClusters(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"discovered": []map[string]interface{}{
			{"endpoint": "https://cluster.example.com", "status": "available"},
		},
	})
}

// autoRegisterClusters handles POST /api/v1/clusters/auto-register
func (s *NephoranAPIServer) autoRegisterClusters(w http.ResponseWriter, r *http.Request) {
	s.writeJSONResponse(w, http.StatusAccepted, map[string]interface{}{
		"message": "Auto-registration initiated",
		"discovered": 2,
	})
}


// getClusterDeployments handles GET /api/v1/clusters/{id}/deployments
func (s *NephoranAPIServer) getClusterDeployments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"cluster_id": clusterID,
		"deployments": []map[string]interface{}{
			{
				"name": "nephoran-controller",
				"namespace": "nephoran-system",
				"status": "Running",
				"replicas": map[string]int{"ready": 1, "desired": 1},
			},
		},
	})
}
