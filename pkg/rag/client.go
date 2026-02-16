// Package rag provides a client for the Nephoran RAG (Retrieval-Augmented Generation) service.
// This client allows Go applications to query the RAG API for intent processing and Q&A.
package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a RAG service client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// Config holds configuration for the RAG client.
type Config struct {
	BaseURL string
	Timeout time.Duration
	APIKey  string
}

// NewClient creates a new RAG service client with the given configuration.
func NewClient(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		apiKey: cfg.APIKey,
	}
}

// QueryRequest represents a query to the RAG service.
type QueryRequest struct {
	Question string            `json:"question"`
	Context  map[string]string `json:"context,omitempty"`
}

// QueryResponse represents the response from the RAG service.
type QueryResponse struct {
	Answer   string                 `json:"answer"`
	Sources  []string               `json:"sources,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// IntentRequest represents an intent processing request.
type IntentRequest struct {
	Intent string            `json:"intent"`
	User   string            `json:"user,omitempty"`
	Tags   map[string]string `json:"tags,omitempty"`
}

// IntentResponse represents the structured intent output.
type IntentResponse struct {
	Type           string                 `json:"type"`
	Name           string                 `json:"name,omitempty"`
	Namespace      string                 `json:"namespace,omitempty"`
	Spec           map[string]interface{} `json:"spec,omitempty"`
	OriginalIntent string                 `json:"original_intent"`
	Error          string                 `json:"error,omitempty"`
}

// Query sends a question to the RAG service and returns the answer.
func (c *Client) Query(ctx context.Context, question string) (*QueryResponse, error) {
	return c.QueryWithContext(ctx, question, nil)
}

// QueryWithContext sends a question with additional context to the RAG service.
func (c *Client) QueryWithContext(ctx context.Context, question string, contextMap map[string]string) (*QueryResponse, error) {
	reqBody := QueryRequest{
		Question: question,
		Context:  contextMap,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/query",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result QueryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ProcessIntent sends a natural language intent to the RAG service for processing.
// The service will return a structured JSON object representing the intent.
func (c *Client) ProcessIntent(ctx context.Context, intent string) (*IntentResponse, error) {
	return c.ProcessIntentWithTags(ctx, intent, "", nil)
}

// ProcessIntentWithTags sends an intent with user and tags for tracking.
func (c *Client) ProcessIntentWithTags(ctx context.Context, intent, user string, tags map[string]string) (*IntentResponse, error) {
	reqBody := IntentRequest{
		Intent: intent,
		User:   user,
		Tags:   tags,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/intent",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result IntentResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != "" {
		return &result, fmt.Errorf("intent processing error: %s", result.Error)
	}

	return &result, nil
}

// Health checks if the RAG service is healthy.
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/health",
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unhealthy (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}
