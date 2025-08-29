#!/bin/bash
set -euo pipefail

# =============================================================================
# Resilient Go Module Download Script for CI/CD
# =============================================================================
# This script implements enterprise-grade retry logic and fallback mechanisms
# for reliable dependency resolution in O-RAN/Nephio environments.
#
# Features:
# - Multiple proxy fallbacks
# - Exponential backoff retry
# - Cache warming strategies
# - Network resilience patterns
# - Detailed error diagnostics
# =============================================================================

# Configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
readonly MAX_RETRIES=5
readonly BASE_DELAY=2
readonly MAX_DELAY=60
readonly TIMEOUT=300

# Color codes for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

# Error handler
error_handler() {
    local line_number=$1
    log_error "Script failed at line $line_number"
    log_error "Last command: ${BASH_COMMAND}"
    
    # Capture Go environment for debugging
    log_error "Go environment debug info:"
    go env GOPROXY GOSUMDB GOMODCACHE GOCACHE || true
    
    exit 1
}

trap 'error_handler $LINENO' ERR

# Exponential backoff retry function
retry_with_backoff() {
    local max_attempts=$1
    local delay=$2
    local max_delay=$3
    shift 3
    local command=("$@")
    
    local attempt=1
    while [ $attempt -le $max_attempts ]; do
        log_info "Attempt $attempt/$max_attempts: ${command[*]}"
        
        if "${command[@]}"; then
            log_success "Command succeeded on attempt $attempt"
            return 0
        fi
        
        if [ $attempt -eq $max_attempts ]; then
            log_error "Command failed after $max_attempts attempts"
            return 1
        fi
        
        local current_delay=$(( delay * (2 ** (attempt - 1)) ))
        if [ $current_delay -gt $max_delay ]; then
            current_delay=$max_delay
        fi
        
        log_warn "Retrying in ${current_delay}s..."
        sleep $current_delay
        ((attempt++))
    done
}

# Validate Go installation
validate_go() {
    log_info "Validating Go installation..."
    
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is not installed or not in PATH"
        return 1
    fi
    
    local go_version
    go_version=$(go version | cut -d' ' -f3 | sed 's/go//')
    log_info "Go version: $go_version"
    
    # Check minimum version (1.21.0)
    if ! printf '%s\n%s\n' "1.21.0" "$go_version" | sort -V | head -n1 | grep -q "1.21.0"; then
        log_error "Go version $go_version is below minimum required version 1.21.0"
        return 1
    fi
    
    log_success "Go installation validated"
    return 0
}

# Configure Go environment for reliability
configure_go_env() {
    log_info "Configuring Go environment for reliability..."
    
    # Load configuration from .goproxy-config if available
    if [ -f "$PROJECT_ROOT/.goproxy-config" ]; then
        log_info "Loading Go proxy configuration..."
        # shellcheck source=/dev/null
        source "$PROJECT_ROOT/.goproxy-config" 2>/dev/null || true
    fi
    
    # Set proxy with fallbacks (preserving existing settings)
    export GOPROXY="${GOPROXY:-https://proxy.golang.org,https://goproxy.cn,https://goproxy.io,direct}"
    export GOSUMDB="${GOSUMDB:-sum.golang.org}"
    export GOPRIVATE="${GOPRIVATE:-github.com/thc1006/*,github.com/nephoran/*}"
    
    # Configure cache directories
    if [ "${CI:-false}" = "true" ]; then
        # CI/CD environment - use predictable cache locations
        export GOMODCACHE="${GOMODCACHE_CI:-/tmp/.cache/go-mod}"
        export GOCACHE="${GOCACHE_CI:-/tmp/.cache/go-build}"
        
        # Ensure cache directories exist and are writable
        mkdir -p "$GOMODCACHE" "$GOCACHE"
        chmod 755 "$GOMODCACHE" "$GOCACHE"
        
        # Set ownership if possible (ignore failures in containers)
        chown "$(id -u):$(id -g)" "$GOMODCACHE" "$GOCACHE" 2>/dev/null || true
    fi
    
    # Performance optimizations
    export GOMAXPROCS="${GOMAXPROCS:-$(nproc)}"
    export GOGC="${GOGC:-100}"
    
    log_info "Go environment configured:"
    log_info "  GOPROXY: $GOPROXY"
    log_info "  GOSUMDB: $GOSUMDB"
    log_info "  GOPRIVATE: $GOPRIVATE"
    log_info "  GOMODCACHE: ${GOMODCACHE:-default}"
    log_info "  GOCACHE: ${GOCACHE:-default}"
}

# Test network connectivity to Go proxy
test_proxy_connectivity() {
    log_info "Testing connectivity to Go proxy services..."
    
    local proxies=("proxy.golang.org" "goproxy.cn" "goproxy.io")
    local available_proxies=()
    
    for proxy in "${proxies[@]}"; do
        if timeout 10 curl -sSf "https://$proxy" >/dev/null 2>&1; then
            log_success "Proxy $proxy is accessible"
            available_proxies+=("$proxy")
        else
            log_warn "Proxy $proxy is not accessible"
        fi
    done
    
    if [ ${#available_proxies[@]} -eq 0 ]; then
        log_error "No Go proxy services are accessible"
        return 1
    fi
    
    log_info "Available proxies: ${available_proxies[*]}"
    return 0
}

# Warm up module cache with critical dependencies
warm_cache() {
    log_info "Warming up module cache with critical dependencies..."
    
    local critical_modules=(
        "k8s.io/client-go@v0.33.3"
        "sigs.k8s.io/controller-runtime@v0.21.0"
        "github.com/prometheus/client_golang@v1.22.0"
        "go.opentelemetry.io/otel@v1.37.0"
        "github.com/go-logr/logr@v1.4.3"
    )
    
    for module in "${critical_modules[@]}"; do
        log_info "Pre-downloading critical module: $module"
        retry_with_backoff 3 1 5 go mod download "$module" || {
            log_warn "Failed to pre-download $module (continuing)"
        }
    done
}

# Clean corrupted cache if needed
clean_cache_if_corrupted() {
    log_info "Checking for cache corruption..."
    
    # Check if go.sum exists and is valid
    if [ -f go.sum ]; then
        if ! go mod verify >/dev/null 2>&1; then
            log_warn "Module verification failed, cleaning cache..."
            go clean -modcache || true
            go clean -cache || true
            log_info "Cache cleaned, will re-download modules"
            return 0
        fi
    fi
    
    log_success "Module cache appears healthy"
    return 0
}

# Main module download function with comprehensive error handling
download_modules() {
    log_info "Starting Go module download with enhanced reliability..."
    
    cd "$PROJECT_ROOT" || {
        log_error "Cannot change to project root directory"
        return 1
    }
    
    # Verify go.mod exists
    if [ ! -f go.mod ]; then
        log_error "go.mod file not found in $PROJECT_ROOT"
        return 1
    fi
    
    # Strategy 1: Standard download with timeout
    log_info "Attempting standard module download..."
    if timeout $TIMEOUT go mod download -x; then
        log_success "Standard module download completed successfully"
        return 0
    fi
    
    log_warn "Standard download failed or timed out, trying recovery strategies..."
    
    # Strategy 2: Clean cache and retry
    log_info "Cleaning cache and retrying..."
    go clean -modcache || true
    go clean -cache || true
    
    if retry_with_backoff 3 "$BASE_DELAY" "$MAX_DELAY" timeout $TIMEOUT go mod download -x; then
        log_success "Module download succeeded after cache clean"
        return 0
    fi
    
    # Strategy 3: Download with verbose logging for debugging
    log_info "Attempting download with verbose logging..."
    if timeout $TIMEOUT go mod download -v 2>&1 | tee /tmp/go-download-debug.log; then
        log_success "Verbose download completed successfully"
        return 0
    fi
    
    # Strategy 4: Individual module download
    log_info "Attempting individual module downloads..."
    local modules
    mapfile -t modules < <(go list -m all | tail -n +2 | cut -d' ' -f1)
    
    local failed_modules=()
    for module in "${modules[@]}"; do
        if ! timeout 30 go mod download "$module" 2>/dev/null; then
            failed_modules+=("$module")
            log_warn "Failed to download module: $module"
        fi
    done
    
    if [ ${#failed_modules[@]} -eq 0 ]; then
        log_success "All modules downloaded individually"
        return 0
    fi
    
    # Strategy 5: Final attempt with direct proxy bypass
    log_info "Final attempt with direct downloads (bypass proxy)..."
    GOPROXY=direct retry_with_backoff 2 5 10 timeout $TIMEOUT go mod download -x
    
    if [ $? -eq 0 ]; then
        log_success "Direct download succeeded"
        return 0
    fi
    
    # If we reach here, all strategies failed
    log_error "All module download strategies failed"
    log_error "Failed modules: ${failed_modules[*]}"
    
    # Provide debugging information
    log_error "Debug information:"
    log_error "Go environment:"
    go env | grep -E "(PROXY|SUMDB|PRIVATE|CACHE)" || true
    
    if [ -f /tmp/go-download-debug.log ]; then
        log_error "Last verbose download log (last 20 lines):"
        tail -n 20 /tmp/go-download-debug.log || true
    fi
    
    return 1
}

# Verify downloaded modules
verify_modules() {
    log_info "Verifying downloaded modules..."
    
    if go mod verify; then
        log_success "All modules verified successfully"
        return 0
    else
        log_error "Module verification failed"
        return 1
    fi
}

# Generate module report
generate_module_report() {
    log_info "Generating module dependency report..."
    
    local report_file="/tmp/go-modules-report.txt"
    {
        echo "=== Go Module Dependency Report ==="
        echo "Generated at: $(date -u -Iseconds)"
        echo "Project: $(basename "$PROJECT_ROOT")"
        echo "Go version: $(go version)"
        echo ""
        echo "=== Direct Dependencies ==="
        go list -m -f '{{.Path}} {{.Version}}' all | head -n 20
        echo ""
        echo "=== Module Graph Summary ==="
        go mod graph | wc -l | xargs echo "Total dependencies:"
        echo ""
        echo "=== Vulnerabilities Check ==="
        if command -v govulncheck >/dev/null 2>&1; then
            govulncheck -show verbose ./... 2>/dev/null | head -n 10 || echo "No vulnerabilities found or govulncheck failed"
        else
            echo "govulncheck not available"
        fi
    } > "$report_file"
    
    log_success "Module report generated: $report_file"
    
    # Display summary
    if [ "${CI:-false}" = "true" ]; then
        log_info "=== Dependency Summary ==="
        head -n 30 "$report_file"
    fi
}

# Main execution
main() {
    log_info "Starting resilient Go module download process..."
    
    # Pre-flight checks
    validate_go || exit 1
    configure_go_env
    
    # Network connectivity test (non-fatal)
    test_proxy_connectivity || log_warn "Proxy connectivity issues detected"
    
    # Cache management
    clean_cache_if_corrupted
    
    # Optional cache warming
    if [ "${WARM_CACHE:-true}" = "true" ]; then
        warm_cache || log_warn "Cache warming failed (continuing)"
    fi
    
    # Main download process
    if ! download_modules; then
        log_error "Module download failed completely"
        exit 1
    fi
    
    # Verification
    verify_modules || {
        log_error "Module verification failed"
        exit 1
    }
    
    # Report generation
    generate_module_report || log_warn "Report generation failed (non-fatal)"
    
    log_success "Resilient Go module download completed successfully!"
    
    # Performance metrics
    if [ "${CI:-false}" = "true" ]; then
        log_info "=== Performance Metrics ==="
        log_info "GOMODCACHE size: $(du -sh "${GOMODCACHE:-$(go env GOMODCACHE)}" 2>/dev/null | cut -f1 || echo "unknown")"
        log_info "GOCACHE size: $(du -sh "${GOCACHE:-$(go env GOCACHE)}" 2>/dev/null | cut -f1 || echo "unknown")"
    fi
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi