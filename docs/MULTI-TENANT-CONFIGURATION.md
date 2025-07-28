# Multi-Tenant Configuration for Nephoran Intent Operator
## Enterprise Multi-Tenancy, Security Patterns, and Isolation Strategies

### Overview

This document provides comprehensive guidance for configuring multi-tenant deployments of the Nephoran Intent Operator RAG system. Multi-tenancy enables multiple organizations, departments, or service providers to share a single Nephoran deployment while maintaining strict isolation, security, and resource management.

The multi-tenant architecture supports various deployment models:
- **Enterprise Departments**: Separate network operations for different business units
- **Service Provider Multi-Tenancy**: Multiple telecom operators sharing infrastructure
- **Geographic Isolation**: Regional deployments with centralized management
- **Development/Staging/Production**: Environment-based tenant separation
- **Partner Integration**: Secure access for third-party network management partners

### Table of Contents

1. [Multi-Tenant Architecture](#multi-tenant-architecture)
2. [Tenant Isolation Strategies](#tenant-isolation-strategies)
3. [Security and Authentication](#security-and-authentication)
4. [Resource Management and Quotas](#resource-management-and-quotas)
5. [RAG System Multi-Tenancy](#rag-system-multi-tenancy)
6. [Network Policy Configuration](#network-policy-configuration)
7. [Deployment Patterns](#deployment-patterns)
8. [Monitoring and Observability](#monitoring-and-observability)
9. [Backup and Recovery](#backup-and-recovery)
10. [Troubleshooting Multi-Tenant Issues](#troubleshooting-multi-tenant-issues)

## Multi-Tenant Architecture

### Conceptual Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Multi-Tenant Nephoran Architecture                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │
│  │   Tenant A      │  │   Tenant B      │  │       Tenant C              │ │
│  │   (Enterprise)  │  │   (Operator 1)  │  │   (Development)             │ │
│  │                 │  │                 │  │                             │ │
│  │ Namespace:      │  │ Namespace:      │  │ Namespace:                  │ │
│  │ nephoran-ent-a  │  │ nephoran-op-b   │  │ nephoran-dev-c              │ │
│  │                 │  │                 │  │                             │ │
│  │ • NetworkIntents│  │ • O-RAN Focus   │  │ • Test Environment          │ │
│  │ • E2NodeSets    │  │ • Core Network  │  │ • Reduced Resources         │ │
│  │ • Custom CRDs   │  │ • Edge Computing│  │ • Shared Knowledge Base     │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │
│           │                       │                         │             │
│           └───────────────────────┼─────────────────────────┘             │
│                                   ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                      Shared Control Plane                              │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐ │ │
│  │  │   Multi-Tenant  │  │  Tenant-Aware   │  │   Shared RAG System     │ │ │
│  │  │   Controllers   │  │  LLM Processor  │  │   with Isolation        │ │ │
│  │  │                 │  │                 │  │                         │ │ │
│  │  │ • Tenant Router │  │ • Context       │  │ • Tenant Namespaces     │ │ │
│  │  │ • RBAC Engine   │  │   Isolation     │  │ • Data Segregation      │ │ │
│  │  │ • Quota Manager │  │ • Resource      │  │ • Access Control        │ │ │
│  │  │ • Audit Logger  │  │   Limits        │  │ • Usage Tracking        │ │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
│                                    │                                       │
│  ┌─────────────────────────────────────────────────────────────────────────┐ │
│  │                     Shared Infrastructure                               │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐ │ │
│  │  │   Weaviate      │  │  Kubernetes     │  │   Monitoring Stack      │ │ │
│  │  │   Vector DB     │  │  Cluster        │  │                         │ │ │
│  │  │                 │  │                 │  │ • Tenant Metrics        │ │ │
│  │  │ • Multi-Tenant  │  │ • Namespaces    │  │ • Isolated Dashboards   │ │ │
│  │  │   Schema        │  │ • Network       │  │ • Audit Logging         │ │ │
│  │  │ • Data          │  │   Policies      │  │ • Usage Reporting       │ │ │
│  │  │   Isolation     │  │ • RBAC          │  │                         │ │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Multi-Tenant Components

#### 1. Tenant-Aware Controllers

```go
// Multi-tenant NetworkIntent Controller
type MultiTenantNetworkIntentController struct {
    client.Client
    Scheme       *runtime.Scheme
    TenantRouter *TenantRouter
    QuotaManager *ResourceQuotaManager
    AuditLogger  *AuditLogger
}

type TenantRouter struct {
    TenantMappings map[string]*TenantConfig
    DefaultTenant  string
    IsolationMode  IsolationLevel
}

type TenantConfig struct {
    TenantID          string            `json:"tenant_id"`
    DisplayName       string            `json:"display_name"`
    Namespace         string            `json:"namespace"`
    ResourceQuotas    ResourceQuotas    `json:"resource_quotas"`
    AllowedCRDs       []string          `json:"allowed_crds"`
    RAGConfiguration  RAGTenantConfig   `json:"rag_configuration"`
    NetworkPolicies   []NetworkPolicy   `json:"network_policies"`
    SecurityPolicies  SecurityConfig    `json:"security_policies"`
    MetadataLabels    map[string]string `json:"metadata_labels"`
}

type IsolationLevel string

const (
    IsolationStrict    IsolationLevel = "strict"    // Complete isolation
    IsolationShared    IsolationLevel = "shared"    // Shared with restrictions
    IsolationDevelopment IsolationLevel = "development" // Minimal isolation
)
```

#### 2. RAG System Multi-Tenancy

```yaml
# Multi-tenant RAG configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: rag-multi-tenant-config
  namespace: nephoran-system
data:
  multi_tenant_config.yaml: |
    multi_tenancy:
      enabled: true
      isolation_mode: "strict"
      default_tenant: "system"
      tenant_discovery: "namespace_label"
      
    tenants:
      enterprise_a:
        tenant_id: "enterprise_a"
        display_name: "Enterprise A - Manufacturing"
        namespace: "nephoran-ent-a"
        weaviate_class_prefix: "EntA_"
        knowledge_base_path: "/kb/enterprise_a"
        resource_quotas:
          max_documents: 100000
          max_queries_per_hour: 10000
          max_concurrent_queries: 50
        allowed_sources: ["3GPP", "O-RAN", "Internal"]
        
      operator_b:
        tenant_id: "operator_b"
        display_name: "Telecom Operator B"
        namespace: "nephoran-op-b"
        weaviate_class_prefix: "OpB_"
        knowledge_base_path: "/kb/operator_b"
        resource_quotas:
          max_documents: 500000
          max_queries_per_hour: 50000
          max_concurrent_queries: 200
        allowed_sources: ["3GPP", "O-RAN", "ETSI", "ITU"]
        
      development:
        tenant_id: "development"
        display_name: "Development Environment"
        namespace: "nephoran-dev"
        weaviate_class_prefix: "Dev_"
        knowledge_base_path: "/kb/development"
        resource_quotas:
          max_documents: 10000
          max_queries_per_hour: 1000
          max_concurrent_queries: 10
        allowed_sources: ["3GPP", "O-RAN", "Test"]
```

## Tenant Isolation Strategies

### 1. Namespace-Based Isolation

```yaml
# Complete tenant namespace setup
apiVersion: v1
kind: Namespace
metadata:
  name: nephoran-ent-a
  labels:
    nephoran.com/tenant: "enterprise_a"
    nephoran.com/isolation-level: "strict"
    nephoran.com/tier: "production"
  annotations:
    nephoran.com/description: "Enterprise A Manufacturing Operations"
    nephoran.com/contact: "netops@enterprise-a.com"
    nephoran.com/created-by: "tenant-provisioner"

---
# Resource quotas for tenant isolation
apiVersion: v1
kind: ResourceQuota
metadata:
  name: tenant-enterprise-a-quota
  namespace: nephoran-ent-a
spec:
  hard:
    # Compute resources
    requests.cpu: "20"
    requests.memory: "40Gi"
    limits.cpu: "40"
    limits.memory: "80Gi"
    
    # Storage resources
    requests.storage: "100Gi"
    persistentvolumeclaims: "10"
    
    # Network resources
    services: "20"
    services.loadbalancers: "5"
    services.nodeports: "0"
    
    # Custom resources
    networkintents.nephoran.com: "100"
    e2nodesets.nephoran.com: "50"
    managedelements.nephoran.com: "200"
    
    # Kubernetes objects
    pods: "100"
    configmaps: "50"
    secrets: "30"

---
# Limit ranges for pod-level controls
apiVersion: v1
kind: LimitRange
metadata:
  name: tenant-enterprise-a-limits
  namespace: nephoran-ent-a
spec:
  limits:
  - default:
      cpu: "1000m"
      memory: "2Gi"
    defaultRequest:
      cpu: "100m"
      memory: "256Mi"
    max:
      cpu: "4000m"
      memory: "8Gi"
    min:
      cpu: "50m"
      memory: "128Mi"
    type: Container
  - default:
      storage: "10Gi"
    max:
      storage: "100Gi"
    min:
      storage: "1Gi"
    type: PersistentVolumeClaim
```

### 2. RBAC-Based Access Control

```yaml
# Tenant-specific service account
apiVersion: v1
kind: ServiceAccount
metadata:
  name: nephoran-enterprise-a-operator
  namespace: nephoran-ent-a
  labels:
    nephoran.com/tenant: "enterprise_a"

---
# Tenant-scoped cluster role
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nephoran-tenant-enterprise-a
rules:
# NetworkIntent permissions
- apiGroups: ["nephoran.com"]
  resources: ["networkintents"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  resourceNames: [] # All resources in tenant namespace

# E2NodeSet permissions
- apiGroups: ["nephoran.com"]
  resources: ["e2nodesets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# ManagedElement permissions (read-only for this tenant)
- apiGroups: ["nephoran.com"]
  resources: ["managedelements"]
  verbs: ["get", "list", "watch"]

# Core Kubernetes resources in tenant namespace
- apiGroups: [""]
  resources: ["pods", "services", "configmaps", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

# Monitoring access
- apiGroups: [""]
  resources: ["pods/log", "pods/status"]
  verbs: ["get", "list"]

# Metrics access (filtered by tenant)
- apiGroups: ["metrics.k8s.io"]
  resources: ["pods", "nodes"]
  verbs: ["get", "list"]

---
# Tenant-scoped role binding
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: nephoran-enterprise-a-binding
  namespace: nephoran-ent-a
subjects:
- kind: ServiceAccount
  name: nephoran-enterprise-a-operator
  namespace: nephoran-ent-a
- kind: User
  name: enterprise-a-admin
  apiGroup: rbac.authorization.k8s.io
- kind: Group
  name: enterprise-a-operators
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: nephoran-tenant-enterprise-a
  apiGroup: rbac.authorization.k8s.io

---
# Cross-namespace permissions for shared resources
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: nephoran-enterprise-a-shared-access
subjects:
- kind: ServiceAccount
  name: nephoran-enterprise-a-operator
  namespace: nephoran-ent-a
roleRef:
  kind: ClusterRole
  name: nephoran-shared-resources-reader
  apiGroup: rbac.authorization.k8s.io
```

### 3. Network Isolation Policies

```yaml
# Tenant network isolation
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tenant-enterprise-a-isolation
  namespace: nephoran-ent-a
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  
  # Ingress rules - what can access this tenant
  ingress:
  # Allow from same tenant namespace
  - from:
    - namespaceSelector:
        matchLabels:
          nephoran.com/tenant: "enterprise_a"
    ports:
    - protocol: TCP
      port: 8080
    - protocol: TCP
      port: 9090
  
  # Allow from shared services
  - from:
    - namespaceSelector:
        matchLabels:
          name: "nephoran-system"
    ports:
    - protocol: TCP
      port: 8080
      
  # Allow from monitoring namespace
  - from:
    - namespaceSelector:
        matchLabels:
          name: "monitoring"
    ports:
    - protocol: TCP
      port: 9090
  
  # Egress rules - what this tenant can access
  egress:
  # Allow to same tenant namespace
  - to:
    - namespaceSelector:
        matchLabels:
          nephoran.com/tenant: "enterprise_a"
  
  # Allow to shared Nephoran services
  - to:
    - namespaceSelector:
        matchLabels:
          name: "nephoran-system"
    ports:
    - protocol: TCP
      port: 5001  # RAG API
    - protocol: TCP
      port: 8080  # LLM Processor
    - protocol: TCP
      port: 8080  # Weaviate
  
  # Allow DNS resolution
  - to:
    - namespaceSelector:
        matchLabels:
          name: "kube-system"
    ports:
    - protocol: UDP
      port: 53
    - protocol: TCP
      port: 53
  
  # Allow external API access (OpenAI, etc.)
  - to: []
    ports:
    - protocol: TCP
      port: 443
    - protocol: TCP
      port: 80

---
# RAG API access policy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: rag-api-tenant-access
  namespace: nephoran-system
spec:
  podSelector:
    matchLabels:
      app: rag-api
  policyTypes:
  - Ingress
  
  ingress:
  # Allow tenant namespaces to access RAG API
  - from:
    - namespaceSelector:
        matchLabels:
          nephoran.com/tenant: "enterprise_a"
    - namespaceSelector:
        matchLabels:
          nephoran.com/tenant: "operator_b"
    - namespaceSelector:
        matchLabels:
          nephoran.com/tenant: "development"
    ports:
    - protocol: TCP
      port: 5001
  
  # Allow LLM processor to access RAG API
  - from:
    - podSelector:
        matchLabels:
          app: llm-processor
    ports:
    - protocol: TCP
      port: 5001
```

## Security and Authentication

### 1. OAuth2 Multi-Tenant Configuration

```yaml
# Multi-tenant OAuth2 configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth2-multi-tenant-config
  namespace: nephoran-system
data:
  oauth2_config.yaml: |
    oauth2:
      multi_tenant: true
      tenant_claim: "tenant_id"
      issuer_validation: true
      
    tenant_issuers:
      enterprise_a:
        issuer: "https://auth.enterprise-a.com"
        audience: "nephoran-enterprise-a"
        client_id: "nephoran-ent-a-client"
        scopes: ["nephoran:read", "nephoran:write", "nephoran:admin"]
        jwks_uri: "https://auth.enterprise-a.com/.well-known/jwks.json"
        
      operator_b:
        issuer: "https://sso.operator-b.net"
        audience: "nephoran-operator-b"
        client_id: "nephoran-op-b-client"
        scopes: ["oran:manage", "core:deploy", "edge:admin"]
        jwks_uri: "https://sso.operator-b.net/.well-known/jwks.json"
        
      development:
        issuer: "https://dev-auth.nephoran.internal"
        audience: "nephoran-development"
        client_id: "nephoran-dev-client"
        scopes: ["dev:all"]
        jwks_uri: "https://dev-auth.nephoran.internal/.well-known/jwks.json"
        
    tenant_mapping:
      claim_extraction:
        tenant_claim: "tenant_id"
        namespace_claim: "namespace"
        roles_claim: "roles"
      
      role_mapping:
        "admin": ["nephoran:admin", "all:manage"]
        "operator": ["nephoran:write", "intent:manage"]
        "viewer": ["nephoran:read", "status:view"]

---
# OAuth2 proxy for multi-tenant authentication
apiVersion: apps/v1
kind: Deployment
metadata:
  name: oauth2-proxy-multi-tenant
  namespace: nephoran-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: oauth2-proxy
  template:
    metadata:
      labels:
        app: oauth2-proxy
    spec:
      serviceAccountName: oauth2-proxy
      containers:
      - name: oauth2-proxy
        image: quay.io/oauth2-proxy/oauth2-proxy:v7.4.0
        args:
        - --provider=oidc
        - --provider-display-name="Multi-Tenant OIDC"
        - --upstream=http://nephio-bridge.nephoran-system.svc.cluster.local:8080
        - --http-address=0.0.0.0:4180
        - --reverse-proxy=true
        - --pass-authorization-header=true
        - --pass-user-headers=true
        - --set-authorization-header=true
        - --set-xauthrequest=true
        - --cookie-secure=true
        - --cookie-httponly=true
        - --cookie-samesite=lax
        - --skip-provider-button=true
        env:
        - name: OAUTH2_PROXY_CLIENT_ID
          valueFrom:
            secretKeyRef:
              name: oauth2-proxy-secrets
              key: client_id
        - name: OAUTH2_PROXY_CLIENT_SECRET
          valueFrom:
            secretKeyRef:
              name: oauth2-proxy-secrets
              key: client_secret
        - name: OAUTH2_PROXY_COOKIE_SECRET
          valueFrom:
            secretKeyRef:
              name: oauth2-proxy-secrets
              key: cookie_secret
        - name: OAUTH2_PROXY_OIDC_ISSUER_URL
          value: "https://auth.nephoran.com"
        ports:
        - containerPort: 4180
          name: http
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
```

### 2. Certificate-Based Authentication

```yaml
# Multi-tenant certificate management
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: enterprise-a-client-cert
  namespace: nephoran-ent-a
spec:
  secretName: enterprise-a-client-tls
  issuerRef:
    name: nephoran-ca-issuer
    kind: ClusterIssuer
  commonName: "enterprise-a.nephoran.client"
  dnsNames:
  - "enterprise-a.nephoran.client"
  - "*.enterprise-a.nephoran.local"
  usages:
  - digital signature
  - key encipherment
  - client auth
  subject:
    organizations:
    - "Enterprise A Manufacturing"
    organizationalUnits:
    - "Network Operations"
    countries:
    - "US"
    localities:
    - "San Francisco"

---
# Tenant-specific certificate authority
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: nephoran-tenant-ca-issuer
spec:
  ca:
    secretName: nephoran-tenant-ca-key-pair
  
---
# Certificate-based authentication for tenant services
apiVersion: v1
kind: Secret
metadata:
  name: tenant-ca-certificates
  namespace: nephoran-system
type: Opaque
data:
  # Base64 encoded CA certificates for each tenant
  enterprise-a-ca.crt: LS0tLS1CRUdJTi... # Enterprise A CA
  operator-b-ca.crt: LS0tLS1CRUdJTi...   # Operator B CA  
  development-ca.crt: LS0tLS1CRUdJTi...  # Development CA
```

## Resource Management and Quotas

### 1. Tenant Resource Quotas

```yaml
# Advanced resource quota with custom metrics
apiVersion: v1
kind: ResourceQuota
metadata:
  name: enterprise-a-advanced-quota
  namespace: nephoran-ent-a
  labels:
    nephoran.com/tenant: "enterprise_a"
    nephoran.com/quota-tier: "enterprise"
spec:
  hard:
    # Standard Kubernetes resources
    requests.cpu: "50"
    requests.memory: "100Gi"
    requests.storage: "1Ti"
    limits.cpu: "100"
    limits.memory: "200Gi"
    
    # Extended resources
    requests.nvidia.com/gpu: "4"
    limits.nvidia.com/gpu: "8"
    
    # Network resources
    services: "50"
    services.loadbalancers: "10"
    services.nodeports: "5"
    ingresses.extensions: "20"
    
    # Custom Nephoran resources
    networkintents.nephoran.com: "500"
    e2nodesets.nephoran.com: "100"
    managedelements.nephoran.com: "1000"
    
    # Storage classes
    bronze.storageclass.storage.k8s.io/requests.storage: "500Gi"
    silver.storageclass.storage.k8s.io/requests.storage: "300Gi"
    gold.storageclass.storage.k8s.io/requests.storage: "100Gi"
    
  scopes:
  - Terminating
  - NotTerminating
  - BestEffort
  - NotBestEffort
  
  # Scope selectors for fine-grained control
  scopeSelector:
    matchExpressions:
    - operator: In
      scopeName: PriorityClass
      values: ["high-priority", "medium-priority"]

---
# Horizontal Pod Autoscaler quota
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: tenant-workload-hpa
  namespace: nephoran-ent-a
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: tenant-workload
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: intent_processing_rate
      target:
        type: AverageValue
        averageValue: "100"
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
```

### 2. Priority Classes for Tenant Workloads

```yaml
# High priority for enterprise tenant
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: nephoran-enterprise-high
value: 1000
globalDefault: false
description: "High priority for enterprise tenant workloads"

---
# Medium priority for operator tenant
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: nephoran-operator-medium
value: 500
globalDefault: false
description: "Medium priority for telecom operator workloads"

---
# Low priority for development tenant
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: nephoran-development-low
value: 100
globalDefault: false
description: "Low priority for development workloads"

---
# Tenant workload with priority class
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephio-bridge-enterprise-a
  namespace: nephoran-ent-a
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nephio-bridge
      tenant: enterprise-a
  template:
    metadata:
      labels:
        app: nephio-bridge
        tenant: enterprise-a
    spec:
      priorityClassName: nephoran-enterprise-high
      serviceAccountName: nephoran-enterprise-a-operator
      nodeSelector:
        node-tier: enterprise
      tolerations:
      - key: "tenant"
        operator: "Equal"
        value: "enterprise-a"
        effect: "NoSchedule"
      containers:
      - name: nephio-bridge
        image: registry.nephoran.com/nephio-bridge:v1.0.0
        resources:
          requests:
            cpu: "1000m"
            memory: "2Gi"
          limits:
            cpu: "2000m"
            memory: "4Gi"
        env:
        - name: TENANT_ID
          value: "enterprise_a"
        - name: TENANT_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
```

## RAG System Multi-Tenancy

### 1. Weaviate Multi-Tenant Schema

```python
# Multi-tenant Weaviate schema configuration
import weaviate
from typing import Dict, List, Any

class MultiTenantWeaviateManager:
    """Manages multi-tenant Weaviate deployments for Nephoran"""
    
    def __init__(self, client: weaviate.Client):
        self.client = client
        self.tenant_schemas = {}
    
    def create_tenant_schema(self, tenant_id: str, tenant_config: Dict[str, Any]) -> bool:
        """Create isolated schema for a specific tenant"""
        
        tenant_class_name = f"{tenant_config['weaviate_class_prefix']}TelecomKnowledge"
        
        schema = {
            "class": tenant_class_name,
            "description": f"Telecommunications knowledge base for tenant {tenant_id}",
            "vectorizer": "text2vec-openai",
            "moduleConfig": {
                "text2vec-openai": {
                    "model": "text-embedding-3-large",
                    "dimensions": 3072,
                    "type": "text"
                }
            },
            "properties": [
                {
                    "name": "content",
                    "dataType": ["text"],
                    "description": "Document content",
                    "moduleConfig": {
                        "text2vec-openai": {
                            "skip": False,
                            "vectorizePropertyName": False
                        }
                    }
                },
                {
                    "name": "tenant_id",
                    "dataType": ["text"],
                    "description": "Tenant identifier for isolation",
                    "moduleConfig": {
                        "text2vec-openai": {
                            "skip": True
                        }
                    }
                },
                {
                    "name": "source",
                    "dataType": ["text"],
                    "description": "Document source"
                },
                {
                    "name": "category",
                    "dataType": ["text"],
                    "description": "Knowledge category"
                },
                {
                    "name": "access_level",
                    "dataType": ["text"],
                    "description": "Access control level for the document"
                },
                {
                    "name": "tenant_metadata",
                    "dataType": ["object"],
                    "description": "Tenant-specific metadata",
                    "nestedProperties": [
                        {
                            "name": "department",
                            "dataType": ["text"]
                        },
                        {
                            "name": "classification",
                            "dataType": ["text"]
                        },
                        {
                            "name": "tags",
                            "dataType": ["text[]"]
                        }
                    ]
                }
            ],
            "vectorIndexConfig": {
                "distance": "cosine",
                "ef": 128,
                "efConstruction": 256,
                "maxConnections": 32
            },
            "invertedIndexConfig": {
                "bm25": {
                    "k1": 1.2,
                    "b": 0.75
                },
                "cleanupIntervalSeconds": 60
            }
        }
        
        try:
            self.client.schema.create_class(schema)
            self.tenant_schemas[tenant_id] = tenant_class_name
            return True
        except Exception as e:
            print(f"Failed to create schema for tenant {tenant_id}: {e}")
            return False
    
    def add_tenant_document(self, tenant_id: str, document: Dict[str, Any]) -> bool:
        """Add document to tenant-specific collection with access control"""
        
        if tenant_id not in self.tenant_schemas:
            raise ValueError(f"Tenant {tenant_id} schema not found")
        
        class_name = self.tenant_schemas[tenant_id]
        
        # Ensure tenant isolation
        document["tenant_id"] = tenant_id
        
        # Add tenant-specific metadata
        if "tenant_metadata" not in document:
            document["tenant_metadata"] = {}
        
        document["tenant_metadata"]["tenant_id"] = tenant_id
        document["tenant_metadata"]["created_at"] = datetime.now().isoformat()
        
        try:
            self.client.data_object.create(
                data_object=document,
                class_name=class_name
            )
            return True
        except Exception as e:
            print(f"Failed to add document for tenant {tenant_id}: {e}")
            return False
    
    def search_tenant_documents(self, tenant_id: str, query: str, limit: int = 10) -> List[Dict]:
        """Search documents within tenant boundaries"""
        
        if tenant_id not in self.tenant_schemas:
            raise ValueError(f"Tenant {tenant_id} schema not found")
        
        class_name = self.tenant_schemas[tenant_id]
        
        # Build tenant-filtered query
        where_filter = {
            "path": ["tenant_id"],
            "operator": "Equal",
            "valueText": tenant_id
        }
        
        try:
            result = self.client.query.get(class_name, [
                "content", "source", "category", "tenant_id", "access_level"
            ]).with_near_text({
                "concepts": [query],
                "certainty": 0.7
            }).with_where(where_filter).with_limit(limit).do()
            
            return result.get("data", {}).get("Get", {}).get(class_name, [])
            
        except Exception as e:
            print(f"Search failed for tenant {tenant_id}: {e}")
            return []
    
    def get_tenant_statistics(self, tenant_id: str) -> Dict[str, Any]:
        """Get statistics for a specific tenant"""
        
        if tenant_id not in self.tenant_schemas:
            return {"error": f"Tenant {tenant_id} not found"}
        
        class_name = self.tenant_schemas[tenant_id]
        
        try:
            # Get document count
            count_result = self.client.query.aggregate(class_name).with_meta_count().with_where({
                "path": ["tenant_id"],
                "operator": "Equal", 
                "valueText": tenant_id
            }).do()
            
            document_count = count_result.get("data", {}).get("Aggregate", {}).get(class_name, [{}])[0].get("meta", {}).get("count", 0)
            
            # Get source distribution
            source_result = self.client.query.aggregate(class_name).with_group_by_filter(["source"]).with_where({
                "path": ["tenant_id"],
                "operator": "Equal",
                "valueText": tenant_id
            }).do()
            
            return {
                "tenant_id": tenant_id,
                "document_count": document_count,
                "class_name": class_name,
                "source_distribution": source_result,
                "last_updated": datetime.now().isoformat()
            }
            
        except Exception as e:
            return {"error": f"Failed to get statistics: {e}"}

# Usage example
if __name__ == "__main__":
    client = weaviate.Client("http://weaviate.nephoran-system.svc.cluster.local:8080")
    manager = MultiTenantWeaviateManager(client)
    
    # Create tenant schemas
    tenant_configs = {
        "enterprise_a": {
            "weaviate_class_prefix": "EntA_",
            "access_level": "enterprise"
        },
        "operator_b": {
            "weaviate_class_prefix": "OpB_",
            "access_level": "operator"
        }
    }
    
    for tenant_id, config in tenant_configs.items():
        manager.create_tenant_schema(tenant_id, config)
```

### 2. Tenant-Aware RAG API

```python
# Enhanced RAG API with multi-tenant support
from flask import Flask, request, jsonify, g
import jwt
from functools import wraps
from typing import Dict, Optional

app = Flask(__name__)

class TenantContext:
    """Thread-local tenant context"""
    def __init__(self):
        self.tenant_id: Optional[str] = None
        self.tenant_config: Optional[Dict] = None
        self.access_level: Optional[str] = None
        self.resource_quotas: Optional[Dict] = None

tenant_context = TenantContext()

def extract_tenant_from_token(token: str) -> Optional[Dict]:
    """Extract tenant information from JWT token"""
    try:
        # Decode JWT without verification for tenant extraction
        # In production, use proper JWT verification
        decoded = jwt.decode(token, options={"verify_signature": False})
        
        return {
            "tenant_id": decoded.get("tenant_id"),
            "access_level": decoded.get("access_level"),
            "namespace": decoded.get("namespace"),
            "roles": decoded.get("roles", [])
        }
    except Exception as e:
        print(f"Token extraction failed: {e}")
        return None

def require_tenant_auth(f):
    """Decorator to require tenant authentication"""
    @wraps(f)
    def decorated_function(*args, **kwargs):
        auth_header = request.headers.get('Authorization')
        if not auth_header or not auth_header.startswith('Bearer '):
            return jsonify({"error": "Missing or invalid authorization header"}), 401
        
        token = auth_header.split(' ')[1]
        tenant_info = extract_tenant_from_token(token)
        
        if not tenant_info or not tenant_info.get("tenant_id"):
            return jsonify({"error": "Invalid tenant information"}), 401
        
        # Set tenant context
        tenant_context.tenant_id = tenant_info["tenant_id"]
        tenant_context.access_level = tenant_info.get("access_level")
        
        # Load tenant configuration
        tenant_context.tenant_config = load_tenant_config(tenant_context.tenant_id)
        if not tenant_context.tenant_config:
            return jsonify({"error": "Tenant configuration not found"}), 404
        
        return f(*args, **kwargs)
    
    return decorated_function

def load_tenant_config(tenant_id: str) -> Optional[Dict]:
    """Load tenant configuration from ConfigMap or database"""
    # In production, this would load from Kubernetes ConfigMap or database
    tenant_configs = {
        "enterprise_a": {
            "max_queries_per_hour": 10000,
            "max_concurrent_queries": 50,
            "allowed_sources": ["3GPP", "O-RAN", "Internal"],
            "weaviate_class": "EntA_TelecomKnowledge"
        },
        "operator_b": {
            "max_queries_per_hour": 50000,
            "max_concurrent_queries": 200,
            "allowed_sources": ["3GPP", "O-RAN", "ETSI", "ITU"],
            "weaviate_class": "OpB_TelecomKnowledge"
        },
        "development": {
            "max_queries_per_hour": 1000,
            "max_concurrent_queries": 10,
            "allowed_sources": ["3GPP", "O-RAN", "Test"],
            "weaviate_class": "Dev_TelecomKnowledge"
        }
    }
    
    return tenant_configs.get(tenant_id)

@app.route('/process_intent', methods=['POST'])
@require_tenant_auth
def process_intent():
    """Process intent with tenant isolation"""
    
    try:
        data = request.get_json()
        if not data or 'intent' not in data:
            return jsonify({"error": "Missing 'intent' in request body"}), 400
        
        intent = data['intent']
        tenant_id = tenant_context.tenant_id
        
        # Apply tenant-specific processing
        processing_result = process_tenant_intent(
            intent=intent,
            tenant_id=tenant_id,
            tenant_config=tenant_context.tenant_config
        )
        
        # Add tenant metadata to response
        processing_result["tenant_metadata"] = {
            "tenant_id": tenant_id,
            "access_level": tenant_context.access_level,
            "processing_timestamp": datetime.now().isoformat()
        }
        
        return jsonify(processing_result)
        
    except Exception as e:
        return jsonify({
            "error": "Failed to process intent",
            "tenant_id": tenant_context.tenant_id,
            "exception": str(e)
        }), 500

def process_tenant_intent(intent: str, tenant_id: str, tenant_config: Dict) -> Dict:
    """Process intent with tenant-specific constraints"""
    
    # Use tenant-specific Weaviate class
    weaviate_class = tenant_config["weaviate_class"]
    
    # Apply tenant-specific filters
    search_filters = {
        "tenant_id": tenant_id,
        "allowed_sources": tenant_config["allowed_sources"]
    }
    
    # Search within tenant boundaries
    search_results = search_tenant_knowledge(
        query=intent,
        weaviate_class=weaviate_class,
        filters=search_filters,
        limit=10
    )
    
    # Generate response using tenant-filtered context
    llm_response = generate_tenant_response(
        intent=intent,
        search_results=search_results,
        tenant_constraints=tenant_config
    )
    
    return {
        "intent_id": f"{tenant_id}-{int(time.time())}",
        "original_intent": intent,
        "structured_output": llm_response,
        "source_documents": search_results,
        "tenant_applied_filters": search_filters
    }

@app.route('/tenant/stats', methods=['GET'])
@require_tenant_auth
def get_tenant_stats():
    """Get tenant-specific statistics"""
    
    tenant_id = tenant_context.tenant_id
    
    try:
        stats = {
            "tenant_id": tenant_id,
            "configuration": tenant_context.tenant_config,
            "usage_statistics": get_tenant_usage_stats(tenant_id),
            "quota_status": check_tenant_quotas(tenant_id),
            "last_updated": datetime.now().isoformat()
        }
        
        return jsonify(stats)
        
    except Exception as e:
        return jsonify({
            "error": "Failed to retrieve tenant statistics",
            "tenant_id": tenant_id,
            "exception": str(e)
        }), 500

@app.route('/tenant/health', methods=['GET'])
@require_tenant_auth
def get_tenant_health():
    """Get tenant-specific health status"""
    
    tenant_id = tenant_context.tenant_id
    
    health_status = {
        "tenant_id": tenant_id,
        "status": "healthy",
        "checks": {
            "weaviate_access": check_weaviate_tenant_access(tenant_id),
            "quota_compliance": check_quota_compliance(tenant_id),
            "authentication": "verified",
            "configuration": "loaded"
        },
        "timestamp": datetime.now().isoformat()
    }
    
    # Determine overall health
    if not all(health_status["checks"].values()):
        health_status["status"] = "unhealthy"
    
    status_code = 200 if health_status["status"] == "healthy" else 503
    return jsonify(health_status), status_code

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=5001, debug=False)
```

## Deployment Patterns

### 1. Shared Infrastructure with Tenant Isolation

```yaml
# Shared Nephoran deployment with tenant routing
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephio-bridge-shared
  namespace: nephoran-system
  labels:
    app: nephio-bridge
    deployment-model: shared
spec:
  replicas: 5
  selector:
    matchLabels:
      app: nephio-bridge
  template:
    metadata:
      labels:
        app: nephio-bridge
    spec:
      serviceAccountName: nephio-bridge-shared
      containers:
      - name: nephio-bridge
        image: registry.nephoran.com/nephio-bridge:v1.0.0
        args:
        - --multi-tenant=true
        - --tenant-isolation=strict
        - --tenant-discovery=namespace-label
        - --audit-logging=true
        env:
        - name: MULTI_TENANT_MODE
          value: "true"
        - name: TENANT_ROUTER_CONFIG
          value: "/etc/config/tenant-router.yaml"
        - name: RBAC_ENFORCEMENT
          value: "true"
        resources:
          requests:
            cpu: "1000m"
            memory: "2Gi"
          limits:
            cpu: "2000m"
            memory: "4Gi"
        volumeMounts:
        - name: tenant-config
          mountPath: /etc/config
          readOnly: true
        - name: tenant-certificates
          mountPath: /etc/certs
          readOnly: true
      volumes:
      - name: tenant-config
        configMap:
          name: tenant-router-config
      - name: tenant-certificates
        secret:
          secretName: tenant-ca-certificates

---
# Tenant router configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: tenant-router-config
  namespace: nephoran-system
data:
  tenant-router.yaml: |
    tenant_routing:
      default_tenant: "system"
      isolation_mode: "strict"
      
    tenants:
      enterprise_a:
        namespace: "nephoran-ent-a"
        authentication:
          - method: "oauth2"
            issuer: "https://auth.enterprise-a.com"
          - method: "client_cert"
            ca_cert: "/etc/certs/enterprise-a-ca.crt"
        authorization:
          roles:
            - "admin"
            - "operator"
            - "viewer"
        resource_limits:
          max_concurrent_requests: 100
          rate_limit_rps: 50
          
      operator_b:
        namespace: "nephoran-op-b"
        authentication:
          - method: "oauth2"
            issuer: "https://sso.operator-b.net"
        authorization:
          roles:
            - "oran-admin"
            - "core-operator"
        resource_limits:
          max_concurrent_requests: 500
          rate_limit_rps: 200
```

### 2. Dedicated Instance per Tenant

```yaml
# Dedicated Nephoran instance for high-security tenant
apiVersion: v1
kind: Namespace
metadata:
  name: nephoran-dedicated-enterprise-a
  labels:
    nephoran.com/tenant: "enterprise_a"
    nephoran.com/deployment-model: "dedicated"
    nephoran.com/isolation-level: "maximum"

---
# Dedicated controller instance
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephio-bridge-dedicated-enterprise-a
  namespace: nephoran-dedicated-enterprise-a
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nephio-bridge
      tenant: enterprise-a
  template:
    metadata:
      labels:
        app: nephio-bridge
        tenant: enterprise-a
    spec:
      serviceAccountName: nephio-bridge-enterprise-a
      nodeSelector:
        tenant: enterprise-a
      tolerations:
      - key: "dedicated"
        operator: "Equal"
        value: "enterprise-a"
        effect: "NoSchedule"
      containers:
      - name: nephio-bridge
        image: registry.nephoran.com/nephio-bridge:v1.0.0
        args:
        - --single-tenant-mode=true
        - --tenant-id=enterprise_a
        - --namespace=nephoran-dedicated-enterprise-a
        env:
        - name: TENANT_ID
          value: "enterprise_a"
        - name: SINGLE_TENANT_MODE
          value: "true"
        - name: DEDICATED_RESOURCES
          value: "true"
        resources:
          requests:
            cpu: "2000m"
            memory: "4Gi"
          limits:
            cpu: "4000m"
            memory: "8Gi"

---
# Dedicated Weaviate instance
apiVersion: apps/v1
kind: Deployment
metadata:
  name: weaviate-dedicated-enterprise-a
  namespace: nephoran-dedicated-enterprise-a
spec:
  replicas: 3
  selector:
    matchLabels:
      app: weaviate
      tenant: enterprise-a
  template:
    metadata:
      labels:
        app: weaviate
        tenant: enterprise-a
    spec:
      containers:
      - name: weaviate
        image: semitechnologies/weaviate:1.28.0
        env:
        - name: SINGLE_TENANT_MODE
          value: "true"
        - name: TENANT_ID
          value: "enterprise_a"
        - name: AUTHENTICATION_APIKEY_ENABLED
          value: "true"
        - name: AUTHENTICATION_APIKEY_ALLOWED_KEYS
          value: "enterprise-a-dedicated-key"
        resources:
          requests:
            cpu: "4000m"
            memory: "8Gi"
          limits:
            cpu: "8000m"
            memory: "16Gi"
        volumeMounts:
        - name: weaviate-data
          mountPath: /var/lib/weaviate
      volumes:
      - name: weaviate-data
        persistentVolumeClaim:
          claimName: weaviate-dedicated-enterprise-a-pvc
```

## Monitoring and Observability

### 1. Tenant-Aware Monitoring

```yaml
# Prometheus configuration for multi-tenant monitoring
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-multi-tenant-config
  namespace: monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      external_labels:
        cluster: 'nephoran-multi-tenant'
        
    rule_files:
    - "/etc/prometheus/rules/*.yml"
    
    scrape_configs:
    # Tenant-specific application metrics
    - job_name: 'nephoran-enterprise-a'
      kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
          - nephoran-ent-a
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__
      - action: labelmap
        regex: __meta_kubernetes_pod_label_(.+)
      - source_labels: [__meta_kubernetes_namespace]
        action: replace
        target_label: kubernetes_namespace
      - source_labels: [__meta_kubernetes_pod_name]
        action: replace
        target_label: kubernetes_pod_name
      - target_label: tenant_id
        replacement: 'enterprise_a'
    
    # Shared system metrics
    - job_name: 'nephoran-system'
      kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
          - nephoran-system
      relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - target_label: tenant_id
        replacement: 'system'
    
    # Multi-tenant alerting rules
    alerting:
      alertmanagers:
      - kubernetes_sd_configs:
        - role: pod
          namespaces:
            names:
            - monitoring
        relabel_configs:
        - source_labels: [__meta_kubernetes_pod_label_app]
          action: keep
          regex: alertmanager

---
# Tenant-specific alerting rules
apiVersion: v1
kind: ConfigMap
metadata:
  name: tenant-alerting-rules
  namespace: monitoring
data:
  tenant-alerts.yml: |
    groups:
    - name: tenant.enterprise_a
      rules:
      - alert: TenantResourceQuotaExceeded
        expr: |
          (
            kube_resourcequota{namespace="nephoran-ent-a", resource="requests.cpu", type="used"} /
            kube_resourcequota{namespace="nephoran-ent-a", resource="requests.cpu", type="hard"}
          ) > 0.9
        for: 5m
        labels:
          severity: warning
          tenant_id: enterprise_a
        annotations:
          summary: "Tenant Enterprise A is approaching CPU quota limit"
          description: "Tenant Enterprise A is using {{ $value | humanizePercentage }} of CPU quota"
      
      - alert: TenantHighLatency
        expr: |
          histogram_quantile(0.95, 
            rate(intent_processing_duration_seconds_bucket{tenant_id="enterprise_a"}[5m])
          ) > 5
        for: 2m
        labels:
          severity: critical
          tenant_id: enterprise_a
        annotations:
          summary: "High intent processing latency for Enterprise A"
          description: "95th percentile latency is {{ $value }}s for tenant Enterprise A"
      
      - alert: TenantRAGServiceDown
        expr: |
          up{job="nephoran-enterprise-a", app="rag-api"} == 0
        for: 1m
        labels:
          severity: critical
          tenant_id: enterprise_a
        annotations:
          summary: "RAG service is down for tenant Enterprise A"
          description: "RAG API service for tenant Enterprise A has been down for more than 1 minute"

---
# Grafana tenant dashboard ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-tenant-dashboards
  namespace: monitoring
data:
  enterprise-a-dashboard.json: |
    {
      "dashboard": {
        "id": null,
        "title": "Enterprise A - Network Operations",
        "tags": ["nephoran", "enterprise-a", "tenant"],
        "timezone": "browser",
        "panels": [
          {
            "id": 1,
            "title": "Intent Processing Rate",
            "type": "graph",
            "targets": [
              {
                "expr": "rate(intent_processing_total{tenant_id=\"enterprise_a\"}[5m])",
                "legendFormat": "Intents/sec"
              }
            ],
            "yAxes": [
              {
                "label": "Intents per second",
                "min": 0
              }
            ]
          },
          {
            "id": 2,
            "title": "Resource Utilization",
            "type": "graph",
            "targets": [
              {
                "expr": "container_memory_usage_bytes{namespace=\"nephoran-ent-a\"} / container_spec_memory_limit_bytes",
                "legendFormat": "Memory Usage %"
              },
              {
                "expr": "rate(container_cpu_usage_seconds_total{namespace=\"nephoran-ent-a\"}[5m])",
                "legendFormat": "CPU Usage"
              }
            ]
          },
          {
            "id": 3,
            "title": "NetworkIntent Status",
            "type": "table",
            "targets": [
              {
                "expr": "kube_customresource_info{customresource_group=\"nephoran.com\", customresource_kind=\"NetworkIntent\", namespace=\"nephoran-ent-a\"}",
                "format": "table"
              }
            ]
          },
          {
            "id": 4,
            "title": "RAG Query Performance",
            "type": "histogram",
            "targets": [
              {
                "expr": "histogram_quantile(0.50, rate(rag_query_duration_seconds_bucket{tenant_id=\"enterprise_a\"}[5m]))",
                "legendFormat": "50th percentile"
              },
              {
                "expr": "histogram_quantile(0.95, rate(rag_query_duration_seconds_bucket{tenant_id=\"enterprise_a\"}[5m]))",
                "legendFormat": "95th percentile"
              }
            ]
          }
        ],
        "time": {
          "from": "now-1h",
          "to": "now"
        },
        "refresh": "5s"
      }
    }
```

## Backup and Recovery

### 1. Tenant-Specific Backup Strategies

```yaml
# Per-tenant backup CronJob
apiVersion: batch/v1
kind: CronJob
metadata:
  name: tenant-backup-enterprise-a
  namespace: nephoran-ent-a
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: tenant-backup-sa
          containers:
          - name: backup
            image: registry.nephoran.com/backup-agent:v1.0.0
            env:
            - name: TENANT_ID
              value: "enterprise_a"
            - name: BACKUP_TYPE
              value: "tenant_full"
            - name: BACKUP_DESTINATION
              value: "s3://nephoran-backups/tenants/enterprise_a"
            - name: ENCRYPTION_KEY_REF
              valueFrom:
                secretKeyRef:
                  name: backup-encryption-keys
                  key: enterprise-a-key
            command:
            - /bin/bash
            - -c
            - |
              set -e
              
              echo "Starting backup for tenant enterprise_a at $(date)"
              
              # Backup CRD instances
              echo "Backing up NetworkIntents..."
              kubectl get networkintents -n nephoran-ent-a -o yaml > /tmp/networkintents.yaml
              
              echo "Backing up E2NodeSets..."
              kubectl get e2nodesets -n nephoran-ent-a -o yaml > /tmp/e2nodesets.yaml
              
              echo "Backing up ManagedElements..."
              kubectl get managedelements -n nephoran-ent-a -o yaml > /tmp/managedelements.yaml
              
              # Backup ConfigMaps and Secrets
              echo "Backing up ConfigMaps..."
              kubectl get configmaps -n nephoran-ent-a -o yaml > /tmp/configmaps.yaml
              
              echo "Backing up Secrets..."
              kubectl get secrets -n nephoran-ent-a -o yaml > /tmp/secrets.yaml
              
              # Backup Weaviate tenant data
              echo "Backing up Weaviate tenant data..."
              curl -X POST "http://weaviate.nephoran-system.svc.cluster.local:8080/v1/backups/filesystem" \
                -H "Content-Type: application/json" \
                -d '{
                  "id": "enterprise-a-'$(date +%Y%m%d-%H%M%S)'",
                  "include": ["EntA_TelecomKnowledge"],
                  "compression": "gzip"
                }'
              
              # Create backup archive
              echo "Creating backup archive..."
              tar -czf /tmp/tenant-enterprise-a-backup-$(date +%Y%m%d-%H%M%S).tar.gz \
                /tmp/*.yaml \
                /var/lib/weaviate/backups/enterprise-a-*
              
              # Upload to S3 with encryption
              echo "Uploading backup to S3..."
              aws s3 cp /tmp/tenant-enterprise-a-backup-*.tar.gz \
                s3://nephoran-backups/tenants/enterprise_a/ \
                --server-side-encryption AES256
              
              # Cleanup local files
              rm -f /tmp/*.yaml /tmp/*.tar.gz
              
              echo "Backup completed successfully at $(date)"
            resources:
              requests:
                cpu: 500m
                memory: 1Gi
              limits:
                cpu: 1000m
                memory: 2Gi
            volumeMounts:
            - name: backup-storage
              mountPath: /backup
          volumes:
          - name: backup-storage
            persistentVolumeClaim:
              claimName: backup-pvc
          restartPolicy: OnFailure

---
# Tenant backup restoration job
apiVersion: batch/v1
kind: Job
metadata:
  name: tenant-restore-enterprise-a
  namespace: nephoran-ent-a
spec:
  template:
    spec:
      serviceAccountName: tenant-backup-sa
      containers:
      - name: restore
        image: registry.nephoran.com/backup-agent:v1.0.0
        env:
        - name: TENANT_ID
          value: "enterprise_a"
        - name: RESTORE_FROM
          value: "s3://nephoran-backups/tenants/enterprise_a/tenant-enterprise-a-backup-20240728-020000.tar.gz"
        - name: ENCRYPTION_KEY_REF
          valueFrom:
            secretKeyRef:
              name: backup-encryption-keys
              key: enterprise-a-key
        command:
        - /bin/bash
        - -c
        - |
          set -e
          
          echo "Starting restore for tenant enterprise_a at $(date)"
          
          # Download backup archive
          echo "Downloading backup from S3..."
          aws s3 cp $RESTORE_FROM /tmp/backup.tar.gz
          
          # Extract backup
          echo "Extracting backup archive..."
          cd /tmp && tar -xzf backup.tar.gz
          
          # Restore CRD instances
          echo "Restoring NetworkIntents..."
          kubectl apply -f /tmp/networkintents.yaml
          
          echo "Restoring E2NodeSets..."
          kubectl apply -f /tmp/e2nodesets.yaml
          
          echo "Restoring ManagedElements..."
          kubectl apply -f /tmp/managedelements.yaml
          
          # Restore ConfigMaps and Secrets
          echo "Restoring ConfigMaps..."
          kubectl apply -f /tmp/configmaps.yaml
          
          echo "Restoring Secrets..."
          kubectl apply -f /tmp/secrets.yaml
          
          # Restore Weaviate data
          echo "Restoring Weaviate tenant data..."
          BACKUP_ID=$(basename $RESTORE_FROM .tar.gz | sed 's/tenant-enterprise-a-backup-//')
          curl -X POST "http://weaviate.nephoran-system.svc.cluster.local:8080/v1/backups/filesystem/enterprise-a-$BACKUP_ID/restore" \
            -H "Content-Type: application/json" \
            -d '{
              "include": ["EntA_TelecomKnowledge"],
              "strategy": "merge"
            }'
          
          # Cleanup
          rm -rf /tmp/*
          
          echo "Restore completed successfully at $(date)"
      restartPolicy: OnFailure
```

## Troubleshooting Multi-Tenant Issues

### 1. Common Multi-Tenant Problems

#### Tenant Isolation Violations

```bash
# Check for tenant isolation violations
kubectl get networkpolicies -A
kubectl describe networkpolicy tenant-enterprise-a-isolation -n nephoran-ent-a

# Verify RBAC permissions
kubectl auth can-i get networkintents --as=system:serviceaccount:nephoran-ent-a:nephoran-enterprise-a-operator -n nephoran-op-b
# Should return "no" for proper isolation

# Check resource quotas
kubectl describe resourcequota tenant-enterprise-a-quota -n nephoran-ent-a
kubectl top pods -n nephoran-ent-a
```

#### Authentication Issues

```bash
# Debug OAuth2 authentication
kubectl logs deployment/oauth2-proxy-multi-tenant -n nephoran-system

# Check JWT token claims
echo "$JWT_TOKEN" | jwt decode -

# Verify certificate authentication
openssl x509 -in /etc/certs/enterprise-a-client.crt -text -noout
openssl verify -CAfile /etc/certs/ca.crt /etc/certs/enterprise-a-client.crt
```

#### RAG System Tenant Issues

```bash
# Check Weaviate tenant data
kubectl exec -it deployment/weaviate -n nephoran-system -- \
  curl -X GET "http://localhost:8080/v1/schema" | jq '.classes[] | select(.class | startswith("EntA_"))'

# Verify tenant-specific queries
kubectl port-forward svc/rag-api 5001:5001 -n nephoran-system
curl -H "Authorization: Bearer $TENANT_TOKEN" \
  -X POST http://localhost:5001/process_intent \
  -d '{"intent": "test query"}'

# Check tenant statistics
curl -H "Authorization: Bearer $TENANT_TOKEN" \
  http://localhost:5001/tenant/stats
```

### 2. Monitoring and Alerting for Multi-Tenant Issues

```yaml
# Multi-tenant specific monitoring rules
apiVersion: v1
kind: ConfigMap
metadata:
  name: multi-tenant-monitoring-rules
  namespace: monitoring
data:
  multi-tenant-rules.yml: |
    groups:
    - name: multi_tenant_violations
      rules:
      - alert: CrossTenantResourceAccess
        expr: |
          increase(kubernetes_audit_total{
            verb!="get",
            objectRef_namespace!="",
            user_username!~"system:.*"
          }[5m]) > 0
        for: 0m
        labels:
          severity: critical
        annotations:
          summary: "Potential cross-tenant resource access detected"
          description: "User {{ $labels.user_username }} accessed resources in namespace {{ $labels.objectRef_namespace }}"
      
      - alert: TenantQuotaViolation
        expr: |
          (
            kube_resourcequota{type="used"} / 
            kube_resourcequota{type="hard"}
          ) > 1
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Tenant quota violation in {{ $labels.namespace }}"
          description: "Resource {{ $labels.resource }} usage exceeds quota in namespace {{ $labels.namespace }}"
      
      - alert: NetworkPolicyViolation
        expr: |
          increase(cilium_policy_verdict_total{verdict="DENIED"}[5m]) > 10
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Increased network policy denials detected"
          description: "{{ $value }} network policy denials in the last 5 minutes"
```

### 3. Debugging Tools and Scripts

```bash
#!/bin/bash
# Multi-tenant debugging script

TENANT_ID=${1:-"enterprise_a"}
NAMESPACE="nephoran-${TENANT_ID//_/-}"

echo "=== Multi-Tenant Debug Report for Tenant: $TENANT_ID ==="
echo "Namespace: $NAMESPACE"
echo "Timestamp: $(date)"
echo

echo "=== Namespace Status ==="
kubectl get namespace $NAMESPACE -o yaml
echo

echo "=== Resource Quotas ==="
kubectl describe resourcequota -n $NAMESPACE
echo

echo "=== Network Policies ==="
kubectl get networkpolicies -n $NAMESPACE
kubectl describe networkpolicy -n $NAMESPACE
echo

echo "=== RBAC Permissions ==="
kubectl describe rolebinding -n $NAMESPACE
kubectl describe clusterrolebinding | grep -A 10 -B 2 $NAMESPACE
echo

echo "=== Pod Status ==="
kubectl get pods -n $NAMESPACE -o wide
echo

echo "=== Events ==="
kubectl get events -n $NAMESPACE --sort-by='.firstTimestamp'
echo

echo "=== RAG System Status ==="
kubectl port-forward svc/rag-api 5001:5001 -n nephoran-system &
PF_PID=$!
sleep 2

if curl -s -f http://localhost:5001/health >/dev/null; then
    echo "RAG API is accessible"
    # Test tenant-specific endpoint if token available
    if [ ! -z "$TENANT_TOKEN" ]; then
        echo "Testing tenant-specific access..."
        curl -s -H "Authorization: Bearer $TENANT_TOKEN" \
             http://localhost:5001/tenant/health | jq '.'
    fi
else
    echo "RAG API is not accessible"
fi

kill $PF_PID 2>/dev/null
echo

echo "=== Weaviate Tenant Data ==="
kubectl exec -it deployment/weaviate -n nephoran-system -- \
  curl -s "http://localhost:8080/v1/schema" | \
  jq --arg prefix "${TENANT_ID^^}_" '.classes[] | select(.class | startswith($prefix))'
echo

echo "=== Recent Logs ==="
echo "Controller logs:"
kubectl logs deployment/nephio-bridge -n nephoran-system --tail=50 | grep -i $TENANT_ID
echo

echo "RAG API logs:"
kubectl logs deployment/rag-api -n nephoran-system --tail=50 | grep -i $TENANT_ID
echo

echo "=== Debug Report Complete ==="
```

This comprehensive multi-tenant configuration guide provides enterprise-grade isolation, security, and management capabilities for the Nephoran Intent Operator RAG system. The configuration supports various deployment models from shared infrastructure with strict isolation to dedicated instances for high-security requirements.