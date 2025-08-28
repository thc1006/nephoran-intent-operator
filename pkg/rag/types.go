//go:build !disable_rag
// +build !disable_rag

// Package rag provides Retrieval-Augmented Generation interfaces
// This file contains interface definitions that allow conditional compilation
// with or without Weaviate dependencies
package rag

import (
	"context"
	"fmt"
	"time"
)

// Doc represents a document retrieved from the RAG system
type Doc struct {
	ID         string
	Content    string
	Confidence float64
	Metadata   map[string]interface{}
}

// RAGClient is the main interface for RAG operations
// This allows for different implementations (Weaviate, no-op, etc.)
type RAGClient interface {
	// Retrieve performs a semantic search for relevant documents
	Retrieve(ctx context.Context, query string) ([]Doc, error)

	// ProcessIntent processes an intent and returns a response
	ProcessIntent(ctx context.Context, intent string) (string, error)

	// IsHealthy returns the health status of the RAG client
	IsHealthy() bool

	// Initialize initializes the RAG client and its dependencies
	Initialize(ctx context.Context) error

	// Shutdown gracefully shuts down the RAG client and releases resources
	Shutdown(ctx context.Context) error
}

// Note: SearchResult is defined in enhanced_rag_integration.go

// RAGClientConfig holds configuration for RAG clients
type RAGClientConfig struct {
	// Common configuration
	Enabled          bool
	MaxSearchResults int
	MinConfidence    float64

	// Weaviate-specific (used only when rag build tag is enabled)
	WeaviateURL    string
	WeaviateAPIKey string

	// LLM configuration
	LLMEndpoint string
	LLMAPIKey   string
	MaxTokens   int
	Temperature float32
}

// Note: TokenUsage is defined in embedding_service.go

// NewRAGClient creates a new RAG client based on build tags
// With "rag" build tag: returns Weaviate-based implementation
// Without "rag" build tag: returns no-op implementation
func NewRAGClient(config *RAGClientConfig) RAGClient {
	// This function will be implemented differently in:
	// - weaviate_client.go (with //go:build rag)
	// - noop/client.go (no build tag)
	return newRAGClientImpl(config)
}

// QueryRequest represents a request for RAG system query processing
type QueryRequest struct {
	Query      string                 `json:"query"`                // The user's query text
	IntentType string                 `json:"intentType,omitempty"` // Type of intent (e.g., "knowledge_request", "deployment_request")
	Context    map[string]interface{} `json:"context,omitempty"`    // Additional context for the query
	UserID     string                 `json:"userID,omitempty"`     // User identifier for personalization
	SessionID  string                 `json:"sessionID,omitempty"`  // Session identifier for conversation context
	MaxResults int                    `json:"maxResults,omitempty"` // Maximum number of results to return
	MinScore   float64                `json:"minScore,omitempty"`   // Minimum relevance score for results
	Filters    map[string]interface{} `json:"filters,omitempty"`    // Additional filters for retrieval
}

// Service is an alias for RAGService for backward compatibility
type Service = RAGService

// AsyncWorkerConfig defines configuration for async worker pools
type AsyncWorkerConfig struct {
	DocumentWorkers   int `json:"document_workers"`
	QueryWorkers      int `json:"query_workers"`
	DocumentQueueSize int `json:"document_queue_size"`
	QueryQueueSize    int `json:"query_queue_size"`
}

// AsyncWorkerPoolForTests represents a test-compatible async worker pool
// This is separate from the main AsyncWorkerPool in pipeline.go
type AsyncWorkerPoolForTests struct {
	documentWorkers int
	queryWorkers    int
	documentQueue   chan TestDocumentJob
	queryQueue      chan TestQueryJob
	metrics         *AsyncWorkerMetrics
	started         bool
}

// AsyncWorkerMetrics tracks async worker pool metrics
type AsyncWorkerMetrics struct {
	DocumentJobsSubmitted int64
	QueryJobsSubmitted    int64
	DocumentJobsCompleted int64
	QueryJobsCompleted    int64
	DocumentJobsFailed    int64
	QueryJobsFailed       int64
}

// RetrievedContext represents retrieved context from a query (for test compatibility)
type RetrievedContext struct {
	ID         string                 `json:"id"`
	Content    string                 `json:"content"`
	Confidence float64                `json:"confidence"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// TestDocumentJob represents a test document job (different from pipeline DocumentJob)
type TestDocumentJob struct {
	ID       string                               `json:"id"`
	FilePath string                               `json:"file_path,omitempty"`
	Content  string                               `json:"content"`
	Metadata map[string]interface{}               `json:"metadata,omitempty"`
	Callback func(string, []DocumentChunk, error) `json:"-"`
}

// TestQueryJob represents a test query job (different from pipeline QueryJob)
type TestQueryJob struct {
	ID       string                                  `json:"id"`
	Query    string                                  `json:"query"`
	Filters  map[string]interface{}                  `json:"filters,omitempty"`
	Limit    int                                     `json:"limit,omitempty"`
	Callback func(string, []RetrievedContext, error) `json:"-"`
}

// NewAsyncWorkerPool creates a new async worker pool for tests
func NewAsyncWorkerPool(config *AsyncWorkerConfig) *AsyncWorkerPoolForTests {
	return &AsyncWorkerPoolForTests{
		documentWorkers: config.DocumentWorkers,
		queryWorkers:    config.QueryWorkers,
		documentQueue:   make(chan TestDocumentJob, config.DocumentQueueSize),
		queryQueue:      make(chan TestQueryJob, config.QueryQueueSize),
		metrics: &AsyncWorkerMetrics{
			DocumentJobsSubmitted: 0,
			QueryJobsSubmitted:    0,
			DocumentJobsCompleted: 0,
			QueryJobsCompleted:    0,
			DocumentJobsFailed:    0,
			QueryJobsFailed:       0,
		},
		started: false,
	}
}

// Start starts the async worker pool
func (p *AsyncWorkerPoolForTests) Start() error {
	if p.started {
		return fmt.Errorf("worker pool already started")
	}
	p.started = true
	return nil
}

// Stop stops the async worker pool
func (p *AsyncWorkerPoolForTests) Stop(timeout time.Duration) error {
	if !p.started {
		return fmt.Errorf("async worker pool not started")
	}
	p.started = false

	// For graceful shutdown, wait a bit for pending jobs to complete
	// This is a simple implementation for testing purposes
	if timeout > 0 {
		time.Sleep(time.Millisecond * 200) // Allow time for goroutines to complete
	}
	return nil
}

// SubmitDocumentJob submits a document job for processing
func (p *AsyncWorkerPoolForTests) SubmitDocumentJob(job TestDocumentJob) error {
	if !p.started {
		return fmt.Errorf("async worker pool not started")
	}

	// Check queue fullness by trying to send to channel with select
	select {
	case p.documentQueue <- job:
		// Successfully queued
	default:
		// Queue is full
		return fmt.Errorf("document queue is full")
	}

	p.metrics.DocumentJobsSubmitted++

	// For testing, simulate processing by calling the callback directly
	go func() {
		time.Sleep(100 * time.Millisecond) // Simulate processing time

		var chunks []DocumentChunk
		var err error

		// Check for failure conditions
		if len(job.Content) == 0 {
			err = fmt.Errorf("document processing failed: empty content")
			p.metrics.DocumentJobsFailed++
		} else {
			// Create mock chunks using the existing DocumentChunk type
			chunks = []DocumentChunk{
				{
					ID:           job.ID + "_chunk_1",
					DocumentID:   job.ID,
					Content:      job.Content[:len(job.Content)/2],
					CleanContent: job.Content[:len(job.Content)/2],
					ChunkIndex:   0,
					StartOffset:  0,
					EndOffset:    len(job.Content) / 2,
				},
				{
					ID:           job.ID + "_chunk_2",
					DocumentID:   job.ID,
					Content:      job.Content[len(job.Content)/2:],
					CleanContent: job.Content[len(job.Content)/2:],
					ChunkIndex:   1,
					StartOffset:  len(job.Content) / 2,
					EndOffset:    len(job.Content),
				},
			}
		}

		p.metrics.DocumentJobsCompleted++

		if job.Callback != nil {
			job.Callback(job.ID, chunks, err)
		}

		// Remove job from queue
		<-p.documentQueue
	}()

	return nil
}

// SubmitQueryJob submits a query job for processing
func (p *AsyncWorkerPoolForTests) SubmitQueryJob(job TestQueryJob) error {
	if !p.started {
		return fmt.Errorf("async worker pool not started")
	}

	// Check queue fullness by trying to send to channel with select
	select {
	case p.queryQueue <- job:
		// Successfully queued
	default:
		// Queue is full
		return fmt.Errorf("query queue is full")
	}

	p.metrics.QueryJobsSubmitted++

	// For testing, simulate processing by calling the callback directly
	go func() {
		time.Sleep(50 * time.Millisecond) // Simulate processing time

		var results []RetrievedContext
		var err error

		// Check for failure conditions
		if len(job.Query) == 0 {
			err = fmt.Errorf("query processing failed: empty query")
			p.metrics.QueryJobsFailed++
		} else {
			// Create mock results using RetrievedContext
			results = []RetrievedContext{
				{
					ID:         "result_1",
					Content:    "Mock search result 1 for query: " + job.Query,
					Confidence: 0.85,
					Metadata:   map[string]interface{}{"source": "test"},
				},
				{
					ID:         "result_2",
					Content:    "Mock search result 2 for query: " + job.Query,
					Confidence: 0.75,
					Metadata:   map[string]interface{}{"source": "test"},
				},
			}
		}

		p.metrics.QueryJobsCompleted++

		if job.Callback != nil {
			job.Callback(job.ID, results, err)
		}

		// Remove job from queue
		<-p.queryQueue
	}()

	return nil
}

// GetMetrics returns current metrics
func (p *AsyncWorkerPoolForTests) GetMetrics() *AsyncWorkerMetrics {
	return p.metrics
}

// RetrievalRequest represents a request for document retrieval
type RetrievalRequest struct {
	Query      string                 `json:"query"`                // The search query
	MaxResults int                    `json:"maxResults,omitempty"` // Maximum number of results to return
	MinScore   float64                `json:"minScore,omitempty"`   // Minimum relevance score for results
	Filters    map[string]interface{} `json:"filters,omitempty"`    // Additional filters for retrieval
	Context    map[string]interface{} `json:"context,omitempty"`    // Additional context for the query
}

// RetrievalResponse represents a response from document retrieval
type RetrievalResponse struct {
	Documents             []map[string]interface{} `json:"documents"`             // Retrieved documents with metadata
	Duration              time.Duration            `json:"duration"`              // Time taken for retrieval
	AverageRelevanceScore float64                  `json:"averageRelevanceScore"` // Average relevance score of results
	TopRelevanceScore     float64                  `json:"topRelevanceScore"`     // Highest relevance score in results
	QueryWasEnhanced      bool                     `json:"queryWasEnhanced"`      // Whether the query was enhanced/expanded
	Metadata              map[string]interface{}   `json:"metadata,omitempty"`    // Additional metadata about the retrieval
	Error                 string                   `json:"error,omitempty"`       // Error message if retrieval failed
}

// Note: QueryResponse and RAGService are defined in other RAG files
