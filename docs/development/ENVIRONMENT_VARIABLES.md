# Nephoran Intent Operator - Environment Variables Reference Guide

## Table of Contents
- [Overview](#overview)
- [Core Feature Toggles](#core-feature-toggles)
- [LLM Configuration](#llm-configuration)
- [HTTP Configuration](#http-configuration)
- [Metrics & Observability](#metrics--observability)
- [Configuration Examples](#configuration-examples)
- [Security Considerations](#security-considerations)
- [Troubleshooting](#troubleshooting)
- [Migration Guide](#migration-guide)

## Overview

The Nephoran Intent Operator uses environment variables for runtime configuration, allowing flexible deployment across different environments without code changes. This document provides a comprehensive reference for all supported environment variables.

### Configuration Precedence
1. Environment variables (highest priority)
2. Configuration files
3. Default values (lowest priority)

### Variable Types
- **Boolean**: Accepts `true`, `false`, `1`, `0` (case-insensitive)
- **Integer**: Numeric values, must be valid integers
- **String**: Text values, some with specific formats
- **List**: Comma-separated values (e.g., `"192.168.1.1,10.0.0.1"`)

## Core Feature Toggles

### ENABLE_NETWORK_INTENT
- **Type**: Boolean
- **Default**: `true`
- **Description**: Enables or disables the NetworkIntent controller functionality
- **Valid Values**: `true`, `false`, `1`, `0`
- **Usage**: Controls whether the operator processes NetworkIntent custom resources
- **Example**:
  ```bash
  export ENABLE_NETWORK_INTENT=true  # Enable NetworkIntent processing
  export ENABLE_NETWORK_INTENT=false # Disable NetworkIntent processing
  ```
- **Notes**: 
  - When disabled, NetworkIntent CRDs will not be reconciled
  - Useful for maintenance mode or troubleshooting
  - Does not affect existing deployed resources

### ENABLE_LLM_INTENT
- **Type**: Boolean
- **Default**: `false`
- **Description**: Enables or disables LLM (Large Language Model) intent processing
- **Valid Values**: `true`, `false`, `1`, `0`
- **Usage**: Controls whether intents are processed through the LLM service for natural language understanding
- **Example**:
  ```bash
  export ENABLE_LLM_INTENT=true  # Enable LLM processing
  export ENABLE_LLM_INTENT=false # Disable LLM processing (default)
  ```
- **Notes**:
  - Requires LLM service to be configured and accessible
  - When enabled, adds AI-powered intent interpretation capabilities
  - May increase processing latency and costs
  - Recommended for production environments with LLM infrastructure

## LLM Configuration

### LLM_TIMEOUT_SECS
- **Type**: Integer (seconds)
- **Default**: `15`
- **Description**: Timeout duration for individual LLM API requests
- **Valid Range**: `1` to `300` (1 second to 5 minutes)
- **Usage**: Controls how long to wait for LLM responses before timing out
- **Example**:
  ```bash
  export LLM_TIMEOUT_SECS=30  # 30-second timeout
  export LLM_TIMEOUT_SECS=5   # 5-second timeout for fast failures
  ```
- **Notes**:
  - Shorter timeouts reduce wait time but may cause premature failures
  - Longer timeouts accommodate complex queries but may delay error detection
  - Consider network latency and LLM model complexity when setting

### LLM_MAX_RETRIES
- **Type**: Integer
- **Default**: `2`
- **Description**: Maximum number of retry attempts for failed LLM requests
- **Valid Range**: `0` to `10`
- **Usage**: Improves resilience against transient failures
- **Example**:
  ```bash
  export LLM_MAX_RETRIES=3    # Retry up to 3 times
  export LLM_MAX_RETRIES=0    # No retries (fail fast)
  ```
- **Notes**:
  - Retries use exponential backoff (1s, 2s, 4s, ...)
  - Total time = initial attempt + retry attempts
  - Set to 0 for development/testing to fail fast
  - Higher values improve reliability but increase latency

### LLM_CACHE_MAX_ENTRIES
- **Type**: Integer
- **Default**: `512`
- **Description**: Maximum number of entries in the LLM response cache
- **Valid Range**: `1` to `10000`
- **Usage**: Controls memory usage and cache effectiveness
- **Example**:
  ```bash
  export LLM_CACHE_MAX_ENTRIES=1024  # Larger cache for high-traffic
  export LLM_CACHE_MAX_ENTRIES=128   # Smaller cache for memory-constrained
  ```
- **Notes**:
  - Cache uses LRU (Least Recently Used) eviction
  - Each entry consumes approximately 1-5KB of memory
  - Higher values improve hit rate but increase memory usage
  - Monitor cache hit rate via metrics to optimize size

## HTTP Configuration

### HTTP_MAX_BODY
- **Type**: Integer (bytes)
- **Default**: `1048576` (1MB)
- **Description**: Maximum allowed size for HTTP request bodies
- **Valid Range**: `1024` to `104857600` (1KB to 100MB)
- **Usage**: Prevents resource exhaustion from large requests
- **Example**:
  ```bash
  export HTTP_MAX_BODY=2097152   # 2MB limit
  export HTTP_MAX_BODY=10485760  # 10MB for large intents
  export HTTP_MAX_BODY=524288    # 512KB for strict limits
  ```
- **Notes**:
  - Applies to all HTTP endpoints
  - Requests exceeding limit receive 413 (Payload Too Large) error
  - Consider network bandwidth and processing capacity
  - Larger values may require increased memory allocation

## Metrics & Observability

### METRICS_ENABLED
- **Type**: Boolean
- **Default**: `false`
- **Description**: Enables or disables the Prometheus metrics endpoint
- **Valid Values**: `true`, `false`, `1`, `0`
- **Usage**: Controls exposure of `/metrics` endpoint
- **Example**:
  ```bash
  export METRICS_ENABLED=true   # Enable metrics endpoint
  export METRICS_ENABLED=false  # Disable metrics (default)
  ```
- **Notes**:
  - When enabled, exposes metrics on configured metrics port
  - Essential for production monitoring
  - May expose sensitive operational data
  - Should be used with METRICS_ALLOWED_IPS for security

### METRICS_ALLOWED_IPS
- **Type**: String (comma-separated IP list or wildcard)
- **Default**: `""` (empty - no access when metrics enabled)
- **Description**: IP addresses allowed to access the metrics endpoint
- **Valid Values**: 
  - Comma-separated IP addresses: `"192.168.1.100,10.0.0.1"`
  - Wildcard for unrestricted: `"*"`
  - Empty string: `""` (blocks all access)
- **Usage**: Restricts metrics endpoint access for security
- **Example**:
  ```bash
  # Allow specific IPs only
  export METRICS_ALLOWED_IPS="192.168.1.100,10.0.0.1,127.0.0.1"
  
  # Allow all IPs (use with caution)
  export METRICS_ALLOWED_IPS="*"
  
  # Block all access (default when empty)
  export METRICS_ALLOWED_IPS=""
  
  # Allow monitoring system and localhost
  export METRICS_ALLOWED_IPS="10.0.0.50,127.0.0.1"
  ```
- **Security Notes**:
  - Never use `"*"` in production without additional network security
  - Prefer specific IP allowlisting for defense in depth
  - Consider using service mesh or network policies for additional security
  - Empty string with METRICS_ENABLED=true effectively disables metrics access

## Configuration Examples

### Development Environment
```bash
# Enable all features for testing
export ENABLE_NETWORK_INTENT=true
export ENABLE_LLM_INTENT=true
export LLM_TIMEOUT_SECS=5          # Fast failures for development
export LLM_MAX_RETRIES=0           # No retries to speed up testing
export LLM_CACHE_MAX_ENTRIES=128   # Small cache for low memory usage
export HTTP_MAX_BODY=524288        # 512KB for basic testing
export METRICS_ENABLED=true
export METRICS_ALLOWED_IPS="*"     # Open access in dev environment
```

### Production Environment
```bash
# Production-optimized configuration
export ENABLE_NETWORK_INTENT=true
export ENABLE_LLM_INTENT=true
export LLM_TIMEOUT_SECS=30         # Allow time for complex queries
export LLM_MAX_RETRIES=3           # Retry for resilience
export LLM_CACHE_MAX_ENTRIES=2048  # Large cache for performance
export HTTP_MAX_BODY=2097152       # 2MB for complex intents
export METRICS_ENABLED=true
export METRICS_ALLOWED_IPS="10.0.0.50,10.0.0.51"  # Monitoring systems only
```

### Minimal Configuration
```bash
# Bare minimum configuration
export ENABLE_NETWORK_INTENT=true  # Core functionality only
# All other variables use defaults
```

### High-Security Environment
```bash
# Security-focused configuration
export ENABLE_NETWORK_INTENT=true
export ENABLE_LLM_INTENT=false     # Disable external LLM calls
export HTTP_MAX_BODY=262144        # 256KB strict limit
export METRICS_ENABLED=false       # No metrics exposure
# Or with restricted metrics:
# export METRICS_ENABLED=true
# export METRICS_ALLOWED_IPS="10.0.0.100"  # Single monitoring host
```

## Security Considerations

### Metrics Endpoint Security
1. **Never expose metrics publicly without IP restrictions**
   - Metrics can reveal system internals and performance data
   - Use `METRICS_ALLOWED_IPS` to restrict access
   - Consider network policies or firewall rules as additional layers

2. **IP Allowlisting Best Practices**
   - Use specific IPs instead of wildcards in production
   - Regularly audit and update allowed IP list
   - Monitor access logs for unauthorized attempts
   - Consider using service mesh for mTLS authentication

### LLM Security
1. **API Key Protection**
   - Store LLM API keys in Kubernetes secrets
   - Never log or expose API keys in metrics
   - Rotate keys regularly

2. **Request Size Limits**
   - Set appropriate `HTTP_MAX_BODY` to prevent DoS attacks
   - Monitor request patterns for anomalies
   - Implement rate limiting at ingress level

### Cache Security
1. **Cache Poisoning Prevention**
   - Validate all cached data before use
   - Set appropriate cache sizes to prevent memory exhaustion
   - Monitor cache hit rates and evictions

## Troubleshooting

### Common Issues and Solutions

#### Issue: LLM requests timing out frequently
**Symptoms**: Logs show timeout errors, high latency
**Solutions**:
1. Increase `LLM_TIMEOUT_SECS` (e.g., from 15 to 30)
2. Check network connectivity to LLM service
3. Verify LLM service health and capacity
4. Consider reducing request complexity

#### Issue: High memory usage
**Symptoms**: OOMKilled pods, memory alerts
**Solutions**:
1. Reduce `LLM_CACHE_MAX_ENTRIES` (e.g., from 512 to 256)
2. Decrease `HTTP_MAX_BODY` limit
3. Monitor cache eviction rates
4. Scale horizontally instead of vertically

#### Issue: Metrics endpoint not accessible
**Symptoms**: 403 Forbidden or connection refused
**Solutions**:
1. Verify `METRICS_ENABLED=true`
2. Check `METRICS_ALLOWED_IPS` includes client IP
3. Verify network policies allow traffic
4. Check service and ingress configurations

#### Issue: Intent processing disabled unexpectedly
**Symptoms**: NetworkIntents not being processed
**Solutions**:
1. Verify `ENABLE_NETWORK_INTENT=true`
2. Check controller logs for startup errors
3. Verify RBAC permissions are correct
4. Ensure CRDs are installed

### Debug Commands

```bash
# Check current environment variables
kubectl exec -n nephoran-system deployment/nephoran-controller -- env | grep -E 'ENABLE_|LLM_|HTTP_|METRICS_'

# Verify configuration is loaded
kubectl logs -n nephoran-system deployment/nephoran-controller | grep -i "configuration"

# Test metrics endpoint access
curl -v http://<pod-ip>:<metrics-port>/metrics

# Check cache statistics
kubectl exec -n nephoran-system deployment/nephoran-controller -- curl localhost:8080/debug/cache/stats
```

## Migration Guide

### Migrating from Previous Versions

#### From v0.x to v1.x
- `MAX_REQUEST_SIZE` → `HTTP_MAX_BODY` (same functionality)
- `METRICS_EXPOSE_PUBLICLY` → Use `METRICS_ENABLED` + `METRICS_ALLOWED_IPS="*"`
- `LLM_TIMEOUT` → `LLM_TIMEOUT_SECS` (now in seconds, not duration string)

#### Deprecated Variables (will be removed in v2.0)
- `MAX_REQUEST_SIZE` - Use `HTTP_MAX_BODY` instead
- `METRICS_EXPOSE_PUBLICLY` - Use combination of `METRICS_ENABLED` and `METRICS_ALLOWED_IPS`

### Environment Variable Validation

The operator performs validation on startup and will:
1. Log warnings for invalid values (uses defaults)
2. Log info messages for non-default configurations
3. Fail fast for critical misconfigurations

### Best Practices

1. **Use Configuration Management**
   - Store configurations in ConfigMaps or Secrets
   - Use Helm values or Kustomize for environment-specific configs
   - Version control all configuration changes

2. **Monitor Configuration Changes**
   - Set up alerts for configuration drift
   - Log all configuration changes
   - Use GitOps for configuration deployment

3. **Test Configuration Changes**
   - Always test in non-production first
   - Use canary deployments for gradual rollout
   - Have rollback procedures ready

4. **Document Custom Configurations**
   - Maintain environment-specific documentation
   - Document reasons for non-default values
   - Keep configuration changelog

## Related Documentation

- [Deployment Guide](./DEPLOYMENT.md) - Deployment procedures and requirements
- [Security Guide](./SECURITY.md) - Security best practices and configurations
- [Monitoring Guide](./MONITORING.md) - Metrics and observability setup
- [API Reference](./API_REFERENCE.md) - REST API documentation
- [Troubleshooting Guide](./TROUBLESHOOTING.md) - Common issues and solutions

## Support

For issues or questions about environment variables:
1. Check the troubleshooting section above
2. Review operator logs for configuration errors
3. Consult the project's GitHub issues
4. Contact the development team

---

*Last Updated: December 2024*
*Version: 1.0.0*