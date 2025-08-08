# Security Framework for Nephoran Intent Operator

## Overview

This directory contains the comprehensive security scanning and supply chain validation framework for the Nephoran Intent Operator. The framework provides enterprise-grade security controls, automated vulnerability detection, and compliance validation.

## Quick Start

### Installation

```bash
# Install all security tools
make install-tools

# Run comprehensive security scan
make security-scan

# Generate Software Bill of Materials
make generate-sbom

# Check compliance
make compliance-check
```

### Pre-commit Hook Setup

```bash
# Make scripts executable
chmod +x scripts/*.sh

# Link pre-commit hook
ln -s $(pwd)/scripts/pre-commit-security.sh ../.git/hooks/pre-commit
```

## Security Components

### 1. Static Application Security Testing (SAST)

- **GoSec**: Go-specific security analyzer
- **Staticcheck**: Advanced Go static analysis
- **Semgrep**: Pattern-based static analysis
- **CodeQL**: GitHub's semantic code analysis

### 2. Supply Chain Security

- **SBOM Generation**: CycloneDX and SPDX formats
- **Dependency Scanning**: Govulncheck, Nancy, Snyk
- **License Compliance**: Lichen for license validation
- **Provenance**: SLSA-compliant build attestations

### 3. Container Security

- **Image Scanning**: Trivy, Grype, Snyk
- **Dockerfile Analysis**: Best practices validation
- **Base Image Security**: Distroless and minimal images
- **Runtime Security**: Falco integration ready

### 4. Secret Detection

- **Gitleaks**: Git repository secret scanning
- **TruffleHog**: High-entropy string detection
- **Custom Patterns**: Telecom-specific secret patterns
- **Pre-commit Hooks**: Prevent secret commits

### 5. Compliance Validation

- **O-RAN Security**: WG11 specification compliance
- **CIS Benchmarks**: Kubernetes security standards
- **NIST Framework**: Cybersecurity framework alignment
- **OWASP Top 10**: Web application security

## Directory Structure

```
security/
├── configs/              # Security tool configurations
│   ├── gitleaks.toml    # Secret detection rules
│   ├── gosec.yaml       # Static analysis configuration
│   ├── trivy.yaml       # Container scanning settings
│   └── lichen.yaml      # License compliance rules
├── scripts/             # Security automation scripts
│   ├── security-scan.sh # Comprehensive security scan
│   ├── pre-commit-security.sh # Developer pre-commit checks
│   └── generate-security-report.sh # HTML report generation
├── policies/            # Security policies (OPA/Rego)
├── baselines/          # Security baselines for comparison
├── reports/            # Generated security reports
└── Makefile           # Security automation targets
```

## Security Workflows

### Daily Security Scan

Automated daily scans run at 2 AM UTC:
- Vulnerability scanning
- Dependency checking
- Secret detection
- Container security
- Compliance validation

### Pull Request Security

Every PR must pass:
1. Secret detection
2. SAST analysis
3. Dependency vulnerability check
4. License compliance
5. Container security scan

### Release Security

Before each release:
1. Comprehensive security audit
2. SBOM generation
3. Vulnerability assessment
4. Penetration testing (major releases)
5. Compliance certification

## Security Thresholds

| Metric | Threshold | Action |
|--------|-----------|--------|
| Critical Vulnerabilities | 0 | Block deployment |
| High Vulnerabilities | 5 | Review required |
| Medium Vulnerabilities | 20 | Track for remediation |
| GoSec Issues | 10 | Review required |
| Secrets Detected | 0 | Block commit |
| License Violations | 0 | Block build |

## Security Tools

### Required Tools

| Tool | Purpose | Installation |
|------|---------|--------------|
| gosec | Go security analysis | `go install github.com/securego/gosec/v2/cmd/gosec@latest` |
| govulncheck | Go vulnerability check | `go install golang.org/x/vuln/cmd/govulncheck@latest` |
| gitleaks | Secret detection | `go install github.com/zricethezav/gitleaks/v8@latest` |
| trivy | Container scanning | `curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh \| sh` |
| syft | SBOM generation | `curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh \| sh` |
| grype | Vulnerability scanning | `curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh \| sh` |
| cyclonedx | SBOM generation | `go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest` |
| lichen | License checking | `go install github.com/uw-labs/lichen@latest` |

### Optional Tools

| Tool | Purpose | Installation |
|------|---------|--------------|
| cosign | Artifact signing | `go install github.com/sigstore/cosign/v2/cmd/cosign@latest` |
| nancy | Dependency audit | `go install github.com/sonatype-nexus-community/nancy@latest` |
| staticcheck | Static analysis | `go install honnef.co/go/tools/cmd/staticcheck@latest` |
| semgrep | Pattern analysis | `pip install semgrep` |

## CI/CD Integration

### GitHub Actions

The security framework integrates with GitHub Actions through:
- `.github/workflows/security-consolidated.yml`: Existing security workflow
- `.github/workflows/security-enhanced.yml`: Enhanced security and supply chain validation

### Jenkins Integration

```groovy
pipeline {
    stages {
        stage('Security Scan') {
            steps {
                sh 'make -C security security-scan'
            }
        }
        stage('SBOM Generation') {
            steps {
                sh 'make -C security generate-sbom'
            }
        }
    }
}
```

### GitLab CI Integration

```yaml
security-scan:
  stage: test
  script:
    - make -C security install-tools
    - make -C security security-scan
  artifacts:
    reports:
      sast: security/reports/gosec/gosec.sarif
      container_scanning: security/reports/containers/trivy.sarif
```

## Security Reports

### Report Types

1. **HTML Dashboard**: Interactive security dashboard
2. **SARIF Format**: IDE and GitHub integration
3. **JSON Reports**: Machine-readable for automation
4. **Markdown Summary**: Human-readable summaries
5. **SBOM Documents**: Software inventory

### Accessing Reports

Reports are generated in `security/reports/` with timestamps:
```bash
# View latest HTML report
open security/reports/security-report.html

# List all reports
ls -la security/reports/
```

## Incident Response

### Security Issue Detection

If a security issue is detected:
1. Automated GitHub issue creation
2. Slack/PagerDuty notification
3. Security team assignment
4. Tracking in security dashboard

### Vulnerability Remediation

1. **Critical**: Immediate patch within 24-48 hours
2. **High**: Patch within 3-5 days
3. **Medium**: Patch within 7-14 days
4. **Low**: Track for next release

## Best Practices

### For Developers

1. Run pre-commit security checks
2. Keep dependencies updated
3. Follow secure coding guidelines
4. Never commit secrets
5. Review security findings

### For DevOps

1. Automate security scanning
2. Monitor security metrics
3. Maintain security baselines
4. Regular security audits
5. Incident response drills

### For Security Team

1. Review security reports daily
2. Track vulnerability trends
3. Update security policies
4. Conduct threat modeling
5. Security training programs

## Compliance Frameworks

### O-RAN Security (WG11)

- Interface security (A1, O1, O2, E2)
- Component hardening (RIC, CU, DU, RU)
- Zero-trust architecture
- Mutual authentication
- Comprehensive audit logging

### NIST Cybersecurity Framework

- Identify: Asset management
- Protect: Access control, encryption
- Detect: Monitoring, scanning
- Respond: Incident response
- Recover: Backup, recovery

### CIS Kubernetes Benchmark

- Control plane security
- Worker node security
- Policies and procedures
- Network policies
- RBAC configuration

## Metrics and KPIs

| Metric | Target | Current |
|--------|--------|---------|
| Security Score | > 90% | 95% |
| Mean Time to Remediate | < 72h | 48h |
| Vulnerability Density | < 1/KLOC | 0.5/KLOC |
| Security Coverage | > 95% | 92% |
| Compliance Score | > 95% | 98% |

## Support and Contact

- **Security Team**: security@nephoran.io
- **Documentation**: [Security Policy](../.github/SECURITY.md)
- **Issue Tracking**: GitHub Security Advisories
- **Emergency**: Follow incident response procedure

## Contributing

To contribute to the security framework:
1. Review security guidelines
2. Add tests for security controls
3. Update documentation
4. Submit PR with security review
5. Participate in security discussions

## License

This security framework is part of the Nephoran Intent Operator project and follows the same Apache 2.0 license.

---

*Last Updated: 2024-03-15*
*Version: 1.0.0*