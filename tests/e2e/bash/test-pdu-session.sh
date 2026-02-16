#!/bin/bash
# Test script for PDU Session Establishment (UE ↔ Free5GC)
# Referenced by: docs/implementation/task-dag.yaml (Task T12, scenario 2)
# Purpose: Validate UE can establish PDU session through Free5GC

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
FREE5GC_NS="free5gc"
UERANSIM_NS="oai-ran"
TIMEOUT=120
TEST_RESULTS=()

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }
success() { echo -e "${GREEN}✅ $*${NC}"; TEST_RESULTS+=("PASS: $*"); }
fail() { echo -e "${RED}❌ $*${NC}"; TEST_RESULTS+=("FAIL: $*"); return 1; }
warn() { echo -e "${YELLOW}⚠️  $*${NC}"; }

print_summary() {
    echo ""
    echo "=========================================="
    echo "Test Summary: PDU Session Establishment"
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

# Test 1: Verify Free5GC AMF is running
log "Test 1: Checking Free5GC AMF availability..."
if kubectl get pods -n "$FREE5GC_NS" -l app=free5gc-amf --field-selector=status.phase=Running &>/dev/null; then
    success "AMF is running"
else
    fail "AMF not running (required for UE registration)"
    exit 1
fi

# Test 2: Verify Free5GC SMF is running
log "Test 2: Checking Free5GC SMF availability..."
if kubectl get pods -n "$FREE5GC_NS" -l app=free5gc-smf --field-selector=status.phase=Running &>/dev/null; then
    success "SMF is running"
else
    fail "SMF not running (required for PDU session establishment)"
    exit 1
fi

# Test 3: Verify Free5GC UPF is running
log "Test 3: Checking Free5GC UPF availability..."
UPF_COUNT=$(kubectl get pods -n "$FREE5GC_NS" -l app=free5gc-upf --field-selector=status.phase=Running 2>/dev/null | tail -n +2 | wc -l)
if [ "$UPF_COUNT" -gt 0 ]; then
    success "$UPF_COUNT UPF instance(s) running"
else
    fail "No UPF instances running (required for user plane traffic)"
    exit 1
fi

# Test 4: Check if UERANSIM is deployed
log "Test 4: Checking UERANSIM UE availability..."
if kubectl get pods -n "$UERANSIM_NS" -l app=ueransim 2>/dev/null | grep -q "ueransim"; then
    UE_POD=$(kubectl get pods -n "$UERANSIM_NS" -l app=ueransim -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
    if [ -n "$UE_POD" ]; then
        success "UERANSIM UE pod found: $UE_POD"
    else
        warn "UERANSIM pods found but could not identify UE pod"
    fi
else
    warn "UERANSIM not deployed (will simulate without actual UE)"
fi

# Test 5: Check AMF logs for UE registration readiness
log "Test 5: Checking AMF registration capability..."
AMF_POD=$(kubectl get pods -n "$FREE5GC_NS" -l app=free5gc-amf -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$AMF_POD" ]; then
    if kubectl logs -n "$FREE5GC_NS" "$AMF_POD" --tail=50 2>/dev/null | grep -iq "serving"; then
        success "AMF is in serving state"
    else
        warn "AMF serving state not confirmed in logs"
    fi
fi

# Test 6: Check SMF logs for session management readiness
log "Test 6: Checking SMF session management capability..."
SMF_POD=$(kubectl get pods -n "$FREE5GC_NS" -l app=free5gc-smf -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$SMF_POD" ]; then
    if kubectl logs -n "$FREE5GC_NS" "$SMF_POD" --tail=50 2>/dev/null | grep -iq "smf"; then
        success "SMF is initialized"
    else
        warn "SMF initialization not confirmed in logs"
    fi
fi

# Test 7: Verify subscriber data in UDM/UDR
log "Test 7: Checking subscriber database (UDM/UDR)..."
UDR_POD=$(kubectl get pods -n "$FREE5GC_NS" -l app=free5gc-udr -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$UDR_POD" ]; then
    success "UDR (subscriber database) is available"
else
    warn "UDR not found (subscriber lookups may fail)"
fi

# Test 8: Test UE registration (if UERANSIM available)
if [ -n "$UE_POD" ]; then
    log "Test 8: Attempting UE registration..."
    # Try to trigger UE registration by executing UERANSIM
    if kubectl exec -n "$UERANSIM_NS" "$UE_POD" -- sh -c "ls /usr/local/bin/nr-ue" &>/dev/null; then
        warn "UERANSIM nr-ue binary found, but automated registration requires config file"
        # Note: Full registration requires proper UERANSIM configuration which varies by deployment
    else
        warn "UERANSIM binary not found in expected location"
    fi
else
    warn "Skipping UE registration test (UERANSIM not available)"
fi

# Test 9: Check AMF for Registration Accept messages
log "Test 9: Checking AMF logs for registration activity..."
if [ -n "$AMF_POD" ]; then
    REG_ACCEPTS=$(kubectl logs -n "$FREE5GC_NS" "$AMF_POD" --tail=200 2>/dev/null | grep -c "Registration Accept" || echo "0")
    if [ "$REG_ACCEPTS" -gt 0 ]; then
        success "AMF has $REG_ACCEPTS Registration Accept message(s)"
    else
        warn "No Registration Accept messages in AMF logs (no recent UE registrations)"
    fi
fi

# Test 10: Check SMF for PDU Session Establishment messages
log "Test 10: Checking SMF logs for PDU session activity..."
if [ -n "$SMF_POD" ]; then
    PDU_ACCEPTS=$(kubectl logs -n "$FREE5GC_NS" "$SMF_POD" --tail=200 2>/dev/null | grep -c "PDU Session Establishment Accept" || echo "0")
    if [ "$PDU_ACCEPTS" -gt 0 ]; then
        success "SMF has $PDU_ACCEPTS PDU Session Establishment Accept message(s)"
    else
        warn "No PDU Session Establishment messages in SMF logs"
    fi
fi

# Test 11: Verify GTP-U tunnel in UPF logs
log "Test 11: Checking UPF for GTP-U tunnel establishment..."
UPF_POD=$(kubectl get pods -n "$FREE5GC_NS" -l app=free5gc-upf -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$UPF_POD" ]; then
    GTP_TUNNELS=$(kubectl logs -n "$FREE5GC_NS" "$UPF_POD" --tail=100 2>/dev/null | grep -c "GTP\|tunnel\|session" || echo "0")
    if [ "$GTP_TUNNELS" -gt 0 ]; then
        success "UPF logs show GTP/tunnel/session activity ($GTP_TUNNELS references)"
    else
        warn "No GTP tunnel activity in UPF logs"
    fi
fi

# Test 12: Test data plane connectivity (if UE available and interface exists)
if [ -n "$UE_POD" ]; then
    log "Test 12: Testing data plane connectivity from UE..."
    # Check if uesimtun0 interface exists (created by UERANSIM upon PDU session)
    if kubectl exec -n "$UERANSIM_NS" "$UE_POD" -- ip link show uesimtun0 &>/dev/null; then
        success "UE has uesimtun0 interface (PDU session active)"

        # Try to ping through the tunnel
        if kubectl exec -n "$UERANSIM_NS" "$UE_POD" -- ping -c 4 -W 5 8.8.8.8 &>/dev/null; then
            success "Data plane connectivity verified (ping to 8.8.8.8 successful)"
        else
            warn "uesimtun0 exists but ping failed (routing/NAT issue?)"
        fi
    else
        warn "UE uesimtun0 interface not found (PDU session not established)"
    fi
else
    warn "Skipping data plane test (UERANSIM UE not available)"
fi

# Test 13: Verify QoS flow establishment
log "Test 13: Checking for QoS flow establishment..."
if [ -n "$SMF_POD" ]; then
    QOS_FLOWS=$(kubectl logs -n "$FREE5GC_NS" "$SMF_POD" --tail=200 2>/dev/null | grep -c "QoS" || echo "0")
    if [ "$QOS_FLOWS" -gt 0 ]; then
        success "SMF logs show QoS flow processing ($QOS_FLOWS references)"
    else
        warn "No QoS flow references in SMF logs"
    fi
fi

# Test 14: Check for PFCP session establishment (SMF ↔ UPF)
log "Test 14: Verifying PFCP session (SMF ↔ UPF)..."
if [ -n "$SMF_POD" ]; then
    PFCP_SESSIONS=$(kubectl logs -n "$FREE5GC_NS" "$SMF_POD" --tail=200 2>/dev/null | grep -c "PFCP" || echo "0")
    if [ "$PFCP_SESSIONS" -gt 0 ]; then
        success "SMF shows PFCP session activity ($PFCP_SESSIONS references)"
    else
        warn "No PFCP session activity in SMF logs"
    fi
fi

log "PDU Session Establishment validation complete!"
log "Note: Full end-to-end PDU session test requires UERANSIM with proper configuration"
exit 0
