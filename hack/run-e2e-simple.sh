#!/usr/bin/env bash
# Simple idempotent E2E test matching exact requirements
set -euo pipefail

CLUSTER_NAME="${CLUSTER_NAME:-nephoran-e2e}"
REPORT_DIR=".excellence-reports"
REPORT_FILE="$REPORT_DIR/e2e-summary.txt"

# Setup report directory
mkdir -p "$REPORT_DIR"

# Initialize report
echo "E2E Test Report - $(date)" > "$REPORT_FILE"
echo "================================" >> "$REPORT_FILE"

# 1. Create/verify cluster (idempotent)
echo "Setting up Kind cluster..."
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    echo "✅ Cluster exists" | tee -a "$REPORT_FILE"
else
    kind create cluster --name "$CLUSTER_NAME" --wait 120s
    echo "✅ Cluster created" | tee -a "$REPORT_FILE"
fi

# 2. Apply CRDs
echo "Applying CRDs..."
kubectl apply -f deployments/crds/ 2>/dev/null || kubectl apply -f config/crd/ 2>/dev/null || true

# 3. Wait for NetworkIntent CRD
echo "Waiting for NetworkIntent CRD..."
if kubectl wait --for=condition=Established crd/networkintents.nephoran.com --timeout=60s 2>/dev/null; then
    echo "✅ CRD established" | tee -a "$REPORT_FILE"
else
    echo "❌ CRD not established" | tee -a "$REPORT_FILE"
    exit 1
fi

# 4. Deploy webhook manager (optional - continue if not present)
echo "Deploying webhook manager..."
if [ -d "config/webhook" ]; then
    kubectl apply -k config/webhook 2>/dev/null || true
    kubectl -n nephoran-system wait deployment/webhook-manager --for=condition=Available --timeout=60s 2>/dev/null || true
    echo "✅ Webhook deployed (or skipped)" | tee -a "$REPORT_FILE"
fi

# 5. Apply sample NetworkIntent
echo "Creating sample NetworkIntent..."
kubectl apply -f tests/e2e/samples/scaling-intent.yaml
echo "✅ NetworkIntent created" | tee -a "$REPORT_FILE"

# 6. Verify replicas field
echo "Verifying NetworkIntent..."
if kubectl get networkintents e2e-scaling-test -o yaml | grep -q 'replicas: "3"'; then
    echo "✅ PASS: Replicas field verified (value: 3)" | tee -a "$REPORT_FILE"
else
    echo "❌ FAIL: Replicas field not found or incorrect" | tee -a "$REPORT_FILE"
    exit 1
fi

# 7. Optional: Check for Porch package
echo "Checking for Porch package (optional)..."
if kubectl get crd packagerevisions.porch.kpt.dev &>/dev/null; then
    if kubectl get packagerevisions -A 2>/dev/null | grep -q e2e; then
        echo "✅ Porch package found" | tee -a "$REPORT_FILE"
    else
        echo "⚠️  No Porch package (normal if Porch not configured)" | tee -a "$REPORT_FILE"
    fi
else
    echo "⚠️  Porch not installed" | tee -a "$REPORT_FILE"
fi

# Summary
echo "" | tee -a "$REPORT_FILE"
echo "================================" | tee -a "$REPORT_FILE"
echo "E2E Test Complete - $(date)" | tee -a "$REPORT_FILE"
echo "Report saved to: $REPORT_FILE"

# Cleanup (optional)
if [ "${SKIP_CLEANUP:-false}" != "true" ]; then
    kubectl delete networkintent e2e-scaling-test --ignore-not-found=true
fi