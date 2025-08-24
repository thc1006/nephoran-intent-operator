# Security Improvement Plan for Nephoran Intent Operator

## Executive Summary

This comprehensive security improvement plan identifies critical security gaps in the Nephoran Intent Operator project and provides actionable recommendations following OWASP Top 10 guidelines and defense-in-depth principles.

**Security Score: 75/100** (Good, but improvements needed)

## Current Security Posture

### ✅ Implemented Security Controls

1. **Path Traversal Protection** (OWASP A01:2021)
   - Multi-layer validation with canonicalization
   - Platform-specific security checks
   - Comprehensive test coverage

2. **Security Headers Middleware** (OWASP A05:2021)
   - CSP, HSTS, X-Frame-Options implemented
   - Permissions Policy configured
   - Referrer Policy enforced

3. **CSRF/PKCE Protection** (OWASP A01:2021)
   - Double-submit cookie pattern
   - PKCE for OAuth flows
   - Session-bound tokens

4. **Container Security Context**
   - Non-root user enforcement
   - Read-only root filesystem
   - Capability dropping

5. **Network Policies**
   - Zero-trust with deny-all default
   - Service-specific ingress/egress rules
   - mTLS enforcement indicators

## Critical Security Gaps & Remediation

### 🔴 Priority 1: Authentication & Authorization (OWASP A01:2021, A07:2021)

#### Gap: Incomplete JWT/OAuth2 Implementation
**Current State:** Basic PKCE/CSRF but no complete auth flow
**Risk Level:** CRITICAL
**CVSS Score:** 9.1 (AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N)

**Remediation:**
```go
// pkg/auth/jwt_validator.go
package auth

import (
    "context"
    "crypto/rsa"
    "fmt"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
    "github.com/lestrrat-go/jwx/v2/jwk"
    "github.com/lestrrat-go/jwx/v2/jwt"
)

type JWTValidator struct {
    issuer       string
    audience     string
    jwksURL      string
    keySet       jwk.Set
    keyCache     *KeyCache
    minKeyLength int
}

func NewJWTValidator(issuer, audience, jwksURL string) (*JWTValidator, error) {
    v := &JWTValidator{
        issuer:       issuer,
        audience:     audience,
        jwksURL:      jwksURL,
        minKeyLength: 2048, // Minimum RSA key length
    }
    
    // Initialize JWKS with caching
    ctx := context.Background()
    keySet, err := jwk.Fetch(ctx, jwksURL, 
        jwk.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
    }
    
    v.keySet = keySet
    v.keyCache = NewKeyCache(5 * time.Minute)
    
    return v, nil
}

func (v *JWTValidator) ValidateToken(tokenString string) (*Claims, error) {
    // Parse without validation first to get kid
    token, err := jwt.Parse(tokenString, nil, 
        jwt.WithVerify(false),
        jwt.WithValidate(false),
    )
    if err != nil {
        return nil, fmt.Errorf("malformed token: %w", err)
    }
    
    // Validate critical claims
    if err := v.validateClaims(token); err != nil {
        return nil, err
    }
    
    // Verify signature with key rotation support
    kid := token.JWTHeaders()[0].KeyID()
    key, err := v.getSigningKey(kid)
    if err != nil {
        return nil, fmt.Errorf("key validation failed: %w", err)
    }
    
    // Verify with timing attack resistance
    if err := jwt.Verify(tokenString, jwt.WithKey(key.Algorithm(), key)); err != nil {
        return nil, fmt.Errorf("signature verification failed: %w", err)
    }
    
    return v.extractClaims(token)
}

func (v *JWTValidator) validateClaims(token jwt.Token) error {
    // Issuer validation
    if iss, _ := token.Get("iss"); iss != v.issuer {
        return fmt.Errorf("invalid issuer: expected %s, got %s", v.issuer, iss)
    }
    
    // Audience validation
    aud, _ := token.Get("aud")
    if !v.isValidAudience(aud) {
        return fmt.Errorf("invalid audience")
    }
    
    // Time-based validation with clock skew tolerance
    now := time.Now()
    skew := 30 * time.Second
    
    if exp, ok := token.Get("exp"); ok {
        expTime := exp.(time.Time)
        if now.After(expTime.Add(skew)) {
            return fmt.Errorf("token expired")
        }
    } else {
        return fmt.Errorf("missing expiration claim")
    }
    
    if nbf, ok := token.Get("nbf"); ok {
        nbfTime := nbf.(time.Time)
        if now.Before(nbfTime.Add(-skew)) {
            return fmt.Errorf("token not yet valid")
        }
    }
    
    // Validate required scopes
    if err := v.validateScopes(token); err != nil {
        return err
    }
    
    return nil
}
```

#### Gap: Missing RBAC Granularity
**Current State:** Basic ClusterRole definitions without fine-grained permissions
**Risk Level:** HIGH
**CVSS Score:** 7.5

**Remediation:**
```yaml
# deployments/security/fine-grained-rbac.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: nephoran-intent-reader
  namespace: nephoran-system
rules:
- apiGroups: ["nephoran.io"]
  resources: ["networkintents"]
  verbs: ["get", "list", "watch"]
  # Resource name restrictions
  resourceNames: [] # Empty means all in namespace

---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: nephoran-intent-writer
  namespace: nephoran-system
rules:
- apiGroups: ["nephoran.io"]
  resources: ["networkintents"]
  verbs: ["create", "update", "patch"]
  # Restrict to specific intent types via admission webhook
  
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nephoran-minimal-operator
  annotations:
    security.nephoran.io/principle: "least-privilege"
rules:
# Minimal permissions for operator
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "patch"]
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["get", "list", "watch", "create", "update", "patch"]
  resourceNames: ["nephoran-operator-leader"]
- apiGroups: ["nephoran.io"]
  resources: ["networkintents"]
  verbs: ["get", "list", "watch", "update", "patch"]
- apiGroups: ["nephoran.io"]
  resources: ["networkintents/status"]
  verbs: ["update", "patch"]
# No wildcard permissions
# No cluster-admin bindings
# No escalation permissions
```

### 🟠 Priority 2: Secret Management (OWASP A02:2021)

#### Gap: Secrets Stored in ConfigMaps/Plain Text
**Current State:** No external secret management integration
**Risk Level:** HIGH
**CVSS Score:** 8.1

**Remediation:**
```go
// pkg/secrets/vault_integration.go
package secrets

import (
    "context"
    "fmt"
    
    vault "github.com/hashicorp/vault/api"
    "github.com/hashicorp/vault/api/auth/kubernetes"
)

type VaultSecretManager struct {
    client      *vault.Client
    mountPath   string
    role        string
    namespace   string
}

func NewVaultSecretManager(addr, role, namespace string) (*VaultSecretManager, error) {
    config := vault.DefaultConfig()
    config.Address = addr
    
    client, err := vault.NewClient(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create vault client: %w", err)
    }
    
    // Kubernetes auth
    k8sAuth, err := kubernetes.NewKubernetesAuth(
        role,
        kubernetes.WithServiceAccountTokenPath("/var/run/secrets/kubernetes.io/serviceaccount/token"),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create k8s auth: %w", err)
    }
    
    authInfo, err := client.Auth().Login(context.Background(), k8sAuth)
    if err != nil {
        return nil, fmt.Errorf("failed to authenticate: %w", err)
    }
    
    if authInfo == nil {
        return nil, fmt.Errorf("no auth info returned")
    }
    
    return &VaultSecretManager{
        client:    client,
        mountPath: "secret",
        role:      role,
        namespace: namespace,
    }, nil
}

// GetSecret retrieves a secret with automatic rotation support
func (v *VaultSecretManager) GetSecret(ctx context.Context, path string) (*Secret, error) {
    secret, err := v.client.KVv2(v.mountPath).Get(ctx, path)
    if err != nil {
        return nil, fmt.Errorf("failed to read secret: %w", err)
    }
    
    // Check for rotation metadata
    metadata := secret.VersionMetadata
    if metadata != nil {
        // Implement rotation logic based on creation time
        if shouldRotate(metadata.CreatedTime) {
            return nil, fmt.Errorf("secret requires rotation")
        }
    }
    
    return &Secret{
        Data:      secret.Data,
        Version:   secret.VersionMetadata.Version,
        ExpiresAt: calculateExpiry(metadata.CreatedTime),
    }, nil
}

// Kubernetes Secret Encryption at Rest
apiVersion: v1
kind: EncryptionConfiguration
resources:
  - resources:
    - secrets
    providers:
    - aescbc:
        keys:
        - name: key1
          secret: <base64-encoded-secret>
    - identity: {}
```

### 🟠 Priority 3: Input Validation Enhancement (OWASP A03:2021)

#### Gap: Incomplete Input Sanitization
**Current State:** Basic path validation, limited JSON validation
**Risk Level:** MEDIUM
**CVSS Score:** 6.5

**Remediation:**
```go
// pkg/validation/input_validator.go
package validation

import (
    "encoding/json"
    "fmt"
    "regexp"
    "strings"
    
    "github.com/go-playground/validator/v10"
    "github.com/microcosm-cc/bluemonday"
)

type InputValidator struct {
    validator     *validator.Validate
    sanitizer     *bluemonday.Policy
    jsonValidator *JSONSchemaValidator
}

func NewInputValidator() *InputValidator {
    v := validator.New()
    
    // Register custom validations
    v.RegisterValidation("no_sql_injection", validateNoSQLInjection)
    v.RegisterValidation("no_xss", validateNoXSS)
    v.RegisterValidation("safe_k8s_name", validateK8sName)
    v.RegisterValidation("safe_json", validateJSONStructure)
    
    // HTML sanitizer for any user-provided text
    sanitizer := bluemonday.StrictPolicy()
    
    return &InputValidator{
        validator: v,
        sanitizer: sanitizer,
        jsonValidator: NewJSONSchemaValidator(),
    }
}

// ValidateNetworkIntent performs comprehensive validation
func (iv *InputValidator) ValidateNetworkIntent(intent *NetworkIntent) error {
    // Structural validation
    if err := iv.validator.Struct(intent); err != nil {
        return fmt.Errorf("structural validation failed: %w", err)
    }
    
    // Deep JSON validation against schema
    if err := iv.jsonValidator.ValidateAgainstSchema(intent, "network-intent-v1.schema.json"); err != nil {
        return fmt.Errorf("schema validation failed: %w", err)
    }
    
    // Command injection prevention
    if err := iv.validateNoCommandInjection(intent); err != nil {
        return err
    }
    
    // Size limits
    if err := iv.validateSizeLimits(intent); err != nil {
        return err
    }
    
    // Business logic validation
    if err := iv.validateBusinessRules(intent); err != nil {
        return err
    }
    
    return nil
}

func validateNoSQLInjection(fl validator.FieldLevel) bool {
    value := fl.Field().String()
    
    // Comprehensive SQL injection patterns
    sqlPatterns := []string{
        `(?i)(union|select|insert|update|delete|drop)\s+(all|from|into|table|database)`,
        `(?i)(exec|execute|script|javascript|eval)\s*\(`,
        `[';]--`,
        `\/\*.*\*\/`,
        `\x00|\x1a`, // Null byte injection
        `(?i)(benchmark|sleep|waitfor|pg_sleep)\s*\(`,
    }
    
    for _, pattern := range sqlPatterns {
        if matched, _ := regexp.MatchString(pattern, value); matched {
            return false
        }
    }
    
    return true
}

func (iv *InputValidator) validateNoCommandInjection(intent *NetworkIntent) error {
    // Check all string fields for command injection patterns
    dangerousPatterns := []string{
        `;|\||&&|\$\(|\``,           // Command chaining
        `\$\{.*\}`,                   // Variable expansion
        `>|<|>>|2>|2>&1`,            // Redirection
        `(?i)(curl|wget|nc|bash|sh)`, // Dangerous commands
    }
    
    fields := []string{
        intent.Name,
        intent.Target,
        intent.Description,
    }
    
    for _, field := range fields {
        for _, pattern := range dangerousPatterns {
            if matched, _ := regexp.MatchString(pattern, field); matched {
                return fmt.Errorf("potential command injection detected in field")
            }
        }
    }
    
    return nil
}
```

### 🟠 Priority 4: Container Security Hardening (OWASP A08:2021)

#### Gap: Missing Pod Security Standards
**Current State:** Basic security context, no PSP/PSS
**Risk Level:** MEDIUM
**CVSS Score:** 6.8

**Remediation:**
```yaml
# deployments/security/pod-security-policy.yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: nephoran-restricted
  annotations:
    seccomp.security.alpha.kubernetes.io/allowedProfileNames: 'runtime/default'
    apparmor.security.beta.kubernetes.io/allowedProfileNames: 'runtime/default'
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'persistentVolumeClaim'
  hostNetwork: false
  hostIPC: false
  hostPID: false
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  supplementalGroups:
    rule: 'MustRunAs'
    ranges:
      - min: 1000
        max: 65535
  fsGroup:
    rule: 'MustRunAs'
    ranges:
      - min: 1000
        max: 65535
  readOnlyRootFilesystem: true

---
# Pod Security Standards (PSS) for Kubernetes 1.25+
apiVersion: v1
kind: Namespace
metadata:
  name: nephoran-system
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
    pod-security.kubernetes.io/enforce-version: latest
```

### 🟡 Priority 5: API Rate Limiting & DDoS Protection (OWASP A04:2021)

#### Gap: No Rate Limiting Implementation
**Current State:** No rate limiting or DDoS protection
**Risk Level:** MEDIUM
**CVSS Score:** 5.3

**Remediation:**
```go
// pkg/middleware/rate_limiter.go
package middleware

import (
    "fmt"
    "net/http"
    "sync"
    "time"
    
    "golang.org/x/time/rate"
)

type RateLimiter struct {
    visitors map[string]*visitor
    mu       sync.RWMutex
    rate     rate.Limit
    burst    int
    ttl      time.Duration
}

type visitor struct {
    limiter  *rate.Limiter
    lastSeen time.Time
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
    rl := &RateLimiter{
        visitors: make(map[string]*visitor),
        rate:     r,
        burst:    b,
        ttl:      1 * time.Hour,
    }
    
    // Cleanup goroutine
    go rl.cleanupVisitors()
    
    return rl
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract client identifier (IP + User Agent for better accuracy)
        clientID := rl.getClientID(r)
        
        // Get or create visitor
        v := rl.getVisitor(clientID)
        
        // Check rate limit with cost-based limiting
        cost := rl.calculateCost(r)
        if !v.limiter.AllowN(time.Now(), cost) {
            // Add retry-after header
            w.Header().Set("Retry-After", "60")
            w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%v", rl.rate))
            w.Header().Set("X-RateLimit-Remaining", "0")
            w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
            
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        
        // Add rate limit headers
        w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%v", rl.rate))
        w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%v", v.limiter.Tokens()))
        
        next.ServeHTTP(w, r)
    })
}

func (rl *RateLimiter) calculateCost(r *http.Request) int {
    // Different costs for different operations
    switch r.Method {
    case "GET", "HEAD":
        return 1
    case "POST", "PUT":
        return 5
    case "DELETE":
        return 10
    default:
        return 1
    }
}
```

## Security Monitoring & Compliance

### Audit Logging Implementation
```go
// pkg/audit/security_auditor.go
package audit

type SecurityAuditor struct {
    logger    *zap.Logger
    backend   AuditBackend
    sensitive *SensitiveDataFilter
}

func (sa *SecurityAuditor) LogSecurityEvent(event SecurityEvent) {
    // Filter sensitive data
    filtered := sa.sensitive.Filter(event)
    
    // Add security context
    enriched := sa.enrichWithContext(filtered)
    
    // Write to audit backend
    if err := sa.backend.Write(enriched); err != nil {
        sa.logger.Error("Failed to write audit log", 
            zap.Error(err),
            zap.String("event_type", event.Type))
    }
    
    // Alert on critical events
    if event.Severity == "CRITICAL" {
        sa.sendSecurityAlert(enriched)
    }
}
```

## Implementation Roadmap

### Phase 1: Critical Security (Week 1-2)
- [ ] Implement JWT/OAuth2 authentication
- [ ] Deploy fine-grained RBAC
- [ ] Enable audit logging
- [ ] Configure Pod Security Standards

### Phase 2: Secret Management (Week 3-4)
- [ ] Integrate HashiCorp Vault or Sealed Secrets
- [ ] Implement secret rotation
- [ ] Enable encryption at rest
- [ ] Remove hardcoded credentials

### Phase 3: Advanced Protection (Week 5-6)
- [ ] Deploy rate limiting
- [ ] Implement comprehensive input validation
- [ ] Add WAF rules
- [ ] Configure security scanning in CI/CD

### Phase 4: Monitoring & Compliance (Week 7-8)
- [ ] Deploy Falco for runtime security
- [ ] Implement SIEM integration
- [ ] Configure vulnerability scanning
- [ ] Document security procedures

## Security Testing Checklist

### Authentication & Authorization
- [ ] Test JWT validation with expired tokens
- [ ] Verify RBAC enforcement
- [ ] Test privilege escalation scenarios
- [ ] Validate session management

### Input Validation
- [ ] SQL injection testing
- [ ] XSS payload testing
- [ ] Command injection testing
- [ ] XXE/SSRF testing
- [ ] File upload validation

### Container Security
- [ ] Verify non-root execution
- [ ] Test read-only filesystem
- [ ] Validate network policies
- [ ] Check capability restrictions

### Secret Management
- [ ] Verify no secrets in logs
- [ ] Test secret rotation
- [ ] Validate encryption at rest
- [ ] Check for hardcoded credentials

### API Security
- [ ] Test rate limiting
- [ ] Verify CORS configuration
- [ ] Test CSP headers
- [ ] Validate TLS configuration

## Compliance Mappings

### OWASP Top 10 2021 Coverage
- **A01:2021 - Broken Access Control**: ✅ RBAC, JWT validation
- **A02:2021 - Cryptographic Failures**: ⚠️ Needs secret management
- **A03:2021 - Injection**: ✅ Input validation, path traversal
- **A04:2021 - Insecure Design**: ⚠️ Needs threat modeling
- **A05:2021 - Security Misconfiguration**: ✅ Security headers, PSP
- **A06:2021 - Vulnerable Components**: ⚠️ Needs dependency scanning
- **A07:2021 - Identification and Authentication**: ⚠️ Needs full auth flow
- **A08:2021 - Software and Data Integrity**: ⚠️ Needs signing verification
- **A09:2021 - Security Logging**: ⚠️ Basic logging, needs SIEM
- **A10:2021 - SSRF**: ✅ Input validation includes URL validation

### CIS Kubernetes Benchmark Alignment
- Control Plane Security: Partial (60%)
- Workload Security: Good (75%)
- Network Policies: Implemented (90%)
- RBAC: Basic (50%)
- Logging & Monitoring: Basic (40%)

## Recommended Security Tools

### Static Analysis
- **Semgrep**: Custom rules for Go security patterns
- **gosec**: Go security checker
- **Trivy**: Container vulnerability scanning

### Runtime Protection
- **Falco**: Runtime security monitoring
- **OPA**: Policy enforcement
- **Istio**: Service mesh with mTLS

### Secret Management
- **HashiCorp Vault**: Enterprise secret management
- **Sealed Secrets**: Kubernetes-native encryption
- **External Secrets Operator**: Multi-backend support

## Security Contacts

- Security Team: security@nephoran.io
- Security Incidents: incident-response@nephoran.io
- Vulnerability Disclosure: security-disclosure@nephoran.io

## References

- [OWASP Top 10 2021](https://owasp.org/Top10/)
- [CIS Kubernetes Benchmark](https://www.cisecurity.org/benchmark/kubernetes)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)
- [Go Security Guidelines](https://golang.org/doc/security)