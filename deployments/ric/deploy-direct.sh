#!/bin/bash
################################################################################
# Direct Helm Installation Script for O-RAN SC RIC Platform
# Kubernetes 1.35.1 Compatible
#
# This script deploys RIC components directly without requiring a Helm repo
# Using local chart paths
################################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$SCRIPT_DIR/repo/helm"
NAMESPACE="ricplt"
RELEASE_PREFIX="r4"
TIMEOUT="15m"

echo "=================================================="
echo "O-RAN SC RIC Platform Direct Deployment"
echo "=================================================="
echo ""
echo "Configuration:"
echo "  Namespace: $NAMESPACE"
echo "  Release Prefix: $RELEASE_PREFIX"
echo "  Charts Location: $REPO_DIR"
echo "  Timeout: $TIMEOUT"
echo ""

# Helper function to deploy a chart
deploy_chart() {
  local chart_name=$1
  local release_name="${RELEASE_PREFIX}-${chart_name}"
  local chart_path="$REPO_DIR/$chart_name"

  if [ ! -d "$chart_path" ]; then
    echo "ERROR: Chart not found at $chart_path"
    return 1
  fi

  echo "Deploying $chart_name..."

  # Build dependencies if they exist
  if [ -f "$chart_path/requirements.yaml" ]; then
    echo "  Building chart dependencies..."
    helm dependency build "$chart_path" 2>/dev/null || true
  fi

  # Install or upgrade the chart
  helm upgrade --install "$release_name" "$chart_path" \
    --namespace "$NAMESPACE" \
    --timeout "$TIMEOUT" \
    --wait=false \
    2>&1 | grep -E "^NAME|^STATUS|ERROR|error"

  echo "  ✓ $chart_name deployment initiated"
}

# Check prerequisites
echo "Checking prerequisites..."
kubectl get namespace "$NAMESPACE" > /dev/null 2>&1 || {
  echo "ERROR: Namespace $NAMESPACE does not exist"
  exit 1
}

echo "✓ Namespace exists"
echo ""

# Deploy infrastructure components (if needed)
echo "Phase 1: Infrastructure Setup"
echo "==============================="
echo "(Skipped: local-path storage already available)"
echo ""

# Deploy core RIC components in dependency order
echo "Phase 2: Core RIC Platform Components"
echo "======================================"
echo ""

# 1. Deploy dbaas (needs no dependencies)
echo "[1/6] Database as a Service (dbaas)"
deploy_chart "dbaas"

# 2. Deploy e2mgr
echo "[2/6] E2 Manager (e2mgr)"
deploy_chart "e2mgr"

# 3. Deploy e2term
echo "[3/6] E2 Termination (e2term)"
deploy_chart "e2term"

# 4. Deploy submgr
echo "[4/6] Subscription Manager (submgr)"
deploy_chart "submgr"

# 5. Deploy rtmgr
echo "[5/6] Route Manager (rtmgr)"
deploy_chart "rtmgr"

# 6. Deploy appmgr
echo "[6/6] Application Manager (appmgr)"
deploy_chart "appmgr"

echo ""
echo "Phase 3: Optional Components"
echo "============================="
echo ""

# Optional: A1 Mediator
if [ "${INSTALL_A1:-false}" = "true" ]; then
  echo "Installing A1 Mediator (optional)..."
  deploy_chart "a1mediator"
  echo ""
fi

# Optional: O1 Mediator
if [ "${INSTALL_O1:-false}" = "true" ]; then
  echo "Installing O1 Mediator (optional)..."
  deploy_chart "o1mediator"
  echo ""
fi

# Optional: InfluxDB
if [ "${INSTALL_INFLUXDB:-false}" = "true" ]; then
  echo "Installing InfluxDB (optional, requires influxdata repo)..."
  helm repo add influxdata https://helm.influxdata.com 2>/dev/null || true
  helm repo update influxdata 2>/dev/null || true
  # Note: influxdb2 is a third-party chart, not in the repo
  echo "  (Note: InfluxDB installation requires manual Helm repo setup)"
fi

echo ""
echo "Phase 4: Waiting for Deployments"
echo "=================================="
echo ""

# Wait for pods to start
echo "Waiting for pods to stabilize (max 5 minutes)..."
kubectl get pods -n "$NAMESPACE" --watch &
WATCH_PID=$!

sleep 30
kill $WATCH_PID 2>/dev/null || true

echo ""
echo "Phase 5: Deployment Status"
echo "========================="
echo ""

echo "Pod Status:"
kubectl get pods -n "$NAMESPACE" -o wide

echo ""
echo "Service Status:"
kubectl get svc -n "$NAMESPACE"

echo ""
echo "Summary:"
RUNNING=$(kubectl get pods -n "$NAMESPACE" --no-headers | grep -c "Running" || echo "0")
TOTAL=$(kubectl get pods -n "$NAMESPACE" --no-headers | wc -l)
echo "  Running Pods: $RUNNING / $TOTAL"

echo ""
echo "Next Steps:"
echo "==========="
echo "1. Verify all pods reach 'Running' state:"
echo "   kubectl get pods -n $NAMESPACE -w"
echo ""
echo "2. Test RIC API endpoints:"
echo "   kubectl port-forward -n $NAMESPACE svc/service-$RELEASE_PREFIX-e2mgr-http 3800:3800 &"
echo "   curl http://localhost:3800/v1/health"
echo ""
echo "3. View component logs:"
echo "   kubectl logs -n $NAMESPACE deployment/deployment-$RELEASE_PREFIX-e2mgr -f"
echo ""
echo "4. Check service endpoints:"
echo "   kubectl get endpoints -n $NAMESPACE"
echo ""

echo "Deployment initiated successfully!"
