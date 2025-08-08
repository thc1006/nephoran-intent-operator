#!/bin/bash
# Pre-commit security checks for developers

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "🔒 Running pre-commit security checks..."

# Check for secrets
echo "Checking for secrets..."
if command -v gitleaks &> /dev/null; then
    gitleaks protect --staged --redact --verbose || {
        echo -e "${RED}❌ Secrets detected! Please remove them before committing.${NC}"
        exit 1
    }
else
    echo -e "${YELLOW}⚠️  Gitleaks not installed. Install with: go install github.com/zricethezav/gitleaks/v8@latest${NC}"
fi

# Run gosec on changed Go files
echo "Running security analysis on Go files..."
if command -v gosec &> /dev/null; then
    changed_go_files=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)
    if [ -n "$changed_go_files" ]; then
        gosec -fmt text $changed_go_files || {
            echo -e "${YELLOW}⚠️  Security issues found. Review and fix if necessary.${NC}"
        }
    fi
else
    echo -e "${YELLOW}⚠️  GoSec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest${NC}"
fi

# Check for vulnerable dependencies
echo "Checking for vulnerable dependencies..."
if command -v govulncheck &> /dev/null; then
    govulncheck ./... || {
        echo -e "${YELLOW}⚠️  Vulnerabilities found in dependencies.${NC}"
    }
else
    echo -e "${YELLOW}⚠️  govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest${NC}"
fi

echo -e "${GREEN}✅ Pre-commit security checks completed${NC}"