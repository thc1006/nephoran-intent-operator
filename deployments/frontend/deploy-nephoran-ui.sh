#!/bin/bash
set -euo pipefail

# Nephoran UI Deployment Script
# Deploys the cyber-terminal web interface for the Nephoran Intent Operator

NAMESPACE="nephoran-system"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🚀 Deploying Nephoran Intent Operator Web UI"
echo "=============================================="
echo ""

# Check prerequisites
echo "📋 Checking prerequisites..."

if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Please install kubectl first."
    exit 1
fi

if ! kubectl cluster-info &> /dev/null; then
    echo "❌ Cannot connect to Kubernetes cluster. Please check your kubeconfig."
    exit 1
fi

echo "✅ kubectl found and cluster accessible"
echo ""

# Create namespace if it doesn't exist
echo "🏗️  Creating namespace ${NAMESPACE}..."
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
echo "✅ Namespace ready"
echo ""

# Create ConfigMap with HTML content
echo "📦 Creating ConfigMap with UI content..."
kubectl create configmap nephoran-ui-html \
  --from-file=index.html="${SCRIPT_DIR}/nephoran-ui.html" \
  --namespace=${NAMESPACE} \
  --dry-run=client -o yaml | kubectl apply -f -
echo "✅ ConfigMap created"
echo ""

# Deploy resources using kustomize (if available) or plain kubectl
echo "🎨 Deploying Kubernetes resources..."

if command -v kustomize &> /dev/null; then
    echo "   Using kustomize..."
    kubectl apply -k "${SCRIPT_DIR}/"
else
    echo "   Using kubectl apply..."
    kubectl apply -f "${SCRIPT_DIR}/deployment.yaml"
    kubectl apply -f "${SCRIPT_DIR}/service.yaml"
fi

echo "✅ Resources deployed"
echo ""

# Wait for deployment to be ready
echo "⏳ Waiting for deployment to be ready..."
kubectl rollout status deployment/nephoran-ui -n ${NAMESPACE} --timeout=120s
echo "✅ Deployment ready"
echo ""

# Get service information
echo "📡 Service Information"
echo "======================"
kubectl get svc nephoran-ui -n ${NAMESPACE}
echo ""

# Get NodePort
NODE_PORT=$(kubectl get svc nephoran-ui -n ${NAMESPACE} -o jsonpath='{.spec.ports[0].nodePort}')
echo "🌐 Access URLs:"
echo "==============="

# Try to get node IP
if NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null); then
    echo "   NodePort:     http://${NODE_IP}:${NODE_PORT}"
else
    echo "   NodePort:     http://<node-ip>:${NODE_PORT}"
fi

echo "   Port Forward: kubectl port-forward -n ${NAMESPACE} svc/nephoran-ui 8080:80"
echo "                 Then access: http://localhost:8080"
echo ""

# Show pod status
echo "📊 Pod Status:"
echo "=============="
kubectl get pods -n ${NAMESPACE} -l app=nephoran-ui
echo ""

# Verify intent-ingest service
echo "🔍 Checking backend service..."
if kubectl get svc intent-ingest-service -n ${NAMESPACE} &> /dev/null; then
    echo "✅ intent-ingest-service found in namespace ${NAMESPACE}"
elif kubectl get svc intent-ingest-service --all-namespaces &> /dev/null; then
    BACKEND_NS=$(kubectl get svc intent-ingest-service --all-namespaces -o jsonpath='{.items[0].metadata.namespace}')
    echo "⚠️  intent-ingest-service found in namespace ${BACKEND_NS}, not ${NAMESPACE}"
    echo "   You may need to update the nginx proxy configuration to use:"
    echo "   http://intent-ingest-service.${BACKEND_NS}.svc.cluster.local:8080/intent"
else
    echo "⚠️  intent-ingest-service not found. Please deploy the backend service."
fi
echo ""

echo "✅ Deployment complete!"
echo ""
echo "🎉 Nephoran Intent Operator UI is now available!"
echo "   Visit the URL above to start submitting natural language intents."
echo ""
