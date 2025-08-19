#!/bin/bash
# Mock porch executable that always fails
if [ "$1" = "--help" ]; then
    echo "Mock Porch - Package Orchestration Tool (Failure Mode)"
    exit 0
fi

echo "Error: Failed to process intent file: $2" >&2
echo "Error: Invalid intent format detected" >&2
echo "Error: Missing required field: spec.target" >&2
echo "Processing failed after 0.123s" >&2
exit 1