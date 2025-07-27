package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is a client for the LLM processor.
type Client struct {
	httpClient    *http.Client
	url           string
	promptEngine  *TelecomPromptEngine
	retryConfig   RetryConfig
	validator     *ResponseValidator
}

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

// ResponseValidator validates LLM responses
type ResponseValidator struct {
	requiredFields map[string]bool
}

// NewClient creates a new LLM client with enhanced capabilities.
func NewClient(url string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		url:          url,
		promptEngine: NewTelecomPromptEngine(),
		retryConfig: RetryConfig{
			MaxRetries: 3,
			BaseDelay:  time.Second,
			MaxDelay:   30 * time.Second,
		},
		validator: NewResponseValidator(),
	}
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

func (c *Client) ProcessIntent(ctx context.Context, intent string) (string, error) {
	// Classify intent to determine processing approach
	intentType := c.classifyIntent(intent)
	
	// Pre-process intent with parameter extraction
	extractedParams := c.promptEngine.ExtractParameters(intent)
	
	req := map[string]interface{}{
		"spec": map[string]string{
			"intent": intent,
		},
		"metadata": map[string]interface{}{
			"intent_type":      intentType,
			"extracted_params": extractedParams,
		},
	}

	// Process with retry logic
	var result string
	err := c.retryWithExponentialBackoff(ctx, func() error {
		var processErr error
		result, processErr = c.processWithValidation(ctx, req)
		return processErr
	})
	
	if err != nil {
		return "", fmt.Errorf("failed to process intent after retries: %w", err)
	}
	
	return result, nil
}

// classifyIntent determines the type of network intent
func (c *Client) classifyIntent(intent string) string {
	lowerIntent := strings.ToLower(intent)
	
	scaleIndicators := []string{"scale", "increase", "decrease", "replicas", "instances", "resize"}
	deployIndicators := []string{"deploy", "create", "setup", "configure", "install", "provision"}
	
	for _, indicator := range scaleIndicators {
		if strings.Contains(lowerIntent, indicator) {
			return "NetworkFunctionScale"
		}
	}
	
	for _, indicator := range deployIndicators {
		if strings.Contains(lowerIntent, indicator) {
			return "NetworkFunctionDeployment"
		}
	}
	
	return "NetworkFunctionDeployment" // Default
}

// processWithValidation handles the core processing with validation
func (c *Client) processWithValidation(ctx context.Context, req map[string]interface{}) (string, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.url+"/process", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "nephoran-intent-operator/v1.0.0")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse error response
		var errorResp map[string]interface{}
		if json.Unmarshal(respBody, &errorResp) == nil {
			if errorMsg, ok := errorResp["error"].(string); ok {
				return "", fmt.Errorf("LLM processor error (%d): %s", resp.StatusCode, errorMsg)
			}
		}
		return "", fmt.Errorf("LLM processor returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Validate response structure
	if err := c.validator.ValidateResponse(respBody); err != nil {
		return "", fmt.Errorf("response validation failed: %w", err)
	}

	return string(respBody), nil
}

// retryWithExponentialBackoff implements retry logic with exponential backoff
func (c *Client) retryWithExponentialBackoff(ctx context.Context, operation func() error) error {
	var lastErr error
	delay := c.retryConfig.BaseDelay
	
	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Exponential backoff with jitter
				delay = time.Duration(float64(delay) * 1.5)
				if delay > c.retryConfig.MaxDelay {
					delay = c.retryConfig.MaxDelay
				}
			}
		}
		
		lastErr = operation()
		if lastErr == nil {
			return nil // Success
		}
		
		// Check if error is retryable
		if !c.isRetryableError(lastErr) {
			return lastErr
		}
	}
	
	return fmt.Errorf("operation failed after %d retries: %w", c.retryConfig.MaxRetries, lastErr)
}

// isRetryableError determines if an error warrants a retry
func (c *Client) isRetryableError(err error) bool {
	errorStr := strings.ToLower(err.Error())
	
	// Network-related errors are typically retryable
	retryablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"service unavailable",
		"internal server error",
		"bad gateway",
		"circuit breaker",
	}
	
	for _, pattern := range retryablePatterns {
		if strings.Contains(errorStr, pattern) {
			return true
		}
	}
	
	return false
}

// ValidateResponse validates the structure of an LLM response
func (v *ResponseValidator) ValidateResponse(responseBody []byte) error {
	var response map[string]interface{}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return fmt.Errorf("invalid JSON response: %w", err)
	}
	
	// Check required fields
	for field := range v.requiredFields {
		if _, exists := response[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
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
func isValidKubernetesName(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}
	
	// Kubernetes names must match DNS subdomain format
	for i, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.') {
			return false
		}
		if i == 0 && (r == '-' || r == '.') {
			return false
		}
		if i == len(name)-1 && (r == '-' || r == '.') {
			return false
		}
	}
	
	return true
}

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