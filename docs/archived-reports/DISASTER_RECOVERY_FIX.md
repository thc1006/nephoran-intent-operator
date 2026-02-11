# Disaster Recovery Test Fix

## Problem

The disaster recovery test was failing with the error:
```
unable to start control plane itself: failed to start the controlplane
```

This error occurs because the test environment (`envtest`) requires Kubernetes control plane binaries (etcd, kube-apiserver, kubectl) but they were not installed at the expected path `/usr/local/kubebuilder/bin/etcd`.

## Root Cause

The `envtest` framework from controller-runtime requires:
1. `etcd` - Kubernetes key-value store
2. `kube-apiserver` - Kubernetes API server  
3. `kubectl` - Kubernetes CLI tool

These binaries are downloaded by the `setup-envtest` tool but were not properly installed or configured.

## Solution

### 1. Automated Fix Script
```bash
# Run the comprehensive fix script
./scripts/fix-disaster-recovery-test.sh
```

This script will:
- Install the `setup-envtest` tool
- Download Kubernetes control plane binaries (v1.28.0)
- Install binaries to `/usr/local/kubebuilder/bin/` (or cache location)
- Set `KUBEBUILDER_ASSETS` environment variable
- Add Makefile targets for disaster recovery testing
- Validate the installation

### 2. Manual Installation
```bash
# Install setup-envtest tool
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

# Download and setup assets
ASSETS_PATH=$(setup-envtest use 1.28.0 --bin-dir=$HOME/.cache/kubebuilder-envtest -p path)
export KUBEBUILDER_ASSETS="$ASSETS_PATH"

# Verify installation
ls -la "$KUBEBUILDER_ASSETS"
```

### 3. Environment Setup
```bash
# Set environment variable (add to .bashrc or .envrc)
export KUBEBUILDER_ASSETS="/usr/local/kubebuilder/bin"

# For direnv users, add to .envrc
echo "export KUBEBUILDER_ASSETS=/usr/local/kubebuilder/bin" > .envrc
direnv allow
```

## Files Created/Modified

### New Files
1. `scripts/setup-envtest.sh` - Basic envtest setup script
2. `scripts/fix-disaster-recovery-test.sh` - Comprehensive fix script
3. `scripts/test-disaster-recovery-fix.sh` - Validation script
4. `hack/testtools/envtest_binaries.go` - Binary management utilities
5. `hack/testtools/envtest_setup_fixed.go` - Enhanced envtest setup
6. `tests/disaster-recovery/disaster_recovery_test_fixed.go` - Enhanced test suite

### Makefile Additions
```makefile
##@ Disaster Recovery Testing (envtest)

.PHONY: setup-envtest
setup-envtest: ## Setup kubebuilder envtest assets
	# Installation commands...

.PHONY: test-disaster-recovery  
test-disaster-recovery: setup-envtest ## Run disaster recovery tests
	# Test execution with proper environment...

.PHONY: test-envtest-validation
test-envtest-validation: ## Validate envtest setup
	# Validation commands...
```

## Usage

### Run Disaster Recovery Tests
```bash
# Complete fix workflow
make fix-disaster-recovery-test

# Just run the tests (after setup)
make test-disaster-recovery

# Validate envtest installation
make test-envtest-validation
```

### Manual Testing
```bash
# Set environment
export KUBEBUILDER_ASSETS="/usr/local/kubebuilder/bin"

# Run specific tests
go test ./tests/disaster-recovery/... -v -timeout=20m

# Run enhanced tests
go test ./tests/disaster-recovery/... -v -timeout=30m -run TestEnhancedDisasterRecovery
```

## CI/CD Integration

### GitHub Actions
```yaml
- name: Setup envtest assets
  run: |
    go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
    ASSETS_PATH=$(setup-envtest use 1.28.0 --bin-dir=${HOME}/.cache/kubebuilder-envtest -p path)
    echo "KUBEBUILDER_ASSETS=${ASSETS_PATH}" >> $GITHUB_ENV

- name: Run disaster recovery tests
  env:
    KUBEBUILDER_ASSETS: ${{ env.KUBEBUILDER_ASSETS }}
  run: make test-disaster-recovery
```

### Jenkins/Other CI
```bash
# In CI script
export KUBEBUILDER_ASSETS="${HOME}/.cache/kubebuilder-envtest/1.28.0-linux-amd64"
make setup-envtest
make test-disaster-recovery
```

## Verification

After applying the fix, verify everything works:

```bash
# 1. Check binaries are installed
ls -la /usr/local/kubebuilder/bin/
# Should show: etcd, kube-apiserver, kubectl

# 2. Verify environment variable
echo $KUBEBUILDER_ASSETS
# Should show path to binaries

# 3. Run validation
make test-envtest-validation

# 4. Run actual tests
make test-disaster-recovery
```

## Troubleshooting

### Common Issues

1. **Permission denied**: Use sudo for system directories
   ```bash
   sudo ./scripts/fix-disaster-recovery-test.sh
   ```

2. **Binary not executable**: Check permissions
   ```bash
   chmod +x /usr/local/kubebuilder/bin/*
   ```

3. **Environment variable not set**: Check shell configuration
   ```bash
   source ~/.bashrc
   # or
   direnv reload
   ```

4. **Network issues**: Use manual download
   ```bash
   # Download manually and extract to cache directory
   curl -L https://go.kubebuilder.io/test-tools/1.28.0/linux/amd64/kubebuilder-tools-1.28.0-linux-amd64.tar.gz \
     -o kubebuilder-tools.tar.gz
   ```

### Debug Commands
```bash
# Check setup-envtest installation
which setup-envtest

# List available Kubernetes versions
setup-envtest list

# Get current assets path
setup-envtest use --print-path

# Verbose test output
go test ./tests/disaster-recovery/... -v -timeout=20m -args -ginkgo.v
```

## Technical Details

### Why This Happens
- `envtest` creates a temporary Kubernetes API server for testing
- It needs real Kubernetes binaries to function
- The binaries must be in the path specified by `KUBEBUILDER_ASSETS`
- Without them, the control plane cannot start

### Binary Sources
- Downloaded from `https://go.kubebuilder.io/test-tools/`
- Versions correspond to Kubernetes releases
- Platform-specific (linux/amd64, darwin/amd64, etc.)

### Environment Variables
- `KUBEBUILDER_ASSETS`: Path to kubebuilder binaries
- `ENVTEST_K8S_VERSION`: Kubernetes version to use (default: 1.28.0)
- `USE_EXISTING_CLUSTER`: Use existing cluster instead of envtest

## Next Steps

1. ✅ Apply the fix using the provided scripts
2. ✅ Verify the installation with validation commands
3. ✅ Run disaster recovery tests to confirm they pass
4. ✅ Update CI/CD pipelines to include envtest setup
5. ✅ Document the process for team members

The disaster recovery test should now run successfully without the "unable to start control plane" error.