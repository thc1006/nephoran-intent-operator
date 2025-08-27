#!/bin/bash
# Script for running specific golangci-lint rules in isolation
# Usage: ./lint-isolated.sh <linter> [path] [additional-args...]

set -euo pipefail

# Configuration
LINTER="${1:-}"
PATH_TARGET="${2:-.}"
shift 2 || true
ADDITIONAL_ARGS=("$@")

# Available linters from .golangci.yml
VALID_LINTERS=(
    "revive" "staticcheck" "govet" "ineffassign" "errcheck" 
    "gocritic" "misspell" "unparam" "unconvert" "prealloc" "gosec"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
GRAY='\033[0;37m'
NC='\033[0m' # No Color

# Usage function
usage() {
    echo "Usage: $0 <linter> [path] [additional-args...]"
    echo "Available linters: ${VALID_LINTERS[*]}"
    exit 1
}

# Validation
if [[ -z "$LINTER" ]]; then
    echo "Error: Linter name is required"
    usage
fi

if [[ ! " ${VALID_LINTERS[*]} " =~ " $LINTER " ]]; then
    echo "Error: Invalid linter: $LINTER"
    echo "Valid options: ${VALID_LINTERS[*]}"
    exit 1
fi

# Build command
CMD=(
    "golangci-lint" "run"
    "--disable-all"
    "--enable" "$LINTER"
    "--timeout" "5m"
    "--out-format" "colored-line-number"
    "--print-issued-lines"
    "--print-linter-name"
    "$PATH_TARGET"
)

# Add any additional arguments
CMD+=("${ADDITIONAL_ARGS[@]}")

echo -e "${CYAN}Running isolated linter: $LINTER${NC}"
echo -e "${GRAY}Command: ${CMD[*]}${NC}"
echo -e "${GRAY}Path: $PATH_TARGET${NC}"
echo ""

# Execute command
START_TIME=$(date +%s.%N)
if "${CMD[@]}"; then
    END_TIME=$(date +%s.%N)
    DURATION=$(echo "$END_TIME - $START_TIME" | bc -l)
    printf "${GREEN}✅ %s passed in %.2f seconds${NC}\n" "$LINTER" "$DURATION"
    exit 0
else
    EXIT_CODE=$?
    END_TIME=$(date +%s.%N)
    DURATION=$(echo "$END_TIME - $START_TIME" | bc -l)
    printf "${RED}❌ %s found issues in %.2f seconds${NC}\n" "$LINTER" "$DURATION"
    exit $EXIT_CODE
fi