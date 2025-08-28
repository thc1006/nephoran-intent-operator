#!/bin/bash

# validate-coverage.sh - Validate coverage file generation
# This script ensures that test coverage is properly generated and accessible.

set -euo pipefail

COVERAGE_DIR=".test-reports"
COVERAGE_FILE="$COVERAGE_DIR/coverage.out"
COVERAGE_HTML="$COVERAGE_DIR/coverage.html"

echo "=== Coverage Validation Script ==="
echo "Checking coverage file generation..."

# Step 1: Create coverage directory
echo "Step 1: Creating coverage directory..."
mkdir -p "$COVERAGE_DIR"
echo "✅ Coverage directory created: $COVERAGE_DIR"

# Step 2: Run tests with coverage (allow failures but check for coverage)
echo "Step 2: Running tests with coverage..."
set +e  # Don't exit on test failures
go test ./... -v -coverprofile="$COVERAGE_FILE" -covermode=atomic -timeout=10m
test_exit_code=$?
set -e

# Step 3: Check if coverage file was generated
echo "Step 3: Checking coverage file..."
if [ -f "$COVERAGE_FILE" ]; then
    file_size=$(stat -c%s "$COVERAGE_FILE" 2>/dev/null || stat -f%z "$COVERAGE_FILE" 2>/dev/null || echo "unknown")
    echo "✅ Coverage file generated: $COVERAGE_FILE ($file_size bytes)"
    
    # Generate HTML report
    if go tool cover -html="$COVERAGE_FILE" -o "$COVERAGE_HTML" 2>/dev/null; then
        echo "✅ HTML coverage report generated: $COVERAGE_HTML"
    else
        echo "⚠️  HTML coverage report generation failed"
    fi
    
    # Show coverage percentage
    if coverage_percent=$(go tool cover -func="$COVERAGE_FILE" | grep total | awk '{print $3}' 2>/dev/null); then
        echo "📊 Total coverage: $coverage_percent"
    else
        echo "⚠️  Could not calculate coverage percentage"
    fi
    
    echo "🎉 Coverage validation successful!"
    exit 0
else
    echo "❌ Coverage file not generated: $COVERAGE_FILE"
    echo "This may be due to compilation errors preventing tests from running."
    echo "Check test output above for specific errors."
    exit 1
fi