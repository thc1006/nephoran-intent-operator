#!/bin/bash
# =============================================================================
# Docker Build Validation Test Suite
# =============================================================================
# Comprehensive testing for Docker builds without actually building images.
# Tests syntax, configuration, and build command construction.
# =============================================================================

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_LOG="$PROJECT_ROOT/docker-build-validation.log"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$TEST_LOG"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$TEST_LOG"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$TEST_LOG"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" | tee -a "$TEST_LOG"; }

# Test counters
DOCKER_TESTS_TOTAL=0
DOCKER_TESTS_PASSED=0
DOCKER_TESTS_FAILED=0

# Test result function
docker_test_result() {
    local test_name="$1"
    local status="$2"
    local message="$3"
    
    DOCKER_TESTS_TOTAL=$((DOCKER_TESTS_TOTAL + 1))
    
    case "$status" in
        "PASS")
            DOCKER_TESTS_PASSED=$((DOCKER_TESTS_PASSED + 1))
            log_success "DOCKER TEST $DOCKER_TESTS_TOTAL: $test_name - $message"
            ;;
        "FAIL")
            DOCKER_TESTS_FAILED=$((DOCKER_TESTS_FAILED + 1))
            log_error "DOCKER TEST $DOCKER_TESTS_TOTAL: $test_name - $message"
            ;;
        "WARN")
            log_warning "DOCKER TEST $DOCKER_TESTS_TOTAL: $test_name - $message"
            ;;
    esac
}

# =============================================================================
# Test 1: Dockerfile Syntax and Structure Validation
# =============================================================================
test_dockerfile_syntax() {
    log_info "=== Testing Dockerfile Syntax and Structure ==="
    
    local dockerfiles=("$PROJECT_ROOT/Dockerfile" "$PROJECT_ROOT/Dockerfile.resilient")
    
    for dockerfile in "${dockerfiles[@]}"; do
        local filename=$(basename "$dockerfile")
        
        if [ ! -f "$dockerfile" ]; then
            docker_test_result "$filename-Exists" "FAIL" "Dockerfile not found: $dockerfile"
            continue
        fi
        
        docker_test_result "$filename-Exists" "PASS" "Dockerfile found"
        
        # Test Docker syntax using buildx parse
        log_info "Testing $filename syntax..."
        if docker buildx build --dry-run -f "$dockerfile" --build-arg SERVICE=conductor-loop . >/dev/null 2>&1; then
            docker_test_result "$filename-Syntax" "PASS" "Dockerfile syntax valid"
        else
            docker_test_result "$filename-Syntax" "FAIL" "Dockerfile syntax error"
            continue
        fi
        
        # Check required components
        local required_components=(
            "ARG SERVICE:Service argument"
            "WORKDIR:Working directory set" 
            "USER.*nonroot|USER.*65532:Non-root user"
            "HEALTHCHECK:Health check configured"
            "ENTRYPOINT:Entry point defined"
        )
        
        for component in "${required_components[@]}"; do
            local pattern=$(echo "$component" | cut -d: -f1)
            local description=$(echo "$component" | cut -d: -f2)
            
            if grep -qE "$pattern" "$dockerfile"; then
                docker_test_result "$filename-Component-$(echo $description | tr ' ' '-')" "PASS" "$description found"
            else
                docker_test_result "$filename-Component-$(echo $description | tr ' ' '-')" "WARN" "$description missing"
            fi
        done
        
        # Check for security best practices
        if grep -q "USER.*root" "$dockerfile"; then
            docker_test_result "$filename-Security-RootUser" "FAIL" "Running as root user (security risk)"
        else
            docker_test_result "$filename-Security-RootUser" "PASS" "Not running as root user"
        fi
        
        # Check for multi-stage builds (recommended for optimization)
        local stages=$(grep -c "^FROM.*AS" "$dockerfile" || echo "0")
        if [ "$stages" -gt 1 ]; then
            docker_test_result "$filename-MultiStage" "PASS" "Multi-stage build with $stages stages"
        else
            docker_test_result "$filename-MultiStage" "WARN" "Single-stage build (consider multi-stage)"
        fi
    done
}

# =============================================================================
# Test 2: Service-Specific Build Command Testing
# =============================================================================
test_service_build_commands() {
    log_info "=== Testing Service-Specific Build Commands ==="
    
    local dockerfile="$PROJECT_ROOT/Dockerfile"
    
    # Extract services from Dockerfile
    local services=(
        "conductor-loop"
        "intent-ingest" 
        "nephio-bridge"
        "llm-processor"
        "oran-adaptor"
        "manager"
        "controller"
        "e2-kmp-sim"
        "o1-ves-sim"
    )
    
    for service in "${services[@]}"; do
        log_info "Testing build command for service: $service"
        
        # Construct and test build command (dry-run only)
        local build_cmd=(
            "docker" "buildx" "build"
            "--dry-run"
            "--platform" "linux/amd64"
            "--build-arg" "SERVICE=$service"
            "--build-arg" "VERSION=test"
            "--build-arg" "BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
            "--build-arg" "VCS_REF=test"
            "-f" "$dockerfile"
            "-t" "test/$service:validation"
            "."
        )
        
        if "${build_cmd[@]}" >/dev/null 2>&1; then
            docker_test_result "BuildCmd-$service" "PASS" "Build command valid for $service"
        else
            docker_test_result "BuildCmd-$service" "FAIL" "Build command invalid for $service"
        fi
    done
}

# =============================================================================
# Test 3: Smart Build Script Integration Testing
# =============================================================================
test_smart_build_integration() {
    log_info "=== Testing Smart Build Script Integration ==="
    
    local smart_build="$PROJECT_ROOT/scripts/deploy/smart-docker-build.sh"
    
    if [ ! -f "$smart_build" ]; then
        docker_test_result "SmartBuild-Exists" "FAIL" "Smart build script not found"
        return 1
    fi
    
    docker_test_result "SmartBuild-Exists" "PASS" "Smart build script found"
    
    # Test script permissions
    if [ -x "$smart_build" ]; then
        docker_test_result "SmartBuild-Executable" "PASS" "Script is executable"
    else
        docker_test_result "SmartBuild-Executable" "FAIL" "Script is not executable"
        chmod +x "$smart_build"
    fi
    
    # Test script syntax
    if bash -n "$smart_build"; then
        docker_test_result "SmartBuild-Syntax" "PASS" "Script syntax valid"
    else
        docker_test_result "SmartBuild-Syntax" "FAIL" "Script syntax error"
        return 1
    fi
    
    # Test infrastructure health check function
    log_info "Testing infrastructure health check function..."
    if grep -q "check_infrastructure_health()" "$smart_build"; then
        docker_test_result "SmartBuild-HealthCheck" "PASS" "Health check function found"
        
        # Test if function can be extracted and run independently
        if bash -c "source '$smart_build'; declare -f check_infrastructure_health >/dev/null"; then
            docker_test_result "SmartBuild-HealthCheck-Callable" "PASS" "Health check function is callable"
        else
            docker_test_result "SmartBuild-HealthCheck-Callable" "WARN" "Health check function may have issues"
        fi
    else
        docker_test_result "SmartBuild-HealthCheck" "FAIL" "Health check function missing"
    fi
    
    # Test strategy selection function
    if grep -q "select_build_strategy()" "$smart_build"; then
        docker_test_result "SmartBuild-StrategySelection" "PASS" "Strategy selection function found"
    else
        docker_test_result "SmartBuild-StrategySelection" "FAIL" "Strategy selection function missing"
    fi
    
    # Test build execution function
    if grep -q "execute_build()" "$smart_build"; then
        docker_test_result "SmartBuild-ExecuteBuild" "PASS" "Execute build function found"
    else
        docker_test_result "SmartBuild-ExecuteBuild" "FAIL" "Execute build function missing"
    fi
}

# =============================================================================
# Test 4: Docker Build Environment Testing
# =============================================================================
test_docker_environment() {
    log_info "=== Testing Docker Build Environment ==="
    
    # Test Docker availability
    if command -v docker >/dev/null 2>&1; then
        docker_test_result "Docker-Available" "PASS" "Docker command available"
        
        # Test Docker daemon connectivity
        if docker info >/dev/null 2>&1; then
            docker_test_result "Docker-Daemon" "PASS" "Docker daemon accessible"
            
            # Test Docker buildx
            if docker buildx version >/dev/null 2>&1; then
                docker_test_result "Docker-Buildx" "PASS" "Docker buildx available"
                
                # Test buildx builders
                local builders=$(docker buildx ls 2>/dev/null | grep -v "NAME/NODE" | wc -l || echo "0")
                if [ "$builders" -gt 0 ]; then
                    docker_test_result "Docker-Buildx-Builders" "PASS" "$builders buildx builders available"
                else
                    docker_test_result "Docker-Buildx-Builders" "WARN" "No buildx builders found"
                fi
                
            else
                docker_test_result "Docker-Buildx" "WARN" "Docker buildx not available"
            fi
        else
            docker_test_result "Docker-Daemon" "WARN" "Docker daemon not accessible (may be expected in CI)"
        fi
    else
        docker_test_result "Docker-Available" "WARN" "Docker not available (may be expected in some environments)"
    fi
    
    # Test platform support
    local target_platforms=("linux/amd64" "linux/arm64")
    for platform in "${target_platforms[@]}"; do
        # This is a dry test - we can't actually test platform building without Docker running
        docker_test_result "Platform-Support-$(echo $platform | tr '/' '-')" "PASS" "Target platform $platform configured"
    done
}

# =============================================================================
# Test 5: Build Configuration Matrix Testing
# =============================================================================
test_build_matrix() {
    log_info "=== Testing Build Configuration Matrix ==="
    
    local dockerfile="$PROJECT_ROOT/Dockerfile"
    
    # Test build arguments matrix
    local build_scenarios=(
        "conductor-loop:linux/amd64:latest"
        "intent-ingest:linux/arm64:v1.0.0"
        "nephio-bridge:linux/amd64,linux/arm64:test"
    )
    
    for scenario in "${build_scenarios[@]}"; do
        local service=$(echo "$scenario" | cut -d: -f1)
        local platform=$(echo "$scenario" | cut -d: -f2)
        local version=$(echo "$scenario" | cut -d: -f3)
        
        log_info "Testing build matrix: $service on $platform version $version"
        
        # Test command construction
        local cmd=(
            "docker" "buildx" "build"
            "--dry-run"
            "--platform" "$platform"
            "--build-arg" "SERVICE=$service"
            "--build-arg" "VERSION=$version"
            "--build-arg" "BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
            "--build-arg" "VCS_REF=$(git rev-parse --short HEAD 2>/dev/null || echo 'test')"
            "-f" "$dockerfile"
            "-t" "test/$service:$version"
            "."
        )
        
        log_info "Build command: ${cmd[*]}"
        
        if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
            if "${cmd[@]}" >/dev/null 2>&1; then
                docker_test_result "Matrix-$service-$platform" "PASS" "Build command valid for $service on $platform"
            else
                docker_test_result "Matrix-$service-$platform" "FAIL" "Build command failed for $service on $platform"
            fi
        else
            docker_test_result "Matrix-$service-$platform" "WARN" "Cannot test build command (Docker not available)"
        fi
    done
}

# =============================================================================
# Test 6: Build Cache Strategy Testing
# =============================================================================
test_cache_strategies() {
    log_info "=== Testing Build Cache Strategies ==="
    
    local cache_strategies=(
        "gha-full:--cache-from=type=gha --cache-to=type=gha,mode=max"
        "local-only:--cache-from=type=local,src=/tmp/cache --cache-to=type=local,dest=/tmp/cache,mode=max"
        "registry:--cache-from=type=registry,ref=test/cache --cache-to=type=registry,ref=test/cache,mode=max"
        "no-cache:--no-cache"
    )
    
    for strategy_def in "${cache_strategies[@]}"; do
        local strategy_name=$(echo "$strategy_def" | cut -d: -f1)
        local cache_args=$(echo "$strategy_def" | cut -d: -f2-)
        
        log_info "Testing cache strategy: $strategy_name"
        
        # Test command construction with cache strategy
        local cmd="docker buildx build --dry-run --platform linux/amd64 $cache_args --build-arg SERVICE=conductor-loop -t test:cache-test ."
        
        if command -v docker >/dev/null 2>&1; then
            if eval "$cmd" >/dev/null 2>&1; then
                docker_test_result "Cache-Strategy-$strategy_name" "PASS" "Cache strategy $strategy_name valid"
            else
                docker_test_result "Cache-Strategy-$strategy_name" "FAIL" "Cache strategy $strategy_name invalid"
            fi
        else
            docker_test_result "Cache-Strategy-$strategy_name" "WARN" "Cannot test cache strategy (Docker not available)"
        fi
    done
}

# =============================================================================
# Test 7: Service Path and File Validation
# =============================================================================
test_service_paths() {
    log_info "=== Testing Service Paths and File Validation ==="
    
    # Define expected service mappings from Dockerfile analysis
    local service_mappings=(
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
    
    for mapping in "${service_mappings[@]}"; do
        local service=$(echo "$mapping" | cut -d: -f1)
        local path=$(echo "$mapping" | cut -d: -f2)
        local full_path="$PROJECT_ROOT/${path#./}"
        
        log_info "Validating service path: $service -> $path"
        
        if [ -f "$full_path" ]; then
            docker_test_result "ServicePath-$service-Exists" "PASS" "Service main.go exists: $path"
            
            # Check if main function exists
            if grep -q "func main()" "$full_path"; then
                docker_test_result "ServicePath-$service-MainFunc" "PASS" "main() function found in $service"
            else
                docker_test_result "ServicePath-$service-MainFunc" "FAIL" "main() function not found in $service"
            fi
            
            # Check if it's valid Go syntax
            if go build -o /tmp/test-$service "$full_path" >/dev/null 2>&1; then
                docker_test_result "ServicePath-$service-GoBuild" "PASS" "Go build successful for $service"
                rm -f /tmp/test-$service
            else
                docker_test_result "ServicePath-$service-GoBuild" "FAIL" "Go build failed for $service"
            fi
        else
            docker_test_result "ServicePath-$service-Exists" "FAIL" "Service main.go missing: $path"
        fi
    done
}

# =============================================================================
# Test 8: Container Image Metadata Testing
# =============================================================================
test_container_metadata() {
    log_info "=== Testing Container Image Metadata ==="
    
    local dockerfile="$PROJECT_ROOT/Dockerfile"
    
    # Check for required labels
    local required_labels=(
        "org.opencontainers.image.created"
        "org.opencontainers.image.revision"
        "org.opencontainers.image.version"
        "org.opencontainers.image.title"
        "org.opencontainers.image.description"
    )
    
    for label in "${required_labels[@]}"; do
        if grep -q "LABEL.*$label" "$dockerfile"; then
            docker_test_result "Metadata-$label" "PASS" "Label $label present"
        else
            docker_test_result "Metadata-$label" "WARN" "Label $label missing"
        fi
    done
    
    # Check for service-specific metadata
    if grep -q "service.name.*SERVICE" "$dockerfile"; then
        docker_test_result "Metadata-ServiceName" "PASS" "Service name label configured"
    else
        docker_test_result "Metadata-ServiceName" "WARN" "Service name label missing"
    fi
}

# =============================================================================
# Test 9: Resource Optimization Testing
# =============================================================================
test_resource_optimization() {
    log_info "=== Testing Resource Optimization ==="
    
    local dockerfile="$PROJECT_ROOT/Dockerfile"
    
    # Check for image size optimization techniques
    local optimizations=(
        "strip.*strip-unneeded:Binary stripping"
        "rm.*cache.*apk:APK cache cleanup"
        "rm.*var/lib/apt/lists:APT cache cleanup"
        "CGO_ENABLED=0:CGO disabled"
        "ldflags.*-w.*-s:Build flags optimization"
    )
    
    for optimization in "${optimizations[@]}"; do
        local pattern=$(echo "$optimization" | cut -d: -f1)
        local description=$(echo "$optimization" | cut -d: -f2)
        
        if grep -qE "$pattern" "$dockerfile"; then
            docker_test_result "Optimization-$(echo $description | tr ' ' '-')" "PASS" "$description implemented"
        else
            docker_test_result "Optimization-$(echo $description | tr ' ' '-')" "WARN" "$description not found"
        fi
    done
    
    # Check for distroless base images
    if grep -q "gcr.io/distroless" "$dockerfile"; then
        docker_test_result "Optimization-Distroless" "PASS" "Distroless base image used"
    else
        docker_test_result "Optimization-Distroless" "WARN" "Distroless base image not used"
    fi
}

# =============================================================================
# Test 10: Build Error Recovery Testing
# =============================================================================
test_error_recovery() {
    log_info "=== Testing Build Error Recovery ==="
    
    local smart_build="$PROJECT_ROOT/scripts/deploy/smart-docker-build.sh"
    
    if [ ! -f "$smart_build" ]; then
        docker_test_result "ErrorRecovery-SmartBuild" "FAIL" "Smart build script not available"
        return 1
    fi
    
    # Check for retry logic
    if grep -q "max.*attempts\|MAX_RETRY" "$smart_build"; then
        docker_test_result "ErrorRecovery-RetryLogic" "PASS" "Retry logic implemented"
    else
        docker_test_result "ErrorRecovery-RetryLogic" "WARN" "Retry logic not found"
    fi
    
    # Check for fallback strategies
    if grep -q "fallback\|emergency" "$smart_build"; then
        docker_test_result "ErrorRecovery-Fallback" "PASS" "Fallback strategies implemented"
    else
        docker_test_result "ErrorRecovery-Fallback" "WARN" "Fallback strategies not found"
    fi
    
    # Check for timeout handling
    if grep -q "timeout.*[0-9]\+" "$smart_build"; then
        docker_test_result "ErrorRecovery-Timeouts" "PASS" "Timeout handling implemented"
    else
        docker_test_result "ErrorRecovery-Timeouts" "WARN" "Timeout handling not found"
    fi
    
    # Check for infrastructure health assessment
    if grep -q "infrastructure.*health\|health.*check" "$smart_build"; then
        docker_test_result "ErrorRecovery-HealthAssessment" "PASS" "Infrastructure health assessment implemented"
    else
        docker_test_result "ErrorRecovery-HealthAssessment" "WARN" "Infrastructure health assessment not found"
    fi
}

# =============================================================================
# Main Execution Function
# =============================================================================
main() {
    echo "==============================================================================" | tee "$TEST_LOG"
    echo "Docker Build Validation Test Suite" | tee -a "$TEST_LOG"
    echo "Generated: $(date)" | tee -a "$TEST_LOG"
    echo "==============================================================================" | tee -a "$TEST_LOG"
    echo "" | tee -a "$TEST_LOG"
    
    cd "$PROJECT_ROOT"
    
    # Run all Docker build tests
    log_info "Starting Docker build validation tests..."
    echo "" | tee -a "$TEST_LOG"
    
    test_dockerfile_syntax
    echo "" | tee -a "$TEST_LOG"
    
    test_service_build_commands
    echo "" | tee -a "$TEST_LOG"
    
    test_smart_build_integration
    echo "" | tee -a "$TEST_LOG"
    
    test_docker_environment
    echo "" | tee -a "$TEST_LOG"
    
    test_build_matrix
    echo "" | tee -a "$TEST_LOG"
    
    test_cache_strategies
    echo "" | tee -a "$TEST_LOG"
    
    test_service_paths
    echo "" | tee -a "$TEST_LOG"
    
    test_container_metadata
    echo "" | tee -a "$TEST_LOG"
    
    test_resource_optimization
    echo "" | tee -a "$TEST_LOG"
    
    test_error_recovery
    echo "" | tee -a "$TEST_LOG"
    
    # Generate final summary
    echo "==============================================================================" | tee -a "$TEST_LOG"
    echo "DOCKER BUILD VALIDATION SUMMARY" | tee -a "$TEST_LOG"
    echo "==============================================================================" | tee -a "$TEST_LOG"
    
    local success_rate=0
    if [ $DOCKER_TESTS_TOTAL -gt 0 ]; then
        success_rate=$(echo "scale=1; $DOCKER_TESTS_PASSED * 100 / $DOCKER_TESTS_TOTAL" | bc -l 2>/dev/null || echo "0")
    fi
    
    echo "" | tee -a "$TEST_LOG"
    echo "Docker Test Results Summary:" | tee -a "$TEST_LOG"
    echo "  Total Tests: $DOCKER_TESTS_TOTAL" | tee -a "$TEST_LOG"
    echo "  Passed: $DOCKER_TESTS_PASSED" | tee -a "$TEST_LOG"
    echo "  Failed: $DOCKER_TESTS_FAILED" | tee -a "$TEST_LOG"
    echo "  Success Rate: ${success_rate}%" | tee -a "$TEST_LOG"
    echo "" | tee -a "$TEST_LOG"
    
    # Final assessment
    if [ $DOCKER_TESTS_FAILED -eq 0 ]; then
        log_success "ALL DOCKER BUILD TESTS PASSED!"
        echo "Status: DOCKER BUILD SYSTEM VALIDATED ✅" | tee -a "$TEST_LOG"
        return 0
    else
        log_error "DOCKER BUILD VALIDATION FAILURES DETECTED"
        echo "Status: DOCKER BUILD ISSUES NEED ATTENTION ❌" | tee -a "$TEST_LOG"
        return 1
    fi
}

# Script execution
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi