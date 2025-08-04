package o1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewYANGModelRegistry(t *testing.T) {
	registry := NewYANGModelRegistry()
	
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.models)
	assert.NotNil(t, registry.modelsByNS)
	assert.NotNil(t, registry.validators)
	assert.NotNil(t, registry.loadedModules)
	
	// Check that standard O-RAN models are loaded
	models := registry.ListModels()
	assert.Greater(t, len(models), 0)
	
	// Verify specific O-RAN models are present
	modelNames := make([]string, len(models))
	for i, model := range models {
		modelNames[i] = model.Name
	}
	
	expectedModels := []string{
		"o-ran-hardware",
		"o-ran-software-management",
		"o-ran-performance-management",
		"o-ran-fault-management",
		"ietf-interfaces",
	}
	
	for _, expectedModel := range expectedModels {
		assert.Contains(t, modelNames, expectedModel)
	}
}

func TestYANGModelRegistry_RegisterModel(t *testing.T) {
	registry := NewYANGModelRegistry()
	
	tests := []struct {
		name    string
		model   *YANGModel
		wantErr bool
	}{
		{
			name: "valid model",
			model: &YANGModel{
				Name:      "test-model",
				Namespace: "urn:test:model:1.0",
				Version:   "1.0",
				Revision:  "2024-01-15",
				Description: "Test YANG model",
				Schema:    make(map[string]interface{}),
			},
			wantErr: false,
		},
		{
			name: "model without name",
			model: &YANGModel{
				Namespace: "urn:test:model:1.0",
				Version:   "1.0",
			},
			wantErr: true,
		},
		{
			name: "model without namespace",
			model: &YANGModel{
				Name:    "test-model",
				Version: "1.0",
			},
			wantErr: true,
		},
		{
			name: "duplicate model with different version",
			model: &YANGModel{
				Name:      "o-ran-hardware", // Already exists
				Namespace: "urn:o-ran:hardware:2.0",
				Version:   "2.0", // Different version
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.RegisterModel(tt.model)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Verify model was registered
				retrievedModel, err := registry.GetModel(tt.model.Name)
				assert.NoError(t, err)
				assert.Equal(t, tt.model.Name, retrievedModel.Name)
				assert.Equal(t, tt.model.Namespace, retrievedModel.Namespace)
			}
		})
	}
}

func TestYANGModelRegistry_GetModel(t *testing.T) {
	registry := NewYANGModelRegistry()
	
	tests := []struct {
		name      string
		modelName string
		wantErr   bool
	}{
		{
			name:      "existing model",
			modelName: "o-ran-hardware",
			wantErr:   false,
		},
		{
			name:      "non-existing model",
			modelName: "non-existent-model",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := registry.GetModel(tt.modelName)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, model)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, model)
				assert.Equal(t, tt.modelName, model.Name)
			}
		})
	}
}

func TestYANGModelRegistry_GetModelByNamespace(t *testing.T) {
	registry := NewYANGModelRegistry()
	
	tests := []struct {
		name      string
		namespace string
		wantErr   bool
	}{
		{
			name:      "existing namespace",
			namespace: "urn:o-ran:hardware:1.0",
			wantErr:   false,
		},
		{
			name:      "non-existing namespace",
			namespace: "urn:non:existent:1.0",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := registry.GetModelByNamespace(tt.namespace)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, model)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, model)
				assert.Equal(t, tt.namespace, model.Namespace)
			}
		})
	}
}

func TestYANGModelRegistry_GetSchemaNode(t *testing.T) {
	registry := NewYANGModelRegistry()
	
	tests := []struct {
		name      string
		modelName string
		path      string
		wantErr   bool
	}{
		{
			name:      "valid path in hardware model",
			modelName: "o-ran-hardware",
			path:      "hardware",
			wantErr:   false,
		},
		{
			name:      "nested path in hardware model",
			modelName: "o-ran-hardware",
			path:      "hardware/component",
			wantErr:   false,
		},
		{
			name:      "invalid model",
			modelName: "non-existent-model",
			path:      "hardware",
			wantErr:   true,
		},
		{
			name:      "invalid path",
			modelName: "o-ran-hardware",
			path:      "non-existent-path",
			wantErr:   true,
		},
		{
			name:      "empty path",
			modelName: "o-ran-hardware",
			path:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := registry.GetSchemaNode(tt.modelName, tt.path)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, node)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, node)
				assert.NotEmpty(t, node.Name)
			}
		})
	}
}

func TestYANGModelRegistry_ValidateConfig(t *testing.T) {
	registry := NewYANGModelRegistry()
	ctx := context.Background()
	
	tests := []struct {
		name      string
		data      interface{}
		modelName string
		wantErr   bool
	}{
		{
			name: "valid hardware configuration",
			data: map[string]interface{}{
				"hardware": map[string]interface{}{
					"component": []interface{}{
						map[string]interface{}{
							"name":  "cpu-1",
							"class": "cpu",
							"state": map[string]interface{}{
								"name": "cpu-1",
							},
						},
					},
				},
			},
			modelName: "o-ran-hardware",
			wantErr:   false,
		},
		{
			name: "invalid configuration - missing mandatory field",
			data: map[string]interface{}{
				"hardware": map[string]interface{}{
					"component": []interface{}{
						map[string]interface{}{
							"class": "cpu", // Missing mandatory "name" field
						},
					},
				},
			},
			modelName: "o-ran-hardware",
			wantErr:   true,
		},
		{
			name:      "non-existent model",
			data:      map[string]interface{}{},
			modelName: "non-existent-model",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.ValidateConfig(ctx, tt.data, tt.modelName)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStandardYANGValidator_ValidateData(t *testing.T) {
	registry := NewYANGModelRegistry()
	validator := &StandardYANGValidator{registry: registry}
	
	tests := []struct {
		name      string
		data      interface{}
		modelName string
		wantErr   bool
	}{
		{
			name: "valid JSON string",
			data: `{"hardware": {"component": [{"name": "cpu-1", "class": "cpu"}]}}`,
			modelName: "o-ran-hardware",
			wantErr:   false,
		},
		{
			name: "valid map data",
			data: map[string]interface{}{
				"hardware": map[string]interface{}{
					"component": []interface{}{
						map[string]interface{}{
							"name":  "memory-1",
							"class": "memory",
						},
					},
				},
			},
			modelName: "o-ran-hardware",
			wantErr:   false,
		},
		{
			name:      "invalid JSON string",
			data:      `{invalid json}`,
			modelName: "o-ran-hardware",
			wantErr:   true,
		},
		{
			name:      "unsupported data type",
			data:      123, // Not a map or string
			modelName: "o-ran-hardware",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateData(tt.data, tt.modelName)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStandardYANGValidator_ValidateXPath(t *testing.T) {
	registry := NewYANGModelRegistry()
	validator := &StandardYANGValidator{registry: registry}
	
	tests := []struct {
		name      string
		xpath     string
		modelName string
		wantErr   bool
	}{
		{
			name:      "valid simple XPath",
			xpath:     "/hardware/component",
			modelName: "o-ran-hardware",
			wantErr:   false,
		},
		{
			name:      "valid XPath with predicate",
			xpath:     "/hardware/component[name='cpu-1']",
			modelName: "o-ran-hardware",
			wantErr:   false,
		},
		{
			name:      "empty XPath",
			xpath:     "",
			modelName: "o-ran-hardware",
			wantErr:   true,
		},
		{
			name:      "invalid XPath syntax",
			xpath:     "invalid xpath syntax",
			modelName: "o-ran-hardware",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateXPath(tt.xpath, tt.modelName)
			
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestYANGModelRegistry_GetStatistics(t *testing.T) {
	registry := NewYANGModelRegistry()
	
	stats := registry.GetStatistics()
	
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "total_models")
	assert.Contains(t, stats, "loaded_modules")
	assert.Contains(t, stats, "validators")
	assert.Contains(t, stats, "models")
	
	// Verify statistics values
	totalModels, ok := stats["total_models"].(int)
	assert.True(t, ok)
	assert.Greater(t, totalModels, 0)
	
	loadedModules, ok := stats["loaded_modules"].(int)
	assert.True(t, ok)
	assert.Equal(t, totalModels, loadedModules)
	
	validators, ok := stats["validators"].(int)
	assert.True(t, ok)
	assert.Greater(t, validators, 0)
	
	models, ok := stats["models"].(map[string]interface{})
	assert.True(t, ok)
	assert.Greater(t, len(models), 0)
	
	// Check individual model statistics
	for modelName, modelInfo := range models {
		modelInfoMap, ok := modelInfo.(map[string]interface{})
		assert.True(t, ok, "Model info should be a map for %s", modelName)
		
		assert.Contains(t, modelInfoMap, "version")
		assert.Contains(t, modelInfoMap, "revision")
		assert.Contains(t, modelInfoMap, "namespace")
		assert.Contains(t, modelInfoMap, "load_time")
		assert.Contains(t, modelInfoMap, "has_schema")
	}
}

func TestYANGNode(t *testing.T) {
	node := &YANGNode{
		Name:        "test-node",
		Type:        "container",
		Description: "Test node description",
		Mandatory:   true,
		Config:      true,
		Children:    make(map[string]*YANGNode),
	}
	
	assert.Equal(t, "test-node", node.Name)
	assert.Equal(t, "container", node.Type)
	assert.True(t, node.Mandatory)
	assert.True(t, node.Config)
	assert.NotNil(t, node.Children)
}

func TestConvertYANGChildren(t *testing.T) {
	children := map[string]*YANGNode{
		"child1": {
			Name: "child1",
			Type: "leaf",
		},
		"child2": {
			Name: "child2",
			Type: "container",
		},
	}
	
	result := convertYANGChildren(children)
	
	assert.Len(t, result, 2)
	assert.Contains(t, result, "child1")
	assert.Contains(t, result, "child2")
	
	child1, ok := result["child1"].(*YANGNode)
	assert.True(t, ok)
	assert.Equal(t, "child1", child1.Name)
	assert.Equal(t, "leaf", child1.Type)
}

// Benchmark tests for YANG validation performance
func BenchmarkYANGModelRegistry_ValidateConfig(b *testing.B) {
	registry := NewYANGModelRegistry()
	ctx := context.Background()
	
	data := map[string]interface{}{
		"hardware": map[string]interface{}{
			"component": []interface{}{
				map[string]interface{}{
					"name":  "cpu-1",
					"class": "cpu",
					"state": map[string]interface{}{
						"name": "cpu-1",
					},
				},
			},
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = registry.ValidateConfig(ctx, data, "o-ran-hardware")
	}
}

func BenchmarkStandardYANGValidator_ValidateXPath(b *testing.B) {
	registry := NewYANGModelRegistry()
	validator := &StandardYANGValidator{registry: registry}
	xpath := "/hardware/component[name='cpu-1']"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateXPath(xpath, "o-ran-hardware")
	}
}