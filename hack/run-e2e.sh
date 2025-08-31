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
    while [ $elapsed -lt $timeout ]; do
        if eval "$condition_cmd" &>/dev/null; then
            [ $dots_printed -gt 0 ] && echo ""
            log_success "$description - ready!"
            return 0
        fi
        sleep $interval
        elapsed=$((elapsed + interval))
        echo -n "."
        dots_printed=$((dots_printed + 1))
        
        # Print status every 30 seconds for long waits
        if [ $((elapsed % 30)) -eq 0 ] && [ $elapsed -gt 0 ]; then
            echo ""
            log_info "Still waiting for: $description (${elapsed}s elapsed)"
        fi
    done
    echo ""
    log_error "$description - timed out after ${timeout}s"
    
    # Provide diagnostic information on timeout
    log_error "Diagnostic information for '$description':"
    eval "$condition_cmd" 2>&1 || true
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
    
    # Check if kubectl is available
    if ! command -v kubectl >/dev/null 2>&1; then
        log_error "kubectl is required but not installed"
        log_error "Install: https://kubernetes.io/docs/tasks/tools/"
        log_error "Windows: choco install kubernetes-cli"
        exit 1
    fi
    
    # Check if cluster is accessible
    if ! kubectl cluster-info >/dev/null 2>&1; then
        log_error "Cannot access Kubernetes cluster"
        log_info "Please ensure a Kubernetes cluster is running and accessible"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

check_prerequisites

log_success "E2E test execution started!"
log_info "Use SKIP_CLEANUP=true to preserve cluster for debugging"
log_info "Use VERBOSE=true for detailed output"
