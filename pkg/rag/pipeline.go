//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// RAGPipeline orchestrates the complete RAG processing pipeline
type RAGPipeline struct {
	// Core components
	documentLoader    *DocumentLoader
	chunkingService   *ChunkingService
	embeddingService  *EmbeddingService
	weaviateClient    *WeaviateClient
	weaviatePool      *WeaviateConnectionPool
	enhancedRetrieval *EnhancedRetrievalService
	llmClient         shared.ClientInterface
	redisCache        *RedisCache
	monitor           *RAGMonitor

	// Configuration
	config *PipelineConfig
	logger *slog.Logger

	// State management
	isInitialized bool
	isProcessing  bool
	mutex         sync.RWMutex

	// Async processing components
	documentQueue   chan DocumentJob
	queryQueue      chan QueryJob
	workerPool      *AsyncWorkerPool
	resultCallbacks map[string]func(interface{}, error)
	callbackMutex   sync.RWMutex

	// Metrics and monitoring
	startTime          time.Time
	processedDocuments int64
	processedQueries   int64
	totalChunks        int64
	activeJobs         int64
	queuedJobs         int64
}

// PipelineConfig holds configuration for the entire RAG pipeline
type PipelineConfig struct {
	// Component configurations
	DocumentLoaderConfig *DocumentLoaderConfig `json:"document_loader"`
	ChunkingConfig       *ChunkingConfig       `json:"chunking"`
	EmbeddingConfig      *EmbeddingConfig      `json:"embedding"`
	WeaviateConfig       *WeaviateConfig       `json:"weaviate"`
	WeaviatePoolConfig   *PoolConfig           `json:"weaviate_pool"`
	RetrievalConfig      *RetrievalConfig      `json:"retrieval"`
	RedisCacheConfig     *RedisCacheConfig     `json:"redis_cache"`
	MonitoringConfig     *MonitoringConfig     `json:"monitoring"`

	// Pipeline-specific settings
	EnableCaching           bool          `json:"enable_caching"`
	EnableMonitoring        bool          `json:"enable_monitoring"`
	MaxConcurrentProcessing int           `json:"max_concurrent_processing"`
	ProcessingTimeout       time.Duration `json:"processing_timeout"`

	// Async processing settings
	AsyncProcessing   bool `json:"async_processing"`
	DocumentQueueSize int  `json:"document_queue_size"`
	QueryQueueSize    int  `json:"query_queue_size"`
	WorkerPoolSize    int  `json:"worker_pool_size"`
	BatchSize         int  `json:"batch_size"`
	MaxRetries        int  `json:"max_retries"`

	// Knowledge base management
	AutoIndexing     bool          `json:"auto_indexing"`
	IndexingInterval time.Duration `json:"indexing_interval"`

	// Quality assurance
	EnableQualityChecks bool    `json:"enable_quality_checks"`
	MinQualityThreshold float32 `json:"min_quality_threshold"`

	// Integration settings
	LLMIntegration        bool `json:"llm_integration"`
	KubernetesIntegration bool `json:"kubernetes_integration"`
}

// PipelineStatus represents the current status of the pipeline
type PipelineStatus struct {
	IsHealthy          bool              `json:"is_healthy"`
	IsProcessing       bool              `json:"is_processing"`
	StartTime          time.Time         `json:"start_time"`
	Uptime             time.Duration     `json:"uptime"`
	ProcessedDocuments int64             `json:"processed_documents"`
	ProcessedQueries   int64             `json:"processed_queries"`
	TotalChunks        int64             `json:"total_chunks"`
	ComponentStatuses  map[string]string `json:"component_statuses"`
	LastProcessingTime time.Time         `json:"last_processing_time"`
	ErrorCount         int64             `json:"error_count"`
	LastError          string            `json:"last_error,omitempty"`
	ActiveJobs         int64             `json:"active_jobs"`
	QueuedJobs         int64             `json:"queued_jobs"`
	AsyncEnabled       bool              `json:"async_enabled"`
}

// DocumentJob represents an async document processing job
type DocumentJob struct {
	ID         string
	Documents  []Document
	Callback   func([]ProcessedDocument, error)
	Context    context.Context
	StartTime  time.Time
	RetryCount int
	MaxRetries int
}

// QueryJob represents an async query processing job
type QueryJob struct {
	ID         string
	Query      string
	Parameters QueryParameters
	Callback   func(QueryResult, error)
	Context    context.Context
	StartTime  time.Time
	RetryCount int
	MaxRetries int
}

// AsyncWorkerPool manages async processing workers
type AsyncWorkerPool struct {
	workers     []*AsyncWorker
	workerCount int
	jobQueue    chan AsyncJob
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	metrics     *AsyncMetrics
}

// AsyncWorker processes jobs asynchronously
type AsyncWorker struct {
	id       int
	pool     *AsyncWorkerPool
	pipeline *RAGPipeline
	logger   *slog.Logger
}

// AsyncJob represents a generic async job
type AsyncJob struct {
	Type    string
	Payload interface{}
	Process func(ctx context.Context, payload interface{}) error
}

// AsyncMetrics tracks async processing metrics
type AsyncMetrics struct {
	TotalJobs      int64
	CompletedJobs  int64
	FailedJobs     int64
	ActiveWorkers  int64
	AverageLatency time.Duration
	mutex          sync.RWMutex
}

// ProcessedDocument represents a processed document result
type ProcessedDocument struct {
	ID          string
	Chunks      []ProcessedChunk
	Embeddings  [][]float32
	Metadata    map[string]interface{}
	ProcessTime time.Duration
	Error       error
}

// ProcessedChunk represents a processed chunk result
type ProcessedChunk struct {
	Content    string
	Embedding  []float32
	Metadata   map[string]interface{}
	ChunkIndex int
	Quality    float32
}

// QueryResult represents the result of a query operation
type QueryResult struct {
	ID          string
	Results     []RetrievedDocument
	Response    string
	Confidence  float32
	ProcessTime time.Duration
	CacheHit    bool
	Metadata    map[string]interface{}
}

// RetrievedDocument represents a document retrieved from the query
type RetrievedDocument struct {
	ID       string
	Content  string
	Score    float32
	Source   string
	Metadata map[string]interface{}
}

// QueryParameters contains parameters for query processing
type QueryParameters struct {
	MaxResults      int
	MinScore        float32
	FilterCriteria  map[string]interface{}
	IncludeMetadata bool
	Timeout         time.Duration
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
		DocumentLoaderConfig: getDefaultLoaderConfig(),
		ChunkingConfig:       getDefaultChunkingConfig(),
		EmbeddingConfig:      getDefaultEmbeddingConfig(),
		WeaviateConfig: &WeaviateConfig{
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
		AsyncProcessing:         true,
		DocumentQueueSize:       1000,
		QueryQueueSize:          5000,
		WorkerPoolSize:          runtime.NumCPU(),
		BatchSize:               50,
		MaxRetries:              3,
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

	// Load secrets from files
	if err := LoadRAGPipelineSecrets(rp.config, rp.logger); err != nil {
		rp.logger.Warn("Failed to load some secrets from files", "error", err.Error())
	}

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

	// Initialize async processing components if enabled
	if rp.config.AsyncProcessing {
		err = rp.initializeAsyncProcessing()
		if err != nil {
			return fmt.Errorf("failed to initialize async processing: %w", err)
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
	// TODO: Implement proper document storage without TelecomDocument conversion
	for _, chunk := range chunks {
		// telecomDoc := rp.convertChunkToTelecomDocument(chunk)
		// if err := rp.weaviateClient.AddDocument(ctx, telecomDoc); err != nil {
		// 	rp.logger.Error("Failed to store chunk in vector database",
		// 		"chunk_id", chunk.ID,
		// 		"error", err,
		// 	)
		// 	continue
		// }
		rp.logger.Debug("Chunk processed but not stored in vector database yet", "chunk_id", chunk.ID)
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
	if rp.redisCache != nil && rp.config.EnableCaching {
		if cached, metadata, found := rp.redisCache.GetContext(ctx, request.Query, request.IntentType); found {
			rp.logger.Debug("Query result found in cache", "query", request.Query)

			if rp.monitor != nil {
				rp.monitor.RecordCacheHitRate("query_result", 1.0)
			}

			return &EnhancedSearchResponse{
				Query:            request.Query,
				AssembledContext: cached,
				ContextMetadata:  metadata,
				ProcessingTime:   time.Since(startTime),
				ProcessedAt:      time.Now(),
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
	if rp.redisCache != nil && rp.config.EnableCaching {
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
// TODO: Define TelecomDocument struct or remove this function if not needed
// func (rp *RAGPipeline) convertChunkToTelecomDocument(chunk *DocumentChunk) *TelecomDocument {
// 	return &TelecomDocument{
// 		ID:              chunk.ID,
// 		Content:         chunk.CleanContent,
// 		Title:           chunk.SectionTitle,
// 		Source:          rp.getSourceFromMetadata(chunk.DocumentMetadata),
// 		Category:        rp.getCategoryFromMetadata(chunk.DocumentMetadata),
// 		Version:         rp.getVersionFromMetadata(chunk.DocumentMetadata),
// 		Keywords:        chunk.TechnicalTerms,
// 		Confidence:      float32(chunk.QualityScore),
// 		Language:        "en", // Default to English
// 		DocumentType:    "chunk",
// 		NetworkFunction: rp.getNetworkFunctionsFromMetadata(chunk.DocumentMetadata),
// 		Technology:      rp.getTechnologiesFromMetadata(chunk.DocumentMetadata),
// 		UseCase:         []string{}, // Could be extracted from chunk analysis
// 		CreatedAt:       chunk.ProcessedAt,
// 		UpdatedAt:       chunk.ProcessedAt,
// 	}
// }

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

// initializeAsyncProcessing initializes async processing components
func (rp *RAGPipeline) initializeAsyncProcessing() error {
	rp.logger.Info("Initializing async processing components")

	// Initialize queues
	rp.documentQueue = make(chan DocumentJob, rp.config.DocumentQueueSize)
	rp.queryQueue = make(chan QueryJob, rp.config.QueryQueueSize)
	rp.resultCallbacks = make(map[string]func(interface{}, error))

	// Initialize worker pool
	ctx, cancel := context.WithCancel(context.Background())
	rp.workerPool = &AsyncWorkerPool{
		workerCount: rp.config.WorkerPoolSize,
		jobQueue:    make(chan AsyncJob, rp.config.WorkerPoolSize*10),
		ctx:         ctx,
		cancel:      cancel,
		metrics: &AsyncMetrics{
			TotalJobs:      0,
			CompletedJobs:  0,
			FailedJobs:     0,
			ActiveWorkers:  0,
			AverageLatency: 0,
		},
	}

	// Create workers
	rp.workerPool.workers = make([]*AsyncWorker, rp.config.WorkerPoolSize)
	for i := 0; i < rp.config.WorkerPoolSize; i++ {
		worker := &AsyncWorker{
			id:       i,
			pool:     rp.workerPool,
			pipeline: rp,
			logger:   rp.logger.With("worker_id", i),
		}
		rp.workerPool.workers[i] = worker
	}

	// Start workers
	for _, worker := range rp.workerPool.workers {
		rp.workerPool.wg.Add(1)
		go worker.start()
	}

	// Start job dispatchers
	go rp.processDocumentJobs()
	go rp.processQueryJobs()

	rp.logger.Info("Async processing initialized",
		"worker_count", rp.config.WorkerPoolSize,
		"document_queue_size", rp.config.DocumentQueueSize,
		"query_queue_size", rp.config.QueryQueueSize,
	)

	return nil
}

// ProcessDocumentsAsync processes documents asynchronously
func (rp *RAGPipeline) ProcessDocumentsAsync(ctx context.Context, jobID string, documents []Document, callback func([]ProcessedDocument, error)) error {
	if !rp.config.AsyncProcessing {
		return rp.processDocumentsSync(ctx, documents, callback)
	}

	job := DocumentJob{
		ID:         jobID,
		Documents:  documents,
		Callback:   callback,
		Context:    ctx,
		StartTime:  time.Now(),
		MaxRetries: rp.config.MaxRetries,
	}

	select {
	case rp.documentQueue <- job:
		rp.updateQueueMetrics(1, 0)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("document queue is full")
	}
}

// ProcessQueryAsync processes a query asynchronously
func (rp *RAGPipeline) ProcessQueryAsync(ctx context.Context, jobID string, query string, params QueryParameters, callback func(QueryResult, error)) error {
	if !rp.config.AsyncProcessing {
		return rp.processQuerySync(ctx, query, params, callback)
	}

	job := QueryJob{
		ID:         jobID,
		Query:      query,
		Parameters: params,
		Callback:   callback,
		Context:    ctx,
		StartTime:  time.Now(),
		MaxRetries: rp.config.MaxRetries,
	}

	select {
	case rp.queryQueue <- job:
		rp.updateQueueMetrics(0, 1)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("query queue is full")
	}
}

// processDocumentJobs processes document jobs from the queue
func (rp *RAGPipeline) processDocumentJobs() {
	for job := range rp.documentQueue {
		asyncJob := AsyncJob{
			Type:    "document",
			Payload: job,
			Process: rp.processDocumentJob,
		}

		select {
		case rp.workerPool.jobQueue <- asyncJob:
			rp.updateQueueMetrics(-1, 0)
		case <-rp.workerPool.ctx.Done():
			return
		}
	}
}

// processQueryJobs processes query jobs from the queue
func (rp *RAGPipeline) processQueryJobs() {
	for job := range rp.queryQueue {
		asyncJob := AsyncJob{
			Type:    "query",
			Payload: job,
			Process: rp.processQueryJob,
		}

		select {
		case rp.workerPool.jobQueue <- asyncJob:
			rp.updateQueueMetrics(0, -1)
		case <-rp.workerPool.ctx.Done():
			return
		}
	}
}

// processDocumentJob processes a single document job
func (rp *RAGPipeline) processDocumentJob(ctx context.Context, payload interface{}) error {
	job, ok := payload.(DocumentJob)
	if !ok {
		return fmt.Errorf("invalid document job payload")
	}

	startTime := time.Now()
	processedDocs := make([]ProcessedDocument, 0, len(job.Documents))

	for _, doc := range job.Documents {
		// Process document through pipeline
		chunks, err := rp.chunkingService.ChunkDocument(ctx, &LoadedDocument{
			ID:       doc.ID,
			Content:  doc.Content,
			Metadata: &DocumentMetadata{},
		})
		if err != nil {
			processedDocs = append(processedDocs, ProcessedDocument{
				ID:    doc.ID,
				Error: fmt.Errorf("chunking failed: %w", err),
			})
			continue
		}

		// Generate embeddings for chunks
		processedChunks := make([]ProcessedChunk, 0, len(chunks))
		embeddings := make([][]float32, 0, len(chunks))

		for i, chunk := range chunks {
			embeddingResp, err := rp.embeddingService.GenerateEmbeddings(ctx, &EmbeddingRequest{
				Texts: []string{chunk.Content},
			})
			if err != nil {
				rp.logger.Warn("Failed to generate embedding for chunk", "chunk_index", i, "error", err)
				continue
			}

			if len(embeddingResp.Embeddings) > 0 {
				processedChunks = append(processedChunks, ProcessedChunk{
					Content:    chunk.Content,
					Embedding:  embeddingResp.Embeddings[0],
					Metadata:   make(map[string]interface{}),
					ChunkIndex: i,
					Quality:    float32(chunk.QualityScore),
				})
				embeddings = append(embeddings, embeddingResp.Embeddings[0])
			}
		}

		processedDocs = append(processedDocs, ProcessedDocument{
			ID:          doc.ID,
			Chunks:      processedChunks,
			Embeddings:  embeddings,
			Metadata:    doc.Metadata,
			ProcessTime: time.Since(startTime),
		})
	}

	// Call callback with results
	if job.Callback != nil {
		job.Callback(processedDocs, nil)
	}

	return nil
}

// processQueryJob processes a single query job
func (rp *RAGPipeline) processQueryJob(ctx context.Context, payload interface{}) error {
	job, ok := payload.(QueryJob)
	if !ok {
		return fmt.Errorf("invalid query job payload")
	}

	startTime := time.Now()

	// Create search request
	searchRequest := &EnhancedSearchRequest{
		Query:                  job.Query,
		EnableQueryEnhancement: true,
		EnableReranking:        true,
		RequiredContextLength:  job.Parameters.MaxResults,
	}

	// Process query
	response, err := rp.ProcessQuery(ctx, searchRequest)
	if err != nil {
		if job.Callback != nil {
			job.Callback(QueryResult{
				ID:          job.ID,
				ProcessTime: time.Since(startTime),
			}, err)
		}
		return err
	}

	// Convert response to query result
	retrievedDocs := make([]RetrievedDocument, len(response.Results))
	for i, result := range response.Results {
		retrievedDocs[i] = RetrievedDocument{
			ID:       result.SearchResult.Document.ID,
			Content:  result.SearchResult.Document.Content,
			Score:    result.SearchResult.Score,
			Source:   result.SearchResult.Document.Source,
			Metadata: make(map[string]interface{}),
		}
	}

	queryResult := QueryResult{
		ID:          job.ID,
		Results:     retrievedDocs,
		Response:    response.AssembledContext,
		Confidence:  response.AverageRelevanceScore,
		ProcessTime: time.Since(startTime),
		CacheHit:    false, // Would need to track this from ProcessQuery
		Metadata:    make(map[string]interface{}),
	}

	// Call callback with results
	if job.Callback != nil {
		job.Callback(queryResult, nil)
	}

	return nil
}

// processDocumentsSync processes documents synchronously (fallback)
func (rp *RAGPipeline) processDocumentsSync(ctx context.Context, documents []Document, callback func([]ProcessedDocument, error)) error {
	job := DocumentJob{
		Documents: documents,
		Context:   ctx,
	}

	err := rp.processDocumentJob(ctx, job)
	return err
}

// processQuerySync processes query synchronously (fallback)
func (rp *RAGPipeline) processQuerySync(ctx context.Context, query string, params QueryParameters, callback func(QueryResult, error)) error {
	job := QueryJob{
		Query:      query,
		Parameters: params,
		Context:    ctx,
	}

	err := rp.processQueryJob(ctx, job)
	return err
}

// AsyncWorker methods

// start starts the worker's processing loop
func (w *AsyncWorker) start() {
	defer w.pool.wg.Done()
	w.pool.metrics.mutex.Lock()
	w.pool.metrics.ActiveWorkers++
	w.pool.metrics.mutex.Unlock()

	defer func() {
		w.pool.metrics.mutex.Lock()
		w.pool.metrics.ActiveWorkers--
		w.pool.metrics.mutex.Unlock()
	}()

	for {
		select {
		case job := <-w.pool.jobQueue:
			w.processJob(job)
		case <-w.pool.ctx.Done():
			return
		}
	}
}

// processJob processes a single async job
func (w *AsyncWorker) processJob(job AsyncJob) {
	startTime := time.Now()

	w.pool.metrics.mutex.Lock()
	w.pool.metrics.TotalJobs++
	w.pool.metrics.mutex.Unlock()

	err := job.Process(w.pool.ctx, job.Payload)

	processingTime := time.Since(startTime)

	w.pool.metrics.mutex.Lock()
	w.pool.metrics.CompletedJobs++
	if err != nil {
		w.pool.metrics.FailedJobs++
		w.logger.Error("Job processing failed", "job_type", job.Type, "error", err)
	}

	// Update average latency
	if w.pool.metrics.CompletedJobs > 0 {
		totalLatency := time.Duration(int64(w.pool.metrics.AverageLatency) * (w.pool.metrics.CompletedJobs - 1))
		totalLatency += processingTime
		w.pool.metrics.AverageLatency = totalLatency / time.Duration(w.pool.metrics.CompletedJobs)
	}
	w.pool.metrics.mutex.Unlock()
}

// GetAsyncMetrics returns current async processing metrics
func (rp *RAGPipeline) GetAsyncMetrics() *AsyncMetrics {
	if rp.workerPool == nil {
		return &AsyncMetrics{}
	}

	rp.workerPool.metrics.mutex.RLock()
	defer rp.workerPool.metrics.mutex.RUnlock()

	// Field-by-field copying to avoid mutex copying
	metrics := &AsyncMetrics{
		TotalJobs:      rp.workerPool.metrics.TotalJobs,
		CompletedJobs:  rp.workerPool.metrics.CompletedJobs,
		FailedJobs:     rp.workerPool.metrics.FailedJobs,
		ActiveWorkers:  rp.workerPool.metrics.ActiveWorkers,
		AverageLatency: rp.workerPool.metrics.AverageLatency,
	}
	return metrics
}

// updateQueueMetrics updates queue size metrics
func (rp *RAGPipeline) updateQueueMetrics(docDelta, queryDelta int64) {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	rp.queuedJobs += docDelta + queryDelta
	if rp.queuedJobs < 0 {
		rp.queuedJobs = 0
	}
}

// GetQueueStatus returns current queue status
func (rp *RAGPipeline) GetQueueStatus() map[string]interface{} {
	rp.mutex.RLock()
	defer rp.mutex.RUnlock()

	return map[string]interface{}{
		"document_queue_length":   len(rp.documentQueue),
		"query_queue_length":      len(rp.queryQueue),
		"document_queue_capacity": cap(rp.documentQueue),
		"query_queue_capacity":    cap(rp.queryQueue),
		"total_queued_jobs":       rp.queuedJobs,
		"active_jobs":             rp.activeJobs,
		"async_enabled":           rp.config.AsyncProcessing,
	}
}

// LoadRAGPipelineSecrets loads secrets from configuration files
func LoadRAGPipelineSecrets(config *PipelineConfig, logger *slog.Logger) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Placeholder implementation - in real scenarios this would load
	// API keys, connection strings, and other sensitive data from
	// secure sources like Kubernetes secrets, HashiCorp Vault, etc.
	logger.Debug("Loading RAG pipeline secrets",
		"embedding_provider", config.EmbeddingConfig.Provider,
		"vector_db_type", config.WeaviateConfig.Host)

	// For now, just validate that required configuration is present
	if config.EmbeddingConfig.Provider == "" {
		return fmt.Errorf("embedding provider not configured")
	}

	if config.WeaviateConfig.Host == "" {
		return fmt.Errorf("Weaviate host not configured")
	}

	return nil
}
