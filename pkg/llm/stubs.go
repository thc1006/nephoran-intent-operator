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
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

// StreamingProcessor handles streaming requests with server-sent events
type StreamingProcessor struct {
	httpClient *http.Client
	ragAPIURL  string
	logger     *slog.Logger
	mutex      sync.RWMutex
}

// StreamingRequest represents a streaming request payload
type StreamingRequest struct {
	Query     string `json:"query"`
	ModelName string `json:"model_name,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
	EnableRAG bool   `json:"enable_rag,omitempty"`
}

func NewStreamingProcessor() *StreamingProcessor {
	return &StreamingProcessor{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		ragAPIURL: "http://rag-api:8080",
		logger:    slog.Default().With("component", "streaming-processor"),
	}
}

func (sp *StreamingProcessor) HandleStreamingRequest(w http.ResponseWriter, r *http.Request, req *StreamingRequest) error {
	sp.logger.Info("Handling streaming request", slog.String("query", req.Query))

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create request to RAG API stream endpoint
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	streamURL := sp.ragAPIURL + "/stream"
	httpReq, err := http.NewRequestWithContext(r.Context(), "POST", streamURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create stream request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	// Execute the request
	resp, err := sp.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to connect to RAG API stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("RAG API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Stream the response using bufio.Scanner
	scanner := bufio.NewScanner(resp.Body)
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming not supported")
	}

	for scanner.Scan() {
		line := scanner.Text()
		
		// Forward the SSE event to client
		fmt.Fprintf(w, "%s\n", line)
		
		// Flush after each line for SSE
		if line == "" {
			flusher.Flush()
		}

		// Check if client disconnected
		select {
		case <-r.Context().Done():
			sp.logger.Info("Client disconnected from stream")
			return nil
		default:
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading stream: %w", err)
	}

	sp.logger.Info("Streaming request completed successfully")
	return nil
}

func (sp *StreamingProcessor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"streaming_enabled": true,
		"status":           "active",
		"rag_api_url":      sp.ragAPIURL,
	}
}

func (sp *StreamingProcessor) Shutdown(ctx context.Context) error {
	sp.logger.Info("Shutting down streaming processor")
	return nil
}

// ContextBuilder stub implementation
type ContextBuilder struct{}

func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{}
}

func (cb *ContextBuilder) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"context_builder_enabled": false,
		"status":                 "not_implemented",
	}
}

// RelevanceScorer stub implementation
type RelevanceScorer struct{}

func NewRelevanceScorer() *RelevanceScorer {
	return &RelevanceScorer{}
}

func (rs *RelevanceScorer) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"relevance_scorer_enabled": false,
		"status":                  "not_implemented",
	}
}

// RAGAwarePromptBuilder stub implementation
type RAGAwarePromptBuilder struct{}

func NewRAGAwarePromptBuilder() *RAGAwarePromptBuilder {
	return &RAGAwarePromptBuilder{}
}

func (rpb *RAGAwarePromptBuilder) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"prompt_builder_enabled": false,
		"status":                "not_implemented",
	}
}

// RAGEnhancedProcessor provides LLM processing enhanced with RAG capabilities
type RAGEnhancedProcessor struct {
	baseClient     *Client
	weaviatePool   *rag.WeaviateConnectionPool
	promptEngine   *TelecomPromptEngine
	config         *RAGProcessorConfig
	logger         *slog.Logger
	circuitBreaker *CircuitBreaker
	cache          *ResponseCache
	httpClient     *http.Client
	ragAPIURL      string
	mutex          sync.RWMutex
}

// RAGProcessorConfig holds configuration for the RAG-enhanced processor
type RAGProcessorConfig struct {
	// RAG configuration
	EnableRAG              bool                   `json:"enable_rag"`
	RAGConfidenceThreshold float32                `json:"rag_confidence_threshold"`
	FallbackToBase         bool                   `json:"fallback_to_base"`
	WeaviateURL            string                 `json:"weaviate_url"`
	WeaviateAPIKey         string                 `json:"weaviate_api_key"`
	
	// Query processing
	MaxContextDocuments    int                    `json:"max_context_documents"`
	QueryTimeout           time.Duration          `json:"query_timeout"`
	TelecomKeywords        []string               `json:"telecom_keywords"`
	
	// LLM configuration
	LLMEndpoint            string                 `json:"llm_endpoint"`
	LLMAPIKey              string                 `json:"llm_api_key"`
	LLMModelName           string                 `json:"llm_model_name"`
	MaxTokens              int                    `json:"max_tokens"`
	Temperature            float64                `json:"temperature"`
	
	// Cache and performance
	EnableCaching          bool                   `json:"enable_caching"`
	CacheTTL               time.Duration          `json:"cache_ttl"`
	MaxRetries             int                    `json:"max_retries"`

	// RAG API configuration
	RAGAPIURL              string                 `json:"rag_api_url"`
}

// getDefaultRAGProcessorConfig returns default configuration
func getDefaultRAGProcessorConfig() *RAGProcessorConfig {
	return &RAGProcessorConfig{
		EnableRAG:              true,
		RAGConfidenceThreshold: 0.6,
		FallbackToBase:         true,
		WeaviateURL:            "http://localhost:8080",
		WeaviateAPIKey:         "",
		MaxContextDocuments:    5,
		QueryTimeout:           30 * time.Second,
		TelecomKeywords: []string{
			"5G", "4G", "LTE", "NR", "gNB", "eNB", "AMF", "SMF", "UPF", "AUSF",
			"O-RAN", "RAN", "Core", "Transport", "3GPP", "ETSI", "ITU",
			"URLLC", "eMBB", "mMTC", "NSA", "SA", "PLMN", "TAC", "QCI", "QoS",
			"handover", "mobility", "bearer", "session", "procedure", "interface",
		},
		LLMEndpoint:    "http://localhost:8080/v1/chat/completions",
		LLMAPIKey:      "",
		LLMModelName:   "gpt-4o-mini",
		MaxTokens:      2048,
		Temperature:    0.0,
		EnableCaching:  true,
		CacheTTL:       5 * time.Minute,
		MaxRetries:     3,
		RAGAPIURL:      "http://rag-api:8080",
	}
}

func NewRAGEnhancedProcessor() *RAGEnhancedProcessor {
	return NewRAGEnhancedProcessorWithConfig(nil)
}

func NewRAGEnhancedProcessorWithConfig(config *RAGProcessorConfig) *RAGEnhancedProcessor {
	if config == nil {
		config = getDefaultRAGProcessorConfig()
	}

	// Create HTTP client for RAG API calls
	httpClient := &http.Client{
		Timeout: config.QueryTimeout,
	}

	return &RAGEnhancedProcessor{
		config:     config,
		logger:     slog.Default().With("component", "rag-enhanced-processor"),
		httpClient: httpClient,
		ragAPIURL:  config.RAGAPIURL,
	}
}

func (rep *RAGEnhancedProcessor) ProcessIntent(ctx context.Context, intent string) (string, error) {
	rep.logger.Info("Processing intent", slog.String("intent", intent))

	// Create request payload
	reqPayload := map[string]interface{}{
		"intent": intent,
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request with context timeout
	apiURL := rep.ragAPIURL + "/process"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := rep.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request to RAG API: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check for non-200 status codes
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("RAG API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	result := string(respBody)
	rep.logger.Debug("RAG API response received", slog.Int("response_length", len(result)))

	return result, nil
}

// GetMetrics returns metrics for the RAG processor
func (rep *RAGEnhancedProcessor) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"rag_enabled": rep.config.EnableRAG,
		"status":      "active",
		"rag_api_url": rep.ragAPIURL,
	}
}

// Shutdown gracefully shuts down the processor
func (rep *RAGEnhancedProcessor) Shutdown(ctx context.Context) error {
	rep.logger.Info("Shutting down RAG enhanced processor")
	return nil
}

// Additional methods to maintain compatibility
func (rep *RAGEnhancedProcessor) buildEnhancedPrompt(intent string, contextDocs []map[string]interface{}) string {
	return intent
}

func (rep *RAGEnhancedProcessor) classifyIntentType(intent string) string {
	return "NetworkFunctionDeployment"
}