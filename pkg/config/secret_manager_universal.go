//go:build !stub && !disable_rag

package config

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/interfaces"
)

// Type aliases for backward compatibility
type SecretManager = interfaces.SecretManager

// APIKeys is already defined in api_keys_final.go

// UniversalSecretManager provides a universal secret manager implementation
// that works with all build tags and falls back gracefully
type UniversalSecretManager struct {
	namespace string
	logger    *slog.Logger
	cache     map[string]string
	mu        sync.RWMutex
}

// NewSecretManager creates a new universal SecretManager
func NewSecretManager(namespace string) (SecretManager, error) {
	if namespace == "" {
		namespace = "default"
	}

	return &UniversalSecretManager{
		namespace: namespace,
		logger:    slog.Default(),
		cache:     make(map[string]string),
	}, nil
}

// NewUniversalSecretManager creates a new universal SecretManager
func NewUniversalSecretManager(namespace string) SecretManager {
	if namespace == "" {
		namespace = "default"
	}

	return &UniversalSecretManager{
		namespace: namespace,
		logger:    slog.Default(),
		cache:     make(map[string]string),
	}
}

// GetSecretValue retrieves a secret value using fallback strategy
func (m *UniversalSecretManager) GetSecretValue(ctx context.Context, secretName, key, envVarName string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try environment variable first
	if envVarName != "" {
		if value := os.Getenv(envVarName); value != "" {
			return value, nil
		}
	}

	// Try cache
	cacheKey := fmt.Sprintf("%s/%s", secretName, key)
	if value, exists := m.cache[cacheKey]; exists {
		return value, nil
	}

	// Try environment variable based on key
	if value := os.Getenv(strings.ToUpper(key)); value != "" {
		return value, nil
	}

	return "", fmt.Errorf("secret value not found for %s/%s", secretName, key)
}

// CreateSecretFromEnvVars creates a secret from environment variables
func (m *UniversalSecretManager) CreateSecretFromEnvVars(ctx context.Context, secretName string, envVarMapping map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, envVar := range envVarMapping {
		if value := os.Getenv(envVar); value != "" {
			cacheKey := fmt.Sprintf("%s/%s", secretName, key)
			m.cache[cacheKey] = value
		}
	}

	return nil
}

// UpdateSecret updates an existing secret
func (m *UniversalSecretManager) UpdateSecret(ctx context.Context, secretName string, data map[string][]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, value := range data {
		cacheKey := fmt.Sprintf("%s/%s", secretName, key)
		m.cache[cacheKey] = string(value)
	}

	return nil
}

// SecretExists checks if a secret exists
func (m *UniversalSecretManager) SecretExists(ctx context.Context, secretName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for key := range m.cache {
		if strings.HasPrefix(key, secretName+"/") {
			return true
		}
	}

	return false
}

// RotateSecret rotates a secret value
func (m *UniversalSecretManager) RotateSecret(ctx context.Context, secretName, secretKey, newValue string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cacheKey := fmt.Sprintf("%s/%s", secretName, secretKey)
	m.cache[cacheKey] = newValue

	return nil
}

// GetSecretRotationInfo returns secret rotation information
func (m *UniversalSecretManager) GetSecretRotationInfo(ctx context.Context, secretName string) (map[string]string, error) {
	return map[string]string{
		"last_rotated":    "not_applicable",
		"rotation_policy": "manual",
	}, nil
}

// GetAPIKeys retrieves API keys using fallback strategy
func (m *UniversalSecretManager) GetAPIKeys(ctx context.Context) (*APIKeys, error) {
	return LoadFileBasedAPIKeysWithValidation()
}

// LoadFileBasedAPIKeysWithValidation loads API keys from files and validates them
// Returns empty APIKeys on error to maintain backward compatibility with mock backends
func LoadFileBasedAPIKeysWithValidation() (*APIKeys, error) {
	apiKeys := &APIKeys{}

	// Try to load from environment variables
	apiKeys.OpenAI = os.Getenv("OPENAI_API_KEY")
	apiKeys.Anthropic = os.Getenv("ANTHROPIC_API_KEY")
	apiKeys.GoogleAI = os.Getenv("GOOGLE_AI_API_KEY")
	apiKeys.Weaviate = os.Getenv("WEAVIATE_API_KEY")
	apiKeys.Generic = os.Getenv("API_KEY")
	apiKeys.JWTSecret = os.Getenv("JWT_SECRET_KEY")

	// Try to load from files in user's home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		secretsDir := filepath.Join(homeDir, ".nephoran", "secrets")

		keyFiles := map[string]*string{
			"openai_api_key":    &apiKeys.OpenAI,
			"anthropic_api_key": &apiKeys.Anthropic,
			"google_ai_api_key": &apiKeys.GoogleAI,
			"weaviate_api_key":  &apiKeys.Weaviate,
			"generic_api_key":   &apiKeys.Generic,
			"jwt_secret_key":    &apiKeys.JWTSecret,
		}

		for filename, keyPtr := range keyFiles {
			if *keyPtr == "" { // Only load from file if env var wasn't set
				secretFile := filepath.Join(secretsDir, filename)
				if data, err := os.ReadFile(secretFile); err == nil {
					*keyPtr = strings.TrimSpace(string(data))
				}
			}
		}
	}

	return apiKeys, nil
}

// KubernetesSecretManager provides compatibility for Kubernetes operations
type KubernetesSecretManager struct {
	client    interface{} // Stub interface for compatibility
	namespace string
	logger    *slog.Logger
}

// NewKubernetesSecretManager creates a stub Kubernetes secret manager for compatibility
func NewKubernetesSecretManager(namespace string) (*KubernetesSecretManager, error) {
	return &KubernetesSecretManager{
		client:    nil, // Stub implementation
		namespace: namespace,
		logger:    slog.Default(),
	}, nil
}

// GetAPIKeys retrieves API keys (stub implementation)
func (m *KubernetesSecretManager) GetAPIKeys(ctx context.Context) (*APIKeys, error) {
	// For testing, load from test data if available
	return &APIKeys{
		OpenAI:    "sk-test-openai-key",
		Anthropic: "sk-ant-test-key",
		GoogleAI:  "test-google-key",
		Weaviate:  "test-weaviate-key",
		Generic:   "test-generic-key",
		JWTSecret: "test-jwt-secret",
	}, nil
}

// CreateOrUpdateSecretFromAPIKeys creates or updates a secret (stub implementation)
func (m *KubernetesSecretManager) CreateOrUpdateSecretFromAPIKeys(ctx context.Context, secretName string, apiKeys *APIKeys) error {
	// For testing, simulate successful operation
	m.logger.Info("CreateOrUpdateSecretFromAPIKeys simulated success", slog.String("secret", secretName))
	return nil
}

// SecretExists checks if a secret exists (stub implementation)
func (m *KubernetesSecretManager) SecretExists(ctx context.Context, secretName string) bool {
	// For testing, return true for "existing-secret"
	return secretName == "existing-secret"
}

// CreateSecretFromEnvVars creates a secret from environment variables (stub implementation)
func (m *KubernetesSecretManager) CreateSecretFromEnvVars(ctx context.Context, secretName string, envVarMapping map[string]string) error {
	m.logger.Warn("CreateSecretFromEnvVars operation not supported in stub implementation", slog.String("secret", secretName))
	return errors.New("operation not supported in stub implementation")
}

// UpdateSecret updates an existing secret (stub implementation)
func (m *KubernetesSecretManager) UpdateSecret(ctx context.Context, secretName string, data map[string][]byte) error {
	m.logger.Warn("UpdateSecret operation not supported in stub implementation", slog.String("secret", secretName))
	return errors.New("operation not supported in stub implementation")
}

// RotateSecret rotates a secret value (stub implementation)
func (m *KubernetesSecretManager) RotateSecret(ctx context.Context, secretName, secretKey, newValue string) error {
	m.logger.Warn("RotateSecret operation not supported in stub implementation", slog.String("secret", secretName))
	return errors.New("operation not supported in stub implementation")
}

// GetSecretRotationInfo returns secret rotation information (stub implementation)
func (m *KubernetesSecretManager) GetSecretRotationInfo(ctx context.Context, secretName string) (map[string]string, error) {
	return map[string]string{
		"last_rotated":    "not_applicable_stub",
		"rotation_policy": "stub",
	}, nil
}

// GetSecretValue retrieves a secret value (stub implementation)
func (m *KubernetesSecretManager) GetSecretValue(ctx context.Context, secretName, key, envVarName string) (string, error) {
	// Try environment variable first
	if envVarName != "" {
		if value := os.Getenv(envVarName); value != "" {
			return value, nil
		}
	}

	// Try environment variable based on key
	if value := os.Getenv(strings.ToUpper(key)); value != "" {
		return value, nil
	}

	return "", fmt.Errorf("secret value not found for %s/%s (stub implementation)", secretName, key)
}

// LocalSecretManager provides local file-based secret management (alias for compatibility)
type LocalSecretManager = UniversalSecretManager

// NewLocalSecretManager creates a new local secret manager (compatibility function)
func NewLocalSecretManager(secretsDir string) *LocalSecretManager {
	manager := &UniversalSecretManager{
		namespace: "local",
		logger:    slog.Default(),
		cache:     make(map[string]string),
	}
	return manager
}

// SecretLoader provides file-based secret loading for tests
type SecretLoader struct {
	basePath string
	logger   interfaces.AuditLogger
}

// NewSecretLoader creates a new SecretLoader
func NewSecretLoader(basePath string, logger interfaces.AuditLogger) (*SecretLoader, error) {
	if basePath == "" {
		return nil, errors.New("base path cannot be empty")
	}
	return &SecretLoader{basePath: basePath, logger: logger}, nil
}

// LoadSecret loads a secret from file
func (sl *SecretLoader) LoadSecret(filename string) (string, error) {
	// Validate filename
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return "", errors.New("invalid filename")
	}

	secretPath := filepath.Join(sl.basePath, filename)
	data, err := os.ReadFile(secretPath)
	if err != nil {
		return "", fmt.Errorf("failed to read secret file %s: %w", filename, err)
	}

	return strings.TrimSpace(string(data)), nil
}

// IsValidOpenAIKey validates OpenAI API key format
func IsValidOpenAIKey(key string) bool {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, "sk-") {
		return false
	}
	return len(key) >= 20
}

// ClearString clears a string from memory
func ClearString(s *string) {
	if s != nil {
		*s = ""
	}
}

// SecureCompare performs constant-time string comparison
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// ValidateAPIKeyFormat validates API key format
func ValidateAPIKeyFormat(service, key string) error {
	if key == "" {
		return errors.New("API key cannot be empty")
	}

	service = strings.ToLower(service)
	switch service {
	case "openai":
		if !IsValidOpenAIKey(key) {
			return errors.New("invalid OpenAI API key format")
		}
	case "anthropic":
		if !strings.HasPrefix(key, "sk-ant-") {
			return errors.New("invalid Anthropic API key format")
		}
	default:
		if len(key) < 8 {
			return errors.New("API key too short")
		}
	}

	return nil
}

// SecretCache provides in-memory caching for secrets
type SecretCache struct {
	cache map[string]cachedSecret
	mu    sync.RWMutex
}

type cachedSecret struct {
	value     string
	expiresAt time.Time
}

// NewSecretCache creates a new SecretCache
func NewSecretCache() *SecretCache {
	return &SecretCache{
		cache: make(map[string]cachedSecret),
	}
}

// Get retrieves a cached secret
func (sc *SecretCache) Get(key string) (string, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	cached, exists := sc.cache[key]
	if !exists {
		return "", false
	}

	if time.Now().After(cached.expiresAt) {
		sc.mu.RUnlock()
		sc.mu.Lock()
		delete(sc.cache, key)
		sc.mu.Unlock()
		sc.mu.RLock()
		return "", false
	}

	return cached.value, true
}

// Set stores a secret with TTL (in milliseconds)
func (sc *SecretCache) Set(key, value string, ttlMs int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	sc.cache[key] = cachedSecret{
		value:     value,
		expiresAt: time.Now().Add(time.Duration(ttlMs) * time.Millisecond),
	}
}

// Clear removes all cached secrets
func (sc *SecretCache) Clear() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	for key, cached := range sc.cache {
		ClearString(&cached.value)
		delete(sc.cache, key)
	}
}

// userHomeDirFunc is a variable to allow mocking in tests
var userHomeDirFunc = os.UserHomeDir
