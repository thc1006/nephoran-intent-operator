# worker-scaling-patch-20250903-054242-2086



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

Generated at: 2025-09-03T05:42:42Z

