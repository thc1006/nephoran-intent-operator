# Search for duplicate type declarations across all Go files
$duplicateTypes = @(
    "IntentProcessingTestSuite",
    "APIEndpointTestSuite", 
    "UserJourneyTestSuite",
    "TestScheduler",
    "TestExecutor",
    "TestResultProcessor"
)

Write-Host "=== Searching for duplicate types ==="

foreach ($type in $duplicateTypes) {
    Write-Host "`n--- Searching for: $type ---"
    Get-ChildItem -Recurse -Filter "*.go" | ForEach-Object {
        $matches = Select-String -Path $_.FullName -Pattern "type\s+$type" -ErrorAction SilentlyContinue
        if ($matches) {
            foreach ($match in $matches) {
                Write-Host "$($_.FullName):$($match.LineNumber): $($match.Line.Trim())"
            }
        }
    }
}

Write-Host "`n=== File inventory ==="
Get-ChildItem -Recurse -Filter "*.go" | Select-Object Name, Directory