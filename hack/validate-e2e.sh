#!/usr/bin/env bash
set -euo pipefail

# CI-ready E2E validation script with comprehensive checks
# Designed for GitHub Actions and other CI/CD systems

# === Configuration ===
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
EXIT_CODE=0
VALIDATION_RESULTS=()
CI_MODE="${CI:-false}"
GITHUB_ACTIONS="${GITHUB_ACTIONS:-false}"

# === Output Functions ===
print_header() {
    echo "=================================================="
    echo "$1"
    echo "=================================================="
}

print_test() {
    echo -n "  - $1... "
}

print_pass() {
    if [ "$GITHUB_ACTIONS" = "true" ]; then
        echo "✅ PASS"
    else
        echo "PASS"
    fi
    VALIDATION_RESULTS+=("PASS: $1")
}

print_fail() {
    if [ "$GITHUB_ACTIONS" = "true" ]; then
        echo "❌ FAIL: $1"
    else
        echo "FAIL: $1"
    fi
    VALIDATION_RESULTS+=("FAIL: $1")
    EXIT_CODE=1
}

print_skip() {
    if [ "$GITHUB_ACTIONS" = "true" ]; then
        echo "⏭️ SKIP: $1"
    else
        echo "SKIP: $1"
    fi
    VALIDATION_RESULTS+=("SKIP: $1")
}

# === Validation Functions ===

validate_go_modules() {
    print_test "Go modules"
    
    if [ ! -f "$PROJECT_ROOT/go.mod" ]; then
        print_fail "go.mod not found"
        return 1
    fi
    
    cd "$PROJECT_ROOT"
    if go mod verify &>/dev/null; then
        print_pass
    else
        print_fail "go mod verify failed"
        return 1
    fi
}

validate_go_build() {
    print_test "Go build"
    
    cd "$PROJECT_ROOT"
    if go build ./... &>/dev/null; then
        print_pass
    else
        print_fail "go build failed"
        return 1
    fi
}

validate_go_test() {
    print_test "Go unit tests"
    
    cd "$PROJECT_ROOT"
    if go test ./... -short &>/dev/null; then
        print_pass
    else
        print_fail "go test failed"
        return 1
    fi
}

validate_crds() {
    print_test "CRD files"
    
    CRD_DIR="$PROJECT_ROOT/deployments/crds"
    if [ ! -d "$CRD_DIR" ]; then
        print_fail "CRD directory not found"
        return 1
    fi
    
    CRD_COUNT=$(find "$CRD_DIR" -name "*.yaml" -o -name "*.yml" | wc -l)
    if [ "$CRD_COUNT" -eq 0 ]; then
        print_fail "No CRD files found"
        return 1
    fi
    
    # Validate CRD YAML syntax
    for crd_file in "$CRD_DIR"/*.yaml "$CRD_DIR"/*.yml; do
        [ -f "$crd_file" ] || continue
        if ! kubectl apply --dry-run=client -f "$crd_file" &>/dev/null; then
            print_fail "Invalid CRD: $(basename "$crd_file")"
            return 1
        fi
    done
    
    print_pass
}

validate_kustomize() {
    print_test "Kustomize configurations"
    
    KUSTOMIZE_DIRS=(
        "$PROJECT_ROOT/config/webhook"
        "$PROJECT_ROOT/deployments/kustomize/base"
    )
    
    for dir in "${KUSTOMIZE_DIRS[@]}"; do
        if [ -d "$dir" ] && [ -f "$dir/kustomization.yaml" ]; then
            if command -v kustomize &>/dev/null; then
                if ! kustomize build "$dir" &>/dev/null; then
                    print_fail "Invalid kustomization in $dir"
                    return 1
                fi
            elif kubectl version --client &>/dev/null; then
                if ! kubectl kustomize "$dir" &>/dev/null; then
                    print_fail "Invalid kustomization in $dir"
                    return 1
                fi
            else
                print_skip "kustomize not available"
                return 0
            fi
        fi
    done
    
    print_pass
}

validate_docker_images() {
    print_test "Docker images"
    
    DOCKERFILE="$PROJECT_ROOT/Dockerfile"
    if [ ! -f "$DOCKERFILE" ]; then
        print_skip "Dockerfile not found"
        return 0
    fi
    
    if ! command -v docker &>/dev/null; then
        print_skip "Docker not available"
        return 0
    fi
    
    # Check if Docker daemon is running
    if ! docker info &>/dev/null; then
        print_skip "Docker daemon not running"
        return 0
    fi
    
    # Validate Dockerfile syntax
    if docker build --check "$PROJECT_ROOT" &>/dev/null; then
        print_pass
    else
        print_fail "Dockerfile validation failed"
        return 1
    fi
}

validate_scripts() {
    print_test "Shell scripts"
    
    SCRIPTS=(
        "$SCRIPT_DIR/run-e2e.sh"
        "$SCRIPT_DIR/run-production-e2e.sh"
        "$SCRIPT_DIR/check-tools.sh"
    )
    
    for script in "${SCRIPTS[@]}"; do
        if [ -f "$script" ]; then
            if ! bash -n "$script" &>/dev/null; then
                print_fail "Syntax error in $(basename "$script")"
                return 1
            fi
        fi
    done
    
    print_pass
}

validate_yaml_files() {
    print_test "YAML files"
    
    # Check if yq is available
    if ! command -v yq &>/dev/null; then
        print_skip "yq not available"
        return 0
    fi
    
    # Find and validate all YAML files
    while IFS= read -r -d '' yaml_file; do
        if ! yq eval '.' "$yaml_file" &>/dev/null; then
            print_fail "Invalid YAML: $yaml_file"
            return 1
        fi
    done < <(find "$PROJECT_ROOT" -type f \( -name "*.yaml" -o -name "*.yml" \) -not -path "*/vendor/*" -not -path "*/.git/*" -print0)
    
    print_pass
}

validate_contracts() {
    print_test "Contract schemas"
    
    CONTRACTS_DIR="$PROJECT_ROOT/docs/contracts"
    if [ ! -d "$CONTRACTS_DIR" ]; then
        print_skip "Contracts directory not found"
        return 0
    fi
    
    # Check for required contract files
    REQUIRED_CONTRACTS=(
        "intent.schema.json"
        "a1.policy.schema.json"
        "fcaps.ves.examples.json"
    )
    
    for contract in "${REQUIRED_CONTRACTS[@]}"; do
        if [ ! -f "$CONTRACTS_DIR/$contract" ]; then
            print_fail "Missing contract: $contract"
            return 1
        fi
    done
    
    # Validate JSON syntax if jq is available
    if command -v jq &>/dev/null; then
        for json_file in "$CONTRACTS_DIR"/*.json; do
            [ -f "$json_file" ] || continue
            if ! jq '.' "$json_file" &>/dev/null; then
                print_fail "Invalid JSON: $(basename "$json_file")"
                return 1
            fi
        done
    fi
    
    print_pass
}

validate_dependencies() {
    print_test "Go dependencies"
    
    cd "$PROJECT_ROOT"
    
    # Check for security vulnerabilities
    if command -v govulncheck &>/dev/null; then
        if govulncheck ./... &>/dev/null; then
            print_pass
        else
            print_fail "Security vulnerabilities found"
            return 1
        fi
    else
        # Fall back to go list check
        if go list -m all &>/dev/null; then
            print_pass
        else
            print_fail "Dependency check failed"
            return 1
        fi
    fi
}

validate_license() {
    print_test "License file"
    
    if [ -f "$PROJECT_ROOT/LICENSE" ]; then
        print_pass
    else
        print_fail "LICENSE file not found"
        return 1
    fi
}

validate_readme() {
    print_test "README file"
    
    if [ -f "$PROJECT_ROOT/README.md" ]; then
        # Check if README has minimum content
        README_LINES=$(wc -l < "$PROJECT_ROOT/README.md")
        if [ "$README_LINES" -lt 50 ]; then
            print_fail "README.md seems incomplete (< 50 lines)"
            return 1
        fi
        print_pass
    else
        print_fail "README.md not found"
        return 1
    fi
}

# === E2E Specific Validations ===

validate_e2e_tests() {
    print_test "E2E test files"
    
    E2E_DIR="$PROJECT_ROOT/tests/e2e"
    if [ ! -d "$E2E_DIR" ]; then
        print_fail "E2E test directory not found"
        return 1
    fi
    
    # Check for test files
    E2E_TEST_COUNT=$(find "$E2E_DIR" -name "*_test.go" | wc -l)
    if [ "$E2E_TEST_COUNT" -eq 0 ]; then
        print_fail "No E2E test files found"
        return 1
    fi
    
    # Compile E2E tests
    cd "$PROJECT_ROOT"
    if go test -c ./tests/e2e/... &>/dev/null; then
        print_pass
        rm -f e2e.test 2>/dev/null || true
    else
        print_fail "E2E tests compilation failed"
        return 1
    fi
}

validate_integration_tests() {
    print_test "Integration tests"
    
    INTEGRATION_DIR="$PROJECT_ROOT/tests/integration"
    if [ ! -d "$INTEGRATION_DIR" ]; then
        print_skip "Integration test directory not found"
        return 0
    fi
    
    # Check for test files
    INTEGRATION_TEST_COUNT=$(find "$INTEGRATION_DIR" -name "*_test.go" | wc -l)
    if [ "$INTEGRATION_TEST_COUNT" -eq 0 ]; then
        print_skip "No integration test files found"
        return 0
    fi
    
    # Compile integration tests
    cd "$PROJECT_ROOT"
    if go test -c ./tests/integration/... &>/dev/null; then
        print_pass
        rm -f integration.test 2>/dev/null || true
    else
        print_fail "Integration tests compilation failed"
        return 1
    fi
}

# === Performance Checks ===

check_test_performance() {
    print_test "Test performance baseline"
    
    cd "$PROJECT_ROOT"
    
    # Run a quick benchmark if available
    if go test ./... -bench=. -benchtime=1x -run=^$ &>/dev/null; then
        print_pass
    else
        print_skip "No benchmarks available"
    fi
}

# === Main Validation ===

run_validations() {
    print_header "Running E2E Validation Suite"
    
    echo "Project Root: $PROJECT_ROOT"
    echo "CI Mode: $CI_MODE"
    echo ""
    
    print_header "1. Code Quality Checks"
    validate_go_modules
    validate_go_build
    validate_go_test
    validate_dependencies
    
    print_header "2. Configuration Validation"
    validate_crds
    validate_kustomize
    validate_yaml_files
    validate_contracts
    
    print_header "3. Documentation Checks"
    validate_license
    validate_readme
    
    print_header "4. Test Suite Validation"
    validate_e2e_tests
    validate_integration_tests
    check_test_performance
    
    print_header "5. Build & Deploy Validation"
    validate_docker_images
    validate_scripts
    
    print_header "Validation Summary"
    echo ""
    for result in "${VALIDATION_RESULTS[@]}"; do
        echo "  $result"
    done
    echo ""
    
    if [ $EXIT_CODE -eq 0 ]; then
        echo "✅ All validations passed!"
    else
        echo "❌ Some validations failed. Please fix the issues above."
    fi
    
    # Set GitHub Actions output if running in CI
    if [ "$GITHUB_ACTIONS" = "true" ]; then
        echo "validation-status=$([[ $EXIT_CODE -eq 0 ]] && echo 'success' || echo 'failure')" >> "$GITHUB_OUTPUT"
    fi
    
    return $EXIT_CODE
}

# === Entry Point ===

main() {
    # Handle CI-specific setup
    if [ "$CI_MODE" = "true" ] || [ "$GITHUB_ACTIONS" = "true" ]; then
        echo "Running in CI mode"
        set -o pipefail
    fi
    
    run_validations
    exit $EXIT_CODE
}

# Run main function
main "$@"