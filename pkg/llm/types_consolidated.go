package llm

import (
	"context"
	"time"
)

// Priority represents request priority levels using Go 1.24 typed constants
type Priority int

const (
	LowPriority Priority = iota
	MediumPriority
	HighPriority
)

// RequestMetadata contains metadata for LLM requests
type RequestMetadata struct {
	RequestID  string            `json:"request_id"`
	UserID     string            `json:"user_id,omitempty"`
	SessionID  string            `json:"session_id,omitempty"`
	Source     string            `json:"source"`
	Properties map[string]string `json:"properties,omitempty"`
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
	ResponseCh chan *ProcessingResult
	Metadata   map[string]interface{}
	SubmitTime time.Time
	Timeout    time.Duration
}

// BatchResult represents the result of a batch request
type BatchResult struct {
	RequestID      string
	Response       string
	Error          error
	ProcessTime    time.Duration // Match batch_processor.go field name
	BatchID        string        // Match batch_processor.go field name
	BatchSize      int           // Match batch_processor.go field name
	QueueTime      time.Duration // Match batch_processor.go field name
	Tokens         int           // Match batch_processor.go field name
	ProcessingTime time.Duration // Additional field for compatibility
	TokensUsed     int           // Additional field for compatibility
	ModelUsed      string        // Additional field for compatibility
	Cost           float64       // Additional field for compatibility
	Metadata       map[string]interface{}
}

// Generic type aliases for backward compatibility using Go 1.24 patterns
type (
	// RequestPriority is an alias for Priority
	RequestPriority = Priority
)

// Note: ProcessingResult is defined separately in processing_result.go

// STUB TYPES for compatibility - these provide minimal implementations

// RAGAwarePromptBuilderStub provides a stub implementation
type RAGAwarePromptBuilderStub struct{}

// ConsolidatedStreamingProcessor provides a stub implementation
type ConsolidatedStreamingProcessor struct{}

// ContextBuilder provides a stub implementation
type ContextBuilder struct{}

// StreamingProcessor provides a stub implementation
type StreamingProcessor struct{}

// InMemoryStreamingContextManager is a stub implementation
type InMemoryStreamingContextManager struct{}

// Implement StreamingContextManager interface methods
func (m *InMemoryStreamingContextManager) CreateContext(sessionID string) (*StreamingContext, error) {
	return &StreamingContext{SessionID: sessionID}, nil
}
func (m *InMemoryStreamingContextManager) GetContext(sessionID string) (*StreamingContext, error) {
	return &StreamingContext{SessionID: sessionID}, nil
}
func (m *InMemoryStreamingContextManager) UpdateContext(sessionID string, content string) error {
	return nil
}
func (m *InMemoryStreamingContextManager) DeleteContext(sessionID string) error {
	return nil
}
func (m *InMemoryStreamingContextManager) ListActiveSessions() []string {
	return []string{}
}
func (m *InMemoryStreamingContextManager) Close() error {
	return nil
}

// StreamingContext is defined in interface_consolidated.go

// Methods for ConsolidatedStreamingProcessor
func (csp *ConsolidatedStreamingProcessor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"active_streams":    0,
		"completed_streams": 0,
		"failed_streams":    0,
	}
}

func (csp *ConsolidatedStreamingProcessor) ProcessStreaming(ctx context.Context, request *StreamingRequest) (<-chan *StreamingResponse, error) {
	responseChan := make(chan *StreamingResponse, 1)
	close(responseChan)
	return responseChan, nil
}

func (csp *ConsolidatedStreamingProcessor) Close() error {
	return nil
}

// NewRAGAwarePromptBuilderStub creates a new stub
func NewRAGAwarePromptBuilderStub() *RAGAwarePromptBuilderStub {
	return &RAGAwarePromptBuilderStub{}
}

// NewConsolidatedTokenManager creates a stub token manager
func NewConsolidatedTokenManager() interface{} {
	return struct{}{}
}

// NewStreamingContextManager creates a stub context manager (with variable arguments for compatibility)
func NewStreamingContextManager(args ...interface{}) StreamingContextManager {
	return &InMemoryStreamingContextManager{}
}

// BatchProcessorStats is defined in batch_processor.go