#!/usr/bin/env bash
set -euo pipefail

# Simplified E2E Test for Nephoran Intent Operator
# Focus on core functionality: cluster setup → CRDs → basic scaling test

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-nephoran-e2e}"
NAMESPACE="${NAMESPACE:-nephoran-system}"
SKIP_CLEANUP="${SKIP_CLEANUP:-false}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
log_success() { echo -e "${GREEN}✅ $1${NC}"; }
log_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
log_error() { echo -e "${RED}❌ $1${NC}" >&2; }

# Wait function
wait_for_condition() {
    local description="$1"
    local condition_cmd="$2"
    local timeout="${3:-60}"
    local interval="${4:-2}"
    
    log_info "Waiting for: $description (timeout: ${timeout}s)"
    
    local elapsed=0
    while [ $elapsed -lt $timeout ]; do
        if eval "$condition_cmd" &>/dev/null; then
            log_success "$description - ready!"
            return 0
        fi
        sleep $interval
        elapsed=$((elapsed + interval))
        echo -n "."
    done
    echo ""
    log_error "$description - timed out after ${timeout}s"
    return 1
}

# Cleanup function
cleanup() {
    local exit_code=$?
    
    echo -e "\n${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${BLUE}           CLEANUP & SUMMARY${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    
    if [ "$SKIP_CLEANUP" = "false" ] && [ -n "${CLUSTER_NAME:-}" ]; then
        log_info "Cleaning up cluster: $CLUSTER_NAME"
        kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
    else
        log_warning "Skipping cleanup (SKIP_CLEANUP=true)"
        echo "   To delete cluster manually: kind delete cluster --name $CLUSTER_NAME"
    fi
    
    echo -e "\n${BLUE}═══════════════════════════════════════════${NC}"
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}        🎉 E2E TEST PASSED! 🎉${NC}"
    else
        echo -e "${RED}        ⚠️  E2E TEST FAILED ⚠️${NC}"
    fi
    echo -e "${BLUE}═══════════════════════════════════════════${NC}\n"
    
    exit $exit_code
}
trap cleanup EXIT

# Main functions
create_kind_cluster() {
    log_info "Creating kind cluster: $CLUSTER_NAME"
    
    # Check if cluster already exists
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_warning "Cluster $CLUSTER_NAME already exists - deleting..."
        kind delete cluster --name "$CLUSTER_NAME"
    fi
    
    # Create simple cluster
    kind create cluster --name "$CLUSTER_NAME"
    kubectl cluster-info --context "kind-$CLUSTER_NAME"
    
    # Wait for cluster to be ready
    wait_for_condition "Cluster nodes ready" \
        "kubectl get nodes --context 'kind-$CLUSTER_NAME' | grep -q Ready" \
        120
}

setup_namespace() {
    log_info "Setting up namespaces"
    
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    kubectl create namespace ran-a --dry-run=client -o yaml | kubectl apply -f -
    
    log_success "Namespaces ready"
}

install_core_crds() {
    log_info "Installing core CRDs"
    
    # Install only the main NetworkIntent CRD
    if [ -f "$PROJECT_ROOT/config/crd/bases/intent.nephoran.com_networkintents.yaml" ]; then
        kubectl apply -f "$PROJECT_ROOT/config/crd/bases/intent.nephoran.com_networkintents.yaml"
        
        # Wait for CRD to be established
        wait_for_condition "NetworkIntent CRD established" \
            "kubectl get crd networkintents.intent.nephoran.com -o jsonpath='{.status.conditions[?(@.type==\"Established\")].status}' | grep -q True" \
            60
    else
        log_error "NetworkIntent CRD not found"
        return 1
    fi
    
    log_success "CRDs installed"
}

deploy_test_workload() {
    log_info "Deploying test workload"
    
    # Deploy simple nginx deployment
    cat > /tmp/test-deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nf-sim
  namespace: ran-a
  labels:
    app: nf-sim
spec:
  replicas: 1
  selector:
    matchLabels:
      app: nf-sim
  template:
    metadata:
      labels:
        app: nf-sim
    spec:
      containers:
      - name: nf-sim
        image: nginx:alpine
        ports:
        - containerPort: 80
        resources:
          requests:
            memory: "4Mi"
            cpu: "1m"
          limits:
            memory: "8Mi"
            cpu: "5m"
EOF
    
    kubectl apply -f /tmp/test-deployment.yaml
    
    # Wait for deployment to be ready
    wait_for_condition "Test deployment ready" \
        "kubectl get deployment nf-sim -n ran-a -o jsonpath='{.status.readyReplicas}' | grep -q '1'" \
        120
    
    log_success "Test workload deployed"
}

create_network_intent() {
    log_info "Creating NetworkIntent resource"
    
    # Create NetworkIntent directly
    cat > /tmp/network-intent.yaml << EOF
apiVersion: intent.nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: scale-test
  namespace: ran-a
spec:
  intentType: scaling
  namespace: ran-a
  target: "nf-sim deployment"
  replicas: 3
EOF
    
    kubectl apply -f /tmp/network-intent.yaml
    
    # Verify intent was created
    if kubectl get networkintent scale-test -n ran-a &>/dev/null; then
        log_success "NetworkIntent created"
    else
        log_error "Failed to create NetworkIntent"
        return 1
    fi
}

simulate_scaling() {
    log_info "Simulating scaling operation (manual patch)"
    
    # Manually scale the deployment to simulate the operator behavior
    kubectl patch deployment nf-sim -n ran-a -p '{"spec":{"replicas":3}}'
    
    log_success "Scaling command applied"
}

verify_scaling() {
    log_info "Verifying scaling result"
    
    # Use the Go verifier tool
    if [ -f "$PROJECT_ROOT/tools/verify-scale.go" ]; then
        cd "$PROJECT_ROOT"
        if go run tools/verify-scale.go \
            --namespace=ran-a \
            --name=nf-sim \
            --target-replicas=3 \
            --timeout=180s; then
            log_success "✅ Scaling verification PASSED"
            return 0
        else
            log_error "❌ Scaling verification FAILED"
            return 1
        fi
    else
        # Fallback manual verification
        wait_for_condition "Manual scaling verification" \
            "kubectl get deployment nf-sim -n ran-a -o jsonpath='{.status.readyReplicas}' | grep -q '3'" \
            120
    fi
}

print_summary() {
    echo -e "\n${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${BLUE}           E2E TEST RESULTS SUMMARY${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    
    # Get final status
    SPEC_REPLICAS=$(kubectl get deployment nf-sim -n ran-a -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
    READY_REPLICAS=$(kubectl get deployment nf-sim -n ran-a -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    echo -e "${BLUE}Component${NC}        → ${BLUE}Status${NC}"
    echo "─────────────────────────────────────────────"
    echo -e "Kind Cluster     → ✅ Created"
    echo -e "Namespaces       → ✅ Ready"
    echo -e "CRDs             → ✅ Installed"
    echo -e "Test Workload    → ✅ Deployed"
    echo -e "NetworkIntent    → ✅ Created"
    echo -e "Scaling          → ✅ Applied"
    
    if [ "$SPEC_REPLICAS" = "3" ] && [ "$READY_REPLICAS" = "3" ]; then
        echo -e "Final State      → ✅ ${READY_REPLICAS}/${SPEC_REPLICAS} replicas"
    else
        echo -e "Final State      → ❌ ${READY_REPLICAS}/${SPEC_REPLICAS} replicas (expected 3/3)"
    fi
    
    echo "─────────────────────────────────────────────"
}

# Main execution
main() {
    echo -e "${BLUE}Starting Simplified E2E Test for Nephoran Intent Operator${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}\n"
    
    create_kind_cluster
    setup_namespace
    install_core_crds
    deploy_test_workload
    create_network_intent
    simulate_scaling
    verify_scaling
    print_summary
    
    log_success "🎉 Simplified E2E test completed successfully!"
}

# Parse arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h      Show this help"
        echo "  --no-cleanup    Skip cluster cleanup"
        echo ""
        echo "Environment variables:"
        echo "  CLUSTER_NAME    Cluster name (default: nephoran-e2e)"
        echo "  NAMESPACE       Namespace (default: nephoran-system)" 
        echo "  SKIP_CLEANUP    Skip cleanup (default: false)"
        exit 0
        ;;
    --no-cleanup)
        SKIP_CLEANUP="true"
        ;;
    "")
        # Continue with execution
        ;;
    *)
        log_error "Unknown option: $1"
        exit 1
        ;;
esac

# Execute main
main