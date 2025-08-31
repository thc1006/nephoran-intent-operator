#!/bin/bash

# =============================================================================
# CI Cache Optimization Script
# =============================================================================
# This script optimizes Go build cache and module cache for faster CI runs
# Usage: ./optimize-ci-cache.sh [--cleanup] [--stats]
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Configuration
GO_CACHE_DIR="${HOME}/.cache/go-build"
GO_MOD_CACHE_DIR="${HOME}/go/pkg/mod"
MAX_CACHE_SIZE_MB=1024  # 1GB max cache size
CLEANUP_OLDER_THAN_DAYS=7

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}" >&2
}

warn() {
    echo -e "${YELLOW}[WARN] $1${NC}"
}

success() {
    echo -e "${GREEN}[SUCCESS] $1${NC}"
}

# Get cache size in MB
get_cache_size() {
    local dir=$1
    if [[ -d "$dir" ]]; then
        du -sm "$dir" 2>/dev/null | cut -f1 || echo "0"
    else
        echo "0"
    fi
}

# Get cache statistics
show_cache_stats() {
    log "Cache Statistics:"
    echo "=================="
    
    local go_cache_size=$(get_cache_size "$GO_CACHE_DIR")
    local mod_cache_size=$(get_cache_size "$GO_MOD_CACHE_DIR")
    local total_size=$((go_cache_size + mod_cache_size))
    
    echo "Go Build Cache: ${go_cache_size}MB (${GO_CACHE_DIR})"
    echo "Go Module Cache: ${mod_cache_size}MB (${GO_MOD_CACHE_DIR})"
    echo "Total Cache Size: ${total_size}MB"
    echo ""
    
    if [[ $total_size -gt $MAX_CACHE_SIZE_MB ]]; then
        warn "Cache size (${total_size}MB) exceeds recommended maximum (${MAX_CACHE_SIZE_MB}MB)"
    else
        success "Cache size is within recommended limits"
    fi
}

# Clean old cache entries
cleanup_cache() {
    log "Cleaning up cache entries older than ${CLEANUP_OLDER_THAN_DAYS} days..."
    
    # Clean Go build cache
    if [[ -d "$GO_CACHE_DIR" ]]; then
        log "Cleaning Go build cache..."
        find "$GO_CACHE_DIR" -type f -mtime +${CLEANUP_OLDER_THAN_DAYS} -delete 2>/dev/null || true
        find "$GO_CACHE_DIR" -type d -empty -delete 2>/dev/null || true
    fi
    
    # Clean module cache (be more careful with this)
    if [[ -d "$GO_MOD_CACHE_DIR" ]]; then
        log "Cleaning old module cache entries..."
        # Only remove cache directories, not the modules themselves
        find "$GO_MOD_CACHE_DIR/cache" -type f -mtime +${CLEANUP_OLDER_THAN_DAYS} -delete 2>/dev/null || true
        find "$GO_MOD_CACHE_DIR/cache" -type d -empty -delete 2>/dev/null || true
    fi
    
    success "Cache cleanup completed"
}

# Optimize cache structure
optimize_cache() {
    log "Optimizing cache structure..."
    
    # Ensure proper permissions
    if [[ -d "$GO_CACHE_DIR" ]]; then
        chmod -R u+w "$GO_CACHE_DIR" 2>/dev/null || true
    fi
    
    if [[ -d "$GO_MOD_CACHE_DIR" ]]; then
        chmod -R u+w "$GO_MOD_CACHE_DIR" 2>/dev/null || true
    fi
    
    # Pre-compile standard library for faster builds
    log "Pre-compiling standard library..."
    go install -a -installsuffix cgo std 2>/dev/null || warn "Failed to pre-compile standard library"
    
    success "Cache optimization completed"
}

# Warm up cache with dependencies
warmup_cache() {
    log "Warming up cache with project dependencies..."
    
    cd "$ROOT_DIR"
    
    # Download all dependencies
    go mod download -x 2>/dev/null || warn "Some dependencies failed to download"
    
    # Pre-build common packages
    log "Pre-building common packages..."
    go build -i std 2>/dev/null || warn "Failed to pre-build standard packages"
    
    # Build tools we commonly use
    local tools=(
        "sigs.k8s.io/controller-tools/cmd/controller-gen@latest"
        "sigs.k8s.io/controller-runtime/tools/setup-envtest@latest"
        "github.com/onsi/ginkgo/v2/ginkgo@latest"
    )
    
    for tool in "${tools[@]}"; do
        log "Installing tool: $tool"
        go install "$tool" 2>/dev/null || warn "Failed to install $tool"
    done
    
    success "Cache warmup completed"
}

# Main execution
main() {
    local cleanup_flag=false
    local stats_flag=false
    local warmup_flag=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --cleanup)
                cleanup_flag=true
                shift
                ;;
            --stats)
                stats_flag=true
                shift
                ;;
            --warmup)
                warmup_flag=true
                shift
                ;;
            --help|-h)
                echo "Usage: $0 [--cleanup] [--stats] [--warmup]"
                echo ""
                echo "Options:"
                echo "  --cleanup   Clean old cache entries"
                echo "  --stats     Show cache statistics"
                echo "  --warmup    Warm up cache with dependencies"
                echo "  --help      Show this help message"
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    log "Starting CI cache optimization..."
    
    # Show initial stats
    show_cache_stats
    
    # Perform cleanup if requested
    if [[ "$cleanup_flag" == true ]]; then
        cleanup_cache
        echo ""
        show_cache_stats
    fi
    
    # Optimize cache structure
    optimize_cache
    
    # Warm up cache if requested
    if [[ "$warmup_flag" == true ]]; then
        warmup_cache
    fi
    
    # Show final stats if requested
    if [[ "$stats_flag" == true ]]; then
        echo ""
        show_cache_stats
    fi
    
    success "CI cache optimization completed successfully!"
    
    # Provide recommendations
    echo ""
    log "Recommendations for faster CI:"
    echo "- Use cache warming in CI: $0 --warmup"
    echo "- Regular cache cleanup: $0 --cleanup"
    echo "- Monitor cache size: $0 --stats"
    echo "- Consider using BuildKit caching for Docker builds"
    echo "- Use Go build tags to exclude unnecessary code: -tags='fast_build,no_swagger,no_e2e'"
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi