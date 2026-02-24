// Package rag provides a client for the Nephoran RAG (Retrieval-Augmented Generation) service.
// This client allows Go applications to query the RAG API for intent processing and Q&A.
package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/security"
)

// Client represents a RAG service client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	logger     *slog.Logger
}

// Config holds configuration for the RAG client.
type Config struct {
	BaseURL string
	Timeout time.Duration
	APIKey  string
}

// NewClient creates a new RAG service client with the given configuration.
// The BaseURL is validated against SSRF attacks. In-cluster private IPs are
// allowed because the RAG service typically runs as a Kubernetes service.
func NewClient(cfg Config) (*Client, error) {
	logger := slog.Default().With("component", "rag-client", "baseURL", cfg.BaseURL)
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	if err := security.ValidateInClusterEndpointURL(cfg.BaseURL); err != nil {
		logger.Error("Failed to validate RAG endpoint URL",
			"error", err,
			"baseURL", cfg.BaseURL,
		)
		return nil, fmt.Errorf("rag client: %w", err)
	}

	return &Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 50,
				IdleConnTimeout:     90 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},
		apiKey: cfg.APIKey,
		logger: logger,
	}, nil
}

// HTTPQueryRequest represents a query to the RAG HTTP service.
type HTTPQueryRequest struct {
	Question string            `json:"question"`
	Context  map[string]string `json:"context,omitempty"`
}

// HTTPQueryResponse represents the response from the RAG HTTP service.
type HTTPQueryResponse struct {
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
func (c *Client) Query(ctx context.Context, question string) (*HTTPQueryResponse, error) {
	return c.QueryWithContext(ctx, question, nil)
}

// QueryWithContext sends a question with additional context to the RAG service.
func (c *Client) QueryWithContext(ctx context.Context, question string, contextMap map[string]string) (*HTTPQueryResponse, error) {
	reqBody := HTTPQueryRequest{
		Question: question,
		Context:  contextMap,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		c.logger.Error("Failed to marshal RAG query request",
			"error", err,
			"question", question,
		)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/query",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		c.logger.Error("Failed to create RAG query HTTP request",
			"error", err,
			"url", c.baseURL+"/query",
			"question", question,
		)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to execute RAG query HTTP request",
			"error", err,
			"url", c.baseURL+"/query",
			"question", question,
		)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read RAG query response body",
			"error", err,
			"statusCode", resp.StatusCode,
		)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("RAG query failed with non-200 status",
			"statusCode", resp.StatusCode,
			"responseBody", string(body),
			"question", question,
			"url", c.baseURL+"/query",
		)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result HTTPQueryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		c.logger.Error("Failed to decode RAG query response",
			"error", err,
			"responseBody", string(body),
		)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.logger.Info("Successfully executed RAG query",
		"question", question,
		"sourcesCount", len(result.Sources),
	)
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
		c.logger.Error("Failed to marshal RAG intent request",
			"error", err,
			"intent", intent,
			"user", user,
		)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/intent",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		c.logger.Error("Failed to create RAG intent HTTP request",
			"error", err,
			"url", c.baseURL+"/intent",
			"intent", intent,
		)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to execute RAG intent HTTP request",
			"error", err,
			"url", c.baseURL+"/intent",
			"intent", intent,
			"user", user,
		)
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error("Failed to read RAG intent response body",
			"error", err,
			"statusCode", resp.StatusCode,
		)
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.logger.Error("RAG intent processing failed with non-200 status",
			"statusCode", resp.StatusCode,
			"responseBody", string(body),
			"intent", intent,
			"user", user,
			"url", c.baseURL+"/intent",
		)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result IntentResponse
	if err := json.Unmarshal(body, &result); err != nil {
		c.logger.Error("Failed to decode RAG intent response",
			"error", err,
			"responseBody", string(body),
		)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Error != "" {
		c.logger.Error("RAG intent processing returned error",
			"intentError", result.Error,
			"intent", intent,
			"user", user,
			"type", result.Type,
		)
		return &result, fmt.Errorf("intent processing error: %s", result.Error)
	}

	c.logger.Info("Successfully processed intent via RAG",
		"intent", intent,
		"user", user,
		"type", result.Type,
		"name", result.Name,
		"namespace", result.Namespace,
	)
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
		c.logger.Error("Failed to create RAG health check request",
			"error", err,
			"url", c.baseURL+"/health",
		)
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("RAG health check HTTP request failed",
			"error", err,
			"url", c.baseURL+"/health",
		)
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Warn("RAG service unhealthy",
			"statusCode", resp.StatusCode,
			"responseBody", string(body),
			"url", c.baseURL+"/health",
		)
		return fmt.Errorf("unhealthy (status %d): %s", resp.StatusCode, string(body))
	}

	c.logger.Debug("RAG health check passed")
	return nil
}
