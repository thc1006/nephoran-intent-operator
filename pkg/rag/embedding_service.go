//go:build !disable_rag && !test

package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// EmbeddingService provides embedding generation for telecom documents
type EmbeddingService struct {
	config        *EmbeddingConfig
	logger        *slog.Logger
	metrics       *EmbeddingMetrics
	httpClient    *http.Client
	rateLimiter   *RateLimiter
	cache         shared.EmbeddingCacheInterface
	redisCache    RedisEmbeddingCache
	providers     map[string]BasicEmbeddingProvider
	fallbackChain []string
	costTracker   *CostTracker
	qualityAssess *QualityAssessment
	mutex         sync.RWMutex
}

// EmbeddingConfig holds configuration for the embedding service
type EmbeddingConfig struct {
	// Provider configuration
	Provider    string `json:"provider"` // "openai", "azure", "local", etc.
	APIKey      string `json:"api_key"`
	APIEndpoint string `json:"api_endpoint"`
	ModelName   string `json:"model_name"`

	// Model-specific settings
	Dimensions     int `json:"dimensions"`      // Embedding dimensions
	MaxTokens      int `json:"max_tokens"`      // Max tokens per request
	BatchSize      int `json:"batch_size"`      // Documents per batch
	MaxConcurrency int `json:"max_concurrency"` // Concurrent requests

	// Request configuration
	RequestTimeout time.Duration `json:"request_timeout"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryDelay     time.Duration `json:"retry_delay"`

	// Rate limiting
	RateLimit      int `json:"rate_limit"`       // Requests per minute
	TokenRateLimit int `json:"token_rate_limit"` // Tokens per minute

	// Quality control
	MinTextLength   int  `json:"min_text_length"`   // Minimum text length for embedding
	MaxTextLength   int  `json:"max_text_length"`   // Maximum text length for embedding
	NormalizeText   bool `json:"normalize_text"`    // Normalize text before embedding
	RemoveStopWords bool `json:"remove_stop_words"` // Remove common stop words

	// Caching
	EnableCaching    bool          `json:"enable_caching"`
	CacheTTL         time.Duration `json:"cache_ttl"`
	CacheDirectory   string        `json:"cache_directory"`
	EnableRedisCache bool          `json:"enable_redis_cache"`
	RedisAddr        string        `json:"redis_addr"`
	RedisPassword    string        `json:"redis_password"`
	RedisDB          int           `json:"redis_db"`
	L1CacheSize      int64         `json:"l1_cache_size"`    // L1 in-memory cache size
	L2CacheEnabled   bool          `json:"l2_cache_enabled"` // L2 Redis cache

	// Multi-provider configuration
	Providers           []ProviderConfig `json:"providers"`             // Multiple provider configurations
	FallbackEnabled     bool             `json:"fallback_enabled"`      // Enable fallback to other providers
	FallbackOrder       []string         `json:"fallback_order"`        // Order of fallback providers
	LoadBalancing       string           `json:"load_balancing"`        // "round_robin", "least_cost", "fastest"
	HealthCheckInterval time.Duration    `json:"health_check_interval"` // Provider health check interval

	// Cost management
	EnableCostTracking bool    `json:"enable_cost_tracking"`
	DailyCostLimit     float64 `json:"daily_cost_limit"`     // Daily cost limit in USD
	MonthlyCostLimit   float64 `json:"monthly_cost_limit"`   // Monthly cost limit in USD
	CostAlertThreshold float64 `json:"cost_alert_threshold"` // Alert when cost reaches % of limit

	// Quality control
	EnableQualityCheck bool    `json:"enable_quality_check"`
	MinQualityScore    float64 `json:"min_quality_score"`    // Minimum embedding quality score
	QualityCheckSample int     `json:"quality_check_sample"` // Sample size for quality checks

	// Telecom-specific preprocessing
	TelecomPreprocessing   bool    `json:"telecom_preprocessing"` // Apply telecom-specific preprocessing
	PreserveTechnicalTerms bool    `json:"preserve_technical_terms"`
	TechnicalTermWeighting float64 `json:"technical_term_weighting"` // Weight boost for technical terms

	// Performance monitoring
	EnableMetrics   bool          `json:"enable_metrics"`
	MetricsInterval time.Duration `json:"metrics_interval"`
}

// EmbeddingRequest represents a request for generating embeddings
type EmbeddingRequest struct {
	Texts     []string               `json:"texts"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Priority  int                    `json:"priority"` // Higher = more important
	ChunkIDs  []string               `json:"chunk_ids,omitempty"`
	UseCache  bool                   `json:"use_cache"`
	Model     string                 `json:"model,omitempty"`
	BatchSize int                    `json:"batch_size,omitempty"`
	Timeout   time.Duration          `json:"timeout,omitempty"`
}

// EmbeddingResponse represents the response from embedding generation
type EmbeddingResponse struct {
	Embeddings     [][]float32            `json:"embeddings"`
	TokenUsage     TokenUsage             `json:"token_usage"`
	ProcessingTime time.Duration          `json:"processing_time"`
	CacheHits      int                    `json:"cache_hits"`
	CacheMisses    int                    `json:"cache_misses"`
	RequestID      string                 `json:"request_id"`
	ModelUsed      string                 `json:"model_used"`
	Model          string                 `json:"model"`
	Provider       string                 `json:"provider"`
	CacheHitRate   float64                `json:"cache_hit_rate"`
	QualityScore   float64                `json:"quality_score"`
	Cost           float64                `json:"cost"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens  int     `json:"prompt_tokens"`
	TotalTokens   int     `json:"total_tokens"`
	EstimatedCost float64 `json:"estimated_cost"`
}

// EmbeddingMetrics tracks embedding service performance
type EmbeddingMetrics struct {
	TotalRequests      int64   `json:"total_requests"`
	SuccessfulRequests int64   `json:"successful_requests"`
	FailedRequests     int64   `json:"failed_requests"`
	TotalTexts         int64   `json:"total_texts"`
	TotalTokens        int64   `json:"total_tokens"`
	TotalCost          float64 `json:"total_cost"`

	AverageLatency    time.Duration `json:"average_latency"`
	AverageTextLength float64       `json:"average_text_length"`

	CacheStats struct {
		TotalLookups int64   `json:"total_lookups"`
		CacheHits    int64   `json:"cache_hits"`
		CacheMisses  int64   `json:"cache_misses"`
		HitRate      float64 `json:"hit_rate"`
	} `json:"cache_stats"`

	RateLimitingStats struct {
		RateLimitHits   int64         `json:"rate_limit_hits"`
		TotalWaitTime   time.Duration `json:"total_wait_time"`
		AverageWaitTime time.Duration `json:"average_wait_time"`
	} `json:"rate_limiting_stats"`

	ModelStats map[string]ModelUsageStats `json:"model_stats"`

	LastUpdated time.Time `json:"last_updated"`
	mutex       sync.RWMutex
}

// ModelUsageStats tracks usage per model
type ModelUsageStats struct {
	RequestCount int64   `json:"request_count"`
	TokenCount   int64   `json:"token_count"`
	TotalCost    float64 `json:"total_cost"`
	SuccessRate  float64 `json:"success_rate"`
}

// RateLimiter manages API rate limiting
type RateLimiter struct {
	requestTokens chan struct{}
	tokenBucket   chan int
	lastRefill    time.Time
	mutex         sync.Mutex
}

// EmbeddingCache interface for caching embeddings
// Use consolidated types from pkg/shared
type EmbeddingCache = shared.EmbeddingCacheInterface
type CacheStats = shared.EmbeddingCacheStats

// InMemoryCache is a simple in-memory cache implementation
type InMemoryCache struct {
	data    map[string]cacheEntry
	stats   CacheStats
	mutex   sync.RWMutex
	maxSize int64
}

type cacheEntry struct {
	embedding []float32
	expiresAt time.Time
}

// OpenAIEmbeddingRequest represents the request format for OpenAI API
type OpenAIEmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// ProviderConfig holds configuration for a specific embedding provider
type ProviderConfig struct {
	Name         string    `json:"name"` // Provider name (openai, azure, local)
	APIKey       string    `json:"api_key"`
	APIEndpoint  string    `json:"api_endpoint"`
	ModelName    string    `json:"model_name"`
	Dimensions   int       `json:"dimensions"`
	MaxTokens    int       `json:"max_tokens"`
	CostPerToken float64   `json:"cost_per_token"` // Cost per 1K tokens
	RateLimit    int       `json:"rate_limit"`     // Requests per minute
	Priority     int       `json:"priority"`       // Priority for load balancing
	Enabled      bool      `json:"enabled"`
	Healthy      bool      `json:"healthy"`
	LastCheck    time.Time `json:"last_check"`
}

// BasicEmbeddingProvider interface for different embedding providers
type BasicEmbeddingProvider interface {
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, TokenUsage, error)
	GetConfig() ProviderConfig
	HealthCheck(ctx context.Context) error
	GetCostEstimate(tokenCount int) float64
	GetName() string
	IsHealthy() bool
	GetLatency() time.Duration
}

// RedisEmbeddingCache interface for Redis-based caching
type RedisEmbeddingCache interface {
	Get(key string) ([]float32, bool, error)
	Set(key string, embedding []float32, ttl time.Duration) error
	Delete(key string) error
	Clear() error
	Stats() CacheStats
	Close() error
}

// CostTracker manages embedding cost tracking and limits
type CostTracker struct {
	dailyCosts   map[string]float64 // date -> cost
	monthlyCosts map[string]float64 // month -> cost
	limits       CostLimits
	mutex        sync.RWMutex
	alerts       []CostAlert
}

// CostLimits defines cost limits and thresholds
type CostLimits struct {
	DailyLimit     float64
	MonthlyLimit   float64
	AlertThreshold float64
}

// CostAlert represents a cost alert
type CostAlert struct {
	Timestamp time.Time
	Type      string // "daily", "monthly", "threshold"
	Amount    float64
	Limit     float64
	Message   string
}

// QualityAssessment manages embedding quality assessment
type QualityAssessment struct {
	config        QualityConfig
	metrics       QualityMetrics
	referenceEmbs map[string][]float32 // Reference embeddings for quality check
	mutex         sync.RWMutex
}

// QualityConfig holds quality assessment configuration
type QualityConfig struct {
	Enabled             bool
	MinScore            float64
	SampleSize          int
	ReferenceTexts      []string
	SimilarityThreshold float64
}

// QualityMetrics tracks embedding quality metrics
type QualityMetrics struct {
	AverageScore float64
	MinScore     float64
	MaxScore     float64
	FailedChecks int64
	TotalChecks  int64
	LastCheck    time.Time
}

// OpenAIEmbeddingResponse represents the response format from OpenAI API
type OpenAIEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(config *EmbeddingConfig) *EmbeddingService {
	if config == nil {
		config = getDefaultEmbeddingConfig()
	}

	// Create HTTP client with appropriate settings
	httpClient := &http.Client{
		Timeout: config.RequestTimeout,
		Transport: &http.Transport{
			MaxIdleConns:       100,
			MaxConnsPerHost:    config.MaxConcurrency,
			IdleConnTimeout:    90 * time.Second,
			DisableCompression: false,
		},
	}

	// Create rate limiter
	rateLimiter := NewRateLimiter(config.RateLimit, config.TokenRateLimit)

	// Create L1 cache (in-memory)
	var cache EmbeddingCache
	if config.EnableCaching {
		cache = NewInMemoryCache(config.L1CacheSize)
	} else {
		cache = &NoOpCache{}
	}

	// Create L2 cache (Redis) if enabled
	var redisCache RedisEmbeddingCache
	if config.EnableRedisCache {
		redisCache = NewRedisEmbeddingCache(config.RedisAddr, config.RedisPassword, config.RedisDB)
	} else {
		redisCache = &NoOpRedisCache{}
	}

	// Initialize providers
	providers := make(map[string]BasicEmbeddingProvider)
	for _, providerConfig := range config.Providers {
		if providerConfig.Enabled {
			provider := createProvider(providerConfig, httpClient)
			providers[providerConfig.Name] = provider
		}
	}

	// If no providers configured, create default OpenAI provider
	if len(providers) == 0 && config.Provider != "" {
		defaultConfig := ProviderConfig{
			Name:         config.Provider,
			APIKey:       config.APIKey,
			APIEndpoint:  config.APIEndpoint,
			ModelName:    config.ModelName,
			Dimensions:   config.Dimensions,
			MaxTokens:    config.MaxTokens,
			CostPerToken: 0.00013, // Default OpenAI cost
			RateLimit:    config.RateLimit,
			Priority:     1,
			Enabled:      true,
			Healthy:      true,
		}
		providers[config.Provider] = createProvider(defaultConfig, httpClient)
	}

	// Create cost tracker
	costTracker := NewCostTracker(CostLimits{
		DailyLimit:     config.DailyCostLimit,
		MonthlyLimit:   config.MonthlyCostLimit,
		AlertThreshold: config.CostAlertThreshold,
	})

	// Create quality assessment
	qualityAssess := NewQualityAssessment(QualityConfig{
		Enabled:             config.EnableQualityCheck,
		MinScore:            config.MinQualityScore,
		SampleSize:          config.QualityCheckSample,
		SimilarityThreshold: 0.8,
	})

	service := &EmbeddingService{
		config: config,
		logger: slog.Default().With("component", "embedding-service"),
		metrics: &EmbeddingMetrics{
			ModelStats:  make(map[string]ModelUsageStats),
			LastUpdated: time.Now(),
		},
		httpClient:    httpClient,
		rateLimiter:   rateLimiter,
		cache:         cache,
		redisCache:    redisCache,
		providers:     providers,
		fallbackChain: config.FallbackOrder,
		costTracker:   costTracker,
		qualityAssess: qualityAssess,
	}

	// Start background services
	if config.EnableMetrics {
		go service.startMetricsCollection()
	}
	if config.HealthCheckInterval > 0 {
		go service.startHealthChecks()
	}
	if config.EnableCostTracking {
		go service.startCostMonitoring()
	}

	return service
}

// GenerateEmbeddings generates embeddings for the provided texts using multi-provider approach
func (es *EmbeddingService) GenerateEmbeddings(ctx context.Context, request *EmbeddingRequest) (*EmbeddingResponse, error) {
	startTime := time.Now()

	if request == nil {
		return nil, fmt.Errorf("embedding request cannot be nil")
	}

	if len(request.Texts) == 0 {
		return nil, fmt.Errorf("no texts provided for embedding")
	}

	// Check cost limits before processing
	if es.config.EnableCostTracking {
		estimatedCost := es.estimateCost(request.Texts)
		if !es.costTracker.CanAfford(estimatedCost) {
			return nil, fmt.Errorf("cost limit exceeded: estimated cost $%.4f would exceed limits", estimatedCost)
		}
	}

	es.logger.Info("Generating embeddings",
		"text_count", len(request.Texts),
		"request_id", request.RequestID,
		"use_cache", request.UseCache,
	)

	// Preprocess texts
	processedTexts := es.preprocessTexts(request.Texts)

	// Check L1 (memory) and L2 (Redis) cache for existing embeddings
	var embeddings [][]float32
	var cacheHits, cacheMisses int
	var textsToEmbed []string
	var textIndices []int

	if request.UseCache {
		embeddings = make([][]float32, len(processedTexts))

		for i, text := range processedTexts {
			cacheKey := es.generateCacheKey(text)
			var cached []float32
			var found bool

			// Try L1 cache first (in-memory)
			if es.config.EnableCaching {
				cached, found = es.cache.Get(cacheKey)
			}

			// Try L2 cache (Redis) if L1 miss
			if !found && es.config.EnableRedisCache {
				var err error
				cached, found, err = es.redisCache.Get(cacheKey)
				if err != nil {
					es.logger.Warn("Redis cache error", "error", err)
				}
				// If found in L2, promote to L1
				if found && es.config.EnableCaching {
					es.cache.Set(cacheKey, cached, es.config.CacheTTL)
				}
			}

			if found {
				embeddings[i] = cached
				cacheHits++
			} else {
				textsToEmbed = append(textsToEmbed, text)
				textIndices = append(textIndices, i)
				cacheMisses++
			}
		}
	} else {
		textsToEmbed = processedTexts
		textIndices = make([]int, len(processedTexts))
		for i := range textIndices {
			textIndices[i] = i
		}
		embeddings = make([][]float32, len(processedTexts))
	}

	// Generate embeddings for texts not in cache using selected provider
	var tokenUsage TokenUsage
	var usedProvider string
	if len(textsToEmbed) > 0 {
		newEmbeddings, usage, provider, err := es.generateEmbeddingsWithProvider(ctx, textsToEmbed)
		if err != nil {
			es.updateMetrics(func(m *EmbeddingMetrics) {
				m.FailedRequests++
			})
			return nil, fmt.Errorf("failed to generate embeddings: %w", err)
		}

		tokenUsage = usage
		usedProvider = provider

		// Quality assessment if enabled
		if es.config.EnableQualityCheck {
			qualityScore := es.qualityAssess.AssessQuality(newEmbeddings, textsToEmbed)
			if qualityScore < es.config.MinQualityScore {
				es.logger.Warn("Low quality embeddings detected", "score", qualityScore, "threshold", es.config.MinQualityScore)
			}
		}

		// Place new embeddings in the correct positions and cache them
		for i, newEmbedding := range newEmbeddings {
			originalIndex := textIndices[i]
			embeddings[originalIndex] = newEmbedding

			// Cache in both L1 and L2 if enabled
			if request.UseCache {
				cacheKey := es.generateCacheKey(textsToEmbed[i])

				// Cache in L1 (memory)
				if es.config.EnableCaching {
					if err := es.cache.Set(cacheKey, newEmbedding, es.config.CacheTTL); err != nil {
						es.logger.Warn("Failed to cache embedding in L1", "error", err)
					}
				}

				// Cache in L2 (Redis)
				if es.config.EnableRedisCache {
					if err := es.redisCache.Set(cacheKey, newEmbedding, es.config.CacheTTL); err != nil {
						es.logger.Warn("Failed to cache embedding in L2", "error", err)
					}
				}
			}
		}

		// Track cost
		if es.config.EnableCostTracking {
			es.costTracker.RecordCost(tokenUsage.EstimatedCost, usedProvider)
		}
	}

	processingTime := time.Since(startTime)

	// Update metrics
	es.updateMetrics(func(m *EmbeddingMetrics) {
		m.TotalRequests++
		m.SuccessfulRequests++
		m.TotalTexts += int64(len(request.Texts))
		m.TotalTokens += int64(tokenUsage.TotalTokens)
		m.TotalCost += tokenUsage.EstimatedCost

		if m.SuccessfulRequests > 0 {
			m.AverageLatency = (m.AverageLatency*time.Duration(m.SuccessfulRequests-1) + processingTime) / time.Duration(m.SuccessfulRequests)
		} else {
			m.AverageLatency = processingTime
		}

		m.CacheStats.TotalLookups += int64(len(request.Texts))
		m.CacheStats.CacheHits += int64(cacheHits)
		m.CacheStats.CacheMisses += int64(cacheMisses)
		if m.CacheStats.TotalLookups > 0 {
			m.CacheStats.HitRate = float64(m.CacheStats.CacheHits) / float64(m.CacheStats.TotalLookups)
		}

		// Update model stats for the used provider
		modelName := usedProvider
		if modelName == "" {
			modelName = es.config.ModelName
		}
		modelStats := m.ModelStats[modelName]
		modelStats.RequestCount++
		modelStats.TokenCount += int64(tokenUsage.TotalTokens)
		modelStats.TotalCost += tokenUsage.EstimatedCost
		if modelStats.RequestCount > 0 {
			modelStats.SuccessRate = float64(m.SuccessfulRequests) / float64(m.TotalRequests)
		}
		m.ModelStats[modelName] = modelStats

		m.LastUpdated = time.Now()
	})

	response := &EmbeddingResponse{
		Embeddings:     embeddings,
		TokenUsage:     tokenUsage,
		ProcessingTime: processingTime,
		CacheHits:      cacheHits,
		CacheMisses:    cacheMisses,
		RequestID:      request.RequestID,
		ModelUsed:      usedProvider,
		Metadata:       request.Metadata,
	}

	es.logger.Info("Embeddings generated successfully",
		"text_count", len(request.Texts),
		"processing_time", processingTime,
		"cache_hits", cacheHits,
		"cache_misses", cacheMisses,
		"provider", usedProvider,
		"cost", tokenUsage.EstimatedCost,
	)

	return response, nil
}

// generateEmbeddingsWithProvider generates embeddings using the best available provider
func (es *EmbeddingService) generateEmbeddingsWithProvider(ctx context.Context, texts []string) ([][]float32, TokenUsage, string, error) {
	// Select best provider based on load balancing strategy
	provider, err := es.selectBestProvider(ctx, texts)
	if err != nil {
		return nil, TokenUsage{}, "", fmt.Errorf("no available providers: %w", err)
	}

	// Try primary provider
	embeddings, usage, err := es.generateEmbeddingsWithSpecificProvider(ctx, texts, provider)
	if err == nil {
		return embeddings, usage, provider.GetName(), nil
	}

	es.logger.Warn("Primary provider failed, trying fallback", "provider", provider.GetName(), "error", err)

	// Try fallback providers if enabled
	if es.config.FallbackEnabled {
		for _, fallbackName := range es.fallbackChain {
			if fallbackName == provider.GetName() {
				continue // Skip the failed provider
			}

			fallbackProvider, exists := es.providers[fallbackName]
			if !exists || !fallbackProvider.GetConfig().Enabled {
				continue
			}

			// Check provider health
			if healthErr := fallbackProvider.HealthCheck(ctx); healthErr != nil {
				es.logger.Warn("Fallback provider unhealthy, skipping", "provider", fallbackName, "error", healthErr)
				continue
			}

			embeddings, usage, fallbackErr := es.generateEmbeddingsWithSpecificProvider(ctx, texts, fallbackProvider)
			if fallbackErr == nil {
				es.logger.Info("Fallback provider succeeded", "provider", fallbackName)
				return embeddings, usage, fallbackProvider.GetName(), nil
			}

			es.logger.Warn("Fallback provider failed", "provider", fallbackName, "error", fallbackErr)
		}
	}

	return nil, TokenUsage{}, "", fmt.Errorf("all providers failed, last error: %w", err)
}

// generateEmbeddingsWithSpecificProvider generates embeddings using a specific provider
func (es *EmbeddingService) generateEmbeddingsWithSpecificProvider(ctx context.Context, texts []string, provider BasicEmbeddingProvider) ([][]float32, TokenUsage, error) {
	// Process in smaller batches to respect API limits
	var allEmbeddings [][]float32
	var totalUsage TokenUsage
	batchSize := es.config.BatchSize

	// Adjust batch size based on provider configuration
	providerConfig := provider.GetConfig()
	if providerConfig.MaxTokens > 0 && batchSize > providerConfig.MaxTokens/100 { // Rough estimation
		batchSize = providerConfig.MaxTokens / 100
	}

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]

		// Wait for rate limiting
		if err := es.rateLimiter.Wait(ctx, len(batch)); err != nil {
			return nil, totalUsage, fmt.Errorf("rate limiting error: %w", err)
		}

		// Generate embeddings for this batch
		embeddings, usage, err := provider.GenerateEmbeddings(ctx, batch)
		if err != nil {
			return nil, totalUsage, fmt.Errorf("provider API call failed for batch %d-%d: %w", i, end, err)
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
		totalUsage.PromptTokens += usage.PromptTokens
		totalUsage.TotalTokens += usage.TotalTokens
		totalUsage.EstimatedCost += usage.EstimatedCost
	}

	return allEmbeddings, totalUsage, nil
}

// selectBestProvider selects the best provider based on load balancing strategy
func (es *EmbeddingService) selectBestProvider(ctx context.Context, texts []string) (BasicEmbeddingProvider, error) {
	var availableProviders []BasicEmbeddingProvider

	// Filter healthy and enabled providers
	for _, provider := range es.providers {
		config := provider.GetConfig()
		if !config.Enabled || !config.Healthy {
			continue
		}

		// Quick health check if needed
		if time.Since(config.LastCheck) > es.config.HealthCheckInterval {
			if err := provider.HealthCheck(ctx); err != nil {
				es.logger.Warn("Provider health check failed", "provider", provider.GetName(), "error", err)
				continue
			}
		}

		availableProviders = append(availableProviders, provider)
	}

	if len(availableProviders) == 0 {
		return nil, fmt.Errorf("no healthy providers available")
	}

	// Select based on load balancing strategy
	switch es.config.LoadBalancing {
	case "least_cost":
		return es.selectLeastCostProvider(availableProviders, texts)
	case "fastest":
		return es.selectFastestProvider(availableProviders)
	case "round_robin":
		return es.selectRoundRobinProvider(availableProviders)
	default:
		// Default to highest priority
		return es.selectHighestPriorityProvider(availableProviders)
	}
}

// selectLeastCostProvider selects the provider with lowest estimated cost
func (es *EmbeddingService) selectLeastCostProvider(providers []BasicEmbeddingProvider, texts []string) (BasicEmbeddingProvider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	estimatedTokens := es.estimateTokenCount(texts)
	bestProvider := providers[0]
	lowestCost := bestProvider.GetCostEstimate(estimatedTokens)

	for _, provider := range providers[1:] {
		cost := provider.GetCostEstimate(estimatedTokens)
		if cost < lowestCost {
			lowestCost = cost
			bestProvider = provider
		}
	}

	return bestProvider, nil
}

// selectFastestProvider selects the provider with best performance metrics
func (es *EmbeddingService) selectFastestProvider(providers []BasicEmbeddingProvider) (BasicEmbeddingProvider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// For now, select based on priority (in real implementation, use historical latency)
	return es.selectHighestPriorityProvider(providers)
}

// selectRoundRobinProvider selects providers in round-robin fashion
func (es *EmbeddingService) selectRoundRobinProvider(providers []BasicEmbeddingProvider) (BasicEmbeddingProvider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// Simple round-robin based on request count
	minRequests := int64(-1)
	var selectedProvider BasicEmbeddingProvider

	for _, provider := range providers {
		modelStats, exists := es.metrics.ModelStats[provider.GetName()]
		if !exists {
			return provider, nil // Use provider with no previous requests
		}

		if minRequests == -1 || modelStats.RequestCount < minRequests {
			minRequests = modelStats.RequestCount
			selectedProvider = provider
		}
	}

	if selectedProvider == nil {
		return providers[0], nil
	}

	return selectedProvider, nil
}

// selectHighestPriorityProvider selects the provider with highest priority
func (es *EmbeddingService) selectHighestPriorityProvider(providers []BasicEmbeddingProvider) (BasicEmbeddingProvider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	bestProvider := providers[0]
	highestPriority := bestProvider.GetConfig().Priority

	for _, provider := range providers[1:] {
		priority := provider.GetConfig().Priority
		if priority > highestPriority {
			highestPriority = priority
			bestProvider = provider
		}
	}

	return bestProvider, nil
}

// estimateTokenCount estimates the number of tokens for texts
func (es *EmbeddingService) estimateTokenCount(texts []string) int {
	totalChars := 0
	for _, text := range texts {
		totalChars += len(text)
	}
	// Rough estimation: 4 characters per token
	return totalChars / 4
}

// estimateCost estimates the cost for processing texts
func (es *EmbeddingService) estimateCost(texts []string) float64 {
	if len(es.providers) == 0 {
		return 0.0
	}

	// Use the default or first available provider for estimation
	var provider BasicEmbeddingProvider
	for _, p := range es.providers {
		if p.GetConfig().Enabled {
			provider = p
			break
		}
	}

	if provider == nil {
		return 0.0
	}

	estimatedTokens := es.estimateTokenCount(texts)
	return provider.GetCostEstimate(estimatedTokens)
}

// Legacy method for backward compatibility
func (es *EmbeddingService) callEmbeddingAPI(ctx context.Context, texts []string) ([][]float32, TokenUsage, error) {
	embeddings, usage, _, err := es.generateEmbeddingsWithProvider(ctx, texts)
	return embeddings, usage, err
}

// callOpenAIAPI calls the OpenAI embeddings API
func (es *EmbeddingService) callOpenAIAPI(ctx context.Context, texts []string) ([][]float32, TokenUsage, error) {
	requestBody := OpenAIEmbeddingRequest{
		Input: texts,
		Model: es.config.ModelName,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", es.config.APIEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+es.config.APIKey)
	req.Header.Set("User-Agent", "Nephoran-Intent-Operator/1.0")

	// Execute request with retries
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= es.config.RetryAttempts; attempt++ {
		resp, lastErr = es.httpClient.Do(req)
		if lastErr == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt < es.config.RetryAttempts {
			es.logger.Warn("API request failed, retrying",
				"attempt", attempt+1,
				"error", lastErr,
				"status", func() int {
					if resp != nil {
						return resp.StatusCode
					}
					return 0
				}(),
			)

			// Wait before retrying
			select {
			case <-ctx.Done():
				return nil, TokenUsage{}, ctx.Err()
			case <-time.After(es.config.RetryDelay * time.Duration(attempt+1)):
			}
		}
	}

	if lastErr != nil {
		return nil, TokenUsage{}, fmt.Errorf("API request failed after %d attempts: %w", es.config.RetryAttempts+1, lastErr)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, TokenUsage{}, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	var apiResponse OpenAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract embeddings
	embeddings := make([][]float32, len(apiResponse.Data))
	for i, data := range apiResponse.Data {
		embeddings[i] = data.Embedding
	}

	// Calculate estimated cost (approximate based on OpenAI pricing)
	costPerToken := 0.00013 / 1000 // Approximate cost for text-embedding-3-large
	estimatedCost := float64(apiResponse.Usage.TotalTokens) * costPerToken

	usage := TokenUsage{
		PromptTokens:  apiResponse.Usage.PromptTokens,
		TotalTokens:   apiResponse.Usage.TotalTokens,
		EstimatedCost: estimatedCost,
	}

	return embeddings, usage, nil
}

// preprocessTexts applies preprocessing to input texts
func (es *EmbeddingService) preprocessTexts(texts []string) []string {
	processed := make([]string, len(texts))

	for i, text := range texts {
		processed[i] = es.preprocessText(text)
	}

	return processed
}

// preprocessText applies preprocessing to a single text
func (es *EmbeddingService) preprocessText(text string) string {
	// Basic normalization
	if es.config.NormalizeText {
		text = strings.TrimSpace(text)
		// Normalize whitespace
		text = strings.Join(strings.Fields(text), " ")
	}

	// Length validation and truncation
	if len(text) < es.config.MinTextLength {
		return text // Return as-is if too short
	}

	if len(text) > es.config.MaxTextLength {
		text = text[:es.config.MaxTextLength]
		es.logger.Debug("Text truncated to max length", "original_length", len(text), "max_length", es.config.MaxTextLength)
	}

	// Telecom-specific preprocessing
	if es.config.TelecomPreprocessing {
		text = es.applyTelecomPreprocessing(text)
	}

	return text
}

// applyTelecomPreprocessing applies telecom-specific text preprocessing
func (es *EmbeddingService) applyTelecomPreprocessing(text string) string {
	// Preserve technical terms by adding emphasis (this is a simple approach)
	if es.config.PreserveTechnicalTerms {
		// Define telecom technical term patterns
		technicalPatterns := []string{
			`\b[A-Z]{2,}(?:-[A-Z]{2,})*\b`, // Acronyms like RAN, AMF, SMF
			`\b\d+G\b`,                     // Technology generations
			`\b(?:Rel|Release)[-\s]*\d+\b`, // Release versions
			`\b[vV]\d+\.\d+(?:\.\d+)?\b`,   // Version numbers
			`\b\d+\.\d+\.\d+\b`,            // Specification numbers
			`\b[A-Z]+\d+\b`,                // Standards like TS38.300
		}

		// Apply weighting by duplicating technical terms (simple approach)
		if es.config.TechnicalTermWeighting > 1.0 {
			for _, pattern := range technicalPatterns {
				// This is a simplified implementation
				// In production, you might use more sophisticated methods
				text = strings.ReplaceAll(text, pattern, pattern+" "+pattern)
			}
		}
	}

	return text
}

// generateCacheKey generates a cache key for the given text using fast FNV hash
func (es *EmbeddingService) generateCacheKey(text string) string {
	// Create a fast hash of the text and configuration that affects embeddings
	h := fnv.New64a()
	h.Write([]byte(text))
	h.Write([]byte(es.config.ModelName))
	h.Write([]byte(fmt.Sprintf("%d", es.config.Dimensions)))
	return fmt.Sprintf("%016x", h.Sum64())
}

// GenerateEmbeddingsForChunks generates embeddings for document chunks
func (es *EmbeddingService) GenerateEmbeddingsForChunks(ctx context.Context, chunks []*DocumentChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	es.logger.Info("Generating embeddings for chunks", "chunk_count", len(chunks))

	// Extract texts from chunks
	texts := make([]string, len(chunks))
	chunkIDs := make([]string, len(chunks))

	for i, chunk := range chunks {
		// Use clean content for embedding
		content := chunk.CleanContent

		// Add section context if available
		if chunk.SectionTitle != "" && es.config.TelecomPreprocessing {
			content = fmt.Sprintf("Section: %s\n\n%s", chunk.SectionTitle, content)
		}

		// Add parent context if available
		if chunk.ParentContext != "" {
			content = fmt.Sprintf("Context: %s\n\n%s", chunk.ParentContext, content)
		}

		texts[i] = content
		chunkIDs[i] = chunk.ID
	}

	// Create embedding request
	request := &EmbeddingRequest{
		Texts:     texts,
		ChunkIDs:  chunkIDs,
		UseCache:  true,
		RequestID: fmt.Sprintf("chunks_%d_%d", time.Now().Unix(), len(chunks)),
		Metadata: map[string]interface{}{
			"chunk_count": len(chunks),
			"source":      "chunk_processing",
		},
	}

	// Generate embeddings
	response, err := es.GenerateEmbeddings(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings for chunks: %w", err)
	}

	// Store embeddings in chunks (if we extended DocumentChunk to include embeddings)
	// For now, we just log the successful generation
	es.logger.Info("Embeddings generated for chunks",
		"chunk_count", len(chunks),
		"processing_time", response.ProcessingTime,
		"cache_hits", response.CacheHits,
	)

	return nil
}

// updateMetrics safely updates the embedding metrics
func (es *EmbeddingService) updateMetrics(updater func(*EmbeddingMetrics)) {
	es.metrics.mutex.Lock()
	defer es.metrics.mutex.Unlock()
	updater(es.metrics)
}

// GetMetrics returns the current embedding metrics
func (es *EmbeddingService) GetMetrics() *EmbeddingMetrics {
	es.metrics.mutex.RLock()
	defer es.metrics.mutex.RUnlock()

	// Return a copy
	metrics := *es.metrics
	return &metrics
}

// startMetricsCollection starts background metrics collection
func (es *EmbeddingService) startMetricsCollection() {
	ticker := time.NewTicker(es.config.MetricsInterval)
	defer ticker.Stop()

	for range ticker.C {
		es.collectMetrics()
	}
}

// collectMetrics collects and logs periodic metrics
func (es *EmbeddingService) collectMetrics() {
	metrics := es.GetMetrics()
	cacheStats := es.cache.Stats()

	es.logger.Info("Embedding service metrics",
		"total_requests", metrics.TotalRequests,
		"success_rate", float64(metrics.SuccessfulRequests)/float64(metrics.TotalRequests),
		"average_latency", metrics.AverageLatency,
		"total_cost", metrics.TotalCost,
		"cache_hit_rate", cacheStats.HitRate,
	)
}

// Rate limiter implementation

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute, tokensPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		requestTokens: make(chan struct{}, requestsPerMinute),
		tokenBucket:   make(chan int, tokensPerMinute),
		lastRefill:    time.Now(),
	}

	// Initialize tokens
	for i := 0; i < requestsPerMinute; i++ {
		rl.requestTokens <- struct{}{}
	}
	for i := 0; i < tokensPerMinute; i++ {
		rl.tokenBucket <- 1
	}

	// Start refill goroutine
	go rl.refillTokens(requestsPerMinute, tokensPerMinute)

	return rl
}

// Wait waits for rate limiting tokens
func (rl *RateLimiter) Wait(ctx context.Context, estimatedTokens int) error {
	// Wait for request token
	select {
	case <-rl.requestTokens:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Wait for enough token bucket capacity
	tokensNeeded := estimatedTokens
	for tokensNeeded > 0 {
		select {
		case <-rl.tokenBucket:
			tokensNeeded--
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// refillTokens periodically refills the rate limiting tokens
func (rl *RateLimiter) refillTokens(requestsPerMinute, tokensPerMinute int) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()

		// Refill request tokens
		for len(rl.requestTokens) < requestsPerMinute {
			select {
			case rl.requestTokens <- struct{}{}:
			default:
				goto refillTokenBucket
			}
		}

	refillTokenBucket:
		// Refill token bucket
		for len(rl.tokenBucket) < tokensPerMinute {
			select {
			case rl.tokenBucket <- 1:
			default:
				break
			}
		}

		rl.lastRefill = time.Now()
		rl.mutex.Unlock()
	}
}

// Cache implementations

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache(maxSize int64) *InMemoryCache {
	return &InMemoryCache{
		data:    make(map[string]cacheEntry),
		maxSize: maxSize,
	}
}

// Get retrieves an embedding from the cache
func (c *InMemoryCache) Get(key string) ([]float32, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists || time.Now().After(entry.expiresAt) {
		c.stats.Misses++
		return nil, false
	}

	c.stats.Hits++
	return entry.embedding, true
}

// Set stores an embedding in the cache
func (c *InMemoryCache) Set(key string, embedding []float32, ttl time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if we need to evict entries
	if int64(len(c.data)) >= c.maxSize {
		c.evictOldest()
	}

	c.data[key] = cacheEntry{
		embedding: embedding,
		expiresAt: time.Now().Add(ttl),
	}

	c.stats.Size = int64(len(c.data))
	return nil
}

// Delete removes an embedding from the cache
func (c *InMemoryCache) Delete(key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)
	c.stats.Size = int64(len(c.data))
	return nil
}

// Clear removes all embeddings from the cache
func (c *InMemoryCache) Clear() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]cacheEntry)
	c.stats.Size = 0
	return nil
}

// Stats returns cache statistics
func (c *InMemoryCache) Stats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	stats := c.stats
	if stats.Hits+stats.Misses > 0 {
		stats.HitRate = float64(stats.Hits) / float64(stats.Hits+stats.Misses)
	}
	return stats
}

// evictOldest evicts the oldest cache entry
func (c *InMemoryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.data {
		if oldestKey == "" || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
		}
	}

	if oldestKey != "" {
		delete(c.data, oldestKey)
	}
}

// NoOpCache is a cache implementation that doesn't cache anything
type NoOpCache struct{}

func (c *NoOpCache) Get(key string) ([]float32, bool)                             { return nil, false }
func (c *NoOpCache) Set(key string, embedding []float32, ttl time.Duration) error { return nil }
func (c *NoOpCache) Delete(key string) error                                      { return nil }
func (c *NoOpCache) Clear() error                                                 { return nil }
func (c *NoOpCache) Stats() CacheStats                                            { return CacheStats{} }

// createProvider creates a provider instance based on configuration
func createProvider(config ProviderConfig, httpClient *http.Client) BasicEmbeddingProvider {
	switch config.Name {
	case "openai":
		return NewBasicOpenAIProvider(config, httpClient)
	case "azure":
		return NewBasicAzureOpenAIProvider(config, httpClient)
	case "huggingface":
		return NewBasicHuggingFaceProvider(config, httpClient)
	case "cohere":
		return NewCohereProvider(config, httpClient)
	case "local":
		return NewLocalProvider(config)
	default:
		return NewBasicOpenAIProvider(config, httpClient)
	}
}

// Cost tracking implementations

// NewCostTracker creates a new cost tracker
func NewCostTracker(limits CostLimits) *CostTracker {
	return &CostTracker{
		dailyCosts:   make(map[string]float64),
		monthlyCosts: make(map[string]float64),
		limits:       limits,
		alerts:       []CostAlert{},
	}
}

// CanAfford checks if the estimated cost can be afforded within limits
func (ct *CostTracker) CanAfford(estimatedCost float64) bool {
	ct.mutex.RLock()
	defer ct.mutex.RUnlock()

	today := time.Now().Format("2006-01-02")
	thisMonth := time.Now().Format("2006-01")

	dailyCost := ct.dailyCosts[today]
	monthlyCost := ct.monthlyCosts[thisMonth]

	return (dailyCost+estimatedCost <= ct.limits.DailyLimit) &&
		(monthlyCost+estimatedCost <= ct.limits.MonthlyLimit)
}

// RecordCost records the actual cost incurred
func (ct *CostTracker) RecordCost(cost float64, provider string) {
	ct.mutex.Lock()
	defer ct.mutex.Unlock()

	today := time.Now().Format("2006-01-02")
	thisMonth := time.Now().Format("2006-01")

	ct.dailyCosts[today] += cost
	ct.monthlyCosts[thisMonth] += cost

	// Check for alerts
	ct.checkAlerts(today, thisMonth)
}

// checkAlerts checks if cost thresholds have been exceeded
func (ct *CostTracker) checkAlerts(today, thisMonth string) {
	dailyCost := ct.dailyCosts[today]
	monthlyCost := ct.monthlyCosts[thisMonth]

	// Daily threshold alert
	if dailyCost >= ct.limits.DailyLimit*ct.limits.AlertThreshold {
		alert := CostAlert{
			Timestamp: time.Now(),
			Type:      "daily",
			Amount:    dailyCost,
			Limit:     ct.limits.DailyLimit,
			Message:   fmt.Sprintf("Daily cost alert: $%.2f (%.1f%% of limit)", dailyCost, dailyCost/ct.limits.DailyLimit*100),
		}
		ct.alerts = append(ct.alerts, alert)
	}

	// Monthly threshold alert
	if monthlyCost >= ct.limits.MonthlyLimit*ct.limits.AlertThreshold {
		alert := CostAlert{
			Timestamp: time.Now(),
			Type:      "monthly",
			Amount:    monthlyCost,
			Limit:     ct.limits.MonthlyLimit,
			Message:   fmt.Sprintf("Monthly cost alert: $%.2f (%.1f%% of limit)", monthlyCost, monthlyCost/ct.limits.MonthlyLimit*100),
		}
		ct.alerts = append(ct.alerts, alert)
	}
}

// GetAlerts returns recent cost alerts
func (ct *CostTracker) GetAlerts() []CostAlert {
	ct.mutex.RLock()
	defer ct.mutex.RUnlock()
	return append([]CostAlert{}, ct.alerts...)
}

// Quality assessment implementations

// NewQualityAssessment creates a new quality assessment service
func NewQualityAssessment(config QualityConfig) *QualityAssessment {
	return &QualityAssessment{
		config:        config,
		metrics:       QualityMetrics{},
		referenceEmbs: make(map[string][]float32),
	}
}

// AssessQuality assesses the quality of generated embeddings
func (qa *QualityAssessment) AssessQuality(embeddings [][]float32, texts []string) float64 {
	if !qa.config.Enabled || len(embeddings) == 0 {
		return 1.0 // Default high quality if assessment is disabled
	}

	qa.mutex.Lock()
	defer qa.mutex.Unlock()

	var totalScore float64
	validAssessments := 0

	for i, embedding := range embeddings {
		if i >= len(texts) {
			break
		}

		// Check embedding dimensions
		if len(embedding) == 0 {
			continue
		}

		// Calculate quality score based on various factors
		score := qa.calculateEmbeddingScore(embedding, texts[i])
		totalScore += score
		validAssessments++
	}

	if validAssessments == 0 {
		return 0.0
	}

	averageScore := totalScore / float64(validAssessments)

	// Update metrics
	qa.metrics.TotalChecks++
	if averageScore < qa.config.MinScore {
		qa.metrics.FailedChecks++
	}
	qa.metrics.AverageScore = (qa.metrics.AverageScore*float64(qa.metrics.TotalChecks-1) + averageScore) / float64(qa.metrics.TotalChecks)
	qa.metrics.LastCheck = time.Now()

	if qa.metrics.MinScore == 0 || averageScore < qa.metrics.MinScore {
		qa.metrics.MinScore = averageScore
	}
	if averageScore > qa.metrics.MaxScore {
		qa.metrics.MaxScore = averageScore
	}

	return averageScore
}

// calculateEmbeddingScore calculates a quality score for a single embedding
func (qa *QualityAssessment) calculateEmbeddingScore(embedding []float32, text string) float64 {
	score := 1.0

	// Check for zero embeddings
	zeroCount := 0
	for _, val := range embedding {
		if val == 0.0 {
			zeroCount++
		}
	}
	zeroRatio := float64(zeroCount) / float64(len(embedding))
	if zeroRatio > 0.5 { // More than 50% zeros is suspicious
		score *= 0.5
	}

	// Check magnitude (embeddings should have reasonable magnitude)
	var magnitude float64
	for _, val := range embedding {
		magnitude += float64(val * val)
	}
	magnitude = math.Sqrt(magnitude)

	if magnitude < 0.1 || magnitude > 100.0 { // Suspicious magnitude
		score *= 0.7
	}

	// Check text length correlation (longer texts should have more informative embeddings)
	if len(text) > 100 && magnitude < 1.0 {
		score *= 0.8
	}

	return score
}

// Background services

// startHealthChecks starts periodic health checks for providers
func (es *EmbeddingService) startHealthChecks() {
	ticker := time.NewTicker(es.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		es.performHealthChecks()
	}
}

// performHealthChecks checks the health of all providers
func (es *EmbeddingService) performHealthChecks() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for name, provider := range es.providers {
		go func(providerName string, p BasicEmbeddingProvider) {
			if err := p.HealthCheck(ctx); err != nil {
				es.logger.Warn("Provider health check failed", "provider", providerName, "error", err)
			} else {
				es.logger.Debug("Provider health check passed", "provider", providerName)
			}
		}(name, provider)
	}
}

// startCostMonitoring starts cost monitoring and alerting
func (es *EmbeddingService) startCostMonitoring() {
	ticker := time.NewTicker(1 * time.Hour) // Check costs hourly
	defer ticker.Stop()

	for range ticker.C {
		alerts := es.costTracker.GetAlerts()
		for _, alert := range alerts {
			es.logger.Warn("Cost alert", "type", alert.Type, "message", alert.Message)
		}
	}
}

// CheckStatus checks the health status of the embedding service
func (es *EmbeddingService) CheckStatus(ctx context.Context) (*ComponentStatus, error) {
	if es == nil {
		return &ComponentStatus{
			Status:    "unhealthy",
			Message:   "Embedding service not initialized",
			LastCheck: time.Now(),
		}, nil
	}

	// Check if we have any providers
	if len(es.providers) == 0 {
		return &ComponentStatus{
			Status:    "unhealthy",
			Message:   "No embedding providers configured",
			LastCheck: time.Now(),
		}, nil
	}

	// Check each provider's health
	healthyProviders := 0
	totalProviders := len(es.providers)
	details := make(map[string]interface{})

	for name, provider := range es.providers {
		providerHealthy := true
		var providerError error

		// Perform a quick test embedding to verify provider functionality
		if provider != nil {
			testCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			_, _, err := provider.GenerateEmbeddings(testCtx, []string{"test"})
			cancel()

			if err != nil {
				providerHealthy = false
				providerError = err
			}
		} else {
			providerHealthy = false
			providerError = fmt.Errorf("provider is nil")
		}

		if providerHealthy {
			healthyProviders++
		}

		details[name] = map[string]interface{}{
			"healthy": providerHealthy,
			"error":   providerError,
		}
	}

	// Determine overall status
	var status, message string
	if healthyProviders == 0 {
		status = "unhealthy"
		message = "All embedding providers are unavailable"
	} else if healthyProviders < totalProviders {
		status = "degraded"
		message = fmt.Sprintf("%d of %d embedding providers are healthy", healthyProviders, totalProviders)
	} else {
		status = "healthy"
		message = "All embedding providers are operational"
	}

	return &ComponentStatus{
		Status:    status,
		Message:   message,
		LastCheck: time.Now(),
		Details:   details,
	}, nil
}
