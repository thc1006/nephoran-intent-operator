package ingest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOllamaProvider(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		model    string
		wantEndpoint string
		wantModel    string
	}{
		{
			name:     "with_custom_values",
			endpoint: "http://custom:11434",
			model:    "llama2",
			wantEndpoint: "http://custom:11434",
			wantModel:    "llama2",
		},
		{
			name:     "with_defaults",
			endpoint: "",
			model:    "",
			wantEndpoint: "http://ollama-service.ollama.svc.cluster.local:11434",
			wantModel:    "llama3.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewOllamaProvider(tt.endpoint, tt.model)
			assert.Equal(t, tt.wantEndpoint, provider.endpoint)
			assert.Equal(t, tt.wantModel, provider.model)
			assert.NotNil(t, provider.client)
		})
	}
}

func TestOllamaProvider_Name(t *testing.T) {
	provider := NewOllamaProvider("", "")
	assert.Equal(t, "ollama", provider.Name())
}

func TestOllamaProvider_ParseIntent_Success(t *testing.T) {
	// Create mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/generate", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Return mock Ollama response with JSON
		response := OllamaResponse{
			Model:    "llama3.1",
			Response: `{"intent_type":"scaling","target":"nf-sim","namespace":"ran-a","replicas":5,"source":"user","status":"pending","target_resources":["deployment/nf-sim"]}`,
			Done:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL, "llama3.1")

	intent, err := provider.ParseIntent(context.Background(), "scale nf-sim to 5 in ns ran-a")

	require.NoError(t, err)
	assert.Equal(t, "scaling", intent["intent_type"])
	assert.Equal(t, "nf-sim", intent["target"])
	assert.Equal(t, "ran-a", intent["namespace"])
	assert.Equal(t, 5, intent["replicas"])
	assert.Equal(t, "user", intent["source"])
	assert.Equal(t, "pending", intent["status"])
}

func TestOllamaProvider_ParseIntent_WithExplanation(t *testing.T) {
	// Test parsing when Ollama includes explanation text
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := OllamaResponse{
			Model:    "llama3.1",
			Response: `Here is the parsed intent: {"intent_type":"deployment","target":"nginx","namespace":"production","replicas":3,"source":"user","status":"pending","target_resources":["deployment/nginx"]} Hope this helps!`,
			Done:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL, "llama3.1")

	intent, err := provider.ParseIntent(context.Background(), "deploy nginx with 3 replicas in namespace production")

	require.NoError(t, err)
	assert.Equal(t, "deployment", intent["intent_type"])
	assert.Equal(t, "nginx", intent["target"])
	assert.Equal(t, 3, intent["replicas"])
}

func TestOllamaProvider_ParseIntent_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL, "llama3.1")

	_, err := provider.ParseIntent(context.Background(), "test command")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ollama API returned status 500")
}

func TestOllamaProvider_ParseIntent_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := OllamaResponse{
			Model:    "llama3.1",
			Response: "This is not valid JSON at all",
			Done:     true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	provider := NewOllamaProvider(server.URL, "llama3.1")

	_, err := provider.ParseIntent(context.Background(), "test command")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no valid JSON found")
}

func TestOllamaProvider_ValidateIntent(t *testing.T) {
	provider := NewOllamaProvider("", "")

	tests := []struct {
		name    string
		intent  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid_scaling_intent",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "nf-sim",
				"namespace":   "default",
				"replicas":    5.0, // float64 from JSON
				"source":      "user",
				"status":      "pending",
			},
			wantErr: false,
		},
		{
			name: "missing_required_field",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "nf-sim",
				// missing namespace
				"source": "user",
				"status": "pending",
			},
			wantErr: true,
			errMsg:  "missing required field: namespace",
		},
		{
			name: "invalid_intent_type",
			intent: map[string]interface{}{
				"intent_type": "invalid",
				"target":      "nf-sim",
				"namespace":   "default",
				"source":      "user",
				"status":      "pending",
			},
			wantErr: true,
			errMsg:  "invalid intent_type",
		},
		{
			name: "replicas_out_of_range",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "nf-sim",
				"namespace":   "default",
				"replicas":    150, // > 100
				"source":      "user",
				"status":      "pending",
			},
			wantErr: true,
			errMsg:  "replicas must be between 0 and 100",
		},
		{
			name: "auto_generate_target_resources",
			intent: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "nf-sim",
				"namespace":   "default",
				"replicas":    5,
				"source":      "user",
				"status":      "pending",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.validateIntent(tt.intent)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				// Check that target_resources was generated if not present
				if _, ok := tt.intent["target_resources"]; !ok {
					t.Error("target_resources should be auto-generated")
				}
			}
		})
	}
}

func TestOllamaProvider_BuildPrompt(t *testing.T) {
	provider := NewOllamaProvider("", "")

	prompt := provider.buildPrompt("scale nf-sim to 5 in ns ran-a")

	assert.Contains(t, prompt, "scale nf-sim to 5 in ns ran-a")
	assert.Contains(t, prompt, "intent_type")
	assert.Contains(t, prompt, "JSON")
	assert.Contains(t, prompt, "Examples:")
}
