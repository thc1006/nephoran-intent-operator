# PowerShell script to validate golangci-lint configuration
# Compatible with Windows development environment

param(
    [string]$ConfigFile = ".golangci-fast.yml",
    [switch]$DryRun = $false,
    [switch]$Fix = $false,
    [switch]$Verbose = $false
)

Write-Host "🔍 Validating golangci-lint configuration for Nephoran Intent Operator" -ForegroundColor Cyan
Write-Host "Go Version: $(go version)" -ForegroundColor Green

# Check if golangci-lint is installed
$golangciPath = Get-Command golangci-lint -ErrorAction SilentlyContinue
if (-not $golangciPath) {
    Write-Host "❌ golangci-lint not found. Installing latest version..." -ForegroundColor Yellow
    
    # Install golangci-lint for Windows
    $tempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
    try {
        Write-Host "📥 Downloading golangci-lint installer..." -ForegroundColor Blue
        $installerUrl = "https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh"
        $installerPath = Join-Path $tempDir "install.sh"
        
        # Use curl if available, otherwise use PowerShell
        if (Get-Command curl -ErrorAction SilentlyContinue) {
            curl -sSfL $installerUrl -o $installerPath
            wsl bash $installerPath -b $env:GOPATH/bin
        } else {
            Write-Host "⚠️  Please install golangci-lint manually:" -ForegroundColor Yellow
            Write-Host "   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" -ForegroundColor White
            Write-Host "   Or download from: https://github.com/golangci/golangci-lint/releases" -ForegroundColor White
            exit 1
        }
    }
    finally {
        Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Validate configuration file exists
if (-not (Test-Path $ConfigFile)) {
    Write-Host "❌ Configuration file not found: $ConfigFile" -ForegroundColor Red
    exit 1
}

Write-Host "✅ Using configuration: $ConfigFile" -ForegroundColor Green

# Test configuration syntax
Write-Host "🔧 Validating configuration syntax..." -ForegroundColor Blue
$configTest = & golangci-lint config path -c $ConfigFile 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Configuration syntax error:" -ForegroundColor Red
    Write-Host $configTest -ForegroundColor Red
    exit 1
}
Write-Host "✅ Configuration syntax is valid" -ForegroundColor Green

# Show enabled linters
Write-Host "📋 Enabled linters:" -ForegroundColor Blue
& golangci-lint linters -c $ConfigFile | Select-String -Pattern "Enabled by your configuration"

# Dry run validation
if ($DryRun) {
    Write-Host "🧪 Running dry-run validation on sample files..." -ForegroundColor Blue
    
    # Find a few Go files to test
    $sampleFiles = Get-ChildItem -Path "." -Filter "*.go" -Recurse | 
                   Where-Object { $_.FullName -notmatch "(vendor|testdata|\.git)" } | 
                   Select-Object -First 5
    
    foreach ($file in $sampleFiles) {
        Write-Host "   Checking: $($file.Name)" -ForegroundColor Gray
        $result = & golangci-lint run -c $ConfigFile --no-config $file.FullName 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "   ✅ $($file.Name)" -ForegroundColor Green
        } else {
            Write-Host "   ⚠️  $($file.Name) has issues" -ForegroundColor Yellow
            if ($Verbose) {
                Write-Host $result -ForegroundColor Gray
            }
        }
    }
}

# Performance benchmark
Write-Host "⚡ Performance benchmark (timeout test)..." -ForegroundColor Blue
$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()

$benchArgs = @(
    "run"
    "-c", $ConfigFile
    "--timeout", "2m"
    "--issues-exit-code", "0"  # Don't fail on issues for benchmark
    "./..."
)

if ($Fix) {
    $benchArgs += "--fix"
    Write-Host "🔧 Auto-fix mode enabled" -ForegroundColor Yellow
}

$result = & golangci-lint @benchArgs 2>&1
$stopwatch.Stop()

$duration = $stopwatch.Elapsed.TotalSeconds
Write-Host "⏱️  Analysis completed in $([math]::Round($duration, 2)) seconds" -ForegroundColor Cyan

if ($duration -gt 300) {  # 5 minutes
    Write-Host "⚠️  Analysis took longer than expected. Consider optimizing configuration." -ForegroundColor Yellow
} elseif ($duration -lt 120) {  # 2 minutes
    Write-Host "🚀 Excellent performance!" -ForegroundColor Green
} else {
    Write-Host "👍 Good performance" -ForegroundColor Green
}

# Show summary
Write-Host "`n📊 Validation Summary:" -ForegroundColor Cyan
Write-Host "   Configuration: ✅ Valid" -ForegroundColor Green
Write-Host "   Performance: $([math]::Round($duration, 2))s" -ForegroundColor Blue
Write-Host "   Go Version: $(go version | Select-String -Pattern 'go\d+\.\d+\.\d+')" -ForegroundColor Blue

if ($result -match "found (\d+) issues") {
    $issueCount = $Matches[1]
    Write-Host "   Issues Found: $issueCount" -ForegroundColor $(if ([int]$issueCount -eq 0) { "Green" } else { "Yellow" })
}

Write-Host "`n🎯 Next steps:" -ForegroundColor Cyan
Write-Host "   • Run: golangci-lint run -c $ConfigFile" -ForegroundColor White
Write-Host "   • Fix: golangci-lint run -c $ConfigFile --fix" -ForegroundColor White
Write-Host "   • CI: Use this config in .github/workflows/" -ForegroundColor White

Write-Host "`n✨ Configuration validation completed!" -ForegroundColor Green