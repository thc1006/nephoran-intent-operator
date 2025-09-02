//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// CONSOLIDATED INTERFACES - Simplified from over-engineered abstractions.

// LLMProcessor is the main interface for LLM processing.
type LLMProcessor interface {
	ProcessIntent(ctx context.Context, request *ProcessingRequest) (*ProcessingResponse, error)
	GetMetrics() ClientMetrics
	Shutdown()
}

// Processor is an alias for backward compatibility.
type Processor = LLMProcessor

// BatchProcessor handles batch processing of multiple intents.
type BatchProcessor interface {
	ProcessRequest(ctx context.Context, intent, intentType, modelName string, priority Priority) (*BatchResult, error)
	GetStats() BatchProcessorStats
	Close() error
}

// StreamingProcessor handles streaming requests (concrete implementation for disable_rag builds).
type ConsolidatedStreamingProcessor struct {
	// Stub implementation fields.
}

// GetMetrics returns streaming processor metrics (stub implementation).
func (sp *ConsolidatedStreamingProcessor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"active_streams":    0,
		"completed_streams": 0,
		"failed_streams":    0,
	}
}

// ProcessStreaming handles streaming requests (stub implementation).
func (sp *ConsolidatedStreamingProcessor) ProcessStreaming(ctx context.Context, request *StreamingRequest) (<-chan *StreamingResponse, error) {
	// Create a channel and close it immediately for stub implementation
	responseChan := make(chan *StreamingResponse, 1)
	close(responseChan)
	return responseChan, nil
}

// Close closes the streaming processor (stub implementation).
func (sp *ConsolidatedStreamingProcessor) Close() error {
	return nil
}

// Priority enum for batch processing
type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// String returns string representation of priority
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// CONSOLIDATED TYPES - Core types needed across the system.

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

// RequestMetadata contains metadata for requests
type RequestMetadata struct {
	RequestID      string  `json:"request_id"`
	Source         string  `json:"source"`
	TokensUsed     int     `json:"tokens_used"`
	ProcessingTime float64 `json:"processing_time_ms"`
	Cost           float64 `json:"cost"`
	ModelUsed      string  `json:"model_used"`
}

// StreamingRequest represents a request for streaming processing
type StreamingRequest struct {
	ID         string            `json:"id"`
	Intent     string            `json:"intent"`
	IntentType string            `json:"intent_type"`
	Model      string            `json:"model"`
	Context    map[string]string `json:"context,omitempty"`
	Stream     bool              `json:"stream"`
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

// BatchRequest represents a batch processing request
type BatchRequest struct {
	ID         string    `json:"id"`
	Intent     string    `json:"intent"`
	IntentType string    `json:"intent_type"`
	ModelName  string    `json:"model_name"`
	Priority   Priority  `json:"priority"`
	Timestamp  time.Time `json:"timestamp"`
}

// BatchResult represents a batch processing result
type BatchResult struct {
	ID                  string        `json:"id"`
	Result              string        `json:"result"`
	ProcessedParameters string        `json:"processed_parameters"`
	Confidence          float32       `json:"confidence"`
	TokensUsed          int           `json:"tokens_used"`
	ProcessingTime      time.Duration `json:"processing_time"`
	Cost                float64       `json:"cost"`
	ModelUsed           string        `json:"model_used"`
	Error               string        `json:"error,omitempty"`
	Status              string        `json:"status"`
	Timestamp           time.Time     `json:"timestamp"`
}

// BatchProcessorStats represents batch processor statistics
type BatchProcessorStats struct {
	QueueLength        int           `json:"queue_length"`
	ProcessedCount     int64         `json:"processed_count"`
	SuccessCount       int64         `json:"success_count"`
	ErrorCount         int64         `json:"error_count"`
	AverageProcessTime time.Duration `json:"average_process_time"`
	LastProcessedAt    *time.Time    `json:"last_processed_at,omitempty"`
}

// ClientMetrics represents LLM client metrics
type ClientMetrics struct {
	RequestCount       int64         `json:"request_count"`
	SuccessCount       int64         `json:"success_count"`
	ErrorCount         int64         `json:"error_count"`
	AverageLatency     time.Duration `json:"average_latency"`
	TokensUsed         int64         `json:"tokens_used"`
	TotalCost          float64       `json:"total_cost"`
	ActiveConnections  int           `json:"active_connections"`
	LastRequestAt      *time.Time    `json:"last_request_at,omitempty"`
	ModelDistribution map[string]int64 `json:"model_distribution,omitempty"`
}

// CONSOLIDATED INTERFACES - Additional interfaces used by various components.

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
	SessionID   string            `json:"session_id,omitempty"`
	UserID      string            `json:"user_id,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	TraceID     string            `json:"trace_id,omitempty"`
	SpanID      string            `json:"span_id,omitempty"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
	RetryPolicy *RetryPolicy      `json:"retry_policy,omitempty"`
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts int           `json:"max_attempts"`
	InitialDelay time.Duration `json:"initial_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
	Multiplier   float64       `json:"multiplier"`
}

// IMPLEMENTATIONS - Concrete implementations of interfaces.

// SimpleEndpointPool is a basic implementation of EndpointPool
type SimpleEndpointPool struct {
	endpoints       []string
	healthyStatus   map[string]bool
	mu              sync.RWMutex
	currentIndex    int
}

// NewSimpleEndpointPool creates a new SimpleEndpointPool
func NewSimpleEndpointPool(endpoints []string) *SimpleEndpointPool {
	pool := &SimpleEndpointPool{
		endpoints:     make([]string, len(endpoints)),
		healthyStatus: make(map[string]bool),
		currentIndex:  0,
	}
	copy(pool.endpoints, endpoints)
	
	// Initialize all endpoints as healthy
	for _, endpoint := range endpoints {
		pool.healthyStatus[endpoint] = true
	}
	
	return pool
}

// GetEndpoint returns the next available endpoint
func (p *SimpleEndpointPool) GetEndpoint() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	// Find the next healthy endpoint
	startIndex := p.currentIndex
	for {
		endpoint := p.endpoints[p.currentIndex]
		p.currentIndex = (p.currentIndex + 1) % len(p.endpoints)
		
		if p.healthyStatus[endpoint] {
			return endpoint, nil
		}
		
		// If we've cycled through all endpoints and none are healthy
		if p.currentIndex == startIndex {
			return "", fmt.Errorf("no healthy endpoints available")
		}
	}
}

// MarkHealthy marks an endpoint as healthy
func (p *SimpleEndpointPool) MarkHealthy(endpoint string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthyStatus[endpoint] = true
}

// MarkUnhealthy marks an endpoint as unhealthy
func (p *SimpleEndpointPool) MarkUnhealthy(endpoint string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.healthyStatus[endpoint] = false
}

// GetHealthyEndpoints returns all healthy endpoints
func (p *SimpleEndpointPool) GetHealthyEndpoints() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	var healthy []string
	for endpoint, isHealthy := range p.healthyStatus {
		if isHealthy {
			healthy = append(healthy, endpoint)
		}
	}
	return healthy
}

// InMemoryStreamingContextManager is an in-memory implementation of StreamingContextManager
type InMemoryStreamingContextManager struct {
	contexts map[string]*StreamingContext
	mu       sync.RWMutex
	logger   *slog.Logger
}

// NewInMemoryStreamingContextManager creates a new in-memory context manager
func NewInMemoryStreamingContextManager(logger *slog.Logger) *InMemoryStreamingContextManager {
	return &InMemoryStreamingContextManager{
		contexts: make(map[string]*StreamingContext),
		logger:   logger,
	}
}

// CreateContext creates a new streaming context
func (m *InMemoryStreamingContextManager) CreateContext(sessionID string) (*StreamingContext, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.contexts[sessionID]; exists {
		return nil, fmt.Errorf("context already exists for session %s", sessionID)
	}
	
	context := &StreamingContext{
		SessionID:     sessionID,
		History:       make([]string, 0),
		Metadata:      make(map[string]string),
		CreatedAt:     time.Now(),
		LastUpdatedAt: time.Now(),
		TokenCount:    0,
	}
	
	m.contexts[sessionID] = context
	return context, nil
}

// GetContext retrieves a streaming context
func (m *InMemoryStreamingContextManager) GetContext(sessionID string) (*StreamingContext, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	context, exists := m.contexts[sessionID]
	if !exists {
		return nil, fmt.Errorf("context not found for session %s", sessionID)
	}
	
	return context, nil
}

// UpdateContext updates a streaming context with new content
func (m *InMemoryStreamingContextManager) UpdateContext(sessionID string, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	context, exists := m.contexts[sessionID]
	if !exists {
		return fmt.Errorf("context not found for session %s", sessionID)
	}
	
	context.History = append(context.History, content)
	context.LastUpdatedAt = time.Now()
	// Simplified token counting - in real implementation, use proper tokenizer
	context.TokenCount += len(strings.Split(content, " "))
	
	return nil
}

// DeleteContext deletes a streaming context
func (m *InMemoryStreamingContextManager) DeleteContext(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.contexts, sessionID)
	return nil
}

// ListActiveSessions returns all active session IDs
func (m *InMemoryStreamingContextManager) ListActiveSessions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	sessions := make([]string, 0, len(m.contexts))
	for sessionID := range m.contexts {
		sessions = append(sessions, sessionID)
	}
	
	return sessions
}