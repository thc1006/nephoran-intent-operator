#!/bin/bash
#
# E2 and KPM xApp Verification Script
# Verifies deployment status of E2 test client and KPM xApp
#

set -e

echo "=================================================="
echo "E2 and KPM xApp Deployment Verification"
echo "=================================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check function
check() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $1"
    else
        echo -e "${RED}✗${NC} $1"
        return 1
    fi
}

echo "1. Checking ricxapp namespace..."
kubectl get namespace ricxapp > /dev/null 2>&1
check "ricxapp namespace exists"
echo ""

echo "2. Checking E2 Test Client..."
E2_TEST_POD=$(kubectl get pods -n ricxapp -l app=e2-test-client -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$E2_TEST_POD" ]; then
    STATUS=$(kubectl get pod -n ricxapp $E2_TEST_POD -o jsonpath='{.status.phase}')
    if [ "$STATUS" == "Running" ]; then
        echo -e "${GREEN}✓${NC} E2 Test Client is Running"
        echo "  Pod: $E2_TEST_POD"
        echo "  IP: $(kubectl get pod -n ricxapp $E2_TEST_POD -o jsonpath='{.status.podIP}')"
    else
        echo -e "${RED}✗${NC} E2 Test Client is not Running (Status: $STATUS)"
    fi
else
    echo -e "${RED}✗${NC} E2 Test Client pod not found"
fi
echo ""

echo "3. Checking KPM xApp..."
KPM_POD=$(kubectl get pods -n ricxapp -l app=ricxapp-kpimon -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$KPM_POD" ]; then
    STATUS=$(kubectl get pod -n ricxapp $KPM_POD -o jsonpath='{.status.phase}')
    if [ "$STATUS" == "Running" ]; then
        echo -e "${GREEN}✓${NC} KPM xApp is Running"
        echo "  Pod: $KPM_POD"
        echo "  IP: $(kubectl get pod -n ricxapp $KPM_POD -o jsonpath='{.status.podIP}')"
    else
        echo -e "${RED}✗${NC} KPM xApp is not Running (Status: $STATUS)"
    fi
else
    echo -e "${RED}✗${NC} KPM xApp pod not found"
fi
echo ""

echo "4. Checking KPM xApp Services..."
kubectl get svc -n ricxapp service-ricxapp-kpimon-http > /dev/null 2>&1
check "KPM xApp HTTP service exists"
kubectl get svc -n ricxapp service-ricxapp-kpimon-rmr > /dev/null 2>&1
check "KPM xApp RMR service exists"
echo ""

echo "5. Testing E2 Manager API connectivity..."
if [ -n "$E2_TEST_POD" ]; then
    E2MGR_TEST=$(kubectl exec -n ricxapp $E2_TEST_POD -- curl -s http://service-ricplt-e2mgr-http.ricplt.svc.cluster.local:3800/v1/nodeb/states 2>&1)
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} E2 Manager API is accessible"
        echo "  Response: $E2MGR_TEST"
    else
        echo -e "${RED}✗${NC} E2 Manager API is not accessible"
    fi
else
    echo -e "${YELLOW}⚠${NC} Skipping E2 Manager test (E2 Test Client not running)"
fi
echo ""

echo "6. Testing KPM xApp health endpoint..."
if [ -n "$KPM_POD" ]; then
    HEALTH=$(kubectl exec -n ricxapp $KPM_POD -- curl -s http://localhost:8080/health 2>&1)
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} KPM xApp health check passed"
        echo "  Response: $HEALTH"
    else
        echo -e "${RED}✗${NC} KPM xApp health check failed"
    fi
else
    echo -e "${YELLOW}⚠${NC} Skipping KPM health test (KPM xApp not running)"
fi
echo ""

echo "7. Checking RIC Platform services..."
kubectl get svc -n ricplt service-ricplt-e2mgr-http > /dev/null 2>&1
check "E2 Manager HTTP service"
kubectl get svc -n ricplt service-ricplt-e2term-sctp-alpha > /dev/null 2>&1
check "E2 Termination SCTP service"
kubectl get svc -n ricplt service-ricplt-submgr-rmr > /dev/null 2>&1
check "Subscription Manager RMR service"
echo ""

echo "=================================================="
echo "Summary"
echo "=================================================="

# Count pods
TOTAL_PODS=$(kubectl get pods -n ricxapp --no-headers | wc -l)
RUNNING_PODS=$(kubectl get pods -n ricxapp --field-selector=status.phase=Running --no-headers | wc -l)

echo "Total Pods in ricxapp: $TOTAL_PODS"
echo "Running Pods: $RUNNING_PODS"
echo ""

if [ "$RUNNING_PODS" -eq "$TOTAL_PODS" ] && [ "$TOTAL_PODS" -gt 0 ]; then
    echo -e "${GREEN}✓ All pods are running successfully!${NC}"
else
    echo -e "${YELLOW}⚠ Some pods are not running. Check 'kubectl get pods -n ricxapp' for details.${NC}"
fi

echo ""
echo "For detailed status, run:"
echo "  kubectl get all -n ricxapp"
echo ""
echo "To view logs:"
echo "  kubectl logs -n ricxapp -l app=e2-test-client --tail=50"
echo "  kubectl logs -n ricxapp -l app=ricxapp-kpimon --tail=50"
echo ""
