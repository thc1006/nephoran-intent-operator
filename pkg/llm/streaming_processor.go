package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// StreamingProcessor handles Server-Sent Events (SSE) streaming for real-time LLM responses
type StreamingProcessor struct {
	baseClient        ClientInterface
	contextManager    *StreamingContextManager
	tokenManager      *TokenManager
	config            *StreamingConfig
	logger            *slog.Logger
	metrics           *StreamingMetrics
	activeStreams     map[string]*StreamingSession
	mutex             sync.RWMutex
}

// StreamingConfig holds configuration for streaming operations
type StreamingConfig struct {
	// SSE Configuration
	MaxConcurrentStreams int           `json:"max_concurrent_streams"`
	StreamTimeout        time.Duration `json:"stream_timeout"`
	HeartbeatInterval    time.Duration `json:"heartbeat_interval"`
	BufferSize           int           `json:"buffer_size"`
	
	// Context Management
	ContextInjectionOverhead time.Duration `json:"context_injection_overhead"`
	MaxContextUpdates        int           `json:"max_context_updates"`
	ContextUpdateThreshold   float64       `json:"context_update_threshold"`
	
	// Performance Settings
	ChunkSize            int           `json:"chunk_size"`
	MaxChunkDelay        time.Duration `json:"max_chunk_delay"`
	EnableCompression    bool          `json:"enable_compression"`
	
	// Error Handling
	MaxRetries           int           `json:"max_retries"`
	RetryDelay           time.Duration `json:"retry_delay"`
	ErrorRecoveryEnabled bool          `json:"error_recovery_enabled"`
	
	// Client Reconnection
	ReconnectionEnabled  bool          `json:"reconnection_enabled"`
	MaxReconnectAttempts int           `json:"max_reconnect_attempts"`
	ReconnectBackoff     time.Duration `json:"reconnect_backoff"`
}

// StreamingMetrics tracks streaming performance
type StreamingMetrics struct {
	ActiveStreams        int64         `json:"active_streams"`
	TotalStreams         int64         `json:"total_streams"`
	CompletedStreams     int64         `json:"completed_streams"`
	FailedStreams        int64         `json:"failed_streams"`
	AverageStreamTime    time.Duration `json:"average_stream_time"`
	TotalBytesStreamed   int64         `json:"total_bytes_streamed"`
	AverageLatency       time.Duration `json:"average_latency"`
	ContextInjections    int64         `json:"context_injections"`
	Reconnections        int64         `json:"reconnections"`
	HeartbeatsSent       int64         `json:"heartbeats_sent"`
	LastUpdated          time.Time     `json:"last_updated"`
	mutex                sync.RWMutex
}

// StreamingSession represents an active streaming session
type StreamingSession struct {
	ID              string                    `json:"id"`
	Writer          http.ResponseWriter       `json:"-"`
	Flusher         http.Flusher             `json:"-"`
	Context         context.Context           `json:"-"`
	Cancel          context.CancelFunc        `json:"-"`
	StartTime       time.Time                `json:"start_time"`
	LastActivity    time.Time                `json:"last_activity"`
	BytesStreamed   int64                    `json:"bytes_streamed"`
	ChunksStreamed  int64                    `json:"chunks_streamed"`
	ContextUpdates  int                      `json:"context_updates"`
	Status          StreamingStatus          `json:"status"`
	Metadata        map[string]interface{}   `json:"metadata"`
	ErrorCount      int                      `json:"error_count"`
	mutex           sync.RWMutex
}

// StreamingStatus represents the status of a streaming session
type StreamingStatus string

const (
	StatusStreaming  StreamingStatus = "streaming"
	StatusCompleted  StreamingStatus = "completed"
	StatusError      StreamingStatus = "error"
	StatusCancelled  StreamingStatus = "cancelled"
)

// StreamingRequest represents a request for streaming processing
type StreamingRequest struct {
	Query           string                 `json:"query"`
	IntentType      string                 `json:"intent_type"`
	ModelName       string                 `json:"model_name"`
	MaxTokens       int                    `json:"max_tokens"`
	Temperature     float32                `json:"temperature"`
	Context         string                 `json:"context,omitempty"`
	EnableRAG       bool                   `json:"enable_rag"`
	SessionID       string                 `json:"session_id,omitempty"`
	ClientID        string                 `json:"client_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// StreamingChunk represents a chunk of streaming data
type StreamingChunk struct {
	Type        string                 `json:"type"`
	Content     string                 `json:"content,omitempty"`
	Delta       string                 `json:"delta,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	ChunkIndex  int                    `json:"chunk_index"`
	IsComplete  bool                   `json:"is_complete,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string `json:"event,omitempty"`
	Data  string `json:"data"`
	ID    string `json:"id,omitempty"`
	Retry int    `json:"retry,omitempty"`
}

// NewStreamingProcessor creates a new streaming processor
func NewStreamingProcessor(baseClient *Client, tokenManager *TokenManager, config *StreamingConfig) *StreamingProcessor {
	if config == nil {
		config = getDefaultStreamingConfig()
	}
	
	sp := &StreamingProcessor{
		baseClient:     baseClient,
		tokenManager:   tokenManager,
		config:         config,
		logger:         slog.Default().With("component", "streaming-processor"),
		metrics:        &StreamingMetrics{LastUpdated: time.Now()},
		activeStreams:  make(map[string]*StreamingSession),
	}
	
	// Initialize context manager
	sp.contextManager = NewStreamingContextManager(tokenManager, config.ContextInjectionOverhead)
	
	// Start background maintenance
	go sp.maintenanceRoutine()
	
	return sp
}

// getDefaultStreamingConfig returns default streaming configuration
func getDefaultStreamingConfig() *StreamingConfig {
	return &StreamingConfig{
		MaxConcurrentStreams:     100,
		StreamTimeout:            5 * time.Minute,
		HeartbeatInterval:        30 * time.Second,
		BufferSize:               4096,
		ContextInjectionOverhead: 100 * time.Millisecond,
		MaxContextUpdates:        5,
		ContextUpdateThreshold:   0.3,
		ChunkSize:                256,
		MaxChunkDelay:            50 * time.Millisecond,
		EnableCompression:        true,
		MaxRetries:               3,
		RetryDelay:               time.Second,
		ErrorRecoveryEnabled:     true,
		ReconnectionEnabled:      true,
		MaxReconnectAttempts:     5,
		ReconnectBackoff:         2 * time.Second,
	}
}

// HandleStreamingRequest handles an SSE streaming request
func (sp *StreamingProcessor) HandleStreamingRequest(w http.ResponseWriter, r *http.Request, request *StreamingRequest) error {
	// Check concurrent stream limit
	if sp.getActiveStreamCount() >= int64(sp.config.MaxConcurrentStreams) {
		return fmt.Errorf("maximum concurrent streams exceeded")
	}
	
	// Setup SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
	
	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}
	
	// Create session
	sessionID := request.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("stream_%d", time.Now().UnixNano())
	}
	
	ctx, cancel := context.WithTimeout(r.Context(), sp.config.StreamTimeout)
	session := &StreamingSession{
		ID:           sessionID,
		Writer:       w,
		Flusher:      flusher,
		Context:      ctx,
		Cancel:       cancel,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		Status:       StatusStreaming,
		Metadata:     request.Metadata,
	}
	
	// Register session
	sp.registerSession(session)
	defer sp.unregisterSession(sessionID)
	defer cancel()
	
	sp.logger.Info("Starting streaming session",
		"session_id", sessionID,
		"query", request.Query,
		"model", request.ModelName,
	)
	
	// Send initial event
	if err := sp.sendSSEEvent(session, &SSEEvent{
		Event: "start",
		Data:  fmt.Sprintf(`{"session_id":"%s","status":"started"}`, sessionID),
		ID:    "start",
	}); err != nil {
		return err
	}
	
	// Start heartbeat
	heartbeatDone := make(chan bool)
	go sp.heartbeatRoutine(session, heartbeatDone)
	defer func() { heartbeatDone <- true }()
	
	// Process the streaming request
	err := sp.processStreamingRequest(session, request)
	if err != nil {
		sp.logger.Error("Streaming request failed",
			"session_id", sessionID,
			"error", err,
		)
		
		// Send error event
		errorChunk := &StreamingChunk{
			Type:      "error",
			Error:     err.Error(),
			Timestamp: time.Now(),
		}
		sp.sendChunk(session, errorChunk)
		
		session.mutex.Lock()
		session.Status = StatusError
		session.ErrorCount++
		session.mutex.Unlock()
		
		sp.updateMetrics(func(m *StreamingMetrics) {
			m.FailedStreams++
		})
		
		return err
	}
	
	// Send completion event
	completionChunk := &StreamingChunk{
		Type:       "completion",
		IsComplete: true,
		Timestamp:  time.Now(),
		Metadata: map[string]interface{}{
			"total_chunks":    session.ChunksStreamed,
			"total_bytes":     session.BytesStreamed,
			"processing_time": time.Since(session.StartTime).String(),
		},
	}
	sp.sendChunk(session, completionChunk)
	
	session.mutex.Lock()
	session.Status = StatusCompleted
	session.mutex.Unlock()
	
	sp.updateMetrics(func(m *StreamingMetrics) {
		m.CompletedStreams++
		processingTime := time.Since(session.StartTime)
		m.AverageStreamTime = (m.AverageStreamTime*time.Duration(m.CompletedStreams-1) + processingTime) / time.Duration(m.CompletedStreams)
		m.TotalBytesStreamed += session.BytesStreamed
	})
	
	sp.logger.Info("Streaming session completed",
		"session_id", sessionID,
		"chunks_streamed", session.ChunksStreamed,
		"bytes_streamed", session.BytesStreamed,
		"processing_time", time.Since(session.StartTime),
	)
	
	return nil
}

// processStreamingRequest processes the actual streaming request
func (sp *StreamingProcessor) processStreamingRequest(session *StreamingSession, request *StreamingRequest) error {
	// Check if model supports streaming
	if !sp.tokenManager.SupportsStreaming(request.ModelName) {
		return fmt.Errorf("model %s does not support streaming", request.ModelName)
	}
	
	// Prepare context if RAG is enabled
	var ragContext string
	if request.EnableRAG {
		// In a full implementation, this would retrieve RAG context
		// For now, we'll use the provided context
		ragContext = request.Context
		
		// Send context injection event
		contextChunk := &StreamingChunk{
			Type:      "context_injection",
			Content:   "Context retrieved and injected",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"context_length": len(ragContext),
				"injection_time": sp.config.ContextInjectionOverhead.String(),
			},
		}
		sp.sendChunk(session, contextChunk)
		
		// Simulate context injection overhead
		time.Sleep(sp.config.ContextInjectionOverhead)
	}
	
	// Create streaming request for the base client
	// Note: This is a simplified implementation
	// In production, you would need to implement actual streaming for each backend
	
	if streamingClient, ok := sp.baseClient.(StreamingClient); ok {
		return sp.handleClientStreaming(session, request, streamingClient, ragContext)
	} else {
		// Fallback: simulate streaming by chunking non-streaming response
		return sp.simulateStreaming(session, request, ragContext)
	}
}

// StreamingClient interface for clients that support streaming
type StreamingClient interface {
	ProcessIntentStream(context.Context, string, chan<- *StreamingChunk) error
}

// handleClientStreaming handles streaming from a client that supports it
func (sp *StreamingProcessor) handleClientStreaming(session *StreamingSession, request *StreamingRequest, client StreamingClient, ragContext string) error {
	chunkChan := make(chan *StreamingChunk, sp.config.BufferSize)
	
	// Start the streaming process
	go func() {
		defer close(chunkChan)
		
		// Build the full prompt
		prompt := request.Query
		if ragContext != "" {
			prompt = ragContext + "\n\nQuery: " + request.Query
		}
		
		if err := client.ProcessIntentStream(session.Context, prompt, chunkChan); err != nil {
			sp.logger.Error("Client streaming failed", "error", err)
			chunkChan <- &StreamingChunk{
				Type:      "error",
				Error:     err.Error(),
				Timestamp: time.Now(),
			}
		}
	}()
	
	// Process chunks as they arrive
	chunkIndex := 0
	for {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				return nil // Channel closed, streaming complete
			}
			
			chunk.ChunkIndex = chunkIndex
			chunkIndex++
			
			if err := sp.sendChunk(session, chunk); err != nil {
				return fmt.Errorf("failed to send chunk: %w", err)
			}
			
			// Check for errors
			if chunk.Type == "error" {
				return fmt.Errorf("streaming error: %s", chunk.Error)
			}
			
		case <-session.Context.Done():
			return session.Context.Err()
		}
	}
}

// simulateStreaming simulates streaming by chunking a complete response
func (sp *StreamingProcessor) simulateStreaming(session *StreamingSession, request *StreamingRequest, ragContext string) error {
	// Build the full prompt
	prompt := request.Query
	if ragContext != "" {
		prompt = ragContext + "\n\nQuery: " + request.Query
	}
	
	// Get complete response from base client
	response, err := sp.baseClient.ProcessIntent(session.Context, prompt)
	if err != nil {
		return fmt.Errorf("base client processing failed: %w", err)
	}
	
	// Chunk the response and stream it
	chunks := sp.chunkResponse(response)
	
	for i, chunkContent := range chunks {
		chunk := &StreamingChunk{
			Type:       "content",
			Delta:      chunkContent,
			Timestamp:  time.Now(),
			ChunkIndex: i,
		}
		
		if err := sp.sendChunk(session, chunk); err != nil {
			return fmt.Errorf("failed to send chunk: %w", err)
		}
		
		// Add small delay to simulate real streaming
		if sp.config.MaxChunkDelay > 0 {
			time.Sleep(sp.config.MaxChunkDelay)
		}
		
		// Check for cancellation
		select {
		case <-session.Context.Done():
			return session.Context.Err()
		default:
		}
	}
	
	return nil
}

// chunkResponse splits a response into chunks for streaming
func (sp *StreamingProcessor) chunkResponse(response string) []string {
	if len(response) <= sp.config.ChunkSize {
		return []string{response}
	}
	
	var chunks []string
	words := strings.Fields(response)
	currentChunk := ""
	
	for _, word := range words {
		if len(currentChunk)+len(word)+1 > sp.config.ChunkSize {
			if currentChunk != "" {
				chunks = append(chunks, currentChunk)
				currentChunk = word
			} else {
				// Word is longer than chunk size, split it
				chunks = append(chunks, word[:sp.config.ChunkSize])
				currentChunk = word[sp.config.ChunkSize:]
			}
		} else {
			if currentChunk != "" {
				currentChunk += " "
			}
			currentChunk += word
		}
	}
	
	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}
	
	return chunks
}

// sendChunk sends a chunk as an SSE event
func (sp *StreamingProcessor) sendChunk(session *StreamingSession, chunk *StreamingChunk) error {
	chunkData, err := json.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("failed to marshal chunk: %w", err)
	}
	
	event := &SSEEvent{
		Event: "chunk",
		Data:  string(chunkData),
		ID:    fmt.Sprintf("chunk_%d", chunk.ChunkIndex),
	}
	
	if err := sp.sendSSEEvent(session, event); err != nil {
		return err
	}
	
	// Update session metrics
	session.mutex.Lock()
	session.BytesStreamed += int64(len(chunkData))
	session.ChunksStreamed++
	session.LastActivity = time.Now()
	session.mutex.Unlock()
	
	return nil
}

// sendSSEEvent sends a Server-Sent Event
func (sp *StreamingProcessor) sendSSEEvent(session *StreamingSession, event *SSEEvent) error {
	var eventStr strings.Builder
	
	if event.ID != "" {
		eventStr.WriteString(fmt.Sprintf("id: %s\n", event.ID))
	}
	if event.Event != "" {
		eventStr.WriteString(fmt.Sprintf("event: %s\n", event.Event))
	}
	if event.Retry > 0 {
		eventStr.WriteString(fmt.Sprintf("retry: %d\n", event.Retry))
	}
	
	// Handle multi-line data
	lines := strings.Split(event.Data, "\n")
	for _, line := range lines {
		eventStr.WriteString(fmt.Sprintf("data: %s\n", line))
	}
	eventStr.WriteString("\n")
	
	// Write to response
	if _, err := session.Writer.Write([]byte(eventStr.String())); err != nil {
		return fmt.Errorf("failed to write SSE event: %w", err)
	}
	
	session.Flusher.Flush()
	return nil
}

// heartbeatRoutine sends periodic heartbeats to keep the connection alive
func (sp *StreamingProcessor) heartbeatRoutine(session *StreamingSession, done <-chan bool) {
	ticker := time.NewTicker(sp.config.HeartbeatInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			heartbeat := &SSEEvent{
				Event: "heartbeat",
				Data:  fmt.Sprintf(`{"timestamp":"%s"}`, time.Now().Format(time.RFC3339)),
			}
			
			if err := sp.sendSSEEvent(session, heartbeat); err != nil {
				sp.logger.Warn("Failed to send heartbeat",
					"session_id", session.ID,
					"error", err,
				)
				return
			}
			
			sp.updateMetrics(func(m *StreamingMetrics) {
				m.HeartbeatsSent++
			})
			
		case <-done:
			return
		case <-session.Context.Done():
			return
		}
	}
}

// maintenanceRoutine performs background maintenance tasks
func (sp *StreamingProcessor) maintenanceRoutine() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		sp.cleanupExpiredSessions()
		sp.updateActiveStreamCount()
	}
}

// cleanupExpiredSessions removes expired or stale sessions
func (sp *StreamingProcessor) cleanupExpiredSessions() {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()
	
	now := time.Now()
	for sessionID, session := range sp.activeStreams {
		session.mutex.RLock()
		expired := now.Sub(session.LastActivity) > sp.config.StreamTimeout
		cancelled := session.Context.Err() != nil
		session.mutex.RUnlock()
		
		if expired || cancelled {
			session.Cancel()
			delete(sp.activeStreams, sessionID)
			
			sp.logger.Debug("Cleaned up session",
				"session_id", sessionID,
				"expired", expired,
				"cancelled", cancelled,
			)
		}
	}
}

// registerSession registers a new active session
func (sp *StreamingProcessor) registerSession(session *StreamingSession) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()
	
	sp.activeStreams[session.ID] = session
	
	sp.updateMetrics(func(m *StreamingMetrics) {
		m.TotalStreams++
		m.ActiveStreams++
	})
}

// unregisterSession removes a session from active sessions
func (sp *StreamingProcessor) unregisterSession(sessionID string) {
	sp.mutex.Lock()
	defer sp.mutex.Unlock()
	
	if _, exists := sp.activeStreams[sessionID]; exists {
		delete(sp.activeStreams, sessionID)
		
		sp.updateMetrics(func(m *StreamingMetrics) {
			m.ActiveStreams--
		})
	}
}

// getActiveStreamCount returns the current number of active streams
func (sp *StreamingProcessor) getActiveStreamCount() int64 {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()
	return int64(len(sp.activeStreams))
}

// updateActiveStreamCount updates the active stream count in metrics
func (sp *StreamingProcessor) updateActiveStreamCount() {
	count := sp.getActiveStreamCount()
	sp.updateMetrics(func(m *StreamingMetrics) {
		m.ActiveStreams = count
		m.LastUpdated = time.Now()
	})
}

// GetActiveSession returns information about an active session
func (sp *StreamingProcessor) GetActiveSession(sessionID string) (*StreamingSession, bool) {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()
	
	session, exists := sp.activeStreams[sessionID]
	if !exists {
		return nil, false
	}
	
	// Return a copy to prevent external modification
	session.mutex.RLock()
	sessionCopy := &StreamingSession{
		ID:             session.ID,
		StartTime:      session.StartTime,
		LastActivity:   session.LastActivity,
		BytesStreamed:  session.BytesStreamed,
		ChunksStreamed: session.ChunksStreamed,
		ContextUpdates: session.ContextUpdates,
		Status:         session.Status,
		Metadata:       session.Metadata,
		ErrorCount:     session.ErrorCount,
	}
	session.mutex.RUnlock()
	
	return sessionCopy, true
}

// CancelSession cancels an active streaming session
func (sp *StreamingProcessor) CancelSession(sessionID string) error {
	sp.mutex.RLock()
	session, exists := sp.activeStreams[sessionID]
	sp.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}
	
	session.Cancel()
	
	session.mutex.Lock()
	session.Status = StatusCancelled
	session.mutex.Unlock()
	
	sp.logger.Info("Session cancelled", "session_id", sessionID)
	return nil
}

// updateMetrics safely updates metrics
func (sp *StreamingProcessor) updateMetrics(updater func(*StreamingMetrics)) {
	sp.metrics.mutex.Lock()
	defer sp.metrics.mutex.Unlock()
	updater(sp.metrics)
}

// GetMetrics returns current streaming metrics
func (sp *StreamingProcessor) GetMetrics() *StreamingMetrics {
	sp.metrics.mutex.RLock()
	defer sp.metrics.mutex.RUnlock()
	
	metrics := *sp.metrics
	return &metrics
}

// GetConfig returns the current configuration
func (sp *StreamingProcessor) GetConfig() *StreamingConfig {
	sp.mutex.RLock()
	defer sp.mutex.RUnlock()
	
	config := *sp.config
	return &config
}

// Close gracefully shuts down the streaming processor
func (sp *StreamingProcessor) Close() error {
	sp.logger.Info("Shutting down streaming processor")
	
	// Cancel all active sessions
	sp.mutex.RLock()
	sessions := make([]*StreamingSession, 0, len(sp.activeStreams))
	for _, session := range sp.activeStreams {
		sessions = append(sessions, session)
	}
	sp.mutex.RUnlock()
	
	for _, session := range sessions {
		session.Cancel()
	}
	
	// Wait for sessions to clean up
	time.Sleep(time.Second)
	
	if sp.contextManager != nil {
		sp.contextManager.Close()
	}
	
	sp.logger.Info("Streaming processor shutdown complete")
	return nil
}