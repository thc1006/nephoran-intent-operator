package llm

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Priority types are defined in batch_processor.go

// BatchRequest and BatchResult types are defined in batch_processor.go

// StreamingRequest type is defined in interface_consolidated.go

// RequestContext type is defined in interface_consolidated.go

// Types referenced here are defined in their respective files:
// - HealthChecker, EndpointPool, BatchProcessorConfig: interface_consolidated.go
// - TokenManager: basic_token_manager.go and interface_consolidated.go
// - StreamingContextManager: interface_consolidated.go
// - ProcessingRequest, ProcessingResponse: interface_consolidated.go

// NetworkTopology and NetworkSlice types are defined elsewhere
// ProcessingRequest and ProcessingResponse types are defined in interface_consolidated.go

// Processor interface is defined in interface_consolidated.go

// Service provides the main LLM service interface
type Service struct {
	client LLMClient
}

// NewService creates a new LLM service
func NewService(client LLMClient) *Service {
	return &Service{client: client}
}

// ProcessIntent processes a natural language intent
func (s *Service) ProcessIntent(ctx context.Context, request *ProcessingRequest) (*ProcessingResponse, error) {
	if s.client == nil {
		return nil, fmt.Errorf("LLM client not initialized")
	}

	// Create a ProcessIntentRequest from the ProcessingRequest
	intentRequest := &ProcessIntentRequest{
		Intent: request.Intent,
		Context: map[string]string{
			"intent_type": request.IntentType,
			"provider":    request.Provider,
			"model":       request.Model,
		},
		Metadata: RequestMetadata{
			RequestID: request.RequestID,
			Source:    "llm-service",
		},
		Timestamp: time.Now(),
	}

	// Process the intent using the underlying client
	result, err := s.client.ProcessIntent(ctx, intentRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to process intent: %w", err)
	}

	// Convert ProcessIntentResponse to ProcessingResponse
	return &ProcessingResponse{
		ProcessedIntent:      result.Reasoning,
		StructuredParameters: result.StructuredIntent,
		ProcessedParameters:  result.Reasoning,
		ConfidenceScore:      result.Confidence,
		TokensUsed:          result.Metadata.TokensUsed,
		ProcessingTime:      time.Duration(result.Metadata.ProcessingTime * float64(time.Millisecond)),
		Metadata:            map[string]interface{}{
			"model_used": result.Metadata.ModelUsed,
			"cost":       result.Metadata.Cost,
		},
	}, nil
}


// TokenUsageInfo provides token usage statistics
type TokenUsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// generateRequestID function is defined elsewhere
