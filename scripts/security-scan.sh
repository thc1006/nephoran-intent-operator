#!/bin/bash

# security-scan.sh - Automated dependency security scanning
# Part of Phase 1: Dependency Management Fixes

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=== Nephoran Intent Operator Security Scan ==="
echo "Project Root: $PROJECT_ROOT"
echo "Scan Date: $(date)"
echo

cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed or not in PATH${NC}"
    exit 1
fi

echo "Go Version: $(go version)"
echo

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install govulncheck if available
install_govulncheck() {
    echo "Attempting to install govulncheck..."
    if go install golang.org/x/vuln/cmd/govulncheck@latest 2>/dev/null; then
        echo -e "${GREEN}✓ govulncheck installed successfully${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠ govulncheck installation failed (Go version compatibility issue)${NC}"
        return 1
    fi
}

# Function to run basic security checks
basic_security_check() {
    echo "=== Basic Security Checks ==="
    
    # Check for direct security-related dependencies
    echo "Checking security-critical dependencies..."
    
    SECURITY_DEPS=$(go list -m all | grep -E "(crypto|jwt|oauth|protobuf|grpc|auth|security|tls)" || true)
    if [ -n "$SECURITY_DEPS" ]; then
        echo -e "${GREEN}Security-critical dependencies found:${NC}"
        echo "$SECURITY_DEPS"
    else
        echo -e "${YELLOW}No security-critical dependencies found${NC}"
    fi
    echo
    
    # Check for potentially vulnerable patterns
    echo "Checking for potentially vulnerable code patterns..."
    
    CRYPTO_USAGE=$(find . -name "*.go" -not -path "./vendor/*" -exec grep -l "crypto\|password\|token\|secret" {} \; 2>/dev/null || true)
    if [ -n "$CRYPTO_USAGE" ]; then
        echo -e "${YELLOW}Files with cryptographic operations:${NC}"
        echo "$CRYPTO_USAGE" | head -10
        if [ $(echo "$CRYPTO_USAGE" | wc -l) -gt 10 ]; then
            echo "... and $(( $(echo "$CRYPTO_USAGE" | wc -l) - 10 )) more files"
        fi
    fi
    echo
}

# Function to check for outdated dependencies
check_outdated_deps() {
    echo "=== Dependency Freshness Check ==="
    
    # This is a simplified check - in production, you'd want more sophisticated tooling
    echo "Checking for Go module updates..."
    
    if go list -u -m all >/dev/null 2>&1; then
        OUTDATED=$(go list -u -m all | grep "\[" || true)
        if [ -n "$OUTDATED" ]; then
            echo -e "${YELLOW}Potentially outdated dependencies:${NC}"
            echo "$OUTDATED"
        else
            echo -e "${GREEN}✓ All dependencies appear current${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ Unable to check for updates${NC}"
    fi
    echo
}

# Function to check go.mod for issues
check_go_mod() {
    echo "=== Go Module Integrity Check ==="
    
    if go mod verify; then
        echo -e "${GREEN}✓ Go modules verified successfully${NC}"
    else
        echo -e "${RED}✗ Go module verification failed${NC}"
        return 1
    fi
    
    # Check for indirect dependencies that could be direct
    INDIRECT_COUNT=$(go list -m all | grep "// indirect" | wc -l)
    TOTAL_COUNT=$(go list -m all | tail -n +2 | wc -l)
    
    echo "Dependency summary:"
    echo "  Total dependencies: $TOTAL_COUNT"
    echo "  Indirect dependencies: $INDIRECT_COUNT"
    echo "  Direct dependencies: $((TOTAL_COUNT - INDIRECT_COUNT))"
    
    if [ $INDIRECT_COUNT -gt $((TOTAL_COUNT / 2)) ]; then
        echo -e "${YELLOW}⚠ High number of indirect dependencies - consider running 'go mod tidy'${NC}"
    fi
    echo
}

# Function to generate security report
generate_report() {
    echo "=== Security Scan Summary ==="
    
    REPORT_FILE="$PROJECT_ROOT/security-scan-$(date +%Y%m%d-%H%M%S).log"
    
    {
        echo "Nephoran Intent Operator Security Scan Report"
        echo "Generated: $(date)"
        echo "Go Version: $(go version)"
        echo
        echo "=== Security-Critical Dependencies ==="
        go list -m all | grep -E "(crypto|jwt|oauth|protobuf|grpc|auth|security|tls)" || echo "None found"
        echo
        echo "=== Go Module Status ==="
        go mod verify 2>&1 || echo "Verification failed"
        echo
        echo "=== Dependency Count ==="
        echo "Total: $(go list -m all | tail -n +2 | wc -l)"
        echo "Indirect: $(go list -m all | grep "// indirect" | wc -l)"
        echo
    } > "$REPORT_FILE"
    
    echo "Report saved to: $REPORT_FILE"
}

# Main execution
main() {
    echo "Starting security scan..."
    echo
    
    # Basic checks
    basic_security_check
    check_go_mod
    check_outdated_deps
    
    # Try to run govulncheck if available
    if command_exists govulncheck; then
        echo "=== Running govulncheck ==="
        if govulncheck ./...; then
            echo -e "${GREEN}✓ No vulnerabilities found by govulncheck${NC}"
        else
            echo -e "${RED}✗ Vulnerabilities detected by govulncheck${NC}"
        fi
        echo
    elif install_govulncheck && command_exists govulncheck; then
        echo "=== Running govulncheck ==="
        if govulncheck ./...; then
            echo -e "${GREEN}✓ No vulnerabilities found by govulncheck${NC}"
        else
            echo -e "${RED}✗ Vulnerabilities detected by govulncheck${NC}"
        fi
        echo
    else
        echo "=== govulncheck not available ==="
        echo -e "${YELLOW}⚠ govulncheck could not be installed - using basic checks only${NC}"
        echo "This may be due to Go version compatibility issues."
        echo "For comprehensive vulnerability scanning, ensure Go 1.20+ is available."
        echo
    fi
    
    # Generate report
    generate_report
    
    echo -e "${GREEN}Security scan completed successfully${NC}"
    echo
    echo "Recommendations:"
    echo "1. Review the generated security audit report"
    echo "2. Monitor dependencies for security updates"
    echo "3. Consider integrating this script into CI/CD pipeline"
    echo "4. Update to Go 1.20+ for govulncheck support if needed"
}

# Run main function
main "$@"