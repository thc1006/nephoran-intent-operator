package modeladapter

import (
	"encoding/json"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

func TestFromGenerated(t *testing.T) {
	tests := []struct {
		name     string
		input    *models.ResourceType
		expected *InternalResourceType
	}{
		{
			name: "complete resource type",
			input: &models.ResourceType{
				ResourceTypeID: "test-compute",
				Name:           "Test Compute",
				Description:    "Test compute resource type",
				Vendor:         "TestVendor",
				Model:          "TestModel",
				Version:        "v1.0",
				Category:       models.ResourceCategoryCompute,
				ResourceClass:  models.ResourceClassVirtual,
				ResourceKind:   models.ResourceKindServer,
				SupportedOperations: []string{"create", "update", "delete"},
				Status:    models.ResourceTypeStatusActive,
				CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				CreatedBy: "test-user",
				UpdatedBy: "test-user",
			},
			expected: &InternalResourceType{
				ResourceTypeID: "test-compute",
				Name:           "Test Compute",
				Description:    "Test compute resource type",
				Vendor:         "TestVendor",
				Model:          "TestModel",
				Version:        "v1.0",
				Specifications: &InternalResourceTypeSpec{
					Category:      models.ResourceCategoryCompute,
					ResourceClass: models.ResourceClassVirtual,
					ResourceKind:  models.ResourceKindServer,
					Properties:    map[string]interface{}{},
				},
				SupportedActions: []string{"create", "update", "delete"},
				Status:           models.ResourceTypeStatusActive,
				CreatedAt:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:        time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				CreatedBy:        "test-user",
				UpdatedBy:        "test-user",
			},
		},
		{
			name: "resource type with capabilities and features",
			input: &models.ResourceType{
				ResourceTypeID: "test-network",
				Name:           "Test Network",
				Category:       models.ResourceCategoryNetwork,
				Capabilities: []models.ResourceCapability{
					{Name: "load-balancing", Type: models.CapabilityTypeFunctional},
					{Name: "high-availability", Type: models.CapabilityTypeOperational},
				},
				Features: []models.ResourceFeature{
					{Name: "ssl-termination", FeatureType: models.FeatureTypeStandard},
					{Name: "custom-routing", FeatureType: models.FeatureTypeVendorSpecific},
				},
				Status:    models.ResourceTypeStatusActive,
				CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: &InternalResourceType{
				ResourceTypeID: "test-network",
				Name:           "Test Network",
				Specifications: &InternalResourceTypeSpec{
					Category:     models.ResourceCategoryNetwork,
					Capabilities: []string{"load-balancing", "high-availability"},
					Features:     []string{"ssl-termination", "custom-routing"},
					Properties:   map[string]interface{}{},
				},
				Status:    models.ResourceTypeStatusActive,
				CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "resource type with resource limits",
			input: &models.ResourceType{
				ResourceTypeID: "test-storage",
				Name:           "Test Storage",
				Category:       models.ResourceCategoryStorage,
				ResourceLimits: &models.ResourceLimits{
					CPULimits: &models.ResourceLimit{
						MinValue:     "100m",
						DefaultValue: "500m",
						MaxValue:     "2",
						Unit:         "cores",
					},
					MemoryLimits: &models.ResourceLimit{
						MinValue:     "128Mi",
						DefaultValue: "512Mi",
						MaxValue:     "4Gi",
						Unit:         "bytes",
					},
				},
				Status:    models.ResourceTypeStatusActive,
				CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expected: &InternalResourceType{
				ResourceTypeID: "test-storage",
				Name:           "Test Storage",
				Specifications: &InternalResourceTypeSpec{
					Category: models.ResourceCategoryStorage,
					MinResources: map[string]string{
						"cpu":    "100m",
						"memory": "128Mi",
					},
					DefaultResources: map[string]string{
						"cpu":    "500m",
						"memory": "512Mi",
					},
					MaxResources: map[string]string{
						"cpu":    "2",
						"memory": "4Gi",
					},
					Properties: map[string]interface{}{},
				},
				Status:    models.ResourceTypeStatusActive,
				CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FromGenerated(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("expected non-nil result")
			}

			// Check core fields
			if result.ResourceTypeID != tt.expected.ResourceTypeID {
				t.Errorf("ResourceTypeID: expected %s, got %s", tt.expected.ResourceTypeID, result.ResourceTypeID)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("Name: expected %s, got %s", tt.expected.Name, result.Name)
			}
			if result.Description != tt.expected.Description {
				t.Errorf("Description: expected %s, got %s", tt.expected.Description, result.Description)
			}
			if result.Status != tt.expected.Status {
				t.Errorf("Status: expected %s, got %s", tt.expected.Status, result.Status)
			}

			// Check specifications
			if result.Specifications == nil && tt.expected.Specifications != nil {
				t.Errorf("Specifications: expected non-nil, got nil")
				return
			}
			if result.Specifications != nil && tt.expected.Specifications == nil {
				t.Errorf("Specifications: expected nil, got non-nil")
				return
			}

			if result.Specifications != nil {
				if result.Specifications.Category != tt.expected.Specifications.Category {
					t.Errorf("Category: expected %s, got %s", tt.expected.Specifications.Category, result.Specifications.Category)
				}

				// Check capabilities
				if len(result.Specifications.Capabilities) != len(tt.expected.Specifications.Capabilities) {
					t.Errorf("Capabilities length: expected %d, got %d", len(tt.expected.Specifications.Capabilities), len(result.Specifications.Capabilities))
				} else {
					for i, cap := range tt.expected.Specifications.Capabilities {
						if result.Specifications.Capabilities[i] != cap {
							t.Errorf("Capability[%d]: expected %s, got %s", i, cap, result.Specifications.Capabilities[i])
						}
					}
				}

				// Check features
				if len(result.Specifications.Features) != len(tt.expected.Specifications.Features) {
					t.Errorf("Features length: expected %d, got %d", len(tt.expected.Specifications.Features), len(result.Specifications.Features))
				} else {
					for i, feature := range tt.expected.Specifications.Features {
						if result.Specifications.Features[i] != feature {
							t.Errorf("Feature[%d]: expected %s, got %s", i, feature, result.Specifications.Features[i])
						}
					}
				}
			}
		})
	}
}

func TestToGenerated(t *testing.T) {
	tests := []struct {
		name     string
		input    *InternalResourceType
		expected *models.ResourceType
	}{
		{
			name: "complete internal type",
			input: &InternalResourceType{
				ResourceTypeID: "test-compute",
				Name:           "Test Compute",
				Description:    "Test compute resource type",
				Vendor:         "TestVendor",
				Model:          "TestModel",
				Version:        "v1.0",
				Specifications: &InternalResourceTypeSpec{
					Category:      models.ResourceCategoryCompute,
					ResourceClass: models.ResourceClassVirtual,
					ResourceKind:  models.ResourceKindServer,
					Properties: map[string]interface{}{
						"testKey": "testValue",
					},
				},
				SupportedActions: []string{"create", "update", "delete"},
				Status:           models.ResourceTypeStatusActive,
				CreatedAt:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:        time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				CreatedBy:        "test-user",
				UpdatedBy:        "test-user",
			},
			expected: &models.ResourceType{
				ResourceTypeID:      "test-compute",
				Name:                "Test Compute",
				Description:         "Test compute resource type",
				Vendor:              "TestVendor",
				Model:               "TestModel",
				Version:             "v1.0",
				Category:            models.ResourceCategoryCompute,
				ResourceClass:       models.ResourceClassVirtual,
				ResourceKind:        models.ResourceKindServer,
				SupportedOperations: []string{"create", "update", "delete"},
				Extensions: map[string]interface{}{
					"testKey": "testValue",
				},
				Status:    models.ResourceTypeStatusActive,
				CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				CreatedBy: "test-user",
				UpdatedBy: "test-user",
			},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.ToGenerated()

			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("expected non-nil result")
			}

			// Check core fields
			if result.ResourceTypeID != tt.expected.ResourceTypeID {
				t.Errorf("ResourceTypeID: expected %s, got %s", tt.expected.ResourceTypeID, result.ResourceTypeID)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("Name: expected %s, got %s", tt.expected.Name, result.Name)
			}
			if result.Category != tt.expected.Category {
				t.Errorf("Category: expected %s, got %s", tt.expected.Category, result.Category)
			}
			if result.Status != tt.expected.Status {
				t.Errorf("Status: expected %s, got %s", tt.expected.Status, result.Status)
			}

			// Check supported operations
			if len(result.SupportedOperations) != len(tt.expected.SupportedOperations) {
				t.Errorf("SupportedOperations length: expected %d, got %d", len(tt.expected.SupportedOperations), len(result.SupportedOperations))
			} else {
				for i, op := range tt.expected.SupportedOperations {
					if result.SupportedOperations[i] != op {
						t.Errorf("SupportedOperations[%d]: expected %s, got %s", i, op, result.SupportedOperations[i])
					}
				}
			}

			// Check extensions
			if tt.expected.Extensions != nil {
				if result.Extensions == nil {
					t.Errorf("Extensions: expected non-nil, got nil")
				} else {
					for key, expectedValue := range tt.expected.Extensions {
						if actualValue, ok := result.Extensions[key]; !ok {
							t.Errorf("Extensions[%s]: key missing", key)
						} else if actualValue != expectedValue {
							t.Errorf("Extensions[%s]: expected %v, got %v", key, expectedValue, actualValue)
						}
					}
				}
			}
		})
	}
}

func TestDefaultResourceTypes(t *testing.T) {
	tests := []struct {
		name     string
		factory  func() *InternalResourceType
		category string
	}{
		{
			name:     "default compute",
			factory:  CreateDefaultComputeResourceType,
			category: models.ResourceCategoryCompute,
		},
		{
			name:     "default network",
			factory:  CreateDefaultNetworkResourceType,
			category: models.ResourceCategoryNetwork,
		},
		{
			name:     "default storage",
			factory:  CreateDefaultStorageResourceType,
			category: models.ResourceCategoryStorage,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceType := tt.factory()

			if resourceType == nil {
				t.Fatalf("factory returned nil")
			}

			if resourceType.ResourceTypeID == "" {
				t.Errorf("ResourceTypeID is empty")
			}

			if resourceType.Name == "" {
				t.Errorf("Name is empty")
			}

			if resourceType.Specifications == nil {
				t.Fatalf("Specifications is nil")
			}

			if resourceType.Specifications.Category != tt.category {
				t.Errorf("Category: expected %s, got %s", tt.category, resourceType.Specifications.Category)
			}

			if len(resourceType.SupportedActions) == 0 {
				t.Errorf("SupportedActions is empty")
			}

			if resourceType.Status != models.ResourceTypeStatusActive {
				t.Errorf("Status: expected %s, got %s", models.ResourceTypeStatusActive, resourceType.Status)
			}

			// Test round-trip conversion
			generated := resourceType.ToGenerated()
			if generated == nil {
				t.Fatalf("ToGenerated returned nil")
			}

			roundTrip := FromGenerated(generated)
			if roundTrip == nil {
				t.Fatalf("FromGenerated returned nil")
			}

			if roundTrip.ResourceTypeID != resourceType.ResourceTypeID {
				t.Errorf("Round-trip ResourceTypeID: expected %s, got %s", resourceType.ResourceTypeID, roundTrip.ResourceTypeID)
			}
		})
	}
}

func TestJSONCompatibility(t *testing.T) {
	// Test that our adapter can handle JSON that might come from the current O-RAN model
	testJSON := `{
		"resourceTypeId": "test-json-type",
		"name": "Test JSON Type",
		"description": "A test resource type from JSON",
		"vendor": "TestVendor",
		"version": "v1.0",
		"category": "COMPUTE",
		"resourceClass": "VIRTUAL",
		"supportedOperations": ["create", "delete"],
		"capabilities": [
			{"name": "cpu-intensive", "type": "PERFORMANCE"},
			{"name": "auto-scaling", "type": "FUNCTIONAL"}
		],
		"resourceLimits": {
			"cpuLimits": {
				"minValue": "100m",
				"defaultValue": "500m",
				"maxValue": "4",
				"unit": "cores"
			}
		},
		"status": "ACTIVE",
		"createdAt": "2023-01-01T00:00:00Z",
		"updatedAt": "2023-01-01T00:00:00Z"
	}`

	var generated models.ResourceType
	err := json.Unmarshal([]byte(testJSON), &generated)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Test conversion through adapter
	internal := FromGenerated(&generated)
	if internal == nil {
		t.Fatalf("FromGenerated returned nil")
	}

	// Verify key fields are preserved
	if internal.ResourceTypeID != "test-json-type" {
		t.Errorf("ResourceTypeID: expected test-json-type, got %s", internal.ResourceTypeID)
	}

	if internal.Specifications == nil {
		t.Fatalf("Specifications is nil")
	}

	if internal.Specifications.Category != models.ResourceCategoryCompute {
		t.Errorf("Category: expected %s, got %s", models.ResourceCategoryCompute, internal.Specifications.Category)
	}

	if len(internal.Specifications.Capabilities) != 2 {
		t.Errorf("Capabilities: expected 2, got %d", len(internal.Specifications.Capabilities))
	}

	if internal.Specifications.MinResources["cpu"] != "100m" {
		t.Errorf("MinResources CPU: expected 100m, got %s", internal.Specifications.MinResources["cpu"])
	}

	// Test round-trip back to generated model
	backToGenerated := internal.ToGenerated()
	if backToGenerated == nil {
		t.Fatalf("ToGenerated returned nil")
	}

	if backToGenerated.ResourceTypeID != "test-json-type" {
		t.Errorf("Round-trip ResourceTypeID: expected test-json-type, got %s", backToGenerated.ResourceTypeID)
	}
}