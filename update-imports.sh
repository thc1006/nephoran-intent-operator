#\!/bin/bash
set -e

# Find and replace all import paths
find . -type f -name "*.go" -print0 | xargs -0 sed -i "s|github.com/thc1006/nephoran-intent-operator|github.com/nephio-project/nephoran-intent-operator|g"

# Verify replacements
echo "Import path replacements complete. Checking for any remaining references..."
if grep -r "github.com/thc1006/nephoran-intent-operator" .; then
    echo "Warning: Some references to the old module path remain."
    exit 1
else
    echo "All import paths successfully updated."
fi
