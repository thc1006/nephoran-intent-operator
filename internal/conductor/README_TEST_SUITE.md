# Conductor Watch Controller Test Suite

## Overview

This directory contains comprehensive integration and unit tests for the conductor-watch controller, which processes NetworkIntent custom resources and generates intent JSON files for the porch CLI to convert into KRM (Kubernetes Resource Model) files.

## Test Architecture

### 1. Dependency Injection Pattern

The reconciler has been enhanced with a `PorchExecutor` interface to enable dependency injection for testing:

```go
type PorchExecutor interface {
    ExecutePorch(ctx context.Context, porchPath string, args []string, outputDir string, intentFile string, mode string) error
}
```

This allows us to:
- Mock porch CLI calls during testing
- Verify exact arguments passed to porch
- Simulate porch failures for error handling tests
- Track call history for validation

### 2. Test Files Structure

```
internal/conductor/
├── reconciler.go                 # Main reconciler implementation  
├── reconciler_test.go            # Original unit tests (parsing, file creation)
├── conductor_test.go             # Comprehensive unit tests with fake K8s client
├── integration_test.go           # Integration tests using Ginkgo/Gomega
└── README_TEST_SUITE.md          # This documentation
```

## Test Categories

### Unit Tests (`conductor_test.go`)

**TestConductorReconcileFlow**
- Tests complete reconcile flow with various intent strings
- Covers success cases: deployment scaling, service scaling, instances vs replicas
- Covers error cases: invalid intents, porch CLI failures
- Uses fake K8s client and mock porch executor
- Verifies JSON generation, file creation, and KRM output

**TestConductorIdempotency**  
- Tests reconciler behavior on repeated calls with same intent
- Verifies that each reconcile creates new timestamped files
- Documents current behavior vs ideal idempotent behavior

**TestMultipleNetworkIntentsProcessing**
- Tests processing multiple NetworkIntents in different namespaces
- Verifies correct parsing and isolation between intents
- Validates JSON content matches expected targets and replica counts

**TestConductorErrorHandling**
- Tests graceful handling of missing NetworkIntents  
- Tests file system permission issues (Unix-only)
- Verifies proper error recovery with requeue behavior

**TestPorchExecutorInterface**
- Validates interface compliance for both real and mock executors
- Ensures dependency injection pattern works correctly

### Integration Tests (`integration_test.go`)

Uses Ginkgo BDD framework with fake Kubernetes client (avoids envtest dependency).

**Context: NetworkIntent Add Event**
- Tests complete flow from intent creation to porch execution
- Verifies JSON schema compliance with `docs/contracts/intent.schema.json`
- Tests complex scaling intents with various formats
- Validates metadata fields (correlation_id, source, reason)

**Context: NetworkIntent Update Event**  
- Tests spec changes trigger new JSON generation
- Verifies updated replica counts in generated files
- Confirms separate porch calls for each update

**Context: Idempotency**
- Documents current non-idempotent behavior
- Tests repeated reconciles with unchanged specs
- Verifies file generation patterns

**Context: Multiple NetworkIntents**
- Tests concurrent processing of multiple intents
- Verifies correct target extraction and replica parsing
- Validates namespace isolation

**Context: Error Handling**
- Tests invalid intent string processing with graceful requeue
- Tests porch CLI failure scenarios
- Verifies proper error logging and recovery

**Context: File System Operations**
- Tests automatic directory creation
- Tests file permission handling
- Verifies output file generation

**Context: Requeue Behavior**
- Tests requeue on persistent failures
- Verifies 5-minute requeue delays
- Tests multiple retry attempts

## Mock Components

### MockPorchExecutor
- Tracks all porch CLI calls with full argument lists
- Simulates successful KRM file generation  
- Configurable failure scenarios for error testing
- Call history for verification

### MockPorchExecutorSimple  
- Simplified mock for unit tests
- Integrates with testify/mock for assertion patterns
- Records call counts and timing
- Generates fake KRM output files

## Key Test Scenarios Covered

### 1. Intent Parsing Validation
```go
// Tests various intent formats:
"scale deployment web-server to 3 replicas"
"scale service api-gateway to 5 instances" 
"scale app nginx replicas: 2"
"scale service app3 to 1 replicas"  // Fixed regex issue
```

### 2. Error Handling
- Missing deployment targets → requeue with 5min delay
- Excessive replica counts (>100) → requeue  
- Porch CLI failures → requeue with error logging
- File system permission issues → graceful handling

### 3. JSON Schema Compliance
All generated JSON validates against `docs/contracts/intent.schema.json`:
```json
{
  "intent_type": "scaling",
  "target": "deployment-name", 
  "namespace": "target-namespace",
  "replicas": 3,
  "source": "user",
  "correlation_id": "unique-id",
  "reason": "Generated from NetworkIntent..."
}
```

### 4. Integration Flow Validation
1. NetworkIntent creation → reconcile trigger
2. Intent parsing → JSON generation  
3. File system operations → porch CLI execution
4. KRM output verification → success logging

## Running the Tests

### All Tests
```bash
go test ./internal/conductor -v
```

### Specific Test Categories  
```bash
# Unit tests only
go test ./internal/conductor -v -run "^Test[^C]"

# Integration tests only  
go test ./internal/conductor -v -run TestConductorIntegration

# Specific scenarios
go test ./internal/conductor -v -run TestConductorReconcileFlow
go test ./internal/conductor -v -run TestMultiple
```

### Test Coverage
```bash
go test ./internal/conductor -v -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Output Examples

### Successful Test Run
```
=== RUN   TestConductorReconcileFlow
=== RUN   TestConductorReconcileFlow/successful_scaling_intent
=== RUN   TestConductorReconcileFlow/deployment_with_instances_keyword
--- PASS: TestConductorReconcileFlow (0.13s)

Running Suite: Conductor Integration Suite
Will run 10 of 10 specs
+++++++++S+

Ran 9 of 10 Specs in 0.325 seconds
SUCCESS! -- 9 Passed | 0 Failed | 0 Pending | 1 Skipped
```

### Failed Test with Details
Tests provide detailed error messages showing:
- Expected vs actual values
- File system state 
- Mock call history
- JSON content validation failures

## Implementation Notes

### Regex Pattern Improvements
Fixed intent parsing issue where `scale.*?(\d+)` was matching digits in deployment names instead of replica counts. Now uses prioritized patterns:

1. `to\s+(\d+)\s+(?:replicas?|instances?)` - Most specific
2. `(\d+)\s+(?:replicas?|instances?)(?:\s|$)` - With word boundaries
3. `replicas?:?\s*(\d+)` - Colon syntax

### Fake Client vs EnvTest
Uses fake Kubernetes client instead of controller-runtime's envtest to:
- Avoid requiring kubebuilder binaries installation  
- Enable running tests in any environment
- Faster test execution
- Simpler CI/CD integration

### Timestamped File Generation
Each reconcile creates files with Unix timestamps to avoid conflicts:
```
intent-${name}-${timestamp}.json
```

This enables testing concurrent reconciles and updates.

## Future Enhancements

### Potential Improvements
1. **True Idempotency**: Add status tracking to avoid reprocessing unchanged intents
2. **Enhanced Validation**: Add more JSON schema validation in tests  
3. **Performance Tests**: Add benchmarks for large-scale intent processing
4. **Real EnvTest**: Optional envtest integration for full K8s API testing
5. **Chaos Testing**: Inject random failures to test resilience

### Test Expansion Ideas
- Webhook validation testing
- Status field updates validation  
- Event emission verification
- Metrics collection testing
- Resource cleanup verification

## Troubleshooting

### Common Issues
1. **Permission denied**: Tests create temp directories - ensure write permissions
2. **Port conflicts**: Integration tests don't use ports, but watch for race conditions
3. **Timing issues**: Tests use proper event waiting, but system load can affect timing
4. **Windows vs Unix**: File permission tests are skipped on Windows

### Debug Tips
- Use `go test -v` for verbose output with step-by-step execution
- Check temp directories for generated files when debugging
- Review mock call history in test failures
- Validate JSON against schema using external tools if needed

This test suite provides comprehensive coverage of the conductor-watch controller functionality while maintaining fast execution and reliable results across different environments.