//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ProcessingEngine provides unified processing for batch, streaming, and RAG requests.
type ProcessingEngine struct {
	// Core components.
	baseClient   *Client
	promptEngine *TelecomPromptEngine
	config       *ProcessingConfig
	logger       *slog.Logger

	// HTTP client for external APIs.
	httpClient *http.Client
	ragAPIURL  string

	// Smart endpoints from configuration.
	processEndpoint string
	streamEndpoint  string
	healthEndpoint  string

	// Processing state.
	mutex          sync.RWMutex
	activeStreams  map[string]*StreamContext
	batchQueue     chan *ProcessingBatchRequest
	batchProcessor *batchProcessor

	// Metrics and monitoring.
	metrics *ProcessingMetrics
}

// ProcessingConfig holds configuration for the processing engine.
type ProcessingConfig struct {
	// RAG configuration.
	EnableRAG              bool    `json:"enable_rag"`
	RAGAPIURL              string  `json:"rag_api_url"`
	RAGConfidenceThreshold float32 `json:"rag_confidence_threshold"`
	FallbackToBase         bool    `json:"fallback_to_base"`

	// Batch processing.
	EnableBatching      bool          `json:"enable_batching"`
	MinBatchSize        int           `json:"min_batch_size"`
	MaxBatchSize        int           `json:"max_batch_size"`
	BatchTimeout        time.Duration `json:"batch_timeout"`
	SimilarityThreshold float64       `json:"similarity_threshold"`

	// Streaming configuration.
	EnableStreaming  bool          `json:"enable_streaming"`
	StreamBufferSize int           `json:"stream_buffer_size"`
	StreamTimeout    time.Duration `json:"stream_timeout"`

	// General processing.
	QueryTimeout          time.Duration `json:"query_timeout"`
	MaxConcurrentRequests int           `json:"max_concurrent_requests"`
	EnableCaching         bool          `json:"enable_caching"`
}

// ProcessingMetrics tracks processing performance.
type ProcessingMetrics struct {
	TotalRequests   int64         `json:"total_requests"`
	BatchedRequests int64         `json:"batched_requests"`
	StreamRequests  int64         `json:"stream_requests"`
	RAGRequests     int64         `json:"rag_requests"`
	AverageLatency  time.Duration `json:"average_latency"`
	ThroughputRPS   float64       `json:"throughput_rps"`
	CacheHitRate    float64       `json:"cache_hit_rate"`
	BatchEfficiency float64       `json:"batch_efficiency"`
	mutex           sync.RWMutex
}

// StreamContext represents an active streaming session.
type StreamContext struct {
	ID        string
	StartTime time.Time
	Writer    http.ResponseWriter
	Context   context.Context
	Cancel    context.CancelFunc
	Flusher   http.Flusher
}

// ProcessingBatchRequest represents a request for batch processing.
type ProcessingBatchRequest struct {
	ID         string                 `json:"id"`
	Intent     string                 `json:"intent"`
	IntentType string                 `json:"intent_type"`
	Parameters map[string]interface{} `json:"parameters"`
	Priority   int                    `json:"priority"`
	SubmitTime time.Time              `json:"submit_time"`
	ResponseCh chan *ProcessingResult `json:"-"`
	Context    context.Context        `json:"-"`
}

// ProcessingResult represents the result of processing.
type ProcessingResult struct {
	Content        string                 `json:"content"`
	TokensUsed     int                    `json:"tokens_used"`
	ProcessingTime time.Duration          `json:"processing_time"`
	CacheHit       bool                   `json:"cache_hit"`
	Batched        bool                   `json:"batched"`
	Metadata       map[string]interface{} `json:"metadata"`
	Error          error                  `json:"error,omitempty"`
}

// ProcessingStreamingRequest represents a streaming request payload.
type ProcessingStreamingRequest struct {
	Query     string `json:"query"`
	ModelName string `json:"model_name,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
	EnableRAG bool   `json:"enable_rag,omitempty"`
}

// batchProcessor handles batch processing internally.
type batchProcessor struct {
	config          *ProcessingConfig
	baseClient      *Client
	processingQueue chan *ProcessingBatchRequest
	activeBatches   map[string][]*ProcessingBatchRequest
	mutex           sync.RWMutex
	logger          *slog.Logger
	stopCh          chan struct{}
	wg              sync.WaitGroup
}

// NewProcessingEngine creates a new unified processing engine.
func NewProcessingEngine(baseClient *Client, config *ProcessingConfig) *ProcessingEngine {
	if config == nil {
		config = getDefaultProcessingConfig()
	}

	// Create HTTP client for external APIs.
	httpClient := &http.Client{
		Timeout: config.QueryTimeout,
	}

	processor := &ProcessingEngine{
		baseClient:    baseClient,
		promptEngine:  NewTelecomPromptEngine(),
		config:        config,
		logger:        slog.Default().With("component", "processing-engine"),
		httpClient:    httpClient,
		ragAPIURL:     config.RAGAPIURL,
		activeStreams: make(map[string]*StreamContext),
		batchQueue:    make(chan *ProcessingBatchRequest, 1000),
		metrics:       &ProcessingMetrics{},
	}

	// Initialize smart endpoints.
	processor.initializeEndpoints()

	// Initialize batch processor if enabled.
	if config.EnableBatching {
		processor.batchProcessor = &batchProcessor{
			config:          config,
			baseClient:      baseClient,
			processingQueue: processor.batchQueue,
			activeBatches:   make(map[string][]*ProcessingBatchRequest),
			logger:          processor.logger.With("component", "batch-processor"),
			stopCh:          make(chan struct{}),
		}

		// Start batch processing workers.
		for i := 0; i < 3; i++ {
			go processor.batchProcessor.processingWorker()
		}
	}

	return processor
}

// initializeEndpoints initializes smart endpoints based on configuration.
func (pe *ProcessingEngine) initializeEndpoints() {
	// For ProcessingEngine, we need to work with ProcessingConfig.
	// Since ProcessingConfig doesn't have GetEffectiveRAGEndpoints, we implement the logic here.
	baseURL := strings.TrimSuffix(pe.ragAPIURL, "/")

	// Determine process endpoint based on URL pattern.
	if strings.HasSuffix(pe.ragAPIURL, "/process_intent") {
		// Legacy pattern - use as configured.
		pe.processEndpoint = pe.ragAPIURL
	} else if strings.HasSuffix(pe.ragAPIURL, "/process") {
		// New pattern - use as configured.
		pe.processEndpoint = pe.ragAPIURL
	} else {
		// Base URL pattern - default to /process for new installations.
		pe.processEndpoint = baseURL + "/process"
	}

	// Streaming endpoint.
	processBase := baseURL
	if strings.HasSuffix(pe.processEndpoint, "/process_intent") {
		processBase = strings.TrimSuffix(pe.processEndpoint, "/process_intent")
	} else if strings.HasSuffix(pe.processEndpoint, "/process") {
		processBase = strings.TrimSuffix(pe.processEndpoint, "/process")
	}
	pe.streamEndpoint = processBase + "/stream"
	pe.healthEndpoint = processBase + "/health"

	pe.logger.Info("Initialized smart endpoints",
		slog.String("process_endpoint", pe.processEndpoint),
		slog.String("stream_endpoint", pe.streamEndpoint),
		slog.String("health_endpoint", pe.healthEndpoint),
	)
}

// getDefaultProcessingConfig returns default processing configuration.
func getDefaultProcessingConfig() *ProcessingConfig {
	return &ProcessingConfig{
		EnableRAG:              true,
		RAGAPIURL:              "http://rag-api:8080",
		RAGConfidenceThreshold: 0.6,
		FallbackToBase:         true,

		EnableBatching:      true,
		MinBatchSize:        2,
		MaxBatchSize:        10,
		BatchTimeout:        100 * time.Millisecond,
		SimilarityThreshold: 0.8,

		EnableStreaming:  true,
		StreamBufferSize: 4096,
		StreamTimeout:    30 * time.Second,

		QueryTimeout:          30 * time.Second,
		MaxConcurrentRequests: 100,
		EnableCaching:         true,
	}
}

// ProcessIntent processes an intent with the appropriate method (single, batch, or streaming).
func (pe *ProcessingEngine) ProcessIntent(ctx context.Context, intent string) (*ProcessingResult, error) {
	start := time.Now()
	pe.updateMetrics(func(m *ProcessingMetrics) {
		m.TotalRequests++
	})

	pe.logger.Debug("Processing intent", slog.String("intent", intent))

	// Determine processing method based on configuration and context.
	if pe.config.EnableRAG {
		return pe.processWithRAG(ctx, intent, start)
	}

	// Fall back to base client processing.
	return pe.processWithBaseClient(ctx, intent, start)
}

// ProcessBatch processes multiple intents as a batch.
func (pe *ProcessingEngine) ProcessBatch(ctx context.Context, requests []*ProcessingBatchRequest) ([]*ProcessingResult, error) {
	if !pe.config.EnableBatching {
		return pe.processIndividually(ctx, requests)
	}

	pe.logger.Debug("Processing batch", slog.Int("count", len(requests)))

	// Group similar requests if similarity batching is enabled.
	groups := pe.groupSimilarRequests(requests)

	results := make([]*ProcessingResult, len(requests))
	var wg sync.WaitGroup

	for _, group := range groups {
		wg.Add(1)
		go func(batch []*ProcessingBatchRequest) {
			defer wg.Done()
			pe.processBatchGroup(ctx, batch, results)
		}(group)
	}

	wg.Wait()
	return results, nil
}

// HandleStreamingRequest handles server-sent events streaming.
func (pe *ProcessingEngine) HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *StreamingRequest) error {
	if !pe.config.EnableStreaming {
		return fmt.Errorf("streaming not enabled")
	}

	pe.logger.Info("Handling streaming request", slog.String("query", req.Query))

	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get flusher for SSE.
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	// Create stream context.
	streamCtx, cancel := context.WithTimeout(r.Context(), pe.config.StreamTimeout)
	defer cancel()

	streamID := fmt.Sprintf("stream_%d", time.Now().UnixNano())
	streamContext := &StreamContext{
		ID:        streamID,
		StartTime: time.Now(),
		Writer:    w,
		Context:   streamCtx,
		Cancel:    cancel,
		Flusher:   flusher,
	}

	// Register stream.
	pe.registerStream(streamID, streamContext)
	defer pe.unregisterStream(streamID)

	// Process streaming request.
	return pe.processStreamingRequest(streamContext, req)
}

// processWithRAG processes intent using RAG API.
func (pe *ProcessingEngine) processWithRAG(ctx context.Context, intent string, startTime time.Time) (*ProcessingResult, error) {
	pe.updateMetrics(func(m *ProcessingMetrics) {
		m.RAGRequests++
	})

	// Create request payload.
	reqPayload := map[string]interface{}{
		"intent": intent,
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request using smart endpoint.
	if pe.processEndpoint == "" {
		return nil, fmt.Errorf("process endpoint not initialized")
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", pe.processEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Execute the request.
	resp, err := pe.httpClient.Do(httpReq)
	if err != nil {
		if pe.config.FallbackToBase {
			pe.logger.Warn("RAG API failed, falling back to base client", slog.String("error", err.Error()))
			return pe.processWithBaseClient(ctx, intent, startTime)
		}
		return nil, fmt.Errorf("failed to send request to RAG API: %w", err)
	}
	defer resp.Body.Close()

	// Read response.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if pe.config.FallbackToBase {
			pe.logger.Warn("RAG API returned error, falling back to base client", slog.Int("status", resp.StatusCode))
			return pe.processWithBaseClient(ctx, intent, startTime)
		}
		return nil, fmt.Errorf("RAG API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	processingTime := time.Since(startTime)
	return &ProcessingResult{
		Content:        string(respBody),
		ProcessingTime: processingTime,
		CacheHit:       false,
		Batched:        false,
		Metadata: map[string]interface{}{
			"method":  "rag",
			"api_url": pe.processEndpoint,
		},
	}, nil
}

// processWithBaseClient processes intent using the base LLM client.
func (pe *ProcessingEngine) processWithBaseClient(ctx context.Context, intent string, startTime time.Time) (*ProcessingResult, error) {
	result, err := pe.baseClient.ProcessIntent(ctx, intent)
	if err != nil {
		return &ProcessingResult{Error: err}, err
	}

	processingTime := time.Since(startTime)
	return &ProcessingResult{
		Content:        result,
		ProcessingTime: processingTime,
		CacheHit:       false,
		Batched:        false,
		Metadata: map[string]interface{}{
			"method": "base_client",
		},
	}, nil
}

// processStreamingRequest processes a streaming request.
func (pe *ProcessingEngine) processStreamingRequest(streamCtx *StreamContext, req *StreamingRequest) error {
	pe.updateMetrics(func(m *ProcessingMetrics) {
		m.StreamRequests++
	})

	// Create request to RAG API stream endpoint.
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	if pe.streamEndpoint == "" {
		return fmt.Errorf("stream endpoint not initialized")
	}

	httpReq, err := http.NewRequestWithContext(streamCtx.Context, "POST", pe.streamEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create stream request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Execute the request.
	resp, err := pe.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to connect to RAG API stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("RAG API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Stream the response.
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-streamCtx.Context.Done():
			return streamCtx.Context.Err()
		default:
			line := scanner.Text()
			fmt.Fprintf(streamCtx.Writer, "%s\n", line)

			if line == "" {
				streamCtx.Flusher.Flush()
			}
		}
	}

	return scanner.Err()
}

// processIndividually processes requests individually when batching is disabled.
func (pe *ProcessingEngine) processIndividually(ctx context.Context, requests []*ProcessingBatchRequest) ([]*ProcessingResult, error) {
	results := make([]*ProcessingResult, len(requests))
	var wg sync.WaitGroup

	for i, req := range requests {
		wg.Add(1)
		go func(idx int, request *ProcessingBatchRequest) {
			defer wg.Done()
			result, _ := pe.ProcessIntent(ctx, request.Intent)
			results[idx] = result
		}(i, req)
	}

	wg.Wait()
	return results, nil
}

// groupSimilarRequests groups similar requests for batch processing.
func (pe *ProcessingEngine) groupSimilarRequests(requests []*ProcessingBatchRequest) [][]*ProcessingBatchRequest {
	// Simple grouping by intent similarity.
	groups := make([][]*ProcessingBatchRequest, 0)
	processed := make(map[int]bool)

	for i, req1 := range requests {
		if processed[i] {
			continue
		}

		group := []*ProcessingBatchRequest{req1}
		processed[i] = true

		// Find similar requests.
		for j, req2 := range requests {
			if i != j && !processed[j] {
				if pe.calculateSimilarity(req1.Intent, req2.Intent) > pe.config.SimilarityThreshold {
					group = append(group, req2)
					processed[j] = true
				}
			}
		}

		groups = append(groups, group)
	}

	return groups
}

// calculateSimilarity calculates similarity between two intents (simplified).
func (pe *ProcessingEngine) calculateSimilarity(intent1, intent2 string) float64 {
	// Simple word-based similarity calculation.
	words1 := strings.Fields(strings.ToLower(intent1))
	words2 := strings.Fields(strings.ToLower(intent2))

	common := 0
	wordMap := make(map[string]bool)

	for _, word := range words1 {
		wordMap[word] = true
	}

	for _, word := range words2 {
		if wordMap[word] {
			common++
		}
	}

	totalWords := len(words1) + len(words2)
	if totalWords == 0 {
		return 0.0
	}

	return float64(common*2) / float64(totalWords)
}

// processBatchGroup processes a group of similar requests.
func (pe *ProcessingEngine) processBatchGroup(ctx context.Context, batch []*ProcessingBatchRequest, results []*ProcessingResult) {
	pe.updateMetrics(func(m *ProcessingMetrics) {
		m.BatchedRequests += int64(len(batch))
	})

	// For now, process each request individually.
	// In a more sophisticated implementation, this could optimize by combining similar requests.
	for _, req := range batch {
		result, _ := pe.ProcessIntent(ctx, req.Intent)
		if result != nil {
			result.Batched = true
		}
		// Note: This simplified implementation assumes results array has correct indexing.
		// In production, you'd need proper index mapping.
	}
}

// registerStream registers an active streaming session.
func (pe *ProcessingEngine) registerStream(streamID string, streamCtx *StreamContext) {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	pe.activeStreams[streamID] = streamCtx
}

// unregisterStream removes a streaming session.
func (pe *ProcessingEngine) unregisterStream(streamID string) {
	pe.mutex.Lock()
	defer pe.mutex.Unlock()
	delete(pe.activeStreams, streamID)
}

// updateMetrics safely updates processing metrics.
func (pe *ProcessingEngine) updateMetrics(updater func(*ProcessingMetrics)) {
	pe.metrics.mutex.Lock()
	defer pe.metrics.mutex.Unlock()
	updater(pe.metrics)
}

// GetMetrics returns current processing metrics.
func (pe *ProcessingEngine) GetMetrics() *ProcessingMetrics {
	pe.metrics.mutex.RLock()
	defer pe.metrics.mutex.RUnlock()

	// Create a copy without the mutex.
	metrics := &ProcessingMetrics{
		TotalRequests:   pe.metrics.TotalRequests,
		BatchedRequests: pe.metrics.BatchedRequests,
		StreamRequests:  pe.metrics.StreamRequests,
		RAGRequests:     pe.metrics.RAGRequests,
		AverageLatency:  pe.metrics.AverageLatency,
		ThroughputRPS:   pe.metrics.ThroughputRPS,
		CacheHitRate:    pe.metrics.CacheHitRate,
		BatchEfficiency: pe.metrics.BatchEfficiency,
	}
	return metrics
}

// Shutdown gracefully shuts down the processing engine.
func (pe *ProcessingEngine) Shutdown(ctx context.Context) error {
	pe.logger.Info("Shutting down processing engine")

	// Stop batch processor if running.
	if pe.batchProcessor != nil {
		close(pe.batchProcessor.stopCh)
		pe.batchProcessor.wg.Wait()
	}

	// Cancel all active streams.
	pe.mutex.Lock()
	for streamID, streamCtx := range pe.activeStreams {
		pe.logger.Debug("Canceling active stream", slog.String("stream_id", streamID))
		streamCtx.Cancel()
	}
	pe.mutex.Unlock()

	return nil
}

// processingWorker processes batch requests.
func (bp *batchProcessor) processingWorker() {
	bp.wg.Add(1)
	defer bp.wg.Done()

	for {
		select {
		case <-bp.stopCh:
			return
		case req := <-bp.processingQueue:
			// Process individual batch request.
			result, err := bp.baseClient.ProcessIntent(req.Context, req.Intent)

			processingResult := &ProcessingResult{
				Content:        result,
				ProcessingTime: time.Since(req.SubmitTime),
				Batched:        true,
				Error:          err,
			}

			// Send result back.
			select {
			case req.ResponseCh <- processingResult:
			case <-req.Context.Done():
				// Request context cancelled.
			}
		}
	}
}
