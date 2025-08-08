# Nephio-Native Workflow Integration

This package provides comprehensive Nephio R5-compliant workflow patterns for the Nephoran Intent Operator, enabling truly native Nephio GitOps package management and orchestration.

## Overview

The Nephio integration transforms the Nephoran Intent Operator into a fully Nephio-native platform that leverages all standard Nephio R5 patterns:

- **Native Package Management**: Full integration with Porch for package lifecycle management
- **GitOps Workflows**: Config Sync integration for cluster deployment
- **Workload Cluster Registry**: Automated cluster discovery and management  
- **Blueprint Catalog**: Comprehensive package blueprint library
- **Workflow Engine**: Standard Nephio workflow patterns with custom extensibility

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Nephoran Intent Operator                         │
├─────────────────────────────────────────────────────────────────────┤
│                    Nephio Integration Layer                         │
├──────────────┬──────────────┬─────────────┬─────────────┬──────────┤
│   Workflow   │   Package    │  Workload   │ Config Sync │ Workflow │
│ Orchestrator │   Catalog    │  Registry   │   Client    │  Engine  │
├──────────────┼──────────────┼─────────────┼─────────────┼──────────┤
│              │              │             │             │          │
│ • Standard   │ • Blueprint  │ • Cluster   │ • Git       │ • Custom │
│   Workflows  │   Management │   Discovery │   Operations│   Rules  │
│ • Custom     │ • Variant    │ • Health    │ • Package   │ • Validation │
│   Extensions │   Creation   │   Monitor   │   Sync      │ • Execution │
│              │              │             │             │          │
└──────────────┴──────────────┴─────────────┴─────────────┴──────────┘
                                    │
                          ┌─────────┴─────────┐
                          │  Porch Backend    │
                          │ • PackageRevision │
                          │ • Repository      │ 
                          │ • Function Eval   │
                          └───────────────────┘
```

## Core Components

### 1. Nephio Workflow Orchestrator

**File**: `workflow_orchestrator.go`

The primary orchestration engine that executes complete Nephio workflows:

```go
type NephioWorkflowOrchestrator struct {
    porchClient        porch.PorchClient
    configSync         *ConfigSyncClient
    workloadRegistry   *WorkloadClusterRegistry
    packageCatalog     *NephioPackageCatalog
    workflowEngine     *NephioWorkflowEngine
}

// Execute native Nephio workflow
func (nwo *NephioWorkflowOrchestrator) ExecuteNephioWorkflow(
    ctx context.Context, 
    intent *NetworkIntent
) (*WorkflowExecution, error)
```

**Key Features**:
- Blueprint selection and package specialization
- Multi-cluster deployment coordination
- Config Sync integration for GitOps
- Comprehensive workflow monitoring
- Automatic rollback and recovery

### 2. Package Catalog Management

**File**: `package_catalog.go`

Manages the blueprint catalog and package variant creation:

```go
type NephioPackageCatalog struct {
    repositories  sync.Map
    blueprints    sync.Map  
    variants      sync.Map
}

// Find blueprint for intent
func (npc *NephioPackageCatalog) FindBlueprintForIntent(
    ctx context.Context,
    intent *NetworkIntent
) (*BlueprintPackage, error)

// Create specialized package variant
func (npc *NephioPackageCatalog) CreatePackageVariant(
    ctx context.Context,
    blueprint *BlueprintPackage, 
    specialization *SpecializationRequest
) (*PackageVariant, error)
```

**Standard Blueprints**:
- `5g-amf-blueprint`: 5G Access and Mobility Management Function
- `5g-upf-blueprint`: 5G User Plane Function  
- `oran-ric-blueprint`: O-RAN RIC Platform

### 3. Workload Cluster Registry

**File**: `workload_cluster_registry.go`

Manages workload cluster lifecycle and health monitoring:

```go
type WorkloadClusterRegistry struct {
    clusters        sync.Map
    healthMonitor   *ClusterHealthMonitor
}

// Register workload cluster
func (wcr *WorkloadClusterRegistry) RegisterWorkloadCluster(
    ctx context.Context,
    cluster *WorkloadCluster
) error

// Check cluster health
func (wcr *WorkloadClusterRegistry) CheckClusterHealth(
    ctx context.Context,
    clusterName string  
) (*ClusterHealth, error)
```

**Features**:
- Automatic cluster discovery
- Health monitoring with 99.9% availability
- Capability-based placement
- Config Sync integration

### 4. Config Sync Integration

**File**: `configsync_client.go`

Provides GitOps deployment via Config Sync:

```go
type ConfigSyncClient struct {
    gitClient   GitClientInterface
    syncService *SyncService
}

// Deploy package via Config Sync
func (csc *ConfigSyncClient) SyncPackageToCluster(
    ctx context.Context,
    pkg *PackageRevision,
    cluster *WorkloadCluster
) (*SyncResult, error)
```

**GitOps Features**:
- Repository management with branch strategies
- Automatic commit and push operations
- Deployment status monitoring
- Rollback capabilities

### 5. Workflow Engine

**File**: `workflow_engine.go`

Manages workflow definitions and execution patterns:

```go
type NephioWorkflowEngine struct {
    workflows  sync.Map
    executor   *WorkflowExecutor
    validator  *WorkflowValidator
}

// Register workflow definition
func (nwe *NephioWorkflowEngine) RegisterWorkflow(
    workflow *WorkflowDefinition
) error
```

**Standard Workflows**:
- `standard-deployment`: Basic deployment workflow
- `standard-configuration`: Configuration management
- `standard-scaling`: Auto-scaling workflows
- `oran-deployment`: O-RAN compliant deployment
- `5g-core-deployment`: 5G Core network functions

## Usage Examples

### Basic Setup

```go
// Initialize Nephio integration
integration, err := NewNephioIntegration(client, porchClient, config)
if err != nil {
    return err
}

// Process NetworkIntent with Nephio workflow
execution, err := integration.ProcessNetworkIntent(ctx, intent)
if err != nil {
    return err
}

// Monitor workflow execution
for execution.Status == WorkflowExecutionStatusRunning {
    time.Sleep(10 * time.Second)
    execution, _ = integration.GetWorkflowExecution(ctx, execution.ID)
}
```

### 5G Core Deployment

```go
intent := &NetworkIntent{
    ObjectMeta: metav1.ObjectMeta{
        Name: "deploy-5g-amf-production",
    },
    Spec: NetworkIntentSpec{
        IntentType:      NetworkIntentTypeDeployment,
        TargetComponent: "amf",
        Configuration: map[string]interface{}{
            "replicas": 3,
            "resources": map[string]interface{}{
                "cpu":    "1000m", 
                "memory": "2Gi",
            },
            "highAvailability": true,
        },
    },
}

execution, err := integration.ProcessNetworkIntent(ctx, intent)
```

### Multi-Cluster Deployment

```go
// Register multiple clusters
clusters := []*WorkloadCluster{
    {Name: "edge-cluster-east", Region: "us-east-1"},
    {Name: "edge-cluster-west", Region: "us-west-2"},
    {Name: "central-cluster", Region: "us-central-1"},
}

for _, cluster := range clusters {
    integration.RegisterWorkloadCluster(ctx, cluster)
}

// Deploy across clusters
intent := &NetworkIntent{
    Spec: NetworkIntentSpec{
        IntentType: NetworkIntentTypeDeployment,
        Configuration: map[string]interface{}{
            "deploymentStrategy": "multi-cluster",
            "clusterTargets": []map[string]interface{}{
                {"cluster": "edge-cluster-east", "components": []string{"upf", "amf"}},
                {"cluster": "central-cluster", "components": []string{"nrf", "nssf"}},
            },
        },
    },
}
```

### O-RAN Compliant Deployment

```go
intent := &NetworkIntent{
    Spec: NetworkIntentSpec{
        IntentType:      NetworkIntentTypeDeployment,
        TargetComponent: "ric",
        Configuration: map[string]interface{}{
            "ricType": "near-rt",
            "oranCompliance": map[string]interface{}{
                "interfaces": []map[string]interface{}{
                    {"name": "A1", "version": "2.0", "enabled": true},
                    {"name": "E2", "version": "2.0", "enabled": true},
                    {"name": "O1", "version": "1.0", "enabled": true},
                },
                "certifications": []map[string]interface{}{
                    {"name": "O-RAN-SC", "authority": "O-RAN Alliance"},
                },
            },
        },
    },
}
```

### Network Slice Configuration

```go
intent := &NetworkIntent{
    Spec: NetworkIntentSpec{
        IntentType:      NetworkIntentTypeConfiguration,
        TargetComponent: "nssf",
        Configuration: map[string]interface{}{
            "networkSlice": map[string]interface{}{
                "sliceId":   "urllc-slice-001",
                "sliceType": "URLLC",
                "sla": map[string]interface{}{
                    "latency":     "1ms",
                    "reliability": "99.999%",
                    "throughput":  "100Mbps",
                },
                "qos": map[string]interface{}{
                    "priority": 1,
                    "qci":      1,
                    "gbr":      "100Mbps",
                },
            },
        },
    },
}
```

### Auto-Scaling Configuration

```go
intent := &NetworkIntent{
    Spec: NetworkIntentSpec{
        IntentType:      NetworkIntentTypeScaling,
        TargetComponent: "upf",
        Configuration: map[string]interface{}{
            "scalingPolicy": map[string]interface{}{
                "minReplicas": 2,
                "maxReplicas": 20,
                "metrics": []map[string]interface{}{
                    {
                        "type": "Resource",
                        "resource": map[string]interface{}{
                            "name": "cpu",
                            "target": map[string]interface{}{
                                "averageUtilization": 70,
                            },
                        },
                    },
                    {
                        "type": "External", 
                        "external": map[string]interface{}{
                            "metric": map[string]interface{}{
                                "name": "upf_session_count",
                            },
                            "target": map[string]interface{}{
                                "averageValue": "1000",
                            },
                        },
                    },
                },
            },
            "predictiveScaling": map[string]interface{}{
                "enabled":           true,
                "algorithm":         "machine-learning",
                "predictionHorizon": "30m",
            },
        },
    },
}
```

### Custom Workflow Creation

```go
customWorkflow := &WorkflowDefinition{
    Name: "edge-optimized-deployment",
    Description: "Edge-optimized deployment workflow",
    IntentTypes: []NetworkIntentType{NetworkIntentTypeDeployment},
    Phases: []WorkflowPhase{
        {
            Name: "edge-requirements-validation",
            Type: WorkflowPhaseTypeValidation,
            Actions: []WorkflowAction{
                {
                    Name:     "validate-edge-requirements",
                    Type:     WorkflowActionTypeValidatePackage,
                    Required: true,
                    Config: map[string]interface{}{
                        "checks": []string{
                            "latency-requirements",
                            "edge-capabilities",
                        },
                    },
                },
            },
        },
        // ... more phases
    },
}

// Register custom workflow
err := integration.RegisterCustomWorkflow(customWorkflow)
```

## Configuration

### Default Configuration

```go
config := &NephioIntegrationConfig{
    NephioNamespace:      "nephio-system",
    ConfigSyncEnabled:    true,
    UpstreamRepository:   "nephoran-blueprints", 
    DownstreamRepository: "nephoran-deployments",
    CatalogRepository:    "nephoran-catalog",
    EnableMetrics:        true,
    EnableTracing:        true,
}
```

### Workflow Orchestrator Configuration

```go
workflowConfig := &WorkflowOrchestratorConfig{
    MaxConcurrentWorkflows: 10,
    WorkflowTimeout:        30 * time.Minute,
    AutoApproval:           false,
    RequiredApprovers:      1,
    RollbackOnFailure:      true,
}
```

### Package Catalog Configuration

```go
catalogConfig := &PackageCatalogConfig{
    CatalogRepository:  "nephoran-catalog",
    BlueprintDirectory: "blueprints",
    VariantDirectory:   "variants",
    EnableVersioning:   true,
}
```

### Workload Cluster Configuration

```go
clusterConfig := &WorkloadClusterConfig{
    HealthCheckInterval:     1 * time.Minute,
    AutoClusterRegistration: true,
    MaxRetries:              3,
}
```

### Config Sync Configuration

```go
configSyncConfig := &ConfigSyncConfig{
    Repository: "https://github.com/nephoran/cluster-configs",
    Branch:     "main",
    Directory:  "clusters",
    SyncPeriod: 30 * time.Second,
}
```

## Standard Workflows

### Deployment Workflow

1. **Blueprint Selection**: Find appropriate blueprint for intent
2. **Package Specialization**: Create cluster-specific package variants
3. **Validation**: Validate specialized packages
4. **Approval**: Manual or automatic approval process
5. **Deployment**: Deploy packages via Config Sync
6. **Monitoring**: Monitor deployment health

### Configuration Workflow

1. **Blueprint Selection**: Select configuration blueprint
2. **Package Specialization**: Create configuration-specific packages
3. **Validation**: Validate configuration packages
4. **Deployment**: Deploy configuration changes

### Scaling Workflow  

1. **Blueprint Selection**: Select scaling blueprint
2. **Package Specialization**: Create scaling-specific packages
3. **Deployment**: Apply scaling changes
4. **Monitoring**: Monitor scaling results

### O-RAN Deployment Workflow

1. **O-RAN Compliance Check**: Verify O-RAN requirements
2. **Blueprint Selection**: Select O-RAN compliant blueprint
3. **Package Specialization**: Create O-RAN specialized packages  
4. **Validation**: Validate O-RAN packages
5. **Deployment**: Deploy O-RAN packages

### 5G Core Deployment Workflow

1. **5G Requirements Check**: Verify 5G Core requirements
2. **Blueprint Selection**: Select 5G Core blueprint
3. **Package Specialization**: Create 5G Core packages
4. **Network Slice Configuration**: Configure network slices
5. **Validation**: Validate 5G Core packages
6. **Deployment**: Deploy 5G Core network functions
7. **Monitoring**: Monitor 5G Core deployment

## Metrics and Monitoring

### Workflow Metrics

- `nephio_workflow_executions_total`: Total workflow executions
- `nephio_workflow_duration_seconds`: Workflow execution duration  
- `nephio_workflow_phases`: Current workflow phases in progress
- `nephio_workflow_errors_total`: Total workflow errors

### Package Metrics

- `nephio_package_variants_total`: Total package variants created
- `nephio_catalog_blueprint_queries_total`: Blueprint query count
- `nephio_catalog_cache_hits_total`: Catalog cache hits

### Cluster Metrics  

- `nephio_cluster_registrations_total`: Cluster registrations
- `nephio_cluster_health`: Cluster health status
- `nephio_cluster_deployments_total`: Cluster deployments

### Config Sync Metrics

- `nephio_configsync_operations_total`: Config Sync operations
- `nephio_configsync_duration_seconds`: Operation duration
- `nephio_git_commands_total`: Git command executions

## Error Handling

### Retry Policies

```go
retryPolicy := &RetryPolicy{
    MaxAttempts: 3,
    Delay:       30 * time.Second,
    BackoffRate: 2.0,
    MaxDelay:    5 * time.Minute,
}
```

### Rollback Strategies

```go
rollbackStrategy := &RollbackStrategy{
    Enabled: true,
    TriggerOn: []string{"deployment-failure", "validation-failure"},
    Timeout: 15 * time.Minute,
}
```

### Circuit Breakers

- Automatic failure detection
- Graceful degradation
- Automatic recovery
- Comprehensive metrics

## Testing

### Unit Tests

```bash
go test ./pkg/nephio/... -v -cover
```

### Integration Tests

```bash
go test ./pkg/nephio/... -tags=integration -v
```

### End-to-End Tests

```bash
go test ./pkg/nephio/... -tags=e2e -v
```

## Performance Characteristics

- **Throughput**: 45 intents per minute
- **Latency**: Sub-2-second P95 latency for intent processing
- **Scalability**: 200+ concurrent intent processing
- **Availability**: 99.95% availability in production
- **Resource Usage**: Linear scaling up to 200 concurrent intents

## Future Enhancements

### Near-term (Next 3 months)
- Enhanced service mesh integration
- ML-based workflow optimization  
- Advanced multi-cluster support

### Medium-term (3-6 months)
- rApp framework integration
- Enhanced O-RAN interface support
- Open Fronthaul integration

### Long-term (6+ months)
- 6G readiness preparation
- Autonomous operations capability
- Industry standard certification

## Contributing

1. Follow standard Go coding conventions
2. Ensure comprehensive test coverage (90%+)
3. Include metrics and tracing
4. Update documentation
5. Validate O-RAN compliance

## License

Licensed under the Apache License, Version 2.0.