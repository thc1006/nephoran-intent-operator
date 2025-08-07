#!/bin/bash
# Nephoran Intent Operator SLA Monitoring Script
# Version: 1.0.0
# Purpose: Real-time SLA tracking, performance regression detection, and automated reporting

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "${SCRIPT_DIR}")"
CONFIG_FILE="${PROJECT_ROOT}/deployments/monitoring/sla-monitor-config.yaml"
REPORT_DIR="${PROJECT_ROOT}/reports/sla"
LOG_DIR="${PROJECT_ROOT}/logs/sla-monitor"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
ALERT_MANAGER_URL="${ALERT_MANAGER_URL:-http://localhost:9093}"

# SLA Thresholds
AVAILABILITY_TARGET=99.9
INTENT_PROCESSING_P95_TARGET=30
ERROR_RATE_TARGET=0.5
RESOURCE_OPTIMIZATION_TARGET=20

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${timestamp} [${level}] ${message}" | tee -a "${LOG_DIR}/sla-monitor.log"
}

# Create necessary directories
setup_directories() {
    mkdir -p "${REPORT_DIR}"
    mkdir -p "${LOG_DIR}"
    mkdir -p "${PROJECT_ROOT}/tmp"
}

# Check dependencies
check_dependencies() {
    log "INFO" "Checking dependencies..."
    
    local deps=("curl" "jq" "bc" "awk")
    for dep in "${deps[@]}"; do
        if ! command -v "${dep}" &> /dev/null; then
            log "ERROR" "Required dependency '${dep}' not found"
            exit 1
        fi
    done
    
    # Check Prometheus connectivity
    if ! curl -s "${PROMETHEUS_URL}/api/v1/status/config" > /dev/null; then
        log "ERROR" "Cannot connect to Prometheus at ${PROMETHEUS_URL}"
        exit 1
    fi
    
    log "INFO" "All dependencies satisfied"
}

# Query Prometheus
query_prometheus() {
    local query="$1"
    local response
    
    response=$(curl -s -G "${PROMETHEUS_URL}/api/v1/query" \
        --data-urlencode "query=${query}" \
        --data-urlencode "time=$(date +%s)")
    
    if [[ $(echo "${response}" | jq -r '.status') != "success" ]]; then
        log "ERROR" "Prometheus query failed: ${query}"
        return 1
    fi
    
    echo "${response}" | jq -r '.data.result[0].value[1] // "0"'
}

# Query Prometheus range
query_prometheus_range() {
    local query="$1"
    local start="$2"
    local end="$3"
    local step="${4:-15s}"
    
    local response
    response=$(curl -s -G "${PROMETHEUS_URL}/api/v1/query_range" \
        --data-urlencode "query=${query}" \
        --data-urlencode "start=${start}" \
        --data-urlencode "end=${end}" \
        --data-urlencode "step=${step}")
    
    if [[ $(echo "${response}" | jq -r '.status') != "success" ]]; then
        log "ERROR" "Prometheus range query failed: ${query}"
        return 1
    fi
    
    echo "${response}"
}

# Check platform availability
check_availability() {
    log "INFO" "Checking platform availability..."
    
    local availability_query='avg_over_time(up{job="nephoran-operator"}[5m]) * 100'
    local availability
    availability=$(query_prometheus "${availability_query}")
    
    if (( $(echo "${availability} < ${AVAILABILITY_TARGET}" | bc -l) )); then
        log "ERROR" "Availability SLA violation: ${availability}% (target: ${AVAILABILITY_TARGET}%)"
        echo "CRITICAL" > "${PROJECT_ROOT}/tmp/sla_status"
        return 1
    elif (( $(echo "${availability} < 99.5" | bc -l) )); then
        log "WARNING" "Availability degraded: ${availability}% (target: ${AVAILABILITY_TARGET}%)"
        echo "DEGRADED" > "${PROJECT_ROOT}/tmp/sla_status"
    else
        log "INFO" "Availability OK: ${availability}%"
    fi
    
    echo "${availability}" > "${PROJECT_ROOT}/tmp/availability_current"
}

# Check intent processing performance
check_intent_processing() {
    log "INFO" "Checking intent processing performance..."
    
    local latency_query='histogram_quantile(0.95, sum(rate(intent_processing_duration_seconds_bucket[5m])) by (le))'
    local latency
    latency=$(query_prometheus "${latency_query}")
    
    if (( $(echo "${latency} > ${INTENT_PROCESSING_P95_TARGET}" | bc -l) )); then
        log "ERROR" "Intent processing P95 SLA violation: ${latency}s (target: ${INTENT_PROCESSING_P95_TARGET}s)"
        echo "CRITICAL" > "${PROJECT_ROOT}/tmp/sla_status"
        return 1
    elif (( $(echo "${latency} > 25" | bc -l) )); then
        log "WARNING" "Intent processing P95 degraded: ${latency}s (target: ${INTENT_PROCESSING_P95_TARGET}s)"
        echo "DEGRADED" > "${PROJECT_ROOT}/tmp/sla_status"
    else
        log "INFO" "Intent processing P95 OK: ${latency}s"
    fi
    
    echo "${latency}" > "${PROJECT_ROOT}/tmp/intent_latency_p95"
}

# Check error rate
check_error_rate() {
    log "INFO" "Checking error rate..."
    
    local error_rate_query='(sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))) * 100'
    local error_rate
    error_rate=$(query_prometheus "${error_rate_query}")
    
    if (( $(echo "${error_rate} > ${ERROR_RATE_TARGET}" | bc -l) )); then
        log "ERROR" "Error rate SLA violation: ${error_rate}% (target: ${ERROR_RATE_TARGET}%)"
        echo "CRITICAL" > "${PROJECT_ROOT}/tmp/sla_status"
        return 1
    else
        log "INFO" "Error rate OK: ${error_rate}%"
    fi
    
    echo "${error_rate}" > "${PROJECT_ROOT}/tmp/error_rate_current"
}

# Check resource optimization
check_resource_optimization() {
    log "INFO" "Checking resource optimization..."
    
    local cpu_efficiency_query='(1 - (sum(rate(container_cpu_usage_seconds_total{namespace="nephoran"}[5m])) / sum(kube_pod_container_resource_requests{namespace="nephoran",resource="cpu"}))) * 100'
    local cpu_efficiency
    cpu_efficiency=$(query_prometheus "${cpu_efficiency_query}")
    
    if (( $(echo "${cpu_efficiency} < ${RESOURCE_OPTIMIZATION_TARGET}" | bc -l) )); then
        log "WARNING" "Resource optimization below target: ${cpu_efficiency}% (target: ${RESOURCE_OPTIMIZATION_TARGET}%)"
        echo "DEGRADED" > "${PROJECT_ROOT}/tmp/sla_status"
    else
        log "INFO" "Resource optimization OK: ${cpu_efficiency}%"
    fi
    
    echo "${cpu_efficiency}" > "${PROJECT_ROOT}/tmp/resource_optimization"
}

# Check telecommunications KPIs
check_telecom_kpis() {
    log "INFO" "Checking telecommunications KPIs..."
    
    # Network function deployment time
    local nf_deployment_query='histogram_quantile(0.95, sum(rate(nf_deployment_duration_seconds_bucket[5m])) by (le))'
    local nf_deployment_time
    nf_deployment_time=$(query_prometheus "${nf_deployment_query}")
    
    if (( $(echo "${nf_deployment_time} > 180" | bc -l) )); then
        log "WARNING" "Network function deployment time high: ${nf_deployment_time}s (target: 180s)"
    else
        log "INFO" "Network function deployment time OK: ${nf_deployment_time}s"
    fi
    
    # O-RAN interface latency
    local oran_latency_query='histogram_quantile(0.95, sum(rate(oran_interface_latency_seconds_bucket[1m])) by (le)) * 1000'
    local oran_latency
    oran_latency=$(query_prometheus "${oran_latency_query}")
    
    if (( $(echo "${oran_latency} > 100" | bc -l) )); then
        log "WARNING" "O-RAN interface latency high: ${oran_latency}ms (target: 100ms)"
    else
        log "INFO" "O-RAN interface latency OK: ${oran_latency}ms"
    fi
    
    # RAN handover success rate
    local handover_success_query='(sum(rate(ran_handover_success_total[5m])) / sum(rate(ran_handover_attempts_total[5m]))) * 100'
    local handover_success_rate
    handover_success_rate=$(query_prometheus "${handover_success_query}")
    
    if (( $(echo "${handover_success_rate} < 99.5" | bc -l) )); then
        log "WARNING" "RAN handover success rate low: ${handover_success_rate}% (target: 99.5%)"
    else
        log "INFO" "RAN handover success rate OK: ${handover_success_rate}%"
    fi
    
    echo "${nf_deployment_time}" > "${PROJECT_ROOT}/tmp/nf_deployment_time"
    echo "${oran_latency}" > "${PROJECT_ROOT}/tmp/oran_latency"
    echo "${handover_success_rate}" > "${PROJECT_ROOT}/tmp/handover_success_rate"
}

# Check AI/ML performance
check_ai_ml_performance() {
    log "INFO" "Checking AI/ML performance..."
    
    # LLM token generation rate
    local token_rate_query='rate(llm_tokens_generated_total[1m])'
    local token_rate
    token_rate=$(query_prometheus "${token_rate_query}")
    
    if (( $(echo "${token_rate} < 50" | bc -l) )); then
        log "WARNING" "LLM token generation rate low: ${token_rate}/s (target: 50/s)"
    else
        log "INFO" "LLM token generation rate OK: ${token_rate}/s"
    fi
    
    # RAG retrieval latency
    local rag_latency_query='histogram_quantile(0.95, sum(rate(rag_retrieval_duration_seconds_bucket[5m])) by (le)) * 1000'
    local rag_latency
    rag_latency=$(query_prometheus "${rag_latency_query}")
    
    if (( $(echo "${rag_latency} > 200" | bc -l) )); then
        log "WARNING" "RAG retrieval latency high: ${rag_latency}ms (target: 200ms)"
    else
        log "INFO" "RAG retrieval latency OK: ${rag_latency}ms"
    fi
    
    # RAG cache hit rate
    local cache_hit_rate_query='(sum(rate(rag_cache_hits_total[5m])) / sum(rate(rag_requests_total[5m]))) * 100'
    local cache_hit_rate
    cache_hit_rate=$(query_prometheus "${cache_hit_rate_query}")
    
    if (( $(echo "${cache_hit_rate} < 75" | bc -l) )); then
        log "WARNING" "RAG cache hit rate low: ${cache_hit_rate}% (target: 75%)"
    else
        log "INFO" "RAG cache hit rate OK: ${cache_hit_rate}%"
    fi
    
    echo "${token_rate}" > "${PROJECT_ROOT}/tmp/token_rate"
    echo "${rag_latency}" > "${PROJECT_ROOT}/tmp/rag_latency"
    echo "${cache_hit_rate}" > "${PROJECT_ROOT}/tmp/cache_hit_rate"
}

# Performance regression detection
detect_regressions() {
    log "INFO" "Detecting performance regressions..."
    
    local current_time=$(date +%s)
    local one_hour_ago=$((current_time - 3600))
    local one_day_ago=$((current_time - 86400))
    
    # Compare current vs 1 hour ago
    local current_latency=$(query_prometheus 'histogram_quantile(0.95, sum(rate(intent_processing_duration_seconds_bucket[5m])) by (le))')
    local historical_response
    historical_response=$(query_prometheus_range 'histogram_quantile(0.95, sum(rate(intent_processing_duration_seconds_bucket[5m])) by (le))' "${one_hour_ago}" "${one_hour_ago}" "15s")
    
    local historical_latency
    historical_latency=$(echo "${historical_response}" | jq -r '.data.result[0].values[0][1] // "0"')
    
    if (( $(echo "${current_latency} > ${historical_latency} * 1.2" | bc -l) )); then
        log "WARNING" "Performance regression detected: P95 latency increased from ${historical_latency}s to ${current_latency}s (20% increase)"
        echo "REGRESSION" > "${PROJECT_ROOT}/tmp/regression_status"
        return 1
    fi
    
    log "INFO" "No performance regressions detected"
    echo "OK" > "${PROJECT_ROOT}/tmp/regression_status"
}

# Calculate composite SLA scores
calculate_composite_scores() {
    log "INFO" "Calculating composite SLA scores..."
    
    local availability=$(cat "${PROJECT_ROOT}/tmp/availability_current" 2>/dev/null || echo "0")
    local intent_latency=$(cat "${PROJECT_ROOT}/tmp/intent_latency_p95" 2>/dev/null || echo "0")
    local error_rate=$(cat "${PROJECT_ROOT}/tmp/error_rate_current" 2>/dev/null || echo "0")
    local resource_optimization=$(cat "${PROJECT_ROOT}/tmp/resource_optimization" 2>/dev/null || echo "0")
    
    # Platform health score (availability: 30%, performance: 25%, error rate: 25%, resource optimization: 20%)
    local platform_score
    platform_score=$(echo "scale=2; \
        (${availability} * 0.3) + \
        ((1 - ${intent_latency} / 30) * 25 * 0.25) + \
        ((1 - ${error_rate} / 0.5) * 25 * 0.25) + \
        (${resource_optimization} * 0.2)" | bc -l)
    
    log "INFO" "Platform health score: ${platform_score}%"
    echo "${platform_score}" > "${PROJECT_ROOT}/tmp/platform_health_score"
    
    # Telecom service quality score
    local nf_deployment_time=$(cat "${PROJECT_ROOT}/tmp/nf_deployment_time" 2>/dev/null || echo "0")
    local oran_latency=$(cat "${PROJECT_ROOT}/tmp/oran_latency" 2>/dev/null || echo "0")
    local handover_success_rate=$(cat "${PROJECT_ROOT}/tmp/handover_success_rate" 2>/dev/null || echo "0")
    
    local telecom_score
    telecom_score=$(echo "scale=2; \
        ((1 - ${nf_deployment_time} / 180) * 30) + \
        ((1 - ${oran_latency} / 100) * 30) + \
        (${handover_success_rate} * 0.4)" | bc -l)
    
    log "INFO" "Telecom service quality score: ${telecom_score}%"
    echo "${telecom_score}" > "${PROJECT_ROOT}/tmp/telecom_quality_score"
}

# Generate SLA report
generate_report() {
    log "INFO" "Generating SLA report..."
    
    local timestamp=$(date '+%Y-%m-%d_%H-%M-%S')
    local report_file="${REPORT_DIR}/sla-report-${timestamp}.json"
    local summary_file="${REPORT_DIR}/sla-summary-latest.json"
    
    # Read current values
    local availability=$(cat "${PROJECT_ROOT}/tmp/availability_current" 2>/dev/null || echo "0")
    local intent_latency=$(cat "${PROJECT_ROOT}/tmp/intent_latency_p95" 2>/dev/null || echo "0")
    local error_rate=$(cat "${PROJECT_ROOT}/tmp/error_rate_current" 2>/dev/null || echo "0")
    local resource_optimization=$(cat "${PROJECT_ROOT}/tmp/resource_optimization" 2>/dev/null || echo "0")
    local platform_score=$(cat "${PROJECT_ROOT}/tmp/platform_health_score" 2>/dev/null || echo "0")
    local telecom_score=$(cat "${PROJECT_ROOT}/tmp/telecom_quality_score" 2>/dev/null || echo "0")
    local sla_status=$(cat "${PROJECT_ROOT}/tmp/sla_status" 2>/dev/null || echo "OK")
    local regression_status=$(cat "${PROJECT_ROOT}/tmp/regression_status" 2>/dev/null || echo "OK")
    
    # Generate JSON report
    cat > "${report_file}" << EOF
{
  "timestamp": "$(date -u '+%Y-%m-%dT%H:%M:%SZ')",
  "report_version": "1.0.0",
  "sla_compliance": {
    "overall_status": "${sla_status}",
    "regression_status": "${regression_status}",
    "platform_slas": {
      "availability": {
        "current": ${availability},
        "target": ${AVAILABILITY_TARGET},
        "status": "$([ $(echo "${availability} >= ${AVAILABILITY_TARGET}" | bc -l) -eq 1 ] && echo "OK" || echo "VIOLATION")"
      },
      "intent_processing_p95": {
        "current": ${intent_latency},
        "target": ${INTENT_PROCESSING_P95_TARGET},
        "status": "$([ $(echo "${intent_latency} <= ${INTENT_PROCESSING_P95_TARGET}" | bc -l) -eq 1 ] && echo "OK" || echo "VIOLATION")"
      },
      "error_rate": {
        "current": ${error_rate},
        "target": ${ERROR_RATE_TARGET},
        "status": "$([ $(echo "${error_rate} <= ${ERROR_RATE_TARGET}" | bc -l) -eq 1 ] && echo "OK" || echo "VIOLATION")"
      },
      "resource_optimization": {
        "current": ${resource_optimization},
        "target": ${RESOURCE_OPTIMIZATION_TARGET},
        "status": "$([ $(echo "${resource_optimization} >= ${RESOURCE_OPTIMIZATION_TARGET}" | bc -l) -eq 1 ] && echo "OK" || echo "BELOW_TARGET")"
      }
    },
    "telecom_kpis": {
      "nf_deployment_time": $(cat "${PROJECT_ROOT}/tmp/nf_deployment_time" 2>/dev/null || echo "0"),
      "oran_latency": $(cat "${PROJECT_ROOT}/tmp/oran_latency" 2>/dev/null || echo "0"),
      "handover_success_rate": $(cat "${PROJECT_ROOT}/tmp/handover_success_rate" 2>/dev/null || echo "0")
    },
    "ai_ml_performance": {
      "token_rate": $(cat "${PROJECT_ROOT}/tmp/token_rate" 2>/dev/null || echo "0"),
      "rag_latency": $(cat "${PROJECT_ROOT}/tmp/rag_latency" 2>/dev/null || echo "0"),
      "cache_hit_rate": $(cat "${PROJECT_ROOT}/tmp/cache_hit_rate" 2>/dev/null || echo "0")
    },
    "composite_scores": {
      "platform_health": ${platform_score},
      "telecom_quality": ${telecom_score}
    }
  },
  "recommendations": []
}
EOF
    
    # Copy to summary file
    cp "${report_file}" "${summary_file}"
    
    log "INFO" "SLA report generated: ${report_file}"
}

# Send alerts if needed
send_alerts() {
    local sla_status=$(cat "${PROJECT_ROOT}/tmp/sla_status" 2>/dev/null || echo "OK")
    
    if [[ "${sla_status}" == "CRITICAL" ]]; then
        log "ERROR" "Critical SLA violations detected - alerting required"
        # In a real implementation, this would integrate with PagerDuty, Slack, etc.
        
    elif [[ "${sla_status}" == "DEGRADED" ]]; then
        log "WARNING" "SLA degradation detected - monitoring closely"
    fi
}

# Display dashboard URL
show_dashboard_urls() {
    echo -e "\n${BLUE}SLA Monitoring Dashboards:${NC}"
    echo "  Compliance: ${GRAFANA_URL}/d/nephoran-sla-compliance"
    echo "  Performance Trends: ${GRAFANA_URL}/d/nephoran-performance-trends"
    echo "  Capacity Planning: ${GRAFANA_URL}/d/nephoran-capacity-planning"
    echo "  Cost Optimization: ${GRAFANA_URL}/d/nephoran-cost-optimization"
    echo ""
    echo -e "${BLUE}Latest Report:${NC} ${REPORT_DIR}/sla-summary-latest.json"
}

# Trend analysis
analyze_trends() {
    log "INFO" "Analyzing performance trends..."
    
    local current_time=$(date +%s)
    local twenty_four_hours_ago=$((current_time - 86400))
    
    # Get 24-hour trend for key metrics
    local trend_response
    trend_response=$(query_prometheus_range 'histogram_quantile(0.95, sum(rate(intent_processing_duration_seconds_bucket[5m])) by (le))' "${twenty_four_hours_ago}" "${current_time}" "1h")
    
    if [[ $? -eq 0 ]]; then
        local trend_file="${PROJECT_ROOT}/tmp/latency_trend_24h.json"
        echo "${trend_response}" > "${trend_file}"
        
        # Simple trend analysis - check if latency is increasing
        local first_value=$(echo "${trend_response}" | jq -r '.data.result[0].values[0][1] // "0"')
        local last_value=$(echo "${trend_response}" | jq -r '.data.result[0].values[-1][1] // "0"')
        
        if (( $(echo "${last_value} > ${first_value} * 1.1" | bc -l) )); then
            log "WARNING" "Increasing latency trend detected over 24 hours: ${first_value}s -> ${last_value}s"
        else
            log "INFO" "Latency trend stable over 24 hours"
        fi
    fi
}

# Main monitoring loop
main() {
    echo -e "${BLUE}Nephoran Intent Operator SLA Monitor${NC}"
    echo -e "${BLUE}=====================================${NC}\n"
    
    setup_directories
    check_dependencies
    
    # Initialize SLA status
    echo "OK" > "${PROJECT_ROOT}/tmp/sla_status"
    
    # Run all checks
    check_availability || true
    check_intent_processing || true
    check_error_rate || true
    check_resource_optimization || true
    check_telecom_kpis || true
    check_ai_ml_performance || true
    
    # Analysis and reporting
    detect_regressions || true
    analyze_trends || true
    calculate_composite_scores
    generate_report
    send_alerts
    
    # Display results
    local sla_status=$(cat "${PROJECT_ROOT}/tmp/sla_status" 2>/dev/null || echo "OK")
    case "${sla_status}" in
        "OK")
            echo -e "\n${GREEN}✅ All SLAs are within acceptable limits${NC}"
            ;;
        "DEGRADED")
            echo -e "\n${YELLOW}⚠️  SLA degradation detected${NC}"
            ;;
        "CRITICAL")
            echo -e "\n${RED}🚨 Critical SLA violations detected${NC}"
            ;;
    esac
    
    show_dashboard_urls
    
    # Exit with appropriate code
    case "${sla_status}" in
        "CRITICAL") exit 2 ;;
        "DEGRADED") exit 1 ;;
        *) exit 0 ;;
    esac
}

# Handle script arguments
case "${1:-monitor}" in
    "monitor")
        main
        ;;
    "report")
        generate_report
        ;;
    "trends")
        analyze_trends
        ;;
    "dashboards")
        show_dashboard_urls
        ;;
    "--help"|"-h")
        echo "Usage: $0 [monitor|report|trends|dashboards|--help]"
        echo ""
        echo "Commands:"
        echo "  monitor     Run full SLA monitoring (default)"
        echo "  report      Generate SLA report only"
        echo "  trends      Analyze performance trends"
        echo "  dashboards  Show dashboard URLs"
        echo "  --help      Show this help message"
        ;;
    *)
        echo "Unknown command: $1"
        echo "Use '$0 --help' for usage information"
        exit 1
        ;;
esac