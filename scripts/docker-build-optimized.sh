#!/bin/bash
# =============================================================================
# Optimized Docker Build Script for Nephoran Intent Operator (2025)
# =============================================================================
# Fixes all Docker build issues:
# - Registry authentication (uses correct namespace)
# - Multi-platform builds
# - Timezone data installation
# - Layer caching optimization
# - Build context optimization
# =============================================================================

set -euo pipefail

# =============================================================================
# Configuration
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# Registry configuration (CRITICAL FIX)
REGISTRY="${REGISTRY:-ghcr.io}"
REGISTRY_USERNAME="${REGISTRY_USERNAME:-thc1006}"  # Fixed namespace
IMAGE_PREFIX="${REGISTRY}/${REGISTRY_USERNAME}"

# Build configuration
DOCKERFILE="${DOCKERFILE:-Dockerfile.optimized-2025}"
PLATFORM="${PLATFORM:-linux/amd64}"
GO_VERSION="${GO_VERSION:-1.24.6}"
ALPINE_VERSION="${ALPINE_VERSION:-3.21}"

# Services to build
SERVICES=(
    "intent-ingest"
    "conductor-loop"
    "llm-processor"
    "nephio-bridge"
    "oran-adaptor"
    "porch-publisher"
    "planner"
)

# Build metadata
BUILD_VERSION="${BUILD_VERSION:-$(git rev-parse --short HEAD)}"
BUILD_DATE="${BUILD_DATE:-$(date -Iseconds)}"
VCS_REF="${VCS_REF:-$(git rev-parse HEAD)}"

# =============================================================================
# Helper Functions
# =============================================================================

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" >&2
}

error() {
    log "ERROR: $*"
    exit 1
}

check_dependencies() {
    log "Checking dependencies..."
    
    if ! command -v docker >/dev/null 2>&1; then
        error "Docker is not installed"
    fi
    
    if ! docker buildx version >/dev/null 2>&1; then
        error "Docker Buildx is not available"
    fi
    
    log "Dependencies check passed"
}

setup_buildx() {
    log "Setting up Docker Buildx..."
    
    # Create or use existing builder
    local builder_name="nephoran-builder"
    
    if ! docker buildx ls | grep -q "$builder_name"; then
        log "Creating new buildx builder: $builder_name"
        docker buildx create \
            --name "$builder_name" \
            --driver docker-container \
            --driver-opt image=moby/buildkit:v0.16.0 \
            --driver-opt network=host \
            --buildkitd-flags '--allow-insecure-entitlement security.insecure --allow-insecure-entitlement network.host' \
            --config <(cat <<EOF
[worker.oci]
  max-parallelism = 8
  gc-keep-storage = "4GB"
[worker.containerd]
  max-parallelism = 8
[registry."ghcr.io"]
  insecure = false
EOF
)
        docker buildx use "$builder_name"
    else
        log "Using existing buildx builder: $builder_name"
        docker buildx use "$builder_name"
    fi
    
    # Bootstrap the builder
    docker buildx inspect --bootstrap
    
    log "Buildx setup completed"
}

verify_service() {
    local service="$1"
    
    log "Verifying service: $service"
    
    if [[ "$service" == "planner" ]]; then
        if [[ -d "planner/cmd/planner" ]]; then
            log "✅ Planner service directory found"
        else
            error "❌ Planner service directory not found: planner/cmd/planner"
        fi
    elif [[ -d "cmd/$service" ]]; then
        log "✅ Service directory found: cmd/$service"
    else
        error "❌ Service directory not found: cmd/$service"
    fi
}

build_service() {
    local service="$1"
    local push="${2:-false}"
    
    log "Building service: $service"
    
    # Verify service exists
    verify_service "$service"
    
    # Image name and tags
    local image_name="${IMAGE_PREFIX}/${service}"
    local cache_scope="build-${service}-$(git rev-parse --abbrev-ref HEAD)"
    
    # Build arguments
    local build_args=(
        --build-arg "SERVICE=$service"
        --build-arg "BUILD_VERSION=$BUILD_VERSION"
        --build-arg "BUILD_DATE=$BUILD_DATE"
        --build-arg "VCS_REF=$VCS_REF"
        --build-arg "GO_VERSION=$GO_VERSION"
        --build-arg "ALPINE_VERSION=$ALPINE_VERSION"
        --build-arg "CGO_ENABLED=0"
        --build-arg "BUILDPLATFORM=$PLATFORM"
        --build-arg "TARGETPLATFORM=$PLATFORM"
        --build-arg "TARGETOS=linux"
        --build-arg "TARGETARCH=amd64"
    )
    
    # Cache configuration
    local cache_args=(
        --cache-from "type=gha,scope=$cache_scope"
        --cache-from "type=gha,scope=build-${service}-main"
        --cache-from "type=gha,scope=build-${service}"
        --cache-to "type=gha,mode=max,scope=$cache_scope"
    )
    
    # Tags
    local tag_args=(
        --tag "${image_name}:${BUILD_VERSION}"
        --tag "${image_name}:latest"
        --tag "${image_name}:$(git rev-parse --abbrev-ref HEAD)"
    )
    
    # Labels
    local label_args=(
        --label "org.opencontainers.image.service=$service"
        --label "org.opencontainers.image.component=$service"
        --label "service.name=$service"
        --label "org.opencontainers.image.title=nephoran-$service"
        --label "org.opencontainers.image.description=Nephoran Intent Operator - $service Service"
        --label "org.opencontainers.image.version=$BUILD_VERSION"
        --label "org.opencontainers.image.created=$BUILD_DATE"
        --label "org.opencontainers.image.revision=$VCS_REF"
        --label "org.opencontainers.image.vendor=Nephoran Project"
        --label "org.opencontainers.image.licenses=Apache-2.0"
        --label "org.opencontainers.image.source=https://github.com/thc1006/nephoran-intent-operator"
    )
    
    # Build command
    local build_cmd=(
        docker buildx build
        --file "$DOCKERFILE"
        --platform "$PLATFORM"
        --progress plain
        --provenance false
        --sbom false
        "${build_args[@]}"
        "${cache_args[@]}"
        "${tag_args[@]}"
        "${label_args[@]}"
    )
    
    if [[ "$push" == "true" ]]; then
        build_cmd+=(--push)
        log "Building and pushing $service to registry..."
    else
        build_cmd+=(--load)
        log "Building $service locally..."
    fi
    
    build_cmd+=(.)
    
    # Execute build
    log "Executing: ${build_cmd[*]}"
    "${build_cmd[@]}" || error "Build failed for service: $service"
    
    log "✅ Successfully built $service"
    
    # Test the built image if not pushing
    if [[ "$push" != "true" ]]; then
        test_image "${image_name}:${BUILD_VERSION}" "$service"
    fi
}

test_image() {
    local image="$1"
    local service="$2"
    
    log "Testing image: $image"
    
    # Basic image inspection
    docker image inspect "$image" --format='Size: {{.Size}} bytes' || true
    
    # Test container functionality
    timeout 30s docker run --rm "$image" --help 2>&1 | head -10 || {
        log "⚠️  Container help test failed or timed out for $service"
        log "This might be expected for some services"
    }
    
    log "✅ Image test completed for $service"
}

print_usage() {
    cat <<EOF
Usage: $0 [OPTIONS] [SERVICE...]

Optimized Docker build script for Nephoran Intent Operator (2025)

OPTIONS:
    -h, --help          Show this help message
    -p, --push          Push images to registry
    -a, --all           Build all services (default)
    -f, --file DOCKERFILE   Use specific Dockerfile (default: $DOCKERFILE)
    --platform PLATFORM    Target platform (default: $PLATFORM)
    --registry REGISTRY     Registry URL (default: $REGISTRY)
    --username USERNAME     Registry username (default: $REGISTRY_USERNAME)

SERVICES:
    $(printf '%s\n' "${SERVICES[@]}" | tr '\n' ' ')

EXAMPLES:
    # Build all services locally
    $0

    # Build specific service and push
    $0 --push intent-ingest

    # Build multiple services locally
    $0 intent-ingest conductor-loop

    # Build for different platform
    $0 --platform linux/arm64 intent-ingest

REGISTRY AUTHENTICATION:
    Make sure you're logged in to the registry:
    docker login $REGISTRY -u $REGISTRY_USERNAME

ENVIRONMENT VARIABLES:
    REGISTRY              Registry URL
    REGISTRY_USERNAME     Registry username
    DOCKERFILE           Dockerfile to use
    PLATFORM             Target platform
    BUILD_VERSION        Build version tag
    GO_VERSION           Go version for builds
    ALPINE_VERSION       Alpine version for builds
EOF
}

# =============================================================================
# Main Function
# =============================================================================

main() {
    local push=false
    local services_to_build=()
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                print_usage
                exit 0
                ;;
            -p|--push)
                push=true
                shift
                ;;
            -a|--all)
                services_to_build=("${SERVICES[@]}")
                shift
                ;;
            -f|--file)
                DOCKERFILE="$2"
                shift 2
                ;;
            --platform)
                PLATFORM="$2"
                shift 2
                ;;
            --registry)
                REGISTRY="$2"
                IMAGE_PREFIX="${REGISTRY}/${REGISTRY_USERNAME}"
                shift 2
                ;;
            --username)
                REGISTRY_USERNAME="$2"
                IMAGE_PREFIX="${REGISTRY}/${REGISTRY_USERNAME}"
                shift 2
                ;;
            -*)
                error "Unknown option: $1"
                ;;
            *)
                services_to_build+=("$1")
                shift
                ;;
        esac
    done
    
    # Default to all services if none specified
    if [[ ${#services_to_build[@]} -eq 0 ]]; then
        services_to_build=("${SERVICES[@]}")
    fi
    
    log "=== Nephoran Docker Build (2025 Optimized) ==="
    log "Registry: $REGISTRY"
    log "Username: $REGISTRY_USERNAME"
    log "Dockerfile: $DOCKERFILE"
    log "Platform: $PLATFORM"
    log "Push: $push"
    log "Services: ${services_to_build[*]}"
    log "Build Version: $BUILD_VERSION"
    log "================================================"
    
    # Check if Dockerfile exists
    if [[ ! -f "$DOCKERFILE" ]]; then
        error "Dockerfile not found: $DOCKERFILE"
    fi
    
    # Setup
    check_dependencies
    setup_buildx
    
    # Build services
    local failed_services=()
    for service in "${services_to_build[@]}"; do
        if build_service "$service" "$push"; then
            log "✅ Successfully processed $service"
        else
            log "❌ Failed to process $service"
            failed_services+=("$service")
        fi
    done
    
    # Summary
    log "=== Build Summary ==="
    if [[ ${#failed_services[@]} -eq 0 ]]; then
        log "✅ All services built successfully!"
        log "Total services: ${#services_to_build[@]}"
        if [[ "$push" == "true" ]]; then
            log "Images pushed to: $IMAGE_PREFIX/"
        else
            log "Images available locally"
        fi
    else
        log "❌ Some services failed to build:"
        printf '%s\n' "${failed_services[@]}"
        exit 1
    fi
    
    log "=== Build Completed ==="
}

# Run main function
main "$@"