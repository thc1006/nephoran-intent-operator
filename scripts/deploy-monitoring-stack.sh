#!/bin/bash

# Nephoran Performance Monitoring Stack Deployment Script
# Production-ready deployment automation with multi-environment support
# 
# This script provides:
# - Multi-environment deployment (dev/staging/prod)
# - Automated pre-flight checks
# - Rolling deployments with health verification
# - Backup and rollback capabilities
# - Comprehensive logging and monitoring
# - Security validations
# - Resource optimization

set -euo pipefail

# Script metadata
readonly SCRIPT_VERSION="1.0.0"
readonly SCRIPT_NAME="$(basename "$0")"
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Configuration
readonly MONITORING_NAMESPACE="nephoran-monitoring"
readonly HELM_CHART_PATH="${PROJECT_ROOT}/deployments/helm/nephoran-performance-monitoring"
readonly CONFIG_DIR="${PROJECT_ROOT}/configs/monitoring"
readonly BACKUP_DIR="/tmp/nephoran-monitoring-backup"
readonly LOG_FILE="/tmp/nephoran-monitoring-deploy-$(date +%Y%m%d-%H%M%S).log"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly PURPLE='\033[0;35m'
readonly CYAN='\033[0;36m'
readonly WHITE='\033[1;37m'
readonly NC='\033[0m' # No Color

# Default values
ENVIRONMENT="development"
ACTION="deploy"
DRY_RUN="false"
SKIP_BACKUP="false"
SKIP_HEALTH_CHECK="false"
FORCE="false"
VALUES_FILE=""
TIMEOUT="600s"
DEBUG="false"

# Logging functions
log() {
    local level="$1"
    shift
    local message="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "${timestamp} [${level}] ${message}" | tee -a "${LOG_FILE}"
}

log_info() { log "INFO" "$@"; }
log_warn() { log "WARN" "${YELLOW}$*${NC}"; }
log_error() { log "ERROR" "${RED}$*${NC}"; }
log_success() { log "SUCCESS" "${GREEN}$*${NC}"; }
log_debug() { [[ "${DEBUG}" == "true" ]] && log "DEBUG" "${BLUE}$*${NC}"; }

# Usage information
usage() {
    cat << EOF
${WHITE}Nephoran Performance Monitoring Stack Deployment${NC}

${CYAN}USAGE:${NC}
    ${SCRIPT_NAME} [OPTIONS] ACTION [ENVIRONMENT]

${CYAN}ACTIONS:${NC}
    deploy          Deploy or upgrade the monitoring stack
    uninstall       Remove the monitoring stack
    status          Show deployment status
    validate        Validate configuration
    backup          Backup current configuration
    restore         Restore from backup
    test            Run post-deployment tests

${CYAN}ENVIRONMENTS:${NC}
    development     Development environment (default)
    staging         Staging environment
    production      Production environment

${CYAN}OPTIONS:${NC}
    -h, --help                Show this help message
    -v, --version             Show script version
    -f, --values-file FILE    Custom values file
    -n, --dry-run             Show what would be deployed without applying
    -d, --debug               Enable debug logging
    --skip-backup             Skip backup creation
    --skip-health-check       Skip health checks
    --force                   Force deployment even if checks fail
    --timeout DURATION        Timeout for operations (default: 600s)

${CYAN}EXAMPLES:${NC}
    # Deploy to development with default settings
    ${SCRIPT_NAME} deploy development

    # Deploy to production with custom values
    ${SCRIPT_NAME} deploy production -f values-production.yaml

    # Dry run for staging deployment
    ${SCRIPT_NAME} deploy staging --dry-run

    # Check deployment status
    ${SCRIPT_NAME} status production

    # Backup current configuration
    ${SCRIPT_NAME} backup production

    # Run validation only
    ${SCRIPT_NAME} validate staging

${CYAN}CONFIGURATION:${NC}
    Environment-specific values files should be located at:
    - ${CONFIG_DIR}/values-development.yaml
    - ${CONFIG_DIR}/values-staging.yaml
    - ${CONFIG_DIR}/values-production.yaml

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                exit 0
                ;;
            -v|--version)
                echo "${SCRIPT_NAME} version ${SCRIPT_VERSION}"
                exit 0
                ;;
            -f|--values-file)
                VALUES_FILE="$2"
                shift 2
                ;;
            -n|--dry-run)
                DRY_RUN="true"
                shift
                ;;
            -d|--debug)
                DEBUG="true"
                shift
                ;;
            --skip-backup)
                SKIP_BACKUP="true"
                shift
                ;;
            --skip-health-check)
                SKIP_HEALTH_CHECK="true"
                shift
                ;;
            --force)
                FORCE="true"
                shift
                ;;
            --timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            deploy|uninstall|status|validate|backup|restore|test)
                ACTION="$1"
                shift
                ;;
            development|staging|production)
                ENVIRONMENT="$1"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Validate environment
validate_environment() {
    log_info "Validating environment: ${ENVIRONMENT}"
    
    case "${ENVIRONMENT}" in
        development|staging|production)
            log_success "Environment '${ENVIRONMENT}' is valid"
            ;;
        *)
            log_error "Invalid environment: ${ENVIRONMENT}"
            log_error "Valid environments: development, staging, production"
            exit 1
            ;;
    esac
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing_tools=()
    
    # Check for required tools
    for tool in kubectl helm jq yq; do
        if ! command -v "${tool}" &> /dev/null; then
            missing_tools+=("${tool}")
        fi
    done
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_error "Please install the missing tools and try again"
        exit 1
    fi
    
    # Check Kubernetes connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        log_error "Please check your kubeconfig and cluster connectivity"
        exit 1
    fi
    
    # Check Helm
    if ! helm version &> /dev/null; then
        log_error "Helm is not properly configured"
        exit 1
    fi
    
    # Verify chart exists
    if [[ ! -d "${HELM_CHART_PATH}" ]]; then
        log_error "Helm chart not found at: ${HELM_CHART_PATH}"
        exit 1
    fi
    
    # Check namespace access
    if kubectl get namespace "${MONITORING_NAMESPACE}" &> /dev/null; then
        log_debug "Namespace ${MONITORING_NAMESPACE} exists"
    else
        log_info "Namespace ${MONITORING_NAMESPACE} will be created"
    fi
    
    log_success "Prerequisites check passed"
}

# Get values file for environment
get_values_file() {
    if [[ -n "${VALUES_FILE}" ]]; then
        if [[ ! -f "${VALUES_FILE}" ]]; then
            log_error "Custom values file not found: ${VALUES_FILE}"
            exit 1
        fi
        echo "${VALUES_FILE}"
        return
    fi
    
    local env_values_file="${CONFIG_DIR}/values-${ENVIRONMENT}.yaml"
    if [[ -f "${env_values_file}" ]]; then
        echo "${env_values_file}"
    else
        log_warn "Environment-specific values file not found: ${env_values_file}"
        log_info "Using default values from chart"
        echo ""
    fi
}

# Create namespace if it doesn't exist
ensure_namespace() {
    log_info "Ensuring namespace ${MONITORING_NAMESPACE} exists..."
    
    if ! kubectl get namespace "${MONITORING_NAMESPACE}" &> /dev/null; then
        log_info "Creating namespace: ${MONITORING_NAMESPACE}"
        
        if [[ "${DRY_RUN}" == "true" ]]; then
            log_info "[DRY RUN] Would create namespace: ${MONITORING_NAMESPACE}"
            return
        fi
        
        kubectl create namespace "${MONITORING_NAMESPACE}"
        
        # Add labels for monitoring and environment
        kubectl label namespace "${MONITORING_NAMESPACE}" \
            nephoran.com/monitoring=enabled \
            nephoran.com/environment="${ENVIRONMENT}" \
            app.kubernetes.io/managed-by=nephoran-monitoring-deploy
        
        log_success "Namespace ${MONITORING_NAMESPACE} created"
    else
        log_debug "Namespace ${MONITORING_NAMESPACE} already exists"
    fi
}

# Backup current configuration
backup_current_config() {
    if [[ "${SKIP_BACKUP}" == "true" ]]; then
        log_info "Skipping backup as requested"
        return
    fi
    
    log_info "Creating backup of current configuration..."
    
    mkdir -p "${BACKUP_DIR}"
    local backup_timestamp=$(date +%Y%m%d-%H%M%S)
    local backup_path="${BACKUP_DIR}/monitoring-backup-${backup_timestamp}"
    
    mkdir -p "${backup_path}"
    
    # Backup Helm release values
    if helm get values nephoran-performance-monitoring -n "${MONITORING_NAMESPACE}" &> /dev/null; then
        helm get values nephoran-performance-monitoring -n "${MONITORING_NAMESPACE}" > "${backup_path}/helm-values.yaml"
        helm get manifest nephoran-performance-monitoring -n "${MONITORING_NAMESPACE}" > "${backup_path}/manifests.yaml"
        log_info "Backed up Helm release to: ${backup_path}"
    fi
    
    # Backup ConfigMaps and Secrets
    kubectl get configmaps -n "${MONITORING_NAMESPACE}" -o yaml > "${backup_path}/configmaps.yaml" 2>/dev/null || true
    kubectl get secrets -n "${MONITORING_NAMESPACE}" -o yaml > "${backup_path}/secrets.yaml" 2>/dev/null || true
    
    # Backup PVCs (metadata only)
    kubectl get pvc -n "${MONITORING_NAMESPACE}" -o yaml > "${backup_path}/pvcs.yaml" 2>/dev/null || true
    
    echo "${backup_path}" > "${BACKUP_DIR}/latest-backup-path"
    
    log_success "Backup created at: ${backup_path}"
}

# Validate Helm chart and values
validate_configuration() {
    log_info "Validating Helm chart and configuration..."
    
    local values_file
    values_file=$(get_values_file)
    
    local helm_args=(
        "template"
        "nephoran-performance-monitoring"
        "${HELM_CHART_PATH}"
        "--namespace" "${MONITORING_NAMESPACE}"
    )
    
    if [[ -n "${values_file}" ]]; then
        helm_args+=("--values" "${values_file}")
    fi
    
    # Validate Helm template
    if ! helm "${helm_args[@]}" > /tmp/helm-template-output.yaml; then
        log_error "Helm template validation failed"
        exit 1
    fi
    
    # Validate against Kubernetes API
    if ! kubectl apply --dry-run=client -f /tmp/helm-template-output.yaml; then
        log_error "Kubernetes validation failed"
        exit 1
    fi
    
    # Additional validations
    validate_resource_requirements
    validate_security_configuration
    validate_network_policies
    
    log_success "Configuration validation passed"
}

# Validate resource requirements
validate_resource_requirements() {
    log_debug "Validating resource requirements..."
    
    # Check node resources
    local node_info
    node_info=$(kubectl get nodes -o json)
    
    local total_cpu_millicores=0
    local total_memory_bytes=0
    
    # Calculate total cluster resources (simplified)
    while read -r cpu memory; do
        cpu_millicores=$(echo "${cpu}" | sed 's/m$//' | bc)
        memory_bytes=$(echo "${memory}" | sed 's/Ki$//' | bc)
        memory_bytes=$((memory_bytes * 1024))
        
        total_cpu_millicores=$((total_cpu_millicores + cpu_millicores))
        total_memory_bytes=$((total_memory_bytes + memory_bytes))
    done < <(echo "${node_info}" | jq -r '.items[] | [.status.allocatable.cpu, .status.allocatable.memory] | @tsv')
    
    log_debug "Cluster resources - CPU: ${total_cpu_millicores}m, Memory: $((total_memory_bytes / 1024 / 1024))Mi"
    
    # Minimum requirements for monitoring stack
    local min_cpu_millicores=4000  # 4 CPU cores
    local min_memory_mb=8192       # 8GB RAM
    
    if [[ $total_cpu_millicores -lt $min_cpu_millicores ]]; then
        log_warn "Cluster may have insufficient CPU resources"
        log_warn "Required: ${min_cpu_millicores}m, Available: ${total_cpu_millicores}m"
    fi
    
    if [[ $((total_memory_bytes / 1024 / 1024)) -lt $min_memory_mb ]]; then
        log_warn "Cluster may have insufficient memory resources"
        log_warn "Required: ${min_memory_mb}Mi, Available: $((total_memory_bytes / 1024 / 1024))Mi"
    fi
}

# Validate security configuration
validate_security_configuration() {
    log_debug "Validating security configuration..."
    
    # Check for security contexts in values
    local values_file
    values_file=$(get_values_file)
    
    if [[ -n "${values_file}" ]]; then
        if ! yq eval '.global.security.enabled' "${values_file}" | grep -q "true"; then
            log_warn "Security contexts are not enabled in values file"
        fi
        
        if ! yq eval '.networkPolicies.enabled' "${values_file}" | grep -q "true"; then
            log_warn "Network policies are not enabled in values file"
        fi
    fi
    
    # Check if Pod Security Standards are enforced
    if kubectl get namespace "${MONITORING_NAMESPACE}" -o yaml | grep -q "pod-security.kubernetes.io/enforce"; then
        log_debug "Pod Security Standards are enforced for namespace"
    else
        log_warn "Pod Security Standards are not enforced for namespace ${MONITORING_NAMESPACE}"
    fi
}

# Validate network policies
validate_network_policies() {
    log_debug "Validating network policies..."
    
    # Check if CNI supports network policies
    if kubectl get nodes -o json | jq -r '.items[0].status.nodeInfo.containerRuntimeVersion' | grep -q "containerd"; then
        log_debug "Container runtime supports network policies"
    else
        log_warn "Container runtime may not support network policies"
    fi
}

# Deploy monitoring stack
deploy_monitoring_stack() {
    log_info "Deploying Nephoran Performance Monitoring Stack to ${ENVIRONMENT}..."
    
    ensure_namespace
    
    if [[ "${DRY_RUN}" != "true" ]]; then
        backup_current_config
    fi
    
    validate_configuration
    
    local values_file
    values_file=$(get_values_file)
    
    local helm_args=(
        "upgrade"
        "--install"
        "nephoran-performance-monitoring"
        "${HELM_CHART_PATH}"
        "--namespace" "${MONITORING_NAMESPACE}"
        "--create-namespace"
        "--timeout" "${TIMEOUT}"
        "--atomic"
        "--cleanup-on-fail"
    )
    
    if [[ -n "${values_file}" ]]; then
        helm_args+=("--values" "${values_file}")
    fi
    
    # Environment-specific overrides
    case "${ENVIRONMENT}" in
        production)
            helm_args+=(
                "--set" "global.environment=production"
                "--set" "extraConfig.highAvailability.enabled=true"
                "--set" "extraConfig.security.enableTLS=true"
            )
            ;;
        staging)
            helm_args+=(
                "--set" "global.environment=staging"
                "--set" "extraConfig.highAvailability.enabled=true"
            )
            ;;
        development)
            helm_args+=(
                "--set" "global.environment=development"
                "--set" "prometheus.server.resources.requests.memory=1Gi"
                "--set" "grafana.resources.requests.memory=256Mi"
            )
            ;;
    esac
    
    if [[ "${DRY_RUN}" == "true" ]]; then
        helm_args+=("--dry-run")
        log_info "[DRY RUN] Would execute: helm ${helm_args[*]}"
    fi
    
    log_info "Executing Helm deployment..."
    log_debug "Helm command: helm ${helm_args[*]}"
    
    if helm "${helm_args[@]}"; then
        log_success "Helm deployment completed successfully"
    else
        log_error "Helm deployment failed"
        exit 1
    fi
    
    if [[ "${DRY_RUN}" != "true" && "${SKIP_HEALTH_CHECK}" != "true" ]]; then
        perform_health_checks
    fi
    
    show_deployment_info
}

# Perform health checks
perform_health_checks() {
    log_info "Performing post-deployment health checks..."
    
    # Wait for pods to be ready
    log_info "Waiting for pods to be ready..."
    if ! kubectl wait --for=condition=ready pod -l "app.kubernetes.io/instance=nephoran-performance-monitoring" -n "${MONITORING_NAMESPACE}" --timeout=300s; then
        log_error "Pods failed to become ready within timeout"
        if [[ "${FORCE}" != "true" ]]; then
            exit 1
        fi
    fi
    
    # Check Prometheus health
    check_prometheus_health
    
    # Check Grafana health
    check_grafana_health
    
    # Check AlertManager health
    check_alertmanager_health
    
    # Validate metrics collection
    validate_metrics_collection
    
    log_success "Health checks completed successfully"
}

# Check Prometheus health
check_prometheus_health() {
    log_info "Checking Prometheus health..."
    
    local prometheus_pod
    prometheus_pod=$(kubectl get pods -n "${MONITORING_NAMESPACE}" -l "app=prometheus,component=server" -o jsonpath="{.items[0].metadata.name}")
    
    if [[ -z "${prometheus_pod}" ]]; then
        log_error "Prometheus pod not found"
        return 1
    fi
    
    # Port forward to check health endpoint
    kubectl port-forward -n "${MONITORING_NAMESPACE}" "${prometheus_pod}" 9090:9090 &
    local port_forward_pid=$!
    
    # Wait a moment for port forward to establish
    sleep 3
    
    # Check health endpoint
    if curl -f -s http://localhost:9090/-/healthy > /dev/null; then
        log_success "Prometheus health check passed"
    else
        log_error "Prometheus health check failed"
        kill "${port_forward_pid}" 2>/dev/null || true
        return 1
    fi
    
    # Check ready endpoint
    if curl -f -s http://localhost:9090/-/ready > /dev/null; then
        log_success "Prometheus ready check passed"
    else
        log_error "Prometheus ready check failed"
        kill "${port_forward_pid}" 2>/dev/null || true
        return 1
    fi
    
    kill "${port_forward_pid}" 2>/dev/null || true
}

# Check Grafana health
check_grafana_health() {
    log_info "Checking Grafana health..."
    
    local grafana_pod
    grafana_pod=$(kubectl get pods -n "${MONITORING_NAMESPACE}" -l "app.kubernetes.io/name=grafana" -o jsonpath="{.items[0].metadata.name}")
    
    if [[ -z "${grafana_pod}" ]]; then
        log_error "Grafana pod not found"
        return 1
    fi
    
    # Check if Grafana is responding
    if kubectl exec -n "${MONITORING_NAMESPACE}" "${grafana_pod}" -- curl -f -s http://localhost:3000/api/health > /dev/null; then
        log_success "Grafana health check passed"
    else
        log_error "Grafana health check failed"
        return 1
    fi
}

# Check AlertManager health
check_alertmanager_health() {
    log_info "Checking AlertManager health..."
    
    local alertmanager_pod
    alertmanager_pod=$(kubectl get pods -n "${MONITORING_NAMESPACE}" -l "app=alertmanager" -o jsonpath="{.items[0].metadata.name}" 2>/dev/null)
    
    if [[ -z "${alertmanager_pod}" ]]; then
        log_warn "AlertManager pod not found (may be disabled)"
        return 0
    fi
    
    # Check AlertManager health
    kubectl port-forward -n "${MONITORING_NAMESPACE}" "${alertmanager_pod}" 9093:9093 &
    local port_forward_pid=$!
    
    sleep 3
    
    if curl -f -s http://localhost:9093/-/healthy > /dev/null; then
        log_success "AlertManager health check passed"
    else
        log_error "AlertManager health check failed"
        kill "${port_forward_pid}" 2>/dev/null || true
        return 1
    fi
    
    kill "${port_forward_pid}" 2>/dev/null || true
}

# Validate metrics collection
validate_metrics_collection() {
    log_info "Validating metrics collection..."
    
    # Check if ServiceMonitors are created
    local servicemonitor_count
    servicemonitor_count=$(kubectl get servicemonitors -n "${MONITORING_NAMESPACE}" --no-headers 2>/dev/null | wc -l)
    
    if [[ $servicemonitor_count -gt 0 ]]; then
        log_success "Found ${servicemonitor_count} ServiceMonitors"
    else
        log_warn "No ServiceMonitors found"
    fi
    
    # Check PrometheusRules
    local prometheusrule_count
    prometheusrule_count=$(kubectl get prometheusrules -n "${MONITORING_NAMESPACE}" --no-headers 2>/dev/null | wc -l)
    
    if [[ $prometheusrule_count -gt 0 ]]; then
        log_success "Found ${prometheusrule_count} PrometheusRules"
    else
        log_warn "No PrometheusRules found"
    fi
}

# Show deployment information
show_deployment_info() {
    log_info "Deployment Information:"
    
    echo ""
    echo -e "${CYAN}=== Nephoran Performance Monitoring Stack ===${NC}"
    echo -e "${WHITE}Environment:${NC} ${ENVIRONMENT}"
    echo -e "${WHITE}Namespace:${NC} ${MONITORING_NAMESPACE}"
    echo -e "${WHITE}Deployment Time:${NC} $(date)"
    echo ""
    
    # Show Helm release info
    echo -e "${CYAN}=== Helm Release Information ===${NC}"
    helm list -n "${MONITORING_NAMESPACE}" | grep nephoran-performance-monitoring || true
    echo ""
    
    # Show pod status
    echo -e "${CYAN}=== Pod Status ===${NC}"
    kubectl get pods -n "${MONITORING_NAMESPACE}" -o wide
    echo ""
    
    # Show services
    echo -e "${CYAN}=== Services ===${NC}"
    kubectl get services -n "${MONITORING_NAMESPACE}"
    echo ""
    
    # Show ingresses if any
    if kubectl get ingresses -n "${MONITORING_NAMESPACE}" --no-headers 2>/dev/null | grep -q .; then
        echo -e "${CYAN}=== Ingresses ===${NC}"
        kubectl get ingresses -n "${MONITORING_NAMESPACE}"
        echo ""
    fi
    
    # Show access information
    show_access_information
}

# Show access information
show_access_information() {
    echo -e "${CYAN}=== Access Information ===${NC}"
    
    # Grafana access
    local grafana_service_type
    grafana_service_type=$(kubectl get service -n "${MONITORING_NAMESPACE}" nephoran-performance-monitoring-grafana -o jsonpath="{.spec.type}" 2>/dev/null || echo "ClusterIP")
    
    echo -e "${WHITE}Grafana:${NC}"
    if [[ "${grafana_service_type}" == "LoadBalancer" ]]; then
        local grafana_lb_ip
        grafana_lb_ip=$(kubectl get service -n "${MONITORING_NAMESPACE}" nephoran-performance-monitoring-grafana -o jsonpath="{.status.loadBalancer.ingress[0].ip}" 2>/dev/null || echo "Pending")
        echo "  External URL: http://${grafana_lb_ip}:3000"
    elif [[ "${grafana_service_type}" == "NodePort" ]]; then
        local grafana_node_port
        grafana_node_port=$(kubectl get service -n "${MONITORING_NAMESPACE}" nephoran-performance-monitoring-grafana -o jsonpath="{.spec.ports[0].nodePort}")
        echo "  NodePort URL: http://<node-ip>:${grafana_node_port}"
    else
        echo "  Port-forward: kubectl port-forward -n ${MONITORING_NAMESPACE} svc/nephoran-performance-monitoring-grafana 3000:3000"
        echo "  Local URL: http://localhost:3000"
    fi
    
    # Prometheus access
    echo -e "${WHITE}Prometheus:${NC}"
    echo "  Port-forward: kubectl port-forward -n ${MONITORING_NAMESPACE} svc/nephoran-performance-monitoring-prometheus-server 9090:9090"
    echo "  Local URL: http://localhost:9090"
    
    # AlertManager access (if enabled)
    if kubectl get service -n "${MONITORING_NAMESPACE}" nephoran-performance-monitoring-alertmanager &>/dev/null; then
        echo -e "${WHITE}AlertManager:${NC}"
        echo "  Port-forward: kubectl port-forward -n ${MONITORING_NAMESPACE} svc/nephoran-performance-monitoring-alertmanager 9093:9093"
        echo "  Local URL: http://localhost:9093"
    fi
    
    echo ""
    echo -e "${GREEN}✅ Monitoring stack deployed successfully!${NC}"
    echo -e "${YELLOW}📊 Access the executive performance dashboard at: /d/nephoran-executive-perf${NC}"
    echo -e "${YELLOW}🔍 Access the detailed component dashboard at: /d/nephoran-detailed-perf${NC}"
    echo ""
}

# Uninstall monitoring stack
uninstall_monitoring_stack() {
    log_info "Uninstalling Nephoran Performance Monitoring Stack from ${ENVIRONMENT}..."
    
    if [[ "${DRY_RUN}" != "true" ]]; then
        backup_current_config
    fi
    
    if [[ "${DRY_RUN}" == "true" ]]; then
        log_info "[DRY RUN] Would uninstall Helm release: nephoran-performance-monitoring"
        log_info "[DRY RUN] Would preserve namespace: ${MONITORING_NAMESPACE}"
        return
    fi
    
    # Uninstall Helm release
    if helm list -n "${MONITORING_NAMESPACE}" | grep -q nephoran-performance-monitoring; then
        log_info "Uninstalling Helm release..."
        helm uninstall nephoran-performance-monitoring -n "${MONITORING_NAMESPACE}"
        log_success "Helm release uninstalled"
    else
        log_info "Helm release not found"
    fi
    
    # Clean up remaining resources
    log_info "Cleaning up remaining resources..."
    
    # Remove CRDs if they exist
    kubectl delete crd \
        servicemonitors.monitoring.coreos.com \
        prometheusrules.monitoring.coreos.com \
        2>/dev/null || true
    
    # Optionally remove namespace (prompt user)
    if [[ "${FORCE}" == "true" ]]; then
        kubectl delete namespace "${MONITORING_NAMESPACE}" --ignore-not-found
        log_success "Namespace ${MONITORING_NAMESPACE} deleted"
    else
        read -p "Do you want to delete the namespace '${MONITORING_NAMESPACE}'? (y/N): " -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            kubectl delete namespace "${MONITORING_NAMESPACE}" --ignore-not-found
            log_success "Namespace ${MONITORING_NAMESPACE} deleted"
        else
            log_info "Namespace ${MONITORING_NAMESPACE} preserved"
        fi
    fi
    
    log_success "Uninstallation completed"
}

# Show deployment status
show_deployment_status() {
    log_info "Checking deployment status for ${ENVIRONMENT}..."
    
    # Check if namespace exists
    if ! kubectl get namespace "${MONITORING_NAMESPACE}" &>/dev/null; then
        log_info "Namespace ${MONITORING_NAMESPACE} does not exist"
        return
    fi
    
    # Check Helm release
    if helm list -n "${MONITORING_NAMESPACE}" | grep -q nephoran-performance-monitoring; then
        echo -e "${GREEN}✅ Helm release found${NC}"
        helm status nephoran-performance-monitoring -n "${MONITORING_NAMESPACE}"
    else
        echo -e "${RED}❌ Helm release not found${NC}"
    fi
    
    echo ""
    
    # Check pods
    local pod_count
    pod_count=$(kubectl get pods -n "${MONITORING_NAMESPACE}" --no-headers 2>/dev/null | wc -l)
    
    if [[ $pod_count -gt 0 ]]; then
        echo -e "${GREEN}✅ Found ${pod_count} pods${NC}"
        kubectl get pods -n "${MONITORING_NAMESPACE}"
    else
        echo -e "${RED}❌ No pods found${NC}"
    fi
    
    echo ""
    
    # Check services
    local service_count
    service_count=$(kubectl get services -n "${MONITORING_NAMESPACE}" --no-headers 2>/dev/null | wc -l)
    
    if [[ $service_count -gt 0 ]]; then
        echo -e "${GREEN}✅ Found ${service_count} services${NC}"
        kubectl get services -n "${MONITORING_NAMESPACE}"
    else
        echo -e "${RED}❌ No services found${NC}"
    fi
}

# Run post-deployment tests
run_deployment_tests() {
    log_info "Running post-deployment tests for ${ENVIRONMENT}..."
    
    local test_results=()
    local failed_tests=0
    
    # Test 1: Pod readiness
    log_info "Test 1: Checking pod readiness..."
    if kubectl get pods -n "${MONITORING_NAMESPACE}" -o jsonpath='{.items[*].status.conditions[?(@.type=="Ready")].status}' | grep -q "False"; then
        test_results+=("❌ Pod readiness check failed")
        ((failed_tests++))
    else
        test_results+=("✅ All pods are ready")
    fi
    
    # Test 2: Service endpoints
    log_info "Test 2: Checking service endpoints..."
    local services_with_endpoints=0
    local total_services
    total_services=$(kubectl get services -n "${MONITORING_NAMESPACE}" --no-headers | wc -l)
    
    while read -r service; do
        if kubectl get endpoints -n "${MONITORING_NAMESPACE}" "${service}" -o jsonpath='{.subsets[*].addresses[*].ip}' | grep -q .; then
            ((services_with_endpoints++))
        fi
    done < <(kubectl get services -n "${MONITORING_NAMESPACE}" -o jsonpath='{.items[*].metadata.name}')
    
    if [[ $services_with_endpoints -eq $total_services ]]; then
        test_results+=("✅ All services have endpoints")
    else
        test_results+=("❌ Some services lack endpoints (${services_with_endpoints}/${total_services})")
        ((failed_tests++))
    fi
    
    # Test 3: Metrics collection
    log_info "Test 3: Testing metrics collection..."
    if kubectl get servicemonitors -n "${MONITORING_NAMESPACE}" --no-headers 2>/dev/null | grep -q .; then
        test_results+=("✅ ServiceMonitors are configured")
    else
        test_results+=("❌ No ServiceMonitors found")
        ((failed_tests++))
    fi
    
    # Test 4: Alerting rules
    log_info "Test 4: Testing alerting rules..."
    if kubectl get prometheusrules -n "${MONITORING_NAMESPACE}" --no-headers 2>/dev/null | grep -q .; then
        test_results+=("✅ PrometheusRules are configured")
    else
        test_results+=("❌ No PrometheusRules found")
        ((failed_tests++))
    fi
    
    # Display results
    echo ""
    echo -e "${CYAN}=== Test Results ===${NC}"
    for result in "${test_results[@]}"; do
        echo "  ${result}"
    done
    
    echo ""
    if [[ $failed_tests -eq 0 ]]; then
        log_success "All tests passed! ✅"
    else
        log_error "${failed_tests} test(s) failed ❌"
        exit 1
    fi
}

# Main function
main() {
    echo -e "${PURPLE}"
    cat << 'EOF'
    _   __           __                        
   / | / /__  ____  / /_  ____  _________ _____
  /  |/ / _ \/ __ \/ __ \/ __ \/ ___/ __ `/ __ \
 / /|  /  __/ /_/ / / / / /_/ / /  / /_/ / / / /
/_/ |_/\___/ .___/_/ /_/\____/_/   \__,_/_/ /_/ 
          /_/                                  
Performance Monitoring Stack Deployment
EOF
    echo -e "${NC}"
    
    log_info "Starting ${SCRIPT_NAME} v${SCRIPT_VERSION}"
    log_info "Log file: ${LOG_FILE}"
    
    parse_args "$@"
    validate_environment
    check_prerequisites
    
    case "${ACTION}" in
        deploy)
            deploy_monitoring_stack
            ;;
        uninstall)
            uninstall_monitoring_stack
            ;;
        status)
            show_deployment_status
            ;;
        validate)
            validate_configuration
            log_success "Configuration validation completed"
            ;;
        backup)
            backup_current_config
            ;;
        restore)
            log_error "Restore functionality not implemented yet"
            exit 1
            ;;
        test)
            run_deployment_tests
            ;;
        *)
            log_error "Unknown action: ${ACTION}"
            usage
            exit 1
            ;;
    esac
    
    log_success "Operation completed successfully"
}

# Execute main function with all arguments
main "$@"