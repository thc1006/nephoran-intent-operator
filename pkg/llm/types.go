package llm

import (
	"context"
	"fmt"
	"time"
)

// Priority types are defined in interface_consolidated.go

// BatchRequest and BatchResult types are defined in interface_consolidated.go

// StreamingRequest type is defined in interface_consolidated.go

// RequestContext type is defined in interface_consolidated.go

// TokenManager interface defines token management capabilities
type TokenManager interface {
	AllocateTokens(request string) (int, error)
	ReleaseTokens(count int) error
	GetAvailableTokens() int
	// Model capability methods
	EstimateTokensForModel(model string, text string) (int, error)
	SupportsSystemPrompt(model string) bool
	SupportsChatFormat(model string) bool
	SupportsStreaming(model string) bool
	TruncateToFit(text string, maxTokens int, model string) (string, error)
	// Additional methods for compatibility
	GetTokenCount(text string) int
	ValidateModel(model string) error
	GetSupportedModels() []string
	// Budget calculation method
	CalculateTokenBudget(context string, requirements map[string]interface{}) (int, error)
}

// RelevanceScorer interface for backwards compatibility with handlers
type RelevanceScorer interface {
	Score(ctx context.Context, doc string, intent string) (float32, error)
	GetMetrics() map[string]interface{}
}

// CircuitBreakerManagerInterface defines the interface for circuit breaker management
type CircuitBreakerManagerInterface interface {
	GetOrCreate(name string, config *CircuitBreakerConfig) *CircuitBreaker
	Get(name string) (*CircuitBreaker, bool)
	Remove(name string)
	List() []string
	GetAllStats() map[string]interface{}
	GetStats() (map[string]interface{}, error)
	Shutdown()
	ResetAll()
}

// CircuitBreakerConfig defines configuration for circuit breakers
type CircuitBreakerConfig struct {
	MaxRequests      uint32        `json:"max_requests"`
	Timeout          time.Duration `json:"timeout"`
	Interval         time.Duration `json:"interval"`
	ReadyToTrip      func(counts map[string]uint64) bool
	OnStateChange    func(name string, from, to string)
	IsSuccessful     func(err error) bool
	MaxRetries       int           `json:"max_retries"`
	RetryTimeout     time.Duration `json:"retry_timeout"`
}

// CircuitBreaker provides circuit breaker functionality
type CircuitBreaker struct {
	name      string
	config    *CircuitBreakerConfig
	state     string
	counts    map[string]uint64
	expiry    time.Time
}

// LLMClient defines the interface for LLM client implementations
type LLMClient interface {
	ProcessIntent(ctx context.Context, request *ProcessIntentRequest) (*ProcessIntentResponse, error)
	GetMetrics() ClientMetrics
	Close() error
}

// ProcessIntentRequest represents a request to process intent
type ProcessIntentRequest struct {
	Intent    string            `json:"intent"`
	Context   map[string]string `json:"context,omitempty"`
	Metadata  RequestMetadata   `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// ProcessIntentResponse represents a response from intent processing
type ProcessIntentResponse struct {
	Reasoning        string                 `json:"reasoning"`
	StructuredIntent map[string]interface{} `json:"structured_intent,omitempty"`
	Confidence       float64                `json:"confidence"`
	Metadata         ResponseMetadata       `json:"metadata"`
}

// ResponseMetadata contains metadata for responses
type ResponseMetadata struct {
	RequestID      string  `json:"request_id"`
	TokensUsed     int     `json:"tokens_used"`
	ProcessingTime float64 `json:"processing_time_ms"`
	Cost           float64 `json:"cost"`
	ModelUsed      string  `json:"model_used"`
}

// Types referenced here are defined in their respective files:
// - HealthChecker, EndpointPool, BatchProcessorConfig: interface_consolidated.go
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
			"model":       request.Model,
		},
		Metadata: RequestMetadata{
			RequestID: request.ID,
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
	// Convert map[string]interface{} to JSON string for ProcessedParameters
	structuredParams := ""
	if result.StructuredIntent != nil {
		structuredParams = fmt.Sprintf("%v", result.StructuredIntent)
	}

	return &ProcessingResponse{
		ID:                  request.ID,
		Response:            result.Reasoning,
		ProcessedParameters: structuredParams,
		Confidence:          float32(result.Confidence),
		TokensUsed:          result.Metadata.TokensUsed,
		ProcessingTime:      time.Duration(result.Metadata.ProcessingTime * float64(time.Millisecond)),
		Cost:                result.Metadata.Cost,
		ModelUsed:           result.Metadata.ModelUsed,
	}, nil
}

// TokenUsageInfo provides token usage statistics
type TokenUsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// generateRequestID function is defined elsewhere