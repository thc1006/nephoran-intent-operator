# Developer Guidelines: Microservices Architecture

## Table of Contents
- [Introduction](#introduction)
- [Architecture Principles](#architecture-principles)
- [Development Environment Setup](#development-environment-setup)
- [Microservice Development Patterns](#microservice-development-patterns)
- [Interface Design Guidelines](#interface-design-guidelines)
- [Error Handling and Resilience](#error-handling-and-resilience)
- [Testing Strategies](#testing-strategies)
- [Performance Optimization](#performance-optimization)
- [Monitoring and Observability](#monitoring-and-observability)
- [Security Best Practices](#security-best-practices)
- [Code Organization and Style](#code-organization-and-style)
- [Deployment Guidelines](#deployment-guidelines)
- [Contributing Workflows](#contributing-workflows)

## Introduction

This document provides comprehensive guidelines for developers contributing to the Nephoran Intent Operator's microservices architecture. Following these practices ensures consistency, maintainability, and optimal performance across all microservice components.

### Architecture Overview

The Nephoran Intent Operator uses a specialized microservices architecture with five core controllers:

```
Intent Processing Controller (LLM/RAG) → Resource Planning Controller → 
Manifest Generation Controller → GitOps Controller → Deployment Verification Controller
```

Each controller implements the `PhaseController` interface and follows event-driven communication patterns.

## Architecture Principles

### 1. Single Responsibility Principle

Each microservice controller has a single, well-defined responsibility:

```go
// ✅ Good: Clear single responsibility
type IntentProcessingController struct {
    // Focuses only on LLM processing and intent interpretation
    LLMClient      *llm.Client
    RAGService     *rag.OptimizedRAGService
    PromptEngine   *llm.TelecomPromptEngine
}

// ❌ Avoid: Multiple responsibilities in one controller
type MonolithicController struct {
    // Handles too many concerns
    LLMClient      *llm.Client
    ManifestGen    *manifest.Generator
    GitOps         *gitops.Manager
    Deployer       *deployment.Manager
}
```

### 2. Interface Segregation

Use focused interfaces rather than large, monolithic ones:

```go
// ✅ Good: Focused interface
type IntentProcessor interface {
    ProcessIntent(ctx context.Context, intent *nephoranv1.NetworkIntent) (*ProcessingResult, error)
    ValidateIntent(ctx context.Context, intent string) error
    EnhanceWithRAG(ctx context.Context, intent string) (map[string]interface{}, error)
}

// ✅ Good: Separate interface for different concerns
type IntentValidator interface {
    ValidateIntent(ctx context.Context, intent string) error
    GetSupportedIntentTypes() []string
}
```

### 3. Dependency Inversion

Controllers should depend on abstractions, not concrete implementations:

```go
// ✅ Good: Depend on interfaces
type ResourcePlanningController struct {
    client.Client
    MetricsCollector MetricsCollectorInterface
    CostEstimator    CostEstimatorInterface
    Optimizer        OptimizerInterface
}

// ❌ Avoid: Direct dependencies on concrete types
type ResourcePlanningController struct {
    client.Client
    PrometheusCollector *prometheus.Collector  // Concrete type
    AWSCostCalculator   *aws.CostCalculator    // Concrete type
}
```

## Development Environment Setup

### Prerequisites

```bash
# Required tools
go version  # Go 1.24+
kubectl version --client  # Kubernetes CLI
docker --version  # Docker for containerization
kubebuilder version  # Kubebuilder v3.8+

# Development dependencies
go mod tidy
make generate
make manifests
```

### Local Development Environment

```bash
# 1. Clone and setup
git clone <repository>
cd nephoran-intent-operator
make dev-setup

# 2. Start local cluster (kind/minikube)
make cluster-create

# 3. Deploy CRDs and dependencies
make install
make deploy-dependencies

# 4. Run specific controller locally
make run-intent-processor
# or
go run cmd/intent-processor/main.go
```

### Environment Variables

```bash
# Required environment variables for development
export KUBECONFIG="${HOME}/.kube/config"
export LLM_ENDPOINT="http://localhost:8080"
export LLM_API_KEY="your-api-key"
export RAG_ENDPOINT="http://localhost:8081"
export WEAVIATE_ENDPOINT="http://localhost:8082"
export LOG_LEVEL="debug"
export METRICS_BIND_ADDRESS=":8080"
export HEALTH_PROBE_BIND_ADDRESS=":8081"
```

## Microservice Development Patterns

### 1. Controller Structure Pattern

Follow this consistent structure for all microservice controllers:

```go
// Controller struct with clear separation of concerns
type SpecializedController struct {
    // Kubernetes client and core dependencies
    client.Client
    Scheme   *runtime.Scheme
    Recorder record.EventRecorder
    Logger   logr.Logger

    // Service-specific dependencies (interfaces preferred)
    ExternalService ExternalServiceInterface
    
    // Configuration
    Config ControllerConfig
    
    // Internal state management
    activeProcessing sync.Map
    metrics         *ControllerMetrics
    cache          *ControllerCache
    
    // Health and lifecycle management
    started      bool
    stopChan     chan struct{}
    healthStatus interfaces.HealthStatus
    mutex        sync.RWMutex
}
```

### 2. Configuration Pattern

Use structured configuration with validation:

```go
type ControllerConfig struct {
    // External service configuration
    ServiceEndpoint   string        `json:"serviceEndpoint" validate:"required,url"`
    ServiceTimeout    time.Duration `json:"serviceTimeout" validate:"min=1s,max=300s"`
    
    // Processing configuration  
    MaxRetries        int           `json:"maxRetries" validate:"min=0,max=10"`
    BatchSize         int           `json:"batchSize" validate:"min=1,max=1000"`
    
    // Feature flags
    CacheEnabled      bool          `json:"cacheEnabled"`
    MetricsEnabled    bool          `json:"metricsEnabled"`
    
    // Circuit breaker configuration
    CircuitBreakerConfig CircuitBreakerConfig `json:"circuitBreaker"`
}

// Validation method
func (c *ControllerConfig) Validate() error {
    validate := validator.New()
    return validate.Struct(c)
}
```

### 3. Processing Session Pattern

Track processing state using session objects:

```go
type ProcessingSession struct {
    // Identity and correlation
    SessionID     string    `json:"sessionId"`
    CorrelationID string    `json:"correlationId"`
    StartTime     time.Time `json:"startTime"`
    
    // Progress tracking
    Status        string    `json:"status"` // pending, processing, completed, failed
    Progress      float64   `json:"progress"` // 0.0 to 1.0
    CurrentStep   string    `json:"currentStep"`
    
    // Results and context
    Input         interface{}            `json:"input,omitempty"`
    Output        interface{}            `json:"output,omitempty"`
    Context       map[string]interface{} `json:"context,omitempty"`
    
    // Error handling
    Errors        []string               `json:"errors,omitempty"`
    RetryCount    int                   `json:"retryCount"`
    LastError     string                `json:"lastError,omitempty"`
    
    // Performance metrics
    Metrics       SessionMetrics        `json:"metrics"`
    
    // Thread safety
    mutex         sync.RWMutex
}
```

### 4. Event-Driven Communication Pattern

Use events for loose coupling between controllers:

```go
// Event definition
type ProcessingEvent struct {
    EventID       string                 `json:"eventId"`
    EventType     string                 `json:"eventType"`
    Source        string                 `json:"source"`
    Timestamp     time.Time              `json:"timestamp"`
    CorrelationID string                 `json:"correlationId"`
    Data          map[string]interface{} `json:"data"`
    Metadata      map[string]string      `json:"metadata,omitempty"`
}

// Event publisher interface
type EventPublisher interface {
    PublishEvent(ctx context.Context, event ProcessingEvent) error
    PublishEventAsync(ctx context.Context, event ProcessingEvent) error
}

// Event handler interface
type EventHandler interface {
    HandleEvent(ctx context.Context, event ProcessingEvent) error
    GetHandledEventTypes() []string
}

// Usage in controller
func (c *Controller) processPhase(ctx context.Context, input InputType) error {
    // Process phase
    result, err := c.doProcessing(ctx, input)
    if err != nil {
        return err
    }
    
    // Publish completion event
    event := ProcessingEvent{
        EventID:   generateEventID(),
        EventType: "PhaseCompleted",
        Source:    c.controllerName,
        Timestamp: time.Now(),
        Data: map[string]interface{}{
            "result": result,
            "phase":  c.phaseName,
        },
    }
    
    return c.eventPublisher.PublishEvent(ctx, event)
}
```

## Interface Design Guidelines

### 1. Core Interfaces

All controllers must implement the `PhaseController` interface:

```go
type PhaseController interface {
    // Core processing - REQUIRED
    ProcessPhase(ctx context.Context, intent *nephoranv1.NetworkIntent, phase ProcessingPhase) (ProcessingResult, error)
    
    // Status and error handling - REQUIRED  
    GetPhaseStatus(ctx context.Context, intentID string) (*PhaseStatus, error)
    HandlePhaseError(ctx context.Context, intentID string, err error) error
    
    // Dependencies - REQUIRED
    GetDependencies() []ProcessingPhase
    GetBlockedPhases() []ProcessingPhase
    
    // Lifecycle - REQUIRED
    SetupWithManager(mgr ctrl.Manager) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    
    // Health and observability - REQUIRED
    GetHealthStatus(ctx context.Context) (HealthStatus, error)
    GetMetrics(ctx context.Context) (map[string]float64, error)
}
```

### 2. Specialized Interfaces

Create focused interfaces for specific capabilities:

```go
// For controllers that support caching
type Cacheable interface {
    GetCacheKey(input interface{}) string
    GetFromCache(ctx context.Context, key string) (interface{}, bool)
    PutInCache(ctx context.Context, key string, value interface{}) error
    InvalidateCache(ctx context.Context, pattern string) error
}

// For controllers that support batching
type BatchProcessor interface {
    ProcessBatch(ctx context.Context, inputs []interface{}) ([]ProcessingResult, error)
    GetOptimalBatchSize() int
}

// For controllers with streaming capabilities
type StreamProcessor interface {
    StartStream(ctx context.Context) (<-chan ProcessingResult, <-chan error)
    ProcessStream(ctx context.Context, input <-chan interface{}) (<-chan ProcessingResult, <-chan error)
    StopStream(ctx context.Context) error
}
```

### 3. Error Interface Design

Use structured error types:

```go
// Base error interface
type ProcessingError interface {
    error
    GetErrorCode() string
    GetErrorCategory() ErrorCategory
    IsRetryable() bool
    GetContext() map[string]interface{}
}

// Error categories
type ErrorCategory string

const (
    ErrorCategoryValidation   ErrorCategory = "validation"
    ErrorCategoryExternal     ErrorCategory = "external_service"
    ErrorCategoryInternal     ErrorCategory = "internal"
    ErrorCategoryTimeout      ErrorCategory = "timeout"
    ErrorCategoryRateLimit    ErrorCategory = "rate_limit"
)

// Concrete error implementation
type StandardProcessingError struct {
    Code       string                 `json:"code"`
    Message    string                 `json:"message"`
    Category   ErrorCategory          `json:"category"`
    Retryable  bool                  `json:"retryable"`
    Context    map[string]interface{} `json:"context,omitempty"`
    Cause      error                 `json:"-"`
}

func (e *StandardProcessingError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Cause)
    }
    return e.Message
}

func (e *StandardProcessingError) GetErrorCode() string {
    return e.Code
}

func (e *StandardProcessingError) GetErrorCategory() ErrorCategory {
    return e.Category
}

func (e *StandardProcessingError) IsRetryable() bool {
    return e.Retryable
}

func (e *StandardProcessingError) GetContext() map[string]interface{} {
    return e.Context
}
```

## Error Handling and Resilience

### 1. Circuit Breaker Pattern

Implement circuit breakers for external service calls:

```go
type CircuitBreaker struct {
    name           string
    state          CircuitBreakerState
    failureCount   int64
    successCount   int64
    failureThreshold int
    recoveryTimeout  time.Duration
    lastFailureTime  time.Time
    mutex          sync.RWMutex
    logger         logr.Logger
}

type CircuitBreakerState string

const (
    CircuitBreakerClosed   CircuitBreakerState = "closed"
    CircuitBreakerOpen     CircuitBreakerState = "open"
    CircuitBreakerHalfOpen CircuitBreakerState = "half_open"
)

func (cb *CircuitBreaker) Execute(operation func() (interface{}, error)) (interface{}, error) {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()

    if cb.state == CircuitBreakerOpen {
        if time.Since(cb.lastFailureTime) < cb.recoveryTimeout {
            return nil, fmt.Errorf("circuit breaker is open")
        }
        cb.state = CircuitBreakerHalfOpen
        cb.successCount = 0
    }

    result, err := operation()
    
    if err != nil {
        cb.recordFailure()
        return nil, err
    }
    
    cb.recordSuccess()
    return result, nil
}

func (cb *CircuitBreaker) recordFailure() {
    cb.failureCount++
    cb.lastFailureTime = time.Now()
    
    if cb.state == CircuitBreakerHalfOpen || cb.failureCount >= int64(cb.failureThreshold) {
        cb.state = CircuitBreakerOpen
        cb.logger.Info("Circuit breaker opened", "name", cb.name, "failures", cb.failureCount)
    }
}

func (cb *CircuitBreaker) recordSuccess() {
    cb.successCount++
    cb.failureCount = 0
    
    if cb.state == CircuitBreakerHalfOpen && cb.successCount >= 3 {
        cb.state = CircuitBreakerClosed
        cb.logger.Info("Circuit breaker closed", "name", cb.name)
    }
}
```

### 2. Retry Pattern with Exponential Backoff

```go
type RetryConfig struct {
    MaxRetries      int           `json:"maxRetries"`
    InitialDelay    time.Duration `json:"initialDelay"`
    MaxDelay        time.Duration `json:"maxDelay"`
    BackoffFactor   float64       `json:"backoffFactor"`
    Jitter          bool          `json:"jitter"`
    RetryableErrors []string      `json:"retryableErrors"`
}

func RetryWithExponentialBackoff(ctx context.Context, config RetryConfig, operation func() error) error {
    var lastErr error
    delay := config.InitialDelay

    for attempt := 0; attempt <= config.MaxRetries; attempt++ {
        if attempt > 0 {
            select {
            case <-time.After(delay):
            case <-ctx.Done():
                return ctx.Err()
            }
        }

        if err := operation(); err != nil {
            lastErr = err
            
            // Check if error is retryable
            if !isRetryableError(err, config.RetryableErrors) {
                return err
            }
            
            // Calculate next delay with exponential backoff
            delay = time.Duration(float64(delay) * config.BackoffFactor)
            if delay > config.MaxDelay {
                delay = config.MaxDelay
            }
            
            // Add jitter to prevent thundering herd
            if config.Jitter {
                jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)
                delay += jitter
            }
            
            continue
        }
        
        return nil // Success
    }
    
    return fmt.Errorf("operation failed after %d retries: %w", config.MaxRetries, lastErr)
}

func isRetryableError(err error, retryableCodes []string) bool {
    if processingErr, ok := err.(ProcessingError); ok {
        return processingErr.IsRetryable()
    }
    
    // Check against specific error codes
    for _, code := range retryableCodes {
        if strings.Contains(err.Error(), code) {
            return true
        }
    }
    
    return false
}
```

### 3. Timeout and Context Management

```go
func (c *Controller) processWithTimeout(ctx context.Context, input interface{}) error {
    // Create context with timeout
    processCtx, cancel := context.WithTimeout(ctx, c.config.ProcessingTimeout)
    defer cancel()
    
    // Create result channel
    resultChan := make(chan error, 1)
    
    // Start processing in goroutine
    go func() {
        defer close(resultChan)
        resultChan <- c.doProcessing(processCtx, input)
    }()
    
    // Wait for completion or timeout
    select {
    case err := <-resultChan:
        return err
        
    case <-processCtx.Done():
        return &StandardProcessingError{
            Code:      "PROCESSING_TIMEOUT",
            Message:   "processing timed out",
            Category:  ErrorCategoryTimeout,
            Retryable: true,
            Context: map[string]interface{}{
                "timeout": c.config.ProcessingTimeout.String(),
            },
        }
    }
}
```

## Testing Strategies

### 1. Unit Testing Pattern

```go
func TestController_ProcessPhase(t *testing.T) {
    tests := []struct {
        name           string
        intent         *nephoranv1.NetworkIntent
        phase          interfaces.ProcessingPhase
        setupMocks     func(*MockExternalService)
        expectedResult interfaces.ProcessingResult
        expectedError  string
    }{
        {
            name: "successful processing",
            intent: &nephoranv1.NetworkIntent{
                ObjectMeta: metav1.ObjectMeta{Name: "test-intent"},
                Spec: nephoranv1.NetworkIntentSpec{
                    Intent: "Deploy high-availability 5G AMF",
                    IntentType: "5g-deployment",
                },
            },
            phase: interfaces.PhaseLLMProcessing,
            setupMocks: func(mockService *MockExternalService) {
                mockService.EXPECT().
                    ProcessIntent(gomock.Any(), gomock.Any()).
                    Return(map[string]interface{}{
                        "network_functions": []string{"amf"},
                        "deployment_pattern": "high-availability",
                        "confidence": 0.95,
                    }, nil)
            },
            expectedResult: interfaces.ProcessingResult{
                Success:   true,
                NextPhase: interfaces.PhaseResourcePlanning,
                Data: map[string]interface{}{
                    "llmResponse": map[string]interface{}{
                        "network_functions": []string{"amf"},
                        "deployment_pattern": "high-availability",
                        "confidence": 0.95,
                    },
                },
            },
        },
        {
            name: "processing error",
            intent: &nephoranv1.NetworkIntent{
                ObjectMeta: metav1.ObjectMeta{Name: "test-intent"},
                Spec: nephoranv1.NetworkIntentSpec{
                    Intent: "Invalid intent",
                },
            },
            phase: interfaces.PhaseLLMProcessing,
            setupMocks: func(mockService *MockExternalService) {
                mockService.EXPECT().
                    ProcessIntent(gomock.Any(), gomock.Any()).
                    Return(nil, errors.New("invalid intent format"))
            },
            expectedError: "invalid intent format",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()

            mockService := NewMockExternalService(ctrl)
            if tt.setupMocks != nil {
                tt.setupMocks(mockService)
            }

            controller := &Controller{
                Client:          fake.NewClientBuilder().Build(),
                ExternalService: mockService,
                Logger:          logr.Discard(),
            }

            // Execute
            result, err := controller.ProcessPhase(context.Background(), tt.intent, tt.phase)

            // Verify
            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expectedResult.Success, result.Success)
                assert.Equal(t, tt.expectedResult.NextPhase, result.NextPhase)
            }
        })
    }
}
```

### 2. Integration Testing Pattern

```go
func TestController_Integration(t *testing.T) {
    // Setup test environment
    testEnv := &envtest.Environment{
        CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
    }

    cfg, err := testEnv.Start()
    require.NoError(t, err)
    defer testEnv.Stop()

    // Create client
    k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
    require.NoError(t, err)

    // Setup controller with real dependencies
    controller := &Controller{
        Client: k8sClient,
        Logger: logr.Discard(),
    }

    // Create test intent
    intent := &nephoranv1.NetworkIntent{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-intent",
            Namespace: "default",
        },
        Spec: nephoranv1.NetworkIntentSpec{
            Intent: "Deploy 5G core network",
            IntentType: "5g-deployment",
        },
    }

    // Create intent in cluster
    err = k8sClient.Create(context.Background(), intent)
    require.NoError(t, err)

    // Test processing
    result, err := controller.ProcessPhase(context.Background(), intent, interfaces.PhaseLLMProcessing)
    assert.NoError(t, err)
    assert.True(t, result.Success)

    // Verify intent status was updated
    var updatedIntent nephoranv1.NetworkIntent
    err = k8sClient.Get(context.Background(), 
        client.ObjectKeyFromObject(intent), &updatedIntent)
    require.NoError(t, err)
    
    assert.Equal(t, interfaces.PhaseResourcePlanning, updatedIntent.Status.ProcessingPhase)
}
```

### 3. Performance Testing Pattern

```go
func BenchmarkController_ProcessPhase(b *testing.B) {
    // Setup controller
    controller := setupTestController()
    
    intent := &nephoranv1.NetworkIntent{
        ObjectMeta: metav1.ObjectMeta{Name: "benchmark-intent"},
        Spec: nephoranv1.NetworkIntentSpec{
            Intent: "Deploy network function",
        },
    }

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := controller.ProcessPhase(
                context.Background(), 
                intent, 
                interfaces.PhaseLLMProcessing,
            )
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}

func TestController_LoadTest(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load test in short mode")
    }

    controller := setupTestController()
    
    // Test concurrent processing
    const numGoroutines = 100
    const requestsPerGoroutine = 10
    
    var wg sync.WaitGroup
    errors := make(chan error, numGoroutines*requestsPerGoroutine)
    
    start := time.Now()
    
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            
            for j := 0; j < requestsPerGoroutine; j++ {
                intent := &nephoranv1.NetworkIntent{
                    ObjectMeta: metav1.ObjectMeta{
                        Name: fmt.Sprintf("load-test-intent-%d-%d", id, j),
                    },
                    Spec: nephoranv1.NetworkIntentSpec{
                        Intent: "Deploy network function",
                    },
                }
                
                _, err := controller.ProcessPhase(
                    context.Background(),
                    intent,
                    interfaces.PhaseLLMProcessing,
                )
                
                if err != nil {
                    errors <- err
                }
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    duration := time.Since(start)
    totalRequests := numGoroutines * requestsPerGoroutine
    
    errorCount := 0
    for err := range errors {
        errorCount++
        t.Logf("Error: %v", err)
    }
    
    successRate := float64(totalRequests-errorCount) / float64(totalRequests) * 100
    throughput := float64(totalRequests) / duration.Seconds()
    
    t.Logf("Load test results:")
    t.Logf("  Total requests: %d", totalRequests)
    t.Logf("  Success rate: %.2f%%", successRate)
    t.Logf("  Throughput: %.2f req/sec", throughput)
    t.Logf("  Duration: %v", duration)
    
    assert.Greater(t, successRate, 95.0, "Success rate should be above 95%")
    assert.Greater(t, throughput, 50.0, "Throughput should be above 50 req/sec")
}
```

## Performance Optimization

### 1. Connection Pooling

```go
type ConnectionPool struct {
    mu      sync.RWMutex
    clients map[string]*PooledClient
    config  PoolConfig
}

type PoolConfig struct {
    MaxIdleConns    int           `json:"maxIdleConns"`
    MaxOpenConns    int           `json:"maxOpenConns"`
    ConnMaxLifetime time.Duration `json:"connMaxLifetime"`
    ConnMaxIdleTime time.Duration `json:"connMaxIdleTime"`
}

type PooledClient struct {
    client    interface{}
    lastUsed  time.Time
    inUse     bool
    created   time.Time
}

func (p *ConnectionPool) GetClient(ctx context.Context, endpoint string) (interface{}, error) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    // Check for available client
    if client, exists := p.clients[endpoint]; exists {
        if !client.inUse && time.Since(client.lastUsed) < p.config.ConnMaxIdleTime {
            client.inUse = true
            client.lastUsed = time.Now()
            return client.client, nil
        }
    }
    
    // Create new client if pool not full
    if len(p.clients) < p.config.MaxOpenConns {
        newClient, err := p.createClient(ctx, endpoint)
        if err != nil {
            return nil, err
        }
        
        p.clients[endpoint] = &PooledClient{
            client:   newClient,
            lastUsed: time.Now(),
            inUse:    true,
            created:  time.Now(),
        }
        
        return newClient, nil
    }
    
    return nil, fmt.Errorf("connection pool exhausted")
}

func (p *ConnectionPool) ReturnClient(endpoint string) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    if client, exists := p.clients[endpoint]; exists {
        client.inUse = false
        client.lastUsed = time.Now()
    }
}
```

### 2. Caching Strategy

```go
type MultiLevelCache struct {
    l1Cache    *sync.Map              // In-memory cache
    l2Cache    redis.UniversalClient  // Redis cache
    l3Cache    storage.Interface      // Persistent storage
    config     CacheConfig
    metrics    *CacheMetrics
}

type CacheConfig struct {
    L1TTL         time.Duration `json:"l1Ttl"`
    L2TTL         time.Duration `json:"l2Ttl"`
    L3TTL         time.Duration `json:"l3Ttl"`
    MaxL1Entries  int          `json:"maxL1Entries"`
    SerializeJSON bool         `json:"serializeJson"`
}

func (c *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, bool) {
    // Try L1 cache (in-memory)
    if value, ok := c.l1Cache.Load(key); ok {
        c.metrics.RecordHit("l1")
        return value, true
    }
    
    // Try L2 cache (Redis)
    if c.l2Cache != nil {
        if value, err := c.l2Cache.Get(ctx, key).Result(); err == nil {
            c.metrics.RecordHit("l2")
            
            // Promote to L1
            c.l1Cache.Store(key, value)
            
            return value, true
        }
    }
    
    // Try L3 cache (persistent storage)
    if c.l3Cache != nil {
        if value, err := c.l3Cache.Get(ctx, key); err == nil {
            c.metrics.RecordHit("l3")
            
            // Promote to L2 and L1
            if c.l2Cache != nil {
                c.l2Cache.Set(ctx, key, value, c.config.L2TTL)
            }
            c.l1Cache.Store(key, value)
            
            return value, true
        }
    }
    
    c.metrics.RecordMiss()
    return nil, false
}

func (c *MultiLevelCache) Set(ctx context.Context, key string, value interface{}) error {
    // Set in all levels
    c.l1Cache.Store(key, value)
    
    if c.l2Cache != nil {
        if err := c.l2Cache.Set(ctx, key, value, c.config.L2TTL).Err(); err != nil {
            return fmt.Errorf("failed to set L2 cache: %w", err)
        }
    }
    
    if c.l3Cache != nil {
        if err := c.l3Cache.Set(ctx, key, value); err != nil {
            return fmt.Errorf("failed to set L3 cache: %w", err)
        }
    }
    
    return nil
}
```

### 3. Batch Processing

```go
type BatchProcessor struct {
    batchSize     int
    flushInterval time.Duration
    processor     func([]interface{}) error
    buffer        []interface{}
    mutex         sync.Mutex
    stopChan      chan struct{}
    flushChan     chan struct{}
}

func NewBatchProcessor(batchSize int, flushInterval time.Duration, processor func([]interface{}) error) *BatchProcessor {
    bp := &BatchProcessor{
        batchSize:     batchSize,
        flushInterval: flushInterval,
        processor:     processor,
        buffer:        make([]interface{}, 0, batchSize),
        stopChan:      make(chan struct{}),
        flushChan:     make(chan struct{}, 1),
    }
    
    go bp.flushWorker()
    return bp
}

func (bp *BatchProcessor) Add(item interface{}) error {
    bp.mutex.Lock()
    defer bp.mutex.Unlock()
    
    bp.buffer = append(bp.buffer, item)
    
    if len(bp.buffer) >= bp.batchSize {
        return bp.flush()
    }
    
    return nil
}

func (bp *BatchProcessor) flush() error {
    if len(bp.buffer) == 0 {
        return nil
    }
    
    batch := make([]interface{}, len(bp.buffer))
    copy(batch, bp.buffer)
    bp.buffer = bp.buffer[:0] // Reset buffer
    
    return bp.processor(batch)
}

func (bp *BatchProcessor) flushWorker() {
    ticker := time.NewTicker(bp.flushInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            bp.mutex.Lock()
            bp.flush()
            bp.mutex.Unlock()
            
        case <-bp.flushChan:
            bp.mutex.Lock()
            bp.flush()
            bp.mutex.Unlock()
            
        case <-bp.stopChan:
            return
        }
    }
}

func (bp *BatchProcessor) Close() error {
    close(bp.stopChan)
    
    bp.mutex.Lock()
    defer bp.mutex.Unlock()
    
    return bp.flush()
}
```

## Monitoring and Observability

### 1. Metrics Collection

```go
type ControllerMetrics struct {
    // Request metrics
    RequestsTotal     prometheus.CounterVec
    RequestDuration   prometheus.HistogramVec
    RequestsInFlight  prometheus.Gauge
    
    // Processing metrics
    ProcessingTotal   prometheus.CounterVec
    ProcessingLatency prometheus.HistogramVec
    
    // Error metrics
    ErrorsTotal       prometheus.CounterVec
    
    // Resource metrics
    ActiveSessions    prometheus.Gauge
    CacheHitRate     prometheus.GaugeVec
    
    // Custom business metrics
    IntentsByType     prometheus.CounterVec
    ConfidenceScore   prometheus.HistogramVec
}

func NewControllerMetrics(controllerName string) *ControllerMetrics {
    labels := []string{"controller", "phase", "status"}
    
    return &ControllerMetrics{
        RequestsTotal: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "controller_requests_total",
                Help: "Total number of requests processed by controller",
            },
            labels,
        ),
        RequestDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "controller_request_duration_seconds",
                Help:    "Request duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            labels,
        ),
        // ... initialize other metrics
    }
}

// Usage in controller
func (c *Controller) ProcessPhase(ctx context.Context, intent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase) (interfaces.ProcessingResult, error) {
    start := time.Now()
    
    // Increment in-flight requests
    c.metrics.RequestsInFlight.Inc()
    defer c.metrics.RequestsInFlight.Dec()
    
    // Process phase
    result, err := c.doProcessPhase(ctx, intent, phase)
    
    // Record metrics
    status := "success"
    if err != nil {
        status = "error"
        c.metrics.ErrorsTotal.WithLabelValues(c.name, string(phase), err.Error()).Inc()
    }
    
    c.metrics.RequestsTotal.WithLabelValues(c.name, string(phase), status).Inc()
    c.metrics.RequestDuration.WithLabelValues(c.name, string(phase), status).Observe(time.Since(start).Seconds())
    
    if result.Success {
        if confidence, ok := result.Metrics["confidence"]; ok {
            c.metrics.ConfidenceScore.WithLabelValues(c.name, string(phase)).Observe(confidence)
        }
    }
    
    return result, err
}
```

### 2. Distributed Tracing

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

func (c *Controller) ProcessPhase(ctx context.Context, intent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase) (interfaces.ProcessingResult, error) {
    // Create span for this operation
    tracer := otel.Tracer("nephoran-controller")
    ctx, span := tracer.Start(ctx, fmt.Sprintf("%s.ProcessPhase", c.name),
        trace.WithAttributes(
            attribute.String("controller.name", c.name),
            attribute.String("phase", string(phase)),
            attribute.String("intent.id", intent.Name),
            attribute.String("intent.type", intent.Spec.IntentType),
        ),
    )
    defer span.End()
    
    // Add correlation ID to span
    if correlationID := getCorrelationID(ctx); correlationID != "" {
        span.SetAttributes(attribute.String("correlation.id", correlationID))
    }
    
    // Process phase with tracing context
    result, err := c.doProcessPhase(ctx, intent, phase)
    
    if err != nil {
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
    } else {
        span.SetStatus(codes.Ok, "processing completed successfully")
        
        // Add result attributes
        if result.Data != nil {
            if confidence, ok := result.Data["confidence"].(float64); ok {
                span.SetAttributes(attribute.Float64("result.confidence", confidence))
            }
        }
    }
    
    return result, err
}

// Helper function to propagate correlation ID
func withCorrelationID(ctx context.Context, correlationID string) context.Context {
    return context.WithValue(ctx, "correlation_id", correlationID)
}

func getCorrelationID(ctx context.Context) string {
    if id := ctx.Value("correlation_id"); id != nil {
        return id.(string)
    }
    return ""
}
```

### 3. Health Checks

```go
type HealthChecker struct {
    checks   map[string]HealthCheck
    mutex    sync.RWMutex
    timeout  time.Duration
}

type HealthCheck interface {
    Name() string
    Check(ctx context.Context) HealthResult
}

type HealthResult struct {
    Status    HealthStatus               `json:"status"`
    Message   string                     `json:"message,omitempty"`
    Details   map[string]interface{}     `json:"details,omitempty"`
    Timestamp time.Time                  `json:"timestamp"`
    Duration  time.Duration             `json:"duration"`
}

type HealthStatus string

const (
    HealthStatusHealthy   HealthStatus = "healthy"
    HealthStatusDegraded  HealthStatus = "degraded"
    HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// External service health check
type ExternalServiceHealthCheck struct {
    name     string
    endpoint string
    client   *http.Client
}

func (h *ExternalServiceHealthCheck) Name() string {
    return h.name
}

func (h *ExternalServiceHealthCheck) Check(ctx context.Context) HealthResult {
    start := time.Now()
    
    req, err := http.NewRequestWithContext(ctx, "GET", h.endpoint+"/health", nil)
    if err != nil {
        return HealthResult{
            Status:    HealthStatusUnhealthy,
            Message:   fmt.Sprintf("failed to create request: %v", err),
            Timestamp: time.Now(),
            Duration:  time.Since(start),
        }
    }
    
    resp, err := h.client.Do(req)
    if err != nil {
        return HealthResult{
            Status:    HealthStatusUnhealthy,
            Message:   fmt.Sprintf("request failed: %v", err),
            Timestamp: time.Now(),
            Duration:  time.Since(start),
        }
    }
    defer resp.Body.Close()
    
    status := HealthStatusHealthy
    message := "service is healthy"
    
    if resp.StatusCode >= 500 {
        status = HealthStatusUnhealthy
        message = "service returned 5xx status"
    } else if resp.StatusCode >= 400 {
        status = HealthStatusDegraded
        message = "service returned 4xx status"
    }
    
    return HealthResult{
        Status:    status,
        Message:   message,
        Timestamp: time.Now(),
        Duration:  time.Since(start),
        Details: map[string]interface{}{
            "status_code": resp.StatusCode,
            "endpoint":    h.endpoint,
        },
    }
}

// Usage in controller
func (c *Controller) GetHealthStatus(ctx context.Context) (interfaces.HealthStatus, error) {
    checker := &HealthChecker{
        checks:  make(map[string]HealthCheck),
        timeout: 10 * time.Second,
    }
    
    // Add health checks
    checker.AddCheck(&ExternalServiceHealthCheck{
        name:     "llm-service",
        endpoint: c.config.LLMEndpoint,
        client:   &http.Client{Timeout: 5 * time.Second},
    })
    
    checker.AddCheck(&ExternalServiceHealthCheck{
        name:     "rag-service",
        endpoint: c.config.RAGEndpoint,
        client:   &http.Client{Timeout: 5 * time.Second},
    })
    
    // Perform health check
    overallStatus, results := checker.CheckAll(ctx)
    
    return interfaces.HealthStatus{
        Status:      string(overallStatus),
        Message:     fmt.Sprintf("controller health: %s", overallStatus),
        LastChecked: time.Now(),
        Metrics: map[string]interface{}{
            "health_checks": results,
        },
    }, nil
}
```

## Security Best Practices

### 1. Input Validation

```go
import "github.com/go-playground/validator/v10"

type IntentValidator struct {
    validator *validator.Validate
}

func NewIntentValidator() *IntentValidator {
    v := validator.New()
    
    // Register custom validators
    v.RegisterValidation("telecom_intent", validateTelecomIntent)
    v.RegisterValidation("safe_text", validateSafeText)
    
    return &IntentValidator{validator: v}
}

type ValidatedIntent struct {
    Intent     string `json:"intent" validate:"required,min=10,max=10000,safe_text,telecom_intent"`
    IntentType string `json:"intentType" validate:"required,oneof=5g-deployment network-slice cnf-deployment monitoring-setup security-config"`
    Priority   int    `json:"priority" validate:"min=1,max=10"`
    Metadata   map[string]string `json:"metadata" validate:"dive,keys,required,endkeys,required"`
}

func validateTelecomIntent(fl validator.FieldLevel) bool {
    intent := fl.Field().String()
    
    // Check for telecom-related keywords
    telecomKeywords := []string{
        "5g", "network", "deployment", "slice", "cnf", "vnf",
        "amf", "smf", "upf", "monitoring", "security",
    }
    
    intentLower := strings.ToLower(intent)
    for _, keyword := range telecomKeywords {
        if strings.Contains(intentLower, keyword) {
            return true
        }
    }
    
    return false
}

func validateSafeText(fl validator.FieldLevel) bool {
    text := fl.Field().String()
    
    // Block potentially malicious patterns
    dangerousPatterns := []string{
        "<script", "javascript:", "onload=", "onerror=",
        "eval(", "exec(", "system(", "shell_exec(",
        "../", "..\\", "/etc/passwd", "cmd.exe",
    }
    
    textLower := strings.ToLower(text)
    for _, pattern := range dangerousPatterns {
        if strings.Contains(textLower, pattern) {
            return false
        }
    }
    
    return true
}

func (v *IntentValidator) ValidateIntent(intent *nephoranv1.NetworkIntent) error {
    validatedIntent := ValidatedIntent{
        Intent:     intent.Spec.Intent,
        IntentType: intent.Spec.IntentType,
        Priority:   intent.Spec.Priority,
        Metadata:   intent.Spec.Metadata,
    }
    
    if err := v.validator.Struct(validatedIntent); err != nil {
        return fmt.Errorf("intent validation failed: %w", err)
    }
    
    return nil
}
```

### 2. Secret Management

```go
type SecretManager struct {
    client    client.Client
    namespace string
    cache     map[string]*SecretCacheEntry
    mutex     sync.RWMutex
    ttl       time.Duration
}

type SecretCacheEntry struct {
    data      map[string][]byte
    timestamp time.Time
}

func (sm *SecretManager) GetSecret(ctx context.Context, secretName, key string) ([]byte, error) {
    sm.mutex.RLock()
    
    // Check cache first
    if entry, exists := sm.cache[secretName]; exists {
        if time.Since(entry.timestamp) < sm.ttl {
            sm.mutex.RUnlock()
            if value, ok := entry.data[key]; ok {
                return value, nil
            }
            return nil, fmt.Errorf("key %s not found in secret %s", key, secretName)
        }
        // Cache expired
        delete(sm.cache, secretName)
    }
    
    sm.mutex.RUnlock()
    
    // Fetch from Kubernetes
    var secret corev1.Secret
    err := sm.client.Get(ctx, types.NamespacedName{
        Name:      secretName,
        Namespace: sm.namespace,
    }, &secret)
    
    if err != nil {
        return nil, fmt.Errorf("failed to get secret %s: %w", secretName, err)
    }
    
    // Update cache
    sm.mutex.Lock()
    sm.cache[secretName] = &SecretCacheEntry{
        data:      secret.Data,
        timestamp: time.Now(),
    }
    sm.mutex.Unlock()
    
    if value, ok := secret.Data[key]; ok {
        return value, nil
    }
    
    return nil, fmt.Errorf("key %s not found in secret %s", key, secretName)
}

func (sm *SecretManager) GetSecretString(ctx context.Context, secretName, key string) (string, error) {
    data, err := sm.GetSecret(ctx, secretName, key)
    if err != nil {
        return "", err
    }
    return string(data), nil
}
```

### 3. RBAC Configuration

```yaml
# Example RBAC configuration for microservice controllers
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: nephoran-intent-processing-controller
rules:
- apiGroups: [""]
  resources: ["events"]
  verbs: ["create", "patch"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list"]
- apiGroups: ["nephoran.io"]
  resources: ["networkintents"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["nephoran.io"]
  resources: ["networkintents/status"]
  verbs: ["get", "update", "patch"]
- apiGroups: ["nephoran.io"]
  resources: ["networkintents/finalizers"]
  verbs: ["update"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: nephoran-intent-processing-controller
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: nephoran-intent-processing-controller
subjects:
- kind: ServiceAccount
  name: nephoran-intent-processing-controller
  namespace: nephoran-system
```

## Code Organization and Style

### 1. Package Structure

```
pkg/controllers/
├── orchestration/
│   ├── coordination_controller.go
│   ├── specialized_intent_processing_controller.go
│   ├── specialized_resource_planning_controller.go
│   ├── specialized_manifest_generation_controller.go
│   ├── specialized_gitops_controller.go
│   └── specialized_deployment_verification_controller.go
├── interfaces/
│   └── controller_interfaces.go
├── common/
│   ├── metrics.go
│   ├── health.go
│   ├── cache.go
│   └── errors.go
├── llm/
│   ├── client.go
│   ├── prompt_engine.go
│   ├── streaming.go
│   └── circuit_breaker.go
├── rag/
│   ├── service.go
│   ├── query.go
│   └── vector_store.go
└── utils/
    ├── retry.go
    ├── timeout.go
    └── validation.go
```

### 2. Naming Conventions

```go
// Controller names: SpecializedXxxController
type SpecializedIntentProcessingController struct {}

// Interface names: XxxInterface or Xxxer
type IntentProcessor interface {}
type Cacheable interface {}

// Method names: Use clear, descriptive verbs
func (c *Controller) ProcessIntent() error {}
func (c *Controller) ValidateInput() error {}
func (c *Controller) HandleError() error {}

// Constants: Use descriptive names with context
const (
    DefaultTimeout = 30 * time.Second
    MaxRetryAttempts = 3
    CircuitBreakerThreshold = 5
)

// Configuration structs: Include validation tags
type ProcessingConfig struct {
    Endpoint string `json:"endpoint" validate:"required,url"`
    Timeout  time.Duration `json:"timeout" validate:"min=1s"`
}
```

### 3. Documentation Standards

```go
// Package documentation example
/*
Package orchestration provides specialized microservice controllers for the
Nephoran Intent Operator. Each controller handles a specific phase of intent
processing, following the Single Responsibility Principle and implementing
the PhaseController interface.

The controllers in this package include:
  - SpecializedIntentProcessingController: Handles LLM-based intent interpretation
  - SpecializedResourcePlanningController: Plans resource allocation and optimization
  - SpecializedManifestGenerationController: Generates Kubernetes manifests
  - SpecializedGitOpsController: Manages Git operations and deployments
  - SpecializedDeploymentVerificationController: Verifies deployment success

Each controller provides:
  - Event-driven communication through the coordination controller
  - Circuit breaker pattern for resilience
  - Comprehensive metrics and health checks
  - Structured error handling and retry logic
*/
package orchestration

// Controller documentation example
/*
SpecializedIntentProcessingController implements specialized intent processing
with LLM and RAG integration. It handles the first phase of intent processing,
converting natural language intents into structured data for subsequent phases.

Key features:
  - GPT-4o-mini integration for intent interpretation
  - RAG enhancement with telecommunications knowledge
  - Streaming processing for real-time feedback
  - Circuit breaker for LLM service resilience
  - Multi-level caching for performance optimization

The controller follows these processing steps:
  1. Validate intent text for safety and relevance
  2. Enhance intent with RAG context from vector database
  3. Process intent through LLM with specialized prompts
  4. Validate confidence threshold and extract structured results
  5. Cache results and publish completion events

Configuration:
  - LLMEndpoint: URL of the LLM service
  - RAGEndpoint: URL of the RAG service
  - StreamingEnabled: Enable streaming processing
  - CacheEnabled: Enable result caching
  - CircuitBreakerEnabled: Enable circuit breaker protection

Metrics:
  - Processing latency and throughput
  - Confidence scores and accuracy
  - Cache hit rates and performance
  - Error rates and circuit breaker state
*/
type SpecializedIntentProcessingController struct {
    // ... struct fields
}

// Method documentation example
/*
ProcessIntent processes a network intent through LLM and RAG enhancement.

This method implements the core intent processing workflow:
  1. Creates a processing session for tracking and metrics
  2. Validates the intent text for security and telecommunications relevance
  3. Enhances the intent with relevant context from RAG vector database
  4. Processes the enhanced intent through the LLM service
  5. Validates the confidence score against configured threshold
  6. Caches the result for future requests
  7. Updates metrics and publishes completion events

Parameters:
  - ctx: Context for cancellation and timeout handling
  - intent: NetworkIntent CR containing the intent text and metadata

Returns:
  - ProcessingResult: Structured result with LLM response and metadata
  - error: Processing error if validation or LLM processing fails

The method supports both streaming and non-streaming processing based on
configuration. Streaming provides real-time feedback but may have higher
latency overhead for small intents.

Example usage:
    result, err := controller.ProcessIntent(ctx, intent)
    if err != nil {
        // Handle processing error
        return err
    }
    
    if result.Success {
        // Use result.Data for next processing phase
        processNextPhase(result.Data)
    }

Error conditions:
  - ValidationError: Intent fails safety or relevance checks
  - LLMProcessingError: LLM service returns error or timeout
  - LowConfidence: Result confidence below threshold
  - CircuitBreakerOpen: LLM service circuit breaker is open
*/
func (c *SpecializedIntentProcessingController) ProcessIntent(ctx context.Context, intent *nephoranv1.NetworkIntent) (*interfaces.ProcessingResult, error) {
    // Implementation...
}
```

## Deployment Guidelines

### 1. Container Configuration

```dockerfile
# Multi-stage build for optimized container
FROM golang:1.24-alpine AS builder

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags '-w -s -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE}' \
    -o manager cmd/controller/main.go

# Final stage: minimal runtime image
FROM gcr.io/distroless/static:nonroot

# Copy CA certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy timezone data
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary
COPY --from=builder /workspace/manager .

# Set non-root user
USER 65532:65532

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
  CMD ["/manager", "healthcheck"]

# Entry point
ENTRYPOINT ["/manager"]
```

### 2. Kubernetes Manifests

```yaml
# Deployment with security best practices
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nephoran-intent-processing-controller
  namespace: nephoran-system
  labels:
    app.kubernetes.io/name: nephoran-intent-processing-controller
    app.kubernetes.io/component: controller
    app.kubernetes.io/part-of: nephoran-intent-operator
spec:
  replicas: 2
  selector:
    matchLabels:
      app.kubernetes.io/name: nephoran-intent-processing-controller
  template:
    metadata:
      labels:
        app.kubernetes.io/name: nephoran-intent-processing-controller
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: nephoran-intent-processing-controller
      securityContext:
        runAsNonRoot: true
        runAsUser: 65532
        runAsGroup: 65532
        fsGroup: 65532
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: controller
        image: nephoran/intent-processing-controller:v1.0.0
        imagePullPolicy: IfNotPresent
        args:
        - --config=/etc/config/config.yaml
        - --metrics-bind-address=:8080
        - --health-probe-bind-address=:8081
        - --leader-elect
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
        ports:
        - name: metrics
          containerPort: 8080
          protocol: TCP
        - name: health
          containerPort: 8081
          protocol: TCP
        livenessProbe:
          httpGet:
            path: /healthz
            port: health
          initialDelaySeconds: 15
          periodSeconds: 20
          timeoutSeconds: 10
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /readyz
            port: health
          initialDelaySeconds: 5
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        env:
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: LLM_API_KEY
          valueFrom:
            secretKeyRef:
              name: llm-service-secret
              key: api-key
        volumeMounts:
        - name: config
          mountPath: /etc/config
          readOnly: true
        - name: tmp
          mountPath: /tmp
      volumes:
      - name: config
        configMap:
          name: nephoran-intent-processing-config
      - name: tmp
        emptyDir: {}
      terminationGracePeriodSeconds: 30
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
      - key: "node.kubernetes.io/not-ready"
        operator: "Exists"
        effect: "NoExecute"
        tolerationSeconds: 300
      - key: "node.kubernetes.io/unreachable"
        operator: "Exists"
        effect: "NoExecute"
        tolerationSeconds: 300
---
apiVersion: v1
kind: Service
metadata:
  name: nephoran-intent-processing-controller-metrics
  namespace: nephoran-system
  labels:
    app.kubernetes.io/name: nephoran-intent-processing-controller
spec:
  selector:
    app.kubernetes.io/name: nephoran-intent-processing-controller
  ports:
  - name: metrics
    port: 8080
    targetPort: metrics
    protocol: TCP
---
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: nephoran-intent-processing-controller
  namespace: nephoran-system
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: nephoran-intent-processing-controller
```

### 3. Configuration Management

```yaml
# ConfigMap for controller configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: nephoran-intent-processing-config
  namespace: nephoran-system
data:
  config.yaml: |
    controller:
      name: "nephoran-intent-processing-controller"
      namespace: "nephoran-system"
      logLevel: "info"
      metricsEnabled: true
      healthCheckEnabled: true
      
    llm:
      endpoint: "https://api.openai.com/v1"
      model: "gpt-4o-mini"
      maxTokens: 4096
      temperature: 0.1
      timeout: "30s"
      
    rag:
      endpoint: "http://rag-service:8080"
      maxContextChunks: 5
      similarityThreshold: 0.7
      timeout: "10s"
      
    processing:
      streamingEnabled: true
      cacheEnabled: true
      cacheTTL: "1h"
      maxRetries: 3
      timeout: "60s"
      confidenceThreshold: 0.7
      
    circuitBreaker:
      enabled: true
      failureThreshold: 5
      recoveryTimeout: "30s"
      
    monitoring:
      metricsInterval: "30s"
      healthCheckInterval: "15s"
```

## Contributing Workflows

### 1. Development Workflow

```bash
# 1. Create feature branch
git checkout -b feature/improve-intent-processing

# 2. Make changes following guidelines
# - Add unit tests for new functionality
# - Update integration tests if needed
# - Add metrics and logging
# - Update documentation

# 3. Run local tests
make test
make integration-test
make lint

# 4. Run end-to-end tests
make e2e-test

# 5. Build and test container locally
make docker-build
make docker-test

# 6. Submit pull request with:
# - Clear description of changes
# - Test results
# - Performance impact analysis
# - Breaking changes documentation
```

### 2. Code Review Checklist

#### Functionality
- [ ] Code follows single responsibility principle
- [ ] Implements required PhaseController interface methods
- [ ] Handles errors appropriately with structured error types
- [ ] Includes comprehensive input validation
- [ ] Implements proper timeout and context handling

#### Performance
- [ ] Uses appropriate data structures and algorithms
- [ ] Implements caching where beneficial
- [ ] Avoids unnecessary allocations in hot paths
- [ ] Includes connection pooling for external services
- [ ] Has reasonable resource limits and requests

#### Observability
- [ ] Adds relevant metrics with appropriate labels
- [ ] Includes distributed tracing spans
- [ ] Has comprehensive logging at appropriate levels
- [ ] Implements health checks for dependencies
- [ ] Provides meaningful error messages

#### Security
- [ ] Validates all inputs against injection attacks
- [ ] Uses secrets properly without logging sensitive data
- [ ] Implements RBAC with minimal required permissions
- [ ] Follows security context best practices
- [ ] Encrypts data in transit and at rest

#### Testing
- [ ] Has unit tests with >90% coverage
- [ ] Includes integration tests for external dependencies
- [ ] Has performance benchmarks for critical paths
- [ ] Includes negative test cases and error conditions
- [ ] Tests concurrent access and race conditions

#### Documentation
- [ ] Has clear package and type documentation
- [ ] Includes usage examples in comments
- [ ] Documents configuration options and defaults
- [ ] Explains error conditions and recovery
- [ ] Updates architectural documentation if needed

### 3. Release Process

```bash
# 1. Version bump and changelog
git checkout main
git pull origin main
make version-bump VERSION=v1.2.0
git add .
git commit -m "chore: bump version to v1.2.0"

# 2. Create release branch
git checkout -b release/v1.2.0

# 3. Final testing
make test-all
make security-scan
make performance-test

# 4. Build release artifacts
make build-release VERSION=v1.2.0

# 5. Create and push release tag
git tag -a v1.2.0 -m "Release v1.2.0"
git push origin v1.2.0

# 6. Trigger automated release pipeline
# Pipeline will:
# - Build and push container images
# - Run security scans
# - Generate release notes
# - Deploy to staging environment
# - Run E2E tests
# - Create GitHub release
```

## Conclusion

These guidelines provide a comprehensive framework for developing and maintaining microservice controllers within the Nephoran Intent Operator. Following these practices ensures consistency, reliability, and maintainability across all components while supporting the system's performance and scalability requirements.

Key takeaways:
1. **Consistency**: Follow established patterns and interfaces across all controllers
2. **Resilience**: Implement circuit breakers, retries, and proper error handling
3. **Observability**: Include comprehensive metrics, logging, and tracing
4. **Security**: Validate inputs, manage secrets properly, and follow security best practices
5. **Testing**: Maintain high test coverage with unit, integration, and performance tests
6. **Documentation**: Keep code well-documented with clear examples and usage patterns

For questions or clarifications, please refer to the architectural documentation or reach out to the development team through the established communication channels.