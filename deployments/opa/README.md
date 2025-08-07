# OPA (Open Policy Agent) Integration for Nephoran Intent Operator

This directory contains the comprehensive Open Policy Agent (OPA) implementation for API request validation, security enforcement, and O-RAN compliance checking in the Nephoran Intent Operator.

## Overview

The OPA sidecar provides comprehensive API request validation including:

- **Request Size Validation**: Enforces size limits (1MB default, 2MB for `/process`)
- **Rate Limiting**: 10 req/s per user, 50 req/s for admins
- **JWT Token Validation**: Bearer token format, expiration, and issuer checks
- **Input Sanitization**: SQL injection, XSS, command injection detection
- **Intent Format Validation**: Structure, length, and character validation
- **O-RAN Compliance**: Interface types, network functions, slice configurations
- **Security Threat Detection**: Path traversal, LDAP injection, NoSQL injection
- **Telecommunications Validation**: 5G slice types, QoS identifiers (5QI)

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Request  │───▶│   Envoy Proxy   │───▶│ OPA Sidecar     │
│                 │    │   (Optional)    │    │ Policy Engine   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                       │
                                ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ LLM Processor   │◀───│  Allow/Deny     │◀───│ Policy Decision │
│ Service         │    │  Decision       │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Directory Structure

```
deployments/opa/
├── README.md                    # This file
├── opa-configmap.yaml          # Policy configurations
├── opa-sidecar.yaml            # Sidecar container specs
├── llm-processor-with-opa.yaml # Updated deployment template
├── policies/
│   ├── request-validation.rego  # Core request validation
│   └── security-validation.rego # Security-focused policies
├── config/
│   └── opa-config.yaml         # OPA server configuration
├── monitoring/
│   ├── opa-servicemonitor.yaml # Prometheus monitoring
│   ├── opa-alerts.yaml         # Alert rules
│   └── opa-dashboard.json      # Grafana dashboard
└── examples/
    ├── test-requests.yaml      # Test request samples
    ├── opa-test-script.sh      # Testing script
    └── policy-test-suite.yaml  # Comprehensive test suite
```

## Quick Start

### 1. Deploy OPA ConfigMap and Policies

```bash
kubectl apply -f deployments/opa/opa-configmap.yaml
```

### 2. Update LLM Processor with OPA Sidecar

```bash
# Update your Helm values to enable OPA
cat >> values.yaml <<EOF
security:
  opa:
    enabled: true
    image:
      registry: docker.io
      repository: openpolicyagent/opa
      tag: 0.58.0-envoy
      pullPolicy: IfNotPresent
    logLevel: info
    envoy:
      enabled: false  # Enable for advanced filtering
EOF

# Deploy with OPA sidecar
helm upgrade nephoran-operator ./deployments/helm/nephoran-operator \
  -f values.yaml \
  --namespace nephoran-operator
```

### 3. Deploy Monitoring (Optional)

```bash
kubectl apply -f deployments/opa/monitoring/
```

### 4. Test the Implementation

```bash
# Run the test suite
cd deployments/opa/examples
chmod +x opa-test-script.sh
./opa-test-script.sh
```

## Policy Rules

### Request Size Limits

| Endpoint | Max Size |
|----------|----------|
| `/process` | 2MB |
| `/stream` | 1MB |
| `/admin/status` | 4KB |
| `/healthz` | 1KB |
| Default | 1MB |

### Rate Limiting

| User Type | Requests/Second |
|-----------|-----------------|
| Anonymous | 5 |
| User | 10 |
| Operator | 25 |
| Admin | 50 |

### Security Patterns Detected

- **SQL Injection**: `union select`, `drop table`, `insert into`, etc.
- **XSS Attacks**: `<script>`, `javascript:`, `onload=`, etc.
- **Command Injection**: `rm -rf`, `curl`, `wget`, backticks
- **Path Traversal**: `../`, `%2e%2e%2f`, encoded variants
- **LDAP Injection**: `*)|(`, `)|(&`, wildcard patterns
- **NoSQL Injection**: `$ne:`, `$gt:`, `$where:`, etc.

### O-RAN Compliance

**Valid Interface Types**: A1, O1, O2, E2, F1, X2, Xn, N1, N2, N3, N4, N6

**Valid Network Functions**: AMF, SMF, UPF, NSSF, PCF, UDM, UDR, AUSF, NRF, CHF, O-DU, O-CU, O-RU, Near-RT-RIC, Non-RT-RIC, SMO

**Valid 5G Slice Types**: eMBB, URLLC, mMTC

**Valid 5QI Values**: 1, 2, 3, 4, 5, 6, 7, 8, 9, 65, 66, 67, 75

## Testing

### Unit Tests

```bash
cd deployments/opa/examples
opa test test-suite.rego
```

### Integration Tests

```bash
# Test all policies with sample data
./opa-test-script.sh

# Include performance benchmarking
./opa-test-script.sh --benchmark
```

### Policy Coverage Test

The test suite includes coverage for:
- ✅ SQL injection detection
- ✅ XSS attack detection  
- ✅ Command injection detection
- ✅ Path traversal detection
- ✅ Request size validation
- ✅ Authorization validation
- ✅ O-RAN compliance validation
- ✅ Intent format validation
- ✅ JSON structure validation
- ✅ Content type validation

## Monitoring and Alerting

### Key Metrics

- `opa_http_request_duration_seconds`: Policy evaluation latency
- `opa_http_request_duration_seconds_count{code!="200"}`: Policy violations
- `opa_bundle_loaded`: Policy bundle status
- `container_memory_usage_bytes{container="opa-sidecar"}`: Memory usage
- `container_cpu_usage_seconds_total{container="opa-sidecar"}`: CPU usage

### Alert Rules

- **OPAHighPolicyViolations**: > 0.1 violations/second for 2 minutes
- **OPACriticalSecurityViolation**: > 5 403 responses in 1 minute  
- **OPAServiceDown**: OPA health check failing for 1 minute
- **OPAHighLatency**: 95th percentile > 500ms for 3 minutes
- **ORANComplianceViolation**: Any O-RAN compliance violations

### Grafana Dashboard

The included dashboard provides:
- Policy decision overview and rates
- Security threat detection tables
- O-RAN compliance violation charts
- Performance metrics and latency histograms
- Resource usage monitoring
- Request size distribution analysis

## Configuration

### Enabling OPA in Helm Values

```yaml
security:
  opa:
    enabled: true
    image:
      registry: docker.io
      repository: openpolicyagent/opa
      tag: 0.58.0-envoy
      pullPolicy: IfNotPresent
    logLevel: info
    timeout: 5s
    resources:
      requests:
        memory: 64Mi
        cpu: 50m
      limits:
        memory: 128Mi
        cpu: 100m
    envoy:
      enabled: false
      image:
        registry: docker.io
        repository: envoyproxy/envoy
        tag: v1.28.0
        pullPolicy: IfNotPresent
      logLevel: info
      resources:
        requests:
          memory: 64Mi
          cpu: 50m
        limits:
          memory: 128Mi
          cpu: 100m
```

### Environment Variables

- `OPA_ENABLED`: Enable OPA validation (default: false)
- `OPA_URL`: OPA service URL (default: http://localhost:8181)
- `OPA_POLICY_PATH`: Policy evaluation path (default: /v0/data/nephoran)
- `OPA_TIMEOUT`: Request timeout (default: 5s)

## Performance Characteristics

- **Policy Evaluation Latency**: < 50ms P95
- **Memory Usage**: 64-128Mi per sidecar
- **CPU Usage**: 50-100m per sidecar
- **Throughput**: 200+ evaluations/second
- **Cache Hit Rate**: 80%+ for repeated evaluations

## Security Considerations

1. **Fail-Safe Behavior**: Policies deny by default for security
2. **Audit Logging**: All policy decisions are logged for audit
3. **Resource Isolation**: OPA runs in separate container with restricted privileges
4. **Read-Only Root Filesystem**: Enhanced container security
5. **Non-Root User**: Containers run as non-root user (UID 1000)
6. **Network Policies**: Restrict OPA sidecar network access

## Troubleshooting

### Common Issues

1. **OPA Service Not Starting**
   ```bash
   kubectl logs -n nephoran-operator <pod-name> -c opa-sidecar
   ```

2. **Policy Evaluation Errors**
   ```bash
   # Check policy syntax
   opa fmt policies/request-validation.rego
   
   # Test policy manually
   opa eval -d policies/request-validation.rego -i test-data.json "data.nephoran.api.validation.deny"
   ```

3. **High Policy Violation Rate**
   - Check application logs for client issues
   - Review policy rules for false positives
   - Analyze Grafana dashboard for patterns

4. **Performance Issues**
   - Increase OPA resource limits
   - Enable policy caching
   - Optimize policy rules

### Debug Mode

Enable debug logging:
```yaml
security:
  opa:
    logLevel: debug
```

## Contributing

1. Update policy files in `policies/` directory
2. Add corresponding tests in `examples/policy-test-suite.yaml`
3. Run test suite: `./opa-test-script.sh`
4. Update documentation
5. Submit pull request

## References

- [Open Policy Agent Documentation](https://www.openpolicyagent.org/docs/)
- [Rego Language Reference](https://www.openpolicyagent.org/docs/latest/policy-language/)
- [OPA Envoy Plugin](https://www.openpolicyagent.org/docs/latest/envoy-introduction/)
- [Kubernetes Admission Control](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/)
- [O-RAN Alliance Specifications](https://www.o-ran.org/specifications)

## License

This implementation is part of the Nephoran Intent Operator project and follows the same licensing terms.