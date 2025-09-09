#!/bin/bash
# ci-local.sh - Local CI checks and reproduction

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
CONFIG_FILE="$SCRIPT_DIR/ci-bot.yaml"
REPORTS_DIR="$SCRIPT_DIR/.reports"

# Create reports directory
mkdir -p "$REPORTS_DIR"

# Load configuration
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: Configuration file $CONFIG_FILE not found"
    exit 1
fi

# Function to run local checks
run_local_checks() {
    echo "🔍 Running local CI checks..."
    
    # Set test environment variables
    export LLM_ALLOWED_ORIGINS="http://localhost:3000,http://127.0.0.1:3000"
    export CORS_ENABLED="true"
    
    # Run Go local checks in order
    echo "📦 go mod tidy"
    go mod tidy
    
    echo "🎨 go fmt ./..."
    go fmt ./...
    
    echo "🔎 go vet ./..."
    if ! go vet ./...; then
        echo "❌ go vet failed"
        return 1
    fi
    
    echo "🔨 go build ./..."
    if ! go build ./...; then
        echo "❌ go build failed"
        return 1
    fi
    
    echo "🧪 go test ./..."
    if ! go test -count=1 ./...; then
        echo "❌ go test failed"
        return 1
    fi
    
    echo "✅ All local checks passed"
    return 0
}

# Function to create failure report
create_failure_report() {
    local timestamp=$(date -Iseconds)
    local report_file="$REPORTS_DIR/local-summary-$timestamp.md"
    
    cat > "$report_file" << 'EOF'
# Local CI Failure Report

## Summary
Local CI checks failed. Review the output above for details.

## Common Issues
- Missing required schema fields in intent validation
- Configuration validation errors (LLM_ALLOWED_ORIGINS)
- Test assertion failures in circuit breaker logic
- Schema path resolution issues

## Recommended Actions
1. Fix intent schema validation issues
2. Set proper test environment variables
3. Update test assertions to match actual behavior
4. Resolve missing file dependencies

## Reproduction Commands
```bash
export LLM_ALLOWED_ORIGINS="http://localhost:3000,http://127.0.0.1:3000"
export CORS_ENABLED="true"
go test -v ./cmd/intent-ingest
go test -v ./cmd/llm-processor  
go test -v ./internal/config
```
EOF
    
    echo "📊 Failure report created: $report_file"
}

# Main execution
if run_local_checks; then
    echo "🎉 Local CI checks completed successfully"
    exit 0
else
    echo "💥 Local CI checks failed"
    create_failure_report
    exit 1
fi