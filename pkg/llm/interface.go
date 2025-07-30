package llm

import (
	"context"
	
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// ClientInterface defines the interface for an LLM client.
// This interface extends the shared interface to maintain backward compatibility
type ClientInterface interface {
	shared.ClientInterface
}

// Ensure our interface is compatible with the shared interface
var _ shared.ClientInterface = (*ClientInterface)(nil)
