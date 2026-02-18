package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
)

func TestNLToIntentHandler(t *testing.T) {
	// Create a test handler
	handler := &LLMProcessorHandler{
		config: &config.LLMProcessorConfig{
			ServiceVersion: "test-1.0.0",
		},
		logger: slog.Default(),
	}

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		validateBody   func(t *testing.T, body map[string]interface{})
	}{
		// Valid cases
		{
			name:           "Valid scaling intent with namespace",
			method:         http.MethodPost,
			body:           "scale nf-sim to 4 in ns ran-a",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				if body["intent_type"] != "scaling" {
					t.Errorf("Expected intent_type 'scaling', got %v", body["intent_type"])
				}
				if body["target"] != "nf-sim" {
					t.Errorf("Expected target 'nf-sim', got %v", body["target"])
				}
				replicas, ok := body["replicas"].(float64) // JSON numbers are float64
				if !ok || int(replicas) != 4 {
					t.Errorf("Expected replicas 4, got %v", body["replicas"])
				}
				if body["namespace"] != "ran-a" {
					t.Errorf("Expected namespace 'ran-a', got %v", body["namespace"])
				}
			},
		},
		{
			name:           "Valid scaling intent without namespace",
			method:         http.MethodPost,
			body:           "scale my-app to 3",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				if body["intent_type"] != "scaling" {
					t.Errorf("Expected intent_type 'scaling', got %v", body["intent_type"])
				}
				if body["target"] != "my-app" {
					t.Errorf("Expected target 'my-app', got %v", body["target"])
				}
				replicas, ok := body["replicas"].(float64)
				if !ok || int(replicas) != 3 {
					t.Errorf("Expected replicas 3, got %v", body["replicas"])
				}
				if body["namespace"] != "default" {
					t.Errorf("Expected namespace 'default', got %v", body["namespace"])
				}
			},
		},
		{
			name:           "Valid deployment intent",
			method:         http.MethodPost,
			body:           "deploy nginx in ns production",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				if body["intent_type"] != "deployment" {
					t.Errorf("Expected intent_type 'deployment', got %v", body["intent_type"])
				}
				if body["target"] != "nginx" {
					t.Errorf("Expected target 'nginx', got %v", body["target"])
				}
				if body["namespace"] != "production" {
					t.Errorf("Expected namespace 'production', got %v", body["namespace"])
				}
			},
		},
		// Invalid cases
		{
			name:           "Empty body",
			method:         http.MethodPost,
			body:           "",
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				if body["error"] == nil {
					t.Error("Expected error in response")
				}
			},
		},
		{
			name:           "Invalid method",
			method:         http.MethodGet,
			body:           "scale nf-sim to 4",
			expectedStatus: http.StatusMethodNotAllowed,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				if body["error"] == nil {
					t.Error("Expected error in response")
				}
			},
		},
		{
			name:           "Unparseable text",
			method:         http.MethodPost,
			body:           "this is not a valid intent command",
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				if body["error"] == nil {
					t.Error("Expected error in response")
				}
				errorMsg, ok := body["error"].(string)
				if !ok || !bytes.Contains([]byte(errorMsg), []byte("Failed to parse intent")) {
					t.Errorf("Expected parse error message, got %v", body["error"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(tt.method, "/nl/intent", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "text/plain")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			handler.NLToIntentHandler(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Parse and validate response body
			var responseBody map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
				t.Fatalf("Failed to parse response body: %v", err)
			}

			// Run custom validation
			tt.validateBody(t, responseBody)
		})
	}
}

// TestIntentParserEdgeCases tests edge cases for the intent parser
func TestIntentParserEdgeCases(t *testing.T) {
	handler := &LLMProcessorHandler{
		config: &config.LLMProcessorConfig{
			ServiceVersion: "test-1.0.0",
		},
		logger: slog.Default(),
	}

	tests := []struct {
		name           string
		body           string
		expectedStatus int
		validateError  string
	}{
		{
			name:           "Scale with negative replicas",
			body:           "scale app to -1 in ns test",
			expectedStatus: http.StatusBadRequest,
			validateError:  "Failed to parse intent",
		},
		{
			name:           "Scale with non-numeric replicas",
			body:           "scale app to abc in ns test",
			expectedStatus: http.StatusBadRequest,
			validateError:  "Failed to parse intent",
		},
		{
			name:           "Delete intent",
			body:           "delete old-app from ns staging",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Update configuration intent",
			body:           "update myapp set replicas=5 in ns prod",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Case insensitive parsing",
			body:           "SCALE APP TO 2 IN NS TEST",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Scale with zero replicas",
			body:           "scale app to 0 in ns test",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/nl/intent", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "text/plain")

			rr := httptest.NewRecorder()
			handler.NLToIntentHandler(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			var responseBody map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &responseBody); err != nil {
				t.Fatalf("Failed to parse response body: %v", err)
			}

			if tt.validateError != "" {
				errorMsg, ok := responseBody["error"].(string)
				if !ok {
					t.Error("Expected error in response")
				} else if !bytes.Contains([]byte(errorMsg), []byte(tt.validateError)) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.validateError, errorMsg)
				}
			}
		})
	}
}
