#!/bin/bash
# Nephoran Intent Operator - System Health Check Script
# Version: 1.0
# Date: 2026-02-21
# Purpose: Verify all components are deployed and operational

set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNING_CHECKS=0

# Function to print section headers
print_header() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Function to check status
check_status() {
    local component=$1
    local check_cmd=$2
    local expected=$3

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    if result=$(eval "$check_cmd" 2>&1); then
        if [[ "$result" == *"$expected"* ]] || [[ -z "$expected" ]]; then
            echo -e "${GREEN}✅ PASS${NC} - $component"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
            return 0
        else
            echo -e "${RED}❌ FAIL${NC} - $component (Expected: $expected, Got: $result)"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
            return 1
        fi
    else
        echo -e "${RED}❌ FAIL${NC} - $component (Command failed: $check_cmd)"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
        return 1
    fi
}

# Function to check optional status
check_optional() {
    local component=$1
    local check_cmd=$2
    local expected=$3

    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    if result=$(eval "$check_cmd" 2>&1); then
        if [[ "$result" == *"$expected"* ]] || [[ -z "$expected" ]]; then
            echo -e "${GREEN}✅ PASS${NC} - $component"
            PASSED_CHECKS=$((PASSED_CHECKS + 1))
            return 0
        else
            echo -e "${YELLOW}⚠️  WARN${NC} - $component (Optional: $expected not found)"
            WARNING_CHECKS=$((WARNING_CHECKS + 1))
            return 2
        fi
    else
        echo -e "${YELLOW}⚠️  WARN${NC} - $component (Optional component not available)"
        WARNING_CHECKS=$((WARNING_CHECKS + 1))
        return 2
    fi
}

# Start health check
print_header "🏥 NEPHORAN INTENT OPERATOR - SYSTEM HEALTH CHECK"
echo "Timestamp: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
echo "Kubernetes Version: $(kubectl version --short 2>/dev/null | grep Server || echo 'N/A')"
echo ""

# 1. Kubernetes Cluster
print_header "1️⃣  KUBERNETES CLUSTER"
check_status "Cluster API Reachable" "kubectl cluster-info" "running"
check_status "Node Status" "kubectl get nodes -o jsonpath='{.items[0].status.conditions[?(@.type==\"Ready\")].status}'" "True"
check_status "Node Version" "kubectl get nodes -o jsonpath='{.items[0].status.nodeInfo.kubeletVersion}'" "v1.35"

# 2. Core Infrastructure
print_header "2️⃣  CORE INFRASTRUCTURE"

# GPU Operator
check_status "GPU Operator Namespace" "kubectl get namespace gpu-operator -o name" "namespace/gpu-operator"
check_status "GPU Operator Running" "kubectl get deployment -n gpu-operator -o jsonpath='{.items[*].status.conditions[?(@.type==\"Available\")].status}' | grep -o True | wc -l | awk '{if(\$1>=1) print \"ok\"; else print \"fail\"}'" "ok"
check_status "GPU Feature Discovery" "kubectl get ds -n gpu-operator gpu-feature-discovery -o jsonpath='{.status.numberReady}'" "1"

# Monitoring
check_status "Monitoring Namespace" "kubectl get namespace monitoring -o name" "namespace/monitoring"
check_status "Prometheus Running" "kubectl get statefulset -n monitoring prometheus-prometheus-kube-prometheus-prometheus -o jsonpath='{.status.readyReplicas}'" "1"
check_status "Grafana Running" "kubectl get deployment -n monitoring prometheus-grafana -o jsonpath='{.status.readyReplicas}'" "1"
check_status "Alertmanager Running" "kubectl get statefulset -n monitoring alertmanager-prometheus-kube-prometheus-alertmanager -o jsonpath='{.status.readyReplicas}'" "1"

# 3. AI/ML Processing Layer
print_header "3️⃣  AI/ML PROCESSING LAYER"

# Weaviate
check_status "Weaviate Namespace" "kubectl get namespace weaviate -o name" "namespace/weaviate"
check_status "Weaviate StatefulSet Ready" "kubectl get statefulset -n weaviate weaviate -o jsonpath='{.status.readyReplicas}'" "1"
check_status "Weaviate HTTP Service" "kubectl get service -n weaviate weaviate -o jsonpath='{.spec.ports[0].port}'" "80"
check_status "Weaviate gRPC Service" "kubectl get service -n weaviate weaviate-grpc -o jsonpath='{.spec.ports[0].port}'" "50051"

# Ollama
check_status "Ollama Namespace" "kubectl get namespace ollama -o name" "namespace/ollama"
check_status "Ollama Deployment Ready" "kubectl get deployment -n ollama ollama -o jsonpath='{.status.readyReplicas}'" "1"
check_status "Ollama Service Port" "kubectl get service -n ollama ollama -o jsonpath='{.spec.ports[0].port}'" "11434"

# RAG Service
check_status "RAG Service Namespace" "kubectl get namespace rag-service -o name" "namespace/rag-service"
check_status "RAG Service Deployment Ready" "kubectl get deployment -n rag-service rag-service -o jsonpath='{.status.readyReplicas}'" "1"
check_status "RAG Service Port" "kubectl get service -n rag-service rag-service -o jsonpath='{.spec.ports[0].port}'" "8000"

# 4. RAG Service Health
print_header "4️⃣  RAG SERVICE HEALTH"
RAG_POD=$(kubectl get pod -n rag-service -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [[ -n "$RAG_POD" ]]; then
    check_status "RAG Health Endpoint" "kubectl exec -n rag-service $RAG_POD -- curl -s http://localhost:8000/health | jq -r '.status'" "ok"
    check_status "RAG Weaviate Connection" "kubectl exec -n rag-service $RAG_POD -- curl -s http://localhost:8000/stats | jq -r '.health.weaviate_ready'" "true"
    check_status "RAG Knowledge Base" "kubectl exec -n rag-service $RAG_POD -- curl -s http://localhost:8000/stats | jq -r '.health.knowledge_objects > 0'" "true"
    check_status "RAG LLM Model" "kubectl exec -n rag-service $RAG_POD -- curl -s http://localhost:8000/stats | jq -r '.health.model'" "llama3.1"
else
    echo -e "${RED}❌ FAIL${NC} - RAG Service Pod not found"
    FAILED_CHECKS=$((FAILED_CHECKS + 4))
    TOTAL_CHECKS=$((TOTAL_CHECKS + 4))
fi

# 5. O-RAN RIC Platform
print_header "5️⃣  O-RAN RIC PLATFORM"

check_status "RIC Platform Namespace" "kubectl get namespace ricplt -o name" "namespace/ricplt"
check_status "A1 Mediator Running" "kubectl get deployment -n ricplt deployment-ricplt-a1mediator -o jsonpath='{.status.readyReplicas}'" "1"
check_status "E2 Manager Running" "kubectl get deployment -n ricplt deployment-ricplt-e2mgr -o jsonpath='{.status.readyReplicas}'" "1"
check_status "E2 Term Running" "kubectl get deployment -n ricplt deployment-ricplt-e2term-alpha -o jsonpath='{.status.readyReplicas}'" "1"
check_status "O1 Mediator Running" "kubectl get deployment -n ricplt deployment-ricplt-o1mediator -o jsonpath='{.status.readyReplicas}'" "1"
check_status "DBAAS Server Running" "kubectl get statefulset -n ricplt statefulset-ricplt-dbaas-server -o jsonpath='{.status.readyReplicas}'" "1"

# 6. Free5GC (5G Core)
print_header "6️⃣  FREE5GC (5G CORE)"

check_status "Free5GC Namespace" "kubectl get namespace free5gc -o name" "namespace/free5gc"
check_status "MongoDB Running" "kubectl get deployment -n free5gc mongodb -o jsonpath='{.status.readyReplicas}'" "1"
check_status "AMF Running" "kubectl get deployment -n free5gc free5gc-free5gc-amf-amf -o jsonpath='{.status.readyReplicas}'" "1"
check_status "SMF Running" "kubectl get deployment -n free5gc free5gc-free5gc-smf-smf -o jsonpath='{.status.readyReplicas}'" "1"
check_status "UPF Running" "kubectl get deployment -n free5gc upf2-free5gc-upf-upf2 -o jsonpath='{.status.readyReplicas}'" "1"
check_status "NRF Running" "kubectl get deployment -n free5gc free5gc-free5gc-nrf-nrf -o jsonpath='{.status.readyReplicas}'" "1"
check_status "UDM Running" "kubectl get deployment -n free5gc free5gc-free5gc-udm-udm -o jsonpath='{.status.readyReplicas}'" "1"
check_status "UDR Running" "kubectl get deployment -n free5gc free5gc-free5gc-udr-udr -o jsonpath='{.status.readyReplicas}'" "1"
check_status "AUSF Running" "kubectl get deployment -n free5gc free5gc-free5gc-ausf-ausf -o jsonpath='{.status.readyReplicas}'" "1"
check_status "PCF Running" "kubectl get deployment -n free5gc free5gc-free5gc-pcf-pcf -o jsonpath='{.status.readyReplicas}'" "1"
check_status "NSSF Running" "kubectl get deployment -n free5gc free5gc-free5gc-nssf-nssf -o jsonpath='{.status.readyReplicas}'" "1"
check_status "WebUI Running" "kubectl get deployment -n free5gc free5gc-free5gc-webui-webui -o jsonpath='{.status.readyReplicas}'" "1"

# 7. UERANSIM (UE/gNB Simulator)
print_header "7️⃣  UERANSIM (UE/gNB SIMULATOR)"

check_status "UERANSIM gNB Running" "kubectl get deployment -n free5gc ueransim-gnb -o jsonpath='{.status.readyReplicas}'" "1"
check_status "UERANSIM UE Running" "kubectl get deployment -n free5gc ueransim-ue -o jsonpath='{.status.readyReplicas}'" "1"

# 8. Custom Resources (CRDs)
print_header "8️⃣  CUSTOM RESOURCES (CRDs)"

check_status "NetworkIntent CRD (intent.nephoran.com)" "kubectl get crd networkintents.intent.nephoran.com -o name" "networkintents.intent.nephoran.com"
check_status "NetworkIntent CRD (nephoran.com)" "kubectl get crd networkintents.nephoran.com -o name" "networkintents.nephoran.com"
check_status "CNFDeployment CRD" "kubectl get crd cnfdeployments.nephoran.com -o name" "cnfdeployments.nephoran.com"
check_status "IntentProcessing CRD" "kubectl get crd intentprocessings.nephoran.com -o name" "intentprocessings.nephoran.com"

# 9. Intent Operator (Optional - may be scaled to 0)
print_header "9️⃣  INTENT OPERATOR (OPTIONAL)"

check_optional "Nephoran Operator Namespace" "kubectl get namespace nephoran-system -o name" "namespace/nephoran-system"
check_optional "Nephoran Operator Deployment" "kubectl get deployment -n nephoran-system nephoran-operator-controller-manager -o name" "deployment.apps/nephoran-operator-controller-manager"

# 10. Helm Releases
print_header "🔟 HELM RELEASES"

check_status "GPU Operator Helm" "helm list -n gpu-operator -o json | jq -r '.[0].status'" "deployed"
check_status "Weaviate Helm" "helm list -n weaviate -o json | jq -r '.[0].status'" "deployed"
check_status "Ollama Helm" "helm list -n ollama -o json | jq -r '.[0].status'" "deployed"
check_status "MongoDB Helm" "helm list -n free5gc -o json | jq -r '.[] | select(.name==\"mongodb\") | .status'" "deployed"
check_status "Free5GC Helm" "helm list -n free5gc -o json | jq -r '.[] | select(.name==\"free5gc\") | .status'" "deployed"
check_status "Prometheus Helm" "helm list -n monitoring -o json | jq -r '.[0].status'" "deployed"

# 11. E2E Data Flow Test
print_header "1️⃣1️⃣  E2E DATA FLOW TEST"

if [[ -n "$RAG_POD" ]]; then
    echo "Testing natural language → RAG → structured intent..."
    INTENT_TEST=$(kubectl exec -n rag-service $RAG_POD -- curl -s -X POST http://localhost:8000/process_intent \
        -H "Content-Type: application/json" \
        -d '{"intent":"Deploy 2 replicas of SMF"}' | jq -r '.status')

    if [[ "$INTENT_TEST" == "completed" ]]; then
        echo -e "${GREEN}✅ PASS${NC} - E2E Intent Processing"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    else
        echo -e "${RED}❌ FAIL${NC} - E2E Intent Processing (Status: $INTENT_TEST)"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    fi
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
else
    echo -e "${YELLOW}⚠️  SKIP${NC} - E2E test (RAG pod not available)"
fi

# Summary
print_header "📊 HEALTH CHECK SUMMARY"
echo -e "Total Checks:   ${BLUE}$TOTAL_CHECKS${NC}"
echo -e "Passed:         ${GREEN}$PASSED_CHECKS${NC}"
echo -e "Failed:         ${RED}$FAILED_CHECKS${NC}"
echo -e "Warnings:       ${YELLOW}$WARNING_CHECKS${NC}"
echo ""

PASS_RATE=$(awk "BEGIN {printf \"%.1f\", ($PASSED_CHECKS/$TOTAL_CHECKS)*100}")
echo -e "Pass Rate:      ${GREEN}${PASS_RATE}%${NC}"
echo ""

if [[ $FAILED_CHECKS -eq 0 ]]; then
    echo -e "${GREEN}✅ ALL CRITICAL COMPONENTS HEALTHY${NC}"
    exit 0
elif [[ $FAILED_CHECKS -le 2 ]]; then
    echo -e "${YELLOW}⚠️  SYSTEM PARTIALLY HEALTHY (${FAILED_CHECKS} failures)${NC}"
    exit 1
else
    echo -e "${RED}❌ SYSTEM UNHEALTHY (${FAILED_CHECKS} failures)${NC}"
    exit 2
fi
