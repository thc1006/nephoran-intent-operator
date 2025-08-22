# Optimized Go Test Commands for Nephoran Intent Operator

## For Development and CI/CD

### 1. Fast Tests (under 30 seconds)
```powershell
# Quick unit tests with minimal parallelization
go test -timeout=30s -short -parallel=2 ./...
```

### 2. Complete Test Suite (sequential for reliability)
```powershell
# Run all tests sequentially to avoid resource conflicts
go test -timeout=5m -parallel=1 ./...
```

### 3. Package-Specific Testing
```powershell
# Test conductor-loop package (most critical)
go test -timeout=2m -v ./cmd/conductor-loop -parallel=2

# Test internal loop package
go test -timeout=2m -v ./internal/loop -parallel=4

# Test other cmd packages individually
go test -timeout=1m -v ./cmd/conductor
go test -timeout=1m -v ./cmd/conductor-watch
```

### 4. Parallel Testing with Safe Limits
```powershell
# Safe parallel testing for CI
go test -timeout=3m -parallel=4 ./cmd/conductor-loop ./internal/loop ./cmd/conductor

# Maximum parallelization for powerful machines
go test -timeout=5m -parallel=8 ./...
```

### 5. Specific Problem Areas
```powershell
# Test just the fixed EndToEndWorkflow tests
go test -timeout=1m -v ./cmd/conductor-loop -run "TestMain_EndToEndWorkflow"

# Test validation and security tests separately
go test -timeout=2m -v ./cmd/conductor-loop -run "TestValidation|TestSecurity"

# Skip problematic long-running tests in CI
go test -timeout=2m -short ./...
```

## Troubleshooting Commands

### Debug Hanging Tests
```powershell
# Run with race detection and verbose output
go test -race -timeout=1m -v ./cmd/conductor-loop

# Run specific test with stack traces on timeout
go test -timeout=30s -v ./cmd/conductor-loop -run "TestSpecificFailingTest"
```

### Performance Testing
```powershell
# Benchmark tests
go test -bench=. -timeout=5m ./cmd/conductor-loop

# Memory profiling
go test -memprofile=mem.prof -timeout=2m ./cmd/conductor-loop
```

## CI/CD Pipeline Recommendations

1. **Use sequential testing** (`-parallel=1`) for release builds
2. **Set conservative timeouts** (5+ minutes for full suite)
3. **Split test execution** by package for better isolation
4. **Use short tests** (`-short`) for quick feedback loops
5. **Run validation tests separately** from unit tests

## Windows-Specific Considerations

1. **File locking**: Unique binary names prevent conflicts
2. **Path handling**: All tests now use absolute paths
3. **Process cleanup**: Proper signal handling and resource cleanup
4. **Antivirus interference**: Exclude test temp directories if needed

## Example CI Pipeline Stage
```yaml
test:
  script:
    - go test -timeout=30s -short -parallel=2 ./...  # Quick smoke test
    - go test -timeout=3m -parallel=1 ./cmd/conductor-loop  # Critical path
    - go test -timeout=5m -parallel=1 ./...  # Full suite
```