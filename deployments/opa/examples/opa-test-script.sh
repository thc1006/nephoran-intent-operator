#!/bin/bash

# OPA Policy Testing Script for Nephoran Intent Operator
# This script tests OPA policies using the OPA CLI and sample data

set -euo pipefail

# Configuration
OPA_BINARY=${OPA_BINARY:-"opa"}
POLICY_DIR="./policies"
TEST_DATA_DIR="./examples"
RESULTS_DIR="./test-results"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Create results directory
mkdir -p "${RESULTS_DIR}"

echo -e "${BLUE}================================${NC}"
echo -e "${BLUE}OPA Policy Testing for Nephoran${NC}"
echo -e "${BLUE}================================${NC}"
echo

# Function to test OPA decision
test_opa_decision() {
    local policy_file=$1
    local data_file=$2
    local query=$3
    local expected_result=$4
    local test_name=$5

    echo -e "Testing: ${YELLOW}${test_name}${NC}"
    echo "Policy: ${policy_file}"
    echo "Data: ${data_file}"
    echo "Query: ${query}"
    echo "Expected: ${expected_result}"

    # Run OPA evaluation
    local result
    result=$(${OPA_BINARY} eval -d "${policy_file}" -i "${data_file}" "${query}" --format pretty 2>/dev/null | jq -r '.[] // empty')
    
    # Check result
    if [[ "${result}" == "${expected_result}" ]]; then
        echo -e "${GREEN}✓ PASS${NC}"
        echo "PASS: ${test_name}" >> "${RESULTS_DIR}/test-results.log"
    else
        echo -e "${RED}✗ FAIL${NC}"
        echo "  Expected: ${expected_result}"
        echo "  Got: ${result}"
        echo "FAIL: ${test_name} - Expected: ${expected_result}, Got: ${result}" >> "${RESULTS_DIR}/test-results.log"
    fi
    echo
}

# Function to test policy denials
test_policy_denial() {
    local policy_file=$1
    local data_file=$2
    local test_name=$3

    echo -e "Testing denial: ${YELLOW}${test_name}${NC}"
    echo "Policy: ${policy_file}"
    echo "Data: ${data_file}"

    # Run OPA evaluation for denials
    local denials
    denials=$(${OPA_BINARY} eval -d "${policy_file}" -i "${data_file}" "data.nephoran.api.validation.deny" --format json 2>/dev/null | jq -r '.result[0].expressions[0].value[]? // empty')
    
    if [[ -n "${denials}" ]]; then
        echo -e "${GREEN}✓ PASS - Policy correctly denied request${NC}"
        echo "Denial reasons:"
        echo "${denials}" | while IFS= read -r line; do
            echo "  - ${line}"
        done
        echo "PASS: ${test_name} - Correctly denied" >> "${RESULTS_DIR}/test-results.log"
    else
        echo -e "${RED}✗ FAIL - Policy should have denied request${NC}"
        echo "FAIL: ${test_name} - Should have been denied" >> "${RESULTS_DIR}/test-results.log"
    fi
    echo
}

# Function to test policy allows
test_policy_allow() {
    local policy_file=$1
    local data_file=$2
    local test_name=$3

    echo -e "Testing allow: ${YELLOW}${test_name}${NC}"
    echo "Policy: ${policy_file}"
    echo "Data: ${data_file}"

    # Run OPA evaluation for denials
    local denials
    denials=$(${OPA_BINARY} eval -d "${policy_file}" -i "${data_file}" "data.nephoran.api.validation.deny" --format json 2>/dev/null | jq -r '.result[0].expressions[0].value[]? // empty')
    
    if [[ -z "${denials}" ]]; then
        echo -e "${GREEN}✓ PASS - Policy correctly allowed request${NC}"
        echo "PASS: ${test_name} - Correctly allowed" >> "${RESULTS_DIR}/test-results.log"
    else
        echo -e "${RED}✗ FAIL - Policy should have allowed request${NC}"
        echo "Denial reasons:"
        echo "${denials}" | while IFS= read -r line; do
            echo "  - ${line}"
        done
        echo "FAIL: ${test_name} - Should have been allowed" >> "${RESULTS_DIR}/test-results.log"
    fi
    echo
}

# Function to generate test data files
generate_test_data() {
    echo -e "${BLUE}Generating test data files...${NC}"
    
    # Create valid intent request
    cat > "${TEST_DATA_DIR}/valid-intent.json" << EOF
{
  "method": "POST",
  "path": "/process",
  "headers": {
    "content-type": "application/json",
    "authorization": "Bearer validtoken",
    "content-length": "450"
  },
  "body": "{\"intent\": \"Deploy a high-availability AMF network function\", \"network_function\": {\"type\": \"AMF\"}, \"oran_config\": {\"interface_type\": \"N2\"}}",
  "client_ip": "10.0.1.100"
}
EOF

    # Create SQL injection test
    cat > "${TEST_DATA_DIR}/sql-injection.json" << EOF
{
  "method": "POST",
  "path": "/process",
  "headers": {
    "content-type": "application/json",
    "authorization": "Bearer validtoken",
    "content-length": "200"
  },
  "body": "{\"intent\": \"Deploy AMF'; DROP TABLE users; --\"}",
  "client_ip": "192.168.1.100"
}
EOF

    # Create oversized request test
    cat > "${TEST_DATA_DIR}/oversized-request.json" << EOF
{
  "method": "POST",
  "path": "/process",
  "headers": {
    "content-type": "application/json",
    "authorization": "Bearer validtoken",
    "content-length": "3000000"
  },
  "body": "large payload",
  "client_ip": "192.168.1.103"
}
EOF

    # Create no auth test
    cat > "${TEST_DATA_DIR}/no-auth.json" << EOF
{
  "method": "POST",
  "path": "/process",
  "headers": {
    "content-type": "application/json",
    "content-length": "100"
  },
  "body": "{\"intent\": \"Deploy AMF network function\"}",
  "client_ip": "192.168.1.104"
}
EOF

    echo -e "${GREEN}Test data files generated${NC}"
    echo
}

# Function to run comprehensive tests
run_comprehensive_tests() {
    echo -e "${BLUE}Running comprehensive OPA policy tests...${NC}"
    echo
    
    # Clear previous results
    > "${RESULTS_DIR}/test-results.log"
    
    # Test 1: Valid intent request should be allowed
    test_policy_allow \
        "${POLICY_DIR}/request-validation.rego" \
        "${TEST_DATA_DIR}/valid-intent.json" \
        "Valid intent processing request"
    
    # Test 2: SQL injection should be denied
    test_policy_denial \
        "${POLICY_DIR}/request-validation.rego" \
        "${TEST_DATA_DIR}/sql-injection.json" \
        "SQL injection attempt"
    
    # Test 3: Oversized request should be denied
    test_policy_denial \
        "${POLICY_DIR}/request-validation.rego" \
        "${TEST_DATA_DIR}/oversized-request.json" \
        "Oversized request"
    
    # Test 4: No authorization should be denied
    test_policy_denial \
        "${POLICY_DIR}/request-validation.rego" \
        "${TEST_DATA_DIR}/no-auth.json" \
        "Request without authorization"
    
    # Test security validation policies
    if [[ -f "${POLICY_DIR}/security-validation.rego" ]]; then
        test_policy_denial \
            "${POLICY_DIR}/security-validation.rego" \
            "${TEST_DATA_DIR}/sql-injection.json" \
            "Security validation - SQL injection"
    fi
    
    # Test O-RAN compliance policies
    if [[ -f "${POLICY_DIR}/oran-compliance.rego" ]]; then
        # Create invalid O-RAN test data
        cat > "${TEST_DATA_DIR}/invalid-oran.json" << EOF
{
  "method": "POST",
  "path": "/process",
  "headers": {
    "content-type": "application/json",
    "authorization": "Bearer validtoken",
    "content-length": "200"
  },
  "body": "{\"intent\": \"Deploy network function\", \"oran_config\": {\"interface_type\": \"INVALID\"}, \"network_function\": {\"type\": \"INVALID_NF\"}}",
  "client_ip": "192.168.1.105"
}
EOF
        
        test_policy_denial \
            "${POLICY_DIR}/oran-compliance.rego" \
            "${TEST_DATA_DIR}/invalid-oran.json" \
            "Invalid O-RAN configuration"
    fi
}

# Function to benchmark policy performance
benchmark_policies() {
    echo -e "${BLUE}Benchmarking policy performance...${NC}"
    
    local iterations=1000
    local start_time
    local end_time
    local duration
    
    start_time=$(date +%s.%N)
    
    for ((i=1; i<=iterations; i++)); do
        ${OPA_BINARY} eval -d "${POLICY_DIR}/request-validation.rego" \
                          -i "${TEST_DATA_DIR}/valid-intent.json" \
                          "data.nephoran.api.validation.deny" \
                          --format json > /dev/null 2>&1
    done
    
    end_time=$(date +%s.%N)
    duration=$(echo "${end_time} - ${start_time}" | bc)
    avg_time=$(echo "scale=4; ${duration} / ${iterations}" | bc)
    
    echo "Performance Results:"
    echo "  Total time: ${duration}s"
    echo "  Iterations: ${iterations}"
    echo "  Average time per evaluation: ${avg_time}s"
    echo "  Evaluations per second: $(echo "scale=2; ${iterations} / ${duration}" | bc)"
    
    echo "PERFORMANCE: ${iterations} evaluations in ${duration}s (${avg_time}s avg)" >> "${RESULTS_DIR}/test-results.log"
}

# Function to validate policy syntax
validate_policy_syntax() {
    echo -e "${BLUE}Validating policy syntax...${NC}"
    
    for policy_file in "${POLICY_DIR}"/*.rego; do
        if [[ -f "${policy_file}" ]]; then
            echo "Validating: $(basename "${policy_file}")"
            if ${OPA_BINARY} fmt "${policy_file}" > /dev/null 2>&1; then
                echo -e "${GREEN}✓ PASS - Syntax valid${NC}"
                echo "SYNTAX_PASS: $(basename "${policy_file}")" >> "${RESULTS_DIR}/test-results.log"
            else
                echo -e "${RED}✗ FAIL - Syntax error${NC}"
                echo "SYNTAX_FAIL: $(basename "${policy_file}")" >> "${RESULTS_DIR}/test-results.log"
            fi
        fi
    done
    echo
}

# Function to test policy coverage
test_policy_coverage() {
    echo -e "${BLUE}Testing policy coverage...${NC}"
    
    local test_scenarios=(
        "sql_injection"
        "xss_attack"
        "command_injection"
        "path_traversal"
        "oversized_request"
        "no_authorization"
        "invalid_oran"
        "rate_limiting"
    )
    
    local covered=0
    local total=${#test_scenarios[@]}
    
    for scenario in "${test_scenarios[@]}"; do
        # Check if we have test coverage for this scenario
        if grep -r "${scenario}" "${POLICY_DIR}/" > /dev/null 2>&1; then
            echo -e "${GREEN}✓${NC} Coverage for: ${scenario}"
            ((covered++))
        else
            echo -e "${YELLOW}!${NC} No coverage for: ${scenario}"
        fi
    done
    
    local coverage_percent=$((covered * 100 / total))
    echo
    echo "Policy Coverage: ${covered}/${total} scenarios (${coverage_percent}%)"
    echo "COVERAGE: ${coverage_percent}%" >> "${RESULTS_DIR}/test-results.log"
}

# Function to generate report
generate_report() {
    echo -e "${BLUE}Generating test report...${NC}"
    
    local total_tests
    local passed_tests
    local failed_tests
    
    total_tests=$(grep -c "PASS\|FAIL" "${RESULTS_DIR}/test-results.log" 2>/dev/null || echo "0")
    passed_tests=$(grep -c "PASS" "${RESULTS_DIR}/test-results.log" 2>/dev/null || echo "0")
    failed_tests=$(grep -c "FAIL" "${RESULTS_DIR}/test-results.log" 2>/dev/null || echo "0")
    
    cat > "${RESULTS_DIR}/test-report.txt" << EOF
OPA Policy Test Report - Nephoran Intent Operator
================================================
Generated: $(date)

Summary:
--------
Total Tests: ${total_tests}
Passed: ${passed_tests}
Failed: ${failed_tests}
Success Rate: $(( passed_tests * 100 / (total_tests > 0 ? total_tests : 1) ))%

Detailed Results:
----------------
$(cat "${RESULTS_DIR}/test-results.log")

EOF
    
    echo "Test report generated: ${RESULTS_DIR}/test-report.txt"
    
    if [[ ${failed_tests} -eq 0 ]]; then
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}${failed_tests} test(s) failed${NC}"
        exit 1
    fi
}

# Main execution
main() {
    # Check if OPA is installed
    if ! command -v ${OPA_BINARY} &> /dev/null; then
        echo -e "${RED}Error: OPA binary not found. Please install OPA first.${NC}"
        echo "Visit: https://www.openpolicyagent.org/docs/latest/#running-opa"
        exit 1
    fi
    
    # Check if policy directory exists
    if [[ ! -d "${POLICY_DIR}" ]]; then
        echo -e "${RED}Error: Policy directory not found: ${POLICY_DIR}${NC}"
        exit 1
    fi
    
    # Create test data directory
    mkdir -p "${TEST_DATA_DIR}"
    
    # Generate test data
    generate_test_data
    
    # Validate policy syntax
    validate_policy_syntax
    
    # Test policy coverage
    test_policy_coverage
    
    # Run comprehensive tests
    run_comprehensive_tests
    
    # Benchmark performance
    if [[ "${1:-}" == "--benchmark" ]]; then
        benchmark_policies
    fi
    
    # Generate report
    generate_report
}

# Script usage
usage() {
    echo "Usage: $0 [--benchmark]"
    echo "  --benchmark: Include performance benchmarking"
    echo
    echo "Environment variables:"
    echo "  OPA_BINARY: Path to OPA binary (default: opa)"
    echo "  POLICY_DIR: Directory containing policies (default: ./policies)"
    echo "  TEST_DATA_DIR: Directory for test data (default: ./examples)"
    echo "  RESULTS_DIR: Directory for results (default: ./test-results)"
}

# Handle arguments
if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    usage
    exit 0
fi

# Run main function
main "$@"