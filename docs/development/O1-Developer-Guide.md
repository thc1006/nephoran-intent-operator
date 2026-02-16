# O1 Interface Developer Guide

## Overview

This developer guide provides comprehensive information for developing, extending, and contributing to the O1 interface in the Nephoran Intent Operator. It covers architecture patterns, API usage, testing strategies, and contribution guidelines.

## Table of Contents

1. [Development Environment Setup](#development-environment-setup)
2. [Architecture and Design Patterns](#architecture-and-design-patterns)
3. [NETCONF Client Architecture](#netconf-client-architecture)
4. [YANG Model Registry](#yang-model-registry)
5. [Extending the O1 Interface](#extending-the-o1-interface)
6. [Testing Strategies](#testing-strategies)
7. [API Development Guidelines](#api-development-guidelines)
8. [Performance Considerations](#performance-considerations)
9. [Security Implementation](#security-implementation)
10. [Contribution Guidelines](#contribution-guidelines)

## Development Environment Setup

### Prerequisites

Before starting development, ensure you have the following tools installed:

#### Required Tools
- **Go**: Version 1.21 or higher
- **Docker**: For containerization and testing
- **kubectl**: For Kubernetes interaction
- **kind** or **minikube**: For local Kubernetes clusters
- **Make**: For build automation

#### Optional Tools
- **GoLand/VS Code**: IDEs with Go support
- **Delve**: Go debugger
- **golangci-lint**: Go linting tool
- **YANG tools**: libyang, pyang for YANG model validation

### Local Development Setup

#### 1. Clone and Build

```bash
# Clone the repository
git clone https://github.com/nephoran/nephoran-intent-operator.git
cd nephoran-intent-operator

# Install dependencies
go mod download

# Build the project
make build-all

# Run tests
make test
```

#### 2. Development Dependencies

```bash
# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/go-delve/delve/cmd/dlv@latest

# Install YANG tools (optional)
# Ubuntu/Debian
sudo apt-get install libyang-dev pyang

# macOS
brew install libyang pyang
```

#### 3. IDE Configuration

**VS Code Configuration (`.vscode/settings.json`):**
```json
{
    "go.useLanguageServer": true,
    "go.gopath": "",
    "go.goroot": "",
    "go.lintTool": "golangci-lint",
    "go.lintFlags": [
        "--fast"
    ],
    "go.testFlags": [
        "-v",
        "-race"
    ],
    "go.buildTags": "integration",
    "go.testTimeout": "30s"
}
```

**GoLand Configuration:**
- Enable Go modules support
- Set up run configurations for tests
- Configure debugger for remote debugging

### Local Testing Environment

#### 1. NETCONF Server Setup

For development and testing, set up a local NETCONF server:

```bash
# Using netopeer2 (open-source NETCONF server)
docker run -d --name netconf-server \
  -p 830:830 \
  -e USER=admin \
  -e PASS=admin \
  netopeer2/netopeer2:latest

# Verify connection
ssh admin@localhost -p 830 -s netconf
```

#### 2. Local Kubernetes Cluster

```bash
# Create kind cluster
kind create cluster --config=hack/kind-config.yaml

# Deploy CRDs
make deploy-crds

# Deploy O1 adaptor for testing
make deploy-dev
```

## Architecture and Design Patterns

### Overall Architecture

The O1 interface follows a layered architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                        │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │ NetworkIntent   │  │  E2NodeSet      │  [Controllers]   │
│  │ Controller      │  │  Controller     │                  │
│  └─────────────────┘  └─────────────────┘                  │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│                  O1 Interface Layer                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │   O1Adaptor     │  │ YANGModelRegistry│  [O1 Layer]     │
│  │   Interface     │  │                 │                  │
│  └─────────────────┘  └─────────────────┘                  │
└─────────────────────────────────────────────────────────────┘
                                │
┌─────────────────────────────────────────────────────────────┐
│                  Transport Layer                            │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐                  │
│  │ NetconfClient   │  │   SSH/TLS       │  [Transport]     │
│  │                 │  │   Transport     │                  │
│  └─────────────────┘  └─────────────────┘                  │
└─────────────────────────────────────────────────────────────┘
```

### Design Patterns

#### 1. Interface Segregation Pattern

The O1 interface uses interface segregation to provide focused, testable components:

```go
// Main interface for O1 operations
type O1AdaptorInterface interface {
    ConfigurationManager
    FaultManager
    PerformanceManager
    AccountingManager
    SecurityManager
    ConnectionManager
}

// Segregated interfaces for specific functionality
type ConfigurationManager interface {
    ApplyConfiguration(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error
    GetConfiguration(ctx context.Context, me *nephoranv1alpha1.ManagedElement) (string, error)
    ValidateConfiguration(ctx context.Context, config string) error
}

type FaultManager interface {
    GetAlarms(ctx context.Context, me *nephoranv1alpha1.ManagedElement) ([]*Alarm, error)
    ClearAlarm(ctx context.Context, me *nephoranv1alpha1.ManagedElement, alarmID string) error
    SubscribeToAlarms(ctx context.Context, me *nephoranv1alpha1.ManagedElement, callback AlarmCallback) error
}
```

#### 2. Factory Pattern

NETCONF client creation uses the factory pattern for flexibility:

```go
// NetconfClientFactory creates NETCONF clients based on configuration
type NetconfClientFactory interface {
    CreateClient(config *NetconfConfig) (NetconfClientInterface, error)
}

type DefaultNetconfClientFactory struct{}

func (f *DefaultNetconfClientFactory) CreateClient(config *NetconfConfig) (NetconfClientInterface, error) {
    if config.TLS.Enabled {
        return NewTLSNetconfClient(config), nil
    }
    return NewSSHNetconfClient(config), nil
}
```

#### 3. Observer Pattern

Event handling uses the observer pattern for alarm subscriptions:

```go
// EventSubscription manages event observers
type EventSubscription struct {
    subscribers map[string][]EventCallback
    mutex       sync.RWMutex
}

func (es *EventSubscription) Subscribe(eventType string, callback EventCallback) {
    es.mutex.Lock()
    defer es.mutex.Unlock()
    
    if es.subscribers[eventType] == nil {
        es.subscribers[eventType] = make([]EventCallback, 0)
    }
    es.subscribers[eventType] = append(es.subscribers[eventType], callback)
}

func (es *EventSubscription) Notify(eventType string, event interface{}) {
    es.mutex.RLock()
    defer es.mutex.RUnlock()
    
    for _, callback := range es.subscribers[eventType] {
        go callback(event)
    }
}
```

#### 4. Strategy Pattern

YANG validation uses the strategy pattern for different validation approaches:

```go
// YANGValidator interface for different validation strategies
type YANGValidator interface {
    ValidateData(data interface{}, modelName string) error
    ValidateXPath(xpath string, modelName string) error
}

// Different validation strategies
type StandardYANGValidator struct {
    registry *YANGModelRegistry
}

type StrictYANGValidator struct {
    registry *YANGModelRegistry
    strictMode bool
}

type CustomYANGValidator struct {
    registry *YANGModelRegistry
    customRules []ValidationRule
}
```

## NETCONF Client Architecture

### Core Components

#### NetconfClient Structure

```go
type NetconfClient struct {
    // Connection management
    conn         net.Conn
    sshClient    *ssh.Client
    session      *ssh.Session
    
    // I/O streams
    stdin        io.WriteCloser
    stdout       io.Reader
    
    // Session information
    sessionID    string
    capabilities []string
    connected    bool
    
    // Thread safety
    mutex        sync.RWMutex
    
    // Configuration
    config       *NetconfConfig
    
    // Message handling
    messageID    int64
    pendingRPCs  map[string]chan *NetconfReply
    
    // Event handling
    eventHandlers map[string][]EventCallback
}
```

#### Connection Management

**Connection Pool Implementation:**
```go
type ConnectionPool struct {
    pools     map[string]*HostPool
    mutex     sync.RWMutex
    config    *PoolConfig
}

type HostPool struct {
    connections chan *NetconfClient
    active      int
    maxActive   int
    maxIdle     int
    host        string
    port        int
}

func (cp *ConnectionPool) GetConnection(host string, port int) (*NetconfClient, error) {
    cp.mutex.RLock()
    pool, exists := cp.pools[fmt.Sprintf("%s:%d", host, port)]
    cp.mutex.RUnlock()
    
    if !exists {
        cp.mutex.Lock()
        pool = &HostPool{
            connections: make(chan *NetconfClient, cp.config.MaxIdle),
            maxActive:   cp.config.MaxActive,
            maxIdle:     cp.config.MaxIdle,
            host:        host,
            port:        port,
        }
        cp.pools[fmt.Sprintf("%s:%d", host, port)] = pool
        cp.mutex.Unlock()
    }
    
    select {
    case client := <-pool.connections:
        if client.IsConnected() {
            return client, nil
        }
        // Connection is dead, create new one
        fallthrough
    default:
        if pool.active >= pool.maxActive {
            return nil, ErrPoolExhausted
        }
        
        pool.active++
        client, err := cp.createNewConnection(host, port)
        if err != nil {
            pool.active--
            return nil, err
        }
        return client, nil
    }
}
```

#### Message Handling

**RPC Message Correlation:**
```go
func (nc *NetconfClient) sendRPCAsync(operation string) (<-chan *NetconfReply, error) {
    messageID := nc.getNextMessageID()
    
    rpc := NetconfRPC{
        MessageID: messageID,
        Namespace: "urn:ietf:params:xml:ns:netconf:base:1.0",
        Operation: operation,
    }
    
    // Create response channel
    responseChannel := make(chan *NetconfReply, 1)
    
    nc.mutex.Lock()
    nc.pendingRPCs[messageID] = responseChannel
    nc.mutex.Unlock()
    
    // Send RPC
    rpcXML, err := xml.Marshal(rpc)
    if err != nil {
        nc.mutex.Lock()
        delete(nc.pendingRPCs, messageID)
        nc.mutex.Unlock()
        return nil, fmt.Errorf("failed to marshal RPC: %w", err)
    }
    
    message := fmt.Sprintf("%s]]>]]>", string(rpcXML))
    if _, err := nc.stdin.Write([]byte(message)); err != nil {
        nc.mutex.Lock()
        delete(nc.pendingRPCs, messageID)
        nc.mutex.Unlock()
        return nil, fmt.Errorf("failed to send RPC: %w", err)
    }
    
    return responseChannel, nil
}
```

#### Error Handling

**Comprehensive Error Types:**
```go
// O1Error represents various O1 operation errors
type O1Error struct {
    Code        string                 `json:"code"`
    Message     string                 `json:"message"`
    Details     map[string]interface{} `json:"details,omitempty"`
    Cause       error                  `json:"-"`
    Timestamp   time.Time              `json:"timestamp"`
}

func (e *O1Error) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("O1 Error [%s]: %s (caused by: %v)", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("O1 Error [%s]: %s", e.Code, e.Message)
}

// Error constructors for different scenarios
func NewConnectionError(host string, port int, cause error) *O1Error {
    return &O1Error{
        Code:    "CONNECTION_FAILED",
        Message: fmt.Sprintf("Failed to connect to %s:%d", host, port),
        Details: map[string]interface{}{
            "host": host,
            "port": port,
        },
        Cause:     cause,
        Timestamp: time.Now(),
    }
}

func NewYANGValidationError(modelName string, path string, cause error) *O1Error {
    return &O1Error{
        Code:    "YANG_VALIDATION_FAILED",
        Message: fmt.Sprintf("YANG validation failed for model %s at path %s", modelName, path),
        Details: map[string]interface{}{
            "model": modelName,
            "path":  path,
        },
        Cause:     cause,
        Timestamp: time.Now(),
    }
}
```

## YANG Model Registry

### Registry Architecture

The YANG Model Registry provides centralized management of YANG schemas:

```go
type YANGModelRegistry struct {
    // Model storage
    models        map[string]*YANGModel        // Models by name
    modelsByNS    map[string]*YANGModel        // Models by namespace
    dependencies  map[string][]string          // Model dependencies
    
    // Validation
    validators    map[string]YANGValidator     // Validators by type
    
    // Caching
    schemaCache   map[string]*CompiledSchema   // Compiled schemas
    
    // Concurrency
    mutex         sync.RWMutex
    
    // Statistics
    loadedModules map[string]time.Time
    stats         *RegistryStatistics
}
```

### Model Loading and Registration

**Dynamic Model Loading:**
```go
func (yr *YANGModelRegistry) LoadModelsFromDirectory(dir string) error {
    return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        if filepath.Ext(path) == ".yang" {
            model, err := yr.parseYANGFile(path)
            if err != nil {
                log.Printf("Failed to parse YANG file %s: %v", path, err)
                return nil // Continue with other files
            }
            
            if err := yr.RegisterModel(model); err != nil {
                log.Printf("Failed to register model %s: %v", model.Name, err)
            }
        }
        
        return nil
    })
}

func (yr *YANGModelRegistry) parseYANGFile(filePath string) (*YANGModel, error) {
    content, err := ioutil.ReadFile(filePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }
    
    // Parse YANG content using libyang or custom parser
    parser := NewYANGParser()
    model, err := parser.Parse(content)
    if err != nil {
        return nil, fmt.Errorf("failed to parse YANG content: %w", err)
    }
    
    model.ModulePath = filePath
    model.LoadTime = time.Now()
    
    return model, nil
}
```

**Dependency Resolution:**
```go
func (yr *YANGModelRegistry) ResolveDependencies(modelName string) error {
    model, err := yr.GetModel(modelName)
    if err != nil {
        return err
    }
    
    // Check if dependencies are loaded
    for _, dep := range model.Dependencies {
        if _, err := yr.GetModel(dep); err != nil {
            // Try to load dependency
            if err := yr.loadDependency(dep); err != nil {
                return fmt.Errorf("failed to load dependency %s for model %s: %w", dep, modelName, err)
            }
        }
    }
    
    return nil
}

func (yr *YANGModelRegistry) loadDependency(depName string) error {
    // Search in standard locations
    standardPaths := []string{
        "/usr/share/yang/modules",
        "/etc/yang/modules",
        "./yang-models",
    }
    
    for _, path := range standardPaths {
        yangFile := filepath.Join(path, depName+".yang")
        if _, err := os.Stat(yangFile); err == nil {
            model, err := yr.parseYANGFile(yangFile)
            if err != nil {
                continue
            }
            return yr.RegisterModel(model)
        }
    }
    
    return fmt.Errorf("dependency %s not found", depName)
}
```

### Advanced Validation

**Schema Compilation and Caching:**
```go
type CompiledSchema struct {
    Model       *YANGModel
    Schema      interface{}     // Compiled schema representation
    Validator   SchemaValidator
    CompileTime time.Time
    LastUsed    time.Time
}

func (yr *YANGModelRegistry) getCompiledSchema(modelName string) (*CompiledSchema, error) {
    yr.mutex.RLock()
    compiled, exists := yr.schemaCache[modelName]
    yr.mutex.RUnlock()
    
    if exists && time.Since(compiled.CompileTime) < time.Hour {
        compiled.LastUsed = time.Now()
        return compiled, nil
    }
    
    // Compile schema
    model, err := yr.GetModel(modelName)
    if err != nil {
        return nil, err
    }
    
    compiled, err = yr.compileSchema(model)
    if err != nil {
        return nil, err
    }
    
    yr.mutex.Lock()
    yr.schemaCache[modelName] = compiled
    yr.mutex.Unlock()
    
    return compiled, nil
}
```

**Advanced Validation Rules:**
```go
type ValidationRule struct {
    Name        string
    Description string
    Path        string
    Condition   string
    ErrorMessage string
    Severity    ValidationSeverity
}

type ValidationSeverity int

const (
    SeverityInfo ValidationSeverity = iota
    SeverityWarning
    SeverityError
    SeverityCritical
)

func (yr *YANGModelRegistry) AddCustomValidationRule(modelName string, rule ValidationRule) error {
    model, err := yr.GetModel(modelName)
    if err != nil {
        return err
    }
    
    if model.CustomRules == nil {
        model.CustomRules = make([]ValidationRule, 0)
    }
    
    model.CustomRules = append(model.CustomRules, rule)
    
    // Recompile schema with new rule
    yr.mutex.Lock()
    delete(yr.schemaCache, modelName)
    yr.mutex.Unlock()
    
    return nil
}
```

## Extending the O1 Interface

### Adding New FCAPS Operations

#### 1. Define Interface Extensions

```go
// ExtendedO1Interface adds custom operations
type ExtendedO1Interface interface {
    O1AdaptorInterface
    
    // Custom operations
    GetDetailedSystemInfo(ctx context.Context, me *nephoranv1alpha1.ManagedElement) (*SystemInfo, error)
    PerformSystemBackup(ctx context.Context, me *nephoranv1alpha1.ManagedElement, backupConfig *BackupConfig) error
    RestoreFromBackup(ctx context.Context, me *nephoranv1alpha1.ManagedElement, backupID string) error
}

// SystemInfo represents detailed system information
type SystemInfo struct {
    Hostname     string                 `json:"hostname"`
    Version      string                 `json:"version"`
    Uptime       time.Duration          `json:"uptime"`
    Resources    ResourceInfo           `json:"resources"`
    Interfaces   []InterfaceInfo        `json:"interfaces"`
    Services     []ServiceInfo          `json:"services"`
    CustomData   map[string]interface{} `json:"custom_data,omitempty"`
}
```

#### 2. Implement Extensions

```go
// ExtendedO1Adaptor implements the extended interface
type ExtendedO1Adaptor struct {
    *O1Adaptor
    systemInfoCollector SystemInfoCollector
    backupManager       BackupManager
}

func NewExtendedO1Adaptor(config *O1Config) *ExtendedO1Adaptor {
    baseAdaptor := NewO1Adaptor(config)
    
    return &ExtendedO1Adaptor{
        O1Adaptor:           baseAdaptor,
        systemInfoCollector: NewSystemInfoCollector(),
        backupManager:       NewBackupManager(),
    }
}

func (ea *ExtendedO1Adaptor) GetDetailedSystemInfo(ctx context.Context, me *nephoranv1alpha1.ManagedElement) (*SystemInfo, error) {
    // Ensure connection
    if !ea.IsConnected(me) {
        if err := ea.Connect(ctx, me); err != nil {
            return nil, fmt.Errorf("failed to connect: %w", err)
        }
    }
    
    // Collect system information using multiple NETCONF queries
    systemInfo := &SystemInfo{
        CustomData: make(map[string]interface{}),
    }
    
    // Get hostname
    hostname, err := ea.getHostname(ctx, me)
    if err != nil {
        return nil, fmt.Errorf("failed to get hostname: %w", err)
    }
    systemInfo.Hostname = hostname
    
    // Get version information
    version, err := ea.getVersionInfo(ctx, me)
    if err != nil {
        return nil, fmt.Errorf("failed to get version: %w", err)
    }
    systemInfo.Version = version
    
    // Get resource information
    resources, err := ea.getResourceInfo(ctx, me)
    if err != nil {
        return nil, fmt.Errorf("failed to get resources: %w", err)
    }
    systemInfo.Resources = resources
    
    return systemInfo, nil
}
```

### Adding Custom YANG Models

#### 1. Define Custom Model Structure

```go
// CustomYANGModel represents a vendor-specific or extended YANG model
type CustomYANGModel struct {
    *YANGModel
    
    // Custom extensions
    VendorExtensions map[string]interface{} `json:"vendor_extensions,omitempty"`
    CustomNodes      map[string]*YANGNode   `json:"custom_nodes,omitempty"`
    ValidationHooks  []ValidationHook       `json:"validation_hooks,omitempty"`
}

type ValidationHook struct {
    Name        string
    TriggerPath string
    Handler     func(data interface{}) error
}
```

#### 2. Implement Model Registration

```go
func (yr *YANGModelRegistry) RegisterCustomModel(customModel *CustomYANGModel) error {
    // Validate custom model
    if err := yr.validateCustomModel(customModel); err != nil {
        return fmt.Errorf("custom model validation failed: %w", err)
    }
    
    // Register base model
    if err := yr.RegisterModel(customModel.YANGModel); err != nil {
        return fmt.Errorf("failed to register base model: %w", err)
    }
    
    // Register validation hooks
    for _, hook := range customModel.ValidationHooks {
        if err := yr.registerValidationHook(customModel.Name, hook); err != nil {
            return fmt.Errorf("failed to register validation hook %s: %w", hook.Name, err)
        }
    }
    
    return nil
}

func (yr *YANGModelRegistry) validateCustomModel(customModel *CustomYANGModel) error {
    // Validate that custom nodes don't conflict with standard nodes
    standardModel, err := yr.GetModel(customModel.Name)
    if err == nil {
        // Model already exists, check compatibility
        if !yr.isCompatible(standardModel, customModel.YANGModel) {
            return fmt.Errorf("custom model is not compatible with existing model")
        }
    }
    
    // Validate custom nodes
    for nodeName, node := range customModel.CustomNodes {
        if err := yr.validateCustomNode(nodeName, node); err != nil {
            return fmt.Errorf("invalid custom node %s: %w", nodeName, err)
        }
    }
    
    return nil
}
```

### Plugin Architecture

#### 1. Plugin Interface

```go
// O1Plugin represents an O1 interface plugin
type O1Plugin interface {
    Name() string
    Version() string
    Initialize(adaptor O1AdaptorInterface) error
    Shutdown() error
    
    // Optional interfaces
    ConfigurationExtender
    ValidationExtender
    MetricsExtender
}

type ConfigurationExtender interface {
    ExtendConfiguration(config *O1Config) error
}

type ValidationExtender interface {
    AddValidationRules() []ValidationRule
}

type MetricsExtender interface {
    GetCustomMetrics() []MetricDefinition
}
```

#### 2. Plugin Manager

```go
type PluginManager struct {
    plugins     map[string]O1Plugin
    pluginPaths []string
    mutex       sync.RWMutex
}

func NewPluginManager(pluginPaths []string) *PluginManager {
    return &PluginManager{
        plugins:     make(map[string]O1Plugin),
        pluginPaths: pluginPaths,
    }
}

func (pm *PluginManager) LoadPlugins() error {
    for _, path := range pm.pluginPaths {
        if err := pm.loadPluginsFromPath(path); err != nil {
            return fmt.Errorf("failed to load plugins from %s: %w", path, err)
        }
    }
    return nil
}

func (pm *PluginManager) RegisterPlugin(plugin O1Plugin) error {
    pm.mutex.Lock()
    defer pm.mutex.Unlock()
    
    name := plugin.Name()
    if _, exists := pm.plugins[name]; exists {
        return fmt.Errorf("plugin %s already registered", name)
    }
    
    pm.plugins[name] = plugin
    return nil
}
```

## Testing Strategies

### Unit Testing

#### Test Structure

```go
package o1

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/suite"
)

// O1AdaptorTestSuite provides test suite for O1Adaptor
type O1AdaptorTestSuite struct {
    suite.Suite
    adaptor    *O1Adaptor
    mockClient *MockNetconfClient
    testElement *nephoranv1alpha1.ManagedElement
}

func (suite *O1AdaptorTestSuite) SetupTest() {
    suite.mockClient = NewMockNetconfClient()
    suite.adaptor = NewO1AdaptorWithClient(suite.mockClient)
    
    suite.testElement = &nephoranv1alpha1.ManagedElement{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-element",
            Namespace: "test",
        },
        Spec: nephoranv1alpha1.ManagedElementSpec{
            Host: "test-host",
            Port: 830,
            Credentials: nephoranv1alpha1.Credentials{
                Username: "admin",
                Password: "test",
            },
        },
    }
}

func (suite *O1AdaptorTestSuite) TestApplyConfiguration() {
    ctx := context.Background()
    
    // Setup mock expectations
    suite.mockClient.On("IsConnected").Return(true)
    suite.mockClient.On("Lock", "running").Return(nil)
    suite.mockClient.On("SetConfig", mock.AnythingOfType("*ConfigData")).Return(nil)
    suite.mockClient.On("Validate", "running").Return(nil)
    suite.mockClient.On("Unlock", "running").Return(nil)
    
    // Execute test
    err := suite.adaptor.ApplyConfiguration(ctx, suite.testElement)
    
    // Assertions
    assert.NoError(suite.T(), err)
    suite.mockClient.AssertExpectations(suite.T())
}

func (suite *O1AdaptorTestSuite) TestConnectionFailure() {
    ctx := context.Background()
    
    // Setup mock expectations for connection failure
    suite.mockClient.On("IsConnected").Return(false)
    suite.mockClient.On("Connect", mock.Anything, mock.Anything).Return(errors.New("connection failed"))
    
    // Execute test
    err := suite.adaptor.ApplyConfiguration(ctx, suite.testElement)
    
    // Assertions
    assert.Error(suite.T(), err)
    assert.Contains(suite.T(), err.Error(), "failed to connect")
}

func TestO1AdaptorTestSuite(t *testing.T) {
    suite.Run(t, new(O1AdaptorTestSuite))
}
```

#### Mock Implementations

```go
// MockNetconfClient provides mock NETCONF client for testing
type MockNetconfClient struct {
    mock.Mock
}

func (m *MockNetconfClient) Connect(endpoint string, auth *AuthConfig) error {
    args := m.Called(endpoint, auth)
    return args.Error(0)
}

func (m *MockNetconfClient) IsConnected() bool {
    args := m.Called()
    return args.Bool(0)
}

func (m *MockNetconfClient) GetConfig(filter string) (*ConfigData, error) {
    args := m.Called(filter)
    return args.Get(0).(*ConfigData), args.Error(1)
}

func (m *MockNetconfClient) SetConfig(config *ConfigData) error {
    args := m.Called(config)
    return args.Error(0)
}

// Helper function to create mock client with default expectations
func NewMockNetconfClientWithDefaults() *MockNetconfClient {
    mock := &MockNetconfClient{}
    
    // Default expectations
    mock.On("IsConnected").Return(true)
    mock.On("GetCapabilities").Return([]string{
        "urn:ietf:params:netconf:base:1.0",
        "urn:o-ran:hardware:1.0",
    })
    
    return mock
}
```

### Integration Testing

#### Test Environment Setup

```go
// IntegrationTestSuite provides integration testing framework
type IntegrationTestSuite struct {
    suite.Suite
    netconfServer *NetconfTestServer
    adaptor       *O1Adaptor
    testPort      int
}

func (suite *IntegrationTestSuite) SetupSuite() {
    // Start test NETCONF server
    var err error
    suite.netconfServer, err = NewNetconfTestServer()
    suite.Require().NoError(err)
    
    suite.testPort, err = suite.netconfServer.Start()
    suite.Require().NoError(err)
    
    // Create O1 adaptor
    config := &O1Config{
        DefaultPort:    suite.testPort,
        ConnectTimeout: 10 * time.Second,
        RequestTimeout: 30 * time.Second,
        MaxRetries:     2,
    }
    
    suite.adaptor = NewO1Adaptor(config)
}

func (suite *IntegrationTestSuite) TearDownSuite() {
    if suite.netconfServer != nil {
        suite.netconfServer.Stop()
    }
}

func (suite *IntegrationTestSuite) TestRealNetconfOperations() {
    ctx := context.Background()
    
    // Create test managed element
    testElement := &nephoranv1alpha1.ManagedElement{
        ObjectMeta: metav1.ObjectMeta{
            Name: "integration-test",
        },
        Spec: nephoranv1alpha1.ManagedElementSpec{
            Host: "localhost",
            Port: suite.testPort,
            Credentials: nephoranv1alpha1.Credentials{
                Username: "test",
                Password: "test",
            },
            O1Config: `<interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
                <interface>
                    <name>eth0</name>
                    <type>ianaift:ethernetCsmacd</type>
                    <enabled>true</enabled>
                </interface>
            </interfaces>`,
        },
    }
    
    // Test connection
    err := suite.adaptor.Connect(ctx, testElement)
    suite.Assert().NoError(err)
    
    // Test configuration application
    err = suite.adaptor.ApplyConfiguration(ctx, testElement)
    suite.Assert().NoError(err)
    
    // Test configuration retrieval
    config, err := suite.adaptor.GetConfiguration(ctx, testElement)
    suite.Assert().NoError(err)
    suite.Assert().NotEmpty(config)
    
    // Test disconnect
    err = suite.adaptor.Disconnect(ctx, testElement)
    suite.Assert().NoError(err)
}
```

#### NETCONF Test Server

```go
// NetconfTestServer provides a test NETCONF server for integration testing
type NetconfTestServer struct {
    server    *ssh.Server
    yangStore *YANGDataStore
    port      int
}

func NewNetconfTestServer() (*NetconfTestServer, error) {
    yangStore := NewYANGDataStore()
    
    // Load test YANG models
    if err := yangStore.LoadModel("ietf-interfaces"); err != nil {
        return nil, fmt.Errorf("failed to load test models: %w", err)
    }
    
    return &NetconfTestServer{
        yangStore: yangStore,
    }, nil
}

func (nts *NetconfTestServer) Start() (int, error) {
    // Create SSH server configuration
    config := &ssh.ServerConfig{
        PasswordCallback: func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
            if conn.User() == "test" && string(password) == "test" {
                return nil, nil
            }
            return nil, fmt.Errorf("authentication failed")
        },
    }
    
    // Generate host key
    hostKey, err := generateHostKey()
    if err != nil {
        return 0, fmt.Errorf("failed to generate host key: %w", err)
    }
    config.AddHostKey(hostKey)
    
    // Find available port
    listener, err := net.Listen("tcp", ":0")
    if err != nil {
        return 0, fmt.Errorf("failed to listen: %w", err)
    }
    
    nts.port = listener.Addr().(*net.TCPAddr).Port
    
    // Start server
    nts.server = &ssh.Server{
        Addr:    fmt.Sprintf(":%d", nts.port),
        Handler: nts.handleSSHConnection,
    }
    
    go nts.server.Serve(listener)
    
    return nts.port, nil
}

func (nts *NetconfTestServer) handleSSHConnection(session ssh.Session) {
    // Handle NETCONF subsystem requests
    if len(session.Command()) > 0 && session.Command()[0] == "netconf" {
        nts.handleNetconfSession(session)
        return
    }
    
    session.Exit(1)
}
```

### Performance Testing

#### Benchmark Tests

```go
func BenchmarkO1Operations(b *testing.B) {
    adaptor := setupBenchmarkAdaptor()
    testElement := createTestElement()
    ctx := context.Background()
    
    b.ResetTimer()
    
    b.Run("ApplyConfiguration", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            err := adaptor.ApplyConfiguration(ctx, testElement)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
    
    b.Run("GetConfiguration", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _, err := adaptor.GetConfiguration(ctx, testElement)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
    
    b.Run("GetAlarms", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _, err := adaptor.GetAlarms(ctx, testElement)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}

func BenchmarkYANGValidation(b *testing.B) {
    registry := NewYANGModelRegistry()
    ctx := context.Background()
    
    testData := map[string]interface{}{
        "hardware": map[string]interface{}{
            "component": []interface{}{
                map[string]interface{}{
                    "name":  "cpu-1",
                    "class": "cpu",
                },
            },
        },
    }
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        err := registry.ValidateConfig(ctx, testData, "o-ran-hardware")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

#### Load Testing

```go
func TestConcurrentConnections(t *testing.T) {
    const numConnections = 100
    
    adaptor := NewO1Adaptor(nil)
    
    var wg sync.WaitGroup
    errors := make(chan error, numConnections)
    
    for i := 0; i < numConnections; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            testElement := createTestElementWithID(id)
            ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            defer cancel()
            
            if err := adaptor.Connect(ctx, testElement); err != nil {
                errors <- fmt.Errorf("connection %d failed: %w", id, err)
                return
            }
            
            if err := adaptor.ApplyConfiguration(ctx, testElement); err != nil {
                errors <- fmt.Errorf("configuration %d failed: %w", id, err)
                return
            }
            
            if err := adaptor.Disconnect(ctx, testElement); err != nil {
                errors <- fmt.Errorf("disconnect %d failed: %w", id, err)
                return
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    errorCount := 0
    for err := range errors {
        t.Error(err)
        errorCount++
    }
    
    if errorCount > 0 {
        t.Fatalf("Failed %d out of %d concurrent operations", errorCount, numConnections)
    }
}
```

## API Development Guidelines

### Interface Design Principles

#### 1. Consistency

Maintain consistent naming and parameter patterns across all operations:

```go
// Good: Consistent parameter order and naming
func (a *O1Adaptor) ApplyConfiguration(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error
func (a *O1Adaptor) GetConfiguration(ctx context.Context, me *nephoranv1alpha1.ManagedElement) (string, error)
func (a *O1Adaptor) ValidateConfiguration(ctx context.Context, config string) error

// Bad: Inconsistent parameter order
func (a *O1Adaptor) ApplyConfiguration(me *nephoranv1alpha1.ManagedElement, ctx context.Context) error
func (a *O1Adaptor) GetConfiguration(ctx context.Context, element *nephoranv1alpha1.ManagedElement) (string, error)
```

#### 2. Error Handling

Use structured error handling with context:

```go
// Good: Structured error with context
func (a *O1Adaptor) ApplyConfiguration(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error {
    if err := a.validateManagedElement(me); err != nil {
        return &O1Error{
            Code:    "INVALID_MANAGED_ELEMENT",
            Message: "Managed element validation failed",
            Cause:   err,
            Details: map[string]interface{}{
                "element_name": me.Name,
                "namespace":    me.Namespace,
            },
        }
    }
    
    // ... rest of implementation
}

// Bad: Generic error without context
func (a *O1Adaptor) ApplyConfiguration(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error {
    if err := a.validateManagedElement(me); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // ... rest of implementation
}
```

#### 3. Context Handling

Always respect context cancellation and timeouts:

```go
func (a *O1Adaptor) LongRunningOperation(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error {
    // Check context before starting
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }
    
    // Periodically check context during operation
    for i := 0; i < 100; i++ {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        // Perform work
        if err := a.performStep(i); err != nil {
            return err
        }
    }
    
    return nil
}
```

### API Versioning Strategy

#### Version Management

```go
// APIVersion represents API version information
type APIVersion struct {
    Major      int    `json:"major"`
    Minor      int    `json:"minor"`
    Patch      int    `json:"patch"`
    PreRelease string `json:"pre_release,omitempty"`
}

func (v APIVersion) String() string {
    version := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
    if v.PreRelease != "" {
        version += "-" + v.PreRelease
    }
    return version
}

// Compatibility check
func (v APIVersion) IsCompatibleWith(other APIVersion) bool {
    // Major version must match
    if v.Major != other.Major {
        return false
    }
    
    // Minor version compatibility (backward compatible)
    return v.Minor >= other.Minor
}
```

#### Deprecation Strategy

```go
// DeprecatedOperation marks operations for future removal
type DeprecatedOperation struct {
    Operation     string
    DeprecatedIn  APIVersion
    RemovedIn     APIVersion
    Replacement   string
    WarningIssued bool
}

func (a *O1Adaptor) DeprecatedMethod(ctx context.Context) error {
    // Issue deprecation warning
    if !a.deprecations["DeprecatedMethod"].WarningIssued {
        log.Warn("DeprecatedMethod is deprecated and will be removed in v2.0.0. Use NewMethod instead.")
        a.deprecations["DeprecatedMethod"].WarningIssued = true
    }
    
    // Delegate to new implementation
    return a.NewMethod(ctx)
}
```

## Contribution Guidelines

### Code Style and Standards

#### Go Code Style

Follow standard Go conventions and project-specific guidelines:

```go
// Package documentation
// Package o1 provides O1 interface implementation for O-RAN network elements.
// It implements NETCONF/YANG-based management operations including FCAPS
// functionality as specified by O-RAN Alliance specifications.
package o1

import (
    "context"
    "fmt"
    "time"
    
    // Standard library imports first
    "encoding/xml"
    "net"
    
    // Third-party imports
    "sigs.k8s.io/controller-runtime/pkg/log"
    
    // Project imports last
    nephoranv1alpha1 "github.com/nephoran/nephoran-intent-operator/api/v1"
)

// Public types should be well documented
// O1Adaptor implements the O1AdaptorInterface for managing O-RAN network elements
// through NETCONF/YANG protocols. It provides comprehensive FCAPS management
// capabilities with connection pooling and concurrent operation support.
type O1Adaptor struct {
    // Public fields should be documented
    Config *O1Config `json:"config"` // Configuration for the O1 adaptor
    
    // Private fields use camelCase
    clients        map[string]*NetconfClient
    clientsMux     sync.RWMutex
    yangRegistry   *YANGModelRegistry
}

// Public methods should be well documented with examples
// Connect establishes a NETCONF connection to the specified managed element.
// It supports both password and key-based authentication and includes
// automatic retry logic with exponential backoff.
//
// Example:
//   err := adaptor.Connect(ctx, managedElement)
//   if err != nil {
//       log.Printf("Connection failed: %v", err)
//   }
func (a *O1Adaptor) Connect(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error {
    logger := log.FromContext(ctx)
    logger.Info("establishing O1 connection", "managedElement", me.Name)
    
    // Implementation...
}
```

#### Error Handling Standards

```go
// Define custom error types for different categories
var (
    ErrConnectionFailed   = errors.New("connection failed")
    ErrAuthenticationFailed = errors.New("authentication failed")
    ErrConfigurationInvalid = errors.New("configuration invalid")
)

// Use structured errors with context
func (a *O1Adaptor) validateAndConnect(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error {
    if me.Spec.Host == "" {
        return &O1Error{
            Code:    "INVALID_HOST",
            Message: "managed element host cannot be empty",
            Details: map[string]interface{}{
                "element": me.Name,
            },
        }
    }
    
    if err := a.connect(ctx, me); err != nil {
        return fmt.Errorf("failed to connect to %s:%d: %w", me.Spec.Host, me.Spec.Port, err)
    }
    
    return nil
}
```

### Testing Requirements

#### Minimum Test Coverage

- **Unit Tests**: 80% code coverage minimum
- **Integration Tests**: All public API methods
- **End-to-End Tests**: Critical user workflows
- **Performance Tests**: Key operations benchmarked

#### Test Organization

```
pkg/oran/o1/
├── o1_adaptor.go
├── o1_adaptor_test.go           # Unit tests
├── o1_adaptor_integration_test.go # Integration tests
├── netconf_client.go
├── netconf_client_test.go
├── yang_models.go
├── yang_models_test.go
└── testdata/                    # Test data files
    ├── yang-models/
    ├── test-configs/
    └── expected-responses/
```

#### Test Naming Conventions

```go
// Test function naming: Test<FunctionName>[_<Scenario>]
func TestO1Adaptor_Connect_Success(t *testing.T) {}
func TestO1Adaptor_Connect_AuthenticationFailure(t *testing.T) {}
func TestO1Adaptor_Connect_NetworkTimeout(t *testing.T) {}

// Benchmark function naming: Benchmark<FunctionName>[_<Scenario>]
func BenchmarkO1Adaptor_ApplyConfiguration(b *testing.B) {}
func BenchmarkO1Adaptor_ApplyConfiguration_LargeConfig(b *testing.B) {}
```

### Pull Request Process

#### 1. Development Workflow

```bash
# 1. Create feature branch
git checkout -b feature/add-custom-yang-validation

# 2. Make changes and commit
git add .
git commit -m "feat: add custom YANG validation rules

- Add ValidationRule struct for custom rules
- Implement rule registration in YANGModelRegistry  
- Add tests for custom validation scenarios
- Update documentation

Closes #123"

# 3. Push and create pull request
git push origin feature/add-custom-yang-validation
```

#### 2. Pull Request Template

```markdown
## Description
Brief description of changes and motivation.

## Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)  
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed
- [ ] Performance testing (if applicable)

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] No breaking changes or migration guide provided

## Related Issues
Closes #123
```

#### 3. Code Review Checklist

**Reviewers should check:**
- [ ] Code follows project conventions
- [ ] Adequate test coverage
- [ ] Documentation is updated
- [ ] Error handling is appropriate
- [ ] Performance implications considered
- [ ] Security implications reviewed
- [ ] Backward compatibility maintained

### Documentation Standards

#### Code Documentation

```go
// Package-level documentation should explain the purpose and scope
// Package o1 implements the O1 interface for O-RAN network element management.
//
// The O1 interface provides FCAPS (Fault, Configuration, Accounting, Performance,
// Security) management capabilities through NETCONF/YANG protocols. It supports
// multiple concurrent connections, connection pooling, and comprehensive error
// handling.
//
// Basic usage:
//   config := &O1Config{DefaultPort: 830}
//   adaptor := NewO1Adaptor(config)
//   err := adaptor.Connect(ctx, managedElement)
//
// The package also provides YANG model registry for schema validation and
// custom model support for vendor-specific extensions.
package o1

// Interface documentation should explain the contract
// O1AdaptorInterface defines the complete O1 management interface for O-RAN
// network elements. It encompasses all FCAPS operations and follows O-RAN
// Alliance specifications for network element management.
//
// Implementations must ensure thread safety for concurrent operations and
// provide proper resource cleanup through connection management.
type O1AdaptorInterface interface {
    // Connect establishes a NETCONF session to the managed element.
    // It performs capability negotiation and maintains session state.
    //
    // The connection supports both password and public key authentication
    // based on the credentials provided in the ManagedElement specification.
    //
    // Returns an error if connection fails after all retry attempts.
    Connect(ctx context.Context, me *nephoranv1alpha1.ManagedElement) error
    
    // Additional methods...
}
```

#### README Updates

When adding new features, update relevant README sections:

```markdown
## O1 Interface Features

### FCAPS Management
- ✅ Configuration Management with YANG validation
- ✅ Fault Management with real-time alarm monitoring  
- ✅ Performance Management with metric collection
- ✅ Accounting Management with usage tracking
- ✅ Security Management with policy enforcement

### Advanced Features  
- ✅ Connection pooling and management
- ✅ Custom YANG model support
- ✅ Plugin architecture for extensions
- ✅ Comprehensive error handling
- ✅ Performance monitoring and metrics

### New in v1.2.0
- 🆕 Custom validation rules for YANG models
- 🆕 Enhanced plugin system with hot-loading
- 🆕 Improved connection recovery mechanisms
```

This comprehensive developer guide provides all the necessary information for developers to understand, extend, and contribute to the O1 interface implementation. It covers architecture patterns, extension mechanisms, testing strategies, and contribution guidelines to ensure high-quality, maintainable code.