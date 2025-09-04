# web-app-scaling-patch-20250820-225814-0028

This package contains a structured patch to scale the web-app deployment.

## Intent Details
- **Target**: web-app
- **Namespace**: default  
- **Replicas**: 5
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
Generated at: 2025-08-20T22:58:14Z
