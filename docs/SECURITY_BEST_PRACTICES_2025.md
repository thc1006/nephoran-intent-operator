# Security Best Practices 2025

## Executive Summary

This document outlines the security best practices for the Nephoran Intent Operator, aligned with 2025 industry standards and emerging threats. These practices ensure defense-in-depth, zero-trust architecture, and comprehensive supply chain security.

## Table of Contents

1. [Container Security](#container-security)
2. [Supply Chain Security](#supply-chain-security)
3. [Secret Management](#secret-management)
4. [Network Security](#network-security)
5. [Application Security](#application-security)
6. [CI/CD Security](#cicd-security)
7. [Monitoring and Incident Response](#monitoring-and-incident-response)
8. [Compliance and Governance](#compliance-and-governance)

## Container Security

### 1. Base Image Hardening

```dockerfile
# ✅ GOOD: Use distroless or minimal base images
FROM gcr.io/distroless/static:nonroot

# ❌ BAD: Using full OS images
FROM ubuntu:latest
```

**Best Practices:**
- Use distroless images for production
- Pin specific image versions (never use `latest`)
- Scan base images before use
- Update base images monthly

### 2. Non-Root User

```dockerfile
# Create non-root user with specific UID/GID
RUN addgroup -g 65532 -S nonroot && \
    adduser -u 65532 -S nonroot -G nonroot -s /sbin/nologin

USER 65532:65532
```

### 3. Security Hardening

```dockerfile
# Remove SUID/SGID bits
RUN find / -xdev -type f -perm +6000 -delete 2>/dev/null || true

# Set read-only root filesystem
# Run with: --read-only --tmpfs /tmp

# Drop all capabilities
# Run with: --cap-drop=ALL

# Enable security options
# Run with: --security-opt=no-new-privileges:true
```

### 4. Health Checks

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/healthcheck", "--secure"]
```

### 5. Multi-Stage Builds

```dockerfile
# Build stage with all tools
FROM golang:1.24-alpine AS builder
# ... build process ...

# Minimal runtime stage
FROM gcr.io/distroless/static:nonroot
COPY --from=builder --chown=65532:65532 /app/binary /binary
```

## Supply Chain Security

### 1. Dependency Management

```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "daily"
    security-updates-only: true
    open-pull-requests-limit: 10
```

### 2. SBOM Generation

```bash
# Generate SBOM for every release
syft . -o spdx-json=sbom.spdx.json
syft . -o cyclonedx-json=sbom.cyclonedx.json

# Attest SBOM to container image
cosign attest --predicate sbom.spdx.json --type spdxjson IMAGE
```

### 3. Image Signing

```bash
# Sign images with keyless signing
cosign sign --yes IMAGE

# Verify signatures
cosign verify \
  --certificate-identity-regexp ".*" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  IMAGE
```

### 4. Provenance

```yaml
# Generate SLSA provenance
- name: Generate provenance
  uses: slsa-framework/slsa-github-generator@v2.0.0
  with:
    subjects: |
      sha256:hash1 binary1
      sha256:hash2 binary2
```

## Secret Management

### 1. Never Commit Secrets

```bash
# Use pre-commit hooks
cat > .pre-commit-config.yaml << EOF
repos:
  - repo: https://github.com/trufflesecurity/trufflehog
    rev: v3.73.0
    hooks:
      - id: trufflehog
        args: ['--fail', '--no-update']
EOF
```

### 2. Use Secret Managers

```go
// Use external secret managers
import "github.com/aws/aws-sdk-go/service/secretsmanager"

func getSecret(name string) (string, error) {
    svc := secretsmanager.New(session.New())
    input := &secretsmanager.GetSecretValueInput{
        SecretId: aws.String(name),
    }
    result, err := svc.GetSecretValue(input)
    if err != nil {
        return "", err
    }
    return *result.SecretString, nil
}
```

### 3. Kubernetes Secrets

```yaml
# Use sealed-secrets or external-secrets
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: app-secrets
spec:
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: app-secrets
  data:
    - secretKey: api-key
      remoteRef:
        key: secret/data/api
        property: key
```

### 4. Environment Variables

```go
// Sanitize environment variables
func init() {
    // Clear sensitive env vars after reading
    apiKey := os.Getenv("API_KEY")
    os.Unsetenv("API_KEY")
    // Use the apiKey...
}
```

## Network Security

### 1. Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all-ingress
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  ingress: []  # Deny all ingress by default
```

### 2. Service Mesh

```yaml
# Istio mTLS configuration
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
spec:
  mtls:
    mode: STRICT
```

### 3. TLS Configuration

```go
// Enforce TLS 1.3 minimum
tlsConfig := &tls.Config{
    MinVersion:               tls.VersionTLS13,
    CurvePreferences:        []tls.CurveID{tls.X25519, tls.CurveP256},
    PreferServerCipherSuites: true,
    CipherSuites: []uint16{
        tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
    },
}
```

## Application Security

### 1. Input Validation

```go
// Validate all inputs
func validateInput(input string) error {
    if len(input) > 1024 {
        return errors.New("input too long")
    }
    if !regexp.MustCompile(`^[a-zA-Z0-9-_]+$`).MatchString(input) {
        return errors.New("invalid characters in input")
    }
    return nil
}
```

### 2. SQL Injection Prevention

```go
// Use parameterized queries
stmt, err := db.Prepare("SELECT * FROM users WHERE id = ?")
rows, err := stmt.Query(userID)
```

### 3. CSRF Protection

```go
// Implement CSRF tokens
func generateCSRFToken() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}
```

### 4. Rate Limiting

```go
// Implement rate limiting
limiter := rate.NewLimiter(rate.Every(time.Second), 10)

func handler(w http.ResponseWriter, r *http.Request) {
    if !limiter.Allow() {
        http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
        return
    }
    // Handle request
}
```

## CI/CD Security

### 1. GitHub Actions Security

```yaml
# Use specific action versions
- uses: actions/checkout@b4ffde65f46336ab88eb53be0d2a3b5a3e8e3e93 # v4.1.1

# Minimize permissions
permissions:
  contents: read
  id-token: write  # For OIDC only

# Use OIDC for authentication
- uses: aws-actions/configure-aws-credentials@v4
  with:
    role-to-assume: arn:aws:iam::123456789012:role/GitHubActions
    aws-region: us-east-1
```

### 2. Security Scanning Pipeline

```yaml
jobs:
  security:
    steps:
      # Secret scanning
      - name: TruffleHog
        run: |
          trufflehog filesystem . --fail --no-update

      # SAST
      - name: Semgrep
        run: |
          semgrep scan --config=auto --sarif -o semgrep.sarif .

      # Dependency scanning
      - name: Nancy
        run: |
          go list -json -deps ./... | nancy sleuth

      # Container scanning
      - name: Trivy
        run: |
          trivy image --severity HIGH,CRITICAL --fail-on critical IMAGE
```

### 3. Build Attestations

```yaml
- name: Generate attestations
  run: |
    # Generate SBOM attestation
    cosign attest --predicate sbom.json --type spdxjson IMAGE
    
    # Generate vulnerability attestation
    cosign attest --predicate vuln-scan.json --type vuln IMAGE
    
    # Generate SLSA provenance
    cosign attest --predicate provenance.json --type slsaprovenance IMAGE
```

## Monitoring and Incident Response

### 1. Security Logging

```go
// Structured security logging
logger.Info("security_event",
    zap.String("event_type", "authentication"),
    zap.String("user_id", userID),
    zap.String("ip_address", clientIP),
    zap.Bool("success", success),
    zap.Time("timestamp", time.Now()),
)
```

### 2. Alerting Rules

```yaml
# Prometheus alerting rules
groups:
  - name: security
    rules:
      - alert: SuspiciousActivity
        expr: rate(failed_login_attempts[5m]) > 10
        for: 2m
        annotations:
          summary: "High rate of failed login attempts"
```

### 3. Incident Response

```markdown
## Incident Response Checklist

1. **Detect & Alert**
   - [ ] Validate the alert
   - [ ] Assess severity
   - [ ] Notify incident team

2. **Contain**
   - [ ] Isolate affected systems
   - [ ] Preserve evidence
   - [ ] Stop the spread

3. **Eradicate**
   - [ ] Remove threat
   - [ ] Patch vulnerabilities
   - [ ] Update security controls

4. **Recover**
   - [ ] Restore services
   - [ ] Monitor for recurrence
   - [ ] Validate security posture

5. **Lessons Learned**
   - [ ] Document timeline
   - [ ] Identify improvements
   - [ ] Update playbooks
```

## Compliance and Governance

### 1. Security Policies

```yaml
# OPA policy for container security
package kubernetes.admission

deny[msg] {
  input.request.kind.kind == "Pod"
  not input.request.object.spec.securityContext.runAsNonRoot
  msg := "Pods must run as non-root user"
}

deny[msg] {
  input.request.kind.kind == "Pod"
  input.request.object.spec.containers[_].securityContext.privileged
  msg := "Privileged containers are not allowed"
}
```

### 2. Compliance Scanning

```bash
# CIS Kubernetes Benchmark
kube-bench run --targets master,node,etcd,policies

# Docker CIS Benchmark
docker run --rm --net host --pid host --cap-add audit_control \
  -v /var/lib:/var/lib -v /var/run/docker.sock:/var/run/docker.sock \
  -v /etc:/etc --label docker_bench_security \
  docker/docker-bench-security
```

### 3. Audit Logging

```yaml
# Kubernetes audit policy
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
  - level: RequestResponse
    omitStages:
      - RequestReceived
    resources:
      - group: ""
        resources: ["secrets", "configmaps"]
    namespaces: ["kube-system", "default"]
```

## Security Checklist

### Daily Tasks
- [ ] Review security alerts
- [ ] Check vulnerability scan results
- [ ] Monitor for new CVEs
- [ ] Review access logs

### Weekly Tasks
- [ ] Update dependencies
- [ ] Review security patches
- [ ] Audit user access
- [ ] Test backup recovery

### Monthly Tasks
- [ ] Security training
- [ ] Penetration testing
- [ ] Compliance audit
- [ ] Update security documentation

### Quarterly Tasks
- [ ] Security architecture review
- [ ] Threat modeling update
- [ ] Disaster recovery drill
- [ ] Third-party security assessment

## Tools and Resources

### Security Tools
- **Scanning**: Trivy, Grype, Snyk
- **SAST**: Semgrep, GoSec, CodeQL
- **Secrets**: TruffleHog, Gitleaks
- **Runtime**: Falco, Tracee
- **Policy**: OPA, Kyverno

### References
- [OWASP Top 10 2025](https://owasp.org/www-project-top-ten/)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [SLSA Framework](https://slsa.dev/)
- [Cloud Native Security Whitepaper](https://www.cncf.io/reports/cloud-native-security-whitepaper/)

## Conclusion

Security is not a one-time effort but a continuous process. These best practices should be regularly reviewed and updated to address emerging threats and new technologies. Remember: **Security is everyone's responsibility**.

---

**Document Version:** 1.0
**Last Updated:** January 29, 2025
**Next Review:** February 28, 2025
**Owner:** Security Team
**Classification:** Public