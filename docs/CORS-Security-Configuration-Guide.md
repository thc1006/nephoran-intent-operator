# CORS Security Configuration Guide

## Overview

The Nephoran Intent Operator implements enterprise-grade Cross-Origin Resource Sharing (CORS) security controls to protect the LLM Processor service from unauthorized cross-origin requests. This guide provides comprehensive documentation for configuring, securing, and troubleshooting CORS settings across different deployment environments.

## Table of Contents

- [Quick Start](#quick-start)
- [Configuration Reference](#configuration-reference)
- [Environment-Specific Configuration](#environment-specific-configuration)
- [Security Best Practices](#security-best-practices)
- [Troubleshooting Guide](#troubleshooting-guide)
- [Common Scenarios](#common-scenarios)
- [Security Considerations](#security-considerations)

## Quick Start

### Production Configuration

```bash
# Kubernetes ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: llm-processor-config
  namespace: nephoran-system
data:
  LLM_ALLOWED_ORIGINS: "https://app.nephoran.com,https://dashboard.nephoran.com"
  ENVIRONMENT: "production"
```

### Development Configuration

```bash
# Local development (.env file)
export LLM_ALLOWED_ORIGINS="http://localhost:3000,http://localhost:8080"
export ENVIRONMENT="development"
```

## Configuration Reference

### Environment Variables

#### `LLM_ALLOWED_ORIGINS`

**Type:** String (comma-separated list)  
**Default:** Empty (uses secure defaults based on environment)  
**Description:** Specifies the allowed origins for CORS requests to the LLM Processor service.

**Format:**
```
LLM_ALLOWED_ORIGINS="origin1,origin2,origin3"
```

**Validation Rules:**
- Each origin must be a valid URL with scheme (http/https)
- No trailing slashes allowed
- Wildcard origins (`*`) are blocked in production
- Empty values trigger environment-based defaults
- Malformed origins are rejected with validation errors

**Examples:**
```bash
# Valid configurations
LLM_ALLOWED_ORIGINS="https://app.example.com,https://api.example.com"
LLM_ALLOWED_ORIGINS="https://dashboard.nephoran.io"

# Invalid configurations (will be rejected)
LLM_ALLOWED_ORIGINS="*"                              # Wildcard not allowed
LLM_ALLOWED_ORIGINS="app.example.com"                # Missing scheme
LLM_ALLOWED_ORIGINS="https://app.example.com/"       # Trailing slash
LLM_ALLOWED_ORIGINS="https://app.example.com:3000/"  # Trailing slash with port
```

#### `ENVIRONMENT`

**Type:** String  
**Default:** "development"  
**Values:** "development", "staging", "production"  
**Description:** Determines the security strictness and default CORS settings.

### Default Configurations by Environment

#### Production Defaults
When `LLM_ALLOWED_ORIGINS` is not set in production:
```go
[]string{
    "https://app.nephoran.com",
    "https://dashboard.nephoran.com",
    "https://api.nephoran.com",
}
```

#### Development Defaults
When `LLM_ALLOWED_ORIGINS` is not set in development:
```go
[]string{
    "http://localhost:3000",
    "http://localhost:8080",
    "http://127.0.0.1:3000",
    "http://127.0.0.1:8080",
}
```

## Environment-Specific Configuration

### Production Environment

**Kubernetes Deployment:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-processor
  namespace: nephoran-system
spec:
  template:
    spec:
      containers:
      - name: llm-processor
        image: nephoran/llm-processor:latest
        env:
        - name: LLM_ALLOWED_ORIGINS
          valueFrom:
            configMapKeyRef:
              name: llm-processor-config
              key: LLM_ALLOWED_ORIGINS
        - name: ENVIRONMENT
          value: "production"
```

**ConfigMap:**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: llm-processor-config
  namespace: nephoran-system
data:
  LLM_ALLOWED_ORIGINS: |
    https://app.nephoran.com,
    https://dashboard.nephoran.com,
    https://monitor.nephoran.com
```

**Security Features in Production:**
- ✅ Wildcard origins automatically blocked
- ✅ HTTPS enforcement for all origins
- ✅ Strict origin validation
- ✅ Comprehensive audit logging
- ✅ Real-time security monitoring

### Staging Environment

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: llm-processor-config
  namespace: nephoran-staging
data:
  LLM_ALLOWED_ORIGINS: |
    https://staging-app.nephoran.com,
    https://staging-dashboard.nephoran.com,
    https://test.nephoran.com
  ENVIRONMENT: "staging"
```

### Development Environment

**Docker Compose:**

```yaml
version: '3.8'
services:
  llm-processor:
    image: nephoran/llm-processor:latest
    environment:
      - LLM_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
      - ENVIRONMENT=development
    ports:
      - "8080:8080"
```

**Local Development:**

```bash
# .env.development
export LLM_ALLOWED_ORIGINS="http://localhost:3000,http://localhost:8080,http://localhost:5173"
export ENVIRONMENT="development"

# Run locally
source .env.development
go run cmd/llm-processor/main.go
```

## Security Best Practices

### 1. Principle of Least Privilege

**✅ DO:**
```bash
# Specify only required origins
LLM_ALLOWED_ORIGINS="https://app.mycompany.com,https://api.mycompany.com"
```

**❌ DON'T:**
```bash
# Never use wildcards or overly permissive settings
LLM_ALLOWED_ORIGINS="*"  # BLOCKED in production
LLM_ALLOWED_ORIGINS="https://*.mycompany.com"  # Subdomain wildcards not supported
```

### 2. Environment Separation

**Production:**
```bash
# Use HTTPS only
LLM_ALLOWED_ORIGINS="https://app.production.com"
```

**Development:**
```bash
# HTTP allowed for local development only
LLM_ALLOWED_ORIGINS="http://localhost:3000"
```

### 3. Regular Auditing

**Audit Script:**
```bash
#!/bin/bash
# audit-cors-config.sh

echo "=== CORS Configuration Audit ==="
kubectl get configmap llm-processor-config -n nephoran-system -o yaml | \
  grep LLM_ALLOWED_ORIGINS | \
  awk '{print $2}' | \
  tr ',' '\n' | \
  while read origin; do
    echo "Checking origin: $origin"
    # Verify HTTPS in production
    if [[ "$ENVIRONMENT" == "production" ]] && [[ ! "$origin" =~ ^https:// ]]; then
      echo "⚠️  WARNING: Non-HTTPS origin in production: $origin"
    fi
    # Check for wildcards
    if [[ "$origin" == "*" ]]; then
      echo "❌ ERROR: Wildcard origin detected!"
    fi
  done
```

### 4. Monitoring and Alerting

**Prometheus Alert Rule:**
```yaml
groups:
- name: cors-security
  rules:
  - alert: CORSViolationAttempt
    expr: rate(llm_processor_cors_violations_total[5m]) > 10
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "High rate of CORS violations detected"
      description: "{{ $value }} CORS violations per second"
```

### 5. Secure Defaults

The system implements secure defaults to prevent misconfiguration:

- **Empty configuration → Secure defaults applied**
- **Invalid origins → Startup failure with clear error**
- **Wildcard in production → Automatic rejection**
- **Malformed URLs → Validation error**

## Troubleshooting Guide

### Common Issues and Solutions

#### Issue 1: CORS Preflight Failures

**Symptoms:**
```
Access to fetch at 'http://llm-processor:8080/api/process' from origin 
'https://app.example.com' has been blocked by CORS policy: 
Response to preflight request doesn't pass access control check
```

**Solution:**
1. Verify origin is in allowed list:
```bash
kubectl get configmap llm-processor-config -n nephoran-system -o yaml
```

2. Check for exact match (no trailing slashes):
```bash
# Correct
LLM_ALLOWED_ORIGINS="https://app.example.com"

# Incorrect (trailing slash)
LLM_ALLOWED_ORIGINS="https://app.example.com/"
```

3. Restart the service after configuration change:
```bash
kubectl rollout restart deployment/llm-processor -n nephoran-system
```

#### Issue 2: Validation Errors on Startup

**Error Message:**
```
CORS configuration error: invalid origin format: app.example.com 
(must include scheme http:// or https://)
```

**Solution:**
Add the proper scheme to all origins:
```bash
# Before (incorrect)
LLM_ALLOWED_ORIGINS="app.example.com,api.example.com"

# After (correct)
LLM_ALLOWED_ORIGINS="https://app.example.com,https://api.example.com"
```

#### Issue 3: Wildcard Rejection in Production

**Error Message:**
```
CORS configuration error: wildcard origins not allowed in production environment
```

**Solution:**
Replace wildcard with specific origins:
```bash
# Before (rejected)
LLM_ALLOWED_ORIGINS="*"

# After (accepted)
LLM_ALLOWED_ORIGINS="https://app.nephoran.com,https://dashboard.nephoran.com"
```

#### Issue 4: Mixed Content Errors

**Browser Console Error:**
```
Mixed Content: The page at 'https://app.example.com' was loaded over HTTPS, 
but requested an insecure resource 'http://llm-processor:8080/'
```

**Solution:**
Ensure HTTPS is used for all production origins:
```bash
# Production must use HTTPS
LLM_ALLOWED_ORIGINS="https://app.example.com"
```

### Debugging CORS Configuration

#### 1. Check Current Configuration

```bash
# View current CORS settings
kubectl exec deployment/llm-processor -n nephoran-system -- env | grep LLM_ALLOWED_ORIGINS

# Check service logs for CORS initialization
kubectl logs deployment/llm-processor -n nephoran-system | grep -i cors
```

#### 2. Test CORS Headers

```bash
# Test preflight request
curl -X OPTIONS \
  -H "Origin: https://app.example.com" \
  -H "Access-Control-Request-Method: POST" \
  -H "Access-Control-Request-Headers: Content-Type" \
  -v http://llm-processor:8080/api/process

# Expected response headers:
# Access-Control-Allow-Origin: https://app.example.com
# Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
# Access-Control-Allow-Headers: Content-Type, Authorization
```

#### 3. Validate Origin Format

```bash
#!/bin/bash
# validate-origins.sh

ORIGINS="https://app.example.com,https://api.example.com"

echo "$ORIGINS" | tr ',' '\n' | while read origin; do
  if [[ ! "$origin" =~ ^https?://[^/]+$ ]]; then
    echo "❌ Invalid origin: $origin"
  else
    echo "✅ Valid origin: $origin"
  fi
done
```

## Common Scenarios

### Scenario 1: Multi-Domain Application

**Requirements:** Application served from multiple domains

```yaml
# Production configuration
LLM_ALLOWED_ORIGINS: |
  https://app.nephoran.com,
  https://app.nephoran.io,
  https://dashboard.nephoran.com,
  https://admin.nephoran.com
```

### Scenario 2: Development with Multiple Ports

**Requirements:** Local development with various frontend frameworks

```bash
# Development configuration
export LLM_ALLOWED_ORIGINS="http://localhost:3000,http://localhost:4200,http://localhost:5173,http://localhost:8080"
```

### Scenario 3: CI/CD Pipeline Testing

**Requirements:** Automated testing in CI environment

```yaml
# CI/CD configuration
env:
  - name: LLM_ALLOWED_ORIGINS
    value: "http://test-frontend:3000,http://test-api:8080"
  - name: ENVIRONMENT
    value: "development"  # Allow HTTP in CI
```

### Scenario 4: Migration from Permissive to Restrictive

**Phase 1: Audit current usage**
```bash
# Log all origins making requests
kubectl logs deployment/llm-processor -n nephoran-system | \
  grep "CORS request from origin" | \
  awk '{print $NF}' | sort | uniq
```

**Phase 2: Update configuration**
```bash
# Add discovered legitimate origins
LLM_ALLOWED_ORIGINS="https://app1.example.com,https://app2.example.com"
```

**Phase 3: Monitor for violations**
```bash
# Watch for rejected origins
kubectl logs deployment/llm-processor -n nephoran-system -f | grep "CORS violation"
```

## Security Considerations

### 1. Attack Vectors and Mitigations

#### Cross-Site Request Forgery (CSRF)
**Mitigation:** Strict origin validation prevents unauthorized cross-origin requests

#### Data Exfiltration
**Mitigation:** Limiting allowed origins reduces attack surface

#### Credential Theft
**Mitigation:** Credentials not included in CORS requests by default

### 2. Security Checklist

- [ ] **Production uses HTTPS exclusively**
- [ ] **No wildcard origins in production**
- [ ] **Minimal set of allowed origins**
- [ ] **Regular audit of CORS configuration**
- [ ] **Monitoring for CORS violations**
- [ ] **Automated validation in CI/CD**
- [ ] **Documentation of all allowed origins**
- [ ] **Regular security reviews**

### 3. Compliance Considerations

**GDPR Compliance:**
- Limit data exposure to authorized origins only
- Document all cross-origin data flows
- Implement audit logging for compliance tracking

**SOC 2 Requirements:**
- Enforce principle of least privilege
- Maintain configuration change logs
- Regular security assessments

### 4. Security Monitoring

**Key Metrics to Monitor:**
- CORS violation attempts per minute
- Unique rejected origins
- Successful cross-origin requests
- Configuration changes

**Alerting Thresholds:**
- \> 10 CORS violations/minute → Warning
- \> 100 CORS violations/minute → Critical
- New unknown origin detected → Information
- Configuration change → Audit log

## Advanced Configuration

### Custom CORS Middleware Extensions

For advanced use cases, the CORS middleware can be extended:

```go
// Custom validation example
func validateCustomOrigin(origin string) bool {
    // Additional business logic validation
    if strings.Contains(origin, "internal") {
        return isInternalNetwork(origin)
    }
    return true
}
```

### Dynamic Origin Loading

For environments requiring dynamic origin updates:

```bash
#!/bin/bash
# dynamic-cors-update.sh

# Fetch allowed origins from external source
ORIGINS=$(curl -s https://config.example.com/cors-origins)

# Update ConfigMap
kubectl patch configmap llm-processor-config \
  -n nephoran-system \
  --type merge \
  -p "{\"data\":{\"LLM_ALLOWED_ORIGINS\":\"$ORIGINS\"}}"

# Trigger rolling update
kubectl rollout restart deployment/llm-processor -n nephoran-system
```

## Testing and Validation

### Unit Testing CORS Configuration

```go
func TestCORSConfiguration(t *testing.T) {
    tests := []struct {
        name        string
        origins     string
        environment string
        wantError   bool
    }{
        {
            name:        "Valid production origins",
            origins:     "https://app.example.com,https://api.example.com",
            environment: "production",
            wantError:   false,
        },
        {
            name:        "Wildcard in production",
            origins:     "*",
            environment: "production",
            wantError:   true,
        },
        {
            name:        "HTTP in production",
            origins:     "http://app.example.com",
            environment: "production",
            wantError:   false, // HTTP technically allowed but logged as warning
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateCORSConfig(tt.origins, tt.environment)
            if (err != nil) != tt.wantError {
                t.Errorf("validateCORSConfig() error = %v, wantError %v", err, tt.wantError)
            }
        })
    }
}
```

### Integration Testing

```bash
#!/bin/bash
# test-cors-integration.sh

echo "=== CORS Integration Test ==="

# Test allowed origin
echo "Testing allowed origin..."
RESPONSE=$(curl -s -X OPTIONS \
  -H "Origin: https://app.nephoran.com" \
  -H "Access-Control-Request-Method: POST" \
  -w "\n%{http_code}" \
  http://llm-processor:8080/api/process | tail -1)

if [ "$RESPONSE" = "204" ]; then
  echo "✅ Allowed origin test passed"
else
  echo "❌ Allowed origin test failed (HTTP $RESPONSE)"
fi

# Test blocked origin
echo "Testing blocked origin..."
RESPONSE=$(curl -s -X OPTIONS \
  -H "Origin: https://malicious.com" \
  -H "Access-Control-Request-Method: POST" \
  -w "\n%{http_code}" \
  http://llm-processor:8080/api/process | tail -1)

if [ "$RESPONSE" = "403" ]; then
  echo "✅ Blocked origin test passed"
else
  echo "❌ Blocked origin test failed (HTTP $RESPONSE)"
fi
```

## Migration Guide

### Migrating from Permissive CORS

**Step 1: Audit Current Usage**
```bash
# Enable detailed logging
kubectl set env deployment/llm-processor LOG_LEVEL=DEBUG -n nephoran-system

# Collect origins for 24 hours
kubectl logs deployment/llm-processor -n nephoran-system --since=24h | \
  grep "Origin:" | awk '{print $2}' | sort | uniq > origins.txt
```

**Step 2: Validate and Filter Origins**
```bash
# Review and validate collected origins
cat origins.txt | while read origin; do
  echo "Validating: $origin"
  # Add validation logic
done > validated-origins.txt
```

**Step 3: Update Configuration**
```bash
# Create new configuration
VALIDATED_ORIGINS=$(cat validated-origins.txt | tr '\n' ',')
kubectl patch configmap llm-processor-config \
  -n nephoran-system \
  --type merge \
  -p "{\"data\":{\"LLM_ALLOWED_ORIGINS\":\"$VALIDATED_ORIGINS\"}}"
```

**Step 4: Monitor and Adjust**
```bash
# Monitor for rejected legitimate origins
kubectl logs deployment/llm-processor -n nephoran-system -f | \
  grep "CORS violation" | \
  tee cors-violations.log
```

## Support and Resources

### Getting Help

If you encounter issues with CORS configuration:

1. **Check the documentation:** Review this guide and the security best practices
2. **Review logs:** `kubectl logs deployment/llm-processor -n nephoran-system`
3. **Test configuration:** Use the provided debugging scripts
4. **Community support:** Open an issue on GitHub with details

### Additional Resources

- [MDN CORS Documentation](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
- [OWASP CORS Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Origin_Resource_Sharing_Cheat_Sheet.html)
- [Kubernetes Network Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
- [Nephoran Security Documentation](./Security-Hardening-Guide.md)

### Version History

- **v1.0.0** (December 2024): Initial CORS security implementation
- **v1.1.0** (December 2024): Enhanced validation and production hardening
- **v1.2.0** (Current): Multiple origin support with comprehensive validation

---

*Last Updated: December 2024*  
*Nephoran Intent Operator - Enterprise CORS Security Configuration*