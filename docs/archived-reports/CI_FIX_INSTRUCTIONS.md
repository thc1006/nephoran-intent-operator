# CI Fix Instructions for Dependabot PR #204

## Problem
Dependabot PR fails CI because the workflow only triggers builds for Go source files, but this PR only changes `elk-stack.yaml` (Docker image version update). The integration validation fails because no binaries are built.

## Required Fixes

### 1. Fix Workflow Change Detection
Edit `.github/workflows/ci-reliability-optimized.yml` line 96:

**Before:**
```yaml
if echo "$CHANGED_FILES" | grep -qE '\.(go|mod|sum)$|^(api|cmd|controllers|pkg|internal)/|Makefile'; then
```

**After:**
```yaml
if echo "$CHANGED_FILES" | grep -qE '\.(go|mod|sum|ya?ml)$|^(api|cmd|controllers|pkg|internal)/|Makefile|\.github/workflows/'; then
```

This ensures YAML file changes (including Docker image updates) trigger builds.

### 2. Create Missing "containers" Label
Run this command or create via GitHub web interface:
```bash
gh label create containers --description "Containerization and Docker-related changes" --color 0366d6
```

Or create manually in GitHub repository settings with:
- **Name:** `containers`
- **Description:** `Containerization and Docker-related changes`
- **Color:** `#0366d6`

### 3. Alternative: Skip Integration Test for Non-Build Changes
If the above fixes aren't applied, the integration validation should be updated to handle cases where builds are legitimately skipped.

## Verification
After applying fixes, the CI should:
1. Detect YAML changes as significant
2. Trigger build jobs for all components
3. Create binary artifacts 
4. Pass integration validation
5. Apply "containers" label without errors

The Fluent Bit version update (2.2.0 → 4.0.9) itself is valid and safe.