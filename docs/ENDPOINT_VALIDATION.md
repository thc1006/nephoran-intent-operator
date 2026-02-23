# Endpoint Validation at Startup

**Feature**: Startup endpoint validation to prevent runtime DNS errors
**Status**: ✅ Implemented (Task #29)
**Added**: 2026-02-23

## Problem

Before this feature, misconfigured endpoints would cause errors at first use (during NetworkIntent reconciliation), leading to:
- Misleading "DNS lookup failed" errors in logs
- Delayed failure discovery (only when first resource is processed)
- Poor operator experience with cryptic error messages

## Solution

Add comprehensive endpoint validation at operator startup with:
1. URL format validation
2. Scheme validation (http/https only, prevents SSRF)
3. Optional DNS resolution checking
4. Clear error messages with configuration suggestions

## Usage

### Basic Startup (Default Behavior)

The operator validates all configured endpoints automatically at startup:

```bash
./nephoran-operator \
  --a1-endpoint=http://a1-mediator:8080 \
  --llm-endpoint=http://ollama:11434 \
  --porch-server=http://porch-server:7007
```

**Default validation** (fast, no network calls):
- ✅ URL format
- ✅ Scheme (http/https only)
- ✅ Hostname presence
- ❌ DNS resolution (disabled)
- ❌ HTTP connectivity (disabled)

### Enable DNS Validation

For stricter validation during debugging:

```bash
./nephoran-operator \
  --a1-endpoint=http://a1-mediator:8080 \
  --validate-endpoints-dns
```

This will check DNS resolution for all configured endpoints, catching hostname typos early.

### Configuration via Environment Variables

Endpoints can be configured via environment variables (flags take precedence):

```bash
export A1_MEDIATOR_URL="http://service-ricplt-a1mediator-http.ricplt:8080"
export LLM_PROCESSOR_URL="http://ollama-service.ollama.svc.cluster.local:11434"
export PORCH_SERVER_URL="http://porch-server.porch-system.svc.cluster.local:7007"
export RAG_API_URL="http://rag-service.rag-service.svc.cluster.local:8000"

./nephoran-operator
```

## Error Messages

### Example 1: Invalid URL Scheme

**Before** (runtime error during reconciliation):
```
Error reconciling NetworkIntent: Post "ftp://a1-mediator:8080/policies": unsupported protocol scheme "ftp"
```

**After** (startup validation):
```
2026-02-23T10:30:00.000Z ERROR Endpoint validation failed at startup
Endpoint validation failed:

A1 Mediator endpoint validation failed: unsupported URL scheme 'ftp' (only http/https allowed)
  → Set A1_MEDIATOR_URL environment variable or --a1-endpoint flag.
     Example: http://service-ricplt-a1mediator-http.ricplt:8080

Fix configuration and restart the operator.
```

### Example 2: Missing Hostname

**Before**:
```
Error reconciling NetworkIntent: dial tcp: missing address
```

**After**:
```
2026-02-23T10:30:00.000Z ERROR Endpoint validation failed at startup
Endpoint validation failed:

LLM Service endpoint validation failed: missing hostname in URL
  → Set LLM_PROCESSOR_URL environment variable or --llm-endpoint flag.
     Example: http://ollama-service:11434

Fix configuration and restart the operator.
```

### Example 3: DNS Resolution Failure (with --validate-endpoints-dns)

**Before**:
```
Error reconciling NetworkIntent: dial tcp: lookup bad-hostname: no such host
```

**After**:
```
2026-02-23T10:30:00.000Z ERROR Endpoint validation failed at startup
Endpoint validation failed:

Porch Server endpoint validation failed: cannot resolve hostname (DNS lookup failed):
lookup bad-hostname: no such host. Check if service is deployed and DNS is configured
  → Set PORCH_SERVER_URL environment variable or --porch-server flag.
     Example: http://porch-server:7007

Fix configuration and restart the operator.
```

## Configuration Reference

### Command-Line Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--a1-endpoint` | A1 Mediator endpoint (overrides `A1_MEDIATOR_URL`) | `http://a1-mediator:8080` |
| `--llm-endpoint` | LLM processor endpoint (overrides `LLM_PROCESSOR_URL`) | `http://ollama:11434` |
| `--porch-server` | Porch server endpoint (overrides `PORCH_SERVER_URL`) | `http://porch-server:7007` |
| `--validate-endpoints-dns` | Enable DNS resolution checking at startup | (boolean flag) |

### Environment Variables

| Variable | Alternative | Description |
|----------|-------------|-------------|
| `A1_MEDIATOR_URL` | `A1_ENDPOINT` | A1 Policy Management Service endpoint |
| `LLM_PROCESSOR_URL` | `LLM_ENDPOINT` | LLM inference endpoint |
| `PORCH_SERVER_URL` | - | Nephio Porch server endpoint |
| `RAG_API_URL` | - | RAG service endpoint |

**Precedence**: Flags > Environment variables

## Deployment

### Kubernetes Deployment

Update your deployment YAML to include environment variables:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephoran-operator
spec:
  template:
    spec:
      containers:
      - name: manager
        image: nephoran-operator:latest
        env:
        - name: A1_MEDIATOR_URL
          value: "http://service-ricplt-a1mediator-http.ricplt:8080"
        - name: LLM_PROCESSOR_URL
          value: "http://ollama-service.ollama.svc.cluster.local:11434"
        - name: PORCH_SERVER_URL
          value: "http://porch-server.porch-system.svc.cluster.local:7007"
        - name: RAG_API_URL
          value: "http://rag-service.rag-service.svc.cluster.local:8000"
        args:
        - --leader-elect
        # Optionally enable DNS validation (may slow startup)
        # - --validate-endpoints-dns
```

### Helm Chart

```yaml
# values.yaml
endpoints:
  a1Mediator: "http://service-ricplt-a1mediator-http.ricplt:8080"
  llmProcessor: "http://ollama-service.ollama.svc.cluster.local:11434"
  porchServer: "http://porch-server.porch-system.svc.cluster.local:7007"
  ragService: "http://rag-service.rag-service.svc.cluster.local:8000"

validation:
  enableDNS: false  # Set to true for stricter validation
```

```yaml
# templates/deployment.yaml
env:
- name: A1_MEDIATOR_URL
  value: {{ .Values.endpoints.a1Mediator | quote }}
- name: LLM_PROCESSOR_URL
  value: {{ .Values.endpoints.llmProcessor | quote }}
- name: PORCH_SERVER_URL
  value: {{ .Values.endpoints.porchServer | quote }}
- name: RAG_API_URL
  value: {{ .Values.endpoints.ragService | quote }}

args:
- --leader-elect
{{- if .Values.validation.enableDNS }}
- --validate-endpoints-dns
{{- end }}
```

## Testing

### Unit Tests

```bash
# Test endpoint validator
go test -v ./pkg/validation/...

# Test main.go validation integration
go test -v ./cmd/... -run TestValidateEndpoints
```

### Manual Testing

#### Test Invalid Scheme
```bash
./nephoran-operator --a1-endpoint="ftp://bad:8080"
# Expected: Startup fails with clear error message
```

#### Test Missing Hostname
```bash
./nephoran-operator --llm-endpoint="http://"
# Expected: Startup fails with "missing hostname" error
```

#### Test DNS Validation
```bash
./nephoran-operator \
  --a1-endpoint="http://nonexistent-host:8080" \
  --validate-endpoints-dns
# Expected: Startup fails with DNS resolution error
```

#### Test Valid Configuration
```bash
./nephoran-operator \
  --a1-endpoint="http://localhost:8080" \
  --llm-endpoint="http://localhost:11434"
# Expected: Startup succeeds, validation passes
```

## Performance Impact

| Validation Level | Startup Overhead | When to Use |
|-----------------|------------------|-------------|
| **Default** (URL format only) | ~1μs per endpoint | Production deployments |
| **DNS Enabled** (`--validate-endpoints-dns`) | ~10-100ms per hostname | Debugging, CI/CD validation |

**Recommendation**: Use default validation for production, enable DNS checking when troubleshooting configuration issues.

## Security Benefits

1. **SSRF Prevention**: Only http/https schemes allowed (blocks file://, ftp://, etc.)
2. **Fail-Fast**: Invalid configuration caught at startup, not during reconciliation
3. **Clear Error Messages**: Operators know exactly how to fix configuration
4. **Input Validation**: All user-provided URLs validated before use

## Related Files

- Implementation: `/pkg/validation/endpoint_validator.go`
- Tests: `/pkg/validation/endpoint_validator_test.go`
- Integration: `/cmd/main.go` (validateEndpoints function)
- Integration Tests: `/cmd/main_validation_test.go`
- Documentation: `/pkg/validation/README.md`

## Architecture Decision Records

### Why DNS Validation is Optional

**Decision**: DNS validation is disabled by default.

**Rationale**:
1. **Startup Speed**: DNS lookups add 10-100ms per hostname
2. **Kubernetes DNS**: Service names may not resolve until after pod startup
3. **CI/CD**: Tests may run before services are deployed
4. **Sufficient**: URL format validation catches most configuration errors

**When to Enable**: Debugging production issues, validating complex multi-cluster setups.

### Why Reachability Check is Disabled

**Decision**: HTTP reachability checking is not enabled (even with DNS flag).

**Rationale**:
1. **Service Startup Order**: Operator may start before dependent services
2. **Health Probes**: Kubernetes liveness/readiness probes handle this better
3. **Timeout Issues**: Would require long timeouts, slowing startup significantly
4. **False Positives**: Service may be temporarily unavailable during deployment

**Alternative**: Services fail gracefully at runtime with retry logic (already implemented in reconciler).

## Future Enhancements

- [ ] Configuration file support (YAML/JSON)
- [ ] Endpoint health check command (`nephoran-operator check-endpoints`)
- [ ] Prometheus metrics for endpoint validation failures
- [ ] Integration with Kubernetes validating webhooks
- [ ] Auto-discovery of service endpoints from ConfigMap/Secret

## References

- Task #29: Add startup endpoint validation to prevent runtime DNS errors
- ARCHITECTURAL_HEALTH_ASSESSMENT.md: Section 3.2 (Missing Configuration Validation)
- Security Audit: SSRF prevention measures
