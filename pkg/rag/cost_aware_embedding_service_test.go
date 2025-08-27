package rag

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"
	"testing"
	"time"
)

// MockProvider implements EmbeddingProvider for testing
type MockProvider struct {
	name         string
	config       ProviderConfig
	failureRate  float64
	latency      time.Duration
	costPerToken float64
	healthStatus bool
	callCount    int
	mu           sync.Mutex
}

func NewMockProvider(name string, costPerToken float64, latency time.Duration) *MockProvider {
	return &MockProvider{
		name: name,
		config: ProviderConfig{
			Name:         name,
			CostPerToken: costPerToken,
			Enabled:      true,
			Healthy:      true,
			Priority:     5,
			RateLimit:    1000,
		},
		costPerToken: costPerToken,
		latency:      latency,
		healthStatus: true,
	}
}

func (m *MockProvider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, TokenUsage, error) {
	m.mu.Lock()
	m.callCount++
	callNum := m.callCount
	m.mu.Unlock()

	// Simulate latency
	select {
	case <-time.After(m.latency):
	case <-ctx.Done():
		return nil, TokenUsage{}, ctx.Err()
	}

	// Simulate failures
	if m.failureRate > 0 && float64(callNum%100)/100.0 < m.failureRate {
		return nil, TokenUsage{}, fmt.Errorf("provider %s failed", m.name)
	}

	// Generate mock embeddings
	embeddings := make([][]float32, len(texts))
	for i := range embeddings {
		embeddings[i] = []float32{0.1, 0.2, 0.3} // Simple mock embedding
	}

	// Calculate token usage
	totalTokens := 0
	for _, text := range texts {
		totalTokens += len(text) / 4 // Rough estimation
	}

	usage := TokenUsage{
		PromptTokens:  totalTokens,
		TotalTokens:   totalTokens,
		EstimatedCost: float64(totalTokens) * m.costPerToken / 1000,
	}

	return embeddings, usage, nil
}

func (m *MockProvider) GetConfig() ProviderConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.config
}

func (m *MockProvider) HealthCheck(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.healthStatus {
		return fmt.Errorf("provider %s is unhealthy", m.name)
	}

	m.config.Healthy = true
	m.config.LastCheck = time.Now()
	return nil
}

func (m *MockProvider) GetCostEstimate(tokenCount int) float64 {
	return float64(tokenCount) * m.costPerToken / 1000
}

func (m *MockProvider) GetName() string {
	return m.name
}

func (m *MockProvider) IsHealthy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.healthStatus
}

func (m *MockProvider) GetLatency() time.Duration {
	return m.latency
}

func (m *MockProvider) SetHealthStatus(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthStatus = healthy
	m.config.Healthy = healthy
}

func (m *MockProvider) SetFailureRate(rate float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failureRate = rate
}

func TestCostAwareEmbeddingService_ProviderSelection(t *testing.T) {
	// Create base embedding service with multiple providers
	baseConfig := &EmbeddingConfig{
		EnableCaching:      true,
		FallbackEnabled:    true,
		EnableCostTracking: true,
		DailyCostLimit:     100.0,
		CacheTTL:           1 * time.Hour,
		L1CacheSize:        100,
	}

	baseService := &EmbeddingService{
		config:      baseConfig,
		logger:      slog.Default(),
		providers:   make(map[string]EmbeddingProvider),
		cache:       NewInMemoryCache(100),
		redisCache:  &NoOpRedisCache{},
		rateLimiter: NewRateLimiter(1000, 10000),
		metrics: &EmbeddingMetrics{
			ModelStats: make(map[string]ModelUsageStats),
		},
	}

	// Add mock providers with different characteristics
	expensiveProvider := NewMockProvider("expensive", 0.001, 100*time.Millisecond)
	cheapProvider := NewMockProvider("cheap", 0.0001, 200*time.Millisecond)
	fastProvider := NewMockProvider("fast", 0.0005, 50*time.Millisecond)

	baseService.providers["expensive"] = expensiveProvider
	baseService.providers["cheap"] = cheapProvider
	baseService.providers["fast"] = fastProvider

	// Create cost-aware service
	costConfig := &CostOptimizerConfig{
		OptimizationStrategy:    "balanced",
		CostWeight:              0.4,
		PerformanceWeight:       0.3,
		QualityWeight:           0.3,
		DailyBudget:             10.0,
		EnableBudgetTracking:    true,
		MinProviderScore:        0.1,
		ProviderTimeout:         1 * time.Second,
		CircuitBreakerThreshold: 3,
		CircuitBreakerTimeout:   5 * time.Second,
	}

	costAwareService := NewCostAwareEmbeddingService(baseService, costConfig)

	// Test provider selection
	tests := []struct {
		name             string
		strategy         string
		expectedProvider string
		setupFunc        func()
	}{
		{
			name:             "Cost optimization selects cheapest",
			strategy:         "aggressive",
			expectedProvider: "cheap",
			setupFunc: func() {
				costAwareService.costOptimizer.config.OptimizationStrategy = "aggressive"
			},
		},
		{
			name:             "Performance optimization selects fastest",
			strategy:         "quality_first",
			expectedProvider: "fast",
			setupFunc: func() {
				costAwareService.costOptimizer.config.OptimizationStrategy = "quality_first"
				// Update metrics to show fast provider has better performance
				costAwareService.providerMonitor.metrics["fast"] = &ProviderMetrics{
					AverageLatency: 50 * time.Millisecond,
					SuccessRate:    1.0,
					HealthScore:    1.0,
				}
			},
		},
		{
			name:             "Balanced strategy considers all factors",
			strategy:         "balanced",
			expectedProvider: "cheap", // Depends on weights and metrics
			setupFunc: func() {
				costAwareService.costOptimizer.config.OptimizationStrategy = "balanced"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			ctx := context.Background()
			request := &EmbeddingRequest{
				Texts:    []string{"Test text for embedding"},
				UseCache: false,
			}

			provider, err := costAwareService.selectOptimalProvider(ctx, request)
			if err != nil {
				t.Fatalf("Failed to select provider: %v", err)
			}

			if provider.GetName() != tt.expectedProvider {
				t.Errorf("Expected provider %s, got %s", tt.expectedProvider, provider.GetName())
			}
		})
	}
}

func TestCostAwareEmbeddingService_Fallback(t *testing.T) {
	// Create base service
	baseService := &EmbeddingService{
		config: &EmbeddingConfig{
			FallbackEnabled: true,
		},
		logger:      slog.Default(),
		providers:   make(map[string]EmbeddingProvider),
		cache:       NewInMemoryCache(100),
		redisCache:  &NoOpRedisCache{},
		rateLimiter: NewRateLimiter(1000, 10000),
		metrics: &EmbeddingMetrics{
			ModelStats: make(map[string]ModelUsageStats),
		},
	}

	// Add providers
	primaryProvider := NewMockProvider("primary", 0.001, 100*time.Millisecond)
	primaryProvider.SetFailureRate(1.0) // Always fails

	fallbackProvider := NewMockProvider("fallback", 0.0005, 150*time.Millisecond)

	baseService.providers["primary"] = primaryProvider
	baseService.providers["fallback"] = fallbackProvider

	// Create cost-aware service
	costConfig := &CostOptimizerConfig{
		OptimizationStrategy:    "balanced",
		EnableBudgetTracking:    true,
		DailyBudget:             10.0,
		CircuitBreakerThreshold: 3,
		CircuitBreakerTimeout:   5 * time.Second,
	}

	costAwareService := NewCostAwareEmbeddingService(baseService, costConfig)

	// Configure fallback chain
	costAwareService.fallbackManager.fallbackChains["primary"] = []string{"fallback"}

	// Test fallback behavior
	ctx := context.Background()
	request := &EmbeddingRequest{
		Texts:    []string{"Test text"},
		UseCache: false,
	}

	response, err := costAwareService.GenerateEmbeddingsOptimized(ctx, request)
	if err != nil {
		t.Fatalf("Failed to generate embeddings with fallback: %v", err)
	}

	// Should have used fallback provider
	if response.ModelUsed != "fallback" {
		t.Errorf("Expected fallback provider to be used, got %s", response.ModelUsed)
	}

	// Check fallback metrics
	fallbackHistory := costAwareService.fallbackManager.fallbackHistory["primary"]
	if fallbackHistory == nil || fallbackHistory.FallbackCount == 0 {
		t.Error("Expected fallback to be recorded")
	}
}

func TestCostAwareEmbeddingService_BudgetConstraints(t *testing.T) {
	// Create base service
	baseService := &EmbeddingService{
		config:      &EmbeddingConfig{},
		logger:      slog.Default(),
		providers:   make(map[string]EmbeddingProvider),
		cache:       NewInMemoryCache(100),
		redisCache:  &NoOpRedisCache{},
		rateLimiter: NewRateLimiter(1000, 10000),
		metrics: &EmbeddingMetrics{
			ModelStats: make(map[string]ModelUsageStats),
		},
	}

	// Add expensive provider
	expensiveProvider := NewMockProvider("expensive", 10.0, 100*time.Millisecond) // Very expensive
	baseService.providers["expensive"] = expensiveProvider

	// Create cost-aware service with tight budget
	costConfig := &CostOptimizerConfig{
		OptimizationStrategy:    "balanced",
		EnableBudgetTracking:    true,
		HourlyBudget:            0.01, // Very low budget
		DailyBudget:             0.10,
		CircuitBreakerThreshold: 3,
	}

	costAwareService := NewCostAwareEmbeddingService(baseService, costConfig)

	// Try to generate embeddings that would exceed budget
	ctx := context.Background()
	request := &EmbeddingRequest{
		Texts:    []string{strings.Repeat("Large text ", 1000)}, // Large text = many tokens
		UseCache: false,
	}

	_, err := costAwareService.GenerateEmbeddingsOptimized(ctx, request)
	if err == nil {
		t.Error("Expected budget constraint error")
	}

	if !strings.Contains(err.Error(), "budget") {
		t.Errorf("Expected budget-related error, got: %v", err)
	}
}

func TestCostAwareEmbeddingService_CircuitBreaker(t *testing.T) {
	// Create base service
	baseService := &EmbeddingService{
		config:      &EmbeddingConfig{},
		logger:      slog.Default(),
		providers:   make(map[string]EmbeddingProvider),
		cache:       NewInMemoryCache(100),
		redisCache:  &NoOpRedisCache{},
		rateLimiter: NewRateLimiter(1000, 10000),
		metrics: &EmbeddingMetrics{
			ModelStats: make(map[string]ModelUsageStats),
		},
	}

	// Add flaky provider
	flakyProvider := NewMockProvider("flaky", 0.001, 100*time.Millisecond)
	flakyProvider.SetFailureRate(0.8) // 80% failure rate

	baseService.providers["flaky"] = flakyProvider

	// Create cost-aware service
	costConfig := &CostOptimizerConfig{
		OptimizationStrategy:     "balanced",
		CircuitBreakerThreshold:  2, // Open after 2 failures
		CircuitBreakerTimeout:    1 * time.Second,
		CircuitBreakerMaxRetries: 1,
		EnableBudgetTracking:     true,
		DailyBudget:              10.0,
	}

	costAwareService := NewCostAwareEmbeddingService(baseService, costConfig)

	ctx := context.Background()
	request := &EmbeddingRequest{
		Texts:    []string{"Test"},
		UseCache: false,
	}

	// Make requests until circuit breaker opens
	failureCount := 0
	for i := 0; i < 5; i++ {
		_, err := costAwareService.GenerateEmbeddingsOptimized(ctx, request)
		if err != nil {
			failureCount++
		}

		// Check circuit breaker state after threshold
		if i >= costConfig.CircuitBreakerThreshold {
			cb := costAwareService.circuitBreakers["flaky"]
			if cb.GetState() != CircuitOpen {
				t.Errorf("Expected circuit breaker to be open after %d failures", failureCount)
			}
		}
	}

	// Wait for circuit breaker timeout
	time.Sleep(costConfig.CircuitBreakerTimeout + 100*time.Millisecond)

	// Circuit should be half-open now
	cb := costAwareService.circuitBreakers["flaky"]
	if cb.GetState() != CircuitHalfOpen {
		t.Error("Expected circuit breaker to be half-open after timeout")
	}
}

func TestCostAwareEmbeddingService_ConcurrentRequests(t *testing.T) {
	// Create base service
	baseService := &EmbeddingService{
		config:      &EmbeddingConfig{},
		logger:      slog.Default(),
		providers:   make(map[string]EmbeddingProvider),
		cache:       NewInMemoryCache(100),
		redisCache:  &NoOpRedisCache{},
		rateLimiter: NewRateLimiter(1000, 10000),
		metrics: &EmbeddingMetrics{
			ModelStats: make(map[string]ModelUsageStats),
		},
	}

	// Add multiple providers
	providers := []struct {
		name    string
		cost    float64
		latency time.Duration
	}{
		{"provider1", 0.001, 50 * time.Millisecond},
		{"provider2", 0.0005, 75 * time.Millisecond},
		{"provider3", 0.0008, 60 * time.Millisecond},
	}

	for _, p := range providers {
		provider := NewMockProvider(p.name, p.cost, p.latency)
		baseService.providers[p.name] = provider
	}

	// Create cost-aware service
	costConfig := &CostOptimizerConfig{
		OptimizationStrategy:    "balanced",
		EnableBudgetTracking:    true,
		DailyBudget:             100.0,
		CircuitBreakerThreshold: 5,
		RebalanceInterval:       1 * time.Minute,
	}

	costAwareService := NewCostAwareEmbeddingService(baseService, costConfig)

	// Run concurrent requests
	numRequests := 20
	var wg sync.WaitGroup
	results := make(chan *EmbeddingResponse, numRequests)
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ctx := context.Background()
			request := &EmbeddingRequest{
				Texts:     []string{fmt.Sprintf("Test text %d", idx)},
				UseCache:  false,
				RequestID: fmt.Sprintf("req_%d", idx),
			}

			response, err := costAwareService.GenerateEmbeddingsOptimized(ctx, request)
			if err != nil {
				errors <- err
			} else {
				results <- response
			}
		}(i)
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check results
	successCount := 0
	providerUsage := make(map[string]int)
	totalCost := 0.0

	for response := range results {
		successCount++
		providerUsage[response.ModelUsed]++
		totalCost += response.TokenUsage.EstimatedCost
	}

	// Check errors
	errorCount := 0
	for range errors {
		errorCount++
	}

	t.Logf("Concurrent test results: %d success, %d errors", successCount, errorCount)
	t.Logf("Provider usage: %v", providerUsage)
	t.Logf("Total cost: $%.4f", totalCost)

	// Verify load distribution
	if len(providerUsage) == 1 {
		t.Error("Expected requests to be distributed across multiple providers")
	}

	// Verify cost tracking
	metrics := costAwareService.budgetManager.currentSpend
	if metrics.DailySpend == 0 {
		t.Error("Expected daily spend to be tracked")
	}
}

func TestCostAwareEmbeddingService_ProviderScoring(t *testing.T) {
	// Create cost-aware service
	costConfig := &CostOptimizerConfig{
		OptimizationStrategy: "balanced",
		CostWeight:           0.4,
		PerformanceWeight:    0.3,
		QualityWeight:        0.3,
		DailyBudget:          10.0,
	}

	baseService := &EmbeddingService{
		config: &EmbeddingConfig{},
		logger: slog.Default(),
	}

	costAwareService := NewCostAwareEmbeddingService(baseService, costConfig)

	// Create mock provider
	provider := NewMockProvider("test", 0.001, 100*time.Millisecond)

	// Create provider metrics
	metrics := &ProviderMetrics{
		AverageCostPerToken: 0.001,
		AverageLatency:      100 * time.Millisecond,
		SuccessRate:         0.95,
		AverageQuality:      0.85,
		QualityVariance:     0.05,
		ConsecutiveFailures: 0,
		LastUsed:            time.Now(),
	}

	// Update provider monitor
	costAwareService.providerMonitor.metrics["test"] = metrics

	// Set budget manager state
	costAwareService.budgetManager.currentSpend = &BudgetSpend{
		DailySpend: 1.0, // $1 spent out of $10 budget
	}

	// Calculate provider score
	request := &EmbeddingRequest{
		Texts: []string{"Test text for scoring"},
	}

	score := costAwareService.calculateProviderScore("test", provider, request)

	// Verify score components
	if score.CostScore < 0 || score.CostScore > 1 {
		t.Errorf("Cost score out of range: %f", score.CostScore)
	}

	if score.PerformanceScore < 0 || score.PerformanceScore > 1 {
		t.Errorf("Performance score out of range: %f", score.PerformanceScore)
	}

	if score.QualityScore < 0 || score.QualityScore > 1 {
		t.Errorf("Quality score out of range: %f", score.QualityScore)
	}

	// Total score should be weighted sum
	expectedTotal := score.CostScore*0.4 + score.PerformanceScore*0.3 + score.QualityScore*0.3
	expectedTotal *= score.HealthScore // Health score is a multiplier

	if math.Abs(float64(score.TotalScore-expectedTotal)) > 0.01 {
		t.Errorf("Total score calculation error: got %f, expected %f", score.TotalScore, expectedTotal)
	}

	t.Logf("Provider score breakdown: Total=%.3f, Cost=%.3f, Performance=%.3f, Quality=%.3f, Health=%.3f",
		score.TotalScore, score.CostScore, score.PerformanceScore, score.QualityScore, score.HealthScore)
}

func BenchmarkCostAwareEmbeddingService(b *testing.B) {
	// Create base service with providers
	baseService := &EmbeddingService{
		config:      &EmbeddingConfig{},
		logger:      slog.Default(),
		providers:   make(map[string]EmbeddingProvider),
		cache:       NewInMemoryCache(1000),
		redisCache:  &NoOpRedisCache{},
		rateLimiter: NewRateLimiter(10000, 100000),
		metrics: &EmbeddingMetrics{
			ModelStats: make(map[string]ModelUsageStats),
		},
	}

	// Add providers
	for i := 0; i < 3; i++ {
		provider := NewMockProvider(
			fmt.Sprintf("provider%d", i),
			0.001*float64(i+1),
			time.Duration(50+i*25)*time.Millisecond,
		)
		baseService.providers[provider.GetName()] = provider
	}

	// Create cost-aware service
	costConfig := &CostOptimizerConfig{
		OptimizationStrategy:    "balanced",
		EnableBudgetTracking:    false, // Disable for benchmark
		CircuitBreakerThreshold: 10,
	}

	costAwareService := NewCostAwareEmbeddingService(baseService, costConfig)

	// Prepare request
	request := &EmbeddingRequest{
		Texts:    []string{"Benchmark text for embedding generation"},
		UseCache: true,
	}

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := costAwareService.GenerateEmbeddingsOptimized(ctx, request)
		if err != nil {
			b.Fatalf("Benchmark error: %v", err)
		}
	}

	b.StopTimer()

	// Report metrics
	metrics := costAwareService.costOptimizer.costHistory
	b.Logf("Total requests: %d", b.N)
	b.Logf("Provider usage: %v", costAwareService.providerMonitor.metrics)
}
