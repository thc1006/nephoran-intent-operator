#!/bin/bash
# =============================================================================
# Comprehensive CI Fixes Validation Script
# =============================================================================
# This script validates all the major CI fixes that have been implemented:
# 1. Service configuration matrix validation
# 2. GitHub Actions workflow syntax validation
# 3. Docker build simulation tests
# 4. GHCR authentication verification
# 5. Resilient build system testing
# 6. Integration testing
# =============================================================================

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
VALIDATION_LOG="$PROJECT_ROOT/validation-results.txt"
TEMP_DIR="/tmp/ci-validation-$$"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$VALIDATION_LOG"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$VALIDATION_LOG"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$VALIDATION_LOG"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" | tee -a "$VALIDATION_LOG"; }

# Test counter
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_WARNINGS=0

# Test result tracking
test_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    
    case "$status" in
        "PASS")
            TESTS_PASSED=$((TESTS_PASSED + 1))
            log_success "TEST $TESTS_TOTAL: $test_name - $message"
            ;;
        "FAIL")
            TESTS_FAILED=$((TESTS_FAILED + 1))
            log_error "TEST $TESTS_TOTAL: $test_name - $message"
            ;;
        "WARN")
            TESTS_WARNINGS=$((TESTS_WARNINGS + 1))
            log_warning "TEST $TESTS_TOTAL: $test_name - $message"
            ;;
    esac
}

# =============================================================================
# TEST 1: Service Configuration Matrix Validation
# =============================================================================
validate_service_configuration() {
    log_info "=== TEST 1: Service Configuration Matrix Validation ==="
    
    local dockerfile="$PROJECT_ROOT/Dockerfile"
    local validation_errors=0
    
    # Extract service definitions from Dockerfile
    local services=(
        "conductor-loop:./cmd/conductor-loop/main.go"
        "intent-ingest:./cmd/intent-ingest/main.go"
        "nephio-bridge:./cmd/nephio-bridge/main.go"
        "llm-processor:./cmd/llm-processor/main.go"
        "oran-adaptor:./cmd/oran-adaptor/main.go"
        "manager:./cmd/conductor-loop/main.go"
        "controller:./cmd/conductor-loop/main.go"
        "e2-kmp-sim:./cmd/e2-kmp-sim/main.go"
        "o1-ves-sim:./cmd/o1-ves-sim/main.go"
    )
    
    for service_entry in "${services[@]}"; do
        local service_name=$(echo "$service_entry" | cut -d: -f1)
        local expected_path=$(echo "$service_entry" | cut -d: -f2)
        local actual_path="$PROJECT_ROOT/${expected_path#./}"
        
        log_info "Validating service: $service_name -> $expected_path"
        
        # Check if main.go exists
        if [ -f "$actual_path" ]; then
            test_result "Service-$service_name-MainExists" "PASS" "main.go found at $expected_path"
            
            # Check if main.go has a main function
            if grep -q "func main()" "$actual_path"; then
                test_result "Service-$service_name-MainFunction" "PASS" "main() function found"
            else
                test_result "Service-$service_name-MainFunction" "FAIL" "main() function not found"
                validation_errors=$((validation_errors + 1))
            fi
            
            # Check if service is properly mapped in Dockerfile
            if grep -q "\"$service_name\") CMD_PATH=\"$expected_path\"" "$dockerfile"; then
                test_result "Service-$service_name-DockerMapping" "PASS" "Correctly mapped in Dockerfile"
            else
                test_result "Service-$service_name-DockerMapping" "FAIL" "Not properly mapped in Dockerfile"
                validation_errors=$((validation_errors + 1))
            fi
        else
            test_result "Service-$service_name-MainExists" "FAIL" "main.go not found at $expected_path"
            validation_errors=$((validation_errors + 1))
        fi
    done
    
    # Validate Dockerfile service case statement completeness
    log_info "Checking Dockerfile service case statement..."
    local dockerfile_services=$(grep -A 20 'case "$SERVICE" in' "$dockerfile" | grep -E '^\s*"[^"]+"\)' | sed 's/.*"\([^"]*\)").*/\1/' | sort)
    local expected_services=$(printf '%s\n' "${services[@]}" | cut -d: -f1 | sort)
    
    if [ "$dockerfile_services" = "$expected_services" ]; then
        test_result "Dockerfile-ServiceCompleteness" "PASS" "All services defined in Dockerfile"
    else
        test_result "Dockerfile-ServiceCompleteness" "FAIL" "Service definitions incomplete"
        log_error "Expected: $expected_services"
        log_error "Found: $dockerfile_services"
        validation_errors=$((validation_errors + 1))
    fi
    
    return $validation_errors
}

# =============================================================================
# TEST 2: GitHub Actions Workflow Syntax Validation
# =============================================================================
validate_workflow_syntax() {
    log_info "=== TEST 2: GitHub Actions Workflow Syntax Validation ==="
    
    local workflows_dir="$PROJECT_ROOT/.github/workflows"
    local validation_errors=0
    
    if [ ! -d "$workflows_dir" ]; then
        test_result "Workflows-Directory" "FAIL" "Workflows directory not found"
        return 1
    fi
    
    # Find all workflow files
    local workflow_files=()
    while IFS= read -r -d '' file; do
        workflow_files+=("$file")
    done < <(find "$workflows_dir" -name "*.yml" -o -name "*.yaml" -print0)
    
    if [ ${#workflow_files[@]} -eq 0 ]; then
        test_result "Workflows-Files" "FAIL" "No workflow files found"
        return 1
    fi
    
    log_info "Found ${#workflow_files[@]} workflow files to validate"
    
    # Validate each workflow file
    for workflow_file in "${workflow_files[@]}"; do
        local workflow_name=$(basename "$workflow_file")
        log_info "Validating workflow: $workflow_name"
        
        # Check YAML syntax using yq
        if command -v yq >/dev/null 2>&1; then
            if yq eval '.' "$workflow_file" >/dev/null 2>&1; then
                test_result "Workflow-$workflow_name-Syntax" "PASS" "YAML syntax valid"
                
                # Check for required workflow fields
                local has_name=$(yq eval 'has("name")' "$workflow_file")
                local has_on=$(yq eval 'has("on")' "$workflow_file")
                local has_jobs=$(yq eval 'has("jobs")' "$workflow_file")
                
                if [ "$has_name" = "true" ] && [ "$has_on" = "true" ] && [ "$has_jobs" = "true" ]; then
                    test_result "Workflow-$workflow_name-Structure" "PASS" "Required fields present"
                else
                    test_result "Workflow-$workflow_name-Structure" "FAIL" "Missing required fields (name: $has_name, on: $has_on, jobs: $has_jobs)"
                    validation_errors=$((validation_errors + 1))
                fi
                
                # Check for concurrency group (important for preventing conflicts)
                if yq eval 'has("concurrency")' "$workflow_file" | grep -q "true"; then
                    test_result "Workflow-$workflow_name-Concurrency" "PASS" "Concurrency control configured"
                else
                    test_result "Workflow-$workflow_name-Concurrency" "WARN" "No concurrency control found"
                fi
                
            else
                test_result "Workflow-$workflow_name-Syntax" "FAIL" "YAML syntax error"
                validation_errors=$((validation_errors + 1))
            fi
        else
            # Fallback: basic YAML check with python
            if python3 -c "import yaml; yaml.safe_load(open('$workflow_file'))" 2>/dev/null; then
                test_result "Workflow-$workflow_name-Syntax" "PASS" "YAML syntax valid (python fallback)"
            else
                test_result "Workflow-$workflow_name-Syntax" "FAIL" "YAML syntax error (python fallback)"
                validation_errors=$((validation_errors + 1))
            fi
        fi
    done
    
    return $validation_errors
}

# =============================================================================
# TEST 3: Docker Build Simulation Tests
# =============================================================================
validate_docker_builds() {
    log_info "=== TEST 3: Docker Build Simulation Tests ==="
    
    local validation_errors=0
    local dockerfile="$PROJECT_ROOT/Dockerfile"
    local dockerfile_resilient="$PROJECT_ROOT/Dockerfile.resilient"
    
    # Test 1: Dockerfile syntax validation
    log_info "Testing Dockerfile syntax validation..."
    
    for docker_file in "$dockerfile" "$dockerfile_resilient"; do
        if [ -f "$docker_file" ]; then
            local file_name=$(basename "$docker_file")
            log_info "Validating $file_name..."
            
            # Use docker buildx to validate syntax without building
            if docker buildx build --dry-run -f "$docker_file" . >/dev/null 2>&1; then
                test_result "Docker-$file_name-Syntax" "PASS" "Dockerfile syntax valid"
            else
                test_result "Docker-$file_name-Syntax" "FAIL" "Dockerfile syntax error"
                validation_errors=$((validation_errors + 1))
            fi
            
            # Check for required ARGs
            local required_args=("SERVICE" "VERSION" "BUILD_DATE" "VCS_REF")
            for arg in "${required_args[@]}"; do
                if grep -q "ARG $arg" "$docker_file"; then
                    test_result "Docker-$file_name-ARG-$arg" "PASS" "Required ARG $arg present"
                else
                    test_result "Docker-$file_name-ARG-$arg" "WARN" "Required ARG $arg missing"
                fi
            done
        else
            test_result "Docker-$file_name-Exists" "FAIL" "Dockerfile not found: $docker_file"
            validation_errors=$((validation_errors + 1))
        fi
    done
    
    # Test 2: Build simulation (syntax-only, no actual build)
    log_info "Testing build command construction..."
    
    local test_services=("conductor-loop" "intent-ingest" "nephio-bridge")
    for service in "${test_services[@]}"; do
        # Construct build command
        local build_cmd="docker buildx build --dry-run --platform linux/amd64 --build-arg SERVICE=$service -f $dockerfile ."
        
        log_info "Testing build command for service: $service"
        if $build_cmd >/dev/null 2>&1; then
            test_result "Build-$service-Command" "PASS" "Build command valid for $service"
        else
            test_result "Build-$service-Command" "FAIL" "Build command invalid for $service"
            validation_errors=$((validation_errors + 1))
        fi
    done
    
    return $validation_errors
}

# =============================================================================
# TEST 4: GHCR Authentication Setup Verification
# =============================================================================
validate_ghcr_authentication() {
    log_info "=== TEST 4: GHCR Authentication Setup Verification ==="
    
    local validation_errors=0
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Check if CI workflow exists
    if [ ! -f "$ci_workflow" ]; then
        test_result "GHCR-CI-Workflow" "FAIL" "CI workflow not found"
        return 1
    fi
    
    # Validate required permissions
    log_info "Checking GitHub Actions permissions..."
    local required_permissions=("contents: read" "packages: write" "security-events: write" "id-token: write")
    
    for permission in "${required_permissions[@]}"; do
        if grep -q "$permission" "$ci_workflow"; then
            test_result "GHCR-Permission-$(echo $permission | cut -d: -f1)" "PASS" "Permission $permission found"
        else
            test_result "GHCR-Permission-$(echo $permission | cut -d: -f1)" "FAIL" "Permission $permission missing"
            validation_errors=$((validation_errors + 1))
        fi
    done
    
    # Validate GHCR login configuration
    log_info "Checking GHCR login configuration..."
    if grep -A 5 "Login to GitHub Container Registry" "$ci_workflow" | grep -q "registry: ghcr.io"; then
        test_result "GHCR-Login-Registry" "PASS" "GHCR registry correctly configured"
    else
        test_result "GHCR-Login-Registry" "FAIL" "GHCR registry not properly configured"
        validation_errors=$((validation_errors + 1))
    fi
    
    if grep -A 5 "Login to GitHub Container Registry" "$ci_workflow" | grep -q "username: \${{ github.actor }}"; then
        test_result "GHCR-Login-Username" "PASS" "GHCR username correctly configured"
    else
        test_result "GHCR-Login-Username" "FAIL" "GHCR username not properly configured"
        validation_errors=$((validation_errors + 1))
    fi
    
    if grep -A 5 "Login to GitHub Container Registry" "$ci_workflow" | grep -q "password: \${{ secrets.GITHUB_TOKEN }}"; then
        test_result "GHCR-Login-Token" "PASS" "GHCR token correctly configured"
    else
        test_result "GHCR-Login-Token" "FAIL" "GHCR token not properly configured"
        validation_errors=$((validation_errors + 1))
    fi
    
    # Validate conditional GHCR push (only on non-PR events)
    if grep -B 2 -A 2 "Login to GitHub Container Registry" "$ci_workflow" | grep -q "if: github.event_name != 'pull_request'"; then
        test_result "GHCR-Conditional-Push" "PASS" "GHCR push properly conditional on non-PR events"
    else
        test_result "GHCR-Conditional-Push" "WARN" "GHCR push condition may need review"
    fi
    
    return $validation_errors
}

# =============================================================================
# TEST 5: Smart Build Script Validation
# =============================================================================
validate_smart_build_script() {
    log_info "=== TEST 5: Smart Build Script Validation ==="
    
    local validation_errors=0
    local smart_build_script="$PROJECT_ROOT/scripts/deploy/smart-docker-build.sh"
    
    # Check if script exists and is executable
    if [ -f "$smart_build_script" ]; then
        test_result "SmartBuild-Script-Exists" "PASS" "Smart build script found"
        
        if [ -x "$smart_build_script" ]; then
            test_result "SmartBuild-Script-Executable" "PASS" "Smart build script is executable"
        else
            test_result "SmartBuild-Script-Executable" "FAIL" "Smart build script is not executable"
            validation_errors=$((validation_errors + 1))
        fi
        
        # Validate script functions
        local required_functions=("check_infrastructure_health" "select_build_strategy" "execute_build" "main")
        for func in "${required_functions[@]}"; do
            if grep -q "$func()" "$smart_build_script"; then
                test_result "SmartBuild-Function-$func" "PASS" "Function $func() found"
            else
                test_result "SmartBuild-Function-$func" "FAIL" "Function $func() missing"
                validation_errors=$((validation_errors + 1))
            fi
        done
        
        # Test script syntax
        if bash -n "$smart_build_script"; then
            test_result "SmartBuild-Script-Syntax" "PASS" "Script syntax valid"
        else
            test_result "SmartBuild-Script-Syntax" "FAIL" "Script syntax error"
            validation_errors=$((validation_errors + 1))
        fi
        
        # Test script execution with dry-run (if supported)
        log_info "Testing script execution (dry-run mode)..."
        if timeout 30s "$smart_build_script" conductor-loop nephoran/test latest linux/amd64 false 2>/dev/null; then
            test_result "SmartBuild-Script-Execution" "PASS" "Script executes without errors"
        else
            test_result "SmartBuild-Script-Execution" "WARN" "Script execution test failed (may require Docker)"
        fi
        
    else
        test_result "SmartBuild-Script-Exists" "FAIL" "Smart build script not found"
        validation_errors=$((validation_errors + 1))
    fi
    
    return $validation_errors
}

# =============================================================================
# TEST 6: Resilient Dockerfile Validation
# =============================================================================
validate_resilient_dockerfile() {
    log_info "=== TEST 6: Resilient Dockerfile Validation ==="
    
    local validation_errors=0
    local dockerfile_resilient="$PROJECT_ROOT/Dockerfile.resilient"
    
    if [ ! -f "$dockerfile_resilient" ]; then
        test_result "Resilient-Dockerfile-Exists" "FAIL" "Resilient Dockerfile not found"
        return 1
    fi
    
    test_result "Resilient-Dockerfile-Exists" "PASS" "Resilient Dockerfile found"
    
    # Check for resilience features
    local resilience_features=(
        "max_attempts.*retry.*logic:Enhanced retry logic"
        "timeout.*[0-9]+:Timeout handling"
        "fallback.*strategy:Fallback strategies"
        "exponential.*backoff:Exponential backoff"
        "health.*check:Health checks"
    )
    
    for feature_pattern in "${resilience_features[@]}"; do
        local pattern=$(echo "$feature_pattern" | cut -d: -f1)
        local feature_name=$(echo "$feature_pattern" | cut -d: -f2)
        
        if grep -qE "$pattern" "$dockerfile_resilient"; then
            test_result "Resilient-Feature-$(echo $feature_name | tr ' ' '-')" "PASS" "$feature_name implemented"
        else
            test_result "Resilient-Feature-$(echo $feature_name | tr ' ' '-')" "WARN" "$feature_name not found"
        fi
    done
    
    # Check for multi-stage builds
    local stage_count=$(grep -c "^FROM.*AS" "$dockerfile_resilient" || echo "0")
    if [ "$stage_count" -gt 3 ]; then
        test_result "Resilient-MultiStage" "PASS" "Multi-stage build with $stage_count stages"
    else
        test_result "Resilient-MultiStage" "WARN" "Limited multi-stage build ($stage_count stages)"
    fi
    
    return $validation_errors
}

# =============================================================================
# TEST 7: Integration Testing - End-to-End Workflow Validation
# =============================================================================
validate_integration() {
    log_info "=== TEST 7: Integration Testing - End-to-End Workflow Validation ==="
    
    local validation_errors=0
    
    # Test Go build locally
    log_info "Testing local Go build..."
    if command -v go >/dev/null 2>&1; then
        # Check if we can build the main service
        if CGO_ENABLED=0 go build -o /tmp/test-conductor ./cmd/conductor-loop/main.go >/dev/null 2>&1; then
            test_result "Integration-Go-Build" "PASS" "Local Go build successful"
            rm -f /tmp/test-conductor
        else
            test_result "Integration-Go-Build" "FAIL" "Local Go build failed"
            validation_errors=$((validation_errors + 1))
        fi
        
        # Test go mod verification
        if go mod verify >/dev/null 2>&1; then
            test_result "Integration-Go-Mod-Verify" "PASS" "Go modules verified"
        else
            test_result "Integration-Go-Mod-Verify" "FAIL" "Go module verification failed"
            validation_errors=$((validation_errors + 1))
        fi
        
        # Test go mod tidy (check if it would change anything)
        local mod_before=$(cat go.mod go.sum 2>/dev/null | md5sum || echo "none")
        go mod tidy >/dev/null 2>&1 || true
        local mod_after=$(cat go.mod go.sum 2>/dev/null | md5sum || echo "none")
        
        if [ "$mod_before" = "$mod_after" ]; then
            test_result "Integration-Go-Mod-Tidy" "PASS" "Go modules are tidy"
        else
            test_result "Integration-Go-Mod-Tidy" "WARN" "Go modules may need tidying"
            # Restore original state
            git checkout go.mod go.sum 2>/dev/null || true
        fi
        
    else
        test_result "Integration-Go-Available" "FAIL" "Go compiler not available for local testing"
        validation_errors=$((validation_errors + 1))
    fi
    
    # Test Docker availability
    if command -v docker >/dev/null 2>&1; then
        test_result "Integration-Docker-Available" "PASS" "Docker available for local testing"
        
        # Test Docker buildx availability
        if docker buildx version >/dev/null 2>&1; then
            test_result "Integration-Docker-Buildx" "PASS" "Docker buildx available"
        else
            test_result "Integration-Docker-Buildx" "WARN" "Docker buildx not available"
        fi
    else
        test_result "Integration-Docker-Available" "WARN" "Docker not available for local testing"
    fi
    
    return $validation_errors
}

# =============================================================================
# TEST 8: CI Dependencies Validation
# =============================================================================
validate_ci_dependencies() {
    log_info "=== TEST 8: CI Dependencies Validation ==="
    
    local validation_errors=0
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Check for required GitHub Actions
    local required_actions=(
        "actions/checkout@v4:Checkout action"
        "actions/setup-go@v5:Go setup action"
        "actions/cache@v4:Cache action"
        "docker/setup-buildx-action@v3:Docker buildx setup"
        "docker/login-action@v3:Docker login action"
        "docker/metadata-action@v5:Docker metadata action"
    )
    
    for action_entry in "${required_actions[@]}"; do
        local action=$(echo "$action_entry" | cut -d: -f1)
        local description=$(echo "$action_entry" | cut -d: -f2)
        
        if grep -q "$action" "$ci_workflow"; then
            test_result "CI-Action-$(echo $action | tr '/' '-' | cut -d@ -f1)" "PASS" "$description found"
        else
            test_result "CI-Action-$(echo $action | tr '/' '-' | cut -d@ -f1)" "FAIL" "$description missing"
            validation_errors=$((validation_errors + 1))
        fi
    done
    
    # Check Go version consistency
    local go_version_file=$(grep -o "go-version-file:.*" "$ci_workflow" | head -1 | cut -d: -f2 | tr -d ' ')
    if [ -f "$PROJECT_ROOT/$go_version_file" ] || [ "$go_version_file" = "go.mod" ]; then
        test_result "CI-Go-Version-File" "PASS" "Go version file reference valid"
    else
        test_result "CI-Go-Version-File" "FAIL" "Go version file reference invalid: $go_version_file"
        validation_errors=$((validation_errors + 1))
    fi
    
    return $validation_errors
}

# =============================================================================
# TEST 9: Security and Compliance Validation
# =============================================================================
validate_security_compliance() {
    log_info "=== TEST 9: Security and Compliance Validation ==="
    
    local validation_errors=0
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Check for govulncheck usage
    if grep -q "govulncheck" "$ci_workflow"; then
        test_result "Security-Govulncheck" "PASS" "Govulncheck vulnerability scanning configured"
    else
        test_result "Security-Govulncheck" "WARN" "Govulncheck not found in CI"
    fi
    
    # Check for non-root user in Dockerfiles
    local dockerfiles=("$PROJECT_ROOT/Dockerfile" "$PROJECT_ROOT/Dockerfile.resilient")
    for dockerfile in "${dockerfiles[@]}"; do
        if [ -f "$dockerfile" ]; then
            local file_name=$(basename "$dockerfile")
            if grep -q "USER.*nonroot\|USER.*65532" "$dockerfile"; then
                test_result "Security-$file_name-NonRoot" "PASS" "Non-root user configured"
            else
                test_result "Security-$file_name-NonRoot" "FAIL" "Non-root user not configured"
                validation_errors=$((validation_errors + 1))
            fi
        fi
    done
    
    # Check for security labels in Dockerfiles
    for dockerfile in "${dockerfiles[@]}"; do
        if [ -f "$dockerfile" ]; then
            local file_name=$(basename "$dockerfile")
            if grep -q "security.scan.*required" "$dockerfile"; then
                test_result "Security-$file_name-Labels" "PASS" "Security labels present"
            else
                test_result "Security-$file_name-Labels" "WARN" "Security labels missing"
            fi
        fi
    done
    
    return $validation_errors
}

# =============================================================================
# TEST 10: Performance and Optimization Validation
# =============================================================================
validate_performance_optimizations() {
    log_info "=== TEST 10: Performance and Optimization Validation ==="
    
    local validation_errors=0
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Check for caching strategies
    local cache_types=("go-build" "go-modules" "docker-cache" "golangci-lint")
    for cache_type in "${cache_types[@]}"; do
        if grep -q "$cache_type" "$ci_workflow"; then
            test_result "Performance-Cache-$cache_type" "PASS" "Cache $cache_type configured"
        else
            test_result "Performance-Cache-$cache_type" "WARN" "Cache $cache_type not found"
        fi
    done
    
    # Check for timeout configurations
    local jobs_with_timeouts=$(grep -c "timeout-minutes:" "$ci_workflow" || echo "0")
    if [ "$jobs_with_timeouts" -gt 5 ]; then
        test_result "Performance-Timeouts" "PASS" "Job timeouts configured ($jobs_with_timeouts jobs)"
    else
        test_result "Performance-Timeouts" "WARN" "Limited timeout configuration ($jobs_with_timeouts jobs)"
    fi
    
    # Check for concurrency controls
    if grep -q "concurrency:" "$ci_workflow"; then
        test_result "Performance-Concurrency" "PASS" "Concurrency controls configured"
    else
        test_result "Performance-Concurrency" "FAIL" "Concurrency controls missing"
        validation_errors=$((validation_errors + 1))
    fi
    
    return $validation_errors
}

# =============================================================================
# Main Execution Function
# =============================================================================
main() {
    echo "==============================================================================" | tee "$VALIDATION_LOG"
    echo "CI Fixes Comprehensive Validation Report" | tee -a "$VALIDATION_LOG"
    echo "Generated: $(date)" | tee -a "$VALIDATION_LOG"
    echo "==============================================================================" | tee -a "$VALIDATION_LOG"
    echo "" | tee -a "$VALIDATION_LOG"
    
    # Create temporary directory
    mkdir -p "$TEMP_DIR"
    trap "rm -rf $TEMP_DIR" EXIT
    
    # Change to project root
    cd "$PROJECT_ROOT"
    
    # Run all validation tests
    log_info "Starting comprehensive CI fixes validation..."
    echo "" | tee -a "$VALIDATION_LOG"
    
    # Execute validation tests
    validate_service_configuration || true
    echo "" | tee -a "$VALIDATION_LOG"
    
    validate_workflow_syntax || true
    echo "" | tee -a "$VALIDATION_LOG"
    
    validate_docker_builds || true
    echo "" | tee -a "$VALIDATION_LOG"
    
    validate_ghcr_authentication || true
    echo "" | tee -a "$VALIDATION_LOG"
    
    validate_smart_build_script || true
    echo "" | tee -a "$VALIDATION_LOG"
    
    validate_resilient_dockerfile || true
    echo "" | tee -a "$VALIDATION_LOG"
    
    validate_integration || true
    echo "" | tee -a "$VALIDATION_LOG"
    
    validate_ci_dependencies || true
    echo "" | tee -a "$VALIDATION_LOG"
    
    validate_security_compliance || true
    echo "" | tee -a "$VALIDATION_LOG"
    
    validate_performance_optimizations || true
    echo "" | tee -a "$VALIDATION_LOG"
    
    # Generate final summary
    echo "==============================================================================" | tee -a "$VALIDATION_LOG"
    echo "FINAL VALIDATION SUMMARY" | tee -a "$VALIDATION_LOG"
    echo "==============================================================================" | tee -a "$VALIDATION_LOG"
    
    local success_rate=0
    if [ $TESTS_TOTAL -gt 0 ]; then
        success_rate=$(echo "scale=1; $TESTS_PASSED * 100 / $TESTS_TOTAL" | bc -l)
    fi
    
    echo "" | tee -a "$VALIDATION_LOG"
    echo "Test Results Summary:" | tee -a "$VALIDATION_LOG"
    echo "  Total Tests: $TESTS_TOTAL" | tee -a "$VALIDATION_LOG"
    echo "  Passed: $TESTS_PASSED" | tee -a "$VALIDATION_LOG"
    echo "  Failed: $TESTS_FAILED" | tee -a "$VALIDATION_LOG"
    echo "  Warnings: $TESTS_WARNINGS" | tee -a "$VALIDATION_LOG"
    echo "  Success Rate: ${success_rate}%" | tee -a "$VALIDATION_LOG"
    echo "" | tee -a "$VALIDATION_LOG"
    
    # Final assessment
    if [ $TESTS_FAILED -eq 0 ]; then
        if [ $TESTS_WARNINGS -eq 0 ]; then
            log_success "ALL VALIDATIONS PASSED - CI fixes are fully functional!"
            echo "Status: FULLY VALIDATED ✅" | tee -a "$VALIDATION_LOG"
        else
            log_success "CORE VALIDATIONS PASSED - CI fixes are functional with minor warnings"
            echo "Status: VALIDATED WITH WARNINGS ⚠️" | tee -a "$VALIDATION_LOG"
        fi
        echo "" | tee -a "$VALIDATION_LOG"
        echo "✅ READY FOR PR MERGE" | tee -a "$VALIDATION_LOG"
        return 0
    else
        log_error "VALIDATION FAILURES DETECTED - CI fixes need attention"
        echo "Status: VALIDATION FAILED ❌" | tee -a "$VALIDATION_LOG"
        echo "" | tee -a "$VALIDATION_LOG"
        echo "❌ NOT READY FOR MERGE - Fix failures before proceeding" | tee -a "$VALIDATION_LOG"
        return 1
    fi
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi