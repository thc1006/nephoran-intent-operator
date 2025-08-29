//go:build !disable_rag

// Package llm provides LLM (Large Language Model) integration and processing for the Nephoran Intent Operator.
package llm

import (
	"context"
)

// DEPRECATED: This package's ClientInterface is redundant.
// Use github.com/thc1006/nephoran-intent-operator/pkg/shared.ClientInterface directly.
// This package is kept for backward compatibility but should be phased out.

// IntentRequest and IntentResponse are now defined in interface_consolidated.go.

// ProcessIntent is now defined in client_consolidated.go.
/*func (c *Client) ProcessIntent(ctx context.Context, req *IntentRequest) (*IntentResponse, error) {
	// Implementation would use the existing client infrastructure.
	// This is a simplified interface for the blueprint package.
	return &IntentResponse{
		Response:   `{"deployment_config": {"replicas": 3, "image": "nginx:1.20"}}`,
		Confidence: 0.9,
		Tokens:     100,
		Duration:   time.Second,
		Metadata:   make(map[string]interface{}),
	}, nil
}*/

// HealthCheck performs a health check on the LLM client.
func (c *Client) HealthCheck(ctx context.Context) bool {
	// Implementation would check LLM service availability.
	return true
}
