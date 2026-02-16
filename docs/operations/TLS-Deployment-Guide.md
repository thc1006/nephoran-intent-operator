# TLS Deployment Guide for Nephoran Intent Operator LLM Processor

## Overview

This guide provides comprehensive instructions for deploying the Nephoran Intent Operator LLM Processor with TLS encryption. The LLM Processor service includes built-in TLS support with automatic environment detection, certificate validation, and secure configuration management.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [TLS Configuration Overview](#tls-configuration-overview)
3. [Certificate Generation](#certificate-generation)
4. [Deployment Scenarios](#deployment-scenarios)
5. [Security Best Practices](#security-best-practices)
6. [Monitoring and Maintenance](#monitoring-and-maintenance)
7. [Troubleshooting](#troubleshooting)
8. [Operational Runbooks](#operational-runbooks)

## Prerequisites

### System Requirements

- **Kubernetes**: v1.25+ with TLS support
- **kubectl**: v1.25+ configured for your cluster
- **OpenSSL**: v1.1.1+ for certificate operations
- **cert-manager**: v1.13+ (for automated certificate management)

### Networking Requirements

- **DNS Resolution**: Internal cluster DNS for service discovery
- **Load Balancer**: Support for TLS termination (if using LoadBalancer service type)
- **Ingress Controller**: NGINX or compatible ingress with TLS support
- **Firewall**: Port 443 (HTTPS) accessible for external access

### Security Prerequisites

- **CA Infrastructure**: Access to Certificate Authority for production certificates
- **Secret Management**: Kubernetes RBAC permissions for secret management
- **Network Policies**: Cluster support for NetworkPolicy resources

## TLS Configuration Overview

### Environment Variables

The LLM Processor service supports comprehensive TLS configuration through environment variables:

```bash
# Core TLS Configuration
TLS_ENABLED="true"                      # Enable TLS encryption (default: false)
TLS_CERT_PATH="/etc/certs/tls.crt"     # Path to TLS certificate file
TLS_KEY_PATH="/etc/certs/tls.key"      # Path to TLS private key file

# Environment Detection (for security validation)
GO_ENV="production"                     # Environment indicator for security policies
LLM_ENVIRONMENT="production"            # Production environment indicator
ENVIRONMENT="production"                # General environment variable

# Service Configuration
PORT="8080"                            # HTTPS port (same as HTTP port)
REQUEST_TIMEOUT="30s"                  # Request timeout for HTTPS requests
```

### Security Features

The TLS implementation includes several security features:

- **Environment Detection**: Automatically detects development vs. production environments
- **Certificate Validation**: Validates certificate file existence and permissions
- **Security Warnings**: Issues warnings when TLS is disabled in production
- **Graceful Degradation**: Falls back to HTTP with security warnings if TLS fails
- **Health Check Integration**: HTTPS-aware health and readiness probes

### Configuration Validation

The service performs comprehensive validation during startup:

1. **File Existence**: Verifies certificate and key files exist
2. **File Permissions**: Ensures appropriate file permissions (600 for private key)
3. **Certificate Validity**: Basic certificate format validation
4. **Environment Consistency**: Validates security settings for environment type

## Certificate Generation

### 1. Self-Signed Certificates (Development/Testing)

For development and testing environments, generate self-signed certificates with proper Subject Alternative Names (SANs):

#### Automated Certificate Generation Script

```bash
#!/bin/bash
# scripts/generate-dev-certificates.sh
# Generate self-signed certificates for development use

set -e

CERT_DIR="${CERT_DIR:-./certs}"
NAMESPACE="${NAMESPACE:-nephoran-system}"
SERVICE_NAME="${SERVICE_NAME:-llm-processor}"
DAYS_VALID="${DAYS_VALID:-365}"

# Create certificate directory
mkdir -p "$CERT_DIR"

echo "=== Generating Development TLS Certificates ==="
echo "Certificate directory: $CERT_DIR"
echo "Service name: $SERVICE_NAME"
echo "Namespace: $NAMESPACE"
echo "Validity period: $DAYS_VALID days"

# Generate private key
echo "1. Generating private key..."
openssl genrsa -out "$CERT_DIR/tls.key" 2048

# Create certificate configuration
echo "2. Creating certificate configuration..."
cat > "$CERT_DIR/cert.conf" <<EOF
[req]
default_bits = 2048
prompt = no
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]
C=US
ST=California
L=San Francisco
O=Nephoran Technologies
OU=Intent Operator Development
CN=$SERVICE_NAME
emailAddress=dev@nephoran.com

[v3_req]
keyUsage = critical, digitalSignature, keyEncipherment
extendedKeyUsage = critical, serverAuth, clientAuth
subjectAltName = @alt_names
basicConstraints = critical, CA:false

[alt_names]
DNS.1 = $SERVICE_NAME
DNS.2 = $SERVICE_NAME.$NAMESPACE
DNS.3 = $SERVICE_NAME.$NAMESPACE.svc
DNS.4 = $SERVICE_NAME.$NAMESPACE.svc.cluster.local
DNS.5 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# Generate certificate
echo "3. Generating self-signed certificate..."
openssl req -new -x509 \
    -key "$CERT_DIR/tls.key" \
    -out "$CERT_DIR/tls.crt" \
    -days $DAYS_VALID \
    -config "$CERT_DIR/cert.conf" \
    -extensions v3_req

# Set secure permissions
echo "4. Setting secure file permissions..."
chmod 600 "$CERT_DIR/tls.key"
chmod 644 "$CERT_DIR/tls.crt"

# Verify certificate
echo "5. Verifying certificate..."
echo "Certificate Details:"
openssl x509 -in "$CERT_DIR/tls.crt" -text -noout | \
    grep -E "(Subject:|DNS:|IP Address:|Not After:|Signature Algorithm:)"

echo "Subject Alternative Names:"
openssl x509 -in "$CERT_DIR/tls.crt" -text -noout | \
    grep -A 10 "Subject Alternative Name" | head -10

# Create Kubernetes secret manifest
echo "6. Creating Kubernetes secret manifest..."
kubectl create secret tls "$SERVICE_NAME-tls" \
    --cert="$CERT_DIR/tls.crt" \
    --key="$CERT_DIR/tls.key" \
    --namespace="$NAMESPACE" \
    --dry-run=client -o yaml > "$CERT_DIR/tls-secret.yaml"

echo "✅ Certificate generation completed successfully!"
echo ""
echo "Next steps:"
echo "1. Apply the secret: kubectl apply -f $CERT_DIR/tls-secret.yaml"
echo "2. Update deployment with TLS_ENABLED=true"
echo "3. Mount certificates in the container"
echo "4. Test HTTPS connectivity"

# Optional: Create complete deployment example
echo "7. Creating complete deployment example..."
cat > "$CERT_DIR/deployment-example.yaml" <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $SERVICE_NAME
  namespace: $NAMESPACE
spec:
  replicas: 1
  selector:
    matchLabels:
      app: $SERVICE_NAME
  template:
    metadata:
      labels:
        app: $SERVICE_NAME
    spec:
      containers:
      - name: $SERVICE_NAME
        image: nephoran/llm-processor:latest
        env:
        - name: TLS_ENABLED
          value: "true"
        - name: TLS_CERT_PATH
          value: "/etc/certs/tls.crt"
        - name: TLS_KEY_PATH
          value: "/etc/certs/tls.key"
        - name: GO_ENV
          value: "development"
        - name: PORT
          value: "8080"
        ports:
        - containerPort: 8080
          name: https
          protocol: TCP
        volumeMounts:
        - name: tls-certs
          mountPath: /etc/certs
          readOnly: true
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
            scheme: HTTPS
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
            scheme: HTTPS
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: tls-certs
        secret:
          secretName: $SERVICE_NAME-tls
          defaultMode: 0600
---
apiVersion: v1
kind: Service
metadata:
  name: $SERVICE_NAME
  namespace: $NAMESPACE
spec:
  type: ClusterIP
  ports:
  - port: 443
    targetPort: 8080
    protocol: TCP
    name: https
  selector:
    app: $SERVICE_NAME
EOF

echo "Deployment example created: $CERT_DIR/deployment-example.yaml"
```

#### Certificate Testing

```bash
#!/bin/bash
# scripts/test-certificate.sh
# Test generated certificates

CERT_FILE="${1:-./certs/tls.crt}"
KEY_FILE="${2:-./certs/tls.key}"

echo "=== Certificate Testing Suite ==="

# Test 1: Certificate and key match
echo "1. Testing certificate and key pair..."
CERT_MODULUS=$(openssl x509 -noout -modulus -in "$CERT_FILE" | openssl md5)
KEY_MODULUS=$(openssl rsa -noout -modulus -in "$KEY_FILE" | openssl md5)

if [ "$CERT_MODULUS" = "$KEY_MODULUS" ]; then
    echo "✅ Certificate and private key match"
else
    echo "❌ Certificate and private key do not match"
    exit 1
fi

# Test 2: Certificate validity period
echo "2. Checking certificate validity..."
if openssl x509 -checkend 86400 -noout -in "$CERT_FILE" >/dev/null; then
    echo "✅ Certificate is valid for at least 24 hours"
else
    echo "⚠️ Certificate expires within 24 hours"
fi

# Test 3: SAN validation
echo "3. Validating Subject Alternative Names..."
SANS=$(openssl x509 -text -noout -in "$CERT_FILE" | grep -A1 "Subject Alternative Name" | tail -1)
echo "SANs: $SANS"

# Test 4: Certificate chain validation
echo "4. Testing certificate properties..."
openssl x509 -text -noout -in "$CERT_FILE" | grep -E "(Subject:|Issuer:|Not After:|Key Usage:|Extended Key Usage:)"

echo "✅ Certificate testing completed"
```

### 2. Let's Encrypt Certificates (Production)

For production deployments, use cert-manager with Let's Encrypt:

#### cert-manager Installation

```bash
#!/bin/bash
# scripts/install-cert-manager.sh
# Install cert-manager for automated certificate management

set -e

CERT_MANAGER_VERSION="v1.13.0"

echo "=== Installing cert-manager $CERT_MANAGER_VERSION ==="

# Install cert-manager
kubectl apply -f "https://github.com/cert-manager/cert-manager/releases/download/$CERT_MANAGER_VERSION/cert-manager.yaml"

# Wait for deployment
echo "Waiting for cert-manager to be ready..."
kubectl wait --for=condition=available \
    --timeout=300s \
    deployment/cert-manager \
    -n cert-manager

kubectl wait --for=condition=available \
    --timeout=300s \
    deployment/cert-manager-cainjector \
    -n cert-manager

kubectl wait --for=condition=available \
    --timeout=300s \
    deployment/cert-manager-webhook \
    -n cert-manager

echo "✅ cert-manager installation completed"
```

#### Let's Encrypt ClusterIssuer Configuration

```yaml
# deployments/security/letsencrypt-clusterissuer.yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
  labels:
    app.kubernetes.io/name: cert-manager
    app.kubernetes.io/component: clusterissuer
spec:
  acme:
    # Production ACME server
    server: https://acme-v02.api.letsencrypt.org/directory
    email: operations@nephoran.com  # Replace with your email
    privateKeySecretRef:
      name: letsencrypt-prod-private-key
    solvers:
    # HTTP-01 challenge solver
    - http01:
        ingress:
          class: nginx
          podTemplate:
            metadata:
              annotations:
                cluster-autoscaler.kubernetes.io/safe-to-evict: "false"
    # DNS-01 challenge solver (example with Cloudflare)
    - dns01:
        cloudflare:
          email: operations@nephoran.com
          apiTokenSecretRef:
            name: cloudflare-api-token-secret
            key: api-token
      selector:
        dnsZones:
        - "nephoran.com"
        - "api.nephoran.com"
---
# Staging issuer for testing
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
  labels:
    app.kubernetes.io/name: cert-manager
    app.kubernetes.io/component: clusterissuer
spec:
  acme:
    # Staging ACME server for testing
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    email: operations@nephoran.com
    privateKeySecretRef:
      name: letsencrypt-staging-private-key
    solvers:
    - http01:
        ingress:
          class: nginx
```

#### Certificate Resource Configuration

```yaml
# deployments/security/llm-processor-certificate.yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: llm-processor-tls
  namespace: nephoran-system
  labels:
    app.kubernetes.io/name: llm-processor
    app.kubernetes.io/component: certificate
spec:
  # Secret name where certificate will be stored
  secretName: llm-processor-tls-secret
  
  # Certificate issuer reference
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
    group: cert-manager.io
  
  # Certificate common name and SANs
  commonName: llm-processor.api.nephoran.com
  dnsNames:
  - llm-processor.api.nephoran.com
  - llm.api.nephoran.com
  - api.nephoran.com
  
  # Certificate configuration
  duration: 2160h  # 90 days
  renewBefore: 360h # 15 days before expiry
  
  # Key configuration
  privateKey:
    algorithm: RSA
    encoding: PKCS1
    size: 2048
    rotationPolicy: Always
  
  # Certificate usage
  usages:
  - digital signature
  - key encipherment
  - server auth
  
  # Additional configuration
  subject:
    organizations:
    - Nephoran Technologies
    organizationalUnits:
    - Intent Operator
    countries:
    - US
    localities:
    - San Francisco
    provinces:
    - California
```

### 3. Enterprise CA Certificates

For enterprise environments with internal Certificate Authorities:

#### Enterprise Certificate Setup

```bash
#!/bin/bash
# scripts/setup-enterprise-certificates.sh
# Setup certificates from enterprise CA

set -e

NAMESPACE="${NAMESPACE:-nephoran-system}"
SERVICE_NAME="${SERVICE_NAME:-llm-processor}"
CA_CERT_FILE="${CA_CERT_FILE:-/path/to/enterprise-ca.crt}"
CERT_FILE="${CERT_FILE:-/path/to/llm-processor.crt}"
KEY_FILE="${KEY_FILE:-/path/to/llm-processor.key}"

echo "=== Enterprise Certificate Setup ==="

# Validate certificate files exist
if [[ ! -f "$CA_CERT_FILE" ]]; then
    echo "❌ CA certificate not found: $CA_CERT_FILE"
    exit 1
fi

if [[ ! -f "$CERT_FILE" ]]; then
    echo "❌ Service certificate not found: $CERT_FILE"
    exit 1
fi

if [[ ! -f "$KEY_FILE" ]]; then
    echo "❌ Private key not found: $KEY_FILE"
    exit 1
fi

# Verify certificate chain
echo "1. Verifying certificate chain..."
if openssl verify -CAfile "$CA_CERT_FILE" "$CERT_FILE" >/dev/null 2>&1; then
    echo "✅ Certificate chain verification successful"
else
    echo "❌ Certificate chain verification failed"
    exit 1
fi

# Check certificate validity
echo "2. Checking certificate validity..."
EXPIRY_DATE=$(openssl x509 -enddate -noout -in "$CERT_FILE" | cut -d= -f2)
echo "Certificate expires: $EXPIRY_DATE"

# Create CA secret
echo "3. Creating CA certificate secret..."
kubectl create secret generic "$SERVICE_NAME-ca" \
    --from-file=ca.crt="$CA_CERT_FILE" \
    --namespace="$NAMESPACE" \
    --dry-run=client -o yaml | kubectl apply -f -

# Create TLS secret
echo "4. Creating TLS certificate secret..."
kubectl create secret tls "$SERVICE_NAME-tls" \
    --cert="$CERT_FILE" \
    --key="$KEY_FILE" \
    --namespace="$NAMESPACE" \
    --dry-run=client -o yaml | kubectl apply -f -

echo "✅ Enterprise certificate setup completed"
```

## Deployment Scenarios

### Scenario 1: Development Environment

Development deployment with self-signed certificates and relaxed security:

```yaml
# deployments/environments/development-tls.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: nephoran-dev
  labels:
    environment: development
    security-policy: relaxed
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-processor
  namespace: nephoran-dev
  labels:
    app: llm-processor
    environment: development
spec:
  replicas: 1
  selector:
    matchLabels:
      app: llm-processor
  template:
    metadata:
      labels:
        app: llm-processor
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
        prometheus.io/scheme: "https"
    spec:
      containers:
      - name: llm-processor
        image: nephoran/llm-processor:dev
        imagePullPolicy: Always
        env:
        # TLS Configuration
        - name: TLS_ENABLED
          value: "true"
        - name: TLS_CERT_PATH
          value: "/etc/certs/tls.crt"
        - name: TLS_KEY_PATH
          value: "/etc/certs/tls.key"
        # Environment Detection
        - name: GO_ENV
          value: "development"
        - name: ENVIRONMENT
          value: "development"
        # Service Configuration
        - name: PORT
          value: "8080"
        - name: LOG_LEVEL
          value: "debug"
        # Security (relaxed for development)
        - name: AUTH_ENABLED
          value: "false"
        - name: REQUIRE_AUTH
          value: "false"
        # API Keys from secrets
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: openai-dev-credentials
              key: api-key
        ports:
        - containerPort: 8080
          name: https
          protocol: TCP
        # Health checks with HTTPS
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
            scheme: HTTPS
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
            scheme: HTTPS
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        # Resource limits (development)
        resources:
          requests:
            cpu: 250m
            memory: 512Mi
          limits:
            cpu: 1000m
            memory: 2Gi
        # Volume mounts
        volumeMounts:
        - name: tls-certs
          mountPath: /etc/certs
          readOnly: true
      volumes:
      - name: tls-certs
        secret:
          secretName: llm-processor-tls
          defaultMode: 0600
---
apiVersion: v1
kind: Service
metadata:
  name: llm-processor
  namespace: nephoran-dev
  labels:
    app: llm-processor
spec:
  type: ClusterIP
  ports:
  - port: 443
    targetPort: 8080
    protocol: TCP
    name: https
  - port: 80
    targetPort: 8080
    protocol: TCP
    name: http-redirect
  selector:
    app: llm-processor
```

### Scenario 2: Production Environment

Production deployment with enterprise-grade security:

```yaml
# deployments/environments/production-tls.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-processor
  namespace: nephoran-system
  labels:
    app: llm-processor
    environment: production
    version: v2.0.0
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  selector:
    matchLabels:
      app: llm-processor
  template:
    metadata:
      labels:
        app: llm-processor
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
        prometheus.io/scheme: "https"
        # Security annotations
        container.apparmor.security.beta.kubernetes.io/llm-processor: runtime/default
    spec:
      # Security context
      securityContext:
        runAsNonRoot: true
        runAsUser: 65534
        runAsGroup: 65534
        fsGroup: 65534
        seccompProfile:
          type: RuntimeDefault
      
      containers:
      - name: llm-processor
        image: nephoran/llm-processor:v2.0.0
        imagePullPolicy: Always
        
        # Environment configuration
        env:
        # TLS Configuration
        - name: TLS_ENABLED
          value: "true"
        - name: TLS_CERT_PATH
          value: "/etc/certs/tls.crt"
        - name: TLS_KEY_PATH
          value: "/etc/certs/tls.key"
        # Environment Detection
        - name: GO_ENV
          value: "production"
        - name: LLM_ENVIRONMENT
          value: "production"
        - name: ENVIRONMENT
          value: "production"
        # Service Configuration
        - name: PORT
          value: "8080"
        - name: LOG_LEVEL
          value: "info"
        - name: REQUEST_TIMEOUT
          value: "30s"
        # Security (production)
        - name: AUTH_ENABLED
          value: "true"
        - name: REQUIRE_AUTH
          value: "true"
        - name: CORS_ENABLED
          value: "true"
        - name: LLM_ALLOWED_ORIGINS
          value: "https://dashboard.nephoran.com,https://api.nephoran.com"
        # Performance tuning
        - name: MAX_CONCURRENT_STREAMS
          value: "100"
        - name: CIRCUIT_BREAKER_ENABLED
          value: "true"
        - name: RATE_LIMIT_ENABLED
          value: "true"
        # Secrets from Kubernetes secrets
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: openai-prod-credentials
              key: api-key
        - name: JWT_SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: jwt-secret
              key: secret-key
        
        # Ports
        ports:
        - containerPort: 8080
          name: https
          protocol: TCP
        
        # Health checks with HTTPS
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
            scheme: HTTPS
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
            scheme: HTTPS
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
        startupProbe:
          httpGet:
            path: /healthz
            port: 8080
            scheme: HTTPS
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 20
        
        # Resource limits (production)
        resources:
          requests:
            cpu: 1000m
            memory: 2Gi
            ephemeral-storage: 1Gi
          limits:
            cpu: 4000m
            memory: 8Gi
            ephemeral-storage: 5Gi
        
        # Security context
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
        
        # Volume mounts
        volumeMounts:
        - name: tls-certs
          mountPath: /etc/certs
          readOnly: true
        - name: tmp
          mountPath: /tmp
        - name: var-tmp
          mountPath: /var/tmp
      
      volumes:
      - name: tls-certs
        secret:
          secretName: llm-processor-tls-secret  # cert-manager generated
          defaultMode: 0600
      - name: tmp
        emptyDir: {}
      - name: var-tmp
        emptyDir: {}
      
      # Pod anti-affinity for high availability
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - llm-processor
              topologyKey: kubernetes.io/hostname
      
      # Tolerations for dedicated nodes (optional)
      tolerations:
      - key: "nephoran.com/dedicated"
        operator: "Equal"
        value: "intent-processor"
        effect: "NoSchedule"
---
# Production service with LoadBalancer
apiVersion: v1
kind: Service
metadata:
  name: llm-processor
  namespace: nephoran-system
  labels:
    app: llm-processor
  annotations:
    # AWS Load Balancer annotations
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
    service.beta.kubernetes.io/aws-load-balancer-ssl-ports: "443"
    service.beta.kubernetes.io/aws-load-balancer-backend-protocol: "https"
    service.beta.kubernetes.io/aws-load-balancer-ssl-negotiation-policy: "ELBSecurityPolicy-TLS-1-2-2017-01"
spec:
  type: LoadBalancer
  loadBalancerSourceRanges:
  - "10.0.0.0/8"    # Internal traffic only
  - "172.16.0.0/12" # Private networks
  - "192.168.0.0/16"
  ports:
  - port: 443
    targetPort: 8080
    protocol: TCP
    name: https
  selector:
    app: llm-processor
---
# Pod Disruption Budget for high availability
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: llm-processor-pdb
  namespace: nephoran-system
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: llm-processor
```

### Scenario 3: Ingress-Based Deployment

Using NGINX Ingress Controller with TLS termination:

```yaml
# deployments/environments/ingress-tls.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: llm-processor-ingress
  namespace: nephoran-system
  labels:
    app: llm-processor
  annotations:
    # Ingress class
    kubernetes.io/ingress.class: "nginx"
    
    # TLS configuration
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
    nginx.ingress.kubernetes.io/proxy-ssl-verify: "on"
    nginx.ingress.kubernetes.io/proxy-ssl-name: "llm-processor.nephoran-system.svc.cluster.local"
    
    # cert-manager
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    cert-manager.io/acme-challenge-type: "http01"
    
    # Security headers
    nginx.ingress.kubernetes.io/configuration-snippet: |
      add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;
      add_header X-Content-Type-Options "nosniff" always;
      add_header X-Frame-Options "DENY" always;
      add_header X-XSS-Protection "1; mode=block" always;
      add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    
    # Rate limiting
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
    nginx.ingress.kubernetes.io/rate-limit-connections: "10"
    
    # Request size
    nginx.ingress.kubernetes.io/proxy-body-size: "1m"
    
    # Timeouts
    nginx.ingress.kubernetes.io/proxy-connect-timeout: "30"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "30"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "30"

spec:
  tls:
  - hosts:
    - llm-processor.api.nephoran.com
    - llm.api.nephoran.com
    secretName: llm-processor-ingress-tls
  
  rules:
  - host: llm-processor.api.nephoran.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: llm-processor
            port:
              number: 443
  - host: llm.api.nephoran.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: llm-processor
            port:
              number: 443
---
# Service for ingress (ClusterIP)
apiVersion: v1
kind: Service
metadata:
  name: llm-processor
  namespace: nephoran-system
  labels:
    app: llm-processor
spec:
  type: ClusterIP
  ports:
  - port: 443
    targetPort: 8080
    protocol: TCP
    name: https
  selector:
    app: llm-processor
```

## Security Best Practices

### Certificate Security

1. **Private Key Protection**
   - Store private keys in Kubernetes secrets with restricted access
   - Use RBAC to limit secret access to necessary service accounts
   - Set appropriate file permissions (600) for private keys
   - Rotate certificates regularly (every 90 days or less)

2. **Certificate Validation**
   - Always validate certificate chains in production
   - Monitor certificate expiration dates
   - Use automated renewal with cert-manager
   - Test certificate validity regularly

### Network Security

1. **Network Policies**
   ```yaml
   # Security network policy for LLM processor
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: llm-processor-tls-policy
     namespace: nephoran-system
   spec:
     podSelector:
       matchLabels:
         app: llm-processor
     policyTypes:
     - Ingress
     - Egress
     ingress:
     - from:
       - namespaceSelector:
           matchLabels:
             name: ingress-nginx
       - podSelector:
           matchLabels:
             app: nephio-bridge
       ports:
       - protocol: TCP
         port: 8080
     egress:
     # Allow outbound HTTPS for OpenAI API
     - to: []
       ports:
       - protocol: TCP
         port: 443
     # Allow DNS resolution
     - to:
       - namespaceSelector:
           matchLabels:
             name: kube-system
       ports:
       - protocol: UDP
         port: 53
   ```

2. **Service Mesh Integration**
   ```yaml
   # Istio VirtualService with TLS
   apiVersion: networking.istio.io/v1beta1
   kind: VirtualService
   metadata:
     name: llm-processor-vs
     namespace: nephoran-system
   spec:
     hosts:
     - llm-processor.api.nephoran.com
     gateways:
     - llm-processor-gateway
     http:
     - match:
       - uri:
           prefix: /
       route:
       - destination:
           host: llm-processor.nephoran-system.svc.cluster.local
           port:
             number: 8080
       headers:
         request:
           add:
             x-forwarded-proto: https
   ```

### Configuration Security

1. **Environment Variables**
   - Never store sensitive data in environment variables
   - Use Kubernetes secrets for all credentials
   - Validate all configuration parameters
   - Log security-relevant configuration changes

2. **RBAC Configuration**
   ```yaml
   # RBAC for TLS certificate management
   apiVersion: rbac.authorization.k8s.io/v1
   kind: Role
   metadata:
     name: llm-processor-tls-manager
     namespace: nephoran-system
   rules:
   - apiGroups: [""]
     resources: ["secrets"]
     resourceNames: ["llm-processor-tls", "llm-processor-tls-secret"]
     verbs: ["get", "list", "watch"]
   - apiGroups: ["cert-manager.io"]
     resources: ["certificates"]
     verbs: ["get", "list", "watch"]
   ---
   apiVersion: rbac.authorization.k8s.io/v1
   kind: RoleBinding
   metadata:
     name: llm-processor-tls-binding
     namespace: nephoran-system
   subjects:
   - kind: ServiceAccount
     name: llm-processor-sa
     namespace: nephoran-system
   roleRef:
     kind: Role
     name: llm-processor-tls-manager
     apiGroup: rbac.authorization.k8s.io
   ```

## Monitoring and Maintenance

### Certificate Monitoring

Monitor certificate expiration and health:

```yaml
# Certificate monitoring with Prometheus
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: cert-manager-metrics
  namespace: nephoran-system
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: cert-manager
  endpoints:
  - port: tcp-prometheus-servicemonitor
    interval: 60s
    path: /metrics
---
# PrometheusRule for certificate alerts
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: certificate-expiry-rules
  namespace: nephoran-system
spec:
  groups:
  - name: certificates
    rules:
    - alert: CertificateExpiringSoon
      expr: certmanager_certificate_expiration_timestamp_seconds - time() < 7 * 24 * 3600
      for: 5m
      labels:
        severity: warning
      annotations:
        summary: "Certificate {{ $labels.name }} expires in less than 7 days"
        description: "Certificate {{ $labels.name }} in namespace {{ $labels.namespace }} expires on {{ $value | humanizeTimestamp }}"
    
    - alert: CertificateExpired
      expr: certmanager_certificate_expiration_timestamp_seconds - time() <= 0
      for: 0m
      labels:
        severity: critical
      annotations:
        summary: "Certificate {{ $labels.name }} has expired"
        description: "Certificate {{ $labels.name }} in namespace {{ $labels.namespace }} has expired"
```

### Health Monitoring

```bash
#!/bin/bash
# scripts/monitor-tls-health.sh
# Monitor TLS health and certificate status

NAMESPACE="nephoran-system"
SERVICE_NAME="llm-processor"

echo "=== TLS Health Monitoring ==="

# Check service pods
echo "1. Checking service pods..."
kubectl get pods -n $NAMESPACE -l app=$SERVICE_NAME -o wide

# Check certificate status
echo "2. Checking certificate status..."
kubectl get certificates -n $NAMESPACE
kubectl describe certificate $SERVICE_NAME-tls -n $NAMESPACE | grep -E "(Status:|Events:)" -A 5

# Test HTTPS connectivity
echo "3. Testing HTTPS connectivity..."
kubectl exec -n $NAMESPACE deployment/$SERVICE_NAME -- \
  curl -k -f -s -o /dev/null -w "HTTP Status: %{http_code}, Total Time: %{time_total}s\n" \
  https://localhost:8080/healthz

# Check certificate expiration
echo "4. Checking certificate expiration..."
kubectl exec -n $NAMESPACE deployment/$SERVICE_NAME -- \
  openssl x509 -enddate -noout -in /etc/certs/tls.crt

# Monitor TLS handshake
echo "5. Testing TLS handshake..."
kubectl exec -n $NAMESPACE deployment/$SERVICE_NAME -- \
  timeout 10 openssl s_client -connect localhost:8080 -servername $SERVICE_NAME \
  </dev/null 2>/dev/null | grep -E "(Verify return code|subject=|issuer=)"

echo "✅ TLS health monitoring completed"
```

### Automated Certificate Renewal

```yaml
# CronJob for certificate health monitoring
apiVersion: batch/v1
kind: CronJob
metadata:
  name: certificate-health-check
  namespace: nephoran-system
spec:
  schedule: "0 6 * * *"  # Daily at 6 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: cert-checker
            image: bitnami/kubectl:latest
            command:
            - /bin/bash
            - -c
            - |
              set -e
              
              echo "=== Daily Certificate Health Check ==="
              
              # Check all certificates in namespace
              for cert in $(kubectl get certificates -n nephoran-system -o jsonpath='{.items[*].metadata.name}'); do
                echo "Checking certificate: $cert"
                
                # Get certificate status
                STATUS=$(kubectl get certificate $cert -n nephoran-system -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}')
                
                if [ "$STATUS" != "True" ]; then
                  echo "❌ Certificate $cert is not ready"
                  kubectl describe certificate $cert -n nephoran-system
                else
                  echo "✅ Certificate $cert is ready"
                fi
                
                # Check expiration
                EXPIRY=$(kubectl get certificate $cert -n nephoran-system -o jsonpath='{.status.notAfter}')
                if [ -n "$EXPIRY" ]; then
                  EXPIRY_TIMESTAMP=$(date -d "$EXPIRY" +%s)
                  CURRENT_TIMESTAMP=$(date +%s)
                  DAYS_REMAINING=$(( (EXPIRY_TIMESTAMP - CURRENT_TIMESTAMP) / 86400 ))
                  
                  echo "Certificate $cert expires in $DAYS_REMAINING days"
                  
                  if [ $DAYS_REMAINING -lt 30 ]; then
                    echo "⚠️ Certificate $cert expires soon, triggering renewal"
                    kubectl annotate certificate $cert -n nephoran-system \
                      cert-manager.io/force-renewal=$(date +%s) --overwrite
                  fi
                fi
              done
              
              echo "✅ Certificate health check completed"
          restartPolicy: OnFailure
          serviceAccountName: cert-health-checker
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cert-health-checker
  namespace: nephoran-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: cert-health-checker
  namespace: nephoran-system
rules:
- apiGroups: ["cert-manager.io"]
  resources: ["certificates"]
  verbs: ["get", "list", "update", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: cert-health-checker
  namespace: nephoran-system
subjects:
- kind: ServiceAccount
  name: cert-health-checker
  namespace: nephoran-system
roleRef:
  kind: Role
  name: cert-health-checker
  apiGroup: rbac.authorization.k8s.io
```

## Troubleshooting

### Common TLS Issues

#### Issue 1: Certificate Not Found

**Symptoms:**
- Service fails to start with "certificate file not found" error
- Pods in CrashLoopBackOff state

**Diagnosis:**
```bash
# Check if secret exists
kubectl get secret llm-processor-tls -n nephoran-system

# Check pod logs
kubectl logs -n nephoran-system deployment/llm-processor

# Check volume mounts
kubectl describe pod -n nephoran-system -l app=llm-processor
```

**Resolution:**
```bash
# Verify secret contains certificate data
kubectl get secret llm-processor-tls -n nephoran-system -o yaml | grep tls.crt

# Recreate secret if missing
kubectl delete secret llm-processor-tls -n nephoran-system
./scripts/generate-dev-certificates.sh
kubectl apply -f certs/tls-secret.yaml
```

#### Issue 2: Certificate/Key Mismatch

**Symptoms:**
- TLS handshake failures
- "private key does not match certificate" errors

**Diagnosis:**
```bash
# Test certificate and key match
kubectl exec -n nephoran-system deployment/llm-processor -- \
  sh -c 'openssl x509 -noout -modulus -in /etc/certs/tls.crt | openssl md5; \
         openssl rsa -noout -modulus -in /etc/certs/tls.key | openssl md5'
```

**Resolution:**
```bash
# Regenerate matching certificate and key
./scripts/generate-dev-certificates.sh
kubectl apply -f certs/tls-secret.yaml
kubectl rollout restart deployment/llm-processor -n nephoran-system
```

#### Issue 3: Certificate Expired

**Symptoms:**
- HTTPS requests fail with certificate errors
- Browser warnings about expired certificates

**Diagnosis:**
```bash
# Check certificate expiration
kubectl exec -n nephoran-system deployment/llm-processor -- \
  openssl x509 -enddate -noout -in /etc/certs/tls.crt
```

**Resolution:**
```bash
# For cert-manager certificates
kubectl annotate certificate llm-processor-tls -n nephoran-system \
  cert-manager.io/force-renewal=$(date +%s) --overwrite

# For manual certificates
./scripts/generate-dev-certificates.sh
kubectl apply -f certs/tls-secret.yaml
kubectl rollout restart deployment/llm-processor -n nephoran-system
```

#### Issue 4: TLS Handshake Failures

**Symptoms:**
- Connection timeouts or resets
- SSL/TLS handshake errors in logs

**Diagnosis:**
```bash
# Test TLS handshake
kubectl exec -n nephoran-system deployment/llm-processor -- \
  openssl s_client -connect localhost:8080 -servername llm-processor -brief

# Check TLS configuration
kubectl exec -n nephoran-system deployment/llm-processor -- \
  curl -k -v https://localhost:8080/healthz
```

**Resolution:**
```bash
# Verify certificate includes proper SANs
kubectl exec -n nephoran-system deployment/llm-processor -- \
  openssl x509 -text -noout -in /etc/certs/tls.crt | grep -A 10 "Subject Alternative Name"

# Update certificate with correct SANs if needed
./scripts/generate-dev-certificates.sh
kubectl apply -f certs/tls-secret.yaml
kubectl rollout restart deployment/llm-processor -n nephoran-system
```

### Debug Commands

#### Certificate Debugging

```bash
#!/bin/bash
# scripts/debug-certificates.sh
# Comprehensive certificate debugging

NAMESPACE="nephoran-system"
SERVICE_NAME="llm-processor"

echo "=== Certificate Debugging Suite ==="

# 1. Check Kubernetes secrets
echo "1. Kubernetes Secrets:"
kubectl get secrets -n $NAMESPACE | grep tls
kubectl describe secret $SERVICE_NAME-tls -n $NAMESPACE

# 2. Check cert-manager resources
echo "2. cert-manager Resources:"
kubectl get certificates,certificaterequests,challenges -n $NAMESPACE
kubectl describe certificate $SERVICE_NAME-tls -n $NAMESPACE

# 3. Check pod certificate files
echo "3. Pod Certificate Files:"
kubectl exec -n $NAMESPACE deployment/$SERVICE_NAME -- ls -la /etc/certs/

# 4. Test certificate properties
echo "4. Certificate Properties:"
kubectl exec -n $NAMESPACE deployment/$SERVICE_NAME -- \
  openssl x509 -text -noout -in /etc/certs/tls.crt | \
  grep -E "(Subject:|Issuer:|Not After:|Subject Alternative Name:)" -A 1

# 5. Test private key
echo "5. Private Key Validation:"
kubectl exec -n $NAMESPACE deployment/$SERVICE_NAME -- \
  openssl rsa -check -noout -in /etc/certs/tls.key

# 6. Test certificate/key pair
echo "6. Certificate/Key Pair Validation:"
CERT_HASH=$(kubectl exec -n $NAMESPACE deployment/$SERVICE_NAME -- \
  openssl x509 -noout -modulus -in /etc/certs/tls.crt | openssl md5)
KEY_HASH=$(kubectl exec -n $NAMESPACE deployment/$SERVICE_NAME -- \
  openssl rsa -noout -modulus -in /etc/certs/tls.key | openssl md5)

echo "Certificate hash: $CERT_HASH"
echo "Private key hash: $KEY_HASH"

if [ "$CERT_HASH" = "$KEY_HASH" ]; then
  echo "✅ Certificate and private key match"
else
  echo "❌ Certificate and private key do not match"
fi

# 7. Test TLS connection
echo "7. TLS Connection Test:"
kubectl exec -n $NAMESPACE deployment/$SERVICE_NAME -- \
  curl -k -v -m 10 https://localhost:8080/healthz

echo "✅ Certificate debugging completed"
```

#### Service Debugging

```bash
#!/bin/bash
# scripts/debug-tls-service.sh
# Debug TLS service connectivity

NAMESPACE="nephoran-system"
SERVICE_NAME="llm-processor"
PORT="8080"

echo "=== TLS Service Debugging ==="

# 1. Check service endpoints
echo "1. Service Endpoints:"
kubectl get endpoints $SERVICE_NAME -n $NAMESPACE
kubectl describe service $SERVICE_NAME -n $NAMESPACE

# 2. Test internal connectivity
echo "2. Internal Connectivity:"
kubectl run tls-debug --rm -it --image=curlimages/curl -- \
  curl -k -v --connect-timeout 10 https://$SERVICE_NAME.$NAMESPACE:443/healthz

# 3. Test from within pod
echo "3. Pod-to-Pod Connectivity:"
kubectl exec -n $NAMESPACE deployment/$SERVICE_NAME -- \
  curl -k -v --connect-timeout 10 https://localhost:$PORT/healthz

# 4. Check ingress (if exists)
echo "4. Ingress Configuration:"
kubectl get ingress -n $NAMESPACE
kubectl describe ingress $SERVICE_NAME-ingress -n $NAMESPACE 2>/dev/null || echo "No ingress found"

# 5. Test external connectivity (if LoadBalancer)
echo "5. External Connectivity:"
EXTERNAL_IP=$(kubectl get service $SERVICE_NAME -n $NAMESPACE -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
if [ -n "$EXTERNAL_IP" ]; then
  echo "Testing external IP: $EXTERNAL_IP"
  curl -k -v --connect-timeout 10 https://$EXTERNAL_IP/healthz || echo "External connectivity failed"
else
  echo "No external IP found"
fi

echo "✅ Service debugging completed"
```

## Operational Runbooks

### Daily Operations Checklist

```bash
#!/bin/bash
# scripts/daily-tls-operations.sh
# Daily TLS operations checklist

echo "=== Daily TLS Operations Checklist ==="
DATE=$(date +%Y-%m-%d)

# 1. Check certificate health
echo "1. Certificate Health Check"
kubectl get certificates -n nephoran-system -o custom-columns="NAME:.metadata.name,READY:.status.conditions[?(@.type=='Ready')].status,SECRET:.spec.secretName,AGE:.metadata.creationTimestamp"

# 2. Check certificate expiration
echo "2. Certificate Expiration Check"
for cert in $(kubectl get certificates -n nephoran-system -o jsonpath='{.items[*].metadata.name}'); do
  EXPIRY=$(kubectl get certificate $cert -n nephoran-system -o jsonpath='{.status.notAfter}' 2>/dev/null)
  if [ -n "$EXPIRY" ]; then
    DAYS_LEFT=$(( ($(date -d "$EXPIRY" +%s) - $(date +%s)) / 86400 ))
    echo "Certificate $cert: $DAYS_LEFT days remaining"
    if [ $DAYS_LEFT -lt 30 ]; then
      echo "⚠️ Certificate $cert expires in $DAYS_LEFT days"
    fi
  fi
done

# 3. Service health check
echo "3. Service Health Check"
kubectl get pods -n nephoran-system -l app=llm-processor -o wide
kubectl exec -n nephoran-system deployment/llm-processor -- curl -k -f -s https://localhost:8080/healthz >/dev/null && echo "✅ Service healthy" || echo "❌ Service unhealthy"

# 4. Check recent events
echo "4. Recent Events"
kubectl get events -n nephoran-system --sort-by='.lastTimestamp' | grep -i -E "(certificate|tls|ssl)" | tail -5

# 5. Check cert-manager logs for errors
echo "5. cert-manager Status"
kubectl get pods -n cert-manager -o wide
kubectl logs -n cert-manager deployment/cert-manager --since=24h | grep -i error | tail -3 || echo "No recent errors"

echo "✅ Daily TLS operations check completed - $DATE"
```

### Weekly Maintenance Tasks

```bash
#!/bin/bash
# scripts/weekly-tls-maintenance.sh
# Weekly TLS maintenance tasks

echo "=== Weekly TLS Maintenance ==="

# 1. Certificate rotation dry run
echo "1. Certificate Rotation Testing"
./scripts/test-certificate-rotation.sh --dry-run

# 2. Security scan
echo "2. TLS Security Scan"
kubectl exec -n nephoran-system deployment/llm-processor -- \
  nmap --script ssl-enum-ciphers -p 8080 localhost | grep -E "(TLSv|cipher)"

# 3. Backup certificates
echo "3. Certificate Backup"
BACKUP_DIR="./cert-backups/$(date +%Y%m%d)"
mkdir -p "$BACKUP_DIR"
kubectl get secrets -n nephoran-system -o yaml | grep -E "(tls|certificate)" > "$BACKUP_DIR/certificates.yaml"

# 4. Update documentation
echo "4. Update Certificate Inventory"
kubectl get certificates -n nephoran-system -o yaml > "./docs/certificate-inventory-$(date +%Y%m%d).yaml"

# 5. Performance testing
echo "5. TLS Performance Testing"
./scripts/tls-performance-test.sh

echo "✅ Weekly TLS maintenance completed"
```

### Emergency Procedures

#### Certificate Compromise Response

```bash
#!/bin/bash
# scripts/emergency-cert-revocation.sh
# Emergency certificate revocation procedure

set -e

NAMESPACE="nephoran-system"
CERT_NAME="llm-processor-tls"

echo "🚨 EMERGENCY: Certificate Compromise Response"
echo "Timestamp: $(date)"
echo "Certificate: $CERT_NAME"

# 1. Immediate revocation
echo "1. Revoking compromised certificate..."
kubectl annotate certificate $CERT_NAME -n $NAMESPACE \
  cert-manager.io/revoke-certificate=true --overwrite

# 2. Generate new certificate
echo "2. Generating new certificate..."
kubectl annotate certificate $CERT_NAME -n $NAMESPACE \
  cert-manager.io/force-renewal=$(date +%s) --overwrite

# 3. Update service immediately
echo "3. Rolling restart to pick up new certificate..."
kubectl rollout restart deployment/llm-processor -n $NAMESPACE

# 4. Wait for new certificate
echo "4. Waiting for new certificate..."
sleep 60

# 5. Validate new certificate
echo "5. Validating new certificate..."
kubectl wait --for=condition=Ready certificate/$CERT_NAME -n $NAMESPACE --timeout=300s

# 6. Test service
echo "6. Testing service with new certificate..."
kubectl exec -n $NAMESPACE deployment/llm-processor -- \
  curl -k -f https://localhost:8080/healthz

echo "✅ Emergency certificate rotation completed"
echo "New certificate serial: $(kubectl exec -n $NAMESPACE deployment/llm-processor -- openssl x509 -serial -noout -in /etc/certs/tls.crt)"
```

This comprehensive TLS deployment guide provides everything needed to securely deploy and manage the Nephoran Intent Operator LLM Processor with TLS encryption across development, staging, and production environments.