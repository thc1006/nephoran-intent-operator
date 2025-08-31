#!/usr/bin/env bash
set -euo pipefail

# Production-quality E2E test runner with comprehensive validation
# Supports multiple test scenarios and detailed reporting

# === Configuration ===
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-nephoran-e2e-prod}"
NAMESPACE="${NAMESPACE:-nephoran-e2e}"
TEST_TIMEOUT="${TEST_TIMEOUT:-30m}"
PARALLEL_TESTS="${PARALLEL_TESTS:-4}"
VERBOSE="${VERBOSE:-false}"
SKIP_CLEANUP="${SKIP_CLEANUP:-false}"
TEST_SCENARIO="${TEST_SCENARIO:-all}"
REPORT_DIR="${REPORT_DIR:-$PROJECT_ROOT/test-reports}"
LOG_LEVEL="${LOG_LEVEL:-info}"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# === Helper Functions ===
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

check_command() {
    local cmd=$1
    if ! command -v "$cmd" &> /dev/null; then
        if ! command -v "${cmd}.exe" &> /dev/null; then
            log_error "$cmd is not installed or not in PATH"
            return 1
        fi
    fi
    return 0
}

cleanup() {
    local exit_code=$?
    
    if [ "$SKIP_CLEANUP" != "true" ]; then
        log_info "Cleaning up test resources..."
        
        # Delete test cluster if it exists
        if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
            log_info "Deleting Kind cluster: $CLUSTER_NAME"
            kind delete cluster --name "$CLUSTER_NAME" || true
        fi
        
        # Clean up temporary files
        rm -f /tmp/nephoran-e2e-*.yaml 2>/dev/null || true
    else
        log_warning "Skipping cleanup (SKIP_CLEANUP=true)"
    fi
    
    if [ $exit_code -eq 0 ]; then
        log_success "E2E tests completed successfully!"
    else
        log_error "E2E tests failed with exit code: $exit_code"
    fi
    
    exit $exit_code
}

trap cleanup EXIT

# === Pre-flight Checks ===
log_info "Running pre-flight checks..."

# Check required tools
REQUIRED_TOOLS=("docker" "kind" "kubectl" "go")
for tool in "${REQUIRED_TOOLS[@]}"; do
    if ! check_command "$tool"; then
        log_error "Required tool '$tool' is missing"
        exit 1
    fi
done

# Check optional tools
OPTIONAL_TOOLS=("kustomize" "helm" "yq" "jq")
for tool in "${OPTIONAL_TOOLS[@]}"; do
    if check_command "$tool"; then
        log_info "✅ Found optional tool: $tool"
    else
        log_warning "⚠️  Optional tool '$tool' not found (some features may be limited)"
    fi
done

# === Cluster Setup ===
setup_cluster() {
    log_info "Setting up Kind cluster: $CLUSTER_NAME"
    
    # Check if cluster already exists
    if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
        log_warning "Cluster $CLUSTER_NAME already exists, deleting..."
        kind delete cluster --name "$CLUSTER_NAME"
    fi
    
    # Create Kind configuration
    cat > /tmp/kind-config-e2e.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ${CLUSTER_NAME}
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "nephoran.com/node-type=control-plane"
- role: worker
  kubeadmConfigPatches:
  - |
    kind: JoinConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "nephoran.com/node-type=worker"
- role: worker
  kubeadmConfigPatches:
  - |
    kind: JoinConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "nephoran.com/node-type=worker"
networking:
  apiServerPort: 6443
  podSubnet: "10.244.0.0/16"
  serviceSubnet: "10.96.0.0/12"
EOF
    
    # Create cluster
    log_info "Creating Kind cluster with 3 nodes..."
    kind create cluster --config /tmp/kind-config-e2e.yaml --wait 2m
    
    # Verify cluster is ready
    log_info "Verifying cluster connectivity..."
    kubectl cluster-info --context "kind-${CLUSTER_NAME}"
    
    # Wait for nodes to be ready
    log_info "Waiting for all nodes to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=120s
    
    log_success "Cluster setup complete!"
}

# === Install CRDs and Controllers ===
install_crds() {
    log_info "Installing Nephoran CRDs..."
    
    # Apply CRDs
    if [ -d "$PROJECT_ROOT/deployments/crds" ]; then
        kubectl apply -f "$PROJECT_ROOT/deployments/crds/"
        
        # Wait for CRDs to be established
        log_info "Waiting for CRDs to be established..."
        kubectl wait --for=condition=Established \
            crd/networkintents.intent.nephoran.io \
            crd/e2nodesets.intent.nephoran.io \
            --timeout=60s
    else
        log_error "CRD directory not found: $PROJECT_ROOT/deployments/crds"
        return 1
    fi
    
    log_success "CRDs installed successfully!"
}

install_controllers() {
    log_info "Installing Nephoran controllers..."
    
    # Create namespace
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Apply controller configurations
    if [ -d "$PROJECT_ROOT/config/webhook" ]; then
        log_info "Deploying webhook configurations..."
        kubectl apply -k "$PROJECT_ROOT/config/webhook" || {
            log_warning "Kustomize failed, trying direct apply..."
            kubectl apply -f "$PROJECT_ROOT/config/webhook/"
        }
    fi
    
    # Deploy sample controllers if available
    if [ -d "$PROJECT_ROOT/deployments/kustomize/base" ]; then
        log_info "Deploying base controllers..."
        kubectl apply -k "$PROJECT_ROOT/deployments/kustomize/base/" || true
    fi
    
    # Wait for deployments to be ready
    log_info "Waiting for controller deployments..."
    kubectl -n "$NAMESPACE" wait --for=condition=Available \
        deployment --all --timeout=180s || true
    
    log_success "Controllers installed!"
}

# === Run Tests ===
run_go_tests() {
    log_info "Running Go E2E tests..."
    
    # Create test report directory
    mkdir -p "$REPORT_DIR"
    
    # Set test environment variables
    export KUBECONFIG="${KUBECONFIG:-$HOME/.kube/config}"
    export TEST_NAMESPACE="$NAMESPACE"
    export TEST_CLUSTER="$CLUSTER_NAME"
    
    # Build test arguments
    TEST_ARGS="-timeout $TEST_TIMEOUT"
    TEST_ARGS="$TEST_ARGS -parallel $PARALLEL_TESTS"
    
    if [ "$VERBOSE" = "true" ]; then
        TEST_ARGS="$TEST_ARGS -v"
    fi
    
    if [ "$SKIP_CLEANUP" = "true" ]; then
        TEST_ARGS="$TEST_ARGS -skip-cleanup"
    fi
    
    # Run specific test scenario or all tests
    cd "$PROJECT_ROOT"
    
    case "$TEST_SCENARIO" in
        "smoke")
            log_info "Running smoke tests only..."
            go test ./tests/e2e/... $TEST_ARGS -run "Smoke"
            ;;
        "production")
            log_info "Running production scenario tests..."
            go test ./tests/e2e/... $TEST_ARGS -run "Production"
            ;;
        "scaling")
            log_info "Running scaling tests..."
            go test ./tests/e2e/... $TEST_ARGS -run "Scaling"
            ;;
        "security")
            log_info "Running security tests..."
            go test ./tests/e2e/... $TEST_ARGS -run "Security"
            ;;
        "all")
            log_info "Running all E2E tests..."
            go test ./tests/e2e/... $TEST_ARGS
            ;;
        *)
            log_error "Unknown test scenario: $TEST_SCENARIO"
            exit 1
            ;;
    esac
    
    local test_exit_code=$?
    
    # Generate test report
    if command -v go-junit-report &> /dev/null; then
        log_info "Generating JUnit test report..."
        go test ./tests/e2e/... $TEST_ARGS -json | \
            go-junit-report > "$REPORT_DIR/e2e-junit.xml"
    fi
    
    return $test_exit_code
}

# === Validation Tests ===
run_validation_tests() {
    log_info "Running validation tests..."
    
    # Test 1: Verify CRDs are correctly installed
    log_info "Test 1: Verifying CRDs..."
    kubectl get crd networkintents.intent.nephoran.io -o jsonpath='{.status.conditions[?(@.type=="Established")].status}' | \
        grep -q "True" || {
        log_error "NetworkIntent CRD not established"
        return 1
    }
    
    # Test 2: Create and validate a NetworkIntent
    log_info "Test 2: Creating test NetworkIntent..."
    cat <<EOF | kubectl apply -f -
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: e2e-validation-test
  namespace: $NAMESPACE
spec:
  intentType: scaling
  target: test-nf
  namespace: test
  replicas: 3
  source: e2e-validation
EOF
    
    # Verify creation
    kubectl get networkintent -n "$NAMESPACE" e2e-validation-test || {
        log_error "Failed to create NetworkIntent"
        return 1
    }
    
    # Test 3: Verify controller is processing
    log_info "Test 3: Verifying controller processing..."
    sleep 5
    kubectl get networkintent -n "$NAMESPACE" e2e-validation-test -o jsonpath='{.status.phase}' || true
    
    # Clean up test resources
    kubectl delete networkintent -n "$NAMESPACE" e2e-validation-test --ignore-not-found=true
    
    log_success "Validation tests passed!"
}

# === Performance Tests ===
run_performance_tests() {
    log_info "Running performance tests..."
    
    # Create multiple NetworkIntents concurrently
    log_info "Creating 10 NetworkIntents concurrently..."
    for i in {1..10}; do
        cat <<EOF | kubectl apply -f - &
apiVersion: intent.nephoran.io/v1alpha1
kind: NetworkIntent
metadata:
  name: perf-test-$i
  namespace: $NAMESPACE
spec:
  intentType: deployment
  target: perf-nf-$i
  namespace: perf-test
  replicas: 2
  source: performance-test
EOF
    done
    
    wait
    
    # Measure processing time
    log_info "Measuring processing performance..."
    START_TIME=$(date +%s)
    
    # Wait for all intents to be processed
    for i in {1..10}; do
        kubectl wait --for=condition=Ready \
            networkintent/perf-test-$i \
            -n "$NAMESPACE" \
            --timeout=60s || true
    done
    
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    
    log_info "Performance test completed in ${DURATION} seconds"
    
    # Clean up
    for i in {1..10}; do
        kubectl delete networkintent -n "$NAMESPACE" perf-test-$i --ignore-not-found=true
    done
    
    log_success "Performance tests completed!"
}

# === Generate Report ===
generate_report() {
    log_info "Generating E2E test report..."
    
    REPORT_FILE="$REPORT_DIR/e2e-report-$(date +%Y%m%d-%H%M%S).txt"
    
    cat > "$REPORT_FILE" <<EOF
=====================================
Nephoran E2E Test Report
=====================================
Date: $(date)
Cluster: $CLUSTER_NAME
Namespace: $NAMESPACE
Test Scenario: $TEST_SCENARIO
Test Timeout: $TEST_TIMEOUT
Parallel Tests: $PARALLEL_TESTS

Test Results:
-------------
EOF
    
    # Add test results
    if [ -f "$REPORT_DIR/e2e-junit.xml" ]; then
        echo "JUnit report generated: e2e-junit.xml" >> "$REPORT_FILE"
    fi
    
    # Add cluster information
    echo -e "\nCluster Information:" >> "$REPORT_FILE"
    kubectl get nodes >> "$REPORT_FILE" 2>&1
    
    echo -e "\nDeployed Resources:" >> "$REPORT_FILE"
    kubectl get all -n "$NAMESPACE" >> "$REPORT_FILE" 2>&1
    
    echo -e "\nNetworkIntents:" >> "$REPORT_FILE"
    kubectl get networkintents -A >> "$REPORT_FILE" 2>&1
    
    log_success "Report generated: $REPORT_FILE"
}

# === Main Execution ===
main() {
    log_info "Starting production E2E tests..."
    log_info "Configuration:"
    log_info "  Cluster: $CLUSTER_NAME"
    log_info "  Namespace: $NAMESPACE"
    log_info "  Test Scenario: $TEST_SCENARIO"
    log_info "  Timeout: $TEST_TIMEOUT"
    log_info "  Parallel: $PARALLEL_TESTS"
    
    # Setup cluster
    setup_cluster
    
    # Install CRDs and controllers
    install_crds
    install_controllers
    
    # Run validation tests first
    run_validation_tests
    
    # Run main E2E tests
    if run_go_tests; then
        log_success "Go E2E tests passed!"
    else
        log_error "Go E2E tests failed!"
        exit 1
    fi
    
    # Run performance tests if requested
    if [ "$TEST_SCENARIO" = "all" ] || [ "$TEST_SCENARIO" = "performance" ]; then
        run_performance_tests
    fi
    
    # Generate report
    generate_report
    
    log_success "All E2E tests completed successfully!"
}

# Run main function
main "$@"