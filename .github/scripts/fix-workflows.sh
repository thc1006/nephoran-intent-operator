#!/bin/bash
# Script to fix Go version configurations in all GitHub Actions workflows
# Ensures all workflows use go.mod as the single source of truth for Go version

set -euo pipefail

echo "=== Fixing Go version configurations in GitHub Actions workflows ==="

# Find all workflow files
WORKFLOWS=$(find .github/workflows -name "*.yml" -o -name "*.yaml" | sort)

for workflow in $WORKFLOWS; do
    echo "Processing: $workflow"
    
    # Skip if already fixed (has go-version-file: go.mod)
    if grep -q "go-version-file: go.mod" "$workflow" 2>/dev/null; then
        echo "  ✓ Already using go.mod for version"
        continue
    fi
    
    # Check if file contains Go setup
    if ! grep -q "actions/setup-go" "$workflow" 2>/dev/null; then
        echo "  - No Go setup found, skipping"
        continue
    fi
    
    echo "  Fixing Go setup configurations..."
    
    # Create a temporary file
    tmpfile=$(mktemp)
    
    # Process the file
    awk '
    BEGIN { in_setup_go = 0; fixed = 0 }
    
    # Remove GO_VERSION and GO_QUALITY_VERSION env vars
    /^\s*GO_VERSION:/ || /^\s*GO_QUALITY_VERSION:/ { next }
    
    # Remove GOTOOLCHAIN: local
    /^\s*GOTOOLCHAIN:\s*local/ { next }
    
    # Detect setup-go action
    /uses:\s*actions\/setup-go/ {
        in_setup_go = 1
        print $0
        # Add id if not present
        if (!match($0, /id:/)) {
            indent = match($0, /[^ ]/) - 1
            printf "%*sid: setup-go\n", indent + 2, ""
        }
        next
    }
    
    # Inside setup-go block
    in_setup_go && /^\s*with:/ {
        print $0
        in_with = 1
        next
    }
    
    # Replace go-version with go-version-file
    in_setup_go && in_with && /^\s*go-version:/ {
        indent = match($0, /[^ ]/) - 1
        printf "%*sgo-version-file: go.mod\n", indent, ""
        fixed = 1
        next
    }
    
    # Keep other setup-go options
    in_setup_go && in_with && /^\s*(check-latest|cache):/ {
        print $0
        next
    }
    
    # End of setup-go block - add version print
    in_setup_go && /^\s*-\s*name:/ {
        if (fixed) {
            # Add Go version print step
            indent = match($0, /[^ ]/) - 1
            printf "\n%*s- name: Print Go version\n", indent, ""
            printf "%*s  run: |\n", indent, ""
            printf "%*s    echo \"Go version: $(go version)\"\n", indent + 4, ""
            printf "%*s    echo \"Go env:\"\n", indent + 4, ""
            printf "%*s    go env\n\n", indent + 4, ""
            fixed = 0
        }
        in_setup_go = 0
        in_with = 0
        print $0
        next
    }
    
    # Fix cache keys that reference GO_VERSION
    /key:.*\$\{\{\s*env\.GO_VERSION\s*\}\}/ {
        gsub(/\$\{\{\s*env\.GO_VERSION\s*\}\}/, "${{ steps.setup-go.outputs.go-version }}")
        print $0
        next
    }
    
    # Fix cache keys that reference GO_QUALITY_VERSION
    /key:.*\$\{\{\s*env\.GO_QUALITY_VERSION\s*\}\}/ {
        gsub(/\$\{\{\s*env\.GO_QUALITY_VERSION\s*\}\}/, "${{ steps.setup-go.outputs.go-version }}")
        print $0
        next
    }
    
    # Fix restore-keys
    /restore-keys:.*go-\$\{\{\s*env\.GO_VERSION\s*\}\}/ {
        gsub(/go-\$\{\{\s*env\.GO_VERSION\s*\}\}/, "go-${{ steps.setup-go.outputs.go-version }}")
        print $0
        next
    }
    
    # Default: print line as-is
    { print $0 }
    ' "$workflow" > "$tmpfile"
    
    # Check if changes were made
    if ! cmp -s "$workflow" "$tmpfile"; then
        mv "$tmpfile" "$workflow"
        echo "  ✓ Fixed Go version configuration"
    else
        rm "$tmpfile"
        echo "  - No changes needed"
    fi
done

# Fix golangci-lint-action versions
echo ""
echo "=== Updating golangci-lint-action versions ==="
for workflow in $WORKFLOWS; do
    if grep -q "golangci/golangci-lint-action" "$workflow" 2>/dev/null; then
        echo "Processing: $workflow"
        
        # Update golangci-lint version to latest
        sed -i.bak -E '
            # Update version in golangci-lint-action
            /golangci-lint-action@v[0-9]/{
                N
                s/(version:\s*)(v[0-9]+\.[0-9]+\.[0-9]+|latest)/\1latest/
            }
            # Update args to include timeout
            /args:.*--verbose/{
                s/--timeout=[0-9]+m/--timeout=5m/
                /--timeout/!s/--verbose/--verbose --timeout=5m/
            }
        ' "$workflow"
        
        # Check if changed
        if ! cmp -s "$workflow" "$workflow.bak"; then
            echo "  ✓ Updated golangci-lint configuration"
            rm "$workflow.bak"
        else
            echo "  - No changes needed"
            rm "$workflow.bak"
        fi
    fi
done

echo ""
echo "=== Summary ==="
echo "✓ All workflow files processed"
echo "✓ Go version now sourced from go.mod"
echo "✓ Cache keys updated to use resolved Go version"
echo "✓ golangci-lint updated to latest version"
echo ""
echo "Next steps:"
echo "1. Review the changes with: git diff .github/workflows/"
echo "2. Test workflows locally with act or push to a test branch"
echo "3. Commit changes with appropriate message"