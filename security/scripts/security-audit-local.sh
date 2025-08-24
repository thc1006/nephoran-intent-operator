#!/bin/bash
set -euo pipefail

# =============================================================================
# Local Security Audit Script for Nephoran Intent Operator
# =============================================================================
# This script runs comprehensive security scans locally to identify and fix
# security issues before committing code.
#
# Usage: ./security/scripts/security-audit-local.sh [OPTIONS]
# Options:
#   --quick         Run quick scan (skip some tools)
#   --fix           Attempt to auto-fix issues where possible
#   --report-only   Generate reports without failing
#   --tools TOOLS   Comma-separated list of tools to run
#
# =============================================================================

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPORTS_DIR="$PROJECT_ROOT/security-reports"
TIMESTAMP=$(date +"%Y%m%d-%H%M%S")

# Tools configuration
TOOLS_AVAILABLE="gosec,trivy,govulncheck,gitleaks,semgrep"
TOOLS_TO_RUN="$TOOLS_AVAILABLE"

# Options
QUICK_SCAN=false
AUTO_FIX=false
REPORT_ONLY=false
VERBOSE=false
FAIL_ON_HIGH=true

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# =============================================================================
# Helper Functions
# =============================================================================

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')] $*${NC}"
}

warn() {
    echo -e "${YELLOW}[WARNING] $*${NC}" >&2
}

error() {
    echo -e "${RED}[ERROR] $*${NC}" >&2
}

success() {
    echo -e "${GREEN}[SUCCESS] $*${NC}"
}

usage() {
    cat << EOF
Usage: $0 [OPTIONS]

OPTIONS:
    --quick         Run quick scan (skip some tools)
    --fix           Attempt to auto-fix issues where possible
    --report-only   Generate reports without failing
    --tools TOOLS   Comma-separated list of tools to run
                   Available: $TOOLS_AVAILABLE
    --verbose       Enable verbose output
    --help          Show this help message

EXAMPLES:
    $0                           # Run all security tools
    $0 --quick                   # Run quick security scan
    $0 --tools gosec,trivy       # Run only gosec and trivy
    $0 --fix                     # Run scan and attempt to fix issues
    $0 --report-only             # Generate reports only (don't fail)

EOF
}

# =============================================================================
# Tool Installation Functions
# =============================================================================

install_gosec() {
    if ! command -v gosec >/dev/null 2>&1; then
        log "Installing gosec..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    fi
}

install_trivy() {
    if ! command -v trivy >/dev/null 2>&1; then
        log "Installing trivy..."
        curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b "$HOME/.local/bin" v0.50.1
        export PATH="$HOME/.local/bin:$PATH"
    fi
}

install_govulncheck() {
    if ! command -v govulncheck >/dev/null 2>&1; then
        log "Installing govulncheck..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
    fi
}

install_gitleaks() {
    if ! command -v gitleaks >/dev/null 2>&1; then
        log "Installing gitleaks..."
        curl -sSfL https://github.com/gitleaks/gitleaks/releases/download/v8.18.1/gitleaks_8.18.1_linux_x64.tar.gz | tar xz -C /tmp
        sudo mv /tmp/gitleaks /usr/local/bin/
    fi
}

install_semgrep() {
    if ! command -v semgrep >/dev/null 2>&1; then
        log "Installing semgrep..."
        python3 -m pip install --user semgrep
        export PATH="$HOME/.local/bin:$PATH"
    fi
}

# =============================================================================
# Security Scanning Functions
# =============================================================================

run_gosec() {
    log "Running GoSec static analysis..."
    mkdir -p "$REPORTS_DIR/gosec"
    
    # Run gosec with configuration
    if gosec -conf="$PROJECT_ROOT/security/configs/gosec.yaml" \
             -fmt=sarif \
             -out="$REPORTS_DIR/gosec/gosec-$TIMESTAMP.sarif" \
             -stdout \
             ./...; then
        success "GoSec scan completed successfully"
        return 0
    else
        error "GoSec scan found security issues"
        
        # Generate human-readable report
        gosec -conf="$PROJECT_ROOT/security/configs/gosec.yaml" \
              -fmt=text \
              -out="$REPORTS_DIR/gosec/gosec-$TIMESTAMP.txt" \
              ./... || true
        
        return 1
    fi
}

run_trivy() {
    log "Running Trivy vulnerability scan..."
    mkdir -p "$REPORTS_DIR/trivy"
    
    local exit_code=0
    
    # Scan filesystem for vulnerabilities
    if trivy fs \
        --config="$PROJECT_ROOT/security/configs/trivy.yaml" \
        --format=sarif \
        --output="$REPORTS_DIR/trivy/trivy-fs-$TIMESTAMP.sarif" \
        "$PROJECT_ROOT"; then
        success "Trivy filesystem scan completed"
    else
        error "Trivy filesystem scan found vulnerabilities"
        exit_code=1
    fi
    
    # Generate SBOM
    trivy fs \
        --format=cyclonedx \
        --output="$REPORTS_DIR/trivy/sbom-$TIMESTAMP.json" \
        "$PROJECT_ROOT" || true
    
    return $exit_code
}

run_govulncheck() {
    log "Running govulncheck for Go vulnerabilities..."
    mkdir -p "$REPORTS_DIR/govulncheck"
    
    if govulncheck -json ./... > "$REPORTS_DIR/govulncheck/govulncheck-$TIMESTAMP.json"; then
        success "govulncheck scan completed successfully"
        return 0
    else
        error "govulncheck found vulnerabilities"
        
        # Generate human-readable report
        govulncheck ./... > "$REPORTS_DIR/govulncheck/govulncheck-$TIMESTAMP.txt" || true
        
        return 1
    fi
}

run_gitleaks() {
    log "Running Gitleaks secret detection..."
    mkdir -p "$REPORTS_DIR/gitleaks"
    
    if gitleaks detect \
        --config="$PROJECT_ROOT/security/configs/gitleaks.toml" \
        --report-format=sarif \
        --report-path="$REPORTS_DIR/gitleaks/gitleaks-$TIMESTAMP.sarif" \
        --source="$PROJECT_ROOT"; then
        success "Gitleaks scan completed - no secrets found"
        return 0
    else
        error "Gitleaks found potential secrets"
        
        # Generate human-readable report
        gitleaks detect \
            --config="$PROJECT_ROOT/security/configs/gitleaks.toml" \
            --report-format=json \
            --report-path="$REPORTS_DIR/gitleaks/gitleaks-$TIMESTAMP.json" \
            --source="$PROJECT_ROOT" || true
        
        return 1
    fi
}

run_semgrep() {
    log "Running Semgrep security analysis..."
    mkdir -p "$REPORTS_DIR/semgrep"
    
    if semgrep \
        --config=auto \
        --sarif \
        --output="$REPORTS_DIR/semgrep/semgrep-$TIMESTAMP.sarif" \
        "$PROJECT_ROOT"; then
        success "Semgrep scan completed successfully"
        return 0
    else
        error "Semgrep found security issues"
        
        # Generate human-readable report
        semgrep \
            --config=auto \
            --text \
            --output="$REPORTS_DIR/semgrep/semgrep-$TIMESTAMP.txt" \
            "$PROJECT_ROOT" || true
        
        return 1
    fi
}

# =============================================================================
# Auto-fix Functions
# =============================================================================

auto_fix_go_mod() {
    log "Attempting to fix Go module issues..."
    
    # Update dependencies
    go get -u ./...
    go mod tidy
    go mod verify
    
    success "Go modules updated and verified"
}

auto_fix_permissions() {
    log "Fixing file permissions..."
    
    # Fix common permission issues
    find "$PROJECT_ROOT" -name "*.sh" -exec chmod +x {} \;
    find "$PROJECT_ROOT" -name "secrets" -type d -exec chmod 700 {} \; 2>/dev/null || true
    find "$PROJECT_ROOT" -name "*.key" -exec chmod 600 {} \; 2>/dev/null || true
    find "$PROJECT_ROOT" -name "*.pem" -exec chmod 600 {} \; 2>/dev/null || true
    
    success "File permissions fixed"
}

# =============================================================================
# Reporting Functions
# =============================================================================

generate_summary_report() {
    log "Generating security summary report..."
    
    local summary_file="$REPORTS_DIR/security-summary-$TIMESTAMP.json"
    
    cat > "$summary_file" << EOF
{
  "timestamp": "$(date -Iseconds)",
  "scan_id": "$TIMESTAMP",
  "project_root": "$PROJECT_ROOT",
  "tools_run": "$(echo "$TOOLS_TO_RUN" | tr ',' ' ')",
  "scan_results": {
EOF

    local first=true
    
    for tool in $(echo "$TOOLS_TO_RUN" | tr ',' ' '); do
        if [ "$first" = true ]; then
            first=false
        else
            echo "," >> "$summary_file"
        fi
        
        echo "    \"$tool\": {" >> "$summary_file"
        
        if [ -f "$REPORTS_DIR/$tool/$tool-$TIMESTAMP.sarif" ]; then
            echo "      \"status\": \"completed\"," >> "$summary_file"
            echo "      \"report_file\": \"$REPORTS_DIR/$tool/$tool-$TIMESTAMP.sarif\"" >> "$summary_file"
        else
            echo "      \"status\": \"failed\"," >> "$summary_file"
            echo "      \"report_file\": null" >> "$summary_file"
        fi
        
        echo "    }" >> "$summary_file"
    done
    
    cat >> "$summary_file" << EOF
  },
  "recommendations": [
    "Review all SARIF reports for detailed findings",
    "Address HIGH and CRITICAL severity issues first",
    "Update dependencies regularly",
    "Run security scans before each commit"
  ]
}
EOF

    success "Security summary report generated: $summary_file"
}

# =============================================================================
# Main Function
# =============================================================================

main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --quick)
                QUICK_SCAN=true
                TOOLS_TO_RUN="gosec,govulncheck,gitleaks"
                shift
                ;;
            --fix)
                AUTO_FIX=true
                shift
                ;;
            --report-only)
                REPORT_ONLY=true
                FAIL_ON_HIGH=false
                shift
                ;;
            --tools)
                TOOLS_TO_RUN="$2"
                shift 2
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --help)
                usage
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
    
    # Initialize
    log "Starting security audit for Nephoran Intent Operator"
    log "Project root: $PROJECT_ROOT"
    log "Reports directory: $REPORTS_DIR"
    log "Tools to run: $TOOLS_TO_RUN"
    
    # Create reports directory
    mkdir -p "$REPORTS_DIR"
    
    # Change to project root
    cd "$PROJECT_ROOT"
    
    # Install required tools
    log "Installing required security tools..."
    for tool in $(echo "$TOOLS_TO_RUN" | tr ',' ' '); do
        case $tool in
            gosec) install_gosec ;;
            trivy) install_trivy ;;
            govulncheck) install_govulncheck ;;
            gitleaks) install_gitleaks ;;
            semgrep) install_semgrep ;;
        esac
    done
    
    # Run auto-fixes if requested
    if [ "$AUTO_FIX" = true ]; then
        log "Running auto-fixes..."
        auto_fix_go_mod
        auto_fix_permissions
    fi
    
    # Run security scans
    local overall_exit_code=0
    
    for tool in $(echo "$TOOLS_TO_RUN" | tr ',' ' '); do
        case $tool in
            gosec)
                if ! run_gosec; then
                    overall_exit_code=1
                fi
                ;;
            trivy)
                if ! run_trivy; then
                    overall_exit_code=1
                fi
                ;;
            govulncheck)
                if ! run_govulncheck; then
                    overall_exit_code=1
                fi
                ;;
            gitleaks)
                if ! run_gitleaks; then
                    overall_exit_code=1
                fi
                ;;
            semgrep)
                if ! run_semgrep; then
                    overall_exit_code=1
                fi
                ;;
        esac
    done
    
    # Generate summary report
    generate_summary_report
    
    # Final status
    if [ $overall_exit_code -eq 0 ]; then
        success "All security scans completed successfully!"
        success "Reports available in: $REPORTS_DIR"
    else
        if [ "$REPORT_ONLY" = true ]; then
            warn "Security issues found (report-only mode)"
            warn "Reports available in: $REPORTS_DIR"
            exit 0
        else
            error "Security issues found - please review reports in: $REPORTS_DIR"
            error "Run with --fix to attempt automatic fixes"
            error "Run with --report-only to generate reports without failing"
            exit $overall_exit_code
        fi
    fi
}

# Run main function
main "$@"