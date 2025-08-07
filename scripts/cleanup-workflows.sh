#!/bin/bash

# GitHub Actions Workflow Cleanup Script
# Purpose: Safely remove redundant workflows and prepare for consolidation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Global variables
DRY_RUN=true
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

# Function to check prerequisites
check_prerequisites() {
    if [ ! -d ".github/workflows" ]; then
        echo -e "${RED}Error: .github/workflows directory not found!${NC}"
        echo "Please run this script from the project root."
        exit 1
    fi
}

# Function to parse command line arguments
parse_arguments() {
    if [ "$1" == "--execute" ]; then
        DRY_RUN=false
        echo -e "${YELLOW}Running in EXECUTE mode - files will be deleted!${NC}"
    else
        echo -e "${GREEN}Running in DRY-RUN mode - no files will be deleted${NC}"
        echo "To execute deletions, run: $0 --execute"
    fi
    echo ""
}

# Function to list workflows marked for deletion
list_workflows() {
    local found_count=0
    local workflow
    
    echo -e "${GREEN}Phase 1: Workflows marked for deletion${NC}"
    echo "----------------------------------------"
    
    for workflow in "${WORKFLOWS_TO_DELETE[@]}"; do
        if [ -f ".github/workflows/$workflow" ]; then
            echo -e "  ${YELLOW}✓${NC} Found: $workflow"
            ((found_count++))
        else
            echo -e "  ${RED}✗${NC} Not found: $workflow"
        fi
    done
    
    echo ""
    echo -e "Found ${GREEN}$found_count${NC} workflows to delete"
    
    # Return the count for use in other functions
    echo "$found_count"
}

# Function to delete workflows
delete_workflows() {
    local deleted_count=0
    local workflow
    
    if [ "$DRY_RUN" = false ]; then
        echo ""
        echo -e "${YELLOW}Phase 2: Deleting redundant workflows${NC}"
        echo "--------------------------------------"
        
        for workflow in "${WORKFLOWS_TO_DELETE[@]}"; do
            if [ -f ".github/workflows/$workflow" ]; then
                rm ".github/workflows/$workflow"
                echo -e "  ${GREEN}✓${NC} Deleted: $workflow"
                ((deleted_count++))
            fi
        done
        
        echo ""
        echo -e "${GREEN}Successfully deleted $deleted_count workflows${NC}"
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
}

# Function to show consolidation plan
show_consolidation_plan() {
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
}

# Function to show summary
show_summary() {
    local found_count=$1
    local current
    local remaining
    local after
    
    echo ""
    echo -e "${GREEN}=== Summary ===${NC}"
    echo ""
    
    if [ "$DRY_RUN" = false ]; then
        remaining=$(find .github/workflows -name "*.yml" -o -name "*.yaml" 2>/dev/null | wc -l)
        echo -e "Workflows remaining: ${GREEN}$remaining${NC}"
    else
        current=$(find .github/workflows -name "*.yml" -o -name "*.yaml" 2>/dev/null | wc -l)
        after=$((current - found_count))
        echo -e "Current workflows: ${YELLOW}$current${NC}"
        echo -e "After cleanup: ${GREEN}$after${NC}"
        if [ "$current" -gt 0 ]; then
            echo -e "Reduction: ${GREEN}$found_count workflows ($(( (found_count * 100) / current ))%)${NC}"
        fi
    fi
    
    echo ""
    echo -e "${GREEN}Backup location:${NC} .github/workflows-backup/"
    echo ""
    
    if [ "$DRY_RUN" = true ]; then
        echo -e "${YELLOW}This was a dry run. To execute the cleanup, run:${NC}"
        echo "  $0 --execute"
        echo ""
    fi
}

# Main execution
main() {
    echo -e "${GREEN}=== GitHub Actions Workflow Cleanup Script ===${NC}"
    echo ""
    
    check_prerequisites
    parse_arguments "$@"
    
    # Capture the found count from list_workflows
    local found_count
    found_count=$(list_workflows | grep -Eo 'Found [0-9]+ workflows' | grep -Eo '[0-9]+')
    
    # Delete workflows (this function is always called, checks DRY_RUN internally)
    delete_workflows
    
    show_consolidation_plan
    show_summary "$found_count"
    
    echo -e "${GREEN}✓ Cleanup script completed${NC}"
}

# Run main function with all arguments
main "$@"