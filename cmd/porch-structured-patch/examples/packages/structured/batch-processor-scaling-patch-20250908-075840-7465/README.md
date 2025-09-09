# batch-processor-scaling-patch-20250908-075840-7465



This package contains a structured patch to scale the batch-processor deployment.



## Intent Details

- **Target**: batch-processor

- **Namespace**: processing  

- **Replicas**: 100

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

Generated at: 2025-09-08T07:58:40Z

