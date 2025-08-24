//go:build integration

package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

const (
	chaosTestPackages     = 100
	networkPartitionDelay = 500 * time.Millisecond
	slowResponseDelay     = 250 * time.Millisecond
)

func simulateNetworkLatency(fn func() error) error {
	randomDelay := time.Duration(rand.Intn(500)) * time.Millisecond
	time.Sleep(randomDelay)
	return fn()
}

// MockPorchClient provides a mock implementation for testing
type MockPorchClient struct {
	mu           sync.Mutex
	packages     map[string]*porch.PackageRevision
	repositories map[string]*porch.Repository
	failures     map[string]bool
}

func NewMockPorchClient() *MockPorchClient {
	return &MockPorchClient{
		packages:     make(map[string]*porch.PackageRevision),
		repositories: make(map[string]*porch.Repository),
		failures:     make(map[string]bool),
	}
}

func (m *MockPorchClient) GetPackageRevision(ctx context.Context, name, revision string) (*porch.PackageRevision, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := fmt.Sprintf("%s@%s", name, revision)
	if m.failures[key] {
		return nil, fmt.Errorf("simulated failure for %s", key)
	}
	
	if pkg, exists := m.packages[key]; exists {
		return pkg, nil
	}
	return nil, errors.NewNotFound(porch.Resource("packagerevision"), key)
}

func (m *MockPorchClient) CreatePackageRevision(ctx context.Context, pkg *porch.PackageRevision) (*porch.PackageRevision, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := fmt.Sprintf("%s@%s", pkg.Spec.PackageName, pkg.Spec.Revision)
	if m.failures[key] {
		return nil, fmt.Errorf("simulated failure for %s", key)
	}
	
	// Create a copy of the package
	created := &porch.PackageRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pkg.Name,
			Namespace: pkg.Namespace,
		},
		Spec: pkg.Spec,
		Status: porch.PackageRevisionStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
					Reason: "Created",
				},
			},
		},
	}
	
	m.packages[key] = created
	return created, nil
}

func (m *MockPorchClient) ListPackageRevisions(ctx context.Context, opts *porch.ListOptions) (*porch.PackageRevisionList, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	var items []porch.PackageRevision
	for _, pkg := range m.packages {
		items = append(items, *pkg)
	}
	
	return &porch.PackageRevisionList{Items: items}, nil
}

func (m *MockPorchClient) SetFailure(key string, fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failures[key] = fail
}

func (m *MockPorchClient) UpdatePackageRevision(ctx context.Context, pkg *porch.PackageRevision) (*porch.PackageRevision, error) {
	return m.CreatePackageRevision(ctx, pkg)
}

func (m *MockPorchClient) DeletePackageRevision(ctx context.Context, name, revision string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	key := fmt.Sprintf("%s@%s", name, revision)
	delete(m.packages, key)
	return nil
}

func (m *MockPorchClient) ApprovePackageRevision(ctx context.Context, name, revision string) error {
	return nil
}

func (m *MockPorchClient) ProposePackageRevision(ctx context.Context, name, revision string) error {
	return nil
}

func (m *MockPorchClient) RejectPackageRevision(ctx context.Context, name, revision, reason string) error {
	return nil
}

func (m *MockPorchClient) GetPackageContents(ctx context.Context, name, revision string) (map[string][]byte, error) {
	return make(map[string][]byte), nil
}

func (m *MockPorchClient) UpdatePackageContents(ctx context.Context, name, revision string, contents map[string][]byte) error {
	return nil
}

func (m *MockPorchClient) RenderPackage(ctx context.Context, name, revision string) (*porch.RenderResult, error) {
	return &porch.RenderResult{}, nil
}

func (m *MockPorchClient) RunFunction(ctx context.Context, req *porch.FunctionRequest) (*porch.FunctionResponse, error) {
	return &porch.FunctionResponse{}, nil
}

func (m *MockPorchClient) ValidatePackage(ctx context.Context, name, revision string) (*porch.ValidationResult, error) {
	return &porch.ValidationResult{Valid: true}, nil
}

func (m *MockPorchClient) GetRepository(ctx context.Context, name string) (*porch.Repository, error) {
	return nil, errors.NewNotFound(porch.Resource("repository"), name)
}

func (m *MockPorchClient) ListRepositories(ctx context.Context, opts *porch.ListOptions) (*porch.RepositoryList, error) {
	return &porch.RepositoryList{}, nil
}

func (m *MockPorchClient) CreateRepository(ctx context.Context, repo *porch.Repository) (*porch.Repository, error) {
	return repo, nil
}

func (m *MockPorchClient) UpdateRepository(ctx context.Context, repo *porch.Repository) (*porch.Repository, error) {
	return repo, nil
}

func (m *MockPorchClient) DeleteRepository(ctx context.Context, name string) error {
	return nil
}

func (m *MockPorchClient) SyncRepository(ctx context.Context, name string) error {
	return nil
}

func (m *MockPorchClient) GetWorkflow(ctx context.Context, name string) (*porch.Workflow, error) {
	return nil, errors.NewNotFound(porch.Resource("workflow"), name)
}

func (m *MockPorchClient) ListWorkflows(ctx context.Context, opts *porch.ListOptions) (*porch.WorkflowList, error) {
	return &porch.WorkflowList{}, nil
}

func (m *MockPorchClient) CreateWorkflow(ctx context.Context, workflow *porch.Workflow) (*porch.Workflow, error) {
	return workflow, nil
}

func (m *MockPorchClient) UpdateWorkflow(ctx context.Context, workflow *porch.Workflow) (*porch.Workflow, error) {
	return workflow, nil
}

func (m *MockPorchClient) DeleteWorkflow(ctx context.Context, name string) error {
	return nil
}

func (m *MockPorchClient) Health(ctx context.Context) (*porch.HealthStatus, error) {
	return &porch.HealthStatus{Status: "healthy"}, nil
}

func (m *MockPorchClient) Version(ctx context.Context) (*porch.VersionInfo, error) {
	return &porch.VersionInfo{Version: "1.0.0"}, nil
}

func TestPorchResilienceUnderFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping resilience test in short mode")
	}

	porchClient := NewMockPorchClient()

	t.Run("NetworkPartitionAndRecovery", func(t *testing.T) {
		var wg sync.WaitGroup
		var successCount, failureCount int32
		var mu sync.Mutex

		for i := 0; i < chaosTestPackages; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				// Simulate network partition with probabilistic failure
				pkg := &porch.PackageRevision{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("chaos-test-%d", idx),
						Namespace: "default",
					},
					Spec: porch.PackageRevisionSpec{
						PackageName: fmt.Sprintf("chaos-pkg-%d", idx),
						Repository:  "test-repo",
						Revision:    "v1.0.0",
						Lifecycle:   porch.PackageRevisionLifecycleDraft,
					},
				}

				err := simulateNetworkLatency(func() error {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()

					// Simulate random failures
					if rand.Float32() < 0.3 { // 30% failure rate
						key := fmt.Sprintf("%s@%s", pkg.Spec.PackageName, pkg.Spec.Revision)
						porchClient.SetFailure(key, true)
					}

					_, createErr := porchClient.CreatePackageRevision(ctx, pkg)
					return createErr
				})

				mu.Lock()
				if err != nil {
					failureCount++
				} else {
					successCount++
				}
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		t.Logf("Success: %d, Failures: %d", successCount, failureCount)
		assert.Greater(t, successCount, int32(0), "Should have some successes")
	})

	t.Run("SlowResponseHandling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		start := time.Now()
		err := simulateNetworkLatency(func() error {
			time.Sleep(slowResponseDelay)
			return nil
		})

		duration := time.Since(start)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, duration, slowResponseDelay)
		t.Logf("Operation took %v", duration)
	})
}

func TestPorchConcurrentOperations(t *testing.T) {
	porchClient := NewMockPorchClient()
	const concurrentOps = 50

	t.Run("ConcurrentPackageCreation", func(t *testing.T) {
		var wg sync.WaitGroup
		var successCount int32
		var mu sync.Mutex

		for i := 0; i < concurrentOps; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				pkg := &porch.PackageRevision{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("concurrent-test-%d", idx),
						Namespace: "default",
					},
					Spec: porch.PackageRevisionSpec{
						PackageName: fmt.Sprintf("concurrent-pkg-%d", idx),
						Repository:  "test-repo",
						Revision:    "v1.0.0",
						Lifecycle:   porch.PackageRevisionLifecycleDraft,
					},
				}

				ctx := context.Background()
				_, err := porchClient.CreatePackageRevision(ctx, pkg)
				if err == nil {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		assert.Equal(t, int32(concurrentOps), successCount)
		t.Logf("Successfully created %d packages concurrently", successCount)
	})

	t.Run("ConcurrentReadOperations", func(t *testing.T) {
		// First create some packages
		for i := 0; i < 10; i++ {
			pkg := &porch.PackageRevision{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("read-test-%d", i),
					Namespace: "default",
				},
				Spec: porch.PackageRevisionSpec{
					PackageName: fmt.Sprintf("read-pkg-%d", i),
					Repository:  "test-repo",
					Revision:    "v1.0.0",
					Lifecycle:   porch.PackageRevisionLifecycleDraft,
				},
			}
			_, err := porchClient.CreatePackageRevision(context.Background(), pkg)
			require.NoError(t, err)
		}

		var wg sync.WaitGroup
		var readCount int32
		var mu sync.Mutex

		for i := 0; i < concurrentOps; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				pkgIdx := idx % 10
				_, err := porchClient.GetPackageRevision(
					context.Background(),
					fmt.Sprintf("read-pkg-%d", pkgIdx),
					"v1.0.0",
				)
				if err == nil {
					mu.Lock()
					readCount++
					mu.Unlock()
				}
			}(i)
		}

		wg.Wait()

		assert.Equal(t, int32(concurrentOps), readCount)
		t.Logf("Successfully read packages %d times concurrently", readCount)
	})
}

func TestPorchErrorHandling(t *testing.T) {
	porchClient := NewMockPorchClient()

	t.Run("HandleNonExistentPackage", func(t *testing.T) {
		_, err := porchClient.GetPackageRevision(
			context.Background(),
			"non-existent-package",
			"v1.0.0",
		)
		assert.Error(t, err)
		assert.True(t, errors.IsNotFound(err))
	})

	t.Run("HandleSimulatedFailures", func(t *testing.T) {
		porchClient.SetFailure("failing-pkg@v1.0.0", true)

		pkg := &porch.PackageRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "failing-test",
				Namespace: "default",
			},
			Spec: porch.PackageRevisionSpec{
				PackageName: "failing-pkg",
				Repository:  "test-repo",
				Revision:    "v1.0.0",
				Lifecycle:   porch.PackageRevisionLifecycleDraft,
			},
		}

		_, err := porchClient.CreatePackageRevision(context.Background(), pkg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "simulated failure")
	})
}

func TestPorchPackageLifecycle(t *testing.T) {
	porchClient := NewMockPorchClient()

	t.Run("CompletePackageLifecycle", func(t *testing.T) {
		pkg := &porch.PackageRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lifecycle-test",
				Namespace: "default",
			},
			Spec: porch.PackageRevisionSpec{
				PackageName: "lifecycle-pkg",
				Repository:  "test-repo",
				Revision:    "v1.0.0",
				Lifecycle:   porch.PackageRevisionLifecycleDraft,
			},
		}

		// Create package
		created, err := porchClient.CreatePackageRevision(context.Background(), pkg)
		require.NoError(t, err)
		assert.NotNil(t, created)
		assert.Equal(t, pkg.Spec.PackageName, created.Spec.PackageName)

		// Retrieve package
		retrieved, err := porchClient.GetPackageRevision(
			context.Background(),
			pkg.Spec.PackageName,
			pkg.Spec.Revision,
		)
		require.NoError(t, err)
		assert.Equal(t, created.Spec.PackageName, retrieved.Spec.PackageName)

		// Delete package
		err = porchClient.DeletePackageRevision(
			context.Background(),
			pkg.Spec.PackageName,
			pkg.Spec.Revision,
		)
		assert.NoError(t, err)
	})
}

func TestPorchHealthCheck(t *testing.T) {
	porchClient := NewMockPorchClient()

	t.Run("HealthStatus", func(t *testing.T) {
		health, err := porchClient.Health(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, health)
		assert.Equal(t, "healthy", health.Status)
	})

	t.Run("VersionInfo", func(t *testing.T) {
		version, err := porchClient.Version(context.Background())
		require.NoError(t, err)
		assert.NotNil(t, version)
		assert.NotEmpty(t, version.Version)
	})
}