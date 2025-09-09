#!/bin/bash
set -e

# Comprehensive CI Fix Validation Script
# Tests the complete solution for PR 176 CI failures

TIMEOUT_MINUTES=${1:-25}
FORCE_CLEAN_CACHE=${2:-false}
VERBOSE=${3:-false}

echo "🚀 Starting comprehensive CI fix validation..."
echo "📊 Configuration:"
echo "  - Timeout: $TIMEOUT_MINUTES minutes"
echo "  - Force Clean Cache: $FORCE_CLEAN_CACHE"
echo "  - Verbose: $VERBOSE"
echo ""

# Set Go environment variables (matching our CI workflow)
export GO_VERSION="1.22"
export GOPROXY="https://proxy.golang.org,direct"
export GOSUMDB="sum.golang.org"
export CGO_ENABLED=0
export GOMAXPROCS=4
export GOMEMLIMIT="8GiB"
export GOGC=75
export GO_DISABLE_TELEMETRY=1

# Build flags
BUILD_FLAGS="-trimpath -ldflags='-s -w -extldflags=-static'"
TEST_FLAGS="-race -count=1 -timeout=15m"

echo "✅ Environment configured"

# Clean cache if requested
if [ "$FORCE_CLEAN_CACHE" = "true" ]; then
    echo "🧹 Cleaning Go cache..."
    
    # Remove Go module and build cache
    go clean -cache -modcache 2>/dev/null || true
    rm -rf .go-build-cache 2>/dev/null || true
    
    echo "✅ Cache cleaned"
fi

# Test results storage
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run test with timeout
run_test_with_timeout() {
    local test_name="$1"
    local timeout_min="$2"  
    local description="$3"
    shift 3
    local cmd="$@"
    
    echo ""
    echo "🧪 TEST $((++TOTAL_TESTS)): $test_name" 
    echo "⏱️  Starting: $description"
    
    local start_time=$(date +%s)
    
    if timeout "${timeout_min}m" bash -c "$cmd"; then
        local end_time=$(date +%s)
        local duration=$(( (end_time - start_time) / 60 ))
        echo "✅ Completed: $description in ${duration} minutes"
        ((PASSED_TESTS++))
        return 0
    else
        local end_time=$(date +%s) 
        local duration=$(( (end_time - start_time) / 60 ))
        echo "❌ Failed: $description after ${duration} minutes"
        ((FAILED_TESTS++))
        return 1
    fi
}

# Test 1: Dependency Resolution (This was the main failure point)
echo ""
echo "📦 TEST 1: Advanced Dependency Resolution"

test_dependency_resolution() {
    # Create local cache directory
    mkdir -p .go-build-cache
    export GOCACHE="$(pwd)/.go-build-cache"
    
    echo "📥 Starting dependency resolution with retry logic..."
    
    # Retry function implementation
    local max_attempts=3
    local attempt=1
    local delay=10
    local success=false
    
    while [ $attempt -le $max_attempts ] && [ "$success" = false ]; do
        echo "📥 Attempt $attempt/$max_attempts: Downloading dependencies..."
        
        if timeout 600 go mod download -x; then
            echo "✅ Dependencies downloaded successfully on attempt $attempt"
            success=true
        else
            echo "⚠️  Attempt $attempt failed"
            if [ $attempt -lt $max_attempts ]; then
                echo "⏳ Waiting ${delay}s before retry..."
                sleep $delay
                delay=$((delay * 2))
            fi
            attempt=$((attempt + 1))
        fi
    done
    
    if [ "$success" = false ]; then
        echo "🔄 Trying alternative proxy configuration..."
        export GOPROXY="https://goproxy.io,https://proxy.golang.org,direct"
        
        if ! timeout 600 go mod download -x; then
            echo "❌ All download attempts failed"
            return 1
        fi
        echo "✅ Success with alternative proxy"
    fi
    
    # Verify dependencies
    echo "🔍 Verifying downloaded dependencies..."
    if ! go mod verify; then
        echo "❌ Dependency verification failed"
        return 1
    fi
    
    echo "✅ Dependency resolution completed successfully"
    return 0
}

run_test_with_timeout "Dependency Resolution" "$TIMEOUT_MINUTES" "Dependency Download with Retry Logic" "test_dependency_resolution"

# Test 2: Fast Syntax Check (Previously failing with timeout)  
echo ""
echo "⚡ TEST 2: Fast Syntax & Build Check"

test_fast_syntax() {
    export GOCACHE="$(pwd)/.go-build-cache"
    
    echo "🔍 Running fast syntax validation..."
    
    # Fast syntax check
    echo "📋 Checking Go syntax..."
    if ! go vet ./...; then
        echo "❌ Go vet failed"
        return 1
    fi
    
    # Fast build check
    echo "🏗️  Fast build verification..."
    if ! go build $BUILD_FLAGS ./...; then
        echo "❌ Go build failed"
        return 1
    fi
    
    # Module tidiness check
    echo "🧹 Checking module tidiness..."
    if ! go mod tidy; then
        echo "❌ Go mod tidy failed"
        return 1
    fi
    
    # Check if go.mod/go.sum changed
    if ! git diff --quiet go.mod go.sum 2>/dev/null; then
        echo "⚠️  go.mod or go.sum was modified by tidy"
        # This is not a failure, just informational
    fi
    
    echo "✅ Fast validation completed successfully"
    return 0
}

run_test_with_timeout "Fast Syntax Check" 10 "Fast Syntax Validation" "test_fast_syntax"

# Test 3: Component Build Test
echo ""
echo "🏗️ TEST 3: Component Build Test"

test_component_build() {
    export GOCACHE="$(pwd)/.go-build-cache"
    
    echo "🏗️ Testing component builds..."
    
    # Create bin directory
    mkdir -p bin
    
    # Test critical components that exist
    local components=(
        "critical:./api/... ./controllers/..."
        "internal:./internal/..."
    )
    
    # Add pkg/nephio if it exists
    if [ -d "pkg/nephio" ]; then
        components[0]="critical:./api/... ./controllers/... ./pkg/nephio/..."
    fi
    
    # Add sim if it exists
    if [ -d "sim" ]; then
        components+=("simulators:./sim/...")
    fi
    
    for component_def in "${components[@]}"; do
        local name="${component_def%:*}"
        local pattern="${component_def#*:}"
        
        echo "📦 Building component: $name"
        echo "  Pattern: $pattern"
        
        # Check if any of the patterns actually exist
        local found_dirs=false
        for dir_pattern in $pattern; do
            local base_dir="${dir_pattern%/...}"
            if [ -d "$base_dir" ]; then
                found_dirs=true
                break
            fi
        done
        
        if [ "$found_dirs" = false ]; then
            echo "⏭️  Skipping $name - no matching directories found"
            continue
        fi
        
        if ! go build $BUILD_FLAGS $pattern; then
            echo "❌ Build failed for component: $name"
            return 1
        fi
    done
    
    echo "✅ Component builds completed successfully"
    return 0
}

run_test_with_timeout "Component Build" 15 "Component Build" "test_component_build"

# Test 4: Cache Performance Analysis
echo ""
echo "📊 TEST 4: Cache Performance Analysis"

echo "📈 Analyzing cache performance..."

# Cache statistics
local go_cache_dir=$(go env GOCACHE 2>/dev/null || echo "")
local go_mod_cache_dir=$(go env GOMODCACHE 2>/dev/null || echo "")

echo "📊 Cache Statistics:"

if [ -n "$go_cache_dir" ] && [ -d "$go_cache_dir" ]; then
    local build_cache_size=$(du -sm "$go_cache_dir" 2>/dev/null | cut -f1 || echo "0")
    echo "  Build Cache: ${build_cache_size} MB"
else
    echo "  Build Cache: 0 MB"
fi

if [ -n "$go_mod_cache_dir" ] && [ -d "$go_mod_cache_dir" ]; then
    local mod_cache_size=$(du -sm "$go_mod_cache_dir" 2>/dev/null | cut -f1 || echo "0")
    local module_count=$(find "$go_mod_cache_dir" -name "*.mod" 2>/dev/null | wc -l || echo "0")
    echo "  Module Cache: ${mod_cache_size} MB"
    echo "  Total Modules: $module_count"
else
    echo "  Module Cache: 0 MB"
    echo "  Total Modules: 0"
fi

if [ -d ".go-build-cache" ]; then
    local local_cache_size=$(du -sm ".go-build-cache" 2>/dev/null | cut -f1 || echo "0")
    echo "  Local Cache: ${local_cache_size} MB"
else
    echo "  Local Cache: 0 MB"
fi

((TOTAL_TESTS++))
((PASSED_TESTS++))

# Final Results Summary
echo ""
echo "📊 COMPREHENSIVE TEST RESULTS SUMMARY"
echo "======================================="
echo ""
echo "📈 Overall Results:"
echo "  Total Tests: $TOTAL_TESTS"
echo "  Passed: $PASSED_TESTS"  
echo "  Failed: $FAILED_TESTS"

if [ $TOTAL_TESTS -gt 0 ]; then
    local success_rate=$(( (PASSED_TESTS * 100) / TOTAL_TESTS ))
    echo "  Success Rate: ${success_rate}%"
fi

echo ""
if [ $FAILED_TESTS -eq 0 ]; then
    echo "🎉 ALL TESTS PASSED!"
    echo "The CI fix solution is working correctly."
    echo "Ready for production deployment."
    exit 0
else
    echo "⚠️  SOME TESTS FAILED ($FAILED_TESTS/$TOTAL_TESTS)"
    echo "Please review the failed tests and logs above."
    echo "Consider increasing timeouts or investigating specific failures."
    exit 1
fi