package oran

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SMOManager manages Service Management and Orchestration integration.

type SMOManager struct {
	mu sync.RWMutex

	config *SMOConfig

	httpClient *http.Client

	policyManager *PolicyManager

	serviceRegistry *ServiceRegistry

	orchestrator *ServiceOrchestrator

	connected bool
}

// SMOConfig holds SMO configuration.

type SMOConfig struct {
	Endpoint string `json:"endpoint"`

	Username string `json:"username"`

	Password string `json:"password"`

	TLSConfig *TLSConfig `json:"tls_config"`

	APIVersion string `json:"api_version"`

	RetryCount int `json:"retry_count"`

	RetryDelay time.Duration `json:"retry_delay"`

	Timeout time.Duration `json:"timeout"`

	ExtraHeaders map[string]string `json:"extra_headers"`

	HealthCheckPath string `json:"health_check_path"`
}

// PolicyManager manages A1 policy orchestration with SMO.

type PolicyManager struct {
	mu sync.RWMutex

	smoClient *SMOClient

	policies map[string]*A1Policy

	policyTypes map[string]*A1PolicyType

	subscriptions map[string]*PolicySubscription

	eventCallbacks map[string]func(*PolicyEvent)
}

// ServiceRegistry manages service discovery and registration with SMO.

type ServiceRegistry struct {
	mu sync.RWMutex

	smoClient *SMOClient

	registeredServices map[string]*ServiceInstance

	discoveredServices map[string]*ServiceInstance

	healthCheckers map[string]*ServiceHealthChecker
}

// ServiceOrchestrator manages Non-RT RIC service orchestration.

type ServiceOrchestrator struct {
	mu sync.RWMutex

	smoClient *SMOClient

	rApps map[string]*RAppInstance

	workflows map[string]*OrchestrationWorkflow

	lifecycleHooks map[string][]LifecycleHook
}

// SMOClient handles HTTP communication with SMO.

type SMOClient struct {
	baseURL string

	httpClient *http.Client

	auth *AuthConfig

	headers map[string]string
}

// Policy management types.

type A1Policy struct {
	ID string `json:"id"`

	TypeID string `json:"type_id"`

	Version string `json:"version"`

	Description string `json:"description"`

	Status string `json:"status"` // ACTIVE, INACTIVE, ENFORCING

	Data map[string]interface{} `json:"data"`

	TargetRICs []string `json:"target_rics"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`

	EnforcedAt *time.Time `json:"enforced_at,omitempty"`

	Metadata map[string]string `json:"metadata"`
}

// A1PolicyType represents a a1policytype.

type A1PolicyType struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Version string `json:"version"`

	Description string `json:"description"`

	Schema map[string]interface{} `json:"schema"`

	SupportedRICs []string `json:"supported_rics"`

	CreatedAt time.Time `json:"created_at"`
}

// PolicySubscription represents a policysubscription.

type PolicySubscription struct {
	ID string `json:"id"`

	PolicyID string `json:"policy_id"`

	Subscriber string `json:"subscriber"`

	Events []string `json:"events"`

	Callback func(*PolicyEvent) `json:"-"`

	CreatedAt time.Time `json:"created_at"`
}

// PolicyEvent represents a policyevent.

type PolicyEvent struct {
	ID string `json:"id"`

	Type string `json:"type"` // CREATED, UPDATED, DELETED, VIOLATED

	PolicyID string `json:"policy_id"`

	RIC string `json:"ric"`

	Timestamp time.Time `json:"timestamp"`

	Data map[string]interface{} `json:"data"`

	Severity string `json:"severity"`
}

// Service management types.

type ServiceInstance struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Type string `json:"type"` // RIC, xApp, rApp

	Version string `json:"version"`

	Endpoint string `json:"endpoint"`

	Status string `json:"status"` // REGISTERED, ACTIVE, INACTIVE, FAILED

	Capabilities []string `json:"capabilities"`

	Metadata map[string]string `json:"metadata"`

	HealthCheck *HealthCheckConfig `json:"health_check"`

	RegisteredAt time.Time `json:"registered_at"`

	LastHeartbeat time.Time `json:"last_heartbeat"`
}

// ServiceHealthChecker represents a servicehealthchecker.

type ServiceHealthChecker struct {
	serviceID string

	config *HealthCheckConfig

	status string

	lastCheck time.Time

	failures int

	stopCh chan struct{}
}

// HealthCheckConfig represents a healthcheckconfig.

type HealthCheckConfig struct {
	Enabled bool `json:"enabled"`

	Path string `json:"path"`

	Interval time.Duration `json:"interval"`

	Timeout time.Duration `json:"timeout"`

	FailureThreshold int `json:"failure_threshold"`

	SuccessThreshold int `json:"success_threshold"`
}

// rApp orchestration types.

type RAppInstance struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Type string `json:"type"`

	Version string `json:"version"`

	Image string `json:"image"`

	Status string `json:"status"` // DEPLOYING, RUNNING, STOPPED, FAILED

	Configuration map[string]interface{} `json:"configuration"`

	Resources *ResourceRequirements `json:"resources"`

	Dependencies []string `json:"dependencies"`

	Lifecycle *LifecycleConfig `json:"lifecycle"`

	CreatedAt time.Time `json:"created_at"`

	StartedAt *time.Time `json:"started_at,omitempty"`
}

// OrchestrationWorkflow represents a orchestrationworkflow.

type OrchestrationWorkflow struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Steps []*WorkflowStep `json:"steps"`

	Status string `json:"status"` // PENDING, RUNNING, COMPLETED, FAILED

	Context map[string]interface{} `json:"context"`

	CreatedAt time.Time `json:"created_at"`

	StartedAt *time.Time `json:"started_at,omitempty"`

	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// WorkflowStep represents a workflowstep.

type WorkflowStep struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Type string `json:"type"` // DEPLOY, CONFIGURE, VALIDATE, NOTIFICATION

	Action string `json:"action"`

	Parameters map[string]interface{} `json:"parameters"`

	Dependencies []string `json:"dependencies"`

	Status string `json:"status"` // PENDING, RUNNING, COMPLETED, FAILED

	Output map[string]interface{} `json:"output,omitempty"`

	Error string `json:"error,omitempty"`
}

// ResourceRequirements represents a resourcerequirements.

type ResourceRequirements struct {
	CPU string `json:"cpu"`

	Memory string `json:"memory"`

	Storage string `json:"storage"`
}

// LifecycleConfig represents a lifecycleconfig.

type LifecycleConfig struct {
	PreStart []LifecycleHook `json:"pre_start"`

	PostStart []LifecycleHook `json:"post_start"`

	PreStop []LifecycleHook `json:"pre_stop"`

	PostStop []LifecycleHook `json:"post_stop"`
}

// LifecycleHook represents a lifecyclehook.

type LifecycleHook struct {
	Name string `json:"name"`

	Type string `json:"type"` // COMMAND, HTTP, SCRIPT

	Action string `json:"action"`

	Params map[string]interface{} `json:"params"`

	Timeout time.Duration `json:"timeout"`
}

// NewSMOManager creates a new SMO manager.

func NewSMOManager(config *SMOConfig) (*SMOManager, error) {

	if config == nil {

		return nil, fmt.Errorf("SMO configuration is required")

	}

	// Set defaults.

	if config.APIVersion == "" {

		config.APIVersion = "v1"

	}

	if config.RetryCount == 0 {

		config.RetryCount = 3

	}

	if config.RetryDelay == 0 {

		config.RetryDelay = 2 * time.Second

	}

	if config.Timeout == 0 {

		config.Timeout = 30 * time.Second

	}

	if config.HealthCheckPath == "" {

		config.HealthCheckPath = "/health"

	}

	httpClient := &http.Client{

		Timeout: config.Timeout,
	}

	// Configure TLS if provided.

	if config.TLSConfig != nil {

		// TLS configuration would be applied here.

	}

	smoClient := &SMOClient{

		baseURL: config.Endpoint,

		httpClient: httpClient,

		auth: &AuthConfig{

			Type: "basic",

			Username: config.Username,

			Password: config.Password,
		},

		headers: config.ExtraHeaders,
	}

	manager := &SMOManager{

		config: config,

		httpClient: httpClient,

		policyManager: &PolicyManager{

			smoClient: smoClient,

			policies: make(map[string]*A1Policy),

			policyTypes: make(map[string]*A1PolicyType),

			subscriptions: make(map[string]*PolicySubscription),

			eventCallbacks: make(map[string]func(*PolicyEvent)),
		},

		serviceRegistry: &ServiceRegistry{

			smoClient: smoClient,

			registeredServices: make(map[string]*ServiceInstance),

			discoveredServices: make(map[string]*ServiceInstance),

			healthCheckers: make(map[string]*ServiceHealthChecker),
		},

		orchestrator: &ServiceOrchestrator{

			smoClient: smoClient,

			rApps: make(map[string]*RAppInstance),

			workflows: make(map[string]*OrchestrationWorkflow),

			lifecycleHooks: make(map[string][]LifecycleHook),
		},
	}

	return manager, nil

}

// Start starts the SMO manager.

func (sm *SMOManager) Start(ctx context.Context) error {

	logger := log.FromContext(ctx)

	logger.Info("starting SMO manager", "endpoint", sm.config.Endpoint)

	// Test connection to SMO.

	if err := sm.testConnection(ctx); err != nil {

		return fmt.Errorf("failed to connect to SMO: %w", err)

	}

	sm.mu.Lock()

	sm.connected = true

	sm.mu.Unlock()

	// Start service health checking.

	go sm.serviceRegistry.startHealthChecking(ctx)

	// Start policy event monitoring.

	go sm.policyManager.startEventMonitoring(ctx)

	logger.Info("SMO manager started successfully")

	return nil

}

// Stop stops the SMO manager.

func (sm *SMOManager) Stop() {

	sm.mu.Lock()

	defer sm.mu.Unlock()

	sm.connected = false

	// Stop health checkers.

	for _, checker := range sm.serviceRegistry.healthCheckers {

		close(checker.stopCh)

	}

}

// testConnection tests connection to SMO.

func (sm *SMOManager) testConnection(ctx context.Context) error {

	url := fmt.Sprintf("%s%s", sm.config.Endpoint, sm.config.HealthCheckPath)

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	// Add authentication.

	if sm.config.Username != "" && sm.config.Password != "" {

		req.SetBasicAuth(sm.config.Username, sm.config.Password)

	}

	// Add extra headers.

	for key, value := range sm.config.ExtraHeaders {

		req.Header.Set(key, value)

	}

	resp, err := sm.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("connection test failed: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {

		return fmt.Errorf("SMO health check failed with status: %d", resp.StatusCode)

	}

	return nil

}

// IsConnected returns true if connected to SMO.

func (sm *SMOManager) IsConnected() bool {

	sm.mu.RLock()

	defer sm.mu.RUnlock()

	return sm.connected

}

// Policy Management Methods.

// CreatePolicy creates a new A1 policy.

func (pm *PolicyManager) CreatePolicy(ctx context.Context, policy *A1Policy) error {

	logger := log.FromContext(ctx)

	logger.Info("creating A1 policy", "policyID", policy.ID, "typeID", policy.TypeID)

	// Validate policy type exists.

	if _, exists := pm.policyTypes[policy.TypeID]; !exists {

		return fmt.Errorf("policy type %s not found", policy.TypeID)

	}

	// Set timestamps.

	policy.CreatedAt = time.Now()

	policy.UpdatedAt = time.Now()

	policy.Status = "ACTIVE"

	// Call SMO API to create policy.

	url := fmt.Sprintf("%s/api/%s/policies", pm.smoClient.baseURL, "v1")

	if err := pm.smoClient.post(ctx, url, policy, nil); err != nil {

		return fmt.Errorf("failed to create policy via SMO: %w", err)

	}

	pm.mu.Lock()

	pm.policies[policy.ID] = policy

	pm.mu.Unlock()

	logger.Info("A1 policy created successfully", "policyID", policy.ID)

	return nil

}

// UpdatePolicy updates an existing A1 policy.

func (pm *PolicyManager) UpdatePolicy(ctx context.Context, policyID string, updates map[string]interface{}) error {

	logger := log.FromContext(ctx)

	logger.Info("updating A1 policy", "policyID", policyID)

	pm.mu.Lock()

	policy, exists := pm.policies[policyID]

	if !exists {

		pm.mu.Unlock()

		return fmt.Errorf("policy %s not found", policyID)

	}

	// Apply updates.

	for key, value := range updates {

		switch key {

		case "data":

			if data, ok := value.(map[string]interface{}); ok {

				policy.Data = data

			}

		case "status":

			if status, ok := value.(string); ok {

				policy.Status = status

			}

		case "description":

			if description, ok := value.(string); ok {

				policy.Description = description

			}

		}

	}

	policy.UpdatedAt = time.Now()

	pm.mu.Unlock()

	// Call SMO API to update policy.

	url := fmt.Sprintf("%s/api/%s/policies/%s", pm.smoClient.baseURL, "v1", policyID)

	if err := pm.smoClient.put(ctx, url, policy, nil); err != nil {

		return fmt.Errorf("failed to update policy via SMO: %w", err)

	}

	logger.Info("A1 policy updated successfully", "policyID", policyID)

	return nil

}

// DeletePolicy deletes an A1 policy.

func (pm *PolicyManager) DeletePolicy(ctx context.Context, policyID string) error {

	logger := log.FromContext(ctx)

	logger.Info("deleting A1 policy", "policyID", policyID)

	pm.mu.Lock()

	if _, exists := pm.policies[policyID]; !exists {

		pm.mu.Unlock()

		return fmt.Errorf("policy %s not found", policyID)

	}

	delete(pm.policies, policyID)

	pm.mu.Unlock()

	// Call SMO API to delete policy.

	url := fmt.Sprintf("%s/api/%s/policies/%s", pm.smoClient.baseURL, "v1", policyID)

	if err := pm.smoClient.delete(ctx, url); err != nil {

		return fmt.Errorf("failed to delete policy via SMO: %w", err)

	}

	logger.Info("A1 policy deleted successfully", "policyID", policyID)

	return nil

}

// GetPolicy retrieves an A1 policy.

func (pm *PolicyManager) GetPolicy(policyID string) (*A1Policy, error) {

	pm.mu.RLock()

	defer pm.mu.RUnlock()

	policy, exists := pm.policies[policyID]

	if !exists {

		return nil, fmt.Errorf("policy %s not found", policyID)

	}

	return policy, nil

}

// ListPolicies lists all A1 policies.

func (pm *PolicyManager) ListPolicies() []*A1Policy {

	pm.mu.RLock()

	defer pm.mu.RUnlock()

	policies := make([]*A1Policy, 0, len(pm.policies))

	for _, policy := range pm.policies {

		policies = append(policies, policy)

	}

	return policies

}

// SubscribeToPolicyEvents subscribes to policy events.

func (pm *PolicyManager) SubscribeToPolicyEvents(ctx context.Context, policyID string, events []string, callback func(*PolicyEvent)) (*PolicySubscription, error) {

	subscription := &PolicySubscription{

		ID: fmt.Sprintf("sub-%s-%d", policyID, time.Now().Unix()),

		PolicyID: policyID,

		Subscriber: "nephoran-intent-operator",

		Events: events,

		Callback: callback,

		CreatedAt: time.Now(),
	}

	pm.mu.Lock()

	pm.subscriptions[subscription.ID] = subscription

	pm.eventCallbacks[subscription.ID] = callback

	pm.mu.Unlock()

	return subscription, nil

}

// startEventMonitoring starts policy event monitoring.

func (pm *PolicyManager) startEventMonitoring(ctx context.Context) {

	ticker := time.NewTicker(10 * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			// Poll for policy events (in a real implementation, this would use WebSocket or SSE).

			pm.pollPolicyEvents(ctx)

		}

	}

}

// pollPolicyEvents polls for policy events.

func (pm *PolicyManager) pollPolicyEvents(ctx context.Context) {

	// This is a simplified implementation.

	// In a real SMO integration, this would subscribe to actual events.

	url := fmt.Sprintf("%s/api/%s/events", pm.smoClient.baseURL, "v1")

	var events []*PolicyEvent

	if err := pm.smoClient.get(ctx, url, &events); err != nil {

		return // Silently continue on error

	}

	for _, event := range events {

		pm.mu.RLock()

		for _, subscription := range pm.subscriptions {

			if subscription.PolicyID == event.PolicyID {

				for _, eventType := range subscription.Events {

					if eventType == event.Type || eventType == "*" {

						if callback := pm.eventCallbacks[subscription.ID]; callback != nil {

							go callback(event)

						}

						break

					}

				}

			}

		}

		pm.mu.RUnlock()

	}

}

// Service Registry Methods.

// RegisterService registers a service with SMO.

func (sr *ServiceRegistry) RegisterService(ctx context.Context, service *ServiceInstance) error {

	logger := log.FromContext(ctx)

	logger.Info("registering service with SMO", "serviceID", service.ID, "name", service.Name)

	service.RegisteredAt = time.Now()

	service.LastHeartbeat = time.Now()

	service.Status = "REGISTERED"

	// Call SMO API to register service.

	url := fmt.Sprintf("%s/api/%s/services", sr.smoClient.baseURL, "v1")

	if err := sr.smoClient.post(ctx, url, service, nil); err != nil {

		return fmt.Errorf("failed to register service with SMO: %w", err)

	}

	sr.mu.Lock()

	sr.registeredServices[service.ID] = service

	sr.mu.Unlock()

	// Start health checking if enabled.

	if service.HealthCheck != nil && service.HealthCheck.Enabled {

		sr.startServiceHealthCheck(service)

	}

	logger.Info("service registered successfully", "serviceID", service.ID)

	return nil

}

// DiscoverServices discovers services from SMO.

func (sr *ServiceRegistry) DiscoverServices(ctx context.Context, serviceType string) ([]*ServiceInstance, error) {

	url := fmt.Sprintf("%s/api/%s/services", sr.smoClient.baseURL, "v1")

	if serviceType != "" {

		url = fmt.Sprintf("%s?type=%s", url, serviceType)

	}

	var services []*ServiceInstance

	if err := sr.smoClient.get(ctx, url, &services); err != nil {

		return nil, fmt.Errorf("failed to discover services: %w", err)

	}

	sr.mu.Lock()

	for _, service := range services {

		sr.discoveredServices[service.ID] = service

	}

	sr.mu.Unlock()

	return services, nil

}

// startServiceHealthCheck starts health checking for a service.

func (sr *ServiceRegistry) startServiceHealthCheck(service *ServiceInstance) {

	checker := &ServiceHealthChecker{

		serviceID: service.ID,

		config: service.HealthCheck,

		status: "UNKNOWN",

		lastCheck: time.Now(),

		stopCh: make(chan struct{}),
	}

	sr.mu.Lock()

	sr.healthCheckers[service.ID] = checker

	sr.mu.Unlock()

	go checker.start()

}

// startHealthChecking starts the health checking routine.

func (sr *ServiceRegistry) startHealthChecking(ctx context.Context) {

	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			// Update heartbeats for registered services.

			sr.updateServiceHeartbeats(ctx)

		}

	}

}

// updateServiceHeartbeats updates service heartbeats.

func (sr *ServiceRegistry) updateServiceHeartbeats(ctx context.Context) {

	sr.mu.Lock()

	defer sr.mu.Unlock()

	for serviceID, service := range sr.registeredServices {

		// Send heartbeat to SMO.

		url := fmt.Sprintf("%s/api/%s/services/%s/heartbeat", sr.smoClient.baseURL, "v1", serviceID)

		if err := sr.smoClient.post(ctx, url, map[string]interface{}{"timestamp": time.Now()}, nil); err == nil {

			service.LastHeartbeat = time.Now()

			service.Status = "ACTIVE"

		} else {

			service.Status = "INACTIVE"

		}

	}

}

// start starts the health checker.

func (hc *ServiceHealthChecker) start() {

	ticker := time.NewTicker(hc.config.Interval)

	defer ticker.Stop()

	for {

		select {

		case <-hc.stopCh:

			return

		case <-ticker.C:

			hc.performHealthCheck()

		}

	}

}

// performHealthCheck performs a health check.

func (hc *ServiceHealthChecker) performHealthCheck() {

	// This would implement actual health checking logic.

	// For now, we'll simulate a successful check.

	hc.lastCheck = time.Now()

	hc.status = "HEALTHY"

}

// Service Orchestrator Methods.

// DeployRApp deploys an rApp.

func (so *ServiceOrchestrator) DeployRApp(ctx context.Context, rApp *RAppInstance) error {

	logger := log.FromContext(ctx)

	logger.Info("deploying rApp", "rAppID", rApp.ID, "name", rApp.Name)

	rApp.CreatedAt = time.Now()

	rApp.Status = "DEPLOYING"

	// Execute pre-start lifecycle hooks.

	if rApp.Lifecycle != nil {

		for _, hook := range rApp.Lifecycle.PreStart {

			if err := so.executeLifecycleHook(ctx, &hook, rApp); err != nil {

				logger.Error(err, "pre-start hook failed", "hook", hook.Name)

				rApp.Status = "FAILED"

				return fmt.Errorf("pre-start hook failed: %w", err)

			}

		}

	}

	// Call SMO API to deploy rApp.

	url := fmt.Sprintf("%s/api/%s/rapps", so.smoClient.baseURL, "v1")

	if err := so.smoClient.post(ctx, url, rApp, nil); err != nil {

		rApp.Status = "FAILED"

		return fmt.Errorf("failed to deploy rApp via SMO: %w", err)

	}

	so.mu.Lock()

	so.rApps[rApp.ID] = rApp

	so.mu.Unlock()

	rApp.Status = "RUNNING"

	now := time.Now()

	rApp.StartedAt = &now

	// Execute post-start lifecycle hooks.

	if rApp.Lifecycle != nil {

		for _, hook := range rApp.Lifecycle.PostStart {

			if err := so.executeLifecycleHook(ctx, &hook, rApp); err != nil {

				logger.Error(err, "post-start hook failed", "hook", hook.Name)

			}

		}

	}

	logger.Info("rApp deployed successfully", "rAppID", rApp.ID)

	return nil

}

// executeLifecycleHook executes a lifecycle hook.

func (so *ServiceOrchestrator) executeLifecycleHook(ctx context.Context, hook *LifecycleHook, rApp *RAppInstance) error {

	logger := log.FromContext(ctx)

	logger.Info("executing lifecycle hook", "hook", hook.Name, "type", hook.Type)

	switch hook.Type {

	case "COMMAND":

		// Execute command (simplified implementation).

		return nil

	case "HTTP":

		// Make HTTP call.

		if endpoint, ok := hook.Params["endpoint"].(string); ok {

			req, err := http.NewRequestWithContext(ctx, "POST", endpoint, http.NoBody)

			if err != nil {

				return err

			}

			client := &http.Client{Timeout: hook.Timeout}

			resp, err := client.Do(req)

			if err != nil {

				return err

			}

			defer resp.Body.Close()

			if resp.StatusCode >= 400 {

				return fmt.Errorf("hook HTTP call failed with status: %d", resp.StatusCode)

			}

		}

		return nil

	case "SCRIPT":

		// Execute script (not implemented for security reasons).

		return fmt.Errorf("script hooks not supported")

	default:

		return fmt.Errorf("unsupported hook type: %s", hook.Type)

	}

}

// SMOClient HTTP methods.

func (c *SMOClient) get(ctx context.Context, url string, result interface{}) error {

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)

	if err != nil {

		return err

	}

	return c.doRequest(req, result)

}

func (c *SMOClient) post(ctx context.Context, url string, body, result interface{}) error {

	var reqBody []byte

	if body != nil {

		var err error

		reqBody, err = json.Marshal(body)

		if err != nil {

			return err

		}

	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, http.NoBody)

	if err != nil {

		return err

	}

	if reqBody != nil {

		req.Header.Set("Content-Type", "application/json")

	}

	return c.doRequest(req, result)

}

func (c *SMOClient) put(ctx context.Context, url string, body, result interface{}) error {

	reqBody, err := json.Marshal(body)

	if err != nil {

		return err

	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(reqBody))

	if err != nil {

		return err

	}

	req.Header.Set("Content-Type", "application/json")

	return c.doRequest(req, result)

}

func (c *SMOClient) delete(ctx context.Context, url string) error {

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, http.NoBody)

	if err != nil {

		return err

	}

	return c.doRequest(req, nil)

}

func (c *SMOClient) doRequest(req *http.Request, result interface{}) error {

	// Add authentication.

	if c.auth != nil {

		switch c.auth.Type {

		case "basic":

			req.SetBasicAuth(c.auth.Username, c.auth.Password)

		case "bearer":

			req.Header.Set("Authorization", "Bearer "+c.auth.Token)

		}

	}

	// Add custom headers.

	for key, value := range c.headers {

		req.Header.Set(key, value)

	}

	resp, err := c.httpClient.Do(req)

	if err != nil {

		return err

	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {

		return fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)

	}

	if result != nil {

		return json.NewDecoder(resp.Body).Decode(result)

	}

	return nil

}
