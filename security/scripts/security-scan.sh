#!/bin/bash
# Comprehensive security scanning script for Nephoran Intent Operator

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SECURITY_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$SECURITY_DIR")"
REPORTS_DIR="${SECURITY_DIR}/reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_PREFIX="${REPORTS_DIR}/${TIMESTAMP}"

# Security thresholds
MAX_CRITICAL_VULNS=0
MAX_HIGH_VULNS=5
MAX_MEDIUM_VULNS=20
MAX_GOSEC_ISSUES=10
MAX_SECRETS=0

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create report directories
prepare_directories() {
    log_info "Preparing report directories..."
    mkdir -p "${REPORTS_DIR}"/{code,secrets,dependencies,containers,vulnerabilities,sbom,compliance,summary}
    mkdir -p "${REPORT_PREFIX}"
}

# Check if required tools are installed
check_tools() {
    log_info "Checking required tools..."
    
    local tools=(
        "gosec:github.com/securego/gosec/v2/cmd/gosec@latest"
        "govulncheck:golang.org/x/vuln/cmd/govulncheck@latest"
        "trivy:curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin"
        "gitleaks:github.com/zricethezav/gitleaks/v8@latest"
        "syft:curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin"
        "grype:curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin"
        "cyclonedx-gomod:github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest"
    )
    
    for tool_info in "${tools[@]}"; do
        IFS=':' read -r tool install_cmd <<< "$tool_info"
        if ! command -v "$tool" &> /dev/null; then
            log_warning "$tool not found. Install with: $install_cmd"
        else
            log_success "$tool is installed"
        fi
    done
}

# Run gosec static analysis
run_gosec() {
    log_info "Running GoSec static analysis..."
    
    local gosec_report="${REPORT_PREFIX}/gosec"
    mkdir -p "$gosec_report"
    
    # Run gosec with different output formats
    gosec -fmt sarif -out "${gosec_report}/gosec.sarif" -severity medium "${PROJECT_ROOT}/..." 2>&1 | tee "${gosec_report}/gosec.log" || true
    gosec -fmt json -out "${gosec_report}/gosec.json" -severity medium "${PROJECT_ROOT}/..." || true
    gosec -fmt text -out "${gosec_report}/gosec.txt" -severity medium "${PROJECT_ROOT}/..." || true
    
    # Parse results
    if [ -f "${gosec_report}/gosec.json" ]; then
        local issue_count=$(jq '.Issues | length' "${gosec_report}/gosec.json" 2>/dev/null || echo "0")
        log_info "GoSec found $issue_count issues"
        
        if [ "$issue_count" -gt "$MAX_GOSEC_ISSUES" ]; then
            log_error "GoSec issues exceed threshold: $issue_count > $MAX_GOSEC_ISSUES"
            return 1
        fi
    fi
    
    log_success "GoSec analysis completed"
}

# Run vulnerability scanning
run_vulnerability_scan() {
    log_info "Running vulnerability scanning..."
    
    local vuln_report="${REPORT_PREFIX}/vulnerabilities"
    mkdir -p "$vuln_report"
    
    # Run govulncheck
    cd "$PROJECT_ROOT"
    govulncheck -json ./... > "${vuln_report}/govulncheck.json" 2>&1 || true
    govulncheck ./... > "${vuln_report}/govulncheck.txt" 2>&1 || true
    
    # Parse vulnerability results
    local critical_count=0
    local high_count=0
    
    if [ -f "${vuln_report}/govulncheck.txt" ]; then
        critical_count=$(grep -c "CRITICAL" "${vuln_report}/govulncheck.txt" || echo "0")
        high_count=$(grep -c "HIGH" "${vuln_report}/govulncheck.txt" || echo "0")
        
        log_info "Vulnerabilities found - Critical: $critical_count, High: $high_count"
        
        if [ "$critical_count" -gt "$MAX_CRITICAL_VULNS" ]; then
            log_error "Critical vulnerabilities exceed threshold: $critical_count > $MAX_CRITICAL_VULNS"
            return 1
        fi
        
        if [ "$high_count" -gt "$MAX_HIGH_VULNS" ]; then
            log_warning "High vulnerabilities exceed threshold: $high_count > $MAX_HIGH_VULNS"
        fi
    fi
    
    log_success "Vulnerability scanning completed"
}

# Run secret detection
run_secret_detection() {
    log_info "Running secret detection..."
    
    local secrets_report="${REPORT_PREFIX}/secrets"
    mkdir -p "$secrets_report"
    
    # Run gitleaks
    if [ -f "${SECURITY_DIR}/configs/gitleaks.toml" ]; then
        gitleaks detect --config "${SECURITY_DIR}/configs/gitleaks.toml" \
            --source "$PROJECT_ROOT" \
            --report-path "${secrets_report}/gitleaks.json" \
            --report-format json \
            --no-git \
            --verbose 2>&1 | tee "${secrets_report}/gitleaks.log" || true
    else
        gitleaks detect --source "$PROJECT_ROOT" \
            --report-path "${secrets_report}/gitleaks.json" \
            --report-format json \
            --no-git \
            --verbose 2>&1 | tee "${secrets_report}/gitleaks.log" || true
    fi
    
    # Parse secret detection results
    if [ -f "${secrets_report}/gitleaks.json" ]; then
        local secret_count=$(jq '. | length' "${secrets_report}/gitleaks.json" 2>/dev/null || echo "0")
        log_info "Secrets found: $secret_count"
        
        if [ "$secret_count" -gt "$MAX_SECRETS" ]; then
            log_error "Secrets detected: $secret_count > $MAX_SECRETS"
            
            # Show secret details (without actual secrets)
            jq -r '.[] | "File: \(.File) Line: \(.StartLine) Rule: \(.RuleID)"' "${secrets_report}/gitleaks.json" 2>/dev/null || true
            
            return 1
        fi
    fi
    
    log_success "Secret detection completed"
}

# Generate SBOM
generate_sbom() {
    log_info "Generating Software Bill of Materials..."
    
    local sbom_report="${REPORT_PREFIX}/sbom"
    mkdir -p "$sbom_report"
    
    cd "$PROJECT_ROOT"
    
    # Generate SBOM with Syft
    if command -v syft &> /dev/null; then
        syft dir:. -o spdx-json > "${sbom_report}/sbom-syft.json" 2>&1 || true
        syft dir:. -o cyclonedx-json > "${sbom_report}/sbom-cyclonedx-syft.json" 2>&1 || true
    fi
    
    # Generate SBOM with CycloneDX
    if command -v cyclonedx-gomod &> /dev/null; then
        cyclonedx-gomod mod -json -output-file "${sbom_report}/sbom-cyclonedx.json" 2>&1 || true
    fi
    
    # Scan SBOM for vulnerabilities
    if command -v grype &> /dev/null && [ -f "${sbom_report}/sbom-syft.json" ]; then
        grype sbom:"${sbom_report}/sbom-syft.json" --output json > "${sbom_report}/sbom-vulnerabilities.json" 2>&1 || true
        grype sbom:"${sbom_report}/sbom-syft.json" --output table > "${sbom_report}/sbom-vulnerabilities.txt" 2>&1 || true
    fi
    
    log_success "SBOM generation completed"
}

# Container security scanning
scan_containers() {
    log_info "Scanning container images..."
    
    local container_report="${REPORT_PREFIX}/containers"
    mkdir -p "$container_report"
    
    # Check if Dockerfile exists
    if [ -f "${PROJECT_ROOT}/Dockerfile" ]; then
        # Build container image for scanning
        docker build -t nephoran-security-scan:latest "$PROJECT_ROOT" 2>&1 | tee "${container_report}/build.log" || true
        
        # Scan with Trivy
        if command -v trivy &> /dev/null; then
            trivy image --severity CRITICAL,HIGH,MEDIUM \
                --format json \
                --output "${container_report}/trivy.json" \
                nephoran-security-scan:latest 2>&1 | tee "${container_report}/trivy.log" || true
            
            trivy image --severity CRITICAL,HIGH,MEDIUM \
                --format sarif \
                --output "${container_report}/trivy.sarif" \
                nephoran-security-scan:latest || true
        fi
        
        # Clean up test image
        docker rmi nephoran-security-scan:latest 2>/dev/null || true
    fi
    
    log_success "Container scanning completed"
}

# Generate security report
generate_report() {
    log_info "Generating security report..."
    
    local summary_report="${REPORT_PREFIX}/summary"
    mkdir -p "$summary_report"
    
    cat > "${summary_report}/security-summary.md" << EOF
# Security Scan Report
**Date:** $(date)
**Project:** Nephoran Intent Operator
**Scan ID:** ${TIMESTAMP}

## Executive Summary

This report provides a comprehensive security analysis of the Nephoran Intent Operator codebase.

## Scan Results

### Static Code Analysis (GoSec)
- Report: [gosec.json](../gosec/gosec.json)
- SARIF: [gosec.sarif](../gosec/gosec.sarif)

### Vulnerability Scanning
- Report: [govulncheck.json](../vulnerabilities/govulncheck.json)
- Text: [govulncheck.txt](../vulnerabilities/govulncheck.txt)

### Secret Detection
- Report: [gitleaks.json](../secrets/gitleaks.json)

### Software Bill of Materials
- SBOM: [sbom-cyclonedx.json](../sbom/sbom-cyclonedx.json)
- Vulnerabilities: [sbom-vulnerabilities.json](../sbom/sbom-vulnerabilities.json)

### Container Security
- Trivy Report: [trivy.json](../containers/trivy.json)
- SARIF: [trivy.sarif](../containers/trivy.sarif)

## Recommendations

1. Review and remediate all critical and high severity vulnerabilities
2. Rotate any detected secrets immediately
3. Update dependencies with known vulnerabilities
4. Implement security gates in CI/CD pipeline
5. Regular security scanning and monitoring

## Compliance Status

- [x] Static code analysis completed
- [x] Vulnerability scanning completed
- [x] Secret detection completed
- [x] SBOM generated
- [x] Container security scanning completed

---
*Generated by Nephoran Security Scanner v1.0*
EOF
    
    # Create JSON summary
    cat > "${summary_report}/summary.json" << EOF
{
  "scan_id": "${TIMESTAMP}",
  "date": "$(date -Iseconds)",
  "project": "nephoran-intent-operator",
  "scans": {
    "gosec": "completed",
    "vulnerabilities": "completed",
    "secrets": "completed",
    "sbom": "completed",
    "containers": "completed"
  },
  "report_location": "${REPORT_PREFIX}"
}
EOF
    
    log_success "Security report generated at: ${summary_report}/security-summary.md"
}

# Archive reports
archive_reports() {
    log_info "Archiving reports..."
    
    cd "$REPORTS_DIR"
    tar -czf "security-scan-${TIMESTAMP}.tar.gz" "${TIMESTAMP}/" 2>/dev/null || true
    
    # Keep only last 10 archives
    ls -t security-scan-*.tar.gz 2>/dev/null | tail -n +11 | xargs rm -f 2>/dev/null || true
    
    log_success "Reports archived"
}

# Main execution
main() {
    log_info "Starting comprehensive security scan..."
    
    # Prepare environment
    prepare_directories
    check_tools
    
    # Run security scans
    local exit_code=0
    
    run_gosec || exit_code=$?
    run_vulnerability_scan || exit_code=$?
    run_secret_detection || exit_code=$?
    generate_sbom || exit_code=$?
    scan_containers || exit_code=$?
    
    # Generate reports
    generate_report
    archive_reports
    
    # Final status
    if [ $exit_code -eq 0 ]; then
        log_success "Security scan completed successfully"
    else
        log_error "Security scan completed with failures"
    fi
    
    return $exit_code
}

# Run main function
main "$@"