#!/bin/bash
# =============================================================================
# CI Consolidation Verification Script
# =============================================================================
# Verifies that only nephoran-ci-consolidated-2025.yml is active for PRs
# =============================================================================

set -e

echo "🔍 CI Workflow Consolidation Verification"
echo "=========================================="
echo

# Count total workflows
TOTAL_WORKFLOWS=$(find .github/workflows -name "*.yml" | wc -l)
echo "📁 Total workflow files: $TOTAL_WORKFLOWS"

# Check for active pull_request triggers
echo
echo "🔍 Checking active pull_request triggers..."
ACTIVE_PR_WORKFLOWS=()

while IFS= read -r -d '' file; do
    # Check if file has uncommented pull_request trigger
    if grep -q "^[[:space:]]*pull_request:" "$file"; then
        workflow_name=$(grep "^name:" "$file" | head -1 | sed 's/name:[[:space:]]*//' | tr -d '"' || echo "Unnamed")
        ACTIVE_PR_WORKFLOWS+=("$(basename "$file"): $workflow_name")
        echo "  ✅ ACTIVE: $(basename "$file") - $workflow_name"
    fi
done < <(find .github/workflows -name "*.yml" -print0)

echo
echo "📊 Summary:"
echo "  Total workflows: $TOTAL_WORKFLOWS"
echo "  Active PR workflows: ${#ACTIVE_PR_WORKFLOWS[@]}"

if [ ${#ACTIVE_PR_WORKFLOWS[@]} -eq 1 ]; then
    echo "  ✅ SUCCESS: Only 1 active PR workflow (as expected)"
    echo "     ${ACTIVE_PR_WORKFLOWS[0]}"
elif [ ${#ACTIVE_PR_WORKFLOWS[@]} -eq 0 ]; then
    echo "  ⚠️  WARNING: No active PR workflows found"
    echo "     This means no CI will run on pull requests"
else
    echo "  ❌ ISSUE: Multiple active PR workflows detected:"
    for workflow in "${ACTIVE_PR_WORKFLOWS[@]}"; do
        echo "     - $workflow"
    done
    echo "     This will cause conflicts and multiple CI runs"
fi

# Verify target workflow exists and is active
TARGET_WORKFLOW=".github/workflows/nephoran-ci-consolidated-2025.yml"
if [ -f "$TARGET_WORKFLOW" ]; then
    if grep -q "^[[:space:]]*pull_request:" "$TARGET_WORKFLOW"; then
        echo "  ✅ Target workflow is active: nephoran-ci-consolidated-2025.yml"
    else
        echo "  ❌ Target workflow exists but is NOT active"
    fi
else
    echo "  ❌ Target workflow not found: $TARGET_WORKFLOW"
fi

# Check for disabled workflows
echo
echo "🔍 Checking properly disabled workflows..."
DISABLED_COUNT=0

while IFS= read -r -d '' file; do
    if grep -q "# pull_request.*DISABLED.*Consolidated" "$file"; then
        ((DISABLED_COUNT++))
        echo "  ✅ Properly disabled: $(basename "$file")"
    fi
done < <(find .github/workflows -name "*.yml" -print0)

echo "  📊 Properly disabled workflows: $DISABLED_COUNT"

# Final validation
echo
echo "🎯 Final Validation:"
if [ ${#ACTIVE_PR_WORKFLOWS[@]} -eq 1 ] && grep -q "nephoran-ci-consolidated-2025" <<< "${ACTIVE_PR_WORKFLOWS[0]}"; then
    echo "  ✅ SUCCESS: CI consolidation is properly configured"
    echo "  ✅ Only the target workflow will run on PRs"
    echo "  ✅ PR 176 CI conflict issue should be resolved"
else
    echo "  ❌ ISSUE: CI consolidation needs adjustment"
    echo "  ❌ PR 176 CI conflicts may persist"
    exit 1
fi

echo
echo "🚀 Verification complete. Ready to test on PR!"