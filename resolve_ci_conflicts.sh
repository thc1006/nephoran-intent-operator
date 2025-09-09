#!/bin/bash

# Resolve all conflicts in ci-2025.yml by choosing Go 1.23.4 and enhanced features

file=".github/workflows/ci-2025.yml"

# Use sed to remove conflict markers and choose appropriate versions
# This keeps Go 1.23.4 but preserves enhanced features from integrate/mvp

# Remove all conflict markers and merge sections
awk '
BEGIN { in_conflict = 0; use_head = 0 }
/^<<<<<<< HEAD/ { in_conflict = 1; use_head = 1; next }
/^=======/ { use_head = 0; next }
/^>>>>>>> integrate\/mvp/ { in_conflict = 0; next }
!in_conflict { print }
in_conflict && use_head && /go-version.*1\.23\.4/ { print "          go-version: '\''1.23.4'\''" }
in_conflict && use_head && /GO_VERSION.*1\.23\.4/ { print "  GO_VERSION: \"1.23.4\"" }
in_conflict && use_head && /timeout-minutes: 5/ { print "    timeout-minutes: 10" }
in_conflict && use_head && /fetch-depth: 2/ { print "          fetch-depth: 1" }
in_conflict && use_head && /check-latest: true/ { print "          check-latest: false  # Ensure exact version" }
in_conflict && use_head && /cache: false/ { print "          cache: true  # We handle caching manually" }
in_conflict && !use_head && !/1\.24\.6|1\.25\.x|disabled|DISABLED/ { print }
' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"

echo "Conflicts resolved in $file"