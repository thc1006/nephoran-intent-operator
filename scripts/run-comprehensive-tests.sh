#!/bin/bash

# Comprehensive Test Runner for Nephoran Intent Operator
# This script runs the complete Ginkgo v2 test suite with coverage analysis

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEST_RESULTS_DIR="${PROJECT_ROOT}/test-results"
COVERAGE_DIR="${TEST_RESULTS_DIR}/coverage"
COVERAGE_FILE="${COVERAGE_DIR}/coverage.out"
COVERAGE_HTML="${COVERAGE_DIR}/coverage.html"
JUNIT_FILE="${TEST_RESULTS_DIR}/junit.xml"

# Test configuration
UNIT_TEST_TIMEOUT="5m"
INTEGRATION_TEST_TIMEOUT="10m"
PERFORMANCE_TEST_TIMEOUT="30m"
PARALLEL_NODES=4

# Functions
print_banner() {
    echo -e "${BLUE}"
    echo "========================================"
    echo "  Nephoran Intent Operator Test Suite  "
    echo "========================================"
    echo -e "${NC}"
}

print_section() {
    echo -e "${YELLOW}== $1 ==${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

setup_test_environment() {
    print_section "Setting up test environment"
    
    # Create test results directories
    mkdir -p "${TEST_RESULTS_DIR}"
    mkdir -p "${COVERAGE_DIR}"
    
    # Clean previous results
    rm -f "${COVERAGE_FILE}" "${COVERAGE_HTML}" "${JUNIT_FILE}"
    
    # Ensure test dependencies are available
    cd "${PROJECT_ROOT}"
    go mod tidy
    go mod download
    
    # Install test tools if not present
    if ! command -v ginkgo &> /dev/null; then
        echo "Installing Ginkgo CLI..."
        go install github.com/onsi/ginkgo/v2/ginkgo@latest
    fi
    
    print_success "Test environment ready"
}

run_unit_tests() {
    print_section "Running Unit Tests"
    
    cd "${PROJECT_ROOT}"
    
    echo "Executing unit tests with coverage..."
    ginkgo run \
        --timeout="${UNIT_TEST_TIMEOUT}" \
        --parallel-node="${PARALLEL_NODES}" \
        --race \
        --trace \
        --cover \
        --coverprofile="${COVERAGE_FILE}" \
        --junit-report="${JUNIT_FILE}" \
        --output-interceptor-mode=none \
        ./tests/unit/...
    
    if [ $? -eq 0 ]; then
        print_success "Unit tests completed successfully"
    else
        print_error "Unit tests failed"
        return 1
    fi
}

run_integration_tests() {
    print_section "Running Integration Tests"
    
    cd "${PROJECT_ROOT}"
    
    echo "Executing integration tests..."
    ginkgo run \
        --timeout="${INTEGRATION_TEST_TIMEOUT}" \
        --parallel-node="2" \
        --race \
        --trace \
        --output-interceptor-mode=none \
        ./tests/integration/...
    
    if [ $? -eq 0 ]; then
        print_success "Integration tests completed successfully"
    else
        print_error "Integration tests failed"
        return 1
    fi
}

run_performance_tests() {
    print_section "Running Performance Tests"
    
    cd "${PROJECT_ROOT}"
    
    echo "Executing performance tests..."
    ginkgo run \
        --timeout="${PERFORMANCE_TEST_TIMEOUT}" \
        --slow-spec-threshold=30s \
        --trace \
        --output-interceptor-mode=none \
        ./tests/performance/...
    
    if [ $? -eq 0 ]; then
        print_success "Performance tests completed successfully"
    else
        print_error "Performance tests failed"
        return 1
    fi
}

generate_coverage_report() {
    print_section "Generating Coverage Report"
    
    if [ -f "${COVERAGE_FILE}" ]; then
        # Generate HTML coverage report
        go tool cover -html="${COVERAGE_FILE}" -o "${COVERAGE_HTML}"
        
        # Calculate coverage percentage
        COVERAGE_PERCENT=$(go tool cover -func="${COVERAGE_FILE}" | grep total: | awk '{print $3}')
        
        echo "Coverage report generated: ${COVERAGE_HTML}"
        echo "Total coverage: ${COVERAGE_PERCENT}"
        
        # Check if coverage meets minimum threshold (90%)
        COVERAGE_VALUE=$(echo "${COVERAGE_PERCENT}" | sed 's/%//')
        if (( $(echo "${COVERAGE_VALUE} >= 90.0" | bc -l) )); then
            print_success "Coverage meets 90%+ requirement: ${COVERAGE_PERCENT}"
        else
            print_error "Coverage below 90% requirement: ${COVERAGE_PERCENT}"
            return 1
        fi
    else
        print_error "Coverage file not found"
        return 1
    fi
}

run_linting() {
    print_section "Running Code Quality Checks"
    
    cd "${PROJECT_ROOT}"
    
    # Run go vet
    echo "Running go vet..."
    go vet ./...
    if [ $? -eq 0 ]; then
        print_success "go vet passed"
    else
        print_error "go vet failed"
        return 1
    fi
    
    # Run golint if available
    if command -v golint &> /dev/null; then
        echo "Running golint..."
        golint ./... | grep -v "should have comment" || true
    fi
    
    # Run gofmt check
    echo "Checking code formatting..."
    UNFORMATTED=$(gofmt -l .)
    if [ -n "${UNFORMATTED}" ]; then
        print_error "The following files are not properly formatted:"
        echo "${UNFORMATTED}"
        return 1
    else
        print_success "All files are properly formatted"
    fi
}

validate_test_structure() {
    print_section "Validating Test Structure"
    
    # Check for required test files
    REQUIRED_FILES=(
        "tests/unit/controllers/networkintent_controller_test.go"
        "tests/unit/controllers/e2nodeset_controller_test.go"
        "tests/integration/controllers/intent_pipeline_test.go"
        "tests/integration/controllers/e2nodeset_scaling_test.go"
        "tests/performance/controllers/load_test.go"
        "tests/utils/test_fixtures.go"
        "tests/utils/mocks.go"
    )
    
    for file in "${REQUIRED_FILES[@]}"; do
        if [ -f "${PROJECT_ROOT}/${file}" ]; then
            print_success "Found: ${file}"
        else
            print_error "Missing: ${file}"
            return 1
        fi
    done
    
    # Validate test naming conventions
    echo "Validating test naming conventions..."
    find "${PROJECT_ROOT}/tests" -name "*_test.go" -exec grep -L "func Test\|var _ = Describe" {} \; | while read -r file; do
        if [ -n "$file" ]; then
            print_error "Test file missing Test function or Describe block: $file"
        fi
    done
}

generate_test_summary() {
    print_section "Test Summary"
    
    if [ -f "${JUNIT_FILE}" ]; then
        # Parse JUnit XML for summary (simplified)
        echo "Test results saved to: ${JUNIT_FILE}"
    fi
    
    if [ -f "${COVERAGE_HTML}" ]; then
        echo "Coverage report: ${COVERAGE_HTML}"
    fi
    
    echo "Test results directory: ${TEST_RESULTS_DIR}"
}

cleanup() {
    print_section "Cleanup"
    
    # Kill any remaining test processes
    pkill -f "ginkgo" || true
    pkill -f "envtest" || true
    
    # Clean up temporary files
    find "${PROJECT_ROOT}" -name "*.test" -delete 2>/dev/null || true
    find "${PROJECT_ROOT}" -name "*.tmp" -delete 2>/dev/null || true
}

# Main execution
main() {
    print_banner
    
    # Parse command line arguments
    TEST_TYPE="all"
    SKIP_PERFORMANCE=false
    VERBOSE=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -t|--type)
                TEST_TYPE="$2"
                shift 2
                ;;
            --skip-performance)
                SKIP_PERFORMANCE=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                echo "Usage: $0 [options]"
                echo "Options:"
                echo "  -t, --type TYPE          Test type (unit|integration|performance|all)"
                echo "  --skip-performance       Skip performance tests"
                echo "  -v, --verbose            Verbose output"
                echo "  -h, --help               Show this help"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Trap cleanup on exit
    trap cleanup EXIT
    
    # Execute test phases
    setup_test_environment || exit 1
    validate_test_structure || exit 1
    run_linting || exit 1
    
    case "${TEST_TYPE}" in
        "unit")
            run_unit_tests || exit 1
            generate_coverage_report || exit 1
            ;;
        "integration")
            run_integration_tests || exit 1
            ;;
        "performance")
            run_performance_tests || exit 1
            ;;
        "all")
            run_unit_tests || exit 1
            generate_coverage_report || exit 1
            run_integration_tests || exit 1
            
            if [ "${SKIP_PERFORMANCE}" != "true" ]; then
                run_performance_tests || exit 1
            fi
            ;;
        *)
            print_error "Invalid test type: ${TEST_TYPE}"
            exit 1
            ;;
    esac
    
    generate_test_summary
    print_success "All tests completed successfully!"
}

# Check if script is being run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi