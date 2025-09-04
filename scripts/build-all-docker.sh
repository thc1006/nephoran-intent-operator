#!/bin/bash
# Build all Docker services for Nephoran Intent Operator
# Production-ready build script with parallel builds and error handling

set -euo pipefail

# Configuration
REGISTRY=${REGISTRY:-"nephoran"}
VERSION=${VERSION:-"$(git describe --tags --always --dirty)"}
BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
VCS_REF=$(git rev-parse HEAD)
PARALLEL_BUILDS=${PARALLEL_BUILDS:-4}
DOCKER_BUILDKIT=${DOCKER_BUILDKIT:-1}

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

# Services to build
SERVICES=(
    "intent-ingest"
    "llm-processor"
    "nephio-bridge"
    "oran-adaptor"
    "conductor"
    "conductor-loop"
    "porch-publisher"
    "a1-sim"
    "e2-kpm-sim"
    "fcaps-sim"
    "o1-ves-sim"
)

# Build function for a single service
build_service() {
    local service=$1
    local dockerfile="cmd/${service}/Dockerfile"
    local image_tag="${REGISTRY}/${service}:${VERSION}"
    local latest_tag="${REGISTRY}/${service}:latest"
    
    log_info "Building ${service}..."
    
    if [[ ! -f "${dockerfile}" ]]; then
        log_error "Dockerfile not found: ${dockerfile}"
        return 1
    fi
    
    # Build with BuildKit for better performance and security
    DOCKER_BUILDKIT=1 docker build \
        --file "${dockerfile}" \
        --tag "${image_tag}" \
        --tag "${latest_tag}" \
        --build-arg VERSION="${VERSION}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        --build-arg VCS_REF="${VCS_REF}" \
        --build-arg CGO_ENABLED=0 \
        --target runtime \
        --progress=plain \
        --no-cache \
        . || {
        log_error "Failed to build ${service}"
        return 1
    }
    
    log_success "Built ${service} -> ${image_tag}"
    
    # Security: Scan the image
    if command -v trivy >/dev/null 2>&1; then
        log_info "Scanning ${service} for vulnerabilities..."
        trivy image --exit-code 0 --severity HIGH,CRITICAL "${image_tag}" || {
            log_warn "Security scan found issues in ${service}"
        }
    fi
    
    return 0
}

# Build all services in parallel
build_all_parallel() {
    local pids=()
    local failed_services=()
    
    log_info "Starting parallel build of ${#SERVICES[@]} services (max ${PARALLEL_BUILDS} concurrent)..."
    
    for service in "${SERVICES[@]}"; do
        # Wait if we've reached max parallel builds
        while (( ${#pids[@]} >= PARALLEL_BUILDS )); do
            for i in "${!pids[@]}"; do
                if ! kill -0 "${pids[i]}" 2>/dev/null; then
                    wait "${pids[i]}" || {
                        failed_services+=("${services_running[i]}")
                    }
                    unset 'pids[i]'
                    unset 'services_running[i]'
                fi
            done
            # Reindex arrays
            pids=("${pids[@]}")
            services_running=("${services_running[@]}")
            sleep 1
        done
        
        # Start build in background
        (
            build_service "${service}"
        ) &
        
        pids+=($!)
        services_running+=("${service}")
    done
    
    # Wait for remaining builds
    for i in "${!pids[@]}"; do
        wait "${pids[i]}" || {
            failed_services+=("${services_running[i]}")
        }
    done
    
    if (( ${#failed_services[@]} > 0 )); then
        log_error "Failed to build services: ${failed_services[*]}"
        return 1
    fi
    
    log_success "All services built successfully!"
    return 0
}

# Build all services sequentially (fallback)
build_all_sequential() {
    local failed_services=()
    
    log_info "Starting sequential build of ${#SERVICES[@]} services..."
    
    for service in "${SERVICES[@]}"; do
        if ! build_service "${service}"; then
            failed_services+=("${service}")
        fi
    done
    
    if (( ${#failed_services[@]} > 0 )); then
        log_error "Failed to build services: ${failed_services[*]}"
        return 1
    fi
    
    log_success "All services built successfully!"
    return 0
}

# Multi-architecture build function
build_multiarch() {
    local platforms="linux/amd64,linux/arm64"
    log_info "Building multi-architecture images for platforms: ${platforms}"
    
    # Create/use buildx builder
    if ! docker buildx inspect nephoran-builder >/dev/null 2>&1; then
        log_info "Creating buildx builder..."
        docker buildx create --name nephoran-builder --use
    else
        docker buildx use nephoran-builder
    fi
    
    for service in "${SERVICES[@]}"; do
        local dockerfile="cmd/${service}/Dockerfile"
        local image_tag="${REGISTRY}/${service}:${VERSION}"
        
        if [[ ! -f "${dockerfile}" ]]; then
            log_error "Dockerfile not found: ${dockerfile}"
            continue
        fi
        
        log_info "Building ${service} for multiple architectures..."
        
        docker buildx build \
            --platform "${platforms}" \
            --file "${dockerfile}" \
            --tag "${image_tag}" \
            --build-arg VERSION="${VERSION}" \
            --build-arg BUILD_DATE="${BUILD_DATE}" \
            --build-arg VCS_REF="${VCS_REF}" \
            --build-arg CGO_ENABLED=0 \
            --target runtime \
            --push \
            . || {
            log_error "Failed to build ${service} for multiple architectures"
            continue
        }
        
        log_success "Built ${service} for multiple architectures"
    done
}

# Cleanup function
cleanup() {
    log_info "Cleaning up build artifacts..."
    docker system prune -f --filter "label=stage=builder" >/dev/null 2>&1 || true
}

# Main execution
main() {
    # Trap to cleanup on exit
    trap cleanup EXIT
    
    log_info "=== Nephoran Docker Build Script ==="
    log_info "Registry: ${REGISTRY}"
    log_info "Version: ${VERSION}"
    log_info "Build Date: ${BUILD_DATE}"
    log_info "VCS Ref: ${VCS_REF}"
    log_info "Services: ${SERVICES[*]}"
    
    # Check prerequisites
    if ! command -v docker >/dev/null 2>&1; then
        log_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker daemon is not running"
        exit 1
    fi
    
    # Parse command line arguments
    case "${1:-parallel}" in
        "sequential")
            build_all_sequential
            ;;
        "multiarch")
            build_multiarch
            ;;
        "parallel"|*)
            build_all_parallel
            ;;
    esac
    
    local exit_code=$?
    
    if (( exit_code == 0 )); then
        log_success "=== Build completed successfully! ==="
        log_info "Images built with tag: ${VERSION}"
        log_info "To run all services: docker-compose -f docker-compose.services.yml up -d"
    else
        log_error "=== Build failed! ==="
        exit 1
    fi
}

# Show usage if help requested
if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
    cat << EOF
Usage: $0 [BUILD_MODE]

BUILD_MODE:
  parallel   - Build services in parallel (default)
  sequential - Build services one by one
  multiarch  - Build multi-architecture images (requires docker buildx)

Environment Variables:
  REGISTRY      - Docker registry prefix (default: nephoran)
  VERSION       - Image version tag (default: git describe)
  PARALLEL_BUILDS - Max concurrent builds (default: 4)

Examples:
  $0                    # Parallel build
  $0 sequential         # Sequential build
  $0 multiarch          # Multi-architecture build
  
  REGISTRY=myregistry VERSION=v1.0.0 $0  # Custom registry and version

Prerequisites:
  - Docker
  - Git
  - Optional: trivy (for security scanning)
  - Optional: docker buildx (for multi-arch builds)
EOF
    exit 0
fi

main "$@"