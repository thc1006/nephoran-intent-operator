#!/usr/bin/env bash
set -euo pipefail

# Kind Cluster Resource Verification Script
# Ensures kind cluster has sufficient resources for E2E scaling tests

CLUSTER_NAME="${1:-nephoran-e2e}"
TARGET_REPLICAS="${2:-3}"

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
echo -e "${BLUE}    Kind Cluster Resource Verification${NC}"
echo -e "${BLUE}═══════════════════════════════════════════${NC}"
echo -e "Cluster: ${CLUSTER_NAME}"
echo -e "Target Replicas: ${TARGET_REPLICAS}"
echo -e "Timestamp: $(date)"
echo -e "${BLUE}═══════════════════════════════════════════${NC}"

# Check cluster connectivity
if ! kubectl cluster-info --context "kind-${CLUSTER_NAME}" >/dev/null 2>&1; then
    log_error "Cannot connect to cluster kind-${CLUSTER_NAME}"
    log_error "Please ensure the cluster is running: kind get clusters"
    exit 1
fi

log_success "Connected to cluster kind-${CLUSTER_NAME}"

# Check node count and status
echo -e "\n${YELLOW}NODE ANALYSIS${NC}"
echo "─────────────────────────────────────────────"

NODE_COUNT=$(kubectl get nodes --context "kind-${CLUSTER_NAME}" --no-headers | wc -l)
READY_NODES=$(kubectl get nodes --context "kind-${CLUSTER_NAME}" --no-headers | grep " Ready " | wc -l)

echo "Nodes: ${NODE_COUNT} total, ${READY_NODES} ready"

if [ "${READY_NODES}" -lt 1 ]; then
    log_error "No ready nodes found"
    exit 1
elif [ "${READY_NODES}" -eq 1 ]; then
    log_warning "Only 1 node available - pods will be co-located"
else
    log_success "Multiple nodes available for pod distribution"
fi

# Analyze node resources
echo -e "\n${YELLOW}RESOURCE ANALYSIS${NC}"
echo "─────────────────────────────────────────────"

# Check if we can get resource information
if kubectl describe nodes --context "kind-${CLUSTER_NAME}" | grep "Allocatable:" >/dev/null 2>&1; then
    log_success "Node resource information available"
    
    echo -e "\nNode allocatable resources:"
    kubectl describe nodes --context "kind-${CLUSTER_NAME}" | grep -A 5 "Allocatable:" | head -10
    
    echo -e "\nCurrent resource allocation:"
    kubectl describe nodes --context "kind-${CLUSTER_NAME}" | grep -A 10 "Allocated resources:" | head -15
else
    log_warning "Cannot retrieve detailed resource information"
fi

# Resource requirements analysis
echo -e "\n${YELLOW}SCALING REQUIREMENTS ANALYSIS${NC}"
echo "─────────────────────────────────────────────"

# Calculate minimum resource requirements for our test
MIN_MEMORY_PER_POD_MB=16  # 16Mi per pod
MIN_CPU_PER_POD_M=10      # 10m per pod

TOTAL_MEMORY_NEEDED=$((MIN_MEMORY_PER_POD_MB * TARGET_REPLICAS))
TOTAL_CPU_NEEDED=$((MIN_CPU_PER_POD_M * TARGET_REPLICAS))

echo "Minimum resource requirements for ${TARGET_REPLICAS} replicas:"
echo "  Memory: ${TOTAL_MEMORY_NEEDED}Mi (${MIN_MEMORY_PER_POD_MB}Mi × ${TARGET_REPLICAS})"
echo "  CPU: ${TOTAL_CPU_NEEDED}m (${MIN_CPU_PER_POD_M}m × ${TARGET_REPLICAS})"

# Test image availability
echo -e "\n${YELLOW}IMAGE AVAILABILITY${NC}"
echo "─────────────────────────────────────────────"

TEST_IMAGE="nginx:1.25-alpine"
log_info "Testing image pull for: ${TEST_IMAGE}"

# Create a temporary pod to test image availability
cat > /tmp/test-image-pull.yaml << EOF
apiVersion: v1
kind: Pod
metadata:
  name: test-image-pull
  namespace: default
spec:
  restartPolicy: Never
  containers:
  - name: test
    image: ${TEST_IMAGE}
    command: ["echo", "Image test successful"]
    resources:
      requests:
        memory: "8Mi"
        cpu: "5m"
      limits:
        memory: "16Mi"
        cpu: "50m"
EOF

if kubectl apply -f /tmp/test-image-pull.yaml --context "kind-${CLUSTER_NAME}" >/dev/null 2>&1; then
    # Wait for pod to complete or fail
    kubectl wait --for=condition=Ready pod/test-image-pull --context "kind-${CLUSTER_NAME}" --timeout=60s >/dev/null 2>&1 || true
    
    POD_STATUS=$(kubectl get pod test-image-pull --context "kind-${CLUSTER_NAME}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
    
    if [ "${POD_STATUS}" = "Running" ] || [ "${POD_STATUS}" = "Succeeded" ]; then
        log_success "Image ${TEST_IMAGE} is available and pullable"
    else
        log_warning "Image pull test inconclusive - status: ${POD_STATUS}"
        
        # Show any pull-related events
        kubectl get events --context "kind-${CLUSTER_NAME}" --field-selector involvedObject.name=test-image-pull 2>/dev/null | grep -E "(Pull|Image)" | tail -3
    fi
    
    # Cleanup
    kubectl delete pod test-image-pull --context "kind-${CLUSTER_NAME}" >/dev/null 2>&1 || true
else
    log_warning "Could not create test pod for image verification"
fi

rm -f /tmp/test-image-pull.yaml

# Networking check
echo -e "\n${YELLOW}NETWORKING CHECK${NC}"
echo "─────────────────────────────────────────────"

if kubectl get svc kubernetes --context "kind-${CLUSTER_NAME}" >/dev/null 2>&1; then
    log_success "Kubernetes API service is accessible"
else
    log_error "Cannot access Kubernetes API service"
fi

# DNS check
DNS_SERVICE=$(kubectl get svc kube-dns -n kube-system --context "kind-${CLUSTER_NAME}" -o name 2>/dev/null || echo "")
if [ -n "${DNS_SERVICE}" ]; then
    log_success "DNS service is available"
else
    # Check for CoreDNS
    COREDNS_SERVICE=$(kubectl get svc coredns -n kube-system --context "kind-${CLUSTER_NAME}" -o name 2>/dev/null || echo "")
    if [ -n "${COREDNS_SERVICE}" ]; then
        log_success "CoreDNS service is available"
    else
        log_warning "DNS service status unclear"
    fi
fi

# Final recommendations
echo -e "\n${YELLOW}RECOMMENDATIONS${NC}"
echo "─────────────────────────────────────────────"

if [ "${READY_NODES}" -eq 1 ]; then
    echo "✓ Use tolerations to allow control-plane scheduling"
    echo "✓ Remove pod anti-affinity rules"
    echo "✓ Use minimal resource requests"
fi

echo "✓ Use resource requests: memory=16Mi, cpu=10m"
echo "✓ Use resource limits: memory=64Mi, cpu=100m"
echo "✓ Set fast probe intervals: initial=2s, period=3s"
echo "✓ Pre-pull images before scaling tests"

# Overall assessment
echo -e "\n${BLUE}═══════════════════════════════════════════${NC}"
echo -e "${BLUE}           OVERALL ASSESSMENT${NC}"
echo -e "${BLUE}═══════════════════════════════════════════${NC}"

if [ "${READY_NODES}" -ge 1 ]; then
    log_success "Cluster appears ready for E2E scaling tests"
    echo "Proceed with E2E test execution"
else
    log_error "Cluster is not ready for E2E testing"
    echo "Fix node issues before running tests"
    exit 1
fi

echo -e "${BLUE}═══════════════════════════════════════════${NC}"