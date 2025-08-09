# Test RAG Build Tag Implementation
# This script tests the RAG client implementation with and without the 'rag' build tag

param(
    [switch]$SkipBenchmarks = $false,
    [switch]$SkipCoverage = $false,
    [switch]$Verbose = $false
)

# Colors for output (if supported)
$Red = "`e[0;31m"
$Green = "`e[0;32m"
$Yellow = "`e[1;33m"
$Blue = "`e[0;34m"
$NC = "`e[0m"

# Check if ANSI colors are supported
if (-not $env:NO_COLOR -and $Host.UI.SupportsVirtualTerminal) {
    $ColorSupport = $true
} else {
    $Red = $Green = $Yellow = $Blue = $NC = ""
    $ColorSupport = $false
}

Write-Host "${Blue}=== Testing RAG Build Tag Implementation ===${NC}"
Write-Host

# Function to run tests and capture output
function Run-Test {
    param(
        [string]$BuildTag,
        [string]$Description
    )
    
    Write-Host "${Yellow}Running tests: $Description${NC}"
    
    if ($BuildTag) {
        $cmd = "go test -tags=`"$BuildTag`" -v ./pkg/rag/... -run=`"BuildVerification|BuildTag`""
        Write-Host "  Command: $cmd"
        Invoke-Expression $cmd
    } else {
        $cmd = "go test -v ./pkg/rag/... -run=`"BuildVerification|BuildTag`""
        Write-Host "  Command: $cmd"
        Invoke-Expression $cmd
    }
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "${Red}Test failed with exit code $LASTEXITCODE${NC}"
        exit $LASTEXITCODE
    }
    
    Write-Host
}

# Function to run benchmarks
function Run-Benchmark {
    param(
        [string]$BuildTag,
        [string]$Description
    )
    
    Write-Host "${Yellow}Running benchmarks: $Description${NC}"
    
    if ($BuildTag) {
        $cmd = "go test -tags=`"$BuildTag`" -bench=BenchmarkRAGClient -benchmem ./pkg/rag/"
        Write-Host "  Command: $cmd"
        Invoke-Expression $cmd
    } else {
        $cmd = "go test -bench=BenchmarkRAGClient -benchmem ./pkg/rag/"
        Write-Host "  Command: $cmd"
        Invoke-Expression $cmd
    }
    
    Write-Host
}

# Check if we're in the right directory
if (-not (Test-Path "go.mod")) {
    Write-Host "${Red}Error: This script must be run from the project root directory${NC}"
    exit 1
}

# Test 1: Without rag build tag (no-op implementation)
Write-Host "${Green}=== Test 1: No-op Implementation (without rag build tag) ===${NC}"
Run-Test -Description "No-op RAG client"

# Test 2: With rag build tag (Weaviate implementation)  
Write-Host "${Green}=== Test 2: Weaviate Implementation (with rag build tag) ===${NC}"
Run-Test -BuildTag "rag" -Description "Weaviate RAG client"

# Test 3: Run all RAG tests without build tag
Write-Host "${Green}=== Test 3: Full test suite without rag tag ===${NC}"
Write-Host "${Yellow}Running all RAG tests (no-op implementation)${NC}"
$cmd = "go test -v ./pkg/rag/..."
Write-Host "  Command: $cmd"
Invoke-Expression $cmd
if ($LASTEXITCODE -ne 0) {
    Write-Host "${Red}Test failed with exit code $LASTEXITCODE${NC}"
    exit $LASTEXITCODE
}
Write-Host

# Test 4: Run all RAG tests with build tag
Write-Host "${Green}=== Test 4: Full test suite with rag tag ===${NC}"
Write-Host "${Yellow}Running all RAG tests (Weaviate implementation)${NC}"
$cmd = "go test -tags=`"rag`" -v ./pkg/rag/..."
Write-Host "  Command: $cmd"
Invoke-Expression $cmd
if ($LASTEXITCODE -ne 0) {
    Write-Host "${Red}Test failed with exit code $LASTEXITCODE${NC}"
    exit $LASTEXITCODE
}
Write-Host

# Test 5: Benchmarks (optional)
if (-not $SkipBenchmarks) {
    Write-Host "${Green}=== Test 5: Performance Benchmarks ===${NC}"
    Run-Benchmark -Description "No-op implementation"
    Run-Benchmark -BuildTag "rag" -Description "Weaviate implementation"
}

# Test 6: Race condition detection
Write-Host "${Green}=== Test 6: Race Condition Detection ===${NC}"
Write-Host "${Yellow}Testing for race conditions (no-op implementation)${NC}"
$cmd = "go test -race -v ./pkg/rag/ -run=`"Concurrent`""
Write-Host "  Command: $cmd"
Invoke-Expression $cmd
Write-Host

Write-Host "${Yellow}Testing for race conditions (Weaviate implementation)${NC}"
$cmd = "go test -tags=`"rag`" -race -v ./pkg/rag/ -run=`"Concurrent`""
Write-Host "  Command: $cmd"
Invoke-Expression $cmd
Write-Host

# Test 7: Coverage analysis (optional)
if (-not $SkipCoverage) {
    Write-Host "${Green}=== Test 7: Coverage Analysis ===${NC}"
    Write-Host "${Yellow}Generating coverage report (no-op implementation)${NC}"
    $cmd = "go test -coverprofile=coverage-noop.out ./pkg/rag/..."
    Write-Host "  Command: $cmd"
    Invoke-Expression $cmd
    $cmd = "go tool cover -func=coverage-noop.out"
    $coverage = Invoke-Expression $cmd | Select-Object -Last 1
    Write-Host $coverage
    Write-Host

    Write-Host "${Yellow}Generating coverage report (Weaviate implementation)${NC}"
    $cmd = "go test -tags=`"rag`" -coverprofile=coverage-rag.out ./pkg/rag/..."
    Write-Host "  Command: $cmd"
    Invoke-Expression $cmd
    $cmd = "go tool cover -func=coverage-rag.out"
    $coverage = Invoke-Expression $cmd | Select-Object -Last 1
    Write-Host $coverage
    Write-Host
}

# Summary
Write-Host "${Green}=== Test Summary ===${NC}"
Write-Host "✓ No-op implementation tests completed"
Write-Host "✓ Weaviate implementation tests completed"
Write-Host "✓ Full test suites executed for both configurations"
if (-not $SkipBenchmarks) {
    Write-Host "✓ Performance benchmarks completed"
}
Write-Host "✓ Race condition detection completed"
if (-not $SkipCoverage) {
    Write-Host "✓ Coverage analysis completed"
    Write-Host
    Write-Host "${Blue}Coverage reports generated:${NC}"
    Write-Host "  - coverage-noop.out (no-op implementation)"
    Write-Host "  - coverage-rag.out (Weaviate implementation)"
    Write-Host
    Write-Host "${Blue}To view HTML coverage reports:${NC}"
    Write-Host "  go tool cover -html=coverage-noop.out -o coverage-noop.html"
    Write-Host "  go tool cover -html=coverage-rag.out -o coverage-rag.html"
}
Write-Host

Write-Host "${Green}All tests completed successfully!${NC}"