# Test Data and Fixtures for Conductor-Loop

This directory contains comprehensive test fixtures, sample data, and testing utilities for the conductor-loop implementation.

## Directory Structure

```
testdata/
├── intents/                    # Sample intent files for testing
│   ├── scale-up-intent.json   # Basic scale-up scenario
│   ├── scale-down-intent.json # Basic scale-down scenario
│   ├── complex-oran-intent.json # Complex O-RAN deployment
│   ├── invalid-intent.json    # Invalid JSON for error testing
│   ├── large-intent.json      # Large file for performance testing
│   ├── empty-intent.json      # Empty file edge case
│   └── special-chars-intent.json # Unicode/special character testing
├── mock-executables/          # Mock porch executables
│   ├── mock-porch-success.bat # Always succeeds
│   ├── mock-porch-failure.bat # Always fails
│   └── mock-porch-timeout.bat # Times out
├── helpers/                   # Test utility functions
│   └── test_helpers.go        # Common testing utilities
└── README.md                  # This file
```

## Test Coverage

### Unit Tests

#### 1. State Manager Tests (`internal/loop/state_test.go`)
- **File State Management**
  - IsProcessed/MarkProcessed with various file states
  - State persistence and loading from JSON
  - Corruption recovery and backup creation
  - Hash calculation accuracy and edge cases
- **Concurrency Safety**
  - Concurrent read/write operations
  - Race condition prevention
  - Thread-safe state updates
- **Edge Cases**
  - Empty files, large files, binary files
  - Invalid file paths, permission issues
  - Disk space constraints
- **Performance**
  - Benchmarks for state operations
  - Memory usage optimization
  - Large state file handling

#### 2. File Manager Tests (`internal/loop/manager_test.go`)
- **File Movement Operations**
  - MoveToProcessed with various scenarios
  - MoveToFailed with error capturing
  - Atomic operations on Windows
  - Handle filename conflicts with timestamps
- **Directory Management**
  - Creation and permissions
  - Cleanup of old files
  - Windows path handling
- **Error Scenarios**
  - Disk space issues
  - Permission problems
  - Cross-volume moves
- **Concurrency**
  - Parallel file operations
  - Lock contention handling

#### 3. Porch Executor Tests (`internal/porch/executor_test.go`)
- **Command Execution**
  - Successful executions
  - Failed executions with various exit codes
  - Command not found scenarios
  - Output capture (stdout/stderr)
- **Timeout Handling**
  - Command timeouts
  - Context cancellation
  - Graceful shutdown
- **Configuration Modes**
  - Direct mode vs structured mode
  - Path handling (absolute/relative)
  - Parameter validation
- **Statistics Tracking**
  - Execution counts
  - Success/failure rates
  - Average execution times
  - Timeout tracking

### Integration Tests

#### 4. Watcher Integration Tests (`internal/loop/watcher_test.go`)
- **File Detection**
  - Detection within 2-second requirement
  - Debouncing on rapid file changes
  - Intent file filtering
- **Processing Workflow**
  - End-to-end file processing
  - Idempotent processing (same file twice)
  - Concurrent file processing
  - Failure handling and recovery
- **Modes**
  - Once mode (process and exit)
  - Continuous mode with file watching
  - Graceful shutdown
- **Status File Generation**
  - Success status files
  - Failure status files with error logs
  - JSON format validation

### End-to-End Tests

#### 5. Main Application Tests (`cmd/conductor-loop/main_test.go`)
- **CLI Flag Parsing**
  - Default values
  - Custom configurations
  - Invalid inputs and error handling
- **Full Workflow Testing**
  - File appears → processed → moved
  - Multiple files concurrently
  - Error scenarios end-to-end
- **Platform Compatibility**
  - Windows path handling
  - Signal handling (Unix/Linux)
  - Exit codes for CI integration
- **Performance**
  - Single file processing benchmarks
  - Multiple file processing
  - Resource usage monitoring

## Test Fixtures

### Intent Files

#### Basic Scenarios
- **scale-up-intent.json**: Standard scaling up scenario
- **scale-down-intent.json**: Standard scaling down scenario
- **empty-intent.json**: Empty JSON object for edge case testing

#### Complex Scenarios
- **complex-oran-intent.json**: Full O-RAN deployment with:
  - Multiple network functions (CU-CP, CU-UP, DU, RIC)
  - Complex connectivity requirements
  - SLA specifications
  - Security constraints
  - Placement policies

#### Error Scenarios
- **invalid-intent.json**: Malformed JSON for error testing
- **special-chars-intent.json**: Unicode and special character testing

#### Performance Scenarios
- **large-intent.json**: Large file (~50KB) with:
  - 100+ labels and annotations
  - Multiple network functions
  - Complex configuration parameters
  - Comprehensive monitoring specs

### Mock Executables

#### Windows Batch Files
- **mock-porch-success.bat**: Always exits with code 0
- **mock-porch-failure.bat**: Always exits with code 1
- **mock-porch-timeout.bat**: Sleeps for 60 seconds (timeout testing)

#### Features
- Proper help message handling (`--help`)
- Parameter echoing for verification
- Configurable output (stdout/stderr)
- Platform-specific implementations

### Test Helpers

#### Common Utilities (`testdata/helpers/test_helpers.go`)
- **TestFixtures**: Centralized access to test data
- **Intent File Scenarios**: Programmatic intent generation
- **Mock Executable Creation**: Runtime mock creation
- **Path Handling**: Windows/Unix path compatibility
- **Content Validation**: JSON, size, content validators
- **Performance Scenarios**: Load testing configurations

## Testing Strategy

### Test Pyramid
1. **Many Unit Tests**: Fast, isolated component testing
2. **Fewer Integration Tests**: Component interaction testing  
3. **Minimal E2E Tests**: Full system workflow testing

### Windows Compatibility
- All tests run on Windows with proper path handling
- Batch file mocks for Windows-specific testing
- Unicode/special character support testing
- Long path name handling

### Performance Testing
- Benchmarks for critical paths
- Race condition detection with `-race` flag
- Memory leak detection
- Large file handling tests

### Error Scenarios
- Network failures, disk space issues
- Permission problems
- Malformed input files
- Command execution failures
- Timeout scenarios

## Usage Examples

### Running Specific Test Suites

```bash
# Unit tests only
go test ./internal/loop -v

# Integration tests
go test ./internal/loop -v -tags=integration

# End-to-end tests
go test ./cmd/conductor-loop -v

# All tests with race detection
go test -race ./...

# Benchmarks
go test -bench=. ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Using Test Fixtures

```go
func TestWithFixtures(t *testing.T) {
    fixtures := helpers.NewTestFixtures(t)
    
    // Get common intent files
    intents := fixtures.GetCommonIntents()
    
    // Create mock executable
    mockPorch := fixtures.CreateMockPorch(t, 0, "success", "", 0)
    
    // Setup test environment
    handoffDir, outDir, cleanup := fixtures.SetupTestEnvironment(t)
    defer cleanup()
    
    // Your test logic here...
}
```

### Creating Custom Test Data

```go
func TestCustomIntent(t *testing.T) {
    fixtures := helpers.NewTestFixtures(t)
    
    customIntent := `{
        "apiVersion": "nephoran.com/v1",
        "kind": "NetworkIntent",
        "spec": { "action": "custom" }
    }`
    
    intentFile := fixtures.CreateTempIntent(t, "custom.json", customIntent)
    // Test with custom intent...
}
```

## Test Execution Requirements

### Dependencies
- Go 1.24+
- testify/assert and testify/require
- fsnotify for file watching tests
- Temp directory access for file operations

### Environment
- Windows, Linux, or macOS
- Admin/sudo access not required
- Network access not required (all mocked)
- ~100MB disk space for temporary files

### CI/CD Integration
- All tests pass on Windows Server, Ubuntu, and macOS
- Tests run in parallel where safe
- No external dependencies or services required
- Deterministic results (no flaky tests)

## Maintenance

### Adding New Test Cases
1. Add intent files to `testdata/intents/`
2. Update `test_helpers.go` with new scenarios
3. Add corresponding test functions
4. Update this README

### Performance Monitoring
- Benchmark tests track performance regressions
- Memory usage monitoring for large files
- Execution time tracking for CI optimization

### Cleanup
- All tests use `t.TempDir()` for automatic cleanup
- No permanent files created
- Resource cleanup in defer statements