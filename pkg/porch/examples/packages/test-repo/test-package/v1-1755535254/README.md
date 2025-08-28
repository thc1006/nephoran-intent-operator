# Scaling Package: test-package

## Intent Details
- Target: test-deployment
- Namespace: test-ns
- Replicas: 5
- Intent Type: scaling

## Package Contents
- Kptfile: Package metadata
- deployment-patch.yaml: Kubernetes deployment patch for scaling
- kustomization.yaml: Kustomize configuration

## Usage
This package is managed by Porch and applies scaling operations to the specified deployment.
