package llm

import (
	"context"
	"net/http"
	"strings"
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

// BuiltContext represents the result of building context
type BuiltContext struct {
	Context       string      `json:"context"`
	UsedDocuments []Document  `json:"used_documents"`
	QualityScore  float64     `json:"quality_score"`
}


// Document represents a document used in context building
type Document struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Source   string `json:"source"`
	Metadata string `json:"metadata"`
}

// ContextBuilder provides a stub implementation
type ContextBuilder struct {
	Config *Config
}

// GetMetrics returns metrics for the context builder
func (cb *ContextBuilder) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"contexts_built":   0,
		"cache_hits":       0,
		"cache_misses":     0,
		"build_time_ms":    0,
	}
}

// NewContextBuilder creates a new ContextBuilder with the given config
func NewContextBuilder(config *Config) *ContextBuilder {
	return &ContextBuilder{
		Config: config,
	}
}

// BuildContext builds context from the given documents
func (cb *ContextBuilder) BuildContext(ctx context.Context, query string, documents []Document) (*BuiltContext, error) {
	// Simple stub implementation for testing
	var contextText string
	var usedDocs []Document
	
	// Use documents based on relevance (stub implementation)
	for i, doc := range documents {
		if i >= 3 { // Limit to 3 documents for testing
			break
		}
		contextText += doc.Title + ": " + doc.Content + "\n"
		usedDocs = append(usedDocs, doc)
	}
	
	// Respect MaxContextTokens if configured
	if cb.Config != nil && cb.Config.MaxContextTokens > 0 {
		// Simple token counting by word count approximation
		words := len(strings.Fields(contextText))
		if words > cb.Config.MaxContextTokens {
			// Truncate to fit token budget
			wordSlice := strings.Fields(contextText)
			if len(wordSlice) > cb.Config.MaxContextTokens {
				contextText = strings.Join(wordSlice[:cb.Config.MaxContextTokens], " ")
			}
		}
	}
	
	return &BuiltContext{
		Context:       contextText,
		UsedDocuments: usedDocs,
		QualityScore:  0.75, // Fixed score for testing
	}, nil
}

// CalculateRelevanceScores calculates relevance scores for documents
func (cb *ContextBuilder) CalculateRelevanceScores(ctx context.Context, query string, documents []Document) ([]RelevanceScore, error) {
	scores := make([]RelevanceScore, len(documents))
	
	// Simple relevance scoring based on keyword matching
	queryWords := strings.Fields(strings.ToLower(query))
	
	for i, doc := range documents {
		overallScore := 0.0
		docText := strings.ToLower(doc.Title + " " + doc.Content)
		
		// Count matching words
		matchCount := 0
		for _, word := range queryWords {
			if strings.Contains(docText, word) {
				matchCount++
			}
		}
		
		// Calculate overall score as percentage of matching words
		if len(queryWords) > 0 {
			overallScore = float64(matchCount) / float64(len(queryWords))
		}
		
		// Calculate authority score based on source
		authorityScore := 0.5 // Default authority
		if doc.Source != "" {
			if strings.Contains(strings.ToLower(doc.Source), "3gpp") {
				authorityScore = 0.9 // High authority for 3GPP standards
			} else if strings.Contains(strings.ToLower(doc.Source), "o-ran") {
				authorityScore = 0.8 // High authority for O-RAN specs
			}
		}
		
		// Apply quality threshold if configured
		if cb.Config != nil && overallScore < cb.Config.QualityThreshold {
			overallScore = 0.0 // Below threshold documents get 0 score
		}
		
		scores[i] = RelevanceScore{
			OverallScore:   float32(overallScore),
			SemanticScore:  float32(overallScore),
			AuthorityScore: float32(authorityScore),
			RecencyScore:   0.5, // Default recency
			DomainScore:    0.7, // Default domain relevance
			IntentScore:    float32(overallScore),
			Explanation:    "Calculated based on keyword matching and source authority",
		}
	}
	
	return scores, nil
}

// StreamingProcessor provides a stub implementation
type StreamingProcessor struct{}

// GetMetrics returns metrics for the streaming processor
func (sp *StreamingProcessor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"active_streams":    0,
		"completed_streams": 0,
		"failed_streams":    0,
		"throughput":        0.0,
	}
}

// CreateSession creates a new streaming session
func (sp *StreamingProcessor) CreateSession(sessionID string, w interface{}, r interface{}) (*StreamingSession, error) {
	return &StreamingSession{
		ID:     sessionID,
		Status: StreamingStatusActive,
	}, nil
}

// GetActiveSessions returns list of active sessions
func (sp *StreamingProcessor) GetActiveSessions() []string {
	return []string{}
}

// StreamChunk streams a chunk to the session
func (sp *StreamingProcessor) StreamChunk(sessionID string, chunk *StreamingChunk) error {
	return nil
}

// CompleteStream completes the streaming session
func (sp *StreamingProcessor) CompleteStream(sessionID string) error {
	return nil
}

// NewStreamingProcessor creates a new streaming processor
func NewStreamingProcessor(config *StreamingConfig) *StreamingProcessor {
	return &StreamingProcessor{}
}

// Streaming constants are defined in streaming_processor.go

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

// StreamingSession and StreamingChunk are defined in streaming_processor.go
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

func (csp *ConsolidatedStreamingProcessor) HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *StreamingRequest) error {
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