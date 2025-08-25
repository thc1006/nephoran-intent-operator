#!/bin/bash
# Quality Framework Validation Script
# Validates that all quality tools and configurations are properly installed

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
WARNINGS=0

# Logging functions
log() {
    echo -e "${BLUE}[INFO] $1${NC}"
}

success() {
    echo -e "${GREEN}[PASS] $1${NC}"
    ((PASSED++))
}

error() {
    echo -e "${RED}[FAIL] $1${NC}" >&2
    ((FAILED++))
}

warning() {
    echo -e "${YELLOW}[WARN] $1${NC}" >&2
    ((WARNINGS++))
}

# Test functions
test_go_installation() {
    log "Testing Go installation..."
    
    if command -v go >/dev/null 2>&1; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        success "Go is installed: version $GO_VERSION"
        
        # Check if Go version meets minimum requirements
        if [[ "$GO_VERSION" < "1.24" ]]; then
            warning "Go version $GO_VERSION is below recommended 1.24+"
        fi
    else
        error "Go is not installed or not in PATH"
    fi
}

test_quality_tools() {
    log "Testing quality tools installation..."
    
    # golangci-lint
    if command -v golangci-lint >/dev/null 2>&1; then
        success "golangci-lint is available"
    else
        warning "golangci-lint not found - will be auto-installed"
    fi
    
    # govulncheck
    if command -v govulncheck >/dev/null 2>&1; then
        success "govulncheck is available"
    else
        warning "govulncheck not found - will be auto-installed"
    fi
    
    # gosec
    if command -v gosec >/dev/null 2>&1; then
        success "gosec is available"
    else
        warning "gosec not found - will be auto-installed"
    fi
}

test_configuration_files() {
    log "Testing configuration files..."
    
    # .golangci.yml
    if [[ -f ".golangci.yml" ]]; then
        success ".golangci.yml configuration file exists"
        
        # Validate YAML syntax
        if command -v yamllint >/dev/null 2>&1; then
            if yamllint .golangci.yml >/dev/null 2>&1; then
                success ".golangci.yml has valid YAML syntax"
            else
                warning ".golangci.yml has YAML syntax issues"
            fi
        fi
    else
        error ".golangci.yml configuration file is missing"
    fi
    
    # GitHub Actions workflow
    if [[ -f ".github/workflows/quality-gate.yml" ]]; then
        success "Quality gate GitHub Actions workflow exists"
    else
        error "Quality gate GitHub Actions workflow is missing"
    fi
    
    # Quality scripts
    local scripts=(
        "scripts/quality-gate.sh"
        "scripts/quality-metrics.go"
        "scripts/performance-regression-test.go"
        "scripts/technical-debt-monitor.go"
    )
    
    for script in "${scripts[@]}"; do
        if [[ -f "$script" ]]; then
            success "$script exists"
            
            # Check if script is executable
            if [[ -x "$script" ]] || [[ "$script" =~ \.go$ ]]; then
                success "$script is executable/runnable"
            else
                warning "$script exists but is not executable"
            fi
        else
            error "$script is missing"
        fi
    done
}

test_makefile_targets() {
    log "Testing Makefile quality targets..."
    
    local targets=(
        "quality-gate"
        "quality-gate-ci"
        "quality-metrics"
        "performance-regression"
        "technical-debt"
        "quality-dashboard"
        "quality-summary"
    )
    
    for target in "${targets[@]}"; do
        if make -n "$target" >/dev/null 2>&1; then
            success "Makefile target '$target' is available"
        else
            error "Makefile target '$target' is missing or invalid"
        fi
    done
}

test_project_structure() {
    log "Testing project structure..."
    
    # Check for Go modules
    if [[ -f "go.mod" ]]; then
        success "go.mod exists"
    else
        error "go.mod is missing"
    fi
    
    if [[ -f "go.sum" ]]; then
        success "go.sum exists"
    else
        warning "go.sum is missing (run 'go mod tidy')"
    fi
    
    # Check for main source directories
    local dirs=(
        "pkg"
        "cmd"
        "api"
        "tests"
    )
    
    for dir in "${dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            success "Directory '$dir' exists"
        else
            warning "Directory '$dir' is missing"
        fi
    done
}

test_dependencies() {
    log "Testing Go dependencies..."
    
    # Check if go mod tidy would make changes
    if go mod tidy -diff >/dev/null 2>&1; then
        success "Go modules are tidy"
    else
        warning "Go modules need tidying (run 'go mod tidy')"
    fi
    
    # Check for common testing dependencies
    if go list -m github.com/onsi/ginkgo/v2 >/dev/null 2>&1; then
        success "Ginkgo testing framework is available"
    else
        warning "Ginkgo testing framework not found"
    fi
    
    if go list -m github.com/onsi/gomega >/dev/null 2>&1; then
        success "Gomega assertion library is available"
    else
        warning "Gomega assertion library not found"
    fi
}

test_quality_thresholds() {
    log "Testing quality threshold configuration..."
    
    # Check Makefile for quality thresholds
    if grep -q "COVERAGE_THRESHOLD.*90" Makefile; then
        success "Coverage threshold is set to 90%"
    else
        warning "Coverage threshold not found or not set to 90%"
    fi
    
    if grep -q "QUALITY_THRESHOLD.*8.0" Makefile; then
        success "Quality threshold is set to 8.0"
    else
        warning "Quality threshold not found or not set to 8.0"
    fi
    
    if grep -q "DEBT_THRESHOLD.*0.3" Makefile; then
        success "Technical debt threshold is set to 30%"
    else
        warning "Technical debt threshold not found or not set to 30%"
    fi
}

run_basic_quality_checks() {
    log "Running basic quality checks..."
    
    # Test go fmt
    if gofmt_output=$(gofmt -l . 2>/dev/null) && [[ -z "$gofmt_output" ]]; then
        success "Code is properly formatted (gofmt)"
    else
        warning "Code formatting issues found (run 'gofmt -w .')"
    fi
    
    # Test go vet
    if go vet ./... >/dev/null 2>&1; then
        success "No issues found by go vet"
    else
        warning "go vet found issues"
    fi
    
    # Test basic compilation
    if go test -c ./... >/dev/null 2>&1; then
        success "Code compiles successfully"
    else
        error "Code compilation failed"
    fi
}

test_documentation() {
    log "Testing documentation..."
    
    if [[ -f "docs/QUALITY_FRAMEWORK.md" ]]; then
        success "Quality framework documentation exists"
    else
        error "Quality framework documentation is missing"
    fi
    
    if [[ -f "README.md" ]]; then
        success "Project README exists"
    else
        warning "Project README is missing"
    fi
}

generate_validation_report() {
    local timestamp=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
    
    cat > quality-framework-validation-report.md << EOF
# Quality Framework Validation Report

**Generated**: $timestamp
**Project**: Nephoran Intent Operator
**Validator**: Quality Framework Validation Script

## Summary

- **Passed**: $PASSED checks
- **Failed**: $FAILED checks  
- **Warnings**: $WARNINGS checks

## Overall Status

$(if [ $FAILED -eq 0 ]; then echo "✅ **PASSED** - Quality framework is properly configured"; else echo "❌ **FAILED** - Quality framework has configuration issues"; fi)

## Validation Results

### Tool Installation
- Go installation and version check
- Quality tools availability (golangci-lint, govulncheck, gosec)

### Configuration Files
- .golangci.yml linting configuration
- GitHub Actions quality workflow
- Quality analysis scripts

### Project Structure
- Go modules and dependencies
- Source code organization
- Testing framework setup

### Quality Standards
- Threshold configuration (coverage, quality score, debt ratio)
- Makefile targets for quality operations
- Documentation completeness

## Recommendations

$(if [ $FAILED -gt 0 ]; then echo "1. **Address Failed Checks**: Fix the $FAILED failed validation items above"; fi)
$(if [ $WARNINGS -gt 0 ]; then echo "2. **Review Warnings**: Consider addressing the $WARNINGS warning items for optimal setup"; fi)
$(if [ $FAILED -eq 0 ] && [ $WARNINGS -eq 0 ]; then echo "✨ **Quality framework is fully configured and ready for use!**"; fi)

## Next Steps

1. Run comprehensive quality analysis:
   \`\`\`bash
   make quality-gate
   \`\`\`

2. View quality dashboard:
   \`\`\`bash
   make quality-dashboard
   \`\`\`

3. Set performance baseline:
   \`\`\`bash
   make quality-baseline
   \`\`\`

4. Configure CI/CD integration by ensuring the quality-gate.yml workflow is active.

---

For more information, see [Quality Framework Documentation](docs/QUALITY_FRAMEWORK.md).
EOF

    echo "📄 Validation report generated: quality-framework-validation-report.md"
}

# Main execution
main() {
    echo "============================================================================="
    echo "🔍 Nephoran Intent Operator - Quality Framework Validation"
    echo "============================================================================="
    echo ""
    
    # Run all validation tests
    test_go_installation
    test_quality_tools
    test_configuration_files
    test_makefile_targets
    test_project_structure
    test_dependencies
    test_quality_thresholds
    run_basic_quality_checks
    test_documentation
    
    echo ""
    echo "============================================================================="
    echo "📊 VALIDATION SUMMARY"
    echo "============================================================================="
    echo ""
    echo "Passed: $PASSED"
    echo "Failed: $FAILED"
    echo "Warnings: $WARNINGS"
    echo ""
    
    # Generate report
    generate_validation_report
    
    # Final status
    if [ $FAILED -eq 0 ]; then
        echo "✅ Quality framework validation PASSED"
        echo ""
        echo "🚀 Ready to use quality framework:"
        echo "   make quality-gate      # Run comprehensive quality analysis"
        echo "   make quality-dashboard # Generate quality dashboard"
        echo "   make quality-summary   # Show quick quality status"
        echo ""
        exit 0
    else
        echo "❌ Quality framework validation FAILED"
        echo ""
        echo "🔧 Fix the $FAILED failed items and re-run validation"
        echo "📄 See quality-framework-validation-report.md for details"
        echo ""
        exit 1
    fi
}

# Run main function
main "$@"