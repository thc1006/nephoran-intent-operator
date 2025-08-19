#!/bin/bash
# validate-line-endings.sh - Check for Windows line endings in shell scripts
set -euo pipefail

echo "Validating line endings in shell scripts..."

# Find all .sh files and check for CRLF line endings
crlf_files=()
while IFS= read -r -d '' file; do
    if file "$file" | grep -q "CRLF"; then
        crlf_files+=("$file")
    fi
done < <(find . -name "*.sh" -print0)

if [ ${#crlf_files[@]} -gt 0 ]; then
    echo "ERROR: Found shell scripts with Windows line endings (CRLF):"
    printf '%s\n' "${crlf_files[@]}"
    echo ""
    echo "To fix these files, run:"
    echo "  sed -i 's/\r$//' \"\$file\""
    echo "Or use dos2unix if available:"
    echo "  dos2unix \"\$file\""
    exit 1
else
    echo "✓ All shell scripts have Unix line endings (LF)"
fi