#!/usr/bin/env pwsh
# PowerShell script to fix json.RawMessage to map[string]interface{} conversion

$files = @(
    "pkg\shared\coordination_manager.go",
    "pkg\shared\recovery_manager.go"
)

foreach ($file in $files) {
    if (Test-Path $file) {
        Write-Host "Fixing $file"
        
        # Read content
        $content = Get-Content $file -Raw
        
        # Replace pattern: metadata := json.RawMessage("{}") followed by TransitionPhase call
        $content = $content -replace 'metadata := json\.RawMessage\("\{\}"\)\s+([^}]+)TransitionPhase\([^,]+,\s*[^,]+,\s*metadata\)', 'metadata := map[string]interface{}{} $1TransitionPhase($1, $1, metadata)'
        
        # More specific replacements for each pattern
        $content = $content -replace 'metadata := json\.RawMessage\("\{\}"\)([^}]*?)return cm\.stateManager\.TransitionPhase\(([^,]+),\s*([^,]+),\s*metadata\)', 'metadata := map[string]interface{}{}$1return cm.stateManager.TransitionPhase($2, $3, metadata)'
        
        $content = $content -replace 'metadata := json\.RawMessage\("\{\}"\)([^}]*?)return rm\.stateManager\.TransitionPhase\(([^,]+),\s*([^,]+),\s*metadata\)', 'metadata := map[string]interface{}{}$1return rm.stateManager.TransitionPhase($2, $3, metadata)'
        
        # Direct approach - replace specific patterns
        $content = $content -replace 'metadata := json\.RawMessage\("\{\}"\)', 'metadata := map[string]interface{}{}'
        
        # Write back
        Set-Content -Path $file -Value $content -NoNewline
        Write-Host "Fixed $file"
    } else {
        Write-Host "File not found: $file"
    }
}

Write-Host "Done!"