# OAuth2 Security Implementation Guide

## Executive Summary

The Nephoran Intent Operator implements enterprise-grade OAuth2 authentication with comprehensive security controls, multi-provider support, and defense-in-depth strategies. This guide consolidates all OAuth2 security documentation including configuration, analysis, and best practices.

## Table of Contents

- [Security Architecture](#security-architecture)
- [Supported Identity Providers](#supported-identity-providers)
- [Configuration Guide](#configuration-guide)
- [Security Implementation](#security-implementation)
- [RBAC and Permissions](#rbac-and-permissions)
- [API Usage](#api-usage)
- [Security Validation](#security-validation)
- [Production Deployment](#production-deployment)
- [Security Monitoring](#security-monitoring)
- [Troubleshooting](#troubleshooting)
- [Compliance and Standards](#compliance-and-standards)

## Security Architecture

### Multi-Layer Security Model

The OAuth2 implementation employs a comprehensive defense-in-depth approach:

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Security Perimeter                           │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    TLS 1.3 Encryption Layer                  │  │
│  │  • Certificate Validation                                    │  │
│  │  • Mutual TLS Support (optional)                            │  │
│  │  • HTTPS-only in Production                                 │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                ▼                                    │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │              OAuth2 Authentication Layer                     │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │  │
│  │  │  Azure AD   │  │    Okta     │  │  Keycloak   │         │  │
│  │  │  Provider   │  │  Provider   │  │  Provider   │         │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘         │  │
│  │  ┌─────────────┐  ┌─────────────┐                          │  │
│  │  │   Google    │  │   Custom    │                          │  │
│  │  │  Provider   │  │  Provider   │                          │  │
│  │  └─────────────┘  └─────────────┘                          │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                ▼                                    │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                    JWT Token Validation                      │  │
│  │  • Signature Verification                                    │  │
│  │  • Expiration Check                                          │  │
│  │  • Claims Validation                                         │  │
│  │  • Role Mapping                                              │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                ▼                                    │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                Role-Based Access Control (RBAC)              │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │  │
│  │  │    Admin    │  │  Operator   │  │   Viewer    │         │  │
│  │  │  Full Mgmt  │  │  Intent Ops │  │  Read-Only  │         │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘         │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                ▼                                    │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                  Request Security Controls                   │  │
│  │  • Request Size Limiting (1MB default)                       │  │
│  │  • Panic Recovery                                            │  │
│  │  • Memory Protection                                         │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Supported Identity Providers

### Provider Specifications and Requirements

| Provider | Client Secret Requirements | Key Features | Use Case |
|----------|---------------------------|--------------|----------|
| **Azure AD** | Minimum 16 characters | Group-based role mapping, Conditional access | Enterprise Microsoft 365/Azure environments |
| **Okta** | Minimum 40 characters | Group claims, Custom scopes, MFA | Organizations with Okta identity management |
| **Keycloak** | Minimum 32 characters | Custom realms, Self-hosted | On-premises or self-hosted deployments |
| **Google** | Minimum 24 characters | Google Groups integration | Google Workspace organizations |
| **Custom** | Minimum 16 characters | Flexible configuration | Custom OAuth2 providers |

### Provider-Specific Configuration

#### Azure Active Directory

```yaml
# Configuration
data:
  AZURE_ENABLED: "true"
  AZURE_CLIENT_ID: "your-application-id"
  AZURE_TENANT_ID: "your-tenant-id"
  AZURE_SCOPES: "openid,profile,email,User.Read"

# Secret (separate file)
apiVersion: v1
kind: Secret
metadata:
  name: oauth2-secrets
stringData:
  azure-client-secret: "your-azure-client-secret"  # Min 16 chars
```

#### Okta

```yaml
# Configuration
data:
  OKTA_ENABLED: "true"
  OKTA_CLIENT_ID: "your-okta-client-id"
  OKTA_DOMAIN: "your-company.okta.com"
  OKTA_SCOPES: "openid,profile,email,groups"

# Secret (separate file)
stringData:
  okta-client-secret: "your-okta-client-secret"  # Min 40 chars
```

#### Keycloak

```yaml
# Configuration
data:
  KEYCLOAK_ENABLED: "true"
  KEYCLOAK_CLIENT_ID: "nephoran-intent-operator"
  KEYCLOAK_BASE_URL: "https://keycloak.company.com"
  KEYCLOAK_REALM: "telecom"

# Secret (separate file)
stringData:
  keycloak-client-secret: "your-keycloak-secret"  # Min 32 chars
```

## Configuration Guide

### Step 1: Enable Authentication

Edit `deployments/kustomize/base/llm-processor/oauth2-config.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth2-environment
  namespace: nephoran-system
data:
  AUTH_ENABLED: "true"        # Enable OAuth2 authentication
  REQUIRE_AUTH: "true"        # Require authentication for protected endpoints
  RBAC_ENABLED: "true"        # Enable role-based access control
  ENVIRONMENT: "production"    # Enforces strict security
```

### Step 2: Configure Secrets

```bash
# Generate secure JWT secret key (minimum 32 characters)
JWT_SECRET=$(openssl rand -base64 32)

# Create secrets with provider credentials
kubectl create secret generic oauth2-secrets \
  --from-literal=jwt-secret-key="$JWT_SECRET" \
  --from-literal=azure-client-secret="your-azure-secret" \
  --from-literal=okta-client-secret="your-okta-secret" \
  -n nephoran-system
```

### Step 3: Deploy Services

```bash
# Apply OAuth2 configuration
kubectl apply -f deployments/kustomize/base/llm-processor/oauth2-config.yaml

# Deploy with authentication enabled
kubectl apply -k deployments/kustomize/base/llm-processor/
```

## Security Implementation

### Secret Management Security

#### Multi-Source Secret Loading

The system implements a secure secret loading hierarchy:

1. **Primary Source**: File-based secrets at `/secrets/oauth2/{provider}-client-secret`
2. **Fallback Source**: Environment variable `{PROVIDER}_CLIENT_SECRET`
3. **Validation Pipeline**: Format, strength, and pattern validation
4. **Audit Logging**: All secret access attempts are logged

#### Secret Validation Pipeline

```go
// Validation stages for each secret:
1. Empty Check       → Reject empty or whitespace-only secrets
2. Length Validation → Provider-specific minimum lengths
3. Placeholder Detection → Reject common placeholders:
   - "changeme", "your-secret", "placeholder"
   - "example", "client-secret", "secret"
4. Pattern Analysis  → Detect weak patterns and repetitions
5. Audit Logging    → Log validation results
```

### JWT Security

#### JWT Secret Requirements

- **Minimum Length**: 32 characters enforced
- **Weak Secret Detection**: Comprehensive blacklist including:
  - Common passwords: "password", "secret", "12345678"
  - Repetitive patterns: "aaaa...", "1111..."
  - Default values: "jwt-secret", "secret-key"
- **Secure Storage**: File-based with 0600 permissions or environment variable

#### Token Configuration

```yaml
token_configuration:
  access_token_ttl: 86400    # 24 hours
  refresh_token_ttl: 604800  # 7 days
  cookie_settings:
    httpOnly: true
    secure: true            # HTTPS only
    sameSite: strict
    path: "/"
```

### Security Headers

All responses include comprehensive security headers:

```yaml
security_headers:
  X-Frame-Options: DENY
  X-Content-Type-Options: nosniff
  X-XSS-Protection: 1; mode=block
  Strict-Transport-Security: max-age=31536000; includeSubDomains
  Content-Security-Policy: default-src 'self'
  Referrer-Policy: strict-origin-when-cross-origin
```

### Error Handling Security

#### Secure Error Messages

- **Public Errors**: Generic messages without sensitive information
- **Internal Logging**: Detailed errors for debugging (not exposed to users)
- **Audit Trail**: All errors logged with context for security monitoring

Example:
```go
// Public error (safe)
"OAuth2 client secret not configured for provider: azure-ad"

// Internal log (detailed)
"Failed to load azure-ad secret from /secrets/oauth2/azure-client-secret: permission denied"
```

## RBAC and Permissions

### Role Hierarchy

```yaml
admin:
  description: Full system management
  permissions:
    - intent:create, intent:read, intent:update, intent:delete
    - e2nodes:manage
    - metrics:view
    - system:manage
    - users:manage
    - logs:view
    - secrets:manage

operator:
  description: Network operations management
  permissions:
    - intent:create, intent:read, intent:update, intent:delete
    - e2nodes:manage
    - metrics:view
    - logs:view

network-operator:
  description: Limited network operations
  permissions:
    - intent:create, intent:read, intent:update
    - e2nodes:manage
    - metrics:view

viewer:
  description: Read-only access
  permissions:
    - intent:read
    - metrics:view
```

### Permission Matrix

| Role | Intent Create | Intent Read | Intent Update | Intent Delete | E2 Nodes | System Admin |
|------|--------------|-------------|---------------|---------------|----------|--------------|
| admin | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| operator | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| network-operator | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| viewer | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ |

### Role Mapping Configuration

```yaml
rbac:
  role_mapping:
    # Map provider roles to internal roles
    "Global Administrator": ["admin"]
    "Network Administrator": ["operator"]
    "Telecom Operator": ["network-operator"]
    "Read Only": ["viewer"]
  
  group_mapping:
    # Map provider groups to internal roles
    "Nephoran-Admins": ["admin"]
    "Telecom-Operators": ["operator"]
    "Network-Engineers": ["network-operator"]
    "Viewers": ["viewer"]
```

## API Usage

### OAuth2 Login Flow

#### 1. Initiate Login

```bash
curl "https://llm-processor.nephoran.com/auth/login/azure-ad"
# Redirects to identity provider
```

#### 2. Handle Callback

After authentication, the user is redirected to:
```
https://llm-processor.nephoran.com/auth/callback/azure-ad?code=...&state=...
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 86400,
  "refresh_token": "refresh_token_here",
  "user_info": {
    "subject": "user@company.com",
    "email": "user@company.com",
    "name": "John Doe",
    "groups": ["Telecom-Operators"],
    "roles": ["operator"],
    "provider": "azure-ad"
  }
}
```

### Making Authenticated Requests

#### Process Intent (Requires Operator Role)

```bash
curl -X POST https://llm-processor.nephoran.com/process \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "Deploy AMF with 3 replicas in production",
    "metadata": {
      "priority": "high",
      "environment": "production"
    }
  }'
```

#### Get System Status (Requires Admin Role)

```bash
curl -X GET https://llm-processor.nephoran.com/admin/status \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Token Management

#### Token Refresh

```bash
curl -X POST https://llm-processor.nephoran.com/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "your-refresh-token",
    "provider": "azure-ad"
  }'
```

#### Get User Information

```bash
curl -X GET https://llm-processor.nephoran.com/auth/userinfo \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## Security Validation

### Development vs Production Mode

The system automatically detects the environment and adjusts security settings:

```go
// Environment detection checks these variables:
- GO_ENV
- NODE_ENV
- ENVIRONMENT
- ENV
- APP_ENV

// Production mode enforces:
- Authentication required for all protected endpoints
- TLS/HTTPS required
- Secure headers enforced
- No wildcard CORS origins
- Strict secret validation
```

### Validation Checkpoints

#### Configuration Load Time
1. JWT secret strength validation (32+ characters)
2. OAuth2 client secret validation (provider-specific lengths)
3. TLS certificate verification
4. CORS origin validation
5. Request size limit validation

#### Runtime Security Checks
1. Token expiration validation
2. Role-based access control
3. Request size enforcement
4. Circuit breaker activation
5. Rate limiting enforcement

### Security Testing Scenarios

The implementation includes comprehensive tests for:

1. **Empty and Invalid Secrets**
   - Empty provider names
   - Missing secrets
   - Whitespace-only secrets

2. **Weak Secret Detection**
   - Common passwords
   - Placeholder values
   - Repetitive patterns

3. **Provider-Specific Validation**
   - Minimum length requirements
   - Format validation
   - Required field validation

4. **Error Handling**
   - Proper error propagation
   - Secure error messages
   - Graceful degradation

## Production Deployment

### Pre-Production Security Checklist

- [ ] **OAuth2 Configuration**
  - [ ] All OAuth2 client secrets meet minimum length requirements
  - [ ] No placeholder secrets in configuration
  - [ ] Provider configurations validated

- [ ] **JWT Security**
  - [ ] JWT secret is unique and strong (32+ characters)
  - [ ] Token TTL values appropriate for use case
  - [ ] Refresh token strategy configured

- [ ] **TLS/HTTPS**
  - [ ] TLS certificates valid and not self-signed
  - [ ] HTTPS enforced for all authentication endpoints
  - [ ] Certificate expiration monitoring configured

- [ ] **Access Control**
  - [ ] Role mappings configured correctly
  - [ ] Group mappings tested with provider
  - [ ] Permission matrix reviewed

- [ ] **Security Headers**
  - [ ] All security headers configured
  - [ ] CSP policy appropriate for application
  - [ ] HSTS enabled with appropriate max-age

- [ ] **Monitoring**
  - [ ] Audit logging configured and tested
  - [ ] Security metrics dashboard created
  - [ ] Alert thresholds configured

- [ ] **Network Security**
  - [ ] CORS origins explicitly configured (no wildcards)
  - [ ] Request size limits appropriate
  - [ ] Rate limiting configured
  - [ ] Circuit breakers configured

### Production Configuration Example

```yaml
# Production OAuth2 Configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth2-environment
  namespace: nephoran-system
data:
  # Core Settings
  AUTH_ENABLED: "true"
  REQUIRE_AUTH: "true"
  ENVIRONMENT: "production"
  
  # TLS Configuration
  TLS_ENABLED: "true"
  TLS_CERT_PATH: "/certs/tls.crt"
  TLS_KEY_PATH: "/certs/tls.key"
  
  # Security Settings
  MAX_REQUEST_SIZE: "1048576"  # 1MB
  CORS_ENABLED: "true"
  LLM_ALLOWED_ORIGINS: "https://production.nephoran.com"
  
  # Rate Limiting
  RATE_LIMIT_ENABLED: "true"
  RATE_LIMIT_REQUESTS_PER_MINUTE: "60"
  
  # Token Configuration
  ACCESS_TOKEN_TTL: "86400"    # 24 hours
  REFRESH_TOKEN_TTL: "604800"  # 7 days
```

## Security Monitoring

### Key Security Metrics

Monitor these metrics for security health:

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| Authentication Success Rate | Successful auth / total attempts | < 95% (warning) |
| Failed Authentication Rate | Failed attempts per minute | > 10/min (warning), > 50/min (critical) |
| Secret Validation Failures | Invalid secret configurations | > 0 (critical) |
| Token Validation Time | Average JWT validation latency | > 10ms (warning) |
| Security Header Compliance | Percentage of responses with headers | < 100% (warning) |
| Audit Log Coverage | Events logged / total events | < 100% (critical) |
| Circuit Breaker Trips | Service protection activations | > 5/hour (warning) |

### Audit Logging

All security events are logged with structured data:

```json
{
  "timestamp": "2024-11-20T10:30:45Z",
  "level": "info",
  "event": "auth.login.success",
  "provider": "azure-ad",
  "user": "user@company.com",
  "roles": ["operator"],
  "ip": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "request_id": "uuid-here"
}
```

### Security Event Types

- `auth.login.attempt` - Login initiated
- `auth.login.success` - Successful authentication
- `auth.login.failure` - Failed authentication
- `auth.token.validated` - JWT token validated
- `auth.token.expired` - Expired token used
- `auth.token.invalid` - Invalid token detected
- `secret.access.success` - Secret loaded successfully
- `secret.access.failure` - Secret loading failed
- `secret.validation.failure` - Secret validation failed
- `rbac.access.denied` - Authorization denied
- `rbac.access.granted` - Authorization granted

## Troubleshooting

### Common Issues and Solutions

#### Authentication Failed

**Error**: `Authentication failed`

**Solutions**:
1. Verify OAuth2 provider configuration
2. Check client ID and secret validity
3. Validate redirect URLs match provider settings
4. Review provider-specific requirements

```bash
# Debug commands
kubectl get configmap oauth2-environment -n nephoran-system -o yaml
kubectl logs -l app=llm-processor -n nephoran-system | grep "auth"
```

#### Invalid JWT Token

**Error**: `Invalid token: token is malformed`

**Solutions**:
1. Verify JWT secret key configuration
2. Check token hasn't expired
3. Validate token format and signing method

```bash
# Verify JWT secret exists
kubectl get secret oauth2-secrets -n nephoran-system -o jsonpath='{.data.jwt-secret-key}' | base64 -d | wc -c
# Should be >= 32 characters
```

#### Access Denied

**Error**: `Required roles: [operator]`

**Solutions**:
1. Verify user role mapping in configuration
2. Check group membership in identity provider
3. Review RBAC configuration

```bash
# Check user's token claims
curl -X GET https://llm-processor.nephoran.com/auth/userinfo \
  -H "Authorization: Bearer YOUR_TOKEN"
```

#### Secret Validation Failed

**Error**: `OAuth2 client secret validation failed`

**Solutions**:
1. Check secret meets minimum length requirements
2. Ensure secret doesn't contain placeholder values
3. Verify secret is properly base64 encoded in Kubernetes

```bash
# Check secret length
kubectl get secret oauth2-secrets -n nephoran-system -o jsonpath='{.data.azure-client-secret}' | base64 -d | wc -c
```

## Compliance and Standards

### OWASP Top 10 Coverage

| Risk | Implementation | Status |
|------|---------------|--------|
| A02:2021 - Cryptographic Failures | Strong secret validation, minimum key lengths, secure storage | ✅ |
| A03:2021 - Injection | Input validation, parameterized queries, path traversal protection | ✅ |
| A04:2021 - Insecure Design | Defense in depth, fail-secure design, threat modeling | ✅ |
| A05:2021 - Security Misconfiguration | Configuration validation, weak secret detection, secure defaults | ✅ |
| A07:2021 - Authentication Failures | Multi-provider OAuth2, strong authentication, account lockout | ✅ |
| A08:2021 - Software and Data Integrity | Signed JWTs, integrity verification, secure updates | ✅ |
| A09:2021 - Security Logging | Comprehensive audit logging, security event monitoring | ✅ |
| A10:2021 - SSRF | URL validation, restricted network access | ✅ |

### Compliance Considerations

#### GDPR Compliance
- Limit data exposure to authorized origins only
- Document all cross-origin data flows
- Implement right to erasure for user data
- Maintain audit logs for compliance tracking

#### SOC 2 Requirements
- Enforce principle of least privilege
- Maintain configuration change logs
- Regular security assessments
- Incident response procedures

#### PCI DSS (if handling payment data)
- Strong cryptography for data transmission
- Regular security testing
- Access control measures
- Security event monitoring

### Security Maintenance Schedule

| Task | Frequency | Description |
|------|-----------|-------------|
| Review authentication logs | Daily | Check for anomalies and failed attempts |
| Validate secret strength | Weekly | Ensure all secrets meet requirements |
| Rotate OAuth2 client secrets | Monthly | Update provider credentials |
| Security configuration audit | Quarterly | Review all security settings |
| Penetration testing | Annually | Third-party security assessment |
| Disaster recovery drill | Bi-annually | Test backup and recovery procedures |

## Security Best Practices Summary

1. **Defense in Depth**: Multiple security layers with OAuth2, JWT, RBAC, and request validation
2. **Fail Secure**: System defaults to secure state on any error condition
3. **Least Privilege**: Users get minimum required permissions through RBAC
4. **Secure by Default**: Production security automatically enforced
5. **Zero Trust**: Validate everything, trust nothing - all requests authenticated and authorized
6. **Comprehensive Auditing**: Every security-relevant event logged
7. **Encryption**: TLS 1.3 for all communications, secure token storage
8. **Input Validation**: All inputs validated against injection and overflow
9. **Secret Management**: Secure storage, validation, and rotation procedures
10. **Regular Updates**: Automated security patching and dependency updates

## Conclusion

The OAuth2 security implementation in the Nephoran Intent Operator provides enterprise-grade authentication and authorization with comprehensive security controls. The system implements defense-in-depth strategies, follows OWASP guidelines, and provides extensive audit capabilities while maintaining developer-friendly configuration options for non-production environments.

### Security Maturity Assessment

- **Authentication**: ⭐⭐⭐⭐⭐ Enterprise-grade multi-provider support
- **Authorization**: ⭐⭐⭐⭐⭐ Fine-grained RBAC with permission matrix
- **Secret Management**: ⭐⭐⭐⭐⭐ Comprehensive validation and protection
- **Audit Logging**: ⭐⭐⭐⭐⭐ Complete security event coverage
- **Configuration Security**: ⭐⭐⭐⭐⭐ Extensive validation and fail-safes
- **Overall Security Posture**: **PRODUCTION READY**

---

*Document Version: 2.0 (Consolidated)*  
*Last Updated: December 2024*  
*Security Classification: Internal*  
*Nephoran Intent Operator - Enterprise OAuth2 Security*