// Package validation provides mock services for O-RAN components.

// This module implements comprehensive mock services for Near-RT RIC, SMO, and E2 interfaces.

// to support comprehensive O-RAN interface integration testing.

package test_validation

import (
	"fmt"
	"sync"
	"time"
)

// NewRICMockService creates a new RIC mock service.

func NewRICMockService(endpoint string) *RICMockService {

	return &RICMockService{

		endpoint: endpoint,

		policies: make(map[string]*A1Policy),

		subscriptions: make(map[string]*E2Subscription),

		xApps: make(map[string]*XAppConfig),

		isHealthy: true,

		latencySimMs: 50, // 50ms simulated latency

	}

}

// NewSMOMockService creates a new SMO mock service.

func NewSMOMockService(endpoint string) *SMOMockService {

	return &SMOMockService{

		endpoint: endpoint,

		managedElements: make(map[string]*ManagedElement),

		configurations: make(map[string]*O1Configuration),

		isHealthy: true,

		latencySimMs: 30, // 30ms simulated latency

	}

}

// NewE2MockService creates a new E2 mock service.

func NewE2MockService(endpoint string) *E2MockService {

	return &E2MockService{

		endpoint: endpoint,

		connectedNodes: make(map[string]*E2Node),

		subscriptions: make(map[string]*E2Subscription),

		serviceModels: make(map[string]*ServiceModel),

		isHealthy: true,

		latencySimMs: 20, // 20ms simulated latency

	}

}

// RICMockService implementation.

var ricMutex sync.RWMutex

// CreatePolicy creates a new A1 policy.

func (rms *RICMockService) CreatePolicy(policy *A1Policy) error {

	ricMutex.Lock()

	defer ricMutex.Unlock()

	// Simulate processing latency.

	time.Sleep(time.Duration(rms.latencySimMs) * time.Millisecond)

	if !rms.isHealthy {

		return fmt.Errorf("RIC service is not healthy")

	}

	if _, exists := rms.policies[policy.PolicyID]; exists {

		return fmt.Errorf("policy %s already exists", policy.PolicyID)

	}

	policy.CreatedAt = time.Now()

	policy.UpdatedAt = time.Now()

	policy.Status = "ACTIVE"

	rms.policies[policy.PolicyID] = policy

	return nil

}

// GetPolicy retrieves an A1 policy.

func (rms *RICMockService) GetPolicy(policyID string) (*A1Policy, error) {

	ricMutex.RLock()

	defer ricMutex.RUnlock()

	time.Sleep(time.Duration(rms.latencySimMs) * time.Millisecond)

	if !rms.isHealthy {

		return nil, fmt.Errorf("RIC service is not healthy")

	}

	policy, exists := rms.policies[policyID]

	if !exists {

		return nil, fmt.Errorf("policy %s not found", policyID)

	}

	return policy, nil

}

// UpdatePolicy updates an A1 policy.

func (rms *RICMockService) UpdatePolicy(policy *A1Policy) error {

	ricMutex.Lock()

	defer ricMutex.Unlock()

	time.Sleep(time.Duration(rms.latencySimMs) * time.Millisecond)

	if !rms.isHealthy {

		return fmt.Errorf("RIC service is not healthy")

	}

	if _, exists := rms.policies[policy.PolicyID]; !exists {

		return fmt.Errorf("policy %s not found", policy.PolicyID)

	}

	policy.UpdatedAt = time.Now()

	rms.policies[policy.PolicyID] = policy

	return nil

}

// DeletePolicy deletes an A1 policy.

func (rms *RICMockService) DeletePolicy(policyID string) error {

	ricMutex.Lock()

	defer ricMutex.Unlock()

	time.Sleep(time.Duration(rms.latencySimMs) * time.Millisecond)

	if !rms.isHealthy {

		return fmt.Errorf("RIC service is not healthy")

	}

	if _, exists := rms.policies[policyID]; !exists {

		return fmt.Errorf("policy %s not found", policyID)

	}

	delete(rms.policies, policyID)

	return nil

}

// DeployXApp deploys an xApp to the Near-RT RIC.

func (rms *RICMockService) DeployXApp(xappConfig *XAppConfig) error {

	ricMutex.Lock()

	defer ricMutex.Unlock()

	time.Sleep(time.Duration(rms.latencySimMs*2) * time.Millisecond) // Deployment takes longer

	if !rms.isHealthy {

		return fmt.Errorf("RIC service is not healthy")

	}

	if _, exists := rms.xApps[xappConfig.Name]; exists {

		return fmt.Errorf("xApp %s already deployed", xappConfig.Name)

	}

	xappConfig.Status = "RUNNING"

	xappConfig.DeployedAt = time.Now()

	rms.xApps[xappConfig.Name] = xappConfig

	return nil

}

// GetXApp retrieves xApp configuration.

func (rms *RICMockService) GetXApp(name string) (*XAppConfig, error) {

	ricMutex.RLock()

	defer ricMutex.RUnlock()

	time.Sleep(time.Duration(rms.latencySimMs) * time.Millisecond)

	if !rms.isHealthy {

		return nil, fmt.Errorf("RIC service is not healthy")

	}

	xapp, exists := rms.xApps[name]

	if !exists {

		return nil, fmt.Errorf("xApp %s not found", name)

	}

	return xapp, nil

}

// UndeployXApp undeploys an xApp from the Near-RT RIC.

func (rms *RICMockService) UndeployXApp(name string) error {

	ricMutex.Lock()

	defer ricMutex.Unlock()

	time.Sleep(time.Duration(rms.latencySimMs) * time.Millisecond)

	if !rms.isHealthy {

		return fmt.Errorf("RIC service is not healthy")

	}

	if _, exists := rms.xApps[name]; !exists {

		return fmt.Errorf("xApp %s not found", name)

	}

	delete(rms.xApps, name)

	return nil

}

// ListPolicies lists all A1 policies.

func (rms *RICMockService) ListPolicies() ([]*A1Policy, error) {

	ricMutex.RLock()

	defer ricMutex.RUnlock()

	time.Sleep(time.Duration(rms.latencySimMs) * time.Millisecond)

	if !rms.isHealthy {

		return nil, fmt.Errorf("RIC service is not healthy")

	}

	policies := make([]*A1Policy, 0, len(rms.policies))

	for _, policy := range rms.policies {

		policies = append(policies, policy)

	}

	return policies, nil

}

// ListXApps lists all deployed xApps.

func (rms *RICMockService) ListXApps() ([]*XAppConfig, error) {

	ricMutex.RLock()

	defer ricMutex.RUnlock()

	time.Sleep(time.Duration(rms.latencySimMs) * time.Millisecond)

	if !rms.isHealthy {

		return nil, fmt.Errorf("RIC service is not healthy")

	}

	xapps := make([]*XAppConfig, 0, len(rms.xApps))

	for _, xapp := range rms.xApps {

		xapps = append(xapps, xapp)

	}

	return xapps, nil

}

// GetHealthStatus returns the health status of the RIC service.

func (rms *RICMockService) GetHealthStatus() bool {

	ricMutex.RLock()

	defer ricMutex.RUnlock()

	return rms.isHealthy

}

// SetHealthStatus sets the health status of the RIC service.

func (rms *RICMockService) SetHealthStatus(healthy bool) {

	ricMutex.Lock()

	defer ricMutex.Unlock()

	rms.isHealthy = healthy

}

// Cleanup performs cleanup of RIC mock service resources.

func (rms *RICMockService) Cleanup() {

	ricMutex.Lock()

	defer ricMutex.Unlock()

	// Clear all maps.

	rms.policies = make(map[string]*A1Policy)

	rms.subscriptions = make(map[string]*E2Subscription)

	rms.xApps = make(map[string]*XAppConfig)

	rms.isHealthy = true

}

// SMOMockService implementation.

var smoMutex sync.RWMutex

// AddManagedElement adds a managed element to SMO.

func (sms *SMOMockService) AddManagedElement(element *ManagedElement) error {

	smoMutex.Lock()

	defer smoMutex.Unlock()

	time.Sleep(time.Duration(sms.latencySimMs) * time.Millisecond)

	if !sms.isHealthy {

		return fmt.Errorf("SMO service is not healthy")

	}

	if _, exists := sms.managedElements[element.ElementID]; exists {

		return fmt.Errorf("managed element %s already exists", element.ElementID)

	}

	element.LastSync = time.Now()

	sms.managedElements[element.ElementID] = element

	return nil

}

// GetManagedElement retrieves a managed element.

func (sms *SMOMockService) GetManagedElement(elementID string) (*ManagedElement, error) {

	smoMutex.RLock()

	defer smoMutex.RUnlock()

	time.Sleep(time.Duration(sms.latencySimMs) * time.Millisecond)

	if !sms.isHealthy {

		return nil, fmt.Errorf("SMO service is not healthy")

	}

	element, exists := sms.managedElements[elementID]

	if !exists {

		return nil, fmt.Errorf("managed element %s not found", elementID)

	}

	return element, nil

}

// UpdateManagedElement updates a managed element.

func (sms *SMOMockService) UpdateManagedElement(element *ManagedElement) error {

	smoMutex.Lock()

	defer smoMutex.Unlock()

	time.Sleep(time.Duration(sms.latencySimMs) * time.Millisecond)

	if !sms.isHealthy {

		return fmt.Errorf("SMO service is not healthy")

	}

	if _, exists := sms.managedElements[element.ElementID]; !exists {

		return fmt.Errorf("managed element %s not found", element.ElementID)

	}

	element.LastSync = time.Now()

	sms.managedElements[element.ElementID] = element

	return nil

}

// RemoveManagedElement removes a managed element from SMO.

func (sms *SMOMockService) RemoveManagedElement(elementID string) error {

	smoMutex.Lock()

	defer smoMutex.Unlock()

	time.Sleep(time.Duration(sms.latencySimMs) * time.Millisecond)

	if !sms.isHealthy {

		return fmt.Errorf("SMO service is not healthy")

	}

	if _, exists := sms.managedElements[elementID]; !exists {

		return fmt.Errorf("managed element %s not found", elementID)

	}

	delete(sms.managedElements, elementID)

	return nil

}

// ApplyConfiguration applies O1 configuration to a managed element.

func (sms *SMOMockService) ApplyConfiguration(config *O1Configuration) error {

	smoMutex.Lock()

	defer smoMutex.Unlock()

	time.Sleep(time.Duration(sms.latencySimMs*2) * time.Millisecond) // Configuration takes longer

	if !sms.isHealthy {

		return fmt.Errorf("SMO service is not healthy")

	}

	// Check if managed element exists.

	if _, exists := sms.managedElements[config.ElementID]; !exists {

		return fmt.Errorf("managed element %s not found", config.ElementID)

	}

	config.AppliedAt = time.Now()

	sms.configurations[config.ConfigID] = config

	// Update managed element configuration.

	element := sms.managedElements[config.ElementID]

	if element.Configuration == nil {

		element.Configuration = make(map[string]interface{})

	}

	// Merge configuration based on type.

	switch config.ConfigType {

	case "FCAPS":

		element.Configuration["fcapsConfig"] = config.ConfigData

	case "SECURITY":

		element.Configuration["securityConfig"] = config.ConfigData

	case "PERFORMANCE":

		element.Configuration["performanceConfig"] = config.ConfigData

	default:

		element.Configuration[config.ConfigType] = config.ConfigData

	}

	element.LastSync = time.Now()

	return nil

}

// GetConfiguration retrieves O1 configuration.

func (sms *SMOMockService) GetConfiguration(configID string) (*O1Configuration, error) {

	smoMutex.RLock()

	defer smoMutex.RUnlock()

	time.Sleep(time.Duration(sms.latencySimMs) * time.Millisecond)

	if !sms.isHealthy {

		return nil, fmt.Errorf("SMO service is not healthy")

	}

	config, exists := sms.configurations[configID]

	if !exists {

		return nil, fmt.Errorf("configuration %s not found", configID)

	}

	return config, nil

}

// ListManagedElements lists all managed elements.

func (sms *SMOMockService) ListManagedElements() ([]*ManagedElement, error) {

	smoMutex.RLock()

	defer smoMutex.RUnlock()

	time.Sleep(time.Duration(sms.latencySimMs) * time.Millisecond)

	if !sms.isHealthy {

		return nil, fmt.Errorf("SMO service is not healthy")

	}

	elements := make([]*ManagedElement, 0, len(sms.managedElements))

	for _, element := range sms.managedElements {

		elements = append(elements, element)

	}

	return elements, nil

}

// ListConfigurations lists all O1 configurations.

func (sms *SMOMockService) ListConfigurations() ([]*O1Configuration, error) {

	smoMutex.RLock()

	defer smoMutex.RUnlock()

	time.Sleep(time.Duration(sms.latencySimMs) * time.Millisecond)

	if !sms.isHealthy {

		return nil, fmt.Errorf("SMO service is not healthy")

	}

	configs := make([]*O1Configuration, 0, len(sms.configurations))

	for _, config := range sms.configurations {

		configs = append(configs, config)

	}

	return configs, nil

}

// GetHealthStatus returns the health status of the SMO service.

func (sms *SMOMockService) GetHealthStatus() bool {

	smoMutex.RLock()

	defer smoMutex.RUnlock()

	return sms.isHealthy

}

// SetHealthStatus sets the health status of the SMO service.

func (sms *SMOMockService) SetHealthStatus(healthy bool) {

	smoMutex.Lock()

	defer smoMutex.Unlock()

	sms.isHealthy = healthy

}

// Cleanup performs cleanup of SMO mock service resources.

func (sms *SMOMockService) Cleanup() {

	smoMutex.Lock()

	defer smoMutex.Unlock()

	// Clear all maps.

	sms.managedElements = make(map[string]*ManagedElement)

	sms.configurations = make(map[string]*O1Configuration)

	sms.isHealthy = true

}

// E2MockService implementation.

var e2Mutex sync.RWMutex

// RegisterNode registers an E2 node.

func (ems *E2MockService) RegisterNode(node *E2Node) error {

	e2Mutex.Lock()

	defer e2Mutex.Unlock()

	time.Sleep(time.Duration(ems.latencySimMs) * time.Millisecond)

	if !ems.isHealthy {

		return fmt.Errorf("E2 service is not healthy")

	}

	if _, exists := ems.connectedNodes[node.NodeID]; exists {

		return fmt.Errorf("E2 node %s already registered", node.NodeID)

	}

	node.Status = "CONNECTED"

	node.LastHeartbeat = time.Now()

	ems.connectedNodes[node.NodeID] = node

	return nil

}

// GetNode retrieves an E2 node.

func (ems *E2MockService) GetNode(nodeID string) (*E2Node, error) {

	e2Mutex.RLock()

	defer e2Mutex.RUnlock()

	time.Sleep(time.Duration(ems.latencySimMs) * time.Millisecond)

	if !ems.isHealthy {

		return nil, fmt.Errorf("E2 service is not healthy")

	}

	node, exists := ems.connectedNodes[nodeID]

	if !exists {

		return nil, fmt.Errorf("E2 node %s not found", nodeID)

	}

	return node, nil

}

// UnregisterNode unregisters an E2 node.

func (ems *E2MockService) UnregisterNode(nodeID string) error {

	e2Mutex.Lock()

	defer e2Mutex.Unlock()

	time.Sleep(time.Duration(ems.latencySimMs) * time.Millisecond)

	if !ems.isHealthy {

		return fmt.Errorf("E2 service is not healthy")

	}

	if _, exists := ems.connectedNodes[nodeID]; !exists {

		return fmt.Errorf("E2 node %s not found", nodeID)

	}

	// Update node status to disconnected before removing.

	ems.connectedNodes[nodeID].Status = "DISCONNECTED"

	delete(ems.connectedNodes, nodeID)

	return nil

}

// CreateSubscription creates an E2 subscription.

func (ems *E2MockService) CreateSubscription(subscription *E2Subscription) error {

	e2Mutex.Lock()

	defer e2Mutex.Unlock()

	time.Sleep(time.Duration(ems.latencySimMs) * time.Millisecond)

	if !ems.isHealthy {

		return fmt.Errorf("E2 service is not healthy")

	}

	// Check if node exists.

	if _, exists := ems.connectedNodes[subscription.NodeID]; !exists {

		return fmt.Errorf("E2 node %s not connected", subscription.NodeID)

	}

	if _, exists := ems.subscriptions[subscription.SubscriptionID]; exists {

		return fmt.Errorf("E2 subscription %s already exists", subscription.SubscriptionID)

	}

	subscription.Status = "ACTIVE"

	subscription.CreatedAt = time.Now()

	ems.subscriptions[subscription.SubscriptionID] = subscription

	return nil

}

// GetSubscription retrieves an E2 subscription.

func (ems *E2MockService) GetSubscription(subscriptionID string) (*E2Subscription, error) {

	e2Mutex.RLock()

	defer e2Mutex.RUnlock()

	time.Sleep(time.Duration(ems.latencySimMs) * time.Millisecond)

	if !ems.isHealthy {

		return nil, fmt.Errorf("E2 service is not healthy")

	}

	subscription, exists := ems.subscriptions[subscriptionID]

	if !exists {

		return nil, fmt.Errorf("E2 subscription %s not found", subscriptionID)

	}

	return subscription, nil

}

// UpdateSubscription updates an E2 subscription.

func (ems *E2MockService) UpdateSubscription(subscription *E2Subscription) error {

	e2Mutex.Lock()

	defer e2Mutex.Unlock()

	time.Sleep(time.Duration(ems.latencySimMs) * time.Millisecond)

	if !ems.isHealthy {

		return fmt.Errorf("E2 service is not healthy")

	}

	if _, exists := ems.subscriptions[subscription.SubscriptionID]; !exists {

		return fmt.Errorf("E2 subscription %s not found", subscription.SubscriptionID)

	}

	ems.subscriptions[subscription.SubscriptionID] = subscription

	return nil

}

// DeleteSubscription deletes an E2 subscription.

func (ems *E2MockService) DeleteSubscription(subscriptionID string) error {

	e2Mutex.Lock()

	defer e2Mutex.Unlock()

	time.Sleep(time.Duration(ems.latencySimMs) * time.Millisecond)

	if !ems.isHealthy {

		return fmt.Errorf("E2 service is not healthy")

	}

	if _, exists := ems.subscriptions[subscriptionID]; !exists {

		return fmt.Errorf("E2 subscription %s not found", subscriptionID)

	}

	delete(ems.subscriptions, subscriptionID)

	return nil

}

// RegisterServiceModel registers an E2 service model.

func (ems *E2MockService) RegisterServiceModel(model *ServiceModel) error {

	e2Mutex.Lock()

	defer e2Mutex.Unlock()

	time.Sleep(time.Duration(ems.latencySimMs) * time.Millisecond)

	if !ems.isHealthy {

		return fmt.Errorf("E2 service is not healthy")

	}

	modelKey := fmt.Sprintf("%s-%s", model.ModelName, model.Version)

	if _, exists := ems.serviceModels[modelKey]; exists {

		return fmt.Errorf("service model %s version %s already registered", model.ModelName, model.Version)

	}

	ems.serviceModels[modelKey] = model

	return nil

}

// GetServiceModel retrieves a service model.

func (ems *E2MockService) GetServiceModel(modelName, version string) (*ServiceModel, error) {

	e2Mutex.RLock()

	defer e2Mutex.RUnlock()

	time.Sleep(time.Duration(ems.latencySimMs) * time.Millisecond)

	if !ems.isHealthy {

		return nil, fmt.Errorf("E2 service is not healthy")

	}

	modelKey := fmt.Sprintf("%s-%s", modelName, version)

	model, exists := ems.serviceModels[modelKey]

	if !exists {

		return nil, fmt.Errorf("service model %s version %s not found", modelName, version)

	}

	return model, nil

}

// ListNodes lists all connected E2 nodes.

func (ems *E2MockService) ListNodes() ([]*E2Node, error) {

	e2Mutex.RLock()

	defer e2Mutex.RUnlock()

	time.Sleep(time.Duration(ems.latencySimMs) * time.Millisecond)

	if !ems.isHealthy {

		return nil, fmt.Errorf("E2 service is not healthy")

	}

	nodes := make([]*E2Node, 0, len(ems.connectedNodes))

	for _, node := range ems.connectedNodes {

		nodes = append(nodes, node)

	}

	return nodes, nil

}

// ListSubscriptions lists all E2 subscriptions.

func (ems *E2MockService) ListSubscriptions() ([]*E2Subscription, error) {

	e2Mutex.RLock()

	defer e2Mutex.RUnlock()

	time.Sleep(time.Duration(ems.latencySimMs) * time.Millisecond)

	if !ems.isHealthy {

		return nil, fmt.Errorf("E2 service is not healthy")

	}

	subscriptions := make([]*E2Subscription, 0, len(ems.subscriptions))

	for _, subscription := range ems.subscriptions {

		subscriptions = append(subscriptions, subscription)

	}

	return subscriptions, nil

}

// ListServiceModels lists all registered service models.

func (ems *E2MockService) ListServiceModels() ([]*ServiceModel, error) {

	e2Mutex.RLock()

	defer e2Mutex.RUnlock()

	time.Sleep(time.Duration(ems.latencySimMs) * time.Millisecond)

	if !ems.isHealthy {

		return nil, fmt.Errorf("E2 service is not healthy")

	}

	models := make([]*ServiceModel, 0, len(ems.serviceModels))

	for _, model := range ems.serviceModels {

		models = append(models, model)

	}

	return models, nil

}

// SendHeartbeat sends a heartbeat from an E2 node.

func (ems *E2MockService) SendHeartbeat(nodeID string) error {

	e2Mutex.Lock()

	defer e2Mutex.Unlock()

	time.Sleep(time.Duration(ems.latencySimMs/10) * time.Millisecond) // Heartbeats are fast

	if !ems.isHealthy {

		return fmt.Errorf("E2 service is not healthy")

	}

	node, exists := ems.connectedNodes[nodeID]

	if !exists {

		return fmt.Errorf("E2 node %s not found", nodeID)

	}

	node.LastHeartbeat = time.Now()

	node.Status = "CONNECTED"

	return nil

}

// GetHealthStatus returns the health status of the E2 service.

func (ems *E2MockService) GetHealthStatus() bool {

	e2Mutex.RLock()

	defer e2Mutex.RUnlock()

	return ems.isHealthy

}

// SetHealthStatus sets the health status of the E2 service.

func (ems *E2MockService) SetHealthStatus(healthy bool) {

	e2Mutex.Lock()

	defer e2Mutex.Unlock()

	ems.isHealthy = healthy

}

// Cleanup performs cleanup of E2 mock service resources.

func (ems *E2MockService) Cleanup() {

	e2Mutex.Lock()

	defer e2Mutex.Unlock()

	// Clear all maps.

	ems.connectedNodes = make(map[string]*E2Node)

	ems.subscriptions = make(map[string]*E2Subscription)

	ems.serviceModels = make(map[string]*ServiceModel)

	ems.isHealthy = true

}
