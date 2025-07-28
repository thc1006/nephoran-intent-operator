package a1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// A1PolicyType represents an A1 policy type
type A1PolicyType struct {
	PolicyTypeID   int                    `json:"policy_type_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	PolicySchema   map[string]interface{} `json:"policy_schema"`
	CreateSchema   map[string]interface{} `json:"create_schema"`
}

// A1PolicyInstance represents an A1 policy instance
type A1PolicyInstance struct {
	PolicyInstanceID string                 `json:"policy_instance_id"`
	PolicyTypeID     int                    `json:"policy_type_id"`
	PolicyData       map[string]interface{} `json:"policy_data"`
	Status           A1PolicyStatus         `json:"status"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// A1PolicyStatus represents the status of an A1 policy
type A1PolicyStatus struct {
	EnforcementStatus     string    `json:"enforcement_status"` // NOT_ENFORCED, ENFORCED
	EnforcementReason     string    `json:"enforcement_reason,omitempty"`
	LastModified          time.Time `json:"last_modified"`
}

// A1AdaptorInterface defines the interface for A1 operations
type A1AdaptorInterface interface {
	// Policy Type Management
	CreatePolicyType(ctx context.Context, policyType *A1PolicyType) error
	GetPolicyType(ctx context.Context, policyTypeID int) (*A1PolicyType, error)
	ListPolicyTypes(ctx context.Context) ([]*A1PolicyType, error)
	DeletePolicyType(ctx context.Context, policyTypeID int) error
	
	// Policy Instance Management
	CreatePolicyInstance(ctx context.Context, policyTypeID int, instance *A1PolicyInstance) error
	GetPolicyInstance(ctx context.Context, policyTypeID int, instanceID string) (*A1PolicyInstance, error)
	ListPolicyInstances(ctx context.Context, policyTypeID int) ([]*A1PolicyInstance, error)
	UpdatePolicyInstance(ctx context.Context, policyTypeID int, instanceID string, instance *A1PolicyInstance) error
	DeletePolicyInstance(ctx context.Context, policyTypeID int, instanceID string) error
	
	// Policy Status
	GetPolicyStatus(ctx context.Context, policyTypeID int, instanceID string) (*A1PolicyStatus, error)
	
	// High-level operations
	ApplyPolicy(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error
	RemovePolicy(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error
}

// A1Adaptor implements the A1 interface for Near-RT RIC communication
type A1Adaptor struct {
	httpClient *http.Client
	ricURL     string
	apiVersion string
	timeout    time.Duration
}

// A1AdaptorConfig holds configuration for the A1 adaptor
type A1AdaptorConfig struct {
	RICURL     string
	APIVersion string
	Timeout    time.Duration
	TLSConfig  *TLSConfig
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	CertFile   string
	KeyFile    string
	CAFile     string
	SkipVerify bool
}

// NewA1Adaptor creates a new A1 adaptor with the given configuration
func NewA1Adaptor(config *A1AdaptorConfig) (*A1Adaptor, error) {
	if config == nil {
		config = &A1AdaptorConfig{
			RICURL:     "http://near-rt-ric:8080",
			APIVersion: "v1",
			Timeout:    30 * time.Second,
		}
	}
	
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}
	
	// Configure TLS if provided
	if config.TLSConfig != nil {
		// TODO: Configure TLS transport
	}
	
	return &A1Adaptor{
		httpClient: httpClient,
		ricURL:     config.RICURL,
		apiVersion: config.APIVersion,
		timeout:    config.Timeout,
	}, nil
}

// CreatePolicyType creates a new policy type in the Near-RT RIC
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

// GetPolicyType retrieves a policy type from the Near-RT RIC
func (a *A1Adaptor) GetPolicyType(ctx context.Context, policyTypeID int) (*A1PolicyType, error) {
	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d", a.ricURL, a.apiVersion, policyTypeID)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

// ListPolicyTypes lists all policy types in the Near-RT RIC
func (a *A1Adaptor) ListPolicyTypes(ctx context.Context) ([]*A1PolicyType, error) {
	url := fmt.Sprintf("%s/a1-p/%s/policytypes", a.ricURL, a.apiVersion)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
	
	// Fetch details for each policy type
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

// DeletePolicyType deletes a policy type from the Near-RT RIC
func (a *A1Adaptor) DeletePolicyType(ctx context.Context, policyTypeID int) error {
	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d", a.ricURL, a.apiVersion, policyTypeID)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
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

// CreatePolicyInstance creates a new policy instance
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

// GetPolicyInstance retrieves a policy instance
func (a *A1Adaptor) GetPolicyInstance(ctx context.Context, policyTypeID int, instanceID string) (*A1PolicyInstance, error) {
	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d/policies/%s", 
		a.ricURL, a.apiVersion, policyTypeID, instanceID)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
	
	// Get status separately
	status, err := a.GetPolicyStatus(ctx, policyTypeID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policy status: %w", err)
	}
	
	return &A1PolicyInstance{
		PolicyInstanceID: instanceID,
		PolicyTypeID:     policyTypeID,
		PolicyData:       policyData,
		Status:           *status,
	}, nil
}

// ListPolicyInstances lists all policy instances for a policy type
func (a *A1Adaptor) ListPolicyInstances(ctx context.Context, policyTypeID int) ([]*A1PolicyInstance, error) {
	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d/policies", a.ricURL, a.apiVersion, policyTypeID)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
	
	// Fetch details for each instance
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

// UpdatePolicyInstance updates an existing policy instance
func (a *A1Adaptor) UpdatePolicyInstance(ctx context.Context, policyTypeID int, instanceID string, instance *A1PolicyInstance) error {
	// Same as create in A1 API
	return a.CreatePolicyInstance(ctx, policyTypeID, instance)
}

// DeletePolicyInstance deletes a policy instance
func (a *A1Adaptor) DeletePolicyInstance(ctx context.Context, policyTypeID int, instanceID string) error {
	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d/policies/%s", 
		a.ricURL, a.apiVersion, policyTypeID, instanceID)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
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

// GetPolicyStatus retrieves the status of a policy instance
func (a *A1Adaptor) GetPolicyStatus(ctx context.Context, policyTypeID int, instanceID string) (*A1PolicyStatus, error) {
	url := fmt.Sprintf("%s/a1-p/%s/policytypes/%d/policies/%s/status", 
		a.ricURL, a.apiVersion, policyTypeID, instanceID)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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

// ApplyPolicy applies an A1 policy from a ManagedElement
func (a *A1Adaptor) ApplyPolicy(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error {
	logger := log.FromContext(ctx)
	logger.Info("applying A1 policy", "managedElement", me.Name)
	
	if me.Spec.A1Policy.Raw == nil {
		logger.Info("no A1 policy to apply", "managedElement", me.Name)
		return nil
	}
	
	// Parse the A1 policy
	var policySpec map[string]interface{}
	if err := json.Unmarshal(me.Spec.A1Policy.Raw, &policySpec); err != nil {
		return fmt.Errorf("failed to unmarshal A1 policy: %w", err)
	}
	
	// Extract policy type ID and instance ID
	policyTypeID, ok := policySpec["policy_type_id"].(float64)
	if !ok {
		return fmt.Errorf("policy_type_id not found or invalid in A1 policy")
	}
	
	instanceID, ok := policySpec["policy_instance_id"].(string)
	if !ok {
		instanceID = fmt.Sprintf("%s-policy-%d", me.Name, int(policyTypeID))
	}
	
	// Extract policy data
	policyData, ok := policySpec["policy_data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("policy_data not found in A1 policy")
	}
	
	// Create policy instance
	instance := &A1PolicyInstance{
		PolicyInstanceID: instanceID,
		PolicyTypeID:     int(policyTypeID),
		PolicyData:       policyData,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	
	// Apply the policy
	if err := a.CreatePolicyInstance(ctx, int(policyTypeID), instance); err != nil {
		return fmt.Errorf("failed to create policy instance: %w", err)
	}
	
	// Wait for policy to be enforced (with timeout)
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

// RemovePolicy removes an A1 policy from a ManagedElement
func (a *A1Adaptor) RemovePolicy(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error {
	logger := log.FromContext(ctx)
	logger.Info("removing A1 policy", "managedElement", me.Name)
	
	if me.Spec.A1Policy.Raw == nil {
		logger.Info("no A1 policy to remove", "managedElement", me.Name)
		return nil
	}
	
	// Parse the A1 policy to get IDs
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
	
	// Delete the policy instance
	if err := a.DeletePolicyInstance(ctx, int(policyTypeID), instanceID); err != nil {
		return fmt.Errorf("failed to delete policy instance: %w", err)
	}
	
	logger.Info("successfully removed policy", 
		"policyTypeID", int(policyTypeID), 
		"instanceID", instanceID)
	
	return nil
}

// Helper function to create common policy types

// CreateQoSPolicyType creates a QoS policy type for network slicing
func CreateQoSPolicyType() *A1PolicyType {
	return &A1PolicyType{
		PolicyTypeID: 1000,
		Name:         "Network Slice QoS Policy",
		Description:  "Policy for managing QoS parameters in network slices",
		PolicySchema: map[string]interface{}{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type":    "object",
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
							"type":    "number",
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

// CreateTrafficSteeringPolicyType creates a traffic steering policy type
func CreateTrafficSteeringPolicyType() *A1PolicyType {
	return &A1PolicyType{
		PolicyTypeID: 2000,
		Name:         "Traffic Steering Policy",
		Description:  "Policy for steering traffic between cells or network functions",
		PolicySchema: map[string]interface{}{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type":    "object",
			"properties": map[string]interface{}{
				"ue_id": map[string]interface{}{
					"type": "string",
				},
				"target_cell": map[string]interface{}{
					"type": "string",
				},
				"traffic_percentage": map[string]interface{}{
					"type":    "number",
					"minimum": 0,
					"maximum": 100,
				},
			},
			"required": []string{"target_cell", "traffic_percentage"},
		},
	}
}