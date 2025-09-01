# Final Build Tag Verification Script for Nephoran
# Tests realistic build tag combinations for the specific packages

Write-Host "=== Nephoran Build Tag Verification - Final Test ===" -ForegroundColor Cyan
Write-Host ""

$successCount = 0
$failureCount = 0

# Function to run build test
function Test-BuildCombination {
    param(
        [string]$TestName,
        [string]$BuildTags,
        [string]$TestPackages
    )
    
    $tagsDisplay = if ($BuildTags) { $BuildTags } else { "(none)" }
    Write-Host "[$TestName] Testing build with tags: $tagsDisplay" -ForegroundColor Yellow
    
    try {
        $buildArgs = @()
        if ($BuildTags) {
            $buildArgs += "-tags"
            $buildArgs += $BuildTags
        }
        $buildArgs += $TestPackages.Split(" ")
        
        $result = & go build @buildArgs 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ $TestName`: SUCCESS" -ForegroundColor Green
            $script:successCount++
        } else {
            Write-Host "✗ $TestName`: FAILED" -ForegroundColor Red
            Write-Host "  Error details:" -ForegroundColor Red
            $result | ForEach-Object { Write-Host "    $_" -ForegroundColor Red }
            $script:failureCount++
        }
    } catch {
        Write-Host "✗ $TestName`: FAILED (Exception)" -ForegroundColor Red
        Write-Host "  Exception: $($_.Exception.Message)" -ForegroundColor Red
        $script:failureCount++
    }
    Write-Host ""
}

# Core package combinations that should work

Write-Host "=== Core Package Tests ===" -ForegroundColor Cyan

# Config package - should work with all tag combinations
Test-BuildCombination "Config (no tags)" "" "./pkg/config"
Test-BuildCombination "Config (disable_rag)" "disable_rag" "./pkg/config"
Test-BuildCombination "Config (stub)" "stub" "./pkg/config"
Test-BuildCombination "Config (stub + disable_rag)" "stub,disable_rag" "./pkg/config"

Write-Host "=== LLM Package Tests ===" -ForegroundColor Cyan

# LLM package - test realistic combinations
Test-BuildCombination "LLM (no tags)" "" "./pkg/llm"
Test-BuildCombination "LLM (disable_rag)" "disable_rag" "./pkg/llm"
Test-BuildCombination "LLM (stub)" "stub" "./pkg/llm"

Write-Host "=== Services Package Tests ===" -ForegroundColor Cyan

# Services package - only works without disable_rag
Test-BuildCombination "Services (no tags)" "" "./pkg/services"
Test-BuildCombination "Services (stub only)" "stub" "./pkg/services"

Write-Host "=== RAG Package Tests ===" -ForegroundColor Cyan

# RAG package - test tag combinations
Test-BuildCombination "RAG (no tags)" "" "./pkg/rag"
Test-BuildCombination "RAG (disable_rag)" "disable_rag" "./pkg/rag"
Test-BuildCombination "RAG (stub)" "stub" "./pkg/rag"

Write-Host "=== Combined Package Tests ===" -ForegroundColor Cyan

# Combined tests for realistic scenarios
Test-BuildCombination "Full Stack (no tags)" "" "./pkg/config ./pkg/llm ./pkg/services"
Test-BuildCombination "RAG Disabled Stack" "disable_rag" "./pkg/config ./pkg/llm"
Test-BuildCombination "Stub Implementation" "stub" "./pkg/config ./pkg/llm ./pkg/services ./pkg/rag"

# Summary
Write-Host "=== Build Verification Summary ===" -ForegroundColor Cyan
$totalTests = $successCount + $failureCount
Write-Host "Total tests: $totalTests" -ForegroundColor Gray
Write-Host "Successful: $successCount" -ForegroundColor Green
Write-Host "Failed: $failureCount" -ForegroundColor $(if ($failureCount -eq 0) { "Green" } else { "Red" })
Write-Host ""

if ($failureCount -eq 0) {
    Write-Host "🎉 All build tag combinations verified successfully!" -ForegroundColor Green
    Write-Host "Build tag system is working correctly across all packages." -ForegroundColor Green
    exit 0
} else {
    Write-Host "❌ $failureCount build test(s) failed. Please review the errors above." -ForegroundColor Red
    Write-Host ""
    Write-Host "Note: Some failures may be expected if certain combinations are not supported." -ForegroundColor Yellow
    Write-Host "Review the error details to determine if fixes are needed." -ForegroundColor Yellow
    exit 1
}