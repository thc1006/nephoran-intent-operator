package llm

import (
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// ClientInterface defines the interface for an LLM client.
// This interface extends the shared interface to maintain backward compatibility
type ClientInterface interface {
	shared.ClientInterface
}
