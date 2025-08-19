# Security Best Practices Guide

## Executive Summary

This guide provides comprehensive security best practices for developers, operators, and security teams working with the Nephoran Intent Operator. It covers secure coding practices, configuration guidelines, operational procedures, and monitoring requirements to maintain a robust security posture.

## Table of Contents

1. [Development Security](#development-security)
2. [Configuration Security](#configuration-security)
3. [Operational Security](#operational-security)
4. [Network Security](#network-security)
5. [Data Security](#data-security)
6. [Authentication and Authorization](#authentication-and-authorization)
7. [Monitoring and Auditing](#monitoring-and-auditing)
8. [Incident Response](#incident-response)
9. [Security Checklists](#security-checklists)

## Development Security

### Secure Coding Practices

#### 1. Input Validation

**Always validate and sanitize all inputs:**

```go
// GOOD: Comprehensive input validation
func ProcessUserInput(input string) error {
    // Length validation
    if len(input) > MaxInputLength {
        return ErrInputTooLong
    }
    
    // Character validation
    if !regexp.MustCompile(`^[a-zA-Z0-9-._]+$`).MatchString(input) {
        return ErrInvalidCharacters
    }
    
    // Semantic validation
    if err := validateBusinessLogic(input); err != nil {
        return fmt.Errorf("business validation failed: %w", err)
    }
    
    // Sanitize for storage/processing
    sanitized := html.EscapeString(input)
    return processData(sanitized)
}

// BAD: No validation
func ProcessUserInput(input string) error {
    return processData(input) // Direct use without validation
}
```

#### 2. SQL Injection Prevention

**Use parameterized queries exclusively:**

```go
// GOOD: Parameterized query
func GetUser(db *sql.DB, userID string) (*User, error) {
    query := `SELECT id, name, email FROM users WHERE id = $1`
    row := db.QueryRow(query, userID)
    
    var user User
    err := row.Scan(&user.ID, &user.Name, &user.Email)
    return &user, err
}

// BAD: String concatenation
func GetUser(db *sql.DB, userID string) (*User, error) {
    query := fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", userID)
    row := db.QueryRow(query) // SQL injection vulnerability
    // ...
}
```

#### 3. Command Injection Prevention

**Never execute shell commands with user input:**

```go
// GOOD: Use Go libraries instead of shell commands
func CreateDirectory(name string) error {
    // Validate directory name
    if !isValidDirectoryName(name) {
        return ErrInvalidName
    }
    
    // Use Go's os package
    path := filepath.Join(BaseDir, name)
    return os.MkdirAll(path, 0750)
}

// BAD: Shell command execution
func CreateDirectory(name string) error {
    cmd := exec.Command("sh", "-c", fmt.Sprintf("mkdir %s", name))
    return cmd.Run() // Command injection vulnerability
}
```

#### 4. Path Traversal Prevention

**Always sanitize file paths:**

```go
// GOOD: Path sanitization
func ReadFile(filename string) ([]byte, error) {
    // Clean the path
    cleanPath := filepath.Clean(filename)
    
    // Ensure it's within allowed directory
    absPath, err := filepath.Abs(cleanPath)
    if err != nil {
        return nil, err
    }
    
    if !strings.HasPrefix(absPath, AllowedDir) {
        return nil, ErrUnauthorizedPath
    }
    
    return os.ReadFile(absPath)
}

// BAD: Direct path usage
func ReadFile(filename string) ([]byte, error) {
    return os.ReadFile(filename) // Path traversal vulnerability
}
```

#### 5. Error Handling

**Avoid exposing sensitive information in errors:**

```go
// GOOD: Generic error messages for users
func AuthenticateUser(username, password string) error {
    user, err := db.GetUser(username)
    if err != nil {
        log.Error("Database error", "error", err, "user", username)
        return ErrAuthenticationFailed // Generic error
    }
    
    if !checkPassword(user.PasswordHash, password) {
        log.Warn("Invalid password attempt", "user", username)
        return ErrAuthenticationFailed // Same generic error
    }
    
    return nil
}

// BAD: Detailed error information
func AuthenticateUser(username, password string) error {
    user, err := db.GetUser(username)
    if err != nil {
        return fmt.Errorf("user %s not found", username) // Information disclosure
    }
    
    if !checkPassword(user.PasswordHash, password) {
        return fmt.Errorf("invalid password for %s", username) // Information disclosure
    }
    
    return nil
}
```

### Dependency Management

#### 1. Dependency Scanning

```bash
# Regular dependency scanning
# Run daily in CI/CD pipeline

# Go vulnerability scanning
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...

# Nancy for Go modules
go list -json -deps | nancy sleuth

# Snyk scanning
snyk test --all-projects
```

#### 2. Dependency Updates

```yaml
# .github/workflows/dependency-update.yml
name: Dependency Updates
on:
  schedule:
    - cron: '0 0 * * MON' # Weekly on Monday
  
jobs:
  update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Update Go dependencies
        run: |
          go get -u ./...
          go mod tidy
          
      - name: Run tests
        run: go test ./...
        
      - name: Security scan
        run: |
          govulncheck ./...
          gosec ./...
          
      - name: Create PR
        uses: peter-evans/create-pull-request@v5
        with:
          title: "Weekly dependency updates"
          body: "Automated dependency updates with security scanning"
```

### Code Review Security Checklist

```markdown
## Security Review Checklist

### Input Validation
- [ ] All user inputs are validated
- [ ] Length limits are enforced
- [ ] Character sets are restricted
- [ ] Injection patterns are blocked

### Authentication & Authorization
- [ ] Authentication is required for sensitive operations
- [ ] Authorization checks are performed
- [ ] Session management is secure
- [ ] Passwords are properly hashed

### Data Protection
- [ ] Sensitive data is encrypted
- [ ] PII is properly handled
- [ ] Secrets are not hardcoded
- [ ] Data retention policies are followed

### Error Handling
- [ ] Errors don't expose sensitive information
- [ ] Stack traces are not shown to users
- [ ] All errors are logged appropriately
- [ ] Failed operations are rolled back

### Security Headers
- [ ] Content-Security-Policy is set
- [ ] X-Frame-Options is configured
- [ ] Strict-Transport-Security is enabled
- [ ] Other security headers are present
```

## Configuration Security

### 1. Environment Variables

```bash
# GOOD: Secure environment variable usage
# .env.example (committed to repo)
DATABASE_HOST=localhost
DATABASE_PORT=5432
LOG_LEVEL=info

# .env.local (NOT committed, contains secrets)
DATABASE_PASSWORD=${VAULT_DATABASE_PASSWORD}
API_KEY=${VAULT_API_KEY}
JWT_SECRET=${VAULT_JWT_SECRET}

# Load from secure vault
export DATABASE_PASSWORD=$(vault kv get -field=password secret/database)
export API_KEY=$(vault kv get -field=api_key secret/apis)
export JWT_SECRET=$(vault kv get -field=jwt_secret secret/auth)
```

### 2. Configuration Files

```yaml
# config/production.yaml
# GOOD: Secure configuration

server:
  port: 8443
  tls:
    enabled: true
    cert_file: /etc/ssl/certs/server.crt
    key_file: /etc/ssl/private/server.key
    min_version: "1.2"
    cipher_suites:
      - TLS_AES_256_GCM_SHA384
      - TLS_AES_128_GCM_SHA256
      - TLS_CHACHA20_POLY1305_SHA256

security:
  cors:
    enabled: true
    allowed_origins:
      - https://app.nephoran.io
    allowed_methods:
      - GET
      - POST
      - PUT
      - DELETE
    allowed_headers:
      - Authorization
      - Content-Type
    max_age: 3600
  
  rate_limiting:
    enabled: true
    requests_per_second: 10
    burst: 20
    
  headers:
    strict_transport_security: "max-age=31536000; includeSubDomains"
    x_frame_options: "DENY"
    x_content_type_options: "nosniff"
    content_security_policy: "default-src 'self'"

authentication:
  jwt:
    issuer: "https://auth.nephoran.io"
    audience: "https://api.nephoran.io"
    expiration: 15m
    refresh_expiration: 7d
    
  oauth2:
    providers:
      - name: google
        client_id: ${GOOGLE_CLIENT_ID}
        client_secret: ${GOOGLE_CLIENT_SECRET}
        scopes:
          - openid
          - email
          - profile

database:
  host: ${DATABASE_HOST}
  port: 5432
  name: nephoran
  user: ${DATABASE_USER}
  password: ${DATABASE_PASSWORD}
  ssl_mode: require
  max_connections: 25
  connection_timeout: 30s
  
logging:
  level: info
  format: json
  output: stdout
  sensitive_fields:
    - password
    - token
    - secret
    - key
    - authorization
```

### 3. Kubernetes Secrets

```yaml
# GOOD: Using Kubernetes secrets properly
apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
  namespace: production
type: Opaque
data:
  database-password: <base64-encoded>
  api-key: <base64-encoded>
  jwt-secret: <base64-encoded>

---
# Use SecretProviderClass for external secret management
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: vault-secrets
spec:
  provider: vault
  parameters:
    vaultAddress: "https://vault.nephoran.io"
    roleName: "nephoran-app"
    objects: |
      - objectName: "database-password"
        secretPath: "secret/data/database"
        secretKey: "password"
      - objectName: "api-key"
        secretPath: "secret/data/api"
        secretKey: "key"
```

### 4. Network Policies

```yaml
# GOOD: Restrictive network policies
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: nephoran-operator-policy
  namespace: production
spec:
  podSelector:
    matchLabels:
      app: nephoran-operator
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: production
        - podSelector:
            matchLabels:
              app: api-gateway
      ports:
        - protocol: TCP
          port: 8443
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              name: production
        - podSelector:
            matchLabels:
              app: database
      ports:
        - protocol: TCP
          port: 5432
    - to:
        - namespaceSelector:
            matchLabels:
              name: kube-system
        - podSelector:
            matchLabels:
              k8s-app: kube-dns
      ports:
        - protocol: UDP
          port: 53
```

## Operational Security

### 1. Deployment Security

```yaml
# GOOD: Secure deployment configuration
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephoran-operator
  namespace: production
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nephoran-operator
  template:
    metadata:
      labels:
        app: nephoran-operator
      annotations:
        # Force pod restart on config change
        checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
    spec:
      serviceAccountName: nephoran-operator
      securityContext:
        runAsNonRoot: true
        runAsUser: 10001
        fsGroup: 10001
        seccompProfile:
          type: RuntimeDefault
      
      containers:
        - name: operator
          image: nephoran/operator:v1.0.0
          imagePullPolicy: Always
          
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            runAsNonRoot: true
            runAsUser: 10001
            capabilities:
              drop:
                - ALL
              add:
                - NET_BIND_SERVICE
          
          resources:
            requests:
              memory: "256Mi"
              cpu: "250m"
            limits:
              memory: "512Mi"
              cpu: "500m"
          
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
              scheme: HTTPS
            initialDelaySeconds: 10
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
          
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
              scheme: HTTPS
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 3
          
          volumeMounts:
            - name: tmp
              mountPath: /tmp
            - name: cache
              mountPath: /app/cache
            - name: tls-certs
              mountPath: /etc/tls
              readOnly: true
          
          env:
            - name: LOG_LEVEL
              value: "info"
            - name: TLS_CERT_FILE
              value: "/etc/tls/tls.crt"
            - name: TLS_KEY_FILE
              value: "/etc/tls/tls.key"
            - name: DATABASE_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: database-credentials
                  key: password
      
      volumes:
        - name: tmp
          emptyDir: {}
        - name: cache
          emptyDir: {}
        - name: tls-certs
          secret:
            secretName: operator-tls
            defaultMode: 0400
      
      nodeSelector:
        kubernetes.io/os: linux
      
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: app
                    operator: In
                    values:
                      - nephoran-operator
              topologyKey: kubernetes.io/hostname
```

### 2. Access Control

```yaml
# GOOD: RBAC configuration
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: nephoran-operator
  namespace: production
rules:
  - apiGroups: [""]
    resources: ["configmaps", "secrets"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch", "update", "patch"]
  - apiGroups: ["nephoran.io"]
    resources: ["networkintents"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: nephoran-operator
  namespace: production
subjects:
  - kind: ServiceAccount
    name: nephoran-operator
    namespace: production
roleRef:
  kind: Role
  name: nephoran-operator
  apiGroup: rbac.authorization.k8s.io
```

### 3. Backup and Recovery

```bash
#!/bin/bash
# Secure backup script

set -euo pipefail

# Configuration
BACKUP_DIR="/secure/backups"
ENCRYPTION_KEY="${BACKUP_ENCRYPTION_KEY}"
RETENTION_DAYS=30

# Create timestamped backup
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="nephoran_backup_${TIMESTAMP}"

# Backup database
pg_dump \
  --host="${DB_HOST}" \
  --port="${DB_PORT}" \
  --username="${DB_USER}" \
  --dbname="${DB_NAME}" \
  --no-password \
  --format=custom \
  --verbose \
  --file="${BACKUP_DIR}/${BACKUP_NAME}.sql"

# Backup Kubernetes resources
kubectl get all,cm,secret,pvc,pv \
  --all-namespaces \
  -o yaml > "${BACKUP_DIR}/${BACKUP_NAME}_k8s.yaml"

# Encrypt backups
gpg \
  --symmetric \
  --cipher-algo AES256 \
  --batch \
  --passphrase "${ENCRYPTION_KEY}" \
  "${BACKUP_DIR}/${BACKUP_NAME}.sql"

gpg \
  --symmetric \
  --cipher-algo AES256 \
  --batch \
  --passphrase "${ENCRYPTION_KEY}" \
  "${BACKUP_DIR}/${BACKUP_NAME}_k8s.yaml"

# Remove unencrypted files
shred -vfz -n 10 "${BACKUP_DIR}/${BACKUP_NAME}.sql"
shred -vfz -n 10 "${BACKUP_DIR}/${BACKUP_NAME}_k8s.yaml"

# Upload to secure storage
aws s3 cp \
  "${BACKUP_DIR}/${BACKUP_NAME}.sql.gpg" \
  "s3://nephoran-backups/${BACKUP_NAME}.sql.gpg" \
  --server-side-encryption AES256 \
  --storage-class GLACIER

# Clean old backups
find "${BACKUP_DIR}" -name "*.gpg" -mtime +${RETENTION_DAYS} -delete

# Verify backup
if gpg --decrypt --batch --passphrase "${ENCRYPTION_KEY}" \
     "${BACKUP_DIR}/${BACKUP_NAME}.sql.gpg" > /dev/null 2>&1; then
    echo "Backup verified successfully"
else
    echo "Backup verification failed!" >&2
    exit 1
fi
```

## Network Security

### 1. TLS Configuration

```go
// GOOD: Strong TLS configuration
func CreateTLSConfig() *tls.Config {
    return &tls.Config{
        MinVersion: tls.VersionTLS12,
        MaxVersion: tls.VersionTLS13,
        
        CipherSuites: []uint16{
            // TLS 1.3 cipher suites
            tls.TLS_AES_256_GCM_SHA384,
            tls.TLS_AES_128_GCM_SHA256,
            tls.TLS_CHACHA20_POLY1305_SHA256,
            
            // TLS 1.2 cipher suites (for compatibility)
            tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
            tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
            tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
        },
        
        PreferServerCipherSuites: true,
        
        CurvePreferences: []tls.CurveID{
            tls.X25519,
            tls.CurveP256,
        },
        
        SessionTicketsDisabled: false,
        
        ClientAuth: tls.RequireAndVerifyClientCert,
        ClientCAs:  loadCAPool(),
        
        GetCertificate: getCertificateFunc(),
        
        VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
            // Additional certificate validation
            return validateCertificate(rawCerts, verifiedChains)
        },
    }
}
```

### 2. Firewall Rules

```bash
# iptables configuration for production

# Default policies
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT

# Allow loopback
iptables -A INPUT -i lo -j ACCEPT
iptables -A OUTPUT -o lo -j ACCEPT

# Allow established connections
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT

# Allow SSH from bastion only
iptables -A INPUT -p tcp --dport 22 -s 10.0.1.10/32 -j ACCEPT

# Allow HTTPS
iptables -A INPUT -p tcp --dport 443 -m conntrack --ctstate NEW -j ACCEPT

# Allow Kubernetes API from internal network
iptables -A INPUT -p tcp --dport 6443 -s 10.0.0.0/16 -j ACCEPT

# Rate limiting for HTTPS
iptables -A INPUT -p tcp --dport 443 -m conntrack --ctstate NEW -m recent --set
iptables -A INPUT -p tcp --dport 443 -m conntrack --ctstate NEW -m recent --update --seconds 60 --hitcount 20 -j DROP

# DDoS protection
iptables -A INPUT -p tcp --dport 443 -m conntrack --ctstate NEW -m limit --limit 100/second --limit-burst 200 -j ACCEPT

# Log dropped packets
iptables -A INPUT -m limit --limit 5/min -j LOG --log-prefix "iptables-dropped: " --log-level 4

# Save rules
iptables-save > /etc/iptables/rules.v4
```

### 3. Service Mesh Security

```yaml
# Istio security configuration
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: production
spec:
  mtls:
    mode: STRICT

---
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: nephoran-operator
  namespace: production
spec:
  selector:
    matchLabels:
      app: nephoran-operator
  action: ALLOW
  rules:
    - from:
        - source:
            principals: ["cluster.local/ns/production/sa/api-gateway"]
      to:
        - operation:
            methods: ["GET", "POST", "PUT", "DELETE"]
            paths: ["/api/*"]
      when:
        - key: request.headers[authorization]
          values: ["Bearer *"]
```

## Data Security

### 1. Encryption at Rest

```go
// GOOD: Proper encryption implementation
package encryption

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "io"
)

type Encryptor struct {
    key []byte
}

func NewEncryptor(key []byte) (*Encryptor, error) {
    if len(key) != 32 {
        return nil, errors.New("key must be 32 bytes for AES-256")
    }
    return &Encryptor{key: key}, nil
}

func (e *Encryptor) Encrypt(plaintext []byte) (string, error) {
    // Create cipher block
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return "", err
    }
    
    // Create GCM
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    // Generate nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    // Encrypt and append nonce
    ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
    
    // Encode to base64 for storage
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *Encryptor) Decrypt(ciphertext string) ([]byte, error) {
    // Decode from base64
    data, err := base64.StdEncoding.DecodeString(ciphertext)
    if err != nil {
        return nil, err
    }
    
    // Create cipher block
    block, err := aes.NewCipher(e.key)
    if err != nil {
        return nil, err
    }
    
    // Create GCM
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    
    // Extract nonce
    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return nil, errors.New("ciphertext too short")
    }
    
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    
    // Decrypt
    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, err
    }
    
    return plaintext, nil
}
```

### 2. Data Classification and Handling

```go
// Data classification system
type DataClassification int

const (
    Public DataClassification = iota
    Internal
    Confidential
    Restricted
)

type DataHandler struct {
    classification DataClassification
    encryptor      *Encryptor
    logger         *log.Logger
}

func (dh *DataHandler) ProcessData(data []byte, metadata map[string]string) error {
    // Audit log
    dh.logger.Printf("Processing data: classification=%d, size=%d", 
        dh.classification, len(data))
    
    switch dh.classification {
    case Public:
        return dh.processPublicData(data, metadata)
    case Internal:
        return dh.processInternalData(data, metadata)
    case Confidential:
        return dh.processConfidentialData(data, metadata)
    case Restricted:
        return dh.processRestrictedData(data, metadata)
    default:
        return errors.New("unknown classification")
    }
}

func (dh *DataHandler) processRestrictedData(data []byte, metadata map[string]string) error {
    // Encrypt data
    encrypted, err := dh.encryptor.Encrypt(data)
    if err != nil {
        return fmt.Errorf("encryption failed: %w", err)
    }
    
    // Store with strict access controls
    if err := storeWithHSM(encrypted, metadata); err != nil {
        return fmt.Errorf("storage failed: %w", err)
    }
    
    // Audit trail
    auditLog := AuditLog{
        Timestamp:      time.Now(),
        Action:         "STORE_RESTRICTED_DATA",
        Classification: "RESTRICTED",
        User:           getCurrentUser(),
        Metadata:       metadata,
    }
    
    return dh.logger.LogAudit(auditLog)
}
```

## Authentication and Authorization

### 1. Multi-Factor Authentication

```go
// MFA implementation
type MFAManager struct {
    totpSecret string
    smsProvider SMSProvider
    emailProvider EmailProvider
}

func (m *MFAManager) VerifyMFA(user *User, token string, method MFAMethod) error {
    switch method {
    case TOTP:
        return m.verifyTOTP(user, token)
    case SMS:
        return m.verifySMS(user, token)
    case Email:
        return m.verifyEmail(user, token)
    default:
        return ErrInvalidMFAMethod
    }
}

func (m *MFAManager) verifyTOTP(user *User, token string) error {
    // Validate TOTP token
    valid := totp.Validate(token, user.TOTPSecret)
    if !valid {
        // Log failed attempt
        log.Warn("Failed TOTP verification", 
            "user", user.ID,
            "ip", getClientIP())
        return ErrInvalidTOTPToken
    }
    
    // Prevent replay attacks
    if m.isTokenUsed(user.ID, token) {
        log.Error("TOTP token replay attempt",
            "user", user.ID,
            "token", token)
        return ErrTokenReplay
    }
    
    // Mark token as used
    m.markTokenUsed(user.ID, token)
    
    return nil
}
```

### 2. OAuth2/OIDC Implementation

```go
// OAuth2 configuration
type OAuth2Config struct {
    Provider     string
    ClientID     string
    ClientSecret string
    RedirectURL  string
    Scopes       []string
}

func (c *OAuth2Config) Exchange(code string) (*oauth2.Token, error) {
    // Exchange authorization code for token
    ctx := context.WithTimeout(context.Background(), 10*time.Second)
    defer ctx.Done()
    
    token, err := c.config.Exchange(ctx, code)
    if err != nil {
        return nil, fmt.Errorf("token exchange failed: %w", err)
    }
    
    // Validate token
    if !token.Valid() {
        return nil, errors.New("invalid token received")
    }
    
    // Verify token signature
    if err := c.verifyTokenSignature(token); err != nil {
        return nil, fmt.Errorf("token signature verification failed: %w", err)
    }
    
    return token, nil
}

func (c *OAuth2Config) verifyTokenSignature(token *oauth2.Token) error {
    // Get ID token
    idToken, ok := token.Extra("id_token").(string)
    if !ok {
        return errors.New("id_token not found")
    }
    
    // Verify with provider's public key
    verifier := c.provider.Verifier(&oidc.Config{
        ClientID: c.ClientID,
    })
    
    _, err := verifier.Verify(context.Background(), idToken)
    return err
}
```

## Monitoring and Auditing

### 1. Security Event Monitoring

```yaml
# Prometheus alert rules for security monitoring
groups:
  - name: security_alerts
    interval: 30s
    rules:
      - alert: HighFailedLoginRate
        expr: rate(authentication_failures_total[5m]) > 10
        for: 2m
        labels:
          severity: warning
          team: security
        annotations:
          summary: High rate of failed login attempts
          description: "{{ $value }} failed logins per second"
      
      - alert: UnauthorizedAPIAccess
        expr: rate(authorization_failures_total[5m]) > 5
        for: 1m
        labels:
          severity: critical
          team: security
        annotations:
          summary: Multiple unauthorized API access attempts
          description: "{{ $value }} unauthorized attempts per second"
      
      - alert: SuspiciousDataAccess
        expr: rate(data_access_anomaly_total[10m]) > 0
        for: 1m
        labels:
          severity: critical
          team: security
        annotations:
          summary: Suspicious data access pattern detected
          description: "Anomalous data access from {{ $labels.source_ip }}"
      
      - alert: CertificateExpiry
        expr: ssl_cert_expiry_days < 30
        for: 1h
        labels:
          severity: warning
          team: security
        annotations:
          summary: SSL certificate expiring soon
          description: "Certificate {{ $labels.cn }} expires in {{ $value }} days"
```

### 2. Audit Logging

```go
// Comprehensive audit logging
type AuditLogger struct {
    output io.Writer
    encoder *json.Encoder
    signer  crypto.Signer
}

type AuditEvent struct {
    ID            string                 `json:"id"`
    Timestamp     time.Time              `json:"timestamp"`
    Actor         Actor                  `json:"actor"`
    Action        string                 `json:"action"`
    Resource      Resource               `json:"resource"`
    Result        string                 `json:"result"`
    SourceIP      string                 `json:"source_ip"`
    UserAgent     string                 `json:"user_agent"`
    RequestID     string                 `json:"request_id"`
    SessionID     string                 `json:"session_id"`
    Metadata      map[string]interface{} `json:"metadata,omitempty"`
    Signature     string                 `json:"signature"`
}

func (al *AuditLogger) Log(event AuditEvent) error {
    // Set timestamp and ID
    event.Timestamp = time.Now().UTC()
    event.ID = uuid.New().String()
    
    // Sign the event for integrity
    signature, err := al.signEvent(event)
    if err != nil {
        return fmt.Errorf("failed to sign event: %w", err)
    }
    event.Signature = signature
    
    // Encode and write
    if err := al.encoder.Encode(event); err != nil {
        return fmt.Errorf("failed to encode event: %w", err)
    }
    
    // Force flush for critical events
    if event.Result == "SECURITY_VIOLATION" {
        if flusher, ok := al.output.(interface{ Flush() error }); ok {
            return flusher.Flush()
        }
    }
    
    return nil
}

func (al *AuditLogger) signEvent(event AuditEvent) (string, error) {
    // Create canonical representation
    canonical, err := json.Marshal(event)
    if err != nil {
        return "", err
    }
    
    // Sign with private key
    hash := sha256.Sum256(canonical)
    signature, err := al.signer.Sign(rand.Reader, hash[:], crypto.SHA256)
    if err != nil {
        return "", err
    }
    
    return base64.StdEncoding.EncodeToString(signature), nil
}
```

## Incident Response

### 1. Incident Detection

```go
// Automated incident detection
type IncidentDetector struct {
    rules      []DetectionRule
    aggregator *EventAggregator
    notifier   *IncidentNotifier
}

func (id *IncidentDetector) ProcessEvent(event SecurityEvent) {
    // Check against detection rules
    for _, rule := range id.rules {
        if rule.Matches(event) {
            id.handlePotentialIncident(rule, event)
        }
    }
    
    // Aggregate for pattern detection
    id.aggregator.Add(event)
    if pattern := id.aggregator.DetectPattern(); pattern != nil {
        id.handleDetectedPattern(pattern)
    }
}

func (id *IncidentDetector) handlePotentialIncident(rule DetectionRule, event SecurityEvent) {
    incident := Incident{
        ID:          uuid.New().String(),
        Timestamp:   time.Now(),
        Severity:    rule.Severity,
        Type:        rule.IncidentType,
        Description: rule.Description,
        Events:      []SecurityEvent{event},
        Status:      "DETECTED",
    }
    
    // Immediate response for critical incidents
    if incident.Severity == Critical {
        id.executeImmediateResponse(incident)
    }
    
    // Notify security team
    id.notifier.Notify(incident)
    
    // Log incident
    log.Error("Security incident detected",
        "incident_id", incident.ID,
        "type", incident.Type,
        "severity", incident.Severity)
}
```

### 2. Incident Response Automation

```go
// Automated incident response
type IncidentResponder struct {
    playbooks map[string]Playbook
    executor  *ActionExecutor
}

func (ir *IncidentResponder) Respond(incident Incident) error {
    // Get appropriate playbook
    playbook, exists := ir.playbooks[incident.Type]
    if !exists {
        return fmt.Errorf("no playbook for incident type: %s", incident.Type)
    }
    
    // Execute playbook actions
    for _, action := range playbook.Actions {
        if err := ir.executeAction(action, incident); err != nil {
            log.Error("Failed to execute action",
                "action", action.Name,
                "incident", incident.ID,
                "error", err)
            
            // Continue with other actions unless critical
            if action.Critical {
                return err
            }
        }
    }
    
    return nil
}

func (ir *IncidentResponder) executeAction(action Action, incident Incident) error {
    switch action.Type {
    case "ISOLATE_POD":
        return ir.isolatePod(action.Target)
    case "BLOCK_IP":
        return ir.blockIP(action.Target)
    case "REVOKE_TOKEN":
        return ir.revokeToken(action.Target)
    case "SNAPSHOT_SYSTEM":
        return ir.createForensicSnapshot()
    case "NOTIFY_TEAM":
        return ir.notifyTeam(action.Target, incident)
    default:
        return fmt.Errorf("unknown action type: %s", action.Type)
    }
}
```

## Security Checklists

### Daily Security Checklist

```markdown
## Daily Security Tasks

### Morning Review (9:00 AM)
- [ ] Review overnight security alerts
- [ ] Check failed authentication attempts
- [ ] Verify backup completion
- [ ] Review system health metrics
- [ ] Check certificate expiration dates

### Midday Check (1:00 PM)
- [ ] Monitor current threat landscape
- [ ] Review access logs for anomalies
- [ ] Verify security patches status
- [ ] Check rate limiting effectiveness
- [ ] Review DDoS protection metrics

### End of Day (5:00 PM)
- [ ] Review daily security events summary
- [ ] Update incident tracker
- [ ] Verify all critical alerts addressed
- [ ] Check compliance dashboard
- [ ] Document any security concerns
```

### Weekly Security Checklist

```markdown
## Weekly Security Tasks

### System Security
- [ ] Run vulnerability scans
- [ ] Review and apply security patches
- [ ] Update threat intelligence feeds
- [ ] Test backup restoration
- [ ] Review firewall rules

### Access Management
- [ ] Review user access reports
- [ ] Audit privileged accounts
- [ ] Check for orphaned accounts
- [ ] Review API key usage
- [ ] Validate MFA enrollment

### Compliance
- [ ] Generate compliance reports
- [ ] Review audit logs
- [ ] Update risk register
- [ ] Check policy violations
- [ ] Document security metrics

### Testing
- [ ] Run penetration tests
- [ ] Test incident response
- [ ] Verify security controls
- [ ] Test disaster recovery
- [ ] Validate monitoring alerts
```

### Deployment Security Checklist

```markdown
## Pre-Deployment Security Checklist

### Code Security
- [ ] Static code analysis completed
- [ ] Dependency vulnerabilities scanned
- [ ] Security code review completed
- [ ] Sensitive data not hardcoded
- [ ] Input validation implemented

### Configuration
- [ ] TLS properly configured
- [ ] Security headers enabled
- [ ] Rate limiting configured
- [ ] Network policies defined
- [ ] RBAC rules reviewed

### Infrastructure
- [ ] Container images scanned
- [ ] Security contexts defined
- [ ] Resource limits set
- [ ] Pod security policies applied
- [ ] Service mesh policies configured

### Monitoring
- [ ] Security alerts configured
- [ ] Audit logging enabled
- [ ] Metrics collection active
- [ ] Log aggregation working
- [ ] Incident response tested

### Documentation
- [ ] Security documentation updated
- [ ] Runbooks current
- [ ] Contact lists verified
- [ ] Recovery procedures documented
- [ ] Compliance requirements met
```

## Security Tools and Resources

### Essential Security Tools

```bash
# Code Security
- gosec: Go security checker
- semgrep: Static analysis
- snyk: Vulnerability scanning
- trivy: Container scanning
- gitleaks: Secret detection

# Runtime Security
- falco: Runtime security
- osquery: System monitoring
- sysdig: Container security
- tracee: Runtime detection

# Network Security
- nmap: Network scanning
- wireshark: Packet analysis
- metasploit: Penetration testing
- zap: Web app security

# Compliance
- openscap: Compliance scanning
- chef inspec: Compliance automation
- cloud custodian: Cloud compliance
```

### Security Training Resources

1. **OWASP Resources**
   - OWASP Top 10
   - OWASP Cheat Sheets
   - OWASP Testing Guide

2. **Cloud Security**
   - AWS Security Best Practices
   - GCP Security Command Center
   - Azure Security Center

3. **Kubernetes Security**
   - CIS Kubernetes Benchmark
   - NSA Kubernetes Hardening Guide
   - CNCF Security Whitepaper

4. **General Security**
   - SANS Security Resources
   - NIST Cybersecurity Framework
   - MITRE ATT&CK Framework

## Conclusion

Security is not a one-time implementation but an ongoing process requiring continuous attention, updates, and improvements. This guide provides comprehensive best practices that should be regularly reviewed and updated based on emerging threats and lessons learned.

Remember:
- Security is everyone's responsibility
- Defense in depth is essential
- Regular training and awareness are crucial
- Automate security where possible
- Always verify and never trust

Stay vigilant, stay secure.

---

**Document Classification**: INTERNAL USE ONLY  
**Last Updated**: 2025-08-19  
**Version**: 1.0.0  
**Review Cycle**: Quarterly  
**Owner**: Security Team