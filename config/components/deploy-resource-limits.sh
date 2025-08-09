#!/bin/bash

# Deploy Resource Limits for Nephoran Intent Operator
# This script deploys all resource management configurations with proper limits to prevent memory issues

set -euo pipefail

# Configuration
NAMESPACE="nephoran-system"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="${SCRIPT_DIR}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi
    
    if ! command -v kustomize &> /dev/null; then
        log_error "kustomize is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we can connect to Kubernetes cluster
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Check available resources
check_cluster_resources() {
    log_info "Checking cluster resources..."
    
    # Get node resources
    local total_cpu=$(kubectl get nodes -o jsonpath='{.items[*].status.allocatable.cpu}' | tr ' ' '\n' | sed 's/m$//' | awk '{sum += $1} END {print sum}')
    local total_memory=$(kubectl get nodes -o jsonpath='{.items[*].status.allocatable.memory}' | tr ' ' '\n' | sed 's/Ki$//' | awk '{sum += $1} END {print sum/1024/1024 "Gi"}')
    
    log_info "Available cluster resources:"
    log_info "  Total CPU: ${total_cpu}m"
    log_info "  Total Memory: ${total_memory}"
    
    # Check if we have enough resources for our requirements
    local required_cpu="2000m"  # 2 CPU cores minimum
    local required_memory="8Gi"  # 8GB memory minimum
    
    if [[ ${total_cpu} -lt 2000 ]]; then
        log_warning "Cluster has less than 2 CPU cores available. Consider scaling your cluster."
    fi
    
    # Check for metrics server
    if kubectl get deployment metrics-server -n kube-system &> /dev/null; then
        log_success "Metrics server is available for HPA"
    else
        log_warning "Metrics server not found. HPA may not work properly."
        log_info "To install metrics server: kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml"
    fi
    
    # Check for KEDA if mentioned in configurations
    if kubectl get deployment keda-operator -n keda-system &> /dev/null 2>&1; then
        log_success "KEDA is available for advanced scaling"
    else
        log_warning "KEDA not found. Advanced scaling features may not work."
        log_info "To install KEDA: kubectl apply -f https://github.com/kedacore/keda/releases/download/v2.12.0/keda-2.12.0.yaml"
    fi
}

# Create namespace if it doesn't exist
create_namespace() {
    log_info "Creating namespace ${NAMESPACE}..."
    
    if kubectl get namespace "${NAMESPACE}" &> /dev/null; then
        log_info "Namespace ${NAMESPACE} already exists"
    else
        kubectl create namespace "${NAMESPACE}"
        log_success "Created namespace ${NAMESPACE}"
    fi
    
    # Label namespace for monitoring
    kubectl label namespace "${NAMESPACE}" \
        app.kubernetes.io/name=nephoran-intent-operator \
        app.kubernetes.io/managed-by=kubectl \
        --overwrite
    
    log_success "Namespace ${NAMESPACE} is ready"
}

# Deploy resource quotas and limits
deploy_resource_management() {
    log_info "Deploying resource management configurations..."
    
    # Apply resource management
    if [[ -f "${CONFIG_DIR}/resource-management.yaml" ]]; then
        log_info "Applying resource quotas and limit ranges..."
        kubectl apply -f "${CONFIG_DIR}/resource-management.yaml"
        log_success "Resource management applied"
    else
        log_warning "Resource management configuration not found"
    fi
    
    # Wait for resource quota to be active
    log_info "Waiting for resource quota to be active..."
    kubectl wait --for=condition=ready --timeout=60s -n "${NAMESPACE}" \
        resourcequota/nephoran-system-quota || true
}

# Deploy main controller with resource limits
deploy_controller() {
    log_info "Deploying controller with resource limits..."
    
    # Apply controller deployment
    if [[ -f "${CONFIG_DIR}/../manager/manager.yaml" ]]; then
        log_info "Applying controller deployment..."
        kubectl apply -f "${CONFIG_DIR}/../manager/manager.yaml"
        
        # Wait for controller to be ready
        log_info "Waiting for controller deployment to be ready..."
        kubectl rollout status deployment/nephoran-operator-controller-manager -n "${NAMESPACE}" --timeout=300s
        log_success "Controller deployment is ready"
    else
        log_error "Controller deployment configuration not found"
        exit 1
    fi
}

# Deploy component resource configurations
deploy_components() {
    log_info "Deploying component resource configurations..."
    
    local components=("llm-processor-resources.yaml" "webhook-resources.yaml" "rag-api-resources.yaml" "nephio-bridge-resources.yaml" "oran-adaptor-resources.yaml")
    
    for component in "${components[@]}"; do
        if [[ -f "${CONFIG_DIR}/${component}" ]]; then
            log_info "Deploying ${component}..."
            kubectl apply -f "${CONFIG_DIR}/${component}"
            
            # Extract deployment name and wait for it to be ready
            local deployment_name=$(grep -E "name:.*" "${CONFIG_DIR}/${component}" | grep -v "namespace" | head -1 | awk '{print $2}')
            if [[ -n "${deployment_name}" ]]; then
                log_info "Waiting for ${deployment_name} to be ready..."
                kubectl rollout status deployment/"${deployment_name}" -n "${NAMESPACE}" --timeout=300s || log_warning "Timeout waiting for ${deployment_name}"
            fi
            
            log_success "${component} deployed"
        else
            log_warning "${component} not found, skipping..."
        fi
    done
}

# Deploy HPA configurations
deploy_hpa() {
    log_info "Deploying HorizontalPodAutoscaler configurations..."
    
    if [[ -f "${CONFIG_DIR}/hpa-configurations.yaml" ]]; then
        log_info "Applying HPA configurations..."
        kubectl apply -f "${CONFIG_DIR}/hpa-configurations.yaml"
        log_success "HPA configurations applied"
    else
        log_warning "HPA configurations not found"
    fi
    
    if [[ -f "${CONFIG_DIR}/controller-hpa.yaml" ]]; then
        log_info "Applying controller HPA configuration..."
        kubectl apply -f "${CONFIG_DIR}/controller-hpa.yaml"
        log_success "Controller HPA configuration applied"
    else
        log_warning "Controller HPA configuration not found"
    fi
    
    # Wait a moment and check HPA status
    sleep 10
    log_info "Checking HPA status..."
    kubectl get hpa -n "${NAMESPACE}" || log_warning "No HPA found or metrics not available yet"
}

# Verify resource limits are applied
verify_resource_limits() {
    log_info "Verifying resource limits are properly applied..."
    
    # Check if pods have resource limits
    local pods=$(kubectl get pods -n "${NAMESPACE}" -o name | head -5)
    
    for pod in ${pods}; do
        local pod_name=$(echo "${pod}" | cut -d'/' -f2)
        log_info "Checking resource limits for ${pod_name}..."
        
        local cpu_limit=$(kubectl get "${pod}" -n "${NAMESPACE}" -o jsonpath='{.spec.containers[0].resources.limits.cpu}' 2>/dev/null || echo "not-set")
        local memory_limit=$(kubectl get "${pod}" -n "${NAMESPACE}" -o jsonpath='{.spec.containers[0].resources.limits.memory}' 2>/dev/null || echo "not-set")
        
        if [[ "${cpu_limit}" != "not-set" && "${memory_limit}" != "not-set" ]]; then
            log_success "  ${pod_name}: CPU=${cpu_limit}, Memory=${memory_limit}"
        else
            log_warning "  ${pod_name}: Resource limits not properly set"
        fi
    done
}

# Check resource utilization
check_resource_utilization() {
    log_info "Checking current resource utilization..."
    
    # Wait for metrics to be available
    sleep 30
    
    # Check pod metrics if metrics server is available
    if kubectl top pods -n "${NAMESPACE}" &> /dev/null; then
        log_info "Current pod resource usage:"
        kubectl top pods -n "${NAMESPACE}" | while IFS= read -r line; do
            log_info "  ${line}"
        done
    else
        log_warning "Pod metrics not available (metrics server might not be ready yet)"
    fi
    
    # Check node resources
    if kubectl top nodes &> /dev/null; then
        log_info "Current node resource usage:"
        kubectl top nodes | while IFS= read -r line; do
            log_info "  ${line}"
        done
    else
        log_warning "Node metrics not available"
    fi
}

# Generate monitoring and alerting recommendations
generate_monitoring_recommendations() {
    log_info "Generating monitoring and alerting recommendations..."
    
    cat << EOF

========================================
MONITORING AND ALERTING RECOMMENDATIONS
========================================

1. Memory Usage Alerts:
   - Set alerts for memory usage > 80% on LLM processor pods
   - Set alerts for memory usage > 75% on controller pods
   - Set alerts for memory usage > 70% on webhook pods

2. CPU Usage Alerts:
   - Set alerts for CPU usage > 80% sustained for 5 minutes
   - Set alerts for CPU throttling incidents

3. HPA Alerts:
   - Monitor HPA scaling events
   - Alert on HPA unable to scale due to resource constraints

4. OOM Kill Alerts:
   - Set critical alerts for any OOMKilled pods
   - Monitor container restart count increases

5. Resource Quota Alerts:
   - Alert when namespace resource usage > 90% of quota
   - Monitor resource request/limit ratios

6. Suggested Prometheus AlertManager rules:
   
   groups:
   - name: nephoran-resource-alerts
     rules:
     - alert: NephoranHighMemoryUsage
       expr: container_memory_usage_bytes{namespace="nephoran-system"} / container_spec_memory_limit_bytes > 0.8
       for: 5m
       labels:
         severity: warning
       annotations:
         summary: "High memory usage in Nephoran component"
         
     - alert: NephoranOOMKilled
       expr: kube_pod_container_status_restarts_total{namespace="nephoran-system"} > 0
       for: 0m
       labels:
         severity: critical
       annotations:
         summary: "Nephoran pod was OOM killed"

7. Grafana Dashboard Metrics to Monitor:
   - container_memory_usage_bytes
   - container_cpu_usage_seconds_total
   - kube_pod_container_resource_limits
   - kube_pod_container_resource_requests
   - kube_horizontalpodautoscaler_status_current_replicas

EOF
}

# Main deployment function
deploy_all() {
    log_info "Starting Nephoran Intent Operator resource limits deployment..."
    
    check_prerequisites
    check_cluster_resources
    create_namespace
    deploy_resource_management
    deploy_controller
    deploy_components
    deploy_hpa
    verify_resource_limits
    check_resource_utilization
    generate_monitoring_recommendations
    
    log_success "Deployment completed successfully!"
    log_info "All components have been deployed with proper resource limits."
    log_info "Monitor the cluster for any memory or CPU issues and adjust limits as needed."
    
    # Show current status
    log_info "Current deployment status:"
    kubectl get pods,hpa,pdb -n "${NAMESPACE}"
}

# Cleanup function
cleanup_all() {
    log_info "Cleaning up Nephoran Intent Operator resources..."
    
    # Delete all resources in the namespace
    if kubectl get namespace "${NAMESPACE}" &> /dev/null; then
        kubectl delete namespace "${NAMESPACE}" --ignore-not-found=true
        log_success "Namespace ${NAMESPACE} deleted"
    fi
    
    log_success "Cleanup completed"
}

# Show help
show_help() {
    cat << EOF
Nephoran Intent Operator Resource Limits Deployment Script

Usage: $0 [COMMAND]

Commands:
    deploy      Deploy all resource configurations (default)
    cleanup     Remove all deployed resources
    verify      Verify current resource limits
    status      Show current deployment status
    help        Show this help message

Examples:
    $0 deploy           # Deploy with resource limits
    $0 verify           # Check current resource settings
    $0 cleanup          # Remove all resources
    $0 status           # Show deployment status

Environment Variables:
    NAMESPACE           # Kubernetes namespace (default: nephoran-system)
    
EOF
}

# Main script logic
case "${1:-deploy}" in
    deploy)
        deploy_all
        ;;
    cleanup)
        cleanup_all
        ;;
    verify)
        verify_resource_limits
        ;;
    status)
        kubectl get pods,deployments,hpa,pdb,resourcequota,limitrange -n "${NAMESPACE}"
        ;;
    help)
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac