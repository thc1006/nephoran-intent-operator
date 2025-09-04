#!/usr/bin/env bash
set -euo pipefail

# Quick prerequisite check for E2E tests
# This script validates that all required tools are available

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔍 Checking E2E Test Prerequisites${NC}"
echo "=================================="

# Check required tools
check_tool() {
    local tool="$1"
    local version_flag="${2:---version}"
    
    if command -v "$tool" &> /dev/null; then
        local version_output
        version_output=$($tool $version_flag 2>&1 | head -1)
        echo -e "✅ ${GREEN}$tool${NC}: $version_output"
        return 0
    else
        echo -e "❌ ${RED}$tool${NC}: Not found"
        return 1
    fi
}

# Track missing tools
missing_tools=0

# Check each required tool
echo -e "\n${BLUE}Required Tools:${NC}"
check_tool "kind" || ((missing_tools++))
check_tool "kubectl" --version || ((missing_tools++))
check_tool "go" "version" || ((missing_tools++))

# Check optional tools
echo -e "\n${BLUE}Optional Tools:${NC}"
check_tool "docker" --version || echo -e "⚠️  ${YELLOW}docker${NC}: Not found (may work with podman)"

# Check important files
echo -e "\n${BLUE}Required Files:${NC}"
if [ -f "config/crd/bases/intent.nephoran.com_networkintents.yaml" ]; then
    echo -e "✅ ${GREEN}NetworkIntent CRD${NC}: Found"
else
    echo -e "❌ ${RED}NetworkIntent CRD${NC}: Not found"
    ((missing_tools++))
fi

if [ -f "tools/verify-scale.go" ]; then
    echo -e "✅ ${GREEN}Scale verifier tool${NC}: Found"
else
    echo -e "⚠️  ${YELLOW}Scale verifier tool${NC}: Not found (will use fallback)"
fi

# Check cluster access (if any cluster exists)
echo -e "\n${BLUE}Cluster Access:${NC}"
if kubectl cluster-info &>/dev/null; then
    current_context=$(kubectl config current-context)
    echo -e "✅ ${GREEN}Current context${NC}: $current_context"
else
    echo -e "ℹ️  ${BLUE}No active cluster${NC} (will create kind cluster)"
fi

# Summary
echo -e "\n${BLUE}Summary:${NC}"
echo "========"

if [ $missing_tools -eq 0 ]; then
    echo -e "🎉 ${GREEN}All prerequisites are met!${NC}"
    echo -e "   Run: ${BLUE}hack/run-e2e-lightweight.sh${NC}"
    exit 0
else
    echo -e "⚠️  ${YELLOW}$missing_tools missing prerequisite(s)${NC}"
    echo ""
    echo "Install missing tools:"
    if ! command -v kind &> /dev/null; then
        echo "  • kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
    fi
    if ! command -v kubectl &> /dev/null; then
        echo "  • kubectl: https://kubernetes.io/docs/tasks/tools/"
    fi
    if ! command -v go &> /dev/null; then
        echo "  • go: https://golang.org/doc/install"
    fi
    echo ""
    echo "Then run: hack/run-e2e-lightweight.sh"
    exit 1
fi