#!/bin/bash
set -euo pipefail

# CI Workflow Validation Script

echo "=== GitHub Actions Workflow Validation ==="

# Validate workflow syntax
echo "1. Validating workflow syntax..."
gh workflow lint .github/workflows/ci-timeout-fix.yml

# Test cache key generation
echo "2. Testing cache key generation..."
CACHE_KEY=$(./scripts/generate-cache-key.sh)
echo "Generated Cache Key: $CACHE_KEY"

# Simulate dependency download
echo "3. Testing dependency download fallback..."
make -f Makefile.ci download-deps-simulate-failure || true

# Run minimal workflow simulation
echo "4. Running minimal workflow simulation..."
act push -j preflight
act push -j build-critical

# Verify cache operations
echo "5. Checking cache restoration logs..."
grep -E "Cache (hit|miss|restored)" /tmp/github-actions.log || echo "No cache logs found"

echo "=== Validation Complete ==="