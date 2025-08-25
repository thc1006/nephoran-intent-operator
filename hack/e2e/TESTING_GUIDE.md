# Comprehensive Testing Guide

This guide provides detailed testing procedures for the Nephoran Intent Operator E2E framework.

## Test Categories

### 1. Unit Tests (Pre-E2E)

Before running E2E tests, ensure unit tests pass:

```bash
# Run all unit tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run specific packages
go test ./pkg/webhooks/...
go test ./pkg/controllers/...
```

### 2. Integration Tests

Test component interactions:

```bash
# Run integration tests
go test -tags=integration ./tests/integration/...

# Run with race detection
go test -race -tags=integration ./tests/integration/...
```

### 3. E2E Tests

Full system tests with real Kubernetes cluster:

```bash
# Basic E2E run
./hack/run-e2e.sh

# With debugging enabled
VERBOSE=true ./hack/run-e2e.sh

# Custom configuration
CLUSTER_NAME=test-cluster NAMESPACE=test-ns ./hack/run-e2e.sh
```

## Test Scenarios

### Scenario 1: Basic NetworkIntent Processing

**Objective**: Verify webhook validation and intent processing

**Setup**:
```bash
./hack/run-e2e.sh
```

**Manual Verification**:
```bash
# 1. Verify webhook is running
kubectl get validatingwebhookconfigurations

# 2. Test valid intent
kubectl apply -f - <<EOF
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: test-scale-intent
  namespace: nephoran-system
spec:
  description: "Scale deployment for load test"
  intent:
    action: scale
    target: test-deployment
    parameters:
      replicas: 5
      resourceProfile: high-performance
  priority: high
  owner: test-user
EOF

# 3. Verify intent was created
kubectl get networkintent test-scale-intent -n nephoran-system

# 4. Check webhook validation logs
kubectl logs -n nephoran-system deployment/webhook-manager
```

**Expected Results**:
- ✅ Valid intent is accepted
- ✅ Intent appears in cluster
- ✅ Webhook logs show validation

### Scenario 2: Invalid Intent Rejection

**Objective**: Verify webhook properly rejects invalid intents

**Test Case**:
```bash
# Should be rejected (missing target)
kubectl apply -f - <<EOF
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: invalid-intent
  namespace: nephoran-system
spec:
  description: "Invalid intent"
  intent:
    action: scale
    # Missing target field
    parameters:
      replicas: -1  # Invalid negative value
EOF
```

**Expected Results**:
- ❌ Intent is rejected with validation error
- ✅ Webhook logs show rejection reason

### Scenario 3: Intent Ingest Processing

**Objective**: Verify intent processing pipeline

**Setup**:
```bash
INTENT_INGEST_MODE=local ./hack/run-e2e.sh
```

**Verification**:
```bash
# 1. Create intent
kubectl apply -f tests/e2e/samples/scale-intent.yaml

# 2. Check for handoff events
kubectl get events -n nephoran-system --field-selector reason=IntentHandoff

# 3. Verify processing logs
kubectl logs -n nephoran-system deployment/intent-ingest
```

### Scenario 4: Conductor Loop Processing

**Objective**: Verify conductor loop generates KRM patches

**Setup**:
```bash
CONDUCTOR_MODE=local PORCH_MODE=structured-patch ./hack/run-e2e.sh
```

**Verification**:
```bash
# 1. Create intent that triggers conductor
kubectl apply -f tests/e2e/samples/scaling-scenario.yaml

# 2. Check for generated patches
ls -la /tmp/nephoran-patches/

# 3. Verify patch content
cat /tmp/nephoran-patches/*.yaml
```

### Scenario 5: Porch Direct Integration

**Objective**: Test direct Porch API integration

**Prerequisites**:
- Porch installed in cluster
- PackageRevision CRDs available

**Setup**:
```bash
PORCH_MODE=direct ./hack/run-e2e.sh
```

**Verification**:
```bash
# 1. Check Porch is accessible
kubectl get packagerevisions -n nephoran-system

# 2. Create intent
kubectl apply -f tests/e2e/samples/porch-intent.yaml

# 3. Verify package creation
kubectl get packagerevisions -n nephoran-system -l nephoran.io/intent=true
```

## Performance Testing

### Load Testing

Test webhook performance under load:

```bash
# Generate load test
for i in {1..100}; do
  kubectl apply -f - <<EOF
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: load-test-$i
  namespace: nephoran-system
spec:
  description: "Load test intent $i"
  intent:
    action: scale
    target: load-test-deployment-$i
    parameters:
      replicas: 3
  priority: medium
  owner: load-test
EOF
done
```

**Metrics to Monitor**:
- Webhook response time
- Memory usage
- CPU utilization
- Event processing rate

### Stress Testing

Test system limits:

```bash
# Create many intents rapidly
seq 1 1000 | xargs -n1 -P10 -I{} kubectl apply -f tests/e2e/samples/stress-intent-{}.yaml
```

## Debugging Procedures

### Debug Failed Tests

1. **Check cluster state**:
```bash
kubectl cluster-info
kubectl get nodes
kubectl get pods -A
```

2. **Examine logs**:
```bash
# Webhook logs
kubectl logs -n nephoran-system deployment/webhook-manager

# Controller logs
kubectl logs -n nephoran-system deployment/controller-manager

# System events
kubectl get events -A --sort-by='.lastTimestamp'
```

3. **Inspect resources**:
```bash
# CRDs
kubectl get crd | grep intent

# NetworkIntents
kubectl get networkintents -A -o yaml

# Webhook configurations
kubectl get validatingwebhookconfigurations -o yaml
```

### Debug Webhook Issues

1. **Check webhook endpoint**:
```bash
kubectl get service -n nephoran-system webhook-service
kubectl describe endpoints -n nephoran-system webhook-service
```

2. **Test webhook directly**:
```bash
# Port-forward to webhook
kubectl port-forward -n nephoran-system service/webhook-service 8443:443

# Test with curl (in another terminal)
curl -k https://localhost:8443/validate-intent
```

3. **Check certificates**:
```bash
kubectl get secret -n nephoran-system webhook-certs
kubectl describe secret -n nephoran-system webhook-certs
```

### Debug Local Components

When running components locally:

1. **Check process status**:
```bash
ps aux | grep -E "(intent-ingest|conductor-loop)"
```

2. **View component logs**:
```bash
# Check stdout/stderr of background processes
tail -f /tmp/intent-ingest.log
tail -f /tmp/conductor-loop.log
```

3. **Test component connectivity**:
```bash
# Test intent-ingest connection
curl -k https://localhost:8443/health

# Test conductor-loop metrics
curl http://localhost:8080/metrics
```

## Test Data Management

### Sample Test Data

Create sample intents in `tests/e2e/samples/`:

```yaml
# basic-scale-intent.yaml
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: basic-scale
  namespace: nephoran-system
spec:
  description: "Basic scaling intent"
  intent:
    action: scale
    target: ran-deployment
    parameters:
      replicas: 3
  priority: medium
  owner: e2e-test
---
# resource-optimization-intent.yaml
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: optimize-resources
  namespace: nephoran-system
spec:
  description: "Optimize resource allocation"
  intent:
    action: optimize
    target: ran-cluster
    parameters:
      cpuTarget: 80
      memoryTarget: 75
  priority: high
  owner: automation
```

### Invalid Test Cases

Create invalid intents for negative testing:

```yaml
# invalid-missing-target.yaml
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: invalid-no-target
  namespace: nephoran-system
spec:
  description: "Missing target field"
  intent:
    action: scale
    # target: missing
    parameters:
      replicas: 5
---
# invalid-negative-replicas.yaml
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: invalid-negative
  namespace: nephoran-system
spec:
  description: "Negative replica count"
  intent:
    action: scale
    target: deployment
    parameters:
      replicas: -5  # Invalid
```

## Continuous Integration

### GitHub Actions Integration

```yaml
name: E2E Tests
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Setup Go
      uses: actions/setup-go@v3
      with:
        go-version: '1.21'
    
    - name: Install kind
      run: |
        curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
        chmod +x ./kind
        sudo mv ./kind /usr/local/bin/kind
    
    - name: Install kubectl
      run: |
        curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
        chmod +x kubectl
        sudo mv kubectl /usr/local/bin/kubectl
    
    - name: Run E2E Tests
      run: ./hack/run-e2e.sh
      env:
        CLUSTER_NAME: "ci-${{ github.run_id }}"
        SKIP_CLEANUP: "false"
    
    - name: Upload test results
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: e2e-test-results
        path: /tmp/test-results/
```

### Test Reporting

Generate test reports:

```bash
# Run with JUnit output
go test -v ./tests/e2e/... -args -test.outputdir=/tmp/test-results

# Convert to HTML
go tool test2json < /tmp/test-results/test.out | tee /tmp/test-results/test.json
```

## Best Practices

### Test Organization

1. **Separate test concerns**:
   - Unit tests: `*_test.go` files
   - Integration tests: `tests/integration/`
   - E2E tests: `tests/e2e/`

2. **Use build tags**:
```go
//go:build e2e
// +build e2e

package e2e
```

3. **Clean test data**:
```bash
# Always clean up resources
defer func() {
    kubectl delete networkintent --all -n test-namespace
}()
```

### Test Isolation

1. **Use unique namespaces**:
```bash
TEST_NAMESPACE="test-$(date +%s)"
kubectl create namespace $TEST_NAMESPACE
```

2. **Clean state between tests**:
```bash
kubectl delete networkintent --all -n $TEST_NAMESPACE
kubectl delete events --all -n $TEST_NAMESPACE
```

3. **Avoid test interdependencies**:
   - Each test should be self-contained
   - Don't rely on test execution order

### Resource Management

1. **Set resource limits**:
```yaml
resources:
  limits:
    memory: "128Mi"
    cpu: "100m"
  requests:
    memory: "64Mi"
    cpu: "50m"
```

2. **Use timeouts**:
```bash
timeout 300s ./hack/run-e2e.sh
```

3. **Monitor resource usage**:
```bash
kubectl top nodes
kubectl top pods -n nephoran-system
```