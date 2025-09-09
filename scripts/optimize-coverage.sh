#!/bin/bash
# =============================================================================
# Nephoran Coverage Optimization Script
# =============================================================================
# This script optimizes Go test coverage collection with minimal overhead
# Features:
# - Selective coverage collection (only for critical packages)
# - Coverage merging and aggregation
# - Differential coverage (only changed code)
# - Performance-optimized coverage analysis
# =============================================================================

set -euo pipefail

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COVERAGE_DIR="${PROJECT_ROOT}/coverage-reports"
MERGED_COVERAGE="${COVERAGE_DIR}/merged-coverage.out"
DIFFERENTIAL_COVERAGE="${COVERAGE_DIR}/differential-coverage.out"

# Coverage thresholds
MIN_COVERAGE_OVERALL=65
MIN_COVERAGE_CRITICAL=80
MIN_COVERAGE_NEW_CODE=75

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Initialize coverage system
init_coverage() {
    log_info "Initializing optimized coverage system..."
    mkdir -p "$COVERAGE_DIR"
    log_success "Coverage system initialized"
}

# Get critical packages that require coverage
get_critical_packages() {
    cat << 'EOF'
./pkg/context/...
./pkg/clients/...
./pkg/nephio/...
./controllers/...
./api/...
./internal/conductor/...
./internal/ingest/...
./internal/intent/...
EOF
}

# Get all packages for comprehensive coverage
get_all_packages() {
    find . -name "*.go" -not -path "./.git/*" -not -path "./vendor/*" -not -name "*_test.go" | \
    xargs dirname | sort -u | sed 's|^|./|' | sed 's|$|/...|'
}

# Run tests with optimized coverage collection
run_with_coverage() {
    local package_list="$1"
    local coverage_mode="${2:-atomic}"  # atomic, count, set
    local parallel="${3:-4}"
    local timeout="${4:-8m}"
    
    log_info "Running tests with optimized coverage collection..."
    log_info "Packages: $(echo "$package_list" | wc -l) packages"
    log_info "Mode: $coverage_mode, Parallel: $parallel, Timeout: $timeout"
    
    init_coverage
    
    local coverage_files=()
    local package_count=0
    local total_packages
    total_packages=$(echo "$package_list" | wc -l)
    
    echo "$package_list" | while read -r package; do
        if [[ -z "$package" ]]; then
            continue
        fi
        
        ((package_count++))
        log_info "[$package_count/$total_packages] Testing package: $package"
        
        # Generate unique coverage file name
        local pkg_name
        pkg_name=$(echo "$package" | sed 's|[./]||g' | sed 's|\.\.\.||g')
        local coverage_file="$COVERAGE_DIR/coverage-${pkg_name}.out"
        
        # Run test with coverage
        local test_cmd="go test -v -timeout=$timeout -parallel=$parallel -coverprofile=$coverage_file -covermode=$coverage_mode $package"
        
        log_info "Executing: $test_cmd"
        
        if timeout $((${timeout%m} * 60 + 30)) $test_cmd 2>&1 | tee "$COVERAGE_DIR/test-${pkg_name}.log"; then
            if [[ -f "$coverage_file" ]] && [[ -s "$coverage_file" ]]; then
                log_success "Coverage collected for $package"
                coverage_files+=("$coverage_file")
            else
                log_warning "No coverage data generated for $package"
            fi
        else
            log_error "Tests failed for $package"
            # Continue with other packages
        fi
    done
    
    log_info "Coverage collection completed for ${#coverage_files[@]} packages"
}

# Merge multiple coverage files into one
merge_coverage_files() {
    local output_file="${1:-$MERGED_COVERAGE}"
    
    log_info "Merging coverage files into $output_file..."
    
    local coverage_files
    coverage_files=$(find "$COVERAGE_DIR" -name "coverage-*.out" -type f)
    
    if [[ -z "$coverage_files" ]]; then
        log_error "No coverage files found to merge"
        return 1
    fi
    
    local file_count
    file_count=$(echo "$coverage_files" | wc -l)
    log_info "Found $file_count coverage files to merge"
    
    # Create merged coverage file
    echo "mode: atomic" > "$output_file"
    
    # Merge all coverage files, skipping the mode line
    echo "$coverage_files" | while read -r file; do
        if [[ -f "$file" ]]; then
            tail -n +2 "$file" >> "$output_file"
        fi
    done
    
    # Sort and deduplicate lines
    local temp_file="${output_file}.tmp"
    (
        head -n 1 "$output_file"
        tail -n +2 "$output_file" | sort -u
    ) > "$temp_file"
    mv "$temp_file" "$output_file"
    
    log_success "Coverage files merged: $output_file"
    
    # Generate summary
    if command -v go >/dev/null 2>&1; then
        local total_coverage
        total_coverage=$(go tool cover -func="$output_file" | grep "total:" | awk '{print $3}' | sed 's/%//' || echo "0")
        log_info "Total coverage: $total_coverage%"
        
        return 0
    fi
}

# Generate differential coverage for changed files only
generate_differential_coverage() {
    local base_ref="${1:-HEAD~1}"
    local output_file="${2:-$DIFFERENTIAL_COVERAGE}"
    
    log_info "Generating differential coverage since $base_ref..."
    
    # Get changed Go files
    local changed_files
    changed_files=$(git diff --name-only "$base_ref" HEAD | grep "\.go$" | grep -v "_test\.go$" || echo "")
    
    if [[ -z "$changed_files" ]]; then
        log_info "No Go source files changed"
        return 0
    fi
    
    log_info "Changed files:"
    echo "$changed_files" | head -10
    
    if [[ ! -f "$MERGED_COVERAGE" ]]; then
        log_error "Merged coverage file not found: $MERGED_COVERAGE"
        return 1
    fi
    
    # Filter coverage for changed files only
    echo "mode: atomic" > "$output_file"
    
    echo "$changed_files" | while read -r file; do
        # Convert file path to Go package path format
        local pkg_path
        pkg_path=$(echo "$file" | sed 's|^./||' | sed 's|\.go$||')
        
        # Extract coverage lines for this file
        grep "^$pkg_path" "$MERGED_COVERAGE" >> "$output_file" 2>/dev/null || true
    done
    
    if [[ $(wc -l < "$output_file") -gt 1 ]]; then
        local diff_coverage
        diff_coverage=$(go tool cover -func="$output_file" | grep "total:" | awk '{print $3}' | sed 's/%//' || echo "0")
        log_success "Differential coverage generated: $diff_coverage%"
        
        # Check if differential coverage meets threshold
        if (( $(echo "$diff_coverage >= $MIN_COVERAGE_NEW_CODE" | bc -l) )); then
            log_success "Differential coverage meets threshold ($MIN_COVERAGE_NEW_CODE%)"
        else
            log_error "Differential coverage below threshold: $diff_coverage% < $MIN_COVERAGE_NEW_CODE%"
            return 1
        fi
    else
        log_info "No coverage data for changed files"
    fi
}

# Generate HTML coverage reports
generate_html_reports() {
    log_info "Generating HTML coverage reports..."
    
    # Overall coverage report
    if [[ -f "$MERGED_COVERAGE" ]]; then
        log_info "Generating overall coverage HTML..."
        go tool cover -html="$MERGED_COVERAGE" -o "$COVERAGE_DIR/coverage-overall.html"
        log_success "Overall coverage HTML: $COVERAGE_DIR/coverage-overall.html"
    fi
    
    # Differential coverage report
    if [[ -f "$DIFFERENTIAL_COVERAGE" ]]; then
        log_info "Generating differential coverage HTML..."
        go tool cover -html="$DIFFERENTIAL_COVERAGE" -o "$COVERAGE_DIR/coverage-differential.html"
        log_success "Differential coverage HTML: $COVERAGE_DIR/coverage-differential.html"
    fi
    
    # Individual package reports for critical packages
    find "$COVERAGE_DIR" -name "coverage-*.out" -type f | head -5 | while read -r coverage_file; do
        local basename
        basename=$(basename "$coverage_file" .out)
        local html_file="$COVERAGE_DIR/${basename}.html"
        
        if go tool cover -html="$coverage_file" -o "$html_file" 2>/dev/null; then
            log_info "Generated: $html_file"
        fi
    done
}

# Analyze coverage and generate detailed report
analyze_coverage() {
    local coverage_file="${1:-$MERGED_COVERAGE}"
    
    log_info "Analyzing coverage data from $coverage_file..."
    
    if [[ ! -f "$coverage_file" ]]; then
        log_error "Coverage file not found: $coverage_file"
        return 1
    fi
    
    local report_file="$COVERAGE_DIR/coverage-analysis.md"
    
    cat > "$report_file" << 'EOF'
# Nephoran Coverage Analysis Report

## Overview
EOF
    
    # Overall statistics
    local func_coverage
    func_coverage=$(go tool cover -func="$coverage_file" | grep "total:" | awk '{print $3}' || echo "0.0%")
    
    local total_statements
    total_statements=$(go tool cover -func="$coverage_file" | grep -v "total:" | awk '{sum += $2} END {print sum}' || echo "0")
    
    local covered_statements  
    covered_statements=$(go tool cover -func="$coverage_file" | grep -v "total:" | awk '{sum += $3} END {print sum}' || echo "0")
    
    cat >> "$report_file" << EOF
- **Overall Coverage**: $func_coverage
- **Total Statements**: $total_statements  
- **Covered Statements**: $covered_statements
- **Analysis Date**: $(date -u +%Y-%m-%dT%H:%M:%SZ)

## Coverage by Package
EOF
    
    # Package-level coverage
    go tool cover -func="$coverage_file" | grep -v "total:" | awk '{print $1}' | sed 's|/[^/]*$||' | sort | uniq -c | sort -nr | head -20 | while read -r count pkg; do
        local pkg_coverage
        pkg_coverage=$(go tool cover -func="$coverage_file" | grep "^$pkg/" | awk '{sum += $2; cov += $3} END {if (sum > 0) print (cov/sum)*100 "%"; else print "0.0%"}' 2>/dev/null || echo "0.0%")
        echo "- **$pkg**: $pkg_coverage ($count functions)" >> "$report_file"
    done
    
    # Low coverage functions (below 50%)
    echo "" >> "$report_file"
    echo "## Low Coverage Functions (< 50%)" >> "$report_file"
    
    go tool cover -func="$coverage_file" | grep -v "total:" | awk '$3 < 50 {print "- " $1 ": " $3 "%"}' | head -20 >> "$report_file"
    
    # High coverage functions (>= 90%)
    echo "" >> "$report_file"
    echo "## Well-Tested Functions (≥ 90%)" >> "$report_file"
    
    local high_coverage_count
    high_coverage_count=$(go tool cover -func="$coverage_file" | grep -v "total:" | awk '$3 >= 90' | wc -l)
    echo "- **Count**: $high_coverage_count functions" >> "$report_file"
    
    # Coverage recommendations
    cat >> "$report_file" << 'EOF'

## Recommendations

1. **Priority**: Focus on low-coverage functions in critical packages
2. **Target**: Aim for 80%+ coverage in controllers and API packages
3. **Strategy**: Add unit tests for uncovered error handling paths
4. **Monitoring**: Set up coverage tracking in CI to prevent regression

## Coverage Thresholds

EOF
    
    local coverage_num
    coverage_num=$(echo "$func_coverage" | sed 's/%//')
    
    if (( $(echo "$coverage_num >= $MIN_COVERAGE_OVERALL" | bc -l) )); then
        echo "- ✅ **Overall Coverage**: Meets threshold ($MIN_COVERAGE_OVERALL%)" >> "$report_file"
    else
        echo "- ❌ **Overall Coverage**: Below threshold ($coverage_num% < $MIN_COVERAGE_OVERALL%)" >> "$report_file"
    fi
    
    log_success "Coverage analysis report generated: $report_file"
    
    # Display summary
    echo ""
    echo "📊 Coverage Summary:"
    echo "  Overall: $func_coverage"
    echo "  Statements: $covered_statements/$total_statements"
    echo "  Report: $report_file"
    
    # Return exit code based on coverage threshold
    if (( $(echo "$coverage_num >= $MIN_COVERAGE_OVERALL" | bc -l) )); then
        return 0
    else
        return 1
    fi
}

# Optimize coverage collection by excluding test files and vendor
optimize_coverage_collection() {
    local package_pattern="${1:-./...}"
    local mode="${2:-critical}"  # critical, comprehensive, differential
    
    log_info "Running optimized coverage collection (mode: $mode)..."
    
    init_coverage
    
    local packages=""
    case "$mode" in
        "critical")
            packages=$(get_critical_packages)
            log_info "Running coverage for critical packages only"
            ;;
        "comprehensive")
            packages=$(get_all_packages)
            log_info "Running comprehensive coverage for all packages"
            ;;
        "differential")
            # Get packages affected by changes
            local changed_files
            changed_files=$(git diff --name-only HEAD~1 HEAD | grep "\.go$" | grep -v "_test\.go$" || echo "")
            if [[ -n "$changed_files" ]]; then
                packages=$(echo "$changed_files" | xargs dirname | sort -u | sed 's|^|./|' | sed 's|$|/...|')
                log_info "Running coverage for changed packages only"
            else
                packages=$(get_critical_packages)
                log_info "No changes found, running critical package coverage"
            fi
            ;;
        *)
            packages="$package_pattern"
            ;;
    esac
    
    # Run coverage collection
    run_with_coverage "$packages" "atomic" 4 "6m"
    
    # Merge results
    merge_coverage_files
    
    # Generate differential coverage if requested
    if [[ "$mode" == "differential" ]]; then
        generate_differential_coverage
    fi
    
    # Generate reports
    generate_html_reports
    analyze_coverage
}

# Clean old coverage data
clean_coverage() {
    local max_age_days="${1:-7}"
    
    log_info "Cleaning coverage data older than $max_age_days days..."
    
    # Clean old coverage files
    find "$COVERAGE_DIR" -name "coverage-*.out" -mtime +$max_age_days -delete 2>/dev/null || true
    find "$COVERAGE_DIR" -name "coverage-*.html" -mtime +$max_age_days -delete 2>/dev/null || true
    find "$COVERAGE_DIR" -name "test-*.log" -mtime +$max_age_days -delete 2>/dev/null || true
    
    log_success "Coverage data cleaned"
}

# Validate coverage thresholds
validate_coverage() {
    local coverage_file="${1:-$MERGED_COVERAGE}"
    local threshold="${2:-$MIN_COVERAGE_OVERALL}"
    
    if [[ ! -f "$coverage_file" ]]; then
        log_error "Coverage file not found: $coverage_file"
        return 1
    fi
    
    local coverage_pct
    coverage_pct=$(go tool cover -func="$coverage_file" | grep "total:" | awk '{print $3}' | sed 's/%//' || echo "0")
    
    log_info "Validating coverage: $coverage_pct% (threshold: $threshold%)"
    
    if (( $(echo "$coverage_pct >= $threshold" | bc -l) )); then
        log_success "Coverage validation passed: $coverage_pct% >= $threshold%"
        return 0
    else
        log_error "Coverage validation failed: $coverage_pct% < $threshold%"
        return 1
    fi
}

# Main command handler
case "${1:-help}" in
    "init")
        init_coverage
        ;;
    "run")
        optimize_coverage_collection "${2:-./...}" "${3:-critical}"
        ;;
    "merge")
        merge_coverage_files "${2:-$MERGED_COVERAGE}"
        ;;
    "diff")
        generate_differential_coverage "${2:-HEAD~1}" "${3:-$DIFFERENTIAL_COVERAGE}"
        ;;
    "html")
        generate_html_reports
        ;;
    "analyze")
        analyze_coverage "${2:-$MERGED_COVERAGE}"
        ;;
    "validate")
        validate_coverage "${2:-$MERGED_COVERAGE}" "${3:-$MIN_COVERAGE_OVERALL}"
        ;;
    "clean")
        clean_coverage "${2:-7}"
        ;;
    "critical")
        optimize_coverage_collection "./..." "critical"
        ;;
    "comprehensive")  
        optimize_coverage_collection "./..." "comprehensive"
        ;;
    "differential")
        optimize_coverage_collection "./..." "differential"
        ;;
    "help"|*)
        echo "Nephoran Coverage Optimization System"
        echo ""
        echo "Usage: $0 <command> [options]"
        echo ""
        echo "Commands:"
        echo "  init                      - Initialize coverage system"
        echo "  run [pattern] [mode]      - Run optimized coverage collection"
        echo "  merge [output]            - Merge coverage files"
        echo "  diff [base] [output]      - Generate differential coverage"
        echo "  html                      - Generate HTML coverage reports"
        echo "  analyze [file]            - Analyze coverage and generate report"
        echo "  validate [file] [threshold] - Validate coverage meets threshold"
        echo "  clean [days]              - Clean old coverage data"
        echo "  critical                  - Run coverage for critical packages"
        echo "  comprehensive             - Run coverage for all packages"
        echo "  differential              - Run coverage for changed packages"
        echo "  help                      - Show this help"
        echo ""
        echo "Coverage Modes:"
        echo "  critical      - Critical packages only (fastest)"
        echo "  comprehensive - All packages (complete)"
        echo "  differential  - Changed packages only (smart)"
        echo ""
        echo "Examples:"
        echo "  $0 critical                           # Quick coverage for critical packages"
        echo "  $0 run './pkg/...' critical          # Coverage for pkg packages"
        echo "  $0 differential                       # Coverage for changed code only"
        echo "  $0 validate coverage.out 80          # Validate 80% coverage threshold"
        echo ""
        echo "Thresholds:"
        echo "  Overall: $MIN_COVERAGE_OVERALL%"
        echo "  Critical: $MIN_COVERAGE_CRITICAL%"  
        echo "  New Code: $MIN_COVERAGE_NEW_CODE%"
        ;;
esac