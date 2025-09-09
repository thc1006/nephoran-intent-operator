# LLM Provider Interface

This package provides a clean, extensible interface for LLM providers that convert natural language input into structured NetworkIntent JSON conforming to `docs/contracts/intent.schema.json`.

## Architecture

The design follows clean architecture principles with these key components:

### Core Interfaces

- **`Provider`**: Main interface for LLM providers that process intents
- **`Factory`**: Creates providers based on configuration
- **`Config`**: Provider configuration with validation
- **`IntentResponse`**: Structured response with metadata

### Provider Types

- **`OFFLINE`**: Deterministic provider for testing and offline scenarios
- **`OPENAI`**: OpenAI GPT models integration
- **`ANTHROPIC`**: Anthropic Claude models integration

### Key Features

- **Environment-based configuration**: Configure via `LLM_PROVIDER`, `LLM_API_KEY`, etc.
- **Mockable interface**: Full mock implementation for testing
- **Error classification**: Retryable, auth, and config error helpers
- **Context support**: Timeout and cancellation support
- **Schema compliance**: All outputs conform to `intent.schema.json`

## Configuration

Configure providers using environment variables:

```bash
# Provider selection
export LLM_PROVIDER=OFFLINE          # OFFLINE|OPENAI|ANTHROPIC

# Authentication (for OPENAI/ANTHROPIC)
export LLM_API_KEY=sk-your-key       # API key
export LLM_BASE_URL=https://api.example.com  # Optional custom endpoint
export LLM_MODEL=gpt-4               # Optional model specification

# Behavior
export LLM_TIMEOUT=30                # Request timeout in seconds
export LLM_MAX_RETRIES=3             # Max retry attempts
```

## Usage Examples

### Basic Usage

```go
// Create provider from environment
provider, err := providers.CreateFromEnvironment()
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

// Process natural language input  
ctx := context.Background()
response, err := provider.ProcessIntent(ctx, "Scale O-DU to 5 replicas in oran-odu namespace")
if err != nil {
    log.Fatal(err)
}

// Parse structured response
var intent map[string]interface{}
json.Unmarshal(response.JSON, &intent)
fmt.Printf("Generated intent: %+v\n", intent)
```

### Factory Pattern

```go
// Create factory
factory := providers.NewFactory()

// Configure provider
config := &providers.Config{
    Type:       providers.ProviderTypeOpenAI,
    APIKey:     "sk-your-key", 
    Timeout:    45 * time.Second,
    MaxRetries: 5,
}

// Create provider
provider, err := factory.CreateProvider(config)
if err != nil {
    log.Fatal(err)
}
defer provider.Close()

// Use provider...
```

### Error Handling

```go
response, err := provider.ProcessIntent(ctx, input)
if err != nil {
    if providers.IsRetryableError(err) {
        // Implement retry logic
        return
    }
    
    if providers.IsAuthError(err) {
        // Handle authentication failure
        return 
    }
    
    if providers.IsConfigError(err) {
        // Handle configuration error
        return
    }
    
    // Handle other errors
    log.Fatal(err)
}
```

### Testing with Mock Provider

```go
// Create mock for testing
mock := providers.NewMockProvider(&providers.Config{
    Type: providers.ProviderType("MOCK"),
})

// Set expected response
expectedIntent := providers.CreateMockIntentResponse("scaling", "test-target", "test-ns", 3)
mock.SetMockResponse("test input", expectedIntent)

// Test your code
response, err := mock.ProcessIntent(ctx, "test input")
assert.NoError(t, err)

// Verify calls
assert.Equal(t, 1, mock.GetCallCount())
assert.Equal(t, "test input", mock.GetLastInput())
```

## Implementation Status

### ✅ Completed
- Core interfaces and types
- Factory pattern implementation  
- Environment configuration
- Mock provider for testing
- Comprehensive test suite
- Error classification helpers

### 🚧 To Be Implemented

The interface design is complete, but individual providers need implementation:

1. **Offline Provider** (`internal/llm/providers/offline.go`)
   - Deterministic rule-based intent generation
   - No external dependencies
   - Fast, consistent responses

2. **OpenAI Provider** (`internal/llm/providers/openai.go`)
   - Integration with OpenAI GPT models
   - API key authentication
   - Rate limiting and retry logic

3. **Anthropic Provider** (`internal/llm/providers/anthropic.go`)
   - Integration with Anthropic Claude models  
   - API key authentication
   - Rate limiting and retry logic

Each provider should:
- Implement the `Provider` interface
- Register with the factory via constructor function
- Generate JSON conforming to `intent.schema.json`
- Handle authentication and rate limits appropriately
- Provide comprehensive error handling

## Schema Compliance

All providers must generate JSON that conforms to `docs/contracts/intent.schema.json`. Required fields:

```json
{
  "intent_type": "scaling|deployment|configuration",
  "target": "resource-name",
  "namespace": "kubernetes-namespace", 
  "replicas": 1-100,
  "reason": "optional-reason",
  "source": "user|planner|test",
  "priority": 0-10,
  "status": "pending|processing|completed|failed"
}
```

## Thread Safety

- Factory is safe for concurrent access
- Provider instances should be thread-safe
- Mock provider includes call tracking for testing

## Extension Points

To add new provider types:

1. Add to `ProviderType` enum
2. Implement `Provider` interface
3. Register constructor with factory
4. Add configuration validation
5. Update tests and documentation

The factory pattern ensures consistent creation and validation across all provider types.