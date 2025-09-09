#!/bin/bash
# =============================================================================
# Nephoran Smart Test Cache and Selection System
# =============================================================================
# This script implements intelligent test caching and selection based on:
# - File change analysis
# - Test dependency mapping  
# - Historical test results
# - Performance metrics
# =============================================================================

set -euo pipefail

# Configuration
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CACHE_DIR="${PROJECT_ROOT}/.test-cache"
TEST_RESULTS_DIR="${PROJECT_ROOT}/test-results"
DEPENDENCY_MAP="${CACHE_DIR}/dependency-map.json"
TEST_HISTORY="${CACHE_DIR}/test-history.json"
PERFORMANCE_METRICS="${CACHE_DIR}/performance-metrics.json"

# Cache version for invalidation
CACHE_VERSION="v2.1"

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

# Initialize cache system
init_cache() {
    log_info "Initializing smart test cache system..."
    
    mkdir -p "$CACHE_DIR" "$TEST_RESULTS_DIR"
    
    # Initialize dependency map
    if [[ ! -f "$DEPENDENCY_MAP" ]]; then
        cat > "$DEPENDENCY_MAP" << EOF
{
  "version": "$CACHE_VERSION",
  "last_updated": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "dependencies": {}
}
EOF
    fi
    
    # Initialize test history
    if [[ ! -f "$TEST_HISTORY" ]]; then
        cat > "$TEST_HISTORY" << EOF
{
  "version": "$CACHE_VERSION",
  "runs": [],
  "test_results": {},
  "last_cleanup": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
    fi
    
    # Initialize performance metrics
    if [[ ! -f "$PERFORMANCE_METRICS" ]]; then
        cat > "$PERFORMANCE_METRICS" << EOF
{
  "version": "$CACHE_VERSION",
  "test_timings": {},
  "parallel_efficiency": {},
  "resource_usage": {},
  "last_updated": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
    fi
    
    log_success "Cache system initialized"
}

# Build dependency map between files and tests
build_dependency_map() {
    log_info "Building test dependency map..."
    
    local temp_map="$CACHE_DIR/temp-dep-map.json"
    
    # Start with empty dependencies object
    jq --arg version "$CACHE_VERSION" \
       --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
       '{
         version: $version,
         last_updated: $timestamp,
         dependencies: {}
       }' <<< '{}' > "$temp_map"
    
    # Find all Go test files and analyze imports
    find . -name "*_test.go" -not -path "./.git/*" | while read -r test_file; do
        local package_dir
        package_dir=$(dirname "$test_file")
        local package_name
        package_name=$(basename "$package_dir")
        
        log_info "Analyzing dependencies for $test_file"
        
        # Extract import statements and local package references
        local dependencies=()
        
        # Get imports from test file
        local imports
        imports=$(grep -E "^\s*\"[^\"]+\"" "$test_file" 2>/dev/null | sed 's/.*"\([^"]*\)".*/\1/' | grep -v "^[^/]*$" || echo "")
        
        # Add local package files that this test might depend on
        if [[ -d "$package_dir" ]]; then
            local go_files
            go_files=$(find "$package_dir" -maxdepth 1 -name "*.go" -not -name "*_test.go" | tr '\n' ',' | sed 's/,$//')
            if [[ -n "$go_files" ]]; then
                dependencies+=("$go_files")
            fi
        fi
        
        # Add imported package paths
        if [[ -n "$imports" ]]; then
            echo "$imports" | while read -r import_path; do
                if [[ -n "$import_path" && "$import_path" == *"nephoran-intent-operator"* ]]; then
                    # Map internal imports to actual file paths
                    local internal_path
                    internal_path=$(echo "$import_path" | sed 's|.*nephoran-intent-operator/||')
                    if [[ -d "$internal_path" ]]; then
                        local dep_files
                        dep_files=$(find "$internal_path" -name "*.go" -not -name "*_test.go" | head -10 | tr '\n' ',' | sed 's/,$//')
                        dependencies+=("$dep_files")
                    fi
                fi
            done
        fi
        
        # Update dependency map
        if [[ ${#dependencies[@]} -gt 0 ]]; then
            local dep_list
            dep_list=$(printf '%s\n' "${dependencies[@]}" | jq -R . | jq -s .)
            
            jq --arg test "$test_file" \
               --argjson deps "$dep_list" \
               '.dependencies[$test] = $deps' "$temp_map" > "$temp_map.tmp"
            mv "$temp_map.tmp" "$temp_map"
        fi
    done
    
    # Finalize dependency map
    mv "$temp_map" "$DEPENDENCY_MAP"
    log_success "Dependency map built with $(jq '.dependencies | keys | length' "$DEPENDENCY_MAP") test files"
}

# Get changed files since last commit or specific revision
get_changed_files() {
    local base_ref="${1:-HEAD~1}"
    local changed_files
    
    log_info "Analyzing changed files since $base_ref..."
    
    # Get changed files
    if git rev-parse --verify "$base_ref" >/dev/null 2>&1; then
        changed_files=$(git diff --name-only "$base_ref" HEAD | grep "\.go$" || echo "")
    else
        log_warning "Base reference $base_ref not found, using git status"
        changed_files=$(git status --porcelain | grep "\.go$" | awk '{print $2}' || echo "")
    fi
    
    if [[ -z "$changed_files" ]]; then
        log_info "No Go files changed"
        return 1
    fi
    
    log_info "Changed files:"
    echo "$changed_files" | head -10
    
    echo "$changed_files"
}

# Select tests affected by changed files
select_affected_tests() {
    local changed_files="$1"
    local affected_tests=""
    
    log_info "Selecting tests affected by changed files..."
    
    if [[ ! -f "$DEPENDENCY_MAP" ]]; then
        log_warning "Dependency map not found, building it first..."
        build_dependency_map
    fi
    
    # Check each test file's dependencies
    jq -r '.dependencies | to_entries[] | "\(.key)|\(.value | join(","))"' "$DEPENDENCY_MAP" | while IFS='|' read -r test_file dependencies; do
        local is_affected=false
        
        # Check if test file itself changed
        if echo "$changed_files" | grep -q "$(basename "$test_file")"; then
            is_affected=true
        fi
        
        # Check if any dependency changed
        if [[ "$is_affected" == false && -n "$dependencies" ]]; then
            IFS=',' read -ra dep_array <<< "$dependencies"
            for dep_file in "${dep_array[@]}"; do
                if echo "$changed_files" | grep -q "$(basename "$dep_file")"; then
                    is_affected=true
                    break
                fi
            done
        fi
        
        if [[ "$is_affected" == true ]]; then
            echo "$test_file"
        fi
    done | sort -u
}

# Record test run results and performance
record_test_run() {
    local test_pattern="$1"
    local duration="$2"
    local exit_code="$3"
    local parallel="$4"
    local log_file="${5:-}"
    
    log_info "Recording test run results..."
    
    local run_id
    run_id="run-$(date +%s)"
    local timestamp
    timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    
    # Update test history
    local temp_history="$CACHE_DIR/temp-history.json"
    jq --arg run_id "$run_id" \
       --arg timestamp "$timestamp" \
       --arg pattern "$test_pattern" \
       --arg duration "$duration" \
       --arg exit_code "$exit_code" \
       --arg parallel "$parallel" \
       '.runs += [{
         id: $run_id,
         timestamp: $timestamp,
         pattern: $pattern,
         duration: ($duration | tonumber),
         exit_code: ($exit_code | tonumber),
         parallel: ($parallel | tonumber)
       }]' "$TEST_HISTORY" > "$temp_history"
    mv "$temp_history" "$TEST_HISTORY"
    
    # Extract individual test timings from log if available
    if [[ -f "$log_file" ]]; then
        extract_test_timings "$log_file"
    fi
    
    # Cleanup old runs (keep last 50)
    jq '.runs |= sort_by(.timestamp) | .runs |= .[-50:]' "$TEST_HISTORY" > "$temp_history"
    mv "$temp_history" "$TEST_HISTORY"
    
    log_success "Test run recorded: $run_id"
}

# Extract individual test timings from test output
extract_test_timings() {
    local log_file="$1"
    
    log_info "Extracting test performance metrics from $log_file"
    
    # Extract test timings using Go test output format
    local test_timings
    test_timings=$(grep -E "(PASS|FAIL).*[0-9]+\.[0-9]+s" "$log_file" | \
                   sed -E 's/.*\s([a-zA-Z_][a-zA-Z0-9_]*)\s+([0-9]+\.[0-9]+s)/\1|\2/' | \
                   head -100 || echo "")
    
    if [[ -n "$test_timings" ]]; then
        local temp_metrics="$CACHE_DIR/temp-metrics.json"
        cp "$PERFORMANCE_METRICS" "$temp_metrics"
        
        echo "$test_timings" | while IFS='|' read -r test_name duration; do
            if [[ -n "$test_name" && -n "$duration" ]]; then
                local duration_sec
                duration_sec=$(echo "$duration" | sed 's/s$//')
                
                # Update test timing with moving average
                jq --arg test "$test_name" \
                   --arg duration "$duration_sec" \
                   --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
                   '.test_timings[$test] = {
                     current: ($duration | tonumber),
                     average: ((.test_timings[$test].average // 0) * 0.8 + ($duration | tonumber) * 0.2),
                     runs: ((.test_timings[$test].runs // 0) + 1),
                     last_run: $timestamp
                   }' "$temp_metrics" > "$temp_metrics.tmp"
                mv "$temp_metrics.tmp" "$temp_metrics"
            fi
        done
        
        # Update last_updated timestamp
        jq --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
           '.last_updated = $timestamp' "$temp_metrics" > "$temp_metrics.tmp"
        mv "$temp_metrics.tmp" "$temp_metrics"
        
        mv "$temp_metrics" "$PERFORMANCE_METRICS"
    fi
}

# Get optimal parallelization for test groups
get_optimal_parallel() {
    local test_pattern="$1"
    local default_parallel="${2:-4}"
    
    # Check historical performance data
    local optimal
    optimal=$(jq -r --arg pattern "$test_pattern" \
              '.parallel_efficiency[$pattern] // empty' "$PERFORMANCE_METRICS" 2>/dev/null || echo "")
    
    if [[ -n "$optimal" ]]; then
        echo "$optimal"
    else
        echo "$default_parallel"
    fi
}

# Check if test results are cached and valid
is_test_cached() {
    local test_pattern="$1"
    local changed_files="$2"
    local cache_max_age="${3:-3600}"  # 1 hour default
    
    log_info "Checking cache validity for: $test_pattern"
    
    # Get last successful run
    local last_run
    last_run=$(jq -r --arg pattern "$test_pattern" \
               '.runs[] | select(.pattern == $pattern and .exit_code == 0) | .timestamp' "$TEST_HISTORY" | \
               tail -1)
    
    if [[ -z "$last_run" || "$last_run" == "null" ]]; then
        log_info "No cached results found"
        return 1
    fi
    
    # Check age
    local last_run_epoch
    local current_epoch
    last_run_epoch=$(date -d "$last_run" +%s 2>/dev/null || echo "0")
    current_epoch=$(date +%s)
    local age=$((current_epoch - last_run_epoch))
    
    if [[ $age -gt $cache_max_age ]]; then
        log_info "Cache expired (age: ${age}s > ${cache_max_age}s)"
        return 1
    fi
    
    # Check if any relevant files changed since last run
    if [[ -n "$changed_files" ]]; then
        local affected_tests
        affected_tests=$(select_affected_tests "$changed_files")
        
        # Check if current pattern matches any affected tests
        if echo "$affected_tests" | grep -q "$test_pattern" || [[ "$test_pattern" == "./..." ]]; then
            log_info "Cache invalid due to file changes affecting tests"
            return 1
        fi
    fi
    
    log_success "Cache valid for: $test_pattern"
    return 0
}

# Smart test selection based on changes and history
smart_test_selection() {
    local base_ref="${1:-HEAD~1}"
    local mode="${2:-changed}"  # changed, critical, full, fast
    
    log_info "Performing smart test selection (mode: $mode)..."
    
    init_cache
    
    local changed_files=""
    if get_changed_files "$base_ref" > /dev/null 2>&1; then
        changed_files=$(get_changed_files "$base_ref")
    fi
    
    case "$mode" in
        "changed")
            if [[ -n "$changed_files" ]]; then
                log_info "Selecting tests affected by changed files..."
                local affected_tests
                affected_tests=$(select_affected_tests "$changed_files")
                
                if [[ -n "$affected_tests" ]]; then
                    # Convert test files to package patterns
                    echo "$affected_tests" | xargs dirname | sort -u | sed 's|^|./|' | sed 's|$|/...|'
                else
                    log_info "No affected tests found, running critical tests"
                    echo "./pkg/context/... ./controllers/... ./api/..."
                fi
            else
                log_info "No changes detected, running critical tests"
                echo "./pkg/context/... ./controllers/... ./api/..."
            fi
            ;;
            
        "critical")
            echo "./pkg/context/... ./pkg/clients/... ./pkg/nephio/... ./controllers/... ./api/..."
            ;;
            
        "fast")
            echo "./pkg/auth/... ./pkg/config/... ./pkg/errors/... ./internal/security/..."
            ;;
            
        "slow")
            # Return slowest tests based on historical data
            jq -r '.test_timings | to_entries | sort_by(.value.average) | reverse | .[0:10] | .[].key' "$PERFORMANCE_METRICS" 2>/dev/null | \
            xargs -I {} dirname {} 2>/dev/null | sort -u | sed 's|^|./|' | sed 's|$|/...|' || echo "./internal/loop/..."
            ;;
            
        "full")
            echo "./..."
            ;;
            
        *)
            log_error "Unknown mode: $mode"
            echo "./pkg/context/... ./controllers/..."
            ;;
    esac
}

# Run tests with caching and smart selection
cached_test_run() {
    local mode="${1:-changed}"
    local base_ref="${2:-HEAD~1}"
    local force_run="${3:-false}"
    
    log_info "Running tests with caching (mode: $mode, force: $force_run)..."
    
    init_cache
    
    local test_patterns
    test_patterns=$(smart_test_selection "$base_ref" "$mode")
    
    local changed_files=""
    if get_changed_files "$base_ref" > /dev/null 2>&1; then
        changed_files=$(get_changed_files "$base_ref")
    fi
    
    log_info "Selected test patterns:"
    echo "$test_patterns"
    
    local total_duration=0
    local patterns_run=0
    local patterns_cached=0
    
    mkdir -p "$TEST_RESULTS_DIR"
    
    echo "$test_patterns" | while read -r pattern; do
        if [[ -z "$pattern" ]]; then
            continue
        fi
        
        log_info "Processing pattern: $pattern"
        
        # Check cache unless forced
        if [[ "$force_run" != "true" ]] && is_test_cached "$pattern" "$changed_files" 1800; then
            log_success "Using cached results for: $pattern"
            ((patterns_cached++))
            continue
        fi
        
        # Run tests
        local start_time
        local end_time
        local duration
        local exit_code
        local log_file
        
        start_time=$(date +%s)
        log_file="$TEST_RESULTS_DIR/cached-run-$(date +%s)-$(echo "$pattern" | sed 's|[/.]||g').log"
        
        log_info "Running tests for: $pattern"
        
        # Get optimal parallelization
        local parallel
        parallel=$(get_optimal_parallel "$pattern" 4)
        
        local test_cmd="go test -v -timeout=8m -parallel=$parallel -shuffle=on $pattern"
        
        if timeout 10m $test_cmd 2>&1 | tee "$log_file"; then
            exit_code=0
            log_success "Tests passed for: $pattern"
        else
            exit_code=$?
            log_error "Tests failed for: $pattern (exit: $exit_code)"
        fi
        
        end_time=$(date +%s)
        duration=$((end_time - start_time))
        total_duration=$((total_duration + duration))
        
        # Record results
        record_test_run "$pattern" "$duration" "$exit_code" "$parallel" "$log_file"
        
        ((patterns_run++))
        
        log_info "Pattern completed in ${duration}s"
    done
    
    log_success "Cached test run completed"
    log_info "Patterns run: $patterns_run, Patterns cached: $patterns_cached"
    log_info "Total duration: ${total_duration}s (~$((total_duration / 60))m $((total_duration % 60))s)"
}

# Generate cache statistics report
generate_cache_report() {
    log_info "Generating cache statistics report..."
    
    if [[ ! -f "$TEST_HISTORY" ]]; then
        log_error "No test history found"
        return 1
    fi
    
    local report_file="$TEST_RESULTS_DIR/cache-report.md"
    
    cat > "$report_file" << 'EOF'
# Smart Test Cache Report

## Cache Statistics
EOF
    
    local total_runs
    local avg_duration
    local cache_hits
    local last_updated
    
    total_runs=$(jq '.runs | length' "$TEST_HISTORY")
    avg_duration=$(jq '.runs | map(.duration) | add / length' "$TEST_HISTORY" 2>/dev/null | cut -d. -f1)
    last_updated=$(jq -r '.last_updated // "unknown"' "$PERFORMANCE_METRICS")
    
    cat >> "$report_file" << EOF
- **Total Test Runs**: $total_runs
- **Average Duration**: ${avg_duration:-0}s
- **Last Updated**: $last_updated

## Performance Insights
EOF
    
    # Top 10 slowest tests
    echo "### Slowest Tests" >> "$report_file"
    jq -r '.test_timings | to_entries | sort_by(.value.average) | reverse | .[0:10] | .[] | "- **\(.key)**: \(.value.average | tostring | split(".")[0])s average"' "$PERFORMANCE_METRICS" >> "$report_file" 2>/dev/null || echo "No timing data available" >> "$report_file"
    
    # Recent runs
    echo "" >> "$report_file"
    echo "### Recent Test Runs" >> "$report_file"
    jq -r '.runs | sort_by(.timestamp) | reverse | .[0:10] | .[] | "- \(.pattern): \(.duration)s (\(.exit_code == 0 | if . then "✅" else "❌" end))"' "$TEST_HISTORY" >> "$report_file" 2>/dev/null || echo "No run data available" >> "$report_file"
    
    log_success "Cache report generated: $report_file"
    
    echo ""
    echo "📊 Cache Statistics:"
    echo "  Total runs: $total_runs"
    echo "  Average duration: ${avg_duration:-0}s"
    echo "  Report: $report_file"
}

# Clean old cache data
clean_cache() {
    local max_age_days="${1:-7}"
    
    log_info "Cleaning cache data older than $max_age_days days..."
    
    # Clean old test history
    local cutoff_date
    cutoff_date=$(date -u -d "$max_age_days days ago" +%Y-%m-%dT%H:%M:%SZ)
    
    local temp_file="$CACHE_DIR/temp-clean.json"
    
    # Clean test history
    jq --arg cutoff "$cutoff_date" \
       '.runs |= map(select(.timestamp > $cutoff))' "$TEST_HISTORY" > "$temp_file"
    mv "$temp_file" "$TEST_HISTORY"
    
    # Clean old log files
    find "$TEST_RESULTS_DIR" -name "cached-run-*.log" -mtime +$max_age_days -delete 2>/dev/null || true
    
    log_success "Cache cleaned"
}

# Main command handler
case "${1:-help}" in
    "init")
        init_cache
        ;;
    "build-deps")
        build_dependency_map
        ;;
    "select")
        smart_test_selection "${2:-HEAD~1}" "${3:-changed}"
        ;;
    "run")
        cached_test_run "${2:-changed}" "${3:-HEAD~1}" "${4:-false}"
        ;;
    "record")
        if [[ $# -ge 4 ]]; then
            record_test_run "$2" "$3" "$4" "${5:-4}" "${6:-}"
        else
            log_error "Usage: $0 record <pattern> <duration> <exit_code> [parallel] [log_file]"
            exit 1
        fi
        ;;
    "report")
        generate_cache_report
        ;;
    "clean")
        clean_cache "${2:-7}"
        ;;
    "status")
        init_cache
        echo "Smart Test Cache Status:"
        if [[ -f "$TEST_HISTORY" ]]; then
            jq -r '"Total runs: " + (.runs | length | tostring)' "$TEST_HISTORY"
        fi
        if [[ -f "$DEPENDENCY_MAP" ]]; then
            jq -r '"Dependencies mapped: " + (.dependencies | keys | length | tostring)' "$DEPENDENCY_MAP"
        fi
        if [[ -f "$PERFORMANCE_METRICS" ]]; then
            jq -r '"Test timings tracked: " + (.test_timings | keys | length | tostring)' "$PERFORMANCE_METRICS"
        fi
        ;;
    "affected")
        base_ref="${2:-HEAD~1}"
        changed_files=$(get_changed_files "$base_ref" || echo "")
        if [[ -n "$changed_files" ]]; then
            select_affected_tests "$changed_files"
        else
            echo "No changes detected"
        fi
        ;;
    "help"|*)
        echo "Nephoran Smart Test Cache and Selection System"
        echo ""
        echo "Usage: $0 <command> [options]"
        echo ""
        echo "Commands:"
        echo "  init                      - Initialize cache system"
        echo "  build-deps               - Build test dependency map"
        echo "  select [base] [mode]     - Smart test selection (changed/critical/fast/slow/full)"
        echo "  run [mode] [base] [force] - Run tests with caching"
        echo "  record <pattern> <duration> <exit> [parallel] [log] - Record test results"
        echo "  report                   - Generate cache statistics report"
        echo "  clean [days]             - Clean cache data older than N days"
        echo "  status                   - Show cache status"
        echo "  affected [base]          - Show tests affected by changes"
        echo "  help                     - Show this help"
        echo ""
        echo "Examples:"
        echo "  $0 init                              # Initialize cache system"
        echo "  $0 select HEAD~5 changed             # Select tests affected by last 5 commits"
        echo "  $0 run changed HEAD~1 false          # Run changed tests with caching"
        echo "  $0 affected HEAD~3                   # Show tests affected since 3 commits ago"
        echo ""
        echo "Modes:"
        echo "  changed  - Tests affected by file changes (default)"
        echo "  critical - Critical component tests (fast)"
        echo "  fast     - Fastest tests only"
        echo "  slow     - Slowest tests (for optimization)"
        echo "  full     - All tests"
        ;;
esac