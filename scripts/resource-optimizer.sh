#!/bin/bash
# =============================================================================
# Nephoran Intent Operator - Resource Optimization Engine  
# =============================================================================
# Advanced memory and CPU optimization for Go 1.24.x builds
# Features: Dynamic resource allocation, memory profiling, CPU optimization
# Target: Maximum resource efficiency for large Kubernetes operator builds
# =============================================================================

set -euo pipefail

# Script configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
readonly RESOURCE_MONITOR_DIR="${PROJECT_ROOT}/build/resource-monitoring"

# Resource optimization settings
readonly MIN_MEMORY_GB=4
readonly RECOMMENDED_MEMORY_GB=12
readonly MIN_CPU_CORES=2
readonly RECOMMENDED_CPU_CORES=8

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly NC='\033[0m'

log_info() { echo -e "${BLUE}[RESOURCE]${NC} $*" >&2; }
log_success() { echo -e "${GREEN}[RESOURCE]${NC} $*" >&2; }
log_warn() { echo -e "${YELLOW}[RESOURCE]${NC} $*" >&2; }
log_error() { echo -e "${RED}[RESOURCE]${NC} $*" >&2; }
log_monitor() { echo -e "${CYAN}[RESOURCE]${NC} $*" >&2; }

# System resource detection and analysis
detect_system_resources() {
    log_info "Detecting and analyzing system resources..."
    
    # CPU information
    local cpu_cores
    cpu_cores=$(nproc 2>/dev/null || echo "1")
    local cpu_model
    cpu_model=$(grep "model name" /proc/cpuinfo | head -1 | cut -d: -f2 | xargs 2>/dev/null || echo "Unknown CPU")
    local cpu_freq
    cpu_freq=$(lscpu | grep "CPU MHz" | awk '{print $3}' 2>/dev/null || echo "Unknown")
    
    # Memory information
    local total_memory_kb
    total_memory_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}' 2>/dev/null || echo "0")
    local total_memory_gb=$((total_memory_kb / 1024 / 1024))
    local available_memory_kb
    available_memory_kb=$(grep MemAvailable /proc/meminfo | awk '{print $2}' 2>/dev/null || echo "$total_memory_kb")
    local available_memory_gb=$((available_memory_kb / 1024 / 1024))
    
    # Storage information
    local disk_space_gb
    disk_space_gb=$(df -BG "$PROJECT_ROOT" | tail -1 | awk '{print $4}' | sed 's/G//' 2>/dev/null || echo "0")
    
    # Load average
    local load_avg
    load_avg=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | sed 's/,//' 2>/dev/null || echo "0.0")
    
    # Export system information
    export DETECTED_CPU_CORES="$cpu_cores"
    export DETECTED_MEMORY_GB="$total_memory_gb"
    export DETECTED_AVAILABLE_MEMORY_GB="$available_memory_gb"
    export DETECTED_DISK_SPACE_GB="$disk_space_gb"
    export DETECTED_LOAD_AVG="$load_avg"
    
    log_success "System resources detected:"
    log_info "  CPU: $cpu_model"
    log_info "  Cores: $cpu_cores"
    log_info "  Frequency: ${cpu_freq} MHz"
    log_info "  Total Memory: ${total_memory_gb}GB"
    log_info "  Available Memory: ${available_memory_gb}GB"
    log_info "  Disk Space: ${disk_space_gb}GB"
    log_info "  Load Average: $load_avg"
    
    # Resource adequacy analysis
    analyze_resource_adequacy "$cpu_cores" "$total_memory_gb" "$disk_space_gb"
}

# Analyze resource adequacy and provide recommendations
analyze_resource_adequacy() {
    local cpu_cores="$1"
    local memory_gb="$2"
    local disk_gb="$3"
    
    log_info "Analyzing resource adequacy for Nephoran build..."
    
    local adequacy_score=0
    local max_score=3
    local recommendations=()
    
    # CPU adequacy
    if [[ $cpu_cores -ge $RECOMMENDED_CPU_CORES ]]; then
        log_success "  CPU: Excellent ($cpu_cores cores >= $RECOMMENDED_CPU_CORES recommended)"
        ((adequacy_score++))
    elif [[ $cpu_cores -ge $MIN_CPU_CORES ]]; then
        log_warn "  CPU: Adequate ($cpu_cores cores >= $MIN_CPU_CORES minimum)"
        recommendations+=("Consider upgrading to $RECOMMENDED_CPU_CORES+ cores for optimal performance")
    else
        log_error "  CPU: Insufficient ($cpu_cores cores < $MIN_CPU_CORES minimum)"
        recommendations+=("CRITICAL: Upgrade to at least $MIN_CPU_CORES cores")
    fi
    
    # Memory adequacy
    if [[ $memory_gb -ge $RECOMMENDED_MEMORY_GB ]]; then
        log_success "  Memory: Excellent (${memory_gb}GB >= ${RECOMMENDED_MEMORY_GB}GB recommended)"
        ((adequacy_score++))
    elif [[ $memory_gb -ge $MIN_MEMORY_GB ]]; then
        log_warn "  Memory: Adequate (${memory_gb}GB >= ${MIN_MEMORY_GB}GB minimum)"
        recommendations+=("Consider upgrading to ${RECOMMENDED_MEMORY_GB}GB+ RAM for optimal caching")
    else
        log_error "  Memory: Insufficient (${memory_gb}GB < ${MIN_MEMORY_GB}GB minimum)"
        recommendations+=("CRITICAL: Upgrade to at least ${MIN_MEMORY_GB}GB RAM")
    fi
    
    # Disk space adequacy
    if [[ $disk_gb -ge 20 ]]; then
        log_success "  Disk: Excellent (${disk_gb}GB available)"
        ((adequacy_score++))
    elif [[ $disk_gb -ge 10 ]]; then
        log_warn "  Disk: Adequate (${disk_gb}GB available)"
        recommendations+=("Monitor disk space during builds")
    else
        log_error "  Disk: Insufficient (${disk_gb}GB available)"
        recommendations+=("CRITICAL: Free up disk space (need 10GB+ for builds)")
    fi
    
    # Overall adequacy assessment
    local adequacy_percentage=$((adequacy_score * 100 / max_score))
    
    if [[ $adequacy_percentage -ge 80 ]]; then
        log_success "Overall resource adequacy: Excellent (${adequacy_percentage}%)"
        export RESOURCE_ADEQUACY="excellent"
    elif [[ $adequacy_percentage -ge 60 ]]; then
        log_warn "Overall resource adequacy: Good (${adequacy_percentage}%)"
        export RESOURCE_ADEQUACY="good"
    else
        log_error "Overall resource adequacy: Poor (${adequacy_percentage}%)"
        export RESOURCE_ADEQUACY="poor"
    fi
    
    # Print recommendations
    if [[ ${#recommendations[@]} -gt 0 ]]; then
        log_warn "Resource optimization recommendations:"
        for rec in "${recommendations[@]}"; do
            log_warn "  - $rec"
        done
    fi
    
    export RESOURCE_ADEQUACY_SCORE="$adequacy_score"
    export RESOURCE_ADEQUACY_PERCENTAGE="$adequacy_percentage"
}

# Calculate optimal resource allocation
calculate_optimal_allocation() {
    local build_mode="${1:-standard}"
    
    log_info "Calculating optimal resource allocation for $build_mode build..."
    
    local cpu_cores="$DETECTED_CPU_CORES"
    local memory_gb="$DETECTED_MEMORY_GB"
    local available_memory_gb="$DETECTED_AVAILABLE_MEMORY_GB"
    
    # Calculate optimal GOMAXPROCS
    local optimal_gomaxprocs
    case "$build_mode" in
        "ultra-fast"|"aggressive")
            # Use maximum available cores for ultra-fast builds
            optimal_gomaxprocs="$cpu_cores"
            ;;
        "balanced"|"standard")
            # Reserve 1 core for system processes
            optimal_gomaxprocs=$((cpu_cores > 1 ? cpu_cores - 1 : 1))
            ;;
        "conservative")
            # Use 75% of cores
            optimal_gomaxprocs=$((cpu_cores * 3 / 4))
            optimal_gomaxprocs=$((optimal_gomaxprocs < 1 ? 1 : optimal_gomaxprocs))
            ;;
        "minimal")
            # Use minimum cores
            optimal_gomaxprocs=$((cpu_cores > 2 ? 2 : 1))
            ;;
        *)
            optimal_gomaxprocs=$((cpu_cores > 1 ? cpu_cores - 1 : 1))
            ;;
    esac
    
    # Calculate optimal memory limit (GOMEMLIMIT)
    local optimal_memory_limit
    local memory_for_go=$((available_memory_gb * 80 / 100))  # Use 80% of available memory
    
    case "$build_mode" in
        "ultra-fast"|"aggressive")
            optimal_memory_limit="${memory_for_go}GiB"
            ;;
        "balanced"|"standard")
            local conservative_memory=$((memory_for_go * 85 / 100))
            optimal_memory_limit="${conservative_memory}GiB"
            ;;
        "conservative"|"minimal")
            local minimal_memory=$((memory_for_go * 70 / 100))
            optimal_memory_limit="${minimal_memory}GiB"
            ;;
        *)
            optimal_memory_limit="${memory_for_go}GiB"
            ;;
    esac
    
    # Calculate build and test parallelism
    local build_parallelism
    local test_parallelism
    
    case "$build_mode" in
        "ultra-fast"|"aggressive")
            build_parallelism="$optimal_gomaxprocs"
            test_parallelism=$((optimal_gomaxprocs > 2 ? optimal_gomaxprocs / 2 : 1))
            ;;
        "balanced"|"standard")
            build_parallelism=$((optimal_gomaxprocs > 1 ? optimal_gomaxprocs - 1 : 1))
            test_parallelism=$((optimal_gomaxprocs > 4 ? optimal_gomaxprocs / 3 : 1))
            ;;
        "conservative")
            build_parallelism=$((optimal_gomaxprocs > 2 ? optimal_gomaxprocs / 2 : 1))
            test_parallelism=$((optimal_gomaxprocs > 4 ? optimal_gomaxprocs / 4 : 1))
            ;;
        "minimal")
            build_parallelism=2
            test_parallelism=1
            ;;
        *)
            build_parallelism="$optimal_gomaxprocs"
            test_parallelism=$((optimal_gomaxprocs > 2 ? optimal_gomaxprocs / 2 : 1))
            ;;
    esac
    
    # Calculate optimal GOGC based on available memory
    local optimal_gogc
    if [[ $memory_gb -ge 16 ]]; then
        optimal_gogc=75  # Aggressive GC for high-memory systems
    elif [[ $memory_gb -ge 8 ]]; then
        optimal_gogc=100  # Default GC for medium-memory systems
    else
        optimal_gogc=150  # Conservative GC for low-memory systems
    fi
    
    # Export optimized settings
    export OPTIMAL_GOMAXPROCS="$optimal_gomaxprocs"
    export OPTIMAL_GOMEMLIMIT="$optimal_memory_limit"
    export OPTIMAL_BUILD_PARALLELISM="$build_parallelism"
    export OPTIMAL_TEST_PARALLELISM="$test_parallelism"
    export OPTIMAL_GOGC="$optimal_gogc"
    
    log_success "Optimal resource allocation calculated:"
    log_info "  GOMAXPROCS: $optimal_gomaxprocs"
    log_info "  GOMEMLIMIT: $optimal_memory_limit"
    log_info "  Build parallelism: $build_parallelism"
    log_info "  Test parallelism: $test_parallelism"
    log_info "  GOGC: $optimal_gogc"
}

# Apply resource optimizations
apply_resource_optimizations() {
    local build_mode="${1:-standard}"
    
    log_info "Applying resource optimizations for $build_mode build..."
    
    # Calculate optimal allocation first
    calculate_optimal_allocation "$build_mode"
    
    # Apply core Go environment optimizations
    export GOMAXPROCS="$OPTIMAL_GOMAXPROCS"
    export GOMEMLIMIT="$OPTIMAL_GOMEMLIMIT"
    export GOGC="$OPTIMAL_GOGC"
    export BUILD_PARALLELISM="$OPTIMAL_BUILD_PARALLELISM"
    export TEST_PARALLELISM="$OPTIMAL_TEST_PARALLELISM"
    
    # Apply memory optimizations
    export GODEBUG="gocachehash=1,gocachetest=1,madvdontneed=1"
    export GOTRACEBACK=crash
    export GOMEMLIMIT_LOG=1
    
    # Create optimized temporary directory
    local temp_dir="${PROJECT_ROOT}/.tmp"
    mkdir -p "$temp_dir"
    export GOTMPDIR="$temp_dir"
    export TMPDIR="$temp_dir"
    
    # System-level optimizations (if possible)
    apply_system_optimizations
    
    # Generate resource configuration file
    generate_resource_config "$build_mode"
    
    log_success "Resource optimizations applied"
}

# Apply system-level optimizations
apply_system_optimizations() {
    log_info "Applying system-level optimizations..."
    
    # Increase file descriptor limits
    if ulimit -n 65536 2>/dev/null; then
        log_success "  File descriptor limit increased to 65536"
    else
        log_warn "  Could not increase file descriptor limit"
    fi
    
    # Set process priority for build performance
    if renice -10 $$ 2>/dev/null; then
        log_success "  Process priority increased"
    else
        log_warn "  Could not increase process priority"
    fi
    
    # Optimize I/O scheduling (if ionice is available)
    if command -v ionice >/dev/null 2>&1; then
        if ionice -c 1 -n 4 -p $$ 2>/dev/null; then
            log_success "  I/O priority optimized for builds"
        else
            log_warn "  Could not optimize I/O priority"
        fi
    fi
    
    # Memory management optimizations (requires root)
    if [[ $EUID -eq 0 ]]; then
        # Reduce swappiness for build workloads
        echo 10 > /proc/sys/vm/swappiness 2>/dev/null || true
        
        # Optimize dirty page handling
        echo 15 > /proc/sys/vm/dirty_ratio 2>/dev/null || true
        echo 5 > /proc/sys/vm/dirty_background_ratio 2>/dev/null || true
        
        log_success "  Kernel memory parameters optimized"
    else
        log_warn "  Kernel optimizations skipped (requires root)"
    fi
}

# Generate resource configuration file
generate_resource_config() {
    local build_mode="$1"
    
    mkdir -p "$RESOURCE_MONITOR_DIR"
    local config_file="$RESOURCE_MONITOR_DIR/resource-config.json"
    
    log_info "Generating resource configuration..."
    
    cat > "$config_file" << EOF
{
  "resource_optimization": {
    "timestamp": "$(date -Iseconds)",
    "build_mode": "$build_mode",
    "system_info": {
      "cpu_cores": $DETECTED_CPU_CORES,
      "total_memory_gb": $DETECTED_MEMORY_GB,
      "available_memory_gb": $DETECTED_AVAILABLE_MEMORY_GB,
      "disk_space_gb": $DETECTED_DISK_SPACE_GB,
      "load_average": "$DETECTED_LOAD_AVG"
    },
    "adequacy_assessment": {
      "score": $RESOURCE_ADEQUACY_SCORE,
      "percentage": $RESOURCE_ADEQUACY_PERCENTAGE,
      "level": "$RESOURCE_ADEQUACY"
    },
    "optimal_settings": {
      "GOMAXPROCS": $OPTIMAL_GOMAXPROCS,
      "GOMEMLIMIT": "$OPTIMAL_GOMEMLIMIT",
      "GOGC": $OPTIMAL_GOGC,
      "BUILD_PARALLELISM": $OPTIMAL_BUILD_PARALLELISM,
      "TEST_PARALLELISM": $OPTIMAL_TEST_PARALLELISM
    },
    "environment_variables": {
      "CGO_ENABLED": "0",
      "GOOS": "linux",
      "GOARCH": "amd64",
      "GOEXPERIMENT": "fieldtrack,boringcrypto",
      "GODEBUG": "gocachehash=1,gocachetest=1,madvdontneed=1",
      "GOFLAGS": "-mod=readonly -trimpath -buildvcs=false"
    },
    "recommendations": [
EOF
    
    # Add recommendations based on resource adequacy
    if [[ $RESOURCE_ADEQUACY_PERCENTAGE -lt 80 ]]; then
        cat >> "$config_file" << EOF
      "Consider upgrading system resources for optimal performance",
      "Monitor resource usage during builds to identify bottlenecks",
      "Use conservative build modes on resource-constrained systems"
EOF
    else
        cat >> "$config_file" << EOF
      "System resources are adequate for ultra-fast builds",
      "Enable aggressive optimization modes for maximum performance",
      "Consider using distributed builds for even faster results"
EOF
    fi
    
    cat >> "$config_file" << EOF
    ]
  }
}
EOF
    
    log_success "Resource configuration generated: $config_file"
}

# Start resource monitoring
start_resource_monitoring() {
    local monitoring_duration="${1:-300}"  # Default 5 minutes
    
    log_info "Starting resource monitoring for ${monitoring_duration}s..."
    
    mkdir -p "$RESOURCE_MONITOR_DIR"
    local monitor_file="$RESOURCE_MONITOR_DIR/resource-usage.log"
    local monitor_pid_file="$RESOURCE_MONITOR_DIR/monitor.pid"
    
    # Background monitoring script
    {
        echo "timestamp,cpu_percent,memory_percent,memory_used_gb,load_avg,disk_io_read,disk_io_write" > "$monitor_file"
        
        local end_time=$(($(date +%s) + monitoring_duration))
        
        while [[ $(date +%s) -lt $end_time ]]; do
            local timestamp
            timestamp=$(date -Iseconds)
            
            local cpu_percent
            cpu_percent=$(top -bn1 | grep "Cpu(s)" | awk '{print 100 - $8}' | cut -d% -f1 2>/dev/null || echo "0")
            
            local memory_info
            memory_info=$(free | grep Mem:)
            local total_mem
            total_mem=$(echo "$memory_info" | awk '{print $2}')
            local used_mem
            used_mem=$(echo "$memory_info" | awk '{print $3}')
            local memory_percent
            memory_percent=$(( used_mem * 100 / total_mem ))
            local memory_used_gb
            memory_used_gb=$(( used_mem / 1024 / 1024 ))
            
            local load_avg
            load_avg=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | sed 's/,//')
            
            # Simple disk I/O monitoring (if iostat available)
            local disk_io_read=0
            local disk_io_write=0
            if command -v iostat >/dev/null 2>&1; then
                local io_stats
                io_stats=$(iostat -d 1 1 | tail -1)
                disk_io_read=$(echo "$io_stats" | awk '{print $3}' 2>/dev/null || echo "0")
                disk_io_write=$(echo "$io_stats" | awk '{print $4}' 2>/dev/null || echo "0")
            fi
            
            echo "$timestamp,$cpu_percent,$memory_percent,$memory_used_gb,$load_avg,$disk_io_read,$disk_io_write" >> "$monitor_file"
            
            sleep 5
        done
        
        log_monitor "Resource monitoring completed"
    } &
    
    local monitor_pid=$!
    echo "$monitor_pid" > "$monitor_pid_file"
    
    log_success "Resource monitoring started (PID: $monitor_pid)"
    log_info "  Monitor file: $monitor_file"
    log_info "  Duration: ${monitoring_duration}s"
    
    export RESOURCE_MONITOR_PID="$monitor_pid"
    export RESOURCE_MONITOR_FILE="$monitor_file"
}

# Stop resource monitoring and generate report
stop_resource_monitoring() {
    log_info "Stopping resource monitoring and generating report..."
    
    local monitor_pid_file="$RESOURCE_MONITOR_DIR/monitor.pid"
    
    if [[ -f "$monitor_pid_file" ]]; then
        local monitor_pid
        monitor_pid=$(cat "$monitor_pid_file")
        
        if kill -TERM "$monitor_pid" 2>/dev/null; then
            log_success "Resource monitoring stopped (PID: $monitor_pid)"
        else
            log_warn "Could not stop resource monitoring process"
        fi
        
        rm -f "$monitor_pid_file"
    fi
    
    # Generate monitoring report
    if [[ -f "$RESOURCE_MONITOR_FILE" ]]; then
        generate_monitoring_report "$RESOURCE_MONITOR_FILE"
    fi
}

# Generate resource monitoring report
generate_monitoring_report() {
    local monitor_file="$1"
    
    log_info "Generating resource monitoring report..."
    
    if [[ ! -f "$monitor_file" ]]; then
        log_warn "Monitor file not found: $monitor_file"
        return
    fi
    
    local report_file="$RESOURCE_MONITOR_DIR/resource-report.txt"
    local json_report="$RESOURCE_MONITOR_DIR/resource-report.json"
    
    # Calculate statistics from monitoring data
    local avg_cpu
    avg_cpu=$(tail -n +2 "$monitor_file" | awk -F, '{sum+=$2; count++} END {printf "%.1f", sum/count}' 2>/dev/null || echo "0.0")
    
    local max_cpu
    max_cpu=$(tail -n +2 "$monitor_file" | awk -F, '{if($2>max) max=$2} END {print max+0}' 2>/dev/null || echo "0")
    
    local avg_memory
    avg_memory=$(tail -n +2 "$monitor_file" | awk -F, '{sum+=$3; count++} END {printf "%.1f", sum/count}' 2>/dev/null || echo "0.0")
    
    local max_memory
    max_memory=$(tail -n +2 "$monitor_file" | awk -F, '{if($3>max) max=$3} END {print max+0}' 2>/dev/null || echo "0")
    
    local avg_load
    avg_load=$(tail -n +2 "$monitor_file" | awk -F, '{sum+=$5; count++} END {printf "%.2f", sum/count}' 2>/dev/null || echo "0.00")
    
    # Generate human-readable report
    cat > "$report_file" << EOF
Nephoran Resource Monitoring Report
==================================

Monitoring Period: $(head -2 "$monitor_file" | tail -1 | cut -d, -f1) to $(tail -1 "$monitor_file" | cut -d, -f1)
Data Points: $(( $(wc -l < "$monitor_file") - 1 ))

Resource Usage Summary:
  CPU Usage:
    Average: ${avg_cpu}%
    Peak: ${max_cpu}%
    
  Memory Usage:
    Average: ${avg_memory}%
    Peak: ${max_memory}%
    
  System Load:
    Average: ${avg_load}

Performance Analysis:
EOF
    
    # Add performance analysis
    if (( $(echo "$avg_cpu > 80" | bc -l) )); then
        echo "  CPU: High utilization - consider reducing parallelism" >> "$report_file"
    elif (( $(echo "$avg_cpu < 30" | bc -l) )); then
        echo "  CPU: Low utilization - can increase parallelism" >> "$report_file"
    else
        echo "  CPU: Optimal utilization" >> "$report_file"
    fi
    
    if (( $(echo "$avg_memory > 85" | bc -l) )); then
        echo "  Memory: High usage - consider reducing GOMEMLIMIT" >> "$report_file"
    elif (( $(echo "$avg_memory < 50" | bc -l) )); then
        echo "  Memory: Low usage - can increase memory allocation" >> "$report_file"
    else
        echo "  Memory: Optimal usage" >> "$report_file"
    fi
    
    echo "" >> "$report_file"
    echo "Raw Data: $monitor_file" >> "$report_file"
    
    # Generate JSON report
    cat > "$json_report" << EOF
{
  "resource_monitoring_report": {
    "timestamp": "$(date -Iseconds)",
    "monitoring_file": "$monitor_file",
    "data_points": $(( $(wc -l < "$monitor_file") - 1 )),
    "statistics": {
      "cpu": {
        "average_percent": $avg_cpu,
        "peak_percent": $max_cpu
      },
      "memory": {
        "average_percent": $avg_memory,
        "peak_percent": $max_memory
      },
      "load_average": $avg_load
    },
    "recommendations": [
EOF
    
    # Add JSON recommendations
    local first_rec=true
    if (( $(echo "$avg_cpu > 80" | bc -l) )); then
        echo "      \"Reduce build parallelism to decrease CPU usage\"" >> "$json_report"
        first_rec=false
    fi
    
    if (( $(echo "$avg_memory > 85" | bc -l) )); then
        if [[ "$first_rec" != "true" ]]; then
            echo "," >> "$json_report"
        fi
        echo "      \"Reduce GOMEMLIMIT to prevent memory pressure\"" >> "$json_report"
        first_rec=false
    fi
    
    if [[ "$first_rec" == "true" ]]; then
        echo "      \"Resource usage is optimal\"" >> "$json_report"
    fi
    
    cat >> "$json_report" << EOF
    ]
  }
}
EOF
    
    log_success "Resource monitoring report generated"
    log_info "  Text report: $report_file"
    log_info "  JSON report: $json_report"
    
    # Display key statistics
    log_info "Resource usage summary:"
    log_info "  Average CPU: ${avg_cpu}%"
    log_info "  Peak CPU: ${max_cpu}%"
    log_info "  Average Memory: ${avg_memory}%"
    log_info "  Peak Memory: ${max_memory}%"
}

# Main function
main() {
    local action="${1:-optimize}"
    local build_mode="${2:-standard}"
    
    log_info "Starting Nephoran resource optimizer"
    log_info "  Action: $action"
    log_info "  Build mode: $build_mode"
    
    mkdir -p "$RESOURCE_MONITOR_DIR"
    
    case "$action" in
        "detect")
            detect_system_resources
            ;;
        "analyze")
            detect_system_resources
            ;;
        "optimize")
            detect_system_resources
            apply_resource_optimizations "$build_mode"
            ;;
        "monitor")
            local duration="${3:-300}"
            start_resource_monitoring "$duration"
            ;;
        "stop-monitor")
            stop_resource_monitoring
            ;;
        "report")
            if [[ -f "$RESOURCE_MONITOR_FILE" ]]; then
                generate_monitoring_report "$RESOURCE_MONITOR_FILE"
            else
                log_error "No monitoring data found"
                return 1
            fi
            ;;
        *)
            log_error "Unknown action: $action"
            echo "Usage: $0 {detect|analyze|optimize|monitor|stop-monitor|report} [build_mode] [duration]"
            return 1
            ;;
    esac
    
    log_success "Resource optimizer completed successfully"
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi