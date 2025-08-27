//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

// RAGEnhancedProcessor provides LLM processing enhanced with RAG capabilities
type RAGEnhancedProcessor struct {
	baseClient     *Client
	ragService     *rag.RAGService
	weaviateClient *rag.WeaviateClient
	config         *RAGProcessorConfig
	logger         *slog.Logger
	metrics        *ProcessorMetrics
	mutex          sync.RWMutex
}

// RAGProcessorConfig holds configuration for the RAG-enhanced processor
type RAGProcessorConfig struct {
	// RAG configuration
	EnableRAG              bool    `json:"enable_rag"`
	RAGConfidenceThreshold float32 `json:"rag_confidence_threshold"`
	FallbackToBase         bool    `json:"fallback_to_base"`

	// Query classification
	EnableQueryClassification bool    `json:"enable_query_classification"`
	ClassificationThreshold   float32 `json:"classification_threshold"`

	// Performance settings
	MaxConcurrentQueries int           `json:"max_concurrent_queries"`
	QueryTimeout         time.Duration `json:"query_timeout"`

	// Intent type mappings
	IntentTypeMapping map[string]string `json:"intent_type_mapping"`

	// Telecom domain settings
	TelecomKeywords []string `json:"telecom_keywords"`
	RequiredSources []string `json:"required_sources"`
}

// ProcessorMetrics holds metrics for the RAG processor
type ProcessorMetrics struct {
	TotalRequests      int64         `json:"total_requests"`
	RAGRequests        int64         `json:"rag_requests"`
	BaseRequests       int64         `json:"base_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	AverageLatency     time.Duration `json:"average_latency"`
	RAGLatency         time.Duration `json:"rag_latency"`
	BaseLatency        time.Duration `json:"base_latency"`
	LastUpdated        time.Time     `json:"last_updated"`
	mutex              sync.RWMutex
}

// EnhancedResponse represents a response with RAG metadata
type EnhancedResponse struct {
	Content        string                 `json:"content"`
	UsedRAG        bool                   `json:"used_rag"`
	Confidence     float32                `json:"confidence"`
	Sources        []*rag.SearchResult    `json:"sources,omitempty"`
	ProcessingTime time.Duration          `json:"processing_time"`
	IntentType     string                 `json:"intent_type,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// NewRAGEnhancedProcessor creates a new RAG-enhanced LLM processor
func NewRAGEnhancedProcessor(
	baseClient *Client,
	weaviateClient *rag.WeaviateClient,
	ragService *rag.RAGService,
	config *RAGProcessorConfig,
) *RAGEnhancedProcessor {
	if config == nil {
		config = getDefaultRAGProcessorConfig()
	}

	return &RAGEnhancedProcessor{
		baseClient:     baseClient,
		ragService:     ragService,
		weaviateClient: weaviateClient,
		config:         config,
		logger:         slog.Default().With("component", "rag-enhanced-processor"),
		metrics:        &ProcessorMetrics{LastUpdated: time.Now()},
	}
}

// getDefaultRAGProcessorConfig returns default configuration
func getDefaultRAGProcessorConfig() *RAGProcessorConfig {
	return &RAGProcessorConfig{
		EnableRAG:                 true,
		RAGConfidenceThreshold:    0.6,
		FallbackToBase:            true,
		EnableQueryClassification: true,
		ClassificationThreshold:   0.7,
		MaxConcurrentQueries:      10,
		QueryTimeout:              60 * time.Second,
		IntentTypeMapping: map[string]string{
			"configure":    "configuration",
			"optimize":     "optimization",
			"troubleshoot": "troubleshooting",
			"monitor":      "monitoring",
			"setup":        "configuration",
			"tune":         "optimization",
			"debug":        "troubleshooting",
			"check":        "monitoring",
		},
		TelecomKeywords: []string{
			"5G", "4G", "LTE", "NR", "gNB", "eNB", "AMF", "SMF", "UPF", "AUSF",
			"O-RAN", "RAN", "Core", "Transport", "3GPP", "ETSI", "ITU",
			"URLLC", "eMBB", "mMTC", "NSA", "SA", "PLMN", "TAC", "QCI", "QoS",
			"handover", "mobility", "bearer", "session", "procedure", "interface",
		},
		RequiredSources: []string{"3GPP", "O-RAN", "ETSI"},
	}
}

// ProcessIntent processes an intent using RAG enhancement when appropriate
func (rep *RAGEnhancedProcessor) ProcessIntent(ctx context.Context, intent string) (string, error) {
	startTime := time.Now()

	// Update metrics
	rep.updateMetrics(func(m *ProcessorMetrics) {
		m.TotalRequests++
	})

	// Classify the query to determine if RAG is beneficial
	shouldUseRAG := rep.shouldUseRAG(intent)

	var response *EnhancedResponse
	var err error

	if shouldUseRAG && rep.config.EnableRAG {
		// Use RAG-enhanced processing
		response, err = rep.processWithRAG(ctx, intent)
		if err != nil && rep.config.FallbackToBase {
			rep.logger.Warn("RAG processing failed, falling back to base client", "error", err)
			response, err = rep.processWithBase(ctx, intent)
		}
	} else {
		// Use base processing
		response, err = rep.processWithBase(ctx, intent)
	}

	if err != nil {
		rep.updateMetrics(func(m *ProcessorMetrics) {
			m.FailedRequests++
		})
		return "", err
	}

	// Update success metrics
	processingTime := time.Since(startTime)
	rep.updateMetrics(func(m *ProcessorMetrics) {
		m.SuccessfulRequests++
		m.AverageLatency = (m.AverageLatency*time.Duration(m.SuccessfulRequests-1) + processingTime) / time.Duration(m.SuccessfulRequests)
		if response.UsedRAG {
			m.RAGRequests++
			m.RAGLatency = (m.RAGLatency*time.Duration(m.RAGRequests-1) + processingTime) / time.Duration(m.RAGRequests)
		} else {
			m.BaseRequests++
			m.BaseLatency = (m.BaseLatency*time.Duration(m.BaseRequests-1) + processingTime) / time.Duration(m.BaseRequests)
		}
		m.LastUpdated = time.Now()
	})

	// Return just the content for backward compatibility
	return response.Content, nil
}

// ProcessIntentEnhanced processes an intent and returns enhanced response with metadata
func (rep *RAGEnhancedProcessor) ProcessIntentEnhanced(ctx context.Context, intent string) (*EnhancedResponse, error) {
	startTime := time.Now()

	rep.updateMetrics(func(m *ProcessorMetrics) {
		m.TotalRequests++
	})

	shouldUseRAG := rep.shouldUseRAG(intent)

	var response *EnhancedResponse
	var err error

	if shouldUseRAG && rep.config.EnableRAG {
		response, err = rep.processWithRAG(ctx, intent)
		if err != nil && rep.config.FallbackToBase {
			rep.logger.Warn("RAG processing failed, falling back to base client", "error", err)
			response, err = rep.processWithBase(ctx, intent)
		}
	} else {
		response, err = rep.processWithBase(ctx, intent)
	}

	if err != nil {
		rep.updateMetrics(func(m *ProcessorMetrics) {
			m.FailedRequests++
		})
		return nil, err
	}

	response.ProcessingTime = time.Since(startTime)

	rep.updateMetrics(func(m *ProcessorMetrics) {
		m.SuccessfulRequests++
		m.AverageLatency = (m.AverageLatency*time.Duration(m.SuccessfulRequests-1) + response.ProcessingTime) / time.Duration(m.SuccessfulRequests)
		if response.UsedRAG {
			m.RAGRequests++
			m.RAGLatency = (m.RAGLatency*time.Duration(m.RAGRequests-1) + response.ProcessingTime) / time.Duration(m.RAGRequests)
		} else {
			m.BaseRequests++
			m.BaseLatency = (m.BaseLatency*time.Duration(m.BaseRequests-1) + response.ProcessingTime) / time.Duration(m.BaseRequests)
		}
		m.LastUpdated = time.Now()
	})

	return response, nil
}

// shouldUseRAG determines if a query should use RAG enhancement
func (rep *RAGEnhancedProcessor) shouldUseRAG(intent string) bool {
	if !rep.config.EnableRAG {
		return false
	}

	// Check if the query contains telecom-specific keywords
	intentLower := strings.ToLower(intent)
	keywordCount := 0

	for _, keyword := range rep.config.TelecomKeywords {
		if strings.Contains(intentLower, strings.ToLower(keyword)) {
			keywordCount++
		}
	}

	// Use RAG if we found telecom keywords or if query classification suggests it
	if keywordCount > 0 {
		return true
	}

	// Additional heuristics for RAG usage
	ragIndicators := []string{
		"how to", "what is", "explain", "configure", "setup", "troubleshoot",
		"optimize", "monitor", "standard", "specification", "procedure",
		"interface", "protocol", "parameter", "algorithm", "implementation",
	}

	for _, indicator := range ragIndicators {
		if strings.Contains(intentLower, indicator) {
			return true
		}
	}

	return false
}

// processWithRAG processes the intent using RAG enhancement
func (rep *RAGEnhancedProcessor) processWithRAG(ctx context.Context, intent string) (*EnhancedResponse, error) {
	rep.logger.Info("Processing intent with RAG enhancement", "intent", intent)

	// Classify intent type
	intentType := rep.classifyIntentType(intent)

	// Create RAG request
	ragRequest := &rag.RAGRequest{
		Query:             intent,
		IntentType:        intentType,
		MaxResults:        10,
		MinConfidence:     rep.config.RAGConfidenceThreshold,
		UseHybridSearch:   true,
		EnableReranking:   true,
		IncludeSourceRefs: true,
	}

	// Add telecom-specific filters
	ragRequest.SearchFilters = map[string]interface{}{
		"confidence": map[string]interface{}{
			"operator": "GreaterThan",
			"value":    rep.config.RAGConfidenceThreshold,
		},
	}

	// Process with RAG
	ragResponse, err := rep.ragService.ProcessQuery(ctx, ragRequest)
	if err != nil {
		return nil, fmt.Errorf("RAG processing failed: %w", err)
	}

	// Check if RAG response meets confidence threshold
	if ragResponse.Confidence < rep.config.RAGConfidenceThreshold {
		rep.logger.Warn("RAG response confidence below threshold",
			"confidence", ragResponse.Confidence,
			"threshold", rep.config.RAGConfidenceThreshold,
		)

		if rep.config.FallbackToBase {
			return rep.processWithBase(ctx, intent)
		}
	}

	return &EnhancedResponse{
		Content:    ragResponse.Answer,
		UsedRAG:    true,
		Confidence: ragResponse.Confidence,
		Sources:    ragResponse.SourceDocuments,
		IntentType: intentType,
		Metadata: map[string]interface{}{
			"rag_retrieval_time":  ragResponse.RetrievalTime,
			"rag_generation_time": ragResponse.GenerationTime,
			"documents_used":      len(ragResponse.SourceDocuments),
			"used_cache":          ragResponse.UsedCache,
		},
	}, nil
}

// processWithBase processes the intent using only the base LLM client
func (rep *RAGEnhancedProcessor) processWithBase(ctx context.Context, intent string) (*EnhancedResponse, error) {
	rep.logger.Info("Processing intent with base client", "intent", intent)

	response, err := rep.baseClient.ProcessIntent(ctx, intent)
	if err != nil {
		return nil, fmt.Errorf("base client processing failed: %w", err)
	}

	return &EnhancedResponse{
		Content:    response,
		UsedRAG:    false,
		Confidence: 0.8, // Default confidence for base responses
		IntentType: rep.classifyIntentType(intent),
		Metadata: map[string]interface{}{
			"method": "base_llm",
		},
	}, nil
}

// classifyIntentType attempts to classify the intent type
func (rep *RAGEnhancedProcessor) classifyIntentType(intent string) string {
	intentLower := strings.ToLower(intent)

	// Check predefined mappings
	for keyword, intentType := range rep.config.IntentTypeMapping {
		if strings.Contains(intentLower, keyword) {
			return intentType
		}
	}

	// Default classification based on common patterns
	if strings.Contains(intentLower, "configure") || strings.Contains(intentLower, "setup") {
		return "configuration"
	}
	if strings.Contains(intentLower, "optimize") || strings.Contains(intentLower, "improve") {
		return "optimization"
	}
	if strings.Contains(intentLower, "troubleshoot") || strings.Contains(intentLower, "debug") || strings.Contains(intentLower, "error") {
		return "troubleshooting"
	}
	if strings.Contains(intentLower, "monitor") || strings.Contains(intentLower, "check") || strings.Contains(intentLower, "status") {
		return "monitoring"
	}

	return "general"
}

// AddTelecomDocument adds a document to the knowledge base
func (rep *RAGEnhancedProcessor) AddTelecomDocument(ctx context.Context, doc *rag.TelecomDocument) error {
	if rep.weaviateClient == nil {
		return fmt.Errorf("weaviate client not available")
	}

	return rep.weaviateClient.AddDocument(ctx, doc)
}

// SearchKnowledgeBase searches the knowledge base directly
func (rep *RAGEnhancedProcessor) SearchKnowledgeBase(ctx context.Context, query *rag.SearchQuery) (*rag.SearchResponse, error) {
	if rep.weaviateClient == nil {
		return nil, fmt.Errorf("weaviate client not available")
	}

	return rep.weaviateClient.Search(ctx, query)
}

// GetHealth returns the health status of the processor and its dependencies
func (rep *RAGEnhancedProcessor) GetHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status":      "healthy",
		"rag_enabled": rep.config.EnableRAG,
		"metrics":     rep.GetMetrics(),
	}

	// Add base client health if available - disabled for now
	// TODO: Add GetHealth method to Client or use interface
	// if healthChecker, ok := rep.baseClient.(interface{ GetHealth() map[string]interface{} }); ok {
	//	health["base_client"] = healthChecker.GetHealth()
	// }

	// Add RAG service health
	if rep.ragService != nil {
		health["rag_service"] = rep.ragService.GetHealth()
	}

	// Add Weaviate health
	if rep.weaviateClient != nil {
		weaviateHealth := rep.weaviateClient.GetHealthStatus()
		health["weaviate"] = map[string]interface{}{
			"healthy":    weaviateHealth.IsHealthy,
			"version":    weaviateHealth.Version,
			"last_check": weaviateHealth.LastCheck,
		}
	}

	return health
}

// GetMetrics returns current processor metrics
func (rep *RAGEnhancedProcessor) GetMetrics() *ProcessorMetrics {
	rep.metrics.mutex.RLock()
	defer rep.metrics.mutex.RUnlock()

	// Return a copy
	metrics := *rep.metrics
	return &metrics
}

// updateMetrics safely updates metrics
func (rep *RAGEnhancedProcessor) updateMetrics(updater func(*ProcessorMetrics)) {
	rep.metrics.mutex.Lock()
	defer rep.metrics.mutex.Unlock()
	updater(rep.metrics)
}

// ValidateConfig validates the processor configuration
func (rep *RAGEnhancedProcessor) ValidateConfig() error {
	if rep.config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	if rep.config.EnableRAG && rep.ragService == nil {
		return fmt.Errorf("RAG is enabled but RAG service is not available")
	}

	if rep.config.EnableRAG && rep.weaviateClient == nil {
		return fmt.Errorf("RAG is enabled but Weaviate client is not available")
	}

	if rep.config.RAGConfidenceThreshold < 0 || rep.config.RAGConfidenceThreshold > 1 {
		return fmt.Errorf("RAG confidence threshold must be between 0 and 1")
	}

	if rep.config.MaxConcurrentQueries <= 0 {
		return fmt.Errorf("max concurrent queries must be positive")
	}

	return nil
}

// Close cleans up resources
func (rep *RAGEnhancedProcessor) Close() error {
	rep.logger.Info("Closing RAG enhanced processor")

	if rep.weaviateClient != nil {
		if err := rep.weaviateClient.Close(); err != nil {
			rep.logger.Error("Failed to close Weaviate client", "error", err)
		}
	}

	// Close base client if it supports closing - disabled for now
	// TODO: Add Close method to Client or use interface
	// if closer, ok := rep.baseClient.(interface{ Close() error }); ok {
	//	if err := closer.Close(); err != nil {
	//		rep.logger.Error("Failed to close base client", "error", err)
	//	}
	// }

	return nil
}

// Note: RAGEnhancedProcessor is a struct that works with Client, not implementing Client interface
// Removing interface assertion since Client is a concrete type, not an interface
