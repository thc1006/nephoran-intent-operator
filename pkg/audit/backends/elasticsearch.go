package backends

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/audit/types"
)

var (
	elasticsearchRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "elasticsearch_audit_requests_total",
		Help: "Total number of requests to Elasticsearch",
	}, []string{"instance", "method", "status"})

	elasticsearchRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "elasticsearch_audit_request_duration_seconds",
		Help:    "Duration of Elasticsearch requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"instance", "method"})
)

// ElasticsearchBackend implements audit logging to Elasticsearch/OpenSearch.
type ElasticsearchBackend struct {
	config      BackendConfig
	httpClient  *http.Client
	logger      logr.Logger
	baseURL     string
	indexPrefix string
	username    string
	password    string
	apiKey      string

	// Metrics.
	eventsWritten int64
	eventsFailed  int64

	// Index management.
	indexTemplate string
	aliasName     string
}

// ElasticsearchConfig represents Elasticsearch-specific configuration.
type ElasticsearchConfig struct {
	URLs            []string `json:"urls" yaml:"urls"`
	IndexPrefix     string   `json:"index_prefix" yaml:"index_prefix"`
	IndexPattern    string   `json:"index_pattern" yaml:"index_pattern"`
	AliasName       string   `json:"alias_name" yaml:"alias_name"`
	Shards          int      `json:"shards" yaml:"shards"`
	Replicas        int      `json:"replicas" yaml:"replicas"`
	RefreshInterval string   `json:"refresh_interval" yaml:"refresh_interval"`

	// Index lifecycle management.
	ILMPolicy       string        `json:"ilm_policy" yaml:"ilm_policy"`
	RetentionPeriod time.Duration `json:"retention_period" yaml:"retention_period"`
	RolloverSize    string        `json:"rollover_size" yaml:"rollover_size"`
	RolloverAge     time.Duration `json:"rollover_age" yaml:"rollover_age"`

	// Performance tuning.
	BulkSize       int           `json:"bulk_size" yaml:"bulk_size"`
	FlushInterval  time.Duration `json:"flush_interval" yaml:"flush_interval"`
	MaxRetries     int           `json:"max_retries" yaml:"max_retries"`
	RequestTimeout time.Duration `json:"request_timeout" yaml:"request_timeout"`

	// Security.
	Username  string `json:"username" yaml:"username"`
	Password  string `json:"password" yaml:"password"`
	APIKey    string `json:"api_key" yaml:"api_key"`
	CloudID   string `json:"cloud_id" yaml:"cloud_id"`
	CertPath  string `json:"cert_path" yaml:"cert_path"`
	KeyPath   string `json:"key_path" yaml:"key_path"`
	CAPath    string `json:"ca_path" yaml:"ca_path"`
	VerifyTLS bool   `json:"verify_tls" yaml:"verify_tls"`
}

// NewElasticsearchBackend creates a new Elasticsearch backend.
func NewElasticsearchBackend(config BackendConfig) (*ElasticsearchBackend, error) {
	esConfig, err := parseElasticsearchConfig(config.Settings)
	if err != nil {
		return nil, fmt.Errorf("invalid Elasticsearch configuration: %w", err)
	}

	backend := &ElasticsearchBackend{
		config:      config,
		logger:      log.Log.WithName("elasticsearch-backend").WithValues("instance", config.Name),
		indexPrefix: esConfig.IndexPrefix,
		username:    esConfig.Username,
		password:    esConfig.Password,
		apiKey:      esConfig.APIKey,
		aliasName:   esConfig.AliasName,
	}

	if backend.indexPrefix == "" {
		backend.indexPrefix = "nephoran-audit"
	}

	if backend.aliasName == "" {
		backend.aliasName = fmt.Sprintf("%s-alias", backend.indexPrefix)
	}

	// Set base URL.
	if len(esConfig.URLs) > 0 {
		backend.baseURL = esConfig.URLs[0] // Use first URL for now
	} else {
		return nil, fmt.Errorf("no Elasticsearch URLs provided")
	}

	// Create HTTP client.
	tlsConfig := &tls.Config{
		InsecureSkipVerify: !esConfig.VerifyTLS,
	}

	backend.httpClient = &http.Client{
		Timeout: esConfig.RequestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return backend, nil
}

// Initialize sets up the Elasticsearch backend.
func (eb *ElasticsearchBackend) Initialize(config BackendConfig) error {
	eb.logger.Info("Initializing Elasticsearch backend")

	// Create index template.
	if err := eb.createIndexTemplate(); err != nil {
		return fmt.Errorf("failed to create index template: %w", err)
	}

	// Create initial index.
	if err := eb.createIndex(); err != nil {
		return fmt.Errorf("failed to create initial index: %w", err)
	}

	// Create alias.
	if err := eb.createAlias(); err != nil {
		return fmt.Errorf("failed to create alias: %w", err)
	}

	eb.logger.Info("Elasticsearch backend initialized successfully")
	return nil
}

// Type returns the backend type.
func (eb *ElasticsearchBackend) Type() string {
	return string(BackendTypeElasticsearch)
}

// WriteEvent writes a single audit event to Elasticsearch.
func (eb *ElasticsearchBackend) WriteEvent(ctx context.Context, event *types.AuditEvent) error {
	return eb.WriteEvents(ctx, []*types.AuditEvent{event})
}

// WriteEvents writes multiple audit events to Elasticsearch using bulk API.
func (eb *ElasticsearchBackend) WriteEvents(ctx context.Context, events []*types.AuditEvent) error {
	if len(events) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		elasticsearchRequestDuration.WithLabelValues(eb.config.Name, "bulk").Observe(duration.Seconds())
	}()

	// Build bulk request.
	var bulkBody bytes.Buffer
	for _, event := range events {
		// Filter event if necessary.
		if !eb.config.Filter.ShouldProcessEvent(event) {
			continue
		}

		filteredEvent := eb.config.Filter.ApplyFieldFilters(event)

		// Create index action.
		indexAction := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": eb.getIndexName(filteredEvent.Timestamp),
				"_id":    filteredEvent.ID,
			},
		}

		actionBytes, err := json.Marshal(indexAction)
		if err != nil {
			eb.logger.Error(err, "Failed to marshal index action")
			atomic.AddInt64(&eb.eventsFailed, 1)
			continue
		}

		eventBytes, err := filteredEvent.ToJSON()
		if err != nil {
			eb.logger.Error(err, "Failed to marshal event")
			atomic.AddInt64(&eb.eventsFailed, 1)
			continue
		}

		bulkBody.Write(actionBytes)
		bulkBody.WriteByte('\n')
		bulkBody.Write(eventBytes)
		bulkBody.WriteByte('\n')
	}

	if bulkBody.Len() == 0 {
		return nil // No events to write after filtering
	}

	// Send bulk request.
	url := fmt.Sprintf("%s/_bulk", eb.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, &bulkBody)
	if err != nil {
		return fmt.Errorf("failed to create bulk request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")
	eb.setAuthHeaders(req)

	resp, err := eb.httpClient.Do(req)
	if err != nil {
		elasticsearchRequestsTotal.WithLabelValues(eb.config.Name, "bulk", "error").Inc()
		atomic.AddInt64(&eb.eventsFailed, int64(len(events)))
		return fmt.Errorf("bulk request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response.
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		elasticsearchRequestsTotal.WithLabelValues(eb.config.Name, "bulk", fmt.Sprintf("%d", resp.StatusCode)).Inc()
		atomic.AddInt64(&eb.eventsFailed, int64(len(events)))
		return fmt.Errorf("bulk request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse bulk response to check for individual failures.
	var bulkResp BulkResponse
	if err := json.NewDecoder(resp.Body).Decode(&bulkResp); err != nil {
		eb.logger.Error(err, "Failed to parse bulk response")
	} else if bulkResp.Errors {
		eb.processBulkErrors(bulkResp.Items)
	}

	elasticsearchRequestsTotal.WithLabelValues(eb.config.Name, "bulk", "success").Inc()
	atomic.AddInt64(&eb.eventsWritten, int64(len(events)))

	return nil
}

// Query searches for audit events in Elasticsearch.
func (eb *ElasticsearchBackend) Query(ctx context.Context, query *QueryRequest) (*QueryResponse, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		elasticsearchRequestDuration.WithLabelValues(eb.config.Name, "search").Observe(duration.Seconds())
	}()

	searchQuery := eb.buildSearchQuery(query)

	queryBytes, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	url := fmt.Sprintf("%s/%s/_search", eb.baseURL, eb.aliasName)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(queryBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	eb.setAuthHeaders(req)

	resp, err := eb.httpClient.Do(req)
	if err != nil {
		elasticsearchRequestsTotal.WithLabelValues(eb.config.Name, "search", "error").Inc()
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		elasticsearchRequestsTotal.WithLabelValues(eb.config.Name, "search", fmt.Sprintf("%d", resp.StatusCode)).Inc()
		return nil, fmt.Errorf("search request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	elasticsearchRequestsTotal.WithLabelValues(eb.config.Name, "search", "success").Inc()

	return eb.parseSearchResponse(&searchResp), nil
}

// Health checks the Elasticsearch backend health.
func (eb *ElasticsearchBackend) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/_cluster/health", eb.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	eb.setAuthHeaders(req)

	resp, err := eb.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	var health ClusterHealth
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return fmt.Errorf("failed to parse health response: %w", err)
	}

	if health.Status == "red" {
		return fmt.Errorf("cluster status is red")
	}

	return nil
}

// Close closes the Elasticsearch backend.
func (eb *ElasticsearchBackend) Close() error {
	eb.logger.Info("Closing Elasticsearch backend")
	return nil
}

// Helper methods.

func (eb *ElasticsearchBackend) setAuthHeaders(req *http.Request) {
	if eb.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", eb.apiKey))
	} else if eb.username != "" && eb.password != "" {
		req.SetBasicAuth(eb.username, eb.password)
	}
}

func (eb *ElasticsearchBackend) getIndexName(timestamp time.Time) string {
	return fmt.Sprintf("%s-%s", eb.indexPrefix, timestamp.Format("2006.01.02"))
}

func (eb *ElasticsearchBackend) createIndexTemplate() error {
	template := map[string]interface{}{
		"index_patterns": []string{fmt.Sprintf("%s-*", eb.indexPrefix)},
		"template": map[string]interface{}{
			"settings": map[string]interface{}{
				"number_of_shards":   1,
				"number_of_replicas": 0,
				"refresh_interval":   "5s",
			},
			"mappings": eb.getIndexMapping(),
		},
	}

	templateBytes, err := json.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal index template: %w", err)
	}

	url := fmt.Sprintf("%s/_index_template/%s-template", eb.baseURL, eb.indexPrefix)
	req, err := http.NewRequestWithContext(context.Background(), "PUT", url, bytes.NewReader(templateBytes))
	if err != nil {
		return fmt.Errorf("failed to create template request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	eb.setAuthHeaders(req)

	resp, err := eb.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("template request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("template request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (eb *ElasticsearchBackend) createIndex() error {
	indexName := eb.getIndexName(time.Now())
	url := fmt.Sprintf("%s/%s", eb.baseURL, indexName)

	req, err := http.NewRequestWithContext(context.Background(), "HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create index check request: %w", err)
	}

	eb.setAuthHeaders(req)

	resp, err := eb.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("index check failed: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil // Index already exists
	}

	// Create the index.
	req, err = http.NewRequestWithContext(context.Background(), "PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create index request: %w", err)
	}

	eb.setAuthHeaders(req)

	resp, err = eb.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("index creation failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("index creation failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (eb *ElasticsearchBackend) createAlias() error {
	aliasAction := map[string]interface{}{
		"actions": []map[string]interface{}{
			{
				"add": map[string]interface{}{
					"index": fmt.Sprintf("%s-*", eb.indexPrefix),
					"alias": eb.aliasName,
				},
			},
		},
	}

	aliasBytes, err := json.Marshal(aliasAction)
	if err != nil {
		return fmt.Errorf("failed to marshal alias action: %w", err)
	}

	url := fmt.Sprintf("%s/_aliases", eb.baseURL)
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(aliasBytes))
	if err != nil {
		return fmt.Errorf("failed to create alias request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	eb.setAuthHeaders(req)

	resp, err := eb.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("alias request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("alias request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (eb *ElasticsearchBackend) getIndexMapping() map[string]interface{} {
	return map[string]interface{}{
		"properties": map[string]interface{}{
			"timestamp": map[string]interface{}{
				"type": "date",
			},
			"event_type": map[string]interface{}{
				"type": "keyword",
			},
			"severity": map[string]interface{}{
				"type": "integer",
			},
			"component": map[string]interface{}{
				"type": "keyword",
			},
			"action": map[string]interface{}{
				"type": "keyword",
			},
			"result": map[string]interface{}{
				"type": "keyword",
			},
			"user_context": map[string]interface{}{
				"properties": map[string]interface{}{
					"user_id": map[string]interface{}{
						"type": "keyword",
					},
					"username": map[string]interface{}{
						"type": "keyword",
					},
					"role": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			"network_context": map[string]interface{}{
				"properties": map[string]interface{}{
					"source_ip": map[string]interface{}{
						"type": "ip",
					},
					"destination_ip": map[string]interface{}{
						"type": "ip",
					},
				},
			},
			"description": map[string]interface{}{
				"type": "text",
			},
			"message": map[string]interface{}{
				"type": "text",
			},
		},
	}
}

func (eb *ElasticsearchBackend) buildSearchQuery(query *QueryRequest) map[string]interface{} {
	esQuery := map[string]interface{}{
		"size": query.Limit,
		"from": query.Offset,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{},
				"filter": []interface{}{
					map[string]interface{}{
						"range": map[string]interface{}{
							"timestamp": map[string]interface{}{
								"gte": query.StartTime.Format(time.RFC3339),
								"lte": query.EndTime.Format(time.RFC3339),
							},
						},
					},
				},
			},
		},
	}

	// Add query string if provided.
	if query.Query != "" {
		must := esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"].([]interface{})
		must = append(must, map[string]interface{}{
			"query_string": map[string]interface{}{
				"query": query.Query,
			},
		})
		esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = must
	}

	// Add filters.
	if len(query.Filters) > 0 {
		filter := esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"].([]interface{})
		for field, value := range query.Filters {
			filter = append(filter, map[string]interface{}{
				"term": map[string]interface{}{
					field: value,
				},
			})
		}
		esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"] = filter
	}

	// Add sorting.
	if query.SortBy != "" {
		order := "desc"
		if query.SortOrder == "asc" {
			order = "asc"
		}
		esQuery["sort"] = []map[string]interface{}{
			{
				query.SortBy: map[string]string{
					"order": order,
				},
			},
		}
	}

	return esQuery
}

func (eb *ElasticsearchBackend) parseSearchResponse(resp *SearchResponse) *QueryResponse {
	events := make([]*types.AuditEvent, len(resp.Hits.Hits))
	for i, hit := range resp.Hits.Hits {
		var event types.AuditEvent
		if err := json.Unmarshal(hit.Source, &event); err != nil {
			eb.logger.Error(err, "Failed to unmarshal search hit")
			continue
		}
		events[i] = &event
	}

	return &QueryResponse{
		Events:     events,
		TotalCount: resp.Hits.Total.Value,
		HasMore:    int64(len(events)) < resp.Hits.Total.Value,
	}
}

func (eb *ElasticsearchBackend) processBulkErrors(items []BulkItem) {
	for _, item := range items {
		if item.Index.Error != nil {
			eb.logger.Error(fmt.Errorf("%s", item.Index.Error.Reason), "Bulk index error",
				"type", item.Index.Error.Type,
				"id", item.Index.ID)
			atomic.AddInt64(&eb.eventsFailed, 1)
		}
	}
}

func parseElasticsearchConfig(settings map[string]interface{}) (*ElasticsearchConfig, error) {
	configBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	var config ElasticsearchConfig
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Set defaults.
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 30 * time.Second
	}
	if config.BulkSize == 0 {
		config.BulkSize = 100
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	return &config, nil
}

// Response types for Elasticsearch API.

// BulkResponse represents a bulkresponse.
type BulkResponse struct {
	Took   int        `json:"took"`
	Errors bool       `json:"errors"`
	Items  []BulkItem `json:"items"`
}

// BulkItem represents a bulkitem.
type BulkItem struct {
	Index struct {
		ID     string     `json:"_id"`
		Result string     `json:"result"`
		Status int        `json:"status"`
		Error  *BulkError `json:"error,omitempty"`
	} `json:"index"`
}

// BulkError represents a bulkerror.
type BulkError struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// SearchResponse represents a searchresponse.
type SearchResponse struct {
	Took int `json:"took"`
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []struct {
			Source json.RawMessage `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// ClusterHealth represents a clusterhealth.
type ClusterHealth struct {
	ClusterName string `json:"cluster_name"`
	Status      string `json:"status"`
}
