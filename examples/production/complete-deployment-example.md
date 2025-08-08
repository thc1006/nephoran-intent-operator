# Nephoran Intent Operator - Complete Production Deployment Example

This document provides comprehensive examples for deploying the Nephoran Intent Operator in production environments across different deployment scenarios.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Enterprise Production Deployment](#enterprise-production-deployment)
- [Carrier-Grade Telecom Deployment](#carrier-grade-telecom-deployment)
- [Edge Computing Deployment](#edge-computing-deployment)
- [Multi-Cloud Deployment](#multi-cloud-deployment)
- [Blue-Green Deployment](#blue-green-deployment)
- [Canary Deployment](#canary-deployment)
- [Monitoring and Observability](#monitoring-and-observability)
- [Backup and Disaster Recovery](#backup-and-disaster-recovery)
- [Security Configuration](#security-configuration)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Tools
```bash
# Install required tools
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh && ./get_helm.sh

wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform

curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl && sudo mv kubectl /usr/local/bin/
```

### Cloud Provider Setup
```bash
# AWS
aws configure
export AWS_DEFAULT_REGION=us-central1

# Azure
az login
az account set --subscription "your-subscription-id"

# GCP
gcloud auth login
gcloud config set project your-project-id
```

## Enterprise Production Deployment

### Step 1: Infrastructure Deployment

```bash
# Clone the repository
git clone https://github.com/thc1006/nephoran-intent-operator.git
cd nephoran-intent-operator

# Set environment variables
export ENVIRONMENT="prod"
export CLOUD_PROVIDER="aws"
export DEPLOYMENT_TYPE="enterprise"
export CONFIG_TYPE="enterprise"
export REGION="us-central1"

# Deploy infrastructure and applications
chmod +x ./scripts/deployment/deploy-nephoran.sh
./scripts/deployment/deploy-nephoran.sh \
  --environment prod \
  --cloud aws \
  --type enterprise \
  --config-type enterprise \
  --strategy rolling \
  --region us-central1
```

### Step 2: Verify Enterprise Deployment

```bash
# Check cluster status
kubectl cluster-info

# Verify all pods are running
kubectl get pods -n nephoran-system

# Check services and endpoints
kubectl get services -n nephoran-system
kubectl get ingress -n nephoran-system

# Verify resource allocation
kubectl top nodes
kubectl top pods -n nephoran-system
```

### Step 3: Enterprise-Specific Configuration

```yaml
# Create enterprise-specific configuration
cat > enterprise-overrides.yaml << EOF
global:
  enterprise:
    enabled: true
  security:
    enforceNetworkPolicies: true
    podSecurityStandard: "restricted"
    enableMTLS: true

nephoranOperator:
  replicaCount: 3
  resources:
    requests:
      cpu: "1000m"
      memory: "2Gi"
    limits:
      cpu: "4000m"
      memory: "8Gi"

llmProcessor:
  replicaCount: 5
  autoscaling:
    enabled: true
    minReplicas: 5
    maxReplicas: 50

monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 15s

backup:
  enabled: true
  schedule: "0 2 * * *"
  retention: "30d"

externalSecrets:
  enabled: true
  secretStore:
    type: "vault"
    name: "vault-backend"
EOF

# Apply enterprise configuration
helm upgrade nephoran-operator \
  ./deployments/helm/nephoran-operator \
  --namespace nephoran-system \
  --values ./deployments/helm/nephoran-operator/environments/values-enterprise.yaml \
  --values enterprise-overrides.yaml \
  --wait --timeout=20m
```

## Carrier-Grade Telecom Deployment

### Step 1: Telecom Infrastructure Setup

```bash
# Deploy carrier-grade infrastructure
export DEPLOYMENT_TYPE="telecom"
export CONFIG_TYPE="telecom-operator"
export SLA_TIER="carrier-grade"

./scripts/deployment/deploy-nephoran.sh \
  --environment prod \
  --cloud gcp \
  --type telecom \
  --config-type telecom-operator \
  --strategy blue-green \
  --region us-central1
```

### Step 2: O-RAN Interface Configuration

```yaml
# Create O-RAN specific configuration
cat > oran-config.yaml << EOF
oranAdaptor:
  enabled: true
  replicaCount: 3
  
  interfaces:
    a1:
      enabled: true
      port: 8084
      version: "v2.1"
      policyTypes:
        - "traffic_steering"
        - "qos_management" 
        - "admission_control"
        - "energy_savings"
    
    o1:
      enabled: true
      port: 8085
      protocol: "netconf"
      yangModels:
        - "ietf-interfaces"
        - "o-ran-interfaces"
        - "o-ran-operations"
    
    e2:
      enabled: true
      port: 8086
      serviceModels:
        - "KPM"
        - "RC"
        - "NI"
        - "CCC"
    
    o2:
      enabled: true
      port: 8087
      cloudProviders:
        - "aws"
        - "azure"
        - "gcp"

networkSlicing:
  enabled: true
  sliceTypes:
    - name: "eMBB"
      qci: [1, 2, 3, 4]
      bandwidth: "1Gbps"
    - name: "URLLC"
      qci: [5, 6]
      latency: "1ms"
    - name: "mMTC"
      qci: [7, 8, 9]
      connections: 1000000

carrierGrade:
  sla:
    availability: "99.999"
    latency_p99: "100ms"
    throughput: "10000rps"
    mttr: "5m"
    mttd: "1m"
  
  redundancy:
    enabled: true
    replicationFactor: 5
    crossZone: true
    crossRegion: true
EOF

# Apply O-RAN configuration
kubectl apply -f oran-config.yaml
```

### Step 3: Telecom Validation

```bash
# Test O-RAN interfaces
curl -X GET http://nephoran-api.telecom.com/api/v1/oran/a1/policies
curl -X GET http://nephoran-api.telecom.com/api/v1/oran/e2/subscriptions

# Validate network slicing
kubectl get networkslices -n nephoran-system
kubectl describe networkslice embb-slice -n nephoran-system

# Check carrier-grade metrics
kubectl port-forward svc/nephoran-operator 8080:8080 -n nephoran-system &
curl http://localhost:8080/metrics | grep nephoran_availability
curl http://localhost:8080/metrics | grep nephoran_latency
```

## Edge Computing Deployment

### Step 1: Edge Infrastructure

```bash
# Deploy to edge regions
export DEPLOYMENT_TYPE="edge"
export CONFIG_TYPE="edge-computing"

# Deploy to multiple edge locations
regions=("us-west-edge" "us-east-edge" "eu-west-edge")

for region in "${regions[@]}"; do
  echo "Deploying to edge region: $region"
  
  ./scripts/deployment/deploy-nephoran.sh \
    --environment prod \
    --cloud aws \
    --type edge \
    --config-type edge-computing \
    --strategy rolling \
    --region "$region" &
done

wait  # Wait for all deployments to complete
```

### Step 2: Edge-Specific Configuration

```yaml
# Edge computing configuration
cat > edge-config.yaml << EOF
global:
  edge:
    enabled: true
  resourceConstraints:
    enabled: true
  offlineMode:
    enabled: true

nephoranOperator:
  replicaCount: 2  # Minimal for edge
  resources:
    requests:
      cpu: "500m"
      memory: "1Gi"
    limits:
      cpu: "2000m"
      memory: "4Gi"

llmProcessor:
  replicaCount: 2
  env:
    - name: MODEL_SIZE
      value: "small"
    - name: CACHE_TTL_SECONDS
      value: "600"
    - name: LIGHTWEIGHT_MODE
      value: "true"

weaviate:
  replicaCount: 3
  config:
    query_defaults_limit: 50
    max_import_goroutine_factor: 0.5

edgeCache:
  enabled: true
  redis:
    enabled: true
    resources:
      requests:
        cpu: "200m"
        memory: "512Mi"
      limits:
        cpu: "1000m"
        memory: "2Gi"

offlineMode:
  enabled: true
  features:
    local_processing: true
    cached_models: true
    local_storage: true
    sync_on_reconnect: true
  
  sync:
    interval: "1h"
    batch_size: 100
    priority_queue: true
  
  modelCache:
    enabled: true
    size: "50Gi"
    models:
      - "gpt-4o-mini-edge"
      - "embedding-small"
EOF
```

### Step 3: Edge Validation

```bash
# Check edge deployments
for region in "${regions[@]}"; do
  echo "Checking edge region: $region"
  
  # Switch kubectl context
  kubectl config use-context "nephoran-edge-${region}"
  
  # Verify deployment
  kubectl get pods -n nephoran-system
  kubectl get services -n nephoran-system
  
  # Test local processing
  kubectl apply -f - << EOF
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: edge-test-${region}
  namespace: nephoran-system
spec:
  intent: "Deploy edge AMF with local processing"
  priority: high
  edge:
    localProcessing: true
    region: "${region}"
EOF

done
```

## Blue-Green Deployment

### Step 1: Prepare Blue-Green Configuration

```bash
# Use the blue-green deployment script
chmod +x ./scripts/deployment/blue-green-deploy.sh

# Deploy using blue-green strategy
./scripts/deployment/blue-green-deploy.sh \
  --namespace nephoran-system \
  --release nephoran-operator \
  --values ./deployments/helm/nephoran-operator/environments/values-production.yaml \
  --health-timeout 600 \
  --traffic-delay 60 \
  --cleanup-delay 900
```

### Step 2: Monitor Blue-Green Deployment

```bash
# Monitor deployment progress
watch kubectl get pods -n nephoran-system -l deployment-strategy=blue-green

# Check service endpoints
kubectl get services -n nephoran-system | grep -E "(blue|green)"

# Verify traffic routing
curl -H "Host: nephoran-api.production.com" \
  http://$(kubectl get service nephoran-operator -n nephoran-system -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')/health
```

### Step 3: Blue-Green Rollback (if needed)

```bash
# Manual rollback if deployment fails
current_color=$(kubectl get service nephoran-operator -n nephoran-system -o jsonpath='{.spec.selector.color}')
previous_color=$([[ "$current_color" == "blue" ]] && echo "green" || echo "blue")

echo "Rolling back from $current_color to $previous_color"

# Switch traffic back
kubectl patch service nephoran-operator -n nephoran-system \
  --type='merge' \
  -p="{\"spec\":{\"selector\":{\"color\":\"$previous_color\"}}}"

# Verify rollback
kubectl get service nephoran-operator -n nephoran-system -o jsonpath='{.spec.selector.color}'
```

## Canary Deployment

### Step 1: Canary Configuration

```bash
# Create canary deployment configuration
cat > canary-config.yaml << EOF
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: nephoran-operator-canary
  namespace: nephoran-system
spec:
  replicas: 10
  strategy:
    canary:
      steps:
      - setWeight: 10
      - pause: {duration: 5m}
      - setWeight: 20
      - pause: {duration: 5m}
      - setWeight: 50
      - pause: {duration: 10m}
      - setWeight: 80
      - pause: {duration: 5m}
      
      # Analysis for automatic promotion/rollback
      analysis:
        templates:
        - templateName: success-rate
        args:
        - name: service-name
          value: nephoran-operator
        
      # Traffic routing
      trafficRouting:
        nginx:
          stableService: nephoran-operator-stable
          canaryService: nephoran-operator-canary
          
  selector:
    matchLabels:
      app: nephoran-operator
  template:
    metadata:
      labels:
        app: nephoran-operator
    spec:
      containers:
      - name: nephoran-operator
        image: nephoran/intent-operator:canary
        ports:
        - containerPort: 8080
        resources:
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            cpu: 2000m
            memory: 4Gi

---
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
  namespace: nephoran-system
spec:
  args:
  - name: service-name
  metrics:
  - name: success-rate
    interval: 30s
    count: 10
    successCondition: result[0] > 0.95
    provider:
      prometheus:
        address: http://prometheus.monitoring:9090
        query: |
          sum(rate(http_requests_total{service="{{args.service-name}}",code!~"5.."}[5m])) /
          sum(rate(http_requests_total{service="{{args.service-name}}"}[5m]))
EOF

# Apply canary configuration
kubectl apply -f canary-config.yaml
```

### Step 2: Monitor Canary Progress

```bash
# Install Argo Rollouts CLI
curl -LO https://github.com/argoproj/argo-rollouts/releases/latest/download/kubectl-argo-rollouts-linux-amd64
chmod +x ./kubectl-argo-rollouts-linux-amd64
sudo mv ./kubectl-argo-rollouts-linux-amd64 /usr/local/bin/kubectl-argo-rollouts

# Watch canary rollout
kubectl argo rollouts get rollout nephoran-operator-canary -n nephoran-system --watch

# Check canary metrics
kubectl argo rollouts get rollout nephoran-operator-canary -n nephoran-system

# Manual promotion (if needed)
kubectl argo rollouts promote nephoran-operator-canary -n nephoran-system

# Manual abort (if issues detected)
kubectl argo rollouts abort nephoran-operator-canary -n nephoran-system
```

## Monitoring and Observability

### Step 1: Deploy Monitoring Stack

```bash
# Add Helm repositories
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo add jaegertracing https://jaegertracing.github.io/helm-charts
helm repo update

# Install Prometheus
helm upgrade --install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --values - << EOF
prometheus:
  prometheusSpec:
    retention: 30d
    storageSpec:
      volumeClaimTemplate:
        spec:
          storageClassName: fast-ssd
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 100Gi

grafana:
  adminPassword: "your-secure-password"
  persistence:
    enabled: true
    storageClassName: fast-ssd
    size: 10Gi
  
  # Nephoran dashboards
  dashboardProviders:
    dashboardproviders.yaml:
      apiVersion: 1
      providers:
      - name: 'nephoran'
        folder: 'Nephoran'
        type: file
        options:
          path: /var/lib/grafana/dashboards/nephoran
  
  dashboards:
    nephoran:
      nephoran-overview:
        url: https://raw.githubusercontent.com/thc1006/nephoran-intent-operator/main/monitoring/grafana/dashboards/overview.json
      nephoran-performance:
        url: https://raw.githubusercontent.com/thc1006/nephoran-intent-operator/main/monitoring/grafana/dashboards/performance.json

alertmanager:
  config:
    global:
      slack_api_url: 'your-slack-webhook-url'
    route:
      group_by: ['alertname', 'cluster', 'service']
      group_wait: 10s
      group_interval: 10s
      repeat_interval: 1h
      receiver: 'slack-notifications'
    
    receivers:
    - name: 'slack-notifications'
      slack_configs:
      - channel: '#nephoran-alerts'
        title: 'Nephoran Alert - {{ .Status | toUpper }}'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'
EOF

# Install Jaeger
helm upgrade --install jaeger jaegertracing/jaeger \
  --namespace monitoring \
  --values - << EOF
storage:
  type: elasticsearch
elasticsearch:
  host: elasticsearch.monitoring
  port: 9200
  
collector:
  replicaCount: 3
  
query:
  replicaCount: 2
  ingress:
    enabled: true
    hosts:
      - jaeger.monitoring.example.com
EOF
```

### Step 2: Configure Nephoran Metrics

```yaml
# Apply ServiceMonitor for Nephoran components
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nephoran-operator
  namespace: nephoran-system
  labels:
    app: nephoran-operator
    release: kube-prometheus-stack
spec:
  selector:
    matchLabels:
      app: nephoran-operator
  endpoints:
  - port: http-metrics
    path: /metrics
    interval: 15s
    scrapeTimeout: 10s
  - port: http-metrics
    path: /metrics/controller-runtime
    interval: 15s
    scrapeTimeout: 10s
```

### Step 3: Alerting Rules

```yaml
# Nephoran-specific alerting rules
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: nephoran-alerts
  namespace: nephoran-system
  labels:
    app: nephoran-operator
    release: kube-prometheus-stack
spec:
  groups:
  - name: nephoran.rules
    rules:
    - alert: NephoranOperatorDown
      expr: up{job="nephoran-operator"} == 0
      for: 2m
      labels:
        severity: critical
        component: controller
      annotations:
        summary: "Nephoran Operator is down"
        description: "Nephoran Operator has been down for more than 2 minutes"
        runbook_url: "https://runbooks.nephoran.com/operator-down"
    
    - alert: NephoranHighLatency
      expr: |
        histogram_quantile(0.95, 
          rate(nephoran_llm_processor_request_duration_seconds_bucket[5m])
        ) > 5
      for: 10m
      labels:
        severity: warning
        component: llm-processor
      annotations:
        summary: "High latency in Nephoran LLM Processor"
        description: "95th percentile latency is above 5 seconds for 10 minutes"
    
    - alert: NephoranHighErrorRate
      expr: |
        (
          rate(nephoran_controller_reconcile_errors_total[5m]) /
          rate(nephoran_controller_reconcile_total[5m])
        ) > 0.1
      for: 5m
      labels:
        severity: warning
        component: controller
      annotations:
        summary: "High error rate in Nephoran Operator"
        description: "Error rate is above 10% for the last 5 minutes"
```

## Backup and Disaster Recovery

### Step 1: Install Velero

```bash
# Install Velero CLI
wget https://github.com/vmware-tanzu/velero/releases/latest/download/velero-linux-amd64.tar.gz
tar -xvf velero-linux-amd64.tar.gz
sudo mv velero-*/velero /usr/local/bin/

# Create backup storage (AWS S3 example)
aws s3 mb s3://nephoran-velero-backups --region us-central1

# Create IAM policy and user for Velero
cat > velero-policy.json << EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:DescribeVolumes",
                "ec2:DescribeSnapshots",
                "ec2:CreateTags",
                "ec2:CreateVolume",
                "ec2:CreateSnapshot",
                "ec2:DeleteSnapshot"
            ],
            "Resource": "*"
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "s3:DeleteObject",
                "s3:PutObject",
                "s3:AbortMultipartUpload",
                "s3:ListMultipartUploadParts"
            ],
            "Resource": [
                "arn:aws:s3:::nephoran-velero-backups/*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::nephoran-velero-backups"
            ]
        }
    ]
}
EOF

aws iam create-policy --policy-name VeleroBackupPolicy --policy-document file://velero-policy.json
aws iam create-user --user-name velero
aws iam attach-user-policy --user-name velero --policy-arn arn:aws:iam::YOUR_ACCOUNT:policy/VeleroBackupPolicy
aws iam create-access-key --user-name velero

# Install Velero
cat > velero-credentials << EOF
[default]
aws_access_key_id=YOUR_ACCESS_KEY
aws_secret_access_key=YOUR_SECRET_KEY
EOF

velero install \
  --provider aws \
  --plugins velero/velero-plugin-for-aws:v1.8.0 \
  --bucket nephoran-velero-backups \
  --backup-location-config region=us-central1 \
  --snapshot-location-config region=us-central1 \
  --secret-file ./velero-credentials
```

### Step 2: Configure Backup Schedules

```bash
# Create backup schedule for Nephoran system
velero schedule create nephoran-daily \
  --schedule="0 2 * * *" \
  --include-namespaces nephoran-system \
  --ttl 720h0m0s

# Create backup schedule for monitoring
velero schedule create monitoring-weekly \
  --schedule="0 1 * * 0" \
  --include-namespaces monitoring \
  --ttl 2160h0m0s

# Manual backup
velero backup create nephoran-manual-$(date +%Y%m%d-%H%M%S) \
  --include-namespaces nephoran-system \
  --wait
```

### Step 3: Disaster Recovery Testing

```bash
# Test restore procedure
velero backup create pre-disaster-test \
  --include-namespaces nephoran-system \
  --wait

# Simulate disaster (delete namespace)
kubectl delete namespace nephoran-system

# Restore from backup
velero restore create nephoran-restore-$(date +%Y%m%d-%H%M%S) \
  --from-backup pre-disaster-test \
  --wait

# Verify restoration
kubectl get pods -n nephoran-system
kubectl get services -n nephoran-system
```

## Security Configuration

### Step 1: Network Policies

```yaml
# Strict network policies for production
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: nephoran-system-isolation
  namespace: nephoran-system
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  
  # Default deny all
  ingress: []
  egress: []

---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: nephoran-operator-ingress
  namespace: nephoran-system
spec:
  podSelector:
    matchLabels:
      app: nephoran-operator
  policyTypes:
  - Ingress
  
  ingress:
  # Allow from ingress controller
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  
  # Allow from monitoring
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8080

---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: nephoran-egress-allowed
  namespace: nephoran-system
spec:
  podSelector: {}
  policyTypes:
  - Egress
  
  egress:
  # Allow DNS
  - to: []
    ports:
    - protocol: UDP
      port: 53
  
  # Allow HTTPS to external APIs
  - to: []
    ports:
    - protocol: TCP
      port: 443
  
  # Allow inter-component communication
  - to:
    - podSelector:
        matchLabels:
          app: nephoran-operator
    ports:
    - protocol: TCP
      port: 8080
```

### Step 2: Pod Security Standards

```yaml
# Pod Security Standards enforcement
apiVersion: v1
kind: Namespace
metadata:
  name: nephoran-system
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

### Step 3: OPA Gatekeeper Policies

```yaml
# Gatekeeper constraint template
apiVersion: templates.gatekeeper.sh/v1beta1
kind: ConstraintTemplate
metadata:
  name: nephoransecurity
spec:
  crd:
    spec:
      names:
        kind: NephoranSecurity
      validation:
        openAPIV3Schema:
          type: object
          properties:
            requiredLabels:
              type: array
              items:
                type: string
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
        package nephoransecurity
        
        violation[{"msg": msg}] {
          required := input.parameters.requiredLabels
          provided := input.review.object.metadata.labels
          missing := required[_]
          not provided[missing]
          msg := sprintf("Missing required label: %v", [missing])
        }

---
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: NephoranSecurity
metadata:
  name: must-have-security-labels
spec:
  match:
    kinds:
      - apiGroups: [""]
        kinds: ["Pod"]
    namespaces: ["nephoran-system"]
  parameters:
    requiredLabels: ["security.nephoran.com/scan-status", "security.nephoran.com/risk-level"]
```

## Troubleshooting

### Common Issues and Solutions

#### 1. Deployment Failures

```bash
# Check deployment status
kubectl get deployments -n nephoran-system
kubectl describe deployment nephoran-operator -n nephoran-system

# Check pod logs
kubectl logs -l app=nephoran-operator -n nephoran-system --tail=100

# Check resource constraints
kubectl describe node
kubectl top pods -n nephoran-system
```

#### 2. Service Discovery Issues

```bash
# Check services and endpoints
kubectl get services -n nephoran-system
kubectl get endpoints -n nephoran-system

# Test service connectivity
kubectl run test-pod --image=busybox -i --tty --rm -- nslookup nephoran-operator.nephoran-system.svc.cluster.local
```

#### 3. Performance Issues

```bash
# Check metrics
kubectl port-forward svc/nephoran-operator 8080:8080 -n nephoran-system &
curl http://localhost:8080/metrics | grep -E "(latency|duration|rate)"

# Check resource usage
kubectl top pods -n nephoran-system --sort-by=cpu
kubectl top pods -n nephoran-system --sort-by=memory
```

#### 4. Certificate Issues

```bash
# Check certificate status
kubectl get certificates -n nephoran-system
kubectl describe certificate nephoran-operator-tls -n nephoran-system

# Renew certificates
kubectl delete certificate nephoran-operator-tls -n nephoran-system
kubectl apply -f cert-config.yaml
```

### Emergency Procedures

#### Quick Rollback

```bash
# Rollback Helm release
helm rollback nephoran-operator -n nephoran-system

# Rollback Kubernetes deployment
kubectl rollout undo deployment/nephoran-operator -n nephoran-system

# Check rollback status
kubectl rollout status deployment/nephoran-operator -n nephoran-system
```

#### Scale Down for Maintenance

```bash
# Scale down components
kubectl scale deployment nephoran-operator --replicas=0 -n nephoran-system
kubectl scale deployment llm-processor --replicas=0 -n nephoran-system

# Scale back up
kubectl scale deployment nephoran-operator --replicas=3 -n nephoran-system
kubectl scale deployment llm-processor --replicas=5 -n nephoran-system
```

#### Emergency Contact Information

```yaml
# Update emergency contact configmap
apiVersion: v1
kind: ConfigMap
metadata:
  name: emergency-contacts
  namespace: nephoran-system
data:
  primary_contact: "ops-team@company.com"
  secondary_contact: "platform-team@company.com"
  escalation_contact: "cto@company.com"
  slack_channel: "#nephoran-alerts"
  pager_duty_service: "PXXXXXX"
```

## Conclusion

This comprehensive deployment guide covers the major production deployment scenarios for the Nephoran Intent Operator. Each configuration is designed for specific use cases:

- **Enterprise**: High availability, security-focused deployment for large organizations
- **Carrier-Grade**: Ultra-reliable, O-RAN compliant deployment for telecommunications operators  
- **Edge**: Resource-optimized deployment for distributed edge computing scenarios
- **Multi-Cloud**: Vendor-agnostic deployment across multiple cloud providers

The deployment strategies (rolling, blue-green, canary) provide different risk/benefit trade-offs for production updates, while the monitoring, backup, and security configurations ensure operational excellence.

For additional support or customization requirements, refer to the project documentation or contact the development team.