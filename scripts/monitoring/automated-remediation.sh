#!/bin/bash
# Nephoran Intent Operator - Automated Remediation Scripts
# TRL 9 Production-Ready Monitoring and Self-Healing Automation

set -euo pipefail

# Configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly CONFIG_FILE="${SCRIPT_DIR}/../../config/monitoring-config.yaml"
readonly LOG_FILE="/var/log/nephoran/automated-remediation.log"
readonly PROMETHEUS_URL="${PROMETHEUS_URL:-http://prometheus:9090}"
readonly ALERTMANAGER_URL="${ALERTMANAGER_URL:-http://alertmanager:9093}"
readonly KUBECTL_CONTEXT="${KUBECTL_CONTEXT:-nephoran-production}"
readonly NAMESPACE="${NAMESPACE:-nephoran-system}"

# Logging setup
exec 1> >(tee -a "${LOG_FILE}")
exec 2> >(tee -a "${LOG_FILE}" >&2)

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"
}

error() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $*" >&2
}

# Utility functions
check_prerequisites() {
    local missing_tools=()
    
    for tool in kubectl curl jq yq prometheus-query; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        error "Missing required tools: ${missing_tools[*]}"
        exit 1
    fi
    
    # Verify kubectl context
    if ! kubectl config current-context | grep -q "$KUBECTL_CONTEXT"; then
        error "Wrong kubectl context. Expected: $KUBECTL_CONTEXT"
        exit 1
    fi
}

query_prometheus() {
    local query="$1"
    local response
    
    response=$(curl -s -G "${PROMETHEUS_URL}/api/v1/query" --data-urlencode "query=${query}")
    echo "$response" | jq -r '.data.result[0].value[1] // "0"'
}

get_pod_status() {
    local deployment="$1"
    kubectl get deployment "$deployment" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}/{.spec.replicas}'
}

scale_deployment() {
    local deployment="$1"
    local replicas="$2"
    local max_wait="${3:-300}"
    
    log "Scaling $deployment to $replicas replicas"
    kubectl scale deployment "$deployment" --replicas="$replicas" -n "$NAMESPACE"
    
    # Wait for rollout to complete
    if kubectl rollout status deployment/"$deployment" -n "$NAMESPACE" --timeout="${max_wait}s"; then
        log "Successfully scaled $deployment to $replicas replicas"
        return 0
    else
        error "Failed to scale $deployment within ${max_wait} seconds"
        return 1
    fi
}

restart_deployment() {
    local deployment="$1"
    local max_wait="${2:-300}"
    
    log "Restarting deployment $deployment"
    kubectl rollout restart deployment/"$deployment" -n "$NAMESPACE"
    
    if kubectl rollout status deployment/"$deployment" -n "$NAMESPACE" --timeout="${max_wait}s"; then
        log "Successfully restarted $deployment"
        return 0
    else
        error "Failed to restart $deployment within ${max_wait} seconds"
        return 1
    fi
}

# Remediation functions for specific issues

remediate_high_memory_usage() {
    local deployment="$1"
    local current_memory_pct
    
    log "Remediating high memory usage for $deployment"
    
    current_memory_pct=$(query_prometheus "container_memory_working_set_bytes{pod=~\"${deployment}-.*\"} / container_spec_memory_limit_bytes{pod=~\"${deployment}-.*\"} * 100")
    
    if (( $(echo "$current_memory_pct > 90" | bc -l) )); then
        log "Memory usage critical (${current_memory_pct}%), restarting deployment"
        restart_deployment "$deployment"
    elif (( $(echo "$current_memory_pct > 80" | bc -l) )); then
        log "Memory usage high (${current_memory_pct}%), scaling up deployment"
        local current_replicas
        current_replicas=$(kubectl get deployment "$deployment" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
        local new_replicas=$((current_replicas + 1))
        scale_deployment "$deployment" "$new_replicas"
    fi
}

remediate_high_cpu_usage() {
    local deployment="$1"
    local current_cpu_pct
    
    log "Remediating high CPU usage for $deployment"
    
    current_cpu_pct=$(query_prometheus "rate(container_cpu_usage_seconds_total{pod=~\"${deployment}-.*\"}[5m]) * 100")
    
    if (( $(echo "$current_cpu_pct > 85" | bc -l) )); then
        log "CPU usage critical (${current_cpu_pct}%), scaling up deployment"
        local current_replicas
        current_replicas=$(kubectl get deployment "$deployment" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
        local max_replicas=10
        local new_replicas=$((current_replicas + 2))
        
        if (( new_replicas > max_replicas )); then
            new_replicas=$max_replicas
        fi
        
        scale_deployment "$deployment" "$new_replicas"
    fi
}

remediate_pod_crash_loop() {
    local deployment="$1"
    local crash_loops
    
    log "Remediating crash loop for $deployment"
    
    crash_loops=$(kubectl get pods -n "$NAMESPACE" -l "app=$deployment" --field-selector=status.phase=Failed -o name | wc -l)
    
    if (( crash_loops > 0 )); then
        log "Found $crash_loops crashed pods, cleaning up and restarting"
        
        # Delete failed pods
        kubectl delete pods -n "$NAMESPACE" -l "app=$deployment" --field-selector=status.phase=Failed
        
        # Wait a bit and check if the issue persists
        sleep 30
        
        crash_loops=$(kubectl get pods -n "$NAMESPACE" -l "app=$deployment" --field-selector=status.phase=Failed -o name | wc -l)
        
        if (( crash_loops > 0 )); then
            log "Crash loops persist, performing rollback"
            kubectl rollout undo deployment/"$deployment" -n "$NAMESPACE"
        fi
    fi
}

remediate_circuit_breaker_open() {
    local service="$1"
    local endpoint_url
    
    log "Remediating open circuit breaker for $service"
    
    case "$service" in
        "llm-processor")
            endpoint_url="http://llm-processor:8080"
            ;;
        "rag-api")
            endpoint_url="http://rag-api:8080"
            ;;
        *)
            error "Unknown service for circuit breaker remediation: $service"
            return 1
            ;;
    esac
    
    # Check service health
    if curl -f -s "${endpoint_url}/health" > /dev/null; then
        log "Service $service is healthy, resetting circuit breaker"
        curl -X POST "${endpoint_url}/admin/circuit-breaker/reset" || true
        
        # Gradually scale up
        log "Gradually scaling up $service after circuit breaker reset"
        local current_replicas
        current_replicas=$(kubectl get deployment "$service" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
        
        if (( current_replicas > 2 )); then
            scale_deployment "$service" 2
            sleep 60
            scale_deployment "$service" "$current_replicas"
        fi
    else
        log "Service $service is unhealthy, restarting deployment"
        restart_deployment "$service"
    fi
}

remediate_database_connection_issues() {
    log "Remediating database connection issues"
    
    # Check database connectivity
    if ! kubectl exec -n "$NAMESPACE" deployment/nephoran-db -- pg_isready -U nephoran; then
        log "Database is not ready, restarting database deployment"
        restart_deployment "nephoran-db"
        
        # Wait for database to be ready
        local retries=0
        while (( retries < 30 )); do
            if kubectl exec -n "$NAMESPACE" deployment/nephoran-db -- pg_isready -U nephoran; then
                log "Database is ready"
                break
            fi
            log "Waiting for database to be ready... (attempt $((retries + 1))/30)"
            sleep 10
            ((retries++))
        done
    fi
    
    # Check connection pool metrics
    local pool_usage
    pool_usage=$(query_prometheus "database_connection_pool_used / database_connection_pool_size * 100")
    
    if (( $(echo "$pool_usage > 90" | bc -l) )); then
        log "Connection pool usage critical (${pool_usage}%), increasing pool size"
        
        # Update connection pool configuration
        kubectl patch configmap/nephoran-config -n "$NAMESPACE" -p '{"data":{"db_max_connections":"200"}}'
        
        # Restart applications using the database
        restart_deployment "nephoran-intent-controller"
        restart_deployment "llm-processor"
        restart_deployment "rag-api"
    fi
}

remediate_oran_interface_failure() {
    local interface="$1"
    
    log "Remediating O-RAN interface failure for $interface"
    
    case "$interface" in
        "a1")
            # Check A1 interface connectivity to Near-RT RIC
            if ! curl -f -s "http://near-rt-ric:8080/a1-p/policies" > /dev/null; then
                log "Cannot connect to Near-RT RIC, checking network policies"
                
                # Apply emergency network policies
                kubectl apply -f "${SCRIPT_DIR}/../oran/emergency-network-policies.yaml"
                
                # Restart A1 interface pods
                restart_deployment "oran-a1-interface"
            fi
            ;;
        "e2")
            # Check E2 node connectivity
            local failed_nodes
            failed_nodes=$(kubectl get e2nodes -n "$NAMESPACE" --no-headers | awk '$2 != "Ready" {print $1}' | wc -l)
            
            if (( failed_nodes > 0 )); then
                log "Found $failed_nodes failed E2 nodes, recreating subscriptions"
                
                # Delete and recreate E2 subscriptions
                kubectl delete e2subscriptions --all -n "$NAMESPACE"
                sleep 10
                kubectl apply -f "${SCRIPT_DIR}/../oran/e2-subscriptions.yaml"
            fi
            ;;
        "o1")
            # Check O1 NETCONF connectivity
            if ! kubectl exec -n "$NAMESPACE" deployment/oran-o1-interface -- netconf-test-connection; then
                log "O1 NETCONF connection failed, restarting interface"
                restart_deployment "oran-o1-interface"
                
                # Clear any stuck NETCONF sessions
                kubectl exec -n "$NAMESPACE" deployment/oran-o1-interface -- netconf-clear-sessions
            fi
            ;;
    esac
}

remediate_nwdaf_analytics_failure() {
    log "Remediating NWDAF analytics failure"
    
    # Check data ingestion pipeline
    local ingestion_rate
    ingestion_rate=$(query_prometheus "rate(nwdaf_data_ingestion_total[5m])")
    
    if (( $(echo "$ingestion_rate < 10" | bc -l) )); then
        log "Data ingestion rate too low (${ingestion_rate}), restarting pipeline"
        
        # Restart data ingestion components
        kubectl rollout restart deployment/nwdaf-collector -n nwdaf-system
        kubectl rollout restart deployment/nwdaf-processor -n nwdaf-system
        
        # Clear stuck Kafka topics
        kubectl exec -n nwdaf-system deployment/kafka -- kafka-topics --bootstrap-server localhost:9092 --delete --topic ingestion-errors || true
    fi
    
    # Check ML model accuracy
    local model_accuracy
    model_accuracy=$(query_prometheus "nwdaf_ml_model_accuracy_ratio")
    
    if (( $(echo "$model_accuracy < 0.8" | bc -l) )); then
        log "ML model accuracy too low (${model_accuracy}), triggering retraining"
        
        # Trigger model retraining
        kubectl create job "nwdaf-retrain-$(date +%s)" --from=cronjob/nwdaf-model-training -n nwdaf-system
    fi
}

# Main alert processing function
process_alert() {
    local alert_name="$1"
    local alert_labels="$2"
    
    log "Processing alert: $alert_name"
    
    case "$alert_name" in
        "NephoranHighMemoryUsage")
            local deployment
            deployment=$(echo "$alert_labels" | jq -r '.deployment // "nephoran-intent-controller"')
            remediate_high_memory_usage "$deployment"
            ;;
        "NephoranHighCPUUsage")
            local deployment
            deployment=$(echo "$alert_labels" | jq -r '.deployment // "nephoran-intent-controller"')
            remediate_high_cpu_usage "$deployment"
            ;;
        "NephoranPodCrashLoop")
            local deployment
            deployment=$(echo "$alert_labels" | jq -r '.deployment // "nephoran-intent-controller"')
            remediate_pod_crash_loop "$deployment"
            ;;
        "LLMCircuitBreakerOpen")
            remediate_circuit_breaker_open "llm-processor"
            ;;
        "RAGCircuitBreakerOpen")
            remediate_circuit_breaker_open "rag-api"
            ;;
        "DatabaseConnectionPoolExhausted"|"DatabaseConnectionFailure")
            remediate_database_connection_issues
            ;;
        "A1InterfacePolicyViolation"|"A1InterfaceConnectionFailure")
            remediate_oran_interface_failure "a1"
            ;;
        "E2InterfaceSubscriptionFailure"|"E2InterfaceIndicationLoss")
            remediate_oran_interface_failure "e2"
            ;;
        "O1InterfaceFaultAlarmHigh"|"O1InterfaceConfigurationDrift")
            remediate_oran_interface_failure "o1"
            ;;
        "NWDAFDataIngestionFailure"|"NWDAFMLModelAccuracyLow")
            remediate_nwdaf_analytics_failure
            ;;
        *)
            log "No automated remediation available for alert: $alert_name"
            ;;
    esac
}

# Webhook handler for AlertManager
handle_webhook() {
    local webhook_data="$1"
    
    # Parse webhook data
    local alerts
    alerts=$(echo "$webhook_data" | jq -c '.alerts[]')
    
    while IFS= read -r alert; do
        local alert_name
        local alert_labels
        local alert_status
        
        alert_name=$(echo "$alert" | jq -r '.labels.alertname')
        alert_labels=$(echo "$alert" | jq -c '.labels')
        alert_status=$(echo "$alert" | jq -r '.status')
        
        if [[ "$alert_status" == "firing" ]]; then
            log "Processing firing alert: $alert_name"
            process_alert "$alert_name" "$alert_labels"
        fi
    done <<< "$alerts"
}

# Health check and periodic monitoring
run_health_checks() {
    log "Running periodic health checks"
    
    # Check critical deployments
    local critical_deployments=("nephoran-intent-controller" "llm-processor" "rag-api" "oran-a1-interface")
    
    for deployment in "${critical_deployments[@]}"; do
        local ready_pods
        ready_pods=$(get_pod_status "$deployment")
        
        if [[ "$ready_pods" =~ ^0/ ]]; then
            log "Deployment $deployment has no ready pods, attempting restart"
            restart_deployment "$deployment"
        fi
    done
    
    # Check system resource usage
    local system_memory_usage
    system_memory_usage=$(query_prometheus 'avg((1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100)')
    
    if (( $(echo "$system_memory_usage > 90" | bc -l) )); then
        log "System memory usage critical (${system_memory_usage}%), triggering cleanup"
        
        # Clean up completed pods
        kubectl delete pods -n "$NAMESPACE" --field-selector=status.phase=Succeeded
        kubectl delete pods -n "$NAMESPACE" --field-selector=status.phase=Failed
        
        # Force garbage collection
        kubectl get nodes -o name | xargs -I {} kubectl cordon {}
        sleep 30
        kubectl get nodes -o name | xargs -I {} kubectl uncordon {}
    fi
}

# Performance optimization
optimize_performance() {
    log "Running performance optimization"
    
    # Optimize based on current load
    local current_load
    current_load=$(query_prometheus 'rate(nephoran_intent_processing_total[5m])')
    
    if (( $(echo "$current_load > 300" | bc -l) )); then
        log "High load detected (${current_load} intents/sec), scaling up"
        
        # Scale up processing components
        local intent_controller_replicas
        intent_controller_replicas=$(kubectl get deployment nephoran-intent-controller -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
        
        if (( intent_controller_replicas < 5 )); then
            scale_deployment "nephoran-intent-controller" $((intent_controller_replicas + 1))
        fi
        
        # Scale up LLM processors if needed
        local llm_queue_depth
        llm_queue_depth=$(query_prometheus 'llm_request_queue_depth')
        
        if (( $(echo "$llm_queue_depth > 100" | bc -l) )); then
            local llm_replicas
            llm_replicas=$(kubectl get deployment llm-processor -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
            
            if (( llm_replicas < 8 )); then
                scale_deployment "llm-processor" $((llm_replicas + 1))
            fi
        fi
    elif (( $(echo "$current_load < 50" | bc -l) )); then
        log "Low load detected (${current_load} intents/sec), scaling down"
        
        # Scale down if we have excess capacity
        local intent_controller_replicas
        intent_controller_replicas=$(kubectl get deployment nephoran-intent-controller -n "$NAMESPACE" -o jsonpath='{.spec.replicas}')
        
        if (( intent_controller_replicas > 3 )); then
            scale_deployment "nephoran-intent-controller" $((intent_controller_replicas - 1))
        fi
    fi
}

# Cost optimization
optimize_costs() {
    log "Running cost optimization"
    
    # Check LLM token usage and optimize caching
    local token_cost_rate
    token_cost_rate=$(query_prometheus 'rate(llm_token_costs_usd_total[1h]) * 24')
    
    if (( $(echo "$token_cost_rate > 100" | bc -l) )); then
        log "High token cost rate detected (\$${token_cost_rate}/day), optimizing"
        
        # Increase cache TTL
        kubectl patch configmap/llm-config -n "$NAMESPACE" -p '{"data":{"cache_ttl":"3600"}}'
        
        # Enable batch processing
        kubectl patch configmap/llm-config -n "$NAMESPACE" -p '{"data":{"batch_processing":"enabled"}}'
        
        # Restart LLM processors to pick up config changes
        kubectl rollout restart deployment/llm-processor -n "$NAMESPACE"
    fi
}

# Main execution based on arguments
main() {
    local action="${1:-health-check}"
    
    log "Starting automated remediation script with action: $action"
    
    check_prerequisites
    
    case "$action" in
        "webhook")
            # Handle webhook from AlertManager
            local webhook_data
            webhook_data=$(cat)
            handle_webhook "$webhook_data"
            ;;
        "health-check")
            run_health_checks
            ;;
        "optimize-performance")
            optimize_performance
            ;;
        "optimize-costs")
            optimize_costs
            ;;
        "full-optimization")
            run_health_checks
            optimize_performance
            optimize_costs
            ;;
        *)
            error "Unknown action: $action"
            echo "Usage: $0 {webhook|health-check|optimize-performance|optimize-costs|full-optimization}"
            exit 1
            ;;
    esac
    
    log "Automated remediation completed successfully"
}

# Execute main function with all arguments
main "$@"