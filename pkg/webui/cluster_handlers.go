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
	"strings"
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
	result, err := s.clusterManager.PropagatePackage(ctx, req.PackageName, options)
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

	err := s.clusterManager.RegisterCluster(ctx, cluster, options)
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

// deleteClusterDeployment handles DELETE /api/v1/clusters/{id}/deployments/{deployment}
func (s *NephoranAPIServer) deleteClusterDeployment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	clusterID := vars["id"]
	deploymentID := vars["deployment"]

	// Check admin permissions for deployment deletion
	if s.authMiddleware != nil && !auth.HasPermission(ctx, auth.PermissionManageSystem) {
		s.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",
			"System management permission required for deployment deletion")
		return
	}

	s.logger.Info("Deleting cluster deployment",
		"cluster", clusterID, "deployment", deploymentID)

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	// Perform deletion (mock implementation)
	result := map[string]interface{}{
		"cluster_id":    clusterID,
		"deployment_id": deploymentID,
		"status":        "deleted",
		"message":       fmt.Sprintf("Deployment %s deleted from cluster %s", deploymentID, clusterID),
		"deleted_at":    time.Now(),
	}

	// Invalidate cache
	if s.cache != nil {
		s.cache.Invalidate(fmt.Sprintf("cluster-deployments:%s", clusterID))
		s.cache.Invalidate(fmt.Sprintf("cluster:%s", clusterID))
	}

	// Broadcast deletion event
	s.broadcastClusterUpdate(&ClusterStatusUpdate{
		ClusterName: clusterID,
		Message:     fmt.Sprintf("Deployment %s deleted", deploymentID),
		Timestamp:   time.Now(),
		EventType:   "deployment",
	})

	s.writeJSONResponse(w, http.StatusOK, result)
}

// getClusterNetwork handles GET /api/v1/clusters/{id}/network
func (s *NephoranAPIServer) getClusterNetwork(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]

	// Check cache first
	cacheKey := fmt.Sprintf("cluster-network:%s", clusterID)
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

	// Mock network information - would integrate with actual cluster manager
	networkInfo := map[string]interface{}{
		"cluster_id": clusterID,
		"network_info": &NetworkInformation{
			ClusterCIDR:  "10.244.0.0/16",
			ServiceCIDR:  "10.96.0.0/12",
			PodCIDR:      "10.244.0.0/16",
			DNSClusterIP: "10.96.0.10",
			LoadBalancers: []*LoadBalancerInfo{
				{
					Name:       "nginx-lb",
					Type:       "LoadBalancer",
					ExternalIP: "203.0.113.10",
					Status:     "Active",
				},
			},
			IngressControllers: []*IngressControllerInfo{
				{
					Name:         "nginx-ingress",
					Class:        "nginx",
					Status:       "Running",
					LoadBalancer: "nginx-lb",
				},
			},
		},
		"connectivity": map[string]interface{}{
			"internal_dns_resolved": true,
			"external_access":       true,
			"service_mesh_enabled":  true,
			"network_policies":      []string{"deny-all", "allow-dns", "allow-egress"},
		},
		"performance": map[string]interface{}{
			"bandwidth_utilization": 45.2,
			"latency_ms":            12.5,
			"packet_loss_percent":   0.01,
			"connection_count":      1250,
		},
		"last_updated": time.Now(),
	}

	// Cache the result
	if s.cache != nil {
		s.cache.Set(cacheKey, networkInfo)
	}

	s.writeJSONResponse(w, http.StatusOK, networkInfo)
}

// updateCluster handles PUT /api/v1/clusters/{id}
func (s *NephoranAPIServer) updateCluster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	clusterID := vars["id"]

	// Check admin permissions for cluster updates
	if s.authMiddleware != nil && !auth.HasPermission(ctx, auth.PermissionManageSystem) {
		s.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",
			"System management permission required for cluster updates")
		return
	}

	var req ClusterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid_json",
			"Invalid JSON in request body")
		return
	}

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	// Mock implementation - would integrate with actual cluster manager
	result := map[string]interface{}{
		"cluster_id":   clusterID,
		"name":         req.Name,
		"region":       req.Region,
		"zone":         req.Zone,
		"status":       "Updated",
		"updated_at":   time.Now(),
		"capabilities": req.Capabilities,
		"resources":    req.Resources,
	}

	// Invalidate cache
	if s.cache != nil {
		s.cache.Invalidate(fmt.Sprintf("cluster:%s", clusterID))
		s.cache.Invalidate("clusters:")
		s.cache.Invalidate("network-topology")
	}

	// Broadcast cluster update event
	s.broadcastClusterUpdate(&ClusterStatusUpdate{
		ClusterName: clusterID,
		Message:     fmt.Sprintf("Cluster %s updated successfully", req.Name),
		Timestamp:   time.Now(),
		EventType:   "updated",
	})

	s.writeJSONResponse(w, http.StatusOK, result)
}

// unregisterCluster handles DELETE /api/v1/clusters/{id}
func (s *NephoranAPIServer) unregisterCluster(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	clusterID := vars["id"]

	// Check admin permissions for cluster deletion
	if s.authMiddleware != nil && !auth.HasPermission(ctx, auth.PermissionManageSystem) {
		s.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",
			"System management permission required for cluster deletion")
		return
	}

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	s.logger.Info("Unregistering cluster", "cluster_id", clusterID)

	// Mock implementation - would integrate with actual cluster manager
	result := map[string]interface{}{
		"cluster_id":     clusterID,
		"status":         "Unregistered",
		"message":        fmt.Sprintf("Cluster %s unregistered successfully", clusterID),
		"unregistered_at": time.Now(),
	}

	// Invalidate cache
	if s.cache != nil {
		s.cache.Invalidate(fmt.Sprintf("cluster:%s", clusterID))
		s.cache.Invalidate("clusters:")
		s.cache.Invalidate("network-topology")
	}

	// Broadcast cluster deletion event
	s.broadcastClusterUpdate(&ClusterStatusUpdate{
		ClusterName: clusterID,
		Message:     fmt.Sprintf("Cluster %s unregistered", clusterID),
		Timestamp:   time.Now(),
		EventType:   "unregistered",
	})

	s.writeJSONResponse(w, http.StatusOK, result)
}

// getClusterHealth handles GET /api/v1/clusters/{id}/health
func (s *NephoranAPIServer) getClusterHealth(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]

	// Check cache first with very short TTL for health data
	cacheKey := fmt.Sprintf("cluster-health:%s", clusterID)
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

	// Mock health information - would integrate with actual cluster manager
	healthStatus := &ClusterHealthStatus{
		Overall:     "healthy",
		HealthScore: 95.8,
		LastChecked: &metav1.Time{Time: time.Now()},
		Components: map[string]string{
			"api-server":        "healthy",
			"etcd":              "healthy",
			"kubelet":           "healthy",
			"kube-proxy":        "healthy",
			"core-dns":          "healthy",
			"container-runtime": "healthy",
			"network-plugin":    "healthy",
			"storage-plugin":    "healthy",
		},
		NextCheckIn: 30 * time.Second,
		Alerts: []*HealthAlert{
			{
				Level:     "info",
				Component: "api-server",
				Message:   "API server response time slightly elevated (45ms avg)",
				Timestamp: time.Now().Add(-5 * time.Minute),
			},
		},
		Recommendations: []*HealthRecommendation{
			{
				Title:       "Optimize etcd performance",
				Description: "Consider defragmenting etcd database for better performance",
				Priority:    "low",
				Category:    "performance",
			},
		},
	}

	result := map[string]interface{}{
		"cluster_id":    clusterID,
		"health_status": healthStatus,
		"last_updated":  time.Now(),
	}

	// Cache with very short TTL for real-time health data
	if s.cache != nil {
		s.cache.SetWithTTL(cacheKey, result, 15*time.Second)
	}

	s.writeJSONResponse(w, http.StatusOK, result)
}

// getClusterResources handles GET /api/v1/clusters/{id}/resources
func (s *NephoranAPIServer) getClusterResources(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]

	// Check cache first
	cacheKey := fmt.Sprintf("cluster-resources:%s", clusterID)
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

	// Mock resource information - would integrate with actual cluster manager
	resourceInfo := map[string]interface{}{
		"cluster_id": clusterID,
		"capacity": map[string]interface{}{
			"cpu_cores":    48,
			"memory_gb":    192,
			"storage_gb":   2000,
			"gpu_count":    4,
			"network_gbps": 100,
		},
		"usage": &ResourceUsageMetrics{
			CPUUsage:     68.2,
			MemoryUsage:  74.5,
			StorageUsage: 42.1,
			NetworkIO: &NetworkIO{
				BytesIn:    125_000_000_000, // 125 GB
				BytesOut:   98_000_000_000,  // 98 GB
				PacketsIn:  1_500_000_000,
				PacketsOut: 1_200_000_000,
			},
			PodCount:    156,
			NodeCount:   12,
			LastUpdated: &metav1.Time{Time: time.Now()},
		},
		"allocatable": map[string]interface{}{
			"cpu_millicores": 44800, // 44.8 cores available (93.3% of capacity)
			"memory_bytes":   198_000_000_000, // ~198 GB
			"storage_bytes":  1_800_000_000_000, // ~1.8 TB
			"pods":           1100,
		},
		"requests": map[string]interface{}{
			"cpu_millicores": 30500, // ~68% utilization
			"memory_bytes":   147_000_000_000, // ~74% utilization
			"storage_bytes":  756_000_000_000, // ~42% utilization
		},
		"limits": map[string]interface{}{
			"cpu_millicores": 38200,
			"memory_bytes":   165_000_000_000,
			"storage_bytes":  950_000_000_000,
		},
		"node_breakdown": []map[string]interface{}{
			{
				"name":         "node-1",
				"instance_type": "m5.2xlarge",
				"cpu_usage":    65.2,
				"memory_usage": 71.8,
				"status":       "Ready",
			},
			{
				"name":         "node-2", 
				"instance_type": "m5.2xlarge",
				"cpu_usage":    72.1,
				"memory_usage": 68.9,
				"status":       "Ready",
			},
		},
		"last_updated": time.Now(),
	}

	// Cache the result
	if s.cache != nil {
		s.cache.SetWithTTL(cacheKey, resourceInfo, 30*time.Second)
	}

	s.writeJSONResponse(w, http.StatusOK, resourceInfo)
}

// getClusterNodes handles GET /api/v1/clusters/{id}/nodes
func (s *NephoranAPIServer) getClusterNodes(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	pagination := s.parsePaginationParams(r)

	// Check cache first
	cacheKey := fmt.Sprintf("cluster-nodes:%s:%v", clusterID, pagination)
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

	// Mock node information - would integrate with actual cluster manager
	allNodes := []map[string]interface{}{
		{
			"name":           "control-plane-1",
			"status":         "Ready",
			"role":           "control-plane",
			"instance_type":  "m5.xlarge",
			"zone":           "us-west-2a",
			"kubernetes_version": "v1.28.0",
			"container_runtime": "containerd://1.7.0",
			"cpu_capacity":      "4000m",
			"memory_capacity":   "16Gi",
			"pod_capacity":      110,
			"cpu_usage":         "850m",
			"memory_usage":      "8.2Gi",
			"pod_count":         42,
			"created_at":        time.Now().Add(-72 * time.Hour),
			"conditions": []map[string]interface{}{
				{"type": "Ready", "status": "True", "reason": "KubeletReady"},
				{"type": "DiskPressure", "status": "False", "reason": "KubeletHasNoDiskPressure"},
				{"type": "MemoryPressure", "status": "False", "reason": "KubeletHasSufficientMemory"},
			},
			"taints": []map[string]interface{}{
				{"key": "node-role.kubernetes.io/control-plane", "effect": "NoSchedule"},
			},
		},
		{
			"name":           "worker-1",
			"status":         "Ready", 
			"role":           "worker",
			"instance_type":  "m5.2xlarge",
			"zone":           "us-west-2a",
			"kubernetes_version": "v1.28.0",
			"container_runtime": "containerd://1.7.0",
			"cpu_capacity":      "8000m",
			"memory_capacity":   "32Gi",
			"pod_capacity":      110,
			"cpu_usage":         "5.2Gi",
			"memory_usage":      "18.7Gi",
			"pod_count":         78,
			"created_at":        time.Now().Add(-71 * time.Hour),
			"conditions": []map[string]interface{}{
				{"type": "Ready", "status": "True", "reason": "KubeletReady"},
				{"type": "DiskPressure", "status": "False", "reason": "KubeletHasNoDiskPressure"},
				{"type": "MemoryPressure", "status": "False", "reason": "KubeletHasSufficientMemory"},
			},
			"taints": []map[string]interface{}{},
		},
		{
			"name":           "worker-2",
			"status":         "Ready",
			"role":           "worker", 
			"instance_type":  "m5.2xlarge",
			"zone":           "us-west-2b",
			"kubernetes_version": "v1.28.0",
			"container_runtime": "containerd://1.7.0",
			"cpu_capacity":      "8000m",
			"memory_capacity":   "32Gi", 
			"pod_capacity":      110,
			"cpu_usage":         "6.1Gi",
			"memory_usage":      "21.3Gi",
			"pod_count":         89,
			"created_at":        time.Now().Add(-71 * time.Hour),
			"conditions": []map[string]interface{}{
				{"type": "Ready", "status": "True", "reason": "KubeletReady"},
				{"type": "DiskPressure", "status": "False", "reason": "KubeletHasNoDiskPressure"},
				{"type": "MemoryPressure", "status": "False", "reason": "KubeletHasSufficientMemory"},
			},
			"taints": []map[string]interface{}{},
		},
	}

	// Apply pagination
	totalItems := len(allNodes)
	startIndex := (pagination.Page - 1) * pagination.PageSize
	endIndex := startIndex + pagination.PageSize
	if endIndex > totalItems {
		endIndex = totalItems
	}
	if startIndex > totalItems {
		startIndex = totalItems
	}

	paginatedNodes := allNodes[startIndex:endIndex]

	// Build response with metadata
	meta := &Meta{
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: (totalItems + pagination.PageSize - 1) / pagination.PageSize,
		TotalItems: totalItems,
	}

	result := map[string]interface{}{
		"cluster_id":   clusterID,
		"nodes":        paginatedNodes,
		"meta":         meta,
		"summary": map[string]interface{}{
			"total_nodes":       len(allNodes),
			"ready_nodes":       3,
			"not_ready_nodes":   0,
			"control_plane_nodes": 1,
			"worker_nodes":      2,
		},
		"last_updated": time.Now(),
	}

	// Cache the result
	if s.cache != nil {
		s.cache.SetWithTTL(cacheKey, result, 45*time.Second)
	}

	s.writeJSONResponse(w, http.StatusOK, result)
}

// getClusterDeployments handles GET /api/v1/clusters/{id}/deployments
func (s *NephoranAPIServer) getClusterDeployments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	pagination := s.parsePaginationParams(r)
	filters := s.parseFilterParams(r)

	// Check cache first
	cacheKey := fmt.Sprintf("cluster-deployments:%s:%v:%v", clusterID, pagination, filters)
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

	// Mock deployment information - would integrate with actual cluster manager
	allDeployments := []*DeploymentHistoryItem{
		{
			PackageName:  "nginx-ingress-controller",
			Version:      "v1.8.1",
			Status:       "Deployed",
			DeployedAt:   &metav1.Time{Time: time.Now().Add(-2 * time.Hour)},
			DeployedBy:   "system",
			Duration:     &metav1.Duration{Duration: 45 * time.Second},
		},
		{
			PackageName:  "prometheus-operator",
			Version:      "v0.68.0",
			Status:       "Deployed",
			DeployedAt:   &metav1.Time{Time: time.Now().Add(-4 * time.Hour)},
			DeployedBy:   "admin@example.com",
			Duration:     &metav1.Duration{Duration: 2 * time.Minute + 15 * time.Second},
		},
		{
			PackageName:  "grafana",
			Version:      "v9.5.2",
			Status:       "Failed",
			DeployedAt:   &metav1.Time{Time: time.Now().Add(-6 * time.Hour)},
			DeployedBy:   "admin@example.com",
			Duration:     &metav1.Duration{Duration: 30 * time.Second},
			ErrorMessage: "insufficient storage space",
		},
	}

	// Apply filtering
	var filteredDeployments []*DeploymentHistoryItem
	for _, deployment := range allDeployments {
		include := true

		if filters.Status != "" && deployment.Status != filters.Status {
			include = false
		}

		if filters.Component != "" && !strings.Contains(deployment.PackageName, filters.Component) {
			include = false
		}

		if include {
			filteredDeployments = append(filteredDeployments, deployment)
		}
	}

	// Apply pagination
	totalItems := len(filteredDeployments)
	startIndex := (pagination.Page - 1) * pagination.PageSize
	endIndex := startIndex + pagination.PageSize
	if endIndex > totalItems {
		endIndex = totalItems
	}
	if startIndex > totalItems {
		startIndex = totalItems
	}

	paginatedDeployments := filteredDeployments[startIndex:endIndex]

	// Build response with metadata
	meta := &Meta{
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: (totalItems + pagination.PageSize - 1) / pagination.PageSize,
		TotalItems: totalItems,
	}

	result := map[string]interface{}{
		"cluster_id":   clusterID,
		"deployments":  paginatedDeployments,
		"meta":         meta,
		"summary": map[string]interface{}{
			"total_deployments":     len(allDeployments),
			"successful_deployments": 2,
			"failed_deployments":    1,
			"pending_deployments":   0,
		},
		"last_updated": time.Now(),
	}

	// Cache the result
	if s.cache != nil {
		s.cache.Set(cacheKey, result)
	}

	s.writeJSONResponse(w, http.StatusOK, result)
}

// getClusterDeployment handles GET /api/v1/clusters/{id}/deployments/{deployment}
func (s *NephoranAPIServer) getClusterDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	deploymentID := vars["deployment"]

	// Check cache first
	cacheKey := fmt.Sprintf("cluster-deployment:%s:%s", clusterID, deploymentID)
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

	// Mock deployment details - would integrate with actual cluster manager
	deploymentDetails := map[string]interface{}{
		"cluster_id":     clusterID,
		"deployment_id":  deploymentID,
		"package_name":   "nginx-ingress-controller",
		"version":        "v1.8.1",
		"status":         "Deployed",
		"deployed_at":    time.Now().Add(-2 * time.Hour),
		"deployed_by":    "system",
		"duration":       "45s",
		"namespace":      "ingress-nginx",
		"replicas":       3,
		"ready_replicas": 3,
		"resources": map[string]interface{}{
			"requests": map[string]string{
				"cpu":    "100m",
				"memory": "128Mi",
			},
			"limits": map[string]string{
				"cpu":    "1000m",
				"memory": "512Mi",
			},
		},
		"services": []map[string]interface{}{
			{
				"name": "ingress-nginx-controller",
				"type": "LoadBalancer",
				"external_ip": "203.0.113.10",
				"ports": []map[string]interface{}{
					{"name": "http", "port": 80, "target_port": "http"},
					{"name": "https", "port": 443, "target_port": "https"},
				},
			},
		},
		"config_maps": []string{
			"ingress-nginx-controller",
			"ingress-nginx-tcp",
		},
		"secrets": []string{
			"ingress-nginx-admission",
		},
		"events": []map[string]interface{}{
			{
				"type":      "Normal",
				"reason":    "Scheduled",
				"message":   "Successfully assigned ingress-nginx/ingress-nginx-controller-xxxx to node-1",
				"timestamp": time.Now().Add(-2 * time.Hour),
			},
			{
				"type":      "Normal", 
				"reason":    "Pulled",
				"message":   "Container image pulled successfully",
				"timestamp": time.Now().Add(-2*time.Hour + 30*time.Second),
			},
		},
		"last_updated": time.Now(),
	}

	// Cache the result
	if s.cache != nil {
		s.cache.Set(cacheKey, deploymentDetails)
	}

	s.writeJSONResponse(w, http.StatusOK, deploymentDetails)
}

// getClusterNamespaces handles GET /api/v1/clusters/{id}/namespaces  
func (s *NephoranAPIServer) getClusterNamespaces(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]
	pagination := s.parsePaginationParams(r)

	// Check cache first
	cacheKey := fmt.Sprintf("cluster-namespaces:%s:%v", clusterID, pagination)
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

	// Mock namespace information - would integrate with actual cluster manager
	allNamespaces := []map[string]interface{}{
		{
			"name":        "default",
			"status":      "Active",
			"created_at":  time.Now().Add(-72 * time.Hour),
			"pod_count":   12,
			"service_count": 3,
			"resource_quota": map[string]string{
				"requests.cpu":    "4",
				"requests.memory": "8Gi", 
				"persistentvolumeclaims": "10",
			},
			"labels": map[string]string{
				"name": "default",
			},
		},
		{
			"name":        "kube-system",
			"status":      "Active", 
			"created_at":  time.Now().Add(-72 * time.Hour),
			"pod_count":   28,
			"service_count": 8,
			"resource_quota": nil,
			"labels": map[string]string{
				"name": "kube-system",
			},
		},
		{
			"name":        "ingress-nginx",
			"status":      "Active",
			"created_at":  time.Now().Add(-24 * time.Hour),
			"pod_count":   3,
			"service_count": 2,
			"resource_quota": map[string]string{
				"requests.cpu":    "2",
				"requests.memory": "4Gi",
			},
			"labels": map[string]string{
				"name": "ingress-nginx",
				"app.kubernetes.io/name": "ingress-nginx",
			},
		},
		{
			"name":        "monitoring",
			"status":      "Active",
			"created_at":  time.Now().Add(-12 * time.Hour),
			"pod_count":   15,
			"service_count": 6,
			"resource_quota": map[string]string{
				"requests.cpu":    "8",
				"requests.memory": "16Gi",
				"requests.storage": "100Gi",
			},
			"labels": map[string]string{
				"name": "monitoring",
				"app.kubernetes.io/name": "prometheus-operator",
			},
		},
	}

	// Apply pagination
	totalItems := len(allNamespaces)
	startIndex := (pagination.Page - 1) * pagination.PageSize
	endIndex := startIndex + pagination.PageSize
	if endIndex > totalItems {
		endIndex = totalItems
	}
	if startIndex > totalItems {
		startIndex = totalItems
	}

	paginatedNamespaces := allNamespaces[startIndex:endIndex]

	// Build response with metadata
	meta := &Meta{
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: (totalItems + pagination.PageSize - 1) / pagination.PageSize,
		TotalItems: totalItems,
	}

	result := map[string]interface{}{
		"cluster_id":   clusterID,
		"namespaces":   paginatedNamespaces,
		"meta":         meta,
		"summary": map[string]interface{}{
			"total_namespaces":   len(allNamespaces),
			"active_namespaces":  4,
			"system_namespaces":  2, // kube-system, kube-public, etc.
			"user_namespaces":    2, // ingress-nginx, monitoring
		},
		"last_updated": time.Now(),
	}

	// Cache the result
	if s.cache != nil {
		s.cache.Set(cacheKey, result)
	}

	s.writeJSONResponse(w, http.StatusOK, result)
}

// testClusterConnectivity handles POST /api/v1/clusters/{id}/connectivity
func (s *NephoranAPIServer) testClusterConnectivity(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	// Mock connectivity test - would perform actual connectivity tests
	connectivityResult := map[string]interface{}{
		"cluster_id":     clusterID,
		"test_timestamp": time.Now(),
		"overall_status": "healthy",
		"tests": []map[string]interface{}{
			{
				"test_name":   "api_server_connectivity",
				"status":      "passed",
				"latency_ms":  15.2,
				"message":     "API server is responsive",
			},
			{
				"test_name":   "dns_resolution",
				"status":      "passed", 
				"latency_ms":  8.7,
				"message":     "DNS resolution working correctly",
			},
			{
				"test_name":   "internet_connectivity",
				"status":      "passed",
				"latency_ms":  45.3,
				"message":     "External connectivity available",
			},
			{
				"test_name":   "cluster_internal_networking",
				"status":      "passed",
				"latency_ms":  2.1,
				"message":     "Pod-to-pod communication working",
			},
		},
		"network_policies": map[string]interface{}{
			"enabled":       true,
			"default_deny":  false,
			"policy_count":  5,
		},
		"load_balancer": map[string]interface{}{
			"status":       "active",
			"external_ip":  "203.0.113.10", 
			"health_check": "passed",
		},
	}

	s.writeJSONResponse(w, http.StatusOK, connectivityResult)
}

// getClusterLatency handles GET /api/v1/clusters/{id}/latency
func (s *NephoranAPIServer) getClusterLatency(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clusterID := vars["id"]

	// Check cache first with short TTL for latency data
	cacheKey := fmt.Sprintf("cluster-latency:%s", clusterID)
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

	// Mock latency information - would gather actual latency metrics
	latencyData := map[string]interface{}{
		"cluster_id":    clusterID,
		"measured_at":   time.Now(),
		"api_server": map[string]interface{}{
			"average_ms":  12.5,
			"p95_ms":      25.8,
			"p99_ms":      45.2,
			"status":      "healthy",
		},
		"etcd": map[string]interface{}{
			"average_ms":  3.2,
			"p95_ms":      8.7,
			"p99_ms":      15.1,
			"status":      "healthy",
		},
		"pod_to_pod": map[string]interface{}{
			"average_ms":  1.8,
			"p95_ms":      4.2,
			"p99_ms":      8.9,
			"status":      "excellent",
		},
		"external": map[string]interface{}{
			"average_ms":  45.6,
			"p95_ms":      78.2,
			"p99_ms":      125.4,
			"status":      "good",
		},
		"inter_node": map[string]interface{}{
			"average_ms":  0.8,
			"p95_ms":      1.5,
			"p99_ms":      2.8,
			"status":      "excellent",
		},
		"trends": map[string]interface{}{
			"last_hour":    []float64{12.1, 12.8, 11.9, 13.2, 12.5},
			"last_24h_avg": 13.2,
			"last_7d_avg":  14.1,
			"direction":    "stable",
		},
	}

	// Cache with short TTL for real-time latency data
	if s.cache != nil {
		s.cache.SetWithTTL(cacheKey, latencyData, 20*time.Second)
	}

	s.writeJSONResponse(w, http.StatusOK, latencyData)
}

// bulkDeployToClusters handles POST /api/v1/clusters/bulk/deploy
func (s *NephoranAPIServer) bulkDeployToClusters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check deployment permissions
	if s.authMiddleware != nil && !auth.HasPermission(ctx, auth.PermissionManageSystem) {
		s.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",
			"System management permission required for bulk deployments")
		return
	}

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

	if len(req.TargetClusters) == 0 {
		s.writeErrorResponse(w, http.StatusBadRequest, "validation_failed",
			"target_clusters is required for bulk deployment")
		return
	}

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	// Mock bulk deployment results - would perform actual bulk deployment
	deploymentResults := make([]map[string]interface{}, len(req.TargetClusters))
	for i, clusterName := range req.TargetClusters {
		deploymentResults[i] = map[string]interface{}{
			"cluster":      clusterName,
			"status":       "success",
			"package_name": req.PackageName,
			"deployed_at":  time.Now(),
			"duration":     fmt.Sprintf("%ds", 30+(i*10)), // Simulated staggered deployments
			"message":      "Deployment completed successfully",
		}

		// Broadcast individual cluster deployment events
		s.broadcastClusterUpdate(&ClusterStatusUpdate{
			ClusterName: clusterName,
			Message:     fmt.Sprintf("Bulk deployment of %s completed", req.PackageName),
			Timestamp:   time.Now(),
			EventType:   "deployment",
		})
	}

	result := map[string]interface{}{
		"bulk_deployment_id": fmt.Sprintf("bulk-%d", time.Now().Unix()),
		"package_name":       req.PackageName,
		"target_clusters":    req.TargetClusters,
		"total_clusters":     len(req.TargetClusters),
		"successful":         len(req.TargetClusters), // All successful in mock
		"failed":             0,
		"results":            deploymentResults,
		"started_at":         time.Now(),
		"completed_at":       time.Now().Add(time.Duration(len(req.TargetClusters)*10) * time.Second),
	}

	s.writeJSONResponse(w, http.StatusOK, result)
}

// getBulkClusterHealth handles GET /api/v1/clusters/bulk/health
func (s *NephoranAPIServer) getBulkClusterHealth(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for cluster filtering
	clusterNames := r.URL.Query()["clusters"]

	// Check cache first
	cacheKey := fmt.Sprintf("bulk-cluster-health:%v", clusterNames)
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

	// If no specific clusters requested, get all clusters
	if len(clusterNames) == 0 {
		clusterNames = []string{"cluster-1", "cluster-2", "cluster-3"} // Mock cluster list
	}

	healthResults := make([]map[string]interface{}, len(clusterNames))
	overallHealthy := 0
	overallUnhealthy := 0

	for i, clusterName := range clusterNames {
		// Mock health data for each cluster
		isHealthy := (i%4 != 3) // 75% healthy rate in mock
		status := "healthy"
		healthScore := 95.0
		
		if !isHealthy {
			status = "degraded"
			healthScore = 65.0
			overallUnhealthy++
		} else {
			overallHealthy++
		}

		runningPods := 38
		criticalAlerts := 2
		warningAlerts := 5
		if isHealthy {
			runningPods = 45
			criticalAlerts = 0
			warningAlerts = 1
		}

		healthResults[i] = map[string]interface{}{
			"cluster_name":  clusterName,
			"status":        status,
			"health_score":  healthScore,
			"last_checked":  time.Now(),
			"components": map[string]string{
				"api-server":        status,
				"etcd":              status,
				"kubelet":           "healthy",
				"container-runtime": "healthy",
			},
			"node_count":         3,
			"ready_nodes":        3,
			"pod_count":          45,
			"running_pods":       runningPods,
			"critical_alerts":    criticalAlerts,
			"warning_alerts":     warningAlerts,
		}
	}

	result := map[string]interface{}{
		"summary": map[string]interface{}{
			"total_clusters":     len(clusterNames),
			"healthy_clusters":   overallHealthy,
			"unhealthy_clusters": overallUnhealthy,
			"overall_health_score": float64(overallHealthy) / float64(len(clusterNames)) * 100,
			"last_updated":       time.Now(),
		},
		"clusters": healthResults,
		"alerts": []map[string]interface{}{
			{
				"level":      "warning",
				"cluster":    "cluster-3",
				"component":  "etcd",
				"message":    "High etcd latency detected",
				"timestamp":  time.Now().Add(-10 * time.Minute),
			},
		},
	}

	// Cache with short TTL for health data
	if s.cache != nil {
		s.cache.SetWithTTL(cacheKey, result, 30*time.Second)
	}

	s.writeJSONResponse(w, http.StatusOK, result)
}

// getOptimalPlacement handles POST /api/v1/clusters/placement
func (s *NephoranAPIServer) getOptimalPlacement(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid_json",
			"Invalid JSON in request body")
		return
	}

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	// Mock optimal placement algorithm - would use actual placement logic
	resourceRequirements := req["resource_requirements"]
	constraints := req["constraints"]
	
	placementResult := map[string]interface{}{
		"request_id":           fmt.Sprintf("placement-%d", time.Now().Unix()),
		"resource_requirements": resourceRequirements,
		"constraints":          constraints,
		"recommendations": []map[string]interface{}{
			{
				"rank":         1,
				"cluster_name": "cluster-1",
				"score":        95.2,
				"reasons": []string{
					"Best resource availability",
					"Optimal network latency", 
					"Meets all constraints",
				},
				"estimated_cost": 125.50,
				"sla_compliance": "99.9%",
			},
			{
				"rank":         2,
				"cluster_name": "cluster-2", 
				"score":        88.7,
				"reasons": []string{
					"Good resource availability",
					"Acceptable network latency",
					"Meets most constraints",
				},
				"estimated_cost": 135.25,
				"sla_compliance": "99.5%",
			},
		},
		"placement_strategy": map[string]interface{}{
			"primary_cluster":   "cluster-1",
			"fallback_clusters": []string{"cluster-2"},
			"load_balancing":    "round-robin",
			"auto_failover":     true,
		},
		"analysis": map[string]interface{}{
			"total_clusters_evaluated": 3,
			"suitable_clusters":         2,
			"excluded_clusters":         1,
			"exclusion_reasons": map[string]interface{}{
				"cluster-3": "Insufficient CPU resources",
			},
		},
		"calculated_at": time.Now(),
	}

	s.logger.Info("Calculated optimal placement",
		"request_id", placementResult["request_id"],
		"primary_cluster", "cluster-1")

	s.writeJSONResponse(w, http.StatusOK, placementResult)
}

// discoverClusters handles GET /api/v1/clusters/discover
func (s *NephoranAPIServer) discoverClusters(w http.ResponseWriter, r *http.Request) {
	// Parse discovery parameters
	provider := r.URL.Query().Get("provider")      // aws, gcp, azure
	region := r.URL.Query().Get("region")
	autoRegister := r.URL.Query().Get("auto_register") == "true"

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	// Mock cluster discovery - would integrate with cloud provider APIs
	discoveredClusters := []map[string]interface{}{
		{
			"name":                "production-eks-cluster",
			"provider":            "aws",
			"region":              "us-west-2",
			"zone":                "us-west-2a",
			"kubernetes_version":  "1.28",
			"node_count":          5,
			"status":              "running",
			"endpoint":            "https://A1B2C3D4E5F6G7H8I9J0.gr7.us-west-2.eks.amazonaws.com",
			"created_at":          time.Now().Add(-168 * time.Hour), // 1 week ago
			"tags": map[string]string{
				"Environment":  "production",
				"Owner":        "platform-team",
				"Application":  "microservices",
			},
			"estimated_cost_monthly": 1250.0,
			"already_registered":     false,
		},
		{
			"name":                "staging-gke-cluster",
			"provider":            "gcp",
			"region":              "us-central1",
			"zone":                "us-central1-a",
			"kubernetes_version":  "1.27",
			"node_count":          3,
			"status":              "running",
			"endpoint":            "https://35.123.456.789",
			"created_at":          time.Now().Add(-72 * time.Hour), // 3 days ago
			"tags": map[string]string{
				"Environment":  "staging",
				"Owner":        "dev-team",
				"Application":  "testing",
			},
			"estimated_cost_monthly": 450.0,
			"already_registered":     true,
		},
	}

	// Filter by provider if specified
	if provider != "" {
		filtered := make([]map[string]interface{}, 0)
		for _, cluster := range discoveredClusters {
			if cluster["provider"] == provider {
				filtered = append(filtered, cluster)
			}
		}
		discoveredClusters = filtered
	}

	// Filter by region if specified
	if region != "" {
		filtered := make([]map[string]interface{}, 0)
		for _, cluster := range discoveredClusters {
			if cluster["region"] == region {
				filtered = append(filtered, cluster)
			}
		}
		discoveredClusters = filtered
	}

	discoveryResult := map[string]interface{}{
		"discovery_id":    fmt.Sprintf("discovery-%d", time.Now().Unix()),
		"filters": map[string]interface{}{
			"provider":      provider,
			"region":        region,
			"auto_register": autoRegister,
		},
		"summary": map[string]interface{}{
			"total_discovered":     len(discoveredClusters),
			"already_registered":   1,
			"available_for_registration": 1,
			"estimated_monthly_cost": 1700.0,
		},
		"clusters":         discoveredClusters,
		"discovered_at":    time.Now(),
		"next_discovery":   time.Now().Add(1 * time.Hour),
	}

	s.writeJSONResponse(w, http.StatusOK, discoveryResult)
}

// autoRegisterClusters handles POST /api/v1/clusters/auto-register
func (s *NephoranAPIServer) autoRegisterClusters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check admin permissions for auto-registration
	if s.authMiddleware != nil && !auth.HasPermission(ctx, auth.PermissionManageSystem) {
		s.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",
			"System management permission required for auto-registration")
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid_json",
			"Invalid JSON in request body")
		return
	}

	if s.clusterManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "cluster_manager_unavailable",
			"Multi-cluster management service is not available")
		return
	}

	// Extract registration criteria
	providers := req["providers"]
	regions := req["regions"]
	tags := req["tags"]
	dryRun := req["dry_run"] == true

	// Mock auto-registration results - would perform actual discovery and registration
	registrationResults := []map[string]interface{}{
		{
			"cluster_name":       "production-eks-cluster",
			"provider":           "aws",
			"region":             "us-west-2",
			"registration_status": "success",
			"registered_at":      time.Now(),
			"cluster_id":         "auto-reg-cluster-1",
			"message":            "Successfully auto-registered cluster",
		},
		{
			"cluster_name":       "dev-aks-cluster",
			"provider":           "azure",
			"region":             "eastus",
			"registration_status": "failed",
			"error":              "Unable to connect to cluster API server",
			"message":            "Auto-registration failed due to connectivity issues",
		},
	}

	result := map[string]interface{}{
		"auto_registration_id": fmt.Sprintf("auto-reg-%d", time.Now().Unix()),
		"criteria": map[string]interface{}{
			"providers": providers,
			"regions":   regions,
			"tags":      tags,
			"dry_run":   dryRun,
		},
		"summary": map[string]interface{}{
			"clusters_discovered":      2,
			"registration_attempts":    2,
			"successful_registrations": 1,
			"failed_registrations":     1,
		},
		"results":        registrationResults,
		"processed_at":   time.Now(),
		"next_auto_run":  time.Now().Add(24 * time.Hour),
	}

	if !dryRun {
		// Broadcast registration events for successful registrations
		for _, result := range registrationResults {
			if result["registration_status"] == "success" {
				s.broadcastClusterUpdate(&ClusterStatusUpdate{
					ClusterName: result["cluster_name"].(string),
					Message:     "Cluster auto-registered successfully",
					Timestamp:   time.Now(),
					EventType:   "registered",
				})
			}
		}
	}

	s.logger.Info("Auto-registration completed",
		"auto_registration_id", result["auto_registration_id"],
		"successful", 1,
		"failed", 1,
		"dry_run", dryRun)

	s.writeJSONResponse(w, http.StatusOK, result)
}
