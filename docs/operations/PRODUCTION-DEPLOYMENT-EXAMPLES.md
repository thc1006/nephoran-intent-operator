# Production Kubernetes Deployment Examples

This guide provides practical, production-ready Kubernetes deployment examples for the Nephoran Intent Operator with comprehensive security configurations.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Environment Variable Templates](#environment-variable-templates)
- [OAuth2 Security Deployment](#oauth2-security-deployment)
- [CORS Middleware Configuration](#cors-middleware-configuration)
- [Complete Production Deployment](#complete-production-deployment)
- [Multi-Environment Setup](#multi-environment-setup)
- [Security Best Practices](#security-best-practices)
- [Monitoring and Observability](#monitoring-and-observability)

## Prerequisites

- Kubernetes cluster v1.25+
- kubectl configured with cluster admin access
- Helm v3.8+ (for optional chart deployment)
- TLS certificates for secure communication
- OAuth2 provider setup (Azure AD, Okta, Keycloak, etc.)

## Environment Variable Templates

### OAuth2 Environment Variables

Create a ConfigMap and Secret for OAuth2 configuration:

```yaml
# oauth2-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth2-config
  namespace: nephoran-system
  labels:
    app.kubernetes.io/name: nephoran-intent-operator
    app.kubernetes.io/component: auth
data:
  # OAuth2 Provider Configuration
  OAUTH2_ENABLED: "true"
  
  # Azure AD Configuration
  AZURE_ENABLED: "true"
  AZURE_CLIENT_ID: "your-azure-client-id"
  AZURE_TENANT_ID: "your-azure-tenant-id"
  AZURE_AUTHORITY: "https://login.microsoftonline.com"
  AZURE_SCOPES: "openid,profile,email,User.Read"
  
  # Keycloak Configuration
  KEYCLOAK_ENABLED: "true"
  KEYCLOAK_CLIENT_ID: "nephoran-client"
  KEYCLOAK_BASE_URL: "https://keycloak.example.com"
  KEYCLOAK_REALM: "nephoran"
  KEYCLOAK_SCOPES: "openid,profile,email,roles"
  
  # Okta Configuration
  OKTA_ENABLED: "false"
  OKTA_CLIENT_ID: "your-okta-client-id"
  OKTA_DOMAIN: "your-domain.okta.com"
  OKTA_SCOPES: "openid,profile,email,groups"
  
  # Google OAuth2 Configuration
  GOOGLE_ENABLED: "false"
  GOOGLE_CLIENT_ID: "your-google-client-id.apps.googleusercontent.com"
  GOOGLE_SCOPES: "openid,profile,email"
  
  # RBAC Configuration
  RBAC_ENABLED: "true"
  DEFAULT_ROLE: "viewer"
  ADMIN_USERS: "admin@example.com,ops-team@example.com"
  OPERATOR_USERS: "network-ops@example.com"
  
  # JWT Configuration
  TOKEN_TTL: "24h"
  REFRESH_TTL: "168h"  # 7 days

---
apiVersion: v1
kind: Secret
metadata:
  name: oauth2-secrets
  namespace: nephoran-system
  labels:
    app.kubernetes.io/name: nephoran-intent-operator
    app.kubernetes.io/component: auth
type: Opaque
data:
  # Base64 encoded OAuth2 client secrets
  OAUTH2_AZURE_CLIENT_SECRET: <base64-encoded-azure-secret>
  OAUTH2_KEYCLOAK_CLIENT_SECRET: <base64-encoded-keycloak-secret>
  OAUTH2_OKTA_CLIENT_SECRET: <base64-encoded-okta-secret>
  OAUTH2_GOOGLE_CLIENT_SECRET: <base64-encoded-google-secret>
  
  # JWT Secret Key (minimum 32 characters)
  JWT_SECRET_KEY: <base64-encoded-jwt-secret>
```

### CORS Configuration

```yaml
# cors-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cors-config
  namespace: nephoran-system
  labels:
    app.kubernetes.io/name: nephoran-intent-operator
    app.kubernetes.io/component: middleware
data:
  # CORS Security Configuration
  CORS_ALLOWED_ORIGINS: "https://nephoran-ui.example.com,https://admin.example.com"
  CORS_ALLOWED_METHODS: "GET,POST,PUT,DELETE,OPTIONS"
  CORS_ALLOWED_HEADERS: "Content-Type,Authorization,X-Requested-With,Accept,Origin"
  CORS_ALLOW_CREDENTIALS: "true"
  CORS_MAX_AGE: "86400"  # 24 hours
  
  # Security Headers
  SECURITY_HEADERS_ENABLED: "true"
  X_CONTENT_TYPE_OPTIONS: "nosniff"
  X_FRAME_OPTIONS: "DENY"
  X_XSS_PROTECTION: "1; mode=block"
  CONTENT_SECURITY_POLICY: "default-src 'self'; script-src 'self' 'unsafe-inline'"
```

## OAuth2 Security Deployment

### Complete LLM Processor with OAuth2

```yaml
# llm-processor-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-processor
  namespace: nephoran-system
  labels:
    app.kubernetes.io/name: nephoran-intent-operator
    app.kubernetes.io/component: llm-processor
    app.kubernetes.io/version: "v1.0.0"
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: nephoran-intent-operator
      app.kubernetes.io/component: llm-processor
  template:
    metadata:
      labels:
        app.kubernetes.io/name: nephoran-intent-operator
        app.kubernetes.io/component: llm-processor
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: nephoran-llm-processor
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 1000
        fsGroup: 1000
      containers:
      - name: llm-processor
        image: nephoran/llm-processor:v1.0.0
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080
          name: http
          protocol: TCP
        - containerPort: 8081
          name: health
          protocol: TCP
        env:
        # Load configuration from ConfigMaps
        - name: OAUTH2_ENABLED
          valueFrom:
            configMapKeyRef:
              name: oauth2-config
              key: OAUTH2_ENABLED
        - name: AZURE_ENABLED
          valueFrom:
            configMapKeyRef:
              name: oauth2-config
              key: AZURE_ENABLED
        - name: AZURE_CLIENT_ID
          valueFrom:
            configMapKeyRef:
              name: oauth2-config
              key: AZURE_CLIENT_ID
        - name: AZURE_TENANT_ID
          valueFrom:
            configMapKeyRef:
              name: oauth2-config
              key: AZURE_TENANT_ID
        - name: KEYCLOAK_ENABLED
          valueFrom:
            configMapKeyRef:
              name: oauth2-config
              key: KEYCLOAK_ENABLED
        - name: KEYCLOAK_CLIENT_ID
          valueFrom:
            configMapKeyRef:
              name: oauth2-config
              key: KEYCLOAK_CLIENT_ID
        - name: KEYCLOAK_BASE_URL
          valueFrom:
            configMapKeyRef:
              name: oauth2-config
              key: KEYCLOAK_BASE_URL
        - name: KEYCLOAK_REALM
          valueFrom:
            configMapKeyRef:
              name: oauth2-config
              key: KEYCLOAK_REALM
        
        # Load sensitive configuration from Secrets
        - name: OAUTH2_AZURE_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: oauth2-secrets
              key: OAUTH2_AZURE_CLIENT_SECRET
        - name: OAUTH2_KEYCLOAK_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: oauth2-secrets
              key: OAUTH2_KEYCLOAK_CLIENT_SECRET
        - name: JWT_SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: oauth2-secrets
              key: JWT_SECRET_KEY
        
        # CORS Configuration
        - name: CORS_ALLOWED_ORIGINS
          valueFrom:
            configMapKeyRef:
              name: cors-config
              key: CORS_ALLOWED_ORIGINS
        - name: CORS_ALLOWED_METHODS
          valueFrom:
            configMapKeyRef:
              name: cors-config
              key: CORS_ALLOWED_METHODS
        - name: CORS_ALLOW_CREDENTIALS
          valueFrom:
            configMapKeyRef:
              name: cors-config
              key: CORS_ALLOW_CREDENTIALS
        
        # LLM Configuration
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: llm-secrets
              key: OPENAI_API_KEY
        - name: LLM_MODEL
          value: "gpt-4o-mini"
        - name: LLM_MAX_TOKENS
          value: "4096"
        - name: LLM_TEMPERATURE
          value: "0.0"
        
        # RAG Configuration
        - name: RAG_API_URL
          value: "http://rag-api:8080"
        - name: WEAVIATE_URL
          value: "http://weaviate:8080"
        - name: WEAVIATE_INDEX
          value: "TelecomKnowledge"
        
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
            ephemeral-storage: "1Gi"
          limits:
            memory: "8Gi"
            cpu: "4000m"
            ephemeral-storage: "5Gi"
        
        livenessProbe:
          httpGet:
            path: /healthz
            port: health
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        
        readinessProbe:
          httpGet:
            path: /readyz
            port: health
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          successThreshold: 1
          failureThreshold: 3
        
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
          seccompProfile:
            type: RuntimeDefault
        
        volumeMounts:
        - name: tmp
          mountPath: /tmp
        - name: cache
          mountPath: /app/cache
        - name: config
          mountPath: /app/config
          readOnly: true
      
      volumes:
      - name: tmp
        emptyDir: {}
      - name: cache
        emptyDir:
          sizeLimit: 1Gi
      - name: config
        configMap:
          name: llm-processor-config
      
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app.kubernetes.io/component
                  operator: In
                  values: [llm-processor]
              topologyKey: kubernetes.io/hostname
      
      tolerations:
      - key: "node-role.kubernetes.io/control-plane"
        operator: "Exists"
        effect: "NoSchedule"

---
# Service for LLM Processor
apiVersion: v1
kind: Service
metadata:
  name: llm-processor
  namespace: nephoran-system
  labels:
    app.kubernetes.io/name: nephoran-intent-operator
    app.kubernetes.io/component: llm-processor
spec:
  type: ClusterIP
  selector:
    app.kubernetes.io/name: nephoran-intent-operator
    app.kubernetes.io/component: llm-processor
  ports:
  - name: http
    port: 8080
    targetPort: http
    protocol: TCP
  - name: health
    port: 8081
    targetPort: health
    protocol: TCP

---
# ServiceAccount with proper RBAC
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nephoran-llm-processor
  namespace: nephoran-system
  labels:
    app.kubernetes.io/name: nephoran-intent-operator
    app.kubernetes.io/component: llm-processor
automountServiceAccountToken: true

---
# RBAC Configuration
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nephoran-llm-processor
  labels:
    app.kubernetes.io/name: nephoran-intent-operator
    app.kubernetes.io/component: llm-processor
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "patch"]
- apiGroups: ["metrics.k8s.io"]
  resources: ["*"]
  verbs: ["get", "list"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: nephoran-llm-processor
  labels:
    app.kubernetes.io/name: nephoran-intent-operator
    app.kubernetes.io/component: llm-processor
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: nephoran-llm-processor
subjects:
- kind: ServiceAccount
  name: nephoran-llm-processor
  namespace: nephoran-system
```

## CORS Middleware Configuration

### Ingress with CORS and Security Headers

```yaml
# ingress-with-cors.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: nephoran-api-ingress
  namespace: nephoran-system
  labels:
    app.kubernetes.io/name: nephoran-intent-operator
    app.kubernetes.io/component: ingress
  annotations:
    # NGINX Ingress Controller annotations
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
    
    # CORS Configuration (handled by application, but can be supplemented)
    nginx.ingress.kubernetes.io/enable-cors: "true"
    nginx.ingress.kubernetes.io/cors-allow-origin: "https://nephoran-ui.example.com"
    nginx.ingress.kubernetes.io/cors-allow-methods: "GET, POST, PUT, DELETE, OPTIONS"
    nginx.ingress.kubernetes.io/cors-allow-headers: "Content-Type, Authorization, X-Requested-With"
    nginx.ingress.kubernetes.io/cors-allow-credentials: "true"
    nginx.ingress.kubernetes.io/cors-max-age: "86400"
    
    # Security Headers
    nginx.ingress.kubernetes.io/server-snippet: |
      add_header X-Content-Type-Options nosniff always;
      add_header X-Frame-Options DENY always;
      add_header X-XSS-Protection "1; mode=block" always;
      add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
      add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline';" always;
    
    # Rate Limiting
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-rps: "10"
    
    # Certificate Management
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    
spec:
  tls:
  - hosts:
    - api.nephoran.example.com
    secretName: nephoran-api-tls
  rules:
  - host: api.nephoran.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: llm-processor
            port:
              number: 8080
      - path: /auth
        pathType: Prefix
        backend:
          service:
            name: llm-processor
            port:
              number: 8080
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: llm-processor
            port:
              number: 8080
```

## Complete Production Deployment

### Namespace and Network Policies

```yaml
# namespace-and-security.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: nephoran-system
  labels:
    name: nephoran-system
    security-policy: restricted
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted

---
# Network Policy for Micro-segmentation
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: nephoran-default-deny
  namespace: nephoran-system
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress

---
# Allow LLM Processor to communicate with RAG API
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: llm-processor-policy
  namespace: nephoran-system
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/component: llm-processor
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
          app.kubernetes.io/name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  - from:
    - podSelector:
        matchLabels:
          app.kubernetes.io/component: nephio-bridge
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app.kubernetes.io/component: rag-api
    ports:
    - protocol: TCP
      port: 8080
  - to:
    - podSelector:
        matchLabels:
          app.kubernetes.io/component: weaviate
    ports:
    - protocol: TCP
      port: 8080
  - to: []  # Allow external access for OAuth2 and OpenAI
    ports:
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 53
    - protocol: UDP
      port: 53
```

## Multi-Environment Setup

### Development Environment

```yaml
# dev-environment.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth2-config-dev
  namespace: nephoran-dev
data:
  OAUTH2_ENABLED: "true"
  AZURE_ENABLED: "true"
  CORS_ALLOWED_ORIGINS: "http://localhost:3000,http://localhost:8080"  # Development origins
  CORS_ALLOW_CREDENTIALS: "false"  # Disabled for development
  LOG_LEVEL: "debug"
  SECURITY_HEADERS_ENABLED: "false"  # Relaxed for development
```

### Staging Environment

```yaml
# staging-environment.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth2-config-staging
  namespace: nephoran-staging
data:
  OAUTH2_ENABLED: "true"
  AZURE_ENABLED: "true"
  CORS_ALLOWED_ORIGINS: "https://staging-ui.nephoran.example.com"
  CORS_ALLOW_CREDENTIALS: "true"
  LOG_LEVEL: "info"
  SECURITY_HEADERS_ENABLED: "true"
  RBAC_ENABLED: "true"
```

### Production Environment

```yaml
# production-environment.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth2-config-prod
  namespace: nephoran-system
data:
  OAUTH2_ENABLED: "true"
  AZURE_ENABLED: "true"
  KEYCLOAK_ENABLED: "true"
  CORS_ALLOWED_ORIGINS: "https://ui.nephoran.example.com,https://admin.nephoran.example.com"
  CORS_ALLOW_CREDENTIALS: "true"
  LOG_LEVEL: "warn"
  SECURITY_HEADERS_ENABLED: "true"
  RBAC_ENABLED: "true"
  TOKEN_BLACKLIST_ENABLED: "true"
  AUDIT_LOGGING_ENABLED: "true"
```

## Security Best Practices

### Pod Security Standards

```yaml
# pod-security-policy.yaml
apiVersion: v1
kind: Pod
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 1000
    runAsGroup: 1000
    fsGroup: 1000
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: llm-processor
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop:
        - ALL
      seccompProfile:
        type: RuntimeDefault
```

### Secret Management with External Secrets Operator

```yaml
# external-secrets.yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: azure-keyvault
  namespace: nephoran-system
spec:
  provider:
    azurekv:
      tenantId: "your-tenant-id"
      vaultUrl: "https://nephoran-vault.vault.azure.net/"
      authSecretRef:
        clientId:
          name: azure-secret
          key: client-id
        clientSecret:
          name: azure-secret
          key: client-secret

---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: oauth2-secrets
  namespace: nephoran-system
spec:
  secretStoreRef:
    name: azure-keyvault
    kind: SecretStore
  target:
    name: oauth2-secrets
    creationPolicy: Owner
  data:
  - secretKey: OAUTH2_AZURE_CLIENT_SECRET
    remoteRef:
      key: oauth2-azure-client-secret
  - secretKey: JWT_SECRET_KEY
    remoteRef:
      key: jwt-secret-key
  - secretKey: OPENAI_API_KEY
    remoteRef:
      key: openai-api-key
```

## Monitoring and Observability

### ServiceMonitor for Prometheus

```yaml
# monitoring.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: llm-processor
  namespace: nephoran-system
  labels:
    app.kubernetes.io/name: nephoran-intent-operator
    app.kubernetes.io/component: llm-processor
spec:
  selector:
    matchLabels:
      app.kubernetes.io/component: llm-processor
  endpoints:
  - port: http
    path: /metrics
    interval: 30s
    scrapeTimeout: 10s
    honorLabels: true

---
# PodMonitor for detailed pod metrics
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: llm-processor-pods
  namespace: nephoran-system
spec:
  selector:
    matchLabels:
      app.kubernetes.io/component: llm-processor
  podMetricsEndpoints:
  - port: http
    path: /metrics
    interval: 15s
```

### Grafana Dashboard ConfigMap

```yaml
# grafana-dashboard.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nephoran-dashboard
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  nephoran-dashboard.json: |
    {
      "dashboard": {
        "title": "Nephoran Intent Operator",
        "panels": [
          {
            "title": "OAuth2 Authentication Requests",
            "type": "stat",
            "targets": [
              {
                "expr": "rate(nephoran_oauth2_requests_total[5m])",
                "legendFormat": "{{provider}}"
              }
            ]
          },
          {
            "title": "CORS Wildcard Warnings",
            "type": "stat",
            "targets": [
              {
                "expr": "increase(nephoran_cors_wildcard_warnings_total[1h])",
                "legendFormat": "Wildcard Warnings"
              }
            ]
          },
          {
            "title": "LLM Processing Latency",
            "type": "graph",
            "targets": [
              {
                "expr": "histogram_quantile(0.95, rate(nephoran_llm_request_duration_seconds_bucket[5m]))",
                "legendFormat": "95th percentile"
              }
            ]
          }
        ]
      }
    }
```

## Deployment Commands

### Quick Deployment Script

```bash
#!/bin/bash
# deploy-production.sh

set -euo pipefail

NAMESPACE="nephoran-system"
ENVIRONMENT="production"

echo "🚀 Deploying Nephoran Intent Operator to ${ENVIRONMENT}"

# Create namespace
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

# Apply security configurations
kubectl apply -f namespace-and-security.yaml

# Deploy secrets (assuming they're prepared)
kubectl apply -f oauth2-config.yaml
kubectl apply -f cors-config.yaml

# Deploy the application
kubectl apply -f llm-processor-deployment.yaml

# Deploy ingress
kubectl apply -f ingress-with-cors.yaml

# Deploy monitoring
kubectl apply -f monitoring.yaml

# Wait for rollout
kubectl rollout status deployment/llm-processor -n ${NAMESPACE} --timeout=300s

# Verify deployment
kubectl get pods -n ${NAMESPACE} -l app.kubernetes.io/component=llm-processor

echo "✅ Deployment complete!"
echo "🔍 Check status: kubectl get pods -n ${NAMESPACE}"
echo "📊 View logs: kubectl logs -f deployment/llm-processor -n ${NAMESPACE}"
```

This comprehensive deployment guide provides production-ready examples with:

- ✅ **Secure OAuth2 configuration** with multiple providers
- ✅ **CORS security** with proper origin validation
- ✅ **Network policies** for micro-segmentation
- ✅ **Pod Security Standards** compliance
- ✅ **External secret management** integration
- ✅ **Monitoring and observability** setup
- ✅ **Multi-environment** configuration examples
- ✅ **Security best practices** implementation

All examples follow Kubernetes security best practices and are ready for production deployment.