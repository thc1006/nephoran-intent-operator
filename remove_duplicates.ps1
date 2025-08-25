# PowerShell script to remove duplicate type declarations

# Remove duplicates from security_manager.go
$file = "pkg/oran/o1/security_manager.go"

# Read file content
$content = Get-Content $file -Raw

# Remove duplicate type blocks (this is a simplified approach)
# In production, would use more sophisticated parsing

# Replace duplicate type declarations with comments
$content = $content -replace "(?s)type AuditEntry struct \{[^}]*\}", "// AuditEntry defined in accounting_manager.go"
$content = $content -replace "(?s)type AuditStorage interface \{[^}]*\}", "// AuditStorage defined in accounting_manager.go"
$content = $content -replace "(?s)type RiskFactor struct \{[^}]*\}", "// RiskFactor defined in accounting_manager.go"
$content = $content -replace "(?s)type ReportSchedule struct \{[^}]*\}", "// ReportSchedule defined in common_types.go"
$content = $content -replace "(?s)type ReportDistributor interface \{[^}]*\}", "// ReportDistributor defined in common_types.go"
$content = $content -replace "(?s)type DistributionChannel interface \{[^}]*\}", "// DistributionChannel defined in common_types.go"
$content = $content -replace "(?s)type NotificationChannel interface \{[^}]*\}", "// NotificationChannel defined in common_types.go"

# Write back to file
Set-Content $file -Value $content

Write-Host "Duplicate types removed from security_manager.go"