#!/bin/bash
set -euo pipefail  # Strict error handling

# Cleanup script for Nephoran Intent Operator obsolete files
# Generated: 2025-07-29T02:47:00Z
# Total files to remove: 14

echo "Starting safe cleanup of obsolete files..."

# Pre-cleanup validation
echo "Validating build system integrity..."
make build-all || { echo "Build validation failed - aborting cleanup"; exit 1; }

echo "Running Go tests to ensure system integrity..."
go test ./... || { echo "Test validation failed - aborting cleanup"; exit 1; }

echo "Validating Python components..."
if command -v python3 &> /dev/null; then
    python3 -m py_compile pkg/rag/*.py || { echo "Python validation failed - aborting cleanup"; exit 1; }
elif command -v python &> /dev/null; then
    python -m py_compile pkg/rag/*.py || { echo "Python validation failed - aborting cleanup"; exit 1; }
fi

echo "All pre-cleanup validations passed. Proceeding with file removal..."

# File removal commands - removing obsolete files
echo "Removing .kiro directory files..."
git rm ./.kiro/specs/comprehensive-test-suite/design.md
git rm ./.kiro/specs/comprehensive-test-suite/requirements.md
git rm ./.kiro/specs/comprehensive-test-suite/tasks.md
git rm ./.kiro/steering/nephoran-code-analyzer.md
git rm ./.kiro/steering/nephoran-docs-specialist.md
git rm ./.kiro/steering/nephoran-troubleshooter.md
git rm ./.kiro/steering/product.md
git rm ./.kiro/steering/structure.md
git rm ./.kiro/steering/tech.md

echo "Removing diagnostic and temporary files..."
git rm ./administrator_report.md
git rm ./cmd/llm-processor/main_original.go
git rm ./diagnostic_output.txt
git rm ./final_administrator_report.md

echo "Removing test executables..."
git rm ./llm.test.exe

# Directory cleanup (if empty after file removal)
echo "Cleaning up empty directories..."
rmdir ./.kiro/specs/comprehensive-test-suite 2>/dev/null || true
rmdir ./.kiro/specs 2>/dev/null || true
rmdir ./.kiro/steering 2>/dev/null || true

# Post-cleanup validation
echo "Validating system after cleanup..."
make build-all || { echo "Post-cleanup build failed"; exit 1; }

echo "Running post-cleanup tests..."
go test ./... || { echo "Post-cleanup tests failed"; exit 1; }

echo "Cleanup completed successfully"
echo "Removed 14 obsolete files and cleaned up empty directories"

echo ""
echo "ROLLBACK COMMANDS (if needed):"
echo "git checkout HEAD^ -- ./.kiro/specs/comprehensive-test-suite/design.md"
echo "git checkout HEAD^ -- ./.kiro/specs/comprehensive-test-suite/requirements.md"
echo "git checkout HEAD^ -- ./.kiro/specs/comprehensive-test-suite/tasks.md"
echo "git checkout HEAD^ -- ./.kiro/steering/nephoran-code-analyzer.md"
echo "git checkout HEAD^ -- ./.kiro/steering/nephoran-docs-specialist.md"
echo "git checkout HEAD^ -- ./.kiro/steering/nephoran-troubleshooter.md"
echo "git checkout HEAD^ -- ./.kiro/steering/product.md"
echo "git checkout HEAD^ -- ./.kiro/steering/structure.md"
echo "git checkout HEAD^ -- ./.kiro/steering/tech.md"
echo "git checkout HEAD^ -- ./administrator_report.md"
echo "git checkout HEAD^ -- ./cmd/llm-processor/main_original.go"
echo "git checkout HEAD^ -- ./diagnostic_output.txt"
echo "git checkout HEAD^ -- ./final_administrator_report.md"
echo "git checkout HEAD^ -- ./llm.test.exe"