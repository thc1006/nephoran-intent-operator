#!/bin/bash

# setup-enhanced-cicd.sh
# Comprehensive setup script for Enhanced CI/CD Pipeline
# Author: Claude Code
# Version: 1.0.0

set -e

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
    echo -e "${PURPLE}================================${NC}"
    echo -e "${PURPLE}$1${NC}"
    echo -e "${PURPLE}================================${NC}"
}

# Configuration variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TEMP_DIR="/tmp/nephoran-cicd-setup-$$"

# Default values
SKIP_DEPS=false
SKIP_TOOLS=false
SKIP_SECRETS=false
DRY_RUN=false
VERBOSE=false

# Usage function
usage() {
    cat << EOF
Enhanced CI/CD Pipeline Setup Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    -v, --verbose           Enable verbose output
    -n, --dry-run           Show what would be done without executing
    --skip-deps             Skip dependency installation
    --skip-tools            Skip build tools installation
    --skip-secrets          Skip secrets setup
    --check-only            Only check current setup status

EXAMPLES:
    $0                      # Full setup
    $0 --check-only         # Check current status
    $0 --skip-deps          # Skip dependency installation
    $0 --dry-run            # Preview actions without executing

PREREQUISITES:
    - Git repository with GitHub remote
    - GitHub CLI (gh) installed and authenticated
    - Docker installed and running
    - kubectl installed
    - Go 1.21+ installed

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                exit 0
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -n|--dry-run)
                DRY_RUN=true
                shift
                ;;
            --skip-deps)
                SKIP_DEPS=true
                shift
                ;;
            --skip-tools)
                SKIP_TOOLS=true
                shift
                ;;
            --skip-secrets)
                SKIP_SECRETS=true
                shift
                ;;
            --check-only)
                CHECK_ONLY=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Check prerequisites
check_prerequisites() {
    log_header "Checking Prerequisites"
    
    local missing_deps=()
    
    # Check required commands
    local required_commands=("git" "docker" "kubectl" "go" "gh")
    
    for cmd in "${required_commands[@]}"; do
        if ! command -v "$cmd" &> /dev/null; then
            missing_deps+=("$cmd")
            log_error "$cmd is not installed or not in PATH"
        else
            local version=$($cmd version 2>/dev/null | head -1 || echo "unknown")
            log_success "$cmd is available: $version"
        fi
    done
    
    # Check Docker daemon
    if command -v docker &> /dev/null; then
        if ! docker info &> /dev/null; then
            log_error "Docker daemon is not running"
            missing_deps+=("docker-daemon")
        else
            log_success "Docker daemon is running"
        fi
    fi
    
    # Check GitHub CLI authentication
    if command -v gh &> /dev/null; then
        if ! gh auth status &> /dev/null; then
            log_error "GitHub CLI is not authenticated. Run: gh auth login"
            missing_deps+=("gh-auth")
        else
            log_success "GitHub CLI is authenticated"
        fi
    fi
    
    # Check if we're in a Git repository
    if ! git rev-parse --git-dir &> /dev/null; then
        log_error "Not in a Git repository"
        missing_deps+=("git-repo")
    else
        local repo_url=$(git remote get-url origin 2>/dev/null || echo "no-remote")
        log_success "Git repository detected: $repo_url"
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing prerequisites: ${missing_deps[*]}"
        log_info "Please install missing dependencies and run the script again"
        exit 1
    fi
    
    log_success "All prerequisites satisfied"
}

# Check current setup status
check_current_status() {
    log_header "Checking Current Setup Status"
    
    # Check if enhanced workflow exists
    local workflow_file="$PROJECT_ROOT/.github/workflows/enhanced-ci-cd.yaml"
    if [[ -f "$workflow_file" ]]; then
        log_success "Enhanced CI/CD workflow exists"
    else
        log_warning "Enhanced CI/CD workflow not found"
    fi
    
    # Check secrets
    log_info "Checking GitHub secrets..."
    local required_secrets=("DOCKERHUB_USERNAME" "DOCKERHUB_TOKEN" "GCR_JSON_KEY" "KUBE_CONFIG_STAGING" "KUBE_CONFIG_PRODUCTION")
    local missing_secrets=()
    
    for secret in "${required_secrets[@]}"; do
        if gh secret list | grep -q "$secret"; then
            log_success "Secret $secret is configured"
        else
            log_warning "Secret $secret is missing"
            missing_secrets+=("$secret")
        fi
    done
    
    # Check optional secrets
    local optional_secrets=("COSIGN_PRIVATE_KEY" "COSIGN_PASSWORD" "SLACK_WEBHOOK_URL")
    for secret in "${optional_secrets[@]}"; do
        if gh secret list | grep -q "$secret"; then
            log_success "Optional secret $secret is configured"
        else
            log_info "Optional secret $secret is not configured"
        fi
    done
    
    # Check build tools
    log_info "Checking build tools..."
    local build_tools=("cosign" "syft" "grype")
    for tool in "${build_tools[@]}"; do
        if command -v "$tool" &> /dev/null; then
            local version=$($tool version 2>/dev/null | head -1 || echo "unknown")
            log_success "$tool is available: $version"
        else
            log_warning "$tool is not installed"
        fi
    done
    
    # Check Kustomize overlays
    local staging_overlay="$PROJECT_ROOT/deployments/kustomize/overlays/staging"
    local production_overlay="$PROJECT_ROOT/deployments/kustomize/overlays/production"
    
    if [[ -d "$staging_overlay" ]]; then
        log_success "Staging Kustomize overlay exists"
    else
        log_warning "Staging Kustomize overlay not found"
    fi
    
    if [[ -d "$production_overlay" ]]; then
        log_success "Production Kustomize overlay exists"
    else
        log_warning "Production Kustomize overlay not found"
    fi
    
    if [[ ${#missing_secrets[@]} -gt 0 ]]; then
        log_warning "Missing required secrets: ${missing_secrets[*]}"
        log_info "Run with --skip-secrets to continue without setting up secrets"
    fi
}

# Install dependencies
install_dependencies() {
    if [[ "$SKIP_DEPS" == "true" ]]; then
        log_info "Skipping dependency installation"
        return
    fi
    
    log_header "Installing Dependencies"
    
    # Install Go dependencies
    log_info "Installing Go dependencies..."
    if [[ "$DRY_RUN" == "false" ]]; then
        cd "$PROJECT_ROOT"
        go mod download
        go mod tidy
        log_success "Go dependencies installed"
    else
        log_info "[DRY RUN] Would install Go dependencies"
    fi
    
    # Install development tools
    log_info "Installing development tools..."
    if [[ "$DRY_RUN" == "false" ]]; then
        go install golang.org/x/tools/cmd/goimports@latest
        go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
        go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
        log_success "Development tools installed"
    else
        log_info "[DRY RUN] Would install development tools"
    fi
}

# Install build tools
install_build_tools() {
    if [[ "$SKIP_TOOLS" == "true" ]]; then
        log_info "Skipping build tools installation"
        return
    fi
    
    log_header "Installing Build Tools"
    
    if [[ "$DRY_RUN" == "false" ]]; then
        cd "$PROJECT_ROOT"
        make install-tools
        make verify-tools
        log_success "Build tools installed and verified"
    else
        log_info "[DRY RUN] Would install build tools using 'make install-tools'"
    fi
}

# Setup Docker Buildx
setup_buildx() {
    log_header "Setting up Docker Buildx"
    
    if [[ "$DRY_RUN" == "false" ]]; then
        cd "$PROJECT_ROOT"
        make setup-buildx
        log_success "Docker Buildx configured"
    else
        log_info "[DRY RUN] Would setup Docker Buildx"
    fi
}

# Interactive secrets setup
setup_secrets() {
    if [[ "$SKIP_SECRETS" == "true" ]]; then
        log_info "Skipping secrets setup"
        return
    fi
    
    log_header "Setting up GitHub Secrets"
    
    log_info "This will guide you through setting up required secrets."
    log_warning "You'll need to provide sensitive information. Ensure you're in a secure environment."
    
    # Docker Hub credentials
    log_info "Setting up Docker Hub credentials..."
    read -p "Docker Hub Username: " DOCKERHUB_USERNAME
    read -s -p "Docker Hub Token: " DOCKERHUB_TOKEN
    echo
    
    if [[ "$DRY_RUN" == "false" ]]; then
        gh secret set DOCKERHUB_USERNAME --body "$DOCKERHUB_USERNAME"
        gh secret set DOCKERHUB_TOKEN --body "$DOCKERHUB_TOKEN"
        log_success "Docker Hub secrets configured"
    else
        log_info "[DRY RUN] Would set Docker Hub secrets"
    fi
    
    # GCR service account
    log_info "Setting up Google Container Registry access..."
    read -p "Path to GCR service account JSON file: " GCR_KEY_PATH
    
    if [[ -f "$GCR_KEY_PATH" ]]; then
        if [[ "$DRY_RUN" == "false" ]]; then
            GCR_JSON_KEY=$(cat "$GCR_KEY_PATH" | base64 -w 0)
            gh secret set GCR_JSON_KEY --body "$GCR_JSON_KEY"
            log_success "GCR secret configured"
        else
            log_info "[DRY RUN] Would set GCR secret from $GCR_KEY_PATH"
        fi
    else
        log_warning "GCR key file not found: $GCR_KEY_PATH"
        log_info "You can set this secret later using: gh secret set GCR_JSON_KEY --body \"\$(cat key.json | base64 -w 0)\""
    fi
    
    # Kubernetes configs
    log_info "Setting up Kubernetes cluster access..."
    
    read -p "Path to staging kubeconfig: " STAGING_CONFIG_PATH
    if [[ -f "$STAGING_CONFIG_PATH" ]]; then
        if [[ "$DRY_RUN" == "false" ]]; then
            KUBE_CONFIG_STAGING=$(cat "$STAGING_CONFIG_PATH" | base64 -w 0)
            gh secret set KUBE_CONFIG_STAGING --body "$KUBE_CONFIG_STAGING"
            log_success "Staging kubeconfig secret configured"
        else
            log_info "[DRY RUN] Would set staging kubeconfig secret"
        fi
    else
        log_warning "Staging kubeconfig not found: $STAGING_CONFIG_PATH"
    fi
    
    read -p "Path to production kubeconfig: " PRODUCTION_CONFIG_PATH
    if [[ -f "$PRODUCTION_CONFIG_PATH" ]]; then
        if [[ "$DRY_RUN" == "false" ]]; then
            KUBE_CONFIG_PRODUCTION=$(cat "$PRODUCTION_CONFIG_PATH" | base64 -w 0)
            gh secret set KUBE_CONFIG_PRODUCTION --body "$KUBE_CONFIG_PRODUCTION"
            log_success "Production kubeconfig secret configured"
        else
            log_info "[DRY RUN] Would set production kubeconfig secret"
        fi
    else
        log_warning "Production kubeconfig not found: $PRODUCTION_CONFIG_PATH"
    fi
    
    # Optional: Cosign key
    log_info "Setting up image signing (optional)..."
    read -p "Path to Cosign private key (press Enter to skip): " COSIGN_KEY_PATH
    
    if [[ -n "$COSIGN_KEY_PATH" && -f "$COSIGN_KEY_PATH" ]]; then
        read -s -p "Cosign private key password: " COSIGN_PASSWORD
        echo
        
        if [[ "$DRY_RUN" == "false" ]]; then
            COSIGN_PRIVATE_KEY=$(cat "$COSIGN_KEY_PATH")
            gh secret set COSIGN_PRIVATE_KEY --body "$COSIGN_PRIVATE_KEY"
            gh secret set COSIGN_PASSWORD --body "$COSIGN_PASSWORD"
            log_success "Cosign secrets configured"
        else
            log_info "[DRY RUN] Would set Cosign secrets"
        fi
    elif [[ -n "$COSIGN_KEY_PATH" ]]; then
        log_warning "Cosign key file not found: $COSIGN_KEY_PATH"
    else
        log_info "Skipping Cosign setup - keyless signing will be used"
    fi
    
    # Optional: Slack webhook
    log_info "Setting up Slack notifications (optional)..."
    read -p "Slack webhook URL (press Enter to skip): " SLACK_WEBHOOK_URL
    
    if [[ -n "$SLACK_WEBHOOK_URL" ]]; then
        if [[ "$DRY_RUN" == "false" ]]; then
            gh secret set SLACK_WEBHOOK_URL --body "$SLACK_WEBHOOK_URL"
            log_success "Slack webhook configured"
        else
            log_info "[DRY RUN] Would set Slack webhook secret"
        fi
    else
        log_info "Skipping Slack webhook setup"
    fi
}

# Test the setup
test_setup() {
    log_header "Testing Setup"
    
    # Test build system
    log_info "Testing build system..."
    if [[ "$DRY_RUN" == "false" ]]; then
        cd "$PROJECT_ROOT"
        
        # Test basic build
        if make build-all &> /dev/null; then
            log_success "Build system works"
        else
            log_warning "Build system test failed - check dependencies"
        fi
        
        # Test enhanced build tools
        if make verify-tools &> /dev/null; then
            log_success "Enhanced build tools work"
        else
            log_warning "Enhanced build tools test failed"
        fi
        
    else
        log_info "[DRY RUN] Would test build system"
    fi
    
    # Test secrets
    log_info "Testing secrets access..."
    if [[ "$DRY_RUN" == "false" ]]; then
        local secret_count=$(gh secret list | wc -l)
        log_info "Found $secret_count configured secrets"
        
        # Test workflow exists and is valid
        if [[ -f "$PROJECT_ROOT/.github/workflows/enhanced-ci-cd.yaml" ]]; then
            log_success "Enhanced CI/CD workflow is in place"
        else
            log_warning "Enhanced CI/CD workflow not found"
        fi
    else
        log_info "[DRY RUN] Would test secrets access"
    fi
}

# Generate setup report
generate_report() {
    log_header "Setup Report"
    
    local report_file="$PROJECT_ROOT/SETUP-REPORT-$(date +%Y%m%d-%H%M%S).md"
    
    cat > "$report_file" << EOF
# Enhanced CI/CD Setup Report

Generated on: $(date)
Setup script version: 1.0.0

## Prerequisites Status
- Git: $(git --version 2>/dev/null || echo "Not found")
- Docker: $(docker --version 2>/dev/null || echo "Not found")  
- kubectl: $(kubectl version --client --short 2>/dev/null || echo "Not found")
- Go: $(go version 2>/dev/null || echo "Not found")
- GitHub CLI: $(gh --version 2>/dev/null || echo "Not found")

## Workflow Configuration
- Enhanced CI/CD workflow: $([ -f "$PROJECT_ROOT/.github/workflows/enhanced-ci-cd.yaml" ] && echo "✅ Configured" || echo "❌ Missing")
- Staging Kustomize overlay: $([ -d "$PROJECT_ROOT/deployments/kustomize/overlays/staging" ] && echo "✅ Configured" || echo "❌ Missing")
- Production Kustomize overlay: $([ -d "$PROJECT_ROOT/deployments/kustomize/overlays/production" ] && echo "✅ Configured" || echo "❌ Missing")

## Build Tools Status
- Cosign: $(cosign version 2>/dev/null | head -1 || echo "Not installed")
- Syft: $(syft version 2>/dev/null | head -1 || echo "Not installed")
- Grype: $(grype version 2>/dev/null | head -1 || echo "Not installed")

## GitHub Secrets Status
$(gh secret list 2>/dev/null | sed 's/^/- /' || echo "Unable to check secrets")

## Next Steps

1. **Test the Pipeline**: Create a test branch and push to verify the workflow
2. **Configure Monitoring**: Set up monitoring for pipeline health
3. **Security Review**: Review and validate all security configurations
4. **Documentation**: Update team documentation with new procedures

## Support

- Setup Documentation: .github/workflows/ENHANCED-CICD-SETUP.md
- Secrets Management: .github/workflows/SECRETS-MANAGEMENT.md
- Troubleshooting: Check workflow logs in GitHub Actions

EOF

    log_success "Setup report generated: $report_file"
}

# Cleanup function
cleanup() {
    if [[ -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
    fi
}

# Main execution function
main() {
    log_header "Enhanced CI/CD Pipeline Setup"
    log_info "Starting setup process..."
    
    # Create temp directory
    mkdir -p "$TEMP_DIR"
    trap cleanup EXIT
    
    # Parse arguments
    parse_args "$@"
    
    # Check prerequisites
    check_prerequisites
    
    # Check current status
    check_current_status
    
    # Exit if check-only mode
    if [[ "$CHECK_ONLY" == "true" ]]; then
        log_info "Check-only mode completed"
        exit 0
    fi
    
    # Main setup steps
    install_dependencies
    install_build_tools
    setup_buildx
    setup_secrets
    
    # Test the setup
    test_setup
    
    # Generate report
    generate_report
    
    log_header "Setup Complete!"
    
    log_success "Enhanced CI/CD pipeline has been configured successfully!"
    echo
    log_info "Next steps:"
    echo "  1. Review the generated setup report"
    echo "  2. Test the pipeline by creating a pull request"
    echo "  3. Monitor the first workflow run in GitHub Actions"
    echo "  4. Configure any additional monitoring or alerting"
    echo
    log_info "Documentation:"
    echo "  - Setup Guide: .github/workflows/ENHANCED-CICD-SETUP.md"
    echo "  - Secrets Management: .github/workflows/SECRETS-MANAGEMENT.md"
    echo
    log_warning "Remember to:"
    echo "  - Keep secrets secure and rotate them regularly"
    echo "  - Monitor pipeline health and performance"
    echo "  - Update the pipeline as your requirements evolve"
}

# Execute main function with all arguments
main "$@"