// Package o2 implements O-RAN Infrastructure Management Service (IMS) API handlers.

// Following O-RAN.WG6.O2ims-Interface-v01.01 specification.

package o2

import (
	"net/http"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// HTTP Error Response following RFC 7807 Problem Details for HTTP APIs.

type ProblemDetail struct {
	Type string `json:"type"`

	Title string `json:"title"`

	Status int `json:"status"`

	Detail string `json:"detail,omitempty"`

	Instance string `json:"instance,omitempty"`

	Extra interface{} `json:"extra,omitempty"`
}

// Service Information Handlers.

// handleGetServiceInfo returns service information.

func (s *O2APIServer) handleGetServiceInfo(w http.ResponseWriter, r *http.Request) {
	serviceInfo := json.RawMessage(`{}`),

		"supported_providers": s.providerRegistry.GetSupportedProviders(),

		"timestamp": time.Now().Format(time.RFC3339),
	}

	s.writeJSONResponse(w, r, StatusOK, serviceInfo)
}

// handleHealthCheck returns health status.

func (s *O2APIServer) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	health := s.healthChecker.GetHealthStatus()

	status := StatusOK

	if health.Status == "DOWN" {
		status = StatusServiceUnavailable
	} else if health.Status == "DEGRADED" {
		status = StatusOK // Still return 200 for degraded but mention in body
	}

	s.writeJSONResponse(w, r, status, health)
}

// handleReadinessCheck returns readiness status.

func (s *O2APIServer) handleReadinessCheck(w http.ResponseWriter, r *http.Request) {
	ready := json.RawMessage(`{}`),
	}

	s.writeJSONResponse(w, r, StatusOK, ready)
}

// Resource Pool Handlers.

// handleGetResourcePools retrieves resource pools with filtering.

func (s *O2APIServer) handleGetResourcePools(w http.ResponseWriter, r *http.Request) {
	filter := s.parseResourcePoolFilter(r)

	pools, err := s.imsService.GetResourcePools(r.Context(), filter)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to retrieve resource pools", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, pools)
}

// handleGetResourcePool retrieves a specific resource pool.

func (s *O2APIServer) handleGetResourcePool(w http.ResponseWriter, r *http.Request) {
	poolID := s.getPathParam(r, "resourcePoolId")

	if poolID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Resource pool ID is required", nil)

		return

	}

	pool, err := s.imsService.GetResourcePool(r.Context(), poolID)
	if err != nil {

		s.writeErrorResponse(w, r, StatusNotFound, "Resource pool not found", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, pool)
}

// handleCreateResourcePool creates a new resource pool.

func (s *O2APIServer) handleCreateResourcePool(w http.ResponseWriter, r *http.Request) {
	var req models.CreateResourcePoolRequest

	if err := s.decodeJSONRequest(r, &req); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	pool, err := s.imsService.CreateResourcePool(r.Context(), &req)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to create resource pool", err)

		return

	}

	s.metrics.RecordResourceOperation("create", "resource_pool", req.Provider, "success")

	s.writeJSONResponse(w, r, StatusCreated, pool)
}

// handleUpdateResourcePool updates an existing resource pool.

func (s *O2APIServer) handleUpdateResourcePool(w http.ResponseWriter, r *http.Request) {
	poolID := s.getPathParam(r, "resourcePoolId")

	if poolID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Resource pool ID is required", nil)

		return

	}

	var req models.UpdateResourcePoolRequest

	if err := s.decodeJSONRequest(r, &req); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	pool, err := s.imsService.UpdateResourcePool(r.Context(), poolID, &req)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to update resource pool", err)

		return

	}

	s.metrics.RecordResourceOperation("update", "resource_pool", "unknown", "success")

	s.writeJSONResponse(w, r, StatusOK, pool)
}

// handleDeleteResourcePool deletes a resource pool.

func (s *O2APIServer) handleDeleteResourcePool(w http.ResponseWriter, r *http.Request) {
	poolID := s.getPathParam(r, "resourcePoolId")

	if poolID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Resource pool ID is required", nil)

		return

	}

	if err := s.imsService.DeleteResourcePool(r.Context(), poolID); err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to delete resource pool", err)

		return

	}

	s.metrics.RecordResourceOperation("delete", "resource_pool", "unknown", "success")

	w.WriteHeader(StatusNoContent)
}

// Resource Type Handlers.

// handleGetResourceTypes retrieves resource types.

func (s *O2APIServer) handleGetResourceTypes(w http.ResponseWriter, r *http.Request) {
	filter := s.parseResourceTypeFilter(r)

	resourceTypes, err := s.imsService.GetResourceTypes(r.Context(), filter)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to retrieve resource types", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, resourceTypes)
}

// handleGetResourceType retrieves a specific resource type.

func (s *O2APIServer) handleGetResourceType(w http.ResponseWriter, r *http.Request) {
	typeID := s.getPathParam(r, "resourceTypeId")

	if typeID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Resource type ID is required", nil)

		return

	}

	resourceType, err := s.imsService.GetResourceType(r.Context(), typeID)
	if err != nil {

		s.writeErrorResponse(w, r, StatusNotFound, "Resource type not found", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, resourceType)
}

// handleCreateResourceType creates a new resource type.

func (s *O2APIServer) handleCreateResourceType(w http.ResponseWriter, r *http.Request) {
	var resourceType models.ResourceType

	if err := s.decodeJSONRequest(r, &resourceType); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	createdType, err := s.imsService.CreateResourceType(r.Context(), &resourceType)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to create resource type", err)

		return

	}

	s.metrics.RecordResourceOperation("create", "resource_type", "system", "success")

	s.writeJSONResponse(w, r, StatusCreated, createdType)
}

// handleUpdateResourceType updates an existing resource type.

func (s *O2APIServer) handleUpdateResourceType(w http.ResponseWriter, r *http.Request) {
	typeID := s.getPathParam(r, "resourceTypeId")

	if typeID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Resource type ID is required", nil)

		return

	}

	var resourceType models.ResourceType

	if err := s.decodeJSONRequest(r, &resourceType); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	updatedType, err := s.imsService.UpdateResourceType(r.Context(), typeID, &resourceType)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to update resource type", err)

		return

	}

	s.metrics.RecordResourceOperation("update", "resource_type", "system", "success")

	s.writeJSONResponse(w, r, StatusOK, updatedType)
}

// handleDeleteResourceType deletes a resource type.

func (s *O2APIServer) handleDeleteResourceType(w http.ResponseWriter, r *http.Request) {
	typeID := s.getPathParam(r, "resourceTypeId")

	if typeID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Resource type ID is required", nil)

		return

	}

	if err := s.imsService.DeleteResourceType(r.Context(), typeID); err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to delete resource type", err)

		return

	}

	s.metrics.RecordResourceOperation("delete", "resource_type", "system", "success")

	w.WriteHeader(StatusNoContent)
}

// Resource Handlers.

// handleGetResources retrieves resources with filtering.

func (s *O2APIServer) handleGetResources(w http.ResponseWriter, r *http.Request) {
	filter := s.parseResourceFilter(r)

	resources, err := s.imsService.GetResources(r.Context(), filter)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to retrieve resources", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, resources)
}

// handleGetResource retrieves a specific resource.

func (s *O2APIServer) handleGetResource(w http.ResponseWriter, r *http.Request) {
	resourceID := s.getPathParam(r, "resourceId")

	if resourceID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Resource ID is required", nil)

		return

	}

	resource, err := s.imsService.GetResource(r.Context(), resourceID)
	if err != nil {

		s.writeErrorResponse(w, r, StatusNotFound, "Resource not found", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, resource)
}

// handleCreateResource creates a new resource.

func (s *O2APIServer) handleCreateResource(w http.ResponseWriter, r *http.Request) {
	var req models.CreateResourceRequest

	if err := s.decodeJSONRequest(r, &req); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	resource, err := s.imsService.CreateResource(r.Context(), &req)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to create resource", err)

		return

	}

	s.metrics.RecordResourceOperation("create", req.ResourceTypeID, "unknown", "success")

	s.writeJSONResponse(w, r, StatusCreated, resource)
}

// handleUpdateResource updates an existing resource.

func (s *O2APIServer) handleUpdateResource(w http.ResponseWriter, r *http.Request) {
	resourceID := s.getPathParam(r, "resourceId")

	if resourceID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Resource ID is required", nil)

		return

	}

	var req models.UpdateResourceRequest

	if err := s.decodeJSONRequest(r, &req); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	resource, err := s.imsService.UpdateResource(r.Context(), resourceID, &req)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to update resource", err)

		return

	}

	s.metrics.RecordResourceOperation("update", "resource", "unknown", "success")

	s.writeJSONResponse(w, r, StatusOK, resource)
}

// handleDeleteResource deletes a resource.

func (s *O2APIServer) handleDeleteResource(w http.ResponseWriter, r *http.Request) {
	resourceID := s.getPathParam(r, "resourceId")

	if resourceID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Resource ID is required", nil)

		return

	}

	if err := s.imsService.DeleteResource(r.Context(), resourceID); err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to delete resource", err)

		return

	}

	s.metrics.RecordResourceOperation("delete", "resource", "unknown", "success")

	w.WriteHeader(StatusNoContent)
}

// Resource Health and Monitoring Handlers.

// handleGetResourceHealth retrieves resource health status.

func (s *O2APIServer) handleGetResourceHealth(w http.ResponseWriter, r *http.Request) {
	resourceID := s.getPathParam(r, "resourceId")

	if resourceID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Resource ID is required", nil)

		return

	}

	health, err := s.imsService.GetResourceHealth(r.Context(), resourceID)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to retrieve resource health", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, health)
}

// handleGetResourceAlarms retrieves resource alarms.

func (s *O2APIServer) handleGetResourceAlarms(w http.ResponseWriter, r *http.Request) {
	resourceID := s.getPathParam(r, "resourceId")

	if resourceID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Resource ID is required", nil)

		return

	}

	filter := s.parseAlarmFilter(r)

	alarms, err := s.imsService.GetResourceAlarms(r.Context(), resourceID, filter)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to retrieve resource alarms", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, alarms)
}

// handleGetResourceMetrics retrieves resource metrics.

func (s *O2APIServer) handleGetResourceMetrics(w http.ResponseWriter, r *http.Request) {
	resourceID := s.getPathParam(r, "resourceId")

	if resourceID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Resource ID is required", nil)

		return

	}

	filter := s.parseMetricsFilter(r)

	metrics, err := s.imsService.GetResourceMetrics(r.Context(), resourceID, filter)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to retrieve resource metrics", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, metrics)
}

// Deployment Template Handlers.

// handleGetDeploymentTemplates retrieves deployment templates.

func (s *O2APIServer) handleGetDeploymentTemplates(w http.ResponseWriter, r *http.Request) {
	filter := s.parseDeploymentTemplateFilter(r)

	templates, err := s.imsService.GetDeploymentTemplates(r.Context(), filter)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to retrieve deployment templates", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, templates)
}

// handleGetDeploymentTemplate retrieves a specific deployment template.

func (s *O2APIServer) handleGetDeploymentTemplate(w http.ResponseWriter, r *http.Request) {
	templateID := s.getPathParam(r, "templateId")

	if templateID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Template ID is required", nil)

		return

	}

	template, err := s.imsService.GetDeploymentTemplate(r.Context(), templateID)
	if err != nil {

		s.writeErrorResponse(w, r, StatusNotFound, "Deployment template not found", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, template)
}

// handleCreateDeploymentTemplate creates a new deployment template.

func (s *O2APIServer) handleCreateDeploymentTemplate(w http.ResponseWriter, r *http.Request) {
	var template DeploymentTemplate

	if err := s.decodeJSONRequest(r, &template); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	createdTemplate, err := s.imsService.CreateDeploymentTemplate(r.Context(), &template)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to create deployment template", err)

		return

	}

	s.metrics.RecordResourceOperation("create", "deployment_template", "system", "success")

	s.writeJSONResponse(w, r, StatusCreated, createdTemplate)
}

// handleUpdateDeploymentTemplate updates an existing deployment template.

func (s *O2APIServer) handleUpdateDeploymentTemplate(w http.ResponseWriter, r *http.Request) {
	templateID := s.getPathParam(r, "templateId")

	if templateID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Template ID is required", nil)

		return

	}

	var template DeploymentTemplate

	if err := s.decodeJSONRequest(r, &template); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	updatedTemplate, err := s.imsService.UpdateDeploymentTemplate(r.Context(), templateID, &template)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to update deployment template", err)

		return

	}

	s.metrics.RecordResourceOperation("update", "deployment_template", "system", "success")

	s.writeJSONResponse(w, r, StatusOK, updatedTemplate)
}

// handleDeleteDeploymentTemplate deletes a deployment template.

func (s *O2APIServer) handleDeleteDeploymentTemplate(w http.ResponseWriter, r *http.Request) {
	templateID := s.getPathParam(r, "templateId")

	if templateID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Template ID is required", nil)

		return

	}

	if err := s.imsService.DeleteDeploymentTemplate(r.Context(), templateID); err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to delete deployment template", err)

		return

	}

	s.metrics.RecordResourceOperation("delete", "deployment_template", "system", "success")

	w.WriteHeader(StatusNoContent)
}

// Deployment Management Handlers.

// handleGetDeployments retrieves deployments.

func (s *O2APIServer) handleGetDeployments(w http.ResponseWriter, r *http.Request) {
	filter := s.parseDeploymentFilter(r)

	deployments, err := s.imsService.GetDeployments(r.Context(), filter)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to retrieve deployments", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, deployments)
}

// handleGetDeployment retrieves a specific deployment.

func (s *O2APIServer) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	deploymentID := s.getPathParam(r, "deploymentId")

	if deploymentID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Deployment ID is required", nil)

		return

	}

	deployment, err := s.imsService.GetDeployment(r.Context(), deploymentID)
	if err != nil {

		s.writeErrorResponse(w, r, StatusNotFound, "Deployment not found", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, deployment)
}

// handleCreateDeployment creates a new deployment.

func (s *O2APIServer) handleCreateDeployment(w http.ResponseWriter, r *http.Request) {
	var req CreateDeploymentRequest

	if err := s.decodeJSONRequest(r, &req); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	deployment, err := s.imsService.CreateDeployment(r.Context(), &req)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to create deployment", err)

		return

	}

	s.metrics.RecordResourceOperation("create", "deployment", "unknown", "success")

	s.writeJSONResponse(w, r, StatusCreated, deployment)
}

// handleUpdateDeployment updates an existing deployment.

func (s *O2APIServer) handleUpdateDeployment(w http.ResponseWriter, r *http.Request) {
	deploymentID := s.getPathParam(r, "deploymentId")

	if deploymentID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Deployment ID is required", nil)

		return

	}

	var req UpdateDeploymentRequest

	if err := s.decodeJSONRequest(r, &req); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	deployment, err := s.imsService.UpdateDeployment(r.Context(), deploymentID, &req)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to update deployment", err)

		return

	}

	s.metrics.RecordResourceOperation("update", "deployment", "unknown", "success")

	s.writeJSONResponse(w, r, StatusOK, deployment)
}

// handleDeleteDeployment deletes a deployment.

func (s *O2APIServer) handleDeleteDeployment(w http.ResponseWriter, r *http.Request) {
	deploymentID := s.getPathParam(r, "deploymentId")

	if deploymentID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Deployment ID is required", nil)

		return

	}

	if err := s.imsService.DeleteDeployment(r.Context(), deploymentID); err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to delete deployment", err)

		return

	}

	s.metrics.RecordResourceOperation("delete", "deployment", "unknown", "success")

	w.WriteHeader(StatusNoContent)
}

// Subscription Handlers.

// handleCreateSubscription creates a new subscription.

func (s *O2APIServer) handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	var subscription Subscription

	if err := s.decodeJSONRequest(r, &subscription); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	createdSubscription, err := s.imsService.CreateSubscription(r.Context(), &subscription)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to create subscription", err)

		return

	}

	s.writeJSONResponse(w, r, StatusCreated, createdSubscription)
}

// handleGetSubscriptions retrieves subscriptions.

func (s *O2APIServer) handleGetSubscriptions(w http.ResponseWriter, r *http.Request) {
	filter := s.parseSubscriptionFilter(r)

	subscriptions, err := s.imsService.GetSubscriptions(r.Context(), filter)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to retrieve subscriptions", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, subscriptions)
}

// handleGetSubscription retrieves a specific subscription.

func (s *O2APIServer) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionID := s.getPathParam(r, "subscriptionId")

	if subscriptionID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Subscription ID is required", nil)

		return

	}

	subscription, err := s.imsService.GetSubscription(r.Context(), subscriptionID)
	if err != nil {

		s.writeErrorResponse(w, r, StatusNotFound, "Subscription not found", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, subscription)
}

// handleUpdateSubscription updates an existing subscription.

func (s *O2APIServer) handleUpdateSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionID := s.getPathParam(r, "subscriptionId")

	if subscriptionID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Subscription ID is required", nil)

		return

	}

	var subscription Subscription

	if err := s.decodeJSONRequest(r, &subscription); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	updatedSubscription, err := s.imsService.UpdateSubscription(r.Context(), subscriptionID, &subscription)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to update subscription", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, updatedSubscription)
}

// handleDeleteSubscription deletes a subscription.

func (s *O2APIServer) handleDeleteSubscription(w http.ResponseWriter, r *http.Request) {
	subscriptionID := s.getPathParam(r, "subscriptionId")

	if subscriptionID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Subscription ID is required", nil)

		return

	}

	if err := s.imsService.DeleteSubscription(r.Context(), subscriptionID); err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to delete subscription", err)

		return

	}

	w.WriteHeader(StatusNoContent)
}

// Cloud Provider Management Handlers.

// handleRegisterCloudProvider registers a new cloud provider.

func (s *O2APIServer) handleRegisterCloudProvider(w http.ResponseWriter, r *http.Request) {
	var provider CloudProviderConfig

	if err := s.decodeJSONRequest(r, &provider); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	_, err := s.imsService.RegisterCloudProvider(r.Context(), &provider)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to register cloud provider", err)

		return

	}

	s.writeJSONResponse(w, r, StatusCreated, provider)
}

// handleGetCloudProviders retrieves cloud providers.

func (s *O2APIServer) handleGetCloudProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := s.imsService.GetCloudProviders(r.Context())
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to retrieve cloud providers", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, providers)
}

// handleGetCloudProvider retrieves a specific cloud provider.

func (s *O2APIServer) handleGetCloudProvider(w http.ResponseWriter, r *http.Request) {
	providerID := s.getPathParam(r, "providerId")

	if providerID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Provider ID is required", nil)

		return

	}

	provider, err := s.imsService.GetCloudProvider(r.Context(), providerID)
	if err != nil {

		s.writeErrorResponse(w, r, StatusNotFound, "Cloud provider not found", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, provider)
}

// handleUpdateCloudProvider updates an existing cloud provider.

func (s *O2APIServer) handleUpdateCloudProvider(w http.ResponseWriter, r *http.Request) {
	providerID := s.getPathParam(r, "providerId")

	if providerID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Provider ID is required", nil)

		return

	}

	var provider CloudProviderConfig

	if err := s.decodeJSONRequest(r, &provider); err != nil {

		s.writeErrorResponse(w, r, StatusBadRequest, "Invalid request body", err)

		return

	}

	_, err := s.imsService.UpdateCloudProvider(r.Context(), providerID, &provider)
	if err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to update cloud provider", err)

		return

	}

	s.writeJSONResponse(w, r, StatusOK, provider)
}

// handleRemoveCloudProvider removes a cloud provider.

func (s *O2APIServer) handleRemoveCloudProvider(w http.ResponseWriter, r *http.Request) {
	providerID := s.getPathParam(r, "providerId")

	if providerID == "" {

		s.writeErrorResponse(w, r, StatusBadRequest, "Provider ID is required", nil)

		return

	}

	if err := s.imsService.RemoveCloudProvider(r.Context(), providerID); err != nil {

		s.writeErrorResponse(w, r, StatusInternalServerError, "Failed to remove cloud provider", err)

		return

	}

	w.WriteHeader(StatusNoContent)
}

