package llm

import (
	"context"
	"encoding/json"
	"fmt"
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

// Note: BuiltContext and Document types are defined in missing_types.go

// ContextBuilder provides a stub implementation
type ContextBuilder struct {
	Config *ContextBuilderConfig
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

// Note: NewContextBuilder is defined in missing_types.go
// CalculateRelevanceScores calculates relevance scores for documents against a query
func (cb *ContextBuilder) CalculateRelevanceScores(ctx context.Context, query string, documents []Document) ([]RelevanceScore, error) {
	scores := make([]RelevanceScore, len(documents))
	
	for i, doc := range documents {
		// Simple scoring based on title and content matching
		titleRelevance := 0.0
		if strings.Contains(strings.ToLower(doc.Title), strings.ToLower(query)) {
			titleRelevance = 0.8
		}
		
		contentRelevance := 0.0
		if strings.Contains(strings.ToLower(doc.Content), strings.ToLower(query)) {
			contentRelevance = 0.6
		}
		
		// Authority score based on source
		authorityScore := 0.5 // Default
		if strings.Contains(doc.Source, "3GPP") {
			authorityScore = 0.9
		} else if strings.Contains(doc.Source, "O-RAN") {
			authorityScore = 0.8
		}
		
		overallScore := (titleRelevance + contentRelevance + authorityScore) / 3.0
		
		// Apply quality threshold if configured
		if cb.Config != nil && cb.Config.QualityThreshold > 0 && overallScore < cb.Config.QualityThreshold {
			overallScore = 0.0 // Below threshold documents get 0 score
		}
		
		factorsJSON, _ := json.Marshal(map[string]interface{}{
			"title_relevance":   titleRelevance,
			"content_relevance": contentRelevance,
			"authority_score":   authorityScore,
		})
		
		scores[i] = RelevanceScore{
			OverallScore:   float32(overallScore),
			SemanticScore:  float32((titleRelevance + contentRelevance) / 2.0),
			AuthorityScore: float32(authorityScore),
			RecencyScore:   0.5,  // Default freshness score
			DomainScore:    0.7,  // Default quality score
			IntentScore:    0.6,  // Default intent relevance score
			Explanation:    fmt.Sprintf("Scored based on title (%f) and content (%f) relevance", titleRelevance, contentRelevance),
			Factors:        json.RawMessage(factorsJSON),
			ProcessingTime: time.Since(time.Now()),
			CacheUsed:      false,
		}
	}
	
	return scores, nil
}

// BuildContext builds a context from documents based on relevance
func (cb *ContextBuilder) BuildContext(ctx context.Context, query string, documents []Document) (*BuiltContext, error) {
	startTime := time.Now()
	
	// Calculate relevance scores
	scores, err := cb.CalculateRelevanceScores(ctx, query, documents)
	if err != nil {
		return nil, err
	}
	
	// Sort documents by relevance score
	documentScoreMap := make(map[string]*RelevanceScore)
	for i := range scores {
		documentScoreMap[documents[i].ID] = &scores[i]
	}
	
	// Select best documents for context
	maxDocs := 5 // Default max documents
	if cb.Config != nil && cb.Config.MaxDocuments > 0 {
		maxDocs = cb.Config.MaxDocuments
	}
	
	selectedDocs := make([]Document, 0, maxDocs)
	contextBuilder := strings.Builder{}
	
	for i, doc := range documents {
		if i >= maxDocs {
			break
		}
		
		score := documentScoreMap[doc.ID]
		if score != nil && score.OverallScore > 0.3 { // Minimum relevance threshold
			selectedDocs = append(selectedDocs, doc)
			
			// Add document to context
			if contextBuilder.Len() > 0 {
				contextBuilder.WriteString("\n\n")
			}
			contextBuilder.WriteString(fmt.Sprintf("Document: %s\nSource: %s\nContent: %s", 
				doc.Title, doc.Source, doc.Content))
		}
	}
	
	contextText := contextBuilder.String()
	
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
	
	// Calculate overall quality score
	qualityScore := 0.0
	if len(selectedDocs) > 0 {
		totalScore := 0.0
		for _, doc := range selectedDocs {
			if score := documentScoreMap[doc.ID]; score != nil {
				totalScore += float64(score.OverallScore)
			}
		}
		qualityScore = totalScore / float64(len(selectedDocs))
	}
	
	return &BuiltContext{
		Context:       contextText,
		UsedDocuments: selectedDocs,
		QualityScore:  qualityScore,
		TokenCount:    len(strings.Fields(contextText)), // Rough token approximation
		BuildTime:     time.Since(startTime),
	}, nil
}
// StreamingProcessor provides a stub implementation
type StreamingProcessor struct{}

// NewStreamingProcessor creates a new streaming processor stub for testing
func NewStreamingProcessor(config *StreamingConfig) *StreamingProcessor {
	return &StreamingProcessor{}
}

// StreamChunk sends a chunk to an active streaming session
func (sp *StreamingProcessor) StreamChunk(sessionID string, chunk *StreamingChunk) error {
	// Stub implementation for testing
	return nil
}

// CompleteStream completes an active streaming session
func (sp *StreamingProcessor) CompleteStream(sessionID string) error {
	// Stub implementation for testing
	return nil
}

// CreateSession creates a mock streaming session for testing
func (sp *StreamingProcessor) CreateSession(sessionID string, w interface{}, r interface{}) (*MockStreamingSession, error) {
	return &MockStreamingSession{
		ID:     sessionID,
		Status: string(StreamingStatusActive),
	}, nil
}

// GetActiveSessions returns mock active sessions for testing
func (sp *StreamingProcessor) GetActiveSessions() []string {
	return []string{}
}

// MockStreamingSession represents a mock streaming session for testing
type MockStreamingSession struct {
	ID     string
	Status string
}

// GetMetrics returns metrics for the streaming processor
func (sp *StreamingProcessor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"active_streams":    0,
		"completed_streams": 0,
		"failed_streams":    0,
		"throughput":        0.0,
	}
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