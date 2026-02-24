package providers

import (
	"context"
	"encoding/json"
	"time"
)

// MockProvider is a test implementation of the Provider interface.
// It allows for controlled testing scenarios with predefined responses.
type MockProvider struct {
	// Predefined responses for testing
	responses map[string]*IntentResponse
	
	// Error to return for testing error scenarios
	mockError error
	
	// Configuration
	config *Config
	
	// Call tracking for verification in tests
	ProcessIntentCalls []string
	ValidateConfigCalls int
	CloseCalls int
	
	// Simulated processing delay
	processingDelay time.Duration
}

// NewMockProvider creates a new mock provider for testing.
func NewMockProvider(config *Config) *MockProvider {
	return &MockProvider{
		config:    config,
		responses: make(map[string]*IntentResponse),
	}
}

// SetMockResponse sets a predefined response for a specific input.
// This allows tests to control what the provider returns.
func (m *MockProvider) SetMockResponse(input string, response *IntentResponse) {
	m.responses[input] = response
}

// SetMockError sets an error that will be returned by all methods.
// Useful for testing error handling scenarios.
func (m *MockProvider) SetMockError(err error) {
	m.mockError = err
}

// SetProcessingDelay sets a delay to simulate processing time.
func (m *MockProvider) SetProcessingDelay(delay time.Duration) {
	m.processingDelay = delay
}

// ProcessIntent implements the Provider interface.
func (m *MockProvider) ProcessIntent(ctx context.Context, input string) (*IntentResponse, error) {
	// Track the call for test verification
	m.ProcessIntentCalls = append(m.ProcessIntentCalls, input)
	
	// Return mock error if set
	if m.mockError != nil {
		return nil, m.mockError
	}
	
	// Simulate processing delay
	if m.processingDelay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.processingDelay):
		}
	}
	
	// Check for predefined response
	if response, exists := m.responses[input]; exists {
		return response, nil
	}
	
	// Default response - creates a basic scaling intent
	defaultIntent := map[string]interface{}{
		"intent_type": "scaling",
		"target":      "mock-target",
		"namespace":   "default",
		"replicas":    3,
		"reason":      "Mock response for input: " + input,
		"source":      "test",
		"priority":    5,
		"status":      "pending",
		"created_at":  time.Now().Format(time.RFC3339),
		"updated_at":  time.Now().Format(time.RFC3339),
	}
	
	intentJSON, _ := json.Marshal(defaultIntent)
	
	return &IntentResponse{
		JSON: intentJSON,
		Metadata: ResponseMetadata{
			Provider:       "MOCK",
			Model:          "mock-model",
			ProcessingTime: m.processingDelay,
			Confidence:     0.95,
		},
	}, nil
}

// GetProviderInfo implements the Provider interface.
func (m *MockProvider) GetProviderInfo() ProviderInfo {
	return ProviderInfo{
		Name:        "MOCK",
		Version:     "1.0.0-test",
		Description: "Mock provider for testing",
		RequiresAuth: false,
		SupportedFeatures: []string{
			"intent_processing",
			"configurable_responses",
			"error_simulation",
		},
	}
}

// ValidateConfig implements the Provider interface.
func (m *MockProvider) ValidateConfig() error {
	// Track the call for test verification
	m.ValidateConfigCalls++
	
	// Return mock error if set
	if m.mockError != nil {
		return m.mockError
	}
	
	// Mock validation always succeeds unless error is set
	return nil
}

// Close implements the Provider interface.
func (m *MockProvider) Close() error {
	// Track the call for test verification
	m.CloseCalls++
	
	// Return mock error if set
	if m.mockError != nil {
		return m.mockError
	}
	
	// Mock close always succeeds unless error is set
	return nil
}

// Reset clears all call tracking and mock state.
// Useful for resetting state between tests.
func (m *MockProvider) Reset() {
	m.ProcessIntentCalls = nil
	m.ValidateConfigCalls = 0
	m.CloseCalls = 0
	m.mockError = nil
	m.processingDelay = 0
	m.responses = make(map[string]*IntentResponse)
}

// GetCallCount returns the number of times ProcessIntent was called.
func (m *MockProvider) GetCallCount() int {
	return len(m.ProcessIntentCalls)
}

// GetLastInput returns the last input passed to ProcessIntent.
func (m *MockProvider) GetLastInput() string {
	if len(m.ProcessIntentCalls) == 0 {
		return ""
	}
	return m.ProcessIntentCalls[len(m.ProcessIntentCalls)-1]
}

// CreateMockIntentResponse creates a mock intent response for testing.
func CreateMockIntentResponse(intentType, target, namespace string, replicas int) *IntentResponse {
	intent := map[string]interface{}{
		"intent_type": intentType,
		"target":      target,
		"namespace":   namespace,
		"replicas":    replicas,
		"reason":      "Mock intent for testing",
		"source":      "test",
		"priority":    5,
		"status":      "pending",
		"created_at":  time.Now().Format(time.RFC3339),
		"updated_at":  time.Now().Format(time.RFC3339),
	}
	
	intentJSON, _ := json.Marshal(intent)
	
	return &IntentResponse{
		JSON: intentJSON,
		Metadata: ResponseMetadata{
			Provider:       "MOCK",
			Model:          "mock-model",
			ProcessingTime: 100 * time.Millisecond,
			Confidence:     0.9,
		},
	}
}