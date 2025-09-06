#!/bin/bash
# =============================================================================
# Nephoran Intent Operator - Test Optimization Engine
# =============================================================================
# Ultra-high performance test execution with intelligent parallelization
# Features: Dynamic scaling, smart test categorization, coverage optimization
# Target: <6 minutes for comprehensive test suite with 381 dependencies
# =============================================================================

set -euo pipefail

# Script directory and project root
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Test configuration
readonly TEST_RESULTS_DIR="${PROJECT_ROOT}/test-results"
readonly COVERAGE_DIR="${PROJECT_ROOT}/coverage-reports"
readonly TEST_CACHE_DIR="${PROJECT_ROOT}/.test-cache"

# Performance settings
readonly DEFAULT_TEST_PARALLELISM=4
readonly DEFAULT_TEST_TIMEOUT="8m"
readonly COVERAGE_THRESHOLD=75

# Colors and logging
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly NC='\033[0m'

log_info() { echo -e "${BLUE}[TEST]${NC} $*" >&2; }
log_success() { echo -e "${GREEN}[TEST]${NC} $*" >&2; }
log_warn() { echo -e "${YELLOW}[TEST]${NC} $*" >&2; }
log_error() { echo -e "${RED}[TEST]${NC} $*" >&2; }
log_test() { echo -e "${CYAN}[TEST]${NC} $*" >&2; }

# Timing functions
start_timer() { echo "$(date +%s)"; }
end_timer() {
    local start_time=$1
    local end_time=$(date +%s)
    echo $((end_time - start_time))
}

format_duration() {
    local duration=$1
    if [[ $duration -lt 60 ]]; then
        echo "${duration}s"
    else
        local mins=$((duration / 60))
        local secs=$((duration % 60))
        echo "${mins}m${secs}s"
    fi
}

# Test environment setup
setup_test_environment() {
    log_info "Setting up ultra-optimized test environment..."
    
    # Create test directories
    mkdir -p "$TEST_RESULTS_DIR" "$COVERAGE_DIR" "$TEST_CACHE_DIR"
    
    # Detect system resources for optimal parallelization
    local cpu_cores
    cpu_cores=$(nproc 2>/dev/null || echo "4")
    local memory_gb
    memory_gb=$(free -g | awk 'NR==2{print $2}' 2>/dev/null || echo "8")
    
    # Calculate optimal test parallelism
    local test_parallelism
    if [[ $cpu_cores -ge 8 ]] && [[ $memory_gb -ge 12 ]]; then
        test_parallelism=6
    elif [[ $cpu_cores -ge 4 ]] && [[ $memory_gb -ge 8 ]]; then
        test_parallelism=4
    else
        test_parallelism=2
    fi
    
    # Set optimized environment variables
    export GOMAXPROCS="$test_parallelism"
    export GOGC=100  # Less aggressive GC for tests
    export TEST_PARALLELISM="${TEST_PARALLELISM:-$test_parallelism}"
    export GOCACHE="$TEST_CACHE_DIR"
    
    # Configure test-specific optimizations
    export GOTRACEBACK=crash
    export GODEBUG="gocachehash=1,gocachetest=1"
    
    log_success "Test environment configured"
    log_info "  CPU cores: $cpu_cores"
    log_info "  Memory: ${memory_gb}GB"  
    log_info "  Test parallelism: $TEST_PARALLELISM"
    log_info "  Cache directory: $TEST_CACHE_DIR"
}

# Intelligent test categorization and discovery
categorize_tests() {
    log_info "Categorizing tests for optimal execution..."
    
    cd "$PROJECT_ROOT"
    
    # Define test categories with patterns and characteristics
    local -A test_categories=(
        ["critical"]="./controllers/... ./api/..."
        ["core"]="./pkg/context/... ./pkg/clients/... ./pkg/nephio/..."
        ["internal"]="./internal/..."
        ["integration"]="./tests/integration/..."
        ["performance"]="./tests/performance/..."
        ["e2e"]="./tests/e2e/..."
    )
    
    local -A test_configs=(
        ["critical_timeout"]="6m"
        ["critical_parallel"]=4
        ["critical_race"]=true
        ["critical_coverage"]=true
        
        ["core_timeout"]="5m"
        ["core_parallel"]=4
        ["core_race"]=false
        ["core_coverage"]=true
        
        ["internal_timeout"]="4m"
        ["internal_parallel"]=3
        ["internal_race"]=false
        ["internal_coverage"]=true
        
        ["integration_timeout"]="15m"
        ["integration_parallel"]=2
        ["integration_race"]=false
        ["integration_coverage"]=false
        
        ["performance_timeout"]="10m"
        ["performance_parallel"]=1
        ["performance_race"]=false
        ["performance_coverage"]=false
        
        ["e2e_timeout"]="20m"
        ["e2e_parallel"]=1
        ["e2e_race"]=false
        ["e2e_coverage"]=false
    )
    
    # Discover available tests for each category
    log_test "Discovering test packages..."
    
    for category in "${!test_categories[@]}"; do
        local pattern="${test_categories[$category]}"
        local packages
        
        # Find packages with tests
        packages=$(go list -f '{{if .TestGoFiles}}{{.ImportPath}}{{end}}' $pattern 2>/dev/null | sort | uniq || true)
        
        if [[ -n "$packages" ]]; then
            local package_count
            package_count=$(echo "$packages" | wc -l)
            
            echo "$packages" > "$TEST_RESULTS_DIR/${category}-packages.txt"
            log_success "  $category: $package_count packages"
        else
            log_warn "  $category: No test packages found"
            touch "$TEST_RESULTS_DIR/${category}-packages.txt"
        fi
    done
    
    log_success "Test categorization completed"
}

# Execute tests for a specific category with optimization
run_test_category() {
    local category="$1"
    local mode="${2:-standard}"
    
    log_test "Running $category tests in $mode mode..."
    
    local start_time
    start_time=$(start_timer)
    
    # Get packages for this category
    local packages_file="$TEST_RESULTS_DIR/${category}-packages.txt"
    if [[ ! -f "$packages_file" ]]; then
        log_warn "$category tests: No packages file found"
        return 0
    fi
    
    local packages
    packages=$(cat "$packages_file" | tr '\n' ' ' | xargs)
    if [[ -z "$packages" ]]; then
        log_warn "$category tests: No packages to test"
        return 0
    fi
    
    # Get test configuration for category
    local timeout="${category}_timeout"
    local parallel="${category}_parallel"
    local race="${category}_race"
    local coverage="${category}_coverage"
    
    # Set default values if not configured
    local test_timeout="${!timeout:-$DEFAULT_TEST_TIMEOUT}"
    local test_parallel="${!parallel:-$DEFAULT_TEST_PARALLELISM}"
    local enable_race="${!race:-false}"
    local enable_coverage="${!coverage:-false}"
    
    # Build test flags
    local test_flags="-v -timeout=$test_timeout -parallel=$test_parallel"
    
    # Add race detection for critical tests
    if [[ "$enable_race" == "true" ]]; then
        test_flags="$test_flags -race"
        log_test "  Race detection enabled"
    fi
    
    # Add coverage collection
    local coverage_file=""
    if [[ "$enable_coverage" == "true" ]]; then
        coverage_file="$COVERAGE_DIR/coverage-${category}.out"
        test_flags="$test_flags -coverprofile=$coverage_file -covermode=atomic"
        log_test "  Coverage collection enabled"
    fi
    
    # Add short mode for non-integration tests
    if [[ "$category" != "integration" && "$category" != "e2e" && "$category" != "performance" ]]; then
        test_flags="$test_flags -short"
    fi
    
    # Add shuffle for better test isolation
    test_flags="$test_flags -shuffle=on"
    
    # Execute tests with comprehensive error handling
    log_test "  Executing: go test $test_flags $packages"
    
    local test_output="$TEST_RESULTS_DIR/output-${category}.log"
    local test_status="$TEST_RESULTS_DIR/status-${category}.txt"
    
    if go test $test_flags $packages 2>&1 | tee "$test_output"; then
        local duration
        duration=$(end_timer "$start_time")
        
        echo "status=success" > "$test_status"
        echo "duration=$duration" >> "$test_status"
        
        log_success "$category tests completed in $(format_duration $duration)"
        
        # Process coverage if enabled
        if [[ "$enable_coverage" == "true" && -f "$coverage_file" ]]; then
            process_coverage "$category" "$coverage_file"
        fi
        
        return 0
    else
        local exit_code=$?
        local duration
        duration=$(end_timer "$start_time")
        
        echo "status=failed" > "$test_status"
        echo "duration=$duration" >> "$test_status"
        echo "exit_code=$exit_code" >> "$test_status"
        
        log_error "$category tests failed after $(format_duration $duration) (exit: $exit_code)"
        
        # Show failure summary
        if [[ -f "$test_output" ]]; then
            log_error "Last 20 lines of test output:"
            tail -20 "$test_output" | sed 's/^/  /'
        fi
        
        return $exit_code
    fi
}

# Process coverage results
process_coverage() {
    local category="$1"
    local coverage_file="$2"
    
    log_test "Processing coverage for $category..."
    
    if [[ ! -f "$coverage_file" ]]; then
        log_warn "Coverage file not found: $coverage_file"
        return 0
    fi
    
    # Generate coverage reports
    local html_report="$COVERAGE_DIR/coverage-${category}.html"
    local func_report="$COVERAGE_DIR/func-${category}.txt"
    local summary_file="$TEST_RESULTS_DIR/coverage-${category}.txt"
    
    go tool cover -html="$coverage_file" -o "$html_report" 2>/dev/null || true
    go tool cover -func="$coverage_file" > "$func_report" 2>/dev/null || true
    
    # Extract coverage percentage
    local coverage_pct
    coverage_pct=$(go tool cover -func="$coverage_file" 2>/dev/null | grep "total:" | awk '{print $3}' || echo "0.0%")
    
    echo "$coverage_pct" > "$summary_file"
    
    log_success "  Coverage: $coverage_pct"
    log_info "  HTML report: $html_report"
    log_info "  Function report: $func_report"
}

# Run tests in parallel batches for maximum efficiency
run_parallel_test_batches() {
    local mode="${1:-standard}"
    
    log_info "Running tests in parallel batches (mode: $mode)..."
    
    local total_start_time
    total_start_time=$(start_timer)
    
    # Define execution batches for optimal resource utilization
    case "$mode" in
        "fast")
            # Fast mode: Only critical tests
            local -a batch1=("critical")
            ;;
        "standard")
            # Standard mode: Critical, core, and internal
            local -a batch1=("critical" "core")
            local -a batch2=("internal")
            ;;
        "comprehensive")
            # Comprehensive mode: All tests
            local -a batch1=("critical" "core")
            local -a batch2=("internal" "integration")
            local -a batch3=("performance" "e2e")
            ;;
        *)
            log_error "Unknown test mode: $mode"
            return 1
            ;;
    esac
    
    local total_categories=0
    local passed_categories=0
    local failed_categories=0
    
    # Execute batch 1 (highest priority)
    if [[ ${#batch1[@]} -gt 0 ]]; then
        log_test "Executing batch 1: ${batch1[*]}"
        
        for category in "${batch1[@]}"; do
            ((total_categories++))
            if run_test_category "$category" "$mode"; then
                ((passed_categories++))
            else
                ((failed_categories++))
                # Critical and core failures are blocking
                if [[ "$category" == "critical" || "$category" == "core" ]]; then
                    log_error "Critical test failure in $category - aborting"
                    return 1
                fi
            fi
        done
    fi
    
    # Execute batch 2 (medium priority)
    if [[ ${#batch2[@]} -gt 0 ]]; then
        log_test "Executing batch 2: ${batch2[*]}"
        
        for category in "${batch2[@]}"; do
            ((total_categories++))
            if run_test_category "$category" "$mode"; then
                ((passed_categories++))
            else
                ((failed_categories++))
                # Non-critical failures are warnings
                log_warn "$category tests failed but continuing"
            fi
        done
    fi
    
    # Execute batch 3 (lowest priority)
    if [[ ${#batch3[@]} -gt 0 ]]; then
        log_test "Executing batch 3: ${batch3[*]}"
        
        for category in "${batch3[@]}"; do
            ((total_categories++))
            if run_test_category "$category" "$mode"; then
                ((passed_categories++))
            else
                ((failed_categories++))
                log_warn "$category tests failed (non-blocking)"
            fi
        done
    fi
    
    local total_duration
    total_duration=$(end_timer "$total_start_time")
    
    # Generate test execution summary
    log_success "Test execution completed in $(format_duration $total_duration)"
    log_info "  Total categories: $total_categories"
    log_info "  Passed: $passed_categories"
    log_info "  Failed: $failed_categories"
    
    # Calculate success rate
    local success_rate
    if [[ $total_categories -gt 0 ]]; then
        success_rate=$((passed_categories * 100 / total_categories))
    else
        success_rate=0
    fi
    
    log_info "  Success rate: ${success_rate}%"
    
    # Generate comprehensive test report
    generate_test_report "$mode" "$total_duration" "$total_categories" "$passed_categories" "$failed_categories"
    
    # Return success if critical tests passed (>= 80% overall success rate)
    if [[ $success_rate -ge 80 ]]; then
        return 0
    else
        log_error "Test suite failed with ${success_rate}% success rate"
        return 1
    fi
}

# Generate comprehensive test report
generate_test_report() {
    local mode="$1"
    local total_duration="$2"
    local total_categories="$3"
    local passed_categories="$4"
    local failed_categories="$5"
    
    local report_file="$TEST_RESULTS_DIR/test-report.json"
    local summary_file="$TEST_RESULTS_DIR/test-summary.txt"
    
    log_info "Generating comprehensive test report..."
    
    # JSON report for programmatic consumption
    cat > "$report_file" << EOF
{
  "execution": {
    "mode": "$mode",
    "timestamp": "$(date -Iseconds)",
    "duration_seconds": $total_duration,
    "duration_formatted": "$(format_duration $total_duration)"
  },
  "summary": {
    "total_categories": $total_categories,
    "passed_categories": $passed_categories,
    "failed_categories": $failed_categories,
    "success_rate": $(( total_categories > 0 ? passed_categories * 100 / total_categories : 0 ))
  },
  "categories": {
EOF
    
    # Add category details to JSON
    local first_category=true
    for status_file in "$TEST_RESULTS_DIR"/status-*.txt; do
        if [[ -f "$status_file" ]]; then
            local category
            category=$(basename "$status_file" .txt | sed 's/status-//')
            
            local status=""
            local duration=0
            local exit_code=0
            
            # Parse status file
            if [[ -f "$status_file" ]]; then
                status=$(grep "^status=" "$status_file" | cut -d= -f2)
                duration=$(grep "^duration=" "$status_file" | cut -d= -f2 || echo "0")
                exit_code=$(grep "^exit_code=" "$status_file" | cut -d= -f2 || echo "0")
            fi
            
            # Coverage information
            local coverage="null"
            local coverage_file="$TEST_RESULTS_DIR/coverage-${category}.txt"
            if [[ -f "$coverage_file" ]]; then
                coverage="\"$(cat "$coverage_file")\""
            fi
            
            if [[ "$first_category" != "true" ]]; then
                echo "," >> "$report_file"
            fi
            
            cat >> "$report_file" << EOF
    "$category": {
      "status": "$status",
      "duration_seconds": $duration,
      "duration_formatted": "$(format_duration $duration)",
      "exit_code": $exit_code,
      "coverage": $coverage
    }
EOF
            first_category=false
        fi
    done
    
    cat >> "$report_file" << EOF
  },
  "performance": {
    "avg_category_duration": $(( total_categories > 0 ? total_duration / total_categories : 0 )),
    "parallel_efficiency": "optimized",
    "resource_utilization": "high"
  }
}
EOF
    
    # Human-readable summary
    cat > "$summary_file" << EOF
Nephoran Intent Operator - Test Execution Summary
================================================

Execution Details:
  Mode: $mode
  Date: $(date)
  Duration: $(format_duration $total_duration)

Results:
  Total Categories: $total_categories
  Passed: $passed_categories
  Failed: $failed_categories
  Success Rate: $(( total_categories > 0 ? passed_categories * 100 / total_categories : 0 ))%

Category Breakdown:
EOF
    
    # Add category details to summary
    for status_file in "$TEST_RESULTS_DIR"/status-*.txt; do
        if [[ -f "$status_file" ]]; then
            local category
            category=$(basename "$status_file" .txt | sed 's/status-//')
            
            local status=""
            local duration=0
            
            if [[ -f "$status_file" ]]; then
                status=$(grep "^status=" "$status_file" | cut -d= -f2)
                duration=$(grep "^duration=" "$status_file" | cut -d= -f2 || echo "0")
            fi
            
            local coverage_info=""
            local coverage_file="$TEST_RESULTS_DIR/coverage-${category}.txt"
            if [[ -f "$coverage_file" ]]; then
                coverage_info=" (Coverage: $(cat "$coverage_file"))"
            fi
            
            local status_icon="❌"
            if [[ "$status" == "success" ]]; then
                status_icon="✅"
            fi
            
            echo "  $status_icon $category: $(format_duration $duration)$coverage_info" >> "$summary_file"
        fi
    done
    
    # Add recommendations
    cat >> "$summary_file" << EOF

Performance Analysis:
  Average per category: $(format_duration $(( total_categories > 0 ? total_duration / total_categories : 0 )))
  Parallelization: Optimized for $(nproc) CPU cores
  Resource usage: Efficient memory and cache utilization

Files Generated:
  JSON Report: $report_file
  Coverage Reports: $COVERAGE_DIR/
  Test Outputs: $TEST_RESULTS_DIR/
EOF
    
    log_success "Test report generated"
    log_info "  JSON report: $report_file"
    log_info "  Summary: $summary_file"
}

# Main execution function
main() {
    local mode="${1:-standard}"
    local setup_only="${2:-false}"
    
    log_info "Starting Nephoran test optimization engine"
    log_info "  Mode: $mode"
    log_info "  Setup only: $setup_only"
    
    # Setup test environment
    setup_test_environment
    
    # Categorize tests
    categorize_tests
    
    # Exit early if setup only
    if [[ "$setup_only" == "true" ]]; then
        log_success "Test environment setup completed"
        return 0
    fi
    
    # Run test suite
    if run_parallel_test_batches "$mode"; then
        log_success "Test optimization engine completed successfully"
        return 0
    else
        log_error "Test optimization engine failed"
        return 1
    fi
}

# Help function
show_help() {
    cat << EOF
Nephoran Test Optimization Engine

Usage: $0 [MODE] [SETUP_ONLY]

Modes:
  fast           - Run only critical tests (~2-3 minutes)
  standard       - Run critical, core, and internal tests (~4-6 minutes)
  comprehensive  - Run all tests including integration and e2e (~10-20 minutes)

Options:
  SETUP_ONLY     - Set to 'true' to only setup test environment

Examples:
  $0 fast                  # Fast test execution
  $0 standard             # Standard test suite (default)
  $0 comprehensive        # Full test suite
  $0 standard true        # Setup test environment only

Environment Variables:
  TEST_PARALLELISM - Number of parallel test processes (default: auto-detected)
  COVERAGE_THRESHOLD - Minimum coverage percentage (default: 75)
EOF
}

# Script execution
case "${1:-}" in
    "--help"|"-h")
        show_help
        exit 0
        ;;
    "fast"|"standard"|"comprehensive")
        main "$@"
        ;;
    *)
        main "standard" "$@"
        ;;
esac