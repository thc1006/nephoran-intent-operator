# CRD Generation Guide

## Overview
This guide helps prevent CRD generation inconsistency issues that can cause CI failures.

## The Problem
When you modify Go types in `api/` directory with kubebuilder annotations, the generated CRD YAML files must be regenerated and committed alongside your code changes. If they're out of sync, CI will fail with:
```
❌ Generated files are not up to date. Run 'make gen' and commit the changes.
```

## Required Tools

### Install controller-gen v0.18.0 (CI version)
```bash
go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.18.0
controller-gen --version  # Should show v0.18.0
```

## Development Workflow

### Before Committing API Changes

1. **After modifying any Go files in `api/` directory:**
   ```bash
   make gen
   ```

2. **Verify changes:**
   ```bash
   git status  # Should show changes in config/crd/ and deployments/crds/
   ```

3. **Commit both source and generated files together:**
   ```bash
   git add api/ config/crd/ deployments/crds/
   git commit -m "feat(api): add new field to NetworkIntent CRD"
   ```

### Manual CRD Generation (if make gen fails)

```bash
# Direct controller-gen command with same flags as CI
controller-gen crd:allowDangerousTypes=true rbac:roleName=manager-role webhook \
    paths="./api/..." output:crd:artifacts:config=config/crd/bases

# Copy to deployments directory for consistency  
cp -r config/crd/bases/* deployments/crds/
```

### Verify CRD Consistency

```bash
# Check if CRDs are up to date
make gen
git diff --exit-code config/crd/ deployments/crds/
```

## Common Issues & Solutions

### Issue 1: Float types without explicit type declarations
**Error:** CRDs fail to generate or contain invalid schemas

**Solution:** Use `crd:allowDangerousTypes=true` flag (already configured in our Makefile)

### Issue 2: Different controller-gen versions
**Error:** Generated CRDs differ between local and CI

**Solution:** Always use the same version as CI (currently v0.18.0)

### Issue 3: Missing CRD directories
**Error:** No CRD files generated

**Solution:**
```bash
mkdir -p config/crd/bases deployments/crds
make gen
```

## Pre-commit Hook Setup

Create a pre-commit hook to automatically check CRD consistency:

```bash
# Create hooks directory in main git repo (not worktree)
cd /path/to/main/repo
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
set -e

echo "🔍 Checking CRD generation consistency..."

if git diff --cached --name-only | grep -E "^api/.*\.go$" > /dev/null; then
    echo "📝 API changes detected, verifying CRD consistency..."
    
    # Generate CRDs
    make gen
    
    # Check if anything changed
    if ! git diff --quiet config/crd/ deployments/crds/; then
        echo "❌ Generated CRDs are out of sync!"
        echo "Please run 'make gen' and stage the changes."
        exit 1
    fi
    
    echo "✅ CRD generation consistency verified"
fi
EOF

chmod +x .git/hooks/pre-commit
```

## Kubebuilder Annotations Reference

### Common annotations that affect CRD generation:

```go
// +kubebuilder:validation:Required
// +kubebuilder:validation:Optional
// +kubebuilder:validation:Minimum=0
// +kubebuilder:validation:Maximum=100
// +kubebuilder:validation:Pattern="^[a-zA-Z0-9]+$"
// +kubebuilder:validation:Enum=value1;value2;value3
// +kubebuilder:default="default-value"

type MySpec struct {
    // Float fields work with allowDangerousTypes=true
    // +kubebuilder:validation:Minimum=0
    ThroughputMbps float64 `json:"throughputMbps"`
    
    // Resource quantities don't need pattern validation
    MemoryLimit resource.Quantity `json:"memoryLimit"`
}
```

## CI Integration

Our CI workflow uses these steps:
1. Install controller-gen v0.18.0
2. Run `make gen`  
3. Check `git diff --exit-code` to ensure no changes
4. If changes detected, fail with helpful error message

## Troubleshooting

### Problem: "command not found: controller-gen"
**Solution:**
```bash
go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.18.0
export PATH=$PATH:$(go env GOPATH)/bin
```

### Problem: "allowDangerousTypes" errors
**Solution:** This is already configured correctly in our Makefile. Float64 types are considered "dangerous" but are necessary for our telecommunications use case.

### Problem: Permission denied on generated files
**Solution:**
```bash
chmod 644 config/crd/bases/*.yaml deployments/crds/*.yaml
```

## Quick Commands Reference

```bash
# Generate CRDs
make gen

# Check consistency  
make gen && git diff --exit-code config/crd/ deployments/crds/

# Install correct controller-gen version
go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.18.0

# Verify installation
controller-gen --version

# Force regeneration
rm -rf config/crd/bases/*.yaml deployments/crds/*.yaml
make gen
```

---

Following this guide will prevent CRD generation issues and keep your CI pipeline green! 🚀