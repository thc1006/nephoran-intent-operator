# Test script to verify CRD generation works correctly
# This script was created to validate the CRD generation fixes

Write-Host "🔍 Testing CRD Generation Process..." -ForegroundColor Cyan

# Check if controller-gen is installed
$controllerGenPath = "$env:GOPATH\bin\controller-gen.exe"
if (-not (Test-Path $controllerGenPath)) {
    Write-Host "❌ controller-gen not found at $controllerGenPath" -ForegroundColor Red
    Write-Host "Installing controller-gen..." -ForegroundColor Yellow
    go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ Failed to install controller-gen" -ForegroundColor Red
        exit 1
    }
}

# Create CRDs directory
$crdsDir = "deployments\crds"
if (-not (Test-Path $crdsDir)) {
    New-Item -ItemType Directory -Path $crdsDir -Force | Out-Null
}

Write-Host "📁 Generating deepcopy methods..." -ForegroundColor Yellow
& $controllerGenPath object:headerFile="hack/boilerplate.go.txt" paths="./api/v1"
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to generate deepcopy methods" -ForegroundColor Red
    exit 1
}
Write-Host "✅ Deepcopy methods generated successfully" -ForegroundColor Green

Write-Host "🏗️  Generating CRDs..." -ForegroundColor Yellow
& $controllerGenPath crd:allowDangerousTypes=true rbac:roleName=manager-role webhook paths="./api/v1" output:crd:artifacts:config=deployments/crds
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to generate CRDs" -ForegroundColor Red
    exit 1
}
Write-Host "✅ CRDs generated successfully" -ForegroundColor Green

# Verify CRDs were created
$crdFiles = Get-ChildItem -Path $crdsDir -Filter "*.yaml"
Write-Host "📋 Generated CRD files:" -ForegroundColor Yellow
foreach ($file in $crdFiles) {
    $lineCount = (Get-Content $file.FullName | Measure-Object -Line).Lines
    Write-Host "   • $($file.Name) ($lineCount lines)" -ForegroundColor White
}

# Check if API types compile
Write-Host "🔨 Testing API compilation..." -ForegroundColor Yellow
go build ./api/...
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ API types failed to compile" -ForegroundColor Red
    exit 1
}
Write-Host "✅ API types compile successfully" -ForegroundColor Green

Write-Host ""
Write-Host "🎉 CRD Generation Test PASSED!" -ForegroundColor Green
Write-Host "All CRDs have been generated successfully with proper type specifications." -ForegroundColor White
Write-Host ""
Write-Host "Key fixes applied:" -ForegroundColor Cyan
Write-Host "  • Removed Min/Max validation from float64 fields" -ForegroundColor White
Write-Host "  • Removed pattern validation from resource.Quantity fields" -ForegroundColor White  
Write-Host "  • Used allowDangerousTypes flag for float64 support" -ForegroundColor White
Write-Host "  • Generated $(($crdFiles | Measure-Object).Count) CRD files successfully" -ForegroundColor White