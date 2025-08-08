#!/bin/bash
# =============================================================================
# Consolidated Docker Build Script for Nephoran Intent Operator
# =============================================================================
# Uses the 3 essential Dockerfiles for all build scenarios
# Supports production, development, and multi-architecture builds
# =============================================================================

set -e

# Configuration
REGISTRY="${DOCKER_REGISTRY:-nephoran}"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "latest")}"
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
VCS_REF=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Services to build
SERVICES=(
    "llm-processor:go"
    "nephio-bridge:go"
    "oran-adaptor:go"
    "manager:go"
    "rag-api:python"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
print_usage() {
    cat << EOF
Usage: $0 [OPTIONS] [SERVICE]

Build Docker images using consolidated Dockerfiles

OPTIONS:
    -p, --production    Build production images (default)
    -d, --dev          Build development images with debugging tools
    -m, --multiarch    Build multi-architecture images (amd64, arm64)
    -s, --service      Build specific service only
    -v, --version      Specify version tag (default: git tag or 'latest')
    -r, --registry     Specify Docker registry (default: 'nephoran')
    -h, --help         Show this help message

SERVICES:
    llm-processor      LLM processing service
    nephio-bridge      Nephio integration bridge
    oran-adaptor       O-RAN compliance adaptor
    manager            Main operator controller
    rag-api           RAG API service (Python)
    all               Build all services (default)

EXAMPLES:
    # Build all production images
    $0 --production

    # Build development image for specific service
    $0 --dev --service llm-processor

    # Build multi-architecture images and push
    $0 --multiarch --push

    # Build specific version
    $0 --version v2.0.0 --service nephio-bridge
EOF
}

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Build production image
build_production() {
    local service=$1
    local service_type=$2
    local tag="${REGISTRY}/${service}:${VERSION}"
    
    log_info "Building production image: $tag"
    
    docker build \
        --build-arg SERVICE="${service}" \
        --build-arg SERVICE_TYPE="${service_type}" \
        --build-arg VERSION="${VERSION}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        --build-arg VCS_REF="${VCS_REF}" \
        -t "${tag}" \
        -t "${REGISTRY}/${service}:latest" \
        -f Dockerfile \
        .
        
    log_info "Successfully built: $tag"
}

# Build development image
build_development() {
    local service=$1
    local service_type=$2
    local tag="${REGISTRY}/${service}:dev"
    
    log_info "Building development image: $tag"
    
    docker build \
        --build-arg SERVICE="${service}" \
        --build-arg SERVICE_TYPE="${service_type}" \
        --build-arg VERSION="dev" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        --build-arg VCS_REF="${VCS_REF}" \
        -t "${tag}" \
        -f Dockerfile.dev \
        .
        
    log_info "Successfully built: $tag"
}

# Build multi-architecture image
build_multiarch() {
    local service=$1
    local service_type=$2
    local tag="${REGISTRY}/${service}:${VERSION}"
    
    log_info "Building multi-architecture image: $tag"
    
    # Check if buildx is available
    if ! docker buildx version &>/dev/null; then
        log_error "Docker buildx is not available. Please install buildx plugin."
        exit 1
    fi
    
    # Create or use existing buildx builder
    if ! docker buildx ls | grep -q multiarch-builder; then
        log_info "Creating multiarch-builder..."
        docker buildx create --use --name multiarch-builder
    else
        docker buildx use multiarch-builder
    fi
    
    # Build for multiple platforms
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        --build-arg SERVICE="${service}" \
        --build-arg SERVICE_TYPE="${service_type}" \
        --build-arg VERSION="${VERSION}" \
        --build-arg BUILD_DATE="${BUILD_DATE}" \
        --build-arg VCS_REF="${VCS_REF}" \
        -t "${tag}" \
        -t "${REGISTRY}/${service}:latest" \
        -f Dockerfile.multiarch \
        --push \
        .
        
    log_info "Successfully built and pushed multi-arch: $tag"
}

# Parse command line arguments
BUILD_TYPE="production"
TARGET_SERVICE="all"
PUSH=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -p|--production)
            BUILD_TYPE="production"
            shift
            ;;
        -d|--dev)
            BUILD_TYPE="development"
            shift
            ;;
        -m|--multiarch)
            BUILD_TYPE="multiarch"
            shift
            ;;
        -s|--service)
            TARGET_SERVICE="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -r|--registry)
            REGISTRY="$2"
            shift 2
            ;;
        --push)
            PUSH=true
            shift
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            print_usage
            exit 1
            ;;
    esac
done

# Main build logic
log_info "Starting Docker build process"
log_info "Build Type: $BUILD_TYPE"
log_info "Version: $VERSION"
log_info "Registry: $REGISTRY"

# Determine which services to build
if [[ "$TARGET_SERVICE" == "all" ]]; then
    BUILD_SERVICES=("${SERVICES[@]}")
else
    # Find the service in the list
    found=false
    for svc in "${SERVICES[@]}"; do
        service_name="${svc%%:*}"
        if [[ "$service_name" == "$TARGET_SERVICE" ]]; then
            BUILD_SERVICES=("$svc")
            found=true
            break
        fi
    done
    
    if [[ "$found" == "false" ]]; then
        log_error "Unknown service: $TARGET_SERVICE"
        print_usage
        exit 1
    fi
fi

# Build each service
for service_info in "${BUILD_SERVICES[@]}"; do
    service_name="${service_info%%:*}"
    service_type="${service_info##*:}"
    
    log_info "Building $service_name (type: $service_type)"
    
    case $BUILD_TYPE in
        production)
            build_production "$service_name" "$service_type"
            ;;
        development)
            build_development "$service_name" "$service_type"
            ;;
        multiarch)
            build_multiarch "$service_name" "$service_type"
            ;;
    esac
done

# Push images if requested (for non-multiarch builds)
if [[ "$PUSH" == "true" ]] && [[ "$BUILD_TYPE" != "multiarch" ]]; then
    log_info "Pushing images to registry..."
    for service_info in "${BUILD_SERVICES[@]}"; do
        service_name="${service_info%%:*}"
        if [[ "$BUILD_TYPE" == "production" ]]; then
            docker push "${REGISTRY}/${service_name}:${VERSION}"
            docker push "${REGISTRY}/${service_name}:latest"
        else
            docker push "${REGISTRY}/${service_name}:dev"
        fi
    done
fi

log_info "Docker build process completed successfully!"

# Print summary
echo ""
echo "========================================="
echo "Build Summary:"
echo "========================================="
for service_info in "${BUILD_SERVICES[@]}"; do
    service_name="${service_info%%:*}"
    if [[ "$BUILD_TYPE" == "production" ]]; then
        echo "  - ${REGISTRY}/${service_name}:${VERSION}"
    elif [[ "$BUILD_TYPE" == "development" ]]; then
        echo "  - ${REGISTRY}/${service_name}:dev"
    else
        echo "  - ${REGISTRY}/${service_name}:${VERSION} (multi-arch)"
    fi
done
echo "========================================="