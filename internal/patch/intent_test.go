package patch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadIntent(t *testing.T) {
	// Create test intent file
	tempDir := t.TempDir()
	intentPath := filepath.Join(tempDir, "test-intent.json")
	
	validIntent := `{
		"intent_type": "scaling",
		"target": "test-deployment",
		"namespace": "test-ns",
		"replicas": 5,
		"reason": "test reason"
	}`
	
	if err := os.WriteFile(intentPath, []byte(validIntent), 0644); err != nil {
		t.Fatalf("Failed to write test intent: %v", err)
	}
	
	// Test loading valid intent
	intent, err := LoadIntent(intentPath)
	if err != nil {
		t.Fatalf("Failed to load valid intent: %v", err)
	}
	
	if intent.Target != "test-deployment" {
		t.Errorf("Expected target 'test-deployment', got '%s'", intent.Target)
	}
	if intent.Namespace != "test-ns" {
		t.Errorf("Expected namespace 'test-ns', got '%s'", intent.Namespace)
	}
	if intent.Replicas != 5 {
		t.Errorf("Expected replicas 5, got %d", intent.Replicas)
	}
}

func TestLoadIntentValidation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name: "invalid intent type",
			content: `{
				"intent_type": "invalid",
				"target": "test",
				"namespace": "ns",
				"replicas": 1
			}`,
			wantErr: "unsupported intent_type",
		},
		{
			name: "missing target",
			content: `{
				"intent_type": "scaling",
				"namespace": "ns",
				"replicas": 1
			}`,
			wantErr: "target is required",
		},
		{
			name: "missing namespace",
			content: `{
				"intent_type": "scaling",
				"target": "test",
				"replicas": 1
			}`,
			wantErr: "namespace is required",
		},
		{
			name: "negative replicas",
			content: `{
				"intent_type": "scaling",
				"target": "test",
				"namespace": "ns",
				"replicas": -1
			}`,
			wantErr: "replicas must be >= 0",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			intentPath := filepath.Join(tempDir, "intent.json")
			
			if err := os.WriteFile(intentPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test intent: %v", err)
			}
			
			_, err := LoadIntent(intentPath)
			if err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.wantErr)
			}
		})
	}
}