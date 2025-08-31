// Package security implements additional secrets backend implementations
package security

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileBackend implements SecretsBackend interface using file storage
type FileBackend struct {
	basePath string
	logger   *slog.Logger
	mu       sync.RWMutex
}

// NewFileBackend creates a new file-based secrets backend
func NewFileBackend(basePath string, logger *slog.Logger) (SecretsBackend, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create backend directory: %w", err)
	}

	return &FileBackend{
		basePath: basePath,
		logger:   logger,
	}, nil
}

// Store stores a secret to file system
func (fb *FileBackend) Store(ctx context.Context, key string, value *EncryptedSecret) error {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal secret: %w", err)
	}

	filePath := filepath.Join(fb.basePath, key+".json")
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write secret file: %w", err)
	}

	return nil
}

// Retrieve retrieves a secret from file system
func (fb *FileBackend) Retrieve(ctx context.Context, key string) (*EncryptedSecret, error) {
	fb.mu.RLock()
	defer fb.mu.RUnlock()

	filePath := filepath.Join(fb.basePath, key+".json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("secret not found: %s", key)
		}
		return nil, fmt.Errorf("failed to read secret file: %w", err)
	}

	var secret EncryptedSecret
	if err := json.Unmarshal(data, &secret); err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret: %w", err)
	}

	return &secret, nil
}

// Delete deletes a secret from file system
func (fb *FileBackend) Delete(ctx context.Context, key string) error {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	filePath := filepath.Join(fb.basePath, key+".json")
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("secret not found: %s", key)
		}
		return fmt.Errorf("failed to delete secret file: %w", err)
	}

	return nil
}

// List lists all secret keys (interface method - no prefix parameter)
func (fb *FileBackend) List(ctx context.Context) ([]string, error) {
	return fb.ListWithPrefix(ctx, "")
}

// ListWithPrefix lists all secrets with optional prefix filter (helper method)
func (fb *FileBackend) ListWithPrefix(ctx context.Context, prefix string) ([]string, error) {
	fb.mu.RLock()
	defer fb.mu.RUnlock()

	entries, err := os.ReadDir(fb.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var result []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		key := strings.TrimSuffix(name, ".json")
		if prefix == "" || strings.HasPrefix(key, prefix) {
			result = append(result, key)
		}
	}

	return result, nil
}

// Exists checks if a secret exists
func (fb *FileBackend) Exists(ctx context.Context, key string) (bool, error) {
	fb.mu.RLock()
	defer fb.mu.RUnlock()

	filePath := filepath.Join(fb.basePath, key+".json")
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if secret exists: %w", err)
	}
	return true, nil
}

// Rotate rotates a secret to a new version
func (fb *FileBackend) Rotate(ctx context.Context, key string, newValue *EncryptedSecret) error {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	// Check if secret exists
	exists, err := fb.Exists(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to check if secret exists during rotation: %w", err)
	}
	if !exists {
		return fmt.Errorf("cannot rotate non-existent secret: %s", key)
	}

	// Store the new version
	return fb.Store(ctx, key, newValue)
}

// Backup creates a backup of all secrets
func (fb *FileBackend) Backup(ctx context.Context) ([]byte, error) {
	fb.mu.RLock()
	defer fb.mu.RUnlock()

	secrets := make(map[string]*EncryptedSecret)

	entries, err := os.ReadDir(fb.basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		key := strings.TrimSuffix(entry.Name(), ".json")
		secret, err := fb.Retrieve(ctx, key)
		if err != nil {
			continue // Skip failed reads
		}
		secrets[key] = secret
	}

	return json.Marshal(secrets)
}

// Restore restores secrets from backup data
func (fb *FileBackend) Restore(ctx context.Context, data []byte) error {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	var secrets map[string]*EncryptedSecret
	if err := json.Unmarshal(data, &secrets); err != nil {
		return fmt.Errorf("failed to unmarshal backup data: %w", err)
	}

	for key, secret := range secrets {
		if err := fb.Store(ctx, key, secret); err != nil {
			return fmt.Errorf("failed to restore secret %s: %w", key, err)
		}
	}

	return nil
}

// Health checks the health of the file backend
func (fb *FileBackend) Health(ctx context.Context) error {
	// Check if directory is accessible
	if _, err := os.Stat(fb.basePath); err != nil {
		return fmt.Errorf("backend directory not accessible: %w", err)
	}

	// Try to create a temporary file to test write permissions
	tempFile := filepath.Join(fb.basePath, ".health_check")
	if err := os.WriteFile(tempFile, []byte("test"), 0600); err != nil {
		return fmt.Errorf("backend not writable: %w", err)
	}

	// Clean up
	os.Remove(tempFile)
	return nil
}

// Close closes the file backend
func (fb *FileBackend) Close() error {
	// No resources to close for file backend
	return nil
}

// HashiCorpVaultBackend implements SecretsBackend interface using HashiCorp Vault
type HashiCorpVaultBackend struct {
	config map[string]string
	logger *slog.Logger
	mu     sync.RWMutex
}

// NewHashiCorpVaultBackend creates a new HashiCorp Vault backend
func NewHashiCorpVaultBackend(config map[string]string, logger *slog.Logger) (SecretsBackend, error) {
	// Validate required configuration
	requiredFields := []string{"address", "token"}
	for _, field := range requiredFields {
		if _, exists := config[field]; !exists {
			return nil, fmt.Errorf("missing required config field: %s", field)
		}
	}

	return &HashiCorpVaultBackend{
		config: config,
		logger: logger,
	}, nil
}

// Store stores a secret in HashiCorp Vault
func (hv *HashiCorpVaultBackend) Store(ctx context.Context, key string, value *EncryptedSecret) error {
	// TODO: Implement actual HashiCorp Vault integration
	hv.logger.Warn("HashiCorp Vault backend not fully implemented", "operation", "Store", "key", key)
	return fmt.Errorf("hashicorp vault backend not implemented yet")
}

// Retrieve retrieves a secret from HashiCorp Vault
func (hv *HashiCorpVaultBackend) Retrieve(ctx context.Context, key string) (*EncryptedSecret, error) {
	// TODO: Implement actual HashiCorp Vault integration
	hv.logger.Warn("HashiCorp Vault backend not fully implemented", "operation", "Retrieve", "key", key)
	return nil, fmt.Errorf("hashicorp vault backend not implemented yet")
}

// Delete deletes a secret from HashiCorp Vault
func (hv *HashiCorpVaultBackend) Delete(ctx context.Context, key string) error {
	// TODO: Implement actual HashiCorp Vault integration
	hv.logger.Warn("HashiCorp Vault backend not fully implemented", "operation", "Delete", "key", key)
	return fmt.Errorf("hashicorp vault backend not implemented yet")
}

// List lists all secret keys
func (hv *HashiCorpVaultBackend) List(ctx context.Context) ([]string, error) {
	// TODO: Implement actual HashiCorp Vault integration
	hv.logger.Warn("HashiCorp Vault backend not fully implemented", "operation", "List")
	return nil, fmt.Errorf("hashicorp vault backend not implemented yet")
}

// Exists checks if a secret exists in HashiCorp Vault
func (hv *HashiCorpVaultBackend) Exists(ctx context.Context, key string) (bool, error) {
	// TODO: Implement actual HashiCorp Vault integration
	hv.logger.Warn("HashiCorp Vault backend not fully implemented", "operation", "Exists", "key", key)
	return false, fmt.Errorf("hashicorp vault backend not implemented yet")
}

// Rotate rotates a secret to a new version in HashiCorp Vault
func (hv *HashiCorpVaultBackend) Rotate(ctx context.Context, key string, newValue *EncryptedSecret) error {
	// TODO: Implement actual HashiCorp Vault integration
	hv.logger.Warn("HashiCorp Vault backend not fully implemented", "operation", "Rotate", "key", key)
	return fmt.Errorf("hashicorp vault backend not implemented yet")
}

// Backup creates a backup of all secrets
func (hv *HashiCorpVaultBackend) Backup(ctx context.Context) ([]byte, error) {
	// TODO: Implement actual HashiCorp Vault integration
	hv.logger.Warn("HashiCorp Vault backend not fully implemented", "operation", "Backup")
	return nil, fmt.Errorf("hashicorp vault backend not implemented yet")
}

// Restore restores secrets from backup data
func (hv *HashiCorpVaultBackend) Restore(ctx context.Context, data []byte) error {
	// TODO: Implement actual HashiCorp Vault integration
	hv.logger.Warn("HashiCorp Vault backend not fully implemented", "operation", "Restore")
	return fmt.Errorf("hashicorp vault backend not implemented yet")
}

// Health checks the health of the HashiCorp Vault backend
func (hv *HashiCorpVaultBackend) Health(ctx context.Context) error {
	// TODO: Implement actual HashiCorp Vault health check
	hv.logger.Warn("HashiCorp Vault backend not fully implemented", "operation", "Health")
	return fmt.Errorf("hashicorp vault backend not implemented yet")
}

// Close closes the HashiCorp Vault backend
func (hv *HashiCorpVaultBackend) Close() error {
	// TODO: Implement proper cleanup
	return nil
}

// KubernetesBackend implements SecretsBackend interface using Kubernetes secrets
type KubernetesBackend struct {
	config    map[string]string
	logger    *slog.Logger
	namespace string
	mu        sync.RWMutex
}

// NewKubernetesBackend creates a new Kubernetes secrets backend
func NewKubernetesBackend(config map[string]string, logger *slog.Logger) (SecretsBackend, error) {
	namespace := config["namespace"]
	if namespace == "" {
		namespace = "default"
	}

	return &KubernetesBackend{
		config:    config,
		logger:    logger,
		namespace: namespace,
	}, nil
}

// Store stores a secret as a Kubernetes secret
func (kb *KubernetesBackend) Store(ctx context.Context, key string, value *EncryptedSecret) error {
	// TODO: Implement actual Kubernetes client integration
	kb.logger.Warn("Kubernetes backend not fully implemented", "operation", "Store", "key", key, "namespace", kb.namespace)
	return fmt.Errorf("kubernetes backend not implemented yet")
}

// Retrieve retrieves a secret from Kubernetes
func (kb *KubernetesBackend) Retrieve(ctx context.Context, key string) (*EncryptedSecret, error) {
	// TODO: Implement actual Kubernetes client integration
	kb.logger.Warn("Kubernetes backend not fully implemented", "operation", "Retrieve", "key", key, "namespace", kb.namespace)
	return nil, fmt.Errorf("kubernetes backend not implemented yet")
}

// Delete deletes a secret from Kubernetes
func (kb *KubernetesBackend) Delete(ctx context.Context, key string) error {
	// TODO: Implement actual Kubernetes client integration
	kb.logger.Warn("Kubernetes backend not fully implemented", "operation", "Delete", "key", key, "namespace", kb.namespace)
	return fmt.Errorf("kubernetes backend not implemented yet")
}

// List lists all secret keys
func (kb *KubernetesBackend) List(ctx context.Context) ([]string, error) {
	// TODO: Implement actual Kubernetes client integration
	kb.logger.Warn("Kubernetes backend not fully implemented", "operation", "List", "namespace", kb.namespace)
	return nil, fmt.Errorf("kubernetes backend not implemented yet")
}

// Exists checks if a secret exists in Kubernetes
func (kb *KubernetesBackend) Exists(ctx context.Context, key string) (bool, error) {
	// TODO: Implement actual Kubernetes client integration
	kb.logger.Warn("Kubernetes backend not fully implemented", "operation", "Exists", "key", key, "namespace", kb.namespace)
	return false, fmt.Errorf("kubernetes backend not implemented yet")
}

// Rotate rotates a secret to a new version in Kubernetes
func (kb *KubernetesBackend) Rotate(ctx context.Context, key string, newValue *EncryptedSecret) error {
	// TODO: Implement actual Kubernetes client integration
	kb.logger.Warn("Kubernetes backend not fully implemented", "operation", "Rotate", "key", key, "namespace", kb.namespace)
	return fmt.Errorf("kubernetes backend not implemented yet")
}

// Backup creates a backup of all secrets
func (kb *KubernetesBackend) Backup(ctx context.Context) ([]byte, error) {
	// TODO: Implement actual Kubernetes client integration
	kb.logger.Warn("Kubernetes backend not fully implemented", "operation", "Backup", "namespace", kb.namespace)
	return nil, fmt.Errorf("kubernetes backend not implemented yet")
}

// Restore restores secrets from backup data
func (kb *KubernetesBackend) Restore(ctx context.Context, data []byte) error {
	// TODO: Implement actual Kubernetes client integration
	kb.logger.Warn("Kubernetes backend not fully implemented", "operation", "Restore", "namespace", kb.namespace)
	return fmt.Errorf("kubernetes backend not implemented yet")
}

// Health checks the health of the Kubernetes backend
func (kb *KubernetesBackend) Health(ctx context.Context) error {
	// TODO: Implement actual Kubernetes client integration
	kb.logger.Warn("Kubernetes backend not fully implemented", "operation", "Health", "namespace", kb.namespace)
	return fmt.Errorf("kubernetes backend not implemented yet")
}

// Close closes the Kubernetes backend
func (kb *KubernetesBackend) Close() error {
	// TODO: Implement proper cleanup
	return nil
}

// MemoryBackend implements SecretsBackend interface using in-memory storage
type MemoryBackend struct {
	secrets map[string]*EncryptedSecret
	logger  *slog.Logger
	mu      sync.RWMutex
}

// NewMemoryBackend creates a new in-memory secrets backend
func NewMemoryBackend(logger *slog.Logger) SecretsBackend {
	return &MemoryBackend{
		secrets: make(map[string]*EncryptedSecret),
		logger:  logger,
	}
}

// Store stores a secret in memory
func (mb *MemoryBackend) Store(ctx context.Context, key string, value *EncryptedSecret) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	mb.secrets[key] = value
	mb.logger.Debug("Secret stored in memory", "key", key)
	return nil
}

// Retrieve retrieves a secret from memory
func (mb *MemoryBackend) Retrieve(ctx context.Context, key string) (*EncryptedSecret, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	secret, exists := mb.secrets[key]
	if !exists {
		return nil, fmt.Errorf("secret not found: %s", key)
	}

	return secret, nil
}

// Delete deletes a secret from memory
func (mb *MemoryBackend) Delete(ctx context.Context, key string) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if _, exists := mb.secrets[key]; !exists {
		return fmt.Errorf("secret not found: %s", key)
	}

	delete(mb.secrets, key)
	mb.logger.Debug("Secret deleted from memory", "key", key)
	return nil
}

// List lists all secret keys in memory
func (mb *MemoryBackend) List(ctx context.Context) ([]string, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	keys := make([]string, 0, len(mb.secrets))
	for key := range mb.secrets {
		keys = append(keys, key)
	}

	return keys, nil
}

// Exists checks if a secret exists in memory
func (mb *MemoryBackend) Exists(ctx context.Context, key string) (bool, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()

	_, exists := mb.secrets[key]
	return exists, nil
}

// Rotate rotates a secret to a new version in memory
func (mb *MemoryBackend) Rotate(ctx context.Context, key string, newValue *EncryptedSecret) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	if _, exists := mb.secrets[key]; !exists {
		return fmt.Errorf("cannot rotate non-existent secret: %s", key)
	}

	mb.secrets[key] = newValue
	mb.logger.Debug("Secret rotated in memory", "key", key)
	return nil
}