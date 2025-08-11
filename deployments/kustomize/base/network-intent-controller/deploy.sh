#!/bin/bash

# Deploy Network Intent Controller
# This script deploys the network-intent-controller to a Kubernetes cluster

set -e

echo "========================================"
echo "Network Intent Controller Deployment"
echo "========================================"
echo ""

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    echo "Error: kubectl is not installed"
    exit 1
fi

# Check if we can connect to a cluster
if ! kubectl cluster-info &> /dev/null; then
    echo "Error: Cannot connect to Kubernetes cluster"
    echo "Please ensure your kubeconfig is properly configured"
    exit 1
fi

# Create namespace if it doesn't exist
echo "1. Creating namespace 'nephoran-system' if it doesn't exist..."
kubectl create namespace nephoran-system --dry-run=client -o yaml | kubectl apply -f -

# Deploy the controller
echo ""
echo "2. Deploying network-intent-controller..."
kubectl apply -k .

# Wait for deployment to be ready
echo ""
echo "3. Waiting for deployment to be ready..."
kubectl -n nephoran-system wait --for=condition=available --timeout=120s deployment/network-intent-controller

# Check the status
echo ""
echo "4. Deployment Status:"
kubectl -n nephoran-system get deployment network-intent-controller

echo ""
echo "5. Pods Status:"
kubectl -n nephoran-system get pods -l app=network-intent-controller

echo ""
echo "========================================"
echo "Deployment Complete!"
echo "========================================"
echo ""
echo "To view logs:"
echo "  kubectl -n nephoran-system logs -l app=network-intent-controller"
echo ""
echo "To view metrics:"
echo "  kubectl -n nephoran-system port-forward svc/network-intent-controller-metrics 8080:8080"
echo "  Then visit: http://localhost:8080/metrics"
echo ""