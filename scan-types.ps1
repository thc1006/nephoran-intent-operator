# Find all Go files in pkg/monitoring and their type declarations
$files = Get-ChildItem -Path "pkg/monitoring" -Recurse -Filter "*.go" -ErrorAction SilentlyContinue

if ($files) {
    foreach ($file in $files) {
        Write-Host "`n=== $($file.FullName) ==="
        $content = Get-Content $file.FullName -Raw -ErrorAction SilentlyContinue
        if ($content -match "type\s+(IntentProcessingTestSuite|APIEndpointTestSuite|UserJourneyTestSuite|TestScheduler|TestExecutor|TestResultProcessor)") {
            $lines = Get-Content $file.FullName
            for ($i = 0; $i -lt $lines.Count; $i++) {
                if ($lines[$i] -match "^type\s+(IntentProcessingTestSuite|APIEndpointTestSuite|UserJourneyTestSuite|TestScheduler|TestExecutor|TestResultProcessor)") {
                    Write-Host "Line $($i+1): $($lines[$i])"
                }
            }
        }
    }
} else {
    Write-Host "No Go files found in pkg/monitoring"
    Write-Host "Current directory contents:"
    Get-ChildItem -Path "." -Recurse -Name -Include "*.go" | Where-Object { $_ -like "*monitoring*" }
}