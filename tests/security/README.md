# Security Testing Suite

This directory contains comprehensive security tests for the Nephoran Intent Operator, validating security controls across multiple domains including container security, RBAC, network policies, secrets management, and TLS/mTLS configurations.

## Test Structure

```
tests/security/
├── container_security_test.go  # Container security validation
├── rbac_test.go               # RBAC and privilege management tests  
├── network_policy_test.go     # Zero-trust network policy tests
├── secrets_test.go            # Secrets management and encryption tests
├── tls_test.go               # TLS/mTLS certificate and protocol tests
├── suite_test.go             # Test suite configuration
├── run_security_tests.sh     # Comprehensive test runner script
└── README.md                 # This file
```

## Test Categories

### 1. Container Security Tests (`container_security_test.go`)

Validates container runtime security configurations:

- **Non-root User Verification**: Ensures all containers run as non-root users
- **Read-only Root Filesystems**: Validates containers use read-only root filesystems
- **Capability Management**: Verifies containers drop all capabilities and don't run privileged
- **Security Context Validation**: Checks pod and container security contexts
- **Image Security**: Validates image tags, registries, and vulnerability scan integration
- **Resource Limits**: Ensures proper resource constraints are configured
- **Pod Security Standards**: Validates namespace-level pod security policies

### 2. RBAC Tests (`rbac_test.go`)

Validates role-based access control and privilege management:

- **Service Account Security**: Verifies proper service account configuration
- **Least Privilege Policies**: Validates roles follow principle of least privilege  
- **Role Binding Verification**: Ensures correct role-to-subject mappings
- **Privilege Escalation Prevention**: Prevents unauthorized privilege elevation
- **Namespace Isolation**: Validates cross-namespace access controls
- **RBAC Manager Integration**: Tests custom RBAC management functionality

### 3. Network Policy Tests (`network_policy_test.go`)

Validates zero-trust network security:

- **Default Deny Policies**: Ensures baseline network denial policies exist
- **Component Communication**: Validates inter-service communication restrictions
- **Egress Controls**: Tests outbound traffic limitations and controls
- **Multi-namespace Policies**: Validates cross-namespace network isolation
- **O-RAN Interface Policies**: Tests O-RAN specific network configurations
- **Service Mesh Integration**: Validates Istio/service mesh policy integration

### 4. Secrets Management Tests (`secrets_test.go`)

Validates secrets protection and lifecycle:

- **Encryption at Rest**: Verifies secrets are encrypted when stored
- **Secret Rotation**: Tests automatic secret rotation capabilities
- **TLS Certificate Validation**: Validates certificate properties and chains
- **External Secret Stores**: Tests integration with external key management
- **Access Controls**: Validates secret access permissions and audit logging
- **Secure String Handling**: Tests secure memory operations for sensitive data

### 5. TLS/mTLS Tests (`tls_test.go`)

Validates transport layer security:

- **Certificate Validation**: Verifies TLS certificate properties and validity
- **Mutual Authentication**: Tests mTLS configuration between services
- **Certificate Rotation**: Validates automatic certificate renewal processes
- **Protocol Security**: Tests TLS versions, cipher suites, and security headers
- **Certificate Authority**: Validates CA certificate chains and trust relationships
- **Compliance Checks**: Ensures adherence to TLS security best practices

## Running Tests

### Quick Start

```bash
# Run all security tests
cd tests/security
./run_security_tests.sh

# Run with verbose output and coverage
./run_security_tests.sh -v --coverage

# Run specific test categories
./run_security_tests.sh --container-only
./run_security_tests.sh --rbac-only  
./run_security_tests.sh --network-only
```

### Advanced Usage

```bash
# Custom namespace and timeout
./run_security_tests.sh -n my-test-namespace -t 45m

# Generate specific report formats
./run_security_tests.sh --report json --report html

# Skip environment setup/cleanup
./run_security_tests.sh --skip-setup --skip-cleanup

# Parallel execution
./run_security_tests.sh -p 8
```

### Using Go Test Directly

```bash
# Run specific test file
go test -v ./tests/security/container_security_test.go

# Run with Ginkgo
ginkgo -v ./tests/security/

# Generate coverage report
ginkgo -v --cover --coverprofile=coverage.out ./tests/security/
```

## Configuration

### Environment Variables

- `TEST_NAMESPACE`: Kubernetes namespace for tests (default: `nephoran-intent-operator-test`)
- `KUBECONFIG`: Path to kubeconfig file (default: `~/.kube/config`)  
- `USE_EXISTING_CLUSTER`: Use existing cluster instead of envtest (default: `false`)
- `PARALLEL_TESTS`: Number of parallel test processes (default: `4`)
- `TEST_TIMEOUT`: Test timeout duration (default: `30m`)
- `VERBOSE`: Enable verbose output (default: `false`)

### Prerequisites

- Go 1.24+
- kubectl configured with cluster access
- Ginkgo test framework
- Optional: Trivy for vulnerability scanning
- Optional: cert-manager for certificate management testing

## Test Reports

The test runner generates comprehensive reports:

```
test-results/security/
├── reports/
│   ├── junit.xml                    # JUnit XML for CI integration
│   ├── security-test-results.json   # Structured JSON results  
│   ├── security-test-report.html    # HTML dashboard
│   ├── coverage.html                # Code coverage report
│   ├── compliance-report.txt        # Security compliance summary
│   └── vulnerability-scan.json     # Container vulnerability scan
├── coverage/
│   └── security.coverprofile       # Go coverage profile
└── security-tests.log              # Detailed test execution logs
```

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Security Tests
on: [push, pull_request]

jobs:
  security-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.24'
      
      - name: Setup test cluster
        uses: helm/kind-action@v1
        
      - name: Run security tests
        run: |
          cd tests/security
          ./run_security_tests.sh --report xml --coverage
          
      - name: Upload test results
        uses: actions/upload-artifact@v3
        with:
          name: security-test-results
          path: test-results/security/
```

### Jenkins Pipeline Example

```groovy
pipeline {
    agent any
    stages {
        stage('Security Tests') {
            steps {
                sh '''
                    cd tests/security
                    ./run_security_tests.sh --report xml --coverage
                '''
            }
            post {
                always {
                    junit 'test-results/security/reports/junit.xml'
                    publishHTML([
                        allowMissing: false,
                        alwaysLinkToLastBuild: true,
                        keepAll: true,
                        reportDir: 'test-results/security/reports',
                        reportFiles: 'security-test-report.html',
                        reportName: 'Security Test Report'
                    ])
                }
            }
        }
    }
}
```

## Security Standards Compliance

These tests validate compliance with:

- **NIST Cybersecurity Framework**: Core security controls and practices
- **CIS Kubernetes Benchmark**: Container and Kubernetes security hardening
- **OWASP Container Security**: Application container security best practices  
- **Pod Security Standards**: Kubernetes native pod security policies
- **O-RAN Security Guidelines**: Telecommunications-specific security requirements

## Customization

### Adding Custom Tests

1. Create test file following naming pattern `*_test.go`
2. Use Ginkgo BDD framework for test structure
3. Import utilities from `tests/utils` package
4. Add test to runner script if needed

### Extending Security Checks

1. Add validation functions to `tests/utils/security_helpers.go`
2. Create new test contexts in existing test files
3. Update runner script with new test categories
4. Document new checks in this README

### Custom Compliance Frameworks

1. Create compliance check script in `tests/security/compliance/`
2. Define compliance rules and validation logic
3. Integrate with main test runner
4. Generate compliance-specific reports

## Troubleshooting

### Common Issues

**Test Environment Setup Fails**
- Verify kubectl connectivity: `kubectl cluster-info`
- Check RBAC permissions for test namespace creation
- Ensure CRDs are properly installed

**Certificate Validation Failures**  
- Verify cert-manager is installed if using automatic certificates
- Check certificate expiry dates and renewal policies
- Validate CA certificate chains

**Network Policy Test Failures**
- Confirm CNI supports network policies (not all do)
- Check if default deny policies conflict with system pods
- Verify namespace isolation is properly configured

**RBAC Test Failures**
- Ensure test service account has appropriate permissions
- Check for conflicting cluster roles or bindings
- Verify namespace-level RBAC is properly configured

### Debug Mode

Enable debug output for troubleshooting:

```bash
./run_security_tests.sh -v --skip-cleanup
```

This provides detailed test execution logs and preserves test resources for inspection.

## Contributing

1. Follow existing test patterns and naming conventions
2. Include both positive and negative test cases  
3. Add comprehensive documentation for new test categories
4. Ensure tests are idempotent and can run in any order
5. Update this README with new test descriptions

## Security Considerations

- Tests create temporary resources in test namespaces
- Sensitive data is handled securely during testing
- Test environments are isolated from production
- Cleanup procedures remove all test artifacts
- Audit trails are maintained for security test execution