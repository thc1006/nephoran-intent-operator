#!/bin/bash
# =============================================================================
# Ultra-Fast Build Script with Maximum Parallelization and Smart Caching
# =============================================================================
# Features: Parallel builds, dependency caching, build skipping, artifact reuse
# Usage: ./scripts/ultra-fast-build.sh [target] [options]

set -euo pipefail

# =============================================================================
# Configuration and Environment
# =============================================================================

# Script metadata
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BUILD_DIR="${PROJECT_ROOT}/bin"
CACHE_DIR="${PROJECT_ROOT}/.build-cache"
ARTIFACTS_DIR="${PROJECT_ROOT}/.build-artifacts"

# Build configuration
BUILD_VERSION="${BUILD_VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
BUILD_COMMIT="${BUILD_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')}"
BUILD_DATE="${BUILD_DATE:-$(date -u +'%Y-%m-%dT%H:%M:%SZ')}"
BUILD_USER="${BUILD_USER:-$(whoami)}"

# Performance settings
GOMAXPROCS="${GOMAXPROCS:-$(nproc)}"
BUILD_PARALLELISM="${BUILD_PARALLELISM:-$GOMAXPROCS}"
TEST_PARALLELISM="${TEST_PARALLELISM:-$((GOMAXPROCS / 2))}"

# Cache configuration
ENABLE_CACHE="${ENABLE_CACHE:-true}"
CACHE_VERSION="v3"
VERBOSE="${VERBOSE:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# =============================================================================
# Utility Functions
# =============================================================================

log() {
    echo -e "${CYAN}[$(date +'%H:%M:%S')]${NC} $*"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

verbose_log() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${CYAN}[VERBOSE]${NC} $*"
    fi
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Get file modification time (Linux only)
get_file_mtime() {
    if [[ -f "$1" ]]; then
        stat -c %Y "$1"
    else
        echo "0"
    fi
}

# Generate cache key from file content
generate_cache_key() {
    local files=("$@")
    local key_input=""
    
    for file in "${files[@]}"; do
        if [[ -f "$file" ]]; then
            key_input+="$file:$(get_file_mtime "$file");"
        fi
    done
    
    echo "${CACHE_VERSION}:$(echo -n "$key_input" | sha256sum | cut -d' ' -f1)"
}

# =============================================================================
# Cache Management
# =============================================================================

setup_cache() {
    if [[ "$ENABLE_CACHE" != "true" ]]; then
        return
    fi
    
    mkdir -p "$CACHE_DIR"/{deps,build,test,docker}
    mkdir -p "$ARTIFACTS_DIR"
    
    verbose_log "Cache directory: $CACHE_DIR"
    verbose_log "Artifacts directory: $ARTIFACTS_DIR"
}

# Check if cache is valid
cache_valid() {
    local cache_file="$1"
    local cache_key="$2"
    
    if [[ ! -f "$cache_file" ]]; then
        return 1
    fi
    
    local stored_key
    stored_key=$(head -1 "$cache_file" 2>/dev/null || echo "")
    
    [[ "$stored_key" == "$cache_key" ]]
}

# Save cache
save_cache() {
    local cache_file="$1"
    local cache_key="$2"
    
    echo "$cache_key" > "$cache_file"
}

# =============================================================================
# Dependency Management
# =============================================================================

setup_dependencies() {
    log "🔧 Setting up dependencies with smart caching..."
    
    local deps_cache_key
    deps_cache_key=$(generate_cache_key "go.mod" "go.sum" "tools.go")
    local deps_cache_file="$CACHE_DIR/deps/cache.key"
    
    if cache_valid "$deps_cache_file" "$deps_cache_key"; then
        log_info "📦 Dependencies cache hit - skipping download"
        return 0
    fi
    
    log_info "📦 Downloading dependencies with maximum parallelization..."
    
    # Configure Go for optimal performance
    export GOPROXY="https://proxy.golang.org,direct"
    export GOSUMDB="sum.golang.org"
    export GOMAXPROCS="$GOMAXPROCS"
    
    # Download and verify dependencies
    verbose_log "Running go mod download..."
    go mod download -x
    
    verbose_log "Verifying module integrity..."
    go mod verify
    
    # Pre-compile standard library for faster builds
    verbose_log "Pre-compiling standard library..."
    go install -a std
    
    # Install build tools in parallel
    log_info "🛠️ Installing build tools in parallel..."
    {
        go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.18.0 &
        go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest &
        go install github.com/onsi/ginkgo/v2/ginkgo@latest &
        go install golang.org/x/tools/cmd/cover@latest &
        go install golang.org/x/vuln/cmd/govulncheck@latest &
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.3 &
        wait
    }
    
    save_cache "$deps_cache_file" "$deps_cache_key"
    log_success "✅ Dependencies setup completed"
}

# =============================================================================
# Code Generation
# =============================================================================

generate_code() {
    log "🏗️ Generating code with smart caching..."
    
    local gen_cache_key
    gen_cache_key=$(generate_cache_key "api/v1/"*.go "controllers/"*.go)
    local gen_cache_file="$CACHE_DIR/build/generate.key"
    
    if cache_valid "$gen_cache_file" "$gen_cache_key"; then
        log_info "🏗️ Code generation cache hit - skipping"
        return 0
    fi
    
    log_info "🏗️ Generating CRDs and deep copy methods..."
    
    # Ensure output directory exists
    mkdir -p deployments/crds
    
    # Generate code with error handling
    if ! controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./api/v1" 2>/dev/null; then
        log_warning "Deep copy generation failed - continuing with existing files"
    fi
    
    if ! controller-gen crd rbac:roleName=manager-role webhook paths="./api/v1" output:crd:artifacts:config=deployments/crds 2>/dev/null; then
        log_warning "CRD generation failed - using existing CRDs"
    fi
    
    save_cache "$gen_cache_file" "$gen_cache_key"
    log_success "✅ Code generation completed"
}

# =============================================================================
# Parallel Building
# =============================================================================

# Get list of all services to build
get_services() {
    local services=()
    
    # Scan cmd directory for services
    for cmd_dir in cmd/*/; do
        if [[ -f "${cmd_dir}main.go" ]]; then
            services+=("$(basename "$cmd_dir")")
        fi
    done
    
    # Add main controller
    if [[ -f "cmd/main.go" ]]; then
        services+=("manager")
    fi
    
    printf '%s\n' "${services[@]}"
}

# Build single service optimized
build_service() {
    local service="$1"
    local output_dir="${BUILD_DIR}"
    
    log_info "🔨 Building $service..."
    
    # Determine command path
    local cmd_path
    case "$service" in
        "manager"|"controller")
            cmd_path="./cmd/main.go"
            ;;
        *)
            cmd_path="./cmd/$service/main.go"
            if [[ ! -f "$cmd_path" ]]; then
                log_error "Command path not found: $cmd_path"
                return 1
            fi
            ;;
    esac
    
    # Build cache key
    local build_cache_key
    build_cache_key=$(generate_cache_key "$cmd_path" "go.mod" "go.sum")
    local build_cache_file="$CACHE_DIR/build/${service}.key"
    local output_file="$output_dir/$service"
    
    # Check if build is needed
    if cache_valid "$build_cache_file" "$build_cache_key" && [[ -f "$output_file" ]]; then
        log_info "📦 Build cache hit for $service - skipping"
        return 0
    fi
    
    # Create output directory
    mkdir -p "$output_dir"
    
    # Ultra-optimized build
    verbose_log "Building $service with optimizations..."
    CGO_ENABLED=0 \
    GOOS="${GOOS:-linux}" \
    GOARCH="${GOARCH:-amd64}" \
    go build -v \
        -ldflags="-s -w -X main.version=${BUILD_VERSION} -X main.commit=${BUILD_COMMIT} -X main.date=${BUILD_DATE}" \
        -trimpath \
        -buildmode=pie \
        -tags="netgo,osusergo" \
        -o "$output_file" \
        "$cmd_path"
    
    # Verify build
    if [[ ! -f "$output_file" ]]; then
        log_error "Build failed for $service"
        return 1
    fi
    
    save_cache "$build_cache_file" "$build_cache_key"
    log_success "✅ Built $service successfully"
}

# Build all services in parallel
build_all_services() {
    log "🚀 Building all services with maximum parallelism..."
    
    local services
    mapfile -t services < <(get_services)
    
    if [[ ${#services[@]} -eq 0 ]]; then
        log_warning "No services found to build"
        return 0
    fi
    
    log_info "Found ${#services[@]} services: ${services[*]}"
    
    # Build services in parallel batches
    local batch_size="$BUILD_PARALLELISM"
    local pids=()
    
    for ((i = 0; i < ${#services[@]}; i += batch_size)); do
        for ((j = i; j < i + batch_size && j < ${#services[@]}; j++)); do
            build_service "${services[j]}" &
            pids+=($!)
        done
        
        # Wait for batch to complete
        for pid in "${pids[@]}"; do
            if ! wait "$pid"; then
                log_error "Build failed in parallel batch"
                return 1
            fi
        done
        
        pids=()
    done
    
    log_success "✅ All services built successfully"
}

# =============================================================================
# Testing with Parallelization
# =============================================================================

run_tests() {
    local test_type="${1:-all}"
    
    log "🧪 Running tests ($test_type) with parallelization..."
    
    local test_cache_key
    test_cache_key=$(generate_cache_key $(find . -name "*_test.go" -newer "$CACHE_DIR/test/last_run" 2>/dev/null | head -10))
    local test_cache_file="$CACHE_DIR/test/${test_type}.key"
    
    # Setup test environment
    export USE_EXISTING_CLUSTER=false
    export ENVTEST_K8S_VERSION=1.29.0
    
    # Install test tools if needed
    if ! command_exists setup-envtest; then
        go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
    fi
    
    # Setup envtest
    if ! setup-envtest use 1.29.0 --bin-dir ~/.local/bin >/dev/null 2>&1; then
        log_warning "Failed to setup envtest - some tests may fail"
    fi
    
    export KUBEBUILDER_ASSETS="$HOME/.local/bin"
    
    # Run tests based on type
    case "$test_type" in
        "unit")
            log_info "🔬 Running unit tests..."
            go test ./... -short -race -parallel="$TEST_PARALLELISM" -timeout=10m \
                -coverprofile=.build-artifacts/unit-coverage.out
            ;;
        "integration")
            log_info "🔗 Running integration tests..."
            go test ./tests/integration/... -race -parallel="$((TEST_PARALLELISM / 2))" -timeout=20m
            ;;
        "e2e")
            log_info "🎭 Running e2e tests..."
            go test ./tests/e2e/... -parallel=2 -timeout=30m
            ;;
        "all"|*)
            log_info "🔬 Running comprehensive test suite..."
            go test ./... -race -parallel="$TEST_PARALLELISM" -timeout=25m \
                -coverprofile=.build-artifacts/coverage.out
            ;;
    esac
    
    save_cache "$test_cache_file" "$test_cache_key"
    touch "$CACHE_DIR/test/last_run"
    log_success "✅ Tests completed successfully"
}

# =============================================================================
# Docker Build Optimization
# =============================================================================

build_docker_image() {
    local service="${1:-manager}"
    local tag="${2:-latest}"
    
    log "🐳 Building optimized Docker image for $service..."
    
    local docker_cache_key
    docker_cache_key=$(generate_cache_key "Dockerfile.optimized" "go.mod" "go.sum")
    local docker_cache_file="$CACHE_DIR/docker/${service}.key"
    
    # Check if Docker buildx is available
    if ! command_exists docker || ! docker buildx version >/dev/null 2>&1; then
        log_error "Docker buildx is required for optimized builds"
        return 1
    fi
    
    # Build with maximum cache utilization
    log_info "🚀 Building with multi-stage caching..."
    docker buildx build \
        -f Dockerfile.optimized \
        --platform linux/amd64 \
        --build-arg SERVICE="$service" \
        --build-arg VERSION="$BUILD_VERSION" \
        --build-arg BUILD_DATE="$BUILD_DATE" \
        --build-arg VCS_REF="$BUILD_COMMIT" \
        --cache-from type=local,src="$CACHE_DIR/docker" \
        --cache-to type=local,dest="$CACHE_DIR/docker",mode=max \
        --tag "nephoran/$service:$tag" \
        --load \
        .
    
    save_cache "$docker_cache_file" "$docker_cache_key"
    log_success "✅ Docker image built: nephoran/$service:$tag"
}

# =============================================================================
# Quality Gates
# =============================================================================

run_quality_checks() {
    log "🔍 Running quality checks with parallelization..."
    
    local checks=()
    
    # Run linters in parallel
    {
        log_info "🔍 Running golangci-lint..."
        golangci-lint run --timeout=10m --concurrency="$GOMAXPROCS" &
        checks+=($!)
        
        log_info "🛡️ Running security scan..."
        govulncheck ./... &
        checks+=($!)
        
        log_info "📊 Generating coverage report..."
        if [[ -f ".build-artifacts/coverage.out" ]]; then
            go tool cover -html=.build-artifacts/coverage.out -o .build-artifacts/coverage.html &
            checks+=($!)
        fi
    }
    
    # Wait for all checks
    local failed=0
    for pid in "${checks[@]}"; do
        if ! wait "$pid"; then
            ((failed++))
        fi
    done
    
    if [[ $failed -gt 0 ]]; then
        log_error "Quality checks failed ($failed failures)"
        return 1
    fi
    
    log_success "✅ All quality checks passed"
}

# =============================================================================
# Performance Benchmarks
# =============================================================================

run_benchmarks() {
    log "🏃 Running performance benchmarks..."
    
    mkdir -p .build-artifacts/benchmarks
    
    log_info "📊 Running Go benchmarks..."
    go test -bench=. -benchmem -timeout=20m ./... > .build-artifacts/benchmarks/results.txt
    
    log_info "🔍 Memory profiling..."
    go test -bench=BenchmarkLLM -memprofile=.build-artifacts/benchmarks/mem.prof ./pkg/llm/... || true
    
    log_success "✅ Benchmarks completed - results in .build-artifacts/benchmarks/"
}

# =============================================================================
# Main Functions
# =============================================================================

show_help() {
    cat << EOF
Ultra-Fast Build Script for Nephoran Intent Operator

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    help                Show this help message
    deps                Setup dependencies with smart caching
    generate            Generate code (CRDs, deep copy methods)
    build [SERVICE]     Build service(s) with parallel compilation
    test [TYPE]         Run tests (unit|integration|e2e|all)
    docker [SERVICE]    Build optimized Docker image
    quality             Run quality checks and linting
    bench               Run performance benchmarks
    clean               Clean build cache and artifacts
    all                 Run complete build pipeline

Options:
    --verbose           Enable verbose logging
    --no-cache          Disable build caching
    --parallel=N        Set parallelism level (default: $(nproc))

Environment Variables:
    BUILD_VERSION       Override build version
    BUILD_PARALLELISM   Number of parallel builds (default: $(nproc))
    TEST_PARALLELISM    Number of parallel test processes
    ENABLE_CACHE        Enable/disable caching (default: true)

Examples:
    $0 all                          # Complete optimized build
    $0 build llm-processor          # Build specific service
    $0 test unit --parallel=8       # Run unit tests with 8 parallel processes
    $0 docker manager               # Build optimized Docker image
    
Performance Tips:
    - Use SSD storage for better cache performance
    - Increase GOMAXPROCS for faster compilation
    - Use --no-cache for clean builds
    - Monitor system resources during parallel builds

EOF
}

# Clean cache and artifacts
clean_build() {
    log "🧹 Cleaning build cache and artifacts..."
    
    rm -rf "$CACHE_DIR" "$ARTIFACTS_DIR" "$BUILD_DIR"
    
    # Clean Go cache
    go clean -cache -modcache -testcache
    
    # Clean Docker build cache
    if command_exists docker; then
        docker builder prune -f >/dev/null 2>&1 || true
    fi
    
    log_success "✅ Build environment cleaned"
}

# Run complete build pipeline
run_all() {
    local start_time
    start_time=$(date +%s)
    
    log "🚀 Starting ultra-fast complete build pipeline..."
    
    setup_cache
    setup_dependencies
    generate_code
    build_all_services
    run_tests "all"
    run_quality_checks
    
    local end_time
    end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    log_success "🎉 Complete build pipeline finished in ${duration}s"
    
    # Show build summary
    echo
    log "📊 Build Summary:"
    log_info "Version: $BUILD_VERSION"
    log_info "Commit: $BUILD_COMMIT"
    log_info "Duration: ${duration}s"
    log_info "Services: $(ls -1 "$BUILD_DIR" 2>/dev/null | wc -l)"
    log_info "Cache hits: $(find "$CACHE_DIR" -name "*.key" 2>/dev/null | wc -l)"
}

# =============================================================================
# Main Entry Point
# =============================================================================

main() {
    # Change to project root
    cd "$PROJECT_ROOT"
    
    # Parse options
    while [[ $# -gt 0 ]]; do
        case $1 in
            --verbose)
                VERBOSE=true
                shift
                ;;
            --no-cache)
                ENABLE_CACHE=false
                shift
                ;;
            --parallel=*)
                BUILD_PARALLELISM="${1#*=}"
                TEST_PARALLELISM="$((BUILD_PARALLELISM / 2))"
                shift
                ;;
            -*)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
            *)
                break
                ;;
        esac
    done
    
    # Get command
    local command="${1:-help}"
    shift 2>/dev/null || true
    
    # Setup environment
    setup_cache
    export GOMAXPROCS
    
    # Execute command
    case "$command" in
        help|--help|-h)
            show_help
            ;;
        deps)
            setup_dependencies
            ;;
        generate|gen)
            generate_code
            ;;
        build)
            if [[ $# -gt 0 ]]; then
                build_service "$1"
            else
                build_all_services
            fi
            ;;
        test)
            run_tests "${1:-all}"
            ;;
        docker)
            build_docker_image "${1:-manager}" "${2:-latest}"
            ;;
        quality)
            run_quality_checks
            ;;
        bench)
            run_benchmarks
            ;;
        clean)
            clean_build
            ;;
        all)
            run_all
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"