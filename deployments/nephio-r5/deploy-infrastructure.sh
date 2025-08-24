#!/bin/bash
# Nephio R5-O-RAN L Release Infrastructure Deployment Script
# Automated deployment of production-ready multi-cluster infrastructure

set -euo pipefail

# Script metadata
SCRIPT_VERSION="1.0.0"
NEPHIO_VERSION="r5"
ORAN_RELEASE="l-release"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

# Configuration with defaults
CLUSTER_NAME="${CLUSTER_NAME:-nephio-r5-production}"
CLOUD_PROVIDER="${CLOUD_PROVIDER:-kind}"
AWS_REGION="${AWS_REGION:-us-west-2}"
ARGOCD_REPO_URL="${ARGOCD_REPO_URL:-https://github.com/thc1006/nephoran-intent-operator.git}"
ARGOCD_REPO_BRANCH="${ARGOCD_REPO_BRANCH:-feat/conductor-loop}"
DRY_RUN="${DRY_RUN:-false}"
SKIP_PREREQUISITES="${SKIP_PREREQUISITES:-false}"
DEPLOYMENT_TIMEOUT="${DEPLOYMENT_TIMEOUT:-1800}"  # 30 minutes

# Prerequisites check
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing_tools=()
    
    # Required CLI tools
    if ! command -v kubectl &> /dev/null; then missing_tools+=("kubectl"); fi
    if ! command -v helm &> /dev/null; then missing_tools+=("helm"); fi
    if ! command -v kustomize &> /dev/null; then missing_tools+=("kustomize"); fi
    if ! command -v argocd &> /dev/null; then missing_tools+=("argocd"); fi
    if ! command -v jq &> /dev/null; then missing_tools+=("jq"); fi
    
    # Cloud-specific tools
    case $CLOUD_PROVIDER in
        aws)
            if ! command -v aws &> /dev/null; then missing_tools+=("aws"); fi
            ;;
        gcp)
            if ! command -v gcloud &> /dev/null; then missing_tools+=("gcloud"); fi
            ;;
        azure)
            if ! command -v az &> /dev/null; then missing_tools+=("az"); fi
            ;;
        metal3)
            if ! command -v clusterctl &> /dev/null; then missing_tools+=("clusterctl"); fi
            ;;
    esac
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        exit 1
    fi
    
    # Check Kubernetes cluster access
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot access Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    
    # Check Kubernetes version
    local k8s_version
    k8s_version=$(kubectl version --short | grep "Server Version" | awk '{print $3}' | sed 's/v//')
    if ! version_gte "$k8s_version" "1.29.0"; then
        log_error "Kubernetes version $k8s_version is not supported. Minimum version is 1.29.0."
        exit 1
    fi
    
    log_success "Prerequisites check completed"
}

# Version comparison helper
version_gte() {
    printf '%s\n%s\n' "$2" "$1" | sort -V -C
}

# Wait for resource to be ready
wait_for_resource() {
    local resource=$1
    local condition=${2:-Ready}
    local timeout=${3:-600}
    local namespace=${4:-default}
    
    log_info "Waiting for $resource to be $condition in namespace $namespace..."
    
    if kubectl wait --for=condition="$condition" "$resource" -n "$namespace" --timeout="${timeout}s"; then
        log_success "$resource is $condition"
        return 0
    else
        log_error "$resource failed to become $condition within ${timeout}s"
        return 1
    fi
}

# Deploy management cluster
deploy_management_cluster() {
    log_info "Deploying Nephio R5 management cluster..."
    
    if [ "$DRY_RUN" = "true" ]; then
        log_info "[DRY RUN] Would deploy management cluster"
        return 0
    fi
    
    # Create namespace for cluster API
    kubectl create namespace cluster-api-system --dry-run=client -o yaml | kubectl apply -f -
    
    # Deploy management cluster
    envsubst < "$SCRIPT_DIR/ocloud-management-cluster.yaml" | kubectl apply -f -
    
    # Wait for cluster to be ready
    wait_for_resource "cluster/nephio-r5-management" "Ready" 1200
    
    log_success "Management cluster deployed successfully"
}

# Install ArgoCD
install_argocd() {
    log_info "Installing ArgoCD..."
    
    if [ "$DRY_RUN" = "true" ]; then
        log_info "[DRY RUN] Would install ArgoCD"
        return 0
    fi
    
    # Create ArgoCD namespace
    kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
    
    # Install ArgoCD
    kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/v2.10.0/manifests/install.yaml
    
    # Wait for ArgoCD to be ready
    wait_for_resource "deployment/argocd-server" "Available" 600 "argocd"
    wait_for_resource "deployment/argocd-application-controller" "Available" 600 "argocd"
    wait_for_resource "deployment/argocd-repo-server" "Available" 600 "argocd"
    
    # Configure ArgoCD
    log_info "Configuring ArgoCD..."
    
    # Add repository
    local argocd_password
    argocd_password=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d 2>/dev/null || echo "")
    
    if [ -n "$argocd_password" ]; then
        # Port forward ArgoCD server
        kubectl port-forward -n argocd svc/argocd-server 8080:443 &
        local port_forward_pid=$!
        
        # Wait for port forward to be ready
        sleep 5
        
        # Login to ArgoCD
        if argocd login localhost:8080 --username admin --password "$argocd_password" --insecure; then
            # Add repository
            argocd repo add "$ARGOCD_REPO_URL" --type git --name nephoran-intent-operator
            log_success "ArgoCD repository added"
        else
            log_warning "Could not login to ArgoCD. Manual repository configuration required."
        fi
        
        # Stop port forward
        kill $port_forward_pid 2>/dev/null || true
    fi
    
    log_success "ArgoCD installation completed"
}

# Deploy ApplicationSets
deploy_applicationsets() {
    log_info "Deploying ArgoCD ApplicationSets..."
    
    if [ "$DRY_RUN" = "true" ]; then
        log_info "[DRY RUN] Would deploy ApplicationSets"
        return 0
    fi
    
    # Apply ApplicationSets
    envsubst < "$SCRIPT_DIR/argocd-applicationsets.yaml" | kubectl apply -f -
    
    # Wait for ApplicationSets to be created
    sleep 10
    
    # Check ApplicationSets status
    local applicationsets
    applicationsets=$(kubectl get applicationsets -n argocd --no-headers | wc -l)
    log_info "Created $applicationsets ApplicationSets"
    
    log_success "ApplicationSets deployed successfully"
}

# Deploy edge clusters
deploy_edge_clusters() {
    log_info "Deploying edge clusters..."
    
    if [ "$DRY_RUN" = "true" ]; then
        log_info "[DRY RUN] Would deploy edge clusters"
        return 0
    fi
    
    # Deploy edge clusters
    envsubst < "$SCRIPT_DIR/baremetal-edge-clusters.yaml" | kubectl apply -f -
    
    # Wait for clusters to start provisioning
    sleep 30
    
    # Monitor cluster provisioning (don't wait for completion as it can take a long time)
    local clusters
    clusters=$(kubectl get clusters --no-headers 2>/dev/null | wc -l || echo "0")
    log_info "$clusters edge clusters are being provisioned"
    
    log_success "Edge cluster provisioning initiated"
}

# Deploy service mesh
deploy_service_mesh() {
    log_info "Deploying Istio ambient service mesh..."
    
    if [ "$DRY_RUN" = "true" ]; then
        log_info "[DRY RUN] Would deploy service mesh"
        return 0
    fi
    
    # Create istio-system namespace
    kubectl create namespace istio-system --dry-run=client -o yaml | kubectl apply -f -
    
    # Deploy service mesh
    kubectl apply -f "$SCRIPT_DIR/service-mesh-ambient.yaml"
    
    # Wait for Istio control plane
    wait_for_resource "deployment/istiod" "Available" 600 "istio-system"
    
    log_success "Service mesh deployed successfully"
}

# Deploy monitoring stack
deploy_monitoring() {
    log_info "Deploying monitoring and observability stack..."
    
    if [ "$DRY_RUN" = "true" ]; then
        log_info "[DRY RUN] Would deploy monitoring stack"
        return 0
    fi
    
    # Create monitoring namespaces
    for ns in monitoring ves analytics; do
        kubectl create namespace "$ns" --dry-run=client -o yaml | kubectl apply -f -
    done
    
    # Deploy monitoring stack
    kubectl apply -f "$SCRIPT_DIR/monitoring-observability.yaml"
    
    # Wait for key monitoring components
    wait_for_resource "deployment/prometheus" "Available" 600 "monitoring"
    wait_for_resource "deployment/ves-collector" "Available" 600 "ves"
    
    log_success "Monitoring stack deployed successfully"
}

# Deploy security policies
deploy_security() {
    log_info "Deploying security policies and compliance..."
    
    if [ "$DRY_RUN" = "true" ]; then
        log_info "[DRY RUN] Would deploy security policies"
        return 0
    fi
    
    # Create security namespaces
    for ns in security-system spire-system cert-manager; do
        kubectl create namespace "$ns" --dry-run=client -o yaml | kubectl apply -f -
    done
    
    # Deploy security policies
    kubectl apply -f "$SCRIPT_DIR/security-policies.yaml"
    
    log_success "Security policies deployed successfully"
}

# Deploy disaster recovery
deploy_disaster_recovery() {
    log_info "Deploying disaster recovery solution..."
    
    if [ "$DRY_RUN" = "true" ]; then
        log_info "[DRY RUN] Would deploy disaster recovery"
        return 0
    fi
    
    # Create velero namespace
    kubectl create namespace velero --dry-run=client -o yaml | kubectl apply -f -
    
    # Check if cloud credentials exist
    if ! kubectl get secret cloud-credentials -n velero &>/dev/null; then
        log_warning "Cloud credentials not found. Creating placeholder secret."
        kubectl create secret generic cloud-credentials -n velero --from-literal=cloud=placeholder
    fi
    
    # Deploy disaster recovery
    kubectl apply -f "$SCRIPT_DIR/disaster-recovery.yaml"
    
    log_success "Disaster recovery deployed successfully"
}

# Verify deployment
verify_deployment() {
    log_info "Verifying deployment..."
    
    local failed_checks=0
    
    # Check core components
    log_info "Checking core components..."
    
    # ArgoCD
    if kubectl get deployment argocd-server -n argocd &>/dev/null; then
        log_success "✓ ArgoCD server is deployed"
    else
        log_error "✗ ArgoCD server not found"
        ((failed_checks++))
    fi
    
    # ApplicationSets
    local app_sets
    app_sets=$(kubectl get applicationsets -n argocd --no-headers 2>/dev/null | wc -l || echo "0")
    if [ "$app_sets" -gt 0 ]; then
        log_success "✓ $app_sets ApplicationSets deployed"
    else
        log_error "✗ No ApplicationSets found"
        ((failed_checks++))
    fi
    
    # Service mesh
    if kubectl get deployment istiod -n istio-system &>/dev/null; then
        log_success "✓ Istio service mesh is deployed"
    else
        log_error "✗ Istio service mesh not found"
        ((failed_checks++))
    fi
    
    # Monitoring
    if kubectl get deployment prometheus -n monitoring &>/dev/null; then
        log_success "✓ Prometheus monitoring is deployed"
    else
        log_error "✗ Prometheus monitoring not found"
        ((failed_checks++))
    fi
    
    # Security
    local network_policies
    network_policies=$(kubectl get networkpolicies -A --no-headers 2>/dev/null | wc -l || echo "0")
    if [ "$network_policies" -gt 0 ]; then
        log_success "✓ $network_policies Network policies deployed"
    else
        log_warning "⚠ No network policies found"
    fi
    
    # Disaster recovery
    if kubectl get deployment velero -n velero &>/dev/null; then
        log_success "✓ Velero backup solution is deployed"
    else
        log_warning "⚠ Velero backup solution not found"
    fi
    
    # Summary
    if [ $failed_checks -eq 0 ]; then
        log_success "All core components verified successfully!"
    else
        log_error "$failed_checks core components failed verification"
        return 1
    fi
    
    # Display access information
    display_access_info
}

# Display access information
display_access_info() {
    log_info "Deployment completed! Access information:"
    
    echo ""
    echo "=== ArgoCD Dashboard ==="
    echo "URL: https://localhost:8080 (with port-forward)"
    echo "Port-forward command: kubectl port-forward -n argocd svc/argocd-server 8080:443"
    echo "Username: admin"
    
    local argocd_password
    argocd_password=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" 2>/dev/null | base64 -d 2>/dev/null || echo "Not available")
    echo "Password: $argocd_password"
    
    echo ""
    echo "=== Grafana Dashboard ==="
    echo "URL: http://localhost:3000 (with port-forward)"
    echo "Port-forward command: kubectl port-forward -n monitoring svc/grafana 3000:3000"
    echo "Username: admin"
    echo "Password: admin (change on first login)"
    
    echo ""
    echo "=== Prometheus Metrics ==="
    echo "URL: http://localhost:9090 (with port-forward)"
    echo "Port-forward command: kubectl port-forward -n monitoring svc/prometheus 9090:9090"
    
    echo ""
    echo "=== Useful Commands ==="
    echo "Monitor ArgoCD applications: kubectl get applications -n argocd"
    echo "Check cluster status: kubectl get clusters -A"
    echo "View conductor-loop logs: kubectl logs -n nephoran-conductor deployment/conductor-loop"
    echo "Check backup schedules: kubectl get schedules -n velero"
    
    echo ""
    echo "=== Next Steps ==="
    echo "1. Configure DNS for external access"
    echo "2. Set up external load balancers if needed"
    echo "3. Configure monitoring alerts"
    echo "4. Test disaster recovery procedures"
    echo "5. Review security policies and compliance"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up temporary resources..."
    # Kill any background processes
    jobs -p | xargs -r kill 2>/dev/null || true
}

# Main deployment function
main() {
    log_info "Starting Nephio R5-O-RAN L Release infrastructure deployment"
    log_info "Script version: $SCRIPT_VERSION"
    log_info "Cluster name: $CLUSTER_NAME"
    log_info "Cloud provider: $CLOUD_PROVIDER"
    log_info "Dry run: $DRY_RUN"
    
    # Trap cleanup on exit
    trap cleanup EXIT
    
    # Check prerequisites
    if [ "$SKIP_PREREQUISITES" != "true" ]; then
        check_prerequisites
    fi
    
    # Set timeout for entire deployment
    (
        sleep $DEPLOYMENT_TIMEOUT
        log_error "Deployment timeout reached (${DEPLOYMENT_TIMEOUT}s)"
        exit 124
    ) &
    local timeout_pid=$!
    
    # Execute deployment steps
    set +e  # Don't exit on individual step failures
    
    # Step 1: Management cluster (if using cluster API)
    if [ "$CLOUD_PROVIDER" != "existing" ]; then
        if ! deploy_management_cluster; then
            log_error "Management cluster deployment failed"
            kill $timeout_pid 2>/dev/null
            exit 1
        fi
    fi
    
    # Step 2: ArgoCD
    if ! install_argocd; then
        log_error "ArgoCD installation failed"
        kill $timeout_pid 2>/dev/null
        exit 1
    fi
    
    # Step 3: ApplicationSets
    if ! deploy_applicationsets; then
        log_error "ApplicationSets deployment failed"
        kill $timeout_pid 2>/dev/null
        exit 1
    fi
    
    # Step 4: Edge clusters
    if ! deploy_edge_clusters; then
        log_warning "Edge cluster deployment had issues (continuing...)"
    fi
    
    # Step 5: Service mesh
    if ! deploy_service_mesh; then
        log_warning "Service mesh deployment had issues (continuing...)"
    fi
    
    # Step 6: Monitoring
    if ! deploy_monitoring; then
        log_warning "Monitoring deployment had issues (continuing...)"
    fi
    
    # Step 7: Security
    if ! deploy_security; then
        log_warning "Security policies deployment had issues (continuing...)"
    fi
    
    # Step 8: Disaster recovery
    if ! deploy_disaster_recovery; then
        log_warning "Disaster recovery deployment had issues (continuing...)"
    fi
    
    # Kill timeout process
    kill $timeout_pid 2>/dev/null || true
    
    # Step 9: Verification
    if ! verify_deployment; then
        log_error "Deployment verification failed"
        exit 1
    fi
    
    set -e  # Re-enable exit on error
    
    log_success "Nephio R5-O-RAN L Release infrastructure deployment completed successfully!"
}

# Help function
show_help() {
    cat << EOF
Nephio R5-O-RAN L Release Infrastructure Deployment Script

Usage: $0 [OPTIONS]

Options:
    -h, --help                  Show this help message
    -d, --dry-run              Perform a dry run without making changes
    -s, --skip-prerequisites   Skip prerequisites check
    -c, --cluster-name NAME    Set cluster name (default: nephio-r5-production)
    -p, --cloud-provider TYPE  Set cloud provider (kind|aws|gcp|azure|metal3)
    -t, --timeout SECONDS     Set deployment timeout (default: 1800)
    
Environment Variables:
    CLUSTER_NAME               Cluster name
    CLOUD_PROVIDER            Cloud provider type
    AWS_REGION                AWS region for deployment
    ARGOCD_REPO_URL           ArgoCD repository URL
    ARGOCD_REPO_BRANCH        ArgoCD repository branch
    DRY_RUN                   Enable dry run mode
    SKIP_PREREQUISITES        Skip prerequisites check
    DEPLOYMENT_TIMEOUT        Deployment timeout in seconds

Examples:
    # Deploy with default settings
    $0
    
    # Dry run deployment
    $0 --dry-run
    
    # Deploy to AWS with custom cluster name
    CLOUD_PROVIDER=aws CLUSTER_NAME=my-nephio-cluster $0
    
    # Skip prerequisites and deploy
    $0 --skip-prerequisites

For more information, see the README.md file.
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -d|--dry-run)
            DRY_RUN="true"
            shift
            ;;
        -s|--skip-prerequisites)
            SKIP_PREREQUISITES="true"
            shift
            ;;
        -c|--cluster-name)
            CLUSTER_NAME="$2"
            shift 2
            ;;
        -p|--cloud-provider)
            CLOUD_PROVIDER="$2"
            shift 2
            ;;
        -t|--timeout)
            DEPLOYMENT_TIMEOUT="$2"
            shift 2
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Run main function
main "$@"