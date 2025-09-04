#!/bin/bash
set -euo pipefail

# =============================================================================
# Docker Build Cache Fix for GitHub Actions
# =============================================================================
# This script addresses the specific cache import failures seen in CI builds:
# - GitHub Actions cache service unavailability (400 errors)
# - GHCR registry cache "not found" errors
# - BuildKit cache manifest parsing failures
#
# The solution implements multiple fallback strategies and graceful degradation.
# =============================================================================

# Configuration
readonly SCRIPT_NAME="docker-cache-fix"
readonly LOG_PREFIX="[$SCRIPT_NAME]"

# Color codes for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}${LOG_PREFIX}${NC} $*" >&2
}

log_warn() {
    echo -e "${YELLOW}${LOG_PREFIX}${NC} $*" >&2
}

log_error() {
    echo -e "${RED}${LOG_PREFIX}${NC} $*" >&2
}

log_success() {
    echo -e "${GREEN}${LOG_PREFIX}${NC} $*" >&2
}

# Extract registry and image information from arguments
parse_build_args() {
    local -A build_args=()
    
    # Parse docker build arguments to extract cache information
    while [[ $# -gt 0 ]]; do
        case $1 in
            --cache-from=*)
                build_args[cache_from]="${1#*=}"
                shift
                ;;
            --cache-to=*)
                build_args[cache_to]="${1#*=}"
                shift
                ;;
            -t|--tag)
                build_args[tag]="$2"
                shift 2
                ;;
            --tag=*)
                build_args[tag]="${1#*=}"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Set global variables for use in other functions
    CACHE_FROM="${build_args[cache_from]:-}"
    CACHE_TO="${build_args[cache_to]:-}"
    IMAGE_TAG="${build_args[tag]:-}"
    
    log_info "Parsed build arguments:"
    log_info "  Image tag: ${IMAGE_TAG:-none}"
    log_info "  Cache from: ${CACHE_FROM:-none}"
    log_info "  Cache to: ${CACHE_TO:-none}"
}

# Test GitHub Actions cache service availability
test_gha_cache_service() {
    log_info "Testing GitHub Actions cache service availability..."
    
    # The cache service errors we've seen indicate GHA service issues
    # We can't directly test the internal cache API, but we can check general connectivity
    if [ "${GITHUB_ACTIONS:-false}" = "true" ]; then
        log_info "Running in GitHub Actions environment"
        
        # Test connectivity to GitHub API as a proxy for service health
        if timeout 10 curl -sSf "https://api.github.com/zen" >/dev/null 2>&1; then
            log_success "GitHub API connectivity confirmed"
            return 0
        else
            log_warn "GitHub API connectivity issues detected"
            return 1
        fi
    else
        log_info "Not running in GitHub Actions, skipping GHA cache test"
        return 0
    fi
}

# Test GHCR registry connectivity and authentication
test_ghcr_connectivity() {
    log_info "Testing GHCR registry connectivity..."
    
    local registry="ghcr.io"
    
    # Test basic connectivity
    if ! timeout 10 curl -sSf "https://${registry}/v2/" >/dev/null 2>&1; then
        log_warn "GHCR registry not accessible via HTTPS"
        return 1
    fi
    
    log_success "GHCR registry is accessible"
    
    # Test authentication if we have credentials
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        log_info "Testing GHCR authentication..."
        
        # Try to authenticate with GHCR
        if echo "$GITHUB_TOKEN" | docker login "$registry" --username "$GITHUB_ACTOR" --password-stdin >/dev/null 2>&1; then
            log_success "GHCR authentication successful"
            return 0
        else
            log_warn "GHCR authentication failed"
            return 1
        fi
    else
        log_info "No GitHub token available for GHCR auth test"
        return 0
    fi
}

# Generate optimized cache configuration
generate_cache_config() {
    local service_name="$1"
    local fallback_mode="${2:-false}"
    
    log_info "Generating cache configuration for service: $service_name"
    log_info "Fallback mode: $fallback_mode"
    
    local registry="${REGISTRY:-ghcr.io}"
    local repo_name="${GITHUB_REPOSITORY:-nephoran/intent-operator}"
    local base_image="$registry/$repo_name/$service_name"
    
    # Cache configuration arrays
    local -a cache_from_opts=()
    local -a cache_to_opts=()
    
    if [ "$fallback_mode" = "false" ]; then
        # Standard cache configuration with multiple sources
        cache_from_opts=(
            "type=gha,scope=buildkit-$service_name"
            "type=registry,ref=$base_image:buildcache"
            "type=registry,ref=$base_image:cache"
        )
        
        cache_to_opts=(
            "type=gha,scope=buildkit-$service_name,mode=max"
            "type=registry,ref=$base_image:buildcache,mode=max,image-manifest=true,oci-mediatypes=true"
        )
    else
        # Fallback configuration - local cache only
        log_warn "Using fallback cache configuration (local only)"
        cache_from_opts=(
            "type=local,src=/tmp/.buildx-cache-$service_name"
        )
        
        cache_to_opts=(
            "type=local,dest=/tmp/.buildx-cache-$service_name,mode=max"
        )
        
        # Ensure local cache directory exists
        mkdir -p "/tmp/.buildx-cache-$service_name"
    fi
    
    # Generate cache arguments
    local cache_from_args=""
    local cache_to_args=""
    
    for cache_from in "${cache_from_opts[@]}"; do
        cache_from_args+="--cache-from=$cache_from "
    done
    
    for cache_to in "${cache_to_opts[@]}"; do
        cache_to_args+="--cache-to=$cache_to "
    done
    
    # Export for use by calling script
    export DOCKER_CACHE_FROM="$cache_from_args"
    export DOCKER_CACHE_TO="$cache_to_args"
    
    log_info "Generated cache configuration:"
    log_info "  Cache from: $cache_from_args"
    log_info "  Cache to: $cache_to_args"
}

# Pre-warm BuildKit builder
prewarm_builder() {
    log_info "Pre-warming BuildKit builder..."
    
    # Create a minimal build to warm up the builder
    local temp_dir
    temp_dir=$(mktemp -d)
    
    cat > "$temp_dir/Dockerfile" << 'EOF'
FROM alpine:3.21
RUN echo "Builder warmed up at $(date)" > /tmp/warmup
EOF
    
    if docker buildx build --platform linux/amd64 --load -t warmup:latest "$temp_dir" >/dev/null 2>&1; then
        log_success "BuildKit builder warmed up successfully"
        docker rmi warmup:latest >/dev/null 2>&1 || true
    else
        log_warn "Builder warmup failed (non-fatal)"
    fi
    
    rm -rf "$temp_dir"
}

# Cleanup stale cache entries
cleanup_stale_cache() {
    log_info "Cleaning up stale cache entries..."
    
    # Clean up old local cache directories (older than 7 days)
    find /tmp -maxdepth 1 -name ".buildx-cache-*" -type d -mtime +7 -exec rm -rf {} \; 2>/dev/null || true
    
    # Prune Docker build cache
    docker builder prune --filter "until=24h" --force >/dev/null 2>&1 || true
    
    log_success "Cache cleanup completed"
}

# Execute Docker build with resilient cache handling
execute_build_with_cache_fallback() {
    local service_name="$1"
    shift
    local build_args=("$@")
    
    log_info "Executing Docker build for service: $service_name"
    
    # Pre-flight checks
    local use_fallback=false
    
    if ! test_gha_cache_service; then
        log_warn "GHA cache service issues detected"
        use_fallback=true
    fi
    
    if ! test_ghcr_connectivity; then
        log_warn "GHCR connectivity issues detected"
        use_fallback=true
    fi
    
    # Generate cache configuration
    generate_cache_config "$service_name" "$use_fallback"
    
    # Pre-warm builder
    prewarm_builder
    
    # Construct final build command
    local -a final_build_args=()
    
    # Add cache arguments
    if [ -n "$DOCKER_CACHE_FROM" ]; then
        read -ra cache_from_array <<< "$DOCKER_CACHE_FROM"
        for cache_arg in "${cache_from_array[@]}"; do
            if [ -n "$cache_arg" ]; then
                final_build_args+=("$cache_arg")
            fi
        done
    fi
    
    if [ -n "$DOCKER_CACHE_TO" ]; then
        read -ra cache_to_array <<< "$DOCKER_CACHE_TO"
        for cache_arg in "${cache_to_array[@]}"; do
            if [ -n "$cache_arg" ]; then
                final_build_args+=("$cache_arg")
            fi
        done
    fi
    
    # Add original build arguments
    final_build_args+=("${build_args[@]}")
    
    log_info "Final build command: docker buildx build ${final_build_args[*]}"
    
    # Execute build with timeout and error handling
    local max_attempts=3
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        log_info "Build attempt $attempt/$max_attempts"
        
        if timeout 1800 docker buildx build "${final_build_args[@]}"; then
            log_success "Docker build completed successfully on attempt $attempt"
            return 0
        fi
        
        local exit_code=$?
        log_warn "Build attempt $attempt failed with exit code $exit_code"
        
        if [ $attempt -eq $max_attempts ]; then
            log_error "Build failed after $max_attempts attempts"
            return $exit_code
        fi
        
        # For subsequent attempts, use fallback cache strategy
        if [ $attempt -eq 1 ]; then
            log_info "Switching to fallback cache strategy for retry"
            generate_cache_config "$service_name" "true"
            
            # Update cache arguments for retry
            final_build_args=()
            if [ -n "$DOCKER_CACHE_FROM" ]; then
                read -ra cache_from_array <<< "$DOCKER_CACHE_FROM"
                for cache_arg in "${cache_from_array[@]}"; do
                    if [ -n "$cache_arg" ]; then
                        final_build_args+=("$cache_arg")
                    fi
                done
            fi
            
            if [ -n "$DOCKER_CACHE_TO" ]; then
                read -ra cache_to_array <<< "$DOCKER_CACHE_TO"
                for cache_arg in "${cache_to_array[@]}"; do
                    if [ -n "$cache_arg" ]; then
                        final_build_args+=("$cache_arg")
                    fi
                done
            fi
            
            final_build_args+=("${build_args[@]}")
        fi
        
        # Wait before retry
        local delay=$((attempt * 10))
        log_info "Waiting ${delay}s before retry..."
        sleep $delay
        
        ((attempt++))
    done
    
    return 1
}

# Display help
show_help() {
    cat << EOF
Docker Build Cache Fix Script

This script addresses cache import failures in GitHub Actions by implementing
fallback strategies and resilient build processes.

Usage:
    $0 [OPTIONS] SERVICE_NAME -- [DOCKER_BUILD_ARGS...]

Options:
    -h, --help      Show this help message
    --cleanup-only  Only perform cache cleanup, skip build
    --test-only     Only test connectivity, skip build
    --force-fallback Use fallback cache strategy immediately

Examples:
    # Build with automatic cache fallback detection
    $0 llm-processor -- --platform linux/amd64 --push -t ghcr.io/nephoran/llm-processor:latest .
    
    # Test connectivity only
    $0 --test-only
    
    # Cleanup stale cache only
    $0 --cleanup-only

Environment Variables:
    REGISTRY         Container registry (default: ghcr.io)
    GITHUB_TOKEN     GitHub token for GHCR authentication
    GITHUB_ACTOR     GitHub username for GHCR authentication
    GITHUB_REPOSITORY GitHub repository name

EOF
}

# Main function
main() {
    local service_name=""
    local cleanup_only=false
    local test_only=false
    local force_fallback=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            --cleanup-only)
                cleanup_only=true
                shift
                ;;
            --test-only)
                test_only=true
                shift
                ;;
            --force-fallback)
                force_fallback=true
                shift
                ;;
            --)
                shift
                break
                ;;
            *)
                if [ -z "$service_name" ]; then
                    service_name="$1"
                    shift
                else
                    log_error "Unexpected argument: $1"
                    show_help
                    exit 1
                fi
                ;;
        esac
    done
    
    log_info "Docker Build Cache Fix starting..."
    
    # Cleanup mode
    if [ "$cleanup_only" = true ]; then
        cleanup_stale_cache
        log_success "Cleanup completed"
        exit 0
    fi
    
    # Test mode
    if [ "$test_only" = true ]; then
        test_gha_cache_service
        test_ghcr_connectivity
        log_success "Connectivity tests completed"
        exit 0
    fi
    
    # Validate service name
    if [ -z "$service_name" ]; then
        log_error "Service name is required"
        show_help
        exit 1
    fi
    
    # Remaining arguments are docker build arguments
    local -a docker_build_args=("$@")
    
    if [ ${#docker_build_args[@]} -eq 0 ]; then
        log_error "Docker build arguments are required"
        show_help
        exit 1
    fi
    
    log_info "Service: $service_name"
    log_info "Build args: ${docker_build_args[*]}"
    
    # Execute build with cache fallback
    if [ "$force_fallback" = true ]; then
        generate_cache_config "$service_name" "true"
        execute_build_with_cache_fallback "$service_name" "${docker_build_args[@]}"
    else
        execute_build_with_cache_fallback "$service_name" "${docker_build_args[@]}"
    fi
    
    # Cleanup after successful build
    cleanup_stale_cache
    
    log_success "Docker build cache fix completed successfully!"
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi