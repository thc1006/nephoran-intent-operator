# End-to-End (E2E) Testing Guide

## Overview

This document describes the E2E testing setup for the Nephoran Intent Operator, which validates the complete flow from natural language intent → LLM processing → structured NetworkIntent → Nephio/Porch → CNF scaling in a simulated O-RAN environment.

## Prerequisites

### Required Tools

- **kind** (Kubernetes in Docker): v0.29.0 or later
- **kubectl**: v1.33.0 or later  
- **Go**: 1.24.x
- **Docker**: For kind cluster
- **OpenSSL**: For webhook certificate generation
- **curl**: For API testing

### System Requirements

- **Operating System**: Ubuntu Linux (production target)
- **Memory**: Minimum 8GB RAM (16GB recommended)
- **CPU**: Minimum 4 cores (8 cores recommended)
- **Storage**: 20GB free disk space
- **Network**: Internet access for pulling container images

### Verification Commands

```bash
# Verify prerequisites
kind version
kubectl version --client
go version
docker --version
openssl version
curl --version
```

## Quick Start

### Basic E2E Test Execution

```bash
# Clone and navigate to the project
cd C:\Users\tingy\dev\_worktrees\nephoran\feat-e2e

# Execute complete E2E pipeline
bash hack/run-e2e.sh
```

### With Debugging Options

```bash
# Preserve cluster after tests for debugging
SKIP_CLEANUP=true bash hack/run-e2e.sh

# Enable verbose output
VERBOSE=true bash hack/run-e2e.sh

# Custom cluster configuration
CLUSTER_NAME=my-test NAMESPACE=custom-ns bash hack/run-e2e.sh
```

## Test Pipeline Components

### 1. Infrastructure Setup
- Creates kind cluster with multi-node configuration
- Sets up namespaces (`nephoran-system`, `ran-a`, `ran-b`)
- Installs Custom Resource Definitions (CRDs)
- Configures webhook with self-signed certificates

### 2. Service Deployment
- **intent-ingest**: Processes incoming natural language intents (OFFLINE mode)
- **llm-processor**: Converts intents to structured NetworkIntent (OFFLINE mode)
- **porch-direct**: Manages Kubernetes package lifecycle
- **O-RAN Simulators**: A1/E2/O1 interfaces (logging mode)

### 3. Test Workloads
- Deploys `nf-sim` deployment in `ran-a` namespace
- Initial replica count: 1
- Target for scaling validation

### 4. Scaling Test Execution
- Submits scaling intent: "scale nf-sim to 3 replicas in namespace ran-a"
- Validates intent processing pipeline
- Verifies final deployment state

## Expected Outputs

### Successful Execution

```
🎉 ALL TESTS PASSED! 🎉
E2E Pipeline Validation: SUCCESS
═══════════════════════════════════════════
           E2E TEST RESULTS SUMMARY
═══════════════════════════════════════════
Intent → PackageRevision → Applied → Replicas OK
─────────────────────────────────────────────
scale nf-sim to 3    → ✅ Generated   → ✅ Applied → ✅ 3/3
─────────────────────────────────────────────
```

### Component Status Verification

```bash
# Check cluster status
kubectl cluster-info --context kind-nephoran-e2e

# Verify deployments
kubectl get deployments -n ran-a
kubectl get networkintents -A

# Check component logs
tail -f /tmp/intent-ingest.log
tail -f /tmp/llm-processor.log
tail -f /tmp/porch-direct.log
```

## Verifier Tool Usage

The `tools/verify-scale.go` tool validates deployment scaling completion:

```bash
# Basic usage
go run tools/verify-scale.go \
  --namespace=ran-a \
  --name=nf-sim \
  --target-replicas=3 \
  --timeout=120s

# Extended timeout
go run tools/verify-scale.go \
  --namespace=ran-a \
  --name=nf-sim \
  --target-replicas=5 \
  --timeout=300s
```

### Verifier Output Examples

**Success:**
```
🔍 Verifying scaling for deployment ran-a/nf-sim to 3 replicas (timeout: 2m0s)
⏳ Polling deployment status every 5s...
📊 Current status: spec=1, ready=1, available=1 (target=3)
📊 Current status: spec=3, ready=2, available=2 (target=3)
📊 Current status: spec=3, ready=3, available=3 (target=3)
✨ Target replicas achieved!
✅ Scaling verification successful: ran-a/nf-sim scaled to 3 replicas
```

**Failure:**
```
❌ Scaling verification failed: timeout waiting for scaling (current: spec=3, ready=2, target=3)
```

## Configuration Options

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CLUSTER_NAME` | `nephoran-e2e` | Kind cluster name |
| `NAMESPACE` | `nephoran-system` | Operator namespace |
| `TIMEOUT` | `300` | Wait timeout in seconds |
| `SKIP_CLEANUP` | `false` | Preserve cluster after tests |
| `VERBOSE` | `false` | Enable detailed output |
| `INTENT_INGEST_MODE` | `local` | Run mode: local/sidecar |
| `CONDUCTOR_MODE` | `local` | Run mode: local/in-cluster |
| `PORCH_MODE` | `structured-patch` | Porch mode: structured-patch/direct |

### Command Line Options

```bash
hack/run-e2e.sh [options]

Options:
  --help, -h          Show help message
  --no-cleanup        Don't clean up test resources

Examples:
  hack/run-e2e.sh --no-cleanup
  TIMEOUT=600 hack/run-e2e.sh
```

## Troubleshooting

### Common Issues

#### 1. Kind Cluster Creation Fails
```bash
# Error: failed to create cluster
# Solution: Check Docker daemon and disk space
docker ps
df -h
```

#### 2. CRD Installation Timeout
```bash
# Error: CRD not established
# Solution: Check cluster connectivity
kubectl get nodes
kubectl get crds | grep nephoran
```

#### 3. Webhook Certificate Issues
```bash
# Error: webhook admission failed
# Solution: Verify certificate generation
ls -la /tmp/webhook-certs/
openssl x509 -in /tmp/webhook-certs/tls.crt -text -noout
```

#### 4. Intent Processing Fails
```bash
# Error: intent not processed
# Solution: Check component logs
tail -f /tmp/intent-ingest.log
tail -f /tmp/llm-processor.log
curl -s http://localhost:8080/health
```

### Log Analysis

Component logs are stored in `/tmp/` during test execution:

```bash
# View all component logs
ls -la /tmp/*.log

# Monitor real-time logs
tail -f /tmp/intent-ingest.log
tail -f /tmp/porch-direct.log

# Search for errors
grep -i error /tmp/*.log
grep -i "failed\|timeout" /tmp/*.log
```

### Debug Mode Execution

```bash
# Preserve cluster and enable detailed logging
SKIP_CLEANUP=true VERBOSE=true bash hack/run-e2e.sh

# Manual verification after test
kubectl get all -n ran-a
kubectl describe deployment nf-sim -n ran-a
kubectl logs -l app=nf-sim -n ran-a

# Cleanup when done
kind delete cluster --name nephoran-e2e
```

### Recovery Procedures

#### Reset Test Environment
```bash
# Clean up everything
kind delete cluster --name nephoran-e2e
rm -f /tmp/intent-*.log /tmp/*-sim.log
docker system prune -f

# Restart fresh
bash hack/run-e2e.sh
```

#### Partial Cleanup
```bash
# Reset only test resources
kubectl delete networkintent --all -A
kubectl delete deployment nf-sim -n ran-a
kubectl apply -f /tmp/nf-sim-deployment.yaml
```

## Integration with CI/CD

### GitHub Actions Integration

```yaml
name: E2E Tests
on: [push, pull_request]
jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
      with:
        go-version: '1.24'
    - name: Install kind
      run: |
        curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.29.0/kind-linux-amd64
        chmod +x ./kind && sudo mv ./kind /usr/local/bin/kind
    - name: Run E2E Tests
      run: bash hack/run-e2e.sh
```

### Local Development Workflow

```bash
# 1. Development cycle
go test ./...
golangci-lint run

# 2. Feature validation
bash hack/run-e2e.sh

# 3. Debug specific issues
SKIP_CLEANUP=true bash hack/run-e2e.sh
kubectl describe networkintent <name> -n <namespace>

# 4. Clean up
kind delete cluster --name nephoran-e2e
```

## Performance Expectations

### Timing Benchmarks

| Phase | Expected Duration | Timeout |
|-------|------------------|---------|
| Cluster Creation | 60-90 seconds | 120s |
| CRD Installation | 10-20 seconds | 60s |
| Webhook Setup | 15-30 seconds | 60s |
| Service Startup | 30-45 seconds | 60s |
| Intent Processing | 10-20 seconds | 30s |
| Scaling Verification | 30-60 seconds | 120s |
| **Total Pipeline** | **3-5 minutes** | **10 minutes** |

### Resource Usage

- **Memory**: ~2-4GB during execution
- **CPU**: ~50-80% on 4 cores
- **Storage**: ~5GB for containers and logs
- **Network**: ~500MB for image pulls

## Advanced Usage

### Custom Intent Testing

```bash
# Create custom intent
cat > custom-intent.json << EOF
{
    "apiVersion": "intent.nephoran.com/v1",
    "kind": "NetworkIntent",
    "metadata": {
        "name": "custom-scale",
        "namespace": "ran-b"
    },
    "spec": {
        "intent": "scale my-service to 5 replicas",
        "target": {
            "namespace": "ran-b",
            "workload": "my-service",
            "type": "deployment"
        },
        "scaling": {
            "replicas": 5
        }
    }
}
EOF

# Submit via API
curl -X POST -H "Content-Type: application/json" \
  -d @custom-intent.json \
  http://localhost:8080/intent
```

### Multi-Namespace Testing

```bash
# Setup additional test namespaces
kubectl create namespace test-ns-1
kubectl create namespace test-ns-2

# Deploy workloads to multiple namespaces
kubectl apply -f /tmp/nf-sim-deployment.yaml -n test-ns-1
kubectl apply -f /tmp/nf-sim-deployment.yaml -n test-ns-2
```

This completes the comprehensive E2E testing documentation for the Nephoran Intent Operator.