package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

// WeaviateClient provides a production-ready client for Weaviate vector database
type WeaviateClient struct {
	client       *weaviate.Client
	config       *WeaviateConfig
	logger       *slog.Logger
	healthStatus *HealthStatus
	mutex        sync.RWMutex
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
	
	// Performance tuning
	ConnectionPool struct {
		MaxIdleConns        int           `json:"max_idle_conns"`
		MaxConnsPerHost     int           `json:"max_conns_per_host"`
		IdleConnTimeout     time.Duration `json:"idle_conn_timeout"`
		DisableCompression  bool          `json:"disable_compression"`
	} `json:"connection_pool"`
}

// HealthStatus represents the health status of the Weaviate cluster
type HealthStatus struct {
	IsHealthy     bool              `json:"is_healthy"`
	LastCheck     time.Time         `json:"last_check"`
	Version       string            `json:"version"`
	Nodes         []NodeStatus      `json:"nodes"`
	Statistics    ClusterStatistics `json:"statistics"`
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
	Timestamp      time.Time         `json:"timestamp"`
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
	Document        *TelecomDocument `json:"document"`
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

	// Configure authentication
	var authConfig auth.Config
	if config.APIKey != "" {
		authConfig = auth.ApiKey{Value: config.APIKey}
	}

	// Create Weaviate client configuration
	clientConfig := weaviate.Config{
		Host:   config.Host,
		Scheme: config.Scheme,
		AuthConfig: authConfig,
		Headers: config.Headers,
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
		healthStatus: &HealthStatus{
			LastCheck: time.Now(),
		},
	}

	// Initialize schema if auto-schema is enabled
	if config.AutoSchema {
		if err := wc.initializeSchema(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to initialize schema: %w", err)
		}
	}

	// Start health checking in background
	go wc.startHealthChecking()

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
	doc := &TelecomDocument{}
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
	if val, ok := item["subcategory"].(string); ok {
		doc.Subcategory = val
	}
	if val, ok := item["version"].(string); ok {
		doc.Version = val
	}
	if val, ok := item["workingGroup"].(string); ok {
		doc.WorkingGroup = val
	}
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
			doc.Timestamp = t
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
func (wc *WeaviateClient) buildWhereFilter(filters map[string]interface{}) *graphql.WhereArgumentBuilder {
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
		"timestamp":       doc.Timestamp.Format(time.RFC3339),
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
func (wc *WeaviateClient) GetHealthStatus() *HealthStatus {
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

// Close closes the Weaviate client and cleans up resources
func (wc *WeaviateClient) Close() error {
	wc.logger.Info("Closing Weaviate client")
	// The Weaviate Go client doesn't have an explicit close method
	// but we can clean up our resources
	return nil
}