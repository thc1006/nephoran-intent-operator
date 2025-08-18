# Conductor-Watch Security Documentation

## Executive Summary

Conductor-Watch is a secure file monitoring and intent processing component designed with defense-in-depth security principles. This document provides comprehensive security configuration guidelines, deployment recommendations, and operational best practices for the conductor-watch component in production environments.

## Table of Contents

1. [Security Architecture Overview](#security-architecture-overview)
2. [TLS/HTTPS Configuration](#tlshttps-configuration)
3. [Authentication Methods](#authentication-methods)
4. [Resource Limits and Rate Limiting](#resource-limits-and-rate-limiting)
5. [Security Headers and Request Signing](#security-headers-and-request-signing)
6. [File Processing Security](#file-processing-security)
7. [Deployment Security](#deployment-security)
8. [Security Monitoring and Auditing](#security-monitoring-and-auditing)
9. [Security Checklist](#security-checklist)
10. [Troubleshooting Guide](#troubleshooting-guide)

## Security Architecture Overview

### Core Security Features

Conductor-Watch implements multiple layers of security controls:

- **Transport Security**: TLS 1.2+ with secure cipher suites, optional mTLS
- **Authentication**: Bearer tokens, API keys, HMAC signing, JWT support
- **Authorization**: RBAC integration with Kubernetes
- **Input Validation**: JSON schema validation, file size limits, extension filtering
- **Resource Protection**: Rate limiting, concurrent request limits, memory bounds
- **Audit Logging**: Comprehensive security event logging with structured formats

### Security Components

```
┌─────────────────────────────────────────────┐
│            External Clients                 │
└─────────────────┬───────────────────────────┘
                  │ TLS/mTLS
┌─────────────────▼───────────────────────────┐
│         SecureHTTPClient                    │
│  • Rate Limiting                            │
│  • Connection Pooling                       │
│  • Request Signing (HMAC)                   │
└─────────────────┬───────────────────────────┘
                  │
┌─────────────────▼───────────────────────────┐
│         SecureWatcher                       │
│  • File Validation                          │
│  • Debouncing                               │
│  • Worker Pool Management                   │
└─────────────────┬───────────────────────────┘
                  │
┌─────────────────▼───────────────────────────┐
│         Intent Processor                    │
│  • Schema Validation                        │
│  • Namespace Isolation                      │
│  • Audit Trail                              │
└──────────────────────────────────────────────┘
```

## TLS/HTTPS Configuration

### Basic TLS Configuration

```go
// config/tls-config.go
type TLSConfig struct {
    // Minimum TLS version (default: TLS 1.2)
    MinVersion uint16 `json:"minVersion,omitempty"`
    
    // CA certificate for server verification
    CAFile string `json:"caFile,omitempty"`
    
    // Client certificate for mTLS
    CertFile string `json:"certFile,omitempty"`
    KeyFile  string `json:"keyFile,omitempty"`
    
    // Secure cipher suites only
    CipherSuites []uint16 `json:"cipherSuites,omitempty"`
}
```

### Recommended TLS Settings

```yaml
# config/conductor-watch.yaml
tls:
  enabled: true
  minVersion: "TLS1.2"  # or "TLS1.3" for enhanced security
  caFile: "/etc/conductor-watch/certs/ca.crt"
  
  # Secure cipher suites (automatically configured)
  # - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
  # - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
  # - TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
  # - TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
```

### mTLS Configuration

For enhanced security with mutual TLS authentication:

```yaml
# config/conductor-watch-mtls.yaml
tls:
  enabled: true
  mode: "mutual"
  minVersion: "TLS1.3"
  
  # Server verification
  caFile: "/etc/conductor-watch/certs/ca.crt"
  
  # Client authentication
  certFile: "/etc/conductor-watch/certs/client.crt"
  keyFile: "/etc/conductor-watch/certs/client.key"
  
  # Additional validation
  validateHostname: true
  allowedCNs:
    - "conductor-watch.nephoran.local"
    - "*.nephoran.local"
```

### Certificate Generation

```bash
#!/bin/bash
# generate-certs.sh

# Generate CA
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt \
  -subj "/C=US/O=Nephoran/CN=Nephoran CA"

# Generate client certificate
openssl genrsa -out client.key 4096
openssl req -new -key client.key -out client.csr \
  -subj "/C=US/O=Nephoran/CN=conductor-watch"
openssl x509 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out client.crt

# Verify certificate chain
openssl verify -CAfile ca.crt client.crt
```

## Authentication Methods

### API Key Authentication

```go
// Implementation in secure_client.go
type AuthConfig struct {
    AuthType     string `json:"authType"`     // "apikey"
    APIKeyHeader string `json:"apiKeyHeader"` // "X-API-Key"
    AuthToken    string `json:"authToken"`    // Actual key value
}
```

Configuration example:
```yaml
authentication:
  type: "apikey"
  header: "X-API-Key"
  token: "${API_KEY}"  # Environment variable
```

### Bearer Token Authentication

```yaml
authentication:
  type: "bearer"
  token: "${BEARER_TOKEN}"
  # Automatically adds: Authorization: Bearer <token>
```

### HMAC Request Signing

For request integrity and authentication:

```go
// HMAC signing implementation
func (w *SecureWatcher) signRequest(payload []byte) string {
    h := hmac.New(sha256.New, w.config.HMACSecret)
    h.Write(payload)
    return hex.EncodeToString(h.Sum(nil))
}
```

Configuration:
```yaml
authentication:
  type: "hmac"
  secret: "${HMAC_SECRET}"  # 32+ byte secret
  algorithm: "SHA256"
  headerName: "X-HMAC-SHA256"
```

## Resource Limits and Rate Limiting

### File Processing Limits

```go
// SecureWatcherConfig limits
type SecureWatcherConfig struct {
    MaxFileSize       int64         // Default: 5MB
    MaxWorkers        int           // Default: 10
    AllowedExtensions []string      // Default: [".json"]
    DebounceDelay     time.Duration // Default: 300ms
}
```

### HTTP Client Limits

```go
// SecureClientConfig limits
type SecureClientConfig struct {
    MaxConcurrent   int           // Max concurrent requests (default: 10)
    MaxBodySize     int64         // Max response size (default: 10MB)
    Timeout         time.Duration // Request timeout (default: 30s)
    RateLimit       float64       // Requests per second (default: 10.0)
    BurstSize       int           // Burst capacity (default: 20)
    
    // Connection pool settings
    MaxIdleConns    int           // Default: 100
    MaxConnsPerHost int           // Default: 10
    IdleConnTimeout time.Duration // Default: 90s
}
```

### Rate Limiting Implementation

The component uses golang.org/x/time/rate for token bucket rate limiting:

```go
// Rate limiter initialization
rateLimiter := rate.NewLimiter(
    rate.Limit(config.RateLimit), // 10 req/s
    config.BurstSize,              // burst of 20
)

// Usage in request handling
if err := rateLimiter.Wait(ctx); err != nil {
    return fmt.Errorf("rate limit exceeded: %w", err)
}
```

## Security Headers and Request Signing

### Outgoing Request Headers

```go
// Headers added to all requests
headers := map[string]string{
    "User-Agent":           "conductor-watch/1.0",
    "X-Request-ID":         uuid.New().String(),
    "X-Timestamp":          time.Now().Format(time.RFC3339),
    "X-Nephoran-Component": "conductor-watch",
}
```

### Request Signing Process

1. Generate request payload
2. Create HMAC signature
3. Add signature header
4. Include timestamp for replay protection

```go
func (c *SecureHTTPClient) signedRequest(method, url string, body []byte) (*http.Request, error) {
    req, err := http.NewRequest(method, url, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    
    // Add timestamp
    timestamp := time.Now().Format(time.RFC3339)
    req.Header.Set("X-Timestamp", timestamp)
    
    // Create signature
    message := fmt.Sprintf("%s\n%s\n%s\n%s", method, url, timestamp, string(body))
    signature := hmacSign(message, c.config.HMACSecret)
    req.Header.Set("X-HMAC-SHA256", signature)
    
    return req, nil
}
```

## File Processing Security

### Input Validation

```go
// File validation checks
func (w *SecureWatcher) validateFile(path string) error {
    // Check file extension
    if !w.isAllowedExtension(path) {
        return fmt.Errorf("invalid file extension")
    }
    
    // Check file size
    info, err := os.Stat(path)
    if err != nil {
        return err
    }
    if info.Size() > w.config.MaxFileSize {
        return fmt.Errorf("file too large: %d > %d", info.Size(), w.config.MaxFileSize)
    }
    
    // Validate JSON schema
    content, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    return w.validator.Validate(content)
}
```

### Schema Validation

The component uses JSON schema validation for all intent files:

```go
// Validator implementation
type Validator struct {
    schema *jsonschema.Schema
}

func (v *Validator) Validate(data []byte) error {
    var intent map[string]interface{}
    if err := json.Unmarshal(data, &intent); err != nil {
        return fmt.Errorf("invalid JSON: %w", err)
    }
    
    if err := v.schema.Validate(intent); err != nil {
        return fmt.Errorf("schema validation failed: %w", err)
    }
    
    return nil
}
```

### Debouncing and Deduplication

Prevents processing the same file multiple times:

```go
// Debouncing implementation
func (w *SecureWatcher) debounceFileEvent(path string) {
    w.mu.Lock()
    defer w.mu.Unlock()
    
    // Cancel existing timer
    if timer, exists := w.pending[path]; exists {
        timer.Stop()
    }
    
    // Create new timer
    w.pending[path] = time.AfterFunc(w.config.DebounceDelay, func() {
        w.processFile(path)
        w.mu.Lock()
        delete(w.pending, path)
        w.mu.Unlock()
    })
}
```

## Deployment Security

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: conductor-watch
  namespace: nephoran-system
spec:
  replicas: 2  # HA configuration
  template:
    spec:
      serviceAccountName: conductor-watch
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534  # nobody
        fsGroup: 65534
        seccompProfile:
          type: RuntimeDefault
      
      containers:
      - name: conductor-watch
        image: nephoran/conductor-watch:1.0.0
        imagePullPolicy: Always
        
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          capabilities:
            drop:
              - ALL
        
        resources:
          limits:
            cpu: "500m"
            memory: "512Mi"
            ephemeral-storage: "1Gi"
          requests:
            cpu: "100m"
            memory: "128Mi"
        
        volumeMounts:
        - name: config
          mountPath: /etc/conductor-watch
          readOnly: true
        - name: certs
          mountPath: /etc/conductor-watch/certs
          readOnly: true
        - name: handoff
          mountPath: /handoff
        - name: tmp
          mountPath: /tmp
        
        env:
        - name: TLS_ENABLED
          value: "true"
        - name: API_KEY
          valueFrom:
            secretKeyRef:
              name: conductor-watch-secrets
              key: api-key
        - name: HMAC_SECRET
          valueFrom:
            secretKeyRef:
              name: conductor-watch-secrets
              key: hmac-secret
```

### Network Policy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: conductor-watch-netpol
  namespace: nephoran-system
spec:
  podSelector:
    matchLabels:
      app: conductor-watch
  policyTypes:
  - Ingress
  - Egress
  
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: nephoran-system
    ports:
    - protocol: TCP
      port: 8080
  
  egress:
  # Allow DNS
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
    ports:
    - protocol: UDP
      port: 53
  
  # Allow HTTPS to intent processor
  - to:
    - podSelector:
        matchLabels:
          app: intent-processor
    ports:
    - protocol: TCP
      port: 443
```

### RBAC Configuration

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: conductor-watch
  namespace: nephoran-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: conductor-watch
  namespace: nephoran-system
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "watch", "list"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get"]
  resourceNames: ["conductor-watch-secrets", "conductor-watch-certs"]
- apiGroups: ["nephoran.com"]
  resources: ["networkintents"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
- apiGroups: ["nephoran.com"]
  resources: ["networkintents/status"]
  verbs: ["get", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: conductor-watch
  namespace: nephoran-system
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: conductor-watch
subjects:
- kind: ServiceAccount
  name: conductor-watch
  namespace: nephoran-system
```

## Security Monitoring and Auditing

### Metrics Collection

The component exposes security-relevant metrics:

```go
type WatcherMetrics struct {
    FilesProcessed   uint64  // Total files processed
    FilesRejected    uint64  // Files rejected by validation
    HTTPSuccess      uint64  // Successful HTTP requests
    HTTPFailure      uint64  // Failed HTTP requests
    ValidationErrors uint64  // Schema validation failures
    SecurityErrors   uint64  // Security-related errors
}
```

### Audit Logging

Structured logging for security events:

```json
{
  "timestamp": "2025-08-18T10:30:45.123Z",
  "level": "info",
  "component": "conductor-watch",
  "event": "file_processed",
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "file": "intent-001.json",
  "file_size": 2048,
  "validation": "passed",
  "http_status": 200,
  "duration_ms": 125,
  "source_ip": "10.0.1.5",
  "result": "success"
}
```

### Prometheus Metrics

```yaml
# Exposed metrics
conductor_watch_files_processed_total
conductor_watch_files_rejected_total
conductor_watch_http_requests_total{status="success|failure"}
conductor_watch_validation_errors_total
conductor_watch_security_errors_total
conductor_watch_request_duration_seconds
conductor_watch_file_size_bytes
```

### Alert Rules

```yaml
groups:
- name: conductor-watch-security
  rules:
  - alert: HighValidationFailureRate
    expr: rate(conductor_watch_validation_errors_total[5m]) > 0.1
    for: 5m
    annotations:
      summary: "High validation failure rate"
      description: "Validation failures: {{ $value }} per second"
  
  - alert: SecurityErrorsDetected
    expr: increase(conductor_watch_security_errors_total[1h]) > 10
    for: 5m
    annotations:
      summary: "Security errors detected"
      description: "{{ $value }} security errors in the last hour"
  
  - alert: HTTPFailureRate
    expr: rate(conductor_watch_http_requests_total{status="failure"}[5m]) > 0.5
    for: 5m
    annotations:
      summary: "High HTTP failure rate"
      description: "HTTP failures: {{ $value }} per second"
```

## Security Best Practices

### Configuration Management

1. **Use Environment Variables**: Store sensitive values in environment variables
2. **Rotate Secrets Regularly**: Implement quarterly rotation for API keys and certificates
3. **Principle of Least Privilege**: Grant minimal required permissions
4. **Secure Defaults**: Ship with secure configuration defaults

### Operational Security

1. **Monitor Security Metrics**: Set up alerts for anomalous behavior
2. **Regular Updates**: Keep dependencies and base images updated
3. **Security Scanning**: Run vulnerability scans on containers
4. **Audit Logs**: Regularly review audit logs for suspicious activity

### Development Security

1. **Code Reviews**: All security-related changes require review
2. **Static Analysis**: Use golangci-lint with security linters
3. **Dependency Scanning**: Check for vulnerable dependencies
4. **Security Testing**: Include security test cases

## Security Checklist

### Pre-Deployment

- [ ] Generate and install TLS certificates
- [ ] Configure authentication (API key, Bearer token, or HMAC)
- [ ] Set appropriate resource limits
- [ ] Configure network policies
- [ ] Set up RBAC permissions
- [ ] Enable audit logging
- [ ] Review security contexts
- [ ] Test certificate validation
- [ ] Verify authentication works
- [ ] Test rate limiting
- [ ] Validate schema enforcement

### Post-Deployment

- [ ] Monitor TLS certificate expiry
- [ ] Review audit logs weekly
- [ ] Monitor resource usage
- [ ] Check for security updates monthly
- [ ] Rotate secrets quarterly
- [ ] Conduct security scanning
- [ ] Review and update network policies
- [ ] Performance security testing

### Incident Response

- [ ] Document security incidents
- [ ] Analyze root causes
- [ ] Update security controls
- [ ] Review and update runbooks
- [ ] Conduct post-mortem analysis

## Troubleshooting Guide

### Common Security Issues

#### TLS Certificate Errors

```bash
# Verify certificate chain
openssl verify -CAfile ca.crt client.crt

# Check certificate expiry
openssl x509 -in client.crt -noout -dates

# Test TLS connection
openssl s_client -connect intent-processor:443 \
  -cert client.crt -key client.key -CAfile ca.crt
```

#### Authentication Failures

```bash
# Test API key authentication
curl -H "X-API-Key: ${API_KEY}" https://conductor-watch/health

# Test Bearer token
curl -H "Authorization: Bearer ${TOKEN}" https://conductor-watch/health

# Verify HMAC signature
echo -n "${PAYLOAD}" | openssl dgst -sha256 -hmac "${HMAC_SECRET}"
```

#### Rate Limiting Issues

```bash
# Check rate limit headers in response
curl -I https://conductor-watch/api/intent | grep -E "X-RateLimit|Retry-After"

# Monitor rate limiting metrics
curl http://conductor-watch:9090/metrics | grep rate_limit
```

#### File Processing Issues

```bash
# Check file permissions
ls -la /handoff/

# Verify JSON schema
jq . intent.json > /dev/null 2>&1 && echo "Valid JSON"

# Test schema validation
ajv validate -s /etc/conductor-watch/schema/intent.schema.json -d intent.json
```

### Debug Logging

Enable debug logging for troubleshooting:

```yaml
logging:
  level: "debug"
  security: true  # Enable security-specific logging
```

## Compliance and Standards

### OWASP Top 10 Coverage

| Risk | Mitigation |
|------|------------|
| A01:2021 - Broken Access Control | RBAC, namespace isolation, mTLS |
| A02:2021 - Cryptographic Failures | TLS 1.2+, secure ciphers, certificate validation |
| A03:2021 - Injection | Schema validation, input sanitization |
| A04:2021 - Insecure Design | Defense in depth, rate limiting |
| A05:2021 - Security Misconfiguration | Secure defaults, hardened deployment |
| A06:2021 - Vulnerable Components | Regular updates, dependency scanning |
| A07:2021 - Authentication Failures | Strong authentication, rate limiting |
| A08:2021 - Data Integrity Failures | HMAC signing, TLS |
| A09:2021 - Security Logging Failures | Comprehensive audit logging |
| A10:2021 - SSRF | URL validation, network policies |

### CIS Kubernetes Benchmark

The deployment configuration follows CIS Kubernetes Benchmark recommendations:

- Non-root user execution
- Read-only root filesystem
- Capability dropping
- Resource limits
- Network policies
- RBAC configuration

## References

- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)
- [OWASP Top 10 2021](https://owasp.org/Top10/)
- [CIS Kubernetes Benchmark](https://www.cisecurity.org/benchmark/kubernetes)
- [Go Security Guidelines](https://golang.org/doc/security)
- [TLS Best Practices](https://github.com/ssllabs/research/wiki/SSL-and-TLS-Deployment-Best-Practices)

## Support and Contact

For security issues or questions:
- Security Team: security@nephoran.com
- Documentation: https://docs.nephoran.com/security
- Issue Tracker: https://github.com/nephoran/conductor-watch/security

---

*Last Updated: 2025-08-18*
*Version: 1.0.0*