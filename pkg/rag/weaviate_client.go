//go:build rag




package rag



import (

	"context"

	"encoding/json"

	"fmt"

	"io"

	"log/slog"

	"net/http"

	"strings"

	"time"

)



// weaviateRAGClient is a Weaviate-based implementation of RAGClient.

type weaviateRAGClient struct {

	config     *RAGClientConfig

	httpClient *http.Client

	logger     *slog.Logger

}



// newRAGClientImpl creates a Weaviate-based RAG client.

func newRAGClientImpl(config *RAGClientConfig) RAGClient {

	// Handle nil config gracefully.

	if config == nil {

		config = &RAGClientConfig{

			Enabled:          false,

			MaxSearchResults: 5,

			MinConfidence:    0.7,

		}

	}



	return &weaviateRAGClient{

		config: config,

		httpClient: &http.Client{

			Timeout: 30 * time.Second,

		},

		logger: slog.Default().With("component", "weaviate-rag-client"),

	}

}



// Retrieve performs a semantic search using Weaviate and returns documents.

func (c *weaviateRAGClient) Retrieve(ctx context.Context, query string) ([]Doc, error) {

	if c.config.WeaviateURL == "" {

		return nil, fmt.Errorf("Weaviate URL not configured")

	}



	// Use configured max results or default to 5.

	limit := c.config.MaxSearchResults

	if limit <= 0 {

		limit = 5

	}



	// Build GraphQL query for Weaviate.

	graphqlQuery := fmt.Sprintf(`{

		Get {

			Document(

				nearText: {

					concepts: ["%s"]

				}

				limit: %d

			) {

				content

				_additional {

					id

					certainty

				}

			}

		}

	}`, query, limit)



	// Make request to Weaviate.

	reqBody := map[string]string{"query": graphqlQuery}

	jsonBody, err := json.Marshal(reqBody)

	if err != nil {

		return nil, fmt.Errorf("failed to marshal query: %w", err)

	}



	req, err := http.NewRequestWithContext(ctx, "POST",

		c.config.WeaviateURL+"/v1/graphql",

		strings.NewReader(string(jsonBody)))

	if err != nil {

		return nil, fmt.Errorf("failed to create request: %w", err)

	}



	req.Header.Set("Content-Type", "application/json")

	if c.config.WeaviateAPIKey != "" {

		req.Header.Set("Authorization", "Bearer "+c.config.WeaviateAPIKey)

	}



	resp, err := c.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("failed to execute request: %w", err)

	}

	defer resp.Body.Close()



	if resp.StatusCode != http.StatusOK {

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("Weaviate returned status %d: %s", resp.StatusCode, string(body))

	}



	// Parse response.

	var graphqlResp struct {

		Data struct {

			Get struct {

				Document []struct {

					Content    string `json:"content"`

					Additional struct {

						ID        string  `json:"id"`

						Certainty float64 `json:"certainty"`

					} `json:"_additional"`

				} `json:"Document"`

			} `json:"Get"`

		} `json:"data"`

		Errors []struct {

			Message string `json:"message"`

		} `json:"errors"`

	}



	body, err := io.ReadAll(resp.Body)

	if err != nil {

		return nil, fmt.Errorf("failed to read response: %w", err)

	}



	if err := json.Unmarshal(body, &graphqlResp); err != nil {

		return nil, fmt.Errorf("failed to decode response: %w", err)

	}



	if len(graphqlResp.Errors) > 0 {

		return nil, fmt.Errorf("GraphQL errors: %s", graphqlResp.Errors[0].Message)

	}



	// Convert to Doc structs and apply confidence filter.

	results := make([]Doc, 0, len(graphqlResp.Data.Get.Document))

	minConfidence := c.config.MinConfidence

	if minConfidence <= 0 {

		minConfidence = 0.0 // Accept all results if no minimum configured

	}



	for _, doc := range graphqlResp.Data.Get.Document {

		if doc.Additional.Certainty >= minConfidence {

			results = append(results, Doc{

				ID:         doc.Additional.ID,

				Content:    doc.Content,

				Confidence: doc.Additional.Certainty,

				Metadata:   map[string]interface{}{},

			})

		}

	}



	c.logger.Debug("Retrieved documents from Weaviate",

		slog.Int("total_found", len(graphqlResp.Data.Get.Document)),

		slog.Int("filtered_results", len(results)),

		slog.Float64("min_confidence", minConfidence))



	return results, nil

}



// Initialize initializes the Weaviate RAG client and validates connectivity.

func (c *weaviateRAGClient) Initialize(ctx context.Context) error {

	if c.config.WeaviateURL == "" {

		return fmt.Errorf("Weaviate URL not configured")

	}



	// Test connectivity by making a simple health check request.

	req, err := http.NewRequestWithContext(ctx, "GET",

		c.config.WeaviateURL+"/v1/.well-known/ready", nil)

	if err != nil {

		return fmt.Errorf("failed to create health check request: %w", err)

	}



	if c.config.WeaviateAPIKey != "" {

		req.Header.Set("Authorization", "Bearer "+c.config.WeaviateAPIKey)

	}



	resp, err := c.httpClient.Do(req)

	if err != nil {

		c.logger.Warn("Weaviate health check failed, but continuing initialization",

			slog.String("error", err.Error()),

			slog.String("url", c.config.WeaviateURL))

		return nil // Don't fail hard on connectivity issues during init

	}

	defer resp.Body.Close()



	if resp.StatusCode != http.StatusOK {

		c.logger.Warn("Weaviate health check returned non-200 status",

			slog.Int("status_code", resp.StatusCode),

			slog.String("url", c.config.WeaviateURL))

		return nil // Don't fail hard on health check failures

	}



	c.logger.Info("Weaviate RAG client initialized successfully",

		slog.String("url", c.config.WeaviateURL))



	return nil

}



// Shutdown gracefully shuts down the Weaviate RAG client.

func (c *weaviateRAGClient) Shutdown(ctx context.Context) error {

	// Close HTTP client idle connections.

	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {

		transport.CloseIdleConnections()

	}



	c.logger.Info("Weaviate RAG client shut down gracefully")

	return nil

}

