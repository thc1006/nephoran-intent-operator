#!/bin/bash

# Nephoran Intent Operator - Blue-Green Deployment Script
# Advanced deployment strategy for zero-downtime deployments

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
LOG_FILE="/tmp/nephoran-blue-green-$(date +%Y%m%d-%H%M%S).log"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default values
NAMESPACE="${NAMESPACE:-nephoran-system}"
RELEASE_NAME="${RELEASE_NAME:-nephoran-operator}"
HELM_CHART_PATH="${HELM_CHART_PATH:-${PROJECT_ROOT}/deployments/helm/nephoran-operator}"
VALUES_FILE="${VALUES_FILE:-}"
HEALTH_CHECK_TIMEOUT="${HEALTH_CHECK_TIMEOUT:-300}"
TRAFFIC_SPLIT_DELAY="${TRAFFIC_SPLIT_DELAY:-30}"
CLEANUP_DELAY="${CLEANUP_DELAY:-300}"
DRY_RUN="${DRY_RUN:-false}"
FORCE="${FORCE:-false}"
MONITORING_ENABLED="${MONITORING_ENABLED:-true}"

# Logging function
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    case "$level" in
        INFO)
            echo -e "${GREEN}[INFO]${NC} $message" | tee -a "$LOG_FILE"
            ;;
        WARN)
            echo -e "${YELLOW}[WARN]${NC} $message" | tee -a "$LOG_FILE"
            ;;
        ERROR)
            echo -e "${RED}[ERROR]${NC} $message" | tee -a "$LOG_FILE"
            ;;
        SUCCESS)
            echo -e "${GREEN}[SUCCESS]${NC} $message" | tee -a "$LOG_FILE"
            ;;
    esac
    
    echo "[$timestamp] [$level] $message" >> "$LOG_FILE"
}

# Usage function
usage() {
    cat << EOF
Nephoran Intent Operator - Blue-Green Deployment

Usage: $0 [OPTIONS]

OPTIONS:
    -n, --namespace NAMESPACE         Kubernetes namespace [default: $NAMESPACE]
    -r, --release RELEASE_NAME        Helm release name [default: $RELEASE_NAME]
    -c, --chart PATH                  Path to Helm chart [default: $HELM_CHART_PATH]
    -f, --values FILE                 Values file for Helm chart
    --health-timeout SECONDS          Health check timeout [default: $HEALTH_CHECK_TIMEOUT]
    --traffic-delay SECONDS           Delay before traffic switch [default: $TRAFFIC_SPLIT_DELAY]
    --cleanup-delay SECONDS           Delay before cleanup [default: $CLEANUP_DELAY]
    --dry-run                         Show what would be done without executing
    --force                           Skip confirmation prompts
    --no-monitoring                   Disable monitoring integration
    -h, --help                        Show this help

EXAMPLES:
    # Basic blue-green deployment
    $0 -f deployments/helm/nephoran-operator/environments/values-production.yaml

    # Blue-green deployment with custom timeouts
    $0 --health-timeout 600 --traffic-delay 60 --cleanup-delay 900

    # Dry run to see planned actions
    $0 --dry-run -f values-production.yaml

EOF
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -r|--release)
                RELEASE_NAME="$2"
                shift 2
                ;;
            -c|--chart)
                HELM_CHART_PATH="$2"
                shift 2
                ;;
            -f|--values)
                VALUES_FILE="$2"
                shift 2
                ;;
            --health-timeout)
                HEALTH_CHECK_TIMEOUT="$2"
                shift 2
                ;;
            --traffic-delay)
                TRAFFIC_SPLIT_DELAY="$2"
                shift 2
                ;;
            --cleanup-delay)
                CLEANUP_DELAY="$2"
                shift 2
                ;;
            --dry-run)
                DRY_RUN="true"
                shift
                ;;
            --force)
                FORCE="true"
                shift
                ;;
            --no-monitoring)
                MONITORING_ENABLED="false"
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                log ERROR "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Validation
validate_prerequisites() {
    log INFO "Validating prerequisites..."
    
    # Check required tools
    local required_tools=("kubectl" "helm" "jq" "curl")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log ERROR "Required tool not found: $tool"
            exit 1
        fi
    done
    
    # Check Kubernetes connection
    if ! kubectl cluster-info &> /dev/null; then
        log ERROR "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check Helm chart exists
    if [[ ! -d "$HELM_CHART_PATH" ]]; then
        log ERROR "Helm chart not found: $HELM_CHART_PATH"
        exit 1
    fi
    
    # Check values file if specified
    if [[ -n "$VALUES_FILE" ]] && [[ ! -f "$VALUES_FILE" ]]; then
        log ERROR "Values file not found: $VALUES_FILE"
        exit 1
    fi
    
    log SUCCESS "Prerequisites validated"
}

# Get current active color
get_active_color() {
    local current_color=""
    
    # Check if blue service exists and is active
    if kubectl get service "${RELEASE_NAME}-blue" -n "$NAMESPACE" &> /dev/null; then
        local blue_selector=$(kubectl get service "${RELEASE_NAME}-blue" -n "$NAMESPACE" -o jsonpath='{.spec.selector.color}' 2>/dev/null)
        local main_selector=$(kubectl get service "${RELEASE_NAME}" -n "$NAMESPACE" -o jsonpath='{.spec.selector.color}' 2>/dev/null)
        
        if [[ "$blue_selector" == "$main_selector" ]]; then
            current_color="blue"
        fi
    fi
    
    # Check if green service exists and is active
    if kubectl get service "${RELEASE_NAME}-green" -n "$NAMESPACE" &> /dev/null; then
        local green_selector=$(kubectl get service "${RELEASE_NAME}-green" -n "$NAMESPACE" -o jsonpath='{.spec.selector.color}' 2>/dev/null)
        local main_selector=$(kubectl get service "${RELEASE_NAME}" -n "$NAMESPACE" -o jsonpath='{.spec.selector.color}' 2>/dev/null)
        
        if [[ "$green_selector" == "$main_selector" ]]; then
            current_color="green"
        fi
    fi
    
    # Default to blue if no active color found
    if [[ -z "$current_color" ]]; then
        current_color="blue"
    fi
    
    echo "$current_color"
}

# Deploy to specific color
deploy_to_color() {
    local target_color="$1"
    local release_name="${RELEASE_NAME}-${target_color}"
    
    log INFO "Deploying to $target_color environment..."
    
    # Prepare Helm values with color-specific settings
    local temp_values=$(mktemp)
    
    if [[ -n "$VALUES_FILE" ]]; then
        cp "$VALUES_FILE" "$temp_values"
    else
        echo "{}" > "$temp_values"
    fi
    
    # Add blue-green specific values
    cat >> "$temp_values" << EOF

# Blue-Green Deployment Configuration
blueGreen:
  enabled: true
  color: "${target_color}"

# Service name suffix for blue-green
nameOverride: "${RELEASE_NAME}-${target_color}"
fullnameOverride: "${RELEASE_NAME}-${target_color}"

# Labels for blue-green deployment
commonLabels:
  color: "${target_color}"
  deployment-strategy: "blue-green"
  release: "${RELEASE_NAME}"

# Pod labels
podLabels:
  color: "${target_color}"
  deployment-strategy: "blue-green"
  
# Service selector
service:
  selector:
    color: "${target_color}"

# Unique services for each color
services:
  main:
    name: "${RELEASE_NAME}-${target_color}"
    selector:
      color: "${target_color}"
  
# Resource naming
resources:
  namePrefix: "${target_color}-"
EOF
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would deploy to $target_color with:"
        log INFO "Release: $release_name"
        log INFO "Namespace: $NAMESPACE"
        log INFO "Chart: $HELM_CHART_PATH"
        log INFO "Values: $temp_values"
        
        helm template "$release_name" "$HELM_CHART_PATH" \
            --namespace "$NAMESPACE" \
            --values "$temp_values" \
            --dry-run
    else
        # Check if release exists
        if helm list -n "$NAMESPACE" | grep -q "$release_name"; then
            log INFO "Upgrading existing $target_color release..."
            helm upgrade "$release_name" "$HELM_CHART_PATH" \
                --namespace "$NAMESPACE" \
                --values "$temp_values" \
                --wait --timeout="${HEALTH_CHECK_TIMEOUT}s" \
                --create-namespace
        else
            log INFO "Installing new $target_color release..."
            helm install "$release_name" "$HELM_CHART_PATH" \
                --namespace "$NAMESPACE" \
                --values "$temp_values" \
                --wait --timeout="${HEALTH_CHECK_TIMEOUT}s" \
                --create-namespace
        fi
    fi
    
    # Cleanup temp file
    rm -f "$temp_values"
    
    log SUCCESS "Deployment to $target_color completed"
}

# Health check for specific color
health_check_color() {
    local color="$1"
    local service_name="${RELEASE_NAME}-${color}"
    
    log INFO "Performing health check for $color environment..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would perform health check for $color"
        return 0
    fi
    
    # Check if service exists
    if ! kubectl get service "$service_name" -n "$NAMESPACE" &> /dev/null; then
        log ERROR "Service $service_name not found"
        return 1
    fi
    
    # Check if pods are ready
    local ready_pods=0
    local total_pods=0
    local max_attempts=30
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        local pod_status=$(kubectl get pods -n "$NAMESPACE" -l "color=$color" -o json)
        total_pods=$(echo "$pod_status" | jq '.items | length')
        ready_pods=$(echo "$pod_status" | jq '[.items[] | select(.status.conditions[]? | select(.type=="Ready" and .status=="True"))] | length')
        
        if [[ $ready_pods -gt 0 ]] && [[ $ready_pods -eq $total_pods ]]; then
            log SUCCESS "All $ready_pods pods are ready for $color environment"
            break
        fi
        
        log INFO "Waiting for pods to be ready: $ready_pods/$total_pods ready"
        sleep 10
        ((attempt++))
    done
    
    if [[ $ready_pods -eq 0 ]] || [[ $ready_pods -ne $total_pods ]]; then
        log ERROR "Health check failed for $color environment: $ready_pods/$total_pods pods ready"
        return 1
    fi
    
    # Perform application-level health checks
    log INFO "Performing application health checks..."
    
    # Port forward and check health endpoints
    local health_endpoints=("health" "ready" "metrics")
    local service_port="8080"
    
    kubectl port-forward "svc/$service_name" "$service_port:$service_port" -n "$NAMESPACE" &> /dev/null &
    local port_forward_pid=$!
    
    sleep 5
    
    local health_passed=true
    for endpoint in "${health_endpoints[@]}"; do
        if curl -f "http://localhost:$service_port/$endpoint" --max-time 10 &> /dev/null; then
            log SUCCESS "Health check passed for endpoint: /$endpoint"
        else
            log WARN "Health check failed for endpoint: /$endpoint"
            if [[ "$endpoint" == "health" ]]; then
                health_passed=false
            fi
        fi
    done
    
    # Cleanup port forward
    kill $port_forward_pid 2>/dev/null || true
    
    if [[ "$health_passed" == "true" ]]; then
        log SUCCESS "Health check passed for $color environment"
        return 0
    else
        log ERROR "Critical health checks failed for $color environment"
        return 1
    fi
}

# Switch traffic to new color
switch_traffic() {
    local new_color="$1"
    local service_name="$RELEASE_NAME"
    local color_service="${RELEASE_NAME}-${new_color}"
    
    log INFO "Switching traffic to $new_color environment..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would switch traffic to $new_color"
        return 0
    fi
    
    # Update main service selector to point to new color
    kubectl patch service "$service_name" -n "$NAMESPACE" \
        --type='merge' \
        -p="{\"spec\":{\"selector\":{\"color\":\"$new_color\"}}}"
    
    # Verify traffic switch
    local updated_selector=$(kubectl get service "$service_name" -n "$NAMESPACE" -o jsonpath='{.spec.selector.color}')
    
    if [[ "$updated_selector" == "$new_color" ]]; then
        log SUCCESS "Traffic successfully switched to $new_color"
        
        # Update service labels for tracking
        kubectl label service "$service_name" -n "$NAMESPACE" \
            "active-color=$new_color" \
            "last-switch=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
            --overwrite
    else
        log ERROR "Failed to switch traffic to $new_color"
        return 1
    fi
}

# Cleanup old color deployment
cleanup_old_deployment() {
    local old_color="$1"
    local release_name="${RELEASE_NAME}-${old_color}"
    
    log INFO "Cleaning up $old_color environment (waiting ${CLEANUP_DELAY}s)..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would cleanup $old_color deployment after ${CLEANUP_DELAY}s"
        return 0
    fi
    
    # Wait before cleanup to allow for quick rollback if needed
    if [[ "$FORCE" != "true" ]]; then
        log INFO "Waiting ${CLEANUP_DELAY} seconds before cleanup (allows for quick rollback)..."
        sleep "$CLEANUP_DELAY"
    fi
    
    # Check if we should proceed with cleanup
    if [[ "$FORCE" != "true" ]]; then
        read -p "Proceed with cleanup of $old_color environment? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log WARN "Cleanup cancelled by user"
            return 0
        fi
    fi
    
    # Scale down old deployment
    log INFO "Scaling down $old_color deployment..."
    kubectl scale deployment -l "color=$old_color" --replicas=0 -n "$NAMESPACE"
    
    # Wait a bit then delete
    sleep 30
    
    # Delete old Helm release
    if helm list -n "$NAMESPACE" | grep -q "$release_name"; then
        log INFO "Uninstalling $old_color Helm release..."
        helm uninstall "$release_name" -n "$NAMESPACE"
    fi
    
    log SUCCESS "Cleanup of $old_color environment completed"
}

# Rollback to previous color
rollback_deployment() {
    local current_color="$1"
    local previous_color="$2"
    
    log WARN "Rolling back from $current_color to $previous_color..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would rollback to $previous_color"
        return 0
    fi
    
    # Switch traffic back
    if switch_traffic "$previous_color"; then
        log SUCCESS "Rollback completed - traffic restored to $previous_color"
        
        # Optionally cleanup failed deployment
        if [[ "$FORCE" == "true" ]]; then
            cleanup_old_deployment "$current_color"
        fi
        
        return 0
    else
        log ERROR "Rollback failed - manual intervention required"
        return 1
    fi
}

# Monitor deployment metrics
monitor_deployment() {
    local color="$1"
    local duration="${2:-60}"
    
    if [[ "$MONITORING_ENABLED" != "true" ]]; then
        log INFO "Monitoring disabled, skipping metrics collection"
        return 0
    fi
    
    log INFO "Monitoring $color deployment for ${duration}s..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log INFO "[DRY RUN] Would monitor $color deployment"
        return 0
    fi
    
    local start_time=$(date +%s)
    local end_time=$((start_time + duration))
    local error_count=0
    local success_count=0
    
    while [[ $(date +%s) -lt $end_time ]]; do
        # Check pod status
        local unhealthy_pods=$(kubectl get pods -n "$NAMESPACE" -l "color=$color" \
            --field-selector=status.phase!=Running -o name 2>/dev/null | wc -l)
        
        if [[ $unhealthy_pods -gt 0 ]]; then
            ((error_count++))
            log WARN "Found $unhealthy_pods unhealthy pods"
        else
            ((success_count++))
        fi
        
        # Simple health check
        if kubectl port-forward "svc/${RELEASE_NAME}-${color}" 8080:8080 -n "$NAMESPACE" &> /dev/null &
        then
            local port_forward_pid=$!
            sleep 2
            
            if curl -f "http://localhost:8080/health" --max-time 5 &> /dev/null; then
                ((success_count++))
            else
                ((error_count++))
            fi
            
            kill $port_forward_pid 2>/dev/null || true
        fi
        
        sleep 10
    done
    
    local total_checks=$((error_count + success_count))
    local success_rate=0
    
    if [[ $total_checks -gt 0 ]]; then
        success_rate=$(( (success_count * 100) / total_checks ))
    fi
    
    log INFO "Monitoring results: $success_count successful, $error_count failed (${success_rate}% success rate)"
    
    # Consider deployment successful if success rate > 95%
    if [[ $success_rate -ge 95 ]]; then
        log SUCCESS "Deployment monitoring passed with ${success_rate}% success rate"
        return 0
    else
        log ERROR "Deployment monitoring failed with ${success_rate}% success rate"
        return 1
    fi
}

# Main blue-green deployment process
execute_blue_green_deployment() {
    log INFO "Starting Blue-Green deployment process..."
    
    # Get current active color
    local current_color=$(get_active_color)
    local new_color=$([[ "$current_color" == "blue" ]] && echo "green" || echo "blue")
    
    log INFO "Current active color: $current_color"
    log INFO "Deploying to: $new_color"
    
    # Step 1: Deploy to new color
    if ! deploy_to_color "$new_color"; then
        log ERROR "Failed to deploy to $new_color environment"
        exit 1
    fi
    
    # Step 2: Health check new deployment
    if ! health_check_color "$new_color"; then
        log ERROR "Health check failed for $new_color environment"
        if [[ "$FORCE" != "true" ]]; then
            cleanup_old_deployment "$new_color"
        fi
        exit 1
    fi
    
    # Step 3: Monitor new deployment
    if ! monitor_deployment "$new_color" 60; then
        log ERROR "Monitoring failed for $new_color environment"
        if [[ "$FORCE" != "true" ]]; then
            cleanup_old_deployment "$new_color"
        fi
        exit 1
    fi
    
    # Step 4: Switch traffic (with delay)
    log INFO "Waiting ${TRAFFIC_SPLIT_DELAY}s before traffic switch..."
    sleep "$TRAFFIC_SPLIT_DELAY"
    
    if ! switch_traffic "$new_color"; then
        log ERROR "Failed to switch traffic to $new_color"
        rollback_deployment "$new_color" "$current_color"
        exit 1
    fi
    
    # Step 5: Post-switch monitoring
    log INFO "Monitoring post-switch performance..."
    if ! monitor_deployment "$new_color" 120; then
        log ERROR "Post-switch monitoring failed"
        if [[ "$FORCE" != "true" ]]; then
            rollback_deployment "$new_color" "$current_color"
            exit 1
        fi
    fi
    
    # Step 6: Cleanup old deployment
    cleanup_old_deployment "$current_color"
    
    log SUCCESS "Blue-Green deployment completed successfully!"
    log INFO "Active color is now: $new_color"
}

# Cleanup function
cleanup() {
    log INFO "Cleaning up Blue-Green deployment process..."
    
    # Kill any background processes
    local pids=$(jobs -p)
    if [[ -n "$pids" ]]; then
        kill $pids 2>/dev/null || true
    fi
    
    log INFO "Blue-Green deployment process cleanup completed"
    log INFO "Deployment log available at: $LOG_FILE"
}

# Main function
main() {
    # Set up trap for cleanup
    trap cleanup EXIT
    
    log INFO "Nephoran Intent Operator - Blue-Green Deployment"
    log INFO "Namespace: $NAMESPACE"
    log INFO "Release: $RELEASE_NAME"
    log INFO "Chart: $HELM_CHART_PATH"
    log INFO "Values: ${VALUES_FILE:-default}"
    log INFO "Dry Run: $DRY_RUN"
    log INFO "Force: $FORCE"
    log INFO "Monitoring: $MONITORING_ENABLED"
    
    # Validate prerequisites
    validate_prerequisites
    
    # Execute blue-green deployment
    execute_blue_green_deployment
    
    log SUCCESS "Blue-Green deployment process completed successfully!"
}

# Parse arguments and run
parse_args "$@"
main