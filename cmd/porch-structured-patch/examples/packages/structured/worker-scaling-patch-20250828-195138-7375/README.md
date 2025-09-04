# worker-scaling-patch-20250828-195138-7375

This package contains a structured patch to scale the worker deployment.

## Intent Details
- **Target**: worker
- **Namespace**: test  
- **Replicas**: 0
- **Intent Type**: scaling

## Files
- `Kptfile`: Package metadata and pipeline configuration
- `scaling-patch.yaml`: Strategic merge patch for deployment scaling

## Usage
Apply this patch package using kpt or Porch:

```bash
kpt fn eval . --image gcr.io/kpt-fn/apply-replacements:v0.1.1
```

## Generated
Generated at: 2025-08-28T19:51:38Z
