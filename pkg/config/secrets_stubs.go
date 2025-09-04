//go:build disable_rag

package config

import (
	"context"
	"fmt"

	"github.com/thc1006/nephoran-intent-operator/pkg/interfaces"
)

// APIKeys is an alias to interfaces.APIKeys for backward compatibility within the config package
type APIKeys = interfaces.APIKeys

// SecretManager interface defines the contract for secret management operations
// (stub implementation for disable_rag builds)
type SecretManager interface {
	// Get retrieves a secret value by key
	Get(ctx context.Context, key string) (string, error)

	// GetAPIKeys retrieves all API keys in a structured format
	GetAPIKeys(ctx context.Context) (*APIKeys, error)

	// Set stores a secret value by key (for development/testing)
	Set(ctx context.Context, key, value string) error

	// Refresh invalidates any cached secrets and forces reload
	Refresh(ctx context.Context) error

	// CreateSecretFromEnvVars creates a secret from environment variables
	CreateSecretFromEnvVars(ctx context.Context, key string) error

	// Close cleans up any resources (optional)
	Close() error
}

// KubernetesSecretManager provides a concrete type for disable_rag builds
type KubernetesSecretManager struct {
	namespace string
}

// Stub implementations for disable_rag builds
func NewSecretManager(namespace string) (SecretManager, error) {
	return &KubernetesSecretManager{namespace: namespace}, nil
}

func LoadFileBasedAPIKeysWithValidation() (*interfaces.APIKeys, error) {
	return &interfaces.APIKeys{}, nil
}

// GetAPIKeys stub method for KubernetesSecretManager (disable_rag builds)
func (sm *KubernetesSecretManager) GetAPIKeys(ctx context.Context) (*interfaces.APIKeys, error) {
	return &interfaces.APIKeys{}, nil
}

// Stub methods for compatibility (disable_rag builds)
func (sm *KubernetesSecretManager) GetSecretValue(ctx context.Context, secretName, key, envVarName string) (string, error) {
	return "", fmt.Errorf("secret manager disabled with disable_rag build tag")
}

func (sm *KubernetesSecretManager) CreateSecretFromEnvVars(ctx context.Context, key string) error {
	return fmt.Errorf("secret manager disabled with disable_rag build tag")
}

func (sm *KubernetesSecretManager) UpdateSecret(ctx context.Context, secretName string, data map[string][]byte) error {
	return fmt.Errorf("secret manager disabled with disable_rag build tag")
}

func (sm *KubernetesSecretManager) SecretExists(ctx context.Context, secretName string) bool {
	return false
}

func (sm *KubernetesSecretManager) RotateSecret(ctx context.Context, secretName, secretKey, newValue string) error {
	return fmt.Errorf("secret manager disabled with disable_rag build tag")
}

func (sm *KubernetesSecretManager) GetSecretRotationInfo(ctx context.Context, secretName string) (map[string]string, error) {
	return nil, fmt.Errorf("secret manager disabled with disable_rag build tag")
}

func (sm *KubernetesSecretManager) Get(ctx context.Context, key string) (string, error) {
	return "", fmt.Errorf("secret manager disabled with disable_rag build tag")
}

func (sm *KubernetesSecretManager) Set(ctx context.Context, key, value string) error {
	return fmt.Errorf("secret manager disabled with disable_rag build tag")
}

func (sm *KubernetesSecretManager) Refresh(ctx context.Context) error {
	return nil
}

func (sm *KubernetesSecretManager) Close() error {
	return nil
}

func (sm *KubernetesSecretManager) SetLogger(logger interface{}) {
	// No-op for stub
}

func (sm *KubernetesSecretManager) CreateOrUpdateSecretFromAPIKeys(ctx context.Context, apiKeys *interfaces.APIKeys) error {
	return fmt.Errorf("secret manager disabled with disable_rag build tag")
}
