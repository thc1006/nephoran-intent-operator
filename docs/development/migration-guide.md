# Migration Guide

## Overview

This guide helps you migrate from existing telecommunications management systems to the Nephoran Intent Operator. Whether you're coming from traditional OSS/BSS systems, other Kubernetes operators, or manual network management processes, this guide provides step-by-step migration strategies and best practices.

## Migration Scenarios

- [From Manual Network Management](#from-manual-network-management)
- [From Traditional OSS/BSS Systems](#from-traditional-ossbss-systems)
- [From Other Kubernetes Operators](#from-other-kubernetes-operators)
- [From ONAP-based Systems](#from-onap-based-systems)
- [From Legacy O-RAN Implementations](#from-legacy-o-ran-implementations)

## Pre-Migration Assessment

### System Inventory

Before starting migration, catalog your existing system:

```bash
#!/bin/bash
# scripts/migration-assessment.sh

echo "=== Migration Assessment Tool ==="

# Document existing network functions
echo "Current Network Functions:"
kubectl get deployments,statefulsets -l type=network-function --all-namespaces

# Document existing intents/configurations
echo "Current Configuration Management:"
find /etc/network-config -name "*.yaml" -o -name "*.json" 2>/dev/null | wc -l

# Document external dependencies
echo "External System Dependencies:"
netstat -an | grep ESTABLISHED | awk '{print $5}' | cut -d: -f1 | sort -u
```

### Compatibility Matrix

| Source System | Complexity | Timeline | Automation Level |
|---------------|------------|----------|------------------|
| Manual Scripts | Low | 1-2 weeks | High |
| OSS/BSS | High | 2-3 months | Medium |
| Other K8s Operators | Medium | 3-6 weeks | High |
| ONAP | High | 3-4 months | Medium |
| Legacy O-RAN | Medium | 6-8 weeks | Medium |

### Risk Assessment

**High Risk Areas:**
- Production traffic management
- Service availability during migration
- Data migration and consistency
- Integration with billing systems

**Mitigation Strategies:**
- Parallel deployment approach
- Phased rollout with rollback capability
- Comprehensive testing in staging
- Monitoring and alerting throughout migration

## From Manual Network Management

### Current State Analysis

If you're currently managing network functions through:
- Manual kubectl commands
- Shell scripts
- Custom automation tools
- Direct API calls

### Migration Strategy: Direct Transition

**Phase 1: Intent Mapping (Week 1)**

Document your current manual processes:

```bash
# Create intent mapping document
cat > migration-mapping.yaml <<EOF
manual_processes:
  - name: "Deploy AMF"
    current_commands:
      - kubectl apply -f amf-deployment.yaml
      - kubectl apply -f amf-service.yaml
      - kubectl apply -f amf-configmap.yaml
    target_intent: "Deploy high-availability AMF for production"
    
  - name: "Scale UPF"
    current_commands:
      - kubectl scale deployment upf --replicas=3
      - kubectl patch service upf -p '{"spec":{"ports":[{"port":80,"targetPort":8080}]}}'
    target_intent: "Scale UPF to handle 1000 concurrent sessions"
    
  - name: "Deploy Test Environment"
    current_commands:
      - ./scripts/deploy-5g-core.sh test
      - kubectl apply -f test-configs/
    target_intent: "Create complete 5G test environment with monitoring"
EOF
```

**Phase 2: Install Nephoran (Week 1-2)**

```bash
# Install Nephoran Intent Operator
kubectl create namespace nephoran-system

# Deploy with existing network functions support
helm install nephoran-operator ./deployments/helm/nephoran-operator \
  --namespace nephoran-system \
  --set migration.enabled=true \
  --set migration.preserve_existing=true
```

**Phase 3: Parallel Deployment (Week 2-3)**

Run both systems in parallel:

```bash
# Create intent for existing manual process
kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: amf-parallel-test
  annotations:
    migration.nephoran.com/source: "manual"
    migration.nephoran.com/validation: "parallel"
spec:
  intent: "Deploy AMF identical to manual deployment"
  migration:
    parallel_mode: true
    compare_with: "amf-manual"
    validation_timeout: "10m"
EOF

# Validate both deployments produce identical results
kubectl get deployment amf-manual -o yaml > manual-amf.yaml
kubectl get deployment amf-parallel-test -o yaml > intent-amf.yaml
diff manual-amf.yaml intent-amf.yaml
```

**Phase 4: Cutover (Week 3-4)**

```bash
# Gradually replace manual processes with intents
for process in amf-deployment upf-scaling monitoring-setup; do
  echo "Migrating $process..."
  
  # Create intent for the process
  kubectl apply -f intents/$process-intent.yaml
  
  # Wait for deployment
  kubectl wait --for=condition=Deployed networkintent/$process --timeout=600s
  
  # Validate functionality
  ./scripts/validate-$process.sh
  
  # Remove manual deployment
  kubectl delete -f manual-configs/$process.yaml
done
```

### Sample Migration Scripts

```bash
#!/bin/bash
# scripts/migrate-manual-process.sh

PROCESS_NAME=$1
INTENT_FILE=$2

if [ -z "$PROCESS_NAME" ] || [ -z "$INTENT_FILE" ]; then
  echo "Usage: $0 <process-name> <intent-file>"
  exit 1
fi

echo "Starting migration of $PROCESS_NAME..."

# Backup current state
kubectl get all -l process=$PROCESS_NAME -o yaml > backup-$PROCESS_NAME.yaml

# Deploy intent
kubectl apply -f $INTENT_FILE

# Wait for processing
kubectl wait --for=condition=Deployed networkintent/$PROCESS_NAME --timeout=600s

# Validate deployment
if ./scripts/validate-$PROCESS_NAME.sh; then
  echo "Migration successful for $PROCESS_NAME"
  # Clean up old resources
  kubectl delete -l process=$PROCESS_NAME,migration=true
else
  echo "Migration failed for $PROCESS_NAME, rolling back..."
  kubectl apply -f backup-$PROCESS_NAME.yaml
  kubectl delete networkintent/$PROCESS_NAME
fi
```

## From Traditional OSS/BSS Systems

### Current State Analysis

OSS/BSS systems typically include:
- Service order management
- Network inventory management
- Fault management
- Performance monitoring
- Billing integration

### Migration Strategy: Service-by-Service

**Phase 1: API Integration Setup (Month 1)**

Create integration layer with existing OSS/BSS:

```yaml
# integrations/oss-bridge.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: oss-integration-config
data:
  config.yaml: |
    oss_systems:
      service_orchestrator:
        endpoint: "https://oss.company.com/api/v1"
        auth_type: "oauth2"
        client_id: "nephoran-integration"
        scopes: ["service.read", "service.write"]
      
      inventory_manager:
        endpoint: "https://inventory.company.com/api/v2"
        auth_type: "basic"
        username: "nephoran"
        
      fault_manager:
        endpoint: "https://fm.company.com/api/v1"
        auth_type: "token"
        token_secret: "fm-token"
    
    service_mapping:
      "5g-amf": "FIVE_G_AMF_SERVICE"
      "5g-smf": "FIVE_G_SMF_SERVICE"
      "o-ran-ric": "ORAN_RIC_SERVICE"
```

Integration service:

```go
// pkg/integrations/oss_client.go
package integrations

type OSSClient struct {
    serviceOrchestrator *ServiceOrchestratorClient
    inventoryManager    *InventoryManagerClient
    faultManager       *FaultManagerClient
}

func (c *OSSClient) CreateServiceOrder(intent *NetworkIntent) (*ServiceOrder, error) {
    // Map NetworkIntent to OSS service order
    order := &ServiceOrder{
        ServiceType: mapIntentToServiceType(intent.Spec.Intent),
        Parameters:  extractParametersFromIntent(intent.Spec),
        RequestedBy: "nephoran-operator",
        Priority:    intent.Spec.Priority,
    }
    
    // Create order in OSS
    response, err := c.serviceOrchestrator.CreateOrder(order)
    if err != nil {
        return nil, fmt.Errorf("failed to create OSS order: %w", err)
    }
    
    return response, nil
}
```

**Phase 2: Parallel Processing (Month 2)**

Run both systems in parallel:

```yaml
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: service-migration-test
  annotations:
    integration.nephoran.com/oss-order-id: "ORD-123456"
    migration.nephoran.com/phase: "parallel"
spec:
  intent: "Deploy AMF service for customer ABC"
  integration:
    oss_systems:
      create_service_order: true
      update_inventory: true
      billing_integration: false  # Disable during testing
  validation:
    compare_with_oss: true
    oss_order_id: "ORD-123456"
```

**Phase 3: Service Migration (Month 2-3)**

Migrate services one by one:

```bash
#!/bin/bash
# scripts/migrate-oss-service.sh

SERVICE_TYPE=$1

# Get list of active services from OSS
OSS_SERVICES=$(curl -H "Authorization: Bearer $OSS_TOKEN" \
  "$OSS_ENDPOINT/services?type=$SERVICE_TYPE&status=active")

# Create intents for each service
echo "$OSS_SERVICES" | jq -r '.services[]' | while read service; do
  SERVICE_ID=$(echo $service | jq -r '.id')
  
  # Create NetworkIntent based on OSS service definition
  kubectl apply -f - <<EOF
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: migrated-${SERVICE_ID}
  annotations:
    migration.nephoran.com/source-oss-id: "${SERVICE_ID}"
spec:
  intent: "Migrate ${SERVICE_TYPE} service ${SERVICE_ID} from OSS"
  migration:
    source_system: "oss"
    source_id: "${SERVICE_ID}"
    preserve_configuration: true
EOF

  # Wait for migration to complete
  kubectl wait --for=condition=Deployed networkintent/migrated-${SERVICE_ID} --timeout=1800s
done
```

## From Other Kubernetes Operators

### Common Source Operators

- **Helm Operator**: Chart-based deployments
- **Custom Operators**: Domain-specific automation
- **ArgoCD/Flux**: GitOps-based deployment

### Migration Strategy: Configuration Translation

**Phase 1: Configuration Analysis**

```bash
# Analyze existing operator configurations
kubectl get all -l app.kubernetes.io/managed-by=helm -o yaml > helm-resources.yaml
kubectl get applications -n argocd -o yaml > argocd-apps.yaml

# Extract deployment patterns
./scripts/analyze-operator-patterns.sh
```

**Phase 2: Intent Translation**

```python
#!/usr/bin/env python3
# scripts/translate-helm-to-intent.py

import yaml
import argparse
from pathlib import Path

def translate_helm_to_intent(helm_values, chart_name):
    """Translate Helm values to NetworkIntent"""
    
    intent_spec = {
        "apiVersion": "nephoran.com/v1alpha1",
        "kind": "NetworkIntent",
        "metadata": {
            "name": f"migrated-{chart_name}",
            "annotations": {
                "migration.nephoran.com/source": "helm",
                "migration.nephoran.com/chart": chart_name
            }
        },
        "spec": {
            "intent": f"Deploy {chart_name} with Helm-equivalent configuration",
            "migration": {
                "source_type": "helm",
                "preserve_values": True
            },
            "parameters": helm_values
        }
    }
    
    return intent_spec

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--values', required=True, help='Helm values file')
    parser.add_argument('--chart', required=True, help='Chart name')
    parser.add_argument('--output', required=True, help='Output intent file')
    args = parser.parse_args()
    
    with open(args.values) as f:
        helm_values = yaml.safe_load(f)
    
    intent = translate_helm_to_intent(helm_values, args.chart)
    
    with open(args.output, 'w') as f:
        yaml.dump(intent, f, default_flow_style=False)
    
    print(f"Intent created: {args.output}")

if __name__ == "__main__":
    main()
```

**Phase 3: Validation and Cutover**

```bash
#!/bin/bash
# scripts/validate-operator-migration.sh

CHART_NAME=$1
NAMESPACE=${2:-default}

echo "Validating migration for $CHART_NAME in $NAMESPACE..."

# Get resources managed by Helm
HELM_RESOURCES=$(helm get manifest $CHART_NAME -n $NAMESPACE)

# Get resources created by NetworkIntent
INTENT_RESOURCES=$(kubectl get networkintent migrated-$CHART_NAME -o jsonpath='{.status.deployedResources}')

# Compare resource configurations
echo "Comparing Helm vs Intent resources..."

# Extract and compare each resource type
for resource_type in deployment service configmap; do
    echo "Checking $resource_type..."
    
    # Get Helm-managed resources
    echo "$HELM_RESOURCES" | kubectl apply --dry-run=client -f - 2>/dev/null | \
        grep "$resource_type" > helm-$resource_type.tmp
    
    # Get Intent-managed resources
    kubectl get $resource_type -l nephoran.com/intent=migrated-$CHART_NAME \
        --dry-run=client -o yaml > intent-$resource_type.tmp
    
    # Compare (ignoring metadata differences)
    if diff -u helm-$resource_type.tmp intent-$resource_type.tmp --ignore-matching-lines="uid:\|resourceVersion:\|creationTimestamp:"; then
        echo "✅ $resource_type matches"
    else
        echo "❌ $resource_type differs"
    fi
done

# Cleanup temp files
rm -f helm-*.tmp intent-*.tmp
```

## From ONAP-based Systems

### ONAP Component Mapping

| ONAP Component | Nephoran Equivalent | Migration Complexity |
|----------------|--------------------|--------------------|
| SO (Service Orchestrator) | NetworkIntent Controller | High |
| SDNC | O-RAN Adaptor | Medium |
| A&AI | Resource Inventory | Medium |
| Policy | Intent Validation | Low |
| DCAE | Monitoring Integration | Medium |

### Migration Strategy: Component-by-Component

**Phase 1: Service Model Translation**

```yaml
# Create service mapping configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: onap-migration-mapping
data:
  service-mapping.yaml: |
    onap_to_nephoran:
      service_models:
        "5G_Core_AMF_v1.0":
          target_intent: "Deploy 5G AMF with ONAP-compatible configuration"
          parameters:
            - onap_param: "vnf_id"
              intent_param: "network_function_id"
            - onap_param: "service_instance_id"
              intent_param: "service_id"
        
        "ORAN_RIC_v2.0":
          target_intent: "Deploy O-RAN Near-RT RIC"
          parameters:
            - onap_param: "ric_id"
              intent_param: "ric_instance_id"
    
    workflow_mapping:
      "InstantiateVNF":
        intent_template: "templates/instantiate-nf.yaml"
        validation_rules:
          - check_resource_availability
          - validate_configuration
```

**Phase 2: API Bridge**

```go
// pkg/integrations/onap_bridge.go
package integrations

type ONAPBridge struct {
    soClient   *ServiceOrchestratorClient
    aaiClient  *ActiveInventoryClient
    sdncClient *SDNControllerClient
}

func (o *ONAPBridge) HandleServiceInstantiation(request *ONAPServiceRequest) error {
    // Translate ONAP service request to NetworkIntent
    intent := &NetworkIntent{
        ObjectMeta: metav1.ObjectMeta{
            Name: fmt.Sprintf("onap-%s", request.ServiceInstanceID),
            Annotations: map[string]string{
                "onap.nephoran.com/service-id": request.ServiceInstanceID,
                "onap.nephoran.com/model-name": request.ServiceModelName,
            },
        },
        Spec: NetworkIntentSpec{
            Intent: o.translateServiceModel(request.ServiceModelName),
            Parameters: o.mapONAPParameters(request.Parameters),
            Integration: IntegrationSpec{
                ONAP: &ONAPIntegration{
                    ServiceInstanceID: request.ServiceInstanceID,
                    UpdateAAI:         true,
                    NotifyDCAE:        true,
                },
            },
        },
    }
    
    // Create the NetworkIntent
    return o.k8sClient.Create(context.Background(), intent)
}
```

## From Legacy O-RAN Implementations

### Legacy System Assessment

```bash
#!/bin/bash
# scripts/assess-oran-legacy.sh

echo "=== O-RAN Legacy System Assessment ==="

# Check existing O-RAN components
echo "Current O-RAN Components:"
kubectl get pods -l app.kubernetes.io/component=oran

# Check interface implementations
echo "O-RAN Interface Status:"
for interface in a1 o1 o2 e2; do
    echo "  $interface interface:"
    kubectl get services -l oran.io/interface=$interface
done

# Check xApp deployments
echo "Existing xApps:"
kubectl get pods -l app.kubernetes.io/component=xapp

# Document custom configurations
echo "Custom O-RAN Configurations:"
find /opt/oran -name "*.conf" -o -name "*.json" 2>/dev/null
```

### Migration Strategy: Interface-by-Interface

**Phase 1: A1 Interface Migration**

```yaml
# Current A1 policy (legacy format)
# /opt/oran/policies/traffic-steering.json
{
  "policy_type_id": 20008,
  "policy_instance_id": "traffic-steering-001",
  "ric_id": "ric-1",
  "policy": {
    "threshold": 75,
    "action": "redistribute"
  }
}

# Migrated to NetworkIntent
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: traffic-steering-policy
  annotations:
    oran.nephoran.com/interface: "a1"
    oran.nephoran.com/legacy-policy-id: "traffic-steering-001"
spec:
  intent: "Configure traffic steering policy with 75% threshold"
  target_components:
    - type: "near-rt-ric"
      id: "ric-1"
  parameters:
    policy_type: "traffic_steering"
    threshold: 75
    action: "redistribute"
```

**Phase 2: E2 Interface Migration**

```yaml
# Legacy E2 node configuration
# /opt/oran/e2nodes/enb-001.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: enb-001-config
data:
  e2ap.conf: |
    node_id: "enb-001"
    plmn_id: "00101"
    cell_id: "0x1234"
    tracking_area_code: 123

# Migrated to E2NodeSet
apiVersion: nephoran.com/v1alpha1
kind: E2NodeSet
metadata:
  name: legacy-enb-nodes
  annotations:
    migration.nephoran.com/source: "legacy-config"
spec:
  intent: "Migrate legacy eNodeB configurations to E2NodeSet"
  replica: 3
  nodeTemplate:
    nodeId: "enb-{index:03d}"
    plmnId: "00101"
    cellConfiguration:
      cellId: "0x{index:04x}"
      trackingAreaCode: 123
  migration:
    preserve_ids: true
    source_configs:
      - "/opt/oran/e2nodes/enb-001.yaml"
      - "/opt/oran/e2nodes/enb-002.yaml"
      - "/opt/oran/e2nodes/enb-003.yaml"
```

## Migration Validation and Testing

### Validation Framework

```bash
#!/bin/bash
# scripts/migration-validation.sh

MIGRATION_NAME=$1
PHASE=${2:-"pre-migration"}

echo "=== Migration Validation: $MIGRATION_NAME ($PHASE) ==="

case $PHASE in
  "pre-migration")
    # Validate source system
    ./scripts/validate-source-system.sh $MIGRATION_NAME
    
    # Check prerequisites
    ./scripts/check-migration-prereqs.sh $MIGRATION_NAME
    
    # Create baseline metrics
    ./scripts/capture-baseline-metrics.sh $MIGRATION_NAME
    ;;
    
  "parallel")
    # Compare old vs new system outputs
    ./scripts/compare-system-outputs.sh $MIGRATION_NAME
    
    # Validate functional equivalence
    ./scripts/functional-equivalence-test.sh $MIGRATION_NAME
    
    # Performance comparison
    ./scripts/performance-comparison.sh $MIGRATION_NAME
    ;;
    
  "post-migration")
    # Validate cutover success
    ./scripts/validate-cutover.sh $MIGRATION_NAME
    
    # Check all services are running
    ./scripts/service-health-check.sh
    
    # Validate end-to-end workflows
    ./scripts/e2e-workflow-test.sh
    ;;
esac
```

### Testing Strategies

**1. Functional Testing**

```yaml
# Test intent processing matches legacy behavior
apiVersion: nephoran.com/v1alpha1
kind: NetworkIntent
metadata:
  name: migration-test-amf
  annotations:
    test.nephoran.com/type: "migration-validation"
    test.nephoran.com/compare-with: "legacy-amf-deployment"
spec:
  intent: "Deploy AMF identical to legacy configuration"
  validation:
    compare_resources: true
    compare_metrics: true
    compare_behavior: true
    timeout: "30m"
```

**2. Load Testing**

```bash
# Compare performance under load
./scripts/load-test-comparison.sh --old-system legacy --new-system nephoran --duration 10m
```

**3. Failover Testing**

```bash
# Test failover scenarios
./scripts/test-migration-failover.sh --scenario "controller-failure"
./scripts/test-migration-failover.sh --scenario "api-failure"
./scripts/test-migration-failover.sh --scenario "storage-failure"
```

## Rollback Procedures

### Emergency Rollback

```bash
#!/bin/bash
# scripts/emergency-rollback.sh

MIGRATION_NAME=$1

echo "EMERGENCY ROLLBACK: $MIGRATION_NAME"

# Stop new intent processing
kubectl scale deployment nephio-bridge --replicas=0 -n nephoran-system

# Restore original system
kubectl apply -f backups/${MIGRATION_NAME}-original-system.yaml

# Verify original system is running
./scripts/validate-original-system.sh $MIGRATION_NAME

# Update DNS/routing to point to original system
./scripts/restore-original-routing.sh $MIGRATION_NAME

echo "Emergency rollback completed for $MIGRATION_NAME"
```

### Planned Rollback

```bash
#!/bin/bash
# scripts/planned-rollback.sh

MIGRATION_NAME=$1

echo "Planned rollback for $MIGRATION_NAME"

# Drain traffic from new system
kubectl patch deployment nephio-bridge -p '{"spec":{"replicas":0}}' -n nephoran-system

# Wait for current intents to complete
kubectl wait --for=delete networkintent --all --timeout=600s

# Switch back to original system
kubectl apply -f backups/${MIGRATION_NAME}-original-system.yaml

# Validate rollback
./scripts/validate-rollback.sh $MIGRATION_NAME
```

## Best Practices and Lessons Learned

### Migration Success Factors

1. **Thorough Planning**
   - Complete system inventory
   - Detailed migration timeline
   - Clear success criteria
   - Comprehensive testing plan

2. **Risk Mitigation**
   - Parallel deployment phases
   - Quick rollback procedures
   - Comprehensive monitoring
   - Stakeholder communication

3. **Validation at Every Step**
   - Functional equivalence testing
   - Performance benchmarking
   - End-to-end workflow validation
   - User acceptance testing

### Common Pitfalls

1. **Insufficient Testing**
   - Always test in staging first
   - Include edge cases and failure scenarios
   - Test with realistic data volumes

2. **Rushed Cutover**
   - Allow adequate time for parallel running
   - Don't skip validation steps
   - Have clear go/no-go criteria

3. **Poor Communication**
   - Keep all stakeholders informed
   - Document all changes
   - Provide training for operations teams

### Post-Migration Optimization

```bash
# Monitor system performance post-migration
./scripts/post-migration-monitoring.sh

# Optimize configurations based on actual usage
./scripts/optimize-post-migration.sh

# Clean up migration artifacts
./scripts/cleanup-migration-artifacts.sh
```

---

**Note**: Migration is a complex process that should be thoroughly planned and tested. Consider engaging with the Nephoran community or professional services for large-scale migrations. This guide provides general patterns but your specific migration may require custom approaches.