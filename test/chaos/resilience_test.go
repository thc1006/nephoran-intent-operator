package chaos

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	porchclient "github.com/thc1006/nephoran-intent-operator/pkg/porch"
	"github.com/thc1006/nephoran-intent-operator/test/testutil"
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

func TestPorchResilienceUnderFailure(t *testing.T) {
	// Get test Kubernetes configuration
	configResult := testutil.GetTestKubernetesConfig(t)
	require.NotNil(t, configResult.Config, "Failed to get Kubernetes config")

	// Skip test if only mock configuration is available (chaos testing requires real connectivity)
	if configResult.Source == testutil.ConfigSourceMock {
		t.Skip("Chaos/resilience testing requires real cluster connectivity - skipping with mock config")
		return
	}

	// Create Porch client
	porchClient, err := testutil.CreateTestPorchClient(t, configResult)
	if err != nil {
		t.Skipf("Cannot create Porch client for resilience testing: %v", err)
		return
	}
	if porchClient == nil {
		t.Skip("Resilience test requires Porch client - skipping")
		return
	}

	t.Run("NetworkPartitionAndRecovery", func(t *testing.T) {
		var wg sync.WaitGroup
		var successCount, failureCount int32
		var mu sync.Mutex

		for i := 0; i < chaosTestPackages; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				// Simulate network partition with probabilistic failure
				err := simulateNetworkLatency(func() error {
					pkgName := fmt.Sprintf("chaos-pkg-%d", idx)
					
					// Mock porch package creation using the porch client
					packageRevision := &porchclient.PackageRevision{
						ObjectMeta: metav1.ObjectMeta{
							Name: pkgName,
							Namespace: "nephio-chaos",
						},
						Spec: porchclient.PackageRevisionSpec{
							Repository:  "chaos-test-repo",
							PackageName: pkgName,
							Revision:    "v1",
							Lifecycle:   porchclient.PackageRevisionLifecycleDraft,
						},
					}
					
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					_, err := porchClient.CreatePackageRevision(ctx, packageRevision)
					return err
				})

				mu.Lock()
				defer mu.Unlock()

				if err == nil {
					successCount++
				} else {
					failureCount++
				}
			}(i)
		}

		wg.Wait()

		t.Logf("Success Count: %d, Failure Count: %d", successCount, failureCount)
		assert.Less(t, float64(failureCount)/float64(chaosTestPackages), 0.2, 
			"Failure rate should be less than 20%")
	})

	t.Run("ServerResponseSlow", func(t *testing.T) {
		pkgName := fmt.Sprintf("slow-response-pkg-%d", time.Now().UnixNano())
		
		// Mock porch package creation for slow response test
		packageReq := &porchclient.PackageRequest{
			Repository: "slow-response-repo",
			Package:    pkgName,
			Workspace:  "main",
			Namespace:  "nephio-chaos",
		}

		// Simulate slow server response
		startTime := time.Now()
		createdPkg, err := porchClient.CreatePackageRevision(packageReq)
		duration := time.Since(startTime)

		require.NoError(t, err, "Failed to create package with slow response")
		assert.True(t, duration >= slowResponseDelay, "Operation should take at least the configured delay")

		// Note: Package deletion would be handled by the porch client in real scenario
		// For now, we just verify creation was successful
		assert.NotNil(t, createdPkg, "Created package should not be nil")
	})
}