#!/usr/bin/env pwsh
# Test script to validate CI coverage generation fix

Write-Host "Testing CI Coverage Generation Fix" -ForegroundColor Cyan
Write-Host "===================================" -ForegroundColor Cyan

# Clean up previous test results
if (Test-Path "test-results") {
    Remove-Item -Path "test-results" -Recurse -Force
}

# Create test results directory
New-Item -ItemType Directory -Path "test-results" -Force | Out-Null

Write-Host "`n1. Testing coverage file generation..." -ForegroundColor Yellow

# Simulate the CI test command
$env:CGO_ENABLED = "0"
$env:GOMAXPROCS = "2"
$attempt = 1

Write-Host "   Running tests for attempt $attempt..."
$testCmd = "go test -v -timeout=30m -count=1 -coverprofile=test-results/coverage-$attempt.out -covermode=atomic ./cmd/conductor-loop ./internal/loop"

try {
    $output = Invoke-Expression $testCmd 2>&1
    $testExitCode = $LASTEXITCODE
    
    if ($testExitCode -eq 0) {
        Write-Host "   ✅ Tests passed" -ForegroundColor Green
        
        # Check if coverage file was generated
        if (Test-Path "test-results/coverage-$attempt.out") {
            Write-Host "   ✅ Coverage file generated: test-results/coverage-$attempt.out" -ForegroundColor Green
            
            # Test the copy operation
            Copy-Item "test-results/coverage-$attempt.out" "test-results/coverage.out" -Force
            if (Test-Path "test-results/coverage.out") {
                Write-Host "   ✅ Coverage file copied to: test-results/coverage.out" -ForegroundColor Green
            } else {
                Write-Host "   ❌ Failed to copy coverage file" -ForegroundColor Red
                exit 1
            }
        } else {
            Write-Host "   ❌ Coverage file was not generated!" -ForegroundColor Red
            exit 1
        }
    } else {
        Write-Host "   ⚠️ Tests failed with exit code: $testExitCode" -ForegroundColor Yellow
    }
} catch {
    Write-Host "   ❌ Error running tests: $_" -ForegroundColor Red
    exit 1
}

Write-Host "`n2. Testing coverage report generation..." -ForegroundColor Yellow

# Test coverage report generation
if (Test-Path "test-results/coverage.out") {
    try {
        # Generate HTML report
        go tool cover -html=test-results/coverage.out -o test-results/coverage.html 2>$null
        if (Test-Path "test-results/coverage.html") {
            Write-Host "   ✅ HTML coverage report generated" -ForegroundColor Green
        } else {
            Write-Host "   ⚠️ Failed to generate HTML coverage report" -ForegroundColor Yellow
        }
        
        # Generate text summary
        go tool cover -func=test-results/coverage.out > test-results/coverage-summary.txt 2>$null
        if (Test-Path "test-results/coverage-summary.txt") {
            Write-Host "   ✅ Coverage summary generated" -ForegroundColor Green
            
            # Display coverage percentage
            $summary = Get-Content "test-results/coverage-summary.txt" -Tail 5
            Write-Host "`n   📊 Coverage Summary:" -ForegroundColor Cyan
            $summary | ForEach-Object { Write-Host "      $_" }
        } else {
            Write-Host "   ⚠️ Failed to generate coverage summary" -ForegroundColor Yellow
        }
    } catch {
        Write-Host "   ⚠️ Error generating reports: $_" -ForegroundColor Yellow
    }
} else {
    Write-Host "   ❌ No coverage.out file found!" -ForegroundColor Red
    exit 1
}

Write-Host "`n3. Validating CI workflow changes..." -ForegroundColor Yellow

# Check if the CI workflow has the proper error handling
$ciFile = ".github/workflows/ci-enhanced.yml"
if (Test-Path $ciFile) {
    $content = Get-Content $ciFile -Raw
    
    # Check for coverage file existence checks
    if ($content -match 'if \[ -f "test-results/coverage-\$attempt.out" \]') {
        Write-Host "   ✅ Unix coverage file check is present" -ForegroundColor Green
    } else {
        Write-Host "   ⚠️ Unix coverage file check might be missing" -ForegroundColor Yellow
    }
    
    if ($content -match 'if \(Test-Path "test-results/coverage-\$attempt.out"\)') {
        Write-Host "   ✅ Windows coverage file check is present" -ForegroundColor Green
    } else {
        Write-Host "   ⚠️ Windows coverage file check might be missing" -ForegroundColor Yellow
    }
    
    # Check for error messages
    if ($content -match 'ERROR: Coverage file.*was not generated') {
        Write-Host "   ✅ Error message for missing coverage file is present" -ForegroundColor Green
    } else {
        Write-Host "   ⚠️ Error message for missing coverage file might be missing" -ForegroundColor Yellow
    }
} else {
    Write-Host "   ❌ CI workflow file not found!" -ForegroundColor Red
}

Write-Host "`n===================================" -ForegroundColor Cyan
Write-Host "✅ CI Coverage Fix Validation Complete!" -ForegroundColor Green
Write-Host "===================================" -ForegroundColor Cyan