# CORS Security Configuration Guide

## Table of Contents
1. [Overview](#overview)
2. [Security Implications](#security-implications)
3. [Environment Variables](#environment-variables)
4. [Configuration Examples](#configuration-examples)
5. [Validation Rules](#validation-rules)
6. [Migration Guide](#migration-guide)
7. [Troubleshooting](#troubleshooting)
8. [Best Practices](#best-practices)

## Overview

The Nephoran Intent Operator's LLM Processor service implements Cross-Origin Resource Sharing (CORS) with strict security controls to protect against unauthorized cross-origin requests. This guide explains how to configure CORS securely for different environments.

### Key Security Changes (v2.0.0+)
- **Default Deny**: CORS origins default to empty (no access) instead of wildcard (`*`)
- **Explicit Configuration**: Origins must be explicitly listed via `LLM_ALLOWED_ORIGINS`
- **Production Hardening**: Wildcard origins are blocked in production environments
- **Validation**: All origins undergo strict format and security validation

## Security Implications

### The CORS Vulnerability
Allowing unrestricted CORS access (`Access-Control-Allow-Origin: *`) enables any website to:
- Make API requests on behalf of authenticated users
- Extract sensitive data from responses
- Bypass same-origin policy protections
- Potentially execute CSRF attacks

### Secure Configuration Approach
Our implementation follows the principle of least privilege:
1. **No default access** - CORS must be explicitly configured
2. **Origin validation** - Each origin is validated for proper format
3. **Environment awareness** - Stricter rules in production
4. **Clear error messages** - Configuration issues are immediately visible

## Environment Variables

### LLM_ALLOWED_ORIGINS
**Required when**: `CORS_ENABLED=true`

Comma-separated list of allowed origins for CORS requests.

```bash
# Production example
LLM_ALLOWED_ORIGINS="https://app.company.com,https://admin.company.com"

# Development example
LLM_ALLOWED_ORIGINS="http://localhost:3000,http://localhost:8080"

# Multiple domains
LLM_ALLOWED_ORIGINS="https://app.example.com,https://mobile.example.com,https://partner.example.com"
```

### LLM_ENVIRONMENT
Controls environment-specific validation rules.

```bash
# Production (strict validation)
LLM_ENVIRONMENT=production

# Development (allows wildcards)
LLM_ENVIRONMENT=development

# Staging
LLM_ENVIRONMENT=staging
```

### CORS_ENABLED
Enables or disables CORS support.

```bash
# Enable CORS (requires LLM_ALLOWED_ORIGINS)
CORS_ENABLED=true

# Disable CORS (no cross-origin requests allowed)
CORS_ENABLED=false
```

## Configuration Examples

### Development Environment

```bash
# .env.development
LLM_ENVIRONMENT=development
CORS_ENABLED=true
LLM_ALLOWED_ORIGINS="http://localhost:3000,http://localhost:3001,http://192.168.1.100:3000"

# Or with wildcard (development only!)
LLM_ALLOWED_ORIGINS="*"
```

### Staging Environment

```bash
# .env.staging
LLM_ENVIRONMENT=staging
CORS_ENABLED=true
LLM_ALLOWED_ORIGINS="https://staging.company.com,https://preview.company.com"
```

### Production Environment

```bash
# .env.production
LLM_ENVIRONMENT=production
CORS_ENABLED=true
LLM_ALLOWED_ORIGINS="https://app.company.com,https://mobile.company.com"

# Additional security headers (recommended)
ENABLE_SECURITY_HEADERS=true
HSTS_MAX_AGE=31536000
```

### Kubernetes ConfigMap Example

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: llm-processor-config
  namespace: nephoran-system
data:
  LLM_ENVIRONMENT: "production"
  CORS_ENABLED: "true"
  LLM_ALLOWED_ORIGINS: "https://app.company.com,https://api.company.com"
```

### Kubernetes Secret Example (for sensitive origins)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: llm-processor-secrets
  namespace: nephoran-system
type: Opaque
stringData:
  LLM_ALLOWED_ORIGINS: "https://internal.company.com,https://admin.company.com"
```

## Validation Rules

### Origin Format Requirements

#### Valid Origins
- `https://app.example.com` - HTTPS origin
- `http://localhost:3000` - Local development
- `http://192.168.1.100:8080` - Local network development
- `*` - Wildcard (development only)

#### Invalid Origins
- `app.example.com` - Missing protocol
- `https://app.example.com/` - Trailing slash
- `https://app.example.com/path` - Includes path
- `*.example.com` - Partial wildcards not supported
- `https://app example.com` - Contains spaces

### Environment-Specific Rules

| Environment | Wildcard (*) | HTTP Origins | HTTPS Required |
|------------|--------------|--------------|----------------|
| development | ✅ Allowed | ✅ Allowed | ❌ No |
| staging | ❌ Blocked | ⚠️ Warning | ✅ Recommended |
| production | ❌ Blocked | ❌ Blocked | ✅ Required |

### Error Messages

```bash
# Missing configuration
Error: "LLM_ALLOWED_ORIGINS must be configured when CORS is enabled"

# Invalid format
Error: "invalid origin 'app.example.com': origin must start with http:// or https://"

# Wildcard in production
Error: "wildcard origin '*' is not allowed in production environments"

# Empty configuration
Error: "LLM_ALLOWED_ORIGINS: origins string cannot be empty"
```

## Migration Guide

### Migrating from Wildcard Configuration

If you're currently using `AllowedOrigins: "*"`, follow this migration path:

#### Phase 1: Audit Current Usage
```bash
# 1. Enable request logging to identify actual origins
LOG_LEVEL=debug
LOG_HTTP_HEADERS=true

# 2. Monitor logs for Origin headers
grep "Origin:" /var/log/llm-processor/*.log | sort | uniq

# 3. Build list of legitimate origins
```

#### Phase 2: Test with Explicit Origins
```bash
# 1. Set development environment
LLM_ENVIRONMENT=development

# 2. Configure discovered origins
LLM_ALLOWED_ORIGINS="https://app1.example.com,https://app2.example.com"

# 3. Test all client applications
```

#### Phase 3: Production Deployment
```bash
# 1. Update production configuration
LLM_ENVIRONMENT=production
CORS_ENABLED=true
LLM_ALLOWED_ORIGINS="https://app.example.com,https://mobile.example.com"

# 2. Deploy with monitoring
# 3. Watch for CORS errors in client applications
```

### Migration Script Example

```bash
#!/bin/bash
# migrate-cors-config.sh

# Backup current configuration
cp .env .env.backup

# Update environment variables
sed -i 's/ALLOWED_ORIGINS=/LLM_ALLOWED_ORIGINS=/g' .env
sed -i 's/LLM_ALLOWED_ORIGINS="\*"/LLM_ALLOWED_ORIGINS=""/g' .env

# Add environment setting
echo "LLM_ENVIRONMENT=development" >> .env

echo "Migration complete. Please update LLM_ALLOWED_ORIGINS with your actual origins."
```

## Troubleshooting

### Common Issues

#### 1. CORS Errors in Browser Console
```
Access to fetch at 'https://api.example.com' from origin 'https://app.example.com' 
has been blocked by CORS policy: No 'Access-Control-Allow-Origin' header is present
```

**Solution**: Add the origin to `LLM_ALLOWED_ORIGINS`:
```bash
LLM_ALLOWED_ORIGINS="https://app.example.com"
```

#### 2. Configuration Validation Errors
```
configuration validation failed: LLM_ALLOWED_ORIGINS must be configured when CORS is enabled
```

**Solution**: Either:
- Set `CORS_ENABLED=false` if CORS not needed
- Configure `LLM_ALLOWED_ORIGINS` with valid origins

#### 3. Wildcard Rejected in Production
```
wildcard origin '*' is not allowed in production environments
```

**Solution**: Replace wildcard with explicit origins:
```bash
# Instead of:
LLM_ALLOWED_ORIGINS="*"

# Use:
LLM_ALLOWED_ORIGINS="https://app.example.com,https://mobile.example.com"
```

### Debug Mode

Enable debug logging for CORS issues:

```bash
# Enable debug logging
LOG_LEVEL=debug
LOG_CORS_DETAILS=true

# Log all HTTP headers
LOG_HTTP_HEADERS=true
```

## Best Practices

### 1. Principle of Least Privilege
- Only allow origins that actually need access
- Use separate configurations per environment
- Regularly audit allowed origins

### 2. Use HTTPS in Production
```bash
# ✅ Good
LLM_ALLOWED_ORIGINS="https://app.example.com"

# ❌ Bad (in production)
LLM_ALLOWED_ORIGINS="http://app.example.com"
```

### 3. Environment Segregation
```bash
# Development
dev.env: LLM_ALLOWED_ORIGINS="http://localhost:3000,*"

# Staging  
staging.env: LLM_ALLOWED_ORIGINS="https://staging.example.com"

# Production
prod.env: LLM_ALLOWED_ORIGINS="https://app.example.com"
```

### 4. Regular Security Audits
```bash
# Audit script
#!/bin/bash
echo "=== CORS Security Audit ==="
echo "Environment: $LLM_ENVIRONMENT"
echo "CORS Enabled: $CORS_ENABLED"
echo "Allowed Origins: $LLM_ALLOWED_ORIGINS"

# Check for wildcards
if [[ "$LLM_ALLOWED_ORIGINS" == *"*"* ]]; then
  echo "⚠️  WARNING: Wildcard origin detected!"
fi

# Check for HTTP in production
if [[ "$LLM_ENVIRONMENT" == "production" ]] && [[ "$LLM_ALLOWED_ORIGINS" == *"http://"* ]]; then
  echo "⚠️  WARNING: HTTP origins in production!"
fi
```

### 5. Monitoring and Alerting

Configure alerts for:
- CORS validation failures
- Unexpected origin requests
- Configuration changes

```yaml
# Prometheus alert example
groups:
  - name: cors_security
    rules:
      - alert: CORSValidationFailure
        expr: rate(llm_processor_cors_validation_errors_total[5m]) > 0
        annotations:
          summary: "CORS validation failures detected"
          description: "{{ $value }} CORS validation errors in the last 5 minutes"
```

## Additional Resources

- [OWASP CORS Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Origin_Resource_Sharing_Cheat_Sheet.html)
- [MDN Web Docs: CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [Nephoran Security Best Practices](./SECURITY-HARDENING-SUMMARY.md)

## Support

For security concerns or questions:
- Create an issue in the Nephoran Intent Operator repository
- Contact the security team for sensitive issues
- Review the [security policy](../SECURITY.md)