# Known Issues - RAG Package

## Date: 2025-08-24
## Priority: For future resolution

### 1. EmbeddingProvider Interface Mismatch
**Files Affected:**
- `cost_aware_embedding_service.go`
- `embedding_support.go`
- `enhanced_embedding_service.go`

**Issue:** 
The EmbeddingProvider interface in `enhanced_rag_integration.go` defines different methods than what some implementations expect:
- Main interface has: `GenerateEmbedding`, `GenerateBatchEmbeddings`
- Some implementations expect: `GetEmbeddings`, `IsHealthy`, `GetLatency`

**Temporary Fix:** 
Consider creating adapter interfaces or consolidating to a single interface definition.

### 2. Build Tag Alignment Issues
**Files Affected:**
- Various files use `//go:build !disable_rag && !test`
- Some files use `//go:build rag`
- Others use `//go:build !rag`

**Issue:**
Inconsistent build tags cause compilation issues when different combinations of tags are used.

**Recommendation:**
Standardize on a single build tag strategy across the entire package.

### 3. Duplicate Type Definitions
**Status:** Partially resolved
- ✅ Fixed: RAGCache, RAGMetrics, cacheEntry, contains, LatencyMetrics
- ⚠️ Remaining: Some interface mismatches between stub and real implementations

### 4. Implementation Completeness
Some types have incomplete implementations or missing methods required by interfaces.

## Resolution Strategy
1. Complete interface standardization
2. Align all build tags
3. Remove or properly isolate stub implementations
4. Add comprehensive unit tests to catch these issues early

## Files to Review
- `missing_types.go` - Now requires `rag_stub` build tag
- `weaviate_client_complex.go` - Aligned to `!disable_rag && !test`
- Interface implementations across the package