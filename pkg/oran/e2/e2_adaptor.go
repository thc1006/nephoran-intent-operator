// Package e2 provides a comprehensive implementation of the E2 interface.

// for Near-RT RIC communication, following O-RAN specifications for.

// service model management, node configuration, and control operations.

package e2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran"
)

// E2NodeFunction represents RAN function exposed by an E2 Node.

type E2NodeFunction struct {
	FunctionID int `json:"function_id"`

	FunctionDefinition string `json:"function_definition"`

	FunctionRevision int `json:"function_revision"`

	FunctionOID string `json:"function_oid"`

	FunctionDescription string `json:"function_description"`

	ServiceModel E2ServiceModel `json:"service_model"`

	Status E2NodeFunctionStatus `json:"status"`
}

// E2ServiceModel represents E2 Service Model information.

type E2ServiceModel struct {
	ServiceModelID string `json:"service_model_id"`

	ServiceModelName string `json:"service_model_name"`

	ServiceModelVersion string `json:"service_model_version"`

	ServiceModelOID string `json:"service_model_oid"`

	SupportedProcedures []string `json:"supported_procedures"`

	Configuration json.RawMessage `json:"configuration,omitempty"`
}

// E2NodeFunctionStatus represents the status of an E2 Node Function.

type E2NodeFunctionStatus struct {
	State string `json:"state"` // IDLE, ACTIVE, BUSY, UNAVAILABLE

	LastHeartbeat time.Time `json:"last_heartbeat"`

	SubscriptionCount int `json:"subscription_count"`

	ErrorCount int `json:"error_count"`

	LastError string `json:"last_error,omitempty"`
}

// E2Subscription represents an E2 subscription for monitoring/control.

type E2Subscription struct {
	SubscriptionID string `json:"subscription_id"`

	RequestorID string `json:"requestor_id"`

	RanFunctionID int `json:"ran_function_id"`

	EventTriggers []E2EventTrigger `json:"event_triggers"`

	Actions []E2Action `json:"actions"`

	ReportingPeriod time.Duration `json:"reporting_period"`

	Status E2SubscriptionStatus `json:"status"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`
}

// E2EventTrigger represents event triggers for E2 subscriptions.

type E2EventTrigger struct {
	TriggerType string `json:"trigger_type"` // PERIODIC, UPON_CHANGE, UPON_RCV_MEAS_REPORT

	ReportingPeriod time.Duration `json:"reporting_period,omitempty"`

	Conditions json.RawMessage `json:"conditions,omitempty"`
}

// E2Action represents actions to be performed on subscription events.

type E2Action struct {
	ActionID int `json:"action_id"`

	ActionType string `json:"action_type"` // REPORT, INSERT, POLICY, CONTROL

	ActionDefinition json.RawMessage `json:"action_definition"`

	SubsequentAction string `json:"subsequent_action,omitempty"`
}

// E2SubscriptionStatus represents the status of an E2 subscription.

type E2SubscriptionStatus struct {
	State string `json:"state"` // ACTIVE, INACTIVE, FAILED

	ResponseCode int `json:"response_code"`

	CauseCode string `json:"cause_code,omitempty"`

	LastUpdate time.Time `json:"last_update"`

	MessagesReceived int64 `json:"messages_received"`

	LastMessageTime time.Time `json:"last_message_time"`
}

// E2ControlRequest represents a control request sent to an E2 Node.

type E2ControlRequest struct {
	RequestID string `json:"request_id"`

	RanFunctionID int `json:"ran_function_id"`

	CallProcessID string `json:"call_process_id,omitempty"`

	ControlHeader json.RawMessage `json:"control_header"`

	ControlMessage json.RawMessage `json:"control_message"`

	ControlAckRequest bool `json:"control_ack_request"`
}

// E2ControlResponse represents the response to an E2 control request.

type E2ControlResponse struct {
	ResponseID string `json:"response_id"`

	RequestID string `json:"request_id"`

	RanFunctionID int `json:"ran_function_id"`

	CallProcessID string `json:"call_process_id,omitempty"`

	ControlOutcome json.RawMessage `json:"control_outcome,omitempty"`

	Status E2ControlStatus `json:"status"`

	Timestamp time.Time `json:"timestamp"`
}

// E2ControlStatus represents the status of control operation.

type E2ControlStatus struct {
	Result string `json:"result"` // SUCCESS, FAILURE

	CauseCode string `json:"cause_code,omitempty"`

	CauseDescription string `json:"cause_description,omitempty"`
}

// E2AdaptorInterface defines the interface for E2 operations following O-RAN specifications.

type E2AdaptorInterface interface {
	// E2 Node Management (based on O-RAN.WG3.E2GAP specifications).

	RegisterE2Node(ctx context.Context, nodeID string, functions []*E2NodeFunction) error

	DeregisterE2Node(ctx context.Context, nodeID string) error

	GetE2Node(ctx context.Context, nodeID string) (*E2NodeInfo, error)

	ListE2Nodes(ctx context.Context) ([]*E2NodeInfo, error)

	UpdateE2Node(ctx context.Context, nodeID string, functions []*E2NodeFunction) error

	// E2 Service Model Management (based on O-RAN.WG3.E2SM specifications).

	GetServiceModel(ctx context.Context, serviceModelID string) (*E2ServiceModel, error)

	ListServiceModels(ctx context.Context) ([]*E2ServiceModel, error)

	ValidateServiceModel(ctx context.Context, serviceModel *E2ServiceModel) error

	// E2 Subscription Management (based on O-RAN.WG3.E2AP specifications).

	CreateSubscription(ctx context.Context, nodeID string, subscription *E2Subscription) error

	GetSubscription(ctx context.Context, nodeID, subscriptionID string) (*E2Subscription, error)

	ListSubscriptions(ctx context.Context, nodeID string) ([]*E2Subscription, error)

	UpdateSubscription(ctx context.Context, nodeID, subscriptionID string, subscription *E2Subscription) error

	DeleteSubscription(ctx context.Context, nodeID, subscriptionID string) error

	// E2 Control Operations (based on O-RAN.WG3.E2AP specifications).

	SendControlRequest(ctx context.Context, nodeID string, request *E2ControlRequest) (*E2ControlResponse, error)

	// E2 Indication and Report Handling.

	GetIndicationData(ctx context.Context, nodeID, subscriptionID string) ([]*E2Indication, error)

	// High-level ManagedElement integration.

	ConfigureE2Interface(ctx context.Context, me *nephoranv1.ManagedElement) error

	RemoveE2Interface(ctx context.Context, me *nephoranv1.ManagedElement) error
}

// E2ConnectionState represents simple connection states.

type E2ConnectionState string

const (

	// E2ConnectionStateConnected holds e2connectionstateconnected value.

	E2ConnectionStateConnected E2ConnectionState = "CONNECTED"

	// E2ConnectionStateDisconnected holds e2connectionstatedisconnected value.

	E2ConnectionStateDisconnected E2ConnectionState = "DISCONNECTED"

	// E2ConnectionStateConnecting holds e2connectionstateconnecting value.

	E2ConnectionStateConnecting E2ConnectionState = "CONNECTING"

	// E2ConnectionStateFailed holds e2connectionstatefailed value.

	E2ConnectionStateFailed E2ConnectionState = "FAILED"
)

// E2NodeInfo represents information about an E2 Node (now uses GlobalE2NodeID from e2ap_messages.go).

type E2NodeInfo struct {
	NodeID string `json:"node_id"`

	GlobalE2NodeID GlobalE2NodeID `json:"global_e2_node_id"`

	RANFunctions []*E2NodeFunction `json:"ran_functions"`

	ConnectionStatus E2ConnectionStatus `json:"connection_status"`

	SubscriptionCount int `json:"subscription_count"`

	LastSeen time.Time `json:"last_seen"`

	Configuration json.RawMessage `json:"configuration,omitempty"`
}

// E2ConnectionStatus represents the connection status of an E2 Node.

type E2ConnectionStatus struct {
	State string `json:"state"` // CONNECTED, DISCONNECTED, CONNECTING

	EstablishedAt time.Time `json:"established_at"`

	LastHeartbeat time.Time `json:"last_heartbeat"`

	ConnectionFailures int `json:"connection_failures"`

	LastFailureReason string `json:"last_failure_reason,omitempty"`
}

// E2Indication represents indication messages from E2 Nodes.

type E2Indication struct {
	IndicationID string `json:"indication_id"`

	SubscriptionID string `json:"subscription_id"`

	RanFunctionID int `json:"ran_function_id"`

	IndicationHeader json.RawMessage `json:"indication_header"`

	IndicationMessage json.RawMessage `json:"indication_message"`

	CallProcessID string `json:"call_process_id,omitempty"`

	Timestamp time.Time `json:"timestamp"`
}

// E2Adaptor implements the E2 interface for Near-RT RIC communication.

// following O-RAN.WG3.E2GAP and O-RAN.WG3.E2AP specifications.

type E2Adaptor struct {
	httpClient *http.Client

	ricURL string

	apiVersion string

	timeout time.Duration

	nodeRegistry map[string]*E2NodeInfo

	subscriptions map[string]map[string]*E2Subscription // nodeID -> subscriptionID -> subscription

	mutex sync.RWMutex

	heartbeatInterval time.Duration

	maxRetries int

	// Circuit breaker and resilience.

	circuitBreaker *llm.CircuitBreaker

	retryConfig *RetryConfig

	encoder *E2APEncoder
}

// E2AdaptorConfig holds configuration for the E2 adaptor.

// RetryConfig holds retry configuration.

type RetryConfig struct {
	MaxRetries int `json:"max_retries"`

	InitialDelay time.Duration `json:"initial_delay"`

	MaxDelay time.Duration `json:"max_delay"`

	BackoffFactor float64 `json:"backoff_factor"`

	Jitter bool `json:"jitter"`

	RetryableErrors []string `json:"retryable_errors"`
}

// E2AdaptorConfig represents a e2adaptorconfig.

type E2AdaptorConfig struct {
	RICURL string

	APIVersion string

	Timeout time.Duration

	HeartbeatInterval time.Duration

	MaxRetries int

	TLSConfig *oran.TLSConfig

	CircuitBreakerConfig *llm.CircuitBreakerConfig

	RetryConfig *RetryConfig
}

// NewE2Adaptor creates a new E2 adaptor following O-RAN specifications.

func NewE2Adaptor(config *E2AdaptorConfig) (*E2Adaptor, error) {
	if config == nil {
		config = &E2AdaptorConfig{
			RICURL: "http://near-rt-ric:38080",

			APIVersion: "v1",

			Timeout: 30 * time.Second,

			HeartbeatInterval: 30 * time.Second,

			MaxRetries: 3,
		}
	}

	// Set default retry configuration.

	if config.RetryConfig == nil {
		config.RetryConfig = &RetryConfig{
			MaxRetries: 3,

			InitialDelay: 1 * time.Second,

			MaxDelay: 30 * time.Second,

			BackoffFactor: 2.0,

			Jitter: true,

			RetryableErrors: []string{
				"connection refused",

				"timeout",

				"temporary failure",

				"service unavailable",
			},
		}
	}

	// Set default circuit breaker configuration.

	if config.CircuitBreakerConfig == nil {
		config.CircuitBreakerConfig = &llm.CircuitBreakerConfig{
			FailureThreshold: 5,

			FailureRate: 0.5,

			MinimumRequestCount: 10,

			Timeout: config.Timeout,

			HalfOpenTimeout: 60 * time.Second,

			SuccessThreshold: 3,

			HalfOpenMaxRequests: 5,

			ResetTimeout: 60 * time.Second,

			SlidingWindowSize: 100,

			EnableHealthCheck: true,

			HealthCheckInterval: 30 * time.Second,

			HealthCheckTimeout: 10 * time.Second,
		}
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	// Configure TLS if provided.

	if config.TLSConfig != nil {

		// Validate TLS configuration.

		if err := oran.ValidateTLSConfig(config.TLSConfig); err != nil {
			return nil, fmt.Errorf("invalid TLS configuration: %w", err)
		}

		// Build TLS configuration.

		tlsConfig, err := oran.BuildTLSConfig(config.TLSConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS configuration: %w", err)
		}

		// Create HTTP transport with TLS configuration.

		transport := &http.Transport{
			TLSClientConfig: tlsConfig,
		}

		httpClient.Transport = transport

	}

	// Create circuit breaker.

	circuitBreaker := llm.NewCircuitBreaker("e2-adaptor", config.CircuitBreakerConfig)

	// Create E2AP encoder.

	encoder := NewE2APEncoder()

	adaptor := &E2Adaptor{
		httpClient: httpClient,

		ricURL: config.RICURL,

		apiVersion: config.APIVersion,

		timeout: config.Timeout,

		nodeRegistry: make(map[string]*E2NodeInfo),

		subscriptions: make(map[string]map[string]*E2Subscription),

		heartbeatInterval: config.HeartbeatInterval,

		maxRetries: config.MaxRetries,

		circuitBreaker: circuitBreaker,

		retryConfig: config.RetryConfig,

		encoder: encoder,
	}

	// Start background heartbeat monitoring.

	go adaptor.startHeartbeatMonitor()

	return adaptor, nil
}

// RegisterE2Node registers an E2 Node with the Near-RT RIC.

func (e *E2Adaptor) RegisterE2Node(ctx context.Context, nodeID string, functions []*E2NodeFunction) error {
	logger := log.FromContext(ctx)

	url := fmt.Sprintf("%s/e2ap/%s/nodes/%s/register", e.ricURL, e.apiVersion, nodeID)

	payload := json.RawMessage("{}")

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal registration payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Accept", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send registration request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {

		bodyBytes, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("failed to register E2 node: status=%d, body=%s", resp.StatusCode, string(bodyBytes))

	}

	// Update local registry.

	e.mutex.Lock()

	defer e.mutex.Unlock()

	nodeInfo := &E2NodeInfo{
		NodeID: nodeID,

		GlobalE2NodeID: GlobalE2NodeID{
			PLMNIdentity: PLMNIdentity{
				MCC: "001", // Default, should be configurable

				MNC: "01", // Default, should be configurable

			},

			E2NodeID: E2NodeID{
				GNBID: &GNBID{
					GNBIDChoice: GNBIDChoice{
						GNBID32: &nodeID,
					},
				},
			},
		},

		RANFunctions: functions,

		ConnectionStatus: E2ConnectionStatus{
			State: "CONNECTED",

			EstablishedAt: time.Now(),

			LastHeartbeat: time.Now(),
		},

		LastSeen: time.Now(),
	}

	e.nodeRegistry[nodeID] = nodeInfo

	e.subscriptions[nodeID] = make(map[string]*E2Subscription)

	logger.Info("successfully registered E2 node", "nodeID", nodeID, "functions", len(functions))

	return nil
}

// DeregisterE2Node deregisters an E2 Node from the Near-RT RIC.

func (e *E2Adaptor) DeregisterE2Node(ctx context.Context, nodeID string) error {
	logger := log.FromContext(ctx)

	url := fmt.Sprintf("%s/e2ap/%s/nodes/%s/deregister", e.ricURL, e.apiVersion, nodeID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send deregistration request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to deregister E2 node: status=%d", resp.StatusCode)
	}

	// Remove from local registry.

	e.mutex.Lock()

	defer e.mutex.Unlock()

	delete(e.nodeRegistry, nodeID)

	delete(e.subscriptions, nodeID)

	logger.Info("successfully deregistered E2 node", "nodeID", nodeID)

	return nil
}

// GetE2Node retrieves information about an E2 Node.

func (e *E2Adaptor) GetE2Node(ctx context.Context, nodeID string) (*E2NodeInfo, error) {
	e.mutex.RLock()

	defer e.mutex.RUnlock()

	nodeInfo, exists := e.nodeRegistry[nodeID]

	if !exists {
		return nil, fmt.Errorf("E2 node not found: %s", nodeID)
	}

	// Create a copy to avoid race conditions.

	nodeInfoCopy := *nodeInfo

	return &nodeInfoCopy, nil
}

// ListE2Nodes lists all registered E2 Nodes.

func (e *E2Adaptor) ListE2Nodes(ctx context.Context) ([]*E2NodeInfo, error) {
	e.mutex.RLock()

	defer e.mutex.RUnlock()

	nodes := make([]*E2NodeInfo, 0, len(e.nodeRegistry))

	for _, nodeInfo := range e.nodeRegistry {

		// Create copy to avoid race conditions.

		nodeInfoCopy := *nodeInfo

		nodes = append(nodes, &nodeInfoCopy)

	}

	return nodes, nil
}

// UpdateE2Node updates an E2 Node's functions.

func (e *E2Adaptor) UpdateE2Node(ctx context.Context, nodeID string, functions []*E2NodeFunction) error {
	logger := log.FromContext(ctx)

	url := fmt.Sprintf("%s/e2ap/%s/nodes/%s/update", e.ricURL, e.apiVersion, nodeID)

	payload := json.RawMessage("{}")

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal update payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send update request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		bodyBytes, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("failed to update E2 node: status=%d, body=%s", resp.StatusCode, string(bodyBytes))

	}

	// Update local registry.

	e.mutex.Lock()

	defer e.mutex.Unlock()

	if nodeInfo, exists := e.nodeRegistry[nodeID]; exists {

		nodeInfo.RANFunctions = functions

		nodeInfo.LastSeen = time.Now()

	}

	logger.Info("successfully updated E2 node", "nodeID", nodeID, "functions", len(functions))

	return nil
}

// GetServiceModel retrieves information about a service model.

func (e *E2Adaptor) GetServiceModel(ctx context.Context, serviceModelID string) (*E2ServiceModel, error) {
	url := fmt.Sprintf("%s/e2ap/%s/service-models/%s", e.ricURL, e.apiVersion, serviceModelID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get service model: status=%d", resp.StatusCode)
	}

	var serviceModel E2ServiceModel

	if err := json.NewDecoder(resp.Body).Decode(&serviceModel); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &serviceModel, nil
}

// ListServiceModels lists all available service models.

func (e *E2Adaptor) ListServiceModels(ctx context.Context) ([]*E2ServiceModel, error) {
	url := fmt.Sprintf("%s/e2ap/%s/service-models", e.ricURL, e.apiVersion)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list service models: status=%d", resp.StatusCode)
	}

	var serviceModels []*E2ServiceModel

	if err := json.NewDecoder(resp.Body).Decode(&serviceModels); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return serviceModels, nil
}

// ValidateServiceModel validates a service model configuration.

func (e *E2Adaptor) ValidateServiceModel(ctx context.Context, serviceModel *E2ServiceModel) error {
	url := fmt.Sprintf("%s/e2ap/%s/service-models/validate", e.ricURL, e.apiVersion)

	body, err := json.Marshal(serviceModel)
	if err != nil {
		return fmt.Errorf("failed to marshal service model: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send validation request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		bodyBytes, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("service model validation failed: status=%d, body=%s", resp.StatusCode, string(bodyBytes))

	}

	return nil
}

// CreateSubscription creates a new E2 subscription.

func (e *E2Adaptor) CreateSubscription(ctx context.Context, nodeID string, subscription *E2Subscription) error {
	logger := log.FromContext(ctx)

	url := fmt.Sprintf("%s/e2ap/%s/nodes/%s/subscriptions", e.ricURL, e.apiVersion, nodeID)

	body, err := json.Marshal(subscription)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send subscription request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {

		bodyBytes, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("failed to create subscription: status=%d, body=%s", resp.StatusCode, string(bodyBytes))

	}

	// Update local registry.

	e.mutex.Lock()

	defer e.mutex.Unlock()

	if _, exists := e.subscriptions[nodeID]; !exists {
		e.subscriptions[nodeID] = make(map[string]*E2Subscription)
	}

	subscription.Status = E2SubscriptionStatus{
		State: "ACTIVE",

		LastUpdate: time.Now(),
	}

	subscription.CreatedAt = time.Now()

	subscription.UpdatedAt = time.Now()

	e.subscriptions[nodeID][subscription.SubscriptionID] = subscription

	logger.Info("successfully created E2 subscription",

		"nodeID", nodeID,

		"subscriptionID", subscription.SubscriptionID,

		"ranFunctionID", subscription.RanFunctionID)

	return nil
}

// GetSubscription retrieves a specific E2 subscription.

func (e *E2Adaptor) GetSubscription(ctx context.Context, nodeID, subscriptionID string) (*E2Subscription, error) {
	e.mutex.RLock()

	defer e.mutex.RUnlock()

	nodeSubscriptions, exists := e.subscriptions[nodeID]

	if !exists {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	subscription, exists := nodeSubscriptions[subscriptionID]

	if !exists {
		return nil, fmt.Errorf("subscription not found: %s", subscriptionID)
	}

	// Create copy to avoid race conditions.

	subscriptionCopy := *subscription

	return &subscriptionCopy, nil
}

// ListSubscriptions lists all subscriptions for a node.

func (e *E2Adaptor) ListSubscriptions(ctx context.Context, nodeID string) ([]*E2Subscription, error) {
	e.mutex.RLock()

	defer e.mutex.RUnlock()

	nodeSubscriptions, exists := e.subscriptions[nodeID]

	if !exists {
		return []*E2Subscription{}, nil
	}

	subscriptions := make([]*E2Subscription, 0, len(nodeSubscriptions))

	for _, subscription := range nodeSubscriptions {

		// Create copy to avoid race conditions.

		subscriptionCopy := *subscription

		subscriptions = append(subscriptions, &subscriptionCopy)

	}

	return subscriptions, nil
}

// UpdateSubscription updates an existing E2 subscription.

func (e *E2Adaptor) UpdateSubscription(ctx context.Context, nodeID, subscriptionID string, subscription *E2Subscription) error {
	logger := log.FromContext(ctx)

	url := fmt.Sprintf("%s/e2ap/%s/nodes/%s/subscriptions/%s", e.ricURL, e.apiVersion, nodeID, subscriptionID)

	body, err := json.Marshal(subscription)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send update request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		bodyBytes, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("failed to update subscription: status=%d, body=%s", resp.StatusCode, string(bodyBytes))

	}

	// Update local registry.

	e.mutex.Lock()

	defer e.mutex.Unlock()

	if nodeSubscriptions, exists := e.subscriptions[nodeID]; exists {

		subscription.UpdatedAt = time.Now()

		nodeSubscriptions[subscriptionID] = subscription

	}

	logger.Info("successfully updated E2 subscription", "nodeID", nodeID, "subscriptionID", subscriptionID)

	return nil
}

// DeleteSubscription deletes an E2 subscription.

func (e *E2Adaptor) DeleteSubscription(ctx context.Context, nodeID, subscriptionID string) error {
	logger := log.FromContext(ctx)

	url := fmt.Sprintf("%s/e2ap/%s/nodes/%s/subscriptions/%s", e.ricURL, e.apiVersion, nodeID, subscriptionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send delete request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete subscription: status=%d", resp.StatusCode)
	}

	// Remove from local registry.

	e.mutex.Lock()

	defer e.mutex.Unlock()

	if nodeSubscriptions, exists := e.subscriptions[nodeID]; exists {
		delete(nodeSubscriptions, subscriptionID)
	}

	logger.Info("successfully deleted E2 subscription", "nodeID", nodeID, "subscriptionID", subscriptionID)

	return nil
}

// SendControlRequest sends a control request to an E2 Node.

func (e *E2Adaptor) SendControlRequest(ctx context.Context, nodeID string, request *E2ControlRequest) (*E2ControlResponse, error) {
	logger := log.FromContext(ctx)

	url := fmt.Sprintf("%s/e2ap/%s/nodes/%s/control", e.ricURL, e.apiVersion, nodeID)

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal control request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Accept", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send control request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		bodyBytes, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("control request failed: status=%d, body=%s", resp.StatusCode, string(bodyBytes))

	}

	var controlResponse E2ControlResponse

	if err := json.NewDecoder(resp.Body).Decode(&controlResponse); err != nil {
		return nil, fmt.Errorf("failed to decode control response: %w", err)
	}

	logger.Info("successfully sent E2 control request",

		"nodeID", nodeID,

		"requestID", request.RequestID,

		"ranFunctionID", request.RanFunctionID)

	return &controlResponse, nil
}

// GetIndicationData retrieves indication data for a subscription.

func (e *E2Adaptor) GetIndicationData(ctx context.Context, nodeID, subscriptionID string) ([]*E2Indication, error) {
	url := fmt.Sprintf("%s/e2ap/%s/nodes/%s/subscriptions/%s/indications",

		e.ricURL, e.apiVersion, nodeID, subscriptionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get indication data: status=%d", resp.StatusCode)
	}

	var indications []*E2Indication

	if err := json.NewDecoder(resp.Body).Decode(&indications); err != nil {
		return nil, fmt.Errorf("failed to decode indications: %w", err)
	}

	return indications, nil
}

// ConfigureE2Interface configures the E2 interface for a ManagedElement.

func (e *E2Adaptor) ConfigureE2Interface(ctx context.Context, me *nephoranv1.ManagedElement) error {
	logger := log.FromContext(ctx)

	logger.Info("configuring E2 interface", "managedElement", me.ObjectMeta.Name)

	if me.Spec.Config == nil || len(me.Spec.Config) == 0 {

		logger.Info("no E2 configuration to apply", "managedElement", me.ObjectMeta.Name)

		return nil

	}

	// Parse E2 configuration.

	var e2Config map[string]interface{}

	// Convert config map to JSON for parsing
	configJSON, err := json.Marshal(me.Spec.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal E2 configuration: %w", err)
	}

	if err := json.Unmarshal(configJSON, &e2Config); err != nil {
		return fmt.Errorf("failed to unmarshal E2 configuration: %w", err)
	}

	// Extract node ID.

	nodeID, ok := e2Config["node_id"].(string)

	if !ok {
		nodeID = me.ObjectMeta.Name
	}

	// Extract RAN functions.

	ranFunctionsData, ok := e2Config["ran_functions"].([]interface{})

	if !ok {
		return fmt.Errorf("ran_functions not found in E2 configuration")
	}

	var ranFunctions []*E2NodeFunction

	for _, funcData := range ranFunctionsData {

		funcMap, ok := funcData.(map[string]interface{})

		if !ok {
			continue
		}

		function := &E2NodeFunction{
			FunctionID: func() int {
				f := funcMap["function_id"].(float64)
				if f < 0 || f > 2147483647 {
					panic("function_id out of int range")
				}
				return int(f)
			}(),

			FunctionDefinition: funcMap["function_definition"].(string),

			FunctionRevision: func() int {
				f := funcMap["function_revision"].(float64)
				if f < 0 || f > 2147483647 {
					panic("function_revision out of int range")
				}
				return int(f)
			}(),

			FunctionOID: funcMap["function_oid"].(string),

			FunctionDescription: funcMap["function_description"].(string),

			Status: E2NodeFunctionStatus{
				State: "ACTIVE",

				LastHeartbeat: time.Now(),
			},
		}

		// Parse service model if present.

		if smData, exists := funcMap["service_model"]; exists {

			smMap := smData.(map[string]interface{})

			function.ServiceModel = E2ServiceModel{
				ServiceModelID: smMap["service_model_id"].(string),

				ServiceModelName: smMap["service_model_name"].(string),

				ServiceModelVersion: smMap["service_model_version"].(string),

				ServiceModelOID: smMap["service_model_oid"].(string),
			}

			if procedures, exists := smMap["supported_procedures"]; exists {

				procList := procedures.([]interface{})

				supportedProcs := make([]string, len(procList))

				for i, proc := range procList {
					supportedProcs[i] = proc.(string)
				}

				function.ServiceModel.SupportedProcedures = supportedProcs

			}

		}

		ranFunctions = append(ranFunctions, function)

	}

	// Register the E2 node.

	if err := e.RegisterE2Node(ctx, nodeID, ranFunctions); err != nil {
		return fmt.Errorf("failed to register E2 node: %w", err)
	}

	// Create default subscriptions if specified.

	if subscriptionsData, exists := e2Config["default_subscriptions"]; exists {

		subscriptionsList := subscriptionsData.([]interface{})

		for _, subData := range subscriptionsList {

			subMap := subData.(map[string]interface{})

			subscription := &E2Subscription{
				SubscriptionID: subMap["subscription_id"].(string),

				RequestorID: subMap["requestor_id"].(string),

				RanFunctionID: func() int {
					f := subMap["ran_function_id"].(float64)
					if f < 0 || f > 2147483647 {
						panic("ran_function_id out of int range")
					}
					return int(f)
				}(),

				ReportingPeriod: func() time.Duration {
					f := subMap["reporting_period_ms"].(float64)
					if f < 0 || f > 2147483647 {
						panic("reporting_period_ms out of int range")
					}
					return time.Duration(int(f)) * time.Millisecond
				}(),
			}

			// Parse event triggers.

			if triggersData, exists := subMap["event_triggers"]; exists {

				triggersList := triggersData.([]interface{})

				for _, triggerData := range triggersList {

					triggerMap := triggerData.(map[string]interface{})

					trigger := E2EventTrigger{
						TriggerType: triggerMap["trigger_type"].(string),
					}

					if period, exists := triggerMap["reporting_period_ms"]; exists {

						f := period.(float64)
						if f < 0 || f > 2147483647 {
							panic("reporting period out of int range")
						}
						trigger.ReportingPeriod = time.Duration(int(f)) * time.Millisecond

					}

					subscription.EventTriggers = append(subscription.EventTriggers, trigger)

				}

			}

			// Parse actions.

			if actionsData, exists := subMap["actions"]; exists {

				actionsList := actionsData.([]interface{})

				for _, actionData := range actionsList {

					actionMap := actionData.(map[string]interface{})

					action := E2Action{
						ActionID: func() int {
							f := actionMap["action_id"].(float64)
							if f < 0 || f > 2147483647 {
								panic("action_id out of int range")
							}
							return int(f)
						}(),

						ActionType: actionMap["action_type"].(string),

						ActionDefinition: actionMap["action_definition"].(map[string]interface{}),
					}

					subscription.Actions = append(subscription.Actions, action)

				}

			}

			if err := e.CreateSubscription(ctx, nodeID, subscription); err != nil {
				logger.Error(err, "failed to create default subscription",

					"subscriptionID", subscription.SubscriptionID)

				// Continue with other subscriptions.
			}

		}

	}

	logger.Info("successfully configured E2 interface",

		"managedElement", me.ObjectMeta.Name,

		"nodeID", nodeID,

		"functions", len(ranFunctions))

	return nil
}

// RemoveE2Interface removes the E2 interface configuration for a ManagedElement.

func (e *E2Adaptor) RemoveE2Interface(ctx context.Context, me *nephoranv1.ManagedElement) error {
	logger := log.FromContext(ctx)

	logger.Info("removing E2 interface", "managedElement", me.ObjectMeta.Name)

	if me.Spec.Config == nil || len(me.Spec.Config) == 0 {

		logger.Info("no E2 configuration to remove", "managedElement", me.ObjectMeta.Name)

		return nil

	}

	// Parse E2 configuration to get node ID.

	var e2Config map[string]interface{}

	// Convert config map to JSON for parsing
	configJSON, err := json.Marshal(me.Spec.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal E2 configuration: %w", err)
	}

	if err := json.Unmarshal(configJSON, &e2Config); err != nil {
		return fmt.Errorf("failed to unmarshal E2 configuration: %w", err)
	}

	nodeID, ok := e2Config["node_id"].(string)

	if !ok {
		nodeID = me.ObjectMeta.Name
	}

	// Delete all subscriptions for this node.

	subscriptions, err := e.ListSubscriptions(ctx, nodeID)

	if err != nil {
		logger.Error(err, "failed to list subscriptions for cleanup", "nodeID", nodeID)
	} else {
		for _, subscription := range subscriptions {
			if err := e.DeleteSubscription(ctx, nodeID, subscription.SubscriptionID); err != nil {
				logger.Error(err, "failed to delete subscription",

					"nodeID", nodeID,

					"subscriptionID", subscription.SubscriptionID)
			}
		}
	}

	// Deregister the E2 node.

	if err := e.DeregisterE2Node(ctx, nodeID); err != nil {
		return fmt.Errorf("failed to deregister E2 node: %w", err)
	}

	logger.Info("successfully removed E2 interface",

		"managedElement", me.ObjectMeta.Name,

		"nodeID", nodeID)

	return nil
}

// startHeartbeatMonitor starts the background heartbeat monitoring.

func (e *E2Adaptor) startHeartbeatMonitor() {
	ticker := time.NewTicker(e.heartbeatInterval)

	defer ticker.Stop()

	for range ticker.C {

		e.mutex.Lock()

		now := time.Now()

		for _, nodeInfo := range e.nodeRegistry {
			// Check if node hasn't sent heartbeat in 2x the interval.

			if now.Sub(nodeInfo.ConnectionStatus.LastHeartbeat) > 2*e.heartbeatInterval {

				nodeInfo.ConnectionStatus.State = "DISCONNECTED"

				nodeInfo.ConnectionStatus.ConnectionFailures++

				// Mark all node functions as unavailable.

				for _, function := range nodeInfo.RANFunctions {
					function.Status.State = "UNAVAILABLE"
				}

			}
		}

		e.mutex.Unlock()

	}
}

// Helper functions for creating common E2 service models.

// CreateKPMServiceModel creates a Key Performance Measurement service model.

func CreateKPMServiceModel() *E2ServiceModel {
	return &E2ServiceModel{
		ServiceModelID: "1.3.6.1.4.1.53148.1.1.2.2",

		ServiceModelName: "KPM",

		ServiceModelVersion: "1.0",

		ServiceModelOID: "1.3.6.1.4.1.53148.1.1.2.2",

		SupportedProcedures: []string{
			"RIC_SUBSCRIPTION",

			"RIC_SUBSCRIPTION_DELETE",

			"RIC_INDICATION",
		},

		Configuration: json.RawMessage("{}"),

			"granularity_period": "1000ms",

			"collection_start_time": "2025-07-29T10:00:00Z",
		},
	}
}

// CreateRCServiceModel creates a RAN Control service model.

func CreateRCServiceModel() *E2ServiceModel {
	return &E2ServiceModel{
		ServiceModelID: "1.3.6.1.4.1.53148.1.1.2.3",

		ServiceModelName: "RC",

		ServiceModelVersion: "1.0",

		ServiceModelOID: "1.3.6.1.4.1.53148.1.1.2.3",

		SupportedProcedures: []string{
			"RIC_CONTROL_REQUEST",

			"RIC_CONTROL_ACKNOWLEDGE",

			"RIC_CONTROL_FAILURE",
		},

		Configuration: json.RawMessage("{}"),

			"control_outcomes": []string{
				"successful",

				"rejected",

				"failed",
			},
		},
	}
}

// CreateDefaultE2NodeFunction creates a default E2 Node function for gNB.

func CreateDefaultE2NodeFunction() *E2NodeFunction {
	return &E2NodeFunction{
		FunctionID: 1,

		FunctionDefinition: "gNB-DU",

		FunctionRevision: 1,

		FunctionOID: "1.3.6.1.4.1.53148.1.1.1.1",

		FunctionDescription: "gNB Distributed Unit",

		ServiceModel: *CreateKPMServiceModel(),

		Status: E2NodeFunctionStatus{
			State: "ACTIVE",

			LastHeartbeat: time.Now(),
		},
	}
}

// Retry and Circuit Breaker Helper Methods.

// executeWithRetry executes an operation with exponential backoff retry.

func (e *E2Adaptor) executeWithRetry(ctx context.Context, operation func() error) error {
	_, err := e.circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
		var lastErr error

		for attempt := 0; attempt <= e.retryConfig.MaxRetries; attempt++ {

			if attempt > 0 {

				delay := e.calculateBackoffDelay(attempt)

				select {

				case <-ctx.Done():

					return nil, ctx.Err()

				case <-time.After(delay):

				}

			}

			if err := operation(); err != nil {

				lastErr = err

				if !e.isRetryableError(err) {
					return nil, err
				}

				continue

			}

			return nil, nil

		}

		return nil, fmt.Errorf("operation failed after %d attempts: %w", e.retryConfig.MaxRetries+1, lastErr)
	})

	return err
}

// calculateBackoffDelay calculates the delay for exponential backoff with jitter.

func (e *E2Adaptor) calculateBackoffDelay(attempt int) time.Duration {
	delay := time.Duration(float64(e.retryConfig.InitialDelay) * math.Pow(e.retryConfig.BackoffFactor, float64(attempt-1)))

	if delay > e.retryConfig.MaxDelay {
		delay = e.retryConfig.MaxDelay
	}

	if e.retryConfig.Jitter {

		jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)

		delay += jitter

	}

	return delay
}

// isRetryableError checks if an error is retryable based on configuration.

func (e *E2Adaptor) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	for _, retryableErr := range e.retryConfig.RetryableErrors {
		if contains(errMsg, retryableErr) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive).

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&

		(s == substr ||

			len(s) > len(substr) &&

				(s[:len(substr)] == substr ||

					s[len(s)-len(substr):] == substr ||

					indexOf(s, substr) >= 0))
}

// indexOf returns the index of substr in s, or -1 if not found.

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}

// sendE2APMessage sends an E2AP message with circuit breaker and retry protection.

func (e *E2Adaptor) sendE2APMessage(ctx context.Context, nodeID string, message *E2APMessage) (*E2APMessage, error) {
	logger := log.FromContext(ctx)

	var response *E2APMessage

	err := e.executeWithRetry(ctx, func() error {
		// Encode the message.

		messageBytes, err := e.encoder.EncodeMessage(message)
		if err != nil {
			return fmt.Errorf("failed to encode E2AP message: %w", err)
		}

		// Create HTTP request.

		url := fmt.Sprintf("%s/api/%s/nodes/%s/messages", e.ricURL, e.apiVersion, nodeID)

		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(messageBytes))
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		req.Header.Set("Accept", "application/json")

		// Send request.

		resp, err := e.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP request failed: %w", err)
		}

		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			return fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
		}

		// Read response.

		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		// Decode response.

		if len(responseBody) > 0 {

			response, err = e.encoder.DecodeMessage(responseBody)
			if err != nil {
				return fmt.Errorf("failed to decode E2AP response: %w", err)
			}

		}

		logger.Info("E2AP message sent successfully",

			"nodeID", nodeID,

			"messageType", message.MessageType,

			"transactionID", message.TransactionID)

		return nil
	})
	if err != nil {

		logger.Error(err, "Failed to send E2AP message",

			"nodeID", nodeID,

			"messageType", message.MessageType)

		return nil, err

	}

	return response, nil
}

// GetCircuitBreakerStats returns circuit breaker statistics.

func (e *E2Adaptor) GetCircuitBreakerStats() map[string]interface{} {
	return e.circuitBreaker.GetStats()
}

// ResetCircuitBreaker manually resets the circuit breaker.

func (e *E2Adaptor) ResetCircuitBreaker() {
	e.circuitBreaker.Reset()
}
