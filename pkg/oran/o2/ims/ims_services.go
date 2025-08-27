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

package ims

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// IMSService provides core Infrastructure Management Services functionality
type IMSService struct {
	catalogService      *CatalogService
	inventoryService    *InventoryService
	lifecycleService    *LifecycleService
	subscriptionService *SubscriptionService
	startTime          time.Time
}

// NewIMSService creates a new IMS service instance
func NewIMSService(catalog *CatalogService, inventory *InventoryService, lifecycle *LifecycleService, subscription *SubscriptionService) *IMSService {
	return &IMSService{
		catalogService:      catalog,
		inventoryService:    inventory,
		lifecycleService:    lifecycle,
		subscriptionService: subscription,
		startTime:          time.Now(),
	}
}

// GetSystemInfo returns system information
func (s *IMSService) GetSystemInfo(ctx context.Context) (*models.SystemInfo, error) {
	return &models.SystemInfo{
		Name:        "Nephoran O2 IMS",
		Description: "O-RAN O2 Infrastructure Management Services",
		Version:     "1.0.0",
		APIVersions: []string{"1.0.0"},
		SupportedResourceTypes: []string{
			"compute", "storage", "network", "accelerator",
			"deployment", "service", "configmap", "secret",
		},
		Extensions: map[string]interface{}{
			"multi_cloud":       true,
			"kubernetes_native": true,
			"auto_scaling":      true,
		},
		Timestamp: time.Now(),
	}, nil
}

// GetServiceHealth returns the health status of the IMS service
func (s *IMSService) GetServiceHealth(ctx context.Context) (*ServiceHealth, error) {
	return &ServiceHealth{
		Status:    "healthy",
		LastCheck: time.Now(),
		Uptime:    time.Since(s.startTime),
		Components: map[string]string{
			"catalog":      "healthy",
			"inventory":    "healthy", 
			"lifecycle":    "healthy",
			"subscription": "healthy",
		},
		Version: "1.0.0",
	}, nil
}

// ServiceHealth represents the health status of the IMS service
type ServiceHealth struct {
	Status     string            `json:"status"`
	LastCheck  time.Time         `json:"lastCheck"`
	Uptime     time.Duration     `json:"uptime"`
	Components map[string]string `json:"components"`
	Version    string            `json:"version"`
}


// GetResourceTypes returns available resource types
func (s *CatalogService) GetResourceTypes(ctx context.Context, filter *models.ResourceTypeFilter) ([]*models.ResourceType, error) {
	// Implementation would query available resource types
	return []*models.ResourceType{
		{
			ResourceTypeID: "compute.vm",
			Name:           "Virtual Machine",
			Category:       "compute",
			Description:    "Virtual compute resource",
			Version:        "1.0.0",
		},
		{
			ResourceTypeID: "storage.volume",
			Name:           "Storage Volume",
			Category:       "storage",
			Description:    "Persistent storage resource",
			Version:        "1.0.0",
		},
	}, nil
}

// InventoryService manages infrastructure inventory
type InventoryService struct {
	kubeClient client.Client
	clientset  kubernetes.Interface
	startTime  time.Time
}

// NewInventoryService creates a new inventory service
func NewInventoryService(kubeClient client.Client, clientset kubernetes.Interface) *InventoryService {
	return &InventoryService{
		kubeClient: kubeClient,
		clientset:  clientset,
		startTime:  time.Now(),
	}
}

// ListResources returns resources with optional filtering
func (s *InventoryService) ListResources(ctx context.Context, filter *models.ResourceFilter) ([]*models.Resource, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("listing resources", "filter", filter)

	// For now, return empty list - would be implemented with actual resource management
	return []*models.Resource{}, nil
}

// GetResource retrieves a specific resource by ID
func (s *InventoryService) GetResource(ctx context.Context, resourceID string) (*models.Resource, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting resource", "resourceID", resourceID)

	// For now, return not found error - would be implemented with actual resource management
	return nil, fmt.Errorf("resource not found: %s", resourceID)
}

// CreateResource creates a new resource
func (s *InventoryService) CreateResource(ctx context.Context, req *models.CreateResourceRequest, pool *models.ResourcePool, resourceType *models.ResourceType) (*models.Resource, error) {
	logger := log.FromContext(ctx)
	logger.Info("creating resource", "name", req.Name, "type", req.ResourceTypeID, "pool", req.ResourcePoolID)

	// For now, return a basic resource - would be implemented with actual resource creation
	resource := &models.Resource{
		ResourceID:     "res-" + generateRandomID(),
		Name:           req.Name,
		Description:    req.Description,
		ResourceTypeID: req.ResourceTypeID,
		ResourcePoolID: req.ResourcePoolID,
		Extensions:     req.Extensions,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	return resource, nil
}

// UpdateResource updates an existing resource
func (s *InventoryService) UpdateResource(ctx context.Context, resourceID string, req *models.UpdateResourceRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("updating resource", "resourceID", resourceID)

	// For now, return success - would be implemented with actual resource updates
	return nil
}

// DeleteResource deletes a resource
func (s *InventoryService) DeleteResource(ctx context.Context, resourceID string) error {
	logger := log.FromContext(ctx)
	logger.Info("deleting resource", "resourceID", resourceID)

	// For now, return success - would be implemented with actual resource deletion
	return nil
}

// ListNodes returns inventory nodes (original method)
func (s *InventoryService) ListNodes(ctx context.Context, filter *models.NodeFilter) ([]*models.Node, error) {
	return s.GetNodes(ctx, filter)
}

// GetNode retrieves a specific node by ID
func (s *InventoryService) GetNode(ctx context.Context, nodeID string) (*models.Node, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting node", "nodeID", nodeID)

	// Get all nodes and find the specific one
	nodes, err := s.GetNodes(ctx, nil)
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		if node.NodeID == nodeID {
			return node, nil
		}
	}

	return nil, fmt.Errorf("node not found: %s", nodeID)
}

// GetNodes returns inventory nodes
func (s *InventoryService) GetNodes(ctx context.Context, filter *models.NodeFilter) ([]*models.Node, error) {
	logger := log.FromContext(ctx)
	
	// Implementation would query Kubernetes nodes and transform to O2 format
	nodes, err := s.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error(err, "Failed to list Kubernetes nodes")
		return nil, err
	}

	var o2Nodes []*models.Node
	for _, node := range nodes.Items {
		o2Node := &models.Node{
			NodeID:         string(node.UID),
			Name:           node.Name,
			Description:    "Kubernetes node",
			ResourcePoolID: "default-pool",
			NodeType:       "VIRTUAL",
			Status: &models.NodeStatus{
				State:         "READY",
				Phase:         "ACTIVE",
				Health:        "HEALTHY",
				LastHeartbeat: time.Now(),
			},
			Architecture: node.Status.NodeInfo.Architecture,
			Extensions: map[string]interface{}{
				"os":               node.Status.NodeInfo.OperatingSystem,
				"containerRuntime": node.Status.NodeInfo.ContainerRuntimeVersion,
				"kubeletVersion":   node.Status.NodeInfo.KubeletVersion,
			},
			CreatedAt: node.CreationTimestamp.Time,
			UpdatedAt: time.Now(),
		}

		// Add resource capacity information
		if node.Status.Capacity != nil {
			o2Node.Capacity = &models.ResourceCapacity{
				CPU: &models.ResourceMetric{
					Total:       node.Status.Capacity.Cpu().String(),
					Available:   node.Status.Allocatable.Cpu().String(),
					Used:        "0",
					Unit:        "cores",
					Utilization: 0.0,
				},
				Memory: &models.ResourceMetric{
					Total:       node.Status.Capacity.Memory().String(),
					Available:   node.Status.Allocatable.Memory().String(),
					Used:        "0",
					Unit:        "bytes",
					Utilization: 0.0,
				},
			}
		}

		o2Nodes = append(o2Nodes, o2Node)
	}

	return o2Nodes, nil
}

// LifecycleService manages resource lifecycle operations
type LifecycleService struct {
	startTime time.Time
}

// NewLifecycleService creates a new lifecycle service
func NewLifecycleService() *LifecycleService {
	return &LifecycleService{
		startTime: time.Now(),
	}
}

// ListDeployments returns deployments with optional filtering
func (s *LifecycleService) ListDeployments(ctx context.Context, filter *models.DeploymentFilter) ([]*models.Deployment, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("listing deployments", "filter", filter)

	// For now, return empty list - would be implemented with actual deployment management
	return []*models.Deployment{}, nil
}

// GetDeployment retrieves a specific deployment by ID
func (s *LifecycleService) GetDeployment(ctx context.Context, deploymentID string) (*models.Deployment, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting deployment", "deploymentID", deploymentID)

	// For now, return not found error - would be implemented with actual deployment management
	return nil, fmt.Errorf("deployment not found: %s", deploymentID)
}

// UpdateDeployment updates an existing deployment
func (s *LifecycleService) UpdateDeployment(ctx context.Context, deploymentID string, req *models.UpdateDeploymentRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("updating deployment", "deploymentID", deploymentID)

	// For now, return success - would be implemented with actual deployment updates
	return nil
}

// DeleteDeployment deletes a deployment
func (s *LifecycleService) DeleteDeployment(ctx context.Context, deploymentID string) error {
	logger := log.FromContext(ctx)
	logger.Info("deleting deployment", "deploymentID", deploymentID)

	// For now, return success - would be implemented with actual deployment deletion
	return nil
}

// CreateDeployment creates a new deployment (updated signature)
func (s *LifecycleService) CreateDeployment(ctx context.Context, req *models.CreateDeploymentRequest, template *models.DeploymentTemplate, pool *models.ResourcePool) (*models.Deployment, error) {
	// Convert metadata to interface{} map if needed
	metadata := make(map[string]interface{})
	for k, v := range req.Metadata {
		metadata[k] = v
	}

	deployment := &models.Deployment{
		DeploymentID:    generateDeploymentID(),
		Name:            req.Name,
		Description:     req.Description,
		TemplateID:      req.TemplateID,
		TemplateVersion: req.TemplateVersion,
		Parameters:      metadata, // Using converted metadata as parameters
		PoolID:          req.ResourcePoolID,
		Status:          "creating",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Extensions:      req.Extensions,
	}

	// Implementation would create actual deployment resources
	// For now, just return the deployment object

	return deployment, nil
}

// SubscriptionService manages event subscriptions
type SubscriptionService struct {
	subscriptions map[string]*models.Subscription
	alarms        map[string]*models.Alarm
	startTime     time.Time
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService() *SubscriptionService {
	return &SubscriptionService{
		subscriptions: make(map[string]*models.Subscription),
		alarms:        make(map[string]*models.Alarm),
		startTime:     time.Now(),
	}
}

// CreateSubscription creates a new event subscription
func (s *SubscriptionService) CreateSubscription(ctx context.Context, req *models.CreateSubscriptionRequest) (*models.Subscription, error) {
	subscription := &models.Subscription{
		SubscriptionID:         generateSubscriptionID(),
		Name:                   "Subscription-" + generateRandomID(),
		Description:            "Event subscription",
		CallbackUri:            req.Callback,
		ConsumerSubscriptionId: req.ConsumerSubscriptionID,
		EventTypes:             req.EventTypes,
		Status: &models.SubscriptionStatus{
			State:           "ACTIVE",
			Health:          "HEALTHY",
			LastHealthCheck: time.Now(),
		},
		Extensions: req.Metadata,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	s.subscriptions[subscription.SubscriptionID] = subscription

	return subscription, nil
}

// ListSubscriptions returns filtered subscriptions
func (s *SubscriptionService) ListSubscriptions(ctx context.Context, filter *models.SubscriptionQueryFilter) ([]*models.Subscription, error) {
	if filter == nil {
		return s.GetSubscriptions(ctx, nil)
	}
	return s.GetSubscriptions(ctx, &models.SubscriptionFilter{
		EventSources: filter.EventTypes, // SubscriptionQueryFilter uses EventTypes field
	})
}

// GetSubscriptions returns filtered subscriptions
func (s *SubscriptionService) GetSubscriptions(ctx context.Context, filter *models.SubscriptionFilter) ([]*models.Subscription, error) {
	var filteredSubs []*models.Subscription

	for _, sub := range s.subscriptions {
		if s.matchesFilter(sub, filter) {
			filteredSubs = append(filteredSubs, sub)
		}
	}

	return filteredSubs, nil
}

// GetSubscription returns a specific subscription
func (s *SubscriptionService) GetSubscription(ctx context.Context, subscriptionID string) (*models.Subscription, error) {
	if sub, exists := s.subscriptions[subscriptionID]; exists {
		return sub, nil
	}
	return nil, ErrSubscriptionNotFound
}

// UpdateSubscription updates an existing subscription
func (s *SubscriptionService) UpdateSubscription(ctx context.Context, subscriptionID string, req *models.UpdateSubscriptionRequest) error {
	sub, exists := s.subscriptions[subscriptionID]
	if !exists {
		return ErrSubscriptionNotFound
	}

	// Update fields if provided
	if req.Callback != "" {
		sub.CallbackUri = req.Callback
	}
	if req.EventTypes != nil {
		sub.EventTypes = req.EventTypes
	}
	if req.Status != "" {
		if sub.Status == nil {
			sub.Status = &models.SubscriptionStatus{}
		}
		sub.Status.State = req.Status
	}
	if req.Metadata != nil {
		sub.Extensions = req.Metadata
	}

	sub.UpdatedAt = time.Now()
	return nil
}

// DeleteSubscription removes a subscription
func (s *SubscriptionService) DeleteSubscription(ctx context.Context, subscriptionID string) error {
	if _, exists := s.subscriptions[subscriptionID]; !exists {
		return ErrSubscriptionNotFound
	}

	delete(s.subscriptions, subscriptionID)
	return nil
}

// GetEventTypes returns available notification event types
func (s *SubscriptionService) GetEventTypes(ctx context.Context) ([]*models.NotificationEventType, error) {
	// Return basic event types using correct field names
	return []*models.NotificationEventType{
		{
			EventType:   "resource.created",
			Description: "Notification when a resource is created",
			Version:     "1.0",
		},
		{
			EventType:   "resource.updated", 
			Description: "Notification when a resource is updated",
			Version:     "1.0",
		},
		{
			EventType:   "deployment.created",
			Description: "Notification when a deployment is created",
			Version:     "1.0",
		},
	}, nil
}

// GetAlarms returns alarms with optional filtering
func (s *SubscriptionService) GetAlarms(ctx context.Context, filter *models.AlarmFilter) ([]*models.Alarm, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting alarms", "filter", filter)

	var alarms []*models.Alarm
	for _, alarm := range s.alarms {
		alarms = append(alarms, alarm)
	}

	return alarms, nil
}

// GetAlarm retrieves a specific alarm by ID
func (s *SubscriptionService) GetAlarm(ctx context.Context, alarmID string) (*models.Alarm, error) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("getting alarm", "alarmID", alarmID)

	alarm, exists := s.alarms[alarmID]
	if !exists {
		return nil, fmt.Errorf("alarm not found: %s", alarmID)
	}

	return alarm, nil
}

// AcknowledgeAlarm acknowledges an alarm  
func (s *SubscriptionService) AcknowledgeAlarm(ctx context.Context, alarmID string, req *models.AlarmAcknowledgementRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("acknowledging alarm", "alarmID", alarmID, "acknowledgedBy", req.AckUser)

	alarm, exists := s.alarms[alarmID]
	if !exists {
		return fmt.Errorf("alarm not found: %s", alarmID)
	}

	// Update alarm acknowledgment fields
	alarm.AckState = "ACKNOWLEDGED"
	alarm.AckUser = req.AckUser
	alarm.AckSystemId = req.AckSystemId
	now := time.Now()
	alarm.AlarmAckTime = &now

	return nil
}

// ClearAlarm clears an alarm
func (s *SubscriptionService) ClearAlarm(ctx context.Context, alarmID string, req *models.AlarmClearRequest) error {
	logger := log.FromContext(ctx)
	logger.Info("clearing alarm", "alarmID", alarmID, "clearedBy", req.ClearUser)

	alarm, exists := s.alarms[alarmID]
	if !exists {
		return fmt.Errorf("alarm not found: %s", alarmID)
	}

	// Update alarm state to cleared
	alarm.AlarmState = "CLEARED"
	now := time.Now()
	alarm.AlarmClearTime = &now
	
	// Store clear information in extensions
	if alarm.Extensions == nil {
		alarm.Extensions = make(map[string]interface{})
	}
	alarm.Extensions["clearUser"] = req.ClearUser
	alarm.Extensions["clearSystemId"] = req.ClearSystemId
	alarm.Extensions["clearReason"] = req.ClearReason

	return nil
}

// Helper methods
func (s *SubscriptionService) matchesFilter(sub *models.Subscription, filter *models.SubscriptionFilter) bool {
	if filter == nil {
		return true
	}

	// The current SubscriptionFilter model is focused on event filtering
	// For basic subscription listing, we'll implement a simple match
	// Event type filtering
	if len(sub.EventTypes) > 0 && len(filter.EventSources) > 0 {
		hasMatchingEventType := false
		for _, filterSource := range filter.EventSources {
			for _, subEventType := range sub.EventTypes {
				if filterSource == subEventType {
					hasMatchingEventType = true
					break
				}
			}
			if hasMatchingEventType {
				break
			}
		}
		if !hasMatchingEventType {
			return false
		}
	}

	return true
}

// Utility functions
func generateDeploymentID() string {
	return "deploy-" + generateRandomID()
}

func generateSubscriptionID() string {
	return "sub-" + generateRandomID()
}

func generateRandomID() string {
	// Simple timestamp-based ID generation
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, n)
	for i := range result {
		result[i] = letters[time.Now().Nanosecond()%len(letters)]
	}
	return string(result)
}

// Error definitions
var (
	ErrSubscriptionNotFound = NewO2Error("SUBSCRIPTION_NOT_FOUND", "Subscription not found")
	ErrDeploymentNotFound   = NewO2Error("DEPLOYMENT_NOT_FOUND", "Deployment not found")
	ErrResourcePoolNotFound = NewO2Error("RESOURCE_POOL_NOT_FOUND", "Resource pool not found")
	ErrResourceNotFound     = NewO2Error("RESOURCE_NOT_FOUND", "Resource not found")
)

// O2Error represents an O2 service error
type O2Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *O2Error) Error() string {
	return e.Code + ": " + e.Message
}

// NewO2Error creates a new O2 error
func NewO2Error(code, message string) *O2Error {
	return &O2Error{
		Code:    code,
		Message: message,
	}
}