#!/bin/bash

# Fix import paths from nephio-project to thc1006
find . -name "*.go" -type f -exec grep -l "github.com/nephio-project/nephoran-intent-operator" {} \; | while read file; do
    echo "Fixing imports in: $file"
    sed -i 's|github.com/nephio-project/nephoran-intent-operator|github.com/thc1006/nephoran-intent-operator|g' "$file"
done

echo "Import path fixes complete!"