#!/bin/bash

# Comprehensive E2E test runner for Nephoran Intent Operator (Unix/Linux version)
# This script orchestrates the execution of all end-to-end tests

set -euo pipefail

# Default values
TEST_SUITE="basic"
SKIP_SERVICE_CHECKS=false
VERBOSE=false
GENERATE_REPORT=false
CLEANUP_ONLY=false
OUTPUT_DIR="./test-results/e2e"
TIMEOUT=30
NAMESPACE="default"
KUBE_CONFIG=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Global variables
START_TIME=$(date +%s)
TEST_RESULTS=()

# Logging functions
log() {
    local level="$1"
    local message="$2"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    case "$level" in
        "ERROR")
            echo -e "${RED}[$timestamp] [ERROR] $message${NC}" >&2
            ;;
        "WARN")
            echo -e "${YELLOW}[$timestamp] [WARN] $message${NC}"
            ;;
        "SUCCESS")
            echo -e "${GREEN}[$timestamp] [SUCCESS] $message${NC}"
            ;;
        "INFO")
            echo -e "[$timestamp] [INFO] $message"
            ;;
        *)
            echo -e "[$timestamp] $message"
            ;;
    esac
    
    if [[ "$VERBOSE" == "true" ]]; then
        echo "[$timestamp] [$level] $message" >> "$OUTPUT_DIR/e2e-test.log"
    fi
}

show_banner() {
    echo -e "${CYAN}"
    cat << 'EOF'
╔═══════════════════════════════════════════════════════╗
║         Nephoran Intent Operator E2E Tests           ║
║                                                       ║
║  Test Suite: %-10s                        ║
║  Namespace:  %-10s                        ║
║  Timeout:    %s minutes                          ║
╚═══════════════════════════════════════════════════════╝
EOF
    echo -e "${NC}"
    printf "${CYAN}║  Test Suite: %-10s                        ║${NC}\n" "$TEST_SUITE"
    printf "${CYAN}║  Namespace:  %-10s                        ║${NC}\n" "$NAMESPACE"
    printf "${CYAN}║  Timeout:    %-2s minutes                          ║${NC}\n" "$TIMEOUT"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════╝${NC}"
}

show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Options:
    -s, --suite SUITE       Test suite to run (all, basic, integration, workflow, stress, cleanup)
    -n, --namespace NS      Kubernetes namespace to use (default: default)
    -t, --timeout MIN       Test timeout in minutes (default: 30)
    -o, --output DIR        Output directory for test results (default: ./test-results/e2e)
    -k, --kubeconfig PATH   Path to kubeconfig file
    --skip-service-checks   Skip service dependency checks
    --verbose               Enable verbose output
    --generate-report       Generate HTML test report
    --cleanup-only          Only perform cleanup operations
    -h, --help              Show this help message

Examples:
    $0 --suite all --verbose
    $0 --suite basic --skip-service-checks --output ./results
    $0 --cleanup-only
EOF
}

parse_arguments() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -s|--suite)
                TEST_SUITE="$2"
                shift 2
                ;;
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            -o|--output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            -k|--kubeconfig)
                KUBE_CONFIG="$2"
                shift 2
                ;;
            --skip-service-checks)
                SKIP_SERVICE_CHECKS=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --generate-report)
                GENERATE_REPORT=true
                shift
                ;;
            --cleanup-only)
                CLEANUP_ONLY=true
                shift
                ;;
            -h|--help)
                show_usage
                exit 0
                ;;
            *)
                echo "Unknown option: $1" >&2
                show_usage >&2
                exit 1
                ;;
        esac
    done
    
    # Validate test suite
    case "$TEST_SUITE" in
        all|basic|integration|workflow|stress|cleanup)
            ;;
        *)
            log "ERROR" "Invalid test suite: $TEST_SUITE"
            show_usage >&2
            exit 1
            ;;
    esac
}

check_prerequisites() {
    log "INFO" "Checking prerequisites..."
    
    local prerequisites=("go" "kubectl" "ginkgo")
    local missing=()
    
    for prereq in "${prerequisites[@]}"; do
        if command -v "$prereq" >/dev/null 2>&1; then
            log "SUCCESS" "✓ $prereq is available"
        else
            missing+=("$prereq")
            log "ERROR" "✗ $prereq is missing or not accessible"
        fi
    done
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        log "ERROR" "Missing required prerequisites: ${missing[*]}"
        exit 1
    fi
}

test_kubernetes_connection() {
    log "INFO" "Testing Kubernetes connection..."
    
    if [[ -n "$KUBE_CONFIG" ]]; then
        export KUBECONFIG="$KUBE_CONFIG"
    fi
    
    if ! kubectl cluster-info >/dev/null 2>&1; then
        log "ERROR" "kubectl cluster-info failed"
        exit 1
    fi
    
    log "SUCCESS" "✓ Kubernetes cluster is accessible"
    
    # Check namespace exists
    if ! kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
        log "WARN" "Creating namespace $NAMESPACE..."
        if ! kubectl create namespace "$NAMESPACE"; then
            log "ERROR" "Failed to create namespace $NAMESPACE"
            exit 1
        fi
    fi
    
    log "SUCCESS" "✓ Namespace $NAMESPACE is available"
}

test_service_dependencies() {
    if [[ "$SKIP_SERVICE_CHECKS" == "true" ]]; then
        log "WARN" "Skipping service dependency checks"
        return
    fi
    
    log "INFO" "Checking service dependencies..."
    
    local services=(
        "Controller:http://localhost:8080/health:false"
        "LLM_Processor:http://localhost:8080/health:true"
        "RAG_Service:http://localhost:8001/health:true"
    )
    
    for service_info in "${services[@]}"; do
        IFS=':' read -r name url optional <<< "$service_info"
        
        if curl -s --max-time 5 "$url" >/dev/null 2>&1; then
            log "SUCCESS" "✓ $name is healthy"
        else
            if [[ "$optional" == "true" ]]; then
                log "WARN" "⚠ $name is not available (tests will be skipped)"
            else
                log "ERROR" "✗ $name is not healthy"
            fi
        fi
    done
}

initialize_test_environment() {
    log "INFO" "Initializing test environment..."
    
    # Create output directory
    mkdir -p "$OUTPUT_DIR"
    
    # Set environment variables for tests
    export E2E_TEST_NAMESPACE="$NAMESPACE"
    export E2E_TEST_TIMEOUT=$((TIMEOUT * 60)) # Convert to seconds
    export E2E_OUTPUT_DIR="$OUTPUT_DIR"
    
    log "SUCCESS" "✓ Test environment initialized"
}

run_test_suite() {
    local suite="$1"
    log "INFO" "Executing test suite: $suite"
    
    local test_dir
    test_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    
    local -A suite_mapping=(
        ["basic"]="networkintent_lifecycle_test.go health_monitoring_test.go"
        ["integration"]="llm_processor_integration_test.go rag_service_integration_test.go"
        ["workflow"]="full_workflow_test.go error_handling_timeout_test.go"
        ["stress"]="concurrency_cleanup_test.go"
        ["cleanup"]="concurrency_cleanup_test.go"
        ["all"]="networkintent_lifecycle_test.go health_monitoring_test.go llm_processor_integration_test.go rag_service_integration_test.go full_workflow_test.go error_handling_timeout_test.go concurrency_cleanup_test.go"
    )
    
    local test_files="${suite_mapping[$suite]}"
    if [[ -z "$test_files" ]]; then
        log "ERROR" "Unknown test suite: $suite"
        exit 1
    fi
    
    local suite_success=true
    local suite_start_time=$(date +%s)
    
    cd "$test_dir"
    
    for test_file in $test_files; do
        log "INFO" "Running test file: $test_file"
        
        local test_start_time=$(date +%s)
        local ginkgo_cmd="ginkgo run --timeout=${TIMEOUT}m --json-report=$OUTPUT_DIR/${test_file}.json --junit-report=$OUTPUT_DIR/${test_file}.xml"
        
        if [[ "$VERBOSE" == "true" ]]; then
            ginkgo_cmd="$ginkgo_cmd -v"
        fi
        
        ginkgo_cmd="$ginkgo_cmd $test_file"
        
        log "INFO" "Executing: $ginkgo_cmd"
        
        if eval "$ginkgo_cmd"; then
            local test_duration=$(($(date +%s) - test_start_time))
            log "SUCCESS" "✓ $test_file completed successfully (${test_duration}s)"
        else
            local test_duration=$(($(date +%s) - test_start_time))
            log "ERROR" "✗ $test_file failed (${test_duration}s)"
            suite_success=false
        fi
    done
    
    local suite_duration=$(($(date +%s) - suite_start_time))
    
    if [[ "$suite_success" == "true" ]]; then
        log "SUCCESS" "Test suite '$suite' completed successfully (${suite_duration}s)"
    else
        log "ERROR" "Test suite '$suite' failed (${suite_duration}s)"
    fi
    
    return $([ "$suite_success" == "true" ] && echo 0 || echo 1)
}

generate_test_report() {
    if [[ "$GENERATE_REPORT" != "true" ]]; then
        return
    fi
    
    log "INFO" "Generating test report..."
    
    local report_path="$OUTPUT_DIR/e2e-test-report.html"
    local execution_time=$(date '+%Y-%m-%d %H:%M:%S')
    local total_duration=$(( $(date +%s) - START_TIME ))
    
    # Count JSON report files to determine test results
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    
    if compgen -G "$OUTPUT_DIR/*.json" > /dev/null; then
        for json_file in "$OUTPUT_DIR"/*.json; do
            if [[ -f "$json_file" ]]; then
                total_tests=$((total_tests + 1))
                # Simple check - if file contains success indicators
                if grep -q '"State":"passed"' "$json_file" 2>/dev/null; then
                    passed_tests=$((passed_tests + 1))
                else
                    failed_tests=$((failed_tests + 1))
                fi
            fi
        done
    fi
    
    cat > "$report_path" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Nephoran E2E Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background: #f0f8ff; padding: 20px; border-radius: 5px; }
        .summary { background: #f9f9f9; padding: 15px; margin: 20px 0; border-radius: 5px; }
        .success { color: #28a745; }
        .failure { color: #dc3545; }
        .test-files { margin: 20px 0; }
        .test-file { padding: 10px; margin: 5px 0; border: 1px solid #ddd; border-radius: 3px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Nephoran Intent Operator - E2E Test Report</h1>
        <p><strong>Test Suite:</strong> $TEST_SUITE</p>
        <p><strong>Execution Time:</strong> $execution_time</p>
        <p><strong>Duration:</strong> ${total_duration} seconds</p>
    </div>
    
    <div class="summary">
        <h2>Test Summary</h2>
        <p><strong>Total Test Files:</strong> $total_tests</p>
        <p><strong>Passed:</strong> <span class="success">$passed_tests</span></p>
        <p><strong>Failed:</strong> <span class="failure">$failed_tests</span></p>
    </div>
    
    <div class="test-files">
        <h2>Test Files</h2>
EOF

    if compgen -G "$OUTPUT_DIR/*.json" > /dev/null; then
        for json_file in "$OUTPUT_DIR"/*.json; do
            local filename=$(basename "$json_file" .json)
            local status="UNKNOWN"
            local status_class="failure"
            
            if grep -q '"State":"passed"' "$json_file" 2>/dev/null; then
                status="PASSED"
                status_class="success"
            elif grep -q '"State":"failed"' "$json_file" 2>/dev/null; then
                status="FAILED"
                status_class="failure"
            fi
            
            echo "        <div class=\"test-file\">" >> "$report_path"
            echo "            <strong>$filename</strong> - <span class=\"$status_class\">$status</span>" >> "$report_path"
            echo "        </div>" >> "$report_path"
        done
    fi
    
    cat >> "$report_path" << EOF
    </div>
</body>
</html>
EOF
    
    log "SUCCESS" "✓ Test report generated: $report_path"
}

cleanup_resources() {
    log "INFO" "Performing cleanup..."
    
    # Clean up test resources
    kubectl delete networkintents --all -n "$NAMESPACE" --ignore-not-found=true 2>/dev/null || true
    
    # Clean up test namespaces
    local test_namespaces
    test_namespaces=$(kubectl get namespaces -o name 2>/dev/null | grep "test-" || true)
    
    for ns in $test_namespaces; do
        local ns_name=${ns#namespace/}
        log "INFO" "Cleaning up test namespace: $ns_name"
        kubectl delete namespace "$ns_name" --ignore-not-found=true 2>/dev/null || true
    done
    
    log "SUCCESS" "✓ Cleanup completed"
}

show_summary() {
    local success="$1"
    local duration=$(( $(date +%s) - START_TIME ))
    local duration_minutes=$(( duration / 60 ))
    
    echo -e "${CYAN}"
    cat << EOF
╔═══════════════════════════════════════════════════════╗
║                    TEST SUMMARY                      ║
╠═══════════════════════════════════════════════════════╣
EOF
    printf "║ Suite: %-10s                                ║\n" "${TEST_SUITE^^}"
    printf "║ Duration: %4s minutes                          ║\n" "$duration_minutes"
    if [[ "$success" == "true" ]]; then
        printf "║ Status: ${GREEN}%-6s${CYAN}                                ║\n" "PASSED"
    else
        printf "║ Status: ${RED}%-6s${CYAN}                                ║\n" "FAILED"
    fi
    echo "╚═══════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Trap for cleanup on exit
trap 'cleanup_resources' EXIT

# Main execution
main() {
    parse_arguments "$@"
    
    show_banner
    
    if [[ "$CLEANUP_ONLY" == "true" ]]; then
        cleanup_resources
        exit 0
    fi
    
    log "INFO" "Starting E2E test execution..."
    
    # Prerequisites and setup
    check_prerequisites
    test_kubernetes_connection
    test_service_dependencies
    initialize_test_environment
    
    # Execute tests
    local test_success=true
    if ! run_test_suite "$TEST_SUITE"; then
        test_success=false
    fi
    
    # Generate reports
    generate_test_report
    
    # Show summary
    show_summary "$test_success"
    
    # Cleanup
    if [[ "$TEST_SUITE" == "cleanup" || "$TEST_SUITE" == "all" ]]; then
        cleanup_resources
    fi
    
    # Exit with appropriate code
    if [[ "$test_success" == "true" ]]; then
        log "SUCCESS" "✓ All tests completed successfully"
        exit 0
    else
        log "ERROR" "✗ Some tests failed"
        exit 1
    fi
}

# Execute main function with all arguments
main "$@"