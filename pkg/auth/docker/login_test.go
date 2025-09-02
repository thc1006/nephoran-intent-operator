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
	"testing"

	"oras.land/oras-go/v2/registry/remote/auth"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name  string
		store CredentialStore
	}{
		{
			name:  "with nil store",
			store: nil,
		},
		{
			name:  "with memory store",
			store: NewInMemoryStore(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.store)
			if client == nil {
				t.Errorf("NewClient() returned nil")
			}
			if client.authClient == nil {
				t.Errorf("NewClient() auth client is nil")
			}
		})
	}
}

func TestClient_Login(t *testing.T) {
	tests := []struct {
		name    string
		config  *LoginConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name:    "empty registry",
			config:  &LoginConfig{},
			wantErr: true,
		},
		{
			name: "valid config with username/password",
			config: &LoginConfig{
				Registry: "docker.io",
				Username: "testuser",
				Password: "testpass",
			},
			wantErr: false,
		},
		{
			name: "valid config with token",
			config: &LoginConfig{
				Registry: "docker.io",
				Token:    "testtoken",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(nil)
			err := client.Login(context.Background(), tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.Login() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_LoginWithToken(t *testing.T) {
	client := NewClient(nil)

	err := client.LoginWithToken(context.Background(), "docker.io", "testtoken")
	if err != nil {
		t.Errorf("Client.LoginWithToken() error = %v", err)
	}
}

func TestClient_GetAuthClient(t *testing.T) {
	client := NewClient(nil)
	authClient := client.GetAuthClient()

	if authClient == nil {
		t.Errorf("Client.GetAuthClient() returned nil")
	}

	if authClient != client.authClient {
		t.Errorf("Client.GetAuthClient() returned different client")
	}
}

func TestClient_IsLoggedIn(t *testing.T) {
	client := NewClient(nil)

	// With no credential store, should return false
	loggedIn, err := client.IsLoggedIn(context.Background(), "docker.io")
	if err != nil {
		t.Errorf("Client.IsLoggedIn() error = %v", err)
	}
	if loggedIn {
		t.Errorf("Client.IsLoggedIn() = %v, want false", loggedIn)
	}
}

func TestClient_GetCredential(t *testing.T) {
	client := NewClient(nil)

	// With no credential store, should return error
	_, err := client.GetCredential(context.Background(), "docker.io")
	if err == nil {
		t.Errorf("Client.GetCredential() expected error, got nil")
	}
}

func TestDefaultCredentialStore(t *testing.T) {
	store := DefaultCredentialStore()
	if store == nil {
		t.Errorf("DefaultCredentialStore() returned nil store")
	}
}

// Mock credential store for testing
type mockCredStore struct {
	credentials map[string]auth.Credential
}

func newMockCredStore() *mockCredStore {
	return &mockCredStore{
		credentials: make(map[string]auth.Credential),
	}
}

func (m *mockCredStore) Get(ctx context.Context, serverURL string) (auth.Credential, error) {
	if cred, exists := m.credentials[serverURL]; exists {
		return cred, nil
	}
	return auth.EmptyCredential, fmt.Errorf("credentials not found")
}

func (m *mockCredStore) Store(ctx context.Context, serverURL string, cred auth.Credential) error {
	m.credentials[serverURL] = cred
	return nil
}

func (m *mockCredStore) Delete(ctx context.Context, serverURL string) error {
	delete(m.credentials, serverURL)
	return nil
}

func TestClientWithMockStore(t *testing.T) {
	mockStore := newMockCredStore()
	client := NewClient(mockStore)

	// Test login
	config := &LoginConfig{
		Registry: "docker.io",
		Username: "testuser",
		Password: "testpass",
	}

	err := client.Login(context.Background(), config)
	if err != nil {
		t.Errorf("Client.Login() error = %v", err)
	}

	// Test is logged in
	loggedIn, err := client.IsLoggedIn(context.Background(), "docker.io")
	if err != nil {
		t.Errorf("Client.IsLoggedIn() error = %v", err)
	}
	if !loggedIn {
		t.Errorf("Client.IsLoggedIn() = %v, want true", loggedIn)
	}

	// Test get credential
	cred, err := client.GetCredential(context.Background(), "docker.io")
	if err != nil {
		t.Errorf("Client.GetCredential() error = %v", err)
	}
	if cred.Username != "testuser" {
		t.Errorf("Client.GetCredential() username = %v, want testuser", cred.Username)
	}

	// Test logout
	err = client.Logout(context.Background(), "docker.io")
	if err != nil {
		t.Errorf("Client.Logout() error = %v", err)
	}

	// Should not be logged in anymore
	loggedIn, err = client.IsLoggedIn(context.Background(), "docker.io")
	if err != nil {
		t.Errorf("Client.IsLoggedIn() after logout error = %v", err)
	}
	if loggedIn {
		t.Errorf("Client.IsLoggedIn() after logout = %v, want false", loggedIn)
	}
}
