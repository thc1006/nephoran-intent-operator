package a1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// A1-C (Consumer) Interface Implementation.

// This implements the O-RAN A1-C specification for Non-RT RIC to Near-RT RIC communication.

// A1ConsumerInterface is defined in types.go.

// A1PolicyTypeRegistration represents a policy type registration from Non-RT RIC.

type A1PolicyTypeRegistration struct {
	PolicyTypeID int `json:"policy_type_id"`

	PolicyTypeName string `json:"policy_type_name"`

	PolicyTypeVersion string `json:"policy_type_version"`

	PolicySchema map[string]interface{} `json:"policy_schema"`

	CreateSchema map[string]interface{} `json:"create_schema,omitempty"`

	Description string `json:"description,omitempty"`

	SupportedServices []string `json:"supported_services,omitempty"`

	ValidityPeriod *ValidityPeriod `json:"validity_period,omitempty"`

	RegistrationTime time.Time `json:"registration_time"`

	SourceNonRTRIC string `json:"source_non_rt_ric"`
}

// ValidityPeriod defines the validity period for a policy type.

type ValidityPeriod struct {
	StartTime time.Time `json:"start_time"`

	EndTime time.Time `json:"end_time"`
}

// PolicyTypeStatus represents the status of a registered policy type.

type PolicyTypeStatus struct {
	PolicyTypeID int `json:"policy_type_id"`

	RegistrationStatus string `json:"registration_status"` // REGISTERED, DEREGISTERED, FAILED

	ActiveInstances int `json:"active_instances"`

	LastStatusUpdate time.Time `json:"last_status_update"`

	StatusReason string `json:"status_reason,omitempty"`

	SupportedXApps []XAppInfo `json:"supported_xapps,omitempty"`

	PerformanceMetrics *PolicyTypeMetrics `json:"performance_metrics,omitempty"`
}

// XAppInfo represents information about xApps that support a policy type.

type XAppInfo struct {
	XAppName string `json:"xapp_name"`

	XAppVersion string `json:"xapp_version"`

	Services []string `json:"services"`

	Status string `json:"status"` // ACTIVE, INACTIVE, ERROR

}

// PolicyTypeMetrics represents performance metrics for a policy type.

type PolicyTypeMetrics struct {
	TotalPoliciesCreated int `json:"total_policies_created"`

	TotalPoliciesEnforced int `json:"total_policies_enforced"`

	AverageEnforcementTime time.Duration `json:"average_enforcement_time"`

	ErrorRate float64 `json:"error_rate"`

	LastMetricsUpdate time.Time `json:"last_metrics_update"`
}

// NearRTRICCapabilities represents the capabilities of a Near-RT RIC.

type NearRTRICCapabilities struct {
	RICID string `json:"ric_id"`

	RICName string `json:"ric_name"`

	RICVersion string `json:"ric_version"`

	SupportedInterfaces []string `json:"supported_interfaces"`

	SupportedServiceModels []ServiceModelCapability `json:"supported_service_models"`

	MaxPolicyTypes int `json:"max_policy_types"`

	MaxPolicyInstances int `json:"max_policy_instances"`

	SupportedQoSLevels []QoSLevel `json:"supported_qos_levels"`

	GeographicCoverage *GeographicArea `json:"geographic_coverage,omitempty"`

	LastCapabilityUpdate time.Time `json:"last_capability_update"`
}

// ServiceModelCapability represents a service model capability.

type ServiceModelCapability struct {
	ServiceModelOID string `json:"service_model_oid"`

	ServiceModelName string `json:"service_model_name"`

	ServiceModelVersion string `json:"service_model_version"`

	Functions []string `json:"functions"`

	RanFunctionID int `json:"ran_function_id"`
}

// QoSLevel represents a supported QoS level.

type QoSLevel struct {
	QoSID int `json:"qos_id"`

	QoSName string `json:"qos_name"`

	Description string `json:"description"`

	Priority int `json:"priority"`
}

// GeographicArea represents the geographic coverage area.

type GeographicArea struct {
	AreaType string `json:"area_type"` // CIRCULAR, POLYGONAL, RECTANGLE

	Coordinates interface{} `json:"coordinates"`

	Radius float64 `json:"radius,omitempty"` // for circular areas

}

// NearRTRICHealth represents the health status of a Near-RT RIC.

type NearRTRICHealth struct {
	RICID string `json:"ric_id"`

	OverallStatus string `json:"overall_status"` // HEALTHY, DEGRADED, UNHEALTHY

	ComponentStatus map[string]ComponentHealth `json:"component_status"`

	ResourceUsage *ResourceUsage `json:"resource_usage"`

	ConnectedE2Nodes int `json:"connected_e2_nodes"`

	ActiveXApps int `json:"active_xapps"`

	LastHealthCheck time.Time `json:"last_health_check"`

	HealthCheckID string `json:"health_check_id"`
}

// ComponentHealth represents the health status of a RIC component.

type ComponentHealth struct {
	ComponentName string `json:"component_name"`

	Status string `json:"status"` // HEALTHY, DEGRADED, UNHEALTHY

	StatusReason string `json:"status_reason,omitempty"`

	LastUpdate time.Time `json:"last_update"`
}

// ResourceUsage represents resource usage information.

type ResourceUsage struct {
	CPUUsagePercent float64 `json:"cpu_usage_percent"`

	MemoryUsagePercent float64 `json:"memory_usage_percent"`

	DiskUsagePercent float64 `json:"disk_usage_percent"`

	NetworkUtilization float64 `json:"network_utilization"`
}

// PolicyEventSubscription represents a subscription to policy events.

type PolicyEventSubscription struct {
	SubscriptionID string `json:"subscription_id"`

	CallbackURL string `json:"callback_url"`

	EventTypes []string `json:"event_types"` // POLICY_CREATED, POLICY_UPDATED, POLICY_DELETED, POLICY_ENFORCED

	PolicyTypeFilter []int `json:"policy_type_filter,omitempty"`

	SubscriberInfo *SubscriberInfo `json:"subscriber_info"`

	NotificationFormat string `json:"notification_format"` // JSON, XML

	ExpirationTime *time.Time `json:"expiration_time,omitempty"`
}

// SubscriberInfo represents information about the event subscriber.

type SubscriberInfo struct {
	SubscriberName string `json:"subscriber_name"`

	SubscriberType string `json:"subscriber_type"` // NON_RT_RIC, EXTERNAL_SYSTEM

	ContactInfo map[string]string `json:"contact_info,omitempty"`

	AuthenticationInfo *AuthInfo `json:"authentication_info,omitempty"`
}

// AuthInfo represents authentication information for callbacks.

type AuthInfo struct {
	AuthType string `json:"auth_type"` // OAUTH2, API_KEY, BASIC

	Credentials map[string]string `json:"credentials"`

	TokenURL string `json:"token_url,omitempty"`

	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// A1ConsumerAdaptor implements the A1-C interface.

type A1ConsumerAdaptor struct {
	httpClient *http.Client

	ricURL string

	apiVersion string

	clientID string
}

// NewA1ConsumerAdaptor creates a new A1-C adaptor.

func NewA1ConsumerAdaptor(ricURL, apiVersion, clientID string) *A1ConsumerAdaptor {

	return &A1ConsumerAdaptor{

		httpClient: &http.Client{Timeout: 30 * time.Second},

		ricURL: ricURL,

		apiVersion: apiVersion,

		clientID: clientID,
	}

}

// RegisterPolicyType registers a policy type with the Near-RT RIC.

func (ac *A1ConsumerAdaptor) RegisterPolicyType(ctx context.Context, policyType *A1PolicyTypeRegistration) error {

	url := fmt.Sprintf("%s/A1-C/v1/policytypes/%d", ac.ricURL, policyType.PolicyTypeID)

	// Set registration time and source.

	policyType.RegistrationTime = time.Now()

	policyType.SourceNonRTRIC = ac.clientID

	body, err := json.Marshal(policyType)

	if err != nil {

		return fmt.Errorf("failed to marshal policy type registration: %w", err)

	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Accept", "application/json, application/problem+json")

	req.Header.Set("X-Client-ID", ac.clientID)

	resp, err := ac.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusCreated:

		return nil // Policy type registered successfully

	case http.StatusOK:

		return nil // Policy type updated successfully

	case http.StatusBadRequest:

		return fmt.Errorf("invalid policy type registration")

	case http.StatusConflict:

		return fmt.Errorf("policy type already registered by different Non-RT RIC")

	case http.StatusServiceUnavailable:

		return fmt.Errorf("Near-RT RIC not available for registration")

	default:

		return fmt.Errorf("failed to register policy type: status=%d", resp.StatusCode)

	}

}

// DeregisterPolicyType deregisters a policy type from the Near-RT RIC.

func (ac *A1ConsumerAdaptor) DeregisterPolicyType(ctx context.Context, policyTypeID int) error {

	url := fmt.Sprintf("%s/A1-C/v1/policytypes/%d", ac.ricURL, policyTypeID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	req.Header.Set("X-Client-ID", ac.clientID)

	resp, err := ac.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusNoContent:

		return nil // Successfully deregistered

	case http.StatusNotFound:

		return fmt.Errorf("policy type %d not registered", policyTypeID)

	case http.StatusBadRequest:

		return fmt.Errorf("cannot deregister policy type with active instances")

	case http.StatusForbidden:

		return fmt.Errorf("not authorized to deregister this policy type")

	default:

		return fmt.Errorf("failed to deregister policy type: status=%d", resp.StatusCode)

	}

}

// GetPolicyTypeStatus gets the status of a registered policy type.

func (ac *A1ConsumerAdaptor) GetPolicyTypeStatus(ctx context.Context, policyTypeID int) (*PolicyTypeStatus, error) {

	url := fmt.Sprintf("%s/A1-C/v1/policytypes/%d/status", ac.ricURL, policyTypeID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	req.Header.Set("X-Client-ID", ac.clientID)

	resp, err := ac.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusOK:

		var status PolicyTypeStatus

		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {

			return nil, fmt.Errorf("failed to decode status response: %w", err)

		}

		return &status, nil

	case http.StatusNotFound:

		return nil, fmt.Errorf("policy type %d not found", policyTypeID)

	default:

		return nil, fmt.Errorf("failed to get policy type status: status=%d", resp.StatusCode)

	}

}

// GetNearRTRICCapabilities gets the capabilities of the Near-RT RIC.

func (ac *A1ConsumerAdaptor) GetNearRTRICCapabilities(ctx context.Context) (*NearRTRICCapabilities, error) {

	url := fmt.Sprintf("%s/A1-C/v1/capabilities", ac.ricURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	req.Header.Set("X-Client-ID", ac.clientID)

	resp, err := ac.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("failed to get RIC capabilities: status=%d", resp.StatusCode)

	}

	var capabilities NearRTRICCapabilities

	if err := json.NewDecoder(resp.Body).Decode(&capabilities); err != nil {

		return nil, fmt.Errorf("failed to decode capabilities response: %w", err)

	}

	return &capabilities, nil

}

// GetNearRTRICHealth gets the health status of the Near-RT RIC.

func (ac *A1ConsumerAdaptor) GetNearRTRICHealth(ctx context.Context) (*NearRTRICHealth, error) {

	url := fmt.Sprintf("%s/A1-C/v1/health", ac.ricURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	req.Header.Set("X-Client-ID", ac.clientID)

	resp, err := ac.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("failed to get RIC health: status=%d", resp.StatusCode)

	}

	var health NearRTRICHealth

	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {

		return nil, fmt.Errorf("failed to decode health response: %w", err)

	}

	return &health, nil

}

// SubscribeToPolicyEvents subscribes to policy-related events.

func (ac *A1ConsumerAdaptor) SubscribeToPolicyEvents(ctx context.Context, subscription *PolicyEventSubscription) error {

	url := fmt.Sprintf("%s/A1-C/v1/subscriptions", ac.ricURL)

	body, err := json.Marshal(subscription)

	if err != nil {

		return fmt.Errorf("failed to marshal subscription: %w", err)

	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Content-Type", "application/json")

	req.Header.Set("Accept", "application/json, application/problem+json")

	req.Header.Set("X-Client-ID", ac.clientID)

	resp, err := ac.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusCreated:

		return nil // Subscription created successfully

	case http.StatusBadRequest:

		return fmt.Errorf("invalid subscription request")

	case http.StatusConflict:

		return fmt.Errorf("subscription with same ID already exists")

	default:

		return fmt.Errorf("failed to create subscription: status=%d", resp.StatusCode)

	}

}

// UnsubscribeFromPolicyEvents unsubscribes from policy events.

func (ac *A1ConsumerAdaptor) UnsubscribeFromPolicyEvents(ctx context.Context, subscriptionID string) error {

	url := fmt.Sprintf("%s/A1-C/v1/subscriptions/%s", ac.ricURL, subscriptionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)

	if err != nil {

		return fmt.Errorf("failed to create request: %w", err)

	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	req.Header.Set("X-Client-ID", ac.clientID)

	resp, err := ac.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("failed to send request: %w", err)

	}

	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusNoContent:

		return nil // Successfully unsubscribed

	case http.StatusNotFound:

		return fmt.Errorf("subscription %s not found", subscriptionID)

	case http.StatusForbidden:

		return fmt.Errorf("not authorized to unsubscribe")

	default:

		return fmt.Errorf("failed to unsubscribe: status=%d", resp.StatusCode)

	}

}
