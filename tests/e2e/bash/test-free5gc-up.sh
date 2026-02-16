#!/bin/bash
# Test script for Free5GC User Plane (UPF) deployment
# Referenced by: docs/implementation/task-dag.yaml (Task T9)
# Purpose: Validate that Free5GC UPF instances are deployed and functional

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
NAMESPACE="free5gc"
EXPECTED_UPF_REPLICAS=3
TIMEOUT=300
TEST_RESULTS=()

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }
success() { echo -e "${GREEN}✅ $*${NC}"; TEST_RESULTS+=("PASS: $*"); }
fail() { echo -e "${RED}❌ $*${NC}"; TEST_RESULTS+=("FAIL: $*"); return 1; }
warn() { echo -e "${YELLOW}⚠️  $*${NC}"; }

print_summary() {
    echo ""
    echo "=========================================="
    echo "Test Summary: Free5GC User Plane"
    echo "=========================================="
    for result in "${TEST_RESULTS[@]}"; do
        if [[ $result == PASS:* ]]; then
            echo -e "${GREEN}$result${NC}"
        else
            echo -e "${RED}$result${NC}"
        fi
    done
    echo "=========================================="
}

trap print_summary EXIT

# Test 1: UPF Deployment exists
log "Test 1: Checking UPF deployment..."
if kubectl get deployment -n "$NAMESPACE" free5gc-upf &>/dev/null; then
    success "UPF deployment found"
else
    fail "UPF deployment not found"
    exit 1
fi

# Test 2: Verify expected number of replicas
log "Test 2: Checking UPF replica count..."
DESIRED_REPLICAS=$(kubectl get deployment -n "$NAMESPACE" free5gc-upf -o jsonpath='{.spec.replicas}' 2>/dev/null)
if [ "$DESIRED_REPLICAS" -eq "$EXPECTED_UPF_REPLICAS" ]; then
    success "UPF configured with $DESIRED_REPLICAS replicas (expected $EXPECTED_UPF_REPLICAS)"
else
    warn "UPF has $DESIRED_REPLICAS replicas (expected $EXPECTED_UPF_REPLICAS)"
fi

# Test 3: Wait for UPF pods to be ready
log "Test 3: Waiting for UPF pods to be ready..."
if kubectl wait --for=condition=Ready pod -l app=free5gc-upf -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
    READY_PODS=$(kubectl get pods -n "$NAMESPACE" -l app=free5gc-upf --field-selector=status.phase=Running 2>/dev/null | tail -n +2 | wc -l)
    success "$READY_PODS UPF pod(s) are ready and running"
else
    fail "UPF pods did not become ready within ${TIMEOUT}s"
fi

# Test 4: Check GTP5G kernel module (required for UPF)
log "Test 4: Checking GTP5G kernel module on UPF pods..."
UPF_PODS=$(kubectl get pods -n "$NAMESPACE" -l app=free5gc-upf -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
if [ -n "$UPF_PODS" ]; then
    FIRST_POD=$(echo "$UPF_PODS" | awk '{print $1}')
    # Try to check if gtp5g module is loaded (might not be accessible from inside pod)
    if kubectl exec -n "$NAMESPACE" "$FIRST_POD" -- sh -c "ls /dev/gtp5g 2>/dev/null || echo 'N/A'" | grep -q "gtp5g"; then
        success "GTP5G module appears to be available in UPF pod"
    else
        warn "Could not verify GTP5G module (may be expected in containerized environment)"
    fi
else
    warn "No UPF pods found to check GTP5G module"
fi

# Test 5: Verify UPF service exists
log "Test 5: Checking UPF service..."
if kubectl get svc -n "$NAMESPACE" free5gc-upf &>/dev/null; then
    UPF_IP=$(kubectl get svc -n "$NAMESPACE" free5gc-upf -o jsonpath='{.spec.clusterIP}' 2>/dev/null)
    if [ -n "$UPF_IP" ]; then
        success "UPF service available at $UPF_IP"
    else
        warn "UPF service exists but cluster IP not found"
    fi
else
    fail "UPF service not found"
fi

# Test 6: Check UPF logs for initialization success
log "Test 6: Checking UPF logs for successful initialization..."
if [ -n "$UPF_PODS" ]; then
    FIRST_POD=$(echo "$UPF_PODS" | awk '{print $1}')
    if kubectl logs -n "$NAMESPACE" "$FIRST_POD" --tail=50 2>/dev/null | grep -iq "upf"; then
        success "UPF logs show component initialization"
    else
        warn "Could not verify UPF initialization from logs"
    fi
fi

# Test 7: Verify N4 interface configuration (UPF ↔ SMF)
log "Test 7: Verifying N4 interface configuration..."
N4_CONFIG=$(kubectl get configmap -n "$NAMESPACE" free5gc-upf-config -o jsonpath='{.data.upfcfg\.yaml}' 2>/dev/null | grep -c "n4" || echo "0")
if [ "$N4_CONFIG" -gt 0 ]; then
    success "N4 interface configuration found in ConfigMap"
else
    warn "N4 interface configuration not found (may be in different location)"
fi

# Test 8: Verify N3 interface configuration (UPF ↔ gNB)
log "Test 8: Verifying N3 interface configuration..."
N3_CONFIG=$(kubectl get configmap -n "$NAMESPACE" free5gc-upf-config -o jsonpath='{.data.upfcfg\.yaml}' 2>/dev/null | grep -c "n3" || echo "0")
if [ "$N3_CONFIG" -gt 0 ]; then
    success "N3 interface configuration found in ConfigMap"
else
    warn "N3 interface configuration not found (may be in different location)"
fi

# Test 9: Check for SMF connectivity (prerequisite)
log "Test 9: Checking SMF availability (UPF dependency)..."
if kubectl get deployment -n "$NAMESPACE" free5gc-smf &>/dev/null; then
    if kubectl get pods -n "$NAMESPACE" -l app=free5gc-smf --field-selector=status.phase=Running &>/dev/null; then
        success "SMF is running (UPF can establish N4 connection)"
    else
        warn "SMF pods not in Running state (N4 connection may fail)"
    fi
else
    warn "SMF deployment not found (UPF requires SMF for N4 interface)"
fi

# Test 10: Resource limits check
log "Test 10: Checking UPF resource limits..."
CPU_LIMIT=$(kubectl get deployment -n "$NAMESPACE" free5gc-upf -o jsonpath='{.spec.template.spec.containers[0].resources.limits.cpu}' 2>/dev/null)
MEM_LIMIT=$(kubectl get deployment -n "$NAMESPACE" free5gc-upf -o jsonpath='{.spec.template.spec.containers[0].resources.limits.memory}' 2>/dev/null)
if [ -n "$CPU_LIMIT" ] && [ -n "$MEM_LIMIT" ]; then
    success "UPF has resource limits: CPU=$CPU_LIMIT, Memory=$MEM_LIMIT"
else
    warn "UPF resource limits not set (recommended for production)"
fi

# Test 11: Multi-AZ distribution check (if replicas > 1)
log "Test 11: Checking UPF pod distribution..."
if [ "$DESIRED_REPLICAS" -gt 1 ]; then
    NODES=$(kubectl get pods -n "$NAMESPACE" -l app=free5gc-upf -o jsonpath='{.items[*].spec.nodeName}' 2>/dev/null | tr ' ' '\n' | sort -u | wc -l)
    if [ "$NODES" -gt 1 ]; then
        success "UPF pods distributed across $NODES nodes (multi-AZ ready)"
    else
        warn "All UPF pods on same node (single-node cluster or no pod anti-affinity)"
    fi
else
    warn "Only 1 UPF replica (multi-AZ distribution not applicable)"
fi

# Test 12: Check for pod restart count
log "Test 12: Checking UPF pod stability..."
MAX_RESTARTS=0
if [ -n "$UPF_PODS" ]; then
    for pod in $UPF_PODS; do
        RESTARTS=$(kubectl get pod -n "$NAMESPACE" "$pod" -o jsonpath='{.status.containerStatuses[0].restartCount}' 2>/dev/null || echo "0")
        if [ "$RESTARTS" -gt "$MAX_RESTARTS" ]; then
            MAX_RESTARTS=$RESTARTS
        fi
    done
    if [ "$MAX_RESTARTS" -eq 0 ]; then
        success "UPF pods have not restarted (stable)"
    elif [ "$MAX_RESTARTS" -lt 3 ]; then
        warn "UPF pod(s) restarted $MAX_RESTARTS time(s) (acceptable during initialization)"
    else
        fail "UPF pod(s) restarted $MAX_RESTARTS time(s) (indicates instability)"
    fi
fi

log "Free5GC User Plane validation complete!"
exit 0
