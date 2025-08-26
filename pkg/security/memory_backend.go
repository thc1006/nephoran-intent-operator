// Package security implements in-memory secrets backend for testing and development
package security

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// MemoryBackend implements SecretsBackend interface using in-memory storage
type MemoryBackend struct {
	secrets map[string]*EncryptedSecret
	mu      sync.RWMutex
}

// NewMemoryBackend creates a new memory backend
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		secrets: make(map[string]*EncryptedSecret),
	}
}

// Store stores a secret in memory
func (mb *MemoryBackend) Store(ctx context.Context, key string, value *EncryptedSecret) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	
	mb.secrets[key] = value
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
	return nil
}

// List lists all secrets with optional prefix filter
func (mb *MemoryBackend) List(ctx context.Context, prefix string) ([]string, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	
	var result []string
	for key := range mb.secrets {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			result = append(result, key)
		}
	}
	
	return result, nil
}

// Backup creates a backup of all secrets
func (mb *MemoryBackend) Backup(ctx context.Context) ([]byte, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	
	return json.Marshal(mb.secrets)
}

// Restore restores secrets from backup data
func (mb *MemoryBackend) Restore(ctx context.Context, data []byte) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	
	var secrets map[string]*EncryptedSecret
	if err := json.Unmarshal(data, &secrets); err != nil {
		return fmt.Errorf("failed to unmarshal backup data: %w", err)
	}
	
	mb.secrets = secrets
	return nil
}

// Health checks the health of the memory backend
func (mb *MemoryBackend) Health(ctx context.Context) error {
	return nil // Memory backend is always healthy
}

// Close closes the memory backend
func (mb *MemoryBackend) Close() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	
	// Clear all secrets
	for k := range mb.secrets {
		delete(mb.secrets, k)
	}
	
	return nil
}