//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"context"
	"time"
)

// CONSOLIDATED INTERFACES - Essential interfaces only, duplicates removed

// LLMProcessor is the main interface for LLM processing
type LLMProcessor interface {
	ProcessIntent(ctx context.Context, request *ProcessingRequest) (*ProcessingResponse, error)
	GetMetrics() ClientMetrics
	Shutdown()
}

// Processor is an alias for backward compatibility using Go 1.24 type alias
type Processor = LLMProcessor

// BatchProcessor handles batch processing of multiple intents
type BatchProcessor interface {
	ProcessRequest(ctx context.Context, intent, intentType, modelName string, priority Priority) (*BatchResult, error)
	GetStats() BatchProcessorStats
	Close() error
}

// CORE TYPES - Only non-duplicate types remain

// ProcessingRequest represents a request for LLM processing
type ProcessingRequest struct {
	ID         string            `json:"id"`
	Intent     string            `json:"intent"`
	IntentType string            `json:"intent_type"`
	Model      string            `json:"model"`
	Context    map[string]string `json:"context,omitempty"`
	Metadata   RequestMetadata   `json:"metadata,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

// ProcessingResponse represents a response from LLM processing
type ProcessingResponse struct {
	ID                  string        `json:"id"`
	Response            string        `json:"response"`
	ProcessedParameters string        `json:"processed_parameters"`
	Confidence          float32       `json:"confidence"`
	TokensUsed          int           `json:"tokens_used"`
	ProcessingTime      time.Duration `json:"processing_time"`
	Cost                float64       `json:"cost"`
	ModelUsed           string        `json:"model_used"`
	Error               string        `json:"error,omitempty"`
}

// StreamingRequest represents a request for streaming processing
type StreamingRequest struct {
	ID         string                 `json:"id"`
	Intent     string                 `json:"intent"`
	IntentType string                 `json:"intent_type"`
	Model      string                 `json:"model"`
	ModelName  string                 `json:"model_name,omitempty"`     // Added for compatibility
	Query      string                 `json:"query,omitempty"`          // Added for compatibility
	SessionID  string                 `json:"session_id,omitempty"`     // Added for compatibility
	EnableRAG  bool                   `json:"enable_rag,omitempty"`     // Added for compatibility
	Metadata   map[string]interface{} `json:"metadata,omitempty"`       // Added for compatibility
	Context    map[string]string      `json:"context,omitempty"`
	Stream     bool                   `json:"stream"`
}

// StreamingResponse represents a streaming response chunk
type StreamingResponse struct {
	ID       string `json:"id"`
	Content  string `json:"content"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
	Metadata struct {
		TokensUsed int `json:"tokens_used"`
	} `json:"metadata,omitempty"`
}

// UTILITY INTERFACES

// HealthChecker interface for health checking
type HealthChecker interface {
	CheckHealth(ctx context.Context) (*HealthStatus, error)
	GetName() string
}

// HealthStatus represents health check status
type HealthStatus struct {
	Healthy   bool              `json:"healthy"`
	Message   string            `json:"message,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// EndpointPool manages LLM endpoints
type EndpointPool interface {
	GetEndpoint() (string, error)
	MarkHealthy(endpoint string)
	MarkUnhealthy(endpoint string)
	GetHealthyEndpoints() []string
}

// StreamingContextManager manages streaming contexts
type StreamingContextManager interface {
	CreateContext(sessionID string) (*StreamingContext, error)
	GetContext(sessionID string) (*StreamingContext, error)
	UpdateContext(sessionID string, content string) error
	DeleteContext(sessionID string) error
	ListActiveSessions() []string
}

// StreamingContext represents a streaming session context
type StreamingContext struct {
	SessionID     string            `json:"session_id"`
	History       []string          `json:"history"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	LastUpdatedAt time.Time         `json:"last_updated_at"`
	TokenCount    int               `json:"token_count"`
}

// CONFIGURATION TYPES

// BatchProcessorConfig represents batch processor configuration
type BatchProcessorConfig struct {
	MaxWorkers      int           `json:"max_workers"`
	QueueSize       int           `json:"queue_size"`
	ProcessTimeout  time.Duration `json:"process_timeout"`
	RetryAttempts   int           `json:"retry_attempts"`
	RetryDelay      time.Duration `json:"retry_delay"`
	MetricsInterval time.Duration `json:"metrics_interval"`
}

// RequestContext contains context information for processing requests
type RequestContext struct {
	ID          string                 `json:"id,omitempty"`          // Added for compatibility
	Intent      string                 `json:"intent,omitempty"`      // Added for compatibility
	StartTime   time.Time              `json:"start_time,omitempty"`  // Added for compatibility
	SessionID   string                 `json:"session_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`    // Changed to interface{} for compatibility
	TraceID     string                 `json:"trace_id,omitempty"`
	SpanID      string                 `json:"span_id,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	RetryPolicy *RetryPolicy           `json:"retry_policy,omitempty"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts  int           `json:"max_attempts"`
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
	Multiplier   float64       `json:"multiplier"`
}