#!/bin/bash
# =============================================================================
# CI/CD Performance Monitoring & Analytics Script
# =============================================================================
# Purpose: Monitor build performance, predict bottlenecks, optimize resource usage
# Features: Real-time metrics, ML prediction, automated optimization suggestions
# =============================================================================

set -euo pipefail

# Configuration
MONITOR_INTERVAL=${MONITOR_INTERVAL:-5}
PREDICTION_WINDOW=${PREDICTION_WINDOW:-60}
ALERT_THRESHOLD_CPU=${ALERT_THRESHOLD_CPU:-85}
ALERT_THRESHOLD_MEMORY=${ALERT_THRESHOLD_MEMORY:-90}
ALERT_THRESHOLD_DISK=${ALERT_THRESHOLD_DISK:-95}
METRICS_DIR="${PROJECT_ROOT:-$(pwd)}/.performance-metrics"
LOG_FILE="$METRICS_DIR/performance-$(date +%Y%m%d-%H%M%S).log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m'

# =============================================================================
# Utility Functions
# =============================================================================

log() {
    local level="$1"
    shift
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "[$timestamp] [$level] $*" | tee -a "$LOG_FILE"
}

log_info() { log "INFO" "$@"; }
log_warn() { log "WARN" "$@"; }
log_error() { log "ERROR" "$@"; }
log_success() { log "SUCCESS" "$@"; }

# Detect system type and capabilities
detect_system() {
    local os_type=$(uname -s)
    local cpu_count=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo "4")
    local total_mem=$(free -h 2>/dev/null | awk '/^Mem:/ {print $2}' || echo "8G")
    
    echo "os_type=$os_type"
    echo "cpu_count=$cpu_count"
    echo "total_memory=$total_mem"
}

# Get system performance metrics
get_system_metrics() {
    local timestamp=$(date +%s)
    
    # CPU usage
    local cpu_usage
    if command -v top >/dev/null 2>&1; then
        cpu_usage=$(top -bn1 | grep "Cpu(s)" | sed "s/.*, *\([0-9.]*\)%* id.*/\1/" | awk '{print 100 - $1}' 2>/dev/null || echo "0")
    else
        cpu_usage="0"
    fi
    
    # Memory usage
    local mem_usage=0
    local mem_total=0
    local mem_used=0
    if command -v free >/dev/null 2>&1; then
        local mem_info=$(free -m | awk 'NR==2{printf "%.2f %.2f %.2f", $3*100/$2, $3, $2}')
        mem_usage=$(echo "$mem_info" | awk '{print $1}')
        mem_used=$(echo "$mem_info" | awk '{print $2}')
        mem_total=$(echo "$mem_info" | awk '{print $3}')
    fi
    
    # Disk usage
    local disk_usage=0
    if command -v df >/dev/null 2>&1; then
        disk_usage=$(df -h . | awk 'NR==2 {print $5}' | sed 's/%//')
    fi
    
    # Load average
    local load_avg
    if [[ -f "/proc/loadavg" ]]; then
        load_avg=$(cut -d' ' -f1-3 /proc/loadavg)
    else
        load_avg="0.00 0.00 0.00"
    fi
    
    # Network I/O (if available)
    local net_rx=0
    local net_tx=0
    if [[ -f "/proc/net/dev" ]]; then
        local net_info=$(awk '/eth0|ens|enp/ {rx+=$2; tx+=$10} END {print rx, tx}' /proc/net/dev 2>/dev/null || echo "0 0")
        net_rx=$(echo "$net_info" | awk '{print $1}')
        net_tx=$(echo "$net_info" | awk '{print $2}')
    fi
    
    # Disk I/O
    local disk_read=0
    local disk_write=0
    if command -v iostat >/dev/null 2>&1; then
        local io_info=$(iostat -d 1 2 | tail -1 | awk '{print $3, $4}' 2>/dev/null || echo "0 0")
        disk_read=$(echo "$io_info" | awk '{print $1}')
        disk_write=$(echo "$io_info" | awk '{print $2}')
    fi
    
    # Output JSON format for easy parsing
    cat <<EOF
{
  "timestamp": $timestamp,
  "cpu": {
    "usage_percent": $cpu_usage,
    "load_avg": "$load_avg"
  },
  "memory": {
    "usage_percent": $mem_usage,
    "used_mb": $mem_used,
    "total_mb": $mem_total
  },
  "disk": {
    "usage_percent": $disk_usage,
    "read_kb": $disk_read,
    "write_kb": $disk_write
  },
  "network": {
    "rx_bytes": $net_rx,
    "tx_bytes": $net_tx
  }
}
EOF
}

# Get CI/CD specific metrics
get_cicd_metrics() {
    local timestamp=$(date +%s)
    
    # Docker metrics
    local docker_containers=0
    local docker_images=0
    local docker_disk_usage=0
    if command -v docker >/dev/null 2>&1; then
        docker_containers=$(docker ps -q | wc -l 2>/dev/null || echo "0")
        docker_images=$(docker images -q | wc -l 2>/dev/null || echo "0")
        docker_disk_usage=$(docker system df --format "{{.Size}}" 2>/dev/null | sed 's/[^0-9.]//g' | head -1 || echo "0")
    fi
    
    # Go build cache size
    local go_cache_size=0
    if [[ -d "$HOME/.cache/go-build" ]]; then
        go_cache_size=$(du -sm "$HOME/.cache/go-build" 2>/dev/null | cut -f1 || echo "0")
    fi
    
    # Module cache size
    local mod_cache_size=0
    if [[ -d "$HOME/go/pkg/mod" ]]; then
        mod_cache_size=$(du -sm "$HOME/go/pkg/mod" 2>/dev/null | cut -f1 || echo "0")
    fi
    
    # Active Go processes
    local go_processes=0
    if command -v pgrep >/dev/null 2>&1; then
        go_processes=$(pgrep -f "go build\|go test\|go run" 2>/dev/null | wc -l)
    fi
    
    # Git repository size
    local repo_size=0
    if [[ -d ".git" ]]; then
        repo_size=$(du -sm .git 2>/dev/null | cut -f1 || echo "0")
    fi
    
    cat <<EOF
{
  "timestamp": $timestamp,
  "docker": {
    "containers": $docker_containers,
    "images": $docker_images,
    "disk_usage_gb": $docker_disk_usage
  },
  "go": {
    "build_cache_mb": $go_cache_size,
    "mod_cache_mb": $mod_cache_size,
    "active_processes": $go_processes
  },
  "git": {
    "repo_size_mb": $repo_size
  }
}
EOF
}

# ML-based performance prediction
predict_performance() {
    local metrics_file="$1"
    local prediction_window="$2"
    
    if [[ ! -f "$metrics_file" ]]; then
        echo '{"prediction": "insufficient_data", "confidence": 0.0}'
        return
    fi
    
    # Simple ML prediction based on historical data
    local recent_cpu=$(tail -n "$prediction_window" "$metrics_file" | jq -r '.cpu.usage_percent' 2>/dev/null | awk '{sum+=$1; count++} END {print sum/count}' || echo "50")
    local recent_mem=$(tail -n "$prediction_window" "$metrics_file" | jq -r '.memory.usage_percent' 2>/dev/null | awk '{sum+=$1; count++} END {print sum/count}' || echo "50")
    
    # Trend analysis
    local cpu_trend=$(tail -n "$prediction_window" "$metrics_file" | jq -r '.cpu.usage_percent' 2>/dev/null | awk 'NR==1{first=$1} END{print ($1-first)/NR}' || echo "0")
    local mem_trend=$(tail -n "$prediction_window" "$metrics_file" | jq -r '.memory.usage_percent' 2>/dev/null | awk 'NR==1{first=$1} END{print ($1-first)/NR}' || echo "0")
    
    # Performance prediction logic
    local predicted_cpu=$(echo "$recent_cpu + $cpu_trend * 10" | bc -l 2>/dev/null || echo "$recent_cpu")
    local predicted_mem=$(echo "$recent_mem + $mem_trend * 10" | bc -l 2>/dev/null || echo "$recent_mem")
    
    # Bottleneck prediction
    local bottleneck="none"
    local confidence=0.5
    
    if (( $(echo "$predicted_cpu > 90" | bc -l 2>/dev/null || echo "0") )); then
        bottleneck="cpu"
        confidence=0.8
    elif (( $(echo "$predicted_mem > 95" | bc -l 2>/dev/null || echo "0") )); then
        bottleneck="memory"
        confidence=0.7
    elif (( $(echo "$predicted_cpu > 80 && $predicted_mem > 80" | bc -l 2>/dev/null || echo "0") )); then
        bottleneck="resource_contention"
        confidence=0.6
    fi
    
    cat <<EOF
{
  "timestamp": $(date +%s),
  "prediction": {
    "cpu_percent": $predicted_cpu,
    "memory_percent": $predicted_mem,
    "bottleneck": "$bottleneck",
    "confidence": $confidence
  },
  "trends": {
    "cpu_trend": $cpu_trend,
    "memory_trend": $mem_trend
  }
}
EOF
}

# Generate optimization recommendations
generate_recommendations() {
    local current_metrics="$1"
    local prediction="$2"
    
    local recommendations=()
    local priority="low"
    
    # Parse current metrics
    local cpu_usage=$(echo "$current_metrics" | jq -r '.cpu.usage_percent' 2>/dev/null || echo "0")
    local mem_usage=$(echo "$current_metrics" | jq -r '.memory.usage_percent' 2>/dev/null || echo "0")
    local disk_usage=$(echo "$current_metrics" | jq -r '.disk.usage_percent' 2>/dev/null || echo "0")
    
    # CPU recommendations
    if (( $(echo "$cpu_usage > 85" | bc -l 2>/dev/null || echo "0") )); then
        recommendations+=("HIGH: Reduce GOMAXPROCS or build parallelism")
        recommendations+=("MEDIUM: Enable build caching to reduce CPU load")
        recommendations+=("LOW: Consider using larger CI runners")
        priority="high"
    elif (( $(echo "$cpu_usage > 70" | bc -l 2>/dev/null || echo "0") )); then
        recommendations+=("MEDIUM: Monitor CPU usage closely")
        recommendations+=("LOW: Consider optimizing build scripts")
        if [[ "$priority" != "high" ]]; then priority="medium"; fi
    fi
    
    # Memory recommendations  
    if (( $(echo "$mem_usage > 90" | bc -l 2>/dev/null || echo "0") )); then
        recommendations+=("HIGH: Reduce GOMEMLIMIT or increase runner memory")
        recommendations+=("HIGH: Enable more aggressive garbage collection (GOGC=25)")
        recommendations+=("MEDIUM: Clear build cache if possible")
        priority="high"
    elif (( $(echo "$mem_usage > 75" | bc -l 2>/dev/null || echo "0") )); then
        recommendations+=("MEDIUM: Monitor memory usage")
        recommendations+=("LOW: Consider memory-optimized build flags")
        if [[ "$priority" == "low" ]]; then priority="medium"; fi
    fi
    
    # Disk recommendations
    if (( $(echo "$disk_usage > 95" | bc -l 2>/dev/null || echo "0") )); then
        recommendations+=("CRITICAL: Clean up disk space immediately")
        recommendations+=("HIGH: Clear Docker build cache")
        recommendations+=("HIGH: Remove old build artifacts")
        priority="critical"
    elif (( $(echo "$disk_usage > 85" | bc -l 2>/dev/null || echo "0") )); then
        recommendations+=("HIGH: Monitor disk usage closely")
        recommendations+=("MEDIUM: Schedule regular cache cleanup")
        if [[ "$priority" != "critical" ]]; then priority="high"; fi
    fi
    
    # Cache-specific recommendations
    local go_cache_mb=$(echo "$current_metrics" | jq -r '.go.build_cache_mb // 0' 2>/dev/null)
    if (( go_cache_mb > 5000 )); then
        recommendations+=("MEDIUM: Go build cache is large (${go_cache_mb}MB)")
        recommendations+=("LOW: Consider periodic cache cleanup")
    fi
    
    # Docker-specific recommendations  
    local docker_images=$(echo "$current_metrics" | jq -r '.docker.images // 0' 2>/dev/null)
    if (( docker_images > 50 )); then
        recommendations+=("LOW: Many Docker images ($docker_images)")
        recommendations+=("LOW: Consider regular image cleanup")
    fi
    
    # Default recommendation if none found
    if [[ ${#recommendations[@]} -eq 0 ]]; then
        recommendations+=("INFO: System performance is optimal")
        priority="info"
    fi
    
    # Output JSON format
    printf '{\n'
    printf '  "timestamp": %s,\n' "$(date +%s)"
    printf '  "priority": "%s",\n' "$priority"
    printf '  "recommendations": [\n'
    
    local first=true
    for rec in "${recommendations[@]}"; do
        if [[ "$first" == "true" ]]; then
            first=false
        else
            printf ',\n'
        fi
        printf '    "%s"' "$rec"
    done
    
    printf '\n  ]\n'
    printf '}\n'
}

# Real-time dashboard display
display_dashboard() {
    local metrics="$1"
    local prediction="$2" 
    local recommendations="$3"
    
    # Clear screen and show header
    clear
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║                    CI/CD Performance Dashboard - $(date '+%H:%M:%S')                    ║${NC}"
    echo -e "${CYAN}╠══════════════════════════════════════════════════════════════════════════════╣${NC}"
    
    # Parse metrics
    local cpu_usage=$(echo "$metrics" | jq -r '.cpu.usage_percent' 2>/dev/null || echo "0")
    local mem_usage=$(echo "$metrics" | jq -r '.memory.usage_percent' 2>/dev/null || echo "0")
    local disk_usage=$(echo "$metrics" | jq -r '.disk.usage_percent' 2>/dev/null || echo "0")
    local load_avg=$(echo "$metrics" | jq -r '.cpu.load_avg' 2>/dev/null || echo "0.00 0.00 0.00")
    
    # CPU display
    local cpu_color=$GREEN
    if (( $(echo "$cpu_usage > 85" | bc -l 2>/dev/null || echo "0") )); then cpu_color=$RED
    elif (( $(echo "$cpu_usage > 70" | bc -l 2>/dev/null || echo "0") )); then cpu_color=$YELLOW; fi
    
    echo -e "${CYAN}║${NC} ${BLUE}CPU Usage:${NC} ${cpu_color}$(printf "%6.1f%%" "$cpu_usage")${NC} │ ${BLUE}Load Avg:${NC} $load_avg"
    
    # Memory display
    local mem_color=$GREEN
    if (( $(echo "$mem_usage > 90" | bc -l 2>/dev/null || echo "0") )); then mem_color=$RED
    elif (( $(echo "$mem_usage > 75" | bc -l 2>/dev/null || echo "0") )); then mem_color=$YELLOW; fi
    
    echo -e "${CYAN}║${NC} ${BLUE}Memory:${NC}    ${mem_color}$(printf "%6.1f%%" "$mem_usage")${NC} │ ${BLUE}Used:${NC} $(echo "$metrics" | jq -r '.memory.used_mb' 2>/dev/null || echo "0")MB / $(echo "$metrics" | jq -r '.memory.total_mb' 2>/dev/null || echo "0")MB"
    
    # Disk display  
    local disk_color=$GREEN
    if (( $(echo "$disk_usage > 95" | bc -l 2>/dev/null || echo "0") )); then disk_color=$RED
    elif (( $(echo "$disk_usage > 85" | bc -l 2>/dev/null || echo "0") )); then disk_color=$YELLOW; fi
    
    echo -e "${CYAN}║${NC} ${BLUE}Disk:${NC}      ${disk_color}$(printf "%6s%%" "$disk_usage")${NC}"
    
    # CI/CD Metrics
    echo -e "${CYAN}╠══════════════════════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${CYAN}║${NC} ${MAGENTA}CI/CD Metrics${NC}"
    
    local go_processes=$(echo "$metrics" | jq -r '.go.active_processes // 0' 2>/dev/null)
    local go_cache_mb=$(echo "$metrics" | jq -r '.go.build_cache_mb // 0' 2>/dev/null)
    local mod_cache_mb=$(echo "$metrics" | jq -r '.go.mod_cache_mb // 0' 2>/dev/null)
    
    echo -e "${CYAN}║${NC} ${BLUE}Go Processes:${NC} $go_processes │ ${BLUE}Build Cache:${NC} ${go_cache_mb}MB │ ${BLUE}Mod Cache:${NC} ${mod_cache_mb}MB"
    
    # Predictions
    if [[ "$prediction" != '{"prediction": "insufficient_data", "confidence": 0.0}' ]]; then
        echo -e "${CYAN}╠══════════════════════════════════════════════════════════════════════════════╣${NC}"
        echo -e "${CYAN}║${NC} ${MAGENTA}Performance Prediction${NC}"
        
        local bottleneck=$(echo "$prediction" | jq -r '.prediction.bottleneck' 2>/dev/null || echo "none")
        local confidence=$(echo "$prediction" | jq -r '.prediction.confidence' 2>/dev/null || echo "0")
        
        local bottleneck_color=$GREEN
        if [[ "$bottleneck" != "none" ]]; then
            bottleneck_color=$YELLOW
            if (( $(echo "$confidence > 0.7" | bc -l 2>/dev/null || echo "0") )); then
                bottleneck_color=$RED
            fi
        fi
        
        echo -e "${CYAN}║${NC} ${BLUE}Bottleneck:${NC} ${bottleneck_color}$bottleneck${NC} │ ${BLUE}Confidence:${NC} $(printf "%.0f%%" $(echo "$confidence * 100" | bc -l 2>/dev/null || echo "0"))"
    fi
    
    # Recommendations
    echo -e "${CYAN}╠══════════════════════════════════════════════════════════════════════════════╣${NC}"
    echo -e "${CYAN}║${NC} ${MAGENTA}Optimization Recommendations${NC}"
    
    local priority=$(echo "$recommendations" | jq -r '.priority' 2>/dev/null || echo "info")
    local priority_color=$GREEN
    case "$priority" in
        "critical") priority_color=$RED ;;
        "high") priority_color=$YELLOW ;;
        "medium") priority_color=$BLUE ;;
    esac
    
    echo -e "${CYAN}║${NC} ${BLUE}Priority:${NC} ${priority_color}$priority${NC}"
    
    # Show top 3 recommendations
    local rec_count=0
    while IFS= read -r rec && [[ $rec_count -lt 3 ]]; do
        if [[ -n "$rec" && "$rec" != "null" ]]; then
            echo -e "${CYAN}║${NC} • $rec"
            ((rec_count++))
        fi
    done < <(echo "$recommendations" | jq -r '.recommendations[]' 2>/dev/null)
    
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
    echo -e "${BLUE}Press Ctrl+C to stop monitoring${NC}"
}

# =============================================================================
# Main Functions
# =============================================================================

setup_monitoring() {
    log_info "🚀 Setting up performance monitoring..."
    
    # Create metrics directory
    mkdir -p "$METRICS_DIR"
    
    # Initialize log files
    touch "$LOG_FILE"
    touch "$METRICS_DIR/system-metrics.jsonl"
    touch "$METRICS_DIR/cicd-metrics.jsonl"
    touch "$METRICS_DIR/predictions.jsonl"
    touch "$METRICS_DIR/recommendations.jsonl"
    
    log_success "✅ Monitoring setup complete"
    log_info "📊 Metrics directory: $METRICS_DIR"
    log_info "📝 Log file: $LOG_FILE"
}

monitor_performance() {
    local duration="${1:-300}"  # Default 5 minutes
    local dashboard="${2:-false}"
    
    log_info "🔍 Starting performance monitoring for ${duration}s..."
    
    local start_time=$(date +%s)
    local end_time=$((start_time + duration))
    
    while [[ $(date +%s) -lt $end_time ]]; do
        # Collect metrics
        local system_metrics=$(get_system_metrics)
        local cicd_metrics=$(get_cicd_metrics)
        local combined_metrics=$(echo "$system_metrics $cicd_metrics" | jq -s '.[0] + .[1]' 2>/dev/null || echo "$system_metrics")
        
        # Save metrics
        echo "$system_metrics" >> "$METRICS_DIR/system-metrics.jsonl"
        echo "$cicd_metrics" >> "$METRICS_DIR/cicd-metrics.jsonl"
        
        # Generate prediction
        local prediction=$(predict_performance "$METRICS_DIR/system-metrics.jsonl" "$PREDICTION_WINDOW")
        echo "$prediction" >> "$METRICS_DIR/predictions.jsonl"
        
        # Generate recommendations
        local recommendations=$(generate_recommendations "$combined_metrics" "$prediction")
        echo "$recommendations" >> "$METRICS_DIR/recommendations.jsonl"
        
        # Display dashboard if requested
        if [[ "$dashboard" == "true" ]]; then
            display_dashboard "$combined_metrics" "$prediction" "$recommendations"
        fi
        
        # Check for alerts
        check_alerts "$combined_metrics" "$recommendations"
        
        # Wait for next interval
        sleep "$MONITOR_INTERVAL"
    done
    
    log_success "✅ Monitoring completed"
}

check_alerts() {
    local metrics="$1"
    local recommendations="$2"
    
    local cpu_usage=$(echo "$metrics" | jq -r '.cpu.usage_percent' 2>/dev/null || echo "0")
    local mem_usage=$(echo "$metrics" | jq -r '.memory.usage_percent' 2>/dev/null || echo "0")
    local disk_usage=$(echo "$metrics" | jq -r '.disk.usage_percent' 2>/dev/null || echo "0")
    local priority=$(echo "$recommendations" | jq -r '.priority' 2>/dev/null || echo "info")
    
    # CPU alerts
    if (( $(echo "$cpu_usage > $ALERT_THRESHOLD_CPU" | bc -l 2>/dev/null || echo "0") )); then
        log_warn "🚨 HIGH CPU USAGE ALERT: ${cpu_usage}% (threshold: ${ALERT_THRESHOLD_CPU}%)"
    fi
    
    # Memory alerts
    if (( $(echo "$mem_usage > $ALERT_THRESHOLD_MEMORY" | bc -l 2>/dev/null || echo "0") )); then
        log_warn "🚨 HIGH MEMORY USAGE ALERT: ${mem_usage}% (threshold: ${ALERT_THRESHOLD_MEMORY}%)"
    fi
    
    # Disk alerts
    if (( $(echo "$disk_usage > $ALERT_THRESHOLD_DISK" | bc -l 2>/dev/null || echo "0") )); then
        log_error "🚨 CRITICAL DISK USAGE ALERT: ${disk_usage}% (threshold: ${ALERT_THRESHOLD_DISK}%)"
    fi
    
    # Priority alerts
    if [[ "$priority" == "critical" ]]; then
        log_error "🚨 CRITICAL PERFORMANCE ALERT: Immediate action required"
    elif [[ "$priority" == "high" ]]; then
        log_warn "⚠️ HIGH PRIORITY PERFORMANCE WARNING"
    fi
}

generate_report() {
    local output_file="${1:-performance-report.md}"
    
    log_info "📊 Generating performance report..."
    
    if [[ ! -f "$METRICS_DIR/system-metrics.jsonl" ]]; then
        log_error "No metrics data found. Run monitoring first."
        return 1
    fi
    
    # Calculate statistics
    local avg_cpu=$(cat "$METRICS_DIR/system-metrics.jsonl" | jq -r '.cpu.usage_percent' | awk '{sum+=$1; count++} END {print sum/count}' 2>/dev/null || echo "0")
    local max_cpu=$(cat "$METRICS_DIR/system-metrics.jsonl" | jq -r '.cpu.usage_percent' | sort -n | tail -1 2>/dev/null || echo "0")
    local avg_mem=$(cat "$METRICS_DIR/system-metrics.jsonl" | jq -r '.memory.usage_percent' | awk '{sum+=$1; count++} END {print sum/count}' 2>/dev/null || echo "0")
    local max_mem=$(cat "$METRICS_DIR/system-metrics.jsonl" | jq -r '.memory.usage_percent' | sort -n | tail -1 2>/dev/null || echo "0")
    
    # Generate report
    cat > "$output_file" << EOF
# CI/CD Performance Monitoring Report

**Generated:** $(date -Iseconds)  
**Monitoring Period:** $(wc -l < "$METRICS_DIR/system-metrics.jsonl") data points  
**Report Location:** $output_file

## Executive Summary

### Performance Metrics
- **Average CPU Usage:** $(printf "%.1f%%" "$avg_cpu")
- **Peak CPU Usage:** $(printf "%.1f%%" "$max_cpu")  
- **Average Memory Usage:** $(printf "%.1f%%" "$avg_mem")
- **Peak Memory Usage:** $(printf "%.1f%%" "$max_mem")

### System Health
- **Overall Status:** $(if (( $(echo "$avg_cpu < 70 && $avg_mem < 80" | bc -l 2>/dev/null || echo "0") )); then echo "✅ Healthy"; elif (( $(echo "$avg_cpu < 85 && $avg_mem < 90" | bc -l 2>/dev/null || echo "0") )); then echo "⚠️ Needs Attention"; else echo "🚨 Critical"; fi)
- **Bottlenecks Detected:** $(tail -10 "$METRICS_DIR/predictions.jsonl" | jq -r '.prediction.bottleneck' | grep -v "none" | sort | uniq -c | sort -nr | head -1 | awk '{print $2}' || echo "none")

## Detailed Analysis

### Resource Utilization Trends

#### CPU Usage Over Time
\`\`\`
$(cat "$METRICS_DIR/system-metrics.jsonl" | jq -r '[.timestamp, .cpu.usage_percent] | @tsv' | tail -20)
\`\`\`

#### Memory Usage Over Time  
\`\`\`
$(cat "$METRICS_DIR/system-metrics.jsonl" | jq -r '[.timestamp, .memory.usage_percent] | @tsv' | tail -20)
\`\`\`

### Top Performance Recommendations

$(tail -5 "$METRICS_DIR/recommendations.jsonl" | jq -r '.recommendations[]' | head -10 | sed 's/^/- /')

## Optimization Opportunities

### Immediate Actions
$(tail -1 "$METRICS_DIR/recommendations.jsonl" | jq -r '.recommendations[] | select(contains("HIGH") or contains("CRITICAL"))' | sed 's/^/- /')

### Long-term Improvements
$(tail -1 "$METRICS_DIR/recommendations.jsonl" | jq -r '.recommendations[] | select(contains("MEDIUM") or contains("LOW"))' | sed 's/^/- /')

## Technical Details

### System Information
- **Monitoring Interval:** ${MONITOR_INTERVAL}s
- **Prediction Window:** ${PREDICTION_WINDOW} data points  
- **Alert Thresholds:** CPU: ${ALERT_THRESHOLD_CPU}%, Memory: ${ALERT_THRESHOLD_MEMORY}%, Disk: ${ALERT_THRESHOLD_DISK}%

### Data Files
- **System Metrics:** \`$METRICS_DIR/system-metrics.jsonl\`
- **CI/CD Metrics:** \`$METRICS_DIR/cicd-metrics.jsonl\`
- **Predictions:** \`$METRICS_DIR/predictions.jsonl\`
- **Recommendations:** \`$METRICS_DIR/recommendations.jsonl\`

---
*Report generated by CI/CD Performance Monitor v2025.1*
EOF
    
    log_success "✅ Performance report generated: $output_file"
}

# =============================================================================
# CLI Interface
# =============================================================================

show_help() {
    cat << EOF
CI/CD Performance Monitoring & Analytics Tool

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    monitor [DURATION] [--dashboard]    Monitor performance for specified seconds
    report [OUTPUT_FILE]               Generate performance report  
    alerts                            Check current alerts
    dashboard                         Show real-time dashboard
    setup                            Initialize monitoring environment
    clean                            Clean old metrics data
    help                             Show this help

Options:
    --interval=N                     Set monitoring interval in seconds (default: 5)
    --prediction-window=N            Set prediction window size (default: 60)
    --cpu-threshold=N               Set CPU alert threshold % (default: 85)
    --memory-threshold=N            Set memory alert threshold % (default: 90)
    --disk-threshold=N              Set disk alert threshold % (default: 95)

Examples:
    $0 monitor 300 --dashboard       Monitor for 5 minutes with live dashboard
    $0 report performance.md         Generate detailed performance report
    $0 alerts                        Check current system alerts
    
Environment Variables:
    METRICS_DIR                      Directory for storing metrics data
    MONITOR_INTERVAL                 Default monitoring interval  
    PREDICTION_WINDOW               Default prediction window size
    ALERT_THRESHOLD_CPU             CPU usage alert threshold
    ALERT_THRESHOLD_MEMORY          Memory usage alert threshold  
    ALERT_THRESHOLD_DISK            Disk usage alert threshold

EOF
}

main() {
    # Parse options
    while [[ $# -gt 0 ]]; do
        case $1 in
            --interval=*)
                MONITOR_INTERVAL="${1#*=}"
                shift
                ;;
            --prediction-window=*)
                PREDICTION_WINDOW="${1#*=}"
                shift
                ;;
            --cpu-threshold=*)
                ALERT_THRESHOLD_CPU="${1#*=}"
                shift
                ;;
            --memory-threshold=*)
                ALERT_THRESHOLD_MEMORY="${1#*=}"
                shift
                ;;
            --disk-threshold=*)
                ALERT_THRESHOLD_DISK="${1#*=}"
                shift
                ;;
            --dashboard)
                DASHBOARD_MODE="true"
                shift
                ;;
            -*)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
            *)
                break
                ;;
        esac
    done
    
    # Get command
    local command="${1:-help}"
    shift 2>/dev/null || true
    
    # Setup metrics directory
    mkdir -p "$METRICS_DIR"
    
    # Execute command
    case "$command" in
        monitor)
            setup_monitoring
            local duration="${1:-300}"
            local dashboard="${DASHBOARD_MODE:-false}"
            monitor_performance "$duration" "$dashboard"
            ;;
        dashboard)
            setup_monitoring
            monitor_performance "86400" "true"  # 24 hours with dashboard
            ;;
        report)
            generate_report "${1:-performance-report.md}"
            ;;
        alerts)
            local current_metrics=$(get_system_metrics)
            local current_cicd=$(get_cicd_metrics)
            local combined=$(echo "$current_metrics $current_cicd" | jq -s '.[0] + .[1]' 2>/dev/null || echo "$current_metrics")
            local recommendations=$(generate_recommendations "$combined" '{}')
            check_alerts "$combined" "$recommendations"
            ;;
        setup)
            setup_monitoring
            detect_system
            ;;
        clean)
            log_info "🧹 Cleaning metrics data..."
            rm -rf "$METRICS_DIR"/*.jsonl "$METRICS_DIR"/*.log
            log_success "✅ Metrics data cleaned"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Handle Ctrl+C gracefully
trap 'log_info "🛑 Monitoring stopped by user"; exit 0' INT

# Run main function
main "$@"