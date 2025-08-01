# xApp Development SDK Guide

## Overview

The Nephoran xApp SDK provides a comprehensive framework for developing E2 applications (xApps) that interact with the Near-RT RIC and E2 Nodes. This guide covers the complete development lifecycle from initial setup to production deployment, including practical examples and best practices.

## Table of Contents

1. [Getting Started](#getting-started)
2. [SDK Architecture](#sdk-architecture)
3. [Basic xApp Development](#basic-xapp-development)
4. [Advanced Features](#advanced-features)
5. [Tutorial: Building a KPM xApp](#tutorial-building-a-kpm-xapp)
6. [Tutorial: Building an RC xApp](#tutorial-building-an-rc-xapp)
7. [Best Practices](#best-practices)
8. [Troubleshooting](#troubleshooting)

## Getting Started

### Prerequisites

- Go 1.19 or higher
- Access to Near-RT RIC deployment
- Basic understanding of E2 interface and O-RAN architecture

### Installation

```bash
# Install the Nephoran xApp SDK
go get github.com/thc1006/nephoran-intent-operator/pkg/oran/e2

# Install SDK CLI tools (optional)
go install github.com/thc1006/nephoran-intent-operator/cmd/xapp-cli@latest
```

### Project Setup

Create a new xApp project:

```bash
# Create project directory
mkdir my-xapp && cd my-xapp

# Initialize Go module
go mod init github.com/myorg/my-xapp

# Create basic structure
mkdir -p cmd/xapp config pkg/handlers
```

## SDK Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                               xApp SDK Architecture                                 │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                            Application Layer                                 │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │   │
│  │  │   Your xApp     │  │   Business      │  │    Custom Handlers          │ │   │
│  │  │  Application    │  │     Logic       │  │  & Processors               │ │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                          │                                         │
│                                          ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                               SDK Core                                       │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │   │
│  │  │    XAppSDK      │  │   Lifecycle     │  │      Metrics                │ │   │
│  │  │   Framework     │  │   Management    │  │    Collection               │ │   │
│  │  │                 │  │                 │  │                             │ │   │
│  │  │ • Init/Start    │  │ • State Mgmt    │  │ • Performance               │ │   │
│  │  │ • Subscription  │  │ • Events        │  │ • Custom Metrics            │ │   │
│  │  │ • Control       │  │ • Health        │  │ • Reporting                 │ │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                          │                                         │
│                                          ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                           E2 Interface Layer                                 │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │   │
│  │  │   E2Manager     │  │   E2Adaptor     │  │   Service Models            │ │   │
│  │  │                 │  │                 │  │                             │ │   │
│  │  │ • Connection    │  │ • Messages      │  │ • KPM                       │ │   │
│  │  │ • Registry      │  │ • Encoding      │  │ • RC                        │ │   │
│  │  │ • Health        │  │ • Transport     │  │ • Custom                    │ │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

## Basic xApp Development

### 1. Creating a Simple xApp

```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
)

func main() {
    // Create xApp configuration
    config := &e2.XAppConfig{
        XAppName:        "my-first-xapp",
        XAppVersion:     "1.0.0",
        XAppDescription: "My First xApp using Nephoran SDK",
        E2NodeID:        "gnb-001",
        NearRTRICURL:    "http://near-rt-ric:8080",
        ServiceModels:   []string{"KPM", "RC"},
        ResourceLimits: &e2.XAppResourceLimits{
            MaxMemoryMB:      1024,
            MaxCPUCores:      2.0,
            MaxSubscriptions: 10,
            RequestTimeout:   30 * time.Second,
        },
        HealthCheck: &e2.XAppHealthConfig{
            Enabled:          true,
            CheckInterval:    30 * time.Second,
            FailureThreshold: 3,
            HealthEndpoint:   "/health",
        },
    }
    
    // Create E2Manager
    e2Manager := e2.NewE2Manager(config.NearRTRICURL)
    
    // Create xApp SDK instance
    sdk, err := e2.NewXAppSDK(config, e2Manager)
    if err != nil {
        log.Fatalf("Failed to create xApp SDK: %v", err)
    }
    
    // Register indication handler
    sdk.RegisterIndicationHandler("default", handleIndication)
    
    // Start xApp
    ctx := context.Background()
    if err := sdk.Start(ctx); err != nil {
        log.Fatalf("Failed to start xApp: %v", err)
    }
    
    log.Printf("xApp %s started successfully", config.XAppName)
    
    // Keep running
    select {}
}

func handleIndication(ctx context.Context, indication *e2.RICIndication) error {
    log.Printf("Received indication from RAN Function %d", indication.RANFunctionID)
    // Process indication
    return nil
}
```

### 2. Creating Subscriptions

```go
func createKPMSubscription(sdk *e2.XAppSDK) error {
    // Create subscription request
    subscriptionReq := &e2.E2SubscriptionRequest{
        SubscriptionID: "kpm-sub-001",
        RequestorID:    "my-xapp",
        NodeID:         "gnb-001",
        RanFunctionID:  1, // KPM function ID
        EventTriggers: []e2.E2EventTrigger{
            {
                TriggerType:     "PERIODIC",
                ReportingPeriod: 1 * time.Second,
            },
        },
        Actions: []e2.E2Action{
            {
                ActionID:   1,
                ActionType: "REPORT",
                ActionDefinition: map[string]interface{}{
                    "measurements": []string{
                        "RRC.ConnEstabAtt",
                        "RRC.ConnEstabSucc",
                        "DRB.PdcpSduVolumeDL",
                        "DRB.PdcpSduVolumeUL",
                    },
                    "granularity": 1000, // 1 second
                    "cell_id":     "cell-001",
                },
            },
        },
        ReportingPeriod: 1 * time.Second,
    }
    
    // Create subscription
    subscription, err := sdk.Subscribe(context.Background(), subscriptionReq)
    if err != nil {
        return fmt.Errorf("failed to create subscription: %w", err)
    }
    
    log.Printf("Created subscription: %s with status: %s", 
        subscription.SubscriptionID, subscription.Status)
    
    return nil
}
```

### 3. Sending Control Messages

```go
func sendTrafficSteeringControl(sdk *e2.XAppSDK) error {
    // Create control request
    controlReq := &e2.RICControlRequest{
        RICRequestID: e2.RICRequestID{
            RICRequestorID: 123,
            RICInstanceID:  1,
        },
        RANFunctionID: 2, // RC function ID
        RICControlHeader: []byte{
            // Control header with UE ID and control type
        },
        RICControlMessage: []byte{
            // Control message with parameters
        },
        RICControlAckRequest: e2.RICControlAckRequestEnum(1), // Request acknowledgment
    }
    
    // Send control message
    ack, err := sdk.SendControlMessage(context.Background(), controlReq)
    if err != nil {
        return fmt.Errorf("failed to send control message: %w", err)
    }
    
    log.Printf("Control message acknowledged: %+v", ack)
    return nil
}
```

## Advanced Features

### 1. Lifecycle Management

```go
// Register lifecycle handlers
sdk.lifecycle.RegisterHandler(e2.XAppEventStartup, func(ctx context.Context, event e2.XAppLifecycleEvent, data interface{}) error {
    log.Println("xApp startup event triggered")
    // Perform initialization
    return nil
})

sdk.lifecycle.RegisterHandler(e2.XAppEventShutdown, func(ctx context.Context, event e2.XAppLifecycleEvent, data interface{}) error {
    log.Println("xApp shutdown event triggered")
    // Perform cleanup
    return nil
})

sdk.lifecycle.RegisterHandler(e2.XAppEventError, func(ctx context.Context, event e2.XAppLifecycleEvent, data interface{}) error {
    log.Printf("xApp error event: %v", data)
    // Handle error
    return nil
})
```

### 2. Custom Metrics

```go
// Set custom metrics
sdk.GetMetrics().SetCustomMetric("processed_messages", 1000)
sdk.GetMetrics().SetCustomMetric("average_latency_ms", 25.5)

// Get metrics snapshot
metrics := sdk.GetMetrics()
log.Printf("Active subscriptions: %d", metrics.SubscriptionsActive)
log.Printf("Indications received: %d", metrics.IndicationsReceived)
log.Printf("Throughput: %.2f/sec", metrics.ThroughputPerSecond)
```

### 3. Health Monitoring

```go
// Implement custom health check
type HealthChecker struct {
    sdk *e2.XAppSDK
}

func (h *HealthChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    state := h.sdk.GetState()
    metrics := h.sdk.GetMetrics()
    
    health := map[string]interface{}{
        "status": string(state),
        "metrics": map[string]interface{}{
            "subscriptions": metrics.SubscriptionsActive,
            "indications":   metrics.IndicationsReceived,
            "errors":        metrics.ErrorCount,
        },
        "timestamp": time.Now().UTC(),
    }
    
    if state != e2.XAppStateRunning {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    json.NewEncoder(w).Encode(health)
}
```

## Tutorial: Building a KPM xApp

This tutorial walks through building a complete Key Performance Measurement (KPM) xApp.

### Step 1: Project Structure

```
kpm-xapp/
├── cmd/
│   └── kpm-xapp/
│       └── main.go
├── pkg/
│   ├── handlers/
│   │   └── kpm_handler.go
│   ├── processor/
│   │   └── metrics_processor.go
│   └── storage/
│       └── metrics_store.go
├── config/
│   └── config.yaml
├── go.mod
└── Dockerfile
```

### Step 2: Main Application

```go
// cmd/kpm-xapp/main.go
package main

import (
    "context"
    "flag"
    "log"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
    "github.com/myorg/kpm-xapp/pkg/handlers"
    "github.com/myorg/kpm-xapp/pkg/processor"
    "github.com/myorg/kpm-xapp/pkg/storage"
)

func main() {
    var configPath string
    flag.StringVar(&configPath, "config", "config/config.yaml", "Path to configuration file")
    flag.Parse()
    
    // Load configuration
    config := loadConfig(configPath)
    
    // Create components
    metricsStore := storage.NewMetricsStore()
    metricsProcessor := processor.NewMetricsProcessor(metricsStore)
    kpmHandler := handlers.NewKPMHandler(metricsProcessor)
    
    // Create E2Manager
    e2Manager := e2.NewE2Manager(config.NearRTRICURL)
    
    // Create xApp SDK
    sdk, err := e2.NewXAppSDK(config, e2Manager)
    if err != nil {
        log.Fatalf("Failed to create xApp SDK: %v", err)
    }
    
    // Register handlers
    sdk.RegisterIndicationHandler("kpm", kpmHandler.HandleKPMIndication)
    
    // Start xApp
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    if err := sdk.Start(ctx); err != nil {
        log.Fatalf("Failed to start xApp: %v", err)
    }
    
    // Create KPM subscriptions
    if err := createKPMSubscriptions(sdk, config); err != nil {
        log.Fatalf("Failed to create subscriptions: %v", err)
    }
    
    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan
    
    // Graceful shutdown
    log.Println("Shutting down xApp...")
    if err := sdk.Stop(ctx); err != nil {
        log.Printf("Error during shutdown: %v", err)
    }
}

func createKPMSubscriptions(sdk *e2.XAppSDK, config *e2.XAppConfig) error {
    // Create subscription for each configured cell
    cells := []string{"cell-001", "cell-002", "cell-003"}
    
    for _, cellID := range cells {
        subscriptionReq := &e2.E2SubscriptionRequest{
            SubscriptionID: fmt.Sprintf("kpm-%s", cellID),
            RequestorID:    config.XAppName,
            NodeID:         config.E2NodeID,
            RanFunctionID:  1, // KPM function ID
            EventTriggers: []e2.E2EventTrigger{
                {
                    TriggerType:     "PERIODIC",
                    ReportingPeriod: 5 * time.Second,
                },
            },
            Actions: []e2.E2Action{
                {
                    ActionID:   1,
                    ActionType: "REPORT",
                    ActionDefinition: map[string]interface{}{
                        "measurements": []string{
                            "RRC.ConnEstabAtt",
                            "RRC.ConnEstabSucc",
                            "RRC.ConnMean",
                            "DRB.PdcpSduVolumeDL",
                            "DRB.PdcpSduVolumeUL",
                            "RRU.PrbUsedDl",
                            "RRU.PrbUsedUl",
                        },
                        "granularity": 5000, // 5 seconds
                        "cell_id":     cellID,
                    },
                },
            },
            ReportingPeriod: 5 * time.Second,
        }
        
        subscription, err := sdk.Subscribe(context.Background(), subscriptionReq)
        if err != nil {
            return fmt.Errorf("failed to create subscription for cell %s: %w", cellID, err)
        }
        
        log.Printf("Created KPM subscription for cell %s: %s", cellID, subscription.SubscriptionID)
    }
    
    return nil
}
```

### Step 3: KPM Handler Implementation

```go
// pkg/handlers/kpm_handler.go
package handlers

import (
    "context"
    "encoding/json"
    "log"
    "time"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/e2/service_models"
    "github.com/myorg/kpm-xapp/pkg/processor"
)

type KPMHandler struct {
    processor *processor.MetricsProcessor
    kpmModel  *service_models.KPMServiceModel
}

func NewKPMHandler(processor *processor.MetricsProcessor) *KPMHandler {
    return &KPMHandler{
        processor: processor,
        kpmModel:  service_models.NewKPMServiceModel(),
    }
}

func (h *KPMHandler) HandleKPMIndication(ctx context.Context, indication *e2.RICIndication) error {
    // Parse indication using KPM service model
    report, err := h.kpmModel.ParseIndication(
        indication.RICIndicationHeader,
        indication.RICIndicationMessage,
    )
    if err != nil {
        return fmt.Errorf("failed to parse KPM indication: %w", err)
    }
    
    kpmReport, ok := report.(*service_models.KPMReport)
    if !ok {
        return fmt.Errorf("unexpected report type")
    }
    
    // Log received metrics
    log.Printf("Received KPM report for cell %s at %v", 
        kpmReport.CellID, kpmReport.Timestamp)
    log.Printf("  UE Count: %d", kpmReport.UECount)
    log.Printf("  Measurements: %+v", kpmReport.Measurements)
    
    // Process metrics
    if err := h.processor.ProcessMetrics(kpmReport); err != nil {
        log.Printf("Error processing metrics: %v", err)
        return err
    }
    
    return nil
}
```

### Step 4: Metrics Processing

```go
// pkg/processor/metrics_processor.go
package processor

import (
    "fmt"
    "time"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/e2/service_models"
    "github.com/myorg/kpm-xapp/pkg/storage"
)

type MetricsProcessor struct {
    store *storage.MetricsStore
    aggregator *MetricsAggregator
}

func NewMetricsProcessor(store *storage.MetricsStore) *MetricsProcessor {
    return &MetricsProcessor{
        store: store,
        aggregator: NewMetricsAggregator(),
    }
}

func (p *MetricsProcessor) ProcessMetrics(report *service_models.KPMReport) error {
    // Calculate derived metrics
    if connAtt, ok := report.Measurements["RRC.ConnEstabAtt"]; ok {
        if connSucc, ok := report.Measurements["RRC.ConnEstabSucc"]; ok {
            successRate := (connSucc / connAtt) * 100
            report.Measurements["RRC.ConnEstabSuccRate"] = successRate
        }
    }
    
    // Calculate PRB utilization
    if prbUsedDl, ok := report.Measurements["RRU.PrbUsedDl"]; ok {
        if prbTotDl, ok := report.Measurements["RRU.PrbTotDl"]; ok {
            prbUtilDl := (prbUsedDl / prbTotDl) * 100
            report.Measurements["RRU.PrbUtilizationDl"] = prbUtilDl
        }
    }
    
    // Store metrics
    metric := &storage.Metric{
        Timestamp:    report.Timestamp,
        CellID:       report.CellID,
        UECount:      report.UECount,
        Measurements: report.Measurements,
    }
    
    if err := p.store.Store(metric); err != nil {
        return fmt.Errorf("failed to store metrics: %w", err)
    }
    
    // Aggregate metrics
    p.aggregator.AddMetric(metric)
    
    // Check thresholds and trigger alerts if needed
    if err := p.checkThresholds(report); err != nil {
        return fmt.Errorf("threshold check failed: %w", err)
    }
    
    return nil
}

func (p *MetricsProcessor) checkThresholds(report *service_models.KPMReport) error {
    // Example: Alert if PRB utilization exceeds 80%
    if prbUtil, ok := report.Measurements["RRU.PrbUtilizationDl"]; ok {
        if prbUtil > 80.0 {
            // Trigger high utilization alert
            alert := &Alert{
                Type:      "HIGH_PRB_UTILIZATION",
                Severity:  "WARNING",
                CellID:    report.CellID,
                Value:     prbUtil,
                Threshold: 80.0,
                Timestamp: time.Now(),
            }
            
            // Send alert (implementation depends on alerting system)
            if err := p.sendAlert(alert); err != nil {
                return err
            }
        }
    }
    
    return nil
}
```

### Step 5: Configuration

```yaml
# config/config.yaml
xapp:
  name: "kpm-monitor-xapp"
  version: "1.0.0"
  description: "KPM Monitoring xApp"
  
e2:
  node_id: "gnb-001"
  ric_url: "http://near-rt-ric:8080"
  service_models:
    - "KPM"
  
resources:
  max_memory_mb: 2048
  max_cpu_cores: 2.0
  max_subscriptions: 20
  request_timeout: "30s"
  
health:
  enabled: true
  check_interval: "30s"
  failure_threshold: 3
  endpoint: "/health"
  
monitoring:
  metrics_retention: "7d"
  aggregation_interval: "1m"
  
alerts:
  prb_utilization_threshold: 80.0
  connection_failure_threshold: 10.0
  
storage:
  type: "memory"  # or "redis", "influxdb"
  retention: "24h"
```

### Step 6: Dockerfile

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o kpm-xapp cmd/kpm-xapp/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/kpm-xapp .
COPY --from=builder /app/config ./config

EXPOSE 8080

CMD ["./kpm-xapp"]
```

## Tutorial: Building an RC xApp

This tutorial demonstrates building a RAN Control (RC) xApp for traffic steering.

### Step 1: RC xApp Structure

```go
// cmd/rc-xapp/main.go
package main

import (
    "context"
    "log"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/e2/service_models"
    "github.com/myorg/rc-xapp/pkg/controller"
)

func main() {
    config := &e2.XAppConfig{
        XAppName:        "traffic-steering-xapp",
        XAppVersion:     "1.0.0",
        XAppDescription: "Traffic Steering RC xApp",
        E2NodeID:        "gnb-001",
        NearRTRICURL:    "http://near-rt-ric:8080",
        ServiceModels:   []string{"RC"},
    }
    
    // Create components
    e2Manager := e2.NewE2Manager(config.NearRTRICURL)
    sdk, err := e2.NewXAppSDK(config, e2Manager)
    if err != nil {
        log.Fatalf("Failed to create xApp SDK: %v", err)
    }
    
    // Create traffic controller
    rcModel := service_models.NewRCServiceModel()
    trafficController := controller.NewTrafficController(sdk, rcModel)
    
    // Register indication handler for load reports
    sdk.RegisterIndicationHandler("load_report", trafficController.HandleLoadReport)
    
    // Start xApp
    ctx := context.Background()
    if err := sdk.Start(ctx); err != nil {
        log.Fatalf("Failed to start xApp: %v", err)
    }
    
    // Start traffic steering logic
    go trafficController.Run(ctx)
    
    // Keep running
    select {}
}
```

### Step 2: Traffic Controller

```go
// pkg/controller/traffic_controller.go
package controller

import (
    "context"
    "log"
    "time"
    
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/e2/service_models"
)

type TrafficController struct {
    sdk      *e2.XAppSDK
    rcModel  *service_models.RCServiceModel
    loadData map[string]*CellLoadInfo
}

type CellLoadInfo struct {
    CellID        string
    PRBUtilization float64
    ActiveUEs     int
    LastUpdate    time.Time
}

func NewTrafficController(sdk *e2.XAppSDK, rcModel *service_models.RCServiceModel) *TrafficController {
    return &TrafficController{
        sdk:      sdk,
        rcModel:  rcModel,
        loadData: make(map[string]*CellLoadInfo),
    }
}

func (tc *TrafficController) HandleLoadReport(ctx context.Context, indication *e2.RICIndication) error {
    // Parse load information from indication
    // (Implementation depends on how load is reported)
    
    // Update load data
    tc.updateLoadData(cellID, prbUtil, ueCount)
    
    return nil
}

func (tc *TrafficController) Run(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            tc.evaluateAndSteer()
        }
    }
}

func (tc *TrafficController) evaluateAndSteer() {
    // Find overloaded and underloaded cells
    overloadedCells := tc.findOverloadedCells(80.0) // 80% threshold
    underloadedCells := tc.findUnderloadedCells(30.0) // 30% threshold
    
    if len(overloadedCells) == 0 || len(underloadedCells) == 0 {
        return // No steering needed
    }
    
    // Perform traffic steering
    for _, overloaded := range overloadedCells {
        target := tc.findBestTarget(overloaded, underloadedCells)
        if target == nil {
            continue
        }
        
        // Get UEs to steer
        uesToSteer := tc.selectUEsForSteering(overloaded, target, 5) // Steer up to 5 UEs
        
        // Send control messages
        for _, ueID := range uesToSteer {
            if err := tc.steerUE(ueID, overloaded.CellID, target.CellID); err != nil {
                log.Printf("Failed to steer UE %s: %v", ueID, err)
            }
        }
    }
}

func (tc *TrafficController) steerUE(ueID, sourceCellID, targetCellID string) error {
    // Create traffic steering control request
    controlReq, err := tc.rcModel.CreateTrafficSteeringControl(ueID, targetCellID)
    if err != nil {
        return fmt.Errorf("failed to create control request: %w", err)
    }
    
    // Send control message
    ack, err := tc.sdk.SendControlMessage(context.Background(), controlReq)
    if err != nil {
        return fmt.Errorf("failed to send control message: %w", err)
    }
    
    log.Printf("Successfully steered UE %s from cell %s to cell %s", 
        ueID, sourceCellID, targetCellID)
    
    return nil
}
```

## Best Practices

### 1. Error Handling

```go
// Use structured error types
type XAppError struct {
    Code    string
    Message string
    Details map[string]interface{}
}

func (e *XAppError) Error() string {
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Wrap errors with context
if err := sdk.Subscribe(ctx, req); err != nil {
    return &XAppError{
        Code:    "SUBSCRIPTION_FAILED",
        Message: "Failed to create subscription",
        Details: map[string]interface{}{
            "subscription_id": req.SubscriptionID,
            "error":          err.Error(),
        },
    }
}
```

### 2. Resource Management

```go
// Always use contexts for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Clean up resources
defer func() {
    if err := sdk.Stop(ctx); err != nil {
        log.Printf("Error during cleanup: %v", err)
    }
}()

// Monitor resource usage
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        metrics := sdk.GetMetrics()
        if metrics.SubscriptionsActive > config.ResourceLimits.MaxSubscriptions * 0.8 {
            log.Warn("Approaching subscription limit")
        }
    }
}()
```

### 3. Testing

```go
// Unit test example
func TestKPMHandler_HandleIndication(t *testing.T) {
    // Create mock components
    mockProcessor := &MockMetricsProcessor{}
    handler := NewKPMHandler(mockProcessor)
    
    // Create test indication
    indication := &e2.RICIndication{
        RICRequestID: e2.RICRequestID{
            RICRequestorID: 123,
            RICInstanceID:  1,
        },
        RANFunctionID: 1,
        RICIndicationHeader: []byte{/* test header */},
        RICIndicationMessage: []byte{/* test message */},
    }
    
    // Test handling
    err := handler.HandleKPMIndication(context.Background(), indication)
    assert.NoError(t, err)
    assert.Equal(t, 1, mockProcessor.ProcessCount)
}
```

### 4. Configuration Management

```go
// Use environment variables for sensitive data
type Config struct {
    RICEndpoint string `env:"RIC_ENDPOINT" envDefault:"http://localhost:8080"`
    LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
    MaxRetries  int    `env:"MAX_RETRIES" envDefault:"3"`
}

// Validate configuration
func (c *Config) Validate() error {
    if c.RICEndpoint == "" {
        return errors.New("RIC endpoint is required")
    }
    if c.MaxRetries < 0 {
        return errors.New("max retries must be non-negative")
    }
    return nil
}
```

## Troubleshooting

### Common Issues

1. **Connection Failed**
   - Check Near-RT RIC URL and availability
   - Verify network connectivity
   - Check firewall rules

2. **Subscription Failed**
   - Verify RAN Function ID is correct
   - Check service model support
   - Validate subscription parameters

3. **No Indications Received**
   - Verify subscription is active
   - Check E2 Node connectivity
   - Review event trigger configuration

4. **High Memory Usage**
   - Monitor metrics storage
   - Implement data retention policies
   - Use external storage for large datasets

### Debug Logging

```go
// Enable debug logging
import "github.com/sirupsen/logrus"

func init() {
    // Set log level based on environment
    level, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL"))
    if err != nil {
        level = logrus.InfoLevel
    }
    logrus.SetLevel(level)
    
    // Add context to logs
    logrus.SetFormatter(&logrus.JSONFormatter{
        FieldMap: logrus.FieldMap{
            logrus.FieldKeyTime:  "timestamp",
            logrus.FieldKeyLevel: "level",
            logrus.FieldKeyMsg:   "message",
        },
    })
}

// Use structured logging
logrus.WithFields(logrus.Fields{
    "xapp":           config.XAppName,
    "subscription_id": sub.SubscriptionID,
    "node_id":        sub.NodeID,
}).Info("Subscription created")
```

### Performance Profiling

```go
// Enable pprof for profiling
import _ "net/http/pprof"

func enableProfiling() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}

// Profile CPU usage
// go tool pprof http://localhost:6060/debug/pprof/profile

// Profile memory usage
// go tool pprof http://localhost:6060/debug/pprof/heap
```

This comprehensive guide provides everything needed to develop, test, and deploy xApps using the Nephoran SDK.