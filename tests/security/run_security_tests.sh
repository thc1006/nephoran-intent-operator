#!/bin/bash

# Security Test Runner Script for Nephoran Intent Operator
# This script executes comprehensive security tests and generates reports

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
TEST_RESULTS_DIR="${PROJECT_ROOT}/test-results/security"
COVERAGE_DIR="${TEST_RESULTS_DIR}/coverage"
REPORTS_DIR="${TEST_RESULTS_DIR}/reports"
LOG_FILE="${TEST_RESULTS_DIR}/security-tests.log"

# Test configuration
NAMESPACE="${TEST_NAMESPACE:-nephoran-intent-operator-test}"
KUBECONFIG="${KUBECONFIG:-${HOME}/.kube/config}"
PARALLEL_TESTS="${PARALLEL_TESTS:-4}"
TIMEOUT="${TEST_TIMEOUT:-30m}"
VERBOSE="${VERBOSE:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $*" | tee -a "$LOG_FILE"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" | tee -a "$LOG_FILE"
}

# Print usage information
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Run comprehensive security tests for the Nephoran Intent Operator.

OPTIONS:
    -h, --help              Show this help message
    -n, --namespace NAME    Kubernetes namespace to test (default: ${NAMESPACE})
    -t, --timeout DURATION Test timeout duration (default: ${TIMEOUT})
    -p, --parallel COUNT    Number of parallel tests (default: ${PARALLEL_TESTS})
    -v, --verbose           Enable verbose output
    --coverage              Generate code coverage report
    --report FORMAT         Generate report in format: json, xml, html (default: all)
    --skip-setup            Skip test environment setup
    --skip-cleanup          Skip test environment cleanup
    --security-only         Run only security-specific tests
    --container-only        Run only container security tests
    --rbac-only            Run only RBAC tests
    --network-only         Run only network policy tests
    --secrets-only         Run only secrets management tests
    --tls-only             Run only TLS/mTLS tests

EXAMPLES:
    $0                      # Run all security tests
    $0 -v --coverage        # Run with verbose output and coverage
    $0 --container-only     # Run only container security tests
    $0 --report json        # Generate only JSON report
    
ENVIRONMENT VARIABLES:
    TEST_NAMESPACE         Kubernetes namespace for tests
    KUBECONFIG            Path to kubeconfig file
    PARALLEL_TESTS        Number of parallel test processes
    TEST_TIMEOUT          Overall test timeout
    VERBOSE               Enable verbose output (true/false)
EOF
}

# Parse command line arguments
parse_args() {
    local skip_setup=false
    local skip_cleanup=false
    local generate_coverage=false
    local report_formats=()
    local test_types=()

    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                exit 0
                ;;
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            -p|--parallel)
                PARALLEL_TESTS="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            --coverage)
                generate_coverage=true
                shift
                ;;
            --report)
                report_formats+=("$2")
                shift 2
                ;;
            --skip-setup)
                skip_setup=true
                shift
                ;;
            --skip-cleanup)
                skip_cleanup=true
                shift
                ;;
            --security-only)
                test_types+=("security")
                shift
                ;;
            --container-only)
                test_types+=("container")
                shift
                ;;
            --rbac-only)
                test_types+=("rbac")
                shift
                ;;
            --network-only)
                test_types+=("network")
                shift
                ;;
            --secrets-only)
                test_types+=("secrets")
                shift
                ;;
            --tls-only)
                test_types+=("tls")
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done

    # Set defaults
    if [[ ${#report_formats[@]} -eq 0 ]]; then
        report_formats=("json" "xml" "html")
    fi

    if [[ ${#test_types[@]} -eq 0 ]]; then
        test_types=("all")
    fi

    # Export variables for use in other functions
    export SKIP_SETUP="$skip_setup"
    export SKIP_CLEANUP="$skip_cleanup"
    export GENERATE_COVERAGE="$generate_coverage"
    export REPORT_FORMATS="${report_formats[*]}"
    export TEST_TYPES="${test_types[*]}"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check Go
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi

    local go_version
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go version: $go_version"

    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed"
        exit 1
    fi

    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi

    log_info "Kubernetes cluster: $(kubectl config current-context)"

    # Check if namespace exists
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_info "Using existing namespace: $NAMESPACE"
    else
        log_warn "Namespace $NAMESPACE does not exist, will create it"
    fi

    # Check Ginkgo
    if ! command -v ginkgo &> /dev/null; then
        log_info "Installing Ginkgo..."
        go install github.com/onsi/ginkgo/v2/ginkgo@latest
    fi

    # Check required directories
    if [[ ! -d "${PROJECT_ROOT}/tests/security" ]]; then
        log_error "Security tests directory not found"
        exit 1
    fi
}

# Setup test environment
setup_test_environment() {
    if [[ "$SKIP_SETUP" == "true" ]]; then
        log_info "Skipping test environment setup"
        return
    fi

    log_info "Setting up test environment..."

    # Create directories
    mkdir -p "$TEST_RESULTS_DIR" "$COVERAGE_DIR" "$REPORTS_DIR"

    # Create namespace if it doesn't exist
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        kubectl create namespace "$NAMESPACE"
        log_info "Created namespace: $NAMESPACE"
    fi

    # Label namespace for testing
    kubectl label namespace "$NAMESPACE" \
        security.nephoran.io/test=true \
        pod-security.kubernetes.io/enforce=restricted \
        pod-security.kubernetes.io/audit=restricted \
        pod-security.kubernetes.io/warn=restricted \
        --overwrite

    # Apply test resources if they exist
    if [[ -d "${PROJECT_ROOT}/tests/security/testdata" ]]; then
        log_info "Applying test resources..."
        kubectl apply -f "${PROJECT_ROOT}/tests/security/testdata" -n "$NAMESPACE" || true
    fi

    # Wait for resources to be ready
    sleep 5
}

# Build test arguments
build_test_args() {
    local test_args=()
    local test_packages=()

    # Set verbosity
    if [[ "$VERBOSE" == "true" ]]; then
        test_args+=("-v" "--progress")
    fi

    # Set parallelism
    test_args+=("--procs=$PARALLEL_TESTS")

    # Set timeout
    test_args+=("--timeout=$TIMEOUT")

    # Set output format
    test_args+=("--junit-report=${REPORTS_DIR}/junit.xml")
    
    # Coverage
    if [[ "$GENERATE_COVERAGE" == "true" ]]; then
        test_args+=("--cover" "--coverprofile=${COVERAGE_DIR}/security.coverprofile")
    fi

    # Determine which tests to run
    local run_all=false
    for test_type in $TEST_TYPES; do
        case "$test_type" in
            all|security)
                run_all=true
                ;;
            container)
                test_packages+=("./tests/security/container_security_test.go")
                ;;
            rbac)
                test_packages+=("./tests/security/rbac_test.go")
                ;;
            network)
                test_packages+=("./tests/security/network_policy_test.go")
                ;;
            secrets)
                test_packages+=("./tests/security/secrets_test.go")
                ;;
            tls)
                test_packages+=("./tests/security/tls_test.go")
                ;;
        esac
    done

    if [[ "$run_all" == "true" ]]; then
        test_packages=("./tests/security")
    fi

    echo "${test_args[*]} ${test_packages[*]}"
}

# Run security tests
run_security_tests() {
    log_info "Running security tests..."

    cd "$PROJECT_ROOT"

    # Set environment variables
    export TEST_NAMESPACE="$NAMESPACE"
    export KUBECONFIG="$KUBECONFIG"

    # Build test arguments
    local test_args
    test_args=$(build_test_args)

    log_info "Test command: ginkgo $test_args"

    # Run tests
    local start_time
    start_time=$(date +%s)

    if ginkgo $test_args; then
        local end_time
        end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        log_success "Security tests completed successfully in ${duration} seconds"
        return 0
    else
        local end_time
        end_time=$(date +%s)
        local duration=$((end_time - start_time))
        
        log_error "Security tests failed after ${duration} seconds"
        return 1
    fi
}

# Generate coverage report
generate_coverage_report() {
    if [[ "$GENERATE_COVERAGE" != "true" ]]; then
        return
    fi

    log_info "Generating coverage report..."

    cd "$PROJECT_ROOT"

    # Generate HTML coverage report
    if [[ -f "${COVERAGE_DIR}/security.coverprofile" ]]; then
        go tool cover -html="${COVERAGE_DIR}/security.coverprofile" -o "${REPORTS_DIR}/coverage.html"
        
        # Generate coverage summary
        local coverage_percent
        coverage_percent=$(go tool cover -func="${COVERAGE_DIR}/security.coverprofile" | tail -1 | awk '{print $3}')
        
        log_info "Security test coverage: $coverage_percent"
        echo "Security Test Coverage: $coverage_percent" > "${REPORTS_DIR}/coverage.txt"
    fi
}

# Generate test reports
generate_reports() {
    log_info "Generating test reports..."

    for format in $REPORT_FORMATS; do
        case "$format" in
            json)
                generate_json_report
                ;;
            xml)
                # JUnit XML is already generated by Ginkgo
                log_info "JUnit XML report: ${REPORTS_DIR}/junit.xml"
                ;;
            html)
                generate_html_report
                ;;
            *)
                log_warn "Unknown report format: $format"
                ;;
        esac
    done
}

# Generate JSON report
generate_json_report() {
    local json_report="${REPORTS_DIR}/security-test-results.json"
    
    cat > "$json_report" << EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "namespace": "$NAMESPACE",
    "test_types": [$(printf '"%s",' $TEST_TYPES | sed 's/,$//')],
    "reports": {
        "junit": "${REPORTS_DIR}/junit.xml",
        "coverage_html": "${REPORTS_DIR}/coverage.html",
        "coverage_text": "${REPORTS_DIR}/coverage.txt"
    }
}
EOF

    log_info "JSON report generated: $json_report"
}

# Generate HTML report
generate_html_report() {
    local html_report="${REPORTS_DIR}/security-test-report.html"
    
    cat > "$html_report" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Nephoran Security Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .header { background-color: #f8f9fa; padding: 20px; border-radius: 5px; }
        .section { margin: 20px 0; }
        .success { color: #28a745; }
        .warning { color: #ffc107; }
        .error { color: #dc3545; }
        .test-summary { background-color: #e9ecef; padding: 15px; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Nephoran Intent Operator Security Test Report</h1>
        <p><strong>Generated:</strong> TIMESTAMP</p>
        <p><strong>Namespace:</strong> NAMESPACE</p>
        <p><strong>Test Types:</strong> TEST_TYPES</p>
    </div>
    
    <div class="section">
        <h2>Test Summary</h2>
        <div class="test-summary">
            <p>Security tests validate the following areas:</p>
            <ul>
                <li><strong>Container Security:</strong> Non-root users, read-only filesystems, capability management</li>
                <li><strong>RBAC:</strong> Least privilege access, service account permissions, role bindings</li>
                <li><strong>Network Policies:</strong> Zero-trust networking, component isolation, egress controls</li>
                <li><strong>Secrets Management:</strong> Encryption at rest, rotation, access controls</li>
                <li><strong>TLS/mTLS:</strong> Certificate validation, mutual authentication, secure protocols</li>
            </ul>
        </div>
    </div>
    
    <div class="section">
        <h2>Reports</h2>
        <ul>
            <li><a href="junit.xml">JUnit XML Results</a></li>
            <li><a href="coverage.html">Coverage Report</a></li>
            <li><a href="../logs/security-tests.log">Test Logs</a></li>
        </ul>
    </div>
</body>
</html>
EOF

    # Replace placeholders
    sed -i "s/TIMESTAMP/$(date -u +%Y-%m-%dT%H:%M:%SZ)/g" "$html_report"
    sed -i "s/NAMESPACE/$NAMESPACE/g" "$html_report"
    sed -i "s/TEST_TYPES/$TEST_TYPES/g" "$html_report"

    log_info "HTML report generated: $html_report"
}

# Cleanup test environment
cleanup_test_environment() {
    if [[ "$SKIP_CLEANUP" == "true" ]]; then
        log_info "Skipping test environment cleanup"
        return
    fi

    log_info "Cleaning up test environment..."

    # Remove test resources
    if [[ -d "${PROJECT_ROOT}/tests/security/testdata" ]]; then
        kubectl delete -f "${PROJECT_ROOT}/tests/security/testdata" -n "$NAMESPACE" --ignore-not-found=true
    fi

    # Clean up namespace if it was created by this script
    if kubectl get namespace "$NAMESPACE" -o jsonpath='{.metadata.labels.security\.nephoran\.io/test}' | grep -q "true"; then
        kubectl delete namespace "$NAMESPACE" --ignore-not-found=true
        log_info "Cleaned up test namespace: $NAMESPACE"
    fi
}

# Security compliance check
run_compliance_checks() {
    log_info "Running security compliance checks..."

    local compliance_script="${SCRIPT_DIR}/compliance_check.sh"
    if [[ -f "$compliance_script" ]]; then
        bash "$compliance_script" -n "$NAMESPACE" > "${REPORTS_DIR}/compliance-report.txt"
        log_info "Compliance report generated: ${REPORTS_DIR}/compliance-report.txt"
    else
        log_warn "Compliance check script not found: $compliance_script"
    fi
}

# Vulnerability scan
run_vulnerability_scan() {
    log_info "Running vulnerability scan..."

    # Check for container image vulnerabilities
    local vuln_report="${REPORTS_DIR}/vulnerability-scan.json"
    
    # This would integrate with tools like Trivy, Clair, or similar
    if command -v trivy &> /dev/null; then
        log_info "Running Trivy vulnerability scan..."
        trivy kubernetes cluster --format json --output "$vuln_report" || log_warn "Trivy scan failed"
    else
        log_warn "Vulnerability scanner (trivy) not found"
        echo '{"message": "No vulnerability scanner available"}' > "$vuln_report"
    fi
}

# Main execution
main() {
    local start_time
    start_time=$(date +%s)
    
    log_info "Starting Nephoran Intent Operator Security Test Suite"
    log_info "Script: $0"
    log_info "Arguments: $*"

    # Parse arguments
    parse_args "$@"

    # Check prerequisites
    check_prerequisites

    # Setup test environment
    setup_test_environment

    local test_result=0

    # Run security tests
    if ! run_security_tests; then
        test_result=1
    fi

    # Generate coverage report
    generate_coverage_report

    # Run additional security checks
    run_compliance_checks
    run_vulnerability_scan

    # Generate reports
    generate_reports

    # Cleanup
    cleanup_test_environment

    local end_time
    end_time=$(date +%s)
    local total_duration=$((end_time - start_time))

    if [[ $test_result -eq 0 ]]; then
        log_success "All security tests completed successfully in ${total_duration} seconds"
        log_info "Reports available in: $REPORTS_DIR"
    else
        log_error "Security tests failed after ${total_duration} seconds"
        log_info "Check logs and reports in: $REPORTS_DIR"
    fi

    exit $test_result
}

# Handle signals
trap cleanup_test_environment EXIT

# Run main function
main "$@"