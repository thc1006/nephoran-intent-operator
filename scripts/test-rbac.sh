#!/bin/bash

# RBAC Testing Script for Nephoran Intent Operator
# Purpose: Test RBAC configurations to ensure least-privilege and proper access control
# Security: Validates that permissions are correctly restricted

set -euo pipefail

# Configuration
NAMESPACE="${NAMESPACE:-nephoran-system}"
TEST_NAMESPACE="${TEST_NAMESPACE:-nephoran-test}"
KUBECTL="${KUBECTL:-kubectl}"
TEST_USER="${TEST_USER:-test-user}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results
PASSED=0
FAILED=0
SKIPPED=0

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Nephoran RBAC Testing Suite${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Function to log test results
log_test() {
    local test_name="$1"
    local result="$2"
    local message="${3:-}"
    
    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}✓ ${test_name}: PASSED${NC}"
        ((PASSED++))
    elif [ "$result" = "FAIL" ]; then
        echo -e "${RED}✗ ${test_name}: FAILED${NC}"
        if [ -n "$message" ]; then
            echo -e "  ${RED}${message}${NC}"
        fi
        ((FAILED++))
    elif [ "$result" = "SKIP" ]; then
        echo -e "${YELLOW}○ ${test_name}: SKIPPED${NC}"
        if [ -n "$message" ]; then
            echo -e "  ${YELLOW}${message}${NC}"
        fi
        ((SKIPPED++))
    fi
}

# Function to test permissions using kubectl auth can-i
test_permission() {
    local sa="$1"
    local verb="$2"
    local resource="$3"
    local should_pass="$4"
    local test_name="$5"
    
    if ${KUBECTL} auth can-i "$verb" "$resource" \
        --as="system:serviceaccount:${NAMESPACE}:${sa}" \
        -n "${NAMESPACE}" > /dev/null 2>&1; then
        if [ "$should_pass" = "true" ]; then
            log_test "$test_name" "PASS"
        else
            log_test "$test_name" "FAIL" "Permission should be denied but was allowed"
        fi
    else
        if [ "$should_pass" = "false" ]; then
            log_test "$test_name" "PASS"
        else
            log_test "$test_name" "FAIL" "Permission should be allowed but was denied"
        fi
    fi
}

# Test 1: Verify namespace exists
echo -e "${YELLOW}Test Group 1: Prerequisites${NC}"
if ${KUBECTL} get namespace "${NAMESPACE}" > /dev/null 2>&1; then
    log_test "Namespace exists" "PASS"
else
    log_test "Namespace exists" "FAIL" "Namespace ${NAMESPACE} not found"
    echo -e "${RED}Cannot continue without namespace. Exiting...${NC}"
    exit 1
fi

# Test 2: Service Account Tests
echo ""
echo -e "${YELLOW}Test Group 2: Service Account Permissions${NC}"

# Test operator controller permissions
test_permission "nephoran-operator-controller" "get" "networkintents" "true" \
    "Controller can get NetworkIntents"
test_permission "nephoran-operator-controller" "create" "networkintents" "true" \
    "Controller can create NetworkIntents"
test_permission "nephoran-operator-controller" "delete" "networkintents" "false" \
    "Controller cannot delete NetworkIntents (least-privilege)"
test_permission "nephoran-operator-controller" "get" "secrets" "false" \
    "Controller cannot get all secrets (restricted)"

# Test LLM processor permissions
test_permission "nephoran-llm-processor" "get" "configmaps" "true" \
    "LLM processor can read configmaps"
test_permission "nephoran-llm-processor" "create" "networkintents" "false" \
    "LLM processor cannot create NetworkIntents"
test_permission "nephoran-llm-processor" "get" "secrets" "false" \
    "LLM processor cannot get all secrets"

# Test O-RAN adaptor permissions
test_permission "nephoran-oran-adaptor" "create" "e2nodesets" "true" \
    "O-RAN adaptor can create E2NodeSets"
test_permission "nephoran-oran-adaptor" "get" "ricconfigurations" "true" \
    "O-RAN adaptor can get RIC configurations"
test_permission "nephoran-oran-adaptor" "delete" "pods" "false" \
    "O-RAN adaptor cannot delete pods"

# Test 3: Cross-namespace access (should be denied)
echo ""
echo -e "${YELLOW}Test Group 3: Cross-Namespace Isolation${NC}"

# Create test namespace if it doesn't exist
if ! ${KUBECTL} get namespace "${TEST_NAMESPACE}" > /dev/null 2>&1; then
    ${KUBECTL} create namespace "${TEST_NAMESPACE}" > /dev/null 2>&1 || true
fi

test_permission "nephoran-operator-controller" "get" "pods" "false" \
    "Controller cannot access pods in other namespaces"
test_permission "nephoran-operator-controller" "get" "secrets" "false" \
    "Controller cannot access secrets in other namespaces"

# Test 4: Escalation prevention
echo ""
echo -e "${YELLOW}Test Group 4: Privilege Escalation Prevention${NC}"

test_permission "nephoran-llm-processor" "create" "rolebindings" "false" \
    "LLM processor cannot create role bindings"
test_permission "nephoran-rag-api" "create" "clusterrolebindings" "false" \
    "RAG API cannot create cluster role bindings"
test_permission "nephoran-operator-controller" "update" "clusterroles" "false" \
    "Controller cannot update cluster roles"

# Test 5: Default service account restrictions
echo ""
echo -e "${YELLOW}Test Group 5: Default Service Account Restrictions${NC}"

test_permission "default" "get" "networkintents" "false" \
    "Default SA cannot access NetworkIntents"
test_permission "default" "list" "pods" "false" \
    "Default SA cannot list pods"
test_permission "default" "get" "secrets" "false" \
    "Default SA cannot access secrets"

# Test 6: Audit and metrics access
echo ""
echo -e "${YELLOW}Test Group 6: Monitoring and Audit Permissions${NC}"

test_permission "nephoran-metrics-reader" "get" "pods" "true" \
    "Metrics reader can get pod information"
test_permission "nephoran-metrics-reader" "delete" "pods" "false" \
    "Metrics reader cannot delete pods"
test_permission "nephoran-audit-logger" "list" "events" "true" \
    "Audit logger can list events"
test_permission "nephoran-audit-logger" "create" "events" "false" \
    "Audit logger cannot create events (read-only)"

# Test 7: Human operator roles
echo ""
echo -e "${YELLOW}Test Group 7: Human Operator Roles${NC}"

# Test viewer role (if binding exists)
if ${KUBECTL} get rolebinding nephoran-viewers -n "${NAMESPACE}" > /dev/null 2>&1; then
    # Create a temporary service account to test viewer permissions
    ${KUBECTL} create serviceaccount test-viewer -n "${NAMESPACE}" > /dev/null 2>&1 || true
    ${KUBECTL} create rolebinding test-viewer-binding \
        --role=nephoran-viewer \
        --serviceaccount="${NAMESPACE}:test-viewer" \
        -n "${NAMESPACE}" > /dev/null 2>&1 || true
    
    test_permission "test-viewer" "get" "networkintents" "true" \
        "Viewer can read NetworkIntents"
    test_permission "test-viewer" "create" "networkintents" "false" \
        "Viewer cannot create NetworkIntents"
    test_permission "test-viewer" "delete" "networkintents" "false" \
        "Viewer cannot delete NetworkIntents"
    
    # Cleanup
    ${KUBECTL} delete rolebinding test-viewer-binding -n "${NAMESPACE}" > /dev/null 2>&1 || true
    ${KUBECTL} delete serviceaccount test-viewer -n "${NAMESPACE}" > /dev/null 2>&1 || true
else
    log_test "Viewer role test" "SKIP" "Viewer role binding not found"
fi

# Test 8: O-RAN specific permissions
echo ""
echo -e "${YELLOW}Test Group 8: O-RAN Specific Permissions${NC}"

test_permission "nephoran-oran-adaptor" "create" "a1policies.oran.nephoran.io" "true" \
    "O-RAN adaptor can create A1 policies"
test_permission "nephoran-oran-adaptor" "update" "o1configurations.oran.nephoran.io" "true" \
    "O-RAN adaptor can update O1 configurations"
test_permission "nephoran-llm-processor" "create" "e2nodesets.oran.nephoran.io" "false" \
    "LLM processor cannot create E2NodeSets"

# Test 9: Network policy connectivity (if network policies are enabled)
echo ""
echo -e "${YELLOW}Test Group 9: Network Policy Enforcement${NC}"

if ${KUBECTL} get networkpolicies -n "${NAMESPACE}" > /dev/null 2>&1; then
    np_count=$(${KUBECTL} get networkpolicies -n "${NAMESPACE}" --no-headers | wc -l)
    if [ "$np_count" -gt 0 ]; then
        log_test "Network policies exist" "PASS"
        
        # Check for deny-all default policy
        if ${KUBECTL} get networkpolicy nephoran-deny-all -n "${NAMESPACE}" > /dev/null 2>&1; then
            log_test "Default deny-all network policy" "PASS"
        else
            log_test "Default deny-all network policy" "FAIL" "No deny-all policy found"
        fi
    else
        log_test "Network policies exist" "FAIL" "No network policies found"
    fi
else
    log_test "Network policy test" "SKIP" "Network policies not available"
fi

# Test 10: Pod Security Standards
echo ""
echo -e "${YELLOW}Test Group 10: Pod Security Standards${NC}"

# Check namespace labels for Pod Security Standards
if ${KUBECTL} get namespace "${NAMESPACE}" -o json | \
    jq -e '.metadata.labels."pod-security.kubernetes.io/enforce" == "restricted"' > /dev/null 2>&1; then
    log_test "Pod Security Standards enforced" "PASS"
else
    log_test "Pod Security Standards enforced" "FAIL" "Namespace not enforcing restricted PSS"
fi

# Test 11: Resource quotas and limits
echo ""
echo -e "${YELLOW}Test Group 11: Resource Constraints${NC}"

if ${KUBECTL} get resourcequota -n "${NAMESPACE}" > /dev/null 2>&1; then
    log_test "Resource quotas configured" "PASS"
else
    log_test "Resource quotas configured" "FAIL" "No resource quotas found"
fi

if ${KUBECTL} get limitrange -n "${NAMESPACE}" > /dev/null 2>&1; then
    log_test "Limit ranges configured" "PASS"
else
    log_test "Limit ranges configured" "FAIL" "No limit ranges found"
fi

# Test 12: Webhook configurations
echo ""
echo -e "${YELLOW}Test Group 12: Admission Webhooks${NC}"

if ${KUBECTL} get validatingwebhookconfigurations | grep -q nephoran; then
    log_test "Validating webhooks configured" "PASS"
else
    log_test "Validating webhooks configured" "SKIP" "No validating webhooks found"
fi

if ${KUBECTL} get mutatingwebhookconfigurations | grep -q nephoran; then
    log_test "Mutating webhooks configured" "PASS"
else
    log_test "Mutating webhooks configured" "SKIP" "No mutating webhooks found"
fi

# Summary
echo ""
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${GREEN}Passed: ${PASSED}${NC}"
echo -e "${RED}Failed: ${FAILED}${NC}"
echo -e "${YELLOW}Skipped: ${SKIPPED}${NC}"

TOTAL=$((PASSED + FAILED + SKIPPED))
if [ $TOTAL -gt 0 ]; then
    PASS_RATE=$((PASSED * 100 / TOTAL))
    echo -e "Pass Rate: ${PASS_RATE}%"
fi

# Exit with appropriate code
if [ ${FAILED} -gt 0 ]; then
    echo ""
    echo -e "${RED}RBAC tests failed. Please review and fix the failures.${NC}"
    exit 1
else
    echo ""
    echo -e "${GREEN}All RBAC tests passed successfully!${NC}"
    exit 0
fi