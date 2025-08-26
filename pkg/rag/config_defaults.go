//go:build !disable_rag && !test

package rag

import (
	"time"
)

// getDefaultLoaderConfig returns default configuration for document loader
func getDefaultLoaderConfig() *DocumentLoaderConfig {
	return &DocumentLoaderConfig{
		LocalPaths:             []string{"./knowledge_base"},
		RemoteURLs:             []string{},
		MaxFileSize:            500 * 1024 * 1024, // 500MB for 3GPP specs
		PDFTextExtractor:       "hybrid",          // Use hybrid approach
		StreamingEnabled:       true,
		StreamingThreshold:     50 * 1024 * 1024,  // 50MB threshold
		MaxMemoryUsage:         200 * 1024 * 1024, // 200MB memory limit
		PageProcessingBatch:    10,                // Process 10 pages at a time
		EnableTableExtraction:  true,
		EnableFigureExtraction: true,
		OCREnabled:             false,
		OCRLanguage:            "eng",
		MinContentLength:       100,
		MaxContentLength:       1000000, // 1MB text
		LanguageFilter:         []string{"en", "eng", "english"},
		EnableCaching:          true,
		CacheDirectory:         "./cache/documents",
		CacheTTL:               24 * time.Hour,
		BatchSize:              10,
		MaxConcurrency:         5,
		ProcessingTimeout:      30 * time.Second,
		MaxRetries:             3,
		RetryDelay:             2 * time.Second,
		PreferredSources: map[string]int{
			"3GPP":  10,
			"O-RAN": 9,
			"ETSI":  8,
			"ITU":   7,
		},
		TechnicalDomains: []string{"RAN", "Core", "Transport", "Management", "O-RAN"},
	}
}

// getDefaultChunkingConfig returns default configuration for chunking service
func getDefaultChunkingConfig() *ChunkingConfig {
	return &ChunkingConfig{
		ChunkSize:               2000,
		ChunkOverlap:            200,
		MinChunkSize:            100,
		MaxChunkSize:            4000,
		PreserveHierarchy:       true,
		MaxHierarchyDepth:       5,
		IncludeParentContext:    true,
		UseSemanticBoundaries:   true,
		SentenceBoundaryWeight:  0.6,
		ParagraphBoundaryWeight: 0.8,
		SectionBoundaryWeight:   1.0,
		PreserveTechnicalTerms:  true,
		TechnicalTermPatterns: []string{
			`\b[A-Z]{2,}(?:-[A-Z]{2,})*\b`, // Acronyms
			`\b\d+G\b`,                     // Technology generations
			`\b(?:Rel|Release)[-\s]*\d+\b`, // Release versions
			`\b[vV]\d+\.\d+(?:\.\d+)?\b`,   // Version numbers
			`\b\d+\.\d+\.\d+\b`,            // Specification numbers
		},
		PreserveTablesAndFigures: true,
		PreserveCodeBlocks:       true,
		MinContentRatio:          0.6,
		MaxEmptyLines:            3,
		FilterNoiseContent:       true,
		BatchSize:                50,
		MaxConcurrency:           5,
		AddSectionHeaders:        true,
		AddDocumentMetadata:      true,
		AddChunkMetadata:         true,
	}
}

// getDefaultEmbeddingConfig returns default configuration for embedding service
func getDefaultEmbeddingConfig() *EmbeddingConfig {
	return &EmbeddingConfig{
		Provider:         "openai",
		APIEndpoint:      "https://api.openai.com/v1/embeddings",
		ModelName:        "text-embedding-3-large",
		Dimensions:       3072,
		MaxTokens:        8191,
		BatchSize:        100,
		MaxConcurrency:   5,
		RequestTimeout:   30 * time.Second,
		RetryAttempts:    3,
		RetryDelay:       2 * time.Second,
		RateLimit:        60,     // 60 requests per minute
		TokenRateLimit:   150000, // 150k tokens per minute
		MinTextLength:    10,
		MaxTextLength:    8000,
		NormalizeText:    true,
		RemoveStopWords:  false,
		EnableCaching:    true,
		CacheTTL:         24 * time.Hour,
		EnableRedisCache: true,
		RedisAddr:        "localhost:6379",
		RedisPassword:    "",
		RedisDB:          0,
		L1CacheSize:      10000, // 10k embeddings in memory
		L2CacheEnabled:   true,
		Providers: []ProviderConfig{
			{
				Name:         "openai",
				APIEndpoint:  "https://api.openai.com/v1/embeddings",
				ModelName:    "text-embedding-3-large",
				Dimensions:   3072,
				MaxTokens:    8191,
				CostPerToken: 0.00013, // $0.13 per 1M tokens
				RateLimit:    60,
				Priority:     1,
				Enabled:      true,
				Healthy:      true,
			},
			{
				Name:         "azure",
				APIEndpoint:  "https://your-resource.openai.azure.com/openai/deployments/your-deployment/embeddings",
				ModelName:    "text-embedding-ada-002",
				Dimensions:   1536,
				MaxTokens:    8191,
				CostPerToken: 0.0001, // Azure pricing
				RateLimit:    240,
				Priority:     2,
				Enabled:      false, // Disabled by default
				Healthy:      true,
			},
			{
				Name:         "huggingface",
				APIEndpoint:  "https://api-inference.huggingface.co/models/sentence-transformers/all-MiniLM-L6-v2",
				ModelName:    "sentence-transformers/all-MiniLM-L6-v2",
				Dimensions:   384,
				MaxTokens:    512,
				CostPerToken: 0.00001, // Very low cost
				RateLimit:    100,
				Priority:     3,
				Enabled:      false, // Disabled by default
				Healthy:      true,
			},
			{
				Name:         "cohere",
				APIEndpoint:  "https://api.cohere.ai/v1/embed",
				ModelName:    "embed-english-v3.0",
				Dimensions:   1024,
				MaxTokens:    512,
				CostPerToken: 0.0001,
				RateLimit:    100,
				Priority:     4,
				Enabled:      false, // Disabled by default
				Healthy:      true,
			},
		},
		FallbackEnabled:        true,
		FallbackOrder:          []string{"openai", "azure", "huggingface"},
		LoadBalancing:          "least_cost",
		HealthCheckInterval:    5 * time.Minute,
		EnableCostTracking:     true,
		DailyCostLimit:         50.0,   // $50 daily limit
		MonthlyCostLimit:       1000.0, // $1000 monthly limit
		CostAlertThreshold:     0.8,    // Alert at 80% of limit
		EnableQualityCheck:     true,
		MinQualityScore:        0.7,
		QualityCheckSample:     10,
		TelecomPreprocessing:   true,
		PreserveTechnicalTerms: true,
		TechnicalTermWeighting: 1.2,
		EnableMetrics:          true,
		MetricsInterval:        5 * time.Minute,
	}
}

// getDefaultRetrievalConfig returns default configuration for retrieval service
func getDefaultRetrievalConfig() *RetrievalConfig {
	return &RetrievalConfig{
		DefaultLimit:             20,
		MaxLimit:                 100,
		DefaultHybridAlpha:       0.7,
		MinConfidenceThreshold:   0.5,
		EnableQueryExpansion:     true,
		QueryExpansionTerms:      5,
		EnableQueryRewriting:     true,
		EnableSpellCorrection:    false, // Can be resource intensive
		EnableSynonymExpansion:   true,
		EnableSemanticReranking:  true,
		RerankingTopK:            50,
		CrossEncoderModel:        "ms-marco-MiniLM-L-6-v2",
		MaxContextLength:         8000,
		ContextOverlapRatio:      0.1,
		IncludeHierarchyInfo:     true,
		IncludeSourceMetadata:    true,
		EnableDiversityFiltering: true,
		DiversityThreshold:       0.8,
		BoostRecentDocuments:     true,
		RecencyBoostFactor:       1.2,
		EnableResultCaching:      true,
		ResultCacheTTL:           1 * time.Hour,
		MaxConcurrentQueries:     10,
		QueryTimeout:             30 * time.Second,
		IntentTypeWeights: map[string]float64{
			"configuration":   1.2,
			"troubleshooting": 1.1,
			"optimization":    1.0,
			"monitoring":      0.9,
			"general":         0.8,
		},
		TechnicalDomainBoosts: map[string]float64{
			"RAN":        1.2,
			"Core":       1.1,
			"Transport":  1.0,
			"Management": 0.9,
			"O-RAN":      1.3,
		},
		SourcePriorityWeights: map[string]float64{
			"3GPP":  1.0,
			"O-RAN": 1.1,
			"ETSI":  0.9,
			"ITU":   0.8,
		},
	}
}

// getDefaultRedisCacheConfig returns default configuration for Redis cache
func getDefaultRedisCacheConfig() *RedisCacheConfig {
	return &RedisCacheConfig{
		Address:            "localhost:6379",
		Password:           "",
		Database:           0,
		PoolSize:           20,               // Doubled for higher concurrency
		MinIdleConns:       5,                // More idle connections
		MaxRetries:         5,                // More retries for reliability
		DialTimeout:        3 * time.Second,  // Faster connection setup
		ReadTimeout:        1 * time.Second,  // Faster reads
		WriteTimeout:       1 * time.Second,  // Faster writes
		IdleTimeout:        10 * time.Minute, // Longer idle timeout
		DefaultTTL:         24 * time.Hour,
		MaxKeyLength:       250,
		EnableMetrics:      true,
		KeyPrefix:          "nephoran:rag:",
		EmbeddingTTL:       48 * time.Hour,     // Longer for expensive embeddings
		DocumentTTL:        7 * 24 * time.Hour, // 7 days
		QueryResultTTL:     2 * time.Hour,      // Longer cache for query results
		ContextTTL:         1 * time.Hour,      // Longer context cache
		EnableCompression:  true,
		CompressionLevel:   4,                  // Faster compression
		MaxValueSize:       20 * 1024 * 1024,   // 20MB for large documents
		EnableCleanup:      true,
		CleanupInterval:    30 * time.Minute,   // More frequent cleanup
		MaxMemoryThreshold: 0.75,               // Lower threshold for proactive cleanup
	}
}

// getDefaultMonitoringConfig returns default configuration for monitoring
func getDefaultMonitoringConfig() *MonitoringConfig {
	return &MonitoringConfig{
		MetricsPort:                8080,
		MetricsPath:                "/metrics",
		HealthCheckPath:            "/health",
		MetricsInterval:            30 * time.Second,
		HealthCheckInterval:        30 * time.Second,
		EnableAlerting:             false,
		AlertThresholds:            make(map[string]AlertThreshold),
		AlertWebhooks:              []string{},
		EnableStructuredLogs:       true,
		LogLevel:                   "info",
		TraceSampleRate:            0.1,
		EnableDistributedTracing:   false,
		EnableResourceMonitoring:   false,
		ResourceMonitoringInterval: 60 * time.Second,
	}
}

// DefaultPipelineConfiguration returns a complete default pipeline configuration
func DefaultPipelineConfiguration() *PipelineConfig {
	return &PipelineConfig{
		DocumentLoaderConfig: getDefaultLoaderConfig(),
		ChunkingConfig:       getDefaultChunkingConfig(),
		EmbeddingConfig:      getDefaultEmbeddingConfig(),
		WeaviateConfig: &WeaviateConfig{
			Host:   "localhost:8080",
			Scheme: "http",
		},
		RetrievalConfig:         getDefaultRetrievalConfig(),
		RedisCacheConfig:        getDefaultRedisCacheConfig(),
		MonitoringConfig:        getDefaultMonitoringConfig(),
		EnableCaching:           true,
		EnableMonitoring:        true,
		MaxConcurrentProcessing: 10,
		ProcessingTimeout:       5 * time.Minute,
		AutoIndexing:            true,
		IndexingInterval:        1 * time.Hour,
		EnableQualityChecks:     true,
		MinQualityThreshold:     0.7,
		LLMIntegration:          true,
		KubernetesIntegration:   true,
	}
}

// ProductionPipelineConfiguration returns a production-optimized configuration
func ProductionPipelineConfiguration() *PipelineConfig {
	config := DefaultPipelineConfiguration()

	// Production optimizations
	config.EnableMonitoring = true
	config.MonitoringConfig.LogLevel = "warn"

	// Higher performance settings
	config.MaxConcurrentProcessing = 20
	config.DocumentLoaderConfig.MaxConcurrency = 10
	config.ChunkingConfig.MaxConcurrency = 10
	config.EmbeddingConfig.MaxConcurrency = 10
	config.RetrievalConfig.MaxConcurrentQueries = 20

	// Longer cache TTLs for production
	config.RedisCacheConfig.EmbeddingTTL = 3 * 24 * time.Hour // 3 days
	config.RedisCacheConfig.DocumentTTL = 7 * 24 * time.Hour  // 7 days
	config.RedisCacheConfig.QueryResultTTL = 2 * time.Hour    // 2 hours
	config.RedisCacheConfig.ContextTTL = 1 * time.Hour        // 1 hour

	// More conservative quality thresholds
	config.MinQualityThreshold = 0.8
	config.EmbeddingConfig.MinQualityScore = 0.8

	// Production-level limits
	config.EmbeddingConfig.DailyCostLimit = 200.0
	config.EmbeddingConfig.MonthlyCostLimit = 5000.0

	return config
}

// DevelopmentPipelineConfiguration returns a development-optimized configuration
func DevelopmentPipelineConfiguration() *PipelineConfig {
	config := DefaultPipelineConfiguration()

	// Development optimizations
	config.MonitoringConfig.LogLevel = "debug"

	// Smaller limits for development
	config.DocumentLoaderConfig.MaxFileSize = 50 * 1024 * 1024 // 50MB
	config.EmbeddingConfig.DailyCostLimit = 10.0               // $10 daily limit
	config.EmbeddingConfig.MonthlyCostLimit = 100.0            // $100 monthly limit

	// More frequent health checks
	config.MonitoringConfig.HealthCheckInterval = 10 * time.Second
	config.EmbeddingConfig.HealthCheckInterval = 1 * time.Minute

	// Shorter cache TTLs for faster iteration
	config.RedisCacheConfig.EmbeddingTTL = 1 * time.Hour
	config.RedisCacheConfig.DocumentTTL = 4 * time.Hour
	config.RedisCacheConfig.QueryResultTTL = 15 * time.Minute
	config.RedisCacheConfig.ContextTTL = 5 * time.Minute

	// Enable local provider for testing
	for i, provider := range config.EmbeddingConfig.Providers {
		if provider.Name == "local" {
			config.EmbeddingConfig.Providers[i].Enabled = true
			break
		}
	}

	return config
}

// TestPipelineConfiguration returns a test-optimized configuration
func TestPipelineConfiguration() *PipelineConfig {
	config := DefaultPipelineConfiguration()

	// Test optimizations
	config.EnableCaching = false       // Disable caching for consistent tests
	config.EnableMonitoring = false    // Disable monitoring for simpler tests
	config.AutoIndexing = false        // No background tasks
	config.EnableQualityChecks = false // Skip quality checks for speed

	// Minimal processing for tests
	config.MaxConcurrentProcessing = 2
	config.DocumentLoaderConfig.MaxConcurrency = 2
	config.ChunkingConfig.MaxConcurrency = 2
	config.EmbeddingConfig.MaxConcurrency = 2
	config.RetrievalConfig.MaxConcurrentQueries = 2

	// Short timeouts for fast test execution
	config.ProcessingTimeout = 10 * time.Second
	config.DocumentLoaderConfig.ProcessingTimeout = 5 * time.Second
	config.EmbeddingConfig.RequestTimeout = 5 * time.Second
	config.RetrievalConfig.QueryTimeout = 5 * time.Second

	// Smaller batch sizes
	config.DocumentLoaderConfig.BatchSize = 2
	config.EmbeddingConfig.BatchSize = 2
	config.ChunkingConfig.BatchSize = 5

	// No cost limits for tests
	config.EmbeddingConfig.EnableCostTracking = false

	// Use mock providers where possible
	for i, provider := range config.EmbeddingConfig.Providers {
		config.EmbeddingConfig.Providers[i].Enabled = false
		if provider.Name == "local" {
			config.EmbeddingConfig.Providers[i].Enabled = true
		}
	}

	return config
}
