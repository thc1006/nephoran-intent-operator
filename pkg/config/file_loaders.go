package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nephio-project/nephoran-intent-operator/pkg/interfaces"
)

// LoadLLMAPIKeyFromFile loads the LLM API key from file or environment variable.

func LoadLLMAPIKeyFromFile(backendType string, logger interfaces.AuditLogger) (string, error) {

	// First try environment variable.

	envVar := fmt.Sprintf("%s_API_KEY", strings.ToUpper(backendType))

	if apiKey := os.Getenv(envVar); apiKey != "" {

		if logger != nil {

			logger.LogSecretAccess("api_key", fmt.Sprintf("env:%s", envVar), "system", "", true, nil)

		}

		return apiKey, nil

	}

	// Try loading from file.

	homeDir, err := os.UserHomeDir()

	if err != nil {

		return "", fmt.Errorf("failed to get home directory: %w", err)

	}

	keyFile := filepath.Join(homeDir, ".nephoran", fmt.Sprintf("%s_api_key", strings.ToLower(backendType)))

	if data, err := os.ReadFile(keyFile); err == nil {

		apiKey := strings.TrimSpace(string(data))

		if apiKey != "" {

			if logger != nil {

				logger.LogSecretAccess("api_key", fmt.Sprintf("file:%s", keyFile), "system", "", true, nil)

			}

			return apiKey, nil

		}

	}

	// Try generic LLM_API_KEY as fallback.

	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {

		if logger != nil {

			logger.LogSecretAccess("api_key", "env:LLM_API_KEY", "system", "", true, nil)

		}

		return apiKey, nil

	}

	err = fmt.Errorf("no API key found for backend %s", backendType)

	if logger != nil {

		logger.LogSecretAccess("api_key", backendType, "system", "", false, err)

	}

	return "", err

}

// LoadAPIKeyFromFile loads the application API key from file or environment variable.

func LoadAPIKeyFromFile(logger interfaces.AuditLogger) (string, error) {

	// First try environment variable.

	if apiKey := os.Getenv("API_KEY"); apiKey != "" {

		if logger != nil {

			logger.LogAPIKeyValidation("application", "environment", true, nil)

		}

		return apiKey, nil

	}

	// Try loading from file.

	homeDir, err := os.UserHomeDir()

	if err != nil {

		return "", fmt.Errorf("failed to get home directory: %w", err)

	}

	keyFile := filepath.Join(homeDir, ".nephoran", "api_key")

	if data, err := os.ReadFile(keyFile); err == nil {

		apiKey := strings.TrimSpace(string(data))

		if apiKey != "" {

			if logger != nil {

				logger.LogAPIKeyValidation("application", "file", true, nil)

			}

			return apiKey, nil

		}

	}

	err = fmt.Errorf("no API key found")

	if logger != nil {

		logger.LogAPIKeyValidation("application", "none", false, err)

	}

	return "", err

}

// LoadJWTSecretKeyFromFile loads the JWT secret key from file or environment variable.

func LoadJWTSecretKeyFromFile(logger interfaces.AuditLogger) (string, error) {

	// First try environment variable.

	if secretKey := os.Getenv("JWT_SECRET_KEY"); secretKey != "" {

		if logger != nil {

			logger.LogSecretAccess("jwt_secret", "env:JWT_SECRET_KEY", "system", "", true, nil)

		}

		return secretKey, nil

	}

	// Try loading from file.

	homeDir, err := os.UserHomeDir()

	if err != nil {

		return "", fmt.Errorf("failed to get home directory: %w", err)

	}

	keyFile := filepath.Join(homeDir, ".nephoran", "jwt_secret")

	if data, err := os.ReadFile(keyFile); err == nil {

		secretKey := strings.TrimSpace(string(data))

		if secretKey != "" {

			if logger != nil {

				logger.LogSecretAccess("jwt_secret", fmt.Sprintf("file:%s", keyFile), "system", "", true, nil)

			}

			return secretKey, nil

		}

	}

	// Generate a default secret key for development (not recommended for production).

	defaultSecret := "default-development-secret-key-change-in-production"

	if logger != nil {

		logger.LogSecretAccess("jwt_secret", "default_generated", "system", "", true, nil)

	}

	return defaultSecret, nil

}
