# End-to-End Testing Guide for Nephoran Intent Operator

This comprehensive guide covers the complete E2E testing framework for the Nephoran Intent Operator, including setup, execution, troubleshooting, and best practices.

## Quick Start

### Windows (PowerShell)
```powershell
# Install prerequisites
choco install kind kubectl golang -y

# Run E2E tests
.\hack\run-e2e.ps1

# Run with custom settings
.\hack\run-e2e.ps1 -ClusterName "my-test" -SkipCleanup -Verbose
```

### Unix/Linux (Bash)
```bash
# Run E2E tests
./hack/run-e2e.sh

# Run with environment variables
CLUSTER_NAME=my-test SKIP_CLEANUP=true ./hack/run-e2e.sh
```

## Overview

The Nephoran Intent Operator E2E testing framework provides comprehensive validation of the complete system workflow:

- **Intent Ingestion**: Natural language → structured NetworkIntent
- **Webhook Validation**: Admission control and schema validation
- **Intent Processing**: Controller reconciliation and event handling
- **Conductor Loop**: KRM patch generation and Porch integration
- **End-to-End Validation**: Complete workflow verification

### Key Features

- ✅ **Cross-platform support** (Windows PowerShell, Unix Bash)
- ✅ **Automated cluster setup** with kind
- ✅ **Comprehensive test reporting** with PASS/FAIL summary
- ✅ **Component isolation** testing (local vs in-cluster)
- ✅ **Integration testing** with Porch and KRM
- ✅ **Sample scenarios** for realistic use cases
- ✅ **Debugging support** with detailed logging

## Test Architecture

### Component Flow

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

### Test Categories

#### 1. Valid Intent Processing
Tests successful intent acceptance and processing

#### 2. Invalid Intent Rejection  
Tests webhook validation for malformed intents

#### 3. Component Integration
Tests component interaction and data flow

#### 4. Porch Integration
Tests package generation and KRM management

## Prerequisites

### Required Tools

#### Core Tools
- **kind** (v0.18.0+): Kubernetes cluster creation
- **kubectl** (v1.25.0+): Kubernetes CLI  
- **Docker**: Container runtime for kind

#### Development Tools (for local mode)
- **Go** (1.21+): Building local components
- **kustomize** (optional): Manifest generation

#### Windows Installation
```powershell
# Using Chocolatey
choco install kind kubectl golang docker-desktop -y
```

#### Linux Installation
```bash
# Install kind
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind && sudo mv ./kind /usr/local/bin/

# Install kubectl  
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl && sudo mv kubectl /usr/local/bin/
```

## Running Tests

### Basic Execution

#### PowerShell (Windows)
```powershell
# Basic run
.\hack\run-e2e.ps1

# Custom configuration
.\hack\run-e2e.ps1 `
  -ClusterName "win-e2e-test" `
  -Namespace "test-system" `
  -IntentIngestMode "local" `
  -ConductorMode "local" `
  -PorchMode "structured-patch" `
  -SkipCleanup `
  -Verbose
```

#### Bash (Unix/Linux)
```bash
# Basic run
./hack/run-e2e.sh

# With environment variables
CLUSTER_NAME=linux-e2e-test \
NAMESPACE=test-system \
INTENT_INGEST_MODE=local \
CONDUCTOR_MODE=local \
PORCH_MODE=structured-patch \
SKIP_CLEANUP=true \
./hack/run-e2e.sh
```

### Configuration Options

#### Environment Variables (Bash)
| Variable | Default | Description |
|----------|---------|-------------|
| `CLUSTER_NAME` | `nephoran-e2e` | Kind cluster name |
| `NAMESPACE` | `nephoran-system` | Test namespace |
| `INTENT_INGEST_MODE` | `local` | `local` or `sidecar` |
| `CONDUCTOR_MODE` | `local` | `local` or `in-cluster` |
| `PORCH_MODE` | `structured-patch` | `structured-patch` or `direct` |
| `SKIP_CLEANUP` | `false` | Skip cluster cleanup |

## Test Scenarios

### 1. Basic NetworkIntent Processing
Tests valid intent acceptance and processing

**File**: `tests/e2e/samples/basic-scale-intent.yaml`

```yaml
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: basic-scale-intent
  namespace: nephoran-system
spec:
  description: "Basic scaling intent for E2E testing"
  intent:
    action: scale
    target: ran-deployment
    parameters:
      replicas: 5
  priority: medium
  owner: e2e-test
```

**Expected Results**:
- ✅ Intent is accepted by webhook
- ✅ Intent appears in cluster
- ✅ Controller processes intent

### 2. Invalid Intent Rejection
Tests webhook validation for malformed intents

**File**: `tests/e2e/samples/invalid-missing-target.yaml`

```yaml
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
spec:
  intent:
    action: scale
    # target: missing-field  # Should cause validation failure
    parameters:
      replicas: 5
```

**Expected Results**:
- ❌ Intent is rejected by webhook
- ✅ Validation error message is clear

## Troubleshooting

### Common Issues

#### Kind Cluster Creation Fails
```bash
# Check Docker is running
docker info

# Try with specific image
KIND_IMAGE=kindest/node:v1.25.3 ./hack/run-e2e.sh
```

#### CRD Installation Fails
```bash
# Check CRD directory
ls -la deployments/crds/

# Manually apply CRDs
kubectl apply -f deployments/crds/
```

#### Webhook Deployment Fails
```bash
# Check webhook config
kubectl kustomize config/webhook

# Check deployment logs
kubectl logs -n nephoran-system deployment/webhook-manager
```

### Debug Mode

Keep cluster for investigation:
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
kubectl get events -n nephoran-system --sort-by='.lastTimestamp'
```

## CI/CD Integration

### GitHub Actions
```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Setup tools
      run: |
        # Install kind, kubectl, go
    - name: Run E2E Tests
      run: ./hack/run-e2e.sh
```

## Documentation

For detailed documentation, see:
- [hack/e2e/README.md](../hack/e2e/README.md) - Complete framework documentation
- [hack/e2e/TESTING_GUIDE.md](../hack/e2e/TESTING_GUIDE.md) - Detailed testing procedures  
- [hack/e2e/WINDOWS_SETUP.md](../hack/e2e/WINDOWS_SETUP.md) - Windows-specific setup

## Support

- **Issues**: https://github.com/nephoran/intent-operator/issues
- **Kind Docs**: https://kind.sigs.k8s.io/
- **Kubectl Docs**: https://kubernetes.io/docs/reference/kubectl/
- **Ginkgo v2**: BDD-style testing framework
- **Gomega**: Matcher library for assertions
- **Kind**: Kubernetes in Docker for local clusters
- **controller-runtime**: Kubernetes client for test interactions

### Directory Structure

```
tests/
├── e2e/
│   ├── main_test.go              # Test suite setup and teardown
│   ├── production_scenarios_test.go # Production-quality test scenarios
│   └── ...
├── integration/
│   └── controllers/
│       └── e2e_workflow_test.go  # Integration workflow tests
hack/
├── run-e2e.sh                    # Basic E2E runner
├── run-production-e2e.sh         # Production-quality E2E runner
├── validate-e2e.sh               # CI validation script
└── check-tools.sh                # Tool availability checker
```

## Test Scenarios

### 1. 5G Core Network Deployment
Tests complete 5G Core deployment with AMF, SMF, UPF, and NSSF components.

**Coverage:**
- High availability configuration
- Auto-scaling policies
- Resource constraints
- Multi-component orchestration

### 2. O-RAN Architecture
Validates O-RAN deployment with RIC and E2 nodes.

**Coverage:**
- Near-RT RIC deployment
- E2NodeSet scaling
- xApp platform setup
- A1/E2/O1 interface validation

### 3. Network Slicing
Tests network slice deployment for different use cases.

**Slice Types:**
- eMBB (Enhanced Mobile Broadband)
- URLLC (Ultra-Reliable Low-Latency)
- mMTC (Massive Machine-Type Communications)

### 4. Auto-scaling and Load Testing
Validates auto-scaling behavior under load.

**Tests:**
- HPA configuration
- Scale-up triggers
- Scale-down behavior
- Load distribution

### 5. Disaster Recovery
Tests failover and recovery scenarios.

**Scenarios:**
- Primary site failure
- Secondary site activation
- Data replication
- Session continuity

### 6. Security and Compliance
Validates security policies and compliance controls.

**Coverage:**
- TLS encryption
- mTLS authentication
- Zero-trust architecture
- Audit logging

## Running Tests

### Prerequisites

```bash
# Required tools
docker --version      # Docker 20.10+
kind --version        # Kind 0.20+
kubectl version       # Kubernetes 1.30+
go version           # Go 1.24+

# Optional tools
kustomize version    # Kustomize 5.0+
helm version         # Helm 3.0+
```

### Quick Start

```bash
# Run basic E2E tests
./hack/run-e2e.sh

# Run production E2E tests
./hack/run-production-e2e.sh

# Run specific test scenario
TEST_SCENARIO=security ./hack/run-production-e2e.sh

# Run with verbose output
VERBOSE=true ./hack/run-production-e2e.sh
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CLUSTER_NAME` | `nephoran-e2e` | Kind cluster name |
| `NAMESPACE` | `nephoran-e2e` | Test namespace |
| `TEST_TIMEOUT` | `30m` | Overall test timeout |
| `PARALLEL_TESTS` | `4` | Number of parallel tests |
| `VERBOSE` | `false` | Enable verbose output |
| `SKIP_CLEANUP` | `false` | Skip resource cleanup |
| `TEST_SCENARIO` | `all` | Test scenario to run |
| `REPORT_DIR` | `./test-reports` | Test report directory |

### Test Scenarios

```bash
# Run all tests
TEST_SCENARIO=all ./hack/run-production-e2e.sh

# Run smoke tests only
TEST_SCENARIO=smoke ./hack/run-production-e2e.sh

# Run production scenarios
TEST_SCENARIO=production ./hack/run-production-e2e.sh

# Run scaling tests
TEST_SCENARIO=scaling ./hack/run-production-e2e.sh

# Run security tests
TEST_SCENARIO=security ./hack/run-production-e2e.sh
```

## CI/CD Integration

### GitHub Actions

```yaml
name: E2E Tests

on:
  pull_request:
    branches: [main]
  push:
    branches: [main]

jobs:
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      
      - name: Setup Kind
        uses: helm/kind-action@v1
        with:
          version: v0.20.0
      
      - name: Run E2E Validation
        run: ./hack/validate-e2e.sh
      
      - name: Run E2E Tests
        run: |
          export CI=true
          export GITHUB_ACTIONS=true
          ./hack/run-production-e2e.sh
      
      - name: Upload Test Reports
        if: always()
        uses: actions/upload-artifact@v3
        with:
          name: e2e-reports
          path: test-reports/
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any
    
    stages {
        stage('Validate') {
            steps {
                sh './hack/validate-e2e.sh'
            }
        }
        
        stage('E2E Tests') {
            steps {
                sh '''
                    export CI=true
                    ./hack/run-production-e2e.sh
                '''
            }
        }
        
        stage('Report') {
            steps {
                junit 'test-reports/*.xml'
                archiveArtifacts 'test-reports/**/*'
            }
        }
    }
    
    post {
        always {
            sh 'kind delete cluster --name nephoran-e2e || true'
        }
    }
}
```

## Writing New Tests

### Test Structure

```go
package e2e

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    // ... other imports
)

var _ = Describe("My E2E Test", Ordered, func() {
    var (
        ctx    context.Context
        cancel context.CancelFunc
    )
    
    BeforeAll(func() {
        ctx, cancel = context.WithTimeout(testCtx, 10*time.Minute)
        // Setup code
    })
    
    AfterAll(func() {
        defer cancel()
        // Cleanup code
    })
    
    Describe("Feature Test", func() {
        It("should perform expected behavior", func() {
            By("Creating a NetworkIntent")
            // Test implementation
            
            By("Verifying the result")
            // Assertions
        })
    })
})
```

### Best Practices

1. **Use Descriptive Names**: Test names should clearly describe what is being tested
2. **Use By() Statements**: Document test steps for better debugging
3. **Set Appropriate Timeouts**: Use Eventually() with reasonable timeouts
4. **Clean Up Resources**: Always clean up test resources in AfterAll
5. **Parallel Safety**: Ensure tests can run in parallel without conflicts
6. **Idempotency**: Tests should produce the same result when run multiple times

### Common Patterns

#### Waiting for Resources

```go
Eventually(func() string {
    intent := &nephoranv1.NetworkIntent{}
    err := k8sClient.Get(ctx, types.NamespacedName{
        Name:      "test-intent",
        Namespace: namespace,
    }, intent)
    if err != nil {
        return ""
    }
    return intent.Status.Phase
}, 5*time.Minute, 10*time.Second).Should(Equal("Ready"))
```

#### Creating Test Resources

```go
intent := &nephoranv1.NetworkIntent{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "test-intent",
        Namespace: namespace,
        Labels: map[string]string{
            "test": "e2e",
        },
    },
    Spec: nephoranv1.NetworkIntentSpec{
        Intent:     "Deploy test network function",
        IntentType: nephoranv1.IntentTypeDeployment,
        Priority:   nephoranv1.PriorityHigh,
    },
}
Expect(k8sClient.Create(ctx, intent)).To(Succeed())
```

## Troubleshooting

### Common Issues

#### 1. Kind Cluster Creation Fails

**Problem:** Kind cluster fails to create
**Solution:**
```bash
# Check Docker is running
docker info

# Clean up existing clusters
kind delete clusters --all

# Retry with custom config
kind create cluster --config kind-config.yaml
```

#### 2. CRD Not Found

**Problem:** NetworkIntent CRD not found
**Solution:**
```bash
# Apply CRDs manually
kubectl apply -f deployments/crds/

# Verify CRDs are established
kubectl get crd | grep nephoran
```

#### 3. Test Timeout

**Problem:** Tests timeout before completion
**Solution:**
```bash
# Increase timeout
TEST_TIMEOUT=60m ./hack/run-production-e2e.sh

# Run specific test
go test ./tests/e2e -run "TestName" -timeout 30m
```

#### 4. Resource Conflicts

**Problem:** Resources from previous test runs cause conflicts
**Solution:**
```bash
# Force cleanup
kubectl delete namespace nephoran-e2e --force --grace-period=0

# Delete and recreate cluster
kind delete cluster --name nephoran-e2e
kind create cluster --name nephoran-e2e
```

### Debug Mode

Enable debug mode for detailed output:

```bash
# Enable verbose logging
export VERBOSE=true
export LOG_LEVEL=debug

# Run with extended timeout and skip cleanup
TEST_TIMEOUT=60m SKIP_CLEANUP=true ./hack/run-production-e2e.sh

# Inspect cluster state after failure
kubectl get all -A
kubectl describe networkintents -A
kubectl logs -n nephoran-system -l app=webhook-manager
```

### Getting Help

For issues and questions:
1. Check the [troubleshooting section](#troubleshooting)
2. Review test logs in `test-reports/`
3. Open an issue on GitHub with:
   - Test output
   - Environment details
   - Steps to reproduce

## Performance Considerations

### Resource Requirements

**Minimum:**
- CPU: 4 cores
- Memory: 8GB RAM
- Disk: 20GB free space

**Recommended:**
- CPU: 8 cores
- Memory: 16GB RAM
- Disk: 50GB free space

### Optimization Tips

1. **Parallel Execution**: Adjust `PARALLEL_TESTS` based on available resources
2. **Cluster Reuse**: Use `SKIP_CLEANUP=true` during development
3. **Selective Testing**: Run specific scenarios instead of all tests
4. **Resource Limits**: Set appropriate resource constraints in test manifests

## Metrics and Reporting

### Test Metrics

The E2E framework tracks:
- Test execution time
- Success/failure rates
- Resource utilization
- Performance baselines

### Report Generation

Reports are generated in `test-reports/`:
- `e2e-junit.xml`: JUnit format for CI integration
- `e2e-report-*.txt`: Detailed test report
- Performance metrics (when available)

### Monitoring During Tests

```bash
# Watch test progress
watch kubectl get networkintents -A

# Monitor pod status
kubectl get pods -n nephoran-e2e -w

# Check controller logs
kubectl logs -n nephoran-system -f deployment/webhook-manager
```

## Contributing

### Adding New Tests

1. Create test file in `tests/e2e/`
2. Follow naming convention: `*_test.go`
3. Use existing test patterns
4. Update documentation
5. Test locally before PR

### Review Checklist

- [ ] Tests pass locally
- [ ] Tests are idempotent
- [ ] Proper cleanup implemented
- [ ] Documentation updated
- [ ] CI validation passes

## References

- [Ginkgo Documentation](https://onsi.github.io/ginkgo/)
- [Gomega Matchers](https://onsi.github.io/gomega/)
- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Kubernetes Testing Best Practices](https://kubernetes.io/docs/reference/using-api/client-libraries/)