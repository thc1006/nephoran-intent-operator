package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ValidationError represents a validation error with missing fields
type ValidationError struct {
	Message       string   `json:"message"`
	MissingFields []string `json:"missing_fields"`
}

// Error implements the error interface
func (ve *ValidationError) Error() string {
	if len(ve.MissingFields) == 0 {
		return ve.Message
	}
	return fmt.Sprintf("%s: missing fields %v", ve.Message, ve.MissingFields)
}

// ResponseValidator validates LLM responses
type ResponseValidator struct {
	requiredFields map[string]bool
}

// NewResponseValidator creates a new response validator
func NewResponseValidator() *ResponseValidator {
	return &ResponseValidator{
		requiredFields: map[string]bool{
			"type":      true,
			"name":      true,
			"namespace": true,
			"spec":      true,
		},
	}
}

// ValidateResponse validates the structure of an LLM response
func (v *ResponseValidator) ValidateResponse(responseBody []byte) error {
	var response map[string]interface{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("invalid JSON response: %w", err)
	}

	// Check required fields and collect missing ones
	var missingFields []string
	for field := range v.requiredFields {
		if _, exists := response[field]; !exists {
			missingFields = append(missingFields, field)
		}
	}

	// Return structured error if any fields are missing
	if len(missingFields) > 0 {
		return &ValidationError{
			Message:       "Response validation failed",
			MissingFields: missingFields,
		}
	}

	// Validate type field
	if responseType, ok := response["type"].(string); ok {
		validTypes := []string{"NetworkFunctionDeployment", "NetworkFunctionScale"}
		valid := false
		for _, validType := range validTypes {
			if responseType == validType {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid response type: %s", responseType)
		}
	} else {
		return fmt.Errorf("type field must be a string")
	}

	// Validate name field format (Kubernetes naming)
	if name, ok := response["name"].(string); ok {
		if !isValidKubernetesName(name) {
			return fmt.Errorf("invalid Kubernetes name format: %s", name)
		}
	} else {
		return fmt.Errorf("name field must be a string")
	}

	// Validate namespace field format
	if namespace, ok := response["namespace"].(string); ok {
		if !isValidKubernetesName(namespace) {
			return fmt.Errorf("invalid Kubernetes namespace format: %s", namespace)
		}
	} else {
		return fmt.Errorf("namespace field must be a string")
	}

	// Validate spec field structure
	if spec, ok := response["spec"].(map[string]interface{}); ok {
		if responseType := response["type"].(string); responseType == "NetworkFunctionDeployment" {
			return v.validateDeploymentSpec(spec)
		} else if responseType == "NetworkFunctionScale" {
			return v.validateScaleSpec(spec)
		}
	} else {
		return fmt.Errorf("spec field must be an object")
	}

	return nil
}

// validateDeploymentSpec validates NetworkFunctionDeployment spec
func (v *ResponseValidator) validateDeploymentSpec(spec map[string]interface{}) error {
	// Check required deployment fields
	requiredFields := []string{"replicas", "image"}
	for _, field := range requiredFields {
		if _, exists := spec[field]; !exists {
			return fmt.Errorf("missing required deployment spec field: %s", field)
		}
	}

	// Validate replicas
	if replicas, ok := spec["replicas"].(float64); ok {
		if replicas < 1 || replicas > 100 {
			return fmt.Errorf("replicas must be between 1 and 100, got: %v", replicas)
		}
	} else {
		return fmt.Errorf("replicas must be a number")
	}

	// Validate image
	if image, ok := spec["image"].(string); ok {
		if image == "" {
			return fmt.Errorf("image cannot be empty")
		}
	} else {
		return fmt.Errorf("image must be a string")
	}

	return nil
}

// validateScaleSpec validates NetworkFunctionScale spec
func (v *ResponseValidator) validateScaleSpec(spec map[string]interface{}) error {
	// For scaling, we need at least one scaling parameter
	hasHorizontal := false
	hasVertical := false

	if scaling, ok := spec["scaling"].(map[string]interface{}); ok {
		if horizontal, exists := scaling["horizontal"]; exists {
			hasHorizontal = true
			if h, ok := horizontal.(map[string]interface{}); ok {
				if replicas, exists := h["replicas"]; exists {
					if r, ok := replicas.(float64); ok {
						if r < 1 || r > 100 {
							return fmt.Errorf("horizontal scaling replicas must be between 1 and 100")
						}
					} else {
						return fmt.Errorf("horizontal scaling replicas must be a number")
					}
				}
			}
		}

		if vertical, exists := scaling["vertical"]; exists {
			hasVertical = true
			if v, ok := vertical.(map[string]interface{}); ok {
				// Validate CPU format if present
				if cpu, exists := v["cpu"]; exists {
					if cpuStr, ok := cpu.(string); ok {
						if !isValidCPUFormat(cpuStr) {
							return fmt.Errorf("invalid CPU format: %s", cpuStr)
						}
					} else {
						return fmt.Errorf("vertical scaling CPU must be a string")
					}
				}

				// Validate memory format if present
				if memory, exists := v["memory"]; exists {
					if memStr, ok := memory.(string); ok {
						if !isValidMemoryFormat(memStr) {
							return fmt.Errorf("invalid memory format: %s", memStr)
						}
					} else {
						return fmt.Errorf("vertical scaling memory must be a string")
					}
				}
			}
		}
	}

	if !hasHorizontal && !hasVertical {
		return fmt.Errorf("scaling spec must include either horizontal or vertical scaling parameters")
	}

	return nil
}

// Helper validation functions
func isValidCPUFormat(cpu string) bool {
	// Valid formats: "100m", "0.1", "1", "2000m"
	if cpu == "" {
		return false
	}

	if strings.HasSuffix(cpu, "m") {
		// Millicores format
		cpuValue := strings.TrimSuffix(cpu, "m")
		for _, r := range cpuValue {
			if r < '0' || r > '9' {
				return false
			}
		}
		return len(cpuValue) > 0
	} else {
		// Cores format (can include decimal)
		for _, r := range cpu {
			if !((r >= '0' && r <= '9') || r == '.') {
				return false
			}
		}
		return len(cpu) > 0
	}
}

func isValidMemoryFormat(memory string) bool {
	// Valid formats: "256Mi", "1Gi", "512Mi", "2Gi"
	if memory == "" {
		return false
	}

	validSuffixes := []string{"Ki", "Mi", "Gi", "Ti", "Pi", "Ei", "K", "M", "G", "T", "P", "E"}

	for _, suffix := range validSuffixes {
		if strings.HasSuffix(memory, suffix) {
			memoryValue := strings.TrimSuffix(memory, suffix)
			for _, r := range memoryValue {
				if r < '0' || r > '9' {
					return false
				}
			}
			return len(memoryValue) > 0
		}
	}

	return false
}