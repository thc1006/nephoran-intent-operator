# Nephoran Intent Operator - End-to-End Tests

This directory contains comprehensive end-to-end tests for the Nephoran Intent Operator, covering the complete workflow from NetworkIntent creation through LLM processing, RAG integration, and Nephio orchestration.

## Test Structure

### Test Files

1. **`networkintent_lifecycle_test.go`** - Core CRD lifecycle tests
   - CRUD operations for NetworkIntent resources
   - Validation and constraint testing
   - Status progression and condition management
   - Multi-component intent handling

2. **`llm_processor_integration_test.go`** - LLM service integration
   - Health monitoring and metrics endpoints
   - Intent processing with structured output
   - Streaming responses for complex intents
   - Circuit breaker and rate limiting
   - Authentication and authorization

3. **`rag_service_integration_test.go`** - RAG service integration
   - Document processing and indexing
   - Knowledge retrieval and semantic search
   - Intent-specific context generation
   - Knowledge base management
   - Performance and concurrent query handling

4. **`health_monitoring_test.go`** - Health and monitoring tests
   - Controller health and metrics
   - Service endpoint health checks
   - Resource usage monitoring
   - Error condition reporting
   - System recovery validation

5. **`full_workflow_test.go`** - Complete workflow tests
   - End-to-end Intent → LLM → Nephio → Scale workflows
   - Multi-component deployment scenarios
   - Workflow state management and recovery
   - Performance and scalability under load

6. **`error_handling_timeout_test.go`** - Error scenarios and resilience
   - Input validation and malformed data handling
   - Processing timeout scenarios
   - Recovery from temporary failures
   - Resource constraint error handling
   - System stability after error scenarios

7. **`concurrency_cleanup_test.go`** - Concurrency and resource management
   - Concurrent request processing
   - Resource cleanup and lifecycle management
   - Load testing and sustained operations
   - Race condition prevention
   - Memory leak prevention

### Test Suites

- **`basic`**: Core functionality (lifecycle + health)
- **`integration`**: Service integration tests (LLM + RAG)  
- **`workflow`**: Full workflow and error handling tests
- **`stress`**: Concurrency and load testing
- **`cleanup`**: Resource cleanup validation
- **`all`**: Complete test suite

## Prerequisites

### Required Tools
- **Go 1.21+** - For running Ginkgo tests
- **kubectl** - For Kubernetes cluster access
- **Ginkgo v2** - Test framework (`go install github.com/onsi/ginkgo/v2/ginkgo@latest`)

### Optional Services
- **Controller Manager** - Running on `localhost:8080` (required for basic tests)
- **LLM Processor Service** - Running on `localhost:8080` (optional, tests will skip if unavailable)
- **RAG Service** - Running on `localhost:8001` (optional, tests will skip if unavailable)

### Kubernetes Cluster
- Access to a Kubernetes cluster (can be local like kind/minikube or remote)
- Proper RBAC permissions for creating/managing NetworkIntent resources
- CRDs properly installed in the cluster

## Running Tests

### Using PowerShell (Windows)

```powershell
# Run basic test suite
.\run_e2e_tests.ps1 -TestSuite basic

# Run all tests with verbose output
.\run_e2e_tests.ps1 -TestSuite all -Verbose -GenerateReport

# Run integration tests, skip service checks
.\run_e2e_tests.ps1 -TestSuite integration -SkipServiceChecks

# Custom output directory and timeout
.\run_e2e_tests.ps1 -TestSuite workflow -OutputDir ".\custom-results" -Timeout 45

# Cleanup only
.\run_e2e_tests.ps1 -CleanupOnly
```

### Using Bash (Linux/macOS)

```bash
# Make script executable
chmod +x run_e2e_tests.sh

# Run basic test suite
./run_e2e_tests.sh --suite basic

# Run all tests with verbose output  
./run_e2e_tests.sh --suite all --verbose --generate-report

# Run integration tests, skip service checks
./run_e2e_tests.sh --suite integration --skip-service-checks

# Custom namespace and timeout
./run_e2e_tests.sh --suite workflow --namespace nephoran-test --timeout 45

# Cleanup only
./run_e2e_tests.sh --cleanup-only
```

### Direct Ginkgo Execution

```bash
# Run specific test file
ginkgo run --timeout=30m networkintent_lifecycle_test.go

# Run with verbose output and reports
ginkgo run -v --json-report=results.json --junit-report=results.xml full_workflow_test.go

# Run all tests in directory
ginkgo run --timeout=45m ./
```

## Test Configuration

### Environment Variables

- `E2E_TEST_NAMESPACE` - Kubernetes namespace for tests (default: `default`)
- `E2E_TEST_TIMEOUT` - Test timeout in seconds (default: 1800)
- `E2E_OUTPUT_DIR` - Output directory for test results
- `KUBECONFIG` - Path to kubeconfig file

### Test Labels and Annotations

Tests use labels and annotations to organize and control execution:

- `nephoran.io/test-type` - Test category identifier
- `nephoran.io/workflow` - Workflow type being tested
- `nephoran.io/test-timeout` - Enable timeout testing
- `nephoran.io/test-recovery` - Enable recovery testing

## CI/CD Integration

### GitHub Actions Example

```yaml
name: E2E Tests
on: [push, pull_request]

jobs:
  e2e-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.21'
          
      - name: Install Ginkgo
        run: go install github.com/onsi/ginkgo/v2/ginkgo@latest
        
      - name: Setup Kind Cluster
        uses: helm/kind-action@v1.4.0
        
      - name: Install CRDs
        run: kubectl apply -f config/crd/bases/
        
      - name: Run E2E Tests
        run: |
          cd tests/e2e
          ./run_e2e_tests.sh --suite basic --generate-report
          
      - name: Upload Test Results
        uses: actions/upload-artifact@v3
        if: always()
        with:
          name: e2e-test-results
          path: tests/e2e/test-results/
```

### Jenkins Pipeline Example

```groovy
pipeline {
    agent any
    
    environment {
        KUBECONFIG = credentials('kubeconfig')
    }
    
    stages {
        stage('Setup') {
            steps {
                sh 'go install github.com/onsi/ginkgo/v2/ginkgo@latest'
            }
        }
        
        stage('E2E Tests') {
            steps {
                dir('tests/e2e') {
                    sh './run_e2e_tests.sh --suite all --generate-report'
                }
            }
            post {
                always {
                    publishTestResults testResultsPattern: 'tests/e2e/test-results/*.xml'
                    publishHTML([
                        allowMissing: false,
                        alwaysLinkToLastBuild: true,
                        keepAll: true,
                        reportDir: 'tests/e2e/test-results',
                        reportFiles: 'e2e-test-report.html',
                        reportName: 'E2E Test Report'
                    ])
                }
            }
        }
    }
}
```

## Test Results and Reports

### Output Files

- `e2e-test.log` - Detailed execution log (when verbose enabled)
- `*.json` - Ginkgo JSON reports for each test file
- `*.xml` - JUnit XML reports for CI integration  
- `e2e-test-report.html` - Comprehensive HTML report
- `e2e-test-report.json` - Machine-readable test results

### Metrics and Monitoring

Tests automatically collect metrics on:
- Test execution duration
- Resource utilization during tests
- Service response times
- Error rates and patterns
- Concurrent operation success rates

## Troubleshooting

### Common Issues

1. **CRDs Not Found**
   ```bash
   # Install CRDs first
   kubectl apply -f ../../config/crd/bases/
   ```

2. **Service Connection Failures**
   ```bash
   # Check service status
   kubectl get pods -n nephoran-system
   kubectl port-forward service/nephoran-controller 8080:8080
   ```

3. **Permission Errors**
   ```bash
   # Check RBAC permissions
   kubectl auth can-i create networkintents
   kubectl auth can-i get networkintents
   ```

4. **Test Timeouts**
   ```bash
   # Increase timeout
   ./run_e2e_tests.sh --suite basic --timeout 60
   ```

### Debug Mode

Enable debug output for troubleshooting:

```bash
# PowerShell
.\run_e2e_tests.ps1 -TestSuite basic -Verbose

# Bash  
./run_e2e_tests.sh --suite basic --verbose
```

### Selective Test Execution

Run specific test categories:

```bash
# Only controller tests
ginkgo run --focus "NetworkIntent Lifecycle" ./

# Skip integration tests
ginkgo run --skip "Integration|RAG|LLM" ./

# Only error handling
ginkgo run --focus "Error.*Timeout" ./
```

## Development

### Adding New Tests

1. Follow the existing test file naming convention: `*_test.go`
2. Use Ginkgo's `Describe` and `It` blocks for organization
3. Include proper cleanup in `AfterEach` blocks
4. Add appropriate labels and annotations
5. Update the test runner scripts to include new files

### Test Patterns

```go
var _ = Describe("New Feature Tests", func() {
    var (
        ctx context.Context
        namespace string
        testResource *nephoran.NetworkIntent
    )
    
    BeforeEach(func() {
        ctx = context.Background()
        namespace = "default"
    })
    
    AfterEach(func() {
        // Cleanup resources
        if testResource != nil {
            Expect(k8sClient.Delete(ctx, testResource)).Should(Succeed())
        }
    })
    
    Context("When testing new feature", func() {
        It("should behave correctly", func() {
            // Test implementation
        })
    })
})
```

## Contributing

1. Add comprehensive tests for new features
2. Ensure tests are deterministic and don't interfere with each other
3. Include both success and failure scenarios
4. Add proper documentation and examples
5. Update the test runner scripts as needed
6. Verify tests pass in CI environment

## Support

For issues with E2E tests:
1. Check the test logs for detailed error information
2. Verify all prerequisites are installed and accessible
3. Ensure proper cluster access and permissions
4. Review service health and connectivity
5. Open an issue with test logs and environment details