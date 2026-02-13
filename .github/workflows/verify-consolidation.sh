#!/bin/bash
set -euo pipefail

echo "Verifying consolidated workflow set..."

expected=(
  "ci-2025.yml"
  "pr-validation.yml"
  "ubuntu-ci.yml"
  "emergency-merge.yml"
  "go-module-cache.yml"
)

missing=0
for wf in "${expected[@]}"; do
  if [[ ! -f ".github/workflows/$wf" ]]; then
    echo "MISSING: $wf"
    missing=1
  fi
done

if rg -n "runner\.os|windows-latest|macos-latest" .github/workflows/*.yml >/dev/null 2>&1; then
  echo "FAIL: cross-platform markers found"
  exit 1
fi

if ! rg -n "^concurrency:|group:\s*\$\{\{ github\.ref \}\}" .github/workflows/*.yml >/dev/null 2>&1; then
  echo "FAIL: concurrency policy not found"
  exit 1
fi

if ! rg -n "name:\s*Basic Validation" .github/workflows/pr-validation.yml >/dev/null 2>&1; then
  echo "FAIL: Basic Validation check missing in pr-validation.yml"
  exit 1
fi

if ! rg -n "name:\s*CI Status" .github/workflows/pr-validation.yml >/dev/null 2>&1; then
  echo "FAIL: CI Status check missing in pr-validation.yml"
  exit 1
fi

if [[ $missing -ne 0 ]]; then
  exit 1
fi

echo "Workflow consolidation verification passed."
