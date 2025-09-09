#!/bin/bash
# Contract Validation Script
# Project Conductor: Ensures all contracts are valid and integration-ready

set -euo pipefail

echo "🔧 Contract Validation Script"
echo "============================="

CONTRACTS_DIR="docs/contracts"
VALIDATION_ERRORS=0

# Check if contracts directory exists
if [ ! -d "$CONTRACTS_DIR" ]; then
    echo "❌ Contracts directory not found: $CONTRACTS_DIR"
    exit 1
fi

echo "📁 Validating contracts in: $CONTRACTS_DIR"

# Validate JSON Schema files
for schema_file in "$CONTRACTS_DIR"/*.schema.json; do
    if [ -f "$schema_file" ]; then
        echo "🧪 Validating JSON Schema: $(basename "$schema_file")"
        
        # Basic JSON syntax validation
        if ! jq empty "$schema_file" >/dev/null 2>&1; then
            echo "❌ Invalid JSON syntax in $schema_file"
            VALIDATION_ERRORS=$((VALIDATION_ERRORS + 1))
            continue
        fi
        
        # Check required schema fields
        if ! jq -e '.["$schema"]' "$schema_file" >/dev/null; then
            echo "⚠️  Missing \$schema field in $schema_file"
        fi
        
        if ! jq -e '.title' "$schema_file" >/dev/null; then
            echo "⚠️  Missing title field in $schema_file"
        fi
        
        if ! jq -e '.type' "$schema_file" >/dev/null; then
            echo "❌ Missing type field in $schema_file"
            VALIDATION_ERRORS=$((VALIDATION_ERRORS + 1))
        fi
        
        echo "✅ $(basename "$schema_file") is valid"
    fi
done

# Validate VES examples
VES_FILE="$CONTRACTS_DIR/fcaps.ves.examples.json"
if [ -f "$VES_FILE" ]; then
    echo "🧪 Validating VES examples: $(basename "$VES_FILE")"
    
    if ! jq empty "$VES_FILE" >/dev/null 2>&1; then
        echo "❌ Invalid JSON syntax in $VES_FILE"
        VALIDATION_ERRORS=$((VALIDATION_ERRORS + 1))
    else
        # Check VES structure
        if jq -e '.fault_example.event.commonEventHeader' "$VES_FILE" >/dev/null; then
            echo "✅ VES fault example structure valid"
        else
            echo "⚠️  VES fault example may be missing required fields"
        fi
        
        if jq -e '.measurement_example.event.commonEventHeader' "$VES_FILE" >/dev/null; then
            echo "✅ VES measurement example structure valid"
        else
            echo "⚠️  VES measurement example may be missing required fields"
        fi
    fi
fi

# Validate E2 KMP profile
E2_FILE="$CONTRACTS_DIR/e2.kmp.profile.md"
if [ -f "$E2_FILE" ]; then
    echo "📋 Validating E2 KMP profile: $(basename "$E2_FILE")"
    
    # Check for required sections
    if grep -q "## Measurements" "$E2_FILE"; then
        echo "✅ E2 KMP measurements section found"
    else
        echo "⚠️  E2 KMP measurements section missing"
    fi
    
    if grep -q "kmp\.p95_latency_ms\|kmp\.prb_utilization\|kmp\.ue_count" "$E2_FILE"; then
        echo "✅ E2 KMP required metrics documented"
    else
        echo "⚠️  E2 KMP required metrics may be missing"
    fi
fi

# Final validation report
echo ""
echo "📊 Validation Results"
echo "===================="

if [ $VALIDATION_ERRORS -eq 0 ]; then
    echo "✅ All contracts validated successfully"
    echo "🎯 Contracts are compliant for Nephio/Porch integration"
    exit 0
else
    echo "❌ $VALIDATION_ERRORS validation errors found"
    echo "🚫 Fix errors before proceeding with integration"
    exit 1
fi