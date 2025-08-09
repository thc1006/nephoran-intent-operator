# RAG/Weaviate Optional Build Support

## Overview
The Nephoran Intent Operator now supports optional RAG (Retrieval-Augmented Generation) with Weaviate through Go build tags. This allows building the operator with or without Weaviate dependencies, reducing binary size and dependencies when RAG functionality is not needed.

## Architecture

### Interface Abstraction
The RAG functionality is abstracted through the `RAGClient` interface defined in `pkg/rag/types.go`:

```go
type RAGClient interface {
    ProcessIntent(ctx context.Context, intent string) (string, error)
    Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
    Initialize(ctx context.Context) error
    Shutdown(ctx context.Context) error
    IsHealthy() bool
}
```

### Implementations

1. **No-op Implementation** (`client_noop.go` - default)
   - Built when the `rag` build tag is NOT specified
   - Returns appropriate errors indicating RAG is not enabled
   - Zero dependencies on Weaviate or other RAG libraries
   - Minimal memory and CPU footprint

2. **Weaviate Implementation** (`client_weaviate.go` - with `rag` tag)
   - Built when the `rag` build tag IS specified
   - Full Weaviate integration with semantic search
   - LLM enhancement capabilities
   - Vector database operations

## Building

### Without RAG Support (Default)
```bash
# Standard build - no RAG support
go build ./...

# Or explicitly without RAG
go build -tags="" ./...
```

When built without RAG support:
- Binary size is smaller
- No Weaviate dependencies
- RAG operations return "not enabled" errors
- Suitable for deployments that don't need RAG

### With RAG Support
```bash
# Build with RAG/Weaviate support
go build -tags=rag ./...

# Or multiple tags
go build -tags="rag,other_feature" ./...
```

When built with RAG support:
- Full Weaviate integration
- Semantic search capabilities
- LLM-enhanced responses
- Requires Weaviate instance to be configured

## Configuration

### Environment Variables
When RAG is enabled, configure these environment variables:

```bash
# Weaviate configuration
export WEAVIATE_URL=http://localhost:8080
export WEAVIATE_API_KEY=your-api-key  # Optional

# LLM configuration for RAG enhancement
export LLM_BACKEND_TYPE=rag
export LLM_ENDPOINT=https://api.openai.com/v1/chat/completions
export LLM_API_KEY=your-openai-key
```

### LLM Client Integration
The LLM client (`pkg/llm/llm.go`) automatically detects and uses RAG when:
1. Built with the `rag` tag
2. `LLM_BACKEND_TYPE` is set to "rag"
3. Weaviate configuration is provided

```go
// In NewClientWithConfig
if config.BackendType == "rag" {
    ragConfig := &rag.RAGClientConfig{
        Enabled:          true,
        WeaviateURL:      os.Getenv("WEAVIATE_URL"),
        WeaviateAPIKey:   os.Getenv("WEAVIATE_API_KEY"),
        // ... other config
    }
    client.ragClient = rag.NewRAGClient(ragConfig)
}
```

## Verification

### Check Build Mode
Use the provided verification script:

```bash
# Test without RAG
go run verify_rag_build.go

# Test with RAG
go run -tags=rag verify_rag_build.go
```

Expected output without RAG:
```
=== RAG Build Verification ===
✓ Initialize succeeded
✓ Running with NO-OP implementation (build without -tags=rag)
✓ IsHealthy: true
✓ Shutdown succeeded
```

Expected output with RAG:
```
=== RAG Build Verification ===
✓ Initialize succeeded
✓ Running with Weaviate implementation (build with -tags=rag)
  Result: [RAG-enhanced response]
✓ IsHealthy: true
✓ Shutdown succeeded
```

## Testing

### Unit Tests
```bash
# Test no-op implementation
go test ./pkg/rag

# Test Weaviate implementation
go test -tags=rag ./pkg/rag
```

### Integration Tests
When testing with RAG enabled, ensure:
1. Weaviate instance is running
2. Environment variables are configured
3. Test data is loaded into Weaviate

## Migration Guide

### For Existing Deployments

1. **No changes needed** for deployments not using RAG
   - Default build excludes RAG support
   - Existing functionality unchanged

2. **To enable RAG** in existing deployments:
   - Rebuild with `-tags=rag`
   - Configure Weaviate environment variables
   - Deploy Weaviate instance if not already available

### CI/CD Considerations

Update your build pipeline to handle both scenarios:

```yaml
# GitHub Actions example
jobs:
  build-standard:
    steps:
      - name: Build without RAG
        run: go build ./...
        
  build-with-rag:
    steps:
      - name: Build with RAG
        run: go build -tags=rag ./...
```

## Benefits

### Reduced Dependencies
- No Weaviate client libraries in default build
- Smaller container images
- Faster build times

### Flexibility
- Deploy with or without RAG based on requirements
- Easy to enable/disable at build time
- No runtime overhead when not using RAG

### Maintainability
- Clean separation of concerns
- Interface-based design
- Easy to add other RAG providers in future

## Troubleshooting

### Build Errors

If you encounter errors like:
```
undefined: WeaviateClient
```
Ensure you're building with the correct tags.

### Runtime Errors

Error: "RAG support is not enabled"
- Solution: Rebuild with `-tags=rag`

Error: "Failed to connect to Weaviate"
- Check `WEAVIATE_URL` is correct
- Verify Weaviate instance is running
- Check network connectivity

### Performance

Without RAG:
- Minimal overhead
- Direct LLM processing

With RAG:
- Additional latency for vector search
- Memory usage for caching
- Network calls to Weaviate

## Future Enhancements

1. **Additional RAG Providers**
   - Pinecone support
   - Qdrant support
   - ChromaDB support

2. **Build Tag Combinations**
   - `rag_weaviate` for Weaviate
   - `rag_pinecone` for Pinecone
   - Allow multiple providers

3. **Runtime Provider Selection**
   - Dynamic provider loading
   - Plugin architecture