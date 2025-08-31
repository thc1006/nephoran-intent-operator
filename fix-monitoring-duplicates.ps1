# Quick scan for duplicate types in pkg/monitoring
Get-ChildItem -Path "pkg/monitoring" -Recurse -Filter "*.go" | ForEach-Object {
    Write-Host "=== $($_.Name) ==="
    Select-String -Path $_.FullName -Pattern "^type (IntentProcessingTestSuite|APIEndpointTestSuite|UserJourneyTestSuite|TestScheduler|TestExecutor|TestResultProcessor)" | ForEach-Object {
        Write-Host "$($_.LineNumber): $($_.Line.Trim())"
    }
}