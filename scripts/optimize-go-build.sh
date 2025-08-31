#!/bin/bash
# =============================================================================
# Ultra-Fast Go Build Optimizer for 2025 (Go 1.24+)
# =============================================================================
# This script implements cutting-edge Go build optimizations for maximum performance
# - Module download acceleration with advanced caching
# - Parallel build optimization with CPU core utilization
# - Memory-optimized compilation settings
# - Supply chain security with integrity checking
# =============================================================================

set -euo pipefail

# =============================================================================
# Configuration and Environment Setup
# =============================================================================

# Performance configuration
export GOMAXPROCS=${GOMAXPROCS:-$(nproc)}
export GOMEMLIMIT=${GOMEMLIMIT:-"8GiB"}
export GOGC=${GOGC:-50}  # Aggressive GC for build performance
export GODEBUG=${GODEBUG:-"gctrace=0,scavtrace=0"}

# Go 1.24+ experimental optimizations
export GOEXPERIMENT=${GOEXPERIMENT:-"swisstable,pacer,nocoverageredesign,rangefunc,aliastypeparams"}

# Build configuration
export CGO_ENABLED=${CGO_ENABLED:-0}
export GOOS=${GOOS:-linux}
export GOARCH=${GOARCH:-amd64}
export GOAMD64=${GOAMD64:-v3}

# Module proxy configuration for speed
export GOPROXY=${GOPROXY:-"https://proxy.golang.org,direct"}
export GOSUMDB=${GOSUMDB:-"sum.golang.org"}
export GONOPROXY=${GONOPROXY:-""}
export GONOSUMDB=${GONOSUMDB:-""}

# Advanced build flags for 2025
BUILD_TIMESTAMP=$(date -Iseconds)
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION=${VERSION:-"dev-${GIT_COMMIT}"}

# Cache directories
CACHE_DIR="${CACHE_DIR:-${HOME}/.cache/nephoran-build}"
GO_MOD_CACHE="${CACHE_DIR}/go-mod"
GO_BUILD_CACHE="${CACHE_DIR}/go-build"

# =============================================================================
# Logging and Utilities
# =============================================================================

log_info() {
    echo "[$(date +'%H:%M:%S')] INFO: $*" >&2
}

log_warn() {
    echo "[$(date +'%H:%M:%S')] WARN: $*" >&2
}

log_error() {
    echo "[$(date +'%H:%M:%S')] ERROR: $*" >&2
}

check_prerequisites() {
    local go_version
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    
    log_info "Go version: ${go_version}"
    log_info "Available CPU cores: $(nproc)"
    log_info "Available memory: $(free -h | awk '/^Mem:/ {print $2}')"
    log_info "GOMAXPROCS: ${GOMAXPROCS}"
    log_info "GOMEMLIMIT: ${GOMEMLIMIT}"
    
    # Verify Go 1.24+ for optimal performance
    if [[ "${go_version}" < "1.24" ]]; then
        log_warn "Go version ${go_version} detected. Consider upgrading to Go 1.24+ for optimal performance"
    fi
}

# =============================================================================
# Advanced Module Download Optimization
# =============================================================================

optimize_module_download() {
    log_info "Starting optimized module download..."
    
    # Create cache directories
    mkdir -p "${GO_MOD_CACHE}" "${GO_BUILD_CACHE}"
    chmod 755 "${GO_MOD_CACHE}" "${GO_BUILD_CACHE}"
    
    # Set cache environment
    export GOMODCACHE="${GO_MOD_CACHE}"
    export GOCACHE="${GO_BUILD_CACHE}"
    
    # Verify module integrity first
    log_info "Verifying go.mod integrity..."
    go mod verify || {
        log_warn "Module verification failed, attempting cleanup..."
        go clean -modcache
        go mod download
        go mod verify
    }
    
    # Parallel module download with retry logic
    log_info "Downloading modules with parallel processing..."
    local max_attempts=3
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        log_info "Download attempt ${attempt}/${max_attempts}"
        
        if timeout 120s go mod download -x 2>&1; then
            log_info "Module download completed successfully"
            break
        else
            log_warn "Download attempt ${attempt} failed"
            if [[ $attempt -lt $max_attempts ]]; then
                log_info "Retrying in 5 seconds..."
                sleep 5
                # Clean partial downloads
                go clean -modcache
            else
                log_error "All download attempts failed"
                return 1
            fi
        fi
        ((attempt++))
    done
    
    # Verify all modules are available
    log_info "Verifying module availability..."
    go list -m all > /dev/null || {
        log_error "Module list verification failed"
        return 1
    }
    
    log_info "Module download optimization completed"
}

# =============================================================================
# Advanced Build Optimization
# =============================================================================

optimize_build() {
    local service_name="$1"
    local output_path="$2"
    
    log_info "Building service: ${service_name}"
    log_info "Output path: ${output_path}"
    
    # Determine the correct main.go path
    local main_path
    case "${service_name}" in
        "planner")
            main_path="./planner/cmd/planner/main.go"
            ;;
        *)
            main_path="./cmd/${service_name}/main.go"
            ;;
    esac
    
    # Verify source exists
    if [[ ! -f "${main_path}" ]]; then
        log_error "Source file not found: ${main_path}"
        return 1
    fi
    
    log_info "Building from: ${main_path}"
    
    # Advanced build flags for maximum performance
    local ldflags=(
        "-s"                                    # Strip symbol table
        "-w"                                    # Strip DWARF debug info
        "-buildid="                             # Remove build ID for reproducibility
        "-X main.version=${VERSION}"
        "-X main.buildDate=${BUILD_TIMESTAMP}"
        "-X main.gitCommit=${GIT_COMMIT}"
        "-extldflags '-static'"                 # Static linking
    )
    
    local build_flags=(
        "-v"                                    # Verbose output
        "-trimpath"                             # Remove absolute paths
        "-buildvcs=false"                       # Disable VCS info for speed
        "-buildmode=exe"                        # Executable build mode
        "-compiler=gc"                          # Use gc compiler
        "-gccgoflags=-O3"                      # Optimization level 3
        "-gcflags=all=-l=4,-B,-dwarf=false"    # Aggressive inlining, no DWARF
        "-asmflags=all=-trimpath=$(pwd)"       # Trim assembly paths
        "-tags=netgo,osusergo,static_build"    # Static build tags
        "-installsuffix=netgo"                  # Install suffix for cgo
        "-a"                                    # Force rebuild
    )
    
    # Use parallel compilation
    local parallel_jobs=$(($(nproc) * 2))
    build_flags+=("-p" "${parallel_jobs}")
    
    log_info "Build configuration:"
    log_info "  - Parallel jobs: ${parallel_jobs}"
    log_info "  - LDFLAGS: ${ldflags[*]}"
    log_info "  - Build flags: ${build_flags[*]}"
    
    # Execute build with timing
    local start_time
    start_time=$(date +%s)
    
    time go build \
        "${build_flags[@]}" \
        -ldflags="${ldflags[*]}" \
        -o "${output_path}" \
        "${main_path}"
    
    local end_time
    end_time=$(date +%s)
    local build_duration=$((end_time - start_time))
    
    # Verify build success
    if [[ -x "${output_path}" ]]; then
        local binary_size
        binary_size=$(stat -c%s "${output_path}")
        log_info "Build completed successfully in ${build_duration}s"
        log_info "Binary size: $(numfmt --to=iec "${binary_size}")"
        log_info "Binary info: $(file "${output_path}")"
        
        # Verify static linking
        if ldd "${output_path}" 2>/dev/null; then
            log_warn "Binary has dynamic dependencies"
        else
            log_info "Static binary confirmed"
        fi
        
        # Optional UPX compression for large binaries
        if command -v upx >/dev/null 2>&1 && [[ "${binary_size}" -gt 10485760 ]]; then
            log_info "Compressing binary with UPX..."
            upx --best --lzma "${output_path}" 2>/dev/null || {
                log_warn "UPX compression failed or not beneficial"
            }
        fi
        
        return 0
    else
        log_error "Build failed or binary not executable"
        return 1
    fi
}

# =============================================================================
# Batch Build Optimization
# =============================================================================

build_all_services() {
    local services=(
        "intent-ingest"
        "conductor-loop"
        "llm-processor"
        "nephio-bridge"
        "oran-adaptor"
        "porch-publisher"
        "planner"
        "a1-sim"
        "e2-kpm-sim"
        "fcaps-sim"
        "o1-ves-sim"
    )
    
    local output_dir="${1:-./bin}"
    mkdir -p "${output_dir}"
    
    log_info "Building ${#services[@]} services in parallel..."
    
    local build_pids=()
    local build_results=()
    
    # Start parallel builds
    for service in "${services[@]}"; do
        local output_path="${output_dir}/${service}"
        
        # Check if source exists before starting build
        local main_path
        case "${service}" in
            "planner")
                main_path="./planner/cmd/planner/main.go"
                ;;
            *)
                main_path="./cmd/${service}/main.go"
                ;;
        esac
        
        if [[ -f "${main_path}" ]]; then
            log_info "Starting parallel build for ${service}..."
            (
                if optimize_build "${service}" "${output_path}"; then
                    echo "${service}:SUCCESS"
                else
                    echo "${service}:FAILED"
                fi
            ) &
            build_pids+=($!)
        else
            log_warn "Skipping ${service} - source not found at ${main_path}"
            build_results+=("${service}:SKIPPED")
        fi
    done
    
    # Wait for all builds to complete
    log_info "Waiting for ${#build_pids[@]} parallel builds to complete..."
    for pid in "${build_pids[@]}"; do
        wait "${pid}"
        # Capture build result would need process substitution for complex case
    done
    
    # Report build summary
    log_info "Build summary:"
    local successful=0
    local failed=0
    local skipped=0
    
    for service in "${services[@]}"; do
        local output_path="${output_dir}/${service}"
        if [[ -x "${output_path}" ]]; then
            local size
            size=$(stat -c%s "${output_path}")
            log_info "  ✅ ${service} ($(numfmt --to=iec "${size}"))"
            ((successful++))
        elif [[ ! -f "./cmd/${service}/main.go" ]] && [[ ! -f "./planner/cmd/${service}/main.go" ]]; then
            log_info "  ⏭️  ${service} (source not found)"
            ((skipped++))
        else
            log_info "  ❌ ${service} (build failed)"
            ((failed++))
        fi
    done
    
    log_info "Build completed: ${successful} successful, ${failed} failed, ${skipped} skipped"
    
    if [[ "${failed}" -gt 0 ]]; then
        return 1
    fi
    
    return 0
}

# =============================================================================
# Performance Analysis
# =============================================================================

analyze_performance() {
    log_info "Analyzing build performance..."
    
    # Go environment info
    log_info "Go environment:"
    go env | grep -E "GO(ROOT|PATH|MOD|CACHE|MAXPROCS|MEMLIMIT|GC|DEBUG|EXPERIMENT)"
    
    # Module cache statistics
    if [[ -d "${GO_MOD_CACHE}" ]]; then
        local mod_cache_size
        mod_cache_size=$(du -sh "${GO_MOD_CACHE}" | cut -f1)
        local mod_count
        mod_count=$(find "${GO_MOD_CACHE}" -name "*.mod" | wc -l)
        log_info "Module cache: ${mod_cache_size} (${mod_count} modules)"
    fi
    
    # Build cache statistics
    if [[ -d "${GO_BUILD_CACHE}" ]]; then
        local build_cache_size
        build_cache_size=$(du -sh "${GO_BUILD_CACHE}" | cut -f1)
        log_info "Build cache: ${build_cache_size}"
    fi
    
    # Dependency analysis
    log_info "Dependency analysis:"
    go list -m all | wc -l | xargs -I {} log_info "Total dependencies: {}"
    go list -m -u all 2>/dev/null | grep -c '\[' | xargs -I {} log_info "Outdated dependencies: {}" || true
}

# =============================================================================
# Main Execution
# =============================================================================

main() {
    local command="${1:-build}"
    shift || true
    
    log_info "Starting Go build optimization..."
    log_info "Command: ${command}"
    
    check_prerequisites
    
    case "${command}" in
        "download"|"deps")
            optimize_module_download
            ;;
        "build")
            local service="${1:-}"
            local output="${2:-./bin/${service}}"
            
            if [[ -z "${service}" ]]; then
                log_error "Usage: $0 build <service-name> [output-path]"
                exit 1
            fi
            
            optimize_module_download
            optimize_build "${service}" "${output}"
            ;;
        "build-all"|"all")
            local output_dir="${1:-./bin}"
            optimize_module_download
            build_all_services "${output_dir}"
            ;;
        "analyze"|"perf")
            analyze_performance
            ;;
        "clean")
            log_info "Cleaning build artifacts..."
            go clean -cache -modcache -testcache
            rm -rf "${CACHE_DIR}"
            log_info "Clean completed"
            ;;
        *)
            log_error "Unknown command: ${command}"
            echo "Usage: $0 {download|build|build-all|analyze|clean} [args...]"
            echo ""
            echo "Commands:"
            echo "  download     - Optimize module download"
            echo "  build        - Build a specific service"
            echo "  build-all    - Build all services"
            echo "  analyze      - Analyze performance"
            echo "  clean        - Clean caches"
            exit 1
            ;;
    esac
    
    log_info "Build optimization completed successfully"
}

# Execute main function with all arguments
main "$@"