package examples

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nephoran/intent-operator/pkg/llm"
)

// SmartEndpointUsageExample demonstrates proper usage of the updated client integration
// with smart URL handling logic to fix 404 errors
func SmartEndpointUsageExample() error {
	logger := slog.Default()

	// Example 1: Legacy URL pattern (backward compatibility)
	legacyClient := llm.NewClientWithConfig("http://rag-api:5001/process_intent", llm.ClientConfig{
		BackendType: "rag",
		Timeout:     30 * time.Second,
	})

	// Example 2: New URL pattern (preferred)
	newClient := llm.NewClientWithConfig("http://rag-api:5001/process", llm.ClientConfig{
		BackendType: "rag",
		Timeout:     30 * time.Second,
	})

	// Example 3: Base URL pattern (auto-detects to /process)
	autoDetectClient := llm.NewClientWithConfig("http://rag-api:5001", llm.ClientConfig{
		BackendType: "rag",
		Timeout:     30 * time.Second,
	})

	// Example 4: ProcessingEngine with smart endpoint configuration
	processingConfig := &llm.ProcessingConfig{
		EnableRAG:       true,
		RAGAPIURL:       "http://rag-api:5001", // Base URL - will auto-detect to /process
		RAGTimeout:      30 * time.Second,
		QueryTimeout:    30 * time.Second,
		FallbackToBase:  true,
		EnableStreaming: true,
		StreamTimeout:   5 * time.Minute,
		EnableBatching:  true,
	}

	processingEngine := llm.NewProcessingEngine(autoDetectClient, processingConfig)
	defer processingEngine.Shutdown(context.Background())

	// Example 5: StreamingProcessor with endpoint configuration
	streamingConfig := &llm.StreamingConfig{
		StreamTimeout:        5 * time.Minute,
		MaxConcurrentStreams: 10,
		HeartbeatInterval:    30 * time.Second,
		BufferSize:           4096,
	}

	tokenManager := llm.NewTokenManager()
	streamingProcessor := llm.NewStreamingProcessor(autoDetectClient, tokenManager, streamingConfig)
	
	// Configure RAG endpoints for streaming
	streamingProcessor.SetRAGEndpoints("http://rag-api:5001")
	defer streamingProcessor.Close()

	ctx := context.Background()

	// Test the configurations
	logger.Info("Testing smart endpoint configurations...")

	// Test cases for different URL patterns
	testCases := []struct {
		name   string
		client *llm.Client
		url    string
	}{
		{"Legacy URL Pattern", legacyClient, "http://rag-api:5001/process_intent"},
		{"New URL Pattern", newClient, "http://rag-api:5001/process"},
		{"Auto-detect Pattern", autoDetectClient, "http://rag-api:5001 (auto-detects to /process)"},
	}

	for _, tc := range testCases {
		logger.Info("Testing configuration", 
			slog.String("name", tc.name),
			slog.String("url", tc.url),
		)

		// Each client will automatically use the correct endpoint
		result, err := tc.client.ProcessIntent(ctx, "Deploy a high-availability AMF instance")
		if err != nil {
			logger.Error("Client test failed", 
				slog.String("name", tc.name),
				slog.String("error", err.Error()),
			)
			continue
		}

		logger.Info("Client test successful", 
			slog.String("name", tc.name),
			slog.String("result_preview", truncate(result, 100)),
		)
	}

	// Test ProcessingEngine
	logger.Info("Testing ProcessingEngine with smart endpoints...")
	processingResult, err := processingEngine.ProcessIntent(ctx, "Configure UPF for edge deployment")
	if err != nil {
		logger.Error("ProcessingEngine test failed", slog.String("error", err.Error()))
	} else {
		logger.Info("ProcessingEngine test successful", 
			slog.String("result_preview", truncate(processingResult.Content, 100)),
			slog.Duration("processing_time", processingResult.ProcessingTime),
			slog.Bool("cache_hit", processingResult.CacheHit),
		)
	}

	// Verify endpoint configuration
	process, stream, health := streamingProcessor.GetConfiguredEndpoints()
	logger.Info("StreamingProcessor endpoints configured",
		slog.String("process", process),
		slog.String("stream", stream),
		slog.String("health", health),
	)

	return nil
}

// ConfigurationFromEnvironmentExample shows how to configure endpoints from environment variables
func ConfigurationFromEnvironmentExample() error {
	logger := slog.Default()

	// Load configuration from environment
	// This demonstrates how the LLMProcessorConfig integrates with smart endpoints
	
	// Example environment variables:
	// RAG_API_URL=http://rag-api:5001                    # Auto-detects to /process
	// RAG_API_URL=http://rag-api:5001/process_intent     # Uses legacy endpoint
	// RAG_API_URL=http://rag-api:5001/process            # Uses new endpoint
	// RAG_PREFER_PROCESS_ENDPOINT=true                   # Prefers /process over /process_intent
	// RAG_ENDPOINT_PATH=/custom_endpoint                 # Custom endpoint override

	logger.Info("Smart endpoint configuration patterns:")
	logger.Info("1. Base URL (http://rag-api:5001) -> Auto-detects to /process")
	logger.Info("2. Legacy URL (http://rag-api:5001/process_intent) -> Uses as configured")
	logger.Info("3. New URL (http://rag-api:5001/process) -> Uses as configured")
	logger.Info("4. Custom path via RAG_ENDPOINT_PATH -> Uses custom endpoint")

	// The configuration system handles all these patterns automatically
	logger.Info("All patterns ensure backward compatibility while supporting new endpoints")

	return nil
}

// TroubleshootingExample shows common issues and solutions
func TroubleshootingExample() error {
	logger := slog.Default()

	logger.Info("Common endpoint configuration issues and solutions:")

	// Issue 1: 404 errors with hardcoded endpoints
	logger.Info("FIXED: 404 errors from hardcoded /process endpoints")
	logger.Info("Solution: Smart endpoint detection automatically uses correct URLs")

	// Issue 2: Legacy vs new endpoint confusion
	logger.Info("FIXED: Confusion between /process_intent and /process endpoints")
	logger.Info("Solution: Auto-detection based on URL pattern, backward compatible")

	// Issue 3: Streaming endpoint mismatches
	logger.Info("FIXED: Streaming endpoints not matching process endpoints")
	logger.Info("Solution: Consistent endpoint derivation for all operations")

	// Issue 4: Configuration complexity
	logger.Info("FIXED: Complex endpoint configuration requirements")
	logger.Info("Solution: Sensible defaults with explicit override options")

	return nil
}

// BestPracticesExample demonstrates best practices for endpoint configuration
func BestPracticesExample() error {
	logger := slog.Default()

	logger.Info("Best practices for endpoint configuration:")

	// Best Practice 1: Use base URLs for new deployments
	logger.Info("✅ Use base URLs (http://rag-api:5001) for new deployments")
	logger.Info("   - Auto-detects to preferred /process endpoint")
	logger.Info("   - Simplifies configuration")
	logger.Info("   - Future-proof")

	// Best Practice 2: Keep existing URLs for legacy systems
	logger.Info("✅ Keep full URLs for existing systems")
	logger.Info("   - Maintains backward compatibility")
	logger.Info("   - No breaking changes required")
	logger.Info("   - Smooth migration path")

	// Best Practice 3: Use environment variables for configuration
	logger.Info("✅ Use environment variables for flexibility")
	logger.Info("   - RAG_API_URL for the main endpoint")
	logger.Info("   - RAG_PREFER_PROCESS_ENDPOINT for preference")
	logger.Info("   - RAG_ENDPOINT_PATH for custom overrides")

	// Best Practice 4: Monitor endpoint usage
	logger.Info("✅ Monitor endpoint usage and errors")
	logger.Info("   - Check logs for endpoint resolution")
	logger.Info("   - Verify correct URLs are being used")
	logger.Info("   - Monitor for 404 errors")

	return nil
}

// Helper function to truncate long strings
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}