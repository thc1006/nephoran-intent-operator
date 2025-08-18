#!/bin/bash
# Script: 04-porch-apply.sh
# Purpose: Apply package via porchctl or kpt live apply

set -e

PACKAGE_NAME="${PACKAGE_NAME:-nf-sim-package}"
PACKAGE_DIR="${PACKAGE_DIR:-}"
NAMESPACE="${NAMESPACE:-mvp-demo}"
METHOD="${METHOD:-direct}"  # "porch" or "direct"
WAIT_SECONDS="${WAIT_SECONDS:-30}"

echo "==== MVP Demo: Apply Package with Porch/KPT ===="
echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "Method: $METHOD"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Set package directory if not provided
if [ -z "$PACKAGE_DIR" ]; then
    PACKAGE_DIR="$SCRIPT_DIR/package-$PACKAGE_NAME"
fi

# Verify package directory exists
if [ ! -d "$PACKAGE_DIR" ]; then
    echo "Error: Package directory not found: $PACKAGE_DIR"
    echo "Run ./02-prepare-nf-sim.sh first to create the package"
    exit 1
fi

echo "Package directory: $PACKAGE_DIR"

if [ "$METHOD" = "porch" ]; then
    # Apply via Porch
    echo ""
    echo "Applying package via Porch..."
    
    # Check if Porch is available
    if kubectl get crd packagerevisions.porch.kpt.dev >/dev/null 2>&1; then
        # Create PackageRevision
        echo "Creating PackageRevision for $PACKAGE_NAME..."
        
        PR_FILE="/tmp/package-revision-$PACKAGE_NAME.yaml"
        cat > "$PR_FILE" << EOF
apiVersion: porch.kpt.dev/v1alpha1
kind: PackageRevision
metadata:
  name: $PACKAGE_NAME-v1
  namespace: default
spec:
  packageName: $PACKAGE_NAME
  revision: v1
  repository: mvp-packages
  lifecycle: Published
EOF
        
        if kubectl apply -f "$PR_FILE"; then
            echo "PackageRevision created successfully"
            
            # Apply the package
            echo "Applying package with porchctl..."
            porchctl rpkg approve default/$PACKAGE_NAME-v1 2>/dev/null || true
            porchctl rpkg propose-delete default/$PACKAGE_NAME-v1 2>/dev/null || true
        else
            echo "Warning: Could not create PackageRevision. Falling back to direct apply."
            METHOD="direct"
        fi
        
        rm -f "$PR_FILE"
    else
        echo "Warning: Porch CRDs not found. Falling back to direct apply method."
        METHOD="direct"
    fi
fi

if [ "$METHOD" = "direct" ]; then
    # Direct kubectl/kpt apply
    echo ""
    echo "Applying package directly with kubectl..."
    
    # Check if kpt is available for live apply
    if command -v kpt >/dev/null 2>&1; then
        echo "Using kpt live apply..."
        
        cd "$PACKAGE_DIR"
        
        # Initialize kpt live if not already done
        kpt live init . 2>/dev/null || true
        
        # Apply the package
        if kpt live apply . --reconcile-timeout=2m; then
            echo "Package applied successfully with kpt!"
        else
            echo "Warning: kpt live apply failed, trying kubectl apply..."
            kubectl apply -f . --namespace="$NAMESPACE"
        fi
        
        cd - >/dev/null
    else
        # Fall back to kubectl apply
        echo "Using kubectl apply..."
        
        for file in "$PACKAGE_DIR"/*.yaml; do
            if [ "$(basename "$file")" != "Kptfile" ]; then
                echo "Applying $(basename "$file")..."
                kubectl apply -f "$file"
            fi
        done
        
        echo "All resources applied successfully!"
    fi
fi

# Wait for deployment to be ready
echo ""
echo "Waiting for deployment to be ready (max $WAIT_SECONDS seconds)..."

START_TIME=$(date +%s)
TIMEOUT=$((START_TIME + WAIT_SECONDS))

while [ $(date +%s) -lt $TIMEOUT ]; do
    if kubectl get deployment nf-sim -n "$NAMESPACE" >/dev/null 2>&1; then
        READY=$(kubectl get deployment nf-sim -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        DESIRED=$(kubectl get deployment nf-sim -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
        
        echo "Status: $READY/$DESIRED replicas ready"
        
        if [ "$READY" = "$DESIRED" ] && [ "$READY" -gt 0 ]; then
            echo "Deployment is ready!"
            break
        fi
    fi
    
    sleep 2
done

# Show final deployment status
echo ""
echo "==== Deployment Status ===="
kubectl get deployment nf-sim -n "$NAMESPACE"
echo ""
kubectl get pods -n "$NAMESPACE" -l app=nf-sim

# Show service status
echo ""
echo "==== Service Status ===="
kubectl get service nf-sim -n "$NAMESPACE"

# Summary
echo ""
echo "==== Apply Summary ===="
echo "✓ Package: $PACKAGE_NAME"
echo "✓ Namespace: $NAMESPACE"
echo "✓ Method: $METHOD"

if kubectl get deployment nf-sim -n "$NAMESPACE" >/dev/null 2>&1; then
    CURRENT_REPLICAS=$(kubectl get deployment nf-sim -n "$NAMESPACE" -o jsonpath='{.status.replicas}' 2>/dev/null || echo "0")
    READY_REPLICAS=$(kubectl get deployment nf-sim -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    echo "✓ Current replicas: $CURRENT_REPLICAS"
    echo "✓ Ready replicas: $READY_REPLICAS"
fi

echo ""
echo "Package applied! Next step: Run ./05-validate.sh"
echo "To scale, send another intent with ./03-send-intent.sh"