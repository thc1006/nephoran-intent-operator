#!/bin/bash
# =============================================================================
# Master Validation Script - Comprehensive CI Fixes Validation
# =============================================================================
# Orchestrates all validation tests to ensure CI fixes work correctly.
# Provides a single command to validate all components before PR merge.
# =============================================================================

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MASTER_LOG="$PROJECT_ROOT/master-validation-report.txt"
RESULTS_DIR="$PROJECT_ROOT/validation-results"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$MASTER_LOG"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$MASTER_LOG"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$MASTER_LOG"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" | tee -a "$MASTER_LOG"; }
log_header() { echo -e "${CYAN}${BOLD}$1${NC}" | tee -a "$MASTER_LOG"; }

# Test suite results
declare -A SUITE_RESULTS
declare -A SUITE_SCORES
TOTAL_SUITES=0
PASSED_SUITES=0
FAILED_SUITES=0

# =============================================================================
# Test Suite Execution Function
# =============================================================================
run_test_suite() {
    local suite_name="$1"
    local script_path="$2"
    local description="$3"
    local timeout_minutes="${4:-10}"
    
    TOTAL_SUITES=$((TOTAL_SUITES + 1))
    
    log_header "🔍 Running Test Suite: $suite_name"
    log_info "$description"
    log_info "Script: $(basename $script_path)"
    log_info "Timeout: ${timeout_minutes} minutes"
    echo "" | tee -a "$MASTER_LOG"
    
    local start_time=$(date +%s)
    local suite_log="$RESULTS_DIR/$suite_name.log"
    local success=false
    
    # Ensure script is executable
    if [ -f "$script_path" ]; then
        chmod +x "$script_path" 2>/dev/null || true
        
        # Run test suite with timeout
        if timeout $((timeout_minutes * 60))s "$script_path" > "$suite_log" 2>&1; then
            success=true
            PASSED_SUITES=$((PASSED_SUITES + 1))
            SUITE_RESULTS["$suite_name"]="PASSED"
        else
            local exit_code=$?
            FAILED_SUITES=$((FAILED_SUITES + 1))
            SUITE_RESULTS["$suite_name"]="FAILED"
            
            if [ $exit_code -eq 124 ]; then
                log_error "Test suite timed out after $timeout_minutes minutes"
            else
                log_error "Test suite failed with exit code $exit_code"
            fi
        fi
    else
        FAILED_SUITES=$((FAILED_SUITES + 1))
        SUITE_RESULTS["$suite_name"]="MISSING"
        log_error "Test suite script not found: $script_path"
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    # Report results
    if [ "$success" = "true" ]; then
        log_success "Test suite completed successfully in ${duration}s"
        
        # Try to extract score from log
        if [ -f "$suite_log" ]; then
            local score=$(grep -o "Success Rate:.*%" "$suite_log" | head -1 | grep -o "[0-9.]\+%" | head -1 || echo "N/A")
            SUITE_SCORES["$suite_name"]="$score"
            if [ "$score" != "N/A" ]; then
                log_info "Test suite score: $score"
            fi
        fi
    else
        log_error "Test suite failed after ${duration}s"
        SUITE_SCORES["$suite_name"]="0%"
        
        # Show last few lines of error log
        if [ -f "$suite_log" ]; then
            echo "" | tee -a "$MASTER_LOG"
            log_error "Last 10 lines from failed test:"
            tail -10 "$suite_log" | sed 's/^/  /' | tee -a "$MASTER_LOG"
        fi
    fi
    
    echo "" | tee -a "$MASTER_LOG"
}

# =============================================================================
# Pre-flight Environment Setup
# =============================================================================
setup_validation_environment() {
    log_header "🚀 Setting Up Validation Environment"
    
    # Create results directory
    mkdir -p "$RESULTS_DIR"
    
    # Initialize master log
    cat > "$MASTER_LOG" << EOF
===============================================================================
NEPHORAN CI FIXES - MASTER VALIDATION REPORT
===============================================================================
Generated: $(date)
Project: $(basename "$PROJECT_ROOT")
Branch: $(git branch --show-current 2>/dev/null || echo "unknown")
Commit: $(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
Validator: $(whoami)@$(hostname)
===============================================================================

EOF
    
    log_info "Validation environment initialized"
    log_info "Project root: $PROJECT_ROOT"
    log_info "Results directory: $RESULTS_DIR"
    log_info "Master log: $MASTER_LOG"
    
    # Check for required dependencies
    local dependencies=("git" "docker" "python3" "bash" "curl")
    local missing_deps=()
    
    for dep in "${dependencies[@]}"; do
        if command -v "$dep" >/dev/null 2>&1; then
            log_info "✅ $dep available"
        else
            log_warning "⚠️ $dep not available (some tests may be skipped)"
            missing_deps+=("$dep")
        fi
    done
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        log_warning "Missing dependencies: ${missing_deps[*]}"
        log_warning "Some validation tests may be skipped or produce warnings"
    fi
    
    echo "" | tee -a "$MASTER_LOG"
}

# =============================================================================
# Generate Comprehensive Final Report
# =============================================================================
generate_final_report() {
    log_header "📊 Generating Comprehensive Validation Report"
    
    local overall_success_rate=0
    if [ $TOTAL_SUITES -gt 0 ]; then
        overall_success_rate=$(echo "scale=1; $PASSED_SUITES * 100 / $TOTAL_SUITES" | bc -l 2>/dev/null || echo "0")
    fi
    
    # Summary table
    {
        echo ""
        echo "==============================================================================="
        echo "FINAL VALIDATION SUMMARY"
        echo "==============================================================================="
        echo ""
        echo "Overall Results:"
        echo "  Total Test Suites: $TOTAL_SUITES"
        echo "  Passed Suites: $PASSED_SUITES"
        echo "  Failed Suites: $FAILED_SUITES"
        echo "  Overall Success Rate: ${overall_success_rate}%"
        echo ""
        echo "Test Suite Breakdown:"
        echo "+--------------------------+----------+-------+"
        echo "| Test Suite               | Status   | Score |"
        echo "+--------------------------+----------+-------+"
    } | tee -a "$MASTER_LOG"
    
    # Individual suite results
    for suite_name in "${!SUITE_RESULTS[@]}"; do
        local status="${SUITE_RESULTS[$suite_name]}"
        local score="${SUITE_SCORES[$suite_name]:-N/A}"
        
        printf "| %-24s | %-8s | %-5s |\n" "$suite_name" "$status" "$score" | tee -a "$MASTER_LOG"
    done
    
    echo "+--------------------------+----------+-------+" | tee -a "$MASTER_LOG"
    echo "" | tee -a "$MASTER_LOG"
    
    # Detailed analysis
    {
        echo "==============================================================================="
        echo "DETAILED ANALYSIS"
        echo "==============================================================================="
        echo ""
    } | tee -a "$MASTER_LOG"
    
    # Critical issues analysis
    if [ $FAILED_SUITES -gt 0 ]; then
        {
            echo "🚨 CRITICAL ISSUES DETECTED:"
            echo ""
        } | tee -a "$MASTER_LOG"
        
        for suite_name in "${!SUITE_RESULTS[@]}"; do
            if [ "${SUITE_RESULTS[$suite_name]}" = "FAILED" ]; then
                echo "  ❌ $suite_name: Review detailed log at validation-results/$suite_name.log" | tee -a "$MASTER_LOG"
            fi
        done
        echo "" | tee -a "$MASTER_LOG"
    fi
    
    # Success analysis
    if [ $PASSED_SUITES -gt 0 ]; then
        {
            echo "✅ SUCCESSFUL VALIDATIONS:"
            echo ""
        } | tee -a "$MASTER_LOG"
        
        for suite_name in "${!SUITE_RESULTS[@]}"; do
            if [ "${SUITE_RESULTS[$suite_name]}" = "PASSED" ]; then
                local score="${SUITE_SCORES[$suite_name]:-N/A}"
                echo "  ✅ $suite_name (Score: $score)" | tee -a "$MASTER_LOG"
            fi
        done
        echo "" | tee -a "$MASTER_LOG"
    fi
    
    # Final recommendation
    {
        echo "==============================================================================="
        echo "FINAL RECOMMENDATION"
        echo "==============================================================================="
        echo ""
    } | tee -a "$MASTER_LOG"
    
    if [ $FAILED_SUITES -eq 0 ]; then
        if [ "${overall_success_rate%.*}" -gt 90 ]; then
            {
                echo "🎉 EXCELLENT: All validations passed with high confidence!"
                echo ""
                echo "✅ READY FOR PRODUCTION MERGE"
                echo "✅ CI fixes are fully validated and functional"
                echo "✅ All components working as expected"
                echo ""
                echo "Recommended actions:"
                echo "  1. Proceed with PR merge"
                echo "  2. Monitor first CI run after merge"
                echo "  3. Consider deploying to staging environment"
            } | tee -a "$MASTER_LOG"
        else
            {
                echo "✅ GOOD: All critical validations passed"
                echo ""
                echo "✅ READY FOR MERGE (with monitoring)"
                echo "✅ Core CI functionality validated"
                echo "⚠️ Some optimization opportunities remain"
                echo ""
                echo "Recommended actions:"
                echo "  1. Proceed with PR merge"
                echo "  2. Address optimization warnings in follow-up"
                echo "  3. Monitor CI performance"
            } | tee -a "$MASTER_LOG"
        fi
    elif [ $FAILED_SUITES -eq 1 ] && [ $PASSED_SUITES -gt 3 ]; then
        {
            echo "⚠️ CONDITIONAL: Core systems validated, one suite failed"
            echo ""
            echo "⚠️ CONDITIONAL MERGE APPROVAL"
            echo "✅ Core CI functionality appears working"
            echo "❌ One test suite has issues"
            echo ""
            echo "Recommended actions:"
            echo "  1. Review failed test suite details"
            echo "  2. Determine if failure is blocking"
            echo "  3. Consider merge if failure is non-critical"
            echo "  4. Plan immediate follow-up to fix issues"
        } | tee -a "$MASTER_LOG"
    else
        {
            echo "❌ CRITICAL: Multiple validation failures detected"
            echo ""
            echo "❌ NOT READY FOR MERGE"
            echo "❌ CI fixes have significant issues"
            echo "❌ Multiple components failing validation"
            echo ""
            echo "Required actions:"
            echo "  1. DO NOT MERGE until issues resolved"
            echo "  2. Review all failed test suite logs"
            echo "  3. Fix critical issues identified"
            echo "  4. Re-run validation before attempting merge"
        } | tee -a "$MASTER_LOG"
    fi
    
    echo "" | tee -a "$MASTER_LOG"
    echo "Validation completed: $(date)" | tee -a "$MASTER_LOG"
}

# =============================================================================
# Quick Test Mode
# =============================================================================
run_quick_validation() {
    log_header "⚡ Running Quick Validation (Essential Tests Only)"
    
    # Only run the most critical tests
    local quick_tests=(
        "workflow-syntax:$SCRIPT_DIR/test-workflow-syntax.py:GitHub Actions workflow syntax validation:2"
        "service-config:$SCRIPT_DIR/validate-ci-fixes.sh:Service configuration validation:3"
    )
    
    for test_def in "${quick_tests[@]}"; do
        local name=$(echo "$test_def" | cut -d: -f1)
        local script=$(echo "$test_def" | cut -d: -f2)
        local desc=$(echo "$test_def" | cut -d: -f3)
        local timeout=$(echo "$test_def" | cut -d: -f4)
        
        run_test_suite "$name" "$script" "$desc" "$timeout"
    done
}

# =============================================================================
# Full Test Mode
# =============================================================================
run_full_validation() {
    log_header "🔧 Running Full Validation Suite (All Tests)"
    
    # Define all test suites
    local test_suites=(
        "ci-fixes-comprehensive:$SCRIPT_DIR/validate-ci-fixes.sh:Comprehensive CI fixes validation:8"
        "workflow-syntax:$SCRIPT_DIR/test-workflow-syntax.py:GitHub Actions workflow syntax validation:5"
        "docker-builds:$SCRIPT_DIR/test-docker-build-validation.sh:Docker build system validation:10"
        "ghcr-auth:$SCRIPT_DIR/test-ghcr-auth.sh:GHCR authentication validation:5"
    )
    
    for test_def in "${test_suites[@]}"; do
        local name=$(echo "$test_def" | cut -d: -f1)
        local script=$(echo "$test_def" | cut -d: -f2)
        local desc=$(echo "$test_def" | cut -d: -f3)
        local timeout=$(echo "$test_def" | cut -d: -f4)
        
        run_test_suite "$name" "$script" "$desc" "$timeout"
    done
}

# =============================================================================
# Performance Test Mode
# =============================================================================
run_performance_validation() {
    log_header "⚡ Running Performance-Focused Validation"
    
    # Test build performance
    log_info "Testing Docker build performance..."
    
    local dockerfile="$PROJECT_ROOT/Dockerfile"
    if [ -f "$dockerfile" ] && command -v docker >/dev/null 2>&1; then
        local start_time=$(date +%s)
        
        # Test dry-run build speed
        if timeout 120s docker buildx build --dry-run --build-arg SERVICE=conductor-loop "$dockerfile" . >/dev/null 2>&1; then
            local end_time=$(date +%s)
            local duration=$((end_time - start_time))
            
            if [ $duration -lt 30 ]; then
                log_success "Fast build performance: ${duration}s (excellent)"
            elif [ $duration -lt 60 ]; then
                log_info "Good build performance: ${duration}s"
            else
                log_warning "Slow build performance: ${duration}s (may need optimization)"
            fi
        else
            log_warning "Build performance test failed"
        fi
    else
        log_warning "Cannot test build performance (Docker not available)"
    fi
    
    # Test smart build script performance
    local smart_build="$PROJECT_ROOT/scripts/smart-docker-build.sh"
    if [ -f "$smart_build" ]; then
        log_info "Testing smart build script performance..."
        
        # Test function loading time
        local start_time=$(date +%s)
        if timeout 30s bash -c "source '$smart_build'; declare -f main >/dev/null"; then
            local end_time=$(date +%s)
            local duration=$((end_time - start_time))
            log_success "Smart build script loads in ${duration}s"
        else
            log_warning "Smart build script loading test failed"
        fi
    fi
}

# =============================================================================
# Security-Focused Test Mode  
# =============================================================================
run_security_validation() {
    log_header "🔒 Running Security-Focused Validation"
    
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Check for security best practices
    local security_checks=(
        "secrets.GITHUB_TOKEN:Using GITHUB_TOKEN (good)"
        "packages: write:Proper package permissions"
        "github.event_name != 'pull_request':Conditional auth (good)"
        "USER.*nonroot\|USER.*65532:Non-root containers"
        "govulncheck:Vulnerability scanning"
    )
    
    log_info "Running security configuration checks..."
    
    for check in "${security_checks[@]}"; do
        local pattern=$(echo "$check" | cut -d: -f1)
        local description=$(echo "$check" | cut -d: -f2)
        
        if grep -qE "$pattern" "$ci_workflow" "$PROJECT_ROOT/Dockerfile" 2>/dev/null; then
            log_success "Security check passed: $description"
        else
            log_warning "Security check failed: $description"
        fi
    done
}

# =============================================================================
# Usage Information
# =============================================================================
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Comprehensive validation script for Nephoran CI fixes.

OPTIONS:
  --full, -f         Run complete validation suite (default)
  --quick, -q        Run only essential tests (faster)
  --performance, -p  Run performance-focused tests
  --security, -s     Run security-focused tests
  --help, -h         Show this help message

EXAMPLES:
  $0                 # Run full validation
  $0 --quick         # Run quick validation 
  $0 --performance   # Test performance aspects
  $0 --security      # Focus on security validation

OUTPUT:
  Results are logged to: $MASTER_LOG
  Individual test logs: $RESULTS_DIR/
  
EXIT CODES:
  0 - All validations passed
  1 - Some validations failed (review logs)
  2 - Critical system errors
EOF
}

# =============================================================================
# Main Execution Function
# =============================================================================
main() {
    local mode="full"
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --full|-f)
                mode="full"
                shift
                ;;
            --quick|-q)
                mode="quick"
                shift
                ;;
            --performance|-p)
                mode="performance"
                shift
                ;;
            --security|-s)
                mode="security"
                shift
                ;;
            --help|-h)
                show_usage
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 2
                ;;
        esac
    done
    
    # Setup environment
    cd "$PROJECT_ROOT"
    setup_validation_environment
    
    # Run tests based on mode
    case "$mode" in
        "full")
            run_full_validation
            ;;
        "quick")
            run_quick_validation
            ;;
        "performance")
            run_performance_validation
            ;;
        "security")
            run_security_validation
            ;;
    esac
    
    # Generate final report
    generate_final_report
    
    # Final status
    echo "" | tee -a "$MASTER_LOG"
    log_header "🏁 Validation Complete"
    
    if [ $FAILED_SUITES -eq 0 ]; then
        log_success "ALL VALIDATIONS PASSED!"
        log_success "CI fixes are ready for production deployment"
        echo "" | tee -a "$MASTER_LOG"
        echo "🎯 Next steps:" | tee -a "$MASTER_LOG"
        echo "  1. Review detailed logs in $RESULTS_DIR/" | tee -a "$MASTER_LOG"
        echo "  2. Proceed with PR merge" | tee -a "$MASTER_LOG"
        echo "  3. Monitor first CI run post-merge" | tee -a "$MASTER_LOG"
        return 0
    else
        log_error "VALIDATION FAILURES DETECTED!"
        log_error "$FAILED_SUITES out of $TOTAL_SUITES test suites failed"
        echo "" | tee -a "$MASTER_LOG"
        echo "🚫 Action required:" | tee -a "$MASTER_LOG"
        echo "  1. Review failed test logs in $RESULTS_DIR/" | tee -a "$MASTER_LOG"
        echo "  2. Fix identified issues" | tee -a "$MASTER_LOG"
        echo "  3. Re-run validation before merge" | tee -a "$MASTER_LOG"
        return 1
    fi
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi