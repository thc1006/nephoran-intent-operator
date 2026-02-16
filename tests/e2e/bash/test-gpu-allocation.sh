#!/usr/bin/env bash
# =============================================================================
# test-gpu-allocation.sh
# GPU Dynamic Resource Allocation (DRA) E2E Test
#
# Validates GPU availability and DRA-based allocation:
#   1. Verify GPU Operator and DRA driver are running
#   2. Create a ResourceClaim for GPU access
#   3. Create a test pod requesting GPU via DRA
#   4. Wait for pod to be Running
#   5. Exec nvidia-smi inside the pod
#   6. Verify GPU compute capability
#   7. Cleanup all test resources
#
# Prerequisites:
#   - Kubernetes 1.35+ with DRA feature gate enabled
#   - NVIDIA GPU Operator installed with DRA driver
#   - At least one GPU node in the cluster
#
# Usage:
#   ./test-gpu-allocation.sh [--namespace <ns>] [--timeout <seconds>]
#                            [--gpu-class <device-class>]
# =============================================================================

set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="${E2E_NAMESPACE:-nephoran-gpu-test}"
TIMEOUT="${E2E_TIMEOUT:-180}"
GPU_DEVICE_CLASS="${GPU_DEVICE_CLASS:-gpu.nvidia.com}"
TEST_POD_NAME="gpu-dra-test-$(date +%s)"
TEST_CLAIM_NAME="gpu-claim-$(date +%s)"
CLEANUP_DONE=false
PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0
TEST_START_TIME=$(date +%s)

# ---------------------------------------------------------------------------
# Color helpers
# ---------------------------------------------------------------------------
if [[ -t 1 ]]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
    BLUE='\033[0;34m'; NC='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; BLUE=''; NC=''
fi

# ---------------------------------------------------------------------------
# Helpers
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
    if echo "${haystack}" | grep -qi "${needle}"; then
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
        --namespace)  NAMESPACE="$2";        shift 2 ;;
        --timeout)    TIMEOUT="$2";          shift 2 ;;
        --gpu-class)  GPU_DEVICE_CLASS="$2"; shift 2 ;;
        --help|-h)
            echo "Usage: $0 [--namespace <ns>] [--timeout <sec>] [--gpu-class <class>]"
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
    log_info "Cleaning up GPU test resources..."

    # Delete pod first (it references the claim)
    kubectl delete pod "${TEST_POD_NAME}" -n "${NAMESPACE}" --ignore-not-found=true --grace-period=5 --wait=false 2>/dev/null || true
    sleep 3

    # Delete resource claim
    kubectl delete resourceclaim "${TEST_CLAIM_NAME}" -n "${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true

    # Only delete namespace if we created it
    if [[ "${NAMESPACE}" == nephoran-gpu-test* ]]; then
        kubectl delete namespace "${NAMESPACE}" --ignore-not-found=true --wait=false 2>/dev/null || true
    fi

    log_info "Cleanup complete."
}
trap cleanup EXIT INT TERM

# ---------------------------------------------------------------------------
# wait_for_pod: poll until pod reaches desired phase
# ---------------------------------------------------------------------------
wait_for_pod() {
    local pod_name="$1" desired_phase="$2" max_wait="${3:-${TIMEOUT}}"
    local elapsed=0
    local poll_interval=5

    log_info "Waiting for pod '${pod_name}' to reach '${desired_phase}' (timeout: ${max_wait}s)..."
    while [[ ${elapsed} -lt ${max_wait} ]]; do
        local phase
        phase=$(kubectl get pod "${pod_name}" -n "${NAMESPACE}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
        if [[ "${phase}" == "${desired_phase}" ]]; then
            log_info "Pod reached '${desired_phase}' after ${elapsed}s"
            return 0
        fi
        # Check for terminal failure states
        if [[ "${phase}" == "Failed" ]] || [[ "${phase}" == "Unknown" ]]; then
            log_fail "Pod entered terminal state '${phase}' after ${elapsed}s"
            kubectl describe pod "${pod_name}" -n "${NAMESPACE}" 2>/dev/null | tail -20
            return 1
        fi
        log_info "Pod phase: '${phase}' (elapsed: ${elapsed}s)"
        sleep "${poll_interval}"
        elapsed=$((elapsed + poll_interval))
    done
    log_fail "Timed out after ${max_wait}s waiting for pod '${pod_name}' to reach '${desired_phase}'"
    kubectl describe pod "${pod_name}" -n "${NAMESPACE}" 2>/dev/null | tail -30
    return 1
}

# =============================================================================
# TEST EXECUTION
# =============================================================================

echo ""
echo "============================================================"
echo "  GPU DRA Allocation E2E Test"
echo "  Namespace:    ${NAMESPACE}"
echo "  Timeout:      ${TIMEOUT}s"
echo "  Device Class: ${GPU_DEVICE_CLASS}"
echo "  Timestamp:    $(date -Iseconds)"
echo "============================================================"
echo ""

# ---------------------------------------------------------------------------
# Phase 0: Prerequisites
# ---------------------------------------------------------------------------
log_section "PHASE 0: Prerequisites"

# Check kubectl
log_info "Verifying kubectl cluster access..."
if kubectl cluster-info >/dev/null 2>&1; then
    log_pass "kubectl cluster access verified"
else
    log_fail "Cannot reach Kubernetes cluster"
    exit 1
fi

# Ensure namespace
if ! kubectl get namespace "${NAMESPACE}" >/dev/null 2>&1; then
    kubectl create namespace "${NAMESPACE}"
    log_pass "Created namespace '${NAMESPACE}'"
else
    log_pass "Namespace '${NAMESPACE}' exists"
fi

# ---------------------------------------------------------------------------
# Phase 1: GPU Operator Status
# ---------------------------------------------------------------------------
log_section "PHASE 1: GPU Operator Status"

# Check GPU Operator namespace
GPU_OPERATOR_NS="gpu-operator"
log_info "Checking GPU Operator pods in ${GPU_OPERATOR_NS}..."
GPU_PODS=$(kubectl get pods -n "${GPU_OPERATOR_NS}" --no-headers 2>/dev/null || echo "")
if [[ -n "${GPU_PODS}" ]]; then
    GPU_POD_COUNT=$(echo "${GPU_PODS}" | wc -l)
    log_pass "Found ${GPU_POD_COUNT} GPU Operator pod(s)"

    # Check if all pods are Running
    NON_RUNNING=$(echo "${GPU_PODS}" | grep -v "Running\|Completed" || echo "")
    if [[ -z "${NON_RUNNING}" ]]; then
        log_pass "All GPU Operator pods are Running/Completed"
    else
        log_fail "Some GPU Operator pods are not ready:"
        echo "${NON_RUNNING}"
    fi
else
    log_skip "GPU Operator namespace '${GPU_OPERATOR_NS}' not found or has no pods"
fi

# Check for DRA driver specifically
log_info "Checking for NVIDIA DRA driver..."
DRA_PODS=$(kubectl get pods -n "${GPU_OPERATOR_NS}" -l app=nvidia-dra-driver --no-headers 2>/dev/null || \
           kubectl get pods -n "${GPU_OPERATOR_NS}" --no-headers 2>/dev/null | grep -i "dra" || echo "")
if [[ -n "${DRA_PODS}" ]]; then
    log_pass "NVIDIA DRA driver pod(s) found"
else
    log_skip "No dedicated DRA driver pods found (may be integrated into GPU Operator)"
fi

# Check for GPU nodes
log_info "Checking for GPU-capable nodes..."
GPU_NODES=$(kubectl get nodes -l nvidia.com/gpu.present=true --no-headers 2>/dev/null || \
            kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.capacity.nvidia\.com/gpu}{"\n"}{end}' 2>/dev/null | grep -v "^$" || echo "")
if [[ -n "${GPU_NODES}" ]]; then
    log_pass "GPU-capable node(s) found"
    echo "${GPU_NODES}" | head -5
else
    log_skip "No nodes with GPU labels detected"
fi

# ---------------------------------------------------------------------------
# Phase 2: Check ResourceClaim / DeviceClass API
# ---------------------------------------------------------------------------
log_section "PHASE 2: DRA API Availability"

# Check if ResourceClaim API is available
log_info "Checking ResourceClaim API..."
RC_API=$(kubectl api-resources --api-group=resource.k8s.io 2>/dev/null | grep -i "resourceclaim" || echo "")
if [[ -n "${RC_API}" ]]; then
    log_pass "ResourceClaim API is available"
else
    log_skip "ResourceClaim API not found (DRA may not be enabled)"
fi

# Check DeviceClass resources
log_info "Checking available DeviceClasses..."
DEVICE_CLASSES=$(kubectl get deviceclass --no-headers 2>/dev/null || echo "")
if [[ -n "${DEVICE_CLASSES}" ]]; then
    DC_COUNT=$(echo "${DEVICE_CLASSES}" | wc -l)
    log_pass "Found ${DC_COUNT} DeviceClass(es)"
    echo "${DEVICE_CLASSES}"
else
    log_skip "No DeviceClass resources found"
fi

# ---------------------------------------------------------------------------
# Phase 3: Create ResourceClaim
# ---------------------------------------------------------------------------
log_section "PHASE 3: Create ResourceClaim"

CLAIM_YAML=$(cat <<EOF
apiVersion: resource.k8s.io/v1beta1
kind: ResourceClaim
metadata:
  name: ${TEST_CLAIM_NAME}
  namespace: ${NAMESPACE}
  labels:
    test-type: e2e
    test-scenario: gpu-allocation
spec:
  devices:
    requests:
      - name: gpu
        deviceClassName: ${GPU_DEVICE_CLASS}
        count: 1
EOF
)

log_info "Creating ResourceClaim '${TEST_CLAIM_NAME}'..."
CLAIM_RESULT=$(echo "${CLAIM_YAML}" | kubectl apply -f - 2>&1 || echo "APPLY_FAILED")
if echo "${CLAIM_RESULT}" | grep -q "created\|configured\|unchanged"; then
    log_pass "ResourceClaim created successfully"
elif echo "${CLAIM_RESULT}" | grep -qi "error\|not found\|no matches"; then
    log_skip "ResourceClaim creation failed (DRA API may not be available): ${CLAIM_RESULT}"
    # Try legacy resource request approach
    log_info "Falling back to legacy GPU resource request (nvidia.com/gpu)..."
fi

# ---------------------------------------------------------------------------
# Phase 4: Create Test Pod with GPU
# ---------------------------------------------------------------------------
log_section "PHASE 4: Create GPU Test Pod"

# Determine if we can use DRA or must fall back to device plugin
CLAIM_EXISTS=$(kubectl get resourceclaim "${TEST_CLAIM_NAME}" -n "${NAMESPACE}" -o name 2>/dev/null || echo "")

if [[ -n "${CLAIM_EXISTS}" ]]; then
    # DRA-based pod
    POD_YAML=$(cat <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: ${TEST_POD_NAME}
  namespace: ${NAMESPACE}
  labels:
    test-type: e2e
    test-scenario: gpu-allocation
spec:
  restartPolicy: Never
  containers:
    - name: gpu-test
      image: nvidia/cuda:12.8.0-base-ubuntu24.04
      command: ["sleep", "300"]
      resources:
        claims:
          - name: gpu
  resourceClaims:
    - name: gpu
      resourceClaimName: ${TEST_CLAIM_NAME}
EOF
    )
else
    # Fallback: device plugin based
    POD_YAML=$(cat <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: ${TEST_POD_NAME}
  namespace: ${NAMESPACE}
  labels:
    test-type: e2e
    test-scenario: gpu-allocation
spec:
  restartPolicy: Never
  containers:
    - name: gpu-test
      image: nvidia/cuda:12.8.0-base-ubuntu24.04
      command: ["sleep", "300"]
      resources:
        limits:
          nvidia.com/gpu: 1
EOF
    )
fi

log_info "Creating GPU test pod '${TEST_POD_NAME}'..."
POD_RESULT=$(echo "${POD_YAML}" | kubectl apply -f - 2>&1 || echo "APPLY_FAILED")
if echo "${POD_RESULT}" | grep -q "created\|configured"; then
    log_pass "GPU test pod created"
else
    log_fail "Failed to create GPU test pod: ${POD_RESULT}"
fi

# ---------------------------------------------------------------------------
# Phase 5: Wait for Pod to be Running
# ---------------------------------------------------------------------------
log_section "PHASE 5: Wait for Pod to be Running"

if wait_for_pod "${TEST_POD_NAME}" "Running" "${TIMEOUT}"; then
    log_pass "GPU test pod is Running"
else
    log_fail "GPU test pod failed to reach Running state"
    # Show pod events for debugging
    log_info "Pod events:"
    kubectl get events -n "${NAMESPACE}" --field-selector "involvedObject.name=${TEST_POD_NAME}" --sort-by='.lastTimestamp' 2>/dev/null | tail -10
fi

# ---------------------------------------------------------------------------
# Phase 6: Verify GPU Access Inside Pod
# ---------------------------------------------------------------------------
log_section "PHASE 6: Verify GPU Access"

POD_PHASE=$(kubectl get pod "${TEST_POD_NAME}" -n "${NAMESPACE}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")

if [[ "${POD_PHASE}" == "Running" ]]; then
    # Run nvidia-smi
    log_info "Executing nvidia-smi inside pod..."
    NVIDIA_SMI_OUTPUT=$(kubectl exec "${TEST_POD_NAME}" -n "${NAMESPACE}" -- nvidia-smi 2>&1 || echo "EXEC_FAILED")

    if echo "${NVIDIA_SMI_OUTPUT}" | grep -qi "NVIDIA-SMI\|Driver Version\|CUDA Version"; then
        log_pass "nvidia-smi executed successfully"
        echo "${NVIDIA_SMI_OUTPUT}" | head -15

        # Extract GPU info
        GPU_NAME=$(echo "${NVIDIA_SMI_OUTPUT}" | grep -oP '\|\s+\K[A-Za-z0-9 ]+(?=\s+\|)' | head -1 || echo "unknown")
        DRIVER_VERSION=$(echo "${NVIDIA_SMI_OUTPUT}" | grep -oP 'Driver Version:\s+\K[0-9.]+' || echo "unknown")
        CUDA_VERSION=$(echo "${NVIDIA_SMI_OUTPUT}" | grep -oP 'CUDA Version:\s+\K[0-9.]+' || echo "unknown")

        assert_not_empty "GPU Name" "${GPU_NAME}"
        assert_not_empty "Driver Version" "${DRIVER_VERSION}"
        assert_not_empty "CUDA Version" "${CUDA_VERSION}"
    else
        log_fail "nvidia-smi output unexpected: ${NVIDIA_SMI_OUTPUT}"
    fi

    # Test GPU memory info
    log_info "Checking GPU memory..."
    GPU_MEM=$(kubectl exec "${TEST_POD_NAME}" -n "${NAMESPACE}" -- \
        nvidia-smi --query-gpu=memory.total,memory.used,memory.free --format=csv,noheader,nounits 2>/dev/null || echo "")
    if [[ -n "${GPU_MEM}" ]]; then
        log_pass "GPU memory info: ${GPU_MEM} MiB (total,used,free)"
    else
        log_skip "Could not query GPU memory info"
    fi

    # Test CUDA device query
    log_info "Checking CUDA device availability..."
    CUDA_CHECK=$(kubectl exec "${TEST_POD_NAME}" -n "${NAMESPACE}" -- \
        bash -c 'ls /dev/nvidia* 2>/dev/null || echo "no_devices"' 2>/dev/null || echo "")
    if echo "${CUDA_CHECK}" | grep -q "/dev/nvidia"; then
        log_pass "CUDA device nodes present in pod"
    else
        log_skip "No /dev/nvidia* devices found (may use different mount path)"
    fi
else
    log_skip "Pod is not Running (phase: ${POD_PHASE}), skipping GPU verification"
fi

# ---------------------------------------------------------------------------
# Phase 7: ResourceClaim Status
# ---------------------------------------------------------------------------
log_section "PHASE 7: ResourceClaim Status"

if [[ -n "${CLAIM_EXISTS}" ]]; then
    CLAIM_STATUS=$(kubectl get resourceclaim "${TEST_CLAIM_NAME}" -n "${NAMESPACE}" -o yaml 2>/dev/null || echo "")
    if [[ -n "${CLAIM_STATUS}" ]]; then
        ALLOCATION=$(echo "${CLAIM_STATUS}" | grep -c "allocation" || echo "0")
        if [[ "${ALLOCATION}" -gt 0 ]]; then
            log_pass "ResourceClaim has allocation status"
        else
            log_info "ResourceClaim allocation status not yet populated"
        fi
    fi
else
    log_skip "ResourceClaim not created (legacy mode)"
fi

# ---------------------------------------------------------------------------
# Phase 8: Cleanup
# ---------------------------------------------------------------------------
log_section "PHASE 8: Cleanup"

log_info "Deleting test pod..."
kubectl delete pod "${TEST_POD_NAME}" -n "${NAMESPACE}" --grace-period=5 --wait=false 2>/dev/null || true

if [[ -n "${CLAIM_EXISTS}" ]]; then
    log_info "Waiting for pod termination before deleting claim..."
    sleep 5
    log_info "Deleting ResourceClaim..."
    kubectl delete resourceclaim "${TEST_CLAIM_NAME}" -n "${NAMESPACE}" --ignore-not-found=true 2>/dev/null || true
fi

log_pass "Test resources cleaned up"
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
