#!/bin/bash
#
# Optimized GitOps-friendly deployment script for Nephoran Intent Operator
# 
# IMPROVEMENTS:
# - Idempotent operations (safe to run multiple times)
# - No in-place kustomize edits (preserves GitOps state)
# - Declarative configuration via environment variables
# - Better error handling and validation
# - Support for dry-run mode
# - Clean separation of build and deploy phases
#

set -euo pipefail

# --- Configuration ---
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly DEFAULT_REGISTRY="${NEPHORAN_REGISTRY:-us-central1-docker.pkg.dev/poised-elf-466913-q2/nephoran}"
readonly DEFAULT_TAG="${NEPHORAN_TAG:-$(git rev-parse --short HEAD 2>/dev/null || echo 'latest')}"
readonly DEFAULT_NAMESPACE="${NEPHORAN_NAMESPACE:-nephoran-system}"

# Color codes for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

# --- Functions ---
log_info() {
    echo -e "${GREEN}[INFO]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

print_usage() {
    cat << EOF
Usage: $0 [OPTIONS] COMMAND

Optimized GitOps-friendly deployment for Nephoran Intent Operator

COMMANDS:
    build       Build container images
    deploy      Deploy to Kubernetes cluster
    all         Build and deploy (default)
    
OPTIONS:
    -e, --env ENV           Environment: local, dev, staging, production (default: local)
    -r, --registry REG      Container registry (default: $DEFAULT_REGISTRY)
    -t, --tag TAG           Image tag (default: git commit or 'latest')
    -n, --namespace NS      Kubernetes namespace (default: $DEFAULT_NAMESPACE)
    -d, --dry-run          Show what would be done without executing
    -s, --skip-build       Skip image building phase
    -p, --push             Push images to registry (auto-enabled for non-local)
    -h, --help             Show this help message

ENVIRONMENT VARIABLES:
    NEPHORAN_REGISTRY      Default container registry
    NEPHORAN_TAG          Default image tag
    NEPHORAN_NAMESPACE    Default Kubernetes namespace
    KUBECONFIG           Kubernetes config file

EXAMPLES:
    # Local development
    $0 --env local build
    
    # Deploy to staging
    $0 --env staging --tag v1.2.3 deploy
    
    # Full production deployment
    $0 --env production --registry gcr.io/myproject/nephoran all
    
    # Dry run to see what would happen
    $0 --env staging --dry-run deploy
EOF
}

validate_prerequisites() {
    local required_tools=("docker" "kubectl" "kustomize")
    local missing_tools=()
    
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        fi
    done
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_error "Please install missing tools and try again"
        exit 1
    fi
    
    # Validate Kubernetes connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        log_error "Please check your KUBECONFIG and cluster connectivity"
        exit 1
    fi
    
    log_info "All prerequisites satisfied"
}

detect_cluster_type() {
    local context
    context=$(kubectl config current-context)
    
    if [[ "$context" == "minikube" ]]; then
        echo "minikube"
    elif [[ "$context" == "kind-"* ]]; then
        echo "kind"
    elif [[ "$context" == "docker-desktop" ]]; then
        echo "docker-desktop"
    elif [[ "$context" == "gke_"* ]]; then
        echo "gke"
    else
        echo "unknown"
    fi
}

build_images() {
    local services=("llm-processor" "nephio-bridge" "oran-adaptor" "rag-api")
    
    log_info "Building images with tag: $IMAGE_TAG"
    
    for service in "${services[@]}"; do
        local dockerfile_path
        if [[ "$service" == "rag-api" ]]; then
            dockerfile_path="pkg/rag/Dockerfile"
        else
            dockerfile_path="cmd/$service/Dockerfile"
        fi
        
        local image_name="${service}:${IMAGE_TAG}"
        
        log_info "Building $image_name"
        
        if [[ "$DRY_RUN" == "true" ]]; then
            echo "Would run: docker build -t $image_name -f $dockerfile_path ."
        else
            docker build \
                --build-arg VERSION="$IMAGE_TAG" \
                --build-arg BUILD_DATE="$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
                --build-arg VCS_REF="$(git rev-parse HEAD 2>/dev/null || echo 'unknown')" \
                -t "$image_name" \
                -f "$dockerfile_path" \
                . || {
                    log_error "Failed to build $service"
                    exit 1
                }
        fi
    done
    
    log_info "All images built successfully"
}

push_images() {
    local services=("llm-processor" "nephio-bridge" "oran-adaptor" "rag-api")
    
    log_info "Pushing images to registry: $REGISTRY"
    
    for service in "${services[@]}"; do
        local local_image="${service}:${IMAGE_TAG}"
        local remote_image="${REGISTRY}/${service}:${IMAGE_TAG}"
        
        log_info "Pushing $remote_image"
        
        if [[ "$DRY_RUN" == "true" ]]; then
            echo "Would run: docker tag $local_image $remote_image"
            echo "Would run: docker push $remote_image"
        else
            docker tag "$local_image" "$remote_image" || {
                log_error "Failed to tag $service"
                exit 1
            }
            
            docker push "$remote_image" || {
                log_error "Failed to push $service"
                exit 1
            }
        fi
    done
    
    log_info "All images pushed successfully"
}

load_images_local() {
    local cluster_type
    cluster_type=$(detect_cluster_type)
    local services=("llm-processor" "nephio-bridge" "oran-adaptor" "rag-api")
    
    log_info "Loading images into $cluster_type cluster"
    
    for service in "${services[@]}"; do
        local image_name="${service}:${IMAGE_TAG}"
        
        if [[ "$DRY_RUN" == "true" ]]; then
            case "$cluster_type" in
                minikube)
                    echo "Would run: minikube image load $image_name"
                    ;;
                kind)
                    echo "Would run: kind load docker-image $image_name"
                    ;;
                docker-desktop)
                    echo "Images already available in Docker Desktop"
                    ;;
                *)
                    log_warn "Unknown cluster type, assuming shared Docker daemon"
                    ;;
            esac
        else
            case "$cluster_type" in
                minikube)
                    minikube image load "$image_name" || {
                        log_error "Failed to load $image_name into minikube"
                        exit 1
                    }
                    ;;
                kind)
                    kind load docker-image "$image_name" || {
                        log_error "Failed to load $image_name into kind"
                        exit 1
                    }
                    ;;
                docker-desktop)
                    log_info "Image $image_name already available in Docker Desktop"
                    ;;
                *)
                    log_warn "Unknown cluster type, assuming shared Docker daemon"
                    ;;
            esac
        fi
    done
    
    log_info "All images loaded successfully"
}

create_temp_kustomization() {
    local env=$1
    local temp_dir
    temp_dir=$(mktemp -d)
    
    # Copy the overlay to temp directory
    cp -r "${SCRIPT_DIR}/deployments/kustomize/overlays/${env}" "${temp_dir}/"
    
    # Create a new kustomization.yaml with proper image references
    cat > "${temp_dir}/${env}/kustomization.yaml" << EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ../../base

namespace: ${NAMESPACE}

images:
EOF

    local services=("llm-processor" "nephio-bridge" "oran-adaptor" "rag-api")
    
    for service in "${services[@]}"; do
        if [[ "$env" == "local" ]]; then
            cat >> "${temp_dir}/${env}/kustomization.yaml" << EOF
- name: ${service}
  newName: ${service}
  newTag: ${IMAGE_TAG}
EOF
        else
            cat >> "${temp_dir}/${env}/kustomization.yaml" << EOF
- name: ${service}
  newName: ${REGISTRY}/${service}
  newTag: ${IMAGE_TAG}
EOF
        fi
    done
    
    # Add environment-specific patches
    if [[ "$env" == "local" ]]; then
        cat >> "${temp_dir}/${env}/kustomization.yaml" << 'EOF'

patches:
- patch: |-
    - op: replace
      path: /spec/template/spec/containers/0/imagePullPolicy
      value: Never
  target:
    kind: Deployment
EOF
    fi
    
    # Copy base resources to temp directory
    cp -r "${SCRIPT_DIR}/deployments/kustomize/base" "${temp_dir}/"
    
    echo "${temp_dir}"
}

deploy_kubernetes() {
    local env=$1
    
    log_info "Deploying to $env environment in namespace: $NAMESPACE"
    
    # Create namespace if it doesn't exist
    if [[ "$DRY_RUN" == "true" ]]; then
        echo "Would run: kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -"
    else
        kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f - || true
    fi
    
    # Create temporary kustomization
    local temp_dir
    temp_dir=$(create_temp_kustomization "$env")
    
    # Apply kustomization
    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "Dry run - showing generated manifests:"
        kustomize build "${temp_dir}/${env}"
    else
        log_info "Applying kustomization..."
        kustomize build "${temp_dir}/${env}" | kubectl apply -f - || {
            log_error "Failed to apply Kubernetes manifests"
            rm -rf "$temp_dir"
            exit 1
        }
    fi
    
    # Cleanup temp directory
    rm -rf "$temp_dir"
    
    if [[ "$DRY_RUN" != "true" ]]; then
        log_info "Waiting for deployments to be ready..."
        kubectl -n "$NAMESPACE" wait --for=condition=available --timeout=300s deployment --all || {
            log_warn "Some deployments did not become ready in time"
        }
    fi
    
    log_info "Deployment completed successfully"
}

verify_deployment() {
    log_info "Verifying deployment status..."
    
    kubectl -n "$NAMESPACE" get deployments,pods,services
    
    log_info "Checking pod health..."
    kubectl -n "$NAMESPACE" get pods -o wide
    
    log_info "Recent events:"
    kubectl -n "$NAMESPACE" get events --sort-by='.lastTimestamp' | tail -10
}

# --- Main Script ---
# Parse command line arguments
ENVIRONMENT="local"
REGISTRY="$DEFAULT_REGISTRY"
IMAGE_TAG="$DEFAULT_TAG"
NAMESPACE="$DEFAULT_NAMESPACE"
DRY_RUN="false"
SKIP_BUILD="false"
PUSH_IMAGES="false"
COMMAND="all"

while [[ $# -gt 0 ]]; do
    case $1 in
        -e|--env)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -r|--registry)
            REGISTRY="$2"
            shift 2
            ;;
        -t|--tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -d|--dry-run)
            DRY_RUN="true"
            shift
            ;;
        -s|--skip-build)
            SKIP_BUILD="true"
            shift
            ;;
        -p|--push)
            PUSH_IMAGES="true"
            shift
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        build|deploy|all)
            COMMAND="$1"
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac
done

# Auto-enable push for non-local environments
if [[ "$ENVIRONMENT" != "local" ]]; then
    PUSH_IMAGES="true"
fi

# Validate environment
if [[ ! "$ENVIRONMENT" =~ ^(local|dev|staging|production)$ ]]; then
    log_error "Invalid environment: $ENVIRONMENT"
    log_error "Valid environments: local, dev, staging, production"
    exit 1
fi

# Show configuration
log_info "Configuration:"
log_info "  Environment: $ENVIRONMENT"
log_info "  Registry: $REGISTRY"
log_info "  Image Tag: $IMAGE_TAG"
log_info "  Namespace: $NAMESPACE"
log_info "  Dry Run: $DRY_RUN"
log_info "  Skip Build: $SKIP_BUILD"
log_info "  Push Images: $PUSH_IMAGES"
log_info "  Command: $COMMAND"

# Validate prerequisites
if [[ "$DRY_RUN" != "true" ]]; then
    validate_prerequisites
fi

# Execute command
case "$COMMAND" in
    build)
        if [[ "$SKIP_BUILD" != "true" ]]; then
            build_images
            if [[ "$PUSH_IMAGES" == "true" ]]; then
                push_images
            elif [[ "$ENVIRONMENT" == "local" ]]; then
                load_images_local
            fi
        fi
        ;;
    deploy)
        deploy_kubernetes "$ENVIRONMENT"
        if [[ "$DRY_RUN" != "true" ]]; then
            verify_deployment
        fi
        ;;
    all)
        if [[ "$SKIP_BUILD" != "true" ]]; then
            build_images
            if [[ "$PUSH_IMAGES" == "true" ]]; then
                push_images
            elif [[ "$ENVIRONMENT" == "local" ]]; then
                load_images_local
            fi
        fi
        deploy_kubernetes "$ENVIRONMENT"
        if [[ "$DRY_RUN" != "true" ]]; then
            verify_deployment
        fi
        ;;
esac

log_info "Operation completed successfully"