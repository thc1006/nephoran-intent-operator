# API Security Guide

## Table of Contents

1. [Introduction](#introduction)
2. [API Security Architecture](#api-security-architecture)
3. [Authentication Methods](#authentication-methods)
4. [Authorization Framework](#authorization-framework)
5. [API Endpoints Security](#api-endpoints-security)
6. [Request/Response Security](#requestresponse-security)
7. [Rate Limiting and Abuse Prevention](#rate-limiting-and-abuse-prevention)
8. [Security Headers](#security-headers)
9. [API Versioning and Compatibility](#api-versioning-and-compatibility)
10. [Security Testing](#security-testing)
11. [Monitoring and Logging](#monitoring-and-logging)

## Introduction

The Nephoran Intent Operator exposes secure APIs for managing network intents, patch generation, and system administration. This guide covers comprehensive security measures implemented to protect these APIs against common threats and ensure compliance with industry standards.

### Security Principles

- **Zero Trust**: Never trust, always verify
- **Defense in Depth**: Multiple layers of security
- **Least Privilege**: Minimal necessary permissions
- **Fail Secure**: Secure defaults and fail-safe mechanisms
- **Auditability**: Complete audit trail for all operations

### Threat Model

| Threat Category | Attack Vectors | Mitigations |
|----------------|----------------|-------------|
| **Injection Attacks** | SQL, NoSQL, LDAP, XSS | Input validation, parameterized queries |
| **Authentication Bypass** | Weak tokens, session hijacking | Strong tokens, MFA, secure sessions |
| **Authorization Bypass** | Privilege escalation, IDOR | RBAC, resource-level controls |
| **Data Exposure** | Information leakage, metadata | Output filtering, field-level security |
| **Denial of Service** | Rate limit bypass, resource exhaustion | Rate limiting, resource quotas |
| **Man-in-the-Middle** | TLS downgrade, certificate spoofing | TLS 1.3, certificate pinning |

## API Security Architecture

### Security Layers

```
┌─────────────────────────────────────────────┐
│              Client Applications             │
└──────────────────┬──────────────────────────┘
                   │ HTTPS/TLS 1.3
┌──────────────────▼──────────────────────────┐
│               Load Balancer                  │
│  • TLS Termination  • DDoS Protection      │
│  • Rate Limiting    • WAF Rules            │
└──────────────────┬──────────────────────────┘
                   │
┌──────────────────▼──────────────────────────┐
│              API Gateway                     │
│  • Authentication  • Authorization          │
│  • Request Validation  • Response Filtering │
└──────────────────┬──────────────────────────┘
                   │
┌──────────────────▼──────────────────────────┐
│            Nephoran API Server               │
│  • Business Logic  • Data Validation        │
│  • Audit Logging   • Error Handling         │
└──────────────────┬──────────────────────────┘
                   │
┌──────────────────▼──────────────────────────┐
│              Backend Services                │
│  • Database      • External APIs            │
│  • Message Queue • File Storage             │
└─────────────────────────────────────────────┘
```

### API Endpoints Overview

| Endpoint Group | Purpose | Security Level | Authentication Required |
|---------------|---------|----------------|------------------------|
| `/api/v1/auth/*` | Authentication & Authorization | High | No (registration/login) |
| `/api/v1/intents/*` | Network Intent Management | Critical | Yes |
| `/api/v1/patches/*` | Patch Generation & Management | Critical | Yes |
| `/api/v1/audit/*` | Audit Log Access | High | Yes (Admin only) |
| `/api/v1/admin/*` | System Administration | Critical | Yes (Admin only) |
| `/api/v1/health` | Health Check | Low | No |
| `/api/v1/metrics` | Monitoring Metrics | Medium | Yes (Monitoring role) |

## Authentication Methods

### 1. OAuth2/OIDC Authentication

#### Configuration
```yaml
# config/oauth2.yaml
oauth2:
  providers:
    google:
      client_id: ${GOOGLE_CLIENT_ID}
      client_secret: ${GOOGLE_CLIENT_SECRET}
      auth_url: https://accounts.google.com/o/oauth2/auth
      token_url: https://oauth2.googleapis.com/token
      user_info_url: https://openidconnect.googleapis.com/v1/userinfo
      scopes:
        - openid
        - email
        - profile
      
    azuread:
      client_id: ${AZURE_CLIENT_ID}
      client_secret: ${AZURE_CLIENT_SECRET}
      tenant_id: ${AZURE_TENANT_ID}
      auth_url: https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize
      token_url: https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token
      scopes:
        - openid
        - email
        - profile
```

#### Usage Example
```bash
# 1. Get authorization URL
curl -X GET "https://api.nephoran.io/api/v1/auth/providers/google/url" \
  -H "Accept: application/json"

# Response:
# {
#   "auth_url": "https://accounts.google.com/o/oauth2/auth?client_id=...",
#   "state": "random-state-string"
# }

# 2. Exchange authorization code for token
curl -X POST "https://api.nephoran.io/api/v1/auth/callback" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "google",
    "code": "authorization-code",
    "state": "random-state-string"
  }'

# Response:
# {
#   "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
#   "refresh_token": "refresh-token-string",
#   "expires_in": 3600,
#   "token_type": "Bearer"
# }
```

### 2. JWT Token Authentication

#### Token Structure
```json
{
  "header": {
    "alg": "RS256",
    "typ": "JWT",
    "kid": "key-id"
  },
  "payload": {
    "iss": "https://api.nephoran.io",
    "sub": "user-uuid",
    "aud": "nephoran-api",
    "exp": 1640995200,
    "iat": 1640991600,
    "jti": "token-uuid",
    "scope": "api:read api:write",
    "roles": ["operator"],
    "permissions": ["intent:create", "patch:generate"]
  }
}
```

#### Usage
```bash
# Include JWT token in Authorization header
curl -X GET "https://api.nephoran.io/api/v1/intents" \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### 3. API Key Authentication

#### API Key Format
```
API-Key: neph_ak_1234567890abcdef1234567890abcdef12345678
```

#### Usage
```bash
# Include API key in header
curl -X GET "https://api.nephoran.io/api/v1/intents" \
  -H "X-API-Key: neph_ak_1234567890abcdef1234567890abcdef12345678"
```

## Authorization Framework

### Role-Based Access Control (RBAC)

#### Role Definitions
```yaml
# roles.yaml
roles:
  admin:
    description: "Full system administration"
    permissions:
      - "*:*:*"  # resource:action:scope
    
  operator:
    description: "Network operations"
    permissions:
      - "intent:*:*"
      - "patch:*:*"
      - "deployment:read:*"
      - "audit:read:self"
    
  viewer:
    description: "Read-only access"
    permissions:
      - "intent:read:*"
      - "patch:read:*"
      - "deployment:read:*"
      - "audit:read:self"
    
  monitor:
    description: "Monitoring and metrics"
    permissions:
      - "metrics:read:*"
      - "health:read:*"
      - "status:read:*"
```

#### Permission Model
```go
type Permission struct {
    Resource string // intent, patch, deployment, audit
    Action   string // create, read, update, delete, list
    Scope    string // *, namespace, resource-id
}

// Examples:
// "intent:create:*" - Can create intents in any namespace
// "patch:read:default" - Can read patches in default namespace
// "audit:read:self" - Can read own audit logs
```

### Resource-Level Authorization

#### Intent Authorization
```go
func (h *IntentHandler) authorizeIntentAccess(ctx context.Context, intentID string, action string) error {
    authCtx := auth.GetAuthContext(ctx)
    
    // Check global permission
    if h.rbac.HasPermission(authCtx, fmt.Sprintf("intent:%s:*", action)) {
        return nil
    }
    
    // Check namespace-level permission
    intent, err := h.getIntent(intentID)
    if err != nil {
        return err
    }
    
    permission := fmt.Sprintf("intent:%s:%s", action, intent.Namespace)
    if h.rbac.HasPermission(authCtx, permission) {
        return nil
    }
    
    // Check resource-level permission
    permission = fmt.Sprintf("intent:%s:%s", action, intentID)
    if h.rbac.HasPermission(authCtx, permission) {
        return nil
    }
    
    return errors.New("insufficient permissions")
}
```

## API Endpoints Security

### 1. Intent Management API

#### Create Intent
```http
POST /api/v1/intents
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "target": "nginx-deployment",
  "namespace": "production",
  "replicas": 5,
  "intent_type": "scaling",
  "reason": "Increased traffic",
  "source": "api-client"
}
```

**Security Controls:**
- JWT authentication required
- Input validation against schema
- RBAC permission: `intent:create:*` or `intent:create:production`
- Audit logging of creation event
- Rate limiting: 100 requests/minute

#### Get Intent
```http
GET /api/v1/intents/{intent-id}
Authorization: Bearer <jwt-token>
```

**Security Controls:**
- JWT authentication required
- Resource-level authorization
- RBAC permission: `intent:read:*` or `intent:read:{namespace}` or `intent:read:{intent-id}`
- Audit logging of access
- Response filtering based on permissions

### 2. Patch Generation API

#### Generate Patch
```http
POST /api/v1/patches/generate
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "intent_id": "intent-uuid",
  "output_format": "kpt",
  "security_validation": true
}
```

**Security Controls:**
- JWT authentication required
- RBAC permission: `patch:create:*`
- Intent ownership validation
- OWASP-compliant input validation
- Secure patch generation with cryptographic naming
- Audit logging with full context

#### List Patches
```http
GET /api/v1/patches?namespace=production&limit=50
Authorization: Bearer <jwt-token>
```

**Security Controls:**
- JWT authentication required
- Namespace-level filtering
- Pagination limits enforced
- Response filtering based on permissions

### 3. Audit Log API

#### Query Audit Logs
```http
GET /api/v1/audit/logs?start_time=2023-01-01T00:00:00Z&event_type=authentication
Authorization: Bearer <jwt-token>
```

**Security Controls:**
- JWT authentication required
- Admin role required for system-wide logs
- Users can only access their own audit logs
- Time range restrictions
- Result filtering and redaction

## Request/Response Security

### Input Validation

#### Validation Framework
```go
type Validator struct {
    owaspValidator *security.OWASPValidator
    schemaValidator *jsonschema.Validator
}

func (v *Validator) ValidateIntent(intent *Intent) error {
    // 1. Schema validation
    if err := v.schemaValidator.Validate(intent); err != nil {
        return fmt.Errorf("schema validation failed: %w", err)
    }
    
    // 2. Security validation
    result, err := v.owaspValidator.ValidateIntentFile(intent)
    if err != nil {
        return fmt.Errorf("security validation failed: %w", err)
    }
    
    if !result.IsValid {
        return fmt.Errorf("security violations detected: %v", result.Violations)
    }
    
    // 3. Business logic validation
    if err := v.validateBusinessRules(intent); err != nil {
        return fmt.Errorf("business validation failed: %w", err)
    }
    
    return nil
}
```

#### Input Sanitization
```go
func sanitizeString(input string) string {
    // Remove dangerous characters
    input = strings.ReplaceAll(input, "<script", "")
    input = strings.ReplaceAll(input, "javascript:", "")
    input = strings.ReplaceAll(input, "eval(", "")
    
    // Limit length
    if len(input) > 512 {
        input = input[:512]
    }
    
    // HTML encode
    return html.EscapeString(input)
}
```

### Output Filtering

#### Response Filtering
```go
func (h *Handler) filterResponse(ctx context.Context, data interface{}) interface{} {
    authCtx := auth.GetAuthContext(ctx)
    
    switch v := data.(type) {
    case *Intent:
        return h.filterIntent(authCtx, v)
    case []*Intent:
        var filtered []*Intent
        for _, intent := range v {
            if filtered := h.filterIntent(authCtx, intent); filtered != nil {
                filtered = append(filtered, filtered)
            }
        }
        return filtered
    }
    
    return data
}

func (h *Handler) filterIntent(authCtx *auth.AuthContext, intent *Intent) *Intent {
    // Check read permission
    if !h.rbac.HasPermission(authCtx, "intent:read:"+intent.ID) {
        return nil
    }
    
    // Filter sensitive fields based on permissions
    result := *intent
    if !h.rbac.HasPermission(authCtx, "intent:read-sensitive:"+intent.ID) {
        result.CorrelationID = ""
        result.Source = "redacted"
    }
    
    return &result
}
```

### Error Handling

#### Secure Error Responses
```go
func (h *Handler) handleError(w http.ResponseWriter, err error) {
    var statusCode int
    var message string
    var code string
    
    switch {
    case errors.Is(err, auth.ErrUnauthorized):
        statusCode = http.StatusUnauthorized
        message = "Authentication required"
        code = "AUTH_REQUIRED"
        
    case errors.Is(err, auth.ErrForbidden):
        statusCode = http.StatusForbidden
        message = "Insufficient permissions"
        code = "INSUFFICIENT_PERMISSIONS"
        
    case errors.Is(err, validation.ErrInvalidInput):
        statusCode = http.StatusBadRequest
        message = "Invalid request data"
        code = "INVALID_INPUT"
        
    default:
        // Don't expose internal errors
        statusCode = http.StatusInternalServerError
        message = "Internal server error"
        code = "INTERNAL_ERROR"
        
        // Log detailed error internally
        h.logger.Error("Internal error", "error", err, "request_id", getRequestID(r))
    }
    
    response := ErrorResponse{
        Code:    code,
        Message: message,
        RequestID: getRequestID(r),
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(response)
}
```

## Rate Limiting and Abuse Prevention

### Rate Limiting Implementation

#### Configuration
```yaml
# rate-limiting.yaml
rate_limiting:
  global:
    requests_per_minute: 1000
    burst_size: 100
    
  per_endpoint:
    "/api/v1/auth/login":
      requests_per_minute: 10
      burst_size: 2
      window: 60s
      
    "/api/v1/intents":
      requests_per_minute: 100
      burst_size: 10
      
    "/api/v1/patches/generate":
      requests_per_minute: 20
      burst_size: 5
      
  per_user:
    requests_per_minute: 200
    burst_size: 20
    
  per_ip:
    requests_per_minute: 500
    burst_size: 50
```

#### Rate Limiter Implementation
```go
type RateLimiter struct {
    store   RateLimitStore
    config  *RateLimitConfig
    metrics *prometheus.CounterVec
}

func (rl *RateLimiter) CheckLimit(ctx context.Context, key string, limit int) (bool, time.Duration, error) {
    current, err := rl.store.Increment(key, time.Minute)
    if err != nil {
        return false, 0, err
    }
    
    if current > limit {
        // Calculate reset time
        resetTime := time.Minute - time.Duration(current%60)*time.Second
        
        // Record metric
        rl.metrics.WithLabelValues("exceeded").Inc()
        
        return false, resetTime, nil
    }
    
    rl.metrics.WithLabelValues("allowed").Inc()
    return true, 0, nil
}
```

### Abuse Prevention

#### Request Validation
```go
func (h *Handler) validateRequest(r *http.Request) error {
    // Check content length
    if r.ContentLength > MaxRequestSize {
        return errors.New("request too large")
    }
    
    // Check request method
    if !isAllowedMethod(r.Method) {
        return errors.New("method not allowed")
    }
    
    // Check content type
    if !isAllowedContentType(r.Header.Get("Content-Type")) {
        return errors.New("unsupported content type")
    }
    
    // Check user agent
    if isBotUserAgent(r.Header.Get("User-Agent")) {
        return errors.New("bot requests not allowed")
    }
    
    return nil
}
```

#### IP Blocking
```go
func (h *Handler) checkIPBlock(ip string) error {
    // Check against static blocklist
    if h.isBlocked(ip) {
        return errors.New("IP address blocked")
    }
    
    // Check against dynamic blocklist
    violations, err := h.getViolations(ip)
    if err != nil {
        return err
    }
    
    if violations > MaxViolations {
        h.blockIP(ip, time.Hour)
        return errors.New("IP address temporarily blocked")
    }
    
    return nil
}
```

## Security Headers

### HTTP Security Headers

```go
func (h *SecurityHeaders) SetHeaders(w http.ResponseWriter, r *http.Request) {
    // X-Frame-Options
    w.Header().Set("X-Frame-Options", "DENY")
    
    // X-Content-Type-Options
    w.Header().Set("X-Content-Type-Options", "nosniff")
    
    // X-XSS-Protection
    w.Header().Set("X-XSS-Protection", "1; mode=block")
    
    // Referrer-Policy
    w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
    
    // Content-Security-Policy
    csp := "default-src 'none'; " +
           "script-src 'self'; " +
           "style-src 'self' 'unsafe-inline'; " +
           "img-src 'self' data: https:; " +
           "font-src 'self'; " +
           "connect-src 'self'; " +
           "frame-ancestors 'none'; " +
           "base-uri 'none'; " +
           "form-action 'self'"
    w.Header().Set("Content-Security-Policy", csp)
    
    // Permissions-Policy
    permissionsPolicy := "geolocation=(), microphone=(), camera=(), " +
                        "payment=(), usb=(), magnetometer=(), " +
                        "gyroscope=(), accelerometer=()"
    w.Header().Set("Permissions-Policy", permissionsPolicy)
    
    // HSTS (only for HTTPS)
    if r.TLS != nil {
        w.Header().Set("Strict-Transport-Security", 
                      "max-age=31536000; includeSubDomains; preload")
    }
    
    // Custom security headers
    w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
    w.Header().Set("X-Download-Options", "noopen")
    w.Header().Set("X-DNS-Prefetch-Control", "off")
}
```

### CORS Configuration

```yaml
# cors.yaml
cors:
  allowed_origins:
    - "https://ui.nephoran.io"
    - "https://admin.nephoran.io"
  
  allowed_methods:
    - GET
    - POST
    - PUT
    - DELETE
    - OPTIONS
  
  allowed_headers:
    - Authorization
    - Content-Type
    - X-Requested-With
    - X-API-Key
    - X-Request-ID
  
  exposed_headers:
    - X-Request-ID
    - X-RateLimit-Remaining
    - X-RateLimit-Reset
  
  allow_credentials: true
  max_age: 86400
```

## API Versioning and Compatibility

### Version Strategy

```
API Version Format: /api/v{major}/...

v1: /api/v1/intents
v2: /api/v2/intents (future)
```

### Backward Compatibility

```go
type VersionHandler struct {
    v1Handler *V1Handler
    v2Handler *V2Handler
}

func (vh *VersionHandler) RouteRequest(w http.ResponseWriter, r *http.Request) {
    version := extractVersion(r.URL.Path)
    
    switch version {
    case "v1":
        vh.v1Handler.ServeHTTP(w, r)
    case "v2":
        vh.v2Handler.ServeHTTP(w, r)
    default:
        // Default to latest stable version
        vh.v1Handler.ServeHTTP(w, r)
    }
}
```

### API Deprecation

```go
func (h *Handler) handleDeprecatedEndpoint(w http.ResponseWriter, r *http.Request) {
    // Add deprecation headers
    w.Header().Set("Deprecation", "Sun, 01 Jan 2024 00:00:00 GMT")
    w.Header().Set("Sunset", "Sun, 01 Jul 2024 00:00:00 GMT")
    w.Header().Set("Link", `</api/v2/intents>; rel="successor-version"`)
    
    // Log deprecation usage
    h.logger.Warn("Deprecated API endpoint accessed",
        "path", r.URL.Path,
        "user", getUser(r),
        "deprecation_date", "2024-07-01")
    
    // Continue with normal processing
    h.handleRequest(w, r)
}
```

## Security Testing

### Automated Security Testing

```yaml
# security-tests.yaml
security_tests:
  authentication:
    - test: "Unauthenticated access blocked"
      endpoint: "/api/v1/intents"
      method: "GET"
      expect_status: 401
      
    - test: "Invalid token rejected"
      endpoint: "/api/v1/intents"
      method: "GET"
      headers:
        Authorization: "Bearer invalid-token"
      expect_status: 401
      
  authorization:
    - test: "Insufficient permissions blocked"
      endpoint: "/api/v1/admin/users"
      method: "GET"
      role: "viewer"
      expect_status: 403
      
  input_validation:
    - test: "SQL injection blocked"
      endpoint: "/api/v1/intents"
      method: "POST"
      body: |
        {
          "target": "'; DROP TABLE intents; --",
          "namespace": "default",
          "replicas": 1
        }
      expect_status: 400
      
    - test: "XSS attempt blocked"
      endpoint: "/api/v1/intents"
      method: "POST"
      body: |
        {
          "target": "<script>alert('xss')</script>",
          "namespace": "default",
          "replicas": 1
        }
      expect_status: 400
      
  rate_limiting:
    - test: "Rate limit enforced"
      endpoint: "/api/v1/intents"
      method: "GET"
      requests: 101
      expect_status: 429
```

### Penetration Testing

```bash
#!/bin/bash
# api-security-test.sh

BASE_URL="https://api.nephoran.io"
OUTPUT_DIR="security-test-results"

mkdir -p $OUTPUT_DIR

echo "Starting API security tests..."

# 1. SSL/TLS Testing
echo "Testing SSL/TLS configuration..."
sslscan $BASE_URL > $OUTPUT_DIR/ssl-scan.txt
testssl.sh $BASE_URL > $OUTPUT_DIR/testssl.txt

# 2. Authentication Testing
echo "Testing authentication endpoints..."
# Test weak authentication
curl -X POST "$BASE_URL/api/v1/auth/login" \
  -d '{"username":"admin","password":"admin"}' \
  -w "%{http_code}" > $OUTPUT_DIR/auth-weak.txt

# 3. Authorization Testing
echo "Testing authorization bypass..."
# Test without token
curl -X GET "$BASE_URL/api/v1/admin/users" \
  -w "%{http_code}" > $OUTPUT_DIR/authz-bypass.txt

# 4. Input Validation Testing
echo "Testing input validation..."
# SQL injection test
curl -X POST "$BASE_URL/api/v1/intents" \
  -H "Authorization: Bearer $VALID_TOKEN" \
  -d '{"target":"'\''OR 1=1--","namespace":"default","replicas":1}' \
  -w "%{http_code}" > $OUTPUT_DIR/sql-injection.txt

# 5. Rate Limiting Testing
echo "Testing rate limiting..."
for i in {1..105}; do
  curl -X GET "$BASE_URL/api/v1/intents" \
    -H "Authorization: Bearer $VALID_TOKEN" \
    -w "%{http_code}\n" >> $OUTPUT_DIR/rate-limit.txt
done

echo "Security tests completed. Results in $OUTPUT_DIR/"
```

## Monitoring and Logging

### Security Metrics

```yaml
# security-metrics.yaml
metrics:
  authentication:
    - name: nephoran_auth_attempts_total
      type: counter
      labels: [method, result]
      description: "Total authentication attempts"
      
    - name: nephoran_auth_duration_seconds
      type: histogram
      labels: [method]
      description: "Authentication duration"
      
  authorization:
    - name: nephoran_authz_decisions_total
      type: counter
      labels: [decision, resource]
      description: "Authorization decisions"
      
  api_security:
    - name: nephoran_api_requests_total
      type: counter
      labels: [endpoint, method, status]
      description: "API request count"
      
    - name: nephoran_rate_limit_violations_total
      type: counter
      labels: [endpoint, client]
      description: "Rate limit violations"
      
    - name: nephoran_security_violations_total
      type: counter
      labels: [type, severity]
      description: "Security violations detected"
```

### Audit Logging

```go
func (h *Handler) auditAPIRequest(r *http.Request, response *Response, duration time.Duration) {
    authCtx := auth.GetAuthContext(r.Context())
    
    event := &audit.Event{
        Type:      audit.EventTypeAPIAccess,
        Timestamp: time.Now(),
        UserID:    authCtx.UserID,
        SessionID: authCtx.SessionID,
        Action:    r.Method + " " + r.URL.Path,
        Resource:  extractResourceFromPath(r.URL.Path),
        Result:    audit.ResultFromHTTPStatus(response.StatusCode),
        Metadata: map[string]interface{}{
            "remote_addr":    getClientIP(r),
            "user_agent":     r.UserAgent(),
            "request_id":     getRequestID(r),
            "response_size":  response.Size,
            "duration_ms":    duration.Milliseconds(),
            "rate_limited":   response.RateLimited,
        },
    }
    
    h.auditor.LogEvent(event)
}
```

### Security Monitoring

```yaml
# security-monitoring.yaml
alerts:
  - name: HighAuthFailureRate
    expr: |
      rate(nephoran_auth_attempts_total{result="failure"}[5m]) > 10
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: High authentication failure rate
      description: "{{ $value }} auth failures per second"
      
  - name: SuspiciousAPIActivity
    expr: |
      increase(nephoran_security_violations_total{severity="high"}[1m]) > 5
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: Suspicious API activity detected
      description: "{{ $value }} high-severity security violations"
      
  - name: RateLimitViolationSpike
    expr: |
      increase(nephoran_rate_limit_violations_total[5m]) > 100
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: Rate limit violation spike
      description: "{{ $value }} rate limit violations in 5 minutes"
```

---

*Document Version: 1.0*  
*Last Updated: 2025-08-19*  
*Classification: Internal*  
*Next Review: 2025-10-19*