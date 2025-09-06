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

package testutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func TestGetTestKubernetesConfig(t *testing.T) {
	tests := []struct {
		name           string
		setupEnv       func()
		cleanupEnv     func()
		expectedSource ConfigSource
		shouldSkip     bool
	}{
		{
			name: "short_test_mode",
			setupEnv: func() {
				// testing.Short() is handled by the Go testing framework
				// We can't easily mock it here, but we can test other paths
			},
			cleanupEnv:     func() {},
			expectedSource: ConfigSourceMock, // Will fall back to mock in most cases
		},
		{
			name: "environment_endpoint",
			setupEnv: func() {
				os.Setenv("TEST_KUBERNETES_ENDPOINT", "http://test-endpoint:8080")
			},
			cleanupEnv: func() {
				os.Unsetenv("TEST_KUBERNETES_ENDPOINT")
			},
			// Note: expectedSource will be ConfigSourceMock in short test mode, ConfigSourceEnvironment otherwise
			expectedSource: "", // Don't enforce specific source as it depends on testing.Short()
		},
		{
			name: "envtest_mode",
			setupEnv: func() {
				os.Setenv("USE_ENVTEST", "true")
			},
			cleanupEnv: func() {
				os.Unsetenv("USE_ENVTEST")
			},
			// This will likely fail in most environments, which is fine
			shouldSkip: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.setupEnv()
			defer tt.cleanupEnv()

			// Skip tests that require special setup
			if tt.shouldSkip {
				t.Skip("Skipping test that requires special environment setup")
				return
			}

			// Test
			result := GetTestKubernetesConfig(t)
			
			// Assertions
			require.NotNil(t, result, "Should return a config result")
			require.NotNil(t, result.Config, "Should return a valid config")
			
			if tt.expectedSource != "" {
				assert.Equal(t, tt.expectedSource, result.Source, "Should use expected config source")
			} else {
				// For tests where source depends on testing.Short(), just verify it's a valid source
				validSources := []ConfigSource{ConfigSourceMock, ConfigSourceEnvironment, ConfigSourceKubeconfig, ConfigSourceInCluster, ConfigSourceEnvtest}
				assert.Contains(t, validSources, result.Source, "Should use a valid config source")
			}

			// Basic config validation
			assert.NotEmpty(t, result.Config.Host, "Config should have a host")
			assert.Greater(t, result.Config.QPS, float32(0), "Config should have positive QPS")
			assert.Greater(t, result.Config.Burst, 0, "Config should have positive burst")
		})
	}
}

func TestIsRealCluster(t *testing.T) {
	tests := []struct {
		name     string
		result   *K8sConfigResult
		expected bool
	}{
		{
			name: "mock_config",
			result: &K8sConfigResult{
				Source: ConfigSourceMock,
			},
			expected: false,
		},
		{
			name: "in_cluster_config",
			result: &K8sConfigResult{
				Source: ConfigSourceInCluster,
			},
			expected: true,
		},
		{
			name: "kubeconfig_file",
			result: &K8sConfigResult{
				Source: ConfigSourceKubeconfig,
			},
			expected: true,
		},
		{
			name: "environment_real_endpoint",
			result: &K8sConfigResult{
				Source: ConfigSourceEnvironment,
				Config: &rest.Config{
					Host: "https://real-cluster:6443",
				},
			},
			expected: true,
		},
		{
			name: "environment_mock_endpoint",
			result: &K8sConfigResult{
				Source: ConfigSourceEnvironment,
				Config: &rest.Config{
					Host: "http://localhost:8080",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRealCluster(tt.result)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateMockConfig(t *testing.T) {
	config := createMockConfig()
	
	require.NotNil(t, config)
	assert.Equal(t, "http://localhost:8080", config.Host)
	assert.Equal(t, float32(100), config.QPS)
	assert.Equal(t, 150, config.Burst)
	assert.NotZero(t, config.Timeout)
}

func TestGetTestPorchEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		source   ConfigSource
		envVar   string
		envValue string
		expected string
	}{
		{
			name:     "mock_source",
			source:   ConfigSourceMock,
			expected: "http://localhost:8080",
		},
		{
			name:     "environment_source",
			source:   ConfigSourceEnvironment,
			expected: "http://localhost:8080",
		},
		{
			name:     "real_cluster_with_env_var",
			source:   ConfigSourceInCluster,
			envVar:   "PORCH_ENDPOINT",
			envValue: "http://custom-porch:9090",
			expected: "http://custom-porch:9090",
		},
		{
			name:     "real_cluster_default",
			source:   ConfigSourceInCluster,
			expected: "http://porch-server:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variable if needed
			if tt.envVar != "" && tt.envValue != "" {
				os.Setenv(tt.envVar, tt.envValue)
				defer os.Unsetenv(tt.envVar)
			}

			result := getTestPorchEndpoint(tt.source)
			assert.Equal(t, tt.expected, result)
		})
	}
}

