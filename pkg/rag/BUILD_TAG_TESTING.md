# RAG Build Tag Testing Guide

This document explains how to test the RAG (Retrieval-Augmented Generation) build tag implementation in the Nephoran Intent Operator. The RAG system uses Go build tags to conditionally compile either a full Weaviate-based implementation or a lightweight no-op implementation.

## Overview

The RAG system supports two build configurations:

1. **No-op Implementation** (default, no build tags): Lightweight stub that returns empty results
2. **Weaviate Implementation** (with `rag` build tag): Full RAG implementation using Weaviate vector database

## Build Tag Architecture

### Files and Build Tags

- `types.go` - Common interfaces and types (no build tags)
- `client_noop.go` - No-op implementation (`//go:build !rag`)  
- `weaviate_client.go` - Weaviate implementation (`//go:build rag`)
- `build_verification_test.go` - Comprehensive tests for both configurations

### Interface Definition

Both implementations satisfy the `RAGClient` interface:

```go
type RAGClient interface {
    Retrieve(ctx context.Context, query string) ([]Doc, error)
    Initialize(ctx context.Context) error
    Shutdown(ctx context.Context) error
}
```

## Running Tests

### Quick Test Commands

```bash
# Test no-op implementation (default)
go test ./pkg/rag/ -run="BuildVerification"

# Test Weaviate implementation
go test -tags="rag" ./pkg/rag/ -run="BuildVerification"

# Test all RAG functionality without build tags
go test ./pkg/rag/...

# Test all RAG functionality with rag build tag
go test -tags="rag" ./pkg/rag/...
```

### Automated Test Scripts

Two test scripts are provided for comprehensive testing:

#### Linux/macOS (Bash)
```bash
./scripts/test-rag-build-tags.sh
```

#### Windows (PowerShell)
```powershell
.\scripts\test-rag-build-tags.ps1

# With options
.\scripts\test-rag-build-tags.ps1 -SkipBenchmarks -SkipCoverage
```

### Test Script Features

The automated scripts perform:

1. **Interface Verification** - Confirms both implementations satisfy the RAGClient interface
2. **Behavioral Testing** - Validates expected behavior differences between implementations
3. **Configuration Testing** - Tests various configuration scenarios and edge cases
4. **Concurrency Testing** - Verifies thread-safety of both implementations
5. **Performance Benchmarks** - Measures performance characteristics
6. **Race Detection** - Tests for race conditions using `go test -race`
7. **Coverage Analysis** - Generates coverage reports for both configurations

## Test Structure

### Test Files

- `build_verification_test.go` - Main test suite for build tag verification
- `build_test.go` - Legacy build mode tests (may contain outdated interface tests)

### Key Test Functions

#### TestRAGClientInterface
Tests that both implementations properly satisfy the RAGClient interface:
- Interface compliance verification
- Method behavior validation
- Error handling consistency
- Configuration handling

#### TestBuildTagConditionalBehavior
Tests implementation-specific behavior:
- No-op client always returns empty results without errors
- Weaviate client handles connection failures gracefully
- Proper error propagation and logging

#### TestRAGClientConfigDefaults
Tests configuration handling:
- Nil configuration handling
- Empty configuration handling
- Partial configuration handling
- Default value application

#### TestConcurrentAccess
Tests concurrent usage:
- Multiple goroutines calling Retrieve simultaneously
- Proper synchronization and thread-safety
- No race conditions or data corruption

#### BenchmarkRAGClientOperations
Performance benchmarks:
- Retrieve operation throughput
- Initialize operation overhead
- Shutdown operation cleanup

## Expected Behaviors

### No-op Implementation (without `rag` build tag)

**Expected behavior:**
- `Initialize()` - Always succeeds, no setup required
- `Retrieve()` - Always returns empty slice `[]Doc{}`, never errors
- `Shutdown()` - Always succeeds, no cleanup required
- Zero external dependencies
- Minimal memory footprint
- Consistent sub-microsecond response times

**Use cases:**
- Testing environments without Weaviate
- Minimal deployments where RAG is not needed
- CI/CD pipelines with dependency constraints
- Air-gapped deployments

### Weaviate Implementation (with `rag` build tag)

**Expected behavior:**
- `Initialize()` - Attempts connection to Weaviate, logs warnings on failure but doesn't error
- `Retrieve()` - Performs GraphQL queries to Weaviate, returns structured results or connection errors
- `Shutdown()` - Closes HTTP connections and cleans up resources
- Full Weaviate dependency with HTTP client
- Configurable timeouts, retry logic, and authentication
- Response times depend on Weaviate performance and network latency

**Use cases:**
- Production deployments with full RAG capabilities
- Development environments with Weaviate instance
- Integration testing with vector database
- Performance testing and optimization

## CI/CD Integration

### GitHub Actions Example

```yaml
name: RAG Build Tag Tests
on: [push, pull_request]

jobs:
  test-no-op:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - name: Test No-op Implementation
        run: go test -v ./pkg/rag/ -run="BuildVerification"
      - name: Race Detection
        run: go test -race ./pkg/rag/ -run="Concurrent"

  test-weaviate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - name: Test Weaviate Implementation
        run: go test -tags="rag" -v ./pkg/rag/ -run="BuildVerification"
      - name: Race Detection with RAG
        run: go test -tags="rag" -race ./pkg/rag/ -run="Concurrent"

  coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - name: Coverage No-op
        run: |
          go test -coverprofile=coverage-noop.out ./pkg/rag/...
          go tool cover -func=coverage-noop.out
      - name: Coverage Weaviate
        run: |
          go test -tags="rag" -coverprofile=coverage-rag.out ./pkg/rag/...
          go tool cover -func=coverage-rag.out
```

### Makefile Integration

```makefile
.PHONY: test-rag-noop test-rag-weaviate test-rag-all

test-rag-noop:
	go test -v ./pkg/rag/ -run="BuildVerification"
	go test -race ./pkg/rag/ -run="Concurrent"

test-rag-weaviate:
	go test -tags="rag" -v ./pkg/rag/ -run="BuildVerification"
	go test -tags="rag" -race ./pkg/rag/ -run="Concurrent"

test-rag-all: test-rag-noop test-rag-weaviate
	@echo "All RAG build tag tests completed"

test-rag-coverage:
	go test -coverprofile=coverage-noop.out ./pkg/rag/...
	go test -tags="rag" -coverprofile=coverage-rag.out ./pkg/rag/...
	go tool cover -func=coverage-noop.out | tail -1
	go tool cover -func=coverage-rag.out | tail -1
```

## Troubleshooting

### Common Issues

#### Import Cycle Errors
If you encounter import cycle errors:
- Ensure test files are in the same package (`package rag`)
- Avoid importing the package being tested in test files
- Use interfaces and dependency injection for external dependencies

#### Build Tag Not Recognized
```bash
# Ensure proper build tag syntax
go test -tags="rag" ./pkg/rag/...

# Not this:
go test -tag="rag" ./pkg/rag/...  # Wrong: -tag instead of -tags
go test -tags rag ./pkg/rag/...   # Works but quotes are recommended
```

#### Weaviate Connection Errors in Tests
This is expected behavior when no Weaviate instance is available:
- Tests should handle connection errors gracefully
- Use mock implementations for unit tests requiring specific responses
- Integration tests can skip or mark as expected failures when Weaviate is unavailable

### Debug Commands

```bash
# List all test functions
go test -v ./pkg/rag/ -list=.*

# Run specific test with verbose output
go test -v ./pkg/rag/ -run="TestBuildTagConditionalBehavior"

# Run with race detection and verbose output
go test -race -v ./pkg/rag/ -run="TestConcurrentAccess"

# Generate detailed coverage HTML report
go test -coverprofile=coverage.out ./pkg/rag/...
go tool cover -html=coverage.out -o coverage.html
```

## Performance Characteristics

### No-op Implementation
- **Initialize**: < 1μs
- **Retrieve**: < 1μs  
- **Shutdown**: < 1μs
- **Memory**: ~100 bytes baseline
- **Goroutines**: 0 additional

### Weaviate Implementation
- **Initialize**: 10-100ms (network dependent)
- **Retrieve**: 50-500ms (network + Weaviate processing)
- **Shutdown**: 1-10ms (connection cleanup)
- **Memory**: ~10KB baseline + HTTP buffers
- **Goroutines**: 2-4 for HTTP client

## Best Practices

### Test Development
1. **Test Both Implementations** - Always run tests with and without build tags
2. **Handle Connection Failures** - Weaviate tests should gracefully handle unavailable services
3. **Use Table-Driven Tests** - Test multiple configurations and scenarios
4. **Mock External Dependencies** - Use interfaces and mocks for unit testing
5. **Validate Interface Compliance** - Ensure both implementations satisfy the interface

### Production Deployment
1. **Build Tag Selection** - Use `rag` tag only when RAG functionality is needed
2. **Configuration Validation** - Validate RAG configuration at startup
3. **Graceful Degradation** - Handle RAG failures without breaking core functionality
4. **Monitoring** - Monitor RAG performance and error rates
5. **Resource Management** - Properly initialize and shutdown RAG clients

### CI/CD Configuration
1. **Test Both Configurations** - Include both no-op and Weaviate tests
2. **Race Detection** - Always run race detection on concurrent tests
3. **Coverage Goals** - Maintain >90% coverage for both implementations
4. **Performance Regression** - Use benchmarks to detect performance regressions
5. **Dependency Isolation** - Test no-op implementation in environments without Weaviate dependencies