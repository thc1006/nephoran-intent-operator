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
	"net/http"
	"sync"

	"oras.land/oras-go/v2/registry/remote/auth"
)

// LoginConfig represents the configuration for Docker registry authentication
type LoginConfig struct {
	Registry string `json:"registry" yaml:"registry"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	Token    string `json:"token,omitempty" yaml:"token,omitempty"`
}

// CredentialStore provides a simple interface for credential storage
type CredentialStore interface {
	Store(ctx context.Context, registry string, cred auth.Credential) error
	Get(ctx context.Context, registry string) (auth.Credential, error)
	Delete(ctx context.Context, registry string) error
}

// Client wraps the ORAS v2 authentication client
type Client struct {
	authClient *auth.Client
	credStore  CredentialStore
	mu         sync.RWMutex
}

// InMemoryStore provides an in-memory credential store
type InMemoryStore struct {
	credentials map[string]auth.Credential
	mu          sync.RWMutex
}

// NewInMemoryStore creates a new in-memory credential store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		credentials: make(map[string]auth.Credential),
	}
}

func (s *InMemoryStore) Store(ctx context.Context, registry string, cred auth.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.credentials[registry] = cred
	return nil
}

func (s *InMemoryStore) Get(ctx context.Context, registry string) (auth.Credential, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if cred, exists := s.credentials[registry]; exists {
		return cred, nil
	}
	return auth.EmptyCredential, fmt.Errorf("credential not found for registry %s", registry)
}

func (s *InMemoryStore) Delete(ctx context.Context, registry string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.credentials, registry)
	return nil
}

// NewClient creates a new Docker authentication client using ORAS v2
func NewClient(store CredentialStore) *Client {
	if store == nil {
		store = NewInMemoryStore()
	}

	client := &auth.Client{
		Client: &http.Client{},
		Cache:  auth.DefaultCache,
	}

	return &Client{
		authClient: client,
		credStore:  store,
	}
}

// Login authenticates with a Docker registry using username/password
func (c *Client) Login(ctx context.Context, config *LoginConfig) error {
	if config == nil {
		return fmt.Errorf("login config cannot be nil")
	}

	if config.Registry == "" {
		return fmt.Errorf("registry URL is required")
	}

	// Create credential for the registry
	cred := auth.Credential{
		Username: config.Username,
		Password: config.Password,
	}

	// If using token authentication
	if config.Token != "" {
		cred.Username = "oauth2accesstoken" // Standard for token auth
		cred.Password = config.Token
	}

	// Store the credential
	err := c.credStore.Store(ctx, config.Registry, cred)
	if err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	// Update the auth client credential function
	c.authClient.Credential = func(ctx context.Context, registry string) (auth.Credential, error) {
		return c.credStore.Get(ctx, registry)
	}

	return nil
}

// LoginWithToken authenticates with a Docker registry using a token
func (c *Client) LoginWithToken(ctx context.Context, registry, token string) error {
	return c.Login(ctx, &LoginConfig{
		Registry: registry,
		Token:    token,
	})
}

// Logout removes stored credentials for a registry
func (c *Client) Logout(ctx context.Context, registry string) error {
	return c.credStore.Delete(ctx, registry)
}

// GetAuthClient returns the underlying ORAS auth client
func (c *Client) GetAuthClient() *auth.Client {
	return c.authClient
}

// IsLoggedIn checks if we have stored credentials for a registry
func (c *Client) IsLoggedIn(ctx context.Context, registry string) (bool, error) {
	_, err := c.credStore.Get(ctx, registry)
	if err != nil {
		// If credentials not found or any error, consider not logged in
		return false, nil
	}
	return true, nil
}

// GetCredential retrieves the stored credential for a registry
func (c *Client) GetCredential(ctx context.Context, registry string) (auth.Credential, error) {
	cred, err := c.credStore.Get(ctx, registry)
	if err != nil {
		return auth.EmptyCredential, fmt.Errorf("failed to get credential: %w", err)
	}
	return cred, nil
}

// DefaultCredentialStore returns a default in-memory credential store
func DefaultCredentialStore() CredentialStore {
	return NewInMemoryStore()
}
