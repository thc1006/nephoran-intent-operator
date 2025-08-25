package llm

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// TokenManager manages token allocation and tracking
type TokenManager interface {
	AllocateTokens(request string) (int, error)
	ReleaseTokens(count int) error
	GetAvailableTokens() int
}

// StreamingContextManager manages streaming request contexts
type StreamingContextManager struct {
	activeStreams map[string]*StreamingContext
	mutex         sync.RWMutex
}

// StreamingContext holds context for a streaming request
type StreamingContext struct {
	SessionID string
	StartTime time.Time
	LastSeen  time.Time
	Metadata  map[string]interface{}
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
	Field    string `json:"field"`
	Message  string `json:"message"`
	Code     string `json:"code"`
	Severity string `json:"severity"`
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

// generateRequestID generates a unique request identifier
func generateRequestID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
