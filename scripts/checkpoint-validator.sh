#!/bin/bash
# Checkpoint Validator Script
# Version: 2.0
# Date: 2026-02-16
# Purpose: Validate completion of implementation phases
# Usage: ./scripts/checkpoint-validator.sh [checkpoint_name]

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LOG_FILE="$PROJECT_ROOT/checkpoint-validation.log"

# Functions
log() {
    echo "[$(date -u +%Y-%m-%dT%H:%M:%S+00:00)] $*" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}✅ $*${NC}" | tee -a "$LOG_FILE"
}

fail() {
    echo -e "${RED}❌ $*${NC}" | tee -a "$LOG_FILE"
}

warn() {
    echo -e "${YELLOW}⚠️  $*${NC}" | tee -a "$LOG_FILE"
}

# Checkpoint validation functions

validate_prerequisites() {
    log "==> Validating Prerequisites Checkpoint..."

    # Hardware checks
    log "Checking hardware requirements..."

    # CPU cores (minimum 16)
    CPU_CORES=$(nproc)
    if [ "$CPU_CORES" -lt 16 ]; then
        fail "CPU: $CPU_CORES cores (minimum 16 required)"
        return 1
    else
        success "CPU: $CPU_CORES cores"
    fi

    # RAM (minimum 64GB)
    RAM_GB=$(free -g | awk '/^Mem:/{print $2}')
    if [ "$RAM_GB" -lt 64 ]; then
        fail "RAM: ${RAM_GB}GB (minimum 64GB required)"
        return 1
    else
        success "RAM: ${RAM_GB}GB"
    fi

    # Disk space (minimum 500GB)
    DISK_GB=$(df -BG / | awk 'NR==2 {print $4}' | tr -d 'G')
    if [ "$DISK_GB" -lt 500 ]; then
        fail "Disk: ${DISK_GB}GB available (minimum 500GB required)"
        return 1
    else
        success "Disk: ${DISK_GB}GB available"
    fi

    # Software checks
    log "Checking software prerequisites..."

    # Ubuntu version
    if ! lsb_release -a 2>/dev/null | grep -q "22.04"; then
        fail "Ubuntu version: $(lsb_release -rs) (22.04 required)"
        return 1
    else
        success "Ubuntu: $(lsb_release -rs)"
    fi

    # Git
    if ! command -v git &> /dev/null; then
        fail "Git not installed"
        return 1
    else
        success "Git: $(git --version | awk '{print $3}')"
    fi

    # Docker or containerd
    if command -v docker &> /dev/null; then
        success "Docker: $(docker --version | awk '{print $3}' | tr -d ',')"
    elif command -v containerd &> /dev/null; then
        success "containerd: $(containerd --version | awk '{print $3}')"
    else
        fail "Neither Docker nor containerd found"
        return 1
    fi

    # Helm
    if ! command -v helm &> /dev/null; then
        fail "Helm not installed"
        return 1
    else
        HELM_VERSION=$(helm version --short | awk '{print $1}' | cut -d'+' -f1)
        success "Helm: $HELM_VERSION"
    fi

    success "All prerequisites validated!"
    return 0
}

validate_infrastructure() {
    log "==> Validating Infrastructure Checkpoint..."

    # Task T1: Kubernetes 1.35.1
    log "Checking Kubernetes installation..."
    if ! command -v kubectl &> /dev/null; then
        fail "kubectl not found"
        return 1
    fi

    if ! kubectl version --short 2>/dev/null | grep -q "v1.35.1"; then
        fail "Kubernetes version mismatch (expected v1.35.1)"
        kubectl version --short 2>/dev/null || true
        return 1
    fi
    success "Kubernetes: v1.35.1"

    if ! kubectl get nodes 2>/dev/null | grep -q "Ready"; then
        fail "No Ready nodes found"
        kubectl get nodes 2>/dev/null || true
        return 1
    fi
    success "Kubernetes nodes: Ready"

    # Task T2: Cilium eBPF
    log "Checking Cilium CNI..."
    if ! command -v cilium &> /dev/null; then
        warn "Cilium CLI not found (optional but recommended)"
    else
        if ! cilium status 2>/dev/null | grep -q "OK"; then
            fail "Cilium status check failed"
            cilium status 2>/dev/null || true
            return 1
        fi
        success "Cilium: OK"
    fi

    # Check Cilium pods
    if ! kubectl get pods -n kube-system -l k8s-app=cilium 2>/dev/null | grep -q "Running"; then
        fail "Cilium pods not running"
        kubectl get pods -n kube-system -l k8s-app=cilium 2>/dev/null || true
        return 1
    fi
    success "Cilium pods: Running"

    # Task T3: GPU Operator (optional)
    log "Checking GPU Operator (optional)..."
    if kubectl get nodes -o yaml 2>/dev/null | grep -q "nvidia.com/dra"; then
        success "GPU Operator with DRA: Detected"
    else
        warn "GPU Operator not detected (optional for LLM workloads)"
    fi

    # Storage class
    log "Checking storage class..."
    if ! kubectl get storageclass 2>/dev/null | grep -q "local-path"; then
        warn "local-path storage class not found (will be needed for persistent volumes)"
    else
        success "Storage class: local-path available"
    fi

    success "Infrastructure checkpoint validated!"
    return 0
}

validate_databases() {
    log "==> Validating Databases Checkpoint..."

    # Task T4: MongoDB
    log "Checking MongoDB..."
    if ! kubectl get pods -n free5gc 2>/dev/null | grep -q "mongodb.*Running"; then
        fail "MongoDB pod not running in free5gc namespace"
        kubectl get pods -n free5gc 2>/dev/null || true
        return 1
    fi
    success "MongoDB pod: Running"

    # Verify MongoDB version
    if ! kubectl exec -n free5gc deployment/mongodb 2>/dev/null -- mongosh --eval "db.version()" | grep -q "7.0"; then
        fail "MongoDB version check failed (expected 7.0+)"
        return 1
    fi
    success "MongoDB version: 7.0+"

    # Task T5: Weaviate
    log "Checking Weaviate..."
    if ! kubectl get pods -n default 2>/dev/null | grep -q "weaviate.*Running"; then
        fail "Weaviate pod not running"
        kubectl get pods -n default | grep weaviate 2>/dev/null || true
        return 1
    fi
    success "Weaviate pod: Running"

    # Verify Weaviate API
    if ! kubectl exec -n default deployment/weaviate 2>/dev/null -- curl -s http://localhost:8080/v1/meta | grep -q "version"; then
        fail "Weaviate API not responding"
        return 1
    fi
    success "Weaviate API: Accessible"

    success "Databases checkpoint validated!"
    return 0
}

validate_core_services() {
    log "==> Validating Core Services Checkpoint..."

    # Task T6: Nephio R5 + Porch
    log "Checking Nephio Porch..."
    if ! command -v kpt &> /dev/null; then
        fail "kpt CLI not found"
        return 1
    fi

    if ! kpt version 2>/dev/null | grep -q "1.0.0-beta.56"; then
        fail "kpt version mismatch (expected 1.0.0-beta.56)"
        kpt version 2>/dev/null || true
        return 1
    fi
    success "kpt: 1.0.0-beta.56"

    # Check Porch API
    if ! kubectl get apiservice v1alpha1.porch.kpt.dev 2>/dev/null &>/dev/null; then
        fail "Porch API service not found"
        return 1
    fi
    success "Porch API: Registered"

    # Verify package repository
    if ! kpt alpha rpkg get 2>/dev/null | grep -q "free5gc"; then
        warn "Free5GC packages not found in Porch (expected after Nephio catalog setup)"
    else
        success "Porch packages: free5gc detected"
    fi

    # Task T7: O-RAN SC Near-RT RIC
    log "Checking O-RAN SC RIC..."
    if ! kubectl get pods -n ricplt 2>/dev/null &>/dev/null; then
        fail "ricplt namespace not found"
        return 1
    fi

    RIC_PODS_RUNNING=$(kubectl get pods -n ricplt 2>/dev/null | grep -c "Running" || echo "0")
    if [ "$RIC_PODS_RUNNING" -lt 5 ]; then
        fail "O-RAN SC RIC: Less than 5 pods running (found $RIC_PODS_RUNNING)"
        kubectl get pods -n ricplt 2>/dev/null || true
        return 1
    fi
    success "O-RAN SC RIC: $RIC_PODS_RUNNING pods running"

    # Check A1 Policy API (Non-RT RIC)
    if ! kubectl exec -n ricplt deployment/nonrtric 2>/dev/null -- curl -s http://localhost:8080/a1-policy/v2/health | grep -q "status"; then
        warn "A1 Policy API not accessible (may not be fully initialized)"
    else
        success "A1 Policy API: Healthy"
    fi

    success "Core services checkpoint validated!"
    return 0
}

validate_network_functions() {
    log "==> Validating Network Functions Checkpoint..."

    # Task T8: Free5GC Control Plane
    log "Checking Free5GC Control Plane..."
    if ! kubectl get namespace free5gc 2>/dev/null &>/dev/null; then
        fail "free5gc namespace not found"
        return 1
    fi

    CP_NFS=("amf" "smf" "nrf" "ausf" "udm" "udr" "pcf" "nssf")
    for nf in "${CP_NFS[@]}"; do
        if ! kubectl get pods -n free5gc 2>/dev/null | grep -q "free5gc-$nf.*Running"; then
            fail "Free5GC $nf pod not running"
            kubectl get pods -n free5gc 2>/dev/null || true
            return 1
        fi
    done
    success "Free5GC Control Plane: All NFs running"

    # Check NRF registration
    if ! kubectl logs -n free5gc deployment/free5gc-nrf --tail=50 2>/dev/null | grep -q "registered"; then
        warn "NRF registration logs not found (may be too early)"
    else
        success "Free5GC NRF: Registration confirmed"
    fi

    # Task T9: Free5GC User Plane
    log "Checking Free5GC User Plane..."
    UPF_REPLICAS=$(kubectl get deployment -n free5gc free5gc-upf 2>/dev/null -o jsonpath='{.spec.replicas}' || echo "0")
    if [ "$UPF_REPLICAS" -lt 1 ]; then
        fail "Free5GC UPF: No replicas configured"
        return 1
    fi
    success "Free5GC UPF: $UPF_REPLICAS replicas configured"

    UPF_READY=$(kubectl get deployment -n free5gc free5gc-upf 2>/dev/null -o jsonpath='{.status.readyReplicas}' || echo "0")
    if [ "$UPF_READY" -lt 1 ]; then
        fail "Free5GC UPF: No ready replicas"
        kubectl get deployment -n free5gc free5gc-upf 2>/dev/null || true
        return 1
    fi
    success "Free5GC UPF: $UPF_READY/$UPF_REPLICAS ready"

    # Task T10: OAI RAN
    log "Checking OAI RAN..."
    if ! kubectl get namespace oai-ran 2>/dev/null &>/dev/null; then
        fail "oai-ran namespace not found"
        return 1
    fi

    OAI_PODS_RUNNING=$(kubectl get pods -n oai-ran 2>/dev/null | grep -c "Running" || echo "0")
    if [ "$OAI_PODS_RUNNING" -lt 1 ]; then
        fail "OAI RAN: No pods running"
        kubectl get pods -n oai-ran 2>/dev/null || true
        return 1
    fi
    success "OAI RAN: $OAI_PODS_RUNNING pods running"

    # Check gNB connection to AMF
    if kubectl logs -n oai-ran deployment/oai-gnb --tail=100 2>/dev/null | grep -q "Connected to AMF"; then
        success "OAI gNB: Connected to AMF (N2 interface)"
    else
        warn "OAI gNB AMF connection not confirmed in logs (may be initializing)"
    fi

    success "Network functions checkpoint validated!"
    return 0
}

validate_integration() {
    log "==> Validating Integration Checkpoint..."

    # Task T11: Intent Operator A1 Integration
    log "Checking Nephoran Intent Operator..."
    if ! kubectl get crd networkintents.intent.nephoran.com 2>/dev/null &>/dev/null; then
        fail "NetworkIntent CRD not found"
        return 1
    fi
    success "NetworkIntent CRD: Registered"

    if ! kubectl get pods -n nephoran-system 2>/dev/null | grep -q "nephoran-operator.*Running"; then
        fail "Nephoran Intent Operator pod not running"
        kubectl get pods -n nephoran-system 2>/dev/null || true
        return 1
    fi
    success "Nephoran Intent Operator: Running"

    # Check A1 policy creation
    if kubectl get networkintents 2>/dev/null | grep -q "scale-upf-demo"; then
        success "NetworkIntent example: Detected (scale-upf-demo)"

        # Verify A1 policy created
        if kubectl exec -n ricplt deployment/nonrtric 2>/dev/null -- curl -s http://localhost:8080/a1-policy/v2/policies | grep -q "scale-upf"; then
            success "A1 Policy: Created from NetworkIntent"
        else
            warn "A1 Policy not found in Non-RT RIC (may be propagating)"
        fi
    else
        warn "No NetworkIntent examples deployed yet (optional for checkpoint)"
    fi

    # Task T12: E2E Validation
    log "Checking E2E test results..."
    if [ -f "$PROJECT_ROOT/test/e2e/test-results.txt" ]; then
        if grep -q "PASS" "$PROJECT_ROOT/test/e2e/test-results.txt"; then
            success "E2E Tests: PASS"
        else
            fail "E2E Tests: FAIL (check test/e2e/test-results.txt)"
            return 1
        fi
    else
        warn "E2E test results not found (run ./test/e2e/run-all-tests.sh)"
    fi

    success "Integration checkpoint validated!"
    return 0
}

# Main execution
main() {
    CHECKPOINT="${1:-}"

    if [ -z "$CHECKPOINT" ]; then
        echo "Usage: $0 [checkpoint_name]"
        echo ""
        echo "Available checkpoints:"
        echo "  prerequisites      - Verify hardware and software prerequisites"
        echo "  infrastructure     - Validate Kubernetes, Cilium, GPU Operator"
        echo "  databases          - Validate MongoDB, Weaviate"
        echo "  core_services      - Validate Nephio Porch, O-RAN SC RIC"
        echo "  network_functions  - Validate Free5GC, OAI RAN"
        echo "  integration        - Validate Intent Operator, A1, E2E tests"
        echo ""
        exit 1
    fi

    log "======================================"
    log "Checkpoint Validator v2.0"
    log "Checkpoint: $CHECKPOINT"
    log "======================================"

    case "$CHECKPOINT" in
        "prerequisites")
            validate_prerequisites
            ;;
        "infrastructure")
            validate_infrastructure
            ;;
        "databases")
            validate_databases
            ;;
        "core_services")
            validate_core_services
            ;;
        "network_functions")
            validate_network_functions
            ;;
        "integration")
            validate_integration
            ;;
        *)
            fail "Unknown checkpoint: $CHECKPOINT"
            exit 1
            ;;
    esac

    RESULT=$?

    if [ $RESULT -eq 0 ]; then
        log "======================================"
        success "✅ Checkpoint '$CHECKPOINT' PASSED!"
        log "======================================"
        exit 0
    else
        log "======================================"
        fail "❌ Checkpoint '$CHECKPOINT' FAILED!"
        log "======================================"
        exit 1
    fi
}

# Run main function
main "$@"
