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
	"time"

	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
	"github.com/thc1006/nephoran-intent-operator/pkg/packagerevision"
)

// PackageRequest represents a request to create or modify a package
type PackageRequest struct {
	PackageName string            `json:"package_name"`
	Repository  string            `json:"repository,omitempty"`
	Revision    string            `json:"revision,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// PackageResponse represents a comprehensive package response
type PackageResponse struct {
	*porch.PackageRevision
	LifecycleStatus   *packagerevision.LifecycleStatus    `json:"lifecycle_status,omitempty"`
	ValidationResults []*packagerevision.ValidationResult `json:"validation_results,omitempty"`
	ApprovalStatus    *packagerevision.ApprovalResult     `json:"approval_status,omitempty"`
	Metrics           *packagerevision.PackageMetrics     `json:"metrics,omitempty"`
	DeploymentTargets []DeploymentTargetInfo              `json:"deployment_targets,omitempty"`
}

// DeploymentTargetInfo represents deployment target information
type DeploymentTargetInfo struct {
	ClusterName  string       `json:"cluster_name"`
	Status       string       `json:"status"`
	LastDeployed *metav1.Time `json:"last_deployed,omitempty"`
	Health       string       `json:"health,omitempty"`
	Version      string       `json:"version,omitempty"`
	ErrorMessage string       `json:"error_message,omitempty"`
}

// TransitionRequest represents a lifecycle transition request
type TransitionRequest struct {
	TargetStage         string                            `json:"target_stage"`
	SkipValidation      bool                              `json:"skip_validation,omitempty"`
	SkipApproval        bool                              `json:"skip_approval,omitempty"`
	CreateRollbackPoint bool                              `json:"create_rollback_point,omitempty"`
	RollbackDescription string                            `json:"rollback_description,omitempty"`
	ForceTransition     bool                              `json:"force_transition,omitempty"`
	ValidationPolicy    *packagerevision.ValidationPolicy `json:"validation_policy,omitempty"`
	ApprovalPolicy      *packagerevision.ApprovalPolicy   `json:"approval_policy,omitempty"`
	NotificationTargets []string                          `json:"notification_targets,omitempty"`
	Metadata            map[string]string                 `json:"metadata,omitempty"`
	DryRun              bool                              `json:"dry_run,omitempty"`
}

// PackageStatusUpdate represents a package status update for streaming
type PackageStatusUpdate struct {
	PackageName   string                         `json:"package_name"`
	Repository    string                         `json:"repository"`
	Revision      string                         `json:"revision"`
	CurrentStage  porch.PackageRevisionLifecycle `json:"current_stage"`
	PreviousStage porch.PackageRevisionLifecycle `json:"previous_stage,omitempty"`
	Progress      int                            `json:"progress"` // 0-100
	Message       string                         `json:"message,omitempty"`
	Timestamp     time.Time                      `json:"timestamp"`
	EventType     string                         `json:"event_type"` // created, transition, validation, approval, error
	Conditions    []metav1.Condition             `json:"conditions,omitempty"`
}

// setupPackageRoutes sets up package management API routes
func (s *NephoranAPIServer) setupPackageRoutes(router *mux.Router) {
	packages := router.PathPrefix("/packages").Subrouter()

	// Apply package-specific middleware
	if s.authMiddleware != nil {
		// Most package operations require at least read permissions
		packages.Use(s.authMiddleware.RequirePermissionMiddleware(auth.PermissionReadIntent))
	}

	// Package CRUD operations
	packages.HandleFunc("", s.listPackages).Methods("GET")
	packages.HandleFunc("", s.createPackage).Methods("POST")
	packages.HandleFunc("/{name}", s.getPackage).Methods("GET")
	packages.HandleFunc("/{name}", s.updatePackage).Methods("PUT")
	packages.HandleFunc("/{name}", s.deletePackage).Methods("DELETE")

	// Package revisions
	packages.HandleFunc("/{name}/revisions", s.listPackageRevisions).Methods("GET")
	packages.HandleFunc("/{name}/revisions/{revision}", s.getPackageRevision).Methods("GET")

	// Package lifecycle operations
	packages.HandleFunc("/{name}/propose", s.proposePackage).Methods("POST")
	packages.HandleFunc("/{name}/approve", s.approvePackage).Methods("POST")
	packages.HandleFunc("/{name}/publish", s.publishPackage).Methods("POST")
	packages.HandleFunc("/{name}/reject", s.rejectPackage).Methods("POST")
	packages.HandleFunc("/{name}/rollback", s.rollbackPackage).Methods("POST")

	// Package validation and testing
	packages.HandleFunc("/{name}/validate", s.validatePackage).Methods("POST")
	packages.HandleFunc("/{name}/test", s.testPackage).Methods("POST")
	packages.HandleFunc("/{name}/lint", s.lintPackage).Methods("POST")

	// Package resources and content
	packages.HandleFunc("/{name}/resources", s.getPackageResources).Methods("GET")
	packages.HandleFunc("/{name}/resources", s.updatePackageResources).Methods("PUT")
	packages.HandleFunc("/{name}/diff", s.getPackageDiff).Methods("GET")
	packages.HandleFunc("/{name}/history", s.getPackageHistory).Methods("GET")

	// Package deployment and propagation
	packages.HandleFunc("/{name}/deploy", s.deployPackage).Methods("POST")
	packages.HandleFunc("/{name}/deployment-status", s.getPackageDeploymentStatus).Methods("GET")
	packages.HandleFunc("/{name}/target-clusters", s.getPackageTargetClusters).Methods("GET")

	// Package templates and blueprints
	packages.HandleFunc("/templates", s.listPackageTemplates).Methods("GET")
	packages.HandleFunc("/templates/{template}", s.getPackageTemplate).Methods("GET")
	packages.HandleFunc("/blueprints", s.listBlueprints).Methods("GET")
	packages.HandleFunc("/blueprints/{blueprint}", s.getBlueprint).Methods("GET")

	// Bulk operations
	packages.HandleFunc("/bulk/transition", s.bulkTransitionPackages).Methods("POST")
	packages.HandleFunc("/bulk/validate", s.bulkValidatePackages).Methods("POST")
	packages.HandleFunc("/bulk/deploy", s.bulkDeployPackages).Methods("POST")
}

// listPackages handles GET /api/v1/packages
func (s *NephoranAPIServer) listPackages(w http.ResponseWriter, r *http.Request) {
	pagination := s.parsePaginationParams(r)
	filters := s.parseFilterParams(r)

	repository := r.URL.Query().Get("repository")
	if repository == "" {
		repository = "default"
	}

	// Check cache first
	cacheKey := fmt.Sprintf("packages:%s:%v:%v", repository, pagination, filters)
	if s.cache != nil {
		if cached, found := s.cache.Get(cacheKey); found {
			s.metrics.CacheHits.Inc()
			s.writeJSONResponse(w, http.StatusOK, cached)
			return
		}
		s.metrics.CacheMisses.Inc()
	}

	// Get packages through package manager if available
	if s.packageManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "package_manager_unavailable",
			"Package management service is not available")
		return
	}

	// For now, return mock data structure until packageManager interface is fully implemented
	packages := make([]PackageResponse, 0)

	// Apply filtering and pagination
	totalItems := len(packages)
	startIndex := (pagination.Page - 1) * pagination.PageSize
	endIndex := startIndex + pagination.PageSize
	if endIndex > totalItems {
		endIndex = totalItems
	}
	if startIndex > totalItems {
		startIndex = totalItems
	}

	paginatedItems := packages[startIndex:endIndex]

	// Build response with metadata
	meta := &Meta{
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: (totalItems + pagination.PageSize - 1) / pagination.PageSize,
		TotalItems: totalItems,
	}

	// Build HATEOAS links
	baseURL := fmt.Sprintf("/api/v1/packages?repository=%s&page_size=%d", repository, pagination.PageSize)
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

// getPackage handles GET /api/v1/packages/{name}
func (s *NephoranAPIServer) getPackage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	name := vars["name"]
	repository := r.URL.Query().Get("repository")
	if repository == "" {
		repository = "default"
	}
	revision := r.URL.Query().Get("revision")
	if revision == "" {
		revision = "latest"
	}

	// Check cache first
	cacheKey := fmt.Sprintf("package:%s:%s:%s", repository, name, revision)
	if s.cache != nil {
		if cached, found := s.cache.Get(cacheKey); found {
			s.metrics.CacheHits.Inc()
			s.writeJSONResponse(w, http.StatusOK, cached)
			return
		}
		s.metrics.CacheMisses.Inc()
	}

	if s.packageManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "package_manager_unavailable",
			"Package management service is not available")
		return
	}

	// Create package reference
	packageRef := &porch.PackageReference{
		Repository:  repository,
		PackageName: name,
		Revision:    revision,
	}

	// Get lifecycle status
	lifecycleStatus, err := s.packageManager.GetLifecycleStatus(ctx, packageRef)
	if err != nil {
		s.logger.Error(err, "Failed to get lifecycle status", "package", name, "repository", repository)
		s.writeErrorResponse(w, http.StatusInternalServerError, "get_lifecycle_failed",
			"Failed to retrieve package lifecycle status")
		return
	}

	// Get package metrics
	metrics, err := s.packageManager.GetPackageMetrics(ctx, packageRef)
	if err != nil {
		s.logger.Error(err, "Failed to get package metrics", "package", name, "repository", repository)
		// Don't fail the request for metrics - just log and continue
		metrics = nil
	}

	// Build comprehensive response
	response := PackageResponse{
		LifecycleStatus: lifecycleStatus,
		Metrics:         metrics,
		// PackageRevision will be populated from actual porch data
		DeploymentTargets: s.getDeploymentTargetInfo(ctx, packageRef),
	}

	// Cache the result
	if s.cache != nil {
		s.cache.Set(cacheKey, response)
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// proposePackage handles POST /api/v1/packages/{name}/propose
func (s *NephoranAPIServer) proposePackage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check permissions for package transitions
	if s.authMiddleware != nil && !auth.HasPermission(ctx, auth.PermissionUpdateIntent) {
		s.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",
			"Update intent permission required for package transitions")
		return
	}

	vars := mux.Vars(r)
	name := vars["name"]
	repository := r.URL.Query().Get("repository")
	if repository == "" {
		repository = "default"
	}
	revision := r.URL.Query().Get("revision")
	if revision == "" {
		revision = "latest"
	}

	var req TransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If no body provided, use defaults
		req = TransitionRequest{
			TargetStage: string(porch.PackageRevisionLifecycleProposed),
		}
	}

	if s.packageManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "package_manager_unavailable",
			"Package management service is not available")
		return
	}

	// Create package reference
	packageRef := &porch.PackageReference{
		Repository:  repository,
		PackageName: name,
		Revision:    revision,
	}

	// Create transition options
	opts := &packagerevision.TransitionOptions{
		SkipValidation:      req.SkipValidation,
		SkipApproval:        req.SkipApproval,
		CreateRollbackPoint: req.CreateRollbackPoint,
		RollbackDescription: req.RollbackDescription,
		ForceTransition:     req.ForceTransition,
		ValidationPolicy:    req.ValidationPolicy,
		ApprovalPolicy:      req.ApprovalPolicy,
		NotificationTargets: req.NotificationTargets,
		DryRun:              req.DryRun,
	}

	if req.Metadata != nil {
		opts.Metadata = req.Metadata
	}

	// Perform the transition
	result, err := s.packageManager.TransitionToProposed(ctx, packageRef, opts)
	if err != nil {
		s.logger.Error(err, "Failed to propose package", "package", name, "repository", repository)
		s.writeErrorResponse(w, http.StatusInternalServerError, "transition_failed",
			fmt.Sprintf("Failed to propose package: %v", err))
		return
	}

	// Invalidate cache
	if s.cache != nil {
		s.cache.Invalidate(fmt.Sprintf("package:%s:%s:", repository, name))
		s.cache.Invalidate(fmt.Sprintf("packages:%s:", repository))
	}

	// Broadcast package status update
	s.broadcastPackageUpdate(&PackageStatusUpdate{
		PackageName:   name,
		Repository:    repository,
		Revision:      revision,
		CurrentStage:  result.NewStage,
		PreviousStage: result.PreviousStage,
		Progress:      100,
		Message:       "Package proposed successfully",
		Timestamp:     result.TransitionTime,
		EventType:     "transition",
	})

	s.writeJSONResponse(w, http.StatusOK, result)
}

// approvePackage handles POST /api/v1/packages/{name}/approve
func (s *NephoranAPIServer) approvePackage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check admin permissions for package approval
	if s.authMiddleware != nil && !auth.HasPermission(ctx, auth.PermissionManageSystem) {
		s.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",
			"System management permission required for package approval")
		return
	}

	vars := mux.Vars(r)
	name := vars["name"]
	repository := r.URL.Query().Get("repository")
	if repository == "" {
		repository = "default"
	}
	revision := r.URL.Query().Get("revision")
	if revision == "" {
		revision = "latest"
	}

	var req TransitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If no body provided, use defaults
		req = TransitionRequest{
			TargetStage: string(porch.PackageRevisionLifecyclePublished),
		}
	}

	if s.packageManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "package_manager_unavailable",
			"Package management service is not available")
		return
	}

	// Create package reference
	packageRef := &porch.PackageReference{
		Repository:  repository,
		PackageName: name,
		Revision:    revision,
	}

	// Create transition options
	opts := &packagerevision.TransitionOptions{
		SkipValidation:      req.SkipValidation,
		SkipApproval:        req.SkipApproval,
		CreateRollbackPoint: req.CreateRollbackPoint,
		RollbackDescription: req.RollbackDescription,
		ForceTransition:     req.ForceTransition,
		ValidationPolicy:    req.ValidationPolicy,
		ApprovalPolicy:      req.ApprovalPolicy,
		NotificationTargets: req.NotificationTargets,
		DryRun:              req.DryRun,
	}

	if req.Metadata != nil {
		opts.Metadata = req.Metadata
	}

	// Perform the transition to published
	result, err := s.packageManager.TransitionToPublished(ctx, packageRef, opts)
	if err != nil {
		s.logger.Error(err, "Failed to approve package", "package", name, "repository", repository)
		s.writeErrorResponse(w, http.StatusInternalServerError, "transition_failed",
			fmt.Sprintf("Failed to approve package: %v", err))
		return
	}

	// Invalidate cache
	if s.cache != nil {
		s.cache.Invalidate(fmt.Sprintf("package:%s:%s:", repository, name))
		s.cache.Invalidate(fmt.Sprintf("packages:%s:", repository))
	}

	// Broadcast package status update
	s.broadcastPackageUpdate(&PackageStatusUpdate{
		PackageName:   name,
		Repository:    repository,
		Revision:      revision,
		CurrentStage:  result.NewStage,
		PreviousStage: result.PreviousStage,
		Progress:      100,
		Message:       "Package approved and published successfully",
		Timestamp:     result.TransitionTime,
		EventType:     "transition",
	})

	s.writeJSONResponse(w, http.StatusOK, result)
}

// validatePackage handles POST /api/v1/packages/{name}/validate
func (s *NephoranAPIServer) validatePackage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	name := vars["name"]
	repository := r.URL.Query().Get("repository")
	if repository == "" {
		repository = "default"
	}
	revision := r.URL.Query().Get("revision")
	if revision == "" {
		revision = "latest"
	}

	if s.packageManager == nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "package_manager_unavailable",
			"Package management service is not available")
		return
	}

	// Create package reference
	packageRef := &porch.PackageReference{
		Repository:  repository,
		PackageName: name,
		Revision:    revision,
	}

	// Perform validation
	result, err := s.packageManager.ValidateConfiguration(ctx, packageRef)
	if err != nil {
		s.logger.Error(err, "Failed to validate package", "package", name, "repository", repository)
		s.writeErrorResponse(w, http.StatusInternalServerError, "validation_failed",
			fmt.Sprintf("Failed to validate package: %v", err))
		return
	}

	// Broadcast validation update
	eventType := "validation"
	message := "Package validation completed successfully"
	if !result.Valid {
		eventType = "error"
		message = "Package validation failed"
	}

	s.broadcastPackageUpdate(&PackageStatusUpdate{
		PackageName: name,
		Repository:  repository,
		Revision:    revision,
		Message:     message,
		Timestamp:   time.Now(),
		EventType:   eventType,
	})

	s.writeJSONResponse(w, http.StatusOK, result)
}

// getPackageDeploymentStatus handles GET /api/v1/packages/{name}/deployment-status
func (s *NephoranAPIServer) getPackageDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	name := vars["name"]
	repository := r.URL.Query().Get("repository")
	if repository == "" {
		repository = "default"
	}

	// Check cache first
	cacheKey := fmt.Sprintf("package-deployment:%s:%s", repository, name)
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

	// Get multi-cluster status
	status, err := s.clusterManager.GetMultiClusterStatus(ctx, name)
	if err != nil {
		s.logger.Error(err, "Failed to get package deployment status", "package", name, "repository", repository)
		s.writeErrorResponse(w, http.StatusInternalServerError, "get_status_failed",
			"Failed to retrieve package deployment status")
		return
	}

	// Convert to API response format
	deploymentStatus := map[string]interface{}{
		"package_name":   status.PackageName,
		"overall_status": status.OverallStatus,
		"last_updated":   status.LastUpdated,
		"cluster_status": status.ClusterStatuses,
	}

	// Cache the result
	if s.cache != nil {
		s.cache.SetWithTTL(cacheKey, deploymentStatus, 1*time.Minute) // Shorter cache for deployment status
	}

	s.writeJSONResponse(w, http.StatusOK, deploymentStatus)
}

// Helper functions

func (s *NephoranAPIServer) getDeploymentTargetInfo(ctx context.Context, packageRef *porch.PackageReference) []DeploymentTargetInfo {
	if s.clusterManager == nil {
		return []DeploymentTargetInfo{}
	}

	// Get multi-cluster status
	status, err := s.clusterManager.GetMultiClusterStatus(ctx, packageRef.PackageName)
	if err != nil {
		s.logger.Error(err, "Failed to get deployment target info", "package", packageRef.PackageName)
		return []DeploymentTargetInfo{}
	}

	targets := make([]DeploymentTargetInfo, 0, len(status.ClusterStatuses))
	for clusterName, clusterStatus := range status.ClusterStatuses {
		targets = append(targets, DeploymentTargetInfo{
			ClusterName:  clusterName,
			Status:       clusterStatus.Status,
			LastDeployed: &clusterStatus.LastUpdated,
			Health:       "unknown", // Would be populated from actual health checks
			Version:      "v1",      // Would be populated from actual version info
			ErrorMessage: clusterStatus.ErrorMessage,
		})
	}

	return targets
}

func (s *NephoranAPIServer) broadcastPackageUpdate(update *PackageStatusUpdate) {
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

// Additional package operation handlers would be implemented here:
// publishPackage, rejectPackage, rollbackPackage, testPackage, lintPackage,
// getPackageResources, updatePackageResources, getPackageDiff, getPackageHistory,
// deployPackage, getPackageTargetClusters, listPackageTemplates, etc.
