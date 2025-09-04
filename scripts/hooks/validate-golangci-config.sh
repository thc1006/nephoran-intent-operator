#!/bin/bash
set -euo pipefail

# validate-golangci-config.sh
# Pre-commit hook to validate golangci-lint configuration files
# Part of Nephoran Intent Operator DevOps pipeline

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

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

# Check if golangci-lint is available
check_golangci_lint() {
    if ! command -v golangci-lint &> /dev/null; then
        log_error "golangci-lint not found. Installing..."
        
        # Try to install golangci-lint
        if command -v go &> /dev/null; then
            log_info "Installing golangci-lint via go install..."
            go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.3
            
            # Add GOPATH/bin to PATH if not already there
            export PATH="$(go env GOPATH)/bin:$PATH"
            
            if ! command -v golangci-lint &> /dev/null; then
                log_error "Failed to install golangci-lint via go install"
                return 1
            fi
        else
            log_error "Go not found. Cannot install golangci-lint"
            return 1
        fi
    fi
    
    log_info "golangci-lint version: $(golangci-lint version 2>/dev/null | head -1 || echo 'unknown')"
    return 0
}

# Validate YAML syntax using golangci-lint itself (most reliable)
validate_yaml_syntax() {
    local config_file="$1"
    
    log_info "Validating YAML syntax for $config_file using golangci-lint"
    
    # Use golangci-lint config path as YAML validator since it parses the file
    if ! golangci-lint config path -c "$config_file" > /dev/null 2>&1; then
        log_error "YAML syntax or structure validation failed for $config_file"
        log_error "golangci-lint cannot parse the configuration file"
        return 1
    fi
    
    log_success "YAML syntax and structure are valid for $config_file"
    return 0
}

# Validate golangci-lint configuration
validate_golangci_config() {
    local config_file="$1"
    
    log_info "Validating golangci-lint configuration: $config_file"
    
    # Check if golangci-lint can parse the config
    if ! golangci-lint config path -c "$config_file" > /dev/null 2>&1; then
        log_error "golangci-lint cannot parse configuration file: $config_file"
        log_error "Run 'golangci-lint config path -c $config_file' to see the error"
        return 1
    fi
    
    # Check if linters are valid
    log_info "Checking linter configuration..."
    if ! golangci-lint linters -c "$config_file" > /dev/null 2>&1; then
        log_error "Invalid linter configuration in: $config_file"
        log_error "Run 'golangci-lint linters -c $config_file' to see available linters"
        return 1
    fi
    
    # Check for common configuration issues
    log_info "Checking for common configuration issues..."
    
    # Check for deprecated linters (as of golangci-lint v1.64+)
    local deprecated_linters=(
        "golint"
        "interfacer"
        "maligned"
        "scopelint"
    )
    
    for linter in "${deprecated_linters[@]}"; do
        if grep -q "- $linter" "$config_file" 2>/dev/null; then
            log_warning "Deprecated linter found: $linter (consider removing or replacing)"
        fi
    done
    
    # Check for reasonable timeout values
    local timeout_line
    timeout_line=$(grep "timeout:" "$config_file" 2>/dev/null || true)
    if [[ -n "$timeout_line" ]]; then
        local timeout_value
        timeout_value=$(echo "$timeout_line" | sed 's/.*timeout: *\([0-9]\+[a-z]*\).*/\1/')
        log_info "Configured timeout: $timeout_value"
        
        # Warn if timeout is very high
        if [[ "$timeout_value" =~ ([0-9]+)h ]] && [[ ${BASH_REMATCH[1]} -gt 1 ]]; then
            log_warning "Very high timeout configured: $timeout_value (consider reducing for CI efficiency)"
        fi
    fi
    
    # Check Go version compatibility
    local go_version_line
    go_version_line=$(grep "go:" "$config_file" 2>/dev/null || true)
    if [[ -n "$go_version_line" ]]; then
        local go_version
        go_version=$(echo "$go_version_line" | sed 's/.*go: *["\x27]*\([0-9]\+\.[0-9]\+\)["\x27]*.*/\1/')
        log_info "Configured Go version: $go_version"
        
        # Check if it's a reasonable Go version
        if command -v go &> /dev/null; then
            local current_go_version
            current_go_version=$(go version | sed 's/.*go\([0-9]\+\.[0-9]\+\).*/\1/')
            log_info "Current Go version: $current_go_version"
            
            # Simple version comparison (assumes format x.y)
            local config_major config_minor current_major current_minor
            config_major=$(echo "$go_version" | cut -d. -f1)
            config_minor=$(echo "$go_version" | cut -d. -f2)
            current_major=$(echo "$current_go_version" | cut -d. -f1)
            current_minor=$(echo "$current_go_version" | cut -d. -f2)
            
            if [[ $config_major -gt $current_major ]] || [[ $config_major -eq $current_major && $config_minor -gt $current_minor ]]; then
                log_warning "Configured Go version ($go_version) is newer than current version ($current_go_version)"
            fi
        fi
    fi
    
    log_success "golangci-lint configuration is valid: $config_file"
    return 0
}

# Test configuration with a dry run
test_config_dry_run() {
    local config_file="$1"
    
    log_info "Testing configuration with dry run..."
    
    # Create a temporary test file if none exists
    local test_file=""
    local cleanup_test_file=false
    
    # Look for existing Go files to test with
    if find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | head -1 | read -r existing_file; then
        test_file="$existing_file"
        log_info "Using existing Go file for testing: $test_file"
    else
        # Create a minimal test file
        test_file="./tmp_test_file_for_linting.go"
        cleanup_test_file=true
        cat > "$test_file" << 'EOF'
package main

import "fmt"

func main() {
    fmt.Println("test file for linting")
}
EOF
        log_info "Created temporary test file: $test_file"
    fi
    
    # Run golangci-lint with the config
    local lint_output
    local lint_exit_code=0
    
    log_info "Running golangci-lint dry run with config: $config_file"
    lint_output=$(golangci-lint run --config="$config_file" --timeout=30s "$test_file" 2>&1) || lint_exit_code=$?
    
    # Clean up temporary file if created
    if [[ "$cleanup_test_file" = true ]]; then
        rm -f "$test_file"
        log_info "Cleaned up temporary test file"
    fi
    
    # Check results
    if [[ $lint_exit_code -eq 0 ]]; then
        log_success "Dry run completed successfully"
    elif [[ $lint_exit_code -eq 1 ]]; then
        # Exit code 1 means issues found, which is normal
        log_info "Dry run completed (issues found, which is normal for testing)"
        log_info "Sample output: $(echo "$lint_output" | head -3 | tr '\n' ' ')"
    else
        log_error "Dry run failed with exit code $lint_exit_code"
        log_error "Output: $lint_output"
        return 1
    fi
    
    return 0
}

# Main validation function
main() {
    local exit_code=0
    
    log_info "Starting golangci-lint configuration validation"
    log_info "Project root: $PROJECT_ROOT"
    
    cd "$PROJECT_ROOT"
    
    # Check prerequisites
    if ! check_golangci_lint; then
        log_error "Cannot proceed without golangci-lint"
        exit 1
    fi
    
    # If specific files are provided via arguments, validate them
    if [[ $# -gt 0 ]]; then
        for config_file in "$@"; do
            log_info "Processing file: $config_file"
            
            if [[ ! -f "$config_file" ]]; then
                log_error "File not found: $config_file"
                exit_code=1
                continue
            fi
            
            # Run all validations
            if ! validate_yaml_syntax "$config_file"; then
                exit_code=1
                continue
            fi
            
            if ! validate_golangci_config "$config_file"; then
                exit_code=1
                continue
            fi
            
            if ! test_config_dry_run "$config_file"; then
                exit_code=1
                continue
            fi
            
            log_success "All validations passed for: $config_file"
        done
    else
        # No files specified, find golangci config files
        log_info "No files specified, searching for golangci-lint config files..."
        
        local config_files=()
        while IFS= read -r -d '' file; do
            config_files+=("$file")
        done < <(find . -maxdepth 1 -name ".golangci*.yml" -o -name ".golangci*.yaml" -print0)
        
        if [[ ${#config_files[@]} -eq 0 ]]; then
            log_warning "No golangci-lint config files found in current directory"
            exit 0
        fi
        
        for config_file in "${config_files[@]}"; do
            log_info "Processing found file: $config_file"
            
            # Run all validations
            if ! validate_yaml_syntax "$config_file"; then
                exit_code=1
                continue
            fi
            
            if ! validate_golangci_config "$config_file"; then
                exit_code=1
                continue
            fi
            
            if ! test_config_dry_run "$config_file"; then
                exit_code=1
                continue
            fi
            
            log_success "All validations passed for: $config_file"
        done
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        log_success "All golangci-lint configurations are valid!"
    else
        log_error "Some golangci-lint configurations have issues"
    fi
    
    exit $exit_code
}

# Run main function with all arguments
main "$@"