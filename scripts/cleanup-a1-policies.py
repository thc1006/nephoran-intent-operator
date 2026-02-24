#!/usr/bin/env python3
"""
A1 Policy Cleanup Script
Purpose: Clean up old/test policies from A1 Mediator
Author: Nephoran Intent Operator
Date: 2026-02-24
"""

import requests
import json
import sys
import time
import argparse
from typing import List, Set

# A1 Mediator configuration
A1_URL = "http://service-ricplt-a1mediator-http.ricplt:10000"
POLICY_TYPE = "100"

# Policies to keep (exact matches or prefixes)
KEEP_PATTERNS = [
    "policy-test-scale-to-5",
    "policy-e2e-lifecycle-test",
]

# Policies to delete (prefixes)
DELETE_PATTERNS = [
    "policy-intent-nf-sim-",
    "policy-intent-test-",
    "policy-intent-api-server-",
    "policy-intent-free5gc-nrf-",
    "policy-intent-ricxapp-kpimon-",
    "policy-validation-test",
    "policy-invalid-intent",
    "test-policy-",
]


def get_all_policies() -> List[str]:
    """Fetch all policies from A1 Mediator"""
    url = f"{A1_URL}/A1-P/v2/policytypes/{POLICY_TYPE}/policies"
    try:
        response = requests.get(url, timeout=10)
        response.raise_for_status()
        policies = response.json()
        return policies if isinstance(policies, list) else []
    except Exception as e:
        print(f"❌ Error fetching policies: {e}")
        return []


def should_keep(policy_id: str) -> bool:
    """Check if policy should be kept"""
    for pattern in KEEP_PATTERNS:
        if policy_id.startswith(pattern) or policy_id == pattern:
            return True
    return False


def should_delete(policy_id: str) -> bool:
    """Check if policy matches deletion pattern"""
    for pattern in DELETE_PATTERNS:
        if policy_id.startswith(pattern):
            return True
    return False


def delete_policy(policy_id: str, dry_run: bool = True) -> bool:
    """Delete a single policy"""
    url = f"{A1_URL}/A1-P/v2/policytypes/{POLICY_TYPE}/policies/{policy_id}"

    if dry_run:
        print(f"  [DRY RUN] Would delete: {policy_id}")
        return True

    try:
        response = requests.delete(url, timeout=10)
        if response.status_code in [200, 204]:
            print(f"  ✅ Deleted: {policy_id}")
            return True
        else:
            print(f"  ❌ Failed to delete {policy_id} (HTTP {response.status_code})")
            return False
    except Exception as e:
        print(f"  ❌ Error deleting {policy_id}: {e}")
        return False


def main():
    parser = argparse.ArgumentParser(description="Clean up old A1 policies")
    parser.add_argument(
        "--dry-run",
        action="store_true",
        default=True,
        help="Perform dry run without actual deletion (default: True)",
    )
    parser.add_argument(
        "--execute",
        action="store_true",
        help="Actually execute the deletion",
    )
    parser.add_argument(
        "--a1-url",
        default=A1_URL,
        help=f"A1 Mediator URL (default: {A1_URL})",
    )
    args = parser.parse_args()

    dry_run = not args.execute
    global A1_URL
    A1_URL = args.a1_url

    print("═══════════════════════════════════════════════════════════════")
    print("  A1 Policy Cleanup Script")
    print("═══════════════════════════════════════════════════════════════")
    print(f"A1 Mediator URL: {A1_URL}")
    print(f"Policy Type: {POLICY_TYPE}")
    print(f"Dry Run: {dry_run}")
    print("")

    # Fetch all policies
    print("Fetching all policies...")
    all_policies = get_all_policies()
    total_count = len(all_policies)

    if total_count == 0:
        print("No policies found. Exiting.")
        return 0

    print(f"Total policies found: {total_count}")
    print("")

    # Categorize policies
    keep_policies = []
    delete_policies = []
    other_policies = []

    for policy_id in all_policies:
        if should_keep(policy_id):
            keep_policies.append(policy_id)
            print(f"  [KEEP] {policy_id}")
        elif should_delete(policy_id):
            delete_policies.append(policy_id)
        else:
            other_policies.append(policy_id)

    print("")
    print(f"Policies to keep: {len(keep_policies)}")
    print(f"Policies to delete: {len(delete_policies)}")
    print(f"Other policies (ignored): {len(other_policies)}")
    print("")

    if len(delete_policies) == 0:
        print("No policies to delete. Exiting.")
        return 0

    # Show first few policies to be deleted
    print("Policies to be deleted (first 20):")
    for policy_id in delete_policies[:20]:
        print(f"  - {policy_id}")
    if len(delete_policies) > 20:
        print(f"  ... and {len(delete_policies) - 20} more")
    print("")

    # Confirm before deletion
    if not dry_run:
        print(f"⚠️  WARNING: This will delete {len(delete_policies)} policies!")
        response = input("Type 'yes' to continue, or anything else to cancel: ")
        if response.lower() != "yes":
            print("Cancelled.")
            return 1

    # Delete policies
    print("Starting deletion...")
    deleted = 0
    failed = 0

    for i, policy_id in enumerate(delete_policies, 1):
        if delete_policy(policy_id, dry_run):
            deleted += 1
        else:
            failed += 1

        # Progress indicator
        if i % 10 == 0:
            print(f"  Progress: {i}/{len(delete_policies)}")

        # Rate limiting
        time.sleep(0.1)

    print("")
    print("═══════════════════════════════════════════════════════════════")
    print("  Cleanup Summary")
    print("═══════════════════════════════════════════════════════════════")
    print(f"Total policies: {total_count}")
    print(f"Deleted: {deleted}")
    print(f"Failed: {failed}")
    print(f"Remaining: {total_count - deleted}")
    print("")

    if dry_run:
        print("ℹ️  This was a DRY RUN. No policies were actually deleted.")
        print("To perform actual deletion, run: python3 cleanup-a1-policies.py --execute")
    else:
        print("✅ Cleanup complete!")

    return 0


if __name__ == "__main__":
    sys.exit(main())
