#!/bin/bash
# =============================================================================
# Nephoran Flaky Test Detection and Quarantine System
# =============================================================================
# This script identifies, tracks, and manages flaky tests in the Nephoran project
# Automatically quarantines tests that fail intermittently and provides retry logic
# =============================================================================

set -euo pipefail

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FLAKY_TEST_DB="${PROJECT_ROOT}/.flaky-tests.json"
TEST_RESULTS_DIR="${PROJECT_ROOT}/test-results"
QUARANTINE_DIR="${PROJECT_ROOT}/test-quarantine"

# Default settings
DETECTION_RUNS="${DETECTION_RUNS:-5}"
FAILURE_THRESHOLD="${FAILURE_THRESHOLD:-2}"  # Failures out of DETECTION_RUNS to mark as flaky
RETRY_ATTEMPTS="${RETRY_ATTEMPTS:-3}"
RETRY_DELAY="${RETRY_DELAY:-5}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Initialize flaky test database
init_flaky_db() {
    if [[ ! -f "$FLAKY_TEST_DB" ]]; then
        log_info "Initializing flaky test database..."
        cat > "$FLAKY_TEST_DB" << 'EOF'
{
  "version": "1.0",
  "last_updated": "",
  "flaky_tests": {},
  "quarantined_tests": [],
  "statistics": {
    "total_runs": 0,
    "total_flaky": 0,
    "total_quarantined": 0
  }
}
EOF
    fi
}

# Update timestamp in flaky test database
update_db_timestamp() {
    local temp_file="${FLAKY_TEST_DB}.tmp"
    jq --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" '.last_updated = $timestamp' "$FLAKY_TEST_DB" > "$temp_file"
    mv "$temp_file" "$FLAKY_TEST_DB"
}

# Add or update flaky test in database
add_flaky_test() {
    local test_name="$1"
    local package_path="$2"
    local failure_rate="$3"
    local error_patterns="$4"
    
    log_warning "Adding flaky test: $test_name (failure rate: $failure_rate)"
    
    local temp_file="${FLAKY_TEST_DB}.tmp"
    jq --arg test "$test_name" \
       --arg pkg "$package_path" \
       --arg rate "$failure_rate" \
       --arg patterns "$error_patterns" \
       --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
       '.flaky_tests[$test] = {
           "package": $pkg,
           "failure_rate": $rate,
           "error_patterns": ($patterns | split("|")),
           "first_detected": (.flaky_tests[$test].first_detected // $timestamp),
           "last_seen": $timestamp,
           "retry_count": ((.flaky_tests[$test].retry_count // 0) + 1)
       }' "$FLAKY_TEST_DB" > "$temp_file"
    mv "$temp_file" "$FLAKY_TEST_DB"
}

# Quarantine a flaky test
quarantine_test() {
    local test_name="$1"
    local reason="$2"
    
    log_warning "Quarantining test: $test_name"
    log_info "Reason: $reason"
    
    local temp_file="${FLAKY_TEST_DB}.tmp"
    jq --arg test "$test_name" \
       --arg reason "$reason" \
       --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
       '.quarantined_tests |= . + [{
           "test_name": $test,
           "reason": $reason,
           "quarantined_at": $timestamp
       }] | .quarantined_tests |= unique_by(.test_name)' "$FLAKY_TEST_DB" > "$temp_file"
    mv "$temp_file" "$FLAKY_TEST_DB"
    
    # Create quarantine skip file
    mkdir -p "$QUARANTINE_DIR"
    echo "// QUARANTINED: $reason" > "$QUARANTINE_DIR/${test_name}.skip"
    echo "// Quarantined at: $(date -u +%Y-%m-%dT%H:%M:%SZ)" >> "$QUARANTINE_DIR/${test_name}.skip"
}

# Check if test is quarantined
is_quarantined() {
    local test_name="$1"
    jq -r --arg test "$test_name" '.quarantined_tests[] | select(.test_name == $test) | .test_name' "$FLAKY_TEST_DB" 2>/dev/null | grep -q "$test_name"
}

# Get flaky test patterns for skipping
get_skip_patterns() {
    jq -r '.quarantined_tests[] | .test_name' "$FLAKY_TEST_DB" 2>/dev/null | tr '\n' '|' | sed 's/|$//' || echo ""
}

# Run test multiple times to detect flakiness
detect_flaky_test() {
    local test_pattern="$1"
    local package_path="$2"
    local runs="${3:-$DETECTION_RUNS}"
    
    log_info "Detecting flakiness in: $test_pattern (package: $package_path)"
    log_info "Running $runs times to detect flakiness..."
    
    local success_count=0
    local failure_count=0
    local error_patterns=()
    
    mkdir -p "$TEST_RESULTS_DIR/flaky-detection"
    
    for ((i=1; i<=runs; i++)); do
        log_info "Run $i/$runs..."
        
        local log_file="$TEST_RESULTS_DIR/flaky-detection/${test_pattern//\//_}-run-$i.log"
        
        if timeout 2m go test -v -timeout=1m -parallel=1 -run="$test_pattern" "$package_path" 2>&1 | tee "$log_file"; then
            ((success_count++))
            echo -n "✅ "
        else
            ((failure_count++))
            echo -n "❌ "
            
            # Extract error patterns
            if [[ -f "$log_file" ]]; then
                local errors
                errors=$(grep -E "(panic|race detected|context deadline|connection refused|timeout)" "$log_file" | head -3 | sed 's/^[[:space:]]*//' || echo "")
                if [[ -n "$errors" ]]; then
                    error_patterns+=("$errors")
                fi
            fi
        fi
        
        # Small delay between runs
        sleep 1
    done
    
    echo ""
    
    local failure_rate=$((failure_count * 100 / runs))
    log_info "Results: $success_count successes, $failure_count failures ($failure_rate% failure rate)"
    
    # Determine if test is flaky
    if [[ $failure_count -ge $FAILURE_THRESHOLD && $failure_count -lt $runs ]]; then
        local unique_errors
        unique_errors=$(printf '%s\n' "${error_patterns[@]}" | sort -u | tr '\n' '|' | sed 's/|$//')
        
        add_flaky_test "$test_pattern" "$package_path" "$failure_rate%" "$unique_errors"
        
        # Quarantine if failure rate is high
        if [[ $failure_rate -ge 60 ]]; then
            quarantine_test "$test_pattern" "High failure rate: $failure_rate%"
        fi
        
        return 1  # Flaky detected
    elif [[ $failure_count -eq $runs ]]; then
        log_error "Test consistently fails - not flaky, but broken"
        return 2  # Consistently broken
    else
        log_success "Test is stable"
        return 0  # Stable
    fi
}

# Run all tests with flaky detection
detect_all_flaky_tests() {
    log_info "Starting comprehensive flaky test detection..."
    
    init_flaky_db
    
    # Get all test functions
    local test_functions
    test_functions=$(grep -r "^func Test" . --include="*_test.go" | grep -v ".git" | sed 's/.*func \(Test[^(]*\).*/\1/' | sort -u)
    
    local total_tests
    total_tests=$(echo "$test_functions" | wc -l)
    local flaky_count=0
    local broken_count=0
    local current=0
    
    log_info "Found $total_tests test functions to analyze"
    
    echo "$test_functions" | while read -r test_func; do
        ((current++))
        
        if [[ -z "$test_func" ]]; then
            continue
        fi
        
        log_info "[$current/$total_tests] Analyzing: $test_func"
        
        # Find package for this test
        local package_path
        package_path=$(grep -r "func $test_func" . --include="*_test.go" | head -1 | cut -d: -f1 | xargs dirname)
        
        if [[ -z "$package_path" ]]; then
            log_warning "Could not find package for $test_func"
            continue
        fi
        
        # Skip if already quarantined
        if is_quarantined "$test_func"; then
            log_info "Skipping quarantined test: $test_func"
            continue
        fi
        
        # Detect flakiness
        local result=0
        detect_flaky_test "$test_func" "$package_path" 3 || result=$?
        
        case $result in
            1) ((flaky_count++)) ;;
            2) ((broken_count++)) ;;
        esac
    done
    
    log_info "Flaky test detection completed"
    log_info "Flaky tests detected: $flaky_count"
    log_info "Broken tests detected: $broken_count"
    
    update_db_timestamp
}

# Run tests with smart retry for flaky tests
run_with_flaky_handling() {
    local test_pattern="${1:-./...}"
    local timeout="${2:-8m}"
    local parallel="${3:-4}"
    
    log_info "Running tests with flaky test handling..."
    log_info "Pattern: $test_pattern, Timeout: $timeout, Parallel: $parallel"
    
    init_flaky_db
    
    # Get skip patterns from quarantined tests
    local skip_patterns
    skip_patterns=$(get_skip_patterns)
    
    # Build test command
    local test_cmd="go test -v -timeout=$timeout -parallel=$parallel -shuffle=on"
    
    if [[ -n "$skip_patterns" ]]; then
        test_cmd="$test_cmd -skip='$skip_patterns'"
        log_info "Skipping quarantined tests: $skip_patterns"
    fi
    
    test_cmd="$test_cmd $test_pattern"
    
    mkdir -p "$TEST_RESULTS_DIR"
    local log_file="$TEST_RESULTS_DIR/test-with-flaky-handling.log"
    
    log_info "Executing: $test_cmd"
    
    if timeout $((${timeout%m} * 60 + 30)) $test_cmd 2>&1 | tee "$log_file"; then
        log_success "Tests completed successfully"
        return 0
    else
        local exit_code=$?
        log_warning "Tests failed, analyzing for flaky test patterns..."
        
        # Check for known flaky test failures
        local flaky_failures
        flaky_failures=$(jq -r '.flaky_tests | keys[]' "$FLAKY_TEST_DB" 2>/dev/null || echo "")
        
        if [[ -n "$flaky_failures" ]]; then
            log_info "Retrying known flaky tests individually..."
            
            local retry_success=true
            echo "$flaky_failures" | while read -r test_name; do
                if grep -q "$test_name" "$log_file"; then
                    log_warning "Retrying flaky test: $test_name"
                    
                    local retry_count=0
                    local test_passed=false
                    
                    while [[ $retry_count -lt $RETRY_ATTEMPTS ]]; do
                        ((retry_count++))
                        log_info "Retry attempt $retry_count/$RETRY_ATTEMPTS for $test_name"
                        
                        if timeout 2m go test -v -timeout=1m -parallel=1 -run="$test_name" $test_pattern; then
                            log_success "Retry successful for $test_name"
                            test_passed=true
                            break
                        else
                            log_warning "Retry $retry_count failed for $test_name"
                            sleep $RETRY_DELAY
                        fi
                    done
                    
                    if [[ "$test_passed" == false ]]; then
                        log_error "All retries failed for $test_name - consider quarantining"
                        quarantine_test "$test_name" "Failed all $RETRY_ATTEMPTS retries"
                        retry_success=false
                    fi
                fi
            done
            
            if [[ "$retry_success" == true ]]; then
                log_success "All flaky tests passed on retry"
                return 0
            fi
        fi
        
        return $exit_code
    fi
}

# Generate flaky test report
generate_flaky_report() {
    log_info "Generating flaky test report..."
    
    if [[ ! -f "$FLAKY_TEST_DB" ]]; then
        log_warning "No flaky test database found"
        return 1
    fi
    
    local report_file="$TEST_RESULTS_DIR/flaky-test-report.md"
    mkdir -p "$TEST_RESULTS_DIR"
    
    cat > "$report_file" << 'EOF'
# Nephoran Flaky Test Report

## Summary
EOF
    
    local total_flaky
    local total_quarantined
    local last_updated
    
    total_flaky=$(jq -r '.flaky_tests | keys | length' "$FLAKY_TEST_DB")
    total_quarantined=$(jq -r '.quarantined_tests | length' "$FLAKY_TEST_DB")
    last_updated=$(jq -r '.last_updated' "$FLAKY_TEST_DB")
    
    cat >> "$report_file" << EOF
- **Total Flaky Tests**: $total_flaky
- **Total Quarantined Tests**: $total_quarantined  
- **Last Updated**: $last_updated

## Flaky Tests
EOF
    
    # List flaky tests
    jq -r '.flaky_tests | to_entries[] | "- **\(.key)** (Package: \(.value.package), Failure Rate: \(.value.failure_rate))"' "$FLAKY_TEST_DB" >> "$report_file"
    
    echo "" >> "$report_file"
    echo "## Quarantined Tests" >> "$report_file"
    
    # List quarantined tests
    jq -r '.quarantined_tests[] | "- **\(.test_name)**: \(.reason) (Quarantined: \(.quarantined_at))"' "$FLAKY_TEST_DB" >> "$report_file"
    
    cat >> "$report_file" << 'EOF'

## Recommendations

1. **High-Priority Fixes**: Focus on quarantined tests first
2. **Investigation**: Analyze error patterns for flaky tests
3. **Environment**: Consider test environment stability issues
4. **Parallelization**: Some flaky tests may need reduced parallelism

## Actions

- Review and fix quarantined tests to restore test coverage
- Monitor failure patterns for newly detected flaky tests  
- Consider test isolation improvements for high-failure-rate tests
EOF
    
    log_success "Flaky test report generated: $report_file"
    
    echo ""
    echo "📊 Flaky Test Summary:"
    echo "  Flaky tests: $total_flaky"
    echo "  Quarantined: $total_quarantined"
    echo "  Report: $report_file"
}

# Clean quarantine (remove false positives)
clean_quarantine() {
    local test_name="${1:-}"
    
    if [[ -z "$test_name" ]]; then
        log_info "Listing quarantined tests for cleanup review..."
        jq -r '.quarantined_tests[] | "\(.test_name): \(.reason)"' "$FLAKY_TEST_DB"
        return 0
    fi
    
    log_info "Removing $test_name from quarantine..."
    
    local temp_file="${FLAKY_TEST_DB}.tmp"
    jq --arg test "$test_name" 'del(.quarantined_tests[] | select(.test_name == $test))' "$FLAKY_TEST_DB" > "$temp_file"
    mv "$temp_file" "$FLAKY_TEST_DB"
    
    # Remove skip file
    rm -f "$QUARANTINE_DIR/${test_name}.skip"
    
    log_success "$test_name removed from quarantine"
}

# Main command handler
case "${1:-help}" in
    "detect")
        detect_all_flaky_tests
        ;;
    "run")
        run_with_flaky_handling "${2:-./...}" "${3:-8m}" "${4:-4}"
        ;;
    "report")
        generate_flaky_report
        ;;
    "quarantine")
        if [[ $# -ge 3 ]]; then
            quarantine_test "$2" "$3"
        else
            log_error "Usage: $0 quarantine <test_name> <reason>"
            exit 1
        fi
        ;;
    "clean")
        clean_quarantine "$2"
        ;;
    "status")
        init_flaky_db
        echo "Flaky Test Database Status:"
        jq -r '"Total flaky: " + (.flaky_tests | keys | length | tostring) + 
               ", Quarantined: " + (.quarantined_tests | length | tostring) + 
               ", Last updated: " + .last_updated' "$FLAKY_TEST_DB"
        ;;
    "help"|*)
        echo "Nephoran Flaky Test Detection and Quarantine System"
        echo ""
        echo "Usage: $0 <command> [options]"
        echo ""
        echo "Commands:"
        echo "  detect                    - Run comprehensive flaky test detection"
        echo "  run [pattern] [timeout] [parallel] - Run tests with flaky handling"
        echo "  report                    - Generate flaky test report"
        echo "  quarantine <test> <reason> - Manually quarantine a test"
        echo "  clean [test]              - Remove test from quarantine (or list all)"
        echo "  status                    - Show flaky test database status"
        echo "  help                      - Show this help"
        echo ""
        echo "Examples:"
        echo "  $0 detect                                    # Detect all flaky tests"
        echo "  $0 run ./internal/loop/... 5m 2             # Run with flaky handling"
        echo "  $0 quarantine TestFlaky \"Intermittent failure\"  # Manual quarantine"
        echo "  $0 clean TestFlaky                          # Remove from quarantine"
        ;;
esac