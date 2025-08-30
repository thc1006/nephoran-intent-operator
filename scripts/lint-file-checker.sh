#!/bin/bash
# Script for checking specific files before and after linting fixes
# Usage: ./lint-file-checker.sh <file-or-pattern> [linter]

set -euo pipefail

FILE_PATTERN="${1:-}"
LINTER="${2:-}"
TEMP_DIR="/tmp/lint-checker-$$"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
GRAY='\033[0;37m'
NC='\033[0m'

usage() {
    echo "Usage: $0 <file-or-pattern> [linter]"
    echo "Example: $0 pkg/controllers/*.go revive"
    echo "Example: $0 internal/loop/processor.go"
    exit 1
}

cleanup() {
    rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

if [[ -z "$FILE_PATTERN" ]]; then
    usage
fi

# Find matching Go files
mapfile -t FILES < <(find . -path "$FILE_PATTERN" -name "*.go" -type f | grep -v -E '_test\.go$|_mock\.go$|\.mock\.go$|_generated\.go$|\.pb\.go$|zz_generated\.')

if [[ ${#FILES[@]} -eq 0 ]]; then
    echo -e "${YELLOW}No Go files found matching pattern: $FILE_PATTERN${NC}"
    exit 0
fi

echo -e "${CYAN}File Lint Checker${NC}"
echo -e "${GRAY}Files to check: ${#FILES[@]}${NC}"
for file in "${FILES[@]}"; do
    echo -e "${GRAY}  - $file${NC}"
done
echo ""

# Create temp directory for backups
mkdir -p "$TEMP_DIR"

# Function to run linter on specific files
run_lint_check() {
    local linter_name="$1"
    shift
    local files=("$@")
    
    echo -e "${YELLOW}Checking $linter_name on ${#files[@]} files...${NC}"
    
    local cmd=(
        "golangci-lint" "run"
        "--disable-all"
        "--enable" "$linter_name"
        "--timeout" "30s"
        "--out-format" "colored-line-number"
        "--print-issued-lines=false"
        "--print-linter-name"
    )
    
    # Add file paths
    cmd+=("${files[@]}")
    
    local start_time
    start_time=$(date +%s.%N)
    
    if "${cmd[@]}" 2>&1; then
        local end_time
        end_time=$(date +%s.%N)
        local duration
        duration=$(echo "$end_time - $start_time" | bc -l)
        printf "${GREEN}  ✅ %s passed (%.2fs)${NC}\n" "$linter_name" "$duration"
        return 0
    else
        local exit_code=$?
        local end_time
        end_time=$(date +%s.%N)
        local duration
        duration=$(echo "$end_time - $start_time" | bc -l)
        printf "${RED}  ❌ %s found issues (%.2fs)${NC}\n" "$linter_name" "$duration"
        return $exit_code
    fi
}

# Available linters
AVAILABLE_LINTERS=(
    "revive" "staticcheck" "govet" "ineffassign" "errcheck" 
    "gocritic" "misspell" "unparam" "unconvert" "prealloc" "gosec"
)

if [[ -n "$LINTER" ]]; then
    # Check specific linter
    if [[ ! " ${AVAILABLE_LINTERS[*]} " =~ " $LINTER " ]]; then
        echo -e "${RED}Invalid linter: $LINTER${NC}"
        echo "Available: ${AVAILABLE_LINTERS[*]}"
        exit 1
    fi
    
    run_lint_check "$LINTER" "${FILES[@]}"
    exit $?
else
    # Check all linters
    echo -e "${CYAN}Running all linters individually...${NC}"
    echo ""
    
    passed=0
    failed=0
    
    for linter in "${AVAILABLE_LINTERS[@]}"; do
        if run_lint_check "$linter" "${FILES[@]}"; then
            ((passed++))
        else
            ((failed++))
        fi
    done
    
    echo ""
    echo -e "${CYAN}Summary for ${#FILES[@]} files:${NC}"
    echo -e "${GREEN}Passed: $passed${NC}"
    echo -e "${RED}Failed: $failed${NC}"
    
    if [[ $failed -gt 0 ]]; then
        exit 1
    fi
fi