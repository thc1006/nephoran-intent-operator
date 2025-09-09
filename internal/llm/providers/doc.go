// Package providers implements the LLM provider abstraction for the nephoran-intent-operator.
//
// This package provides a clean, extensible interface for converting natural language
// input into structured NetworkIntent JSON that conforms to docs/contracts/intent.schema.json.
//
// # Architecture
//
// The package follows a clean architecture pattern with these key components:
//
//   - Provider interface: Core abstraction for LLM providers
//   - Factory pattern: Creates providers based on configuration
//   - Mock implementation: Enables testing without external dependencies
//   - Environment-based configuration: Supports deployment scenarios
//
// # Supported Providers
//
// The system supports multiple provider types through the ProviderType enum:
//
//   - OFFLINE: Deterministic provider for testing and offline scenarios
//   - OPENAI: OpenAI GPT models integration
//   - ANTHROPIC: Anthropic Claude models integration
//
// # Configuration
//
// Providers are configured through environment variables:
//
//   - LLM_PROVIDER: Specifies the provider type (OFFLINE|OPENAI|ANTHROPIC)
//   - LLM_API_KEY: API key for authenticated providers
//   - LLM_BASE_URL: Override default API endpoint
//   - LLM_MODEL: Specify model to use
//   - LLM_TIMEOUT: Request timeout in seconds
//   - LLM_MAX_RETRIES: Maximum number of retries for failed requests
//
// # Usage Example
//
//	// Create provider from environment
//	provider, err := providers.CreateFromEnvironment()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer provider.Close()
//
//	// Process natural language input
//	ctx := context.Background()
//	response, err := provider.ProcessIntent(ctx, "Scale the O-DU to 5 replicas in oran-odu namespace")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// The response.JSON contains NetworkIntent that conforms to intent.schema.json
//	fmt.Printf("Generated intent: %s\n", response.JSON)
//
// # Error Handling
//
// The package defines common error types and helper functions for error classification:
//
//   - IsRetryableError(): Checks if an error indicates a retryable condition
//   - IsAuthError(): Checks if an error is authentication-related  
//   - IsConfigError(): Checks if an error is configuration-related
//
// # Testing
//
// The MockProvider enables comprehensive testing without external API dependencies:
//
//	// Create mock provider
//	mock := providers.NewMockProvider(&providers.Config{Type: "MOCK"})
//
//	// Set predefined response
//	mock.SetMockResponse("test input", expectedResponse)
//
//	// Test your code
//	response, err := mock.ProcessIntent(ctx, "test input")
//
// # Extensibility
//
// New providers can be added by:
//
// 1. Implementing the Provider interface
// 2. Registering a constructor with the factory
// 3. Adding the provider type to the ProviderType enum
//
// The factory pattern ensures providers are created consistently and validated properly.
//
// # Schema Compliance
//
// All providers must generate JSON that conforms to docs/contracts/intent.schema.json.
// The schema defines the structure for NetworkIntent operations including:
//
//   - intent_type: Type of operation (scaling, deployment, configuration)
//   - target: Resource to modify
//   - namespace: Kubernetes namespace
//   - replicas: Desired replica count
//   - Additional metadata and constraints
//
// # Thread Safety
//
// Provider implementations should be thread-safe for concurrent use.
// The factory is safe for concurrent access.
// Individual provider instances may have specific thread-safety requirements.
package providers