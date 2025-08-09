#!/bin/bash

# Test RAG Build Tag Implementation
# This script tests the RAG client implementation with and without the 'rag' build tag

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Testing RAG Build Tag Implementation ===${NC}"
echo

# Function to run tests and capture output
run_test() {
    local build_tag="$1"
    local description="$2"
    
    echo -e "${YELLOW}Running tests: $description${NC}"
    
    if [ -n "$build_tag" ]; then
        echo "  Command: go test -tags=\"$build_tag\" -v ./pkg/rag/... -run=\"BuildVerification|BuildTag\""
        go test -tags="$build_tag" -v ./pkg/rag/... -run="BuildVerification|BuildTag"
    else
        echo "  Command: go test -v ./pkg/rag/... -run=\"BuildVerification|BuildTag\""
        go test -v ./pkg/rag/... -run="BuildVerification|BuildTag"
    fi
    
    echo
}

# Function to run benchmarks
run_benchmark() {
    local build_tag="$1"
    local description="$2"
    
    echo -e "${YELLOW}Running benchmarks: $description${NC}"
    
    if [ -n "$build_tag" ]; then
        echo "  Command: go test -tags=\"$build_tag\" -bench=BenchmarkRAGClient -benchmem ./pkg/rag/"
        go test -tags="$build_tag" -bench=BenchmarkRAGClient -benchmem ./pkg/rag/
    else
        echo "  Command: go test -bench=BenchmarkRAGClient -benchmem ./pkg/rag/"
        go test -bench=BenchmarkRAGClient -benchmem ./pkg/rag/
    fi
    
    echo
}

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo -e "${RED}Error: This script must be run from the project root directory${NC}"
    exit 1
fi

# Test 1: Without rag build tag (no-op implementation)
echo -e "${GREEN}=== Test 1: No-op Implementation (without rag build tag) ===${NC}"
run_test "" "No-op RAG client"

# Test 2: With rag build tag (Weaviate implementation)  
echo -e "${GREEN}=== Test 2: Weaviate Implementation (with rag build tag) ===${NC}"
run_test "rag" "Weaviate RAG client"

# Test 3: Run all RAG tests without build tag
echo -e "${GREEN}=== Test 3: Full test suite without rag tag ===${NC}"
echo -e "${YELLOW}Running all RAG tests (no-op implementation)${NC}"
echo "  Command: go test -v ./pkg/rag/..."
go test -v ./pkg/rag/...
echo

# Test 4: Run all RAG tests with build tag
echo -e "${GREEN}=== Test 4: Full test suite with rag tag ===${NC}"
echo -e "${YELLOW}Running all RAG tests (Weaviate implementation)${NC}"
echo "  Command: go test -tags=\"rag\" -v ./pkg/rag/..."
go test -tags="rag" -v ./pkg/rag/...
echo

# Test 5: Benchmarks
echo -e "${GREEN}=== Test 5: Performance Benchmarks ===${NC}"
run_benchmark "" "No-op implementation"
run_benchmark "rag" "Weaviate implementation"

# Test 6: Race condition detection
echo -e "${GREEN}=== Test 6: Race Condition Detection ===${NC}"
echo -e "${YELLOW}Testing for race conditions (no-op implementation)${NC}"
echo "  Command: go test -race -v ./pkg/rag/ -run=\"Concurrent\""
go test -race -v ./pkg/rag/ -run="Concurrent"
echo

echo -e "${YELLOW}Testing for race conditions (Weaviate implementation)${NC}"
echo "  Command: go test -tags=\"rag\" -race -v ./pkg/rag/ -run=\"Concurrent\""
go test -tags="rag" -race -v ./pkg/rag/ -run="Concurrent"
echo

# Test 7: Coverage analysis
echo -e "${GREEN}=== Test 7: Coverage Analysis ===${NC}"
echo -e "${YELLOW}Generating coverage report (no-op implementation)${NC}"
echo "  Command: go test -coverprofile=coverage-noop.out ./pkg/rag/..."
go test -coverprofile=coverage-noop.out ./pkg/rag/...
go tool cover -func=coverage-noop.out | tail -1
echo

echo -e "${YELLOW}Generating coverage report (Weaviate implementation)${NC}"
echo "  Command: go test -tags=\"rag\" -coverprofile=coverage-rag.out ./pkg/rag/..."
go test -tags="rag" -coverprofile=coverage-rag.out ./pkg/rag/...
go tool cover -func=coverage-rag.out | tail -1
echo

# Summary
echo -e "${GREEN}=== Test Summary ===${NC}"
echo "✓ No-op implementation tests completed"
echo "✓ Weaviate implementation tests completed"
echo "✓ Full test suites executed for both configurations"
echo "✓ Performance benchmarks completed"
echo "✓ Race condition detection completed"
echo "✓ Coverage analysis completed"
echo
echo -e "${BLUE}Coverage reports generated:${NC}"
echo "  - coverage-noop.out (no-op implementation)"
echo "  - coverage-rag.out (Weaviate implementation)"
echo
echo -e "${BLUE}To view HTML coverage reports:${NC}"
echo "  go tool cover -html=coverage-noop.out -o coverage-noop.html"
echo "  go tool cover -html=coverage-rag.out -o coverage-rag.html"
echo

echo -e "${GREEN}All tests completed successfully!${NC}"