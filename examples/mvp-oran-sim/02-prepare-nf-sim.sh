#!/bin/bash
# Script: 02-prepare-nf-sim.sh
# Purpose: Create a minimal KRM package for NF simulator deployment

set -e

NAMESPACE="${NAMESPACE:-mvp-demo}"
PACKAGE_NAME="${PACKAGE_NAME:-nf-sim-package}"
PACKAGE_REPO="${PACKAGE_REPO:-mvp-packages}"
SKIP_NAMESPACE="${SKIP_NAMESPACE:-false}"

echo "==== MVP Demo: Prepare NF Simulator Package ===="
echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Create namespace if needed
if [ "$SKIP_NAMESPACE" != "true" ]; then
    echo ""
    echo "Creating namespace: $NAMESPACE"
    
    if kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
        echo "Namespace $NAMESPACE already exists"
    else
        kubectl create namespace "$NAMESPACE"
        echo "Namespace $NAMESPACE created successfully"
    fi
fi

# Create local package directory structure
PACKAGE_DIR="$SCRIPT_DIR/package-$PACKAGE_NAME"
echo ""
echo "Creating package directory: $PACKAGE_DIR"

if [ -d "$PACKAGE_DIR" ]; then
    echo "Removing existing package directory..."
    rm -rf "$PACKAGE_DIR"
fi

mkdir -p "$PACKAGE_DIR"

# Create Kptfile
cat > "$PACKAGE_DIR/Kptfile" << EOF
apiVersion: kpt.dev/v1
kind: Kptfile
metadata:
  name: $PACKAGE_NAME
  annotations:
    config.kubernetes.io/local-config: "true"
info:
  description: NF Simulator package for MVP demo
  keywords:
  - nf-sim
  - mvp
  - oran
pipeline:
  mutators:
  - image: gcr.io/kpt-fn/set-namespace:v0.4.1
    configMap:
      namespace: $NAMESPACE
EOF

echo "Created Kptfile"

# Copy or create deployment YAML
DEPLOYMENT_SOURCE="$SCRIPT_DIR/nf-sim-deployment.yaml"
DEPLOYMENT_DEST="$PACKAGE_DIR/nf-sim-deployment.yaml"

if [ -f "$DEPLOYMENT_SOURCE" ]; then
    cp "$DEPLOYMENT_SOURCE" "$DEPLOYMENT_DEST"
    echo "Copied nf-sim-deployment.yaml"
else
    # Create deployment YAML if it doesn't exist
    cat > "$DEPLOYMENT_DEST" << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nf-sim
  namespace: $NAMESPACE
  labels:
    app: nf-sim
    component: cnf-simulator
    orchestrator: porch
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nf-sim
  template:
    metadata:
      labels:
        app: nf-sim
        component: cnf-simulator
    spec:
      containers:
      - name: simulator
        image: nginx:alpine
        ports:
        - containerPort: 80
          name: http
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "128Mi"
            cpu: "100m"
        livenessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: nf-sim
  namespace: $NAMESPACE
  labels:
    app: nf-sim
spec:
  selector:
    app: nf-sim
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
  type: ClusterIP
EOF
    echo "Created nf-sim-deployment.yaml"
fi

# Create package README
cat > "$PACKAGE_DIR/README.md" << EOF
# NF Simulator Package

This package deploys a Network Function (NF) simulator for the MVP demo.

## Contents
- \`nf-sim-deployment.yaml\`: Kubernetes Deployment and Service for the NF simulator
- \`Kptfile\`: KPT package metadata and pipeline configuration

## Usage
This package is managed by Porch and can be scaled using NetworkIntent CRDs.

## Default Configuration
- Namespace: $NAMESPACE
- Initial replicas: 1
- Container: nginx:alpine (lightweight simulator)
EOF

echo "Created README.md"

# Initialize kpt package
echo ""
echo "Initializing kpt package..."
cd "$PACKAGE_DIR"
kpt pkg init . 2>/dev/null || echo "Note: kpt pkg init may show warnings, but package is still valid"
cd - >/dev/null

# If Porch is available, register the package repository
echo ""
echo "Checking Porch availability..."
if kubectl get crd repositories.porch.kpt.dev >/dev/null 2>&1; then
    echo "Porch CRDs are available"
    
    # Check if repository exists
    if ! kubectl get repository "$PACKAGE_REPO" -n default >/dev/null 2>&1; then
        echo ""
        echo "Creating Porch repository: $PACKAGE_REPO"
        
        REPO_FILE="/tmp/repo-$PACKAGE_REPO.yaml"
        cat > "$REPO_FILE" << EOF
apiVersion: porch.kpt.dev/v1alpha1
kind: Repository
metadata:
  name: $PACKAGE_REPO
  namespace: default
spec:
  type: git
  content: Package
  deployment: false
  git:
    repo: https://github.com/example/mvp-packages
    branch: main
    directory: /
EOF
        
        if kubectl apply -f "$REPO_FILE"; then
            echo "Repository $PACKAGE_REPO created"
        else
            echo "Warning: Could not create repository. Manual setup may be required."
        fi
        rm -f "$REPO_FILE"
    else
        echo "Repository $PACKAGE_REPO already exists"
    fi
else
    echo "Warning: Porch is not installed. Package created locally but not registered with Porch."
    echo "Install Porch first using ./01-install-porch.sh"
fi

# Summary
echo ""
echo "==== Package Preparation Summary ===="
echo "✓ Namespace: $NAMESPACE"
echo "✓ Package directory: $PACKAGE_DIR"
echo "✓ Package contents:"
ls -la "$PACKAGE_DIR" | grep -v "^total" | grep -v "^\." | awk '{print "  - " $NF}'

echo ""
echo "Package prepared! Next step: Run ./03-send-intent.sh"
echo "Package location: $PACKAGE_DIR"