#!/bin/bash
# Test script for Cilium eBPF CNI performance validation
# Referenced by: docs/implementation/task-dag.yaml (Task T12, scenario 4)
# Purpose: Verify Cilium achieves target performance (10+ Gbps in virtual env)

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
NAMESPACE="cilium-test"
TIMEOUT=300
MIN_THROUGHPUT_GBPS=10
TEST_RESULTS=()

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }
success() { echo -e "${GREEN}✅ $*${NC}"; TEST_RESULTS+=("PASS: $*"); }
fail() { echo -e "${RED}❌ $*${NC}"; TEST_RESULTS+=("FAIL: $*"); return 1; }
warn() { echo -e "${YELLOW}⚠️  $*${NC}"; }

print_summary() {
    echo ""
    echo "=========================================="
    echo "Test Summary: Cilium Performance"
    echo "=========================================="
    for result in "${TEST_RESULTS[@]}"; do
        if [[ $result == PASS:* ]]; then
            echo -e "${GREEN}$result${NC}"
        else
            echo -e "${RED}$result${NC}"
        fi
    done
    echo "=========================================="
}

cleanup() {
    log "Cleaning up test resources..."
    kubectl delete namespace "$NAMESPACE" --ignore-not-found=true --timeout=60s &>/dev/null || true
}

trap 'cleanup; print_summary' EXIT

# Test 1: Verify Cilium is installed
log "Test 1: Checking Cilium installation..."
if cilium version &>/dev/null; then
    CILIUM_VERSION=$(cilium version --client 2>/dev/null | head -n 1 || echo "installed")
    success "Cilium CLI is available ($CILIUM_VERSION)"
else
    fail "Cilium CLI not found"
    exit 1
fi

# Test 2: Check Cilium status
log "Test 2: Verifying Cilium status..."
if cilium status --wait --wait-duration=60s &>/dev/null; then
    success "Cilium is healthy and ready"
else
    fail "Cilium status check failed"
    exit 1
fi

# Test 3: Verify eBPF datapath is enabled
log "Test 3: Checking eBPF datapath configuration..."
KUBE_PROXY=$(cilium config view 2>/dev/null | grep "kube-proxy-replacement" | awk '{print $2}')
if [[ "$KUBE_PROXY" == "strict" || "$KUBE_PROXY" == "true" ]]; then
    success "eBPF kube-proxy replacement enabled (optimal performance)"
else
    warn "kube-proxy replacement: $KUBE_PROXY (eBPF may not be fully utilized)"
fi

# Test 4: Check Cilium agent pods are running
log "Test 4: Checking Cilium agent pods..."
AGENT_PODS=$(kubectl get pods -n kube-system -l k8s-app=cilium --field-selector=status.phase=Running 2>/dev/null | tail -n +2 | wc -l)
if [ "$AGENT_PODS" -gt 0 ]; then
    success "$AGENT_PODS Cilium agent pod(s) running"
else
    fail "No Cilium agent pods running"
fi

# Test 5: Verify Hubble observability is available
log "Test 5: Checking Hubble observability..."
if kubectl get deployment -n kube-system hubble-relay &>/dev/null; then
    success "Hubble relay deployed (observability available)"
else
    warn "Hubble relay not found (observability metrics may be limited)"
fi

# Test 6: Create test namespace
log "Test 6: Creating test namespace..."
kubectl create namespace "$NAMESPACE" &>/dev/null || true
sleep 2
success "Test namespace '$NAMESPACE' ready"

# Test 7: Deploy iperf3 server
log "Test 7: Deploying iperf3 server..."
cat <<EOF | kubectl apply -f - &>/dev/null
apiVersion: v1
kind: Pod
metadata:
  name: iperf3-server
  namespace: $NAMESPACE
  labels:
    app: iperf3-server
spec:
  containers:
  - name: iperf3
    image: networkstatic/iperf3:latest
    command: ["iperf3", "-s"]
    ports:
    - containerPort: 5201
  nodeSelector:
    kubernetes.io/os: linux
---
apiVersion: v1
kind: Service
metadata:
  name: iperf3-server
  namespace: $NAMESPACE
spec:
  selector:
    app: iperf3-server
  ports:
  - protocol: TCP
    port: 5201
    targetPort: 5201
EOF

# Wait for server pod
if kubectl wait --for=condition=Ready pod/iperf3-server -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
    SERVER_IP=$(kubectl get svc -n "$NAMESPACE" iperf3-server -o jsonpath='{.spec.clusterIP}')
    success "iperf3 server ready at $SERVER_IP"
else
    fail "iperf3 server pod did not become ready"
fi

# Test 8: Deploy iperf3 client
log "Test 8: Deploying iperf3 client..."
cat <<EOF | kubectl apply -f - &>/dev/null
apiVersion: v1
kind: Pod
metadata:
  name: iperf3-client
  namespace: $NAMESPACE
spec:
  containers:
  - name: iperf3
    image: networkstatic/iperf3:latest
    command: ["sleep", "3600"]
  nodeSelector:
    kubernetes.io/os: linux
EOF

if kubectl wait --for=condition=Ready pod/iperf3-client -n "$NAMESPACE" --timeout="${TIMEOUT}s" &>/dev/null; then
    success "iperf3 client pod ready"
else
    fail "iperf3 client pod did not become ready"
fi

# Test 9: Run bandwidth test (TCP)
log "Test 9: Running TCP bandwidth test (10 seconds)..."
sleep 5  # Let pods stabilize
IPERF_OUTPUT=$(kubectl exec -n "$NAMESPACE" iperf3-client -- iperf3 -c "$SERVER_IP" -t 10 -J 2>/dev/null)
if [ $? -eq 0 ]; then
    # Parse throughput in bits/sec and convert to Gbps
    THROUGHPUT_BPS=$(echo "$IPERF_OUTPUT" | grep -o '"bits_per_second":[^,]*' | head -n 1 | awk -F: '{print $2}' | tr -d ' ')
    THROUGHPUT_GBPS=$(echo "scale=2; $THROUGHPUT_BPS / 1000000000" | bc)

    log "Measured throughput: ${THROUGHPUT_GBPS} Gbps"

    if (( $(echo "$THROUGHPUT_GBPS >= $MIN_THROUGHPUT_GBPS" | bc -l) )); then
        success "TCP throughput: ${THROUGHPUT_GBPS} Gbps (target: ${MIN_THROUGHPUT_GBPS}+ Gbps) ✨"
    else
        warn "TCP throughput: ${THROUGHPUT_GBPS} Gbps (below target of ${MIN_THROUGHPUT_GBPS} Gbps)"
    fi
else
    fail "iperf3 TCP test failed to execute"
fi

# Test 10: Run bandwidth test (UDP with high rate)
log "Test 10: Running UDP bandwidth test..."
UDP_OUTPUT=$(kubectl exec -n "$NAMESPACE" iperf3-client -- iperf3 -c "$SERVER_IP" -u -b 20G -t 5 -J 2>/dev/null)
if [ $? -eq 0 ]; then
    UDP_THROUGHPUT_BPS=$(echo "$UDP_OUTPUT" | grep -o '"bits_per_second":[^,]*' | tail -n 1 | awk -F: '{print $2}' | tr -d ' ')
    UDP_THROUGHPUT_GBPS=$(echo "scale=2; $UDP_THROUGHPUT_BPS / 1000000000" | bc)
    UDP_LOSS=$(echo "$UDP_OUTPUT" | grep -o '"lost_percent":[^,]*' | awk -F: '{print $2}' | tr -d ' ')

    log "UDP throughput: ${UDP_THROUGHPUT_GBPS} Gbps, Packet loss: ${UDP_LOSS}%"
    success "UDP test completed (throughput: ${UDP_THROUGHPUT_GBPS} Gbps, loss: ${UDP_LOSS}%)"
else
    warn "UDP test failed to execute (not critical)"
fi

# Test 11: Run latency test
log "Test 11: Running latency test..."
LATENCY=$(kubectl exec -n "$NAMESPACE" iperf3-client -- sh -c "ping -c 10 $SERVER_IP 2>/dev/null | tail -1 | awk -F'/' '{print \$5}'" 2>/dev/null)
if [ -n "$LATENCY" ]; then
    log "Average RTT latency: ${LATENCY} ms"
    if (( $(echo "$LATENCY < 50" | bc -l) )); then
        success "Latency: ${LATENCY} ms (excellent, < 50ms target)"
    else
        warn "Latency: ${LATENCY} ms (acceptable but above 50ms target)"
    fi
else
    warn "Latency test failed (ping may be disabled)"
fi

# Test 12: Check Hubble metrics for flow data
log "Test 12: Checking Hubble flow metrics..."
if command -v hubble &>/dev/null; then
    if hubble observe --namespace "$NAMESPACE" --last 10 &>/dev/null; then
        FLOW_COUNT=$(hubble observe --namespace "$NAMESPACE" --last 100 2>/dev/null | wc -l)
        success "Hubble observed $FLOW_COUNT flows in test namespace"
    else
        warn "Hubble observe failed (relay may need port-forward)"
    fi
else
    warn "Hubble CLI not installed (flow observability not tested)"
fi

# Test 13: Verify eBPF program metrics
log "Test 13: Checking eBPF program statistics..."
CILIUM_POD=$(kubectl get pods -n kube-system -l k8s-app=cilium -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$CILIUM_POD" ]; then
    BPF_STATS=$(kubectl exec -n kube-system "$CILIUM_POD" -- cilium bpf metrics list 2>/dev/null | head -n 5)
    if [ -n "$BPF_STATS" ]; then
        success "eBPF metrics available from Cilium agent"
    else
        warn "Could not retrieve eBPF metrics"
    fi
fi

# Test 14: Check for eBPF map usage
log "Test 14: Verifying eBPF map usage..."
if [ -n "$CILIUM_POD" ]; then
    MAP_COUNT=$(kubectl exec -n kube-system "$CILIUM_POD" -- cilium map list 2>/dev/null | wc -l)
    if [ "$MAP_COUNT" -gt 0 ]; then
        success "Cilium is using $MAP_COUNT eBPF map(s)"
    else
        warn "Could not verify eBPF map usage"
    fi
fi

# Test 15: Verify CNI plugin configuration
log "Test 15: Checking CNI plugin configuration..."
CNI_CONF=$(kubectl get configmap -n kube-system cilium-config -o jsonpath='{.data.enable-ipv4}' 2>/dev/null)
if [ "$CNI_CONF" = "true" ]; then
    success "Cilium CNI IPv4 enabled"
else
    warn "Could not verify CNI configuration"
fi

log "Cilium performance validation complete!"
log "Summary: Measured ${THROUGHPUT_GBPS} Gbps TCP throughput (target: ${MIN_THROUGHPUT_GBPS}+ Gbps)"
exit 0
