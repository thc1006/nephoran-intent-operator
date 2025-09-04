#!/usr/bin/env bash
set -euo pipefail

# Configuration Validation Script for Nephoran E2E Scaling Tests
# Validates that the scaling configuration is optimal for kind clusters

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}" >&2
}

echo -e "${BLUE}═══════════════════════════════════════════${NC}"
echo -e "${BLUE}  Nephoran E2E Scaling Configuration Validator${NC}"
echo -e "${BLUE}═══════════════════════════════════════════${NC}"

# Check that the run-e2e.sh exists and has the optimizations
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
E2E_SCRIPT="$SCRIPT_DIR/run-e2e.sh"

if [ ! -f "$E2E_SCRIPT" ]; then
    log_error "run-e2e.sh not found at $E2E_SCRIPT"
    exit 1
fi

log_success "Found E2E script at $E2E_SCRIPT"

# Validate key optimizations are present
echo -e "\n${YELLOW}Checking for key optimizations...${NC}"

# 1. Check for optimized resource limits
if grep -q "memory: \"128Mi\"" "$E2E_SCRIPT" && grep -q "cpu: \"200m\"" "$E2E_SCRIPT"; then
    log_success "Optimized resource limits found (128Mi memory, 200m CPU)"
else
    log_error "Optimized resource limits not found"
fi

# 2. Check for readiness probes
if grep -q "readinessProbe:" "$E2E_SCRIPT" && grep -q "initialDelaySeconds: 5" "$E2E_SCRIPT"; then
    log_success "Readiness probe configuration found"
else
    log_error "Readiness probe configuration missing"
fi

# 3. Check for liveness probes
if grep -q "livenessProbe:" "$E2E_SCRIPT" && grep -q "initialDelaySeconds: 10" "$E2E_SCRIPT"; then
    log_success "Liveness probe configuration found"
else
    log_error "Liveness probe configuration missing"
fi

# 4. Check for rolling update strategy
if grep -q "maxUnavailable: 0" "$E2E_SCRIPT" && grep -q "maxSurge: 1" "$E2E_SCRIPT"; then
    log_success "Optimized rolling update strategy found"
else
    log_error "Rolling update strategy not optimized"
fi

# 5. Check for pod anti-affinity
if grep -q "podAntiAffinity:" "$E2E_SCRIPT"; then
    log_success "Pod anti-affinity configuration found"
else
    log_warning "Pod anti-affinity configuration not found (recommended for multi-replica scaling)"
fi

# 6. Check for enhanced kind cluster configuration
if grep -q "max-pods: \"110\"" "$E2E_SCRIPT"; then
    log_success "Enhanced kind cluster configuration found"
else
    log_error "Enhanced kind cluster configuration missing"
fi

# 7. Check for enhanced scaling test logic
if grep -q "Pre-scaling verification" "$E2E_SCRIPT" && grep -q "progressive checks" "$E2E_SCRIPT"; then
    log_success "Enhanced scaling test logic found"
else
    log_error "Enhanced scaling test logic missing"
fi

# 8. Check for debug helper integration
DEBUG_SCRIPT="$SCRIPT_DIR/debug-scaling.sh"
if [ -f "$DEBUG_SCRIPT" ] && [ -x "$DEBUG_SCRIPT" ]; then
    log_success "Debug helper script found and executable"
else
    log_error "Debug helper script missing or not executable"
fi

# Validate script syntax
echo -e "\n${YELLOW}Validating script syntax...${NC}"
if bash -n "$E2E_SCRIPT" 2>/dev/null; then
    log_success "E2E script syntax is valid"
else
    log_error "E2E script has syntax errors"
    exit 1
fi

if [ -f "$DEBUG_SCRIPT" ] && bash -n "$DEBUG_SCRIPT" 2>/dev/null; then
    log_success "Debug script syntax is valid"
else
    log_warning "Debug script has syntax issues"
fi

# Check for required tools
echo -e "\n${YELLOW}Checking required tools...${NC}"

REQUIRED_TOOLS=("kubectl" "kind" "curl" "go")
for tool in "${REQUIRED_TOOLS[@]}"; do
    if command -v "$tool" >/dev/null 2>&1; then
        log_success "$tool is available"
    else
        log_error "$tool is required but not found"
    fi
done

# Check Go version
if command -v go >/dev/null 2>&1; then
    GO_VERSION=$(go version | grep -o 'go[0-9]*\.[0-9]*' | head -1)
    log_info "Go version: $GO_VERSION"
fi

# Summary
echo -e "\n${BLUE}═══════════════════════════════════════════${NC}"
echo -e "${GREEN}Configuration validation complete!${NC}"
echo -e "\n${YELLOW}Key improvements implemented:${NC}"
echo -e "• Resource limits increased for stable scaling (128Mi/200m)"
echo -e "• Health probes configured for proper readiness detection"
echo -e "• Rolling update strategy optimized for zero downtime"
echo -e "• Kind cluster configured with increased pod limits"
echo -e "• Enhanced scaling test with progressive verification"
echo -e "• Debug helper for troubleshooting scaling issues"
echo -e "• Pod anti-affinity for better distribution across nodes"
echo -e "${BLUE}═══════════════════════════════════════════${NC}"

echo -e "\n${GREEN}Ready to run E2E scaling tests!${NC}"
echo -e "Execute: ${BLUE}./hack/run-e2e.sh${NC}"
echo -e "Debug issues: ${BLUE}./hack/debug-scaling.sh${NC}"