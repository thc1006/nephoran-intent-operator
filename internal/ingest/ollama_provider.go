// Package ingest provides intent processing capabilities including Ollama LLM integration.
package ingest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// OllamaProvider implements IntentProvider using Ollama LLM API.
type OllamaProvider struct {
	endpoint string
	model    string
	client   *http.Client
}

// OllamaRequest represents a request to Ollama API.
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaResponse represents a response from Ollama API.
type OllamaResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	CreatedAt string `json:"created_at"`
}

// NewOllamaProvider creates a new Ollama-based intent provider.
func NewOllamaProvider(endpoint, model string) *OllamaProvider {
	if endpoint == "" {
		endpoint = "http://ollama-service.ollama.svc.cluster.local:11434"
	}
	if model == "" {
		model = "llama3.1" // Default model
	}

	return &OllamaProvider{
		endpoint: endpoint,
		model:    model,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 50,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// Name returns the provider name.
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// ParseIntent converts natural language to structured intent using Ollama.
func (p *OllamaProvider) ParseIntent(ctx context.Context, text string) (map[string]interface{}, error) {
	// Build prompt for Ollama
	prompt := p.buildPrompt(text)

	// Call Ollama API
	response, err := p.callOllama(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("ollama API call failed: %w", err)
	}

	// Parse response to extract intent
	intent, err := p.parseResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w", err)
	}

	return intent, nil
}

// buildPrompt creates a structured prompt for Ollama to parse network intents.
func (p *OllamaProvider) buildPrompt(text string) string {
	return fmt.Sprintf(`You are a network intent parser for Kubernetes/5G orchestration. Parse the following natural language command into a structured JSON intent.

User Command: "%s"

Output a JSON object with these fields:
- intent_type: "scaling", "deployment", or "service"
- target: name of the resource (e.g., "nf-sim", "nginx")
- namespace: Kubernetes namespace (default: "default")
- replicas: number of replicas (for scaling/deployment)
- source: always set to "user"
- status: always set to "pending"
- target_resources: array of resource identifiers (e.g., ["deployment/nf-sim"])

Examples:
Input: "scale nf-sim to 5 in ns ran-a"
Output: {"intent_type":"scaling","target":"nf-sim","namespace":"ran-a","replicas":5,"source":"user","status":"pending","target_resources":["deployment/nf-sim"]}

Input: "deploy nginx with 3 replicas in namespace production"
Output: {"intent_type":"deployment","target":"nginx","namespace":"production","replicas":3,"source":"user","status":"pending","target_resources":["deployment/nginx"]}

Now parse the user command and output ONLY the JSON object (no explanation):`, text)
}

// callOllama makes an HTTP request to Ollama API.
func (p *OllamaProvider) callOllama(ctx context.Context, prompt string) (string, error) {
	// Prepare request
	reqBody := OllamaRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/generate", p.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close() // #nosec G307

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse Ollama response
	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to parse ollama response: %w", err)
	}

	return ollamaResp.Response, nil
}

// parseResponse extracts structured intent from Ollama's text response.
func (p *OllamaProvider) parseResponse(response string) (map[string]interface{}, error) {
	// Try to extract JSON from response
	// Ollama might include explanation text, so we need to find the JSON part
	startIdx := -1
	endIdx := -1

	for i := 0; i < len(response); i++ {
		if response[i] == '{' && startIdx == -1 {
			startIdx = i
		}
		if response[i] == '}' {
			endIdx = i + 1
		}
	}

	if startIdx == -1 || endIdx == -1 {
		return nil, fmt.Errorf("no valid JSON found in response: %s", response)
	}

	jsonStr := response[startIdx:endIdx]

	// Parse JSON
	var intent map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &intent); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate required fields
	if err := p.validateIntent(intent); err != nil {
		return nil, fmt.Errorf("invalid intent structure: %w", err)
	}

	return intent, nil
}

// validateIntent ensures the parsed intent has required fields.
func (p *OllamaProvider) validateIntent(intent map[string]interface{}) error {
	// Check required fields
	requiredFields := []string{"intent_type", "target", "namespace", "source", "status"}
	for _, field := range requiredFields {
		if _, ok := intent[field]; !ok {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate intent_type
	intentType, ok := intent["intent_type"].(string)
	if !ok {
		return fmt.Errorf("intent_type must be a string")
	}
	validTypes := map[string]bool{"scaling": true, "deployment": true, "service": true}
	if !validTypes[intentType] {
		return fmt.Errorf("invalid intent_type: %s", intentType)
	}

	// Validate replicas for scaling/deployment intents
	if intentType == "scaling" || intentType == "deployment" {
		if replicas, ok := intent["replicas"]; ok {
			// Convert to int if it's a float64 (from JSON unmarshaling)
			switch v := replicas.(type) {
			case float64:
				intent["replicas"] = int(v)
			case string:
				if i, err := strconv.Atoi(v); err == nil {
					intent["replicas"] = i
				} else {
					return fmt.Errorf("invalid replicas value: %s", v)
				}
			case int:
				// Already correct type
			default:
				return fmt.Errorf("replicas must be a number")
			}

			// Validate range
			replicaCount := intent["replicas"].(int)
			if replicaCount < 0 || replicaCount > 100 {
				return fmt.Errorf("replicas must be between 0 and 100")
			}
		}
	}

	// Ensure target_resources exists
	if _, ok := intent["target_resources"]; !ok {
		// Generate default target_resources
		target, _ := intent["target"].(string)
		intent["target_resources"] = []string{fmt.Sprintf("deployment/%s", target)}
	}

	return nil
}
