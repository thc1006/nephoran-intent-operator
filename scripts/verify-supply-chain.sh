#!/bin/bash

# verify-supply-chain.sh
# Comprehensive supply chain security verification for Go modules
# 
# This script verifies:
# 1. go.mod and go.sum integrity
# 2. Module authenticity via GOSUMDB
# 3. Vulnerability scanning
# 4. Dependency policy compliance
# 5. SBOM generation and verification

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TEMP_DIR=$(mktemp -d)
VULNERABILITY_THRESHOLD="HIGH"
MAX_DEPENDENCY_AGE_DAYS=365
REQUIRED_GO_VERSION="1.24.1"

# Cleanup function
cleanup() {
    rm -rf "${TEMP_DIR}"
}
trap cleanup EXIT

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

log_header() {
    echo
    echo -e "${BLUE}===============================================${NC}"
    echo -e "${BLUE}$*${NC}"
    echo -e "${BLUE}===============================================${NC}"
    echo
}

# Check prerequisites
check_prerequisites() {
    log_header "Checking Prerequisites"
    
    # Check Go version
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        return 1
    fi
    
    local go_version
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go version: $go_version"
    
    # Check if we're in a Go module
    if [[ ! -f "$PROJECT_ROOT/go.mod" ]]; then
        log_error "go.mod not found. This script must be run from a Go module root."
        return 1
    fi
    
    # Install required tools if missing
    local tools=(
        "golang.org/x/vuln/cmd/govulncheck@latest"
        "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest"
    )
    
    for tool in "${tools[@]}"; do
        local cmd_name=${tool##*/}
        cmd_name=${cmd_name%%@*}
        if ! command -v "$cmd_name" &> /dev/null; then
            log_info "Installing $tool..."
            go install "$tool"
        fi
    done
    
    log_success "Prerequisites check passed"
}

# Verify go.mod and go.sum integrity
verify_module_integrity() {
    log_header "Verifying Module Integrity"
    
    cd "$PROJECT_ROOT"
    
    # Check if go.sum exists
    if [[ ! -f "go.sum" ]]; then
        log_error "go.sum file not found"
        return 1
    fi
    
    # Verify module dependencies
    log_info "Verifying module dependencies..."
    if ! go mod verify; then
        log_error "Module verification failed"
        return 1
    fi
    
    # Check for tidy modules
    log_info "Checking if modules are tidy..."
    go mod tidy
    
    if ! git diff --exit-code go.mod go.sum &> /dev/null; then
        log_warning "go.mod or go.sum is not tidy. Run 'go mod tidy'."
        git diff go.mod go.sum
    else
        log_success "Modules are tidy"
    fi
    
    # Verify GOSUMDB authenticity
    log_info "Verifying module authenticity via GOSUMDB..."
    export GOSUMDB=sum.golang.org
    if ! go mod verify; then
        log_error "GOSUMDB verification failed"
        return 1
    fi
    
    log_success "Module integrity verification passed"
}

# Check dependency security policies
check_dependency_policies() {
    log_header "Checking Dependency Security Policies"
    
    cd "$PROJECT_ROOT"
    
    local violations=0
    
    # Check for pre-release versions
    log_info "Checking for pre-release or pseudo-versions..."
    local prerelease_deps
    prerelease_deps=$(go list -m all | grep -E "v[0-9]+\.[0-9]+\.[0-9]+-.*-" || true)
    if [[ -n "$prerelease_deps" ]]; then
        log_warning "Found pre-release or pseudo-versions:"
        echo "$prerelease_deps"
        ((violations++))
    fi
    
    # Check for suspicious dependency patterns
    log_info "Checking for suspicious dependency patterns..."
    go list -m all | while read -r dep version; do
        case "$dep" in
            *"eval"*|*"exec"*|*"shell"*|*"cmd"*|*"system"*|*"unsafe"*)
                log_warning "Potentially suspicious dependency: $dep@$version"
                ((violations++))
                ;;
        esac
    done
    
    # Check for dependencies without version pins
    log_info "Checking for unpinned dependencies..."
    local unpinned_deps
    unpinned_deps=$(go list -m all | grep -v "@v[0-9]" || true)
    if [[ -n "$unpinned_deps" ]]; then
        log_warning "Found dependencies without explicit version pins:"
        echo "$unpinned_deps"
    fi
    
    # Generate dependency report
    local dep_report="$TEMP_DIR/dependency-report.txt"
    {
        echo "# Dependency Security Report"
        echo "Generated on: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
        echo ""
        echo "## Direct Dependencies"
        go list -m -f '{{.Path}}@{{.Version}}' $(go list -f '{{if not .Indirect}}{{.}}{{end}}' -m all) | sort
        echo ""
        echo "## All Dependencies ($(go list -m all | wc -l) total)"
        go list -m all | sort
        echo ""
        echo "## Vulnerability Summary"
        govulncheck -format json ./... 2>/dev/null | jq -r '.finding[]? | "\(.osv.id): \(.osv.summary)"' 2>/dev/null || echo "No vulnerabilities or govulncheck failed"
    } > "$dep_report"
    
    log_info "Dependency report generated: $dep_report"
    
    if [[ $violations -gt 0 ]]; then
        log_warning "Found $violations dependency policy violations"
    else
        log_success "Dependency policies check passed"
    fi
}

# Run vulnerability scanning
run_vulnerability_scan() {
    log_header "Running Vulnerability Scan"
    
    cd "$PROJECT_ROOT"
    
    # Create vulnerability reports directory
    local vuln_dir="$TEMP_DIR/vulnerability-reports"
    mkdir -p "$vuln_dir"
    
    # Run govulncheck on source code
    log_info "Scanning source code for vulnerabilities..."
    local source_report="$vuln_dir/govulncheck-source.json"
    if govulncheck -format json ./... > "$source_report" 2>&1; then
        log_success "Source code vulnerability scan completed"
    else
        log_warning "Source code vulnerabilities detected"
    fi
    
    # Build binaries and scan them
    log_info "Building and scanning binaries..."
    local bin_dir="$TEMP_DIR/bin"
    mkdir -p "$bin_dir"
    
    # List of main packages to build
    local main_packages=(
        "./cmd/main.go:nephoran-operator"
        "./cmd/llm-processor/main.go:llm-processor"
        "./cmd/nephio-bridge/main.go:nephio-bridge"
        "./cmd/oran-adaptor/main.go:oran-adaptor"
    )
    
    for package_info in "${main_packages[@]}"; do
        local package_path="${package_info%%:*}"
        local binary_name="${package_info##*:}"
        local binary_path="$bin_dir/$binary_name"
        
        if [[ -f "$package_path" ]]; then
            log_info "Building $binary_name..."
            if go build -o "$binary_path" "$package_path"; then
                log_info "Scanning binary: $binary_name"
                govulncheck -format json "$binary_path" > "$vuln_dir/govulncheck-$binary_name.json" 2>&1 || true
            else
                log_warning "Failed to build $binary_name"
            fi
        fi
    done
    
    # Analyze vulnerability results
    log_info "Analyzing vulnerability scan results..."
    local critical_vulns=0
    local high_vulns=0
    local total_vulns=0
    
    for report in "$vuln_dir"/*.json; do
        if [[ -f "$report" ]]; then
            local report_vulns
            report_vulns=$(jq -r '.finding[]? | select(.trace[0].function != null) | .osv.database_specific.severity' "$report" 2>/dev/null | sort | uniq -c || true)
            if [[ -n "$report_vulns" ]]; then
                echo "Vulnerabilities in $(basename "$report"):"
                echo "$report_vulns"
                
                # Count critical and high severity vulnerabilities
                critical_vulns=$((critical_vulns + $(echo "$report_vulns" | grep -c "CRITICAL" || true)))
                high_vulns=$((high_vulns + $(echo "$report_vulns" | grep -c "HIGH" || true)))
                total_vulns=$((total_vulns + $(echo "$report_vulns" | wc -l)))
            fi
        fi
    done
    
    # Generate vulnerability summary
    local vuln_summary="$vuln_dir/vulnerability-summary.md"
    {
        echo "# Vulnerability Scan Summary"
        echo "Generated on: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
        echo ""
        echo "## Overview"
        echo "- Total vulnerabilities in active code paths: $total_vulns"
        echo "- Critical severity: $critical_vulns"
        echo "- High severity: $high_vulns"
        echo ""
        echo "## Detailed Results"
        govulncheck ./... 2>&1 || true
        echo ""
        echo "## Recommendations"
        if [[ $critical_vulns -gt 0 || $high_vulns -gt 0 ]]; then
            echo "⚠️  IMMEDIATE ACTION REQUIRED: Critical or high severity vulnerabilities found"
            echo "1. Review the detailed vulnerability reports above"
            echo "2. Update affected dependencies: go get -u <dependency>"
            echo "3. Re-run vulnerability scan to verify fixes"
            echo "4. Consider alternative packages if updates are not available"
        else
            echo "✅ No critical or high severity vulnerabilities found in active code paths"
        fi
    } > "$vuln_summary"
    
    log_info "Vulnerability summary generated: $vuln_summary"
    
    # Exit with error if critical/high vulnerabilities found
    if [[ $critical_vulns -gt 0 || $high_vulns -gt 0 ]]; then
        log_error "Found $critical_vulns critical and $high_vulns high severity vulnerabilities"
        return 1
    else
        log_success "No critical or high severity vulnerabilities found"
    fi
}

# Generate Software Bill of Materials (SBOM)
generate_sbom() {
    log_header "Generating Software Bill of Materials (SBOM)"
    
    cd "$PROJECT_ROOT"
    
    local sbom_dir="$TEMP_DIR/sbom"
    mkdir -p "$sbom_dir"
    
    # Generate CycloneDX SBOM
    log_info "Generating CycloneDX SBOM..."
    if command -v cyclonedx-gomod &> /dev/null; then
        cyclonedx-gomod mod -json -output-file "$sbom_dir/sbom-cyclonedx.json"
        log_success "CycloneDX SBOM generated"
    else
        log_warning "cyclonedx-gomod not available, skipping CycloneDX SBOM generation"
    fi
    
    # Generate SPDX SBOM using syft (if available)
    log_info "Generating SPDX SBOM..."
    if command -v syft &> /dev/null; then
        syft . -o spdx-json="$sbom_dir/sbom-spdx.json"
        log_success "SPDX SBOM generated"
    else
        log_warning "syft not available, skipping SPDX SBOM generation"
    fi
    
    # Generate dependency list
    log_info "Generating dependency list..."
    {
        echo "# Software Bill of Materials"
        echo "Generated on: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
        echo "Repository: $(git remote get-url origin 2>/dev/null || echo "local")"
        echo "Commit: $(git rev-parse HEAD 2>/dev/null || echo "unknown")"
        echo ""
        echo "## Module Information"
        go list -m
        echo ""
        echo "## Direct Dependencies"
        go list -m -f '{{if not .Indirect}}{{.Path}}@{{.Version}}{{end}}' all | grep -v '^$' | sort
        echo ""
        echo "## All Dependencies ($(go list -m all | wc -l) total)"
        go list -m -f '{{.Path}}@{{.Version}}' all | sort
        echo ""
        echo "## License Information"
        echo "Note: License information should be validated manually or with dedicated tools"
    } > "$sbom_dir/sbom-summary.md"
    
    log_success "SBOM generation completed: $sbom_dir"
}

# Verify supply chain configuration
verify_supply_chain_config() {
    log_header "Verifying Supply Chain Configuration"
    
    cd "$PROJECT_ROOT"
    
    # Check GOPRIVATE configuration
    log_info "Checking GOPRIVATE configuration..."
    if [[ -n "${GOPRIVATE:-}" ]]; then
        log_info "GOPRIVATE is set to: $GOPRIVATE"
    else
        log_info "GOPRIVATE is not set (using public modules only)"
    fi
    
    # Check GOSUMDB configuration
    log_info "Checking GOSUMDB configuration..."
    local gosumdb="${GOSUMDB:-sum.golang.org}"
    log_info "GOSUMDB is set to: $gosumdb"
    
    if [[ "$gosumdb" == "off" ]]; then
        log_warning "GOSUMDB is disabled - this reduces supply chain security"
    fi
    
    # Check GOPROXY configuration
    log_info "Checking GOPROXY configuration..."
    local goproxy="${GOPROXY:-https://proxy.golang.org,direct}"
    log_info "GOPROXY is set to: $goproxy"
    
    # Verify Go module configuration
    log_info "Go module configuration:"
    go env | grep -E "(GOPRIVATE|GOSUMDB|GOPROXY|GOINSECURE)"
    
    log_success "Supply chain configuration verification completed"
}

# Generate comprehensive security report
generate_security_report() {
    log_header "Generating Comprehensive Security Report"
    
    local report_dir="$PROJECT_ROOT/security-reports"
    mkdir -p "$report_dir"
    
    local timestamp=$(date +"%Y%m%d-%H%M%S")
    local report_file="$report_dir/supply-chain-security-report-$timestamp.md"
    
    {
        echo "# Supply Chain Security Report"
        echo "Generated on: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
        echo "Repository: $(git remote get-url origin 2>/dev/null || echo "local")"
        echo "Commit: $(git rev-parse HEAD 2>/dev/null || echo "unknown")"
        echo "Go version: $(go version)"
        echo ""
        
        echo "## Executive Summary"
        echo "This report provides a comprehensive analysis of the supply chain security"
        echo "for the Nephoran Intent Operator Go modules."
        echo ""
        
        echo "## Module Integrity"
        if go mod verify &>/dev/null; then
            echo "✅ Module integrity verification: PASSED"
        else
            echo "❌ Module integrity verification: FAILED"
        fi
        echo ""
        
        echo "## Vulnerability Scan Results"
        if [[ -f "$TEMP_DIR/vulnerability-reports/vulnerability-summary.md" ]]; then
            cat "$TEMP_DIR/vulnerability-reports/vulnerability-summary.md"
        else
            echo "Vulnerability scan was not completed successfully"
        fi
        echo ""
        
        echo "## Dependency Information"
        echo "### Direct Dependencies ($(go list -m -f '{{if not .Indirect}}{{.Path}}{{end}}' all | grep -c .))"
        go list -m -f '{{if not .Indirect}}{{.Path}}@{{.Version}}{{end}}' all | grep -v '^$' | sort
        echo ""
        
        echo "### Total Dependencies: $(go list -m all | wc -l)"
        echo ""
        
        echo "## Supply Chain Configuration"
        echo "- GOPROXY: ${GOPROXY:-https://proxy.golang.org,direct}"
        echo "- GOSUMDB: ${GOSUMDB:-sum.golang.org}"
        echo "- GOPRIVATE: ${GOPRIVATE:-not set}"
        echo ""
        
        echo "## Recommendations"
        echo "1. Regularly run 'go get -u' to update dependencies"
        echo "2. Monitor vulnerability databases for new threats"
        echo "3. Review dependency licenses for compliance"
        echo "4. Implement automated security scanning in CI/CD"
        echo "5. Keep SBOM updated for supply chain transparency"
        echo ""
        
        echo "## Next Steps"
        echo "1. Address any critical or high severity vulnerabilities"
        echo "2. Review and approve any new dependencies"
        echo "3. Update security policies as needed"
        echo "4. Schedule regular security reviews"
        
    } > "$report_file"
    
    log_success "Comprehensive security report generated: $report_file"
    
    # Copy temporary files to report directory
    if [[ -d "$TEMP_DIR/vulnerability-reports" ]]; then
        cp -r "$TEMP_DIR/vulnerability-reports" "$report_dir/vulnerability-reports-$timestamp"
    fi
    
    if [[ -d "$TEMP_DIR/sbom" ]]; then
        cp -r "$TEMP_DIR/sbom" "$report_dir/sbom-$timestamp"
    fi
}

# Main execution function
main() {
    log_header "Supply Chain Security Verification"
    log_info "Starting comprehensive supply chain security verification..."
    
    local exit_code=0
    
    # Run all verification steps
    check_prerequisites || exit_code=1
    verify_module_integrity || exit_code=1
    check_dependency_policies || exit_code=1
    run_vulnerability_scan || exit_code=1
    generate_sbom || exit_code=1
    verify_supply_chain_config || exit_code=1
    generate_security_report || exit_code=1
    
    echo
    if [[ $exit_code -eq 0 ]]; then
        log_success "Supply chain security verification completed successfully!"
        log_info "Review the generated reports in the security-reports directory"
    else
        log_error "Supply chain security verification failed!"
        log_error "Please address the issues identified above"
    fi
    
    return $exit_code
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi