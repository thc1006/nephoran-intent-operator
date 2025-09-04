# NL to Intent Complete Test Script
# Tests POST /nl/intent endpoint functionality

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "  NL to Intent Endpoint Test Suite" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

# Test endpoint
$endpoint = "http://localhost:8090/nl/intent"

# Test counters
$passCount = 0
$failCount = 0

# Test function
function Test-NLIntent {
    param(
        [string]$TestName,
        [string]$Input,
        [string]$ExpectedPattern,
        [bool]$ShouldFail = $false
    )
    
    Write-Host "[$($passCount + $failCount + 1)] Testing: $TestName" -ForegroundColor Yellow
    Write-Host "    Input: '$Input'" -ForegroundColor Gray
    
    try {
        $response = curl.exe -X POST $endpoint -H "Content-Type: text/plain" -d $Input 2>$null
        Write-Host "    Response: $response" -ForegroundColor Gray
        
        if ($ShouldFail) {
            if ($response -like "*error*") {
                Write-Host "    PASS - Expected error response" -ForegroundColor Green
                $script:passCount++
            }
            else {
                Write-Host "    FAIL - Should have failed but succeeded" -ForegroundColor Red
                $script:failCount++
            }
        }
        else {
            if ($response -like $ExpectedPattern) {
                Write-Host "    PASS" -ForegroundColor Green
                $script:passCount++
            }
            else {
                Write-Host "    FAIL - Response doesn't match expected pattern" -ForegroundColor Red
                Write-Host "    Expected pattern: $ExpectedPattern" -ForegroundColor Red
                $script:failCount++
            }
        }
    }
    catch {
        Write-Host "    FAIL - Request failed: $_" -ForegroundColor Red
        $script:failCount++
    }
    
    Write-Host ""
}

# Check server status
Write-Host "Checking server status..." -ForegroundColor Cyan
try {
    $testResponse = curl.exe -X GET "http://localhost:8090/" 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Note: Server may not be fully ready, but will continue testing" -ForegroundColor Yellow
    }
}
catch {
    Write-Host "Warning: Cannot connect to server, please ensure server is running on port 8090" -ForegroundColor Yellow
    Write-Host "  Start with: .\test-nl-server.exe" -ForegroundColor Yellow
    Write-Host ""
}

Write-Host "Starting tests..." -ForegroundColor Cyan
Write-Host "----------------------------------------`n" -ForegroundColor Gray

# === Positive Test Cases ===
Write-Host "[Positive Test Cases]" -ForegroundColor White
Write-Host "=====================`n" -ForegroundColor Gray

Test-NLIntent `
    -TestName "Standard scaling intent with namespace" `
    -Input "scale nf-sim to 4 in ns ran-a" `
    -ExpectedPattern '*"intent_type":"scaling"*"replicas":4*"namespace":"ran-a"*'

Test-NLIntent `
    -TestName "Scaling without namespace (uses default)" `
    -Input "scale my-app to 3" `
    -ExpectedPattern '*"intent_type":"scaling"*"replicas":3*"namespace":"default"*'

Test-NLIntent `
    -TestName "Case insensitive test" `
    -Input "SCALE APP-NAME TO 5 IN NS PRODUCTION" `
    -ExpectedPattern '*"intent_type":"scaling"*"replicas":5*"namespace":"PRODUCTION"*'

Test-NLIntent `
    -TestName "Minimum replicas value (1)" `
    -Input "scale service to 1 in ns test" `
    -ExpectedPattern '*"intent_type":"scaling"*"replicas":1*'

Test-NLIntent `
    -TestName "Maximum replicas value (100)" `
    -Input "scale service to 100 in ns prod" `
    -ExpectedPattern '*"intent_type":"scaling"*"replicas":100*'

Test-NLIntent `
    -TestName "Complex service name" `
    -Input "scale nginx-ingress-controller to 3 in ns kube-system" `
    -ExpectedPattern '*"intent_type":"scaling"*"target":"nginx-ingress-controller"*'

# === Negative Test Cases ===
Write-Host "`n[Negative Test Cases]" -ForegroundColor White
Write-Host "=====================`n" -ForegroundColor Gray

Test-NLIntent `
    -TestName "Exceeds max replicas limit (>100)" `
    -Input "scale app to 150 in ns test" `
    -ExpectedPattern "*error*" `
    -ShouldFail $true

Test-NLIntent `
    -TestName "Below min replicas limit (<1)" `
    -Input "scale app to 0 in ns test" `
    -ExpectedPattern "*error*" `
    -ShouldFail $true

Test-NLIntent `
    -TestName "Negative replicas" `
    -Input "scale app to -5 in ns test" `
    -ExpectedPattern "*error*" `
    -ShouldFail $true

Test-NLIntent `
    -TestName "Invalid command format" `
    -Input "this is not a valid command" `
    -ExpectedPattern "*error*" `
    -ShouldFail $true

Test-NLIntent `
    -TestName "Empty input" `
    -Input "" `
    -ExpectedPattern "*error*" `
    -ShouldFail $true

Test-NLIntent `
    -TestName "Non-numeric replicas" `
    -Input "scale app to abc in ns test" `
    -ExpectedPattern "*error*" `
    -ShouldFail $true

# === Other Intent Types ===
Write-Host "`n[Other Intent Types]" -ForegroundColor White
Write-Host "====================`n" -ForegroundColor Gray

Test-NLIntent `
    -TestName "Deploy intent" `
    -Input "deploy nginx in ns production" `
    -ExpectedPattern '*"intent_type":"deployment"*"target":"nginx"*'

Test-NLIntent `
    -TestName "Delete intent" `
    -Input "delete old-service from ns staging" `
    -ExpectedPattern '*"intent_type":"deletion"*"target":"old-service"*'

Test-NLIntent `
    -TestName "Update configuration intent" `
    -Input "update myapp set replicas=5 in ns prod" `
    -ExpectedPattern '*"intent_type":"configuration"*"config":*"replicas":"5"*'

# === Test Summary ===
Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "           Test Results Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Passed: $passCount" -ForegroundColor Green
if ($failCount -eq 0) {
    Write-Host "  Failed: $failCount" -ForegroundColor Green
}
else {
    Write-Host "  Failed: $failCount" -ForegroundColor Red
}
Write-Host "  Total:  $($passCount + $failCount)" -ForegroundColor White

if ($failCount -eq 0) {
    Write-Host "`nAll tests passed!" -ForegroundColor Green
}
else {
    Write-Host "`n$failCount test(s) failed" -ForegroundColor Red
}

Write-Host "`n========================================" -ForegroundColor Cyan

# Return error code for CI/CD
exit $failCount