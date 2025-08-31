# Verify the duplicate fix worked
Write-Host "=== VERIFICATION: Checking for remaining duplicates ==="

$duplicateTypes = @(
    "IntentProcessingTestSuite",
    "APIEndpointTestSuite", 
    "UserJourneyTestSuite",
    "TestScheduler",
    "TestExecutor",
    "TestResultProcessor"
)

$allClean = $true

foreach ($type in $duplicateTypes) {
    $found = @()
    Get-ChildItem -Recurse -Filter "*.go" | ForEach-Object {
        $matches = Select-String -Path $_.FullName -Pattern "^type\s+$type\b" -ErrorAction SilentlyContinue
        if ($matches) {
            foreach ($match in $matches) {
                $found += "$($_.Name):$($match.LineNumber)"
            }
        }
    }
    
    if ($found.Count -gt 1) {
        Write-Host "❌ STILL DUPLICATE: $type found in: $($found -join ', ')"
        $allClean = $false
    } elseif ($found.Count -eq 1) {
        Write-Host "✅ OK: $type found once in $($found[0])"
    } else {
        Write-Host "⚠️  NOT FOUND: $type"
    }
}

if ($allClean) {
    Write-Host "`n🎉 ALL DUPLICATES RESOLVED!"
} else {
    Write-Host "`n❌ DUPLICATES STILL EXIST - Manual intervention required"
}

Write-Host "`n=== Testing build ==="
try {
    $buildResult = go build ./... 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✅ Build successful"
    } else {
        Write-Host "❌ Build failed:"
        Write-Host $buildResult
    }
} catch {
    Write-Host "❌ Build test failed: $($_.Exception.Message)"
}