# KRM Scaling Patch

## Target
- Deployment: example-deployment
- Namespace: default
- Replicas: 3

## Usage
Apply this patch using kpt:
```bash
kpt fn eval --image gcr.io/kpt-fn/apply-setters:v0.2.0
kubectl apply -f deployment-patch.yaml
```

## Files
- Kptfile: Package metadata
- deployment-patch.yaml: Deployment replica patch
- setters.yaml: Configuration values
