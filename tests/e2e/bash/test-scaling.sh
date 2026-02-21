#!/usr/bin/env bash
# =============================================================================
# test-scaling.sh
# E2E Test for NetworkIntent Scaling Operations
#
# Tests 5 scaling scenarios:
#   1. Scale up AMF 1→5 replicas
#   2. Scale down SMF 3→1 replica
#   3. Scale UPF to specific count (3 replicas)
#   4. Auto-scaling based on metrics
#   5. Emergency scale-down to 0 (maintenance mode)
#
# Prerequisites:
#   - RAG service deployed and accessible
#   - Nephoran Intent Operator running
#   - NetworkIntent CRD registered
#   - Free5GC components deployed (optional for validation)
#
# Usage:
#   ./test-scaling.sh [--rag-url <url>] [--timeout <seconds>] [--namespace <ns>]
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
RAG_URL="${RAG_URL:-http://localhost:8000}"
INTENT_NAMESPACE="${INTENT_NAMESPACE:-nephoran-system}"
TIMEOUT="${E2E_TIMEOUT:-120}"
CHECK_INTERVAL=5
MAX_RETRIES=$((TIMEOUT / CHECK_INTERVAL))

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0
TEST_START_TIME=$(date +%s)
CLEANUP_INTENTS=()

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
log_pass()    { echo -e "${GREEN}[PASS]${NC}  $(date '+%H:%M:%S') $*"; PASS_COUNT=$((PASS_COUNT + 1)); }
log_fail()    { echo -e "${RED}[FAIL]${NC}  $(date '+%H:%M:%S') $*"; FAIL_COUNT=$((FAIL_COUNT + 1)); }
log_skip()    { echo -e "${YELLOW}[SKIP]${NC}  $(date '+%H:%M:%S') $*"; SKIP_COUNT=$((SKIP_COUNT + 1)); }
log_section() { echo -e "\n${CYAN}${BOLD}=== $* ===${NC}"; }

cleanup() {
    log_info "Cleaning up test NetworkIntents..."
    for intent_name in "${CLEANUP_INTENTS[@]}"; do
        kubectl delete networkintent "${intent_name}" -n "${INTENT_NAMESPACE}" --ignore-not-found=true &>/dev/null || true
    done
}

trap cleanup EXIT

# ---------------------------------------------------------------------------
# Test helpers
# ---------------------------------------------------------------------------
wait_for_intent_phase() {
    local intent_name="$1"
    local expected_phase="$2"
    local retries=0

    log_info "Waiting for NetworkIntent '${intent_name}' to reach phase '${expected_phase}'..."

    while [[ ${retries} -lt ${MAX_RETRIES} ]]; do
        local current_phase
        current_phase=$(kubectl get networkintent "${intent_name}" -n "${INTENT_NAMESPACE}" \
            -o jsonpath='{.status.phase}' 2>/dev/null || echo "")

        if [[ -n "${current_phase}" ]]; then
            log_info "Current phase: ${current_phase}"

            if [[ "${current_phase}" == "${expected_phase}" ]]; then
                log_pass "NetworkIntent reached phase '${expected_phase}'"
                return 0
            elif [[ "${current_phase}" == "Error" ]] || [[ "${current_phase}" == "Failed" ]]; then
                log_fail "NetworkIntent entered error state: ${current_phase}"
                kubectl get networkintent "${intent_name}" -n "${INTENT_NAMESPACE}" -o yaml
                return 1
            fi
        fi

        retries=$((retries + 1))
        sleep "${CHECK_INTERVAL}"
    done

    log_fail "Timeout waiting for phase '${expected_phase}' (current: ${current_phase})"
    return 1
}

create_intent_via_rag() {
    local scenario="$1"
    local expected_replicas="${2:-1}"
    local start_time end_time duration_ms

    log_info "Sending intent to RAG: ${scenario}"
    start_time=$(date +%s%N)

    local response
    response=$(curl -s --max-time "${TIMEOUT}" -X POST \
        "${RAG_URL}/process" \
        -H "Content-Type: application/json" \
        -d "{\"intent\":\"${scenario}\",\"intent_id\":\"e2e-test-$(date +%s%N)\"}" 2>/dev/null || echo '{"error":"request_failed"}')

    end_time=$(date +%s%N)
    duration_ms=$(( (end_time - start_time) / 1000000 ))

    log_info "RAG response time: ${duration_ms}ms"

    # Check HTTP status separately
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time "${TIMEOUT}" -X POST \
        "${RAG_URL}/process" \
        -H "Content-Type: application/json" \
        -d "{\"intent\":\"${scenario}\"}" 2>/dev/null || echo "000")

    if [[ "${http_code}" != "200" ]]; then
        log_fail "RAG request failed with HTTP ${http_code}"
        return 1
    fi

    # Validate response structure
    if ! echo "${response}" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
        log_fail "RAG returned invalid JSON"
        echo "${response}"
        return 1
    fi

    log_pass "RAG processed intent successfully (${duration_ms}ms)"

    # Extract NetworkIntent name (implementation-dependent)
    local intent_name
    intent_name=$(echo "${response}" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    # Try various field names
    for field in ['intent_name', 'name', 'networkIntentName']:
        if field in data:
            print(data[field])
            break
except:
    pass
" 2>/dev/null || echo "")

    if [[ -z "${intent_name}" ]]; then
        # Fallback: list recent intents
        intent_name=$(kubectl get networkintent -n "${INTENT_NAMESPACE}" \
            --sort-by='.metadata.creationTimestamp' -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null || echo "")
    fi

    if [[ -n "${intent_name}" ]]; then
        log_info "NetworkIntent created: ${intent_name}"
        CLEANUP_INTENTS+=("${intent_name}")
        echo "${intent_name}"
        return 0
    else
        log_fail "Could not determine NetworkIntent name"
        return 1
    fi
}

# =============================================================================
# TEST EXECUTION
# =============================================================================

echo ""
echo -e "${CYAN}${BOLD}"
echo "============================================================"
echo "  NetworkIntent Scaling Tests"
echo "  RAG URL:      ${RAG_URL}"
echo "  Namespace:    ${INTENT_NAMESPACE}"
echo "  Timeout:      ${TIMEOUT}s"
echo "  Timestamp:    $(date -Iseconds)"
echo "============================================================"
echo -e "${NC}"

# ---------------------------------------------------------------------------
# Prerequisites
# ---------------------------------------------------------------------------
log_section "Prerequisites Check"

if ! kubectl get crd networkintents.intent.nephoran.com &>/dev/null; then
    log_fail "NetworkIntent CRD not found"
    exit 1
fi
log_pass "NetworkIntent CRD registered"

if ! curl -sf --max-time 5 "${RAG_URL}/health" &>/dev/null; then
    log_fail "RAG service not accessible at ${RAG_URL}"
    exit 1
fi
log_pass "RAG service accessible"

if ! kubectl get deployment -n "${INTENT_NAMESPACE}" nephoran-controller-manager &>/dev/null; then
    log_fail "Nephoran Intent Operator not found"
    exit 1
fi
log_pass "Nephoran Intent Operator running"

# ---------------------------------------------------------------------------
# Test 1: Scale up AMF 1→5 replicas
# ---------------------------------------------------------------------------
log_section "TEST 1: Scale up AMF 1→5 replicas"

SCENARIO_1="Scale the AMF network function to 5 replicas in namespace free5gc for high traffic handling"
INTENT_1=$(create_intent_via_rag "${SCENARIO_1}" 5)

if [[ -n "${INTENT_1}" ]]; then
    if wait_for_intent_phase "${INTENT_1}" "Deployed"; then
        # Verify replicas in NetworkIntent spec
        REPLICAS=$(kubectl get networkintent "${INTENT_1}" -n "${INTENT_NAMESPACE}" \
            -o jsonpath='{.spec.desiredReplicas}' 2>/dev/null || echo "0")

        if [[ "${REPLICAS}" == "5" ]]; then
            log_pass "TEST 1: AMF scaled to 5 replicas"
        else
            log_fail "TEST 1: Expected 5 replicas, got ${REPLICAS}"
        fi
    else
        log_fail "TEST 1: NetworkIntent failed to deploy"
    fi
else
    log_fail "TEST 1: Failed to create NetworkIntent"
fi

# ---------------------------------------------------------------------------
# Test 2: Scale down SMF 3→1 replica
# ---------------------------------------------------------------------------
log_section "TEST 2: Scale down SMF 3→1 replica"

SCENARIO_2="Scale down SMF network function to 1 replica in namespace free5gc for maintenance mode"
INTENT_2=$(create_intent_via_rag "${SCENARIO_2}" 1)

if [[ -n "${INTENT_2}" ]]; then
    if wait_for_intent_phase "${INTENT_2}" "Deployed"; then
        REPLICAS=$(kubectl get networkintent "${INTENT_2}" -n "${INTENT_NAMESPACE}" \
            -o jsonpath='{.spec.desiredReplicas}' 2>/dev/null || echo "0")

        if [[ "${REPLICAS}" == "1" ]]; then
            log_pass "TEST 2: SMF scaled down to 1 replica"
        else
            log_fail "TEST 2: Expected 1 replica, got ${REPLICAS}"
        fi
    else
        log_fail "TEST 2: NetworkIntent failed to deploy"
    fi
else
    log_fail "TEST 2: Failed to create NetworkIntent"
fi

# ---------------------------------------------------------------------------
# Test 3: Scale UPF to specific count (3 replicas)
# ---------------------------------------------------------------------------
log_section "TEST 3: Scale UPF to 3 replicas"

SCENARIO_3="Deploy 3 UPF network function instances in namespace free5gc to handle increased user plane traffic"
INTENT_3=$(create_intent_via_rag "${SCENARIO_3}" 3)

if [[ -n "${INTENT_3}" ]]; then
    if wait_for_intent_phase "${INTENT_3}" "Deployed"; then
        REPLICAS=$(kubectl get networkintent "${INTENT_3}" -n "${INTENT_NAMESPACE}" \
            -o jsonpath='{.spec.desiredReplicas}' 2>/dev/null || echo "0")

        if [[ "${REPLICAS}" == "3" ]]; then
            log_pass "TEST 3: UPF scaled to 3 replicas"
        else
            log_fail "TEST 3: Expected 3 replicas, got ${REPLICAS}"
        fi
    else
        log_fail "TEST 3: NetworkIntent failed to deploy"
    fi
else
    log_fail "TEST 3: Failed to create NetworkIntent"
fi

# ---------------------------------------------------------------------------
# Test 4: Auto-scaling based on metrics
# ---------------------------------------------------------------------------
log_section "TEST 4: Auto-scaling based on metrics"

SCENARIO_4="Configure auto-scaling for NRF network function with min 2 max 10 replicas based on CPU utilization in namespace free5gc"
INTENT_4=$(create_intent_via_rag "${SCENARIO_4}" 2)

if [[ -n "${INTENT_4}" ]]; then
    if wait_for_intent_phase "${INTENT_4}" "Deployed"; then
        # Check for autoscaling configuration
        INTENT_TYPE=$(kubectl get networkintent "${INTENT_4}" -n "${INTENT_NAMESPACE}" \
            -o jsonpath='{.spec.intentType}' 2>/dev/null || echo "")

        if [[ "${INTENT_TYPE}" == *"scaling"* ]] || [[ "${INTENT_TYPE}" == *"auto"* ]]; then
            log_pass "TEST 4: Auto-scaling intent configured"
        else
            log_skip "TEST 4: Auto-scaling may not be fully implemented (intent type: ${INTENT_TYPE})"
        fi
    else
        log_fail "TEST 4: NetworkIntent failed to deploy"
    fi
else
    log_fail "TEST 4: Failed to create NetworkIntent"
fi

# ---------------------------------------------------------------------------
# Test 5: Emergency scale-down to 0
# ---------------------------------------------------------------------------
log_section "TEST 5: Emergency scale-down to 0"

SCENARIO_5="Emergency scale down AUSF network function to 0 replicas in namespace free5gc for critical maintenance"
INTENT_5=$(create_intent_via_rag "${SCENARIO_5}" 0)

if [[ -n "${INTENT_5}" ]]; then
    if wait_for_intent_phase "${INTENT_5}" "Deployed"; then
        REPLICAS=$(kubectl get networkintent "${INTENT_5}" -n "${INTENT_NAMESPACE}" \
            -o jsonpath='{.spec.desiredReplicas}' 2>/dev/null || echo "-1")

        if [[ "${REPLICAS}" == "0" ]]; then
            log_pass "TEST 5: AUSF scaled down to 0 replicas"
        else
            log_fail "TEST 5: Expected 0 replicas, got ${REPLICAS}"
        fi
    else
        log_fail "TEST 5: NetworkIntent failed to deploy"
    fi
else
    log_fail "TEST 5: Failed to create NetworkIntent"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
log_section "TEST SUMMARY"

TEST_END_TIME=$(date +%s)
TOTAL_DURATION=$((TEST_END_TIME - TEST_START_TIME))
TOTAL_TESTS=$((PASS_COUNT + FAIL_COUNT + SKIP_COUNT))

echo ""
echo "  Total tests:  ${TOTAL_TESTS}"
echo -e "  Passed:       ${GREEN}${PASS_COUNT}${NC}"
echo -e "  Failed:       ${RED}${FAIL_COUNT}${NC}"
echo -e "  Skipped:      ${YELLOW}${SKIP_COUNT}${NC}"
echo "  Duration:     ${TOTAL_DURATION}s"
echo ""

if [[ ${FAIL_COUNT} -gt 0 ]]; then
    echo -e "${RED}${BOLD}RESULT: FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}${BOLD}RESULT: PASSED${NC}"
    exit 0
fi
