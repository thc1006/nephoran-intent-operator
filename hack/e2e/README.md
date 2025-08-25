# E2E Testing Framework for Nephoran Intent Operator

This directory contains comprehensive end-to-end testing tools and documentation for the Nephoran Intent Operator project.

## Overview

The E2E testing framework provides:
- **Robust test harness** with clear PASS/FAIL reporting
- **Cross-platform support** (Windows PowerShell and Unix Bash)
- **Component testing** for webhook validation, intent processing, and conductor loops
- **Integration testing** with Porch and KRM patches
- **Sample scenarios** for common use cases

## Quick Start

### Windows (PowerShell)
```powershell
# Native PowerShell (recommended)
.\hack\run-e2e.ps1

# Using Git Bash (fallback)
.\hack\run-e2e.ps1 -UseBash

# With custom parameters
.\hack\run-e2e.ps1 -ClusterName "my-test" -Namespace "test-ns" -SkipCleanup
```

### Unix/Linux/macOS (Bash)
```bash
# Basic run
./hack/run-e2e.sh

# With environment variables
CLUSTER_NAME=my-test NAMESPACE=test-ns ./hack/run-e2e.sh

# Skip cleanup for debugging
SKIP_CLEANUP=true ./hack/run-e2e.sh
```

## Test Architecture

The E2E framework follows this workflow:

```
1. Setup Phase
   ├── Tool Detection (kind, kubectl, kustomize, go)
   ├── Kind Cluster Creation
   ├── CRD Installation
   └── Namespace Setup

2. Component Deployment
   ├── Webhook Manager Deployment
   ├── Intent Ingest Component (local/sidecar)
   └── Conductor Loop Component (local/in-cluster)

3. Validation Tests
   ├── Valid NetworkIntent Acceptance
   ├── Invalid NetworkIntent Rejection
   ├── Sample Scenario Tests
   └── Porch Integration Tests

4. Cleanup Phase
   ├── Test Resource Cleanup
   └── Cluster Deletion (optional)
```

## Configuration Options

### Environment Variables (Bash)

| Variable | Default | Description |
|----------|---------|-------------|
| `CLUSTER_NAME` | `nephoran-e2e` | Kind cluster name |
| `KIND_IMAGE` | (default) | Custom kind node image |
| `NAMESPACE` | `nephoran-system` | Test namespace |
| `CRD_DIR` | `deployments/crds` | CRD directory path |
| `WEBHOOK_CONFIG` | `config/webhook` | Webhook kustomize config |
| `SAMPLES_DIR` | `tests/e2e/samples` | Sample YAML directory |
| `SKIP_CLEANUP` | `false` | Skip cluster cleanup |
| `INTENT_INGEST_MODE` | `local` | `local` or `sidecar` |
| `CONDUCTOR_MODE` | `local` | `local` or `in-cluster` |
| `PORCH_MODE` | `structured-patch` | `structured-patch` or `direct` |

### PowerShell Parameters

```powershell
-ClusterName "nephoran-e2e"           # Kind cluster name
-KindImage ""                         # Custom kind image
-Namespace "nephoran-system"          # Test namespace
-IntentIngestMode "local"             # local/sidecar
-ConductorMode "local"                # local/in-cluster
-PorchMode "structured-patch"         # structured-patch/direct
-SkipCleanup                          # Skip cleanup
-UseBash                              # Use bash script
-Verbose                              # Verbose output
-Timeout 300                          # Timeout in seconds
```

## Prerequisites

### Required Tools

1. **kind** - Kubernetes in Docker
   - Install: https://kind.sigs.k8s.io/docs/user/quick-start/#installation
   - Windows: `choco install kind`

2. **kubectl** - Kubernetes CLI
   - Install: https://kubernetes.io/docs/tasks/tools/
   - Windows: `choco install kubernetes-cli`

3. **kustomize** (optional)
   - Install: https://kustomize.io/
   - Can use kubectl's built-in kustomize

4. **go** (for local components)
   - Install: https://golang.org/dl/
   - Required when running components locally

### Optional Tools

- **Docker** - For kind cluster management
- **Git Bash** - For Windows bash script execution

## Test Scenarios

### 1. Basic Webhook Validation

Tests webhook admission control:
- ✅ Valid NetworkIntent accepted
- ❌ Invalid NetworkIntent rejected

### 2. Component Integration

Tests component interaction:
- Intent Ingest processing
- Conductor Loop execution
- Handoff event generation

### 3. Porch Integration

Tests Porch package management:
- Structured patch generation
- Direct Porch API calls
- Package revision validation

### 4. Sample Scenarios

Tests realistic use cases from `tests/e2e/samples/`:
- Scaling scenarios
- Resource optimization
- Multi-tenant deployments

## Troubleshooting

### Common Issues

#### "kind not found"
```bash
# Install kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Windows
choco install kind
```

#### "kubectl not found"
```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# Windows
choco install kubernetes-cli
```

#### "CRD directory not found"
Ensure you're running from the project root:
```bash
cd /path/to/nephoran-intent-operator
./hack/run-e2e.sh
```

#### "Webhook deployment failed"
Check webhook configuration:
```bash
kubectl kustomize config/webhook
kubectl get validatingwebhookconfigurations
```

### Debug Mode

Keep cluster for inspection:
```bash
# Bash
SKIP_CLEANUP=true ./hack/run-e2e.sh

# PowerShell
.\hack\run-e2e.ps1 -SkipCleanup
```

Then inspect:
```bash
kubectl get all -n nephoran-system
kubectl logs -n nephoran-system deployment/webhook-manager
```

### Log Analysis

View component logs:
```bash
# Webhook manager logs
kubectl logs -n nephoran-system deployment/webhook-manager

# Event logs
kubectl get events -n nephoran-system --sort-by='.lastTimestamp'

# CRD status
kubectl get networkintents -n nephoran-system -o yaml
```

## Development

### Adding New Tests

1. Create test scenario in `tests/e2e/samples/`
2. Add validation logic in test scripts
3. Update documentation

### Custom Test Components

To test custom components:
1. Build component: `go build -o /tmp/my-component ./cmd/my-component`
2. Run in background: `KUBECONFIG=$(kind get kubeconfig --name nephoran-e2e) /tmp/my-component &`
3. Add cleanup: `kill $COMPONENT_PID`

### CI Integration

For CI/CD pipelines:
```yaml
- name: Run E2E Tests
  run: |
    ./hack/run-e2e.sh
  env:
    SKIP_CLEANUP: "false"
    CLUSTER_NAME: "ci-${{ github.run_id }}"
```

## Files in this Framework

- `../run-e2e.sh` - Main bash test harness
- `../run-e2e.ps1` - PowerShell test harness
- `README.md` - This documentation
- `TESTING_GUIDE.md` - Detailed testing procedures
- `WINDOWS_SETUP.md` - Windows-specific setup guide

## References

- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Kubectl Reference](https://kubernetes.io/docs/reference/kubectl/)
- [Kustomize Documentation](https://kustomize.io/)
- [Nephoran Intent Operator](../../README.md)