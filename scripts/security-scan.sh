#!/bin/bash

# Security Scanning Script for Nephoran Intent Operator
# Performs comprehensive security scanning including SAST, container scanning, and dependency checks

set -euo pipefail

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
SCAN_REPORT_DIR="${SCAN_REPORT_DIR:-./security-scan-reports}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="${SCAN_REPORT_DIR}/security_scan_${TIMESTAMP}.json"
FAIL_ON_HIGH="${FAIL_ON_HIGH:-true}"
FAIL_ON_CRITICAL="${FAIL_ON_CRITICAL:-true}"

# Create report directory
mkdir -p "${SCAN_REPORT_DIR}"

# Initialize counters
TOTAL_CRITICAL=0
TOTAL_HIGH=0
TOTAL_MEDIUM=0
TOTAL_LOW=0

# Function to log messages
log() {
    echo -e "${1}${2}${NC}"
}

# Function to check if command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        log "${YELLOW}" "Warning: $1 is not installed. Skipping $2 scan."
        return 1
    fi
    return 0
}

# Function to run Trivy container scanning
run_trivy_scan() {
    log "${BLUE}" "Running Trivy Container Security Scan..."
    
    if ! check_command "trivy" "container vulnerability"; then
        return 0
    fi
    
    local trivy_report="${SCAN_REPORT_DIR}/trivy_${TIMESTAMP}.json"
    
    # Scan Dockerfile
    if [ -f "Dockerfile" ]; then
        log "${BLUE}" "  Scanning Dockerfile..."
        trivy config --severity CRITICAL,HIGH,MEDIUM,LOW \
            --format json \
            --output "${trivy_report}" \
            Dockerfile || true
        
        # Parse results
        if [ -f "${trivy_report}" ]; then
            local critical=$(jq '[.Results[]?.Misconfigurations[]? | select(.Severity == "CRITICAL")] | length' "${trivy_report}")
            local high=$(jq '[.Results[]?.Misconfigurations[]? | select(.Severity == "HIGH")] | length' "${trivy_report}")
            local medium=$(jq '[.Results[]?.Misconfigurations[]? | select(.Severity == "MEDIUM")] | length' "${trivy_report}")
            local low=$(jq '[.Results[]?.Misconfigurations[]? | select(.Severity == "LOW")] | length' "${trivy_report}")
            
            TOTAL_CRITICAL=$((TOTAL_CRITICAL + critical))
            TOTAL_HIGH=$((TOTAL_HIGH + high))
            TOTAL_MEDIUM=$((TOTAL_MEDIUM + medium))
            TOTAL_LOW=$((TOTAL_LOW + low))
            
            log "${BLUE}" "  Dockerfile scan results:"
            [ ${critical} -gt 0 ] && log "${RED}" "    CRITICAL: ${critical}"
            [ ${high} -gt 0 ] && log "${YELLOW}" "    HIGH: ${high}"
            [ ${medium} -gt 0 ] && log "${BLUE}" "    MEDIUM: ${medium}"
            [ ${low} -gt 0 ] && log "${GREEN}" "    LOW: ${low}"
        fi
    fi
    
    # Scan built images
    local images=$(find . -name "*.yaml" -o -name "*.yml" | xargs grep -h "image:" 2>/dev/null | \
        sed 's/.*image: *//;s/"//g;s/#.*//' | sort -u | grep -v "^$" || true)
    
    for image in ${images}; do
        log "${BLUE}" "  Scanning image: ${image}..."
        local image_report="${SCAN_REPORT_DIR}/trivy_image_$(echo ${image} | tr '/:' '_')_${TIMESTAMP}.json"
        
        trivy image --severity CRITICAL,HIGH,MEDIUM,LOW \
            --format json \
            --output "${image_report}" \
            "${image}" 2>/dev/null || continue
        
        if [ -f "${image_report}" ]; then
            local critical=$(jq '[.Results[]?.Vulnerabilities[]? | select(.Severity == "CRITICAL")] | length' "${image_report}")
            local high=$(jq '[.Results[]?.Vulnerabilities[]? | select(.Severity == "HIGH")] | length' "${image_report}")
            
            TOTAL_CRITICAL=$((TOTAL_CRITICAL + critical))
            TOTAL_HIGH=$((TOTAL_HIGH + high))
            
            [ ${critical} -gt 0 ] && log "${RED}" "    Found ${critical} CRITICAL vulnerabilities"
            [ ${high} -gt 0 ] && log "${YELLOW}" "    Found ${high} HIGH vulnerabilities"
        fi
    done
}

# Function to run Grype vulnerability scanning
run_grype_scan() {
    log "${BLUE}" "Running Grype Vulnerability Scan..."
    
    if ! check_command "grype" "dependency vulnerability"; then
        return 0
    fi
    
    local grype_report="${SCAN_REPORT_DIR}/grype_${TIMESTAMP}.json"
    
    grype dir:. \
        --output json \
        --file "${grype_report}" \
        --fail-on high || true
    
    if [ -f "${grype_report}" ]; then
        local critical=$(jq '[.matches[]? | select(.vulnerability.severity == "Critical")] | length' "${grype_report}")
        local high=$(jq '[.matches[]? | select(.vulnerability.severity == "High")] | length' "${grype_report}")
        
        TOTAL_CRITICAL=$((TOTAL_CRITICAL + critical))
        TOTAL_HIGH=$((TOTAL_HIGH + high))
        
        log "${BLUE}" "  Grype scan results:"
        [ ${critical} -gt 0 ] && log "${RED}" "    CRITICAL: ${critical}"
        [ ${high} -gt 0 ] && log "${YELLOW}" "    HIGH: ${high}"
    fi
}

# Function to run gosec static analysis
run_gosec_scan() {
    log "${BLUE}" "Running GoSec Static Security Analysis..."
    
    if ! check_command "gosec" "Go code security"; then
        return 0
    fi
    
    local gosec_report="${SCAN_REPORT_DIR}/gosec_${TIMESTAMP}.json"
    
    gosec -fmt json -out "${gosec_report}" -severity high ./... 2>/dev/null || true
    
    if [ -f "${gosec_report}" ]; then
        local high=$(jq '.Issues | map(select(.severity == "HIGH")) | length' "${gosec_report}")
        local medium=$(jq '.Issues | map(select(.severity == "MEDIUM")) | length' "${gosec_report}")
        
        TOTAL_HIGH=$((TOTAL_HIGH + high))
        TOTAL_MEDIUM=$((TOTAL_MEDIUM + medium))
        
        log "${BLUE}" "  GoSec scan results:"
        [ ${high} -gt 0 ] && log "${YELLOW}" "    HIGH: ${high}"
        [ ${medium} -gt 0 ] && log "${BLUE}" "    MEDIUM: ${medium}"
    fi
}

# Function to run Nancy dependency check
run_nancy_scan() {
    log "${BLUE}" "Running Nancy Dependency Check..."
    
    if ! check_command "nancy" "Go dependency"; then
        return 0
    fi
    
    if [ -f "go.sum" ]; then
        local nancy_report="${SCAN_REPORT_DIR}/nancy_${TIMESTAMP}.txt"
        
        go list -json -m all | nancy sleuth -o json > "${nancy_report}" 2>/dev/null || true
        
        if [ -f "${nancy_report}" ] && [ -s "${nancy_report}" ]; then
            local vulnerabilities=$(jq '.vulnerabilities | length' "${nancy_report}" 2>/dev/null || echo "0")
            
            if [ "${vulnerabilities}" -gt 0 ]; then
                TOTAL_HIGH=$((TOTAL_HIGH + vulnerabilities))
                log "${YELLOW}" "  Found ${vulnerabilities} vulnerable dependencies"
            else
                log "${GREEN}" "  No vulnerable dependencies found"
            fi
        fi
    fi
}

# Function to run Snyk security testing
run_snyk_scan() {
    log "${BLUE}" "Running Snyk Security Testing..."
    
    if ! check_command "snyk" "comprehensive security"; then
        return 0
    fi
    
    # Check if authenticated
    if ! snyk auth 2>/dev/null | grep -q "Authenticated"; then
        log "${YELLOW}" "  Snyk not authenticated. Skipping scan."
        return 0
    fi
    
    local snyk_report="${SCAN_REPORT_DIR}/snyk_${TIMESTAMP}.json"
    
    # Test for vulnerabilities
    snyk test --json --severity-threshold=low > "${snyk_report}" 2>/dev/null || true
    
    if [ -f "${snyk_report}" ]; then
        local critical=$(jq '[.vulnerabilities[]? | select(.severity == "critical")] | length' "${snyk_report}")
        local high=$(jq '[.vulnerabilities[]? | select(.severity == "high")] | length' "${snyk_report}")
        local medium=$(jq '[.vulnerabilities[]? | select(.severity == "medium")] | length' "${snyk_report}")
        
        TOTAL_CRITICAL=$((TOTAL_CRITICAL + critical))
        TOTAL_HIGH=$((TOTAL_HIGH + high))
        TOTAL_MEDIUM=$((TOTAL_MEDIUM + medium))
        
        log "${BLUE}" "  Snyk scan results:"
        [ ${critical} -gt 0 ] && log "${RED}" "    CRITICAL: ${critical}"
        [ ${high} -gt 0 ] && log "${YELLOW}" "    HIGH: ${high}"
        [ ${medium} -gt 0 ] && log "${BLUE}" "    MEDIUM: ${medium}"
    fi
    
    # Container scanning
    if [ -f "Dockerfile" ]; then
        snyk container test --json --severity-threshold=low \
            --file=Dockerfile > "${snyk_report}.container" 2>/dev/null || true
    fi
    
    # IaC scanning
    snyk iac test --json > "${snyk_report}.iac" 2>/dev/null || true
}

# Function to run git-secrets scan
run_git_secrets_scan() {
    log "${BLUE}" "Running Git-Secrets Scan..."
    
    if ! check_command "git-secrets" "secret detection"; then
        # Try gitleaks as alternative
        if check_command "gitleaks" "secret detection"; then
            run_gitleaks_scan
        fi
        return 0
    fi
    
    # Initialize git-secrets if not already done
    git secrets --install 2>/dev/null || true
    git secrets --register-aws 2>/dev/null || true
    
    # Scan for secrets
    local secrets_found=0
    git secrets --scan 2>/dev/null || secrets_found=$?
    
    if [ ${secrets_found} -ne 0 ]; then
        TOTAL_CRITICAL=$((TOTAL_CRITICAL + 1))
        log "${RED}" "  Potential secrets detected in code!"
    else
        log "${GREEN}" "  No secrets detected"
    fi
}

# Function to run gitleaks scan
run_gitleaks_scan() {
    log "${BLUE}" "Running Gitleaks Secret Detection..."
    
    if ! check_command "gitleaks" "secret detection"; then
        return 0
    fi
    
    local gitleaks_report="${SCAN_REPORT_DIR}/gitleaks_${TIMESTAMP}.json"
    
    gitleaks detect --report-format json --report-path "${gitleaks_report}" || true
    
    if [ -f "${gitleaks_report}" ]; then
        local leaks=$(jq '. | length' "${gitleaks_report}")
        
        if [ ${leaks} -gt 0 ]; then
            TOTAL_CRITICAL=$((TOTAL_CRITICAL + leaks))
            log "${RED}" "  Found ${leaks} potential secrets!"
        else
            log "${GREEN}" "  No secrets detected"
        fi
    fi
}

# Function to run Kubernetes manifest scanning
run_kubesec_scan() {
    log "${BLUE}" "Running Kubesec Kubernetes Security Scan..."
    
    if ! check_command "kubesec" "Kubernetes manifest"; then
        return 0
    fi
    
    local manifests=$(find . -name "*.yaml" -o -name "*.yml" | \
        xargs grep -l "kind:" 2>/dev/null | head -10)
    
    for manifest in ${manifests}; do
        local filename=$(basename "${manifest}")
        log "${BLUE}" "  Scanning ${filename}..."
        
        local kubesec_report="${SCAN_REPORT_DIR}/kubesec_${filename}_${TIMESTAMP}.json"
        kubesec scan "${manifest}" > "${kubesec_report}" 2>/dev/null || continue
        
        if [ -f "${kubesec_report}" ]; then
            local score=$(jq '.[0].score' "${kubesec_report}" 2>/dev/null || echo "0")
            local critical=$(jq '[.[0].scoring.critical[]?] | length' "${kubesec_report}" 2>/dev/null || echo "0")
            
            if [ "${score}" -lt 0 ]; then
                TOTAL_CRITICAL=$((TOTAL_CRITICAL + 1))
                log "${RED}" "    Score: ${score} (CRITICAL issues found)"
            elif [ "${score}" -lt 5 ]; then
                TOTAL_HIGH=$((TOTAL_HIGH + 1))
                log "${YELLOW}" "    Score: ${score} (Security improvements needed)"
            else
                log "${GREEN}" "    Score: ${score} (Good security posture)"
            fi
        fi
    done
}

# Function to run OWASP dependency check
run_owasp_dependency_check() {
    log "${BLUE}" "Running OWASP Dependency Check..."
    
    if ! check_command "dependency-check" "OWASP dependency"; then
        return 0
    fi
    
    dependency-check \
        --scan . \
        --format JSON \
        --out "${SCAN_REPORT_DIR}" \
        --project "nephoran-intent-operator" \
        --suppression dependency-check-suppression.xml 2>/dev/null || true
    
    local owasp_report="${SCAN_REPORT_DIR}/dependency-check-report.json"
    
    if [ -f "${owasp_report}" ]; then
        local critical=$(jq '[.dependencies[]?.vulnerabilities[]? | select(.severity == "CRITICAL")] | length' "${owasp_report}")
        local high=$(jq '[.dependencies[]?.vulnerabilities[]? | select(.severity == "HIGH")] | length' "${owasp_report}")
        
        TOTAL_CRITICAL=$((TOTAL_CRITICAL + critical))
        TOTAL_HIGH=$((TOTAL_HIGH + high))
        
        log "${BLUE}" "  OWASP scan results:"
        [ ${critical} -gt 0 ] && log "${RED}" "    CRITICAL: ${critical}"
        [ ${high} -gt 0 ] && log "${YELLOW}" "    HIGH: ${high}"
    fi
}

# Function to generate comprehensive report
generate_report() {
    log "${BLUE}" "Generating comprehensive security report..."
    
    cat > "${REPORT_FILE}" <<EOF
{
  "timestamp": "${TIMESTAMP}",
  "scan_summary": {
    "total_critical": ${TOTAL_CRITICAL},
    "total_high": ${TOTAL_HIGH},
    "total_medium": ${TOTAL_MEDIUM},
    "total_low": ${TOTAL_LOW},
    "status": "$([ ${TOTAL_CRITICAL} -eq 0 ] && [ ${TOTAL_HIGH} -eq 0 ] && echo "PASS" || echo "FAIL")"
  },
  "scans_performed": [
    "trivy_container",
    "grype_vulnerability",
    "gosec_static_analysis",
    "nancy_dependency",
    "snyk_comprehensive",
    "git_secrets",
    "kubesec_kubernetes",
    "owasp_dependency"
  ],
  "report_location": "${SCAN_REPORT_DIR}",
  "recommendations": {
    "critical": "Address all CRITICAL vulnerabilities immediately",
    "high": "Plan remediation for HIGH vulnerabilities within 7 days",
    "medium": "Include MEDIUM vulnerabilities in next release cycle",
    "low": "Review LOW vulnerabilities for false positives"
  }
}
EOF
    
    log "${GREEN}" "  Report saved to: ${REPORT_FILE}"
}

# Function to upload results to security dashboard
upload_results() {
    if [ -n "${SECURITY_DASHBOARD_URL:-}" ]; then
        log "${BLUE}" "Uploading results to security dashboard..."
        
        curl -X POST \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer ${SECURITY_DASHBOARD_TOKEN:-}" \
            -d @"${REPORT_FILE}" \
            "${SECURITY_DASHBOARD_URL}/api/v1/scan-results" || \
            log "${YELLOW}" "  Failed to upload results"
    fi
}

# Main execution
main() {
    log "${GREEN}" "================================================"
    log "${GREEN}" "Nephoran Intent Operator Security Scanning"
    log "${GREEN}" "================================================"
    log "${BLUE}" "Timestamp: ${TIMESTAMP}"
    echo ""
    
    # Run all security scans
    run_trivy_scan
    echo ""
    
    run_grype_scan
    echo ""
    
    run_gosec_scan
    echo ""
    
    run_nancy_scan
    echo ""
    
    run_snyk_scan
    echo ""
    
    run_git_secrets_scan
    echo ""
    
    run_kubesec_scan
    echo ""
    
    run_owasp_dependency_check
    echo ""
    
    # Generate report
    generate_report
    echo ""
    
    # Display summary
    log "${GREEN}" "================================================"
    log "${GREEN}" "Security Scan Summary:"
    log "${GREEN}" "================================================"
    
    if [ ${TOTAL_CRITICAL} -gt 0 ]; then
        log "${RED}" "CRITICAL vulnerabilities: ${TOTAL_CRITICAL}"
    fi
    
    if [ ${TOTAL_HIGH} -gt 0 ]; then
        log "${YELLOW}" "HIGH vulnerabilities: ${TOTAL_HIGH}"
    fi
    
    if [ ${TOTAL_MEDIUM} -gt 0 ]; then
        log "${BLUE}" "MEDIUM vulnerabilities: ${TOTAL_MEDIUM}"
    fi
    
    if [ ${TOTAL_LOW} -gt 0 ]; then
        log "${GREEN}" "LOW vulnerabilities: ${TOTAL_LOW}"
    fi
    
    # Upload results
    upload_results
    
    # Determine exit code
    local exit_code=0
    
    if [ "${FAIL_ON_CRITICAL}" == "true" ] && [ ${TOTAL_CRITICAL} -gt 0 ]; then
        log "${RED}" "Failing due to CRITICAL vulnerabilities"
        exit_code=1
    fi
    
    if [ "${FAIL_ON_HIGH}" == "true" ] && [ ${TOTAL_HIGH} -gt 0 ]; then
        log "${YELLOW}" "Failing due to HIGH vulnerabilities"
        exit_code=1
    fi
    
    if [ ${exit_code} -eq 0 ]; then
        log "${GREEN}" "✓ Security scan completed successfully!"
    else
        log "${RED}" "✗ Security scan failed due to vulnerabilities"
    fi
    
    log "${GREEN}" "================================================"
    
    exit ${exit_code}
}

# Run main function
main "$@"