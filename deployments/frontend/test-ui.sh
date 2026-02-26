#!/bin/bash
set -euo pipefail

# Nephoran UI Test Script
# Tests the frontend UI and API integration

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║   Nephoran Intent Operator UI - End-to-End Test Suite        ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Function to print test result
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ PASS${NC} - $2"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗ FAIL${NC} - $2"
        ((TESTS_FAILED++))
    fi
}

# Test 1: Check if UI service is accessible
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test 1: UI Service Accessibility"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://192.168.10.65:30081/ || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
    print_result 0 "UI is accessible at http://192.168.10.65:30081/"
else
    print_result 1 "UI returned HTTP $HTTP_CODE"
fi
echo ""

# Test 2: Check page title
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test 2: UI Page Content"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

TITLE=$(curl -s http://192.168.10.65:30081/ | grep -o '<title>.*</title>' || echo "")
if echo "$TITLE" | grep -q "Nephoran Intent Operator"; then
    print_result 0 "Page title contains 'Nephoran Intent Operator'"
else
    print_result 1 "Page title not found or incorrect"
fi
echo ""

# Test 3: Check backend service health
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test 3: Backend Service Health"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

BACKEND_HEALTH=$(kubectl exec -n nephoran-system deployment/nephoran-ui -- \
    wget -T 3 -q -O- http://intent-ingest-service.nephoran-intent.svc.cluster.local:8080/healthz 2>&1 || echo "error")
if echo "$BACKEND_HEALTH" | grep -q "ok"; then
    print_result 0 "Backend service is healthy"
else
    print_result 1 "Backend service health check failed"
fi
echo ""

# Test 4: Test intent submission directly to backend
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test 4: Direct Backend Intent Submission"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo "Submitting test intent: 'scale test-app to 2 replicas'"
BACKEND_RESPONSE=$(kubectl exec -n nephoran-system deployment/nephoran-ui -- \
    wget -T 60 -q -O- --post-data='scale test-app to 2 replicas' \
    --header='Content-Type: text/plain' \
    http://intent-ingest-service.nephoran-intent.svc.cluster.local:8080/intent 2>&1 || echo "error")

if echo "$BACKEND_RESPONSE" | grep -q '"status":"accepted"'; then
    print_result 0 "Backend accepted the intent"
    echo "Response preview:"
    echo "$BACKEND_RESPONSE" | jq -r '.preview.description' 2>/dev/null || echo "$BACKEND_RESPONSE" | head -c 100
else
    print_result 1 "Backend did not accept the intent"
    echo "Response: $BACKEND_RESPONSE" | head -c 200
fi
echo ""

# Test 5: Check nginx configuration
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test 5: Nginx Configuration"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

TIMEOUT_CONFIG=$(kubectl get configmap nephoran-ui-nginx-config -n nephoran-system -o yaml | grep "proxy_read_timeout")
if echo "$TIMEOUT_CONFIG" | grep -q "120s"; then
    print_result 0 "Nginx timeout configured correctly (120s)"
else
    print_result 1 "Nginx timeout not set to 120s"
fi
echo ""

# Test 6: Check pod status
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test 6: Pod Status"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

POD_COUNT=$(kubectl get pods -n nephoran-system -l app=nephoran-ui --field-selector=status.phase=Running --no-headers | wc -l)
if [ "$POD_COUNT" -eq 2 ]; then
    print_result 0 "2 UI pods are running"
else
    print_result 1 "Expected 2 pods, found $POD_COUNT"
fi
echo ""

# Test 7: Test healthz endpoint
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test 7: UI Health Endpoint"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

HEALTH_RESPONSE=$(curl -s http://192.168.10.65:30081/healthz)
if echo "$HEALTH_RESPONSE" | grep -q "healthy"; then
    print_result 0 "Health endpoint returns 'healthy'"
else
    print_result 1 "Health endpoint response unexpected: $HEALTH_RESPONSE"
fi
echo ""

# Summary
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                       Test Summary                             ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
echo -e "${GREEN}Passed:${NC} $TESTS_PASSED"
echo -e "${RED}Failed:${NC} $TESTS_FAILED"
echo ""

TOTAL_TESTS=$((TESTS_PASSED + TESTS_FAILED))
PASS_RATE=$((TESTS_PASSED * 100 / TOTAL_TESTS))

if [ $PASS_RATE -eq 100 ]; then
    echo -e "${GREEN}✓ All tests passed! (100%)${NC}"
    exit 0
elif [ $PASS_RATE -ge 80 ]; then
    echo -e "${YELLOW}⚠ Most tests passed ($PASS_RATE%)${NC}"
    exit 0
else
    echo -e "${RED}✗ Multiple test failures ($PASS_RATE% pass rate)${NC}"
    exit 1
fi
