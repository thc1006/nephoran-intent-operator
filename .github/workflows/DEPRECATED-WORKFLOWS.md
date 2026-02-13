# Deprecated Workflows

These workflows were removed during CI consolidation in favor of:
- `ci-2025.yml` (primary CI)
- `pr-validation.yml` (required PR gate producing `Basic Validation`)
- `ubuntu-ci.yml` (manual full suite)
- `emergency-merge.yml` (manual emergency)
- `go-module-cache.yml` (reusable cache helper)

Removed set includes old/overlapping CI variants, temporary bypass flows, and legacy 2025 experimental pipelines.
