#!/bin/bash
# Deploy Nephoran Web UI to Kubernetes

set -e

NAMESPACE="nephoran-system"
UI_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=== Deploying Nephoran Web UI ==="
echo ""

# Create namespace if it doesn't exist
echo "[1/5] Ensuring namespace exists..."
kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -

# Create ConfigMap from HTML file
echo "[2/5] Creating ConfigMap with web UI content..."
kubectl create configmap nephoran-web-ui \
    --from-file=index.html="$UI_DIR/index.html" \
    --namespace=$NAMESPACE \
    --dry-run=client -o yaml | kubectl apply -f -

# Apply nginx config
echo "[3/5] Applying nginx configuration..."
kubectl apply -f "$UI_DIR/deployment.yaml"

# Apply services
echo "[4/5] Creating services..."
kubectl apply -f "$UI_DIR/service.yaml"

# Wait for deployment
echo "[5/5] Waiting for deployment to be ready..."
kubectl rollout status deployment/nephoran-web-ui -n $NAMESPACE --timeout=60s

echo ""
echo "✅ Deployment complete!"
echo ""
echo "Access the UI:"
echo "  ClusterIP: kubectl port-forward -n $NAMESPACE svc/nephoran-web-ui 8080:80"
echo "  NodePort:  http://<node-ip>:30080"
echo ""
echo "Verify deployment:"
echo "  kubectl get all -n $NAMESPACE -l app=nephoran-web-ui"
