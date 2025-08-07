#!/bin/bash

# GitHub Actions Workflow Cleanup Script
# Purpose: Safely remove redundant workflows and prepare for consolidation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== GitHub Actions Workflow Cleanup Script ===${NC}"
echo ""

# Check if we're in the right directory
if [ ! -d ".github/workflows" ]; then
    echo -e "${RED}Error: .github/workflows directory not found!${NC}"
    echo "Please run this script from the project root."
    exit 1
fi

# Dry run mode by default
DRY_RUN=true
if [ "$1" == "--execute" ]; then
    DRY_RUN=false
    echo -e "${YELLOW}Running in EXECUTE mode - files will be deleted!${NC}"
else
    echo -e "${GREEN}Running in DRY-RUN mode - no files will be deleted${NC}"
    echo "To execute deletions, run: $0 --execute"
fi
echo ""

# Phase 1: List workflows to be deleted
echo -e "${GREEN}Phase 1: Workflows marked for deletion${NC}"
echo "----------------------------------------"

WORKFLOWS_TO_DELETE=(
    "ci-cd.yaml"           # Duplicate of ci.yaml
    "testing.yml"          # Redundant with full-suite
    "test-coverage.yml"    # Covered by ci.yaml
    "claude.yml"           # Experimental, not needed
    "claude-code-review.yml" # Experimental, not needed
    "docs.yaml"            # Superseded by docs-publish.yml
    "security.yml"         # Duplicate of security-scan.yml
    "excellence-validation.yml" # Over-engineered for PoC
)

# Count existing files
FOUND_COUNT=0
for workflow in "${WORKFLOWS_TO_DELETE[@]}"; do
    if [ -f ".github/workflows/$workflow" ]; then
        echo -e "  ${YELLOW}✓${NC} Found: $workflow"
        ((FOUND_COUNT++))
    else
        echo -e "  ${RED}✗${NC} Not found: $workflow"
    fi
done

echo ""
echo -e "Found ${GREEN}$FOUND_COUNT${NC} workflows to delete"

# Phase 2: Delete redundant workflows
if [ "$DRY_RUN" = false ]; then
    echo ""
    echo -e "${YELLOW}Phase 2: Deleting redundant workflows${NC}"
    echo "--------------------------------------"
    
    DELETED_COUNT=0
    for workflow in "${WORKFLOWS_TO_DELETE[@]}"; do
        if [ -f ".github/workflows/$workflow" ]; then
            rm ".github/workflows/$workflow"
            echo -e "  ${GREEN}✓${NC} Deleted: $workflow"
            ((DELETED_COUNT++))
        fi
    done
    
    echo ""
    echo -e "${GREEN}Successfully deleted $DELETED_COUNT workflows${NC}"
else
    echo ""
    echo -e "${YELLOW}Phase 2: Would delete these workflows (dry-run)${NC}"
    echo "------------------------------------------------"
    for workflow in "${WORKFLOWS_TO_DELETE[@]}"; do
        if [ -f ".github/workflows/$workflow" ]; then
            echo "  Would delete: $workflow"
        fi
    done
fi

# Phase 3: List workflows to be consolidated
echo ""
echo -e "${GREEN}Phase 3: Workflows to be consolidated${NC}"
echo "--------------------------------------"
echo "The following groups will be consolidated:"
echo ""
echo "1. CI/CD Pipeline → main-ci.yml"
echo "   - ci.yaml (keep as base)"
echo "   - pr-validation.yml (merge)"
echo "   - full-suite.yml (merge)"
echo ""
echo "2. Security Scanning → security.yml"
echo "   - security-scan.yml (keep as base)"
echo "   - security-audit.yml (merge)"
echo "   - codeql-analysis.yml (merge)"
echo "   - docker-security-scan.yml (merge)"
echo "   - govulncheck.yml (merge)"
echo ""
echo "3. Documentation → docs.yml"
echo "   - docs-publish.yml (keep as base)"
echo "   - link-checker.yml (merge)"

# Phase 4: Summary
echo ""
echo -e "${GREEN}=== Summary ===${NC}"
echo ""

# Count remaining workflows
if [ "$DRY_RUN" = false ]; then
    REMAINING=$(ls -1 .github/workflows/*.yml .github/workflows/*.yaml 2>/dev/null | wc -l)
    echo -e "Workflows remaining: ${GREEN}$REMAINING${NC}"
else
    CURRENT=$(ls -1 .github/workflows/*.yml .github/workflows/*.yaml 2>/dev/null | wc -l)
    AFTER=$((CURRENT - FOUND_COUNT))
    echo -e "Current workflows: ${YELLOW}$CURRENT${NC}"
    echo -e "After cleanup: ${GREEN}$AFTER${NC}"
    echo -e "Reduction: ${GREEN}$FOUND_COUNT workflows ($(( (FOUND_COUNT * 100) / CURRENT ))%)${NC}"
fi

echo ""
echo -e "${GREEN}Backup location:${NC} .github/workflows-backup/"
echo ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}This was a dry run. To execute the cleanup, run:${NC}"
    echo "  $0 --execute"
    echo ""
fi

echo -e "${GREEN}✓ Cleanup script completed${NC}"