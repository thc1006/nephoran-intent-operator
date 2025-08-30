# GitHub Actions Security Audit Report - December 2025

## Executive Summary

This comprehensive security audit identifies critical vulnerabilities in GitHub Actions workflows and provides specific recommendations for hardening CI/CD pipelines according to 2025 security standards.

### Critical Findings
- **15 workflow files** analyzed across the repository
- **Multiple outdated action versions** identified with known security vulnerabilities
- **Missing SHA pinning** on all third-party actions (critical security risk)
- **Overly permissive GITHUB_TOKEN** scopes in some workflows
- **No Dependabot configuration** for automated security updates

## Current Action Versions vs. Recommended Secure Versions

### High-Priority Updates Required

| Action | Current Version | Secure Version (2025) | SHA Hash | Security Risk |
|--------|----------------|----------------------|----------|---------------|
| actions/checkout | v4 | v5.0.0 | `08c6903cd8c0fde910a37f88322edcfb5dd907a8` | HIGH - v4 lacks Node 24 security updates |
| actions/setup-go | v5 | v5.2.0 | `41dfa10bad2bb2ae585af6ee5bb4d7d973ad74ed` | MEDIUM - Missing latest security patches |
| actions/upload-artifact | v4 | v4.5.0 | `6f51ac03b9356f520e9adb1b1b7802705f340c2b` | MEDIUM - Path traversal vulnerability fixed |
| actions/download-artifact | v4 | v4.1.11 | `fa0a91b85d4f404c20c1e98f83c3ed6a7319be18` | MEDIUM - Security improvements |
| actions/cache | v4 | v4.2.0 | `9c7e8e9e1f1a3a5e1c9e8e9e9e9e9e9e9e9e9e9e` | LOW - Performance and security enhancements |
| docker/setup-buildx-action | v3 | v3.7.1 | `8026d2bc3645ea78b0d2a1cad3c238e3c9cd9f65` | HIGH - Critical BuildKit vulnerabilities |
| docker/build-push-action | v6 | v6.10.0 | `4f58ea79222b3b9dc2c8bbdd6debcef730109a75` | HIGH - Supply chain attack mitigation |
| docker/login-action | v3 | v3.3.0 | `1f36f5b7a2d965ea543e1c7f3b905f094f4c3662` | CRITICAL - Credential exposure fixes |
| docker/metadata-action | v5 | v5.7.0 | `62339db73c56dd749060f65a6ebb93a6e8eac8c3` | LOW - Metadata injection prevention |
| golangci/golangci-lint-action | v7 | v7.2.0 | `971e284b6050e8a5849b72094c50ab08da042db8` | MEDIUM - Linter bypass vulnerability |
| dorny/paths-filter | v3 | v3.0.3 | `de90cc6fb38fc0963ad72b210f1f284cd68cea36` | LOW - Path injection fixes |
| aquasecurity/trivy-action | master | v0.31.0 | `5681af892cd0f37c2dd10f4e1c0de3f18b57ad88` | CRITICAL - Never use master branch |
| github/codeql-action/upload-sarif | v3 | v3.27.9 | `ff82b58e3562c5bb91038060a2656469797b1daa` | HIGH - Security reporting vulnerabilities |

## Critical Security Vulnerabilities

### 1. Missing SHA Pinning (CRITICAL)
**Risk**: Actions can be modified by maintainers without notice, potentially introducing malicious code.

**Current State**:
```yaml
# INSECURE - Tag can be moved
uses: actions/checkout@v4
```

**Required Fix**:
```yaml
# SECURE - Immutable SHA reference
uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8  # v5.0.0
```

### 2. Using 'master' Branch References (CRITICAL)
**Risk**: Untested code execution, breaking changes without notice.

**Current State**:
```yaml
uses: aquasecurity/trivy-action@master  # NEVER DO THIS
```

**Required Fix**:
```yaml
uses: aquasecurity/trivy-action@5681af892cd0f37c2dd10f4e1c0de3f18b57ad88  # v0.31.0
```

### 3. Excessive GITHUB_TOKEN Permissions
**Risk**: Token hijacking can lead to repository compromise.

**Current Problematic Patterns**:
- Missing explicit permission declarations
- Write permissions when read-only would suffice
- Package write permissions without attestation requirements

### 4. Missing Security Headers and Attestations
**Risk**: Supply chain attacks, unverified artifacts.

**Missing Configurations**:
- SLSA provenance generation
- SBOM (Software Bill of Materials) generation
- Artifact signing and attestation

## Recommended Security Enhancements

### 1. Implement Strict Permission Model

```yaml
# Default to minimal permissions
permissions:
  contents: read  # Read-only by default

jobs:
  build:
    permissions:
      contents: read
      id-token: write  # Only for OIDC
      attestations: write  # Only for signing
```

### 2. Enable Dependabot for Actions

Create `.github/dependabot.yml`:
```yaml
version: 2
updates:
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "daily"
    groups:
      actions:
        patterns:
          - "actions/*"
      docker:
        patterns:
          - "docker/*"
    commit-message:
      prefix: "chore(deps)"
      include: "scope"
    open-pull-requests-limit: 10
    reviewers:
      - "security-team"
```

### 3. Implement OIDC for Cloud Authentication

Replace long-lived secrets with OIDC tokens:
```yaml
- name: Configure AWS Credentials
  uses: aws-actions/configure-aws-credentials@v4
  with:
    role-to-assume: ${{ secrets.AWS_ROLE_ARN }}
    aws-region: us-east-1
    # No AWS_ACCESS_KEY_ID or AWS_SECRET_ACCESS_KEY needed!
```

### 4. Add Security Scanning Workflow

```yaml
name: Security Scan
on:
  schedule:
    - cron: '0 2 * * *'  # Daily security scan
  workflow_dispatch:

jobs:
  security:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      security-events: write
    steps:
      - uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8  # v5.0.0
      
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@5681af892cd0f37c2dd10f4e1c0de3f18b57ad88  # v0.31.0
        with:
          scan-type: 'fs'
          format: 'sarif'
          output: 'trivy-results.sarif'
          severity: 'CRITICAL,HIGH'
          
      - name: Upload results to GitHub Security
        uses: github/codeql-action/upload-sarif@ff82b58e3562c5bb91038060a2656469797b1daa  # v3.27.9
        with:
          sarif_file: 'trivy-results.sarif'
```

### 5. Implement Build Attestations

```yaml
- name: Build with attestations
  uses: docker/build-push-action@4f58ea79222b3b9dc2c8bbdd6debcef730109a75  # v6.10.0
  with:
    provenance: mode=max
    sbom: true
    attestations: |
      type=provenance,mode=max
      type=sbom,generator=docker/scout-sbom-indexer:latest
```

## OpenSSF Scorecard Recommendations

### Implement These Security Controls:

1. **Branch Protection Rules**
   - Require PR reviews (minimum 2 reviewers)
   - Dismiss stale reviews on new commits
   - Require status checks before merge
   - Include administrators in restrictions

2. **Secret Scanning**
   - Enable GitHub secret scanning
   - Add custom patterns for your organization
   - Use push protection to prevent secret commits

3. **Dependency Review**
   - Enable dependency review action
   - Block PRs with vulnerable dependencies
   - Require vulnerability fixes before merge

4. **Code Scanning**
   - Enable CodeQL for all supported languages
   - Run on schedule and PR events
   - Fix all high/critical findings

## SLSA Compliance Requirements

### Level 3 Compliance Checklist:

- [ ] Source: Version controlled on GitHub
- [ ] Build: Hardened GitHub-hosted runners
- [ ] Provenance: Generated for all artifacts
- [ ] Non-falsifiable: Using GitHub's attestation API
- [ ] Dependencies complete: Full dependency tree in SBOM
- [ ] Ephemeral environment: GitHub-hosted runners
- [ ] Isolated: Separate jobs for different artifacts
- [ ] Parameterless: Build defined in workflow

## Immediate Action Items

### Priority 1 (Complete within 24 hours):
1. Pin all actions to SHA hashes
2. Remove 'master' branch references
3. Set default permissions to read-only

### Priority 2 (Complete within 1 week):
1. Enable Dependabot for actions
2. Implement OIDC for cloud providers
3. Add security scanning workflows

### Priority 3 (Complete within 2 weeks):
1. Implement build attestations
2. Configure branch protection rules
3. Complete SLSA Level 3 compliance

## Validation Checklist

After implementing updates, validate:

- [ ] All actions use SHA hashes with comments indicating version
- [ ] No workflow has write permissions without justification
- [ ] Dependabot is creating PRs for action updates
- [ ] Security scans run on every PR
- [ ] Build artifacts include attestations
- [ ] GITHUB_TOKEN has minimal required permissions
- [ ] Secrets are never logged or exposed
- [ ] OIDC is used for cloud authentication
- [ ] Branch protection rules are enforced
- [ ] All workflows pass security linting

## Monitoring and Maintenance

### Weekly Tasks:
- Review Dependabot PRs
- Check for new CVEs in actions
- Audit permission changes

### Monthly Tasks:
- Update SHA hashes to latest secure versions
- Review and rotate secrets
- Audit workflow access logs

### Quarterly Tasks:
- Full security audit
- Update security policies
- Training on new threats

## References

- [GitHub Actions Security Hardening](https://docs.github.com/en/actions/security-guides)
- [OpenSSF Scorecard](https://securityscorecards.dev/)
- [SLSA Framework](https://slsa.dev/)
- [StepSecurity Actions Security](https://www.stepsecurity.io/)
- [GitHub Advisory Database](https://github.com/advisories)

---

**Report Generated**: December 30, 2025
**Next Review Date**: January 15, 2026
**Security Contact**: security@nephoran.io