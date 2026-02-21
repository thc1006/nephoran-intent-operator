#!/usr/bin/env bash
# =============================================================================
# run-all-e2e-tests.sh
# Master E2E Test Runner for NetworkIntent Operator
#
# Orchestrates all E2E test suites and generates comprehensive reports
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="${SCRIPT_DIR}/lib"
OUTPUT_DIR="${OUTPUT_DIR:-/tmp/e2e-test-results}"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
RUN_DIR="${OUTPUT_DIR}/run-${TIMESTAMP}"

# Source helper library
if [[ -f "${LIB_DIR}/test-helpers.sh" ]]; then
    source "${LIB_DIR}/test-helpers.sh"
else
    echo "ERROR: test-helpers.sh not found at ${LIB_DIR}/test-helpers.sh"
    exit 1
fi

# Test configuration
RAG_URL="${RAG_URL:-http://localhost:8000}"
INTENT_NAMESPACE="${INTENT_NAMESPACE:-nephoran-system}"
TIMEOUT="${E2E_TIMEOUT:-120}"

# Test suites to run
SUITES=(
    "test-comprehensive-pipeline.sh"
)

# Results tracking
declare -A SUITE_RESULTS
TOTAL_PASS=0
TOTAL_FAIL=0
TOTAL_SKIP=0
OVERALL_START=$(date +%s)

# ---------------------------------------------------------------------------
# Setup
# ---------------------------------------------------------------------------
mkdir -p "${RUN_DIR}"

echo ""
echo -e "${CYAN}${BOLD}"
cat << "EOF"
╔══════════════════════════════════════════════════════════════════════╗
║                                                                      ║
║         Nephoran Intent Operator - E2E Test Suite Runner            ║
║                                                                      ║
╚══════════════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"
echo "  Timestamp:     $(date -Iseconds)"
echo "  Output Dir:    ${RUN_DIR}"
echo "  RAG URL:       ${RAG_URL}"
echo "  Namespace:     ${INTENT_NAMESPACE}"
echo "  Timeout:       ${TIMEOUT}s"
echo ""

# ---------------------------------------------------------------------------
# Prerequisites
# ---------------------------------------------------------------------------
log_section "System Prerequisites"

if ! check_prerequisites; then
    log_fail "Prerequisites check failed"
    exit 1
fi

# Check RAG service details
log_info "RAG Service Details:"
if RAG_VERSION=$(curl -sf --max-time 5 "${RAG_URL}/version" 2>/dev/null); then
    echo "  Version: ${RAG_VERSION}"
else
    echo "  Version: Unknown (endpoint not available)"
fi

# Check Ollama models
log_info "Checking available Ollama models..."
if OLLAMA_MODELS=$(kubectl exec -n ollama deployment/ollama -- ollama list 2>/dev/null); then
    echo "${OLLAMA_MODELS}" | head -5
else
    log_skip "Could not query Ollama models"
fi

# ---------------------------------------------------------------------------
# Test Suite Execution
# ---------------------------------------------------------------------------
log_section "Executing Test Suites"

for suite_script in "${SUITES[@]}"; do
    suite_path="${SCRIPT_DIR}/${suite_script}"
    suite_name=$(basename "${suite_script}" .sh)
    suite_log="${RUN_DIR}/${suite_name}.log"
    suite_result="${RUN_DIR}/${suite_name}.result"

    log_section "Running: ${suite_name}"

    if [[ ! -f "${suite_path}" ]]; then
        log_fail "Suite not found: ${suite_path}"
        SUITE_RESULTS["${suite_name}"]="NOT_FOUND"
        continue
    fi

    # Make executable
    chmod +x "${suite_path}"

    # Run suite
    suite_start=$(date +%s)
    if OUTPUT_DIR="${RUN_DIR}" "${suite_path}" &> "${suite_log}"; then
        suite_status="PASS"
        log_pass "Suite ${suite_name} completed successfully"
    else
        suite_status="FAIL"
        log_fail "Suite ${suite_name} failed"
    fi
    suite_end=$(date +%s)
    suite_duration=$((suite_end - suite_start))

    SUITE_RESULTS["${suite_name}"]="${suite_status}"

    # Extract metrics from suite output
    if [[ -f "${suite_log}" ]]; then
        suite_pass=$(grep -c "^\[PASS\]" "${suite_log}" 2>/dev/null || echo "0")
        suite_fail=$(grep -c "^\[FAIL\]" "${suite_log}" 2>/dev/null || echo "0")
        suite_skip=$(grep -c "^\[SKIP\]" "${suite_log}" 2>/dev/null || echo "0")

        TOTAL_PASS=$((TOTAL_PASS + suite_pass))
        TOTAL_FAIL=$((TOTAL_FAIL + suite_fail))
        TOTAL_SKIP=$((TOTAL_SKIP + suite_skip))

        echo "${suite_status}|${suite_pass}|${suite_fail}|${suite_skip}|${suite_duration}" > "${suite_result}"
    fi

    # Show last 10 lines of output
    log_debug "Last lines of suite output:"
    tail -10 "${suite_log}" | sed 's/^/  /' >&2 || true
done

OVERALL_END=$(date +%s)
OVERALL_DURATION=$((OVERALL_END - OVERALL_START))

# ---------------------------------------------------------------------------
# Aggregate Results
# ---------------------------------------------------------------------------
log_section "Generating Aggregate Reports"

TOTAL_TESTS=$((TOTAL_PASS + TOTAL_FAIL + TOTAL_SKIP))
if [[ ${TOTAL_TESTS} -gt 0 ]]; then
    PASS_RATE=$((TOTAL_PASS * 100 / TOTAL_TESTS))
else
    PASS_RATE=0
fi

# Generate summary report
SUMMARY_FILE="${RUN_DIR}/SUMMARY.txt"
cat > "${SUMMARY_FILE}" << EOF
╔══════════════════════════════════════════════════════════════════════╗
║       NetworkIntent Operator - E2E Test Results Summary             ║
╚══════════════════════════════════════════════════════════════════════╝

RUN METADATA
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Timestamp:         $(date -Iseconds)
  Run ID:            ${TIMESTAMP}
  Duration:          ${OVERALL_DURATION}s
  Output Directory:  ${RUN_DIR}

ENVIRONMENT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  RAG Service:       ${RAG_URL}
  Namespace:         ${INTENT_NAMESPACE}
  Timeout:           ${TIMEOUT}s
  Kubernetes:        $(kubectl version --short 2>/dev/null | grep Server | awk '{print $3}' || echo "Unknown")

OVERALL RESULTS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Total Tests:       ${TOTAL_TESTS}
  ✅ Passed:         ${TOTAL_PASS} (${PASS_RATE}%)
  ❌ Failed:         ${TOTAL_FAIL}
  ⏭️  Skipped:       ${TOTAL_SKIP}

  Pass Rate:         $( [[ ${PASS_RATE} -ge 80 ]] && echo "✅" || echo "❌" ) ${PASS_RATE}%

SUITE RESULTS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF

# Add suite breakdown
for suite_name in "${!SUITE_RESULTS[@]}"; do
    suite_result_file="${RUN_DIR}/${suite_name}.result"
    if [[ -f "${suite_result_file}" ]]; then
        IFS='|' read -r status pass fail skip duration < "${suite_result_file}"
        printf "  %-40s %s (%s/%s/%s) %ss\n" \
            "${suite_name}:" \
            "$( [[ "${status}" == "PASS" ]] && echo "✅ PASS" || echo "❌ FAIL" )" \
            "${pass}P" "${fail}F" "${skip}S" "${duration}" >> "${SUMMARY_FILE}"
    else
        printf "  %-40s ❌ NOT_RUN\n" "${suite_name}:" >> "${SUMMARY_FILE}"
    fi
done

cat >> "${SUMMARY_FILE}" << EOF

PERFORMANCE METRICS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF

# Extract performance metrics if available
if [[ -f "${RUN_DIR}/performance.csv" ]]; then
    # Calculate average RAG response time
    AVG_RAG_TIME=$(awk -F',' 'NR>1 && $4>0 {sum+=$4; count++} END {if(count>0) printf "%.2f", sum/count/1000; else print "N/A"}' \
        "${RUN_DIR}/performance.csv" 2>/dev/null || echo "N/A")
    echo "  Avg RAG Response:  ${AVG_RAG_TIME}s" >> "${SUMMARY_FILE}"
fi

cat >> "${SUMMARY_FILE}" << EOF

TEST ARTIFACTS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  - Summary:         ${SUMMARY_FILE}
  - JUnit XML:       ${RUN_DIR}/junit-results.xml
  - JSON Results:    ${RUN_DIR}/results.json
  - CSV Metrics:     ${RUN_DIR}/performance.csv
  - Full Logs:       ${RUN_DIR}/*.log

RECOMMENDATION
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF

if [[ ${TOTAL_FAIL} -eq 0 ]]; then
    echo "  ✅ All tests passed! System ready for production." >> "${SUMMARY_FILE}"
elif [[ ${PASS_RATE} -ge 80 ]]; then
    echo "  ⚠️  Most tests passed (${PASS_RATE}%). Review failures before production." >> "${SUMMARY_FILE}"
else
    echo "  ❌ Significant test failures (${PASS_RATE}% pass rate). DO NOT deploy." >> "${SUMMARY_FILE}"
fi

cat >> "${SUMMARY_FILE}" << EOF

╚══════════════════════════════════════════════════════════════════════╝

Generated by: Nephoran E2E Test Runner
Cluster: $(kubectl config current-context 2>/dev/null || echo "Unknown")
EOF

# Display summary
cat "${SUMMARY_FILE}"

# ---------------------------------------------------------------------------
# Generate JUnit XML (aggregate)
# ---------------------------------------------------------------------------
if [[ -f "${RUN_DIR}/junit-results.xml" ]]; then
    # Already generated by comprehensive test
    log_pass "JUnit XML report available: ${RUN_DIR}/junit-results.xml"
else
    log_skip "JUnit XML not generated (no compatible test suite ran)"
fi

# ---------------------------------------------------------------------------
# Generate JSON report
# ---------------------------------------------------------------------------
JSON_REPORT="${RUN_DIR}/aggregate-results.json"
cat > "${JSON_REPORT}" << EOF
{
  "run_id": "${TIMESTAMP}",
  "timestamp": "$(date -Iseconds)",
  "duration_seconds": ${OVERALL_DURATION},
  "environment": {
    "rag_url": "${RAG_URL}",
    "namespace": "${INTENT_NAMESPACE}",
    "timeout": ${TIMEOUT}
  },
  "summary": {
    "total_tests": ${TOTAL_TESTS},
    "passed": ${TOTAL_PASS},
    "failed": ${TOTAL_FAIL},
    "skipped": ${TOTAL_SKIP},
    "pass_rate": ${PASS_RATE}
  },
  "suites": [
EOF

# Add suite details
FIRST_SUITE=1
for suite_name in "${!SUITE_RESULTS[@]}"; do
    [[ ${FIRST_SUITE} -eq 0 ]] && echo "," >> "${JSON_REPORT}"
    FIRST_SUITE=0

    suite_result_file="${RUN_DIR}/${suite_name}.result"
    if [[ -f "${suite_result_file}" ]]; then
        IFS='|' read -r status pass fail skip duration < "${suite_result_file}"
        cat >> "${JSON_REPORT}" << SUITE_EOF
    {
      "name": "${suite_name}",
      "status": "${status}",
      "passed": ${pass},
      "failed": ${fail},
      "skipped": ${skip},
      "duration_seconds": ${duration}
    }
SUITE_EOF
    fi
done

cat >> "${JSON_REPORT}" << EOF

  ]
}
EOF

log_pass "JSON report generated: ${JSON_REPORT}"

# ---------------------------------------------------------------------------
# Cleanup and Exit
# ---------------------------------------------------------------------------
log_section "Cleanup"

# Optional: Clean up test NetworkIntents
if [[ "${CLEANUP_INTENTS:-1}" == "1" ]]; then
    cleanup_test_intents
    log_pass "Test resources cleaned up"
else
    log_skip "Resource cleanup disabled (CLEANUP_INTENTS=0)"
fi

# Final status
echo ""
if [[ ${TOTAL_FAIL} -eq 0 ]]; then
    echo -e "${GREEN}${BOLD}╔══════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}${BOLD}║  ✅ ALL E2E TESTS PASSED SUCCESSFULLY       ║${NC}"
    echo -e "${GREEN}${BOLD}╚══════════════════════════════════════════════╝${NC}"
    exit 0
else
    echo -e "${RED}${BOLD}╔══════════════════════════════════════════════╗${NC}"
    echo -e "${RED}${BOLD}║  ❌ SOME E2E TESTS FAILED                   ║${NC}"
    echo -e "${RED}${BOLD}╚══════════════════════════════════════════════╝${NC}"
    echo ""
    echo "Review logs in: ${RUN_DIR}"
    exit 1
fi
