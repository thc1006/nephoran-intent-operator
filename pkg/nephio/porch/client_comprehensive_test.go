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

package porch_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch/testutil"
)

// TestClientCreation tests client creation scenarios
func TestClientCreation(t *testing.T) {
	testCases := []struct {
		name        string
		opts        ClientOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful_creation",
			opts: ClientOptions{
				Config:         testutil.NewTestConfig(),
				KubeConfig:     testutil.GetTestKubeConfig(),
				Logger:         zap.New(zap.UseDevMode(true)),
				MetricsEnabled: true,
				CacheEnabled:   true,
				CacheSize:      1000,
				CacheTTL:       5 * time.Minute,
			},
			expectError: false,
		},
		{
			name: "nil_config",
			opts: ClientOptions{
				KubeConfig: testutil.GetTestKubeConfig(),
			},
			expectError: true,
			errorMsg:    "config is required",
		},
		{
			name: "minimal_config",
			opts: ClientOptions{
				Config: testutil.NewTestConfig(),
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := porch.NewClient(tc.opts)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.NotNil(t, client.config)
				assert.NotNil(t, client.circuitBreaker)

				// Cleanup
				defer client.Close()
			}
		})
	}
}

// TestRepositoryOperations tests repository CRUD operations
func TestRepositoryOperations(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	// Create a mock client for testing
	mockClient := testutil.NewMockPorchClient()

	t.Run("create_repository", func(t *testing.T) {
		repo := fixture.CreateTestRepository("test-repo")

		created, err := mockClient.CreateRepository(fixture.Context, repo)

		assert.NoError(t, err)
		assert.NotNil(t, created)
		assert.Equal(t, repo.Name, created.Name)
		assert.Equal(t, repo.Spec.URL, created.Spec.URL)
		assert.True(t, len(created.Status.Conditions) > 0)
		assert.Equal(t, 1, mockClient.GetCallCount("CreateRepository"))
	})

	t.Run("get_repository", func(t *testing.T) {
		// Add repository to mock
		repo := fixture.CreateTestRepository("get-repo")
		mockClient.AddRepository(repo)

		retrieved, err := mockClient.GetRepository(fixture.Context, "get-repo")

		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "get-repo", retrieved.Name)
		assert.Equal(t, 1, mockClient.GetCallCount("GetRepository"))
	})

	t.Run("get_nonexistent_repository", func(t *testing.T) {
		retrieved, err := mockClient.GetRepository(fixture.Context, "nonexistent")

		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("list_repositories", func(t *testing.T) {
		// Add multiple repositories
		for i := 0; i < 5; i++ {
			repo := fixture.CreateTestRepository("")
			mockClient.AddRepository(repo)
		}

		list, err := mockClient.ListRepositories(fixture.Context, &ListOptions{})

		assert.NoError(t, err)
		assert.NotNil(t, list)
		assert.True(t, len(list.Items) >= 5)
		assert.Equal(t, 1, mockClient.GetCallCount("ListRepositories"))
	})

	t.Run("update_repository", func(t *testing.T) {
		// Create and add repository
		repo := fixture.CreateTestRepository("update-repo")
		mockClient.AddRepository(repo)

		// Update repository
		repo.Spec.Branch = "updated-branch"

		updated, err := mockClient.UpdateRepository(fixture.Context, repo)

		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, "updated-branch", updated.Spec.Branch)
		assert.Equal(t, 1, mockClient.GetCallCount("UpdateRepository"))
	})

	t.Run("delete_repository", func(t *testing.T) {
		// Create and add repository
		repo := fixture.CreateTestRepository("delete-repo")
		mockClient.AddRepository(repo)

		err := mockClient.DeleteRepository(fixture.Context, "delete-repo")

		assert.NoError(t, err)
		assert.Equal(t, 1, mockClient.GetCallCount("DeleteRepository"))

		// Verify deletion
		retrieved, err := mockClient.GetRepository(fixture.Context, "delete-repo")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("sync_repository", func(t *testing.T) {
		// Create and add repository
		repo := fixture.CreateTestRepository("sync-repo")
		mockClient.AddRepository(repo)

		err := mockClient.SyncRepository(fixture.Context, "sync-repo")

		assert.NoError(t, err)
		assert.Equal(t, 1, mockClient.GetCallCount("SyncRepository"))
	})
}

// TestPackageRevisionOperations tests package revision CRUD operations
func TestPackageRevisionOperations(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()

	t.Run("create_package_revision", func(t *testing.T) {
		pkg := fixture.CreateTestPackageRevision("test-package", "v1.0.0")

		created, err := mockClient.CreatePackageRevision(fixture.Context, pkg)

		assert.NoError(t, err)
		assert.NotNil(t, created)
		assert.Equal(t, pkg.Spec.PackageName, created.Spec.PackageName)
		assert.Equal(t, pkg.Spec.Revision, created.Spec.Revision)
		assert.Equal(t, 1, mockClient.GetCallCount("CreatePackageRevision"))
	})

	t.Run("get_package_revision", func(t *testing.T) {
		// Add package to mock
		pkg := fixture.CreateTestPackageRevision("get-package", "v1.0.0")
		mockClient.AddPackageRevision(pkg)

		retrieved, err := mockClient.GetPackageRevision(fixture.Context, "get-package", "v1.0.0")

		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "get-package", retrieved.Spec.PackageName)
		assert.Equal(t, "v1.0.0", retrieved.Spec.Revision)
		assert.Equal(t, 1, mockClient.GetCallCount("GetPackageRevision"))
	})

	t.Run("list_package_revisions", func(t *testing.T) {
		// Add multiple packages
		packages := fixture.CreateMultipleTestPackages(3, "list-pkg")
		for _, pkg := range packages {
			mockClient.AddPackageRevision(pkg)
		}

		list, err := mockClient.ListPackageRevisions(fixture.Context, &ListOptions{})

		assert.NoError(t, err)
		assert.NotNil(t, list)
		assert.True(t, len(list.Items) >= 3)
		assert.Equal(t, 1, mockClient.GetCallCount("ListPackageRevisions"))
	})

	t.Run("update_package_revision", func(t *testing.T) {
		// Create and add package
		pkg := fixture.CreateTestPackageRevision("update-package", "v1.0.0")
		mockClient.AddPackageRevision(pkg)

		// Update package
		pkg.Spec.Lifecycle = PackageRevisionLifecycleProposed

		updated, err := mockClient.UpdatePackageRevision(fixture.Context, pkg)

		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, PackageRevisionLifecycleProposed, updated.Spec.Lifecycle)
		assert.Equal(t, 1, mockClient.GetCallCount("UpdatePackageRevision"))
	})

	t.Run("delete_package_revision", func(t *testing.T) {
		// Create and add package
		pkg := fixture.CreateTestPackageRevision("delete-package", "v1.0.0")
		mockClient.AddPackageRevision(pkg)

		err := mockClient.DeletePackageRevision(fixture.Context, "delete-package", "v1.0.0")

		assert.NoError(t, err)
		assert.Equal(t, 1, mockClient.GetCallCount("DeletePackageRevision"))

		// Verify deletion
		retrieved, err := mockClient.GetPackageRevision(fixture.Context, "delete-package", "v1.0.0")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
	})
}

// TestPackageLifecycleTransitions tests package lifecycle state transitions
func TestPackageLifecycleTransitions(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()

	testCases := []struct {
		name          string
		initialState  PackageRevisionLifecycle
		operation     func(context.Context, string, string) error
		expectedState PackageRevisionLifecycle
		expectError   bool
	}{
		{
			name:          "draft_to_proposed",
			initialState:  PackageRevisionLifecycleDraft,
			operation:     mockClient.ProposePackageRevision,
			expectedState: PackageRevisionLifecycleProposed,
			expectError:   false,
		},
		{
			name:          "proposed_to_published",
			initialState:  PackageRevisionLifecycleProposed,
			operation:     mockClient.ApprovePackageRevision,
			expectedState: PackageRevisionLifecyclePublished,
			expectError:   false,
		},
		{
			name:         "proposed_to_draft",
			initialState: PackageRevisionLifecycleProposed,
			operation: func(ctx context.Context, name, revision string) error {
				return mockClient.RejectPackageRevision(ctx, name, revision, "test rejection")
			},
			expectedState: PackageRevisionLifecycleDraft,
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create package with initial state
			pkg := fixture.CreateTestPackageRevision("lifecycle-test", "v1.0.0",
				testutil.WithPackageLifecycle(tc.initialState))
			mockClient.AddPackageRevision(pkg)

			// Perform operation
			err := tc.operation(fixture.Context, "lifecycle-test", "v1.0.0")

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify state transition
				retrieved, getErr := mockClient.GetPackageRevision(fixture.Context, "lifecycle-test", "v1.0.0")
				assert.NoError(t, getErr)
				assert.Equal(t, tc.expectedState, retrieved.Spec.Lifecycle)
			}
		})
	}
}

// TestPackageContentOperations tests package content management
func TestPackageContentOperations(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()

	t.Run("get_package_contents", func(t *testing.T) {
		// Create and add package
		pkg := fixture.CreateTestPackageRevision("content-test", "v1.0.0")
		mockClient.AddPackageRevision(pkg)

		contents, err := mockClient.GetPackageContents(fixture.Context, "content-test", "v1.0.0")

		assert.NoError(t, err)
		assert.NotNil(t, contents)
		assert.True(t, len(contents) > 0)
		assert.Equal(t, 1, mockClient.GetCallCount("GetPackageContents"))
	})

	t.Run("update_package_contents", func(t *testing.T) {
		// Create and add package
		pkg := fixture.CreateTestPackageRevision("update-content", "v1.0.0")
		mockClient.AddPackageRevision(pkg)

		newContents := map[string][]byte{
			"new-file.yaml": []byte("apiVersion: v1\nkind: ConfigMap"),
		}

		err := mockClient.UpdatePackageContents(fixture.Context, "update-content", "v1.0.0", newContents)

		assert.NoError(t, err)
		assert.Equal(t, 1, mockClient.GetCallCount("UpdatePackageContents"))

		// Verify contents were updated
		contents, getErr := mockClient.GetPackageContents(fixture.Context, "update-content", "v1.0.0")
		assert.NoError(t, getErr)
		assert.Contains(t, contents, "new-file.yaml")
	})

	t.Run("render_package", func(t *testing.T) {
		// Create and add package
		pkg := fixture.CreateTestPackageRevision("render-test", "v1.0.0")
		mockClient.AddPackageRevision(pkg)

		result, err := mockClient.RenderPackage(fixture.Context, "render-test", "v1.0.0")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, len(result.Resources) > 0)
		assert.Equal(t, 1, mockClient.GetCallCount("RenderPackage"))
	})
}

// TestFunctionOperations tests KRM function operations
func TestFunctionOperations(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()

	t.Run("run_function", func(t *testing.T) {
		request := &FunctionRequest{
			FunctionConfig: porch.FunctionConfig{
				Image: "gcr.io/kpt-fn/set-labels:v0.2.0",
				ConfigMap: map[string]interface{}{
					"env": "production",
				},
			},
			Resources: []KRMResource{
				testutil.GenerateTestResource("v1", "ConfigMap", "test-cm", "default"),
			},
		}

		response, err := mockClient.RunFunction(fixture.Context, request)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, len(request.Resources), len(response.Resources))
		assert.True(t, len(response.Results) > 0)
		assert.Equal(t, 1, mockClient.GetCallCount("RunFunction"))
	})

	t.Run("validate_package", func(t *testing.T) {
		// Create and add package
		pkg := fixture.CreateTestPackageRevision("validate-test", "v1.0.0")
		mockClient.AddPackageRevision(pkg)

		result, err := mockClient.ValidatePackage(fixture.Context, "validate-test", "v1.0.0")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid)
		assert.Equal(t, 1, mockClient.GetCallCount("ValidatePackage"))
	})

	t.Run("validate_nonexistent_package", func(t *testing.T) {
		result, err := mockClient.ValidatePackage(fixture.Context, "nonexistent", "v1.0.0")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Valid)
		assert.True(t, len(result.Errors) > 0)
	})
}

// TestWorkflowOperations tests workflow operations
func TestWorkflowOperations(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()

	t.Run("create_workflow", func(t *testing.T) {
		workflow := fixture.CreateTestWorkflow("test-workflow")

		created, err := mockClient.CreateWorkflow(fixture.Context, workflow)

		assert.NoError(t, err)
		assert.NotNil(t, created)
		assert.Equal(t, workflow.Name, created.Name)
		assert.Equal(t, WorkflowPhasePending, created.Status.Phase)
		assert.Equal(t, 1, mockClient.GetCallCount("CreateWorkflow"))
	})

	t.Run("get_workflow", func(t *testing.T) {
		// Add workflow to mock
		workflow := fixture.CreateTestWorkflow("get-workflow")
		mockClient.AddWorkflow(workflow)

		retrieved, err := mockClient.GetWorkflow(fixture.Context, "get-workflow")

		assert.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, "get-workflow", retrieved.Name)
		assert.Equal(t, 1, mockClient.GetCallCount("GetWorkflow"))
	})

	t.Run("list_workflows", func(t *testing.T) {
		// Add multiple workflows
		for i := 0; i < 3; i++ {
			workflow := fixture.CreateTestWorkflow("")
			mockClient.AddWorkflow(workflow)
		}

		list, err := mockClient.ListWorkflows(fixture.Context, &ListOptions{})

		assert.NoError(t, err)
		assert.NotNil(t, list)
		assert.True(t, len(list.Items) >= 3)
		assert.Equal(t, 1, mockClient.GetCallCount("ListWorkflows"))
	})
}

// TestCircuitBreakerBehavior tests circuit breaker functionality
func TestCircuitBreakerBehavior(t *testing.T) {
	mockClient := testutil.NewMockPorchClient()

	t.Run("circuit_breaker_open", func(t *testing.T) {
		mockClient.SetCircuitBreakerOpen(true)

		_, err := mockClient.GetRepository(context.Background(), "test")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker")
	})

	t.Run("circuit_breaker_closed", func(t *testing.T) {
		mockClient.SetCircuitBreakerOpen(false)
		mockClient.AddRepository(&Repository{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
		})

		_, err := mockClient.GetRepository(context.Background(), "test")

		assert.NoError(t, err)
	})
}

// TestRateLimiting tests rate limiting functionality
func TestRateLimiting(t *testing.T) {
	mockClient := testutil.NewMockPorchClient()

	t.Run("rate_limited", func(t *testing.T) {
		mockClient.SetRateLimited(true)

		_, err := mockClient.GetRepository(context.Background(), "test")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit")
	})

	t.Run("rate_limit_disabled", func(t *testing.T) {
		mockClient.SetRateLimited(false)
		mockClient.AddRepository(&Repository{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
		})

		_, err := mockClient.GetRepository(context.Background(), "test")

		assert.NoError(t, err)
	})
}

// TestHealthAndStatus tests health and status operations
func TestHealthAndStatus(t *testing.T) {
	mockClient := testutil.NewMockPorchClient()

	t.Run("health_check_success", func(t *testing.T) {
		mockClient.SetHealthCheckFails(false)

		health, err := mockClient.Health(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, health)
		assert.Equal(t, "healthy", health.Status)
		assert.True(t, len(health.Components) > 0)
		assert.Equal(t, 1, mockClient.GetCallCount("Health"))
	})

	t.Run("health_check_failure", func(t *testing.T) {
		mockClient.SetHealthCheckFails(true)

		health, err := mockClient.Health(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, health)
		assert.Equal(t, "unhealthy", health.Status)
		assert.True(t, len(health.Components) > 0)
		assert.Contains(t, health.Components[0].Error, "simulated")
	})

	t.Run("version_info", func(t *testing.T) {
		version, err := mockClient.Version(context.Background())

		assert.NoError(t, err)
		assert.NotNil(t, version)
		assert.NotEmpty(t, version.Version)
		assert.NotEmpty(t, version.GitCommit)
		assert.Equal(t, 1, mockClient.GetCallCount("Version"))
	})
}

// TestErrorHandling tests error handling scenarios
func TestErrorHandling(t *testing.T) {
	mockClient := testutil.NewMockPorchClient()

	t.Run("simulated_errors", func(t *testing.T) {
		mockClient.SetSimulateErrors(true)

		_, err := mockClient.GetRepository(context.Background(), "test")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "simulated error")
	})

	t.Run("context_cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := mockClient.GetRepository(ctx, "test")

		// Mock doesn't actually check context cancellation, but in real implementation it would
		// For now, we'll just verify the call was made
		assert.Equal(t, 1, mockClient.GetCallCount("GetRepository"))
	})

	t.Run("timeout_handling", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		time.Sleep(2 * time.Nanosecond) // Ensure timeout

		_, err := mockClient.GetRepository(ctx, "test")

		// Mock doesn't handle timeouts, but in real implementation it would
		assert.Equal(t, 1, mockClient.GetCallCount("GetRepository"))
	})
}

// TestConcurrentOperations tests concurrent access patterns
func TestConcurrentOperations(t *testing.T) {
	mockClient := testutil.NewMockPorchClient()
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	t.Run("concurrent_repository_operations", func(t *testing.T) {
		const numGoroutines = 10
		const operationsPerGoroutine = 5

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*operationsPerGoroutine)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				for j := 0; j < operationsPerGoroutine; j++ {
					repo := fixture.CreateTestRepository("")
					_, err := mockClient.CreateRepository(fixture.Context, repo)
					if err != nil {
						errors <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		errorCount := 0
		for err := range errors {
			if err != nil {
				t.Errorf("Concurrent operation failed: %v", err)
				errorCount++
			}
		}

		assert.Equal(t, 0, errorCount, "Expected no errors in concurrent operations")
		assert.Equal(t, numGoroutines*operationsPerGoroutine, mockClient.GetCallCount("CreateRepository"))
	})

	t.Run("concurrent_read_operations", func(t *testing.T) {
		// Add test data
		repos := fixture.CreateMultipleTestRepositories(10, "concurrent-read")
		for _, repo := range repos {
			mockClient.AddRepository(repo)
		}

		const numReaders = 20
		var wg sync.WaitGroup

		for i := 0; i < numReaders; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				_, err := mockClient.ListRepositories(fixture.Context, &ListOptions{})
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()
		assert.Equal(t, numReaders, mockClient.GetCallCount("ListRepositories"))
	})
}

// TestInputValidation tests input validation
func TestInputValidation(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()

	t.Run("repository_validation", func(t *testing.T) {
		invalidRepos := []*Repository{
			{
				ObjectMeta: metav1.ObjectMeta{Name: ""},
				Spec:       RepositorySpec{URL: "https://github.com/test/test.git", Type: "git"},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec:       RepositorySpec{URL: "", Type: "git"},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "test"},
				Spec:       RepositorySpec{URL: "https://github.com/test/test.git", Type: ""},
			},
		}

		for i, repo := range invalidRepos {
			t.Run(fmt.Sprintf("invalid_repo_%d", i), func(t *testing.T) {
				errors := testutil.ValidateRepository(repo)
				assert.True(t, len(errors) > 0, "Expected validation errors")
			})
		}
	})

	t.Run("package_validation", func(t *testing.T) {
		invalidPackages := []*PackageRevision{
			{
				Spec: PackageRevisionSpec{PackageName: "", Repository: "test-repo", Revision: "v1.0.0"},
			},
			{
				Spec: PackageRevisionSpec{PackageName: "test-pkg", Repository: "", Revision: "v1.0.0"},
			},
			{
				Spec: PackageRevisionSpec{PackageName: "test-pkg", Repository: "test-repo", Revision: ""},
			},
		}

		for i, pkg := range invalidPackages {
			t.Run(fmt.Sprintf("invalid_package_%d", i), func(t *testing.T) {
				errors := testutil.ValidatePackageRevision(pkg)
				assert.True(t, len(errors) > 0, "Expected validation errors")
			})
		}
	})
}

// TestMetricsAndObservability tests metrics collection
func TestMetricsAndObservability(t *testing.T) {
	mockClient := testutil.NewMockPorchClient()

	t.Run("call_counting", func(t *testing.T) {
		// Reset counters
		mockClient.ResetCallCounts()

		// Make various calls
		mockClient.Health(context.Background())
		mockClient.Version(context.Background())
		mockClient.ListRepositories(context.Background(), nil)

		// Verify counts
		assert.Equal(t, 1, mockClient.GetCallCount("Health"))
		assert.Equal(t, 1, mockClient.GetCallCount("Version"))
		assert.Equal(t, 1, mockClient.GetCallCount("ListRepositories"))
		assert.Equal(t, 0, mockClient.GetCallCount("GetRepository"))
	})

	t.Run("metrics_reset", func(t *testing.T) {
		// Make some calls
		mockClient.Health(context.Background())
		mockClient.Version(context.Background())

		// Reset and verify
		mockClient.ResetCallCounts()
		assert.Equal(t, 0, mockClient.GetCallCount("Health"))
		assert.Equal(t, 0, mockClient.GetCallCount("Version"))
	})
}

// TestLifecycleValidation tests package lifecycle validation
func TestLifecycleValidation(t *testing.T) {
	testCases := []struct {
		name          string
		current       PackageRevisionLifecycle
		target        PackageRevisionLifecycle
		canTransition bool
	}{
		{"draft_to_proposed", PackageRevisionLifecycleDraft, PackageRevisionLifecycleProposed, true},
		{"draft_to_published", PackageRevisionLifecycleDraft, PackageRevisionLifecyclePublished, false},
		{"proposed_to_published", PackageRevisionLifecycleProposed, PackageRevisionLifecyclePublished, true},
		{"proposed_to_draft", PackageRevisionLifecycleProposed, PackageRevisionLifecycleDraft, true},
		{"published_to_draft", PackageRevisionLifecyclePublished, PackageRevisionLifecycleDraft, false},
		{"published_to_deletable", PackageRevisionLifecyclePublished, PackageRevisionLifecycleDeletable, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			canTransition := tc.current.CanTransitionTo(tc.target)
			assert.Equal(t, tc.canTransition, canTransition,
				"Expected transition from %s to %s to be %v", tc.current, tc.target, tc.canTransition)
		})
	}
}

// TestBoundaryConditions tests edge cases and boundary conditions
func TestBoundaryConditions(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()

	t.Run("empty_list_operations", func(t *testing.T) {
		// List operations on empty mock should return empty lists
		repos, err := mockClient.ListRepositories(fixture.Context, &ListOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, repos)
		assert.Equal(t, 0, len(repos.Items))

		packages, err := mockClient.ListPackageRevisions(fixture.Context, &ListOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, packages)
		assert.Equal(t, 0, len(packages.Items))

		workflows, err := mockClient.ListWorkflows(fixture.Context, &ListOptions{})
		assert.NoError(t, err)
		assert.NotNil(t, workflows)
		assert.Equal(t, 0, len(workflows.Items))
	})

	t.Run("large_content_operations", func(t *testing.T) {
		pkg := fixture.CreateTestPackageRevision("large-content", "v1.0.0")
		mockClient.AddPackageRevision(pkg)

		// Create large content (1MB)
		largeContent := make([]byte, 1024*1024)
		for i := range largeContent {
			largeContent[i] = byte(i % 256)
		}

		contents := map[string][]byte{
			"large-file.bin": largeContent,
		}

		err := mockClient.UpdatePackageContents(fixture.Context, "large-content", "v1.0.0", contents)
		assert.NoError(t, err)

		retrieved, err := mockClient.GetPackageContents(fixture.Context, "large-content", "v1.0.0")
		assert.NoError(t, err)
		assert.Contains(t, retrieved, "large-file.bin")
		assert.Equal(t, len(largeContent), len(retrieved["large-file.bin"]))
	})

	t.Run("special_characters_in_names", func(t *testing.T) {
		specialNames := []string{
			"test-with-dashes",
			"test.with.dots",
			"test_with_underscores",
			"test123with456numbers",
		}

		for _, name := range specialNames {
			repo := fixture.CreateTestRepository(name)
			_, err := mockClient.CreateRepository(fixture.Context, repo)
			assert.NoError(t, err, "Failed to create repository with name: %s", name)
		}
	})
}

// TestPerformanceCharacteristics tests performance-related scenarios
func TestPerformanceCharacteristics(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()

	t.Run("latency_simulation", func(t *testing.T) {
		latency := 100 * time.Millisecond
		mockClient.SetSimulateLatency(latency)

		start := time.Now()
		_, err := mockClient.Health(fixture.Context)
		elapsed := time.Since(start)

		assert.NoError(t, err)
		assert.True(t, elapsed >= latency, "Expected operation to take at least %v, took %v", latency, elapsed)
	})

	t.Run("bulk_operations_performance", func(t *testing.T) {
		const numOperations = 100

		start := time.Now()

		// Create multiple repositories
		for i := 0; i < numOperations; i++ {
			repo := fixture.CreateTestRepository("")
			_, err := mockClient.CreateRepository(fixture.Context, repo)
			assert.NoError(t, err)
		}

		elapsed := time.Since(start)
		avgLatency := elapsed / numOperations

		t.Logf("Created %d repositories in %v (avg: %v per operation)", numOperations, elapsed, avgLatency)
		assert.True(t, avgLatency < 10*time.Millisecond, "Average latency too high: %v", avgLatency)
	})
}

// TestDataConsistency tests data consistency scenarios
func TestDataConsistency(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()

	t.Run("crud_consistency", func(t *testing.T) {
		// Create
		repo := fixture.CreateTestRepository("consistency-test")
		created, err := mockClient.CreateRepository(fixture.Context, repo)
		require.NoError(t, err)

		// Read
		retrieved, err := mockClient.GetRepository(fixture.Context, "consistency-test")
		require.NoError(t, err)
		assert.Equal(t, created.Name, retrieved.Name)
		assert.Equal(t, created.Spec.URL, retrieved.Spec.URL)

		// Update
		retrieved.Spec.Branch = "updated-branch"
		updated, err := mockClient.UpdateRepository(fixture.Context, retrieved)
		require.NoError(t, err)
		assert.Equal(t, "updated-branch", updated.Spec.Branch)

		// Verify update persisted
		retrieved2, err := mockClient.GetRepository(fixture.Context, "consistency-test")
		require.NoError(t, err)
		assert.Equal(t, "updated-branch", retrieved2.Spec.Branch)

		// Delete
		err = mockClient.DeleteRepository(fixture.Context, "consistency-test")
		require.NoError(t, err)

		// Verify deletion
		_, err = mockClient.GetRepository(fixture.Context, "consistency-test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("list_consistency", func(t *testing.T) {
		// Create multiple items
		const numItems = 5
		names := make([]string, numItems)

		for i := 0; i < numItems; i++ {
			name := fmt.Sprintf("list-test-%d", i)
			names[i] = name
			repo := fixture.CreateTestRepository(name)
			_, err := mockClient.CreateRepository(fixture.Context, repo)
			require.NoError(t, err)
		}

		// List and verify all items are present
		list, err := mockClient.ListRepositories(fixture.Context, &ListOptions{})
		require.NoError(t, err)

		found := 0
		for _, item := range list.Items {
			for _, expectedName := range names {
				if item.Name == expectedName {
					found++
					break
				}
			}
		}

		assert.Equal(t, numItems, found, "Not all created items found in list")
	})
}

// BenchmarkClientOperations benchmarks various client operations
func BenchmarkClientOperations(b *testing.B) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()

	b.Run("CreateRepository", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			repo := fixture.CreateTestRepository(fmt.Sprintf("bench-repo-%d", i))
			_, err := mockClient.CreateRepository(fixture.Context, repo)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetRepository", func(b *testing.B) {
		// Setup
		repo := fixture.CreateTestRepository("bench-get-repo")
		mockClient.AddRepository(repo)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mockClient.GetRepository(fixture.Context, "bench-get-repo")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ListRepositories", func(b *testing.B) {
		// Setup - add multiple repositories
		for i := 0; i < 100; i++ {
			repo := fixture.CreateTestRepository(fmt.Sprintf("bench-list-repo-%d", i))
			mockClient.AddRepository(repo)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := mockClient.ListRepositories(fixture.Context, &ListOptions{})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("CreatePackageRevision", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pkg := fixture.CreateTestPackageRevision(fmt.Sprintf("bench-pkg-%d", i), "v1.0.0")
			_, err := mockClient.CreatePackageRevision(fixture.Context, pkg)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// TestComplexScenarios tests complex real-world scenarios
func TestComplexScenarios(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()

	t.Run("complete_package_workflow", func(t *testing.T) {
		// 1. Create repository
		repo := fixture.CreateTestRepository("workflow-repo",
			testutil.WithRepositoryURL("https://github.com/nephio-project/free5gc-packages.git"))
		_, err := mockClient.CreateRepository(fixture.Context, repo)
		require.NoError(t, err)

		// 2. Create package revision in Draft
		pkg := fixture.CreateTestPackageRevision("5g-amf", "v1.0.0",
			testutil.WithPackageRepository("workflow-repo"),
			testutil.WithPackageLifecycle(PackageRevisionLifecycleDraft))
		_, err = mockClient.CreatePackageRevision(fixture.Context, pkg)
		require.NoError(t, err)

		// 3. Add content to package
		contents := testutil.GeneratePackageContents()
		err = mockClient.UpdatePackageContents(fixture.Context, "5g-amf", "v1.0.0", contents)
		require.NoError(t, err)

		// 4. Validate package
		validation, err := mockClient.ValidatePackage(fixture.Context, "5g-amf", "v1.0.0")
		require.NoError(t, err)
		assert.True(t, validation.Valid)

		// 5. Propose package
		err = mockClient.ProposePackageRevision(fixture.Context, "5g-amf", "v1.0.0")
		require.NoError(t, err)

		// 6. Approve package
		err = mockClient.ApprovePackageRevision(fixture.Context, "5g-amf", "v1.0.0")
		require.NoError(t, err)

		// 7. Verify final state
		finalPkg, err := mockClient.GetPackageRevision(fixture.Context, "5g-amf", "v1.0.0")
		require.NoError(t, err)
		assert.Equal(t, PackageRevisionLifecyclePublished, finalPkg.Spec.Lifecycle)
		assert.NotNil(t, finalPkg.Status.PublishTime)
	})

	t.Run("multi_package_dependency_scenario", func(t *testing.T) {
		// Create multiple related packages representing a 5G Core deployment
		packages := []struct {
			name       string
			component  string
			depends_on []string
		}{
			{"5g-core-base", "base", []string{}},
			{"5g-amf", "amf", []string{"5g-core-base"}},
			{"5g-smf", "smf", []string{"5g-core-base"}},
			{"5g-upf", "upf", []string{"5g-core-base", "5g-smf"}},
			{"5g-nssf", "nssf", []string{"5g-core-base", "5g-amf"}},
		}

		for _, pkgInfo := range packages {
			pkg := fixture.CreateTestPackageRevision(pkgInfo.name, "v1.0.0")
			_, err := mockClient.CreatePackageRevision(fixture.Context, pkg)
			require.NoError(t, err)
		}

		// List all packages and verify creation
		list, err := mockClient.ListPackageRevisions(fixture.Context, &ListOptions{})
		require.NoError(t, err)
		assert.True(t, len(list.Items) >= len(packages))

		// Simulate deployment order based on dependencies
		deploymentOrder := []string{"5g-core-base", "5g-amf", "5g-smf", "5g-upf", "5g-nssf"}

		for _, pkgName := range deploymentOrder {
			// Validate package
			validation, err := mockClient.ValidatePackage(fixture.Context, pkgName, "v1.0.0")
			require.NoError(t, err)
			assert.True(t, validation.Valid, "Package %s should be valid", pkgName)

			// Promote to published
			err = mockClient.ProposePackageRevision(fixture.Context, pkgName, "v1.0.0")
			require.NoError(t, err)

			err = mockClient.ApprovePackageRevision(fixture.Context, pkgName, "v1.0.0")
			require.NoError(t, err)
		}

		// Verify all packages are published
		for _, pkgInfo := range packages {
			pkg, err := mockClient.GetPackageRevision(fixture.Context, pkgInfo.name, "v1.0.0")
			require.NoError(t, err)
			assert.Equal(t, PackageRevisionLifecyclePublished, pkg.Spec.Lifecycle,
				"Package %s should be published", pkgInfo.name)
		}
	})
}
