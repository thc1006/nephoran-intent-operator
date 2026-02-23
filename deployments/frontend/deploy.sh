#!/bin/bash
set -euo pipefail

# Deployment script for Nephoran Intent Operator Frontend
# This script deploys the complete intent system including frontend and backend

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE="nephoran-intent"
DOCKER_IMAGE="nephoran/intent-ingest:latest"
FRONTEND_HTML="${SCRIPT_DIR}/index.html"
K8S_MANIFESTS="${SCRIPT_DIR}/k8s-manifests.yaml"

# Functions
log_info() {
    echo -e "${CYAN}[INFO]${NC} $1"
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

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl not found. Please install kubectl."
        exit 1
    fi

    # Check Docker (for building image)
    if ! command -v docker &> /dev/null; then
        log_warning "Docker not found. Skipping image build."
    fi

    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi

    log_success "Prerequisites check passed"
}

build_docker_image() {
    log_info "Building Docker image for intent-ingest..."

    if ! command -v docker &> /dev/null; then
        log_warning "Docker not available, skipping image build"
        return
    fi

    cd "${PROJECT_ROOT}"

    docker build \
        -f "${SCRIPT_DIR}/Dockerfile.intent-ingest" \
        -t "${DOCKER_IMAGE}" \
        .

    log_success "Docker image built: ${DOCKER_IMAGE}"
}

create_frontend_configmap() {
    log_info "Creating frontend ConfigMap..."

    kubectl create configmap nephoran-frontend-html \
        --from-file=index.html="${FRONTEND_HTML}" \
        --namespace="${NAMESPACE}" \
        --dry-run=client -o yaml | kubectl apply -f -

    log_success "Frontend ConfigMap created"
}

create_schema_configmap() {
    log_info "Creating intent schema ConfigMap..."

    local schema_file="${PROJECT_ROOT}/docs/contracts/intent.schema.json"

    if [ -f "${schema_file}" ]; then
        kubectl create configmap intent-schema \
            --from-file=intent.schema.json="${schema_file}" \
            --namespace="${NAMESPACE}" \
            --dry-run=client -o yaml | kubectl apply -f -
        log_success "Schema ConfigMap created"
    else
        log_warning "Schema file not found at ${schema_file}, skipping"
    fi
}

deploy_manifests() {
    log_info "Deploying Kubernetes manifests..."

    kubectl apply -f "${K8S_MANIFESTS}"

    log_success "Kubernetes manifests deployed"
}

wait_for_rollout() {
    log_info "Waiting for deployments to be ready..."

    kubectl wait --for=condition=available \
        --timeout=300s \
        deployment/nephoran-frontend \
        deployment/intent-ingest \
        -n "${NAMESPACE}"

    log_success "All deployments are ready"
}

setup_port_forward() {
    log_info "Setting up port forwarding..."

    # Kill any existing port-forward on port 8888
    pkill -f "kubectl.*port-forward.*8888" || true

    # Start port-forward in background
    kubectl port-forward \
        -n "${NAMESPACE}" \
        svc/nephoran-frontend \
        8888:80 > /dev/null 2>&1 &

    sleep 2

    log_success "Port forwarding setup on http://localhost:8888"
}

display_info() {
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}   Nephoran Intent Operator Deployed   ${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo -e "${CYAN}Frontend URL:${NC} http://localhost:8888"
    echo -e "${CYAN}Namespace:${NC} ${NAMESPACE}"
    echo ""
    echo -e "${CYAN}To access the frontend:${NC}"
    echo -e "  ${YELLOW}kubectl port-forward -n ${NAMESPACE} svc/nephoran-frontend 8888:80${NC}"
    echo ""
    echo -e "${CYAN}To view logs:${NC}"
    echo -e "  ${YELLOW}kubectl logs -n ${NAMESPACE} -l app=intent-ingest -f${NC}"
    echo ""
    echo -e "${CYAN}To view pods:${NC}"
    echo -e "  ${YELLOW}kubectl get pods -n ${NAMESPACE}${NC}"
    echo ""
    echo -e "${CYAN}To delete the deployment:${NC}"
    echo -e "  ${YELLOW}kubectl delete -f ${K8S_MANIFESTS}${NC}"
    echo ""
}

main() {
    log_info "Starting Nephoran Intent Operator deployment..."

    check_prerequisites

    # Build Docker image (optional)
    if [ "${BUILD_IMAGE:-false}" = "true" ]; then
        build_docker_image
    fi

    # Deploy
    deploy_manifests
    create_frontend_configmap
    create_schema_configmap

    # Wait for ready
    wait_for_rollout

    # Setup access
    if [ "${SETUP_PORT_FORWARD:-true}" = "true" ]; then
        setup_port_forward
    fi

    # Display info
    display_info

    log_success "Deployment completed successfully!"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --build-image)
            BUILD_IMAGE=true
            shift
            ;;
        --no-port-forward)
            SETUP_PORT_FORWARD=false
            shift
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --build-image         Build Docker image before deploying"
            echo "  --no-port-forward     Skip automatic port forwarding setup"
            echo "  --help                Show this help message"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Run main
main
