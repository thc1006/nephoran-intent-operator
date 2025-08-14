#!/bin/bash
# CI Validation Test Script

echo "=== CI Path Filter and Job Validation Tests ==="
echo ""

# Test 1: Happy Path - Validate YAML Syntax
echo "Test 1: Happy Path - YAML Syntax Validation"
python -c "import yaml; yaml.safe_load(open('.github/workflows/ci.yml', 'r', encoding='utf-8'))" 2>/dev/null
if [ $? -eq 0 ]; then
    echo "✅ PASS: YAML syntax is valid"
else
    echo "❌ FAIL: YAML syntax error"
    exit 1
fi
echo ""

# Test 2: Failure Case - Check for duplicate job names
echo "Test 2: Failure Case - Check for duplicate job names"
job_names=$(grep -E "^  [a-z-]+:" .github/workflows/ci.yml | sed 's/://g' | sed 's/  //g' | sort)
duplicate_jobs=$(echo "$job_names" | uniq -d)
if [ -z "$duplicate_jobs" ]; then
    echo "✅ PASS: No duplicate job names found"
else
    echo "❌ FAIL: Duplicate job names found: $duplicate_jobs"
    exit 1
fi
echo ""

# Test 3: Failure Case - Check concurrency group configuration
echo "Test 3: Failure Case - Validate concurrency group"
concurrency_group=$(grep -A1 "^concurrency:" .github/workflows/ci.yml | grep "group:" | grep -o '\${{ github.ref }}')
if [ ! -z "$concurrency_group" ]; then
    echo "✅ PASS: Concurrency group correctly set to \${{ github.ref }}"
else
    echo "❌ FAIL: Concurrency group not properly configured"
    exit 1
fi
echo ""

# Test 4: Happy Path - Validate path filters exist
echo "Test 4: Happy Path - Validate module path filters"
expected_filters=("api" "controllers" "pkg-nephio" "pkg-oran" "pkg-llm" "pkg-rag" "pkg-core" "cmd" "internal")
missing_filters=""
for filter in "${expected_filters[@]}"; do
    if grep -q "      $filter:" .github/workflows/ci.yml; then
        echo "  ✓ Found filter: $filter"
    else
        missing_filters="$missing_filters $filter"
    fi
done

if [ -z "$missing_filters" ]; then
    echo "✅ PASS: All expected path filters found"
else
    echo "❌ FAIL: Missing filters:$missing_filters"
    exit 1
fi
echo ""

# Test 5: Validate module test jobs exist
echo "Test 5: Validate module-specific test jobs"
expected_jobs=("test-api" "test-controllers" "test-pkg-nephio" "test-pkg-oran" "test-pkg-llm" "test-pkg-rag" "test-pkg-core" "test-cmd" "test-internal")
missing_jobs=""
for job in "${expected_jobs[@]}"; do
    if grep -q "^  $job:" .github/workflows/ci.yml; then
        echo "  ✓ Found job: $job"
    else
        missing_jobs="$missing_jobs $job"
    fi
done

if [ -z "$missing_jobs" ]; then
    echo "✅ PASS: All module test jobs found"
else
    echo "❌ FAIL: Missing jobs:$missing_jobs"
    exit 1
fi
echo ""

# Test 6: Validate integration gate job
echo "Test 6: Validate integrate-mvp-gate job"
if grep -q "^  integrate-mvp-gate:" .github/workflows/ci.yml; then
    echo "✅ PASS: Integration gate job exists"
else
    echo "❌ FAIL: Integration gate job not found"
    exit 1
fi
echo ""

# Test 7: Validate job dependencies
echo "Test 7: Validate job dependencies for integrate-mvp-gate"
deps_line=$(grep -A20 "^  integrate-mvp-gate:" .github/workflows/ci.yml | grep -A15 "needs:")
if echo "$deps_line" | grep -q "hygiene" && \
   echo "$deps_line" | grep -q "generate" && \
   echo "$deps_line" | grep -q "build" && \
   echo "$deps_line" | grep -q "test"; then
    echo "✅ PASS: Core dependencies properly configured"
else
    echo "❌ FAIL: Missing core job dependencies"
    exit 1
fi
echo ""

# Test 8: Validate path filter conditions
echo "Test 8: Validate conditional execution based on path filters"
test_jobs_with_conditions=0
for job in "${expected_jobs[@]}"; do
    filter_name=${job#test-}
    filter_name=${filter_name//-/_}
    if grep -A3 "^  $job:" .github/workflows/ci.yml | grep -q "needs.changes.outputs"; then
        ((test_jobs_with_conditions++))
    fi
done

if [ $test_jobs_with_conditions -eq ${#expected_jobs[@]} ]; then
    echo "✅ PASS: All test jobs have path filter conditions"
else
    echo "⚠️  WARNING: Only $test_jobs_with_conditions/${#expected_jobs[@]} jobs have conditions"
fi
echo ""

echo "=== Test Summary ==="
echo "All critical tests passed. CI configuration is valid."