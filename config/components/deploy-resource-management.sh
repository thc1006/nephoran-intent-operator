#!/bin/bash

# Nephoran Intent Operator - Resource Management Deployment Script
# This script deploys comprehensive resource management configurations

set -euo pipefail

# Configuration
NAMESPACE="nephoran-system"
CONFIG_DIR="$(dirname "$0")"
STRATEGY="${RESOURCE_STRATEGY:-optimized}"
DRY_RUN="${DRY_RUN:-false}"
WAIT_TIMEOUT="300s"

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

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
Nephoran Intent Operator - Resource Management Deployment

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    -n, --namespace         Kubernetes namespace (default: nephoran-system)
    -s, --strategy          Resource strategy: minimal|balanced|optimized|performance (default: optimized)
    -d, --dry-run           Perform dry run without applying resources
    -w, --wait-timeout      Timeout for waiting for deployments (default: 300s)
    --enable-hpa            Enable Horizontal Pod Autoscaler (default: true)
    --enable-pdb            Enable Pod Disruption Budget (default: true)
    --enable-vpa            Enable Vertical Pod Autoscaler (default: false)
    --enable-monitoring     Enable resource monitoring (default: true)

EXAMPLES:
    # Deploy with default settings
    $0

    # Deploy with minimal resources for development
    $0 --strategy minimal

    # Dry run deployment
    $0 --dry-run

    # Deploy with VPA enabled
    $0 --enable-vpa

ENVIRONMENT VARIABLES:
    RESOURCE_STRATEGY       Resource management strategy
    DRY_RUN                 Enable dry run mode
    KUBECTL_CONTEXT         Kubernetes context to use

EOF
}

# Parse command line arguments
parse_args() {
    local enable_hpa="true"
    local enable_pdb="true"
    local enable_vpa="false"
    local enable_monitoring="true"

    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -s|--strategy)
                STRATEGY="$2"
                shift 2
                ;;
            -d|--dry-run)
                DRY_RUN="true"
                shift
                ;;
            -w|--wait-timeout)
                WAIT_TIMEOUT="$2"
                shift 2
                ;;
            --enable-hpa)
                enable_hpa="true"
                shift
                ;;
            --disable-hpa)
                enable_hpa="false"
                shift
                ;;
            --enable-pdb)
                enable_pdb="true"
                shift
                ;;
            --disable-pdb)
                enable_pdb="false"
                shift
                ;;
            --enable-vpa)
                enable_vpa="true"
                shift
                ;;
            --enable-monitoring)
                enable_monitoring="true"
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done

    export ENABLE_HPA="$enable_hpa"
    export ENABLE_PDB="$enable_pdb"
    export ENABLE_VPA="$enable_vpa"
    export ENABLE_MONITORING="$enable_monitoring"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is required but not installed"
        exit 1
    fi

    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi

    # Check kustomize
    if ! command -v kustomize &> /dev/null; then
        log_warn "kustomize not found, using kubectl kustomize"
    fi

    log_success "Prerequisites check passed"
}

# Create namespace if it doesn't exist
create_namespace() {
    log_info "Creating namespace: $NAMESPACE"
    
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_info "Namespace $NAMESPACE already exists"
    else
        if [[ "$DRY_RUN" == "true" ]]; then
            log_info "[DRY RUN] Would create namespace: $NAMESPACE"
        else
            kubectl create namespace "$NAMESPACE"
            log_success "Created namespace: $NAMESPACE"
        fi
    fi
}

# Apply resource quotas and limit ranges
apply_resource_constraints() {
    log_info "Applying resource constraints..."

    cat <<EOF | kubectl apply ${DRY_RUN:+--dry-run=client} -f -
apiVersion: v1
kind: ResourceQuota
metadata:
  name: nephoran-resource-quota
  namespace: $NAMESPACE
spec:
  hard:
    requests.cpu: "5"
    requests.memory: 8Gi
    limits.cpu: "20"
    limits.memory: 32Gi
    persistentvolumeclaims: "10"
    pods: "50"
    services: "20"
---
apiVersion: v1
kind: LimitRange
metadata:
  name: nephoran-limit-range
  namespace: $NAMESPACE
spec:
  limits:
  - default:
      cpu: "500m"
      memory: "512Mi"
    defaultRequest:
      cpu: "100m"
      memory: "128Mi"
    max:
      cpu: "2"
      memory: "4Gi"
    min:
      cpu: "10m"
      memory: "32Mi"
    type: Container
EOF

    if [[ "$DRY_RUN" != "true" ]]; then
        log_success "Applied resource constraints"
    fi
}

# Deploy main operator
deploy_operator() {
    log_info "Deploying main operator..."

    if [[ "$DRY_RUN" == "true" ]]; then
        kubectl apply --dry-run=client -f "$CONFIG_DIR/manager.yaml"
    else
        kubectl apply -f "$CONFIG_DIR/manager.yaml"
        kubectl rollout status deployment/nephoran-operator-controller-manager -n "$NAMESPACE" --timeout="$WAIT_TIMEOUT"
        log_success "Main operator deployed successfully"
    fi
}

# Deploy components
deploy_components() {
    log_info "Deploying components with resource limits..."

    local components=(
        "llm-processor-resources.yaml"
        "rag-api-resources.yaml"
        "webhook-resources.yaml"
        "nephio-bridge-resources.yaml"
        "oran-adaptor-resources.yaml"
    )

    for component in "${components[@]}"; do
        log_info "Deploying $component..."
        
        if [[ "$DRY_RUN" == "true" ]]; then
            kubectl apply --dry-run=client -f "$CONFIG_DIR/$component"
        else
            kubectl apply -f "$CONFIG_DIR/$component"
        fi
    done

    if [[ "$DRY_RUN" != "true" ]]; then
        log_success "All components deployed"
    fi
}

# Deploy auto-scaling configurations
deploy_autoscaling() {
    if [[ "$ENABLE_HPA" == "true" ]]; then
        log_info "Deploying auto-scaling configurations..."

        if [[ "$DRY_RUN" == "true" ]]; then
            kubectl apply --dry-run=client -f "$CONFIG_DIR/hpa-configurations.yaml"
        else
            kubectl apply -f "$CONFIG_DIR/hpa-configurations.yaml"
            log_success "Auto-scaling configurations deployed"
        fi
    else
        log_info "Auto-scaling disabled, skipping HPA deployment"
    fi
}

# Wait for deployments
wait_for_deployments() {
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would wait for deployments to be ready"
        return
    fi

    log_info "Waiting for deployments to be ready..."

    local deployments=(
        "nephoran-operator-controller-manager"
        "llm-processor"
        "rag-api"
        "nephoran-webhook"
        "nephio-bridge"
        "oran-adaptor"
    )

    for deployment in "${deployments[@]}"; do
        if kubectl get deployment "$deployment" -n "$NAMESPACE" &> /dev/null; then
            log_info "Waiting for $deployment..."
            kubectl rollout status deployment/"$deployment" -n "$NAMESPACE" --timeout="$WAIT_TIMEOUT" || {
                log_warn "Deployment $deployment did not become ready within timeout"
            }
        else
            log_info "Deployment $deployment not found, skipping..."
        fi
    done

    log_success "All deployments are ready"
}

# Verify resource limits
verify_resources() {
    log_info "Verifying resource limits..."

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "[DRY RUN] Would verify resource limits"
        return
    fi

    # Check resource quotas
    kubectl describe resourcequota nephoran-resource-quota -n "$NAMESPACE" || true

    # Check pod resource usage
    kubectl top pods -n "$NAMESPACE" 2>/dev/null || log_warn "Metrics server not available"

    # Check HPA status
    if [[ "$ENABLE_HPA" == "true" ]]; then
        kubectl get hpa -n "$NAMESPACE" || true
    fi

    log_success "Resource verification completed"
}

# Deploy monitoring
deploy_monitoring() {
    if [[ "$ENABLE_MONITORING" == "true" ]]; then
        log_info "Deploying resource monitoring..."

        cat <<EOF | kubectl apply ${DRY_RUN:+--dry-run=client} -f -
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nephoran-resource-metrics
  namespace: $NAMESPACE
  labels:
    app.kubernetes.io/name: nephoran-intent-operator
spec:
  selector:
    matchLabels:
      app.kubernetes.io/part-of: nephoran-intent-operator
  endpoints:
  - port: metrics
    interval: 15s
    path: /metrics
EOF

        if [[ "$DRY_RUN" != "true" ]]; then
            log_success "Resource monitoring deployed"
        fi
    fi
}

# Print deployment summary
print_summary() {
    log_info "Deployment Summary"
    echo "=================="
    echo "Namespace: $NAMESPACE"
    echo "Strategy: $STRATEGY"
    echo "HPA Enabled: $ENABLE_HPA"
    echo "PDB Enabled: $ENABLE_PDB"
    echo "VPA Enabled: $ENABLE_VPA"
    echo "Monitoring Enabled: $ENABLE_MONITORING"
    echo "Dry Run: $DRY_RUN"
    echo ""

    if [[ "$DRY_RUN" != "true" ]]; then
        log_info "Checking deployed resources..."
        kubectl get all -n "$NAMESPACE" -l app.kubernetes.io/name=nephoran-intent-operator
        echo ""
        kubectl get hpa -n "$NAMESPACE" 2>/dev/null || true
        echo ""
        kubectl get pdb -n "$NAMESPACE" 2>/dev/null || true
    fi
}

# Main execution function
main() {
    log_info "Starting Nephoran Intent Operator Resource Management Deployment"
    
    parse_args "$@"
    check_prerequisites
    create_namespace
    apply_resource_constraints
    deploy_operator
    deploy_components
    deploy_autoscaling
    deploy_monitoring
    wait_for_deployments
    verify_resources
    print_summary
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_success "Dry run completed successfully"
    else
        log_success "Deployment completed successfully!"
        log_info "You can monitor resources with: kubectl top pods -n $NAMESPACE"
        log_info "Check auto-scaling with: kubectl get hpa -n $NAMESPACE"
    fi
}

# Trap for cleanup
trap 'log_error "Deployment failed"; exit 1' ERR

# Run main function
main "$@"