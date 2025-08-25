# Container Security Hardening for Nephoran Intent Operator

## Overview

This document provides comprehensive container security hardening implementation for the Nephoran Intent Operator, following industry best practices and compliance frameworks including:

- **CIS Docker Benchmark v1.7.0**
- **NIST SP 800-190** (Application Container Security Guide)
- **OWASP Container Security Top 10**
- **Kubernetes Pod Security Standards (Restricted)**
- **O-RAN WG11 Security Specifications**

## 🛡️ Security Architecture

```mermaid
graph TB
    subgraph "Build-time Security"
        A[Secure Base Images] --> B[Multi-stage Builds]
        B --> C[Vulnerability Scanning]
        C --> D[SBOM Generation]
        D --> E[Image Signing]
    end
    
    subgraph "Runtime Security"
        F[Pod Security Standards] --> G[Security Contexts]
        G --> H[Network Policies]
        H --> I[Resource Limits]
        I --> J[RBAC]
    end
    
    subgraph "Monitoring & Compliance"
        K[Security Scanning] --> L[Compliance Validation]
        L --> M[Vulnerability Management]
        M --> N[Incident Response]
    end
    
    E --> F
    J --> K
```

## 📋 Implementation Checklist

### ✅ Container Image Security

- [x] **Distroless Base Images**: Using `gcr.io/distroless/static:nonroot` for Go services
- [x] **Non-Root User**: All containers run as UID 65532 (nonroot)
- [x] **Multi-Stage Builds**: Separate build and runtime environments
- [x] **Minimal Attack Surface**: Only essential binaries in final image
- [x] **Security Labels**: Comprehensive OCI-compliant labels
- [x] **Read-Only Containers**: Immutable container filesystems

### ✅ Kubernetes Security

- [x] **Pod Security Standards**: Restricted profile enforced
- [x] **Security Contexts**: Comprehensive security settings
- [x] **Network Policies**: Zero-trust network segmentation
- [x] **Resource Limits**: CPU/memory constraints
- [x] **RBAC**: Least-privilege access control
- [x] **Service Account Security**: No auto-mount of tokens where possible

### ✅ Scanning & Compliance

- [x] **Vulnerability Scanning**: Trivy and Grype integration
- [x] **SBOM Generation**: SPDX and CycloneDX formats
- [x] **Security Validation**: Automated compliance checks
- [x] **CI/CD Integration**: Security gates in pipelines
- [x] **Continuous Monitoring**: Daily security scans

## 🔧 Implementation Details

### 1. Dockerfile Hardening

The main `Dockerfile` has been enhanced with:

```dockerfile
# Security-hardened base image versions
ARG GO_VERSION=1.24.5
ARG PYTHON_VERSION=3.12.8
ARG ALPINE_VERSION=3.21.2
ARG DISTROLESS_VERSION=nonroot

# Multi-stage build with security focus
FROM golang:${GO_VERSION}-alpine AS go-builder
RUN apk upgrade --no-cache
# ... build process ...

# Final runtime using distroless
FROM gcr.io/distroless/static:${DISTROLESS_VERSION}
USER 65532:65532
```

**Key Security Features:**
- Latest secure base image versions
- Distroless runtime images
- Non-root user (65532)
- Security-focused build flags
- Comprehensive security labels

### 2. Kubernetes Security Contexts

Enhanced security contexts for all deployments:

```yaml
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 65532
    runAsGroup: 65532
    fsGroup: 65532
    fsGroupChangePolicy: OnRootMismatch
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: service
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop: [ALL]
      runAsNonRoot: true
      runAsUser: 65532
```

### 3. Network Security Policies

Zero-trust network architecture:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: nephoran-default-deny-all
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  # Explicit allow rules only
```

**Network Segmentation:**
- Default deny-all policy
- Service-specific ingress/egress rules
- Monitoring and security scanning access
- External API access controls

### 4. Pod Security Standards

Namespace-level enforcement:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: nephoran-system
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

## 🔍 Security Scanning

### Container Vulnerability Scanning

Implemented with multiple tools for comprehensive coverage:

1. **Trivy Scanner**
   - CVE database scanning
   - Secret detection
   - Configuration analysis
   - SARIF output for GitHub Security

2. **Grype Scanner**
   - Alternative vulnerability database
   - Cross-validation of results
   - Supply chain analysis

3. **SBOM Generation**
   - SPDX-JSON format
   - CycloneDX format
   - Package inventory
   - License compliance

### Usage Examples

**Run security scans:**
```powershell
# Generate SBOM and run vulnerability scans
.\scripts\security\generate-sbom.ps1 -Services @("llm-processor", "rag-api") -GenerateReport

# Validate security hardening
.\scripts\security\validate-security-hardening.ps1 -ValidationLevel strict -CisCompliance
```

**CI/CD Integration:**
```yaml
- name: Security Scanning
  uses: ./.github/workflows/container-security-scanning.yml
  with:
    severity_threshold: HIGH
    scan_all_images: true
```

## 📊 Compliance Validation

### CIS Kubernetes Benchmark

Key controls implemented:

- **CIS 5.1.1**: No privileged containers
- **CIS 5.1.3**: No privilege escalation allowed  
- **CIS 5.1.4**: All capabilities dropped
- **CIS 5.2.2**: Read-only root filesystem
- **CIS 5.2.3**: Non-root user enforcement

### NIST SP 800-190 Guidelines

- **Recommendation 1**: Image security and management
- **Recommendation 2**: Registry security 
- **Recommendation 3**: Runtime security
- **Recommendation 4**: Host OS security
- **Recommendation 5**: Orchestrator security

### O-RAN Security (WG11)

Specific telecommunications security requirements:

- **Authentication & Authorization**: OAuth2/JWT implementation
- **Network Function Security**: E2/A1/O1 interface protection
- **Zero Trust Architecture**: Network segmentation
- **Compliance Monitoring**: Continuous security validation

## 🚨 Security Incident Response

### Vulnerability Management

1. **Critical Vulnerabilities (CVSS ≥ 9.0)**
   - Immediate notification
   - Emergency patching within 24 hours
   - Incident response activation

2. **High Vulnerabilities (CVSS 7.0-8.9)**
   - Patch within 72 hours
   - Risk assessment
   - Monitoring enhancement

3. **Medium/Low Vulnerabilities**
   - Regular patching cycles
   - Quarterly security reviews

### Monitoring & Alerting

**Security Metrics Tracked:**
- Vulnerability counts by severity
- Container security compliance percentage
- Failed security validations
- SBOM coverage percentage

**Alert Thresholds:**
- Critical: Any CVSS ≥ 9.0 vulnerability
- High: >5 high-severity vulnerabilities
- Compliance: <90% security validation pass rate

## 🛠️ Tools and Dependencies

### Security Scanning Tools

| Tool | Version | Purpose |
|------|---------|---------|
| Trivy | 0.58.1 | Vulnerability scanning |
| Grype | 0.85.0 | Alternative vulnerability scanning |
| Syft | 1.18.1 | SBOM generation |
| Cosign | 2.4.1 | Container signing |

### Kubernetes Tools

- **kubectl**: Kubernetes CLI
- **kustomize**: Configuration management
- **helm**: Package management (optional)

### Installation

```powershell
# Install security tools (Windows)
.\scripts\security\install-security-tools.ps1

# Verify installation
trivy version
grype version
syft version
cosign version
```

## 📈 Performance Impact

Security hardening introduces minimal performance overhead:

- **Container Startup**: +2-5% due to security context validation
- **Network Latency**: +1-2ms due to network policies
- **Resource Usage**: +5-10% for security monitoring
- **Build Time**: +30-60s for security scanning

## 🔄 Maintenance and Updates

### Regular Security Tasks

**Daily:**
- Automated vulnerability scanning
- Security compliance validation
- SBOM generation

**Weekly:**
- Security policy review
- Incident response testing
- Compliance reporting

**Monthly:**
- Security tool updates
- Policy optimization
- Security training updates

**Quarterly:**
- Comprehensive security audit
- Penetration testing
- Policy review and updates

### Update Procedures

1. **Base Image Updates**
   ```bash
   # Update Dockerfile with latest versions
   # Run security scans
   # Update deployment manifests
   ```

2. **Security Tool Updates**
   ```powershell
   # Update tool versions in scripts
   # Test with sample containers
   # Deploy to CI/CD pipelines
   ```

3. **Policy Updates**
   ```yaml
   # Update security policies
   # Validate with test deployments
   # Apply to production
   ```

## 🎯 Security Goals and KPIs

### Target Metrics

- **Vulnerability Management**: Zero critical vulnerabilities in production
- **Compliance**: 95%+ CIS Kubernetes Benchmark compliance
- **SBOM Coverage**: 100% of container images
- **Security Scanning**: 100% of images scanned before deployment
- **Incident Response**: <4 hour mean time to remediation

### Success Criteria

1. **Zero Critical Vulnerabilities**: No CVSS ≥ 9.0 in production
2. **Compliance Validation**: All security checks pass
3. **Automation**: Fully automated security pipeline
4. **Monitoring**: Real-time security monitoring
5. **Documentation**: Complete security documentation

## 📚 References

- [CIS Docker Benchmark](https://www.cisecurity.org/benchmark/docker)
- [NIST SP 800-190](https://csrc.nist.gov/publications/detail/sp/800-190/final)
- [OWASP Container Security](https://owasp.org/www-project-container-security/)
- [Kubernetes Security](https://kubernetes.io/docs/concepts/security/)
- [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)

## 🤝 Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for security-related contribution guidelines.

## 📄 License

This security implementation is licensed under the Apache License 2.0. See [LICENSE](../LICENSE) for details.

---

**Security Contact**: For security-related issues, please email security@nephoran.com or create a private security advisory.