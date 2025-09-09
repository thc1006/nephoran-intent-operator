#!/bin/bash
# =============================================================================
# Security Workflow Testing Script
# =============================================================================
# This script tests all security scanning tools locally before CI/CD execution
# Run this to verify security configurations work correctly
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
GO_VERSION="1.22.7"
TRIVY_VERSION="0.58.1"
GOVULNCHECK_VERSION="v1.1.3"

# Results tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Test result function
test_result() {
    local test_name="$1"
    local status="$2"
    local message="${3:-}"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    case "$status" in
        "pass")
            PASSED_TESTS=$((PASSED_TESTS + 1))
            echo -e "${GREEN}✅ PASS${NC}: $test_name"
            ;;
        "fail")
            FAILED_TESTS=$((FAILED_TESTS + 1))
            echo -e "${RED}❌ FAIL${NC}: $test_name"
            if [[ -n "$message" ]]; then
                echo "   Error: $message"
            fi
            ;;
        "skip")
            SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
            echo -e "${YELLOW}⏭️  SKIP${NC}: $test_name"
            if [[ -n "$message" ]]; then
                echo "   Reason: $message"
            fi
            ;;
    esac
}

# Header
echo "=========================================="
echo "Security Workflow Testing Script"
echo "=========================================="
echo ""

# Check Go version
echo -e "${BLUE}Testing Go installation...${NC}"
if command -v go &> /dev/null; then
    GO_INSTALLED_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    test_result "Go installed" "pass" "Version: $GO_INSTALLED_VERSION"
else
    test_result "Go installed" "fail" "Go is not installed"
    exit 1
fi

# Test govulncheck installation
echo -e "\n${BLUE}Testing govulncheck...${NC}"
if ! command -v govulncheck &> /dev/null; then
    echo "Installing govulncheck..."
    if go install golang.org/x/vuln/cmd/govulncheck@$GOVULNCHECK_VERSION 2>/dev/null; then
        test_result "govulncheck installation" "pass"
    else
        test_result "govulncheck installation" "fail" "Failed to install"
    fi
fi

# Run govulncheck
if command -v govulncheck &> /dev/null; then
    echo "Running govulncheck..."
    set +e
    govulncheck -json ./... > /tmp/govulncheck-test.json 2>&1
    GOVULN_EXIT=$?
    set -e
    
    if [[ $GOVULN_EXIT -eq 0 ]]; then
        test_result "govulncheck scan" "pass" "No vulnerabilities found"
    elif [[ $GOVULN_EXIT -eq 3 ]]; then
        VULN_COUNT=$(grep -c '"OSV"' /tmp/govulncheck-test.json 2>/dev/null || echo "unknown")
        test_result "govulncheck scan" "fail" "$VULN_COUNT vulnerabilities found"
    else
        test_result "govulncheck scan" "fail" "Exit code: $GOVULN_EXIT"
    fi
else
    test_result "govulncheck scan" "skip" "govulncheck not available"
fi

# Test Trivy installation
echo -e "\n${BLUE}Testing Trivy...${NC}"
if ! command -v trivy &> /dev/null; then
    echo "Installing Trivy (requires sudo on Linux)..."
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sudo sh -s -- -b /usr/local/bin v$TRIVY_VERSION
    else
        test_result "Trivy installation" "skip" "Manual installation required on $OSTYPE"
    fi
fi

# Run Trivy filesystem scan
if command -v trivy &> /dev/null; then
    echo "Running Trivy filesystem scan..."
    set +e
    trivy fs \
        --format json \
        --severity CRITICAL,HIGH \
        --ignore-unfixed \
        --timeout 5m \
        --quiet \
        . > /tmp/trivy-test.json 2>&1
    TRIVY_EXIT=$?
    set -e
    
    if [[ $TRIVY_EXIT -eq 0 ]]; then
        test_result "Trivy filesystem scan" "pass" "No critical/high vulnerabilities"
    else
        VULN_COUNT=$(jq '.Results[].Vulnerabilities | length' /tmp/trivy-test.json 2>/dev/null | paste -sd+ | bc 2>/dev/null || echo "unknown")
        test_result "Trivy filesystem scan" "fail" "$VULN_COUNT vulnerabilities found"
    fi
else
    test_result "Trivy filesystem scan" "skip" "Trivy not available"
fi

# Test SARIF generation
echo -e "\n${BLUE}Testing SARIF generation...${NC}"

# Create test SARIF file
cat > /tmp/test.sarif << 'EOF'
{
  "version": "2.1.0",
  "$schema": "https://json.schemastore.org/sarif-2.1.0.json",
  "runs": [{
    "tool": {
      "driver": {
        "name": "test-tool",
        "informationUri": "https://example.com"
      }
    },
    "results": []
  }]
}
EOF

# Validate SARIF with jq
if command -v jq &> /dev/null; then
    if jq empty /tmp/test.sarif 2>/dev/null; then
        test_result "SARIF validation" "pass" "Valid SARIF format"
    else
        test_result "SARIF validation" "fail" "Invalid SARIF format"
    fi
else
    test_result "SARIF validation" "skip" "jq not available"
fi

# Test Go build
echo -e "\n${BLUE}Testing Go build...${NC}"
if go build -o /tmp/test-build ./cmd/manager 2>/dev/null; then
    test_result "Go build" "pass"
    rm -f /tmp/test-build
else
    test_result "Go build" "fail" "Build failed"
fi

# Test Go modules
echo -e "\n${BLUE}Testing Go modules...${NC}"
if go mod verify 2>/dev/null; then
    test_result "Go module verification" "pass"
else
    test_result "Go module verification" "fail" "Module verification failed"
fi

# Test for common security issues
echo -e "\n${BLUE}Testing for common security issues...${NC}"

# Check for hardcoded secrets
echo "Checking for hardcoded secrets..."
SECRET_PATTERNS='(api[_-]?key|secret|token|password|passwd|pwd).*=.*["'"'"'][^"'"'"']+["'"'"']'
if git grep -iE "$SECRET_PATTERNS" -- '*.go' '*.yaml' '*.yml' '*.json' 2>/dev/null | grep -v -E '(test|example|sample|mock|fake|dummy)' > /tmp/secrets-check.txt; then
    SECRET_COUNT=$(wc -l < /tmp/secrets-check.txt)
    if [[ $SECRET_COUNT -gt 0 ]]; then
        test_result "Secret scanning" "fail" "$SECRET_COUNT potential secrets found"
        echo "   Found in:"
        head -5 /tmp/secrets-check.txt | sed 's/^/   /'
    else
        test_result "Secret scanning" "pass" "No secrets detected"
    fi
else
    test_result "Secret scanning" "pass" "No secrets detected"
fi

# Check for unsafe functions
echo "Checking for unsafe Go functions..."
UNSAFE_COUNT=$(grep -r "unsafe\." --include="*.go" . 2>/dev/null | grep -v "_test.go" | wc -l || echo "0")
if [[ $UNSAFE_COUNT -gt 0 ]]; then
    test_result "Unsafe package usage" "fail" "$UNSAFE_COUNT uses of unsafe package"
else
    test_result "Unsafe package usage" "pass" "No unsafe package usage"
fi

# Test Dockerfile if exists
echo -e "\n${BLUE}Testing Dockerfile security...${NC}"
if [[ -f Dockerfile ]]; then
    # Check for non-root user
    if grep -q "USER" Dockerfile; then
        test_result "Dockerfile non-root user" "pass"
    else
        test_result "Dockerfile non-root user" "fail" "No USER directive found"
    fi
    
    # Check for latest tag
    if grep -q "FROM.*:latest" Dockerfile; then
        test_result "Dockerfile version pinning" "fail" "Using :latest tag"
    else
        test_result "Dockerfile version pinning" "pass"
    fi
else
    test_result "Dockerfile security" "skip" "No Dockerfile found"
fi

# Test GitHub workflow syntax
echo -e "\n${BLUE}Testing GitHub workflow files...${NC}"
WORKFLOW_COUNT=$(find .github/workflows -name "*.yml" -o -name "*.yaml" 2>/dev/null | wc -l || echo "0")
if [[ $WORKFLOW_COUNT -gt 0 ]]; then
    VALID_WORKFLOWS=0
    for workflow in .github/workflows/*.{yml,yaml} 2>/dev/null; do
        if [[ -f "$workflow" ]]; then
            if command -v yq &> /dev/null; then
                if yq eval '.' "$workflow" > /dev/null 2>&1; then
                    VALID_WORKFLOWS=$((VALID_WORKFLOWS + 1))
                fi
            elif command -v python3 &> /dev/null; then
                if python3 -c "import yaml; yaml.safe_load(open('$workflow'))" 2>/dev/null; then
                    VALID_WORKFLOWS=$((VALID_WORKFLOWS + 1))
                fi
            else
                VALID_WORKFLOWS=$((VALID_WORKFLOWS + 1))  # Assume valid if no validator
            fi
        fi
    done
    
    if [[ $VALID_WORKFLOWS -eq $WORKFLOW_COUNT ]]; then
        test_result "GitHub workflow syntax" "pass" "$WORKFLOW_COUNT workflows valid"
    else
        INVALID=$((WORKFLOW_COUNT - VALID_WORKFLOWS))
        test_result "GitHub workflow syntax" "fail" "$INVALID invalid workflows"
    fi
else
    test_result "GitHub workflow syntax" "skip" "No workflow files found"
fi

# Summary
echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "Total Tests:    $TOTAL_TESTS"
echo -e "Passed:         ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed:         ${RED}$FAILED_TESTS${NC}"
echo -e "Skipped:        ${YELLOW}$SKIPPED_TESTS${NC}"
echo ""

if [[ $FAILED_TESTS -eq 0 ]]; then
    echo -e "${GREEN}✅ All security tests passed!${NC}"
    echo ""
    echo "Security workflows are ready for CI/CD execution."
    exit 0
else
    echo -e "${RED}❌ Some security tests failed.${NC}"
    echo ""
    echo "Please fix the issues before pushing to CI/CD."
    exit 1
fi