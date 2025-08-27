package porch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseIntentFromFile(t *testing.T) {
	tests := []struct {
		name        string
		intentFile  string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid intent",
			intentFile:  "testdata/valid_intent.json",
			expectError: false,
		},
		{
			name:        "Invalid intent - negative replicas",
			intentFile:  "testdata/invalid_intent.json",
			expectError: true,
			errorMsg:    "replicas must be at least 1",
		},
		{
			name:        "Non-existent file",
			intentFile:  "testdata/nonexistent.json",
			expectError: true,
			errorMsg:    "failed to read intent file",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, err := ParseIntentFromFile(tt.intentFile)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if intent == nil {
					t.Error("Expected intent but got nil")
				} else {
					// Validate parsed intent
					if intent.IntentType != "scaling" {
						t.Errorf("Expected intent_type 'scaling', got %s", intent.IntentType)
					}
					if intent.Target == "" {
						t.Error("Target should not be empty")
					}
					if intent.Namespace == "" {
						t.Error("Namespace should not be empty")
					}
				}
			}
		})
	}
}

func TestBuildKRMPackage(t *testing.T) {
	validIntent := &ScalingIntent{
		IntentType: "scaling",
		Target:     "test-deployment",
		Namespace:  "test-ns",
		Replicas:   3,
	}
	
	// FIXME: Correcting linter error 'cannot use &string as string value'. The struct fields expect string values, not pointers.
	// Changed from pointer assignment (&reasonStr) to direct string assignment.
	reasonStr := "Test reason"
	sourceStr := "test"
	correlationStr := "corr-123"
	
	intentWithOptionals := &ScalingIntent{
		IntentType:    "scaling",
		Target:        "test-deployment",
		Namespace:     "test-ns",
		Replicas:      5,
		Reason:        reasonStr,
		Source:        sourceStr,
		CorrelationID: correlationStr,
	}
	
	tests := []struct {
		name        string
		intent      *ScalingIntent
		packageName string
		expectError bool
	}{
		{
			name:        "Valid intent - minimal",
			intent:      validIntent,
			packageName: "test-package",
			expectError: false,
		},
		{
			name:        "Valid intent - with optionals",
			intent:      intentWithOptionals,
			packageName: "test-package-full",
			expectError: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg, err := BuildKRMPackage(tt.intent, tt.packageName)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if pkg == nil {
					t.Error("Expected package but got nil")
				} else {
					// Check required files exist
					requiredFiles := []string{"Kptfile", "deployment-patch.yaml", "kustomization.yaml", "README.md"}
					for _, file := range requiredFiles {
						if _, exists := pkg.Content[file]; !exists {
							t.Errorf("Missing required file: %s", file)
						}
					}
					
					// Check deployment patch contains replicas
					if deployPatch, ok := pkg.Content["deployment-patch.yaml"]; ok {
						if !strings.Contains(deployPatch, "replicas:") {
							t.Error("Deployment patch should contain replicas field")
						}
						if !strings.Contains(deployPatch, tt.intent.Target) {
							t.Error("Deployment patch should contain target name")
						}
					}
				}
			}
		})
	}
}

func TestCreatePackageFromIntent_DryRun(t *testing.T) {
	// Create a test client with dry-run enabled
	client, err := NewDirectClient("http://test-endpoint", "test-namespace")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.SetDryRun(true)
	
	ctx := context.Background()
	
	// Test with valid intent
	result, err := client.CreatePackageFromIntent(
		ctx,
		"testdata/valid_intent.json",
		"test-repo",
		"test-package",
		"Test revision message",
	)
	
	if err != nil {
		t.Errorf("Unexpected error in dry-run: %v", err)
	}
	
	if result == nil {
		t.Error("Expected result but got nil")
	} else {
		if result.Name != "test-package" {
			t.Errorf("Expected package name 'test-package', got %s", result.Name)
		}
		if !strings.Contains(result.PackagePath, "examples/packages/test-repo/test-package") {
			t.Errorf("Unexpected package path: %s", result.PackagePath)
		}
		if !strings.Contains(result.CommitURL, "test-repo/packages/test-package") {
			t.Errorf("Unexpected commit URL: %s", result.CommitURL)
		}
	}
	
	// Verify no files were actually created (dry-run)
	if _, err := os.Stat(result.PackagePath); !os.IsNotExist(err) {
		t.Error("Dry-run should not create actual files")
	}
}

func TestCreatePackageFromIntent_InvalidIntent(t *testing.T) {
	client, err := NewDirectClient("http://test-endpoint", "test-namespace")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	
	ctx := context.Background()
	
	// Test with invalid intent (negative replicas)
	_, err = client.CreatePackageFromIntent(
		ctx,
		"testdata/invalid_intent.json",
		"test-repo",
		"test-package",
		"",
	)
	
	if err == nil {
		t.Error("Expected error for invalid intent but got nil")
	} else if !strings.Contains(err.Error(), "replicas must be at least 1") {
		t.Errorf("Expected replicas validation error, got: %v", err)
	}
}

func TestCreatePackageFromIntent_RealMode(t *testing.T) {
	// Create a test client without dry-run
	client, err := NewDirectClient("http://test-endpoint", "test-namespace")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	
	ctx := context.Background()
	
	// Test with valid intent in real mode
	result, err := client.CreatePackageFromIntent(
		ctx,
		"testdata/valid_intent.json",
		"test-repo",
		"test-package-real",
		"Real mode test",
	)
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if result == nil {
		t.Error("Expected result but got nil")
	} else {
		// Verify files were created
		kptfilePath := filepath.Join(result.PackagePath, "Kptfile")
		if _, err := os.Stat(kptfilePath); os.IsNotExist(err) {
			t.Error("Expected Kptfile to be created")
		}
		
		// Clean up test files
		os.RemoveAll(result.PackagePath)
	}
}