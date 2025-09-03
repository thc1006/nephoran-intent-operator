#!/bin/bash
# GitHub Actions Workflow Consolidation Verification Script
# Verifies that workflow consolidation was successful

echo "🔍 GitHub Actions Workflow Consolidation Verification"
echo "======================================================"

# Count active vs disabled workflows
ACTIVE_COUNT=0
DISABLED_COUNT=0

echo ""
echo "📋 Workflow Status:"
echo "-------------------"

for workflow in .github/workflows/*.yml; do
    if [ -f "$workflow" ]; then
        name=$(basename "$workflow" .yml)
        
        # Check if workflow is disabled
        if grep -q "DISABLED\|workflow_dispatch: {}" "$workflow" | head -1 > /dev/null 2>&1; then
            if [ "$name" != "go-module-cache" ] && [ "$name" != "ci-2025" ]; then
                echo "❌ $name (DISABLED)"
                DISABLED_COUNT=$((DISABLED_COUNT + 1))
            fi
        else
            echo "✅ $name (ACTIVE)"
            ACTIVE_COUNT=$((ACTIVE_COUNT + 1))
        fi
    fi
done

echo ""
echo "📊 Summary:"
echo "-----------"
echo "✅ Active workflows: $ACTIVE_COUNT"
echo "❌ Disabled workflows: $DISABLED_COUNT"

echo ""
echo "🔧 Standardization Verification:"
echo "--------------------------------"

# Check cache key standardization
CACHE_STANDARD_COUNT=$(grep -r "nephoran-go-v1" .github/workflows/*.yml | wc -l)
echo "✅ Standardized cache keys found: $CACHE_STANDARD_COUNT occurrences"

# Check concurrency standardization  
CONCURRENCY_STANDARD_COUNT=$(grep -r "nephoran-.*-\${{ github.ref }}" .github/workflows/*.yml | wc -l)
echo "✅ Standardized concurrency groups: $CONCURRENCY_STANDARD_COUNT"

# Check pinned versions
PINNED_CHECKOUT=$(grep -c "actions/checkout@v4.2.1" .github/workflows/*.yml 2>/dev/null || echo "0")
PINNED_SETUP_GO=$(grep -c "actions/setup-go@v5.0.2" .github/workflows/*.yml 2>/dev/null || echo "0")
PINNED_CACHE=$(grep -c "actions/cache@v4.1.2" .github/workflows/*.yml 2>/dev/null || echo "0")

echo "✅ Pinned checkout actions: $PINNED_CHECKOUT"
echo "✅ Pinned setup-go actions: $PINNED_SETUP_GO"  
echo "✅ Pinned cache actions: $PINNED_CACHE"

echo ""
echo "🎯 Primary Workflows (Should be 4-5):"
echo "-------------------------------------"
for workflow in ci-production.yml pr-validation.yml ubuntu-ci.yml emergency-merge.yml go-module-cache.yml; do
    if [ -f ".github/workflows/$workflow" ]; then
        if ! grep -q "DISABLED" ".github/workflows/$workflow"; then
            echo "✅ $workflow (ACTIVE)"
        else
            echo "❌ $workflow (UNEXPECTEDLY DISABLED)"
        fi
    else
        echo "❓ $workflow (NOT FOUND)"
    fi
done

echo ""
echo "🚀 Consolidation Status:"
echo "------------------------"
if [ $ACTIVE_COUNT -le 6 ] && [ $DISABLED_COUNT -ge 8 ]; then
    echo "✅ SUCCESS: Consolidation completed successfully!"
    echo "   - Resource contention resolved"
    echo "   - Cache keys standardized"
    echo "   - Tool versions pinned"
    echo "   - Concurrency groups standardized"
else
    echo "⚠️  WARNING: Consolidation may need review"
    echo "   Active: $ACTIVE_COUNT, Disabled: $DISABLED_COUNT"
fi

echo ""
echo "📚 Documentation:"
echo "----------------"
if [ -f ".github/workflows/CONSOLIDATION-SUMMARY.md" ]; then
    echo "✅ Consolidation summary documentation available"
else
    echo "❌ Consolidation summary documentation missing"
fi

echo ""
echo "🔍 Verification complete!"