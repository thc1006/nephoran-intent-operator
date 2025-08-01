#!/bin/bash

# Nephoran Intent Operator - Production Monitoring Operations Script
# This script provides automated monitoring operations, health checks, and troubleshooting
# for the comprehensive monitoring stack implemented in the monitoring infrastructure.

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="${NEPHORAN_NAMESPACE:-nephoran-system}"
MONITORING_NAMESPACE="${MONITORING_NAMESPACE:-nephoran-monitoring}"
LOG_LEVEL="${LOG_LEVEL:-INFO}"
DRY_RUN="${DRY_RUN:-false}"

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    if [[ "$LOG_LEVEL" == "DEBUG" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $1"
    fi
}

# Health check functions
check_monitoring_stack() {
    log_info "Checking monitoring stack health..."
    
    local failed_components=0
    
    # Check Prometheus
    if ! kubectl get deployment prometheus -n "$MONITORING_NAMESPACE" &>/dev/null; then
        log_error "Prometheus deployment not found"
        failed_components=$((failed_components + 1))
    else
        local ready_replicas=$(kubectl get deployment prometheus -n "$MONITORING_NAMESPACE" -o jsonpath='{.status.readyReplicas}')
        local desired_replicas=$(kubectl get deployment prometheus -n "$MONITORING_NAMESPACE" -o jsonpath='{.spec.replicas}')
        if [[ "$ready_replicas" != "$desired_replicas" ]]; then
            log_error "Prometheus deployment not ready: $ready_replicas/$desired_replicas replicas ready"
            failed_components=$((failed_components + 1))
        else
            log_info "✓ Prometheus deployment healthy"
        fi
    fi
    
    # Check Grafana
    if ! kubectl get deployment grafana -n "$MONITORING_NAMESPACE" &>/dev/null; then
        log_error "Grafana deployment not found"
        failed_components=$((failed_components + 1))
    else
        local ready_replicas=$(kubectl get deployment grafana -n "$MONITORING_NAMESPACE" -o jsonpath='{.status.readyReplicas}')
        local desired_replicas=$(kubectl get deployment grafana -n "$MONITORING_NAMESPACE" -o jsonpath='{.spec.replicas}')
        if [[ "$ready_replicas" != "$desired_replicas" ]]; then
            log_error "Grafana deployment not ready: $ready_replicas/$desired_replicas replicas ready"
            failed_components=$((failed_components + 1))
        else
            log_info "✓ Grafana deployment healthy"
        fi
    fi
    
    # Check Jaeger
    if ! kubectl get deployment jaeger -n "$MONITORING_NAMESPACE" &>/dev/null; then
        log_error "Jaeger deployment not found"
        failed_components=$((failed_components + 1))
    else
        local ready_replicas=$(kubectl get deployment jaeger -n "$MONITORING_NAMESPACE" -o jsonpath='{.status.readyReplicas}')
        local desired_replicas=$(kubectl get deployment jaeger -n "$MONITORING_NAMESPACE" -o jsonpath='{.spec.replicas}')
        if [[ "$ready_replicas" != "$desired_replicas" ]]; then
            log_error "Jaeger deployment not ready: $ready_replicas/$desired_replicas replicas ready"
            failed_components=$((failed_components + 1))
        else
            log_info "✓ Jaeger deployment healthy"
        fi
    fi
    
    # Check AlertManager
    if ! kubectl get deployment alertmanager -n "$MONITORING_NAMESPACE" &>/dev/null; then
        log_error "AlertManager deployment not found"
        failed_components=$((failed_components + 1))
    else
        local ready_replicas=$(kubectl get deployment alertmanager -n "$MONITORING_NAMESPACE" -o jsonpath='{.status.readyReplicas}')
        local desired_replicas=$(kubectl get deployment alertmanager -n "$MONITORING_NAMESPACE" -o jsonpath='{.spec.replicas}')
        if [[ "$ready_replicas" != "$desired_replicas" ]]; then
            log_error "AlertManager deployment not ready: $ready_replicas/$desired_replicas replicas ready"
            failed_components=$((failed_components + 1))
        else
            log_info "✓ AlertManager deployment healthy"
        fi
    fi
    
    if [[ $failed_components -eq 0 ]]; then
        log_info "All monitoring components are healthy"
        return 0
    else
        log_error "$failed_components monitoring components are unhealthy"
        return 1
    fi
}

check_nephoran_services() {
    log_info "Checking Nephoran services health..."
    
    local failed_services=0
    local services=("llm-processor" "rag-api" "nephio-bridge" "oran-adaptor")
    
    for service in "${services[@]}"; do
        if ! kubectl get deployment "$service" -n "$NAMESPACE" &>/dev/null; then
            log_warn "Service $service deployment not found (may not be deployed yet)"
            continue
        fi
        
        local ready_replicas=$(kubectl get deployment "$service" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        local desired_replicas=$(kubectl get deployment "$service" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
        
        if [[ "$ready_replicas" != "$desired_replicas" ]] || [[ "$ready_replicas" == "0" ]]; then
            log_error "Service $service not ready: $ready_replicas/$desired_replicas replicas ready"
            failed_services=$((failed_services + 1))
        else
            log_info "✓ Service $service healthy"
        fi
    done
    
    if [[ $failed_services -eq 0 ]]; then
        log_info "All Nephoran services are healthy"
        return 0
    else
        log_error "$failed_services Nephoran services are unhealthy"
        return 1
    fi
}

check_prometheus_targets() {
    log_info "Checking Prometheus targets..."
    
    # Port-forward to Prometheus
    kubectl port-forward svc/prometheus 9090:9090 -n "$MONITORING_NAMESPACE" &
    local pf_pid=$!
    sleep 3
    
    # Query Prometheus targets
    local targets_response=$(curl -s "http://localhost:9090/api/v1/targets" 2>/dev/null || echo '{"data":{"activeTargets":[]}}')
    
    # Kill port-forward
    kill $pf_pid 2>/dev/null || true
    
    # Parse targets
    local active_targets=$(echo "$targets_response" | jq -r '.data.activeTargets | length' 2>/dev/null || echo "0")
    local healthy_targets=$(echo "$targets_response" | jq -r '.data.activeTargets | map(select(.health == "up")) | length' 2>/dev/null || echo "0")
    
    log_info "Prometheus targets: $healthy_targets/$active_targets healthy"
    
    if [[ "$active_targets" == "0" ]]; then
        log_warn "No Prometheus targets found - monitoring may not be configured"
        return 1
    elif [[ "$healthy_targets" != "$active_targets" ]]; then
        log_warn "Some Prometheus targets are unhealthy"
        return 1
    else
        log_info "✓ All Prometheus targets are healthy"
        return 0
    fi
}

# Troubleshooting functions
restart_monitoring_component() {
    local component=$1
    log_info "Restarting monitoring component: $component"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would restart deployment $component in namespace $MONITORING_NAMESPACE"
        return 0
    fi
    
    kubectl rollout restart deployment "$component" -n "$MONITORING_NAMESPACE"
    kubectl rollout status deployment "$component" -n "$MONITORING_NAMESPACE" --timeout=300s
    
    log_info "✓ Component $component restarted successfully"
}

restart_nephoran_service() {
    local service=$1
    log_info "Restarting Nephoran service: $service"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would restart deployment $service in namespace $NAMESPACE"
        return 0
    fi
    
    kubectl rollout restart deployment "$service" -n "$NAMESPACE"
    kubectl rollout status deployment "$service" -n "$NAMESPACE" --timeout=300s
    
    log_info "✓ Service $service restarted successfully"
}

fix_prometheus_config() {
    log_info "Attempting to fix Prometheus configuration..."
    
    # Check if prometheus-config ConfigMap exists
    if ! kubectl get configmap prometheus-config -n "$MONITORING_NAMESPACE" &>/dev/null; then
        log_error "Prometheus config ConfigMap not found"
        return 1
    fi
    
    # Restart Prometheus to reload config
    restart_monitoring_component "prometheus"
    
    log_info "✓ Prometheus configuration reload attempted"
}

scale_monitoring_component() {
    local component=$1
    local replicas=$2
    
    log_info "Scaling monitoring component $component to $replicas replicas"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would scale deployment $component to $replicas replicas"
        return 0
    fi
    
    kubectl scale deployment "$component" --replicas="$replicas" -n "$MONITORING_NAMESPACE"
    kubectl rollout status deployment "$component" -n "$MONITORING_NAMESPACE" --timeout=300s
    
    log_info "✓ Component $component scaled to $replicas replicas"
}

# Alert management functions
check_active_alerts() {
    log_info "Checking active alerts..."
    
    # Port-forward to AlertManager
    kubectl port-forward svc/alertmanager 9093:9093 -n "$MONITORING_NAMESPACE" &
    local pf_pid=$!
    sleep 3
    
    # Query AlertManager alerts
    local alerts_response=$(curl -s "http://localhost:9093/api/v1/alerts" 2>/dev/null || echo '{"data":[]}')
    
    # Kill port-forward
    kill $pf_pid 2>/dev/null || true
    
    # Parse alerts
    local active_alerts=$(echo "$alerts_response" | jq -r '.data | length' 2>/dev/null || echo "0")
    local firing_alerts=$(echo "$alerts_response" | jq -r '.data | map(select(.status.state == "active")) | length' 2>/dev/null || echo "0")
    
    log_info "Active alerts: $firing_alerts firing, $active_alerts total"
    
    if [[ "$firing_alerts" -gt "0" ]]; then
        log_warn "Active firing alerts detected:"
        echo "$alerts_response" | jq -r '.data[] | select(.status.state == "active") | "- \(.labels.alertname): \(.annotations.summary // .annotations.description // "No description")"' 2>/dev/null || log_warn "Could not parse alert details"
        return 1
    else
        log_info "✓ No firing alerts"
        return 0
    fi
}

silence_alert() {
    local alertname=$1
    local duration=${2:-"1h"}
    local comment=${3:-"Silenced by monitoring operations script"}
    
    log_info "Silencing alert $alertname for $duration"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would silence alert $alertname for $duration"
        return 0
    fi
    
    # Port-forward to AlertManager
    kubectl port-forward svc/alertmanager 9093:9093 -n "$MONITORING_NAMESPACE" &
    local pf_pid=$!
    sleep 3
    
    # Create silence
    local silence_data=$(cat <<EOF
{
  "matchers": [
    {
      "name": "alertname",
      "value": "$alertname",
      "isRegex": false
    }
  ],
  "startsAt": "$(date -u +%Y-%m-%dT%H:%M:%S.000Z)",
  "endsAt": "$(date -u -d "+$duration" +%Y-%m-%dT%H:%M:%S.000Z)",
  "createdBy": "monitoring-operations-script",
  "comment": "$comment"
}
EOF
)
    
    local response=$(curl -s -X POST "http://localhost:9093/api/v1/silences" \
        -H "Content-Type: application/json" \
        -d "$silence_data" 2>/dev/null || echo '{"silenceID":"error"}')
    
    # Kill port-forward
    kill $pf_pid 2>/dev/null || true
    
    local silence_id=$(echo "$response" | jq -r '.silenceID' 2>/dev/null || echo "error")
    
    if [[ "$silence_id" != "error" && "$silence_id" != "null" ]]; then
        log_info "✓ Alert $alertname silenced with ID: $silence_id"
        return 0
    else
        log_error "Failed to silence alert $alertname"
        return 1
    fi
}

# Metrics collection functions
collect_system_metrics() {
    log_info "Collecting system metrics..."
    
    local timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    local metrics_file="/tmp/nephoran-metrics-$timestamp.json"
    
    # Collect pod metrics
    local pod_metrics=$(kubectl top pods -n "$NAMESPACE" --no-headers 2>/dev/null | awk '{print "{\"pod\":\"" $1 "\",\"cpu\":\"" $2 "\",\"memory\":\"" $3 "\"}"}' | jq -s '.')
    
    # Collect deployment status
    local deployment_status=$(kubectl get deployments -n "$NAMESPACE" -o json | jq '.items | map({name: .metadata.name, ready: .status.readyReplicas, desired: .spec.replicas})')
    
    # Collect service status
    local service_status=$(kubectl get services -n "$NAMESPACE" -o json | jq '.items | map({name: .metadata.name, type: .spec.type, ports: .spec.ports | length})')
    
    # Create metrics report
    cat > "$metrics_file" <<EOF
{
  "timestamp": "$timestamp",
  "namespace": "$NAMESPACE",
  "pod_metrics": $pod_metrics,
  "deployment_status": $deployment_status,
  "service_status": $service_status
}
EOF
    
    log_info "System metrics collected in: $metrics_file"
    echo "$metrics_file"
}

# Performance analysis functions
analyze_performance() {
    log_info "Analyzing system performance..."
    
    # Check resource usage
    log_info "Resource usage analysis:"
    kubectl top pods -n "$NAMESPACE" --containers 2>/dev/null || log_warn "Could not get pod resource usage"
    
    # Check for OOMKilled pods
    local oom_pods=$(kubectl get pods -n "$NAMESPACE" -o json | jq -r '.items[] | select(.status.containerStatuses[]?.lastState.terminated.reason == "OOMKilled") | .metadata.name' 2>/dev/null)
    if [[ -n "$oom_pods" ]]; then
        log_warn "Pods killed due to OOM:"
        echo "$oom_pods"
    fi
    
    # Check for failed pods
    local failed_pods=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase=Failed -o json | jq -r '.items[] | .metadata.name' 2>/dev/null)
    if [[ -n "$failed_pods" ]]; then
        log_warn "Failed pods:"
        echo "$failed_pods"
    fi
    
    log_info "Performance analysis completed"
}

# Main operation functions
run_health_check() {
    log_info "Running comprehensive health check..."
    
    local overall_health=0
    
    if ! check_monitoring_stack; then
        overall_health=1
    fi
    
    if ! check_nephoran_services; then
        overall_health=1
    fi
    
    if ! check_prometheus_targets; then
        overall_health=1
    fi
    
    if ! check_active_alerts; then
        overall_health=1
    fi
    
    if [[ $overall_health -eq 0 ]]; then
        log_info "✅ Overall system health: HEALTHY"
    else
        log_error "❌ Overall system health: DEGRADED"
    fi
    
    return $overall_health
}

auto_repair() {
    log_info "Running automatic repair procedures..."
    
    local repairs_needed=0
    
    # Check and repair monitoring stack
    if ! check_monitoring_stack; then
        log_warn "Monitoring stack issues detected, attempting repairs..."
        
        # Restart unhealthy components
        for component in prometheus grafana jaeger alertmanager; do
            local ready_replicas=$(kubectl get deployment "$component" -n "$MONITORING_NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
            local desired_replicas=$(kubectl get deployment "$component" -n "$MONITORING_NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
            
            if [[ "$ready_replicas" != "$desired_replicas" ]]; then
                log_info "Repairing component: $component"
                restart_monitoring_component "$component"
                repairs_needed=1
            fi
        done
    fi
    
    # Check and repair Nephoran services
    if ! check_nephoran_services; then
        log_warn "Nephoran services issues detected, attempting repairs..."
        
        for service in llm-processor rag-api nephio-bridge oran-adaptor; do
            if kubectl get deployment "$service" -n "$NAMESPACE" &>/dev/null; then
                local ready_replicas=$(kubectl get deployment "$service" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
                local desired_replicas=$(kubectl get deployment "$service" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
                
                if [[ "$ready_replicas" != "$desired_replicas" ]] && [[ "$desired_replicas" != "0" ]]; then
                    log_info "Repairing service: $service"
                    restart_nephoran_service "$service"
                    repairs_needed=1
                fi
            fi
        done
    fi
    
    if [[ $repairs_needed -eq 0 ]]; then
        log_info "✅ No repairs needed"
    else
        log_info "🔧 Automatic repairs completed"
        
        # Wait for stabilization
        log_info "Waiting 30 seconds for system stabilization..."
        sleep 30
        
        # Re-run health check
        run_health_check
    fi
}

# Usage information
usage() {
    cat <<EOF
Nephoran Intent Operator - Monitoring Operations Script

Usage: $0 [COMMAND] [OPTIONS]

Commands:
  health-check          Run comprehensive health check
  auto-repair          Attempt automatic repair of detected issues
  restart-component    Restart a monitoring component
  restart-service      Restart a Nephoran service
  scale-component      Scale a monitoring component
  check-alerts         Check active alerts
  silence-alert        Silence an alert
  collect-metrics      Collect system metrics
  analyze-performance  Analyze system performance

Options:
  --namespace          Nephoran namespace (default: nephoran-system)
  --monitoring-ns      Monitoring namespace (default: nephoran-monitoring)
  --dry-run           Show what would be done without executing
  --debug             Enable debug logging
  --help              Show this help message

Examples:
  $0 health-check
  $0 auto-repair --dry-run
  $0 restart-component prometheus
  $0 restart-service llm-processor
  $0 scale-component grafana 2
  $0 silence-alert HighErrorRate 2h "Investigating issue"
  $0 collect-metrics
  $0 analyze-performance

Environment Variables:
  NEPHORAN_NAMESPACE    Nephoran namespace
  MONITORING_NAMESPACE  Monitoring namespace
  LOG_LEVEL            Logging level (INFO, DEBUG)
  DRY_RUN              Enable dry run mode (true/false)
EOF
}

# Main script logic
main() {
    case "${1:-}" in
        health-check)
            run_health_check
            ;;
        auto-repair)
            auto_repair
            ;;
        restart-component)
            if [[ -z "${2:-}" ]]; then
                log_error "Component name required"
                exit 1
            fi
            restart_monitoring_component "$2"
            ;;
        restart-service)
            if [[ -z "${2:-}" ]]; then
                log_error "Service name required"
                exit 1
            fi
            restart_nephoran_service "$2"
            ;;
        scale-component)
            if [[ -z "${2:-}" ]] || [[ -z "${3:-}" ]]; then
                log_error "Component name and replica count required"
                exit 1
            fi
            scale_monitoring_component "$2" "$3"
            ;;
        check-alerts)
            check_active_alerts
            ;;
        silence-alert)
            if [[ -z "${2:-}" ]]; then
                log_error "Alert name required"
                exit 1
            fi
            silence_alert "$2" "${3:-1h}" "${4:-Silenced by monitoring operations script}"
            ;;
        collect-metrics)
            collect_system_metrics
            ;;
        analyze-performance)
            analyze_performance
            ;;
        --help|help)
            usage
            exit 0
            ;;
        *)
            log_error "Unknown command: ${1:-}"
            usage
            exit 1
            ;;
    esac
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        --monitoring-ns)
            MONITORING_NAMESPACE="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN="true"
            shift
            ;;
        --debug)
            LOG_LEVEL="DEBUG"
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            break
            ;;
    esac
done

# Check prerequisites
if ! command -v kubectl &> /dev/null; then
    log_error "kubectl is required but not installed"
    exit 1
fi

if ! command -v jq &> /dev/null; then
    log_error "jq is required but not installed"
    exit 1
fi

# Run main function
main "$@"