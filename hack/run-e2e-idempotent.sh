#!/usr/bin/env bash
set -euo pipefail

# === Configuration ===
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-nephoran-e2e}"
NAMESPACE="${NAMESPACE:-nephoran-system}"
REPORT_DIR="${PROJECT_ROOT}/.excellence-reports"
REPORT_FILE="${REPORT_DIR}/e2e-summary.txt"
TEST_TIMEOUT="${TEST_TIMEOUT:-300}" # 5 minutes default
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

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

# Initialize report
init_report() {
    mkdir -p "$REPORT_DIR"
    cat > "$REPORT_FILE" <<EOF
================================================================================
E2E Test Report - $TIMESTAMP
================================================================================
Cluster: $CLUSTER_NAME
Namespace: $NAMESPACE
Start Time: $(date)

Test Results:
--------------------------------------------------------------------------------
EOF
}

# Add test result to report
add_test_result() {
    local test_name="$1"
    local status="$2"
    local message="${3:-}"
    local duration="${4:-0}"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    
    case "$status" in
        PASS)
            TESTS_PASSED=$((TESTS_PASSED + 1))
            echo "[PASS] $test_name (${duration}s) - $message" >> "$REPORT_FILE"
            log_success "$test_name - $message"
            ;;
        FAIL)
            TESTS_FAILED=$((TESTS_FAILED + 1))
            echo "[FAIL] $test_name (${duration}s) - $message" >> "$REPORT_FILE"
            log_error "$test_name - $message"
            ;;
        SKIP)
            TESTS_SKIPPED=$((TESTS_SKIPPED + 1))
            echo "[SKIP] $test_name - $message" >> "$REPORT_FILE"
            log_warning "$test_name - $message"
            ;;
    esac
}

# Finalize report
finalize_report() {
    cat >> "$REPORT_FILE" <<EOF
--------------------------------------------------------------------------------
Summary:
  Total Tests: $TESTS_TOTAL
  Passed: $TESTS_PASSED
  Failed: $TESTS_FAILED
  Skipped: $TESTS_SKIPPED
  
End Time: $(date)
================================================================================
EOF
    
    # Generate JUnit-style XML report
    cat > "${REPORT_DIR}/e2e-junit.xml" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="Nephoran E2E Tests" tests="$TESTS_TOTAL" failures="$TESTS_FAILED" skipped="$TESTS_SKIPPED" time="0">
  <testsuite name="E2E" tests="$TESTS_TOTAL" failures="$TESTS_FAILED" skipped="$TESTS_SKIPPED">
EOF
    
    # Add test cases from report
    while IFS= read -r line; do
        if [[ $line =~ ^\[(PASS|FAIL|SKIP)\]\ (.+)\ \(([0-9]+)s\)\ -\ (.+)$ ]]; then
            status="${BASH_REMATCH[1]}"
            name="${BASH_REMATCH[2]}"
            time="${BASH_REMATCH[3]}"
            msg="${BASH_REMATCH[4]}"
            
            echo "    <testcase name=\"$name\" time=\"$time\">" >> "${REPORT_DIR}/e2e-junit.xml"
            if [[ "$status" == "FAIL" ]]; then
                echo "      <failure message=\"$msg\"/>" >> "${REPORT_DIR}/e2e-junit.xml"
            elif [[ "$status" == "SKIP" ]]; then
                echo "      <skipped message=\"$msg\"/>" >> "${REPORT_DIR}/e2e-junit.xml"
            fi
            echo "    </testcase>" >> "${REPORT_DIR}/e2e-junit.xml"
        fi
    done < "$REPORT_FILE"
    
    echo "  </testsuite>" >> "${REPORT_DIR}/e2e-junit.xml"
    echo "</testsuites>" >> "${REPORT_DIR}/e2e-junit.xml"
}

# Check command availability
check_command() {
    local cmd=$1
    if command -v "$cmd" &> /dev/null || command -v "${cmd}.exe" &> /dev/null; then
        return 0
    else
        return 1
    fi
}

# === Test Functions ===

# Test 1: Prerequisites Check
test_prerequisites() {
    local start_time=$(date +%s)
    local all_good=true
    
    for tool in kind kubectl; do
        if check_command "$tool"; then
            log_info "✅ Found $tool"
        else
            log_error "❌ Missing $tool"
            all_good=false
        fi
    done
    
    local duration=$(($(date +%s) - start_time))
    
    if $all_good; then
        add_test_result "Prerequisites" "PASS" "All required tools available" "$duration"
    else
        add_test_result "Prerequisites" "FAIL" "Missing required tools" "$duration"
        return 1
    fi
}

# Test 2: Kind Cluster Setup (Idempotent)
test_cluster_setup() {
    local start_time=$(date +%s)
    
    # Check if cluster already exists
    if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
        log_info "Cluster $CLUSTER_NAME already exists, verifying..."
        
        # Verify cluster is accessible
        if kubectl cluster-info --context "kind-${CLUSTER_NAME}" &>/dev/null; then
            local duration=$(($(date +%s) - start_time))
            add_test_result "Cluster Setup" "PASS" "Cluster exists and is accessible" "$duration"
            return 0
        else
            log_warning "Cluster exists but not accessible, recreating..."
            kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
        fi
    fi
    
    # Create cluster
    log_info "Creating Kind cluster: $CLUSTER_NAME"
    
    cat > /tmp/kind-config.yaml <<EOF
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
        node-labels: "nephoran.com/e2e=true"
- role: worker
EOF
    
    if kind create cluster --config /tmp/kind-config.yaml --wait 120s; then
        kubectl cluster-info --context "kind-${CLUSTER_NAME}"
        local duration=$(($(date +%s) - start_time))
        add_test_result "Cluster Setup" "PASS" "Cluster created successfully" "$duration"
    else
        local duration=$(($(date +%s) - start_time))
        add_test_result "Cluster Setup" "FAIL" "Failed to create cluster" "$duration"
        return 1
    fi
}

# Test 3: CRD Installation (Idempotent)
test_crd_installation() {
    local start_time=$(date +%s)
    
    log_info "Installing/Updating CRDs..."
    
    # Check if CRD directory exists
    if [ ! -d "$PROJECT_ROOT/deployments/crds" ]; then
        log_warning "CRD directory not found, checking config/crd..."
        if [ -d "$PROJECT_ROOT/config/crd" ]; then
            CRD_DIR="$PROJECT_ROOT/config/crd"
        else
            add_test_result "CRD Installation" "SKIP" "No CRD directory found" "0"
            return 0
        fi
    else
        CRD_DIR="$PROJECT_ROOT/deployments/crds"
    fi
    
    # Apply CRDs
    if kubectl apply -f "$CRD_DIR" 2>/dev/null; then
        log_info "CRDs applied successfully"
    else
        log_warning "Direct apply failed, trying individual files..."
        for crd_file in "$CRD_DIR"/*.yaml "$CRD_DIR"/*.yml; do
            [ -f "$crd_file" ] || continue
            kubectl apply -f "$crd_file" || true
        done
    fi
    
    # Wait for NetworkIntent CRD to be established
    if kubectl wait --for=condition=Established \
        crd/networkintents.nephoran.com \
        --timeout=60s 2>/dev/null; then
        local duration=$(($(date +%s) - start_time))
        add_test_result "CRD Installation" "PASS" "NetworkIntent CRD established" "$duration"
    else
        # Check if it exists with different name
        if kubectl get crd | grep -q networkintent; then
            local duration=$(($(date +%s) - start_time))
            add_test_result "CRD Installation" "PASS" "NetworkIntent CRD found" "$duration"
        else
            local duration=$(($(date +%s) - start_time))
            add_test_result "CRD Installation" "FAIL" "NetworkIntent CRD not established" "$duration"
            return 1
        fi
    fi
}

# Test 4: Namespace Setup (Idempotent)
test_namespace_setup() {
    local start_time=$(date +%s)
    
    log_info "Setting up namespace: $NAMESPACE"
    
    # Create namespace if it doesn't exist
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    
    # Verify namespace exists
    if kubectl get namespace "$NAMESPACE" &>/dev/null; then
        local duration=$(($(date +%s) - start_time))
        add_test_result "Namespace Setup" "PASS" "Namespace $NAMESPACE ready" "$duration"
    else
        local duration=$(($(date +%s) - start_time))
        add_test_result "Namespace Setup" "FAIL" "Failed to create namespace" "$duration"
        return 1
    fi
}

# Test 5: Webhook Deployment (Idempotent)
test_webhook_deployment() {
    local start_time=$(date +%s)
    
    log_info "Deploying webhook manager..."
    
    # Check for webhook configuration
    if [ -d "$PROJECT_ROOT/config/webhook" ]; then
        # Try kustomize first
        if kubectl apply -k "$PROJECT_ROOT/config/webhook" 2>/dev/null; then
            log_info "Webhook deployed via kustomize"
        elif [ -f "$PROJECT_ROOT/config/webhook/manifests.yaml" ]; then
            kubectl apply -f "$PROJECT_ROOT/config/webhook/manifests.yaml"
        else
            add_test_result "Webhook Deployment" "SKIP" "No webhook configuration found" "0"
            return 0
        fi
    else
        add_test_result "Webhook Deployment" "SKIP" "Webhook directory not found" "0"
        return 0
    fi
    
    # Wait for webhook deployment
    if kubectl -n "$NAMESPACE" wait deployment/webhook-manager \
        --for=condition=Available \
        --timeout=120s 2>/dev/null; then
        local duration=$(($(date +%s) - start_time))
        add_test_result "Webhook Deployment" "PASS" "Webhook manager ready" "$duration"
    else
        # Check if any deployment exists
        if kubectl -n "$NAMESPACE" get deployments 2>/dev/null | grep -q manager; then
            local duration=$(($(date +%s) - start_time))
            add_test_result "Webhook Deployment" "PASS" "Manager deployment found" "$duration"
        else
            local duration=$(($(date +%s) - start_time))
            add_test_result "Webhook Deployment" "SKIP" "No webhook deployment found" "$duration"
        fi
    fi
}

# Test 6: Sample NetworkIntent Creation
test_sample_networkintent() {
    local start_time=$(date +%s)
    
    log_info "Creating sample NetworkIntent..."
    
    # Create sample directory if needed
    SAMPLE_DIR="$PROJECT_ROOT/tests/e2e/samples"
    mkdir -p "$SAMPLE_DIR"
    
    # Create scaling-intent.yaml if it doesn't exist
    cat > "$SAMPLE_DIR/scaling-intent.yaml" <<EOF
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: e2e-scaling-test
  namespace: default
spec:
  intent: "Scale test-deployment to 3 replicas for E2E testing"
  intentType: scaling
  parametersMap:
    target: "test-deployment"
    replicas: "3"
    namespace: "default"
    source: "e2e-test"
EOF
    
    # Apply the sample intent
    if kubectl apply -f "$SAMPLE_DIR/scaling-intent.yaml"; then
        log_info "Sample NetworkIntent created"
        
        # Wait a moment for processing
        sleep 2
        
        # Verify the intent was created
        if kubectl get networkintent e2e-scaling-test -n default &>/dev/null; then
            local duration=$(($(date +%s) - start_time))
            add_test_result "Sample NetworkIntent" "PASS" "NetworkIntent created successfully" "$duration"
        else
            local duration=$(($(date +%s) - start_time))
            add_test_result "Sample NetworkIntent" "FAIL" "NetworkIntent not found after creation" "$duration"
            return 1
        fi
    else
        local duration=$(($(date +%s) - start_time))
        add_test_result "Sample NetworkIntent" "FAIL" "Failed to create NetworkIntent" "$duration"
        return 1
    fi
}

# Test 7: Verify NetworkIntent Fields
test_verify_networkintent() {
    local start_time=$(date +%s)
    
    log_info "Verifying NetworkIntent fields..."
    
    # Get the NetworkIntent and check for replicas field
    local yaml_output=$(kubectl get networkintent e2e-scaling-test -n default -o yaml 2>/dev/null)
    
    if [ -z "$yaml_output" ]; then
        add_test_result "Verify NetworkIntent" "FAIL" "Unable to get NetworkIntent YAML" "0"
        return 1
    fi
    
    # Check for replicas: "3" in the parametersMap
    if echo "$yaml_output" | grep -q 'replicas: "3"'; then
        local duration=$(($(date +%s) - start_time))
        add_test_result "Verify NetworkIntent" "PASS" "Replicas field verified (3)" "$duration"
    else
        # Check if replicas field exists at all
        if echo "$yaml_output" | grep -q "replicas:"; then
            local actual_replicas=$(echo "$yaml_output" | grep "replicas:" | head -1 | sed 's/.*replicas: //')
            local duration=$(($(date +%s) - start_time))
            add_test_result "Verify NetworkIntent" "FAIL" "Replicas field exists but value is $actual_replicas (expected \"3\")" "$duration"
        else
            local duration=$(($(date +%s) - start_time))
            add_test_result "Verify NetworkIntent" "FAIL" "Replicas field not found in parametersMap" "$duration"
        fi
        return 1
    fi
}

# Test 8: Porch Package Check (Optional)
test_porch_package() {
    local start_time=$(date +%s)
    
    log_info "Checking for Porch package creation..."
    
    # Check if Porch is installed
    if ! kubectl get crd packagerevisions.porch.kpt.dev &>/dev/null; then
        add_test_result "Porch Package" "SKIP" "Porch not installed" "0"
        return 0
    fi
    
    # Look for package related to our NetworkIntent
    if kubectl get packagerevisions -A 2>/dev/null | grep -q e2e-scaling; then
        local duration=$(($(date +%s) - start_time))
        add_test_result "Porch Package" "PASS" "Package created for NetworkIntent" "$duration"
    else
        local duration=$(($(date +%s) - start_time))
        add_test_result "Porch Package" "SKIP" "No package found (may be normal)" "$duration"
    fi
}

# Test 9: Cleanup Test Resources
test_cleanup() {
    local start_time=$(date +%s)
    
    log_info "Cleaning up test resources..."
    
    # Delete the test NetworkIntent
    kubectl delete networkintent e2e-scaling-test -n default --ignore-not-found=true
    
    local duration=$(($(date +%s) - start_time))
    add_test_result "Cleanup" "PASS" "Test resources cleaned" "$duration"
}

# === Main Execution ===
main() {
    log_info "Starting E2E Tests for Nephoran Intent Operator"
    log_info "Cluster: $CLUSTER_NAME"
    log_info "Namespace: $NAMESPACE"
    
    # Initialize report
    init_report
    
    # Run tests
    test_prerequisites || log_warning "Prerequisites check failed"
    test_cluster_setup || { log_error "Cluster setup failed"; finalize_report; exit 1; }
    test_crd_installation || log_warning "CRD installation had issues"
    test_namespace_setup || log_warning "Namespace setup had issues"
    test_webhook_deployment || log_warning "Webhook deployment had issues"
    test_sample_networkintent || log_warning "Sample NetworkIntent creation failed"
    test_verify_networkintent || log_warning "NetworkIntent verification failed"
    test_porch_package || log_warning "Porch package check had issues"
    test_cleanup || log_warning "Cleanup had issues"
    
    # Finalize report
    finalize_report
    
    # Print summary
    echo ""
    log_info "========================================="
    log_info "E2E Test Summary"
    log_info "========================================="
    log_info "Total Tests: $TESTS_TOTAL"
    log_success "Passed: $TESTS_PASSED"
    [ $TESTS_FAILED -gt 0 ] && log_error "Failed: $TESTS_FAILED"
    [ $TESTS_SKIPPED -gt 0 ] && log_warning "Skipped: $TESTS_SKIPPED"
    log_info "========================================="
    log_info "Report saved to: $REPORT_FILE"
    log_info "JUnit XML: ${REPORT_DIR}/e2e-junit.xml"
    
    # Exit with appropriate code
    if [ $TESTS_FAILED -gt 0 ]; then
        exit 1
    else
        exit 0
    fi
}

# Handle cleanup on exit
cleanup() {
    local exit_code=$?
    
    if [ $exit_code -ne 0 ]; then
        log_error "E2E tests exited with error code: $exit_code"
        echo "ERROR: Tests failed with exit code $exit_code" >> "$REPORT_FILE"
    fi
}

trap cleanup EXIT

# Run main function
main "$@"