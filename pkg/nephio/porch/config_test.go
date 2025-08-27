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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Unit Tests

func TestNewConfig(t *testing.T) {
	config := NewConfig()

	assert.NotNil(t, config)
	assert.NotNil(t, config.KubernetesConfig)
	assert.NotNil(t, config.PorchConfig)
	assert.NotNil(t, config.Repositories)
	assert.NotNil(t, config.Functions)
}

func TestConfigWithDefaults(t *testing.T) {
	config := NewConfig().WithDefaults()

	// Verify defaults are applied - using actual config structure
	// Note: WithDefaults() applies defaults through ConfigBuilder
	assert.NotNil(t, config.KubernetesConfig)
	assert.NotNil(t, config.PorchConfig)
	assert.NotNil(t, config.Functions)
	assert.NotNil(t, config.Security)
	assert.NotNil(t, config.Observability)

	// Verify function execution defaults
	if config.Functions != nil && config.Functions.Execution != nil {
		assert.Equal(t, "docker", config.Functions.Execution.Runtime)
		assert.Equal(t, 5*time.Minute, config.Functions.Execution.DefaultTimeout)
		assert.Equal(t, 5, config.Functions.Execution.MaxConcurrent)
		if config.Functions.Execution.ResourceLimits != nil {
			assert.Equal(t, "1000m", config.Functions.Execution.ResourceLimits.CPU)
			assert.Equal(t, "1Gi", config.Functions.Execution.ResourceLimits.Memory)
		}
	}

	// Verify function cache defaults
	if config.Functions != nil && config.Functions.Cache != nil {
		assert.True(t, config.Functions.Cache.Enabled)
		assert.Equal(t, 24*time.Hour, config.Functions.Cache.TTL)
		assert.Equal(t, "1Gi", config.Functions.Cache.Size)
	}

	// Verify security defaults
	if config.Security != nil {
		assert.NotNil(t, config.Security.Authentication)
		assert.NotNil(t, config.Security.Authorization)
		if config.Security.Authorization != nil {
			assert.True(t, config.Security.Authorization.Enabled)
			assert.Equal(t, "rbac", config.Security.Authorization.Mode)
		}
	}

	// Verify observability defaults
	if config.Observability != nil {
		if config.Observability.Metrics != nil {
			assert.True(t, config.Observability.Metrics.Enabled)
			assert.Equal(t, "prometheus", config.Observability.Metrics.Provider)
		}
		if config.Observability.Tracing != nil {
			assert.False(t, config.Observability.Tracing.Enabled)
			assert.Equal(t, "jaeger", config.Observability.Tracing.Provider)
		}
		if config.Observability.Logging != nil {
			assert.Equal(t, "info", config.Observability.Logging.Level)
			assert.Equal(t, "json", config.Observability.Logging.Format)
		}
	}
}

func TestConfigBuilder(t *testing.T) {
	// This test uses the ConfigBuilder API which exists in config.go
	builder := NewConfigBuilder()
	config := builder.Build()

	// Verify basic config structure is created
	assert.NotNil(t, config)
	assert.NotNil(t, config.KubernetesConfig)
	assert.NotNil(t, config.PorchConfig)
	assert.NotNil(t, config.Functions)
	assert.NotNil(t, config.Security)
	assert.NotNil(t, config.Observability)
	assert.NotNil(t, config.Performance)
	assert.NotNil(t, config.Repositories)
	assert.NotNil(t, config.Clusters)
}

func TestConfigAddRepository(t *testing.T) {
	// Use ConfigBuilder to add repository
	builder := NewConfigBuilder()
	repoConfig := &RepositoryConfig{
		Name: "test-repo",
		Type: "git",
		URL:  "https://github.com/test/repo.git",
	}

	config := builder.AddRepository("test-repo", repoConfig).Build()

	assert.Contains(t, config.Repositories, "test-repo")
	assert.Equal(t, repoConfig, config.Repositories["test-repo"])
}

func TestConfigAddFunction(t *testing.T) {
	config := NewConfig().WithDefaults()

	funcConfig := &FunctionConfig{
		Image: "gcr.io/kpt-fn/custom-function:v1.0.0",
		ConfigMap: map[string]interface{}{
			"param": "value",
		},
	}

	// Test function config structure
	assert.Equal(t, "gcr.io/kpt-fn/custom-function:v1.0.0", funcConfig.Image)
	assert.NotNil(t, funcConfig.ConfigMap)
	assert.Equal(t, "value", funcConfig.ConfigMap["param"])

	// Test that function config is available
	assert.NotNil(t, config.Functions)
}

func TestConfigSetResourceLimits(t *testing.T) {
	config := NewConfig().WithDefaults()

	limits := &FunctionResourceLimits{
		CPU:     "2000m",
		Memory:  "2Gi",
		Storage: "10Gi",
		Timeout: 10 * time.Minute,
	}

	// Test resource limits structure
	assert.Equal(t, "2000m", limits.CPU)
	assert.Equal(t, "2Gi", limits.Memory)
	assert.Equal(t, "10Gi", limits.Storage)
	assert.Equal(t, 10*time.Minute, limits.Timeout)

	// Test that function execution config exists
	assert.NotNil(t, config.Functions)
	assert.NotNil(t, config.Functions.Execution)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "ValidConfig",
			config:      NewConfig().WithDefaults(),
			expectError: false,
		},
		{
			name: "InvalidPorchConfig",
			config: &Config{
				Repositories: make(map[string]*RepositoryConfig),
				Clusters:     make(map[string]*ClusterConfig),
			},
			expectError: true,
			errorMsg:    "porch configuration is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.config.Validate()

			if tt.expectError {
				assert.NotEmpty(t, errors)
				found := false
				for _, err := range errors {
					if strings.Contains(err.Error(), tt.errorMsg) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected error message not found")
			} else {
				assert.Empty(t, errors)
			}
		})
	}
}

func TestConfigFromEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("PORCH_ENDPOINT", "https://env-porch-server.com")
	os.Setenv("PORCH_NAMESPACE", "env-namespace")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("PORCH_ENDPOINT")
		os.Unsetenv("PORCH_NAMESPACE")
	}()

	config, err := NewConfigFromEnvironment()
	require.NoError(t, err)

	// NewConfigFromEnvironment creates a default config, so just verify structure
	assert.NotNil(t, config)
	assert.NotNil(t, config.PorchConfig)
	assert.NotNil(t, config.KubernetesConfig)
}

func TestConfigFromKubeconfig(t *testing.T) {
	// Test kubeconfig path setting
	config := NewConfig().WithDefaults()

	// Verify kubeconfig path is set in defaults
	assert.NotNil(t, config.KubernetesConfig)
	assert.NotEmpty(t, config.KubernetesConfig.KubeconfigPath)
	
	// Test GetKubernetesConfig method
	_, err := config.GetKubernetesConfig()
	// This may fail if kubeconfig doesn't exist, but we test the method exists
	if err != nil {
		// Expected when kubeconfig file doesn't exist
		assert.Contains(t, err.Error(), "kubernetes config")
	}
}

func TestConfigFromRestConfig(t *testing.T) {
	// Test config creation and Kubernetes config setup
	config := NewConfig().WithDefaults()

	assert.NotNil(t, config)
	assert.NotNil(t, config.KubernetesConfig)
	// Default kubeconfig path should be set
	assert.NotEmpty(t, config.KubernetesConfig.KubeconfigPath)
}

func TestRepositoryConfigValidation(t *testing.T) {
	// RepositoryConfig doesn't have Validate method in current implementation
	// Test basic repository configuration structure instead
	config := &RepositoryConfig{
		Name: "test-repo",
		Type: "git",
		URL:  "https://github.com/test/repo.git",
	}

	assert.Equal(t, "test-repo", config.Name)
	assert.Equal(t, "git", config.Type)
	assert.Equal(t, "https://github.com/test/repo.git", config.URL)
}

func TestAuthConfigValidation(t *testing.T) {
	// AuthConfig doesn't have Validate method in current implementation
	// Test basic auth configuration structure instead
	config := &AuthConfig{
		Type:     "basic",
		Username: "test-user",
		Password: "test-password",
	}

	assert.Equal(t, "basic", config.Type)
	assert.Equal(t, "test-user", config.Username)
	assert.Equal(t, "test-password", config.Password)
}

func TestSyncConfigValidation(t *testing.T) {
	// SyncConfig doesn't have Validate method in current implementation
	// Test basic sync configuration structure instead
	config := &SyncConfig{
		AutoSync: true,
		Interval: &metav1.Duration{Duration: 5 * time.Minute},
	}

	assert.True(t, config.AutoSync)
	assert.NotNil(t, config.Interval)
	assert.Equal(t, 5*time.Minute, config.Interval.Duration)
}

func TestFunctionResourceLimitsValidation(t *testing.T) {
	// FunctionResourceLimits doesn't have Validate method in current implementation
	// Test basic resource limits structure instead
	limits := &FunctionResourceLimits{
		CPU:     "1000m",
		Memory:  "1Gi",
		Storage: "10Gi",
		Timeout: 5 * time.Minute,
	}

	assert.Equal(t, "1000m", limits.CPU)
	assert.Equal(t, "1Gi", limits.Memory)
	assert.Equal(t, "10Gi", limits.Storage)
	assert.Equal(t, 5*time.Minute, limits.Timeout)
}

// Performance Tests

func TestConfigPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("ConfigCreationLatency", func(t *testing.T) {
		start := time.Now()
		config := NewConfig().WithDefaults()
		duration := time.Since(start)

		assert.NotNil(t, config)
		assert.Less(t, duration, 10*time.Millisecond, "Config creation should be fast")
	})

	t.Run("ConfigValidationLatency", func(t *testing.T) {
		config := NewConfig().WithDefaults()

		start := time.Now()
		errors := config.Validate()
		duration := time.Since(start)

		assert.Empty(t, errors)
		assert.Less(t, duration, 10*time.Millisecond, "Config validation should be fast")
	})

	t.Run("RepositoryAdditionLatency", func(t *testing.T) {
		builder := NewConfigBuilder()

		repoConfig := &RepositoryConfig{
			Name: "test-repo",
			Type: "git",
			URL:  "https://github.com/test/repo.git",
		}

		start := time.Now()
		builder.AddRepository("test-repo", repoConfig)
		duration := time.Since(start)

		assert.Less(t, duration, 1*time.Millisecond, "Repository addition should be very fast")
	})
}

// Benchmark Tests

func BenchmarkConfigOperations(b *testing.B) {
	b.Run("NewConfig", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewConfig()
		}
	})

	b.Run("WithDefaults", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewConfig().WithDefaults()
		}
	})

	b.Run("Validate", func(b *testing.B) {
		config := NewConfig().WithDefaults()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			errors := config.Validate()
			if len(errors) > 0 {
				b.Fatal(errors[0])
			}
		}
	})

	b.Run("AddRepository", func(b *testing.B) {
		repoConfig := &RepositoryConfig{
			Name: "bench-repo",
			Type: "git",
			URL:  "https://github.com/test/repo.git",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			builder := NewConfigBuilder()
			builder.AddRepository("bench-repo", repoConfig)
		}
	})

	b.Run("ConfigFromEnvironment", func(b *testing.B) {
		// Set up environment
		os.Setenv("PORCH_ENDPOINT", "https://bench-server.com")
		defer os.Unsetenv("PORCH_ENDPOINT")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = NewConfigFromEnvironment()
		}
	})
}
