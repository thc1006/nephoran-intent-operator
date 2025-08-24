# worker-scaling-patch-20250822-140744-7404

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
Generated at: 2025-08-22T14:07:44Z
