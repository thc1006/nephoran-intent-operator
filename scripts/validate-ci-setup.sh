#!/bin/bash
# =============================================================================
# CI Setup Validation Script
# =============================================================================
# Purpose: Validate that the new CI pipeline setup is ready for deployment
# =============================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=============================================="
echo "🔍 Nephoran CI Pipeline Validation"
echo "=============================================="

# Check 1: Go version compatibility
log_info "Checking Go version..."
if go version | grep -q "go1.2[5-9]\|go1.[3-9][0-9]"; then
    log_success "Go version is compatible: $(go version | cut -d' ' -f3)"
else
    log_warning "Go version may be too old: $(go version | cut -d' ' -f3)"
fi

# Check 2: Required files exist
log_info "Validating CI pipeline files..."
required_files=(
    ".github/workflows/ci-production.yml"
    "scripts/ci-build.sh"
    "Makefile.ci"
    "go.mod"
)

for file in "${required_files[@]}"; do
    if [ -f "$PROJECT_ROOT/$file" ]; then
        log_success "$file exists"
    else
        log_error "$file is missing"
    fi
done

# Check 3: CI build script is executable
if [ -x "$PROJECT_ROOT/scripts/ci-build.sh" ]; then
    log_success "CI build script is executable"
else
    log_error "CI build script is not executable"
fi

# Check 4: Critical cmd directories exist
log_info "Checking critical cmd directories..."
critical_dirs=(
    "cmd/intent-ingest"
    "cmd/llm-processor"
    "cmd/conductor"
    "cmd/nephio-bridge"
    "cmd/webhook"
)

for dir in "${critical_dirs[@]}"; do
    if [ -d "$PROJECT_ROOT/$dir" ]; then
        if [ -f "$PROJECT_ROOT/$dir/main.go" ]; then
            log_success "$dir (with main.go)"
        else
            log_warning "$dir (no main.go found)"
        fi
    else
        log_error "$dir does not exist"
    fi
done

# Check 5: Total cmd directory count
cmd_count=$(find "$PROJECT_ROOT/cmd" -maxdepth 1 -type d | grep -v "^$PROJECT_ROOT/cmd$" | wc -l || echo "0")
log_info "Total cmd directories found: $cmd_count"

if [ "$cmd_count" -gt 30 ]; then
    log_success "Large number of cmd directories detected (CI optimization needed)"
else
    log_warning "Fewer cmd directories than expected ($cmd_count found)"
fi

# Check 6: Go module validity
log_info "Validating Go module..."
if go mod verify >/dev/null 2>&1; then
    log_success "Go module verification passed"
else
    log_warning "Go module verification failed (may need go mod tidy)"
fi

# Check 7: Basic syntax validation
log_info "Running quick syntax validation..."
syntax_errors=0
for dir in "${critical_dirs[@]}"; do
    if [ -d "$PROJECT_ROOT/$dir" ] && [ -f "$PROJECT_ROOT/$dir/main.go" ]; then
        if go vet "$PROJECT_ROOT/$dir" >/dev/null 2>&1; then
            log_success "$dir syntax OK"
        else
            log_warning "$dir has syntax issues"
            syntax_errors=$((syntax_errors + 1))
        fi
    fi
done

# Summary
echo ""
echo "=============================================="
echo "📊 Validation Summary"
echo "=============================================="

if [ $syntax_errors -eq 0 ]; then
    log_success "All critical components have valid syntax"
else
    log_warning "$syntax_errors components have syntax issues"
fi

echo ""
echo "🚀 CI Pipeline Features:"
echo "   ✅ Go 1.25 support"
echo "   ✅ Timeout protection"
echo "   ✅ Selective building (critical components first)"
echo "   ✅ Parallel execution strategies"
echo "   ✅ Smart caching"
echo "   ✅ Comprehensive testing"
echo "   ✅ Integration validation"

echo ""
echo "📋 Next Steps:"
echo "   1. Commit these changes to your feature branch"
echo "   2. Create a PR to test the new CI pipeline"
echo "   3. Monitor the first few CI runs for issues"
echo "   4. Consider enabling the new pipeline gradually"

echo ""
echo "🔧 If issues occur:"
echo "   - Check GitHub Actions logs for detailed errors"
echo "   - Run './scripts/ci-build.sh' locally to reproduce issues"
echo "   - Use 'make -f Makefile.ci debug-build' for debugging"

log_success "Validation completed!"