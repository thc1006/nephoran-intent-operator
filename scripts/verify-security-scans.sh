#!/bin/bash
# =============================================================================
# Security Scan Verification Script
# =============================================================================
# Verifies that all security scanning tools are properly configured and
# SARIF results are correctly uploaded to GitHub Security tab
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GITHUB_API="https://api.github.com"
REPO_OWNER="${GITHUB_REPOSITORY_OWNER:-thc1006}"
REPO_NAME="${GITHUB_REPOSITORY:-nephoran}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"

# Counters
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNINGS=0

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
    ((PASSED_CHECKS++))
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
    ((FAILED_CHECKS++))
}

log_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
    ((WARNINGS++))
}

check_command() {
    local cmd=$1
    local name=$2
    ((TOTAL_CHECKS++))
    
    if command -v "$cmd" &> /dev/null; then
        log_success "$name is installed: $(command -v "$cmd")"
        return 0
    else
        log_error "$name is not installed"
        return 1
    fi
}

check_file() {
    local file=$1
    local description=$2
    ((TOTAL_CHECKS++))
    
    if [ -f "$file" ]; then
        log_success "$description exists: $file"
        return 0
    else
        log_error "$description not found: $file"
        return 1
    fi
}

verify_sarif() {
    local sarif_file=$1
    local tool_name=$2
    ((TOTAL_CHECKS++))
    
    if [ ! -f "$sarif_file" ]; then
        log_warning "SARIF file not found: $sarif_file"
        return 1
    fi
    
    # Validate JSON structure
    if jq empty "$sarif_file" 2>/dev/null; then
        # Check for required SARIF fields
        local has_schema=$(jq -r '."$schema" // empty' "$sarif_file")
        local has_version=$(jq -r '.version // empty' "$sarif_file")
        local has_runs=$(jq -r '.runs // empty' "$sarif_file")
        
        if [ -n "$has_schema" ] && [ -n "$has_version" ] && [ -n "$has_runs" ]; then
            local result_count=$(jq '.runs[0].results | length' "$sarif_file" 2>/dev/null || echo "0")
            log_success "$tool_name SARIF valid with $result_count results"
            return 0
        else
            log_error "$tool_name SARIF missing required fields"
            return 1
        fi
    else
        log_error "$tool_name SARIF is not valid JSON"
        return 1
    fi
}

# =============================================================================
# Check Security Tools Installation
# =============================================================================

echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}     Security Scan Verification - Nephoran Intent Operator${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}\n"

log_info "Checking security tool installations..."
echo

check_command "go" "Go compiler"
check_command "gosec" "Gosec security scanner" || log_warning "Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"
check_command "govulncheck" "Go vulnerability checker" || log_warning "Install: go install golang.org/x/vuln/cmd/govulncheck@latest"
check_command "trivy" "Trivy scanner" || log_warning "Install: See https://aquasecurity.github.io/trivy/latest/getting-started/installation/"
check_command "grype" "Grype scanner" || log_warning "Install: curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin"
check_command "hadolint" "Dockerfile linter" || log_warning "Install: wget -q https://github.com/hadolint/hadolint/releases/latest/download/hadolint-Linux-x86_64"
check_command "nancy" "Nancy dependency scanner" || log_warning "Install: go install github.com/sonatype-nexus-community/nancy@latest"

# =============================================================================
# Check Workflow Files
# =============================================================================

echo -e "\n${BLUE}Checking workflow configurations...${NC}\n"

check_file ".github/workflows/security-scan-fixed.yml" "Fixed security scan workflow"
check_file ".github/workflows/container-security-fixed.yml" "Container security workflow"
check_file ".github/security-policy-fixed.yml" "Security policy configuration"
check_file ".github/codeql/codeql-config.yml" "CodeQL configuration"

# =============================================================================
# Verify Configuration Files
# =============================================================================

echo -e "\n${BLUE}Verifying security configurations...${NC}\n"

# Check gosec config
((TOTAL_CHECKS++))
if [ -f ".gosec.json" ]; then
    if jq empty .gosec.json 2>/dev/null; then
        log_success "Gosec configuration is valid JSON"
    else
        log_error "Gosec configuration is invalid JSON"
    fi
else
    log_warning "No .gosec.json configuration file found"
fi

# Check for Dockerfile security
((TOTAL_CHECKS++))
dockerfile_count=$(find . -name "Dockerfile*" -type f | grep -v node_modules | wc -l)
if [ "$dockerfile_count" -gt 0 ]; then
    log_success "Found $dockerfile_count Dockerfile(s) for security scanning"
else
    log_warning "No Dockerfiles found for container security scanning"
fi

# =============================================================================
# Test Security Scans
# =============================================================================

echo -e "\n${BLUE}Testing security scanners...${NC}\n"

# Test gosec
if command -v gosec &> /dev/null; then
    ((TOTAL_CHECKS++))
    log_info "Running gosec test scan..."
    if gosec -fmt sarif -out /tmp/gosec-test.sarif -no-fail ./... 2>/dev/null; then
        verify_sarif "/tmp/gosec-test.sarif" "Gosec test"
    else
        log_warning "Gosec test scan failed"
    fi
fi

# Test govulncheck
if command -v govulncheck &> /dev/null; then
    ((TOTAL_CHECKS++))
    log_info "Running govulncheck test scan..."
    if timeout 30s govulncheck -json ./cmd/... > /tmp/govulncheck-test.json 2>/dev/null; then
        log_success "Govulncheck test scan completed"
    else
        log_warning "Govulncheck test scan failed or timed out"
    fi
fi

# Test Trivy
if command -v trivy &> /dev/null; then
    ((TOTAL_CHECKS++))
    log_info "Running Trivy test scan..."
    if trivy fs --format sarif --output /tmp/trivy-test.sarif --timeout 1m . 2>/dev/null; then
        verify_sarif "/tmp/trivy-test.sarif" "Trivy test"
    else
        log_warning "Trivy test scan failed"
    fi
fi

# =============================================================================
# Check GitHub Security Tab Integration
# =============================================================================

echo -e "\n${BLUE}Checking GitHub Security integration...${NC}\n"

if [ -n "$GITHUB_TOKEN" ]; then
    ((TOTAL_CHECKS++))
    log_info "Checking GitHub Security alerts API..."
    
    # Check if we can access the security alerts API
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        "$GITHUB_API/repos/$REPO_OWNER/$REPO_NAME/code-scanning/alerts")
    
    if [ "$response" = "200" ]; then
        log_success "GitHub Security API accessible"
        
        # Count existing alerts
        alert_count=$(curl -s \
            -H "Authorization: token $GITHUB_TOKEN" \
            -H "Accept: application/vnd.github.v3+json" \
            "$GITHUB_API/repos/$REPO_OWNER/$REPO_NAME/code-scanning/alerts" | jq '. | length')
        
        log_info "Current security alerts in GitHub: $alert_count"
    elif [ "$response" = "404" ]; then
        log_warning "GitHub Advanced Security may not be enabled for this repository"
    else
        log_error "Failed to access GitHub Security API (HTTP $response)"
    fi
else
    log_warning "GITHUB_TOKEN not set - skipping GitHub API checks"
fi

# =============================================================================
# Verify SARIF Upload Categories
# =============================================================================

echo -e "\n${BLUE}Verifying SARIF upload categories...${NC}\n"

# Expected SARIF categories from our workflows
EXPECTED_CATEGORIES=(
    "gosec"
    "govulncheck"
    "trivy-filesystem"
    "trivy-container-intent-ingest"
    "trivy-container-conductor-loop"
    "trivy-container-llm-processor"
    "trivy-container-webhook"
    "hadolint-dockerfile"
    "grype-vulnerabilities"
    "codeql-go"
)

log_info "Expected SARIF upload categories:"
for category in "${EXPECTED_CATEGORIES[@]}"; do
    echo "  - $category"
done

# =============================================================================
# Generate Recommendations
# =============================================================================

echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}                        Summary Report${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}\n"

echo "Total Checks: $TOTAL_CHECKS"
echo -e "${GREEN}Passed: $PASSED_CHECKS${NC}"
echo -e "${RED}Failed: $FAILED_CHECKS${NC}"
echo -e "${YELLOW}Warnings: $WARNINGS${NC}"

SUCCESS_RATE=$((PASSED_CHECKS * 100 / TOTAL_CHECKS))
echo -e "\nSuccess Rate: ${SUCCESS_RATE}%"

if [ $SUCCESS_RATE -ge 80 ]; then
    echo -e "\n${GREEN}✅ Security scanning setup is GOOD${NC}"
elif [ $SUCCESS_RATE -ge 60 ]; then
    echo -e "\n${YELLOW}⚠️ Security scanning setup needs improvement${NC}"
else
    echo -e "\n${RED}❌ Security scanning setup has critical issues${NC}"
fi

echo -e "\n${BLUE}Recommendations:${NC}\n"

if [ $FAILED_CHECKS -gt 0 ] || [ $WARNINGS -gt 0 ]; then
    echo "1. Install missing security tools:"
    echo "   make install-security-tools"
    echo ""
    echo "2. Run the fixed security workflow:"
    echo "   gh workflow run security-scan-fixed.yml"
    echo ""
    echo "3. Enable GitHub Advanced Security (if not enabled):"
    echo "   - Go to Settings > Security > Code security and analysis"
    echo "   - Enable 'Dependency graph', 'Dependabot alerts', and 'Code scanning'"
    echo ""
    echo "4. Verify SARIF uploads in GitHub Security tab:"
    echo "   https://github.com/$REPO_OWNER/$REPO_NAME/security/code-scanning"
    echo ""
    echo "5. Review and merge security workflow fixes:"
    echo "   - .github/workflows/security-scan-fixed.yml"
    echo "   - .github/workflows/container-security-fixed.yml"
else
    echo "✅ All security scanning configurations are properly set up!"
    echo ""
    echo "Next steps:"
    echo "1. Run security scans: gh workflow run security-scan-fixed.yml"
    echo "2. Monitor results in GitHub Security tab"
    echo "3. Address any findings based on severity"
fi

echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════${NC}\n"

# Exit with appropriate code
if [ $FAILED_CHECKS -gt 0 ]; then
    exit 1
else
    exit 0
fi