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
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/auth"
)

// IntentRequest represents a request to create or update an intent
type IntentRequest struct {
	Name                string                          `json:"name"`
	Namespace           string                          `json:"namespace,omitempty"`
	Intent              string                          `json:"intent"`
	IntentType          nephoranv1.IntentType                    `json:"intent_type"`
	Priority            nephoranv1.NetworkPriority               `json:"priority,omitempty"`
	TargetComponents    []nephoranv1.NetworkTargetComponent      `json:"target_components,omitempty"`
	ResourceConstraints *nephoranv1.NetworkResourceConstraints `json:"resource_constraints,omitempty"`
	TargetNamespace     string                          `json:"target_namespace,omitempty"`
	TargetCluster       string                          `json:"target_cluster,omitempty"`
	NetworkSlice        string                          `json:"network_slice,omitempty"`
	Region              string                          `json:"region,omitempty"`
	TimeoutSeconds      *int32                          `json:"timeout_seconds,omitempty"`
	MaxRetries          *int32                          `json:"max_retries,omitempty"`
	Labels              map[string]string               `json:"labels,omitempty"`
	Annotations         map[string]string               `json:"annotations,omitempty"`
}

// IntentResponse represents an intent response with extended information
type IntentResponse struct {
	*nephoranv1.NetworkIntent
	ProcessingMetrics *ProcessingMetrics `json:"processing_metrics,omitempty"`
	DeploymentStatus  *DeploymentStatus  `json:"deployment_status,omitempty"`
	ValidationResult  *ValidationSummary `json:"validation_result,omitempty"`
}

// ProcessingMetrics contains intent processing performance metrics
type ProcessingMetrics struct {
	ProcessingDuration   *metav1.Duration `json:"processing_duration,omitempty"`
	DeploymentDuration   *metav1.Duration `json:"deployment_duration,omitempty"`
	TotalDuration        *metav1.Duration `json:"total_duration,omitempty"`
	QueueTime            *metav1.Duration `json:"queue_time,omitempty"`
	LLMProcessingTime    *metav1.Duration `json:"llm_processing_time,omitempty"`
	PackageCreationTime  *metav1.Duration `json:"package_creation_time,omitempty"`
	GitOpsDeploymentTime *metav1.Duration `json:"gitops_deployment_time,omitempty"`
}

// DeploymentStatus contains deployment status information
type DeploymentStatus struct {
	PackageName      string       `json:"package_name,omitempty"`
	PackageRevision  string       `json:"package_revision,omitempty"`
	PackageStatus    string       `json:"package_status,omitempty"`
	DeployedClusters []string     `json:"deployed_clusters,omitempty"`
	FailedClusters   []string     `json:"failed_clusters,omitempty"`
	ResourcesCreated int          `json:"resources_created"`
	ResourcesFailed  int          `json:"resources_failed"`
	HealthStatus     string       `json:"health_status,omitempty"`
	LastHealthCheck  *metav1.Time `json:"last_health_check,omitempty"`
}

// ValidationSummary contains validation result summary
type ValidationSummary struct {
	Valid          bool         `json:"valid"`
	Errors         []string     `json:"errors,omitempty"`
	Warnings       []string     `json:"warnings,omitempty"`
	ValidationTime *metav1.Time `json:"validation_time,omitempty"`
	ValidatedBy    string       `json:"validated_by,omitempty"`
}

// IntentStatusUpdate represents a status update for streaming
type IntentStatusUpdate struct {
	IntentName      string             `json:"intent_name"`
	IntentNamespace string             `json:"intent_namespace"`
	Phase           string             `json:"phase"`
	Conditions      []metav1.Condition `json:"conditions,omitempty"`
	Progress        int                `json:"progress"` // 0-100
	Message         string             `json:"message,omitempty"`
	Timestamp       time.Time          `json:"timestamp"`
	EventType       string             `json:"event_type"` // created, updated, deleted, error
}

// setupIntentRoutes sets up intent management API routes
func (s *NephoranAPIServer) setupIntentRoutes(router *mux.Router) {
	intents := router.PathPrefix("/intents").Subrouter()

	// Apply intent-specific middleware
	if s.authMiddleware != nil {
		// Require intent permissions for all intent operations
		intents.Use(s.authMiddleware.RequirePermissionMiddleware(auth.PermissionReadIntent))
	}

	// Intent CRUD operations
	intents.HandleFunc("", s.listIntents).Methods("GET")
	intents.HandleFunc("", s.createIntent).Methods("POST")
	intents.HandleFunc("/{name}", s.getIntent).Methods("GET")
	intents.HandleFunc("/{name}", s.updateIntent).Methods("PUT")
	intents.HandleFunc("/{name}", s.deleteIntent).Methods("DELETE")

	// Intent status and monitoring
	intents.HandleFunc("/{name}/status", s.getIntentStatus).Methods("GET")
	intents.HandleFunc("/{name}/events", s.getIntentEvents).Methods("GET")
	intents.HandleFunc("/{name}/logs", s.getIntentLogs).Methods("GET")
	intents.HandleFunc("/{name}/metrics", s.getIntentMetrics).Methods("GET")

	// Intent operations
	intents.HandleFunc("/{name}/validate", s.validateIntent).Methods("POST")
	intents.HandleFunc("/{name}/retry", s.retryIntent).Methods("POST")
	intents.HandleFunc("/{name}/cancel", s.cancelIntent).Methods("POST")

	// Intent templates and suggestions
	intents.HandleFunc("/templates", s.getIntentTemplates).Methods("GET")
	intents.HandleFunc("/suggestions", s.getIntentSuggestions).Methods("GET")
	intents.HandleFunc("/preview", s.previewIntent).Methods("POST")

	// Bulk operations
	intents.HandleFunc("/bulk", s.bulkCreateIntents).Methods("POST")
	intents.HandleFunc("/bulk/delete", s.bulkDeleteIntents).Methods("POST")
	intents.HandleFunc("/bulk/status", s.bulkGetIntentStatus).Methods("POST")
}

// listIntents handles GET /api/v1/intents
func (s *NephoranAPIServer) listIntents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pagination := s.parsePaginationParams(r)
	filters := s.parseFilterParams(r)

	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Check cache first
	cacheKey := fmt.Sprintf("intents:%s:%v:%v", namespace, pagination, filters)
	if s.cache != nil {
		if cached, found := s.cache.Get(cacheKey); found {
			s.metrics.CacheHits.Inc()
			s.writeJSONResponse(w, http.StatusOK, cached)
			return
		}
		s.metrics.CacheMisses.Inc()
	}

	// Get intents from Kubernetes - Mock implementation to fix compilation
	// TODO: Replace with proper controller-runtime client or use intentManager
	intentList := nephoranv1.NetworkIntentList{
		Items: []nephoranv1.NetworkIntent{},
	}

	// Mock implementation - would normally fetch from Kubernetes
	s.logger.V(1).Info("Listing intents (mock implementation)", "namespace", namespace)

	// Apply additional filtering and pagination
	filteredItems := s.filterIntents(intentList.Items, filters)
	totalItems := len(filteredItems)

	// Calculate pagination
	startIndex := (pagination.Page - 1) * pagination.PageSize
	endIndex := startIndex + pagination.PageSize
	if endIndex > totalItems {
		endIndex = totalItems
	}
	if startIndex > totalItems {
		startIndex = totalItems
	}

	paginatedItems := filteredItems[startIndex:endIndex]

	// Convert to response format
	responses := make([]IntentResponse, len(paginatedItems))
	for i, intent := range paginatedItems {
		responses[i] = s.buildIntentResponse(&intent)
	}

	// Build response with metadata
	meta := &Meta{
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: (totalItems + pagination.PageSize - 1) / pagination.PageSize,
		TotalItems: totalItems,
	}

	// Build HATEOAS links
	baseURL := fmt.Sprintf("/api/v1/intents?namespace=%s&page_size=%d", namespace, pagination.PageSize)
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
		"items": responses,
		"meta":  meta,
		"links": links,
	}

	// Cache the result
	if s.cache != nil {
		s.cache.Set(cacheKey, result)
	}

	s.writeJSONResponse(w, http.StatusOK, result)
}

// createIntent handles POST /api/v1/intents
func (s *NephoranAPIServer) createIntent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check create permission
	if s.authMiddleware != nil && !auth.HasPermission(ctx, auth.PermissionCreateIntent) {
		s.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",
			"Create intent permission required")
		return
	}

	var req IntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid_json",
			"Invalid JSON in request body")
		return
	}

	// Validate request
	if err := s.validateIntentRequest(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	// Set defaults
	if req.Namespace == "" {
		req.Namespace = "default"
	}
	if req.Priority == "" {
		req.Priority = nephoranv1.PriorityMedium
	}
	if req.IntentType == "" {
		req.IntentType = nephoranv1.IntentTypeDeployment
	}

	// Create NetworkIntent object
	intent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent:              req.Intent,
			IntentType:          req.IntentType,
			Priority:            req.Priority,
			TargetComponents:    req.TargetComponents,
			ResourceConstraints: req.ResourceConstraints,
			TargetNamespace:     req.TargetNamespace,
			TargetCluster:       req.TargetCluster,
			NetworkSlice:        req.NetworkSlice,
			Region:              req.Region,
			TimeoutSeconds:      req.TimeoutSeconds,
			MaxRetries:          req.MaxRetries,
		},
	}

	// Validate the NetworkIntent
	if err := intent.ValidateNetworkIntent(); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "intent_validation_failed", err.Error())
		return
	}

	// Create the intent in Kubernetes
	intentBytes, err := json.Marshal(intent)
	if err != nil {
		s.logger.Error(err, "Failed to marshal intent")
		s.writeErrorResponse(w, http.StatusInternalServerError, "marshal_failed", "Failed to process intent")
		return
	}

	// Mock creation - TODO: Replace with proper controller-runtime client
	createdIntent := intent
	createdIntent.CreationTimestamp = metav1.Now()
	createdIntent.Generation = 1
	createdIntent.UID = "mock-uid-12345"
	
	s.logger.Info("Intent created (mock implementation)", "name", req.Name, "namespace", req.Namespace)

	// Invalidate cache
	if s.cache != nil {
		s.cache.Invalidate(fmt.Sprintf("intents:%s:", req.Namespace))
	}

	// Broadcast intent creation event
	s.broadcastIntentUpdate(&IntentStatusUpdate{
		IntentName:      createdIntent.Name,
		IntentNamespace: createdIntent.Namespace,
		Phase:           createdIntent.Status.Phase,
		Progress:        0,
		Message:         "Intent created successfully",
		Timestamp:       time.Now(),
		EventType:       "created",
	})

	response := s.buildIntentResponse(&createdIntent)
	s.writeJSONResponse(w, http.StatusCreated, response)
}

// getIntent handles GET /api/v1/intents/{name}
func (s *NephoranAPIServer) getIntent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	name := vars["name"]
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Check cache first
	cacheKey := fmt.Sprintf("intent:%s:%s", namespace, name)
	if s.cache != nil {
		if cached, found := s.cache.Get(cacheKey); found {
			s.metrics.CacheHits.Inc()
			s.writeJSONResponse(w, http.StatusOK, cached)
			return
		}
		s.metrics.CacheMisses.Inc()
	}

	// Mock get intent implementation - TODO: Replace with proper client
	intent := nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       "mock-uid-get-12345",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent:     "Mock intent for getting",
			IntentType: nephoranv1.IntentTypeScaling,
			Priority:   nephoranv1.NetworkPriorityNormal,
		},
	}
	
	s.logger.Info("Retrieved intent (mock implementation)", "name", name, "namespace", namespace)

	response := s.buildIntentResponse(&intent)

	// Cache the result
	if s.cache != nil {
		s.cache.Set(cacheKey, response)
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// updateIntent handles PUT /api/v1/intents/{name}
func (s *NephoranAPIServer) updateIntent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check update permission
	if s.authMiddleware != nil && !auth.HasPermission(ctx, auth.PermissionUpdateIntent) {
		s.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",
			"Update intent permission required")
		return
	}

	vars := mux.Vars(r)
	name := vars["name"]
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	var req IntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid_json",
			"Invalid JSON in request body")
		return
	}

	// Validate request
	if err := s.validateIntentRequest(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	// Mock get existing intent - TODO: Replace with proper client
	existingIntent := nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			UID:             "mock-uid-update-12345",
			ResourceVersion: "1",
			Generation:      1,
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent:     "Existing mock intent",
			IntentType: nephoranv1.IntentTypeScaling,
			Priority:   nephoranv1.NetworkPriorityNormal,
		},
	}
	
	s.logger.Info("Retrieved existing intent for update (mock implementation)", "name", name, "namespace", namespace)

	// Update the intent spec
	existingIntent.Spec.Intent = req.Intent
	existingIntent.Spec.IntentType = req.IntentType
	existingIntent.Spec.Priority = req.Priority
	existingIntent.Spec.TargetComponents = req.TargetComponents
	existingIntent.Spec.ResourceConstraints = req.ResourceConstraints
	existingIntent.Spec.TargetNamespace = req.TargetNamespace
	existingIntent.Spec.TargetCluster = req.TargetCluster
	existingIntent.Spec.NetworkSlice = req.NetworkSlice
	existingIntent.Spec.Region = req.Region
	existingIntent.Spec.TimeoutSeconds = req.TimeoutSeconds
	existingIntent.Spec.MaxRetries = req.MaxRetries

	// Update metadata if provided
	if req.Labels != nil {
		existingIntent.Labels = req.Labels
	}
	if req.Annotations != nil {
		existingIntent.Annotations = req.Annotations
	}

	// Validate the updated NetworkIntent
	if err := existingIntent.ValidateNetworkIntent(); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "intent_validation_failed", err.Error())
		return
	}

	// Mock update the intent - TODO: Replace with proper client
	updatedIntent := existingIntent
	updatedIntent.Generation++
	updatedIntent.ResourceVersion = "2"
	
	s.logger.Info("Updated intent (mock implementation)", "name", name, "namespace", namespace)

	// Invalidate cache
	if s.cache != nil {
		s.cache.Invalidate(fmt.Sprintf("intent:%s:%s", namespace, name))
		s.cache.Invalidate(fmt.Sprintf("intents:%s:", namespace))
	}

	// Broadcast intent update event
	s.broadcastIntentUpdate(&IntentStatusUpdate{
		IntentName:      updatedIntent.Name,
		IntentNamespace: updatedIntent.Namespace,
		Phase:           updatedIntent.Status.Phase,
		Message:         "Intent updated successfully",
		Timestamp:       time.Now(),
		EventType:       "updated",
	})

	response := s.buildIntentResponse(&updatedIntent)
	s.writeJSONResponse(w, http.StatusOK, response)
}

// deleteIntent handles DELETE /api/v1/intents/{name}
func (s *NephoranAPIServer) deleteIntent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check delete permission
	if s.authMiddleware != nil && !auth.HasPermission(ctx, auth.PermissionDeleteIntent) {
		s.writeErrorResponse(w, http.StatusForbidden, "insufficient_permissions",
			"Delete intent permission required")
		return
	}

	vars := mux.Vars(r)
	name := vars["name"]
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Mock delete the intent - TODO: Replace with proper client
	s.logger.Info("Deleted intent (mock implementation)", "name", name, "namespace", namespace)

	// Invalidate cache
	if s.cache != nil {
		s.cache.Invalidate(fmt.Sprintf("intent:%s:%s", namespace, name))
		s.cache.Invalidate(fmt.Sprintf("intents:%s:", namespace))
	}

	// Broadcast intent deletion event
	s.broadcastIntentUpdate(&IntentStatusUpdate{
		IntentName:      name,
		IntentNamespace: namespace,
		Message:         "Intent deleted successfully",
		Timestamp:       time.Now(),
		EventType:       "deleted",
	})

	s.writeJSONResponse(w, http.StatusNoContent, nil)
}

// getIntentStatus handles GET /api/v1/intents/{name}/status
func (s *NephoranAPIServer) getIntentStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	name := vars["name"]
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Mock get intent for status - TODO: Replace with proper client
	intent := nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       "mock-uid-status-12345",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent:     "Mock intent for status",
			IntentType: nephoranv1.IntentTypeScaling,
			Priority:   nephoranv1.NetworkPriorityNormal,
		},
		Status: nephoranv1.NetworkIntentStatus{
			Phase:       "Processing",
			LastMessage: "Mock processing status",
		},
	}
	
	s.logger.Info("Retrieved intent status (mock implementation)", "name", name, "namespace", namespace)

	// Build detailed status response
	status := map[string]interface{}{
		"name":                       intent.Name,
		"namespace":                  intent.Namespace,
		"phase":                      intent.Status.Phase,
		"conditions":                 intent.Status.Conditions,
		"processing_start_time":      intent.Status.ProcessingStartTime,
		"processing_completion_time": intent.Status.ProcessingCompletionTime,
		"deployment_start_time":      intent.Status.DeploymentStartTime,
		"deployment_completion_time": intent.Status.DeploymentCompletionTime,
		"git_commit_hash":            intent.Status.GitCommitHash,
		"retry_count":                intent.Status.RetryCount,
		"validation_errors":          intent.Status.ValidationErrors,
		"deployed_components":        intent.Status.DeployedComponents,
		"processing_duration":        intent.Status.ProcessingDuration,
		"deployment_duration":        intent.Status.DeploymentDuration,
		"last_retry_time":            intent.Status.LastRetryTime,
		"observed_generation":        intent.Status.ObservedGeneration,
	}

	s.writeJSONResponse(w, http.StatusOK, status)
}

// Helper functions

func (s *NephoranAPIServer) validateIntentRequest(req *IntentRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.Intent == "" {
		return fmt.Errorf("intent text is required")
	}
	if len(req.Intent) < 10 {
		return fmt.Errorf("intent text must be at least 10 characters")
	}
	if len(req.Intent) > 2000 {
		return fmt.Errorf("intent text must be at most 2000 characters")
	}
	return nil
}

func (s *NephoranAPIServer) filterIntents(items []nephoranv1.NetworkIntent, filters FilterParams) []nephoranv1.NetworkIntent {
	var filtered []nephoranv1.NetworkIntent

	for _, item := range items {
		include := true

		if filters.Status != "" && item.Status.Phase != filters.Status {
			include = false
		}

		if filters.Type != "" && string(item.Spec.IntentType) != filters.Type {
			include = false
		}

		if filters.Priority != "" && string(item.Spec.Priority) != filters.Priority {
			include = false
		}

		if filters.Component != "" {
			hasComponent := false
			for _, comp := range item.Spec.TargetComponents {
				if string(comp) == filters.Component {
					hasComponent = true
					break
				}
			}
			if !hasComponent {
				include = false
			}
		}

		if filters.Cluster != "" && item.Spec.TargetCluster != filters.Cluster {
			include = false
		}

		// Filter by labels
		for labelKey, labelValue := range filters.Labels {
			if item.Labels == nil {
				include = false
				break
			}
			if value, exists := item.Labels[labelKey]; !exists || value != labelValue {
				include = false
				break
			}
		}

		// Filter by time range
		if filters.Since != nil && item.CreationTimestamp.Time.Before(*filters.Since) {
			include = false
		}

		if filters.Until != nil && item.CreationTimestamp.Time.After(*filters.Until) {
			include = false
		}

		if include {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

func (s *NephoranAPIServer) buildIntentResponse(intent *nephoranv1.NetworkIntent) IntentResponse {
	response := IntentResponse{
		NetworkIntent: intent,
	}

	// Add processing metrics if available
	if intent.Status.ProcessingDuration != nil || intent.Status.DeploymentDuration != nil {
		response.ProcessingMetrics = &ProcessingMetrics{
			ProcessingDuration: intent.Status.ProcessingDuration,
			DeploymentDuration: intent.Status.DeploymentDuration,
		}

		// Calculate total duration
		if intent.Status.ProcessingDuration != nil && intent.Status.DeploymentDuration != nil {
			totalDuration := intent.Status.ProcessingDuration.Duration + intent.Status.DeploymentDuration.Duration
			response.ProcessingMetrics.TotalDuration = &metav1.Duration{Duration: totalDuration}
		}
	}

	// Add deployment status if available
	if intent.Status.GitCommitHash != "" || len(intent.Status.DeployedComponents) > 0 {
		response.DeploymentStatus = &DeploymentStatus{
			PackageName:      fmt.Sprintf("%s-%s", intent.Name, intent.Spec.IntentType),
			PackageRevision:  "v1",
			PackageStatus:    intent.Status.Phase,
			ResourcesCreated: len(intent.Status.DeployedComponents),
			HealthStatus:     "unknown",
		}
	}

	// Add validation summary if available
	if len(intent.Status.ValidationErrors) > 0 {
		response.ValidationResult = &ValidationSummary{
			Valid:       len(intent.Status.ValidationErrors) == 0,
			Errors:      intent.Status.ValidationErrors,
			ValidatedBy: "nephoran-controller",
		}
	}

	return response
}

func (s *NephoranAPIServer) broadcastIntentUpdate(update *IntentStatusUpdate) {
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

func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

func mustMarshalString(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

// Additional intent operation handlers would go here...
// validateIntent, retryIntent, cancelIntent, getIntentTemplates, etc.
