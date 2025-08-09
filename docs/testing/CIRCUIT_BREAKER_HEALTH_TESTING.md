# Circuit Breaker Health Check Testing Guide

This document provides comprehensive testing guidance for validating the circuit breaker health check fixes in the Nephoran Intent Operator project.

## Overview

The circuit breaker health check implementation was fixed to address an issue where the service manager would return early when encountering the first open circuit breaker, instead of collecting ALL open breakers before returning. The fix ensures that health check responses properly report all open circuit breakers in the format: `"Circuit breakers in open state: [breaker1 breaker2]"`.

## Test Coverage

### 1. Integration Tests (`cmd/llm-processor/circuit_breaker_health_integration_test.go`)

**File**: `cmd/llm-processor/circuit_breaker_health_integration_test.go`

This comprehensive integration test suite validates the circuit breaker health check fixes across multiple scenarios:

#### Test Categories:

**A. End-to-End Health Check Behavior**
- Tests actual HTTP health endpoints (`/healthz`, `/readyz`)
- Validates response format and status codes
- Tests with multiple circuit breakers in various states (closed, open, half-open)
- Ensures all open breakers are properly reported in health responses

**B. Concurrent Circuit Breaker State Changes**
- Tests health checks while circuit breakers are transitioning states
- Validates thread-safety and absence of race conditions
- Ensures health check accuracy during concurrent state modifications
- Tests with 20 concurrent goroutines performing health checks and state changes

**C. Performance Tests**
- Benchmarks health check performance with many circuit breakers (10, 50, 100, 200)
- Ensures no performance degradation with the fix
- Validates sub-50ms response times even with 200+ circuit breakers
- Tests linear scalability of the health check system

**D. Regression Tests**
- Ensures the fix doesn't break existing functionality
- Tests backward compatibility with existing health check infrastructure
- Validates proper handling of edge cases (nil managers, malformed stats)
- Confirms HTTP endpoint integration remains functional

#### Key Test Functions:

```go
// Tests actual HTTP endpoints with multiple circuit breaker states
func testEndToEndHealthCheckBehavior(t *testing.T, logger *slog.Logger)

// Tests concurrent health checks during state changes
func testConcurrentStateChanges(t *testing.T, logger *slog.Logger) 

// Benchmarks performance with many circuit breakers
func testPerformanceWithManyBreakers(t *testing.T, logger *slog.Logger)

// Validates backward compatibility
func testRegressionTests(t *testing.T, logger *slog.Logger)
```

### 2. Unit Tests (`pkg/llm/circuit_breaker_health_test.go`)

**File**: `pkg/llm/circuit_breaker_health_test.go`

Focused unit tests for circuit breaker health check logic:

#### Test Categories:

**A. Circuit Breaker Manager Stats**
- Tests `GetAllStats()` method used by health checks
- Validates correct identification of open circuit breakers
- Tests various combinations of circuit breaker states
- Ensures proper stats format for health check parsing

**B. Health Check Logic Validation**
- Tests the exact logic used in service manager health checks
- Validates that ALL open breakers are collected (tests the fix)
- Tests message formatting for different scenarios
- Ensures proper handling of mixed circuit breaker states

**C. Concurrent Access Tests**
- Tests concurrent access to `GetAllStats()` while states change
- Validates thread-safety of circuit breaker statistics
- Tests with 20 circuit breakers and multiple goroutines
- Ensures no data corruption during concurrent operations

**D. Performance Benchmarks**
- Benchmarks `GetAllStats()` performance with varying numbers of breakers
- Tests with 10, 50, 100, 200, and 500 circuit breakers
- Validates sub-1ms performance requirements
- Ensures linear scaling characteristics

#### Key Test Functions:

```go
// Tests GetAllStats method used by health checks
func TestCircuitBreakerManagerGetAllStats(t *testing.T)

// Tests the exact health check logic that was fixed
func TestCircuitBreakerHealthCheckLogic(t *testing.T)

// Tests concurrent access patterns
func TestCircuitBreakerConcurrentHealthChecks(t *testing.T)

// Benchmarks the fixed health check implementation
func BenchmarkCircuitBreakerHealthCheck(b *testing.B)
```

### 3. Test Runner Script (`scripts/run_circuit_breaker_tests.sh`)

**File**: `scripts/run_circuit_breaker_tests.sh`

Automated test execution script that runs all circuit breaker health check tests:

#### Features:
- **Comprehensive Test Execution**: Runs all related tests in sequence
- **Coverage Analysis**: Generates coverage reports for health check code
- **Performance Benchmarking**: Executes benchmarks to validate performance
- **Error Handling**: Provides detailed feedback on test failures
- **Configurable Options**: Supports verbose mode, coverage, and benchmarking flags

#### Usage:
```bash
# Run all tests with default settings
./scripts/run_circuit_breaker_tests.sh

# Run with verbose output and benchmarks
VERBOSE=true BENCHMARK=true ./scripts/run_circuit_breaker_tests.sh

# Run with coverage disabled
COVERAGE=false ./scripts/run_circuit_breaker_tests.sh
```

## Test Scenarios

### 1. Basic Functionality Tests

**Scenario**: Multiple circuit breakers with different states
```
- service-a: closed
- service-b: open  
- service-c: half-open
- service-d: open
```

**Expected Result**: Health check reports `"Circuit breakers in open state: [service-b service-d]"`

### 2. Concurrent State Change Tests

**Scenario**: Health checks running while circuit breakers transition between states
- 10 goroutines performing health checks every 50ms
- 5 goroutines changing circuit breaker states every 100ms
- Test duration: 2 seconds

**Expected Result**: All health check results are valid, no race conditions detected

### 3. Performance Tests

**Scenario**: Health checks with large numbers of circuit breakers
- Test with 10, 50, 100, 200, and 500 circuit breakers
- 20% of breakers in open state
- Measure average health check latency

**Expected Result**: Sub-50ms health check latency even with 200+ circuit breakers

### 4. Edge Case Tests

**Scenarios**:
- No circuit breakers registered
- All circuit breakers in same state (all open, all closed)
- Malformed circuit breaker statistics
- Nil circuit breaker manager

**Expected Results**: Graceful handling of all edge cases without crashes

## Validation Commands

### Run All Tests
```bash
# Execute comprehensive test suite
./scripts/run_circuit_breaker_tests.sh
```

### Run Specific Test Categories
```bash
# Core circuit breaker functionality
go test -v ./pkg/llm -run 'TestCircuitBreaker'

# Health check logic tests
go test -v ./pkg/llm -run 'TestCircuitBreakerHealthCheckLogic'

# Integration tests
go test -v ./cmd/llm-processor -run 'TestCircuitBreakerHealthIntegration'

# Performance benchmarks
go test -bench=BenchmarkCircuitBreakerHealthCheck -benchmem ./pkg/llm
```

### Race Condition Detection
```bash
# Run tests with race detector
go test -race -v ./pkg/llm ./cmd/llm-processor
```

### Coverage Analysis
```bash
# Generate coverage report
go test -coverprofile=coverage.out ./pkg/llm ./cmd/llm-processor
go tool cover -html=coverage.out -o coverage.html
```

## Expected Test Results

### Success Criteria

1. **Functional Correctness**
   - All circuit breaker health check tests pass
   - Health checks correctly identify ALL open circuit breakers
   - Proper message formatting: `"Circuit breakers in open state: [breaker1 breaker2]"`

2. **Performance Requirements**
   - Health check latency < 50ms with 200+ circuit breakers
   - `GetAllStats()` method < 1ms with 500+ circuit breakers
   - Linear scaling characteristics maintained

3. **Concurrency Safety**
   - No race conditions detected in concurrent tests
   - Thread-safe access to circuit breaker statistics
   - Consistent results under concurrent load

4. **Backward Compatibility**
   - All existing health check functionality preserved
   - HTTP endpoints return expected response format
   - No breaking changes to health check API

### Performance Benchmarks

**Target Performance Characteristics**:
- **Small Scale** (10 breakers): < 1ms health check
- **Medium Scale** (50 breakers): < 10ms health check  
- **Large Scale** (100 breakers): < 25ms health check
- **Extra Large** (200+ breakers): < 50ms health check

### Memory Usage
- No memory leaks during extended test runs
- Reasonable memory usage scaling with number of circuit breakers
- Efficient garbage collection behavior

## Troubleshooting

### Common Issues

1. **Test Timeouts**
   - Increase test timeout if running on slower systems
   - Check for deadlocks in concurrent tests

2. **Race Condition Warnings**
   - Run with `-race` flag to identify data races
   - Review concurrent access patterns in circuit breaker code

3. **Performance Degradation**
   - Profile health check performance with `go test -cpuprofile`
   - Check for inefficient loops or excessive memory allocation

4. **Flaky Tests**
   - Review timing assumptions in concurrent tests
   - Increase stabilization delays if needed

### Debug Commands
```bash
# Run tests with detailed output
go test -v -race ./pkg/llm ./cmd/llm-processor

# Profile CPU usage
go test -cpuprofile=cpu.prof -bench=BenchmarkCircuitBreakerHealthCheck ./pkg/llm

# Profile memory usage  
go test -memprofile=mem.prof -bench=BenchmarkCircuitBreakerHealthCheck ./pkg/llm

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof
```

## Continuous Integration

### Pre-commit Hooks
Add circuit breaker health tests to pre-commit validation:
```bash
#!/bin/bash
# Run critical circuit breaker tests before commit
go test -race -timeout=5m ./pkg/llm -run 'TestCircuitBreakerHealthCheckLogic'
go test -race -timeout=5m ./cmd/llm-processor -run 'TestCircuitBreakerHealthIntegration'
```

### CI Pipeline Integration
```yaml
- name: Circuit Breaker Health Tests
  run: |
    ./scripts/run_circuit_breaker_tests.sh
    # Upload coverage reports
    # Store benchmark results for regression tracking
```

## Conclusion

This comprehensive testing approach ensures the circuit breaker health check fixes are thoroughly validated across multiple dimensions:

- **Functional correctness** through integration and unit tests
- **Performance characteristics** through benchmarking
- **Concurrency safety** through race condition testing
- **Backward compatibility** through regression testing

The test suite provides confidence that the fix properly addresses the original issue while maintaining system reliability and performance.