# A1 Policy Management Service - Integration Guide

## Table of Contents

1. [Overview](#overview)
2. [Integration Architecture](#integration-architecture)
3. [Nephoran Controller Integration](#nephoran-controller-integration)
4. [Near-RT RIC Integration](#near-rt-ric-integration)
5. [xApp Integration](#xapp-integration)
6. [Service Mesh Integration](#service-mesh-integration)
7. [Event-Driven Architecture](#event-driven-architecture)
8. [Testing and Validation](#testing-and-validation)
9. [Production Deployment Patterns](#production-deployment-patterns)
10. [Troubleshooting Integration Issues](#troubleshooting-integration-issues)

## Overview

This guide provides comprehensive instructions for integrating the A1 Policy Management Service with various components in the Nephoran Intent Operator ecosystem and O-RAN compliant networks. The service acts as a critical bridge between high-level network intents and concrete policy implementations.

### Integration Principles

- **Loose Coupling**: Components communicate through well-defined interfaces
- **Asynchronous Processing**: Event-driven patterns for high throughput
- **Fault Tolerance**: Circuit breakers and retry mechanisms
- **Observable Integration**: Comprehensive monitoring and tracing
- **Security by Default**: mTLS and authentication at all integration points

### Key Integration Points

```
┌─────────────────────────────────────────────────────────────────────┐
│                 Nephoran Intent Operator Ecosystem                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  ┌─────────────────┐    ┌─────────────────┐    ┌────────────────┐  │
│  │  NetworkIntent  │───▶│ LLM Processor   │───▶│  RAG Service   │  │
│  │  Controller     │    │                 │    │                │  │
│  └─────────────────┘    └─────────────────┘    └────────────────┘  │
│           │                       │                       │        │
│           └───────────────────────┼───────────────────────┘        │
│                                   ▼                                │
│  ┌─────────────────────────────────────────────────────────────────┐  │
│  │                 A1 Policy Management Service                   │  │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐  │
│  │  │    A1-P     │ │    A1-C     │ │   A1-EI     │ │  Internal   │  │
│  │  │ Interface   │ │ Interface   │ │ Interface   │ │  APIs       │  │
│  │  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘  │
│  └─────────────────────────────────────────────────────────────────┘  │
│                                   │                                │
└───────────────────────────────────┼────────────────────────────────┘
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        O-RAN Network Layer                          │
├─────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │
│  │ Near-RT RIC │  │ xApps       │  │ SMO/Non-RT  │  │ Network     │ │
│  │             │  │             │  │ RIC         │  │ Functions   │ │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

## Integration Architecture

### Component Communication Patterns

The A1 Policy Management Service uses several communication patterns for different integration scenarios:

#### 1. Synchronous HTTP/REST
- **Use Case**: CRUD operations, status queries
- **Components**: Nephoran controllers, management interfaces
- **Patterns**: Request-response, circuit breakers

#### 2. Asynchronous Messaging
- **Use Case**: Policy notifications, event propagation
- **Components**: Consumer applications, event subscribers
- **Patterns**: Publish-subscribe, event sourcing

#### 3. Service Mesh (gRPC/HTTP2)
- **Use Case**: Service-to-service communication
- **Components**: Internal microservices, Near-RT RIC
- **Patterns**: Load balancing, traffic management

### Integration Security Model

```yaml
security_layers:
  transport:
    - mTLS between all services
    - Certificate rotation via cert-manager
  application:
    - OAuth2 authentication
    - RBAC authorization
    - API key validation
  network:
    - Network policies for microsegmentation
    - Service mesh security policies
  data:
    - Encryption at rest
    - Secret management via Kubernetes secrets
```

## Nephoran Controller Integration

### NetworkIntent to Policy Translation Flow

The A1 service integrates with Nephoran controllers to translate high-level network intents into concrete A1 policies.

#### Integration Architecture

```go
// IntentPolicyTranslator interface for Nephoran integration
type IntentPolicyTranslator interface {
    TranslateIntent(ctx context.Context, intent *NetworkIntent) (*PolicyTranslation, error)
    ValidateTranslation(ctx context.Context, translation *PolicyTranslation) error
    ApplyPolicies(ctx context.Context, translation *PolicyTranslation) error
    SyncStatus(ctx context.Context, intentName string) error
}

// PolicyTranslation represents the result of intent translation
type PolicyTranslation struct {
    IntentID      string           `json:"intent_id"`
    PolicyTypes   []PolicyType     `json:"policy_types"`
    PolicyInstances []PolicyInstance `json:"policy_instances"`
    ConsumerBindings []ConsumerBinding `json:"consumer_bindings"`
    EIJobs        []EIJobDefinition `json:"ei_jobs"`
    Dependencies  []string         `json:"dependencies"`
    ValidationResults []ValidationResult `json:"validation_results"`
}
```

#### Implementation Example

```go
package integration

import (
    "context"
    "fmt"
    "time"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/a1"
    nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"
)

type NephoranA1Bridge struct {
    a1Client   a1.A1Service
    logger     *logging.StructuredLogger
    metrics    *prometheus.Registry
    translator IntentPolicyTranslator
}

func NewNephoranA1Bridge(config *BridgeConfig) (*NephoranA1Bridge, error) {
    return &NephoranA1Bridge{
        a1Client:   config.A1Client,
        logger:     config.Logger,
        metrics:    config.Metrics,
        translator: NewLLMPolicyTranslator(config.LLMConfig),
    }, nil
}

// ProcessNetworkIntent translates and applies a NetworkIntent as A1 policies
func (b *NephoranA1Bridge) ProcessNetworkIntent(ctx context.Context, intent *nephoran.NetworkIntent) error {
    startTime := time.Now()
    
    // Record processing metrics
    defer func() {
        b.metrics.WithLabelValues("process_intent", "total").
            Observe(time.Since(startTime).Seconds())
    }()
    
    b.logger.InfoWithContext("Processing NetworkIntent", 
        "intent_name", intent.Name,
        "intent_spec", intent.Spec.Intent,
        "priority", intent.Spec.Priority,
    )
    
    // Step 1: Translate intent to policy specifications
    translation, err := b.translator.TranslateIntent(ctx, intent)
    if err != nil {
        return fmt.Errorf("failed to translate intent: %w", err)
    }
    
    // Step 2: Validate the translation
    if err := b.translator.ValidateTranslation(ctx, translation); err != nil {
        return fmt.Errorf("translation validation failed: %w", err)
    }
    
    // Step 3: Create required policy types
    for _, policyType := range translation.PolicyTypes {
        err := b.a1Client.CreatePolicyType(ctx, policyType.PolicyTypeID, &policyType)
        if err != nil {
            b.logger.WarnWithContext("Failed to create policy type", 
                "policy_type_id", policyType.PolicyTypeID,
                "error", err,
            )
            // Continue with existing policy type
        }
    }
    
    // Step 4: Create policy instances
    for _, instance := range translation.PolicyInstances {
        err := b.a1Client.CreatePolicyInstance(ctx, 
            instance.PolicyTypeID, 
            instance.PolicyID, 
            &instance,
        )
        if err != nil {
            return fmt.Errorf("failed to create policy instance %s: %w", 
                instance.PolicyID, err)
        }
        
        b.logger.InfoWithContext("Created policy instance", 
            "policy_id", instance.PolicyID,
            "policy_type_id", instance.PolicyTypeID,
        )
    }
    
    // Step 5: Register consumers and setup notifications
    for _, binding := range translation.ConsumerBindings {
        err := b.a1Client.RegisterConsumer(ctx, binding.ConsumerID, binding.ConsumerInfo)
        if err != nil {
            b.logger.WarnWithContext("Failed to register consumer", 
                "consumer_id", binding.ConsumerID,
                "error", err,
            )
        }
    }
    
    // Step 6: Create EI jobs if specified
    for _, jobDef := range translation.EIJobs {
        job := &a1.EnrichmentInfoJob{
            EiJobID:    jobDef.JobID,
            EiTypeID:   jobDef.TypeID,
            EiJobData:  jobDef.JobData,
            TargetURI:  jobDef.TargetURI,
            JobOwner:   "nephoran-intent-operator",
        }
        
        err := b.a1Client.CreateEIJob(ctx, jobDef.JobID, job)
        if err != nil {
            b.logger.WarnWithContext("Failed to create EI job", 
                "job_id", jobDef.JobID,
                "error", err,
            )
        }
    }
    
    // Step 7: Update NetworkIntent status
    return b.updateIntentStatus(ctx, intent, translation)
}

// StatusSyncLoop continuously syncs A1 policy status back to NetworkIntent
func (b *NephoranA1Bridge) StatusSyncLoop(ctx context.Context, syncInterval time.Duration) error {
    ticker := time.NewTicker(syncInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if err := b.syncAllIntentStatuses(ctx); err != nil {
                b.logger.ErrorWithContext("Status sync failed", err)
            }
        }
    }
}

func (b *NephoranA1Bridge) syncAllIntentStatuses(ctx context.Context) error {
    // Implementation would fetch all NetworkIntents and sync their policy statuses
    return nil
}

func (b *NephoranA1Bridge) updateIntentStatus(ctx context.Context, intent *nephoran.NetworkIntent, translation *PolicyTranslation) error {
    // Update NetworkIntent status with policy deployment results
    status := &nephoran.NetworkIntentStatus{
        Phase: "PolicyDeployed",
        Conditions: []nephoran.NetworkIntentCondition{
            {
                Type:   "PolicyTranslationReady",
                Status: "True",
                LastTransitionTime: metav1.Now(),
                Reason: "TranslationSuccessful",
                Message: fmt.Sprintf("Translated to %d policy instances", 
                    len(translation.PolicyInstances)),
            },
            {
                Type:   "PoliciesDeployed", 
                Status: "True",
                LastTransitionTime: metav1.Now(),
                Reason: "DeploymentSuccessful",
                Message: "All policies successfully deployed to A1 interface",
            },
        },
        PolicyDeployment: &nephoran.PolicyDeploymentStatus{
            PolicyTypes:      len(translation.PolicyTypes),
            PolicyInstances:  len(translation.PolicyInstances),
            ConsumerBindings: len(translation.ConsumerBindings),
            EIJobs:          len(translation.EIJobs),
        },
    }
    
    // Update status (implementation depends on controller-runtime client)
    return nil
}
```

### Configuration Integration

```yaml
# Nephoran controller configuration for A1 integration
apiVersion: v1
kind: ConfigMap
metadata:
  name: nephoran-a1-bridge-config
data:
  config.yaml: |
    a1_service:
      endpoint: "https://a1-policy-service.nephoran-a1.svc.cluster.local"
      timeout: "30s"
      retry_attempts: 3
      circuit_breaker:
        failure_threshold: 5
        recovery_timeout: "60s"
    
    intent_translation:
      llm_processor_endpoint: "http://llm-processor.nephoran-system.svc.cluster.local:8080"
      rag_service_endpoint: "http://rag-api.nephoran-system.svc.cluster.local:8080"
      policy_templates_path: "/etc/templates/policy-types"
      validation_strict: true
    
    status_sync:
      interval: "30s"
      batch_size: 50
      parallel_workers: 5
    
    metrics:
      enabled: true
      port: 9090
      path: "/metrics"
```

## Near-RT RIC Integration

### A1 Interface Implementation

The A1 service provides standard O-RAN A1 interfaces for Near-RT RIC integration.

#### Policy Lifecycle Management

```go
// Near-RT RIC integration for policy enforcement
type NearRTRICClient struct {
    baseURL    string
    httpClient *http.Client
    logger     *logging.StructuredLogger
}

func NewNearRTRICClient(config *NearRTRICConfig) *NearRTRICClient {
    return &NearRTRICClient{
        baseURL:    config.BaseURL,
        httpClient: createHTTPClient(config),
        logger:     config.Logger,
    }
}

// RegisterWithA1Service registers the Near-RT RIC with A1 Policy Service
func (c *NearRTRICClient) RegisterWithA1Service(ctx context.Context, a1Endpoint string) error {
    consumerInfo := &a1.ConsumerInfo{
        ConsumerID:   "near-rt-ric-001",
        ConsumerName: "Near-RT RIC Main Instance",
        CallbackURL:  fmt.Sprintf("%s/a1-notifications", c.baseURL),
        Capabilities: []string{"policy_notifications", "status_updates"},
        Metadata: a1.ConsumerMetadata{
            Version:        "3.0.0",
            Description:    "O-RAN compliant Near-RT RIC",
            SupportedTypes: []int{20008, 20009, 20010, 20011},
            AdditionalInfo: map[string]interface{}{
                "ric_id":      "ric-east-001",
                "location":    "east-region",
                "capabilities": []string{"traffic_steering", "qos_management", "energy_saving"},
            },
        },
    }
    
    // Register consumer with A1 service
    client := &http.Client{Timeout: 30 * time.Second}
    body, _ := json.Marshal(consumerInfo)
    
    req, err := http.NewRequestWithContext(ctx, "POST", 
        fmt.Sprintf("%s/A1-C/v1/consumers/near-rt-ric-001", a1Endpoint),
        bytes.NewBuffer(body))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+getServiceToken())
    
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusCreated {
        return fmt.Errorf("registration failed with status %d", resp.StatusCode)
    }
    
    c.logger.InfoWithContext("Successfully registered with A1 Policy Service",
        "a1_endpoint", a1Endpoint,
        "consumer_id", "near-rt-ric-001")
    
    return nil
}

// HandlePolicyNotification processes incoming policy notifications from A1
func (c *NearRTRICClient) HandlePolicyNotification(w http.ResponseWriter, r *http.Request) {
    var notification a1.PolicyNotification
    
    if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
        http.Error(w, "Invalid notification format", http.StatusBadRequest)
        return
    }
    
    c.logger.InfoWithContext("Received policy notification",
        "notification_type", notification.NotificationType,
        "policy_id", notification.PolicyID,
        "policy_type_id", notification.PolicyTypeID,
        "status", notification.Status,
    )
    
    switch notification.NotificationType {
    case "CREATE":
        if err := c.handlePolicyCreate(r.Context(), &notification); err != nil {
            c.logger.ErrorWithContext("Failed to handle policy creation", err,
                "policy_id", notification.PolicyID)
            http.Error(w, "Policy creation handling failed", http.StatusInternalServerError)
            return
        }
        
    case "UPDATE":
        if err := c.handlePolicyUpdate(r.Context(), &notification); err != nil {
            c.logger.ErrorWithContext("Failed to handle policy update", err,
                "policy_id", notification.PolicyID)
            http.Error(w, "Policy update handling failed", http.StatusInternalServerError)
            return
        }
        
    case "DELETE":
        if err := c.handlePolicyDelete(r.Context(), &notification); err != nil {
            c.logger.ErrorWithContext("Failed to handle policy deletion", err,
                "policy_id", notification.PolicyID)
            http.Error(w, "Policy deletion handling failed", http.StatusInternalServerError)
            return
        }
        
    default:
        c.logger.WarnWithContext("Unknown notification type",
            "notification_type", notification.NotificationType,
            "policy_id", notification.PolicyID)
        http.Error(w, "Unknown notification type", http.StatusBadRequest)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "acknowledged",
        "timestamp": time.Now().UTC().Format(time.RFC3339),
    })
}

func (c *NearRTRICClient) handlePolicyCreate(ctx context.Context, notification *a1.PolicyNotification) error {
    // Fetch the full policy instance from A1 service
    // Apply the policy to relevant xApps
    // Update policy status
    return c.deployPolicyToXApps(ctx, notification.PolicyID, notification.PolicyTypeID)
}

func (c *NearRTRICClient) deployPolicyToXApps(ctx context.Context, policyID string, policyTypeID int) error {
    // Implementation would:
    // 1. Determine which xApps should receive this policy
    // 2. Transform policy data to xApp-specific format
    // 3. Deploy to relevant xApps
    // 4. Monitor deployment status
    // 5. Report back to A1 service
    
    c.logger.InfoWithContext("Deploying policy to xApps",
        "policy_id", policyID,
        "policy_type_id", policyTypeID)
    
    // Simulate xApp deployment
    time.Sleep(100 * time.Millisecond)
    
    return nil
}
```

### Status Reporting Integration

```go
// ReportPolicyStatus reports policy enforcement status back to A1 service
func (c *NearRTRICClient) ReportPolicyStatus(ctx context.Context, policyID string, policyTypeID int, status string, reason string) error {
    a1Client := &http.Client{Timeout: 10 * time.Second}
    
    statusUpdate := map[string]interface{}{
        "enforcement_status": status,
        "enforcement_reason": reason,
        "timestamp":         time.Now().UTC().Format(time.RFC3339),
        "ric_id":            "ric-east-001",
    }
    
    body, _ := json.Marshal(statusUpdate)
    req, err := http.NewRequestWithContext(ctx, "POST",
        fmt.Sprintf("%s/A1-P/v2/policytypes/%d/policies/%s/status", 
            c.getA1ServiceEndpoint(), policyTypeID, policyID),
        bytes.NewBuffer(body))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+getServiceToken())
    
    resp, err := a1Client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return fmt.Errorf("status report failed with code %d", resp.StatusCode)
    }
    
    return nil
}
```

## xApp Integration

### xApp Policy Consumption Pattern

xApps consume policies through the A1-C consumer interface and receive notifications.

#### xApp Consumer Implementation

```go
package xapp

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/a1"
)

type TrafficSteeringXApp struct {
    consumerID      string
    a1ServiceURL    string
    callbackPort    int
    httpServer      *http.Server
    logger          *logging.StructuredLogger
    policyStore     *PolicyStore
    ricConnector    *RICConnector
}

func NewTrafficSteeringXApp(config *XAppConfig) *TrafficSteeringXApp {
    return &TrafficSteeringXApp{
        consumerID:   "traffic-steering-xapp",
        a1ServiceURL: config.A1ServiceURL,
        callbackPort: config.CallbackPort,
        logger:       config.Logger,
        policyStore:  NewPolicyStore(),
        ricConnector: NewRICConnector(config.RICConfig),
    }
}

// Start initializes the xApp and registers with A1 service
func (app *TrafficSteeringXApp) Start(ctx context.Context) error {
    // Register with A1 Policy Service
    if err := app.registerWithA1Service(ctx); err != nil {
        return fmt.Errorf("failed to register with A1 service: %w", err)
    }
    
    // Start HTTP server for policy notifications
    if err := app.startNotificationServer(ctx); err != nil {
        return fmt.Errorf("failed to start notification server: %w", err)
    }
    
    // Start policy enforcement loop
    go app.policyEnforcementLoop(ctx)
    
    app.logger.InfoWithContext("Traffic Steering xApp started successfully",
        "consumer_id", app.consumerID,
        "callback_port", app.callbackPort,
    )
    
    return nil
}

func (app *TrafficSteeringXApp) registerWithA1Service(ctx context.Context) error {
    consumerInfo := &a1.ConsumerInfo{
        ConsumerID:   app.consumerID,
        ConsumerName: "Traffic Steering xApp",
        CallbackURL:  fmt.Sprintf("http://%s:%d/a1-notifications", 
            app.getExternalIP(), app.callbackPort),
        Capabilities: []string{"policy_notifications", "status_updates"},
        Metadata: a1.ConsumerMetadata{
            Version:        "1.3.0",
            Description:    "Intelligent traffic steering and load balancing",
            SupportedTypes: []int{20008}, // Traffic steering policy type
            AdditionalInfo: map[string]interface{}{
                "xapp_type":     "traffic_steering",
                "deployment_id": "ts-xapp-prod-001",
                "features":      []string{"load_balancing", "failover", "optimization"},
            },
        },
    }
    
    client := &http.Client{Timeout: 30 * time.Second}
    body, _ := json.Marshal(consumerInfo)
    
    req, err := http.NewRequestWithContext(ctx, "POST",
        fmt.Sprintf("%s/A1-C/v1/consumers/%s", app.a1ServiceURL, app.consumerID),
        bytes.NewBuffer(body))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+app.getXAppToken())
    
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == http.StatusCreated {
        app.logger.InfoWithContext("Successfully registered with A1 Policy Service")
        return nil
    }
    
    return fmt.Errorf("registration failed with status %d", resp.StatusCode)
}

func (app *TrafficSteeringXApp) startNotificationServer(ctx context.Context) error {
    mux := http.NewServeMux()
    mux.HandleFunc("/a1-notifications", app.handlePolicyNotification)
    mux.HandleFunc("/health", app.handleHealthCheck)
    mux.HandleFunc("/metrics", app.handleMetrics)
    
    app.httpServer = &http.Server{
        Addr:    fmt.Sprintf(":%d", app.callbackPort),
        Handler: mux,
    }
    
    go func() {
        if err := app.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            app.logger.ErrorWithContext("Notification server failed", err)
        }
    }()
    
    return nil
}

func (app *TrafficSteeringXApp) handlePolicyNotification(w http.ResponseWriter, r *http.Request) {
    var notification a1.PolicyNotification
    
    if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
        app.logger.ErrorWithContext("Failed to decode policy notification", err)
        http.Error(w, "Invalid notification format", http.StatusBadRequest)
        return
    }
    
    app.logger.InfoWithContext("Received policy notification",
        "notification_type", notification.NotificationType,
        "policy_id", notification.PolicyID,
        "policy_type_id", notification.PolicyTypeID,
    )
    
    // Validate policy type compatibility
    if !app.supportsPolicyType(notification.PolicyTypeID) {
        app.logger.WarnWithContext("Unsupported policy type",
            "policy_type_id", notification.PolicyTypeID)
        http.Error(w, "Unsupported policy type", http.StatusBadRequest)
        return
    }
    
    // Process the notification
    switch notification.NotificationType {
    case "CREATE":
        if err := app.handlePolicyCreate(r.Context(), &notification); err != nil {
            app.logger.ErrorWithContext("Failed to handle policy creation", err)
            http.Error(w, "Policy creation failed", http.StatusInternalServerError)
            return
        }
        
    case "UPDATE":
        if err := app.handlePolicyUpdate(r.Context(), &notification); err != nil {
            app.logger.ErrorWithContext("Failed to handle policy update", err)
            http.Error(w, "Policy update failed", http.StatusInternalServerError)
            return
        }
        
    case "DELETE":
        if err := app.handlePolicyDelete(r.Context(), &notification); err != nil {
            app.logger.ErrorWithContext("Failed to handle policy deletion", err)
            http.Error(w, "Policy deletion failed", http.StatusInternalServerError)
            return
        }
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "acknowledged"})
}

func (app *TrafficSteeringXApp) handlePolicyCreate(ctx context.Context, notification *a1.PolicyNotification) error {
    // Fetch the full policy instance from A1 service
    policyInstance, err := app.fetchPolicyInstance(ctx, 
        notification.PolicyTypeID, notification.PolicyID)
    if err != nil {
        return fmt.Errorf("failed to fetch policy instance: %w", err)
    }
    
    // Validate policy data
    if err := app.validateTrafficSteeringPolicy(policyInstance); err != nil {
        return fmt.Errorf("policy validation failed: %w", err)
    }
    
    // Store the policy
    app.policyStore.StorPolicy(notification.PolicyID, policyInstance)
    
    // Apply the policy to traffic steering logic
    if err := app.applyTrafficSteeringPolicy(ctx, policyInstance); err != nil {
        return fmt.Errorf("failed to apply policy: %w", err)
    }
    
    // Report successful enforcement
    return app.reportPolicyStatus(ctx, notification.PolicyID, 
        notification.PolicyTypeID, "ENFORCED", "Policy successfully applied")
}

func (app *TrafficSteeringXApp) validateTrafficSteeringPolicy(policy *a1.PolicyInstance) error {
    policyData := policy.PolicyData
    
    // Validate scope
    scope, ok := policyData["scope"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("invalid scope format")
    }
    
    cellList, ok := scope["cell_list"].([]interface{})
    if !ok || len(cellList) == 0 {
        return fmt.Errorf("cell_list is required and must not be empty")
    }
    
    // Validate QoS preference
    qosPreference, ok := policyData["qos_preference"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("qos_preference is required")
    }
    
    loadBalancing, ok := qosPreference["load_balancing"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("load_balancing configuration is required")
    }
    
    // Validate weight values
    siteAWeight := int(loadBalancing["site_a_weight"].(float64))
    siteBWeight := int(loadBalancing["site_b_weight"].(float64))
    
    if siteAWeight+siteBWeight != 100 {
        return fmt.Errorf("site weights must sum to 100, got %d", siteAWeight+siteBWeight)
    }
    
    return nil
}

func (app *TrafficSteeringXApp) applyTrafficSteeringPolicy(ctx context.Context, policy *a1.PolicyInstance) error {
    // Extract policy parameters
    scope := policy.PolicyData["scope"].(map[string]interface{})
    qosPreference := policy.PolicyData["qos_preference"].(map[string]interface{})
    loadBalancing := qosPreference["load_balancing"].(map[string]interface{})
    
    cellList := scope["cell_list"].([]interface{})
    siteAWeight := int(loadBalancing["site_a_weight"].(float64))
    siteBWeight := int(loadBalancing["site_b_weight"].(float64))
    
    app.logger.InfoWithContext("Applying traffic steering policy",
        "policy_id", policy.PolicyID,
        "cells", len(cellList),
        "site_a_weight", siteAWeight,
        "site_b_weight", siteBWeight,
    )
    
    // Configure traffic steering algorithm
    steeringConfig := &TrafficSteeringConfig{
        PolicyID:     policy.PolicyID,
        TargetCells:  convertToStringSlice(cellList),
        SiteAWeight:  siteAWeight,
        SiteBWeight:  siteBWeight,
        UpdatedAt:    time.Now(),
    }
    
    // Apply configuration to RIC connector
    return app.ricConnector.ApplySteeringConfig(ctx, steeringConfig)
}

func (app *TrafficSteeringXApp) policyEnforcementLoop(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            app.enforceActivePolicies(ctx)
        }
    }
}

func (app *TrafficSteeringXApp) enforceActivePolicies(ctx context.Context) {
    policies := app.policyStore.GetActivePolicies()
    
    for _, policy := range policies {
        // Check if policy is still effective
        if !app.isPolicyEffective(policy) {
            app.logger.WarnWithContext("Policy is no longer effective",
                "policy_id", policy.PolicyID)
            continue
        }
        
        // Apply real-time traffic steering
        if err := app.executeTrafficSteering(ctx, policy); err != nil {
            app.logger.ErrorWithContext("Failed to execute traffic steering", err,
                "policy_id", policy.PolicyID)
        }
    }
}

// Utility functions
func (app *TrafficSteeringXApp) fetchPolicyInstance(ctx context.Context, policyTypeID int, policyID string) (*a1.PolicyInstance, error) {
    client := &http.Client{Timeout: 10 * time.Second}
    
    req, err := http.NewRequestWithContext(ctx, "GET",
        fmt.Sprintf("%s/A1-P/v2/policytypes/%d/policies/%s", 
            app.a1ServiceURL, policyTypeID, policyID), nil)
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Authorization", "Bearer "+app.getXAppToken())
    
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("failed to fetch policy: status %d", resp.StatusCode)
    }
    
    var policy a1.PolicyInstance
    if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
        return nil, err
    }
    
    return &policy, nil
}

func (app *TrafficSteeringXApp) reportPolicyStatus(ctx context.Context, policyID string, policyTypeID int, status, reason string) error {
    // Report policy status back to A1 service via Near-RT RIC
    return app.ricConnector.ReportPolicyStatus(ctx, policyID, policyTypeID, status, reason)
}
```

## Service Mesh Integration

### Istio Service Mesh Configuration

```yaml
# Istio configuration for A1 service mesh integration
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: a1-policy-service
  namespace: nephoran-a1
spec:
  hosts:
  - a1-policy-service
  http:
  - match:
    - uri:
        prefix: "/A1-P/v2"
    route:
    - destination:
        host: a1-policy-service
        port:
          number: 8080
    timeout: 30s
    retries:
      attempts: 3
      perTryTimeout: 10s
      retryOn: gateway-error,connect-failure,refused-stream
  - match:
    - uri:
        prefix: "/A1-C/v1"
    route:
    - destination:
        host: a1-policy-service
        port:
          number: 8080
    timeout: 30s
  - match:
    - uri:
        prefix: "/A1-EI/v1"
    route:
    - destination:
        host: a1-policy-service
        port:
          number: 8080
    timeout: 60s  # Longer timeout for EI jobs

---
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: a1-policy-service
  namespace: nephoran-a1
spec:
  host: a1-policy-service
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 50
        http2MaxRequests: 100
        maxRequestsPerConnection: 10
        maxRetries: 3
    loadBalancer:
      simple: LEAST_CONN
    circuitBreaker:
      consecutiveErrors: 5
      interval: 30s
      baseEjectionTime: 30s
      maxEjectionPercent: 50
  portLevelSettings:
  - port:
      number: 8080
    connectionPool:
      tcp:
        maxConnections: 50

---
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: a1-policy-service
  namespace: nephoran-a1
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: a1-policy-service
  mtls:
    mode: STRICT

---
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: a1-policy-service
  namespace: nephoran-a1
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: a1-policy-service
  rules:
  # Allow access from Nephoran controllers
  - from:
    - source:
        namespaces: ["nephoran-system"]
    - source:
        principals: ["cluster.local/ns/nephoran-system/sa/nephoran-controller"]
    to:
    - operation:
        methods: ["GET", "POST", "PUT", "DELETE"]
        paths: ["/A1-P/v2/*", "/A1-C/v1/*", "/A1-EI/v1/*"]
  
  # Allow access from Near-RT RIC
  - from:
    - source:
        namespaces: ["oran"]
        principals: ["cluster.local/ns/oran/sa/near-rt-ric"]
    to:
    - operation:
        methods: ["GET", "POST", "PUT", "DELETE"]
  
  # Allow health checks from monitoring
  - from:
    - source:
        namespaces: ["monitoring", "istio-system"]
    to:
    - operation:
        methods: ["GET"]
        paths: ["/health", "/ready", "/metrics"]
```

## Event-Driven Architecture

### Kafka Integration for Policy Events

```go
// Kafka event producer for policy lifecycle events
type A1EventProducer struct {
    producer kafka.Producer
    logger   *logging.StructuredLogger
    config   *EventConfig
}

type PolicyEvent struct {
    EventType    string                 `json:"event_type"`
    Timestamp    time.Time              `json:"timestamp"`
    Source       string                 `json:"source"`
    PolicyID     string                 `json:"policy_id,omitempty"`
    PolicyTypeID int                    `json:"policy_type_id,omitempty"`
    ConsumerID   string                 `json:"consumer_id,omitempty"`
    EIJobID      string                 `json:"ei_job_id,omitempty"`
    Data         map[string]interface{} `json:"data"`
    TraceID      string                 `json:"trace_id"`
    SpanID       string                 `json:"span_id"`
}

func (p *A1EventProducer) PublishPolicyEvent(ctx context.Context, event *PolicyEvent) error {
    event.Timestamp = time.Now().UTC()
    event.Source = "a1-policy-service"
    
    // Add tracing information
    if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
        event.TraceID = span.SpanContext().TraceID().String()
        event.SpanID = span.SpanContext().SpanID().String()
    }
    
    topic := p.getTopicForEvent(event.EventType)
    key := p.getEventKey(event)
    
    data, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }
    
    msg := &kafka.Message{
        Topic: topic,
        Key:   []byte(key),
        Value: data,
        Headers: []kafka.Header{
            {Key: "event_type", Value: []byte(event.EventType)},
            {Key: "source", Value: []byte(event.Source)},
            {Key: "trace_id", Value: []byte(event.TraceID)},
        },
    }
    
    return p.producer.WriteMessages(ctx, msg)
}

// Event topics configuration
func (p *A1EventProducer) getTopicForEvent(eventType string) string {
    switch eventType {
    case "policy_created", "policy_updated", "policy_deleted":
        return "a1.policy.lifecycle"
    case "consumer_registered", "consumer_unregistered":
        return "a1.consumer.lifecycle"
    case "ei_job_created", "ei_job_updated", "ei_job_deleted":
        return "a1.ei.lifecycle"
    case "policy_enforcement_status":
        return "a1.policy.status"
    default:
        return "a1.events.general"
    }
}
```

### Event Consumers

```go
// Event consumer for external integrations
type A1EventConsumer struct {
    consumer kafka.Consumer
    handlers map[string]EventHandler
    logger   *logging.StructuredLogger
}

type EventHandler func(ctx context.Context, event *PolicyEvent) error

func (c *A1EventConsumer) Start(ctx context.Context) error {
    topics := []string{
        "a1.policy.lifecycle",
        "a1.consumer.lifecycle", 
        "a1.ei.lifecycle",
        "a1.policy.status",
    }
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            msg, err := c.consumer.ReadMessage(ctx)
            if err != nil {
                c.logger.ErrorWithContext("Failed to read message", err)
                continue
            }
            
            if err := c.processMessage(ctx, msg); err != nil {
                c.logger.ErrorWithContext("Failed to process message", err)
            }
        }
    }
}

func (c *A1EventConsumer) processMessage(ctx context.Context, msg *kafka.Message) error {
    var event PolicyEvent
    if err := json.Unmarshal(msg.Value, &event); err != nil {
        return fmt.Errorf("failed to unmarshal event: %w", err)
    }
    
    // Add tracing context
    if event.TraceID != "" && event.SpanID != "" {
        // Reconstruct tracing context for distributed tracing
        ctx = c.addTracingContext(ctx, event.TraceID, event.SpanID)
    }
    
    handler, ok := c.handlers[event.EventType]
    if !ok {
        c.logger.WarnWithContext("No handler for event type", 
            "event_type", event.EventType)
        return nil
    }
    
    return handler(ctx, &event)
}

// Register event handlers for different integrations
func (c *A1EventConsumer) RegisterHandlers() {
    // Handler for Near-RT RIC notifications
    c.handlers["policy_created"] = func(ctx context.Context, event *PolicyEvent) error {
        // Notify Near-RT RIC about new policy
        return c.notifyNearRTRIC(ctx, event)
    }
    
    // Handler for analytics and monitoring
    c.handlers["policy_enforcement_status"] = func(ctx context.Context, event *PolicyEvent) error {
        // Send policy status to analytics platform
        return c.sendToAnalytics(ctx, event)
    }
    
    // Handler for audit logging
    c.handlers["policy_deleted"] = func(ctx context.Context, event *PolicyEvent) error {
        // Log policy deletion for audit
        return c.auditPolicyDeletion(ctx, event)
    }
}
```

## Testing and Validation

### Integration Test Suite

```go
package integration_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
    "github.com/testcontainers/testcontainers-go"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/a1"
    "github.com/thc1006/nephoran-intent-operator/tests/integration"
)

type A1IntegrationTestSuite struct {
    suite.Suite
    
    a1Service       *a1.A1Server
    testContainers  *integration.TestContainers
    nephoranBridge  *integration.NephoranA1Bridge
    nearRTRICMock   *integration.NearRTRICMock
    xAppMock        *integration.XAppMock
}

func (suite *A1IntegrationTestSuite) SetupSuite() {
    // Start test containers
    suite.testContainers = integration.StartTestContainers(suite.T())
    
    // Start A1 service
    suite.a1Service = integration.StartA1Service(suite.T(), suite.testContainers)
    
    // Start mock services
    suite.nearRTRICMock = integration.StartNearRTRICMock(suite.T())
    suite.xAppMock = integration.StartXAppMock(suite.T())
    
    // Initialize Nephoran bridge
    suite.nephoranBridge = integration.NewNephoranA1Bridge(suite.T(), suite.a1Service)
}

func (suite *A1IntegrationTestSuite) TearDownSuite() {
    suite.testContainers.Cleanup()
    suite.nearRTRICMock.Stop()
    suite.xAppMock.Stop()
}

func (suite *A1IntegrationTestSuite) TestNetworkIntentToPolicyTranslation() {
    ctx := context.Background()
    
    // Create a NetworkIntent
    intent := &nephoran.NetworkIntent{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "traffic-optimization-intent",
            Namespace: "test",
        },
        Spec: nephoran.NetworkIntentSpec{
            Intent:   "Implement load balancing for the 5G network with 70% traffic to site A and 30% to site B",
            Priority: "high",
            Scope:    "regional",
        },
    }
    
    // Process the intent
    err := suite.nephoranBridge.ProcessNetworkIntent(ctx, intent)
    assert.NoError(suite.T(), err, "Intent processing should succeed")
    
    // Verify policy creation
    policyTypes, err := suite.a1Service.GetPolicyTypes(ctx)
    assert.NoError(suite.T(), err)
    assert.NotEmpty(suite.T(), policyTypes, "Policy types should be created")
    
    // Verify policy instance creation
    instances, err := suite.a1Service.GetPolicyInstances(ctx, 20008)
    assert.NoError(suite.T(), err)
    assert.NotEmpty(suite.T(), instances, "Policy instances should be created")
    
    // Verify consumer notification
    suite.Eventually(func() bool {
        return suite.xAppMock.HasReceivedNotification("CREATE")
    }, 5*time.Second, 100*time.Millisecond, "xApp should receive policy notification")
}

func (suite *A1IntegrationTestSuite) TestPolicyEnforcementFlow() {
    ctx := context.Background()
    
    // Create a policy type
    policyType := &a1.PolicyType{
        PolicyTypeID:   99999,
        PolicyTypeName: "Test Policy",
        Description:    "Test policy for integration testing",
        Schema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "test_param": map[string]interface{}{
                    "type": "string",
                },
            },
        },
    }
    
    err := suite.a1Service.CreatePolicyType(ctx, 99999, policyType)
    assert.NoError(suite.T(), err)
    
    // Create a policy instance
    policyInstance := &a1.PolicyInstance{
        PolicyID:     "test-policy-001",
        PolicyTypeID: 99999,
        PolicyData: map[string]interface{}{
            "test_param": "test_value",
        },
    }
    
    err = suite.a1Service.CreatePolicyInstance(ctx, 99999, "test-policy-001", policyInstance)
    assert.NoError(suite.T(), err)
    
    // Verify Near-RT RIC receives the policy
    suite.Eventually(func() bool {
        return suite.nearRTRICMock.HasReceivedPolicy("test-policy-001")
    }, 10*time.Second, 500*time.Millisecond)
    
    // Simulate policy enforcement status update
    suite.nearRTRICMock.ReportPolicyStatus("test-policy-001", 99999, "ENFORCED", "Policy successfully applied")
    
    // Verify status is updated
    suite.Eventually(func() bool {
        status, err := suite.a1Service.GetPolicyStatus(ctx, 99999, "test-policy-001")
        if err != nil {
            return false
        }
        return status.EnforcementStatus == "ENFORCED"
    }, 5*time.Second, 100*time.Millisecond, "Policy status should be updated to ENFORCED")
}

func (suite *A1IntegrationTestSuite) TestConsumerRegistrationAndNotification() {
    ctx := context.Background()
    
    // Register a consumer
    consumerInfo := &a1.ConsumerInfo{
        ConsumerID:   "test-consumer",
        ConsumerName: "Test Consumer",
        CallbackURL:  suite.xAppMock.GetCallbackURL(),
        Capabilities: []string{"policy_notifications"},
    }
    
    err := suite.a1Service.RegisterConsumer(ctx, "test-consumer", consumerInfo)
    assert.NoError(suite.T(), err)
    
    // Create a policy instance
    policyInstance := &a1.PolicyInstance{
        PolicyID:     "notification-test-policy",
        PolicyTypeID: 20008,
        PolicyData: map[string]interface{}{
            "scope": map[string]interface{}{
                "ue_id":     "*",
                "cell_list": []string{"cell_001"},
            },
            "qos_preference": map[string]interface{}{
                "priority_level": 1,
            },
        },
    }
    
    err = suite.a1Service.CreatePolicyInstance(ctx, 20008, "notification-test-policy", policyInstance)
    assert.NoError(suite.T(), err)
    
    // Verify consumer receives notification
    suite.Eventually(func() bool {
        notifications := suite.xAppMock.GetReceivedNotifications()
        for _, notif := range notifications {
            if notif.PolicyID == "notification-test-policy" && notif.NotificationType == "CREATE" {
                return true
            }
        }
        return false
    }, 5*time.Second, 100*time.Millisecond, "Consumer should receive policy notification")
}

func TestA1IntegrationSuite(t *testing.T) {
    suite.Run(t, new(A1IntegrationTestSuite))
}
```

### Load Testing

```javascript
// K6 load test script for A1 Policy Service
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 10 },   // Ramp up
    { duration: '5m', target: 50 },   // Stay at 50 users
    { duration: '2m', target: 100 },  // Ramp to 100 users
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95% of requests under 500ms
    http_req_failed: ['rate<0.05'],    // Error rate under 5%
    errors: ['rate<0.05'],
  },
};

// Test data
const policyTypes = [20008, 20009, 20010, 20011];
const baseURL = 'https://a1-api.nephoran.io';

// Authentication token (replace with actual token)
const authToken = __ENV.AUTH_TOKEN || 'your-auth-token';

export default function () {
  const policyTypeId = policyTypes[Math.floor(Math.random() * policyTypes.length)];
  const policyId = `load-test-policy-${__VU}-${__ITER}`;
  
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${authToken}`,
  };
  
  // Test 1: Get policy types
  let response = http.get(`${baseURL}/A1-P/v2/policytypes`, { headers });
  check(response, {
    'get policy types status is 200': (r) => r.status === 200,
  }) || errorRate.add(1);
  
  // Test 2: Create policy instance
  const policyData = {
    policy_data: {
      scope: {
        ue_id: '*',
        cell_list: [`cell_${Math.floor(Math.random() * 100)}`],
      },
      qos_preference: {
        priority_level: Math.floor(Math.random() * 15) + 1,
      },
    },
  };
  
  response = http.put(
    `${baseURL}/A1-P/v2/policytypes/${policyTypeId}/policies/${policyId}`,
    JSON.stringify(policyData),
    { headers }
  );
  
  check(response, {
    'create policy status is 201': (r) => r.status === 201,
  }) || errorRate.add(1);
  
  // Test 3: Get policy status
  response = http.get(
    `${baseURL}/A1-P/v2/policytypes/${policyTypeId}/policies/${policyId}/status`,
    { headers }
  );
  
  check(response, {
    'get policy status is 200': (r) => r.status === 200,
  }) || errorRate.add(1);
  
  // Test 4: Delete policy instance
  response = http.del(
    `${baseURL}/A1-P/v2/policytypes/${policyTypeId}/policies/${policyId}`,
    null,
    { headers }
  );
  
  check(response, {
    'delete policy status is 202': (r) => r.status === 202,
  }) || errorRate.add(1);
  
  sleep(1);
}

export function handleSummary(data) {
  return {
    'load-test-results.json': JSON.stringify(data),
    'load-test-summary.html': htmlReport(data),
  };
}
```

## Production Deployment Patterns

### Blue-Green Deployment

```yaml
# Blue-Green deployment configuration
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: a1-policy-service-rollout
  namespace: nephoran-a1
spec:
  replicas: 5
  strategy:
    blueGreen:
      activeService: a1-policy-service-active
      previewService: a1-policy-service-preview
      autoPromotionEnabled: false
      scaleDownDelaySeconds: 30
      prePromotionAnalysis:
        templates:
        - templateName: success-rate
        args:
        - name: service-name
          value: a1-policy-service-preview
      postPromotionAnalysis:
        templates:
        - templateName: success-rate
        args:
        - name: service-name
          value: a1-policy-service-active
  selector:
    matchLabels:
      app: a1-policy-service
  template:
    metadata:
      labels:
        app: a1-policy-service
    spec:
      containers:
      - name: a1-policy-service
        image: ghcr.io/nephoran/a1-policy-service:v1.1.0
        ports:
        - containerPort: 8080
        - containerPort: 9090

---
# Analysis template for automated promotion
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
  namespace: nephoran-a1
spec:
  args:
  - name: service-name
  metrics:
  - name: success-rate
    interval: 10s
    count: 5
    successCondition: result[0] >= 0.95
    provider:
      prometheus:
        address: http://prometheus.monitoring.svc.cluster.local:9090
        query: |
          sum(rate(http_requests_total{service="{{args.service-name}}",code!~"5.."}[2m])) /
          sum(rate(http_requests_total{service="{{args.service-name}}"}[2m]))
```

### Canary Deployment

```yaml
# Canary deployment with Flagger
apiVersion: flagger.app/v1beta1
kind: Canary
metadata:
  name: a1-policy-service
  namespace: nephoran-a1
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: a1-policy-service
  progressDeadlineSeconds: 60
  service:
    port: 8080
    targetPort: 8080
    gateways:
    - a1-policy-gateway
    hosts:
    - a1.nephoran.io
  analysis:
    interval: 15s
    threshold: 5
    maxWeight: 30
    stepWeight: 10
    metrics:
    - name: request-success-rate
      thresholdRange:
        min: 99
      interval: 1m
    - name: request-duration
      thresholdRange:
        max: 500
      interval: 30s
    - name: custom-metric
      templateRef:
        name: a1-policy-success-rate
      thresholdRange:
        min: 95
      interval: 1m
  webhooks:
  - name: acceptance-test
    type: pre-rollout
    url: http://a1-acceptance-tests.nephoran-a1.svc.cluster.local/
    timeout: 30s
    metadata:
      type: bash
      cmd: "curl -sd 'test' a1-policy-service-canary:8080/A1-P/v2/policytypes | grep '200 OK'"
  - name: load-test
    type: rollout
    url: http://a1-load-tester.nephoran-a1.svc.cluster.local/
    metadata:
      cmd: "hey -z 1m -q 10 -c 2 http://a1-policy-service-canary.nephoran-a1:8080/A1-P/v2/policytypes"
```

## Troubleshooting Integration Issues

### Common Integration Problems

#### 1. Authentication Issues

**Problem**: OAuth2 token validation failures
**Symptoms**: 401 Unauthorized responses, authentication errors in logs
**Diagnosis**:
```bash
# Check token validity
curl -H "Authorization: Bearer $TOKEN" \
  https://a1-api.nephoran.io/A1-P/v2/policytypes

# Verify OAuth2 configuration
kubectl get configmap a1-policy-service-config -o yaml
kubectl get secret a1-policy-service-secrets -o yaml
```

**Resolution**:
- Verify OAuth2 issuer configuration
- Check token expiration and refresh logic
- Validate required scopes in token

#### 2. Near-RT RIC Connectivity

**Problem**: Policy notifications not reaching Near-RT RIC
**Symptoms**: Policies created but not enforced, timeout errors
**Diagnosis**:
```bash
# Check network connectivity
kubectl exec a1-policy-service-xxx -- \
  curl -v http://near-rt-ric.oran.svc.cluster.local:8080/health

# Check consumer registrations
curl -H "Authorization: Bearer $TOKEN" \
  https://a1-api.nephoran.io/A1-C/v1/consumers
```

**Resolution**:
- Verify network policies allow communication
- Check service discovery configuration
- Validate consumer callback URLs

#### 3. Policy Translation Failures

**Problem**: NetworkIntent to policy translation errors
**Symptoms**: Intent processing failures, validation errors
**Diagnosis**:
```bash
# Check LLM processor logs
kubectl logs -n nephoran-system deployment/llm-processor

# Check policy validation
kubectl logs -n nephoran-a1 deployment/a1-policy-service | grep validation
```

**Resolution**:
- Verify policy type schemas
- Check LLM processor connectivity
- Validate intent structure

#### 4. Performance Issues

**Problem**: High latency or timeout errors
**Symptoms**: Slow API responses, circuit breaker activation
**Diagnosis**:
```bash
# Check metrics
kubectl port-forward svc/a1-policy-service 9090:9090
curl http://localhost:9090/metrics | grep a1_

# Check resource usage
kubectl top pods -n nephoran-a1
```

**Resolution**:
- Scale up deployment replicas
- Optimize database queries
- Increase timeout values
- Check circuit breaker configuration

### Debugging Tools and Commands

```bash
# Debug A1 service deployment
kubectl describe pod -n nephoran-a1 -l app.kubernetes.io/name=a1-policy-service

# Check service connectivity
kubectl run debug --image=nicolaka/netshoot -it --rm -- /bin/bash
# Inside pod: curl http://a1-policy-service.nephoran-a1.svc.cluster.local:8080/health

# View A1 service logs with filtering
kubectl logs -n nephoran-a1 deployment/a1-policy-service --follow | jq '
  select(.level == "error" or .component == "policy-handler")
'

# Test policy creation manually
kubectl run curl --image=curlimages/curl -it --rm -- /bin/sh
# Inside pod: curl -X PUT "http://a1-policy-service.nephoran-a1:8080/A1-P/v2/policytypes/99999" \
#   -H "Content-Type: application/json" \
#   -d '{"policy_type_name": "Debug Policy", "schema": {"type": "object"}}'

# Monitor events
kubectl get events -n nephoran-a1 --sort-by=.metadata.creationTimestamp

# Check certificate status
kubectl get certificates -n nephoran-a1
kubectl describe certificate a1-policy-service-tls -n nephoran-a1

# Verify RBAC permissions
kubectl auth can-i --list --as=system:serviceaccount:nephoran-a1:a1-policy-service
```

This comprehensive integration guide provides all the necessary information and examples for successfully integrating the A1 Policy Management Service with the Nephoran ecosystem and O-RAN compliant networks.