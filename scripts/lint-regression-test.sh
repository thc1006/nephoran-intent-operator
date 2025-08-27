#!/bin/bash
# Regression testing script to ensure lint fixes don't break existing functionality
# Usage: ./lint-regression-test.sh [test-suite] [options]

set -euo pipefail

# Configuration
TEST_SUITE="${1:-all}"
TEMP_DIR="/tmp/lint-regression-$$"
PARALLEL_JOBS="${PARALLEL_JOBS:-4}"
VERBOSE="${VERBOSE:-false}"
FAIL_FAST="${FAIL_FAST:-false}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
GRAY='\033[0;37m'
BOLD='\033[1m'
NC='\033[0m'

# Test configurations
declare -A TEST_SUITES=(
    ["unit"]="./pkg/... ./internal/... ./cmd/..."
    ["integration"]="./tests/..."
    ["controllers"]="./pkg/controllers/..."
    ["core"]="./pkg/audit/... ./pkg/auth/... ./pkg/config/..."
    ["cmd"]="./cmd/..."
    ["all"]="./..."
)

usage() {
    echo "Usage: $0 [test-suite] [options]"
    echo ""
    echo "Test suites:"
    for suite in "${!TEST_SUITES[@]}"; do
        echo "  $suite: ${TEST_SUITES[$suite]}"
    done
    echo ""
    echo "Environment variables:"
    echo "  PARALLEL_JOBS=N    Number of parallel test jobs (default: 4)"
    echo "  VERBOSE=true       Enable verbose output"
    echo "  FAIL_FAST=true     Stop on first failure"
    exit 1
}

log() {
    echo -e "${GRAY}[$(date +'%H:%M:%S')] $1${NC}" >&2
}

log_info() {
    echo -e "${CYAN}[INFO] $1${NC}" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}" >&2
}

log_error() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING] $1${NC}" >&2
}

cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

# Validate test suite
if [[ -z "${TEST_SUITES[$TEST_SUITE]:-}" ]]; then
    log_error "Invalid test suite: $TEST_SUITE"
    usage
fi

TEST_PATTERNS="${TEST_SUITES[$TEST_SUITE]}"
mkdir -p "$TEMP_DIR"

log_info "Lint Regression Test Suite"
log_info "Test suite: $TEST_SUITE"
log_info "Patterns: $TEST_PATTERNS"
log_info "Parallel jobs: $PARALLEL_JOBS"
log_info "Temp dir: $TEMP_DIR"
echo ""

# Test results tracking
RESULTS_FILE="$TEMP_DIR/results.json"
echo '{"tests": [], "summary": {}}' > "$RESULTS_FILE"

add_test_result() {
    local test_name="$1"
    local success="$2"
    local duration="$3"
    local output="$4"
    
    jq --arg name "$test_name" \
       --argjson success "$success" \
       --argjson duration "$duration" \
       --arg output "$output" \
       '.tests += [{name: $name, success: $success, duration: $duration, output: $output}]' \
       "$RESULTS_FILE" > "$TEMP_DIR/results_tmp.json" && mv "$TEMP_DIR/results_tmp.json" "$RESULTS_FILE"
}

run_test_phase() {
    local phase_name="$1"
    local command=("${@:2}")
    
    log_info "Running $phase_name..."
    local start_time
    start_time=$(date +%s.%N)
    
    if [[ "$VERBOSE" == "true" ]]; then
        log "Command: ${command[*]}"
    fi
    
    local output_file="$TEMP_DIR/${phase_name}_output.log"
    local success=true
    
    if "${command[@]}" > "$output_file" 2>&1; then
        local end_time
        end_time=$(date +%s.%N)
        local duration
        duration=$(echo "$end_time - $start_time" | bc -l)
        
        log_success "$phase_name completed (${duration}s)"
        add_test_result "$phase_name" true "$duration" "$(cat "$output_file")"
    else
        local exit_code=$?
        local end_time
        end_time=$(date +%s.%N)
        local duration
        duration=$(echo "$end_time - $start_time" | bc -l)
        
        log_error "$phase_name failed with exit code $exit_code (${duration}s)"
        
        if [[ "$VERBOSE" == "true" ]] || [[ "$FAIL_FAST" == "true" ]]; then
            echo "Output:"
            cat "$output_file"
        fi
        
        add_test_result "$phase_name" false "$duration" "$(cat "$output_file")"
        success=false
        
        if [[ "$FAIL_FAST" == "true" ]]; then
            log_error "Fail fast enabled, stopping regression test"
            exit 1
        fi
    fi
    
    return $([ "$success" = true ])
}

# Phase 1: Dependency check
log_info "Phase 1: Dependency Verification"
run_test_phase "go_mod_download" go mod download
run_test_phase "go_mod_tidy_check" bash -c 'go mod tidy && git diff --exit-code go.mod go.sum'

# Phase 2: Build verification
log_info "Phase 2: Build Verification"
for pattern in $TEST_PATTERNS; do
    # Skip if pattern doesn't exist
    if ! ls $pattern >/dev/null 2>&1; then
        log_warning "Skipping non-existent pattern: $pattern"
        continue
    fi
    
    # Clean build
    run_test_phase "build_${pattern//\//_}" go build -v "$pattern"
done

# Phase 3: Static analysis
log_info "Phase 3: Static Analysis"
run_test_phase "go_vet" go vet $TEST_PATTERNS
run_test_phase "go_fmt_check" bash -c "gofmt -l $TEST_PATTERNS | tee $TEMP_DIR/fmt_issues.txt && [ ! -s $TEMP_DIR/fmt_issues.txt ]"

# Phase 4: Test execution
log_info "Phase 4: Test Execution"
for pattern in $TEST_PATTERNS; do
    if ! ls $pattern >/dev/null 2>&1; then
        continue
    fi
    
    # Unit tests
    run_test_phase "unit_tests_${pattern//\//_}" go test -v -race -timeout=10m -p="$PARALLEL_JOBS" "$pattern"
    
    # Test with coverage
    run_test_phase "coverage_${pattern//\//_}" go test -v -race -coverprofile="$TEMP_DIR/coverage_${pattern//\//_}.out" "$pattern"
done

# Phase 5: Lint verification (if golangci-lint is available)
if command -v golangci-lint >/dev/null 2>&1; then
    log_info "Phase 5: Lint Verification"
    run_test_phase "golangci_lint" golangci-lint run --timeout=10m $TEST_PATTERNS
else
    log_warning "golangci-lint not available, skipping lint verification"
    add_test_result "golangci_lint" true 0 "skipped - tool not available"
fi

# Phase 6: Integration tests (if they exist)
if [[ -d "tests" ]] && [[ "$TEST_SUITE" == "all" || "$TEST_SUITE" == "integration" ]]; then
    log_info "Phase 6: Integration Tests"
    if [[ -f "tests/integration_test.go" ]]; then
        run_test_phase "integration_tests" go test -v -timeout=15m ./tests/...
    else
        log_warning "No integration tests found"
        add_test_result "integration_tests" true 0 "skipped - no tests found"
    fi
fi

# Generate summary
log_info "Generating test summary..."

TOTAL_TESTS=$(jq '.tests | length' "$RESULTS_FILE")
PASSED_TESTS=$(jq '[.tests[] | select(.success == true)] | length' "$RESULTS_FILE")
FAILED_TESTS=$(jq '[.tests[] | select(.success == false)] | length' "$RESULTS_FILE")
TOTAL_DURATION=$(jq '[.tests[].duration] | add' "$RESULTS_FILE")

# Update results file with summary
jq --argjson total "$TOTAL_TESTS" \
   --argjson passed "$PASSED_TESTS" \
   --argjson failed "$FAILED_TESTS" \
   --argjson duration "$TOTAL_DURATION" \
   '.summary = {total: $total, passed: $passed, failed: $failed, duration: $duration}' \
   "$RESULTS_FILE" > "$TEMP_DIR/final_results.json" && mv "$TEMP_DIR/final_results.json" "$RESULTS_FILE"

echo ""
log_info "Regression Test Summary"
echo "========================="
printf "Total tests: %d\n" "$TOTAL_TESTS"
printf "Passed: %d\n" "$PASSED_TESTS"
printf "Failed: %d\n" "$FAILED_TESTS"
printf "Total duration: %.2f seconds\n" "$TOTAL_DURATION"

if [[ "$FAILED_TESTS" -gt 0 ]]; then
    echo ""
    log_error "Failed tests:"
    jq -r '.tests[] | select(.success == false) | "  - " + .name' "$RESULTS_FILE"
fi

# Save results to project directory
RESULTS_OUTPUT="./test-results/lint-regression-$(date +%Y%m%d_%H%M%S).json"
mkdir -p "$(dirname "$RESULTS_OUTPUT")"
cp "$RESULTS_FILE" "$RESULTS_OUTPUT"
log_info "Results saved to: $RESULTS_OUTPUT"

# Exit with appropriate code
if [[ "$FAILED_TESTS" -eq 0 ]]; then
    log_success "All regression tests passed!"
    exit 0
else
    log_error "Some regression tests failed!"
    exit 1
fi