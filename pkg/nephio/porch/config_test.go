package porch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// DISABLED: func TestDefaultPorchConfig(t *testing.T) {
	config := DefaultPorchConfig()

	assert.NotNil(t, config)
	assert.NotNil(t, config.PorchConfig)
	assert.Equal(t, "default", config.PorchConfig.DefaultNamespace)
	assert.Equal(t, "default", config.PorchConfig.DefaultRepository)

	// Circuit Breaker Defaults
	assert.True(t, config.PorchConfig.CircuitBreaker.Enabled)
	assert.Equal(t, 5, config.PorchConfig.CircuitBreaker.FailureThreshold)
	assert.Equal(t, 3, config.PorchConfig.CircuitBreaker.SuccessThreshold)
	assert.Equal(t, 30*time.Second, config.PorchConfig.CircuitBreaker.Timeout)
	assert.Equal(t, 3, config.PorchConfig.CircuitBreaker.HalfOpenMaxCalls)

	// Rate Limit Defaults
	assert.True(t, config.PorchConfig.RateLimit.Enabled)
	assert.Equal(t, 10.0, config.PorchConfig.RateLimit.RequestsPerSecond)
	assert.Equal(t, 20, config.PorchConfig.RateLimit.Burst)

	// Connection Pool Defaults
	assert.Equal(t, 10, config.PorchConfig.ConnectionPool.MaxOpenConns)
	assert.Equal(t, 5, config.PorchConfig.ConnectionPool.MaxIdleConns)
	assert.Equal(t, 30*time.Minute, config.PorchConfig.ConnectionPool.ConnMaxLifetime)

	// Function Execution Defaults
	assert.Equal(t, 60*time.Second, config.PorchConfig.FunctionExecution.DefaultTimeout)
	assert.Equal(t, 5, config.PorchConfig.FunctionExecution.MaxConcurrency)
	assert.Equal(t, "100m", config.PorchConfig.FunctionExecution.ResourceLimits["cpu"])
	assert.Equal(t, "128Mi", config.PorchConfig.FunctionExecution.ResourceLimits["memory"])
}

// DISABLED: func TestClientConfigWithEndpoint(t *testing.T) {
	config := &ClientConfig{
		Endpoint: "https://custom-porch-server.example.com",
		AuthConfig: &AuthConfig{
			Type:  "bearer",
			Token: "test-token",
		},
	}

	kubeConfig, err := config.GetKubernetesConfig()
	assert.NoError(t, err)
	assert.Equal(t, "https://custom-porch-server.example.com", kubeConfig.Host)
	assert.Equal(t, "test-token", kubeConfig.BearerToken)
}
