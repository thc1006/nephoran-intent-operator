#!/usr/bin/env bash
# =============================================================================
# run-all-tests.sh
# Master Smoke Test Runner for Nephoran Intent Operator E2E Suite
#
# Runs all E2E test scripts in sequence, collects results, and generates
# a summary report. Exits with code 0 only if all tests pass.
#
# Test execution order:
#   1. test-intent-lifecycle.sh  -- NetworkIntent CR lifecycle
#   2. test-rag-pipeline.sh      -- RAG service integration
#   3. test-gpu-allocation.sh    -- GPU DRA allocation
#   4. test-monitoring.sh        -- Monitoring stack verification
#
# Usage:
#   ./run-all-tests.sh [--skip <test-name>] [--only <test-name>]
#                      [--report-dir <dir>] [--timeout <seconds>]
#                      [--parallel]
#
# Environment Variables (overridable):
#   E2E_NAMESPACE       -- Kubernetes namespace for tests
#   E2E_TIMEOUT         -- Default timeout per test (seconds)
#   RAG_URL             -- RAG service URL
#   WEAVIATE_URL        -- Weaviate URL
#   OLLAMA_URL          -- Ollama URL
#   PROMETHEUS_URL      -- Prometheus URL
#   GRAFANA_URL         -- Grafana URL
# =============================================================================

set -uo pipefail
# Note: not using set -e because we want to continue after individual test failures

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_DIR="${REPORT_DIR:-${SCRIPT_DIR}/reports}"
PER_TEST_TIMEOUT="${E2E_TIMEOUT:-180}"
SKIP_TESTS=()
ONLY_TESTS=()
RUN_PARALLEL=false
SUITE_START_TIME=$(date +%s)

# Test registry: name -> script path
declare -A TEST_SCRIPTS=(
    ["intent-lifecycle"]="${SCRIPT_DIR}/test-intent-lifecycle.sh"
    ["rag-pipeline"]="${SCRIPT_DIR}/test-rag-pipeline.sh"
    ["gpu-allocation"]="${SCRIPT_DIR}/test-gpu-allocation.sh"
    ["monitoring"]="${SCRIPT_DIR}/test-monitoring.sh"
)

# Ordered execution list
TEST_ORDER=("intent-lifecycle" "rag-pipeline" "gpu-allocation" "monitoring")

# Results tracking
declare -A TEST_RESULTS
declare -A TEST_DURATIONS

# ---------------------------------------------------------------------------
# Color helpers
# ---------------------------------------------------------------------------
if [[ -t 1 ]]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
    BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; BLUE=''; CYAN=''; BOLD=''; NC=''
fi

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log_info()    { echo -e "${BLUE}[INFO]${NC}  $(date '+%H:%M:%S') $*"; }
log_pass()    { echo -e "${GREEN}[PASS]${NC}  $(date '+%H:%M:%S') $*"; }
log_fail()    { echo -e "${RED}[FAIL]${NC}  $(date '+%H:%M:%S') $*"; }
log_skip()    { echo -e "${YELLOW}[SKIP]${NC}  $(date '+%H:%M:%S') $*"; }
log_section() { echo -e "\n${CYAN}${BOLD}=== $* ===${NC}"; }

# ---------------------------------------------------------------------------
# Parse CLI arguments
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --skip)
            IFS=',' read -ra SKIP_TESTS <<< "$2"
            shift 2
            ;;
        --only)
            IFS=',' read -ra ONLY_TESTS <<< "$2"
            shift 2
            ;;
        --report-dir)
            REPORT_DIR="$2"
            shift 2
            ;;
        --timeout)
            PER_TEST_TIMEOUT="$2"
            shift 2
            ;;
        --parallel)
            RUN_PARALLEL=true
            shift
            ;;
        --help|-h)
            cat <<EOF
Usage: $0 [OPTIONS]

Options:
    --skip <tests>       Comma-separated list of tests to skip
                         (intent-lifecycle,rag-pipeline,gpu-allocation,monitoring)
    --only <tests>       Comma-separated list of tests to run exclusively
    --report-dir <dir>   Directory for reports (default: ./reports)
    --timeout <seconds>  Timeout per test (default: 180)
    --parallel           Run tests in parallel (experimental)
    -h, --help           Show this help message

Available tests:
    intent-lifecycle     NetworkIntent CR lifecycle
    rag-pipeline         RAG service integration
    gpu-allocation       GPU DRA allocation
    monitoring           Monitoring stack verification

Environment Variables:
    E2E_NAMESPACE        Kubernetes namespace (default: nephoran-e2e-test)
    RAG_URL              RAG service URL (default: http://localhost:8000)
    WEAVIATE_URL         Weaviate URL (default: http://localhost:8080)
    OLLAMA_URL           Ollama URL (default: http://localhost:11434)
    PROMETHEUS_URL       Prometheus URL (default: http://localhost:9090)
    GRAFANA_URL          Grafana URL (default: http://localhost:3000)

Examples:
    $0                                    # Run all tests
    $0 --only intent-lifecycle            # Run only intent lifecycle test
    $0 --skip gpu-allocation,monitoring   # Skip GPU and monitoring tests
    $0 --timeout 300 --report-dir /tmp    # Custom timeout and report dir
EOF
            exit 0
            ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

# ---------------------------------------------------------------------------
# Determine which tests to run
# ---------------------------------------------------------------------------
should_run_test() {
    local test_name="$1"

    # If --only is specified, only run those tests
    if [[ ${#ONLY_TESTS[@]} -gt 0 ]]; then
        for only in "${ONLY_TESTS[@]}"; do
            if [[ "${only}" == "${test_name}" ]]; then
                return 0
            fi
        done
        return 1
    fi

    # Check skip list
    for skip in "${SKIP_TESTS[@]}"; do
        if [[ "${skip}" == "${test_name}" ]]; then
            return 1
        fi
    done

    return 0
}

# ---------------------------------------------------------------------------
# Run a single test
# ---------------------------------------------------------------------------
run_single_test() {
    local test_name="$1"
    local script_path="${TEST_SCRIPTS[${test_name}]}"
    local log_file="${REPORT_DIR}/${test_name}.log"
    local start_time duration exit_code

    if [[ ! -f "${script_path}" ]]; then
        log_fail "Test script not found: ${script_path}"
        TEST_RESULTS["${test_name}"]="ERROR"
        TEST_DURATIONS["${test_name}"]=0
        return 1
    fi

    if [[ ! -x "${script_path}" ]]; then
        chmod +x "${script_path}"
    fi

    log_section "Running: ${test_name}"
    log_info "Script: ${script_path}"
    log_info "Log:    ${log_file}"

    start_time=$(date +%s)

    # Run with timeout, capture both stdout and stderr
    set +e
    timeout "${PER_TEST_TIMEOUT}" bash "${script_path}" 2>&1 | tee "${log_file}"
    exit_code=${PIPESTATUS[0]}
    set -e

    duration=$(( $(date +%s) - start_time ))
    TEST_DURATIONS["${test_name}"]=${duration}

    if [[ ${exit_code} -eq 0 ]]; then
        TEST_RESULTS["${test_name}"]="PASSED"
        log_pass "${test_name} completed in ${duration}s"
    elif [[ ${exit_code} -eq 124 ]]; then
        TEST_RESULTS["${test_name}"]="TIMEOUT"
        log_fail "${test_name} timed out after ${PER_TEST_TIMEOUT}s"
    else
        TEST_RESULTS["${test_name}"]="FAILED"
        log_fail "${test_name} failed with exit code ${exit_code} (${duration}s)"
    fi

    return ${exit_code}
}

# =============================================================================
# MAIN EXECUTION
# =============================================================================

echo ""
echo -e "${CYAN}${BOLD}"
echo "============================================================"
echo "  Nephoran Intent Operator - E2E Test Suite"
echo "  $(date -Iseconds)"
echo "============================================================"
echo -e "${NC}"

# Create report directory
mkdir -p "${REPORT_DIR}"
log_info "Report directory: ${REPORT_DIR}"

# ---------------------------------------------------------------------------
# Prerequisites check
# ---------------------------------------------------------------------------
log_section "Prerequisites"

if ! command -v kubectl >/dev/null 2>&1; then
    log_fail "kubectl not found in PATH"
    exit 1
fi
log_pass "kubectl available"

if ! command -v curl >/dev/null 2>&1; then
    log_fail "curl not found in PATH"
    exit 1
fi
log_pass "curl available"

if ! command -v python3 >/dev/null 2>&1; then
    log_info "python3 not found -- some tests may have limited JSON parsing"
else
    log_pass "python3 available"
fi

if kubectl cluster-info >/dev/null 2>&1; then
    log_pass "Kubernetes cluster accessible"
else
    log_fail "Cannot reach Kubernetes cluster"
    exit 1
fi

# ---------------------------------------------------------------------------
# Execute tests
# ---------------------------------------------------------------------------
TOTAL_PASS=0
TOTAL_FAIL=0
TOTAL_SKIP=0
TOTAL_TIMEOUT=0

for test_name in "${TEST_ORDER[@]}"; do
    if ! should_run_test "${test_name}"; then
        log_skip "Skipping ${test_name} (filtered)"
        TEST_RESULTS["${test_name}"]="SKIPPED"
        TEST_DURATIONS["${test_name}"]=0
        TOTAL_SKIP=$((TOTAL_SKIP + 1))
        continue
    fi

    if run_single_test "${test_name}"; then
        TOTAL_PASS=$((TOTAL_PASS + 1))
    else
        case "${TEST_RESULTS[${test_name}]}" in
            TIMEOUT) TOTAL_TIMEOUT=$((TOTAL_TIMEOUT + 1)) ;;
            *)       TOTAL_FAIL=$((TOTAL_FAIL + 1)) ;;
        esac
    fi

    echo ""
done

# ---------------------------------------------------------------------------
# Generate Summary Report
# ---------------------------------------------------------------------------
log_section "GENERATING SUMMARY REPORT"

SUITE_END_TIME=$(date +%s)
SUITE_DURATION=$((SUITE_END_TIME - SUITE_START_TIME))
SUITE_DURATION_MIN=$((SUITE_DURATION / 60))
SUITE_DURATION_SEC=$((SUITE_DURATION % 60))
TOTAL_RUN=$((TOTAL_PASS + TOTAL_FAIL + TOTAL_TIMEOUT))
REPORT_FILE="${REPORT_DIR}/e2e-summary.txt"

# Console output
echo ""
echo -e "${CYAN}${BOLD}"
echo "============================================================"
echo "                 E2E TEST SUITE RESULTS"
echo "============================================================"
echo -e "${NC}"

printf "  %-25s %-10s %s\n" "TEST" "RESULT" "DURATION"
printf "  %-25s %-10s %s\n" "----" "------" "--------"

for test_name in "${TEST_ORDER[@]}"; do
    result="${TEST_RESULTS[${test_name}]:-UNKNOWN}"
    duration="${TEST_DURATIONS[${test_name}]:-0}"

    case "${result}" in
        PASSED)  color="${GREEN}" ;;
        FAILED)  color="${RED}" ;;
        TIMEOUT) color="${RED}" ;;
        SKIPPED) color="${YELLOW}" ;;
        *)       color="${NC}" ;;
    esac

    printf "  %-25s ${color}%-10s${NC} %ss\n" "${test_name}" "${result}" "${duration}"
done

echo ""
echo "  -------------------------------------------"
echo -e "  Total:    ${TOTAL_RUN} test(s) executed"
echo -e "  Passed:   ${GREEN}${TOTAL_PASS}${NC}"
echo -e "  Failed:   ${RED}${TOTAL_FAIL}${NC}"
echo -e "  Timeout:  ${RED}${TOTAL_TIMEOUT}${NC}"
echo -e "  Skipped:  ${YELLOW}${TOTAL_SKIP}${NC}"
echo "  Duration: ${SUITE_DURATION_MIN}m ${SUITE_DURATION_SEC}s"
echo ""

# Write text report file
{
    echo "Nephoran Intent Operator - E2E Test Suite Results"
    echo "================================================="
    echo "Date:     $(date -Iseconds)"
    echo "Duration: ${SUITE_DURATION_MIN}m ${SUITE_DURATION_SEC}s"
    echo ""
    printf "%-25s %-10s %s\n" "TEST" "RESULT" "DURATION"
    printf "%-25s %-10s %s\n" "----" "------" "--------"
    for test_name in "${TEST_ORDER[@]}"; do
        result="${TEST_RESULTS[${test_name}]:-UNKNOWN}"
        duration="${TEST_DURATIONS[${test_name}]:-0}"
        printf "%-25s %-10s %ss\n" "${test_name}" "${result}" "${duration}"
    done
    echo ""
    echo "Summary: ${TOTAL_PASS} passed, ${TOTAL_FAIL} failed, ${TOTAL_TIMEOUT} timed out, ${TOTAL_SKIP} skipped"
} > "${REPORT_FILE}"

log_info "Summary report written to: ${REPORT_FILE}"

# Generate JSON report
JSON_REPORT="${REPORT_DIR}/e2e-summary.json"
python3 -c "
import json, time

results = {}
$(for test_name in "${TEST_ORDER[@]}"; do
    echo "results['${test_name}'] = {'result': '${TEST_RESULTS[${test_name}]:-UNKNOWN}', 'duration_seconds': ${TEST_DURATIONS[${test_name}]:-0}}"
done)

report = {
    'suite': 'nephoran-e2e',
    'timestamp': '$(date -Iseconds)',
    'duration_seconds': ${SUITE_DURATION},
    'summary': {
        'total': ${TOTAL_RUN},
        'passed': ${TOTAL_PASS},
        'failed': ${TOTAL_FAIL},
        'timeout': ${TOTAL_TIMEOUT},
        'skipped': ${TOTAL_SKIP},
    },
    'tests': results,
    'overall_result': 'PASSED' if ${TOTAL_FAIL} == 0 and ${TOTAL_TIMEOUT} == 0 else 'FAILED'
}

with open('${JSON_REPORT}', 'w') as f:
    json.dump(report, f, indent=2)
print(f'JSON report written to: ${JSON_REPORT}')
" 2>/dev/null || log_info "JSON report generation skipped (python3 not available)"

# ---------------------------------------------------------------------------
# Final exit code
# ---------------------------------------------------------------------------
if [[ $((TOTAL_FAIL + TOTAL_TIMEOUT)) -gt 0 ]]; then
    echo -e "${RED}${BOLD}OVERALL RESULT: FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}${BOLD}OVERALL RESULT: PASSED${NC}"
    exit 0
fi
