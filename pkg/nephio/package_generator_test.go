package nephio

import (
	"strings"
	"testing"

	"github.com/thc1006/nephoran-intent-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGeneratePackage_IntentTypeRouting(t *testing.T) {
	tests := []struct {
		name          string
		intentType    string
		expectedFiles []string
		expectError   bool
		errorContains string
	}{
		{
			name:       "deployment intent type",
			intentType: "deployment",
			expectedFiles: []string{
				"packages/test-namespace/test-intent-package/Kptfile",
				"packages/test-namespace/test-intent-package/deployment.yaml",
				"packages/test-namespace/test-intent-package/service.yaml",
				"packages/test-namespace/test-intent-package/setters.yaml",
				"packages/test-namespace/test-intent-package/README.md",
				"packages/test-namespace/test-intent-package/fn-config.yaml",
			},
			expectError: false,
		},
		{
			name:       "scaling intent type",
			intentType: "scaling",
			expectedFiles: []string{
				"packages/test-namespace/test-intent-package/Kptfile",
				"packages/test-namespace/test-intent-package/scaling-patch.yaml",
				"packages/test-namespace/test-intent-package/setters.yaml",
				"packages/test-namespace/test-intent-package/README.md",
				"packages/test-namespace/test-intent-package/fn-config.yaml",
			},
			expectError: false,
		},
		{
			name:       "policy intent type",
			intentType: "policy",
			expectedFiles: []string{
				"packages/test-namespace/test-intent-package/Kptfile",
				"packages/test-namespace/test-intent-package/policy.yaml",
				"packages/test-namespace/test-intent-package/README.md",
				"packages/test-namespace/test-intent-package/fn-config.yaml",
			},
			expectError: false,
		},
		{
			name:       "empty intent type defaults to deployment",
			intentType: "",
			expectedFiles: []string{
				"packages/test-namespace/test-intent-package/Kptfile",
				"packages/test-namespace/test-intent-package/deployment.yaml",
				"packages/test-namespace/test-intent-package/service.yaml",
				"packages/test-namespace/test-intent-package/setters.yaml",
				"packages/test-namespace/test-intent-package/README.md",
				"packages/test-namespace/test-intent-package/fn-config.yaml",
			},
			expectError: false,
		},
		{
			name:          "unsupported intent type",
			intentType:    "unsupported",
			expectError:   true,
			errorContains: "unsupported intent type: unsupported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a package generator
			pg, err := NewPackageGenerator()
			if err != nil {
				t.Fatalf("Failed to create package generator: %v", err)
			}

			// Create test parameters based on intent type
			_ = createTestParameters(tt.intentType)

			// Create a test NetworkIntent
			intent := &v1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-intent",
					Namespace: "test-namespace",
				},
				Spec: v1.NetworkIntentSpec{
					Intent:     "Test intent for " + tt.intentType,
					IntentType: v1.IntentType(tt.intentType),
				},
			}

			// Generate the package
			files, err := pg.GeneratePackage(intent)

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check that expected files were generated
			for _, expectedFile := range tt.expectedFiles {
				if _, exists := files[expectedFile]; !exists {
					t.Errorf("Expected file %s was not generated", expectedFile)
				}
			}

			// Verify file contents contain expected elements
			if kptfile, exists := files["packages/test-namespace/test-intent-package/Kptfile"]; exists {
				if !strings.Contains(kptfile, "test-intent-package") {
					t.Errorf("Kptfile does not contain expected package name")
				}
			}
		})
	}
}

func TestGeneratePackage_DefaultsToDeployment(t *testing.T) {
	// Create a package generator
	pg, err := NewPackageGenerator()
	if err != nil {
		t.Fatalf("Failed to create package generator: %v", err)
	}

	// Create test parameters for deployment
	_ = createTestParameters("deployment")

	// Create a NetworkIntent without IntentType specified
	intent := &v1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-intent",
			Namespace: "test-namespace",
		},
		Spec: v1.NetworkIntentSpec{
			Intent: "Deploy a network function",
			// IntentType is intentionally not set
		},
	}

	// Generate the package
	files, err := pg.GeneratePackage(intent)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify deployment files were generated (default behavior)
	deploymentFile := "packages/test-namespace/test-intent-package/deployment.yaml"
	if _, exists := files[deploymentFile]; !exists {
		t.Errorf("Expected deployment.yaml to be generated when IntentType is empty")
	}

	serviceFile := "packages/test-namespace/test-intent-package/service.yaml"
	if _, exists := files[serviceFile]; !exists {
		t.Errorf("Expected service.yaml to be generated when IntentType is empty")
	}
}

// Helper function to create test parameters based on intent type
func createTestParameters(intentType string) map[string]interface{} {
	baseParams := map[string]interface{}{
		"name":      "test-app",
		"namespace": "test-namespace",
		"component": "test-component",
		"intent_id": "test-intent",
	}

	switch intentType {
	case "deployment", "":
		baseParams["replicas"] = 3
		baseParams["image"] = "test-image:latest"
		baseParams["ports"] = []map[string]interface{}{
			{
				"Name":     "http",
				"Port":     8080,
				"Protocol": "TCP",
			},
		}
		baseParams["env"] = []map[string]interface{}{
			{
				"Name":  "ENV_VAR",
				"Value": "test-value",
			},
		}
		baseParams["resources"] = map[string]interface{}{
			"Requests": map[string]interface{}{
				"CPU":    "100m",
				"Memory": "128Mi",
			},
			"Limits": map[string]interface{}{
				"CPU":    "200m",
				"Memory": "256Mi",
			},
		}
	case "scaling":
		baseParams["target"] = "existing-deployment"
		baseParams["replicas"] = 5
	case "policy":
		baseParams["policy_spec"] = map[string]interface{}{
			"podSelector": map[string]interface{}{
				"matchLabels": map[string]string{
					"app": "test-app",
				},
			},
			"policyTypes": []string{"Ingress", "Egress"},
		}
		baseParams["a1_policy"] = map[string]interface{}{
			"policy_type_id":     20000,
			"policy_instance_id": "slice-policy-001",
			"policy_data": map[string]interface{}{
				"scope":  "network-slice",
				"target": "slice-001",
			},
		}
	}

	return baseParams
}
