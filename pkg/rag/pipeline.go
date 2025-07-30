package rag

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// RAGPipeline orchestrates the complete RAG processing pipeline
type RAGPipeline struct {
	// Core components
	documentLoader       *DocumentLoader
	chunkingService      *ChunkingService
	embeddingService     *EmbeddingService
	weaviateClient      *WeaviateClient
	enhancedRetrieval   *EnhancedRetrievalService
	llmClient           shared.ClientInterface
	redisCache          *RedisCache
	monitor             *RAGMonitor

	// Configuration
	config              *PipelineConfig
	logger              *slog.Logger

	// State management
	isInitialized       bool
	isProcessing        bool
	mutex               sync.RWMutex

	// Metrics and monitoring
	startTime           time.Time
	processedDocuments  int64
	processedQueries    int64
	totalChunks         int64
}

// PipelineConfig holds configuration for the entire RAG pipeline
type PipelineConfig struct {
	// Component configurations
	DocumentLoaderConfig    *DocumentLoaderConfig    `json:"document_loader"`
	ChunkingConfig          *ChunkingConfig          `json:"chunking"`
	EmbeddingConfig         *EmbeddingConfig         `json:"embedding"`
	WeaviateConfig          *WeaviateConfig          `json:"weaviate"`
	RetrievalConfig         *RetrievalConfig         `json:"retrieval"`
	RedisCacheConfig        *RedisCacheConfig        `json:"redis_cache"`
	MonitoringConfig        *MonitoringConfig        `json:"monitoring"`

	// Pipeline-specific settings
	EnableCaching           bool          `json:"enable_caching"`
	EnableMonitoring        bool          `json:"enable_monitoring"`
	MaxConcurrentProcessing int           `json:"max_concurrent_processing"`
	ProcessingTimeout       time.Duration `json:"processing_timeout"`
	
	// Knowledge base management
	AutoIndexing            bool          `json:"auto_indexing"`
	IndexingInterval        time.Duration `json:"indexing_interval"`
	
	// Quality assurance
	EnableQualityChecks     bool    `json:"enable_quality_checks"`
	MinQualityThreshold     float32 `json:"min_quality_threshold"`
	
	// Integration settings
	LLMIntegration          bool   `json:"llm_integration"`
	KubernetesIntegration   bool   `json:"kubernetes_integration"`
}

// PipelineStatus represents the current status of the pipeline
type PipelineStatus struct {
	IsHealthy           bool      `json:"is_healthy"`
	IsProcessing        bool      `json:"is_processing"`
	StartTime           time.Time `json:"start_time"`
	Uptime              time.Duration `json:"uptime"`
	ProcessedDocuments  int64     `json:"processed_documents"`
	ProcessedQueries    int64     `json:"processed_queries"`
	TotalChunks         int64     `json:"total_chunks"`
	ComponentStatuses   map[string]string `json:"component_statuses"`
	LastProcessingTime  time.Time `json:"last_processing_time"`
	ErrorCount          int64     `json:"error_count"`
	LastError           string    `json:"last_error,omitempty"`
}

// NewRAGPipeline creates a new RAG pipeline with all components
func NewRAGPipeline(config *PipelineConfig, llmClient shared.ClientInterface) (*RAGPipeline, error) {
	if config == nil {
		config = getDefaultPipelineConfig()
	}

	pipeline := &RAGPipeline{
		config:    config,
		llmClient: llmClient,
		logger:    slog.Default().With("component", "rag-pipeline"),
		startTime: time.Now(),
	}

	// Initialize components in dependency order
	if err := pipeline.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize pipeline components: %w", err)
	}

	// Set up monitoring
	if config.EnableMonitoring {
		if err := pipeline.setupMonitoring(); err != nil {
			pipeline.logger.Warn("Failed to setup monitoring", "error", err)
		}
	}

	// Start background tasks
	if config.AutoIndexing {
		go pipeline.startAutoIndexing()
	}

	pipeline.isInitialized = true
	pipeline.logger.Info("RAG pipeline initialized successfully")

	return pipeline, nil
}

// getDefaultPipelineConfig returns default pipeline configuration
func getDefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		DocumentLoaderConfig:    getDefaultLoaderConfig(),
		ChunkingConfig:          getDefaultChunkingConfig(),
		EmbeddingConfig:         getDefaultEmbeddingConfig(),
		WeaviateConfig:          &WeaviateConfig{
			Host:   "localhost:8080",
			Scheme: "http",
		},
		RetrievalConfig:         getDefaultRetrievalConfig(),
		RedisCacheConfig:        getDefaultRedisCacheConfig(),
		MonitoringConfig:        getDefaultMonitoringConfig(),
		EnableCaching:           true,
		EnableMonitoring:        true,
		MaxConcurrentProcessing: 10,
		ProcessingTimeout:       5 * time.Minute,
		AutoIndexing:            true,
		IndexingInterval:        1 * time.Hour,
		EnableQualityChecks:     true,
		MinQualityThreshold:     0.7,
		LLMIntegration:          true,
		KubernetesIntegration:   true,
	}
}

// initializeComponents initializes all pipeline components
func (rp *RAGPipeline) initializeComponents() error {
	rp.logger.Info("Initializing RAG pipeline components")

	// Initialize document loader
	rp.documentLoader = NewDocumentLoader(rp.config.DocumentLoaderConfig)

	// Initialize chunking service
	rp.chunkingService = NewChunkingService(rp.config.ChunkingConfig)

	// Initialize embedding service
	rp.embeddingService = NewEmbeddingService(rp.config.EmbeddingConfig)

	// Initialize Weaviate client
	var err error
	rp.weaviateClient, err = NewWeaviateClient(rp.config.WeaviateConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize Weaviate client: %w", err)
	}

	// Initialize enhanced retrieval service
	rp.enhancedRetrieval = NewEnhancedRetrievalService(
		rp.weaviateClient,
		rp.embeddingService,
		rp.config.RetrievalConfig,
	)

	// Initialize Redis cache if enabled
	if rp.config.EnableCaching {
		rp.redisCache, err = NewRedisCache(rp.config.RedisCacheConfig)
		if err != nil {
			rp.logger.Warn("Failed to initialize Redis cache", "error", err)
			// Continue without caching
		}
	}

	rp.logger.Info("All RAG pipeline components initialized successfully")
	return nil
}

// setupMonitoring sets up monitoring and health checks
func (rp *RAGPipeline) setupMonitoring() error {
	rp.monitor = NewRAGMonitor(rp.config.MonitoringConfig)

	// Register health checkers for components
	if rp.redisCache != nil {
		rp.monitor.RegisterHealthChecker(&RedisHealthChecker{redisCache: rp.redisCache})
	}

	rp.monitor.RegisterHealthChecker(&EmbeddingServiceHealthChecker{embeddingService: rp.embeddingService})

	rp.logger.Info("Monitoring setup completed")
	return nil
}

// ProcessDocument processes a document through the complete RAG pipeline
func (rp *RAGPipeline) ProcessDocument(ctx context.Context, documentPath string) error {
	if !rp.isInitialized {
		return fmt.Errorf("pipeline not initialized")
	}

	rp.mutex.Lock()
	rp.isProcessing = true
	rp.mutex.Unlock()

	defer func() {
		rp.mutex.Lock()
		rp.isProcessing = false
		rp.mutex.Unlock()
	}()

	startTime := time.Now()
	rp.logger.Info("Starting document processing", "document", documentPath)

	// Monitor processing if monitoring is enabled
	if rp.monitor != nil {
		rp.monitor.RecordDocumentProcessing(1)
		defer rp.monitor.RecordDocumentProcessing(0)
	}

	// Step 1: Load document
	loadStart := time.Now()
	documents, err := rp.documentLoader.LoadDocuments(ctx)
	if err != nil {
		if rp.monitor != nil {
			rp.monitor.RecordQueryError("document_processing", "load_error")
		}
		return fmt.Errorf("failed to load documents: %w", err)
	}
	loadTime := time.Since(loadStart)

	// Step 2: Process documents in parallel
	if err := rp.processDocumentsParallel(ctx, documents); err != nil {
		rp.logger.Error("Failed to process documents in parallel", "error", err)
		return fmt.Errorf("parallel document processing failed: %w", err)
	}

	processingTime := time.Since(startTime)
	
	rp.logger.Info("Document processing completed",
		"document", documentPath,
		"documents_processed", len(documents),
		"load_time", loadTime,
		"total_time", processingTime,
	)

	return nil
}

// processIndividualDocument processes a single document through the pipeline
func (rp *RAGPipeline) processIndividualDocument(ctx context.Context, doc *LoadedDocument) error {
	// Step 1: Chunk the document
	chunkStart := time.Now()
	chunks, err := rp.chunkingService.ChunkDocument(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to chunk document %s: %w", doc.ID, err)
	}
	chunkTime := time.Since(chunkStart)

	if rp.monitor != nil {
		rp.monitor.RecordChunkingLatency(chunkTime, "intelligent")
	}

	rp.totalChunks += int64(len(chunks))

	// Step 2: Generate embeddings for chunks
	embeddingStart := time.Now()
	err = rp.embeddingService.GenerateEmbeddingsForChunks(ctx, chunks)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings for document %s: %w", doc.ID, err)
	}
	embeddingTime := time.Since(embeddingStart)

	if rp.monitor != nil {
		rp.monitor.RecordEmbeddingLatency(embeddingTime, rp.config.EmbeddingConfig.ModelName, len(chunks))
	}

	// Step 3: Store in vector database
	for _, chunk := range chunks {
		telecomDoc := rp.convertChunkToTelecomDocument(chunk)
		if err := rp.weaviateClient.AddDocument(ctx, telecomDoc); err != nil {
			rp.logger.Error("Failed to store chunk in vector database", 
				"chunk_id", chunk.ID, 
				"error", err,
			)
			continue
		}
	}

	// Step 4: Cache document if caching is enabled
	if rp.redisCache != nil {
		if err := rp.redisCache.SetDocument(ctx, doc); err != nil {
			rp.logger.Warn("Failed to cache document", "doc_id", doc.ID, "error", err)
		}
	}

	rp.logger.Debug("Document processed successfully",
		"doc_id", doc.ID,
		"chunks", len(chunks),
		"chunk_time", chunkTime,
		"embedding_time", embeddingTime,
	)

	return nil
}

// processDocumentsParallel processes multiple documents concurrently using a worker pool
func (rp *RAGPipeline) processDocumentsParallel(ctx context.Context, documents []*LoadedDocument) error {
	maxWorkers := rp.config.MaxConcurrentProcessing
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	
	// Limit to reasonable maximum to avoid resource exhaustion
	if maxWorkers > 20 {
		maxWorkers = 20
	}
	
	// Don't create more workers than documents
	if maxWorkers > len(documents) {
		maxWorkers = len(documents)
	}
	
	rp.logger.Info("Starting parallel document processing", 
		"documents", len(documents), 
		"workers", maxWorkers,
	)
	
	// Create channels for work distribution
	docChan := make(chan *LoadedDocument, len(documents))
	resultChan := make(chan error, len(documents))
	
	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			rp.logger.Debug("Starting document processing worker", "worker_id", workerID)
			
			for doc := range docChan {
				select {
				case <-ctx.Done():
					resultChan <- ctx.Err()
					return
				default:
					startTime := time.Now()
					err := rp.processIndividualDocument(ctx, doc)
					processingTime := time.Since(startTime)
					
					if err != nil {
						rp.logger.Error("Worker failed to process document", 
							"worker_id", workerID,
							"doc_id", doc.ID, 
							"error", err,
							"processing_time", processingTime,
						)
						resultChan <- fmt.Errorf("worker %d failed to process doc %s: %w", workerID, doc.ID, err)
					} else {
						rp.logger.Debug("Worker successfully processed document", 
							"worker_id", workerID,
							"doc_id", doc.ID,
							"processing_time", processingTime,
						)
						rp.processedDocuments++
						resultChan <- nil
					}
				}
			}
		}(i)
	}
	
	// Send documents to workers
	go func() {
		defer close(docChan)
		for _, doc := range documents {
			select {
			case docChan <- doc:
			case <-ctx.Done():
				return
			}
		}
	}()
	
	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// Collect results and track errors
	var errors []error
	processedCount := 0
	
	for err := range resultChan {
		if err != nil {
			errors = append(errors, err)
		} else {
			processedCount++
		}
	}
	
	rp.logger.Info("Parallel document processing completed", 
		"total_documents", len(documents),
		"processed_successfully", processedCount,
		"errors", len(errors),
		"workers", maxWorkers,
	)
	
	// If we have some errors but not all failed, log warnings but continue
	if len(errors) > 0 {
		errorRate := float64(len(errors)) / float64(len(documents))
		if errorRate > 0.5 {
			// More than 50% failed - consider this a critical failure
			return fmt.Errorf("parallel processing failed with %d/%d documents failing: %v", len(errors), len(documents), errors[:min(5, len(errors))])
		} else {
			// Less than 50% failed - log warnings but continue
			rp.logger.Warn("Some documents failed processing in parallel mode",
				"failed_count", len(errors),
				"success_count", processedCount,
				"error_rate", errorRate,
			)
		}
	}
	
	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ProcessQuery processes a query through the enhanced retrieval pipeline
func (rp *RAGPipeline) ProcessQuery(ctx context.Context, request *EnhancedSearchRequest) (*EnhancedSearchResponse, error) {
	if !rp.isInitialized {
		return nil, fmt.Errorf("pipeline not initialized")
	}

	startTime := time.Now()
	rp.processedQueries++

	rp.logger.Info("Processing query", 
		"query", request.Query,
		"intent_type", request.IntentType,
	)

	// Check cache first if enabled
	if rp.redisCache != nil && request.UseCache {
		if cached, metadata, found := rp.redisCache.GetContext(ctx, request.Query, request.IntentType); found {
			rp.logger.Debug("Query result found in cache", "query", request.Query)
			
			if rp.monitor != nil {
				rp.monitor.RecordCacheHitRate("query_result", 1.0)
			}

			return &EnhancedSearchResponse{
				Query:              request.Query,
				AssembledContext:   cached,
				ContextMetadata:    metadata,
				ProcessingTime:     time.Since(startTime),
				ProcessedAt:        time.Now(),
			}, nil
		}
		
		if rp.monitor != nil {
			rp.monitor.RecordCacheHitRate("query_result", 0.0)
		}
	}

	// Process query through enhanced retrieval
	response, err := rp.enhancedRetrieval.SearchEnhanced(ctx, request)
	if err != nil {
		if rp.monitor != nil {
			rp.monitor.RecordQueryError(request.IntentType, "retrieval_error")
		}
		return nil, fmt.Errorf("enhanced retrieval failed: %w", err)
	}

	processingTime := time.Since(startTime)

	// Cache the result if caching is enabled
	if rp.redisCache != nil && request.UseCache {
		contextKey := request.IntentType
		if contextKey == "" {
			contextKey = "general"
		}
		
		if err := rp.redisCache.SetContext(ctx, request.Query, contextKey, response.AssembledContext, response.ContextMetadata); err != nil {
			rp.logger.Warn("Failed to cache query result", "query", request.Query, "error", err)
		}
	}

	// Record metrics
	if rp.monitor != nil {
		rp.monitor.RecordQueryLatency(
			processingTime,
			request.IntentType,
			request.EnableQueryEnhancement,
			request.EnableReranking,
		)
		rp.monitor.RecordRetrievalLatency(response.RetrievalTime, "enhanced")
		rp.monitor.RecordContextAssemblyLatency(response.ContextAssemblyTime, "intelligent")
	}

	rp.logger.Info("Query processing completed",
		"query", request.Query,
		"results", len(response.Results),
		"processing_time", processingTime,
		"average_relevance", response.AverageRelevanceScore,
	)

	return response, nil
}

// ProcessIntent processes a high-level intent through the complete RAG pipeline
func (rp *RAGPipeline) ProcessIntent(ctx context.Context, intent string) (string, error) {
	if !rp.config.LLMIntegration || rp.llmClient == nil {
		return "", fmt.Errorf("LLM integration not enabled or client not available")
	}

	rp.logger.Info("Processing intent", "intent", intent)

	// Step 1: Enhance and retrieve relevant context
	searchRequest := &EnhancedSearchRequest{
		Query:                  intent,
		EnableQueryEnhancement: true,
		EnableReranking:        true,
		RequiredContextLength:  rp.config.RetrievalConfig.MaxContextLength,
		UseCache:              rp.config.EnableCaching,
	}

	// Classify intent type (simplified)
	searchRequest.IntentType = rp.classifyIntent(intent)

	// Get enhanced context
	response, err := rp.ProcessQuery(ctx, searchRequest)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve context for intent: %w", err)
	}

	// Step 2: Create enhanced prompt with context
	enhancedPrompt := rp.buildEnhancedPrompt(intent, response.AssembledContext, searchRequest.IntentType)

	// Step 3: Process through LLM
	result, err := rp.llmClient.ProcessIntent(ctx, enhancedPrompt)
	if err != nil {
		return "", fmt.Errorf("LLM processing failed: %w", err)
	}

	rp.logger.Info("Intent processing completed",
		"intent", intent,
		"intent_type", searchRequest.IntentType,
		"context_length", len(response.AssembledContext),
		"response_length", len(result),
	)

	return result, nil
}

// Utility methods

// convertChunkToTelecomDocument converts a DocumentChunk to TelecomDocument
func (rp *RAGPipeline) convertChunkToTelecomDocument(chunk *DocumentChunk) *TelecomDocument {
	return &TelecomDocument{
		ID:              chunk.ID,
		Content:         chunk.CleanContent,
		Title:           chunk.SectionTitle,
		Source:          rp.getSourceFromMetadata(chunk.DocumentMetadata),
		Category:        rp.getCategoryFromMetadata(chunk.DocumentMetadata),
		Version:         rp.getVersionFromMetadata(chunk.DocumentMetadata),
		Keywords:        chunk.TechnicalTerms,
		Confidence:      chunk.QualityScore,
		Language:        "en", // Default to English
		DocumentType:    "chunk",
		NetworkFunction: rp.getNetworkFunctionsFromMetadata(chunk.DocumentMetadata),
		Technology:      rp.getTechnologiesFromMetadata(chunk.DocumentMetadata),
		UseCase:         []string{}, // Could be extracted from chunk analysis
		Timestamp:       chunk.ProcessedAt,
	}
}

// Helper methods for metadata extraction
func (rp *RAGPipeline) getSourceFromMetadata(metadata *DocumentMetadata) string {
	if metadata != nil {
		return metadata.Source
	}
	return "Unknown"
}

func (rp *RAGPipeline) getCategoryFromMetadata(metadata *DocumentMetadata) string {
	if metadata != nil {
		return metadata.Category
	}
	return "General"
}

func (rp *RAGPipeline) getVersionFromMetadata(metadata *DocumentMetadata) string {
	if metadata != nil {
		return metadata.Version
	}
	return ""
}

func (rp *RAGPipeline) getNetworkFunctionsFromMetadata(metadata *DocumentMetadata) []string {
	if metadata != nil {
		return metadata.NetworkFunctions
	}
	return []string{}
}

func (rp *RAGPipeline) getTechnologiesFromMetadata(metadata *DocumentMetadata) []string {
	if metadata != nil {
		return metadata.Technologies
	}
	return []string{}
}

// classifyIntent performs simple intent classification
func (rp *RAGPipeline) classifyIntent(intent string) string {
	intentLower := strings.ToLower(intent)
	
	configKeywords := []string{"configure", "config", "setup", "parameter", "setting"}
	troubleshootKeywords := []string{"problem", "issue", "error", "troubleshoot", "debug", "fix"}
	optimizeKeywords := []string{"optimize", "improve", "performance", "enhance", "tuning"}
	monitorKeywords := []string{"monitor", "metric", "measurement", "kpi", "tracking"}

	for _, keyword := range configKeywords {
		if strings.Contains(intentLower, keyword) {
			return "configuration"
		}
	}

	for _, keyword := range troubleshootKeywords {
		if strings.Contains(intentLower, keyword) {
			return "troubleshooting"
		}
	}

	for _, keyword := range optimizeKeywords {
		if strings.Contains(intentLower, keyword) {
			return "optimization"
		}
	}

	for _, keyword := range monitorKeywords {
		if strings.Contains(intentLower, keyword) {
			return "monitoring"
		}
	}

	return "general"
}

// buildEnhancedPrompt creates an enhanced prompt with RAG context
func (rp *RAGPipeline) buildEnhancedPrompt(intent, context, intentType string) string {
	var promptBuilder strings.Builder

	// System prompt
	promptBuilder.WriteString("You are an expert telecommunications engineer with deep knowledge of 5G, 4G, O-RAN, and network infrastructure. ")
	promptBuilder.WriteString("You provide accurate, actionable responses based on technical documentation and specifications.\n\n")

	// Intent-specific instructions
	switch intentType {
	case "configuration":
		promptBuilder.WriteString("Focus on providing step-by-step configuration guidance with specific parameters and best practices.\n\n")
	case "troubleshooting":
		promptBuilder.WriteString("Focus on diagnostic procedures, root cause analysis, and resolution steps.\n\n")
	case "optimization":
		promptBuilder.WriteString("Focus on performance improvement techniques, tuning recommendations, and optimization strategies.\n\n")
	case "monitoring":
		promptBuilder.WriteString("Focus on monitoring approaches, key metrics, alerting strategies, and measurement techniques.\n\n")
	}

	// Context section
	if context != "" {
		promptBuilder.WriteString("Relevant Technical Documentation:\n")
		promptBuilder.WriteString(context)
		promptBuilder.WriteString("\n\n")
	}

	// User intent
	promptBuilder.WriteString("User Request:\n")
	promptBuilder.WriteString(intent)
	promptBuilder.WriteString("\n\n")

	// Response instructions
	promptBuilder.WriteString("Please provide a comprehensive, accurate response based on the provided documentation. ")
	promptBuilder.WriteString("Include specific references to standards or specifications when applicable. ")
	promptBuilder.WriteString("If the documentation doesn't contain sufficient information for a complete answer, ")
	promptBuilder.WriteString("clearly state what information is missing and provide general guidance based on telecom best practices.")

	return promptBuilder.String()
}

// startAutoIndexing starts the automatic indexing background task
func (rp *RAGPipeline) startAutoIndexing() {
	ticker := time.NewTicker(rp.config.IndexingInterval)
	defer ticker.Stop()

	for range ticker.C {
		rp.logger.Info("Starting automatic indexing")
		ctx, cancel := context.WithTimeout(context.Background(), rp.config.ProcessingTimeout)
		
		// Process documents from configured paths
		for _, path := range rp.config.DocumentLoaderConfig.LocalPaths {
			if err := rp.ProcessDocument(ctx, path); err != nil {
				rp.logger.Error("Auto-indexing failed for path", "path", path, "error", err)
			}
		}
		
		cancel()
		rp.logger.Info("Automatic indexing completed")
	}
}

// Status and management methods

// GetStatus returns the current pipeline status
func (rp *RAGPipeline) GetStatus() PipelineStatus {
	rp.mutex.RLock()
	defer rp.mutex.RUnlock()

	status := PipelineStatus{
		IsHealthy:          rp.isInitialized,
		IsProcessing:       rp.isProcessing,
		StartTime:          rp.startTime,
		Uptime:             time.Since(rp.startTime),
		ProcessedDocuments: rp.processedDocuments,
		ProcessedQueries:   rp.processedQueries,
		TotalChunks:        rp.totalChunks,
		ComponentStatuses:  make(map[string]string),
	}

	// Check component statuses
	if rp.weaviateClient != nil {
		health := rp.weaviateClient.GetHealthStatus()
		if health.IsHealthy {
			status.ComponentStatuses["weaviate"] = "healthy"
		} else {
			status.ComponentStatuses["weaviate"] = "unhealthy"
			status.IsHealthy = false
		}
	}

	if rp.redisCache != nil {
		health := rp.redisCache.GetHealthStatus(context.Background())
		if healthy, ok := health["healthy"].(bool); ok && healthy {
			status.ComponentStatuses["redis"] = "healthy"
		} else {
			status.ComponentStatuses["redis"] = "unhealthy"
		}
	}

	return status
}

// Shutdown gracefully shuts down the pipeline
func (rp *RAGPipeline) Shutdown(ctx context.Context) error {
	rp.logger.Info("Shutting down RAG pipeline")

	// Wait for current processing to complete
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	var errors []error

	// Shutdown components
	if rp.monitor != nil {
		if err := rp.monitor.Shutdown(ctx); err != nil {
			errors = append(errors, fmt.Errorf("monitor shutdown: %w", err))
		}
	}

	if rp.redisCache != nil {
		if err := rp.redisCache.Close(); err != nil {
			errors = append(errors, fmt.Errorf("redis cache shutdown: %w", err))
		}
	}

	if rp.weaviateClient != nil {
		if err := rp.weaviateClient.Close(); err != nil {
			errors = append(errors, fmt.Errorf("weaviate client shutdown: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	rp.logger.Info("RAG pipeline shutdown completed")
	return nil
}