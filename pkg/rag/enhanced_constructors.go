//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Enhanced type constructors using composition/embedding patterns

// NewEnhancedLoadBalancer creates a new enhanced load balancer that embeds LoadBalancer
func NewEnhancedLoadBalancer(strategy string, providers []string) *EnhancedLoadBalancer {
	// Create base LoadBalancer
	baseLoadBalancer := NewLoadBalancer(strategy, providers)

	// Create enhanced version by embedding the base type
	enhanced := &EnhancedLoadBalancer{
		strategy:   baseLoadBalancer.strategy,
		providers:  baseLoadBalancer.providers,
		currentIdx: baseLoadBalancer.currentIdx,
		weights:    baseLoadBalancer.weights,
		mutex:      sync.RWMutex{}, // Initialize a new RWMutex
	}

	return enhanced
}

// NewEnhancedCostManager creates a new enhanced cost manager that embeds CostManager
func NewEnhancedCostManager(limits EnhancedCostLimits) *EnhancedCostManager {
	// Convert enhanced limits to base limits
	baseLimits := CostLimits{
		DailyLimit:     limits.DailyLimit,
		MonthlyLimit:   limits.MonthlyLimit,
		AlertThreshold: limits.AlertThreshold,
	}

	// Create base CostManager
	baseCostManager := NewCostManager(baseLimits)

	// Create enhanced version by embedding the base type data
	enhanced := &EnhancedCostManager{
		dailySpend:   baseCostManager.dailySpend,
		monthlySpend: baseCostManager.monthlySpend,
		limits:       limits,
		alerts:       []EnhancedCostAlert{}, // Convert alerts when needed
		mutex:        sync.RWMutex{},        // Initialize new mutex instead of copying
	}

	return enhanced
}

// NewEnhancedProviderConfig creates a new enhanced provider config that embeds ProviderConfig
func NewEnhancedProviderConfig(baseConfig ProviderConfig) *EnhancedProviderConfig {
	enhanced := &EnhancedProviderConfig{
		Name:           baseConfig.Name,
		Type:           "enhanced", // Default type
		APIKey:         baseConfig.APIKey,
		APIEndpoint:    baseConfig.APIEndpoint,
		ModelName:      baseConfig.ModelName,
		Priority:       baseConfig.Priority,
		MaxConcurrency: 10, // Default value
		RateLimit:      baseConfig.RateLimit,
		CostPerToken:   baseConfig.CostPerToken,
		Timeout:        30 * time.Second, // Default timeout
		Headers:        make(map[string]string),
		Enabled:        baseConfig.Enabled,
	}

	return enhanced
}

// CanAfford implements the cost checking interface for EnhancedCostManager
func (ecm *EnhancedCostManager) CanAfford(estimatedCost float64) bool {
	ecm.mutex.RLock()
	defer ecm.mutex.RUnlock()

	today := time.Now().Format("2006-01-02")
	thisMonth := time.Now().Format("2006-01")

	dailySpent := ecm.dailySpend[today]
	monthlySpent := ecm.monthlySpend[thisMonth]

	if ecm.limits.DailyLimit > 0 && dailySpent+estimatedCost > ecm.limits.DailyLimit {
		return false
	}

	if ecm.limits.MonthlyLimit > 0 && monthlySpent+estimatedCost > ecm.limits.MonthlyLimit {
		return false
	}

	return true
}

// RecordSpending implements the spending recording interface for EnhancedCostManager
func (ecm *EnhancedCostManager) RecordSpending(cost float64) {
	ecm.mutex.Lock()
	defer ecm.mutex.Unlock()

	today := time.Now().Format("2006-01-02")
	thisMonth := time.Now().Format("2006-01")

	ecm.dailySpend[today] += cost
	ecm.monthlySpend[thisMonth] += cost

	// Check for enhanced alerts
	ecm.checkEnhancedAlerts(today, thisMonth)
}

// checkEnhancedAlerts checks and generates enhanced cost alerts
func (ecm *EnhancedCostManager) checkEnhancedAlerts(today, thisMonth string) {
	dailySpent := ecm.dailySpend[today]
	monthlySpent := ecm.monthlySpend[thisMonth]

	// Daily alerts
	if ecm.limits.DailyLimit > 0 {
		dailyPercent := dailySpent / ecm.limits.DailyLimit
		if dailyPercent >= ecm.limits.AlertThreshold {
			level := "warning"
			if dailyPercent >= 0.95 {
				level = "critical"
			}

			alert := EnhancedCostAlert{
				Timestamp: time.Now(),
				Level:     level,
				Message:   "Daily spending has reached threshold",
				Amount:    dailySpent,
				Limit:     ecm.limits.DailyLimit,
			}
			ecm.alerts = append(ecm.alerts, alert)
		}
	}

	// Monthly alerts
	if ecm.limits.MonthlyLimit > 0 {
		monthlyPercent := monthlySpent / ecm.limits.MonthlyLimit
		if monthlyPercent >= ecm.limits.AlertThreshold {
			level := "warning"
			if monthlyPercent >= 0.95 {
				level = "critical"
			}

			alert := EnhancedCostAlert{
				Timestamp: time.Now(),
				Level:     level,
				Message:   "Monthly spending has reached threshold",
				Amount:    monthlySpent,
				Limit:     ecm.limits.MonthlyLimit,
			}
			ecm.alerts = append(ecm.alerts, alert)
		}
	}
}

// GetSummary returns enhanced cost summary
func (ecm *EnhancedCostManager) GetSummary() *EnhancedCostSummary {
	ecm.mutex.RLock()
	defer ecm.mutex.RUnlock()

	today := time.Now().Format("2006-01-02")
	thisMonth := time.Now().Format("2006-01")

	return &EnhancedCostSummary{
		DailySpending:   ecm.dailySpend[today],
		MonthlySpending: ecm.monthlySpend[thisMonth],
		DailyLimit:      ecm.limits.DailyLimit,
		MonthlyLimit:    ecm.limits.MonthlyLimit,
		Alerts:          ecm.alerts,
	}
}

// SelectProvider implements the provider selection interface for EnhancedLoadBalancer
func (elb *EnhancedLoadBalancer) SelectProvider(availableProviders []string, request *EmbeddingRequest) string {
	elb.mutex.Lock()
	defer elb.mutex.Unlock()

	if len(availableProviders) == 0 {
		return ""
	}

	if len(availableProviders) == 1 {
		return availableProviders[0]
	}

	switch elb.strategy {
	case "round_robin":
		return elb.roundRobin(availableProviders)
	case "least_cost":
		return elb.leastCost(availableProviders)
	case "fastest":
		return elb.fastest(availableProviders)
	case "quality":
		return elb.bestQuality(availableProviders)
	default:
		return elb.roundRobin(availableProviders)
	}
}

// Enhanced load balancer strategy methods
func (elb *EnhancedLoadBalancer) roundRobin(providers []string) string {
	if elb.currentIdx >= len(providers) {
		elb.currentIdx = 0
	}
	selected := providers[elb.currentIdx]
	elb.currentIdx++
	return selected
}

func (elb *EnhancedLoadBalancer) leastCost(providers []string) string {
	// Enhanced cost-based selection logic
	bestProvider := providers[0]
	bestCost := elb.weights[bestProvider]

	for _, provider := range providers[1:] {
		cost := elb.weights[provider]
		if cost < bestCost {
			bestCost = cost
			bestProvider = provider
		}
	}

	return bestProvider
}

func (elb *EnhancedLoadBalancer) fastest(providers []string) string {
	// Enhanced latency-based selection logic
	return providers[0] // Simplified for now
}

func (elb *EnhancedLoadBalancer) bestQuality(providers []string) string {
	// Enhanced quality-based selection logic
	return providers[0] // Simplified for now
}

// Enhanced provider constructor adapters

// NewOpenAIProvider creates an enhanced OpenAI provider adapter
func NewOpenAIProvider(config EnhancedProviderConfig) (EnhancedEmbeddingProvider, error) {
	// Convert enhanced config to base config
	baseConfig := ProviderConfig{
		Name:         config.Name,
		APIKey:       config.APIKey,
		APIEndpoint:  config.APIEndpoint,
		ModelName:    config.ModelName,
		Dimensions:   768,                          // Default dimensions
		MaxTokens:    config.MaxConcurrency * 1000, // Approximate
		CostPerToken: config.CostPerToken,
		RateLimit:    config.RateLimit,
		Priority:     config.Priority,
		Enabled:      config.Enabled,
		Healthy:      true,
		LastCheck:    time.Now(),
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	// Create base provider
	baseProvider := NewBasicOpenAIProvider(baseConfig, httpClient)

	// Return enhanced adapter
	return &EnhancedProviderAdapter{
		baseProvider: baseProvider,
		config:       config,
	}, nil
}

// NewAzureOpenAIProvider creates an enhanced Azure OpenAI provider adapter
func NewAzureOpenAIProvider(config EnhancedProviderConfig) (EnhancedEmbeddingProvider, error) {
	// Convert enhanced config to base config
	baseConfig := ProviderConfig{
		Name:         config.Name,
		APIKey:       config.APIKey,
		APIEndpoint:  config.APIEndpoint,
		ModelName:    config.ModelName,
		Dimensions:   768,                          // Default dimensions
		MaxTokens:    config.MaxConcurrency * 1000, // Approximate
		CostPerToken: config.CostPerToken,
		RateLimit:    config.RateLimit,
		Priority:     config.Priority,
		Enabled:      config.Enabled,
		Healthy:      true,
		LastCheck:    time.Now(),
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	// Create base provider
	baseProvider := NewBasicAzureOpenAIProvider(baseConfig, httpClient)

	// Return enhanced adapter
	return &EnhancedProviderAdapter{
		baseProvider: baseProvider,
		config:       config,
	}, nil
}

// NewHuggingFaceProvider creates an enhanced Hugging Face provider adapter
func NewHuggingFaceProvider(config EnhancedProviderConfig) (EnhancedEmbeddingProvider, error) {
	// Convert enhanced config to base config
	baseConfig := ProviderConfig{
		Name:         config.Name,
		APIKey:       config.APIKey,
		APIEndpoint:  config.APIEndpoint,
		ModelName:    config.ModelName,
		Dimensions:   768,                          // Default dimensions
		MaxTokens:    config.MaxConcurrency * 1000, // Approximate
		CostPerToken: config.CostPerToken,
		RateLimit:    config.RateLimit,
		Priority:     config.Priority,
		Enabled:      config.Enabled,
		Healthy:      true,
		LastCheck:    time.Now(),
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	// Create base provider
	baseProvider := NewBasicHuggingFaceProvider(baseConfig, httpClient)

	// Return enhanced adapter
	return &EnhancedProviderAdapter{
		baseProvider: baseProvider,
		config:       config,
	}, nil
}

// NewLocalEnhancedEmbeddingProvider creates an enhanced local provider
func NewLocalEnhancedEmbeddingProvider(config EnhancedProviderConfig) (EnhancedEmbeddingProvider, error) {
	// For local provider, create a simple implementation
	return &LocalEnhancedProvider{
		config: config,
	}, nil
}

// EnhancedProviderAdapter adapts base providers to enhanced interface
type EnhancedProviderAdapter struct {
	baseProvider BasicEmbeddingProvider
	config       EnhancedProviderConfig
}

// GenerateEmbeddings implements EnhancedEmbeddingProvider
func (epa *EnhancedProviderAdapter) GenerateEmbeddings(ctx context.Context, texts []string) (*EmbeddingResponse, error) {
	// Call base provider
	embeddings, tokenUsage, err := epa.baseProvider.GenerateEmbeddings(ctx, texts)
	if err != nil {
		return nil, err
	}

	// Convert to enhanced response
	response := &EmbeddingResponse{
		Embeddings: embeddings,
		TokenUsage: tokenUsage,
		ModelUsed:  epa.config.ModelName,
	}

	return response, nil
}

// GetModelInfo implements EnhancedEmbeddingProvider
func (epa *EnhancedProviderAdapter) GetModelInfo() ModelInfo {
	baseConfig := epa.baseProvider.GetConfig()
	return ModelInfo{
		Name:         baseConfig.ModelName,
		Dimensions:   baseConfig.Dimensions,
		MaxTokens:    baseConfig.MaxTokens,
		CostPerToken: baseConfig.CostPerToken,
		Provider:     baseConfig.Name,
		ApiVersion:   "v1",
		Capabilities: []string{"embedding"},
	}
}

// EstimateCost implements EnhancedEmbeddingProvider
func (epa *EnhancedProviderAdapter) EstimateCost(texts []string) float64 {
	// Estimate token count (rough approximation)
	totalTokens := 0
	for _, text := range texts {
		totalTokens += len(text) / 4 // Rough token estimate
	}

	return epa.baseProvider.GetCostEstimate(totalTokens)
}

// IsHealthy implements EnhancedEmbeddingProvider
func (epa *EnhancedProviderAdapter) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return epa.baseProvider.HealthCheck(ctx) == nil
}

// GetLatency implements EnhancedEmbeddingProvider
func (epa *EnhancedProviderAdapter) GetLatency() time.Duration {
	// Simple latency check
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	epa.baseProvider.HealthCheck(ctx)
	return time.Since(start)
}

// GetName implements EnhancedEmbeddingProvider
func (epa *EnhancedProviderAdapter) GetName() string {
	return epa.config.Name
}

// LocalEnhancedProvider implements a simple local provider
type LocalEnhancedProvider struct {
	config EnhancedProviderConfig
}

// GenerateEmbeddings implements EnhancedEmbeddingProvider
func (lep *LocalEnhancedProvider) GenerateEmbeddings(ctx context.Context, texts []string) (*EmbeddingResponse, error) {
	// Simple stub implementation - generates random embeddings
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		embedding := make([]float32, 768) // Standard embedding size
		for j := range embedding {
			embedding[j] = float32((i+j)%100) / 100.0 // Simple deterministic pattern
		}
		embeddings[i] = embedding
	}

	tokenUsage := &TokenUsage{
		PromptTokens:  len(texts) * 10, // Rough estimate
		TotalTokens:   len(texts) * 10,
		EstimatedCost: float64(len(texts)) * lep.config.CostPerToken,
	}

	return &EmbeddingResponse{
		Embeddings: embeddings,
		TokenUsage: *tokenUsage, // Dereference the pointer
		ModelUsed:  lep.config.ModelName,
	}, nil
}

// GetModelInfo implements EnhancedEmbeddingProvider
func (lep *LocalEnhancedProvider) GetModelInfo() ModelInfo {
	return ModelInfo{
		Name:         lep.config.ModelName,
		Dimensions:   768,
		MaxTokens:    1000,
		CostPerToken: lep.config.CostPerToken,
		Provider:     lep.config.Name,
		ApiVersion:   "local",
		Capabilities: []string{"embedding"},
	}
}

// EstimateCost implements EnhancedEmbeddingProvider
func (lep *LocalEnhancedProvider) EstimateCost(texts []string) float64 {
	return float64(len(texts)) * lep.config.CostPerToken
}

// IsHealthy implements EnhancedEmbeddingProvider
func (lep *LocalEnhancedProvider) IsHealthy() bool {
	return true // Local provider is always healthy
}

// GetLatency implements EnhancedEmbeddingProvider
func (lep *LocalEnhancedProvider) GetLatency() time.Duration {
	return 1 * time.Millisecond // Very fast local provider
}

// GetName implements EnhancedEmbeddingProvider
func (lep *LocalEnhancedProvider) GetName() string {
	return lep.config.Name
}

// ProviderHealthAdapter adapts EnhancedEmbeddingProvider to EmbeddingProvider for health monitoring
type ProviderHealthAdapter struct {
	provider EnhancedEmbeddingProvider
}

// GetEmbeddings implements EmbeddingProvider for health monitoring
func (pha *ProviderHealthAdapter) GetEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	response, err := pha.provider.GenerateEmbeddings(ctx, texts)
	if err != nil {
		return nil, err
	}

	// Convert [][]float32 to [][]float64
	embeddings := make([][]float64, len(response.Embeddings))
	for i, embedding := range response.Embeddings {
		embeddings[i] = make([]float64, len(embedding))
		for j, val := range embedding {
			embeddings[i][j] = float64(val)
		}
	}

	return embeddings, nil
}

// IsHealthy implements EmbeddingProvider
func (pha *ProviderHealthAdapter) IsHealthy() bool {
	return pha.provider.IsHealthy()
}

// GetLatency implements EmbeddingProvider
func (pha *ProviderHealthAdapter) GetLatency() time.Duration {
	return pha.provider.GetLatency()
}

// GenerateEmbedding implements EmbeddingProvider (from enhanced_rag_integration.go)
func (pha *ProviderHealthAdapter) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	response, err := pha.provider.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}

	if len(response.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return response.Embeddings[0], nil
}

// GenerateBatchEmbeddings implements EmbeddingProvider (from enhanced_rag_integration.go)
func (pha *ProviderHealthAdapter) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	response, err := pha.provider.GenerateEmbeddings(ctx, texts)
	if err != nil {
		return nil, err
	}

	return response.Embeddings, nil
}

// GetDimensions implements EmbeddingProvider (from enhanced_rag_integration.go)
func (pha *ProviderHealthAdapter) GetDimensions() int {
	// Try to get dimensions from model info if available
	modelInfo := pha.provider.GetModelInfo()
	if modelInfo.Dimensions > 0 {
		return modelInfo.Dimensions
	}

	// Default fallback - most common embedding dimensions
	return 384
}

// Enhanced types that were referenced but not defined
type EnhancedCostSummary struct {
	DailySpending   float64             `json:"daily_spending"`
	MonthlySpending float64             `json:"monthly_spending"`
	DailyLimit      float64             `json:"daily_limit"`
	MonthlyLimit    float64             `json:"monthly_limit"`
	Alerts          []EnhancedCostAlert `json:"alerts"`
}
