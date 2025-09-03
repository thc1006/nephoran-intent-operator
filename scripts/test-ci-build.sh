#!/bin/bash
# =============================================================================
# CI Build Test Script - Verify Timeout Fixes
# =============================================================================
# This script tests the optimized CI build process to ensure it doesn't timeout
# =============================================================================

set -e

echo "🧪 Testing CI Build Timeout Fixes"
echo "=================================="

# Start time tracking
start_time=$(date +%s)

# Set build environment (same as CI)
export CGO_ENABLED=0
export GOOS=linux
export GOARCH=amd64
export GOMAXPROCS=4
export GOMEMLIMIT=4GiB

echo "📊 Environment Setup:"
echo "  Go Version: $(go version)"
echo "  CGO_ENABLED: $CGO_ENABLED"
echo "  GOMAXPROCS: $GOMAXPROCS"
echo "  GOMEMLIMIT: $GOMEMLIMIT"
echo ""

# Test 1: Critical components build (individual)
echo "🔨 Test 1: Building Critical Components (Individual)"
echo "====================================================="

CRITICAL_CMDS=(
    "cmd/intent-ingest"
    "cmd/llm-processor"
    "cmd/conductor"
    "cmd/nephio-bridge"
    "cmd/webhook"
)

mkdir -p bin/test

success_count=0
total_count=${#CRITICAL_CMDS[@]}

for cmd in "${CRITICAL_CMDS[@]}"; do
    echo "  Building $cmd..."
    component_start=$(date +%s)
    
    if timeout 60s go build -v -ldflags="-s -w" -o "bin/test/$(basename $cmd)" ./$cmd 2>/dev/null; then
        component_end=$(date +%s)
        duration=$((component_end - component_start))
        echo "  ✅ $cmd completed in ${duration}s"
        success_count=$((success_count + 1))
    else
        echo "  ❌ $cmd failed or timed out"
    fi
done

echo ""
echo "📊 Individual Build Results: $success_count/$total_count successful"

# Test 2: Controllers build
echo "🎮 Test 2: Building Controllers"
echo "==============================="

controller_start=$(date +%s)
if timeout 30s go build -v ./controllers 2>/dev/null; then
    controller_end=$(date +%s)
    duration=$((controller_end - controller_start))
    echo "✅ Controllers built successfully in ${duration}s"
    controllers_success=true
else
    echo "❌ Controllers build failed or timed out"
    controllers_success=false
fi

# Test 3: Parallel build simulation (if make is available)
echo ""
echo "🚀 Test 3: Simulating Parallel Builds"
echo "====================================="

parallel_start=$(date +%s)
parallel_success=0

# Simulate parallel builds with background processes
for cmd in "${CRITICAL_CMDS[@]:0:3}"; do  # Test first 3 components
    (
        if timeout 45s go build -v -ldflags="-s -w" -o "bin/test/parallel-$(basename $cmd)" ./$cmd 2>/dev/null; then
            echo "  ✅ Parallel: $cmd completed"
        else
            echo "  ❌ Parallel: $cmd failed"
        fi
    ) &
done

# Wait for all parallel builds
wait

parallel_end=$(date +%s)
parallel_duration=$((parallel_end - parallel_start))
echo "📊 Parallel build simulation completed in ${parallel_duration}s"

# Test 4: Syntax validation (fast check)
echo ""
echo "🔍 Test 4: Syntax Validation"
echo "============================"

syntax_errors=0
for cmd in "${CRITICAL_CMDS[@]}"; do
    echo "  Checking $cmd..."
    if ! timeout 10s go vet ./$cmd/... 2>/dev/null; then
        echo "  ⚠️  $cmd has syntax issues"
        syntax_errors=$((syntax_errors + 1))
    else
        echo "  ✅ $cmd syntax OK"
    fi
done

# Calculate total time
end_time=$(date +%s)
total_duration=$((end_time - start_time))

# Generate report
echo ""
echo "📋 Final Test Report"
echo "===================="
echo "Total Test Duration: ${total_duration}s"
echo ""
echo "Results Summary:"
echo "  Individual Builds: $success_count/$total_count successful"
echo "  Controllers Build: $([ "$controllers_success" = true ] && echo "✅ Success" || echo "❌ Failed")"
echo "  Syntax Validation: $((total_count - syntax_errors))/$total_count clean"
echo "  Total Components Tested: $total_count"
echo ""

# Determine overall status
if [ $success_count -ge 3 ] && [ $syntax_errors -le 1 ] && [ $total_duration -lt 300 ]; then
    echo "🎉 OVERALL STATUS: ✅ CI BUILD TIMEOUT FIXES WORKING"
    echo "   - At least 3/5 critical components build successfully"
    echo "   - Total time under 5 minutes (${total_duration}s)"
    echo "   - Minimal syntax issues ($syntax_errors)"
    echo ""
    echo "🚀 Ready for CI deployment!"
    exit 0
else
    echo "⚠️  OVERALL STATUS: ❌ ISSUES DETECTED"
    echo "   - Success rate: $success_count/$total_count"
    echo "   - Total time: ${total_duration}s"
    echo "   - Syntax errors: $syntax_errors"
    echo ""
    echo "🔧 Further optimization needed"
    exit 1
fi