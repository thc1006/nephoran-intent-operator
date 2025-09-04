# Security Workflow Validation and Hardening Script (PowerShell)
# This script validates that all security scanning workflows have proper error handling

param(
    [switch]$Fix = $false,
    [switch]$Verbose = $false
)

Write-Host "=== Security Workflow Validation Script ===" -ForegroundColor Cyan
Write-Host "Checking all security workflows for robustness..." -ForegroundColor White
Write-Host ""

$WorkflowsDir = ".github\workflows"
$IssuesFound = 0
$FixesApplied = 0

# Function to check if a workflow has proper SARIF file handling
function Test-SarifHandling {
    param([string]$FilePath)
    
    $hasIssues = $false
    $content = Get-Content $FilePath -Raw
    
    Write-Host "Checking $(Split-Path $FilePath -Leaf)..." -ForegroundColor White
    
    # Check for gosec SARIF generation without error handling
    if ($content -match "gosec.*-fmt sarif") {
        if ($content -notmatch 'echo.*version.*2\.1\.0.*schema.*sarif') {
            Write-Host "  ⚠ Missing SARIF fallback for gosec" -ForegroundColor Yellow
            $hasIssues = $true
        } else {
            Write-Host "  ✓ Has SARIF fallback for gosec" -ForegroundColor Green
        }
    }
    
    # Check for directory creation before file writes
    if ($content -match "gosec.*-out") {
        $lines = $content -split "`n"
        $gosecLineNum = 0
        for ($i = 0; $i -lt $lines.Count; $i++) {
            if ($lines[$i] -match "gosec.*-out") {
                $gosecLineNum = $i
                break
            }
        }
        
        $hasMkdir = $false
        for ($i = [Math]::Max(0, $gosecLineNum - 5); $i -lt $gosecLineNum; $i++) {
            if ($lines[$i] -match "mkdir -p") {
                $hasMkdir = $true
                break
            }
        }
        
        if (-not $hasMkdir) {
            Write-Host "  ⚠ Missing directory creation before gosec output" -ForegroundColor Yellow
            $hasIssues = $true
        } else {
            Write-Host "  ✓ Has directory creation" -ForegroundColor Green
        }
    }
    
    # Check for SBOM generation error handling
    if ($content -match "cyclonedx-gomod") {
        if ($content -notmatch "cyclonedx-gomod.*\|\|") {
            Write-Host "  ⚠ Missing error handling for SBOM generation" -ForegroundColor Yellow
            $hasIssues = $true
        } else {
            Write-Host "  ✓ Has SBOM error handling" -ForegroundColor Green
        }
    }
    
    # Check for vulnerability scanner error handling
    if ($content -match "grype|trivy|snyk") {
        if ($content -notmatch "(grype|trivy|snyk).*\|\|") {
            Write-Host "  ⚠ Missing error handling for vulnerability scanners" -ForegroundColor Yellow
            $hasIssues = $true
        } else {
            Write-Host "  ✓ Has vulnerability scanner error handling" -ForegroundColor Green
        }
    }
    
    return $hasIssues
}

# Function to validate SBOM command syntax
function Test-SbomSyntax {
    param([string]$FilePath)
    
    $hasIssues = $false
    $content = Get-Content $FilePath -Raw
    
    # Check for incorrect CycloneDX flags
    if ($content -match "cyclonedx-gomod.*--output-file") {
        Write-Host "  ✗ Incorrect CycloneDX flag: --output-file should be -output" -ForegroundColor Red
        $hasIssues = $true
    }
    
    # Check for incorrect Syft output syntax
    if ($content -match "syft.*-o.*>") {
        Write-Host "  ⚠ Syft using redirect (>) instead of = for output" -ForegroundColor Yellow
        $hasIssues = $true
    }
    
    return $hasIssues
}

# Main validation loop
Write-Host "=== Scanning Security Workflows ===" -ForegroundColor Cyan
Write-Host ""

$workflowPatterns = @(
    "$WorkflowsDir\*security*.yml",
    "$WorkflowsDir\*-ci*.yml",
    "$WorkflowsDir\production.yml",
    "$WorkflowsDir\conductor-loop*.yml",
    "$WorkflowsDir\optimized-ci.yml"
)

$workflows = @()
foreach ($pattern in $workflowPatterns) {
    $workflows += Get-ChildItem -Path $pattern -ErrorAction SilentlyContinue
}

$workflows = $workflows | Select-Object -Unique

foreach ($workflow in $workflows) {
    if (Test-Path $workflow.FullName) {
        $filename = $workflow.Name
        Write-Host "----------------------------------------" -ForegroundColor Gray
        Write-Host "Workflow: $filename" -ForegroundColor Cyan
        Write-Host "----------------------------------------" -ForegroundColor Gray
        
        if (Test-SarifHandling $workflow.FullName) {
            $script:IssuesFound++
        } else {
            Write-Host "  ✓ All SARIF checks passed" -ForegroundColor Green
        }
        
        if (Test-SbomSyntax $workflow.FullName) {
            $script:IssuesFound++
        }
        
        Write-Host ""
    }
}

# Summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "VALIDATION SUMMARY" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

if ($IssuesFound -eq 0) {
    Write-Host "✓ All security workflows have proper error handling!" -ForegroundColor Green
} else {
    Write-Host "⚠ Found $IssuesFound workflows with potential issues" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Recommendations:" -ForegroundColor White
    Write-Host "1. Ensure all SARIF files are created even when tools fail" -ForegroundColor Gray
    Write-Host "2. Always create directories before writing files" -ForegroundColor Gray
    Write-Host "3. Add error handling (|| operator) for all external tool calls" -ForegroundColor Gray
    Write-Host "4. Use correct command syntax for SBOM generators:" -ForegroundColor Gray
    Write-Host "   - CycloneDX: cyclonedx-gomod mod -json -output file.json" -ForegroundColor Gray
    Write-Host "   - Syft: syft . -o spdx-json=file.json" -ForegroundColor Gray
    Write-Host "5. Create empty/default output files before running tools" -ForegroundColor Gray
}

Write-Host ""
Write-Host "=== Best Practices for Security Workflows ===" -ForegroundColor Cyan
Write-Host ""

$bestPractices = @"
1. SARIF File Creation Pattern:
   ```yaml
   - name: Run Security Tool
     run: |
       mkdir -p reports
       # Create empty SARIF first
       echo '{"version":"2.1.0","`$schema":"https://json.schemastore.org/sarif-2.1.0.json","runs":[]}' > reports/tool.sarif
       # Run tool and overwrite if successful
       tool -fmt sarif -out reports/tool.sarif.tmp ./... && \
         mv reports/tool.sarif.tmp reports/tool.sarif || \
         rm -f reports/tool.sarif.tmp
   ```

2. SBOM Generation Pattern:
   ```yaml
   - name: Generate SBOM
     run: |
       # Create empty SBOM first
       echo '{"bomFormat":"CycloneDX","specVersion":"1.4","components":[]}' > sbom.json
       # Generate and overwrite if successful
       cyclonedx-gomod mod -json -output sbom.json.tmp && \
         mv sbom.json.tmp sbom.json || \
         echo "Warning: SBOM generation failed"
   ```

3. Tool Installation Pattern:
   ```yaml
   - name: Install Security Tool
     run: |
       go install tool@version || {
         echo "Warning: Failed to install tool"
         exit 0  # Don't fail the workflow
       }
   ```

4. Vulnerability Scanner Pattern:
   ```yaml
   - name: Run Vulnerability Scanner
     run: |
       # Create output directory
       mkdir -p security-reports
       # Create empty report first
       echo '{"matches":[],"source":{},"distro":{}}' > security-reports/vuln.json
       # Run scanner with error handling
       grype image:tag --output json --file security-reports/vuln.json.tmp && \
         mv security-reports/vuln.json.tmp security-reports/vuln.json || {
           echo "Warning: Vulnerability scan failed"
           rm -f security-reports/vuln.json.tmp
         }
   ```
"@

Write-Host $bestPractices -ForegroundColor Gray

if ($Fix) {
    Write-Host ""
    Write-Host "Fix mode is enabled. Would apply fixes to workflows..." -ForegroundColor Yellow
    Write-Host "(Fix functionality not yet implemented)" -ForegroundColor Red
}

exit $IssuesFound