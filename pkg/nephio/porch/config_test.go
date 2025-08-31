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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
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

	// Verify defaults are applied
	assert.Equal(t, "default", config.KubernetesConfig.Namespace)
	assert.Equal(t, "https://porch-server.porch-system.svc.cluster.local", config.PorchConfig.ServerURL)
	assert.Equal(t, 30*time.Second, config.PorchConfig.Timeout)
	assert.Equal(t, 3, config.PorchConfig.RetryConfig.MaxAttempts)
	assert.Equal(t, 1*time.Second, config.PorchConfig.RetryConfig.InitialDelay)
	assert.Equal(t, 2.0, config.PorchConfig.RetryConfig.Multiplier)

	// Verify function execution defaults
	assert.Equal(t, "docker", config.Functions.Execution.Runtime)
	assert.Equal(t, 5*time.Minute, config.Functions.Execution.DefaultTimeout)
	assert.Equal(t, 10, config.Functions.Execution.MaxConcurrent)
	assert.Equal(t, "1000m", config.Functions.Execution.ResourceLimits.CPU)
	assert.Equal(t, "1Gi", config.Functions.Execution.ResourceLimits.Memory)

	// Verify cache defaults
	assert.True(t, config.Cache.Enabled)
	assert.Equal(t, 5*time.Minute, config.Cache.DefaultTTL)
	assert.Equal(t, 1000, config.Cache.MaxEntries)

	// Verify security defaults
	assert.True(t, config.Security.Enabled)
	assert.True(t, config.Security.ImageScanningEnabled)
	assert.Equal(t, "medium", config.Security.RequiredSecurityLevel)

	// Verify observability defaults
	assert.True(t, config.Observability.Metrics.Enabled)
	assert.Equal(t, ":8080", config.Observability.Metrics.Address)
	assert.True(t, config.Observability.Tracing.Enabled)
	assert.Equal(t, "jaeger", config.Observability.Tracing.Provider)
	assert.True(t, config.Observability.Logging.Enabled)
	assert.Equal(t, "info", config.Observability.Logging.Level)
}

func TestConfigBuilder(t *testing.T) {
	config := NewConfig().
		WithNamespace("test-namespace").
		WithPorchServer("https://custom-porch-server.example.com").
		WithTimeout(60*time.Second).
		WithMaxRetries(5).
		WithCache(true, 10*time.Minute).
		WithSecurity(true, true, "high").
		WithMetrics(true, ":9090").
		WithTracing(true, "zipkin", "http://zipkin:9411/api/v2/spans").
		WithLogging(true, "debug")

	assert.Equal(t, "test-namespace", config.KubernetesConfig.Namespace)
	assert.Equal(t, "https://custom-porch-server.example.com", config.PorchConfig.ServerURL)
	assert.Equal(t, 60*time.Second, config.PorchConfig.Timeout)
	assert.Equal(t, 5, config.PorchConfig.RetryConfig.MaxAttempts)
	assert.True(t, config.Cache.Enabled)
	assert.Equal(t, 10*time.Minute, config.Cache.DefaultTTL)
	assert.True(t, config.Security.Enabled)
	assert.True(t, config.Security.ImageScanningEnabled)
	assert.Equal(t, "high", config.Security.RequiredSecurityLevel)
	assert.True(t, config.Observability.Metrics.Enabled)
	assert.Equal(t, ":9090", config.Observability.Metrics.Address)
	assert.True(t, config.Observability.Tracing.Enabled)
	assert.Equal(t, "zipkin", config.Observability.Tracing.Provider)
	assert.Equal(t, "http://zipkin:9411/api/v2/spans", config.Observability.Tracing.Endpoint)
	assert.True(t, config.Observability.Logging.Enabled)
	assert.Equal(t, "debug", config.Observability.Logging.Level)
}

func TestConfigAddRepository(t *testing.T) {
	config := NewConfig()

	repoConfig := &RepositoryConfig{
		Name: "test-repo",
		Type: "git",
		URL:  "https://github.com/test/repo.git",
	}

	config.AddRepository(repoConfig)

	assert.Contains(t, config.Repositories, "test-repo")
	assert.Equal(t, repoConfig, config.Repositories["test-repo"])
}

func TestConfigAddFunction(t *testing.T) {
	config := NewConfig().WithDefaults()

	funcConfig := &FunctionConfig{
		Image: "gcr.io/kpt-fn/custom-function:v1.0.0",
		ConfigMap: map[string]string{
			"param": "value",
		},
	}

	config.AddFunction("custom-function", funcConfig)

	assert.Contains(t, config.Functions.Registry.Functions, "custom-function")
	assert.Equal(t, funcConfig, config.Functions.Registry.Functions["custom-function"])
}

func TestConfigSetResourceLimits(t *testing.T) {
	config := NewConfig().WithDefaults()

	limits := &FunctionResourceLimits{
		CPU:     "2000m",
		Memory:  "2Gi",
		Storage: "10Gi",
		Timeout: 10 * time.Minute,
	}

	config.SetResourceLimits(limits)

	assert.Equal(t, limits, config.Functions.Execution.ResourceLimits)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "ValidConfig",
			config: NewConfig().WithDefaults().
				WithNamespace("valid-namespace").
				WithPorchServer("https://valid-server.com"),
			expectError: false,
		},
		{
			name: "InvalidNamespace",
			config: NewConfig().WithDefaults().
				WithNamespace("").
				WithPorchServer("https://valid-server.com"),
			expectError: true,
			errorMsg:    "namespace cannot be empty",
		},
		{
			name: "InvalidServerURL",
			config: NewConfig().WithDefaults().
				WithNamespace("valid-namespace").
				WithPorchServer("invalid-url"),
			expectError: true,
			errorMsg:    "invalid server URL",
		},
		{
			name: "NegativeTimeout",
			config: NewConfig().WithDefaults().
				WithNamespace("valid-namespace").
				WithPorchServer("https://valid-server.com").
				WithTimeout(-1 * time.Second),
			expectError: true,
			errorMsg:    "timeout must be positive",
		},
		{
			name: "InvalidRetryCount",
			config: NewConfig().WithDefaults().
				WithNamespace("valid-namespace").
				WithPorchServer("https://valid-server.com").
				WithMaxRetries(-1),
			expectError: true,
			errorMsg:    "max retry attempts must be non-negative",
		},
		{
			name: "InvalidCacheTTL",
			config: NewConfig().WithDefaults().
				WithNamespace("valid-namespace").
				WithPorchServer("https://valid-server.com").
				WithCache(true, -1*time.Second),
			expectError: true,
			errorMsg:    "cache TTL must be positive when caching is enabled",
		},
		{
			name: "InvalidSecurityLevel",
			config: NewConfig().WithDefaults().
				WithNamespace("valid-namespace").
				WithPorchServer("https://valid-server.com").
				WithSecurity(true, true, "invalid"),
			expectError: true,
			errorMsg:    "invalid security level",
		},
		{
			name: "InvalidLogLevel",
			config: NewConfig().WithDefaults().
				WithNamespace("valid-namespace").
				WithPorchServer("https://valid-server.com").
				WithLogging(true, "invalid"),
			expectError: true,
			errorMsg:    "invalid log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigFromEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("PORCH_SERVER_URL", "https://env-porch-server.com")
	os.Setenv("PORCH_NAMESPACE", "env-namespace")
	os.Setenv("PORCH_TIMEOUT", "45s")
	os.Setenv("PORCH_MAX_RETRIES", "7")
	os.Setenv("PORCH_CACHE_ENABLED", "false")
	os.Setenv("PORCH_SECURITY_ENABLED", "false")
	os.Setenv("PORCH_METRICS_ENABLED", "true")
	os.Setenv("PORCH_METRICS_ADDRESS", ":8888")
	os.Setenv("PORCH_LOG_LEVEL", "warn")

	defer func() {
		// Clean up environment variables
		os.Unsetenv("PORCH_SERVER_URL")
		os.Unsetenv("PORCH_NAMESPACE")
		os.Unsetenv("PORCH_TIMEOUT")
		os.Unsetenv("PORCH_MAX_RETRIES")
		os.Unsetenv("PORCH_CACHE_ENABLED")
		os.Unsetenv("PORCH_SECURITY_ENABLED")
		os.Unsetenv("PORCH_METRICS_ENABLED")
		os.Unsetenv("PORCH_METRICS_ADDRESS")
		os.Unsetenv("PORCH_LOG_LEVEL")
	}()

	config := NewConfigFromEnvironment()

	assert.Equal(t, "https://env-porch-server.com", config.PorchConfig.ServerURL)
	assert.Equal(t, "env-namespace", config.KubernetesConfig.Namespace)
	assert.Equal(t, 45*time.Second, config.PorchConfig.Timeout)
	assert.Equal(t, 7, config.PorchConfig.RetryConfig.MaxAttempts)
	assert.False(t, config.Cache.Enabled)
	assert.False(t, config.Security.Enabled)
	assert.True(t, config.Observability.Metrics.Enabled)
	assert.Equal(t, ":8888", config.Observability.Metrics.Address)
	assert.Equal(t, "warn", config.Observability.Logging.Level)
}

func TestConfigFromKubeconfig(t *testing.T) {
	// Create a temporary kubeconfig file
	kubeconfigContent := `
apiVersion: v1
kind: Config
current-context: test-context
contexts:
- context:
    cluster: test-cluster
    namespace: kubeconfig-namespace
    user: test-user
  name: test-context
clusters:
- cluster:
    server: https://test-k8s-server.com
  name: test-cluster
users:
- name: test-user
  user:
    token: test-token
`

	kubeconfigFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	require.NoError(t, err)
	defer os.Remove(kubeconfigFile.Name())

	_, err = kubeconfigFile.WriteString(kubeconfigContent)
	require.NoError(t, err)
	kubeconfigFile.Close()

	config, err := NewConfigFromKubeconfig(kubeconfigFile.Name())
	require.NoError(t, err)

	assert.NotNil(t, config)
	assert.NotNil(t, config.KubernetesConfig.RestConfig)
	// The namespace from context should be set
	assert.Equal(t, "kubeconfig-namespace", config.KubernetesConfig.Namespace)
}

func TestConfigFromRestConfig(t *testing.T) {
	restConfig := &rest.Config{
		Host:        "https://test-k8s-server.com",
		BearerToken: "test-token",
	}

	config := NewConfigFromRestConfig(restConfig)

	assert.NotNil(t, config)
	assert.Equal(t, restConfig, config.KubernetesConfig.RestConfig)
	// Default namespace should be set when not specified in rest config
	assert.Equal(t, "default", config.KubernetesConfig.Namespace)
}

func TestRepositoryConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *RepositoryConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "ValidGitRepository",
			config: &RepositoryConfig{
				Name: "valid-git-repo",
				Type: "git",
				URL:  "https://github.com/test/repo.git",
			},
			expectError: false,
		},
		{
			name: "ValidOCIRepository",
			config: &RepositoryConfig{
				Name: "valid-oci-repo",
				Type: "oci",
				URL:  "oci://registry.example.com/packages",
			},
			expectError: false,
		},
		{
			name: "MissingName",
			config: &RepositoryConfig{
				Type: "git",
				URL:  "https://github.com/test/repo.git",
			},
			expectError: true,
			errorMsg:    "repository name is required",
		},
		{
			name: "MissingType",
			config: &RepositoryConfig{
				Name: "test-repo",
				URL:  "https://github.com/test/repo.git",
			},
			expectError: true,
			errorMsg:    "repository type is required",
		},
		{
			name: "MissingURL",
			config: &RepositoryConfig{
				Name: "test-repo",
				Type: "git",
			},
			expectError: true,
			errorMsg:    "repository URL is required",
		},
		{
			name: "InvalidURL",
			config: &RepositoryConfig{
				Name: "test-repo",
				Type: "git",
				URL:  "invalid-url",
			},
			expectError: true,
			errorMsg:    "invalid repository URL",
		},
		{
			name: "UnsupportedType",
			config: &RepositoryConfig{
				Name: "test-repo",
				Type: "svn",
				URL:  "https://svn.example.com/repo",
			},
			expectError: true,
			errorMsg:    "unsupported repository type",
		},
		{
			name: "GitWithInvalidProtocol",
			config: &RepositoryConfig{
				Name: "test-repo",
				Type: "git",
				URL:  "ftp://ftp.example.com/repo.git",
			},
			expectError: true,
			errorMsg:    "git repository URL must use https:// or git@ protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAuthConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *AuthConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "ValidBasicAuth",
			config: &AuthConfig{
				Type:     "basic",
				Username: "test-user",
				Password: "test-password",
			},
			expectError: false,
		},
		{
			name: "ValidTokenAuth",
			config: &AuthConfig{
				Type:  "token",
				Token: "test-token",
			},
			expectError: false,
		},
		{
			name: "ValidSSHAuth",
			config: &AuthConfig{
				Type:       "ssh",
				PrivateKey: []byte("-----BEGIN PRIVATE KEY-----\ntest-key\n-----END PRIVATE KEY-----"),
			},
			expectError: false,
		},
		{
			name: "MissingType",
			config: &AuthConfig{
				Username: "test-user",
				Password: "test-password",
			},
			expectError: true,
			errorMsg:    "auth type is required",
		},
		{
			name: "BasicAuthMissingUsername",
			config: &AuthConfig{
				Type:     "basic",
				Password: "test-password",
			},
			expectError: true,
			errorMsg:    "username is required for basic auth",
		},
		{
			name: "BasicAuthMissingPassword",
			config: &AuthConfig{
				Type:     "basic",
				Username: "test-user",
			},
			expectError: true,
			errorMsg:    "password is required for basic auth",
		},
		{
			name: "TokenAuthMissingToken",
			config: &AuthConfig{
				Type: "token",
			},
			expectError: true,
			errorMsg:    "token is required for token auth",
		},
		{
			name: "SSHAuthMissingPrivateKey",
			config: &AuthConfig{
				Type: "ssh",
			},
			expectError: true,
			errorMsg:    "private key is required for SSH auth",
		},
		{
			name: "UnsupportedAuthType",
			config: &AuthConfig{
				Type: "oauth",
			},
			expectError: true,
			errorMsg:    "unsupported auth type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *SyncConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "ValidSyncConfig",
			config: &SyncConfig{
				AutoSync: true,
				Interval: &metav1.Duration{Duration: 5 * time.Minute},
			},
			expectError: false,
		},
		{
			name: "ValidDisabledSync",
			config: &SyncConfig{
				AutoSync: false,
			},
			expectError: false,
		},
		{
			name: "AutoSyncWithoutInterval",
			config: &SyncConfig{
				AutoSync: true,
			},
			expectError: true,
			errorMsg:    "sync interval is required when auto sync is enabled",
		},
		{
			name: "InvalidInterval",
			config: &SyncConfig{
				AutoSync: true,
				Interval: &metav1.Duration{Duration: -1 * time.Second},
			},
			expectError: true,
			errorMsg:    "sync interval must be positive",
		},
		{
			name: "IntervalTooShort",
			config: &SyncConfig{
				AutoSync: true,
				Interval: &metav1.Duration{Duration: 10 * time.Second},
			},
			expectError: true,
			errorMsg:    "sync interval must be at least 1 minute",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFunctionResourceLimitsValidation(t *testing.T) {
	tests := []struct {
		name        string
		limits      *FunctionResourceLimits
		expectError bool
		errorMsg    string
	}{
		{
			name: "ValidLimits",
			limits: &FunctionResourceLimits{
				CPU:     "1000m",
				Memory:  "1Gi",
				Storage: "10Gi",
				Timeout: 5 * time.Minute,
			},
			expectError: false,
		},
		{
			name: "InvalidCPUFormat",
			limits: &FunctionResourceLimits{
				CPU:     "invalid",
				Memory:  "1Gi",
				Storage: "10Gi",
				Timeout: 5 * time.Minute,
			},
			expectError: true,
			errorMsg:    "invalid CPU format",
		},
		{
			name: "InvalidMemoryFormat",
			limits: &FunctionResourceLimits{
				CPU:     "1000m",
				Memory:  "invalid",
				Storage: "10Gi",
				Timeout: 5 * time.Minute,
			},
			expectError: true,
			errorMsg:    "invalid memory format",
		},
		{
			name: "InvalidTimeout",
			limits: &FunctionResourceLimits{
				CPU:     "1000m",
				Memory:  "1Gi",
				Storage: "10Gi",
				Timeout: -1 * time.Second,
			},
			expectError: true,
			errorMsg:    "timeout must be positive",
		},
		{
			name: "TimeoutTooLong",
			limits: &FunctionResourceLimits{
				CPU:     "1000m",
				Memory:  "1Gi",
				Storage: "10Gi",
				Timeout: 2 * time.Hour,
			},
			expectError: true,
			errorMsg:    "timeout must not exceed 1 hour",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.limits.Validate()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
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
		err := config.Validate()
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.Less(t, duration, 10*time.Millisecond, "Config validation should be fast")
	})

	t.Run("RepositoryAdditionLatency", func(t *testing.T) {
		config := NewConfig().WithDefaults()

		repoConfig := &RepositoryConfig{
			Name: "test-repo",
			Type: "git",
			URL:  "https://github.com/test/repo.git",
		}

		start := time.Now()
		config.AddRepository(repoConfig)
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
			err := config.Validate()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("AddRepository", func(b *testing.B) {
		config := NewConfig().WithDefaults()
		repoConfig := &RepositoryConfig{
			Name: "bench-repo",
			Type: "git",
			URL:  "https://github.com/test/repo.git",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			config.AddRepository(repoConfig)
		}
	})

	b.Run("ConfigFromEnvironment", func(b *testing.B) {
		// Set up environment
		os.Setenv("PORCH_SERVER_URL", "https://bench-server.com")
		defer os.Unsetenv("PORCH_SERVER_URL")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewConfigFromEnvironment()
		}
	})
}
