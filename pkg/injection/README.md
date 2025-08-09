# Dependency Injection System

This package provides a comprehensive dependency injection container for the Nephoran Intent Operator, implementing proper dependency inversion principles and making testing significantly easier.

## Overview

The dependency injection system follows these principles:
- **Singleton Pattern**: Dependencies are created once and reused
- **Factory Pattern**: Providers create configured instances
- **Interface Segregation**: Clear interfaces between components
- **Dependency Inversion**: Dependencies are injected, not created internally

## Architecture

### Container (`container.go`)
The `Container` is the main dependency injection container that:
- Manages singleton instances
- Registers and executes providers
- Thread-safe operations with proper locking
- Lazy initialization of dependencies

### Providers (`providers.go`)
Provider functions create and configure dependencies:
- `HTTPClientProvider`: Creates configured HTTP clients
- `GitClientProvider`: Creates Git clients with repository configuration
- `LLMClientProvider`: Creates LLM clients for AI processing
- `PackageGeneratorProvider`: Creates Nephio package generators
- `TelecomKnowledgeBaseProvider`: Creates telecom domain knowledge bases
- `MetricsCollectorProvider`: Creates Prometheus metrics collectors
- `LLMSanitizerProvider`: Creates security sanitizers for LLM inputs

### Interfaces (`interfaces.go`)
Defines the `Dependencies` interface that controllers expect, ensuring type safety and testability.

## Usage

### Basic Usage in main.go

```go
package main

import (
    "github.com/thc1006/nephoran-intent-operator/pkg/config"
    "github.com/thc1006/nephoran-intent-operator/pkg/injection"
    "github.com/thc1006/nephoran-intent-operator/pkg/controllers"
)

func main() {
    // Load configuration
    constants := config.LoadConstants()
    
    // Create dependency injection container
    container := injection.NewContainer(constants)
    
    // Set manager-specific dependencies
    container.SetEventRecorder(mgr.GetEventRecorderFor("nephoran-intent-operator"))
    
    // Create controller with injected dependencies
    controller, err := controllers.NewNetworkIntentReconciler(
        mgr.GetClient(),
        mgr.GetScheme(),
        container, // Container implements Dependencies interface
        controllerConfig,
    )
}
```

### Testing with Dependency Injection

```go
func TestControllerWithMockDependencies(t *testing.T) {
    container := injection.NewContainer(nil)
    
    // Override with mock implementations
    container.RegisterProvider("git_client", func(c *injection.Container) (interface{}, error) {
        return &MockGitClient{}, nil
    })
    
    container.RegisterProvider("llm_client", func(c *injection.Container) (interface{}, error) {
        return &MockLLMClient{}, nil
    })
    
    // Use container in tests
    controller := &NetworkIntentReconciler{
        dependencies: container,
    }
    
    // Test controller behavior with mocked dependencies
}
```

## Dependencies

The system manages these core dependencies:

### HTTP Client
- Configured with timeouts and connection pooling
- Used for external API communications
- Provider: `HTTPClientProvider`

### Git Client
- Repository management for GitOps workflows
- Configured from environment variables
- Provider: `GitClientProvider`
- Config: `GIT_REPO_URL`, `GIT_BRANCH`, `GIT_TOKEN`

### LLM Client
- Integration with AI/ML services
- Implements `shared.ClientInterface`
- Provider: `LLMClientProvider`
- Config: `LLM_PROCESSOR_URL`

### Package Generator
- Nephio package creation and management
- O-RAN network function deployment
- Provider: `PackageGeneratorProvider`
- Config: `PORCH_URL`

### Telecom Knowledge Base
- Domain-specific telecommunications knowledge
- RAG (Retrieval-Augmented Generation) support
- Provider: `TelecomKnowledgeBaseProvider`

### Metrics Collector
- Prometheus metrics collection
- Performance and operational monitoring
- Provider: `MetricsCollectorProvider`

### LLM Sanitizer
- Security validation for AI/ML inputs and outputs
- Prevents injection attacks
- Provider: `LLMSanitizerProvider`

## Configuration

Dependencies are configured through:

1. **Environment Variables**: External service endpoints and credentials
2. **Config Constants**: Internal timeouts, limits, and operational parameters
3. **Runtime Settings**: Manager-provided instances like EventRecorder

### Environment Variables

```bash
# Git Configuration
GIT_REPO_URL=https://github.com/your-org/nephoran-packages.git
GIT_BRANCH=main
GIT_TOKEN=ghp_your_token_here
GIT_TOKEN_PATH=/path/to/token/file
GIT_REPO_PATH=/tmp/nephoran-git

# LLM Configuration
LLM_PROCESSOR_URL=http://localhost:8080

# Nephio Configuration
PORCH_URL=https://porch.nephio.io
USE_NEPHIO_PORCH=true
```

## Error Handling

The system includes comprehensive error handling:

- **Provider Errors**: Graceful degradation when dependencies fail to initialize
- **Missing Configuration**: Appropriate defaults and error messages
- **Runtime Errors**: Proper error propagation without system crashes

### Error Types

```go
type ErrMissingGitConfig struct {
    Field string
}

type ErrMissingLLMConfig struct {
    Field string
}
```

## Thread Safety

The container is fully thread-safe:
- RWMutex protects internal maps
- Singleton creation is atomic
- Concurrent access is supported

## Extension Points

### Custom Providers

Register custom providers for specialized dependencies:

```go
container.RegisterProvider("custom_service", func(c *injection.Container) (interface{}, error) {
    config := c.GetConfig()
    return NewCustomService(config), nil
})
```

### Advanced Providers

The system includes advanced provider examples:
- `SecureHTTPClientProvider`: Enhanced security settings
- `CircuitBreakerHTTPClientProvider`: Fault tolerance patterns
- `ConfiguredGitClientProvider`: Full Git configuration
- `ProductionLLMClientProvider`: Production-ready LLM settings

## Best Practices

1. **Interface-Driven Development**: Always program against interfaces
2. **Minimal Dependencies**: Only inject what's actually needed
3. **Configuration Separation**: Keep config separate from business logic
4. **Error Handling**: Always handle dependency creation errors
5. **Testing**: Use dependency injection for comprehensive testing

## Migration Guide

### From Direct Dependencies

**Before:**
```go
func NewController() *Controller {
    gitClient := git.NewGitClient(&git.Config{...})
    llmClient := shared.NewHTTPClient(url)
    return &Controller{
        gitClient: gitClient,
        llmClient: llmClient,
    }
}
```

**After:**
```go
func NewController(deps Dependencies) *Controller {
    return &Controller{
        dependencies: deps,
    }
}

// In main.go
container := injection.NewContainer(constants)
controller := NewController(container)
```

### Testing Migration

**Before:**
```go
func TestController(t *testing.T) {
    // Hard to mock dependencies
    controller := NewController()
    // Limited testing capabilities
}
```

**After:**
```go
func TestController(t *testing.T) {
    container := injection.NewContainer(nil)
    container.RegisterProvider("git_client", func(c *injection.Container) (interface{}, error) {
        return &MockGitClient{}, nil
    })
    
    controller := NewController(container)
    // Full control over dependencies for testing
}
```

## Performance Considerations

- **Lazy Loading**: Dependencies created only when first accessed
- **Singleton Caching**: No duplicate instance creation
- **Minimal Overhead**: Container operations are highly optimized
- **Memory Efficient**: Shared instances reduce memory footprint

## Security

The dependency injection system enhances security by:
- **Centralized Configuration**: Single point for security settings
- **Environment Isolation**: Clean separation of secrets
- **Sanitization**: Built-in LLM input/output validation
- **Audit Trail**: All dependency creation is logged

## Monitoring

Dependencies include built-in monitoring:
- Prometheus metrics for all major operations
- Health checks for external services
- Performance tracking and alerting
- Resource utilization monitoring