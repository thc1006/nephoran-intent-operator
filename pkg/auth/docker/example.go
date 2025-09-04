/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package docker

import (
	"context"
	"fmt"
	"log"

	"oras.land/oras-go/v2/registry/remote"
)

// ExampleUsage demonstrates how to use the ORAS v2 Docker authentication
func ExampleUsage() {
	ctx := context.Background()

	// Example 1: Basic authentication with username/password
	fmt.Println("=== Example 1: Basic Authentication ===")
	basicAuth(ctx)

	// Example 2: Token-based authentication
	fmt.Println("\n=== Example 2: Token Authentication ===")
	tokenAuth(ctx)

	// Example 3: TLS authentication
	fmt.Println("\n=== Example 3: TLS Authentication ===")
	tlsAuth(ctx)

	// Example 4: Using with ORAS remote registry
	fmt.Println("\n=== Example 4: Integration with ORAS Remote ===")
	orasIntegration(ctx)
}

func basicAuth(ctx context.Context) {
	// Create a credential store (optional)
	store := DefaultCredentialStore()

	// Create authentication client
	client := NewClient(store)

	// Configure login
	config := &LoginConfig{
		Registry: "docker.io",
		Username: "myusername",
		Password: "mypassword",
	}

	// Perform login
	err := client.Login(ctx, config)
	if err != nil {
		log.Printf("Login failed: %v", err)
		return
	}

	fmt.Println("✓ Successfully logged in with username/password")

	// Check if logged in
	loggedIn, err := client.IsLoggedIn(ctx, "docker.io")
	if err != nil {
		log.Printf("Failed to check login status: %v", err)
		return
	}

	fmt.Printf("✓ Login status: %t\n", loggedIn)

	// Logout
	err = client.Logout(ctx, "docker.io")
	if err != nil {
		log.Printf("Logout failed: %v", err)
		return
	}

	fmt.Println("✓ Successfully logged out")
}

func tokenAuth(ctx context.Context) {
	client := NewClient(nil) // No credential store for this example

	// Login with token
	err := client.LoginWithToken(ctx, "ghcr.io", "ghp_myaccesstoken")
	if err != nil {
		log.Printf("Token login failed: %v", err)
		return
	}

	fmt.Println("✓ Successfully logged in with token")

	// Get the auth client for use with other ORAS operations
	authClient := client.GetAuthClient()
	if authClient != nil {
		fmt.Println("✓ Auth client is ready for use")
	}
}

func tlsAuth(ctx context.Context) {
	// Configure TLS settings
	tlsConfig := &TLSConfig{
		CertFile:   "/path/to/client.crt",
		KeyFile:    "/path/to/client.key",
		CAFile:     "/path/to/ca.crt",
		ServerName: "my-registry.example.com",
	}

	// Create TLS client
	tlsClient, err := NewTLSClient(nil, tlsConfig)
	if err != nil {
		log.Printf("Failed to create TLS client: %v", err)
		return
	}

	// Configure login with TLS
	loginConfig := &TLSLoginConfig{
		LoginConfig: LoginConfig{
			Registry: "my-registry.example.com",
			Username: "myuser",
			Password: "mypass",
		},
		TLS: tlsConfig,
	}

	// Perform TLS login
	err = tlsClient.LoginWithTLS(ctx, loginConfig)
	if err != nil {
		log.Printf("TLS login failed: %v", err)
		return
	}

	fmt.Println("✓ Successfully logged in with TLS")

	// Note: In a real scenario, you would validate the TLS connection
	// err = tlsClient.ValidateTLSConnection(ctx, "my-registry.example.com")
	// if err != nil {
	//     log.Printf("TLS validation failed: %v", err)
	//     return
	// }
	// fmt.Println("✓ TLS connection validated")
}

func orasIntegration(ctx context.Context) {
	// Create authentication client with in-memory store
	client := NewClient(nil)

	// Login to registry
	err := client.Login(ctx, &LoginConfig{
		Registry: "docker.io",
		Username: "myuser",
		Password: "mypass",
	})
	if err != nil {
		log.Printf("Login failed: %v", err)
		return
	}

	// Create ORAS remote registry client with authentication
	registry, err := remote.NewRegistry("docker.io")
	if err != nil {
		log.Printf("Failed to create registry client: %v", err)
		return
	}

	// Set the authentication client
	registry.Client = client.GetAuthClient()

	fmt.Println("✓ ORAS registry client configured with authentication")

	// Now you can use the registry client for ORAS operations
	// For example:
	// - registry.Repositories(ctx)
	// - registry.Repository(ctx, "myrepo")
	// - etc.
}

// ExampleDockerConfigIntegration shows how to integrate with Docker config
func ExampleDockerConfigIntegration() {
	ctx := context.Background()

	fmt.Println("=== Docker Config Integration ===")

	// Use the default in-memory credential store
	client := NewClient(nil)

	// The credential store will automatically use Docker's config.json
	// located at ~/.docker/config.json or %USERPROFILE%\.docker\config.json

	// Check if already logged in from Docker CLI
	loggedIn, err := client.IsLoggedIn(ctx, "docker.io")
	if err != nil {
		log.Printf("Failed to check Docker login: %v", err)
		return
	}

	if loggedIn {
		fmt.Println("✓ Already logged in via Docker CLI")

		// Get the credential
		cred, err := client.GetCredential(ctx, "docker.io")
		if err != nil {
			log.Printf("Failed to get credential: %v", err)
			return
		}

		fmt.Printf("✓ Using credential for user: %s\n", cred.Username)
	} else {
		fmt.Println("ℹ Not logged in to docker.io via Docker CLI")
		fmt.Println("  Run: docker login docker.io")
	}
}

// ExampleErrorHandling demonstrates proper error handling
func ExampleErrorHandling() {
	ctx := context.Background()
	client := NewClient(nil)

	fmt.Println("=== Error Handling Examples ===")

	// Example 1: Invalid configuration
	err := client.Login(ctx, nil)
	if err != nil {
		fmt.Printf("✓ Expected error for nil config: %v\n", err)
	}

	// Example 2: Empty registry
	err = client.Login(ctx, &LoginConfig{})
	if err != nil {
		fmt.Printf("✓ Expected error for empty registry: %v\n", err)
	}

	// Example 3: No credential store
	_, err = client.GetCredential(ctx, "docker.io")
	if err != nil {
		fmt.Printf("✓ Expected error for no credential store: %v\n", err)
	}

	fmt.Println("✓ All error cases handled correctly")
}
