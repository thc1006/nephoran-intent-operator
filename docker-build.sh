#!/bin/bash

# Production Docker build script with security scanning and multi-architecture support
# Usage: ./docker-build.sh <service> [options]

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
VCS_REF=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION=${VERSION:-v2.0.0}
REGISTRY=${REGISTRY:-ghcr.io/thc1006/nephoran-intent-operator}
PLATFORMS=${PLATFORMS:-linux/amd64,linux/arm64}

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
  llm-processor    Build LLM Processor service
  nephio-bridge    Build Nephio Bridge service
  oran-adaptor     Build ORAN Adaptor service
  all              Build all services

Options:
  --push           Push images to registry after build
  --scan           Run security scan using Trivy
  --multi-arch     Build for multiple architectures (${PLATFORMS})
  --no-cache       Build without using Docker cache
  --registry       Set custom registry (default: ${REGISTRY})
  --version        Set version tag (default: ${VERSION})
  --platforms      Set target platforms (default: ${PLATFORMS})
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

while [[ $# -gt 0 ]]; do
    case $1 in
        llm-processor|nephio-bridge|oran-adaptor|all)
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

# Build function for individual service
build_service() {
    local service_name="$1"
    local image_name="${REGISTRY}/${service_name}:${VERSION}"
    local latest_name="${REGISTRY}/${service_name}:latest"
    
    log_info "Building ${service_name} service..."
    log_info "Image: ${image_name}"
    log_info "Build date: ${BUILD_DATE}"
    log_info "VCS ref: ${VCS_REF}"
    
    # Build arguments
    local build_args=(
        --build-arg "SERVICE=${service_name}"
        --build-arg "BUILD_DATE=${BUILD_DATE}"
        --build-arg "VCS_REF=${VCS_REF}"
        --build-arg "VERSION=${VERSION}"
        --tag "${image_name}"
        --tag "${latest_name}"
    )
    
    # Add no-cache if requested
    if [[ "$NO_CACHE" == true ]]; then
        build_args+=(--no-cache)
    fi
    
    # Choose build method based on multi-arch requirement
    if [[ "$MULTI_ARCH" == true ]]; then
        log_info "Building multi-architecture image for platforms: ${PLATFORMS}"
        
        # Create builder if it doesn't exist
        if ! docker buildx inspect nephoran-builder >/dev/null 2>&1; then
            docker buildx create --name nephoran-builder --use
        fi
        
        docker buildx build \
            "${build_args[@]}" \
            --platform "${PLATFORMS}" \
            --progress plain \
            --file "${SCRIPT_DIR}/Dockerfile" \
            "${SCRIPT_DIR}"
            
    else
        log_info "Building single-architecture image"
        docker build \
            "${build_args[@]}" \
            --progress plain \
            --file "${SCRIPT_DIR}/Dockerfile" \
            "${SCRIPT_DIR}"
    fi
    
    log_success "Successfully built ${service_name}"
    
    # Run security scan if requested
    if [[ "$SCAN" == true ]]; then
        run_security_scan "${image_name}"
    fi
    
    # Push if requested
    if [[ "$PUSH" == true ]]; then
        log_info "Pushing ${image_name}..."
        docker push "${image_name}"
        docker push "${latest_name}"
        log_success "Successfully pushed ${image_name}"
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
    
    # Build services
    if [[ "$SERVICE" == "all" ]]; then
        for svc in llm-processor nephio-bridge oran-adaptor; do
            build_service "$svc"
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

# Trap to cleanup on exit
cleanup() {
    if [[ "$MULTI_ARCH" == true ]] && docker buildx inspect nephoran-builder >/dev/null 2>&1; then
        docker buildx rm nephoran-builder 2>/dev/null || true
    fi
}
trap cleanup EXIT

# Run main function
main "$@"