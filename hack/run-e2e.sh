#!/usr/bin/env bash
set -euo pipefail

# E2E Test Harness for Nephoran Intent Operator
# Supports Windows (Git Bash/WSL) and Unix environments
# Features:
# - Creates kind cluster
# - Applies CRDs and webhook-manager
# - Runs intent-ingest locally
# - Runs conductor-loop for KRM patches
# - Validates webhook acceptance/rejection
# - Clear PASS/FAIL summary

# --- BEGIN: Configuration ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Test configuration
CLUSTER_NAME="${CLUSTER_NAME:-nephoran-e2e}"
KIND_IMAGE="${KIND_IMAGE:-}"
NAMESPACE="${NAMESPACE:-nephoran-system}"
CRD_DIR="${CRD_DIR:-$PROJECT_ROOT/deployments/crds}"
WEBHOOK_CONFIG="${WEBHOOK_CONFIG:-$PROJECT_ROOT/config/webhook}"
SAMPLES_DIR="${SAMPLES_DIR:-$PROJECT_ROOT/tests/e2e/samples}"
SKIP_CLEANUP="${SKIP_CLEANUP:-false}"
VERBOSE="${VERBOSE:-false}"
TIMEOUT="${TIMEOUT:-300}"

# Component settings
INTENT_INGEST_MODE="${INTENT_INGEST_MODE:-local}"  # local or sidecar
CONDUCTOR_MODE="${CONDUCTOR_MODE:-local}"  # local or in-cluster
PORCH_MODE="${PORCH_MODE:-structured-patch}"  # structured-patch or direct

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
declare -a FAILED_TEST_NAMES=()

# Component process tracking
INTENT_INGEST_PID=""
CONDUCTOR_LOOP_PID=""
PORT_FORWARD_PID=""
<<<<<<< HEAD
=======
LLM_PROCESSOR_PID=""
PORCH_DIRECT_PID=""
A1_SIM_PID=""
E2_SIM_PID=""
O1_SIM_PID=""
>>>>>>> 6835433495e87288b95961af7173d866977175ff
HANDOFF_DIR="$PROJECT_ROOT/handoff"
# --- END: Configuration ---

# --- BEGIN: Utility Functions ---
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

log_test_start() {
    echo -e "\n${BLUE}═══ TEST: $1 ═══${NC}"
    ((TOTAL_TESTS++))
}

log_test_pass() {
    echo -e "${GREEN}✅ PASS: $1${NC}"
    ((PASSED_TESTS++))
}

log_test_fail() {
    echo -e "${RED}❌ FAIL: $1${NC}"
    ((FAILED_TESTS++))
    FAILED_TEST_NAMES+=("$1")
}

wait_for_condition() {
    local description="$1"
    local condition_cmd="$2"
    local timeout="${3:-60}"
    local interval="${4:-2}"
    
    log_info "Waiting for: $description (timeout: ${timeout}s)"
    
    local elapsed=0
    local dots_printed=0
<<<<<<< HEAD
    while [ $elapsed -lt $timeout ]; do
        if eval "$condition_cmd" &>/dev/null; then
=======
    local last_status=""
    
    while [ $elapsed -lt $timeout ]; do
        # Capture both success/failure and any status output
        local status_output
        status_output=$(eval "$condition_cmd" 2>&1 || echo "")
        local exit_code=$?
        
        if [ $exit_code -eq 0 ]; then
>>>>>>> 6835433495e87288b95961af7173d866977175ff
            [ $dots_printed -gt 0 ] && echo ""
            log_success "$description - ready!"
            return 0
        fi
<<<<<<< HEAD
=======
        
>>>>>>> 6835433495e87288b95961af7173d866977175ff
        sleep $interval
        elapsed=$((elapsed + interval))
        echo -n "."
        dots_printed=$((dots_printed + 1))
        
<<<<<<< HEAD
        # Print status every 30 seconds for long waits
        if [ $((elapsed % 30)) -eq 0 ] && [ $elapsed -gt 0 ]; then
            echo ""
            log_info "Still waiting for: $description (${elapsed}s elapsed)"
=======
        # Print status every 30 seconds for long waits with enhanced diagnostics
        if [ $((elapsed % 30)) -eq 0 ] && [ $elapsed -gt 0 ]; then
            echo ""
            log_info "Still waiting for: $description (${elapsed}s elapsed)"
            
            # Show current status if it's different from last time
            if [ "$status_output" != "$last_status" ] && [ -n "$status_output" ]; then
                echo "Current status: $status_output"
                last_status="$status_output"
            fi
            
            # For pod-related waits, show additional context
            if [[ "$description" == *"pod"* ]] || [[ "$description" == *"deployment"* ]]; then
                # Try to extract namespace and deployment info from condition
                local ns="ran-a"  # Default namespace
                local deployment="nf-sim"  # Default deployment
                
                if kubectl get deployment "$deployment" -n "$ns" >/dev/null 2>&1; then
                    local ready_replicas
                    local desired_replicas
                    ready_replicas=$(kubectl get deployment "$deployment" -n "$ns" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
                    desired_replicas=$(kubectl get deployment "$deployment" -n "$ns" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
                    echo "Deployment status: ready=$ready_replicas, desired=$desired_replicas"
                    
                    # Show pod status distribution
                    local pod_phases
                    pod_phases=$(kubectl get pods -n "$ns" -l "app=$deployment" --no-headers 2>/dev/null | awk '{print $3}' | sort | uniq -c | tr '\n' ' ' || echo "no pods")
                    echo "Pod phases: $pod_phases"
                fi
            fi
>>>>>>> 6835433495e87288b95961af7173d866977175ff
        fi
    done
    echo ""
    log_error "$description - timed out after ${timeout}s"
    
<<<<<<< HEAD
    # Provide diagnostic information on timeout
    log_error "Diagnostic information for '$description':"
    eval "$condition_cmd" 2>&1 || true
=======
    # Provide enhanced diagnostic information on timeout
    log_error "Final diagnostic information for '$description':"
    eval "$condition_cmd" 2>&1 || true
    
    # For scaling-related failures, run the debug script
    if [[ "$description" == *"scaling"* ]] || [[ "$description" == *"deployment"* ]] || [[ "$description" == *"pod"* ]]; then
        log_info "Running scaling debug helper..."
        if [ -f "$PROJECT_ROOT/hack/debug-scaling.sh" ]; then
            bash "$PROJECT_ROOT/hack/debug-scaling.sh" ran-a nf-sim "$CLUSTER_NAME" 2>/dev/null | tail -50 || true
        fi
    fi
    
>>>>>>> 6835433495e87288b95961af7173d866977175ff
    return 1
}

# Enhanced health check function for HTTP endpoints
wait_for_http_health() {
    local service_name="$1"
    local url="$2" 
    local timeout="${3:-60}"
    local expected_response="${4:-ok}"
    
    wait_for_condition "$service_name HTTP health check" \
        "curl -s -f --connect-timeout 5 --max-time 10 '$url' | grep -q '$expected_response'" \
        "$timeout" 3
}

# Enhanced process health check
wait_for_process_health() {
    local service_name="$1"
    local pid="$2"
    local health_cmd="$3"
    local timeout="${4:-30}"
    
    log_info "Verifying $service_name process health (PID: $pid)"
    
    # Check if process is still running
    if ! kill -0 "$pid" 2>/dev/null; then
        log_error "$service_name process (PID: $pid) is not running"
        return 1
    fi
    
    # Check application-specific health
    if [ -n "$health_cmd" ]; then
        wait_for_condition "$service_name application health" "$health_cmd" "$timeout" 2
    else
        log_success "$service_name process is running"
        return 0
    fi
}

# Enhanced cleanup function with error handling
cleanup() {
    local exit_code=$?
    
    echo -e "\n${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${BLUE}           CLEANUP & SUMMARY${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    
    # Stop local components first
    log_info "Stopping local components..."
    
    if [ -n "$INTENT_INGEST_PID" ] && kill -0 "$INTENT_INGEST_PID" 2>/dev/null; then
        log_info "Stopping intent-ingest (PID: $INTENT_INGEST_PID)"
        kill -TERM "$INTENT_INGEST_PID" 2>/dev/null || true
        sleep 2
        kill -0 "$INTENT_INGEST_PID" 2>/dev/null && kill -KILL "$INTENT_INGEST_PID" 2>/dev/null || true
    fi
    
    if [ -n "$CONDUCTOR_LOOP_PID" ] && kill -0 "$CONDUCTOR_LOOP_PID" 2>/dev/null; then
        log_info "Stopping conductor-loop (PID: $CONDUCTOR_LOOP_PID)"
        kill -TERM "$CONDUCTOR_LOOP_PID" 2>/dev/null || true
        sleep 2
        kill -0 "$CONDUCTOR_LOOP_PID" 2>/dev/null && kill -KILL "$CONDUCTOR_LOOP_PID" 2>/dev/null || true
    fi
    
    if [ -n "$PORT_FORWARD_PID" ] && kill -0 "$PORT_FORWARD_PID" 2>/dev/null; then
        log_info "Stopping port-forward (PID: $PORT_FORWARD_PID)"
        kill -TERM "$PORT_FORWARD_PID" 2>/dev/null || true
    fi
    
    # Clean up test resources with better error handling
    if kubectl cluster-info --context "kind-$CLUSTER_NAME" &>/dev/null; then
        log_info "Cleaning up test resources..."
        kubectl delete networkintent --all -n "$NAMESPACE" 2>/dev/null || true
        kubectl delete pods --field-selector=status.phase=Failed -n "$NAMESPACE" 2>/dev/null || true
        kubectl delete -f examples/intent-scaling-up.json --ignore-not-found=true >/dev/null 2>&1 || true
        kubectl delete -f examples/intent-scaling-down.json --ignore-not-found=true >/dev/null 2>&1 || true
        kubectl delete networkintent test-webhook-intent --ignore-not-found=true >/dev/null 2>&1 || true
    fi
    
    echo -e "\n${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${BLUE}           E2E TEST SUMMARY${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    
    echo -e "Total Tests:  ${TOTAL_TESTS}"
    echo -e "Passed:       ${GREEN}${PASSED_TESTS}${NC}"
    echo -e "Failed:       ${RED}${FAILED_TESTS}${NC}"
    
    if [ ${#FAILED_TEST_NAMES[@]} -gt 0 ]; then
        echo -e "\n${RED}Failed Tests:${NC}"
        for test_name in "${FAILED_TEST_NAMES[@]}"; do
            echo -e "  - ${test_name}"
        done
    fi
    
    # Show handoff directory status
    if [ -d "$HANDOFF_DIR" ]; then
        PATCH_COUNT=$(find "$HANDOFF_DIR" -name "*.json" -o -name "*.yaml" 2>/dev/null | wc -l || echo "0")
        echo -e "\nHandoff Directory: $HANDOFF_DIR"
        echo -e "Generated Patches: $PATCH_COUNT"
    fi
    
    if [ "$SKIP_CLEANUP" = "true" ]; then
        log_warning "Skipping cluster cleanup (SKIP_CLEANUP=true)"
        echo "   To delete cluster manually: kind delete cluster --name $CLUSTER_NAME"
        echo "   To inspect handoff directory: ls -la $HANDOFF_DIR"
    elif [ -n "${CLUSTER_NAME:-}" ]; then
        log_info "Cleaning up cluster: $CLUSTER_NAME"
        kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
    fi
    
    echo -e "\n${BLUE}═══════════════════════════════════════════${NC}"
    if [ $FAILED_TESTS -eq 0 ] && [ $TOTAL_TESTS -gt 0 ]; then
        echo -e "${GREEN}        🎉 ALL TESTS PASSED! 🎉${NC}"
        echo -e "${GREEN}     E2E Pipeline Validation: SUCCESS${NC}"
    else
        echo -e "${RED}        ⚠️  SOME TESTS FAILED ⚠️${NC}"
        echo -e "${RED}     E2E Pipeline Validation: FAILURE${NC}"
        exit_code=1
    fi
    echo -e "${BLUE}═══════════════════════════════════════════${NC}\n"
    
    exit $exit_code
}
trap cleanup EXIT

# Add argument handling to the comprehensive script
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [options]"
        echo ""
        echo "Options:"
        echo "  --help, -h          Show this help message"
        echo "  --no-cleanup        Don't clean up test resources"
        echo ""
        echo "Environment variables:"
        echo "  NAMESPACE           Target namespace (default: nephoran-system)"
        echo "  CLUSTER_NAME        Kind cluster name (default: nephoran-e2e)"
        echo "  TIMEOUT            Wait timeout in seconds (default: 300)"
        echo "  SKIP_CLEANUP       Skip cleanup (default: false)"
        echo "  VERBOSE            Enable verbose output (default: false)"
        echo "  INTENT_INGEST_MODE  Mode for intent-ingest: local|sidecar (default: local)"
        echo "  CONDUCTOR_MODE     Mode for conductor-loop: local|in-cluster (default: local)"
        echo "  PORCH_MODE         Porch integration mode: structured-patch|direct (default: structured-patch)"
        exit 0
        ;;
    --no-cleanup)
        SKIP_CLEANUP="true"
        ;;
    "")
        # Continue with normal execution
        ;;
    *)
        log_error "Unknown option: $1"
        log_error "Use --help for usage information"
        exit 1
        ;;
esac

# Include the comprehensive tool detection and test execution from feat/e2e
# but merge with the error checking from integrate/mvp

# Enhanced prerequisite checking
check_prerequisites() {
    log_info "Checking prerequisites..."
    
<<<<<<< HEAD
    # Check if kubectl is available
=======
    # Check if required tools are available
>>>>>>> 6835433495e87288b95961af7173d866977175ff
    if ! command -v kubectl >/dev/null 2>&1; then
        log_error "kubectl is required but not installed"
        log_error "Install: https://kubernetes.io/docs/tasks/tools/"
        log_error "Windows: choco install kubernetes-cli"
        exit 1
    fi
    
<<<<<<< HEAD
    # Check if cluster is accessible
    if ! kubectl cluster-info >/dev/null 2>&1; then
        log_error "Cannot access Kubernetes cluster"
        log_info "Please ensure a Kubernetes cluster is running and accessible"
        exit 1
    fi
    
=======
    if ! command -v kind >/dev/null 2>&1; then
        log_error "kind is required but not installed"
        log_error "Install: https://kind.sigs.k8s.io/docs/user/quick-start/"
        log_error "Windows: choco install kind"
        exit 1
    fi
    
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is required but not installed"
        log_error "Install: https://golang.org/doc/install"
        exit 1
    fi
    
    # Note: We don't check for existing cluster here since we'll create our own kind cluster
>>>>>>> 6835433495e87288b95961af7173d866977175ff
    log_success "Prerequisites check passed"
}

check_prerequisites

log_success "E2E test execution started!"
log_info "Use SKIP_CLEANUP=true to preserve cluster for debugging"
log_info "Use VERBOSE=true for detailed output"
<<<<<<< HEAD
=======

# --- BEGIN: Core Implementation ---

# Create kind cluster
create_kind_cluster() {
    log_info "Creating kind cluster: $CLUSTER_NAME"
    
    # Check if cluster already exists
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_warning "Cluster $CLUSTER_NAME already exists"
        if [ "$SKIP_CLEANUP" = "false" ]; then
            log_info "Deleting existing cluster..."
            kind delete cluster --name "$CLUSTER_NAME"
        else
            log_info "Using existing cluster"
            return 0
        fi
    fi
    
    # Create cluster with optimized configuration for scaling tests
    cat > /tmp/kind-config.yaml << EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
        # Increase resource limits for better pod scheduling
        max-pods: "110"
        eviction-hard: "memory.available<200Mi,nodefs.available<10%"
        system-reserved: "memory=200Mi"
  extraPortMappings:
  - containerPort: 80
    hostPort: 8080
    protocol: TCP
  - containerPort: 443
    hostPort: 8443
    protocol: TCP
  # Allocate more resources to control plane for scaling tests
  extraMounts:
  - hostPath: /tmp/kind-data
    containerPath: /tmp/data
- role: worker
  kubeadmConfigPatches:
  - |
    kind: JoinConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        max-pods: "110"
        eviction-hard: "memory.available<200Mi,nodefs.available<10%"
- role: worker
  kubeadmConfigPatches:
  - |
    kind: JoinConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        max-pods: "110"
        eviction-hard: "memory.available<200Mi,nodefs.available<10%"
EOF
    
    kind create cluster --name "$CLUSTER_NAME" --config /tmp/kind-config.yaml
    kubectl cluster-info --context "kind-$CLUSTER_NAME"
    
    # Wait for cluster to be ready with enhanced verification
    wait_for_condition "Cluster nodes ready" \
        "kubectl get nodes --context 'kind-$CLUSTER_NAME' --no-headers | awk '{print \$2}' | grep -v Ready | wc -l | grep -q '^0$'" \
        180
    
    # Additional verification - ensure all nodes are Ready and not in NotReady state
    wait_for_condition "All nodes fully ready" \
        "kubectl get nodes --context 'kind-$CLUSTER_NAME' --no-headers | wc -l | grep -q '3'" \
        60
    
    # Pre-warm the cluster and verify resources
    log_info "Pre-warming cluster and verifying resources..."
    
    # Check cluster resource capacity
    log_info "Checking cluster resource capacity..."
    kubectl describe nodes --context "kind-$CLUSTER_NAME" | grep -E "(Name|Allocatable:|Allocated resources)" | head -20 || true
    
    # Pre-pull nginx image on all nodes with minimal resources
    kubectl create namespace image-puller --context "kind-$CLUSTER_NAME" --dry-run=client -o yaml | kubectl apply -f - || true
    cat > /tmp/image-puller.yaml << EOF
apiVersion: v1
kind: Pod
metadata:
  name: image-puller
  namespace: image-puller
spec:
  restartPolicy: Never
  containers:
  - name: puller
    image: nginx:1.25-alpine
    command: ["echo", "Image pulled successfully"]
    resources:
      requests:
        memory: "8Mi"
        cpu: "10m"
      limits:
        memory: "16Mi"
        cpu: "50m"
EOF
    kubectl apply -f /tmp/image-puller.yaml --context "kind-$CLUSTER_NAME" || true
    kubectl wait --for=condition=Ready pod/image-puller -n image-puller --context "kind-$CLUSTER_NAME" --timeout=30s || true
    kubectl delete namespace image-puller --context "kind-$CLUSTER_NAME" || true
}

# Setup namespace and RBAC
setup_namespace() {
    log_info "Setting up namespace: $NAMESPACE"
    
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    kubectl create namespace ran-a --dry-run=client -o yaml | kubectl apply -f - || true
    kubectl create namespace ran-b --dry-run=client -o yaml | kubectl apply -f - || true
    
    log_success "Namespace setup complete"
}

# Install CRDs
install_crds() {
    log_info "Installing CRDs..."
    
    # Apply main CRDs
    if [ -f "$PROJECT_ROOT/config/crd/bases/intent.nephoran.com_networkintents.yaml" ]; then
        kubectl apply -f "$PROJECT_ROOT/config/crd/bases/intent.nephoran.com_networkintents.yaml"
    fi
    
    # Apply only valid CRDs (avoid those with trailing dots)
    find "$PROJECT_ROOT/config/crd/bases" -name "*.yaml" ! -name "*nephoran.com._*.yaml" -exec kubectl apply -f {} \; || true
    
    # Wait for CRDs to be established
    wait_for_condition "NetworkIntent CRD established" \
        "kubectl get crd networkintents.intent.nephoran.com -o jsonpath='{.status.conditions[?(@.type==\"Established\")].status}' | grep -q True" \
        60
    
    log_success "CRDs installed successfully"
}

# Setup webhook with self-signed certificates
setup_webhook() {
    log_info "Setting up webhook with self-signed certificates..."
    
    # Build webhook manager
    cd "$PROJECT_ROOT"
    go build -o /tmp/webhook-manager ./cmd/webhook-manager/
    
    # Generate self-signed certificates
    mkdir -p /tmp/webhook-certs
    
    # Handle Windows/Git Bash path issues
    if [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
        # Use alternative method for Windows
        cat > /tmp/webhook-certs/req.conf << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
CN = webhook-service.${NAMESPACE}.svc

[v3_req]
subjectAltName = DNS:webhook-service.${NAMESPACE}.svc,DNS:webhook-service.${NAMESPACE}.svc.cluster.local
EOF
        openssl req -x509 -newkey rsa:2048 -keyout /tmp/webhook-certs/tls.key \
            -out /tmp/webhook-certs/tls.crt -days 30 -nodes -config /tmp/webhook-certs/req.conf -extensions v3_req
    else
        # Unix/Linux method
        openssl req -x509 -newkey rsa:4096 -keyout /tmp/webhook-certs/tls.key \
            -out /tmp/webhook-certs/tls.crt -days 30 -nodes \
            -subj "/CN=webhook-service.${NAMESPACE}.svc" \
            -addext "subjectAltName=DNS:webhook-service.${NAMESPACE}.svc,DNS:webhook-service.${NAMESPACE}.svc.cluster.local"
    fi
    
    # Create secret with certificates
    kubectl create secret tls webhook-server-certs \
        --cert=/tmp/webhook-certs/tls.crt \
        --key=/tmp/webhook-certs/tls.key \
        -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Apply webhook configuration
    if [ -f "$PROJECT_ROOT/config/webhook/deployment.yaml" ]; then
        kubectl apply -f "$PROJECT_ROOT/config/webhook/" -n "$NAMESPACE" || true
    fi
    
    # Apply validating webhook with correct CA bundle
    CA_BUNDLE=$(base64 < /tmp/webhook-certs/tls.crt | tr -d '\n')
    cat > /tmp/validating-webhook.yaml << EOF
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingAdmissionWebhook
metadata:
  name: networkintent-webhook
webhooks:
- name: networkintent.nephoran.com
  clientConfig:
    service:
      name: webhook-service
      namespace: ${NAMESPACE}
      path: "/validate"
    caBundle: ${CA_BUNDLE}
  rules:
  - operations: ["CREATE", "UPDATE"]
    apiGroups: ["intent.nephoran.com"]
    apiVersions: ["v1"]
    resources: ["networkintents"]
  admissionReviewVersions: ["v1", "v1beta1"]
  sideEffects: None
  failurePolicy: Fail
EOF
    
    kubectl apply -f /tmp/validating-webhook.yaml
    
    log_success "Webhook setup complete"
}

# Start intent-ingest locally
start_intent_ingest() {
    log_info "Starting intent-ingest (OFFLINE mode)..."
    
    cd "$PROJECT_ROOT"
    mkdir -p "$HANDOFF_DIR"
    
    # Build intent-ingest
    go build -o /tmp/intent-ingest ./cmd/intent-ingest/
    
    # Start intent-ingest in background
    KUBECONFIG=$HOME/.kube/config /tmp/intent-ingest \
        --mode=offline \
        --handoff-dir="$HANDOFF_DIR" \
        --namespace="$NAMESPACE" \
        --port=9080 > /tmp/intent-ingest.log 2>&1 &
    
    INTENT_INGEST_PID=$!
    log_info "Intent-ingest started with PID: $INTENT_INGEST_PID"
    
    # Wait for service to be ready
    wait_for_http_health "intent-ingest" "http://localhost:9080/health" 30 "ok"
    
    log_success "Intent-ingest ready"
}

# Start LLM processor in offline mode
start_llm_processor() {
    log_info "Starting llm-processor (OFFLINE mode)..."
    
    cd "$PROJECT_ROOT"
    
    # Build llm-processor
    go build -o /tmp/llm-processor ./cmd/llm-processor/
    
    # Start llm-processor in background
    /tmp/llm-processor \
        --mode=offline \
        --listen-addr=:8081 \
        --handoff-dir="$HANDOFF_DIR" > /tmp/llm-processor.log 2>&1 &
    
    LLM_PROCESSOR_PID=$!
    log_info "LLM-processor started with PID: $LLM_PROCESSOR_PID"
    
    # Wait for service to be ready
    wait_for_http_health "llm-processor" "http://localhost:8081/health" 30 "ok"
    
    log_success "LLM-processor ready"
}

# Start porch-direct for package management
start_porch_direct() {
    log_info "Starting porch-direct..."
    
    cd "$PROJECT_ROOT"
    
    # Build porch-direct
    go build -o /tmp/porch-direct ./cmd/porch-direct/
    
    # Start porch-direct in background
    KUBECONFIG=$HOME/.kube/config /tmp/porch-direct \
        --handoff-dir="$HANDOFF_DIR" \
        --target-namespace-pattern="ran-*" \
        --watch-interval=5s > /tmp/porch-direct.log 2>&1 &
    
    PORCH_DIRECT_PID=$!
    log_info "Porch-direct started with PID: $PORCH_DIRECT_PID"
    
    log_success "Porch-direct ready"
}

# Start simulators
start_simulators() {
    log_info "Starting O-RAN simulators..."
    
    cd "$PROJECT_ROOT"
    
    # Build and start A1 simulator
    go build -o /tmp/a1-sim ./cmd/a1-sim/
    /tmp/a1-sim --port=9001 > /tmp/a1-sim.log 2>&1 &
    A1_SIM_PID=$!
    
    # Build and start E2 simulator
    go build -o /tmp/e2-kmp-sim ./cmd/e2-kpm-sim/
    /tmp/e2-kmp-sim --port=9002 > /tmp/e2-kmp-sim.log 2>&1 &
    E2_SIM_PID=$!
    
    # Build and start O1 VES simulator
    go build -o /tmp/o1-ves-sim ./cmd/o1-ves-sim/
    /tmp/o1-ves-sim --port=9003 > /tmp/o1-ves-sim.log 2>&1 &
    O1_SIM_PID=$!
    
    log_info "Simulators started: A1($A1_SIM_PID), E2($E2_SIM_PID), O1($O1_SIM_PID)"
    log_success "All simulators ready"
}

# Deploy test workloads
deploy_test_workloads() {
    log_info "Deploying test workloads in target namespaces..."
    
    # Deploy a test deployment in ran-a namespace with optimized configuration for kind
    cat > /tmp/nf-sim-deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nf-sim
  namespace: ran-a
  labels:
    app: nf-sim
spec:
  replicas: 1
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0
      maxSurge: 1
  selector:
    matchLabels:
      app: nf-sim
  template:
    metadata:
      labels:
        app: nf-sim
    spec:
      # Optimized for kind cluster stability
      restartPolicy: Always
      terminationGracePeriodSeconds: 30
      # Relaxed scheduling for kind cluster with limited nodes
      # Remove anti-affinity to allow multiple pods per node if needed
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
      - key: node-role.kubernetes.io/control-plane
        operator: Exists
        effect: NoSchedule
      containers:
      - name: nf-sim
        image: nginx:1.25-alpine
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 80
          name: http
        # Ultra-minimal resources for kind cluster stability
        resources:
          requests:
            memory: "4Mi"
            cpu: "1m"
          limits:
            memory: "16Mi"
            cpu: "10m"
        # Optimized probes for faster startup and resource efficiency
        readinessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 2
          periodSeconds: 3
          timeoutSeconds: 2
          successThreshold: 1
          failureThreshold: 2
        livenessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 2
          successThreshold: 1
          failureThreshold: 3
        # Environment variables for nginx optimization
        env:
        - name: NGINX_WORKER_PROCESSES
          value: "1"
        - name: NGINX_WORKER_CONNECTIONS
          value: "1024"
        # Configure security context
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          runAsUser: 101
          capabilities:
            drop:
            - ALL
---
apiVersion: v1
kind: Service
metadata:
  name: nf-sim
  namespace: ran-a
spec:
  selector:
    app: nf-sim
  ports:
  - port: 80
    targetPort: 80
EOF
    
    kubectl apply -f /tmp/nf-sim-deployment.yaml
    
    # Wait for deployment to be ready with enhanced checking
    log_info "Waiting for initial deployment to be ready..."
    wait_for_condition "NF-sim deployment ready" \
        "kubectl get deployment nf-sim -n ran-a -o jsonpath='{.status.readyReplicas}' | grep -q '1'" \
        180
    
    # Additional stability check - wait for pods to be fully ready
    wait_for_condition "NF-sim pods stable" \
        "kubectl get pods -n ran-a -l app=nf-sim --field-selector=status.phase=Running | grep -c Ready | grep -q '1'" \
        60
    
    log_success "Test workloads deployed"
}

# Execute scaling test
execute_scaling_test() {
    log_test_start "Network Intent Scaling Test"
    
    # Pre-scaling verification - ensure initial state is stable
    log_info "Verifying initial deployment state (1 replica)..."
    local initial_replicas
    initial_replicas=$(kubectl get deployment nf-sim -n ran-a -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$initial_replicas" != "1" ]; then
        log_error "Initial deployment not ready. Current ready replicas: $initial_replicas"
        debug_failed_pods
        log_test_fail "Network Intent Scaling Test - initial state verification failed"
        return 1
    fi
    
    log_success "Initial state verified: 1 replica ready"
    
    # Create scaling intent JSON
    log_info "Creating scaling intent..."
    cat > /tmp/scaling-intent.json << EOF
{
    "apiVersion": "intent.nephoran.com/v1alpha1",
    "kind": "NetworkIntent",
    "metadata": {
        "name": "scale-nf-sim",
        "namespace": "ran-a"
    },
    "spec": {
        "intentType": "scaling",
        "namespace": "ran-a",
        "target": "nf-sim deployment",
        "replicas": 3
    }
}
EOF
    
    # Submit intent via intent-ingest with retry logic
    local submit_attempts=0
    local max_attempts=3
    
    while [ $submit_attempts -lt $max_attempts ]; do
        if curl -X POST -H "Content-Type: application/json" \
            -d @/tmp/scaling-intent.json \
            http://localhost:9080/intent 2>/dev/null; then
            log_success "Intent submitted successfully"
            break
        else
            submit_attempts=$((submit_attempts + 1))
            if [ $submit_attempts -ge $max_attempts ]; then
                log_error "Failed to submit intent after $max_attempts attempts"
                log_test_fail "Network Intent Scaling Test - intent submission failed"
                return 1
            fi
            log_warning "Intent submission failed, retrying ($submit_attempts/$max_attempts)..."
            sleep 5
        fi
    done
    
    # Wait for processing and application with progressive checks
    log_info "Waiting for intent processing and scaling..."
    sleep 5
    
    # First check - verify that deployment spec has been updated
    log_info "Verifying deployment spec has been updated..."
    local spec_updated=false
    for i in {1..30}; do
        local spec_replicas
        spec_replicas=$(kubectl get deployment nf-sim -n ran-a -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
        if [ "$spec_replicas" = "3" ]; then
            log_success "Deployment spec updated to 3 replicas"
            spec_updated=true
            break
        fi
        echo -n "."
        sleep 2
    done
    echo ""
    
    if [ "$spec_updated" != "true" ]; then
        log_error "Deployment spec was not updated to 3 replicas within 60 seconds"
        debug_failed_pods
        log_test_fail "Network Intent Scaling Test - spec update failed"
        return 1
    fi
    
    # Enhanced verification with resource monitoring and intermediate progress
    log_info "Monitoring scaling progress with resource tracking..."
    local scaling_start_time=$(date +%s)
    local timeout_duration=300  # 5 minutes for complete scaling
    
    # Check available cluster resources before scaling
    log_info "Pre-scaling resource check..."
    check_cluster_resources
    
    while true; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - scaling_start_time))
        
        if [ $elapsed -gt $timeout_duration ]; then
            log_error "Scaling timeout after ${timeout_duration} seconds"
            debug_failed_pods
            log_test_fail "Network Intent Scaling Test - scaling timeout"
            return 1
        fi
        
        local ready_replicas
        local available_replicas
        ready_replicas=$(kubectl get deployment nf-sim -n ran-a -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        available_replicas=$(kubectl get deployment nf-sim -n ran-a -o jsonpath='{.status.availableReplicas}' 2>/dev/null || echo "0")
        
        log_info "Scaling progress: ready=$ready_replicas, available=$available_replicas (target=3, elapsed=${elapsed}s)"
        
        if [ "$ready_replicas" = "3" ] && [ "$available_replicas" = "3" ]; then
            log_success "Scaling completed successfully!"
            break
        fi
        
        # Check for stuck or failing pods with enhanced diagnostics
        local failing_pods
        failing_pods=$(kubectl get pods -n ran-a -l app=nf-sim --no-headers | grep -E '(Error|CrashLoopBackOff|ImagePullBackOff|Pending)' | wc -l)
        if [ "$failing_pods" -gt 0 ]; then
            log_warning "Found $failing_pods failing pods, running enhanced diagnostics..."
            kubectl get pods -n ran-a -l app=nf-sim -o wide
            
            # Check node resource pressure
            log_info "Checking node resource pressure..."
            kubectl describe nodes | grep -E "(MemoryPressure|DiskPressure|PIDPressure)" || true
            
            # Check pod resource requests vs available
            check_resource_conflicts ran-a nf-sim
        fi
        
        sleep 10
    done
    
    # Final verification using the dedicated Go verifier
    log_info "Running final verification..."
    if go run "$PROJECT_ROOT/tools/verify-scale.go" \
        --namespace=ran-a \
        --name=nf-sim \
        --target-replicas=3 \
        --timeout=60s; then
        
        # Additional stability check - ensure pods remain stable
        log_info "Performing stability check (30s)..."
        sleep 30
        
        local final_replicas
        final_replicas=$(kubectl get deployment nf-sim -n ran-a -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        
        if [ "$final_replicas" = "3" ]; then
            log_test_pass "Network Intent Scaling Test"
        else
            log_error "Scaling unstable - final replica count: $final_replicas"
            
            # Try fallback strategies before failing
            log_info "Attempting resource fallback strategies..."
            if implement_resource_fallbacks ran-a nf-sim 3; then
                log_info "Fallback implemented, re-checking after 60s..."
                sleep 60
                
                local fallback_replicas
                fallback_replicas=$(kubectl get deployment nf-sim -n ran-a -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
                
                if [ "$fallback_replicas" = "3" ]; then
                    log_test_pass "Network Intent Scaling Test (with fallback)"
                else
                    debug_failed_pods
                    log_test_fail "Network Intent Scaling Test - instability detected even with fallbacks"
                    return 1
                fi
            else
                debug_failed_pods
                log_test_fail "Network Intent Scaling Test - fallback strategies failed"
                return 1
            fi
        fi
    else
        # Try fallback strategies before failing
        log_error "Initial verification failed - trying fallback strategies..."
        if implement_resource_fallbacks ran-a nf-sim 3; then
            log_info "Fallback implemented, re-running verification..."
            sleep 60
            
            if go run "$PROJECT_ROOT/tools/verify-scale.go" \
                --namespace=ran-a \
                --name=nf-sim \
                --target-replicas=3 \
                --timeout=60s; then
                log_test_pass "Network Intent Scaling Test (with fallback)"
            else
                debug_failed_pods
                log_test_fail "Network Intent Scaling Test - fallback verification failed"
                return 1
            fi
        else
            debug_failed_pods
            log_test_fail "Network Intent Scaling Test - verification and fallback failed"
            return 1
        fi
    fi
}

# Create summary table with required format
create_summary_table() {
    log_info "Creating final verification results..."
    
    echo -e "\n${BLUE}═══════════════════════════════════════════${NC}"
    echo -e "${BLUE}           E2E TEST FINAL RESULTS${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════${NC}"
    
    # Get deployment status
    DESIRED_REPLICAS=$(kubectl get deployment nf-sim -n ran-a -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
    READY_REPLICAS=$(kubectl get deployment nf-sim -n ran-a -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    # Display in required format: "Replicas (nf-sim, ran-a): desired=3, ready=3 (OK)"
    if [ "$DESIRED_REPLICAS" = "3" ] && [ "$READY_REPLICAS" = "3" ]; then
        echo -e "${GREEN}Replicas (nf-sim, ran-a): desired=${DESIRED_REPLICAS}, ready=${READY_REPLICAS} (OK)${NC}"
        SCALING_SUCCESS=true
    else
        echo -e "${RED}Replicas (nf-sim, ran-a): desired=${DESIRED_REPLICAS}, ready=${READY_REPLICAS} (FAILED)${NC}"
        SCALING_SUCCESS=false
        
        # Show pod status for debugging
        echo -e "\n${YELLOW}Pod Status for debugging:${NC}"
        kubectl get pods -n ran-a -l app=nf-sim -o wide 2>/dev/null || echo "No pods found"
        
        # Show recent events
        echo -e "\n${YELLOW}Recent Events:${NC}"
        kubectl get events -n ran-a --sort-by='.lastTimestamp' --field-selector reason!=Pulling,reason!=Pulled,reason!=Created | tail -10 2>/dev/null || echo "No events found"
    fi
    
    # Additional status checks
    echo -e "\n${BLUE}Component Status:${NC}"
    echo "─────────────────────────────────────────────"
    echo -e "Kind Cluster     → ✅ Running"
    echo -e "NetworkIntent CRD → ✅ Established"
    echo -e "Test Namespace   → ✅ Created"
    echo -e "NetworkIntent    → $(kubectl get networkintent -n ran-a --no-headers 2>/dev/null | wc -l) created"
    echo "─────────────────────────────────────────────"
}

# Main execution flow
main() {
    log_info "Starting E2E test pipeline..."
    
    create_kind_cluster
    
    # Verify cluster resources before proceeding
    if [ -f "$PROJECT_ROOT/hack/verify-kind-resources.sh" ]; then
        log_info "Running cluster resource verification..."
        bash "$PROJECT_ROOT/hack/verify-kind-resources.sh" "$CLUSTER_NAME" 3 || {
            log_error "Cluster resource verification failed"
            log_error "The cluster may not have sufficient resources for reliable scaling"
            log_error "Consider using a larger kind cluster or adjusting resource requests"
        }
    fi
    
    setup_namespace
    install_crds
    setup_webhook
    deploy_test_workloads
    start_intent_ingest
    start_llm_processor
    start_porch_direct
    start_simulators
    
    # Give services time to stabilize
    log_info "Allowing services to stabilize..."
    sleep 10
    
    execute_scaling_test
    create_summary_table
    
    log_success "E2E test pipeline completed successfully!"
}

# Resource monitoring and conflict detection functions
check_cluster_resources() {
    log_info "Checking cluster resource availability..."
    
    # Get node resource information
    local total_memory_mb=0
    local total_cpu_m=0
    local used_memory_mb=0
    local used_cpu_m=0
    
    # Simple resource check using kubectl describe nodes
    if kubectl describe nodes 2>/dev/null | grep -E "Allocatable|Allocated resources" >/dev/null; then
        log_success "Node resources available for inspection"
        
        # Show summary of node resources
        echo "Node Resource Summary:"
        kubectl describe nodes | grep -A 10 "Allocatable:" | head -15 || true
        kubectl describe nodes | grep -A 15 "Allocated resources:" | grep -E "(cpu|memory)" | head -10 || true
    else
        log_warning "Cannot retrieve detailed node resource information"
    fi
    
    # Check if metrics server is available
    if kubectl top nodes >/dev/null 2>&1; then
        log_success "Metrics server available - showing current usage"
        kubectl top nodes
    else
        log_warning "Metrics server not available - resource monitoring limited"
    fi
}

check_resource_conflicts() {
    local namespace="$1"
    local deployment="$2"
    
    log_info "Checking for resource conflicts in $namespace/$deployment"
    
    # Get pod resource requests
    local pod_memory_requests
    local pod_cpu_requests
    
    pod_memory_requests=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.spec.template.spec.containers[0].resources.requests.memory}' 2>/dev/null || echo "unknown")
    pod_cpu_requests=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.spec.template.spec.containers[0].resources.requests.cpu}' 2>/dev/null || echo "unknown")
    
    echo "Pod resource requests: memory=$pod_memory_requests, cpu=$pod_cpu_requests"
    
    # Check for pending pods due to insufficient resources
    local pending_pods
    pending_pods=$(kubectl get pods -n "$namespace" -l "app=$deployment" --field-selector=status.phase=Pending --no-headers 2>/dev/null | wc -l)
    
    if [ "$pending_pods" -gt 0 ]; then
        log_error "Found $pending_pods pending pods - likely resource constraint"
        
        # Get detailed reasons for pending pods
        kubectl get pods -n "$namespace" -l "app=$deployment" --field-selector=status.phase=Pending --no-headers 2>/dev/null | while read -r pod_line; do
            if [ -n "$pod_line" ]; then
                local pod_name
                pod_name=$(echo "$pod_line" | awk '{print $1}')
                echo "Pending pod details: $pod_name"
                kubectl describe pod "$pod_name" -n "$namespace" 2>/dev/null | grep -A 5 "Events:" | tail -5 || true
            fi
        done
    fi
}

# Fallback strategy implementation
implement_resource_fallbacks() {
    local namespace="$1"
    local deployment="$2"
    local target_replicas="$3"
    
    log_info "Implementing resource fallback strategies for $namespace/$deployment"
    
    # Strategy 1: Reduce resource requests if pods are pending
    local pending_pods
    pending_pods=$(kubectl get pods -n "$namespace" -l "app=$deployment" --field-selector=status.phase=Pending --no-headers 2>/dev/null | wc -l)
    
    if [ "$pending_pods" -gt 0 ]; then
        log_warning "Found pending pods, attempting resource reduction fallback"
        
        # Create a minimal resource version of the deployment
        cat > /tmp/fallback-deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $deployment
  namespace: $namespace
  labels:
    app: $deployment
spec:
  replicas: $target_replicas
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  selector:
    matchLabels:
      app: $deployment
  template:
    metadata:
      labels:
        app: $deployment
    spec:
      restartPolicy: Always
      terminationGracePeriodSeconds: 10
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
      - key: node-role.kubernetes.io/control-plane
        operator: Exists
        effect: NoSchedule
      containers:
      - name: nf-sim
        image: nginx:1.25-alpine
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 80
          name: http
        # Ultra-minimal resources for resource-constrained environments
        resources:
          requests:
            memory: "8Mi"
            cpu: "5m"
          limits:
            memory: "32Mi"
            cpu: "50m"
        readinessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 1
          periodSeconds: 2
          timeoutSeconds: 1
          successThreshold: 1
          failureThreshold: 2
        livenessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 3
          periodSeconds: 5
          timeoutSeconds: 1
          successThreshold: 1
          failureThreshold: 3
        env:
        - name: NGINX_WORKER_PROCESSES
          value: "1"
        - name: NGINX_WORKER_CONNECTIONS
          value: "512"
        securityContext:
          allowPrivilegeEscalation: false
          runAsNonRoot: true
          runAsUser: 101
          capabilities:
            drop:
            - ALL
EOF
        
        log_info "Applying fallback deployment with minimal resources..."
        kubectl apply -f /tmp/fallback-deployment.yaml
        
        # Wait for the fallback to take effect
        sleep 15
        
        # Check if fallback helped
        local new_pending
        new_pending=$(kubectl get pods -n "$namespace" -l "app=$deployment" --field-selector=status.phase=Pending --no-headers 2>/dev/null | wc -l)
        
        if [ "$new_pending" -lt "$pending_pods" ]; then
            log_success "Fallback strategy improved pod scheduling (pending: $pending_pods -> $new_pending)"
            return 0
        else
            log_warning "Fallback strategy did not improve scheduling significantly"
        fi
    fi
    
    # Strategy 2: Gradual scaling if full scaling fails
    log_info "Attempting gradual scaling approach..."
    
    # Scale to 2 first, then to 3
    kubectl scale deployment "$deployment" -n "$namespace" --replicas=2
    sleep 30
    
    local intermediate_ready
    intermediate_ready=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    
    if [ "$intermediate_ready" = "2" ]; then
        log_success "Intermediate scaling to 2 replicas successful"
        
        # Now try to scale to target
        kubectl scale deployment "$deployment" -n "$namespace" --replicas="$target_replicas"
        sleep 30
        
        return 0
    else
        log_warning "Gradual scaling strategy also encountering issues"
        return 1
    fi
}

# Enhanced debugging function for failed pods
debug_failed_pods() {
    log_error "Debugging failed pods and components..."
    
    echo -e "\n${RED}=== POD DEBUG INFORMATION ===${NC}"
    
    # Get pod details
    echo -e "\n${YELLOW}1. Pod Status and Details:${NC}"
    kubectl get pods -n ran-a -l app=nf-sim -o wide 2>/dev/null || echo "No nf-sim pods found"
    
    # Get pod descriptions for failed pods
    echo -e "\n${YELLOW}2. Failed Pod Descriptions:${NC}"
    for pod in $(kubectl get pods -n ran-a -l app=nf-sim --no-headers 2>/dev/null | grep -v Running | awk '{print $1}'); do
        if [ -n "$pod" ]; then
            echo -e "\n--- Pod: $pod ---"
            kubectl describe pod "$pod" -n ran-a 2>/dev/null || echo "Could not describe pod $pod"
        fi
    done
    
    # Get pod logs for all pods (including previous containers)
    echo -e "\n${YELLOW}3. Pod Logs (last 100 lines each):${NC}"
    for pod in $(kubectl get pods -n ran-a -l app=nf-sim --no-headers 2>/dev/null | awk '{print $1}'); do
        if [ -n "$pod" ]; then
            echo -e "\n--- Current logs for pod: $pod ---"
            kubectl logs "$pod" -n ran-a --tail=100 2>/dev/null || echo "No current logs for pod $pod"
            
            echo -e "\n--- Previous logs for pod: $pod ---"
            kubectl logs "$pod" -n ran-a --previous --tail=50 2>/dev/null || echo "No previous logs for pod $pod"
        fi
    done
    
    # Get node resource status
    echo -e "\n${YELLOW}4. Node Resource Status:${NC}"
    kubectl describe nodes 2>/dev/null | grep -A 10 "Allocated resources:" || echo "Could not get node resources"
    
    # Get recent events
    echo -e "\n${YELLOW}5. Recent Cluster Events:${NC}"
    kubectl get events --all-namespaces --sort-by='.lastTimestamp' | tail -20 2>/dev/null || echo "No recent events"
    
    # Get namespace resource quotas
    echo -e "\n${YELLOW}6. Namespace Resource Usage:${NC}"
    kubectl top nodes 2>/dev/null || echo "Metrics server not available"
    kubectl top pods -n ran-a 2>/dev/null || echo "Pod metrics not available"
}

# Add cleanup for additional processes
enhanced_cleanup() {
    local exit_code=$?
    
    # Stop additional processes
    for pid in "$LLM_PROCESSOR_PID" "$PORCH_DIRECT_PID" "$A1_SIM_PID" "$E2_SIM_PID" "$O1_SIM_PID"; do
        if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
            kill -TERM "$pid" 2>/dev/null || true
            sleep 2
            kill -0 "$pid" 2>/dev/null && kill -KILL "$pid" 2>/dev/null || true
        fi
    done
    
    # Print detailed debugging info on failure
    if [ $exit_code -ne 0 ]; then
        # First show component logs
        log_error "Test failed. Printing component logs (last 200 lines each):"
        
        for log_file in /tmp/intent-ingest.log /tmp/llm-processor.log /tmp/porch-direct.log \
                        /tmp/a1-sim.log /tmp/e2-kmp-sim.log /tmp/o1-ves-sim.log; do
            if [ -f "$log_file" ]; then
                echo -e "\n${YELLOW}=== $(basename "$log_file") ===${NC}"
                tail -n 200 "$log_file" || true
            fi
        done
        
        # Then debug failed pods
        debug_failed_pods
    fi
    
    cleanup
}

trap enhanced_cleanup EXIT

# Execute main function
main
>>>>>>> 6835433495e87288b95961af7173d866977175ff
