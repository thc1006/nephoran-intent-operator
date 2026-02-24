#!/bin/bash
# A1 Policy Cleanup Script
# Purpose: Clean up old/test policies from A1 Mediator
# Author: Nephoran Intent Operator
# Date: 2026-02-24

set -e

A1_URL="${A1_MEDIATOR_URL:-http://service-ricplt-a1mediator-http.ricplt:10000}"
POLICY_TYPE="100"
DRY_RUN="${DRY_RUN:-true}"

echo "═══════════════════════════════════════════════════════════════"
echo "  A1 Policy Cleanup Script"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "A1 Mediator URL: $A1_URL"
echo "Policy Type: $POLICY_TYPE"
echo "Dry Run: $DRY_RUN"
echo ""

# Function to delete a policy
delete_policy() {
    local policy_id=$1
    local url="$A1_URL/A1-P/v2/policytypes/$POLICY_TYPE/policies/$policy_id"

    if [ "$DRY_RUN" = "true" ]; then
        echo "  [DRY RUN] Would delete: $policy_id"
        return 0
    fi

    response=$(curl -s -w "\n%{http_code}" -X DELETE "$url")
    http_code=$(echo "$response" | tail -n1)

    if [ "$http_code" = "200" ] || [ "$http_code" = "204" ]; then
        echo "  ✅ Deleted: $policy_id"
        return 0
    else
        echo "  ❌ Failed to delete $policy_id (HTTP $http_code)"
        return 1
    fi
}

# Get all policies
echo "Fetching all policies..."
all_policies=$(curl -s "$A1_URL/A1-P/v2/policytypes/$POLICY_TYPE/policies")
total_count=$(echo "$all_policies" | jq '. | length')

echo "Total policies found: $total_count"
echo ""

# Policies to keep (pattern-based)
KEEP_PATTERNS=(
    "policy-test-scale-to-5"      # Our E2E test policy
    "policy-e2e-lifecycle-test"   # Lifecycle test
)

# Policies to delete (pattern-based)
DELETE_PATTERNS=(
    "policy-intent-nf-sim-"       # Bulk test policies for nf-sim
    "policy-intent-test-"         # Generic test policies
    "policy-intent-api-server-"   # Non-existent api-server
    "policy-intent-free5gc-nrf-"  # Non-existent free5gc-nrf
    "policy-intent-ricxapp-kpimon-" # KPIMON policies (deployment is ricxapp-kpimon not kpimon)
    "policy-validation-test"      # Validation test
    "policy-invalid-intent"       # Invalid intent test
    "test-policy-"                # Test policies
)

# Extract policy IDs to delete
policies_to_delete=()

while IFS= read -r policy_id; do
    # Check if policy matches any keep pattern
    keep=false
    for pattern in "${KEEP_PATTERNS[@]}"; do
        if [[ "$policy_id" == "$pattern"* ]]; then
            keep=true
            break
        fi
    done

    if [ "$keep" = "true" ]; then
        echo "  [KEEP] $policy_id"
        continue
    fi

    # Check if policy matches any delete pattern
    delete=false
    for pattern in "${DELETE_PATTERNS[@]}"; do
        if [[ "$policy_id" == "$pattern"* ]]; then
            delete=true
            break
        fi
    done

    if [ "$delete" = "true" ]; then
        policies_to_delete+=("$policy_id")
    fi
done < <(echo "$all_policies" | jq -r '.[]')

echo ""
echo "Policies to delete: ${#policies_to_delete[@]}"
echo "Policies to keep: $((total_count - ${#policies_to_delete[@]}))"
echo ""

if [ ${#policies_to_delete[@]} -eq 0 ]; then
    echo "No policies to delete. Exiting."
    exit 0
fi

# Confirm before deletion (if not dry run)
if [ "$DRY_RUN" != "true" ]; then
    echo "⚠️  WARNING: This will delete ${#policies_to_delete[@]} policies!"
    echo "Press Ctrl+C to cancel, or Enter to continue..."
    read -r
fi

# Delete policies
echo "Starting deletion..."
deleted=0
failed=0

for policy_id in "${policies_to_delete[@]}"; do
    if delete_policy "$policy_id"; then
        ((deleted++))
    else
        ((failed++))
    fi

    # Rate limiting (avoid overwhelming A1 Mediator)
    sleep 0.1
done

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  Cleanup Summary"
echo "═══════════════════════════════════════════════════════════════"
echo "Total policies: $total_count"
echo "Deleted: $deleted"
echo "Failed: $failed"
echo "Remaining: $((total_count - deleted))"
echo ""

if [ "$DRY_RUN" = "true" ]; then
    echo "ℹ️  This was a DRY RUN. No policies were actually deleted."
    echo "To perform actual deletion, run: DRY_RUN=false $0"
fi

echo ""
echo "✅ Cleanup complete!"
