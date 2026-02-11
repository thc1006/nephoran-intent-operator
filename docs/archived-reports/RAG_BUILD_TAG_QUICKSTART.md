# RAG Build Tag Testing - Quick Start Guide

This guide provides simple commands to test the RAG (Retrieval-Augmented Generation) build tag implementation.

## Overview

The RAG system uses Go build tags for conditional compilation:
- **Without `rag` tag**: Uses lightweight no-op client
- **With `rag` tag**: Uses full Weaviate client implementation

## Quick Test Commands

### 1. Test No-op Implementation (Default)

```bash
# Test interface compliance
go test -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/client_noop.go -run="TestRAGClientInterface"

# Test build tag behavior detection
go test -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/client_noop.go -run="TestBuildTagConditionalBehavior"

# Test concurrent access
go test -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/client_noop.go -run="TestConcurrentAccess"
```

### 2. Test Weaviate Implementation (With rag tag)

```bash
# Test interface compliance
go test -tags="rag" -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/weaviate_client.go -run="TestRAGClientInterface"

# Test build tag behavior detection  
go test -tags="rag" -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/weaviate_client.go -run="TestBuildTagConditionalBehavior"

# Test concurrent access
go test -tags="rag" -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/weaviate_client.go -run="TestConcurrentAccess"
```

### 3. Run All Build Verification Tests

```bash
# No-op implementation (all tests)
go test -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/client_noop.go

# Weaviate implementation (all tests)  
go test -tags="rag" -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/weaviate_client.go
```

### 4. Performance Benchmarks

```bash
# Benchmark no-op implementation
go test -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/client_noop.go -bench="BenchmarkRAGClientOperations" -benchmem

# Benchmark Weaviate implementation
go test -tags="rag" -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/weaviate_client.go -bench="BenchmarkRAGClientOperations" -benchmem
```

### 5. Race Detection

```bash
# Test for race conditions (no-op)
go test -race -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/client_noop.go -run="TestConcurrentAccess"

# Test for race conditions (Weaviate)
go test -tags="rag" -race -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/weaviate_client.go -run="TestConcurrentAccess"
```

## Automated Scripts

### Linux/macOS
```bash
# Run comprehensive test suite
./scripts/test-rag-simple.sh

# Run full test suite (if monitoring package issues are resolved)
./scripts/test-rag-build-tags.sh
```

### Windows
```powershell
# Run comprehensive test suite with options
.\scripts\test-rag-build-tags.ps1 -SkipBenchmarks -SkipCoverage

# Run basic tests only
.\scripts\test-rag-build-tags.ps1 -SkipBenchmarks -SkipCoverage
```

## Expected Results

### No-op Implementation
✅ **Expected Behavior:**
- All operations return immediately without error
- `Retrieve()` returns empty slice `[]Doc{}`
- Zero memory allocations
- Sub-nanosecond performance
- Perfect for testing/minimal deployments

### Weaviate Implementation  
✅ **Expected Behavior:**
- May fail with connection errors (expected without Weaviate server)
- Logs warnings but continues gracefully
- Proper error handling and resource cleanup
- Ready for production with Weaviate instance

## Sample Output

### No-op Success
```
✓ Running with no-op RAG client (without rag build tag)
✓ All concurrent operations completed successfully (no-op client)
BenchmarkRAGClientOperations/Retrieve-8    1000000000   0.32 ns/op   0 B/op   0 allocs/op
```

### Weaviate Success (No Server)
```
✓ Running with Weaviate RAG client (with rag build tag)
Weaviate client error (expected in test environment): connection refused
INFO Weaviate RAG client shut down gracefully
```

## Key Verification Points

1. **Interface Compliance**: Both implementations satisfy `RAGClient` interface
2. **Build Tag Detection**: Correct implementation selected based on build tags
3. **Error Handling**: Graceful degradation when services unavailable
4. **Concurrency Safety**: Thread-safe operations under load
5. **Resource Management**: Proper initialization and cleanup
6. **Performance**: Appropriate performance characteristics for each implementation

## Troubleshooting

### Common Issues

**Build tag not working:**
```bash
# Ensure correct syntax
go test -tags="rag" ./pkg/rag/...
# Not: go test -tag="rag" ./pkg/rag/...
```

**Connection errors with Weaviate:**
- This is expected behavior in test environments
- Tests should pass with connection error logging
- Use mock implementations for unit tests requiring specific responses

**Compilation errors in other packages:**
- Run tests on specific files as shown in commands above
- This avoids compilation issues in unrelated packages

## CI/CD Integration

### Makefile Example
```makefile
test-rag-noop:
	go test -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/client_noop.go

test-rag-weaviate:
	go test -tags="rag" -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/weaviate_client.go

test-rag-all: test-rag-noop test-rag-weaviate
```

### GitHub Actions Example
```yaml
- name: Test No-op RAG
  run: go test -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/client_noop.go
  
- name: Test Weaviate RAG  
  run: go test -tags="rag" -v pkg/rag/build_verification_test.go pkg/rag/types.go pkg/rag/weaviate_client.go
```

## Next Steps

1. **Production Setup**: Configure Weaviate URL and credentials for production builds
2. **Integration Testing**: Use Docker Compose to test with actual Weaviate instance
3. **Performance Tuning**: Benchmark with real Weaviate server and optimize
4. **Monitoring**: Add metrics and alerting for RAG operations
5. **Documentation**: Update deployment guides with build tag selection