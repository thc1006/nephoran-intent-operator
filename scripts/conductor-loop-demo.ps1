# Conductor Loop Demo Script for Windows PowerShell
# Comprehensive demo showcasing conductor-loop functionality

param(
    [string]$LogLevel = "info",
    [string]$PorchUrl = "",
    [switch]$SkipBuild,
    [switch]$CleanFirst,
    [switch]$Help,
    [switch]$Verbose
)

# Configuration
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$BinaryPath = Join-Path $ProjectRoot "bin\conductor-loop.exe"
$ConfigDir = Join-Path $ProjectRoot "deployments\conductor-loop\config"
$LogDir = Join-Path $ProjectRoot "deployments\conductor-loop\logs"
$DataDir = Join-Path $ProjectRoot "data\conductor-loop"
$HandoffInDir = Join-Path $ProjectRoot "handoff\in"
$HandoffOutDir = Join-Path $ProjectRoot "handoff\out"

# Colors for output
$Red = "Red"
$Green = "Green"
$Yellow = "Yellow"
$Blue = "Blue"
$Cyan = "Cyan"

function Write-Log {
    param([string]$Message, [string]$Level = "INFO")
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $color = switch ($Level) {
        "ERROR" { $Red }
        "WARN" { $Yellow }
        "SUCCESS" { $Green }
        "INFO" { $Blue }
        default { $Blue }
    }
    Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $color
}

function Write-Error-Log {
    param([string]$Message)
    Write-Log $Message "ERROR"
}

function Write-Warn-Log {
    param([string]$Message)
    Write-Log $Message "WARN"
}

function Write-Success-Log {
    param([string]$Message)
    Write-Log $Message "SUCCESS"
}

function Show-Usage {
    @"
Conductor Loop Demo Script for Windows

USAGE:
    .\conductor-loop-demo.ps1 [OPTIONS]

OPTIONS:
    -LogLevel <level>       Log level (debug, info, warn, error) [default: info]
    -PorchUrl <url>        Porch server URL override
    -SkipBuild             Skip building the binary
    -CleanFirst            Clean artifacts before starting
    -Help                  Show this help message
    -Verbose               Enable verbose output

EXAMPLES:
    .\conductor-loop-demo.ps1                    # Run demo with defaults
    .\conductor-loop-demo.ps1 -LogLevel debug    # Run with debug logging
    .\conductor-loop-demo.ps1 -CleanFirst        # Clean and run demo
    .\conductor-loop-demo.ps1 -Help              # Show this help

DESCRIPTION:
    This script demonstrates the conductor-loop functionality by:
    1. Setting up the development environment
    2. Building the conductor-loop binary
    3. Starting mock services (if needed)
    4. Creating sample intent files
    5. Running conductor-loop to process the intents
    6. Showing the results

"@ | Write-Host -ForegroundColor Cyan
}

function Test-Dependencies {
    Write-Log "Checking dependencies..."
    
    $missing = @()
    
    # Check Go
    if (!(Get-Command "go.exe" -ErrorAction SilentlyContinue)) {
        $missing += "go"
    }
    
    # Check Docker (optional)
    if (!(Get-Command "docker.exe" -ErrorAction SilentlyContinue)) {
        Write-Warn-Log "Docker not found - Docker features will be disabled"
    }
    
    # Check Git (for version info)
    if (!(Get-Command "git.exe" -ErrorAction SilentlyContinue)) {
        Write-Warn-Log "Git not found - version info may be limited"
    }
    
    if ($missing.Count -gt 0) {
        Write-Error-Log "Missing required dependencies: $($missing -join ', ')"
        Write-Host "Please install the missing dependencies and try again." -ForegroundColor Red
        exit 1
    }
    
    Write-Success-Log "All required dependencies found"
}

function Initialize-Environment {
    Write-Log "Initializing conductor-loop environment..."
    
    # Create directories
    $dirs = @($ConfigDir, $LogDir, $DataDir, $HandoffInDir, $HandoffOutDir, 
              (Join-Path $ProjectRoot "bin"), (Join-Path $ProjectRoot ".coverage"))
    
    foreach ($dir in $dirs) {
        if (!(Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
        }
    }
    
    # Create default configuration
    $configFile = Join-Path $ConfigDir "config.json"
    if (!(Test-Path $configFile)) {
        Write-Log "Creating default configuration..."
        $config = @{
            server = @{
                host = "127.0.0.1"
                port = 8080
                metrics_port = 9090
            }
            logging = @{
                level = $LogLevel
                format = "text"
                output = "stdout"
            }
            conductor = @{
                handoff_in_path = "./handoff/in"
                handoff_out_path = "./handoff/out"
                poll_interval = "5s"
                batch_size = 5
                retry_attempts = 3
            }
            porch = @{
                endpoint = if ($PorchUrl) { $PorchUrl } else { "http://localhost:7007" }
                timeout = "30s"
                repository = "nf-packages"
            }
        } | ConvertTo-Json -Depth 4
        
        $config | Out-File -FilePath $configFile -Encoding UTF8
        Write-Success-Log "Configuration created: $configFile"
    }
    
    Create-SampleIntents
    Write-Success-Log "Environment initialized successfully!"
}

function Create-SampleIntents {
    Write-Log "Creating sample intent files for demo..."
    
    # Sample 1: Scale-up intent
    $scaleUpIntent = @{
        kind = "NetworkIntent"
        metadata = @{
            name = "demo-scale-up-nf-sim"
            namespace = "default"
            labels = @{
                "demo" = "conductor-loop"
                "action" = "scale-up"
            }
        }
        spec = @{
            target = "nf-sim"
            action = "scale"
            replicas = 3
            resources = @{
                cpu = "200m"
                memory = "256Mi"
            }
            reason = "Demo: Simulating load increase"
            priority = "high"
        }
    } | ConvertTo-Json -Depth 4
    
    $scaleUpPath = Join-Path $HandoffInDir "demo-scale-up-$(Get-Date -Format 'yyyyMMdd-HHmmss').json"
    $scaleUpIntent | Out-File -FilePath $scaleUpPath -Encoding UTF8
    
    # Sample 2: Scale-down intent
    $scaleDownIntent = @{
        kind = "NetworkIntent"
        metadata = @{
            name = "demo-scale-down-nf-sim"
            namespace = "default"
            labels = @{
                "demo" = "conductor-loop"
                "action" = "scale-down"
            }
        }
        spec = @{
            target = "nf-sim"
            action = "scale"
            replicas = 1
            resources = @{
                cpu = "100m"
                memory = "128Mi"
            }
            reason = "Demo: Simulating load decrease"
            priority = "normal"
        }
    } | ConvertTo-Json -Depth 4
    
    $scaleDownPath = Join-Path $HandoffInDir "demo-scale-down-$(Get-Date -Format 'yyyyMMdd-HHmmss').json"
    $scaleDownIntent | Out-File -FilePath $scaleDownPath -Encoding UTF8
    
    # Sample 3: Configuration update intent
    $configUpdateIntent = @{
        kind = "NetworkIntent"
        metadata = @{
            name = "demo-config-update-nf-sim"
            namespace = "default"
            labels = @{
                "demo" = "conductor-loop"
                "action" = "config-update"
            }
        }
        spec = @{
            target = "nf-sim"
            action = "configure"
            configuration = @{
                logging_level = "debug"
                metrics_enabled = $true
                health_check_interval = "15s"
            }
            reason = "Demo: Configuration update"
            priority = "low"
        }
    } | ConvertTo-Json -Depth 4
    
    $configUpdatePath = Join-Path $HandoffInDir "demo-config-update-$(Get-Date -Format 'yyyyMMdd-HHmmss').json"
    $configUpdateIntent | Out-File -FilePath $configUpdatePath -Encoding UTF8
    
    Write-Success-Log "Sample intent files created:"
    Write-Host "  - Scale-up: $scaleUpPath" -ForegroundColor Green
    Write-Host "  - Scale-down: $scaleDownPath" -ForegroundColor Green
    Write-Host "  - Config update: $configUpdatePath" -ForegroundColor Green
}

function Build-ConductorLoop {
    if ($SkipBuild -and (Test-Path $BinaryPath)) {
        Write-Log "Skipping build (binary exists and -SkipBuild specified)"
        return
    }
    
    Write-Log "Building conductor-loop binary..."
    
    Push-Location $ProjectRoot
    try {
        # Get version information
        $version = "dev"
        $commit = "unknown"
        if (Get-Command "git.exe" -ErrorAction SilentlyContinue) {
            try {
                $version = git describe --tags --always --dirty 2>$null
                if (!$version) { $version = "dev" }
                $commit = git rev-parse --short HEAD 2>$null
                if (!$commit) { $commit = "unknown" }
            }
            catch {
                Write-Warn-Log "Could not get git version info"
            }
        }
        
        $date = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
        
        # Set environment variables for build
        $env:CGO_ENABLED = "0"
        $env:GOOS = "windows"
        $env:GOARCH = "amd64"
        
        # Build command
        $ldflags = "-s -w -X main.version=$version -X main.commit=$commit -X main.date=$date"
        
        Write-Log "Building with version=$version commit=$commit"
        
        & go.exe build -ldflags $ldflags -o $BinaryPath ./cmd/conductor-loop
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success-Log "Binary built successfully: $BinaryPath"
            $fileInfo = Get-Item $BinaryPath
            Write-Host "  Size: $([math]::Round($fileInfo.Length / 1MB, 2)) MB" -ForegroundColor Green
        }
        else {
            throw "Build failed with exit code $LASTEXITCODE"
        }
    }
    catch {
        Write-Error-Log "Failed to build conductor-loop: $_"
        exit 1
    }
    finally {
        Pop-Location
    }
}

function Start-MockServices {
    Write-Log "Starting mock services for demo..."
    
    # Create a simple mock Porch server using Python if available
    if (Get-Command "python.exe" -ErrorAction SilentlyContinue) {
        $mockServerScript = @"
import http.server
import socketserver
import json
from urllib.parse import urlparse, parse_qs

class MockPorchHandler(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            response = json.dumps({"status": "ok", "service": "mock-porch"})
            self.wfile.write(response.encode())
        elif self.path.startswith('/api/porch/v1alpha1/repositories'):
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            response = json.dumps({
                "apiVersion": "porch.kpt.dev/v1alpha1",
                "kind": "RepositoryList",
                "items": [
                    {
                        "metadata": {"name": "nf-packages"},
                        "spec": {"type": "git"},
                        "status": {"ready": True}
                    }
                ]
            })
            self.wfile.write(response.encode())
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_POST(self):
        self.send_response(200)
        self.send_header('Content-type', 'application/json')
        self.end_headers()
        response = json.dumps({"status": "accepted", "message": "Mock response"})
        self.wfile.write(response.encode())

if __name__ == "__main__":
    PORT = 7007
    with socketserver.TCPServer(("", PORT), MockPorchHandler) as httpd:
        print(f"Mock Porch server running on port {PORT}")
        httpd.serve_forever()
"@
        
        $mockScriptPath = Join-Path $DataDir "mock-porch.py"
        $mockServerScript | Out-File -FilePath $mockScriptPath -Encoding UTF8
        
        # Start mock server in background
        Write-Log "Starting mock Porch server on port 7007..."
        $mockProcess = Start-Process -FilePath "python.exe" -ArgumentList $mockScriptPath -PassThru -WindowStyle Hidden
        
        # Give it a moment to start
        Start-Sleep -Seconds 2
        
        # Test if mock server is responding
        try {
            $response = Invoke-RestMethod -Uri "http://localhost:7007/health" -TimeoutSec 5
            if ($response.status -eq "ok") {
                Write-Success-Log "Mock Porch server is running"
                return $mockProcess
            }
        }
        catch {
            Write-Warn-Log "Mock Porch server may not be responding correctly"
        }
    }
    else {
        Write-Warn-Log "Python not found - mock Porch server will not be started"
        Write-Host "  The demo will still run but Porch integration features will not work" -ForegroundColor Yellow
    }
    
    return $null
}

function Run-Demo {
    Write-Log "Starting conductor-loop demo..."
    
    $configFile = Join-Path $ConfigDir "config.json"
    $logFile = Join-Path $LogDir "conductor-loop-demo.log"
    
    if (!(Test-Path $configFile)) {
        Write-Error-Log "Configuration file not found: $configFile"
        exit 1
    }
    
    # Set environment variables
    $env:CONDUCTOR_LOG_LEVEL = $LogLevel
    $env:CONDUCTOR_DATA_PATH = $DataDir
    $env:CONDUCTOR_CONFIG_PATH = $configFile
    if ($PorchUrl) {
        $env:CONDUCTOR_PORCH_ENDPOINT = $PorchUrl
    }
    
    Write-Log "Configuration:"
    Write-Host "  Config file: $configFile" -ForegroundColor Cyan
    Write-Host "  Log level: $LogLevel" -ForegroundColor Cyan
    Write-Host "  Data path: $DataDir" -ForegroundColor Cyan
    Write-Host "  Handoff IN: $HandoffInDir" -ForegroundColor Cyan
    Write-Host "  Handoff OUT: $HandoffOutDir" -ForegroundColor Cyan
    Write-Host "  Log file: $logFile" -ForegroundColor Cyan
    
    Write-Host "`n" + "="*60 -ForegroundColor Yellow
    Write-Host "Starting Conductor Loop - Press Ctrl+C to stop" -ForegroundColor Yellow
    Write-Host "="*60 -ForegroundColor Yellow
    
    # Start conductor-loop
    try {
        if ($Verbose) {
            & $BinaryPath --config=$configFile --log-level=$LogLevel
        }
        else {
            & $BinaryPath --config=$configFile --log-level=$LogLevel | Tee-Object -FilePath $logFile
        }
    }
    catch {
        Write-Error-Log "Failed to run conductor-loop: $_"
        exit 1
    }
}

function Watch-HandoffDirectories {
    Write-Log "Monitoring handoff directories for changes..."
    Write-Host "  IN:  $HandoffInDir" -ForegroundColor Green
    Write-Host "  OUT: $HandoffOutDir" -ForegroundColor Green
    Write-Host ""
    Write-Host "You can add files to the IN directory to see them processed..." -ForegroundColor Cyan
    
    # Start a background job to monitor the directories
    $watchJob = Start-Job -ScriptBlock {
        param($InDir, $OutDir)
        
        $inFiles = @{}
        $outFiles = @{}
        
        while ($true) {
            # Check for new files in IN directory
            if (Test-Path $InDir) {
                Get-ChildItem -Path $InDir -File | ForEach-Object {
                    if (!$inFiles.ContainsKey($_.Name)) {
                        $inFiles[$_.Name] = $_.LastWriteTime
                        Write-Host "NEW IN: $($_.Name)" -ForegroundColor Green
                    }
                }
            }
            
            # Check for new files in OUT directory
            if (Test-Path $OutDir) {
                Get-ChildItem -Path $OutDir -File | ForEach-Object {
                    if (!$outFiles.ContainsKey($_.Name)) {
                        $outFiles[$_.Name] = $_.LastWriteTime
                        Write-Host "NEW OUT: $($_.Name)" -ForegroundColor Blue
                    }
                }
            }
            
            Start-Sleep -Seconds 2
        }
    } -ArgumentList $HandoffInDir, $HandoffOutDir
    
    return $watchJob
}

function Show-DemoStatus {
    Write-Host "`n" + "="*60 -ForegroundColor Cyan
    Write-Host "CONDUCTOR LOOP DEMO STATUS" -ForegroundColor Cyan
    Write-Host "="*60 -ForegroundColor Cyan
    
    Write-Host "`nFiles in handoff/in:" -ForegroundColor Yellow
    if (Test-Path $HandoffInDir) {
        $inFiles = Get-ChildItem -Path $HandoffInDir -File
        if ($inFiles) {
            $inFiles | ForEach-Object {
                Write-Host "  - $($_.Name) ($($_.Length) bytes)" -ForegroundColor Green
            }
        }
        else {
            Write-Host "  (no files)" -ForegroundColor Gray
        }
    }
    
    Write-Host "`nFiles in handoff/out:" -ForegroundColor Yellow
    if (Test-Path $HandoffOutDir) {
        $outFiles = Get-ChildItem -Path $HandoffOutDir -File
        if ($outFiles) {
            $outFiles | ForEach-Object {
                Write-Host "  - $($_.Name) ($($_.Length) bytes)" -ForegroundColor Blue
            }
        }
        else {
            Write-Host "  (no files)" -ForegroundColor Gray
        }
    }
    
    Write-Host "`nAPI Endpoints:" -ForegroundColor Yellow
    Write-Host "  Health:  http://localhost:8080/healthz" -ForegroundColor Cyan
    Write-Host "  Metrics: http://localhost:9090/metrics" -ForegroundColor Cyan
    Write-Host "  Ready:   http://localhost:8080/readyz" -ForegroundColor Cyan
}

function Cleanup-Demo {
    Write-Log "Cleaning up demo resources..."
    
    # Stop any running processes
    Get-Process -Name "conductor-loop" -ErrorAction SilentlyContinue | Stop-Process -Force
    Get-Process -Name "python" -ErrorAction SilentlyContinue | Where-Object { $_.CommandLine -like "*mock-porch.py*" } | Stop-Process -Force
    
    if ($CleanFirst) {
        Write-Log "Removing build artifacts and logs..."
        if (Test-Path $BinaryPath) { Remove-Item $BinaryPath -Force }
        if (Test-Path $LogDir) { Remove-Item $LogDir -Recurse -Force }
        if (Test-Path (Join-Path $ProjectRoot ".coverage")) { 
            Remove-Item (Join-Path $ProjectRoot ".coverage") -Recurse -Force 
        }
    }
    
    Write-Success-Log "Cleanup complete"
}

# Main execution
function Main {
    if ($Help) {
        Show-Usage
        exit 0
    }
    
    Write-Host "Conductor Loop Demo for Windows PowerShell" -ForegroundColor Cyan
    Write-Host "===========================================" -ForegroundColor Cyan
    
    try {
        # Cleanup first if requested
        if ($CleanFirst) {
            Cleanup-Demo
        }
        
        # Initialize
        Test-Dependencies
        Initialize-Environment
        Build-ConductorLoop
        
        # Start mock services
        $mockProcess = Start-MockServices
        
        Write-Host "`nDemo environment ready!" -ForegroundColor Green
        Show-DemoStatus
        
        # Start file monitoring in background
        $watchJob = Watch-HandoffDirectories
        
        Write-Host "`nPress Enter to start conductor-loop (or Ctrl+C to exit)..." -ForegroundColor Yellow
        Read-Host
        
        # Run the demo
        Run-Demo
    }
    catch {
        Write-Error-Log "Demo failed: $_"
        exit 1
    }
    finally {
        # Cleanup
        if ($mockProcess) {
            Stop-Process -Id $mockProcess.Id -Force -ErrorAction SilentlyContinue
        }
        if ($watchJob) {
            Stop-Job -Job $watchJob -ErrorAction SilentlyContinue
            Remove-Job -Job $watchJob -ErrorAction SilentlyContinue
        }
    }
}

# Handle Ctrl+C gracefully
$null = Register-EngineEvent -SourceIdentifier PowerShell.Exiting -Action {
    Write-Host "`nDemo interrupted - cleaning up..." -ForegroundColor Yellow
    Cleanup-Demo
}

# Run the demo
Main