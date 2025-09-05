# Lightweight E2E Test

## Overview

This lightweight E2E test focuses on **core validation** rather than full scaling scenarios. It uses **mock scaling** (1 replica instead of 3) to avoid resource constraints while still validating the complete E2E pipeline.

## Key Features

- ✅ **Fast & Reliable**: Uses minimal resources (1 replica) for consistent results
- ✅ **Core Validation**: Tests all critical components without heavy resource usage  
- ✅ **Comprehensive Debugging**: Detailed output when things fail
- ✅ **Proper Cleanup**: Automatically cleans up kind clusters
- ✅ **Mock Scaling**: Simulates scaling behavior without resource constraints

## Quick Start

### 1. Check Prerequisites
```bash
hack/check-e2e-prereqs.sh
```

### 2. Run Lightweight E2E Test
```bash
# Basic run
hack/run-e2e-lightweight.sh

# With debug output
hack/run-e2e-lightweight.sh --debug

# Skip cleanup (useful for debugging)
hack/run-e2e-lightweight.sh --no-cleanup
```

## Expected Output

The test produces output in this format:
```
Replicas (nf-sim, ran-a): desired=1, ready=1 (OK)
```

## Test Flow

1. **Create kind cluster** → Validates cluster setup
2. **Install CRDs** → Tests CRD installation and functionality  
3. **Deploy test workload** → Lightweight nginx deployment
4. **Create NetworkIntent** → Tests intent resource creation
5. **Mock scaling** → Simulates operator behavior (1→1 replicas)
6. **Verify results** → Validates complete pipeline

## Configuration

Environment variables:
- `CLUSTER_NAME` - Cluster name (default: `nephoran-e2e`)
- `NAMESPACE` - Operator namespace (default: `nephoran-system`)
- `SKIP_CLEANUP` - Skip cleanup (default: `false`)
- `DEBUG` - Enable debug output (default: `false`)

## Why Mock Scaling?

- **Resource Efficiency**: Uses minimal CPU/memory in CI environments
- **Reliability**: Avoids resource contention and flaky tests
- **Speed**: Faster execution without waiting for multiple pods
- **Focus**: Tests the E2E pipeline logic, not Kubernetes scaling

## Debugging

For debugging failed tests:

```bash
# Run with debug output
DEBUG=true hack/run-e2e-lightweight.sh --no-cleanup

# Manual cluster inspection
kubectl get all -n ran-a
kubectl get networkintents -n ran-a
kubectl describe deployment nf-sim -n ran-a
```

## Integration with CI

The test is designed to work in GitHub Actions and other CI environments with minimal resource requirements:

```yaml
- name: Run E2E Test
  run: |
    hack/check-e2e-prereqs.sh
    hack/run-e2e-lightweight.sh
```

## Comparison with Full E2E

| Aspect | Lightweight E2E | Full E2E |
|--------|-----------------|----------|
| Replicas | 1 → 1 (mock) | 1 → 3 (real) |
| Resources | Minimal | Higher |
| Speed | ~2-3 minutes | ~5-10 minutes |
| Reliability | High | Medium |
| Focus | Pipeline validation | Full scaling |

## Files

- `hack/run-e2e-lightweight.sh` - Main test script
- `hack/check-e2e-prereqs.sh` - Prerequisite checker
- `tools/verify-scale.go` - Go-based verification tool
- `config/crd/bases/intent.nephoran.com_networkintents.yaml` - Required CRD

## Success Criteria

The test passes when:
1. Kind cluster creates successfully
2. CRDs install and become functional
3. Test workload deploys and becomes ready
4. NetworkIntent resource creates successfully
5. Mock scaling completes with expected replica count
6. Final verification shows: `desired=1, ready=1 (OK)`