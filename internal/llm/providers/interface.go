// Package providers defines the LLM provider interface for the nephoran-intent-operator.
// This package provides a clean abstraction for transforming natural language input
// into structured NetworkIntent JSON that conforms to the intent.schema.json contract.
package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Provider represents the core interface for LLM providers that convert
// natural language text into structured NetworkIntent JSON.
type Provider interface {
	// ProcessIntent converts free-form text input into structured JSON
	// that conforms to docs/contracts/intent.schema.json.
	// The returned JSON must be valid against the intent schema.
	ProcessIntent(ctx context.Context, input string) (*IntentResponse, error)

	// GetProviderInfo returns metadata about the provider implementation.
	GetProviderInfo() ProviderInfo

	// ValidateConfig validates the provider's configuration.
	ValidateConfig() error

	// Close performs cleanup of resources (connections, clients, etc.).
	Close() error
}

// IntentResponse represents the structured response from an LLM provider.
// The JSON field contains the parsed intent that conforms to intent.schema.json.
type IntentResponse struct {
	// JSON contains the structured NetworkIntent that conforms to intent.schema.json
	JSON json.RawMessage `json:"json"`

	// Metadata contains additional information about the processing
	Metadata ResponseMetadata `json:"metadata"`
}

// ResponseMetadata contains additional information about the LLM processing.
type ResponseMetadata struct {
	// Provider identifies which provider processed the request
	Provider string `json:"provider"`

	// Model identifies the specific model used (if applicable)
	Model string `json:"model,omitempty"`

	// ProcessingTime records how long the processing took
	ProcessingTime time.Duration `json:"processing_time"`

	// TokensUsed records token consumption (if applicable)
	TokensUsed int `json:"tokens_used,omitempty"`

	// Confidence represents the provider's confidence in the result (0.0-1.0)
	Confidence float64 `json:"confidence,omitempty"`

	// Warnings contains any warnings from the processing
	Warnings []string `json:"warnings,omitempty"`
}

// ProviderInfo contains metadata about a provider implementation.
type ProviderInfo struct {
	// Name is the provider identifier (OFFLINE, OPENAI, ANTHROPIC)
	Name string `json:"name"`

	// Version is the provider implementation version
	Version string `json:"version"`

	// Description provides a human-readable description
	Description string `json:"description"`

	// RequiresAuth indicates if the provider needs authentication
	RequiresAuth bool `json:"requires_auth"`

	// SupportedFeatures lists capabilities of this provider
	SupportedFeatures []string `json:"supported_features"`
}

// ProviderType represents the available LLM provider types.
type ProviderType string

const (
	// ProviderTypeOffline represents the deterministic offline provider
	ProviderTypeOffline ProviderType = "OFFLINE"

	// ProviderTypeOpenAI represents the OpenAI provider
	ProviderTypeOpenAI ProviderType = "OPENAI"

	// ProviderTypeAnthropic represents the Anthropic provider
	ProviderTypeAnthropic ProviderType = "ANTHROPIC"
)

// IsValid checks if the provider type is supported.
func (pt ProviderType) IsValid() bool {
	switch pt {
	case ProviderTypeOffline, ProviderTypeOpenAI, ProviderTypeAnthropic:
		return true
	default:
		return false
	}
}

// String returns the string representation of the provider type.
func (pt ProviderType) String() string {
	return string(pt)
}

// Config holds configuration for provider initialization.
type Config struct {
	// Type specifies which provider to use
	Type ProviderType `json:"type"`

	// APIKey for authenticated providers (OpenAI, Anthropic)
	APIKey string `json:"api_key,omitempty"`

	// BaseURL allows overriding the default API endpoint
	BaseURL string `json:"base_url,omitempty"`

	// Model specifies the model to use (provider-specific)
	Model string `json:"model,omitempty"`

	// Timeout for API requests
	Timeout time.Duration `json:"timeout"`

	// MaxRetries for failed requests
	MaxRetries int `json:"max_retries"`

	// Additional provider-specific configuration
	Extra map[string]interface{} `json:"extra,omitempty"`
}

// Validate ensures the configuration is valid for the specified provider type.
func (c *Config) Validate() error {
	if !c.Type.IsValid() {
		return fmt.Errorf("provider type %s: %w", c.Type, ErrProviderNotSupported)
	}

	// Validate authenticated providers have API keys
	if c.Type == ProviderTypeOpenAI || c.Type == ProviderTypeAnthropic {
		if c.APIKey == "" {
			return fmt.Errorf("API key is required for provider %s: %w", c.Type, ErrInvalidConfiguration)
		}
	}

	// Validate timeout - set default if zero, error if negative
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must be positive: %w", ErrInvalidConfiguration)
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second // Default timeout if not set
	}

	// Validate max retries
	if c.MaxRetries < 0 {
		c.MaxRetries = 3 // Default retries
	}

	return nil
}

// Factory interface for creating providers based on configuration.
type Factory interface {
	// CreateProvider creates a new provider instance based on the config.
	CreateProvider(config *Config) (Provider, error)

	// GetSupportedProviders returns a list of supported provider types.
	GetSupportedProviders() []ProviderType

	// ValidateProviderConfig validates configuration for a specific provider type.
	ValidateProviderConfig(providerType ProviderType, config *Config) error
}

// Common errors that providers may return.
var (
	// ErrProviderNotSupported indicates the requested provider is not available
	ErrProviderNotSupported = errors.New("provider not supported")

	// ErrInvalidConfiguration indicates the provider configuration is invalid
	ErrInvalidConfiguration = errors.New("invalid provider configuration")

	// ErrAuthenticationFailed indicates authentication with the provider failed
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrInvalidInput indicates the input text cannot be processed
	ErrInvalidInput = errors.New("invalid input")

	// ErrIntentParsing indicates the LLM output could not be parsed as valid intent JSON
	ErrIntentParsing = errors.New("failed to parse intent from LLM response")

	// ErrSchemaValidation indicates the generated JSON doesn't conform to intent.schema.json
	ErrSchemaValidation = errors.New("generated intent does not conform to schema")

	// ErrRateLimited indicates the provider has rate limited the request
	ErrRateLimited = errors.New("rate limited")

	// ErrQuotaExceeded indicates the provider quota has been exceeded
	ErrQuotaExceeded = errors.New("quota exceeded")

	// ErrProviderUnavailable indicates the provider service is unavailable
	ErrProviderUnavailable = errors.New("provider unavailable")
)

// IsRetryableError checks if an error indicates a retryable condition.
func IsRetryableError(err error) bool {
	return errors.Is(err, ErrRateLimited) ||
		errors.Is(err, ErrProviderUnavailable)
}

// IsAuthError checks if an error is authentication-related.
func IsAuthError(err error) bool {
	return errors.Is(err, ErrAuthenticationFailed)
}

// IsConfigError checks if an error is configuration-related.
func IsConfigError(err error) bool {
	return errors.Is(err, ErrInvalidConfiguration) ||
		errors.Is(err, ErrProviderNotSupported)
}