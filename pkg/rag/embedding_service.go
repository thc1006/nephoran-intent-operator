package rag

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
	"bytes"
)

// EmbeddingService provides embedding generation for telecom documents
type EmbeddingService struct {
	config      *EmbeddingConfig
	logger      *slog.Logger
	metrics     *EmbeddingMetrics
	httpClient  *http.Client
	rateLimiter *RateLimiter
	cache       EmbeddingCache
	mutex       sync.RWMutex
}

// EmbeddingConfig holds configuration for the embedding service
type EmbeddingConfig struct {
	// Provider configuration
	Provider     string `json:"provider"`      // "openai", "azure", "local", etc.
	APIKey       string `json:"api_key"`
	APIEndpoint  string `json:"api_endpoint"`
	ModelName    string `json:"model_name"`
	
	// Model-specific settings
	Dimensions      int     `json:"dimensions"`       // Embedding dimensions
	MaxTokens       int     `json:"max_tokens"`       // Max tokens per request
	BatchSize       int     `json:"batch_size"`       // Documents per batch
	MaxConcurrency  int     `json:"max_concurrency"`  // Concurrent requests
	
	// Request configuration
	RequestTimeout  time.Duration `json:"request_timeout"`
	RetryAttempts   int           `json:"retry_attempts"`
	RetryDelay      time.Duration `json:"retry_delay"`
	
	// Rate limiting
	RateLimit       int           `json:"rate_limit"`        // Requests per minute
	TokenRateLimit  int           `json:"token_rate_limit"`  // Tokens per minute
	
	// Quality control
	MinTextLength   int     `json:"min_text_length"`   // Minimum text length for embedding
	MaxTextLength   int     `json:"max_text_length"`   // Maximum text length for embedding
	NormalizeText   bool    `json:"normalize_text"`    // Normalize text before embedding
	RemoveStopWords bool    `json:"remove_stop_words"` // Remove common stop words
	
	// Caching
	EnableCaching   bool          `json:"enable_caching"`
	CacheTTL        time.Duration `json:"cache_ttl"`
	CacheDirectory  string        `json:"cache_directory"`
	
	// Telecom-specific preprocessing
	TelecomPreprocessing bool     `json:"telecom_preprocessing"` // Apply telecom-specific preprocessing
	PreserveTechnicalTerms bool   `json:"preserve_technical_terms"`
	TechnicalTermWeighting float64 `json:"technical_term_weighting"` // Weight boost for technical terms
	
	// Performance monitoring
	EnableMetrics   bool `json:"enable_metrics"`
	MetricsInterval time.Duration `json:"metrics_interval"`
}

// EmbeddingRequest represents a request for generating embeddings
type EmbeddingRequest struct {
	Texts       []string          `json:"texts"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	RequestID   string            `json:"request_id,omitempty"`
	Priority    int               `json:"priority"`           // Higher = more important
	ChunkIDs    []string          `json:"chunk_ids,omitempty"`
	UseCache    bool              `json:"use_cache"`
}

// EmbeddingResponse represents the response from embedding generation
type EmbeddingResponse struct {
	Embeddings     [][]float32       `json:"embeddings"`
	TokenUsage     TokenUsage        `json:"token_usage"`
	ProcessingTime time.Duration     `json:"processing_time"`
	CacheHits      int               `json:"cache_hits"`
	CacheMisses    int               `json:"cache_misses"`
	RequestID      string            `json:"request_id"`
	ModelUsed      string            `json:"model_used"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	TotalTokens      int `json:"total_tokens"`
	EstimatedCost    float64 `json:"estimated_cost"`
}

// EmbeddingMetrics tracks embedding service performance
type EmbeddingMetrics struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	TotalTexts         int64         `json:"total_texts"`
	TotalTokens        int64         `json:"total_tokens"`
	TotalCost          float64       `json:"total_cost"`
	
	AverageLatency     time.Duration `json:"average_latency"`
	AverageTextLength  float64       `json:"average_text_length"`
	
	CacheStats struct {
		TotalLookups   int64   `json:"total_lookups"`
		CacheHits      int64   `json:"cache_hits"`
		CacheMisses    int64   `json:"cache_misses"`
		HitRate        float64 `json:"hit_rate"`
	} `json:"cache_stats"`
	
	RateLimitingStats struct {
		RateLimitHits    int64         `json:"rate_limit_hits"`
		TotalWaitTime    time.Duration `json:"total_wait_time"`
		AverageWaitTime  time.Duration `json:"average_wait_time"`
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
type EmbeddingCache interface {
	Get(key string) ([]float32, bool)
	Set(key string, embedding []float32, ttl time.Duration) error
	Delete(key string) error
	Clear() error
	Stats() CacheStats
}

// CacheStats represents cache statistics
type CacheStats struct {
	Size     int64 `json:"size"`
	Hits     int64 `json:"hits"`
	Misses   int64 `json:"misses"`
	HitRate  float64 `json:"hit_rate"`
}

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
			MaxIdleConns:        100,
			MaxConnsPerHost:     config.MaxConcurrency,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
		},
	}

	// Create rate limiter
	rateLimiter := NewRateLimiter(config.RateLimit, config.TokenRateLimit)

	// Create cache
	var cache EmbeddingCache
	if config.EnableCaching {
		cache = NewInMemoryCache(10000) // 10k embeddings max
	} else {
		cache = &NoOpCache{}
	}

	service := &EmbeddingService{
		config:      config,
		logger:      slog.Default().With("component", "embedding-service"),
		metrics:     &EmbeddingMetrics{
			ModelStats:  make(map[string]ModelUsageStats),
			LastUpdated: time.Now(),
		},
		httpClient:  httpClient,
		rateLimiter: rateLimiter,
		cache:       cache,
	}

	// Start metrics collection if enabled
	if config.EnableMetrics {
		go service.startMetricsCollection()
	}

	return service
}

// getDefaultEmbeddingConfig returns default configuration
func getDefaultEmbeddingConfig() *EmbeddingConfig {
	return &EmbeddingConfig{
		Provider:               "openai",
		APIEndpoint:           "https://api.openai.com/v1/embeddings",
		ModelName:             "text-embedding-3-large",
		Dimensions:            3072,
		MaxTokens:             8191,
		BatchSize:             100,
		MaxConcurrency:        5,
		RequestTimeout:        30 * time.Second,
		RetryAttempts:         3,
		RetryDelay:            2 * time.Second,
		RateLimit:             60,   // 60 requests per minute
		TokenRateLimit:        150000, // 150k tokens per minute
		MinTextLength:         10,
		MaxTextLength:         8000,
		NormalizeText:         true,
		RemoveStopWords:       false,
		EnableCaching:         true,
		CacheTTL:              24 * time.Hour,
		TelecomPreprocessing:  true,
		PreserveTechnicalTerms: true,
		TechnicalTermWeighting: 1.2,
		EnableMetrics:         true,
		MetricsInterval:       5 * time.Minute,
	}
}

// GenerateEmbeddings generates embeddings for the provided texts
func (es *EmbeddingService) GenerateEmbeddings(ctx context.Context, request *EmbeddingRequest) (*EmbeddingResponse, error) {
	startTime := time.Now()

	if request == nil {
		return nil, fmt.Errorf("embedding request cannot be nil")
	}

	if len(request.Texts) == 0 {
		return nil, fmt.Errorf("no texts provided for embedding")
	}

	es.logger.Info("Generating embeddings",
		"text_count", len(request.Texts),
		"request_id", request.RequestID,
		"use_cache", request.UseCache,
	)

	// Preprocess texts
	processedTexts := es.preprocessTexts(request.Texts)

	// Check cache for existing embeddings if enabled
	var embeddings [][]float32
	var cacheHits, cacheMisses int
	var textsToEmbed []string
	var textIndices []int

	if request.UseCache && es.config.EnableCaching {
		embeddings = make([][]float32, len(processedTexts))
		
		for i, text := range processedTexts {
			cacheKey := es.generateCacheKey(text)
			if cached, found := es.cache.Get(cacheKey); found {
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

	// Generate embeddings for texts not in cache
	var tokenUsage TokenUsage
	if len(textsToEmbed) > 0 {
		newEmbeddings, usage, err := es.generateEmbeddingsBatch(ctx, textsToEmbed)
		if err != nil {
			es.updateMetrics(func(m *EmbeddingMetrics) {
				m.FailedRequests++
			})
			return nil, fmt.Errorf("failed to generate embeddings: %w", err)
		}

		tokenUsage = usage

		// Place new embeddings in the correct positions
		for i, newEmbedding := range newEmbeddings {
			originalIndex := textIndices[i]
			embeddings[originalIndex] = newEmbedding

			// Cache the new embedding if caching is enabled
			if request.UseCache && es.config.EnableCaching {
				cacheKey := es.generateCacheKey(textsToEmbed[i])
				if err := es.cache.Set(cacheKey, newEmbedding, es.config.CacheTTL); err != nil {
					es.logger.Warn("Failed to cache embedding", "error", err)
				}
			}
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

		// Update model stats
		modelStats := m.ModelStats[es.config.ModelName]
		modelStats.RequestCount++
		modelStats.TokenCount += int64(tokenUsage.TotalTokens)
		modelStats.TotalCost += tokenUsage.EstimatedCost
		if modelStats.RequestCount > 0 {
			modelStats.SuccessRate = float64(m.SuccessfulRequests) / float64(m.TotalRequests)
		}
		m.ModelStats[es.config.ModelName] = modelStats

		m.LastUpdated = time.Now()
	})

	response := &EmbeddingResponse{
		Embeddings:     embeddings,
		TokenUsage:     tokenUsage,
		ProcessingTime: processingTime,
		CacheHits:      cacheHits,
		CacheMisses:    cacheMisses,
		RequestID:      request.RequestID,
		ModelUsed:      es.config.ModelName,
		Metadata:       request.Metadata,
	}

	es.logger.Info("Embeddings generated successfully",
		"text_count", len(request.Texts),
		"processing_time", processingTime,
		"cache_hits", cacheHits,
		"cache_misses", cacheMisses,
	)

	return response, nil
}

// generateEmbeddingsBatch generates embeddings for a batch of texts
func (es *EmbeddingService) generateEmbeddingsBatch(ctx context.Context, texts []string) ([][]float32, TokenUsage, error) {
	// Process in smaller batches to respect API limits
	var allEmbeddings [][]float32
	var totalUsage TokenUsage

	for i := 0; i < len(texts); i += es.config.BatchSize {
		end := i + es.config.BatchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		
		// Wait for rate limiting
		if err := es.rateLimiter.Wait(ctx, len(batch)); err != nil {
			return nil, totalUsage, fmt.Errorf("rate limiting error: %w", err)
		}

		// Generate embeddings for this batch
		embeddings, usage, err := es.callEmbeddingAPI(ctx, batch)
		if err != nil {
			return nil, totalUsage, fmt.Errorf("API call failed for batch %d-%d: %w", i, end, err)
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
		totalUsage.PromptTokens += usage.PromptTokens
		totalUsage.TotalTokens += usage.TotalTokens
		totalUsage.EstimatedCost += usage.EstimatedCost
	}

	return allEmbeddings, totalUsage, nil
}

// callEmbeddingAPI makes the actual API call to generate embeddings
func (es *EmbeddingService) callEmbeddingAPI(ctx context.Context, texts []string) ([][]float32, TokenUsage, error) {
	switch es.config.Provider {
	case "openai":
		return es.callOpenAIAPI(ctx, texts)
	default:
		return nil, TokenUsage{}, fmt.Errorf("unsupported provider: %s", es.config.Provider)
	}
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
			`\b[A-Z]{2,}(?:-[A-Z]{2,})*\b`,           // Acronyms like RAN, AMF, SMF
			`\b\d+G\b`,                               // Technology generations
			`\b(?:Rel|Release)[-\s]*\d+\b`,           // Release versions
			`\b[vV]\d+\.\d+(?:\.\d+)?\b`,            // Version numbers
			`\b\d+\.\d+\.\d+\b`,                     // Specification numbers
			`\b[A-Z]+\d+\b`,                         // Standards like TS38.300
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

// generateCacheKey generates a cache key for the given text
func (es *EmbeddingService) generateCacheKey(text string) string {
	// Create a hash of the text and configuration that affects embeddings
	hasher := md5.New()
	hasher.Write([]byte(text))
	hasher.Write([]byte(es.config.ModelName))
	hasher.Write([]byte(fmt.Sprintf("%d", es.config.Dimensions)))
	return fmt.Sprintf("%x", hasher.Sum(nil))
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

func (c *NoOpCache) Get(key string) ([]float32, bool)                      { return nil, false }
func (c *NoOpCache) Set(key string, embedding []float32, ttl time.Duration) error { return nil }
func (c *NoOpCache) Delete(key string) error                               { return nil }
func (c *NoOpCache) Clear() error                                          { return nil }
func (c *NoOpCache) Stats() CacheStats                                     { return CacheStats{} }