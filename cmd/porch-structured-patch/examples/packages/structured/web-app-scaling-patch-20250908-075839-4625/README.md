# web-app-scaling-patch-20250908-075839-4625



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

Generated at: 2025-09-08T07:58:39Z

