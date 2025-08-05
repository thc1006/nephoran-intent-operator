//go:build !disable_rag && !test

package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v5/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

// WeaviateClient provides a production-ready client for Weaviate vector database
type WeaviateClient struct {
	client           *weaviate.Client
	config           *WeaviateConfig
	logger           *slog.Logger
	healthStatus     *WeaviateHealthStatus
	circuitBreaker   *CircuitBreaker
	rateLimiter      *WeaviateRateLimiter
	embeddingFallback *EmbeddingFallback
	mutex            sync.RWMutex
}

// CircuitBreaker implements circuit breaker pattern for API calls
type CircuitBreaker struct {
	maxFailures    int32
	timeout        time.Duration
	currentState   int32 // 0: Closed, 1: Open, 2: HalfOpen
	failureCount   int32
	lastFailTime   int64
	successCount   int32
	mutex          sync.RWMutex
}

// Circuit breaker states
const (
	CircuitClosed   = 0
	CircuitOpen     = 1
	CircuitHalfOpen = 2
)

// RateLimiter implements token bucket rate limiting for OpenAI API calls
type WeaviateRateLimiter struct {
	requestsPerMinute int32
	tokensPerMinute   int64
	requestTokens     int32
	tokenBucketSize   int64
	lastRefill        int64
	requestBucket     chan struct{}
	tokenBucket       int64
	mutex             sync.Mutex
}

// EmbeddingFallback provides local embedding model fallback
type EmbeddingFallback struct {
	enabled     bool
	modelPath   string
	dimensions  int
	available   bool
	mutex       sync.RWMutex
}

// RetryConfig holds configuration for exponential backoff
type RetryConfig struct {
	MaxRetries   int           `json:"max_retries"`
	BaseDelay    time.Duration `json:"base_delay"`
	MaxDelay     time.Duration `json:"max_delay"`
	BackoffMultiplier float64  `json:"backoff_multiplier"`
	Jitter       bool          `json:"jitter"`
}

// WeaviateConfig holds configuration for the Weaviate client
type WeaviateConfig struct {
	Host     string `json:"host"`
	Scheme   string `json:"scheme"`
	APIKey   string `json:"api_key"`
	Headers  map[string]string `json:"headers"`
	Timeout  time.Duration `json:"timeout"`
	Retries  int `json:"retries"`
	
	// Schema configuration
	AutoSchema bool `json:"auto_schema"`
	
	// OpenAI configuration for vectorization
	OpenAIAPIKey string `json:"openai_api_key"`
	
	// Rate limiting configuration
	RateLimiting struct {
		RequestsPerMinute int32 `json:"requests_per_minute"`
		TokensPerMinute   int64 `json:"tokens_per_minute"`
		Enabled           bool  `json:"enabled"`
	} `json:"rate_limiting"`
	
	// Circuit breaker configuration
	CircuitBreaker struct {
		MaxFailures int32         `json:"max_failures"`
		Timeout     time.Duration `json:"timeout"`
		Enabled     bool          `json:"enabled"`
	} `json:"circuit_breaker"`
	
	// Retry configuration
	Retry RetryConfig `json:"retry"`
	
	// Embedding fallback configuration
	EmbeddingFallback struct {
		Enabled     bool   `json:"enabled"`
		ModelPath   string `json:"model_path"`
		Dimensions  int    `json:"dimensions"`
	} `json:"embedding_fallback"`
	
	// Performance tuning
	ConnectionPool struct {
		MaxIdleConns        int           `json:"max_idle_conns"`
		MaxConnsPerHost     int           `json:"max_conns_per_host"`
		IdleConnTimeout     time.Duration `json:"idle_conn_timeout"`
		DisableCompression  bool          `json:"disable_compression"`
	} `json:"connection_pool"`
}

// HealthStatus represents the health status of the Weaviate cluster
type WeaviateHealthStatus struct {
	IsHealthy     bool              `json:"is_healthy"`
	LastCheck     time.Time         `json:"last_check"`
	Version       string            `json:"version"`
	Nodes         []NodeStatus      `json:"nodes"`
	Statistics    ClusterStatistics `json:"statistics"`
	Details       string            `json:"details"`
	ErrorCount    int64             `json:"error_count"`
	LastError     error             `json:"last_error,omitempty"`
}

// NodeStatus represents the status of a single Weaviate node
type NodeStatus struct {
	Name      string                 `json:"name"`
	Status    string                 `json:"status"`
	Version   string                 `json:"version"`
	GitHash   string                 `json:"git_hash"`
	Stats     map[string]interface{} `json:"stats"`
}

// ClusterStatistics holds cluster-wide statistics
type ClusterStatistics struct {
	ObjectCount    int64                  `json:"object_count"`
	ClassCount     int                    `json:"class_count"`
	ShardCount     int                    `json:"shard_count"`
	VectorDimensions map[string]int       `json:"vector_dimensions"`
	IndexSize      int64                  `json:"index_size"`
	QueryLatency   LatencyMetrics         `json:"query_latency"`
}

// LatencyMetrics holds latency statistics
type LatencyMetrics struct {
	P50 time.Duration `json:"p50"`
	P95 time.Duration `json:"p95"`
	P99 time.Duration `json:"p99"`
}

// TelecomDocument represents a document in the telecom knowledge base
type TelecomDocument struct {
	ID             string            `json:"id"`
	Content        string            `json:"content"`
	Title          string            `json:"title"`
	Source         string            `json:"source"`
	Category       string            `json:"category"`
	Subcategory    string            `json:"subcategory"`
	Version        string            `json:"version"`
	WorkingGroup   string            `json:"working_group"`
	Keywords       []string          `json:"keywords"`
	Confidence     float32           `json:"confidence"`
	Language       string            `json:"language"`
	DocumentType   string            `json:"document_type"`
	NetworkFunction []string         `json:"network_function"`
	Technology     []string          `json:"technology"`
	UseCase        []string          `json:"use_case"`
	// Timestamp removed to avoid conflicts
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// SearchQuery represents a query to the vector database
type SearchQuery struct {
	Query            string                 `json:"query"`
	Limit            int                    `json:"limit"`
	Offset           int                    `json:"offset"`
	Filters          map[string]interface{} `json:"filters,omitempty"`
	HybridSearch     bool                   `json:"hybrid_search"`
	HybridAlpha      float32                `json:"hybrid_alpha"`
	UseReranker      bool                   `json:"use_reranker"`
	MinConfidence    float32                `json:"min_confidence"`
	IncludeVector    bool                   `json:"include_vector"`
	ExpandQuery      bool                   `json:"expand_query"`
}

// SearchResult represents a search result from the vector database
type SearchResult struct {
	Document        *shared.TelecomDocument `json:"document"`
	Score           float32          `json:"score"`
	Distance        float32          `json:"distance"`
	Vector          []float32        `json:"vector,omitempty"`
	ExplainScore    string           `json:"explain_score,omitempty"`
}

// SearchResponse represents the response from a search query
type SearchResponse struct {
	Results       []*SearchResult `json:"results"`
	Total         int             `json:"total"`
	Took          time.Duration   `json:"took"`
	Query         string          `json:"query"`
	ProcessedAt   time.Time       `json:"processed_at"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// NewWeaviateClient creates a new Weaviate client with production-ready configuration
func NewWeaviateClient(config *WeaviateConfig) (*WeaviateClient, error) {
	if config == nil {
		return nil, fmt.Errorf("weaviate config cannot be nil")
	}

	// Set default values
	if config.Scheme == "" {
		config.Scheme = "https"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.Retries == 0 {
		config.Retries = 3
	}

	// Set retry configuration defaults
	if config.Retry.MaxRetries == 0 {
		config.Retry.MaxRetries = 3
	}
	if config.Retry.BaseDelay == 0 {
		config.Retry.BaseDelay = 1 * time.Second
	}
	if config.Retry.MaxDelay == 0 {
		config.Retry.MaxDelay = 30 * time.Second
	}
	if config.Retry.BackoffMultiplier == 0 {
		config.Retry.BackoffMultiplier = 2.0
	}
	config.Retry.Jitter = true

	// Set circuit breaker defaults
	if config.CircuitBreaker.MaxFailures == 0 {
		config.CircuitBreaker.MaxFailures = 5
	}
	if config.CircuitBreaker.Timeout == 0 {
		config.CircuitBreaker.Timeout = 60 * time.Second
	}
	config.CircuitBreaker.Enabled = true

	// Set rate limiting defaults
	if config.RateLimiting.RequestsPerMinute == 0 {
		config.RateLimiting.RequestsPerMinute = 3000 // Conservative OpenAI API limit
	}
	if config.RateLimiting.TokensPerMinute == 0 {
		config.RateLimiting.TokensPerMinute = 1000000 // Conservative token limit
	}
	config.RateLimiting.Enabled = true

	// Configure authentication
	var authConfig auth.Config
	if config.APIKey != "" {
		authConfig = auth.ApiKey{Value: config.APIKey}
	}

	// Create HTTP client with optimized settings
	httpClient := &http.Client{
		Timeout: config.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:          config.ConnectionPool.MaxIdleConns,
			MaxConnsPerHost:       config.ConnectionPool.MaxConnsPerHost,
			IdleConnTimeout:       config.ConnectionPool.IdleConnTimeout,
			DisableCompression:    config.ConnectionPool.DisableCompression,
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}

	// Create Weaviate client configuration
	clientConfig := weaviate.Config{
		Host:       config.Host,
		Scheme:     config.Scheme,
		AuthConfig: authConfig,
		Headers:    config.Headers,
		Client:     httpClient,
	}

	// Create client
	client, err := weaviate.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create weaviate client: %w", err)
	}

	wc := &WeaviateClient{
		client: client,
		config: config,
		logger: slog.Default().With("component", "weaviate-client"),
		healthStatus: &WeaviateHealthStatus{
			LastCheck: time.Now(),
		},
		circuitBreaker: &CircuitBreaker{
			maxFailures:  config.CircuitBreaker.MaxFailures,
			timeout:      config.CircuitBreaker.Timeout,
			currentState: CircuitClosed,
		},
		rateLimiter: &WeaviateRateLimiter{
			requestsPerMinute: config.RateLimiting.RequestsPerMinute,
			tokensPerMinute:   config.RateLimiting.TokensPerMinute,
			requestBucket:     make(chan struct{}, config.RateLimiting.RequestsPerMinute),
			lastRefill:        time.Now().Unix(),
		},
		embeddingFallback: &EmbeddingFallback{
			enabled:    config.EmbeddingFallback.Enabled,
			modelPath:  config.EmbeddingFallback.ModelPath,
			dimensions: config.EmbeddingFallback.Dimensions,
			available:  false, // Will be set during initialization
		},
	}

	// Initialize rate limiter buckets
	wc.initializeRateLimiter()

	// Initialize embedding fallback if enabled
	if wc.embeddingFallback.enabled {
		if err := wc.initializeEmbeddingFallback(); err != nil {
			wc.logger.Warn("Failed to initialize embedding fallback", "error", err)
		}
	}

	// Initialize schema if auto-schema is enabled
	if config.AutoSchema {
		if err := wc.initializeSchemaWithRetry(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to initialize schema: %w", err)
		}
	}

	// Start background services
	go wc.startHealthChecking()
	go wc.startRateLimiterRefresh()

	return wc, nil
}

// initializeSchema creates the telecom-specific schema classes
func (wc *WeaviateClient) initializeSchema(ctx context.Context) error {
	wc.logger.Info("Initializing Weaviate schema for telecom domain")

	// Define TelecomKnowledge class
	telecomKnowledgeClass := &models.Class{
		Class:       "TelecomKnowledge",
		Description: "Comprehensive telecommunications domain knowledge base",
		Vectorizer:  "text2vec-openai",
		ModuleConfig: map[string]interface{}{
			"text2vec-openai": map[string]interface{}{
				"model":      "text-embedding-3-large",
				"dimensions": 3072,
				"type":       "text",
			},
		},
		Properties: []*models.Property{
			{
				Name:         "content",
				DataType:     []string{"text"},
				Description:  "Full document content with telecom terminology",
				IndexFilterable: &[]bool{true}[0],
				IndexSearchable: &[]bool{true}[0],
			},
			{
				Name:         "title",
				DataType:     []string{"text"},
				Description:  "Document or section title",
				IndexFilterable: &[]bool{true}[0],
				IndexSearchable: &[]bool{true}[0],
			},
			{
				Name:         "source",
				DataType:     []string{"text"},
				Description:  "Document source organization (3GPP, O-RAN, ETSI, ITU, etc.)",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "category",
				DataType:     []string{"text"},
				Description:  "Knowledge category (RAN, Core, Transport, Management, etc.)",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "subcategory",
				DataType:     []string{"text"},
				Description:  "Detailed subcategory for fine-grained classification",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "version",
				DataType:     []string{"text"},
				Description:  "Specification version (e.g., Rel-17, v1.5.0)",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "workingGroup",
				DataType:     []string{"text"},
				Description:  "Working group responsible (e.g., RAN1, SA2, O-RAN WG1)",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "keywords",
				DataType:     []string{"text[]"},
				Description:  "Extracted telecom keywords and terminology",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "confidence",
				DataType:     []string{"number"},
				Description:  "Content quality and relevance confidence score",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "language",
				DataType:     []string{"text"},
				Description:  "Document language (ISO 639-1 code)",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "documentType",
				DataType:     []string{"text"},
				Description:  "Type of document (specification, report, presentation, etc.)",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "networkFunction",
				DataType:     []string{"text[]"},
				Description:  "Related network functions (gNB, AMF, SMF, UPF, etc.)",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "technology",
				DataType:     []string{"text[]"},
				Description:  "Related technologies (5G, 4G, O-RAN, vRAN, etc.)",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "useCase",
				DataType:     []string{"text[]"},
				Description:  "Applicable use cases (eMBB, URLLC, mMTC, etc.)",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "timestamp",
				DataType:     []string{"date"},
				Description:  "Last updated timestamp",
				IndexFilterable: &[]bool{true}[0],
			},
		},
	}

	// Create the class
	err := wc.client.Schema().ClassCreator().WithClass(telecomKnowledgeClass).Do(ctx)
	if err != nil {
		// Check if class already exists
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create TelecomKnowledge class: %w", err)
		}
		wc.logger.Info("TelecomKnowledge class already exists")
	} else {
		wc.logger.Info("Created TelecomKnowledge class successfully")
	}

	// Define IntentPatterns class
	intentPatternsClass := &models.Class{
		Class:       "IntentPatterns",
		Description: "Intent processing patterns and templates for telecom operations",
		Vectorizer:  "text2vec-openai",
		ModuleConfig: map[string]interface{}{
			"text2vec-openai": map[string]interface{}{
				"model":      "text-embedding-3-large",
				"dimensions": 3072,
				"type":       "text",
			},
		},
		Properties: []*models.Property{
			{
				Name:         "pattern",
				DataType:     []string{"text"},
				Description:  "Intent pattern template",
				IndexFilterable: &[]bool{true}[0],
				IndexSearchable: &[]bool{true}[0],
			},
			{
				Name:         "category",
				DataType:     []string{"text"},
				Description:  "Intent category (configuration, optimization, troubleshooting)",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "networkDomain",
				DataType:     []string{"text"},
				Description:  "Target network domain (RAN, Core, Transport)",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "parameters",
				DataType:     []string{"text[]"},
				Description:  "Required and optional parameters",
				IndexFilterable: &[]bool{true}[0],
			},
			{
				Name:         "examples",
				DataType:     []string{"text[]"},
				Description:  "Example intent statements",
				IndexSearchable: &[]bool{true}[0],
			},
			{
				Name:         "confidence",
				DataType:     []string{"number"},
				Description:  "Pattern matching confidence threshold",
				IndexFilterable: &[]bool{true}[0],
			},
		},
	}

	// Create IntentPatterns class
	err = wc.client.Schema().ClassCreator().WithClass(intentPatternsClass).Do(ctx)
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create IntentPatterns class: %w", err)
		}
		wc.logger.Info("IntentPatterns class already exists")
	} else {
		wc.logger.Info("Created IntentPatterns class successfully")
	}

	return nil
}

// Search performs a hybrid search across the telecom knowledge base
func (wc *WeaviateClient) Search(ctx context.Context, query *SearchQuery) (*SearchResponse, error) {
	startTime := time.Now()

	// Validate query
	if query == nil {
		return nil, fmt.Errorf("search query cannot be nil")
	}
	if query.Query == "" {
		return nil, fmt.Errorf("search query text cannot be empty")
	}

	// Set defaults
	if query.Limit == 0 {
		query.Limit = 10
	}
	if query.HybridAlpha == 0 {
		query.HybridAlpha = 0.7 // 70% vector, 30% keyword
	}

	// Build the GraphQL query
	var gqlQuery *graphql.NearTextArgumentBuilder
	if query.HybridSearch {
		// Use hybrid search
		gqlQuery = wc.client.GraphQL().Get().
			WithClassName("TelecomKnowledge").
			WithHybrid(
				wc.client.GraphQL().HybridArgumentBuilder().
					WithQuery(query.Query).
					WithAlpha(query.HybridAlpha),
			)
	} else {
		// Use pure vector search
		gqlQuery = wc.client.GraphQL().Get().
			WithClassName("TelecomKnowledge").
			WithNearText(
				wc.client.GraphQL().NearTextArgumentBuilder().
					WithConcepts([]string{query.Query}),
			)
	}

	// Add fields to retrieve
	fields := []graphql.Field{
		{Name: "content"},
		{Name: "title"},
		{Name: "source"},
		{Name: "category"},
		{Name: "subcategory"},
		{Name: "version"},
		{Name: "workingGroup"},
		{Name: "keywords"},
		{Name: "confidence"},
		{Name: "language"},
		{Name: "documentType"},
		{Name: "networkFunction"},
		{Name: "technology"},
		{Name: "useCase"},
		{Name: "timestamp"},
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "id"},
			{Name: "score"},
			{Name: "distance"},
		}},
	}

	if query.IncludeVector {
		fields = append(fields, graphql.Field{Name: "_additional", Fields: []graphql.Field{
			{Name: "vector"},
		}})
	}

	// Apply filters if provided
	if len(query.Filters) > 0 {
		where := wc.buildWhereFilter(query.Filters)
		if where != nil {
			gqlQuery = gqlQuery.WithWhere(where)
		}
	}

	// Execute the query
	result, err := gqlQuery.
		WithFields(fields...).
		WithLimit(query.Limit).
		WithOffset(query.Offset).
		Do(ctx)

	if err != nil {
		wc.logger.Error("Weaviate search failed", "error", err, "query", query.Query)
		return nil, fmt.Errorf("weaviate search failed: %w", err)
	}

	// Parse results
	searchResponse := &SearchResponse{
		Results:     make([]*SearchResult, 0),
		Query:       query.Query,
		ProcessedAt: time.Now(),
		Took:        time.Since(startTime),
		Metadata: map[string]interface{}{
			"hybrid_search": query.HybridSearch,
			"hybrid_alpha":  query.HybridAlpha,
		},
	}

	// Extract data from GraphQL result
	if result.Data != nil {
		if data, ok := result.Data["Get"].(map[string]interface{}); ok {
			if telecomData, ok := data["TelecomKnowledge"].([]interface{}); ok {
				for _, item := range telecomData {
					if itemMap, ok := item.(map[string]interface{}); ok {
						searchResult := wc.parseSearchResult(itemMap)
						if searchResult != nil && searchResult.Document.Confidence >= query.MinConfidence {
							searchResponse.Results = append(searchResponse.Results, searchResult)
						}
					}
				}
			}
		}
	}

	searchResponse.Total = len(searchResponse.Results)

	wc.logger.Info("Weaviate search completed",
		"query", query.Query,
		"results", searchResponse.Total,
		"took", searchResponse.Took,
	)

	return searchResponse, nil
}

// parseSearchResult converts a GraphQL result item to a SearchResult
func (wc *WeaviateClient) parseSearchResult(item map[string]interface{}) *SearchResult {
	doc := &shared.TelecomDocument{}
	result := &SearchResult{Document: doc}

	// Parse document fields
	if val, ok := item["content"].(string); ok {
		doc.Content = val
	}
	if val, ok := item["title"].(string); ok {
		doc.Title = val
	}
	if val, ok := item["source"].(string); ok {
		doc.Source = val
	}
	if val, ok := item["category"].(string); ok {
		doc.Category = val
	}
	// subcategory field not available in shared.TelecomDocument, skip
	if val, ok := item["version"].(string); ok {
		doc.Version = val
	}
	// workingGroup field not available in shared.TelecomDocument, skip
	if val, ok := item["confidence"].(float64); ok {
		doc.Confidence = float32(val)
	}
	if val, ok := item["language"].(string); ok {
		doc.Language = val
	}
	if val, ok := item["documentType"].(string); ok {
		doc.DocumentType = val
	}

	// Parse array fields
	if val, ok := item["keywords"].([]interface{}); ok {
		doc.Keywords = make([]string, len(val))
		for i, v := range val {
			if str, ok := v.(string); ok {
				doc.Keywords[i] = str
			}
		}
	}
	if val, ok := item["networkFunction"].([]interface{}); ok {
		doc.NetworkFunction = make([]string, len(val))
		for i, v := range val {
			if str, ok := v.(string); ok {
				doc.NetworkFunction[i] = str
			}
		}
	}
	if val, ok := item["technology"].([]interface{}); ok {
		doc.Technology = make([]string, len(val))
		for i, v := range val {
			if str, ok := v.(string); ok {
				doc.Technology[i] = str
			}
		}
	}
	if val, ok := item["useCase"].([]interface{}); ok {
		doc.UseCase = make([]string, len(val))
		for i, v := range val {
			if str, ok := v.(string); ok {
				doc.UseCase[i] = str
			}
		}
	}

	// Parse timestamp
	if val, ok := item["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			doc.UpdatedAt = t
		}
	}

	// Parse additional fields
	if additional, ok := item["_additional"].(map[string]interface{}); ok {
		if id, ok := additional["id"].(string); ok {
			doc.ID = id
		}
		if score, ok := additional["score"].(float64); ok {
			result.Score = float32(score)
		}
		if distance, ok := additional["distance"].(float64); ok {
			result.Distance = float32(distance)
		}
		if vector, ok := additional["vector"].([]interface{}); ok {
			result.Vector = make([]float32, len(vector))
			for i, v := range vector {
				if f, ok := v.(float64); ok {
					result.Vector[i] = float32(f)
				}
			}
		}
	}

	return result
}

// buildWhereFilter constructs a WHERE filter from the query filters
func (wc *WeaviateClient) buildWhereFilter(filters map[string]interface{}) interface{} {
	if len(filters) == 0 {
		return nil
	}

	// For now, implement basic filtering
	// In a full implementation, you would build complex filters
	where := wc.client.GraphQL().WhereArgumentBuilder()

	// Example: filter by source
	if source, ok := filters["source"].(string); ok && source != "" {
		where = where.WithPath([]string{"source"}).
			WithOperator(graphql.Equal).
			WithValueText(source)
	}

	// Example: filter by category
	if category, ok := filters["category"].(string); ok && category != "" {
		where = where.WithPath([]string{"category"}).
			WithOperator(graphql.Equal).
			WithValueText(category)
	}

	return where
}

// AddDocument adds a new document to the telecom knowledge base
func (wc *WeaviateClient) AddDocument(ctx context.Context, doc *TelecomDocument) error {
	if doc == nil {
		return fmt.Errorf("document cannot be nil")
	}

	// Prepare the object
	properties := map[string]interface{}{
		"content":         doc.Content,
		"title":           doc.Title,
		"source":          doc.Source,
		"category":        doc.Category,
		"subcategory":     doc.Subcategory,
		"version":         doc.Version,
		"workingGroup":    doc.WorkingGroup,
		"keywords":        doc.Keywords,
		"confidence":      doc.Confidence,
		"language":        doc.Language,
		"documentType":    doc.DocumentType,
		"networkFunction": doc.NetworkFunction,
		"technology":      doc.Technology,
		"useCase":         doc.UseCase,
		"timestamp":       doc.UpdatedAt.Format(time.RFC3339),
	}

	// Create the object
	_, err := wc.client.Data().Creator().
		WithClassName("TelecomKnowledge").
		WithID(doc.ID).
		WithProperties(properties).
		Do(ctx)

	if err != nil {
		wc.logger.Error("Failed to add document", "error", err, "doc_id", doc.ID)
		return fmt.Errorf("failed to add document: %w", err)
	}

	wc.logger.Info("Document added successfully", "doc_id", doc.ID, "title", doc.Title)
	return nil
}

// GetHealthStatus returns the current health status of the Weaviate cluster
func (wc *WeaviateClient) GetHealthStatus() *WeaviateHealthStatus {
	wc.mutex.RLock()
	defer wc.mutex.RUnlock()
	
	// Return a copy to prevent modifications
	status := *wc.healthStatus
	return &status
}

// startHealthChecking starts the background health checking routine
func (wc *WeaviateClient) startHealthChecking() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wc.checkHealth()
		}
	}
}

// checkHealth performs a health check on the Weaviate cluster
func (wc *WeaviateClient) checkHealth() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wc.mutex.Lock()
	defer wc.mutex.Unlock()

	// Check if cluster is reachable
	ready, err := wc.client.Misc().ReadyChecker().Do(ctx)
	if err != nil {
		wc.healthStatus.IsHealthy = false
		wc.healthStatus.LastError = err
		wc.healthStatus.ErrorCount++
		wc.logger.Error("Weaviate health check failed", "error", err)
		return
	}

	wc.healthStatus.IsHealthy = ready
	wc.healthStatus.LastCheck = time.Now()
	wc.healthStatus.LastError = nil

	// Get cluster metadata
	meta, err := wc.client.Misc().MetaGetter().Do(ctx)
	if err != nil {
		wc.logger.Warn("Failed to get cluster metadata", "error", err)
		return
	}

	if meta != nil {
		wc.healthStatus.Version = meta.Version
	}

	wc.logger.Debug("Weaviate health check completed", "healthy", wc.healthStatus.IsHealthy)
}

// initializeRateLimiter sets up the rate limiter token buckets
func (wc *WeaviateClient) initializeRateLimiter() {
	// Fill initial request bucket
	for i := int32(0); i < wc.rateLimiter.requestsPerMinute; i++ {
		select {
		case wc.rateLimiter.requestBucket <- struct{}{}:
		default:
			break
		}
	}
	wc.rateLimiter.tokenBucket = wc.rateLimiter.tokensPerMinute
	wc.logger.Info("Rate limiter initialized", 
		"requests_per_minute", wc.rateLimiter.requestsPerMinute,
		"tokens_per_minute", wc.rateLimiter.tokensPerMinute)
}

// startRateLimiterRefresh starts the background rate limiter token refresh
func (wc *WeaviateClient) startRateLimiterRefresh() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wc.refillRateLimiter()
		}
	}
}

// refillRateLimiter refills the rate limiter buckets
func (wc *WeaviateClient) refillRateLimiter() {
	wc.rateLimiter.mutex.Lock()
	defer wc.rateLimiter.mutex.Unlock()

	// Refill request bucket
	for i := int32(0); i < wc.rateLimiter.requestsPerMinute; i++ {
		select {
		case wc.rateLimiter.requestBucket <- struct{}{}:
		default:
			break
		}
	}

	// Refill token bucket
	wc.rateLimiter.tokenBucket = wc.rateLimiter.tokensPerMinute
	wc.rateLimiter.lastRefill = time.Now().Unix()
}

// checkRateLimit checks if we can make a request within rate limits
func (wc *WeaviateClient) checkRateLimit(estimatedTokens int64) error {
	// Check request limit
	select {
	case <-wc.rateLimiter.requestBucket:
		// Got a request token
	case <-time.After(time.Minute):
		return fmt.Errorf("rate limit exceeded: too many requests per minute")
	}

	// Check token limit
	wc.rateLimiter.mutex.Lock()
	defer wc.rateLimiter.mutex.Unlock()

	if wc.rateLimiter.tokenBucket < estimatedTokens {
		return fmt.Errorf("rate limit exceeded: insufficient tokens (need %d, have %d)", 
			estimatedTokens, wc.rateLimiter.tokenBucket)
	}

	wc.rateLimiter.tokenBucket -= estimatedTokens
	return nil
}

// checkCircuitBreaker checks if circuit breaker allows the operation
func (wc *WeaviateClient) checkCircuitBreaker() error {
	wc.circuitBreaker.mutex.RLock()
	state := atomic.LoadInt32(&wc.circuitBreaker.currentState)
	lastFailTime := atomic.LoadInt64(&wc.circuitBreaker.lastFailTime)
	wc.circuitBreaker.mutex.RUnlock()

	switch state {
	case CircuitClosed:
		return nil
	case CircuitOpen:
		// Check if timeout period has passed
		if time.Since(time.Unix(0, lastFailTime)) > wc.circuitBreaker.timeout {
			// Transition to half-open
			atomic.StoreInt32(&wc.circuitBreaker.currentState, CircuitHalfOpen)
			atomic.StoreInt32(&wc.circuitBreaker.successCount, 0)
			wc.logger.Info("Circuit breaker transitioning to half-open state")
			return nil
		}
		return fmt.Errorf("circuit breaker is open")
	case CircuitHalfOpen:
		return nil
	default:
		return fmt.Errorf("unknown circuit breaker state: %d", state)
	}
}

// recordCircuitBreakerSuccess records a successful operation
func (wc *WeaviateClient) recordCircuitBreakerSuccess() {
	state := atomic.LoadInt32(&wc.circuitBreaker.currentState)
	if state == CircuitHalfOpen {
		successCount := atomic.AddInt32(&wc.circuitBreaker.successCount, 1)
		if successCount >= 3 { // Require 3 successes to close
			atomic.StoreInt32(&wc.circuitBreaker.currentState, CircuitClosed)
			atomic.StoreInt32(&wc.circuitBreaker.failureCount, 0)
			wc.logger.Info("Circuit breaker closed after successful operations")
		}
	} else if state == CircuitClosed {
		// Reset failure count on success
		atomic.StoreInt32(&wc.circuitBreaker.failureCount, 0)
	}
}

// recordCircuitBreakerFailure records a failed operation
func (wc *WeaviateClient) recordCircuitBreakerFailure() {
	failureCount := atomic.AddInt32(&wc.circuitBreaker.failureCount, 1)
	
	if failureCount >= wc.circuitBreaker.maxFailures {
		atomic.StoreInt32(&wc.circuitBreaker.currentState, CircuitOpen)
		atomic.StoreInt64(&wc.circuitBreaker.lastFailTime, time.Now().UnixNano())
		wc.logger.Warn("Circuit breaker opened due to failures", 
			"failure_count", failureCount, 
			"max_failures", wc.circuitBreaker.maxFailures)
	}
}

// executeWithRetry executes a function with exponential backoff retry
func (wc *WeaviateClient) executeWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error
	
	for attempt := 0; attempt <= wc.config.Retry.MaxRetries; attempt++ {
		// Check circuit breaker
		if err := wc.checkCircuitBreaker(); err != nil {
			return fmt.Errorf("circuit breaker check failed: %w", err)
		}

		// Execute operation
		if err := operation(); err != nil {
			lastErr = err
			wc.recordCircuitBreakerFailure()
			
			// Don't retry on final attempt
			if attempt == wc.config.Retry.MaxRetries {
				break
			}

			// Calculate backoff delay
			delay := wc.calculateBackoffDelay(attempt)
			wc.logger.Warn("Operation failed, retrying", 
				"attempt", attempt+1, 
				"max_attempts", wc.config.Retry.MaxRetries+1,
				"delay", delay,
				"error", err)

			// Wait with context cancellation support
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}

		// Success
		wc.recordCircuitBreakerSuccess()
		return nil
	}

	return fmt.Errorf("operation failed after %d attempts: %w", 
		wc.config.Retry.MaxRetries+1, lastErr)
}

// calculateBackoffDelay calculates exponential backoff delay with jitter
func (wc *WeaviateClient) calculateBackoffDelay(attempt int) time.Duration {
	delay := time.Duration(float64(wc.config.Retry.BaseDelay) * 
		math.Pow(wc.config.Retry.BackoffMultiplier, float64(attempt)))
	
	if delay > wc.config.Retry.MaxDelay {
		delay = wc.config.Retry.MaxDelay
	}

	// Add jitter if enabled
	if wc.config.Retry.Jitter {
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.1) // 10% jitter
		delay += jitter
	}

	return delay
}

// initializeEmbeddingFallback initializes the local embedding model fallback
func (wc *WeaviateClient) initializeEmbeddingFallback() error {
	if !wc.embeddingFallback.enabled {
		return nil
	}

	// For now, just mark as available - in a full implementation,
	// you would load the actual model here
	wc.embeddingFallback.mutex.Lock()
	wc.embeddingFallback.available = true
	wc.embeddingFallback.mutex.Unlock()

	wc.logger.Info("Embedding fallback initialized", 
		"model_path", wc.embeddingFallback.modelPath,
		"dimensions", wc.embeddingFallback.dimensions)
	
	return nil
}

// initializeSchemaWithRetry initializes schema with retry logic
func (wc *WeaviateClient) initializeSchemaWithRetry(ctx context.Context) error {
	return wc.executeWithRetry(ctx, func() error {
		return wc.initializeSchema(ctx)
	})
}

// SearchWithRetry performs search with circuit breaker and retry logic
func (wc *WeaviateClient) SearchWithRetry(ctx context.Context, query *SearchQuery) (*SearchResponse, error) {
	// Estimate tokens for rate limiting (rough estimate based on query length)
	estimatedTokens := int64(len(query.Query) / 4) // Rough token estimation
	if estimatedTokens < 10 {
		estimatedTokens = 10 // Minimum token estimate
	}

	// Check rate limits
	if err := wc.checkRateLimit(estimatedTokens); err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}

	// Execute search with retry logic
	var result *SearchResponse
	err := wc.executeWithRetry(ctx, func() error {
		var searchErr error
		result, searchErr = wc.Search(ctx, query)
		return searchErr
	})

	return result, err
}

// AddDocumentWithRetry adds document with retry logic  
func (wc *WeaviateClient) AddDocumentWithRetry(ctx context.Context, doc *TelecomDocument) error {
	// Estimate tokens for rate limiting
	estimatedTokens := int64(len(doc.Content) / 4)
	if estimatedTokens < 10 {
		estimatedTokens = 10
	}

	// Check rate limits
	if err := wc.checkRateLimit(estimatedTokens); err != nil {
		return fmt.Errorf("rate limit check failed: %w", err)
	}

	// Execute with retry logic
	return wc.executeWithRetry(ctx, func() error {
		return wc.AddDocument(ctx, doc)
	})
}

// Close closes the Weaviate client and cleans up resources
func (wc *WeaviateClient) Close() error {
	wc.logger.Info("Closing Weaviate client")
	// The Weaviate Go client doesn't have an explicit close method
	// but we can clean up our resources
	return nil
}