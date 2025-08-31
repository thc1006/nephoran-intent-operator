package llm

import "github.com/thc1006/nephoran-intent-operator/pkg/rag"

// Type aliases to bridge the gap between handler expectations and available stubs

// RelevanceScorer alias for the handlers - use the existing implementation
type CompatibilityRelevanceScorer = RelevanceScorerImpl

// ConsolidatedRAGAwarePromptBuilder alias for the handlers - use the stub
type ConsolidatedRAGAwarePromptBuilder = RAGAwarePromptBuilderStub

// StreamingProcessor alias for the handlers - use the stub 
type CompatibilityStreamingProcessor = ConsolidatedStreamingProcessor

// Constructor aliases to match handler expectations
func NewCompatibilityRelevanceScorer(args ...interface{}) *CompatibilityRelevanceScorer {
	// Use the existing default config function
	config := getDefaultRelevanceScorerConfig()
	
	// Create a minimal embedding service stub
	var embeddingService rag.EmbeddingServiceInterface = nil // Use nil for now
	
	return NewRelevanceScorerImpl(config, embeddingService)
}

func NewCompatibilityRAGAwarePromptBuilder(args ...interface{}) *ConsolidatedRAGAwarePromptBuilder {
	return NewRAGAwarePromptBuilderStub()
}

// NewStreamingProcessor is already defined in clean_stubs.go