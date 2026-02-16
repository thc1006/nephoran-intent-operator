#!/bin/bash
# Test script for OpenAirInterface RAN deployment
# Referenced by: docs/implementation/task-dag.yaml (Task T10)
# Purpose: Validate OAI RAN components (gNB, CU-CP, CU-UP, DU) deployment

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
NAMESPACE="oai-ran"
TIMEOUT=300
TEST_RESULTS=()

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }
success() { echo -e "${GREEN}✅ $*${NC}"; TEST_RESULTS+=("PASS: $*"); }
fail() { echo -e "${RED}❌ $*${NC}"; TEST_RESULTS+=("FAIL: $*"); return 1; }
warn() { echo -e "${YELLOW}⚠️  $*${NC}"; }

print_summary() {
    echo ""
    echo "=========================================="
    echo "Test Summary: OAI RAN"
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

# Test 1: Namespace exists
log "Test 1: Checking namespace '$NAMESPACE'..."
if kubectl get namespace "$NAMESPACE" &>/dev/null; then
    success "Namespace '$NAMESPACE' exists"
else
    fail "Namespace '$NAMESPACE' not found"
    exit 1
fi

# Test 2: CU-CP (Central Unit Control Plane) deployment
log "Test 2: Checking CU-CP deployment..."
if kubectl get deployment -n "$NAMESPACE" oai-cu-cp &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/oai-cu-cp -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        success "CU-CP deployed and available"
    else
        fail "CU-CP deployment not ready within ${TIMEOUT}s"
    fi
else
    warn "CU-CP deployment not found (may use monolithic gNB)"
fi

# Test 3: CU-UP (Central Unit User Plane) deployment
log "Test 3: Checking CU-UP deployment..."
if kubectl get deployment -n "$NAMESPACE" oai-cu-up &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/oai-cu-up -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        success "CU-UP deployed and available"
    else
        fail "CU-UP deployment not ready within ${TIMEOUT}s"
    fi
else
    warn "CU-UP deployment not found (may use monolithic gNB)"
fi

# Test 4: DU (Distributed Unit) deployment
log "Test 4: Checking DU deployment..."
if kubectl get deployment -n "$NAMESPACE" oai-du &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/oai-du -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        success "DU deployed and available"
    else
        fail "DU deployment not ready within ${TIMEOUT}s"
    fi
else
    warn "DU deployment not found (may use monolithic gNB)"
fi

# Test 5: Monolithic gNB deployment (alternative to disaggregated)
log "Test 5: Checking monolithic gNB deployment..."
if kubectl get deployment -n "$NAMESPACE" oai-gnb &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/oai-gnb -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        success "Monolithic gNB deployed and available"
    else
        fail "Monolithic gNB deployment not ready within ${TIMEOUT}s"
    fi
else
    warn "Monolithic gNB not found (using disaggregated architecture)"
fi

# Test 6: Verify N2 interface configuration (gNB/AMF ↔ AMF)
log "Test 6: Checking N2 interface configuration (to AMF)..."
N2_CONFIG=$(kubectl get configmap -n "$NAMESPACE" -o yaml 2>/dev/null | grep -c "amfIpAddress\|amfHostname" || echo "0")
if [ "$N2_CONFIG" -gt 0 ]; then
    success "N2 interface (AMF) configuration found"
else
    warn "N2 interface configuration not verified (check manually)"
fi

# Test 7: Verify N3 interface configuration (gNB ↔ UPF)
log "Test 7: Checking N3 interface configuration (to UPF)..."
N3_CONFIG=$(kubectl get configmap -n "$NAMESPACE" -o yaml 2>/dev/null | grep -c "upfIpAddress\|upfHostname" || echo "0")
if [ "$N3_CONFIG" -gt 0 ]; then
    success "N3 interface (UPF) configuration found"
else
    warn "N3 interface configuration not verified (check manually)"
fi

# Test 8: Check AMF availability (prerequisite for N2)
log "Test 8: Verifying AMF availability (N2 dependency)..."
if kubectl get pods -n free5gc -l app=free5gc-amf --field-selector=status.phase=Running &>/dev/null 2>&1; then
    AMF_IP=$(kubectl get svc -n free5gc free5gc-amf -o jsonpath='{.spec.clusterIP}' 2>/dev/null)
    if [ -n "$AMF_IP" ]; then
        success "AMF available at $AMF_IP (N2 connection possible)"
    else
        success "AMF pods running (N2 connection possible)"
    fi
else
    warn "AMF not running in free5gc namespace (N2 connection will fail)"
fi

# Test 9: Check UPF availability (prerequisite for N3)
log "Test 9: Verifying UPF availability (N3 dependency)..."
if kubectl get pods -n free5gc -l app=free5gc-upf --field-selector=status.phase=Running &>/dev/null 2>&1; then
    UPF_COUNT=$(kubectl get pods -n free5gc -l app=free5gc-upf --field-selector=status.phase=Running 2>/dev/null | tail -n +2 | wc -l)
    success "$UPF_COUNT UPF instance(s) available (N3 connection possible)"
else
    warn "UPF not running in free5gc namespace (N3 connection will fail)"
fi

# Test 10: Check E2 interface configuration (to Near-RT RIC)
log "Test 10: Checking E2 interface configuration (to RIC)..."
E2_CONFIG=$(kubectl get configmap -n "$NAMESPACE" -o yaml 2>/dev/null | grep -c "ricIpAddress\|e2term\|nearRtRic" || echo "0")
if [ "$E2_CONFIG" -gt 0 ]; then
    success "E2 interface (Near-RT RIC) configuration found"
else
    warn "E2 interface configuration not found (may not be configured yet)"
fi

# Test 11: Verify OAI RAN pod health
log "Test 11: Checking OAI RAN pod health..."
UNHEALTHY_PODS=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase!=Running,status.phase!=Succeeded 2>/dev/null | tail -n +2 | wc -l)
if [ "$UNHEALTHY_PODS" -eq 0 ]; then
    TOTAL_PODS=$(kubectl get pods -n "$NAMESPACE" 2>/dev/null | tail -n +2 | wc -l)
    success "All $TOTAL_PODS OAI RAN pod(s) are healthy"
else
    fail "$UNHEALTHY_PODS OAI RAN pod(s) are not healthy"
fi

# Test 12: Check OAI RAN logs for initialization
log "Test 12: Checking OAI RAN logs for successful initialization..."
OAI_PODS=$(kubectl get pods -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)
if [ -n "$OAI_PODS" ]; then
    FIRST_POD=$(echo "$OAI_PODS" | awk '{print $1}')
    if kubectl logs -n "$NAMESPACE" "$FIRST_POD" --tail=50 2>/dev/null | grep -iq "start\|init\|ready"; then
        success "OAI RAN logs show initialization activity"
    else
        warn "Could not verify initialization from logs (check manually)"
    fi
fi

# Test 13: Verify RAN services are exposed
log "Test 13: Checking OAI RAN services..."
SERVICES=$(kubectl get svc -n "$NAMESPACE" 2>/dev/null | tail -n +2 | wc -l)
if [ "$SERVICES" -gt 0 ]; then
    success "$SERVICES OAI RAN service(s) exposed"
else
    warn "No OAI RAN services found (pods may use host networking)"
fi

# Test 14: Check for SCTP protocol support (required for N2)
log "Test 14: Verifying SCTP protocol support..."
if [ -n "$OAI_PODS" ]; then
    FIRST_POD=$(echo "$OAI_PODS" | awk '{print $1}')
    # Check if SCTP kernel module is available
    if kubectl exec -n "$NAMESPACE" "$FIRST_POD" -- sh -c "cat /proc/modules 2>/dev/null | grep -q sctp" 2>/dev/null; then
        success "SCTP kernel module available (required for N2 interface)"
    else
        warn "Could not verify SCTP support (may be in host kernel)"
    fi
fi

log "OAI RAN validation complete!"
exit 0
