#!/bin/bash
#
# This script actively troubleshoots the "resource mapping not found" error
# when the CRD appears to be correctly registered.

set -e
echo "--- Running Targeted Diagnostics for Resource Mapping Conflict ---"
echo ""

# --- ACTION 1: Clear local kubectl cache ---
echo "1. Clearing local kubectl cache..."
echo "   This will remove any stale discovery information from the client."
echo "---------------------------------------------------------------------"
rm -rf ~/.kube/cache
echo "Cache cleared."
echo ""
echo ""

# --- DIAGNOSIS 2: Verify User Permissions ---
echo "2. Verifying permissions for the current user..."
echo "   This checks if you have the 'get' and 'create' rights for 'e2nodesets'."
echo "---------------------------------------------------------------------"
echo "Checking 'get' permission:"
kubectl auth can-i get e2nodesets -n default
echo ""
echo "Checking 'create' permission:"
kubectl auth can-i create e2nodesets -n default
echo ""
echo ""

# --- ACTION 3: Attempt Resource Creation with Strict Validation ---
# Create a temporary, minimal E2NodeSet manifest for the test.
cat <<EOF > ./e2nodeset-test.yaml
apiVersion: nephoran.com/v1alpha1
kind: E2NodeSet
metadata:
  name: e2nodeset-validation-test
spec:
  replicas: 1
EOF

echo "3. Attempting to create a test E2NodeSet with strict validation..."
echo "   This will force the API server to validate the request against the schema."
echo "---------------------------------------------------------------------"
if kubectl apply -f ./e2nodeset-test.yaml --validate=true; then
  echo "SUCCESS: Test resource created successfully."
  kubectl delete -f ./e2nodeset-test.yaml
else
  echo "FAILURE: The 'apply' command failed again. This strongly points to an RBAC issue."
fi
rm ./e2nodeset-test.yaml
echo ""
echo ""

# --- DIAGNOSIS 4: Find Relevant RBAC Bindings ---
echo "4. Searching for RBAC bindings related to the current user..."
echo "   Review these policies to see if they grant the required permissions."
echo "---------------------------------------------------------------------"
# Get the current user from the kubeconfig
CURRENT_USER=$(kubectl config view --minify -o jsonpath='{.users[0].name}')
echo "Current user is: $CURRENT_USER"
echo ""

echo "--- ClusterRoleBindings for user '$CURRENT_USER' ---"
kubectl get clusterrolebindings -o json | jq -r --arg user "$CURRENT_USER" '.items[] | select(.subjects[]?.name == $user) | .metadata.name' | xargs -I {} kubectl get clusterrolebinding {} -o yaml

echo ""
echo "--- RoleBindings in 'default' namespace for user '$CURRENT_USER' ---"
kubectl get rolebindings -n default -o json | jq -r --arg user "$CURRENT_USER" '.items[] | select(.subjects[]?.name == $user) | .metadata.name' | xargs -I {} kubectl get rolebinding {} -n default -o yaml

echo ""
echo "--- Diagnostics Complete ---"
