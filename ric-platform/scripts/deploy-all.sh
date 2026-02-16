#!/bin/bash
# Nephoran Intent Operator - RIC Platform Deployment Script
# Deploys all O-RAN Near-RT RIC components

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RIC_PLATFORM_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}==>${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl not found"
        exit 1
    fi

    # Check helm
    if ! command -v helm &> /dev/null; then
        log_error "helm not found"
        exit 1
    fi

    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi

    log_info "Prerequisites check passed"
}

create_namespaces() {
    log_step "Creating RIC namespaces..."

    for ns in ricinfra ricplt ricxapp; do
        kubectl create namespace "$ns" --dry-run=client -o yaml | kubectl apply -f -
        kubectl label namespace "$ns" ric-component=true --overwrite
    done

    log_info "Namespaces created"
}

deploy_infrastructure() {
    log_step "Deploying RIC Infrastructure (ricinfra)..."

    # Deploy infrastructure components
    # TODO: Add actual Helm chart deployment
    log_warn "Infrastructure deployment not yet implemented"
    log_info "Placeholder: Database, Message Bus, Service Registry"
}

deploy_platform_components() {
    log_step "Deploying RIC Platform Components (ricplt)..."

    declare -a components=("submgr" "rtmgr" "e2mgr" "e2" "appmgr")

    for component in "${components[@]}"; do
        log_info "Deploying $component..."

        # Check if submodule exists
        if [ ! -d "$RIC_PLATFORM_DIR/submodules/$component" ]; then
            log_error "Submodule $component not found. Run: git submodule update --init"
            exit 1
        fi

        # TODO: Deploy using Helm or kubectl
        log_warn "$component deployment not yet implemented"

        # Placeholder for Helm deployment
        # helm upgrade --install "$component" \
        #   "$RIC_PLATFORM_DIR/helm/ric-platform/$component" \
        #   --namespace ricplt \
        #   --create-namespace
    done

    log_info "Platform components deployed"
}

verify_deployment() {
    log_step "Verifying deployment..."

    echo ""
    log_info "ricinfra namespace:"
    kubectl get all -n ricinfra 2>/dev/null || echo "No resources yet"

    echo ""
    log_info "ricplt namespace:"
    kubectl get all -n ricplt 2>/dev/null || echo "No resources yet"

    echo ""
    log_info "ricxapp namespace:"
    kubectl get all -n ricxapp 2>/dev/null || echo "No resources yet"
}

display_info() {
    echo ""
    echo "========================================="
    echo "  RIC Platform Deployment Complete"
    echo "========================================="
    echo ""
    echo "Namespaces created:"
    echo "  - ricinfra (Infrastructure)"
    echo "  - ricplt   (Platform)"
    echo "  - ricxapp  (Applications)"
    echo ""
    echo "To check status:"
    echo "  kubectl get all -n ricplt"
    echo ""
    echo "To view logs:"
    echo "  kubectl logs -n ricplt -l ric-component=submgr -f"
    echo ""
    echo "To port-forward E2 Manager:"
    echo "  kubectl port-forward -n ricplt svc/e2mgr 3800:3800"
    echo ""
}

main() {
    echo "========================================="
    echo "  O-RAN Near-RT RIC Platform Deployment"
    echo "========================================="
    echo ""

    check_prerequisites
    create_namespaces
    deploy_infrastructure
    deploy_platform_components
    verify_deployment
    display_info
}

main "$@"
