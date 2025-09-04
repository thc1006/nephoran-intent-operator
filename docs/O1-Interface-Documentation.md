# O1 Interface Documentation - Nephoran Intent Operator

## Overview

The O1 interface in the Nephoran Intent Operator provides comprehensive FCAPS (Fault, Configuration, Accounting, Performance, Security) management capabilities for O-RAN network elements. This implementation follows O-RAN Alliance specifications and IETF NETCONF/YANG standards to ensure interoperability with O-RAN compliant network functions.

## Table of Contents

1. [Standards Compliance](#standards-compliance)
2. [Architecture Overview](#architecture-overview)
3. [API Reference](#api-reference)
4. [YANG Models](#yang-models)
5. [Integration Guides](#integration-guides)
6. [Deployment and Configuration](#deployment-and-configuration)
7. [Troubleshooting](#troubleshooting)
8. [Performance Tuning](#performance-tuning)
9. [Developer Guide](#developer-guide)
10. [Security Considerations](#security-considerations)

## Standards Compliance

### O-RAN Alliance Specifications

The O1 interface implementation adheres to the following O-RAN Alliance specifications:

- **O-RAN.WG4.MP.0-v01.00**: O-RAN Management Plane Specification
- **O-RAN.WG10.O1-Interface.0-v04.00**: O-RAN O1 Interface Specification
- **O-RAN.WG4.O1-Interface.0-v05.00**: O1 Interface General Aspects and Principles

### IETF Standards Compliance

The implementation follows these IETF standards:

- **RFC 6241**: Network Configuration Protocol (NETCONF)
- **RFC 7950**: The YANG 1.1 Data Modeling Language
- **RFC 8040**: RESTCONF Protocol
- **RFC 8342**: Network Management Datastore Architecture (NMDA)

### YANG Model Support

The implementation includes comprehensive support for:

- **o-ran-hardware**: Hardware component management (O-RAN.WG4.MP.0)
- **o-ran-software-management**: Software lifecycle management
- **o-ran-performance-management**: Performance monitoring and KPI collection
- **o-ran-fault-management**: Alarm management and fault handling
- **ietf-interfaces**: Standard interface configuration

## Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    O1 Interface Architecture                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐    ┌─────────────────┐                    │
│  │  Kubernetes     │    │   O1 Adaptor    │                    │
│  │  Controllers    │◄──►│   Interface     │                    │
│  │                 │    │                 │                    │
│  │ • NetworkIntent │    │ • FCAPS Mgmt    │                    │
│  │ • E2NodeSet     │    │ • NETCONF Client│                    │
│  │ • ManagedElem   │    │ • YANG Registry │                    │
│  └─────────────────┘    └─────────────────┘                    │
│                                   │                            │
│                                   ▼                            │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │              NETCONF/SSH Transport Layer                   ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐││
│  │  │    SSH      │  │  NETCONF    │  │      TLS/mTLS       │││
│  │  │  Client     │  │   RPC       │  │    (Optional)       │││
│  │  └─────────────┘  └─────────────┘  └─────────────────────┘││
│  └─────────────────────────────────────────────────────────────┘│
│                                   │                            │
│                                   ▼                            │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                O-RAN Network Elements                      ││
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐││
│  │  │    O-CU     │  │    O-DU     │  │       O-RU          │││
│  │  │ (Centralized│  │(Distributed │  │  (Radio Unit)       │││
│  │  │    Unit)    │  │    Unit)    │  │                     │││
│  │  └─────────────┘  └─────────────┘  └─────────────────────┘││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

### Component Overview

#### O1Adaptor
The main interface component providing:
- FCAPS management operations
- Connection management and pooling
- Thread-safe client operations
- Retry logic and error handling

#### NetconfClient
NETCONF protocol implementation featuring:
- SSH-based transport
- NETCONF 1.0/1.1 protocol support
- Capability negotiation
- Notification subscriptions

#### YANGModelRegistry
YANG model management system providing:
- Model registration and validation
- Schema-based configuration validation
- XPath expression validation
- Multi-model support

## API Reference

### O1AdaptorInterface

The main interface for O1 operations:

```go
type O1AdaptorInterface interface {
    // Configuration Management (CM)
    ApplyConfiguration(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error
    GetConfiguration(ctx context.Context, me *nephoranv1alpha1.ManagedElement) (string, error)
    ValidateConfiguration(ctx context.Context, config string) error
    
    // Fault Management (FM)
    GetAlarms(ctx context.Context, me *nephoranv1alpha1.ManagedElement) ([]*Alarm, error)
    ClearAlarm(ctx context.Context, me *nephoranv1alpha1.ManagedElement, alarmID string) error
    SubscribeToAlarms(ctx context.Context, me *nephoranv1alpha1.ManagedElement, callback AlarmCallback) error
    
    // Performance Management (PM)
    GetMetrics(ctx context.Context, me *nephoranv1alpha1.ManagedElement, metricNames []string) (map[string]interface{}, error)
    StartMetricCollection(ctx context.Context, me *nephoranv1alpha1.ManagedElement, config *MetricConfig) error
    StopMetricCollection(ctx context.Context, me *nephoranv1alpha1.ManagedElement, collectionID string) error
    
    // Accounting Management (AM)
    GetUsageRecords(ctx context.Context, me *nephoranv1alpha1.ManagedElement, filter *UsageFilter) ([]*UsageRecord, error)
    
    // Security Management (SM)
    UpdateSecurityPolicy(ctx context.Context, me *nephoranv1alpha1.ManagedElement, policy *SecurityPolicy) error
    GetSecurityStatus(ctx context.Context, me *nephoranv1alpha1.ManagedElement) (*SecurityStatus, error)
    
    // Connection Management
    Connect(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error
    Disconnect(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error
    IsConnected(me *nephoranv1alpha1.ManagedElement) bool
}
```

### Configuration Management API

#### ApplyConfiguration
Applies O1 configuration to a managed element using NETCONF edit-config operation.

**Function Signature:**
```go
func (a *O1Adaptor) ApplyConfiguration(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error
```

**Parameters:**
- `ctx`: Context for cancellation and timeout
- `me`: ManagedElement resource containing configuration data

**Usage Example:**
```go
managedElement := &nephoranv1alpha1.ManagedElement{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "o-cu-01",
        Namespace: "ran-network",
    },
    Spec: nephoranv1alpha1.ManagedElementSpec{
        Host: "10.0.1.100",
        Port: 830,
        Credentials: nephoranv1alpha1.Credentials{
            Username: "admin",
            Password: "secret",
        },
        O1Config: `
        <config>
            <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
                <interface>
                    <name>eth0</name>
                    <type>ianaift:ethernetCsmacd</type>
                    <enabled>true</enabled>
                </interface>
            </interfaces>
        </config>`,
    },
}

err := o1Adaptor.ApplyConfiguration(ctx, managedElement)
if err != nil {
    log.Printf("Configuration failed: %v", err)
}
```

#### GetConfiguration
Retrieves current configuration from a managed element.

**Function Signature:**
```go
func (a *O1Adaptor) GetConfiguration(ctx context.Context, me *nephoranv1alpha1.ManagedElement) (string, error)
```

**Usage Example:**
```go
config, err := o1Adaptor.GetConfiguration(ctx, managedElement)
if err != nil {
    log.Printf("Failed to retrieve configuration: %v", err)
    return
}
fmt.Printf("Current configuration:\n%s\n", config)
```

### Fault Management API

#### GetAlarms
Retrieves active alarms from a managed element.

**Function Signature:**
```go
func (a *O1Adaptor) GetAlarms(ctx context.Context, me *nephoranv1alpha1.ManagedElement) ([]*Alarm, error)
```

**Alarm Structure:**
```go
type Alarm struct {
    ID               string    `json:"alarm_id"`
    ManagedElementID string    `json:"managed_element_id"`
    Severity         string    `json:"severity"` // CRITICAL, MAJOR, MINOR, WARNING, CLEAR
    Type             string    `json:"type"`
    ProbableCause    string    `json:"probable_cause"`
    SpecificProblem  string    `json:"specific_problem"`
    AdditionalInfo   string    `json:"additional_info"`
    TimeRaised       time.Time `json:"time_raised"`
    TimeCleared      time.Time `json:"time_cleared,omitempty"`
}
```

**Usage Example:**
```go
alarms, err := o1Adaptor.GetAlarms(ctx, managedElement)
if err != nil {
    log.Printf("Failed to retrieve alarms: %v", err)
    return
}

for _, alarm := range alarms {
    fmt.Printf("Alarm ID: %s, Severity: %s, Problem: %s\n", 
        alarm.ID, alarm.Severity, alarm.SpecificProblem)
}
```

#### SubscribeToAlarms
Sets up alarm notification subscriptions.

**Function Signature:**
```go
func (a *O1Adaptor) SubscribeToAlarms(ctx context.Context, me *nephoranv1alpha1.ManagedElement, callback AlarmCallback) error

type AlarmCallback func(alarm *Alarm)
```

**Usage Example:**
```go
alarmHandler := func(alarm *Alarm) {
    log.Printf("Received alarm notification: ID=%s, Severity=%s", 
        alarm.ID, alarm.Severity)
    
    // Handle alarm based on severity
    switch alarm.Severity {
    case "CRITICAL":
        // Trigger immediate response
        handleCriticalAlarm(alarm)
    case "MAJOR":
        // Escalate to operations team
        escalateAlarm(alarm)
    default:
        // Log for later analysis
        logAlarm(alarm)
    }
}

err := o1Adaptor.SubscribeToAlarms(ctx, managedElement, alarmHandler)
if err != nil {
    log.Printf("Failed to subscribe to alarms: %v", err)
}
```

### Performance Management API

#### GetMetrics
Retrieves performance metrics from a managed element.

**Function Signature:**
```go
func (a *O1Adaptor) GetMetrics(ctx context.Context, me *nephoranv1alpha1.ManagedElement, metricNames []string) (map[string]interface{}, error)
```

**Supported Metrics:**
- `cpu_usage`: CPU utilization percentage
- `memory_usage`: Memory utilization percentage
- `temperature`: Component temperature (Celsius)
- `throughput_mbps`: Network throughput in Mbps
- `packet_loss_rate`: Packet loss percentage
- `power_consumption`: Power consumption in watts

**Usage Example:**
```go
metricNames := []string{"cpu_usage", "memory_usage", "temperature"}
metrics, err := o1Adaptor.GetMetrics(ctx, managedElement, metricNames)
if err != nil {
    log.Printf("Failed to retrieve metrics: %v", err)
    return
}

for name, value := range metrics {
    fmt.Printf("Metric %s: %v\n", name, value)
}
```

#### StartMetricCollection
Starts periodic metric collection.

**Function Signature:**
```go
func (a *O1Adaptor) StartMetricCollection(ctx context.Context, me *nephoranv1alpha1.ManagedElement, config *MetricConfig) error

type MetricConfig struct {
    MetricNames      []string      `json:"metric_names"`
    CollectionPeriod time.Duration `json:"collection_period"`
    ReportingPeriod  time.Duration `json:"reporting_period"`
    Aggregation      string        `json:"aggregation"` // MIN, MAX, AVG, SUM
}
```

**Usage Example:**
```go
metricConfig := &MetricConfig{
    MetricNames:      []string{"cpu_usage", "memory_usage"},
    CollectionPeriod: 30 * time.Second,
    ReportingPeriod:  5 * time.Minute,
    Aggregation:      "AVG",
}

err := o1Adaptor.StartMetricCollection(ctx, managedElement, metricConfig)
if err != nil {
    log.Printf("Failed to start metric collection: %v", err)
}
```

### Security Management API

#### UpdateSecurityPolicy
Updates security policies on a managed element.

**Function Signature:**
```go
func (a *O1Adaptor) UpdateSecurityPolicy(ctx context.Context, me *nephoranv1alpha1.ManagedElement, policy *SecurityPolicy) error

type SecurityPolicy struct {
    PolicyID     string         `json:"policy_id"`
    PolicyType   string         `json:"policy_type"`
    Rules        []SecurityRule `json:"rules"`
    Enforcement  string         `json:"enforcement"` // STRICT, PERMISSIVE
}

type SecurityRule struct {
    RuleID      string                 `json:"rule_id"`
    Action      string                 `json:"action"` // ALLOW, DENY, LOG
    Conditions  map[string]interface{} `json:"conditions"`
}
```

**Usage Example:**
```go
securityPolicy := &SecurityPolicy{
    PolicyID:   "firewall-policy-01",
    PolicyType: "access-control",
    Enforcement: "STRICT",
    Rules: []SecurityRule{
        {
            RuleID: "rule-001",
            Action: "ALLOW",
            Conditions: map[string]interface{}{
                "source": "10.0.0.0/8",
                "port":   "22",
                "protocol": "tcp",
            },
        },
        {
            RuleID: "rule-002",
            Action: "DENY",
            Conditions: map[string]interface{}{
                "source": "0.0.0.0/0",
                "port":   "*",
            },
        },
    },
}

err := o1Adaptor.UpdateSecurityPolicy(ctx, managedElement, securityPolicy)
if err != nil {
    log.Printf("Failed to update security policy: %v", err)
}
```

## YANG Models

### Supported YANG Models

The O1 interface implementation includes comprehensive YANG model support:

#### O-RAN Hardware Model (o-ran-hardware)
Manages hardware components and their states.

**Key Elements:**
- **hardware/component**: List of hardware components
- **state**: Operational state information
- **properties**: Hardware-specific properties

**Example Configuration:**
```xml
<hardware xmlns="urn:o-ran:hardware:1.0">
    <component>
        <name>cpu-1</name>
        <class>cpu</class>
        <state>
            <name>cpu-1</name>
            <type>iana-hardware:cpu</type>
            <admin-state>unlocked</admin-state>
            <oper-state>enabled</oper-state>
        </state>
    </component>
</hardware>
```

#### O-RAN Software Management Model (o-ran-software-management)
Handles software lifecycle operations.

**Key Elements:**
- **software-inventory/software-slot**: Software slots and versions
- **software-download**: Software download operations
- **software-install**: Installation procedures

**Example Configuration:**
```xml
<software-inventory xmlns="urn:o-ran:software-management:1.0">
    <software-slot>
        <name>slot-1</name>
        <status>valid</status>
        <active>true</active>
        <running>true</running>
        <access>read-write</access>
        <build-info>
            <build-name>o-cu-software</build-name>
            <build-version>2.1.0</build-version>
        </build-info>
    </software-slot>
</software-inventory>
```

#### O-RAN Performance Management Model (o-ran-performance-management)
Defines performance monitoring capabilities.

**Key Elements:**
- **performance-measurement-objects**: Available measurements
- **measurement-object**: Specific measurement definitions
- **measurement-type**: Types of measurements

**Example Configuration:**
```xml
<performance-measurement-objects xmlns="urn:o-ran:performance-management:1.0">
    <measurement-object>
        <measurement-object-id>cpu-utilization</measurement-object-id>
        <object-unit>percentage</object-unit>
        <function>system-monitoring</function>
        <measurement-type>
            <measurement-type-id>avg-cpu-usage</measurement-type-id>
            <measurement-description>Average CPU usage over collection period</measurement-description>
        </measurement-type>
    </measurement-object>
</performance-measurement-objects>
```

#### O-RAN Fault Management Model (o-ran-fault-management)
Manages alarm and fault information.

**Key Elements:**
- **active-alarm-list/active-alarms**: Current active alarms
- **alarm-notification**: Alarm notifications
- **fault-id**: Unique fault identifiers

**Example Alarm Data:**
```xml
<active-alarm-list xmlns="urn:o-ran:fault-management:1.0">
    <active-alarms>
        <fault-id>1001</fault-id>
        <fault-source>cpu-1</fault-source>
        <fault-severity>major</fault-severity>
        <is-cleared>false</is-cleared>
        <fault-text>CPU temperature exceeded threshold</fault-text>
        <event-time>2024-01-15T10:30:00Z</event-time>
    </active-alarms>
</active-alarm-list>
```

### YANG Model Registry

The YANGModelRegistry provides centralized model management:

#### Model Registration
```go
registry := NewYANGModelRegistry()

customModel := &YANGModel{
    Name:         "custom-model",
    Namespace:    "urn:company:custom:1.0",
    Version:      "1.0",
    Revision:     "2024-01-15",
    Description:  "Custom YANG model",
    Schema:       schemaDefinition,
}

err := registry.RegisterModel(customModel)
if err != nil {
    log.Printf("Failed to register model: %v", err)
}
```

#### Configuration Validation
```go
configData := map[string]interface{}{
    "hardware": map[string]interface{}{
        "component": []interface{}{
            map[string]interface{}{
                "name":  "cpu-1",
                "class": "cpu",
            },
        },
    },
}

err := registry.ValidateConfig(ctx, configData, "o-ran-hardware")
if err != nil {
    log.Printf("Configuration validation failed: %v", err)
}
```

## Integration Guides

### Integration with Kubernetes Controllers

The O1 interface integrates seamlessly with Kubernetes controllers:

#### NetworkIntent Controller Integration
```go
func (r *NetworkIntentReconciler) processO1Configuration(ctx context.Context, intent *nephoranv1alpha1.NetworkIntent) error {
    // Create ManagedElement from intent
    managedElement := &nephoranv1alpha1.ManagedElement{
        ObjectMeta: metav1.ObjectMeta{
            Name:      intent.Name + "-managed-element",
            Namespace: intent.Namespace,
        },
        Spec: nephoranv1alpha1.ManagedElementSpec{
            Host:       intent.Spec.TargetHost,
            Port:       830,
            O1Config:   intent.Spec.O1Configuration,
            Credentials: intent.Spec.Credentials,
        },
    }
    
    // Apply O1 configuration
    o1Adaptor := o1.NewO1Adaptor(nil)
    if err := o1Adaptor.ApplyConfiguration(ctx, managedElement); err != nil {
        return fmt.Errorf("O1 configuration failed: %w", err)
    }
    
    return nil
}
```

#### E2NodeSet Controller Integration
```go
func (r *E2NodeSetReconciler) configureE2Nodes(ctx context.Context, nodeSet *nephoranv1alpha1.E2NodeSet) error {
    for i := 0; i < int(nodeSet.Spec.Replicas); i++ {
        nodeName := fmt.Sprintf("%s-%d", nodeSet.Name, i)
        
        managedElement := &nephoranv1alpha1.ManagedElement{
            ObjectMeta: metav1.ObjectMeta{
                Name:      nodeName,
                Namespace: nodeSet.Namespace,
            },
            Spec: nephoranv1alpha1.ManagedElementSpec{
                Host: fmt.Sprintf("e2node-%d.%s", i, nodeSet.Spec.Domain),
                Port: 830,
                O1Config: generateE2NodeConfig(i, nodeSet.Spec),
            },
        }
        
        if err := r.o1Adaptor.ApplyConfiguration(ctx, managedElement); err != nil {
            return fmt.Errorf("failed to configure E2 node %s: %w", nodeName, err)
        }
    }
    
    return nil
}
```

### Multi-Vendor Compatibility

The O1 interface supports multi-vendor network elements through:

#### Vendor-Specific YANG Models
```go
// Register vendor-specific models
vendorModels := []*YANGModel{
    {
        Name:         "vendor-a-extensions",
        Namespace:    "urn:vendor-a:extensions:1.0",
        Version:      "1.0",
        Description:  "Vendor A specific extensions",
        Schema:       vendorASchema,
    },
    {
        Name:         "vendor-b-interfaces",
        Namespace:    "urn:vendor-b:interfaces:1.0", 
        Version:      "1.0",
        Description:  "Vendor B interface definitions",
        Schema:       vendorBSchema,
    },
}

for _, model := range vendorModels {
    registry.RegisterModel(model)
}
```

#### Capability-Based Configuration
```go
func (a *O1Adaptor) adaptConfigurationForVendor(config string, capabilities []string) string {
    // Check vendor capabilities and adapt configuration
    if containsCapability(capabilities, "urn:vendor-a:proprietary:1.0") {
        return adaptForVendorA(config)
    } else if containsCapability(capabilities, "urn:vendor-b:custom:1.0") {
        return adaptForVendorB(config)
    }
    
    // Return standard O-RAN configuration
    return config
}
```

## Deployment and Configuration

### Kubernetes Deployment

#### ManagedElement CRD
```yaml
apiVersion: nephoran.com/v1alpha1
kind: ManagedElement
metadata:
  name: o-cu-cp-01
  namespace: ran-network
spec:
  deploymentName: "o-cu-cp-deployment"
  host: "192.168.1.100"
  port: 830
  credentials:
    username: "admin"
    password: "secret"
  o1Config: |
    <config>
      <interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
        <interface>
          <name>eth0</name>
          <type>ianaift:ethernetCsmacd</type>
          <enabled>true</enabled>
          <ipv4 xmlns="urn:ietf:params:xml:ns:yang:ietf-ip">
            <address>
              <ip>192.168.1.100</ip>
              <prefix-length>24</prefix-length>
            </address>
          </ipv4>
        </interface>
      </interfaces>
    </config>
```

#### O1 Adaptor Configuration
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: o1-adaptor-config
  namespace: nephoran-system
data:
  config.yaml: |
    o1:
      defaultPort: 830
      connectTimeout: 30s
      requestTimeout: 60s
      maxRetries: 3
      retryInterval: 5s
      tls:
        enabled: false
        certFile: ""
        keyFile: ""
        caFile: ""
        insecureSkipVerify: false
    yangModels:
      autoLoad: true
      modelPaths:
        - "/etc/yang-models"
        - "/usr/share/yang"
    metrics:
      enabled: true
      collectionInterval: 30s
      retentionPeriod: 24h
```

### Container Deployment

#### Dockerfile
```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o manager cmd/main.go

FROM alpine:3.18
RUN apk --no-cache add ca-certificates openssh-client
WORKDIR /root/

# Install YANG models
COPY --from=builder /workspace/yang-models /etc/yang-models/
COPY --from=builder /workspace/manager .

EXPOSE 8080 8443
ENTRYPOINT ["/root/manager"]
```

#### Kubernetes Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephoran-o1-adaptor
  namespace: nephoran-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nephoran-o1-adaptor
  template:
    metadata:
      labels:
        app: nephoran-o1-adaptor
    spec:
      serviceAccountName: nephoran-o1-adaptor
      containers:
      - name: manager
        image: nephoran/o1-adaptor:latest
        ports:
        - containerPort: 8080
          name: metrics
        - containerPort: 8443
          name: webhook
        env:
        - name: O1_CONFIG_FILE
          value: /etc/config/config.yaml
        volumeMounts:
        - name: config
          mountPath: /etc/config
        - name: yang-models
          mountPath: /etc/yang-models
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 15
          periodSeconds: 20
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: config
        configMap:
          name: o1-adaptor-config
      - name: yang-models
        configMap:
          name: yang-models
```

### Security Configuration

#### TLS/mTLS Setup
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: o1-tls-certs
  namespace: nephoran-system
type: kubernetes.io/tls
data:
  tls.crt: <base64-encoded-certificate>
  tls.key: <base64-encoded-private-key>
  ca.crt: <base64-encoded-ca-certificate>
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: o1-adaptor-tls-config
  namespace: nephoran-system
data:
  tls-config.yaml: |
    tls:
      enabled: true
      certFile: "/etc/tls/tls.crt"
      keyFile: "/etc/tls/tls.key"
      caFile: "/etc/tls/ca.crt"
      insecureSkipVerify: false
      minVersion: "1.3"
      cipherSuites:
        - "TLS_AES_256_GCM_SHA384"
        - "TLS_CHACHA20_POLY1305_SHA256"
        - "TLS_AES_128_GCM_SHA256"
```

#### SSH Key Management
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: o1-ssh-keys
  namespace: nephoran-system
type: Opaque
data:
  private-key: <base64-encoded-ssh-private-key>
  public-key: <base64-encoded-ssh-public-key>
  known-hosts: <base64-encoded-known-hosts-file>
```

## Error Handling and Recovery

### Connection Management

The O1 interface implements robust connection management:

#### Automatic Reconnection
```go
func (a *O1Adaptor) ensureConnection(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error {
    if a.IsConnected(me) {
        return nil
    }
    
    // Implement exponential backoff
    backoff := time.Second
    maxRetries := a.config.MaxRetries
    
    for attempt := 1; attempt <= maxRetries; attempt++ {
        if err := a.Connect(ctx, me); err == nil {
            return nil
        }
        
        if attempt < maxRetries {
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(backoff):
                backoff *= 2
                if backoff > time.Minute {
                    backoff = time.Minute
                }
            }
        }
    }
    
    return fmt.Errorf("failed to establish connection after %d attempts", maxRetries)
}
```

#### Circuit Breaker Pattern
```go
type CircuitBreaker struct {
    failures    int
    lastFailure time.Time
    threshold   int
    timeout     time.Duration
    state       CircuitState
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    if cb.state == StateOpen {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = StateHalfOpen
        } else {
            return ErrCircuitOpen
        }
    }
    
    err := fn()
    
    if err != nil {
        cb.failures++
        cb.lastFailure = time.Now()
        
        if cb.failures >= cb.threshold {
            cb.state = StateOpen
        }
        
        return err
    }
    
    cb.failures = 0
    cb.state = StateClosed
    return nil
}
```

### Error Types and Handling

#### Custom Error Types
```go
type O1Error struct {
    Code    string
    Message string
    Cause   error
}

func (e *O1Error) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("O1 Error %s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("O1 Error %s: %s", e.Code, e.Message)
}

// Predefined error codes
const (
    ErrCodeConnectionFailed   = "CONNECTION_FAILED"
    ErrCodeAuthenticationFailed = "AUTH_FAILED"
    ErrCodeConfigInvalid      = "CONFIG_INVALID"
    ErrCodeNetconfError       = "NETCONF_ERROR"
    ErrCodeTimeout            = "TIMEOUT"
)
```

#### Error Recovery Strategies
```go
func (a *O1Adaptor) executeWithRetry(ctx context.Context, operation func() error) error {
    return retry.Do(
        operation,
        retry.Context(ctx),
        retry.Attempts(uint(a.config.MaxRetries)),
        retry.Delay(a.config.RetryInterval),
        retry.DelayType(retry.BackOffDelay),
        retry.OnRetry(func(n uint, err error) {
            log.Printf("Operation failed (attempt %d): %v", n+1, err)
        }),
    )
}
```

This documentation provides comprehensive coverage of the O1 interface implementation, including standards compliance, architecture, API reference, integration guides, and deployment procedures. The next sections will cover troubleshooting, performance tuning, and developer guides.