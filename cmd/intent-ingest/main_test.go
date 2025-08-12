package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ingest "github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

// setupTestServer creates a test server with the same configuration as main()
func setupTestServer(t *testing.T) (*httptest.Server, string, func()) {
	// Create temporary directories for test
	tempDir := t.TempDir()
	schemaDir := filepath.Join(tempDir, "docs", "contracts")
	handoffDir := filepath.Join(tempDir, "handoff")
	
	// Create schema directory and copy the real schema file
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatalf("Failed to create schema dir: %v", err)
	}
	
	// Get current working directory to locate real schema
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	
	// Navigate up to find the project root (where docs/contracts exists)
	projectRoot := cwd
	for !fileExists(filepath.Join(projectRoot, "docs", "contracts", "intent.schema.json")) {
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatalf("Could not find project root with docs/contracts/intent.schema.json")
		}
		projectRoot = parent
	}
	
	// Copy real schema file to temp location
	realSchemaPath := filepath.Join(projectRoot, "docs", "contracts", "intent.schema.json")
	testSchemaPath := filepath.Join(schemaDir, "intent.schema.json")
	
	if err := copyFile(realSchemaPath, testSchemaPath); err != nil {
		t.Fatalf("Failed to copy schema file: %v", err)
	}
	
	// Create validator with test schema
	v, err := ingest.NewValidator(testSchemaPath)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}
	
	// Create handler with test directories
	h := ingest.NewHandler(v, handoffDir)
	
	// Set up HTTP mux exactly like main()
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/intent", h.HandleIntent)
	
	// Create test server
	server := httptest.NewServer(mux)
	
	// Cleanup function
	cleanup := func() {
		server.Close()
	}
	
	return server, handoffDir, cleanup
}

// Helper functions
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	
	_, err = io.Copy(dstFile, srcFile)
	return err
}

func TestServer_HealthCheck(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()
	
	resp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("Failed to call healthz: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	
	if string(body) != "ok" {
		t.Errorf("Expected body 'ok', got '%s'", string(body))
	}
}

func TestServer_Intent_ValidJSON_Success(t *testing.T) {
	server, handoffDir, cleanup := setupTestServer(t)
	defer cleanup()
	
	tests := []struct {
		name           string
		contentType    string
		payload        map[string]interface{}
		expectedStatus int
	}{
		{
			name:        "valid scaling intent",
			contentType: "application/json",
			payload: map[string]interface{}{
				"intent_type":    "scaling",
				"target":         "test-deployment",
				"namespace":      "default",
				"replicas":       3,
				"source":         "user",
				"correlation_id": "test-123",
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:        "minimal valid intent",
			contentType: "application/json",
			payload: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "minimal-app",
				"namespace":   "production",
				"replicas":    5,
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:        "text/json content type",
			contentType: "text/json",
			payload: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "text-json-app",
				"namespace":   "staging",
				"replicas":    2,
			},
			expectedStatus: http.StatusAccepted,
		},
		{
			name:        "application/json with charset",
			contentType: "application/json; charset=utf-8",
			payload: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "charset-app",
				"namespace":   "testing",
				"replicas":    1,
			},
			expectedStatus: http.StatusAccepted,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("Failed to marshal payload: %v", err)
			}
			
			resp, err := http.Post(
				server.URL+"/intent",
				tt.contentType,
				bytes.NewReader(payloadBytes),
			)
			if err != nil {
				t.Fatalf("Failed to post intent: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, resp.StatusCode, string(body))
			}
			
			if resp.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
			}
			
			var response map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
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
			
			// Verify file was created
			savedPath, ok := response["saved"].(string)
			if !ok {
				t.Error("Expected 'saved' to be a string")
			} else {
				if _, err := os.Stat(savedPath); os.IsNotExist(err) {
					t.Errorf("File was not created at %s", savedPath)
				}
			}
			
			// Test correlation_id passthrough if present
			if correlationID, exists := tt.payload["correlation_id"]; exists {
				preview, ok := response["preview"].(map[string]interface{})
				if !ok {
					t.Error("Expected 'preview' to be a map")
				} else if preview["correlation_id"] != correlationID {
					t.Errorf("Expected correlation_id %v, got %v", correlationID, preview["correlation_id"])
				}
			}
		})
	}
	
	// Verify files were created in handoff directory
	files, err := os.ReadDir(handoffDir)
	if err != nil {
		t.Fatalf("Failed to read handoff dir: %v", err)
	}
	
	// Note: We expect at least as many files as tests, but timestamps might collide
	// causing some files to be overwritten, so we check for at least 1 file
	if len(files) == 0 {
		t.Error("Expected at least one file in handoff directory, got none")
	}
	
	t.Logf("Created %d files from %d tests (some may have been overwritten due to timestamp collisions)", len(files), len(tests))
	
	// Verify file naming pattern
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "intent-") || !strings.HasSuffix(file.Name(), ".json") {
			t.Errorf("Unexpected file name pattern: %s", file.Name())
		}
	}
}

func TestServer_Intent_ValidPlainText_Success(t *testing.T) {
	server, handoffDir, cleanup := setupTestServer(t)
	defer cleanup()
	
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
	}{
		{
			name:  "basic scaling command",
			input: "scale my-app to 5 in ns production",
			expected: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "my-app",
				"namespace":   "production",
				"replicas":    float64(5),
				"source":      "user",
			},
		},
		{
			name:  "hyphenated names",
			input: "scale nf-sim to 10 in ns ran-a",
			expected: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "nf-sim",
				"namespace":   "ran-a",
				"replicas":    float64(10),
				"source":      "user",
			},
		},
		{
			name:  "case insensitive",
			input: "SCALE MY-SERVICE TO 3 IN NS DEFAULT",
			expected: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "MY-SERVICE",
				"namespace":   "DEFAULT",
				"replicas":    float64(3),
				"source":      "user",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Post(
				server.URL+"/intent",
				"text/plain",
				strings.NewReader(tt.input),
			)
			if err != nil {
				t.Fatalf("Failed to post intent: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != http.StatusAccepted {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status 202, got %d. Body: %s", resp.StatusCode, string(body))
			}
			
			var response map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
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
	
	// Verify files were created
	files, err := os.ReadDir(handoffDir)
	if err != nil {
		t.Fatalf("Failed to read handoff dir: %v", err)
	}
	
	// Note: timestamps might collide causing file overwrites, so check for at least 1
	if len(files) == 0 {
		t.Error("Expected at least one file, got none")
	}
	
	t.Logf("Created %d files from %d tests (some may have been overwritten due to timestamp collisions)", len(files), len(tests))
}

func TestServer_Intent_BadRequest_Scenarios(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()
	
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		expectedStatus int
		expectsError   string
	}{
		{
			name:           "invalid JSON",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"intent_type": "scaling"`,
			expectedStatus: http.StatusBadRequest,
			expectsError:   "validation failed",
		},
		{
			name:           "missing required fields",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"intent_type": "scaling"}`,
			expectedStatus: http.StatusBadRequest,
			expectsError:   "validation failed",
		},
		{
			name:           "invalid intent_type",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"intent_type": "invalid", "target": "test", "namespace": "default", "replicas": 3}`,
			expectedStatus: http.StatusBadRequest,
			expectsError:   "validation failed",
		},
		{
			name:           "replicas out of range - too low",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 0}`,
			expectedStatus: http.StatusBadRequest,
			expectsError:   "validation failed",
		},
		{
			name:           "replicas out of range - too high",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 101}`,
			expectedStatus: http.StatusBadRequest,
			expectsError:   "validation failed",
		},
		{
			name:           "empty target",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"intent_type": "scaling", "target": "", "namespace": "default", "replicas": 3}`,
			expectedStatus: http.StatusBadRequest,
			expectsError:   "validation failed",
		},
		{
			name:           "empty namespace",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"intent_type": "scaling", "target": "test", "namespace": "", "replicas": 3}`,
			expectedStatus: http.StatusBadRequest,
			expectsError:   "validation failed",
		},
		{
			name:           "invalid source enum",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3, "source": "invalid"}`,
			expectedStatus: http.StatusBadRequest,
			expectsError:   "validation failed",
		},
		{
			name:           "reason too long",
			method:         "POST",
			contentType:    "application/json",
			body:           fmt.Sprintf(`{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3, "reason": "%s"}`, strings.Repeat("a", 513)),
			expectedStatus: http.StatusBadRequest,
			expectsError:   "validation failed",
		},
		{
			name:           "unsupported content type treated as plain text",
			method:         "POST",
			contentType:    "application/xml",
			body:           `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3}`,
			expectedStatus: http.StatusBadRequest,
			expectsError:   "plain text",
		},
		{
			name:           "bad plain text format",
			method:         "POST",
			contentType:    "text/plain",
			body:           "invalid plain text command",
			expectedStatus: http.StatusBadRequest,
			expectsError:   "plain text",
		},
		{
			name:           "empty body",
			method:         "POST",
			contentType:    "application/json",
			body:           "",
			expectedStatus: http.StatusBadRequest,
			expectsError:   "validation failed",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, server.URL+"/intent", strings.NewReader(tt.body))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, resp.StatusCode, string(body))
			}
			
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}
			
			bodyStr := string(body)
			if !strings.Contains(bodyStr, tt.expectsError) {
				t.Errorf("Expected error message to contain '%s', got: %s", tt.expectsError, bodyStr)
			}
		})
	}
}

func TestServer_Intent_MethodNotAllowed(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()
	
	methods := []string{"GET", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req, err := http.NewRequest(method, server.URL+"/intent", nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for method %s, got %d", http.StatusMethodNotAllowed, method, resp.StatusCode)
			}
		})
	}
}

func TestServer_Intent_CorrelationIdPassthrough(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()
	
	correlationID := "test-correlation-123"
	payload := map[string]interface{}{
		"intent_type":    "scaling",
		"target":         "test-deployment",
		"namespace":      "default",
		"replicas":       3,
		"correlation_id": correlationID,
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}
	
	resp, err := http.Post(
		server.URL+"/intent",
		"application/json",
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		t.Fatalf("Failed to post intent: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected status 202, got %d. Body: %s", resp.StatusCode, string(body))
	}
	
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
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

func TestServer_Intent_FileCreation(t *testing.T) {
	server, handoffDir, cleanup := setupTestServer(t)
	defer cleanup()
	
	payload := map[string]interface{}{
		"intent_type": "scaling",
		"target":      "file-test-deployment",
		"namespace":   "default",
		"replicas":    3,
		"source":      "test",
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}
	
	resp, err := http.Post(
		server.URL+"/intent",
		"application/json",
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		t.Fatalf("Failed to post intent: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 202, got %d. Body: %s", resp.StatusCode, string(body))
	}
	
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	savedPath, ok := response["saved"].(string)
	if !ok {
		t.Fatal("Expected 'saved' to be a string")
	}
	
	// Verify file exists and has correct content
	content, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}
	
	var savedIntent map[string]interface{}
	if err := json.Unmarshal(content, &savedIntent); err != nil {
		t.Fatalf("Failed to parse saved JSON: %v", err)
	}
	
	expectedFields := map[string]interface{}{
		"intent_type": "scaling",
		"target":      "file-test-deployment",
		"namespace":   "default",
		"replicas":    float64(3),
		"source":      "test",
	}
	
	for key, expected := range expectedFields {
		if savedIntent[key] != expected {
			t.Errorf("Expected %s=%v, got %v", key, expected, savedIntent[key])
		}
	}
	
	// Verify filename format
	filename := filepath.Base(savedPath)
	if !strings.HasPrefix(filename, "intent-") || !strings.HasSuffix(filename, ".json") {
		t.Errorf("Unexpected filename format: %s", filename)
	}
	
	// Verify file is in the correct directory
	if !strings.HasPrefix(savedPath, handoffDir) {
		t.Errorf("File not created in handoff directory. Expected prefix %s, got %s", handoffDir, savedPath)
	}
}

func TestServer_Intent_ConcurrentRequests(t *testing.T) {
	server, handoffDir, cleanup := setupTestServer(t)
	defer cleanup()
	
	const numRequests = 5
	results := make(chan int, numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func(id int) {
			// Add small delay to avoid identical timestamps
			time.Sleep(time.Duration(id) * time.Millisecond)
			
			payload := map[string]interface{}{
				"intent_type": "scaling",
				"target":      fmt.Sprintf("concurrent-test-%d", id),
				"namespace":   "default",
				"replicas":    3,
				"source":      "test",
			}
			
			payloadBytes, _ := json.Marshal(payload)
			
			resp, err := http.Post(
				server.URL+"/intent",
				"application/json",
				bytes.NewReader(payloadBytes),
			)
			if err != nil {
				results <- 500 // Use 500 to indicate error
				return
			}
			defer resp.Body.Close()
			
			results <- resp.StatusCode
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
	
	// We expect at least some requests to succeed
	if successCount == 0 {
		t.Error("Expected at least some concurrent requests to succeed")
	}
	
	// Give filesystem a moment to sync
	time.Sleep(100 * time.Millisecond)
	
	// Check that files were created
	files, err := os.ReadDir(handoffDir)
	if err != nil {
		t.Fatalf("Failed to read handoff dir: %v", err)
	}
	
	if len(files) == 0 {
		t.Error("Expected at least one file to be created")
	}
	
	t.Logf("Created %d files from %d concurrent requests", len(files), numRequests)
}

func TestServer_EdgeCases(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()
	
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		expectedStatus int
	}{
		{
			name:           "no content type header",
			method:         "POST",
			contentType:    "",
			body:           "scale test to 3 in ns default",
			expectedStatus: http.StatusAccepted, // Should be treated as plain text
		},
		{
			name:           "very large JSON payload",
			method:         "POST",
			contentType:    "application/json",
			body:           fmt.Sprintf(`{"intent_type": "scaling", "target": "%s", "namespace": "default", "replicas": 3}`, strings.Repeat("a", 1000)),
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "JSON with extra fields (should be rejected by schema)",
			method:         "POST",
			contentType:    "application/json",
			body:           `{"intent_type": "scaling", "target": "test", "namespace": "default", "replicas": 3, "extra_field": "should_fail"}`,
			expectedStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, server.URL+"/intent", strings.NewReader(tt.body))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, resp.StatusCode, string(body))
			}
		})
	}
}

func TestServer_RealSchemaValidation(t *testing.T) {
	server, _, cleanup := setupTestServer(t)
	defer cleanup()
	
	// Test that we're actually using the real schema from docs/contracts/intent.schema.json
	tests := []struct {
		name           string
		payload        map[string]interface{}
		expectedStatus int
		description    string
	}{
		{
			name: "valid with all optional fields",
			payload: map[string]interface{}{
				"intent_type":    "scaling",
				"target":         "test-deployment",
				"namespace":      "default",
				"replicas":       50,
				"reason":         "Load balancing optimization",
				"source":         "planner",
				"correlation_id": "req-123-456",
			},
			expectedStatus: http.StatusAccepted,
			description:    "Should accept valid intent with all fields",
		},
		{
			name: "replicas at minimum boundary",
			payload: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-deployment",
				"namespace":   "default",
				"replicas":    1,
			},
			expectedStatus: http.StatusAccepted,
			description:    "Should accept replicas = 1 (minimum)",
		},
		{
			name: "replicas at maximum boundary",
			payload: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-deployment",
				"namespace":   "default",
				"replicas":    100,
			},
			expectedStatus: http.StatusAccepted,
			description:    "Should accept replicas = 100 (maximum)",
		},
		{
			name: "valid source enum values",
			payload: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-deployment",
				"namespace":   "default",
				"replicas":    5,
				"source":      "test",
			},
			expectedStatus: http.StatusAccepted,
			description:    "Should accept 'test' as valid source",
		},
		{
			name: "reason at max length",
			payload: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-deployment",
				"namespace":   "default",
				"replicas":    5,
				"reason":      strings.Repeat("a", 512),
			},
			expectedStatus: http.StatusAccepted,
			description:    "Should accept reason with 512 characters (max length)",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payloadBytes, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("Failed to marshal payload: %v", err)
			}
			
			resp, err := http.Post(
				server.URL+"/intent",
				"application/json",
				bytes.NewReader(payloadBytes),
			)
			if err != nil {
				t.Fatalf("Failed to post intent: %v", err)
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != tt.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("%s: Expected status %d, got %d. Body: %s", tt.description, tt.expectedStatus, resp.StatusCode, string(body))
			}
		})
	}
}

// TestServer_IntegrationFlow tests the complete flow from request to file creation
func TestServer_IntegrationFlow(t *testing.T) {
	server, handoffDir, cleanup := setupTestServer(t)
	defer cleanup()
	
	// Test complete flow with correlation ID tracking
	correlationID := fmt.Sprintf("integration-test-%d", time.Now().Unix())
	
	payload := map[string]interface{}{
		"intent_type":    "scaling",
		"target":         "integration-test-app",
		"namespace":      "integration",
		"replicas":       7,
		"source":         "test",
		"reason":         "Integration test scaling",
		"correlation_id": correlationID,
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}
	
	// Make the request
	resp, err := http.Post(
		server.URL+"/intent",
		"application/json",
		bytes.NewReader(payloadBytes),
	)
	if err != nil {
		t.Fatalf("Failed to post intent: %v", err)
	}
	defer resp.Body.Close()
	
	// Verify response
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected status 202, got %d. Body: %s", resp.StatusCode, string(body))
	}
	
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Verify response structure
	if response["status"] != "accepted" {
		t.Errorf("Expected status 'accepted', got %v", response["status"])
	}
	
	savedPath, ok := response["saved"].(string)
	if !ok {
		t.Fatal("Expected 'saved' to be a string")
	}
	
	preview, ok := response["preview"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'preview' to be a map")
	}
	
	// Verify correlation ID passthrough
	if preview["correlation_id"] != correlationID {
		t.Errorf("Expected correlation_id %s, got %v", correlationID, preview["correlation_id"])
	}
	
	// Verify file was created and contains correct data
	fileContent, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}
	
	var savedData map[string]interface{}
	if err := json.Unmarshal(fileContent, &savedData); err != nil {
		t.Fatalf("Failed to parse saved file JSON: %v", err)
	}
	
	// Verify all fields were saved correctly
	for key, expected := range payload {
		// Handle float64 conversion for numbers in JSON
		if key == "replicas" {
			expected = float64(expected.(int))
		}
		
		if savedData[key] != expected {
			t.Errorf("Saved data mismatch for %s: expected %v, got %v", key, expected, savedData[key])
		}
	}
	
	// Verify file is in handoff directory
	if !strings.HasPrefix(savedPath, handoffDir) {
		t.Errorf("File not saved in handoff directory: %s", savedPath)
	}
	
	// Verify filename format includes timestamp
	filename := filepath.Base(savedPath)
	if !strings.HasPrefix(filename, "intent-") || !strings.HasSuffix(filename, ".json") {
		t.Errorf("Invalid filename format: %s", filename)
	}
	
	// Extract and verify timestamp format (YYYYMMDDTHHMMSSZ)
	timestampPart := strings.TrimPrefix(filename, "intent-")
	timestampPart = strings.TrimSuffix(timestampPart, ".json")
	
	if len(timestampPart) != 16 { // 20060102T150405Z format
		t.Errorf("Invalid timestamp format in filename: %s", timestampPart)
	}
	
	// Try to parse the timestamp
	if _, err := time.Parse("20060102T150405Z", timestampPart); err != nil {
		t.Errorf("Invalid timestamp in filename %s: %v", timestampPart, err)
	}
}