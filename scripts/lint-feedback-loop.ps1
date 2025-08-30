#!/usr/bin/env pwsh
# Fast feedback loop for iterative lint fixing
# Usage: .\lint-feedback-loop.ps1 [options]

param(
    [string]$LinterName = "",
    [string]$TargetPath = ".",
    [int]$MaxIterations = 10,
    [switch]$AutoFix = $false,
    [switch]$Interactive = $true,
    [switch]$ContinueOnFail = $false,
    [int]$DelayBetweenRuns = 2
)

$ErrorActionPreference = "Continue"

# Configuration
$Colors = @{
    Success = "Green"
    Error = "Red"
    Warning = "Yellow"
    Info = "Cyan"
    Debug = "Gray"
    Prompt = "Magenta"
}

function Write-ColorOutput {
    param([string]$Message, [string]$Color = "White")
    Write-Host $Message -ForegroundColor $Colors.$Color
}

function Show-Progress {
    param([int]$Current, [int]$Total, [string]$Status)
    $percent = [math]::Round(($Current / $Total) * 100, 1)
    $progressBar = "█" * [math]::Floor($percent / 5) + "░" * (20 - [math]::Floor($percent / 5))
    Write-ColorOutput "[$progressBar] $percent% - $Status" "Info"
}

function Get-UserChoice {
    param([string]$Prompt, [string[]]$Options, [int]$Default = 0)
    
    if (-not $Interactive) {
        return $Default
    }
    
    Write-ColorOutput $Prompt "Prompt"
    for ($i = 0; $i -lt $Options.Count; $i++) {
        $marker = if ($i -eq $Default) { "* " } else { "  " }
        Write-ColorOutput "$marker$($i + 1). $($Options[$i])" "Debug"
    }
    
    do {
        $input = Read-Host "Enter choice (1-$($Options.Count)) [default: $($Default + 1)]"
        if ([string]::IsNullOrWhiteSpace($input)) {
            return $Default
        }
        $choice = $null
        if ([int]::TryParse($input, [ref]$choice) -and $choice -ge 1 -and $choice -le $Options.Count) {
            return $choice - 1
        }
        Write-ColorOutput "Invalid choice. Please enter a number between 1 and $($Options.Count)" "Warning"
    } while ($true)
}

# Available linters
$AllLinters = @(
    "revive", "staticcheck", "govet", "ineffassign", "errcheck", 
    "gocritic", "misspell", "unparam", "unconvert", "prealloc", "gosec"
)

# Determine which linters to run
$LintersToRun = if ($LinterName) { 
    if ($LinterName -notin $AllLinters) {
        Write-ColorOutput "Invalid linter: $LinterName" "Error"
        Write-ColorOutput "Valid options: $($AllLinters -join ', ')" "Debug"
        exit 1
    }
    @($LinterName) 
} else { 
    $AllLinters 
}

Write-ColorOutput "🔄 LINT FEEDBACK LOOP" "Info"
Write-ColorOutput "=====================" "Info"
Write-ColorOutput "Linters: $($LintersToRun -join ', ')" "Debug"
Write-ColorOutput "Path: $TargetPath" "Debug"
Write-ColorOutput "Max iterations: $MaxIterations" "Debug"
Write-ColorOutput "Auto-fix: $AutoFix" "Debug"
Write-ColorOutput "Interactive: $Interactive" "Debug"
Write-ColorOutput ""

$iteration = 1
$overallSuccess = $false
$results = @()

while ($iteration -le $MaxIterations) {
    Write-ColorOutput "📍 Iteration $iteration of $MaxIterations" "Info"
    Show-Progress -Current $iteration -Total $MaxIterations -Status "Running linters"
    
    $iterationResults = @{}
    $iterationSuccess = $true
    
    foreach ($linter in $LintersToRun) {
        Write-ColorOutput "Running $linter..." "Debug"
        
        $startTime = Get-Date
        
        # Build golangci-lint command
        $cmd = @(
            "golangci-lint", "run",
            "--disable-all",
            "--enable", $linter,
            "--timeout", "2m",
            "--out-format", "colored-line-number"
        )
        
        if ($AutoFix) {
            $cmd += "--fix"
        }
        
        $cmd += $TargetPath
        
        try {
            $output = & $cmd[0] $cmd[1..($cmd.Length-1)] 2>&1
            $exitCode = $LASTEXITCODE
            $endTime = Get-Date
            $duration = ($endTime - $startTime).TotalSeconds
            
            $iterationResults[$linter] = @{
                ExitCode = $exitCode
                Duration = $duration
                Output = $output -join "`n"
                Success = ($exitCode -eq 0)
            }
            
            if ($exitCode -eq 0) {
                Write-ColorOutput "  ✅ $linter passed" "Success"
            } else {
                Write-ColorOutput "  ❌ $linter found issues" "Error"
                $iterationSuccess = $false
                
                # Show first few issues
                $issueLines = @($output | Where-Object { $_ -match ':(line \d+|col \d+)' } | Select-Object -First 5)
                if ($issueLines.Count -gt 0) {
                    Write-ColorOutput "    Sample issues:" "Warning"
                    $issueLines | ForEach-Object { Write-ColorOutput "    $_" "Warning" }
                }
            }
            
        } catch {
            Write-ColorOutput "  ❌ $linter failed with error: $_" "Error"
            $iterationResults[$linter] = @{
                ExitCode = 1
                Duration = 0
                Output = "Error: $_"
                Success = $false
            }
            $iterationSuccess = $false
        }
    }
    
    $results += @{
        Iteration = $iteration
        Success = $iterationSuccess
        Results = $iterationResults
        Timestamp = Get-Date
    }
    
    if ($iterationSuccess) {
        Write-ColorOutput "`n🎉 All linters passed on iteration $iteration!" "Success"
        $overallSuccess = $true
        break
    }
    
    # Interactive decision point
    if ($Interactive -and $iteration -lt $MaxIterations) {
        Write-ColorOutput ""
        $choices = @(
            "Continue to next iteration",
            "Run with --fix flag once",
            "Show detailed output",
            "Change target path",
            "Skip failing linters",
            "Exit loop"
        )
        
        $choice = Get-UserChoice -Prompt "What would you like to do?" -Options $choices -Default 0
        
        switch ($choice) {
            0 { 
                Write-ColorOutput "Continuing to iteration $($iteration + 1)..." "Info"
            }
            1 { 
                Write-ColorOutput "Running with --fix flag..." "Info"
                $AutoFix = $true
            }
            2 { 
                Write-ColorOutput "`nDetailed output for iteration $iteration:" "Info"
                foreach ($linter in $LintersToRun) {
                    if (-not $iterationResults[$linter].Success) {
                        Write-ColorOutput "`n$linter output:" "Warning"
                        Write-ColorOutput $iterationResults[$linter].Output "Debug"
                    }
                }
            }
            3 {
                $newPath = Read-Host "Enter new target path"
                if (-not [string]::IsNullOrWhiteSpace($newPath)) {
                    $TargetPath = $newPath
                    Write-ColorOutput "Target path changed to: $TargetPath" "Info"
                }
            }
            4 {
                $failedLinters = @($LintersToRun | Where-Object { -not $iterationResults[$_].Success })
                if ($failedLinters.Count -gt 0) {
                    Write-ColorOutput "Failed linters: $($failedLinters -join ', ')" "Warning"
                    $LintersToRun = @($LintersToRun | Where-Object { $iterationResults[$_].Success })
                    Write-ColorOutput "Continuing with: $($LintersToRun -join ', ')" "Info"
                }
            }
            5 { 
                Write-ColorOutput "Exiting feedback loop..." "Info"
                break
            }
        }
    } elseif (-not $ContinueOnFail -and -not $Interactive) {
        Write-ColorOutput "Non-interactive mode: stopping on failure" "Warning"
        break
    }
    
    if ($DelayBetweenRuns -gt 0) {
        Write-ColorOutput "Waiting $DelayBetweenRuns seconds..." "Debug"
        Start-Sleep -Seconds $DelayBetweenRuns
    }
    
    $iteration++
    Write-ColorOutput ""
}

# Final summary
Write-ColorOutput "📊 FEEDBACK LOOP SUMMARY" "Info"
Write-ColorOutput "=========================" "Info"
Write-ColorOutput "Total iterations: $($results.Count)" "Debug"
Write-ColorOutput "Overall success: $overallSuccess" $(if ($overallSuccess) { "Success" } else { "Error" })

if ($results.Count -gt 0) {
    Write-ColorOutput "`nIteration breakdown:" "Debug"
    for ($i = 0; $i -lt $results.Count; $i++) {
        $result = $results[$i]
        $status = if ($result.Success) { "✅ PASS" } else { "❌ FAIL" }
        Write-ColorOutput "  Iteration $($result.Iteration): $status" $(if ($result.Success) { "Success" } else { "Error" })
    }
}

# Generate recommendations
if (-not $overallSuccess) {
    Write-ColorOutput "`n💡 RECOMMENDATIONS:" "Info"
    
    # Find most problematic linter
    $linterIssues = @{}
    foreach ($result in $results) {
        foreach ($linter in $result.Results.Keys) {
            if (-not $result.Results[$linter].Success) {
                $linterIssues[$linter] = ($linterIssues[$linter] ?? 0) + 1
            }
        }
    }
    
    if ($linterIssues.Count -gt 0) {
        $mostProblematic = $linterIssues.GetEnumerator() | Sort-Object Value -Descending | Select-Object -First 3
        Write-ColorOutput "Most problematic linters:" "Warning"
        $mostProblematic | ForEach-Object { 
            Write-ColorOutput "  - $($_.Key): failed $($_.Value) times" "Warning"
        }
    }
    
    Write-ColorOutput "`nSuggested next steps:" "Info"
    Write-ColorOutput "  1. Run individual linter scripts for focused fixing" "Debug"
    Write-ColorOutput "  2. Use --fix flag for auto-fixable issues" "Debug"
    Write-ColorOutput "  3. Review specific linter documentation" "Debug"
    Write-ColorOutput "  4. Consider excluding problematic files temporarily" "Debug"
}

# Save results to file
$reportFile = "./test-results/lint-feedback-$(Get-Date -Format 'yyyyMMdd_HHmmss').json"
$reportData = @{
    timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
    success = $overallSuccess
    totalIterations = $results.Count
    maxIterations = $MaxIterations
    configuration = @{
        linters = $LintersToRun
        targetPath = $TargetPath
        autoFix = $AutoFix
        interactive = $Interactive
    }
    results = $results
}

try {
    New-Item -Path (Split-Path $reportFile) -ItemType Directory -Force -ErrorAction SilentlyContinue | Out-Null
    $reportData | ConvertTo-Json -Depth 10 | Out-File -FilePath $reportFile -Encoding UTF8
    Write-ColorOutput "Results saved to: $reportFile" "Debug"
} catch {
    Write-ColorOutput "Failed to save results: $_" "Warning"
}

exit $(if ($overallSuccess) { 0 } else { 1 })