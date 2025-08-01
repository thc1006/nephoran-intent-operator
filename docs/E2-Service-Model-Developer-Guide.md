# E2 Service Model Developer Guide

## Overview

The E2 Service Model defines the capabilities and procedures that can be performed between E2 Nodes (gNB/eNB) and the Near-RT RIC. This guide provides comprehensive documentation for implementing, extending, and using E2 Service Models in the Nephoran Intent Operator framework.

## Service Model Architecture

```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                           E2 Service Model Architecture                             │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                      Standard Service Models                                │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐ │   │
│  │  │  KPM v2.0       │  │   RC v1.0       │  │    NI v1.0                  │ │   │
│  │  │ (Key Performance│  │ (RAN Control)   │  │ (Network Information)       │ │   │
│  │  │  Measurement)   │  │                 │  │                             │ │   │
│  │  │                 │  │                 │  │                             │ │   │
│  │  │ • Cell Metrics  │  │ • Traffic Steer │  │ • Cell Info                 │ │   │
│  │  │ • UE Metrics    │  │ • QoS Control   │  │ • Neighbor Relations        │ │   │
│  │  │ • Bearer Stats  │  │ • Handover Ctrl │  │ • Configuration             │ │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘ │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                          │                                         │
│                                          ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                     Service Model Components                                │   │
│  │                                                                             │   │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │   │
│  │  │                 Service Model Definition                             │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │   │
│  │  │  │   Model ID  │  │  Procedures │  │   ASN.1     │                 │   │   │
│  │  │  │   & OID     │  │             │  │ Definition  │                 │   │   │
│  │  │  │             │  │ • Subscribe │  │             │                 │   │   │
│  │  │  │ • Unique ID │  │ • Control   │  │ • Messages  │                 │   │   │
│  │  │  │ • Version   │  │ • Report    │  │ • IEs       │                 │   │   │
│  │  │  │ • Name      │  │ • Indicate  │  │ • Encoding  │                 │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘                 │   │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │   │
│  │                                                                             │   │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │   │
│  │  │              Service Model Implementation                            │   │   │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                 │   │   │
│  │  │  │   Event     │  │   Action    │  │  Report     │                 │   │   │
│  │  │  │  Triggers   │  │ Definitions │  │  Formats    │                 │   │   │
│  │  │  │             │  │             │  │             │                 │   │   │
│  │  │  │ • Periodic  │  │ • Report    │  │ • Cell-level│                 │   │   │
│  │  │  │ • On Change │  │ • Insert    │  │ • UE-level  │                 │   │   │
│  │  │  │ • Threshold │  │ • Policy    │  │ • QoS-level │                 │   │   │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘                 │   │   │
│  │  └─────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

## Standard Service Models

### 1. KPM (Key Performance Measurement) Service Model

The KPM service model enables collection of performance measurements from E2 Nodes.

#### KPM v2.0 Features
- **Cell-level measurements**: PRB utilization, throughput, active UEs
- **UE-level measurements**: DL/UL throughput, RSRP, RSRQ, CQI
- **QoS-level measurements**: Per-QCI statistics, packet loss, latency
- **Slice-level measurements**: Slice-specific KPIs

#### Implementation Example

```go
package service_models

import (
    "github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
    "time"
)

// KPMServiceModel implements the E2SM-KPM v2.0 service model
type KPMServiceModel struct {
    ServiceModelID      string
    ServiceModelName    string
    ServiceModelVersion string
    ServiceModelOID     string
}

// NewKPMServiceModel creates a new KPM service model instance
func NewKPMServiceModel() *KPMServiceModel {
    return &KPMServiceModel{
        ServiceModelID:      "1.3.6.1.4.1.53148.1.1.2.2",
        ServiceModelName:    "KPM",
        ServiceModelVersion: "v2.0",
        ServiceModelOID:     "1.3.6.1.4.1.53148.1.1.2.2",
    }
}

// CreateEventTrigger creates a KPM event trigger definition
func (kpm *KPMServiceModel) CreateEventTrigger(config *KPMTriggerConfig) []byte {
    // Event trigger encoding based on E2SM-KPM ASN.1
    trigger := &E2SMKPMEventTriggerDefinition{
        EventDefinitionFormats: &E2SMKPMEventTriggerDefinitionFormat1{
            ReportingPeriod: config.ReportingPeriodMs,
        },
    }
    
    // Encode to ASN.1/JSON
    return encodeEventTrigger(trigger)
}

// CreateActionDefinition creates a KPM action definition
func (kpm *KPMServiceModel) CreateActionDefinition(config *KPMActionConfig) []byte {
    action := &E2SMKPMActionDefinition{
        ActionDefinitionFormats: &E2SMKPMActionDefinitionFormat1{
            MeasInfoList: config.Measurements,
            GranularityPeriod: config.GranularityPeriod,
            CellObjectID: config.CellID,
        },
    }
    
    return encodeActionDefinition(action)
}

// ParseIndication parses a KPM indication message
func (kpm *KPMServiceModel) ParseIndication(header, message []byte) (*KPMReport, error) {
    // Parse indication header
    indHeader := &E2SMKPMIndicationHeader{}
    if err := decodeIndicationHeader(header, indHeader); err != nil {
        return nil, err
    }
    
    // Parse indication message
    indMessage := &E2SMKPMIndicationMessage{}
    if err := decodeIndicationMessage(message, indMessage); err != nil {
        return nil, err
    }
    
    // Convert to KPM report
    report := &KPMReport{
        Timestamp:     time.Now(),
        CellID:        indHeader.CellObjectID,
        UECount:       indMessage.UEMeasurementReport.UECount,
        Measurements:  make(map[string]float64),
    }
    
    // Extract measurements
    for _, measData := range indMessage.MeasDataList {
        for _, measRecord := range measData.MeasRecordList {
            report.Measurements[measRecord.MeasName] = measRecord.MeasValue
        }
    }
    
    return report, nil
}
```

### 2. RC (RAN Control) Service Model

The RC service model enables control operations on RAN functions.

#### RC v1.0 Features
- **Traffic Steering**: Redirect UE traffic between cells
- **QoS Control**: Modify QoS parameters for flows
- **Handover Control**: Trigger or modify handover decisions
- **Dual Connectivity**: Control secondary cell additions

#### Implementation Example

```go
// RCServiceModel implements the E2SM-RC v1.0 service model
type RCServiceModel struct {
    ServiceModelID      string
    ServiceModelName    string
    ServiceModelVersion string
    ServiceModelOID     string
}

// NewRCServiceModel creates a new RC service model instance
func NewRCServiceModel() *RCServiceModel {
    return &RCServiceModel{
        ServiceModelID:      "1.3.6.1.4.1.53148.1.1.2.3",
        ServiceModelName:    "RC",
        ServiceModelVersion: "v1.0",
        ServiceModelOID:     "1.3.6.1.4.1.53148.1.1.2.3",
    }
}

// CreateControlHeader creates an RC control header
func (rc *RCServiceModel) CreateControlHeader(ueID string, controlType RCControlType) []byte {
    header := &E2SMRCControlHeader{
        ControlHeaderFormats: &E2SMRCControlHeaderFormat1{
            UEID:        ueID,
            ControlType: controlType,
            ControlStyle: 1,
        },
    }
    
    return encodeControlHeader(header)
}

// CreateControlMessage creates an RC control message
func (rc *RCServiceModel) CreateControlMessage(params *RCControlParams) []byte {
    message := &E2SMRCControlMessage{
        ControlMessageFormats: &E2SMRCControlMessageFormat1{
            RANParameters: params.ToRANParameterList(),
        },
    }
    
    return encodeControlMessage(message)
}

// ParseControlOutcome parses an RC control outcome
func (rc *RCServiceModel) ParseControlOutcome(outcome []byte) (*RCControlResult, error) {
    outcomeMsg := &E2SMRCControlOutcome{}
    if err := decodeControlOutcome(outcome, outcomeMsg); err != nil {
        return nil, err
    }
    
    result := &RCControlResult{
        Success: outcomeMsg.ControlOutcomeFormats.ControlStyle1.Result == "SUCCESS",
        Cause:   outcomeMsg.ControlOutcomeFormats.ControlStyle1.Cause,
    }
    
    return result, nil
}

// Example: Traffic Steering Control
func (rc *RCServiceModel) CreateTrafficSteeringControl(ueID string, targetCellID string) (*e2.RICControlRequest, error) {
    // Create control header
    header := rc.CreateControlHeader(ueID, RCControlTypeTrafficSteering)
    
    // Create control message with parameters
    params := &RCControlParams{
        Parameters: []RCParameter{
            {
                ParameterID:    1, // Target Cell ID
                ParameterValue: targetCellID,
            },
            {
                ParameterID:    2, // Handover Cause
                ParameterValue: "load-balancing",
            },
        },
    }
    message := rc.CreateControlMessage(params)
    
    // Build control request
    request := &e2.RICControlRequest{
        RICRequestID: e2.RICRequestID{
            RICRequestorID: 123,
            RICInstanceID:  1,
        },
        RANFunctionID:     e2.RANFunctionID(rc.GetFunctionID()),
        RICControlHeader:  header,
        RICControlMessage: message,
    }
    
    return request, nil
}
```

### 3. NI (Network Information) Service Model

The NI service model provides network configuration and topology information.

#### NI v1.0 Features
- **Cell Information**: Cell configuration, capabilities, status
- **Neighbor Relations**: Automatic neighbor relation updates
- **Network Slicing Info**: Slice configuration and capabilities
- **Interface Status**: X2/Xn interface status

## Custom Service Model Development

### Step 1: Define Service Model Interface

```go
// ServiceModel defines the interface for E2 service models
type ServiceModel interface {
    // Identification
    GetServiceModelID() string
    GetServiceModelName() string
    GetServiceModelVersion() string
    GetServiceModelOID() string
    
    // Subscription procedures
    CreateEventTrigger(config interface{}) ([]byte, error)
    CreateActionDefinition(config interface{}) ([]byte, error)
    ParseIndication(header, message []byte) (interface{}, error)
    
    // Control procedures
    CreateControlHeader(params interface{}) ([]byte, error)
    CreateControlMessage(params interface{}) ([]byte, error)
    ParseControlOutcome(outcome []byte) (interface{}, error)
    
    // Validation
    ValidateEventTrigger(trigger []byte) error
    ValidateActionDefinition(action []byte) error
    ValidateControlMessage(message []byte) error
}
```

### Step 2: Implement Service Model

```go
// CustomServiceModel implements a custom service model
type CustomServiceModel struct {
    serviceModelID      string
    serviceModelName    string
    serviceModelVersion string
    serviceModelOID     string
    supportedProcedures []string
}

// NewCustomServiceModel creates a new custom service model
func NewCustomServiceModel(name string, version string) *CustomServiceModel {
    return &CustomServiceModel{
        serviceModelID:      generateServiceModelID(name, version),
        serviceModelName:    name,
        serviceModelVersion: version,
        serviceModelOID:     generateOID(name, version),
        supportedProcedures: []string{
            "RIC_SUBSCRIPTION",
            "RIC_CONTROL",
            "RIC_POLICY",
        },
    }
}

// Implement required methods...
```

### Step 3: Register Service Model

```go
// ServiceModelRegistry manages service model registration
type ServiceModelRegistry struct {
    models map[string]ServiceModel
    mutex  sync.RWMutex
}

// Register adds a service model to the registry
func (r *ServiceModelRegistry) Register(model ServiceModel) error {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    
    modelID := model.GetServiceModelID()
    if _, exists := r.models[modelID]; exists {
        return fmt.Errorf("service model already registered: %s", modelID)
    }
    
    r.models[modelID] = model
    return nil
}

// Example registration
func RegisterServiceModels(registry *ServiceModelRegistry) {
    // Register standard models
    registry.Register(NewKPMServiceModel())
    registry.Register(NewRCServiceModel())
    registry.Register(NewNIServiceModel())
    
    // Register custom models
    registry.Register(NewCustomServiceModel("CUSTOM-ML", "v1.0"))
}
```

## Service Model Message Formats

### Event Trigger Definition

Event triggers define when the E2 Node should send indication messages.

```go
// EventTriggerType defines types of event triggers
type EventTriggerType string

const (
    EventTriggerTypePeriodic  EventTriggerType = "PERIODIC"
    EventTriggerTypeOnChange  EventTriggerType = "ON_CHANGE"
    EventTriggerTypeThreshold EventTriggerType = "THRESHOLD"
)

// EventTriggerConfig configures event triggers
type EventTriggerConfig struct {
    TriggerType     EventTriggerType
    ReportingPeriod time.Duration    // For periodic triggers
    ThresholdValue  float64          // For threshold triggers
    MonitoredMetric string           // Metric to monitor
}
```

### Action Definition

Actions define what data to collect or operations to perform.

```go
// ActionType defines types of actions
type ActionType string

const (
    ActionTypeReport ActionType = "REPORT"
    ActionTypeInsert ActionType = "INSERT"
    ActionTypePolicy ActionType = "POLICY"
)

// ActionConfig configures subscription actions
type ActionConfig struct {
    ActionID         int
    ActionType       ActionType
    Measurements     []string        // For report actions
    PolicyParameters map[string]string // For policy actions
}
```

### Indication Message Format

Indications carry measurement reports or event notifications.

```go
// IndicationHeader contains common indication information
type IndicationHeader struct {
    Timestamp       time.Time
    IndicationType  string
    CellGlobalID    string
    UEID            string // Optional
}

// IndicationMessage contains the actual data
type IndicationMessage struct {
    MeasurementData []MeasurementRecord
    EventData       map[string]interface{}
}

// MeasurementRecord represents a single measurement
type MeasurementRecord struct {
    MeasurementName  string
    MeasurementValue interface{}
    MeasurementUnit  string
    Timestamp        time.Time
}
```

## Best Practices

### 1. Service Model Design

- **Clear Procedures**: Define clear subscription, control, and reporting procedures
- **Extensibility**: Design for future extensions without breaking compatibility
- **Efficiency**: Minimize message size and processing overhead
- **Error Handling**: Provide detailed error information for debugging

### 2. Implementation Guidelines

- **Thread Safety**: Ensure thread-safe access to shared resources
- **Resource Management**: Implement proper cleanup for subscriptions
- **Validation**: Validate all inputs and outputs
- **Logging**: Provide comprehensive logging for troubleshooting

### 3. Testing Recommendations

- **Unit Tests**: Test individual message encoding/decoding
- **Integration Tests**: Test end-to-end message flows
- **Performance Tests**: Validate performance under load
- **Compatibility Tests**: Ensure interoperability with RIC

## Advanced Topics

### 1. Multi-Version Support

Support multiple versions of the same service model:

```go
type VersionedServiceModel struct {
    versions map[string]ServiceModel
    defaultVersion string
}

func (v *VersionedServiceModel) GetModel(version string) (ServiceModel, error) {
    if version == "" {
        version = v.defaultVersion
    }
    
    model, exists := v.versions[version]
    if !exists {
        return nil, fmt.Errorf("unsupported version: %s", version)
    }
    
    return model, nil
}
```

### 2. Service Model Plugins

Implement service models as plugins for dynamic loading:

```go
// ServiceModelPlugin defines the plugin interface
type ServiceModelPlugin interface {
    Name() string
    Version() string
    Load() (ServiceModel, error)
    Unload() error
}

// PluginManager manages service model plugins
type PluginManager struct {
    plugins map[string]ServiceModelPlugin
    loaded  map[string]ServiceModel
}

func (pm *PluginManager) LoadPlugin(pluginPath string) error {
    // Load plugin from file
    plugin, err := loadPluginFromFile(pluginPath)
    if err != nil {
        return err
    }
    
    // Initialize service model
    model, err := plugin.Load()
    if err != nil {
        return err
    }
    
    // Register in manager
    pm.plugins[plugin.Name()] = plugin
    pm.loaded[model.GetServiceModelID()] = model
    
    return nil
}
```

### 3. Service Model Composition

Combine multiple service models for complex scenarios:

```go
// CompositeServiceModel combines multiple service models
type CompositeServiceModel struct {
    models []ServiceModel
    name   string
}

func (c *CompositeServiceModel) CreateEventTrigger(configs map[string]interface{}) ([]byte, error) {
    triggers := make([][]byte, 0, len(c.models))
    
    for _, model := range c.models {
        modelName := model.GetServiceModelName()
        if config, exists := configs[modelName]; exists {
            trigger, err := model.CreateEventTrigger(config)
            if err != nil {
                return nil, fmt.Errorf("failed to create trigger for %s: %w", modelName, err)
            }
            triggers = append(triggers, trigger)
        }
    }
    
    // Combine triggers
    return combineEventTriggers(triggers), nil
}
```

## Troubleshooting

### Common Issues

1. **Invalid Service Model ID**: Ensure the service model is registered
2. **Encoding/Decoding Errors**: Validate message format against ASN.1 schema
3. **Unsupported Procedures**: Check service model capabilities
4. **Version Mismatch**: Ensure client and server use compatible versions

### Debug Logging

Enable detailed logging for service model operations:

```go
// ServiceModelLogger provides detailed logging
type ServiceModelLogger struct {
    logger *log.Logger
    debug  bool
}

func (l *ServiceModelLogger) LogMessage(direction string, modelID string, messageType string, data []byte) {
    if l.debug {
        l.logger.Printf("[%s] Model: %s, Type: %s, Size: %d bytes\nData: %x\n",
            direction, modelID, messageType, len(data), data)
    }
}
```

This comprehensive guide provides the foundation for developing and using E2 Service Models within the Nephoran Intent Operator framework.