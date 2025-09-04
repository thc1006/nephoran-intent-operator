# Security Policy

## Reporting Security Vulnerabilities

The Nephoran Intent Operator team takes security seriously. We appreciate your efforts to responsibly disclose your findings and will make every effort to acknowledge your contributions.

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| 0.9.x   | :white_check_mark: |
| < 0.9   | :x:                |

## Reporting a Vulnerability

To report a security vulnerability, please follow these steps:

1. **DO NOT** create a public GitHub issue for the vulnerability
2. Email your findings to security@nephoran.io
3. Include the following information:
   - Type of vulnerability (e.g., XSS, SQL injection, RCE)
   - Full paths of source file(s) related to the vulnerability
   - Location of the affected source code (tag/branch/commit or direct URL)
   - Step-by-step instructions to reproduce the issue
   - Proof-of-concept or exploit code (if possible)
   - Impact of the vulnerability, including how an attacker might exploit it

## Response Timeline

- **Initial Response**: Within 48 hours
- **Vulnerability Assessment**: Within 7 days
- **Patch Development**: Based on severity
  - CRITICAL: Within 24 hours
  - HIGH: Within 7 days
  - MEDIUM: Within 30 days
  - LOW: Within 90 days

## Disclosure Policy

- Security vulnerabilities will be disclosed via GitHub Security Advisories
- We follow a coordinated disclosure timeline:
  1. Security report received
  2. Vulnerability confirmed and assessed
  3. Patch developed and tested
  4. Patch released with security advisory
  5. Public disclosure after 90 days or when patch is available

## Dependency Security Management

### Principles

1. **Minimal Dependencies**: We maintain the smallest possible dependency footprint
2. **Regular Updates**: Dependencies are reviewed and updated monthly
3. **Security First**: Security vulnerabilities take priority over features
4. **Supply Chain Security**: All dependencies are verified and scanned

### Dependency Guidelines

#### Acceptable Dependencies
- Actively maintained (commits within last 6 months)
- Clear security policy and vulnerability disclosure
- Compatible open-source license (MIT, Apache 2.0, BSD)
- Passes security scanning without high/critical vulnerabilities
- Provides clear value that cannot be easily implemented

#### Prohibited Dependencies
- GPL/AGPL licensed packages (incompatible with project license)
- Deprecated or archived projects
- Dependencies with known unpatched vulnerabilities
- Packages without clear maintainership
- Heavy dependencies that can be replaced with lighter alternatives

### Security Scanning

All dependencies are automatically scanned using:
- **Go**: govulncheck, nancy (Sonatype OSS Index)
- **Python**: pip-audit, safety, bandit
- **Containers**: Trivy, Snyk
- **SBOM**: CycloneDX format for supply chain transparency

### Update Process

1. **Monthly Review**: First Tuesday of each month
2. **Automated PR**: Dependabot creates update PRs
3. **Security Updates**: Applied within 48 hours for critical vulnerabilities
4. **Testing**: All updates must pass full test suite
5. **Rollback Plan**: Previous versions maintained for quick rollback

## Recent Security Enhancements (v0.2.0)

### Critical Security Fixes

The latest release includes significant security improvements addressing input validation, command injection prevention, and secure timestamp handling:

#### 1. Enhanced Input Validation and Path Traversal Prevention

**Module**: `internal/patchgen` and `planner/internal/security`

- **Comprehensive Input Validation**: All user inputs validated against strict schemas and security rules
- **Path Traversal Protection**: File paths sanitized using `filepath.Clean()` to prevent directory traversal attacks
- **URL Validation**: URLs parsed and validated against allowed schemes (HTTP/HTTPS only)
- **Node ID Security**: O-RAN node identifiers validated using regex patterns to prevent injection attacks

Implementation:
```go
// Prevent directory traversal attacks
cleanPath := filepath.Clean(path)
if strings.Contains(cleanPath, "..") {
    return ValidationError{
        Field:   "file_path",
        Reason:  "file path contains directory traversal sequences (..)",
    }
}
```

#### 2. Secure Command Execution

**Module**: `internal/patchgen` and `internal/patch`

- **Input Sanitization**: All command inputs validated before execution
- **Command Injection Prevention**: Use of parameterized commands and input validation
- **File Permission Controls**: Proper file permissions (0644 for files, 0755 for directories)
- **Error Handling**: Comprehensive error handling preventing information disclosure

#### 3. Timestamp Security and Collision Prevention

**Module**: `internal/patchgen`

- **RFC3339 Timestamp Format**: All timestamps use standardized RFC3339 format for consistency
- **UTC Timezone Normalization**: All timestamps normalized to UTC preventing timezone attacks
- **Collision-Resistant Naming**: Package names include timestamps to prevent naming collisions
- **Timestamp Validation**: Timestamps validated to prevent replay attacks and future-dating

Implementation:
```go
// Secure timestamp generation
"nephoran.io/generated-at": time.Now().UTC().Format(time.RFC3339)
```

#### 4. JSON Schema Validation Enhancement

**Module**: `internal/patchgen/validator.go`

- **JSON Schema 2020-12 Compliance**: Uses latest JSON Schema specification
- **Strict Validation**: All intent data validated against comprehensive schema
- **Type Safety**: Strong typing prevents injection through type confusion
- **Bounds Checking**: Numeric values validated within acceptable ranges

#### 5. Secure Logging and Audit Trail

**Module**: `planner/internal/security/validation.go`

- **Log Injection Prevention**: All logged values sanitized to prevent log injection
- **Sensitive Data Filtering**: Automatic truncation and sanitization of logged values
- **Structured Logging**: Use of structured logging to prevent format string attacks

### Migration from internal/patch to internal/patchgen

The migration introduced several security enhancements:

- **Enhanced Validation**: Comprehensive JSON Schema validation with OWASP compliance
- **Secure File Handling**: Path traversal prevention, permission controls, error handling
- **Timestamp Security**: RFC3339 format with collision prevention and validation

**Migration Steps**:
1. Update imports: `internal/patch` → `internal/patchgen`
2. All intent data must pass JSON Schema validation
3. Update error handling for new validation errors
4. Update tests to account for enhanced security validation

## Security Measures

### Architecture Security

The Nephoran Intent Operator implements defense-in-depth with multiple security layers:

#### 1. Zero Trust Architecture
- Never trust, always verify principle
- Continuous authentication and authorization
- Micro-segmentation for network isolation
- Least privilege access model

#### 2. Container Security
- Non-root user execution (UID > 10000)
- Read-only root filesystem
- All capabilities dropped
- Seccomp and AppArmor profiles enabled
- Distroless base images

#### 3. Network Security
- TLS 1.3 minimum for all communications
- Mutual TLS (mTLS) for service-to-service
- Network policies with default deny
- Egress restrictions to approved endpoints

#### 4. Secrets Management
- AES-256-GCM encryption for secrets at rest
- Automatic rotation every 90 days
- HashiCorp Vault integration
- No plaintext secrets in code or configuration

#### 5. O-RAN Security Compliance
- WG11 security specifications implementation
- Interface-specific security (A1, O1, O2, E2)
- IPsec for CU-DU communications
- xApp sandboxing and code signing

### Compliance Standards

The platform maintains compliance with:

- **SOC2 Type 2**: Continuous monitoring and audit controls
- **ISO 27001:2022**: Information security management system
- **PCI-DSS v4**: Payment card data protection
- **GDPR**: Data privacy and protection
- **O-RAN Alliance WG11**: Telecommunications security specifications
- **3GPP TS 33.501**: 5G security architecture

### Security Testing

Continuous security validation through:

#### Automated Testing
- Static Application Security Testing (SAST) with SonarQube
- Dynamic Application Security Testing (DAST) with OWASP ZAP
- Container vulnerability scanning with Trivy
- Dependency scanning with Snyk
- Secret scanning with GitLeaks

#### Manual Testing
- Quarterly penetration testing
- Annual security audits
- Red team exercises
- Threat modeling sessions

### Vulnerability Management

- **Scanning Frequency**: Daily automated scans
- **Critical Vulnerabilities**: Blocked at build time
- **High Vulnerabilities**: Require approval before deployment
- **Patch Management**: Automated patching with rollback capability
- **CVE Tracking**: Continuous monitoring of CVE databases

### Access Control

#### RBAC Policies
- No wildcard permissions
- Least privilege principle
- Regular access reviews
- Automated deprovisioning

#### Authentication
- Multi-factor authentication (MFA) required
- OAuth2/OIDC for external access
- Certificate-based authentication for services
- Session timeout after 15 minutes of inactivity

### Incident Response

#### Response Team
- 24/7 on-call rotation
- Defined escalation procedures
- Regular incident response drills

#### Response Procedures
1. **Detection**: SIEM alerts and monitoring
2. **Containment**: Automated isolation of affected components
3. **Eradication**: Root cause analysis and remediation
4. **Recovery**: Validated restoration procedures
5. **Lessons Learned**: Post-incident review and improvement

### Data Protection

#### Encryption
- **At Rest**: AES-256-GCM with HSM key management
- **In Transit**: TLS 1.3 minimum
- **Key Management**: Automated rotation and secure storage

#### Data Classification
- Public, Internal, Confidential, Restricted
- Mandatory data tagging
- Access controls based on classification

#### Privacy
- GDPR compliance with privacy by design
- Data minimization principles
- Right to erasure implementation
- Data portability support

## Security Contacts

- **Security Team Email**: security@nephoran.io
- **Security Updates**: https://github.com/nephoran/nephoran-intent-operator/security/advisories
- **Bug Bounty Program**: https://nephoran.io/security/bug-bounty

## Acknowledgments

We would like to thank the following individuals for responsibly disclosing security vulnerabilities:

- [Security Hall of Fame](https://nephoran.io/security/hall-of-fame)

## Security Testing and Validation

### Automated Security Tests

**Location**: `planner/internal/security/`

#### Test Coverage:
- Input validation boundary testing
- Path traversal attack simulation  
- Injection attack prevention verification
- Timestamp manipulation testing
- File permission verification

#### Running Security Tests:
```bash
# Run all security tests
go test ./planner/internal/security/... -v

# Run specific security test suites
go test ./planner/internal/security/ -run TestPathTraversalPrevention
go test ./planner/internal/security/ -run TestInputValidation
```

### Security Metrics and Monitoring

Monitor these security-related metrics:
- `nephoran_security_validation_failures_total`: Input validation failures
- `nephoran_security_path_traversal_attempts_total`: Path traversal attempts
- `nephoran_security_injection_attempts_total`: Injection attack attempts
- `nephoran_security_timestamp_violations_total`: Timestamp validation failures

## Security Resources

- [Security Best Practices](docs/security/best-practices.md)
- [Hardening Guide](docs/security/hardening.md)
- [Threat Model](docs/security/threat-model.md)
- [Compliance Documentation](docs/security/compliance.md)
- [Security Audit Report](planner/internal/security/SECURITY_TESTS_SUMMARY.md)

## PGP Key

For encrypted communications, use our PGP key:

```
-----BEGIN PGP PUBLIC KEY BLOCK-----
[PGP key would be inserted here]
-----END PGP PUBLIC KEY BLOCK-----
```

## Commitment

The Nephoran Intent Operator team is committed to:
- Rapid response to security reports
- Transparent communication about security issues
- Continuous improvement of security posture
- Regular security training for team members
- Collaboration with the security community

Thank you for helping keep the Nephoran Intent Operator and its users safe!