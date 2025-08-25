#!/usr/bin/env bash
set -euo pipefail

# Quick validation script for E2E harness enhancements
# Tests the new functionality without running a full E2E test

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== E2E Harness Enhancement Validation ===${NC}"

# Test 1: Script syntax validation
echo -e "\n${BLUE}1. Testing script syntax...${NC}"
if bash -n "$SCRIPT_DIR/run-e2e.sh"; then
    echo -e "${GREEN}✓ Script syntax is valid${NC}"
else
    echo -e "${RED}✗ Script syntax errors found${NC}"
    exit 1
fi

# Test 2: Check for expected functions
echo -e "\n${BLUE}2. Checking for enhanced functions...${NC}"
EXPECTED_FUNCTIONS=(
    "wait_for_http_health"
    "wait_for_process_health"
    "verify_handoff_structure"
)

for func in "${EXPECTED_FUNCTIONS[@]}"; do
    if grep -q "^${func}()" "$SCRIPT_DIR/run-e2e.sh"; then
        echo -e "${GREEN}✓ Function $func found${NC}"
    else
        echo -e "${RED}✗ Function $func missing${NC}"
        exit 1
    fi
done

# Test 3: Check for proper error handling patterns
echo -e "\n${BLUE}3. Checking error handling patterns...${NC}"
ERROR_PATTERNS=(
    "log_test_fail"
    "timed out after"
    "wait_for_http_health"
)

for pattern in "${ERROR_PATTERNS[@]}"; do
    if grep -q "$pattern" "$SCRIPT_DIR/run-e2e.sh"; then
        echo -e "${GREEN}✓ Error pattern '$pattern' found${NC}"
    else
        echo -e "${RED}✗ Error pattern '$pattern' missing${NC}"
        exit 1
    fi
done

# Test 4: Validate configuration variables
echo -e "\n${BLUE}4. Checking configuration variables...${NC}"
CONFIG_VARS=(
    "INTENT_INGEST_PID"
    "CONDUCTOR_LOOP_PID"
    "HANDOFF_DIR"
    "CURL_BIN"
)

for var in "${CONFIG_VARS[@]}"; do
    if grep -q "$var" "$SCRIPT_DIR/run-e2e.sh"; then
        echo -e "${GREEN}✓ Variable $var found${NC}"
    else
        echo -e "${RED}✗ Variable $var missing${NC}"
        exit 1
    fi
done

# Test 5: Check webhook configuration alignment
echo -e "\n${BLUE}5. Checking webhook configuration...${NC}"
if grep -q "nephoran.com" "$PROJECT_ROOT/config/webhook/manifests.yaml"; then
    echo -e "${GREEN}✓ Webhook uses correct API group (nephoran.com)${NC}"
else
    echo -e "${RED}✗ Webhook API group mismatch${NC}"
    exit 1
fi

# Test 6: Validate contract schema exists
echo -e "\n${BLUE}6. Checking contract schema...${NC}"
if [ -f "$PROJECT_ROOT/docs/contracts/intent.schema.json" ]; then
    echo -e "${GREEN}✓ Intent schema contract found${NC}"
    
    # Validate JSON syntax (if jq is available)
    if command -v jq >/dev/null 2>&1; then
        if jq . "$PROJECT_ROOT/docs/contracts/intent.schema.json" >/dev/null 2>&1; then
            echo -e "${GREEN}✓ Schema JSON is valid${NC}"
        else
            echo -e "${RED}✗ Schema JSON is invalid${NC}"
            exit 1
        fi
    else
        echo -e "${BLUE}ℹ jq not available, skipping JSON validation${NC}"
    fi
else
    echo -e "${RED}✗ Intent schema contract missing${NC}"
    exit 1
fi

# Test 7: Check for Windows compatibility
echo -e "\n${BLUE}7. Checking Windows compatibility...${NC}"
WINDOWS_PATTERNS=(
    "mingw.*msys"
    "\.exe"
    "Git Bash"
)

for pattern in "${WINDOWS_PATTERNS[@]}"; do
    if grep -q "$pattern" "$SCRIPT_DIR/run-e2e.sh"; then
        echo -e "${GREEN}✓ Windows pattern '$pattern' found${NC}"
    else
        echo -e "${RED}✗ Windows pattern '$pattern' missing${NC}"
        exit 1
    fi
done

# Test 8: Check test samples exist and have correct format
echo -e "\n${BLUE}8. Checking test samples...${NC}"
SAMPLES_DIR="$PROJECT_ROOT/tests/e2e/samples"
if [ -d "$SAMPLES_DIR" ]; then
    SAMPLE_COUNT=$(find "$SAMPLES_DIR" -name "*.yaml" | wc -l)
    if [ "$SAMPLE_COUNT" -gt 0 ]; then
        echo -e "${GREEN}✓ Found $SAMPLE_COUNT test samples${NC}"
        
        # Check if samples use correct API version
        if grep -q "apiVersion: nephoran.com/v1" "$SAMPLES_DIR"/*.yaml; then
            echo -e "${GREEN}✓ Samples use correct API version${NC}"
        else
            echo -e "${RED}✗ Samples API version mismatch${NC}"
            exit 1
        fi
    else
        echo -e "${RED}✗ No test samples found${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ Samples directory missing${NC}"
    exit 1
fi

echo -e "\n${GREEN}=== All Enhancement Validations Passed! ===${NC}"
echo -e "${BLUE}The E2E harness is ready for testing.${NC}"
echo -e "${BLUE}Run: ./hack/run-e2e.sh to execute the full test suite${NC}"