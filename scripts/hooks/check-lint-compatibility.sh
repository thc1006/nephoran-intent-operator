#!/bin/bash
set -euo pipefail

# check-lint-compatibility.sh
# Pre-commit hook to check golangci-lint version compatibility
# Part of Nephoran Intent Operator DevOps pipeline

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Expected golangci-lint version (from CI configuration)
EXPECTED_VERSION="v1.64.3"
MIN_SUPPORTED_VERSION="v1.60.0"

# Extract version number for comparison
version_to_number() {
    local version="$1"
    # Remove 'v' prefix and convert to comparable number
    echo "$version" | sed 's/^v//' | awk -F. '{ printf("%03d%03d%03d", $1, $2, $3) }'
}

# Check golangci-lint version compatibility
check_version_compatibility() {
    log_info "Checking golangci-lint version compatibility..."
    
    # Check if golangci-lint is available
    if ! command -v golangci-lint &> /dev/null; then
        log_error "golangci-lint not found"
        log_error "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@$EXPECTED_VERSION"
        return 1
    fi
    
    # Get current version
    local current_version_output
    current_version_output=$(golangci-lint version 2>/dev/null || echo "unknown")
    
    # Extract version from output (format: "golangci-lint has version v1.64.3 built from ...")
    local current_version
    current_version=$(echo "$current_version_output" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "unknown")
    
    if [[ "$current_version" == "unknown" ]]; then
        log_error "Could not determine golangci-lint version"
        log_error "Output: $current_version_output"
        return 1
    fi
    
    log_info "Current version: $current_version"
    log_info "Expected version: $EXPECTED_VERSION"
    log_info "Minimum supported: $MIN_SUPPORTED_VERSION"
    
    # Convert versions to numbers for comparison
    local current_number expected_number min_number
    current_number=$(version_to_number "$current_version")
    expected_number=$(version_to_number "$EXPECTED_VERSION")
    min_number=$(version_to_number "$MIN_SUPPORTED_VERSION")
    
    # Check if version is supported
    if [[ $current_number -lt $min_number ]]; then
        log_error "golangci-lint version $current_version is too old"
        log_error "Minimum supported version: $MIN_SUPPORTED_VERSION"
        log_error "Update with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@$EXPECTED_VERSION"
        return 1
    fi
    
    # Check if version matches expected
    if [[ $current_number -eq $expected_number ]]; then
        log_success "golangci-lint version matches expected version: $current_version"
    elif [[ $current_number -gt $expected_number ]]; then
        log_warning "golangci-lint version $current_version is newer than expected $EXPECTED_VERSION"
        log_warning "This may work, but CI uses $EXPECTED_VERSION"
        log_info "Consider updating CI or downgrading to: go install github.com/golangci/golangci-lint/cmd/golangci-lint@$EXPECTED_VERSION"
    else
        log_warning "golangci-lint version $current_version is older than expected $EXPECTED_VERSION"
        log_warning "Consider updating to: go install github.com/golangci/golangci-lint/cmd/golangci-lint@$EXPECTED_VERSION"
    fi
    
    return 0
}

# Check for version-specific configuration issues
check_config_compatibility() {
    log_info "Checking configuration compatibility..."
    
    local config_file=".golangci-fast.yml"
    if [[ ! -f "$config_file" ]]; then
        log_warning "Primary config file not found: $config_file"
        return 0
    fi
    
    # Get golangci-lint version for specific checks
    local version
    version=$(golangci-lint version 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    local version_number
    version_number=$(version_to_number "$version")
    
    # Check for linters that were added in specific versions
    local v164_number v160_number
    v164_number=$(version_to_number "v1.64.0")
    v160_number=$(version_to_number "v1.60.0")
    
    # Linters introduced in v1.64+
    local new_linters_164=("copyloopvar" "intrange")
    
    for linter in "${new_linters_164[@]}"; do
        if grep -q "- $linter" "$config_file"; then
            if [[ $version_number -lt $v164_number ]]; then
                log_warning "Linter '$linter' requires golangci-lint v1.64+ (current: $version)"
            fi
        fi
    done
    
    # Check output format compatibility
    if grep -q "format: sarif" "$config_file"; then
        if [[ $version_number -lt $v160_number ]]; then
            log_warning "SARIF output format may not be fully supported in version $version"
        fi
    fi
    
    # Check for deprecated configuration options
    if grep -q "enable-all: true" "$config_file"; then
        log_warning "enable-all: true is deprecated and may cause issues"
        log_info "Consider explicitly listing enabled linters instead"
    fi
    
    # Check for reasonable concurrency settings
    local concurrency_line
    concurrency_line=$(grep "concurrency:" "$config_file" || true)
    if [[ -n "$concurrency_line" ]]; then
        local concurrency_value
        concurrency_value=$(echo "$concurrency_line" | sed 's/.*concurrency: *\([0-9]\+\).*/\1/')
        if [[ $concurrency_value -gt 8 ]]; then
            log_warning "High concurrency value ($concurrency_value) may cause resource issues"
        fi
    fi
    
    log_success "Configuration compatibility check completed"
    return 0
}

# Check Go version compatibility
check_go_compatibility() {
    log_info "Checking Go version compatibility..."
    
    if ! command -v go &> /dev/null; then
        log_warning "Go not found, skipping Go version check"
        return 0
    fi
    
    local go_version
    go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1 | sed 's/go//')
    
    log_info "Current Go version: $go_version"
    
    # Check minimum Go version for golangci-lint
    local go_version_number
    go_version_number=$(echo "$go_version" | awk -F. '{ printf("%03d%03d", $1, $2) }')
    local min_go_number
    min_go_number=$(echo "1.21" | awk -F. '{ printf("%03d%03d", $1, $2) }')
    
    if [[ $go_version_number -lt $min_go_number ]]; then
        log_error "Go version $go_version is too old for golangci-lint $EXPECTED_VERSION"
        log_error "Minimum required Go version: 1.21"
        return 1
    fi
    
    # Check against configured Go version in golangci config
    local config_file=".golangci-fast.yml"
    if [[ -f "$config_file" ]]; then
        local config_go_version
        config_go_version=$(grep "go:" "$config_file" | sed 's/.*go: *["\x27]*\([0-9]\+\.[0-9]\+\)["\x27]*.*/\1/' || echo "")
        
        if [[ -n "$config_go_version" ]]; then
            log_info "Configured Go version in lint config: $config_go_version"
            
            if [[ "$config_go_version" != "$go_version" ]]; then
                log_warning "Go version mismatch:"
                log_warning "  Current Go version: $go_version"
                log_warning "  Config Go version:  $config_go_version"
                log_info "Consider updating the configuration or Go installation"
            fi
        fi
    fi
    
    log_success "Go version compatibility check completed"
    return 0
}

# Main function
main() {
    local exit_code=0
    
    log_info "Starting golangci-lint compatibility check for Nephoran Intent Operator"
    
    # Run all compatibility checks
    if ! check_version_compatibility; then
        exit_code=1
    fi
    
    if ! check_config_compatibility; then
        exit_code=1
    fi
    
    if ! check_go_compatibility; then
        exit_code=1
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        log_success "All compatibility checks passed!"
        log_info "golangci-lint configuration is compatible with your environment"
    else
        log_error "Compatibility issues detected"
        log_error "Please resolve the issues above before committing"
    fi
    
    return $exit_code
}

# Run main function
main "$@"