# Software Bill of Materials (SBOM) Generation Guide

**Document Type**: Security Best Practice Guide
**Date**: 2026-02-16
**Purpose**: Track dependencies and vulnerabilities for production security

---

## Overview

A Software Bill of Materials (SBOM) is a **complete inventory of all components** in your application, including direct and transitive dependencies. SBOMs are critical for:

1. **Vulnerability Tracking**: Identify CVEs in dependencies
2. **License Compliance**: Audit open source licenses
3. **Supply Chain Security**: Detect compromised packages
4. **Regulatory Compliance**: Required by Executive Order 14028 (US Federal)

---

## SBOM Generation Methods

### Method 1: Syft (Recommended - Most Comprehensive)

**Install Syft**:
```bash
# Option A: Binary download
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin

# Option B: Homebrew (macOS/Linux)
brew install syft

# Option C: Docker
docker pull anchore/syft:latest
```

**Generate SBOM** (SPDX format):
```bash
cd /home/thc1006/dev/nephoran-intent-operator

# Generate SBOM for entire project
syft . -o spdx-json > sbom-nephoran-operator.spdx.json

# Generate SBOM for specific Go binary
syft bin/manager -o spdx-json > sbom-manager-binary.spdx.json

# Generate SBOM for container image
syft docker.io/nephoran-operator:v2.0 -o spdx-json > sbom-container.spdx.json

# Multiple output formats
syft . -o json -o cyclonedx-json -o spdx-json -o table
```

**Output Formats**:
- `spdx-json`: SPDX 2.3 (industry standard, recommended)
- `cyclonedx-json`: CycloneDX 1.5 (OWASP standard)
- `json`: Syft native format
- `table`: Human-readable table
- `text`: Simple text list

---

### Method 2: Go Modules Native (Go dependencies only)

**Generate Go dependency list**:
```bash
cd /home/thc1006/dev/nephoran-intent-operator

# List all dependencies with versions
go list -m all > sbom-go-modules.txt

# List with JSON metadata
go list -m -json all > sbom-go-modules.json

# Show dependency graph
go mod graph > sbom-dependency-graph.txt
```

**Limitation**: Only tracks Go modules, misses OS packages and container layers.

---

### Method 3: Grype (Vulnerability Scanning)

**Install Grype**:
```bash
curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin
```

**Scan for vulnerabilities**:
```bash
# Scan current directory
grype .

# Scan specific SBOM
grype sbom:./sbom-nephoran-operator.spdx.json

# Scan container image
grype docker.io/nephoran-operator:v2.0

# Output JSON report
grype . -o json > vulnerability-report.json

# Only show HIGH and CRITICAL
grype . --fail-on high
```

---

## Current Nephoran Operator SBOM

### Summary (as of 2026-02-16)

```yaml
Project: Nephoran Intent Operator
Version: 2.0.0
Go Version: 1.24.x

Direct Dependencies: ~40
Total Dependencies: ~150 (including transitive)

Key Components:
  - sigs.k8s.io/controller-runtime v0.23.1
  - k8s.io/api v0.35.1
  - k8s.io/apimachinery v0.35.1
  - k8s.io/client-go v0.35.1
  - github.com/go-logr/logr v1.4.2
  - github.com/prometheus/client_golang v1.20.5

Container Base: gcr.io/distroless/static:nonroot
OS Packages: 0 (distroless has no package manager)
```

### Generate Now (Manual Command)

```bash
# Prerequisites: Install syft
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin

# Generate SBOM
cd /home/thc1006/dev/nephoran-intent-operator
syft . -o spdx-json=sbom-nephoran-operator-2.0.0.spdx.json

# Verify SBOM
cat sbom-nephoran-operator-2.0.0.spdx.json | jq '.packages | length'
# Expected: ~150 packages

# Scan for vulnerabilities
grype sbom:./sbom-nephoran-operator-2.0.0.spdx.json --fail-on critical
```

---

## SBOM Integration into CI/CD

### GitHub Actions Workflow

Create `.github/workflows/sbom-generation.yml`:

```yaml
name: Generate SBOM and Scan Vulnerabilities

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 0 * * 0'  # Weekly on Sunday

jobs:
  sbom:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Syft
        run: |
          curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | \
            sh -s -- -b /usr/local/bin

      - name: Generate SBOM
        run: |
          syft . -o spdx-json=sbom.spdx.json

      - name: Upload SBOM Artifact
        uses: actions/upload-artifact@v4
        with:
          name: sbom-spdx
          path: sbom.spdx.json
          retention-days: 90

      - name: Install Grype
        run: |
          curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | \
            sh -s -- -b /usr/local/bin

      - name: Scan for Vulnerabilities
        run: |
          grype sbom:./sbom.spdx.json --fail-on high -o json > vulnerabilities.json

      - name: Upload Vulnerability Report
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: vulnerability-report
          path: vulnerabilities.json
          retention-days: 90

      - name: Comment PR with Summary
        if: github.event_name == 'pull_request'
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const vulns = JSON.parse(fs.readFileSync('vulnerabilities.json'));
            const summary = `## SBOM Vulnerability Scan

            **Critical**: ${vulns.matches.filter(m => m.vulnerability.severity === 'Critical').length}
            **High**: ${vulns.matches.filter(m => m.vulnerability.severity === 'High').length}
            **Medium**: ${vulns.matches.filter(m => m.vulnerability.severity === 'Medium').length}
            **Low**: ${vulns.matches.filter(m => m.vulnerability.severity === 'Low').length}
            `;

            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: summary
            });
```

---

## SBOM Storage and Versioning

### Recommended Storage

1. **Git Repository** (small SBOMs < 1 MB):
   ```bash
   mkdir -p docs/sbom
   syft . -o spdx-json > docs/sbom/nephoran-operator-v2.0.0.spdx.json
   git add docs/sbom/
   git commit -m "docs(sbom): add SBOM for v2.0.0"
   ```

2. **GitHub Releases** (preferred for large SBOMs):
   ```bash
   syft . -o spdx-json > sbom.spdx.json
   gh release create v2.0.0 \
     --title "Nephoran Operator v2.0.0" \
     --notes "See CHANGELOG.md" \
     sbom.spdx.json
   ```

3. **Artifact Registry** (for container images):
   ```bash
   # Attach SBOM to container image (ORAS)
   oras attach \
     docker.io/nephoran-operator:v2.0.0 \
     --artifact-type application/spdx+json \
     sbom.spdx.json:application/json
   ```

### Naming Convention

```
sbom-<component>-<version>.<format>

Examples:
  sbom-nephoran-operator-2.0.0.spdx.json
  sbom-rag-service-1.5.0.cyclonedx.json
  sbom-weaviate-deployment-1.0.0.spdx.json
```

---

## Vulnerability Management Workflow

### 1. Daily Automated Scan (Dependabot/Renovate)

Enable Dependabot in `.github/dependabot.yml`:

```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 10
    labels:
      - "dependencies"
      - "security"
```

### 2. Weekly Manual Review

```bash
# Step 1: Generate fresh SBOM
syft . -o spdx-json > sbom-current.spdx.json

# Step 2: Scan for vulnerabilities
grype sbom:./sbom-current.spdx.json -o json > vulns-current.json

# Step 3: Review CRITICAL and HIGH findings
cat vulns-current.json | jq '.matches[] | select(.vulnerability.severity == "Critical" or .vulnerability.severity == "High")'

# Step 4: Check CVE databases
# - https://osv.dev/
# - https://nvd.nist.gov/
# - https://github.com/advisories

# Step 5: Update dependencies if needed
go get -u <module>@<safe-version>
go mod tidy
```

### 3. Incident Response (CVE discovered)

```bash
# Step 1: Identify affected versions
grype . --only-fixed

# Step 2: Check if CVE applies to your usage
# (Not all CVEs are exploitable in every context)

# Step 3: Update dependency
go get -u <vulnerable-module>@<patched-version>

# Step 4: Verify fix
grype . | grep <CVE-ID>

# Step 5: Generate new SBOM
syft . -o spdx-json > sbom-patched.spdx.json

# Step 6: Deploy patched version
make docker-build IMG=nephoran-operator:v2.0.1-security
make deploy IMG=nephoran-operator:v2.0.1-security

# Step 7: Update security advisory
echo "## CVE-YYYY-XXXXX Patched
Fixed in v2.0.1 by upgrading <module> to <version>
" >> docs/SECURITY.md
```

---

## SBOM Analysis Tools

### 1. Dependency Track (OWASP)

**Self-hosted vulnerability management platform**:

```bash
# Install via Docker Compose
curl -LO https://dependencytrack.org/docker-compose.yml
docker-compose up -d

# Upload SBOM via UI or API
curl -X POST "http://localhost:8080/api/v1/bom" \
  -H "X-Api-Key: $DT_API_KEY" \
  -F "project=$PROJECT_UUID" \
  -F "bom=@sbom.spdx.json"
```

**Features**:
- Vulnerability database aggregation (NVD, OSV, GitHub)
- Policy engine (fail builds on HIGH severity)
- License compliance tracking
- Portfolio management (multiple projects)

### 2. Snyk (SaaS)

```bash
# Install Snyk CLI
npm install -g snyk

# Authenticate
snyk auth

# Scan Go project
snyk test

# Monitor project (continuous scanning)
snyk monitor

# Test container image
snyk container test nephoran-operator:v2.0.0
```

### 3. Trivy (Aqua Security)

```bash
# Install Trivy
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin

# Scan filesystem
trivy fs .

# Scan container image
trivy image nephoran-operator:v2.0.0

# Generate SBOM with Trivy
trivy image --format spdx-json -o sbom.spdx.json nephoran-operator:v2.0.0
```

---

## SBOM for Nephoran Deployment

### Components to Track

| Component | Type | SBOM Location |
|-----------|------|---------------|
| Nephoran Intent Operator | Go Binary | `sbom-nephoran-operator.spdx.json` |
| RAG Service (FastAPI) | Python | `sbom-rag-service.spdx.json` |
| Weaviate | Container | `sbom-weaviate-1.34.0.spdx.json` |
| Ollama | Container | `sbom-ollama-0.16.1.spdx.json` |
| Free5GC (78 Nephio packages) | Helm Charts | `sbom-free5gc-3.4.3.spdx.json` |
| OAI RAN | Container | `sbom-oai-ran-2024.w52.spdx.json` |
| Near-RT RIC | Helm Charts | `sbom-ric-plt-l-release.spdx.json` |

### Generate All SBOMs

```bash
#!/bin/bash
# generate-all-sboms.sh

set -e

SBOM_DIR="docs/sbom"
mkdir -p "$SBOM_DIR"

echo "Generating SBOMs for Nephoran deployment..."

# 1. Nephoran Operator
syft . -o spdx-json > "$SBOM_DIR/sbom-nephoran-operator-2.0.0.spdx.json"

# 2. Container images
syft docker.io/weaviate/weaviate:1.34.0 -o spdx-json > "$SBOM_DIR/sbom-weaviate-1.34.0.spdx.json"
syft docker.io/ollama/ollama:0.16.1 -o spdx-json > "$SBOM_DIR/sbom-ollama-0.16.1.spdx.json"

# 3. Free5GC components (example - requires deployment)
# kubectl get deployment -n free5gc -o yaml | syft - -o spdx-json > "$SBOM_DIR/sbom-free5gc-3.4.3.spdx.json"

# 4. Generate summary
echo "## SBOM Generation Summary" > "$SBOM_DIR/README.md"
echo "Generated: $(date)" >> "$SBOM_DIR/README.md"
echo "" >> "$SBOM_DIR/README.md"

for sbom in "$SBOM_DIR"/*.spdx.json; do
  pkg_count=$(cat "$sbom" | jq '.packages | length')
  echo "- $(basename "$sbom"): $pkg_count packages" >> "$SBOM_DIR/README.md"
done

echo "✅ All SBOMs generated in $SBOM_DIR/"
```

---

## Compliance and Reporting

### Executive Summary Report Template

```markdown
# SBOM and Vulnerability Report

**Period**: 2026-02-01 to 2026-02-16
**Project**: Nephoran Intent Operator v2.0.0
**Total Components**: 150 packages

## Vulnerability Summary

| Severity | Count | Resolved | Open |
|----------|-------|----------|------|
| CRITICAL | 0     | 0        | 0    |
| HIGH     | 2     | 1        | 1    |
| MEDIUM   | 5     | 3        | 2    |
| LOW      | 12    | 8        | 4    |

## Open Vulnerabilities (HIGH)

1. **CVE-2024-XXXXX** - golang.org/x/net
   - Severity: HIGH
   - Impact: HTTP/2 denial of service
   - Affected Versions: < v0.20.0
   - Current Version: v0.19.0
   - Remediation: Upgrade to v0.20.0
   - ETA: 2026-02-20
   - Status: PR #123 in review

## License Compliance

| License | Count | Approved |
|---------|-------|----------|
| Apache-2.0 | 85 | ✅ Yes |
| MIT | 45 | ✅ Yes |
| BSD-3-Clause | 15 | ✅ Yes |
| GPL-3.0 | 0 | N/A |

## SBOM Artifacts

- Main SBOM: `sbom-nephoran-operator-2.0.0.spdx.json` (150 packages)
- Vulnerability Report: `vulnerabilities-2026-02-16.json`
- Remediation Plan: [Internal Wiki Link]
```

---

## References

- [SPDX Specification](https://spdx.github.io/spdx-spec/v2.3/)
- [CycloneDX Specification](https://cyclonedx.org/specification/overview/)
- [NTIA SBOM Minimum Elements](https://www.ntia.gov/page/sbom-minimum-elements)
- [Executive Order 14028 (US Federal)](https://www.whitehouse.gov/briefing-room/presidential-actions/2021/05/12/executive-order-on-improving-the-nations-cybersecurity/)
- [Syft Documentation](https://github.com/anchore/syft)
- [Grype Documentation](https://github.com/anchore/grype)

---

**Next Steps**:
1. Install syft: `curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin`
2. Generate SBOM: `syft . -o spdx-json > sbom-nephoran-operator-2.0.0.spdx.json`
3. Scan vulnerabilities: `grype sbom:./sbom-nephoran-operator-2.0.0.spdx.json`
4. Review findings and create remediation plan
