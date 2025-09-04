# worker-scaling-patch-20250901-141912-5385



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

Generated at: 2025-09-01T14:19:12Z

