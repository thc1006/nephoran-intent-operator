#!/bin/bash
# Conductor Loop Integration Test Script
# End-to-end testing of conductor-loop with real Kubernetes/Porch integration

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_NAMESPACE="conductor-loop-test"
TEST_TIMEOUT=300  # 5 minutes
CLEANUP_ON_EXIT=true

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log() { echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }

# Cleanup function
cleanup() {
    if [ "$CLEANUP_ON_EXIT" = true ]; then
        log "Cleaning up test resources..."
        kubectl delete namespace "$TEST_NAMESPACE" --ignore-not-found=true --timeout=60s
        docker-compose -f "$PROJECT_ROOT/deployments/docker-compose.conductor-loop.yml" down -v 2>/dev/null || true
        rm -rf "$PROJECT_ROOT/test-artifacts" 2>/dev/null || true
    fi
}

# Set up cleanup trap
trap cleanup EXIT INT TERM

# Usage function
usage() {
    cat << EOF
Conductor Loop Integration Test Suite

Usage: $0 [OPTIONS]

OPTIONS:
    -n, --namespace NAME    Test namespace (default: $TEST_NAMESPACE)
    -t, --timeout SECONDS  Test timeout (default: $TEST_TIMEOUT)
    --no-cleanup           Don't cleanup resources on exit
    --skip-build           Skip building conductor-loop binary
    --kind                 Use kind cluster for testing
    --docker               Use Docker Compose for testing
    -h, --help             Show this help

EXAMPLES:
    $0                     # Run all tests with defaults
    $0 --kind              # Use kind cluster
    $0 --docker            # Use Docker Compose only
    $0 --no-cleanup        # Keep test resources after completion

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                TEST_NAMESPACE="$2"
                shift 2
                ;;
            -t|--timeout)
                TEST_TIMEOUT="$2"
                shift 2
                ;;
            --no-cleanup)
                CLEANUP_ON_EXIT=false
                shift
                ;;
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --kind)
                USE_KIND=true
                shift
                ;;
            --docker)
                USE_DOCKER=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# Check dependencies
check_dependencies() {
    log "Checking dependencies..."
    
    local deps=("go" "kubectl")
    local optional_deps=("docker" "kind" "helm")
    local missing=()
    
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            missing+=("$dep")
        fi
    done
    
    if [ ${#missing[@]} -ne 0 ]; then
        error "Missing required dependencies: ${missing[*]}"
        exit 1
    fi
    
    for dep in "${optional_deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            warn "$dep not found - some features will be disabled"
        fi
    done
    
    success "Dependencies check passed"
}

# Check Kubernetes connectivity
check_kubernetes() {
    log "Checking Kubernetes connectivity..."
    
    if ! kubectl cluster-info &>/dev/null; then
        error "Cannot connect to Kubernetes cluster"
        error "Please ensure kubectl is configured and cluster is accessible"
        exit 1
    fi
    
    local context=$(kubectl config current-context)
    success "Connected to Kubernetes cluster (context: $context)"
}

# Build conductor-loop binary
build_conductor_loop() {
    if [ "${SKIP_BUILD:-false}" = true ]; then
        log "Skipping build (--skip-build specified)"
        return
    fi
    
    log "Building conductor-loop binary..."
    
    cd "$PROJECT_ROOT"
    
    local version=$(git describe --tags --always --dirty 2>/dev/null || echo "test")
    local commit=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    local date=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
    
    CGO_ENABLED=0 go build \
        -ldflags="-s -w -X main.version=$version -X main.commit=$commit -X main.date=$date" \
        -o bin/conductor-loop \
        ./cmd/conductor-loop
    
    success "Binary built successfully"
}

# Create test namespace
create_test_namespace() {
    log "Creating test namespace: $TEST_NAMESPACE"
    
    kubectl create namespace "$TEST_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    kubectl label namespace "$TEST_NAMESPACE" test=conductor-loop --overwrite
    
    success "Test namespace created"
}

# Deploy conductor-loop to Kubernetes
deploy_to_kubernetes() {
    log "Deploying conductor-loop to Kubernetes..."
    
    # Copy and modify manifests for testing
    local test_manifests_dir="$PROJECT_ROOT/test-artifacts/k8s"
    mkdir -p "$test_manifests_dir"
    
    # Copy base manifests
    cp -r "$PROJECT_ROOT/deployments/k8s/conductor-loop/"* "$test_manifests_dir/"
    
    # Update namespace in all manifests
    find "$test_manifests_dir" -name "*.yaml" -exec sed -i "s/namespace: conductor-loop/namespace: $TEST_NAMESPACE/g" {} \;
    
    # Deploy
    kubectl apply -k "$test_manifests_dir"
    
    # Wait for deployment to be ready
    log "Waiting for deployment to be ready..."
    if ! kubectl wait --for=condition=available --timeout="${TEST_TIMEOUT}s" \
        deployment/conductor-loop -n "$TEST_NAMESPACE"; then
        error "Deployment failed to become ready"
        kubectl describe deployment conductor-loop -n "$TEST_NAMESPACE"
        kubectl logs -l app.kubernetes.io/name=conductor-loop -n "$TEST_NAMESPACE" --tail=50
        exit 1
    fi
    
    success "Deployment is ready"
}

# Deploy with Docker Compose
deploy_with_docker() {
    log "Starting conductor-loop with Docker Compose..."
    
    cd "$PROJECT_ROOT"
    
    # Ensure directories exist
    mkdir -p handoff/in handoff/out deployments/conductor-loop/config
    
    # Create test configuration
    cat > deployments/conductor-loop/config/config.json << 'EOF'
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "metrics_port": 9090
  },
  "logging": {
    "level": "debug",
    "format": "json"
  },
  "conductor": {
    "handoff_in_path": "/data/handoff/in",
    "handoff_out_path": "/data/handoff/out",
    "poll_interval": "2s",
    "batch_size": 3
  },
  "porch": {
    "endpoint": "http://porch-server:7007",
    "timeout": "10s"
  }
}
EOF
    
    # Start services
    docker-compose -f deployments/docker-compose.conductor-loop.yml up -d
    
    # Wait for service to be ready
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s http://localhost:8080/healthz > /dev/null 2>&1; then
            success "Docker Compose deployment is ready"
            return
        fi
        
        ((attempt++))
        log "Waiting for service to be ready (attempt $attempt/$max_attempts)..."
        sleep 2
    done
    
    error "Service failed to become ready"
    docker-compose -f deployments/docker-compose.conductor-loop.yml logs
    exit 1
}

# Create test intent files
create_test_intents() {
    log "Creating test intent files..."
    
    local intent_dir
    if [ "${USE_DOCKER:-false}" = true ]; then
        intent_dir="$PROJECT_ROOT/handoff/in"
    else
        # For Kubernetes, we need to copy files to the pod
        intent_dir="$PROJECT_ROOT/test-artifacts/intents"
        mkdir -p "$intent_dir"
    fi
    
    # Test intent 1: Scale up
    cat > "$intent_dir/test-scale-up.json" << 'EOF'
{
  "kind": "NetworkIntent",
  "metadata": {
    "name": "test-scale-up-nf-sim",
    "namespace": "default",
    "labels": {
      "test": "integration",
      "action": "scale-up"
    }
  },
  "spec": {
    "target": "nf-sim",
    "action": "scale",
    "replicas": 2,
    "resources": {
      "cpu": "100m",
      "memory": "128Mi"
    },
    "reason": "Integration test: scale up"
  }
}
EOF
    
    # Test intent 2: Scale down
    cat > "$intent_dir/test-scale-down.json" << 'EOF'
{
  "kind": "NetworkIntent",
  "metadata": {
    "name": "test-scale-down-nf-sim",
    "namespace": "default",
    "labels": {
      "test": "integration",
      "action": "scale-down"
    }
  },
  "spec": {
    "target": "nf-sim",
    "action": "scale",
    "replicas": 1,
    "resources": {
      "cpu": "50m",
      "memory": "64Mi"
    },
    "reason": "Integration test: scale down"
  }
}
EOF
    
    # Test intent 3: Configuration update
    cat > "$intent_dir/test-config-update.json" << 'EOF'
{
  "kind": "NetworkIntent",
  "metadata": {
    "name": "test-config-update-nf-sim",
    "namespace": "default",
    "labels": {
      "test": "integration",
      "action": "config-update"
    }
  },
  "spec": {
    "target": "nf-sim",
    "action": "configure",
    "configuration": {
      "logging_level": "debug",
      "metrics_enabled": true
    },
    "reason": "Integration test: configuration update"
  }
}
EOF
    
    success "Test intent files created"
}

# Copy intents to Kubernetes pod
copy_intents_to_pod() {
    log "Copying intent files to conductor-loop pod..."
    
    local pod_name=$(kubectl get pods -n "$TEST_NAMESPACE" -l app.kubernetes.io/name=conductor-loop -o jsonpath='{.items[0].metadata.name}')
    
    if [ -z "$pod_name" ]; then
        error "No conductor-loop pods found"
        exit 1
    fi
    
    local intent_files=("test-scale-up.json" "test-scale-down.json" "test-config-update.json")
    
    for file in "${intent_files[@]}"; do
        kubectl cp "$PROJECT_ROOT/test-artifacts/intents/$file" \
            "$TEST_NAMESPACE/$pod_name:/data/handoff/in/$file"
    done
    
    success "Intent files copied to pod"
}

# Run health check tests
run_health_check_tests() {
    log "Running health check tests..."
    
    local base_url
    if [ "${USE_DOCKER:-false}" = true ]; then
        base_url="http://localhost:8080"
    else
        # Port forward for Kubernetes
        kubectl port-forward -n "$TEST_NAMESPACE" svc/conductor-loop 8080:8080 &
        local pf_pid=$!
        sleep 5
        base_url="http://localhost:8080"
        
        # Cleanup function for port-forward
        cleanup_port_forward() {
            kill $pf_pid 2>/dev/null || true
        }
        trap cleanup_port_forward EXIT
    fi
    
    # Test health endpoint
    if curl -f -s "$base_url/healthz" > /dev/null; then
        success "Health check endpoint is working"
    else
        error "Health check endpoint failed"
        return 1
    fi
    
    # Test readiness endpoint
    if curl -f -s "$base_url/readyz" > /dev/null; then
        success "Readiness check endpoint is working"
    else
        error "Readiness check endpoint failed"
        return 1
    fi
    
    # Test metrics endpoint
    if curl -f -s "http://localhost:9090/metrics" > /dev/null; then
        success "Metrics endpoint is working"
    else
        warn "Metrics endpoint not accessible (this may be expected)"
    fi
}

# Test intent processing
test_intent_processing() {
    log "Testing intent processing..."
    
    # Create and deploy intents
    create_test_intents
    
    if [ "${USE_DOCKER:-false}" = true ]; then
        # For Docker, files are already in the right place
        log "Intent files are in handoff/in directory"
    else
        # For Kubernetes, copy files to pod
        copy_intents_to_pod
    fi
    
    # Wait for processing
    log "Waiting for intents to be processed..."
    local max_wait=60
    local waited=0
    
    while [ $waited -lt $max_wait ]; do
        local processed_count=0
        
        if [ "${USE_DOCKER:-false}" = true ]; then
            # Count files in handoff/out
            if [ -d "$PROJECT_ROOT/handoff/out" ]; then
                processed_count=$(find "$PROJECT_ROOT/handoff/out" -name "*.json" | wc -l)
            fi
        else
            # Check pod's handoff/out directory
            local pod_name=$(kubectl get pods -n "$TEST_NAMESPACE" -l app.kubernetes.io/name=conductor-loop -o jsonpath='{.items[0].metadata.name}')
            processed_count=$(kubectl exec -n "$TEST_NAMESPACE" "$pod_name" -- find /data/handoff/out -name "*.json" 2>/dev/null | wc -l)
        fi
        
        if [ "$processed_count" -ge 3 ]; then
            success "All intents have been processed"
            return 0
        fi
        
        log "Processed $processed_count/3 intents, waiting..."
        sleep 5
        ((waited += 5))
    done
    
    error "Timeout waiting for intent processing"
    return 1
}

# Run performance tests
run_performance_tests() {
    log "Running performance tests..."
    
    # Create a large number of intent files
    local intent_dir
    if [ "${USE_DOCKER:-false}" = true ]; then
        intent_dir="$PROJECT_ROOT/handoff/in"
    else
        intent_dir="$PROJECT_ROOT/test-artifacts/perf-intents"
        mkdir -p "$intent_dir"
    fi
    
    log "Creating 50 intent files for performance testing..."
    for i in {1..50}; do
        cat > "$intent_dir/perf-test-$i.json" << EOF
{
  "kind": "NetworkIntent",
  "metadata": {
    "name": "perf-test-nf-sim-$i",
    "namespace": "default",
    "labels": {
      "test": "performance",
      "batch": "1"
    }
  },
  "spec": {
    "target": "nf-sim-$i",
    "action": "scale",
    "replicas": $((i % 5 + 1)),
    "reason": "Performance test $i"
  }
}
EOF
    done
    
    # Copy to pod if using Kubernetes
    if [ "${USE_DOCKER:-false}" != true ]; then
        local pod_name=$(kubectl get pods -n "$TEST_NAMESPACE" -l app.kubernetes.io/name=conductor-loop -o jsonpath='{.items[0].metadata.name}')
        
        for i in {1..50}; do
            kubectl cp "$intent_dir/perf-test-$i.json" \
                "$TEST_NAMESPACE/$pod_name:/data/handoff/in/perf-test-$i.json"
        done
    fi
    
    # Measure processing time
    local start_time=$(date +%s)
    log "Starting performance test at $(date)"
    
    # Wait for all files to be processed
    local max_wait=300  # 5 minutes
    local waited=0
    
    while [ $waited -lt $max_wait ]; do
        local processed_count
        
        if [ "${USE_DOCKER:-false}" = true ]; then
            processed_count=$(find "$PROJECT_ROOT/handoff/out" -name "perf-test-*.json" 2>/dev/null | wc -l)
        else
            local pod_name=$(kubectl get pods -n "$TEST_NAMESPACE" -l app.kubernetes.io/name=conductor-loop -o jsonpath='{.items[0].metadata.name}')
            processed_count=$(kubectl exec -n "$TEST_NAMESPACE" "$pod_name" -- find /data/handoff/out -name "perf-test-*.json" 2>/dev/null | wc -l)
        fi
        
        if [ "$processed_count" -ge 50 ]; then
            local end_time=$(date +%s)
            local duration=$((end_time - start_time))
            success "Performance test completed in ${duration}s (50 files processed)"
            
            if [ $duration -lt 120 ]; then
                success "Performance is good (< 2 minutes)"
            else
                warn "Performance is slow (> 2 minutes)"
            fi
            return 0
        fi
        
        log "Processed $processed_count/50 files, waiting... (${waited}s elapsed)"
        sleep 10
        ((waited += 10))
    done
    
    error "Performance test timed out"
    return 1
}

# Collect logs and metrics
collect_diagnostics() {
    log "Collecting diagnostics..."
    
    local diag_dir="$PROJECT_ROOT/test-artifacts/diagnostics"
    mkdir -p "$diag_dir"
    
    if [ "${USE_DOCKER:-false}" = true ]; then
        # Docker diagnostics
        docker-compose -f "$PROJECT_ROOT/deployments/docker-compose.conductor-loop.yml" logs > "$diag_dir/docker-logs.txt" 2>&1
        docker stats --no-stream > "$diag_dir/docker-stats.txt" 2>&1 || true
    else
        # Kubernetes diagnostics
        kubectl describe deployment conductor-loop -n "$TEST_NAMESPACE" > "$diag_dir/deployment.txt" 2>&1
        kubectl get events -n "$TEST_NAMESPACE" > "$diag_dir/events.txt" 2>&1
        kubectl logs -l app.kubernetes.io/name=conductor-loop -n "$TEST_NAMESPACE" --tail=1000 > "$diag_dir/pods.txt" 2>&1
        kubectl top pods -n "$TEST_NAMESPACE" > "$diag_dir/resource-usage.txt" 2>&1 || true
    fi
    
    success "Diagnostics collected in $diag_dir"
}

# Main test execution
run_integration_tests() {
    log "Starting conductor-loop integration tests..."
    
    local test_results=()
    
    # Setup
    check_dependencies
    build_conductor_loop
    
    if [ "${USE_DOCKER:-false}" = true ]; then
        deploy_with_docker
    else
        check_kubernetes
        create_test_namespace
        deploy_to_kubernetes
    fi
    
    # Run tests
    log "Running test suite..."
    
    if run_health_check_tests; then
        test_results+=("✓ Health checks")
    else
        test_results+=("✗ Health checks")
    fi
    
    if test_intent_processing; then
        test_results+=("✓ Intent processing")
    else
        test_results+=("✗ Intent processing")
    fi
    
    if run_performance_tests; then
        test_results+=("✓ Performance tests")
    else
        test_results+=("✗ Performance tests")
    fi
    
    # Collect diagnostics
    collect_diagnostics
    
    # Report results
    log "Test Results:"
    for result in "${test_results[@]}"; do
        if [[ $result == ✓* ]]; then
            echo -e "  ${GREEN}$result${NC}"
        else
            echo -e "  ${RED}$result${NC}"
        fi
    done
    
    # Check if any tests failed
    local failed_count=$(echo "${test_results[@]}" | grep -o "✗" | wc -l)
    if [ "$failed_count" -eq 0 ]; then
        success "All integration tests passed!"
        return 0
    else
        error "$failed_count test(s) failed"
        return 1
    fi
}

# Main execution
main() {
    parse_args "$@"
    
    log "Conductor Loop Integration Test Suite"
    log "====================================="
    log "Namespace: $TEST_NAMESPACE"
    log "Timeout: ${TEST_TIMEOUT}s"
    log "Cleanup on exit: $CLEANUP_ON_EXIT"
    
    if run_integration_tests; then
        success "Integration test suite completed successfully"
        exit 0
    else
        error "Integration test suite failed"
        exit 1
    fi
}

# Run main function
main "$@"