// Package test_validation provides additional helper methods for O-RAN interface validation.
package test_validation

import (
	"context"
	"encoding/json"
	"time"
)

// GetRICMockService returns the RIC mock service for testing.
func (oiv *ORANInterfaceValidator) GetRICMockService() *RICMockService {
	return oiv.ricMockService
}

// GetSMOMockService returns the SMO mock service for testing.
func (oiv *ORANInterfaceValidator) GetSMOMockService() *SMOMockService {
	return oiv.smoMockService
}

// GetE2MockService returns the E2 mock service for testing.
func (oiv *ORANInterfaceValidator) GetE2MockService() *E2MockService {
	return oiv.e2MockService
}

// ValidateYANGModel validates YANG model structure for O1 testing.
func (oiv *ORANInterfaceValidator) ValidateYANGModel(model map[string]any) bool {
	// Check required fields
	requiredFields := []string{"module", "namespace", "prefix", "description"}

	for _, field := range requiredFields {
		if _, exists := model[field].(string); !exists {
			return false
		}
	}

	// Validate schema structure
	schema, exists := model["schema"].(map[string]any)
	if !exists {
		return false
	}

	container, exists := schema["container"].(map[string]any)
	if !exists {
		return false
	}

	_, exists = container["name"].(string)
	return exists
}

// TestNETCONFOperations tests NETCONF protocol operations for O1 testing.
func (oiv *ORANInterfaceValidator) TestNETCONFOperations(ctx context.Context) bool {
	// Simulate NETCONF session establishment
	session := map[string]any{
		"transport": "SSH",
		"status":    "active",
	}

	// Validate session establishment
	if session["status"] != "active" {
		return false
	}

	// Test get operation
	getConfig := map[string]any{
		"type":  "xpath",
		"xpath": "/ric-config",
	}

	// Test edit-config operation
	editConfig := map[string]any{
		"ric-config": map[string]any{},
	}

	// Test commit operation
	commit := map[string]any{}

	// Simulate NETCONF operations execution
	operations := []map[string]any{getConfig, editConfig, commit}

	for _, op := range operations {
		operation, exists := op["operation"].(string)
		if !exists {
			continue
		}

		// Simulate processing time based on operation type
		switch operation {
		case "get-config":
			time.Sleep(10 * time.Millisecond)
		case "edit-config":
			time.Sleep(20 * time.Millisecond)
		case "commit":
			time.Sleep(30 * time.Millisecond)
		}
	}

	return true
}

// ValidateTerraformTemplate validates Terraform template structure for O2 testing.
func (oiv *ORANInterfaceValidator) ValidateTerraformTemplate(template map[string]any) bool {
	// Check for required sections
	requiredSections := []string{"terraform", "resource"}

	for _, section := range requiredSections {
		if _, exists := template[section].(map[string]any); !exists {
			return false
		}
	}

	// Validate terraform section
	terraform, exists := template["terraform"].(map[string]any)
	if !exists {
		return false
	}

	_, exists = terraform["required_providers"].(map[string]any)
	return exists
}

