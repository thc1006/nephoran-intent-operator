#!/bin/bash

# Format Bypass Safety Checker
# This script helps determine if it's safe to bypass formatting checks
# Usage: ./scripts/format-bypass-check.sh

set -e

echo "🔍 Format Bypass Safety Checker"
echo "================================"
echo ""

# Check if golangci-lint is available
if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "❌ golangci-lint not found. Please install it first:"
    echo "   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
    exit 1
fi

echo "📋 Running critical checks (non-formatting)..."

# Run bypass configuration (without formatting linters)
if golangci-lint run -c .golangci-bypass.yml --out-format=json > /tmp/bypass-results.json 2>/dev/null; then
    critical_count=0
else
    # Count critical issues
    critical_count=$(jq '[.Issues // [] | .[] | select(.Severity == "error")] | length' /tmp/bypass-results.json 2>/dev/null || echo "unknown")
fi

echo "📋 Running full linting check..."

# Run full configuration to see formatting issues
if golangci-lint run --out-format=json > /tmp/full-results.json 2>/dev/null; then
    full_count=0
else
    full_count=$(jq '[.Issues // []] | length' /tmp/full-results.json 2>/dev/null || echo "unknown")
fi

# Analyze results
echo ""
echo "📊 Results Analysis:"
echo "===================="

if [ -f "/tmp/bypass-results.json" ] && jq empty < /tmp/bypass-results.json 2>/dev/null; then
    critical_issues=$(jq '[.Issues // [] | .[] | select(.Severity == "error")] | length' /tmp/bypass-results.json)
    echo "Critical errors (non-formatting): $critical_issues"
else
    critical_issues="unknown"
    echo "Critical errors (non-formatting): Could not analyze"
fi

if [ -f "/tmp/full-results.json" ] && jq empty < /tmp/full-results.json 2>/dev/null; then
    total_issues=$(jq '[.Issues // []] | length' /tmp/full-results.json)
    format_issues=$(jq '[.Issues // [] | .[] | select(.FromLinter | test("gofmt|gofumpt|gci|whitespace"))] | length' /tmp/full-results.json)
    echo "Total issues (full check): $total_issues"
    echo "Formatting-only issues: $format_issues"
else
    total_issues="unknown"
    format_issues="unknown"
    echo "Total issues (full check): Could not analyze"
fi

echo ""
echo "🎯 Safety Assessment:"
echo "====================="

if [ "$critical_issues" = "0" ] || [ "$critical_issues" = "unknown" ]; then
    echo "✅ SAFE TO BYPASS"
    echo ""
    echo "No critical errors detected. You can safely use format bypass:"
    echo ""
    echo "   git commit -m \"[bypass-format] your commit message here\""
    echo ""
    echo "Or use one of these bypass keywords in your commit message:"
    echo "   [bypass-format], [format-bypass], [bypass-lint], [skip-lint]"
    echo ""
    echo "⚠️  Remember to fix formatting issues in a follow-up commit:"
    echo "   golangci-lint run --fix"
    echo "   gofmt -w ."
    echo ""
    safety_status="SAFE"
else
    echo "❌ NOT SAFE TO BYPASS"
    echo ""
    echo "Critical errors found: $critical_issues"
    echo ""
    echo "You must fix these critical issues before bypassing:"
    echo ""
    # Show critical issues if available
    if [ -f "/tmp/bypass-results.json" ]; then
        echo "Critical Issues:"
        jq -r '.Issues[] | select(.Severity == "error") | "  - \(.Pos.Filename):\(.Pos.Line): \(.Text)"' /tmp/bypass-results.json | head -10
    fi
    echo ""
    echo "Fix critical issues first, then you can bypass formatting issues."
    safety_status="UNSAFE"
fi

echo ""
echo "📝 Quick Fix Commands:"
echo "======================"
echo "Format code:      gofmt -w ."
echo "Fix imports:      gci write --skip-generated -s standard -s default -s \"prefix(github.com/nephio-project/nephoran-intent-operator)\" ."
echo "Auto-fix lint:    golangci-lint run --fix"
echo "Full lint check:  golangci-lint run"
echo "Bypass check:     golangci-lint run -c .golangci-bypass.yml"

# Cleanup
rm -f /tmp/bypass-results.json /tmp/full-results.json

# Exit with appropriate code
if [ "$safety_status" = "SAFE" ]; then
    exit 0
else
    exit 1
fi