// Package a1 provides implementation for Near-RT RIC A1 interface.

// for policy management, service orchestration, and event-driven workflows.

package a1

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

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
	"github.com/nephio-project/nephoran-intent-operator/pkg/llm"
	"github.com/nephio-project/nephoran-intent-operator/pkg/oran"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// A1PolicyType represents an A1 policy type.

type A1PolicyType struct {
	PolicyTypeID int `json:"policy_type_id"`

	Name string `json:"name"`

	Description string `json:"description"`

	PolicySchema map[string]interface{} `json:"policy_schema"`

	CreateSchema map[string]interface{} `json:"create_schema"`
}

// A1PolicyInstance represents an A1 policy instance.

type A1PolicyInstance struct {
	PolicyInstanceID string `json:"policy_instance_id"`

	PolicyTypeID int `json:"policy_type_id"`

	PolicyData map[string]interface{} `json:"policy_data"`

	Status A1PolicyStatus `json:"status"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`
}

// A1PolicyStatus represents the status of an A1 policy.

type A1PolicyStatus struct {
	EnforcementStatus string `json:"enforcement_status"` // NOT_ENFORCED, ENFORCED

	EnforcementReason string `json:"enforcement_reason,omitempty"`

	LastModified time.Time `json:"last_modified"`
}

// A1AdaptorInterface defines the interface for A1 operations.

type A1AdaptorInterface interface {

	// Policy Type Management.

	CreatePolicyType(ctx context.Context, policyType *A1PolicyType) error

	GetPolicyType(ctx context.Context, policyTypeID int) (*A1PolicyType, error)

	ListPolicyTypes(ctx context.Context) ([]*A1PolicyType, error)

	DeletePolicyType(ctx context.Context, policyTypeID int) error

	// Policy Instance Management.

	CreatePolicyInstance(ctx context.Context, policyTypeID int, instance *A1PolicyInstance) error

	GetPolicyInstance(ctx context.Context, policyTypeID int, instanceID string) (*A1PolicyInstance, error)

	ListPolicyInstances(ctx context.Context, policyTypeID int) ([]*A1PolicyInstance, error)

	UpdatePolicyInstance(ctx context.Context, policyTypeID int, instanceID string, instance *A1PolicyInstance) error

	DeletePolicyInstance(ctx context.Context, policyTypeID int, instanceID string) error

	// Policy Status.

	GetPolicyStatus(ctx context.Context, policyTypeID int, instanceID string) (*A1PolicyStatus, error)

	// High-level operations.

	ApplyPolicy(ctx context.Context, me *nephoranv1.ManagedElement) error

	RemovePolicy(ctx context.Context, me *nephoranv1.ManagedElement) error
}

// A1Adaptor implements the A1 interface for Near-RT RIC communication.

type A1Adaptor struct {
	httpClient *http.Client

	ricURL string

	apiVersion string

	timeout time.Duration

	// Resilience components.

	circuitBreaker *llm.CircuitBreaker

	retryConfig *RetryConfig

	// Policy management.

	policyTypes map[int]*A1PolicyType

	policyInstances map[string]*A1PolicyInstance

	mutex sync.RWMutex
}

// RetryConfig holds retry configuration for A1 interface.

type RetryConfig struct {
	MaxRetries int `json:"max_retries"`

	InitialDelay time.Duration `json:"initial_delay"`

	MaxDelay time.Duration `json:"max_delay"`

	BackoffFactor float64 `json:"backoff_factor"`

	Jitter bool `json:"jitter"`

	RetryableErrors []string `json:"retryable_errors"`
}

// A1AdaptorConfig holds configuration for the A1 adaptor.

type A1AdaptorConfig struct {
	RICURL string

	APIVersion string

	Timeout time.Duration

	TLSConfig *oran.TLSConfig

	CircuitBreakerConfig *llm.CircuitBreakerConfig

	RetryConfig *RetryConfig
}

// SMOServiceRegistry represents the SMO service registry integration.

type SMOServiceRegistry struct {
	URL string

	APIKey string

	httpClient *http.Client
}

// SMOPolicyOrchestrator manages cross-domain policy coordination.

type SMOPolicyOrchestrator struct {
	registry *SMOServiceRegistry

	a1Adaptors map[string]*A1Adaptor

	eventQueue chan *PolicyEvent

	workflows map[string]*PolicyWorkflow
}

// PolicyEvent represents a policy-related event.

type PolicyEvent struct {
	ID string `json:"id"`

	Type string `json:"type"` // CREATE, UPDATE, DELETE, ENFORCE

	PolicyID string `json:"policy_id"`

	Source string `json:"source"`

	Target string `json:"target"`

	Data map[string]interface{} `json:"data"`

	Timestamp time.Time `json:"timestamp"`

	Status string `json:"status"`
}

// PolicyWorkflow represents a multi-step policy orchestration workflow.

type PolicyWorkflow struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Steps []*PolicyWorkflowStep `json:"steps"`

	CurrentStep int `json:"current_step"`

	Status string `json:"status"` // PENDING, RUNNING, COMPLETED, FAILED

	Context map[string]interface{} `json:"context"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`
}

// PolicyWorkflowStep represents a single step in a policy workflow.

type PolicyWorkflowStep struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Type string `json:"type"` // POLICY_CREATE, POLICY_UPDATE, VALIDATION, NOTIFICATION

	Target string `json:"target"`

	Parameters map[string]interface{} `json:"parameters"`

	Conditions []string `json:"conditions"`

	OnSuccess string `json:"on_success"`

	OnFailure string `json:"on_failure"`

	Status string `json:"status"`

	ExecutedAt *time.Time `json:"executed_at,omitempty"`
}

// ServiceInfo represents information about registered services.

type ServiceInfo struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Type string `json:"type"` // RIC, xApp, rApp

	Version string `json:"version"`

	Status string `json:"status"`

	Endpoints []ServiceEndpoint `json:"endpoints"`

	Capabilities []string `json:"capabilities"`

	Metadata map[string]string `json:"metadata"`

	RegisteredAt time.Time `json:"registered_at"`
}

// ServiceEndpoint represents a service endpoint.

type ServiceEndpoint struct {
	Name string `json:"name"`

	URL string `json:"url"`

	Protocol string `json:"protocol"`

	Version string `json:"version"`
}

// NewA1Adaptor creates a new A1 adaptor with the given configuration.

func NewA1Adaptor(config *A1AdaptorConfig) (*A1Adaptor, error) {

	if config == nil {

		config = &A1AdaptorConfig{

			RICURL: "http://near-rt-ric:8080",

			APIVersion: "v1",

			Timeout: 30 * time.Second,
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

	circuitBreaker := llm.NewCircuitBreaker("a1-adaptor", config.CircuitBreakerConfig)

	return &A1Adaptor{

		httpClient: httpClient,

		ricURL: config.RICURL,

		apiVersion: config.APIVersion,

		timeout: config.Timeout,

		circuitBreaker: circuitBreaker,

		retryConfig: config.RetryConfig,

		policyTypes: make(map[int]*A1PolicyType),

		policyInstances: make(map[string]*A1PolicyInstance),
	}, nil

}

// CreatePolicyType creates a new policy type in the Near-RT RIC.

func (a *A1Adaptor) CreatePolicyType(ctx context.Context, policyType *A1PolicyType) error {

	logger := log.FromContext(ctx)

	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d", a.ricURL, a.apiVersion, policyType.PolicyTypeID)

	body, err := json.Marshal(policyType)

	if err != nil {

		return fmt.Errorf("failed to marshal policy type: %w", err)

	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {

		bodyBytes, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("failed to create policy type: status=%d, body=%s", resp.StatusCode, string(bodyBytes))

	}

	logger.Info("successfully created policy type", "policyTypeID", policyType.PolicyTypeID)

	return nil

}

// GetPolicyType retrieves a policy type from the Near-RT RIC.

func (a *A1Adaptor) GetPolicyType(ctx context.Context, policyTypeID int) (*A1PolicyType, error) {

	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d", a.ricURL, a.apiVersion, policyTypeID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	resp, err := a.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("failed to get policy type: status=%d", resp.StatusCode)

	}

	var policyType A1PolicyType

	if err := json.NewDecoder(resp.Body).Decode(&policyType); err != nil {

		return nil, fmt.Errorf("failed to decode response: %w", err)

	}

	return &policyType, nil

}

// ListPolicyTypes lists all policy types in the Near-RT RIC.

func (a *A1Adaptor) ListPolicyTypes(ctx context.Context) ([]*A1PolicyType, error) {

	url := fmt.Sprintf("%s/a1-p/%s/policytypes", a.ricURL, a.apiVersion)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	resp, err := a.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("failed to list policy types: status=%d", resp.StatusCode)

	}

	var policyTypeIDs []int

	if err := json.NewDecoder(resp.Body).Decode(&policyTypeIDs); err != nil {

		return nil, fmt.Errorf("failed to decode response: %w", err)

	}

	// Fetch details for each policy type.

	var policyTypes []*A1PolicyType

	for _, id := range policyTypeIDs {

		policyType, err := a.GetPolicyType(ctx, id)

		if err != nil {

			return nil, fmt.Errorf("failed to get policy type %d: %w", id, err)

		}

		policyTypes = append(policyTypes, policyType)

	}

	return policyTypes, nil

}

// DeletePolicyType deletes a policy type from the Near-RT RIC.

func (a *A1Adaptor) DeletePolicyType(ctx context.Context, policyTypeID int) error {

	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d", a.ricURL, a.apiVersion, policyTypeID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	resp, err := a.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {

		return fmt.Errorf("failed to delete policy type: status=%d", resp.StatusCode)

	}

	return nil

}

// CreatePolicyInstance creates a new policy instance.

func (a *A1Adaptor) CreatePolicyInstance(ctx context.Context, policyTypeID int, instance *A1PolicyInstance) error {

	logger := log.FromContext(ctx)

	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d/policies/%s",

		a.ricURL, a.apiVersion, policyTypeID, instance.PolicyInstanceID)

	body, err := json.Marshal(instance.PolicyData)

	if err != nil {

		return fmt.Errorf("failed to marshal policy data: %w", err)

	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {

		bodyBytes, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("failed to create policy instance: status=%d, body=%s", resp.StatusCode, string(bodyBytes))

	}

	logger.Info("successfully created policy instance",

		"policyTypeID", policyTypeID,

		"instanceID", instance.PolicyInstanceID)

	return nil

}

// GetPolicyInstance retrieves a policy instance.

func (a *A1Adaptor) GetPolicyInstance(ctx context.Context, policyTypeID int, instanceID string) (*A1PolicyInstance, error) {

	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d/policies/%s",

		a.ricURL, a.apiVersion, policyTypeID, instanceID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	resp, err := a.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("failed to get policy instance: status=%d", resp.StatusCode)

	}

	var policyData map[string]interface{}

	if err := json.NewDecoder(resp.Body).Decode(&policyData); err != nil {

		return nil, fmt.Errorf("failed to decode response: %w", err)

	}

	// Get status separately.

	status, err := a.GetPolicyStatus(ctx, policyTypeID, instanceID)

	if err != nil {

		return nil, fmt.Errorf("failed to get policy status: %w", err)

	}

	return &A1PolicyInstance{

		PolicyInstanceID: instanceID,

		PolicyTypeID: policyTypeID,

		PolicyData: policyData,

		Status: *status,
	}, nil

}

// ListPolicyInstances lists all policy instances for a policy type.

func (a *A1Adaptor) ListPolicyInstances(ctx context.Context, policyTypeID int) ([]*A1PolicyInstance, error) {

	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d/policies", a.ricURL, a.apiVersion, policyTypeID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	resp, err := a.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("failed to list policy instances: status=%d", resp.StatusCode)

	}

	var instanceIDs []string

	if err := json.NewDecoder(resp.Body).Decode(&instanceIDs); err != nil {

		return nil, fmt.Errorf("failed to decode response: %w", err)

	}

	// Fetch details for each instance.

	var instances []*A1PolicyInstance

	for _, id := range instanceIDs {

		instance, err := a.GetPolicyInstance(ctx, policyTypeID, id)

		if err != nil {

			return nil, fmt.Errorf("failed to get policy instance %s: %w", id, err)

		}

		instances = append(instances, instance)

	}

	return instances, nil

}

// UpdatePolicyInstance updates an existing policy instance.

func (a *A1Adaptor) UpdatePolicyInstance(ctx context.Context, policyTypeID int, instanceID string, instance *A1PolicyInstance) error {

	// Same as create in A1 API.

	return a.CreatePolicyInstance(ctx, policyTypeID, instance)

}

// DeletePolicyInstance deletes a policy instance.

func (a *A1Adaptor) DeletePolicyInstance(ctx context.Context, policyTypeID int, instanceID string) error {

	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d/policies/%s",

		a.ricURL, a.apiVersion, policyTypeID, instanceID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	resp, err := a.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {

		return fmt.Errorf("failed to delete policy instance: status=%d", resp.StatusCode)

	}

	return nil

}

// GetPolicyStatus retrieves the status of a policy instance.

func (a *A1Adaptor) GetPolicyStatus(ctx context.Context, policyTypeID int, instanceID string) (*A1PolicyStatus, error) {

	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d/policies/%s/status",

		a.ricURL, a.apiVersion, policyTypeID, instanceID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	resp, err := a.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("failed to get policy status: status=%d", resp.StatusCode)

	}

	var status A1PolicyStatus

	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {

		return nil, fmt.Errorf("failed to decode response: %w", err)

	}

	return &status, nil

}

// ApplyPolicy applies an A1 policy from a ManagedElement.

func (a *A1Adaptor) ApplyPolicy(ctx context.Context, me *nephoranv1.ManagedElement) error {

	logger := log.FromContext(ctx)

	logger.Info("applying A1 policy", "managedElement", me.Name)

	if me.Spec.A1Policy.Raw == nil {

		logger.Info("no A1 policy to apply", "managedElement", me.Name)

		return nil

	}

	// Parse the A1 policy.

	var policySpec map[string]interface{}

	if err := json.Unmarshal(me.Spec.A1Policy.Raw, &policySpec); err != nil {

		return fmt.Errorf("failed to unmarshal A1 policy: %w", err)

	}

	// Extract policy type ID and instance ID.

	policyTypeID, ok := policySpec["policy_type_id"].(float64)

	if !ok {

		return fmt.Errorf("policy_type_id not found or invalid in A1 policy")

	}

	instanceID, ok := policySpec["policy_instance_id"].(string)

	if !ok {

		instanceID = fmt.Sprintf("%s-policy-%d", me.Name, int(policyTypeID))

	}

	// Extract policy data.

	policyData, ok := policySpec["policy_data"].(map[string]interface{})

	if !ok {

		return fmt.Errorf("policy_data not found in A1 policy")

	}

	// Create policy instance.

	instance := &A1PolicyInstance{

		PolicyInstanceID: instanceID,

		PolicyTypeID: int(policyTypeID),

		PolicyData: policyData,

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}

	// Apply the policy.

	if err := a.CreatePolicyInstance(ctx, int(policyTypeID), instance); err != nil {

		return fmt.Errorf("failed to create policy instance: %w", err)

	}

	// Wait for policy to be enforced (with timeout).

	enforced := false

	timeout := time.After(30 * time.Second)

	ticker := time.NewTicker(2 * time.Second)

	defer ticker.Stop()

	for !enforced {

		select {

		case <-timeout:

			return fmt.Errorf("timeout waiting for policy to be enforced")

		case <-ticker.C:

			status, err := a.GetPolicyStatus(ctx, int(policyTypeID), instanceID)

			if err != nil {

				logger.Error(err, "failed to get policy status")

				continue

			}

			if status.EnforcementStatus == "ENFORCED" {

				enforced = true

				logger.Info("policy successfully enforced",

					"policyTypeID", int(policyTypeID),

					"instanceID", instanceID)

			}

		}

	}

	return nil

}

// RemovePolicy removes an A1 policy from a ManagedElement.

func (a *A1Adaptor) RemovePolicy(ctx context.Context, me *nephoranv1.ManagedElement) error {

	logger := log.FromContext(ctx)

	logger.Info("removing A1 policy", "managedElement", me.Name)

	if me.Spec.A1Policy.Raw == nil {

		logger.Info("no A1 policy to remove", "managedElement", me.Name)

		return nil

	}

	// Parse the A1 policy to get IDs.

	var policySpec map[string]interface{}

	if err := json.Unmarshal(me.Spec.A1Policy.Raw, &policySpec); err != nil {

		return fmt.Errorf("failed to unmarshal A1 policy: %w", err)

	}

	policyTypeID, ok := policySpec["policy_type_id"].(float64)

	if !ok {

		return fmt.Errorf("policy_type_id not found in A1 policy")

	}

	instanceID, ok := policySpec["policy_instance_id"].(string)

	if !ok {

		instanceID = fmt.Sprintf("%s-policy-%d", me.Name, int(policyTypeID))

	}

	// Delete the policy instance.

	if err := a.DeletePolicyInstance(ctx, int(policyTypeID), instanceID); err != nil {

		return fmt.Errorf("failed to delete policy instance: %w", err)

	}

	logger.Info("successfully removed policy",

		"policyTypeID", int(policyTypeID),

		"instanceID", instanceID)

	return nil

}

// Helper function to create common policy types.

// CreateQoSPolicyType creates a QoS policy type for network slicing.

func CreateQoSPolicyType() *A1PolicyType {

	return &A1PolicyType{

		PolicyTypeID: 1000,

		Name: "Network Slice QoS Policy",

		Description: "Policy for managing QoS parameters in network slices",

		PolicySchema: map[string]interface{}{

			"$schema": "http://json-schema.org/draft-07/schema#",

			"type": "object",

			"properties": map[string]interface{}{

				"slice_id": map[string]interface{}{

					"type": "string",
				},

				"qos_parameters": map[string]interface{}{

					"type": "object",

					"properties": map[string]interface{}{

						"latency_ms": map[string]interface{}{

							"type": "number",
						},

						"throughput_mbps": map[string]interface{}{

							"type": "number",
						},

						"reliability": map[string]interface{}{

							"type": "number",

							"minimum": 0,

							"maximum": 1,
						},
					},
				},
			},

			"required": []string{"slice_id", "qos_parameters"},
		},
	}

}

// CreateTrafficSteeringPolicyType creates a traffic steering policy type.

func CreateTrafficSteeringPolicyType() *A1PolicyType {

	return &A1PolicyType{

		PolicyTypeID: 2000,

		Name: "Traffic Steering Policy",

		Description: "Policy for steering traffic between cells or network functions",

		PolicySchema: map[string]interface{}{

			"$schema": "http://json-schema.org/draft-07/schema#",

			"type": "object",

			"properties": map[string]interface{}{

				"ue_id": map[string]interface{}{

					"type": "string",
				},

				"target_cell": map[string]interface{}{

					"type": "string",
				},

				"traffic_percentage": map[string]interface{}{

					"type": "number",

					"minimum": 0,

					"maximum": 100,
				},
			},

			"required": []string{"target_cell", "traffic_percentage"},
		},
	}

}

// NewSMOServiceRegistry creates a new SMO service registry client.

func NewSMOServiceRegistry(url, apiKey string) *SMOServiceRegistry {

	return &SMOServiceRegistry{

		URL: url,

		APIKey: apiKey,

		httpClient: &http.Client{

			Timeout: 30 * time.Second,
		},
	}

}

// RegisterService registers a service with the SMO service registry.

func (r *SMOServiceRegistry) RegisterService(ctx context.Context, service *ServiceInfo) error {

	logger := log.FromContext(ctx)

	url := fmt.Sprintf("%s/services", r.URL)

	body, err := json.Marshal(service)

	if err != nil {

		return fmt.Errorf("failed to marshal service info: %w", err)

	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("X-API-Key", r.APIKey)

	resp, err := r.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {

		bodyBytes, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("failed to register service: status=%d, body=%s", resp.StatusCode, string(bodyBytes))

	}

	logger.Info("service registered with SMO", "serviceID", service.ID, "name", service.Name)

	return nil

}

// DiscoverServices discovers available services from the SMO service registry.

func (r *SMOServiceRegistry) DiscoverServices(ctx context.Context, serviceType string) ([]*ServiceInfo, error) {

	url := fmt.Sprintf("%s/services", r.URL)

	if serviceType != "" {

		url += fmt.Sprintf("?type=%s", serviceType)

	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("X-API-Key", r.APIKey)

	resp, err := r.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("failed to discover services: status=%d", resp.StatusCode)

	}

	var services []*ServiceInfo

	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {

		return nil, fmt.Errorf("failed to decode response: %w", err)

	}

	return services, nil

}

// NotifyPolicyEvent sends a policy event notification to the SMO.

func (r *SMOServiceRegistry) NotifyPolicyEvent(ctx context.Context, event *PolicyEvent) error {

	url := fmt.Sprintf("%s/events", r.URL)

	body, err := json.Marshal(event)

	if err != nil {

		return fmt.Errorf("failed to marshal event: %w", err)

	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("X-API-Key", r.APIKey)

	resp, err := r.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {

		return fmt.Errorf("failed to notify policy event: status=%d", resp.StatusCode)

	}

	return nil

}

// NewSMOPolicyOrchestrator creates a new SMO policy orchestrator.

func NewSMOPolicyOrchestrator(registry *SMOServiceRegistry) *SMOPolicyOrchestrator {

	return &SMOPolicyOrchestrator{

		registry: registry,

		a1Adaptors: make(map[string]*A1Adaptor),

		eventQueue: make(chan *PolicyEvent, 100),

		workflows: make(map[string]*PolicyWorkflow),
	}

}

// RegisterA1Adaptor registers an A1 adaptor with the orchestrator.

func (o *SMOPolicyOrchestrator) RegisterA1Adaptor(ricID string, adaptor *A1Adaptor) {

	o.a1Adaptors[ricID] = adaptor

}

// StartEventProcessor starts the event processing loop.

func (o *SMOPolicyOrchestrator) StartEventProcessor(ctx context.Context) {

	logger := log.FromContext(ctx)

	logger.Info("starting SMO policy orchestrator event processor")

	go func() {

		for {

			select {

			case <-ctx.Done():

				logger.Info("stopping event processor")

				return

			case event := <-o.eventQueue:

				if err := o.processEvent(ctx, event); err != nil {

					logger.Error(err, "failed to process policy event", "eventID", event.ID)

				}

			}

		}

	}()

}

// processEvent processes a single policy event.

func (o *SMOPolicyOrchestrator) processEvent(ctx context.Context, event *PolicyEvent) error {

	logger := log.FromContext(ctx)

	logger.Info("processing policy event", "eventID", event.ID, "type", event.Type)

	switch event.Type {

	case "CREATE":

		return o.handlePolicyCreate(ctx, event)

	case "UPDATE":

		return o.handlePolicyUpdate(ctx, event)

	case "DELETE":

		return o.handlePolicyDelete(ctx, event)

	case "ENFORCE":

		return o.handlePolicyEnforce(ctx, event)

	default:

		return fmt.Errorf("unknown event type: %s", event.Type)

	}

}

// handlePolicyCreate handles policy creation events.

func (o *SMOPolicyOrchestrator) handlePolicyCreate(ctx context.Context, event *PolicyEvent) error {

	logger := log.FromContext(ctx)

	// Extract policy information from event data.

	policyTypeID, ok := event.Data["policy_type_id"].(float64)

	if !ok {

		return fmt.Errorf("policy_type_id not found in event data")

	}

	instanceID, ok := event.Data["policy_instance_id"].(string)

	if !ok {

		return fmt.Errorf("policy_instance_id not found in event data")

	}

	policyData, ok := event.Data["policy_data"].(map[string]interface{})

	if !ok {

		return fmt.Errorf("policy_data not found in event data")

	}

	// Create policy instance.

	instance := &A1PolicyInstance{

		PolicyInstanceID: instanceID,

		PolicyTypeID: int(policyTypeID),

		PolicyData: policyData,

		CreatedAt: time.Now(),

		UpdatedAt: time.Now(),
	}

	// Apply policy to target RIC.

	adaptor, ok := o.a1Adaptors[event.Target]

	if !ok {

		return fmt.Errorf("A1 adaptor not found for RIC: %s", event.Target)

	}

	if err := adaptor.CreatePolicyInstance(ctx, int(policyTypeID), instance); err != nil {

		// Notify SMO of failure.

		failureEvent := &PolicyEvent{

			ID: fmt.Sprintf("%s-failure", event.ID),

			Type: "FAILURE",

			PolicyID: event.PolicyID,

			Source: event.Target,

			Target: event.Source,

			Data: map[string]interface{}{

				"error": err.Error(),

				"original_event": event.ID,
			},

			Timestamp: time.Now(),

			Status: "FAILED",
		}

		o.registry.NotifyPolicyEvent(ctx, failureEvent)

		return fmt.Errorf("failed to create policy instance: %w", err)

	}

	// Notify SMO of success.

	successEvent := &PolicyEvent{

		ID: fmt.Sprintf("%s-success", event.ID),

		Type: "SUCCESS",

		PolicyID: event.PolicyID,

		Source: event.Target,

		Target: event.Source,

		Data: map[string]interface{}{

			"policy_instance_id": instanceID,

			"policy_type_id": int(policyTypeID),

			"original_event": event.ID,
		},

		Timestamp: time.Now(),

		Status: "COMPLETED",
	}

	o.registry.NotifyPolicyEvent(ctx, successEvent)

	logger.Info("policy creation completed", "eventID", event.ID, "policyID", event.PolicyID)

	return nil

}

// handlePolicyUpdate handles policy update events.

func (o *SMOPolicyOrchestrator) handlePolicyUpdate(ctx context.Context, event *PolicyEvent) error {

	// Similar to handlePolicyCreate but updates existing policy.

	return o.handlePolicyCreate(ctx, event) // Reuse create logic for simplicity

}

// handlePolicyDelete handles policy deletion events.

func (o *SMOPolicyOrchestrator) handlePolicyDelete(ctx context.Context, event *PolicyEvent) error {

	logger := log.FromContext(ctx)

	policyTypeID, ok := event.Data["policy_type_id"].(float64)

	if !ok {

		return fmt.Errorf("policy_type_id not found in event data")

	}

	instanceID, ok := event.Data["policy_instance_id"].(string)

	if !ok {

		return fmt.Errorf("policy_instance_id not found in event data")

	}

	// Delete policy from target RIC.

	adaptor, ok := o.a1Adaptors[event.Target]

	if !ok {

		return fmt.Errorf("A1 adaptor not found for RIC: %s", event.Target)

	}

	if err := adaptor.DeletePolicyInstance(ctx, int(policyTypeID), instanceID); err != nil {

		return fmt.Errorf("failed to delete policy instance: %w", err)

	}

	logger.Info("policy deletion completed", "eventID", event.ID, "policyID", event.PolicyID)

	return nil

}

// handlePolicyEnforce handles policy enforcement events.

func (o *SMOPolicyOrchestrator) handlePolicyEnforce(ctx context.Context, event *PolicyEvent) error {

	logger := log.FromContext(ctx)

	policyTypeID, ok := event.Data["policy_type_id"].(float64)

	if !ok {

		return fmt.Errorf("policy_type_id not found in event data")

	}

	instanceID, ok := event.Data["policy_instance_id"].(string)

	if !ok {

		return fmt.Errorf("policy_instance_id not found in event data")

	}

	// Check policy status.

	adaptor, ok := o.a1Adaptors[event.Target]

	if !ok {

		return fmt.Errorf("A1 adaptor not found for RIC: %s", event.Target)

	}

	status, err := adaptor.GetPolicyStatus(ctx, int(policyTypeID), instanceID)

	if err != nil {

		return fmt.Errorf("failed to get policy status: %w", err)

	}

	// Notify SMO of enforcement status.

	statusEvent := &PolicyEvent{

		ID: fmt.Sprintf("%s-status", event.ID),

		Type: "STATUS",

		PolicyID: event.PolicyID,

		Source: event.Target,

		Target: event.Source,

		Data: map[string]interface{}{

			"enforcement_status": status.EnforcementStatus,

			"enforcement_reason": status.EnforcementReason,

			"last_modified": status.LastModified,
		},

		Timestamp: time.Now(),

		Status: "COMPLETED",
	}

	o.registry.NotifyPolicyEvent(ctx, statusEvent)

	logger.Info("policy enforcement check completed", "eventID", event.ID, "status", status.EnforcementStatus)

	return nil

}

// CreatePolicyWorkflow creates a new policy workflow.

func (o *SMOPolicyOrchestrator) CreatePolicyWorkflow(ctx context.Context, workflow *PolicyWorkflow) error {

	logger := log.FromContext(ctx)

	workflow.ID = fmt.Sprintf("workflow-%d", time.Now().UnixNano())

	workflow.Status = "PENDING"

	workflow.CurrentStep = 0

	workflow.CreatedAt = time.Now()

	workflow.UpdatedAt = time.Now()

	if workflow.Context == nil {

		workflow.Context = make(map[string]interface{})

	}

	o.workflows[workflow.ID] = workflow

	// Start workflow execution.

	go o.executeWorkflow(context.Background(), workflow.ID)

	logger.Info("policy workflow created", "workflowID", workflow.ID, "name", workflow.Name)

	return nil

}

// executeWorkflow executes a policy workflow.

func (o *SMOPolicyOrchestrator) executeWorkflow(ctx context.Context, workflowID string) {

	logger := log.FromContext(ctx)

	workflow, ok := o.workflows[workflowID]

	if !ok {

		logger.Error(fmt.Errorf("workflow not found"), "workflowID", workflowID)

		return

	}

	workflow.Status = "RUNNING"

	workflow.UpdatedAt = time.Now()

	for i, step := range workflow.Steps {

		if i < workflow.CurrentStep {

			continue // Skip already completed steps

		}

		workflow.CurrentStep = i

		logger.Info("executing workflow step", "workflowID", workflowID, "step", i, "stepName", step.Name)

		if err := o.executeWorkflowStep(ctx, workflow, step); err != nil {

			logger.Error(err, "workflow step failed", "workflowID", workflowID, "step", i)

			workflow.Status = "FAILED"

			workflow.UpdatedAt = time.Now()

			return

		}

		step.Status = "COMPLETED"

		step.ExecutedAt = &[]time.Time{time.Now()}[0]

		workflow.UpdatedAt = time.Now()

	}

	workflow.Status = "COMPLETED"

	workflow.UpdatedAt = time.Now()

	logger.Info("workflow completed", "workflowID", workflowID)

}

// executeWorkflowStep executes a single workflow step.

func (o *SMOPolicyOrchestrator) executeWorkflowStep(ctx context.Context, workflow *PolicyWorkflow, step *PolicyWorkflowStep) error {

	switch step.Type {

	case "POLICY_CREATE":

		return o.executeCreatePolicyStep(ctx, workflow, step)

	case "POLICY_UPDATE":

		return o.executeUpdatePolicyStep(ctx, workflow, step)

	case "VALIDATION":

		return o.executeValidationStep(ctx, workflow, step)

	case "NOTIFICATION":

		return o.executeNotificationStep(ctx, workflow, step)

	default:

		return fmt.Errorf("unknown step type: %s", step.Type)

	}

}

// executeCreatePolicyStep executes a policy creation step.

func (o *SMOPolicyOrchestrator) executeCreatePolicyStep(ctx context.Context, workflow *PolicyWorkflow, step *PolicyWorkflowStep) error {

	event := &PolicyEvent{

		ID: fmt.Sprintf("%s-step-%s", workflow.ID, step.ID),

		Type: "CREATE",

		PolicyID: fmt.Sprintf("%s-policy", workflow.ID),

		Source: "orchestrator",

		Target: step.Target,

		Data: step.Parameters,

		Timestamp: time.Now(),

		Status: "PENDING",
	}

	select {

	case o.eventQueue <- event:

		return nil

	case <-time.After(5 * time.Second):

		return fmt.Errorf("timeout queuing policy event")

	}

}

// executeUpdatePolicyStep executes a policy update step.

func (o *SMOPolicyOrchestrator) executeUpdatePolicyStep(ctx context.Context, workflow *PolicyWorkflow, step *PolicyWorkflowStep) error {

	// Similar to create step but with UPDATE type.

	event := &PolicyEvent{

		ID: fmt.Sprintf("%s-step-%s", workflow.ID, step.ID),

		Type: "UPDATE",

		PolicyID: fmt.Sprintf("%s-policy", workflow.ID),

		Source: "orchestrator",

		Target: step.Target,

		Data: step.Parameters,

		Timestamp: time.Now(),

		Status: "PENDING",
	}

	select {

	case o.eventQueue <- event:

		return nil

	case <-time.After(5 * time.Second):

		return fmt.Errorf("timeout queuing policy event")

	}

}

// executeValidationStep executes a validation step.

func (o *SMOPolicyOrchestrator) executeValidationStep(ctx context.Context, workflow *PolicyWorkflow, step *PolicyWorkflowStep) error {

	// Perform validation logic based on step parameters.

	validationType, ok := step.Parameters["validation_type"].(string)

	if !ok {

		return fmt.Errorf("validation_type not specified in step parameters")

	}

	switch validationType {

	case "policy_enforcement":

		// Check if policies are enforced.

		return o.validatePolicyEnforcement(ctx, workflow, step)

	case "resource_availability":

		// Check if resources are available.

		return o.validateResourceAvailability(ctx, workflow, step)

	default:

		return fmt.Errorf("unknown validation type: %s", validationType)

	}

}

// executeNotificationStep executes a notification step.

func (o *SMOPolicyOrchestrator) executeNotificationStep(ctx context.Context, workflow *PolicyWorkflow, step *PolicyWorkflowStep) error {

	// Send notification based on step parameters.

	notificationType, ok := step.Parameters["notification_type"].(string)

	if !ok {

		return fmt.Errorf("notification_type not specified in step parameters")

	}

	message, ok := step.Parameters["message"].(string)

	if !ok {

		message = fmt.Sprintf("Workflow %s step %s completed", workflow.ID, step.Name)

	}

	switch notificationType {

	case "smo_event":

		event := &PolicyEvent{

			ID: fmt.Sprintf("%s-notification-%s", workflow.ID, step.ID),

			Type: "NOTIFICATION",

			PolicyID: fmt.Sprintf("%s-policy", workflow.ID),

			Source: "orchestrator",

			Target: "smo",

			Data: map[string]interface{}{

				"workflow_id": workflow.ID,

				"step_id": step.ID,

				"message": message,
			},

			Timestamp: time.Now(),

			Status: "PENDING",
		}

		return o.registry.NotifyPolicyEvent(ctx, event)

	default:

		return fmt.Errorf("unknown notification type: %s", notificationType)

	}

}

// validatePolicyEnforcement validates that policies are properly enforced.

func (o *SMOPolicyOrchestrator) validatePolicyEnforcement(ctx context.Context, workflow *PolicyWorkflow, step *PolicyWorkflowStep) error {

	ricID, ok := step.Parameters["ric_id"].(string)

	if !ok {

		return fmt.Errorf("ric_id not specified in validation parameters")

	}

	adaptor, ok := o.a1Adaptors[ricID]

	if !ok {

		return fmt.Errorf("A1 adaptor not found for RIC: %s", ricID)

	}

	// List all policy instances and check their status.

	policyTypes, err := adaptor.ListPolicyTypes(ctx)

	if err != nil {

		return fmt.Errorf("failed to list policy types: %w", err)

	}

	for _, policyType := range policyTypes {

		instances, err := adaptor.ListPolicyInstances(ctx, policyType.PolicyTypeID)

		if err != nil {

			continue // Skip if unable to list instances

		}

		for _, instance := range instances {

			if instance.Status.EnforcementStatus != "ENFORCED" {

				return fmt.Errorf("policy instance %s not enforced", instance.PolicyInstanceID)

			}

		}

	}

	return nil

}

// validateResourceAvailability validates that resources are available.

func (o *SMOPolicyOrchestrator) validateResourceAvailability(ctx context.Context, workflow *PolicyWorkflow, step *PolicyWorkflowStep) error {

	// This would integrate with cloud management APIs to check resource availability.

	// For now, we'll do a simple validation.

	requiredCPU, ok := step.Parameters["required_cpu"].(float64)

	if !ok {

		requiredCPU = 0

	}

	requiredMemory, ok := step.Parameters["required_memory"].(float64)

	if !ok {

		requiredMemory = 0

	}

	// In a real implementation, this would check actual resource availability.

	// For now, we'll assume resources are available if requirements are reasonable.

	if requiredCPU > 1000 || requiredMemory > 16384 { // 1000 CPU cores or 16GB memory

		return fmt.Errorf("insufficient resources available")

	}

	return nil

}

// Retry and Circuit Breaker Helper Methods for A1 Adaptor.

// executeWithRetry executes an operation with exponential backoff retry.

func (a *A1Adaptor) executeWithRetry(ctx context.Context, operation func() error) error {

	_, err := a.circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {

		var lastErr error

		for attempt := 0; attempt <= a.retryConfig.MaxRetries; attempt++ {

			if attempt > 0 {

				delay := a.calculateBackoffDelay(attempt)

				select {

				case <-ctx.Done():

					return nil, ctx.Err()

				case <-time.After(delay):

				}

			}

			if err := operation(); err != nil {

				lastErr = err

				if !a.isRetryableError(err) {

					return nil, err

				}

				continue

			}

			return nil, nil

		}

		return nil, fmt.Errorf("operation failed after %d attempts: %w", a.retryConfig.MaxRetries+1, lastErr)

	})

	return err

}

// calculateBackoffDelay calculates the delay for exponential backoff with jitter.

func (a *A1Adaptor) calculateBackoffDelay(attempt int) time.Duration {

	delay := time.Duration(float64(a.retryConfig.InitialDelay) * math.Pow(a.retryConfig.BackoffFactor, float64(attempt-1)))

	if delay > a.retryConfig.MaxDelay {

		delay = a.retryConfig.MaxDelay

	}

	if a.retryConfig.Jitter {

		jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)

		delay += jitter

	}

	return delay

}

// isRetryableError checks if an error is retryable based on configuration.

func (a *A1Adaptor) isRetryableError(err error) bool {

	if err == nil {

		return false

	}

	errMsg := err.Error()

	for _, retryableErr := range a.retryConfig.RetryableErrors {

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

// GetCircuitBreakerStats returns circuit breaker statistics.

func (a *A1Adaptor) GetCircuitBreakerStats() map[string]interface{} {

	return a.circuitBreaker.GetStats()

}

// ResetCircuitBreaker manually resets the circuit breaker.

func (a *A1Adaptor) ResetCircuitBreaker() {

	a.circuitBreaker.Reset()

}

// Enhanced Policy Management Methods with Circuit Breaker Protection.

// createPolicyTypeWithRetry creates a policy type with retry and circuit breaker protection.

func (a *A1Adaptor) createPolicyTypeWithRetry(ctx context.Context, policyType *A1PolicyType) error {

	return a.executeWithRetry(ctx, func() error {

		jsonData, err := json.Marshal(policyType)

		if err != nil {

			return fmt.Errorf("failed to marshal policy type: %w", err)

		}

		url := fmt.Sprintf("%s/a1-p/policytypes/%d", a.ricURL, policyType.PolicyTypeID)

		req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))

		if err != nil {

			return fmt.Errorf("failed to create HTTP request: %w", err)

		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := a.httpClient.Do(req)

		if err != nil {

			return fmt.Errorf("HTTP request failed: %w", err)

		}

		defer resp.Body.Close()

		if resp.StatusCode >= 400 {

			return fmt.Errorf("failed to create policy type, status: %d", resp.StatusCode)

		}

		// Cache the policy type locally.

		a.mutex.Lock()

		a.policyTypes[policyType.PolicyTypeID] = policyType

		a.mutex.Unlock()

		return nil

	})

}

// createPolicyInstanceWithRetry creates a policy instance with retry and circuit breaker protection.

func (a *A1Adaptor) createPolicyInstanceWithRetry(ctx context.Context, policyTypeID int, policyInstanceID string, policyData map[string]interface{}) error {

	return a.executeWithRetry(ctx, func() error {

		jsonData, err := json.Marshal(policyData)

		if err != nil {

			return fmt.Errorf("failed to marshal policy data: %w", err)

		}

		url := fmt.Sprintf("%s/a1-p/policytypes/%d/policies/%s", a.ricURL, policyTypeID, policyInstanceID)

		req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))

		if err != nil {

			return fmt.Errorf("failed to create HTTP request: %w", err)

		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := a.httpClient.Do(req)

		if err != nil {

			return fmt.Errorf("HTTP request failed: %w", err)

		}

		defer resp.Body.Close()

		if resp.StatusCode >= 400 {

			return fmt.Errorf("failed to create policy instance, status: %d", resp.StatusCode)

		}

		// Cache the policy instance locally.

		policyInstance := &A1PolicyInstance{

			PolicyInstanceID: policyInstanceID,

			PolicyTypeID: policyTypeID,

			PolicyData: policyData,

			Status: A1PolicyStatus{

				EnforcementStatus: "ENFORCED",

				LastModified: time.Now(),
			},

			CreatedAt: time.Now(),

			UpdatedAt: time.Now(),
		}

		a.mutex.Lock()

		a.policyInstances[policyInstanceID] = policyInstance

		a.mutex.Unlock()

		return nil

	})

}
