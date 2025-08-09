#!/bin/bash

# Circuit Breaker Health Check Test Runner
# This script runs comprehensive tests for the circuit breaker health check fixes

set -e

echo "🔧 Circuit Breaker Health Check Test Suite"
echo "==========================================="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
TEST_TIMEOUT="10m"
VERBOSE=${VERBOSE:-false}
COVERAGE=${COVERAGE:-true}
BENCHMARK=${BENCHMARK:-false}

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "INFO")
            echo -e "${BLUE}ℹ️  $message${NC}"
            ;;
        "SUCCESS")
            echo -e "${GREEN}✅ $message${NC}"
            ;;
        "WARNING")
            echo -e "${YELLOW}⚠️  $message${NC}"
            ;;
        "ERROR")
            echo -e "${RED}❌ $message${NC}"
            ;;
    esac
}

# Function to run tests with error handling
run_test() {
    local test_name=$1
    local test_command=$2
    local description=$3
    
    echo
    print_status "INFO" "Running: $description"
    echo "Command: $test_command"
    echo
    
    if [ "$VERBOSE" = true ]; then
        if eval $test_command; then
            print_status "SUCCESS" "$test_name passed"
        else
            print_status "ERROR" "$test_name failed"
            return 1
        fi
    else
        if eval $test_command > /tmp/${test_name}.log 2>&1; then
            print_status "SUCCESS" "$test_name passed"
        else
            print_status "ERROR" "$test_name failed"
            echo "See /tmp/${test_name}.log for details"
            return 1
        fi
    fi
}

# Function to run benchmarks
run_benchmark() {
    local bench_name=$1
    local bench_command=$2
    local description=$3
    
    echo
    print_status "INFO" "Running benchmark: $description"
    echo "Command: $bench_command"
    echo
    
    if eval $bench_command; then
        print_status "SUCCESS" "$bench_name benchmark completed"
    else
        print_status "WARNING" "$bench_name benchmark failed (non-critical)"
    fi
}

# Ensure we're in the correct directory
if [ ! -f "go.mod" ]; then
    print_status "ERROR" "Please run this script from the project root directory"
    exit 1
fi

# Check if Go is available
if ! command -v go &> /dev/null; then
    print_status "ERROR" "Go is not installed or not in PATH"
    exit 1
fi

print_status "INFO" "Starting Circuit Breaker Health Check Tests"
echo "Test timeout: $TEST_TIMEOUT"
echo "Verbose mode: $VERBOSE"
echo "Coverage enabled: $COVERAGE"
echo "Benchmarks enabled: $BENCHMARK"

# Test 1: Core circuit breaker functionality tests
run_test "core_circuit_breaker" \
    "go test -timeout $TEST_TIMEOUT -v ./pkg/llm -run 'TestCircuitBreaker'" \
    "Core circuit breaker functionality tests"

# Test 2: Circuit breaker health check unit tests  
run_test "circuit_breaker_health" \
    "go test -timeout $TEST_TIMEOUT -v ./pkg/llm -run 'TestCircuitBreakerManagerGetAllStats|TestCircuitBreakerHealthCheckLogic|TestCircuitBreakerStatsFormat'" \
    "Circuit breaker health check logic tests"

# Test 3: Circuit breaker concurrent tests
run_test "circuit_breaker_concurrent" \
    "go test -timeout $TEST_TIMEOUT -v ./pkg/llm -run 'TestCircuitBreakerConcurrentHealthChecks'" \
    "Circuit breaker concurrent access tests"

# Test 4: Service manager integration tests
run_test "service_manager_integration" \
    "go test -timeout $TEST_TIMEOUT -v ./cmd/llm-processor -run 'TestCircuitBreakerHealthIntegration'" \
    "Service manager circuit breaker health integration tests"

# Test 5: End-to-end health check behavior
run_test "e2e_health_check" \
    "go test -timeout $TEST_TIMEOUT -v ./cmd/llm-processor -run 'TestCircuitBreakerHealthIntegration/EndToEndHealthCheckBehavior'" \
    "End-to-end health check HTTP endpoint tests"

# Test 6: Concurrent state changes during health checks
run_test "concurrent_state_changes" \
    "go test -timeout $TEST_TIMEOUT -v ./cmd/llm-processor -run 'TestCircuitBreakerHealthIntegration/ConcurrentStateChanges'" \
    "Health checks during concurrent state changes"

# Test 7: Performance tests with many circuit breakers
run_test "performance_many_breakers" \
    "go test -timeout $TEST_TIMEOUT -v ./cmd/llm-processor -run 'TestCircuitBreakerHealthIntegration/PerformanceWithManyBreakers'" \
    "Performance tests with many circuit breakers"

# Test 8: Regression tests
run_test "regression_tests" \
    "go test -timeout $TEST_TIMEOUT -v ./cmd/llm-processor -run 'TestCircuitBreakerHealthIntegration/RegressionTests'" \
    "Regression tests for backward compatibility"

# Test 9: Message formatting tests
run_test "message_formatting" \
    "go test -timeout $TEST_TIMEOUT -v ./cmd/llm-processor -run 'TestCircuitBreakerHealthMessageFormatting'" \
    "Circuit breaker health message formatting tests"

# Test 10: Existing service manager tests (to ensure no regression)
run_test "existing_service_manager" \
    "go test -timeout $TEST_TIMEOUT -v ./cmd/llm-processor -run 'TestCircuitBreakerHealthValidation|TestRegisterHealthChecksIntegration'" \
    "Existing service manager health check tests"

# Coverage tests (if enabled)
if [ "$COVERAGE" = true ]; then
    echo
    print_status "INFO" "Running coverage analysis"
    
    # Generate coverage for circuit breaker health check fixes
    if go test -coverprofile=/tmp/circuit_breaker_health.out -coverpkg=./pkg/llm,./cmd/llm-processor ./pkg/llm ./cmd/llm-processor -run 'TestCircuitBreaker.*Health|TestCircuitBreakerManager.*Stats' > /tmp/coverage.log 2>&1; then
        
        # Display coverage summary
        coverage_percent=$(go tool cover -func=/tmp/circuit_breaker_health.out | grep total | awk '{print $3}')
        print_status "INFO" "Circuit breaker health check coverage: $coverage_percent"
        
        # Generate HTML coverage report
        if go tool cover -html=/tmp/circuit_breaker_health.out -o /tmp/circuit_breaker_health_coverage.html; then
            print_status "SUCCESS" "Coverage report generated: /tmp/circuit_breaker_health_coverage.html"
        fi
    else
        print_status "WARNING" "Coverage analysis failed (see /tmp/coverage.log)"
    fi
fi

# Benchmark tests (if enabled)
if [ "$BENCHMARK" = true ]; then
    echo
    print_status "INFO" "Running performance benchmarks"
    
    # Circuit breaker health check benchmarks
    run_benchmark "circuit_breaker_health_bench" \
        "go test -bench=BenchmarkCircuitBreakerHealthCheck -benchmem -benchtime=5s ./pkg/llm" \
        "Circuit breaker health check performance"
    
    run_benchmark "circuit_breaker_fix_bench" \
        "go test -bench=BenchmarkCircuitBreakerHealthCheckFix -benchmem -benchtime=5s ./cmd/llm-processor" \
        "Circuit breaker fix performance validation"
    
    run_benchmark "circuit_breaker_stats_bench" \
        "go test -bench=BenchmarkCircuitBreakerHealthCheckPerformance -benchmem -benchtime=3s ./pkg/llm" \
        "GetAllStats method performance"
fi

# Summary
echo
echo "============================================"
print_status "SUCCESS" "Circuit Breaker Health Check Tests Complete"
echo

# Test summary commands
echo "📊 Test Summary Commands:"
echo "  View all logs: ls -la /tmp/*circuit_breaker* /tmp/*health*"
echo "  Coverage report: open /tmp/circuit_breaker_health_coverage.html"
echo "  Re-run specific test: go test -v ./pkg/llm -run TestCircuitBreakerManagerGetAllStats"
echo "  Re-run with race detection: go test -race -v ./pkg/llm ./cmd/llm-processor"
echo

# Additional validation suggestions
echo "🔍 Additional Validation Steps:"
echo "  1. Manual testing: Start service and check /healthz endpoint"
echo "  2. Load testing: Use ab or curl to test health endpoint under load"
echo "  3. Monitor logs for race conditions or deadlocks"
echo "  4. Verify fix in production-like environment"
echo

print_status "SUCCESS" "All circuit breaker health check tests completed successfully!"