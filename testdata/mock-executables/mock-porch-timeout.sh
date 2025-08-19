#!/bin/bash
# Mock porch executable that times out (sleeps for a long time)
if [ "$1" = "--help" ]; then
    echo "Mock Porch - Package Orchestration Tool (Timeout Mode)"
    exit 0
fi

echo "Starting long-running process..."
sleep 60
echo "This should not be reached due to timeout"
exit 0