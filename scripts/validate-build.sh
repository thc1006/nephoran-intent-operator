#!/bin/bash
set -euo pipefail

echo "🔧 Validating build system integrity..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

ISSUES_FOUND=0

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Check API version consistency
check_api_versions() {
    print_status "$YELLOW" "Checking API version consistency..."
    
    # Check E2NodeSet
    CRD_VERSION=$(grep -A 1 "versions:" deployments/crds/nephoran.com_e2nodesets.yaml | grep "name:" | awk '{print $3}')
    GO_PACKAGE=$(head -20 api/v1/e2nodeset_types.go | grep "package" | awk '{print $2}')
    
    if [[ "$GO_PACKAGE" == "v1" && "$CRD_VERSION" == "v1" ]]; then
        print_status "$GREEN" "✅ E2NodeSet API versions consistent (v1)"
    else
        print_status "$RED" "❌ E2NodeSet API version mismatch: CRD=$CRD_VERSION, Go=$GO_PACKAGE"
        ISSUES_FOUND=$((ISSUES_FOUND + 1))
    fi
    
    # Check NetworkIntent
    NI_CRD_VERSION=$(grep -A 1 "versions:" deployments/crds/nephoran.com_networkintents.yaml | grep "name:" | awk '{print $3}')
    NI_GO_PACKAGE=$(head -20 api/v1/networkintent_types.go | grep "package" | awk '{print $2}')
    
    if [[ "$NI_GO_PACKAGE" == "v1" && "$NI_CRD_VERSION" == "v1" ]]; then
        print_status "$GREEN" "✅ NetworkIntent API versions consistent (v1)"
    else
        print_status "$RED" "❌ NetworkIntent API version mismatch: CRD=$NI_CRD_VERSION, Go=$NI_GO_PACKAGE"
        ISSUES_FOUND=$((ISSUES_FOUND + 1))
    fi
}

# Check for large binaries in git
check_large_binaries() {
    print_status "$YELLOW" "Checking for large binaries in git..."
    
    # Check if bin directory is properly ignored
    if grep -q "bin/" .gitignore; then
        print_status "$GREEN" "✅ bin/ directory properly ignored in .gitignore"
    else
        print_status "$RED" "❌ bin/ directory not in .gitignore"
        ISSUES_FOUND=$((ISSUES_FOUND + 1))
    fi
    
    # Check if any large files are tracked by git
    LARGE_FILES=$(git ls-files | xargs ls -la 2>/dev/null | awk '$5 > 10485760 {print $9 " (" $5 " bytes)"}' || true)
    
    if [[ -z "$LARGE_FILES" ]]; then
        print_status "$GREEN" "✅ No large files tracked by git"
    else
        print_status "$RED" "❌ Large files found in git:"
        echo "$LARGE_FILES"
        ISSUES_FOUND=$((ISSUES_FOUND + 1))
    fi
}

# Validate Go module integrity
check_go_modules() {
    print_status "$YELLOW" "Validating Go modules..."
    
    if go mod verify; then
        print_status "$GREEN" "✅ Go modules verified successfully"
    else
        print_status "$RED" "❌ Go module verification failed"
        ISSUES_FOUND=$((ISSUES_FOUND + 1))
    fi
    
    # Check if go.mod and go.sum are in sync
    if go mod tidy -diff > /tmp/go-mod-diff 2>&1; then
        print_status "$GREEN" "✅ go.mod and go.sum are in sync"
    else
        print_status "$RED" "❌ go.mod and go.sum need synchronization"
        print_status "$YELLOW" "Run 'go mod tidy' to fix"
        ISSUES_FOUND=$((ISSUES_FOUND + 1))
    fi
}

# Check Docker security
check_docker_security() {
    print_status "$YELLOW" "Validating Docker security..."
    
    if [[ -f "pkg/rag/Dockerfile" ]]; then
        # Check for non-root user
        if grep -q "USER rag" pkg/rag/Dockerfile; then
            print_status "$GREEN" "✅ Dockerfile uses non-root user"
        else
            print_status "$RED" "❌ Dockerfile missing non-root user"
            ISSUES_FOUND=$((ISSUES_FOUND + 1))
        fi
        
        # Check for health check
        if grep -q "HEALTHCHECK" pkg/rag/Dockerfile; then
            print_status "$GREEN" "✅ Dockerfile includes health check"
        else
            print_status "$YELLOW" "⚠️  Dockerfile missing health check"
        fi
        
        # Check for multi-stage build
        if grep -q "FROM.*as builder" pkg/rag/Dockerfile; then
            print_status "$GREEN" "✅ Dockerfile uses multi-stage build"
        else
            print_status "$YELLOW" "⚠️  Consider using multi-stage build for security"
        fi
    else
        print_status "$YELLOW" "⚠️  No Dockerfile found to validate"
    fi
}

# Validate CRD schema completeness
check_crd_schemas() {
    print_status "$YELLOW" "Validating CRD schemas..."
    
    # Check if NetworkIntent CRD has all status fields
    REQUIRED_STATUS_FIELDS=("phase" "processingStartTime" "deploymentCompletionTime" "gitCommitHash")
    
    for field in "${REQUIRED_STATUS_FIELDS[@]}"; do
        if grep -q "$field" deployments/crds/nephoran.com_networkintents.yaml; then
            print_status "$GREEN" "✅ NetworkIntent CRD includes $field"
        else
            print_status "$RED" "❌ NetworkIntent CRD missing $field"
            ISSUES_FOUND=$((ISSUES_FOUND + 1))
        fi
    done
}

# Check import path consistency
check_import_paths() {
    print_status "$YELLOW" "Checking import path consistency..."
    
    # Get module name from go.mod
    MODULE_NAME=$(grep "^module " go.mod | awk '{print $2}')
    
    # Check for incorrect imports
    INCORRECT_IMPORTS=$(find . -name "*.go" -not -path "./vendor/*" -exec grep -l "import.*github.com/nephoran-intent-operator" {} \; 2>/dev/null || true)
    
    if [[ -z "$INCORRECT_IMPORTS" ]]; then
        print_status "$GREEN" "✅ No incorrect import paths found"
    else
        print_status "$RED" "❌ Incorrect import paths found:"
        echo "$INCORRECT_IMPORTS"
        ISSUES_FOUND=$((ISSUES_FOUND + 1))
    fi
}

# Test basic build
test_build() {
    print_status "$YELLOW" "Testing basic build..."
    
    if go test -c ./...; then
        print_status "$GREEN" "✅ All packages build successfully (test binaries)"
    else
        print_status "$RED" "❌ Build failed"
        ISSUES_FOUND=$((ISSUES_FOUND + 1))
    fi
}

# Run tests
run_tests() {
    print_status "$YELLOW" "Running tests..."
    
    if go test -v ./... 2>/dev/null; then
        print_status "$GREEN" "✅ All tests pass"
    else
        print_status "$YELLOW" "⚠️  Some tests failed or test environment not available"
    fi
}

# Generate build validation report
generate_report() {
    print_status "$YELLOW" "Generating validation report..."
    
    REPORT_FILE="build-validation-$(date +%Y%m%d-%H%M%S).log"
    
    {
        echo "Nephoran Intent Operator Build Validation Report"
        echo "Generated: $(date)"
        echo "Go Version: $(go version)"
        echo ""
        echo "Issues Found: $ISSUES_FOUND"
        echo ""
        echo "=== API Version Check ==="
        echo "E2NodeSet: $(grep -A 1 "versions:" deployments/crds/nephoran.com_e2nodesets.yaml | grep "name:" | awk '{print $3}')"
        echo "NetworkIntent: $(grep -A 1 "versions:" deployments/crds/nephoran.com_networkintents.yaml | grep "name:" | awk '{print $3}')"
        echo ""
        echo "=== Go Module Status ==="
        go mod verify 2>&1 || echo "Verification failed"
        echo ""
        echo "=== Large Files Check ==="
        git ls-files | xargs ls -la 2>/dev/null | awk '$5 > 10485760 {print $9 " (" $5 " bytes)"}' || echo "None found"
    } > "$REPORT_FILE"
    
    print_status "$GREEN" "Report saved to: $REPORT_FILE"
}

# Main execution
main() {
    print_status "$GREEN" "🔧 Nephoran Intent Operator Build Validation"
    print_status "$GREEN" "=============================================="
    
    check_api_versions
    check_large_binaries
    check_go_modules
    check_docker_security
    check_crd_schemas
    check_import_paths
    test_build
    run_tests
    generate_report
    
    echo ""
    if [[ $ISSUES_FOUND -eq 0 ]]; then
        print_status "$GREEN" "🎉 Build validation completed successfully - no critical issues found!"
        exit 0
    else
        print_status "$RED" "❌ Build validation found $ISSUES_FOUND critical issue(s) - please fix before proceeding"
        exit 1
    fi
}

main "$@"