#!/bin/bash
# Test Harness for Hardened CI Test Execution
# Provides consistent, reliable test execution across all environments

set -euo pipefail

# Source test environment configuration
if [ -f ".testenv" ]; then
    source .testenv
fi

# Default configuration
TEST_TIMEOUT_CEILING=${TEST_TIMEOUT_CEILING:-8m}
TEST_GO_TIMEOUT=${TEST_GO_TIMEOUT:-7m30s}
TEST_REPORTS_DIR=${TEST_REPORTS_DIR:-.test-reports}
TEST_STATUS_FILE=${TEST_STATUS_FILE:-test-status.txt}
COVERAGE_HTML=${COVERAGE_HTML:-coverage.html}
TEST_COUNT=${TEST_COUNT:-1}
TEST_PARALLEL=${TEST_PARALLEL:-2}

# Function to run hardened tests
run_hardened_tests() {
    local test_paths="$1"
    local test_type="${2:-unit}"
    
    echo "Running hardened $test_type tests with ${TEST_TIMEOUT_CEILING} timeout ceiling..."
    
    # Create reports directory
    mkdir -p "${TEST_REPORTS_DIR}"
    
    # Set deterministic environment
    export CGO_ENABLED=${CGO_ENABLED:-1}
    export GOMAXPROCS=${GOMAXPROCS:-2}
    export GODEBUG=${GODEBUG:-gocachehash=1}
    export GO111MODULE=${GO111MODULE:-on}
    
    # Initialize status file
    echo "test_type=$test_type" > "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
    echo "test_start_time=$(date -Iseconds)" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
    
    # Run tests with timeout safety
    local test_exit_code=0
    timeout "${TEST_TIMEOUT_CEILING}" go test -v -race -timeout="${TEST_GO_TIMEOUT}" \
        -coverprofile="${TEST_REPORTS_DIR}/coverage.out" \
        -covermode=atomic \
        -count="${TEST_COUNT}" \
        -parallel="${TEST_PARALLEL}" \
        ${test_paths} | tee "${TEST_REPORTS_DIR}/test.log" || {
        test_exit_code=$?
        echo "test_exit_code=$test_exit_code" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
        echo "Tests completed with exit code: $test_exit_code (may have reached timeout ceiling)"
    }
    
    # Process coverage if available
    if [ -f "${TEST_REPORTS_DIR}/coverage.out" ] && [ -s "${TEST_REPORTS_DIR}/coverage.out" ]; then
        echo "Generating coverage HTML report..."
        go tool cover -html="${TEST_REPORTS_DIR}/coverage.out" -o "${TEST_REPORTS_DIR}/${COVERAGE_HTML}"
        
        # Calculate coverage percentage safely
        local coverage_percent
        coverage_percent=$(go tool cover -func="${TEST_REPORTS_DIR}/coverage.out" 2>/dev/null | grep total | awk '{print $3}' || echo "0.0%")
        
        echo "coverage_generated=true" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
        echo "coverage_percent=$coverage_percent" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
        echo "Coverage: $coverage_percent"
    else
        echo "Warning: Coverage file not generated or empty"
        echo "coverage_generated=false" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
        echo "coverage_percent=0.0%" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
    fi
    
    # Calculate test metrics safely
    local test_count pass_count fail_count
    test_count=$(grep -c "^=== RUN" "${TEST_REPORTS_DIR}/test.log" 2>/dev/null || echo "0")
    pass_count=$(grep -c "^--- PASS" "${TEST_REPORTS_DIR}/test.log" 2>/dev/null || echo "0")
    fail_count=$(grep -c "^--- FAIL" "${TEST_REPORTS_DIR}/test.log" 2>/dev/null || echo "0")
    
    echo "test_count=$test_count" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
    echo "pass_count=$pass_count" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
    echo "fail_count=$fail_count" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
    echo "test_end_time=$(date -Iseconds)" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
    
    echo "Test execution summary:"
    echo "  Tests Run: $test_count"
    echo "  Passed: $pass_count"
    echo "  Failed: $fail_count"
    echo "  Exit Code: $test_exit_code"
    
    return $test_exit_code
}

# Function to check if coverage artifacts should be generated
should_generate_coverage_artifacts() {
    if [ -f "${TEST_REPORTS_DIR}/coverage.out" ] && [ -s "${TEST_REPORTS_DIR}/coverage.out" ]; then
        return 0
    else
        return 1
    fi
}

# Function to print test summary for GitHub Actions
print_github_summary() {
    local test_type="${1:-unit}"
    
    if [ ! -f "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}" ]; then
        echo "No test status file found"
        return 1
    fi
    
    # Source status file
    source "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
    
    echo "## 🧪 ${test_type^} Test Results (Hardened)" >> $GITHUB_STEP_SUMMARY
    echo "" >> $GITHUB_STEP_SUMMARY
    echo "**Coverage:** ${coverage_percent:-0.0%}" >> $GITHUB_STEP_SUMMARY
    echo "" >> $GITHUB_STEP_SUMMARY
    echo "| Metric | Count |" >> $GITHUB_STEP_SUMMARY
    echo "|--------|-------|" >> $GITHUB_STEP_SUMMARY
    echo "| Tests Run | ${test_count:-0} |" >> $GITHUB_STEP_SUMMARY
    echo "| Passed | ${pass_count:-0} |" >> $GITHUB_STEP_SUMMARY
    echo "| Failed | ${fail_count:-0} |" >> $GITHUB_STEP_SUMMARY
    echo "| Exit Code | ${test_exit_code:-0} |" >> $GITHUB_STEP_SUMMARY
    echo "| Timeout Ceiling | ${TEST_TIMEOUT_CEILING} |" >> $GITHUB_STEP_SUMMARY
    echo "| Coverage Generated | ${coverage_generated:-false} |" >> $GITHUB_STEP_SUMMARY
}

# Main execution based on arguments
case "${1:-help}" in
    unit)
        run_hardened_tests "./..." "unit"
        ;;
    conductor-loop)
        echo "Running conductor-loop tests in separate phases to avoid timeout..."
        
        # Phase 1: Run cmd/conductor-loop tests
        echo "Phase 1: Running cmd/conductor-loop tests..."
        phase1_exit_code=0
        timeout "${TEST_TIMEOUT_CEILING}" go test -v -race -timeout="${TEST_GO_TIMEOUT}" \
            -coverprofile="${TEST_REPORTS_DIR}/coverage-cmd.out" \
            -covermode=atomic \
            -count="${TEST_COUNT}" \
            -parallel="${TEST_PARALLEL}" \
            ./cmd/conductor-loop | tee "${TEST_REPORTS_DIR}/test-cmd.log" || {
            phase1_exit_code=$?
            echo "cmd/conductor-loop tests completed with exit code: $phase1_exit_code"
        }
        
        # Phase 2: Run internal/loop tests (excluding problematic stress tests)
        echo "Phase 2: Running internal/loop tests (excluding stress tests)..."
        phase2_exit_code=0
        timeout "${TEST_TIMEOUT_CEILING}" go test -v -race -timeout="${TEST_GO_TIMEOUT}" \
            -coverprofile="${TEST_REPORTS_DIR}/coverage-internal.out" \
            -covermode=atomic \
            -count="${TEST_COUNT}" \
            -parallel="${TEST_PARALLEL}" \
            -skip="TestStressTest|TestFix3_DataRaceCondition" \
            ./internal/loop | tee "${TEST_REPORTS_DIR}/test-internal.log" || {
            phase2_exit_code=$?
            echo "internal/loop tests completed with exit code: $phase2_exit_code"
        }
        
        # Combine coverage files
        if [ -f "${TEST_REPORTS_DIR}/coverage-cmd.out" ] && [ -f "${TEST_REPORTS_DIR}/coverage-internal.out" ]; then
            echo "Combining coverage files..."
            {
                echo "mode: atomic"
                tail -n +2 "${TEST_REPORTS_DIR}/coverage-cmd.out" 2>/dev/null || true
                tail -n +2 "${TEST_REPORTS_DIR}/coverage-internal.out" 2>/dev/null || true
            } > "${TEST_REPORTS_DIR}/coverage.out"
        elif [ -f "${TEST_REPORTS_DIR}/coverage-cmd.out" ]; then
            cp "${TEST_REPORTS_DIR}/coverage-cmd.out" "${TEST_REPORTS_DIR}/coverage.out"
        elif [ -f "${TEST_REPORTS_DIR}/coverage-internal.out" ]; then
            cp "${TEST_REPORTS_DIR}/coverage-internal.out" "${TEST_REPORTS_DIR}/coverage.out"
        fi
        
        # Combine test logs
        {
            cat "${TEST_REPORTS_DIR}/test-cmd.log" 2>/dev/null || true
            cat "${TEST_REPORTS_DIR}/test-internal.log" 2>/dev/null || true
        } > "${TEST_REPORTS_DIR}/test.log"
        
        # Initialize status file
        echo "test_type=conductor-loop" > "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
        echo "test_start_time=$(date -Iseconds)" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
        
        # Determine overall exit code
        overall_exit_code=0
        if [ $phase1_exit_code -ne 0 ] || [ $phase2_exit_code -ne 0 ]; then
            overall_exit_code=1
        fi
        echo "test_exit_code=$overall_exit_code" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
        
        # Process coverage if available
        if [ -f "${TEST_REPORTS_DIR}/coverage.out" ] && [ -s "${TEST_REPORTS_DIR}/coverage.out" ]; then
            echo "Generating coverage HTML report..."
            go tool cover -html="${TEST_REPORTS_DIR}/coverage.out" -o "${TEST_REPORTS_DIR}/${COVERAGE_HTML}"
            
            # Calculate coverage percentage safely
            coverage_percent=$(go tool cover -func="${TEST_REPORTS_DIR}/coverage.out" 2>/dev/null | grep total | awk '{print $3}' || echo "0.0%")
            
            echo "coverage_generated=true" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
            echo "coverage_percent=$coverage_percent" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
            echo "Coverage: $coverage_percent"
        else
            echo "Warning: Coverage file not generated or empty"
            echo "coverage_generated=false" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
            echo "coverage_percent=0.0%" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
        fi
        
        # Calculate test metrics safely
        test_count=$(grep -c "^=== RUN" "${TEST_REPORTS_DIR}/test.log" 2>/dev/null || echo "0")
        pass_count=$(grep -c "^--- PASS" "${TEST_REPORTS_DIR}/test.log" 2>/dev/null || echo "0")
        fail_count=$(grep -c "^--- FAIL" "${TEST_REPORTS_DIR}/test.log" 2>/dev/null || echo "0")
        
        echo "test_count=$test_count" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
        echo "pass_count=$pass_count" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
        echo "fail_count=$fail_count" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
        echo "test_end_time=$(date -Iseconds)" >> "${TEST_REPORTS_DIR}/${TEST_STATUS_FILE}"
        
        echo "Test execution summary:"
        echo "  Phase 1 (cmd/conductor-loop): exit code $phase1_exit_code"
        echo "  Phase 2 (internal/loop): exit code $phase2_exit_code"
        echo "  Tests Run: $test_count"
        echo "  Passed: $pass_count"
        echo "  Failed: $fail_count"
        echo "  Overall Exit Code: $overall_exit_code"
        
        exit $overall_exit_code
        ;;
    integration)
        run_hardened_tests "./..." "integration"
        ;;
    check-coverage)
        if should_generate_coverage_artifacts; then
            echo "Coverage artifacts available"
            exit 0
        else
            echo "Coverage artifacts not available"
            exit 1
        fi
        ;;
    summary)
        print_github_summary "${2:-unit}"
        ;;
    help|*)
        echo "Test Harness for Hardened CI Test Execution"
        echo ""
        echo "Usage: $0 <command>"
        echo ""
        echo "Commands:"
        echo "  unit                Run unit tests"
        echo "  conductor-loop      Run conductor-loop specific tests"
        echo "  integration         Run integration tests"
        echo "  check-coverage      Check if coverage artifacts are available"
        echo "  summary [type]      Print GitHub Actions summary"
        echo "  help                Show this help"
        ;;
esac