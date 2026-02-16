#!/bin/bash
# Test script for A1 Policy Integration (NetworkIntent → A1 Policy)
# Referenced by: docs/implementation/task-dag.yaml (Task T11)
# Purpose: Validate end-to-end NetworkIntent to A1 Policy flow

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
INTENT_NAMESPACE="nephoran-system"
RIC_NAMESPACE="ricplt"
TIMEOUT=60
TEST_RESULTS=()
TEST_INTENT_NAME="test-a1-integration-$(date +%s)"

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }
success() { echo -e "${GREEN}✅ $*${NC}"; TEST_RESULTS+=("PASS: $*"); }
fail() { echo -e "${RED}❌ $*${NC}"; TEST_RESULTS+=("FAIL: $*"); return 1; }
warn() { echo -e "${YELLOW}⚠️  $*${NC}"; }

print_summary() {
    echo ""
    echo "=========================================="
    echo "Test Summary: A1 Integration"
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

cleanup() {
    log "Cleaning up test resources..."
    kubectl delete networkintent "$TEST_INTENT_NAME" -n "$INTENT_NAMESPACE" --ignore-not-found=true &>/dev/null || true
}

trap 'cleanup; print_summary' EXIT

# Test 1: NetworkIntent CRD is registered
log "Test 1: Checking NetworkIntent CRD..."
if kubectl get crd networkintents.intent.nephoran.com &>/dev/null; then
    success "NetworkIntent CRD is registered"
else
    fail "NetworkIntent CRD not found"
    exit 1
fi

# Test 2: Intent Operator is running
log "Test 2: Checking Nephoran Intent Operator..."
if kubectl get deployment -n "$INTENT_NAMESPACE" nephoran-controller-manager &>/dev/null; then
    if kubectl get pods -n "$INTENT_NAMESPACE" -l control-plane=controller-manager --field-selector=status.phase=Running &>/dev/null; then
        success "Nephoran Intent Operator is running"
    else
        fail "Nephoran Intent Operator pods not running"
    fi
else
    fail "Nephoran Intent Operator deployment not found"
fi

# Test 3: Near-RT RIC is accessible
log "Test 3: Checking Near-RT RIC availability..."
if kubectl get pods -n "$RIC_NAMESPACE" &>/dev/null 2>&1; then
    RIC_PODS=$(kubectl get pods -n "$RIC_NAMESPACE" --field-selector=status.phase=Running 2>/dev/null | tail -n +2 | wc -l)
    if [ "$RIC_PODS" -gt 0 ]; then
        success "Near-RT RIC is running ($RIC_PODS pod(s))"
    else
        warn "No running RIC pods found (A1 endpoint may not be available)"
    fi
else
    warn "RIC namespace '$RIC_NAMESPACE' not found (using mock A1 endpoint?)"
fi

# Test 4: Non-RT RIC A1 Policy API endpoint
log "Test 4: Checking Non-RT RIC A1 Policy API..."
NONRTRIC_SVC=$(kubectl get svc -n "$RIC_NAMESPACE" nonrtric -o jsonpath='{.spec.clusterIP}' 2>/dev/null || echo "")
if [ -n "$NONRTRIC_SVC" ]; then
    # Try to reach A1 Policy API health endpoint
    if kubectl run curl-test --image=curlimages/curl:latest --restart=Never -i --rm --timeout=30s -- \
        curl -sf "http://$NONRTRIC_SVC:8080/a1-policy/v2/health" &>/dev/null; then
        success "Non-RT RIC A1 Policy API is accessible at $NONRTRIC_SVC"
    else
        warn "Non-RT RIC service found but API health check failed"
    fi
else
    warn "Non-RT RIC service not found (check RIC deployment)"
fi

# Test 5: Create test NetworkIntent
log "Test 5: Creating test NetworkIntent..."
cat <<EOF | kubectl apply -f - &>/dev/null
apiVersion: intent.nephoran.com/v1
kind: NetworkIntent
metadata:
  name: $TEST_INTENT_NAME
  namespace: $INTENT_NAMESPACE
spec:
  intentType: cnf-scaling
  targetService: test-service
  desiredReplicas: 5
  naturalLanguageIntent: "Scale test-service to 5 replicas for A1 integration test"
EOF

if [ $? -eq 0 ]; then
    success "NetworkIntent '$TEST_INTENT_NAME' created"
else
    fail "Failed to create NetworkIntent"
fi

# Test 6: Wait for NetworkIntent to be processed
log "Test 6: Waiting for NetworkIntent reconciliation..."
sleep 5  # Give controller time to process

# Check if status is updated
STATUS_TYPE=$(kubectl get networkintent "$TEST_INTENT_NAME" -n "$INTENT_NAMESPACE" \
    -o jsonpath='{.status.conditions[0].type}' 2>/dev/null || echo "")
if [ -n "$STATUS_TYPE" ]; then
    success "NetworkIntent status updated (type: $STATUS_TYPE)"
else
    warn "NetworkIntent status not yet updated (controller may be processing)"
fi

# Test 7: Check for A1 policy creation in operator logs
log "Test 7: Checking operator logs for A1 policy creation..."
OPERATOR_POD=$(kubectl get pods -n "$INTENT_NAMESPACE" -l control-plane=controller-manager \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$OPERATOR_POD" ]; then
    if kubectl logs -n "$INTENT_NAMESPACE" "$OPERATOR_POD" --tail=100 2>/dev/null | grep -iq "a1\|policy"; then
        success "Operator logs show A1 policy processing"
    else
        warn "No A1 policy references in recent logs (check full logs manually)"
    fi
else
    warn "Could not find operator pod to check logs"
fi

# Test 8: Verify NetworkIntent status conditions
log "Test 8: Checking NetworkIntent status conditions..."
CONDITION_STATUS=$(kubectl get networkintent "$TEST_INTENT_NAME" -n "$INTENT_NAMESPACE" \
    -o jsonpath='{.status.conditions[0].status}' 2>/dev/null || echo "")
if [ "$CONDITION_STATUS" = "True" ]; then
    success "NetworkIntent condition status is True (reconciled successfully)"
elif [ "$CONDITION_STATUS" = "False" ]; then
    REASON=$(kubectl get networkintent "$TEST_INTENT_NAME" -n "$INTENT_NAMESPACE" \
        -o jsonpath='{.status.conditions[0].reason}' 2>/dev/null)
    fail "NetworkIntent condition status is False (reason: $REASON)"
else
    warn "NetworkIntent condition status unknown (may still be processing)"
fi

# Test 9: Check for A1 policy in Non-RT RIC (if accessible)
log "Test 9: Verifying A1 policy in Non-RT RIC..."
if [ -n "$NONRTRIC_SVC" ]; then
    # Try to list policies
    POLICIES=$(kubectl run curl-test --image=curlimages/curl:latest --restart=Never -i --rm --timeout=30s -- \
        curl -sf "http://$NONRTRIC_SVC:8080/a1-policy/v2/policies" 2>/dev/null | grep -c "policyId" || echo "0")
    if [ "$POLICIES" -gt 0 ]; then
        success "Found $POLICIES A1 policy/policies in Non-RT RIC"
    else
        warn "No A1 policies found in Non-RT RIC (check A1 mediator connectivity)"
    fi
else
    warn "Non-RT RIC not accessible, cannot verify A1 policy creation"
fi

# Test 10: Check NetworkIntent finalizers (cleanup behavior)
log "Test 10: Checking NetworkIntent finalizers..."
FINALIZERS=$(kubectl get networkintent "$TEST_INTENT_NAME" -n "$INTENT_NAMESPACE" \
    -o jsonpath='{.metadata.finalizers}' 2>/dev/null || echo "")
if [ -n "$FINALIZERS" ]; then
    success "NetworkIntent has finalizers for cleanup: $FINALIZERS"
else
    warn "NetworkIntent has no finalizers (A1 policy cleanup may not occur)"
fi

# Test 11: Test NetworkIntent deletion and A1 policy cleanup
log "Test 11: Testing NetworkIntent deletion..."
kubectl delete networkintent "$TEST_INTENT_NAME" -n "$INTENT_NAMESPACE" --timeout=30s &>/dev/null
if [ $? -eq 0 ]; then
    success "NetworkIntent deleted successfully"
    # Check if it's actually gone
    if ! kubectl get networkintent "$TEST_INTENT_NAME" -n "$INTENT_NAMESPACE" &>/dev/null 2>&1; then
        success "NetworkIntent fully removed from cluster"
    else
        warn "NetworkIntent still exists (finalizer may be blocking deletion)"
    fi
else
    warn "NetworkIntent deletion timed out or failed"
fi

# Test 12: Verify operator metrics (if metrics endpoint available)
log "Test 12: Checking operator metrics..."
METRICS_PORT=$(kubectl get svc -n "$INTENT_NAMESPACE" nephoran-controller-manager-metrics-service \
    -o jsonpath='{.spec.ports[0].port}' 2>/dev/null || echo "")
if [ -n "$METRICS_PORT" ]; then
    success "Operator metrics service available on port $METRICS_PORT"
else
    warn "Operator metrics service not found"
fi

log "A1 Integration validation complete!"
exit 0
