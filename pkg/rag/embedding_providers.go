//go:build !disable_rag && !test

package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// OpenAIProvider implements the OpenAI embedding provider
type OpenAIProvider struct {
	config     ProviderConfig
	httpClient *http.Client
	logger     *slog.Logger
	mutex      sync.RWMutex
}

// NewBasicOpenAIProvider creates a new basic OpenAI provider
func NewBasicOpenAIProvider(config ProviderConfig, httpClient *http.Client) *OpenAIProvider {
	return &OpenAIProvider{
		config:     config,
		httpClient: httpClient,
		logger:     slog.Default().With("component", "openai-provider"),
	}
}

// GenerateEmbeddings implements EmbeddingProvider
func (p *OpenAIProvider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, TokenUsage, error) {
	requestBody := OpenAIEmbeddingRequest{
		Input: texts,
		Model: p.config.ModelName,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.APIEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("User-Agent", "Nephoran-Intent-Operator/1.0")

	// Execute request with retries
	var resp *http.Response
	var lastErr error
	maxRetries := 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, lastErr = p.httpClient.Do(req)
		if lastErr == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt < maxRetries {
			p.logger.Warn("OpenAI API request failed, retrying",
				"attempt", attempt+1,
				"error", lastErr,
				"status", func() int {
					if resp != nil {
						return resp.StatusCode
					}
					return 0
				}(),
			)

			// Exponential backoff
			backoffDelay := time.Duration(attempt+1) * 2 * time.Second
			select {
			case <-ctx.Done():
				return nil, TokenUsage{}, ctx.Err()
			case <-time.After(backoffDelay):
			}
		}
	}

	if lastErr != nil {
		return nil, TokenUsage{}, fmt.Errorf("OpenAI API request failed after %d attempts: %w", maxRetries+1, lastErr)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, TokenUsage{}, fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	var apiResponse OpenAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	// Extract embeddings
	embeddings := make([][]float32, len(apiResponse.Data))
	for i, data := range apiResponse.Data {
		embeddings[i] = data.Embedding
	}

	// Calculate estimated cost
	estimatedCost := float64(apiResponse.Usage.TotalTokens) * p.config.CostPerToken / 1000

	usage := TokenUsage{
		PromptTokens:  apiResponse.Usage.PromptTokens,
		TotalTokens:   apiResponse.Usage.TotalTokens,
		EstimatedCost: estimatedCost,
	}

	return embeddings, usage, nil
}

// GetConfig implements EmbeddingProvider
func (p *OpenAIProvider) GetConfig() ProviderConfig {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.config
}

// HealthCheck implements EmbeddingProvider
func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
	// Simple health check with a minimal embedding request
	testTexts := []string{"health check"}
	_, _, err := p.GenerateEmbeddings(ctx, testTexts)

	p.mutex.Lock()
	p.config.Healthy = (err == nil)
	p.config.LastCheck = time.Now()
	p.mutex.Unlock()

	return err
}

// GetCostEstimate implements EmbeddingProvider
func (p *OpenAIProvider) GetCostEstimate(tokenCount int) float64 {
	return float64(tokenCount) * p.config.CostPerToken / 1000
}

// GetName implements EmbeddingProvider
func (p *OpenAIProvider) GetName() string {
	return p.config.Name
}

func (p *OpenAIProvider) IsHealthy() bool {
	// Simple health check - could be enhanced with actual API call
	return p.httpClient != nil && p.config.APIKey != ""
}

func (p *OpenAIProvider) GetLatency() time.Duration {
	// Return estimated latency for OpenAI API
	// This could be measured dynamically in a real implementation
	return 200 * time.Millisecond
}

// AzureOpenAIProvider implements the Azure OpenAI embedding provider
type AzureOpenAIProvider struct {
	config     ProviderConfig
	httpClient *http.Client
	logger     *slog.Logger
	mutex      sync.RWMutex
}

// NewBasicAzureOpenAIProvider creates a new basic Azure OpenAI provider
func NewBasicAzureOpenAIProvider(config ProviderConfig, httpClient *http.Client) *AzureOpenAIProvider {
	return &AzureOpenAIProvider{
		config:     config,
		httpClient: httpClient,
		logger:     slog.Default().With("component", "azure-openai-provider"),
	}
}

// GenerateEmbeddings implements EmbeddingProvider for Azure OpenAI
func (p *AzureOpenAIProvider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, TokenUsage, error) {
	// Azure OpenAI uses a slightly different API format
	requestBody := map[string]interface{}{
		"input": texts,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to marshal Azure request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.APIEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to create Azure request: %w", err)
	}

	// Set Azure-specific headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.config.APIKey)
	req.Header.Set("User-Agent", "Nephoran-Intent-Operator/1.0")

	// Execute request with retries
	var resp *http.Response
	var lastErr error
	maxRetries := 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, lastErr = p.httpClient.Do(req)
		if lastErr == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if resp != nil {
			resp.Body.Close()
		}

		if attempt < maxRetries {
			p.logger.Warn("Azure OpenAI API request failed, retrying",
				"attempt", attempt+1,
				"error", lastErr,
				"status", func() int {
					if resp != nil {
						return resp.StatusCode
					}
					return 0
				}(),
			)

			// Exponential backoff
			backoffDelay := time.Duration(attempt+1) * 2 * time.Second
			select {
			case <-ctx.Done():
				return nil, TokenUsage{}, ctx.Err()
			case <-time.After(backoffDelay):
			}
		}
	}

	if lastErr != nil {
		return nil, TokenUsage{}, fmt.Errorf("Azure OpenAI API request failed after %d attempts: %w", maxRetries+1, lastErr)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, TokenUsage{}, fmt.Errorf("Azure OpenAI API returned status %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	var apiResponse OpenAIEmbeddingResponse // Azure uses similar response format
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to decode Azure response: %w", err)
	}

	// Extract embeddings
	embeddings := make([][]float32, len(apiResponse.Data))
	for i, data := range apiResponse.Data {
		embeddings[i] = data.Embedding
	}

	// Calculate estimated cost
	estimatedCost := float64(apiResponse.Usage.TotalTokens) * p.config.CostPerToken / 1000

	usage := TokenUsage{
		PromptTokens:  apiResponse.Usage.PromptTokens,
		TotalTokens:   apiResponse.Usage.TotalTokens,
		EstimatedCost: estimatedCost,
	}

	return embeddings, usage, nil
}

// GetConfig implements EmbeddingProvider
func (p *AzureOpenAIProvider) GetConfig() ProviderConfig {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.config
}

// HealthCheck implements EmbeddingProvider
func (p *AzureOpenAIProvider) HealthCheck(ctx context.Context) error {
	testTexts := []string{"health check"}
	_, _, err := p.GenerateEmbeddings(ctx, testTexts)

	p.mutex.Lock()
	p.config.Healthy = (err == nil)
	p.config.LastCheck = time.Now()
	p.mutex.Unlock()

	return err
}

// GetCostEstimate implements EmbeddingProvider
func (p *AzureOpenAIProvider) GetCostEstimate(tokenCount int) float64 {
	return float64(tokenCount) * p.config.CostPerToken / 1000
}

// GetName implements EmbeddingProvider
func (p *AzureOpenAIProvider) GetName() string {
	return p.config.Name
}

func (p *AzureOpenAIProvider) IsHealthy() bool {
	// Simple health check - could be enhanced with actual API call
	return p.httpClient != nil && p.config.APIKey != ""
}

func (p *AzureOpenAIProvider) GetLatency() time.Duration {
	// Return estimated latency for Azure OpenAI API
	return 250 * time.Millisecond
}

// LocalProvider implements a local embedding provider (placeholder for future implementation)
type LocalProvider struct {
	config ProviderConfig
	logger *slog.Logger
	mutex  sync.RWMutex
}

// NewLocalProvider creates a new local provider
func NewLocalProvider(config ProviderConfig) *LocalProvider {
	return &LocalProvider{
		config: config,
		logger: slog.Default().With("component", "local-provider"),
	}
}

// GenerateEmbeddings implements EmbeddingProvider for local models
func (p *LocalProvider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, TokenUsage, error) {
	// This is a placeholder for local embedding generation
	// In a real implementation, this would interface with local models like:
	// - sentence-transformers via Python subprocess
	// - ONNX Runtime with embedding models
	// - TensorFlow Lite models
	// - Ollama for local LLM embeddings

	p.logger.Warn("Local provider called but not implemented - would generate embeddings locally")

	// For now, return an error indicating this is not yet implemented
	return nil, TokenUsage{}, fmt.Errorf("local embedding provider not implemented yet - requires integration with local models")
}

// GetConfig implements EmbeddingProvider
func (p *LocalProvider) GetConfig() ProviderConfig {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.config
}

// HealthCheck implements EmbeddingProvider
func (p *LocalProvider) HealthCheck(ctx context.Context) error {
	p.mutex.Lock()
	p.config.Healthy = true // Always healthy for local provider (when implemented)
	p.config.LastCheck = time.Now()
	p.mutex.Unlock()

	// For now, return healthy status
	return nil
}

// GetCostEstimate implements EmbeddingProvider (local is free)
func (p *LocalProvider) GetCostEstimate(tokenCount int) float64 {
	return 0.0 // Local processing is free
}

// GetName implements EmbeddingProvider
func (p *LocalProvider) GetName() string {
	return p.config.Name
}

func (p *LocalProvider) IsHealthy() bool {
	// Local provider is always healthy if properly configured
	return true
}

func (p *LocalProvider) GetLatency() time.Duration {
	// Local provider should have very low latency
	return 10 * time.Millisecond
}

// HuggingFaceProvider implements Hugging Face Inference API provider
type HuggingFaceProvider struct {
	config     ProviderConfig
	httpClient *http.Client
	logger     *slog.Logger
	mutex      sync.RWMutex
}

// NewBasicHuggingFaceProvider creates a new basic Hugging Face provider
func NewBasicHuggingFaceProvider(config ProviderConfig, httpClient *http.Client) *HuggingFaceProvider {
	return &HuggingFaceProvider{
		config:     config,
		httpClient: httpClient,
		logger:     slog.Default().With("component", "huggingface-provider"),
	}
}

// GenerateEmbeddings implements EmbeddingProvider for Hugging Face
func (p *HuggingFaceProvider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, TokenUsage, error) {
	// Hugging Face Inference API format
	requestBody := map[string]interface{}{
		"inputs": texts,
		"options": map[string]interface{}{
			"wait_for_model": true,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to marshal HuggingFace request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.APIEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to create HuggingFace request: %w", err)
	}

	// Set Hugging Face headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("User-Agent", "Nephoran-Intent-Operator/1.0")

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("HuggingFace API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, TokenUsage{}, fmt.Errorf("HuggingFace API returned status %d", resp.StatusCode)
	}

	// HuggingFace returns embeddings directly as arrays
	var embeddings [][]float32
	if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to decode HuggingFace response: %w", err)
	}

	// Estimate token usage (HuggingFace doesn't provide this)
	totalTokens := 0
	for _, text := range texts {
		totalTokens += len(text) / 4 // Rough estimation
	}

	usage := TokenUsage{
		PromptTokens:  totalTokens,
		TotalTokens:   totalTokens,
		EstimatedCost: p.GetCostEstimate(totalTokens),
	}

	return embeddings, usage, nil
}

// GetConfig implements EmbeddingProvider
func (p *HuggingFaceProvider) GetConfig() ProviderConfig {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.config
}

// HealthCheck implements EmbeddingProvider
func (p *HuggingFaceProvider) HealthCheck(ctx context.Context) error {
	testTexts := []string{"health check"}
	_, _, err := p.GenerateEmbeddings(ctx, testTexts)

	p.mutex.Lock()
	p.config.Healthy = (err == nil)
	p.config.LastCheck = time.Now()
	p.mutex.Unlock()

	return err
}

// GetCostEstimate implements EmbeddingProvider
func (p *HuggingFaceProvider) GetCostEstimate(tokenCount int) float64 {
	return float64(tokenCount) * p.config.CostPerToken / 1000
}

// GetName implements EmbeddingProvider
func (p *HuggingFaceProvider) GetName() string {
	return p.config.Name
}

func (p *HuggingFaceProvider) IsHealthy() bool {
	// Simple health check
	return p.httpClient != nil && p.config.APIKey != ""
}

func (p *HuggingFaceProvider) GetLatency() time.Duration {
	// Return estimated latency for Hugging Face API
	return 500 * time.Millisecond
}

// CohereProvider implements Cohere embedding provider
type CohereProvider struct {
	config     ProviderConfig
	httpClient *http.Client
	logger     *slog.Logger
	mutex      sync.RWMutex
}

// NewCohereProvider creates a new Cohere provider
func NewCohereProvider(config ProviderConfig, httpClient *http.Client) *CohereProvider {
	return &CohereProvider{
		config:     config,
		httpClient: httpClient,
		logger:     slog.Default().With("component", "cohere-provider"),
	}
}

// GenerateEmbeddings implements EmbeddingProvider for Cohere
func (p *CohereProvider) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, TokenUsage, error) {
	requestBody := map[string]interface{}{
		"texts":      texts,
		"model":      p.config.ModelName,
		"input_type": "search_document", // For retrieval use cases
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to marshal Cohere request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.config.APIEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to create Cohere request: %w", err)
	}

	// Set Cohere headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.config.APIKey)
	req.Header.Set("User-Agent", "Nephoran-Intent-Operator/1.0")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, TokenUsage{}, fmt.Errorf("Cohere API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, TokenUsage{}, fmt.Errorf("Cohere API returned status %d", resp.StatusCode)
	}

	var apiResponse struct {
		Embeddings [][]float32 `json:"embeddings"`
		Meta       struct {
			BilledUnits struct {
				InputTokens int `json:"input_tokens"`
			} `json:"billed_units"`
		} `json:"meta"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, TokenUsage{}, fmt.Errorf("failed to decode Cohere response: %w", err)
	}

	usage := TokenUsage{
		PromptTokens:  apiResponse.Meta.BilledUnits.InputTokens,
		TotalTokens:   apiResponse.Meta.BilledUnits.InputTokens,
		EstimatedCost: p.GetCostEstimate(apiResponse.Meta.BilledUnits.InputTokens),
	}

	return apiResponse.Embeddings, usage, nil
}

// GetConfig implements EmbeddingProvider
func (p *CohereProvider) GetConfig() ProviderConfig {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.config
}

// HealthCheck implements EmbeddingProvider
func (p *CohereProvider) HealthCheck(ctx context.Context) error {
	testTexts := []string{"health check"}
	_, _, err := p.GenerateEmbeddings(ctx, testTexts)

	p.mutex.Lock()
	p.config.Healthy = (err == nil)
	p.config.LastCheck = time.Now()
	p.mutex.Unlock()

	return err
}

// GetCostEstimate implements EmbeddingProvider
func (p *CohereProvider) GetCostEstimate(tokenCount int) float64 {
	return float64(tokenCount) * p.config.CostPerToken / 1000
}

// GetName implements EmbeddingProvider
func (p *CohereProvider) GetName() string {
	return p.config.Name
}

func (p *CohereProvider) IsHealthy() bool {
	// Simple health check
	return p.httpClient != nil && p.config.APIKey != ""
}

func (p *CohereProvider) GetLatency() time.Duration {
	// Return estimated latency for Cohere API
	return 300 * time.Millisecond
}
