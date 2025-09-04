#!/usr/bin/env bash
set -euo pipefail

# Scaling Debug Helper Script for Nephoran E2E Tests
# This script helps diagnose nginx pod scaling issues in kind clusters

# Configuration
NAMESPACE="${1:-ran-a}"
DEPLOYMENT="${2:-nf-sim}"
CLUSTER_NAME="${3:-nephoran-e2e}"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}" >&2
}

echo -e "${BLUE}═══════════════════════════════════════════${NC}"
echo -e "${BLUE}       Nephoran Scaling Debug Report${NC}"
echo -e "${BLUE}═══════════════════════════════════════════${NC}"
echo -e "Namespace: ${NAMESPACE}"
echo -e "Deployment: ${DEPLOYMENT}"
echo -e "Cluster: ${CLUSTER_NAME}"
echo -e "Timestamp: $(date)"
echo -e "${BLUE}═══════════════════════════════════════════${NC}"

# 1. Cluster Health Check
echo -e "\n${YELLOW}1. CLUSTER HEALTH CHECK${NC}"
echo "───────────────────────────────────────────"

if ! kubectl cluster-info --context "kind-${CLUSTER_NAME}" >/dev/null 2>&1; then
    log_error "Cannot connect to cluster kind-${CLUSTER_NAME}"
    exit 1
fi

log_success "Cluster connection established"

# Check nodes
echo -e "\nNode Status:"
kubectl get nodes --context "kind-${CLUSTER_NAME}" -o wide

echo -e "\nNode Resource Usage:"
kubectl top nodes --context "kind-${CLUSTER_NAME}" 2>/dev/null || log_warning "Metrics server not available"

echo -e "\nDetailed Node Resource Allocation:"
kubectl describe nodes --context "kind-${CLUSTER_NAME}" | grep -A 10 "Allocated resources:" | head -20

# Check node conditions
echo -e "\nNode Conditions:"
kubectl get nodes --context "kind-${CLUSTER_NAME}" -o json | jq -r '.items[] | "\(.metadata.name): " + (.status.conditions[] | select(.type=="Ready") | "\(.type)=\(.status)")' 2>/dev/null || echo "jq not available for JSON parsing"

# 2. Namespace and Deployment Check
echo -e "\n${YELLOW}2. DEPLOYMENT STATUS${NC}"
echo "───────────────────────────────────────────"

if ! kubectl get namespace "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" >/dev/null 2>&1; then
    log_error "Namespace ${NAMESPACE} does not exist"
    exit 1
fi

log_success "Namespace ${NAMESPACE} exists"

# Deployment details
echo -e "\nDeployment Overview:"
kubectl get deployment "${DEPLOYMENT}" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o wide 2>/dev/null || log_error "Deployment ${DEPLOYMENT} not found"

echo -e "\nDeployment Detailed Status:"
kubectl describe deployment "${DEPLOYMENT}" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" 2>/dev/null || log_error "Cannot describe deployment"

# ReplicaSet details
echo -e "\nReplicaSet Status:"
kubectl get rs -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -l app="${DEPLOYMENT}" 2>/dev/null || log_warning "No ReplicaSets found"

# 3. Pod Analysis
echo -e "\n${YELLOW}3. POD ANALYSIS${NC}"
echo "───────────────────────────────────────────"

echo -e "\nAll Pods for app=${DEPLOYMENT}:"
kubectl get pods -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -l app="${DEPLOYMENT}" -o wide

echo -e "\nPod Phase Distribution:"
kubectl get pods -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -l app="${DEPLOYMENT}" --no-headers | awk '{print $3}' | sort | uniq -c

# Detailed pod inspection
echo -e "\nDetailed Pod Status:"
for pod in $(kubectl get pods -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -l app="${DEPLOYMENT}" --no-headers | awk '{print $1}'); do
    if [ -n "$pod" ]; then
        echo -e "\n--- Pod: $pod ---"
        
        # Pod ready conditions
        ready_status=$(kubectl get pod "$pod" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "Unknown")
        phase=$(kubectl get pod "$pod" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
        
        echo "Phase: $phase, Ready: $ready_status"
        
        # Container statuses
        echo "Container Status:"
        kubectl get pod "$pod" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o jsonpath='{.status.containerStatuses[*].name}' 2>/dev/null | while read -r container; do
            if [ -n "$container" ]; then
                ready=$(kubectl get pod "$pod" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o jsonpath="{.status.containerStatuses[?(@.name=='$container')].ready}" 2>/dev/null || echo "unknown")
                restarts=$(kubectl get pod "$pod" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o jsonpath="{.status.containerStatuses[?(@.name=='$container')].restartCount}" 2>/dev/null || echo "unknown")
                echo "  $container: ready=$ready, restarts=$restarts"
            fi
        done
        
        # Show failing pods details
        if [[ "$phase" != "Running" || "$ready_status" != "True" ]]; then
            echo "Pod Events (last 10):"
            kubectl get events --context "kind-${CLUSTER_NAME}" -n "${NAMESPACE}" --field-selector involvedObject.name="$pod" --sort-by='.lastTimestamp' | tail -10
            
            echo "Pod Description:"
            kubectl describe pod "$pod" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" | tail -20
        fi
    fi
done

# 4. Resource Analysis
echo -e "\n${YELLOW}4. RESOURCE ANALYSIS${NC}"
echo "───────────────────────────────────────────"

echo -e "\nNamespace Resource Quotas:"
kubectl get resourcequota -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" 2>/dev/null || echo "No resource quotas found"

echo -e "\nPod Resource Usage:"
kubectl top pods -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -l app="${DEPLOYMENT}" 2>/dev/null || log_warning "Pod metrics not available"

echo -e "\nNode Allocatable Resources:"
kubectl describe nodes --context "kind-${CLUSTER_NAME}" | grep -A 5 "Allocatable:" || log_warning "Cannot get node resources"

echo -e "\nResource Request Analysis:"
if kubectl get deployment "${DEPLOYMENT}" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" >/dev/null 2>&1; then
    DESIRED_REPLICAS=$(kubectl get deployment "${DEPLOYMENT}" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
    POD_MEM_REQUEST=$(kubectl get deployment "${DEPLOYMENT}" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o jsonpath='{.spec.template.spec.containers[0].resources.requests.memory}' 2>/dev/null || echo "unknown")
    POD_CPU_REQUEST=$(kubectl get deployment "${DEPLOYMENT}" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o jsonpath='{.spec.template.spec.containers[0].resources.requests.cpu}' 2>/dev/null || echo "unknown")
    
    echo "Deployment resource requirements:"
    echo "  Desired replicas: ${DESIRED_REPLICAS}"
    echo "  Per-pod memory request: ${POD_MEM_REQUEST}"
    echo "  Per-pod CPU request: ${POD_CPU_REQUEST}"
    
    if [ "${POD_MEM_REQUEST}" != "unknown" ] && [ "${DESIRED_REPLICAS}" != "0" ]; then
        echo "  Total memory needed: ~$(echo "${POD_MEM_REQUEST}" | sed 's/Mi/* ${DESIRED_REPLICAS}/')Mi minimum"
    fi
fi

# 5. Network Analysis
echo -e "\n${YELLOW}5. NETWORK ANALYSIS${NC}"
echo "───────────────────────────────────────────"

echo -e "\nServices:"
kubectl get svc -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o wide

echo -e "\nEndpoints:"
kubectl get endpoints -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o wide

# 6. Recent Events
echo -e "\n${YELLOW}6. RECENT EVENTS${NC}"
echo "───────────────────────────────────────────"

echo -e "\nRecent Namespace Events (last 20):"
kubectl get events -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" --sort-by='.lastTimestamp' | tail -20

echo -e "\nRecent Cluster Events (last 10):"
kubectl get events --all-namespaces --context "kind-${CLUSTER_NAME}" --sort-by='.lastTimestamp' | tail -10

# 7. Scaling Recommendations
echo -e "\n${YELLOW}7. SCALING RECOMMENDATIONS${NC}"
echo "───────────────────────────────────────────"

# Check current vs desired replicas
DESIRED_REPLICAS=$(kubectl get deployment "${DEPLOYMENT}" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
READY_REPLICAS=$(kubectl get deployment "${DEPLOYMENT}" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
AVAILABLE_REPLICAS=$(kubectl get deployment "${DEPLOYMENT}" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -o jsonpath='{.status.availableReplicas}' 2>/dev/null || echo "0")

echo "Current Status: desired=${DESIRED_REPLICAS}, ready=${READY_REPLICAS}, available=${AVAILABLE_REPLICAS}"

if [ "$READY_REPLICAS" -lt "$DESIRED_REPLICAS" ]; then
    echo -e "\n${RED}ISSUES DETECTED:${NC}"
    echo "- Ready replicas ($READY_REPLICAS) < Desired replicas ($DESIRED_REPLICAS)"
    
    # Common troubleshooting steps
    echo -e "\n${BLUE}TROUBLESHOOTING STEPS:${NC}"
    echo "1. Check resource constraints (CPU/memory limits)"
    echo "2. Verify image pull policy and availability"
    echo "3. Check node capacity and scheduling constraints"
    echo "4. Review readiness/liveness probe configuration"
    echo "5. Check for network policies blocking traffic"
    
    # Specific checks with detailed analysis
    PENDING_PODS=$(kubectl get pods -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -l app="${DEPLOYMENT}" --field-selector=status.phase=Pending --no-headers | wc -l)
    if [ "$PENDING_PODS" -gt 0 ]; then
        echo -e "\n${RED}Found $PENDING_PODS pending pods - analyzing scheduling issues${NC}"
        
        # Get detailed reasons for pending pods
        echo "Pending pod analysis:"
        kubectl get pods -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -l app="${DEPLOYMENT}" --field-selector=status.phase=Pending --no-headers | while read -r pod_line; do
            if [ -n "$pod_line" ]; then
                pod_name=$(echo "$pod_line" | awk '{print $1}')
                echo "--- Pod: $pod_name ---"
                kubectl describe pod "$pod_name" -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" 2>/dev/null | grep -A 10 "Events:" | tail -10
                echo ""
            fi
        done
        
        # Check if this is a resource constraint issue
        echo "Resource constraint analysis:"
        kubectl describe nodes --context "kind-${CLUSTER_NAME}" | grep -E "(OutOfmemory|OutOfcpu|Insufficient)" || echo "No obvious resource constraints detected at node level"
    fi
    
    FAILED_PODS=$(kubectl get pods -n "${NAMESPACE}" --context "kind-${CLUSTER_NAME}" -l app="${DEPLOYMENT}" --field-selector=status.phase=Failed --no-headers | wc -l)
    if [ "$FAILED_PODS" -gt 0 ]; then
        echo -e "\n${RED}Found $FAILED_PODS failed pods - check logs and events${NC}"
    fi
    
else
    log_success "Scaling appears to be working correctly"
    
    # Even when successful, provide optimization recommendations
    echo -e "\n${BLUE}OPTIMIZATION RECOMMENDATIONS:${NC}"
    echo "1. Current resource requests appear appropriate for kind cluster"
    echo "2. Consider monitoring pod startup times during scaling"
    echo "3. Verify readiness probe timing for faster availability"
fi

# Additional recommendations for kind clusters
echo -e "\n${BLUE}KIND CLUSTER SPECIFIC TIPS:${NC}"
echo "1. Use minimal resource requests (8-16Mi memory, 5-10m CPU)"
echo "2. Allow pod co-location on nodes (remove anti-affinity)"
echo "3. Enable control-plane scheduling if needed"
echo "4. Use faster probe intervals for quicker scaling"
echo "5. Pre-pull images to avoid ImagePullBackOff delays"

echo -e "\n${BLUE}═══════════════════════════════════════════${NC}"
echo -e "${BLUE}         Debug Report Complete${NC}"
echo -e "${BLUE}═══════════════════════════════════════════${NC}"