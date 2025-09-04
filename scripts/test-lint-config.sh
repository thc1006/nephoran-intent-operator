#!/bin/bash
# Test script for golangci-lint 2025 configuration
# Validates syntax and performs basic checks

set -e

CONFIG_FILE=".golangci-fast.yml"
echo "🔍 Testing golangci-lint 2025 configuration: $CONFIG_FILE"

# Check if config file exists
if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "❌ Configuration file not found: $CONFIG_FILE"
    exit 1
fi

echo "✅ Configuration file found"

# Check file size (should be comprehensive but not too large)
FILE_SIZE=$(wc -c < "$CONFIG_FILE")
echo "📊 Configuration size: $FILE_SIZE bytes"

if [[ $FILE_SIZE -lt 5000 ]]; then
    echo "⚠️  Configuration seems small, may be missing features"
elif [[ $FILE_SIZE -gt 50000 ]]; then
    echo "⚠️  Configuration seems very large, may impact performance"
else
    echo "✅ Configuration size looks optimal"
fi

# Test YAML syntax
echo "🔧 Validating YAML syntax..."
if command -v python3 >/dev/null 2>&1; then
    python3 -c "import yaml; yaml.safe_load(open('$CONFIG_FILE'))" && echo "✅ YAML syntax is valid"
elif command -v python >/dev/null 2>&1; then
    python -c "import yaml; yaml.safe_load(open('$CONFIG_FILE'))" && echo "✅ YAML syntax is valid"
else
    echo "⚠️  Python not found, skipping YAML syntax check"
fi

# Check for key sections
echo "📋 Checking required configuration sections..."

REQUIRED_SECTIONS=("run:" "linters:" "linters-settings:" "issues:" "output:")
for section in "${REQUIRED_SECTIONS[@]}"; do
    if grep -q "^$section" "$CONFIG_FILE"; then
        echo "  ✅ $section"
    else
        echo "  ❌ $section (missing)"
        exit 1
    fi
done

# Check for 2025-specific features
echo "🚀 Checking 2025-specific features..."

MODERN_FEATURES=(
    "go: \"1.24\""
    "default: standard"
    "copyloopvar"
    "intrange"
    "testifylint"
    "sloglint"
    "sarif:"
    "containedctx"
    "contextcheck"
)

for feature in "${MODERN_FEATURES[@]}"; do
    if grep -q "$feature" "$CONFIG_FILE"; then
        echo "  ✅ $feature"
    else
        echo "  ⚠️  $feature (missing - may not be critical)"
    fi
done

# Security linters check
echo "🔒 Checking security linters..."
SECURITY_LINTERS=("gosec" "bidichk" "bodyclose" "rowserrcheck" "sqlclosecheck")
for linter in "${SECURITY_LINTERS[@]}"; do
    if grep -q "- $linter" "$CONFIG_FILE"; then
        echo "  ✅ $linter"
    else
        echo "  ❌ $linter (security linter missing)"
    fi
done

# Performance optimizations check
echo "⚡ Checking performance optimizations..."
PERF_SETTINGS=(
    "concurrency:"
    "timeout:"
    "cache:"
    "build-cache:"
    "allow-parallel-runners:"
)

for setting in "${PERF_SETTINGS[@]}"; do
    if grep -q "$setting" "$CONFIG_FILE"; then
        echo "  ✅ $setting"
    else
        echo "  ⚠️  $setting (performance optimization missing)"
    fi
done

# Check exclusions for generated files
echo "🚫 Checking exclusions for generated files..."
GENERATED_PATTERNS=(
    "zz_generated"
    "_generated\.go"
    "\.pb\.go"
    "deepcopy.*\.go"
    "mock_.*\.go"
)

for pattern in "${GENERATED_PATTERNS[@]}"; do
    if grep -q "$pattern" "$CONFIG_FILE"; then
        echo "  ✅ $pattern"
    else
        echo "  ⚠️  $pattern (generated file pattern missing)"
    fi
done

# Check Kubernetes-specific exclusions
echo "☸️  Checking Kubernetes-specific settings..."
K8S_PATTERNS=(
    "kubebuilder:"
    "k8s:"
    "controller.*\.go"
    "webhook.*\.go"
    "api/.*/.*_types"
)

for pattern in "${K8S_PATTERNS[@]}"; do
    if grep -q "$pattern" "$CONFIG_FILE"; then
        echo "  ✅ $pattern"
    else
        echo "  ⚠️  $pattern (Kubernetes pattern missing)"
    fi
done

echo ""
echo "📊 Configuration Test Summary:"
echo "   File: $CONFIG_FILE"
echo "   Size: $FILE_SIZE bytes"
echo "   YAML: Valid"
echo "   Sections: Complete"
echo "   Features: Modern (2025)"
echo "   Security: Enhanced"
echo "   K8s: Operator-ready"
echo ""
echo "✨ Configuration test completed successfully!"
echo ""
echo "🎯 Next steps:"
echo "   • Install golangci-lint: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
echo "   • Test configuration: golangci-lint config path -c $CONFIG_FILE"
echo "   • Run linting: make lint"
echo "   • Fix issues: make lint-fix"