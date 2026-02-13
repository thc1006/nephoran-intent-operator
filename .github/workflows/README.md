# GitHub Workflows

## Active workflows
- `ci-2025.yml`: primary CI workflow
- `pr-validation.yml`: PR gate checks (`Basic Validation`, `CI Status`)
- `ubuntu-ci.yml`: manual full validation suite
- `emergency-merge.yml`: manual emergency pipeline
- `go-module-cache.yml`: reusable cache workflow
- `ci-monitoring.yml`: scheduled CI health monitoring
- `cache-recovery-system.yml`: manual/cache-recovery helper
- `branch-protection-setup.yml`: manual branch policy setup
- `debug-ghcr-auth.yml`: manual GHCR debug
- `claude.yml`, `emergency-disable.yml`: specialized/manual workflows

## Policy
- Ubuntu-only runners.
- Per-branch concurrency group: `${{ github.ref }}`.
- Required gate context for protected branches: `CI Status`.
