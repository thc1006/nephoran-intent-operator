#!/bin/bash

# Example script showing how to configure the Nephoran Intent Operator
# using the 8 environment variables

echo "==================================================================="
echo "Nephoran Intent Operator - Environment Configuration Examples"
echo "==================================================================="
echo ""

# Development Environment Configuration
echo "1. DEVELOPMENT ENVIRONMENT (Local Testing)"
echo "-------------------------------------------"
cat << 'EOF'
export ENABLE_NETWORK_INTENT=true      # Enable NetworkIntent controller
export ENABLE_LLM_INTENT=true          # Enable LLM processing for testing
export LLM_TIMEOUT_SECS=30              # Longer timeout for debugging
export LLM_MAX_RETRIES=5                # More retries during development
export LLM_CACHE_MAX_ENTRIES=256       # Smaller cache for local testing
export HTTP_MAX_BODY=5242880           # 5MB for development
export METRICS_ENABLED=true            # Enable metrics for monitoring
export METRICS_ALLOWED_IPS="127.0.0.1,localhost"  # Local access only
EOF
echo ""

# Production Environment Configuration
echo "2. PRODUCTION ENVIRONMENT (Secure Defaults)"
echo "--------------------------------------------"
cat << 'EOF'
export ENABLE_NETWORK_INTENT=true      # Enable core functionality
export ENABLE_LLM_INTENT=false         # Disable AI features by default
export LLM_TIMEOUT_SECS=15              # Standard timeout
export LLM_MAX_RETRIES=2                # Limited retries
export LLM_CACHE_MAX_ENTRIES=512       # Standard cache size
export HTTP_MAX_BODY=1048576           # 1MB limit for security
export METRICS_ENABLED=false           # Disable metrics by default
export METRICS_ALLOWED_IPS=""          # No access when disabled
EOF
echo ""

# High-Performance Environment Configuration
echo "3. HIGH-PERFORMANCE ENVIRONMENT (Optimized)"
echo "--------------------------------------------"
cat << 'EOF'
export ENABLE_NETWORK_INTENT=true      # Enable core functionality
export ENABLE_LLM_INTENT=true          # Enable AI processing
export LLM_TIMEOUT_SECS=10              # Fast timeout for performance
export LLM_MAX_RETRIES=1                # Minimal retries
export LLM_CACHE_MAX_ENTRIES=2048      # Large cache for performance
export HTTP_MAX_BODY=2097152           # 2MB for larger payloads
export METRICS_ENABLED=true            # Enable for monitoring
export METRICS_ALLOWED_IPS="10.0.0.0/8"  # Internal network only
EOF
echo ""

# Security-Focused Environment Configuration
echo "4. HIGH-SECURITY ENVIRONMENT (Restricted)"
echo "------------------------------------------"
cat << 'EOF'
export ENABLE_NETWORK_INTENT=true      # Core functionality only
export ENABLE_LLM_INTENT=false         # No AI features
export LLM_TIMEOUT_SECS=5               # Minimal timeout
export LLM_MAX_RETRIES=0                # No retries
export LLM_CACHE_MAX_ENTRIES=0         # No caching
export HTTP_MAX_BODY=524288            # 512KB strict limit
export METRICS_ENABLED=false           # No metrics exposure
export METRICS_ALLOWED_IPS=""          # No access
EOF
echo ""

# Show current configuration
echo "5. CURRENT CONFIGURATION"
echo "------------------------"
echo "ENABLE_NETWORK_INTENT=${ENABLE_NETWORK_INTENT:-true (default)}"
echo "ENABLE_LLM_INTENT=${ENABLE_LLM_INTENT:-false (default)}"
echo "LLM_TIMEOUT_SECS=${LLM_TIMEOUT_SECS:-15 (default)}"
echo "LLM_MAX_RETRIES=${LLM_MAX_RETRIES:-2 (default)}"
echo "LLM_CACHE_MAX_ENTRIES=${LLM_CACHE_MAX_ENTRIES:-512 (default)}"
echo "HTTP_MAX_BODY=${HTTP_MAX_BODY:-1048576 (default)}"
echo "METRICS_ENABLED=${METRICS_ENABLED:-false (default)}"
echo "METRICS_ALLOWED_IPS=${METRICS_ALLOWED_IPS:-(empty default)}"
echo ""

# Validation helper
echo "6. VALIDATION HELPER"
echo "--------------------"
cat << 'EOF'
# Run this to validate your configuration:
go test -v ./pkg/config -run TestEnvironmentVariables

# Or test a specific variable:
ENABLE_LLM_INTENT=true go test -v ./pkg/config -run TestEnvironmentVariables/ENABLE_LLM_INTENT
EOF
echo ""

echo "==================================================================="
echo "For more details, see docs/ENVIRONMENT_VARIABLES.md"
echo "==================================================================="