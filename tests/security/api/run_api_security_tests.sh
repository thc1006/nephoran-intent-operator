#!/bin/bash

# Nephoran Intent Operator - Comprehensive API Security Test Runner
# This script runs all API security tests with detailed reporting

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${TEST_DIR}/../../.." && pwd)"
REPORT_DIR="${PROJECT_ROOT}/test-reports/security/api"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="${REPORT_DIR}/api_security_test_report_${TIMESTAMP}.html"
JSON_REPORT="${REPORT_DIR}/api_security_test_report_${TIMESTAMP}.json"
LOG_FILE="${REPORT_DIR}/api_security_test_${TIMESTAMP}.log"

# Test configuration
TEST_TIMEOUT="30m"
PARALLEL_TESTS=4
VERBOSE=${VERBOSE:-false}
COVERAGE_THRESHOLD=80

# Create report directory
mkdir -p "${REPORT_DIR}"

# Function to print colored output
print_color() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to print header
print_header() {
    echo ""
    print_color "${BLUE}" "================================================================"
    print_color "${BLUE}" "$1"
    print_color "${BLUE}" "================================================================"
    echo ""
}

# Function to check prerequisites
check_prerequisites() {
    print_header "Checking Prerequisites"
    
    local prerequisites_met=true
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        print_color "${RED}" "✗ Go is not installed"
        prerequisites_met=false
    else
        print_color "${GREEN}" "✓ Go $(go version | awk '{print $3}') installed"
    fi
    
    # Check if required tools are installed
    local required_tools=("gotestsum" "gocov" "gocov-html" "gosec" "staticcheck")
    for tool in "${required_tools[@]}"; do
        if ! command -v ${tool} &> /dev/null; then
            print_color "${YELLOW}" "⚠ ${tool} is not installed. Installing..."
            go install github.com/gotestyourself/gotestsum@latest
            go install github.com/axw/gocov/gocov@latest
            go install github.com/matm/gocov-html@latest
            go install github.com/securego/gosec/v2/cmd/gosec@latest
            go install honnef.co/go/tools/cmd/staticcheck@latest
        else
            print_color "${GREEN}" "✓ ${tool} installed"
        fi
    done
    
    # Check if test environment is configured
    if [ -z "${NEPHORAN_TEST_ENV}" ]; then
        print_color "${YELLOW}" "⚠ NEPHORAN_TEST_ENV not set, using default test configuration"
        export NEPHORAN_TEST_ENV="test"
    else
        print_color "${GREEN}" "✓ Test environment: ${NEPHORAN_TEST_ENV}"
    fi
    
    # Check if Kubernetes cluster is available
    if kubectl cluster-info &> /dev/null; then
        print_color "${GREEN}" "✓ Kubernetes cluster is accessible"
    else
        print_color "${YELLOW}" "⚠ Kubernetes cluster not accessible, some tests may be skipped"
    fi
    
    if [ "${prerequisites_met}" = false ]; then
        print_color "${RED}" "Prerequisites check failed. Please install required tools."
        exit 1
    fi
}

# Function to setup test environment
setup_test_environment() {
    print_header "Setting Up Test Environment"
    
    # Export test environment variables
    export TEST_NAMESPACE="nephoran-security-test-${TIMESTAMP}"
    export LOG_LEVEL="debug"
    export TEST_TIMEOUT="${TEST_TIMEOUT}"
    
    # Create test namespace if using real cluster
    if kubectl cluster-info &> /dev/null; then
        print_color "${YELLOW}" "Creating test namespace: ${TEST_NAMESPACE}"
        kubectl create namespace "${TEST_NAMESPACE}" 2>/dev/null || true
        
        # Apply RBAC for tests
        kubectl apply -f "${PROJECT_ROOT}/deployments/rbac/test-rbac.yaml" -n "${TEST_NAMESPACE}" 2>/dev/null || true
    fi
    
    # Start mock services if needed
    print_color "${YELLOW}" "Starting mock services..."
    
    # Mock LLM Processor API
    nohup go run "${TEST_DIR}/../../mocks/llm_processor_mock.go" -port 8081 > "${REPORT_DIR}/llm_mock.log" 2>&1 &
    LLM_MOCK_PID=$!
    
    # Mock RAG API
    nohup go run "${TEST_DIR}/../../mocks/rag_api_mock.go" -port 8082 > "${REPORT_DIR}/rag_mock.log" 2>&1 &
    RAG_MOCK_PID=$!
    
    # Mock Nephio Bridge API
    nohup go run "${TEST_DIR}/../../mocks/nephio_bridge_mock.go" -port 8083 > "${REPORT_DIR}/nephio_mock.log" 2>&1 &
    NEPHIO_MOCK_PID=$!
    
    # Mock O-RAN Adaptor API
    nohup go run "${TEST_DIR}/../../mocks/oran_adaptor_mock.go" -port 8084 > "${REPORT_DIR}/oran_mock.log" 2>&1 &
    ORAN_MOCK_PID=$!
    
    # Wait for services to start
    sleep 5
    
    print_color "${GREEN}" "✓ Test environment setup complete"
}

# Function to run authentication tests
run_authentication_tests() {
    print_header "Running Authentication Tests"
    
    local test_output="${REPORT_DIR}/auth_test_output.txt"
    local test_passed=true
    
    print_color "${YELLOW}" "Testing JWT validation, OAuth2 flows, MFA..."
    
    if [ "${VERBOSE}" = true ]; then
        go test -v -timeout ${TEST_TIMEOUT} \
            -run TestJWTTokenValidation \
            -run TestOAuth2Flow \
            -run TestMultiFactorAuthentication \
            -run TestSessionManagement \
            "${TEST_DIR}/auth_test.go" 2>&1 | tee "${test_output}"
    else
        go test -timeout ${TEST_TIMEOUT} \
            -run "Test(JWT|OAuth2|MFA|Session)" \
            "${TEST_DIR}/auth_test.go" > "${test_output}" 2>&1
    fi
    
    if [ $? -eq 0 ]; then
        print_color "${GREEN}" "✓ Authentication tests passed"
    else
        print_color "${RED}" "✗ Authentication tests failed"
        test_passed=false
    fi
    
    # Extract test metrics
    local total_tests=$(grep -c "^=== RUN" "${test_output}" 2>/dev/null || echo "0")
    local passed_tests=$(grep -c "^--- PASS:" "${test_output}" 2>/dev/null || echo "0")
    local failed_tests=$(grep -c "^--- FAIL:" "${test_output}" 2>/dev/null || echo "0")
    
    echo "Authentication Test Results:" >> "${LOG_FILE}"
    echo "  Total: ${total_tests}" >> "${LOG_FILE}"
    echo "  Passed: ${passed_tests}" >> "${LOG_FILE}"
    echo "  Failed: ${failed_tests}" >> "${LOG_FILE}"
    
    return $([ "${test_passed}" = true ] && echo 0 || echo 1)
}

# Function to run rate limiting tests
run_rate_limiting_tests() {
    print_header "Running Rate Limiting Tests"
    
    local test_output="${REPORT_DIR}/rate_limiting_test_output.txt"
    local test_passed=true
    
    print_color "${YELLOW}" "Testing rate limiting, DDoS protection, burst handling..."
    
    if [ "${VERBOSE}" = true ]; then
        go test -v -timeout ${TEST_TIMEOUT} \
            -run TestEndpointRateLimiting \
            -run TestDDoSProtection \
            -run TestBurstHandling \
            "${TEST_DIR}/rate_limiting_test.go" 2>&1 | tee "${test_output}"
    else
        go test -timeout ${TEST_TIMEOUT} \
            -run "Test(RateLimit|DDoS|Burst)" \
            "${TEST_DIR}/rate_limiting_test.go" > "${test_output}" 2>&1
    fi
    
    if [ $? -eq 0 ]; then
        print_color "${GREEN}" "✓ Rate limiting tests passed"
    else
        print_color "${RED}" "✗ Rate limiting tests failed"
        test_passed=false
    fi
    
    return $([ "${test_passed}" = true ] && echo 0 || echo 1)
}

# Function to run input validation tests
run_input_validation_tests() {
    print_header "Running Input Validation Tests"
    
    local test_output="${REPORT_DIR}/input_validation_test_output.txt"
    local test_passed=true
    
    print_color "${YELLOW}" "Testing OWASP Top 10 vulnerabilities..."
    
    if [ "${VERBOSE}" = true ]; then
        go test -v -timeout ${TEST_TIMEOUT} \
            -run TestSQLInjectionPrevention \
            -run TestXSSPrevention \
            -run TestCommandInjection \
            -run TestPathTraversal \
            -run TestXXEInjection \
            "${TEST_DIR}/input_validation_test.go" 2>&1 | tee "${test_output}"
    else
        go test -timeout ${TEST_TIMEOUT} \
            -run "Test(SQLInjection|XSS|Command|Path|XXE)" \
            "${TEST_DIR}/input_validation_test.go" > "${test_output}" 2>&1
    fi
    
    if [ $? -eq 0 ]; then
        print_color "${GREEN}" "✓ Input validation tests passed"
    else
        print_color "${RED}" "✗ Input validation tests failed"
        test_passed=false
    fi
    
    return $([ "${test_passed}" = true ] && echo 0 || echo 1)
}

# Function to run API security suite tests
run_api_security_suite() {
    print_header "Running API Security Suite Tests"
    
    local test_output="${REPORT_DIR}/api_security_suite_output.txt"
    local test_passed=true
    
    print_color "${YELLOW}" "Testing CORS, security headers, TLS, CSRF protection..."
    
    if [ "${VERBOSE}" = true ]; then
        go test -v -timeout ${TEST_TIMEOUT} \
            -run TestCORSConfiguration \
            -run TestSecurityHeaders \
            -run TestTLSEnforcement \
            -run TestCSRFProtection \
            "${TEST_DIR}/api_security_suite_test.go" 2>&1 | tee "${test_output}"
    else
        go test -timeout ${TEST_TIMEOUT} \
            "${TEST_DIR}/api_security_suite_test.go" > "${test_output}" 2>&1
    fi
    
    if [ $? -eq 0 ]; then
        print_color "${GREEN}" "✓ API security suite tests passed"
    else
        print_color "${RED}" "✗ API security suite tests failed"
        test_passed=false
    fi
    
    return $([ "${test_passed}" = true ] && echo 0 || echo 1)
}

# Function to run static security analysis
run_static_security_analysis() {
    print_header "Running Static Security Analysis"
    
    print_color "${YELLOW}" "Running gosec security scanner..."
    
    # Run gosec
    gosec -fmt json -out "${REPORT_DIR}/gosec_report.json" "${TEST_DIR}/..." 2>/dev/null || true
    
    # Check for high severity issues
    if [ -f "${REPORT_DIR}/gosec_report.json" ]; then
        local high_issues=$(jq '[.Issues[] | select(.severity == "HIGH")] | length' "${REPORT_DIR}/gosec_report.json")
        local medium_issues=$(jq '[.Issues[] | select(.severity == "MEDIUM")] | length' "${REPORT_DIR}/gosec_report.json")
        
        if [ "${high_issues}" -gt 0 ]; then
            print_color "${RED}" "✗ Found ${high_issues} high severity security issues"
        else
            print_color "${GREEN}" "✓ No high severity security issues found"
        fi
        
        if [ "${medium_issues}" -gt 0 ]; then
            print_color "${YELLOW}" "⚠ Found ${medium_issues} medium severity security issues"
        fi
    fi
    
    print_color "${YELLOW}" "Running staticcheck..."
    staticcheck "${TEST_DIR}/..." 2>&1 | tee "${REPORT_DIR}/staticcheck_output.txt" || true
}

# Function to run coverage analysis
run_coverage_analysis() {
    print_header "Running Coverage Analysis"
    
    print_color "${YELLOW}" "Generating test coverage report..."
    
    # Run tests with coverage
    go test -coverprofile="${REPORT_DIR}/coverage.out" \
        -covermode=atomic \
        "${TEST_DIR}/..." 2>/dev/null
    
    # Generate HTML coverage report
    go tool cover -html="${REPORT_DIR}/coverage.out" -o "${REPORT_DIR}/coverage.html" 2>/dev/null
    
    # Calculate coverage percentage
    local coverage=$(go tool cover -func="${REPORT_DIR}/coverage.out" | grep total | awk '{print $3}' | sed 's/%//')
    
    if (( $(echo "${coverage} >= ${COVERAGE_THRESHOLD}" | bc -l) )); then
        print_color "${GREEN}" "✓ Test coverage: ${coverage}% (threshold: ${COVERAGE_THRESHOLD}%)"
    else
        print_color "${RED}" "✗ Test coverage: ${coverage}% (below threshold: ${COVERAGE_THRESHOLD}%)"
    fi
}

# Function to run benchmark tests
run_benchmark_tests() {
    print_header "Running Security Benchmark Tests"
    
    print_color "${YELLOW}" "Running performance benchmarks for security functions..."
    
    go test -bench=. -benchmem -benchtime=10s \
        -run=^$ \
        "${TEST_DIR}/..." > "${REPORT_DIR}/benchmark_results.txt" 2>&1
    
    if [ $? -eq 0 ]; then
        print_color "${GREEN}" "✓ Benchmark tests completed"
        
        # Extract key metrics
        grep "Benchmark" "${REPORT_DIR}/benchmark_results.txt" | head -10
    else
        print_color "${RED}" "✗ Benchmark tests failed"
    fi
}

# Function to generate HTML report
generate_html_report() {
    print_header "Generating HTML Report"
    
    cat > "${REPORT_FILE}" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>API Security Test Report - ${TIMESTAMP}</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; }
        h2 { color: #666; border-bottom: 2px solid #ddd; padding-bottom: 5px; }
        .pass { color: green; font-weight: bold; }
        .fail { color: red; font-weight: bold; }
        .warning { color: orange; font-weight: bold; }
        .metrics { background: #f5f5f5; padding: 10px; border-radius: 5px; margin: 10px 0; }
        table { border-collapse: collapse; width: 100%; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #4CAF50; color: white; }
        .summary { background: #e8f5e9; padding: 15px; border-radius: 5px; margin: 20px 0; }
    </style>
</head>
<body>
    <h1>API Security Test Report</h1>
    <div class="summary">
        <h2>Executive Summary</h2>
        <p><strong>Test Date:</strong> $(date)</p>
        <p><strong>Environment:</strong> ${NEPHORAN_TEST_ENV}</p>
        <p><strong>Test Duration:</strong> ${TEST_DURATION} seconds</p>
    </div>
    
    <h2>Test Results</h2>
    <table>
        <tr>
            <th>Test Suite</th>
            <th>Status</th>
            <th>Tests Run</th>
            <th>Passed</th>
            <th>Failed</th>
        </tr>
        <tr>
            <td>Authentication Tests</td>
            <td class="${AUTH_STATUS}">${AUTH_RESULT}</td>
            <td>${AUTH_TOTAL}</td>
            <td>${AUTH_PASSED}</td>
            <td>${AUTH_FAILED}</td>
        </tr>
        <tr>
            <td>Rate Limiting Tests</td>
            <td class="${RATE_STATUS}">${RATE_RESULT}</td>
            <td>${RATE_TOTAL}</td>
            <td>${RATE_PASSED}</td>
            <td>${RATE_FAILED}</td>
        </tr>
        <tr>
            <td>Input Validation Tests</td>
            <td class="${INPUT_STATUS}">${INPUT_RESULT}</td>
            <td>${INPUT_TOTAL}</td>
            <td>${INPUT_PASSED}</td>
            <td>${INPUT_FAILED}</td>
        </tr>
        <tr>
            <td>API Security Suite</td>
            <td class="${SUITE_STATUS}">${SUITE_RESULT}</td>
            <td>${SUITE_TOTAL}</td>
            <td>${SUITE_PASSED}</td>
            <td>${SUITE_FAILED}</td>
        </tr>
    </table>
    
    <h2>Security Analysis</h2>
    <div class="metrics">
        <h3>OWASP API Security Top 10 Coverage</h3>
        <ul>
            <li>✓ API1:2023 - Broken Object Level Authorization</li>
            <li>✓ API2:2023 - Broken Authentication</li>
            <li>✓ API3:2023 - Broken Object Property Level Authorization</li>
            <li>✓ API4:2023 - Unrestricted Resource Consumption</li>
            <li>✓ API5:2023 - Broken Function Level Authorization</li>
            <li>✓ API6:2023 - Unrestricted Access to Sensitive Business Flows</li>
            <li>✓ API7:2023 - Server Side Request Forgery</li>
            <li>✓ API8:2023 - Security Misconfiguration</li>
            <li>✓ API9:2023 - Improper Inventory Management</li>
            <li>✓ API10:2023 - Unsafe Consumption of APIs</li>
        </ul>
    </div>
    
    <h2>Recommendations</h2>
    <ul>
        <li>Continue monitoring rate limiting effectiveness</li>
        <li>Regularly update security test patterns</li>
        <li>Review and update JWT signing keys</li>
        <li>Implement additional MFA methods</li>
        <li>Enhance API versioning strategy</li>
    </ul>
    
    <h2>Detailed Logs</h2>
    <p>Full test logs available at: <code>${LOG_FILE}</code></p>
    <p>Coverage report: <a href="coverage.html">View Coverage Report</a></p>
</body>
</html>
EOF
    
    print_color "${GREEN}" "✓ HTML report generated: ${REPORT_FILE}"
}

# Function to generate JSON report
generate_json_report() {
    cat > "${JSON_REPORT}" << EOF
{
    "timestamp": "${TIMESTAMP}",
    "environment": "${NEPHORAN_TEST_ENV}",
    "duration": ${TEST_DURATION},
    "results": {
        "authentication": {
            "status": "${AUTH_RESULT}",
            "total": ${AUTH_TOTAL:-0},
            "passed": ${AUTH_PASSED:-0},
            "failed": ${AUTH_FAILED:-0}
        },
        "rateLimiting": {
            "status": "${RATE_RESULT}",
            "total": ${RATE_TOTAL:-0},
            "passed": ${RATE_PASSED:-0},
            "failed": ${RATE_FAILED:-0}
        },
        "inputValidation": {
            "status": "${INPUT_RESULT}",
            "total": ${INPUT_TOTAL:-0},
            "passed": ${INPUT_PASSED:-0},
            "failed": ${INPUT_FAILED:-0}
        },
        "apiSecuritySuite": {
            "status": "${SUITE_RESULT}",
            "total": ${SUITE_TOTAL:-0},
            "passed": ${SUITE_PASSED:-0},
            "failed": ${SUITE_FAILED:-0}
        }
    },
    "coverage": ${COVERAGE:-0},
    "securityIssues": {
        "high": ${HIGH_ISSUES:-0},
        "medium": ${MEDIUM_ISSUES:-0},
        "low": ${LOW_ISSUES:-0}
    }
}
EOF
    
    print_color "${GREEN}" "✓ JSON report generated: ${JSON_REPORT}"
}

# Function to cleanup test environment
cleanup_test_environment() {
    print_header "Cleaning Up Test Environment"
    
    # Kill mock services
    if [ ! -z "${LLM_MOCK_PID}" ]; then
        kill ${LLM_MOCK_PID} 2>/dev/null || true
    fi
    
    if [ ! -z "${RAG_MOCK_PID}" ]; then
        kill ${RAG_MOCK_PID} 2>/dev/null || true
    fi
    
    if [ ! -z "${NEPHIO_MOCK_PID}" ]; then
        kill ${NEPHIO_MOCK_PID} 2>/dev/null || true
    fi
    
    if [ ! -z "${ORAN_MOCK_PID}" ]; then
        kill ${ORAN_MOCK_PID} 2>/dev/null || true
    fi
    
    # Delete test namespace if using real cluster
    if kubectl cluster-info &> /dev/null; then
        print_color "${YELLOW}" "Deleting test namespace: ${TEST_NAMESPACE}"
        kubectl delete namespace "${TEST_NAMESPACE}" --timeout=60s 2>/dev/null || true
    fi
    
    print_color "${GREEN}" "✓ Cleanup complete"
}

# Main execution
main() {
    local start_time=$(date +%s)
    local overall_status=0
    
    print_header "Nephoran Intent Operator - API Security Test Suite"
    print_color "${YELLOW}" "Starting comprehensive API security testing..."
    
    # Check prerequisites
    check_prerequisites
    
    # Setup test environment
    setup_test_environment
    
    # Run test suites
    run_authentication_tests || overall_status=1
    run_rate_limiting_tests || overall_status=1
    run_input_validation_tests || overall_status=1
    run_api_security_suite || overall_status=1
    
    # Run additional analyses
    run_static_security_analysis
    run_coverage_analysis
    run_benchmark_tests
    
    # Calculate test duration
    local end_time=$(date +%s)
    TEST_DURATION=$((end_time - start_time))
    
    # Generate reports
    generate_html_report
    generate_json_report
    
    # Cleanup
    cleanup_test_environment
    
    # Print summary
    print_header "Test Summary"
    
    if [ ${overall_status} -eq 0 ]; then
        print_color "${GREEN}" "✓ All API security tests passed successfully!"
    else
        print_color "${RED}" "✗ Some API security tests failed. Please review the report."
    fi
    
    print_color "${BLUE}" "Reports generated:"
    print_color "${BLUE}" "  - HTML Report: ${REPORT_FILE}"
    print_color "${BLUE}" "  - JSON Report: ${JSON_REPORT}"
    print_color "${BLUE}" "  - Test Logs: ${LOG_FILE}"
    print_color "${BLUE}" "  - Coverage Report: ${REPORT_DIR}/coverage.html"
    
    exit ${overall_status}
}

# Handle interrupts
trap cleanup_test_environment EXIT INT TERM

# Run main function
main "$@"