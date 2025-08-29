// Package interfaces provides common interfaces and types for the Nephoran Intent Operator.


package interfaces



import (

	"context"

	"time"

)



// AuditLevel represents the severity level of audit events.

type AuditLevel int



const (

	// AuditLevelInfo represents informational audit events.

	AuditLevelInfo AuditLevel = iota

	// AuditLevelWarn holds auditlevelwarn value.

	AuditLevelWarn

	// AuditLevelError holds auditlevelerror value.

	AuditLevelError

	// AuditLevelCritical holds auditlevelcritical value.

	AuditLevelCritical

)



// SecretManager interface defines methods for secure secret operations.

type SecretManager interface {

	// GetSecretValue retrieves a value from a secret source (Kubernetes or environment).

	GetSecretValue(ctx context.Context, secretName, key, envVarName string) (string, error)



	// CreateSecretFromEnvVars creates a secret from environment variables.

	CreateSecretFromEnvVars(ctx context.Context, secretName string, envVarMapping map[string]string) error



	// UpdateSecret updates an existing secret.

	UpdateSecret(ctx context.Context, secretName string, data map[string][]byte) error



	// SecretExists checks if a secret exists.

	SecretExists(ctx context.Context, secretName string) bool



	// RotateSecret rotates a secret value.

	RotateSecret(ctx context.Context, secretName, secretKey, newValue string) error



	// GetSecretRotationInfo returns information about when a secret was last rotated.

	GetSecretRotationInfo(ctx context.Context, secretName string) (map[string]string, error)

}



// AuditLogger interface defines methods for security audit logging.

type AuditLogger interface {

	// LogSecretAccess logs when secrets are accessed.

	LogSecretAccess(secretType, source, userID, sessionID string, success bool, err error)



	// LogAuthenticationAttempt logs authentication attempts.

	LogAuthenticationAttempt(provider, userID, ipAddress, userAgent string, success bool, err error)



	// LogSecretRotation logs secret rotation events.

	LogSecretRotation(secretName, rotationType, userID string, success bool, err error)



	// LogAPIKeyValidation logs API key validation events.

	LogAPIKeyValidation(keyType, provider string, success bool, err error)



	// LogUnauthorizedAccess logs unauthorized access attempts.

	LogUnauthorizedAccess(resource, userID, ipAddress, userAgent, reason string)



	// LogSecurityViolation logs security violations.

	LogSecurityViolation(violationType, description, userID, ipAddress string, severity AuditLevel)



	// SetEnabled enables or disables audit logging.

	SetEnabled(enabled bool)



	// IsEnabled returns whether audit logging is enabled.

	IsEnabled() bool



	// Close closes the audit logger and any open files.

	Close() error

}



// ConfigProvider interface defines methods for configuration access.

type ConfigProvider interface {

	// GetRAGAPIURL returns the appropriate RAG API URL based on environment.

	GetRAGAPIURL(useInternal bool) string



	// GetLLMProcessorURL returns the LLM processor URL.

	GetLLMProcessorURL() string



	// GetLLMProcessorTimeout returns the LLM processor timeout.

	GetLLMProcessorTimeout() time.Duration



	// GetGitRepoURL returns the Git repository URL.

	GetGitRepoURL() string



	// GetGitToken returns the Git token.

	GetGitToken() string



	// GetGitBranch returns the Git branch.

	GetGitBranch() string



	// GetWeaviateURL returns the Weaviate URL.

	GetWeaviateURL() string



	// GetWeaviateIndex returns the Weaviate index name.

	GetWeaviateIndex() string



	// GetOpenAIAPIKey returns the OpenAI API key.

	GetOpenAIAPIKey() string



	// GetOpenAIModel returns the OpenAI model.

	GetOpenAIModel() string



	// GetOpenAIEmbeddingModel returns the OpenAI embedding model.

	GetOpenAIEmbeddingModel() string



	// GetNamespace returns the Kubernetes namespace.

	GetNamespace() string



	// Validate checks that required configuration is present.

	Validate() error

}



// APIKeys holds all API keys used by the system.

type APIKeys struct {

	OpenAI    string

	Weaviate  string

	Generic   string

	JWTSecret string

}



// IsEmpty returns true if all API keys are empty.

func (ak *APIKeys) IsEmpty() bool {

	return ak.OpenAI == "" && ak.Weaviate == "" && ak.Generic == "" && ak.JWTSecret == ""

}



// RotationResult contains the result of a secret rotation operation.

type RotationResult struct {

	SecretName    string    `json:"secret_name"`

	RotationType  string    `json:"rotation_type"`

	Success       bool      `json:"success"`

	OldSecretHash string    `json:"old_secret_hash,omitempty"`

	NewSecretHash string    `json:"new_secret_hash,omitempty"`

	BackupCreated bool      `json:"backup_created"`

	Timestamp     time.Time `json:"timestamp"`

	Error         string    `json:"error,omitempty"`

}

