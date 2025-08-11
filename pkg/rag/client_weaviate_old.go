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

// weaviateRAGClient is a Weaviate-based implementation of RAGClient
type weaviateRAGClient struct {
	config     *RAGClientConfig
	httpClient *http.Client
	logger     *slog.Logger
}

// newRAGClientImpl creates a Weaviate-based RAG client
func newRAGClientImpl(config *RAGClientConfig) RAGClient {
	return &weaviateRAGClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: slog.Default().With("component", "weaviate-rag-client"),
	}
}

// ProcessIntent processes an intent using Weaviate RAG
func (c *weaviateRAGClient) ProcessIntent(ctx context.Context, intent string) (string, error) {
	// First, search for relevant context
	searchResults, err := c.Search(ctx, intent, c.config.MaxSearchResults)
	if err != nil {
		return "", fmt.Errorf("failed to search for context: %w", err)
	}

	// Build context from search results
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Based on the following context:\n\n")

	for i, result := range searchResults {
		if result.Confidence >= c.config.MinConfidence {
			contextBuilder.WriteString(fmt.Sprintf("Context %d (confidence: %.2f):\n%s\n\n",
				i+1, result.Confidence, result.Content))
		}
	}

	// If we have LLM configuration, enhance the response
	if c.config.LLMEndpoint != "" {
		enhancedResponse, err := c.callLLM(ctx, intent, contextBuilder.String())
		if err != nil {
			c.logger.Warn("Failed to enhance with LLM, returning basic context",
				slog.String("error", err.Error()))
			return contextBuilder.String(), nil
		}
		return enhancedResponse, nil
	}

	return contextBuilder.String(), nil
}

// Search performs a semantic search using Weaviate
func (c *weaviateRAGClient) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if c.config.WeaviateURL == "" {
		return nil, fmt.Errorf("Weaviate URL not configured")
	}

	// Build GraphQL query for Weaviate
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

	// Make request to Weaviate
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

	// Parse response
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

	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(graphqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL errors: %s", graphqlResp.Errors[0].Message)
	}

	// Convert to SearchResult
	results := make([]SearchResult, 0, len(graphqlResp.Data.Get.Document))
	for _, doc := range graphqlResp.Data.Get.Document {
		results = append(results, SearchResult{
			ID:         doc.Additional.ID,
			Content:    doc.Content,
			Confidence: doc.Additional.Certainty,
			Metadata:   map[string]interface{}{},
		})
	}

	return results, nil
}

// callLLM enhances the response using an LLM
func (c *weaviateRAGClient) callLLM(ctx context.Context, intent, context string) (string, error) {
	prompt := fmt.Sprintf(`Given the following context and user intent, provide a comprehensive response.

Context:
%s

User Intent: %s

Response:`, context, intent)

	reqBody := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant for network function management."},
			{"role": "user", "content": prompt},
		},
		"max_tokens":  c.config.MaxTokens,
		"temperature": c.config.Temperature,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal LLM request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		c.config.LLMEndpoint,
		strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create LLM request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.LLMAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.LLMAPIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute LLM request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(body))
	}

	var llmResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return "", fmt.Errorf("failed to decode LLM response: %w", err)
	}

	if len(llmResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in LLM response")
	}

	return llmResp.Choices[0].Message.Content, nil
}

// Initialize initializes the Weaviate client
func (c *weaviateRAGClient) Initialize(ctx context.Context) error {
	// Test connection to Weaviate
	if c.config.WeaviateURL != "" {
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
			return fmt.Errorf("failed to connect to Weaviate: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Weaviate health check failed with status %d", resp.StatusCode)
		}

		c.logger.Info("Successfully connected to Weaviate",
			slog.String("url", c.config.WeaviateURL))
	}

	return nil
}

// Shutdown gracefully shuts down the client
func (c *weaviateRAGClient) Shutdown(ctx context.Context) error {
	c.logger.Info("Shutting down Weaviate RAG client")
	// Clean up any resources if needed
	return nil
}

// IsHealthy checks if the Weaviate service is healthy
func (c *weaviateRAGClient) IsHealthy() bool {
	if c.config.WeaviateURL == "" {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET",
		c.config.WeaviateURL+"/v1/.well-known/ready", nil)
	if err != nil {
		return false
	}

	if c.config.WeaviateAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.WeaviateAPIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
