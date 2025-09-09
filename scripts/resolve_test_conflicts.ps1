# PowerShell script to resolve test file merge conflicts
# This script favors the integrate/mvp side (comprehensive tests) over HEAD (stub tests)

$ErrorActionPreference = "Continue"

# Get all Go files with merge conflicts
$conflictFiles = @(
    "pkg/nephio/advanced_benchmarks_test.go",
    "pkg/nephio/blueprint/manager_comprehensive_test.go", 
    "pkg/nephio/krm/runtime_comprehensive_test.go",
    "pkg/nephio/multicluster/chaos_resilience_test.go",
    "pkg/oran/a1/handlers_test.go",
    "pkg/oran/a1/validation_test.go", 
    "pkg/oran/o1/o1_adaptor_test.go",
    "pkg/oran/o2/o2_manager_test.go",
    "pkg/oran/o2/providers/providers_test.go"
)

foreach ($file in $conflictFiles) {
    $fullPath = "C:\Users\tingy\dev\_worktrees\nephoran\feat-llm-provider\$file"
    
    if (Test-Path $fullPath) {
        Write-Host "Processing: $file" -ForegroundColor Green
        
        # Read the file content
        $content = Get-Content $fullPath -Raw
        
        # Check if it has merge conflicts
        if ($content -match "<<<<<<< HEAD") {
            # For test files, we generally want to keep the integrate/mvp version (comprehensive tests)
            # Remove everything between <<<<<<< HEAD and ======= (keeping integrate/mvp part)
            $resolved = $content -replace "(?s)<<<<<<< HEAD.*?=======\s*", ""
            
            # Remove the >>>>>>> integrate/mvp marker
            $resolved = $resolved -replace ">>>>>>> integrate/mvp\s*", ""
            
            # Write back the resolved content
            Set-Content $fullPath -Value $resolved -NoNewline
            
            Write-Host "  Resolved conflicts in $file" -ForegroundColor Yellow
        } else {
            Write-Host "  No conflicts found in $file" -ForegroundColor Cyan
        }
    } else {
        Write-Host "File not found: $fullPath" -ForegroundColor Red
    }
}

Write-Host "Test conflict resolution completed!" -ForegroundColor Green