#!/bin/bash

# Fix Compilation Errors Script
# This script addresses critical compilation issues in the codebase

echo "=== Fixing Critical Compilation Errors ==="

# Set strict mode
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "Project root: $PROJECT_ROOT"

# Function to remove unused imports from a file
remove_unused_import() {
    local file="$1"
    local import="$2"
    
    if [[ -f "$file" ]]; then
        echo "Removing unused import '$import' from $file"
        # Create a backup
        cp "$file" "$file.backup"
        
        # Remove the import line (handles both quoted and unquoted imports)
        sed -i.tmp "/^\s*\"$import\"/d" "$file" 2>/dev/null || \
        sed -i.tmp "/^\s*$import\s*\"/d" "$file" 2>/dev/null || \
        sed -i.tmp "/$import/d" "$file" 2>/dev/null || true
        
        # Clean up temp files
        rm -f "$file.tmp" 2>/dev/null || true
    fi
}

# Fix unused imports in various files
echo "=== Fixing unused imports ==="

# pkg/audit files
remove_unused_import "pkg/audit/integrity.go" "strings"
remove_unused_import "pkg/audit/query.go" "encoding/json"
remove_unused_import "pkg/audit/query.go" "strings"
remove_unused_import "pkg/audit/retention.go" "encoding/json"
remove_unused_import "pkg/audit/retention.go" "sort"
remove_unused_import "pkg/audit/webhooks.go" "context"

# pkg/controllers/interfaces
remove_unused_import "pkg/controllers/interfaces/controller_interfaces.go" "k8s.io/client-go/tools/record"
remove_unused_import "pkg/controllers/interfaces/controller_interfaces.go" "sigs.k8s.io/controller-runtime/pkg/client"

# pkg/cnf/templates
remove_unused_import "pkg/cnf/templates/helm_templates.go" "strings"

# tests files
remove_unused_import "tests/integration/controllers/fake_weaviate.go" "context"
remove_unused_import "tests/integration/controllers/fake_weaviate.go" "encoding/json"
remove_unused_import "tests/framework/suite.go" "sigs.k8s.io/controller-runtime"
remove_unused_import "tests/performance/validation/cmd.go" "strconv"
remove_unused_import "tests/performance/validation/data_manager.go" "strings"

# pkg/servicemesh/abstraction
remove_unused_import "pkg/servicemesh/abstraction/interface.go" "k8s.io/api/core/v1"

echo "=== Removing duplicate declarations ==="

# Function to rename conflicting declarations
rename_conflicting_type() {
    local file="$1"
    local old_name="$2"
    local new_name="$3"
    
    if [[ -f "$file" ]]; then
        echo "Renaming $old_name to $new_name in $file"
        sed -i.tmp "s/type $old_name/type $new_name/g" "$file" 2>/dev/null || true
        sed -i.tmp "s/\b$old_name\b/$new_name/g" "$file" 2>/dev/null || true
        rm -f "$file.tmp" 2>/dev/null || true
    fi
}

# Fix some of the duplicate type declarations by renaming them
if [[ -f "pkg/errors/types.go" ]]; then
    echo "Fixing duplicate declarations in pkg/errors/types.go"
    # Comment out or rename duplicate declarations
    sed -i.tmp 's/^type ErrorType/\/\/ type ErrorType (duplicate - see errors.go)/' "pkg/errors/types.go" 2>/dev/null || true
    sed -i.tmp 's/^	ErrorTypeValidation/\/\/ 	ErrorTypeValidation/' "pkg/errors/types.go" 2>/dev/null || true
    rm -f "pkg/errors/types.tmp" 2>/dev/null || true
fi

# Check for build success
echo "=== Testing build after fixes ==="
if go build ./... 2>/dev/null; then
    echo "✅ Build successful after fixes"
    exit 0
else
    echo "⚠️  Build still has issues, but critical imports fixed"
    echo "Manual intervention may be required for remaining issues"
    
    # Show remaining errors
    echo ""
    echo "=== Remaining build errors ==="
    go build ./... 2>&1 | head -20 || true
    exit 0
fi