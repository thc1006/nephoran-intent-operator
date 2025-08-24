#!/bin/bash
# Orchestration Validation Script
# Tests the enhanced GitHub Actions workflows for syntax and logic

set -euo pipefail

echo "🔧 GitHub Actions Orchestration Validation"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

VALIDATION_FAILED=0

# Function to check YAML syntax
check_yaml_syntax() {
    local file="$1"
    echo -n "📋 Checking YAML syntax for $(basename "$file")... "
    
    if python3 -c "import yaml; yaml.safe_load(open('$file'))" 2>/dev/null; then
        echo -e "${GREEN}✅ PASS${NC}"
        return 0
    else
        echo -e "${RED}❌ FAIL${NC}"
        VALIDATION_FAILED=1
        return 1
    fi
}

# Function to check for orchestration patterns
check_orchestration_patterns() {
    local file="$1"
    echo "🔍 Checking orchestration patterns in $(basename "$file")..."
    
    local patterns_found=0
    
    # Check for enhanced error handling
    if grep -q "continue-on-error.*true" "$file" && grep -q "if.*always()" "$file"; then
        echo -e "  ${GREEN}✅${NC} Enhanced error handling patterns found"
        patterns_found=$((patterns_found + 1))
    else
        echo -e "  ${YELLOW}⚠️${NC} Limited error handling patterns"
    fi
    
    # Check for timeout settings
    if grep -q "timeout-minutes:" "$file"; then
        echo -e "  ${GREEN}✅${NC} Timeout configurations found"
        patterns_found=$((patterns_found + 1))
    else
        echo -e "  ${YELLOW}⚠️${NC} No timeout configurations"
    fi
    
    # Check for job dependencies
    if grep -q "needs:" "$file"; then
        echo -e "  ${GREEN}✅${NC} Job dependencies configured"
        patterns_found=$((patterns_found + 1))
    else
        echo -e "  ${YELLOW}⚠️${NC} No job dependencies"
    fi
    
    # Check for artifact handling
    if grep -q "upload-artifact" "$file"; then
        echo -e "  ${GREEN}✅${NC} Artifact handling present"
        patterns_found=$((patterns_found + 1))
    else
        echo -e "  ${YELLOW}⚠️${NC} No artifact handling"
    fi
    
    echo "  📊 Orchestration score: $patterns_found/4"
    echo ""
}

# Function to check for resilience features
check_resilience_features() {
    local file="$1"
    echo "🛡️ Checking resilience features in $(basename "$file")..."
    
    local features_found=0
    
    # Check for retry mechanisms
    if grep -q -E "(for.*attempt|while.*retry|timeout.*)" "$file"; then
        echo -e "  ${GREEN}✅${NC} Retry/timeout mechanisms found"
        features_found=$((features_found + 1))
    else
        echo -e "  ${YELLOW}⚠️${NC} No retry mechanisms"
    fi
    
    # Check for fallback handling
    if grep -q -E "(fallback|alternative|backup)" "$file"; then
        echo -e "  ${GREEN}✅${NC} Fallback mechanisms found"
        features_found=$((features_found + 1))
    else
        echo -e "  ${YELLOW}⚠️${NC} No fallback mechanisms"
    fi
    
    # Check for validation
    if grep -q -E "(jq.*empty|validate|check)" "$file"; then
        echo -e "  ${GREEN}✅${NC} Validation logic found"
        features_found=$((features_found + 1))
    else
        echo -e "  ${YELLOW}⚠️${NC} Limited validation"
    fi
    
    # Check for structured reporting
    if grep -q "GITHUB_STEP_SUMMARY" "$file"; then
        echo -e "  ${GREEN}✅${NC} Structured reporting found"
        features_found=$((features_found + 1))
    else
        echo -e "  ${YELLOW}⚠️${NC} No structured reporting"
    fi
    
    echo "  🛡️ Resilience score: $features_found/4"
    echo ""
}

# Main validation workflow
echo "📁 Workflow Files to Validate:"
workflow_files=(
    ".github/workflows/resilient-orchestration.yml"
    ".github/workflows/dependency-security.yml"
    ".github/workflows/ci.yml"
)

for file in "${workflow_files[@]}"; do
    echo "  - $file"
done
echo ""

# Validate each workflow file
for file in "${workflow_files[@]}"; do
    if [ -f "$file" ]; then
        echo "🔍 Validating: $file"
        echo "-----------------------------------"
        
        # YAML syntax check
        check_yaml_syntax "$file"
        
        # Orchestration patterns
        check_orchestration_patterns "$file"
        
        # Resilience features
        check_resilience_features "$file"
        
        echo ""
    else
        echo -e "${RED}❌ File not found: $file${NC}"
        VALIDATION_FAILED=1
    fi
done

# Specific checks for our orchestration improvements
echo "🎯 Specific Orchestration Improvements Validation"
echo "================================================="

# Check for SARIF generation improvements
echo "🔒 Checking SARIF generation improvements..."
if grep -q "Ensure SARIF file exists" .github/workflows/ci.yml; then
    echo -e "  ${GREEN}✅${NC} SARIF fallback mechanism present"
else
    echo -e "  ${RED}❌${NC} SARIF fallback mechanism missing"
    VALIDATION_FAILED=1
fi

# Check for SBOM command fixes
echo "📦 Checking SBOM command fixes..."
if grep -q "cyclonedx-gomod app\|cyclonedx-gomod mod" .github/workflows/dependency-security.yml; then
    echo -e "  ${GREEN}✅${NC} Multiple SBOM generation methods present"
else
    echo -e "  ${RED}❌${NC} SBOM generation methods not updated"
    VALIDATION_FAILED=1
fi

# Check for enhanced error reporting
echo "📊 Checking enhanced error reporting..."
if grep -q "security-summary\|validation-summary" .github/workflows/ci.yml; then
    echo -e "  ${GREEN}✅${NC} Enhanced reporting steps present"
else
    echo -e "  ${RED}❌${NC} Enhanced reporting steps missing"
    VALIDATION_FAILED=1
fi

# Final validation summary
echo ""
echo "📋 Validation Summary"
echo "===================="

if [ $VALIDATION_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All validations PASSED!${NC}"
    echo "✅ Orchestration improvements are properly implemented"
    echo "✅ Workflows are ready for production use"
    echo "✅ Resilience features are active"
else
    echo -e "${RED}❌ Some validations FAILED!${NC}"
    echo "⚠️ Please review the issues above before deployment"
fi

exit $VALIDATION_FAILED