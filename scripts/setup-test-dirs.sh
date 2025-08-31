#!/bin/bash

# Setup test directories for all test targets
set -e

echo "🔧 Setting up test directories..."

# Create all necessary test directories
mkdir -p .excellence-reports
mkdir -p .quality-reports/coverage
mkdir -p test-results
mkdir -p regression-artifacts

echo "✅ Test directories created successfully"