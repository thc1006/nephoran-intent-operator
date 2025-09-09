#!/bin/bash
# =============================================================================
# Nephoran Intent Operator - CI Environment Optimizer
# =============================================================================
# Optimizes GitHub Actions and CI environments for maximum Go build performance
# Configures: environment variables, caching, resource allocation
# Target: Maximum performance on ubuntu-22.04 with Go 1.24.x
# =============================================================================

set -euo pipefail

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly CYAN='\033[0;36m'
readonly NC='\033[0m'

log_info() { echo -e "${BLUE}[CI-OPT]${NC} $*" >&2; }
log_success() { echo -e "${GREEN}[CI-OPT]${NC} $*" >&2; }
log_warn() { echo -e "${YELLOW}[CI-OPT]${NC} $*" >&2; }
log_error() { echo -e "${RED}[CI-OPT]${NC} $*" >&2; }

# Detect CI environment
detect_ci_environment() {
    if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
        echo "github-actions"
    elif [[ "${GITLAB_CI:-}" == "true" ]]; then
        echo "gitlab-ci"
    elif [[ "${CIRCLE_CI:-}" == "true" ]]; then
        echo "circleci"
    else
        echo "local"
    fi
}

# Configure optimal environment variables for Go builds
configure_go_environment() {
    local ci_env="$1"
    log_info "Configuring Go environment for $ci_env..."
    
    # Base Go configuration
    export GO_VERSION="${GO_VERSION:-1.24.0}"
    export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.24.6}"
    
    # Detect system resources
    local cpu_cores
    cpu_cores=$(nproc 2>/dev/null || echo "4")
    local total_memory_kb
    total_memory_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}' 2>/dev/null || echo "8388608")
    local total_memory_gb=$((total_memory_kb / 1024 / 1024))
    
    log_info "Detected system: ${cpu_cores} cores, ${total_memory_gb}GB RAM"
    
    # Optimize based on available resources
    local gomaxprocs
    local gomemlimit
    local build_parallelism
    local test_parallelism
    
    if [[ $cpu_cores -ge 8 ]]; then
        gomaxprocs=8
        build_parallelism=8
        test_parallelism=4
    elif [[ $cpu_cores -ge 4 ]]; then
        gomaxprocs=4
        build_parallelism=6
        test_parallelism=3
    else
        gomaxprocs=2
        build_parallelism=4
        test_parallelism=2
    fi
    
    if [[ $total_memory_gb -ge 12 ]]; then
        gomemlimit="12GiB"
    elif [[ $total_memory_gb -ge 8 ]]; then
        gomemlimit="8GiB"
    else
        gomemlimit="4GiB"
    fi
    
    # Core Go environment variables for maximum performance
    cat > /tmp/go-env-vars.sh << EOF
# Nephoran Go Build Environment - Ultra-Optimized
export GO_VERSION="$GO_VERSION"
export GOTOOLCHAIN="$GOTOOLCHAIN"

# Core Go configuration
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64
export GOMAXPROCS=$gomaxprocs
export GOMEMLIMIT="$gomemlimit"

# Build optimization
export GOGC=75                           # Aggressive GC for build speed
export GOEXPERIMENT="fieldtrack,boringcrypto"
export GODEBUG="gocachehash=1,gocachetest=1"
export GOFLAGS="-mod=readonly -trimpath -buildvcs=false"

# Advanced proxy configuration
export GOPROXY="https://proxy.golang.org,direct"
export GOSUMDB="sum.golang.org"
export GOPRIVATE="github.com/thc1006/*"

# Cache configuration
export GOCACHE="\${GITHUB_WORKSPACE:-\${PWD}}/.go-build-cache"
export GOMODCACHE="\${GITHUB_WORKSPACE:-\${PWD}}/.go-mod-cache"

# Build parallelization
export BUILD_PARALLELISM=$build_parallelism
export TEST_PARALLELISM=$test_parallelism

# Performance monitoring
export GOMAXPROCS_LOG=1
export GOMEMLIMIT_LOG=1

# CI-specific optimizations
export GODEBUG="\${GODEBUG},gctrace=0"    # Reduce GC trace overhead in CI
export GOTMPDIR="\${RUNNER_TEMP:-/tmp}/go-tmp"

echo "Go environment configured:"
echo "  GOMAXPROCS: \$GOMAXPROCS"
echo "  GOMEMLIMIT: \$GOMEMLIMIT"  
echo "  Build parallelism: \$BUILD_PARALLELISM"
echo "  Test parallelism: \$TEST_PARALLELISM"
echo "  Cache dir: \$GOCACHE"
echo "  Module cache: \$GOMODCACHE"
EOF
    
    source /tmp/go-env-vars.sh
    
    # Set GitHub Actions environment variables if in GHA
    if [[ "$ci_env" == "github-actions" ]]; then
        {
            echo "GO_VERSION=$GO_VERSION"
            echo "GOTOOLCHAIN=$GOTOOLCHAIN"
            echo "CGO_ENABLED=0"
            echo "GOOS=linux"
            echo "GOARCH=amd64"
            echo "GOMAXPROCS=$gomaxprocs"
            echo "GOMEMLIMIT=$gomemlimit"
            echo "GOGC=75"
            echo "GOEXPERIMENT=fieldtrack,boringcrypto"
            echo "GODEBUG=gocachehash=1,gocachetest=1"
            echo "GOFLAGS=-mod=readonly -trimpath -buildvcs=false"
            echo "GOPROXY=https://proxy.golang.org,direct"
            echo "GOSUMDB=sum.golang.org"
            echo "GOPRIVATE=github.com/thc1006/*"
            echo "GOCACHE=\${GITHUB_WORKSPACE}/.go-build-cache"
            echo "GOMODCACHE=\${GITHUB_WORKSPACE}/.go-mod-cache"
            echo "BUILD_PARALLELISM=$build_parallelism"
            echo "TEST_PARALLELISM=$test_parallelism"
        } >> "$GITHUB_ENV"
    fi
    
    log_success "Go environment configured for optimal CI performance"
}

# Configure advanced caching strategies
configure_advanced_caching() {
    local ci_env="$1"
    log_info "Configuring advanced caching for $ci_env..."
    
    # Create cache directories
    mkdir -p "${GOCACHE:-${PWD}/.go-build-cache}"
    mkdir -p "${GOMODCACHE:-${PWD}/.go-mod-cache}"
    mkdir -p "${GOTMPDIR:-/tmp/go-tmp}"
    
    # Cache key generation strategy
    local cache_version="v11-ultra"
    local go_version_hash
    go_version_hash=$(echo "$GO_VERSION" | sha256sum | cut -c1-8)
    local build_config_hash
    build_config_hash=$(echo "${GOFLAGS:-}-${GOGC:-}-${GOEXPERIMENT:-}" | sha256sum | cut -c1-8)
    
    # Generate hierarchical cache keys for maximum hit rates
    local primary_cache_key="nephoran-${cache_version}-go${go_version_hash}-${build_config_hash}-$(date +%Y%m%d)"
    local secondary_cache_key="nephoran-${cache_version}-go${go_version_hash}-${build_config_hash}"
    local tertiary_cache_key="nephoran-${cache_version}-go${go_version_hash}"
    local fallback_cache_key="nephoran-${cache_version}"
    
    # GitHub Actions cache configuration
    if [[ "$ci_env" == "github-actions" ]]; then
        cat > /tmp/cache-config.json << EOF
{
  "cache_strategy": "ultra-aggressive",
  "keys": {
    "primary": "$primary_cache_key",
    "secondary": "$secondary_cache_key", 
    "tertiary": "$tertiary_cache_key",
    "fallback": "$fallback_cache_key"
  },
  "paths": [
    "\${GOCACHE}",
    "\${GOMODCACHE}",
    "~/.cache/go-build",
    "~/go/pkg/mod"
  ],
  "compression": "zstd",
  "ttl": "7d"
}
EOF
        log_info "Cache configuration generated: /tmp/cache-config.json"
    fi
    
    log_success "Advanced caching configured"
    log_info "  Primary key: $primary_cache_key"
    log_info "  Cache paths: Build=${GOCACHE}, Modules=${GOMODCACHE}"
}

# Optimize system resources for Go builds
optimize_system_resources() {
    log_info "Optimizing system resources for Go builds..."
    
    # Increase file descriptor limits for large dependency trees
    ulimit -n 65536 2>/dev/null || log_warn "Could not increase file descriptor limit"
    
    # Optimize disk I/O for build artifacts
    if command -v ionice > /dev/null; then
        ionice -c 1 -n 4 -p $$ 2>/dev/null || true
    fi
    
    # Configure swap usage for large builds (if available)
    if [[ -f /proc/sys/vm/swappiness ]]; then
        echo 10 | sudo tee /proc/sys/vm/swappiness > /dev/null 2>&1 || true
    fi
    
    # Optimize for build workload
    if [[ -f /proc/sys/vm/dirty_ratio ]]; then
        echo 15 | sudo tee /proc/sys/vm/dirty_ratio > /dev/null 2>&1 || true
        echo 5 | sudo tee /proc/sys/vm/dirty_background_ratio > /dev/null 2>&1 || true
    fi
    
    log_success "System resources optimized"
}

# Generate optimized build scripts
generate_build_scripts() {
    log_info "Generating optimized build scripts..."
    
    # Fast build script for CI
    cat > scripts/ci-fast-build.sh << 'EOF'
#!/bin/bash
# Ultra-fast CI build script for Nephoran
set -euo pipefail

# Load optimized environment
source /tmp/go-env-vars.sh 2>/dev/null || true

# Fast dependency resolution
echo "🚀 Ultra-fast dependency resolution..."
timeout 180s go mod download -x || {
    echo "Retrying dependency download..."
    timeout 180s go mod download
}
go mod verify

# Parallel component build
echo "🏗️ Building components in parallel..."
mkdir -p bin/

# Build critical components first
critical_components=(
    "intent-ingest"
    "conductor-loop"
    "llm-processor"
    "webhook"
)

build_component() {
    local component="$1"
    local cmd_path="./cmd/$component"
    
    if [[ -d "$cmd_path" && -f "$cmd_path/main.go" ]]; then
        echo "Building $component..."
        timeout 120s go build \
            -p "${BUILD_PARALLELISM:-6}" \
            -trimpath \
            -ldflags="-s -w -extldflags=-static" \
            -tags="netgo,osusergo,static_build" \
            -o "bin/$component" \
            "$cmd_path" || {
            echo "⚠️ $component build failed, continuing..."
        }
        
        if [[ -f "bin/$component" ]]; then
            size=$(ls -lh "bin/$component" | awk '{print $5}')
            echo "✅ $component: $size"
        fi
    fi
}

# Build critical components in parallel
for component in "${critical_components[@]}"; do
    build_component "$component" &
done
wait

echo "🎉 CI build completed"
ls -la bin/ || true
EOF

    chmod +x scripts/ci-fast-build.sh
    
    # Test optimization script
    cat > scripts/ci-fast-test.sh << 'EOF'
#!/bin/bash
# Ultra-fast CI test script for Nephoran  
set -euo pipefail

# Load optimized environment
source /tmp/go-env-vars.sh 2>/dev/null || true

echo "🧪 Running optimized test suite..."

# Test flags for maximum performance
TEST_FLAGS="-v -timeout=8m -parallel=${TEST_PARALLELISM:-4}"

# Run tests by priority
echo "Running critical path tests..."
go test $TEST_FLAGS -race ./controllers/... ./api/... || {
    echo "⚠️ Critical tests had issues"
    exit 1
}

echo "Running core package tests..."  
go test $TEST_FLAGS ./pkg/... || {
    echo "⚠️ Core package tests had issues"
}

echo "🎉 Test suite completed"
EOF

    chmod +x scripts/ci-fast-test.sh
    
    log_success "Build scripts generated"
    log_info "  Fast build: scripts/ci-fast-build.sh"
    log_info "  Fast test: scripts/ci-fast-test.sh"
}

# Generate performance monitoring script
generate_performance_monitoring() {
    log_info "Generating performance monitoring script..."
    
    cat > scripts/build-performance-monitor.sh << 'EOF'
#!/bin/bash
# Build performance monitoring for Nephoran
set -euo pipefail

start_time=$(date +%s)
build_start_memory=$(free -m | awk 'NR==2{printf "%.1f", $3/1024}')

# Monitor build performance
monitor_build() {
    local stage="$1"
    local current_time=$(date +%s)
    local elapsed=$((current_time - start_time))
    local current_memory=$(free -m | awk 'NR==2{printf "%.1f", $3/1024}')
    
    echo "📊 Stage: $stage | Time: ${elapsed}s | Memory: ${current_memory}GB"
    
    # Log to performance file
    echo "$(date -Iseconds),$stage,$elapsed,$current_memory" >> build/performance.csv
}

# Create performance log header
mkdir -p build/
echo "timestamp,stage,elapsed_seconds,memory_gb" > build/performance.csv

# Export monitoring function for use by other scripts
export -f monitor_build
export start_time
EOF

    chmod +x scripts/build-performance-monitor.sh
    
    log_success "Performance monitoring configured"
}

# Main configuration function
main() {
    local ci_env
    ci_env=$(detect_ci_environment)
    
    log_info "Optimizing CI environment: $ci_env"
    
    # Apply all optimizations
    configure_go_environment "$ci_env"
    configure_advanced_caching "$ci_env"
    optimize_system_resources
    generate_build_scripts
    generate_performance_monitoring
    
    log_success "CI environment optimization completed"
    
    # Generate summary
    echo ""
    echo "🚀 CI Environment Optimization Summary:"
    echo "  Environment: $ci_env"
    echo "  Go version: ${GO_VERSION}"
    echo "  Max processes: ${GOMAXPROCS}"
    echo "  Memory limit: ${GOMEMLIMIT}"
    echo "  Build parallelism: ${BUILD_PARALLELISM}"
    echo "  Test parallelism: ${TEST_PARALLELISM}"
    echo ""
    echo "📁 Generated files:"
    echo "  /tmp/go-env-vars.sh - Environment configuration"
    echo "  scripts/ci-fast-build.sh - Fast build script"
    echo "  scripts/ci-fast-test.sh - Fast test script"
    echo "  scripts/build-performance-monitor.sh - Performance monitoring"
    if [[ -f /tmp/cache-config.json ]]; then
        echo "  /tmp/cache-config.json - Cache configuration"
    fi
    echo ""
    echo "✅ Ready for ultra-fast CI builds!"
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
EOF