#!/bin/bash
# Quick API test script for intent-ingest service

set -euo pipefail

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
NAMESPACE="nephoran-intent"
SERVICE="intent-ingest-service"
PORT=8080

echo -e "${CYAN}=== Nephoran Intent API Test ===${NC}\n"

# Check if service exists
if ! kubectl get svc -n "${NAMESPACE}" "${SERVICE}" &>/dev/null; then
    echo -e "${RED}Error: Service ${SERVICE} not found in namespace ${NAMESPACE}${NC}"
    echo "Please run ./deploy.sh first"
    exit 1
fi

# Setup port-forward
echo -e "${CYAN}Setting up port-forward...${NC}"
pkill -f "kubectl.*port-forward.*${PORT}" || true
kubectl port-forward -n "${NAMESPACE}" "svc/${SERVICE}" "${PORT}:${PORT}" > /dev/null 2>&1 &
PF_PID=$!
sleep 2

# Cleanup on exit
trap "kill ${PF_PID} 2>/dev/null || true" EXIT

# Test 1: Health check
echo -e "\n${CYAN}Test 1: Health Check${NC}"
if curl -s "http://localhost:${PORT}/healthz" | grep -q "ok"; then
    echo -e "${GREEN}✓ Health check passed${NC}"
else
    echo -e "${RED}✗ Health check failed${NC}"
fi

# Test 2: Scaling intent
echo -e "\n${CYAN}Test 2: Scaling Intent${NC}"
RESPONSE=$(curl -s -X POST "http://localhost:${PORT}/intent" \
    -H "Content-Type: text/plain" \
    -d "scale nf-sim to 5 in ns ran-a")

if echo "${RESPONSE}" | jq . &>/dev/null; then
    echo -e "${GREEN}✓ Scaling intent processed${NC}"
    echo "${RESPONSE}" | jq .
else
    echo -e "${RED}✗ Scaling intent failed${NC}"
    echo "${RESPONSE}"
fi

# Test 3: JSON intent
echo -e "\n${CYAN}Test 3: JSON Intent${NC}"
JSON_PAYLOAD='{
  "intent_type": "scaling",
  "target": "nginx-deployment",
  "namespace": "default",
  "replicas": 3,
  "source": "user",
  "status": "pending"
}'

RESPONSE=$(curl -s -X POST "http://localhost:${PORT}/intent" \
    -H "Content-Type: application/json" \
    -d "${JSON_PAYLOAD}")

if echo "${RESPONSE}" | jq . &>/dev/null; then
    echo -e "${GREEN}✓ JSON intent processed${NC}"
    echo "${RESPONSE}" | jq .
else
    echo -e "${RED}✗ JSON intent failed${NC}"
    echo "${RESPONSE}"
fi

echo -e "\n${CYAN}=== Tests Complete ===${NC}\n"
