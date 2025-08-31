#!/bin/bash

# =============================================================================
# Optimized Docker Build Script for Nephoran Intent Operator (2025 Standards)
# =============================================================================
# Production-ready Docker build script with GitHub Actions optimization,
# security scanning, multi-architecture support, and advanced caching
# Usage: ./docker-build.sh <service> [options]

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_DATE=$(date -Iseconds)
VCS_REF=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
VERSION=${VERSION:-v2.0.0}
REGISTRY=${REGISTRY:-ghcr.io}
IMAGE_PREFIX=${IMAGE_PREFIX:-nephoran-intent-operator}
PLATFORMS=${PLATFORMS:-linux/amd64,linux/arm64}
BUILD_TYPE=${BUILD_TYPE:-production}

# 2025 optimizations
export DOCKER_BUILDKIT=1
export BUILDX_NO_DEFAULT_ATTESTATIONS=0
export BUILDX_ATTESTATION_MODE=max
export DOCKER_CLI_EXPERIMENTAL=enabled

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

# Help function
show_help() {
    cat << EOF
Production Docker Build Script for Nephoran Intent Operator

Usage: $0 <service> [options]

Services:
  conductor-loop   Build Conductor Loop service
  intent-ingest    Build Intent Ingest service  
  llm-processor    Build LLM Processor service
  nephio-bridge    Build Nephio Bridge service
  oran-adaptor     Build ORAN Adaptor service
  porch-publisher  Build Porch Publisher service
  planner          Build Planner service
  rag-api          Build RAG API service (Python)
  all              Build all services

Options:
  --push           Push images to registry after build
  --scan           Run security scan using Trivy
  --multi-arch     Build for multiple architectures (${PLATFORMS})
  --no-cache       Build without using Docker cache
  --registry       Set custom registry (default: ${REGISTRY})
  --version        Set version tag (default: ${VERSION})
  --platforms      Set target platforms (default: ${PLATFORMS})
  --cache-from     Cache from scope (for GitHub Actions)
  --cache-to       Cache to scope (for GitHub Actions)
  --build-arg      Additional build arguments
  --provenance     Enable provenance attestation (default: mode=max)
  --sbom           Enable SBOM generation (default: true)
  --verbose        Enable verbose output
  --dry-run        Show commands without executing
  --help           Show this help message

Examples:
  $0 llm-processor --push --scan
  $0 all --multi-arch --push
  $0 nephio-bridge --registry my-registry.com --version v1.2.3

Environment Variables:
  REGISTRY         Container registry URL
  VERSION          Image version tag
  PLATFORMS        Target platforms for multi-arch build
  DOCKER_BUILDKIT  Enable BuildKit (recommended: 1)
  TRIVY_SEVERITY   Trivy scan severity (default: HIGH,CRITICAL)
EOF
}

# Parse command line arguments
SERVICE=""
PUSH=false
SCAN=false
MULTI_ARCH=false
NO_CACHE=false

# Parse command line arguments with 2025 enhancements
CACHE_FROM_SCOPES=""
CACHE_TO_SCOPES=""
BUILD_ARGS=""
VERBOSE=false
DRY_RUN=false
PROVENANCE="mode=max"
SBOM="true"

while [[ $# -gt 0 ]]; do
    case $1 in
        conductor-loop|intent-ingest|llm-processor|nephio-bridge|oran-adaptor|porch-publisher|planner|rag-api|all)
            SERVICE="$1"
            shift
            ;;
        --push)
            PUSH=true
            shift
            ;;
        --scan)
            SCAN=true
            shift
            ;;
        --multi-arch)
            MULTI_ARCH=true
            shift
            ;;
        --no-cache)
            NO_CACHE=true
            shift
            ;;
        --registry)
            REGISTRY="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --platforms)
            PLATFORMS="$2"
            shift 2
            ;;
        --cache-from)
            CACHE_FROM_SCOPES="$CACHE_FROM_SCOPES $2"
            shift 2
            ;;
        --cache-to)
            CACHE_TO_SCOPES="$CACHE_TO_SCOPES $2"
            shift 2
            ;;
        --build-arg)
            BUILD_ARGS="$BUILD_ARGS --build-arg $2"
            shift 2
            ;;
        --provenance)
            PROVENANCE="$2"
            shift 2
            ;;
        --sbom)
            SBOM="$2"
            shift 2
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

if [[ -z "$SERVICE" ]]; then
    log_error "Service name is required"
    show_help
    exit 1
fi

# Validate prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    # Check Docker BuildKit
    if [[ "${DOCKER_BUILDKIT:-0}" != "1" ]]; then
        log_warning "DOCKER_BUILDKIT is not enabled. Enable for better performance."
        export DOCKER_BUILDKIT=1
    fi
    
    # Check if we're in a Git repository
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        log_warning "Not in a Git repository. VCS_REF will be 'unknown'"
    fi
    
    # Check Trivy if scanning is requested
    if [[ "$SCAN" == true ]] && ! command -v trivy &> /dev/null; then
        log_error "Trivy is required for security scanning but not found"
        log_info "Install Trivy: https://aquasecurity.github.io/trivy/latest/getting-started/installation/"
        exit 1
    fi
    
    log_success "Prerequisites check completed"
}

# Security scan function
run_security_scan() {
    local image_name="$1"
    local severity="${TRIVY_SEVERITY:-HIGH,CRITICAL}"
    
    log_info "Running security scan for ${image_name}..."
    
    # Create scan results directory
    mkdir -p "${SCRIPT_DIR}/security-reports"
    
    # Run Trivy scan
    trivy image \
        --severity "$severity" \
        --format json \
        --output "${SCRIPT_DIR}/security-reports/${SERVICE}-$(date +%Y%m%d-%H%M%S).json" \
        "$image_name"
    
    # Run table format for console output
    trivy image \
        --severity "$severity" \
        --format table \
        "$image_name"
    
    log_success "Security scan completed for ${image_name}"
}

# Enhanced build function for individual service with 2025 optimizations
build_service() {
    local service_name="$1"
    local image_name="${REGISTRY}/${IMAGE_PREFIX}-${service_name}:${VERSION}"
    local latest_name="${REGISTRY}/${IMAGE_PREFIX}-${service_name}:latest"
    local branch_name="${REGISTRY}/${IMAGE_PREFIX}-${service_name}:${BRANCH_NAME}-${VCS_REF}"
    
    log_info "Building ${service_name} service..."
    log_info "Image: ${image_name}"
    log_info "Build date: ${BUILD_DATE}"
    log_info "VCS ref: ${VCS_REF}"
    
    # Determine service type
    local service_type="go"
    if [[ "$service_name" == "rag-api" ]]; then
        service_type="python"
    fi
    
    # Enhanced build arguments with 2025 standards
    local build_args=(
        --build-arg "SERVICE=${service_name}"
        --build-arg "SERVICE_TYPE=${service_type}"
        --build-arg "BUILD_DATE=${BUILD_DATE}"
        --build-arg "VCS_REF=${VCS_REF}"
        --build-arg "VERSION=${VERSION}"
        --build-arg "BUILDPLATFORM=linux/amd64"
        --build-arg "TARGETPLATFORM=${PLATFORMS%%,*}"
        --build-arg "TARGETOS=linux"
        --build-arg "TARGETARCH=amd64"
        --tag "${image_name}"
        --tag "${branch_name}"
        --provenance "${PROVENANCE}"
        --sbom "${SBOM}"
    )
    
    # Add latest tag only for main branch
    if [[ "$BRANCH_NAME" == "main" ]]; then
        build_args+=(--tag "${latest_name}")
    fi
    
    # Add custom build args
    if [[ -n "$BUILD_ARGS" ]]; then
        build_args+=($BUILD_ARGS)
    fi
    
    # Add no-cache if requested
    if [[ "$NO_CACHE" == true ]]; then
        build_args+=(--no-cache)
    fi
    
    # Build cache arguments with GitHub Actions optimization
    local cache_args=()
    
    # Auto-detect GitHub Actions cache or use provided scopes
    if [[ -n "${GITHUB_ACTIONS:-}" ]] && [[ -z "$CACHE_FROM_SCOPES" ]]; then
        cache_args+=(--cache-from "type=gha,scope=docker-${service_name}-${BRANCH_NAME}")
        cache_args+=(--cache-from "type=gha,scope=docker-${service_name}-main")
    else
        for scope in $CACHE_FROM_SCOPES; do
            cache_args+=(--cache-from "$scope")
        done
    fi
    
    if [[ -n "${GITHUB_ACTIONS:-}" ]] && [[ -z "$CACHE_TO_SCOPES" ]]; then
        cache_args+=(--cache-to "type=gha,mode=max,scope=docker-${service_name}-${BRANCH_NAME}")
    else
        for scope in $CACHE_TO_SCOPES; do
            cache_args+=(--cache-to "$scope")
        done
    fi
    
    # Setup enhanced buildx builder for 2025 standards
    local builder_name="nephoran-builder-2025"
    if [[ "$MULTI_ARCH" == true ]] || [[ -n "${GITHUB_ACTIONS:-}" ]]; then
        log_info "Setting up Docker buildx builder with 2025 optimizations"
        
        if ! docker buildx inspect "$builder_name" >/dev/null 2>&1; then
            docker buildx create --name "$builder_name" --use --driver docker-container \
                --bootstrap --config-inline '
[worker.oci]
  max-parallelism = 4
  gc-keep-storage = "4GB"
[registry."ghcr.io"]
  mirrors = ["mirror.gcr.io"]'
        else
            docker buildx use "$builder_name"
        fi
    fi
    
    # Build command with enhanced options
    local docker_cmd="docker buildx build"
    local common_args=(
        "${build_args[@]}"
        "${cache_args[@]}"
        --progress plain
        --metadata-file "/tmp/metadata-${service_name}.json"
        --file "${SCRIPT_DIR}/../Dockerfile"
    )
    
    # Add platform and output arguments
    if [[ "$MULTI_ARCH" == true ]]; then
        log_info "Building multi-architecture image for platforms: ${PLATFORMS}"
        common_args+=(--platform "${PLATFORMS}")
        
        if [[ "$PUSH" == true ]]; then
            common_args+=(--push)
        else
            # For multi-arch builds without push, use registry cache
            common_args+=(--output "type=image,push=false")
        fi
    else
        log_info "Building single-architecture image"
        common_args+=(--load)
    fi
    
    # Add no-cache if requested
    if [[ "$NO_CACHE" == true ]]; then
        common_args+=(--no-cache)
    fi
    
    # Final build command
    common_args+=("${SCRIPT_DIR}/..")
    
    # Execute build (dry run or actual)
    if [[ "${DRY_RUN:-false}" == "true" ]]; then
        log_info "DRY RUN - Build command:"
        echo "$docker_cmd ${common_args[*]}"
        return 0
    fi
    
    log_info "Executing build command..."
    if [[ "$VERBOSE" == "true" ]]; then
        $docker_cmd "${common_args[@]}"
    else
        $docker_cmd "${common_args[@]}" 2>&1 | while IFS= read -r line; do
            if [[ "$line" =~ ^#[0-9]+ || "$line" =~ "DONE" || "$line" =~ "ERROR" || "$line" =~ "WARNING" ]]; then
                echo "$line"
            fi
        done
    fi
    
    log_success "Successfully built ${service_name}"
    
    # Run security scan if requested
    if [[ "$SCAN" == true ]]; then
        run_security_scan "${image_name}"
    fi
    
    # Push single-arch images if not already pushed in multi-arch build
    if [[ "$PUSH" == true ]] && [[ "$MULTI_ARCH" != true ]]; then
        log_info "Pushing ${image_name}..."
        docker push "${image_name}"
        docker push "${branch_name}"
        if [[ "$BRANCH_NAME" == "main" ]]; then
            docker push "${latest_name}"
        fi
        log_success "Successfully pushed ${service_name} images"
    fi
    
    # Display build metadata if available
    if [[ -f "/tmp/metadata-${service_name}.json" ]] && [[ "$VERBOSE" == "true" ]]; then
        log_info "Build metadata for ${service_name}:"
        jq . "/tmp/metadata-${service_name}.json" 2>/dev/null || cat "/tmp/metadata-${service_name}.json"
    fi
}

# Main execution
main() {
    log_info "Starting Docker build process for Nephoran Intent Operator"
    log_info "Service: ${SERVICE}"
    log_info "Version: ${VERSION}"
    log_info "Registry: ${REGISTRY}"
    
    # Check prerequisites
    check_prerequisites
    
    # Change to script directory
    cd "${SCRIPT_DIR}"
    
    # Build services with enhanced service list
    if [[ "$SERVICE" == "all" ]]; then
        local services=("conductor-loop" "intent-ingest" "llm-processor" "nephio-bridge" "oran-adaptor" "porch-publisher" "planner" "rag-api")
        log_info "Building all services: ${services[*]}"
        
        for svc in "${services[@]}"; do
            log_info "Starting build for service: $svc"
            build_service "$svc"
            echo "" # Add spacing between builds
        done
    else
        build_service "$SERVICE"
    fi
    
    log_success "Docker build process completed successfully!"
    
    # Display final information
    echo
    log_info "Built images:"
    if [[ "$SERVICE" == "all" ]]; then
        for svc in llm-processor nephio-bridge oran-adaptor; do
            echo "  - ${REGISTRY}/${svc}:${VERSION}"
        done
    else
        echo "  - ${REGISTRY}/${SERVICE}:${VERSION}"
    fi
    
    if [[ "$SCAN" == true ]]; then
        echo
        log_info "Security scan reports available in: ${SCRIPT_DIR}/security-reports/"
    fi
}

# Enhanced cleanup function for 2025 standards
cleanup() {
    local builder_name="nephoran-builder-2025"
    
    # Cleanup temporary files
    rm -f /tmp/metadata-*.json 2>/dev/null || true
    
    # Cleanup buildx builder (only in local development, keep in CI)
    if [[ -z "${GITHUB_ACTIONS:-}" ]] && [[ "$MULTI_ARCH" == true ]] && docker buildx inspect "$builder_name" >/dev/null 2>&1; then
        log_info "Cleaning up buildx builder"
        docker buildx rm "$builder_name" 2>/dev/null || true
    fi
    
    # Cleanup unused images if not in CI
    if [[ -z "${GITHUB_ACTIONS:-}" ]] && command -v docker >/dev/null 2>&1; then
        docker system prune -f --filter "until=1h" 2>/dev/null || true
    fi
}
trap cleanup EXIT

# Run main function
main "$@"