package llm

import "context"

// ClientInterface defines the interface for an LLM client.
type ClientInterface interface {
	ProcessIntent(ctx context.Context, intent string) (string, error)
}
