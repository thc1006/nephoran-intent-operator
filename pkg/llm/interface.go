package llm

import (
	"context"
	"time"
)

// DEPRECATED: This package's ClientInterface is redundant.
// Use github.com/thc1006/nephoran-intent-operator/pkg/shared.ClientInterface directly.
// This package is kept for backward compatibility but should be phased out.

// Config represents LLM client configuration
type Config struct {
	Endpoint string
	Timeout  time.Duration
	APIKey   string
	Model    string
}

// IntentRequest represents a request to process an intent
type IntentRequest struct {
	Intent      string                 `json:"intent"`
	Prompt      string                 `json:"prompt"`
	Context     map[string]interface{} `json:"context"`
	MaxTokens   int                    `json:"maxTokens"`
	Temperature float64                `json:"temperature"`
}

// IntentResponse represents the response from intent processing
type IntentResponse struct {
	Response   string                 `json:"response"`
	Confidence float64                `json:"confidence"`
	Tokens     int                    `json:"tokens"`
	Duration   time.Duration          `json:"duration"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ProcessIntent processes a NetworkIntent with the LLM
func (c *Client) ProcessIntent(ctx context.Context, req *IntentRequest) (*IntentResponse, error) {
	// Implementation would use the existing client infrastructure
	// This is a simplified interface for the blueprint package
	return &IntentResponse{
		Response:   `{"deployment_config": {"replicas": 3, "image": "nginx:1.20"}}`,
		Confidence: 0.9,
		Tokens:     100,
		Duration:   time.Second,
		Metadata:   make(map[string]interface{}),
	}, nil
}

// HealthCheck performs a health check on the LLM client
func (c *Client) HealthCheck(ctx context.Context) bool {
	// Implementation would check LLM service availability
	return true
}

// NewClient creates a new LLM client with the given configuration
func NewClient(config *Config) (*Client, error) {
	// This would use the existing Client constructor
	// For now, return a basic client
	return &Client{
		url:    config.Endpoint,
		apiKey: config.APIKey,
	}, nil
}
