package a1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// O-RAN compliant A1 interface implementation

// A1PolicyTypeResponse represents O-RAN compliant policy type response
type A1PolicyTypeResponse struct {
	PolicyTypeID int                    `json:"policy_type_id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Schema       map[string]interface{} `json:"schema"`
	CreateSchema map[string]interface{} `json:"create_schema,omitempty"`
}

// A1ErrorResponse represents O-RAN compliant error response (RFC 7807)
type A1ErrorResponse struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// EnrichmentInfoType is defined in types.go

// EnrichmentInfoJob is defined in types.go

// O-RAN Compliant A1-P Interface Methods

// CreatePolicyTypeCompliant creates a policy type using O-RAN compliant endpoints
func (a *A1Adaptor) CreatePolicyTypeCompliant(ctx context.Context, policyType *A1PolicyType) error {
	// O-RAN compliant URL format
	url := fmt.Sprintf("%s/A1-P/v2/policytypes/%d", a.ricURL, policyType.PolicyTypeID)

	// Convert to O-RAN compliant format
	oranPolicyType := A1PolicyTypeResponse{
		PolicyTypeID: policyType.PolicyTypeID,
		Name:         policyType.Name,
		Description:  policyType.Description,
		Schema:       policyType.PolicySchema,
		CreateSchema: policyType.CreateSchema,
	}

	body, err := json.Marshal(oranPolicyType)
	if err != nil {
		return fmt.Errorf("failed to marshal policy type: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// O-RAN compliant headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// O-RAN compliant status code handling
	switch resp.StatusCode {
	case http.StatusCreated:
		return nil // Policy type created successfully
	case http.StatusOK:
		return nil // Policy type updated successfully
	case http.StatusBadRequest:
		return a.handleErrorResponse(resp, "Invalid policy type definition")
	case http.StatusConflict:
		return a.handleErrorResponse(resp, "Policy type already exists with different definition")
	case http.StatusNotImplemented:
		return a.handleErrorResponse(resp, "Policy type not supported by Near-RT RIC")
	default:
		return a.handleErrorResponse(resp, "Unexpected error")
	}
}

// GetPolicyTypeCompliant retrieves a policy type using O-RAN compliant format
func (a *A1Adaptor) GetPolicyTypeCompliant(ctx context.Context, policyTypeID int) (*A1PolicyTypeResponse, error) {
	url := fmt.Sprintf("%s/A1-P/v2/policytypes/%d", a.ricURL, policyTypeID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var policyType A1PolicyTypeResponse
		if err := json.NewDecoder(resp.Body).Decode(&policyType); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &policyType, nil
	case http.StatusNotFound:
		return nil, fmt.Errorf("policy type %d not found", policyTypeID)
	default:
		return nil, a.handleErrorResponse(resp, "Failed to get policy type")
	}
}

// ListPolicyTypesCompliant lists all policy types in O-RAN compliant format
func (a *A1Adaptor) ListPolicyTypesCompliant(ctx context.Context) ([]int, error) {
	url := fmt.Sprintf("%s/A1-P/v2/policytypes", a.ricURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, a.handleErrorResponse(resp, "Failed to list policy types")
	}

	var policyTypeIDs []int
	if err := json.NewDecoder(resp.Body).Decode(&policyTypeIDs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return policyTypeIDs, nil
}

// DeletePolicyTypeCompliant deletes a policy type using O-RAN compliant format
func (a *A1Adaptor) DeletePolicyTypeCompliant(ctx context.Context, policyTypeID int) error {
	url := fmt.Sprintf("%s/A1-P/v2/policytypes/%d", a.ricURL, policyTypeID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil // Successfully deleted
	case http.StatusNotFound:
		return fmt.Errorf("policy type %d not found", policyTypeID)
	case http.StatusBadRequest:
		return a.handleErrorResponse(resp, "Cannot delete policy type with active instances")
	default:
		return a.handleErrorResponse(resp, "Failed to delete policy type")
	}
}

// CreatePolicyInstanceCompliant creates a policy instance with proper validation
func (a *A1Adaptor) CreatePolicyInstanceCompliant(ctx context.Context, policyTypeID int, instanceID string, policyData map[string]interface{}) error {
	// First, validate policy data against schema
	if err := a.validatePolicyData(ctx, policyTypeID, policyData); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}

	url := fmt.Sprintf("%s/A1-P/v2/policytypes/%d/policies/%s", a.ricURL, policyTypeID, instanceID)

	body, err := json.Marshal(policyData)
	if err != nil {
		return fmt.Errorf("failed to marshal policy data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil // Policy instance created
	case http.StatusOK:
		return nil // Policy instance updated
	case http.StatusBadRequest:
		return a.handleErrorResponse(resp, "Invalid policy instance data")
	case http.StatusNotFound:
		return fmt.Errorf("policy type %d not found", policyTypeID)
	case http.StatusConflict:
		return a.handleErrorResponse(resp, "Policy instance conflicts with existing instance")
	default:
		return a.handleErrorResponse(resp, "Failed to create policy instance")
	}
}

// GetPolicyStatusCompliant retrieves policy status with detailed information
func (a *A1Adaptor) GetPolicyStatusCompliant(ctx context.Context, policyTypeID int, instanceID string) (*A1PolicyStatusCompliant, error) {
	url := fmt.Sprintf("%s/A1-P/v2/policytypes/%d/policies/%s/status", a.ricURL, policyTypeID, instanceID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json, application/problem+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var status A1PolicyStatusCompliant
		if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
			return nil, fmt.Errorf("failed to decode status response: %w", err)
		}
		return &status, nil
	case http.StatusNotFound:
		return nil, fmt.Errorf("policy instance %s of type %d not found", instanceID, policyTypeID)
	default:
		return nil, a.handleErrorResponse(resp, "Failed to get policy status")
	}
}

// A1PolicyStatusCompliant represents O-RAN compliant policy status
type A1PolicyStatusCompliant struct {
	EnforcementStatus string                 `json:"enforcement_status"`
	EnforcementReason string                 `json:"enforcement_reason,omitempty"`
	HasBeenDeleted    bool                   `json:"has_been_deleted,omitempty"`
	Deleted           bool                   `json:"deleted,omitempty"`
	CreatedAt         time.Time              `json:"created_at,omitempty"`
	ModifiedAt        time.Time              `json:"modified_at,omitempty"`
	AdditionalInfo    map[string]interface{} `json:"additional_info,omitempty"`
}

// validatePolicyData validates policy data against the policy type schema
func (a *A1Adaptor) validatePolicyData(ctx context.Context, policyTypeID int, policyData map[string]interface{}) error {
	// Get policy type schema
	policyType, err := a.GetPolicyTypeCompliant(ctx, policyTypeID)
	if err != nil {
		return fmt.Errorf("failed to get policy type: %w", err)
	}

	// Implement JSON schema validation here
	// This is a simplified version - in production, use a proper JSON schema validator
	if policyType.Schema != nil {
		// Check required fields
		if requiredFields, ok := policyType.Schema["required"].([]interface{}); ok {
			for _, field := range requiredFields {
				if fieldName, ok := field.(string); ok {
					if _, exists := policyData[fieldName]; !exists {
						return fmt.Errorf("required field '%s' missing from policy data", fieldName)
					}
				}
			}
		}
	}

	return nil
}

// handleErrorResponse handles O-RAN compliant error responses
func (a *A1Adaptor) handleErrorResponse(resp *http.Response, defaultMessage string) error {
	bodyBytes, _ := io.ReadAll(resp.Body)

	// Try to parse as O-RAN error response (RFC 7807 format)
	if resp.Header.Get("Content-Type") == "application/problem+json" {
		var errorResp A1ErrorResponse
		if err := json.Unmarshal(bodyBytes, &errorResp); err == nil {
			return fmt.Errorf("A1 error: %s - %s", errorResp.Title, errorResp.Detail)
		}
	}

	// Fallback to generic error
	return fmt.Errorf("%s: status=%d, body=%s", defaultMessage, resp.StatusCode, string(bodyBytes))
}
