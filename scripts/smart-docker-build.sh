#!/bin/bash
# =============================================================================
# Smart Docker Build Script with Infrastructure-Aware Fallback Strategy
# =============================================================================
# This script automatically selects the most appropriate build strategy based
# on the current state of external services and infrastructure availability.
# =============================================================================

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DOCKERFILE_MAIN="$PROJECT_ROOT/Dockerfile"
DOCKERFILE_RESILIENT="$PROJECT_ROOT/Dockerfile.resilient"
HEALTH_CHECK_TIMEOUT=30
MAX_RETRY_ATTEMPTS=3

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Infrastructure health check function
check_infrastructure_health() {
    log_info "Performing comprehensive infrastructure health check..."
    
    local health_score=0
    local max_score=100
    
    # Test 1: GitHub Actions cache service (20 points)
    log_info "Testing GitHub Actions cache availability..."
    if timeout ${HEALTH_CHECK_TIMEOUT}s docker buildx build \
        --platform linux/amd64 \
        --cache-from=type=gha \
        --cache-to=type=gha,mode=max,ignore-error=true \
        -f /dev/null . >/dev/null 2>&1; then
        log_success "GHA cache service responsive"
        health_score=$((health_score + 20))
        GHA_CACHE_HEALTHY=true
    else
        log_warning "GHA cache service experiencing issues"
        GHA_CACHE_HEALTHY=false
    fi
    
    # Test 2: Primary registries (30 points total - 10 each)
    local registries=("gcr.io" "docker.io" "ghcr.io")
    local healthy_registries=0
    
    for registry in "${registries[@]}"; do
        log_info "Testing $registry connectivity..."
        if timeout 15s docker buildx imagetools inspect "$registry/hello-world:latest" >/dev/null 2>&1; then
            log_success "$registry is responsive"
            health_score=$((health_score + 10))
            healthy_registries=$((healthy_registries + 1))
        else
            log_warning "$registry may be experiencing issues"
        fi
    done
    
    # Test 3: Go proxy services (25 points)
    log_info "Testing Go proxy services..."
    local go_proxies=("proxy.golang.org" "goproxy.cn" "goproxy.io")
    local healthy_proxies=0
    
    for proxy in "${go_proxies[@]}"; do
        if timeout 10s curl -s "https://$proxy" >/dev/null 2>&1; then
            healthy_proxies=$((healthy_proxies + 1))
        fi
    done
    
    if [ $healthy_proxies -gt 0 ]; then
        proxy_points=$((healthy_proxies * 8))
        health_score=$((health_score + proxy_points))
        log_success "$healthy_proxies/$#{go_proxies[@]} Go proxies responsive"
    else
        log_warning "All Go proxy services may be experiencing issues"
    fi
    
    # Test 4: DNS resolution (15 points)
    log_info "Testing DNS resolution..."
    if timeout 5s nslookup google.com >/dev/null 2>&1; then
        log_success "DNS resolution working"
        health_score=$((health_score + 15))
    else
        log_warning "DNS resolution issues detected"
    fi
    
    # Test 5: Internet connectivity (10 points)
    log_info "Testing internet connectivity..."
    if timeout 10s ping -c 1 8.8.8.8 >/dev/null 2>&1; then
        log_success "Internet connectivity confirmed"
        health_score=$((health_score + 10))
    else
        log_warning "Internet connectivity issues"
    fi
    
    # Calculate health percentage
    local health_percentage=$((health_score * 100 / max_score))
    
    log_info "Infrastructure health score: $health_score/$max_score ($health_percentage%)"
    log_info "Healthy registries: $healthy_registries/${#registries[@]}"
    log_info "Healthy Go proxies: $healthy_proxies/${#go_proxies[@]}"
    
    # Export results
    export INFRASTRUCTURE_HEALTH_SCORE=$health_score
    export INFRASTRUCTURE_HEALTH_PERCENTAGE=$health_percentage
    export HEALTHY_REGISTRIES=$healthy_registries
    export HEALTHY_GO_PROXIES=$healthy_proxies
    
    return 0
}

# Strategy selection function
select_build_strategy() {
    local health_percentage=${INFRASTRUCTURE_HEALTH_PERCENTAGE:-0}
    local healthy_registries=${HEALTHY_REGISTRIES:-0}
    
    log_info "Selecting optimal build strategy based on infrastructure health..."
    
    if [ $health_percentage -ge 80 ]; then
        SELECTED_STRATEGY="optimal"
        DOCKERFILE_PATH="$DOCKERFILE_MAIN"
        CACHE_STRATEGY="gha-full"
        BUILD_TIMEOUT=2400  # 40 minutes
        log_success "Using OPTIMAL strategy (health: ${health_percentage}%)"
    elif [ $health_percentage -ge 60 ] && [ $healthy_registries -ge 2 ]; then
        SELECTED_STRATEGY="resilient"
        DOCKERFILE_PATH="$DOCKERFILE_RESILIENT"
        CACHE_STRATEGY="local-with-fallback"
        BUILD_TIMEOUT=1800  # 30 minutes
        log_info "Using RESILIENT strategy (health: ${health_percentage}%)"
    elif [ $health_percentage -ge 40 ]; then
        SELECTED_STRATEGY="conservative"
        DOCKERFILE_PATH="$DOCKERFILE_RESILIENT"
        CACHE_STRATEGY="local-only"
        BUILD_TIMEOUT=1200  # 20 minutes
        log_warning "Using CONSERVATIVE strategy (health: ${health_percentage}%)"
    else
        SELECTED_STRATEGY="emergency"
        DOCKERFILE_PATH="$DOCKERFILE_RESILIENT"
        CACHE_STRATEGY="no-cache"
        BUILD_TIMEOUT=900   # 15 minutes
        log_error "Using EMERGENCY strategy (health: ${health_percentage}%)"
    fi
    
    export SELECTED_STRATEGY
    export DOCKERFILE_PATH
    export CACHE_STRATEGY
    export BUILD_TIMEOUT
    
    log_info "Build configuration:"
    log_info "  Strategy: $SELECTED_STRATEGY"
    log_info "  Dockerfile: $(basename $DOCKERFILE_PATH)"
    log_info "  Cache strategy: $CACHE_STRATEGY"
    log_info "  Timeout: ${BUILD_TIMEOUT}s"
}

# Build execution function
execute_build() {
    local service=${1:-"conductor-loop"}
    local image_name=${2:-"nephoran/conductor-loop"}
    local version=${3:-"latest"}
    local platforms=${4:-"linux/amd64,linux/arm64"}
    local push=${5:-"false"}
    
    log_info "Executing Docker build with selected strategy..."
    log_info "Service: $service"
    log_info "Image: $image_name:$version"
    log_info "Platforms: $platforms"
    log_info "Push: $push"
    
    # Prepare cache arguments based on strategy
    local cache_args=""
    case "$CACHE_STRATEGY" in
        "gha-full")
            cache_args="--cache-from=type=gha --cache-from=type=local,src=/tmp/cache"
            cache_args="$cache_args --cache-to=type=gha,mode=max --cache-to=type=local,dest=/tmp/cache,mode=max"
            ;;
        "local-with-fallback")
            cache_args="--cache-from=type=local,src=/tmp/cache --cache-from=type=registry,ref=$image_name:cache,ignore-error=true"
            cache_args="$cache_args --cache-to=type=local,dest=/tmp/cache,mode=max"
            ;;
        "local-only")
            cache_args="--cache-from=type=local,src=/tmp/cache --cache-to=type=local,dest=/tmp/cache,mode=max"
            ;;
        "no-cache")
            cache_args="--no-cache"
            ;;
    esac
    
    # Prepare build arguments
    local build_args=(
        "--platform" "$platforms"
        "--build-arg" "SERVICE=$service"
        "--build-arg" "VERSION=$version"
        "--build-arg" "BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
        "--build-arg" "VCS_REF=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
        "-f" "$DOCKERFILE_PATH"
        "-t" "$image_name:$version"
    )
    
    # Add cache arguments
    if [ -n "$cache_args" ]; then
        read -ra cache_array <<< "$cache_args"
        build_args+=("${cache_array[@]}")
    fi
    
    # Add push/load argument
    if [ "$push" = "true" ]; then
        build_args+=("--push")
    else
        build_args+=("--load")
    fi
    
    # Add context
    build_args+=(".")
    
    # Prepare cache directory
    mkdir -p /tmp/cache
    
    # Execute build with timeout and retry logic
    local attempt=1
    local build_success=false
    
    while [ $attempt -le $MAX_RETRY_ATTEMPTS ] && [ "$build_success" = "false" ]; do
        log_info "Build attempt $attempt/$MAX_RETRY_ATTEMPTS"
        
        local start_time=$(date +%s)
        
        if timeout ${BUILD_TIMEOUT}s docker buildx build "${build_args[@]}"; then
            local end_time=$(date +%s)
            local duration=$((end_time - start_time))
            log_success "Build completed successfully in ${duration}s"
            build_success=true
        else
            local exit_code=$?
            local end_time=$(date +%s)
            local duration=$((end_time - start_time))
            
            log_error "Build attempt $attempt failed (exit code: $exit_code, duration: ${duration}s)"
            
            if [ $attempt -lt $MAX_RETRY_ATTEMPTS ]; then
                local backoff=$((attempt * 30))
                log_info "Retrying in ${backoff}s..."
                sleep $backoff
                
                # Clean up for retry
                docker system prune -f --volumes >/dev/null 2>&1 || true
                docker builder prune -f >/dev/null 2>&1 || true
            fi
        fi
        
        attempt=$((attempt + 1))
    done
    
    if [ "$build_success" = "true" ]; then
        log_success "Docker build completed successfully using $SELECTED_STRATEGY strategy"
        return 0
    else
        log_error "All build attempts failed with $SELECTED_STRATEGY strategy"
        return 1
    fi
}

# Main execution function
main() {
    local service=${1:-"conductor-loop"}
    local image_name=${2:-"nephoran/conductor-loop"}
    local version=${3:-"latest"}
    local platforms=${4:-"linux/amd64,linux/arm64"}
    local push=${5:-"false"}
    
    log_info "Starting smart Docker build process..."
    log_info "Target service: $service"
    
    # Step 1: Health check
    check_infrastructure_health
    
    # Step 2: Strategy selection
    select_build_strategy
    
    # Step 3: Build execution
    if execute_build "$service" "$image_name" "$version" "$platforms" "$push"; then
        log_success "Smart Docker build process completed successfully!"
        
        # Output summary
        echo ""
        log_info "Build Summary:"
        log_info "  Strategy used: $SELECTED_STRATEGY"
        log_info "  Infrastructure health: ${INFRASTRUCTURE_HEALTH_PERCENTAGE}%"
        log_info "  Dockerfile: $(basename $DOCKERFILE_PATH)"
        log_info "  Cache strategy: $CACHE_STRATEGY"
        log_info "  Service: $service"
        log_info "  Image: $image_name:$version"
        log_info "  Platforms: $platforms"
        
        return 0
    else
        log_error "Smart Docker build process failed!"
        
        # Output failure analysis
        echo ""
        log_error "Failure Analysis:"
        log_error "  Strategy attempted: $SELECTED_STRATEGY"
        log_error "  Infrastructure health: ${INFRASTRUCTURE_HEALTH_PERCENTAGE}%"
        log_error "  Healthy registries: $HEALTHY_REGISTRIES/3"
        log_error "  Healthy Go proxies: $HEALTHY_GO_PROXIES/3"
        
        return 1
    fi
}

# Script entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi