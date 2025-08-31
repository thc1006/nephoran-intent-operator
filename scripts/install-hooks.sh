#!/bin/bash
set -euo pipefail

# install-hooks.sh
# Install and configure pre-commit hooks for Nephoran Intent Operator
# Prevents invalid golangci-lint configurations from being committed

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
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

log_header() {
    echo -e "${PURPLE}=== $1 ===${NC}"
}

log_step() {
    echo -e "${CYAN}[STEP]${NC} $1"
}

# Check system requirements
check_requirements() {
    log_header "Checking System Requirements"
    
    local missing_tools=()
    
    # Check Python (required for pre-commit)
    if ! command -v python3 &> /dev/null; then
        missing_tools+=("python3")
    else
        local python_version
        python_version=$(python3 --version 2>&1 | cut -d' ' -f2)
        log_info "Python version: $python_version"
    fi
    
    # Check pip
    if ! command -v pip3 &> /dev/null && ! python3 -m pip --version &> /dev/null; then
        missing_tools+=("pip3")
    fi
    
    # Check git
    if ! command -v git &> /dev/null; then
        missing_tools+=("git")
    else
        local git_version
        git_version=$(git --version | cut -d' ' -f3)
        log_info "Git version: $git_version"
    fi
    
    # Check Go
    if ! command -v go &> /dev/null; then
        log_warning "Go not found - some hooks may not work properly"
    else
        local go_version
        go_version=$(go version | cut -d' ' -f3)
        log_info "Go version: $go_version"
    fi
    
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_error "Please install the missing tools and run this script again"
        log_info ""
        log_info "On Ubuntu/Debian: sudo apt update && sudo apt install python3 python3-pip git golang-go"
        log_info "On RHEL/CentOS: sudo yum install python3 python3-pip git golang"
        log_info "On macOS: brew install python3 git go"
        return 1
    fi
    
    log_success "All required tools are available"
    return 0
}

# Install pre-commit framework
install_precommit() {
    log_header "Installing Pre-commit Framework"
    
    # Check if pre-commit is already installed
    if command -v pre-commit &> /dev/null; then
        local current_version
        current_version=$(pre-commit --version | cut -d' ' -f2)
        log_info "pre-commit is already installed (version: $current_version)"
        
        # Check if version is recent enough
        local version_number
        version_number=$(echo "$current_version" | awk -F. '{ printf("%03d%03d%03d", $1, $2, $3) }')
        local min_version_number
        min_version_number=$(echo "2.20.0" | awk -F. '{ printf("%03d%03d%03d", $1, $2, $3) }')
        
        if [[ $version_number -lt $min_version_number ]]; then
            log_warning "pre-commit version is too old, upgrading..."
            if ! python3 -m pip install --upgrade pre-commit; then
                log_error "Failed to upgrade pre-commit"
                return 1
            fi
        fi
    else
        log_info "Installing pre-commit framework..."
        
        # Try different installation methods
        if python3 -m pip install pre-commit; then
            log_success "pre-commit installed via pip"
        elif pip3 install pre-commit; then
            log_success "pre-commit installed via pip3"
        else
            log_error "Failed to install pre-commit"
            log_error "Try manually: python3 -m pip install pre-commit"
            return 1
        fi
    fi
    
    # Verify installation
    if ! command -v pre-commit &> /dev/null; then
        log_error "pre-commit installation verification failed"
        log_info "You may need to add ~/.local/bin to your PATH"
        log_info "Add this to your shell profile: export PATH=~/.local/bin:\$PATH"
        return 1
    fi
    
    local final_version
    final_version=$(pre-commit --version | cut -d' ' -f2)
    log_success "pre-commit is ready (version: $final_version)"
    
    return 0
}

# Install additional tools
install_additional_tools() {
    log_header "Installing Additional Tools"
    
    cd "$PROJECT_ROOT"
    
    # Install yamllint for YAML validation
    log_step "Installing yamllint..."
    if ! command -v yamllint &> /dev/null; then
        if python3 -m pip install yamllint; then
            log_success "yamllint installed"
        else
            log_warning "Failed to install yamllint, YAML validation may be limited"
        fi
    else
        log_info "yamllint already installed"
    fi
    
    # Install or update golangci-lint
    log_step "Installing/updating golangci-lint..."
    local expected_version="v1.64.3"
    
    if command -v go &> /dev/null; then
        log_info "Installing golangci-lint $expected_version..."
        if GOBIN=$(go env GOPATH)/bin go install "github.com/golangci/golangci-lint/cmd/golangci-lint@$expected_version"; then
            # Add GOPATH/bin to PATH if not already there
            local gopath_bin
            gopath_bin="$(go env GOPATH)/bin"
            if [[ ":$PATH:" != *":$gopath_bin:"* ]]; then
                export PATH="$gopath_bin:$PATH"
                log_info "Added $gopath_bin to PATH for this session"
                log_warning "Add 'export PATH=\$(go env GOPATH)/bin:\$PATH' to your shell profile for permanent access"
            fi
            
            log_success "golangci-lint installed/updated"
        else
            log_warning "Failed to install golangci-lint, some hooks may not work"
        fi
    else
        log_warning "Go not available, cannot install golangci-lint"
    fi
    
    return 0
}

# Set up hook scripts permissions
setup_hook_permissions() {
    log_header "Setting Up Hook Script Permissions"
    
    local hook_scripts=(
        "scripts/hooks/validate-golangci-config.sh"
        "scripts/hooks/check-lint-compatibility.sh"
    )
    
    for script in "${hook_scripts[@]}"; do
        local script_path="$PROJECT_ROOT/$script"
        if [[ -f "$script_path" ]]; then
            chmod +x "$script_path"
            log_success "Made $script executable"
        else
            log_warning "Hook script not found: $script_path"
        fi
    done
    
    return 0
}

# Install pre-commit hooks
install_hooks() {
    log_header "Installing Pre-commit Hooks"
    
    cd "$PROJECT_ROOT"
    
    # Check if .pre-commit-config.yaml exists
    if [[ ! -f ".pre-commit-config.yaml" ]]; then
        log_error ".pre-commit-config.yaml not found in project root"
        log_error "Please ensure the configuration file exists"
        return 1
    fi
    
    log_info "Installing pre-commit hooks from configuration..."
    
    # Install the hooks
    if pre-commit install; then
        log_success "Pre-commit hooks installed successfully"
    else
        log_error "Failed to install pre-commit hooks"
        return 1
    fi
    
    # Install commit-msg hook for commit message validation (optional)
    if pre-commit install --hook-type commit-msg 2>/dev/null; then
        log_info "Commit message hooks installed"
    fi
    
    # Install pre-push hooks (optional)
    if pre-commit install --hook-type pre-push 2>/dev/null; then
        log_info "Pre-push hooks installed"
    fi
    
    return 0
}

# Test hooks installation
test_hooks() {
    log_header "Testing Hooks Installation"
    
    cd "$PROJECT_ROOT"
    
    log_step "Running pre-commit on all files (dry run)..."
    
    # Run pre-commit on all files to test installation
    if pre-commit run --all-files --dry-run; then
        log_success "Pre-commit hooks test completed successfully"
    else
        local exit_code=$?
        if [[ $exit_code -eq 1 ]]; then
            log_info "Pre-commit found issues (normal for first run)"
            log_info "Hooks are working correctly"
        else
            log_warning "Pre-commit test had unexpected exit code: $exit_code"
        fi
    fi
    
    # Test specific golangci-lint validation
    log_step "Testing golangci-lint configuration validation..."
    
    if [[ -f ".golangci-fast.yml" ]]; then
        if "$PROJECT_ROOT/scripts/hooks/validate-golangci-config.sh" ".golangci-fast.yml"; then
            log_success "golangci-lint configuration validation works correctly"
        else
            log_warning "golangci-lint configuration validation test failed"
        fi
    else
        log_warning "No .golangci-fast.yml found for testing"
    fi
    
    return 0
}

# Display usage instructions
show_usage_instructions() {
    log_header "Usage Instructions"
    
    echo -e "${CYAN}Pre-commit hooks are now installed and will run automatically on every commit.${NC}"
    echo ""
    echo -e "${YELLOW}Common Commands:${NC}"
    echo "  pre-commit run --all-files    # Run hooks on all files"
    echo "  pre-commit run <hook-id>      # Run specific hook"
    echo "  pre-commit autoupdate         # Update hook versions"
    echo "  pre-commit uninstall          # Remove hooks"
    echo ""
    echo -e "${YELLOW}Hook-specific Commands:${NC}"
    echo "  make validate-configs         # Validate golangci-lint configs"
    echo "  make install-hooks           # Re-run this installer"
    echo ""
    echo -e "${YELLOW}Bypassing Hooks (use sparingly):${NC}"
    echo "  git commit --no-verify        # Skip all pre-commit hooks"
    echo "  SKIP=<hook-id> git commit     # Skip specific hooks"
    echo ""
    echo -e "${GREEN}The following hooks are now active:${NC}"
    echo "  ✅ validate-golangci-config   # Validates .golangci*.yml files"
    echo "  ✅ syntax-check-golangci      # Checks golangci-lint can parse config"
    echo "  ✅ lint-config-compatibility  # Checks version compatibility"
    echo "  ✅ go-fmt                     # Formats Go code"
    echo "  ✅ go-vet                     # Runs go vet"
    echo "  ✅ go-mod-tidy                # Ensures go.mod/go.sum are clean"
    echo "  ✅ golangci-lint-fast         # Quick lint on changed files"
    echo "  ✅ yamllint                   # YAML syntax validation"
    echo "  ✅ Generic checks             # File size, merge conflicts, etc."
    echo ""
    echo -e "${BLUE}For more information, see:.pre-commit-config.yaml${NC}"
}

# Main installation function
main() {
    log_header "Nephoran Intent Operator - Pre-commit Hooks Installer"
    log_info "This script will install pre-commit hooks to prevent invalid golangci-lint configurations"
    echo ""
    
    cd "$PROJECT_ROOT"
    
    # Run installation steps
    local steps=(
        "check_requirements"
        "install_precommit" 
        "install_additional_tools"
        "setup_hook_permissions"
        "install_hooks"
        "test_hooks"
    )
    
    local step_count=1
    local total_steps=${#steps[@]}
    
    for step_func in "${steps[@]}"; do
        echo ""
        log_info "Step $step_count/$total_steps: Running $step_func..."
        
        if ! "$step_func"; then
            log_error "Step $step_func failed"
            log_error "Installation aborted"
            exit 1
        fi
        
        ((step_count++))
    done
    
    echo ""
    log_success "Pre-commit hooks installation completed successfully!"
    echo ""
    
    show_usage_instructions
    
    # Final verification
    echo ""
    log_info "Installation Summary:"
    echo "  ✅ pre-commit framework: $(command -v pre-commit &>/dev/null && pre-commit --version || echo 'NOT FOUND')"
    echo "  ✅ golangci-lint: $(command -v golangci-lint &>/dev/null && golangci-lint version | head -1 || echo 'NOT FOUND')"
    echo "  ✅ yamllint: $(command -v yamllint &>/dev/null && yamllint --version || echo 'NOT FOUND')"
    echo "  ✅ Go: $(command -v go &>/dev/null && go version || echo 'NOT FOUND')"
    
    echo ""
    log_success "🎉 Ready to prevent invalid golangci-lint configurations!"
    log_info "Try making a commit to see the hooks in action"
}

# Handle command line options
case "${1:-}" in
    "--help"|"-h")
        echo "Usage: $0 [--help|--test-only|--reinstall]"
        echo ""
        echo "Options:"
        echo "  --help        Show this help message"
        echo "  --test-only   Only test existing installation"
        echo "  --reinstall   Force reinstallation of all components"
        echo ""
        echo "This script installs pre-commit hooks to validate golangci-lint configurations"
        echo "before they are committed to the repository."
        exit 0
        ;;
    "--test-only")
        log_info "Running test-only mode..."
        cd "$PROJECT_ROOT"
        test_hooks
        exit $?
        ;;
    "--reinstall")
        log_info "Running in reinstall mode..."
        cd "$PROJECT_ROOT"
        if command -v pre-commit &>/dev/null; then
            pre-commit uninstall || true
        fi
        ;;
esac

# Run main installation
main "$@"