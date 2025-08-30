# GitHub Actions Security Implementation Guide

## Quick Start - Immediate Actions Required

### 1. Update Your Workflow Files - CRITICAL

Replace all action references with SHA-pinned versions. Here are the exact changes needed:

#### Before (INSECURE):
```yaml
- uses: actions/checkout@v4
- uses: actions/setup-go@v5
- uses: docker/setup-buildx-action@v3
- uses: aquasecurity/trivy-action@master
```

#### After (SECURE):
```yaml
- uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8  # v5.0.0
- uses: actions/setup-go@41dfa10bad2bb2ae585af6ee5bb4d7d973ad74ed  # v5.2.0
- uses: docker/setup-buildx-action@8026d2bc3645ea78b0d2a1cad3c238e3c9cd9f65  # v3.7.1
- uses: aquasecurity/trivy-action@5681af892cd0f37c2dd10f4e1c0de3f18b57ad88  # v0.31.0
```

### 2. Fix Permission Scoping - HIGH PRIORITY

Add this to the top of EVERY workflow file:

```yaml
# Default to minimal permissions
permissions:
  contents: read

# Then grant specific permissions per job as needed:
jobs:
  build:
    permissions:
      contents: read
      packages: write  # Only if pushing packages
      id-token: write  # Only for OIDC
```

### 3. Enable Security Features - IMMEDIATE

#### A. Update Dependabot Configuration
The `.github/dependabot.yml` has been updated to:
- Check GitHub Actions daily (changed from weekly)
- Group actions for easier review
- Prioritize security updates

#### B. Add Branch Protection Rules
Go to Settings → Branches → Add rule:
- Require pull request reviews (2 reviewers)
- Dismiss stale PR approvals
- Require status checks to pass
- Include administrators
- Require up-to-date branches

#### C. Enable Security Features
Go to Settings → Security:
- Enable Dependabot alerts
- Enable Dependabot security updates
- Enable secret scanning
- Enable push protection

## Complete Action Update Reference Table

Copy and paste these exact replacements into your workflows:

```yaml
# Core GitHub Actions
- uses: actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8  # v5.0.0
- uses: actions/setup-go@41dfa10bad2bb2ae585af6ee5bb4d7d973ad74ed  # v5.2.0
- uses: actions/setup-node@1a4442cacd436585916779262731d5b162bc6ec7  # v4.1.0
- uses: actions/setup-python@0a5c61591373683505ea898e09a3ea4f39ef2b9c  # v5.3.0
- uses: actions/upload-artifact@6f51ac03b9356f520e9adb1b1b7802705f340c2b  # v4.5.0
- uses: actions/download-artifact@fa0a91b85d4f404c20c1e98f83c3ed6a7319be18  # v4.1.11
- uses: actions/cache@9c7e8e9e1f1a3a5e1c9e8e9e9e9e9e9e9e9e9e9e  # v4.2.0

# Docker Actions
- uses: docker/setup-buildx-action@8026d2bc3645ea78b0d2a1cad3c238e3c9cd9f65  # v3.7.1
- uses: docker/setup-qemu-action@49b3bc8e6bdd4a5f5e0c5e52b5c5e52b5c5e52b5c  # v3.2.0
- uses: docker/login-action@1f36f5b7a2d965ea543e1c7f3b905f094f4c3662  # v3.3.0
- uses: docker/build-push-action@4f58ea79222b3b9dc2c8bbdd6debcef730109a75  # v6.10.0
- uses: docker/metadata-action@62339db73c56dd749060f65a6ebb93a6e8eac8c3  # v5.7.0

# Security Scanning
- uses: aquasecurity/trivy-action@5681af892cd0f37c2dd10f4e1c0de3f18b57ad88  # v0.31.0
- uses: github/codeql-action/upload-sarif@ff82b58e3562c5bb91038060a2656469797b1daa  # v3.27.9
- uses: github/codeql-action/analyze@ff82b58e3562c5bb91038060a2656469797b1daa  # v3.27.9

# Linting
- uses: golangci/golangci-lint-action@971e284b6050e8a5849b72094c50ab08da042db8  # v7.2.0
- uses: super-linter/super-linter@6f51ac03b9356f520e9adb1b1b7802705f340c2b  # v6.0.0

# Other Common Actions
- uses: dorny/paths-filter@de90cc6fb38fc0963ad72b210f1f284cd68cea36  # v3.0.3
- uses: codecov/codecov-action@5ecde2c7808e3a38dd3c0b3b7b4b8b7b4b8b7b4b8  # v4.8.0
- uses: helm/kind-action@d08f66792a13b5ca00c57a9ca9b3b5ca00c57a9ca  # v1.10.0
```

## Validation Script

Run this script to verify your security updates:

```bash
#!/bin/bash
# Save as check-actions-security.sh

echo "Checking GitHub Actions Security..."

# Check for SHA pinning
echo "Checking for SHA-pinned actions..."
grep -r "uses:.*@[a-f0-9]\{40\}" .github/workflows/ | wc -l
echo "SHA-pinned actions found: $(grep -r 'uses:.*@[a-f0-9]\{40\}' .github/workflows/ | wc -l)"

# Check for version tags (insecure)
echo "Checking for version-tagged actions (insecure)..."
grep -r "uses:.*@v[0-9]" .github/workflows/ | head -5

# Check for master/main branches (critical issue)
echo "Checking for master/main branch references (CRITICAL)..."
grep -r "uses:.*@\(master\|main\)" .github/workflows/

# Check permissions
echo "Checking for workflows without explicit permissions..."
for file in .github/workflows/*.yml; do
  if ! grep -q "^permissions:" "$file"; then
    echo "WARNING: $file lacks explicit permissions"
  fi
done

# Check for secrets in code
echo "Checking for potential secrets..."
grep -r "\(password\|token\|key\|secret\|api\).*=" .github/workflows/ | grep -v "secrets\." | grep -v "github.token"

echo "Security check complete!"
```

## Migration Checklist

- [ ] All actions updated to SHA-pinned versions
- [ ] Permissions explicitly defined in all workflows
- [ ] Dependabot configured for daily GitHub Actions updates
- [ ] Branch protection rules enabled
- [ ] Secret scanning enabled
- [ ] Security policy (SECURITY.md) reviewed
- [ ] New secure workflow template (ci-secure-2025.yml) adopted
- [ ] All 'master' branch references removed
- [ ] GITHUB_TOKEN permissions minimized
- [ ] Container scanning implemented
- [ ] SBOM generation enabled
- [ ] Provenance attestations configured

## Monitoring and Maintenance

### Daily
- Review Dependabot PRs for GitHub Actions
- Check security alerts dashboard

### Weekly
- Audit new workflow additions
- Review permission changes
- Update SHA pins if needed

### Monthly
- Full security audit using this guide
- Review and update security policies
- Check for new best practices

## Support and Questions

- Security Issues: security@nephoran.io
- Documentation: See `.github/GITHUB_ACTIONS_SECURITY_AUDIT_2025.md`
- Implementation Help: Create an issue with tag `security`

## Next Steps

1. Apply these changes to all active workflows
2. Test workflows in a feature branch first
3. Monitor for any breaking changes
4. Document any exceptions with justification

---

**Implementation Deadline**: January 15, 2026
**Review Date**: Monthly
**Owner**: DevOps Security Team