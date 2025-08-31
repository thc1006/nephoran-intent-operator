package llm

import "github.com/thc1006/nephoran-intent-operator/pkg/rag"

// Type aliases to bridge the gap between handler expectations and available stubs

// RelevanceScorer alias for the handlers - use the existing implementation
type RelevanceScorer = RelevanceScorerImpl

// RAGAwarePromptBuilder alias for the handlers - use the stub
type RAGAwarePromptBuilder = RAGAwarePromptBuilderStub

// StreamingProcessor alias for the handlers - use the stub 
type StreamingProcessor = StreamingProcessorStub

// Constructor aliases to match handler expectations
func NewRelevanceScorer(args ...interface{}) *RelevanceScorer {
	// Use the existing default config function
	config := getDefaultRelevanceScorerConfig()
	
	// Create a minimal embedding service stub
	var embeddingService rag.EmbeddingServiceInterface = nil // Use nil for now
	
	return NewRelevanceScorerImpl(config, embeddingService)
}

func NewRAGAwarePromptBuilder(args ...interface{}) *RAGAwarePromptBuilder {
	return NewRAGAwarePromptBuilderStub()
}

// NewStreamingProcessor is already defined in clean_stubs.go