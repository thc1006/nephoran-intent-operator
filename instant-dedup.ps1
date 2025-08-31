# INSTANT DEDUPLICATION - Remove duplicate type declarations NOW
$ErrorActionPreference = "Stop"

# Find and process types.go and sla_monitoring_architecture.go specifically
$files = @()
Get-ChildItem -Recurse -Filter "types.go" | ForEach-Object { $files += $_ }
Get-ChildItem -Recurse -Filter "sla_monitoring_architecture.go" | ForEach-Object { $files += $_ }

if ($files.Count -eq 0) {
    Write-Host "ERROR: Cannot find types.go or sla_monitoring_architecture.go files"
    Get-ChildItem -Recurse -Filter "*.go" | Select-Object Name, Directory | Sort-Object Name
    exit 1
}

$duplicateTypes = @(
    "IntentProcessingTestSuite",
    "APIEndpointTestSuite", 
    "UserJourneyTestSuite",
    "TestScheduler",
    "TestExecutor",
    "TestResultProcessor"
)

# Strategy: Keep definitions in types.go, remove from sla_monitoring_architecture.go
$typesFile = $files | Where-Object { $_.Name -eq "types.go" } | Select-Object -First 1
$slaFile = $files | Where-Object { $_.Name -eq "sla_monitoring_architecture.go" } | Select-Object -First 1

Write-Host "Processing files:"
if ($typesFile) { Write-Host "  types.go: $($typesFile.FullName)" }
if ($slaFile) { Write-Host "  sla_monitoring_architecture.go: $($slaFile.FullName)" }

if ($slaFile) {
    Write-Host "`nRemoving duplicates from sla_monitoring_architecture.go..."
    $content = Get-Content $slaFile.FullName
    $newContent = @()
    $skipLines = 0
    
    for ($i = 0; $i -lt $content.Length; $i++) {
        if ($skipLines -gt 0) {
            $skipLines--
            continue
        }
        
        $line = $content[$i]
        $isDuplicate = $false
        
        foreach ($type in $duplicateTypes) {
            if ($line -match "^type\s+$type\b") {
                Write-Host "REMOVING: Line $($i+1): $line"
                $isDuplicate = $true
                
                # Calculate lines to skip (find end of type definition)
                $braceCount = 0
                $foundOpenBrace = $false
                
                for ($j = $i; $j -lt $content.Length; $j++) {
                    $checkLine = $content[$j]
                    
                    if ($checkLine -match "{") {
                        $foundOpenBrace = $true
                        $braceCount += ($checkLine.ToCharArray() | Where-Object { $_ -eq "{" }).Count
                    }
                    if ($checkLine -match "}") {
                        $braceCount -= ($checkLine.ToCharArray() | Where-Object { $_ -eq "}" }).Count
                    }
                    
                    if ($foundOpenBrace -and $braceCount -eq 0) {
                        $skipLines = $j - $i
                        break
                    }
                    
                    # Simple type declaration (single line)
                    if (-not $foundOpenBrace -and $j -gt $i -and 
                        ($checkLine -match "^type\s+" -or $checkLine -match "^func\s+" -or 
                         $checkLine -match "^var\s+" -or $checkLine -match "^const\s+" -or 
                         $checkLine.Trim() -eq "")) {
                        $skipLines = $j - $i - 1
                        break
                    }
                }
                break
            }
        }
        
        if (-not $isDuplicate) {
            $newContent += $line
        }
    }
    
    # Write cleaned content back
    $newContent | Set-Content -Path $slaFile.FullName -Force
    Write-Host "✓ Cleaned sla_monitoring_architecture.go"
}

Write-Host "`n=== DEDUPLICATION COMPLETE ==="
Write-Host "Run 'go build ./...' to verify the fix"