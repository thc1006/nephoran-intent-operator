#!/bin/bash

# =============================================================================
# CI Performance Monitoring Script
# =============================================================================
# Monitors and reports CI pipeline performance metrics
# Usage: ./monitor-ci-performance.sh [--export-metrics] [--alert-webhook URL]
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Configuration
METRICS_FILE="${ROOT_DIR}/.ci-metrics.json"
PERFORMANCE_LOG="${ROOT_DIR}/.ci-performance.log"
ALERT_WEBHOOK=""
EXPORT_METRICS=false

# Performance thresholds (in seconds)
MAX_BUILD_TIME=300      # 5 minutes
MAX_TEST_TIME=1200      # 20 minutes  
MAX_TOTAL_TIME=1800     # 30 minutes

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1" >> "$PERFORMANCE_LOG"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
    echo "[ERROR] $1" >> "$PERFORMANCE_LOG"
}

warn() {
    echo -e "${YELLOW}[WARN] $1${NC}"
    echo "[WARN] $1" >> "$PERFORMANCE_LOG"
}

success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
    echo "[SUCCESS] $1" >> "$PERFORMANCE_LOG"
}

# Get current timestamp in seconds
get_timestamp() {
    date +%s
}

# Calculate duration between two timestamps
calculate_duration() {
    local start=$1
    local end=$2
    echo $((end - start))
}

# Format duration for display
format_duration() {
    local duration=$1
    local minutes=$((duration / 60))
    local seconds=$((duration % 60))
    printf "%dm %ds" $minutes $seconds
}

# Initialize metrics tracking
init_metrics() {
    local start_time=$(get_timestamp)
    
    cat > "$METRICS_FILE" <<EOF
{
  "pipeline": {
    "start_time": $start_time,
    "end_time": null,
    "total_duration": null,
    "status": "running"
  },
  "jobs": {},
  "system": {
    "go_version": "$(go version 2>/dev/null || echo 'unknown')",
    "runner_os": "${RUNNER_OS:-unknown}",
    "github_ref": "${GITHUB_REF:-unknown}",
    "github_sha": "${GITHUB_SHA:-unknown}"
  },
  "thresholds": {
    "max_build_time": $MAX_BUILD_TIME,
    "max_test_time": $MAX_TEST_TIME,
    "max_total_time": $MAX_TOTAL_TIME
  }
}
EOF
    
    log "Metrics tracking initialized"
}

# Start job tracking
start_job() {
    local job_name=$1
    local start_time=$(get_timestamp)
    
    # Update metrics file using jq if available, otherwise use sed
    if command -v jq >/dev/null 2>&1; then
        jq --arg job "$job_name" --arg start "$start_time" \
           '.jobs[$job] = {start_time: ($start | tonumber), end_time: null, duration: null, status: "running"}' \
           "$METRICS_FILE" > "$METRICS_FILE.tmp" && mv "$METRICS_FILE.tmp" "$METRICS_FILE"
    else
        # Fallback without jq
        echo "Warning: jq not available, using basic tracking"
    fi
    
    log "Started tracking job: $job_name"
}

# End job tracking
end_job() {
    local job_name=$1
    local status=${2:-"success"}
    local end_time=$(get_timestamp)
    
    if command -v jq >/dev/null 2>&1; then
        # Get start time and calculate duration
        local start_time=$(jq -r --arg job "$job_name" '.jobs[$job].start_time // empty' "$METRICS_FILE")
        if [[ -n "$start_time" ]]; then
            local duration=$((end_time - start_time))
            jq --arg job "$job_name" --arg end "$end_time" --arg duration "$duration" --arg status "$status" \
               '.jobs[$job].end_time = ($end | tonumber) | .jobs[$job].duration = ($duration | tonumber) | .jobs[$job].status = $status' \
               "$METRICS_FILE" > "$METRICS_FILE.tmp" && mv "$METRICS_FILE.tmp" "$METRICS_FILE"
            
            log "Job $job_name completed in $(format_duration $duration) with status: $status"
            
            # Check thresholds
            check_job_threshold "$job_name" "$duration"
        fi
    fi
}

# Check if job exceeded threshold
check_job_threshold() {
    local job_name=$1
    local duration=$2
    
    case $job_name in
        "build")
            if [[ $duration -gt $MAX_BUILD_TIME ]]; then
                warn "Build job exceeded threshold: $(format_duration $duration) > $(format_duration $MAX_BUILD_TIME)"
                send_alert "Build Performance Alert" "Build job took $(format_duration $duration), exceeding threshold of $(format_duration $MAX_BUILD_TIME)"
            fi
            ;;
        "test")
            if [[ $duration -gt $MAX_TEST_TIME ]]; then
                warn "Test job exceeded threshold: $(format_duration $duration) > $(format_duration $MAX_TEST_TIME)"
                send_alert "Test Performance Alert" "Test job took $(format_duration $duration), exceeding threshold of $(format_duration $MAX_TEST_TIME)"
            fi
            ;;
    esac
}

# Finalize pipeline metrics
finalize_metrics() {
    local end_time=$(get_timestamp)
    local status=${1:-"success"}
    
    if command -v jq >/dev/null 2>&1; then
        local start_time=$(jq -r '.pipeline.start_time' "$METRICS_FILE")
        local total_duration=$((end_time - start_time))
        
        jq --arg end "$end_time" --arg duration "$total_duration" --arg status "$status" \
           '.pipeline.end_time = ($end | tonumber) | .pipeline.total_duration = ($duration | tonumber) | .pipeline.status = $status' \
           "$METRICS_FILE" > "$METRICS_FILE.tmp" && mv "$METRICS_FILE.tmp" "$METRICS_FILE"
        
        log "Pipeline completed in $(format_duration $total_duration) with status: $status"
        
        # Check total time threshold
        if [[ $total_duration -gt $MAX_TOTAL_TIME ]]; then
            warn "Pipeline exceeded total time threshold: $(format_duration $total_duration) > $(format_duration $MAX_TOTAL_TIME)"
            send_alert "Pipeline Performance Alert" "Pipeline took $(format_duration $total_duration), exceeding threshold of $(format_duration $MAX_TOTAL_TIME)"
        fi
        
        # Generate summary
        generate_summary
    fi
}

# Generate performance summary
generate_summary() {
    log "Generating performance summary..."
    
    if command -v jq >/dev/null 2>&1; then
        echo ""
        echo "================================="
        echo "CI Performance Summary"
        echo "================================="
        
        local total_duration=$(jq -r '.pipeline.total_duration' "$METRICS_FILE")
        local status=$(jq -r '.pipeline.status' "$METRICS_FILE")
        
        echo "Total Duration: $(format_duration $total_duration)"
        echo "Pipeline Status: $status"
        echo ""
        echo "Job Breakdown:"
        
        jq -r '.jobs | to_entries[] | "\(.key): \(.value.duration)s (\(.value.status))"' "$METRICS_FILE" | while read -r line; do
            echo "  $line"
        done
        
        echo ""
        
        # Performance indicators
        if [[ $total_duration -lt 600 ]]; then  # < 10 minutes
            success "Excellent performance: Pipeline completed in under 10 minutes"
        elif [[ $total_duration -lt 1200 ]]; then  # < 20 minutes
            echo "Good performance: Pipeline completed in reasonable time"
        else
            warn "Poor performance: Pipeline took longer than expected"
        fi
        
        # Save summary to GitHub step summary if in CI
        if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
            {
                echo "## 📊 CI Performance Summary"
                echo ""
                echo "| Metric | Value |"
                echo "|--------|-------|"
                echo "| Total Duration | $(format_duration $total_duration) |"
                echo "| Status | $status |"
                echo ""
                echo "### Job Breakdown"
                echo "| Job | Duration | Status |"
                echo "|-----|----------|--------|"
                jq -r '.jobs | to_entries[] | "| \(.key) | \(.value.duration)s | \(.value.status) |"' "$METRICS_FILE"
            } >> "$GITHUB_STEP_SUMMARY"
        fi
    fi
}

# Send alert webhook
send_alert() {
    local title=$1
    local message=$2
    
    if [[ -n "$ALERT_WEBHOOK" ]]; then
        log "Sending alert: $title"
        curl -s -X POST "$ALERT_WEBHOOK" \
             -H "Content-Type: application/json" \
             -d "{\"title\": \"$title\", \"message\": \"$message\", \"timestamp\": \"$(date -Iseconds)\"}" \
             || warn "Failed to send alert webhook"
    fi
}

# Export metrics for analysis
export_metrics() {
    if [[ "$EXPORT_METRICS" == true ]]; then
        local export_file="ci-metrics-$(date +%Y%m%d-%H%M%S).json"
        cp "$METRICS_FILE" "$export_file"
        log "Metrics exported to: $export_file"
        
        # Also create a CSV export for easy analysis
        if command -v jq >/dev/null 2>&1; then
            local csv_file="ci-metrics-$(date +%Y%m%d-%H%M%S).csv"
            {
                echo "job,duration,status,timestamp"
                jq -r '.jobs | to_entries[] | "\(.key),\(.value.duration),\(.value.status),\(.value.end_time)"' "$METRICS_FILE"
            } > "$csv_file"
            log "CSV metrics exported to: $csv_file"
        fi
    fi
}

# Main execution
main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --export-metrics)
                EXPORT_METRICS=true
                shift
                ;;
            --alert-webhook)
                ALERT_WEBHOOK="$2"
                shift 2
                ;;
            --init)
                init_metrics
                exit 0
                ;;
            --start-job)
                start_job "$2"
                shift 2
                ;;
            --end-job)
                end_job "$2" "${3:-success}"
                shift 2
                [[ $# -gt 0 ]] && shift
                ;;
            --finalize)
                finalize_metrics "${2:-success}"
                shift
                [[ $# -gt 0 ]] && shift
                ;;
            --summary)
                generate_summary
                exit 0
                ;;
            --help|-h)
                echo "Usage: $0 [options]"
                echo ""
                echo "Options:"
                echo "  --init                    Initialize metrics tracking"
                echo "  --start-job JOB_NAME      Start tracking a job"
                echo "  --end-job JOB_NAME [STATUS] End tracking a job"
                echo "  --finalize [STATUS]       Finalize pipeline metrics"
                echo "  --summary                 Show performance summary"
                echo "  --export-metrics          Export metrics to files"
                echo "  --alert-webhook URL       Send alerts to webhook"
                echo "  --help                    Show this help"
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Default action: show summary if metrics exist
    if [[ -f "$METRICS_FILE" ]]; then
        generate_summary
        export_metrics
    else
        warn "No metrics file found. Use --init to start tracking."
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi