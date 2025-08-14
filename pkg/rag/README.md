# Nephoran Intent Operator - RAG Pipeline

This package implements a comprehensive Retrieval-Augmented Generation (RAG) pipeline specifically designed for the Nephoran Intent Operator. The pipeline is optimized for processing telecom specifications (3GPP, O-RAN, ETSI, ITU) and providing intelligent responses to network intent queries.

## Architecture Overview

The RAG pipeline consists of several interconnected components:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Document       │────│  Intelligent     │────│  Embedding      │
│  Loader         │    │  Chunking        │    │  Generation     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         v                       v                       v
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Vector Store   │    │  Query           │    │  Context        │
│  (Weaviate)     │────│  Enhancement     │────│  Assembly       │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         v                       v                       v
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Enhanced       │    │  Semantic        │    │  Intent         │
│  Retrieval      │────│  Reranking       │────│  Processing     │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         v                       v                       v
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Redis Cache    │    │  Monitoring &    │    │  LLM            │
│  Layer          │    │  Observability   │    │  Integration    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Core Components

### 1. Document Loader (`document_loader.go`)

Handles loading and parsing of telecom specification documents with:

- **Robust PDF Parsing**: Extracts text, metadata, and structure from PDF documents
- **3GPP/O-RAN Specialization**: Recognizes telecom-specific document patterns
- **Metadata Extraction**: Automatically identifies source, version, working group, and technical domains
- **Content Validation**: Ensures document quality and relevance
- **Batch Processing**: Efficiently processes multiple documents

**Key Features:**
- Support for local and remote document sources
- Automatic source detection (3GPP, O-RAN, ETSI, ITU)
- Technical term recognition and extraction
- Document structure analysis
- Content quality scoring

### 2. Intelligent Chunking (`chunking_service.go`)

Provides sophisticated document chunking that preserves hierarchical structure:

- **Hierarchy Preservation**: Maintains document section structure and relationships
- **Semantic Boundaries**: Respects paragraph and section boundaries
- **Technical Term Protection**: Avoids splitting technical terms and acronyms
- **Context Preservation**: Includes parent section context in chunks
- **Quality Scoring**: Evaluates chunk quality and relevance

**Chunking Strategies:**
- Hierarchy-aware chunking
- Semantic boundary detection
- Technical term preservation
- Adaptive chunk sizing
- Context overlap management

### 3. Embedding Service (`embedding_service.go`)

Generates high-quality embeddings with:

- **Batch Processing**: Efficient processing of multiple texts
- **Rate Limiting**: Respects API rate limits and token budgets
- **Caching**: Reduces redundant embedding generation
- **Telecom Preprocessing**: Enhances technical term recognition
- **Multiple Providers**: Support for OpenAI, Azure, and local models

**Features:**
- Configurable embedding models and dimensions
- Automatic text preprocessing and normalization
- Technical term weighting for telecom content
- Comprehensive metrics and monitoring
- Retry logic and error handling

### 4. Vector Store Integration (`weaviate_client.go`)

Provides production-ready Weaviate integration:

- **Schema Management**: Telecom-optimized schema definitions
- **Hybrid Search**: Combines vector and keyword search
- **Advanced Filtering**: Multi-dimensional document filtering
- **Performance Optimization**: Connection pooling and caching
- **Health Monitoring**: Continuous cluster health monitoring

**Capabilities:**
- Automatic schema initialization
- Complex query support
- Multi-modal search (vector + text)
- Batch operations
- Real-time indexing

### 5. Query Enhancement (`query_enhancement.go`)

Enhances user queries for better retrieval:

- **Acronym Expansion**: Expands telecom acronyms to full forms
- **Synonym Expansion**: Adds relevant synonyms and related terms
- **Spell Correction**: Corrects common telecom term misspellings
- **Context Integration**: Uses conversation history for enhancement
- **Intent-based Rewriting**: Optimizes queries based on intent type

**Enhancement Techniques:**
- Technical dictionary lookup
- Contextual term expansion
- Intent-specific query rewriting
- Historical context integration
- Telecom domain specialization

### 6. Semantic Reranking (`semantic_reranker.go`)

Improves result relevance through advanced reranking:

- **Multi-factor Scoring**: Combines semantic, lexical, and structural relevance
- **Cross-encoder Models**: Uses advanced transformer models for reranking
- **Authority Weighting**: Prioritizes authoritative sources
- **Freshness Scoring**: Considers document recency
- **Diversity Filtering**: Ensures result diversity

### 7. Context Assembly (`context_assembler.go`)

Assembles coherent context from search results:

- **Strategy Selection**: Chooses optimal assembly strategy based on results
- **Hierarchy Preservation**: Maintains document structure in context
- **Source Balancing**: Ensures diverse source representation
- **Length Management**: Respects token limits while maximizing information
- **Quality Optimization**: Prioritizes high-quality content

**Assembly Strategies:**
- Rank-based assembly
- Hierarchical grouping
- Topical clustering
- Progressive building
- Balanced source representation

### 8. Redis Caching (`redis_cache.go`)

Provides high-performance caching:

- **Multi-level Caching**: Caches embeddings, documents, queries, and contexts
- **TTL Management**: Configurable expiration for different content types
- **Compression**: Reduces memory usage for large objects
- **Metrics**: Comprehensive cache performance tracking
- **Health Monitoring**: Continuous cache health monitoring

### 9. Enhanced Retrieval (`enhanced_retrieval_service.go`)

Orchestrates the complete retrieval pipeline:

- **Query Processing**: Handles enhanced search requests
- **Component Integration**: Coordinates query enhancement, search, and reranking
- **Result Optimization**: Applies post-processing and quality filters
- **Performance Tracking**: Monitors and optimizes retrieval performance
- **Caching Integration**: Leverages caching for improved performance

### 10. Monitoring & Observability (`monitoring.go`)

Provides comprehensive monitoring:

- **Prometheus Metrics**: Detailed performance and usage metrics
- **Health Checks**: Component-level health monitoring
- **Alerting**: Configurable alerting based on thresholds
- **Dashboard Integration**: Ready for Grafana and other dashboards
- **Distributed Tracing**: Optional tracing support

## Pipeline Orchestration (`pipeline.go`)

The main pipeline orchestrator that:

- **Component Initialization**: Sets up and configures all components
- **Processing Coordination**: Manages document processing workflows
- **Query Handling**: Processes user queries through the complete pipeline
- **Resource Management**: Handles concurrent processing and resource limits
- **Integration**: Provides seamless integration with Kubernetes CRDs and LLM services

## Configuration

### Environment Variables

```bash
# Weaviate Configuration
WEAVIATE_HOST=localhost:8080
WEAVIATE_SCHEME=http
WEAVIATE_API_KEY=

# OpenAI Configuration
OPENAI_API_KEY=your_openai_api_key
OPENAI_MODEL=text-embedding-3-large

# Redis Configuration
REDIS_ADDRESS=localhost:6379
REDIS_PASSWORD=
REDIS_DATABASE=0

# Monitoring Configuration
METRICS_PORT=8080
ENABLE_MONITORING=true
```

### Configuration Structure

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
```

## Usage Examples

### Basic Pipeline Setup

```go
// Initialize pipeline
config := getDefaultPipelineConfig()
llmClient := &YourLLMClient{} // Implement llm.ClientInterface

pipeline, err := NewRAGPipeline(config, llmClient)
if err != nil {
    log.Fatal("Failed to initialize pipeline:", err)
}
defer pipeline.Shutdown(context.Background())

// Process documents
err = pipeline.ProcessDocument(context.Background(), "./documents/3gpp_spec.pdf")
if err != nil {
    log.Error("Failed to process document:", err)
}
```

### Query Processing

```go
// Enhanced search request
request := &EnhancedSearchRequest{
    Query:                  "How to configure AMF for 5G standalone?",
    IntentType:            "configuration",
    EnableQueryEnhancement: true,
    EnableReranking:        true,
    Limit:                 10,
}

// Process query
response, err := pipeline.ProcessQuery(context.Background(), request)
if err != nil {
    log.Error("Query processing failed:", err)
    return
}

fmt.Printf("Found %d results\n", len(response.Results))
fmt.Printf("Assembled context: %s\n", response.AssembledContext)
```

### Intent Processing

```go
// Process high-level intent
intent := "I need to troubleshoot AMF connectivity issues in my 5G network"
result, err := pipeline.ProcessIntent(context.Background(), intent)
if err != nil {
    log.Error("Intent processing failed:", err)
    return
}

fmt.Printf("AI Response: %s\n", result)
```

## Integration with Kubernetes

The RAG pipeline integrates seamlessly with Kubernetes CRDs:

```go
// In your controller
func (r *NetworkIntentReconciler) processIntent(ctx context.Context, intent *v1.NetworkIntent) error {
    // Use RAG pipeline to enhance intent processing
    enhancedResponse, err := r.ragPipeline.ProcessIntent(ctx, intent.Spec.Intent)
    if err != nil {
        return fmt.Errorf("RAG processing failed: %w", err)
    }
    
    // Use enhanced response for intent resolution
    return r.processEnhancedIntent(ctx, intent, enhancedResponse)
}
```

## Performance Characteristics

### Throughput
- **Document Processing**: ~100 documents/minute (depending on size)
- **Query Processing**: ~50 queries/second (with caching)
- **Embedding Generation**: ~1000 chunks/minute (with batching)

### Latency
- **Query Processing**: <200ms (cached), <2s (cold)
- **Document Chunking**: ~500ms per document
- **Context Assembly**: ~100ms per request

### Resource Usage
- **Memory**: ~2GB base + 10MB per 1000 cached embeddings
- **CPU**: ~2 cores for optimal performance
- **Storage**: Vector store size depends on document corpus

## Monitoring and Metrics

### Prometheus Metrics

```
# Query processing metrics
rag_query_latency_seconds{intent_type="configuration"}
rag_queries_total{intent_type="configuration",status="success"}

# Embedding metrics
rag_embedding_latency_seconds{model_name="text-embedding-3-large"}
rag_embeddings_total{model_name="text-embedding-3-large",status="success"}

# Cache metrics
rag_cache_hit_rate{cache_type="embedding"}
rag_cache_hit_rate{cache_type="query_result"}

# System metrics
rag_system_cpu_usage_percent
rag_system_memory_usage_percent
```

### Health Endpoints

- `/health` - Overall system health
- `/metrics` - Prometheus metrics
- `/status` - Detailed system status

## Testing

The package includes comprehensive tests:

```bash
# Run all tests
go test ./pkg/rag/...

# Run specific test suite
go test ./pkg/rag/ -run TestDocumentLoader

# Run benchmarks
go test ./pkg/rag/ -bench=.

# Run integration tests (requires external services)
INTEGRATION_TEST=true go test ./pkg/rag/ -run TestEndToEnd
```

## Error Handling

The pipeline implements robust error handling:

- **Retry Logic**: Automatic retries for transient failures
- **Circuit Breakers**: Prevents cascade failures
- **Graceful Degradation**: Continues operation with reduced functionality
- **Error Classification**: Distinguishes between recoverable and fatal errors
- **Monitoring Integration**: All errors are tracked and alerted

## Security Considerations

- **API Key Management**: Secure handling of OpenAI and other API keys
- **Data Isolation**: Multi-tenant support with proper data isolation
- **Input Validation**: Comprehensive validation of all inputs
- **Rate Limiting**: Protection against abuse and resource exhaustion
- **Audit Logging**: All operations are logged for security auditing

## Troubleshooting

### Common Issues

1. **Weaviate Connection Failed**
   - Check Weaviate service status and network connectivity
   - Verify authentication credentials
   - Review firewall and security group settings

2. **Embedding Generation Slow**
   - Check OpenAI API rate limits and quotas
   - Consider increasing batch size
   - Verify network latency to OpenAI servers

3. **Poor Query Results**
   - Review query enhancement configuration
   - Check document processing quality
   - Adjust reranking and filtering parameters

4. **High Memory Usage**
   - Review cache configuration and TTL settings
   - Consider reducing embedding cache size
   - Monitor for memory leaks in long-running processes

### Debugging

Enable debug logging:

```go
slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
})))
```

## Future Enhancements

- **Multi-modal Support**: Process diagrams and tables from documents
- **Advanced NLP**: Better intent classification and entity extraction
- **Federated Search**: Search across multiple knowledge bases
- **Real-time Updates**: Live document updates and reindexing
- **Advanced Analytics**: User behavior analysis and optimization

## Contributing

When contributing to this package:

1. Follow Go best practices and coding standards
2. Add comprehensive tests for new functionality
3. Update documentation and examples
4. Ensure backward compatibility
5. Run the full test suite before submitting

## License

This code is part of the Nephoran Intent Operator and is licensed under the Apache License 2.0.