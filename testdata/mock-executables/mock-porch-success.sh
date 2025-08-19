#!/bin/bash
# Mock porch executable that always succeeds
if [ "$1" = "--help" ]; then
    echo "Mock Porch - Package Orchestration Tool"
    echo "Usage: porch [options]"
    echo "Options:"
    echo "  -intent PATH    Path to intent file"
    echo "  -out PATH       Output directory"
    echo "  -structured     Use structured output mode"
    echo "  --help          Show this help message"
    exit 0
fi

echo "Processing intent file: $2"
echo "Output directory: $4"
echo "Mode: $5"
echo "Package generated successfully at $4"
echo "Generated manifest files:"
echo "  - deployment.yaml"
echo "  - service.yaml"
echo "  - configmap.yaml"
echo "Processing completed in 1.234s"
exit 0