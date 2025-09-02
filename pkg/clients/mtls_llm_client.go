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

// MTLSLLMClient provides mTLS-enabled LLM processor client functionality.

type MTLSLLMClient struct {
	baseURL string

	httpClient *http.Client

	logger *logging.StructuredLogger
}

// LLMRequest represents a request to the LLM processor.

type LLMRequest struct {
	Prompt string `json:"prompt"`

	Model string `json:"model,omitempty"`

	MaxTokens int `json:"max_tokens,omitempty"`

	Temperature float64 `json:"temperature,omitempty"`

	Metadata json.RawMessage `json:"metadata,omitempty"`

	UseRAG bool `json:"use_rag,omitempty"`

	RAGContext []string `json:"rag_context,omitempty"`
}

// LLMResponse represents a response from the LLM processor.

type LLMResponse struct {
	Content string `json:"content"`

	Model string `json:"model"`

	TokensUsed int `json:"tokens_used"`

	FinishReason string `json:"finish_reason"`

	Metadata json.RawMessage `json:"metadata"`

	ProcessingTime time.Duration `json:"processing_time"`

	RAGUsed bool `json:"rag_used"`

	RAGSources []string `json:"rag_sources,omitempty"`
}

// StreamingResponse represents a streaming response chunk.

type StreamingResponse struct {
	Content string `json:"content"`

	Delta string `json:"delta"`

	IsLast bool `json:"is_last"`

	Metadata json.RawMessage `json:"metadata"`

	Timestamp time.Time `json:"timestamp"`
}

// ProcessIntent processes an intent using the LLM processor with mTLS.

func (c *MTLSLLMClient) ProcessIntent(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt cannot be empty")
	}

	c.logger.Debug("processing intent via mTLS LLM client",

		"prompt_length", len(prompt),

		"base_url", c.baseURL)

	// Create request.

	req := &LLMRequest{
		Prompt: prompt,

		UseRAG: true, // Enable RAG by default for intent processing

		Metadata: json.RawMessage(`{}`),
	}

	// Make request to LLM processor.

	response, err := c.makeRequest(ctx, "/api/v1/process", req)
	if err != nil {
		return "", fmt.Errorf("failed to process intent: %w", err)
	}

	c.logger.Debug("intent processed successfully",

		"response_length", len(response.Content),

		"tokens_used", response.TokensUsed,

		"processing_time", response.ProcessingTime,

		"rag_used", response.RAGUsed)

	return response.Content, nil
}

// ProcessIntentStream processes an intent with streaming response.

func (c *MTLSLLMClient) ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *shared.StreamingChunk) error {
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	defer close(chunks)

	c.logger.Debug("processing intent stream via mTLS LLM client",

		"prompt_length", len(prompt),

		"base_url", c.baseURL)

	// Create request.

	req := &LLMRequest{
		Prompt: prompt,

		UseRAG: true,

		Metadata: json.RawMessage(`{}`),
	}

	// Create HTTP request.

	url := fmt.Sprintf("%s/api/v1/stream", c.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpReq.Header.Set("Accept", "text/event-stream")

	// Make request.

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to make streaming request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("streaming request failed with status %d: %s", resp.StatusCode, string(body))

	}

	// Process streaming response.

	decoder := json.NewDecoder(resp.Body)

	for {

		select {

		case <-ctx.Done():

			return ctx.Err()

		default:

		}

		var streamResp StreamingResponse

		if err := decoder.Decode(&streamResp); err != nil {

			if err == io.EOF {
				break
			}

			return fmt.Errorf("failed to decode streaming response: %w", err)

		}

		// Convert to shared.StreamingChunk.

		chunk := &shared.StreamingChunk{
			Content: streamResp.Content,

			IsLast: streamResp.IsLast,

			Metadata: streamResp.Metadata,

			Timestamp: streamResp.Timestamp,
		}

		select {

		case chunks <- chunk:

		case <-ctx.Done():

			return ctx.Err()

		}

		if streamResp.IsLast {
			break
		}

	}

	c.logger.Debug("intent stream processing completed")

	return nil
}

// GetSupportedModels returns the list of supported models.

func (c *MTLSLLMClient) GetSupportedModels() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	url := fmt.Sprintf("%s/api/v1/models", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {

		c.logger.Error("failed to create models request", "error", err)

		return []string{}

	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {

		c.logger.Error("failed to get supported models", "error", err)

		return []string{}

	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		c.logger.Error("models request failed", "status", resp.StatusCode)

		return []string{}

	}

	var models []string

	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {

		c.logger.Error("failed to decode models response", "error", err)

		return []string{}

	}

	return models
}

// GetModelCapabilities returns default model capabilities.
func (c *MTLSLLMClient) GetModelCapabilities() shared.ModelCapabilities {
	// Return default capabilities - in production this should be retrieved from service
	return shared.ModelCapabilities{
		SupportsStreaming:    true,
		SupportsSystemPrompt: true,
		SupportsChatFormat:   true,
		SupportsChat:         true,
		SupportsFunction:     false,
		MaxTokens:            4096,
		CostPerToken:         0.001,
		SupportedMimeTypes:   []string{"text/plain", "application/json"},
		ModelVersion:         "1.0.0",
		Features:             make(map[string]interface{}),
	}
}

// GetModelCapabilitiesForModel returns capabilities for a specific model.
func (c *MTLSLLMClient) GetModelCapabilitiesForModel(modelName string) (*shared.ModelCapabilities, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	url := fmt.Sprintf("%s/api/v1/models/%s/capabilities", c.baseURL, modelName)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create capabilities request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get model capabilities: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("capabilities request failed with status %d: %s", resp.StatusCode, string(body))

	}

	var capabilities shared.ModelCapabilities

	if err := json.NewDecoder(resp.Body).Decode(&capabilities); err != nil {
		return nil, fmt.Errorf("failed to decode capabilities response: %w", err)
	}

	return &capabilities, nil
}

// ValidateModel validates if a model is supported.

func (c *MTLSLLMClient) ValidateModel(modelName string) error {
	supportedModels := c.GetSupportedModels()

	for _, model := range supportedModels {
		if model == modelName {
			return nil
		}
	}

	return fmt.Errorf("model %s is not supported", modelName)
}

// EstimateTokens estimates the token count for given text.

func (c *MTLSLLMClient) EstimateTokens(text string) int {
	// Simple token estimation (4 characters per token on average).

	// This is a rough approximation and should ideally call the service.

	return len(text) / 4
}

// GetMaxTokens returns the maximum tokens for a model.

func (c *MTLSLLMClient) GetMaxTokens(modelName string) int {
	capabilities, err := c.GetModelCapabilitiesForModel(modelName)
	if err != nil {

		c.logger.Warn("failed to get model capabilities, using default max tokens",

			"model", modelName,

			"error", err)

		return 4096 // Default fallback

	}

	return capabilities.MaxTokens
}

// makeRequest makes an HTTP request to the LLM processor.

func (c *MTLSLLMClient) makeRequest(ctx context.Context, endpoint string, request *LLMRequest) (*LLMResponse, error) {
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

	c.logger.Debug("making LLM request",

		"url", url,

		"method", httpReq.Method,

		"content_length", len(reqBody))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}

	defer resp.Body.Close()

	c.logger.Debug("received LLM response",

		"status_code", resp.StatusCode,

		"content_length", resp.ContentLength)

	if resp.StatusCode != http.StatusOK {

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))

	}

	var response LLMResponse

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// Close closes the LLM client and cleans up resources.

func (c *MTLSLLMClient) Close() error {
	c.logger.Debug("closing mTLS LLM client")

	// Close idle connections in HTTP client.

	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	return nil
}

// HealthCheck performs a health check on the LLM service.
func (c *MTLSLLMClient) HealthCheck(ctx context.Context) error {
	healthStatus, err := c.GetHealth()
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if healthStatus.Status != "healthy" && healthStatus.Status != "ok" {
		return fmt.Errorf("service unhealthy: %s - %s", healthStatus.Status, healthStatus.Message)
	}

	return nil
}

// GetStatus returns the current client status.
func (c *MTLSLLMClient) GetStatus() shared.ClientStatus {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.HealthCheck(ctx); err != nil {
		c.logger.Debug("health check failed", "error", err)
		return shared.ClientStatusUnhealthy
	}

	return shared.ClientStatusHealthy
}

// GetEndpoint returns the base URL of the LLM service.
func (c *MTLSLLMClient) GetEndpoint() string {
	return c.baseURL
}

// ProcessRequest processes an LLM request and returns a response.
func (c *MTLSLLMClient) ProcessRequest(ctx context.Context, request *shared.LLMRequest) (*shared.LLMResponse, error) {
	if request == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	// Convert shared.LLMRequest to internal LLMRequest format
	internalReq := &LLMRequest{
		Prompt:      extractPromptFromMessages(request.Messages),
		Model:       request.Model,
		MaxTokens:   request.MaxTokens,
		Temperature: float64(request.Temperature),
		Metadata:    request.Metadata,
	}

	// Use existing makeRequest method
	response, err := c.makeRequest(ctx, "/api/v1/process", internalReq)
	if err != nil {
		return nil, fmt.Errorf("failed to process request: %w", err)
	}

	// Convert internal LLMResponse to shared.LLMResponse
	return &shared.LLMResponse{
		ID:      fmt.Sprintf("llm-req-%d", time.Now().UnixNano()),
		Content: response.Content,
		Model:   response.Model,
		Usage: shared.TokenUsage{
			PromptTokens:     c.EstimateTokens(internalReq.Prompt),
			CompletionTokens: response.TokensUsed,
			TotalTokens:      c.EstimateTokens(internalReq.Prompt) + response.TokensUsed,
		},
		Created: time.Now(),
	}, nil
}

// ProcessStreamingRequest processes a streaming LLM request.
func (c *MTLSLLMClient) ProcessStreamingRequest(ctx context.Context, request *shared.LLMRequest) (<-chan *shared.StreamingChunk, error) {
	if request == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	prompt := extractPromptFromMessages(request.Messages)
	chunks := make(chan *shared.StreamingChunk, 10)

	go func() {
		err := c.ProcessIntentStream(ctx, prompt, chunks)
		if err != nil {
			// Send error chunk
			select {
			case chunks <- &shared.StreamingChunk{
				Error: &shared.LLMError{
					Code:    "streaming_error",
					Message: err.Error(),
					Type:    "internal_error",
				},
				IsLast: true,
			}:
			case <-ctx.Done():
			}
		}
	}()

	return chunks, nil
}

// extractPromptFromMessages converts chat messages to a single prompt string.
func extractPromptFromMessages(messages []shared.ChatMessage) string {
	if len(messages) == 0 {
		return ""
	}

	var prompt string
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			prompt += fmt.Sprintf("System: %s\n", msg.Content)
		case "user":
			prompt += fmt.Sprintf("User: %s\n", msg.Content)
		case "assistant":
			prompt += fmt.Sprintf("Assistant: %s\n", msg.Content)
		default:
			prompt += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
		}
	}

	return prompt
}

// GetHealth returns the health status of the LLM service.

func (c *MTLSLLMClient) GetHealth() (*HealthStatus, error) {
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

// HealthStatus represents the health status of a service.

type HealthStatus struct {
	Status string `json:"status"`

	Message string `json:"message,omitempty"`

	Timestamp time.Time `json:"timestamp"`

	Details json.RawMessage `json:"details,omitempty"`
}

