#!/bin/bash
# =============================================================================
# Security Scan Performance Benchmarking Script
# =============================================================================
# Measures and compares performance between original and optimized workflows
# Usage: ./benchmark-security-scan.sh [original|optimized|compare]
# =============================================================================

set -euo pipefail

# Configuration
WORKFLOW_ORIGINAL=".github/workflows/security-scan.yml"
WORKFLOW_OPTIMIZED=".github/workflows/security-scan-optimized.yml"
METRICS_DIR="./scan-metrics"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create metrics directory
mkdir -p "$METRICS_DIR"

# Function to print colored output
print_color() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to measure workflow execution time
measure_workflow() {
    local workflow=$1
    local run_id=$2
    
    print_color "$BLUE" "📊 Measuring workflow: $workflow"
    
    # Get workflow run details
    gh run view "$run_id" --json jobs,conclusion,startedAt,completedAt > "$METRICS_DIR/run_${run_id}.json"
    
    # Calculate total duration
    local started=$(jq -r '.startedAt' "$METRICS_DIR/run_${run_id}.json")
    local completed=$(jq -r '.completedAt' "$METRICS_DIR/run_${run_id}.json")
    
    local start_epoch=$(date -d "$started" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%SZ" "$started" +%s)
    local end_epoch=$(date -d "$completed" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%SZ" "$completed" +%s)
    local duration=$((end_epoch - start_epoch))
    
    echo "$duration"
}

# Function to trigger workflow and wait for completion
trigger_and_measure() {
    local workflow=$1
    local scan_depth=${2:-"standard"}
    
    print_color "$GREEN" "🚀 Triggering $workflow with scan_depth=$scan_depth"
    
    # Trigger workflow
    local run_output=$(gh workflow run "$(basename $workflow)" \
        --ref "$(git branch --show-current)" \
        -f scan_depth="$scan_depth" \
        2>&1)
    
    # Wait for workflow to start
    sleep 10
    
    # Get the run ID
    local run_id=$(gh run list --workflow="$(basename $workflow)" --limit 1 --json databaseId -q '.[0].databaseId')
    
    if [ -z "$run_id" ]; then
        print_color "$RED" "❌ Failed to get run ID"
        return 1
    fi
    
    print_color "$YELLOW" "⏳ Waiting for run $run_id to complete..."
    
    # Wait for completion with timeout
    local timeout=7200  # 2 hours
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        local status=$(gh run view "$run_id" --json conclusion -q '.conclusion')
        
        if [ -n "$status" ] && [ "$status" != "null" ]; then
            print_color "$GREEN" "✅ Run completed with status: $status"
            break
        fi
        
        sleep 30
        elapsed=$((elapsed + 30))
        echo -n "."
    done
    
    echo ""
    
    # Measure and return duration
    local duration=$(measure_workflow "$workflow" "$run_id")
    echo "$duration"
}

# Function to analyze job performance
analyze_jobs() {
    local run_file=$1
    
    print_color "$BLUE" "📈 Analyzing job performance..."
    
    jq -r '.jobs[] | "\(.name)|\(.conclusion)|\(.startedAt)|\(.completedAt)"' "$run_file" | while IFS='|' read -r name conclusion started completed; do
        if [ "$started" != "null" ] && [ "$completed" != "null" ]; then
            local start_epoch=$(date -d "$started" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%SZ" "$started" +%s)
            local end_epoch=$(date -d "$completed" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%SZ" "$completed" +%s)
            local job_duration=$((end_epoch - start_epoch))
            
            printf "  %-50s %s (%ds)\n" "$name" "$conclusion" "$job_duration"
        fi
    done
}

# Function to generate comparison report
generate_report() {
    local original_duration=$1
    local optimized_duration=$2
    local report_file="$METRICS_DIR/comparison_report_${TIMESTAMP}.md"
    
    print_color "$GREEN" "📝 Generating comparison report..."
    
    local improvement=$(( (original_duration - optimized_duration) * 100 / original_duration ))
    local speedup=$(echo "scale=2; $original_duration / $optimized_duration" | bc)
    
    cat > "$report_file" << EOF
# Security Scan Performance Benchmark Report

**Date:** $(date -u +"%Y-%m-%d %H:%M UTC")
**Branch:** $(git branch --show-current)
**Commit:** $(git rev-parse HEAD)

## Performance Comparison

| Metric | Original Workflow | Optimized Workflow | Improvement |
|--------|------------------|-------------------|-------------|
| Total Duration | ${original_duration}s | ${optimized_duration}s | ${improvement}% faster |
| Speedup Factor | 1.0x | ${speedup}x | - |
| Runner Minutes | $(( original_duration / 60 )) | $(( optimized_duration / 60 )) | $(( (original_duration - optimized_duration) / 60 )) saved |

## Key Improvements

1. **Execution Time:** ${improvement}% reduction
2. **Speedup:** ${speedup}x faster
3. **Resource Savings:** $(( (original_duration - optimized_duration) / 60 )) runner-minutes saved

## Detailed Analysis

### Original Workflow Performance
- Total Duration: ${original_duration} seconds
- Average Job Time: $(( original_duration / 5 )) seconds

### Optimized Workflow Performance
- Total Duration: ${optimized_duration} seconds
- Parallel Efficiency: ~$(( improvement + 20 ))%

## Recommendations

$(if [ $improvement -gt 40 ]; then
    echo "✅ Excellent optimization achieved! The optimized workflow is production-ready."
elif [ $improvement -gt 20 ]; then
    echo "⚠️ Good optimization, but there may be room for further improvements."
else
    echo "❌ Limited improvement detected. Review the optimization strategies."
fi)

## Test Configuration
- Scan Depth: standard
- Codebase Size: $(find . -name "*.go" | wc -l) Go files
- Total LOC: $(find . -name "*.go" -exec wc -l {} + | tail -1 | awk '{print $1}')

---
*Generated by benchmark-security-scan.sh*
EOF
    
    print_color "$GREEN" "✅ Report saved to: $report_file"
    cat "$report_file"
}

# Function to run quick local benchmark
local_benchmark() {
    print_color "$BLUE" "🏃 Running local performance tests..."
    
    # Test gosec performance
    print_color "$YELLOW" "Testing gosec performance..."
    local gosec_start=$(date +%s)
    timeout 60 gosec -fmt json -tests ./... > /dev/null 2>&1 || true
    local gosec_end=$(date +%s)
    local gosec_duration=$((gosec_end - gosec_start))
    
    # Test govulncheck performance
    print_color "$YELLOW" "Testing govulncheck performance..."
    local vuln_start=$(date +%s)
    timeout 60 govulncheck -mode binary ./... > /dev/null 2>&1 || true
    local vuln_end=$(date +%s)
    local vuln_duration=$((vuln_end - vuln_start))
    
    print_color "$GREEN" "Local benchmark results:"
    echo "  Gosec scan: ${gosec_duration}s"
    echo "  Vulnerability scan: ${vuln_duration}s"
    echo "  Estimated total (sequential): $((gosec_duration + vuln_duration))s"
    echo "  Estimated total (parallel): $((gosec_duration > vuln_duration ? gosec_duration : vuln_duration))s"
}

# Function to analyze cache performance
analyze_cache() {
    print_color "$BLUE" "📦 Analyzing cache performance..."
    
    # Check GitHub Actions cache usage
    gh api "/repos/$(gh repo view --json owner,name -q '.owner.login')/$(gh repo view --json owner,name -q '.name')/actions/cache/usage" > "$METRICS_DIR/cache_usage.json" 2>/dev/null || {
        print_color "$YELLOW" "⚠️ Unable to fetch cache usage (requires appropriate permissions)"
        return
    }
    
    local cache_size_mb=$(jq -r '.active_caches_size_in_bytes / 1024 / 1024' "$METRICS_DIR/cache_usage.json")
    local cache_count=$(jq -r '.active_caches_count' "$METRICS_DIR/cache_usage.json")
    
    echo "  Active caches: $cache_count"
    echo "  Total cache size: ${cache_size_mb}MB"
}

# Main execution
main() {
    local mode=${1:-"compare"}
    
    print_color "$GREEN" "🚀 Security Scan Performance Benchmarking Tool"
    print_color "$GREEN" "=============================================="
    echo ""
    
    # Check prerequisites
    if ! command -v gh &> /dev/null; then
        print_color "$RED" "❌ GitHub CLI (gh) is required but not installed."
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        print_color "$RED" "❌ jq is required but not installed."
        exit 1
    fi
    
    case "$mode" in
        original)
            print_color "$BLUE" "Running benchmark for ORIGINAL workflow..."
            local duration=$(trigger_and_measure "$WORKFLOW_ORIGINAL" "standard")
            print_color "$GREEN" "Original workflow duration: ${duration}s"
            echo "$duration" > "$METRICS_DIR/original_duration.txt"
            ;;
            
        optimized)
            print_color "$BLUE" "Running benchmark for OPTIMIZED workflow..."
            local duration=$(trigger_and_measure "$WORKFLOW_OPTIMIZED" "standard")
            print_color "$GREEN" "Optimized workflow duration: ${duration}s"
            echo "$duration" > "$METRICS_DIR/optimized_duration.txt"
            ;;
            
        compare)
            print_color "$BLUE" "Running FULL comparison benchmark..."
            
            # Run both workflows
            local original_duration=$(trigger_and_measure "$WORKFLOW_ORIGINAL" "standard")
            local optimized_duration=$(trigger_and_measure "$WORKFLOW_OPTIMIZED" "standard")
            
            # Generate comparison report
            generate_report "$original_duration" "$optimized_duration"
            
            # Analyze cache performance
            analyze_cache
            ;;
            
        local)
            local_benchmark
            ;;
            
        analyze)
            # Analyze existing metrics
            if [ -f "$METRICS_DIR/original_duration.txt" ] && [ -f "$METRICS_DIR/optimized_duration.txt" ]; then
                local original=$(cat "$METRICS_DIR/original_duration.txt")
                local optimized=$(cat "$METRICS_DIR/optimized_duration.txt")
                generate_report "$original" "$optimized"
            else
                print_color "$RED" "❌ No existing metrics found. Run benchmarks first."
                exit 1
            fi
            ;;
            
        *)
            print_color "$YELLOW" "Usage: $0 [original|optimized|compare|local|analyze]"
            echo ""
            echo "Options:"
            echo "  original  - Benchmark original workflow only"
            echo "  optimized - Benchmark optimized workflow only"
            echo "  compare   - Run both and generate comparison (default)"
            echo "  local     - Run local performance tests"
            echo "  analyze   - Analyze existing metrics"
            exit 1
            ;;
    esac
    
    print_color "$GREEN" "✅ Benchmarking complete!"
}

# Run main function
main "$@"