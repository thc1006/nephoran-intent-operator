# ULTRA SPEED BUILD VERIFICATION SCRIPT
# Runs go build on all packages, captures compilation errors, categorizes by type

param(
    [switch]$Verbose,
    [switch]$FailFast,
    [string]$OutputFile = "build-verification-report.json"
)

# Error categories for classification
$ErrorCategories = @{
    "ImportCycle" = @("import cycle not allowed")
    "UndefinedType" = @("undefined:", "not declared by package", "cannot use.*as.*in argument")  
    "MissingPackage" = @("cannot find package", "no Go files in")
    "SyntaxError" = @("syntax error:", "expected", "unexpected")
    "BuildConstraint" = @("build constraints exclude all Go files")
    "InterfaceImplementation" = @("does not implement", "missing method")
    "Conversion" = @("cannot convert", "cannot use.*as.*in return")
    "Visibility" = @("cannot refer to unexported", "undefined.*not exported")
    "GoModule" = @("go.mod", "module", "require")
    "CGO" = @("cgo", "C\\.")
    "Generic" = @("type.*does not exist", "instantiation")
}

Write-Host "🚀 ULTRA SPEED BUILD VERIFICATION - Starting..." -ForegroundColor Cyan
Write-Host "Target: Nephoran Intent Operator (Kubernetes Cloud-Native)" -ForegroundColor Yellow

# Build results tracking
$BuildResults = @{
    TotalPackages = 0
    SuccessfulBuilds = 0
    FailedBuilds = 0
    ErrorsByCategory = @{}
    PackageResults = @()
    StartTime = Get-Date
    EndTime = $null
}

# Initialize error category counters
foreach ($category in $ErrorCategories.Keys) {
    $BuildResults.ErrorsByCategory[$category] = 0
}
$BuildResults.ErrorsByCategory["Unknown"] = 0

function Classify-Error {
    param([string]$ErrorText)
    
    foreach ($category in $ErrorCategories.Keys) {
        foreach ($pattern in $ErrorCategories[$category]) {
            if ($ErrorText -match $pattern) {
                return $category
            }
        }
    }
    return "Unknown"
}

function Get-AllGoPackages {
    $packages = @()
    
    # Main module (root)
    if (Test-Path "*.go") {
        $packages += "."
    }
    
    # All cmd packages
    $cmdPackages = Get-ChildItem -Path "cmd" -Directory -ErrorAction SilentlyContinue | ForEach-Object {
        if (Test-Path (Join-Path $_.FullName "*.go")) {
            "./cmd/$($_.Name)"
        }
    }
    $packages += $cmdPackages
    
    # All pkg packages (recursively)
    if (Test-Path "pkg") {
        $pkgPackages = Get-ChildItem -Path "pkg" -Recurse -Directory | ForEach-Object {
            $goFiles = Get-ChildItem -Path $_.FullName -Filter "*.go" -File
            if ($goFiles.Count -gt 0) {
                $relativePath = $_.FullName.Replace((Get-Location).Path, "").Replace("\", "/").TrimStart("/")
                "./$relativePath"
            }
        } | Where-Object { $_ -ne $null }
        $packages += $pkgPackages
    }
    
    # Controllers directory  
    if (Test-Path "controllers" -PathType Container) {
        $controllersFiles = Get-ChildItem -Path "controllers" -Filter "*.go" -File
        if ($controllersFiles.Count -gt 0) {
            $packages += "./controllers"
        }
    }
    
    # Internal packages
    if (Test-Path "internal") {
        $internalPackages = Get-ChildItem -Path "internal" -Recurse -Directory | ForEach-Object {
            $goFiles = Get-ChildItem -Path $_.FullName -Filter "*.go" -File
            if ($goFiles.Count -gt 0) {
                $relativePath = $_.FullName.Replace((Get-Location).Path, "").Replace("\", "/").TrimStart("/")
                "./$relativePath"
            }
        } | Where-Object { $_ -ne $null }
        $packages += $internalPackages
    }
    
    return $packages | Sort-Object -Unique
}

function Test-PackageBuild {
    param([string]$Package)
    
    Write-Host "Building: $Package" -ForegroundColor Gray
    
    # Use cmd /c to capture output properly
    $result = cmd /c "go build -v `"$Package`" 2>&1"
    $exitCode = $LASTEXITCODE
    
    $success = ($exitCode -eq 0)
    $output = ""
    $errors = ""
    
    if ($result) {
        $resultText = $result -join "`n"
        if ($success) {
            $output = $resultText
        } else {
            $errors = $resultText
        }
    }
    
    return @{
        Package = $Package
        Success = $success
        Output = $output
        Errors = $errors
        ExitCode = $exitCode
    }
}

# Get all Go packages
Write-Host "🔍 Discovering Go packages..." -ForegroundColor Yellow
$allPackages = Get-AllGoPackages

Write-Host "Found $($allPackages.Count) Go packages to build" -ForegroundColor Green
if ($Verbose) {
    Write-Host "Packages:" -ForegroundColor Gray
    $allPackages | ForEach-Object { Write-Host "  $_" -ForegroundColor DarkGray }
}

$BuildResults.TotalPackages = $allPackages.Count

# Build each package
Write-Host "`n🔨 Building packages..." -ForegroundColor Yellow

foreach ($package in $allPackages) {
    $result = Test-PackageBuild -Package $package
    
    $packageResult = @{
        Package = $package
        Success = $result.Success
        Output = $result.Output
        Errors = $result.Errors
        ExitCode = $result.ExitCode
        ErrorCategories = @()
    }
    
    if ($result.Success) {
        $BuildResults.SuccessfulBuilds++
        Write-Host "✅ $package" -ForegroundColor Green
    } else {
        $BuildResults.FailedBuilds++
        Write-Host "❌ $package" -ForegroundColor Red
        
        if ($Verbose -and $result.Errors) {
            Write-Host "   Error: $($result.Errors)" -ForegroundColor DarkRed
        }
        
        # Categorize errors
        if ($result.Errors) {
            $errorLines = $result.Errors -split "`n" | Where-Object { $_.Trim() -ne "" }
            foreach ($errorLine in $errorLines) {
                $category = Classify-Error -ErrorText $errorLine
                $BuildResults.ErrorsByCategory[$category]++
                if ($packageResult.ErrorCategories -notcontains $category) {
                    $packageResult.ErrorCategories += $category
                }
            }
        }
        
        # Fail fast if requested
        if ($FailFast) {
            Write-Host "⏹️  Stopping due to -FailFast flag" -ForegroundColor Red
            break
        }
    }
    
    $BuildResults.PackageResults += $packageResult
}

$BuildResults.EndTime = Get-Date
$duration = ($BuildResults.EndTime - $BuildResults.StartTime).TotalSeconds

# Generate summary
Write-Host "`n📊 BUILD VERIFICATION SUMMARY" -ForegroundColor Cyan
Write-Host "=" * 50 -ForegroundColor Cyan
Write-Host "Total Packages: $($BuildResults.TotalPackages)" -ForegroundColor White
Write-Host "Successful Builds: $($BuildResults.SuccessfulBuilds)" -ForegroundColor Green  
Write-Host "Failed Builds: $($BuildResults.FailedBuilds)" -ForegroundColor Red
Write-Host "Success Rate: $([math]::Round(($BuildResults.SuccessfulBuilds / $BuildResults.TotalPackages) * 100, 1))%" -ForegroundColor Yellow
Write-Host "Duration: $([math]::Round($duration, 2)) seconds" -ForegroundColor Gray

if ($BuildResults.FailedBuilds -gt 0) {
    Write-Host "`n🏷️  ERROR CATEGORIES:" -ForegroundColor Red
    foreach ($category in $BuildResults.ErrorsByCategory.Keys | Sort-Object) {
        $count = $BuildResults.ErrorsByCategory[$category]
        if ($count -gt 0) {
            Write-Host "  $category`: $count" -ForegroundColor Yellow
        }
    }
    
    # Show failed packages
    Write-Host "`n❌ FAILED PACKAGES:" -ForegroundColor Red
    $BuildResults.PackageResults | Where-Object { -not $_.Success } | ForEach-Object {
        $categoryStr = if ($_.ErrorCategories.Count -gt 0) { " (" + ($_.ErrorCategories -join ", ") + ")" } else { "" }
        Write-Host "  $($_.Package)$categoryStr" -ForegroundColor Red
        
        if ($Verbose -and $_.Errors) {
            $_.Errors -split "`n" | Where-Object { $_.Trim() -ne "" } | ForEach-Object {
                Write-Host "    $_" -ForegroundColor DarkRed
            }
        }
    }
}

# Export detailed results to JSON
$jsonReport = $BuildResults | ConvertTo-Json -Depth 10 -Compress:$false
$jsonReport | Out-File -FilePath $OutputFile -Encoding UTF8
Write-Host "`n📄 Detailed report saved to: $OutputFile" -ForegroundColor Cyan

# Return appropriate exit code
if ($BuildResults.FailedBuilds -eq 0) {
    Write-Host "`n🎉 ALL BUILDS SUCCESSFUL!" -ForegroundColor Green
    exit 0
} else {
    Write-Host "`n⚠️  BUILD VERIFICATION COMPLETED WITH ERRORS" -ForegroundColor Red
    exit 1
}