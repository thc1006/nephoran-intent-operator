#!/bin/bash
# Security Workflow Validation and Hardening Script
# This script validates that all security scanning workflows have proper error handling

set -euo pipefail

echo "=== Security Workflow Validation Script ==="
echo "Checking all security workflows for robustness..."
echo ""

WORKFLOWS_DIR=".github/workflows"
ISSUES_FOUND=0
FIXES_APPLIED=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to check if a workflow has proper SARIF file handling
check_sarif_handling() {
    local file="$1"
    local has_issues=0
    
    echo "Checking $file..."
    
    # Check for gosec SARIF generation without error handling
    if grep -q "gosec.*-fmt sarif" "$file" 2>/dev/null; then
        if ! grep -q "echo.*version.*2.1.0.*schema.*sarif" "$file" 2>/dev/null; then
            echo -e "  ${YELLOW}⚠ Missing SARIF fallback for gosec${NC}"
            has_issues=1
        else
            echo -e "  ${GREEN}✓ Has SARIF fallback for gosec${NC}"
        fi
    fi
    
    # Check for directory creation before file writes
    if grep -q "gosec.*-out" "$file" 2>/dev/null; then
        if ! grep -B5 "gosec.*-out" "$file" | grep -q "mkdir -p" 2>/dev/null; then
            echo -e "  ${YELLOW}⚠ Missing directory creation before gosec output${NC}"
            has_issues=1
        else
            echo -e "  ${GREEN}✓ Has directory creation${NC}"
        fi
    fi
    
    # Check for SBOM generation error handling
    if grep -q "cyclonedx-gomod" "$file" 2>/dev/null; then
        if ! grep -A2 "cyclonedx-gomod" "$file" | grep -q "||" 2>/dev/null; then
            echo -e "  ${YELLOW}⚠ Missing error handling for SBOM generation${NC}"
            has_issues=1
        else
            echo -e "  ${GREEN}✓ Has SBOM error handling${NC}"
        fi
    fi
    
    # Check for vulnerability scanner error handling
    if grep -q "grype\|trivy\|snyk" "$file" 2>/dev/null; then
        if ! grep -A2 "grype\|trivy\|snyk" "$file" | grep -q "||" 2>/dev/null; then
            echo -e "  ${YELLOW}⚠ Missing error handling for vulnerability scanners${NC}"
            has_issues=1
        else
            echo -e "  ${GREEN}✓ Has vulnerability scanner error handling${NC}"
        fi
    fi
    
    return $has_issues
}

# Function to validate SBOM command syntax
check_sbom_syntax() {
    local file="$1"
    local has_issues=0
    
    # Check for incorrect CycloneDX flags
    if grep -q "cyclonedx-gomod.*--output-file" "$file" 2>/dev/null; then
        echo -e "  ${RED}✗ Incorrect CycloneDX flag: --output-file should be -output${NC}"
        has_issues=1
    fi
    
    # Check for incorrect Syft output syntax
    if grep -q "syft.*-o.*>" "$file" 2>/dev/null; then
        echo -e "  ${YELLOW}⚠ Syft using redirect (>) instead of = for output${NC}"
        has_issues=1
    fi
    
    return $has_issues
}

# Main validation loop
echo "=== Scanning Security Workflows ==="
echo ""

for workflow in "$WORKFLOWS_DIR"/*security*.yml "$WORKFLOWS_DIR"/*-ci*.yml "$WORKFLOWS_DIR"/production.yml; do
    if [ -f "$workflow" ]; then
        filename=$(basename "$workflow")
        echo "----------------------------------------"
        echo "Workflow: $filename"
        echo "----------------------------------------"
        
        if check_sarif_handling "$workflow"; then
            echo -e "  ${GREEN}✓ All checks passed${NC}"
        else
            ISSUES_FOUND=$((ISSUES_FOUND + 1))
        fi
        
        if ! check_sbom_syntax "$workflow"; then
            ISSUES_FOUND=$((ISSUES_FOUND + 1))
        fi
        
        echo ""
    fi
done

# Summary
echo "========================================"
echo "VALIDATION SUMMARY"
echo "========================================"

if [ $ISSUES_FOUND -eq 0 ]; then
    echo -e "${GREEN}✓ All security workflows have proper error handling!${NC}"
else
    echo -e "${YELLOW}⚠ Found $ISSUES_FOUND workflows with potential issues${NC}"
    echo ""
    echo "Recommendations:"
    echo "1. Ensure all SARIF files are created even when tools fail"
    echo "2. Always create directories before writing files"
    echo "3. Add error handling (|| operator) for all external tool calls"
    echo "4. Use correct command syntax for SBOM generators:"
    echo "   - CycloneDX: cyclonedx-gomod mod -json -output file.json"
    echo "   - Syft: syft . -o spdx-json=file.json"
    echo "5. Create empty/default output files before running tools"
fi

echo ""
echo "=== Best Practices for Security Workflows ==="
echo ""
cat << 'EOF'
1. SARIF File Creation Pattern:
   ```yaml
   - name: Run Security Tool
     run: |
       mkdir -p reports
       # Create empty SARIF first
       echo '{"version":"2.1.0","$schema":"https://json.schemastore.org/sarif-2.1.0.json","runs":[]}' > reports/tool.sarif
       # Run tool and overwrite if successful
       tool -fmt sarif -out reports/tool.sarif.tmp ./... && \
         mv reports/tool.sarif.tmp reports/tool.sarif || \
         rm -f reports/tool.sarif.tmp
   ```

2. SBOM Generation Pattern:
   ```yaml
   - name: Generate SBOM
     run: |
       # Create empty SBOM first
       echo '{"bomFormat":"CycloneDX","specVersion":"1.4","components":[]}' > sbom.json
       # Generate and overwrite if successful
       cyclonedx-gomod mod -json -output sbom.json.tmp && \
         mv sbom.json.tmp sbom.json || \
         echo "Warning: SBOM generation failed"
   ```

3. Tool Installation Pattern:
   ```yaml
   - name: Install Security Tool
     run: |
       go install tool@version || {
         echo "Warning: Failed to install tool"
         exit 0  # Don't fail the workflow
       }
   ```
EOF

exit $ISSUES_FOUND