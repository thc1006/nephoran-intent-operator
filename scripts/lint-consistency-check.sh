#!/bin/bash
# Consistency check script to ensure linter rules pass reliably across multiple runs
# Usage: ./lint-consistency-check.sh [options]

set -euo pipefail

# Configuration
TARGET_PATH="${TARGET_PATH:-.}"
CONFIG_FILE="${CONFIG_FILE:-./.golangci.yml}"
NUM_RUNS="${NUM_RUNS:-5}"
PARALLEL_RUNS="${PARALLEL_RUNS:-2}"
TIMEOUT="${TIMEOUT:-120}"
TEMP_DIR="/tmp/lint-consistency-$$"
VERBOSE="${VERBOSE:-false}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
GRAY='\033[0;37m'
BOLD='\033[1m'
NC='\033[0m'

log() {
    echo -e "${GRAY}[$(date +'%H:%M:%S')] $1${NC}" >&2
}

log_info() {
    echo -e "${CYAN}[INFO] $1${NC}" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}" >&2
}

log_error() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING] $1${NC}" >&2
}

cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

usage() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Environment variables:"
    echo "  TARGET_PATH=path      Target path to lint (default: .)"
    echo "  CONFIG_FILE=file      Golangci-lint config file (default: ./.golangci.yml)"
    echo "  NUM_RUNS=N           Number of consistency runs (default: 5)"
    echo "  PARALLEL_RUNS=N      Number of parallel runs (default: 2)" 
    echo "  TIMEOUT=N            Timeout per run in seconds (default: 120)"
    echo "  VERBOSE=true         Enable verbose output"
    echo ""
    echo "Examples:"
    echo "  NUM_RUNS=10 $0"
    echo "  TARGET_PATH=./pkg VERBOSE=true $0"
    exit 1
}

# Check prerequisites
check_prerequisites() {
    local missing=()
    
    if ! command -v golangci-lint >/dev/null 2>&1; then
        missing+=("golangci-lint")
    fi
    
    if ! command -v go >/dev/null 2>&1; then
        missing+=("go")
    fi
    
    if ! command -v jq >/dev/null 2>&1; then
        missing+=("jq")
    fi
    
    if [[ ! -f "$CONFIG_FILE" ]]; then
        missing+=("config file at $CONFIG_FILE")
    fi
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing prerequisites:"
        printf '%s\n' "${missing[@]}" | sed 's/^/  - /'
        return 1
    fi
    
    return 0
}

# Get enabled linters from config
get_enabled_linters() {
    local config_file="$1"
    
    if command -v yq >/dev/null 2>&1; then
        yq eval '.linters.enable[]' "$config_file" 2>/dev/null || echo "revive staticcheck govet ineffassign errcheck gocritic misspell unparam unconvert prealloc gosec" | tr ' ' '\n'
    else
        # Fallback: simple grep-based parsing
        awk '/enable:/{flag=1; next} /^[[:space:]]*-/{if(flag) print $2} /^[^[:space:]-]/{flag=0}' "$config_file" 2>/dev/null || {
            echo "revive staticcheck govet ineffassign errcheck gocritic misspell unparam unconvert prealloc gosec" | tr ' ' '\n'
        }
    fi
}

# Run single lint check
run_single_check() {
    local run_id="$1"
    local linter="$2"
    local output_file="$3"
    
    local start_time
    start_time=$(date +%s.%N)
    
    local cmd=(
        golangci-lint run
        --disable-all
        --enable "$linter"
        --timeout "${TIMEOUT}s"
        --out-format json
        --no-cache
        "$TARGET_PATH"
    )
    
    if [[ "$VERBOSE" == "true" ]]; then
        log "Run $run_id: ${cmd[*]}"
    fi
    
    local exit_code=0
    if "${cmd[@]}" > "$output_file" 2>&1; then
        exit_code=0
    else
        exit_code=$?
    fi
    
    local end_time
    end_time=$(date +%s.%N)
    local duration
    duration=$(echo "$end_time - $start_time" | bc -l)
    
    # Create result JSON
    local result_json
    result_json=$(jq -n \
        --arg run_id "$run_id" \
        --arg linter "$linter" \
        --argjson exit_code "$exit_code" \
        --argjson duration "$duration" \
        --argjson start_time "$start_time" \
        '{
            run_id: $run_id,
            linter: $linter,
            exit_code: $exit_code,
            duration: $duration,
            start_time: $start_time,
            success: ($exit_code == 0)
        }')
    
    echo "$result_json" > "${output_file}.result"
    return $exit_code
}

# Analyze consistency results
analyze_consistency() {
    local results_dir="$1"
    local linter="$2"
    
    local results_files
    mapfile -t results_files < <(find "$results_dir" -name "*_${linter}.result" | sort)
    
    if [[ ${#results_files[@]} -eq 0 ]]; then
        log_warning "No results found for linter: $linter"
        return 1
    fi
    
    # Combine all results
    local combined_results="$TEMP_DIR/combined_${linter}.json"
    jq -s '.' "${results_files[@]}" > "$combined_results"
    
    # Calculate statistics
    local total_runs
    local successful_runs
    local failed_runs
    local consistency_rate
    local avg_duration
    local min_duration
    local max_duration
    local duration_variance
    
    total_runs=$(jq 'length' "$combined_results")
    successful_runs=$(jq '[.[] | select(.success == true)] | length' "$combined_results")
    failed_runs=$(jq '[.[] | select(.success == false)] | length' "$combined_results")
    consistency_rate=$(echo "scale=2; $successful_runs * 100 / $total_runs" | bc -l)
    avg_duration=$(jq '[.[].duration] | add / length' "$combined_results")
    min_duration=$(jq '[.[].duration] | min' "$combined_results")
    max_duration=$(jq '[.[].duration] | max' "$combined_results")
    
    # Calculate variance
    duration_variance=$(jq --argjson avg "$avg_duration" '[.[].duration] | map(. - $avg) | map(. * .) | add / length | sqrt' "$combined_results")
    
    # Create analysis result
    local analysis
    analysis=$(jq -n \
        --arg linter "$linter" \
        --argjson total "$total_runs" \
        --argjson successful "$successful_runs" \
        --argjson failed "$failed_runs" \
        --argjson consistency_rate "$consistency_rate" \
        --argjson avg_duration "$avg_duration" \
        --argjson min_duration "$min_duration" \
        --argjson max_duration "$max_duration" \
        --argjson duration_variance "$duration_variance" \
        '{
            linter: $linter,
            runs: {
                total: $total,
                successful: $successful,
                failed: $failed
            },
            consistency: {
                rate: $consistency_rate,
                reliable: ($consistency_rate >= 95)
            },
            performance: {
                avg_duration: $avg_duration,
                min_duration: $min_duration,
                max_duration: $max_duration,
                variance: $duration_variance,
                stable: ($duration_variance < 1.0)
            }
        }')
    
    echo "$analysis" > "$TEMP_DIR/analysis_${linter}.json"
    
    # Output human-readable summary
    if (( $(echo "$consistency_rate >= 95" | bc -l) )); then
        log_success "$linter: $consistency_rate% consistent (${successful_runs}/${total_runs})"
    else
        log_error "$linter: $consistency_rate% consistent (${successful_runs}/${total_runs})"
    fi
    
    if [[ "$VERBOSE" == "true" ]]; then
        printf "  Duration: avg=%.2fs, min=%.2fs, max=%.2fs, variance=%.2f\n" \
            "$avg_duration" "$min_duration" "$max_duration" "$duration_variance"
    fi
    
    return $([ "$(echo "$consistency_rate >= 95" | bc -l)" == "1" ])
}

# Main execution
mkdir -p "$TEMP_DIR"

log_info "Golangci-lint Consistency Check"
log_info "Target: $TARGET_PATH"
log_info "Config: $CONFIG_FILE"
log_info "Runs per linter: $NUM_RUNS"
log_info "Parallel runs: $PARALLEL_RUNS"
log_info "Timeout: ${TIMEOUT}s"
echo ""

# Check prerequisites
if ! check_prerequisites; then
    exit 1
fi

# Get enabled linters
log_info "Reading linter configuration..."
mapfile -t LINTERS < <(get_enabled_linters "$CONFIG_FILE")

if [[ ${#LINTERS[@]} -eq 0 ]]; then
    log_error "No linters found in configuration"
    exit 1
fi

log_info "Found ${#LINTERS[@]} enabled linters"
if [[ "$VERBOSE" == "true" ]]; then
    printf '%s\n' "${LINTERS[@]}" | sed 's/^/  - /'
fi
echo ""

# Run consistency checks
log_info "Running consistency checks..."
results_dir="$TEMP_DIR/results"
mkdir -p "$results_dir"

for linter in "${LINTERS[@]}"; do
    log_info "Testing $linter consistency (${NUM_RUNS} runs)..."
    
    # Run multiple checks for this linter
    for ((run=1; run<=NUM_RUNS; run++)); do
        run_id=$(printf "run_%02d" "$run")
        output_file="$results_dir/${run_id}_${linter}.out"
        
        if [[ "$VERBOSE" == "true" ]]; then
            log "  $linter run $run/$NUM_RUNS"
        fi
        
        run_single_check "$run_id" "$linter" "$output_file" &
        
        # Limit parallel runs
        if (( run % PARALLEL_RUNS == 0 )); then
            wait
        fi
    done
    
    # Wait for remaining jobs
    wait
done

# Analyze results
log_info "Analyzing consistency results..."
analyses=()
overall_reliable=true

for linter in "${LINTERS[@]}"; do
    if analyze_consistency "$results_dir" "$linter"; then
        analyses+=("$TEMP_DIR/analysis_${linter}.json")
    else
        overall_reliable=false
    fi
done

# Generate final report
log_info "Generating final report..."

final_report="$TEMP_DIR/consistency_report.json"
jq -s '{
    timestamp: (now | strftime("%Y-%m-%dT%H:%M:%SZ")),
    configuration: {
        target_path: "'"$TARGET_PATH"'",
        config_file: "'"$CONFIG_FILE"'",
        num_runs: '"$NUM_RUNS"',
        parallel_runs: '"$PARALLEL_RUNS"',
        timeout: '"$TIMEOUT"'
    },
    summary: {
        total_linters: length,
        reliable_linters: [.[] | select(.consistency.reliable == true)] | length,
        unreliable_linters: [.[] | select(.consistency.reliable == false)] | length,
        overall_reliable: ([.[] | select(.consistency.reliable == false)] | length == 0)
    },
    linter_analyses: .
}' "${analyses[@]}" > "$final_report"

# Summary output
echo ""
log_info "CONSISTENCY CHECK SUMMARY"
echo "========================="

total_linters=${#LINTERS[@]}
reliable_count=$(jq '.summary.reliable_linters' "$final_report")
unreliable_count=$(jq '.summary.unreliable_linters' "$final_report")

echo "Total linters tested: $total_linters"
echo "Reliable linters: $reliable_count"
echo "Unreliable linters: $unreliable_count"

if [[ "$unreliable_count" -gt 0 ]]; then
    echo ""
    log_warning "Unreliable linters:"
    jq -r '.linter_analyses[] | select(.consistency.reliable == false) | "  - " + .linter + " (" + (.consistency.rate|tostring) + "% consistent)"' "$final_report"
fi

# Save results
output_file="./test-results/lint-consistency-$(date +%Y%m%d_%H%M%S).json"
mkdir -p "$(dirname "$output_file")"
cp "$final_report" "$output_file"
log_info "Results saved to: $output_file"

# Exit with appropriate code
if [[ "$overall_reliable" == "true" ]]; then
    log_success "All linters are consistent!"
    exit 0
else
    log_error "Some linters are inconsistent!"
    exit 1
fi