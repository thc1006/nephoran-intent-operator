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

// MTLSLLMClient provides mTLS-enabled LLM processor client functionality
type MTLSLLMClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logging.StructuredLogger
}

// LLMRequest represents a request to the LLM processor
type LLMRequest struct {
	Prompt      string                 `json:"prompt"`
	Model       string                 `json:"model,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Temperature float64                `json:"temperature,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	UseRAG      bool                   `json:"use_rag,omitempty"`
	RAGContext  []string               `json:"rag_context,omitempty"`
}

// LLMResponse represents a response from the LLM processor
type LLMResponse struct {
	Content        string                 `json:"content"`
	Model          string                 `json:"model"`
	TokensUsed     int                    `json:"tokens_used"`
	FinishReason   string                 `json:"finish_reason"`
	Metadata       map[string]interface{} `json:"metadata"`
	ProcessingTime time.Duration          `json:"processing_time"`
	RAGUsed        bool                   `json:"rag_used"`
	RAGSources     []string               `json:"rag_sources,omitempty"`
}

// StreamingResponse represents a streaming response chunk
type StreamingResponse struct {
	Content   string                 `json:"content"`
	Delta     string                 `json:"delta"`
	IsLast    bool                   `json:"is_last"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

// ProcessIntent processes an intent using the LLM processor with mTLS
func (c *MTLSLLMClient) ProcessIntent(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt cannot be empty")
	}

	c.logger.Debug("processing intent via mTLS LLM client",
		"prompt_length", len(prompt),
		"base_url", c.baseURL)

	// Create request
	req := &LLMRequest{
		Prompt: prompt,
		UseRAG: true, // Enable RAG by default for intent processing
		Metadata: map[string]interface{}{
			"request_type": "intent_processing",
			"timestamp":    time.Now(),
		},
	}

	// Make request to LLM processor
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

// ProcessIntentStream processes an intent with streaming response
func (c *MTLSLLMClient) ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *shared.StreamingChunk) error {
	if prompt == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	defer close(chunks)

	c.logger.Debug("processing intent stream via mTLS LLM client",
		"prompt_length", len(prompt),
		"base_url", c.baseURL)

	// Create request
	req := &LLMRequest{
		Prompt: prompt,
		UseRAG: true,
		Metadata: map[string]interface{}{
			"request_type": "streaming_intent_processing",
			"timestamp":    time.Now(),
		},
	}

	// Create HTTP request
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

	// Make request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to make streaming request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("streaming request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Process streaming response
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

		// Convert to shared.StreamingChunk
		chunk := &shared.StreamingChunk{
			Content:   streamResp.Content,
			IsLast:    streamResp.IsLast,
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

// GetSupportedModels returns the list of supported models
func (c *MTLSLLMClient) GetSupportedModels() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/v1/models", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

// getModelCapabilitiesInternal returns capabilities for a specific model (internal method)
func (c *MTLSLLMClient) getModelCapabilitiesInternal(modelName string) (*shared.ModelCapabilities, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/api/v1/models/%s/capabilities", c.baseURL, modelName)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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

// ValidateModel validates if a model is supported
func (c *MTLSLLMClient) ValidateModel(modelName string) error {
	supportedModels := c.GetSupportedModels()

	for _, model := range supportedModels {
		if model == modelName {
			return nil
		}
	}

	return fmt.Errorf("model %s is not supported", modelName)
}

// EstimateTokens estimates the token count for given text
func (c *MTLSLLMClient) EstimateTokens(text string) int {
	// Simple token estimation (4 characters per token on average)
	// This is a rough approximation and should ideally call the service
	return len(text) / 4
}

// GetMaxTokens returns the maximum tokens for a model
func (c *MTLSLLMClient) GetMaxTokens(modelName string) int {
	capabilities, err := c.getModelCapabilitiesInternal(modelName)
	if err != nil {
		c.logger.Warn("failed to get model capabilities, using default max tokens",
			"model", modelName,
			"error", err)
		return 4096 // Default fallback
	}

	return capabilities.MaxTokens
}

// makeRequest makes an HTTP request to the LLM processor
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

// Close closes the LLM client and cleans up resources
func (c *MTLSLLMClient) Close() error {
	c.logger.Debug("closing mTLS LLM client")

	// Close idle connections in HTTP client
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	return nil
}

// GetHealth returns the health status of the LLM service
func (c *MTLSLLMClient) GetHealth() (*HealthStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/health", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return &HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("failed to connect: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("health check failed with status %d", resp.StatusCode),
		}, nil
	}

	var health HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return &HealthStatus{
			Status:  "unknown",
			Message: fmt.Sprintf("failed to decode health response: %v", err),
		}, nil
	}

	return &health, nil
}

// HealthStatus represents the health status of a service
type HealthStatus struct {
	Status    string                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// ProcessRequest implements shared.ClientInterface
func (c *MTLSLLMClient) ProcessRequest(ctx context.Context, request *shared.LLMRequest) (*shared.LLMResponse, error) {
	// Convert shared.LLMRequest to internal LLMRequest
	internal := &LLMRequest{
		Prompt:      request.Messages[0].Content, // Use first message content as prompt
		Model:       request.Model,
		MaxTokens:   request.MaxTokens,
		Temperature: float64(request.Temperature),
		Metadata:    request.Metadata,
	}

	// Make request
	response, err := c.makeRequest(ctx, "/api/v1/process", internal)
	if err != nil {
		return nil, err
	}

	// Convert response to shared format
	return &shared.LLMResponse{
		Content: response.Content,
		Model:   response.Model,
		Usage: shared.TokenUsage{
			TotalTokens: response.TokensUsed,
		},
		Created: time.Now(),
	}, nil
}

// ProcessStreamingRequest implements shared.ClientInterface
func (c *MTLSLLMClient) ProcessStreamingRequest(ctx context.Context, request *shared.LLMRequest) (<-chan *shared.StreamingChunk, error) {
	chunks := make(chan *shared.StreamingChunk, 10)
	
	go func() {
		prompt := request.Messages[0].Content
		err := c.ProcessIntentStream(ctx, prompt, chunks)
		if err != nil {
			c.logger.Error("streaming request failed", "error", err)
			chunks <- &shared.StreamingChunk{
				Error: &shared.LLMError{
					Message: err.Error(),
				},
			}
		}
	}()
	
	return chunks, nil
}

// HealthCheck implements shared.ClientInterface
func (c *MTLSLLMClient) HealthCheck(ctx context.Context) error {
	health, err := c.GetHealth()
	if err != nil {
		return err
	}
	
	if health.Status != "healthy" {
		return fmt.Errorf("service unhealthy: %s", health.Message)
	}
	
	return nil
}

// GetStatus implements shared.ClientInterface
func (c *MTLSLLMClient) GetStatus() shared.ClientStatus {
	health, err := c.GetHealth()
	if err != nil {
		return shared.ClientStatusUnavailable
	}
	
	switch health.Status {
	case "healthy":
		return shared.ClientStatusHealthy
	case "unhealthy":
		return shared.ClientStatusUnhealthy
	default:
		return shared.ClientStatusUnavailable
	}
}

// GetModelCapabilities implements shared.ClientInterface (wrapper)
func (c *MTLSLLMClient) GetModelCapabilities() shared.ModelCapabilities {
	caps, err := c.getModelCapabilitiesInternal("default")
	if err != nil {
		return shared.ModelCapabilities{
			MaxTokens: 4096,
			SupportsStreaming: true,
		}
	}
	return *caps
}

// GetEndpoint implements shared.ClientInterface
func (c *MTLSLLMClient) GetEndpoint() string {
	return c.baseURL
}
