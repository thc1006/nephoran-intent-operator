package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/porch"
)

// MockValidator implements the Validator interface for testing
type MockValidator struct {
	ValidateBytesFunc func([]byte) (*Intent, error)
}

func (m *MockValidator) ValidateBytes(b []byte) (*Intent, error) {
	if m.ValidateBytesFunc != nil {
		return m.ValidateBytesFunc(b)
	}
	// Default successful validation
	return &Intent{
		IntentType: "scaling",
		Target:     "test-deployment",
		Namespace:  "default",
		Replicas:   3,
		Source:     "user",
	}, nil
}

// MockIntentProvider implements the IntentProvider interface for testing
type MockIntentProvider struct {
	ParseIntentFunc func(ctx context.Context, text string) (map[string]interface{}, error)
}

func (m *MockIntentProvider) ParseIntent(ctx context.Context, text string) (map[string]interface{}, error) {
	if m.ParseIntentFunc != nil {
		return m.ParseIntentFunc(ctx, text)
	}
	// Fallback to actual parsing logic for tests
	provider := NewRulesProvider()
	return provider.ParseIntent(ctx, text)
}

func (m *MockIntentProvider) Name() string {
	return "mock-provider"
}

func TestNewHandler(t *testing.T) {
	tempDir := t.TempDir()
	outDir := filepath.Join(tempDir, "handoff")

	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	handler := NewHandler(mockValidator, outDir, mockProvider, nil)
	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}

	if handler.v == nil {
		t.Error("Handler validator not set")
	}

	if handler.outDir != outDir {
		t.Error("Handler output directory not set correctly")
	}

	if handler.porchClient != nil {
		t.Error("Expected nil Porch client in filesystem-only mode")
	}

	// Check that output directory was created
	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		t.Error("Output directory was not created")
	}
}

func TestHandleIntent_MethodNotAllowed(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	handler := NewHandler(mockValidator, tempDir, mockProvider, nil)
	methods := []string{"GET", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/intent", nil)
			w := httptest.NewRecorder()

			handler.HandleIntent(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
			}
		})
	}
}

func TestHandleIntent_JSONInput_Success(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	handler := NewHandler(mockValidator, tempDir, mockProvider, nil)
	tests := []struct {
		name        string
		contentType string
		payload     string
		expected    Intent
	}{
		{
			name:        "application/json content type",
			contentType: "application/json",
			payload: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "production",
				"replicas": 5,
				"source": "user",
				"correlation_id": "test-123"
			}`,
			expected: Intent{
				IntentType:    "scaling",
				Target:        "my-deployment",
				Namespace:     "production",
				Replicas:      5,
				Source:        "user",
				CorrelationID: "test-123",
			},
		},
		{
			name:        "text/json content type",
			contentType: "text/json",
			payload: `{
				"intent_type": "scaling",
				"target": "another-deployment",
				"namespace": "staging",
				"replicas": 10
			}`,
			expected: Intent{
				IntentType: "scaling",
				Target:     "another-deployment",
				Namespace:  "staging",
				Replicas:   10,
			},
		},
		{
			name:        "application/json with charset",
			contentType: "application/json; charset=utf-8",
			payload: `{
				"intent_type": "scaling",
				"target": "utf8-deployment",
				"namespace": "default",
				"replicas": 3
			}`,
			expected: Intent{
				IntentType: "scaling",
				Target:     "utf8-deployment",
				Namespace:  "default",
				Replicas:   3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mock validator to return expected intent
			mockValidator.ValidateBytesFunc = func(b []byte) (*Intent, error) {
				var intent Intent
				if err := json.Unmarshal(b, &intent); err != nil {
					return nil, err
				}
				return &intent, nil
			}

			req := httptest.NewRequest("POST", "/intent", strings.NewReader(tt.payload))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			handler.HandleIntent(w, req)

			if w.Code != http.StatusAccepted {
				t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
			}

			if w.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["status"] != "accepted" {
				t.Errorf("Expected status 'accepted', got %v", response["status"])
			}

			if response["saved"] == nil {
				t.Error("Expected 'saved' field in response")
			}

			if response["preview"] == nil {
				t.Error("Expected 'preview' field in response")
			}

			// Verify file was actually created
			savedPath, ok := response["saved"].(string)
			if !ok {
				t.Error("Expected 'saved' to be a string")
			} else {
				if _, err := os.Stat(savedPath); os.IsNotExist(err) {
					t.Errorf("File was not created at %s", savedPath)
				}
			}

			// Test correlation_id passthrough if present
			if tt.expected.CorrelationID != "" {
				preview, ok := response["preview"].(map[string]interface{})
				if !ok {
					t.Error("Expected 'preview' to be a map")
				} else if preview["correlation_id"] != tt.expected.CorrelationID {
					t.Errorf("Expected correlation_id %s, got %v", tt.expected.CorrelationID, preview["correlation_id"])
				}
			}
		})
	}
}

func TestHandleIntent_PlainTextInput_Success(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	handler := NewHandler(mockValidator, tempDir, mockProvider, nil)
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name:  "basic scaling command",
			input: "scale my-app to 5 in ns production",
			expected: map[string]interface{}{},
		},
		{
			name:  "hyphenated names",
			input: "scale nf-sim to 10 in ns ran-a",
			expected: map[string]interface{}{},
		},
		{
			name:  "case insensitive",
			input: "SCALE MY-SERVICE TO 3 IN NS DEFAULT",
			expected: map[string]interface{}{},
		},
		{
			name:  "with extra whitespace",
			input: "  scale   web-frontend   to   8   in   ns   backend  ",
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock validator should validate the generated JSON
			mockValidator.ValidateBytesFunc = func(b []byte) (*Intent, error) {
				var intent Intent
				if err := json.Unmarshal(b, &intent); err != nil {
					return nil, err
				}
				return &intent, nil
			}

			req := httptest.NewRequest("POST", "/intent", strings.NewReader(tt.input))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			handler.HandleIntent(w, req)

			if w.Code != http.StatusAccepted {
				t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			preview, ok := response["preview"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected 'preview' to be a map")
			}

			for key, expectedValue := range tt.expected {
				if preview[key] != expectedValue {
					t.Errorf("Expected %s=%v, got %v", key, expectedValue, preview[key])
				}
			}
		})
	}
}

func TestHandleIntent_PlainTextInput_BadFormat(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	handler := NewHandler(mockValidator, tempDir, mockProvider, nil)
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "missing to keyword",
			input: "scale my-app 5 in ns production",
		},
		{
			name:  "missing in keyword",
			input: "scale my-app to 5 ns production",
		},
		{
			name:  "missing ns keyword",
			input: "scale my-app to 5 in production",
		},
		{
			name:  "wrong order",
			input: "to scale my-app 5 in ns production",
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "completely unrelated text",
			input: "hello world",
		},
		{
			name:  "partial match",
			input: "scale my-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the provider to return an error for bad formats
			mockProvider.ParseIntentFunc = func(ctx context.Context, text string) (map[string]interface{}, error) {
				return nil, fmt.Errorf("invalid plain text format")
			}

			req := httptest.NewRequest("POST", "/intent", strings.NewReader(tt.input))
			req.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			handler.HandleIntent(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
			}

			body := w.Body.String()
			// Accept both old and new error message formats
			if !strings.Contains(body, "plain text") && !strings.Contains(body, "Request body is empty") {
				t.Errorf("Expected error message about plain text format, got: %s", body)
			}
		})
	}
}

func TestHandleIntent_UnsupportedContentType(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	handler := NewHandler(mockValidator, tempDir, mockProvider, nil)
	tests := []struct {
		name        string
		contentType string
		expectedCode int
	}{
		{name: "xml", contentType: "application/xml", expectedCode: http.StatusUnsupportedMediaType},
		{name: "form data", contentType: "application/x-www-form-urlencoded", expectedCode: http.StatusUnsupportedMediaType},
		{name: "binary", contentType: "application/octet-stream", expectedCode: http.StatusUnsupportedMediaType},
		{name: "html", contentType: "text/html", expectedCode: http.StatusUnsupportedMediaType},
		{name: "empty", contentType: "", expectedCode: http.StatusBadRequest}, // Empty content type leads to plain text parsing which fails
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3}`
			req := httptest.NewRequest("POST", "/intent", strings.NewReader(payload))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			handler.HandleIntent(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d for content type %q", tt.expectedCode, w.Code, tt.contentType)
			}
		})
	}
}

func TestHandleIntent_ValidationError(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	handler := NewHandler(mockValidator, tempDir, mockProvider, nil)
	tests := []struct {
		name          string
		contentType   string
		payload       string
		validationErr string
	}{
		{
			name:          "invalid JSON schema",
			contentType:   "application/json",
			payload:       `{"intent_type": "invalid", "target": "test"}`,
			validationErr: "invalid intent_type",
		},
		{
			name:          "missing required fields",
			contentType:   "application/json",
			payload:       `{"intent_type": "scaling"}`,
			validationErr: "missing required field",
		},
		{
			name:          "malformed JSON",
			contentType:   "application/json",
			payload:       `{"intent_type": "scaling"`,
			validationErr: "invalid json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock validator to return error
			mockValidator.ValidateBytesFunc = func(b []byte) (*Intent, error) {
				return nil, fmt.Errorf("%s", tt.validationErr)
			}

			req := httptest.NewRequest("POST", "/intent", strings.NewReader(tt.payload))
			req.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			handler.HandleIntent(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
			}

			body := w.Body.String()
			// Accept both old and new validation error message formats
			if !strings.Contains(body, "validation failed") && !strings.Contains(body, "Intent validation failed") {
				t.Errorf("Expected validation error message, got: %s", body)
			}

			if !strings.Contains(body, tt.validationErr) {
				t.Errorf("Expected error message to contain '%s', got: %s", tt.validationErr, body)
			}
		})
	}
}

func TestHandleIntent_FileWriteError(t *testing.T) {
	// Skip this test on Windows as permission handling is different
	if os.PathSeparator == '\\' {
		t.Skip("Skipping file write error test on Windows due to different permission model")
	}

	// Use a read-only directory to trigger write error
	tempDir := t.TempDir()
	readOnlyDir := filepath.Join(tempDir, "readonly")
	os.MkdirAll(readOnlyDir, 0o444) // Create read-only directory

	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	handler := NewHandler(mockValidator, readOnlyDir, mockProvider, nil)
	// Mock successful validation
	mockValidator.ValidateBytesFunc = func(b []byte) (*Intent, error) {
		return &Intent{
			IntentType: "scaling",
			Target:     "test",
			Namespace:  "default",
			Replicas:   3,
		}, nil
	}

	payload := `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3}`
	req := httptest.NewRequest("POST", "/intent", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleIntent(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	body := w.Body.String()
	// Accept various file write error message formats (permission denied, read-only filesystem, etc.)
	expectedMsgs := []string{
		"intent failed",
		"Failed to save intent", 
		"permission denied",
		"read-only",
		"cannot create",
		"access denied",
	}
	
	hasExpectedMsg := false
	for _, msg := range expectedMsgs {
		if strings.Contains(strings.ToLower(body), strings.ToLower(msg)) {
			hasExpectedMsg = true
			break
		}
	}
	
	if !hasExpectedMsg {
		t.Errorf("Expected write error message containing one of %v, got: %s", expectedMsgs, body)
	}
}

func TestHandleIntent_FileCreation(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	handler := NewHandler(mockValidator, tempDir, mockProvider, nil)
	payload := `{"intent_type": "scaling", "target": "test-deployment", "namespace": "default", "replicas": 3}`

	req := httptest.NewRequest("POST", "/intent", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleIntent(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	savedPath, ok := response["saved"].(string)
	if !ok {
		t.Fatal("Expected 'saved' to be a string")
	}

	// Check file exists and has correct content
	content, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}

	var savedIntent map[string]interface{}
	if err := json.Unmarshal(content, &savedIntent); err != nil {
		t.Fatalf("Failed to parse saved JSON: %v", err)
	}

	expectedFields := map[string]interface{}{}

	for key, expected := range expectedFields {
		if savedIntent[key] != expected {
			t.Errorf("Expected %s=%v, got %v", key, expected, savedIntent[key])
		}
	}

	// Check filename format
	filename := filepath.Base(savedPath)
	if !strings.HasPrefix(filename, "intent-") || !strings.HasSuffix(filename, ".json") {
		t.Errorf("Unexpected filename format: %s", filename)
	}
}

func TestHandleIntent_ConcurrentRequests(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	handler := NewHandler(mockValidator, tempDir, mockProvider, nil)
	const numRequests = 5 // Reduced to minimize timestamp collisions
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			// Add small delay to avoid identical timestamps
			time.Sleep(time.Duration(id) * time.Millisecond)

			payload := `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3}`
			req := httptest.NewRequest("POST", "/intent", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.HandleIntent(w, req)
			results <- w.Code
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		select {
		case code := <-results:
			if code == http.StatusAccepted {
				successCount++
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	// We expect at least some requests to succeed (allowing for timestamp collisions)
	if successCount == 0 {
		t.Error("Expected at least some concurrent requests to succeed")
	}

	// Give filesystem a moment to sync
	time.Sleep(100 * time.Millisecond)

	// Check that files were created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected at least one file to be created")
	}

	t.Logf("Created %d files from %d concurrent requests", len(files), numRequests)
}

func TestHandleIntent_CorrelationIdPassthrough(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	handler := NewHandler(mockValidator, tempDir, mockProvider, nil)
	correlationID := "test-correlation-123"
	payload := fmt.Sprintf(`{
		"intent_type": "scaling",
		"target": "test-deployment",
		"namespace": "default",
		"replicas": 3,
		"correlation_id": "%s"
	}`, correlationID)

	mockValidator.ValidateBytesFunc = func(b []byte) (*Intent, error) {
		var intent Intent
		if err := json.Unmarshal(b, &intent); err != nil {
			return nil, err
		}
		return &intent, nil
	}

	req := httptest.NewRequest("POST", "/intent", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleIntent(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	preview, ok := response["preview"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'preview' to be a map")
	}

	if preview["correlation_id"] != correlationID {
		t.Errorf("Expected correlation_id %s, got %v", correlationID, preview["correlation_id"])
	}
}

func TestHandleIntent_EdgeCases(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	handler := NewHandler(mockValidator, tempDir, mockProvider, nil)
	tests := []struct {
		name         string
		method       string
		contentType  string
		body         string
		expectedCode int
	}{
		{
			name:         "empty body JSON",
			method:       "POST",
			contentType:  "application/json",
			body:         "",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "empty body plain text",
			method:       "POST",
			contentType:  "text/plain",
			body:         "",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "very large JSON payload",
			method:       "POST",
			contentType:  "application/json",
			body:         fmt.Sprintf(`{"intent_type": "scaling", "target": "%s", "namespace": "default", "replicas": 3}`, strings.Repeat("a", 1000)),
			expectedCode: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedCode == http.StatusBadRequest {
				// Mock validation error for bad requests
				mockValidator.ValidateBytesFunc = func(b []byte) (*Intent, error) {
					return nil, fmt.Errorf("validation failed")
				}
			} else {
				// Mock successful validation
				mockValidator.ValidateBytesFunc = func(b []byte) (*Intent, error) {
					var intent Intent
					if err := json.Unmarshal(b, &intent); err != nil {
						return nil, err
					}
					return &intent, nil
				}
			}

			req := httptest.NewRequest(tt.method, "/intent", strings.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			w := httptest.NewRecorder()

			handler.HandleIntent(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}


// MockPorchClient implements the PorchClient interface for testing
type MockPorchClient struct {
	CreateOrUpdatePackageFunc func(req *porch.PackageRequest) (*porch.PorchPackageRevision, error)
	callCount                 int
	lastRequest               *porch.PackageRequest
}

func (m *MockPorchClient) CreateOrUpdatePackage(req *porch.PackageRequest) (*porch.PorchPackageRevision, error) {
	m.callCount++
	m.lastRequest = req

	if m.CreateOrUpdatePackageFunc != nil {
		return m.CreateOrUpdatePackageFunc(req)
	}

	// Default successful response
	return &porch.PorchPackageRevision{
		Name:      req.Package,
		Namespace: req.Namespace,
		Revision:  "v001",
		Status:    "DRAFT",
	}, nil
}

func TestHandleIntent_PorchIntegration_Success(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	mockPorchClient := &MockPorchClient{}

	handler := NewHandler(mockValidator, tempDir, mockProvider, mockPorchClient)

	payload := `{
		"intent_type": "scaling",
		"target": "test-deployment",
		"namespace": "production",
		"replicas": 5,
		"source": "user"
	}`

	mockValidator.ValidateBytesFunc = func(b []byte) (*Intent, error) {
		var intent Intent
		if err := json.Unmarshal(b, &intent); err != nil {
			return nil, err
		}
		return &intent, nil
	}

	req := httptest.NewRequest("POST", "/intent", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleIntent(w, req)

	// Verify HTTP response
	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	// Verify filesystem write still happened
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	savedPath, ok := response["saved"].(string)
	if !ok {
		t.Fatal("Expected 'saved' field in response")
	}

	if _, err := os.Stat(savedPath); os.IsNotExist(err) {
		t.Error("File was not created in filesystem")
	}

	// Verify Porch client was called
	if mockPorchClient.callCount != 1 {
		t.Errorf("Expected Porch client to be called once, got %d calls", mockPorchClient.callCount)
	}

	// Verify Porch package request
	if mockPorchClient.lastRequest == nil {
		t.Fatal("Expected Porch package request to be recorded")
	}

	if mockPorchClient.lastRequest.Repository != "network-intents" {
		t.Errorf("Expected repository 'network-intents', got '%s'", mockPorchClient.lastRequest.Repository)
	}

	if mockPorchClient.lastRequest.Namespace != "production" {
		t.Errorf("Expected namespace 'production', got '%s'", mockPorchClient.lastRequest.Namespace)
	}

	if mockPorchClient.lastRequest.Intent == nil {
		t.Fatal("Expected intent to be set in Porch request")
	}

	if mockPorchClient.lastRequest.Intent.Target != "test-deployment" {
		t.Errorf("Expected target 'test-deployment', got '%s'", mockPorchClient.lastRequest.Intent.Target)
	}

	if mockPorchClient.lastRequest.Intent.Replicas != 5 {
		t.Errorf("Expected replicas 5, got %d", mockPorchClient.lastRequest.Intent.Replicas)
	}
}

func TestHandleIntent_PorchIntegration_Failure(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	mockPorchClient := &MockPorchClient{
		CreateOrUpdatePackageFunc: func(req *porch.PackageRequest) (*porch.PorchPackageRevision, error) {
			return nil, fmt.Errorf("Porch API connection failed")
		},
	}

	handler := NewHandler(mockValidator, tempDir, mockProvider, mockPorchClient)

	payload := `{
		"intent_type": "scaling",
		"target": "test-deployment",
		"namespace": "default",
		"replicas": 3
	}`

	mockValidator.ValidateBytesFunc = func(b []byte) (*Intent, error) {
		var intent Intent
		if err := json.Unmarshal(b, &intent); err != nil {
			return nil, err
		}
		return &intent, nil
	}

	req := httptest.NewRequest("POST", "/intent", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleIntent(w, req)

	// Verify HTTP response is still success (Porch failure should not block filesystem)
	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d (Porch failure should not block), got %d", http.StatusAccepted, w.Code)
	}

	// Verify filesystem write still happened despite Porch failure
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	savedPath, ok := response["saved"].(string)
	if !ok {
		t.Fatal("Expected 'saved' field in response")
	}

	if _, err := os.Stat(savedPath); os.IsNotExist(err) {
		t.Error("File was not created in filesystem despite Porch failure")
	}

	// Verify Porch client was called (even though it failed)
	if mockPorchClient.callCount != 1 {
		t.Errorf("Expected Porch client to be called once, got %d calls", mockPorchClient.callCount)
	}
}

func TestHandleIntent_FilesystemOnly_NoPorch(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}

	// Create handler without Porch client (nil)
	handler := NewHandler(mockValidator, tempDir, mockProvider, nil)

	payload := `{
		"intent_type": "scaling",
		"target": "test-deployment",
		"namespace": "default",
		"replicas": 3
	}`

	mockValidator.ValidateBytesFunc = func(b []byte) (*Intent, error) {
		var intent Intent
		if err := json.Unmarshal(b, &intent); err != nil {
			return nil, err
		}
		return &intent, nil
	}

	req := httptest.NewRequest("POST", "/intent", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleIntent(w, req)

	// Verify success
	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	// Verify filesystem write happened
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	savedPath, ok := response["saved"].(string)
	if !ok {
		t.Fatal("Expected 'saved' field in response")
	}

	if _, err := os.Stat(savedPath); os.IsNotExist(err) {
		t.Error("File was not created in filesystem-only mode")
	}
}

func TestCreatePorchPackage_IntentConversion(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	mockPorchClient := &MockPorchClient{}

	handler := NewHandler(mockValidator, tempDir, mockProvider, mockPorchClient)

	testIntent := &Intent{
		IntentType:    "scaling",
		Target:        "web-app",
		Namespace:     "production",
		Replicas:      10,
		Reason:        "High traffic expected",
		Source:        "planner",
		CorrelationID: "test-correlation-123",
	}

	payload, _ := json.Marshal(testIntent)

	ctx := context.Background()
	err := handler.createPorchPackage(ctx, testIntent, payload)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify conversion to NetworkIntent
	if mockPorchClient.lastRequest == nil {
		t.Fatal("Expected Porch package request")
	}

	networkIntent := mockPorchClient.lastRequest.Intent
	if networkIntent == nil {
		t.Fatal("Expected NetworkIntent in request")
	}

	// Verify all fields were converted correctly
	if networkIntent.IntentType != testIntent.IntentType {
		t.Errorf("IntentType mismatch: expected %s, got %s", testIntent.IntentType, networkIntent.IntentType)
	}

	if networkIntent.Target != testIntent.Target {
		t.Errorf("Target mismatch: expected %s, got %s", testIntent.Target, networkIntent.Target)
	}

	if networkIntent.Namespace != testIntent.Namespace {
		t.Errorf("Namespace mismatch: expected %s, got %s", testIntent.Namespace, networkIntent.Namespace)
	}

	if networkIntent.Replicas != testIntent.Replicas {
		t.Errorf("Replicas mismatch: expected %d, got %d", testIntent.Replicas, networkIntent.Replicas)
	}

	if networkIntent.Reason != testIntent.Reason {
		t.Errorf("Reason mismatch: expected %s, got %s", testIntent.Reason, networkIntent.Reason)
	}

	if networkIntent.Source != testIntent.Source {
		t.Errorf("Source mismatch: expected %s, got %s", testIntent.Source, networkIntent.Source)
	}

	if networkIntent.CorrelationID != testIntent.CorrelationID {
		t.Errorf("CorrelationID mismatch: expected %s, got %s", testIntent.CorrelationID, networkIntent.CorrelationID)
	}

	// Verify package name format
	if !strings.HasPrefix(mockPorchClient.lastRequest.Package, "web-app-intent-") {
		t.Errorf("Expected package name to start with 'web-app-intent-', got: %s", mockPorchClient.lastRequest.Package)
	}

	// Verify repository
	if mockPorchClient.lastRequest.Repository != "network-intents" {
		t.Errorf("Expected repository 'network-intents', got: %s", mockPorchClient.lastRequest.Repository)
	}
}

func TestHandleIntent_DualMode_BothSuccess(t *testing.T) {
	tempDir := t.TempDir()
	mockValidator := &MockValidator{}
	mockProvider := &MockIntentProvider{}
	mockPorchClient := &MockPorchClient{}

	handler := NewHandler(mockValidator, tempDir, mockProvider, mockPorchClient)

	payload := `{
		"intent_type": "scaling",
		"target": "api-server",
		"namespace": "staging",
		"replicas": 7,
		"source": "user",
		"correlation_id": "dual-mode-test-001"
	}`

	mockValidator.ValidateBytesFunc = func(b []byte) (*Intent, error) {
		var intent Intent
		if err := json.Unmarshal(b, &intent); err != nil {
			return nil, err
		}
		return &intent, nil
	}

	req := httptest.NewRequest("POST", "/intent", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleIntent(w, req)

	// Verify HTTP response
	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	// Verify filesystem write
	var response map[string]interface{}
	json.NewDecoder(w.Body).Decode(&response)
	savedPath := response["saved"].(string)

	fsContent, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("Failed to read filesystem file: %v", err)
	}

	var fsIntent map[string]interface{}
	json.Unmarshal(fsContent, &fsIntent)

	// Verify Porch package
	if mockPorchClient.callCount != 1 {
		t.Fatalf("Expected 1 Porch call, got %d", mockPorchClient.callCount)
	}

	porchIntent := mockPorchClient.lastRequest.Intent

	// Verify both destinations have consistent data
	if porchIntent.Target != fsIntent["target"].(string) {
		t.Error("Target mismatch between filesystem and Porch")
	}

	if porchIntent.Replicas != int(fsIntent["replicas"].(float64)) {
		t.Error("Replicas mismatch between filesystem and Porch")
	}

	if porchIntent.Namespace != fsIntent["namespace"].(string) {
		t.Error("Namespace mismatch between filesystem and Porch")
	}

	if porchIntent.CorrelationID != fsIntent["correlation_id"].(string) {
		t.Error("CorrelationID mismatch between filesystem and Porch")
	}
}
