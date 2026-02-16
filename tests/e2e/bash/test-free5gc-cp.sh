#!/bin/bash
# Test script for Free5GC Control Plane deployment
# Referenced by: docs/implementation/task-dag.yaml (Task T8)
# Purpose: Validate that all Free5GC CP NFs are deployed and functional

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
NAMESPACE="free5gc"
TIMEOUT=300
TEST_RESULTS=()

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }
success() { echo -e "${GREEN}✅ $*${NC}"; TEST_RESULTS+=("PASS: $*"); }
fail() { echo -e "${RED}❌ $*${NC}"; TEST_RESULTS+=("FAIL: $*"); return 1; }
warn() { echo -e "${YELLOW}⚠️  $*${NC}"; }

print_summary() {
    echo ""
    echo "=========================================="
    echo "Test Summary: Free5GC Control Plane"
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

# Test 2: MongoDB is running
log "Test 2: Checking MongoDB dependency..."
if kubectl get deployment -n "$NAMESPACE" mongodb &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/mongodb -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        # Verify MongoDB version
        MONGO_VERSION=$(kubectl exec -n "$NAMESPACE" deployment/mongodb -- mongosh --quiet --eval "db.version()" 2>/dev/null | tr -d '"')
        if [[ $MONGO_VERSION =~ ^7\. ]]; then
            success "MongoDB v$MONGO_VERSION is running"
        else
            warn "MongoDB version $MONGO_VERSION (expected 7.0+)"
        fi
    else
        fail "MongoDB deployment not ready"
    fi
else
    fail "MongoDB deployment not found"
fi

# Test 3: NRF (Network Repository Function) - Core dependency
log "Test 3: Checking NRF (Network Repository Function)..."
if kubectl get deployment -n "$NAMESPACE" free5gc-nrf &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/free5gc-nrf -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        # Check if NRF service is accessible
        NRF_SVC=$(kubectl get svc -n "$NAMESPACE" free5gc-nrf -o jsonpath='{.spec.clusterIP}' 2>/dev/null)
        if [ -n "$NRF_SVC" ]; then
            success "NRF deployed and service available at $NRF_SVC"
        else
            warn "NRF deployed but service IP not found"
        fi
    else
        fail "NRF deployment not ready within ${TIMEOUT}s"
    fi
else
    fail "NRF deployment not found"
fi

# Test 4: AMF (Access and Mobility Management Function)
log "Test 4: Checking AMF..."
if kubectl get deployment -n "$NAMESPACE" free5gc-amf &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/free5gc-amf -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        # Check AMF logs for successful NRF registration
        if kubectl logs -n "$NAMESPACE" deployment/free5gc-amf --tail=50 2>/dev/null | grep -q "NRF"; then
            success "AMF deployed and communicating with NRF"
        else
            warn "AMF deployed but NRF communication not verified in logs"
        fi
    else
        fail "AMF deployment not ready within ${TIMEOUT}s"
    fi
else
    fail "AMF deployment not found"
fi

# Test 5: SMF (Session Management Function)
log "Test 5: Checking SMF..."
if kubectl get deployment -n "$NAMESPACE" free5gc-smf &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/free5gc-smf -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        success "SMF deployed successfully"
    else
        fail "SMF deployment not ready within ${TIMEOUT}s"
    fi
else
    fail "SMF deployment not found"
fi

# Test 6: AUSF (Authentication Server Function)
log "Test 6: Checking AUSF..."
if kubectl get deployment -n "$NAMESPACE" free5gc-ausf &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/free5gc-ausf -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        success "AUSF deployed successfully"
    else
        fail "AUSF deployment not ready within ${TIMEOUT}s"
    fi
else
    fail "AUSF deployment not found"
fi

# Test 7: UDM (Unified Data Management)
log "Test 7: Checking UDM..."
if kubectl get deployment -n "$NAMESPACE" free5gc-udm &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/free5gc-udm -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        success "UDM deployed successfully"
    else
        fail "UDM deployment not ready within ${TIMEOUT}s"
    fi
else
    fail "UDM deployment not found"
fi

# Test 8: UDR (Unified Data Repository)
log "Test 8: Checking UDR..."
if kubectl get deployment -n "$NAMESPACE" free5gc-udr &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/free5gc-udr -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        success "UDR deployed successfully"
    else
        fail "UDR deployment not ready within ${TIMEOUT}s"
    fi
else
    fail "UDR deployment not found"
fi

# Test 9: PCF (Policy Control Function)
log "Test 9: Checking PCF..."
if kubectl get deployment -n "$NAMESPACE" free5gc-pcf &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/free5gc-pcf -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        success "PCF deployed successfully"
    else
        fail "PCF deployment not ready within ${TIMEOUT}s"
    fi
else
    fail "PCF deployment not found"
fi

# Test 10: NSSF (Network Slice Selection Function) - Optional
log "Test 10: Checking NSSF (optional)..."
if kubectl get deployment -n "$NAMESPACE" free5gc-nssf &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/free5gc-nssf -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        success "NSSF deployed successfully"
    else
        warn "NSSF deployment found but not ready"
    fi
else
    warn "NSSF deployment not found (optional NF)"
fi

# Test 11: WebUI - Management Interface
log "Test 11: Checking WebUI..."
if kubectl get deployment -n "$NAMESPACE" free5gc-webui &>/dev/null; then
    if kubectl wait --for=condition=Available deployment/free5gc-webui -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
        WEBUI_PORT=$(kubectl get svc -n "$NAMESPACE" free5gc-webui -o jsonpath='{.spec.ports[0].port}' 2>/dev/null)
        if [ -n "$WEBUI_PORT" ]; then
            success "WebUI deployed and accessible on port $WEBUI_PORT"
        else
            success "WebUI deployed"
        fi
    else
        warn "WebUI deployment not ready"
    fi
else
    warn "WebUI deployment not found"
fi

# Test 12: NF Registration with NRF
log "Test 12: Verifying NF registration with NRF..."
NRF_POD=$(kubectl get pod -n "$NAMESPACE" -l app=free5gc-nrf -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$NRF_POD" ]; then
    # Check NRF logs for successful NF registrations
    REGISTERED_NFS=$(kubectl logs -n "$NAMESPACE" "$NRF_POD" --tail=100 2>/dev/null | grep -c "Register" || echo "0")
    if [ "$REGISTERED_NFS" -gt 0 ]; then
        success "NRF shows $REGISTERED_NFS NF registration events"
    else
        warn "No NF registration events found in NRF logs"
    fi
else
    warn "Could not find NRF pod to check registrations"
fi

# Test 13: Pod health check
log "Test 13: Checking pod health status..."
UNHEALTHY_PODS=$(kubectl get pods -n "$NAMESPACE" -l 'app in (free5gc-amf,free5gc-smf,free5gc-nrf,free5gc-ausf,free5gc-udm,free5gc-udr,free5gc-pcf)' \
    --field-selector=status.phase!=Running 2>/dev/null | tail -n +2 | wc -l)
if [ "$UNHEALTHY_PODS" -eq 0 ]; then
    success "All Free5GC CP pods are healthy"
else
    fail "$UNHEALTHY_PODS Free5GC CP pod(s) are not in Running state"
fi

# Test 14: Service endpoints
log "Test 14: Verifying service endpoints..."
SERVICES=$(kubectl get svc -n "$NAMESPACE" -l 'app in (free5gc-amf,free5gc-smf,free5gc-nrf)' -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null)
SERVICE_COUNT=$(echo "$SERVICES" | wc -l)
if [ "$SERVICE_COUNT" -ge 3 ]; then
    success "$SERVICE_COUNT core services exposed (AMF, SMF, NRF)"
else
    warn "Only $SERVICE_COUNT core services found (expected at least 3)"
fi

log "Free5GC Control Plane validation complete!"
exit 0
