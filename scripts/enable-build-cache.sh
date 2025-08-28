#!/bin/bash
# Bash script to enable ultra-fast build caching for Linux/Mac development
# This script sets up persistent caching for Go builds and Docker layers

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Cache directories
GO_CACHE="${HOME}/.cache/go-build"
GO_MOD_CACHE="${HOME}/.cache/go-mod"
DOCKER_CACHE="${HOME}/.cache/docker-buildx"
CCACHE_DIR="${HOME}/.cache/ccache"

# Parse arguments
CLEAN=false
STATUS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --clean)
            CLEAN=true
            shift
            ;;
        --status)
            STATUS=true
            shift
            ;;
        --help)
            echo "Usage: $0 [--clean|--status|--help]"
            echo "  --clean   Clean all build caches"
            echo "  --status  Show cache status"
            echo "  --help    Show this help message"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

show_status() {
    echo -e "${CYAN}Build Cache Status${NC}"
    echo -e "${CYAN}==================${NC}"
    
    # Check Go cache
    if [ -d "$GO_CACHE" ]; then
        size=$(du -sh "$GO_CACHE" 2>/dev/null | cut -f1)
        echo -e "${GREEN}Go Build Cache: $size${NC}"
    else
        echo -e "${YELLOW}Go Build Cache: Not initialized${NC}"
    fi
    
    # Check Go module cache
    if [ -d "$GO_MOD_CACHE" ]; then
        size=$(du -sh "$GO_MOD_CACHE" 2>/dev/null | cut -f1)
        echo -e "${GREEN}Go Module Cache: $size${NC}"
    else
        echo -e "${YELLOW}Go Module Cache: Not initialized${NC}"
    fi
    
    # Check Docker cache
    if [ -d "$DOCKER_CACHE" ]; then
        size=$(du -sh "$DOCKER_CACHE" 2>/dev/null | cut -f1)
        echo -e "${GREEN}Docker BuildX Cache: $size${NC}"
    else
        echo -e "${YELLOW}Docker BuildX Cache: Not initialized${NC}"
    fi
    
    # Check ccache
    if [ -d "$CCACHE_DIR" ]; then
        size=$(du -sh "$CCACHE_DIR" 2>/dev/null | cut -f1)
        echo -e "${GREEN}CCache: $size${NC}"
    else
        echo -e "${YELLOW}CCache: Not initialized${NC}"
    fi
    
    # Show environment variables
    echo -e "\n${CYAN}Environment Variables:${NC}"
    echo -e "GOCACHE: ${GOCACHE:-not set}"
    echo -e "GOMODCACHE: ${GOMODCACHE:-not set}"
    echo -e "DOCKER_BUILDKIT: ${DOCKER_BUILDKIT:-not set}"
    echo -e "GOMAXPROCS: ${GOMAXPROCS:-not set}"
}

clean_cache() {
    echo -e "${YELLOW}Cleaning build caches...${NC}"
    
    if [ -d "$GO_CACHE" ]; then
        rm -rf "$GO_CACHE"
        echo -e "${GREEN}Cleaned Go build cache${NC}"
    fi
    
    if [ -d "$GO_MOD_CACHE" ]; then
        rm -rf "$GO_MOD_CACHE"
        echo -e "${GREEN}Cleaned Go module cache${NC}"
    fi
    
    if [ -d "$DOCKER_CACHE" ]; then
        rm -rf "$DOCKER_CACHE"
        echo -e "${GREEN}Cleaned Docker buildx cache${NC}"
    fi
    
    if [ -d "$CCACHE_DIR" ]; then
        rm -rf "$CCACHE_DIR"
        echo -e "${GREEN}Cleaned ccache${NC}"
    fi
    
    # Clean Docker system
    docker system prune -f --volumes 2>/dev/null || true
    echo -e "${GREEN}Cleaned Docker system${NC}"
}

enable_cache() {
    echo -e "${CYAN}Enabling ultra-fast build caching...${NC}"
    
    # Create cache directories
    mkdir -p "$GO_CACHE"
    mkdir -p "$GO_MOD_CACHE"
    mkdir -p "$DOCKER_CACHE"
    mkdir -p "$CCACHE_DIR"
    
    # Detect shell and config file
    SHELL_CONFIG=""
    if [ -n "$ZSH_VERSION" ]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        SHELL_CONFIG="$HOME/.bashrc"
    else
        SHELL_CONFIG="$HOME/.profile"
    fi
    
    # Add environment variables to shell config
    {
        echo ""
        echo "# Ultra-fast build caching configuration"
        echo "export GOCACHE=\"$GO_CACHE\""
        echo "export GOMODCACHE=\"$GO_MOD_CACHE\""
        echo "export GOMAXPROCS=16"
        echo "export GOGC=200"
        echo "export GOMEMLIMIT=8GiB"
        echo "export DOCKER_BUILDKIT=1"
        echo "export BUILDKIT_PROGRESS=plain"
        echo "export COMPOSE_DOCKER_CLI_BUILD=1"
        echo "export CCACHE_DIR=\"$CCACHE_DIR\""
        echo "export CCACHE_MAXSIZE=5G"
    } >> "$SHELL_CONFIG"
    
    # Export for current session
    export GOCACHE="$GO_CACHE"
    export GOMODCACHE="$GO_MOD_CACHE"
    export GOMAXPROCS=16
    export GOGC=200
    export GOMEMLIMIT=8GiB
    export DOCKER_BUILDKIT=1
    export BUILDKIT_PROGRESS=plain
    export COMPOSE_DOCKER_CLI_BUILD=1
    export CCACHE_DIR="$CCACHE_DIR"
    export CCACHE_MAXSIZE=5G
    
    echo -e "${GREEN}Build caching enabled!${NC}"
    echo ""
    echo -e "${GREEN}Performance optimizations applied:${NC}"
    echo -e "  - Go build cache: $GO_CACHE"
    echo -e "  - Go module cache: $GO_MOD_CACHE"
    echo -e "  - Docker BuildKit: Enabled"
    echo -e "  - GOMAXPROCS: 16"
    echo -e "  - GOGC: 200 (less frequent GC)"
    echo -e "  - GOMEMLIMIT: 8GiB"
    echo ""
    echo -e "${YELLOW}Run 'source $SHELL_CONFIG' or restart your terminal for changes to take effect.${NC}"
    
    # Pre-warm caches
    echo -e "${CYAN}Pre-warming caches...${NC}"
    
    # Download Go standard library
    go build -v std 2>/dev/null || true
    
    # Pre-download common dependencies
    if [ -f "go.mod" ]; then
        go mod download 2>/dev/null || true
    fi
    
    echo -e "${GREEN}Caches warmed up!${NC}"
}

# Main logic
if [ "$STATUS" = true ]; then
    show_status
elif [ "$CLEAN" = true ]; then
    clean_cache
else
    enable_cache
    show_status
fi

echo ""
echo -e "${CYAN}Quick Commands:${NC}"
echo "  make ultra-fast     # Run ultra-fast build (<2 min)"
echo "  make build-ultra    # Build all binaries in parallel"
echo "  make test-ultra     # Run tests with max parallelization"
echo "  make docker-ultra   # Build Docker images ultra-fast"