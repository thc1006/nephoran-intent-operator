//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// MultiProviderEmbeddingService provides intelligent embedding generation with multiple providers.

type MultiProviderEmbeddingService struct {
	config *EmbeddingConfig

	logger *slog.Logger

	providers map[string]EnhancedEmbeddingProvider

	loadBalancer *EnhancedLoadBalancer

	costManager *EnhancedCostManager

	qualityManager *QualityManager

	cacheManager *EmbeddingCacheManager

	healthMonitor *ProviderHealthMonitor

	metrics *EmbeddingMetrics

	mutex sync.RWMutex
}

// EnhancedEmbeddingProvider interface for different embedding providers.

type EnhancedEmbeddingProvider interface {
	GenerateEmbeddings(ctx context.Context, texts []string) (*EmbeddingResponse, error)

	GetModelInfo() ModelInfo

	EstimateCost(texts []string) float64

	IsHealthy() bool

	GetLatency() time.Duration

	GetName() string
}

// ModelInfo contains information about the embedding model.

type ModelInfo struct {
	Name string `json:"name"`

	Dimensions int `json:"dimensions"`

	MaxTokens int `json:"max_tokens"`

	CostPerToken float64 `json:"cost_per_token"`

	Provider string `json:"provider"`

	ApiVersion string `json:"api_version"`

	Capabilities []string `json:"capabilities"`
}

// EnhancedProviderConfig holds configuration for a specific provider.

type EnhancedProviderConfig struct {
	Name string `json:"name"`

	Type string `json:"type"` // "openai", "azure", "local", "huggingface"

	APIKey string `json:"api_key"`

	APIEndpoint string `json:"api_endpoint"`

	ModelName string `json:"model_name"`

	Priority int `json:"priority"` // Higher = preferred

	MaxConcurrency int `json:"max_concurrency"`

	RateLimit int `json:"rate_limit"` // Requests per minute

	CostPerToken float64 `json:"cost_per_token"`

	Timeout time.Duration `json:"timeout"`

	Headers map[string]string `json:"headers"`

	Enabled bool `json:"enabled"`

	// Provider-specific settings.

	AzureDeployment string `json:"azure_deployment,omitempty"`

	HuggingfaceModel string `json:"huggingface_model,omitempty"`

	LocalModelPath string `json:"local_model_path,omitempty"`
}

// LoadBalancer manages provider selection and load distribution.

type EnhancedLoadBalancer struct {
	strategy string // "round_robin", "least_cost", "fastest", "quality"

	providers []string

	currentIdx int

	weights map[string]float64

	mutex sync.RWMutex
}

// CostManager tracks and manages embedding costs.

type EnhancedCostManager struct {
	dailySpend map[string]float64 // date -> amount

	monthlySpend map[string]float64 // month -> amount

	limits EnhancedCostLimits

	alerts []EnhancedCostAlert

	mutex sync.RWMutex
}

// CostLimits defines spending limits.

type EnhancedCostLimits struct {
	DailyLimit float64 `json:"daily_limit"`

	MonthlyLimit float64 `json:"monthly_limit"`

	AlertThreshold float64 `json:"alert_threshold"` // Percentage of limit

}

// CostAlert represents a cost alert.

type EnhancedCostAlert struct {
	Timestamp time.Time `json:"timestamp"`

	Level string `json:"level"` // "warning", "critical"

	Message string `json:"message"`

	Amount float64 `json:"amount"`

	Limit float64 `json:"limit"`
}

// All type definitions moved to embedding_support.go to avoid duplicates.

// NewMultiProviderEmbeddingService creates a new multi-provider embedding service.

func NewMultiProviderEmbeddingService(config *EmbeddingConfig) (*MultiProviderEmbeddingService, error) {

	if config == nil {

		config = getDefaultEmbeddingConfig()

	}

	service := &MultiProviderEmbeddingService{

		config: config,

		logger: slog.Default().With("component", "multi-provider-embedding"),

		providers: make(map[string]EnhancedEmbeddingProvider),

		metrics: &EmbeddingMetrics{ModelStats: make(map[string]ModelUsageStats)},
	}

	// Initialize providers.

	if err := service.initializeProviders(); err != nil {

		return nil, fmt.Errorf("failed to initialize providers: %w", err)

	}

	// Initialize load balancer.

	service.loadBalancer = NewEnhancedLoadBalancer(config.LoadBalancing, service.getProviderNames())

	// Initialize cost manager.

	service.costManager = NewEnhancedCostManager(EnhancedCostLimits{

		DailyLimit: config.DailyCostLimit,

		MonthlyLimit: config.MonthlyCostLimit,

		AlertThreshold: config.CostAlertThreshold,
	})

	// Initialize quality manager.

	service.qualityManager = NewQualityManager(config.MinQualityScore, config.QualityCheckSample)

	// Initialize cache manager.

	var err error

	service.cacheManager, err = NewEmbeddingCacheManager(config)

	if err != nil {

		service.logger.Warn("Failed to initialize cache manager", "error", err)

	}

	// Initialize health monitor.

	service.healthMonitor = NewProviderHealthMonitor(config.HealthCheckInterval)

	// Start health monitoring with adapted providers.

	adaptedProviders := make(map[string]EmbeddingProvider)

	for name, provider := range service.providers {

		adaptedProviders[name] = &ProviderHealthAdapter{provider: provider}

	}

	go service.healthMonitor.StartMonitoring(adaptedProviders)

	service.logger.Info("Multi-provider embedding service initialized",

		"providers", len(service.providers),

		"load_balancing", config.LoadBalancing,

		"cost_tracking", config.EnableCostTracking,
	)

	return service, nil

}

// GenerateEmbeddings generates embeddings using the best available provider.

func (mps *MultiProviderEmbeddingService) GenerateEmbeddings(ctx context.Context, request *EmbeddingRequest) (*EmbeddingResponse, error) {

	startTime := time.Now()

	mps.logger.Debug("Generating embeddings",

		"texts", len(request.Texts),

		"use_cache", request.UseCache,

		"priority", request.Priority,
	)

	// Validate request.

	if err := mps.validateRequest(request); err != nil {

		return nil, fmt.Errorf("invalid request: %w", err)

	}

	// Check cache first if enabled.

	if request.UseCache && mps.cacheManager != nil {

		if cachedResponse := mps.checkCache(request); cachedResponse != nil {

			cachedResponse.ProcessingTime = time.Since(startTime)

			mps.updateMetrics(cachedResponse, true)

			return cachedResponse, nil

		}

	}

	// Check cost limits.

	if mps.config.EnableCostTracking {

		estimatedCost := mps.estimateTotalCost(request.Texts)

		if !mps.costManager.CanAfford(estimatedCost) {

			return nil, fmt.Errorf("request would exceed cost limits: estimated cost $%.4f", estimatedCost)

		}

	}

	// Select best provider.

	provider, err := mps.selectProvider(request)

	if err != nil {

		return nil, fmt.Errorf("failed to select provider: %w", err)

	}

	// Preprocess and deduplicate texts.

	deduplicatedRequest, indexMapping := mps.preprocessAndDeduplicate(request)

	// Generate embeddings for deduplicated texts.

	response, err := mps.generateWithProvider(ctx, provider, deduplicatedRequest)

	if err != nil {

		// Try fallback providers if enabled.

		if mps.config.FallbackEnabled {

			response, err = mps.generateWithFallback(ctx, provider, deduplicatedRequest)

		}

		if err != nil {

			mps.updateMetrics(nil, false)

			return nil, fmt.Errorf("embedding generation failed: %w", err)

		}

	}

	// Expand response back to original text count using index mapping.

	response = mps.expandResponse(response, indexMapping, len(request.Texts))

	// Quality check.

	if mps.config.EnableQualityCheck && len(response.Embeddings) > 0 {

		qualityScore := mps.qualityManager.AssessQuality(response.Embeddings[0], provider.GetName(), provider.GetModelInfo().Name)

		if qualityScore < mps.config.MinQualityScore {

			mps.logger.Warn("Low quality embeddings detected",

				"provider", provider.GetName(),

				"quality_score", qualityScore,

				"threshold", mps.config.MinQualityScore,
			)

		}

	}

	// Cache the response.

	if request.UseCache && mps.cacheManager != nil {

		mps.cacheResponse(request, response)

	}

	// Update cost tracking.

	if mps.config.EnableCostTracking {

		mps.costManager.RecordSpending(response.TokenUsage.EstimatedCost)

	}

	// Update metrics.

	response.ProcessingTime = time.Since(startTime)

	mps.updateMetrics(response, false)

	mps.logger.Debug("Embeddings generated successfully",

		"provider", provider.GetName(),

		"embeddings", len(response.Embeddings),

		"processing_time", response.ProcessingTime,

		"cost", response.TokenUsage.EstimatedCost,
	)

	return response, nil

}

// initializeProviders initializes all configured providers.

func (mps *MultiProviderEmbeddingService) initializeProviders() error {

	for _, providerConfig := range mps.config.Providers {

		if !providerConfig.Enabled {

			continue

		}

		// Convert ProviderConfig to EnhancedProviderConfig.

		enhancedConfig := EnhancedProviderConfig{

			Name: providerConfig.Name,

			Type: providerConfig.Name, // Use Name as Type for backward compatibility

			APIKey: providerConfig.APIKey,

			APIEndpoint: providerConfig.APIEndpoint,

			ModelName: providerConfig.ModelName,

			Priority: providerConfig.Priority,

			MaxConcurrency: 10, // Default value

			RateLimit: providerConfig.RateLimit,

			CostPerToken: providerConfig.CostPerToken,

			Timeout: 30 * time.Second, // Default timeout

			Enabled: providerConfig.Enabled,
		}

		provider, err := mps.createProvider(enhancedConfig)

		if err != nil {

			mps.logger.Error("Failed to create provider", "name", providerConfig.Name, "error", err)

			continue

		}

		mps.providers[providerConfig.Name] = provider

		mps.logger.Info("Provider initialized", "name", providerConfig.Name, "type", enhancedConfig.Type)

	}

	if len(mps.providers) == 0 {

		return fmt.Errorf("no providers successfully initialized")

	}

	return nil

}

// createProvider creates a specific provider instance.

func (mps *MultiProviderEmbeddingService) createProvider(config EnhancedProviderConfig) (EnhancedEmbeddingProvider, error) {

	switch config.Type {

	case "openai":

		return NewOpenAIProvider(config)

	case "azure":

		return NewAzureOpenAIProvider(config)

	case "huggingface":

		return NewHuggingFaceProvider(config)

	case "local":

		return NewLocalEnhancedEmbeddingProvider(config)

	default:

		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)

	}

}

// selectProvider selects the best provider for the request.

func (mps *MultiProviderEmbeddingService) selectProvider(request *EmbeddingRequest) (EnhancedEmbeddingProvider, error) {

	availableProviders := mps.getHealthyProviders()

	if len(availableProviders) == 0 {

		return nil, fmt.Errorf("no healthy providers available")

	}

	// Use load balancer to select provider.

	providerName := mps.loadBalancer.SelectProvider(availableProviders, request)

	provider, exists := mps.providers[providerName]

	if !exists {

		return nil, fmt.Errorf("selected provider %s not found", providerName)

	}

	return provider, nil

}

// generateWithProvider generates embeddings using a specific provider.

func (mps *MultiProviderEmbeddingService) generateWithProvider(ctx context.Context, provider EnhancedEmbeddingProvider, request *EmbeddingRequest) (*EmbeddingResponse, error) {

	// Preprocess texts if telecom preprocessing is enabled.

	processedTexts := request.Texts

	if mps.config.TelecomPreprocessing {

		processedTexts = mps.preprocessTelecomTexts(request.Texts)

	}

	// Create new request with processed texts.

	processedRequest := &EmbeddingRequest{

		Texts: processedTexts,

		Metadata: request.Metadata,

		RequestID: request.RequestID,

		Priority: request.Priority,

		ChunkIDs: request.ChunkIDs,

		UseCache: request.UseCache,
	}

	// Generate embeddings.

	response, err := provider.GenerateEmbeddings(ctx, processedRequest.Texts)

	if err != nil {

		return nil, fmt.Errorf("provider %s failed: %w", provider.GetName(), err)

	}

	// Enhance response metadata.

	response.RequestID = request.RequestID

	response.ModelUsed = provider.GetModelInfo().Name

	return response, nil

}

// generateWithFallback tries fallback providers if the primary fails.

func (mps *MultiProviderEmbeddingService) generateWithFallback(ctx context.Context, failedProvider EnhancedEmbeddingProvider, request *EmbeddingRequest) (*EmbeddingResponse, error) {

	fallbackOrder := mps.config.FallbackOrder

	if len(fallbackOrder) == 0 {

		// Use all healthy providers except the failed one.

		fallbackOrder = mps.getHealthyProviders()

	}

	for _, providerName := range fallbackOrder {

		if providerName == failedProvider.GetName() {

			continue // Skip the failed provider

		}

		provider, exists := mps.providers[providerName]

		if !exists || !provider.IsHealthy() {

			continue

		}

		mps.logger.Info("Trying fallback provider", "provider", providerName)

		response, err := mps.generateWithProvider(ctx, provider, request)

		if err == nil {

			return response, nil

		}

		mps.logger.Warn("Fallback provider failed", "provider", providerName, "error", err)

	}

	return nil, fmt.Errorf("all fallback providers failed")

}

// preprocessTelecomTexts applies telecom-specific preprocessing.

func (mps *MultiProviderEmbeddingService) preprocessTelecomTexts(texts []string) []string {

	var processedTexts []string

	for _, text := range texts {

		processed := mps.preprocessSingleText(text)

		processedTexts = append(processedTexts, processed)

	}

	return processedTexts

}

// preprocessSingleText preprocesses a single text for telecom content.

func (mps *MultiProviderEmbeddingService) preprocessSingleText(text string) string {

	// Normalize telecom acronyms.

	text = mps.expandTelecomAcronyms(text)

	// Preserve technical terms.

	if mps.config.PreserveTechnicalTerms {

		text = mps.preserveTechnicalTerms(text)

	}

	// Apply technical term weighting.

	if mps.config.TechnicalTermWeighting > 1.0 {

		text = mps.applyTechnicalTermWeighting(text)

	}

	return text

}

// expandTelecomAcronyms expands common telecom acronyms.

func (mps *MultiProviderEmbeddingService) expandTelecomAcronyms(text string) string {

	acronymMap := map[string]string{

		"5G": "Fifth Generation 5G",

		"4G": "Fourth Generation 4G",

		"gNB": "Next Generation NodeB gNB",

		"eNB": "Evolved NodeB eNB",

		"AMF": "Access and Mobility Management Function AMF",

		"SMF": "Session Management Function SMF",

		"UPF": "User Plane Function UPF",

		"O-RAN": "Open Radio Access Network O-RAN",

		"RAN": "Radio Access Network RAN",

		"QoS": "Quality of Service QoS",

		"URLLC": "Ultra-Reliable Low Latency Communication URLLC",

		"eMBB": "Enhanced Mobile Broadband eMBB",

		"mMTC": "Massive Machine Type Communication mMTC",
	}

	for acronym, expansion := range acronymMap {

		text = strings.ReplaceAll(text, acronym, expansion)

	}

	return text

}

// preserveTechnicalTerms ensures technical terms are properly formatted.

func (mps *MultiProviderEmbeddingService) preserveTechnicalTerms(text string) string {

	// Add special markers around technical terms to preserve them.

	technicalTerms := []string{

		"MHz", "GHz", "dBm", "EIRP", "SINR", "RSRP", "RSRQ",

		"MAC", "RLC", "PDCP", "RRC", "PHY",

		"CN", "SA", "NSA", "EN-DC",

		"PLMN", "IMSI", "IMEI", "SUPI", "SUCI",
	}

	for _, term := range technicalTerms {

		text = strings.ReplaceAll(text, term, fmt.Sprintf("[TECH]%s[/TECH]", term))

	}

	return text

}

// applyTechnicalTermWeighting boosts technical terms for better embedding.

func (mps *MultiProviderEmbeddingService) applyTechnicalTermWeighting(text string) string {

	// This would typically involve duplicating technical terms based on weighting.

	weight := int(mps.config.TechnicalTermWeighting)

	technicalTerms := []string{

		"specification", "protocol", "interface", "procedure",

		"frequency", "bandwidth", "power", "antenna",

		"handover", "measurement", "configuration",
	}

	for _, term := range technicalTerms {

		if strings.Contains(strings.ToLower(text), term) {

			// Duplicate the term to increase its weight.

			replacement := strings.Repeat(term+" ", weight) + term

			text = strings.ReplaceAll(strings.ToLower(text), term, replacement)

		}

	}

	return text

}

// checkCache checks for cached embeddings.

func (mps *MultiProviderEmbeddingService) checkCache(request *EmbeddingRequest) *EmbeddingResponse {

	if mps.cacheManager == nil {

		return nil

	}

	var embeddings [][]float32

	cacheHits := 0

	cacheMisses := 0

	for _, text := range request.Texts {

		cacheKey := mps.generateCacheKey(text)

		if embedding, found := mps.cacheManager.Get(cacheKey); found {

			embeddings = append(embeddings, embedding)

			cacheHits++

		} else {

			// If any text is not cached, we need to generate all embeddings.
			// Log cache miss (cacheMisses would be used for metrics if not returning early)
			_ = cacheMisses // Acknowledge variable to avoid linter warning

			return nil

		}

	}

	if len(embeddings) == len(request.Texts) {

		return &EmbeddingResponse{

			Embeddings: embeddings,

			CacheHits: cacheHits,

			CacheMisses: cacheMisses,

			RequestID: request.RequestID,
		}

	}

	return nil

}

// cacheResponse caches the embedding response.

func (mps *MultiProviderEmbeddingService) cacheResponse(request *EmbeddingRequest, response *EmbeddingResponse) {

	if mps.cacheManager == nil {

		return

	}

	for i, text := range request.Texts {

		if i < len(response.Embeddings) {

			cacheKey := mps.generateCacheKey(text)

			mps.cacheManager.Set(cacheKey, response.Embeddings[i], mps.config.CacheTTL)

		}

	}

}

// generateCacheKey generates a cache key for text using fast FNV hash.

func (mps *MultiProviderEmbeddingService) generateCacheKey(text string) string {

	h := fnv.New64a()

	h.Write([]byte(text))

	return fmt.Sprintf("emb:%016x", h.Sum64())

}

// preprocessAndDeduplicate preprocesses texts and removes duplicates for efficient API usage.

func (mps *MultiProviderEmbeddingService) preprocessAndDeduplicate(request *EmbeddingRequest) (*EmbeddingRequest, map[int]int) {

	processed := make([]string, 0, len(request.Texts))

	dedupeMap := make(map[string]int)

	indexMapping := make(map[int]int) // maps original index to deduplicated index

	duplicateCount := 0

	for i, text := range request.Texts {

		// Preprocess the text (normalize, clean, etc.).

		processedText := mps.preprocessSingleText(text)

		if existingIdx, exists := dedupeMap[processedText]; exists {

			// Text already exists, map to existing index.

			indexMapping[i] = existingIdx

			duplicateCount++

		} else {

			// New unique text, add to processed list.

			newIdx := len(processed)

			processed = append(processed, processedText)

			dedupeMap[processedText] = newIdx

			indexMapping[i] = newIdx

		}

	}

	// Log deduplication statistics.

	if duplicateCount > 0 {

		deduplicationRatio := float64(duplicateCount) / float64(len(request.Texts)) * 100

		mps.logger.Info("Text deduplication applied",

			"original_count", len(request.Texts),

			"unique_count", len(processed),

			"duplicates_removed", duplicateCount,

			"deduplication_ratio", fmt.Sprintf("%.1f%%", deduplicationRatio),
		)

	}

	// Create new request with deduplicated texts.

	deduplicatedRequest := &EmbeddingRequest{

		Texts: processed,

		UseCache: request.UseCache,

		Priority: request.Priority,

		Metadata: request.Metadata,

		Model: request.Model,

		BatchSize: request.BatchSize,

		Timeout: request.Timeout,
	}

	return deduplicatedRequest, indexMapping

}

// expandResponse expands the deduplicated response back to the original text count.

func (mps *MultiProviderEmbeddingService) expandResponse(response *EmbeddingResponse, indexMapping map[int]int, originalCount int) *EmbeddingResponse {

	if len(indexMapping) == originalCount && len(response.Embeddings) == originalCount {

		// No deduplication was applied.

		return response

	}

	// Create expanded embeddings array.

	expandedEmbeddings := make([][]float32, originalCount)

	for originalIdx := 0; originalIdx < originalCount; originalIdx++ {

		deduplicatedIdx := indexMapping[originalIdx]

		if deduplicatedIdx < len(response.Embeddings) {

			expandedEmbeddings[originalIdx] = response.Embeddings[deduplicatedIdx]

		}

	}

	// Update response with expanded embeddings.

	expandedResponse := &EmbeddingResponse{

		Embeddings: expandedEmbeddings,

		Model: response.Model,

		Provider: response.Provider,

		TokenUsage: response.TokenUsage,

		ProcessingTime: response.ProcessingTime,

		CacheHitRate: response.CacheHitRate,

		QualityScore: response.QualityScore,

		Cost: response.Cost,

		RequestID: response.RequestID,

		Metadata: response.Metadata,
	}

	mps.logger.Debug("Response expanded after deduplication",

		"original_embedding_count", len(response.Embeddings),

		"expanded_embedding_count", len(expandedEmbeddings),
	)

	return expandedResponse

}

// estimateTotalCost estimates the total cost for the request.

func (mps *MultiProviderEmbeddingService) estimateTotalCost(texts []string) float64 {

	if len(mps.providers) == 0 {

		return 0

	}

	// Use the first available provider for cost estimation.

	for _, provider := range mps.providers {

		return provider.EstimateCost(texts)

	}

	return 0

}

// validateRequest validates the embedding request.

func (mps *MultiProviderEmbeddingService) validateRequest(request *EmbeddingRequest) error {

	if len(request.Texts) == 0 {

		return fmt.Errorf("no texts provided")

	}

	for i, text := range request.Texts {

		if len(text) < mps.config.MinTextLength {

			return fmt.Errorf("text %d too short: %d characters (minimum %d)", i, len(text), mps.config.MinTextLength)

		}

		if len(text) > mps.config.MaxTextLength {

			return fmt.Errorf("text %d too long: %d characters (maximum %d)", i, len(text), mps.config.MaxTextLength)

		}

	}

	return nil

}

// getProviderNames returns list of provider names.

func (mps *MultiProviderEmbeddingService) getProviderNames() []string {

	mps.mutex.RLock()

	defer mps.mutex.RUnlock()

	var names []string

	for name := range mps.providers {

		names = append(names, name)

	}

	return names

}

// getHealthyProviders returns list of healthy provider names.

func (mps *MultiProviderEmbeddingService) getHealthyProviders() []string {

	mps.mutex.RLock()

	defer mps.mutex.RUnlock()

	var healthy []string

	for name, provider := range mps.providers {

		if provider.IsHealthy() {

			healthy = append(healthy, name)

		}

	}

	return healthy

}

// updateMetrics updates service metrics.

func (mps *MultiProviderEmbeddingService) updateMetrics(response *EmbeddingResponse, fromCache bool) {

	mps.metrics.mutex.Lock()

	defer mps.metrics.mutex.Unlock()

	mps.metrics.TotalRequests++

	mps.metrics.LastUpdated = time.Now()

	if response != nil {

		mps.metrics.SuccessfulRequests++

		mps.metrics.TotalTexts += int64(len(response.Embeddings))

		mps.metrics.TotalTokens += int64(response.TokenUsage.TotalTokens)

		mps.metrics.TotalCost += response.TokenUsage.EstimatedCost

		// Update average latency.

		if mps.metrics.SuccessfulRequests > 0 {

			mps.metrics.AverageLatency = time.Duration(

				(int64(mps.metrics.AverageLatency)*mps.metrics.SuccessfulRequests + int64(response.ProcessingTime)) /

					(mps.metrics.SuccessfulRequests + 1),
			)

		}

		// Update cache stats.

		if fromCache {

			mps.metrics.CacheStats.CacheHits++

		} else {

			mps.metrics.CacheStats.CacheMisses++

		}

		mps.metrics.CacheStats.TotalLookups++

		if mps.metrics.CacheStats.TotalLookups > 0 {

			mps.metrics.CacheStats.HitRate = float64(mps.metrics.CacheStats.CacheHits) / float64(mps.metrics.CacheStats.TotalLookups)

		}

		// Update model stats.

		if response.ModelUsed != "" {

			stats := mps.metrics.ModelStats[response.ModelUsed]

			stats.RequestCount++

			stats.TokenCount += int64(response.TokenUsage.TotalTokens)

			stats.TotalCost += response.TokenUsage.EstimatedCost

			mps.metrics.ModelStats[response.ModelUsed] = stats

		}

	} else {

		mps.metrics.FailedRequests++

	}

}

// GetMetrics returns current service metrics.

func (mps *MultiProviderEmbeddingService) GetMetrics() *EmbeddingMetrics {

	mps.metrics.mutex.RLock()

	defer mps.metrics.mutex.RUnlock()

	// Return a copy without the mutex.

	metrics := &EmbeddingMetrics{

		TotalRequests: mps.metrics.TotalRequests,

		SuccessfulRequests: mps.metrics.SuccessfulRequests,

		FailedRequests: mps.metrics.FailedRequests,

		TotalTexts: mps.metrics.TotalTexts,

		TotalTokens: mps.metrics.TotalTokens,

		TotalCost: mps.metrics.TotalCost,

		AverageLatency: mps.metrics.AverageLatency,

		AverageTextLength: mps.metrics.AverageTextLength,

		CacheStats: struct {
			TotalLookups int64 `json:"total_lookups"`

			CacheHits int64 `json:"cache_hits"`

			CacheMisses int64 `json:"cache_misses"`

			HitRate float64 `json:"hit_rate"`
		}{

			TotalLookups: mps.metrics.CacheStats.TotalLookups,

			CacheHits: mps.metrics.CacheStats.CacheHits,

			CacheMisses: mps.metrics.CacheStats.CacheMisses,

			HitRate: mps.metrics.CacheStats.HitRate,
		},

		RateLimitingStats: struct {
			RateLimitHits int64 `json:"rate_limit_hits"`

			TotalWaitTime time.Duration `json:"total_wait_time"`

			AverageWaitTime time.Duration `json:"average_wait_time"`
		}{

			RateLimitHits: mps.metrics.RateLimitingStats.RateLimitHits,

			TotalWaitTime: mps.metrics.RateLimitingStats.TotalWaitTime,

			AverageWaitTime: mps.metrics.RateLimitingStats.AverageWaitTime,
		},

		ModelStats: copyModelStats(mps.metrics.ModelStats),

		LastUpdated: mps.metrics.LastUpdated,
	}

	return metrics

}

// GetCostSummary returns cost tracking summary.

func (mps *MultiProviderEmbeddingService) GetCostSummary() *CostSummary {

	if mps.costManager == nil {

		return nil

	}

	enhancedSummary := mps.costManager.GetSummary()

	if enhancedSummary == nil {

		return nil

	}

	// Convert EnhancedCostSummary to CostSummary.

	alerts := make([]CostAlert, len(enhancedSummary.Alerts))

	for i, alert := range enhancedSummary.Alerts {

		alerts[i] = CostAlert{

			Type: alert.Level,

			Message: alert.Message,

			Amount: alert.Amount,

			Limit: alert.Limit,

			Timestamp: alert.Timestamp,
		}

	}

	return &CostSummary{

		DailySpending: enhancedSummary.DailySpending,

		MonthlySpending: enhancedSummary.MonthlySpending,

		DailyLimit: enhancedSummary.DailyLimit,

		MonthlyLimit: enhancedSummary.MonthlyLimit,

		Alerts: alerts,
	}

}

// GetProviderStatus returns status of all providers.

func (mps *MultiProviderEmbeddingService) GetProviderStatus() map[string]*HealthStatus {

	if mps.healthMonitor == nil {

		return nil

	}

	return mps.healthMonitor.GetStatus()

}

// Close shuts down the service.

func (mps *MultiProviderEmbeddingService) Close() error {

	if mps.healthMonitor != nil {

		mps.healthMonitor.Stop()

	}

	if mps.cacheManager != nil {

		return mps.cacheManager.Close()

	}

	return nil

}
