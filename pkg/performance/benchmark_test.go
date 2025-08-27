package performance

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/handlers"
	"github.com/thc1006/nephoran-intent-operator/pkg/health"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/services"
)

// BenchmarkLLMProcessing tests the performance of LLM intent processing
func BenchmarkLLMProcessing(b *testing.B) {
	// Setup
	_ = &config.LLMProcessorConfig{
		LLMBackendType: "mock",
		LLMModelName:   "test-model",
		LLMMaxTokens:   1000,
		LLMTimeout:     30 * time.Second,
		ServiceVersion: "test",
	}
	
	logger := slog.Default()
	
	// Create mock LLM client
	client := createMockLLMClient()
	
	// Create intent processor
	processor := &handlers.IntentProcessor{
		LLMClient: client,
		Logger:    logger,
	}

	testIntent := "Scale up the 5G core deployment to handle increased traffic"
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := processor.ProcessIntent(ctx, testIntent)
			if err != nil {
				b.Errorf("ProcessIntent failed: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentLLMProcessing tests concurrent processing performance
func BenchmarkConcurrentLLMProcessing(b *testing.B) {
	_ = &config.LLMProcessorConfig{
		LLMBackendType: "mock",
		LLMModelName:   "test-model",
		LLMMaxTokens:   1000,
		LLMTimeout:     30 * time.Second,
		ServiceVersion: "test",
	}
	
	logger := slog.Default()
	client := createMockLLMClient()
	
	processor := &handlers.IntentProcessor{
		LLMClient: client,
		Logger:    logger,
	}

	testIntents := []string{
		"Scale up the 5G core deployment to handle increased traffic",
		"Deploy new RAN components in the edge location",
		"Configure load balancing for the core network",
		"Optimize the database performance for user data",
		"Setup monitoring alerts for network degradation",
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			intent := testIntents[i%len(testIntents)]
			
			_, err := processor.ProcessIntent(ctx, intent)
			if err != nil {
				b.Errorf("ProcessIntent failed: %v", err)
			}
			
			cancel()
			i++
		}
	})
}

// BenchmarkHealthChecks tests health check performance
func BenchmarkHealthChecks(b *testing.B) {
	logger := slog.Default()
	hc := health.NewHealthChecker("test-service", "1.0.0", logger)
	
	// Register some test health checks
	for i := 0; i < 10; i++ {
		name := "test-check-" + string(rune('0'+i))
		hc.RegisterCheck(name, func(ctx context.Context) *health.Check {
			return &health.Check{
				Status:  health.StatusHealthy,
				Message: "OK",
			}
		})
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			response := hc.Check(ctx)
			if response.Status != health.StatusHealthy {
				b.Errorf("Health check failed: %v", response.Status)
			}
		}
	})
}

// BenchmarkStringOperations tests string building performance optimizations
func BenchmarkStringOperations(b *testing.B) {
	testStrings := []string{
		"backend",
		"model",
		"intent-type",
		"this is a test intent that we want to process",
	}

	b.Run("StringBuilder", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			var builder strings.Builder
			totalLen := 0
			for _, s := range testStrings {
				totalLen += len(s)
			}
			totalLen += len(testStrings) - 1 // separators
			
			builder.Grow(totalLen)
			for j, s := range testStrings {
				if j > 0 {
					builder.WriteByte(':')
				}
				builder.WriteString(s)
			}
			_ = builder.String()
		}
	})

	b.Run("StringJoin", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			_ = strings.Join(testStrings, ":")
		}
	})
}

// BenchmarkMemoryPools tests sync.Pool performance
func BenchmarkMemoryPools(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			builder := llm.GetStringBuilder()
			builder.WriteString("test-string")
			builder.WriteString(":")
			builder.WriteString("another-test-string")
			result := builder.String()
			llm.PutStringBuilder(builder)
			_ = result
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		
		for i := 0; i < b.N; i++ {
			var builder strings.Builder
			builder.WriteString("test-string")
			builder.WriteString(":")
			builder.WriteString("another-test-string")
			result := builder.String()
			_ = result
		}
	})
}

// BenchmarkServiceInitialization tests service startup performance
func BenchmarkServiceInitialization(b *testing.B) {
	config := &config.LLMProcessorConfig{
		LLMBackendType: "mock",
		LLMModelName:   "test-model",
		LLMMaxTokens:   1000,
		LLMTimeout:     30 * time.Second,
		ServiceVersion: "test",
	}
	
	logger := slog.Default()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		service := services.NewLLMProcessorService(config, logger)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		
		err := service.Initialize(ctx)
		if err != nil {
			b.Errorf("Service initialization failed: %v", err)
		}
		
		_ = service.Shutdown(ctx)
		cancel()
	}
}

// createMockLLMClient creates a mock LLM client for testing
func createMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: map[string]string{
			"default": `{"type":"NetworkFunctionDeployment","name":"5g-core","namespace":"default","spec":{"replicas":3,"image":"5g-core:latest"}}`,
		},
		latency: 10 * time.Millisecond,
	}
}

// MockLLMClient simulates an LLM client for performance testing
type MockLLMClient struct {
	responses map[string]string
	latency   time.Duration
}

func (m *MockLLMClient) ProcessIntent(ctx context.Context, intent string) (string, error) {
	// Simulate processing time
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(m.latency):
	}
	
	// Return mock response
	if response, exists := m.responses[intent]; exists {
		return response, nil
	}
	return m.responses["default"], nil
}

func (m *MockLLMClient) Shutdown() {
	// Mock shutdown
}