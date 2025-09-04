# EMERGENCY: Find and fix duplicate type declarations immediately
$ErrorActionPreference = "Continue"

Write-Host "=== EMERGENCY DUPLICATE TYPE FIX ==="
Write-Host "Timestamp: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"

# Step 1: Find all Go files
$goFiles = Get-ChildItem -Recurse -Filter "*.go" -ErrorAction SilentlyContinue
Write-Host "`nFound $($goFiles.Count) Go files"

# Step 2: Search for duplicate types
$duplicateTypes = @(
    "IntentProcessingTestSuite",
    "APIEndpointTestSuite", 
    "UserJourneyTestSuite",
    "TestScheduler",
    "TestExecutor",
    "TestResultProcessor"
)

$typeLocations = @{}

foreach ($type in $duplicateTypes) {
    $typeLocations[$type] = @()
    foreach ($file in $goFiles) {
        $content = Get-Content $file.FullName -ErrorAction SilentlyContinue
        if ($content) {
            for ($i = 0; $i -lt $content.Length; $i++) {
                if ($content[$i] -match "^type\s+$type\b") {
                    $typeLocations[$type] += @{
                        File = $file.FullName
                        Line = $i + 1
                        Content = $content[$i]
                    }
                    Write-Host "FOUND: $type at $($file.Name):$($i+1)"
                }
            }
        }
    }
}

# Step 3: Report duplicates and prepare fixes
foreach ($type in $duplicateTypes) {
    if ($typeLocations[$type].Count -gt 1) {
        Write-Host "`n!!! DUPLICATE FOUND: $type ($($typeLocations[$type].Count) occurrences) !!!"
        
        # Keep the first occurrence, mark others for removal
        for ($i = 1; $i -lt $typeLocations[$type].Count; $i++) {
            $duplicate = $typeLocations[$type][$i]
            Write-Host "REMOVING: $($duplicate.File):$($duplicate.Line)"
            
            # Read file, remove duplicate type definition
            $fileContent = Get-Content $duplicate.File
            
            # Find the end of the type definition (next type or end of file)
            $startLine = $duplicate.Line - 1
            $endLine = $startLine
            
            # Find the end of this type definition
            $braceCount = 0
            $foundOpenBrace = $false
            
            for ($j = $startLine; $j -lt $fileContent.Length; $j++) {
                $line = $fileContent[$j]
                
                if ($line -match "{") {
                    $foundOpenBrace = $true
                    $braceCount += ($line.ToCharArray() | Where-Object { $_ -eq "{" }).Count
                }
                if ($line -match "}") {
                    $braceCount -= ($line.ToCharArray() | Where-Object { $_ -eq "}" }).Count
                }
                
                if ($foundOpenBrace -and $braceCount -eq 0) {
                    $endLine = $j
                    break
                }
                
                # If it's a simple type without braces, just remove the single line
                if (-not $foundOpenBrace -and $j -gt $startLine -and ($line -match "^type\s+" -or $line -match "^func\s+" -or $line -match "^var\s+" -or $line -match "^const\s+")) {
                    $endLine = $j - 1
                    break
                }
            }
            
            Write-Host "Removing lines $($startLine+1) to $($endLine+1) from $($duplicate.File)"
            
            # Create new content without the duplicate lines
            $newContent = @()
            for ($k = 0; $k -lt $fileContent.Length; $k++) {
                if ($k -lt $startLine -or $k -gt $endLine) {
                    $newContent += $fileContent[$k]
                }
            }
            
            # Write back to file
            try {
                $newContent | Set-Content -Path $duplicate.File -Force
                Write-Host "✓ Fixed: $($duplicate.File)"
            } catch {
                Write-Host "✗ Error fixing $($duplicate.File): $($_.Exception.Message)"
            }
        }
    } elseif ($typeLocations[$type].Count -eq 1) {
        Write-Host "✓ OK: $type (single occurrence)"
    } else {
        Write-Host "? Not found: $type"
    }
}

Write-Host "`n=== FIX COMPLETE ==="
Write-Host "Please run 'go build ./...' to verify fixes"