//go:build !disable_rag

package config

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/thc1006/nephoran-intent-operator/pkg/interfaces"
)

// ExampleUsageKubernetesSecretManager demonstrates how to use the KubernetesSecretManager
func ExampleUsageKubernetesSecretManager() error {
	// Create a new secret manager for the "nephoran-system" namespace
	secretManager, err := NewSecretManager("nephoran-system")
	if err != nil {
		return fmt.Errorf("failed to create secret manager: %w", err)
	}

	// Set a custom logger (optional)
	logger := slog.Default()
	secretManager.SetLogger(logger)

	ctx := context.Background()

	// Check if a secret exists
	exists := secretManager.SecretExists(ctx, "llm-api-keys")
	if !exists {
		logger.Info("Secret does not exist, creating it...")
		
		// Create API keys structure
		apiKeys := &interfaces.APIKeys{
			OpenAI:    "sk-example-openai-key",
			Anthropic: "sk-ant-example-key",
			GoogleAI:  "example-google-ai-key",
			Weaviate:  "example-weaviate-key",
			Generic:   "example-generic-key",
			JWTSecret: "example-jwt-secret-key",
		}

		// Create the secret in Kubernetes
		err = secretManager.CreateOrUpdateSecretFromAPIKeys(ctx, "llm-api-keys", apiKeys)
		if err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}
		
		logger.Info("Secret created successfully")
	}

	// Retrieve API keys from Kubernetes
	apiKeys, err := secretManager.GetAPIKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to get API keys: %w", err)
	}

	// Check if we have any keys
	if apiKeys.IsEmpty() {
		logger.Warn("No API keys found")
	} else {
		logger.Info("Successfully loaded API keys",
			slog.Bool("has_openai", apiKeys.OpenAI != ""),
			slog.Bool("has_anthropic", apiKeys.Anthropic != ""),
			slog.Bool("has_google_ai", apiKeys.GoogleAI != ""),
			slog.Bool("has_weaviate", apiKeys.Weaviate != ""),
			slog.Bool("has_generic", apiKeys.Generic != ""),
			slog.Bool("has_jwt_secret", apiKeys.JWTSecret != ""),
		)
	}

	return nil
}

// ExampleFallbackBehavior demonstrates the fallback behavior when Kubernetes client is not available
func ExampleFallbackBehavior() error {
	// This example shows how the secret manager gracefully falls back to mounted secrets
	// when the Kubernetes client-go API is not available (e.g., no RBAC permissions)
	
	secretManager, err := NewSecretManager("nephoran-system")
	if err != nil {
		return fmt.Errorf("failed to create secret manager: %w", err)
	}

	ctx := context.Background()

	// The GetAPIKeys method will first try the client-go API
	// If that fails, it will automatically fall back to the mounted secrets approach
	apiKeys, err := secretManager.GetAPIKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to get API keys: %w", err)
	}

	// The secret manager automatically logs which method was successful
	if !apiKeys.IsEmpty() {
		fmt.Printf("Successfully loaded API keys via fallback mechanism\n")
	}

	return nil
}

// ExampleBackwardCompatibility shows how existing code using config.APIKeys still works
func ExampleBackwardCompatibility() {
	// This demonstrates that existing code continues to work unchanged
	// because we created a type alias: type APIKeys = interfaces.APIKeys

	// Create APIKeys using the alias
	apiKeys := &APIKeys{
		OpenAI:    "sk-example-key",
		JWTSecret: "example-jwt-secret",
	}

	// Use methods from interfaces.APIKeys
	isEmpty := apiKeys.IsEmpty() // This method is available through the alias
	if !isEmpty {
		fmt.Printf("API keys are populated: OpenAI present: %t, JWT present: %t\n",
			apiKeys.OpenAI != "", apiKeys.JWTSecret != "")
	}

	// Create using helper function
	newKeys := NewAPIKeys()
	if newKeys.IsEmpty() {
		fmt.Printf("New API keys are initially empty\n")
	}
}