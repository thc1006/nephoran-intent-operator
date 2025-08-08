# Generic Components for Nephoran Intent Operator

This package provides advanced Go 1.24+ generic implementations for the Nephoran Intent Operator's telecommunications network orchestration platform. All components are designed with type safety, performance, and developer experience in mind.

## Features

- **Zero Runtime Overhead**: All generic abstractions compile away to efficient machine code
- **Type Safety**: Compile-time guarantees with excellent error messages
- **Functional Programming**: Monadic Result/Option types with chainable operations
- **High Performance**: Optimized collections with memory pools and efficient algorithms
- **Thread Safety**: Safe concurrent operations where needed
- **Extensive Testing**: Comprehensive test coverage with benchmarks

## Components

### 1. Result and Option Types (`result.go`)

Functional programming primitives for explicit error handling and nullable values.

#### Result<T, E>

```go
// Success case
result := Ok[string, error]("hello world")
if result.IsOk() {
    fmt.Println(result.Value()) // "hello world"
}

// Error case  
result := Err[string, error](fmt.Errorf("something went wrong"))
if result.IsErr() {
    fmt.Println(result.Error()) // error details
}

// Chaining operations
final := NewChain(Ok[int, error](5)).
    Map(func(x int) int { return x * 2 }).
    Map(func(x int) int { return x + 10 }).
    Result()
// final contains Ok(20)

// Batch operations
results := []Result[int, error]{
    Ok[int, error](1),
    Ok[int, error](2), 
    Ok[int, error](3),
}
combined := All(results...) // Ok([]int{1, 2, 3})
```

#### Option<T>

```go
// Present value
option := Some("hello")
fmt.Println(option.ValueOr("default")) // "hello"

// Absent value
option := None[string]()
fmt.Println(option.ValueOr("default")) // "default"

// Chaining transformations
transformed := MapOption(Some(42), func(x int) string {
    return fmt.Sprintf("Value: %d", x)
})
// transformed contains Some("Value: 42")
```

### 2. High-Performance Collections (`collections.go`)

Generic collections with functional operations and thread-safe variants.

#### Set<T>

```go
// Basic set operations
set1 := NewSet(1, 2, 3)
set2 := NewSet(3, 4, 5)

union := set1.Union(set2)        // {1, 2, 3, 4, 5}
intersection := set1.Intersection(set2) // {3}
difference := set1.Difference(set2)     // {1, 2}

// Thread-safe variant
safeSet := NewSafeSet(1, 2, 3)
safeSet.Add(4) // Safe for concurrent access
```

#### Map<K, V>

```go
// Type-safe map operations
m := NewMap[string, int]()
m.Set("count", 42)

value := m.Get("count") // Returns Option[int]
if value.IsSome() {
    fmt.Println(value.Value()) // 42
}

// Functional operations
doubled := MapValues(m, func(v int) int { return v * 2 })
evens := FilterMap(m, func(k string, v int) bool { return v%2 == 0 })
```

#### Slice<T>

```go
// Enhanced slice operations
slice := NewSlice(1, 2, 3, 4, 5)

// Functional operations
doubled := MapSlice(slice, func(x int) int { return x * 2 })
evens := FilterSlice(slice, func(x int) bool { return x%2 == 0 })
sum := ReduceSlice(slice, 0, func(acc, x int) int { return acc + x })

// Utility operations
unique := UniqueSlice(NewSlice(1, 2, 2, 3, 3)) // {1, 2, 3}
chunks := ChunkSlice(slice, 2) // [[1,2], [3,4], [5]]
```

### 3. Type-Safe Client Wrappers (`clients.go`)

Generic client interfaces for different API types with built-in retry and circuit breaking.

#### HTTP Client

```go
type APIRequest struct {
    UserID int `json:"user_id"`
}

type APIResponse struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Configure HTTP client
config := HTTPClientConfig[APIRequest, APIResponse]{
    BaseURL:  "https://api.example.com",
    Endpoint: "/users",
    Method:   "POST",
    Timeout:  30 * time.Second,
}

client := NewHTTPClient(config)

// Execute request
request := APIRequest{UserID: 123}
result := client.Execute(context.Background(), request)

if result.IsOk() {
    response := result.Value()
    fmt.Printf("User: %s <%s>", response.Name, response.Email)
}

// With retry
result = client.ExecuteWithRetry(context.Background(), request, 3)
```

#### Kubernetes Client

```go
// Type-safe Kubernetes operations
config := KubernetesClientConfig{
    Config:    kubeConfig,
    Scheme:    scheme,
    Namespace: "default",
}

client, err := NewKubernetesClient(config, &corev1.Pod{})
if err != nil {
    log.Fatal(err)
}

// Type-safe operations
pod := &corev1.Pod{
    ObjectMeta: metav1.ObjectMeta{Name: "test-pod"},
    // ... pod spec
}

result := client.Create(context.Background(), pod)
if result.IsOk() {
    fmt.Println("Pod created successfully")
}
```

#### Client Pool

```go
// Load balancing and failover
pool := NewClientPool[APIRequest, APIResponse](
    client1, client2, client3,
)

// Round-robin execution
result := pool.Execute(context.Background(), request)

// Failover execution (tries all clients)
result = pool.ExecuteWithFailover(context.Background(), request)
```

### 4. Event System (`events.go`)

Type-safe event handling with filtering, routing, and aggregation.

#### Event Bus

```go
type UserEvent struct {
    UserID   string
    Action   string
    Metadata map[string]any
}

// Create event bus
config := EventBusConfig{
    BufferSize:  1000,
    WorkerCount: 5,
}
bus := NewEventBus[UserEvent](config)

// Subscribe to events
bus.Subscribe(func(ctx context.Context, event Event[UserEvent]) Result[bool, error] {
    fmt.Printf("Processing user event: %s", event.Data.Action)
    return Ok[bool, error](true)
})

// Publish events
event := NewEvent("user-123", "login", UserEvent{
    UserID: "user-123",
    Action: "login",
})

result := bus.Publish(event)
```

#### Event Filtering and Routing

```go
// Add filters
bus.AddFilter(TypeFilter[UserEvent]("login", "logout"))
bus.AddFilter(TimeRangeFilter[UserEvent](time.Now().Add(-1*time.Hour), time.Now()))

// Event routing
router := NewEventRouter[UserEvent](func(event Event[UserEvent]) string {
    return event.Data.Action // Route by action type
})

loginBus := NewEventBus[UserEvent](EventBusConfig{})
logoutBus := NewEventBus[UserEvent](EventBusConfig{})

router.AddRoute("login", loginBus)
router.AddRoute("logout", logoutBus)

// Route events
router.Route(event)
```

#### Event Aggregation

```go
// Aggregate events into summaries
aggregator := NewEventAggregator(
    inputBus,
    outputBus,
    func(events []Event[UserEvent]) Result[Event[UserSummary], error] {
        summary := UserSummary{
            TotalEvents: len(events),
            Actions:     make(map[string]int),
        }
        
        for _, event := range events {
            summary.Actions[event.Data.Action]++
        }
        
        return Ok[Event[UserSummary], error](
            NewEvent("summary", "aggregation", summary),
        )
    },
    5 * time.Minute, // Aggregation window
)
```

### 5. Configuration Management (`config.go`)

Type-safe configuration with multiple sources, validation, and hot reloading.

#### Environment Configuration

```go
type AppConfig struct {
    DatabaseURL string `env:"DATABASE_URL" validate:"required,url"`
    Port        int    `env:"PORT" default:"8080" validate:"min=1,max=65535"`
    Debug       bool   `env:"DEBUG" default:"false"`
    Timeout     time.Duration `env:"TIMEOUT" default:"30s"`
}

// Create configuration manager
manager := NewConfigManager(AppConfig{
    Port: 8080, // Default values
    Debug: false,
})

// Register environment provider
envProvider := NewEnvironmentProvider("APP_", ReflectiveEnvParser[AppConfig]{})
manager.RegisterProvider(ConfigSourceEnv, envProvider)

// Load configuration
result := manager.Load(context.Background(), "app", ConfigSourceEnv)
if result.IsOk() {
    config := result.Value()
    fmt.Printf("Server running on port %d", config.Port)
}
```

#### File Configuration

```go
// JSON configuration files
fileProvider := NewFileProvider[AppConfig]("/etc/app", FormatJSON)
manager.RegisterProvider(ConfigSourceFile, fileProvider)

// Load with cascading (env -> file -> kubernetes)
config := manager.Load(context.Background(), "app", 
    ConfigSourceEnv, ConfigSourceFile, ConfigSourceKubernetes)
```

#### Configuration Validation

```go
// Add validators
manager.AddValidator(func(config AppConfig) Result[bool, error] {
    if config.Port < 1024 && !config.Debug {
        return Ok[bool, error](false) // Privileged ports in production
    }
    return Ok[bool, error](true)
})

// Watch for changes
configChan := manager.Watch(context.Background(), "app", ConfigSourceFile)
go func() {
    for result := range configChan {
        if result.IsOk() {
            newConfig := result.Value()
            fmt.Println("Configuration updated:", newConfig)
        }
    }
}()
```

### 6. Middleware System (`middleware.go`)

Composable middleware with type safety and common implementations.

#### Middleware Chain

```go
type APIRequest struct {
    UserID string
    Data   map[string]any
}

type APIResponse struct {
    Status  string
    Message string
}

// Build middleware chain
chain := NewMiddlewareChain[APIRequest, APIResponse]().
    Add(LoggingMiddleware[APIRequest, APIResponse](logger)).
    Add(AuthenticationMiddleware[APIRequest, APIResponse](authenticator)).
    Add(RateLimitingMiddleware[APIRequest, APIResponse](rateLimiter)).
    Add(TimeoutMiddleware[APIRequest, APIResponse](30 * time.Second))

// Final handler
handler := func(ctx context.Context, req APIRequest) Result[APIResponse, error] {
    // Business logic
    return Ok[APIResponse, error](APIResponse{
        Status:  "success",
        Message: "Request processed",
    })
}

// Execute with middleware
result := chain.Execute(context.Background(), request, handler)
```

#### Built-in Middlewares

```go
// CORS middleware
corsMiddleware := CORSMiddleware[APIRequest, APIResponse](CORSConfig{
    AllowedOrigins: []string{"https://example.com"},
    AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowCredentials: true,
})

// Rate limiting
rateLimiter := NewTokenBucketRateLimiter(
    func(req APIRequest) string { return req.UserID }, // Key extractor
    100,                    // Capacity
    1 * time.Minute,       // Refill rate
)
rateLimitingMiddleware := RateLimitingMiddleware[APIRequest, APIResponse](rateLimiter)

// Circuit breaker
circuitBreakerMiddleware := CircuitBreakerMiddleware[APIRequest, APIResponse](breaker)

// Predefined middleware groups
webAPIChain := WebAPIGroup[APIRequest, APIResponse]()
secureAPIChain := SecureAPIGroup[APIRequest, APIResponse](authenticator, authorizer)
```

### 7. Validation Framework (`validation.go`)

Comprehensive validation with fluent builders, struct tags, and async support.

#### Validation Builder

```go
type User struct {
    Name     string
    Email    string
    Age      int
    Password string
}

// Build validator
validator := NewValidationBuilder[User]().
    Required("name", func(u User) any { return u.Name }).
    MinLength("name", 2, func(u User) string { return u.Name }).
    MaxLength("name", 50, func(u User) string { return u.Name }).
    Email("email", func(u User) string { return u.Email }).
    Range("age", 0, 120, func(u User) int64 { return int64(u.Age) }).
    Custom(PasswordStrength("password", func(u User) string { return u.Password })).
    Build()

// Validate
user := User{
    Name:     "John Doe",
    Email:    "john@example.com", 
    Age:      25,
    Password: "StrongPass123!",
}

result := validator(user)
if !result.Valid {
    for _, err := range result.Errors {
        fmt.Printf("Validation error - %s: %s\n", err.Field, err.Message)
    }
}
```

#### Struct Tag Validation

```go
type User struct {
    Name     string `validate:"required,min=2,max=50"`
    Email    string `validate:"required,email"`
    Age      int    `validate:"min=0,max=120"`
    Password string `validate:"required"`
}

structValidator := NewStructValidator[User]()
result := structValidator.Validate(user)
```

#### Async and Batch Validation

```go
// Async validation
asyncValidator := NewAsyncValidator(validator)
resultChan := asyncValidator.ValidateAsync(context.Background(), user)

// Batch validation
batchValidator := NewBatchValidator(validator, 10) // Max 10 concurrent
users := []User{ /* ... */ }
resultsChan := batchValidator.ValidateBatch(context.Background(), users)

for batchResult := range resultsChan {
    fmt.Printf("User %d: Valid=%v", batchResult.Index, batchResult.Result.Valid)
}
```

#### Conditional Validation

```go
conditionalValidator := NewConditionalValidator[User]().
    When(
        func(u User) bool { return u.Age >= 18 }, // Adult condition
        func(u User) ValidationResult {
            result := NewValidationResult()
            if len(u.Password) < 8 {
                result.AddError("password", "Adults need strong passwords", "password_strength", u.Password)
            }
            return result
        },
    ).
    When(
        func(u User) bool { return u.Age < 18 }, // Minor condition  
        func(u User) ValidationResult {
            // Different validation rules for minors
            return NewValidationResult()
        },
    )

result := conditionalValidator.Validate(user)
```

## Usage in Nephoran Intent Operator

### Network Intent Validation

```go
type NetworkIntent struct {
    Name        string
    Type        string
    Region      string
    QoSProfile  string
    Resources   ResourceRequirements
}

validator := NewValidationBuilder[NetworkIntent]().
    Required("name", func(ni NetworkIntent) any { return ni.Name }).
    OneOf("type", []string{"5G-Core", "RAN", "Edge"}, func(ni NetworkIntent) string { return ni.Type }).
    OneOf("region", []string{"us-east", "us-west", "eu-central"}, func(ni NetworkIntent) string { return ni.Region }).
    Custom(func(ni NetworkIntent) ValidationResult {
        // Custom business logic validation
        result := NewValidationResult()
        if ni.Type == "5G-Core" && ni.Resources.CPU < 4 {
            result.AddError("resources", "5G Core requires minimum 4 CPU cores", "insufficient_resources", ni.Resources)
        }
        return result
    }).
    Build()
```

### Configuration Management for Telecom Components

```go
type LLMConfig struct {
    APIKey      string        `env:"LLM_API_KEY" validate:"required"`
    Model       string        `env:"LLM_MODEL" default:"gpt-4o-mini" validate:"required"`
    Temperature float64       `env:"LLM_TEMPERATURE" default:"0.7" validate:"min=0,max=2"`
    Timeout     time.Duration `env:"LLM_TIMEOUT" default:"30s"`
}

type RAGConfig struct {
    VectorDB    string `env:"RAG_VECTOR_DB" default:"weaviate" validate:"oneof=weaviate pinecone"`
    IndexName   string `env:"RAG_INDEX_NAME" default:"telecom-knowledge" validate:"required"`
    Dimensions  int    `env:"RAG_DIMENSIONS" default:"1536" validate:"min=1"`
}

type NephoranConfig struct {
    LLM LLMConfig
    RAG RAGConfig
}

configManager := NewConfigManager(NephoranConfig{})
configManager.RegisterProvider(ConfigSourceEnv, envProvider)
configManager.RegisterProvider(ConfigSourceKubernetes, k8sProvider)

config := configManager.Load(context.Background(), "nephoran")
```

### Event-Driven Network Operations

```go
type NetworkEvent struct {
    IntentID    string
    EventType   string
    Status      string
    Timestamp   time.Time
    Details     map[string]any
}

eventBus := NewEventBus[NetworkEvent](EventBusConfig{
    BufferSize:  10000,
    WorkerCount: 20,
})

// Subscribe to deployment events
eventBus.SubscribeWithFilter(
    func(ctx context.Context, event Event[NetworkEvent]) Result[bool, error] {
        // Process network deployment events
        return deploymentHandler.Process(ctx, event.Data)
    },
    TypeFilter[NetworkEvent]("deployment.started", "deployment.completed", "deployment.failed"),
)

// Event aggregation for monitoring
aggregator := NewEventAggregator(
    eventBus,
    monitoringBus,
    func(events []Event[NetworkEvent]) Result[Event[NetworkMetrics], error] {
        metrics := calculateNetworkMetrics(events)
        return Ok[Event[NetworkMetrics], error](
            NewEvent("metrics", "aggregation", metrics),
        )
    },
    5 * time.Minute,
)
```

## Performance Characteristics

All generic components are designed for zero-runtime overhead:

- **Result/Option**: Compile-time abstractions with no boxing
- **Collections**: Memory-efficient with object pooling where appropriate  
- **Clients**: Connection pooling and request batching
- **Events**: Lock-free operations where possible
- **Validation**: Reflection used only during setup, not execution

## Testing

Each component includes comprehensive tests:

```bash
# Run all tests
go test ./pkg/generics/...

# Run with coverage
go test -cover ./pkg/generics/...

# Run benchmarks
go test -bench=. ./pkg/generics/...

# Run specific test
go test -run TestResult_Map ./pkg/generics/
```

## Integration Examples

See the test files for complete integration examples:
- `result_test.go` - Result/Option patterns
- `collections_test.go` - Collection operations  
- `validation_test.go` - Validation scenarios

## Migration Guide

When migrating existing code to use these generics:

1. **Start with Result types** for error handling
2. **Replace maps/slices** with generic collections where type safety matters
3. **Add validation** to struct definitions
4. **Use event bus** for decoupled communication
5. **Wrap clients** with generic interfaces

The components are designed to be incrementally adoptable without breaking existing code.