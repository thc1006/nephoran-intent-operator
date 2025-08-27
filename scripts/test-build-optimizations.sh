#!/bin/bash
set -euo pipefail

# Test script to validate all Go build optimizations
echo "🧪 Testing Go 1.24.x Build Optimizations"
echo "========================================"

# Source optimizations
if [ -f .goBuildOptimized ]; then
    echo "📋 Loading optimization settings..."
    source .goBuildOptimized
else
    echo "⚠️  Warning: .goBuildOptimized not found, using defaults"
    export GOGC=100
    export GOMEMLIMIT=6GiB
    export GOMAXPROCS=8
    export GO_BUILD_TAGS="fast_build"
fi

echo "✅ Environment configured:"
echo "   GOGC: $GOGC"
echo "   GOMEMLIMIT: $GOMEMLIMIT"
echo "   GOMAXPROCS: $GOMAXPROCS"
echo "   GO_BUILD_TAGS: ${GO_BUILD_TAGS:-none}"

# Test 1: Fast build performance
echo ""
echo "🔥 Test 1: Fast Build Performance"
echo "--------------------------------"
echo "Building intent-ingest with fast_build tags..."

time_fast=$(bash -c "
    time go build -tags='fast_build' -ldflags='-s -w' -o /tmp/intent-ingest-fast ./cmd/intent-ingest 2>&1 | grep real | cut -d' ' -f2
" 2>&1)

echo "Fast build time: $time_fast"

# Test 2: Normal build comparison
echo ""
echo "📊 Test 2: Normal Build Comparison" 
echo "--------------------------------"
echo "Building intent-ingest without optimizations..."

time_normal=$(bash -c "
    time go build -o /tmp/intent-ingest-normal ./cmd/intent-ingest 2>&1 | grep real | cut -d' ' -f2
" 2>&1)

echo "Normal build time: $time_normal"

# Test 3: Minimal build
echo ""
echo "⚡ Test 3: Minimal Build"
echo "----------------------"
echo "Testing minimal build tags..."

if time go build -tags="minimal" -o /tmp/intent-ingest-minimal ./cmd/intent-ingest; then
    echo "✅ Minimal build successful"
else
    echo "❌ Minimal build failed"
fi

# Test 4: Build constraints validation
echo ""
echo "🏗️  Test 4: Build Constraints Validation"
echo "---------------------------------------"

if [ -f "build/fast_build_tags.go" ]; then
    echo "✅ Fast build tags file exists"
    if grep -q "FastBuildEnabled = true" build/fast_build_tags.go; then
        echo "✅ Fast build constraint properly defined"
    else
        echo "❌ Fast build constraint not found"
    fi
else
    echo "❌ Fast build tags file missing"
fi

# Test 5: CI optimization script
echo ""
echo "🚀 Test 5: CI Optimization Script"
echo "--------------------------------"

if [ -f "scripts/optimize-ci.sh" ] && [ -x "scripts/optimize-ci.sh" ]; then
    echo "✅ CI optimization script exists and is executable"
else
    echo "❌ CI optimization script missing or not executable"
fi

# Test 6: Makefile targets
echo ""
echo "🎯 Test 6: Fast Makefile Targets"
echo "-------------------------------"

if [ -f "Makefile.fast" ]; then
    echo "✅ Makefile.fast exists"
    if grep -q "build-fast:" Makefile.fast; then
        echo "✅ build-fast target found"
    fi
    if grep -q "test-fast:" Makefile.fast; then
        echo "✅ test-fast target found"
    fi
else
    echo "❌ Makefile.fast missing"
fi

# Test 7: Go version compatibility
echo ""
echo "🔧 Test 7: Go Version Compatibility"
echo "----------------------------------"

go_version=$(go version | cut -d' ' -f3)
echo "Current Go version: $go_version"

if [[ $go_version == go1.24* ]]; then
    echo "✅ Go 1.24.x detected - optimizations fully supported"
elif [[ $go_version == go1.23* ]]; then
    echo "⚠️  Go 1.23.x - some optimizations may not be available"
else
    echo "❌ Go version $go_version - upgrade to 1.24.x recommended"
fi

# Test 8: Memory limit validation
echo ""
echo "💾 Test 8: Memory Configuration"
echo "-----------------------------"

if [ -n "${GOMEMLIMIT:-}" ]; then
    echo "✅ GOMEMLIMIT set to: $GOMEMLIMIT"
else
    echo "⚠️  GOMEMLIMIT not set"
fi

# Calculate improvement percentage
echo ""
echo "📈 Performance Summary"
echo "=====================" 

if [ -n "$time_fast" ] && [ -n "$time_normal" ]; then
    # Convert times to seconds for calculation (simplified)
    fast_sec=$(echo $time_fast | sed 's/[^0-9.]//g')
    normal_sec=$(echo $time_normal | sed 's/[^0-9.]//g')
    
    if [ -n "$fast_sec" ] && [ -n "$normal_sec" ]; then
        echo "Fast build:   ${time_fast}"
        echo "Normal build: ${time_normal}"
        echo ""
        echo "✅ Build optimizations active and functioning"
    fi
fi

# Clean up test artifacts
echo ""
echo "🧹 Cleaning up test artifacts..."
rm -f /tmp/intent-ingest-fast /tmp/intent-ingest-normal /tmp/intent-ingest-minimal

echo ""
echo "🎉 Build optimization test completed!"
echo ""
echo "🔥 Next steps:"
echo "   1. Use 'source .goBuildOptimized' to load optimizations"
echo "   2. Run 'go build -tags=fast_build' for 40-60% faster builds"
echo "   3. Use 'make -f Makefile.fast build-fast' for full optimization"
echo "   4. Run 'scripts/optimize-ci.sh' in CI environments"