package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestIsIntentFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"valid intent file", "intent-12345.json", true},
		{"valid intent file with timestamp", "intent-20250816-123456.json", true},
		{"not intent prefix", "config-12345.json", false},
		{"not json suffix", "intent-12345.yaml", false},
		{"no extension", "intent-12345", false},
		{"just intent", "intent.json", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIntentFile(tt.filename); got != tt.want {
				t.Errorf("isIntentFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestProcessIntentFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create test schema
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["intent_type", "target", "namespace", "replicas"],
		"properties": {
			"intent_type": {"const": "scaling"},
			"target": {"type": "string", "minLength": 1},
			"namespace": {"type": "string", "minLength": 1},
			"replicas": {"type": "integer", "minimum": 1, "maximum": 100}
		}
	}`

	// Parse and compile schema
	var schemaJSON interface{}
	if err := json.Unmarshal([]byte(schemaContent), &schemaJSON); err != nil {
		t.Fatalf("Failed to parse test schema: %v", err)
	}

	compiler := jsonschema.NewCompiler()
	schemaPath := "test-schema.json"
	if err := compiler.AddResource(schemaPath, schemaJSON); err != nil {
		t.Fatalf("Failed to add schema to compiler: %v", err)
	}
	
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to compile schema: %v", err)
	}

	tests := []struct {
		name        string
		content     string
		shouldBeOK  bool
		description string
	}{
		{
			name: "valid-intent",
			content: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 5
			}`,
			shouldBeOK:  true,
			description: "Valid intent should pass validation",
		},
		{
			name: "invalid-missing-field",
			content: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"replicas": 5
			}`,
			shouldBeOK:  false,
			description: "Intent missing namespace should fail",
		},
		{
			name: "invalid-wrong-type",
			content: `{
				"intent_type": "update",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 5
			}`,
			shouldBeOK:  false,
			description: "Intent with wrong type should fail",
		},
		{
			name: "invalid-replicas-too-high",
			content: `{
				"intent_type": "scaling",
				"target": "my-deployment",
				"namespace": "default",
				"replicas": 200
			}`,
			shouldBeOK:  false,
			description: "Intent with replicas > 100 should fail",
		},
		{
			name:        "invalid-json",
			content:     `{invalid json content}`,
			shouldBeOK:  false,
			description: "Invalid JSON should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tempDir, fmt.Sprintf("intent-%s.json", tt.name))
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Capture logs by temporarily redirecting
			// For simplicity, we'll just call the function and check for panics
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("processIntentFile panicked: %v", r)
				}
			}()

			// Process the file
			processIntentFile(testFile, schema, "", true)
			
			// Note: In a real test, we'd capture and verify log output
			// For now, we just ensure it doesn't panic
		})
	}
}

func TestPostIntent(t *testing.T) {
	// Create test server
	var receivedBody []byte
	var receivedHeaders http.Header
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		body, _ := io.ReadAll(r.Body)
		receivedBody = body
		
		// Return success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "accepted"}`))
	}))
	defer server.Close()

	// Test data
	testIntent := `{
		"intent_type": "scaling",
		"target": "test-deployment",
		"namespace": "default",
		"replicas": 3
	}`

	// Create temp file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "intent-test.json")
	if err := os.WriteFile(testFile, []byte(testIntent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Call postIntent
	postIntent(testFile, []byte(testIntent), server.URL)

	// Give it time to complete (since it's async)
	time.Sleep(100 * time.Millisecond)

	// Verify request
	if string(receivedBody) != testIntent {
		t.Errorf("POST body mismatch\nGot: %s\nWant: %s", receivedBody, testIntent)
	}

	if ct := receivedHeaders.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	if xif := receivedHeaders.Get("X-Intent-File"); xif != "intent-test.json" {
		t.Errorf("X-Intent-File = %q, want %q", xif, "intent-test.json")
	}

	if ts := receivedHeaders.Get("X-Timestamp"); ts == "" {
		t.Error("X-Timestamp header not set")
	}
}

func TestProcessExistingFiles(t *testing.T) {
	// Create temporary directory with test files
	tempDir := t.TempDir()

	// Create test schema
	schemaContent := `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"required": ["intent_type", "target", "namespace", "replicas"],
		"properties": {
			"intent_type": {"const": "scaling"},
			"target": {"type": "string", "minLength": 1},
			"namespace": {"type": "string", "minLength": 1},
			"replicas": {"type": "integer", "minimum": 1, "maximum": 100}
		}
	}`

	var schemaJSON interface{}
	if err := json.Unmarshal([]byte(schemaContent), &schemaJSON); err != nil {
		t.Fatalf("Failed to parse test schema: %v", err)
	}

	compiler := jsonschema.NewCompiler()
	schemaPath := "test-schema.json"
	if err := compiler.AddResource(schemaPath, schemaJSON); err != nil {
		t.Fatalf("Failed to add schema to compiler: %v", err)
	}
	
	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		t.Fatalf("Failed to compile schema: %v", err)
	}

	// Create test files
	testFiles := []struct {
		name    string
		content string
	}{
		{
			name: "intent-1.json",
			content: `{
				"intent_type": "scaling",
				"target": "app1",
				"namespace": "default",
				"replicas": 2
			}`,
		},
		{
			name: "intent-2.json",
			content: `{
				"intent_type": "scaling",
				"target": "app2",
				"namespace": "production",
				"replicas": 5
			}`,
		},
		{
			name:    "not-an-intent.json",
			content: `{"some": "other", "data": true}`,
		},
		{
			name:    "config.yaml",
			content: `key: value`,
		},
	}

	for _, tf := range testFiles {
		filePath := filepath.Join(tempDir, tf.name)
		if err := os.WriteFile(filePath, []byte(tf.content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", tf.name, err)
		}
	}

	// Process existing files
	processExistingFiles(tempDir, schema, "")

	// Verify by checking that the function completes without error
	// In a real test, we'd capture logs and verify the correct files were processed
}

func TestMainIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This is a placeholder for an integration test
	// In a real scenario, we would:
	// 1. Start the main function in a goroutine
	// 2. Create test files in the watched directory
	// 3. Verify the expected log outputs
	// 4. Send a shutdown signal
	// 5. Verify graceful shutdown

	t.Log("Integration test placeholder - would test full watcher functionality")
}