# Docker Security Guide - Nephoran Intent Operator

**Last Updated:** July 29, 2025  
**Security Level:** Production Grade  
**Compliance:** NIST, CIS Docker Benchmark

## Overview

The Nephoran Intent Operator implements comprehensive Docker security best practices with multi-stage builds, distroless runtime images, and security scanning integration. This document covers all security implementations and procedures.

## Security Architecture

### Multi-Stage Build Security

#### Standard Build Pattern
```dockerfile
# Build stage - Full development environment
FROM golang:1.24-alpine AS builder
WORKDIR /app

# Security: Copy only required files
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -trimpath -a -installsuffix cgo \
    -o service ./cmd/service

# Runtime stage - Distroless for security
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /app/service .
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/service"]
```

### Security Features Implementation

#### 1. Distroless Runtime Images
- **Base Image**: `gcr.io/distroless/static:nonroot`
- **Benefits**: Minimal attack surface, no shell, no package manager
- **Size**: ~2MB vs ~100MB+ for full Linux distributions
- **Vulnerabilities**: Significantly reduced attack vectors

#### 2. Non-Root User Execution
```dockerfile
# All containers run as non-root
USER nonroot:nonroot

# Numeric UID for maximum compatibility
USER 65532:65532
```

#### 3. Binary Optimization and Security
```bash
# Secure build flags
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE}" \
    -trimpath -a -installsuffix cgo \
    -o service ./cmd/service
```

**Security Build Flags:**
- `-w`: Remove DWARF debugging information
- `-s`: Remove symbol table and debug information
- `-trimpath`: Remove file system paths from executable
- `CGO_ENABLED=0`: Disable CGO for static linking

## Container Security Scanning

### Automated Security Validation

#### 1. Build-Time Scanning
```bash
# Integrated into docker-build target
make docker-build                 # Includes security scanning
make validate-images              # Post-build security validation
```

#### 2. Vulnerability Scanning
```bash
# Comprehensive vulnerability scanning
./scripts/security/vulnerability-scanner.sh

# Container-specific security
./scripts/security/security-config-validator.sh
```

#### 3. Security Audit
```bash
# Complete security audit including containers
./scripts/security/execute-security-audit.sh
```

### Security Scanning Tools Integration

#### Implemented Scanners
1. **govulncheck**: Go vulnerability database scanning
2. **Docker Scout**: Container vulnerability scanning
3. **Trivy**: Comprehensive security scanner
4. **Custom Scripts**: Configuration and runtime security

#### Scanning Configuration
```yaml
# Security scanning in CI/CD
security_scan:
  - name: "Go Vulnerability Check"
    command: "govulncheck ./..."
  - name: "Container Scanning"
    command: "docker scout cves --format json"
  - name: "Configuration Security"
    command: "./scripts/security/security-config-validator.sh"
```

## Security Hardening Measures

### 1. Container Runtime Security

#### Resource Limits
```yaml
# Kubernetes resource limits for security
resources:
  limits:
    cpu: "1000m"
    memory: "2Gi"
  requests:
    cpu: "500m"
    memory: "1Gi"
```

#### Security Context
```yaml
# Pod security context
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  runAsGroup: 65532
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
```

### 2. Network Security

#### Network Policies
```yaml
# Restrict network traffic
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: nephoran-network-policy
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/part-of: nephoran-intent-operator
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: nephoran-system
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 443  # HTTPS only
```

#### TLS Configuration
- **Internal TLS**: Service-to-service encryption
- **External TLS**: HTTPS endpoints with valid certificates
- **Certificate Management**: Automated rotation with cert-manager

### 3. Secret Management

#### Secret Handling
```yaml
# Secure secret mounting
env:
- name: OPENAI_API_KEY
  valueFrom:
    secretKeyRef:
      name: openai-secret
      key: api-key
```

#### Secret Security
- **Encryption at Rest**: Kubernetes secret encryption
- **Access Control**: RBAC-based secret access
- **Rotation**: Automated secret rotation
- **Audit**: Secret access logging

## Image Security Best Practices

### 1. Base Image Security

#### Distroless Advantages
- **No Shell**: Cannot execute shell commands
- **No Package Manager**: Cannot install additional packages
- **Minimal Dependencies**: Reduced attack surface
- **Static Binaries**: No dynamic library vulnerabilities

#### Image Layers
```dockerfile
# Optimized layer structure
FROM golang:1.24-alpine AS builder
# ... build steps

# Final minimal layer
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /app/service .
```

### 2. Image Scanning and Validation

#### Automated Scanning
```bash
# Image security validation
make validate-images

# Manual scanning
docker scout cves $(REGISTRY)/llm-processor:$(VERSION)
docker scout recommendations $(REGISTRY)/llm-processor:$(VERSION)
```

#### Security Benchmarks
- **CIS Docker Benchmark**: Compliance validation
- **NIST Guidelines**: Security framework adherence
- **Container Security**: Industry best practices

### 3. Image Signing and Verification

#### Image Integrity
```bash
# Image signing (cosign integration planned)
cosign sign $(REGISTRY)/llm-processor:$(VERSION)

# Verification
cosign verify $(REGISTRY)/llm-processor:$(VERSION)
```

## Production Deployment Security

### 1. Runtime Security

#### Security Policies
```yaml
# Pod Security Standards
apiVersion: v1
kind: Namespace
metadata:
  name: nephoran-system
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

#### Service Mesh Security
```yaml
# Istio security configuration
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: nephoran-system
spec:
  mtls:
    mode: STRICT
```

### 2. Monitoring and Alerting

#### Security Monitoring
```yaml
# Security monitoring alerts
groups:
- name: container-security
  rules:
  - alert: ContainerHighCPU
    expr: container_cpu_usage_seconds_total > 0.8
  - alert: ContainerPrivilegeEscalation
    expr: kube_pod_container_security_context_privileged == 1
  - alert: ContainerRootUser
    expr: kube_pod_container_security_context_run_as_non_root == 0
```

#### Audit Logging
- **Container Events**: All container lifecycle events
- **Security Events**: Privilege escalation attempts
- **Network Events**: Unauthorized network connections
- **File System Events**: Read-only filesystem violations

### 3. Incident Response

#### Security Incident Procedures
1. **Detection**: Automated security monitoring
2. **Isolation**: Container/pod isolation procedures
3. **Investigation**: Security event analysis
4. **Recovery**: Clean container deployment
5. **Reporting**: Security incident documentation

## Security Compliance

### 1. Compliance Frameworks

#### NIST Cybersecurity Framework
- **Identify**: Asset and vulnerability identification
- **Protect**: Access control and data security
- **Detect**: Security monitoring and detection
- **Respond**: Incident response procedures
- **Recover**: Recovery and restoration procedures

#### CIS Docker Benchmark
- **Host Configuration**: Secure Docker daemon configuration
- **Docker Daemon**: Secure daemon startup options
- **Container Images**: Secure image creation and management
- **Container Runtime**: Secure container deployment

### 2. Compliance Validation

#### Automated Compliance Checks
```bash
# Run compliance validation
./scripts/security/security-config-validator.sh

# CIS benchmark validation
./scripts/cis-docker-benchmark.sh

# NIST framework assessment
./scripts/nist-compliance-check.sh
```

## Security Testing

### 1. Penetration Testing

#### Automated Penetration Testing
```bash
# Security penetration testing
./scripts/security/security-penetration-test.sh

# Container escape testing
./scripts/container-escape-test.sh

# Network security testing
./scripts/network-security-test.sh
```

### 2. Vulnerability Assessment

#### Regular Security Assessments
```bash
# Weekly vulnerability scans
./scripts/security/vulnerability-scanner.sh

# Dependency vulnerability check
make security-scan

# Container vulnerability assessment
docker scout cves --format json
```

## Troubleshooting Security Issues

### 1. Common Security Problems

#### Permission Denied Errors
```bash
# Symptoms: Container fails to start with permission errors
# Solution: Verify non-root user configuration
kubectl describe pod <pod-name>
kubectl logs <pod-name>

# Check security context
kubectl get pod <pod-name> -o yaml | grep -A 10 securityContext
```

#### Security Context Violations
```bash
# Symptoms: Pod Security Standards violations
# Solution: Update security context
kubectl describe pod <pod-name> | grep -A 5 "Warning"

# Validate security policies
kubectl get psp,netpol -A
```

### 2. Security Validation Failures

#### Container Security Scan Failures
```bash
# Symptoms: Security scans report vulnerabilities
# Solution: Update base images and dependencies
make update-deps
make docker-build
make validate-images
```

#### Certificate Issues
```bash
# Symptoms: TLS/certificate errors
# Solution: Verify certificate configuration
kubectl get certificates -A
kubectl describe certificate <cert-name>
```

## Best Practices Summary

### 1. Development Security
- Use distroless base images
- Implement multi-stage builds
- Run as non-root user
- Enable read-only root filesystem
- Drop all Linux capabilities

### 2. Runtime Security
- Implement network policies
- Use Pod Security Standards
- Enable audit logging
- Monitor security events
- Regular vulnerability scanning

### 3. Operational Security
- Automate security scanning
- Regular security assessments
- Incident response procedures
- Security compliance validation
- Continuous security monitoring

## Security Checklist

### Pre-Deployment Security Validation
- [ ] Base images updated to latest versions
- [ ] Security scanning completed without critical issues
- [ ] Non-root user configured
- [ ] Resource limits defined
- [ ] Network policies implemented
- [ ] Secret management configured
- [ ] TLS encryption enabled
- [ ] Audit logging configured

### Runtime Security Monitoring
- [ ] Security alerts configured
- [ ] Vulnerability scanning scheduled
- [ ] Compliance validation automated
- [ ] Incident response procedures documented
- [ ] Security training completed

## Conclusion

The Nephoran Intent Operator implements comprehensive Docker security measures following industry best practices and compliance frameworks. Regular security assessments and continuous monitoring ensure production-grade security for cloud-native deployments.

For security questions or incidents:
1. Review security scan results: `make security-scan`
2. Check compliance status: `./scripts/security/security-config-validator.sh`
3. Run security audit: `./scripts/security/execute-security-audit.sh`
4. Consult security procedures in incident response documentation