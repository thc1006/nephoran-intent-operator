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
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// IntegrationTestSuite provides integration testing for the complete Porch client
type IntegrationTestSuite struct {
	suite.Suite
	client  PorchClient
	config  *Config
	cleanup []func()
}

// SetupSuite initializes the test suite
func (suite *IntegrationTestSuite) SetupSuite() {
	// Skip integration tests if not running in integration mode
	if testing.Short() {
		suite.T().Skip("Skipping integration tests in short mode")
	}
	
	// Check if we should skip integration tests based on environment
	if os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		suite.T().Skip("Integration tests skipped via environment variable")
	}
	
	// Initialize configuration with reasonable defaults for testing
	suite.config = NewConfig().WithDefaults().
		WithNamespace("porch-integration-test").
		WithTimeout(30 * time.Second).
		WithMaxRetries(3).
		WithCache(true, 1*time.Minute).
		WithSecurity(false, false, "low"). // Disable security for testing
		WithMetrics(true, ":0").           // Use random port for metrics
		WithLogging(true, "debug")
	
	// Create client
	client, err := NewClient(suite.config)
	suite.Require().NoError(err)
	suite.client = client
	
	// Add cleanup for client
	suite.cleanup = append(suite.cleanup, func() {
		if closer, ok := suite.client.(interface{ Close() error }); ok {
			closer.Close()
		}
	})
}

// TearDownSuite cleans up after all tests
func (suite *IntegrationTestSuite) TearDownSuite() {
	for _, cleanup := range suite.cleanup {
		cleanup()
	}
}

// TestCompleteWorkflow tests the complete workflow from repository registration to function execution
func (suite *IntegrationTestSuite) TestCompleteWorkflow() {
	ctx := context.Background()
	
	// Step 1: Register a repository
	repoConfig := &RepositoryConfig{
		Name: "integration-test-repo",
		Type: "git",
		URL:  "https://github.com/GoogleContainerTools/kpt.git",
		Auth: &AuthConfig{
			Type: "none", // Public repository
		},
	}
	
	repo, err := suite.client.RegisterRepository(ctx, repoConfig)
	suite.Require().NoError(err)
	suite.NotNil(repo)
	suite.Equal("integration-test-repo", repo.Name)
	
	// Add cleanup for repository
	suite.cleanup = append(suite.cleanup, func() {
		suite.client.UnregisterRepository(context.Background(), "integration-test-repo")
	})
	
	// Step 2: Wait for repository to become healthy (with timeout)
	suite.Eventually(func() bool {
		health, err := suite.client.GetRepositoryHealth(ctx, "integration-test-repo")
		if err != nil {
			return false
		}
		return *health == RepositoryHealthHealthy
	}, 30*time.Second, 1*time.Second, "Repository should become healthy")
	
	// Step 3: Create a package
	packageSpec := &PackageSpec{
		Repository:  "integration-test-repo",
		PackageName: "test-package",
		Revision:    "v1.0.0",
		Lifecycle:   PackageLifecycleDraft,
	}
	
	pkg, err := suite.client.CreatePackage(ctx, packageSpec)
	suite.Require().NoError(err)
	suite.NotNil(pkg)
	
	// Step 4: Update package content
	resources := []KRMResource{
		{
			APIVersion: "v1",
			Kind:       "ConfigMap",
			Metadata: map[string]interface{}{
				"name":      "test-config",
				"namespace": "default",
			},
			Data: map[string]interface{}{
				"config.yaml": "test: value",
			},
		},
	}
	
	updatedPkg, err := suite.client.UpdatePackageContent(ctx, pkg.Name, resources)
	suite.Require().NoError(err)
	suite.NotNil(updatedPkg)
	
	// Step 5: Execute a function on the package
	functionRequest := &FunctionRequest{
		FunctionConfig: FunctionConfig{
			Image: "gcr.io/kpt-fn/apply-setters:v0.1.1",
			ConfigMap: map[string]string{
				"namespace": "integration-test",
			},
		},
		Resources: resources,
	}
	
	functionResponse, err := suite.client.ExecuteFunction(ctx, functionRequest)
	suite.Require().NoError(err)
	suite.NotNil(functionResponse)
	
	// Step 6: Validate function execution results
	suite.NotEmpty(functionResponse.Resources)
	suite.Empty(functionResponse.Results) // Should be no validation errors
	
	// Step 7: Promote package to proposed
	packageRef := &PackageReference{
		Repository:  "integration-test-repo",
		PackageName: "test-package",
		Revision:    "v1.0.0",
	}
	
	err = suite.client.PromoteToProposed(ctx, packageRef)
	suite.Require().NoError(err)
	
	// Step 8: Execute a pipeline
	pipelineRequest := &PipelineRequest{
		Pipeline: Pipeline{
			Mutators: []FunctionConfig{
				{
					Image: "gcr.io/kpt-fn/set-namespace:v0.1.1",
					ConfigMap: map[string]string{
						"namespace": "integration-test-namespace",
					},
				},
			},
			Validators: []FunctionConfig{
				{
					Image: "gcr.io/kpt-fn/kubeval:v0.1.1",
				},
			},
		},
		Resources: functionResponse.Resources,
	}
	
	pipelineResponse, err := suite.client.ExecutePipeline(ctx, pipelineRequest)
	suite.Require().NoError(err)
	suite.NotNil(pipelineResponse)
	suite.NoError(err)
	
	// Step 9: Get package history
	history, err := suite.client.GetPackageHistory(ctx, packageRef)
	suite.Require().NoError(err)
	suite.NotEmpty(history)
}

// TestRepositoryOperations tests repository-related operations
func (suite *IntegrationTestSuite) TestRepositoryOperations() {
	ctx := context.Background()
	
	// Test repository registration
	repoConfig := &RepositoryConfig{
		Name: "test-git-repo",
		Type: "git",
		URL:  "https://github.com/GoogleContainerTools/kpt.git",
		Sync: &SyncConfig{
			AutoSync: false, // Disable auto-sync for testing
		},
	}
	
	repo, err := suite.client.RegisterRepository(ctx, repoConfig)
	suite.Require().NoError(err)
	suite.NotNil(repo)
	
	defer suite.client.UnregisterRepository(ctx, "test-git-repo")
	
	// Test repository listing
	repos, err := suite.client.ListRepositories(ctx, &ListOptions{})
	suite.Require().NoError(err)
	suite.NotNil(repos)
	
	found := false
	for _, r := range repos.Items {
		if r.Name == "test-git-repo" {
			found = true
			break
		}
	}
	suite.True(found, "Registered repository should be found in list")
	
	// Test repository synchronization
	syncResult, err := suite.client.SynchronizeRepository(ctx, "test-git-repo")
	suite.Require().NoError(err)
	suite.NotNil(syncResult)
	
	// Test access validation
	err = suite.client.ValidateAccess(ctx, "test-git-repo")
	suite.NoError(err)
	
	// Test branch operations
	branches, err := suite.client.ListBranches(ctx, "test-git-repo")
	suite.Require().NoError(err)
	suite.NotEmpty(branches)
	suite.Contains(branches, "main") // kpt repository should have main branch
}

// TestFunctionOperations tests function-related operations
func (suite *IntegrationTestSuite) TestFunctionOperations() {
	ctx := context.Background()
	
	// Test function listing
	functions, err := suite.client.ListFunctions(ctx)
	suite.Require().NoError(err)
	suite.NotEmpty(functions)
	
	// Test function validation
	validation, err := suite.client.ValidateFunction(ctx, "gcr.io/kpt-fn/apply-setters:v0.1.1")
	suite.Require().NoError(err)
	suite.True(validation.Valid)
	
	// Test function schema retrieval
	schema, err := suite.client.GetFunctionSchema(ctx, "gcr.io/kpt-fn/apply-setters:v0.1.1")
	suite.Require().NoError(err)
	suite.NotNil(schema)
	
	// Test simple function execution
	request := &FunctionRequest{
		FunctionConfig: FunctionConfig{
			Image: "gcr.io/kpt-fn/apply-setters:v0.1.1",
		},
		Resources: []KRMResource{
			{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Metadata: map[string]interface{}{
					"name": "test-configmap",
				},
				Data: map[string]interface{}{
					"key": "value",
				},
			},
		},
	}
	
	response, err := suite.client.ExecuteFunction(ctx, request)
	suite.Require().NoError(err)
	suite.NotNil(response)
}

// TestPackageOperations tests package-related operations
func (suite *IntegrationTestSuite) TestPackageOperations() {
	ctx := context.Background()
	
	// First register a repository
	repoConfig := &RepositoryConfig{
		Name: "package-test-repo",
		Type: "git",
		URL:  "https://github.com/GoogleContainerTools/kpt.git",
	}
	
	_, err := suite.client.RegisterRepository(ctx, repoConfig)
	suite.Require().NoError(err)
	defer suite.client.UnregisterRepository(ctx, "package-test-repo")
	
	// Test package creation
	packageSpec := &PackageSpec{
		Repository:  "package-test-repo",
		PackageName: "test-package-ops",
		Revision:    "v1.0.0",
		Lifecycle:   PackageLifecycleDraft,
	}
	
	pkg, err := suite.client.CreatePackage(ctx, packageSpec)
	suite.Require().NoError(err)
	suite.NotNil(pkg)
	
	// Test package listing
	packages, err := suite.client.ListPackageRevisions(ctx, &ListOptions{})
	suite.Require().NoError(err)
	suite.NotNil(packages)
	
	// Test package retrieval
	retrievedPkg, err := suite.client.GetPackageRevision(ctx, pkg.Name)
	suite.Require().NoError(err)
	suite.Equal(pkg.Name, retrievedPkg.Name)
	
	// Test package content operations
	resources := []KRMResource{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Metadata: map[string]interface{}{
				"name": "test-deployment",
			},
			Spec: map[string]interface{}{
				"replicas": 1,
			},
		},
	}
	
	updatedPkg, err := suite.client.UpdatePackageContent(ctx, pkg.Name, resources)
	suite.Require().NoError(err)
	suite.NotNil(updatedPkg)
	
	// Test package content retrieval
	retrievedResources, err := suite.client.GetPackageContent(ctx, pkg.Name)
	suite.Require().NoError(err)
	suite.NotEmpty(retrievedResources)
}

// TestPerformanceRequirements tests that the client meets performance requirements
func (suite *IntegrationTestSuite) TestPerformanceRequirements() {
	ctx := context.Background()
	
	// Test operation latency requirements (<500ms)
	suite.Run("OperationLatency", func() {
		operations := []struct {
			name string
			fn   func() error
		}{
			{
				"ListRepositories",
				func() error {
					_, err := suite.client.ListRepositories(ctx, &ListOptions{})
					return err
				},
			},
			{
				"ListFunctions",
				func() error {
					_, err := suite.client.ListFunctions(ctx)
					return err
				},
			},
			{
				"ValidateFunction",
				func() error {
					_, err := suite.client.ValidateFunction(ctx, "gcr.io/kpt-fn/apply-setters:v0.1.1")
					return err
				},
			},
		}
		
		for _, op := range operations {
			suite.Run(op.name, func() {
				start := time.Now()
				err := op.fn()
				duration := time.Since(start)
				
				suite.NoError(err)
				suite.Less(duration, 500*time.Millisecond, "%s should complete in <500ms", op.name)
			})
		}
	})
	
	// Test concurrent operations (100+ concurrent operations)
	suite.Run("ConcurrentOperations", func() {
		concurrency := 100
		var wg sync.WaitGroup
		results := make([]error, concurrency)
		
		start := time.Now()
		
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				
				_, err := suite.client.ValidateFunction(ctx, "gcr.io/kpt-fn/apply-setters:v0.1.1")
				results[index] = err
			}(i)
		}
		
		wg.Wait()
		duration := time.Since(start)
		
		// Check all operations succeeded
		for i, err := range results {
			suite.NoError(err, "Operation %d should succeed", i)
		}
		
		// Check throughput
		opsPerSecond := float64(concurrency) / duration.Seconds()
		suite.Greater(opsPerSecond, 10.0, "Should handle >10 operations per second")
	})
	
	// Test memory usage requirements (<50MB memory footprint)
	suite.Run("MemoryFootprint", func() {
		// Force garbage collection before measuring
		runtime.GC()
		runtime.GC()
		
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)
		
		// Perform operations
		for i := 0; i < 1000; i++ {
			_, err := suite.client.ValidateFunction(ctx, "gcr.io/kpt-fn/apply-setters:v0.1.1")
			suite.NoError(err)
			
			if i%100 == 0 {
				_, err = suite.client.ListFunctions(ctx)
				suite.NoError(err)
			}
		}
		
		// Force garbage collection after operations
		runtime.GC()
		runtime.GC()
		
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)
		
		memoryUsed := m2.Alloc - m1.Alloc
		memoryUsedMB := float64(memoryUsed) / (1024 * 1024)
		
		suite.Less(memoryUsedMB, 50.0, "Memory usage should be <50MB for 1000+ operations")
	})
}

// TestErrorHandlingAndResilience tests error handling and resilience features
func (suite *IntegrationTestSuite) TestErrorHandlingAndResilience() {
	ctx := context.Background()
	
	// Test handling of non-existent resources
	suite.Run("NonExistentResources", func() {
		_, err := suite.client.GetRepository(ctx, "non-existent-repo")
		suite.Error(err)
		
		_, err = suite.client.GetPackageRevision(ctx, "non-existent-package")
		suite.Error(err)
		
		err = suite.client.ValidateAccess(ctx, "non-existent-repo")
		suite.Error(err)
	})
	
	// Test timeout handling
	suite.Run("TimeoutHandling", func() {
		timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()
		
		time.Sleep(1 * time.Millisecond) // Ensure timeout
		
		_, err := suite.client.ListRepositories(timeoutCtx, &ListOptions{})
		suite.Error(err)
		suite.Contains(err.Error(), "context deadline exceeded")
	})
	
	// Test cancellation handling
	suite.Run("CancellationHandling", func() {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately
		
		_, err := suite.client.ListRepositories(cancelCtx, &ListOptions{})
		suite.Error(err)
		suite.Contains(err.Error(), "context canceled")
	})
	
	// Test invalid input handling
	suite.Run("InvalidInputs", func() {
		// Invalid repository configuration
		invalidRepoConfig := &RepositoryConfig{
			Name: "", // Empty name should fail
			Type: "git",
			URL:  "https://github.com/test/repo.git",
		}
		
		_, err := suite.client.RegisterRepository(ctx, invalidRepoConfig)
		suite.Error(err)
		
		// Invalid function validation
		validation, err := suite.client.ValidateFunction(ctx, "invalid-function-name")
		suite.NoError(err) // Validation doesn't error but returns invalid result
		suite.False(validation.Valid)
		suite.NotEmpty(validation.Errors)
	})
}

// TestO2RANCompliance tests O-RAN compliance features
func (suite *IntegrationTestSuite) TestORANCompliance() {
	ctx := context.Background()
	
	// Test O-RAN validation
	suite.Run("ORANValidation", func() {
		validationRequest := &ORANValidationRequest{
			Resources: []KRMResource{
				{
					APIVersion: "o-ran.org/v1alpha1",
					Kind:       "O1Interface",
					Metadata: map[string]interface{}{
						"name": "test-o1-interface",
					},
					Spec: map[string]interface{}{
						"endpoint": "https://o1.example.com",
						"version":  "v1.0",
					},
				},
			},
			InterfaceTypes: []string{"O1", "A1"},
		}
		
		response, err := suite.client.ValidateORANCompliance(ctx, validationRequest)
		suite.NoError(err)
		suite.NotNil(response)
	})
	
	// Test network function configuration
	suite.Run("NetworkFunctionConfig", func() {
		transformationRequest := &TransformationRequest{
			Intent: NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-network-intent",
				},
				Spec: NetworkIntentSpec{
					NetworkFunction: NetworkFunctionSpec{
						Type: "AMF",
						Resources: map[string]interface{}{
							"cpu":    "2",
							"memory": "4Gi",
						},
					},
					Intent: "Deploy high-availability AMF with auto-scaling",
					Requirements: NetworkRequirements{
						Availability: "99.99%",
						Latency:      "< 10ms",
						Throughput:   "1000 TPS",
					},
				},
			},
			TargetCluster: "production-cluster",
		}
		
		response, err := suite.client.GetResourceTransformations(ctx, transformationRequest)
		suite.NoError(err)
		suite.NotNil(response)
	})
}

// TestConfigurationManagement tests configuration management features
func (suite *IntegrationTestSuite) TestConfigurationManagement() {
	ctx := context.Background()
	
	// Test configuration validation
	suite.Run("ConfigValidation", func() {
		config := suite.config
		err := config.Validate()
		suite.NoError(err)
	})
	
	// Test configuration updates
	suite.Run("ConfigUpdates", func() {
		// Test updating repository configuration
		repoConfig := &RepositoryConfig{
			Name: "config-test-repo",
			Type: "git",
			URL:  "https://github.com/test/repo.git",
		}
		
		suite.config.AddRepository(repoConfig)
		suite.Contains(suite.config.Repositories, "config-test-repo")
		
		// Test updating function configuration
		funcConfig := &FunctionConfig{
			Image: "custom/function:latest",
		}
		
		suite.config.AddFunction("custom-function", funcConfig)
		suite.Contains(suite.config.Functions.Registry.Functions, "custom-function")
	})
}

// Run the integration test suite
func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

// Additional standalone integration tests

func TestClientLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	
	// Test complete client lifecycle
	config := NewConfig().WithDefaults()
	
	// Create client
	client, err := NewClient(config)
	require.NoError(t, err)
	require.NotNil(t, client)
	
	// Use client for basic operations
	ctx := context.Background()
	functions, err := client.ListFunctions(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, functions)
	
	// Close client
	if closer, ok := client.(interface{ Close() error }); ok {
		err = closer.Close()
		assert.NoError(t, err)
	}
}

func TestStressTest(t *testing.T) {
	if testing.Short() || os.Getenv("SKIP_STRESS_TESTS") == "true" {
		t.Skip("Skipping stress tests")
	}
	
	config := NewConfig().WithDefaults().WithTimeout(60 * time.Second)
	client, err := NewClient(config)
	require.NoError(t, err)
	
	defer func() {
		if closer, ok := client.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()
	
	ctx := context.Background()
	
	// Stress test with high concurrency
	concurrency := 500
	duration := 30 * time.Second
	
	var wg sync.WaitGroup
	var successCount, errorCount int64
	var mu sync.Mutex
	
	start := time.Now()
	
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for time.Since(start) < duration {
				_, err := client.ValidateFunction(ctx, "gcr.io/kpt-fn/apply-setters:v0.1.1")
				
				mu.Lock()
				if err != nil {
					errorCount++
				} else {
					successCount++
				}
				mu.Unlock()
				
				time.Sleep(10 * time.Millisecond) // Small delay to prevent overwhelming
			}
		}()
	}
	
	wg.Wait()
	
	totalOps := successCount + errorCount
	errorRate := float64(errorCount) / float64(totalOps) * 100
	
	t.Logf("Stress test results: %d total operations, %d successes, %d errors (%.2f%% error rate)",
		totalOps, successCount, errorCount, errorRate)
	
	// Allow up to 5% error rate under stress
	assert.Less(t, errorRate, 5.0, "Error rate should be less than 5%% under stress")
	assert.Greater(t, successCount, int64(0), "Should have some successful operations")
}