#!/bin/bash
# ==============================================================================
# Build Performance Optimization Script for Nephoran Intent Operator
# Diagnoses and fixes common Go build performance issues
# ==============================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ==============================================================================
# Helper Functions
# ==============================================================================

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

# ==============================================================================
# System Analysis
# ==============================================================================

analyze_system() {
    log_info "Analyzing system configuration..."
    
    echo "System Information:"
    echo "  OS: $(uname -s)"
    echo "  Architecture: $(uname -m)"
    echo "  CPUs: $(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 'unknown')"
    
    if [ -f /proc/meminfo ]; then
        echo "  Memory: $(awk '/MemTotal/ {printf "%.1f GB", $2/1024/1024}' /proc/meminfo)"
        echo "  Available: $(awk '/MemAvailable/ {printf "%.1f GB", $2/1024/1024}' /proc/meminfo)"
    fi
    
    echo ""
    echo "Go Environment:"
    go version
    echo "  GOPATH: $(go env GOPATH)"
    echo "  GOCACHE: $(go env GOCACHE)"
    echo "  GOMODCACHE: $(go env GOMODCACHE)"
    echo "  CGO_ENABLED: $(go env CGO_ENABLED)"
    echo ""
}

# ==============================================================================
# Dependency Analysis
# ==============================================================================

analyze_dependencies() {
    log_info "Analyzing Go module dependencies..."
    
    # Count dependencies
    total_deps=$(go list -m all 2>/dev/null | wc -l)
    direct_deps=$(go list -m all 2>/dev/null | grep -v "// indirect" | wc -l)
    indirect_deps=$(go list -m all 2>/dev/null | grep "// indirect" | wc -l)
    
    echo "Dependency Statistics:"
    echo "  Total dependencies: $total_deps"
    echo "  Direct dependencies: $direct_deps"
    echo "  Indirect dependencies: $indirect_deps"
    
    # Find heavy dependencies
    log_info "Identifying heavy dependencies..."
    echo "Top 10 largest dependencies by import count:"
    go list -deps ./... 2>/dev/null | \
        grep -E "^[^/]+" | \
        sort | uniq -c | sort -rn | head -10
    
    echo ""
}

# ==============================================================================
# Build Performance Testing
# ==============================================================================

test_build_performance() {
    log_info "Testing build performance..."
    
    # Create temporary directory for test builds
    BUILD_TEST_DIR=$(mktemp -d)
    trap "rm -rf $BUILD_TEST_DIR" EXIT
    
    # Test 1: Simple package build
    log_info "Test 1: Building simple package..."
    start_time=$(date +%s)
    timeout 30 go build -o $BUILD_TEST_DIR/test1 ./cmd/intent-ingest 2>/dev/null || {
        log_error "Simple build timed out after 30 seconds"
        return 1
    }
    end_time=$(date +%s)
    simple_time=$((end_time - start_time))
    log_success "Simple build completed in ${simple_time}s"
    
    # Test 2: Parallel build
    log_info "Test 2: Testing parallel build..."
    start_time=$(date +%s)
    timeout 60 go build -p=8 -o $BUILD_TEST_DIR/test2 ./cmd/intent-ingest 2>/dev/null || {
        log_error "Parallel build timed out after 60 seconds"
        return 1
    }
    end_time=$(date +%s)
    parallel_time=$((end_time - start_time))
    log_success "Parallel build completed in ${parallel_time}s"
    
    # Test 3: Optimized build
    log_info "Test 3: Testing optimized build..."
    start_time=$(date +%s)
    timeout 60 go build \
        -p=8 \
        -mod=readonly \
        -trimpath \
        -ldflags="-s -w" \
        -gcflags="-l=4" \
        -tags="fast_build" \
        -o $BUILD_TEST_DIR/test3 \
        ./cmd/intent-ingest 2>/dev/null || {
        log_error "Optimized build timed out after 60 seconds"
        return 1
    }
    end_time=$(date +%s)
    optimized_time=$((end_time - start_time))
    log_success "Optimized build completed in ${optimized_time}s"
    
    echo ""
    echo "Build Performance Summary:"
    echo "  Simple build: ${simple_time}s"
    echo "  Parallel build: ${parallel_time}s"
    echo "  Optimized build: ${optimized_time}s"
    
    # Calculate speedup
    if [ $simple_time -gt 0 ]; then
        speedup=$(echo "scale=2; $simple_time / $optimized_time" | bc 2>/dev/null || echo "N/A")
        echo "  Speedup: ${speedup}x"
    fi
    
    echo ""
}

# ==============================================================================
# Cache Optimization
# ==============================================================================

optimize_cache() {
    log_info "Optimizing build cache..."
    
    # Get current cache size
    GOCACHE=$(go env GOCACHE)
    GOMODCACHE=$(go env GOMODCACHE)
    
    if [ -d "$GOCACHE" ]; then
        cache_size=$(du -sh "$GOCACHE" 2>/dev/null | cut -f1)
        log_info "Current build cache size: $cache_size"
        
        # Clean old cache entries
        log_info "Cleaning old cache entries..."
        go clean -cache -testcache
        
        # Warm up cache with common dependencies
        log_info "Warming up cache with common dependencies..."
        go list -deps ./... | head -100 | xargs -P 8 -I {} go build {} 2>/dev/null || true
        
        log_success "Cache optimization completed"
    fi
    
    echo ""
}

# ==============================================================================
# Generate Optimized Makefile
# ==============================================================================

generate_optimized_makefile() {
    log_info "Generating optimized Makefile..."
    
    # Detect optimal settings
    CPUS=$(nproc 2>/dev/null || echo 4)
    PARALLEL=$((CPUS * 2))
    
    cat > Makefile.generated <<EOF
# Auto-generated optimized Makefile
# Generated on $(date)

# Optimized for your system
GOMAXPROCS := $CPUS
BUILD_PARALLEL := $PARALLEL
GOMEMLIMIT := 4GiB

# Build with all optimizations
.PHONY: build-optimized
build-optimized:
	@echo "Building with optimizations..."
	@go build \\
		-p=\$(BUILD_PARALLEL) \\
		-mod=readonly \\
		-trimpath \\
		-buildvcs=false \\
		-ldflags="-s -w" \\
		-gcflags="all=-l=4 -B" \\
		-tags="fast_build" \\
		./...

# Quick CI build
.PHONY: ci-build
ci-build:
	@echo "CI Build with timeout protection..."
	@timeout 120 \$(MAKE) build-optimized || (echo "Build timeout" && exit 1)

# Test with timeout
.PHONY: test-fast
test-fast:
	@echo "Running fast tests..."
	@go test -p=\$(BUILD_PARALLEL) -short -timeout=2m ./...
EOF
    
    log_success "Generated Makefile.generated with optimized settings"
    echo ""
}

# ==============================================================================
# Fix Common Issues
# ==============================================================================

fix_common_issues() {
    log_info "Fixing common build issues..."
    
    # 1. Fix go.mod
    log_info "Tidying go.mod..."
    go mod tidy -v
    
    # 2. Download dependencies
    log_info "Downloading all dependencies..."
    go mod download -x
    
    # 3. Verify modules
    log_info "Verifying module integrity..."
    go mod verify
    
    # 4. Fix build tags
    log_info "Checking for problematic build tags..."
    find . -name "*.go" -exec grep -l "// +build" {} \; | while read file; do
        if grep -q "// +build.*windows" "$file"; then
            log_warning "Found Windows-specific file: $file"
        fi
    done
    
    # 5. Clean vendor if exists
    if [ -d "vendor" ]; then
        log_warning "Vendor directory found. Consider removing it for faster builds."
        echo "Run: rm -rf vendor"
    fi
    
    log_success "Common issues fixed"
    echo ""
}

# ==============================================================================
# Recommendations
# ==============================================================================

print_recommendations() {
    log_info "Build Optimization Recommendations:"
    echo ""
    echo "1. Use the optimized Makefile:"
    echo "   make -f Makefile.optimized ci-build"
    echo ""
    echo "2. Set environment variables for CI:"
    echo "   export GOMAXPROCS=4"
    echo "   export GOMEMLIMIT=4GiB"
    echo "   export CGO_ENABLED=0"
    echo ""
    echo "3. Use build caching in CI:"
    echo "   - Cache ~/.cache/go-build"
    echo "   - Cache ~/go/pkg/mod"
    echo ""
    echo "4. Split large builds:"
    echo "   - Build cmd/* separately"
    echo "   - Build controllers separately"
    echo "   - Build pkg/* separately"
    echo ""
    echo "5. Use timeouts to prevent hanging:"
    echo "   timeout 120 go build ./..."
    echo ""
    echo "6. Consider using Go 1.24.6 specifically:"
    echo "   The go.mod specifies 1.24.6 but CI might use different version"
    echo ""
}

# ==============================================================================
# Main Execution
# ==============================================================================

main() {
    echo "========================================"
    echo "Go Build Performance Optimizer"
    echo "========================================"
    echo ""
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Run analysis and optimization
    analyze_system
    analyze_dependencies
    test_build_performance
    optimize_cache
    fix_common_issues
    generate_optimized_makefile
    print_recommendations
    
    log_success "Optimization complete!"
    echo ""
    echo "Next steps:"
    echo "1. Review the generated Makefile.generated"
    echo "2. Update your CI workflow to use Go 1.24.6"
    echo "3. Apply the recommended optimizations"
    echo "4. Test the build with: make -f Makefile.optimized ci-build"
}

# Run main function
main "$@"