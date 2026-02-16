# NetworkIntent Controller Implementation Guide

## Overview

The NetworkIntent controller is a sophisticated Kubernetes controller that transforms natural language network intentions into deployable configurations using LLM processing and GitOps patterns. This guide covers the complete implementation with LLM integration, GitOps deployment, and comprehensive error recovery.

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  NetworkIntent  │───▶│ NetworkIntent    │───▶│ Git Repository  │
│  Custom Resource│    │ Controller       │    │ (Deployment)    │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │ LLM Processor    │
                       │ Service          │
                       └──────────────────┘
```

## Features

### 1. LLM Integration
- **Service Integration**: Connects to llm-processor service endpoint
- **Intent Translation**: Sends natural language intent for structured parameter generation
- **Response Processing**: Parses and validates LLM responses
- **Error Handling**: Comprehensive error handling with detailed logging

### 2. GitOps Integration
- **Git Client**: Uses existing git client in `pkg/git/`
- **Automatic Commits**: Commits translated configurations to target repository
- **Deployment Tracking**: Tracks deployment status with commit hashes
- **Repository Management**: Handles repository initialization and authentication

### 3. Error Recovery
- **Retry Logic**: Configurable retry attempts for failed operations
- **Exponential Backoff**: Intelligent retry delays to prevent system overload
- **Status Reporting**: Detailed status conditions and phase tracking
- **Event Logging**: Comprehensive event logging for debugging and monitoring

### 4. Enhanced Status Tracking
- **Processing Phases**: Clear phase indication (Processing, Deploying, Completed)
- **Timing Information**: Start and completion times for each phase
- **Retry Tracking**: Last retry time and attempt counts
- **Git Integration**: Commit hash tracking for deployment verification

## Custom Resource Definition

### NetworkIntent Spec
```yaml
apiVersion: nephoran.com/v1
kind: NetworkIntent
metadata:
  name: example-intent
  namespace: default
spec:
  intent: "Configure a 5G network slice for IoT devices..."
  parameters: # Auto-populated by LLM processing
    bandwidth: "100 Mbps"
    qos_priority: 5
    network_slice_type: "IoT"
```

### NetworkIntent Status
```yaml
status:
  phase: "Completed"
  conditions:
    - type: "Processed"
      status: "True"
      reason: "LLMProcessingSucceeded"
      message: "Intent successfully processed by LLM"
    - type: "Deployed"
      status: "True"
      reason: "GitDeploymentSucceeded"
      message: "Configuration successfully deployed via GitOps"
  processingStartTime: "2025-01-15T10:00:00Z"
  processingCompletionTime: "2025-01-15T10:00:30Z"
  deploymentStartTime: "2025-01-15T10:00:30Z"
  deploymentCompletionTime: "2025-01-15T10:01:00Z"
  gitCommitHash: "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0"
  observedGeneration: 1
```

## Controller Configuration

### Environment Variables
```bash
# LLM Processor Configuration
LLM_PROCESSOR_URL=http://llm-processor.default.svc.cluster.local:8080
LLM_PROCESSOR_TIMEOUT=30s

# Git Repository Configuration
GIT_REPO_URL=https://github.com/your-org/network-configs.git
GIT_BRANCH=main
GIT_TOKEN=your-git-token

# Retry Configuration (optional, defaults provided)
MAX_RETRIES=3
RETRY_DELAY=30s
```

### Initialization Example
```go
package main

import (
    "github.com/thc1006/nephoran-intent-operator/pkg/config"
    "github.com/thc1006/nephoran-intent-operator/pkg/controllers"
)

func main() {
    cfg, err := config.LoadFromEnv()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    reconciler := controllers.NewNetworkIntentReconciler(
        controllers.NetworkIntentReconcilerConfig{
            Client:        mgr.GetClient(),
            Scheme:        mgr.GetScheme(),
            EventRecorder: mgr.GetEventRecorderFor("networkintent-controller"),
            Config:        cfg,
        },
    )

    if err := reconciler.SetupWithManager(mgr); err != nil {
        log.Fatalf("Failed to setup controller: %v", err)
    }
}
```

## Processing Flow

### 1. Intent Processing Phase
1. **Validation**: Controller validates NetworkIntent resource
2. **LLM Request**: Sends intent to LLM processor service
3. **Response Processing**: Parses structured parameters from LLM response
4. **Parameter Update**: Updates NetworkIntent with processed parameters
5. **Status Update**: Marks processing as completed

### 2. GitOps Deployment Phase
1. **File Generation**: Creates deployment files from processed parameters
2. **Repository Initialization**: Ensures git repository is accessible
3. **Commit Creation**: Commits configuration files with descriptive message
4. **Push Operation**: Pushes changes to remote repository
5. **Status Update**: Marks deployment as completed with commit hash

### 3. Error Recovery
1. **Retry Logic**: Implements exponential backoff for failed operations
2. **Status Tracking**: Updates status with retry attempts and failures
3. **Event Recording**: Creates Kubernetes events for monitoring
4. **Max Retry Handling**: Marks as failed after maximum retry attempts

## Status Conditions

### Processing Conditions
- **Processed**: Indicates LLM processing completion status
- **Deployed**: Indicates GitOps deployment completion status

### Condition Reasons
- `LLMProcessingSucceeded`: Intent successfully processed
- `LLMProcessingRetrying`: LLM processing failed, retrying
- `LLMProcessingFailedMaxRetries`: Processing failed after max retries
- `GitDeploymentSucceeded`: Deployment successful
- `GitCommitPushFailed`: Git operations failed
- `GitDeploymentFailedMaxRetries`: Deployment failed after max retries

## Monitoring and Debugging

### Events
The controller creates Kubernetes events for major operations:
```bash
kubectl get events --field-selector involvedObject.kind=NetworkIntent
```

### Status Monitoring
```bash
kubectl get networkintents -o wide
kubectl describe networkintent example-intent
```

### Logs
```bash
kubectl logs -l app=nephoran-intent-operator -f
```

## Deployment Files Generation

The controller generates Kubernetes manifests based on processed parameters:

```json
{
  "apiVersion": "v1",
  "kind": "ConfigMap",
  "metadata": {
    "name": "networkintent-example-intent",
    "namespace": "default",
    "labels": {
      "app.kubernetes.io/name": "networkintent",
      "app.kubernetes.io/instance": "example-intent",
      "app.kubernetes.io/managed-by": "nephoran-intent-operator"
    }
  },
  "data": {
    "bandwidth": "100 Mbps",
    "qos_priority": "5",
    "network_slice_type": "IoT"
  }
}
```

## Best Practices

### 1. Intent Writing
- Be specific and detailed in natural language intentions
- Include technical requirements and constraints
- Specify expected outcomes and configurations

### 2. Repository Management
- Use dedicated branches for different environments
- Implement proper access controls for git repositories
- Monitor git repository size and cleanup old configurations

### 3. Monitoring
- Set up alerts for failed processing or deployments
- Monitor LLM processor service availability
- Track git repository accessibility

### 4. Error Handling
- Review failed NetworkIntent resources regularly
- Investigate retry patterns and adjust configurations
- Monitor event logs for system health

## Troubleshooting

### Common Issues

1. **LLM Processing Failures**
   - Check LLM processor service availability
   - Verify intent complexity and clarity
   - Review processor service logs

2. **Git Deployment Failures**
   - Verify git repository accessibility
   - Check authentication credentials
   - Ensure proper permissions for target repository

3. **Retry Loops**
   - Review retry count annotations
   - Adjust retry delays for system load
   - Investigate underlying service issues

### Resolution Steps

1. Check NetworkIntent status conditions
2. Review controller logs for detailed error messages
3. Verify service dependencies (LLM processor, git repository)
4. Test individual components in isolation
5. Review configuration and environment variables

## Integration Points

### With Existing Services
- **LLM Processor**: HTTP service integration for intent translation
- **RAG API**: Knowledge-base powered intent understanding
- **Git Repository**: Version-controlled configuration storage
- **Kubernetes Events**: Standard Kubernetes monitoring integration

### With GitOps Tools
- **ArgoCD**: Automatic sync of committed configurations
- **Flux**: GitOps-based deployment automation
- **Kustomize**: Configuration management and overlays

This implementation provides a robust, production-ready NetworkIntent controller with comprehensive error handling, monitoring, and GitOps integration capabilities.