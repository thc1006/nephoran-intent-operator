#!/bin/bash
#
# This script gathers diagnostic information from a Kubernetes cluster to troubleshoot
# issues with Custom Resource Definitions (CRDs) and the API server.
# It is non-destructive and read-only.

set -e
echo "--- Running Kubernetes Cluster Diagnostics ---"
echo ""

# 1. Verify the installed CRD YAML
echo "1. Verifying the YAML of the installed 'e2nodesets.nephoran.com' CRD..."
echo "   This confirms the CRD exists in the cluster."
echo "---------------------------------------------------------------------"
kubectl get crd e2nodesets.nephoran.com -o yaml || echo "CRD 'e2nodesets.nephoran.com' not found."
echo ""
echo ""

# 2. Check if the API server recognizes the new resource type
echo "2. Checking if the API server is aware of the 'e2nodeset' resource..."
echo "   If this command returns no output, the API server has not registered the CRD."
echo "---------------------------------------------------------------------"
kubectl api-resources | grep e2nodeset || echo "API resource 'e2nodeset' not found."
echo ""
echo ""

# 3. Check the health of control plane components
echo "3. Checking the status of core control plane pods in 'kube-system'..."
echo "   All pods here should be in the 'Running' state."
echo "---------------------------------------------------------------------"
kubectl get pods -n kube-system
echo ""
echo ""

# 4. Retrieve kube-apiserver logs
echo "4. Retrieving the last 100 lines of logs from 'kube-apiserver' pods..."
echo "   Looking for any errors related to CRDs, resource mapping, or caching."
echo "---------------------------------------------------------------------"
# This command might vary slightly depending on the Kubernetes distribution (e.g., k3s uses a different label)
# First, try the standard label
APISERVER_PODS=$(kubectl get pods -n kube-system -l component=kube-apiserver -o name)
if [ -z "$APISERVER_PODS" ]; then
  # Fallback for k3s
  APISERVER_PODS=$(kubectl get pods -n kube-system -l app=k3s -o name | grep 'server')
fi

if [ -n "$APISERVER_PODS" ]; then
  for pod in $APISERVER_PODS; do
    echo "Logs for $pod:"
    kubectl logs -n kube-system "$pod" --tail=100 || echo "Could not retrieve logs for $pod."
  done
else
  echo "Could not find any kube-apiserver pods."
fi
echo ""
echo ""

# 5. Display client and server version information
echo "5. Displaying Kubernetes client and server version..."
echo "---------------------------------------------------------------------"
kubectl version
echo ""
echo ""

# 6. Check the API server's health endpoints
echo "6. Checking the API server's '/readyz' health endpoint..."
echo "   This checks the health of all API server components."
echo "---------------------------------------------------------------------"
kubectl get --raw /readyz?verbose
echo ""
echo ""

echo "--- Diagnostics Complete ---"
