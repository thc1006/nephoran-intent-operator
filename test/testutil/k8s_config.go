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
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	porch "github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// ConfigSource represents where the configuration comes from
type ConfigSource string

const (
	ConfigSourceMock        ConfigSource = "mock"
	ConfigSourceKubeconfig  ConfigSource = "kubeconfig"
	ConfigSourceInCluster   ConfigSource = "in-cluster"
	ConfigSourceEnvironment ConfigSource = "environment"
	ConfigSourceEnvtest     ConfigSource = "envtest"
)

// K8sConfigResult contains the configuration and metadata about its source
type K8sConfigResult struct {
	Config *rest.Config
	Source ConfigSource
	Error  error
}

// GetTestKubernetesConfig returns a Kubernetes config suitable for testing.
// It tries multiple methods in order of preference and returns metadata about the source.
func GetTestKubernetesConfig(t *testing.T) *K8sConfigResult {
	t.Helper()

	// 1. Check if running in short test mode
	if testing.Short() {
		t.Log("Running in short test mode, using mock configuration")
		return &K8sConfigResult{
			Config: createMockConfig(),
			Source: ConfigSourceMock,
		}
	}

	// 2. Try envtest first (controlled test environment)
	if useEnvtest := os.Getenv("USE_ENVTEST"); useEnvtest == "true" {
		if config, err := createEnvtestConfig(t); err == nil {
			t.Log("Using envtest configuration")
			return &K8sConfigResult{
				Config: config,
				Source: ConfigSourceEnvtest,
			}
		}
		t.Log("Envtest requested but failed to initialize, falling back")
	}

	// 3. Try environment variables (CI/testing)
	if testEndpoint := os.Getenv("TEST_KUBERNETES_ENDPOINT"); testEndpoint != "" {
		t.Logf("Using test endpoint from environment: %s", testEndpoint)
		return &K8sConfigResult{
			Config: &rest.Config{
				Host:    testEndpoint,
				QPS:     100,
				Burst:   150,
				Timeout: 30 * time.Second,
			},
			Source: ConfigSourceEnvironment,
		}
	}

	// 4. Try kubeconfig file (local development)
	if config, err := loadKubeconfigFile(); err == nil {
		t.Log("Using kubeconfig file")
		return &K8sConfigResult{
			Config: config,
			Source: ConfigSourceKubeconfig,
		}
	}

	// 5. Try in-cluster config (actual cluster deployment)
	if config, err := rest.InClusterConfig(); err == nil {
		t.Log("Using in-cluster configuration")
		// Optimize for performance testing
		config.QPS = 100
		config.Burst = 150
		config.Timeout = 30 * time.Second
		return &K8sConfigResult{
			Config: config,
			Source: ConfigSourceInCluster,
		}
	}

	// 6. Fall back to mock configuration
	t.Log("All real configuration methods failed, using mock configuration")
	return &K8sConfigResult{
		Config: createMockConfig(),
		Source: ConfigSourceMock,
	}
}

// createMockConfig creates a mock Kubernetes configuration for testing
func createMockConfig() *rest.Config {
	return &rest.Config{
		Host:    "http://localhost:8080",
		QPS:     100,
		Burst:   150,
		Timeout: 30 * time.Second,
	}
}

// loadKubeconfigFile attempts to load configuration from kubeconfig file
func loadKubeconfigFile() (*rest.Config, error) {
	var kubeconfigPath string
	
	// Check KUBECONFIG environment variable first
	if kconfig := os.Getenv("KUBECONFIG"); kconfig != "" {
		kubeconfigPath = kconfig
	} else if home := homedir.HomeDir(); home != "" {
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	if kubeconfigPath == "" {
		return nil, fmt.Errorf("no kubeconfig path available")
	}

	// Check if file exists
	if _, err := os.Stat(kubeconfigPath); err != nil {
		return nil, fmt.Errorf("kubeconfig file not found: %w", err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	// Optimize for testing
	config.QPS = 100
	config.Burst = 150
	config.Timeout = 30 * time.Second

	return config, nil
}

// createEnvtestConfig creates a configuration using controller-runtime's envtest
func createEnvtestConfig(t *testing.T) (*rest.Config, error) {
	t.Helper()

	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{"../../config/crd/bases"},
	}

	config, err := testEnv.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start envtest environment: %w", err)
	}

	// Store the test environment for cleanup
	t.Cleanup(func() {
		if err := testEnv.Stop(); err != nil {
			t.Errorf("Failed to stop test environment: %v", err)
		}
	})

	return config, nil
}

// CreateTestPorchClient creates a Porch client for testing with appropriate fallbacks
func CreateTestPorchClient(t *testing.T, configResult *K8sConfigResult) (*porch.Client, error) {
	t.Helper()

	if configResult.Error != nil {
		return nil, configResult.Error
	}

	// Create appropriate client configuration based on source
	clientConfig := &porch.ClientConfig{
		Endpoint: getTestPorchEndpoint(configResult.Source),
		AuthConfig: &porch.AuthConfig{
			Type: "none", // No auth for testing
		},
	}

	clientOpts := porch.ClientOptions{
		Config:     clientConfig,
		KubeConfig: configResult.Config,
	}

	client, err := porch.NewClient(clientOpts)
	if err != nil {
		// For mock configurations, client creation might fail
		// but we want to test the structure anyway
		if configResult.Source == ConfigSourceMock {
			t.Logf("Mock client creation failed (expected in unit test environment): %v", err)
			return nil, fmt.Errorf("mock client unavailable for testing: %w", err)
		}
		return nil, fmt.Errorf("failed to create porch client: %w", err)
	}

	return client, nil
}

// getTestPorchEndpoint returns the appropriate Porch endpoint for the configuration source
func getTestPorchEndpoint(source ConfigSource) string {
	switch source {
	case ConfigSourceMock, ConfigSourceEnvironment:
		return "http://localhost:8080"
	case ConfigSourceEnvtest:
		return "http://localhost:8080"
	default:
		// Real cluster configurations
		if endpoint := os.Getenv("PORCH_ENDPOINT"); endpoint != "" {
			return endpoint
		}
		return "http://porch-server:8080"
	}
}

// SkipIfNoRealCluster skips the test if no real Kubernetes cluster is available
func SkipIfNoRealCluster(t *testing.T, configResult *K8sConfigResult) {
	t.Helper()
	
	if configResult.Source == ConfigSourceMock {
		t.Skip("Skipping test - no real Kubernetes cluster available")
	}
}

// IsRealCluster returns true if the configuration represents a real cluster
func IsRealCluster(configResult *K8sConfigResult) bool {
	return configResult.Source == ConfigSourceInCluster || 
		   configResult.Source == ConfigSourceKubeconfig ||
		   (configResult.Source == ConfigSourceEnvironment && configResult.Config.Host != "http://localhost:8080")
}

// LogConfigSource logs information about the configuration source being used
func LogConfigSource(t *testing.T, configResult *K8sConfigResult) {
	t.Helper()
	
	t.Logf("Kubernetes config source: %s", configResult.Source)
	if configResult.Config != nil {
		t.Logf("Kubernetes endpoint: %s", configResult.Config.Host)
		t.Logf("QPS: %.1f, Burst: %d, Timeout: %v", 
			configResult.Config.QPS, configResult.Config.Burst, configResult.Config.Timeout)
	}
}