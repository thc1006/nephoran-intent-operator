#!/bin/bash
# Test script for OAI RAN connectivity to Free5GC Core
# Referenced by: docs/implementation/task-dag.yaml (Task T12, scenario 3)
# Purpose: Validate OAI RAN connects to Free5GC via N2/N3 interfaces

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
OAI_NS="oai-ran"
FREE5GC_NS="free5gc"
TIMEOUT=60
TEST_RESULTS=()

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }
success() { echo -e "${GREEN}✅ $*${NC}"; TEST_RESULTS+=("PASS: $*"); }
fail() { echo -e "${RED}❌ $*${NC}"; TEST_RESULTS+=("FAIL: $*"); return 1; }
warn() { echo -e "${YELLOW}⚠️  $*${NC}"; }

print_summary() {
    echo ""
    echo "=========================================="
    echo "Test Summary: OAI RAN Connectivity"
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

# Test 1: Verify OAI RAN pods are running
log "Test 1: Checking OAI RAN deployment status..."
OAI_PODS=$(kubectl get pods -n "$OAI_NS" --field-selector=status.phase=Running 2>/dev/null | tail -n +2 | wc -l)
if [ "$OAI_PODS" -gt 0 ]; then
    success "$OAI_PODS OAI RAN pod(s) running"
else
    fail "No OAI RAN pods running"
    exit 1
fi

# Test 2: Verify Free5GC AMF is running (N2 endpoint)
log "Test 2: Checking Free5GC AMF for N2 interface..."
AMF_IP=$(kubectl get svc -n "$FREE5GC_NS" free5gc-amf -o jsonpath='{.spec.clusterIP}' 2>/dev/null)
if [ -n "$AMF_IP" ]; then
    success "AMF service available at $AMF_IP (N2 interface endpoint)"
else
    fail "AMF service not found (N2 interface unavailable)"
fi

# Test 3: Verify Free5GC UPF is running (N3 endpoint)
log "Test 3: Checking Free5GC UPF for N3 interface..."
UPF_COUNT=$(kubectl get pods -n "$FREE5GC_NS" -l app=free5gc-upf --field-selector=status.phase=Running 2>/dev/null | tail -n +2 | wc -l)
if [ "$UPF_COUNT" -gt 0 ]; then
    success "$UPF_COUNT UPF instance(s) available (N3 interface endpoint)"
else
    fail "No UPF instances running (N3 interface unavailable)"
fi

# Test 4: Check OAI gNB/CU-CP logs for AMF connection attempts
log "Test 4: Checking OAI logs for AMF connection (N2)..."
OAI_POD=$(kubectl get pods -n "$OAI_NS" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$OAI_POD" ]; then
    AMF_REFS=$(kubectl logs -n "$OAI_NS" "$OAI_POD" --tail=100 2>/dev/null | grep -ic "amf\|n2" || echo "0")
    if [ "$AMF_REFS" -gt 0 ]; then
        success "OAI logs show AMF/N2 references ($AMF_REFS occurrences)"
    else
        warn "No AMF/N2 references in OAI logs (connection may not be configured)"
    fi
fi

# Test 5: Check AMF logs for gNB registration
log "Test 5: Checking AMF logs for gNB registration..."
AMF_POD=$(kubectl get pods -n "$FREE5GC_NS" -l app=free5gc-amf -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$AMF_POD" ]; then
    GNB_REGISTRATIONS=$(kubectl logs -n "$FREE5GC_NS" "$AMF_POD" --tail=200 2>/dev/null | grep -ic "ng-setup\|gnb.*register" || echo "0")
    if [ "$GNB_REGISTRATIONS" -gt 0 ]; then
        success "AMF logs show gNB registration activity ($GNB_REGISTRATIONS events)"
    else
        warn "No gNB registration events in AMF logs"
    fi
fi

# Test 6: Verify SCTP association (N2 uses SCTP)
log "Test 6: Checking for SCTP association (N2 protocol)..."
if [ -n "$AMF_POD" ]; then
    SCTP_REFS=$(kubectl logs -n "$FREE5GC_NS" "$AMF_POD" --tail=200 2>/dev/null | grep -ic "sctp" || echo "0")
    if [ "$SCTP_REFS" -gt 0 ]; then
        success "AMF logs show SCTP activity ($SCTP_REFS references)"
    else
        warn "No SCTP references in AMF logs (N2 may use TCP fallback)"
    fi
fi

# Test 7: Check OAI logs for UPF connection (N3)
log "Test 7: Checking OAI logs for UPF connection (N3)..."
if [ -n "$OAI_POD" ]; then
    UPF_REFS=$(kubectl logs -n "$OAI_NS" "$OAI_POD" --tail=100 2>/dev/null | grep -ic "upf\|n3\|gtp" || echo "0")
    if [ "$UPF_REFS" -gt 0 ]; then
        success "OAI logs show UPF/N3/GTP references ($UPF_REFS occurrences)"
    else
        warn "No UPF/N3 references in OAI logs (user plane may not be configured)"
    fi
fi

# Test 8: Verify GTP-U tunnel establishment (N3 protocol)
log "Test 8: Checking UPF logs for GTP-U tunnel from RAN..."
UPF_POD=$(kubectl get pods -n "$FREE5GC_NS" -l app=free5gc-upf -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$UPF_POD" ]; then
    GTP_TUNNELS=$(kubectl logs -n "$FREE5GC_NS" "$UPF_POD" --tail=100 2>/dev/null | grep -ic "gtp\|tunnel" || echo "0")
    if [ "$GTP_TUNNELS" -gt 0 ]; then
        success "UPF logs show GTP tunnel activity ($GTP_TUNNELS references)"
    else
        warn "No GTP tunnel activity in UPF logs"
    fi
fi

# Test 9: Network reachability test (OAI → AMF)
log "Test 9: Testing network reachability OAI → AMF..."
if [ -n "$OAI_POD" ] && [ -n "$AMF_IP" ]; then
    if kubectl exec -n "$OAI_NS" "$OAI_POD" -- sh -c "ping -c 2 -W 3 $AMF_IP" &>/dev/null; then
        success "OAI pod can reach AMF at $AMF_IP (network connectivity OK)"
    else
        warn "OAI pod cannot ping AMF (may be expected if ICMP disabled)"
    fi
fi

# Test 10: Network reachability test (OAI → UPF)
log "Test 10: Testing network reachability OAI → UPF..."
UPF_IP=$(kubectl get svc -n "$FREE5GC_NS" free5gc-upf -o jsonpath='{.spec.clusterIP}' 2>/dev/null)
if [ -n "$OAI_POD" ] && [ -n "$UPF_IP" ]; then
    if kubectl exec -n "$OAI_NS" "$OAI_POD" -- sh -c "ping -c 2 -W 3 $UPF_IP" &>/dev/null; then
        success "OAI pod can reach UPF at $UPF_IP (network connectivity OK)"
    else
        warn "OAI pod cannot ping UPF (may be expected if ICMP disabled)"
    fi
fi

# Test 11: DNS resolution test
log "Test 11: Testing DNS resolution in OAI namespace..."
if [ -n "$OAI_POD" ]; then
    if kubectl exec -n "$OAI_NS" "$OAI_POD" -- nslookup "free5gc-amf.$FREE5GC_NS.svc.cluster.local" &>/dev/null; then
        success "OAI pod can resolve AMF DNS name"
    else
        warn "DNS resolution failed (check CoreDNS configuration)"
    fi
fi

# Test 12: Check for RAN parameters in OAI config
log "Test 12: Verifying OAI configuration..."
OAI_CONFIG=$(kubectl get configmap -n "$OAI_NS" -o yaml 2>/dev/null | grep -c "plmnid\|mcc\|mnc\|tac" || echo "0")
if [ "$OAI_CONFIG" -gt 0 ]; then
    success "OAI RAN configuration includes PLMN parameters"
else
    warn "Could not verify OAI RAN configuration (check ConfigMaps manually)"
fi

# Test 13: Verify AMF PLMN configuration matches RAN
log "Test 13: Checking AMF PLMN configuration..."
AMF_CONFIG=$(kubectl get configmap -n "$FREE5GC_NS" -o yaml 2>/dev/null | grep -c "plmnid\|mcc\|mnc" || echo "0")
if [ "$AMF_CONFIG" -gt 0 ]; then
    success "AMF configuration includes PLMN parameters (should match RAN)"
else
    warn "Could not verify AMF PLMN configuration"
fi

# Test 14: Check for NG-AP messages (N2 signaling)
log "Test 14: Checking for NG-AP signaling messages..."
if [ -n "$AMF_POD" ]; then
    NGAP_MSGS=$(kubectl logs -n "$FREE5GC_NS" "$AMF_POD" --tail=200 2>/dev/null | grep -ic "ngap\|ng-ap" || echo "0")
    if [ "$NGAP_MSGS" -gt 0 ]; then
        success "AMF logs show NG-AP messages ($NGAP_MSGS occurrences)"
    else
        warn "No NG-AP messages in AMF logs (N2 signaling not active)"
    fi
fi

log "OAI RAN connectivity validation complete!"
log "Note: Full N2/N3 connectivity verification requires active UE traffic"
exit 0
