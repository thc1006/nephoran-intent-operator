# xApp Development SDK Documentation

## Overview

The Nephoran Intent Operator provides comprehensive SDK support for developing xApps (external applications) that interact with the E2 interface and Near-RT RIC. This documentation covers both Go and Python SDK usage, xApp lifecycle management, and practical examples for common use cases in O-RAN environments.

## Go xApp SDK

### 1. Core SDK Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              Go xApp SDK Architecture                               │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                        Application Layer                                    │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │   │
│  │  │    KPM xApp     │  │     RC xApp     │  │      Custom xApp            │ │   │
│  │  │  (Analytics)    │  │ (Optimization)  │  │    (Business Logic)         │ │   │
│  │  │                 │  │                 │  │                             │ │   │
│  │  │ • Measurements  │  │ • Control       │  │ • Custom Algorithms         │ │   │
│  │  │ • KPI Analysis  │  │ • Traffic Mgmt  │  │ • ML/AI Integration         │ │   │
│  │  │ • Monitoring    │  │ • QoS Policies  │  │ • External APIs             │ │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                          │                                         │
│                                          ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                         xApp SDK Core                                       │   │
│  │                                                                             │   │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │   │
│  │  │                     XAppClient                                      │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │   │
│  │  │  │ Subscription│  │   Control   │  │ Lifecycle   │                 │   │   │
│  │  │  │   Manager   │  │   Manager   │  │   Manager   │                 │   │   │
│  │  │  │             │  │             │  │             │                 │   │   │
│  │  │  │ • Subscribe │  │ • Commands  │  │ • Start     │                 │   │   │
│  │  │  │ • Monitor   │  │ • Policies  │  │ • Stop      │                 │   │   │
│  │  │  │ • Process   │  │ • Control   │  │ • Health    │                 │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘                 │   │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │   │
│  │                                      │                                     │   │
│  │                                      ▼                                     │   │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │   │
│  │  │                E2 Interface Client                                  │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │   │
│  │  │  │    E2AP     │  │   Service   │  │   Message   │                 │   │   │
│  │  │  │  Protocol   │  │   Models    │  │  Handlers   │                 │   │   │
│  │  │  │             │  │             │  │             │                 │   │   │
│  │  │  │ • Messages  │  │ • KPM       │  │ • Async     │                 │   │   │
│  │  │  │ • Codecs    │  │ • RC        │  │ • Callback  │                 │   │   │
│  │  │  │ • Transport │  │ • Custom    │  │ • Event     │                 │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘                 │   │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                          │                                         │
│                                          ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                   Nephoran E2 Manager Interface                            │   │
│  │                                                                             │   │
│  │  HTTP/gRPC Endpoints:                                                      │   │
│  │  • POST /e2/v1/xapps/{xappId}/subscriptions                               │   │
│  │  • POST /e2/v1/xapps/{xappId}/control                                     │   │
│  │  • GET /e2/v1/xapps/{xappId}/indications                                  │   │
│  │  • WebSocket /e2/v1/xapps/{xappId}/stream                                 │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

### 2. Go SDK Installation and Setup

#### Installation

```bash
# Initialize Go module
go mod init my-xapp

# Add Nephoran xApp SDK dependency
go get github.com/thc1006/nephoran-intent-operator/pkg/xapp/sdk
```

#### Basic Setup

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/thc1006/nephoran-intent-operator/pkg/xapp/sdk"
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
)

func main() {
    // Configure xApp
    config := &sdk.XAppConfig{
        AppID:           "my-kmp-analyzer",
        AppName:         "KMP Analytics xApp",
        AppVersion:      "1.0.0",
        E2ManagerURL:    "http://nephoran-e2-manager:8080",
        HealthPort:      8090,
        MetricsPort:     9090,
        LogLevel:        "INFO",
        EnableTracing:   true,
    }

    // Create xApp client
    client, err := sdk.NewXAppClient(config)
    if err != nil {
        log.Fatalf("Failed to create xApp client: %v", err)
    }
    defer client.Close()

    // Start xApp
    ctx := context.Background()
    if err := client.Start(ctx); err != nil {
        log.Fatalf("Failed to start xApp: %v", err)
    }

    log.Println("xApp started successfully")
    
    // Keep running
    select {}
}
```

### 3. XAppClient API Reference

#### Core Interface

```go
// XAppClient provides the main interface for xApp development
type XAppClient interface {
    // Lifecycle management
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    IsHealthy() bool
    GetStatus() *XAppStatus

    // E2 Interface operations
    CreateSubscription(ctx context.Context, req *SubscriptionRequest) (*Subscription, error)
    DeleteSubscription(ctx context.Context, subscriptionID string) error
    SendControlRequest(ctx context.Context, req *ControlRequest) (*ControlResponse, error)
    
    // Event handling
    RegisterIndicationHandler(handler IndicationHandler) error
    RegisterControlAckHandler(handler ControlAckHandler) error
    RegisterEventHandler(eventType string, handler EventHandler) error
    
    // Configuration and monitoring
    UpdateConfiguration(config *XAppConfig) error
    GetMetrics() *XAppMetrics
    GetSubscriptions() []*Subscription
    
    // Service model support
    RegisterServiceModel(model *e2.E2ServiceModel) error
    GetSupportedServiceModels() []*e2.E2ServiceModel
}
```

#### Configuration Structure

```go
type XAppConfig struct {
    // Application identity
    AppID           string `json:"app_id"`
    AppName         string `json:"app_name"`
    AppVersion      string `json:"app_version"`
    AppDescription  string `json:"app_description,omitempty"`
    
    // Connection settings
    E2ManagerURL    string        `json:"e2_manager_url"`
    APIVersion      string        `json:"api_version"`
    Timeout         time.Duration `json:"timeout"`
    RetryCount      int           `json:"retry_count"`
    
    // Server settings
    HealthPort      int    `json:"health_port"`
    MetricsPort     int    `json:"metrics_port"`
    APIPort         int    `json:"api_port,omitempty"`
    
    // Observability
    LogLevel        string `json:"log_level"`
    EnableTracing   bool   `json:"enable_tracing"`
    EnableMetrics   bool   `json:"enable_metrics"`
    
    // Service model preferences
    PreferredModels []string `json:"preferred_models,omitempty"`
    
    // Advanced settings
    BufferSize      int           `json:"buffer_size"`
    WorkerCount     int           `json:"worker_count"`
    HeartbeatInterval time.Duration `json:"heartbeat_interval"`
}
```

### 4. Subscription Management

#### Creating Subscriptions

```go
// KPM Subscription Example
func createKMPSubscription(client sdk.XAppClient) error {
    ctx := context.Background()
    
    request := &sdk.SubscriptionRequest{
        NodeID:          "gnb-001",
        RanFunctionID:   1, // KPM function
        RequestorID:     "my-kmp-analyzer",
        EventTriggers: []sdk.EventTrigger{
            {
                TriggerType:     "PERIODIC",
                ReportingPeriod: 5 * time.Second,
                Conditions: map[string]interface{}{
                    "measurement_types": []string{
                        "DRB.RlcSduDelayDl",
                        "DRB.UEThpDl",
                        "RRU.PrbUsedDl",
                    },
                },
            },
        },
        Actions: []sdk.Action{
            {
                ActionID:   1,
                ActionType: "REPORT",
                ActionDefinition: map[string]interface{}{
                    "granularity_period": 5000,
                    "measurement_info_list": []map[string]interface{}{
                        {
                            "measurement_type": "DRB.RlcSduDelayDl",
                            "label_info_list": []map[string]interface{}{
                                {
                                    "measurement_label": "UE_ID",
                                    "measurement_value": "all",
                                },
                            },
                        },
                    },
                },
            },
        },
    }
    
    subscription, err := client.CreateSubscription(ctx, request)
    if err != nil {
        return fmt.Errorf("failed to create subscription: %w", err)
    }
    
    log.Printf("Created subscription: %s", subscription.ID)
    return nil
}
```

#### Handling Indications

```go
// Indication Handler Implementation
type KMPAnalyzer struct {
    client      sdk.XAppClient
    storage     *MetricsStorage
    analyzer    *MLAnalyzer
    alerts      *AlertManager
}

func (k *KMPAnalyzer) HandleIndication(indication *sdk.Indication) error {
    // Parse KMP indication data
    kmpData, err := k.parseKMPIndication(indication)
    if err != nil {
        return fmt.Errorf("failed to parse KMP indication: %w", err)
    }
    
    // Process measurements
    for _, measurement := range kmpData.Measurements {
        // Store raw data
        if err := k.storage.Store(measurement); err != nil {
            log.Printf("Failed to store measurement: %v", err)
        }
        
        // Run analytics
        if anomaly := k.analyzer.DetectAnomaly(measurement); anomaly != nil {
            // Trigger alert
            if err := k.alerts.TriggerAlert(anomaly); err != nil {
                log.Printf("Failed to trigger alert: %v", err)
            }
            
            // Optionally send control action
            if anomaly.Severity == "CRITICAL" {
                controlReq := k.createMitigationControl(anomaly)
                if _, err := k.client.SendControlRequest(context.Background(), controlReq); err != nil {
                    log.Printf("Failed to send control request: %v", err)
                }
            }
        }
    }
    
    return nil
}

// Register the handler
func (k *KMPAnalyzer) Initialize() error {
    return k.client.RegisterIndicationHandler(k)
}
```

### 5. Control Message Handling

#### Sending Control Messages

```go
// QoS Flow Control Example
func sendQoSControl(client sdk.XAppClient, nodeID, ueID string, qfi int, newQoS *QoSParameters) error {
    ctx := context.Background()
    
    controlRequest := &sdk.ControlRequest{
        NodeID:        nodeID,
        RanFunctionID: 2, // RC function
        RequestorID:   "qos-optimizer",
        ControlHeader: map[string]interface{}{
            "target_ue_id":   ueID,
            "target_cell_id": "cell-001",
            "control_action": "modify_qos_flow",
        },
        ControlMessage: map[string]interface{}{
            "qos_flow_id": qfi,
            "qos_parameters": map[string]interface{}{
                "priority_level":        newQoS.PriorityLevel,
                "packet_delay_budget":   newQoS.PacketDelayBudget,
                "packet_error_rate":     newQoS.PacketErrorRate,
                "averaging_window":      newQoS.AveragingWindow,
                "maximum_data_burst":    newQoS.MaximumDataBurst,
            },
            "resource_allocation": map[string]interface{}{
                "guaranteed_bit_rate_dl": newQoS.GuaranteedBitRateDL,
                "guaranteed_bit_rate_ul": newQoS.GuaranteedBitRateUL,
                "maximum_bit_rate_dl":    newQoS.MaximumBitRateDL,
                "maximum_bit_rate_ul":    newQoS.MaximumBitRateUL,
            },
        },
        AckRequested: true,
        Timeout:      30 * time.Second,
    }
    
    response, err := client.SendControlRequest(ctx, controlRequest)
    if err != nil {
        return fmt.Errorf("control request failed: %w", err)
    }
    
    if response.Status != "SUCCESS" {
        return fmt.Errorf("control request rejected: %s", response.CauseDescription)
    }
    
    log.Printf("QoS control successful for UE %s, QFI %d", ueID, qfi)
    return nil
}

type QoSParameters struct {
    PriorityLevel        int     `json:"priority_level"`
    PacketDelayBudget    int     `json:"packet_delay_budget"`    // milliseconds
    PacketErrorRate      float64 `json:"packet_error_rate"`
    AveragingWindow      int     `json:"averaging_window"`       // milliseconds
    MaximumDataBurst     int     `json:"maximum_data_burst"`     // bytes
    GuaranteedBitRateDL  int64   `json:"guaranteed_bit_rate_dl"` // bps
    GuaranteedBitRateUL  int64   `json:"guaranteed_bit_rate_ul"` // bps
    MaximumBitRateDL     int64   `json:"maximum_bit_rate_dl"`    // bps
    MaximumBitRateUL     int64   `json:"maximum_bit_rate_ul"`    // bps
}
```

#### Control Acknowledgment Handling

```go
// Control Acknowledgment Handler
type ControlAckHandler struct {
    pendingControls map[string]*PendingControl
    mutex          sync.RWMutex
}

func (h *ControlAckHandler) HandleControlAck(ack *sdk.ControlAck) error {
    h.mutex.Lock()
    defer h.mutex.Unlock()
    
    if pending, exists := h.pendingControls[ack.RequestID]; exists {
        // Update pending control status
        pending.Status = ack.Status
        pending.CompletedAt = time.Now()
        pending.Response = ack.Response
        
        // Notify waiting goroutines
        close(pending.Done)
        
        // Clean up
        delete(h.pendingControls, ack.RequestID)
        
        log.Printf("Control request %s completed with status: %s", ack.RequestID, ack.Status)
    }
    
    return nil
}

type PendingControl struct {
    RequestID   string
    Status      string
    Response    interface{}
    StartedAt   time.Time
    CompletedAt time.Time
    Done        chan struct{}
}
```

### 6. Complete xApp Examples

#### KPM Analytics xApp

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/xapp/sdk"
)

// KMPAnalyticsApp demonstrates a complete KMP analytics xApp
type KMPAnalyticsApp struct {
    client      sdk.XAppClient
    config      *KMPAnalyticsConfig
    storage     *MetricsStorage
    analyzer    *AnomalyDetector
    dashboard   *AnalyticsDashboard
    subscriptions map[string]*Subscription
}

type KMPAnalyticsConfig struct {
    AnalysisWindow      time.Duration `json:"analysis_window"`
    AnomalyThreshold    float64       `json:"anomaly_threshold"`
    AlertingEnabled     bool          `json:"alerting_enabled"`
    MachineLearning     bool          `json:"machine_learning"`
    PredictionEnabled   bool          `json:"prediction_enabled"`
    DashboardPort       int           `json:"dashboard_port"`
}

func NewKMPAnalyticsApp(config *sdk.XAppConfig) (*KMPAnalyticsApp, error) {
    client, err := sdk.NewXAppClient(config)
    if err != nil {
        return nil, err
    }
    
    app := &KMPAnalyticsApp{
        client:        client,
        config:        &KMPAnalyticsConfig{
            AnalysisWindow:   5 * time.Minute,
            AnomalyThreshold: 0.8,
            AlertingEnabled:  true,
            MachineLearning:  true,
            DashboardPort:    3000,
        },
        storage:       NewMetricsStorage(),
        analyzer:      NewAnomalyDetector(),
        dashboard:     NewAnalyticsDashboard(),
        subscriptions: make(map[string]*Subscription),
    }
    
    // Register handlers
    client.RegisterIndicationHandler(app)
    
    return app, nil
}

func (app *KMPAnalyticsApp) Start(ctx context.Context) error {
    // Start the xApp client
    if err := app.client.Start(ctx); err != nil {
        return err
    }
    
    // Start analytics dashboard
    go app.dashboard.Start(app.config.DashboardPort)
    
    // Create default subscriptions
    if err := app.createDefaultSubscriptions(ctx); err != nil {
        return err
    }
    
    // Start background analytics
    go app.runAnalytics(ctx)
    
    log.Println("KMP Analytics xApp started successfully")
    return nil
}

func (app *KMPAnalyticsApp) createDefaultSubscriptions(ctx context.Context) error {
    // Subscribe to key performance metrics
    request := &sdk.SubscriptionRequest{
        NodeID:        "gnb-001",
        RanFunctionID: 1,
        RequestorID:   "kmp-analytics",
        EventTriggers: []sdk.EventTrigger{
            {
                TriggerType:     "PERIODIC",
                ReportingPeriod: 5 * time.Second,
            },
        },
        Actions: []sdk.Action{
            {
                ActionID:   1,
                ActionType: "REPORT",
                ActionDefinition: map[string]interface{}{
                    "measurement_info_list": []map[string]interface{}{
                        {
                            "measurement_type": "DRB.RlcSduDelayDl",
                            "label_info_list": []map[string]interface{}{
                                {"measurement_label": "UE_ID", "measurement_value": "all"},
                            },
                        },
                        {
                            "measurement_type": "DRB.UEThpDl",
                            "label_info_list": []map[string]interface{}{
                                {"measurement_label": "UE_ID", "measurement_value": "all"},
                            },
                        },
                        {
                            "measurement_type": "RRU.PrbUsedDl",
                            "label_info_list": []map[string]interface{}{
                                {"measurement_label": "CELL_ID", "measurement_value": "all"},
                            },
                        },
                    },
                },
            },
        },
    }
    
    subscription, err := app.client.CreateSubscription(ctx, request)
    if err != nil {
        return err
    }
    
    app.subscriptions[subscription.ID] = subscription
    return nil
}

func (app *KMPAnalyticsApp) HandleIndication(indication *sdk.Indication) error {
    // Parse and store measurements
    measurements, err := app.parseKMPMeasurements(indication)
    if err != nil {
        return err
    }
    
    for _, measurement := range measurements {
        // Store in time series database
        if err := app.storage.Store(measurement); err != nil {
            log.Printf("Failed to store measurement: %v", err)
        }
        
        // Real-time anomaly detection
        if app.config.MachineLearning {
            if anomaly := app.analyzer.DetectAnomaly(measurement); anomaly != nil {
                log.Printf("Anomaly detected: %+v", anomaly)
                
                // Update dashboard
                app.dashboard.AddAnomaly(anomaly)
                
                // Trigger alerts if enabled
                if app.config.AlertingEnabled {
                    app.triggerAlert(anomaly)
                }
            }
        }
    }
    
    return nil
}

func (app *KMPAnalyticsApp) runAnalytics(ctx context.Context) {
    ticker := time.NewTicker(app.config.AnalysisWindow)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            app.performPeriodicAnalysis()
        case <-ctx.Done():
            return
        }
    }
}

func (app *KMPAnalyticsApp) performPeriodicAnalysis() {
    log.Println("Performing periodic analytics...")
    
    // Get recent measurements
    endTime := time.Now()
    startTime := endTime.Add(-app.config.AnalysisWindow)
    
    measurements, err := app.storage.GetRange(startTime, endTime)
    if err != nil {
        log.Printf("Failed to get measurements: %v", err)
        return
    }
    
    // Analyze trends
    trends := app.analyzer.AnalyzeTrends(measurements)
    app.dashboard.UpdateTrends(trends)
    
    // Generate predictions if enabled
    if app.config.PredictionEnabled {
        predictions := app.analyzer.GeneratePredictions(measurements, 10*time.Minute)
        app.dashboard.UpdatePredictions(predictions)
    }
    
    // Update analytics dashboard
    stats := app.calculateStatistics(measurements)
    app.dashboard.UpdateStatistics(stats)
    
    log.Printf("Analytics completed. Processed %d measurements", len(measurements))
}

func main() {
    config := &sdk.XAppConfig{
        AppID:           "kmp-analytics-v1",
        AppName:         "KMP Analytics xApp",
        AppVersion:      "1.0.0",
        E2ManagerURL:    "http://nephoran-e2-manager:8080",
        HealthPort:      8090,
        MetricsPort:     9090,
        LogLevel:        "INFO",
        EnableTracing:   true,
    }
    
    app, err := NewKMPAnalyticsApp(config)
    if err != nil {
        log.Fatalf("Failed to create KMP analytics app: %v", err)
    }
    
    ctx := context.Background()
    if err := app.Start(ctx); err != nil {
        log.Fatalf("Failed to start KMP analytics app: %v", err)
    }
    
    // Keep running
    select {}
}
```

#### RAN Control Optimization xApp

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/xapp/sdk"
)

// RCOptimizationApp demonstrates a RAN Control optimization xApp
type RCOptimizationApp struct {
    client          sdk.XAppClient
    optimizer       *RANOptimizer
    policyEngine    *PolicyEngine
    controlHistory  *ControlHistory
    config          *OptimizationConfig
}

type OptimizationConfig struct {
    OptimizationInterval  time.Duration `json:"optimization_interval"`
    LoadBalancingEnabled  bool          `json:"load_balancing_enabled"`
    QoSOptimizationEnabled bool         `json:"qos_optimization_enabled"`
    TrafficSteeringEnabled bool         `json:"traffic_steering_enabled"`
    MaxControlsPerMinute  int           `json:"max_controls_per_minute"`
}

func NewRCOptimizationApp(config *sdk.XAppConfig) (*RCOptimizationApp, error) {
    client, err := sdk.NewXAppClient(config)
    if err != nil {
        return nil, err
    }
    
    app := &RCOptimizationApp{
        client:       client,
        optimizer:    NewRANOptimizer(),
        policyEngine: NewPolicyEngine(),
        controlHistory: NewControlHistory(),
        config: &OptimizationConfig{
            OptimizationInterval:   30 * time.Second,
            LoadBalancingEnabled:   true,
            QoSOptimizationEnabled: true,
            TrafficSteeringEnabled: true,
            MaxControlsPerMinute:   10,
        },
    }
    
    // Register handlers
    client.RegisterIndicationHandler(app)
    client.RegisterControlAckHandler(app)
    
    return app, nil
}

func (app *RCOptimizationApp) Start(ctx context.Context) error {
    if err := app.client.Start(ctx); err != nil {
        return err
    }
    
    // Start optimization loop
    go app.runOptimization(ctx)
    
    log.Println("RAN Control Optimization xApp started")
    return nil
}

func (app *RCOptimizationApp) HandleIndication(indication *sdk.Indication) error {
    // Process measurement data for optimization decisions
    measurements, err := app.parseKMPMeasurements(indication)
    if err != nil {
        return err
    }
    
    // Feed data to optimizer
    for _, measurement := range measurements {
        app.optimizer.ProcessMeasurement(measurement)
    }
    
    return nil
}

func (app *RCOptimizationApp) HandleControlAck(ack *sdk.ControlAck) error {
    // Record control outcome
    app.controlHistory.RecordOutcome(ack.RequestID, ack.Status, ack.Response)
    
    // Update optimizer with control effectiveness
    if outcome, exists := app.controlHistory.GetOutcome(ack.RequestID); exists {
        app.optimizer.UpdateControlEffectiveness(outcome)
    }
    
    return nil
}

func (app *RCOptimizationApp) runOptimization(ctx context.Context) {
    ticker := time.NewTicker(app.config.OptimizationInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            app.performOptimization()
        case <-ctx.Done():
            return
        }
    }
}

func (app *RCOptimizationApp) performOptimization() {
    log.Println("Performing RAN optimization...")
    
    // Get optimization recommendations
    recommendations := app.optimizer.GetRecommendations()
    
    // Apply policy filters
    validRecommendations := app.policyEngine.FilterRecommendations(recommendations)
    
    // Check rate limiting
    if !app.controlHistory.CanSendControl(app.config.MaxControlsPerMinute) {
        log.Printf("Rate limit reached, skipping optimization")
        return
    }
    
    // Execute valid recommendations
    for _, rec := range validRecommendations {
        if err := app.executeRecommendation(rec); err != nil {
            log.Printf("Failed to execute recommendation: %v", err)
        }
    }
    
    log.Printf("Optimization completed. Executed %d recommendations", len(validRecommendations))
}

func (app *RCOptimizationApp) executeRecommendation(rec *OptimizationRecommendation) error {
    ctx := context.Background()
    
    switch rec.Type {
    case "LOAD_BALANCING":
        return app.executeLoadBalancing(ctx, rec)
    case "QOS_OPTIMIZATION":
        return app.executeQoSOptimization(ctx, rec)
    case "TRAFFIC_STEERING":
        return app.executeTrafficSteering(ctx, rec)
    default:
        return fmt.Errorf("unknown recommendation type: %s", rec.Type)
    }
}

func (app *RCOptimizationApp) executeLoadBalancing(ctx context.Context, rec *OptimizationRecommendation) error {
    controlRequest := &sdk.ControlRequest{
        NodeID:        rec.NodeID,
        RanFunctionID: 2, // RC function
        RequestorID:   "ran-optimizer",
        ControlHeader: map[string]interface{}{
            "control_action": "load_balancing",
            "target_cells":   rec.Parameters["target_cells"],
        },
        ControlMessage: map[string]interface{}{
            "balancing_algorithm": rec.Parameters["algorithm"],
            "target_distribution": rec.Parameters["distribution"],
        },
        AckRequested: true,
    }
    
    _, err := app.client.SendControlRequest(ctx, controlRequest)
    if err != nil {
        return fmt.Errorf("load balancing control failed: %w", err)
    }
    
    log.Printf("Load balancing control sent for node %s", rec.NodeID)
    return nil
}

func main() {
    config := &sdk.XAppConfig{
        AppID:        "ran-optimizer-v1",
        AppName:      "RAN Control Optimization xApp",
        AppVersion:   "1.0.0",
        E2ManagerURL: "http://nephoran-e2-manager:8080",
        HealthPort:   8091,
        MetricsPort:  9091,
        LogLevel:     "INFO",
    }
    
    app, err := NewRCOptimizationApp(config)
    if err != nil {
        log.Fatalf("Failed to create RAN optimization app: %v", err)
    }
    
    ctx := context.Background()
    if err := app.Start(ctx); err != nil {
        log.Fatalf("Failed to start RAN optimization app: %v", err)
    }
    
    select {}
}
```

## Python xApp SDK Wrapper

### 1. Python SDK Overview

The Python xApp SDK provides a high-level wrapper around the Go SDK, enabling rapid xApp development in Python with full access to E2 interface capabilities.

#### Installation

```bash
# Install Python dependencies
pip install nephoran-xapp-sdk requests websocket-client

# Or using requirements.txt
echo "nephoran-xapp-sdk>=1.0.0" >> requirements.txt
echo "requests>=2.28.0" >> requirements.txt
echo "websocket-client>=1.4.0" >> requirements.txt
pip install -r requirements.txt
```

#### Basic Python xApp Structure

```python
from nephoran_xapp import XAppClient, XAppConfig
import asyncio
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MyPythonXApp:
    def __init__(self, config: XAppConfig):
        self.client = XAppClient(config)
        self.subscriptions = {}
        
    async def start(self):
        """Start the xApp"""
        await self.client.start()
        
        # Register event handlers
        self.client.on_indication(self.handle_indication)
        self.client.on_control_ack(self.handle_control_ack)
        
        # Create subscriptions
        await self.create_subscriptions()
        
        logger.info("Python xApp started successfully")
    
    async def handle_indication(self, indication):
        """Handle E2 indications"""
        logger.info(f"Received indication: {indication.subscription_id}")
        
        # Process the indication data
        await self.process_indication_data(indication)
    
    async def handle_control_ack(self, ack):
        """Handle control acknowledgments"""
        logger.info(f"Control request {ack.request_id} status: {ack.status}")
    
    async def create_subscriptions(self):
        """Create E2 subscriptions"""
        subscription_request = {
            "node_id": "gnb-001",
            "ran_function_id": 1,
            "requestor_id": "python-xapp",
            "event_triggers": [
                {
                    "trigger_type": "PERIODIC",
                    "reporting_period": 5000,  # 5 seconds
                }
            ],
            "actions": [
                {
                    "action_id": 1,
                    "action_type": "REPORT",
                    "action_definition": {
                        "measurement_info_list": [
                            {
                                "measurement_type": "DRB.RlcSduDelayDl",
                                "label_info_list": [
                                    {
                                        "measurement_label": "UE_ID",
                                        "measurement_value": "all"
                                    }
                                ]
                            }
                        ]
                    }
                }
            ]
        }
        
        subscription = await self.client.create_subscription(subscription_request)
        self.subscriptions[subscription.id] = subscription
        logger.info(f"Created subscription: {subscription.id}")
    
    async def process_indication_data(self, indication):
        """Process indication data - implement your logic here"""
        # Example: Extract measurement data
        if hasattr(indication, 'measurements'):
            for measurement in indication.measurements:
                logger.info(f"Measurement: {measurement.type} = {measurement.value}")
                
                # Perform analysis
                if self.detect_anomaly(measurement):
                    await self.send_control_action(measurement)
    
    def detect_anomaly(self, measurement):
        """Simple anomaly detection example"""
        if measurement.type == "DRB.RlcSduDelayDl":
            return measurement.value > 100  # threshold in ms
        return False
    
    async def send_control_action(self, measurement):
        """Send control action based on measurement"""
        control_request = {
            "node_id": "gnb-001",
            "ran_function_id": 2,  # RC function
            "requestor_id": "python-xapp",
            "control_header": {
                "target_ue_id": measurement.labels.get("UE_ID"),
                "control_action": "modify_qos"
            },
            "control_message": {
                "qos_flow_id": 1,
                "new_priority": 1  # Higher priority
            },
            "ack_requested": True
        }
        
        response = await self.client.send_control_request(control_request)
        logger.info(f"Control request sent: {response.request_id}")

async def main():
    config = XAppConfig(
        app_id="python-xapp-example",
        app_name="Python xApp Example",
        app_version="1.0.0",
        e2_manager_url="http://nephoran-e2-manager:8080",
        health_port=8092,
        metrics_port=9092,
        log_level="INFO"
    )
    
    app = MyPythonXApp(config)
    await app.start()
    
    # Keep running
    try:
        while True:
            await asyncio.sleep(1)
    except KeyboardInterrupt:
        logger.info("Shutting down...")
        await app.client.stop()

if __name__ == "__main__":
    asyncio.run(main())
```

### 2. Python SDK API Reference

#### XAppClient Class

```python
class XAppClient:
    """Python xApp client for E2 interface interaction"""
    
    def __init__(self, config: XAppConfig):
        """Initialize the xApp client"""
        pass
    
    async def start(self) -> None:
        """Start the xApp client"""
        pass
    
    async def stop(self) -> None:
        """Stop the xApp client"""
        pass
    
    async def create_subscription(self, request: dict) -> Subscription:
        """Create an E2 subscription"""
        pass
    
    async def delete_subscription(self, subscription_id: str) -> None:
        """Delete an E2 subscription"""
        pass
    
    async def send_control_request(self, request: dict) -> ControlResponse:
        """Send an E2 control request"""
        pass
    
    def on_indication(self, handler: callable) -> None:
        """Register indication handler"""
        pass
    
    def on_control_ack(self, handler: callable) -> None:
        """Register control acknowledgment handler"""
        pass
    
    def get_subscriptions(self) -> list:
        """Get all active subscriptions"""
        pass
    
    def get_metrics(self) -> dict:
        """Get xApp metrics"""
        pass
    
    def is_healthy(self) -> bool:
        """Check if xApp is healthy"""
        pass
```

#### Configuration Classes

```python
from dataclasses import dataclass
from typing import Optional, List

@dataclass
class XAppConfig:
    """xApp configuration"""
    app_id: str
    app_name: str
    app_version: str
    e2_manager_url: str
    app_description: Optional[str] = None
    api_version: str = "v1"
    timeout: int = 30
    retry_count: int = 3
    health_port: int = 8090
    metrics_port: int = 9090
    api_port: Optional[int] = None
    log_level: str = "INFO"
    enable_tracing: bool = False
    enable_metrics: bool = True
    preferred_models: Optional[List[str]] = None
    buffer_size: int = 1000
    worker_count: int = 4
    heartbeat_interval: int = 30

@dataclass
class Subscription:
    """E2 subscription representation"""
    id: str
    node_id: str
    ran_function_id: int
    requestor_id: str
    status: str
    created_at: str
    actions: List[dict]
    event_triggers: List[dict]

@dataclass
class Indication:
    """E2 indication representation"""
    subscription_id: str
    node_id: str
    ran_function_id: int
    action_id: int
    indication_sn: int
    indication_type: str
    indication_header: dict
    indication_message: dict
    measurements: Optional[List[dict]] = None
    timestamp: Optional[str] = None

@dataclass
class ControlResponse:
    """E2 control response representation"""
    request_id: str
    status: str
    response_data: Optional[dict] = None
    cause_description: Optional[str] = None
    timestamp: Optional[str] = None
```

## xApp Lifecycle Management

### 1. Deployment and Configuration

#### Kubernetes Deployment

```yaml
# xapp-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kmp-analytics-xapp
  namespace: nephoran-system
  labels:
    app: kmp-analytics-xapp
    component: xapp
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kmp-analytics-xapp
  template:
    metadata:
      labels:
        app: kmp-analytics-xapp
        component: xapp
    spec:
      containers:
      - name: xapp
        image: my-registry/kmp-analytics-xapp:v1.0.0
        ports:
        - containerPort: 8090
          name: health
        - containerPort: 9090
          name: metrics
        - containerPort: 3000
          name: dashboard
        env:
        - name: E2_MANAGER_URL
          value: "http://nephoran-e2-manager:8080"
        - name: APP_ID
          value: "kmp-analytics-v1"
        - name: LOG_LEVEL
          value: "INFO"
        - name: ENABLE_TRACING
          value: "true"
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8090
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8090
          initialDelaySeconds: 5
          periodSeconds: 5
        volumeMounts:
        - name: config
          mountPath: /etc/xapp
        - name: data
          mountPath: /var/lib/xapp
      volumes:
      - name: config
        configMap:
          name: kmp-analytics-config
      - name: data
        persistentVolumeClaim:
          claimName: xapp-data-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: kmp-analytics-xapp
  namespace: nephoran-system
spec:
  selector:
    app: kmp-analytics-xapp
  ports:
  - name: health
    port: 8090
    targetPort: 8090
  - name: metrics
    port: 9090
    targetPort: 9090
  - name: dashboard
    port: 3000
    targetPort: 3000
```

#### Configuration Management

```yaml
# xapp-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kmp-analytics-config
  namespace: nephoran-system
data:
  app.yaml: |
    app:
      id: "kmp-analytics-v1"
      name: "KMP Analytics xApp"
      version: "1.0.0"
      description: "Advanced KMP analytics with ML-based anomaly detection"
    
    e2:
      manager_url: "http://nephoran-e2-manager:8080"
      api_version: "v1"
      timeout: 30s
      retry_count: 3
    
    analytics:
      analysis_window: "5m"
      anomaly_threshold: 0.8
      alerting_enabled: true
      machine_learning: true
      prediction_enabled: true
    
    dashboard:
      enabled: true
      port: 3000
      update_interval: "10s"
    
    storage:
      type: "timeseries"
      retention: "24h"
      compression: true
    
    logging:
      level: "INFO"
      format: "json"
      tracing: true
```

### 2. Health Monitoring and Observability

#### Health Check Implementation

```go
// Health check endpoint implementation
func (app *KMPAnalyticsApp) SetupHealthChecks() {
    http.HandleFunc("/health", app.healthHandler)
    http.HandleFunc("/ready", app.readinessHandler)
    http.HandleFunc("/metrics", app.metricsHandler)
    
    log.Printf("Health checks available on port %d", app.config.HealthPort)
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", app.config.HealthPort), nil))
}

func (app *KMPAnalyticsApp) healthHandler(w http.ResponseWriter, r *http.Request) {
    status := map[string]interface{}{
        "status": "healthy",
        "timestamp": time.Now().Format(time.RFC3339),
        "app_id": app.client.GetConfig().AppID,
        "version": app.client.GetConfig().AppVersion,
        "uptime": time.Since(app.startTime).String(),
    }
    
    // Check component health
    checks := map[string]bool{
        "e2_manager_connection": app.client.IsHealthy(),
        "storage_connection":    app.storage.IsHealthy(),
        "analyzer_status":       app.analyzer.IsHealthy(),
        "dashboard_status":      app.dashboard.IsHealthy(),
    }
    
    allHealthy := true
    for component, healthy := range checks {
        status[component] = healthy
        if !healthy {
            allHealthy = false
        }
    }
    
    if allHealthy {
        status["status"] = "healthy"
        w.WriteHeader(http.StatusOK)
    } else {
        status["status"] = "unhealthy"
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(status)
}

func (app *KMPAnalyticsApp) readinessHandler(w http.ResponseWriter, r *http.Request) {
    ready := app.client.IsHealthy() && 
             len(app.subscriptions) > 0 &&
             app.storage.IsReady()
    
    status := map[string]interface{}{
        "ready": ready,
        "timestamp": time.Now().Format(time.RFC3339),
        "subscriptions": len(app.subscriptions),
    }
    
    if ready {
        w.WriteHeader(http.StatusOK)
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(status)
}
```

#### Metrics Collection

```go
// Prometheus metrics for xApp
var (
    indicationsReceived = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "xapp_indications_received_total",
            Help: "Total number of E2 indications received",
        },
        []string{"node_id", "subscription_id", "measurement_type"},
    )
    
    controlsSent = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "xapp_controls_sent_total",
            Help: "Total number of control messages sent",
        },
        []string{"node_id", "control_type", "status"},
    )
    
    anomaliesDetected = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "xapp_anomalies_detected_total",
            Help: "Total number of anomalies detected",
        },
        []string{"measurement_type", "severity"},
    )
    
    processingLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "xapp_processing_duration_seconds",
            Help: "Time spent processing indications",
            Buckets: prometheus.DefBuckets,
        },
        []string{"processing_type"},
    )
)

func (app *KMPAnalyticsApp) initMetrics() {
    prometheus.MustRegister(indicationsReceived)
    prometheus.MustRegister(controlsSent)
    prometheus.MustRegister(anomaliesDetected)
    prometheus.MustRegister(processingLatency)
}

func (app *KMPAnalyticsApp) metricsHandler(w http.ResponseWriter, r *http.Request) {
    promhttp.Handler().ServeHTTP(w, r)
}
```

### 3. Testing and Validation

#### Unit Testing

```go
// xapp_test.go
package main

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/xapp/sdk"
)

// Mock client for testing
type MockXAppClient struct {
    mock.Mock
}

func (m *MockXAppClient) Start(ctx context.Context) error {
    args := m.Called(ctx)
    return args.Error(0)
}

func (m *MockXAppClient) CreateSubscription(ctx context.Context, req *sdk.SubscriptionRequest) (*sdk.Subscription, error) {
    args := m.Called(ctx, req)
    return args.Get(0).(*sdk.Subscription), args.Error(1)
}

func (m *MockXAppClient) SendControlRequest(ctx context.Context, req *sdk.ControlRequest) (*sdk.ControlResponse, error) {
    args := m.Called(ctx, req)
    return args.Get(0).(*sdk.ControlResponse), args.Error(1)
}

func TestKMPAnalyticsApp_Start(t *testing.T) {
    // Setup mock client
    mockClient := new(MockXAppClient)
    mockClient.On("Start", mock.Anything).Return(nil)
    
    // Create app with mock client
    app := &KMPAnalyticsApp{
        client: mockClient,
        config: &KMPAnalyticsConfig{
            AnalysisWindow:   5 * time.Minute,
            AnomalyThreshold: 0.8,
        },
        subscriptions: make(map[string]*sdk.Subscription),
    }
    
    // Test start
    ctx := context.Background()
    err := app.Start(ctx)
    
    assert.NoError(t, err)
    mockClient.AssertExpectations(t)
}

func TestKMPAnalyticsApp_HandleIndication(t *testing.T) {
    app := &KMPAnalyticsApp{
        storage:  NewMockMetricsStorage(),
        analyzer: NewMockAnomalyDetector(),
    }
    
    indication := &sdk.Indication{
        SubscriptionID: "test-sub-001",
        NodeID:        "gnb-001",
        RanFunctionID: 1,
        Measurements: []sdk.Measurement{
            {
                Type:  "DRB.RlcSduDelayDl",
                Value: 150.0, // Above threshold
                Labels: map[string]interface{}{
                    "UE_ID": "ue-001",
                },
                Timestamp: time.Now(),
            },
        },
    }
    
    err := app.HandleIndication(indication)
    assert.NoError(t, err)
}

// Benchmark tests
func BenchmarkKMPAnalyticsApp_ProcessIndication(b *testing.B) {
    app := &KMPAnalyticsApp{
        storage:  NewMockMetricsStorage(),
        analyzer: NewMockAnomalyDetector(),
    }
    
    indication := &sdk.Indication{
        SubscriptionID: "benchmark-sub",
        NodeID:        "gnb-001",
        RanFunctionID: 1,
        Measurements:  generateBenchmarkMeasurements(100),
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        app.HandleIndication(indication)
    }
}
```

#### Integration Testing

```go
// integration_test.go
package main

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/suite"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/xapp/sdk"
)

type XAppIntegrationTestSuite struct {
    suite.Suite
    client  sdk.XAppClient
    app     *KMPAnalyticsApp
}

func (suite *XAppIntegrationTestSuite) SetupSuite() {
    config := &sdk.XAppConfig{
        AppID:        "integration-test-app",
        AppName:      "Integration Test xApp",
        AppVersion:   "1.0.0-test",
        E2ManagerURL: "http://localhost:8080", // Test E2 manager
        HealthPort:   8093,
        MetricsPort:  9093,
        LogLevel:     "DEBUG",
    }
    
    client, err := sdk.NewXAppClient(config)
    suite.Require().NoError(err)
    
    suite.client = client
    suite.app = &KMPAnalyticsApp{
        client:        client,
        subscriptions: make(map[string]*sdk.Subscription),
    }
}

func (suite *XAppIntegrationTestSuite) TearDownSuite() {
    if suite.client != nil {
        suite.client.Stop(context.Background())
    }
}

func (suite *XAppIntegrationTestSuite) TestFullWorkflow() {
    ctx := context.Background()
    
    // Test 1: Start xApp
    err := suite.app.Start(ctx)
    suite.Require().NoError(err)
    
    // Test 2: Create subscription
    subRequest := &sdk.SubscriptionRequest{
        NodeID:        "test-gnb-001",
        RanFunctionID: 1,
        RequestorID:   "integration-test",
        EventTriggers: []sdk.EventTrigger{
            {
                TriggerType:     "PERIODIC",
                ReportingPeriod: 1 * time.Second,
            },
        },
        Actions: []sdk.Action{
            {
                ActionID:   1,
                ActionType: "REPORT",
            },
        },
    }
    
    subscription, err := suite.client.CreateSubscription(ctx, subRequest)
    suite.Require().NoError(err)
    suite.Assert().NotEmpty(subscription.ID)
    
    // Test 3: Wait for indications
    time.Sleep(5 * time.Second)
    
    // Test 4: Send control message
    controlRequest := &sdk.ControlRequest{
        NodeID:        "test-gnb-001",
        RanFunctionID: 2,
        RequestorID:   "integration-test",
        ControlHeader: map[string]interface{}{
            "test": "control",
        },
        ControlMessage: map[string]interface{}{
            "action": "test",
        },
        AckRequested: true,
    }
    
    response, err := suite.client.SendControlRequest(ctx, controlRequest)
    suite.Require().NoError(err)
    suite.Assert().NotEmpty(response.RequestID)
    
    // Test 5: Cleanup
    err = suite.client.DeleteSubscription(ctx, subscription.ID)
    suite.Assert().NoError(err)
}

func TestXAppIntegrationSuite(t *testing.T) {
    suite.Run(t, new(XAppIntegrationTestSuite))
}
```

This comprehensive xApp Development SDK Documentation provides complete guidance for developing both Go and Python xApps, including practical examples, lifecycle management, and testing procedures. The documentation covers all aspects needed to build production-ready xApps that integrate with the Nephoran Intent Operator's E2 interface system.