#!/bin/bash
# Cross-Platform Race Detection Test Script for Linux/macOS
# Handles CGO setup and race detection testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
GRAY='\033[0;90m'
NC='\033[0m' # No Color

# Default values
TIMEOUT_MINUTES=15
VERBOSE=false
SKIP_BUILD_CHECK=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --timeout)
            TIMEOUT_MINUTES="$2"
            shift 2
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --skip-build)
            SKIP_BUILD_CHECK=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo -e "${GREEN}🔍 Race Detection Test Suite${NC}"
echo -e "${GREEN}================================${NC}"
echo ""

# Platform Detection
OS_TYPE="Unknown"
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS_TYPE="Linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS_TYPE="macOS"
elif [[ "$OSTYPE" == "cygwin" ]] || [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "win32" ]]; then
    OS_TYPE="Windows"
fi

echo -e "${CYAN}📊 System Information:${NC}"
echo -e "${GRAY}  Platform: $OS_TYPE${NC}"
echo -e "${GRAY}  Go Version: $(go version)${NC}"
echo -e "${GRAY}  CPU Architecture: $(uname -m)${NC}"
echo ""

# Function to detect C compiler
find_c_compiler() {
    local compilers_found=0
    
    echo -e "${CYAN}🔧 Checking for C Compiler...${NC}"
    
    # Check for GCC
    if command -v gcc &> /dev/null; then
        GCC_VERSION=$(gcc --version | head -n1)
        echo -e "${GREEN}  ✅ Found: GCC - $GCC_VERSION${NC}"
        compilers_found=$((compilers_found + 1))
        PREFERRED_CC="gcc"
    fi
    
    # Check for Clang
    if command -v clang &> /dev/null; then
        CLANG_VERSION=$(clang --version | head -n1)
        echo -e "${GREEN}  ✅ Found: Clang - $CLANG_VERSION${NC}"
        compilers_found=$((compilers_found + 1))
        if [ -z "$PREFERRED_CC" ]; then
            PREFERRED_CC="clang"
        fi
    fi
    
    # Check for CC environment variable
    if [ -n "$CC" ] && command -v "$CC" &> /dev/null; then
        echo -e "${GREEN}  ✅ Found: \$CC=$CC${NC}"
        compilers_found=$((compilers_found + 1))
        PREFERRED_CC="$CC"
    fi
    
    return $compilers_found
}

# Check for C compiler
find_c_compiler
COMPILER_COUNT=$?

if [ $COMPILER_COUNT -eq 0 ]; then
    echo ""
    echo -e "${YELLOW}⚠️  WARNING: No C compiler found!${NC}"
    echo ""
    echo -e "${YELLOW}Race detection requires CGO, which needs a C compiler.${NC}"
    echo ""
    
    if [ "$OS_TYPE" == "Linux" ]; then
        echo -e "${CYAN}📦 To enable race detection on Linux, install GCC:${NC}"
        
        # Detect Linux distribution
        if [ -f /etc/os-release ]; then
            . /etc/os-release
            case "$ID" in
                ubuntu|debian)
                    echo -e "${BLUE}  sudo apt-get update && sudo apt-get install build-essential${NC}"
                    ;;
                fedora)
                    echo -e "${BLUE}  sudo dnf groupinstall 'Development Tools'${NC}"
                    ;;
                centos|rhel)
                    echo -e "${BLUE}  sudo yum groupinstall 'Development Tools'${NC}"
                    ;;
                arch)
                    echo -e "${BLUE}  sudo pacman -S base-devel${NC}"
                    ;;
                alpine)
                    echo -e "${BLUE}  apk add build-base${NC}"
                    ;;
                *)
                    echo -e "${BLUE}  Please install GCC or Clang using your package manager${NC}"
                    ;;
            esac
        fi
    elif [ "$OS_TYPE" == "macOS" ]; then
        echo -e "${CYAN}📦 To enable race detection on macOS:${NC}"
        echo -e "${BLUE}  Install Xcode Command Line Tools:${NC}"
        echo -e "${BLUE}  xcode-select --install${NC}"
    fi
    
    echo ""
    echo -e "${YELLOW}⏭️  Continuing with standard tests (no race detection)...${NC}"
    CGO_ENABLED=0
    export CGO_ENABLED
else
    echo -e "${GREEN}✅ C compiler(s) available for CGO${NC}"
    echo -e "${GRAY}  Using: $PREFERRED_CC${NC}"
    CGO_ENABLED=1
    export CGO_ENABLED
    export CC="$PREFERRED_CC"
fi

echo ""

# Function to run tests with proper CGO setup
run_go_test() {
    local test_pattern="${1:-./...}"
    local test_name="${2:-All Tests}"
    local use_race="${3:-false}"
    local timeout="${4:-10}"
    
    echo -e "${CYAN}🧪 Running: $test_name${NC}"
    
    # Build test command
    local test_cmd="go test $test_pattern -timeout=${timeout}m"
    
    if [ "$use_race" == "true" ] && [ "$CGO_ENABLED" == "1" ]; then
        test_cmd="$test_cmd -race"
        echo -e "${GRAY}  CGO_ENABLED=1 (Race detection enabled)${NC}"
    else
        echo -e "${GRAY}  CGO_ENABLED=$CGO_ENABLED ($([ "$CGO_ENABLED" == "1" ] && echo "CGO enabled" || echo "CGO disabled"))${NC}"
    fi
    
    if [ "$VERBOSE" == "true" ]; then
        test_cmd="$test_cmd -v"
    fi
    
    echo -e "${GRAY}  Command: $test_cmd${NC}"
    echo ""
    
    # Execute the test command
    local start_time=$(date +%s)
    local exit_code=0
    
    # Run test and capture output
    if [ "$VERBOSE" == "true" ]; then
        CGO_ENABLED=$CGO_ENABLED $test_cmd || exit_code=$?
    else
        CGO_ENABLED=$CGO_ENABLED $test_cmd > /tmp/test_output_$$.log 2>&1 || exit_code=$?
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}✅ $test_name completed in ${duration}s${NC}"
        return 0
    else
        echo -e "${RED}❌ $test_name failed after ${duration}s${NC}"
        
        # Check for specific CGO/race errors
        if [ -f /tmp/test_output_$$.log ]; then
            if grep -q "requires cgo\|CGO_ENABLED" /tmp/test_output_$$.log; then
                echo -e "${RED}  Error: CGO not properly configured${NC}"
                echo -e "${YELLOW}  Please install a C compiler (see instructions above)${NC}"
            fi
            
            if [ "$VERBOSE" != "true" ]; then
                echo -e "${GRAY}Output (last 20 lines):${NC}"
                tail -n 20 /tmp/test_output_$$.log | sed 's/^/  /'
            fi
            
            rm -f /tmp/test_output_$$.log
        fi
        
        return 1
    fi
}

# Track test results
declare -a test_results
declare -a test_names
test_count=0
passed_count=0

# Test 1: Build verification (without race detection)
if [ "$SKIP_BUILD_CHECK" != "true" ]; then
    echo -e "${MAGENTA}📦 TEST 1: Build Verification${NC}"
    echo -e "${GRAY}-------------------------------${NC}"
    
    if run_go_test "./..." "Build Test" "false" "5"; then
        test_results[test_count]=1
        passed_count=$((passed_count + 1))
    else
        test_results[test_count]=0
    fi
    test_names[test_count]="Build Verification"
    test_count=$((test_count + 1))
    
    echo ""
fi

# Test 2: Standard tests (without race detection)
echo -e "${MAGENTA}🧪 TEST 2: Standard Unit Tests${NC}"
echo -e "${GRAY}-------------------------------${NC}"

if run_go_test "./..." "Standard Tests" "false" "10"; then
    test_results[test_count]=1
    passed_count=$((passed_count + 1))
else
    test_results[test_count]=0
fi
test_names[test_count]="Standard Tests"
test_count=$((test_count + 1))

echo ""

# Test 3: Race detection tests (if CGO available)
if [ "$CGO_ENABLED" == "1" ]; then
    echo -e "${MAGENTA}🏃 TEST 3: Race Detection Tests${NC}"
    echo -e "${GRAY}-------------------------------${NC}"
    
    # Test specific packages known to have race conditions
    packages=(
        "./pkg/security/..."
        "./pkg/controllers/..."
        "./pkg/llm/..."
        "./internal/loop/..."
    )
    
    package_names=(
        "Security Package"
        "Controllers"
        "LLM Package"
        "Loop Package"
    )
    
    for i in "${!packages[@]}"; do
        pkg="${packages[$i]}"
        name="${package_names[$i]}"
        
        # Check if package directory exists
        pkg_dir=$(echo "$pkg" | sed 's/\.\.\.$//' | sed 's/^\.\///')
        if [ -d "$pkg_dir" ]; then
            if run_go_test "$pkg" "$name Race Tests" "true" "$TIMEOUT_MINUTES"; then
                test_results[test_count]=1
                passed_count=$((passed_count + 1))
            else
                test_results[test_count]=0
            fi
            test_names[test_count]="$name Race"
            test_count=$((test_count + 1))
            echo ""
        fi
    done
else
    echo -e "${YELLOW}⏭️  TEST 3: Race Detection Tests - SKIPPED (No C compiler)${NC}"
    echo ""
fi

# Test 4: Verify race detector functionality
if [ "$CGO_ENABLED" == "1" ]; then
    echo -e "${MAGENTA}🔍 TEST 4: Race Detector Verification${NC}"
    echo -e "${GRAY}-------------------------------${NC}"
    
    # Create a test file with intentional race condition
    TEMP_TEST="/tmp/race_test_$$.go"
    cat > "$TEMP_TEST" << 'EOF'
package main

import (
    "testing"
    "sync"
)

func TestRaceDetection(t *testing.T) {
    var counter int
    var wg sync.WaitGroup
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            counter++ // Intentional race condition
        }()
    }
    
    wg.Wait()
    t.Logf("Counter value: %d", counter)
}
EOF
    
    echo -e "${GRAY}  Testing race detector functionality...${NC}"
    
    # Run the test and check for race detection
    if CGO_ENABLED=1 go test -race "$TEMP_TEST" 2>&1 | grep -q "WARNING: DATA RACE\|race detected"; then
        echo -e "${GREEN}✅ Race detector is working correctly${NC}"
        test_results[test_count]=1
        passed_count=$((passed_count + 1))
    else
        echo -e "${YELLOW}⚠️  Race detector may not be functioning properly${NC}"
        test_results[test_count]=0
    fi
    test_names[test_count]="Race Detector Verification"
    test_count=$((test_count + 1))
    
    rm -f "$TEMP_TEST"
    echo ""
fi

# Summary
echo -e "${GREEN}📊 TEST RESULTS SUMMARY${NC}"
echo -e "${GREEN}=======================${NC}"
echo ""

failed_count=$((test_count - passed_count))

echo -e "${CYAN}📈 Results:${NC}"
echo -e "${GRAY}  Total Tests: $test_count${NC}"
echo -e "${GREEN}  Passed: $passed_count${NC}"
if [ $failed_count -gt 0 ]; then
    echo -e "${RED}  Failed: $failed_count${NC}"
else
    echo -e "${GRAY}  Failed: $failed_count${NC}"
fi

success_rate=$((passed_count * 100 / test_count))
if [ $failed_count -eq 0 ]; then
    echo -e "${GREEN}  Success Rate: ${success_rate}%${NC}"
else
    echo -e "${YELLOW}  Success Rate: ${success_rate}%${NC}"
fi

echo ""
echo -e "${CYAN}⏱️  Test Results:${NC}"
for i in "${!test_names[@]}"; do
    if [ "${test_results[$i]}" -eq 1 ]; then
        echo -e "${GREEN}  ✅ ${test_names[$i]}${NC}"
    else
        echo -e "${RED}  ❌ ${test_names[$i]}${NC}"
    fi
done

echo ""

if [ "$CGO_ENABLED" == "0" ]; then
    echo -e "${YELLOW}💡 TIP: Install a C compiler to enable race detection tests${NC}"
    echo -e "${YELLOW}        This helps catch concurrency bugs before production!${NC}"
    echo ""
fi

if [ $failed_count -eq 0 ]; then
    echo -e "${GREEN}🎉 All tests passed successfully!${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some tests failed. Review the output above for details.${NC}"
    exit 1
fi