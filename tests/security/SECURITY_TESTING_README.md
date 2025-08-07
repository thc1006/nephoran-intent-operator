# mTLS Security Testing Suite

This directory contains comprehensive security testing and compliance validation for the Nephoran Intent Operator's complete mTLS system implementation.

## Overview

The mTLS Security Testing Suite validates all aspects of the mutual TLS implementation including:

- **Security Testing**: Comprehensive attack simulation and vulnerability testing
- **Integration Testing**: End-to-end validation across all mTLS components  
- **Performance Testing**: Load testing and benchmarking of certificate operations
- **Compliance Validation**: O-RAN, NIST, and TLS 1.3 requirements verification
- **Chaos Engineering**: Failure injection and recovery testing
- **Automation**: Continuous security validation and reporting

## Test Components

### 1. mTLS Security Test Suite (`mtls_security_test.go`)

Comprehensive security testing focusing on:

- **Handshake Testing**
  - Valid certificate scenarios
  - Expired certificate rejection
  - Revoked certificate handling
  - Malformed certificate detection
  - Mutual authentication enforcement
  - Certificate chain validation

- **Certificate Validation**
  - Property validation (validity, key usage, algorithms)
  - Tampering detection
  - Signature verification
  - Key usage constraint validation
  - Subject Alternative Name validation

- **Certificate Rotation**
  - Zero-downtime rotation
  - Coordination between services
  - Metrics validation

- **Attack Simulation**
  - Certificate spoofing attacks
  - Man-in-the-middle prevention
  - TLS downgrade attack resistance
  - Certificate replay attack prevention
  - Certificate enumeration attack handling

- **Service Mesh Integration**
  - Istio mTLS policy enforcement
  - Linkerd automatic encryption validation
  - Consul Connect mTLS verification

### 2. Integration Testing (`mtls_integration_test.go`)

End-to-end integration testing covering:

- **Controller-to-Service Communication**
  - NetworkIntent controller mTLS connections
  - Certificate rotation in live communication
  - Certificate policy validation

- **Service Mesh Integration**
  - Automatic certificate provisioning
  - Policy enforcement validation
  - Cross-mesh communication testing

- **CA Backend Integration**
  - cert-manager integration
  - HashiCorp Vault PKI integration
  - External PKI system integration
  - Multi-backend coordination

- **Certificate Automation**
  - Coordinated rotation across clusters
  - Certificate distribution testing
  - Validation engine integration

- **End-to-End Scenarios**
  - Complete NetworkIntent processing with mTLS
  - Fault injection and recovery testing

### 3. Performance Testing (`mtls_performance_test.go`)

Comprehensive performance validation including:

- **Connection Performance**
  - mTLS handshake benchmarking
  - Concurrent connection testing
  - Sustained load throughput testing
  - Connection pooling efficiency

- **Certificate Operations**
  - Certificate validation performance
  - Certificate rotation performance
  - Certificate cache efficiency

- **Service Mesh Overhead**
  - mTLS overhead measurement
  - Cross-mesh performance testing

- **Stress Testing**
  - High connection churn handling
  - Memory usage under load
  - Certificate rotation under load

### 4. Compliance Validation (`mtls_compliance_test.go`)

Standards compliance verification for:

- **O-RAN Security Requirements**
  - A1 interface security (mTLS, validation, TLS versions)
  - O1 interface security (NETCONF over TLS, certificate authentication)
  - E2 interface security (secure SCTP, xApp authentication)

- **NIST Cybersecurity Framework**
  - Identity (ID) requirements
  - Protect (PR) requirements  
  - Detect (DE) requirements

- **TLS 1.3 Compliance**
  - Mandatory cipher suite support
  - Forward secrecy validation
  - Certificate verification requirements

- **Industry Standards**
  - 3GPP Release 16 security requirements
  - Cloud Security Alliance controls

### 5. Chaos Engineering (`mtls_chaos_test.go`)

Fault injection and resilience testing:

- **Certificate Authority Failures**
  - Complete CA failure scenarios
  - Intermittent CA failures
  - Certificate corruption handling
  - Key compromise response

- **Service Mesh Failures**
  - Control plane failures
  - Certificate rotation failures
  - Sidecar proxy failures

- **Network Failures**
  - Network partitions
  - Intermittent connectivity
  - Packet loss and latency

- **Certificate Lifecycle Chaos**
  - Mass certificate expiration
  - Cascade failure prevention

- **Recovery Testing**
  - Disaster recovery procedures
  - Automated recovery mechanisms

## Usage

### Running Individual Test Suites

```bash
# Run security tests
ginkgo tests/security/mtls_security_test.go

# Run integration tests  
ginkgo tests/security/mtls_integration_test.go

# Run performance tests
ginkgo tests/security/mtls_performance_test.go

# Run compliance validation
ginkgo tests/security/mtls_compliance_test.go

# Run chaos engineering tests
ginkgo tests/security/mtls_chaos_test.go
```

### Running Complete Security Test Suite

```bash
# Run all security tests
ginkgo tests/security/

# Run with detailed output
ginkgo -v tests/security/

# Run with focus on specific scenarios
ginkgo --focus="Attack Simulation" tests/security/

# Run performance benchmarks
go test -bench=. tests/security/mtls_performance_test.go
```

### Automated Security Testing

```bash
# Run security test automation
./run_security_tests.sh

# Run continuous compliance validation
./run_compliance_validation.sh

# Run chaos engineering scenarios
./run_chaos_tests.sh
```

## Configuration

### Test Environment Setup

```yaml
# test-config.yaml
security_testing:
  ca_backend: "self-signed"  # or "cert-manager", "vault"
  service_mesh: "none"       # or "istio", "linkerd", "consul"
  performance_testing:
    load_duration: "60s"
    concurrent_connections: 100
    requests_per_second: 200
  chaos_testing:
    failure_scenarios: ["ca_failure", "network_partition", "cert_corruption"]
    recovery_timeout: "300s"
  compliance:
    standards: ["oran", "nist_csf", "tls13", "3gpp"]
    generate_reports: true
```

### Required Dependencies

- **Kubernetes cluster** with appropriate permissions
- **cert-manager** (optional, for cert-manager integration tests)
- **HashiCorp Vault** (optional, for Vault PKI tests)  
- **Istio/Linkerd/Consul** (optional, for service mesh tests)
- **Chaos engineering tools** (optional, for advanced fault injection)

## Test Results and Reporting

### Performance Benchmarks

Expected performance characteristics:
- **mTLS Handshake P95**: < 100ms
- **Certificate Validation**: > 1000 validations/sec
- **Sustained Throughput**: > 80 requests/sec at 50 concurrent connections
- **Error Rate**: < 1% under normal load
- **Memory Usage**: < 100MB increase under load

### Compliance Scoring

- **O-RAN Compliance**: Target 95%+ compliance score
- **NIST CSF Compliance**: Target 90%+ compliance score  
- **TLS 1.3 Compliance**: Target 100% compliance score
- **Overall Security Score**: Target 90%+ across all standards

### Chaos Engineering Metrics

- **Recovery Time**: < 5 minutes for most scenarios
- **Service Availability**: > 95% during non-critical failures
- **Data Integrity**: 100% preservation during failures
- **Automated Recovery**: > 90% success rate

## Security Test Automation

### Continuous Integration

The security tests integrate with CI/CD pipelines:

```yaml
# .github/workflows/security-tests.yml
- name: Run Security Tests
  run: |
    make test-security-comprehensive
    
- name: Compliance Validation
  run: |
    make validate-compliance
    
- name: Performance Benchmarking
  run: |
    make benchmark-security-performance
```

### Scheduled Security Scans

```bash
# Daily comprehensive security scan
0 2 * * * /usr/local/bin/run_security_tests.sh --comprehensive

# Weekly compliance validation
0 3 * * 0 /usr/local/bin/run_compliance_validation.sh --generate-report

# Monthly chaos engineering
0 4 1 * * /usr/local/bin/run_chaos_tests.sh --full-suite
```

## Troubleshooting

### Common Issues

1. **Certificate Generation Failures**
   - Check CA manager configuration
   - Verify Kubernetes permissions
   - Check certificate storage paths

2. **Service Mesh Integration Issues**
   - Verify service mesh is properly installed
   - Check sidecar injection configuration
   - Validate service mesh mTLS policies

3. **Performance Test Failures**
   - Increase timeout values for slow environments
   - Reduce concurrency for resource-constrained systems
   - Check network connectivity and latency

4. **Compliance Test Failures**
   - Review specific requirement failures in test output
   - Check system configuration against standards
   - Verify all required security features are enabled

### Debug Mode

Enable debug logging for detailed test execution information:

```bash
export SECURITY_TEST_DEBUG=true
export GINKGO_VERBOSE=true
ginkgo -v tests/security/
```

## Security Considerations

### Test Environment Security

- Use isolated test environments
- Rotate test certificates regularly
- Secure test CA private keys
- Limit network access to test clusters

### Test Data Management  

- Generate test certificates dynamically
- Clean up test resources after execution
- Avoid hardcoded credentials or keys
- Use secure random generation for test data

## Contributing

### Adding New Security Tests

1. Follow existing test patterns and structure
2. Include both positive and negative test scenarios
3. Add comprehensive documentation and comments
4. Ensure tests are deterministic and repeatable
5. Include performance and compliance validation where applicable

### Test Coverage Requirements

- All new mTLS features must include security tests
- Maintain >95% test coverage for security-critical code paths
- Include chaos engineering scenarios for new failure modes
- Add compliance validation for relevant security standards

## References

- [O-RAN Security Specifications](https://www.o-ran.org/specifications)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [TLS 1.3 RFC 8446](https://tools.ietf.org/html/rfc8446)
- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)
- [Service Mesh Security Guidelines](https://istio.io/latest/docs/concepts/security/)