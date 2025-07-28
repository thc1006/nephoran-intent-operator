# Nephoran RAG Pipeline - Developer Guide

## Overview

This comprehensive developer guide covers the implementation, integration, and extension of the Nephoran Intent Operator's RAG (Retrieval-Augmented Generation) pipeline. The RAG system is specifically designed for telecommunications domain knowledge processing and provides enterprise-grade capabilities for natural language intent processing.

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [API Reference](#api-reference)
3. [Usage Examples](#usage-examples)
4. [Integration Patterns](#integration-patterns)
5. [Custom Document Types](#custom-document-types)
6. [Performance Tuning](#performance-tuning)
7. [Extension Points](#extension-points)
8. [Testing Strategies](#testing-strategies)
9. [Troubleshooting](#troubleshooting)

## Architecture Overview

### Core Components

The RAG pipeline consists of several interconnected components designed for production use:

```go
// Main pipeline orchestrator
type RAGPipeline struct {
    documentLoader     *DocumentLoader          // Document ingestion
    chunkingService    *ChunkingService         // Intelligent segmentation
    embeddingService   *EmbeddingService        // Vector generation
    weaviateClient     *WeaviateClient          // Vector storage
    enhancedRetrieval  *EnhancedRetrievalService // Advanced search
    llmClient          llm.ClientInterface      // LLM integration
    redisCache         *RedisCache              // Performance caching
    monitor            *RAGMonitor              // Observability
}
```

### Data Flow

```
Intent → Query Enhancement → Vector Search → Semantic Reranking → Context Assembly → LLM Processing → Structured Response
```

## API Reference

### RAGPipeline Core API

#### Pipeline Initialization

```go
func NewRAGPipeline(config *PipelineConfig, llmClient llm.ClientInterface) (*RAGPipeline, error)
```

**Parameters:**
- `config`: Pipeline configuration with component settings
- `llmClient`: LLM client implementing the ClientInterface

**Returns:**
- Initialized RAGPipeline instance
- Error if initialization fails

**Example:**
```go
config := &PipelineConfig{
    DocumentLoaderConfig: &DocumentLoaderConfig{
        LocalPaths:       []string{"./knowledge_base"},
        MaxFileSize:      100 * 1024 * 1024, // 100MB
        BatchSize:        10,
        MaxConcurrency:   5,
    },
    ChunkingConfig: &ChunkingConfig{
        ChunkSize:            1000,
        ChunkOverlap:         200,
        PreserveHierarchy:    true,
        UseSemanticBoundaries: true,
    },
    EmbeddingConfig: &EmbeddingConfig{
        Provider:     "openai",
        ModelName:    "text-embedding-3-large",
        Dimensions:   3072,
        BatchSize:    100,
    },
    EnableCaching:    true,
    EnableMonitoring: true,
}

llmClient := &OpenAIClient{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4o-mini",
}

pipeline, err := NewRAGPipeline(config, llmClient)
if err != nil {
    log.Fatal("Failed to initialize pipeline:", err)
}
defer pipeline.Shutdown(context.Background())
```

#### Document Processing

```go
func (rp *RAGPipeline) ProcessDocument(ctx context.Context, documentPath string) error
```

**Purpose:** Processes a document through the complete RAG pipeline (load, chunk, embed, store)

**Example:**
```go
// Process single document
err := pipeline.ProcessDocument(ctx, "./specs/3gpp_ts_23_501.pdf")
if err != nil {
    log.Error("Document processing failed:", err)
}

// Process multiple documents
documents := []string{
    "./specs/3gpp_ts_23_501.pdf",
    "./specs/oran_architecture.pdf",
    "./specs/etsi_nfv_001.pdf",
}

for _, doc := range documents {
    if err := pipeline.ProcessDocument(ctx, doc); err != nil {
        log.Error("Failed to process", "document", doc, "error", err)
        continue
    }
    log.Info("Successfully processed", "document", doc)
}
```

#### Query Processing

```go
func (rp *RAGPipeline) ProcessQuery(ctx context.Context, request *EnhancedSearchRequest) (*EnhancedSearchResponse, error)
```

**Enhanced Search Request:**
```go
type EnhancedSearchRequest struct {
    Query                  string                 `json:"query"`
    Limit                  int                    `json:"limit"`
    Filters                map[string]interface{} `json:"filters,omitempty"`
    EnableQueryEnhancement bool                   `json:"enable_query_enhancement"`
    EnableReranking        bool                   `json:"enable_reranking"`
    RequiredContextLength  int                    `json:"required_context_length"`
    IntentType             string                 `json:"intent_type,omitempty"`
    NetworkDomain          string                 `json:"network_domain,omitempty"`
    TechnicalLevel         string                 `json:"technical_level,omitempty"`
    PreferredSources       []string               `json:"preferred_sources,omitempty"`
    ExcludedSources        []string               `json:"excluded_sources,omitempty"`
    MinQualityScore        float32                `json:"min_quality_score"`
}
```

**Example:**
```go
request := &EnhancedSearchRequest{
    Query:                  "How to configure AMF for 5G standalone deployment?",
    Limit:                 10,
    EnableQueryEnhancement: true,
    EnableReranking:        true,
    IntentType:            "configuration",
    NetworkDomain:         "Core",
    TechnicalLevel:        "advanced",
    PreferredSources:      []string{"3GPP", "O-RAN"},
    MinQualityScore:       0.7,
}

response, err := pipeline.ProcessQuery(ctx, request)
if err != nil {
    log.Error("Query processing failed:", err)
    return
}

fmt.Printf("Found %d results\n", len(response.Results))
fmt.Printf("Average relevance: %.2f\n", response.AverageRelevanceScore)
fmt.Printf("Processing time: %v\n", response.ProcessingTime)
fmt.Printf("Assembled context: %s\n", response.AssembledContext)
```

#### Intent Processing

```go
func (rp *RAGPipeline) ProcessIntent(ctx context.Context, intent string) (string, error)
```

**Purpose:** High-level intent processing with automatic context retrieval and LLM integration

**Example:**
```go
intent := "I need to troubleshoot AMF connectivity issues in my 5G network"
result, err := pipeline.ProcessIntent(ctx, intent)
if err != nil {
    log.Error("Intent processing failed:", err)
    return
}

fmt.Printf("AI Response: %s\n", result)
```

### Document Loader API

#### Configuration

```go
type DocumentLoaderConfig struct {
    LocalPaths        []string          `json:"local_paths"`
    RemoteURLs        []string          `json:"remote_urls"`
    MaxFileSize       int64             `json:"max_file_size"`
    PDFTextExtractor  string            `json:"pdf_text_extractor"`
    MinContentLength  int               `json:"min_content_length"`
    MaxContentLength  int               `json:"max_content_length"`
    EnableCaching     bool              `json:"enable_caching"`
    CacheDirectory    string            `json:"cache_directory"`
    CacheTTL          time.Duration     `json:"cache_ttl"`
    BatchSize         int               `json:"batch_size"`
    MaxConcurrency    int               `json:"max_concurrency"`
    ProcessingTimeout time.Duration     `json:"processing_timeout"`
    PreferredSources  map[string]int    `json:"preferred_sources"`
}
```

#### Usage

```go
// Initialize document loader
config := &DocumentLoaderConfig{
    LocalPaths:        []string{"./knowledge_base", "./specs"},
    RemoteURLs:        []string{"https://example.com/spec.pdf"},
    MaxFileSize:       100 * 1024 * 1024, // 100MB
    PDFTextExtractor:  "native",
    MinContentLength:  100,
    MaxContentLength:  1000000, // 1MB text
    EnableCaching:     true,
    CacheDirectory:    "./cache/documents",
    CacheTTL:          24 * time.Hour,
    BatchSize:         10,
    MaxConcurrency:    5,
    ProcessingTimeout: 30 * time.Second,
    PreferredSources: map[string]int{
        "3GPP":  10,
        "O-RAN": 9,
        "ETSI":  8,
        "ITU":   7,
    },
}

loader := NewDocumentLoader(config)

// Load documents
documents, err := loader.LoadDocuments(ctx)
if err != nil {
    log.Error("Failed to load documents:", err)
    return
}

for _, doc := range documents {
    fmt.Printf("Loaded: %s (%.2fKB, %s)\n", 
        doc.Filename, 
        float64(doc.Size)/1024, 
        doc.Metadata.Source)
}
```

### Embedding Service API

#### Configuration and Usage

```go
config := &EmbeddingConfig{
    Provider:         "openai",
    ModelName:        "text-embedding-3-large",
    Dimensions:       3072,
    BatchSize:        100,
    RateLimitRPM:     3000,
    RateLimitTPM:     1000000,
    MaxRetries:       3,
    RetryDelay:       time.Second,
    EnableCaching:    true,
    CacheDirectory:   "./cache/embeddings",
    CacheTTL:         7 * 24 * time.Hour, // 1 week
}

embeddingService := NewEmbeddingService(config)

// Generate embeddings for chunks
var chunks []*DocumentChunk
// ... populate chunks ...

err := embeddingService.GenerateEmbeddingsForChunks(ctx, chunks)
if err != nil {
    log.Error("Embedding generation failed:", err)
    return
}

// Chunks now have embeddings populated
for _, chunk := range chunks {
    fmt.Printf("Chunk %s: embedding dimension %d\n", 
        chunk.ID, len(chunk.Embedding))
}
```

### Enhanced Retrieval Service API

#### Search Configuration

```go
type RetrievalConfig struct {
    DefaultLimit            int                    `json:"default_limit"`
    MaxLimit               int                    `json:"max_limit"`
    DefaultHybridAlpha     float32                `json:"default_hybrid_alpha"`
    MinConfidenceThreshold float32                `json:"min_confidence_threshold"`
    EnableQueryExpansion   bool                   `json:"enable_query_expansion"`
    EnableSemanticReranking bool                  `json:"enable_semantic_reranking"`
    RerankingTopK          int                    `json:"reranking_top_k"`
    MaxContextLength       int                    `json:"max_context_length"`
    IntentTypeWeights      map[string]float64     `json:"intent_type_weights"`
    SourcePriorityWeights  map[string]float64     `json:"source_priority_weights"`
}
```

#### Advanced Search Examples

```go
// Configuration search
searchRequest := &EnhancedSearchRequest{
    Query:                  "Configure UPF for network slice",
    IntentType:            "configuration",
    NetworkDomain:         "Core",
    EnableQueryEnhancement: true,
    EnableReranking:        true,
    Filters: map[string]interface{}{
        "network_function": []string{"UPF"},
        "use_case":        []string{"eMBB", "URLLC"},
    },
}

// Troubleshooting search  
troubleshootRequest := &EnhancedSearchRequest{
    Query:                 "AMF registration failure debugging",
    IntentType:           "troubleshooting",
    TechnicalLevel:       "advanced",
    PreferredSources:     []string{"3GPP", "O-RAN"},
    RequiredContextLength: 4000,
}

// Optimization search
optimizeRequest := &EnhancedSearchRequest{
    Query:                "5G network slice performance optimization",
    IntentType:           "optimization", 
    NetworkDomain:        "RAN",
    MinQualityScore:      0.8,
    Filters: map[string]interface{}{
        "category": "Performance",
        "technology": []string{"5G", "O-RAN"},
    },
}
```

## Usage Examples

### Basic Pipeline Setup

```go
package main

import (
    "context"
    "log"
    "os"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/rag"
    "github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

func main() {
    // Initialize configuration
    config := getProductionConfig()
    
    // Initialize LLM client
    llmClient := &llm.OpenAIClient{
        APIKey: os.Getenv("OPENAI_API_KEY"),
        Model:  "gpt-4o-mini",
    }
    
    // Create pipeline
    pipeline, err := rag.NewRAGPipeline(config, llmClient)
    if err != nil {
        log.Fatal("Pipeline initialization failed:", err)
    }
    defer pipeline.Shutdown(context.Background())
    
    // Process documents
    if err := processDocuments(pipeline); err != nil {
        log.Error("Document processing failed:", err)
    }
    
    // Handle queries
    if err := handleQueries(pipeline); err != nil {
        log.Error("Query handling failed:", err)
    }
}

func getProductionConfig() *rag.PipelineConfig {
    return &rag.PipelineConfig{
        DocumentLoaderConfig: &rag.DocumentLoaderConfig{
            LocalPaths:       []string{"./knowledge_base"},
            MaxFileSize:      100 * 1024 * 1024,
            BatchSize:        10,
            MaxConcurrency:   5,
            ProcessingTimeout: 30 * time.Second,
        },
        ChunkingConfig: &rag.ChunkingConfig{
            ChunkSize:             1000,
            ChunkOverlap:          200,
            PreserveHierarchy:     true,
            UseSemanticBoundaries: true,
        },
        EmbeddingConfig: &rag.EmbeddingConfig{
            Provider:    "openai",
            ModelName:   "text-embedding-3-large",
            Dimensions:  3072,
            BatchSize:   100,
        },
        EnableCaching:    true,
        EnableMonitoring: true,
        MaxConcurrentProcessing: 10,
    }
}
```

### Kubernetes Controller Integration

```go
// NetworkIntent Controller integration
func (r *NetworkIntentReconciler) processIntent(ctx context.Context, intent *v1.NetworkIntent) error {
    // Use RAG pipeline for enhanced processing
    enhancedResponse, err := r.ragPipeline.ProcessIntent(ctx, intent.Spec.Intent)
    if err != nil {
        return fmt.Errorf("RAG processing failed: %w", err)
    }
    
    // Parse structured response
    var structuredParams map[string]interface{}
    if err := json.Unmarshal([]byte(enhancedResponse), &structuredParams); err != nil {
        return fmt.Errorf("failed to parse RAG response: %w", err)
    }
    
    // Update intent with structured parameters
    intent.Spec.Parameters = structuredParams
    intent.Status.ProcessingStatus = "Enhanced"
    intent.Status.EnhancementTimestamp = &metav1.Time{Time: time.Now()}
    
    // Continue with intent processing
    return r.processEnhancedIntent(ctx, intent, structuredParams)
}
```

### Batch Document Processing

```go
func processTelecomDocuments(pipeline *rag.RAGPipeline) error {
    ctx := context.Background()
    
    // Define document sets
    documentSets := map[string][]string{
        "3GPP Core": {
            "./specs/3gpp/TS_23_501_5G_System_Architecture.pdf",
            "./specs/3gpp/TS_23_502_5G_System_Procedures.pdf",
            "./specs/3gpp/TS_29_500_5G_System_Technical_Realization.pdf",
        },
        "O-RAN": {
            "./specs/oran/O-RAN-WG1.Use-Cases-Detailed-Specification-v07.00.pdf",
            "./specs/oran/O-RAN-WG2.AAD-v04.00.pdf",
            "./specs/oran/O-RAN-WG3.E2AP-v03.00.pdf",
        },
        "ETSI NFV": {
            "./specs/etsi/gs_NFV_001_v010401p.pdf",
            "./specs/etsi/gs_NFV-MAN_001_v010101p.pdf",
        },
    }
    
    // Process each document set
    for setName, documents := range documentSets {
        log.Info("Processing document set", "set", setName, "count", len(documents))
        
        for i, docPath := range documents {
            log.Info("Processing document", 
                "set", setName, 
                "progress", fmt.Sprintf("%d/%d", i+1, len(documents)),
                "document", filepath.Base(docPath))
            
            if err := pipeline.ProcessDocument(ctx, docPath); err != nil {
                log.Error("Document processing failed", 
                    "document", docPath, 
                    "error", err)
                continue
            }
            
            // Add delay to respect rate limits
            time.Sleep(100 * time.Millisecond)
        }
    }
    
    return nil
}
```

### Advanced Query Processing

```go
func demonstrateAdvancedQuerying(pipeline *rag.RAGPipeline) error {
    ctx := context.Background()
    
    queries := []struct {
        name    string
        request *rag.EnhancedSearchRequest
    }{
        {
            name: "Configuration Query",
            request: &rag.EnhancedSearchRequest{
                Query:                  "How to configure AMF for network slice isolation?",
                IntentType:            "configuration",
                NetworkDomain:         "Core",
                TechnicalLevel:        "advanced",
                EnableQueryEnhancement: true,
                EnableReranking:        true,
                PreferredSources:      []string{"3GPP", "O-RAN"},
                Filters: map[string]interface{}{
                    "network_function": []string{"AMF"},
                    "category":        "Configuration",
                },
                MinQualityScore: 0.7,
                Limit:          10,
            },
        },
        {
            name: "Troubleshooting Query",
            request: &rag.EnhancedSearchRequest{
                Query:                  "gNB handover failure analysis procedures",
                IntentType:            "troubleshooting",
                NetworkDomain:         "RAN",
                TechnicalLevel:        "expert",
                EnableQueryEnhancement: true,
                EnableReranking:        true,
                RequiredContextLength: 5000,
                Filters: map[string]interface{}{
                    "procedure_type": "handover",
                    "network_function": []string{"gNB"},
                },
                MinQualityScore: 0.8,
            },
        },
        {
            name: "Optimization Query",
            request: &rag.EnhancedSearchRequest{
                Query:             "URLLC slice latency optimization techniques",
                IntentType:       "optimization",
                NetworkDomain:    "RAN",
                TechnicalLevel:   "advanced",
                EnableQueryEnhancement: true,
                EnableReranking:       true,
                Filters: map[string]interface{}{
                    "use_case": []string{"URLLC"},
                    "category": "Performance",
                },
                PreferredSources: []string{"3GPP", "O-RAN", "ETSI"},
                MinQualityScore:  0.75,
            },
        },
    }
    
    for _, query := range queries {
        log.Info("Processing query", "name", query.name)
        
        response, err := pipeline.ProcessQuery(ctx, query.request)
        if err != nil {
            log.Error("Query failed", "name", query.name, "error", err)
            continue
        }
        
        // Display results
        fmt.Printf("\n=== %s ===\n", query.name)
        fmt.Printf("Query: %s\n", query.request.Query)
        fmt.Printf("Results: %d\n", len(response.Results))
        fmt.Printf("Processing time: %v\n", response.ProcessingTime)
        fmt.Printf("Average relevance: %.3f\n", response.AverageRelevanceScore)
        fmt.Printf("Coverage score: %.3f\n", response.CoverageScore)
        
        if response.QueryEnhancements != nil {
            fmt.Printf("Enhanced query: %s\n", response.QueryEnhancements.RewrittenQuery)
            if len(response.QueryEnhancements.ExpandedTerms) > 0 {
                fmt.Printf("Expanded terms: %v\n", response.QueryEnhancements.ExpandedTerms)
            }
        }
        
        // Show top results
        for i, result := range response.Results[:min(3, len(response.Results))] {
            fmt.Printf("\nResult %d:\n", i+1)
            fmt.Printf("  Title: %s\n", result.Document.Title)
            fmt.Printf("  Source: %s\n", result.Document.Source)
            fmt.Printf("  Relevance: %.3f\n", result.RelevanceScore)
            fmt.Printf("  Quality: %.3f\n", result.QualityScore)
            fmt.Printf("  Content preview: %s...\n", 
                truncateString(result.Document.Content, 100))
        }
    }
    
    return nil
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func truncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen]
}
```

## Integration Patterns

### 1. Microservice Integration

#### HTTP API Integration

```go
// HTTP handler for RAG queries
func (h *RAGHandler) HandleQuery(w http.ResponseWriter, r *http.Request) {
    var request struct {
        Query      string `json:"query"`
        IntentType string `json:"intent_type,omitempty"`
        Limit      int    `json:"limit,omitempty"`
    }
    
    if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    searchRequest := &rag.EnhancedSearchRequest{
        Query:                  request.Query,
        IntentType:            request.IntentType,
        EnableQueryEnhancement: true,
        EnableReranking:        true,
        Limit:                 request.Limit,
    }
    
    if searchRequest.Limit == 0 {
        searchRequest.Limit = 10
    }
    
    response, err := h.pipeline.ProcessQuery(r.Context(), searchRequest)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

#### gRPC Integration

```protobuf
// rag_service.proto
syntax = "proto3";

package rag;

service RAGService {
    rpc ProcessQuery(QueryRequest) returns (QueryResponse);
    rpc ProcessIntent(IntentRequest) returns (IntentResponse);
    rpc GetStatus(StatusRequest) returns (StatusResponse);
}

message QueryRequest {
    string query = 1;
    string intent_type = 2;
    string network_domain = 3;
    repeated string preferred_sources = 4;
    int32 limit = 5;
    bool enable_enhancement = 6;
    bool enable_reranking = 7;
}

message QueryResponse {
    repeated SearchResult results = 1;
    string assembled_context = 2;
    double processing_time_ms = 3;
    double average_relevance = 4;
}
```

```go
// gRPC server implementation
type RAGServer struct {
    pipeline *rag.RAGPipeline
    pb.UnimplementedRAGServiceServer
}

func (s *RAGServer) ProcessQuery(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
    searchRequest := &rag.EnhancedSearchRequest{
        Query:                  req.Query,
        IntentType:            req.IntentType,
        NetworkDomain:         req.NetworkDomain,
        PreferredSources:      req.PreferredSources,
        Limit:                 int(req.Limit),
        EnableQueryEnhancement: req.EnableEnhancement,
        EnableReranking:       req.EnableReranking,
    }
    
    response, err := s.pipeline.ProcessQuery(ctx, searchRequest)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "query processing failed: %v", err)
    }
    
    // Convert to protobuf response
    pbResults := make([]*pb.SearchResult, len(response.Results))
    for i, result := range response.Results {
        pbResults[i] = &pb.SearchResult{
            Id:        result.Document.ID,
            Title:     result.Document.Title,
            Content:   result.Document.Content,
            Source:    result.Document.Source,
            Score:     result.RelevanceScore,
            Confidence: result.QualityScore,
        }
    }
    
    return &pb.QueryResponse{
        Results:           pbResults,
        AssembledContext:  response.AssembledContext,
        ProcessingTimeMs:  float64(response.ProcessingTime.Nanoseconds()) / 1e6,
        AverageRelevance:  float64(response.AverageRelevanceScore),
    }, nil
}
```

### 2. Event-Driven Integration

#### Kafka Integration

```go
type RAGEventProcessor struct {
    pipeline     *rag.RAGPipeline
    kafkaProducer sarama.SyncProducer
    kafkaConsumer sarama.Consumer
}

func (rep *RAGEventProcessor) ProcessDocumentEvent(event *DocumentEvent) error {
    ctx := context.Background()
    
    // Process document through RAG pipeline
    err := rep.pipeline.ProcessDocument(ctx, event.DocumentPath)
    if err != nil {
        // Publish failure event
        failureEvent := &DocumentProcessingEvent{
            DocumentID: event.DocumentID,
            Status:     "failed",
            Error:      err.Error(),
            Timestamp:  time.Now(),
        }
        return rep.publishEvent("document.processing.failed", failureEvent)
    }
    
    // Publish success event
    successEvent := &DocumentProcessingEvent{
        DocumentID: event.DocumentID,
        Status:     "completed",
        Timestamp:  time.Now(),
    }
    return rep.publishEvent("document.processing.completed", successEvent)
}

func (rep *RAGEventProcessor) ProcessQueryEvent(event *QueryEvent) error {
    ctx := context.Background()
    
    request := &rag.EnhancedSearchRequest{
        Query:                  event.Query,
        IntentType:            event.IntentType,
        EnableQueryEnhancement: true,
        EnableReranking:        true,
    }
    
    response, err := rep.pipeline.ProcessQuery(ctx, request)
    if err != nil {
        return fmt.Errorf("query processing failed: %w", err)
    }
    
    // Publish query result event
    resultEvent := &QueryResultEvent{
        QueryID:       event.QueryID,
        Results:       response.Results,
        Context:       response.AssembledContext,
        ProcessingTime: response.ProcessingTime,
        Timestamp:     time.Now(),
    }
    
    return rep.publishEvent("query.result", resultEvent)
}
```

### 3. Database Integration

#### PostgreSQL Integration for Metadata

```go
type PostgreSQLMetadataStore struct {
    db *sql.DB
}

func (pms *PostgreSQLMetadataStore) StoreDocumentMetadata(ctx context.Context, doc *rag.LoadedDocument) error {
    query := `
        INSERT INTO document_metadata (
            id, source_path, filename, title, source, category, version,
            content_length, processing_time, confidence, technologies,
            network_functions, use_cases, keywords, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
        ON CONFLICT (id) DO UPDATE SET
            processing_time = EXCLUDED.processing_time,
            confidence = EXCLUDED.confidence,
            updated_at = NOW()
    `
    
    _, err := pms.db.ExecContext(ctx, query,
        doc.ID,
        doc.SourcePath,
        doc.Filename,
        doc.Title,
        doc.Metadata.Source,
        doc.Metadata.Category,
        doc.Metadata.Version,
        len(doc.Content),
        doc.ProcessingTime,
        doc.Metadata.Confidence,
        pq.Array(doc.Metadata.Technologies),
        pq.Array(doc.Metadata.NetworkFunctions),
        pq.Array(doc.Metadata.UseCases),
        pq.Array(doc.Metadata.Keywords),
        doc.LoadedAt,
    )
    
    return err
}

func (pms *PostgreSQLMetadataStore) GetDocumentsBySource(ctx context.Context, source string) ([]*DocumentMetadata, error) {
    query := `
        SELECT id, source_path, filename, title, source, category, version,
               content_length, processing_time, confidence, technologies,
               network_functions, use_cases, keywords, created_at, updated_at
        FROM document_metadata 
        WHERE source = $1 
        ORDER BY created_at DESC
    `
    
    rows, err := pms.db.QueryContext(ctx, query, source)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var documents []*DocumentMetadata
    for rows.Next() {
        var meta DocumentMetadata
        var technologies, networkFunctions, useCases, keywords []string
        
        err := rows.Scan(
            &meta.ID, &meta.SourcePath, &meta.Filename, &meta.Title,
            &meta.Source, &meta.Category, &meta.Version,
            &meta.ContentLength, &meta.ProcessingTime, &meta.Confidence,
            pq.Array(&technologies), pq.Array(&networkFunctions),
            pq.Array(&useCases), pq.Array(&keywords),
            &meta.CreatedAt, &meta.UpdatedAt,
        )
        if err != nil {
            return nil, err
        }
        
        meta.Technologies = technologies
        meta.NetworkFunctions = networkFunctions
        meta.UseCases = useCases
        meta.Keywords = keywords
        
        documents = append(documents, &meta)
    }
    
    return documents, rows.Err()
}
```

## Custom Document Types

### Implementing Custom Document Processors

```go
// CustomDocumentProcessor interface
type DocumentProcessor interface {
    CanProcess(filePath string) bool
    Process(ctx context.Context, filePath string) (*ProcessedDocument, error)
    GetSupportedExtensions() []string
}

// YAML document processor example
type YAMLDocumentProcessor struct {
    config *ProcessorConfig
}

func (ydp *YAMLDocumentProcessor) CanProcess(filePath string) bool {
    ext := strings.ToLower(filepath.Ext(filePath))
    return ext == ".yaml" || ext == ".yml"
}

func (ydp *YAMLDocumentProcessor) Process(ctx context.Context, filePath string) (*ProcessedDocument, error) {
    content, err := os.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }
    
    // Parse YAML structure
    var yamlDoc map[string]interface{}
    if err := yaml.Unmarshal(content, &yamlDoc); err != nil {
        return nil, fmt.Errorf("failed to parse YAML: %w", err)
    }
    
    // Extract metadata from YAML structure
    metadata := &DocumentMetadata{
        Source:   extractSource(yamlDoc),
        Category: extractCategory(yamlDoc),
        Version:  extractVersion(yamlDoc),
    }
    
    // Convert to text representation
    textContent := yamlToText(yamlDoc)
    
    return &ProcessedDocument{
        ID:       generateID(filePath),
        Content:  textContent,
        Metadata: metadata,
    }, nil
}

func yamlToText(yamlDoc map[string]interface{}) string {
    var builder strings.Builder
    yamlToTextRecursive(yamlDoc, &builder, 0)
    return builder.String()
}

func yamlToTextRecursive(data interface{}, builder *strings.Builder, depth int) {
    indent := strings.Repeat("  ", depth)
    
    switch v := data.(type) {
    case map[string]interface{}:
        for key, value := range v {
            builder.WriteString(fmt.Sprintf("%s%s:\n", indent, key))
            yamlToTextRecursive(value, builder, depth+1)
        }
    case []interface{}:
        for i, item := range v {
            builder.WriteString(fmt.Sprintf("%s- Item %d:\n", indent, i+1))
            yamlToTextRecursive(item, builder, depth+1)
        }
    case string:
        builder.WriteString(fmt.Sprintf("%s%s\n", indent, v))
    case float64, int:
        builder.WriteString(fmt.Sprintf("%s%v\n", indent, v))
    }
}

// Register custom processor
func (dl *DocumentLoader) RegisterProcessor(processor DocumentProcessor) {
    dl.customProcessors = append(dl.customProcessors, processor)
}

// Usage
yamlProcessor := &YAMLDocumentProcessor{
    config: &ProcessorConfig{
        ExtractMetadata: true,
        ValidateContent: true,
    },
}
documentLoader.RegisterProcessor(yamlProcessor)
```

### Custom Metadata Extractors

```go
// Telecom-specific metadata extractor
type TelecomMetadataExtractor struct {
    patterns map[string]*regexp.Regexp
}

func NewTelecomMetadataExtractor() *TelecomMetadataExtractor {
    return &TelecomMetadataExtractor{
        patterns: map[string]*regexp.Regexp{
            "3gpp_ts":      regexp.MustCompile(`(?i)\bTS\s+(\d+\.\d+)\b`),
            "3gpp_tr":      regexp.MustCompile(`(?i)\bTR\s+(\d+\.\d+)\b`),
            "oran_spec":    regexp.MustCompile(`(?i)\bO-RAN\.WG(\d+)\.([^-\s]+)`),
            "etsi_gs":      regexp.MustCompile(`(?i)\bETSI\s+GS\s+([^-\s]+)`),
            "release":      regexp.MustCompile(`(?i)\brel(?:ease)?[-\s]*(\d+)\b`),
            "version":      regexp.MustCompile(`(?i)\bv(\d+\.\d+(?:\.\d+)?)\b`),
            "working_group": regexp.MustCompile(`(?i)\b(RAN|SA|CT)\s*(\d+)\b`),
        },
    }
}

func (tme *TelecomMetadataExtractor) ExtractMetadata(content, filename string) *EnhancedMetadata {
    metadata := &EnhancedMetadata{
        Standard:      make(map[string]string),
        TechnicalInfo: make(map[string]interface{}),
        Classifications: make(map[string][]string),
    }
    
    // Extract standard information
    if matches := tme.patterns["3gpp_ts"].FindStringSubmatch(content); len(matches) > 1 {
        metadata.Standard["type"] = "3GPP TS"
        metadata.Standard["number"] = matches[1]
        metadata.Source = "3GPP"
    }
    
    if matches := tme.patterns["oran_spec"].FindStringSubmatch(content); len(matches) > 2 {
        metadata.Standard["type"] = "O-RAN"
        metadata.Standard["working_group"] = "WG" + matches[1]
        metadata.Standard["specification"] = matches[2]
        metadata.Source = "O-RAN"
    }
    
    // Extract version information
    if matches := tme.patterns["version"].FindStringSubmatch(content); len(matches) > 1 {
        metadata.Version = matches[1]
    }
    
    // Extract release information
    if matches := tme.patterns["release"].FindStringSubmatch(content); len(matches) > 1 {
        metadata.Release = "Rel-" + matches[1]
    }
    
    // Extract working group
    if matches := tme.patterns["working_group"].FindStringSubmatch(content); len(matches) > 2 {
        metadata.WorkingGroup = matches[1] + matches[2]
    }
    
    // Classify content
    metadata.Classifications["domain"] = classifyTechnicalDomain(content)
    metadata.Classifications["use_cases"] = extractUseCases(content)
    metadata.Classifications["network_functions"] = extractNetworkFunctions(content)
    
    return metadata
}
```

## Performance Tuning

### 1. Caching Optimization

```go
// Multi-level caching configuration
type CacheConfig struct {
    L1Cache struct {
        MaxSize     int           `json:"max_size"`
        TTL         time.Duration `json:"ttl"`
        EvictionPolicy string     `json:"eviction_policy"` // LRU, LFU, FIFO
    } `json:"l1_cache"`
    
    L2Cache struct {
        RedisConfig struct {
            Address     string `json:"address"`
            Password    string `json:"password"`
            Database    int    `json:"database"`
            MaxRetries  int    `json:"max_retries"`
            PoolSize    int    `json:"pool_size"`
        } `json:"redis_config"`
        TTL         time.Duration `json:"ttl"`
        Compression bool          `json:"compression"`
    } `json:"l2_cache"`
    
    DocumentCache struct {
        Directory   string        `json:"directory"`
        MaxSize     int64         `json:"max_size"` // bytes
        TTL         time.Duration `json:"ttl"`
        CleanupInterval time.Duration `json:"cleanup_interval"`
    } `json:"document_cache"`
}

// Optimized cache implementation
type OptimizedCache struct {
    l1Cache    *lru.Cache
    redisClient *redis.Client
    config     *CacheConfig
    metrics    *CacheMetrics
}

func (oc *OptimizedCache) Get(key string) (interface{}, bool) {
    // Check L1 cache first
    if value, exists := oc.l1Cache.Get(key); exists {
        oc.metrics.RecordL1Hit()
        return value, true
    }
    
    // Check L2 cache (Redis)
    ctx := context.Background()
    value, err := oc.redisClient.Get(ctx, key).Result()
    if err == nil {
        oc.metrics.RecordL2Hit()
        
        // Promote to L1 cache
        oc.l1Cache.Add(key, value)
        return value, true
    }
    
    oc.metrics.RecordCacheMiss()
    return nil, false
}

func (oc *OptimizedCache) Set(key string, value interface{}, ttl time.Duration) error {
    // Store in L1 cache
    oc.l1Cache.Add(key, value)
    
    // Store in L2 cache with compression if enabled
    ctx := context.Background()
    var data []byte
    var err error
    
    if oc.config.L2Cache.Compression {
        data, err = compress(value)
    } else {
        data, err = serialize(value)
    }
    
    if err != nil {
        return err
    }
    
    return oc.redisClient.Set(ctx, key, data, ttl).Err()
}
```

### 2. Embedding Generation Optimization

```go
// Batch embedding optimization
type BatchEmbeddingProcessor struct {
    client       EmbeddingClient
    batchSize    int
    maxWorkers   int
    rateLimiter  *rate.Limiter
    cache        EmbeddingCache
}

func (bep *BatchEmbeddingProcessor) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
    // Check cache for existing embeddings
    embeddings := make([][]float32, len(texts))
    uncachedTexts := make([]string, 0)
    uncachedIndices := make([]int, 0)
    
    for i, text := range texts {
        if cached, exists := bep.cache.Get(text); exists {
            embeddings[i] = cached
        } else {
            uncachedTexts = append(uncachedTexts, text)
            uncachedIndices = append(uncachedIndices, i)
        }
    }
    
    if len(uncachedTexts) == 0 {
        return embeddings, nil
    }
    
    // Process uncached texts in batches
    batches := bep.createBatches(uncachedTexts, bep.batchSize)
    resultCh := make(chan BatchResult, len(batches))
    
    // Start workers
    sem := make(chan struct{}, bep.maxWorkers)
    for i, batch := range batches {
        go func(batchIndex int, batchTexts []string) {
            sem <- struct{}{}
            defer func() { <-sem }()
            
            // Rate limiting
            if err := bep.rateLimiter.Wait(ctx); err != nil {
                resultCh <- BatchResult{Error: err}
                return
            }
            
            // Generate embeddings for batch
            batchEmbeddings, err := bep.client.GenerateEmbeddings(ctx, batchTexts)
            if err != nil {
                resultCh <- BatchResult{Error: err}
                return
            }
            
            resultCh <- BatchResult{
                BatchIndex:  batchIndex,
                Embeddings:  batchEmbeddings,
            }
        }(i, batch)
    }
    
    // Collect results
    results := make(map[int][][]float32)
    for i := 0; i < len(batches); i++ {
        result := <-resultCh
        if result.Error != nil {
            return nil, result.Error
        }
        results[result.BatchIndex] = result.Embeddings
    }
    
    // Reassemble embeddings and cache them
    textIndex := 0
    for batchIndex := 0; batchIndex < len(batches); batchIndex++ {
        batchEmbeddings := results[batchIndex]
        for _, embedding := range batchEmbeddings {
            originalIndex := uncachedIndices[textIndex]
            embeddings[originalIndex] = embedding
            
            // Cache the embedding
            bep.cache.Set(uncachedTexts[textIndex], embedding)
            textIndex++
        }
    }
    
    return embeddings, nil
}

type BatchResult struct {
    BatchIndex int
    Embeddings [][]float32
    Error     error
}
```

### 3. Query Processing Optimization

```go
// Parallel query processing
type ParallelQueryProcessor struct {
    weaviateClients []*WeaviateClient
    loadBalancer    LoadBalancer
    resultMerger    ResultMerger
}

func (pqp *ParallelQueryProcessor) ProcessQuery(ctx context.Context, request *EnhancedSearchRequest) (*EnhancedSearchResponse, error) {
    // Determine optimal parallelization strategy
    strategy := pqp.determineStrategy(request)
    
    switch strategy {
    case "client_parallel":
        return pqp.processWithClientParallelism(ctx, request)
    case "query_decomposition":
        return pqp.processWithQueryDecomposition(ctx, request)
    case "single_client":
        return pqp.processWithSingleClient(ctx, request)
    default:
        return pqp.processWithSingleClient(ctx, request)
    }
}

func (pqp *ParallelQueryProcessor) processWithClientParallelism(ctx context.Context, request *EnhancedSearchRequest) (*EnhancedSearchResponse, error) {
    numClients := len(pqp.weaviateClients)
    resultCh := make(chan ClientResult, numClients)
    
    // Distribute query across clients
    for i, client := range pqp.weaviateClients {
        go func(clientIndex int, weaviateClient *WeaviateClient) {
            subRequest := *request
            subRequest.Limit = request.Limit / numClients
            if clientIndex == 0 {
                subRequest.Limit += request.Limit % numClients // Handle remainder
            }
            
            response, err := weaviateClient.Search(ctx, &SearchQuery{
                Query:   request.Query,
                Limit:   subRequest.Limit,
                Filters: request.Filters,
            })
            
            resultCh <- ClientResult{
                ClientIndex: clientIndex,
                Response:    response,
                Error:       err,
            }
        }(i, client)
    }
    
    // Collect and merge results
    var allResults []*SearchResult
    for i := 0; i < numClients; i++ {
        result := <-resultCh
        if result.Error != nil {
            return nil, result.Error
        }
        allResults = append(allResults, result.Response.Results...)
    }
    
    // Merge and rerank results
    mergedResults := pqp.resultMerger.MergeAndRerank(allResults, request)
    
    return &EnhancedSearchResponse{
        Results: mergedResults,
        Total:   len(mergedResults),
    }, nil
}

type ClientResult struct {
    ClientIndex int
    Response    *SearchResponse
    Error       error
}
```

### 4. Memory Management

```go
// Memory-efficient document processing
type MemoryEfficientProcessor struct {
    memoryLimit     int64
    currentUsage    int64
    usageMutex      sync.RWMutex
    gc              *GarbageCollector
}

func (mep *MemoryEfficientProcessor) ProcessDocuments(ctx context.Context, documents []*LoadedDocument) error {
    // Sort documents by size (process smaller first)
    sort.Slice(documents, func(i, j int) bool {
        return documents[i].Size < documents[j].Size
    })
    
    for _, doc := range documents {
        // Check memory usage before processing
        if err := mep.checkMemoryUsage(doc.Size); err != nil {
            // Trigger garbage collection and retry
            mep.gc.ForceGC()
            if err := mep.checkMemoryUsage(doc.Size); err != nil {
                return fmt.Errorf("insufficient memory for document %s: %w", doc.ID, err)
            }
        }
        
        // Process document
        if err := mep.processDocument(ctx, doc); err != nil {
            return err
        }
        
        // Update memory usage
        mep.updateMemoryUsage(doc.Size)
        
        // Process in chunks to avoid memory pressure
        if mep.currentUsage > mep.memoryLimit*8/10 { // 80% threshold
            mep.gc.ForceGC()
            time.Sleep(100 * time.Millisecond) // Allow GC to complete
        }
    }
    
    return nil
}

func (mep *MemoryEfficientProcessor) checkMemoryUsage(requiredSize int64) error {
    mep.usageMutex.RLock()
    defer mep.usageMutex.RUnlock()
    
    if mep.currentUsage+requiredSize > mep.memoryLimit {
        return fmt.Errorf("memory limit exceeded: current=%d, required=%d, limit=%d", 
            mep.currentUsage, requiredSize, mep.memoryLimit)
    }
    
    return nil
}

func (mep *MemoryEfficientProcessor) updateMemoryUsage(size int64) {
    mep.usageMutex.Lock()
    defer mep.usageMutex.Unlock()
    mep.currentUsage += size
}

type GarbageCollector struct {
    lastGC    time.Time
    minInterval time.Duration
}

func (gc *GarbageCollector) ForceGC() {
    if time.Since(gc.lastGC) < gc.minInterval {
        return
    }
    
    runtime.GC()
    gc.lastGC = time.Now()
}
```

## Extension Points

### 1. Custom Query Enhancers

```go
// Query enhancer interface
type QueryEnhancer interface {
    EnhanceQuery(ctx context.Context, request *EnhancedSearchRequest) (string, *QueryEnhancements, error)
    GetName() string
    GetPriority() int
}

// Telecom-specific query enhancer
type TelecomQueryEnhancer struct {
    acronymDatabase map[string]string
    synonymDatabase map[string][]string
    spellChecker   SpellChecker
}

func (tqe *TelecomQueryEnhancer) EnhanceQuery(ctx context.Context, request *EnhancedSearchRequest) (string, *QueryEnhancements, error) {
    enhancements := &QueryEnhancements{
        OriginalQuery:       request.Query,
        ExpandedTerms:       []string{},
        SynonymReplacements: make(map[string]string),
        SpellingCorrections: make(map[string]string),
        EnhancementApplied:  []string{},
    }
    
    query := request.Query
    
    // 1. Spell correction
    correctedQuery, corrections := tqe.spellChecker.CorrectTelecomTerms(query)
    if len(corrections) > 0 {
        query = correctedQuery
        enhancements.SpellingCorrections = corrections
        enhancements.EnhancementApplied = append(enhancements.EnhancementApplied, "spell_correction")
    }
    
    // 2. Acronym expansion
    expandedQuery, expandedTerms := tqe.expandAcronyms(query)
    if len(expandedTerms) > 0 {
        query = expandedQuery
        enhancements.ExpandedTerms = expandedTerms
        enhancements.EnhancementApplied = append(enhancements.EnhancementApplied, "acronym_expansion")
    }
    
    // 3. Synonym enhancement
    synonymQuery, synonyms := tqe.addSynonyms(query)
    if len(synonyms) > 0 {
        query = synonymQuery
        enhancements.SynonymReplacements = synonyms
        enhancements.EnhancementApplied = append(enhancements.EnhancementApplied, "synonym_enhancement")
    }
    
    // 4. Intent-based enhancement
    if request.IntentType != "" {
        intentQuery := tqe.enhanceForIntent(query, request.IntentType)
        if intentQuery != query {
            query = intentQuery
            enhancements.EnhancementApplied = append(enhancements.EnhancementApplied, "intent_enhancement")
        }
    }
    
    enhancements.RewrittenQuery = query
    return query, enhancements, nil
}

func (tqe *TelecomQueryEnhancer) expandAcronyms(query string) (string, []string) {
    words := strings.Fields(query)
    var expandedTerms []string
    var result []string
    
    for _, word := range words {
        cleanWord := strings.ToUpper(strings.Trim(word, ".,!?"))
        if expansion, exists := tqe.acronymDatabase[cleanWord]; exists {
            result = append(result, word+" ("+expansion+")")
            expandedTerms = append(expandedTerms, expansion)
        } else {
            result = append(result, word)
        }
    }
    
    return strings.Join(result, " "), expandedTerms
}

func (tqe *TelecomQueryEnhancer) addSynonyms(query string) (string, map[string]string) {
    synonyms := make(map[string]string)
    words := strings.Fields(query)
    var result []string
    
    for _, word := range words {
        cleanWord := strings.ToLower(strings.Trim(word, ".,!?"))
        if synonymList, exists := tqe.synonymDatabase[cleanWord]; exists {
            // Add the most relevant synonym
            synonym := synonymList[0]
            result = append(result, word+" OR "+synonym)
            synonyms[cleanWord] = synonym
        } else {
            result = append(result, word)
        }
    }
    
    return strings.Join(result, " "), synonyms
}

func (tqe *TelecomQueryEnhancer) enhanceForIntent(query, intentType string) string {
    switch intentType {
    case "configuration":
        return query + " configuration setup parameters"
    case "troubleshooting":
        return query + " troubleshooting debugging issues problems"
    case "optimization":
        return query + " optimization performance tuning improvement"
    case "monitoring":
        return query + " monitoring metrics measurement KPI"
    default:
        return query
    }
}

// Register custom enhancer
func (ers *EnhancedRetrievalService) RegisterQueryEnhancer(enhancer QueryEnhancer) {
    ers.queryEnhancers = append(ers.queryEnhancers, enhancer)
    // Sort by priority
    sort.Slice(ers.queryEnhancers, func(i, j int) bool {
        return ers.queryEnhancers[i].GetPriority() > ers.queryEnhancers[j].GetPriority()
    })
}
```

### 2. Custom Rerankers

```go
// Custom semantic reranker
type TelecomSemanticReranker struct {
    crossEncoder    CrossEncoderModel
    authorityWeights map[string]float64
    domainWeights   map[string]float64
}

func (tsr *TelecomSemanticReranker) RerankResults(ctx context.Context, query string, results []*EnhancedSearchResult) ([]*EnhancedSearchResult, error) {
    if len(results) <= 1 {
        return results, nil
    }
    
    // Create query-document pairs for cross-encoder
    pairs := make([]QueryDocumentPair, len(results))
    for i, result := range results {
        pairs[i] = QueryDocumentPair{
            Query:    query,
            Document: result.Document.Content,
            ID:       result.Document.ID,
        }
    }
    
    // Get semantic similarity scores
    similarities, err := tsr.crossEncoder.PredictSimilarities(ctx, pairs)
    if err != nil {
        return results, err // Fall back to original ranking
    }
    
    // Update semantic similarity scores
    for i, result := range results {
        result.SemanticSimilarity = similarities[i]
    }
    
    // Apply telecom-specific ranking factors
    for _, result := range results {
        // Authority weighting
        if weight, exists := tsr.authorityWeights[result.Document.Source]; exists {
            result.AuthorityScore = float32(weight)
        }
        
        // Domain relevance weighting
        if weight, exists := tsr.domainWeights[result.Document.Category]; exists {
            result.CombinedScore *= float32(weight)
        }
        
        // Recalculate combined score with semantic similarity
        result.CombinedScore = tsr.calculateCombinedScore(result)
    }
    
    // Sort by combined score
    sort.Slice(results, func(i, j int) bool {
        return results[i].CombinedScore > results[j].CombinedScore
    })
    
    return results, nil
}

func (tsr *TelecomSemanticReranker) calculateCombinedScore(result *EnhancedSearchResult) float32 {
    weights := map[string]float32{
        "relevance":  0.25,
        "semantic":   0.30,
        "quality":    0.20,
        "authority":  0.15,
        "freshness":  0.10,
    }
    
    return result.RelevanceScore*weights["relevance"] +
        result.SemanticSimilarity*weights["semantic"] +
        result.QualityScore*weights["quality"] +
        result.AuthorityScore*weights["authority"] +
        result.FreshnessScore*weights["freshness"]
}
```

### 3. Custom Context Assemblers

```go
// Hierarchical context assembler
type HierarchicalContextAssembler struct {
    maxTokens        int
    tokenizer        Tokenizer
    structureWeights map[string]float64
}

func (hca *HierarchicalContextAssembler) AssembleContext(results []*EnhancedSearchResult, request *EnhancedSearchRequest) (string, *ContextMetadata) {
    // Group results by document hierarchy
    hierarchyGroups := hca.groupByHierarchy(results)
    
    // Calculate tokens for each group
    groupTokens := make(map[string]int)
    totalAvailableTokens := hca.maxTokens
    
    for groupName, groupResults := range hierarchyGroups {
        groupTokens[groupName] = hca.calculateGroupTokens(groupResults, totalAvailableTokens, len(hierarchyGroups))
    }
    
    // Assemble context maintaining hierarchy
    var contextBuilder strings.Builder
    metadata := &ContextMetadata{
        DocumentCount:      0,
        HierarchyLevels:    []int{},
        SourceDistribution: make(map[string]int),
        TechnicalTermCount: 0,
    }
    
    for groupName, groupResults := range hierarchyGroups {
        availableTokens := groupTokens[groupName]
        
        contextBuilder.WriteString(fmt.Sprintf("\n=== %s ===\n", groupName))
        
        for _, result := range groupResults {
            content := result.Document.Content
            contentTokens := hca.tokenizer.CountTokens(content)
            
            if availableTokens < contentTokens {
                // Truncate content to fit
                content = hca.tokenizer.TruncateToTokens(content, availableTokens)
                contentTokens = availableTokens
            }
            
            contextBuilder.WriteString(content)
            contextBuilder.WriteString("\n\n")
            
            availableTokens -= contentTokens
            metadata.DocumentCount++
            metadata.SourceDistribution[result.Document.Source]++
            
            if availableTokens <= 0 {
                break
            }
        }
    }
    
    context := contextBuilder.String()
    metadata.TotalLength = len(context)
    metadata.TechnicalTermCount = hca.countTechnicalTerms(context)
    
    return context, metadata
}

func (hca *HierarchicalContextAssembler) groupByHierarchy(results []*EnhancedSearchResult) map[string][]*EnhancedSearchResult {
    groups := make(map[string][]*EnhancedSearchResult)
    
    for _, result := range results {
        groupKey := hca.determineHierarchyGroup(result)
        groups[groupKey] = append(groups[groupKey], result)
    }
    
    return groups
}

func (hca *HierarchicalContextAssembler) determineHierarchyGroup(result *EnhancedSearchResult) string {
    doc := result.Document
    
    // Create hierarchical group key
    if doc.Source != "" && doc.Category != "" {
        return fmt.Sprintf("%s - %s", doc.Source, doc.Category)
    } else if doc.Source != "" {
        return doc.Source
    } else if doc.Category != "" {
        return doc.Category
    }
    
    return "General"
}
```

## Testing Strategies

### 1. Unit Testing

```go
// Test document loader
func TestDocumentLoader(t *testing.T) {
    tests := []struct {
        name           string
        config         *DocumentLoaderConfig
        testFiles      []string
        expectedDocs   int
        expectedError  bool
    }{
        {
            name: "Load PDF documents",
            config: &DocumentLoaderConfig{
                LocalPaths: []string{"./testdata"},
                MaxFileSize: 10 * 1024 * 1024,
                BatchSize: 5,
            },
            testFiles:     []string{"test_spec.pdf", "another_spec.pdf"},
            expectedDocs:  2,
            expectedError: false,
        },
        {
            name: "Handle large files",
            config: &DocumentLoaderConfig{
                LocalPaths: []string{"./testdata"},
                MaxFileSize: 1024, // Very small limit
            },
            testFiles:     []string{"large_spec.pdf"},
            expectedDocs:  0,
            expectedError: false, // Should skip, not error
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup test directory
            testDir := setupTestDirectory(t, tt.testFiles)
            defer os.RemoveAll(testDir)
            
            tt.config.LocalPaths = []string{testDir}
            
            // Create loader and load documents
            loader := NewDocumentLoader(tt.config)
            docs, err := loader.LoadDocuments(context.Background())
            
            if tt.expectedError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Len(t, docs, tt.expectedDocs)
                
                // Verify document metadata
                for _, doc := range docs {
                    assert.NotEmpty(t, doc.ID)
                    assert.NotEmpty(t, doc.Content)
                    assert.NotNil(t, doc.Metadata)
                }
            }
        })
    }
}

// Test embedding service
func TestEmbeddingService(t *testing.T) {
    mockClient := &MockEmbeddingClient{
        embeddings: map[string][]float32{
            "test text 1": make([]float32, 1536),
            "test text 2": make([]float32, 1536),
        },
    }
    
    config := &EmbeddingConfig{
        Provider:  "mock",
        BatchSize: 2,
    }
    
    service := NewEmbeddingService(config)
    service.client = mockClient
    
    chunks := []*DocumentChunk{
        {ID: "1", CleanContent: "test text 1"},
        {ID: "2", CleanContent: "test text 2"},
    }
    
    err := service.GenerateEmbeddingsForChunks(context.Background(), chunks)
    assert.NoError(t, err)
    
    for _, chunk := range chunks {
        assert.NotNil(t, chunk.Embedding)
        assert.Len(t, chunk.Embedding, 1536)
    }
}

type MockEmbeddingClient struct {
    embeddings map[string][]float32
}

func (mec *MockEmbeddingClient) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
    result := make([][]float32, len(texts))
    for i, text := range texts {
        if embedding, exists := mec.embeddings[text]; exists {
            result[i] = embedding
        } else {
            result[i] = make([]float32, 1536) // Default embedding
        }
    }
    return result, nil
}
```

### 2. Integration Testing

```go
func TestRAGPipelineIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Setup test environment
    testEnv := setupIntegrationTestEnvironment(t)
    defer testEnv.Cleanup()
    
    // Create pipeline with test configuration
    config := &PipelineConfig{
        DocumentLoaderConfig: &DocumentLoaderConfig{
            LocalPaths: []string{testEnv.DocumentDir},
            BatchSize:  2,
        },
        ChunkingConfig: &ChunkingConfig{
            ChunkSize:    500,
            ChunkOverlap: 50,
        },
        EmbeddingConfig: &EmbeddingConfig{
            Provider: "mock",
        },
        WeaviateConfig: &WeaviateConfig{
            Host:   testEnv.WeaviateHost,
            Scheme: "http",
        },
        EnableCaching: false, // Disable for consistent testing
    }
    
    mockLLMClient := &MockLLMClient{}
    pipeline, err := NewRAGPipeline(config, mockLLMClient)
    require.NoError(t, err)
    defer pipeline.Shutdown(context.Background())
    
    // Test document processing
    testDoc := filepath.Join(testEnv.DocumentDir, "test_spec.pdf")
    err = pipeline.ProcessDocument(context.Background(), testDoc)
    assert.NoError(t, err)
    
    // Wait for processing to complete
    time.Sleep(2 * time.Second)
    
    // Test query processing
    request := &EnhancedSearchRequest{
        Query:                  "AMF configuration parameters",
        EnableQueryEnhancement: true,
        EnableReranking:        true,
        Limit:                 5,
    }
    
    response, err := pipeline.ProcessQuery(context.Background(), request)
    assert.NoError(t, err)
    assert.NotNil(t, response)
    assert.GreaterOrEqual(t, len(response.Results), 1)
    
    // Test intent processing
    intent := "How to configure AMF for 5G standalone?"
    result, err := pipeline.ProcessIntent(context.Background(), intent)
    assert.NoError(t, err)
    assert.NotEmpty(t, result)
    
    // Verify metrics
    status := pipeline.GetStatus()
    assert.True(t, status.IsHealthy)
    assert.Greater(t, status.ProcessedDocuments, int64(0))
    assert.Greater(t, status.ProcessedQueries, int64(0))
}

type IntegrationTestEnvironment struct {
    DocumentDir   string
    WeaviateHost  string
    cleanup       func()
}

func (ite *IntegrationTestEnvironment) Cleanup() {
    if ite.cleanup != nil {
        ite.cleanup()
    }
}

func setupIntegrationTestEnvironment(t *testing.T) *IntegrationTestEnvironment {
    // Create temporary document directory
    docDir, err := os.MkdirTemp("", "rag-test-docs-*")
    require.NoError(t, err)
    
    // Copy test documents
    copyTestDocuments(t, docDir)
    
    // Start test Weaviate instance (using Docker)
    weaviateHost := startTestWeaviate(t)
    
    return &IntegrationTestEnvironment{
        DocumentDir:  docDir,
        WeaviateHost: weaviateHost,
        cleanup: func() {
            os.RemoveAll(docDir)
            stopTestWeaviate(t, weaviateHost)
        },
    }
}
```

### 3. Performance Testing

```go
func BenchmarkDocumentProcessing(b *testing.B) {
    config := &DocumentLoaderConfig{
        LocalPaths:     []string{"./benchdata"},
        MaxConcurrency: 4,
        BatchSize:      10,
    }
    
    loader := NewDocumentLoader(config)
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        docs, err := loader.LoadDocuments(context.Background())
        if err != nil {
            b.Fatal(err)
        }
        
        if len(docs) == 0 {
            b.Fatal("No documents loaded")
        }
    }
}

func BenchmarkQueryProcessing(b *testing.B) {
    pipeline := setupBenchmarkPipeline(b)
    defer pipeline.Shutdown(context.Background())
    
    queries := []string{
        "AMF configuration",
        "5G network slice setup",
        "UPF troubleshooting",
        "gNB optimization",
    }
    
    b.ResetTimer()
    b.ReportAllocs()
    
    for i := 0; i < b.N; i++ {
        query := queries[i%len(queries)]
        request := &EnhancedSearchRequest{
            Query: query,
            Limit: 10,
        }
        
        _, err := pipeline.ProcessQuery(context.Background(), request)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkConcurrentQueries(b *testing.B) {
    pipeline := setupBenchmarkPipeline(b)
    defer pipeline.Shutdown(context.Background())
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            request := &EnhancedSearchRequest{
                Query: "5G core network functions",
                Limit: 5,
            }
            
            _, err := pipeline.ProcessQuery(context.Background(), request)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Memory Issues

**Problem**: Out of memory errors during document processing

**Symptoms**:
```
fatal error: runtime: out of memory
```

**Solutions**:
```go
// Reduce batch sizes
config.DocumentLoaderConfig.BatchSize = 5
config.EmbeddingConfig.BatchSize = 50

// Enable document streaming
config.DocumentLoaderConfig.EnableStreaming = true

// Reduce chunk size
config.ChunkingConfig.ChunkSize = 500

// Enable garbage collection tuning
runtime.GC()
debug.SetGCPercent(20) // More aggressive GC
```

#### 2. Slow Query Performance

**Problem**: Queries taking too long to process

**Diagnostic Steps**:
```go
// Enable detailed timing
response, err := pipeline.ProcessQuery(ctx, &EnhancedSearchRequest{
    Query: "your query",
    DebugMode: true, // Add debug information
})

if err == nil {
    fmt.Printf("Retrieval time: %v\n", response.RetrievalTime)
    fmt.Printf("Enhancement time: %v\n", response.EnhancementTime)
    fmt.Printf("Reranking time: %v\n", response.RerankingTime)
    fmt.Printf("Context assembly time: %v\n", response.ContextAssemblyTime)
}
```

**Solutions**:
```go
// Optimize cache settings
config.RedisCacheConfig.MaxSize = 10000
config.RedisCacheConfig.TTL = 2 * time.Hour

// Reduce reranking scope
config.RetrievalConfig.RerankingTopK = 20

// Optimize Weaviate queries
config.WeaviateConfig.QueryTimeout = 5 * time.Second
config.WeaviateConfig.MaxConnections = 100
```

#### 3. Poor Search Quality

**Problem**: Search results are not relevant

**Diagnostic Steps**:
```go
// Analyze query enhancements
if response.QueryEnhancements != nil {
    fmt.Printf("Original: %s\n", response.QueryEnhancements.OriginalQuery)
    fmt.Printf("Enhanced: %s\n", response.QueryEnhancements.RewrittenQuery)
    fmt.Printf("Expanded terms: %v\n", response.QueryEnhancements.ExpandedTerms)
}

// Check result scores
for i, result := range response.Results {
    fmt.Printf("Result %d: relevance=%.3f, quality=%.3f, authority=%.3f\n",
        i+1, result.RelevanceScore, result.QualityScore, result.AuthorityScore)
}
```

**Solutions**:
```go
// Tune scoring weights
config.RetrievalConfig.IntentTypeWeights = map[string]float64{
    "configuration":    1.3,
    "troubleshooting":  1.4,
    "optimization":     1.2,
}

config.RetrievalConfig.SourcePriorityWeights = map[string]float64{
    "3GPP":  1.4,
    "O-RAN": 1.3,
    "ETSI":  1.2,
}

// Improve query enhancement
config.RetrievalConfig.EnableQueryExpansion = true
config.RetrievalConfig.QueryExpansionTerms = 5
config.RetrievalConfig.EnableSynonymExpansion = true
```

#### 4. Embedding Generation Failures

**Problem**: OpenAI API errors or timeouts

**Error Examples**:
```
rate limit exceeded
context deadline exceeded
invalid API key
```

**Solutions**:
```go
// Implement retry logic with exponential backoff
config.EmbeddingConfig.MaxRetries = 5
config.EmbeddingConfig.RetryDelay = 2 * time.Second
config.EmbeddingConfig.BackoffMultiplier = 2.0

// Reduce API rate
config.EmbeddingConfig.RateLimitRPM = 2000
config.EmbeddingConfig.BatchSize = 50

// Add API key rotation
config.EmbeddingConfig.APIKeys = []string{
    "sk-key1...",
    "sk-key2...",
    "sk-key3...",
}
```

#### 5. Weaviate Connection Issues

**Problem**: Cannot connect to Weaviate cluster

**Diagnostic Steps**:
```go
// Test connection
health := weaviateClient.GetHealthStatus()
fmt.Printf("Weaviate healthy: %v\n", health.IsHealthy)
fmt.Printf("Last check: %v\n", health.LastCheck)

// Check cluster status
status, err := weaviateClient.GetClusterStatus()
if err != nil {
    fmt.Printf("Cluster status error: %v\n", err)
} else {
    fmt.Printf("Nodes: %d\n", len(status.Nodes))
    for _, node := range status.Nodes {
        fmt.Printf("Node %s: %s\n", node.Name, node.Status)
    }
}
```

**Solutions**:
```go
// Configure connection pooling
config.WeaviateConfig.MaxConnections = 50
config.WeaviateConfig.MaxIdleConnections = 10
config.WeaviateConfig.ConnectionTimeout = 10 * time.Second

// Add retry logic
config.WeaviateConfig.MaxRetries = 3
config.WeaviateConfig.RetryDelay = time.Second

// Enable authentication
config.WeaviateConfig.APIKey = "your-api-key"
config.WeaviateConfig.EnableTLS = true
```

### Debug Tools

#### 1. Query Analyzer

```go
func AnalyzeQuery(pipeline *RAGPipeline, query string) {
    request := &EnhancedSearchRequest{
        Query:                  query,
        EnableQueryEnhancement: true,
        EnableReranking:        true,
        Limit:                 10,
        DebugMode:             true,
    }
    
    response, err := pipeline.ProcessQuery(context.Background(), request)
    if err != nil {
        fmt.Printf("Query failed: %v\n", err)
        return
    }
    
    fmt.Printf("=== Query Analysis ===\n")
    fmt.Printf("Original Query: %s\n", query)
    
    if response.QueryEnhancements != nil {
        fmt.Printf("Enhanced Query: %s\n", response.QueryEnhancements.RewrittenQuery)
        fmt.Printf("Enhancements Applied: %v\n", response.QueryEnhancements.EnhancementApplied)
        
        if len(response.QueryEnhancements.ExpandedTerms) > 0 {
            fmt.Printf("Expanded Terms: %v\n", response.QueryEnhancements.ExpandedTerms)
        }
        
        if len(response.QueryEnhancements.SpellingCorrections) > 0 {
            fmt.Printf("Spelling Corrections: %v\n", response.QueryEnhancements.SpellingCorrections)
        }
    }
    
    fmt.Printf("\n=== Performance Metrics ===\n")
    fmt.Printf("Total Processing Time: %v\n", response.ProcessingTime)
    fmt.Printf("Enhancement Time: %v\n", response.EnhancementTime)
    fmt.Printf("Retrieval Time: %v\n", response.RetrievalTime)
    fmt.Printf("Reranking Time: %v\n", response.RerankingTime)
    fmt.Printf("Context Assembly Time: %v\n", response.ContextAssemblyTime)
    
    fmt.Printf("\n=== Quality Metrics ===\n")
    fmt.Printf("Average Relevance: %.3f\n", response.AverageRelevanceScore)
    fmt.Printf("Coverage Score: %.3f\n", response.CoverageScore)
    fmt.Printf("Diversity Score: %.3f\n", response.DiversityScore)
    
    fmt.Printf("\n=== Top Results ===\n")
    for i, result := range response.Results[:min(5, len(response.Results))] {
        fmt.Printf("Result %d:\n", i+1)
        fmt.Printf("  Title: %s\n", result.Document.Title)
        fmt.Printf("  Source: %s\n", result.Document.Source)
        fmt.Printf("  Relevance: %.3f\n", result.RelevanceScore)
        fmt.Printf("  Quality: %.3f\n", result.QualityScore)
        fmt.Printf("  Authority: %.3f\n", result.AuthorityScore)
        fmt.Printf("  Combined: %.3f\n", result.CombinedScore)
        fmt.Printf("\n")
    }
    
    if response.DebugInfo != nil {
        fmt.Printf("=== Debug Information ===\n")
        for key, value := range response.DebugInfo {
            fmt.Printf("%s: %v\n", key, value)
        }
    }
}
```

#### 2. Performance Profiler

```go
func ProfilePipeline(pipeline *RAGPipeline) {
    // Start CPU profiling
    f, err := os.Create("rag_cpu_profile.prof")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()
    
    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()
    
    // Run test queries
    queries := []string{
        "AMF configuration parameters",
        "5G network slice management",
        "gNB handover procedures",
        "UPF session establishment",
    }
    
    start := time.Now()
    for i := 0; i < 100; i++ {
        for _, query := range queries {
            request := &EnhancedSearchRequest{
                Query: query,
                Limit: 10,
            }
            pipeline.ProcessQuery(context.Background(), request)
        }
    }
    totalTime := time.Since(start)
    
    fmt.Printf("Processed %d queries in %v\n", len(queries)*100, totalTime)
    fmt.Printf("Average time per query: %v\n", totalTime/time.Duration(len(queries)*100))
    
    // Memory profiling
    mf, err := os.Create("rag_memory_profile.prof")
    if err != nil {
        log.Fatal(err)
    }
    defer mf.Close()
    
    runtime.GC()
    pprof.WriteHeapProfile(mf)
}
```

This comprehensive developer guide provides detailed information for implementing, integrating, and extending the Nephoran RAG pipeline. The examples and patterns shown here enable developers to effectively leverage the system's capabilities while maintaining production-grade performance and reliability.