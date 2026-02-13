# Integration Patterns and Best Practices

## Overview

This guide provides proven patterns and best practices for integrating the Nephoran Intent Operator with existing telecommunications infrastructure, development workflows, and operational procedures. These patterns have been developed through practical experience and community feedback.

## Table of Contents

- [CI/CD Integration Patterns](#cicd-integration-patterns)
- [GitOps Workflow Patterns](#gitops-workflow-patterns)
- [Monitoring and Observability](#monitoring-and-observability)
- [Multi-Environment Patterns](#multi-environment-patterns)
- [Security Integration](#security-integration)
- [Telco Platform Integration](#telco-platform-integration)
- [API Integration Patterns](#api-integration-patterns)
- [Troubleshooting Integration Issues](#troubleshooting-integration-issues)

## CI/CD Integration Patterns

### Pattern 1: Pipeline-Driven Intent Deployment

Integrate Nephoran Intent Operator with your existing CI/CD pipeline for automated network function deployments.

```yaml
# .github/workflows/deploy-network-intent.yml
name: Deploy Network Intent

on:
  push:
    branches: [main]
    paths: ['intents/**']

jobs:
  deploy-intent:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up kubectl
        uses: azure/setup-kubectl@v1
        with:
          version: 'latest'
      
      - name: Configure kubectl
        run: |
          echo "${{ secrets.KUBECONFIG }}" | base64 -d > kubeconfig
          export KUBECONFIG=kubeconfig
          
      - name: Deploy NetworkIntent
        run: |
          envsubst < intents/production-amf.yaml | kubectl apply -f -
        env:
          ENVIRONMENT: ${{ github.ref == 'refs/heads/main' && 'production' || 'staging' }}
          REPLICA_COUNT: ${{ github.ref == 'refs/heads/main' && '3' || '1' }}
          
      - name: Wait for Intent Processing
        run: |
          kubectl wait --for=condition=Deployed \
            networkintent/production-amf \
            --timeout=600s
            
      - name: Validate Deployment
        run: |
          kubectl get networkintent production-amf -o yaml
          kubectl get pods -l nephoran.com/intent=production-amf
```

### Pattern 2: Test-Driven Intent Development

Validate intents in lower environments before production deployment.

```yaml
# intents/test-templates/amf-intent-test.yaml
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: test-amf-${TEST_ID}
  labels:
    test-suite: "integration"
    environment: "test"
spec:
  intent: "Deploy AMF for testing with ${TEST_SCENARIO}"
  target_environment: "test"
  validation:
    enabled: true
    timeout: "300s"
  cleanup:
    enabled: true
    after: "1h"
```

Integration test script:

```bash
#!/bin/bash
# scripts/test-intent.sh

set -e

TEST_ID=$(date +%s)
TEST_SCENARIO=${1:-"basic-functionality"}

echo "Testing intent deployment: $TEST_SCENARIO"

# Deploy test intent
envsubst < intents/test-templates/amf-intent-test.yaml | kubectl apply -f -

# Wait for processing
kubectl wait --for=condition=Deployed \
  networkintent/test-amf-$TEST_ID \
  --timeout=600s

# Validate deployment
kubectl get networkintent test-amf-$TEST_ID -o jsonpath='{.status.phase}' | grep -q "Deployed"

# Run functional tests
./scripts/run-amf-tests.sh test-amf-$TEST_ID

echo "Test completed successfully: $TEST_SCENARIO"
```

## GitOps Workflow Patterns

### Pattern 1: Intent-as-Code with ArgoCD

Manage NetworkIntents through GitOps workflows using ArgoCD.

```yaml
# gitops/applications/nephoran-intents.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: nephoran-intents
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/yourorg/telco-intents
    path: intents/production
    targetRevision: main
  destination:
    server: https://kubernetes.default.svc
    namespace: nephoran-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### Pattern 2: Multi-Stage Intent Promotion

Promote intents through environments using Git branches.

```bash
# scripts/promote-intent.sh
#!/bin/bash

INTENT_NAME=$1
FROM_ENV=${2:-development}
TO_ENV=${3:-staging}

# Create promotion branch
git checkout -b promote-$INTENT_NAME-to-$TO_ENV

# Copy intent configuration
cp intents/$FROM_ENV/$INTENT_NAME.yaml intents/$TO_ENV/

# Update environment-specific values
sed -i "s/target_environment: $FROM_ENV/target_environment: $TO_ENV/g" \
  intents/$TO_ENV/$INTENT_NAME.yaml

# Commit and push
git add intents/$TO_ENV/$INTENT_NAME.yaml
git commit -m "Promote $INTENT_NAME from $FROM_ENV to $TO_ENV"
git push origin promote-$INTENT_NAME-to-$TO_ENV

echo "Created promotion PR for $INTENT_NAME: $FROM_ENV -> $TO_ENV"
```

## Monitoring and Observability

### Pattern 1: Intent Processing Metrics

Set up comprehensive monitoring for intent processing pipeline.

```yaml
# monitoring/intent-monitoring.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: nephoran-intent-metrics
spec:
  selector:
    matchLabels:
      app: nephio-bridge
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

Key metrics to monitor:

```prometheus
# Intent processing rate
rate(networkintent_processed_total[5m])

# Intent success rate
(
  rate(networkintent_processed_total{status="success"}[5m]) /
  rate(networkintent_processed_total[5m])
) * 100

# Processing latency
histogram_quantile(0.95, rate(networkintent_processing_duration_seconds_bucket[5m]))

# LLM API call success rate
rate(llm_api_calls_total{status="success"}[5m]) / rate(llm_api_calls_total[5m]) * 100
```

### Pattern 2: Intent Lifecycle Alerts

Configure alerts for intent processing issues.

```yaml
# monitoring/intent-alerts.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: nephoran-intent-alerts
spec:
  groups:
  - name: intent-processing
    rules:
    - alert: IntentProcessingFailure
      expr: increase(networkintent_failed_total[10m]) > 0
      for: 2m
      annotations:
        summary: "NetworkIntent processing failures detected"
        description: "{{ $value }} intents have failed in the last 10 minutes"
        
    - alert: IntentProcessingLatencyHigh
      expr: histogram_quantile(0.95, rate(networkintent_processing_duration_seconds_bucket[5m])) > 60
      for: 5m
      annotations:
        summary: "High intent processing latency"
        description: "95th percentile processing time is {{ $value }} seconds"
        
    - alert: LLMAPIFailure
      expr: rate(llm_api_calls_total{status="error"}[5m]) / rate(llm_api_calls_total[5m]) > 0.1
      for: 2m
      annotations:
        summary: "High LLM API failure rate"
        description: "LLM API error rate is {{ $value | humanizePercentage }}"
```

## Multi-Environment Patterns

### Pattern 1: Environment-Specific Intent Configuration

Manage different configurations across development, staging, and production.

```yaml
# environments/base/amf-intent.yaml
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: core-amf
spec:
  intent: "Deploy AMF with high availability configuration"
  components:
    - name: amf
      type: "5gc-amf"
      config:
        replicas: 1  # Override in overlays
        resources:
          cpu: "100m"
          memory: "256Mi"
```

```yaml
# environments/production/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- ../base

patchesStrategicMerge:
- amf-production-config.yaml

patches:
- target:
    kind: NetworkIntent
    name: core-amf
  patch: |-
    - op: replace
      path: /spec/components/0/config/replicas
      value: 3
    - op: replace
      path: /spec/components/0/config/resources/cpu
      value: "500m"
    - op: replace
      path: /spec/components/0/config/resources/memory
      value: "1Gi"
```

### Pattern 2: Progressive Rollout

Implement canary and blue-green deployments through intent versioning.

```yaml
# intents/canary-amf.yaml
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: amf-v2-canary
  labels:
    version: "v2"
    rollout: "canary"
spec:
  intent: "Deploy AMF v2 for 10% of traffic"
  rollout:
    strategy: "canary"
    percentage: 10
    criteria:
      success_rate: ">99%"
      latency_p95: "<100ms"
      error_rate: "<1%"
  predecessor: "amf-v1-stable"
```

## Security Integration

### Pattern 1: Secure Secret Management

Integrate with external secret management systems.

```yaml
# security/external-secrets.yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: vault-secret-store
spec:
  provider:
    vault:
      server: "https://vault.example.com"
      path: "secret"
      version: "v2"
      auth:
        kubernetes:
          mountPath: "kubernetes"
          role: "nephoran-operator"
---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: llm-api-credentials
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: vault-secret-store
    kind: SecretStore
  target:
    name: llm-secrets
    creationPolicy: Owner
  data:
  - secretKey: openai-api-key
    remoteRef:
      key: nephoran/llm
      property: openai_api_key
```

### Pattern 2: Network Policy Integration

Implement zero-trust networking with Kubernetes network policies.

```yaml
# security/network-policies.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: nephoran-intent-isolation
spec:
  podSelector:
    matchLabels:
      app: nephio-bridge
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: nephoran-system
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: nephoran-system
  - to: []  # Allow external LLM API calls
    ports:
    - protocol: TCP
      port: 443
```

## Telco Platform Integration

### Pattern 1: OSS/BSS Integration

Connect intent processing with existing OSS/BSS systems.

```go
// pkg/integrations/oss_integration.go
package integrations

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type OSSIntegration struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
}

type ServiceOrder struct {
    OrderID     string                 `json:"order_id"`
    ServiceType string                 `json:"service_type"`
    Parameters  map[string]interface{} `json:"parameters"`
    Status      string                 `json:"status"`
}

func (oss *OSSIntegration) CreateServiceOrder(ctx context.Context, intent *NetworkIntent) (*ServiceOrder, error) {
    order := &ServiceOrder{
        OrderID:     fmt.Sprintf("ORD-%s", intent.Name),
        ServiceType: intent.Spec.ServiceType,
        Parameters:  intent.Spec.Parameters,
        Status:      "pending",
    }
    
    // Submit order to OSS system
    body, err := json.Marshal(order)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal order: %w", err)
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", 
        fmt.Sprintf("%s/api/v1/orders", oss.baseURL), 
        bytes.NewBuffer(body))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", oss.apiKey))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := oss.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to submit order: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusCreated {
        return nil, fmt.Errorf("order submission failed with status: %d", resp.StatusCode)
    }
    
    var result ServiceOrder
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }
    
    return &result, nil
}
```

### Pattern 2: ONAP Integration

Integrate with ONAP (Open Network Automation Platform).

```yaml
# integrations/onap-integration.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: onap-integration-config
data:
  onap.yaml: |
    onap:
      so:
        endpoint: "https://so.onap.example.com"
        auth:
          username: "nephoran-user"
          password_secret: "onap-credentials"
      aai:
        endpoint: "https://aai.onap.example.com"
        auth:
          username: "nephoran-user"
          password_secret: "onap-credentials"
      policy:
        endpoint: "https://policy-api.onap.example.com"
        auth:
          username: "nephoran-user"
          password_secret: "onap-credentials"
    
    service_mapping:
      "5gc-amf": "ONAP_5GC_AMF_SERVICE"
      "5gc-smf": "ONAP_5GC_SMF_SERVICE"
      "oran-ric": "ONAP_ORAN_NEARRT_RIC_SERVICE"
```

## API Integration Patterns

### Pattern 1: Webhook Integration

Receive events from external systems through webhooks.

```go
// pkg/webhooks/intent_webhook.go
package webhooks

import (
    "context"
    "encoding/json"
    "net/http"
    
    "k8s.io/client-go/kubernetes"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

type IntentWebhook struct {
    client client.Client
}

type WebhookPayload struct {
    EventType   string                 `json:"event_type"`
    IntentSpec  map[string]interface{} `json:"intent_spec"`
    Metadata    map[string]string      `json:"metadata"`
    Timestamp   string                 `json:"timestamp"`
}

func (w *IntentWebhook) HandleWebhook(rw http.ResponseWriter, req *http.Request) {
    var payload WebhookPayload
    if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
        http.Error(rw, "Invalid JSON payload", http.StatusBadRequest)
        return
    }
    
    // Validate webhook signature
    if !w.validateSignature(req, payload) {
        http.Error(rw, "Invalid signature", http.StatusUnauthorized)
        return
    }
    
    // Convert webhook payload to NetworkIntent
    intent := w.convertToNetworkIntent(payload)
    
    // Create NetworkIntent resource
    if err := w.client.Create(context.Background(), intent); err != nil {
        http.Error(rw, "Failed to create intent", http.StatusInternalServerError)
        return
    }
    
    rw.WriteHeader(http.StatusCreated)
    json.NewEncoder(rw).Encode(map[string]string{
        "status": "created",
        "intent": intent.Name,
    })
}
```

### Pattern 2: Event-Driven Integration

Use event systems like Apache Kafka for asynchronous integration.

```yaml
# integrations/kafka-integration.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephoran-kafka-consumer
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nephoran-kafka-consumer
  template:
    metadata:
      labels:
        app: nephoran-kafka-consumer
    spec:
      containers:
      - name: kafka-consumer
        image: nephoran/kafka-consumer:latest
        env:
        - name: KAFKA_BROKERS
          value: "kafka-cluster:9092"
        - name: CONSUMER_GROUP
          value: "nephoran-intents"
        - name: TOPICS
          value: "network-events,service-requests"
        - name: KUBECONFIG
          value: "/etc/kubeconfig/config"
        volumeMounts:
        - name: kubeconfig
          mountPath: /etc/kubeconfig
```

## Troubleshooting Integration Issues

### Common Integration Problems

1. **Authentication Issues**
   ```bash
   # Check service account permissions
   kubectl auth can-i create networkintents --as=system:serviceaccount:nephoran-system:nephio-bridge
   
   # Verify API keys are correctly configured
   kubectl get secret llm-secrets -o yaml | base64 -d
   ```

2. **Network Connectivity**
   ```bash
   # Test external API connectivity
   kubectl run debug-pod --image=curlimages/curl -it --rm -- \
     curl -H "Authorization: Bearer $OPENAI_API_KEY" \
     https://api.openai.com/v1/models
   
   # Check internal service connectivity
   kubectl run debug-pod --image=busybox -it --rm -- \
     wget -O- http://llm-processor.nephoran-system:8080/health
   ```

3. **Resource Conflicts**
   ```bash
   # Check for conflicting CRD versions
   kubectl get crd networkintents.nephoran.com -o yaml
   
   # Verify resource quotas
   kubectl describe resourcequota -n nephoran-system
   ```

### Integration Testing Checklist

- [ ] API authentication works correctly
- [ ] Network policies allow required traffic
- [ ] Resource quotas don't block creation
- [ ] External system connectivity is established
- [ ] Webhook endpoints are accessible
- [ ] Event processing completes successfully
- [ ] Error handling and retries work properly
- [ ] Monitoring and alerting are functional

---

## Best Practices Summary

1. **Security First** - Always validate inputs and use secure communication
2. **Idempotent Operations** - Design integrations to handle retries safely
3. **Comprehensive Monitoring** - Track all integration points and failure modes
4. **Graceful Degradation** - Handle external system failures gracefully
5. **Version Compatibility** - Maintain backward compatibility in APIs
6. **Documentation** - Keep integration documentation current and comprehensive
7. **Testing** - Test integrations thoroughly in all environments
8. **Observability** - Implement comprehensive logging and tracing