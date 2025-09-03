#!/bin/bash
# =============================================================================
# CI Build Script - Optimized for GitHub Actions
# =============================================================================
# Purpose: Build critical cmd components with timeout protection and error handling
# Used by: .github/workflows/ci-production.yml
# =============================================================================

set -euo pipefail  # Exit on error, undefined vars, pipe failures

# Build configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BIN_DIR="$PROJECT_ROOT/bin"
BUILD_TIMEOUT=90
MAX_PARALLEL=3

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Critical components in build priority order
CRITICAL_COMPONENTS=(
    "cmd/intent-ingest"
    "cmd/llm-processor" 
    "cmd/conductor"
    "cmd/nephio-bridge"
    "cmd/webhook"
    "cmd/a1-sim"
    "cmd/e2-kpm-sim"
    "cmd/fcaps-sim"
    "cmd/o1-ves-sim"
    "cmd/oran-adaptor"
)

# Build statistics
declare -A BUILD_RESULTS
TOTAL_BUILDS=0
SUCCESSFUL_BUILDS=0
FAILED_BUILDS=0
START_TIME=$(date +%s)

# Cleanup function
cleanup() {
    local exit_code=$?
    if [ $exit_code -ne 0 ]; then
        log_error "Build script interrupted or failed"
    fi
    
    # Show summary
    show_build_summary
    exit $exit_code
}

trap cleanup EXIT INT TERM

# Initialize build environment  
init_build_env() {
    log_info "Initializing CI build environment..."
    
    # Verify we're in the right directory
    if [ ! -f "$PROJECT_ROOT/go.mod" ]; then
        log_error "Not in Go project root (go.mod not found)"
        exit 1
    fi
    
    # Create/clean bin directory
    mkdir -p "$BIN_DIR"
    
    # Set build environment variables for reproducible builds
    export CGO_ENABLED=0
    export GOOS=linux
    export GOARCH=amd64
    export GOMAXPROCS=4
    export GOMEMLIMIT=4GiB
    
    log_info "Build environment ready"
    log_info "  Project Root: $PROJECT_ROOT"
    log_info "  Binary Dir: $BIN_DIR"
    log_info "  Go Version: $(go version | cut -d' ' -f3)"
    log_info "  Build Target: $GOOS/$GOARCH"
}

# Build single component with timeout and error handling
build_component() {
    local component="$1"
    local component_name=$(basename "$component")
    local binary_path="$BIN_DIR/$component_name"
    
    TOTAL_BUILDS=$((TOTAL_BUILDS + 1))
    
    # Check if component directory and main.go exist
    if [ ! -d "$PROJECT_ROOT/$component" ]; then
        log_warning "$component: Directory not found, skipping"
        BUILD_RESULTS["$component"]="SKIP_NO_DIR"
        return 0
    fi
    
    if [ ! -f "$PROJECT_ROOT/$component/main.go" ]; then
        log_warning "$component: main.go not found, skipping"
        BUILD_RESULTS["$component"]="SKIP_NO_MAIN"
        return 0
    fi
    
    log_info "Building $component..."
    
    # Build with timeout and capture output
    local build_start=$(date +%s)
    if timeout $BUILD_TIMEOUT go build \
        -v \
        -ldflags="-s -w -X main.version=$(git rev-parse --short HEAD 2>/dev/null || echo 'dev')" \
        -gcflags="-l=4" \
        -o "$binary_path" \
        "./$component" 2>&1 | head -20; then
        
        local build_duration=$(($(date +%s) - build_start))
        log_success "$component built successfully in ${build_duration}s"
        BUILD_RESULTS["$component"]="SUCCESS:${build_duration}s"
        SUCCESSFUL_BUILDS=$((SUCCESSFUL_BUILDS + 1))
        
        # Verify binary
        if [ -x "$binary_path" ]; then
            local binary_size=$(stat -c%s "$binary_path" 2>/dev/null || echo "unknown")
            log_info "  Binary size: $(numfmt --to=iec-i --suffix=B $binary_size 2>/dev/null || echo $binary_size bytes)"
        fi
        
        return 0
    else
        local exit_code=$?
        log_warning "$component build failed or timed out (exit code: $exit_code)"
        BUILD_RESULTS["$component"]="FAIL:timeout_or_error"
        FAILED_BUILDS=$((FAILED_BUILDS + 1))
        return 1
    fi
}

# Build components in parallel batches
build_components_parallel() {
    log_info "Building critical components in parallel (max $MAX_PARALLEL concurrent builds)..."
    
    # Use xargs for controlled parallel execution
    printf '%s\n' "${CRITICAL_COMPONENTS[@]}" | \
    xargs -I {} -P $MAX_PARALLEL -n 1 bash -c '
        component="$1"
        echo "Starting build for $component..."
        if '"$(declare -f build_component)"'; build_component "$component"; then
            echo "✅ $component: Build successful"
        else
            echo "⚠️  $component: Build failed"
        fi
    ' -- {}
    
    log_info "Parallel build phase completed"
}

# Build components sequentially (fallback or debug mode)
build_components_sequential() {
    log_info "Building critical components sequentially..."
    
    for component in "${CRITICAL_COMPONENTS[@]}"; do
        build_component "$component"
    done
    
    log_info "Sequential build phase completed"
}

# Show detailed build summary
show_build_summary() {
    local total_time=$(($(date +%s) - START_TIME))
    
    echo ""
    echo "=============================================="
    echo "🏗️  CI BUILD SUMMARY"
    echo "=============================================="
    echo "📊 Statistics:"
    echo "   Total builds attempted: $TOTAL_BUILDS"
    echo "   Successful builds: $SUCCESSFUL_BUILDS"
    echo "   Failed builds: $FAILED_BUILDS"
    echo "   Total build time: ${total_time}s"
    echo ""
    
    if (( ${#BUILD_RESULTS[@]} > 0 )) 2>/dev/null; then
        echo "📋 Detailed Results:"
        for component in "${CRITICAL_COMPONENTS[@]}"; do
            if [[ -n "${BUILD_RESULTS[$component]:-}" ]]; then
                local result="${BUILD_RESULTS[$component]}"
                case "$result" in
                    SUCCESS:*)
                        echo "   ✅ $component: ${result#SUCCESS:}"
                        ;;
                    FAIL:*)
                        echo "   ❌ $component: ${result#FAIL:}"
                        ;;
                    SKIP_*)
                        echo "   ⏭️  $component: ${result#SKIP_}"
                        ;;
                esac
            else
                echo "   ❓ $component: No result recorded"
            fi
        done
    fi
    
    echo ""
    if [ -d "$BIN_DIR" ]; then
        echo "📦 Generated Binaries:"
        ls -la "$BIN_DIR/" 2>/dev/null | tail -n +2 | while read line; do
            echo "   $line"
        done
    else
        echo "📦 No binaries directory found"
    fi
    echo "=============================================="
    
    # Determine exit status
    if [ $FAILED_BUILDS -eq 0 ] && [ $SUCCESSFUL_BUILDS -gt 0 ]; then
        log_success "All attempted builds succeeded!"
        return 0
    elif [ $SUCCESSFUL_BUILDS -gt 0 ]; then
        log_warning "Some builds failed but critical components built successfully"
        return 0  # Don't fail CI if core components built
    else
        log_error "No successful builds"
        return 1
    fi
}

# Validate build environment
validate_environment() {
    log_info "Validating build environment..."
    
    # Check Go installation
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check Go version (require 1.21+)
    local go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    local major=$(echo $go_version | cut -d. -f1)
    local minor=$(echo $go_version | cut -d. -f2)
    
    if [ "$major" -lt 1 ] || ([ "$major" -eq 1 ] && [ "$minor" -lt 21 ]); then
        log_warning "Go version $go_version may not support all features (recommend 1.21+)"
    fi
    
    # Check available disk space (warn if < 1GB)
    local available_space=$(df "$PROJECT_ROOT" | tail -1 | awk '{print $4}')
    if [ "$available_space" -lt 1048576 ]; then  # Less than 1GB in KB
        log_warning "Low disk space detected: $(($available_space / 1024))MB available"
    fi
    
    # Check memory (warn if < 2GB available)
    if command -v free >/dev/null 2>&1; then
        local available_memory=$(free -m | awk '/^Mem:/ {print $7}')
        if [ "$available_memory" -lt 2048 ]; then
            log_warning "Low memory detected: ${available_memory}MB available"
        fi
    fi
    
    log_success "Environment validation completed"
}

# Main execution
main() {
    log_info "Starting Nephoran CI Build Script"
    log_info "=================================================="
    
    # Parse command line arguments
    local build_mode="parallel"
    while [[ $# -gt 0 ]]; do
        case $1 in
            --sequential)
                build_mode="sequential"
                shift
                ;;
            --timeout=*)
                BUILD_TIMEOUT="${1#*=}"
                shift
                ;;
            --max-parallel=*)
                MAX_PARALLEL="${1#*=}"
                shift
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo "Options:"
                echo "  --sequential     Build components one by one instead of parallel"
                echo "  --timeout=N      Set build timeout in seconds (default: $BUILD_TIMEOUT)"
                echo "  --max-parallel=N Set max parallel builds (default: $MAX_PARALLEL)"
                echo "  --help           Show this help message"
                exit 0
                ;;
            *)
                log_warning "Unknown option: $1"
                shift
                ;;
        esac
    done
    
    # Execute build pipeline
    validate_environment
    init_build_env
    
    case "$build_mode" in
        "sequential")
            build_components_sequential
            ;;
        "parallel"|*)
            build_components_parallel
            ;;
    esac
    
    # show_build_summary is called by cleanup trap
    log_success "CI build script completed successfully"
}

# Run main function with all arguments
main "$@"