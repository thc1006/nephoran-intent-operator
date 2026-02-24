#!/bin/bash
set -e

XAPP_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$XAPP_DIR"

echo "═══════════════════════════════════════════════════════════════"
echo "  Building and Deploying Scaling xApp"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# Step 1: Initialize go.mod if needed
echo "Step 1: Initializing Go modules..."
if [ ! -f "go.sum" ]; then
    go mod tidy
fi
echo "✅ Go modules ready"
echo ""

# Step 2: Build Docker image
echo "Step 2: Building Docker image..."
docker build -t scaling-xapp:latest .
echo "✅ Docker image built"
echo ""

# Step 3: Load image into kind/k3s if needed
# Uncomment if using kind:
# kind load docker-image scaling-xapp:latest

# Step 4: Deploy to Kubernetes
echo "Step 3: Deploying to Kubernetes..."
kubectl apply -f deployment.yaml
echo "✅ Deployed to Kubernetes"
echo ""

# Step 5: Wait for deployment
echo "Step 4: Waiting for deployment..."
kubectl wait --for=condition=available --timeout=60s deployment/ricxapp-scaling -n ricxapp
echo "✅ Deployment ready"
echo ""

# Step 6: Show status
echo "Step 5: Deployment status:"
kubectl get all -n ricxapp | grep scaling
echo ""

echo "═══════════════════════════════════════════════════════════════"
echo "  ✅ Scaling xApp deployed successfully!"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "Check logs with:"
echo "  kubectl logs -n ricxapp deployment/ricxapp-scaling -f"
