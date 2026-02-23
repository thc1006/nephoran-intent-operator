#!/usr/bin/env bash
# =============================================================================
# test-helpers.sh
# Shared helper functions for E2E tests
# =============================================================================

# ---------------------------------------------------------------------------
# Configuration defaults
# ---------------------------------------------------------------------------
export RAG_URL="${RAG_URL:-http://localhost:8000}"
export INTENT_NAMESPACE="${INTENT_NAMESPACE:-nephoran-system}"
export TIMEOUT="${E2E_TIMEOUT:-120}"
export CHECK_INTERVAL="${CHECK_INTERVAL:-5}"
export MAX_RETRIES=$((TIMEOUT / CHECK_INTERVAL))

# ---------------------------------------------------------------------------
# Color codes
# ---------------------------------------------------------------------------
if [[ -t 1 ]]; then
    export RED='\033[0;31m'
    export GREEN='\033[0;32m'
    export YELLOW='\033[1;33m'
    export BLUE='\033[0;34m'
    export CYAN='\033[0;36m'
    export BOLD='\033[1m'
    export NC='\033[0m'
else
    export RED='' GREEN='' YELLOW='' BLUE='' CYAN='' BOLD='' NC=''
fi

# ---------------------------------------------------------------------------
# Logging functions
# ---------------------------------------------------------------------------
log_info() {
    echo -e "${BLUE}[INFO]${NC}  $(date '+%H:%M:%S') $*" >&2
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC}  $(date '+%H:%M:%S') $*" >&2
}

log_fail() {
    echo -e "${RED}[FAIL]${NC}  $(date '+%H:%M:%S') $*" >&2
}

log_skip() {
    echo -e "${YELLOW}[SKIP]${NC}  $(date '+%H:%M:%S') $*" >&2
}

log_section() {
    echo -e "\n${CYAN}${BOLD}=== $* ===${NC}" >&2
}

log_debug() {
    if [[ "${DEBUG:-0}" == "1" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $(date '+%H:%M:%S') $*" >&2
    fi
}

# ---------------------------------------------------------------------------
# Prerequisite checks
# ---------------------------------------------------------------------------
check_prerequisites() {
    local errors=0

    log_section "Checking Prerequisites"

    # Check NetworkIntent CRD
    if ! kubectl get crd networkintents.intent.nephoran.com &>/dev/null; then
        log_fail "NetworkIntent CRD not found"
        ((errors++))
    else
        log_pass "NetworkIntent CRD registered"
    fi

    # Check RAG service
    if ! curl -sf --max-time 5 "${RAG_URL}/health" &>/dev/null; then
        log_fail "RAG service not accessible at ${RAG_URL}"
        ((errors++))
    else
        log_pass "RAG service accessible"
    fi

    # Check operator (optional warning)
    if ! kubectl get deployment -n "${INTENT_NAMESPACE}" nephoran-controller-manager &>/dev/null; then
        log_skip "Nephoran Intent Operator not found"
    else
        log_pass "Nephoran Intent Operator running"
    fi

    # Check kubectl access
    if ! kubectl version --client &>/dev/null; then
        log_fail "kubectl not available"
        ((errors++))
    else
        log_pass "kubectl available"
    fi

    return ${errors}
}

# ---------------------------------------------------------------------------
# NetworkIntent operations
# ---------------------------------------------------------------------------

# Wait for NetworkIntent to reach expected phase
# Usage: wait_for_intent_phase <intent_name> <expected_phase>
# Returns: 0 on success, 1 on failure
wait_for_intent_phase() {
    local intent_name="$1"
    local expected_phase="$2"
    local retries=0
    local last_phase=""

    log_info "Waiting for NetworkIntent '${intent_name}' to reach phase '${expected_phase}' (timeout: ${TIMEOUT}s)"

    while [[ ${retries} -lt ${MAX_RETRIES} ]]; do
        local current_phase
        current_phase=$(kubectl get networkintent "${intent_name}" -n "${INTENT_NAMESPACE}" \
            -o jsonpath='{.status.phase}' 2>/dev/null || echo "Pending")

        # Log phase transitions
        if [[ "${current_phase}" != "${last_phase}" ]]; then
            log_info "Phase: ${last_phase:-None} -> ${current_phase}"
            last_phase="${current_phase}"
        fi

        # Success
        if [[ "${current_phase}" == "${expected_phase}" ]]; then
            log_pass "NetworkIntent reached phase '${expected_phase}'"
            return 0
        fi

        # Error states
        if [[ "${current_phase}" == "Error" ]] || [[ "${current_phase}" == "Failed" ]]; then
            log_fail "NetworkIntent entered error state: ${current_phase}"
            log_debug "NetworkIntent YAML:"
            kubectl get networkintent "${intent_name}" -n "${INTENT_NAMESPACE}" -o yaml >&2
            return 1
        fi

        ((retries++))
        sleep "${CHECK_INTERVAL}"
    done

    log_fail "Timeout waiting for phase '${expected_phase}' (last: ${current_phase})"
    return 1
}

# Create NetworkIntent via RAG service
# Usage: create_intent_via_rag <scenario> [expected_replicas] [test_id]
# Outputs: NetworkIntent name on success
# Returns: 0 on success, 1 on failure
create_intent_via_rag() {
    local scenario="$1"
    local expected_replicas="${2:-1}"
    local test_id="${3:-$(date +%s%N)}"
    local start_time end_time duration_ms

    log_info "Creating intent via RAG service"
    log_debug "Scenario: ${scenario}"
    log_debug "Expected replicas: ${expected_replicas}"

    start_time=$(date +%s%N)

    # Call RAG service
    local response http_code
    response=$(curl -s --max-time "${TIMEOUT}" -w "\n%{http_code}" -X POST \
        "${RAG_URL}/process" \
        -H "Content-Type: application/json" \
        -d "{\"intent\":\"${scenario}\",\"intent_id\":\"e2e-${test_id}\"}" 2>&1 || echo -e '\n000')

    http_code=$(echo "${response}" | tail -1)
    response=$(echo "${response}" | head -n -1)

    end_time=$(date +%s%N)
    duration_ms=$(( (end_time - start_time) / 1000000 ))

    log_info "RAG response time: ${duration_ms}ms (HTTP ${http_code})"

    # Check HTTP status
    if [[ "${http_code}" != "200" ]]; then
        log_fail "RAG request failed with HTTP ${http_code}"
        log_debug "Response: ${response}"
        return 1
    fi

    # Validate JSON
    if ! echo "${response}" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
        log_fail "RAG returned invalid JSON"
        log_debug "Response: ${response}"
        return 1
    fi

    # Performance check
    if [[ ${duration_ms} -gt 10000 ]]; then
        log_fail "RAG response exceeded 10s threshold (${duration_ms}ms)"
    else
        log_pass "RAG response within threshold (${duration_ms}ms)"
    fi

    # Extract NetworkIntent name from response
    local intent_name
    intent_name=$(echo "${response}" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    # Try common field names
    for field in ['intent_name', 'name', 'networkIntentName', 'intentName', 'id']:
        if field in data and data[field]:
            print(data[field])
            sys.exit(0)
except Exception as e:
    pass
" 2>/dev/null || echo "")

    # Fallback: Get most recent NetworkIntent
    if [[ -z "${intent_name}" ]]; then
        sleep 2  # Wait for K8s API
        intent_name=$(kubectl get networkintent -n "${INTENT_NAMESPACE}" \
            --sort-by='.metadata.creationTimestamp' \
            -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null || echo "")
    fi

    if [[ -z "${intent_name}" ]]; then
        log_fail "Could not determine NetworkIntent name"
        return 1
    fi

    log_pass "NetworkIntent created: ${intent_name}"

    # Verify replicas if specified
    if [[ ${expected_replicas} -ge 0 ]]; then
        sleep 2  # Wait for spec
        local actual_replicas
        actual_replicas=$(kubectl get networkintent "${intent_name}" -n "${INTENT_NAMESPACE}" \
            -o jsonpath='{.spec.desiredReplicas}' 2>/dev/null || echo "-1")

        if [[ "${actual_replicas}" == "${expected_replicas}" ]]; then
            log_pass "NetworkIntent replicas correct: ${actual_replicas}"
        else
            log_fail "Replicas mismatch: expected ${expected_replicas}, got ${actual_replicas}"
        fi
    fi

    echo "${intent_name}"
    return 0
}

# Delete NetworkIntent
# Usage: delete_intent <intent_name>
delete_intent() {
    local intent_name="$1"

    log_debug "Deleting NetworkIntent: ${intent_name}"
    kubectl delete networkintent "${intent_name}" -n "${INTENT_NAMESPACE}" \
        --ignore-not-found=true --wait=false &>/dev/null || true
}

# ---------------------------------------------------------------------------
# Performance metrics
# ---------------------------------------------------------------------------

# Measure RAG latency
# Usage: measure_rag_latency <scenario>
# Outputs: Latency in milliseconds
measure_rag_latency() {
    local scenario="$1"
    local start_time end_time

    start_time=$(date +%s%N)
    curl -s --max-time "${TIMEOUT}" -X POST \
        "${RAG_URL}/process" \
        -H "Content-Type: application/json" \
        -d "{\"intent\":\"${scenario}\"}" &>/dev/null || true
    end_time=$(date +%s%N)

    echo $(( (end_time - start_time) / 1000000 ))
}

# ---------------------------------------------------------------------------
# Validation helpers
# ---------------------------------------------------------------------------

# Validate JSON schema
# Usage: validate_json <json_string>
validate_json() {
    local json="$1"
    echo "${json}" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null
}

# Check Kubernetes resource exists
# Usage: resource_exists <resource_type> <name> [namespace]
resource_exists() {
    local resource_type="$1"
    local name="$2"
    local namespace="${3:-${INTENT_NAMESPACE}}"

    kubectl get "${resource_type}" "${name}" -n "${namespace}" &>/dev/null
}

# ---------------------------------------------------------------------------
# Test result tracking
# ---------------------------------------------------------------------------

# Global counters (initialize if not set)
export PASS_COUNT="${PASS_COUNT:-0}"
export FAIL_COUNT="${FAIL_COUNT:-0}"
export SKIP_COUNT="${SKIP_COUNT:-0}"

increment_pass() { PASS_COUNT=$((PASS_COUNT + 1)); }
increment_fail() { FAIL_COUNT=$((FAIL_COUNT + 1)); }
increment_skip() { SKIP_COUNT=$((SKIP_COUNT + 1)); }

get_total_tests() { echo $((PASS_COUNT + FAIL_COUNT + SKIP_COUNT)); }
get_pass_rate() {
    local total=$(get_total_tests)
    if [[ ${total} -eq 0 ]]; then echo "0"; return; fi
    echo $(( PASS_COUNT * 100 / total ))
}

# ---------------------------------------------------------------------------
# Cleanup utilities
# ---------------------------------------------------------------------------

# Array to track resources for cleanup
declare -ga CLEANUP_INTENTS=()

register_cleanup_intent() {
    CLEANUP_INTENTS+=("$1")
}

cleanup_test_intents() {
    log_info "Cleaning up ${#CLEANUP_INTENTS[@]} test NetworkIntents..."
    for intent_name in "${CLEANUP_INTENTS[@]}"; do
        delete_intent "${intent_name}"
    done
    CLEANUP_INTENTS=()
}

# ---------------------------------------------------------------------------
# Report generation
# ---------------------------------------------------------------------------

# Print test summary
print_test_summary() {
    local total=$(get_total_tests)
    local pass_rate=$(get_pass_rate)

    echo ""
    echo -e "${CYAN}${BOLD}========================================${NC}"
    echo -e "${CYAN}${BOLD}TEST SUMMARY${NC}"
    echo -e "${CYAN}${BOLD}========================================${NC}"
    echo "  Total Tests:  ${total}"
    echo -e "  Passed:       ${GREEN}${PASS_COUNT}${NC} (${pass_rate}%)"
    echo -e "  Failed:       ${RED}${FAIL_COUNT}${NC}"
    echo -e "  Skipped:      ${YELLOW}${SKIP_COUNT}${NC}"
    echo ""

    if [[ ${FAIL_COUNT} -eq 0 ]]; then
        echo -e "${GREEN}${BOLD}✓ ALL TESTS PASSED${NC}"
        return 0
    else
        echo -e "${RED}${BOLD}✗ SOME TESTS FAILED${NC}"
        return 1
    fi
}

# Export all functions
export -f log_info log_pass log_fail log_skip log_section log_debug
export -f check_prerequisites
export -f wait_for_intent_phase create_intent_via_rag delete_intent
export -f measure_rag_latency validate_json resource_exists
export -f increment_pass increment_fail increment_skip
export -f get_total_tests get_pass_rate
export -f register_cleanup_intent cleanup_test_intents
export -f print_test_summary
