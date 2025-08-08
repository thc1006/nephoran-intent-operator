/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package porch

import (
	"testing"
)

func TestDefaultPorchConfig(t *testing.T) {
	config := DefaultPorchConfig()
	
	if config == nil {
		t.Fatal("Expected config but got nil")
	}
	
	if config.PorchConfig == nil {
		t.Fatal("Expected PorchConfig but got nil")
	}
	
	if config.PorchConfig.DefaultNamespace != "default" {
		t.Errorf("Expected default namespace 'default' but got '%s'", config.PorchConfig.DefaultNamespace)
	}
	
	if config.PorchConfig.DefaultRepository != "default" {
		t.Errorf("Expected default repository 'default' but got '%s'", config.PorchConfig.DefaultRepository)
	}
	
	if config.PorchConfig.CircuitBreaker == nil {
		t.Fatal("Expected CircuitBreaker config but got nil")
	}
	
	if !config.PorchConfig.CircuitBreaker.Enabled {
		t.Error("Expected CircuitBreaker to be enabled")
	}
	
	if config.PorchConfig.RateLimit == nil {
		t.Fatal("Expected RateLimit config but got nil")
	}
	
	if !config.PorchConfig.RateLimit.Enabled {
		t.Error("Expected RateLimit to be enabled")
	}
}

func TestIsValidLifecycle(t *testing.T) {
	tests := []struct {
		lifecycle PackageRevisionLifecycle
		expected  bool
	}{
		{PackageRevisionLifecycleDraft, true},
		{PackageRevisionLifecycleProposed, true},
		{PackageRevisionLifecyclePublished, true},
		{PackageRevisionLifecycleDeletable, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.lifecycle), func(t *testing.T) {
			result := IsValidLifecycle(tt.lifecycle)
			if result != tt.expected {
				t.Errorf("IsValidLifecycle(%s) = %v, want %v", tt.lifecycle, result, tt.expected)
			}
		})
	}
}

func TestPackageRevisionLifecycle_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name     string
		from     PackageRevisionLifecycle
		to       PackageRevisionLifecycle
		expected bool
	}{
		{
			name:     "draft to proposed",
			from:     PackageRevisionLifecycleDraft,
			to:       PackageRevisionLifecycleProposed,
			expected: true,
		},
		{
			name:     "proposed to published",
			from:     PackageRevisionLifecycleProposed,
			to:       PackageRevisionLifecyclePublished,
			expected: true,
		},
		{
			name:     "published to draft - not allowed",
			from:     PackageRevisionLifecyclePublished,
			to:       PackageRevisionLifecycleDraft,
			expected: false,
		},
		{
			name:     "draft directly to published - not allowed",
			from:     PackageRevisionLifecycleDraft,
			to:       PackageRevisionLifecyclePublished,
			expected: false,
		},
		{
			name:     "proposed back to draft",
			from:     PackageRevisionLifecycleProposed,
			to:       PackageRevisionLifecycleDraft,
			expected: true,
		},
		{
			name:     "any to deletable",
			from:     PackageRevisionLifecyclePublished,
			to:       PackageRevisionLifecycleDeletable,
			expected: true,
		},
		{
			name:     "from deletable - not allowed",
			from:     PackageRevisionLifecycleDeletable,
			to:       PackageRevisionLifecycleDraft,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			if result != tt.expected {
				t.Errorf("CanTransitionTo() = %v, want %v for %s -> %s", 
					result, tt.expected, tt.from, tt.to)
			}
		})
	}
}

func TestGetPackageReference(t *testing.T) {
	repository := "test-repo"
	packageName := "test-package"
	revision := "v1.0.0"
	
	ref := GetPackageReference(repository, packageName, revision)
	
	if ref == nil {
		t.Fatal("Expected PackageReference but got nil")
	}
	
	if ref.Repository != repository {
		t.Errorf("Expected repository '%s' but got '%s'", repository, ref.Repository)
	}
	
	if ref.PackageName != packageName {
		t.Errorf("Expected packageName '%s' but got '%s'", packageName, ref.PackageName)
	}
	
	if ref.Revision != revision {
		t.Errorf("Expected revision '%s' but got '%s'", revision, ref.Revision)
	}
}

func TestPackageReference_GetPackageKey(t *testing.T) {
	ref := &PackageReference{
		Repository:  "test-repo",
		PackageName: "test-package",
		Revision:    "v1.0.0",
	}
	
	expected := "test-repo/test-package@v1.0.0"
	result := ref.GetPackageKey()
	
	if result != expected {
		t.Errorf("GetPackageKey() = '%s', want '%s'", result, expected)
	}
}

func TestConvertKRMResourceToYAML(t *testing.T) {
	resource := KRMResource{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Metadata: map[string]interface{}{
			"name":      "test-config",
			"namespace": "default",
		},
		Data: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}

	yamlData, err := convertKRMResourceToYAML(resource)
	if err != nil {
		t.Fatalf("convertKRMResourceToYAML() error = %v", err)
	}
	
	if len(yamlData) == 0 {
		t.Error("Expected non-empty YAML data")
	}
}

func TestConvertYAMLToKRMResource(t *testing.T) {
	yamlData := []byte(`{
		"apiVersion": "v1",
		"kind": "ConfigMap",
		"metadata": {
			"name": "test-config",
			"namespace": "default"
		},
		"data": {
			"key1": "value1"
		}
	}`)

	resource, err := convertYAMLToKRMResource(yamlData)
	if err != nil {
		t.Fatalf("convertYAMLToKRMResource() error = %v", err)
	}
	
	if resource == nil {
		t.Fatal("Expected resource but got nil")
	}
	
	if resource.APIVersion != "v1" {
		t.Errorf("Expected APIVersion 'v1' but got '%s'", resource.APIVersion)
	}
	
	if resource.Kind != "ConfigMap" {
		t.Errorf("Expected Kind 'ConfigMap' but got '%s'", resource.Kind)
	}
	
	if name, ok := resource.Metadata["name"]; !ok || name != "test-config" {
		t.Errorf("Expected metadata name 'test-config' but got '%v'", name)
	}
}

func TestGenerateResourceFilename(t *testing.T) {
	tests := []struct {
		name     string
		resource KRMResource
		expected string
	}{
		{
			name: "configmap with name",
			resource: KRMResource{
				Kind: "ConfigMap",
				Metadata: map[string]interface{}{
					"name": "test-config",
				},
			},
			expected: "configmap-test-config.yaml",
		},
		{
			name: "deployment with name",
			resource: KRMResource{
				Kind: "Deployment",
				Metadata: map[string]interface{}{
					"name": "my-app",
				},
			},
			expected: "deployment-my-app.yaml",
		},
		{
			name: "resource without name",
			resource: KRMResource{
				Kind: "Service",
				Metadata: map[string]interface{}{},
			},
			expected: "service-resource.yaml",
		},
		{
			name: "resource without kind",
			resource: KRMResource{
				Metadata: map[string]interface{}{
					"name": "test",
				},
			},
			expected: "resource-test.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateResourceFilename(tt.resource)
			if result != tt.expected {
				t.Errorf("generateResourceFilename() = '%s', want '%s'", result, tt.expected)
			}
		})
	}
}

func TestIsYAMLFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"config.yaml", true},
		{"config.yml", true},
		{"CONFIG.YAML", true},
		{"Config.YML", true},
		{"config.json", false},
		{"config.txt", false},
		{"config", false},
		{"yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isYAMLFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isYAMLFile('%s') = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

// Benchmark tests

func BenchmarkConvertKRMResourceToYAML(b *testing.B) {
	resource := KRMResource{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Metadata: map[string]interface{}{
			"name":      "test-config",
			"namespace": "default",
			"labels": map[string]interface{}{
				"app": "test",
				"env": "prod",
			},
		},
		Data: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = convertKRMResourceToYAML(resource)
	}
}

func BenchmarkGenerateResourceFilename(b *testing.B) {
	resource := KRMResource{
		Kind: "Deployment",
		Metadata: map[string]interface{}{
			"name": "my-application-deployment",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateResourceFilename(resource)
	}
}

func BenchmarkIsValidLifecycle(b *testing.B) {
	lifecycles := []PackageRevisionLifecycle{
		PackageRevisionLifecycleDraft,
		PackageRevisionLifecycleProposed,
		PackageRevisionLifecyclePublished,
		PackageRevisionLifecycleDeletable,
		"invalid",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsValidLifecycle(lifecycles[i%len(lifecycles)])
	}
}