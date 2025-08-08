#!/bin/bash

# Test script for consolidated Dockerfiles
# Validates all deployment scenarios work with new consolidated Dockerfiles
# Usage: ./test-consolidated-dockerfiles.sh

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/.."
TEST_VERSION="test-$(date +%s)"
REGISTRY="localhost:5000"
FAILED_TESTS=()

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Test result tracking
test_passed() {
    log_success "$1 - PASSED"
}

test_failed() {
    log_error "$1 - FAILED"
    FAILED_TESTS+=("$1")
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test environment..."
    
    # Stop any running containers
    docker-compose -f "${PROJECT_ROOT}/deployments/docker-compose.consolidated.yml" down --remove-orphans || true
    
    # Remove test images
    docker images --filter "reference=*:${TEST_VERSION}" -q | xargs -r docker rmi -f || true
    
    # Remove builder if exists
    docker buildx rm nephoran-test-builder 2>/dev/null || true
    
    log_info "Cleanup completed"
}

trap cleanup EXIT

# Prerequisites check
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi
    
    # Check BuildKit
    if [[ "${DOCKER_BUILDKIT:-0}" != "1" ]]; then
        export DOCKER_BUILDKIT=1
        log_warning "DOCKER_BUILDKIT enabled for tests"
    fi
    
    # Check if consolidated Dockerfiles exist
    for dockerfile in Dockerfile.production Dockerfile.dev Dockerfile.multiarch; do
        if [[ ! -f "${PROJECT_ROOT}/${dockerfile}" ]]; then
            log_error "Missing consolidated Dockerfile: ${dockerfile}"
            exit 1
        fi
    done
    
    log_success "Prerequisites check completed"
}

# Test 1: Production Dockerfile builds
test_production_builds() {
    log_info "Testing production Dockerfile builds..."
    
    local services=("llm-processor" "nephio-bridge" "oran-adaptor" "rag-api")
    local build_success=true
    
    for service in "${services[@]}"; do
        log_info "Building ${service} with production Dockerfile..."
        
        if docker build -f "${PROJECT_ROOT}/Dockerfile.production" \
            --build-arg SERVICE_NAME="${service}" \
            --build-arg SERVICE_TYPE=$(get_service_type "$service") \
            --build-arg VERSION="${TEST_VERSION}" \
            --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
            --build-arg VCS_REF="test" \
            --target final \
            -t "test-${service}:${TEST_VERSION}" \
            "${PROJECT_ROOT}" &> "/tmp/build-${service}.log"; then
            log_success "Built ${service} successfully"
        else
            log_error "Failed to build ${service}"
            build_success=false
        fi
    done
    
    if $build_success; then
        test_passed "Production Dockerfile builds"
    else
        test_failed "Production Dockerfile builds"
    fi
}

# Test 2: Development Dockerfile builds
test_development_builds() {
    log_info "Testing development Dockerfile builds..."
    
    local services=("llm-processor" "rag-api")
    local build_success=true
    
    for service in "${services[@]}"; do
        log_info "Building ${service} with development Dockerfile..."
        
        if docker build -f "${PROJECT_ROOT}/Dockerfile.dev" \
            --build-arg SERVICE_NAME="${service}" \
            --build-arg SERVICE_TYPE=$(get_service_type "$service") \
            --build-arg VERSION="${TEST_VERSION}" \
            --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
            --build-arg VCS_REF="test" \
            --target final \
            -t "test-${service}-dev:${TEST_VERSION}" \
            "${PROJECT_ROOT}" &> "/tmp/build-${service}-dev.log"; then
            log_success "Built ${service} development image successfully"
        else
            log_error "Failed to build ${service} development image"
            build_success=false
        fi
    done
    
    if $build_success; then
        test_passed "Development Dockerfile builds"
    else
        test_failed "Development Dockerfile builds"
    fi
}

# Test 3: Multi-architecture builds (if buildx available)
test_multiarch_builds() {
    log_info "Testing multi-architecture Dockerfile builds..."
    
    # Check if buildx is available
    if ! docker buildx version &> /dev/null; then
        log_warning "Docker buildx not available, skipping multi-arch tests"
        return
    fi
    
    # Create test builder
    docker buildx create --name nephoran-test-builder --use &> /dev/null || true
    
    local services=("llm-processor" "rag-api")
    local build_success=true
    
    for service in "${services[@]}"; do
        log_info "Building ${service} with multi-arch Dockerfile..."
        
        if docker buildx build -f "${PROJECT_ROOT}/Dockerfile.multiarch" \
            --platform linux/amd64 \
            --build-arg SERVICE_NAME="${service}" \
            --build-arg SERVICE_TYPE=$(get_service_type "$service") \
            --build-arg VERSION="${TEST_VERSION}" \
            --build-arg BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ') \
            --build-arg VCS_REF="test" \
            --target final \
            -t "test-${service}-multiarch:${TEST_VERSION}" \
            "${PROJECT_ROOT}" \
            --load &> "/tmp/build-${service}-multiarch.log"; then
            log_success "Built ${service} multi-arch image successfully"
        else
            log_error "Failed to build ${service} multi-arch image"
            build_success=false
        fi
    done
    
    if $build_success; then
        test_passed "Multi-architecture Dockerfile builds"
    else
        test_failed "Multi-architecture Dockerfile builds"
    fi
}

# Test 4: Image security and compliance
test_image_security() {
    log_info "Testing image security and compliance..."
    
    local image="test-llm-processor:${TEST_VERSION}"
    local security_success=true
    
    # Test non-root user
    if docker run --rm "$image" whoami 2>/dev/null | grep -q "nonroot"; then
        log_success "Image runs as non-root user"
    else
        log_error "Image does not run as non-root user"
        security_success=false
    fi
    
    # Test static binary
    if docker run --rm "$image" /service --version 2>/dev/null | grep -q "test"; then
        log_success "Service binary is functional"
    else
        log_error "Service binary is not functional"
        security_success=false
    fi
    
    # Test distroless base (should have minimal packages)
    local package_count=$(docker run --rm "$image" sh -c 'ls /bin 2>/dev/null | wc -l' 2>/dev/null || echo "0")
    if [[ "$package_count" -lt 10 ]]; then
        log_success "Image uses minimal base (distroless)"
    else
        log_warning "Image may not be using distroless base"
    fi
    
    if $security_success; then
        test_passed "Image security and compliance"
    else
        test_failed "Image security and compliance"
    fi
}

# Test 5: Docker Compose integration
test_docker_compose() {
    log_info "Testing Docker Compose integration..."
    
    local compose_file="${PROJECT_ROOT}/deployments/docker-compose.consolidated.yml"
    local compose_success=true
    
    # Set test environment
    export VERSION="${TEST_VERSION}"
    export BUILD_DATE=$(date -u +'%Y-%m-%dT%H:%M:%SZ')
    export VCS_REF="test"
    
    # Test compose file validation
    if docker-compose -f "$compose_file" config &> /dev/null; then
        log_success "Docker Compose configuration is valid"
    else
        log_error "Docker Compose configuration is invalid"
        compose_success=false
    fi
    
    # Test build with compose
    if docker-compose -f "$compose_file" build llm-processor &> /tmp/compose-build.log; then
        log_success "Docker Compose build succeeded"
    else
        log_error "Docker Compose build failed"
        compose_success=false
    fi
    
    if $compose_success; then
        test_passed "Docker Compose integration"
    else
        test_failed "Docker Compose integration"
    fi
}

# Test 6: Build script integration
test_build_script() {
    log_info "Testing consolidated build script..."
    
    local build_script="${SCRIPT_DIR}/docker-build-consolidated.sh"
    local script_success=true
    
    # Check if script exists and is executable
    if [[ -x "$build_script" ]]; then
        log_success "Build script is executable"
    else
        log_error "Build script is not executable"
        script_success=false
    fi
    
    # Test script help
    if "$build_script" --help &> /dev/null; then
        log_success "Build script help works"
    else
        log_error "Build script help fails"
        script_success=false
    fi
    
    # Test building a single service
    export VERSION="${TEST_VERSION}"
    if "$build_script" llm-processor --build-type production &> /tmp/script-build.log; then
        log_success "Build script service build succeeded"
    else
        log_error "Build script service build failed"
        script_success=false
    fi
    
    if $script_success; then
        test_passed "Build script integration"
    else
        test_failed "Build script integration"
    fi
}

# Helper function to get service type
get_service_type() {
    local service_name="$1"
    case "$service_name" in
        "rag-api") echo "python" ;;
        "security-scanner") echo "scanner" ;;
        *) echo "go" ;;
    esac
}

# Main test execution
main() {
    log_info "Starting Nephoran Intent Operator Consolidated Dockerfiles Test Suite"
    log_info "Test version: ${TEST_VERSION}"
    echo
    
    check_prerequisites
    echo
    
    # Run all tests
    test_production_builds
    echo
    
    test_development_builds
    echo
    
    test_multiarch_builds
    echo
    
    test_image_security
    echo
    
    test_docker_compose
    echo
    
    test_build_script
    echo
    
    # Display results
    log_info "Test Results Summary"
    echo "===================="
    
    if [[ ${#FAILED_TESTS[@]} -eq 0 ]]; then
        log_success "All tests PASSED! ✅"
        echo
        log_info "The consolidated Dockerfiles are working correctly."
        log_info "Ready to remove redundant Dockerfile variants."
        exit 0
    else
        log_error "Some tests FAILED! ❌"
        echo
        echo "Failed tests:"
        for test in "${FAILED_TESTS[@]}"; do
            echo "  - $test"
        done
        echo
        log_error "Please fix the issues before removing redundant Dockerfiles."
        exit 1
    fi
}

# Run main function
main "$@"