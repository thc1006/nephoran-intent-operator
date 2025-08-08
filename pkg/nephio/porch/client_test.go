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
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// MockPorchClient implements PorchClient for testing
type MockPorchClient struct {
	mock.Mock
}

func (m *MockPorchClient) GetRepository(ctx context.Context, name string) (*Repository, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*Repository), args.Error(1)
}

func (m *MockPorchClient) ListRepositories(ctx context.Context, opts *ListOptions) (*RepositoryList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*RepositoryList), args.Error(1)
}

func (m *MockPorchClient) CreateRepository(ctx context.Context, repo *Repository) (*Repository, error) {
	args := m.Called(ctx, repo)
	return args.Get(0).(*Repository), args.Error(1)
}

func (m *MockPorchClient) UpdateRepository(ctx context.Context, repo *Repository) (*Repository, error) {
	args := m.Called(ctx, repo)
	return args.Get(0).(*Repository), args.Error(1)
}

func (m *MockPorchClient) DeleteRepository(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockPorchClient) GetPackageRevision(ctx context.Context, name string) (*PackageRevision, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPorchClient) ListPackageRevisions(ctx context.Context, opts *ListOptions) (*PackageRevisionList, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*PackageRevisionList), args.Error(1)
}

func (m *MockPorchClient) CreatePackageRevision(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error) {
	args := m.Called(ctx, pkg)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPorchClient) UpdatePackageRevision(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error) {
	args := m.Called(ctx, pkg)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPorchClient) DeletePackageRevision(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockPorchClient) ApprovePackageRevision(ctx context.Context, name string) (*PackageRevision, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPorchClient) ProposePackageRevision(ctx context.Context, name string) (*PackageRevision, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPorchClient) ExecuteFunction(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*FunctionResponse), args.Error(1)
}

func (m *MockPorchClient) ExecutePipeline(ctx context.Context, req *PipelineRequest) (*PipelineResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*PipelineResponse), args.Error(1)
}

func (m *MockPorchClient) ValidateFunction(ctx context.Context, functionName string) (*FunctionValidation, error) {
	args := m.Called(ctx, functionName)
	return args.Get(0).(*FunctionValidation), args.Error(1)
}

func (m *MockPorchClient) ListFunctions(ctx context.Context) ([]*FunctionInfo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*FunctionInfo), args.Error(1)
}

func (m *MockPorchClient) GetFunctionSchema(ctx context.Context, functionName string) (*FunctionSchema, error) {
	args := m.Called(ctx, functionName)
	return args.Get(0).(*FunctionSchema), args.Error(1)
}

func (m *MockPorchClient) RegisterFunction(ctx context.Context, info *FunctionInfo) error {
	args := m.Called(ctx, info)
	return args.Error(0)
}

func (m *MockPorchClient) GetResourceTransformations(ctx context.Context, req *TransformationRequest) (*TransformationResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*TransformationResponse), args.Error(1)
}

func (m *MockPorchClient) ValidateORANCompliance(ctx context.Context, req *ORANValidationRequest) (*ORANValidationResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ORANValidationResponse), args.Error(1)
}

func (m *MockPorchClient) RegisterRepository(ctx context.Context, config *RepositoryConfig) (*Repository, error) {
	args := m.Called(ctx, config)
	return args.Get(0).(*Repository), args.Error(1)
}

func (m *MockPorchClient) UnregisterRepository(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockPorchClient) SynchronizeRepository(ctx context.Context, name string) (*SyncResult, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*SyncResult), args.Error(1)
}

func (m *MockPorchClient) GetRepositoryHealth(ctx context.Context, name string) (*RepositoryHealth, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*RepositoryHealth), args.Error(1)
}

func (m *MockPorchClient) CreateBranch(ctx context.Context, repoName string, branchName string, baseBranch string) error {
	args := m.Called(ctx, repoName, branchName, baseBranch)
	return args.Error(0)
}

func (m *MockPorchClient) DeleteBranch(ctx context.Context, repoName string, branchName string) error {
	args := m.Called(ctx, repoName, branchName)
	return args.Error(0)
}

func (m *MockPorchClient) ListBranches(ctx context.Context, repoName string) ([]string, error) {
	args := m.Called(ctx, repoName)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockPorchClient) UpdateCredentials(ctx context.Context, repoName string, creds *Credentials) error {
	args := m.Called(ctx, repoName, creds)
	return args.Error(0)
}

func (m *MockPorchClient) ValidateAccess(ctx context.Context, repoName string) error {
	args := m.Called(ctx, repoName)
	return args.Error(0)
}

func (m *MockPorchClient) CreatePackage(ctx context.Context, spec *PackageSpec) (*PackageRevision, error) {
	args := m.Called(ctx, spec)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPorchClient) ClonePackage(ctx context.Context, source string, target *PackageSpec) (*PackageRevision, error) {
	args := m.Called(ctx, source, target)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPorchClient) UpdatePackageContent(ctx context.Context, name string, resources []KRMResource) (*PackageRevision, error) {
	args := m.Called(ctx, name, resources)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPorchClient) GetPackageContent(ctx context.Context, name string) ([]KRMResource, error) {
	args := m.Called(ctx, name)
	return args.Get(0).([]KRMResource), args.Error(1)
}

func (m *MockPorchClient) PromoteToProposed(ctx context.Context, ref *PackageReference) error {
	args := m.Called(ctx, ref)
	return args.Error(0)
}

func (m *MockPorchClient) PromoteToPublished(ctx context.Context, ref *PackageReference) error {
	args := m.Called(ctx, ref)
	return args.Error(0)
}

func (m *MockPorchClient) GetPackageHistory(ctx context.Context, ref *PackageReference) ([]*PackageRevision, error) {
	args := m.Called(ctx, ref)
	return args.Get(0).([]*PackageRevision), args.Error(1)
}

func (m *MockPorchClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Test fixtures
func createTestRepository() *Repository {
	return &Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-repo",
			Labels: map[string]string{
				LabelComponent:  "repository",
				LabelRepository: "test-repo",
			},
		},
		Spec: RepositorySpec{
			Type:      "git",
			URL:       "https://github.com/test/test-repo.git",
			Branch:    "main",
			Directory: "",
		},
		Status: RepositoryStatus{
			Health: RepositoryHealthHealthy,
		},
	}
}

func createTestPackageRevision() *PackageRevision {
	return &PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-package-v1",
			Labels: map[string]string{
				LabelPackage: "test-package",
				LabelVersion: "v1",
			},
		},
		Spec: PackageRevisionSpec{
			PackageName: "test-package",
			Revision:    "v1",
			Repository:  "test-repo",
		},
		Status: PackageRevisionStatus{
			PublishedBy: "test-user",
			PublishedAt: &metav1.Time{Time: time.Now()},
		},
	}
}

func createTestClient() *Client {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	fakeDynamic := dynamicfake.NewSimpleDynamicClient(scheme)
	
	config := NewConfig().WithDefaults()
	
	return &Client{
		client:   fakeClient,
		dynamic:  fakeDynamic,
		config:   config,
		cache:    newClientCache(config),
		metrics:  initClientMetrics(),
	}
}

// Unit Tests

func TestNewClient(t *testing.T) {
	config := NewConfig().WithDefaults()
	
	client, err := NewClient(config)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.config)
	assert.NotNil(t, client.cache)
	assert.NotNil(t, client.metrics)
}

func TestClientRepositoryOperations(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T, client *Client)
	}{
		{
			name: "CreateRepository",
			fn: func(t *testing.T, client *Client) {
				repo := createTestRepository()
				
				created, err := client.CreateRepository(context.Background(), repo)
				assert.NoError(t, err)
				assert.NotNil(t, created)
				assert.Equal(t, repo.Name, created.Name)
			},
		},
		{
			name: "GetRepository",
			fn: func(t *testing.T, client *Client) {
				repo := createTestRepository()
				
				// Create first
				_, err := client.CreateRepository(context.Background(), repo)
				require.NoError(t, err)
				
				// Get
				retrieved, err := client.GetRepository(context.Background(), repo.Name)
				assert.NoError(t, err)
				assert.NotNil(t, retrieved)
				assert.Equal(t, repo.Name, retrieved.Name)
			},
		},
		{
			name: "ListRepositories",
			fn: func(t *testing.T, client *Client) {
				repo := createTestRepository()
				
				// Create first
				_, err := client.CreateRepository(context.Background(), repo)
				require.NoError(t, err)
				
				// List
				list, err := client.ListRepositories(context.Background(), &ListOptions{})
				assert.NoError(t, err)
				assert.NotNil(t, list)
				assert.Len(t, list.Items, 1)
			},
		},
		{
			name: "UpdateRepository",
			fn: func(t *testing.T, client *Client) {
				repo := createTestRepository()
				
				// Create first
				created, err := client.CreateRepository(context.Background(), repo)
				require.NoError(t, err)
				
				// Update
				created.Spec.Branch = "develop"
				updated, err := client.UpdateRepository(context.Background(), created)
				assert.NoError(t, err)
				assert.NotNil(t, updated)
				assert.Equal(t, "develop", updated.Spec.Branch)
			},
		},
		{
			name: "DeleteRepository",
			fn: func(t *testing.T, client *Client) {
				repo := createTestRepository()
				
				// Create first
				_, err := client.CreateRepository(context.Background(), repo)
				require.NoError(t, err)
				
				// Delete
				err = client.DeleteRepository(context.Background(), repo.Name)
				assert.NoError(t, err)
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := createTestClient()
			tt.fn(t, client)
		})
	}
}

func TestClientPackageRevisionOperations(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T, client *Client)
	}{
		{
			name: "CreatePackageRevision",
			fn: func(t *testing.T, client *Client) {
				pkg := createTestPackageRevision()
				
				created, err := client.CreatePackageRevision(context.Background(), pkg)
				assert.NoError(t, err)
				assert.NotNil(t, created)
				assert.Equal(t, pkg.Name, created.Name)
			},
		},
		{
			name: "GetPackageRevision",
			fn: func(t *testing.T, client *Client) {
				pkg := createTestPackageRevision()
				
				// Create first
				_, err := client.CreatePackageRevision(context.Background(), pkg)
				require.NoError(t, err)
				
				// Get
				retrieved, err := client.GetPackageRevision(context.Background(), pkg.Name)
				assert.NoError(t, err)
				assert.NotNil(t, retrieved)
				assert.Equal(t, pkg.Name, retrieved.Name)
			},
		},
		{
			name: "ListPackageRevisions",
			fn: func(t *testing.T, client *Client) {
				pkg := createTestPackageRevision()
				
				// Create first
				_, err := client.CreatePackageRevision(context.Background(), pkg)
				require.NoError(t, err)
				
				// List
				list, err := client.ListPackageRevisions(context.Background(), &ListOptions{})
				assert.NoError(t, err)
				assert.NotNil(t, list)
				assert.Len(t, list.Items, 1)
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := createTestClient()
			tt.fn(t, client)
		})
	}
}

func TestClientFunctionOperations(t *testing.T) {
	client := createTestClient()
	
	t.Run("ValidateFunction", func(t *testing.T) {
		validation, err := client.ValidateFunction(context.Background(), "gcr.io/kpt-fn/apply-setters:v0.1.1")
		assert.NoError(t, err)
		assert.NotNil(t, validation)
		assert.True(t, validation.Valid)
	})
	
	t.Run("ListFunctions", func(t *testing.T) {
		functions, err := client.ListFunctions(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, functions)
		assert.Greater(t, len(functions), 0)
	})
	
	t.Run("ExecuteFunction", func(t *testing.T) {
		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: "gcr.io/kpt-fn/apply-setters:v0.1.1",
			},
			Resources: []KRMResource{},
		}
		
		response, err := client.ExecuteFunction(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, response)
	})
}

// Performance Tests

func TestClientPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}
	
	client := createTestClient()
	
	t.Run("RepositoryOperationLatency", func(t *testing.T) {
		repo := createTestRepository()
		
		start := time.Now()
		_, err := client.CreateRepository(context.Background(), repo)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Less(t, duration, 500*time.Millisecond, "Repository creation should complete in <500ms")
	})
	
	t.Run("PackageRevisionOperationLatency", func(t *testing.T) {
		pkg := createTestPackageRevision()
		
		start := time.Now()
		_, err := client.CreatePackageRevision(context.Background(), pkg)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Less(t, duration, 500*time.Millisecond, "Package revision creation should complete in <500ms")
	})
	
	t.Run("FunctionExecutionLatency", func(t *testing.T) {
		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: "gcr.io/kpt-fn/apply-setters:v0.1.1",
			},
			Resources: []KRMResource{},
		}
		
		start := time.Now()
		_, err := client.ExecuteFunction(context.Background(), req)
		duration := time.Since(start)
		
		assert.NoError(t, err)
		assert.Less(t, duration, 500*time.Millisecond, "Function execution should complete in <500ms")
	})
}

func TestClientConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}
	
	client := createTestClient()
	concurrency := 100
	
	t.Run("ConcurrentRepositoryOperations", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]error, concurrency)
		
		start := time.Now()
		
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				
				repo := createTestRepository()
				repo.Name = fmt.Sprintf("test-repo-%d", index)
				
				_, err := client.CreateRepository(context.Background(), repo)
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
	})
	
	t.Run("ConcurrentPackageRevisionOperations", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]error, concurrency)
		
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				
				pkg := createTestPackageRevision()
				pkg.Name = fmt.Sprintf("test-package-v%d", index)
				
				_, err := client.CreatePackageRevision(context.Background(), pkg)
				results[index] = err
			}(i)
		}
		
		wg.Wait()
		
		// Check all operations succeeded
		for i, err := range results {
			assert.NoError(t, err, "Operation %d should succeed", i)
		}
	})
}

func TestClientMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory tests in short mode")
	}
	
	client := createTestClient()
	
	// Force garbage collection before measuring
	runtime.GC()
	runtime.GC()
	
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	
	// Perform a series of operations
	for i := 0; i < 100; i++ {
		repo := createTestRepository()
		repo.Name = fmt.Sprintf("test-repo-%d", i)
		
		_, err := client.CreateRepository(context.Background(), repo)
		require.NoError(t, err)
		
		pkg := createTestPackageRevision()
		pkg.Name = fmt.Sprintf("test-package-v%d", i)
		
		_, err = client.CreatePackageRevision(context.Background(), pkg)
		require.NoError(t, err)
	}
	
	// Force garbage collection after operations
	runtime.GC()
	runtime.GC()
	
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)
	
	memoryUsed := m2.Alloc - m1.Alloc
	memoryUsedMB := float64(memoryUsed) / (1024 * 1024)
	
	assert.Less(t, memoryUsedMB, 50.0, "Memory usage should be <50MB for 200 operations")
}

// Benchmark Tests

func BenchmarkClientRepositoryOperations(b *testing.B) {
	client := createTestClient()
	
	b.Run("CreateRepository", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			repo := createTestRepository()
			repo.Name = fmt.Sprintf("bench-repo-%d", i)
			
			_, err := client.CreateRepository(context.Background(), repo)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("GetRepository", func(b *testing.B) {
		// Setup
		repo := createTestRepository()
		_, err := client.CreateRepository(context.Background(), repo)
		if err != nil {
			b.Fatal(err)
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.GetRepository(context.Background(), repo.Name)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("ListRepositories", func(b *testing.B) {
		// Setup
		for i := 0; i < 10; i++ {
			repo := createTestRepository()
			repo.Name = fmt.Sprintf("setup-repo-%d", i)
			client.CreateRepository(context.Background(), repo)
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.ListRepositories(context.Background(), &ListOptions{})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkClientPackageRevisionOperations(b *testing.B) {
	client := createTestClient()
	
	b.Run("CreatePackageRevision", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pkg := createTestPackageRevision()
			pkg.Name = fmt.Sprintf("bench-package-v%d", i)
			
			_, err := client.CreatePackageRevision(context.Background(), pkg)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("GetPackageRevision", func(b *testing.B) {
		// Setup
		pkg := createTestPackageRevision()
		_, err := client.CreatePackageRevision(context.Background(), pkg)
		if err != nil {
			b.Fatal(err)
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.GetPackageRevision(context.Background(), pkg.Name)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkClientFunctionOperations(b *testing.B) {
	client := createTestClient()
	
	b.Run("ValidateFunction", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.ValidateFunction(context.Background(), "gcr.io/kpt-fn/apply-setters:v0.1.1")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("ExecuteFunction", func(b *testing.B) {
		req := &FunctionRequest{
			FunctionConfig: FunctionConfig{
				Image: "gcr.io/kpt-fn/apply-setters:v0.1.1",
			},
			Resources: []KRMResource{},
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := client.ExecuteFunction(context.Background(), req)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// Error handling tests

func TestClientErrorHandling(t *testing.T) {
	client := createTestClient()
	
	t.Run("GetNonexistentRepository", func(t *testing.T) {
		_, err := client.GetRepository(context.Background(), "nonexistent")
		assert.Error(t, err)
	})
	
	t.Run("CreateDuplicateRepository", func(t *testing.T) {
		repo := createTestRepository()
		
		// Create first time
		_, err := client.CreateRepository(context.Background(), repo)
		assert.NoError(t, err)
		
		// Try to create again
		_, err = client.CreateRepository(context.Background(), repo)
		assert.Error(t, err)
	})
	
	t.Run("DeleteNonexistentRepository", func(t *testing.T) {
		err := client.DeleteRepository(context.Background(), "nonexistent")
		assert.Error(t, err)
	})
	
	t.Run("InvalidFunctionValidation", func(t *testing.T) {
		validation, err := client.ValidateFunction(context.Background(), "invalid-function")
		assert.NoError(t, err) // Validation doesn't error, but returns invalid result
		assert.NotNil(t, validation)
		assert.False(t, validation.Valid)
		assert.NotEmpty(t, validation.Errors)
	})
}

// Context cancellation tests

func TestClientContextCancellation(t *testing.T) {
	client := createTestClient()
	
	t.Run("CancelledContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		repo := createTestRepository()
		_, err := client.CreateRepository(ctx, repo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
	
	t.Run("TimeoutContext", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		
		time.Sleep(1 * time.Millisecond) // Ensure timeout
		
		repo := createTestRepository()
		_, err := client.CreateRepository(ctx, repo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}

// Integration tests with metrics

func TestClientMetrics(t *testing.T) {
	client := createTestClient()
	
	// Perform some operations
	repo := createTestRepository()
	_, err := client.CreateRepository(context.Background(), repo)
	require.NoError(t, err)
	
	_, err = client.GetRepository(context.Background(), repo.Name)
	require.NoError(t, err)
	
	// Check that metrics are recorded
	assert.NotNil(t, client.metrics)
	
	// Verify metrics are registered with Prometheus
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	
	var found bool
	for _, mf := range metricFamilies {
		if strings.Contains(*mf.Name, "porch_client") {
			found = true
			break
		}
	}
	assert.True(t, found, "Porch client metrics should be registered")
}