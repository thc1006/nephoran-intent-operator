package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// MTLSRAGClient provides mTLS-enabled RAG service client functionality.

type MTLSRAGClient struct {
	baseURL string

	httpClient *http.Client

	logger *logging.StructuredLogger
}

// RAGSearchRequest represents a request to the RAG service.

type RAGSearchRequest struct {
	Query string `json:"query"`

	Limit int `json:"limit,omitempty"`

	Filters json.RawMessage `json:"filters,omitempty"`

	HybridSearch bool `json:"hybrid_search,omitempty"`

	HybridAlpha float32 `json:"hybrid_alpha,omitempty"`

	UseReranker bool `json:"use_reranker,omitempty"`

	MinConfidence float32 `json:"min_confidence,omitempty"`

	ExpandQuery bool `json:"expand_query,omitempty"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// RAGSearchResponse represents a response from the RAG service.

type RAGSearchResponse struct {
	Results []*shared.SearchResult `json:"results"`

	Took int64 `json:"took"`

	Total int64 `json:"total"`

	Query string `json:"query"`

	Filters json.RawMessage `json:"filters,omitempty"`
}

// RAGDocumentRequest represents a document ingestion request.

type RAGDocumentRequest struct {
	Documents []*shared.TelecomDocument `json:"documents"`

	Options *IngestionOptions `json:"options,omitempty"`
}

// IngestionOptions represents options for document ingestion.

type IngestionOptions struct {
	ChunkSize int `json:"chunk_size,omitempty"`

	ChunkOverlap int `json:"chunk_overlap,omitempty"`

	BatchSize int `json:"batch_size,omitempty"`

	Async bool `json:"async,omitempty"`

	UpdateExisting bool `json:"update_existing,omitempty"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// RAGDocumentResponse represents a response from document ingestion.

type RAGDocumentResponse struct {
	DocumentsProcessed int `json:"documents_processed"`

	DocumentsUpdated int `json:"documents_updated"`

	DocumentsFailed int `json:"documents_failed"`

	ProcessingTime time.Duration `json:"processing_time"`

	TaskID string `json:"task_id,omitempty"`

	Errors []string `json:"errors,omitempty"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// RAGStatsResponse represents statistics from the RAG service.

type RAGStatsResponse struct {
	TotalDocuments int64 `json:"total_documents"`

	TotalChunks int64 `json:"total_chunks"`

	IndexSize int64 `json:"index_size"`

	LastIndexed time.Time `json:"last_indexed"`

	Categories map[string]int `json:"categories"`

	Technologies map[string]int `json:"technologies"`

	DocumentTypes map[string]int `json:"document_types"`

	PerformanceStats json.RawMessage `json:"performance_stats"`
}

// SearchDocuments searches for documents using the RAG service with mTLS.

func (c *MTLSRAGClient) SearchDocuments(ctx context.Context, query *shared.SearchQuery) (*shared.SearchResponse, error) {
	if query == nil || query.Query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	c.logger.Debug("searching documents via mTLS RAG client",

		"query", query.Query,

		"limit", query.Limit,

		"hybrid_search", query.HybridSearch,

		"base_url", c.baseURL)

	// Convert shared.SearchQuery to internal request format.

	req := &RAGSearchRequest{
		Query: query.Query,

		Limit: query.Limit,

		Filters: query.Filters,

		HybridSearch: query.HybridSearch,

		HybridAlpha: query.HybridAlpha,

		UseReranker: query.UseReranker,

		MinConfidence: query.MinConfidence,

		ExpandQuery: query.ExpandQuery,

		Metadata: json.RawMessage(`{}`),
	}

	// Set defaults.

	if req.Limit <= 0 {
		req.Limit = 10
	}

	if req.HybridAlpha <= 0 {
		req.HybridAlpha = 0.5
	}

	if req.MinConfidence <= 0 {
		req.MinConfidence = 0.1
	}

	// Make request to RAG service.

	response, err := c.makeSearchRequest(ctx, "/api/v1/search", req)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	c.logger.Debug("document search completed successfully",

		"results_count", len(response.Results),

		"total_results", response.Total,

		"processing_time", response.Took)

	// Convert to shared.SearchResponse.

	return &shared.SearchResponse{
		Results: response.Results,

		Took: response.Took,

		Total: response.Total,
	}, nil
}

// IngestDocuments ingests documents into the RAG service.

func (c *MTLSRAGClient) IngestDocuments(ctx context.Context, documents []*shared.TelecomDocument, options *IngestionOptions) (*RAGDocumentResponse, error) {
	if len(documents) == 0 {
		return nil, fmt.Errorf("no documents provided for ingestion")
	}

	c.logger.Debug("ingesting documents via mTLS RAG client",

		"document_count", len(documents),

		"base_url", c.baseURL)

	// Create request.

	req := &RAGDocumentRequest{
		Documents: documents,

		Options: options,
	}

	if req.Options == nil {
		req.Options = &IngestionOptions{
			ChunkSize: 1000,

			ChunkOverlap: 100,

			BatchSize: 10,

			Async: false,
		}
	}

	// Make request to RAG service.

	response, err := c.makeDocumentRequest(ctx, "/api/v1/documents", req)
	if err != nil {
		return nil, fmt.Errorf("failed to ingest documents: %w", err)
	}

	c.logger.Info("document ingestion completed",

		"documents_processed", response.DocumentsProcessed,

		"documents_updated", response.DocumentsUpdated,

		"documents_failed", response.DocumentsFailed,

		"processing_time", response.ProcessingTime)

	return response, nil
}

// GetDocumentByID retrieves a specific document by ID.

func (c *MTLSRAGClient) GetDocumentByID(ctx context.Context, documentID string) (*shared.TelecomDocument, error) {
	if documentID == "" {
		return nil, fmt.Errorf("document ID cannot be empty")
	}

	url := fmt.Sprintf("%s/api/v1/documents/%s", c.baseURL, documentID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")

	c.logger.Debug("retrieving document by ID",

		"document_id", documentID,

		"url", url)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("document not found: %s", documentID)
	}

	if resp.StatusCode != http.StatusOK {

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))

	}

	var document shared.TelecomDocument

	if err := json.NewDecoder(resp.Body).Decode(&document); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &document, nil
}

// DeleteDocument deletes a document by ID.

func (c *MTLSRAGClient) DeleteDocument(ctx context.Context, documentID string) error {
	if documentID == "" {
		return fmt.Errorf("document ID cannot be empty")
	}

	url := fmt.Sprintf("%s/api/v1/documents/%s", c.baseURL, documentID)

	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", url, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	c.logger.Debug("deleting document",

		"document_id", documentID,

		"url", url)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("document not found: %s", documentID)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {

		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("delete request failed with status %d: %s", resp.StatusCode, string(body))

	}

	c.logger.Info("document deleted successfully", "document_id", documentID)

	return nil
}

// GetStats retrieves statistics from the RAG service.

func (c *MTLSRAGClient) GetStats(ctx context.Context) (*RAGStatsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/stats", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("stats request failed with status %d: %s", resp.StatusCode, string(body))

	}

	var stats RAGStatsResponse

	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode stats response: %w", err)
	}

	return &stats, nil
}

// GetHealth returns the health status of the RAG service.

func (c *MTLSRAGClient) GetHealth() (*HealthStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	url := fmt.Sprintf("%s/health", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return &HealthStatus{
			Status: "unhealthy",

			Message: fmt.Sprintf("failed to connect: %v", err),
		}, nil
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &HealthStatus{
			Status: "unhealthy",

			Message: fmt.Sprintf("health check failed with status %d", resp.StatusCode),
		}, nil
	}

	var health HealthStatus

	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return &HealthStatus{
			Status: "unknown",

			Message: fmt.Sprintf("failed to decode health response: %v", err),
		}, nil
	}

	return &health, nil
}

// makeSearchRequest makes an HTTP search request to the RAG service.

func (c *MTLSRAGClient) makeSearchRequest(ctx context.Context, endpoint string, request *RAGSearchRequest) (*RAGSearchResponse, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpReq.Header.Set("Accept", "application/json")

	c.logger.Debug("making RAG search request",

		"url", url,

		"method", httpReq.Method,

		"content_length", len(reqBody))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}

	defer resp.Body.Close()

	c.logger.Debug("received RAG search response",

		"status_code", resp.StatusCode,

		"content_length", resp.ContentLength)

	if resp.StatusCode != http.StatusOK {

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))

	}

	var response RAGSearchResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// makeDocumentRequest makes an HTTP document request to the RAG service.

func (c *MTLSRAGClient) makeDocumentRequest(ctx context.Context, endpoint string, request *RAGDocumentRequest) (*RAGDocumentResponse, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpReq.Header.Set("Accept", "application/json")

	c.logger.Debug("making RAG document request",

		"url", url,

		"method", httpReq.Method,

		"content_length", len(reqBody))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}

	defer resp.Body.Close()

	c.logger.Debug("received RAG document response",

		"status_code", resp.StatusCode,

		"content_length", resp.ContentLength)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))

	}

	var response RAGDocumentResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// Close closes the RAG client and cleans up resources.

func (c *MTLSRAGClient) Close() error {
	c.logger.Debug("closing mTLS RAG client")

	// Close idle connections in HTTP client.

	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	return nil
}

