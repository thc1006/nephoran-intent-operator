//go:build ignore
// +build ignore

// Package main demonstrates secure usage patterns for the LLM client.

// This file shows both secure production usage and controlled insecure development usage.

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

func main() {
	fmt.Println("LLM Client Security Usage Examples")

	fmt.Println("==================================")

	// Example 1: Secure Production Usage (Recommended).

	fmt.Println("\n1. Secure Production Usage (Default)")

	productionExample()

	// Example 2: Secure Production Usage with Explicit Config.

	fmt.Println("\n2. Secure Production Usage with Explicit Configuration")

	secureConfigExample()

	// Example 3: Development Usage with Self-Signed Certificates.

	fmt.Println("\n3. Development Usage (Insecure - Development Only)")

	developmentExample()

	// Example 4: Attempting Insecure Without Permission (Will Fail).

	fmt.Println("\n4. Security Violation Example (Will Panic)")

	securityViolationExample()
}

// productionExample shows the simplest secure usage.

func productionExample() {
	// This is the simplest and most secure way to create an LLM client.

	// It uses secure defaults with TLS verification enabled.

	client := llm.NewClient("https://api.openai.com/v1/chat/completions")

	fmt.Println("✓ Created secure client with default settings")

	fmt.Println("  - TLS verification: ENABLED")

	fmt.Println("  - Minimum TLS version: 1.2")

	fmt.Println("  - Secure cipher suites: ENABLED")

	// Use the client for processing (example).

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	// In a real application, you would process intents here.

	_ = ctx

	_ = client
}

// secureConfigExample shows explicit secure configuration.

func secureConfigExample() {
	config := llm.ClientConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"), // Load from environment

		ModelName: "gpt-4",

		MaxTokens: 2048,

		BackendType: "openai",

		Timeout: 60 * time.Second,

		// SkipTLSVerification is false by default - secure by design.

	}

	client := llm.NewClientWithConfig("https://api.openai.com/v1/chat/completions", config)

	fmt.Println("✓ Created secure client with explicit configuration")

	fmt.Println("  - API Key: Loaded from environment")

	fmt.Println("  - TLS verification: ENABLED (default)")

	fmt.Println("  - Timeout: 60 seconds")

	_ = client
}

// developmentExample shows how to safely disable TLS for development.

func developmentExample() {
	// SECURITY WARNING: This is for development environments only!.

	// Never use this in production!.

	// Step 1: Check if we're in a development environment.

	if os.Getenv("ENVIRONMENT") == "production" {
		log.Fatal("ERROR: Cannot use insecure TLS in production environment")
	}

	// Step 2: Set the environment variable to allow insecure connections.

	// In real development, this would be set in your development environment.

	os.Setenv("ALLOW_INSECURE_CLIENT", "true")

	defer os.Unsetenv("ALLOW_INSECURE_CLIENT") // Clean up for other examples

	// Step 3: Configure the client with insecure TLS.

	config := llm.ClientConfig{
		// SECURITY: Never hardcode API keys - load from environment
		APIKey: os.Getenv("LLM_DEV_API_KEY"), // Load from environment

		ModelName: "gpt-4",

		MaxTokens: 2048,

		BackendType: "openai",

		Timeout: 30 * time.Second,

		SkipTLSVerification: true, // EXPLICITLY request insecure mode

	}

	// This will succeed but log a security warning.

	client := llm.NewClientWithConfig("https://dev-server.local:8443", config)

	fmt.Println("✓ Created development client with TLS verification disabled")

	fmt.Println("  - SECURITY WARNING: TLS verification is DISABLED")

	fmt.Println("  - This should ONLY be used in development environments")

	fmt.Println("  - With self-signed certificates or internal CAs")

	_ = client
}

// securityViolationExample demonstrates what happens when security is violated.

func securityViolationExample() {
	// This will demonstrate the security protection in action.

	defer func() {
		if r := recover(); r != nil {

			fmt.Println("✓ Security violation correctly prevented!")

			fmt.Printf("  - Panic message: %v\n", r)

			fmt.Println("  - This protects against accidental insecure configuration")

		}
	}()

	// Ensure the environment variable is NOT set.

	os.Unsetenv("ALLOW_INSECURE_CLIENT")

	// Try to create an insecure client without permission.

	config := llm.ClientConfig{
		// SECURITY: Never hardcode API keys - load from environment
		APIKey: os.Getenv("LLM_TEST_API_KEY"), // Load from environment

		ModelName: "test-model",

		MaxTokens: 100,

		BackendType: "openai",

		Timeout: 30 * time.Second,

		SkipTLSVerification: true, // This should fail without environment permission

	}

	// This WILL panic due to security violation.

	_ = llm.NewClientWithConfig("https://example.com", config)

	// We should never reach this line.

	fmt.Println("✗ ERROR: Security violation was not caught!")
}

// Additional utility functions for production usage.

// CreateProductionClient creates a properly configured production client.

func CreateProductionClient(endpoint string) *llm.Client {
	// Validate environment.

	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create secure configuration.

	config := llm.ClientConfig{
		APIKey: os.Getenv("OPENAI_API_KEY"),

		ModelName: getModelFromEnv(),

		MaxTokens: getMaxTokensFromEnv(),

		BackendType: getBackendFromEnv(),

		Timeout: getTimeoutFromEnv(),

		// SkipTLSVerification is intentionally omitted - defaults to false (secure).

	}

	return llm.NewClientWithConfig(endpoint, config)
}

// Utility functions to load configuration from environment.

func getModelFromEnv() string {
	if model := os.Getenv("LLM_MODEL"); model != "" {
		return model
	}

	return "gpt-4" // Secure default
}

func getMaxTokensFromEnv() int {
	// In a real application, you'd parse this from environment.

	return 2048
}

func getBackendFromEnv() string {
	if backend := os.Getenv("LLM_BACKEND"); backend != "" {
		return backend
	}

	return "openai" // Secure default
}

func getTimeoutFromEnv() time.Duration {
	// In a real application, you'd parse this from environment.

	return 60 * time.Second
}
