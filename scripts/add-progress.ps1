param(
  [Parameter(Mandatory=$true)][string]$Module,
  [Parameter(Mandatory=$true)][string]$Summary
)

# 取分支名與 ISO-8601 時戳
$branch = (git rev-parse --abbrev-ref HEAD)
$ts     = Get-Date -Format o   # ISO-8601

$line = "| $ts | $branch | $Module | $Summary |"
Add-Content -Path ".\docs\PROGRESS.md" -Value $line

Write-Host "Appended: $line"
