#!/bin/bash

# validate-golangci-config.sh
# Validates golangci-lint configuration files to prevent CI failures

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
GOLANGCI_LINT_VERSION="v1.63.4"

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

# Check if golangci-lint is installed
check_golangci_lint() {
    if ! command -v golangci-lint &> /dev/null; then
        log_error "golangci-lint not found. Installing version $GOLANGCI_LINT_VERSION..."
        
        # Install golangci-lint
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin $GOLANGCI_LINT_VERSION
        
        if ! command -v golangci-lint &> /dev/null; then
            log_error "Failed to install golangci-lint"
            exit 1
        fi
    fi
    
    local version
    version=$(golangci-lint version 2>/dev/null || echo "unknown")
    log_info "Using golangci-lint: $version"
}

# Find all golangci-lint configuration files
find_config_files() {
    local config_files=()
    
    # Common config file patterns
    local patterns=(
        ".golangci.yml"
        ".golangci.yaml" 
        ".golangci.toml"
        ".golangci.json"
        "golangci.yml"
        "golangci.yaml"
        "golangci.toml"
        "golangci.json"
    )
    
    log_info "Searching for golangci-lint configuration files..."
    
    for pattern in "${patterns[@]}"; do
        while IFS= read -r -d '' file; do
            config_files+=("$file")
        done < <(find "$PROJECT_ROOT" -name "$pattern" -type f -print0 2>/dev/null)
    done
    
    if [ ${#config_files[@]} -eq 0 ]; then
        log_warning "No golangci-lint configuration files found"
        return 1
    fi
    
    printf '%s\n' "${config_files[@]}"
}

# Validate a single configuration file
validate_config() {
    local config_file="$1"
    local relative_path
    relative_path=$(realpath --relative-to="$PROJECT_ROOT" "$config_file")
    
    log_info "Validating: $relative_path"
    
    # Check if file is readable
    if [ ! -r "$config_file" ]; then
        log_error "Cannot read config file: $relative_path"
        return 1
    fi
    
    # Get directory containing the config file
    local config_dir
    config_dir=$(dirname "$config_file")
    
    # Test configuration by running golangci-lint with dry-run
    local validation_output
    local validation_exit_code
    
    cd "$config_dir"
    
    # Try to validate configuration syntax
    if validation_output=$(golangci-lint config verify --config="$config_file" 2>&1); then
        log_success "✓ Configuration syntax valid: $relative_path"
    else
        validation_exit_code=$?
        log_error "✗ Configuration syntax invalid: $relative_path"
        echo "Error details:"
        echo "$validation_output" | sed 's/^/  /'
        cd "$PROJECT_ROOT"
        return $validation_exit_code
    fi
    
    # Test if linters can be loaded (dry run)
    if validation_output=$(golangci-lint linters --config="$config_file" 2>&1 >/dev/null); then
        log_success "✓ Linters load successfully: $relative_path"
    else
        log_warning "⚠ Some linters may have issues: $relative_path"
        echo "Warning details:"
        echo "$validation_output" | sed 's/^/  /'
    fi
    
    cd "$PROJECT_ROOT"
    return 0
}

# Generate validation report
generate_report() {
    local total_configs="$1"
    local valid_configs="$2"
    local invalid_configs="$3"
    
    echo
    echo "=================================="
    echo "  GOLANGCI-LINT CONFIG VALIDATION"
    echo "=================================="
    echo "Total configurations found: $total_configs"
    echo "Valid configurations: $valid_configs"
    echo "Invalid configurations: $invalid_configs"
    echo
    
    if [ "$invalid_configs" -eq 0 ]; then
        log_success "All golangci-lint configurations are valid! ✨"
        return 0
    else
        log_error "Found $invalid_configs invalid configuration(s). Please fix before proceeding."
        return 1
    fi
}

# Main validation function
main() {
    log_info "Starting golangci-lint configuration validation..."
    echo
    
    # Check dependencies
    check_golangci_lint
    echo
    
    # Find configuration files
    local config_files
    if ! config_files=$(find_config_files); then
        log_warning "No configuration files to validate"
        exit 0
    fi
    
    echo "Found configuration files:"
    while IFS= read -r file; do
        echo "  - $(realpath --relative-to="$PROJECT_ROOT" "$file")"
    done <<< "$config_files"
    echo
    
    # Validate each configuration
    local total_configs=0
    local valid_configs=0
    local invalid_configs=0
    local failed_files=()
    
    while IFS= read -r config_file; do
        ((total_configs++))
        
        if validate_config "$config_file"; then
            ((valid_configs++))
        else
            ((invalid_configs++))
            failed_files+=("$(realpath --relative-to="$PROJECT_ROOT" "$config_file")")
        fi
        echo
    done <<< "$config_files"
    
    # Generate report
    generate_report "$total_configs" "$valid_configs" "$invalid_configs"
    local report_exit_code=$?
    
    # List failed files if any
    if [ ${#failed_files[@]} -gt 0 ]; then
        echo
        log_error "Failed configuration files:"
        for file in "${failed_files[@]}"; do
            echo "  ✗ $file"
        done
    fi
    
    exit $report_exit_code
}

# Handle script arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [--help]"
        echo
        echo "Validates all golangci-lint configuration files in the project."
        echo
        echo "Options:"
        echo "  --help, -h    Show this help message"
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac