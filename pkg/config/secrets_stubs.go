//go:build disable_rag

package config

import (
	"context"
	"fmt"

	"github.com/thc1006/nephoran-intent-operator/pkg/interfaces"
)

// KubernetesSecretManager provides a concrete type for disable_rag builds
type KubernetesSecretManager struct {
	namespace string
}

// Stub implementations for disable_rag builds
func NewSecretManager(namespace string) (*KubernetesSecretManager, error) {
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

func (sm *KubernetesSecretManager) CreateSecretFromEnvVars(ctx context.Context, secretName string, envVarMapping map[string]string) error {
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

// Deprecated - use KubernetesSecretManager instead
type SecretManager = KubernetesSecretManager

func (sm *KubernetesSecretManager) Get(ctx context.Context, key string) (string, error) {
	return "", fmt.Errorf("secret manager disabled with disable_rag build tag")
}

func (sm *KubernetesSecretManager) Refresh(ctx context.Context) error {
	return nil
}

func (sm *KubernetesSecretManager) SetLogger(logger interface{}) {
	// No-op for stub
}

func (sm *KubernetesSecretManager) CreateOrUpdateSecretFromAPIKeys(ctx context.Context, apiKeys *interfaces.APIKeys) error {
	return fmt.Errorf("secret manager disabled with disable_rag build tag")
}