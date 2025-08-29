package config

import "github.com/thc1006/nephoran-intent-operator/pkg/interfaces"

// APIKeys is an alias to interfaces.APIKeys for backward compatibility within the config package
type APIKeys = interfaces.APIKeys

// NewAPIKeys creates a new APIKeys instance
func NewAPIKeys() *APIKeys {
	return &APIKeys{}
}