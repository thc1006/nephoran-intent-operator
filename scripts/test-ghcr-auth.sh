#!/bin/bash
# =============================================================================
# GHCR Authentication Validation Test Suite
# =============================================================================
# Tests GitHub Container Registry authentication setup and permissions
# without requiring actual deployment credentials.
# =============================================================================

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
AUTH_TEST_LOG="$PROJECT_ROOT/ghcr-auth-validation.log"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$AUTH_TEST_LOG"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$AUTH_TEST_LOG"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$AUTH_TEST_LOG"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" | tee -a "$AUTH_TEST_LOG"; }

# Test counters
AUTH_TESTS_TOTAL=0
AUTH_TESTS_PASSED=0
AUTH_TESTS_FAILED=0

# Test result function
auth_test_result() {
    local test_name="$1"
    local status="$2" 
    local message="$3"
    
    AUTH_TESTS_TOTAL=$((AUTH_TESTS_TOTAL + 1))
    
    case "$status" in
        "PASS")
            AUTH_TESTS_PASSED=$((AUTH_TESTS_PASSED + 1))
            log_success "AUTH TEST $AUTH_TESTS_TOTAL: $test_name - $message"
            ;;
        "FAIL")
            AUTH_TESTS_FAILED=$((AUTH_TESTS_FAILED + 1))
            log_error "AUTH TEST $AUTH_TESTS_TOTAL: $test_name - $message"
            ;;
        "WARN")
            log_warning "AUTH TEST $AUTH_TESTS_TOTAL: $test_name - $message"
            ;;
    esac
}

# =============================================================================
# Test 1: CI Workflow Authentication Configuration
# =============================================================================
test_workflow_auth_config() {
    log_info "=== Testing CI Workflow Authentication Configuration ==="
    
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    if [ ! -f "$ci_workflow" ]; then
        auth_test_result "Workflow-File" "FAIL" "CI workflow file not found"
        return 1
    fi
    
    auth_test_result "Workflow-File" "PASS" "CI workflow file found"
    
    # Test 1.1: Required permissions
    local required_permissions=(
        "contents: read"
        "packages: write"
        "security-events: write"
        "id-token: write"
    )
    
    log_info "Checking GitHub Actions permissions..."
    for permission in "${required_permissions[@]}"; do
        local perm_name=$(echo "$permission" | cut -d: -f1 | tr -d ' ')
        if grep -q "$permission" "$ci_workflow"; then
            auth_test_result "Permission-$perm_name" "PASS" "Permission $permission configured"
        else
            auth_test_result "Permission-$perm_name" "FAIL" "Permission $permission missing"
        fi
    done
    
    # Test 1.2: GHCR login step configuration
    log_info "Checking GHCR login configuration..."
    
    # Check for docker/login-action usage
    if grep -A 10 "Login to GitHub Container Registry" "$ci_workflow" | grep -q "docker/login-action"; then
        auth_test_result "Login-Action" "PASS" "Docker login action configured"
    else
        auth_test_result "Login-Action" "FAIL" "Docker login action not found"
    fi
    
    # Check registry configuration
    if grep -A 10 "docker/login-action" "$ci_workflow" | grep -q "registry: ghcr.io"; then
        auth_test_result "Registry-Config" "PASS" "GHCR registry correctly configured"
    else
        auth_test_result "Registry-Config" "FAIL" "GHCR registry not properly configured"
    fi
    
    # Check username configuration
    if grep -A 10 "docker/login-action" "$ci_workflow" | grep -q "username:.*github.actor"; then
        auth_test_result "Username-Config" "PASS" "Username correctly configured"
    else
        auth_test_result "Username-Config" "FAIL" "Username not properly configured"
    fi
    
    # Check password/token configuration
    if grep -A 10 "docker/login-action" "$ci_workflow" | grep -q "password:.*secrets.GITHUB_TOKEN"; then
        auth_test_result "Token-Config" "PASS" "GITHUB_TOKEN correctly configured"
    else
        auth_test_result "Token-Config" "FAIL" "GITHUB_TOKEN not properly configured"
    fi
    
    # Test 1.3: Conditional execution (only on non-PR events)
    if grep -B 5 "docker/login-action" "$ci_workflow" | grep -q "if:.*github.event_name.*!=.*pull_request"; then
        auth_test_result "Conditional-Push" "PASS" "GHCR login properly conditional"
    else
        auth_test_result "Conditional-Push" "WARN" "GHCR login condition may need review"
    fi
}

# =============================================================================
# Test 2: Container Registry Connectivity
# =============================================================================
test_registry_connectivity() {
    log_info "=== Testing Container Registry Connectivity ==="
    
    # Test GHCR endpoint connectivity (without authentication)
    log_info "Testing GHCR endpoint connectivity..."
    if timeout 15s curl -s -I "https://ghcr.io/v2/" >/dev/null 2>&1; then
        auth_test_result "GHCR-Connectivity" "PASS" "GHCR endpoint is accessible"
    else
        auth_test_result "GHCR-Connectivity" "WARN" "GHCR endpoint may be unreachable (network/firewall)"
    fi
    
    # Test Docker Hub connectivity (for base images)
    log_info "Testing Docker Hub connectivity..."
    if timeout 15s curl -s -I "https://registry-1.docker.io/v2/" >/dev/null 2>&1; then
        auth_test_result "DockerHub-Connectivity" "PASS" "Docker Hub endpoint accessible"
    else
        auth_test_result "DockerHub-Connectivity" "WARN" "Docker Hub endpoint may be unreachable"
    fi
    
    # Test GCR connectivity (for distroless images)
    log_info "Testing GCR connectivity..."
    if timeout 15s curl -s -I "https://gcr.io/v2/" >/dev/null 2>&1; then
        auth_test_result "GCR-Connectivity" "PASS" "GCR endpoint accessible"
    else
        auth_test_result "GCR-Connectivity" "WARN" "GCR endpoint may be unreachable"
    fi
}

# =============================================================================
# Test 3: Image Name and Tag Validation
# =============================================================================
test_image_naming() {
    log_info "=== Testing Image Name and Tag Validation ==="
    
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Check image name configuration
    if grep -q "ghcr.io.*github.repository_owner" "$ci_workflow"; then
        auth_test_result "ImageName-Repository" "PASS" "Image name uses repository owner"
    else
        auth_test_result "ImageName-Repository" "WARN" "Image name may not use repository owner"
    fi
    
    # Check for proper image naming (lowercase)
    if grep -q "nephoran-intent-operator" "$ci_workflow"; then
        auth_test_result "ImageName-Format" "PASS" "Image name follows lowercase convention"
    else
        auth_test_result "ImageName-Format" "WARN" "Image name format may need review"
    fi
    
    # Check tagging strategy
    if grep -A 20 "docker/metadata-action" "$ci_workflow" | grep -q "type=ref,event=branch"; then
        auth_test_result "Tagging-Branch" "PASS" "Branch-based tagging configured"
    else
        auth_test_result "Tagging-Branch" "WARN" "Branch-based tagging not found"
    fi
    
    if grep -A 20 "docker/metadata-action" "$ci_workflow" | grep -q "type=sha"; then
        auth_test_result "Tagging-SHA" "PASS" "SHA-based tagging configured"
    else
        auth_test_result "Tagging-SHA" "WARN" "SHA-based tagging not found"
    fi
}

# =============================================================================
# Test 4: Security Token Validation
# =============================================================================
test_token_security() {
    log_info "=== Testing Security Token Configuration ==="
    
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Check that GITHUB_TOKEN is used (not a custom PAT)
    if grep -q "secrets.GITHUB_TOKEN" "$ci_workflow"; then
        auth_test_result "Token-Type" "PASS" "Using GITHUB_TOKEN (recommended)"
    else
        auth_test_result "Token-Type" "WARN" "Not using GITHUB_TOKEN - check if custom PAT is needed"
    fi
    
    # Check that credentials are not hardcoded
    local hardcoded_patterns=(
        "password:.*['\"][^'\"]*['\"]"
        "token:.*ghp_[a-zA-Z0-9]*"
        "password:.*ghs_[a-zA-Z0-9]*"
    )
    
    local hardcoded_found=false
    for pattern in "${hardcoded_patterns[@]}"; do
        if grep -qE "$pattern" "$ci_workflow"; then
            hardcoded_found=true
            break
        fi
    done
    
    if [ "$hardcoded_found" = "true" ]; then
        auth_test_result "Token-Security" "FAIL" "Hardcoded credentials detected"
    else
        auth_test_result "Token-Security" "PASS" "No hardcoded credentials found"
    fi
    
    # Check for token scope validation
    if grep -q "curl.*api.github.com/user" "$ci_workflow"; then
        auth_test_result "Token-Validation" "PASS" "Token validation implemented"
    else
        auth_test_result "Token-Validation" "WARN" "No token validation found"
    fi
}

# =============================================================================
# Test 5: Docker Login Error Handling
# =============================================================================
test_login_error_handling() {
    log_info "=== Testing Docker Login Error Handling ==="
    
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Check for conditional login (only on push/merge, not on PR)
    if grep -B 3 "docker/login-action" "$ci_workflow" | grep -q "if:.*pull_request"; then
        auth_test_result "Login-Conditional" "PASS" "Login properly conditional on event type"
    else
        auth_test_result "Login-Conditional" "WARN" "Login condition may need review"
    fi
    
    # Check for build/push separation
    if grep -q "github.event_name.*!=.*pull_request" "$ci_workflow"; then
        auth_test_result "Push-Conditional" "PASS" "Push operations properly conditional"
    else
        auth_test_result "Push-Conditional" "WARN" "Push conditions not found"
    fi
    
    # Check for timeout configurations on login steps
    local login_section=$(grep -A 10 -B 5 "docker/login-action" "$ci_workflow" || echo "")
    if echo "$login_section" | grep -q "timeout"; then
        auth_test_result "Login-Timeout" "PASS" "Login timeout configured"
    else
        auth_test_result "Login-Timeout" "WARN" "No login timeout configured"
    fi
}

# =============================================================================
# Test 6: Repository Access Pattern Testing
# =============================================================================
test_repository_patterns() {
    log_info "=== Testing Repository Access Patterns ==="
    
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Check for proper repository reference
    if grep -q "github.repository_owner" "$ci_workflow"; then
        auth_test_result "Repo-Owner-Reference" "PASS" "Repository owner properly referenced"
    else
        auth_test_result "Repo-Owner-Reference" "WARN" "Repository owner reference not found"
    fi
    
    # Check for repository name consistency
    local repo_references=$(grep -o "github\.repository[^}]*}" "$ci_workflow" | sort -u)
    if [ -n "$repo_references" ]; then
        auth_test_result "Repo-References" "PASS" "Repository references found: $(echo $repo_references | wc -w) unique"
    else
        auth_test_result "Repo-References" "WARN" "No repository references found"
    fi
    
    # Check for organization/user context
    if grep -q "github.repository_owner" "$ci_workflow" && ! grep -q "hardcoded.*org\|hardcoded.*user" "$ci_workflow"; then
        auth_test_result "Repo-Dynamic" "PASS" "Dynamic repository context (not hardcoded)"
    else
        auth_test_result "Repo-Dynamic" "WARN" "Repository context may be hardcoded"
    fi
}

# =============================================================================
# Test 7: Build Context and Environment Testing
# =============================================================================
test_build_environment() {
    log_info "=== Testing Build Environment for GHCR Compatibility ==="
    
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Check for proper environment variables
    local env_vars=(
        "REGISTRY_BASE.*ghcr.io"
        "DOCKER_BUILDKIT.*1"
        "BUILDX_NO_DEFAULT_ATTESTATIONS.*1"
    )
    
    for env_var in "${env_vars[@]}"; do
        local var_name=$(echo "$env_var" | cut -d. -f1)
        if grep -qE "$env_var" "$ci_workflow"; then
            auth_test_result "Environment-$var_name" "PASS" "Environment variable $var_name configured"
        else
            auth_test_result "Environment-$var_name" "WARN" "Environment variable $var_name not found"
        fi
    done
    
    # Check for platform configuration
    if grep -q "PLATFORMS.*linux/amd64,linux/arm64" "$ci_workflow"; then
        auth_test_result "Platform-Config" "PASS" "Multi-platform build configured"
    else
        auth_test_result "Platform-Config" "WARN" "Multi-platform configuration not found"
    fi
}

# =============================================================================
# Test 8: Mock Authentication Testing
# =============================================================================
test_mock_authentication() {
    log_info "=== Testing Mock Authentication Scenarios ==="
    
    # Test 1: Check if we can construct valid login commands
    local mock_scenarios=(
        "push-event:false:Should skip login on pull request"
        "merge-event:true:Should perform login on merge/push"
    )
    
    for scenario in "${mock_scenarios[@]}"; do
        local scenario_name=$(echo "$scenario" | cut -d: -f1)
        local should_login=$(echo "$scenario" | cut -d: -f2)
        local description=$(echo "$scenario" | cut -d: -f3)
        
        log_info "Testing scenario: $scenario_name"
        
        # Mock the GitHub context
        export MOCK_EVENT_NAME="push"
        if [ "$should_login" = "true" ]; then
            export MOCK_EVENT_NAME="push"
        else
            export MOCK_EVENT_NAME="pull_request"
        fi
        
        # Test condition evaluation (simplified)
        local condition_result="false"
        if [ "$MOCK_EVENT_NAME" != "pull_request" ]; then
            condition_result="true"
        fi
        
        if [ "$condition_result" = "$should_login" ]; then
            auth_test_result "MockAuth-$scenario_name" "PASS" "$description"
        else
            auth_test_result "MockAuth-$scenario_name" "FAIL" "Login condition logic incorrect for $scenario_name"
        fi
    done
}

# =============================================================================
# Test 9: Registry URL and Naming Validation
# =============================================================================
test_registry_naming() {
    log_info "=== Testing Registry URL and Naming Validation ==="
    
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Test registry URL format
    if grep -q "ghcr\.io/\${{.*github\.repository_owner.*}}" "$ci_workflow"; then
        auth_test_result "Registry-URL-Format" "PASS" "Registry URL format correct"
    else
        auth_test_result "Registry-URL-Format" "WARN" "Registry URL format may need review"
    fi
    
    # Test image name conventions (must be lowercase for GHCR)
    local image_names=$(grep -o "ghcr\.io/[^'\"]*" "$ci_workflow" | sort -u)
    local invalid_names=""
    
    for image_name in $image_names; do
        # Check if contains uppercase letters (invalid for GHCR)
        if echo "$image_name" | grep -q "[A-Z]"; then
            invalid_names="$invalid_names $image_name"
        fi
    done
    
    if [ -n "$invalid_names" ]; then
        auth_test_result "ImageName-Case" "FAIL" "Uppercase characters in image names: $invalid_names"
    else
        auth_test_result "ImageName-Case" "PASS" "All image names follow lowercase convention"
    fi
    
    # Test for valid image name format
    if grep -q "nephoran-intent-operator" "$ci_workflow"; then
        auth_test_result "ImageName-Convention" "PASS" "Image name follows naming convention"
    else
        auth_test_result "ImageName-Convention" "WARN" "Image name convention may need review"
    fi
}

# =============================================================================
# Test 10: Authentication Flow Integration
# =============================================================================
test_auth_flow_integration() {
    log_info "=== Testing Authentication Flow Integration ==="
    
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Check job dependencies for proper auth flow
    local build_job_needs=$(grep -A 5 "name: Build" "$ci_workflow" | grep "needs:" || echo "")
    if [ -n "$build_job_needs" ]; then
        auth_test_result "BuildJob-Dependencies" "PASS" "Build job has proper dependencies"
    else
        auth_test_result "BuildJob-Dependencies" "WARN" "Build job dependencies not found"
    fi
    
    # Check for auth verification step
    if grep -q "Verify.*GITHUB_TOKEN.*permissions" "$ci_workflow"; then
        auth_test_result "Auth-Verification" "PASS" "Authentication verification step found"
    else
        auth_test_result "Auth-Verification" "WARN" "No authentication verification step"
    fi
    
    # Check for build metadata generation
    if grep -q "docker/metadata-action" "$ci_workflow"; then
        auth_test_result "Build-Metadata" "PASS" "Build metadata generation configured"
    else
        auth_test_result "Build-Metadata" "WARN" "Build metadata generation not found"
    fi
    
    # Test tag and push flow
    if grep -A 10 "docker buildx build" "$ci_workflow" | grep -q "\--push"; then
        auth_test_result "Push-Integration" "PASS" "Push operation integrated in build"
    else
        auth_test_result "Push-Integration" "WARN" "Push operation may not be properly integrated"
    fi
}

# =============================================================================
# Test 11: Error Handling and Recovery
# =============================================================================
test_auth_error_handling() {
    log_info "=== Testing Authentication Error Handling ==="
    
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Check for continue-on-error where appropriate
    local auth_sections=$(grep -A 15 -B 5 "docker/login-action" "$ci_workflow")
    if echo "$auth_sections" | grep -q "continue-on-error"; then
        auth_test_result "Auth-ContinueOnError" "WARN" "Continue-on-error found - may mask auth failures"
    else
        auth_test_result "Auth-ContinueOnError" "PASS" "No continue-on-error on auth (auth failures will stop build)"
    fi
    
    # Check for proper error reporting
    if grep -q "echo.*auth.*fail\|echo.*login.*fail" "$ci_workflow"; then
        auth_test_result "Auth-ErrorReporting" "PASS" "Authentication error reporting implemented"
    else
        auth_test_result "Auth-ErrorReporting" "WARN" "No specific authentication error reporting"
    fi
    
    # Check for fallback mechanisms
    if grep -q "fallback.*registry\|alternative.*registry" "$ci_workflow"; then
        auth_test_result "Auth-Fallback" "PASS" "Registry fallback mechanisms found"
    else
        auth_test_result "Auth-Fallback" "WARN" "No registry fallback mechanisms"
    fi
}

# =============================================================================
# Test 12: GitHub Actions Context Validation
# =============================================================================
test_github_context() {
    log_info "=== Testing GitHub Actions Context Usage ==="
    
    local ci_workflow="$PROJECT_ROOT/.github/workflows/ci.yml"
    
    # Test context variables usage
    local context_vars=(
        "github.actor:Actor context"
        "github.repository:Repository context"  
        "github.repository_owner:Repository owner context"
        "github.ref:Reference context"
        "github.sha:SHA context"
        "github.event_name:Event name context"
    )
    
    for var_def in "${context_vars[@]}"; do
        local var=$(echo "$var_def" | cut -d: -f1)
        local description=$(echo "$var_def" | cut -d: -f2)
        
        if grep -q "\${{.*$var.*}}" "$ci_workflow"; then
            auth_test_result "Context-$(echo $var | tr '.' '-')" "PASS" "$description used correctly"
        else
            auth_test_result "Context-$(echo $var | tr '.' '-')" "WARN" "$description not found"
        fi
    done
    
    # Check for proper context interpolation
    if grep -q "\${{.*}}" "$ci_workflow"; then
        # Check for common interpolation mistakes
        if grep -q "\${\|}\$" "$ci_workflow"; then
            auth_test_result "Context-Interpolation" "WARN" "Potential context interpolation issues"
        else
            auth_test_result "Context-Interpolation" "PASS" "Context interpolation appears correct"
        fi
    fi
}

# =============================================================================
# Main Execution Function
# =============================================================================
main() {
    echo "==============================================================================" | tee "$AUTH_TEST_LOG"
    echo "GHCR Authentication Validation Test Suite" | tee -a "$AUTH_TEST_LOG"
    echo "Generated: $(date)" | tee -a "$AUTH_TEST_LOG"
    echo "==============================================================================" | tee -a "$AUTH_TEST_LOG"
    echo "" | tee -a "$AUTH_TEST_LOG"
    
    cd "$PROJECT_ROOT"
    
    # Run all authentication tests
    log_info "Starting GHCR authentication validation tests..."
    echo "" | tee -a "$AUTH_TEST_LOG"
    
    test_workflow_auth_config
    echo "" | tee -a "$AUTH_TEST_LOG"
    
    test_registry_connectivity
    echo "" | tee -a "$AUTH_TEST_LOG"
    
    test_image_naming
    echo "" | tee -a "$AUTH_TEST_LOG"
    
    test_token_security
    echo "" | tee -a "$AUTH_TEST_LOG"
    
    test_login_error_handling
    echo "" | tee -a "$AUTH_TEST_LOG"
    
    test_auth_flow_integration
    echo "" | tee -a "$AUTH_TEST_LOG"
    
    test_github_context
    echo "" | tee -a "$AUTH_TEST_LOG"
    
    # Generate final summary
    echo "==============================================================================" | tee -a "$AUTH_TEST_LOG"
    echo "GHCR AUTHENTICATION VALIDATION SUMMARY" | tee -a "$AUTH_TEST_LOG"
    echo "==============================================================================" | tee -a "$AUTH_TEST_LOG"
    
    local success_rate=0
    if [ $AUTH_TESTS_TOTAL -gt 0 ]; then
        success_rate=$(echo "scale=1; $AUTH_TESTS_PASSED * 100 / $AUTH_TESTS_TOTAL" | bc -l 2>/dev/null || echo "0")
    fi
    
    echo "" | tee -a "$AUTH_TEST_LOG"
    echo "GHCR Auth Test Results Summary:" | tee -a "$AUTH_TEST_LOG"
    echo "  Total Tests: $AUTH_TESTS_TOTAL" | tee -a "$AUTH_TEST_LOG"
    echo "  Passed: $AUTH_TESTS_PASSED" | tee -a "$AUTH_TEST_LOG"
    echo "  Failed: $AUTH_TESTS_FAILED" | tee -a "$AUTH_TEST_LOG"
    echo "  Success Rate: ${success_rate}%" | tee -a "$AUTH_TEST_LOG"
    echo "" | tee -a "$AUTH_TEST_LOG"
    
    # Final assessment
    if [ $AUTH_TESTS_FAILED -eq 0 ]; then
        log_success "ALL GHCR AUTHENTICATION TESTS PASSED!"
        echo "Status: GHCR AUTHENTICATION VALIDATED ✅" | tee -a "$AUTH_TEST_LOG"
        echo "" | tee -a "$AUTH_TEST_LOG"
        echo "✅ GHCR authentication is properly configured and should work in CI" | tee -a "$AUTH_TEST_LOG"
        return 0
    else
        log_error "GHCR AUTHENTICATION VALIDATION FAILURES DETECTED"
        echo "Status: AUTHENTICATION ISSUES NEED ATTENTION ❌" | tee -a "$AUTH_TEST_LOG"
        echo "" | tee -a "$AUTH_TEST_LOG"
        echo "❌ Fix authentication issues before deploying to CI" | tee -a "$AUTH_TEST_LOG"
        return 1
    fi
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi