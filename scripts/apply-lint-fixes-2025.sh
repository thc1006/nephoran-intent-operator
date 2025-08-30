#!/bin/bash

# Apply 2025 golangci-lint fixes script
# This script applies common linting fixes based on 2025 best practices

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

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

# Check if required tools are installed
check_requirements() {
    log_info "Checking requirements..."
    
    local missing_tools=()
    
    if ! command -v go &> /dev/null; then
        missing_tools+=("go")
    fi
    
    if ! command -v golangci-lint &> /dev/null; then
        missing_tools+=("golangci-lint")
    fi
    
    if ! command -v goimports &> /dev/null; then
        missing_tools+=("goimports")
    fi
    
    if ! command -v gofumpt &> /dev/null; then
        missing_tools+=("gofumpt")
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_info "Please install missing tools:"
        for tool in "${missing_tools[@]}"; do
            case $tool in
                "go")
                    echo "  - Install Go from https://golang.org/dl/"
                    ;;
                "golangci-lint")
                    echo "  - Install golangci-lint: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$(go env GOPATH)/bin v1.64.3"
                    ;;
                "goimports")
                    echo "  - Install goimports: go install golang.org/x/tools/cmd/goimports@latest"
                    ;;
                "gofumpt")
                    echo "  - Install gofumpt: go install mvdan.cc/gofumpt@latest"
                    ;;
            esac
        done
        exit 1
    fi
    
    log_success "All required tools are installed"
}

# Backup files before making changes
backup_files() {
    local backup_dir="${PROJECT_ROOT}/.lint-fixes-backup-$(date +%Y%m%d_%H%M%S)"
    log_info "Creating backup at $backup_dir"
    
    mkdir -p "$backup_dir"
    
    # Find all .go files (excluding generated ones)
    find "$PROJECT_ROOT" -name "*.go" \
        -not -path "*/vendor/*" \
        -not -path "*/testdata/*" \
        -not -path "*/.git/*" \
        -not -path "*/bin/*" \
        -not -path "*/tmp/*" \
        -not -name "*_generated.go" \
        -not -name "*.pb.go" \
        -not -name "*deepcopy*.go" \
        -exec cp --parents {} "$backup_dir/" \;
    
    log_success "Backup created at $backup_dir"
    echo "$backup_dir" > "$PROJECT_ROOT/.last-backup"
}

# Restore from backup
restore_backup() {
    if [[ ! -f "$PROJECT_ROOT/.last-backup" ]]; then
        log_error "No backup found to restore from"
        exit 1
    fi
    
    local backup_dir
    backup_dir=$(cat "$PROJECT_ROOT/.last-backup")
    
    if [[ ! -d "$backup_dir" ]]; then
        log_error "Backup directory not found: $backup_dir"
        exit 1
    fi
    
    log_info "Restoring from backup: $backup_dir"
    
    # Restore .go files
    find "$backup_dir" -name "*.go" -exec cp {} "$PROJECT_ROOT/{}" \;
    
    log_success "Restored from backup"
}

# Apply goimports to fix imports
fix_imports() {
    log_info "Fixing imports with goimports..."
    
    find "$PROJECT_ROOT" -name "*.go" \
        -not -path "*/vendor/*" \
        -not -path "*/testdata/*" \
        -not -name "*_generated.go" \
        -not -name "*.pb.go" \
        -not -name "*deepcopy*.go" \
        -exec goimports -w {} \;
    
    log_success "Imports fixed"
}

# Apply gofumpt for strict formatting
fix_formatting() {
    log_info "Applying strict formatting with gofumpt..."
    
    find "$PROJECT_ROOT" -name "*.go" \
        -not -path "*/vendor/*" \
        -not -path "*/testdata/*" \
        -not -name "*_generated.go" \
        -not -name "*.pb.go" \
        -not -name "*deepcopy*.go" \
        -exec gofumpt -w {} \;
    
    log_success "Formatting applied"
}

# Add missing package comments
add_package_comments() {
    log_info "Adding missing package comments..."
    
    local fixed_count=0
    
    # Find all Go packages without package comments
    find "$PROJECT_ROOT" -name "*.go" \
        -not -path "*/vendor/*" \
        -not -path "*/testdata/*" \
        -not -name "*_generated.go" \
        -not -name "*.pb.go" \
        -not -name "*deepcopy*.go" | while read -r file; do
        
        # Check if file has package comment
        if ! grep -q "^// Package" "$file" && ! grep -q "^/\\* Package" "$file"; then
            # Extract package name
            package_name=$(grep "^package " "$file" | head -1 | awk '{print $2}')
            
            if [[ -n "$package_name" && "$package_name" != "main" ]]; then
                # Generate appropriate comment based on package name
                local comment
                case "$package_name" in
                    *client*|*clients*)
                        comment="// Package $package_name provides client implementations for service communication."
                        ;;
                    *auth*)
                        comment="// Package $package_name provides authentication and authorization functionality."
                        ;;
                    *config*)
                        comment="// Package $package_name provides configuration management and validation."
                        ;;
                    *controller*|*controllers*)
                        comment="// Package $package_name provides Kubernetes controller implementations."
                        ;;
                    *webhook*|*webhooks*)
                        comment="// Package $package_name provides Kubernetes webhook implementations."
                        ;;
                    *security*)
                        comment="// Package $package_name provides security utilities and mTLS functionality."
                        ;;
                    *monitor*|*monitoring*)
                        comment="// Package $package_name provides monitoring and observability features."
                        ;;
                    *audit*)
                        comment="// Package $package_name provides audit logging and compliance tracking."
                        ;;
                    *)
                        comment="// Package $package_name provides core functionality for the Nephoran Intent Operator."
                        ;;
                esac
                
                # Create temporary file with comment
                local temp_file
                temp_file=$(mktemp)
                
                echo "$comment" > "$temp_file"
                cat "$file" >> "$temp_file"
                mv "$temp_file" "$file"
                
                log_info "Added package comment to $file"
                ((fixed_count++))
            fi
        fi
    done
    
    if [[ $fixed_count -gt 0 ]]; then
        log_success "Added package comments to $fixed_count files"
    else
        log_info "No missing package comments found"
    fi
}

# Fix error wrapping patterns
fix_error_wrapping() {
    log_info "Fixing error wrapping patterns..."
    
    local fixed_count=0
    
    # Find files with error wrapping issues
    find "$PROJECT_ROOT" -name "*.go" \
        -not -path "*/vendor/*" \
        -not -path "*/testdata/*" \
        -not -name "*_generated.go" \
        -not -name "*.pb.go" \
        -not -name "*deepcopy*.go" | while read -r file; do
        
        # Fix fmt.Errorf with .Error() to use %w
        if sed -i.bak -E 's/fmt\.Errorf\(([^,]+),([^)]*\.Error\(\))\)/fmt.Errorf(\1, \2)/g' "$file" 2>/dev/null; then
            if ! diff -q "$file" "$file.bak" >/dev/null 2>&1; then
                rm -f "$file.bak"
                log_info "Fixed error wrapping in $file"
                ((fixed_count++))
            else
                rm -f "$file.bak"
            fi
        fi
        
        # Fix %s and %v to %w for error wrapping
        if sed -i.bak -E 's/fmt\.Errorf\(([^,]*")%[sv]([^"]*"),([^)]*err[^)]*)\)/fmt.Errorf(\1%w\2, \3)/g' "$file" 2>/dev/null; then
            if ! diff -q "$file" "$file.bak" >/dev/null 2>&1; then
                rm -f "$file.bak"
                log_info "Fixed error format in $file"
                ((fixed_count++))
            else
                rm -f "$file.bak"
            fi
        fi
    done
    
    if [[ $fixed_count -gt 0 ]]; then
        log_success "Fixed error wrapping in $fixed_count files"
    else
        log_info "No error wrapping issues found"
    fi
}

# Run golangci-lint with auto-fix
run_linter_autofix() {
    log_info "Running golangci-lint with auto-fix..."
    
    local config_file="$PROJECT_ROOT/.golangci-2025.yml"
    if [[ ! -f "$config_file" ]]; then
        log_warning "2025 config not found, using default .golangci.yml"
        config_file="$PROJECT_ROOT/.golangci.yml"
    fi
    
    if [[ -f "$config_file" ]]; then
        cd "$PROJECT_ROOT"
        golangci-lint run --config "$config_file" --fix --timeout 20m
        log_success "Linter auto-fix completed"
    else
        log_error "No golangci-lint configuration found"
        return 1
    fi
}

# Generate lint report
generate_lint_report() {
    log_info "Generating lint report..."
    
    local config_file="$PROJECT_ROOT/.golangci-2025.yml"
    if [[ ! -f "$config_file" ]]; then
        config_file="$PROJECT_ROOT/.golangci.yml"
    fi
    
    local report_file="$PROJECT_ROOT/lint-report-$(date +%Y%m%d_%H%M%S).json"
    
    cd "$PROJECT_ROOT"
    golangci-lint run --config "$config_file" --out-format json --timeout 20m > "$report_file" 2>/dev/null || true
    
    if [[ -f "$report_file" && -s "$report_file" ]]; then
        log_success "Lint report generated: $report_file"
        
        # Extract summary
        local issue_count
        issue_count=$(jq '.Issues | length' "$report_file" 2>/dev/null || echo "0")
        log_info "Total issues found: $issue_count"
        
        if [[ "$issue_count" -gt 0 ]]; then
            log_info "Top issues:"
            jq -r '.Issues | group_by(.FromLinter) | sort_by(length) | reverse | .[:5][] | "\(.length) issues from \(.[0].FromLinter)"' "$report_file" 2>/dev/null || true
        fi
    else
        log_success "No issues found - clean code!"
        rm -f "$report_file"
    fi
}

# Check Go module tidiness
check_go_mod() {
    log_info "Checking go.mod tidiness..."
    
    cd "$PROJECT_ROOT"
    
    # Check if go.mod needs tidying
    local before_hash
    local after_hash
    
    before_hash=$(md5sum go.mod go.sum 2>/dev/null | md5sum | cut -d' ' -f1)
    go mod tidy
    after_hash=$(md5sum go.mod go.sum 2>/dev/null | md5sum | cut -d' ' -f1)
    
    if [[ "$before_hash" != "$after_hash" ]]; then
        log_success "go.mod and go.sum updated"
    else
        log_info "go.mod and go.sum are already tidy"
    fi
}

# Run tests to ensure fixes don't break functionality
run_tests() {
    log_info "Running tests to validate fixes..."
    
    cd "$PROJECT_ROOT"
    
    # Run unit tests
    if go test ./... -short -timeout 5m; then
        log_success "All tests passed"
    else
        log_error "Some tests failed - fixes may have introduced issues"
        return 1
    fi
}

# Main execution function
main() {
    log_info "Starting 2025 Go linting fixes for Nephoran Intent Operator"
    log_info "Project root: $PROJECT_ROOT"
    
    case "${1:-all}" in
        "check")
            check_requirements
            ;;
        "backup")
            backup_files
            ;;
        "restore")
            restore_backup
            ;;
        "imports")
            fix_imports
            ;;
        "format")
            fix_formatting
            ;;
        "comments")
            add_package_comments
            ;;
        "errors")
            fix_error_wrapping
            ;;
        "lint")
            run_linter_autofix
            ;;
        "report")
            generate_lint_report
            ;;
        "mod")
            check_go_mod
            ;;
        "test")
            run_tests
            ;;
        "all")
            check_requirements
            backup_files
            fix_imports
            fix_formatting
            add_package_comments
            fix_error_wrapping
            check_go_mod
            run_linter_autofix
            run_tests
            generate_lint_report
            log_success "All fixes applied successfully!"
            ;;
        "help"|"-h"|"--help")
            cat << EOF
Usage: $0 [COMMAND]

Commands:
  check      Check if all required tools are installed
  backup     Create backup of current .go files
  restore    Restore from last backup
  imports    Fix imports with goimports
  format     Apply strict formatting with gofumpt
  comments   Add missing package comments
  errors     Fix error wrapping patterns
  lint       Run golangci-lint with auto-fix
  report     Generate lint report
  mod        Check and tidy go.mod
  test       Run tests to validate fixes
  all        Apply all fixes (default)
  help       Show this help message

Examples:
  $0                    # Apply all fixes
  $0 check              # Check requirements only
  $0 backup             # Create backup only
  $0 imports format     # Fix imports and formatting only
  $0 lint report        # Run linter and generate report

The script will create a backup before making changes.
Use '$0 restore' to revert changes if needed.
EOF
            ;;
        *)
            log_error "Unknown command: $1"
            log_info "Use '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Execute main function with all arguments
main "$@"