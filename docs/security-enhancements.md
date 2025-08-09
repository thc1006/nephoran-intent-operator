# LLM Processor HTTP Service Security Enhancements

## Overview
This document describes the security improvements implemented in the LLM processor HTTP service to provide safe defaults and defense-in-depth protection.

## Security Features Implemented

### 1. Request Body Size Limiting
- **Environment Variable**: `HTTP_MAX_BODY`
- **Default**: 1048576 bytes (1MB)
- **Behavior**: Returns HTTP 413 (Request Entity Too Large) for oversized requests
- **Implementation**: Applied as the first middleware in the chain to reject oversized requests early

### 2. Security Headers
All responses include the following security headers:

#### Always Applied:
- `X-Content-Type-Options: nosniff` - Prevents MIME type sniffing
- `X-Frame-Options: DENY` - Prevents clickjacking attacks
- `Content-Security-Policy: default-src 'none'` - Strict CSP for API service

#### Conditionally Applied:
- `Strict-Transport-Security` - Only when TLS is enabled (HSTS)
  - Default max-age: 31536000 seconds (1 year)
  - Prevents protocol downgrade attacks

### 3. Metrics Endpoint Control
- **Environment Variable**: `METRICS_ENABLED`
- **Default**: `false` (disabled by default for security)
- **Behavior**: When disabled, `/metrics` endpoint returns 404

### 4. IP-based Metrics Access Control
- **Environment Variable**: `METRICS_ALLOWED_IPS`
- **Format**: Comma-separated list of IP addresses
- **Default**: Empty (if metrics enabled, allows all)
- **Behavior**: Returns HTTP 403 (Forbidden) for unauthorized IPs
- **IP Detection**: Checks X-Forwarded-For, X-Real-IP, and RemoteAddr

## Configuration Examples

### Basic Secure Configuration
```bash
# Default secure settings (metrics disabled)
export HTTP_MAX_BODY=1048576
export METRICS_ENABLED=false
```

### Production Configuration with Monitoring
```bash
# Enable metrics with IP restrictions
export HTTP_MAX_BODY=5242880  # 5MB limit
export METRICS_ENABLED=true
export METRICS_ALLOWED_IPS="10.0.0.1,10.0.0.2,192.168.1.100"
export TLS_ENABLED=true
export TLS_CERT_PATH=/path/to/cert.pem
export TLS_KEY_PATH=/path/to/key.pem
```

### Development Configuration
```bash
# More permissive for development
export HTTP_MAX_BODY=10485760  # 10MB
export METRICS_ENABLED=true
export METRICS_ALLOWED_IPS=""  # Allow all IPs
export TLS_ENABLED=false
```

## Security Best Practices

### 1. Always Enable TLS in Production
When TLS is enabled, HSTS headers are automatically added to enforce secure connections.

### 2. Restrict Metrics Access
- Keep `METRICS_ENABLED=false` unless monitoring is required
- When enabled, always use `METRICS_ALLOWED_IPS` to restrict access
- Consider using monitoring IPs from your internal network only

### 3. Set Appropriate Request Size Limits
- Default 1MB is suitable for most JSON API requests
- Increase only if necessary for your use case
- Monitor for 413 errors to tune appropriately

### 4. Security Headers
The implemented headers provide defense against common web vulnerabilities:
- **X-Content-Type-Options**: Prevents browsers from MIME-sniffing responses
- **X-Frame-Options**: Prevents embedding in iframes (clickjacking protection)
- **CSP**: Restricts resource loading to prevent XSS and data injection

## Testing

### Unit Tests
Comprehensive tests are provided in `cmd/llm-processor/security_test.go`:
- Request body size limiting with 413 responses
- Security header verification
- Metrics endpoint control
- IP-based access restrictions
- Integration testing of complete security stack

### Running Tests
```bash
go test ./cmd/llm-processor -run TestSecurity
```

### Manual Testing

#### Test Request Size Limiting
```bash
# Should return 413
curl -X POST http://localhost:8080/process \
  -H "Content-Type: application/json" \
  -d "$(python -c 'print("x" * 2000000)')"
```

#### Test Security Headers
```bash
curl -I http://localhost:8080/health
# Check for X-Content-Type-Options, X-Frame-Options, etc.
```

#### Test Metrics Access Control
```bash
# With METRICS_ENABLED=false
curl http://localhost:8080/metrics  # Should return 404

# With METRICS_ENABLED=true and METRICS_ALLOWED_IPS=127.0.0.1
curl http://localhost:8080/metrics  # Should work from localhost
curl http://remote-host:8080/metrics  # Should return 403
```

## Migration Guide

### For Existing Deployments
1. The system maintains backward compatibility
2. If `HTTP_MAX_BODY` is not set, falls back to `MAX_REQUEST_SIZE`
3. If neither is set, uses default 1MB limit
4. Metrics are disabled by default (secure by default)

### Environment Variable Changes
```bash
# Old configuration
MAX_REQUEST_SIZE=5242880

# New configuration (preferred)
HTTP_MAX_BODY=5242880
METRICS_ENABLED=true
METRICS_ALLOWED_IPS="10.0.0.1,10.0.0.2"
```

## Security Considerations

### Defense in Depth
The implementation follows defense-in-depth principles:
1. **Input validation**: Request size limiting
2. **Output encoding**: Security headers
3. **Access control**: IP-based restrictions
4. **Secure defaults**: Metrics disabled by default

### Audit Logging
All security events are logged:
- Oversized request rejections
- Metrics access denials
- IP restriction violations

### Future Enhancements
Consider implementing:
- Rate limiting per IP for metrics endpoint
- JWT-based authentication for metrics
- Prometheus bearer token authentication
- Custom security headers via environment variables

## Compliance
These security enhancements help meet common compliance requirements:
- **OWASP Top 10**: Addresses injection, XSS, and security misconfiguration
- **PCI DSS**: Implements secure defaults and access controls
- **NIST**: Follows least privilege and defense-in-depth principles