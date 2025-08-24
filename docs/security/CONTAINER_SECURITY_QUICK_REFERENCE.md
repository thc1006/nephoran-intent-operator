# 🔒 Container Security Quick Reference Guide

## Security Commands Cheat Sheet

### Container Security Validation
```powershell
# Full security audit
.\scripts\security\validate-container-security.ps1 -DetailedReport

# CI/CD validation (exit codes for automation)
.\scripts\security\validate-container-security.ps1 -CiMode

# Export results for reporting
.\scripts\security\validate-container-security.ps1 -ExportPath results.json
```

### Image Scanning & Signing
```bash
# Vulnerability scanning
trivy image nephoran/llm-processor:latest --severity HIGH,CRITICAL

# Generate SBOM
syft nephoran/llm-processor:latest -o spdx-json=sbom.json

# Sign images
cosign sign --yes nephoran/llm-processor:latest

# Verify signatures
cosign verify nephoran/llm-processor:latest
```

### Build Security-Hardened Images
```bash
# Production build
docker build --build-arg SERVICE=llm-processor -t nephoran/llm-processor:latest .

# Multi-arch build
docker buildx build --platform linux/amd64,linux/arm64 \
  --build-arg SERVICE=llm-processor -t nephoran/llm-processor:latest .

# Security scan after build
trivy image nephoran/llm-processor:latest
```

## Security Configuration Templates

### Docker Compose Security Block
```yaml
services:
  service-name:
    # Enhanced security configuration
    security_opt:
      - no-new-privileges:true
      - seccomp:unconfined
      - apparmor:docker-default
    read_only: true
    user: "65532:65532"
    cap_drop: [ALL]
    cap_add: []
    privileged: false
    pid: ""
    userns_mode: "host"
    
    # Resource limits
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M
    
    # Health checks
    healthcheck:
      test: ["CMD", "/service", "--health-check", "--secure"]
      interval: 30s
      timeout: 5s
      retries: 3
    
    # Secure tmpfs
    tmpfs:
      - /tmp:noexec,nosuid,size=100m
```

### Kubernetes Security Context
```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      # Pod-level security
      securityContext:
        runAsNonRoot: true
        runAsUser: 65532
        runAsGroup: 65532
        fsGroup: 65532
        seccompProfile:
          type: RuntimeDefault
      
      containers:
      - name: container-name
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop: [ALL]
          runAsNonRoot: true
          runAsUser: 65532
          runAsGroup: 65532
        
        resources:
          limits:
            cpu: 2000m
            memory: 2Gi
            ephemeral-storage: 4Gi
          requests:
            cpu: 100m
            memory: 128Mi
```

### Dockerfile Security Template
```dockerfile
# Use specific, security-hardened base image
FROM golang:1.24.8-alpine AS builder

# Install security updates
RUN apk update && apk upgrade --no-cache && \
    apk add --no-cache git=~2.45 ca-certificates=~20241010

# Create non-root user
RUN addgroup -g 65532 -S nonroot && \
    adduser -u 65532 -S nonroot -G nonroot

# Set working directory and copy source
WORKDIR /build
COPY --chown=nonroot:nonroot . .

# Build as non-root
USER nonroot
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o service ./cmd/service

# Runtime stage with distroless
FROM gcr.io/distroless/static:nonroot-amd64

# Copy binary with restricted permissions
COPY --from=builder --chmod=555 /build/service /service

# Security labels
LABEL security.hardened="true" \
      security.nonroot="true" \
      security.readonly="true"

# Non-root user
USER 65532:65532

# Health check
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD ["/service", "--health-check", "--secure"]

ENTRYPOINT ["/service"]
```

## Security Validation Checklist

### ✅ Container Runtime Security
- [ ] Non-root user execution (UID 65532)
- [ ] Read-only root filesystem enabled
- [ ] ALL capabilities dropped
- [ ] No privilege escalation allowed
- [ ] Seccomp profile configured
- [ ] Resource limits defined
- [ ] Health checks implemented

### ✅ Image Security
- [ ] Latest security-patched base images
- [ ] No hardcoded secrets
- [ ] Minimal image size (distroless preferred)
- [ ] Version-pinned dependencies
- [ ] Multi-stage builds
- [ ] Security labels present
- [ ] SBOM generated

### ✅ Network Security
- [ ] Custom networks for isolation
- [ ] Network policies defined
- [ ] No host network mode
- [ ] Minimal exposed ports
- [ ] TLS encryption enabled
- [ ] Service mesh integration

### ✅ Supply Chain Security
- [ ] Image signatures verified
- [ ] SLSA provenance attestation
- [ ] Vulnerability scanning automated
- [ ] Dependency validation
- [ ] Build reproducibility
- [ ] Audit trail maintained

## Security Thresholds

| Metric | Target | Critical |
|--------|--------|----------|
| Security Score | ≥80% | <60% |
| Critical CVEs | 0 | >0 |
| High CVEs | <5 | >10 |
| Image Size | <100MB | >500MB |
| Build Time | <10min | >30min |
| Scan Time | <5min | >15min |

## Troubleshooting Common Issues

### Permission Denied Errors
```bash
# Check if running as non-root
docker exec container-name id

# Verify file permissions
docker exec container-name ls -la /

# Fix tmpfs permissions
tmpfs:
  - /tmp:noexec,nosuid,uid=65532,gid=65532,size=100m
```

### Image Pull/Sign Failures
```bash
# Login to registry
docker login ghcr.io

# Verify Cosign configuration
cosign version
export COSIGN_EXPERIMENTAL=1

# Check image signature
cosign verify --certificate-identity-regexp=".*" --certificate-oidc-issuer-regexp=".*" image:tag
```

### Security Scan Issues
```bash
# Clear Trivy cache
trivy clean --scan-cache

# Update vulnerability database
trivy image --download-db-only

# Scan with debug output
trivy image --debug image:tag
```

### Kubernetes Security Context Problems
```bash
# Check PSP/PSS violations
kubectl describe pod pod-name

# Verify security context
kubectl get pod pod-name -o jsonpath='{.spec.securityContext}'

# Check admission controller logs
kubectl logs -n kube-system deployment/admission-controller
```

## Emergency Security Procedures

### 1. Critical Vulnerability Response
1. **Immediate Action**: Stop affected containers
2. **Assessment**: Scan all images for vulnerability
3. **Remediation**: Update base images and rebuild
4. **Validation**: Re-scan and verify fixes
5. **Deployment**: Roll out updates with zero downtime

### 2. Compromised Container Detection
1. **Isolation**: Network isolate affected pods
2. **Investigation**: Collect logs and forensics
3. **Analysis**: Determine attack vector
4. **Recovery**: Rebuild from clean images
5. **Hardening**: Apply additional security controls

### 3. Security Policy Violation
1. **Alert**: Immediate notification to security team
2. **Block**: Prevent deployment of non-compliant images
3. **Analysis**: Review policy compliance
4. **Correction**: Fix configuration issues
5. **Monitoring**: Enhanced security monitoring

## Security Contacts & Resources

- **Security Team**: security@nephoran.com
- **Incident Response**: incidents@nephoran.com  
- **Security Documentation**: `/docs/security/`
- **Emergency Runbook**: `/docs/runbooks/security-incident-response.md`

## Quick Links

- [Security Hardening Report](../../CONTAINER-SECURITY-HARDENING-REPORT.md)
- [Security Validation Script](../../scripts/security/validate-container-security.ps1)
- [CI/CD Security Pipeline](../../.github/workflows/container-security-hardening.yml)
- [Security Configurations](../../deployments/security/)

---
**Last Updated**: August 24, 2025  
**Version**: 2.0.0  
**Maintained by**: Nephoran Security Team