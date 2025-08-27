package llm

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Priority represents request priority levels
type Priority int

const (
	LowPriority Priority = iota
	NormalPriority
	HighPriority
	CriticalPriority
	PriorityUrgent = CriticalPriority
)

// Aliases for compatibility
const (
	PriorityLow    = LowPriority
	PriorityNormal = NormalPriority
	PriorityHigh   = HighPriority
)

// String returns the string representation of the priority
func (p Priority) String() string {
	switch p {
	case LowPriority:
		return "low"
	case NormalPriority:
		return "normal"
	case HighPriority:
		return "high"
	case CriticalPriority:
		return "critical"
	default:
		return "unknown"
	}
}

// BatchRequest represents a request to be processed in a batch
type BatchRequest struct {
	ID         string
	Intent     string
	IntentType string
	ModelName  string
	Priority   Priority
	Context    context.Context
	ResultChan chan *BatchResult
	ResponseCh chan *ProcessingResult // Added for processing.go compatibility
	Metadata   map[string]interface{}
	SubmitTime time.Time
	Timeout    time.Duration
}

// BatchResult represents the result of a batch request
type BatchResult struct {
	RequestID   string
	Response    string
	Error       error
	ProcessTime time.Duration
	BatchID     string
	BatchSize   int
	QueueTime   time.Duration
	Tokens      int // Added for optimized_controller.go compatibility
}

// StreamingRequest represents a request for streaming processing
type StreamingRequest struct {
	Query       string                 `json:"query"`
	IntentType  string                 `json:"intent_type"`
	ModelName   string                 `json:"model_name"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature float32                `json:"temperature"`
	Context     string                 `json:"context,omitempty"`
	EnableRAG   bool                   `json:"enable_rag"`
	SessionID   string                 `json:"session_id,omitempty"`
	ClientID    string                 `json:"client_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RequestContext holds context information for processing requests
type RequestContext struct {
	ID        string
	Intent    string
	StartTime time.Time
	Metadata  map[string]interface{}
}

// Missing types for undefined references

// HealthChecker provides health check functionality
type HealthChecker struct {
	// Implementation details would go here
}

// EndpointPool manages connection pooling for endpoints
type EndpointPool struct {
	// Implementation details would go here
}

// BatchProcessorConfig holds configuration for batch processing
type BatchProcessorConfig struct {
	BatchSize       int           `json:"batch_size"`
	FlushInterval   time.Duration `json:"flush_interval"`
	MaxConcurrency  int           `json:"max_concurrency"`
	RetryAttempts   int           `json:"retry_attempts"`
	TimeoutDuration time.Duration `json:"timeout_duration"`
}

// LLMClient interface for LLM operations
type LLMClient interface {
	ProcessIntent(ctx context.Context, intent string) (string, error)
}

// TokenManager manages token allocation and tracking
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
}

// StreamingContextManager manages streaming request contexts
type StreamingContextManager struct {
	activeStreams     map[string]*StreamingContext
	mutex             sync.RWMutex
	tokenManager      TokenManager
	injectionOverhead time.Duration
}

// StreamingContext holds context for a streaming request
type StreamingContext struct {
	SessionID string
	StartTime time.Time
	LastSeen  time.Time
	Metadata  map[string]interface{}
}

// NewStreamingContextManager creates a new streaming context manager
func NewStreamingContextManager(tokenManager TokenManager, injectionOverhead time.Duration) *StreamingContextManager {
	return &StreamingContextManager{
		activeStreams:     make(map[string]*StreamingContext),
		tokenManager:      tokenManager,
		injectionOverhead: injectionOverhead,
	}
}

// Close cleans up the streaming context manager
func (scm *StreamingContextManager) Close() error {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	// Clear all active streams
	scm.activeStreams = make(map[string]*StreamingContext)
	return nil
}

// NetworkTopology represents network topology information for processing
type NetworkTopology struct {
	Region           string            `json:"region"`
	AvailabilityZone string            `json:"availability_zone"`
	NetworkSlices    []NetworkSlice    `json:"network_slices"`
	Constraints      map[string]string `json:"constraints"`
}

// NetworkSlice represents a network slice configuration
type NetworkSlice struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"` // eMBB, URLLC, mMTC
	Status       string  `json:"status"`
	Capacity     int     `json:"capacity,omitempty"`
	Utilization  float64 `json:"utilization,omitempty"`
	Throughput   float64 `json:"throughput,omitempty"`
	Latency      float64 `json:"latency,omitempty"`
	Reliability  float64 `json:"reliability,omitempty"`
	ConnectedUEs int     `json:"connected_ues,omitempty"`
}

// ValidationError represents a validation error with field-level detail
type ValidationError struct {
	Field         string   `json:"field"`
	Message       string   `json:"message"`
	Code          string   `json:"code"`
	Severity      string   `json:"severity"`
	MissingFields []string `json:"missing_fields,omitempty"` // For compatibility with tests
}

func (v *ValidationError) Error() string {
	return fmt.Sprintf("validation error in field '%s': %s", v.Field, v.Message)
}

// ProcessingResult represents the result of LLM processing
type ProcessingResult struct {
	Content           string                 `json:"content"`
	TokensUsed        int                    `json:"tokens_used"`
	ProcessingTime    time.Duration          `json:"processing_time"`
	CacheHit          bool                   `json:"cache_hit"`
	Batched           bool                   `json:"batched"`
	Metadata          map[string]interface{} `json:"metadata"`
	Error             error                  `json:"error,omitempty"`
	ProcessingContext *ProcessingContext     `json:"processing_context,omitempty"` // Added for processing_pipeline.go
	Success           bool                   `json:"success"`                      // Added for processing_pipeline.go
}

// ProcessingContext, ClassificationResult, EnrichmentContext, ValidationResult
// are defined in processing_pipeline.go and security_validator.go

// ProcessingRequest represents a request for LLM processing
type ProcessingRequest struct {
	Intent            string                 `json:"intent"`
	SystemPrompt      string                 `json:"system_prompt"`
	UserPrompt        string                 `json:"user_prompt"`
	Context           map[string]interface{} `json:"context"`
	MaxTokens         int                    `json:"max_tokens"`
	Temperature       float64                `json:"temperature"`
	RequestID         string                 `json:"request_id"`
	ProcessingTimeout time.Duration          `json:"processing_timeout"`
}

// ProcessingResponse represents the response from LLM processing
type ProcessingResponse struct {
	// Core fields
	ProcessedIntent      string                 `json:"processed_intent"`
	StructuredParameters map[string]interface{} `json:"structured_parameters,omitempty"`
	NetworkFunctions     []interface{}          `json:"network_functions,omitempty"`
	ExtractedEntities    map[string]interface{} `json:"extracted_entities,omitempty"`
	TelecomContext       map[string]interface{} `json:"telecom_context,omitempty"`
	TokenUsage           *TokenUsageInfo        `json:"token_usage,omitempty"`
	
	// Legacy fields for compatibility
	ProcessedParameters string                 `json:"processed_parameters"`
	ConfidenceScore     float64                `json:"confidence_score"`
	TokensUsed          int                    `json:"tokens_used"`
	ProcessingTime      time.Duration          `json:"processing_time"`
	Metadata            map[string]interface{} `json:"metadata"`
	Error               error                  `json:"error,omitempty"`
}

// Processor interface for LLM processing
type Processor interface {
	ProcessIntent(ctx context.Context, request *ProcessingRequest) (*ProcessingResponse, error)
}

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

	// Process the intent using the underlying client
	result, err := s.client.ProcessIntent(ctx, request.Intent)
	if err != nil {
		return nil, fmt.Errorf("failed to process intent: %w", err)
	}

	return &ProcessingResponse{
		ProcessedParameters: result,
		ConfidenceScore:     0.95, // Default confidence
		TokensUsed:         request.MaxTokens / 2, // Estimated
		ProcessingTime:     time.Second,
		Metadata:           make(map[string]interface{}),
	}, nil
}


// TokenUsageInfo provides token usage statistics
type TokenUsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// generateRequestID generates a unique request identifier
func generateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
