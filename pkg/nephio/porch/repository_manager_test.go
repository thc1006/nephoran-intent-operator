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

package porch

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Mock implementations for testing

type MockGitClient struct {
	mock.Mock
}

func (m *MockGitClient) Clone(ctx context.Context, targetDir string) error {
	args := m.Called(ctx, targetDir)
	return args.Error(0)
}

func (m *MockGitClient) Pull(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockGitClient) ListBranches(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockGitClient) CreateBranch(ctx context.Context, branchName string, baseBranch string) error {
	args := m.Called(ctx, branchName, baseBranch)
	return args.Error(0)
}

func (m *MockGitClient) DeleteBranch(ctx context.Context, branchName string) error {
	args := m.Called(ctx, branchName)
	return args.Error(0)
}

func (m *MockGitClient) GetCommitHash(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) GetDefaultBranch(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockGitClient) ValidateAccess(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type MockGitClientFactory struct {
	mock.Mock
}

func (m *MockGitClientFactory) CreateClient(ctx context.Context, config *RepositoryConfig) (GitClient, error) {
	args := m.Called(ctx, config)
	return args.Get(0).(GitClient), args.Error(1)
}

func (m *MockGitClientFactory) ValidateRepository(ctx context.Context, config *RepositoryConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

type MockCredentialStore struct {
	mock.Mock
}

func (m *MockCredentialStore) StoreCredentials(ctx context.Context, repoName string, creds *Credentials) error {
	args := m.Called(ctx, repoName, creds)
	return args.Error(0)
}

func (m *MockCredentialStore) GetCredentials(ctx context.Context, repoName string) (*Credentials, error) {
	args := m.Called(ctx, repoName)
	return args.Get(0).(*Credentials), args.Error(1)
}

func (m *MockCredentialStore) DeleteCredentials(ctx context.Context, repoName string) error {
	args := m.Called(ctx, repoName)
	return args.Error(0)
}

func (m *MockCredentialStore) ValidateCredentials(ctx context.Context, repoName string) error {
	args := m.Called(ctx, repoName)
	return args.Error(0)
}

type MockRepositoryValidationRule struct {
	mock.Mock
	name string
}

func (m *MockRepositoryValidationRule) Validate(ctx context.Context, config *RepositoryConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *MockRepositoryValidationRule) GetName() string {
	return m.name
}

type MockRepositoryEventHandler struct {
	mock.Mock
}

func (m *MockRepositoryEventHandler) OnRepositoryRegistered(ctx context.Context, repo *Repository) {
	m.Called(ctx, repo)
}

func (m *MockRepositoryEventHandler) OnRepositoryUnregistered(ctx context.Context, repoName string) {
	m.Called(ctx, repoName)
}

func (m *MockRepositoryEventHandler) OnRepositorySynced(ctx context.Context, repoName string, result *SyncResult) {
	m.Called(ctx, repoName, result)
}

func (m *MockRepositoryEventHandler) OnRepositoryError(ctx context.Context, repoName string, err error) {
	m.Called(ctx, repoName, err)
}

// Test fixtures

func createTestRepositoryConfig() *RepositoryConfig {
	return &RepositoryConfig{
		Name:      "test-repo",
		Type:      "git",
		URL:       "https://github.com/test/test-repo.git",
		Branch:    "main",
		Directory: "",
		Auth: &AuthConfig{
			Type:     "token",
			Username: "test-user",
		},
		Sync: &SyncConfig{
			AutoSync: true,
			Interval: &metav1.Duration{Duration: 5 * time.Minute},
		},
		Capabilities: []string{"read", "write"},
	}
}

func createTestRepositoryManager(client *Client) *repositoryManager {
	rm := &repositoryManager{
		client:       client,
		repositories: make(map[string]*repositoryState),
		metrics:      initRepositoryMetrics(),
	}
	
	// Set up mocks
	mockFactory := &MockGitClientFactory{}
	mockGitClient := &MockGitClient{}
	mockCredStore := &MockCredentialStore{}
	
	// Configure default mock behaviors
	mockFactory.On("CreateClient", mock.Anything, mock.Anything).Return(mockGitClient, nil)
	mockGitClient.On("ValidateAccess", mock.Anything).Return(nil)
	mockGitClient.On("GetCommitHash", mock.Anything).Return("abc123", nil)
	mockGitClient.On("ListBranches", mock.Anything).Return([]string{"main", "develop"}, nil)
	mockGitClient.On("GetDefaultBranch", mock.Anything).Return("main", nil)
	mockGitClient.On("CreateBranch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockGitClient.On("DeleteBranch", mock.Anything, mock.Anything).Return(nil)
	
	mockCredStore.On("StoreCredentials", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockCredStore.On("DeleteCredentials", mock.Anything, mock.Anything).Return(nil)
	
	rm.SetGitClientFactory(mockFactory)
	rm.SetCredentialStore(mockCredStore)
	
	return rm
}

// Unit Tests

func TestNewRepositoryManager(t *testing.T) {
	client := createTestClient()
	rm := NewRepositoryManager(client)
	
	assert.NotNil(t, rm)
	assert.IsType(t, &repositoryManager{}, rm)
	
	rmImpl := rm.(*repositoryManager)
	assert.Equal(t, client, rmImpl.client)
	assert.NotNil(t, rmImpl.repositories)
	assert.NotNil(t, rmImpl.metrics)
}

func TestRepositoryManagerRegisterRepository(t *testing.T) {
	tests := []struct {
		name        string
		config      *RepositoryConfig
		expectError bool
		setupMocks  func(*MockPorchClient)
	}{
		{
			name:        "ValidGitRepository",
			config:      createTestRepositoryConfig(),
			expectError: false,
			setupMocks: func(client *MockPorchClient) {
				repo := &Repository{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-repo",
					},
				}
				client.On("CreateRepository", mock.Anything, mock.Anything).Return(repo, nil)
			},
		},
		{
			name: "InvalidRepositoryURL",
			config: &RepositoryConfig{
				Name: "invalid-repo",
				Type: "git",
				URL:  "invalid-url",
			},
			expectError: true,
			setupMocks: func(client *MockPorchClient) {
				// No setup needed - validation should fail before API call
			},
		},
		{
			name: "MissingRepositoryName",
			config: &RepositoryConfig{
				Type: "git",
				URL:  "https://github.com/test/test-repo.git",
			},
			expectError: true,
			setupMocks: func(client *MockPorchClient) {
				// No setup needed - validation should fail before API call
			},
		},
		{
			name: "UnsupportedRepositoryType",
			config: &RepositoryConfig{
				Name: "unsupported-repo",
				Type: "svn",
				URL:  "https://svn.example.com/repo",
			},
			expectError: true,
			setupMocks: func(client *MockPorchClient) {
				// No setup needed - validation should fail before API call
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := createTestClient()
			mockClient := &MockPorchClient{}
			rm := createTestRepositoryManager(client)
			
			// Override the client
			rmImpl := rm.(*repositoryManager)
			rmImpl.client = mockClient
			
			tt.setupMocks(mockClient)
			
			result, err := rm.RegisterRepository(context.Background(), tt.config)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				
				// Verify repository is tracked
				state, exists := rmImpl.repositories[tt.config.Name]
				assert.True(t, exists)
				assert.NotNil(t, state)
				assert.Equal(t, tt.config, state.config)
			}
			
			mockClient.AssertExpectations(t)
		})
	}
}

func TestRepositoryManagerUnregisterRepository(t *testing.T) {
	client := createTestClient()
	mockClient := &MockPorchClient{}
	rm := createTestRepositoryManager(client)
	rmImpl := rm.(*repositoryManager)
	rmImpl.client = mockClient
	
	config := createTestRepositoryConfig()
	
	// Setup: Register repository first
	repo := &Repository{ObjectMeta: metav1.ObjectMeta{Name: config.Name}}
	mockClient.On("CreateRepository", mock.Anything, mock.Anything).Return(repo, nil)
	mockClient.On("DeleteRepository", mock.Anything, config.Name).Return(nil)
	
	_, err := rm.RegisterRepository(context.Background(), config)
	require.NoError(t, err)
	
	// Test: Unregister repository
	err = rm.UnregisterRepository(context.Background(), config.Name)
	assert.NoError(t, err)
	
	// Verify repository is no longer tracked
	_, exists := rmImpl.repositories[config.Name]
	assert.False(t, exists)
	
	mockClient.AssertExpectations(t)
}

func TestRepositoryManagerSynchronizeRepository(t *testing.T) {
	client := createTestClient()
	mockClient := &MockPorchClient{}
	rm := createTestRepositoryManager(client)
	rmImpl := rm.(*repositoryManager)
	rmImpl.client = mockClient
	
	config := createTestRepositoryConfig()
	
	// Setup: Register repository first
	repo := &Repository{ObjectMeta: metav1.ObjectMeta{Name: config.Name}}
	mockClient.On("CreateRepository", mock.Anything, mock.Anything).Return(repo, nil)
	mockClient.On("GetRepository", mock.Anything, config.Name).Return(repo, nil)
	mockClient.On("UpdateRepository", mock.Anything, mock.Anything).Return(repo, nil)
	
	_, err := rm.RegisterRepository(context.Background(), config)
	require.NoError(t, err)
	
	// Test: Synchronize repository
	result, err := rm.SynchronizeRepository(context.Background(), config.Name)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "abc123", result.CommitHash)
	
	mockClient.AssertExpectations(t)
}

func TestRepositoryManagerBranchOperations(t *testing.T) {
	client := createTestClient()
	mockClient := &MockPorchClient{}
	rm := createTestRepositoryManager(client)
	rmImpl := rm.(*repositoryManager)
	rmImpl.client = mockClient
	
	config := createTestRepositoryConfig()
	
	// Setup: Register repository first
	repo := &Repository{ObjectMeta: metav1.ObjectMeta{Name: config.Name}}
	mockClient.On("CreateRepository", mock.Anything, mock.Anything).Return(repo, nil)
	
	_, err := rm.RegisterRepository(context.Background(), config)
	require.NoError(t, err)
	
	t.Run("CreateBranch", func(t *testing.T) {
		err := rm.CreateBranch(context.Background(), config.Name, "feature-branch", "main")
		assert.NoError(t, err)
	})
	
	t.Run("ListBranches", func(t *testing.T) {
		branches, err := rm.ListBranches(context.Background(), config.Name)
		assert.NoError(t, err)
		assert.Contains(t, branches, "main")
		assert.Contains(t, branches, "develop")
	})
	
	t.Run("DeleteBranch", func(t *testing.T) {
		err := rm.DeleteBranch(context.Background(), config.Name, "feature-branch")
		assert.NoError(t, err)
	})
	
	t.Run("DeleteDefaultBranch", func(t *testing.T) {
		err := rm.DeleteBranch(context.Background(), config.Name, "main")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete default branch")
	})
	
	mockClient.AssertExpectations(t)
}

func TestRepositoryManagerCredentialOperations(t *testing.T) {
	client := createTestClient()
	mockClient := &MockPorchClient{}
	rm := createTestRepositoryManager(client)
	rmImpl := rm.(*repositoryManager)
	rmImpl.client = mockClient
	
	config := createTestRepositoryConfig()
	
	// Setup: Register repository first
	repo := &Repository{ObjectMeta: metav1.ObjectMeta{Name: config.Name}}
	mockClient.On("CreateRepository", mock.Anything, mock.Anything).Return(repo, nil)
	
	_, err := rm.RegisterRepository(context.Background(), config)
	require.NoError(t, err)
	
	t.Run("UpdateValidCredentials", func(t *testing.T) {
		creds := &Credentials{
			Type:     "token",
			Username: "test-user",
			Token:    "test-token",
		}
		
		err := rm.UpdateCredentials(context.Background(), config.Name, creds)
		assert.NoError(t, err)
	})
	
	t.Run("UpdateInvalidCredentials", func(t *testing.T) {
		creds := &Credentials{
			Type: "basic",
			// Missing username and password
		}
		
		err := rm.UpdateCredentials(context.Background(), config.Name, creds)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "username and password are required")
	})
	
	t.Run("ValidateAccess", func(t *testing.T) {
		err := rm.ValidateAccess(context.Background(), config.Name)
		assert.NoError(t, err)
	})
	
	mockClient.AssertExpectations(t)
}

func TestRepositoryManagerGetHealth(t *testing.T) {
	client := createTestClient()
	mockClient := &MockPorchClient{}
	rm := createTestRepositoryManager(client)
	rmImpl := rm.(*repositoryManager)
	rmImpl.client = mockClient
	
	config := createTestRepositoryConfig()
	
	// Setup: Register repository first
	repo := &Repository{ObjectMeta: metav1.ObjectMeta{Name: config.Name}}
	mockClient.On("CreateRepository", mock.Anything, mock.Anything).Return(repo, nil)
	
	_, err := rm.RegisterRepository(context.Background(), config)
	require.NoError(t, err)
	
	// Test: Get repository health
	health, err := rm.GetRepositoryHealth(context.Background(), config.Name)
	assert.NoError(t, err)
	assert.NotNil(t, health)
	
	mockClient.AssertExpectations(t)
}

func TestRepositoryManagerValidationRules(t *testing.T) {
	client := createTestClient()
	rm := createTestRepositoryManager(client)
	rmImpl := rm.(*repositoryManager)
	
	// Add validation rule
	mockRule := &MockRepositoryValidationRule{name: "test-rule"}
	mockRule.On("Validate", mock.Anything, mock.Anything).Return(nil)
	mockRule.On("GetName").Return("test-rule")
	
	rm.AddValidationRule(mockRule)
	
	// Verify rule was added
	assert.Len(t, rmImpl.validationRules, 1)
	assert.Equal(t, "test-rule", rmImpl.validationRules[0].GetName())
}

func TestRepositoryManagerEventHandlers(t *testing.T) {
	client := createTestClient()
	mockClient := &MockPorchClient{}
	rm := createTestRepositoryManager(client)
	rmImpl := rm.(*repositoryManager)
	rmImpl.client = mockClient
	
	// Add event handler
	mockHandler := &MockRepositoryEventHandler{}
	rm.AddEventHandler(mockHandler)
	
	config := createTestRepositoryConfig()
	repo := &Repository{ObjectMeta: metav1.ObjectMeta{Name: config.Name}}
	
	// Setup expectations
	mockClient.On("CreateRepository", mock.Anything, mock.Anything).Return(repo, nil)
	mockHandler.On("OnRepositoryRegistered", mock.Anything, repo)
	
	// Test: Register repository should trigger event
	_, err := rm.RegisterRepository(context.Background(), config)
	assert.NoError(t, err)
	
	mockClient.AssertExpectations(t)
	mockHandler.AssertExpectations(t)
}

// Performance Tests

func TestRepositoryManagerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}
	
	client := createTestClient()
	mockClient := &MockPorchClient{}
	rm := createTestRepositoryManager(client)
	rmImpl := rm.(*repositoryManager)
	rmImpl.client = mockClient
	
	t.Run("RegisterRepositoryLatency", func(t *testing.T) {
		config := createTestRepositoryConfig()
		repo := &Repository{ObjectMeta: metav1.ObjectMeta{Name: config.Name}}
		mockClient.On("CreateRepository", mock.Anything, mock.Anything).Return(repo, nil)
		
		start := time.Now()
		_, err := rm.RegisterRepository(context.Background(), config)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Less(t, duration, 500*time.Millisecond, "Repository registration should complete in <500ms")
	})
	
	t.Run("SynchronizeRepositoryLatency", func(t *testing.T) {
		config := createTestRepositoryConfig()
		config.Name = "sync-test-repo"
		repo := &Repository{ObjectMeta: metav1.ObjectMeta{Name: config.Name}}
		
		mockClient.On("CreateRepository", mock.Anything, mock.Anything).Return(repo, nil)
		mockClient.On("GetRepository", mock.Anything, config.Name).Return(repo, nil)
		mockClient.On("UpdateRepository", mock.Anything, mock.Anything).Return(repo, nil)
		
		_, err := rm.RegisterRepository(context.Background(), config)
		require.NoError(t, err)
		
		start := time.Now()
		_, err = rm.SynchronizeRepository(context.Background(), config.Name)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Less(t, duration, 500*time.Millisecond, "Repository sync should complete in <500ms")
	})
}

func TestRepositoryManagerConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}
	
	client := createTestClient()
	mockClient := &MockPorchClient{}
	rm := createTestRepositoryManager(client)
	rmImpl := rm.(*repositoryManager)
	rmImpl.client = mockClient
	
	concurrency := 50
	
	t.Run("ConcurrentRepositoryRegistration", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]error, concurrency)
		
		for i := 0; i < concurrency; i++ {
			config := createTestRepositoryConfig()
			config.Name = fmt.Sprintf("concurrent-repo-%d", i)
			repo := &Repository{ObjectMeta: metav1.ObjectMeta{Name: config.Name}}
			
			mockClient.On("CreateRepository", mock.Anything, mock.MatchedBy(func(r *Repository) bool {
				return r.Name == config.Name
			})).Return(repo, nil)
		}
		
		start := time.Now()
		
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				
				config := createTestRepositoryConfig()
				config.Name = fmt.Sprintf("concurrent-repo-%d", index)
				
				_, err := rm.RegisterRepository(context.Background(), config)
				results[index] = err
			}(i)
		}
		
		wg.Wait()
		duration := time.Since(start)
		
		// Check all operations succeeded
		for i, err := range results {
			assert.NoError(t, err, "Operation %d should succeed", i)
		}
		
		// Check throughput
		opsPerSecond := float64(concurrency) / duration.Seconds()
		assert.Greater(t, opsPerSecond, 10.0, "Should handle >10 operations per second")
		
		// Verify all repositories are tracked
		assert.Len(t, rmImpl.repositories, concurrency)
	})
}

func TestRepositoryManagerErrorHandling(t *testing.T) {
	client := createTestClient()
	mockClient := &MockPorchClient{}
	rm := createTestRepositoryManager(client)
	rmImpl := rm.(*repositoryManager)
	rmImpl.client = mockClient
	
	t.Run("RegisterNonexistentRepository", func(t *testing.T) {
		config := &RepositoryConfig{
			Name: "nonexistent-repo",
			Type: "git",
			URL:  "https://github.com/nonexistent/repo.git",
		}
		
		mockClient.On("CreateRepository", mock.Anything, mock.Anything).Return((*Repository)(nil), fmt.Errorf("repository not found"))
		
		_, err := rm.RegisterRepository(context.Background(), config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create repository in Porch")
	})
	
	t.Run("SynchronizeNonexistentRepository", func(t *testing.T) {
		_, err := rm.SynchronizeRepository(context.Background(), "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "repository nonexistent not found")
	})
	
	t.Run("UnregisterNonexistentRepository", func(t *testing.T) {
		err := rm.UnregisterRepository(context.Background(), "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "repository nonexistent not found")
	})
	
	t.Run("GetHealthOfNonexistentRepository", func(t *testing.T) {
		_, err := rm.GetRepositoryHealth(context.Background(), "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "repository nonexistent not found")
	})
}

func TestRepositoryManagerContextCancellation(t *testing.T) {
	client := createTestClient()
	mockClient := &MockPorchClient{}
	rm := createTestRepositoryManager(client)
	rmImpl := rm.(*repositoryManager)
	rmImpl.client = mockClient
	
	t.Run("CancelledContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		config := createTestRepositoryConfig()
		_, err := rm.RegisterRepository(ctx, config)
		// Error might occur during validation or API call
		// We mainly want to ensure the context is respected
		if err != nil {
			// Context cancellation should be handled gracefully
			t.Logf("Expected error due to cancelled context: %v", err)
		}
	})
}

func TestRepositoryManagerClose(t *testing.T) {
	client := createTestClient()
	rm := createTestRepositoryManager(client)
	
	err := rm.Close()
	assert.NoError(t, err)
	
	// Verify cleanup
	rmImpl := rm.(*repositoryManager)
	assert.Empty(t, rmImpl.repositories)
}

// Benchmark Tests

func BenchmarkRepositoryManagerOperations(b *testing.B) {
	client := createTestClient()
	mockClient := &MockPorchClient{}
	rm := createTestRepositoryManager(client)
	rmImpl := rm.(*repositoryManager)
	rmImpl.client = mockClient
	
	b.Run("RegisterRepository", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			config := createTestRepositoryConfig()
			config.Name = fmt.Sprintf("bench-repo-%d", i)
			repo := &Repository{ObjectMeta: metav1.ObjectMeta{Name: config.Name}}
			
			mockClient.On("CreateRepository", mock.Anything, mock.MatchedBy(func(r *Repository) bool {
				return r.Name == config.Name
			})).Return(repo, nil).Once()
			
			_, err := rm.RegisterRepository(context.Background(), config)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("ValidateAccess", func(b *testing.B) {
		// Setup
		config := createTestRepositoryConfig()
		repo := &Repository{ObjectMeta: metav1.ObjectMeta{Name: config.Name}}
		mockClient.On("CreateRepository", mock.Anything, mock.Anything).Return(repo, nil)
		
		_, err := rm.RegisterRepository(context.Background(), config)
		if err != nil {
			b.Fatal(err)
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := rm.ValidateAccess(context.Background(), config.Name)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}