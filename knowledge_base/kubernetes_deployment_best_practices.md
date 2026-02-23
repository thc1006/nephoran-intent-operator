# Kubernetes Deployment Best Practices for Nephoran Intent Operator

## Overview
This document provides Kubernetes deployment best practices specifically tailored for 5G network function deployments using the Nephoran Intent Operator with Kubernetes 1.35.1.

## Resource Management

### CPU and Memory Requests/Limits

#### Best Practice Pattern
Always specify both `requests` and `limits` for predictable resource allocation:

```yaml
resources:
  requests:
    cpu: "500m"       # Guaranteed CPU allocation
    memory: "512Mi"   # Guaranteed memory allocation
  limits:
    cpu: "1000m"      # Maximum CPU burst capacity
    memory: "1Gi"     # Maximum memory (OOMKilled if exceeded)
```

#### 5G Network Function Resource Guidelines

| Network Function | CPU Request | CPU Limit | Memory Request | Memory Limit | Notes |
|-----------------|-------------|-----------|----------------|--------------|-------|
| **AMF** | 300m | 500m | 384Mi | 512Mi | + 100m CPU per 500 UEs |
| **SMF** | 300m | 500m | 384Mi | 512Mi | + 200m CPU per 500 PDU sessions |
| **UPF** | 2000m | 4000m | 2Gi | 4Gi | Data plane intensive |
| **NRF** | 200m | 500m | 256Mi | 512Mi | Service discovery |
| **UDM/AUSF** | 200m | 400m | 256Mi | 512Mi | Auth services |
| **MongoDB** | 1000m | 2000m | 2Gi | 4Gi | Database backend |

### GPU Resource Allocation (DRA - GA in K8s 1.34+)

#### Dynamic Resource Allocation (DRA) for GPUs
Kubernetes 1.35.1 supports DRA GA for advanced GPU sharing:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: llm-processor
  namespace: rag-service
spec:
  containers:
  - name: ollama
    image: ollama/ollama:latest
    resources:
      claims:
      - name: gpu-claim
  resourceClaims:
  - name: gpu-claim
    resourceClaimTemplateName: gpu-claim-template
---
apiVersion: resource.k8s.io/v1alpha3
kind: ResourceClaimTemplate
metadata:
  name: gpu-claim-template
  namespace: rag-service
spec:
  spec:
    devices:
      requests:
      - name: gpu
        deviceClassName: nvidia-gpu
        count: 1
```

#### Legacy GPU Allocation (Device Plugin)
For pre-DRA environments:

```yaml
resources:
  requests:
    nvidia.com/gpu: 1
  limits:
    nvidia.com/gpu: 1
```

## High Availability Patterns

### Multi-Replica Deployments with Anti-Affinity

Ensure replicas are distributed across nodes to prevent single point of failure:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: amf
  namespace: free5gc
spec:
  replicas: 3
  selector:
    matchLabels:
      app: amf
  template:
    metadata:
      labels:
        app: amf
        tier: control-plane
    spec:
      affinity:
        podAntiAffinity:
          # Prefer different nodes
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - amf
              topologyKey: kubernetes.io/hostname
          # Require different availability zones (if multi-zone cluster)
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - amf
            topologyKey: topology.kubernetes.io/zone
      containers:
      - name: amf
        image: free5gc/amf:v3.4.3
        # ... container spec ...
```

### Pod Disruption Budgets

Protect critical services during node maintenance or upgrades:

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: amf-pdb
  namespace: free5gc
spec:
  minAvailable: 2  # Always keep at least 2 AMF instances running
  selector:
    matchLabels:
      app: amf
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: nrf-pdb
  namespace: free5gc
spec:
  maxUnavailable: 1  # Allow only 1 NRF instance to be down
  selector:
    matchLabels:
      app: nrf
```

### StatefulSet for Stateful Services

Use StatefulSets for databases and services requiring stable network identity:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mongodb
  namespace: free5gc
spec:
  serviceName: mongodb
  replicas: 3
  selector:
    matchLabels:
      app: mongodb
  template:
    metadata:
      labels:
        app: mongodb
    spec:
      containers:
      - name: mongodb
        image: mongo:8.0
        ports:
        - containerPort: 27017
          name: mongo
        volumeMounts:
        - name: mongo-data
          mountPath: /data/db
        env:
        - name: MONGO_INITDB_ROOT_USERNAME
          valueFrom:
            secretKeyRef:
              name: mongodb-secret
              key: username
        - name: MONGO_INITDB_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: mongodb-secret
              key: password
  volumeClaimTemplates:
  - metadata:
      name: mongo-data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      storageClassName: local-path
      resources:
        requests:
          storage: 10Gi
```

## Health Checks and Probes

### Liveness, Readiness, and Startup Probes

#### Liveness Probe
Determines if container should be restarted (detect deadlocks):

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8000
  initialDelaySeconds: 30  # Wait 30s before first check
  periodSeconds: 10        # Check every 10s
  timeoutSeconds: 5        # 5s timeout per check
  failureThreshold: 3      # Restart after 3 consecutive failures
```

#### Readiness Probe
Determines if pod should receive traffic (detect startup delays):

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8000
  initialDelaySeconds: 15
  periodSeconds: 5
  timeoutSeconds: 3
  successThreshold: 1      # Ready after 1 success
  failureThreshold: 3      # Not ready after 3 failures
```

#### Startup Probe
For slow-starting applications (prevents liveness from killing during startup):

```yaml
startupProbe:
  httpGet:
    path: /health
    port: 8000
  initialDelaySeconds: 0
  periodSeconds: 10
  timeoutSeconds: 3
  failureThreshold: 30     # Allow up to 300s (30 * 10s) for startup
```

### Example: 5G NRF with All Probes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nrf
  namespace: free5gc
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nrf
  template:
    metadata:
      labels:
        app: nrf
    spec:
      containers:
      - name: nrf
        image: free5gc/nrf:v3.4.3
        ports:
        - containerPort: 8000
          name: sbi
        startupProbe:
          httpGet:
            path: /nnrf-nfm/v1/nf-instances
            port: 8000
          initialDelaySeconds: 0
          periodSeconds: 10
          failureThreshold: 30  # 300s total startup window
        livenessProbe:
          httpGet:
            path: /nnrf-nfm/v1/nf-instances
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /nnrf-nfm/v1/nf-instances
            port: 8000
          initialDelaySeconds: 15
          periodSeconds: 5
          timeoutSeconds: 3
          successThreshold: 1
          failureThreshold: 3
        resources:
          requests:
            cpu: 200m
            memory: 256Mi
          limits:
            cpu: 500m
            memory: 512Mi
```

## ConfigMap and Secret Management

### ConfigMap for Application Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: amf-config
  namespace: free5gc
data:
  amfcfg.yaml: |
    info:
      version: 1.0.0
      description: AMF configuration
    configuration:
      amfName: AMF
      ngapIpList:
        - 10.100.200.2
      sbi:
        scheme: http
        registerIPv4: amf
        bindingIPv4: 0.0.0.0
        port: 8000
      serviceNameList:
        - namf-comm
        - namf-evts
        - namf-mt
        - namf-loc
        - namf-oam
      servedGuamiList:
        - plmnId:
            mcc: 208
            mnc: 93
          amfId: cafe00
      supportTaiList:
        - plmnId:
            mcc: 208
            mnc: 93
          tac: 000001
      plmnSupportList:
        - plmnId:
            mcc: 208
            mnc: 93
          snssaiList:
            - sst: 1
              sd: 010203
            - sst: 1
              sd: 112233
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: amf
  namespace: free5gc
spec:
  template:
    spec:
      containers:
      - name: amf
        image: free5gc/amf:v3.4.3
        volumeMounts:
        - name: config
          mountPath: /free5gc/config/amfcfg.yaml
          subPath: amfcfg.yaml
      volumes:
      - name: config
        configMap:
          name: amf-config
```

### Secret for Sensitive Data

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mongodb-secret
  namespace: free5gc
type: Opaque
stringData:
  username: free5gc_admin
  password: "MySecurePassword123!"
  connection_string: "mongodb://free5gc_admin:MySecurePassword123!@mongodb-0.mongodb:27017,mongodb-1.mongodb:27017,mongodb-2.mongodb:27017/free5gc?replicaSet=rs0"
```

**Usage in Pod**:
```yaml
env:
- name: DB_URI
  valueFrom:
    secretKeyRef:
      name: mongodb-secret
      key: connection_string
```

## Network Policies

### Restrict Traffic Between Namespaces

Only allow specific services to communicate:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: free5gc-network-policy
  namespace: free5gc
spec:
  podSelector:
    matchLabels:
      tier: control-plane
  policyTypes:
  - Ingress
  - Egress
  ingress:
  # Allow traffic from other control plane components
  - from:
    - namespaceSelector:
        matchLabels:
          name: free5gc
      podSelector:
        matchLabels:
          tier: control-plane
    ports:
    - protocol: TCP
      port: 8000  # SBI port
  # Allow traffic from O-RAN RIC (A1 interface)
  - from:
    - namespaceSelector:
        matchLabels:
          name: ricplt
    ports:
    - protocol: TCP
      port: 8000
  egress:
  # Allow DNS
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
      podSelector:
        matchLabels:
          k8s-app: kube-dns
    ports:
    - protocol: UDP
      port: 53
  # Allow MongoDB access
  - to:
    - podSelector:
        matchLabels:
          app: mongodb
    ports:
    - protocol: TCP
      port: 27017
  # Allow NRF access
  - to:
    - podSelector:
        matchLabels:
          app: nrf
    ports:
    - protocol: TCP
      port: 8000
```

## Service Types and Load Balancing

### ClusterIP for Internal Services

```yaml
apiVersion: v1
kind: Service
metadata:
  name: nrf
  namespace: free5gc
spec:
  type: ClusterIP
  selector:
    app: nrf
  ports:
  - port: 8000
    targetPort: 8000
    name: sbi
  sessionAffinity: ClientIP  # Maintain session affinity
  sessionAffinityConfig:
    clientIP:
      timeoutSeconds: 3600
```

### Headless Service for StatefulSets

```yaml
apiVersion: v1
kind: Service
metadata:
  name: mongodb
  namespace: free5gc
spec:
  clusterIP: None  # Headless service
  selector:
    app: mongodb
  ports:
  - port: 27017
    targetPort: 27017
    name: mongo
```

### LoadBalancer for External Access (Testing)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: amf-external
  namespace: free5gc
spec:
  type: LoadBalancer
  selector:
    app: amf
  ports:
  - port: 38412
    targetPort: 38412
    protocol: SCTP
    name: ngap
  externalTrafficPolicy: Local  # Preserve source IP
```

## Horizontal Pod Autoscaling

### HPA for User Plane Functions

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: upf-hpa
  namespace: free5gc
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: upf
  minReplicas: 2
  maxReplicas: 10
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
        name: active_pdu_sessions
      target:
        type: AverageValue
        averageValue: "500"
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300  # Wait 5 min before scale down
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0  # Scale up immediately
      policies:
      - type: Percent
        value: 100
        periodSeconds: 30
      - type: Pods
        value: 2
        periodSeconds: 30
      selectPolicy: Max
```

## Namespace Organization

### Label and Annotate Namespaces

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: free5gc
  labels:
    name: free5gc
    component: 5g-core
    tier: network-function
    managed-by: nephoran-intent-operator
    environment: production
  annotations:
    description: "Free5GC 5G Core Network Functions"
    owner: "network-ops-team"
    version: "v3.4.3"
```

### ResourceQuota to Prevent Resource Exhaustion

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: free5gc-quota
  namespace: free5gc
spec:
  hard:
    requests.cpu: "20"
    requests.memory: 40Gi
    limits.cpu: "40"
    limits.memory: 80Gi
    persistentvolumeclaims: "10"
    services.loadbalancers: "2"
```

### LimitRange for Default Resources

```yaml
apiVersion: v1
kind: LimitRange
metadata:
  name: free5gc-limit-range
  namespace: free5gc
spec:
  limits:
  - max:
      cpu: "4"
      memory: 8Gi
    min:
      cpu: "100m"
      memory: 128Mi
    default:
      cpu: "500m"
      memory: 512Mi
    defaultRequest:
      cpu: "200m"
      memory: 256Mi
    type: Container
```

## Monitoring and Observability

### Prometheus ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: amf-metrics
  namespace: free5gc
  labels:
    release: prometheus
spec:
  selector:
    matchLabels:
      app: amf
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

### Pod Annotations for Prometheus Scraping

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: amf
  namespace: free5gc
spec:
  template:
    metadata:
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: amf
        image: free5gc/amf:v3.4.3
        ports:
        - containerPort: 9090
          name: metrics
```

## Security Best Practices

### Pod Security Standards (K8s 1.35+)

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: free5gc
  labels:
    pod-security.kubernetes.io/enforce: baseline
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

### SecurityContext for Containers

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nrf
  namespace: free5gc
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: nrf
        image: free5gc/nrf:v3.4.3
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
            add:
            - NET_BIND_SERVICE
        volumeMounts:
        - name: tmp
          mountPath: /tmp
      volumes:
      - name: tmp
        emptyDir: {}
```

## Deployment Strategies

### Rolling Update (Default)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: amf
  namespace: free5gc
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1        # Create 1 extra pod during update
      maxUnavailable: 0  # Never reduce below desired replicas
  template:
    # ... pod spec ...
```

### Blue-Green Deployment Pattern

```yaml
# Blue deployment (current production)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: amf-blue
  namespace: free5gc
spec:
  replicas: 3
  selector:
    matchLabels:
      app: amf
      version: blue
  template:
    metadata:
      labels:
        app: amf
        version: blue
    spec:
      containers:
      - name: amf
        image: free5gc/amf:v3.4.2  # Old version
---
# Green deployment (new version)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: amf-green
  namespace: free5gc
spec:
  replicas: 3
  selector:
    matchLabels:
      app: amf
      version: green
  template:
    metadata:
      labels:
        app: amf
        version: green
    spec:
      containers:
      - name: amf
        image: free5gc/amf:v3.4.3  # New version
---
# Service selector switches between blue/green
apiVersion: v1
kind: Service
metadata:
  name: amf
  namespace: free5gc
spec:
  selector:
    app: amf
    version: blue  # Change to "green" to switch traffic
  ports:
  - port: 8000
    targetPort: 8000
```

## References
- **Kubernetes Documentation**: https://kubernetes.io/docs/
- **Kubernetes 1.35 Release Notes**: DRA GA, improved scheduler
- **3GPP TS 28.531**: Management of network functions
- **O-RAN.WG6**: Cloud Platform Requirements
