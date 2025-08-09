package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/thc1006/nephoran-intent-operator/pkg/interfaces"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SecretRotationManager handles automatic secret rotation
type SecretRotationManager struct {
	secretManager interfaces.SecretManager
	k8sClient     kubernetes.Interface
	namespace     string
	logger        logr.Logger
	auditLogger   *AuditLogger
}

// RotationConfig holds configuration for secret rotation
type SecretRotationConfig struct {
	SecretName       string        `json:"secret_name"`
	RotationPeriod   time.Duration `json:"rotation_period"`
	BackupCount      int           `json:"backup_count"`
	NotifyBeforeDays int           `json:"notify_before_days"`
}

// NewSecretRotationManager creates a new secret rotation manager
func NewSecretRotationManager(secretManager interfaces.SecretManager, k8sClient kubernetes.Interface, namespace string, auditLogger *AuditLogger) *SecretRotationManager {
	return &SecretRotationManager{
		secretManager: secretManager,
		k8sClient:     k8sClient,
		namespace:     namespace,
		logger:        log.Log.WithName("secret-rotation"),
		auditLogger:   auditLogger,
	}
}

// RotateJWTSecret rotates the JWT secret key
func (srm *SecretRotationManager) RotateJWTSecret(ctx context.Context, userID string) (*interfaces.RotationResult, error) {
	secretName := "auth-keys"
	secretKey := "jwt-secret"

	result := &interfaces.RotationResult{
		SecretName:   secretName,
		RotationType: "jwt_secret",
		Timestamp:    time.Now().UTC(),
	}

	// Get current secret for backup
	currentSecret, err := srm.getCurrentSecret(ctx, secretName, secretKey)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get current secret: %v", err)
		srm.auditRotation(result, userID, false, err)
		return result, err
	}

	if currentSecret != "" {
		result.OldSecretHash = hashSecret(currentSecret)

		// Create backup
		if err := srm.createSecretBackup(ctx, secretName, secretKey, currentSecret); err != nil {
			srm.logger.Error(err, "Failed to create secret backup", "secret", secretName)
		} else {
			result.BackupCreated = true
		}
	}

	// Generate new JWT secret (256-bit)
	newSecret, err := generateJWTSecret()
	if err != nil {
		result.Error = fmt.Sprintf("failed to generate new JWT secret: %v", err)
		srm.auditRotation(result, userID, false, err)
		return result, err
	}

	result.NewSecretHash = hashSecret(newSecret)

	// Update Kubernetes secret
	if err := srm.updateKubernetesSecret(ctx, secretName, secretKey, newSecret); err != nil {
		result.Error = fmt.Sprintf("failed to update Kubernetes secret: %v", err)
		srm.auditRotation(result, userID, false, err)
		return result, err
	}

	// Update file-based secret if it exists
	if err := srm.updateFileSecret("/secrets/jwt/jwt-secret-key", newSecret); err != nil {
		srm.logger.Error(err, "Failed to update file-based secret", "path", "/secrets/jwt/jwt-secret-key")
	}

	result.Success = true
	srm.auditRotation(result, userID, true, nil)

	return result, nil
}

// RotateOAuth2ClientSecret rotates OAuth2 client secrets
func (srm *SecretRotationManager) RotateOAuth2ClientSecret(ctx context.Context, provider, newClientSecret, userID string) (*interfaces.RotationResult, error) {
	secretName := "oauth2-secrets"
	secretKey := fmt.Sprintf("%s-client-secret", strings.ToLower(provider))

	result := &interfaces.RotationResult{
		SecretName:   secretName,
		RotationType: "oauth2_client_secret",
		Timestamp:    time.Now().UTC(),
	}

	// Validate new secret
	if newClientSecret == "" {
		err := fmt.Errorf("new client secret cannot be empty")
		result.Error = err.Error()
		srm.auditRotation(result, userID, false, err)
		return result, err
	}

	// Get current secret for backup
	currentSecret, err := srm.getCurrentSecret(ctx, secretName, secretKey)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get current secret: %v", err)
		srm.auditRotation(result, userID, false, err)
		return result, err
	}

	if currentSecret != "" {
		result.OldSecretHash = hashSecret(currentSecret)

		// Create backup
		if err := srm.createSecretBackup(ctx, secretName, secretKey, currentSecret); err != nil {
			srm.logger.Error(err, "Failed to create secret backup", "secret", secretName)
		} else {
			result.BackupCreated = true
		}
	}

	result.NewSecretHash = hashSecret(newClientSecret)

	// Update Kubernetes secret
	if err := srm.updateKubernetesSecret(ctx, secretName, secretKey, newClientSecret); err != nil {
		result.Error = fmt.Sprintf("failed to update Kubernetes secret: %v", err)
		srm.auditRotation(result, userID, false, err)
		return result, err
	}

	// Update file-based secret if it exists
	filePath := fmt.Sprintf("/secrets/oauth2/%s-client-secret", strings.ToLower(provider))
	if err := srm.updateFileSecret(filePath, newClientSecret); err != nil {
		srm.logger.Error(err, "Failed to update file-based secret", "path", filePath)
	}

	result.Success = true
	srm.auditRotation(result, userID, true, nil)

	return result, nil
}

// RotateAPIKey rotates API keys for LLM providers
func (srm *SecretRotationManager) RotateAPIKey(ctx context.Context, provider, newAPIKey, userID string) (*interfaces.RotationResult, error) {
	secretName := "llm-api-keys"
	secretKey := fmt.Sprintf("%s-api-key", strings.ToLower(provider))

	result := &interfaces.RotationResult{
		SecretName:   secretName,
		RotationType: "api_key",
		Timestamp:    time.Now().UTC(),
	}

	// Validate new API key format
	if err := srm.validateAPIKey(provider, newAPIKey); err != nil {
		result.Error = fmt.Sprintf("invalid API key format: %v", err)
		srm.auditRotation(result, userID, false, err)
		return result, err
	}

	// Get current secret for backup
	currentSecret, err := srm.getCurrentSecret(ctx, secretName, secretKey)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get current secret: %v", err)
		srm.auditRotation(result, userID, false, err)
		return result, err
	}

	if currentSecret != "" {
		result.OldSecretHash = hashSecret(currentSecret)

		// Create backup
		if err := srm.createSecretBackup(ctx, secretName, secretKey, currentSecret); err != nil {
			srm.logger.Error(err, "Failed to create secret backup", "secret", secretName)
		} else {
			result.BackupCreated = true
		}
	}

	result.NewSecretHash = hashSecret(newAPIKey)

	// Update Kubernetes secret
	if err := srm.updateKubernetesSecret(ctx, secretName, secretKey, newAPIKey); err != nil {
		result.Error = fmt.Sprintf("failed to update Kubernetes secret: %v", err)
		srm.auditRotation(result, userID, false, err)
		return result, err
	}

	// Update file-based secret if it exists
	filePath := fmt.Sprintf("/secrets/llm/%s-api-key", strings.ToLower(provider))
	if err := srm.updateFileSecret(filePath, newAPIKey); err != nil {
		srm.logger.Error(err, "Failed to update file-based secret", "path", filePath)
	}

	result.Success = true
	srm.auditRotation(result, userID, true, nil)

	return result, nil
}

// getCurrentSecret retrieves the current secret value
func (srm *SecretRotationManager) getCurrentSecret(ctx context.Context, secretName, secretKey string) (string, error) {
	if srm.k8sClient == nil {
		return "", fmt.Errorf("no Kubernetes client available")
	}

	secret, err := srm.k8sClient.CoreV1().Secrets(srm.namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	if value, exists := secret.Data[secretKey]; exists {
		return string(value), nil
	}

	return "", fmt.Errorf("key %s not found in secret %s", secretKey, secretName)
}

// createSecretBackup creates a backup of the current secret
func (srm *SecretRotationManager) createSecretBackup(ctx context.Context, secretName, secretKey, secretValue string) error {
	if srm.k8sClient == nil {
		return fmt.Errorf("no Kubernetes client available")
	}

	backupName := fmt.Sprintf("%s-backup-%s", secretName, time.Now().Format("20060102-150405"))

	backup := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupName,
			Namespace: srm.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "nephoran-intent-operator",
				"app.kubernetes.io/component": "secret-backup",
				"backup-of":                   secretName,
				"backup-key":                  secretKey,
			},
			Annotations: map[string]string{
				"backup-timestamp": time.Now().UTC().Format(time.RFC3339),
				"original-secret":  secretName,
				"original-key":     secretKey,
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			secretKey: []byte(secretValue),
		},
	}

	_, err := srm.k8sClient.CoreV1().Secrets(srm.namespace).Create(ctx, backup, metav1.CreateOptions{})
	return err
}

// updateKubernetesSecret updates a secret in Kubernetes
func (srm *SecretRotationManager) updateKubernetesSecret(ctx context.Context, secretName, secretKey, newValue string) error {
	if srm.k8sClient == nil {
		return fmt.Errorf("no Kubernetes client available")
	}

	// Get existing secret
	secret, err := srm.k8sClient.CoreV1().Secrets(srm.namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		// Create new secret if it doesn't exist
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: srm.namespace,
				Labels: map[string]string{
					"app.kubernetes.io/name":      "nephoran-intent-operator",
					"app.kubernetes.io/component": "secrets",
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: make(map[string][]byte),
		}
	}

	// Update the specific key
	secret.Data[secretKey] = []byte(newValue)

	// Add rotation metadata
	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}
	secret.Annotations[fmt.Sprintf("%s-rotated-at", secretKey)] = time.Now().UTC().Format(time.RFC3339)

	// Update or create the secret
	if err != nil {
		_, err = srm.k8sClient.CoreV1().Secrets(srm.namespace).Create(ctx, secret, metav1.CreateOptions{})
	} else {
		_, err = srm.k8sClient.CoreV1().Secrets(srm.namespace).Update(ctx, secret, metav1.UpdateOptions{})
	}

	return err
}

// updateFileSecret updates a file-based secret
func (srm *SecretRotationManager) updateFileSecret(filePath, newValue string) error {
	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write the new secret with secure permissions
	if err := os.WriteFile(filePath, []byte(newValue), 0600); err != nil {
		return fmt.Errorf("failed to write secret to %s: %w", filePath, err)
	}

	return nil
}

// validateAPIKey validates API key format for different providers
func (srm *SecretRotationManager) validateAPIKey(provider, apiKey string) error {
	switch strings.ToLower(provider) {
	case "openai":
		if !strings.HasPrefix(apiKey, "sk-") || len(apiKey) < 40 {
			return fmt.Errorf("invalid OpenAI API key format")
		}
	case "weaviate":
		if len(apiKey) < 16 {
			return fmt.Errorf("Weaviate API key too short")
		}
	default:
		if len(apiKey) < 8 {
			return fmt.Errorf("API key too short")
		}
	}
	return nil
}

// generateJWTSecret generates a new JWT secret
func generateJWTSecret() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// hashSecret creates a hash of the secret for audit purposes
func hashSecret(secret string) string {
	if len(secret) <= 8 {
		return strings.Repeat("*", len(secret))
	}
	return secret[:4] + strings.Repeat("*", len(secret)-8) + secret[len(secret)-4:]
}

// auditRotation logs the rotation event
func (srm *SecretRotationManager) auditRotation(result *interfaces.RotationResult, userID string, success bool, err error) {
	if srm.auditLogger != nil {
		srm.auditLogger.LogSecretRotation(result.SecretName, result.RotationType, userID, success, err)
	}
}
