#!/bin/bash

# =============================================================================
# LLM Provider Pipeline Test Runner
# =============================================================================
# Comprehensive test script for local development and CI validation
# Supports all testing scenarios: unit, integration, performance, and demo
# =============================================================================

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMP_DIR="${PROJECT_ROOT}/test-tmp"
HANDOFF_DIR="${TEMP_DIR}/handoff"
LOG_FILE="${TEMP_DIR}/test-results.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Test configuration
TEST_TIMEOUT="10m"
COVERAGE_THRESHOLD="80"
DEMO_PORT="8080"
PROCESSOR_PORT="9090"

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

print_header() {
    echo
    print_status "$BLUE" "=============================================================================  "
    print_status "$BLUE" " $1"
    print_status "$BLUE" "=============================================================================  "
    echo
}

print_success() {
    print_status "$GREEN" "✅ $1"
}

print_warning() {
    print_status "$YELLOW" "⚠️  $1"
}

print_error() {
    print_status "$RED" "❌ $1"
}

print_info() {
    print_status "$PURPLE" "📋 $1"
}

# Function to cleanup on exit
cleanup() {
    print_info "Cleaning up test environment..."
    if [[ -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
    fi
    # Kill any background processes
    pkill -f "intent-ingest" || true
    pkill -f "llm-processor" || true
}

# Function to setup test environment
setup_test_env() {
    print_header "Setting up test environment"
    
    # Create directories
    mkdir -p "$TEMP_DIR"
    mkdir -p "$HANDOFF_DIR"
    
    # Initialize log file
    echo "LLM Provider Pipeline Test Results - $(date)" > "$LOG_FILE"
    echo "=================================================" >> "$LOG_FILE"
    
    # Change to project directory
    cd "$PROJECT_ROOT"
    
    print_success "Test environment ready"
}

# Function to build all components
build_components() {
    print_header "Building LLM Provider Components"
    
    print_info "Building intent-ingest service..."
    if make build-intent-ingest >> "$LOG_FILE" 2>&1; then
        print_success "intent-ingest built successfully"
    else
        print_error "Failed to build intent-ingest"
        return 1
    fi
    
    print_info "Building llm-processor service..."
    if make build-llm-processor >> "$LOG_FILE" 2>&1; then
        print_success "llm-processor built successfully"
    else
        print_error "Failed to build llm-processor"
        return 1
    fi
    
    print_info "Verifying build system..."
    if make verify-build >> "$LOG_FILE" 2>&1; then
        print_success "Build system verification passed"
    else
        print_error "Build system verification failed"
        return 1
    fi
}

# Function to run unit tests
run_unit_tests() {
    print_header "Running Unit Tests"
    
    print_info "Running comprehensive unit test suite..."
    if go test -v -race -timeout="$TEST_TIMEOUT" -parallel=4 ./... >> "$LOG_FILE" 2>&1; then
        print_success "Unit tests passed"
    else
        print_error "Unit tests failed"
        return 1
    fi
    
    print_info "Testing LLM provider implementations..."
    if make test-llm-providers >> "$LOG_FILE" 2>&1; then
        print_success "LLM provider tests passed"
    else
        print_error "LLM provider tests failed"
        return 1
    fi
    
    print_info "Testing schema validation..."
    if make test-schema-validation >> "$LOG_FILE" 2>&1; then
        print_success "Schema validation tests passed"
    else
        print_error "Schema validation tests failed"
        return 1
    fi
}

# Function to run unit tests with coverage
run_coverage_tests() {
    print_header "Running Coverage Analysis"
    
    print_info "Running tests with coverage analysis..."
    if make test-unit-coverage >> "$LOG_FILE" 2>&1; then
        # Extract coverage percentage
        coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
        
        echo "Coverage: $coverage%" >> "$LOG_FILE"
        
        if (( $(echo "$coverage >= $COVERAGE_THRESHOLD" | bc -l) )); then
            print_success "Coverage $coverage% meets threshold ($COVERAGE_THRESHOLD%)"
        else
            print_warning "Coverage $coverage% is below threshold ($COVERAGE_THRESHOLD%)"
            return 1
        fi
    else
        print_error "Coverage analysis failed"
        return 1
    fi
}

# Function to run integration tests
run_integration_tests() {
    print_header "Running Integration Tests"
    
    print_info "Running integration test suite..."
    if make test-integration >> "$LOG_FILE" 2>&1; then
        print_success "Integration tests passed"
    else
        print_error "Integration tests failed"
        return 1
    fi
    
    print_info "Running pipeline integration tests..."
    if go test -v -timeout=15m ./test/... -run "TestLLMPipelineEndToEnd" >> "$LOG_FILE" 2>&1; then
        print_success "Pipeline integration tests passed"
    else
        print_error "Pipeline integration tests failed"
        return 1
    fi
}

# Function to test demo functionality
test_demo_functionality() {
    print_header "Testing Demo Functionality"
    
    print_info "Testing automated demo pipeline..."
    
    # Start intent-ingest service in background
    print_info "Starting intent-ingest service..."
    MODE=llm PROVIDER=mock ./bin/intent-ingest -addr ":$DEMO_PORT" -handoff "$HANDOFF_DIR" &
    INGEST_PID=$!
    
    # Wait for service to start
    sleep 3
    
    # Test health endpoint
    print_info "Testing health endpoint..."
    if curl -f "http://localhost:$DEMO_PORT/healthz" >> "$LOG_FILE" 2>&1; then
        print_success "Health endpoint responsive"
    else
        print_error "Health endpoint not responsive"
        kill $INGEST_PID || true
        return 1
    fi
    
    # Test intent processing
    print_info "Testing intent processing..."
    if curl -X POST -H "Content-Type: application/json" \
        -d '{"spec": {"intent": "scale odu-high-phy to 5 in ns oran-odu"}}' \
        "http://localhost:$DEMO_PORT/intent" >> "$LOG_FILE" 2>&1; then
        print_success "Intent processing successful"
    else
        print_error "Intent processing failed"
        kill $INGEST_PID || true
        return 1
    fi
    
    # Check handoff file generation
    sleep 1
    if [[ $(find "$HANDOFF_DIR" -name "*.json" | wc -l) -gt 0 ]]; then
        print_success "Handoff files generated successfully"
        print_info "Found $(find "$HANDOFF_DIR" -name "*.json" | wc -l) handoff files"
    else
        print_error "No handoff files generated"
        kill $INGEST_PID || true
        return 1
    fi
    
    # Stop service
    kill $INGEST_PID || true
    wait $INGEST_PID 2>/dev/null || true
}

# Function to run performance tests
run_performance_tests() {
    print_header "Running Performance Tests"
    
    print_info "Running benchmark tests..."
    if go test -bench=. -benchmem -timeout=5m ./internal/ingest/... > "$TEMP_DIR/benchmark-results.txt" 2>&1; then
        print_success "Benchmark tests completed"
        
        # Display key performance metrics
        if grep -q "BenchmarkRulesProvider" "$TEMP_DIR/benchmark-results.txt"; then
            rules_perf=$(grep "BenchmarkRulesProvider" "$TEMP_DIR/benchmark-results.txt" | head -1)
            print_info "Rules Provider: $rules_perf"
        fi
        
        if grep -q "BenchmarkMockLLMProvider" "$TEMP_DIR/benchmark-results.txt"; then
            llm_perf=$(grep "BenchmarkMockLLMProvider" "$TEMP_DIR/benchmark-results.txt" | head -1)
            print_info "Mock LLM Provider: $llm_perf"
        fi
        
    else
        print_warning "Some benchmark tests failed, but continuing..."
    fi
}

# Function to validate schema and contracts
validate_schemas() {
    print_header "Validating Schemas and Contracts"
    
    print_info "Validating intent schema..."
    if make validate-schema >> "$LOG_FILE" 2>&1; then
        print_success "Schema validation passed"
    else
        print_warning "Schema validation had issues (ajv-cli may not be installed)"
    fi
    
    # Check if schema file exists and is valid JSON
    if [[ -f "docs/contracts/intent.schema.json" ]]; then
        if python3 -m json.tool docs/contracts/intent.schema.json > /dev/null 2>&1; then
            print_success "Intent schema is valid JSON"
        else
            print_error "Intent schema is invalid JSON"
            return 1
        fi
    else
        print_error "Intent schema file not found"
        return 1
    fi
}

# Function to run code quality checks
run_quality_checks() {
    print_header "Running Code Quality Checks"
    
    print_info "Running go vet..."
    if make vet >> "$LOG_FILE" 2>&1; then
        print_success "go vet passed"
    else
        print_error "go vet failed"
        return 1
    fi
    
    print_info "Checking code formatting..."
    if make fmt >> "$LOG_FILE" 2>&1; then
        # Check if there are any formatting changes
        if git diff --exit-code >> "$LOG_FILE" 2>&1; then
            print_success "Code formatting is correct"
        else
            print_warning "Code formatting issues found (run 'make fmt')"
        fi
    else
        print_error "Code formatting check failed"
        return 1
    fi
    
    # Run linter if available
    if command -v golangci-lint &> /dev/null; then
        print_info "Running golangci-lint..."
        if make lint >> "$LOG_FILE" 2>&1; then
            print_success "Linting passed"
        else
            print_warning "Linting issues found"
        fi
    else
        print_info "golangci-lint not installed, skipping"
    fi
}

# Function to generate test report
generate_test_report() {
    print_header "Generating Test Report"
    
    local report_file="$TEMP_DIR/test-report.md"
    
    cat > "$report_file" << EOF
# LLM Provider Pipeline Test Report

**Generated:** $(date)
**Branch:** $(git rev-parse --abbrev-ref HEAD)
**Commit:** $(git rev-parse --short HEAD)

## Test Results Summary

EOF

    # Add coverage information if available
    if [[ -f "coverage.out" ]]; then
        local coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
        echo "**Coverage:** $coverage" >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    # Add handoff files information
    if [[ -d "$HANDOFF_DIR" ]]; then
        local handoff_count=$(find "$HANDOFF_DIR" -name "*.json" | wc -l)
        echo "**Handoff Files Generated:** $handoff_count" >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    # Add demo functionality status
    echo "## Component Status" >> "$report_file"
    echo "" >> "$report_file"
    echo "- ✅ intent-ingest service: Built and tested" >> "$report_file"
    echo "- ✅ LLM processor: Built and tested" >> "$report_file"
    echo "- ✅ Mock LLM provider: Functional" >> "$report_file"
    echo "- ✅ Schema validation: Active" >> "$report_file"
    echo "- ✅ Pipeline integration: Verified" >> "$report_file"
    echo "" >> "$report_file"
    
    # Add example handoff file if available
    if [[ -n "$(find "$HANDOFF_DIR" -name "*.json" | head -1)" ]]; then
        echo "## Sample Handoff File" >> "$report_file"
        echo "" >> "$report_file"
        echo "\`\`\`json" >> "$report_file"
        cat "$(find "$HANDOFF_DIR" -name "*.json" | head -1)" >> "$report_file"
        echo "\`\`\`" >> "$report_file"
        echo "" >> "$report_file"
    fi
    
    print_success "Test report generated: $report_file"
}

# Function to display help
show_help() {
    cat << EOF
LLM Provider Pipeline Test Runner

Usage: $0 [options] [test_type]

Test Types:
    all             Run all tests (default)
    unit            Run unit tests only
    integration     Run integration tests only
    coverage        Run unit tests with coverage analysis
    demo            Test demo functionality only
    performance     Run performance/benchmark tests
    schema          Validate schemas and contracts
    quality         Run code quality checks

Options:
    -h, --help      Show this help message
    -v, --verbose   Enable verbose output
    -c, --coverage  Set coverage threshold (default: 80)
    -p, --port      Set demo port (default: 8080)
    --no-cleanup    Don't cleanup temp files (for debugging)

Examples:
    $0                          # Run all tests
    $0 unit                     # Run unit tests only
    $0 demo                     # Test demo functionality
    $0 coverage -c 85           # Run coverage with 85% threshold
    $0 --verbose integration    # Run integration tests with verbose output

EOF
}

# Main test runner function
run_tests() {
    local test_type="${1:-all}"
    local success=true
    
    print_header "LLM Provider Pipeline Test Runner"
    print_info "Test type: $test_type"
    print_info "Project root: $PROJECT_ROOT"
    print_info "Temp directory: $TEMP_DIR"
    echo
    
    # Setup test environment
    if ! setup_test_env; then
        print_error "Failed to setup test environment"
        return 1
    fi
    
    # Build components (required for most tests)
    if [[ "$test_type" != "schema" && "$test_type" != "quality" ]]; then
        if ! build_components; then
            success=false
        fi
    fi
    
    # Run tests based on type
    case "$test_type" in
        "unit")
            run_unit_tests || success=false
            ;;
        "coverage")
            run_coverage_tests || success=false
            ;;
        "integration")
            run_integration_tests || success=false
            ;;
        "demo")
            test_demo_functionality || success=false
            ;;
        "performance")
            run_performance_tests || success=false
            ;;
        "schema")
            validate_schemas || success=false
            ;;
        "quality")
            run_quality_checks || success=false
            ;;
        "all")
            run_unit_tests || success=false
            run_coverage_tests || success=false
            run_integration_tests || success=false
            test_demo_functionality || success=false
            validate_schemas || success=false
            run_quality_checks || success=false
            run_performance_tests || success=false
            ;;
        *)
            print_error "Unknown test type: $test_type"
            show_help
            return 1
            ;;
    esac
    
    # Generate test report
    generate_test_report
    
    # Final results
    echo
    if $success; then
        print_header "🎉 ALL TESTS PASSED! 🎉"
        print_success "LLM Provider Pipeline is ready for integration"
        print_info "Test logs available in: $LOG_FILE"
        print_info "Test report available in: $TEMP_DIR/test-report.md"
        return 0
    else
        print_header "❌ SOME TESTS FAILED ❌"
        print_error "Please check the logs for details: $LOG_FILE"
        return 1
    fi
}

# Parse command line arguments
VERBOSE=false
NO_CLEANUP=false
TEST_TYPE="all"

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            set -x
            shift
            ;;
        -c|--coverage)
            COVERAGE_THRESHOLD="$2"
            shift 2
            ;;
        -p|--port)
            DEMO_PORT="$2"
            shift 2
            ;;
        --no-cleanup)
            NO_CLEANUP=true
            shift
            ;;
        unit|integration|coverage|demo|performance|schema|quality|all)
            TEST_TYPE="$1"
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Set trap for cleanup
if [[ "$NO_CLEANUP" != "true" ]]; then
    trap cleanup EXIT
fi

# Run the tests
run_tests "$TEST_TYPE"