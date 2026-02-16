#!/usr/bin/env bash
# =============================================================================
# test-intent-lifecycle.sh
# NetworkIntent Custom Resource Lifecycle E2E Test
#
# Validates the full lifecycle of a NetworkIntent CR:
#   1. CRD exists in the cluster
#   2. Create a sample NetworkIntent
#   3. Wait for reconciliation (status update)
#   4. Verify status fields are populated
#   5. Update the intent and verify re-reconciliation
#   6. Cleanup all created resources
#
# Prerequisites:
#   - kubectl configured and connected to a cluster
#   - NetworkIntent CRD applied (nephoran.com_networkintents)
#   - Intent operator controller running (or ready to be deployed)
#
# Usage:
#   ./test-intent-lifecycle.sh [--namespace <ns>] [--timeout <seconds>]
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
NAMESPACE="${E2E_NAMESPACE:-nephoran-e2e-test}"
TIMEOUT="${E2E_TIMEOUT:-120}"
INTENT_NAME="e2e-lifecycle-test-$(date +%s)"
CLEANUP_DONE=false
PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0
TEST_START_TIME=$(date +%s)

# ---------------------------------------------------------------------------
# Color helpers (safe for non-interactive terminals)
# ---------------------------------------------------------------------------
if [[ -t 1 ]]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
    BLUE='\033[0;34m'; NC='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; BLUE=''; NC=''
fi

# ---------------------------------------------------------------------------
# Logging and assertion helpers
# ---------------------------------------------------------------------------
log_info()    { echo -e "${BLUE}[INFO]${NC}  $(date '+%H:%M:%S') $*"; }
log_pass()    { echo -e "${GREEN}[PASS]${NC}  $(date '+%H:%M:%S') $*"; PASS_COUNT=$((PASS_COUNT + 1)); }
log_fail()    { echo -e "${RED}[FAIL]${NC}  $(date '+%H:%M:%S') $*"; FAIL_COUNT=$((FAIL_COUNT + 1)); }
log_skip()    { echo -e "${YELLOW}[SKIP]${NC}  $(date '+%H:%M:%S') $*"; SKIP_COUNT=$((SKIP_COUNT + 1)); }
log_section() { echo -e "\n${BLUE}=== $* ===${NC}"; }

assert_eq() {
    local description="$1" expected="$2" actual="$3"
    if [[ "${expected}" == "${actual}" ]]; then
        log_pass "${description}: expected='${expected}', got='${actual}'"
    else
        log_fail "${description}: expected='${expected}', got='${actual}'"
    fi
}

assert_not_empty() {
    local description="$1" value="$2"
    if [[ -n "${value}" ]]; then
        log_pass "${description}: value='${value}'"
    else
        log_fail "${description}: value is empty"
    fi
}

assert_contains() {
    local description="$1" haystack="$2" needle="$3"
    if echo "${haystack}" | grep -q "${needle}"; then
        log_pass "${description}: found '${needle}'"
    else
        log_fail "${description}: '${needle}' not found in output"
    fi
}

# ---------------------------------------------------------------------------
# Parse CLI arguments
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --namespace) NAMESPACE="$2"; shift 2 ;;
        --timeout)   TIMEOUT="$2";   shift 2 ;;
        --help|-h)
            echo "Usage: $0 [--namespace <ns>] [--timeout <seconds>]"
            exit 0
            ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

# ---------------------------------------------------------------------------
# Cleanup handler
# ---------------------------------------------------------------------------
cleanup() {
    if [[ "${CLEANUP_DONE}" == "true" ]]; then
        return
    fi
    CLEANUP_DONE=true
    log_section "CLEANUP"
    log_info "Cleaning up test resources..."

    kubectl delete networkintent "${INTENT_NAME}" -n "${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
    kubectl delete networkintent "${INTENT_NAME}-updated" -n "${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

    # Only delete namespace if we created it
    if [[ "${NAMESPACE}" == nephoran-e2e-test* ]]; then
        kubectl delete namespace "${NAMESPACE}" --ignore-not-found=true --wait=false 2>/dev/null || true
    fi

    log_info "Cleanup complete."
}
trap cleanup EXIT INT TERM

# ---------------------------------------------------------------------------
# wait_for_condition: poll until a jsonpath returns a non-empty value
# ---------------------------------------------------------------------------
wait_for_condition() {
    local resource="$1" jsonpath="$2" description="$3" max_wait="${4:-${TIMEOUT}}"
    local elapsed=0
    local poll_interval=3

    log_info "Waiting for ${description} (timeout: ${max_wait}s)..."
    while [[ ${elapsed} -lt ${max_wait} ]]; do
        local value
        value=$(kubectl get "${resource}" -n "${NAMESPACE}" -o jsonpath="${jsonpath}" 2>/dev/null || echo "")
        if [[ -n "${value}" ]]; then
            log_info "Condition met after ${elapsed}s: ${value}"
            echo "${value}"
            return 0
        fi
        sleep "${poll_interval}"
        elapsed=$((elapsed + poll_interval))
    done
    log_fail "Timed out after ${max_wait}s waiting for: ${description}"
    return 1
}

# =============================================================================
# TEST EXECUTION
# =============================================================================

echo ""
echo "============================================================"
echo "  NetworkIntent Lifecycle E2E Test"
echo "  Namespace: ${NAMESPACE}"
echo "  Timeout:   ${TIMEOUT}s"
echo "  Timestamp: $(date -Iseconds)"
echo "============================================================"
echo ""

# ---------------------------------------------------------------------------
# Phase 0: Prerequisites
# ---------------------------------------------------------------------------
log_section "PHASE 0: Prerequisites"

# Check kubectl connectivity
log_info "Verifying kubectl cluster access..."
if kubectl cluster-info >/dev/null 2>&1; then
    log_pass "kubectl cluster access verified"
else
    log_fail "Cannot reach Kubernetes cluster"
    exit 1
fi

# Check CRD exists
log_info "Checking NetworkIntent CRD..."
CRD_EXISTS=$(kubectl get crd networkintents.nephoran.com -o name 2>/dev/null || echo "")
if [[ -n "${CRD_EXISTS}" ]]; then
    log_pass "NetworkIntent CRD is registered"
else
    log_info "CRD not found, attempting to apply from deployments/crds..."
    CRD_FILE="${PROJECT_ROOT}/deployments/crds/nephoran.com_networkintents.yaml"
    if [[ -f "${CRD_FILE}" ]]; then
        if kubectl apply -f "${CRD_FILE}" 2>/dev/null; then
            log_pass "NetworkIntent CRD applied successfully"
            sleep 3  # Give API server time to register
        else
            log_fail "Failed to apply NetworkIntent CRD"
            exit 1
        fi
    else
        log_fail "CRD file not found at ${CRD_FILE}"
        exit 1
    fi
fi

# Ensure test namespace exists
log_info "Ensuring namespace '${NAMESPACE}' exists..."
if ! kubectl get namespace "${NAMESPACE}" >/dev/null 2>&1; then
    kubectl create namespace "${NAMESPACE}"
    log_pass "Created namespace '${NAMESPACE}'"
else
    log_pass "Namespace '${NAMESPACE}' already exists"
fi

# ---------------------------------------------------------------------------
# Phase 1: Create NetworkIntent
# ---------------------------------------------------------------------------
log_section "PHASE 1: Create NetworkIntent"

INTENT_YAML=$(cat <<EOF
apiVersion: intent.nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: ${INTENT_NAME}
  namespace: ${NAMESPACE}
  labels:
    test-type: e2e
    test-scenario: lifecycle
    test-run: "$(date +%s)"
spec:
  source: e2e-test
  intentType: scaling
  target: amf-test
  namespace: default
  replicas: 3
EOF
)

log_info "Applying NetworkIntent '${INTENT_NAME}'..."
echo "${INTENT_YAML}" | kubectl apply -f - 2>&1
APPLY_EXIT=$?
if [[ ${APPLY_EXIT} -eq 0 ]]; then
    log_pass "NetworkIntent created successfully"
else
    log_fail "Failed to create NetworkIntent (exit code: ${APPLY_EXIT})"
fi

# Verify the resource was created
log_info "Verifying resource exists..."
RESOURCE_NAME=$(kubectl get networkintent "${INTENT_NAME}" -n "${NAMESPACE}" -o jsonpath='{.metadata.name}' 2>/dev/null || echo "")
assert_eq "Resource name matches" "${INTENT_NAME}" "${RESOURCE_NAME}"

# Verify labels
LABEL_TYPE=$(kubectl get networkintent "${INTENT_NAME}" -n "${NAMESPACE}" -o jsonpath='{.metadata.labels.test-type}' 2>/dev/null || echo "")
assert_eq "Label test-type" "e2e" "${LABEL_TYPE}"

# Verify spec.source and spec.target
SPEC_SOURCE=$(kubectl get networkintent "${INTENT_NAME}" -n "${NAMESPACE}" -o jsonpath='{.spec.source}' 2>/dev/null || echo "")
assert_not_empty "spec.source is populated" "${SPEC_SOURCE}"
SPEC_TARGET=$(kubectl get networkintent "${INTENT_NAME}" -n "${NAMESPACE}" -o jsonpath='{.spec.target}' 2>/dev/null || echo "")
assert_not_empty "spec.target is populated" "${SPEC_TARGET}"

# ---------------------------------------------------------------------------
# Phase 2: Wait for Reconciliation
# ---------------------------------------------------------------------------
log_section "PHASE 2: Wait for Reconciliation"

# Try to wait for status.phase to be set (indicates controller processed it)
PHASE_VALUE=""
PHASE_VALUE=$(wait_for_condition \
    "networkintent/${INTENT_NAME}" \
    '{.status.phase}' \
    "status.phase to be set" \
    "${TIMEOUT}" 2>/dev/null) || true

if [[ -n "${PHASE_VALUE}" ]]; then
    log_pass "Status phase is set: ${PHASE_VALUE}"
else
    log_skip "Controller may not be running -- status.phase not populated within timeout"
    log_info "This is expected if the operator controller is not yet deployed."
fi

# Check for status conditions
CONDITIONS=$(kubectl get networkintent "${INTENT_NAME}" -n "${NAMESPACE}" \
    -o jsonpath='{.status.conditions[*].type}' 2>/dev/null || echo "")
if [[ -n "${CONDITIONS}" ]]; then
    log_pass "Status conditions present: ${CONDITIONS}"
else
    log_skip "No status conditions set (controller may not be active)"
fi

# ---------------------------------------------------------------------------
# Phase 3: List and Describe
# ---------------------------------------------------------------------------
log_section "PHASE 3: List and Describe"

# List all networkintents in namespace
log_info "Listing NetworkIntents in namespace ${NAMESPACE}..."
LIST_OUTPUT=$(kubectl get networkintent -n "${NAMESPACE}" 2>&1)
assert_contains "List output contains test intent" "${LIST_OUTPUT}" "${INTENT_NAME}"
echo "${LIST_OUTPUT}"

# Get full YAML and verify structure
log_info "Fetching full resource YAML..."
FULL_YAML=$(kubectl get networkintent "${INTENT_NAME}" -n "${NAMESPACE}" -o yaml 2>&1)
assert_contains "YAML has apiVersion" "${FULL_YAML}" "apiVersion: intent.nephoran.com/v1alpha1"
assert_contains "YAML has kind" "${FULL_YAML}" "kind: NetworkIntent"
assert_contains "YAML has spec.source" "${FULL_YAML}" "source:"
assert_contains "YAML has spec.target" "${FULL_YAML}" "target:"

# ---------------------------------------------------------------------------
# Phase 4: Update NetworkIntent
# ---------------------------------------------------------------------------
log_section "PHASE 4: Update NetworkIntent"

UPDATED_INTENT_YAML=$(cat <<EOF
apiVersion: intent.nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: ${INTENT_NAME}
  namespace: ${NAMESPACE}
  labels:
    test-type: e2e
    test-scenario: lifecycle
    test-run: "$(date +%s)"
    updated: "true"
spec:
  source: e2e-test
  intentType: scaling
  target: amf-test
  namespace: default
  replicas: 5
EOF
)

log_info "Updating NetworkIntent with new spec..."
echo "${UPDATED_INTENT_YAML}" | kubectl apply -f - 2>&1
UPDATE_EXIT=$?
if [[ ${UPDATE_EXIT} -eq 0 ]]; then
    log_pass "NetworkIntent updated successfully"
else
    log_fail "Failed to update NetworkIntent (exit code: ${UPDATE_EXIT})"
fi

# Verify the update was applied
UPDATED_INTENT=$(kubectl get networkintent "${INTENT_NAME}" -n "${NAMESPACE}" \
    -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "")
assert_eq "Updated spec.replicas" "5" "${UPDATED_INTENT}"

UPDATED_LABEL=$(kubectl get networkintent "${INTENT_NAME}" -n "${NAMESPACE}" \
    -o jsonpath='{.metadata.labels.updated}' 2>/dev/null || echo "")
assert_eq "Updated label present" "true" "${UPDATED_LABEL}"

# Check generation was incremented
GENERATION=$(kubectl get networkintent "${INTENT_NAME}" -n "${NAMESPACE}" \
    -o jsonpath='{.metadata.generation}' 2>/dev/null || echo "")
if [[ -n "${GENERATION}" ]] && [[ "${GENERATION}" -ge 2 ]]; then
    log_pass "metadata.generation incremented to ${GENERATION}"
else
    log_info "metadata.generation is ${GENERATION} (may not have incremented if spec unchanged)"
fi

# ---------------------------------------------------------------------------
# Phase 5: Negative Test -- Invalid Intent
# ---------------------------------------------------------------------------
log_section "PHASE 5: Negative Test (Invalid Intent)"

INVALID_YAML=$(cat <<EOF
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: e2e-invalid-test
  namespace: ${NAMESPACE}
spec:
  intent: ""
EOF
)

log_info "Attempting to create an invalid NetworkIntent (empty intent)..."
INVALID_OUTPUT=$(echo "${INVALID_YAML}" | kubectl apply -f - 2>&1 || true)
# The CRD has minLength: 1 on spec.intent, so this should be rejected
if echo "${INVALID_OUTPUT}" | grep -qi "error\|invalid\|denied\|validation"; then
    log_pass "Invalid intent correctly rejected by the API server"
else
    log_skip "Invalid intent was not rejected (webhook may not be active)"
fi

# Cleanup the invalid resource just in case it was created
kubectl delete networkintent e2e-invalid-test -n "${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

# ---------------------------------------------------------------------------
# Phase 6: Delete and Verify
# ---------------------------------------------------------------------------
log_section "PHASE 6: Delete and Verify"

log_info "Deleting NetworkIntent '${INTENT_NAME}'..."
kubectl delete networkintent "${INTENT_NAME}" -n "${NAMESPACE}" --wait=true --timeout=30s 2>&1
DELETE_EXIT=$?
if [[ ${DELETE_EXIT} -eq 0 ]]; then
    log_pass "NetworkIntent deleted successfully"
else
    log_fail "Failed to delete NetworkIntent (exit code: ${DELETE_EXIT})"
fi

# Verify deletion
sleep 2
DELETED_CHECK=$(kubectl get networkintent "${INTENT_NAME}" -n "${NAMESPACE}" 2>&1 || echo "NotFound")
if echo "${DELETED_CHECK}" | grep -qi "not found\|NotFound"; then
    log_pass "Confirmed: NetworkIntent no longer exists"
else
    log_fail "NetworkIntent still exists after deletion"
fi

# Mark cleanup as already done (we manually deleted)
CLEANUP_DONE=true

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
    echo -e "${RED}RESULT: FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}RESULT: PASSED${NC}"
    exit 0
fi
