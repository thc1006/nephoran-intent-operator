#!/bin/bash

# Run tests with coverage in CI environment
set -e

echo "🧪 Running comprehensive test suite with coverage analysis..."

# Setup directories
mkdir -p .quality-reports/coverage
mkdir -p .excellence-reports

# Run tests with coverage
go test ./... -v -race -coverprofile=.quality-reports/coverage/coverage.out -covermode=atomic || {
    echo "❌ Tests failed"
    exit 1
}

# Copy coverage to both locations for compatibility
cp .quality-reports/coverage/coverage.out .excellence-reports/coverage.out 2>/dev/null || true

echo "✅ Tests completed successfully"
echo "Coverage report: .quality-reports/coverage/coverage.out"