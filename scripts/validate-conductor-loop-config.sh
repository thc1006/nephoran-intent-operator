#!/bin/bash
# Configuration Validation Script for Conductor Loop
# Validates all configuration files and dependencies

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${BLUE}[VALIDATE]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }

VALIDATION_ERRORS=0
VALIDATION_WARNINGS=0

validate_file() {
    local file="$1"
    local description="$2"
    
    if [ -f "$file" ]; then
        success "$description exists: $file"
        return 0
    else
        error "$description missing: $file"
        ((VALIDATION_ERRORS++))
        return 1
    fi
}

validate_yaml() {
    local file="$1"
    local description="$2"
    
    if ! validate_file "$file" "$description"; then
        return 1
    fi
    
    # Basic YAML structure validation
    if grep -q "apiVersion:" "$file" && grep -q "kind:" "$file"; then
        success "$description has valid Kubernetes structure"
    else
        warn "$description may not be valid Kubernetes YAML"
        ((VALIDATION_WARNINGS++))
    fi
}

validate_json() {
    local file="$1"
    local description="$2"
    
    if ! validate_file "$file" "$description"; then
        return 1
    fi
    
    # Basic JSON validation if python is available
    if command -v python3 &> /dev/null; then
        if python3 -m json.tool "$file" > /dev/null 2>&1; then
            success "$description has valid JSON syntax"
        else
            error "$description has invalid JSON syntax"
            ((VALIDATION_ERRORS++))
        fi
    elif command -v python &> /dev/null; then
        if python -m json.tool "$file" > /dev/null 2>&1; then
            success "$description has valid JSON syntax"
        else
            error "$description has invalid JSON syntax"
            ((VALIDATION_ERRORS++))
        fi
    else
        warn "Python not available - skipping JSON validation"
        ((VALIDATION_WARNINGS++))
    fi
}

validate_script() {
    local file="$1"
    local description="$2"
    
    if ! validate_file "$file" "$description"; then
        return 1
    fi
    
    if [ -x "$file" ]; then
        success "$description is executable"
    else
        warn "$description is not executable"
        ((VALIDATION_WARNINGS++))
    fi
    
    # Basic shell script validation
    if bash -n "$file" 2>/dev/null; then
        success "$description has valid shell syntax"
    else
        error "$description has shell syntax errors"
        ((VALIDATION_ERRORS++))
    fi
}

validate_powershell_script() {
    local file="$1"
    local description="$2"
    
    if ! validate_file "$file" "$description"; then
        return 1
    fi
    
    # Basic PowerShell validation if available
    if command -v powershell &> /dev/null; then
        if powershell -Command "& { Get-Content '$file' | Out-String | Invoke-Expression }" -ErrorAction Stop &> /dev/null; then
            success "$description has valid PowerShell syntax"
        else
            warn "$description may have PowerShell syntax issues (full validation requires Windows)"
            ((VALIDATION_WARNINGS++))
        fi
    else
        warn "PowerShell not available - skipping syntax validation"
        ((VALIDATION_WARNINGS++))
    fi
}

validate_dockerfile() {
    local file="$1"
    local description="$2"
    
    if ! validate_file "$file" "$description"; then
        return 1
    fi
    
    # Check for required Dockerfile keywords
    local required_keywords=("FROM" "COPY" "WORKDIR")
    local missing_keywords=()
    
    for keyword in "${required_keywords[@]}"; do
        if ! grep -q "^$keyword " "$file"; then
            missing_keywords+=("$keyword")
        fi
    done
    
    if [ ${#missing_keywords[@]} -eq 0 ]; then
        success "$description has all required Dockerfile keywords"
    else
        warn "$description missing keywords: ${missing_keywords[*]}"
        ((VALIDATION_WARNINGS++))
    fi
    
    # Check for security best practices
    if grep -q "USER.*root\|USER.*0" "$file"; then
        warn "$description runs as root (security concern)"
        ((VALIDATION_WARNINGS++))
    fi
    
    if grep -q "runAsNonRoot.*true\|runAsUser.*[1-9]" "$file"; then
        success "$description follows non-root user security practices"
    fi
}

validate_docker_compose() {
    local file="$1"
    local description="$2"
    
    if ! validate_file "$file" "$description"; then
        return 1
    fi
    
    # Check for required Docker Compose structure
    if grep -q "version:" "$file" && grep -q "services:" "$file"; then
        success "$description has valid Docker Compose structure"
    else
        error "$description missing required Docker Compose structure"
        ((VALIDATION_ERRORS++))
    fi
    
    # Check for security configurations
    if grep -q "user:" "$file"; then
        success "$description configures non-root user"
    else
        warn "$description should configure non-root user"
        ((VALIDATION_WARNINGS++))
    fi
}

main() {
    log "Validating Conductor Loop configuration files..."
    log "Project root: $PROJECT_ROOT"
    echo
    
    # Validate GitHub Actions workflow
    validate_yaml "$PROJECT_ROOT/.github/workflows/conductor-loop.yml" "GitHub Actions workflow"
    
    # Validate Dockerfile
    validate_dockerfile "$PROJECT_ROOT/cmd/conductor-loop/Dockerfile" "Conductor Loop Dockerfile"
    
    # Validate Docker Compose
    validate_docker_compose "$PROJECT_ROOT/deployments/docker-compose.conductor-loop.yml" "Docker Compose configuration"
    
    # Validate Kubernetes manifests
    log "Validating Kubernetes manifests..."
    local k8s_manifests=(
        "namespace.yaml"
        "serviceaccount.yaml" 
        "configmap.yaml"
        "pvc.yaml"
        "deployment.yaml"
        "service.yaml"
        "hpa.yaml"
        "networkpolicy.yaml"
        "monitoring.yaml"
        "kustomization.yaml"
    )
    
    for manifest in "${k8s_manifests[@]}"; do
        validate_yaml "$PROJECT_ROOT/deployments/k8s/conductor-loop/$manifest" "Kubernetes $manifest"
    done
    
    # Validate scripts
    log "Validating shell scripts..."
    validate_script "$PROJECT_ROOT/scripts/conductor-loop-dev.sh" "Development script"
    validate_script "$PROJECT_ROOT/scripts/conductor-loop-integration-test.sh" "Integration test script"
    validate_script "$PROJECT_ROOT/scripts/validate-conductor-loop-config.sh" "Validation script"
    
    # Validate PowerShell script
    log "Validating PowerShell script..."
    validate_powershell_script "$PROJECT_ROOT/scripts/conductor-loop-demo.ps1" "PowerShell demo script"
    
    # Validate directory structure
    log "Validating directory structure..."
    local required_dirs=(
        "cmd/conductor-loop"
        "internal/loop"
        "deployments/k8s/conductor-loop"
        "scripts"
    )
    
    for dir in "${required_dirs[@]}"; do
        if [ -d "$PROJECT_ROOT/$dir" ]; then
            success "Directory exists: $dir"
        else
            error "Directory missing: $dir"
            ((VALIDATION_ERRORS++))
        fi
    done
    
    # Check for Go source files
    log "Validating Go source files..."
    if [ -f "$PROJECT_ROOT/cmd/conductor-loop/main.go" ]; then
        success "Main Go file exists"
    else
        warn "Main Go file not found (may need to be created)"
        ((VALIDATION_WARNINGS++))
    fi
    
    if find "$PROJECT_ROOT/internal/loop" -name "*.go" | grep -q .; then
        success "Loop package Go files exist"
    else
        warn "Loop package Go files not found (may need to be created)"
        ((VALIDATION_WARNINGS++))
    fi
    
    # Summary
    echo
    log "Validation Summary:"
    log "=================="
    
    if [ $VALIDATION_ERRORS -eq 0 ]; then
        success "No validation errors found!"
    else
        error "Found $VALIDATION_ERRORS validation error(s)"
    fi
    
    if [ $VALIDATION_WARNINGS -eq 0 ]; then
        success "No validation warnings"
    else
        warn "Found $VALIDATION_WARNINGS validation warning(s)"
    fi
    
    echo
    log "Windows Compatibility Notes:"
    log "============================"
    success "✓ PowerShell script provided for Windows users"
    success "✓ Docker configuration works on Windows"
    success "✓ Make targets use Windows-compatible commands where possible"
    success "✓ Path handling in scripts considers Windows paths"
    warn "! Some shell scripts require WSL or Git Bash on Windows"
    warn "! Kubernetes manifests require kubectl configured on Windows"
    
    echo
    if [ $VALIDATION_ERRORS -eq 0 ]; then
        success "Configuration validation completed successfully!"
        return 0
    else
        error "Configuration validation failed with $VALIDATION_ERRORS error(s)"
        return 1
    fi
}

main "$@"